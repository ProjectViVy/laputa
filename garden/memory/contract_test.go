package memory_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ProjectViVy/laputa/garden/agentapi"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

var (
	scopeX = evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-x"}
	scopeY = evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-y"}
)

func TestAuthorizedMutationValidate(t *testing.T) {
	base := memory.AuthorizedMutation{
		Scope:         scopeX,
		DestinationID: "mentle.default",
		OperationID:   "op-1",
		PayloadDigest: "ab12",
		RecordID:      "rec-1",
		Body:          "fact",
		Inference:     evolution.InferenceObserved,
	}
	create := base
	create.Operation = evolution.MutationCreate
	create.ExpectedAbsent = true
	if err := create.Validate(); err != nil {
		t.Fatalf("valid create rejected: %v", err)
	}
	update := base
	update.Operation = evolution.MutationUpdate
	update.ExpectedRevision = 7
	if err := update.Validate(); err != nil {
		t.Fatalf("valid update rejected: %v", err)
	}
	bad := map[string]memory.AuthorizedMutation{
		"create without absent":   {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationCreate, RecordID: "r", Body: "b", Inference: evolution.InferenceObserved},
		"create with revision":    {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationCreate, ExpectedAbsent: true, ExpectedRevision: 2, RecordID: "r", Body: "b", Inference: evolution.InferenceObserved},
		"update without revision": {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationUpdate, RecordID: "r", Body: "b", Inference: evolution.InferenceObserved},
		"update flagged absent":   {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationTombstone, ExpectedAbsent: true, ExpectedRevision: 2, RecordID: "r", Inference: evolution.InferenceObserved},
		"missing scope":           {DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationUpdate, ExpectedRevision: 1, RecordID: "r", Inference: evolution.InferenceObserved},
		"unknown operation":       {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: "replace", RecordID: "r", Inference: evolution.InferenceObserved},
		"unknown inference":       {Scope: scopeX, DestinationID: "d", OperationID: "o", PayloadDigest: "p", Operation: evolution.MutationCreate, ExpectedAbsent: true, RecordID: "r", Inference: "guessed"},
	}
	for name, m := range bad {
		if err := m.Validate(); err == nil {
			t.Fatalf("%s: invalid mutation accepted", name)
		}
	}
}

func TestBoundWriterExactScope(t *testing.T) {
	m := memory.AuthorizedMutation{
		Scope: scopeX, DestinationID: "mentle.default", OperationID: "o", PayloadDigest: "p",
		Operation: evolution.MutationCreate, ExpectedAbsent: true, RecordID: "r", Body: "b",
		Inference: evolution.InferenceObserved,
	}
	if err := m.MatchesWriter(scopeX, "mentle.default"); err != nil {
		t.Fatalf("bound writer rejected: %v", err)
	}
	if err := m.MatchesWriter(scopeY, "mentle.default"); err == nil || evolution.CodeOf(err) != evolution.ErrInvalidScope {
		t.Fatalf("cross-workspace scope must fail exact match: %v", err)
	}
	personal := evolution.Scope{SubjectID: "subject-a", Kind: evolution.ScopePersonal}
	if err := m.MatchesWriter(personal, "mentle.default"); err == nil {
		t.Fatalf("personal scope must not match workspace mutation")
	}
	if err := m.MatchesWriter(scopeX, "other-destination"); err == nil {
		t.Fatalf("other destination must fail")
	}
}

