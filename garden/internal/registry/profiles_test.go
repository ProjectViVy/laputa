package registry

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func profileStore(t *testing.T) (*Store, string) {
	t.Helper()
	home := t.TempDir()
	s, err := Open(filepath.Join(home, "garden.db"), home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, home
}

func TestProfilePathConflict(t *testing.T) {
	s, home := profileStore(t)
	root := filepath.Join(home, "original")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "IDENTITY.MD"), []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	p, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "diva", AuthorityDir: root, DisplayName: "DIVA"})
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	again, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "diva", AuthorityDir: alias})
	if err != nil || again.ProfileID != p.ProfileID || again.AuthorityDir != root {
		t.Fatalf("alias: %+v %v", again, err)
	}
	if _, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "other", AuthorityDir: alias}); err == nil {
		t.Fatal("same authority registered under another ID")
	}
	other := filepath.Join(home, "other")
	_ = os.Mkdir(other, 0700)
	if _, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "diva", AuthorityDir: other}); err == nil {
		t.Fatal("ID collision merged")
	}
	for _, id := range []string{"../escape", "shared:x", "a/b", "a\\b", " ", "."} {
		if _, err := s.RegisterProfile(ctx, ProfileRegistration{ProfileID: id, Create: true}); err == nil {
			t.Fatalf("invalid profile %q", id)
		}
	}
	content, _ := os.ReadFile(filepath.Join(root, "IDENTITY.MD"))
	if string(content) != "unchanged" {
		t.Fatal("registration changed authority")
	}
}

func TestProfileRegistryRestart(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "garden.db")
	ctx := context.Background()
	s, err := Open(path, home)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.RegisterProfile(ctx, ProfileRegistration{DisplayName: "新人格", Create: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.ProfileID == "" || p.InstallationID == "" || p.AuthorityDir != filepath.Join(home, "subjects", p.ProfileID) {
		t.Fatal(p)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path, home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.ListProfiles(ctx)
	if err != nil || !reflect.DeepEqual(got, []Profile{p}) {
		t.Fatalf("restart: %+v %v", got, err)
	}
	if _, err = s.RegisterProfile(ctx, ProfileRegistration{ProfileID: "missing", AuthorityDir: filepath.Join(home, "does-not-exist")}); err == nil {
		t.Fatal("attachment invented missing authority")
	}
}

func TestProfileCreateCanonicalParent(t *testing.T) {
	s, home := profileStore(t)
	real := filepath.Join(home, "real")
	_ = os.Mkdir(real, 0700)
	alias := filepath.Join(home, "link")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	p, err := s.RegisterProfile(context.Background(), ProfileRegistration{ProfileID: "new", AuthorityDir: filepath.Join(alias, "new"), Create: true})
	if err != nil || p.AuthorityDir != filepath.Join(real, "new") {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestSessionKeyExactTuple(t *testing.T) {
	seen := map[string]bool{}
	for _, parts := range [][3]string{{"A", "host", "chat"}, {"B", "host", "chat"}, {"A", "other", "chat"}, {"A", "a:b", "c"}, {"A", "a", "b:c"}} {
		key, err := SessionKey(parts[0], parts[1], parts[2])
		if err != nil || seen[key] {
			t.Fatal(key, err)
		}
		seen[key] = true
	}
	if _, err := SessionKey("A", "host", ""); err == nil {
		t.Fatal("empty session admitted")
	}
}
