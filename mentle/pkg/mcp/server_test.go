package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestNewServerCreatesServerWithEmptyTools(t *testing.T) {
	stdin := bufio.NewReader(strings.NewReader(""))
	stdout := bufio.NewWriter(&bytes.Buffer{})
	stderr := bufio.NewWriter(&bytes.Buffer{})
	server := NewServer(stdin, stdout, stderr)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.tools == nil {
		t.Error("server.tools is nil, expected empty map")
	}
}

func TestServerHandlesToolsListRequest(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"tools/list","id":1}` + "\n"
	stdin := bufio.NewReader(strings.NewReader(input))
	var output bytes.Buffer
	stdout := bufio.NewWriter(&output)
	stderr := bufio.NewWriter(&bytes.Buffer{})

	server := NewServer(stdin, stdout, stderr)
	server.RegisterTool("test_tool", "A test tool", nil, func(params map[string]any) (any, error) {
		return nil, nil
	})

	var wg sync.WaitGroup
	wg.Go(func() {
		server.Run()
	})

	wg.Wait()

	respBytes := output.String()
	if respBytes == "" {
		t.Fatal("No response received")
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(respBytes)), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID.(float64) != 1 {
		t.Errorf("Expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}
	toolsRaw, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("Result.tools is not an array")
	}
	if len(toolsRaw) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsRaw))
	}
}

func TestServerHandlesToolsCallWithValidTool(t *testing.T) {
	callParams := ToolCallParams{
		Name:      "test_tool",
		Arguments: map[string]any{"query": "hello"},
	}
	paramsBytes, _ := json.Marshal(callParams)

	input := `{"jsonrpc":"2.0","method":"tools/call","params":` + string(paramsBytes) + `,"id":2}` + "\n"
	stdin := bufio.NewReader(strings.NewReader(input))
	var output bytes.Buffer
	stdout := bufio.NewWriter(&output)
	stderr := bufio.NewWriter(&bytes.Buffer{})

	server := NewServer(stdin, stdout, stderr)
	server.RegisterTool("test_tool", "A test tool", nil, func(params map[string]any) (any, error) {
		return map[string]any{"result": "success"}, nil
	})

	var wg sync.WaitGroup
	wg.Go(func() {
		server.Run()
	})

	wg.Wait()

	respBytes := output.String()
	if respBytes == "" {
		t.Fatal("No response received")
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(respBytes)), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}
	if result["result"] != "success" {
		t.Errorf("Expected result 'success', got %v", result["result"])
	}
}

func TestServerHandlesToolsCallWithInvalidTool(t *testing.T) {
	callParams := ToolCallParams{
		Name:      "nonexistent",
		Arguments: map[string]any{},
	}
	paramsBytes, _ := json.Marshal(callParams)

	input := `{"jsonrpc":"2.0","method":"tools/call","params":` + string(paramsBytes) + `,"id":3}` + "\n"
	stdin := bufio.NewReader(strings.NewReader(input))
	var output bytes.Buffer
	stdout := bufio.NewWriter(&output)
	stderr := bufio.NewWriter(&bytes.Buffer{})

	server := NewServer(stdin, stdout, stderr)

	var wg sync.WaitGroup
	wg.Go(func() {
		server.Run()
	})

	wg.Wait()

	respBytes := output.String()
	if respBytes == "" {
		t.Fatal("No response received")
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(respBytes)), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error for nonexistent tool")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestJSONRPCRequestMarshaling(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var unmarshaled JSONRPCRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", unmarshaled.JSONRPC)
	}
	if unmarshaled.Method != "tools/list" {
		t.Errorf("Expected method 'tools/list', got %s", unmarshaled.Method)
	}
	if unmarshaled.ID.(float64) != 1 {
		t.Errorf("Expected ID 1, got %v", unmarshaled.ID)
	}
}

func TestJSONRPCResponseMarshaling(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  map[string]any{"tools": []any{}},
		ID:      1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled JSONRPCResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", unmarshaled.JSONRPC)
	}
	if unmarshaled.ID.(float64) != 1 {
		t.Errorf("Expected ID 1, got %v", unmarshaled.ID)
	}
	if unmarshaled.Error != nil {
		t.Errorf("Expected no error, got %v", unmarshaled.Error)
	}
}
