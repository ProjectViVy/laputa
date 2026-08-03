package mailbox

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dashimaki/garden/internal/evolution"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "mailbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestInboxReviewLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	item, err := store.AddInbox(ctx, map[string]any{"note": "hello"}, []string{"mentle:cards/abc"})
	if err != nil {
		t.Fatal(err)
	}
	if item.State != StateReceived || item.Direction != DirectionInbox {
		t.Fatalf("item=%+v", item)
	}
	reviewed, err := store.Review(ctx, item.ID, StateApproved, "looks good")
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.State != StateApproved {
		t.Fatalf("state=%s", reviewed.State)
	}
	if _, err := store.Review(ctx, item.ID, StateApproved, "again"); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("second review err=%v", err)
	}
	if _, err := store.Review(ctx, item.ID, "maybe", ""); !errors.Is(err, evolution.ErrInvalidDecision) {
		t.Fatalf("invalid decision err=%v", err)
	}
}

func TestOutboxTransitionsAndIllegalRejection(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	item, err := store.QueueOutbox(ctx, map[string]any{"kind": "proposal"}, []string{"mentle:cards/abc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(ctx, item.ID, StateAcked, "skip"); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("illegal transition err=%v", err)
	}
	dispatched, err := store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched.State != StateAcked || dispatched.Reason != "local delivery (no transport)" {
		t.Fatalf("dispatched=%+v", dispatched)
	}
}

func TestDispatchRetryThenDeadLetter(t *testing.T) {
	store := openTestStore(t)
	store.MaxRetries = 1
	store.Delivery = func(ctx context.Context, item Item) error { return errors.New("transport down") }
	ctx := context.Background()
	item, err := store.QueueOutbox(ctx, map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	retried, err := store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retried.State != StateQueued || retried.RetryCount != 1 {
		t.Fatalf("retried=%+v", retried)
	}
	dead, err := store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dead.State != StateDeadLetter || dead.Reason != "delivery_failed: retries exhausted" {
		t.Fatalf("dead=%+v", dead)
	}
	deadList, err := store.ListDeadLetter(ctx)
	if err != nil || len(deadList) != 1 {
		t.Fatalf("deadList=%+v err=%v", deadList, err)
	}
}

func TestPrivacyGateDeadLettersProhibitedPayload(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	item, err := store.QueueOutbox(ctx, map[string]any{"api_key": "hunter2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.State != StateDeadLetter || blocked.Reason != "privacy_gate" {
		t.Fatalf("blocked=%+v", blocked)
	}
	if blocked.Leakage == nil || blocked.Leakage.Clean {
		t.Fatalf("leakage=%+v", blocked.Leakage)
	}
}

func TestPrivacyGateDeadLettersProhibitedRefs(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	item, err := store.QueueOutbox(ctx, map[string]any{}, []string{".laputa/sections/01-frozen"})
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := store.Dispatch(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.State != StateDeadLetter || blocked.Leakage == nil || blocked.Leakage.Clean {
		t.Fatalf("blocked=%+v", blocked)
	}
}

func TestHubPublishDisabledByDefault(t *testing.T) {
	policy := evolution.DefaultHubPolicy()
	if policy.HubPublishEnabled || policy.CanPublishHub(true) {
		t.Fatal("hub publish must stay disabled by default")
	}
}

func TestListFiltersByDirectionAndState(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if _, err := store.AddInbox(ctx, map[string]any{}, nil); err != nil {
		t.Fatal(err)
	}
	out, err := store.QueueOutbox(ctx, map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inbox, err := store.List(ctx, DirectionInbox, "")
	if err != nil || len(inbox) != 1 {
		t.Fatalf("inbox=%+v err=%v", inbox, err)
	}
	queued, err := store.List(ctx, DirectionOutbox, StateQueued)
	if err != nil || len(queued) != 1 || queued[0].ID != out.ID {
		t.Fatalf("queued=%+v err=%v", queued, err)
	}
	empty, err := store.List(ctx, DirectionOutbox, StateAcked)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty=%+v err=%v", empty, err)
	}
}
