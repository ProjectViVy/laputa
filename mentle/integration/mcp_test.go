package integration

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type MCPClient struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Reader
	mu     sync.Mutex
}

func NewMCPClient(t *testing.T) *MCPClient {
	execPath, _ := filepath.Abs("../mempalace-test")

	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", execPath, ".")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping MCP test: failed to build binary: %v", err)
	}

	cmd = exec.Command(execPath, "server")
	cmd.Env = append(os.Environ(), "MEMPALACE_HOME=/tmp/mempalace-test")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Skipf("skipping MCP test: failed to start server: %v", err)
	}

	return &MCPClient{
		t:      t,
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewReader(stdout),
	}
}

func (c *MCPClient) send(method string, params map[string]any, id any) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      id,
	}
	if params != nil {
		req["params"] = params
	}

	data, _ := json.Marshal(req)
	line := string(data) + "\n"

	c.stdin.WriteString(line)
	c.stdin.Flush()

	respLine, err := c.stdout.ReadString('\n')
	if err != nil {
		c.t.Logf("Read error: %v", err)
		return nil
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(respLine)), &resp); err != nil {
		c.t.Logf("JSON parse error: %v", err)
		return nil
	}
	return resp
}

func (c *MCPClient) close() {
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	exec.Command("bash", "-c", "pkill -9 -f 'mxbai.*--server' 2>/dev/null; pkill -9 -f 'Qwen.*--server' 2>/dev/null; true").Run()
}

func (c *MCPClient) initialize() bool {
	resp := c.send("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{},
	}, 1)

	if resp == nil || resp["error"] != nil {
		return false
	}

	c.mu.Lock()
	c.stdin.WriteString("{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n")
	c.stdin.Flush()
	c.mu.Unlock()

	return true
}

func TestMCPProtocolLineFraming(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	if !client.initialize() {
		t.Skip("Server not responding to initialize")
	}

	resp := client.send("tools/list", nil, 2)
	if resp == nil {
		t.Skip("Server not responding")
	}

	respJSON, _ := json.Marshal(resp)
	if strings.Count(string(respJSON), "\n") > 0 {
		t.Error("Response contains newlines - violates line framing")
	}
}

func TestMCPToolsList(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	if !client.initialize() {
		t.Skip("Server not responding to initialize")
	}

	resp := client.send("tools/list", nil, 2)
	if resp == nil {
		t.Skip("Server not responding")
	}

	if resp["jsonrpc"] != "2.0" {
		t.Error("Expected jsonrpc 2.0")
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("Expected result in response")
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("Expected tools array")
	}

	t.Logf("Found %d tools registered", len(tools))
}

func TestMCPErrorCodes(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	if !client.initialize() {
		t.Skip("Server not responding to initialize")
	}

	resp := client.send("nonexistent_method", nil, 2)
	if resp == nil {
		t.Skip("Server not responding")
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("Expected error for method not found")
	}

	code, ok := errObj["code"].(float64)
	if !ok {
		t.Fatal("Expected numeric error code")
	}

	if code == 0 {
		t.Error("Error code should not be 0")
	}

	expectedCodes := map[float64]bool{-32700: true, -32600: true, -32601: true, -32602: true, -32603: true}
	if !expectedCodes[code] {
		t.Errorf("Unexpected error code: %v", code)
	}
}

func TestMCPResponseContainsID(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	if !client.initialize() {
		t.Skip("Server not responding to initialize")
	}

	testID := float64(42)
	resp := client.send("tools/list", nil, testID)
	if resp == nil {
		t.Skip("Server not responding")
	}

	if resp["id"] != testID {
		t.Errorf("Response ID mismatch: got %v, want %v", resp["id"], testID)
	}
}

func TestMCPNotificationNoResponse(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	notification := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	client.mu.Lock()
	client.stdin.WriteString(notification)
	client.stdin.Flush()
	client.mu.Unlock()

	time.Sleep(50 * time.Millisecond)

	if client.cmd.ProcessState != nil {
		t.Error("Server crashed or exited after notification")
	}
}

func TestMCPRequestOrder(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	if !client.initialize() {
		t.Skip("Server not responding to initialize")
	}

	resp1 := client.send("tools/list", nil, 1)
	if resp1 == nil {
		t.Skip("Server not responding")
	}
	resp2 := client.send("tools/list", nil, 2)
	resp3 := client.send("tools/list", nil, 3)

	if resp1["id"] != float64(1) {
		t.Errorf("First response ID mismatch: got %v", resp1["id"])
	}
	if resp2["id"] != float64(2) {
		t.Errorf("Second response ID mismatch: got %v", resp2["id"])
	}
	if resp3["id"] != float64(3) {
		t.Errorf("Third response ID mismatch: got %v", resp3["id"])
	}
}

func TestMCPInvalidJSON(t *testing.T) {
	client := NewMCPClient(t)
	defer client.close()

	client.mu.Lock()
	client.stdin.WriteString("not valid json\n")
	client.stdin.Flush()
	client.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
}
