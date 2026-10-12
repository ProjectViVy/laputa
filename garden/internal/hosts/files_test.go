package hosts

import (
	"context"
	"fmt"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"os"
	"strings"
	"testing"
)

func activeHost(t *testing.T) (*Service, string, string, FilePlan) {
	t.Helper()
	s, id, path := hostFixture(t)
	if err := s.registry.SetHostControls(context.Background(), id, registry.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	p, err := s.Preview(context.Background(), ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	return s, id, path, p
}
func TestManagedBlockPreservation(t *testing.T) {
	s, id, path, _ := activeHost(t)
	ctx := context.Background()
	original := "User instructions before\n"
	if err := os.WriteFile(path, []byte(original), 0640); err != nil {
		t.Fatal(err)
	}
	plan, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Apply(ctx, plan)
	if err == nil || status.ManualAction != "apply_projection" || status.WrittenHash != "" {
		t.Fatal("automatic existing-file write", status, err)
	}
	file, _ := os.ReadFile(path)
	if string(file) != original {
		t.Fatal("outside bytes overwritten")
	}
	state, err := s.state(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(state.BackupPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("backup not private", info, err)
	}
	before, _ := os.ReadFile(state.BackupPath)
	if string(before) != original {
		t.Fatal("backup lost original")
	}
	// The owner applies the reviewed bytes; Garden only observes exact intent.
	if err = os.WriteFile(path, []byte(plan.ProposedContent), 0640); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(ctx, id)
	if err != nil || status.WrittenHash != plan.ManagedHash || status.LoadedHash != "" || status.ManualAction != "" {
		t.Fatal(status, err)
	}
	edited := "Outside edit\n" + plan.ProposedContent + "After instructions\n"
	if err = os.WriteFile(path, []byte(edited), 0640); err != nil {
		t.Fatal(err)
	}
	status, err = s.Revoke(ctx, id)
	if err == nil || status.ManualAction != "remove_projection" {
		t.Fatal(status, err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != edited {
		t.Fatal("revoke lost outside edits")
	}
	b, _ := s.registry.Binding(ctx, id)
	if !b.Revoked {
		t.Fatal("credential live after revoke")
	}
	// Owner removes the block; a retry acknowledges without editing the file.
	cleaned := "Outside edit\n" + original + "After instructions\n"
	if err = os.WriteFile(path, []byte(cleaned), 0640); err != nil {
		t.Fatal(err)
	}
	status, err = s.Revoke(ctx, id)
	if err != nil || status.State != "revoked" || status.ManualAction != "" {
		t.Fatal(status, err)
	}
	after, _ = os.ReadFile(path)
	if string(after) != cleaned {
		t.Fatal("cleanup acknowledgment edited file")
	}
}

func TestManagedBlockConflict(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if err := os.WriteFile(path, []byte("external edit"), 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, p); err == nil {
		t.Fatal("unexpected whole-file change overwritten")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "external edit" {
		t.Fatal(string(raw))
	}
	marker := managedBlock(id, "one")
	if err := os.WriteFile(path, []byte(marker+marker), 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path}); err == nil {
		t.Fatal("duplicate markers accepted")
	}
}
func TestQueuedHostRevoke(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if err := s.registry.RevokeHostBinding(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, p); err == nil {
		t.Fatal("queued revoked write succeeded")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("revoked write created target", err)
	}
}
func TestRevokeUserEditPreserved(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, p); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	edited := strings.Replace(string(raw), "identity", "user edited managed block", 1)
	if err := os.WriteFile(path, []byte(edited), 0640); err != nil {
		t.Fatal(err)
	}
	st, err := s.Revoke(ctx, id)
	if err == nil || st.State != "conflict" {
		t.Fatal("unsafe cleanup did not conflict", st, err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != edited {
		t.Fatal("user edit lost")
	}
	b, _ := s.registry.Binding(ctx, id)
	if !b.Revoked {
		t.Fatal("cleanup conflict kept credential")
	}
}
func TestManualCleanupPreservesOriginalEmptyAndMissingChoices(t *testing.T) {
	for _, existed := range []bool{false, true} {
		t.Run(fmt.Sprintf("original-exists-%t", existed), func(t *testing.T) {
			s, id, path := hostFixture(t)
			ctx := context.Background()
			if err := s.registry.SetHostControls(ctx, id, registry.HostControls{PersonaSync: true}); err != nil {
				t.Fatal(err)
			}
			if existed {
				if err := os.WriteFile(path, nil, 0640); err != nil {
					t.Fatal(err)
				}
			}
			p, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
			if err != nil {
				t.Fatal(err)
			}
			status, err := s.Apply(ctx, p)
			if existed {
				if err == nil || status.ManualAction != "apply_projection" {
					t.Fatal(status, err)
				}
				if err = os.WriteFile(path, []byte(p.ProposedContent), 0640); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			status, err = s.Revoke(ctx, id)
			if err == nil || status.ManualAction != "remove_projection" {
				t.Fatal(status, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(before) {
				t.Fatal("revoke changed original state", err)
			}
			if existed {
				err = os.WriteFile(path, nil, 0640)
			} else {
				err = os.Remove(path)
			}
			if err != nil {
				t.Fatal(err)
			}
			status, err = s.Revoke(ctx, id)
			if err != nil || status.State != "revoked" {
				t.Fatal(status, err)
			}
			raw, err := os.ReadFile(path)
			if existed {
				if err != nil || len(raw) != 0 {
					t.Fatal("owner's empty choice lost", err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("owner's missing choice lost", err)
			}
		})
	}
}
