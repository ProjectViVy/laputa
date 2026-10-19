package actmem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The host owns this envelope. Whitespace, version and evidence fields remain
// opaque owner data, including versions this owner cannot interpret.
const issue58ReconcileReceipt = " {\"v\":99,\"dispositions\":\"已更正\",\"correction_evidence_digest\":\"opaque\"}\r\n"

func TestIssue58RecapReconcile(t *testing.T) {
	for _, coverage := range []uint64{0, 3} {
		t.Run(fmt.Sprint(coverage), func(t *testing.T) {
			s := issue58Store(t)
			base := RecapHead{Binding: issue58Binding}
			if coverage != 0 {
				base = issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "plain-summary", Cause: "summary", Text: "历史原文", Sources: SourceRange{Through: 3, Digest: "historical-plain-receipt"}})
				base = issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: base.Revision, OperationID: "clear", Cause: "clear"})
			}
			before := issue58Files(t, s.Root())
			text := "  用户纠正：日期是十八日。\r\n\r\n## 自由标题\r\n---\r\n```\r\n承诺原文\r\n```\r\n\r\n"
			request := RecapCommit{Binding: issue58Binding, BaseRevision: base.Revision, OperationID: "reconcile", Cause: "reconcile", Text: text, Sources: SourceRange{After: coverage, Through: coverage, Digest: issue58ReconcileReceipt}}
			h := issue58Commit(t, s, request)
			if h.Text != text || h.Sources != request.Sources || h.CoveredThrough != coverage || h.InvalidationEpoch != base.InvalidationEpoch || h.Revision != base.Revision+1 || h.RestoredFrom != 0 {
				t.Fatalf("reconciliation lost exact text/provenance or changed coverage/epoch: %+v", h)
			}
			// Independently hash the public request, including the ENTIRE receipt.
			preimage, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(preimage)
			if h.OperationDigest != hex.EncodeToString(sum[:]) {
				t.Fatal("request digest omitted opaque receipt data")
			}
			after := issue58Files(t, s.Root())
			for path, raw := range before {
				if path != "ACTMEM.MD" && after[path] != raw {
					t.Fatalf("rewrote historical receipt: %s", path)
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(filepath.Dir(s.Root()))
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			got, err := reopened.ReadRecap(context.Background(), issue58Binding)
			if err != nil || got != h {
				t.Fatalf("reopen changed exact receipt: %+v %v", got, err)
			}
			history, err := reopened.ListRecapHistory(context.Background(), issue58Binding, 0, 20)
			if err != nil || len(history) != int(h.Revision) || history[0] != h {
				t.Fatalf("reachable reconcile history: %+v %v", history, err)
			}
		})
	}
}

