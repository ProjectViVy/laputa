package hosts

import (
	"context"
	"errors"
	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"testing"
)

func TestLoadedRequiresExactEvidence(t *testing.T) {
	s, id, _, p := activeHost(t)
	ctx := context.Background()
	st, err := s.Apply(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if st.LoadedHash != "" {
		t.Fatal("write reported loaded")
	}
	if _, err = s.AcknowledgeLoaded(ctx, id, "hook-submitted", st.WrittenHash); err == nil {
		t.Fatal("generic hook became loaded")
	}
	s.LoadedEvidence = func(_ context.Context, _ registry.HostBinding, session, hash string) error {
		if session != "actual-session" || hash != st.WrittenHash {
			return errors.New("exact loaded evidence absent")
		}
		return nil
	}
	for _, pair := range [][2]string{{"wrong-session", st.WrittenHash}, {"actual-session", "wrong-hash"}} {
		if _, err = s.AcknowledgeLoaded(ctx, id, pair[0], pair[1]); err == nil {
			t.Fatal("false load acknowledged")
		}
	}
	loaded, err := s.AcknowledgeLoaded(ctx, id, "actual-session", st.WrittenHash)
	if err != nil || loaded.State != "loaded" || loaded.LoadedHash != st.WrittenHash || len(loaded.LoadedRevisions) != 4 {
		t.Fatal(loaded, err)
	}
	status, err := s.Status(ctx, id)
	if err != nil || status.LoadedHash != loaded.LoadedHash {
		t.Fatal("persisted status lost", status, err)
	}
}
