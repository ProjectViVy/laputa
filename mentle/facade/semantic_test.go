package facade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/mentle/internal/hybrid"
	"github.com/ProjectViVy/laputa/mentle/internal/search"
	"github.com/ProjectViVy/laputa/mentle/storage/govector"
)

func semanticTestService(t *testing.T) *Service {
	t.Helper()
	store := &fakeStore{points: map[string]govector.SearchResult{}}
	embedder := fakeEmbedder{}
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "canonical.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { catalog.Close() })
	return &Service{Searcher: search.NewSearcher(store, embedder), Hybrid: hybrid.NewSearcher(store, embedder, .7), Catalog: catalog}
}

func TestCreateMemoryAcceptsArtifactKinds(t *testing.T) {
	svc := semanticTestService(t)
	ctx := context.Background()
	for _, kind := range []string{"source_artifact", "semantic_unit"} {
		if _, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "content for " + kind, Kind: kind}, "", ""); err != nil {
			t.Fatalf("kind %s rejected: %v", kind, err)
		}
	}
	if _, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: "x", Kind: "bogus"}, "", ""); err == nil {
		t.Fatal("unknown kind accepted")
	}
}

func TestProvenanceRoundTrip(t *testing.T) {
	svc := semanticTestService(t)
	ctx := context.Background()
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{
		Content: "unit body",
		Kind:    "semantic_unit",
		Source:  MemorySource{Type: "import", URI: "vault/notes/example.md", Revision: "rev-7"},
		Metadata: map[string]any{
			"source_path":  "vault/notes/example.md",
			"heading_path": []string{"Architecture", "Recall"},
			"start_offset": 10,
			"end_offset":   19,
			"content_hash": "sha256:abc",
			"source_ref":   "mem_parent",
			"sync_state":   "current",
		},
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetMemory(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source.Type != "import" || got.Source.URI != "vault/notes/example.md" || got.Source.Revision != "rev-7" {
		t.Fatalf("source=%+v", got.Source)
	}
	if got.Metadata["source_path"] != "vault/notes/example.md" || got.Metadata["source_ref"] != "mem_parent" {
		t.Fatalf("metadata=%+v", got.Metadata)
	}
}

func TestEvidenceRealOffsetsAndExcerptHash(t *testing.T) {
	svc := semanticTestService(t)
	ctx := context.Background()
	content := "header paragraph\n\nunit body text here\n\ntail"
	start, end := 18, 37
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{
		Content: content,
		Kind:    "semantic_unit",
		Source:  MemorySource{Type: "import", URI: "vault/notes/example.md", Revision: "rev-7"},
		Metadata: map[string]any{
			"start_offset": start,
			"end_offset":   end,
		},
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	frags, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{created.ID}, PerItemBudget: 800, TotalBudget: 4000})
	if err != nil || len(frags) != 1 {
		t.Fatalf("frags=%+v err=%v", frags, err)
	}
	f := frags[0]
	if f.Excerpt != content[start:end] {
		t.Fatalf("excerpt=%q", f.Excerpt)
	}
	if f.StartOffset != start || f.EndOffset != end {
		t.Fatalf("offsets=%d..%d", f.StartOffset, f.EndOffset)
	}
	want := sha256.Sum256([]byte(f.Excerpt))
	if f.ContentHash != hex.EncodeToString(want[:]) {
		t.Fatalf("hash=%s", f.ContentHash)
	}
	if f.SourceURI != "vault/notes/example.md" || f.SourceRev != "rev-7" {
		t.Fatalf("source fields uri=%q rev=%q", f.SourceURI, f.SourceRev)
	}
}

func TestEvidenceLegacySyntheticOffsets(t *testing.T) {
	svc := semanticTestService(t)
	ctx := context.Background()
	content := "legacy memory content"
	created, err := svc.CreateMemory(ctx, CreateMemoryRequest{Content: content, Kind: "note"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	frags, err := svc.ReadEvidence(ctx, EvidenceQuery{CardIDs: []string{created.ID}})
	if err != nil || len(frags) != 1 {
		t.Fatalf("frags=%+v err=%v", frags, err)
	}
	f := frags[0]
	if f.StartOffset != 0 {
		t.Fatalf("start=%d", f.StartOffset)
	}
	want := sha256.Sum256([]byte(content))
	if f.ContentHash != hex.EncodeToString(want[:]) {
		t.Fatalf("hash=%s", f.ContentHash)
	}
}
