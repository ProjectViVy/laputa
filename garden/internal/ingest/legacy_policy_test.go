package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/laputa/actmem"
)

var legacyTestBinding = actmem.RecapBinding{SubjectID: "trusted-subject", DomainID: "whole-domain"}

func TestIssue58LegacyConsumerEffectAndPolicyStaySeparate(t *testing.T) {
	for _, stage := range []string{"before_effect", "archived", "missing_before", "malformed_before", "conflicting_archive", "partial_archive"} {
		t.Run(stage, func(t *testing.T) {
			s := openActivityIngest(t, t.TempDir())
			defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
			s.RecapBinding = legacyTestBinding
			ctx := context.Background()
			accepted, err := s.Submit(ctx, activityRequest())
			if err != nil {
				t.Fatal(err)
			}
			intent, seq, session, event, workspace, err := s.loadActivity(ctx, accepted.IngestionID)
			if err != nil {
				t.Fatal(err)
			}
			intent.Before, err = s.Actmem.AppendSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			// Revision zero is the actual first legacy snapshot, not a missing proof.
			if intent.Before.Revision != 0 || intent.Before.SHA256 == "" {
				t.Fatal("fixture lost actual first snapshot")
			}
			intent.Status, intent.Error = "unknown", "original uncertain legacy attempt"
			pair := s.projectedPair(intent, seq, session, event, workspace)
			var actualIDs []string
			if stage != "before_effect" {
				if stage == "partial_archive" {
					if _, err = s.Actmem.AppendEntry(pair[0]); err != nil {
						t.Fatal(err)
					}
				} else {
					actual, err := s.Actmem.AppendCaptured(pair, intent.Before)
					if err != nil {
						t.Fatal(err)
					}
					for _, e := range actual.Entries {
						actualIDs = append(actualIDs, e.ID)
					}
				}
				if _, err = s.Actmem.FoldSession(session); err != nil {
					t.Fatal(err)
				}
			}
			switch stage {
			case "missing_before":
				intent.Before = actmem.AppendSnapshot{}
			case "malformed_before":
				intent.Before.SHA256 = "not-a-native-snapshot"
			case "conflicting_archive":
				intent.Source.UserText += " changed payload"
			}
			if err = s.saveActivity(ctx, accepted.IngestionID, intent); err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.Exec(`UPDATE ingestions SET status='completed'`); err != nil {
				t.Fatal(err)
			}
			head, err := s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "不推进的C1"})
			if err != nil {
				t.Fatal(err)
			}
			before := legacyAuthorityBytes(t, s.Actmem.Root())
			if err = s.Start(); err != nil {
				t.Fatal(err)
			}
			got := awaitLegacyConsumer(t, s, accepted.IngestionID)
			if stage == "archived" {
				if got.Status != "applied" || got.Revision != intent.Before.Revision+1 || !reflect.DeepEqual(got.Entries, actualIDs) || got.LegacyConsumer != nil {
					t.Fatalf("real archived effect not rejoined: %+v", got)
				}
			} else {
				if got.LegacyConsumer == nil {
					t.Fatalf("consumer not closed: %+v", got)
				}
				got.LegacyConsumer = nil
				if !reflect.DeepEqual(got, intent) {
					t.Fatalf("closure rewrote unknown effect/evidence: before=%+v after=%+v", intent, got)
				}
			}
			if high, err := s.HighWatermark(ctx, ""); err != nil || high != seq {
				t.Fatalf("consumer still fences: %d %v", high, err)
			}
			if err = s.ArchiveSession(ctx, session); err != nil {
				t.Fatal(err)
			}
			if after := legacyAuthorityBytes(t, s.Actmem.Root()); !reflect.DeepEqual(before, after) {
				t.Fatal("consumer changed authority/archive bytes")
			}
			if current, err := s.Actmem.ReadRecap(ctx, legacyTestBinding); err != nil || current != head {
				t.Fatalf("consumer moved C1: %+v %v", current, err)
			}
		})
	}
}

func TestIssue58LegacyConsumerDoesNotHideArchiveIO(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	s.RecapBinding = legacyTestBinding
	ctx := context.Background()
	accepted, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	intent, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	intent.Before, err = s.Actmem.AppendSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	intent.Status = "unknown"
	if err = s.saveActivity(ctx, accepted.IngestionID, intent); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "valid C1"}); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(s.Actmem.CapsulesDir(), []byte("directory blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.projectActivity(ctx, accepted.IngestionID); err == nil {
		t.Fatal("archive IO failure was disguised as policy success")
	}
	got, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil || !reflect.DeepEqual(got, intent) {
		t.Fatalf("IO erased uncertain intent: %+v %v", got, err)
	}
}

