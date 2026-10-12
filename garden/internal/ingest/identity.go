package ingest

import (
	"context"
	"encoding/json"
	"errors"
)

// Identity is trusted admission metadata; request JSON cannot grant a scope.
// Physical session/event keys are namespaced; original host IDs remain provenance.
type Identity struct {
	ProfileID     string `json:"profile_id"`
	BindingID     string `json:"binding_id"`
	HostSessionID string `json:"host_session_id"`
	HostEventID   string `json:"host_event_id"`
	Scope         string `json:"scope"`
}

func (s *Service) identity(ctx context.Context, id string) (Identity, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT identity_json FROM ingestions WHERE ingestion_id=?`, id).Scan(&raw); err != nil {
		return Identity{}, err
	}
	if raw == "" {
		return Identity{}, nil
	}
	var v Identity
	err := json.Unmarshal([]byte(raw), &v)
	return v, err
}
func (s *Service) admitted(ctx context.Context, id string) (Identity, error) {
	v, err := s.identity(ctx, id)
	if err != nil {
		return v, err
	}
	if s.CheckIdentity != nil {
		err = s.CheckIdentity(ctx, v)
	}
	return v, err
}
func initializeIdentityColumn(s *Service) error {
	rows, err := s.db.Query(`PRAGMA table_info(ingestions)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var id, nn, pk int
		var name, kind string
		var def any
		if err = rows.Scan(&id, &name, &kind, &nn, &def, &pk); err != nil {
			rows.Close()
			return err
		}
		found = found || name == "identity_json"
	}
	err = rows.Err()
	rows.Close()
	if err != nil || found {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE ingestions ADD COLUMN identity_json TEXT NOT NULL DEFAULT ''`)
	return err
}
func (s *Service) rowScope(ctx context.Context, id, workspace string) (string, error) {
	v, err := s.admitted(ctx, id)
	if err != nil {
		return "", err
	}
	if v.ProfileID != "" {
		if v.Scope == "" {
			return "", errors.New("unclassified capture scope")
		}
		return v.Scope, nil
	}
	if s.CheckIdentity != nil {
		return "", errors.New("unclassified legacy capture")
	}
	return s.scopeFor(workspace), nil
}
