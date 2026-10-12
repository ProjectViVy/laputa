package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixtureConfig(t *testing.T, endpoint string) Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", dir)
	credential := filepath.Join(dir, "credential")
	if err := os.WriteFile(credential, []byte("synthetic-private-token"), 0600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "codex")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf 'codex-cli 0.162.1\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(cwd))
	return Config{Endpoint: endpoint, BindingID: "binding", CredentialFile: credential, HostKind: "codex-cli", HostVersion: "0.162.1", HostExecutable: executable, HostHome: dir, HostInstance: "existing-native-home", ScopeFingerprint: hex.EncodeToString(hash[:]), CWD: cwd}
}

func eventJSON(cfg Config, event string) string {
	data, _ := json.Marshal(map[string]any{"session_id": "native-session", "cwd": cfg.CWD, "hook_event_name": event, "source": "resume", "prompt": "private human text", "transcript_path": filepath.Join(cfg.HostHome, "private-transcript")})
	return string(data)
}

func TestHostContextEnvelope(t *testing.T) {
	for _, event := range []string{"UserPromptSubmit", "SessionStart"} {
		t.Run(event, func(t *testing.T) {
			calls := 0
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v2/host-bindings/binding/projection/prompt" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic-private-token" || r.Header.Get("X-Garden-Session") != "native-session" {
					t.Error("wrong bound request")
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if len(body) != 5 || body["session_id"] != "native-session" || body["adapter_version"] != "garden-native-prompt/v1" || body["host_version"] != "0.162.1" || body["host_instance"] != "existing-native-home" || body["scope_fingerprint"] == "" {
					t.Error("wrong native metadata")
				}
				json.NewEncoder(w).Encode(map[string]any{"binding_id": "binding", "markdown": "fresh Persona", "hash": "sha256:fixture", "source_revisions": map[string]uint64{"identity": 2}})
			}))
			defer backend.Close()
			cfg := fixtureConfig(t, backend.URL)
			var out, diagnostics bytes.Buffer
			if err := run(context.Background(), strings.NewReader(eventJSON(cfg, event)), &out, &diagnostics, cfg, backend.Client()); err != nil {
				t.Fatal(err)
			}
			var got struct {
				HookSpecificOutput struct {
					HookEventName     string
					AdditionalContext string
				}
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.HookSpecificOutput.HookEventName != event || got.HookSpecificOutput.AdditionalContext != "fresh Persona" || calls != 1 || diagnostics.Len() != 0 {
				t.Fatalf("invalid contribution: %s %s calls%d", out.String(), diagnostics.String(), calls)
			}
		})
	}
}

func TestHostContextScope(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer backend.Close()
	cfg := fixtureConfig(t, backend.URL)
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		raw    string
	}{
		{name: "unknown", mutate: func(v map[string]any) { v["hook_event_name"] = "SubagentStart" }},
		{name: "empty session", mutate: func(v map[string]any) { v["session_id"] = "" }},
		{name: "worker", mutate: func(v map[string]any) { v["agent_id"] = "child" }},
		{name: "parent", mutate: func(v map[string]any) { v["parent_session_id"] = "parent" }},
		{name: "ephemeral", mutate: func(v map[string]any) { v["transcript_path"] = nil }},
		{name: "other home", mutate: func(v map[string]any) { v["transcript_path"] = filepath.Join(os.TempDir(), "foreign-transcript") }},
		{name: "fork", mutate: func(v map[string]any) { v["source"] = "fork" }},
		{name: "wrong cwd", mutate: func(v map[string]any) { v["cwd"] = os.TempDir() }},
		{name: "malformed", raw: "{"}, {name: "oversize", raw: strings.Repeat("x", 65537)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.raw
			if input == "" {
				var body map[string]any
				json.Unmarshal([]byte(eventJSON(cfg, "SessionStart")), &body)
				tc.mutate(body)
				raw, _ := json.Marshal(body)
				input = string(raw)
			}
			var out, diag bytes.Buffer
			if err := run(context.Background(), strings.NewReader(input), &out, &diag, cfg, backend.Client()); err != nil {
				t.Fatal(err)
			}
			if out.Len() != 0 || strings.Contains(diag.String(), "private") || strings.Contains(diag.String(), "synthetic-private-token") {
				t.Fatalf("unsafe omission %q %q", out.String(), diag.String())
			}
		})
	}
	if calls != 0 {
		t.Fatalf("invalid event contacted Garden %d times", calls)
	}
	if err := os.WriteFile(cfg.HostExecutable, []byte("#!/bin/sh\nprintf 'codex-cli 0.162.0\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	run(context.Background(), strings.NewReader(eventJSON(cfg, "UserPromptSubmit")), &out, &diag, cfg, backend.Client())
	if calls != 0 || out.Len() != 0 {
		t.Fatal("configured version bypassed actual version")
	}
}

func TestHostContextUnavailableAndNoCache(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		delay  time.Duration
	}{
		{"forbidden", 403, "private response", 0}, {"pending", 409, "private response", 0}, {"redirect", 302, "", 0},
		{"timeout", 200, "", 1200 * time.Millisecond}, {"oversize", 200, strings.Repeat("界", 6001), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tc.delay)
				w.Header().Set("Location", "http://127.0.0.1:1")
				w.WriteHeader(tc.status)
				json.NewEncoder(w).Encode(map[string]any{"binding_id": "binding", "markdown": tc.body, "hash": "fixture", "source_revisions": map[string]uint64{"identity": 2}})
			}))
			defer backend.Close()
			cfg := fixtureConfig(t, backend.URL)
			var out, diag bytes.Buffer
			start := time.Now()
			run(context.Background(), strings.NewReader(eventJSON(cfg, "UserPromptSubmit")), &out, &diag, cfg, backend.Client())
			if out.Len() != 0 || strings.Contains(diag.String(), "private") || strings.Contains(diag.String(), "synthetic-private-token") || time.Since(start) > 1500*time.Millisecond {
				t.Fatalf("unsafe failure %q %q %s", out.String(), diag.String(), time.Since(start))
			}
		})
	}
}

