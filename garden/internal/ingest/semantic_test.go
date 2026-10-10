package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ProjectViVy/laputa/mentle/facade"
)

type fakeMemoryWriter struct {
	mu    sync.Mutex
	memos map[string]facade.Memory
	calls []facade.CreateMemoryRequest
	seq   int
}

func (f *fakeMemoryWriter) CreateMemory(_ context.Context, req facade.CreateMemoryRequest, idempotencyKey, bodyHash string) (facade.Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	if m, ok := f.memos[idempotencyKey]; ok {
		return m, nil
	}
	f.seq++
	m := facade.Memory{ID: fmt.Sprintf("mem_%d", f.seq), Kind: req.Kind, Content: req.Content, Version: 1, Source: req.Source, Metadata: req.Metadata}
	f.memos[idempotencyKey] = m
	return m, nil
}

func semanticTestService(t *testing.T, writer MemoryWriter) *Service {
	t.Helper()
	svc, err := Open(filepath.Join(t.TempDir(), "ingest.db"), writer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func TestIngestSemanticUnitsCreatesAndDeduplicates(t *testing.T) {
	writer := &fakeMemoryWriter{memos: map[string]facade.Memory{}}
	svc := semanticTestService(t, writer)
	units := []SemanticUnit{
		{Content: "first block", SourcePath: "vault/notes/example.md", HeadingPath: []string{"Intro"}, StartOffset: 0, EndOffset: 11, Scope: "project:garden"},
		{Content: "second block", SourcePath: "vault/notes/example.md", HeadingPath: []string{"Intro", "Details"}, StartOffset: 12, EndOffset: 24},
	}
	first, err := svc.IngestSemanticUnits(context.Background(), units)
	if err != nil {
		t.Fatal(err)
	}
	if first.Created != 2 || first.Skipped != 0 || len(first.MemoryIDs) != 2 {
		t.Fatalf("first=%+v", first)
	}
	if writer.calls[0].Kind != "semantic_unit" || writer.calls[0].Source.Type != "import" || writer.calls[0].Source.URI != "vault/notes/example.md" {
		t.Fatalf("call=%+v", writer.calls[0])
	}
	if writer.calls[0].Metadata["heading_path"] == nil || writer.calls[0].Metadata["content_hash"] == "" {
		t.Fatalf("metadata=%+v", writer.calls[0].Metadata)
	}

	replay, err := svc.IngestSemanticUnits(context.Background(), units)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.MemoryIDs) != 2 || replay.MemoryIDs[0] != first.MemoryIDs[0] || replay.MemoryIDs[1] != first.MemoryIDs[1] {
		t.Fatalf("replay=%+v want ids=%v", replay, first.MemoryIDs)
	}
	if len(writer.memos) != 2 {
		t.Fatalf("mentle-side memories=%d, want 2 (idempotent)", len(writer.memos))
	}

	dup, err := svc.IngestSemanticUnits(context.Background(), []SemanticUnit{units[0], units[0]})
	if err != nil {
		t.Fatal(err)
	}
	if dup.Created != 1 || dup.Skipped != 1 {
		t.Fatalf("dup=%+v", dup)
	}
}

func TestIngestSemanticUnitsValidation(t *testing.T) {
	writer := &fakeMemoryWriter{memos: map[string]facade.Memory{}}
	svc := semanticTestService(t, writer)
	if _, err := svc.IngestSemanticUnits(context.Background(), []SemanticUnit{{Content: "x"}}); err == nil {
		t.Fatal("missing source_path accepted")
	}
	if _, err := svc.IngestSemanticUnits(context.Background(), []SemanticUnit{{SourcePath: "p"}}); err == nil {
		t.Fatal("missing content accepted")
	}
}

func TestIngestSemanticUnitsUnavailable(t *testing.T) {
	svc := semanticTestService(t, nil)
	if _, err := svc.IngestSemanticUnits(context.Background(), []SemanticUnit{{Content: "x", SourcePath: "p"}}); err != facade.ErrUnavailable {
		t.Fatalf("err=%v", err)
	}
}
