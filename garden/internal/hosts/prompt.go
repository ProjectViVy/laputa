package hosts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

const nativeAdapterVersion = "garden-native-prompt/v1"
const nativeMode = "native_prompt"

type NativeBindingConfig struct {
	AdapterID        string         `json:"adapter_id"`
	AdapterVersion   string         `json:"adapter_version"`
	HostVersion      string         `json:"host_version"`
	HostInstance     string         `json:"host_instance"`
	ScopeFingerprint string         `json:"scope_fingerprint"`
	Kinds            []persona.Kind `json:"kinds"`
}
type NativePromptRequest struct {
	AdapterVersion   string `json:"adapter_version"`
	HostVersion      string `json:"host_version"`
	HostInstance     string `json:"host_instance"`
	SessionID        string `json:"session_id"`
	ScopeFingerprint string `json:"scope_fingerprint"`
}
type PromptProjection struct {
	BindingID       string            `json:"binding_id"`
	SourceRevisions map[string]uint64 `json:"source_revisions"`
	Hash            string            `json:"hash"`
	Markdown        string            `json:"markdown"`
}
type RetryPreview struct {
	ID               string           `json:"preview_id"`
	Projection       PromptProjection `json:"projection"`
	Mode             string           `json:"mode"`
	ScopeFingerprint string           `json:"scope_fingerprint"`
	ExpiresAt        time.Time        `json:"expires_at"`
}
type nativePreview struct {
	RetryPreview
	fingerprint string
	hashes      map[string]string
	generations map[persona.Kind]uint64
	armed       bool
}

