package evolution_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/evolution/testkit"
	"github.com/dashimaki/laputa/persona"
)

// Shared contracts 2-5 wire fixtures. Decoders are strict: unknown fields,
// trailing JSON, unknown enum values and duplicate keys are rejected.

func TestScopeExactMatch(t *testing.T) {
	base := testkit.SubjectAWorkspaceX()
	same := evolution.Scope{SubjectID: testkit.SubjectAID, Kind: evolution.ScopeWorkspace, WorkspaceID: testkit.WorkspaceXID}
	if !base.Equal(same) || base != same {
		t.Fatalf("identical scope tuples must match exactly")
	}
	cases := []evolution.Scope{
		{SubjectID: "subject-a1", Kind: evolution.ScopeWorkspace, WorkspaceID: testkit.WorkspaceXID}, // prefix is not a match
		{SubjectID: testkit.SubjectAID, Kind: evolution.ScopeWorkspace, WorkspaceID: testkit.WorkspaceYID},
		{SubjectID: testkit.SubjectAID, Kind: evolution.ScopeWorkspace, WorkspaceID: testkit.WorkspaceXID + "/sub"},
		{SubjectID: testkit.SubjectAID, Kind: evolution.ScopePersonal},
		{SubjectID: testkit.SubjectBID, Kind: evolution.ScopeWorkspace, WorkspaceID: testkit.WorkspaceXID},
	}
	for i, other := range cases {
		if base.Equal(other) {
			t.Fatalf("case %d: scope comparison must be exact, never prefix-based: %+v", i, other)
		}
	}
}

func TestScopeStrictDecode(t *testing.T) {
	valid := []string{
		`{"subject_id":"subject-a","kind":"personal","workspace_id":""}`,
		`{"subject_id":"subject-a","kind":"workspace","workspace_id":"ws-x"}`,
	}
	for _, wire := range valid {
		if _, err := evolution.DecodeScope([]byte(wire)); err != nil {
			t.Fatalf("valid scope rejected: %v", err)
		}
	}
	invalid := map[string]string{
		"unknown field":        `{"subject_id":"a","kind":"personal","workspace_id":"","extra":1}`,
		"unknown kind":         `{"subject_id":"a","kind":"vault","workspace_id":"x"}`,
		"workspace missing id": `{"subject_id":"a","kind":"workspace","workspace_id":""}`,
		"personal with ws":     `{"subject_id":"a","kind":"personal","workspace_id":"ws-x"}`,
		"empty subject":        `{"subject_id":"","kind":"personal","workspace_id":""}`,
		"trailing json":        `{"subject_id":"a","kind":"personal","workspace_id":""} {"x":1}`,
		"duplicate key":        `{"subject_id":"a","kind":"personal","kind":"workspace","workspace_id":"x"}`,
	}
	for name, wire := range invalid {
		if _, err := evolution.DecodeScope([]byte(wire)); err == nil {
			t.Fatalf("%s: expected rejection, wire accepted", name)
		} else if evolution.CodeOf(err) != evolution.ErrInvalidSchema {
			t.Fatalf("%s: want invalid_schema, got %v", name, err)
		}
	}
	if _, err := evolution.DecodeScope([]byte(testkit.UnknownScopeLegacyRecordJSON)); err == nil {
		t.Fatalf("unknown-scope legacy record must fail strict decode")
	}
}

