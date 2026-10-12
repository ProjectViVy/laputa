package hosts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProjectViVy/laputa/garden/internal/registry"
	"github.com/ProjectViVy/laputa/laputa/persona"
)

func nativeFixture(t *testing.T) (*Service, string, *persona.Service, NativeBindingConfig, NativePromptRequest) {
	t.Helper()
	s, id, _ := hostFixture(t)
	ctx := context.Background()
	p, err := s.persona(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NativeBindingConfig{AdapterID: "openclaw", AdapterVersion: "garden-native-prompt/v1", HostVersion: "2026.10.2", HostInstance: "isolated-existing-host", ScopeFingerprint: strings.Repeat("a", 64)}
	req := NativePromptRequest{AdapterVersion: cfg.AdapterVersion, HostVersion: cfg.HostVersion, HostInstance: cfg.HostInstance, ScopeFingerprint: cfg.ScopeFingerprint, SessionID: "existing-session"}
	if _, err = s.ConfigureNative(ctx, id, cfg); err != nil {
		t.Fatal(err)
	}
	return s, id, p, cfg, req
}
func activateNative(t *testing.T, s *Service, id string) {
	t.Helper()
	ctx := context.Background()
	preview, err := s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetControls(ctx, id, registry.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	status, err := s.ConfirmRetry(ctx, id, preview.ID)
	if err != nil || !status.WaitingHost {
		t.Fatal(status, err)
	}
}
func TestNativePromptFreshRetry(t *testing.T) {
	s, id, p, _, req := nativeFixture(t)
	ctx := context.Background()
	activateNative(t, s, id)
	first, err := s.PreparePrompt(ctx, id, req)
	if err != nil || first.SourceRevisions["identity"] != 1 || strings.Contains(first.Markdown, "WORLD_SECRET") {
		t.Fatal(first, err)
	}
	doc, _ := p.GetDocument(persona.KindIdentity)
	lease, err := p.AcquireEdit(persona.KindIdentity, doc.Revision, doc.ContentHash, "editor")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		out, e := s.PreparePrompt(ctx, id, req)
		if persona.CodeOf(e) != "persona_edit_locked" || out.Markdown != "" {
			t.Fatal(out, e)
		}
	}
	status, err := s.Status(ctx, id)
	if err != nil || !status.PendingRetry || status.PendingReason != "editing" {
		t.Fatal(status, err)
	}
	_, err = p.SaveEditorDocument(persona.EditorWrite{Kind: persona.KindIdentity, Content: "N_PLUS_ONE", BaseRevision: doc.Revision, BaseHash: doc.ContentHash, Proof: persona.EditProof{Token: lease.Token, Generation: lease.Generation}})
	if err != nil {
		t.Fatal(err)
	}
	if out, e := s.PreparePrompt(ctx, id, req); e == nil || out.Markdown != "" {
		t.Fatal("pending replay", out, e)
	}
	old, err := s.RetryPreview(ctx, id)
	if err != nil || old.Projection.SourceRevisions["identity"] != 2 {
		t.Fatal(old, err)
	}
	_, err = p.SaveUserDocument(persona.KindIdentity, "N_PLUS_TWO", 2, "user", persona.SourceUserDirect, "newer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfirmRetry(ctx, id, old.ID); err == nil {
		t.Fatal("old preview confirmed")
	}
	fresh, err := s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	latest, _ := p.GetDocument(persona.KindIdentity)
	cycle, err := p.AcquireEdit(persona.KindIdentity, latest.Revision, latest.ContentHash, "edit-and-cancel")
	if err != nil {
		t.Fatal(err)
	}
	if err = p.ReleaseEdit(persona.KindIdentity, persona.EditProof{Token: cycle.Token, Generation: cycle.Generation}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfirmRetry(ctx, id, fresh.ID); err == nil {
		t.Fatal("edit/cancel did not invalidate preview")
	}
	fresh, err = s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfirmRetry(ctx, id, fresh.ID); err != nil {
		t.Fatal(err)
	}
	out, err := s.PreparePrompt(ctx, id, req)
	if err != nil || out.Hash != fresh.Projection.Hash || out.Markdown != fresh.Projection.Markdown || out.SourceRevisions["identity"] != 3 {
		t.Fatal(out, err)
	}
	status, err = s.Status(ctx, id)
	if err != nil || status.PendingRetry || status.WaitingHost || status.OfferedHash != out.Hash || status.WrittenHash != "" || status.LoadedHash != "" {
		t.Fatal(status, err)
	}
	raw, _ := json.Marshal(status)
	if strings.Contains(string(raw), "preview_id") || strings.Contains(string(raw), "N_PLUS_TWO") {
		t.Fatal("status leak", string(raw))
	}
	_, err = p.SaveUserDocument(persona.KindIdentity, "AUTOMATIC_N_PLUS_THREE", 3, "user", persona.SourceUserDirect, "later")
	if err != nil {
		t.Fatal(err)
	}
	out, err = s.PreparePrompt(ctx, id, req)
	if err != nil || out.SourceRevisions["identity"] != 4 {
		t.Fatal(out, err)
	}
}
func TestNativePromptLifecycle(t *testing.T) {
	s, id, p, cfg, req := nativeFixture(t)
	ctx := context.Background()
	activateNative(t, s, id)
	if _, err := s.PreparePrompt(ctx, id, req); err != nil {
		t.Fatal(err)
	}
	preview, err := s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetControls(ctx, id, registry.HostControls{}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetControls(ctx, id, registry.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfirmRetry(ctx, id, preview.ID); err == nil {
		t.Fatal("disable-enable revived preview")
	}
	if _, err = s.ConfigureNative(ctx, id, cfg); err != nil {
		t.Fatal(err)
	}
	preview, err = s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Now().Add(61 * time.Second) }
	if _, err = s.ConfirmRetry(ctx, id, preview.ID); err == nil {
		t.Fatal("expired preview")
	}
	s.now = time.Now
	// Restart drops previews but keeps the persisted gate.
	if _, err = s.ConfigureNative(ctx, id, cfg); err != nil {
		t.Fatal(err)
	}
	preview, err = s.RetryPreview(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfirmRetry(ctx, id, preview.ID); err != nil {
		t.Fatal(err)
	}
	restarted := reopenHost(t, s)
	if out, e := restarted.PreparePrompt(ctx, id, req); e == nil || out.Markdown != "" {
		t.Fatal("restart armed replay", out, e)
	}
	status, err := restarted.Status(ctx, id)
	if err != nil || !status.PendingRetry || status.WaitingHost {
		t.Fatal(status, err)
	}
	doc, _ := p.GetDocument(persona.KindIdentity)
	if err = os.WriteFile(filepath.Join(p.Root(), "IDENTITY.MD"), []byte("EXTERNAL_DRIFT"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, e := restarted.PreparePrompt(ctx, id, req); persona.CodeOf(e) != "persona_authority_drift" || out.Markdown != "" {
		t.Fatal("drift leaked", out, e)
	}
	if _, err = p.SaveUserDocument(persona.KindIdentity, "EXTERNAL_DRIFT", doc.Revision, "user", persona.SourceUserDirect, "inspected"); err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Preview(ctx, ProjectionRequest{BindingID: id, Path: filepath.Join(t.TempDir(), "native-file")}); err == nil {
		t.Fatal("native became file writer")
	}
	if _, err = restarted.Apply(ctx, FilePlan{BindingID: id, Path: filepath.Join(t.TempDir(), "native-file")}); err == nil {
		t.Fatal("file Apply bypass")
	}
	bad := req
	bad.ScopeFingerprint = strings.Repeat("b", 64)
	if _, err = restarted.PreparePrompt(ctx, id, bad); err == nil {
		t.Fatal("wrong native scope")
	}
	if _, err = restarted.Revoke(ctx, id); err != nil {
		t.Fatal(err)
	}
	if out, e := restarted.PreparePrompt(ctx, id, req); e == nil || out.Markdown != "" {
		t.Fatal("revoked output", out, e)
	}
}

func TestNativePromptRefusesOversizeAndMissingPermission(t *testing.T) {
	s, id, p, cfg, req := nativeFixture(t)
	ctx := context.Background()
	activateNative(t, s, id)
	_, err := p.SaveUserDocument(persona.KindIdentity, "a"+strings.Repeat(" ", 6001)+"b", 1, "user", persona.SourceUserDirect, "valid visible length, oversized rendered bytes")
	if err != nil {
		t.Fatal(err)
	}
	if out, e := s.PreparePrompt(ctx, id, req); e == nil || out.Markdown != "" {
		t.Fatal("oversize projection returned", out, e)
	}
	b, _, err := s.registry.CreateHostBinding(ctx, registry.HostBinding{ProfileID: "A", AgentID: "missing-op", Platform: "test", Operations: []string{"bootstrap"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConfigureNative(ctx, b.BindingID, cfg); err != nil {
		t.Fatal(err)
	}
	if err = s.SetControls(ctx, b.BindingID, registry.HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	if out, e := s.PreparePrompt(ctx, b.BindingID, req); accessCode(e) != "principal_forbidden" || out.Markdown != "" {
		t.Fatal("persona_get permission bypass", out, e)
	}
}
func TestNativePromptMetadataFailureReturnsNoBody(t *testing.T) {
	s, id, p, _, req := nativeFixture(t)
	ctx := context.Background()
	activateNative(t, s, id)
	// Fail actual SQLite metadata updates after reads/configuration succeeded.
	db, err := sql.Open("sqlite", filepath.Join(filepath.Dir(s.backupDir), "garden.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER fail_projection BEFORE UPDATE ON host_bindings BEGIN SELECT RAISE(FAIL, 'fixture metadata failure'); END`); err != nil {
		t.Fatal(err)
	}
	if out, e := s.PreparePrompt(ctx, id, req); accessCode(e) != "unavailable" || out.Markdown != "" {
		t.Fatal("offered before persistence", out, e)
	}
	doc, _ := p.GetDocument(persona.KindIdentity)
	if _, err = p.AcquireEdit(persona.KindIdentity, doc.Revision, doc.ContentHash, "locked"); err != nil {
		t.Fatal(err)
	}
	if out, e := s.PreparePrompt(ctx, id, req); accessCode(e) != "unavailable" || out.Markdown != "" {
		t.Fatal("pending persistence ignored", out, e)
	}
}
func TestNativePromptPreservesManualFileCleanup(t *testing.T) {
	s, id, path, plan := activeHost(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, plan); err != nil {
		t.Fatal(err)
	}
	cfg := NativeBindingConfig{AdapterID: "openclaw", AdapterVersion: nativeAdapterVersion, HostVersion: "2026.10.2", HostInstance: "fixture", ScopeFingerprint: strings.Repeat("a", 64)}
	if _, err := s.ConfigureNative(ctx, id, cfg); accessCode(err) != "conflict" {
		t.Fatal("file ownership lost", err)
	}
	v, err := s.state(ctx, id)
	if err != nil || v.Path != path || v.WrittenHash == "" {
		t.Fatal(v, err)
	}
}