func normalizeNativeConfig(cfg NativeBindingConfig) (NativeBindingConfig, error) {
	versions := map[string]string{"openclaw": "2026.10.2", "hermes": "0.21.7", "codex-cli": "0.162.1", "claude-code": "2.1.296"}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || versions[cfg.AdapterID] == "" || versions[cfg.AdapterID] != cfg.HostVersion || cfg.AdapterVersion != nativeAdapterVersion {
		return cfg, failure("unsupported_host", "native adapter/version/platform is not qualified")
	}
	scope, err := hex.DecodeString(cfg.ScopeFingerprint)
	if err != nil || len(scope) != 32 || hex.EncodeToString(scope) != cfg.ScopeFingerprint || strings.TrimSpace(cfg.HostInstance) == "" || len(cfg.HostInstance) > 256 || !utf8.ValidString(cfg.HostInstance) {
		return cfg, failure("invalid_request", "exact host instance and scope fingerprint required")
	}
	if len(cfg.Kinds) == 0 {
		cfg.Kinds = []persona.Kind{persona.KindIdentity, persona.KindRelationship, persona.KindRedline, persona.KindUser}
	}
	seen := map[persona.Kind]bool{}
	for _, k := range cfg.Kinds {
		if k.String() == "unknown" || k == persona.KindWorld || seen[k] {
			return cfg, failure("invalid_request", "invalid or tool-only native source")
		}
		seen[k] = true
	}
	cfg.Kinds = append([]persona.Kind{}, cfg.Kinds...)
	slices.Sort(cfg.Kinds)
	return cfg, nil
}
func nativeFingerprint(b registry.HostBinding, cfg NativeBindingConfig) string {
	// Enabling after an owner preview is permitted; disabling invalidates under s.mu.
	b.Projection = nil
	b.Controls = registry.HostControls{}
	raw, _ := json.Marshal(struct {
		Binding registry.HostBinding
		Config  NativeBindingConfig
	}{b, cfg})
	return hash(string(raw))
}
func (s *Service) invalidatePreview(id string) { delete(s.previews, id) }
func (s *Service) SetControls(ctx context.Context, id string, c registry.HostControls) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return failure("unavailable", "host service closed")
	}
	b, err := s.registry.Binding(ctx, id)
	if err != nil {
		return err
	}
	if b.Revoked {
		return failure("binding_revoked", "host binding revoked")
	}
	if b.Controls.PersonaSync && !c.PersonaSync {
		s.invalidatePreview(id)
	}
	return s.registry.SetHostControls(ctx, id, c)
}
func (s *Service) ConfigureNative(ctx context.Context, id string, cfg NativeBindingConfig) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.binding(ctx, id, false); err != nil {
		return HostStatus{}, err
	}
	cfg, err := normalizeNativeConfig(cfg)
	if err != nil {
		return HostStatus{}, err
	}
	v, err := s.state(ctx, id)
	if err != nil {
		return HostStatus{}, err
	}
	if v.Path != "" || v.ManualAction != "" || v.WrittenHash != "" || v.PlannedHash != "" {
		return v.HostStatus, failure("conflict", "complete existing file cleanup or use a new binding")
	}
	s.invalidatePreview(id)
	v.Native = &cfg
	v.Mode = nativeMode
	v.AdapterVersion = cfg.AdapterVersion
	v.PendingRetry = true
	v.PendingReason = "activation"
	v.WaitingHost = false
	v.State = "inactive"
	v.FirstBlockedAt = ptrTime(s.now())
	v.RequestedRevisions = nil
	v.LoadedHash = ""
	v.LoadedRevisions = nil
	v.SessionID = ""
	v.OfferedHash = ""
	v.OfferedRevisions = nil
	v.LastHostSeenAt = nil
	return v.HostStatus, s.save(ctx, v)
}
func ptrTime(t time.Time) *time.Time { return &t }
func (s *Service) nativeBinding(ctx context.Context, id string, enabled bool) (registry.HostBinding, projectionState, error) {
	b, err := s.binding(ctx, id, enabled)
	if err != nil {
		return b, projectionState{}, err
	}
	v, err := s.state(ctx, id)
	if err != nil {
		return b, v, err
	}
	if v.Mode != nativeMode || v.Native == nil {
		return b, v, failure("unsupported_host", "binding has no native prompt configuration")
	}
	return b, v, nil
}
func (s *Service) snapshot(ctx context.Context, b registry.HostBinding, cfg NativeBindingConfig, accept func(persona.ProjectionSnapshot, PromptProjection) error) error {
	authority, err := s.persona(ctx, b.ProfileID)
	if err != nil {
		return err
	}
	return authority.WithProjectionSnapshot(cfg.Kinds, func(snapshot persona.ProjectionSnapshot) error {
		current, err := s.binding(ctx, b.BindingID, false)
		if err != nil {
			return err
		}
		state, err := s.state(ctx, b.BindingID)
		if err != nil {
			return err
		}
		if state.Native == nil || nativeFingerprint(current, *state.Native) != nativeFingerprint(b, cfg) {
			return failure("stale_preview", "binding configuration changed")
		}
		projection, err := renderNative(b.BindingID, cfg, snapshot)
		if err != nil {
			return err
		}
		return accept(snapshot, projection)
	})
}
func renderNative(id string, cfg NativeBindingConfig, snapshot persona.ProjectionSnapshot) (PromptProjection, error) {
	var body strings.Builder
	fmt.Fprintf(&body, "[Garden Persona %s]\n", nativeAdapterVersion)
	revisions := map[string]uint64{}
	for _, k := range cfg.Kinds {
		doc := snapshot.Documents[k]
		if !doc.Exists || !doc.Valid {
			return PromptProjection{}, failure("unavailable", "selected Persona document unavailable")
		}
		fmt.Fprintf(&body, "## %s (revision %d)\n%s\n", k.FileName(), doc.Revision, doc.Content)
		revisions[k.String()] = doc.Revision
	}
	body.WriteString("[/Garden Persona]\n")
	if utf8.RuneCountInString(body.String()) > 6000 {
		return PromptProjection{}, failure("invalid_request", "native Persona projection exceeds 6000 characters")
	}
	return PromptProjection{BindingID: id, SourceRevisions: revisions, Hash: hash(body.String()), Markdown: body.String()}, nil
}
func (s *Service) markPending(ctx context.Context, v projectionState, profile, reason string) error {
	v.PendingRetry = true
	v.PendingReason = reason
	v.WaitingHost = false
	v.State = "pending_retry"
	s.invalidatePreview(v.BindingID)
	if v.FirstBlockedAt == nil {
		v.FirstBlockedAt = ptrTime(s.now())
	}
	v.RequestedRevisions = map[string]uint64{}
	if p, err := s.persona(ctx, profile); err == nil {
		for _, k := range v.Native.Kinds {
			if d, e := p.GetDocument(k); e == nil {
				v.RequestedRevisions[k.String()] = d.Revision
			}
		}
	}
	if err := s.save(ctx, v); err != nil {
		return failure("unavailable", "cannot persist native retry intent")
	}
	return nil
}
func retryReason(err error) string {
	switch persona.CodeOf(err) {
	case "persona_edit_locked":
		return "editing"
	case "persona_authority_drift":
		return "authority_drift"
	}
	return "stale_preview"
}
func (s *Service) RetryPreview(ctx context.Context, id string) (RetryPreview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, v, err := s.nativeBinding(ctx, id, false)
	if err != nil {
		return RetryPreview{}, err
	}
	var out RetryPreview
	err = s.snapshot(ctx, b, *v.Native, func(snapshot persona.ProjectionSnapshot, p PromptProjection) error {
		secret := make([]byte, 24)
		if _, err := rand.Read(secret); err != nil {
			return err
		}
		out = RetryPreview{ID: hex.EncodeToString(secret), Projection: p, Mode: nativeMode, ScopeFingerprint: v.Native.ScopeFingerprint, ExpiresAt: s.now().Add(60 * time.Second)}
		hashes := map[string]string{}
		for k, d := range snapshot.Documents {
			hashes[k.String()] = d.ContentHash
		}
		s.previews[id] = &nativePreview{RetryPreview: out, fingerprint: nativeFingerprint(b, *v.Native), hashes: hashes, generations: snapshot.EditGenerations}
		return nil
	})
	if err != nil {
		return RetryPreview{}, err
	}
	return out, nil
}
func (s *Service) previewMatches(preview *nativePreview, b registry.HostBinding, cfg NativeBindingConfig, snapshot persona.ProjectionSnapshot, p PromptProjection) bool {
	if preview == nil || !s.now().Before(preview.ExpiresAt) || preview.fingerprint != nativeFingerprint(b, cfg) || preview.Projection.Hash != p.Hash || !reflect.DeepEqual(preview.Projection.SourceRevisions, p.SourceRevisions) || !reflect.DeepEqual(preview.generations, snapshot.EditGenerations) {
		return false
	}
	for k, d := range snapshot.Documents {
		if preview.hashes[k.String()] != d.ContentHash {
			return false
		}
	}
	return true
}
func (s *Service) ConfirmRetry(ctx context.Context, id, previewID string) (HostStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, v, err := s.nativeBinding(ctx, id, false)
	if err != nil {
		return HostStatus{}, err
	}
	preview := s.previews[id]
	if preview == nil || preview.ID != previewID {
		return v.HostStatus, failure("stale_preview", "retry preview is unavailable or superseded")
	}
	if !s.now().Before(preview.ExpiresAt) {
		s.invalidatePreview(id)
		return v.HostStatus, failure("stale_preview", "retry preview expired")
	}
	err = s.snapshot(ctx, b, *v.Native, func(snapshot persona.ProjectionSnapshot, p PromptProjection) error {
		if !s.previewMatches(preview, b, *v.Native, snapshot, p) {
			return failure("stale_preview", "Persona or edit generation changed after preview")
		}
		v.PendingRetry = true
		v.WaitingHost = true
		v.State = "waiting_host"
		if err := s.save(ctx, v); err != nil {
			return failure("unavailable", "cannot persist retry admission")
		}
		preview.armed = true
		return nil
	})
	if err != nil {
		if e := s.markPending(ctx, v, b.ProfileID, retryReason(err)); e != nil {
			return HostStatus{}, e
		}
		latest, _ := s.state(ctx, id)
		return latest.HostStatus, failure("stale_preview", "retry requires a fresh preview")
	}
	return v.HostStatus, nil
}
func (s *Service) PreparePrompt(ctx context.Context, id string, req NativePromptRequest) (PromptProjection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, v, err := s.nativeBinding(ctx, id, true)
	if err != nil {
		return PromptProjection{}, err
	}
	cfg := *v.Native
	if req.SessionID == "" || req.AdapterVersion != cfg.AdapterVersion || req.HostVersion != cfg.HostVersion || req.HostInstance != cfg.HostInstance || req.ScopeFingerprint != cfg.ScopeFingerprint {
		return PromptProjection{}, failure("invalid_binding", "native host version, scope or session mismatch")
	}
	if _, err = s.registry.Resolve(ctx, registry.AccessRequest{BindingID: id, SessionID: req.SessionID, Operation: "persona_get"}); err != nil {
		return PromptProjection{}, err
	}
	var out PromptProjection
	err = s.snapshot(ctx, b, cfg, func(snapshot persona.ProjectionSnapshot, p PromptProjection) error {
		if _, err := s.registry.Resolve(ctx, registry.AccessRequest{BindingID: id, SessionID: req.SessionID, Operation: "persona_get"}); err != nil {
			return err
		}
		if v.PendingRetry {
			preview := s.previews[id]
			if preview == nil || !preview.armed {
				return failure("retry_required", "owner must preview and confirm a fresh retry")
			}
			if !s.previewMatches(preview, b, cfg, snapshot, p) {
				return failure("stale_preview", "armed preview changed before host pull")
			}
		}
		v.PendingRetry = false
		v.PendingReason = ""
		v.FirstBlockedAt = nil
		v.RequestedRevisions = nil
		v.WaitingHost = false
		v.SavedRevisions = p.SourceRevisions
		v.OfferedRevisions = p.SourceRevisions
		v.OfferedHash = p.Hash
		v.LastHostSeenAt = ptrTime(s.now())
		v.State = "offered"
		v.SessionID = req.SessionID
		if err := s.save(ctx, v); err != nil {
			return failure("unavailable", "cannot persist native offered state")
		}
		s.invalidatePreview(id)
		out = p
		return nil
	})
	if err != nil {
		if persona.CodeOf(err) == "persona_edit_locked" || persona.CodeOf(err) == "persona_authority_drift" || accessCode(err) == "stale_preview" {
			if e := s.markPending(ctx, v, b.ProfileID, retryReason(err)); e != nil {
				return PromptProjection{}, e
			}
		}
		return PromptProjection{}, err
	}
	return out, nil
}
func accessCode(err error) string {
	if e, ok := err.(*registry.AccessError); ok {
		return e.Code
	}
	return ""
}
