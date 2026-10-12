package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type Profile struct {
	ProfileID      string `json:"profile_id"`
	DisplayName    string `json:"display_name"`
	AuthorityDir   string `json:"authority_dir"`
	InstallationID string `json:"installation_id"`
}
type ProfileRegistration struct {
	ProfileID    string `json:"profile_id"`
	DisplayName  string `json:"display_name"`
	AuthorityDir string `json:"authority_dir"`
	Create       bool   `json:"create"`
}

var profileIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ValidateProfileID(id string) error {
	if !profileIDPattern.MatchString(id) || strings.HasPrefix(id, "shared:") || len(id) > 128 {
		return errors.New("invalid profile_id")
	}
	return nil
}

func (s *Store) RegisterProfile(ctx context.Context, req ProfileRegistration) (Profile, error) {
	if req.ProfileID == "" {
		if !req.Create {
			return Profile{}, errors.New("attachment requires explicit profile_id")
		}
		req.ProfileID = uuid.NewString()
	}
	if err := ValidateProfileID(req.ProfileID); err != nil {
		return Profile{}, err
	}
	if req.AuthorityDir == "" {
		if !req.Create {
			return Profile{}, errors.New("attachment requires authority_dir")
		}
		req.AuthorityDir = filepath.Join(s.home, "subjects", req.ProfileID)
	}
	path, err := CanonicalPath(req.AuthorityDir)
	if err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return Profile{}, errors.New("registry closed")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT profile_id,display_name,authority_dir,installation_id FROM profiles WHERE profile_id=? OR authority_dir=?`, req.ProfileID, path)
	if err != nil {
		return Profile{}, err
	}
	var existing []Profile
	for rows.Next() {
		var p Profile
		if err = rows.Scan(&p.ProfileID, &p.DisplayName, &p.AuthorityDir, &p.InstallationID); err != nil {
			rows.Close()
			return Profile{}, err
		}
		existing = append(existing, p)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return Profile{}, err
	}
	if len(existing) > 0 {
		p := existing[0]
		if len(existing) != 1 || p.ProfileID != req.ProfileID || p.AuthorityDir != path || p.InstallationID != s.installationID {
			return Profile{}, errors.New("profile_conflict: ID or authority already registered")
		}
		return p, nil
	}
	if req.Create {
		if err = os.MkdirAll(path, 0700); err != nil {
			return Profile{}, err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return Profile{}, err
	}
	if !info.IsDir() {
		return Profile{}, errors.New("authority_dir must be a directory")
	}
	p := Profile{ProfileID: req.ProfileID, DisplayName: req.DisplayName, AuthorityDir: path, InstallationID: s.installationID}
	if p.DisplayName == "" {
		p.DisplayName = p.ProfileID
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO profiles(profile_id,display_name,authority_dir,installation_id) VALUES(?,?,?,?)`, p.ProfileID, p.DisplayName, p.AuthorityDir, p.InstallationID)
	return p, err
}

func (s *Store) ListProfiles(ctx context.Context) ([]Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("registry closed")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT profile_id,display_name,authority_dir,installation_id FROM profiles ORDER BY profile_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Profile{}
	for rows.Next() {
		var p Profile
		if err = rows.Scan(&p.ProfileID, &p.DisplayName, &p.AuthorityDir, &p.InstallationID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Profile(ctx context.Context, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return Profile{}, errors.New("registry closed")
	}
	var p Profile
	err := s.db.QueryRowContext(ctx, `SELECT profile_id,display_name,authority_dir,installation_id FROM profiles WHERE profile_id=?`, id).Scan(&p.ProfileID, &p.DisplayName, &p.AuthorityDir, &p.InstallationID)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, fmt.Errorf("profile_not_found: %s", id)
	}
	return p, err
}
