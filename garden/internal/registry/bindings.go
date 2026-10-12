package registry

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type HostControls struct {
	PersonaSync bool `json:"persona_sync"`
	Recall      bool `json:"recall"`
	Capture     bool `json:"capture"`
}
type HostBinding struct {
	Projection    json.RawMessage `json:"projection,omitempty"`
	BindingID     string          `json:"binding_id"`
	ProfileID     string          `json:"profile_id"`
	AgentID       string          `json:"agent_id"`
	Platform      string          `json:"platform"`
	WorkspaceID   string          `json:"workspace_id"`
	Controls      HostControls    `json:"controls"`
	Operations    []string        `json:"operations"`
	Worker        bool            `json:"worker"`
	Revoked       bool            `json:"revoked"`
	SelectedCards []string        `json:"selected_cards,omitempty"`
}
type AccessError struct{ Code, Message string }

func (e *AccessError) Error() string { return e.Message }
func denied(code string) error       { return &AccessError{Code: code, Message: code} }

func (s *Store) CreateHostBinding(ctx context.Context, b HostBinding) (HostBinding, string, error) {
	if err := ValidateProfileID(b.ProfileID); err != nil {
		return HostBinding{}, "", err
	}
	if strings.TrimSpace(b.AgentID) == "" || strings.TrimSpace(b.Platform) == "" {
		return HostBinding{}, "", denied("invalid_binding")
	}
	if len(b.Operations) == 0 {
		b.Operations = []string{"bootstrap", "search", "expand", "index_health", "capture", "remember", "persona_get", "actmem_read", "actmem_query"}
	}
	for _, op := range b.Operations {
		if !knownOperation(op) {
			return HostBinding{}, "", denied("invalid_binding")
		}
	}
	b.Controls = HostControls{} // Creation/discovery never activates a host.
	b.BindingID = uuid.NewString()
	b.Revoked = false
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return HostBinding{}, "", err
	}
	credential := base64.RawURLEncoding.EncodeToString(key)
	digest := sha256.Sum256([]byte(credential))
	if err := s.insertBinding(ctx, b, hex.EncodeToString(digest[:])); err != nil {
		return HostBinding{}, "", err
	}
	return b, credential, nil
}

// TrustedBinding persists stable in-process identity without an external
// credential. Only the trusted composition root calls this, not a transport.
func (s *Store) TrustedBinding(ctx context.Context, b HostBinding) (HostBinding, error) {
	identity, _ := json.Marshal([]string{b.ProfileID, b.AgentID, b.Platform, b.WorkspaceID})
	digest := sha256.Sum256(identity)
	b.BindingID = "trusted-" + hex.EncodeToString(digest[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return HostBinding{}, errors.New("registry closed")
	}
	existing, err := s.bindingLocked(ctx, b.BindingID)
	if err == nil {
		return existing, nil
	}
	var ae *AccessError
	if !errors.As(err, &ae) || ae.Code != "invalid_binding" {
		return HostBinding{}, err
	}
	raw, _ := json.Marshal(b)
	_, err = s.db.ExecContext(ctx, `INSERT INTO host_bindings(binding_id,profile_id,body,credential_digest) VALUES(?,?,?,NULL)`, b.BindingID, b.ProfileID, string(raw))
	return b, err
}

func (s *Store) insertBinding(ctx context.Context, b HostBinding, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("registry closed")
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO host_bindings(binding_id,profile_id,body,credential_digest) VALUES(?,?,?,?)`, b.BindingID, b.ProfileID, string(raw), digest)
	return err
}
func (s *Store) bindingLocked(ctx context.Context, id string) (HostBinding, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT body FROM host_bindings WHERE binding_id=?`, id).Scan(&raw); err != nil {
		return HostBinding{}, denied("invalid_binding")
	}
	var b HostBinding
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		return HostBinding{}, err
	}
	return b, nil
}
func (s *Store) Binding(ctx context.Context, id string) (HostBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return HostBinding{}, errors.New("registry closed")
	}
	return s.bindingLocked(ctx, id)
}
func (s *Store) Authenticate(ctx context.Context, credential string) (HostBinding, error) {
	if credential == "" {
		return HostBinding{}, denied("authentication_required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return HostBinding{}, errors.New("registry closed")
	}
	digest := sha256.Sum256([]byte(credential))
	var id string
	if err := s.db.QueryRowContext(ctx, `SELECT binding_id FROM host_bindings WHERE credential_digest=?`, hex.EncodeToString(digest[:])).Scan(&id); err != nil {
		return HostBinding{}, denied("authentication_required")
	}
	b, err := s.bindingLocked(ctx, id)
	if err != nil {
		return HostBinding{}, err
	}
	if b.Revoked {
		return HostBinding{}, denied("binding_revoked")
	}
	return b, nil
}
func (s *Store) ListHostBindings(ctx context.Context) ([]HostBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("registry closed")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT body FROM host_bindings WHERE credential_digest IS NOT NULL ORDER BY binding_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HostBinding{}
	for rows.Next() {
		var raw string
		var b HostBinding
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *Store) SetHostControls(ctx context.Context, id string, c HostControls) error {
	return s.updateBinding(ctx, id, func(b *HostBinding) { b.Controls = c })
}
func (s *Store) RevokeHostBinding(ctx context.Context, id string) error {
	return s.updateBinding(ctx, id, func(b *HostBinding) { b.Revoked = true; b.Controls = HostControls{} })
}
func (s *Store) updateBinding(ctx context.Context, id string, update func(*HostBinding)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("registry closed")
	}
	b, err := s.bindingLocked(ctx, id)
	if err != nil {
		return err
	}
	if b.Revoked {
		return denied("binding_revoked")
	}
	update(&b)
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE host_bindings SET body=? WHERE binding_id=?`, string(raw), id)
	return err
}

func knownOperation(op string) bool {
	switch op {
	case "bootstrap", "search", "expand", "index_health", "capture", "remember", "persona_get", "persona_propose", "persona_p16", "persona_review", "persona_repair", "actmem_read", "actmem_query", "actmem_write", "actmem_maintain", "actmem_save":
		return true
	}
	return false
}

// SetProjection changes only owner control metadata, preserving current
// controls/revocation. Cleanup receipts may be saved after credential revoke.
func (s *Store) SetProjection(ctx context.Context, id string, value json.RawMessage) error {
	if !json.Valid(value) || len(value) > 32<<10 {
		return errors.New("invalid projection metadata")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("registry closed")
	}
	b, err := s.bindingLocked(ctx, id)
	if err != nil {
		return err
	}
	b.Projection = append(json.RawMessage{}, value...)
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE host_bindings SET body=? WHERE binding_id=?`, string(raw), id)
	return err
}
