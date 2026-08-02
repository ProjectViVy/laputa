package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Contract struct {
	Requires     []string
	Produces     []string
	Capabilities []string
	Idempotent   bool
}

type Result struct {
	Next     string
	Stop     bool
	Warnings []string
}

type State map[string]any

type Step interface {
	ID() string
	Contract() Contract
	Run(context.Context, State) (Result, error)
}

type Interceptor interface {
	BeforeStep(context.Context, Step, State) error
	AfterStep(context.Context, Step, State, Result) error
	OnError(context.Context, Step, State, error) error
}

type StepConfig struct {
	ID        string   `yaml:"id" json:"id"`
	Enabled   *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	TimeoutMS int      `yaml:"timeout_ms,omitempty" json:"timeout_ms,omitempty"`
	OnError   string   `yaml:"on_error,omitempty" json:"on_error,omitempty"`
	Next      []string `yaml:"next,omitempty" json:"next,omitempty"`
}

func (s StepConfig) IsEnabled() bool { return s.Enabled == nil || *s.Enabled }

type Definition struct {
	Name             string       `yaml:"name" json:"name"`
	Version          string       `yaml:"version" json:"version"`
	Enabled          *bool        `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Capabilities     []string     `yaml:"capabilities" json:"capabilities"`
	MaxSteps         int          `yaml:"max_steps" json:"max_steps"`
	MaxVisitsPerStep int          `yaml:"max_visits_per_step" json:"max_visits_per_step"`
	Steps            []StepConfig `yaml:"steps" json:"steps"`
}

func (d Definition) IsEnabled() bool { return d.Enabled == nil || *d.Enabled }

type StepTrace struct {
	Step       string `json:"step"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type RunTrace struct {
	ID         string      `json:"trace_id"`
	Pipeline   string      `json:"pipeline"`
	Revision   string      `json:"revision"`
	StartedAt  time.Time   `json:"started_at"`
	DurationMS int64       `json:"duration_ms"`
	Status     string      `json:"status"`
	Warnings   []string    `json:"warnings,omitempty"`
	Steps      []StepTrace `json:"steps"`
}

type Manager struct {
	mu           sync.RWMutex
	definitions  map[string]Definition
	steps        map[string]Step
	revision     string
	history      []RunTrace
	historySize  int
	interceptors []Interceptor
}

func (m *Manager) Use(interceptor Interceptor) {
	if interceptor == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interceptors = append(m.interceptors, interceptor)
}

func NewManager(definitions []Definition, revision string) (*Manager, error) {
	m := &Manager{definitions: map[string]Definition{}, steps: map[string]Step{}, revision: revision, historySize: 100}
	for _, definition := range definitions {
		if definition.MaxSteps <= 0 {
			definition.MaxSteps = len(definition.Steps) * 3
		}
		if definition.MaxVisitsPerStep <= 0 {
			definition.MaxVisitsPerStep = 2
		}
		if err := validateDefinition(definition); err != nil {
			return nil, err
		}
		m.definitions[definition.Name] = definition
	}
	return m, nil
}

func (m *Manager) Register(step Step) error {
	if step == nil || step.ID() == "" {
		return errors.New("pipeline step id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.steps[step.ID()]; exists {
		return fmt.Errorf("pipeline step %q already registered", step.ID())
	}
	m.steps[step.ID()] = step
	return nil
}

func (m *Manager) ValidateRegistrations() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, definition := range m.definitions {
		for _, configured := range definition.Steps {
			if configured.IsEnabled() && m.steps[configured.ID] == nil {
				return fmt.Errorf("pipeline %s: step %q is not registered", definition.Name, configured.ID)
			}
		}
	}
	return nil
}

func (m *Manager) Run(ctx context.Context, name, traceID string, state State) (RunTrace, error) {
	m.mu.RLock()
	definition, ok := m.definitions[name]
	registered := make(map[string]Step, len(m.steps))
	for k, v := range m.steps {
		registered[k] = v
	}
	revision := m.revision
	interceptors := append([]Interceptor(nil), m.interceptors...)
	m.mu.RUnlock()
	if !ok {
		return RunTrace{}, fmt.Errorf("pipeline %q not found", name)
	}
	if !definition.IsEnabled() {
		return RunTrace{}, fmt.Errorf("pipeline %q is disabled", name)
	}
	if traceID == "" {
		traceID = fmt.Sprintf("run_%d", time.Now().UnixNano())
	}
	trace := RunTrace{ID: traceID, Pipeline: name, Revision: revision, StartedAt: time.Now().UTC(), Status: "running", Steps: []StepTrace{}}
	started := time.Now()
	allowedCaps := stringSet(definition.Capabilities)
	positions := map[string]int{}
	for i, cfg := range definition.Steps {
		if cfg.IsEnabled() {
			positions[cfg.ID] = i
		}
	}
	visits := map[string]int{}
	index := firstEnabled(definition.Steps, 0)
	var runErr error
	for index >= 0 && index < len(definition.Steps) {
		if len(trace.Steps) >= definition.MaxSteps {
			runErr = errors.New("pipeline step budget exceeded")
			break
		}
		cfg := definition.Steps[index]
		if !cfg.IsEnabled() {
			index = firstEnabled(definition.Steps, index+1)
			continue
		}
		visits[cfg.ID]++
		if visits[cfg.ID] > definition.MaxVisitsPerStep {
			runErr = fmt.Errorf("step %s visit budget exceeded", cfg.ID)
			break
		}
		step := registered[cfg.ID]
		if step == nil {
			runErr = fmt.Errorf("step %s is not registered", cfg.ID)
			break
		}
		contract := step.Contract()
		if err := validateState(contract.Requires, state); err != nil {
			runErr = fmt.Errorf("step %s: %w", cfg.ID, err)
			break
		}
		if err := validateCapabilities(contract.Capabilities, allowedCaps); err != nil {
			runErr = fmt.Errorf("step %s: %w", cfg.ID, err)
			break
		}
		stepStarted := time.Now()
		var result Result
		var err error
		for _, interceptor := range interceptors {
			if err = interceptor.BeforeStep(ctx, step, state); err != nil {
				break
			}
		}
		if err == nil {
			result, err = runStep(ctx, step, cfg, state)
		}
		if err == nil {
			for _, interceptor := range interceptors {
				if err = interceptor.AfterStep(ctx, step, state, result); err != nil {
					break
				}
			}
		}
		if err != nil {
			for _, interceptor := range interceptors {
				if replacement := interceptor.OnError(ctx, step, state, err); replacement != nil {
					err = replacement
				}
			}
		}
		status := "ok"
		code := ""
		if err != nil {
			status = "error"
			code = "step_failed"
		}
		trace.Steps = append(trace.Steps, StepTrace{Step: cfg.ID, Status: status, DurationMS: time.Since(stepStarted).Milliseconds(), ErrorCode: code})
		trace.Warnings = append(trace.Warnings, result.Warnings...)
		if err != nil {
			if cfg.OnError == "skip" {
				trace.Warnings = append(trace.Warnings, cfg.ID+": "+err.Error())
			} else {
				runErr = err
				break
			}
		}
		if err == nil {
			if err := validateState(contract.Produces, state); err != nil {
				runErr = fmt.Errorf("step %s: %w", cfg.ID, err)
				break
			}
		}
		if result.Stop {
			break
		}
		if result.Next != "" {
			nextIndex, exists := positions[result.Next]
			if !exists || !contains(cfg.Next, result.Next) {
				runErr = fmt.Errorf("step %s requested forbidden transition to %s", cfg.ID, result.Next)
				break
			}
			index = nextIndex
			continue
		}
		index = firstEnabled(definition.Steps, index+1)
	}
	trace.DurationMS = time.Since(started).Milliseconds()
	if runErr != nil {
		trace.Status = "error"
	} else if len(trace.Warnings) > 0 {
		trace.Status = "degraded"
	} else {
		trace.Status = "ok"
	}
	m.record(trace)
	return trace, runErr
}

func runStep(parent context.Context, step Step, cfg StepConfig, state State) (Result, error) {
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	run := func() (Result, error) {
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()
		return step.Run(ctx, state)
	}
	result, err := run()
	if err != nil && step.Contract().Idempotent && cfg.OnError == "retry" {
		result, err = run()
	}
	return result, err
}

func (m *Manager) record(trace RunTrace) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = append(m.history, trace)
	if len(m.history) > m.historySize {
		m.history = m.history[len(m.history)-m.historySize:]
	}
}

