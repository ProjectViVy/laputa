package rag

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/dashimaki/garden/internal/pipeline"
	"github.com/dashimaki/mentle/facade"
)

type Retriever interface {
	Retrieve(context.Context, facade.RetrievalQuery) ([]facade.RetrievalHit, error)
	QueryEntity(context.Context, string, string, string) ([]facade.GraphFact, error)
	Timeline(context.Context, string) ([]facade.TimelineEvent, error)
}

type Service struct {
	Pipelines *pipeline.Manager
	Policy    PolicyResolver
	Retriever Retriever
	Planner   Planner
}

func NewService(manager *pipeline.Manager, policy PolicyResolver, retriever Retriever, planner Planner) (*Service, error) {
	if manager == nil {
		return nil, errors.New("pipeline manager is required")
	}
	if planner == nil {
		planner = RulePlanner{}
	}
	s := &Service{Pipelines: manager, Policy: policy, Retriever: retriever, Planner: planner}
	steps := []pipeline.Step{
		loadGovernanceStep{s}, resolvePolicyStep{}, buildPlanStep{s}, retrieveStep{s}, expandGraphStep{s},
		judgeStep{s}, rankStep{}, filterStep{}, assembleStep{s},
	}
	for _, step := range steps {
		if err := manager.Register(step); err != nil {
			return nil, err
		}
	}
	if err := manager.ValidateRegistrations(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Resolve(ctx context.Context, request ResolveRequest) (ContextPackage, error) {
	request.Intent = strings.TrimSpace(request.Intent)
	if request.Intent == "" {
		return ContextPackage{}, errors.New("intent is required")
	}
	request.Mode = strings.ToLower(strings.TrimSpace(request.Mode))
	if request.Mode == "" {
		request.Mode = "auto"
	}
	if request.Mode != "auto" && request.Mode != "basic" && request.Mode != "advanced" {
		return ContextPackage{}, errors.New("mode must be basic, advanced, or auto")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if request.Mode == "basic" {
		return s.resolveBasic(ctx, request)
	}
	response, err := s.resolveAdvanced(ctx, request)
	if err == nil {
		return response, nil
	}
	fallback, fallbackErr := s.resolveBasic(ctx, request)
	if fallbackErr != nil {
		return ContextPackage{}, err
	}
	fallback.Degraded = true
	fallback.Warnings = uniqueStrings(append(fallback.Warnings, "advanced resolution failed; basic fallback used"))
	return fallback, nil
}

func (s *Service) resolveAdvanced(ctx context.Context, request ResolveRequest) (ContextPackage, error) {
	traceID := fmt.Sprintf("run_%d", time.Now().UnixNano())
	state := pipeline.State{"intent": request.Intent, "session_id": request.SessionID, "trace_id": traceID, "round": 0}
	trace, err := s.Pipelines.Run(ctx, "agentic_recall_v1", traceID, state)
	if err != nil {
		return ContextPackage{}, err
	}
	response, ok := state["response"].(ContextPackage)
	if !ok {
		return ContextPackage{}, errors.New("pipeline produced no context package")
	}
	response.Warnings = uniqueStrings(append(response.Warnings, trace.Warnings...))
	response.Degraded = response.Degraded || trace.Status == "degraded"
	response.Mode = "advanced"
	return response, nil
}

func (s *Service) resolveBasic(ctx context.Context, request ResolveRequest) (ContextPackage, error) {
	traceID := fmt.Sprintf("run_%d", time.Now().UnixNano())
	sections, err := s.Policy.Load(ctx)
	if err != nil {
		return ContextPackage{}, err
	}
	policy := ResolvePolicy(sections)
	candidates := []Candidate{}
	for _, name := range []string{"01-identity", "03-commitment", "04-preferences", "05-memory_md", "06-history_md"} {
		if data := sections[name]; meaningful(data) {
			raw, _ := json.Marshal(data)
			candidate := Candidate{Source: "governance", Locator: name, Content: string(raw), Score: .75}
			if candidateAllowed(candidate, policy) {
				candidates = append(candidates, candidate)
			}
		}
	}
	warnings := []string{}
	if s.Retriever != nil && policy.AllowedCapabilities["hybrid"] {
		hits, e := s.Retriever.Retrieve(ctx, facade.RetrievalQuery{Text: request.Intent, Limit: 12})
		if e != nil {
			warnings = append(warnings, "memory retrieval failed")
		} else {
			for _, hit := range hits {
				candidate := Candidate{Source: "memory", Locator: strings.Trim(strings.Join([]string{hit.Wing, hit.Room, hit.ID}, "/"), "/"), Content: hit.Content, Score: math.Min(1, hit.Score*60), Channels: hit.Channels}
				if candidateAllowed(candidate, policy) {
					candidates = append(candidates, candidate)
				}
			}
		}
	} else if s.Retriever == nil {
		warnings = append(warnings, "mentle unavailable; governance-only context")
	}
	state := pipeline.State{"candidates": candidates, "policy": policy}
	_, _ = rankStep{}.Run(ctx, state)
	candidates = state["candidates"].([]Candidate)
	if len(candidates) > 12 {
		candidates = candidates[:12]
	}
	evidence := make([]Evidence, 0, len(candidates))
	for i, c := range candidates {
		evidence = append(evidence, Evidence{ID: fmt.Sprintf("ev_%03d", i+1), Source: c.Source, Locator: c.Locator, Excerpt: truncate(c.Content, 800), Score: roundScore(c.Score)})
	}
	return ContextPackage{TraceID: traceID, Mode: "basic", Context: extractiveContext(request.Intent, evidence, 2000), Evidence: evidence, Confidence: confidenceFor(evidence, len(warnings) > 0), Degraded: len(warnings) > 0, Warnings: warnings}, nil
}

type loadGovernanceStep struct{ s *Service }

func (loadGovernanceStep) ID() string { return "load_governance" }
func (loadGovernanceStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"intent"}, Produces: []string{"governance", "candidates"}, Capabilities: []string{"governance"}, Idempotent: true}
}
func (st loadGovernanceStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	sections, err := st.s.Policy.Load(ctx)
	if err != nil {
		return pipeline.Result{}, err
	}
	state["governance"] = sections
	candidates := []Candidate{}
	for _, name := range []string{"01-identity", "03-commitment", "04-preferences", "05-memory_md", "06-history_md"} {
		if data := sections[name]; meaningful(data) {
			raw, _ := json.Marshal(data)
			candidates = append(candidates, Candidate{Source: "governance", Locator: name, Content: string(raw), Score: .75, Channels: []string{"section"}})
		}
	}
	state["candidates"] = candidates
	return pipeline.Result{}, nil
}

type resolvePolicyStep struct{}

func (resolvePolicyStep) ID() string { return "resolve_policy" }
func (resolvePolicyStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"governance"}, Produces: []string{"policy"}, Capabilities: []string{"governance"}}
}
func (resolvePolicyStep) Run(_ context.Context, state pipeline.State) (pipeline.Result, error) {
	sections, _ := state["governance"].(map[string]map[string]any)
	state["policy"] = ResolvePolicy(sections)
	return pipeline.Result{}, nil
}

