package agentapi

import (
	"context"
	"errors"
	"github.com/ProjectViVy/laputa/garden/internal/hosts"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

type HostFilePlan = hosts.FilePlan
type HostProjectionRequest = hosts.ProjectionRequest
type HostStatus = hosts.HostStatus
type HostCandidate = hosts.Candidate

// These are owner control methods. Host model tools cannot select file paths.
func (i *Installation) DiscoverHosts(paths []string) ([]HostCandidate, error) {
	out, err := i.hosts.Discover(paths)
	return out, hostError(err)
}
func (i *Installation) PreviewHostProjection(ctx context.Context, r HostProjectionRequest) (HostFilePlan, error) {
	out, err := i.hosts.Preview(ctx, r)
	return out, hostError(err)
}

func hostError(err error) error {
	if err == nil {
		return nil
	}
	if code := persona.CodeOf(err); code == "persona_edit_locked" || code == "persona_authority_drift" {
		return failure(code, err.Error())
	}
	err = accessError(err)
	var domain *Error
	if errors.As(err, &domain) {
		return domain
	}
	return failure("unavailable", "host projection operation unavailable")
}
func (i *Installation) ApplyHostProjection(ctx context.Context, p HostFilePlan) (HostStatus, error) {
	status, err := i.hosts.Apply(ctx, p)
	return status, hostError(err)
}
func (i *Installation) HostProjectionStatus(ctx context.Context, id string) (HostStatus, error) {
	status, err := i.hosts.Status(ctx, id)
	return status, hostError(err)
}
func (i *Installation) RevokeHostProjection(ctx context.Context, id string) (HostStatus, error) {
	status, err := i.hosts.Revoke(ctx, id)
	err = hostError(err)
	var domain *Error
	if status.State != "" && errors.As(err, &domain) {
		domain.Details["host_status"] = status
	}
	return status, err
}

// Authentication fixes binding/session at the transport. Qualified native
// adapter proof is additionally required; none is enabled by this core.
func (i *Installation) AcknowledgeHostLoaded(ctx context.Context, id, session, hash string) (HostStatus, error) {
	status, err := i.hosts.AcknowledgeLoaded(ctx, id, session, hash)
	return status, hostError(err)
}

// Native operations serialize with hosts.Close before owner.Close. Do not hold
// Owner.Enter while the coordinator resolves Owner.Subject (an exclusive lock).
// Host pulls remain bound credentials and owner configuration stays explicit.
type NativeBindingConfig = hosts.NativeBindingConfig
type NativePromptRequest = hosts.NativePromptRequest
type PromptProjection = hosts.PromptProjection
type RetryPreview = hosts.RetryPreview

func (i *Installation) ConfigureNativeHost(ctx context.Context, id string, cfg NativeBindingConfig) (HostStatus, error) {
	out, err := i.hosts.ConfigureNative(ctx, id, cfg)
	return out, hostError(err)
}
func (i *Installation) PrepareHostPrompt(ctx context.Context, id string, req NativePromptRequest) (PromptProjection, error) {
	out, err := i.hosts.PreparePrompt(ctx, id, req)
	return out, hostError(err)
}
func (i *Installation) PreviewHostRetry(ctx context.Context, id string) (RetryPreview, error) {
	out, err := i.hosts.RetryPreview(ctx, id)
	return out, hostError(err)
}
func (i *Installation) ConfirmHostRetry(ctx context.Context, id, previewID string) (HostStatus, error) {
	out, err := i.hosts.ConfirmRetry(ctx, id, previewID)
	return out, hostError(err)
}
