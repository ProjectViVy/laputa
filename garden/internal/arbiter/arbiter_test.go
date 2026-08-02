package arbiter

import "testing"

func TestDetectConflictsReplace(t *testing.T) {
	a := New()
	assertions := []Assertion{
		{Subject: "garden", Predicate: "leads", Object: "alice", ValidFrom: "2026-01-01", Confidence: 0.9, Status: "active"},
		{Subject: "garden", Predicate: "leads", Object: "bob", ValidFrom: "2026-03-01", Confidence: 0.7, Status: "active"},
	}
	proposals := a.DetectConflicts(assertions)
	if len(proposals) != 1 {
		t.Fatalf("proposals=%d, want 1", len(proposals))
	}
	p := proposals[0]
	if p.ConflictType != ConflictReplace {
		t.Fatalf("type=%s, want replace", p.ConflictType)
	}
	if len(p.SupersedesEdges) != 1 {
		t.Fatalf("edges=%v", p.SupersedesEdges)
	}
	if p.SupersedesEdges[0].From != "garden/leads/alice" || p.SupersedesEdges[0].To != "garden/leads/bob" {
		t.Fatalf("edge=%+v", p.SupersedesEdges[0])
	}
	if len(p.ValidityChanges) != 1 || p.ValidityChanges[0].AssertionKey != "garden/leads/bob" {
		t.Fatalf("validity=%+v", p.ValidityChanges)
	}
}

func TestDetectConflictCoexist(t *testing.T) {
	a := New()
	assertions := []Assertion{
		{Subject: "garden", Predicate: "leads", Object: "alice", ValidFrom: "2026-01-01", ValidTo: "2026-02-28", Confidence: 0.9, Status: "superseded"},
		{Subject: "garden", Predicate: "leads", Object: "bob", ValidFrom: "2026-03-01", Confidence: 0.8, Status: "active"},
	}
	proposals := a.DetectConflicts(assertions)
	if len(proposals) != 1 {
		t.Fatalf("proposals=%d, want 1", len(proposals))
	}
	if proposals[0].ConflictType != ConflictCoexist {
		t.Fatalf("type=%s, want coexist", proposals[0].ConflictType)
	}
}

func TestDetectConflictRefine(t *testing.T) {
	a := New()
	assertions := []Assertion{
		{Subject: "garden", Predicate: "status", Object: "active", ValidFrom: "2026-01-01", Confidence: 0.75, Status: "active"},
		{Subject: "garden", Predicate: "status", Object: "paused", ValidFrom: "2026-02-01", Confidence: 0.72, Status: "active"},
	}
	proposals := a.DetectConflicts(assertions)
	if len(proposals) != 1 {
		t.Fatalf("proposals=%d, want 1", len(proposals))
	}
	if proposals[0].ConflictType != ConflictRefine {
		t.Fatalf("type=%s, want refine", proposals[0].ConflictType)
	}
}

func TestNoConflictSingleAssertion(t *testing.T) {
	a := New()
	assertions := []Assertion{
		{Subject: "garden", Predicate: "leads", Object: "alice", ValidFrom: "2026-01-01", Confidence: 0.9, Status: "active"},
	}
	proposals := a.DetectConflicts(assertions)
	if len(proposals) != 0 {
		t.Fatalf("proposals=%d, want 0", len(proposals))
	}
}
