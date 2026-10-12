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
	if err := os.WriteFile(path, []byte("User instructions before\n"), 0640); err != nil {
		t.Fatal(err)
	}
	p, err := s.Preview(context.Background(), ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	return s, id, path, p
}
func TestManagedBlockPreservation(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	status, err := s.Apply(ctx, p)
	if err != nil || status.WrittenHash == "" || status.LoadedHash != "" || status.State != "pending_reload" {
		t.Fatal(status, err)
	}
	file, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(file), "User instructions before\n") {
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
	if string(before) != "User instructions before\n" {
		t.Fatal("backup lost original")
	}
	if err = os.WriteFile(path, append([]byte("Outside edit\n"), append(file, []byte("After instructions\n")...)...), 0640); err != nil {
		t.Fatal(err)
	}
	_, err = s.Revoke(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != "Outside edit\nUser instructions before\nAfter instructions\n" {
		t.Fatal("revoke lost outside edits", string(after))
	}
	b, _ := s.registry.Binding(ctx, id)
	if !b.Revoked {
		t.Fatal("credential live after revoke")
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
	raw, _ := os.ReadFile(path)
	if string(raw) != "User instructions before\n" {
		t.Fatal("revoked write touched disk")
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
func TestManagedFileMissingRestoredOnRevoke(t *testing.T) {
	for _, existed := range []bool{false, true} {
		t.Run(fmt.Sprintf("original-exists-%t", existed), func(t *testing.T) {
			s, id, path := hostFixture(t)
			ctx := context.Background()
			s.registry.SetHostControls(ctx, id, registry.HostControls{PersonaSync: true})
			if existed {
				if err := os.WriteFile(path, nil, 0640); err != nil {
					t.Fatal(err)
				}
			}
			p, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Apply(ctx, p); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Revoke(ctx, id); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if existed {
				if err != nil || len(raw) != 0 {
					t.Fatal("original empty state lost", raw, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("original missing state lost", err)
			}
		})
	}
}
