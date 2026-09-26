package mailbox

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMailboxRetryStateSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mailbox.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	item, err := store.QueueOutbox(ctx, map[string]any{"kind": "capsule"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	store.Delivery = func(context.Context, Item) error { return errDeliveryFailure }
	_, err = store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateQueued || got.RetryCount != 1 {
		t.Fatalf("retry after reopen: %+v", got)
	}
}

var errDeliveryFailure = &deliveryFailure{}

type deliveryFailure struct{}

func (*deliveryFailure) Error() string { return "temporary delivery failure" }
