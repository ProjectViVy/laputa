package ingest

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/actmem"
)

// Positive archive proof supplies entries, not the original head revision.
// A missing/null revision must not decode into a usable first-append snapshot.
func TestIssue58LegacySnapshotRequiresOriginalRevision(t *testing.T) {
	for _, state := range []string{"unknown", "applying"} {
		for _, form := range []string{"explicit_zero", "explicit_nonzero", "missing_revision", "null_revision", "missing_before", "null_before"} {
			t.Run(state+"/"+form, func(t *testing.T) {
				ctx := context.Background()
				s := openActivityIngest(t, t.TempDir())
				defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
				s.RecapBinding = legacyTestBinding
				accepted, err := s.Submit(ctx, activityRequest())
				if err != nil {
					t.Fatal(err)
				}
				intent, seq, session, event, workspace, err := s.loadActivity(ctx, accepted.IngestionID)
				if err != nil {
					t.Fatal(err)
				}
				pair := s.projectedPair(intent, seq, session, event, workspace)
				if form != "explicit_zero" {
					seed := append(pair[:0:0], pair...)
					for i := range seed {
						seed[i].EventID, seed[i].SessionID = "earlier-event", "earlier-session"
					}
					snapshot, err := s.Actmem.AppendSnapshot()
					if err != nil {
						t.Fatal(err)
					}
					if _, err = s.Actmem.AppendCaptured(seed, snapshot); err != nil {
						t.Fatal(err)
					}
					if _, err = s.Actmem.FoldSession("earlier-session"); err != nil {
						t.Fatal(err)
					}
				}
				intent.Before, err = s.Actmem.AppendSnapshot()
				if err != nil {
					t.Fatal(err)
				}
				actual, err := s.Actmem.AppendCaptured(pair, intent.Before)
				if err != nil {
					t.Fatal(err)
				}
				if actual.Revision != intent.Before.Revision+1 {
					t.Fatal("fixture lost original native revision")
				}
				if _, err = s.Actmem.FoldSession(session); err != nil {
					t.Fatal(err)
				}
				intent.Status, intent.Error = state, "retained uncertain native outcome"
				if err = s.saveActivity(ctx, accepted.IngestionID, intent); err != nil {
					t.Fatal(err)
				}
				raw := legacyActivityJSON(t, s, accepted.IngestionID)
				complete := form == "explicit_zero" || form == "explicit_nonzero"
				if !complete {
					var fields map[string]json.RawMessage
					if err = json.Unmarshal([]byte(raw), &fields); err != nil {
						t.Fatal(err)
					}
					if form == "missing_before" {
						delete(fields, "before")
					} else if form == "null_before" {
						fields["before"] = json.RawMessage("null")
					} else {
						var before map[string]json.RawMessage
						if err = json.Unmarshal(fields["before"], &before); err != nil {
							t.Fatal(err)
						}
						if form == "missing_revision" {
							delete(before, "revision")
						} else {
							before["revision"] = json.RawMessage("null")
						}
						fields["before"], err = json.Marshal(before)
						if err != nil {
							t.Fatal(err)
						}
					}
					encoded, err := json.MarshalIndent(fields, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					raw = string(encoded) + "\n"
				}
				if _, err = s.db.Exec(`UPDATE ingestions SET status='completed',activity_json=? WHERE ingestion_id=?`, raw, accepted.IngestionID); err != nil {
					t.Fatal(err)
				}
				head, err := s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "metadata-v3", Cause: "edit", Text: "unchanged C1"})
				if err != nil {
					t.Fatal(err)
				}
				authority := legacyAuthorityBytes(t, s.Actmem.Root())
				if complete {
					if err = s.projectActivity(ctx, accepted.IngestionID); err != nil {
						t.Fatal(err)
					}
					got, _, _, _, _, err := s.loadActivity(ctx, accepted.IngestionID)
					if err != nil {
						t.Fatal(err)
					}
					ids := []string{actual.Entries[0].ID, actual.Entries[1].ID}
					if got.Status != "applied" || got.Revision != actual.Revision || !reflect.DeepEqual(got.Entries, ids) {
						t.Fatalf("actual positive receipt lost: %+v", got)
					}
				} else {
					assertLegacyMetadataRejected(t, s, accepted.IngestionID, session, raw)
				}
				if current, err := s.Actmem.ReadRecap(ctx, legacyTestBinding); err != nil || current != head {
					t.Fatalf("C1 moved: %+v %v", current, err)
				}
				if !reflect.DeepEqual(authority, legacyAuthorityBytes(t, s.Actmem.Root())) {
					t.Fatal("authority/archive bytes changed")
				}
			})
		}
	}
}

