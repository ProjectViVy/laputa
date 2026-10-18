package runtimecore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProjectViVy/laputa/laputa/actmem"
)

func TestIssue58RecapResolverGetsCanonicalOwnedRoot(t *testing.T) {
	cfg := testConfig(t)
	original := filepath.Join(t.TempDir(), "actual")
	alternative := filepath.Join(t.TempDir(), "alternative")
	for _, dir := range []string{original, alternative} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(t.TempDir(), "profile-alias")
	if err := os.Symlink(original, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cfg.PersonaDir = alias
	canonical, err := filepath.EvalSymlinks(original)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical, "actmem")
	called := false
	cfg.ResolveRecapBinding = func(_ context.Context, root string) (actmem.RecapBinding, error) {
		called = true
		if root != want {
			t.Errorf("resolver got unresolved owner root %q, want %q", root, want)
		}
		// Binding observes a locked, frozen real path even if its input alias changes
		// before the first C1 operation or ingestion worker starts.
		if competitor, err := actmem.Open(original); actmem.CodeOf(err) != "recap_owner_busy" {
			if competitor != nil {
				_ = competitor.Close()
			}
			t.Errorf("resolver ran before owner acquisition: %v", err)
		}
		if err := os.Remove(alias); err != nil {
			return actmem.RecapBinding{}, err
		}
		if err := os.Symlink(alternative, alias); err != nil {
			return actmem.RecapBinding{}, err
		}
		return actmem.RecapBinding{SubjectID: "subject", DomainID: "personal"}, nil
	}
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	if !called || core.Actmem.Root() != want {
		t.Fatalf("owner changed after binding: %q", core.Actmem.Root())
	}
	if _, err := core.Actmem.CommitRecap(context.Background(), actmem.RecapCommit{Binding: core.RecapBinding, OperationID: "bound", Cause: "edit", Text: "冻结的所有者"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(want, actmem.ACTMEMFileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(alternative, "actmem")); !os.IsNotExist(err) {
		t.Fatalf("alias retarget moved owner: %v", err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := actmem.Open(original)
	if err != nil {
		t.Fatalf("close leaked canonical owner: %v", err)
	}
	_ = reopened.Close()
}

func TestIssue58RecapArmedOpenFailureReleasesOwner(t *testing.T) {
	for _, failure := range []string{"resolver_error", "invalid_binding", "downstream_open"} {
		t.Run(failure, func(t *testing.T) {
			cfg := testConfig(t)
			want := errors.New("binding registry failed")
			cfg.ResolveRecapBinding = func(_ context.Context, root string) (actmem.RecapBinding, error) {
				wantRoot, err := filepath.EvalSymlinks(filepath.Join(cfg.PersonaDir, "actmem"))
				if err != nil {
					t.Errorf("resolve expected owner root: %v", err)
				} else if root != wantRoot {
					t.Errorf("unexpected owner root %q want %q", root, wantRoot)
				}
				competitor, err := actmem.Open(cfg.PersonaDir)
				if competitor != nil {
					_ = competitor.Close()
				}
				if actmem.CodeOf(err) != "recap_owner_busy" {
					t.Errorf("binding callback was not under ownership: %v", err)
				}
				if failure == "resolver_error" {
					return actmem.RecapBinding{}, want
				}
				if failure == "invalid_binding" {
					return actmem.RecapBinding{}, nil
				}
				return actmem.RecapBinding{SubjectID: "subject", DomainID: "personal"}, nil
			}
			if failure == "downstream_open" {
				if err := os.MkdirAll(cfg.StateDB, 0700); err != nil {
					t.Fatal(err)
				}
			}
			core, err := Open(context.Background(), cfg)
			if core != nil {
				_ = core.Close()
			}
			if err == nil {
				t.Fatal("expected armed construction failure")
			}
			if failure == "resolver_error" && !errors.Is(err, want) {
				t.Fatalf("resolver error lost: %v", err)
			}
			reopened, err := actmem.Open(cfg.PersonaDir)
			if err != nil {
				t.Fatalf("partial Open leaked ACTMEM owner: %v", err)
			}
			_ = reopened.Close()
			if _, err := os.Stat(filepath.Join(cfg.PersonaDir, "actmem", actmem.ACTMEMFileName)); !os.IsNotExist(err) {
				t.Fatalf("construction initialized ACTMEM: %v", err)
			}
		})
	}
}

func TestIssue58RecapUnarmedOpenKeepsLazyStore(t *testing.T) {
	cfg := testConfig(t)
	core, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	if _, err := os.Stat(core.Actmem.Root()); !os.IsNotExist(err) {
		t.Fatalf("nil resolver touched ACTMEM directory: %v", err)
	}
}