func (m *Manager) Definitions() []Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Definition, 0, len(m.definitions))
	for _, d := range m.definitions {
		out = append(out, d)
	}
	return out
}
func (m *Manager) Revision() string { m.mu.RLock(); defer m.mu.RUnlock(); return m.revision }
func (m *Manager) Runs(name string, limit int) []RunTrace {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	out := []RunTrace{}
	for i := len(m.history) - 1; i >= 0 && len(out) < limit; i-- {
		if name == "" || m.history[i].Pipeline == name {
			out = append(out, m.history[i])
		}
	}
	return out
}
func (m *Manager) RunByID(name, id string) (RunTrace, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].Pipeline == name && m.history[i].ID == id {
			return m.history[i], true
		}
	}
	return RunTrace{}, false
}

func validateDefinition(d Definition) error {
	if d.Name == "" {
		return errors.New("pipeline name is required")
	}
	if len(d.Steps) == 0 {
		return fmt.Errorf("pipeline %s has no steps", d.Name)
	}
	if d.MaxSteps <= 0 {
		d.MaxSteps = len(d.Steps) * 3
	}
	if d.MaxVisitsPerStep <= 0 {
		d.MaxVisitsPerStep = 2
	}
	seen := map[string]bool{}
	for _, s := range d.Steps {
		if s.ID == "" || seen[s.ID] {
			return fmt.Errorf("pipeline %s has invalid or duplicate step %q", d.Name, s.ID)
		}
		seen[s.ID] = true
	}
	for _, s := range d.Steps {
		for _, next := range s.Next {
			if !seen[next] {
				return fmt.Errorf("pipeline %s step %s references unknown step %s", d.Name, s.ID, next)
			}
		}
	}
	return nil
}
func validateState(keys []string, state State) error {
	for _, key := range keys {
		if _, ok := state[key]; !ok {
			return fmt.Errorf("required state %q is missing", key)
		}
	}
	return nil
}
func validateCapabilities(required []string, allowed map[string]struct{}) error {
	for _, capability := range required {
		if _, ok := allowed[capability]; !ok {
			return fmt.Errorf("capability %q is denied", capability)
		}
	}
	return nil
}
func firstEnabled(steps []StepConfig, start int) int {
	for i := start; i < len(steps); i++ {
		if steps[i].IsEnabled() {
			return i
		}
	}
	return -1
}
func stringSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