func TestBackendDTOsWireShape(t *testing.T) {
	search := memory.AuthorizedSearch{Scopes: []evolution.Scope{scopeX}, Query: "q", Cursor: "c1", Limit: 10, BudgetChars: 4000}
	encoded, err := json.Marshal(search)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"scopes"`, `"query"`, `"cursor"`, `"limit"`, `"budget_chars"`, `"subject_id"`, `"workspace_id"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("missing %s in %s", key, encoded)
		}
	}
	receipt := memory.MutationReceipt{
		EffectReceipt:   evolution.EffectReceipt{OperationID: "op", PayloadDigest: "d", Status: evolution.StatusApplied, TargetRef: "rec-1", Revision: 3},
		CanonicalStatus: memory.CanonicalCompleted,
		IndexStatus:     memory.IndexPending,
	}
	encoded, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"operation_id"`, `"payload_digest"`, `"status"`, `"canonical_status"`, `"index_status"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("missing %s in %s", key, encoded)
		}
	}
	page := memory.CardPage{Items: []agentapi.MemoryCard{{ID: "c", Scope: "ws-x", Revision: 1, Status: "active"}}, NextCursor: "n"}
	encoded, _ = json.Marshal(page)
	if !strings.Contains(string(encoded), `"next_cursor"`) {
		t.Fatalf("card page wire shape wrong: %s", encoded)
	}
	evidence := memory.EvidencePage{Items: []agentapi.EvidenceFragment{{CardID: "c", Scope: "ws-x", Revision: 1, Status: "valid"}}, Truncated: true}
	encoded, _ = json.Marshal(evidence)
	for _, key := range []string{`"items"`, `"truncated"`, `"scope"`, `"revision"`, `"status"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("evidence page must expose scope/revision/status envelopes: %s", encoded)
		}
	}
}

func TestCapabilitiesRequiredForPrimaryWriter(t *testing.T) {
	full := memory.Capabilities{Search: true, Expand: true, Mutate: true, MutationLookup: true}
	if !full.ServesPrimaryWriter() {
		t.Fatalf("first four capabilities must satisfy primary writer")
	}
	if (memory.Capabilities{Search: true, Expand: true, Mutate: true, Vector: true, Timeline: true, KnowledgeGraph: true}).ServesPrimaryWriter() {
		t.Fatalf("mutation_lookup missing must fail primary writer")
	}
	if (memory.Capabilities{}).ServesPrimaryWriter() {
		t.Fatalf("empty capabilities imply nothing")
	}
}

func TestHealthVocabulary(t *testing.T) {
	for _, status := range []memory.HealthStatus{memory.HealthAvailable, memory.HealthDegraded, memory.HealthUnavailable} {
		h := memory.Health{Status: status, ReasonCode: "ok", DerivedIndexState: "ready"}
		encoded, err := json.Marshal(h)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encoded), `"derived_index_state"`) {
			t.Fatalf("health must carry derived_index_state")
		}
	}
}

func TestStrictDecodeRejectsUnknownFields(t *testing.T) {
	valid := `{"scopes":[{"subject_id":"a","kind":"workspace","workspace_id":"ws-x"}],"query":"q","cursor":"","limit":5,"budget_chars":100}`
	if _, err := memory.DecodeAuthorizedSearch([]byte(valid)); err != nil {
		t.Fatalf("valid search rejected: %v", err)
	}
	for name, wire := range map[string]string{
		"unknown field": `{"scopes":[],"query":"q","cursor":"","limit":5,"budget_chars":100,"vendor":"x"}`,
		"invalid scope": `{"scopes":[{"subject_id":"a","kind":"vault","workspace_id":"x"}],"query":"q","cursor":"","limit":5,"budget_chars":100}`,
		"duplicate key": `{"scopes":[],"query":"q","query":"r","cursor":"","limit":5,"budget_chars":100}`,
		"trailing":      valid + ` {}`,
	} {
		if _, err := memory.DecodeAuthorizedSearch([]byte(wire)); err == nil {
			t.Fatalf("%s: invalid search accepted", name)
		}
	}
}

// Compile-time check: the backend contract is implementable.
type fakeBackend struct{}

func (fakeBackend) Capabilities() memory.Capabilities { return memory.Capabilities{} }
func (fakeBackend) Search(context.Context, memory.AuthorizedSearch) (memory.CardPage, error) {
	return memory.CardPage{}, nil
}
func (fakeBackend) Expand(context.Context, memory.AuthorizedExpansion) (memory.EvidencePage, error) {
	return memory.EvidencePage{}, nil
}
func (fakeBackend) Mutate(context.Context, memory.AuthorizedMutation) (memory.MutationReceipt, error) {
	return memory.MutationReceipt{}, nil
}
func (fakeBackend) MutationStatus(context.Context, string) (memory.MutationReceipt, error) {
	return memory.MutationReceipt{}, nil
}
func (fakeBackend) Health(context.Context) (memory.Health, error) { return memory.Health{}, nil }
func (fakeBackend) Close() error                                  { return nil }

var _ memory.Backend = fakeBackend{}
