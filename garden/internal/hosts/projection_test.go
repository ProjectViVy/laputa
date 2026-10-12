package hosts

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hostFixture(t *testing.T) (*Service, string, string) {
	t.Helper()
	home := t.TempDir()
	reg, err := registry.Open(filepath.Join(home, "garden.db"), home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reg.Close() })
	p, err := reg.RegisterProfile(context.Background(), registry.ProfileRegistration{ProfileID: "A", Create: true})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := persona.Open(p.AuthorityDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Initialize(persona.Initialization{Identity: "identity", Relationship: "relationship", Redline: "redline", User: "user", World: "WORLD_SECRET"}, "user", persona.SourceInit, "init")
	if err != nil {
		t.Fatal(err)
	}
	for kind, body := range map[persona.Kind]string{persona.KindDream: "OPTIONAL_DREAM", persona.KindDark: "OPTIONAL_DARK"} {
		if _, e := svc.SaveUserDocument(kind, body, 0, "user", persona.SourceUserDirect, "optional"); e != nil {
			t.Fatal(e)
		}
	}
	b, _, err := reg.CreateHostBinding(context.Background(), registry.HostBinding{ProfileID: "A", AgentID: "host", Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	s := New(reg, filepath.Join(home, "backups"), func(context.Context, string) (*persona.Service, error) { return svc, nil })
	return s, b.BindingID, filepath.Join(home, "CLAUDE.md")
}
func TestHostDiscoveryInert(t *testing.T) {
	s, id, path := hostFixture(t)
	before, _ := s.registry.Binding(context.Background(), id)
	got, err := s.Discover([]string{path})
	if err != nil || len(got) != 1 || got[0].State != "inactive" {
		t.Fatal(got, err)
	}
	after, _ := s.registry.Binding(context.Background(), id)
	if after.Controls != before.Controls {
		t.Fatal("discovery activated")
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("discovery touched file", err)
	}
}
func TestHostProjectionSelection(t *testing.T) {
	s, id, path := hostFixture(t)
	plan, err := s.Preview(context.Background(), ProjectionRequest{BindingID: id, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.SourceRevisions) != 4 || !strings.Contains(plan.ProposedContent, "identity") || strings.Contains(plan.ProposedContent, "OPTIONAL_DREAM") || strings.Contains(plan.ProposedContent, "WORLD_SECRET") {
		t.Fatal(plan)
	}
	for _, kind := range []persona.Kind{persona.KindWorld, persona.Kind(99)} {
		if _, err = s.Preview(context.Background(), ProjectionRequest{BindingID: id, Path: path, Kinds: []persona.Kind{kind}}); err == nil {
			t.Fatal("forbidden static projection", kind)
		}
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("preview wrote file")
	}
}
func TestHostControlsIndependent(t *testing.T) {
	s, id, path := hostFixture(t)
	ctx := context.Background()
	for mask := 0; mask < 8; mask++ {
		controls := registry.HostControls{PersonaSync: mask&1 != 0, Recall: mask&2 != 0, Capture: mask&4 != 0}
		if err := s.registry.SetHostControls(ctx, id, controls); err != nil {
			t.Fatal(err)
		}
		for _, pair := range []struct {
			op   string
			want bool
		}{{"search", controls.Recall}, {"capture", controls.Capture}, {"persona_get", controls.PersonaSync}} {
			_, err := s.registry.Resolve(ctx, registry.AccessRequest{BindingID: id, SessionID: "test", Operation: pair.op})
			if (err == nil) != pair.want {
				t.Fatal(mask, pair, err)
			}
		}
		plan, err := s.Preview(ctx, ProjectionRequest{BindingID: id, Path: path})
		if err != nil {
			t.Fatal(err)
		}
		if !controls.PersonaSync {
			if _, err = s.Apply(ctx, plan); err == nil {
				t.Fatal("recall/capture enabled file sync")
			}
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("inactive sync touched file")
	}
}
