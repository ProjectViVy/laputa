package agentapi

import (
	"context"
	"github.com/ProjectViVy/laputa/garden/internal/hosts"
)

type HostFilePlan = hosts.FilePlan
type HostProjectionRequest = hosts.ProjectionRequest
type HostStatus = hosts.HostStatus
type HostCandidate = hosts.Candidate

// These are owner control methods. Host model tools cannot select file paths.
func (i *Installation) DiscoverHosts(paths []string) ([]HostCandidate, error) {
	return i.hosts.Discover(paths)
}
func (i *Installation) PreviewHostProjection(ctx context.Context, r HostProjectionRequest) (HostFilePlan, error) {
	return i.hosts.Preview(ctx, r)
}
