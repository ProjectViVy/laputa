// An independent Go host consumer of Garden's public in-process Agent API.
// No Garden internal package, HTTP listener, model download, or Vivy code is used.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dashimaki/garden/agentapi"
	"github.com/dashimaki/laputa/persona"
)

func main() {
	if err := smoke(); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("PASS: external consumer; offline pure-Go open/bind/bootstrap/recall/explicit WORLD/capture/replay/restart/close")
}

func smoke() error {
	ctx := context.Background()
	root, err := os.MkdirTemp("", "vivy-embed-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	cfg := agentapi.Config{
		PersonaDir: filepath.Join(root, "persona"), PalacePath: filepath.Join(root, "palace"),
		ModelsDir: filepath.Join(root, "missing-models"), StateDB: filepath.Join(root, "state.db"),
		ProfileID: "default", AgentID: "vivy-smoke", Platform: "vivy", Principal: agentapi.PrincipalAgent,
	}
	// Operator setup uses Laputa's public authority API, not a Garden private path.
	authority, err := persona.Open(cfg.PersonaDir)
	if err != nil {
		return fmt.Errorf("open authority: %w", err)
	}
	const identity = "smoke identity"
	const secret = "world-only sentinel"
	_, err = authority.Initialize(persona.Initialization{
		Identity: identity, Relationship: "smoke relationship", Redline: "smoke redline",
		User: "smoke user", World: secret,
	}, "user", persona.SourceInit, "smoke setup")
	if err != nil {
		return fmt.Errorf("initialize authority: %w", err)
	}
	client, err := agentapi.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open embedded client: %w", err)
	}
	bound, err := client.BindSession("host-session-1")
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("bind host session: %w", err)
	}
	bootstrap, err := bound.Bootstrap(ctx, agentapi.BootstrapRequest{Intent: "smoke", BudgetChars: 1024})
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("bootstrap: %w", err)
	}
	if bootstrap.FrozenCore.SessionID != "host-session-1" || !strings.Contains(bootstrap.Context, identity) || strings.Contains(bootstrap.Context, secret) || len(bootstrap.Evidence) != 0 {
		_ = client.Close()
		return fmt.Errorf("bootstrap leaked tool-only data or failed frozen-only projection: %+v", bootstrap)
	}
	view, err := bound.FastRecall(ctx, agentapi.FastRecallRequest{Query: "smoke", BudgetChars: 1024})
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("fast recall: %w", err)
	}
	if !strings.Contains(view.Context, identity) || strings.Contains(view.Context, secret) {
		_ = client.Close()
		return fmt.Errorf("fast recall leaked tool-only WORLD or lost frozen identity: %+v", view)
	}
	world, err := bound.ReadPersona(ctx, "WORLD")
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("explicit WORLD read: %w", err)
	}
	if !strings.Contains(world.Content, secret) {
		_ = client.Close()
		return fmt.Errorf("explicit WORLD read lost sentinel: %+v", world)
	}
	if _, err := bound.SearchCards(ctx, agentapi.CardSearch{Query: "smoke"}); !codeIs(err, "unavailable") {
		_ = client.Close()
		return fmt.Errorf("missing offline model should degrade card search: %v", err)
	}
	if _, err := bound.Bootstrap(ctx, agentapi.BootstrapRequest{Binding: agentapi.Binding{ProfileID: "other"}}); !codeIs(err, "invalid_binding") {
		_ = client.Close()
		return fmt.Errorf("conflicting binding was not rejected: %v", err)
	}
	content := "durable terminal smoke content"
	digest := sha256.Sum256([]byte(content))
	req := agentapi.CaptureRequest{
		Phase: agentapi.CaptureCompleted, Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest),
		Provenance: agentapi.CaptureProvenance{RunID: "host-run-1", EventSeq: 1},
	}
	if _, err := bound.Capture(ctx, agentapi.CaptureRequest{Phase: "running"}); !codeIs(err, "invalid_request") {
		_ = client.Close()
		return fmt.Errorf("nonterminal capture was not rejected: %v", err)
	}
	first, err := bound.Capture(ctx, req)
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("terminal capture: %w", err)
	}
	second, err := bound.Capture(ctx, req)
	if err != nil || second.IngestionID != first.IngestionID || second.EventID != first.EventID {
		_ = client.Close()
		return fmt.Errorf("idempotent replay: first=%+v second=%+v err=%v", first, second, err)
	}
	if _, err := bound.CaptureStatus(ctx, first.IngestionID, first.EventID); err != nil {
		_ = client.Close()
		return fmt.Errorf("capture status: %w", err)
	}
	if err := client.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	if _, err := bound.Bootstrap(ctx, agentapi.BootstrapRequest{}); !codeIs(err, "unavailable") {
		return fmt.Errorf("closed handle remained usable: %v", err)
	}
	client, err = agentapi.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("reopen persisted runtime: %w", err)
	}
	defer client.Close()
	bound, err = client.BindSession("host-session-1")
	if err != nil {
		return fmt.Errorf("rebind persisted session: %w", err)
	}
	status, err := bound.CaptureStatus(ctx, first.IngestionID, first.EventID)
	if err != nil || status.IngestionID != first.IngestionID {
		return fmt.Errorf("durable status after restart: %+v %v", status, err)
	}
	return nil
}

func codeIs(err error, code string) bool {
	var apiErr *agentapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == code
}
