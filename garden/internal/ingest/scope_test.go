package ingest

import (
	"context"
	"testing"

	"github.com/ProjectViVy/laputa/garden/internal/activity"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"github.com/ProjectViVy/laputa/mentle/facade"
)

// A workspace submission writes under the encoded workspace scope, never a
// caller-supplied scope string.
func TestProcessUsesWorkspaceScope(t *testing.T) {
	ctx := context.Background()
	var got facade.CreateMemoryRequest
	svc, err := OpenPaused(t.TempDir()+"/ingest.db", drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		got = req
		return facade.Memory{ID: "mem_1"}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	svc.ProfileID = "profile_1"
	acc, err := svc.Submit(ctx, SubmitRequest{SessionID: "s", EventID: "e", Phase: "session_end", Content: "body", ContentHash: "sha256:230d8358dc8e8890b4c58deeb62912ee2f20357ae92a5cc861b98e68fe31acb5", Workspace: "ws-9"})
	if err != nil {
		t.Fatal(err)
	}
	svc.process(ctx, acc.IngestionID)
	want := memory.EncodeScope(evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-9"})
	if got.Scope != want {
		t.Fatalf("scope = %q want %q", got.Scope, want)
	}
}

// A spooled entry with an unclassifiable scope fails visibly and is drained
// — it is never re-routed into another scope.
func TestDrainSpoolFailsUnclassifiedScope(t *testing.T) {
	ctx := context.Background()
	var writes []string
	svc, _ := newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		writes = append(writes, req.Scope)
		return facade.Memory{ID: "mem_x"}, nil
	}))
	entry := activity.TransientEntry{EventID: "event-legacy", SessionID: "session-legacy", ContentHash: "hash-legacy", Content: "legacy", Kind: "source_artifact", Scope: "project:garden"}
	if svc.Spool == nil {
		t.Fatal("fixture has no spool")
	}
	if err := svc.Spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO ingestions(ingestion_id,session_id,event_id,phase,content,content_hash,occurred_at,status,error,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, "ing_legacy", entry.SessionID, entry.EventID, "session_end", entry.Content, entry.ContentHash, "now", "spooled", "mentle unavailable; spooled", "now", "now"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DrainSpool(ctx); err != nil {
		t.Fatalf("DrainSpool() = %v", err)
	}
	for _, s := range writes {
		if s == "project:garden" {
			t.Fatal("unclassified scope reached the writer")
		}
	}
	status, err := svc.GetByIdentity(ctx, "ing_legacy", "session-legacy", "event-legacy")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "failed" {
		t.Fatalf("status = %q, want failed", status.Status)
	}
}

// A spooled entry drains under its stored scope after restart.
func TestDrainSpoolUsesStoredScope(t *testing.T) {
	ctx := context.Background()
	var got facade.CreateMemoryRequest
	svc, _ := newDrainFixture(t, drainWriterFunc(func(_ context.Context, req facade.CreateMemoryRequest, _, _ string) (facade.Memory, error) {
		got = req
		return facade.Memory{ID: "mem_drain"}, nil
	}))
	want := memory.EncodeScope(evolution.Scope{SubjectID: "profile_1", Kind: evolution.ScopeWorkspace, WorkspaceID: "ws-drain"})
	entry := activity.TransientEntry{EventID: "event-drain2", SessionID: "session-drain2", ContentHash: "hash-drain2", Content: "recover me too", Kind: "source_artifact", Scope: want}
	if err := svc.Spool.Append(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DrainSpool(ctx); err != nil {
		t.Fatal(err)
	}
	if got.Scope != want {
		t.Fatalf("drained scope = %q want %q", got.Scope, want)
	}
}
