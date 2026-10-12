package agentapi

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// Polling/pull and explicit controls must make progress under the same owner.
func TestNativePromptConcurrentControls(t *testing.T) {
	ctx := context.Background()
	i, err := OpenInstallation(ctx, installationConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer i.Close()
	installPersona(t, i, "A")
	b, _, err := i.CreateHostBinding(ctx, HostBinding{ProfileID: "A", AgentID: "host", Platform: "openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := NativeBindingConfig{AdapterID: "openclaw", AdapterVersion: "garden-native-prompt/v1", HostVersion: "2026.10.2", HostInstance: "fixture", ScopeFingerprint: strings.Repeat("a", 64)}
	if _, err = i.ConfigureNativeHost(ctx, b.BindingID, cfg); err != nil {
		t.Fatal(err)
	}
	if err = i.SetHostControls(ctx, b.BindingID, HostControls{PersonaSync: true}); err != nil {
		t.Fatal(err)
	}
	preview, err := i.PreviewHostRetry(ctx, b.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = i.ConfirmHostRetry(ctx, b.BindingID, preview.ID); err != nil {
		t.Fatal(err)
	}
	req := NativePromptRequest{AdapterVersion: cfg.AdapterVersion, HostVersion: cfg.HostVersion, HostInstance: cfg.HostInstance, ScopeFingerprint: cfg.ScopeFingerprint, SessionID: "session"}
	start := make(chan struct{})
	failures := make(chan error, 3)
	var wg sync.WaitGroup
	for _, work := range []func() error{func() error { _, e := i.HostProjectionStatus(ctx, b.BindingID); return e }, func() error { _, e := i.PrepareHostPrompt(ctx, b.BindingID, req); return e }, func() error {
		return i.SetHostControls(ctx, b.BindingID, HostControls{PersonaSync: true, Recall: true})
	}} {
		wg.Add(1)
		go func(run func() error) {
			defer wg.Done()
			<-start
			for range 100 {
				if err := run(); err != nil {
					failures <- err
					return
				}
			}
		}(work)
	}
	close(start)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}
