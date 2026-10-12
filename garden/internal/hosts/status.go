package hosts

import (
	"context"
	"errors"
)

type HostStatus struct {
	BindingID        string            `json:"binding_id"`
	SavedRevisions   map[string]uint64 `json:"saved_revisions"`
	WrittenRevisions map[string]uint64 `json:"written_revisions,omitempty"`
	LoadedRevisions  map[string]uint64 `json:"loaded_revisions,omitempty"`
	WrittenHash      string            `json:"written_hash,omitempty"`
	LoadedHash       string            `json:"loaded_hash,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	AdapterVersion   string            `json:"adapter_version,omitempty"`
	State            string            `json:"state"`
}

func (s *Service) Apply(ctx context.Context, p FilePlan) (HostStatus, error) {
	if _, err := s.binding(ctx, p.BindingID, true); err != nil {
		return HostStatus{}, err
	}
	return HostStatus{}, errors.New("host file application pending")
}
