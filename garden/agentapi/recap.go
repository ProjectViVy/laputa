package agentapi

import (
	"context"
	"errors"

	"github.com/ProjectViVy/laputa/laputa/actmem"
)

// BindRecap returns a nonpersisting subject-level owner handle. It creates no
// Session/Run and carries only Open's trusted identity and optional domain grant.
// Reduced worker handles cannot acquire this control-plane grant.
func (c *Client) BindRecap() (*BoundClient, error) {
	if c == nil {
		return nil, failure("recap_unavailable", "Recap owner unavailable")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, failure("recap_unavailable", "Recap owner unavailable")
	}
	return &BoundClient{client: c, binding: c.identity}, nil
}

// recapStore holds the existing owner's lifecycle read lock through the entire
// call. Only its trusted user control-plane grant admits the complete domain;
// legacy agent/workspace read permissions never imply a free-text Recap grant.
func (b *BoundClient) recapStore() (*actmem.Store, actmem.RecapBinding, func(), error) {
	if b == nil || b.client == nil {
		return nil, actmem.RecapBinding{}, nil, failure("recap_unavailable", "Recap owner unavailable")
	}
	c := b.client
	c.mu.RLock()
	deny := func(code, message string) (*actmem.Store, actmem.RecapBinding, func(), error) {
		c.mu.RUnlock()
		return nil, actmem.RecapBinding{}, nil, failure(code, message)
	}
	if c.runtime == nil || c.runtime.Actmem == nil {
		return deny("recap_unavailable", "Recap owner unavailable")
	}
	if Authorize(c.runtime.ProfileID, b.binding, b.caller(), OpActmemSave) != nil {
		return deny("recap_scope_denied", "complete Recap domain grant required")
	}
	binding := c.runtime.RecapBinding
	if binding == (actmem.RecapBinding{}) {
		return deny("recap_unavailable", "trusted Recap binding unavailable")
	}
	return c.runtime.Actmem, binding, c.mu.RUnlock, nil
}

func recapFailure(err error) error {
	if err == nil {
		return nil
	}
	var owner *actmem.Error
	if errors.As(err, &owner) {
		return failure(owner.Code, "Recap operation failed")
	}
	return err
}

func (b *BoundClient) ReadRecap(ctx context.Context) (actmem.RecapHead, error) {
	store, binding, unlock, err := b.recapStore()
	if err != nil {
		return actmem.RecapHead{}, err
	}
	defer unlock()
	head, err := store.ReadRecap(ctx, binding)
	return head, recapFailure(err)
}

// CommitRecap stamps the trusted binding. A supplied binding is only a
// consistency assertion, never a way to select another subject or domain.
func (b *BoundClient) CommitRecap(ctx context.Context, request actmem.RecapCommit) (actmem.RecapHead, error) {
	store, binding, unlock, err := b.recapStore()
	if err != nil {
		return actmem.RecapHead{}, err
	}
	defer unlock()
	if request.Binding != (actmem.RecapBinding{}) && request.Binding != binding {
		return actmem.RecapHead{}, failure("recap_scope_denied", "request does not match trusted Recap binding")
	}
	request.Binding = binding
	head, err := store.CommitRecap(ctx, request)
	return head, recapFailure(err)
}

func (b *BoundClient) LookupRecapOperation(ctx context.Context, operationID string) (actmem.RecapHead, bool, error) {
	store, binding, unlock, err := b.recapStore()
	if err != nil {
		return actmem.RecapHead{}, false, err
	}
	defer unlock()
	head, found, err := store.LookupRecapOperation(ctx, binding, operationID)
	return head, found, recapFailure(err)
}

func (b *BoundClient) ListRecapHistory(ctx context.Context, before uint64, limit int) ([]actmem.RecapHead, error) {
	store, binding, unlock, err := b.recapStore()
	if err != nil {
		return nil, err
	}
	defer unlock()
	history, err := store.ListRecapHistory(ctx, binding, before, limit)
	return history, recapFailure(err)
}

func (b *BoundClient) ReadRecapRevision(ctx context.Context, revision uint64) (actmem.RecapHead, error) {
	store, binding, unlock, err := b.recapStore()
	if err != nil {
		return actmem.RecapHead{}, err
	}
	defer unlock()
	head, err := store.ReadRecapRevision(ctx, binding, revision)
	return head, recapFailure(err)
}
