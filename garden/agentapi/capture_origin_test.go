package agentapi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
	"testing"
	"time"
)

func TestCaptureOriginAcrossRestart(t *testing.T) {
	ctx := context.Background()
	cfg := installationConfig(t)
	cfg.CatalogOnly = true
	i, err := OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	for _, id := range []string{"A", "B"} {
		installPersona(t, i, id)
	}
	receipts := map[string]CaptureReceipt{}
	for _, id := range []string{"A", "B"} {
		b, _ := installedClient(t, i, id).BindSession("same-session")
		body := "same captured fact"
		sum := sha256.Sum256([]byte(body))
		req := CaptureRequest{Phase: CaptureCompleted, Provenance: CaptureProvenance{RunID: "same-run", EventSeq: 1}, Content: body, ContentHash: fmt.Sprintf("sha256:%x", sum)}
		r, err := b.Capture(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		receipts[id] = r
		if r.SessionID != "same-session" {
			t.Fatal("lost host session", r)
		}
		again, err := b.Capture(ctx, req)
		if err != nil || again.IngestionID != r.IngestionID {
			t.Fatal("dedupe", again, err)
		}
	}
	if receipts["A"].IngestionID == receipts["B"].IngestionID {
		t.Fatal("cross-subject dedupe")
	}
	if err = i.Close(); err != nil {
		t.Fatal(err)
	}
	i, err = OpenInstallation(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	for _, id := range []string{"A", "B"} {
		b, _ := installedClient(t, i, id).BindSession("same-session")
		var st CaptureStatus
		until := time.Now().Add(5 * time.Second)
		for {
			st, err = b.CaptureStatus(ctx, receipts[id].IngestionID, receipts[id].EventID)
			if err != nil {
				t.Fatal(err)
			}
			if st.Status == "completed" {
				break
			}
			if time.Now().After(until) {
				t.Fatal("capture incomplete", st)
			}
			time.Sleep(10 * time.Millisecond)
		}
		root, _ := i.owner.Subject(ctx, id)
		r, err := root.Mentle.GetMemory(ctx, st.MemoryIDs[0])
		if err != nil {
			t.Fatal(err)
		}
		want := memory.EncodeScope(evolution.Scope{SubjectID: id, Kind: evolution.ScopePersonal})
		if r.Scope != want {
			t.Fatal("wrong restart scope", r.Scope, want)
		}
		foreign := receipts["B"]
		if id == "B" {
			foreign = receipts["A"]
		}
		if _, err = b.CaptureStatus(ctx, foreign.IngestionID, foreign.EventID); err == nil {
			t.Fatal("foreign receipt disclosed")
		}
	}
}