type buildPlanStep struct{ s *Service }

func (buildPlanStep) ID() string { return "build_plan" }
func (buildPlanStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"intent", "governance", "policy"}, Produces: []string{"plan"}, Capabilities: []string{"llm"}, Idempotent: true}
}
func (st buildPlanStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	intent := state["intent"].(string)
	sections := state["governance"].(map[string]map[string]any)
	policy := state["policy"].(Policy)
	planner := st.s.Planner
	warning := []string{}
	if !externalPlannerAvailable(planner) {
		warning = append(warning, "external planner unavailable; deterministic plan used")
	}
	if !policy.AllowedCapabilities["llm"] {
		planner = RulePlanner{}
		warning = append(warning, "llm planning denied by governance")
	}
	visible := sections
	if policy.ExternalContext != "full" || policy.DeniedSources["governance"] {
		visible = nil
	}
	plan, err := planner.Plan(ctx, PlannerInput{Intent: intent, Governance: visible})
	if err != nil {
		plan, _ = RulePlanner{}.Plan(ctx, PlannerInput{Intent: intent})
		warning = append(warning, "planner unavailable; deterministic plan used")
	}
	state["plan"] = sanitizePlan(plan, intent)
	return pipeline.Result{Warnings: warning}, nil
}

type retrieveStep struct{ s *Service }