func TestRunBindingRejectsEmptyTrustedIdentity(t *testing.T) {
	binding, err := evolution.DecodeRunBinding([]byte(
		`{"subject_id":"subject-a","workspace_id":"ws-x","destination_id":"mentle.default","policy_revision":"pol-3","strategy_digest":"sha256:aa","mission_revision":4}`))
	if err != nil {
		t.Fatalf("valid binding rejected: %v", err)
	}
	if binding.MissionRevision != 4 || !binding.MissionAssigned() {
		t.Fatalf("mission_revision decode failed: %+v", binding)
	}
	empty := map[string]string{
		"subject":      `{"subject_id":"","workspace_id":"x","destination_id":"d","policy_revision":"p","strategy_digest":"s","mission_revision":0}`,
		"destination":  `{"subject_id":"a","workspace_id":"x","destination_id":"","policy_revision":"p","strategy_digest":"s","mission_revision":0}`,
		"policy":       `{"subject_id":"a","workspace_id":"x","destination_id":"d","policy_revision":"","strategy_digest":"s","mission_revision":0}`,
		"digest":       `{"subject_id":"a","workspace_id":"x","destination_id":"d","policy_revision":"p","strategy_digest":"","mission_revision":0}`,
		"unknown key":  `{"subject_id":"a","workspace_id":"x","destination_id":"d","policy_revision":"p","strategy_digest":"s","mission_revision":0,"actor":"model"}`,
		"negative rev": `{"subject_id":"a","workspace_id":"x","destination_id":"d","policy_revision":"p","strategy_digest":"s","mission_revision":-1}`,
	}
	for name, wire := range empty {
		if _, err := evolution.DecodeRunBinding([]byte(wire)); err == nil {
			t.Fatalf("%s: empty/invalid trusted identity accepted", name)
		}
	}
	unassigned, err := evolution.DecodeRunBinding([]byte(
		`{"subject_id":"subject-a","workspace_id":"","destination_id":"mentle.default","policy_revision":"pol-3","strategy_digest":"sha256:aa"}`))
	if err != nil {
		t.Fatalf("unassigned personal binding rejected: %v", err)
	}
	if unassigned.MissionAssigned() {
		t.Fatalf("mission_revision=0 must read as unassigned")
	}
}