func TestIssue58RecapReconcileValidationReplay(t *testing.T) {
	s := issue58Store(t)
	base := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "summary", Cause: "summary", Text: "原文", Sources: SourceRange{Through: 3, Digest: "plain"}})
	request := RecapCommit{Binding: issue58Binding, BaseRevision: base.Revision, OperationID: "reconcile", Cause: "reconcile", Text: "修订原文\r\n", Sources: SourceRange{After: 3, Through: 3, Digest: issue58ReconcileReceipt}}
	before := issue58Files(t, s.Root())
	for name, mutate := range map[string]func(*RecapCommit){
		"wrong_after":           func(r *RecapCommit) { r.Sources.After = 2 },
		"wrong_through":         func(r *RecapCommit) { r.Sources.Through = 4 },
		"wrong_equal_endpoints": func(r *RecapCommit) { r.Sources.After = 2; r.Sources.Through = 2 },
		"empty_receipt":         func(r *RecapCommit) { r.Sources.Digest = "" },
		"blank_receipt":         func(r *RecapCommit) { r.Sources.Digest = " \r\n" },
		"restore_metadata":      func(r *RecapCommit) { r.RestoredFrom = 1 },
		"empty_summary_range":   func(r *RecapCommit) { r.Cause = "summary" },
		"unknown_cause":         func(r *RecapCommit) { r.Cause = "future" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := request
			mutate(&bad)
			if h, err := s.CommitRecap(context.Background(), bad); CodeOf(err) != "recap_invalid_range" || h != (RecapHead{}) {
				t.Fatalf("invalid request accepted: %+v %v", h, err)
			}
			if !reflect.DeepEqual(before, issue58Files(t, s.Root())) {
				t.Fatal("invalid request changed files")
			}
		})
	}
	receipt := issue58Commit(t, s, request)
	current := receipt
	for _, cause := range []string{"edit", "restore", "clear"} {
		if cause != "edit" {
			current = issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: current.Revision, OperationID: "certificate-before-" + cause, Cause: "reconcile", Text: "更正原文", Sources: request.Sources})
		}
		next := RecapCommit{Binding: issue58Binding, BaseRevision: current.Revision, OperationID: cause, Cause: cause}
		if cause == "edit" {
			next.Text = "手工编辑"
		}
		if cause == "restore" {
			next.RestoredFrom = 1
		}
		bad := next
		bad.Sources = request.Sources
		if _, err := s.CommitRecap(context.Background(), bad); CodeOf(err) != "recap_invalid_range" {
			t.Fatalf("%s accepted certificate: %v", cause, err)
		}
		current = issue58Commit(t, s, next)
		if current.Sources != (SourceRange{}) || current.CoveredThrough != 3 {
			t.Fatalf("%s retained certificate or changed coverage: %+v", cause, current)
		}
		if cause == "restore" && current.Text != base.Text {
			t.Fatal("restore trusted caller text")
		}
		got, err := s.CommitRecap(context.Background(), request)
		if err != nil || got != receipt {
			t.Fatalf("%s lost original replay receipt: %+v %v", cause, got, err)
		}
	}
	if current.Text != "" || current.InvalidationEpoch != 1 {
		t.Fatal("clear semantics changed")
	}
	for _, mutate := range []func(*RecapCommit){
		func(r *RecapCommit) { r.Text += "changed" },
		func(r *RecapCommit) { r.Sources.Digest += " " },
		func(r *RecapCommit) {
			r.Sources.Digest = " {\"v\":99,\"dispositions\":\"已更正\",\"correction_evidence_digest\":\"different\"}\r\n"
		},
	} {
		changed := request
		mutate(&changed)
		if _, err := s.CommitRecap(context.Background(), changed); CodeOf(err) != "recap_operation_conflict" {
			t.Fatalf("changed request replayed: %v", err)
		}
	}
	stale := request
	stale.OperationID = "stale-reconcile"
	if _, err := s.CommitRecap(context.Background(), stale); CodeOf(err) != "recap_conflict" {
		t.Fatalf("stale reconcile overwrote clear: %v", err)
	}
	got, err := s.ReadRecap(context.Background(), issue58Binding)
	if err != nil || got != current {
		t.Fatalf("replay/conflict altered clear: %+v %v", got, err)
	}
}

// Recompute integrity hashes so malformed metadata is rejected for semantics,
// rather than merely failing a stale checksum. Unknown causes must fail closed.
func TestIssue58RecapReconcileValidators(t *testing.T) {
	s := issue58Store(t)
	previous := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "summary", Cause: "summary", Sources: SourceRange{Through: 3, Digest: "plain"}})
	valid := issue58Commit(t, s, RecapCommit{Binding: issue58Binding, BaseRevision: 1, OperationID: "reconcile", Cause: "reconcile", Text: "更正", Sources: SourceRange{After: 3, Through: 3, Digest: issue58ReconcileReceipt}})
	for name, mutate := range map[string]func(*RecapHead){
		"after":    func(h *RecapHead) { h.Sources.After = 2 },
		"through":  func(h *RecapHead) { h.Sources.Through = 4 },
		"receipt":  func(h *RecapHead) { h.Sources.Digest = "" },
		"restore":  func(h *RecapHead) { h.RestoredFrom = 1 },
		"coverage": func(h *RecapHead) { h.CoveredThrough = 4 },
		"unknown":  func(h *RecapHead) { h.Cause = "future" },
	} {
		t.Run("format_"+name, func(t *testing.T) {
			bad := valid
			mutate(&bad)
			bad.Digest = recapHeadDigest(bad)
			raw, err := encodeRecap(bad)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := decodeRecap(raw); CodeOf(err) != "recap_unavailable" || got != (RecapHead{}) {
				t.Fatalf("malformed reconcile decoded: %+v %v", got, err)
			}
		})
	}
	for name, mutate := range map[string]func(*RecapHead){
		"different_current_coverage": func(h *RecapHead) { h.CoveredThrough = 4; h.Sources.After = 4; h.Sources.Through = 4 },
		"wrong_epoch":                func(h *RecapHead) { h.InvalidationEpoch = 1 },
		"wrong_endpoints":            func(h *RecapHead) { h.Sources.After = 2; h.Sources.Through = 2 },
	} {
		t.Run("chain_"+name, func(t *testing.T) {
			bad := valid
			mutate(&bad)
			bad.OperationDigest = recapHash(RecapCommit{Binding: bad.Binding, BaseRevision: previous.Revision, OperationID: bad.OperationID, Cause: bad.Cause, Text: bad.Text, Sources: bad.Sources})
			bad.Digest = recapHeadDigest(bad)
			if err := validateRecapTransition(previous, bad); CodeOf(err) != "recap_unavailable" {
				t.Fatalf("invalid reconcile transition accepted: %v", err)
			}
		})
	}
}

