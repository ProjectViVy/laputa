package activity

import (
	"context"
	"testing"
)

func TestCheckpointSaveAndLoad(t *testing.T) {
	store, err := OpenCheckpointStore(t.TempDir() + "\\garden.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ws := NewWorkingSet()
	ws.Update("project:garden", []string{"mem_1", "mem_2"}, []string{"ref_a"})
	cp := &Checkpointer{Store: store, WS: ws}
	if err := cp.Save(context.Background(), "project:garden"); err != nil {
		t.Fatal(err)
	}

	ws2 := NewWorkingSet()
	cp2 := &Checkpointer{Store: store, WS: ws2}
	if err := cp2.Load(context.Background(), "project:garden"); err != nil {
		t.Fatal(err)
	}
	snap := ws2.Get("project:garden")
	if len(snap.ActiveCardIDs) != 2 || snap.ActiveCardIDs[0] != "mem_1" {
		t.Fatalf("restored=%v", snap)
	}
	if len(snap.EvidenceRefs) != 1 || snap.EvidenceRefs[0] != "ref_a" {
		t.Fatalf("refs=%v", snap)
	}
}

func TestCheckpointSurvivesStoreRestart(t *testing.T) {
	path := t.TempDir() + "\\garden.db"
	store, err := OpenCheckpointStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ws := NewWorkingSet()
	ws.Update("scope", []string{"card"}, []string{"evidence"})
	if err := (&Checkpointer{Store: store, WS: ws}).Save(context.Background(), "scope"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenCheckpointStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restored := NewWorkingSet()
	if err := (&Checkpointer{Store: store, WS: restored}).Load(context.Background(), "scope"); err != nil {
		t.Fatal(err)
	}
	if got := restored.Get("scope"); len(got.ActiveCardIDs) != 1 || got.ActiveCardIDs[0] != "card" {
		t.Fatalf("restored=%v", got)
	}
}

func TestCheckpointUnavailable(t *testing.T) {
	ws := NewWorkingSet()
	cp := &Checkpointer{Store: nil, WS: ws}
	if err := cp.Save(context.Background(), ""); err == nil {
		t.Fatal("expected unavailable error")
	}
	if err := cp.Load(context.Background(), ""); err == nil {
		t.Fatal("expected unavailable error")
	}
}

func TestCheckpointLoadEmpty(t *testing.T) {
	store, err := OpenCheckpointStore(t.TempDir() + "\\garden.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ws := NewWorkingSet()
	if err := (&Checkpointer{Store: store, WS: ws}).Load(context.Background(), "scope"); err != nil {
		t.Fatal(err)
	}
	if got := ws.Get("scope"); len(got.ActiveCardIDs) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