func (retrieveStep) ID() string { return "retrieve_candidates" }
func (retrieveStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"plan", "candidates", "policy"}, Produces: []string{"candidates"}, Capabilities: []string{"hybrid"}, Idempotent: true}
}
func (st retrieveStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	if st.s.Retriever == nil {
		return pipeline.Result{Warnings: []string{"mentle unavailable; governance-only context"}}, nil
	}
	policy := state["policy"].(Policy)
	if !policy.AllowedCapabilities["hybrid"] {
		return pipeline.Result{Warnings: []string{"hybrid retrieval denied by governance"}}, nil
	}
	plan := state["plan"].(RetrievalPlan)
	candidates := state["candidates"].([]Candidate)
	warnings := []string{}
	for _, query := range plan.Queries {
		hits, err := st.s.Retriever.Retrieve(ctx, facade.RetrievalQuery{Text: query, Limit: 20})
		if err != nil {
			warnings = append(warnings, "memory retrieval failed")
			continue
		}
		for _, hit := range hits {
			score := math.Min(1, hit.Score*60)
			candidate := Candidate{Source: "memory", Locator: strings.Trim(strings.Join([]string{hit.Wing, hit.Room, hit.ID}, "/"), "/"), Content: hit.Content, Score: score, Channels: hit.Channels}
			if candidateAllowed(candidate, policy) {
				candidates = append(candidates, candidate)
			}
		}
	}
	state["candidates"] = candidates
	return pipeline.Result{Warnings: warnings}, nil
}

type expandGraphStep struct{ s *Service }

func (expandGraphStep) ID() string { return "expand_graph" }
func (expandGraphStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"plan", "candidates", "policy"}, Produces: []string{"candidates"}, Capabilities: []string{"kg", "timeline"}, Idempotent: true}
}
func (st expandGraphStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	if st.s.Retriever == nil {
		return pipeline.Result{}, nil
	}
	policy := state["policy"].(Policy)
	plan := state["plan"].(RetrievalPlan)
	candidates := state["candidates"].([]Candidate)
	warnings := []string{}
	for _, entity := range plan.Entities {
		if policy.AllowedCapabilities["kg"] {
			facts, err := st.s.Retriever.QueryEntity(ctx, entity, "", "outgoing")
			if err != nil {
				warnings = append(warnings, "knowledge graph expansion failed")
			} else {
				for _, fact := range facts {
					raw, _ := json.Marshal(fact)
					candidate := Candidate{Source: "kg", Locator: entity, Content: string(raw), Score: fact.Confidence, Channels: []string{"kg"}}
					if candidateAllowed(candidate, policy) {
						candidates = append(candidates, candidate)
					}
				}
			}
		}
		if plan.Temporal && policy.AllowedCapabilities["timeline"] {
			events, err := st.s.Retriever.Timeline(ctx, entity)
			if err != nil {
				warnings = append(warnings, "timeline expansion failed")
			} else {
				for _, event := range events {
					raw, _ := json.Marshal(event)
					candidate := Candidate{Source: "timeline", Locator: entity, Content: string(raw), Score: .7, Channels: []string{"timeline"}}
					if candidateAllowed(candidate, policy) {
						candidates = append(candidates, candidate)
					}
				}
			}
		}
	}
	state["candidates"] = candidates
	return pipeline.Result{Warnings: warnings}, nil
}

