package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/dashimaki/laputa/evolution"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dashimaki/laputa/actmem"
	"github.com/dashimaki/mentle/facade"
)

func openActivityIngest(t *testing.T, dir string) *Service {
	t.Helper()
	catalog, err := facade.OpenCatalogService(context.Background(), filepath.Join(dir, "palace"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.Close() })
	svc, err := OpenPaused(filepath.Join(dir, "state.db"), catalog)
	if err != nil {
		t.Fatal(err)
	}
	svc.ProfileID = "profile"
	svc.Actmem = actmem.New(filepath.Join(dir, "persona"))
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func activityRequest() SubmitRequest {
	content := "durable real user source"
	hash := sha256.Sum256([]byte(content))
	return SubmitRequest{SessionID: "session", EventID: "host:run:7", Phase: "session_end", Content: content, ContentHash: fmt.Sprintf("sha256:%x", hash), Activity: &CaptureActivity{Phase: "completed", UserText: "synthetic user-only fact"}}
}

func awaitActivityProjection(t *testing.T, svc *Service, id string) ActivityProjection {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		projection, err := svc.ActivityProjection(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if projection.Status == "applied" {
			return projection
		}
		if time.Now().After(deadline) {
			t.Fatalf("activity projection=%+v", projection)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCaptureActivityRawFirstAndExactReceiptReplay(t *testing.T) {
	svc := openActivityIngest(t, t.TempDir())
	ctx := context.Background()
	req := activityRequest()
	accepted, err := svc.Submit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	high, err := svc.HighWatermark(ctx, "")
	if err != nil || high != 0 {
		t.Fatalf("pending activity crossed watermark: %d %v", high, err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	first := awaitActivityProjection(t, svc, accepted.IngestionID)
	if len(first.Entries) != 2 || first.Revision != 1 {
		t.Fatalf("real native projection=%+v", first)
	}
	again, err := svc.Submit(ctx, req)
	if err != nil || again.IngestionID != accepted.IngestionID || again.Seq != accepted.Seq {
		t.Fatalf("original source replay: %+v %v", again, err)
	}
	repeated, err := svc.ActivityProjection(ctx, accepted.IngestionID)
	if err != nil || !reflect.DeepEqual(first, repeated) {
		t.Fatalf("original activity receipt changed: %+v %v", repeated, err)
	}
	high, err = svc.HighWatermark(ctx, "")
	if err != nil || high != accepted.Seq {
		t.Fatalf("applied activity not admitted: %d %v", high, err)
	}
	if _, err := svc.Actmem.FoldSession(req.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(ctx, req); err != nil {
		t.Fatal(err)
	}
	head, err := svc.Actmem.Read()
	if err != nil || len(head.Entries()) != 0 {
		t.Fatalf("replay resurrected archived activity: %+v %v", head, err)
	}
}

func TestCaptureActivityKeepsPreviouslyAcceptedRowsUnchanged(t *testing.T) {
	svc := openActivityIngest(t, t.TempDir())
	ctx := context.Background()
	req := activityRequest()
	req.Activity = nil
	original, err := svc.Submit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	req.Activity = activityRequest().Activity
	replay, err := svc.Submit(ctx, req)
	if err != nil || replay.IngestionID != original.IngestionID {
		t.Fatalf("legacy receipt changed: %+v %v", replay, err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	projection, err := svc.ActivityProjection(ctx, original.IngestionID)
	if err != nil || projection.Status != "not_requested" {
		t.Fatalf("old accepted input backfilled: %+v %v", projection, err)
	}
}

func TestCaptureActivityRetryWithoutNewQueueDelivery(t *testing.T) {
	dir := t.TempDir()
	svc := openActivityIngest(t, dir)
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	svc.Actmem = actmem.New(blocked)
	accepted, err := svc.Submit(context.Background(), activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		status, err := svc.Get(context.Background(), accepted.IngestionID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("raw source did not complete: %+v", status)
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	awaitActivityProjection(t, svc, accepted.IngestionID)
}

func TestCaptureActivityOriginalIntentRecovery(t *testing.T) {
	for _, stage := range []string{"before_effect", "after_effect", "after_archive", "changed_head"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			svc := openActivityIngest(t, dir)
			ctx := context.Background()
			accepted, err := svc.Submit(ctx, activityRequest())
			if err != nil {
				t.Fatal(err)
			}
			intent, seq, session, event, workspace, err := svc.loadActivity(ctx, accepted.IngestionID)
			if err != nil {
				t.Fatal(err)
			}
			intent.Before, err = svc.Actmem.AppendSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			intent.Status = "applying"
			if err := svc.saveActivity(ctx, accepted.IngestionID, intent); err != nil {
				t.Fatal(err)
			}
			pair := svc.projectedPair(intent, seq, session, event, workspace)
			var original []string
			if stage == "after_effect" || stage == "after_archive" {
				result, err := svc.Actmem.AppendCaptured(pair, intent.Before)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range result.Entries {
					original = append(original, entry.ID)
				}
				if stage == "after_archive" {
					if _, err := svc.Actmem.FoldSession(session); err != nil {
						t.Fatal(err)
					}
				}
			}
			if stage == "changed_head" {
				other := append([]evolution.Entry(nil), pair...)
				for i := range other {
					other[i].EventID = "other-event"
				}
				if _, err := svc.Actmem.AppendCaptured(other, intent.Before); err != nil {
					t.Fatal(err)
				}
			}
			if err := svc.Close(); err != nil {
				t.Fatal(err)
			}
			reopened := openActivityIngest(t, dir)
			if err := reopened.Start(); err != nil {
				t.Fatal(err)
			}
			if stage == "changed_head" {
				deadline := time.Now().Add(time.Second)
				for {
					projection, err := reopened.ActivityProjection(ctx, accepted.IngestionID)
					if err != nil {
						t.Fatal(err)
					}
					if projection.Status == "unknown" {
						break
					}
					if time.Now().After(deadline) {
						t.Fatalf("missing unknown fence: %+v", projection)
					}
					time.Sleep(time.Millisecond)
				}
				high, err := reopened.HighWatermark(ctx, "")
				if err != nil || high != 0 {
					t.Fatalf("unresolved crossed watermark: %d %v", high, err)
				}
				head, err := reopened.Actmem.Read()
				if err != nil || len(head.Entries()) != 2 {
					t.Fatalf("unknown operation reappended: %+v %v", head, err)
				}
				return
			}
			result := awaitActivityProjection(t, reopened, accepted.IngestionID)
			if original != nil && !reflect.DeepEqual(result.Entries, original) {
				t.Fatalf("recovery replaced original entries: %v -> %v", original, result.Entries)
			}
			head, err := reopened.Actmem.Read()
			if err != nil {
				t.Fatal(err)
			}
			expected := 2
			if stage == "after_archive" {
				expected = 0
			}
			if len(head.Entries()) != expected {
				t.Fatalf("recovery duplicated/resurrected entries: %d", len(head.Entries()))
			}
		})
	}
}

func TestCaptureActivityArchiveWaitsForAcceptedProjection(t *testing.T) {
	svc := openActivityIngest(t, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	accepted, err := svc.Submit(ctx, activityRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveSession(ctx, "session"); err != nil {
		t.Fatal(err)
	}
	projection, err := svc.ActivityProjection(ctx, accepted.IngestionID)
	if err != nil || projection.Status != "applied" || len(projection.Entries) != 2 {
		t.Fatalf("archive crossed incomplete original capture: %+v %v", projection, err)
	}
	head, err := svc.Actmem.Read()
	if err != nil || len(head.Entries()) != 0 {
		t.Fatalf("archive raced behind source writer: %+v %v", head, err)
	}
	capsules, err := svc.Actmem.ListCapsules()
	if err != nil || len(capsules) == 0 {
		t.Fatalf("no actual archive: %v %v", capsules, err)
	}
}
