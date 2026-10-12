// garden-host-context supplies additive Persona through opt-in native command
// hooks. It writes neither host instructions nor host configuration/history.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/ProjectViVy/laputa/garden/internal/hosts"
)

type Config struct {
	Endpoint         string `json:"endpoint"`
	BindingID        string `json:"binding_id"`
	CredentialFile   string `json:"credential_file"`
	HostKind         string `json:"host_kind"`
	HostVersion      string `json:"host_version"`
	HostExecutable   string `json:"host_executable"`
	HostHome         string `json:"host_home"`
	HostInstance     string `json:"host_instance"`
	ScopeFingerprint string `json:"scope_fingerprint"`
	CWD              string `json:"cwd"`
}

type hookEvent struct {
	SessionID       string `json:"session_id"`
	CWD             string `json:"cwd"`
	Name            string `json:"hook_event_name"`
	Source          string `json:"source"`
	AgentID         string `json:"agent_id"`
	AgentType       string `json:"agent_type"`
	ParentSessionID string `json:"parent_session_id"`
	TranscriptPath  string `json:"transcript_path"`
}

func main() {
	path := flag.String("config", "", "absolute Garden-owned private JSON config")
	flag.Parse()
	raw, err := privateRead(*path, 64*1024)
	var cfg Config
	if err != nil || json.Unmarshal(raw, &cfg) != nil {
		fmt.Fprintln(os.Stderr, "Garden Persona omitted: unavailable_config")
		return
	}
	// Native exec supplies an inherited blocking pipe. Register our own
	// nonblocking descriptor with Go's poller so its read deadline is real.
	fd, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Garden Persona omitted: unavailable_input")
		return
	}
	if err = syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		fmt.Fprintln(os.Stderr, "Garden Persona omitted: unavailable_input")
		return
	}
	input := os.NewFile(uintptr(fd), "native-hook-input")
	defer input.Close()
	_ = run(context.Background(), input, os.Stdout, os.Stderr, cfg, nil)
}

// Expected unavailable/invalid hooks return successful inert output so native
// work continues. Only an output-writer failure is returned to the caller.
func run(ctx context.Context, in io.Reader, out, diagnostics io.Writer, cfg Config, client *http.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	omit := func(code string) error { fmt.Fprintln(diagnostics, "Garden Persona omitted: "+code); return nil }
	if file, ok := in.(*os.File); ok {
		deadline, _ := ctx.Deadline()
		if err := file.SetReadDeadline(deadline); err != nil {
			info, statErr := file.Stat()
			if statErr != nil || !info.Mode().IsRegular() {
				return omit("unavailable_input")
			}
		} else {
			defer file.SetReadDeadline(time.Time{})
		}
	}
	raw, err := io.ReadAll(io.LimitReader(in, 64*1024+1))
	var event hookEvent
	if err != nil || len(raw) > 64*1024 || !utf8.Valid(raw) || json.Unmarshal(raw, &event) != nil {
		return omit("invalid_event")
	}
	if event.Name != "UserPromptSubmit" && event.Name != "SessionStart" || !validSession(event.SessionID) || event.AgentID != "" || event.AgentType != "" || event.ParentSessionID != "" {
		return omit("unsupported_scope")
	}
	// Reject fork-start and identified worker events. Later root-fork prompt
	// events have no fork metadata in the native schema and remain root turns.
	if event.Source == "fork" || event.Name == "SessionStart" && event.Source != "startup" && event.Source != "resume" && event.Source != "clear" && event.Source != "compact" {
		return omit("unsupported_scope")
	}
	cwd, err := canonicalDir(cfg.CWD)
	actual, actualErr := os.Getwd()
	actual, actualErr = canonicalDirWithError(actual, actualErr)
	eventCWD, eventErr := canonicalDir(event.CWD)
	hash := sha256.Sum256([]byte(cwd))
	if err != nil || actualErr != nil || eventErr != nil || cwd != actual || cwd != eventCWD || cfg.ScopeFingerprint != hex.EncodeToString(hash[:]) {
		return omit("unsupported_scope")
	}
	versions := map[string]string{"codex-cli": "0.162.1", "claude-code": "2.1.296"}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || versions[cfg.HostKind] == "" || cfg.HostVersion != versions[cfg.HostKind] || !filepath.IsAbs(cfg.HostExecutable) || cfg.HostInstance == "" || len(cfg.HostInstance) > 256 || !utf8.ValidString(cfg.HostInstance) || cfg.BindingID == "" {
		return omit("unsupported_host")
	}
	nativeHome := os.Getenv("CODEX_HOME")
	defaultHome := ".codex"
	if cfg.HostKind == "claude-code" {
		nativeHome = os.Getenv("CLAUDE_CONFIG_DIR")
		defaultHome = ".claude"
	}
	if nativeHome == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return omit("unsupported_scope")
		}
		nativeHome = filepath.Join(home, defaultHome)
	}
	approvedHome, homeErr := canonicalDir(cfg.HostHome)
	actualHome, actualHomeErr := canonicalDir(nativeHome)
	transcriptDir, transcriptErr := canonicalDir(filepath.Dir(event.TranscriptPath))
	if homeErr != nil || actualHomeErr != nil || transcriptErr != nil || approvedHome != actualHome || !filepath.IsAbs(event.TranscriptPath) || !strings.HasPrefix(transcriptDir+string(os.PathSeparator), approvedHome+string(os.PathSeparator)) {
		return omit("unsupported_scope")
	}
	// Claude2.1.296 exposes the same not-yet-created transcript path in fresh
	// persistent and --no-session-persistence hooks. Check the owning native
	// process; file existence would reject the first valid persistent turn.
	if cfg.HostKind == "claude-code" && !persistentClaude(ctx, cfg.HostExecutable, event.SessionID) {
		return omit("unsupported_scope")
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme != "http" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Path != "" && endpoint.Path != "/" || endpoint.Hostname() != "127.0.0.1" && endpoint.Hostname() != "::1" && endpoint.Hostname() != "localhost" {
		return omit("unsupported_endpoint")
	}
	versionCtx, versionCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	cmd := exec.CommandContext(versionCtx, cfg.HostExecutable, "--version")
	cmd.WaitDelay = 100 * time.Millisecond
	var version boundedOutput
	cmd.Stdout = &version
	cmd.Stderr = io.Discard
	err = cmd.Run()
	versionCancel()
	expected := "codex-cli " + cfg.HostVersion
	if cfg.HostKind == "claude-code" {
		expected = cfg.HostVersion + " (Claude Code)"
	}
	if err != nil || strings.TrimSpace(version.String()) != expected {
		return omit("unsupported_host")
	}
	token, err := privateRead(cfg.CredentialFile, 4096)
	credential := strings.TrimSpace(string(token))
	if err != nil || credential == "" || len(credential) > 512 || strings.ContainsAny(credential, "\r\n\x00") {
		return omit("unavailable_credential")
	}
	body, _ := json.Marshal(hosts.NativePromptRequest{AdapterVersion: "garden-native-prompt/v1", HostVersion: cfg.HostVersion, HostInstance: cfg.HostInstance, SessionID: event.SessionID, ScopeFingerprint: cfg.ScopeFingerprint})
	requestCtx, requestCancel := context.WithTimeout(ctx, time.Second)
	defer requestCancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, strings.TrimRight(cfg.Endpoint, "/")+"/v2/host-bindings/"+url.PathEscape(cfg.BindingID)+"/projection/prompt", bytes.NewReader(body))
	if err != nil {
		return omit("unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+credential)
	request.Header.Set("X-Garden-Session", event.SessionID)
	var httpClient http.Client
	if client != nil {
		httpClient = *client
	} else {
		// Numeric loopback transport avoids proxy routing and name-service waits.
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if host == "localhost" {
				host = "127.0.0.1"
			}
			return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(host, port))
		}
		transport.DisableKeepAlives = true
		defer transport.CloseIdleConnections()
		httpClient.Transport = transport
	}
	httpClient.Timeout = time.Second
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := httpClient.Do(request)
	if err != nil {
		return omit("unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return omit("unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 96*1024+1))
	var projection hosts.PromptProjection
	if err != nil || len(data) > 96*1024 || !utf8.Valid(data) || json.Unmarshal(data, &projection) != nil || projection.BindingID != cfg.BindingID || projection.Markdown == "" || utf8.RuneCountInString(projection.Markdown) > 6000 || projection.Hash == "" || len(projection.SourceRevisions) == 0 || ctx.Err() != nil {
		return omit("invalid_projection")
	}
	result := struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}{}
	result.HookSpecificOutput.HookEventName = event.Name
	result.HookSpecificOutput.AdditionalContext = projection.Markdown
	return json.NewEncoder(out).Encode(result)
}