func TestIssue58LegacyClosureRequiresCompleteJSONDocument(t *testing.T) {
	for _, corruption := range []string{"}", "]", " {}", " null", "duplicate", "unknown"} {
		t.Run(corruption, func(t *testing.T) {
			ctx := context.Background()
			s := openActivityIngest(t, t.TempDir())
			defer func() { _ = s.Close(); _ = s.Actmem.Close() }()
			s.RecapBinding = legacyTestBinding
			accepted, err := s.Submit(ctx, activityRequest())
			if err != nil {
				t.Fatal(err)
			}
			head, err := s.Actmem.CommitRecap(ctx, actmem.RecapCommit{Binding: legacyTestBinding, OperationID: "metadata-v3", Cause: "edit", Text: "unchanged C1"})
			if err != nil {
				t.Fatal(err)
			}
			if err = s.projectActivity(ctx, accepted.IngestionID); err != nil {
				t.Fatal(err)
			}
			raw := legacyActivityJSON(t, s, accepted.IngestionID)
			switch corruption {
			case "duplicate":
				raw = `{"schema":"duplicate",` + raw[1:]
			case "unknown":
				raw = `{"unexpected":true,` + raw[1:]
			default:
				raw += corruption
			}
			if _, err = s.db.Exec(`UPDATE ingestions SET status='completed',activity_json=? WHERE ingestion_id=?`, raw, accepted.IngestionID); err != nil {
				t.Fatal(err)
			}
			authority := legacyAuthorityBytes(t, s.Actmem.Root())
			assertLegacyMetadataRejected(t, s, accepted.IngestionID, activityRequest().SessionID, raw)
			if current, err := s.Actmem.ReadRecap(ctx, legacyTestBinding); err != nil || current != head {
				t.Fatalf("C1 moved: %+v %v", current, err)
			}
			if !reflect.DeepEqual(authority, legacyAuthorityBytes(t, s.Actmem.Root())) {
				t.Fatal("authority/archive bytes changed")
			}
		})
	}
}

func legacyActivityJSON(t *testing.T, s *Service, id string) string {
	t.Helper()
	var raw string
	if err := s.db.QueryRow(`SELECT activity_json FROM ingestions WHERE ingestion_id=?`, id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertLegacyMetadataRejected(t *testing.T, s *Service, id, session, raw string) {
	t.Helper()
	ctx := context.Background()
	checks := []struct {
		name string
		call func() error
	}{
		{"projection", func() error { _, err := s.ActivityProjection(ctx, id); return err }},
		{"project", func() error { return s.projectActivity(ctx, id) }},
		{"watermark", func() error { _, err := s.HighWatermark(ctx, ""); return err }},
		{"archive", func() error { return s.ArchiveSession(ctx, session) }},
		{"drain", func() error { return s.drainActivity(ctx) }},
		{"start", s.Start},
	}
	for _, check := range checks {
		if err := check.call(); err == nil {
			t.Errorf("%s accepted incomplete/malformed activity metadata", check.name)
		}
		if got := legacyActivityJSON(t, s, id); got != raw {
			t.Fatalf("%s rewrote original JSON/source/error/Before:\nbefore=%s\nafter=%s", check.name, raw, got)
		}
	}
}