func TestIssue58RecapReconcileRecovery(t *testing.T) {
	for _, coverage := range []uint64{0, 3} {
		for _, boundary := range []string{"before_replace", "cancel_before_replace", "after_replace", "before_directory_sync", "after_directory_sync", "lost_response"} {
			t.Run(fmt.Sprintf("%d/%s", coverage, boundary), func(t *testing.T) {
				dir := t.TempDir()
				s, err := Open(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				base := RecapHead{Binding: issue58Binding}
				if coverage != 0 {
					base = issue58Commit(t, s, RecapCommit{Binding: issue58Binding, OperationID: "summary", Cause: "summary", Text: "原文", Sources: SourceRange{Through: 3, Digest: "plain"}})
				}
				request := RecapCommit{Binding: issue58Binding, BaseRevision: base.Revision, OperationID: "reconcile", Cause: "reconcile", Text: "修订原文\r\n", Sources: SourceRange{After: coverage, Through: coverage, Digest: issue58ReconcileReceipt}}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				s.recapFault = func(point string) error {
					if boundary == "cancel_before_replace" && point == "before_replace" {
						cancel()
						return nil
					}
					if point == boundary {
						return errors.New("injected reconcile publication failure")
					}
					return nil
				}
				returned, err := s.CommitRecap(ctx, request)
				published := boundary != "before_replace" && boundary != "cancel_before_replace"
				if boundary == "lost_response" {
					if err != nil {
						t.Fatal(err)
					}
				} else if boundary == "cancel_before_replace" {
					if !errors.Is(err, context.Canceled) || returned != (RecapHead{}) {
						t.Fatalf("cancellation: %+v %v", returned, err)
					}
				} else {
					want := "recap_unavailable"
					if published {
						want = "recap_commit_unknown"
					}
					if CodeOf(err) != want || returned != (RecapHead{}) {
						t.Fatalf("publication boundary: %+v %v", returned, err)
					}
				}
				s.recapFault = nil
				var visible RecapHead
				if published {
					raw, err := os.ReadFile(s.HeadPath())
					if err != nil {
						t.Fatal(err)
					}
					visible, err = decodeRecap(raw)
					if err != nil {
						t.Fatal(err)
					}
					if boundary != "lost_response" {
						issue58AssertUnknown(t, s, request)
					}
				} else {
					got, err := s.ReadRecap(context.Background(), issue58Binding)
					if err != nil || got != base {
						t.Fatalf("pre-replace changed current view: %+v %v", got, err)
					}
					got, found, err := s.LookupRecapOperation(context.Background(), issue58Binding, request.OperationID)
					if coverage == 0 && !recapSupportsAbsenceConfirmation {
						if CodeOf(err) != "recap_commit_unknown" || found || got != (RecapHead{}) {
							t.Fatalf("missing head became unsupported absence proof: %+v %v %v", got, found, err)
						}
					} else if err != nil || found || got != (RecapHead{}) {
						t.Fatalf("orphan became reachable receipt: %+v %v %v", got, found, err)
					}
				}
				files := issue58Files(t, s.Root())
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				if published {
					// Continuing recovery failures may never turn visible data into a receipt.
					for _, point := range []string{"recovery_before_replace", "recovery_after_replace", "recovery_before_directory_sync", "recovery_after_directory_sync"} {
						failed, err := Open(dir)
						if err != nil {
							t.Fatal(err)
						}
						failed.recapFault = func(name string) error {
							if name == point {
								return errors.New("recovery remains unconfirmed")
							}
							return nil
						}
						issue58AssertUnknown(t, failed, request)
						failed.recapFault = nil
						issue58AssertUnknown(t, failed, request)
						if err := failed.Close(); err != nil {
							t.Fatal(err)
						}
					}
				}
				reopened, err := Open(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				if published {
					got, err := reopened.CommitRecap(context.Background(), request)
					if err != nil || got != visible {
						t.Fatalf("recovery changed positive receipt: %+v %v", got, err)
					}
					if !reflect.DeepEqual(files, issue58Files(t, reopened.Root())) {
						t.Fatal("recovery rewrote receipt/history bytes")
					}
				} else {
					// Retry publishes the requested operation; it does not promote its orphan.
					got := issue58Commit(t, reopened, request)
					if got.Revision != base.Revision+1 || got.CoveredThrough != coverage || got.Sources != request.Sources {
						t.Fatalf("retry altered reconcile: %+v", got)
					}
				}
				history, err := reopened.ListRecapHistory(context.Background(), issue58Binding, 0, 20)
				if err != nil || len(history) != int(base.Revision+1) {
					t.Fatalf("orphan promoted or reachable history lost: %+v %v", history, err)
				}
			})
		}
	}
}