func TestHostContextPipeDeadline(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	var out, diag bytes.Buffer
	start := time.Now()
	if err := run(ctx, reader, &out, &diag, Config{}, nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 || time.Since(start) > 200*time.Millisecond {
		t.Fatal("native stdin escaped deadline")
	}
}

func TestHostContextPrivateCredential(t *testing.T) {
	cfg := fixtureConfig(t, "http://127.0.0.1:1")
	if _, err := privateRead(cfg.CredentialFile, 4096); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(cfg.CredentialFile), "link")
	if err := os.Symlink(cfg.CredentialFile, link); err != nil {
		t.Fatal(err)
	}
	if _, err := privateRead(link, 4096); err == nil {
		t.Fatal("credential symlink accepted")
	}
	if err := os.Chmod(cfg.CredentialFile, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := privateRead(cfg.CredentialFile, 4096); err == nil {
		t.Fatal("public credential accepted")
	}
}

func TestHostContextInheritedPipe(t *testing.T) {
	if path := os.Getenv("GARDEN_TEST_HELPER_CONFIG"); path != "" {
		flag.CommandLine = flag.NewFlagSet("bridge", flag.ContinueOnError)
		os.Args = []string{"bridge", "--config", path}
		main()
		os.Exit(0)
	}
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{"binding_id": "binding", "markdown": "fresh Persona", "hash": "fixture", "source_revisions": map[string]uint64{"identity": 2}})
	}))
	defer backend.Close()
	cfg := fixtureConfig(t, backend.URL)
	path := filepath.Join(filepath.Dir(cfg.CredentialFile), "config.json")
	raw, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, incomplete := range []bool{false, true} {
		cmd := exec.Command(binary, "-test.run=^TestHostContextInheritedPipe$")
		cmd.Env = append(os.Environ(), "GARDEN_TEST_HELPER_CONFIG="+path)
		pipe, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		start := time.Now()
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		if incomplete {
			io.WriteString(pipe, "{")
		} else {
			io.WriteString(pipe, eventJSON(cfg, "UserPromptSubmit"))
			pipe.Close()
		}
		if err := cmd.Wait(); err != nil {
			t.Fatalf("child: %v %s", err, diag.String())
		}
		pipe.Close()
		if incomplete {
			if out.Len() != 0 || time.Since(start) > 1700*time.Millisecond {
				t.Fatal("inherited incomplete pipe exceeded deadline")
			}
		} else if !strings.Contains(out.String(), "fresh Persona") {
			t.Fatalf("inherited valid pipe failed: %s %s", out.String(), diag.String())
		}
	}
	if calls != 1 {
		t.Fatalf("Garden calls%d", calls)
	}
}

func TestHostContextVersionOutputBound(t *testing.T) {
	cfg := fixtureConfig(t, "http://127.0.0.1:1")
	// exec's io.Copy must not find an embedded bytes.Buffer.ReadFrom and bypass
	// the version writer cap, even when the trimmed text is the exact version.
	if err := os.WriteFile(cfg.HostExecutable, []byte("#!/bin/sh\nprintf '%20000s' 'codex-cli 0.162.1'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	run(context.Background(), strings.NewReader(eventJSON(cfg, "UserPromptSubmit")), &out, &diag, cfg, nil)
	if out.Len() != 0 || !strings.Contains(diag.String(), "unsupported_host") {
		t.Fatal("oversize actual version output accepted")
	}
}
