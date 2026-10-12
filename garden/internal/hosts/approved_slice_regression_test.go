package hosts

import (
	"context"
	"os"
	"testing"
)

func TestApprovedSliceConcurrentRevokePreservesOutsideEdit(t *testing.T) {
	s, id, path, p := activeHost(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, p); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := string(raw) + "CONCURRENT_USER_EDIT\n"
	if err = os.WriteFile(path, []byte(edited), 0640); err != nil {
		t.Fatal(err)
	}
	status, err := s.Revoke(ctx, id)
	if err == nil || status.State != "conflict" || status.ManualAction != "remove_projection" || status.ManualPath != path {
		t.Fatal(status, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != edited {
		t.Fatal("revoke rewrote outside edit", string(after), err)
	}
}
