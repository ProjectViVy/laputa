//go:build windows

package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/actmem"
)

// The real Start/restart positive-publication case is platform-neutral and also
// runs on Windows. Its unqualified missing-head/orphan view is never policy proof.
func TestIssue58LegacyConsumerWindowsMissingHeadIsNotPolicyProof(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	s.RecapBinding = legacyTestBinding
	ctx := context.Background()
	accepted, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "orphan-only", Cause: "edit", Text: "not selected"}); err != nil {
		t.Fatal(err)
	}
	root := s.Actmem.Root()
	if err = os.Remove(s.Actmem.HeadPath()); err != nil {
		t.Fatal(err)
	}
	if err = s.Actmem.Close(); err != nil {
		t.Fatal(err)
	}
	s.Actmem, err = actmem.Open(filepath.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	intent, seq, session, event, workspace, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := s.closeLegacyConsumer(ctx, accepted.IngestionID, intent, s.projectedPair(intent, seq, session, event, workspace))
	if err != nil || closed {
		t.Fatalf("orphan/zero view closed legacy consumer: %v %v", closed, err)
	}
	if current, err := s.ActivityProjection(ctx, accepted.IngestionID); err != nil || current.LegacyConsumer != nil || current.Status != "pending" || current.Revision != 0 || len(current.Entries) != 0 {
		t.Fatalf("unqualified absence became receipt: %+v %v", current, err)
	}
}
