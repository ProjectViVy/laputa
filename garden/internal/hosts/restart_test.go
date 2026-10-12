package hosts

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"os"
	"path/filepath"
	"testing"
)

func reopenHost(t *testing.T, s *Service) *Service {
	t.Helper()
	s.Close()
	if err := s.registry.Close(); err != nil {
		t.Fatal(err)
	}
	reg, err := registry.Open(filepath.Join(filepath.Dir(s.backupDir), "garden.db"), filepath.Dir(s.backupDir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reg.Close() })
	return New(reg, s.backupDir, s.persona, s.protected...)
}
func TestHostRestartIntentRecovery(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	v, err := s.state(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	v.Path = path
	v.State = "unknown"
	v.OriginalExists = true
	v.PlannedHash = p.ManagedHash
	v.PlannedRevisions = p.SourceRevisions
	if err = s.save(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(p.ProposedContent), 0640); err != nil {
		t.Fatal(err)
	}
	s = reopenHost(t, s)
	next, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal("observed exact intent treated as user edit", err)
	}
	if next.OriginalHash != hash(p.ProposedContent) {
		t.Fatal(next)
	}
	status, err := s.Status(ctx, id)
	if err != nil || status.WrittenHash != p.ManagedHash || status.State != "pending_reload" || status.LoadedHash != "" {
		t.Fatal(status, err)
	}
}
func TestHostSavedWrittenLoadedHeadsAndStalePreview(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	status, err := s.Apply(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := s.persona(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := authority.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = authority.SaveUserDocument(persona.KindIdentity, "new saved identity", doc.Revision, "user", persona.SourceUserDirect, "edit"); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(ctx, id)
	if err != nil || status.SavedRevisions["identity"] != 2 || status.WrittenRevisions["identity"] != 1 || status.LoadedHash != "" {
		t.Fatal(status, err)
	}
	next, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	doc, err = authority.GetDocument(persona.KindIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = authority.SaveUserDocument(persona.KindIdentity, "newer identity", doc.Revision, "user", persona.SourceUserDirect, "edit"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if _, err = s.Apply(ctx, next); err == nil {
		t.Fatal("stale source preview applied")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("stale preview touched file")
	}
	s = reopenHost(t, s)
	status, err = s.Status(ctx, id)
	if err != nil || status.SavedRevisions["identity"] != 3 || status.WrittenRevisions["identity"] != 1 {
		t.Fatal("restart heads conflated", status, err)
	}
}
func TestHostDisabledAndSymlinkTargets(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if err := s.registry.SetHostControls(ctx, id, registry.HostControls{Recall: true, Capture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, p); err == nil {
		t.Fatal("disabled sync wrote")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("disabled sync created target", err)
	}
	if err := os.WriteFile(path, []byte("owner-created"), 0640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(path), "symlink.md")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: link}); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestHostProjectionCannotOverwriteOwnedAuthority(t *testing.T) {
	s, id, _ := hostFixture(t)
	ctx := context.Background()
	profile, err := s.registry.Profile(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.registry.RegisterProfile(ctx, registry.ProfileRegistration{ProfileID: "B", Create: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(profile.AuthorityDir, "persona", "IDENTITY.MD"), filepath.Join(other.AuthorityDir, "ACTMEM.MD"), filepath.Join(filepath.Dir(s.backupDir), "garden.db")} {
		before, beforeErr := os.ReadFile(path)
		if _, err = s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path}); err == nil {
			t.Fatal("owned data accepted as host projection", path)
		}
		after, afterErr := os.ReadFile(path)
		if string(before) != string(after) || os.IsNotExist(beforeErr) != os.IsNotExist(afterErr) {
			t.Fatal("source changed")
		}
	}
}
