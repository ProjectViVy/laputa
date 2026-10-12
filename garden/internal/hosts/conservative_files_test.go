package hosts

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"os"
	"path/filepath"
	"testing"
)

func TestConservativeRevokeNeverRewritesHostFile(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, p); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Revoke(ctx, id)
	if err == nil || status.State != "conflict" {
		t.Fatal("revocation silently rewrote external file", status, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("revoke changed host bytes", string(after), err)
	}
	binding, err := s.registry.Binding(ctx, id)
	if err != nil || !binding.Revoked {
		t.Fatal("manual cleanup left authority active", binding, err)
	}
}

func TestConservativeExistingFileRequiresManualApply(t *testing.T) {
	s, id, path := hostFixture(t)
	ctx := context.Background()
	if err := s.registry.SetHostControls(ctx, id, registry.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	original := "user-owned instructions\n"
	if err := os.WriteFile(path, []byte(original), 0640); err != nil {
		t.Fatal(err)
	}
	plan, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Apply(ctx, plan)
	if err == nil || status.State != "conflict" {
		t.Fatal("existing external file silently replaced", status, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != original {
		t.Fatal("apply changed user bytes", string(after), err)
	}
}

func TestConservativeCreateDoesNotClobberExistingInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.md")
	if err := os.WriteFile(path, []byte("EDITOR_SAVE"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := createFile(path, "Garden projection"); err == nil {
		t.Fatal("create replaced competing target")
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "EDITOR_SAVE" {
		t.Fatal("competing save lost", string(raw), err)
	}
}