type judgeStep struct{ s *Service }

func (judgeStep) ID() string { return "judge_sufficiency" }
func (judgeStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"intent", "plan", "candidates", "policy", "round"}, Produces: []string{"plan", "round"}, Capabilities: []string{"llm"}, Idempotent: true}
}
func (st judgeStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	candidates := state["candidates"].([]Candidate)
	round := state["round"].(int)
	policy := state["policy"].(Policy)
	if len(candidates) >= 5 || round >= 1 || !policy.AllowedCapabilities["llm"] {
		return pipeline.Result{}, nil
	}
	sections := state["governance"].(map[string]map[string]any)
	if policy.ExternalContext != "full" || policy.DeniedSources["governance"] {
		sections = nil
	}
	plan := state["plan"].(RetrievalPlan)
	refined, err := st.s.Planner.Refine(ctx, PlannerInput{Intent: state["intent"].(string), Governance: sections, Candidates: candidates, Prior: plan})
	if err != nil || refined.Stop || len(refined.Queries) == 0 {
		return pipeline.Result{Warnings: []string{"retrieval refinement stopped or unavailable"}}, nil
	}
	plan.Queries = uniqueStrings(append(plan.Queries, refined.Queries...))
	plan.Entities = uniqueStrings(append(plan.Entities, refined.Entities...))
	plan.Temporal = plan.Temporal || refined.Temporal
	state["plan"] = sanitizePlan(plan, state["intent"].(string))
	state["round"] = round + 1
	return pipeline.Result{Next: "retrieve_candidates"}, nil
}

type rankStep struct{}

func (rankStep) ID() string { return "deduplicate_and_rerank" }
func (rankStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"candidates", "policy"}, Produces: []string{"candidates"}}
}
func (rankStep) Run(_ context.Context, state pipeline.State) (pipeline.Result, error) {
	policy := state["policy"].(Policy)
	input := state["candidates"].([]Candidate)
	seen := map[[32]byte]bool{}
	out := []Candidate{}
	for _, candidate := range input {
		key := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(candidate.Content))))
		if candidate.Content == "" || seen[key] {
			continue
		}
		seen[key] = true
		parts := strings.Split(candidate.Locator, "/")
		if len(parts) > 0 && policy.PreferredWings[parts[0]] {
			candidate.Score = math.Min(1, candidate.Score+.1)
		}
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > 40 {
		out = out[:40]
	}
	state["candidates"] = out
	return pipeline.Result{}, nil
}

type filterStep struct{}

func (filterStep) ID() string { return "governance_filter" }
func (filterStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"candidates", "policy"}, Produces: []string{"candidates"}, Capabilities: []string{"governance"}}
}
func (filterStep) Run(_ context.Context, state pipeline.State) (pipeline.Result, error) {
	policy := state["policy"].(Policy)
	out := []Candidate{}
	for _, candidate := range state["candidates"].([]Candidate) {
		if candidateAllowed(candidate, policy) {
			out = append(out, candidate)
		}
	}
	state["candidates"] = out
	return pipeline.Result{}, nil
}

type assembleStep struct{ s *Service }

