package activity

import (
	"context"
	"testing"

	"github.com/dashimaki/laputa/governance"
)

type fakeGovWriter struct {
	sections map[governance.SectionName]map[string]any
}

func (f *fakeGovWriter) GetSection(_ context.Context, name governance.SectionName) (map[string]any, error) {
	if data, ok := f.sections[name]; ok {
		return data, nil
	}
	return map[string]any{}, nil
}

func (f *fakeGovWriter) SetSection(_ context.Context, name governance.SectionName, data map[string]any) error {
	if f.sections == nil {
		f.sections = map[governance.SectionName]map[string]any{}
	}
	f.sections[name] = data
	return nil
}

func TestCheckpointSaveAndLoad(t *testing.T) {
	gov := &fakeGovWriter{sections: map[governance.SectionName]map[string]any{
		governance.SectionMemoryMD: {},
	}}
	ws := NewWorkingSet()
	ws.Update("project:garden", []string{"mem_1", "mem_2"}, []string{"ref_a"})

	cp := &Checkpointer{Gov: gov, WS: ws}
	if err := cp.Save(context.Background(), "project:garden"); err != nil {
		t.Fatal(err)
	}

	ws2 := NewWorkingSet()
	cp2 := &Checkpointer{Gov: gov, WS: ws2}
	if err := cp2.Load(context.Background(), "project:garden"); err != nil {
		t.Fatal(err)
	}
	snap := ws2.Get("project:garden")
	if len(snap.ActiveCardIDs) != 2 || snap.ActiveCardIDs[0] != "mem_1" {
		t.Fatalf("restored=%v", snap)
	}
	if len(snap.EvidenceRefs) != 1 || snap.EvidenceRefs[0] != "ref_a" {
		t.Fatalf("refs=%v", snap.EvidenceRefs)
	}
}

func TestCheckpointNilGov(t *testing.T) {
	ws := NewWorkingSet()
	cp := &Checkpointer{Gov: nil, WS: ws}
	if err := cp.Save(context.Background(), ""); err == nil {
		t.Fatal("expected error for nil gov")
	}
	if err := cp.Load(context.Background(), ""); err == nil {
		t.Fatal("expected error for nil gov")
	}
}

func TestCheckpointLoadEmpty(t *testing.T) {
	gov := &fakeGovWriter{sections: map[governance.SectionName]map[string]any{
		governance.SectionMemoryMD: {},
	}}
	ws := NewWorkingSet()
	cp := &Checkpointer{Gov: gov, WS: ws}
	if err := cp.Load(context.Background(), "scope"); err != nil {
		t.Fatal(err)
	}
	snap := ws.Get("scope")
	if len(snap.ActiveCardIDs) != 0 {
		t.Fatalf("expected empty, got %v", snap)
	}
}