func validSession(value string) bool {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func persistentClaude(ctx context.Context, executable, session string) bool {
	nativePID, err := strconv.Atoi(os.Getenv("CLAUDE_PID"))
	if err != nil || nativePID <= 1 || os.Getenv("CLAUDE_CODE_SESSION_ID") != session {
		return false
	}
	expected, err := os.Stat(executable)
	if err != nil {
		return false
	}
	pid := os.Getppid()
	for depth := 0; depth < 8 && pid > 1 && ctx.Err() == nil; depth++ {
		base := "/proc/" + strconv.Itoa(pid)
		info, err := os.Stat(base)
		if err != nil || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
			return false
		}
		if pid == nativePID {
			actual, err := os.Stat(base + "/exe")
			if err != nil || !os.SameFile(expected, actual) {
				return false
			}
			args, err := procBytes(base+"/cmdline", 64*1024)
			if err != nil || ctx.Err() != nil {
				return false
			}
			for _, arg := range bytes.Split(args, []byte{0}) {
				if string(arg) == "--no-session-persistence" {
					return false
				}
			}
			return len(args) > 0
		}
		status, err := procBytes(base+"/stat", 4096)
		if err != nil {
			return false
		}
		end := bytes.LastIndexByte(status, ')')
		if end < 0 {
			return false
		}
		fields := strings.Fields(string(status[end+1:]))
		if len(fields) < 2 {
			return false
		}
		pid, err = strconv.Atoi(fields[1])
		if err != nil {
			return false
		}
	}
	return false
}

func procBytes(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("native process metadata unavailable")
	}
	return raw, nil
}

func canonicalDirWithError(path string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return canonicalDir(path)
}
func canonicalDir(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("absolute directory required")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("directory required")
	}
	return resolved, nil
}

func privateRead(path string, limit int64) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("absolute private file required")
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if dir.Mode().Perm()&0077 != 0 || dir.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
		return nil, errors.New("private owner directory required")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) || info.Size() > limit {
		return nil, errors.New("private owner file required")
	}
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("private file unavailable")
	}
	return raw, nil
}

type boundedOutput struct{ buffer bytes.Buffer }

func (b *boundedOutput) String() string { return b.buffer.String() }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > 4096 {
		return 0, errors.New("version output too large")
	}
	return b.buffer.Write(p)
}
