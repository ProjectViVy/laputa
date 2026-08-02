package facade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestReadEvidencePerItemBudget(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	longContent := strings.Repeat("evidence text ", 200)
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: longContent, Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{created.ID}, PerItemBudget: 100, TotalBudget: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}
	if len([]rune(fragments[0].Excerpt)) > 101 {
		t.Fatalf("excerpt exceeds per-item budget: %d runes", len([]rune(fragments[0].Excerpt)))
	}
}

func TestReadEvidenceTotalBudget(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	ids := make([]string, 5)
	for i := range ids {
		created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: strings.Repeat("x", 500), Kind: "fact"}, "k"+string(rune('a'+i)), "h"+string(rune('a'+i)))
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = created.ID
	}
	fragments, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: ids, PerItemBudget: 300, TotalBudget: 800})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, f := range fragments {
		total += len([]rune(f.Excerpt))
	}
	if total > 800 {
		t.Fatalf("total evidence %d exceeds budget 800", total)
	}
	if len(fragments) == 0 {
		t.Fatal("expected at least one fragment")
	}
}

func TestReadEvidenceContentHash(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	content := "hash verification content"
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: content, Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{created.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}
	hash := sha256.Sum256([]byte(content))
	expected := hex.EncodeToString(hash[:])
	if fragments[0].ContentHash != expected {
		t.Fatalf("hash=%q want %q", fragments[0].ContentHash, expected)
	}
}

func TestReadEvidenceMissingCard(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "real content", Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{"mem_nonexistent", created.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment (missing card skipped), got %d", len(fragments))
	}
	if fragments[0].CardID != created.ID {
		t.Fatalf("fragment card=%q want %q", fragments[0].CardID, created.ID)
	}
}

func TestReadEvidenceValidity(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "valid content", Kind: "fact"}, "k1", "h1")
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{created.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 || fragments[0].Validity != "active" {
		t.Fatalf("fragments=%+v", fragments)
	}
}

func TestReadEvidenceUnavailable(t *testing.T) {
	svc := &Service{}
	if _, err := svc.ReadEvidence(context.Background(), EvidenceQuery{CardIDs: []string{"x"}}); err == nil {
		t.Fatal("expected error on uninitialized service")
	}
}
