package agentapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/memory"
	"github.com/ProjectViVy/laputa/laputa/evolution"
)

// These tests open the production Garden/Mentle composition. No backend is
// injected; all authority and canonical files live under a temporary root.
func memoryLoopRealConfig(t *testing.T) Config {
	t.Helper()
	cfg := userEmbeddedConfig(t)
	models, err := filepath.Abs(filepath.Join("..", "..", "mentle", "models"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelsDir = models
	cfg.RequireLocalModel = true
	initAuthority(t, cfg)
	return cfg
}

func memoryLoopMutation(cfg Config) memory.AuthorizedMutation {
	return memory.AuthorizedMutation{
		Scope:         evolution.Scope{SubjectID: cfg.ProfileID, Kind: evolution.ScopePersonal},
		DestinationID: BackendMentle, Operation: evolution.MutationCreate,
		RecordID: "mem_loop_probe", ExpectedAbsent: true,
		Body:      "memoryloopuniquefact: the synthetic vault code is heliotrope-7826",
		Inference: evolution.InferenceObserved,
	}
}

func TestMemoryLoopRealBackendRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cfg := memoryLoopRealConfig(t)
	client, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("real runtime open: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	human, err := client.BindHumanSession("memory-loop-a", "")
	if err != nil {
		t.Fatal(err)
	}
	mutation := memoryLoopMutation(cfg)
	receipt, err := human.MutateMemory(ctx, mutation)
	if err != nil {
		t.Fatalf("canonical write: %v", err)
	}
	if receipt.CanonicalStatus != memory.CanonicalCompleted || receipt.Revision != 1 || receipt.OperationID == "" {
		t.Fatalf("canonical receipt: %+v", receipt)
	}
	check := func(human *HumanClient) {
		t.Helper()
		lookup, err := human.MemoryReceipt(ctx, receipt.OperationID)
		if err != nil || lookup.OperationID != receipt.OperationID || lookup.TargetRef != receipt.TargetRef || lookup.Revision != receipt.Revision {
			t.Fatalf("receipt lookup: %+v err=%v want=%+v", lookup, err, receipt)
		}
		page, err := human.SearchMemory(ctx, memory.AuthorizedSearch{Query: "memoryloopuniquefact", Limit: 10, BudgetChars: 4096})
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("card search: %+v err=%v", page, err)
		}
		card := page.Items[0]
		// The human capability mints record identity on the trusted side;
		// a model/browser-supplied create ID is not the canonical identity.
		if card.ID != receipt.TargetRef || uint64(card.Revision) != receipt.Revision {
			t.Fatalf("card/receipt identity mismatch: %+v %+v", card, receipt)
		}
		expanded, err := human.ExpandMemory(ctx, memory.AuthorizedExpansion{CardID: card.ID, ExpectedRevision: receipt.Revision, BudgetChars: 4096})
		if err != nil || len(expanded.Items) == 0 {
			t.Fatalf("expansion: %+v err=%v", expanded, err)
		}
		body := ""
		for _, fragment := range expanded.Items {
			if fragment.CardID != card.ID || fragment.Revision != receipt.Revision || fragment.ContentHash == "" {
				t.Fatalf("evidence identity/hash mismatch: %+v", fragment)
			}
			body += fragment.Excerpt
		}
		if !strings.Contains(body, mutation.Body) {
			t.Fatalf("durable body lost: %q", body)
		}
		t.Logf("operation=%s record=%s revision=%d canonical=%s index=%s", receipt.OperationID, card.ID, receipt.Revision, receipt.CanonicalStatus, receipt.IndexStatus)
	}
	check(human)
	check(human) // repeated reads must not create canonical effects
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	human, err = reopened.BindHumanSession("memory-loop-b", "")
	if err != nil {
		t.Fatal(err)
	}
	check(human)
}

func TestMemoryLoopBackendAvailabilityMatrix(t *testing.T) {
	t.Run("new-without-model", func(t *testing.T) {
		cfg := userEmbeddedConfig(t)
		initAuthority(t, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		client, err := Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		human, err := client.BindHumanSession("fresh-missing", "")
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := human.MutateMemory(ctx, memoryLoopMutation(cfg))
		if err == nil || receipt.CanonicalStatus == memory.CanonicalCompleted {
			t.Fatalf("missing model falsely writable: %+v %v", receipt, err)
		}
		t.Logf("expected unavailable: %v", err)
	})
	t.Run("normal-model", func(t *testing.T) {
		cfg := memoryLoopRealConfig(t)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		client, err := Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		human, err := client.BindHumanSession("normal", "")
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := human.MutateMemory(ctx, memoryLoopMutation(cfg))
		if err != nil || receipt.CanonicalStatus != memory.CanonicalCompleted {
			t.Fatalf("normal write: %+v %v", receipt, err)
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		cfg.ModelsDir = filepath.Join(t.TempDir(), "missing")
		cfg.RequireLocalModel = false
		offline, err := Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer offline.Close()
		human, err = offline.BindHumanSession("offline", "")
		if err != nil {
			t.Fatal(err)
		}
		page, err := human.SearchMemory(ctx, memory.AuthorizedSearch{Query: "memoryloopuniquefact", Limit: 10})
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("offline canonical read: %+v %v", page, err)
		}
		mutation := memoryLoopMutation(cfg)
		mutation.RecordID = "mem_offline_should_not_write"
		rejected, err := human.MutateMemory(ctx, mutation)
		if err == nil || rejected.CanonicalStatus == memory.CanonicalCompleted {
			t.Fatalf("readonly falsely writable: %+v %v", rejected, err)
		}
		t.Logf("expected readonly: %v", err)
	})
	t.Run("corrupt-model", func(t *testing.T) {
		cfg := userEmbeddedConfig(t)
		initAuthority(t, cfg)
		if err := os.MkdirAll(cfg.ModelsDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfg.ModelsDir, "model.onnx"), []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		client, err := Open(ctx, cfg)
		if client != nil {
			defer client.Close()
		}
		if err == nil {
			t.Fatal("corrupt model silently accepted")
		}
		t.Logf("expected startup failure: %v", err)
	})
}