func (assembleStep) ID() string { return "assemble_context" }
func (assembleStep) Contract() pipeline.Contract {
	return pipeline.Contract{Requires: []string{"intent", "trace_id", "candidates"}, Produces: []string{"response"}, Capabilities: []string{"llm"}, Idempotent: true}
}
func (st assembleStep) Run(ctx context.Context, state pipeline.State) (pipeline.Result, error) {
	candidates := state["candidates"].([]Candidate)
	if len(candidates) > 12 {
		candidates = candidates[:12]
	}
	evidence := make([]Evidence, 0, len(candidates))
	for i, candidate := range candidates {
		evidence = append(evidence, Evidence{ID: fmt.Sprintf("ev_%03d", i+1), Source: candidate.Source, Locator: candidate.Locator, Excerpt: truncate(candidate.Content, 800), Score: roundScore(candidate.Score)})
	}
	planner := st.s.Planner
	if len(evidence) == 0 {
		planner = RulePlanner{}
	}
	if policy, ok := state["policy"].(Policy); ok && (!policy.AllowedCapabilities["llm"] || policy.ExternalContext != "full") {
		planner = RulePlanner{}
	}
	contextText, err := planner.Summarize(ctx, state["intent"].(string), evidence)
	warnings := []string{}
	if err != nil || !validCitations(contextText, evidence) {
		contextText = extractiveContext(state["intent"].(string), evidence, 4000)
		warnings = append(warnings, "extractive context fallback used")
	}
	confidence := confidenceFor(evidence, len(warnings) > 0)
	state["response"] = ContextPackage{TraceID: state["trace_id"].(string), Context: contextText, Evidence: evidence, Confidence: confidence, Degraded: len(warnings) > 0, Warnings: warnings}
	return pipeline.Result{Warnings: warnings}, nil
}

func meaningful(data map[string]any) bool {
	for key, value := range data {
		if key == "_meta" || key == "status" {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return true
			}
		case []any:
			if len(v) > 0 {
				return true
			}
		case map[string]any:
			if meaningful(v) {
				return true
			}
		default:
			if value != nil {
				return true
			}
		}
	}
	return false
}
func truncate(value string, max int) string {
	r := []rune(value)
	if len(r) <= max {
		return value
	}
	return string(r[:max]) + "…"
}
func roundScore(value float64) float64 {
	return math.Round(math.Max(0, math.Min(1, value))*1000) / 1000
}
func confidenceFor(evidence []Evidence, degraded bool) float64 {
	if len(evidence) == 0 {
		return 0
	}
	score := 0.35 + math.Min(.45, float64(len(evidence))*.05)
	channels := map[string]bool{}
	for _, item := range evidence {
		channels[item.Source] = true
	}
	score += math.Min(.15, float64(len(channels)-1)*.05)
	if degraded {
		score -= .2
	}
	return roundScore(score)
}
func extractiveContext(intent string, evidence []Evidence, maxTokens int) string {
	var b strings.Builder
	b.WriteString("Intent: " + intent + "\n\nRelevant evidence:\n")
	limit := maxTokens * 4
	for _, item := range evidence {
		line := fmt.Sprintf("- [%s] (%s: %s) %s\n", item.ID, item.Source, item.Locator, item.Excerpt)
		if b.Len()+len(line) > limit {
			break
		}
		b.WriteString(line)
	}
	return b.String()
}
func validCitations(value string, evidence []Evidence) bool {
	if len(evidence) == 0 {
		return true
	}
	valid := map[string]bool{}
	for _, item := range evidence {
		valid[item.ID] = true
	}
	found := false
	for start := 0; start < len(value); {
		i := strings.Index(value[start:], "[ev_")
		if i < 0 {
			break
		}
		i += start
		j := strings.Index(value[i:], "]")
		if j < 0 {
			return false
		}
		id := value[i+1 : i+j]
		if !valid[id] {
			return false
		}
		found = true
		start = i + j + 1
	}
	return found
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func candidateAllowed(candidate Candidate, policy Policy) bool {
	if policy.DeniedSources[candidate.Source] {
		return false
	}
	parts := strings.Split(candidate.Locator, "/")
	if len(parts) > 0 && policy.DeniedWings[parts[0]] {
		return false
	}
	if len(parts) > 1 && policy.DeniedRooms[parts[1]] {
		return false
	}
	return true
}
func externalPlannerAvailable(planner Planner) bool {
	switch value := planner.(type) {
	case RulePlanner, *RulePlanner:
		return false
	case FallbackPlanner:
		return value.Primary != nil
	case *FallbackPlanner:
		return value != nil && value.Primary != nil
	default:
		return planner != nil
	}
}
