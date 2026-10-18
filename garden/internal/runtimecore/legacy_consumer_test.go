package runtimecore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/ingest"
	"github.com/ProjectViVy/laputa/garden/internal/sqliteconn"
	"github.com/ProjectViVy/laputa/laputa/actmem"
)

// Missing runtime-owned binding injection or treating v3 as retryable legacy
// failure leaves this real Start/restart queue pending forever.
func TestIssue58LegacyConsumerRealOpenRestart(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	binding := actmem.RecapBinding{SubjectID: "actual-subject", DomainID: "complete-domain"}
	cfg.ResolveRecapBinding = func(context.Context, string) (actmem.RecapBinding, error) { return binding, nil }
	if err := os.MkdirAll(filepath.Dir(cfg.StateDB), 0700); err != nil {
		t.Fatal(err)
	}
	seed, err := ingest.OpenPaused(cfg.StateDB, nil)
	if err != nil {
		t.Fatal(err)
	}
	seed.Actmem, seed.ProfileID = actmem.New(cfg.PersonaDir), cfg.ProfileID
	content := "kept legacy source"
	accepted, err := seed.Submit(ctx, ingest.SubmitRequest{SessionID: "legacy", EventID: "legacy-event", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content))), Activity: &ingest.CaptureActivity{Phase: "completed", UserText: content}})
	if err != nil {
		t.Fatal(err)
	}
	if err = seed.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sqliteconn.Open(cfg.StateDB)
	if err != nil {
		t.Fatal(err)
	}
	// The material already committed; only ACTMEM activity delivery is pending.
	if _, err = db.Exec(`UPDATE ingestions SET status='completed'`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE updates(n INTEGER); INSERT INTO updates VALUES(0); CREATE TRIGGER count_activity AFTER UPDATE OF activity_json ON ingestions BEGIN UPDATE updates SET n=n+1; END`); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner, err := actmem.Open(cfg.PersonaDir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := owner.CommitRecap(ctx, actmem.RecapCommit{Binding: binding, OperationID: "activate-v3", Cause: "edit", Text: "不修改的主体记忆"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(owner.HeadPath())
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.Close(); err != nil {
		t.Fatal(err)
	}
	for restart := 0; restart < 2; restart++ {
		core, err := Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer core.Close()
		deadline := time.Now().Add(3 * time.Second)
		var result map[string]any
		for {
			projection, err := core.Ingest.ActivityProjection(ctx, accepted.IngestionID)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(projection)
			if err = json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			if result["legacy_consumer"] != nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("real worker never closed incompatible consumer: %s", raw)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if result["status"] != "pending" || result["revision"] != nil || result["entries"] != nil {
			t.Fatalf("policy closure fabricated effect: %+v", result)
		}
		if high, err := core.Ingest.HighWatermark(ctx, ""); err != nil || high != accepted.Seq {
			t.Fatalf("closed consumer still fences: %d %v", high, err)
		}
		archiveCtx, cancel := context.WithTimeout(ctx, time.Second)
		err = core.Ingest.ArchiveSession(archiveCtx, "legacy")
		cancel()
		if err != nil {
			t.Fatalf("v3 archive lifecycle still called legacy fold: %v", err)
		}
		// Observe longer than the existing one-second retry interval.
		time.Sleep(1200 * time.Millisecond)
		var writes int
		if err = db.QueryRow(`SELECT n FROM updates`).Scan(&writes); err != nil || writes != 1 {
			t.Fatalf("policy closure retries/rewrite: %d %v", writes, err)
		}
		after, err := os.ReadFile(core.Actmem.HeadPath())
		if err != nil || !reflect.DeepEqual(after, before) {
			t.Fatal("queue changed C1 bytes")
		}
		got, err := core.Actmem.ReadRecap(ctx, binding)
		if err != nil || got != head {
			t.Fatalf("queue advanced C1: %+v %v", got, err)
		}
		if err = core.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
