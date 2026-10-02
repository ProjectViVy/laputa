package personactx

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dashimaki/laputa/evolution"
	"github.com/dashimaki/laputa/persona"
)

// initMissionPersona opens a Persona store with the five required files.
func initMissionPersona(t *testing.T) *persona.Service {
	t.Helper()
	svc, err := persona.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Initialize(persona.Initialization{
		Identity:     "identity body",
		Relationship: "relationship body",
		Redline:      "redline body",
		User:         "## Preferences\nuser preferences",
		World:        "world body",
	}, "user", persona.SourceInit, "init"); err != nil {
		t.Fatal(err)
	}
	return svc
}

func writeMission(t *testing.T, svc *persona.Service, content string, base uint64) {
	t.Helper()
	outcome, err := svc.SaveUserDocument(persona.KindMission, content, base, "user", persona.SourceUserDirect, "set mission")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Document.Revision != base+1 {
		t.Fatalf("mission revision=%d want %d", outcome.Document.Revision, base+1)
	}
}

// TestCaptureV2MissionAssigned: a v2 capture carries the exact seven-slot
// roster, the mission slot first, verbatim content, and the assigned status.
func TestCaptureV2MissionAssigned(t *testing.T) {
	svc := initMissionPersona(t)
	writeMission(t, svc, "protect the archive", 0)

	core, err := Capture(svc, "s1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if core.SchemaVersion != evolution.FrozenCoreV2SchemaVersion {
		t.Fatalf("schema_version=%q", core.SchemaVersion)
	}
	if err := core.Validate(); err != nil {
		t.Fatalf("capture must be a valid v2 envelope: %v", err)
	}
	if core.MissionStatus != evolution.MissionAssigned {
		t.Fatalf("mission_status=%q", core.MissionStatus)
	}
	if core.Sections[0].Kind != evolution.FrozenKindMission || core.Sections[0].Content != "protect the archive" || core.Sections[0].SourceRevision != 1 {
		t.Fatalf("mission slot=%+v", core.Sections[0])
	}
	if len(core.Sections) != 7 {
		t.Fatalf("sections=%d want 7", len(core.Sections))
	}
	for _, section := range core.Sections {
		if section.Kind == evolution.FrozenCoreKind("world") || section.Kind == evolution.FrozenCoreKind("actmem") {
			t.Fatalf("tool-only kind %q entered frozen core", section.Kind)
		}
	}
}

// TestCaptureV2MissionUnassigned: a missing Mission is an explicit unassigned
// envelope — never generated, never blocking capture.
func TestCaptureV2MissionUnassigned(t *testing.T) {
	svc := initMissionPersona(t)
	core, err := Capture(svc, "s1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if core.MissionStatus != evolution.MissionUnassigned {
		t.Fatalf("mission_status=%q", core.MissionStatus)
	}
	mission := core.Sections[0]
	if mission.Content != "" || mission.SourceRevision != 0 {
		t.Fatalf("unassigned mission slot=%+v", mission)
	}
}

// TestStoreVersionsSnapshots: a new session sees the new mission revision; a
// resumed session keeps its admitted snapshot forever.
func TestStoreVersionsSnapshots(t *testing.T) {
	svc := initMissionPersona(t)
	writeMission(t, svc, "mission v1", 0)
	store, err := OpenStore(filepath.Join(t.TempDir(), "frozen.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	old, err := store.Capture(ctx, "session-old", svc)
	if err != nil {
		t.Fatal(err)
	}
	writeMission(t, svc, "mission v2", 1)

	fresh, err := store.Capture(ctx, "session-new", svc)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := store.Get(ctx, "session-old")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Sections[0].SourceRevision != 2 || fresh.Sections[0].Content != "mission v2" {
		t.Fatalf("new session mission slot=%+v", fresh.Sections[0])
	}
	if resumed.Sections[0].SourceRevision != 1 || resumed.Sections[0].Content != "mission v1" {
		t.Fatalf("resumed session drifted: %+v", resumed.Sections[0])
	}
	if resumed.Sections[0] != old.Sections[0] {
		t.Fatalf("admitted snapshot changed: before=%+v after=%+v", old.Sections[0], resumed.Sections[0])
	}
}

// TestStoreV1AdmissionRequiresRenewal: a pre-Mission v1 row can never be
// relabelled v2; the store surfaces an explicit session-renewal requirement.
func TestStoreV1AdmissionRequiresRenewal(t *testing.T) {
	svc := initMissionPersona(t)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "frozen.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Seed a legacy v1-shaped row: numeric sections, no schema_version.
	v1 := `{"session_id":"legacy","captured_at":"2026-08-14T00:00:00Z","sections":[{"section":0,"content":"old identity","source_revision":1,"source_hash":"h"}]}`
	if _, err := store.db.ExecContext(ctx, `INSERT INTO frozen_core_sessions(session_id,captured_at,core_json) VALUES('legacy','2026-08-14T00:00:00Z',?)`, v1); err != nil {
		t.Fatal(err)
	}

	for _, op := range []string{"Get", "Capture"} {
		var err error
		if op == "Get" {
			_, err = store.Get(ctx, "legacy")
		} else {
			_, err = store.Capture(ctx, "legacy", svc)
		}
		if !errors.Is(err, ErrSessionRenewalRequired) {
			t.Fatalf("%s on v1 row err=%v, want ErrSessionRenewalRequired", op, err)
		}
		if evolution.CodeOf(err) != evolution.ErrRecoveryRequired {
			t.Fatalf("%s code=%q want recovery_required", op, evolution.CodeOf(err))
		}
	}
}

// TestMissionRevisionPinBlocksStaleEffects: an autonomous run pinned to the
// old mission revision is blocked after a human edit.
func TestMissionRevisionPinBlocksStaleEffects(t *testing.T) {
	svc := initMissionPersona(t)
	writeMission(t, svc, "mission v1", 0)
	binding := evolution.RunBinding{
		SubjectID:       "sub",
		DestinationID:   "dest",
		PolicyRevision:  "p1",
		StrategyDigest:  "d",
		MissionRevision: 1,
	}

	writeMission(t, svc, "mission v2", 1)
	doc, err := svc.GetDocument(persona.KindMission)
	if err != nil {
		t.Fatal(err)
	}
	err = binding.CheckMissionRevision(doc.Revision)
	if evolution.CodeOf(err) != evolution.ErrMissionRevisionChanged {
		t.Fatalf("stale pin err=%v, want mission_revision_changed", err)
	}
	if err := binding.CheckMissionRevision(1); err != nil {
		t.Fatalf("matching revision must pass: %v", err)
	}
}

// TestRenderMissionFirst: the mission section renders verbatim before the
// bounded six.
func TestRenderMissionFirst(t *testing.T) {
	svc := initMissionPersona(t)
	writeMission(t, svc, "mission first", 0)
	core, err := Capture(svc, "s1", nil)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(core, 8000)
	if !strings.Contains(out, "## Frozen Core — mission\nmission first") {
		t.Fatalf("render missing mission: %q", out)
	}
	if strings.Index(out, "mission first") > strings.Index(out, "identity body") {
		t.Fatalf("mission must render first: %q", out)
	}
}