func awaitLegacyConsumer(t *testing.T, s *Service, id string) activityIntent {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		got, _, _, _, _, err := s.loadActivity(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == "applied" || got.LegacyConsumer != nil {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker did not resolve consumer: %+v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func legacyAuthorityBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".md" {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[path] = string(raw)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestIssue58LegacyConsumerAppliedReceiptPrecedesCurrentPolicy(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	ctx := context.Background()
	accepted, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.projectActivity(ctx, accepted.IngestionID); err != nil {
		t.Fatal(err)
	}
	before, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil || before.Status != "applied" || len(before.Entries) != 2 {
		t.Fatalf("real receipt setup: %+v %v", before, err)
	}
	if err = s.Actmem.Close(); err != nil {
		t.Fatal(err)
	}
	s.RecapBinding = legacyTestBinding
	if err = s.projectActivity(ctx, accepted.IngestionID); err != nil {
		t.Fatal(err)
	}
	after, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("current owner hid real receipt: %+v %v", after, err)
	}
	projection, err := s.ActivityProjection(ctx, accepted.IngestionID)
	if err != nil || !reflect.DeepEqual(projection, before.ActivityProjection) {
		t.Fatal("receipt not readable")
	}
}

func TestIssue58LegacyConsumerRequiresPositiveCurrentPolicy(t *testing.T) {
	for _, invalid := range []string{"unbound", "prefix", "missing_history", "foreign", "head_io"} {
		t.Run(invalid, func(t *testing.T) {
			s := openActivityIngest(t, t.TempDir())
			defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
			s.RecapBinding = legacyTestBinding
			ctx := context.Background()
			accepted, err := s.Submit(ctx, activityRequest())
			if err != nil {
				t.Fatal(err)
			}
			head, err := s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "valid C1"})
			if err != nil {
				t.Fatal(err)
			}
			switch invalid {
			case "unbound":
				s.RecapBinding = actmem.RecapBinding{}
			case "prefix":
				err = os.WriteFile(s.Actmem.HeadPath(), []byte("---\nschema: actmem/v3\n---\ninvalid"), 0600)
			case "missing_history":
				err = os.Remove(filepath.Join(s.Actmem.Root(), "recap-history", fmt.Sprintf("%020d-%s.md", head.Revision, head.Digest)))
			case "foreign":
				s.RecapBinding.SubjectID = "foreign-subject"
			case "head_io":
				if err = os.Remove(s.Actmem.HeadPath()); err == nil {
					err = os.Mkdir(s.Actmem.HeadPath(), 0700)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			before, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.projectActivity(ctx, accepted.IngestionID); err == nil {
				t.Fatal("invalid owner authorized policy closure")
			}
			after, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("invalid owner changed effect: %+v %v", after, err)
			}
			if high, err := s.HighWatermark(ctx, ""); err != nil || high != 0 {
				t.Fatalf("unresolved activity escaped fence: %d %v", high, err)
			}
		})
	}
}

func TestIssue58LegacyConsumerClosurePersistenceFailure(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	s.RecapBinding = legacyTestBinding
	ctx := context.Background()
	accepted, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "unchanged"}); err != nil {
		t.Fatal(err)
	}
	before, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER deny_closure BEFORE UPDATE OF activity_json ON ingestions BEGIN SELECT RAISE(FAIL,'simulated durable metadata failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.projectActivity(ctx, accepted.IngestionID); err == nil {
		t.Fatal("failed persistence acknowledged consumer closure")
	}
	after, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed commit changed intent: %+v %v", after, err)
	}
	if high, err := s.HighWatermark(ctx, ""); err != nil || high != 0 {
		t.Fatalf("failed commit removed fence: %d %v", high, err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER deny_closure`); err != nil {
		t.Fatal(err)
	}
	if err = s.Start(); err != nil {
		t.Fatal(err)
	}
	if got := awaitLegacyConsumer(t, s, accepted.IngestionID); got.Status != before.Status || got.LegacyConsumer == nil {
		t.Fatalf("durable retry lost disposition: %+v", got)
	}
}

func TestIssue58LegacyConsumerMalformedDispositionNeverSuppresses(t *testing.T) {
	for _, invalid := range []string{"state", "digest", "revision", "binding", "unknown_field", "duplicate", "malformed"} {
		t.Run(invalid, func(t *testing.T) {
			s := openActivityIngest(t, t.TempDir())
			defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
			s.RecapBinding = legacyTestBinding
			ctx := context.Background()
			accepted, err := s.Submit(ctx, activityRequest())
			if err != nil {
				t.Fatal(err)
			}
			head, err := s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "current"})
			if err != nil {
				t.Fatal(err)
			}
			intent, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
			if err != nil {
				t.Fatal(err)
			}
			intent.LegacyConsumer = &LegacyConsumerDisposition{State: "policy_closed", Reason: "recap_v3", Binding: head.Binding, PolicyRevision: head.Revision, PolicyDigest: head.Digest}
			switch invalid {
			case "state":
				intent.LegacyConsumer.State = "unknown"
			case "digest":
				intent.LegacyConsumer.PolicyDigest = strings.Repeat("0", 64)
			case "revision":
				intent.LegacyConsumer.PolicyRevision++
			case "binding":
				intent.LegacyConsumer.Binding.SubjectID = "foreign"
			}
			raw, err := json.Marshal(intent)
			if err != nil {
				t.Fatal(err)
			}
			switch invalid {
			case "unknown_field":
				raw = append([]byte(`{"extra":true,`), raw[1:]...)
			case "duplicate":
				raw = append([]byte(`{"legacy_consumer":null,`), raw[1:]...)
			case "malformed":
				raw = []byte(`{"legacy_consumer":`)
			}
			if _, err = s.db.Exec(`UPDATE ingestions SET status='completed',activity_json=?`, string(raw)); err != nil {
				t.Fatal(err)
			}
			if err = s.Start(); err == nil {
				t.Fatal("invalid closure suppressed Start recovery")
			}
			if _, err = s.HighWatermark(ctx, ""); err == nil {
				t.Fatal("invalid closure removed watermark fence")
			}
			if err = s.ArchiveSession(ctx, "session"); err == nil {
				t.Fatal("invalid closure claimed archive success")
			}
		})
	}
}

func TestIssue58LegacyConsumerClosurePreservesRawMaterialReplay(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	s.RecapBinding = legacyTestBinding
	ctx := context.Background()
	accepted, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "raw material is not C1"}); err != nil {
		t.Fatal(err)
	}
	if err = s.projectActivity(ctx, accepted.IngestionID); err != nil {
		t.Fatal(err)
	}
	// status=accepted remains an independent Start branch even when activity is
	// already closed. The real catalog receives its existing canonical write.
	if err = s.Start(); err != nil {
		t.Fatal(err)
	}
	waitForCompleted(t, s, accepted.IngestionID)
	status, err := s.Get(ctx, accepted.IngestionID)
	if err != nil || len(status.MemoryIDs) != 1 {
		t.Fatalf("raw canonical replay suppressed: %+v %v", status, err)
	}
	projection, err := s.ActivityProjection(ctx, accepted.IngestionID)
	if err != nil || projection.Status != "pending" || projection.LegacyConsumer == nil || len(projection.Entries) != 0 {
		t.Fatalf("raw replay fabricated ACTMEM result: %+v %v", projection, err)
	}
}

func TestIssue58LegacyConsumerWatermarkKeepsEarlierGapAndConcurrentArchive(t *testing.T) {
	s := openActivityIngest(t, t.TempDir())
	defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
	s.RecapBinding = legacyTestBinding
	ctx := context.Background()
	first, err := s.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	req := activityRequest()
	req.EventID, req.SessionID = "later-event", "later-session"
	second, err := s.Submit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "v3", Cause: "edit", Text: "same owner"}); err != nil {
		t.Fatal(err)
	}
	if err = s.projectActivity(ctx, second.IngestionID); err != nil {
		t.Fatal(err)
	}
	if high, err := s.HighWatermark(ctx, ""); err != nil || high != 0 {
		t.Fatalf("later closure skipped earlier gap: %d %v", high, err)
	}
	if err = s.projectActivity(ctx, first.IngestionID); err != nil {
		t.Fatal(err)
	}
	if high, err := s.HighWatermark(ctx, ""); err != nil || high != second.Seq {
		t.Fatalf("closed prefix not admitted: %d %v", high, err)
	}
	before := legacyAuthorityBytes(t, s.Actmem.Root())
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if err := s.drainActivity(ctx); err != nil {
					t.Error(err)
					return
				}
				if err := s.ArchiveSession(ctx, "session"); err != nil {
					t.Error(err)
					return
				}
				if high, err := s.HighWatermark(ctx, ""); err != nil || high != second.Seq {
					t.Errorf("concurrent snapshot: %d %v", high, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if after := legacyAuthorityBytes(t, s.Actmem.Root()); !reflect.DeepEqual(before, after) {
		t.Fatal("concurrent archive changed C1")
	}
}
