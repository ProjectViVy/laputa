package registry

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type SharedSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type SharedMember struct {
	ProfileID string `json:"profile_id"`
	Access    string `json:"access"`
}

func (s *Store) CreateSharedSpace(ctx context.Context, name string) (SharedSpace, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 256 {
		return SharedSpace{}, errors.New("invalid shared-space name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return SharedSpace{}, errors.New("registry closed")
	}
	space := SharedSpace{ID: uuid.NewString(), Name: name}
	_, err := s.db.ExecContext(ctx, `INSERT INTO shared_spaces(space_id,name) VALUES(?,?)`, space.ID, space.Name)
	return space, err
}
func (s *Store) SetSharedMember(ctx context.Context, spaceID, profileID, access string) error {
	if access != "read" && access != "read_write" {
		return errors.New("invalid shared access")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("registry closed")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO shared_members(space_id,profile_id,access) VALUES(?,?,?) ON CONFLICT(space_id,profile_id) DO UPDATE SET access=excluded.access`, spaceID, profileID, access)
	return err
}
func (s *Store) RemoveSharedMember(ctx context.Context, spaceID, profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("registry closed")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM shared_members WHERE space_id=? AND profile_id=?`, spaceID, profileID)
	return err
}
func (s *Store) ListSharedSpaces(ctx context.Context) ([]SharedSpace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("registry closed")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT space_id,name FROM shared_spaces ORDER BY space_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SharedSpace{}
	for rows.Next() {
		var v SharedSpace
		if err = rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ListSharedMembers(ctx context.Context, spaceID string) ([]SharedMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("registry closed")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT profile_id,access FROM shared_members WHERE space_id=? ORDER BY profile_id`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SharedMember{}
	for rows.Next() {
		var v SharedMember
		if err = rows.Scan(&v.ProfileID, &v.Access); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