func TestEffectStrictUnion(t *testing.T) {
	valid := map[string]string{
		"work_patch":          `{"operation_id":"op-1","payload_digest":"ab12","kind":"work_patch","work_patch":{"base_revision":3,"changes":[{"kind":"add","entry_id":"","field":"goal","body":"ship it","sources":[]}]}}`,
		"memory_mutation":     `{"operation_id":"op-2","payload_digest":"cd34","kind":"memory_mutation","memory_mutation":{"operation":"create","record_id":"r1","expected_revision":0,"expected_absent":true,"body":"fact","sources":[],"inference":"observed"}}`,
		"persona_request":     `{"operation_id":"op-3","payload_digest":"ef56","kind":"persona_request","persona_request":{"kind":"user_observations","base_revision":2,"proposed_markdown":"# Notes\n\n- prefers terse answers","reason":"observed twice","sources":[]}}`,
		"capability_proposal": `{"operation_id":"op-4","payload_digest":"gh78","kind":"capability_proposal","capability_proposal":{"name":"summarize","description":"d","proposed_artifact":"skill.md","sources":[]}}`,
		"reflection_note":     `{"operation_id":"op-5","payload_digest":"ij90","kind":"reflection_note","reflection_note":{"body":"# Review\n\nno drift","sources":[]}}`,
	}
	for name, wire := range valid {
		effect, err := evolution.DecodeEffect([]byte(wire))
		if err != nil {
			t.Fatalf("%s: permitted variant rejected: %v", name, err)
		}
		if string(effect.Kind) != name {
			t.Fatalf("%s: decoded as %s", name, effect.Kind)
		}
		if effect.Payload() == nil {
			t.Fatalf("%s: typed payload missing", name)
		}
	}
	rejected := map[string]string{
		"unknown kind":       `{"operation_id":"o","payload_digest":"d","kind":"telemetry","telemetry":{}}`,
		"mission effect":     `{"operation_id":"o","payload_digest":"d","kind":"mission","mission":{"body":"rewrite"}}`,
		"dream effect":       `{"operation_id":"o","payload_digest":"d","kind":"dream","dream":{"body":"rewrite"}}`,
		"two payloads":       `{"operation_id":"o","payload_digest":"d","kind":"reflection_note","reflection_note":{"body":"b","sources":[]},"memory_mutation":{"operation":"tombstone","record_id":"r","expected_revision":1,"expected_absent":false,"body":"","sources":[],"inference":"observed"}}`,
		"payload key drift":  `{"operation_id":"o","payload_digest":"d","kind":"reflection_note","memory_mutation":{"operation":"tombstone","record_id":"r","expected_revision":1,"expected_absent":false,"body":"","sources":[],"inference":"observed"}}`,
		"unknown envelope":   `{"operation_id":"o","payload_digest":"d","kind":"reflection_note","reflection_note":{"body":"b","sources":[]},"scope":{"subject_id":"a","kind":"personal","workspace_id":""}}`,
		"payload bad field":  `{"operation_id":"o","payload_digest":"d","kind":"reflection_note","reflection_note":{"body":"b","sources":[],"target":"x"}}`,
		"unknown enum":       `{"operation_id":"o","payload_digest":"d","kind":"memory_mutation","memory_mutation":{"operation":"replace","record_id":"r","expected_revision":1,"expected_absent":false,"body":"","sources":[],"inference":"observed"}}`,
		"duplicate key":      `{"operation_id":"o","operation_id":"p","payload_digest":"d","kind":"reflection_note","reflection_note":{"body":"b","sources":[]}}`,
		"empty operation id": `{"operation_id":"","payload_digest":"d","kind":"reflection_note","reflection_note":{"body":"b","sources":[]}}`,
	}
	for name, wire := range rejected {
		if _, err := evolution.DecodeEffect([]byte(wire)); err == nil {
			t.Fatalf("%s: wire accepted", name)
		}
	}
	// Markdown payloads ride through verbatim.
	body := "## Boundaries\n\n* item `code`\n\n```json\n{}\n```"
	wire, err := json.Marshal(map[string]any{
		"operation_id": "op-md", "payload_digest": "k", "kind": "persona_request",
		"persona_request": map[string]any{"kind": "identity", "base_revision": 1, "proposed_markdown": body, "reason": "r", "sources": []any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(wire)
	effect, err := evolution.DecodeEffect([]byte(markdown))
	if err != nil {
		t.Fatalf("markdown payload rejected: %v", err)
	}
	if !strings.Contains(effect.PersonaRequest.ProposedMarkdown, "```json") {
		t.Fatalf("markdown payload mangled")
	}
}

func TestFrozenCoreV2(t *testing.T) {
	section := func(kind string, rev int) string {
		return `{"kind":"` + kind + `","content":"body","source_revision":` + itoa(rev) + `,"source_hash":"h"}`
	}
	valid := `{"schema_version":"laputa.frozen-core/v2","session_id":"s1","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","sections":[` +
		section("mission", 7) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`
	core, err := evolution.DecodeFrozenCoreV2([]byte(valid))
	if err != nil {
		t.Fatalf("valid v2 frozen core rejected: %v", err)
	}
	if len(core.Sections) != 7 || core.Sections[0].Kind != evolution.FrozenKindMission || core.Sections[6].Kind != evolution.FrozenKindDark {
		t.Fatalf("v2 kind order lost: %+v", core.Sections)
	}
	unassigned := `{"schema_version":"laputa.frozen-core/v2","session_id":"s1","captured_at":"2026-10-02T00:00:00Z","mission_status":"unassigned","sections":[` +
		`{"kind":"mission","content":"","source_revision":0,"source_hash":""}` + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`
	if _, err := evolution.DecodeFrozenCoreV2([]byte(unassigned)); err != nil {
		t.Fatalf("unassigned mission shape rejected: %v", err)
	}
	v1Sections := strings.Join([]string{
		section("identity", 1), section("relationship", 1), section("redline", 1), section("user", 1), section("dream", 1), section("dark", 1)}, ",")
	rejected := map[string]string{
		"six-slot mislabelled v2": `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","sections":[` + v1Sections + `]}`,
		"world slot":              `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","sections":[` + section("mission", 1) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("world", 1) + `]}`,
		"actmem slot":             `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","sections":[` + section("mission", 1) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("actmem", 1) + `]}`,
		"unassigned with content": `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"unassigned","sections":[` + section("mission", 9) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`,
		"unknown status":          `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"maybe","sections":[` + section("mission", 1) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`,
		"wrong order":             `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","sections":[` + section("identity", 1) + `,` + section("mission", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`,
		"unknown field":           `{"schema_version":"laputa.frozen-core/v2","session_id":"s","captured_at":"2026-10-02T00:00:00Z","mission_status":"assigned","world":"x","sections":[` + section("mission", 1) + `,` + section("identity", 1) + `,` + section("relationship", 1) + `,` + section("redline", 1) + `,` + section("user", 1) + `,` + section("dream", 1) + `,` + section("dark", 1) + `]}`,
	}
	for name, wire := range rejected {
		if _, err := evolution.DecodeFrozenCoreV2([]byte(wire)); err == nil {
			t.Fatalf("%s: wire accepted", name)
		}
	}
}

func TestEffectReplayDigest(t *testing.T) {
	scope := testkit.SubjectAWorkspaceX()
	mutation := &evolution.MemoryMutationPayload{Operation: evolution.MutationCreate, RecordID: "r1", ExpectedAbsent: true, Body: "fact", Inference: evolution.InferenceObserved}
	effect, err := evolution.NewEffect("op-1", evolution.KindMemoryMutation, mutation, scope, "mentle.default")
	if err != nil {
		t.Fatal(err)
	}
	same, err := evolution.NewEffect("op-1", evolution.KindMemoryMutation, &evolution.MemoryMutationPayload{Operation: evolution.MutationCreate, RecordID: "r1", ExpectedAbsent: true, Body: "fact", Inference: evolution.InferenceObserved}, scope, "mentle.default")
	if err != nil {
		t.Fatal(err)
	}
	if effect.PayloadDigest != same.PayloadDigest {
		t.Fatalf("identical payload produced different digest: %s vs %s", effect.PayloadDigest, same.PayloadDigest)
	}
	if err := evolution.CheckReplay(effect, same); err != nil {
		t.Fatalf("same operation/digest must replay: %v", err)
	}
	changedBody, err := evolution.NewEffect("op-1", evolution.KindMemoryMutation, &evolution.MemoryMutationPayload{Operation: evolution.MutationCreate, RecordID: "r1", ExpectedAbsent: true, Body: "changed", Inference: evolution.InferenceObserved}, scope, "mentle.default")
	if err != nil {
		t.Fatal(err)
	}
	changedScope, err := evolution.NewEffect("op-1", evolution.KindMemoryMutation, mutation, testkit.SubjectAWorkspaceY(), "mentle.default")
	if err != nil {
		t.Fatal(err)
	}
	for name, other := range map[string]evolution.Effect{"changed payload": changedBody, "changed scope": changedScope} {
		err := evolution.CheckReplay(effect, other)
		if err == nil || evolution.CodeOf(err) != evolution.ErrIdempotencyConflict {
			t.Fatalf("%s: want idempotency_conflict, got %v", name, err)
		}
	}
}

func TestActmemGoldenGrammar(t *testing.T) {
	doc, err := evolution.ParseActmemDocument(testkit.ActmemGoldenV2)
	if err != nil {
		t.Fatalf("golden document rejected: %v", err)
	}
	if doc.Revision != 3 || len(doc.Sections[evolution.SectionPulse]) != 2 || len(doc.Sections[evolution.SectionRecap]) != 1 || len(doc.Sections[evolution.SectionWork]) != 2 {
		t.Fatalf("golden parse shape wrong: %+v", doc)
	}
	work := doc.Sections[evolution.SectionWork]
	if work[0].Meta.Field != evolution.FieldGoal || work[1].Meta.Field != evolution.FieldNext {
		t.Fatalf("work field metadata lost: %+v", work)
	}
	rendered := doc.Render()
	reparsed, err := evolution.ParseActmemDocument(rendered)
	if err != nil {
		t.Fatalf("rendered document rejected: %v\n%s", err, rendered)
	}
	if reparsed.Render() != rendered {
		t.Fatalf("round-trip not stable")
	}
	if len(reparsed.Sections[evolution.SectionPulse]) != 2 {
		t.Fatalf("round-trip lost entries")
	}
	rejected := map[string]string{
		"delimiter collision": strings.Replace(testkit.ActmemGoldenV2, "quiet shift", "x\n<!-- actmem-entry:e_ffffffffffffffffffffffffffffffff -->", 1),
		"unassociated text":   strings.Replace(testkit.ActmemGoldenV2, "## Work", "## Work\n\nloose commentary", 1),
		"misplaced section":   strings.Replace(testkit.ActmemGoldenV2, "    section: work", "    section: recap", 1),
		"missing close":       strings.Replace(testkit.ActmemGoldenV2, "<!-- /actmem-entry:"+testkit.ActmemGoldenWorkID+" -->", "", 1),
	}
	for name, text := range rejected {
		if _, err := evolution.ParseActmemDocument(text); err == nil {
			t.Fatalf("%s: invalid v2 input accepted", name)
		} else if evolution.CodeOf(err) != evolution.ErrActmemFormat {
			t.Fatalf("%s: want actmem_format_error, got %v", name, err)
		}
	}
	if _, err := evolution.ParseActmemDocument(testkit.ActmemOrphanHeaderV2); err == nil {
		t.Fatalf("orphan header metadata accepted")
	}
	if _, err := evolution.ParseActmemDocument(testkit.ActmemDuplicateHeaderKeyV2); err == nil {
		t.Fatalf("duplicate header key accepted")
	}
	if _, err := evolution.ParseActmemDocument(testkit.ActmemUnknownHeaderKeyV2); err == nil {
		t.Fatalf("unknown header key accepted")
	}
}

func TestEvaluateEligibilityOrder(t *testing.T) {
	policy := evolution.TriggerPolicy{Enabled: true, MinIntervalMS: 1000}
	base := evolution.Wake{NowUnixMS: 10_000, NewActivity: true}
	reason := func(w evolution.Wake, p evolution.TriggerPolicy, s evolution.TriggerState) string {
		return string(evolution.Evaluate(w, p, s).Reason)
	}
	if got := reason(base, evolution.TriggerPolicy{Enabled: false}, evolution.TriggerState{}); got != "disabled" {
		t.Fatalf("disabled gate: %s", got)
	}
	if got := reason(base, policy, evolution.TriggerState{ActiveRunID: "run-1"}); got != "active" {
		t.Fatalf("active gate: %s", got)
	}
	if got := reason(evolution.Wake{NowUnixMS: 10_000}, policy, evolution.TriggerState{}); got != "no_new_input" {
		t.Fatalf("no-input gate: %s", got)
	}
	if got := reason(evolution.Wake{NowUnixMS: 10_000, NewActivity: true, ForegroundBusy: true}, policy, evolution.TriggerState{}); got != "foreground_busy" {
		t.Fatalf("foreground gate: %s", got)
	}
	if got := reason(evolution.Wake{NowUnixMS: 5_000, NewActivity: true}, policy, evolution.TriggerState{LastCompletedUnixMS: 9_000}); got != "not_before_clock" {
		t.Fatalf("backward clock: %s", got)
	}
	if got := reason(base, policy, evolution.TriggerState{LastCompletedUnixMS: 9_500}); got != "interval" {
		t.Fatalf("interval gate: %s", got)
	}
	if got := reason(base, policy, evolution.TriggerState{LastCompletedUnixMS: 8_999}); got != "eligible" {
		t.Fatalf("eligible: %s", got)
	}
	// Manual ignores foreground/interval/clock but not disabled/active/no-input.
	manual := evolution.Wake{NowUnixMS: 5_000, Manual: true, NewActivity: true, ForegroundBusy: true}
	if got := reason(manual, policy, evolution.TriggerState{LastCompletedUnixMS: 9_999}); got != "eligible" {
		t.Fatalf("manual must bypass clock/interval/foreground: %s", got)
	}
	if got := reason(manual, policy, evolution.TriggerState{ActiveRunID: "r"}); got != "active" {
		t.Fatalf("manual still coalesces active runs: %s", got)
	}
	// min_interval_ms=0 still coalesces but never blocks on interval.
	if got := reason(base, evolution.TriggerPolicy{Enabled: true, MinIntervalMS: 0}, evolution.TriggerState{LastCompletedUnixMS: 9_999}); got != "eligible" {
		t.Fatalf("zero interval must not gate: %s", got)
	}
}

func TestReflectionOutputExactlyOneBranch(t *testing.T) {
	withCandidates := `{"candidates":[{"kind":"reflection_note","reflection_note":{"body":"b","sources":[]}}],"no_change_reason":""}`
	out, err := evolution.DecodeReflectionOutput([]byte(withCandidates))
	if err != nil {
		t.Fatalf("candidate output rejected: %v", err)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("candidates lost")
	}
	noChange := `{"candidates":[],"no_change_reason":"nothing new"}`
	if _, err := evolution.DecodeReflectionOutput([]byte(noChange)); err != nil {
		t.Fatalf("no_change output rejected: %v", err)
	}
	for name, wire := range map[string]string{
		"both":    `{"candidates":[{"kind":"reflection_note","reflection_note":{"body":"b","sources":[]}}],"no_change_reason":"x"}`,
		"neither": `{"candidates":[],"no_change_reason":""}`,
	} {
		if _, err := evolution.DecodeReflectionOutput([]byte(wire)); err == nil {
			t.Fatalf("%s: one-of invariant broken", name)
		}
	}
}

// TestPersonaKindNumbersUnmoved guards the storage intenum while v2 named
// kinds land beside it. Mission is a new v2 slot, not a renumbered int.
func TestPersonaKindNumbersUnmoved(t *testing.T) {
	want := map[persona.Kind]int{
		persona.KindIdentity: 0, persona.KindRelationship: 1, persona.KindRedline: 2,
		persona.KindUser: 3, persona.KindWorld: 4, persona.KindDream: 5, persona.KindDark: 6,
	}
	for kind, number := range want {
		if int(kind) != number {
			t.Fatalf("persona kind %v renumbered from %d to %d", kind, number, int(kind))
		}
	}
	mapping := map[evolution.FrozenCoreKind]persona.Kind{
		evolution.FrozenKindIdentity:     persona.KindIdentity,
		evolution.FrozenKindRelationship: persona.KindRelationship,
		evolution.FrozenKindRedline:      persona.KindRedline,
		evolution.FrozenKindUser:         persona.KindUser,
		evolution.FrozenKindDream:        persona.KindDream,
		evolution.FrozenKindDark:         persona.KindDark,
	}
	for v2, legacy := range mapping {
		if string(v2) != legacy.String() {
			t.Fatalf("v2 kind %q drifted from persona kind %q", v2, legacy.String())
		}
	}
	if _, ok := mapping[evolution.FrozenKindMission]; ok {
		t.Fatalf("mission must not alias a legacy persona kind")
	}
	if len(evolution.FrozenCoreV2Kinds) != 7 || evolution.FrozenCoreV2Kinds[0] != evolution.FrozenKindMission {
		t.Fatalf("v2 kind roster changed")
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
