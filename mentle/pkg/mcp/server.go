// Package mcp implements the Model Context Protocol server using JSON-RPC over stdio.
// It exposes mempalace tools to MCP clients like Claude Desktop.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Server struct {
	tools    map[string]ToolHandler
	toolMeta map[string]ToolMeta
	stdin    *bufio.Reader
	stdout   *bufio.Writer
	stderr   *bufio.Writer
	stack    any
	searcher any
	kgDB     any
	redis    any
}

type ToolHandler func(params map[string]any) (any, error)

type ToolMeta struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func NewServer(stdin *bufio.Reader, stdout, stderr *bufio.Writer) *Server {
	return &Server{
		tools:    make(map[string]ToolHandler),
		toolMeta: make(map[string]ToolMeta),
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
	}
}

func NewServerDefault() *Server {
	return NewServer(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout), bufio.NewWriter(os.Stderr))
}

func (s *Server) Initialize(ctx context.Context, stack, searcher, kgDB, redis any) error {
	s.stack = stack
	s.searcher = searcher
	s.kgDB = kgDB
	s.redis = redis
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

func (s *Server) RegisterTool(name, description string, inputSchema json.RawMessage, handler ToolHandler) {
	s.tools[name] = handler
	s.toolMeta[name] = ToolMeta{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

func (s *Server) sendError(id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	}
	respBytes, _ := json.Marshal(resp)
	s.stdout.WriteString(string(respBytes) + "\n")
	s.stdout.Flush()
}

func (s *Server) Run() error {
	for {
		line, err := s.stdin.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error: invalid JSON")
			continue
		}

		if req.JSONRPC != "2.0" {
			s.sendError(req.ID, -32600, "Invalid Request: jsonrpc must be 2.0")
			continue
		}

		if req.Method == "" {
			s.sendError(req.ID, -32600, "Invalid Request: method is required")
			continue
		}

		s.handleRequest(req)
	}
}

func (s *Server) handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "tools/list":
		var toolList []Tool
		for _, meta := range s.toolMeta {
			toolList = append(toolList, Tool{
				Name:        meta.Name,
				Description: meta.Description,
				InputSchema: meta.InputSchema,
			})
		}
		s.sendResult(req.ID, ToolListResult{Tools: toolList})

	case "resources/list":
		s.sendResult(req.ID, map[string]any{
			"resources": []map[string]any{
				{
					"uri":         "mempalace://palace/status",
					"name":        "Palace Status",
					"description": "Current palace status and statistics",
				},
			},
		})

	case "resources/read":
		var resourceURI string
		if req.Params != nil {
			var params map[string]any
			json.Unmarshal(req.Params, &params)
			if uri, ok := params["uri"].(string); ok {
				resourceURI = uri
			}
		}

		if resourceURI == "mempalace://palace/status" {
			s.sendResult(req.ID, map[string]any{
				"contents": []map[string]any{
					{
						"uri":  resourceURI,
						"text": "Palace resource - use tools/list and tools/call for interactions",
					},
				},
			})
		} else {
			s.sendError(req.ID, -32601, fmt.Sprintf("Resource not found: %s", resourceURI))
		}

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
			return
		}
		handler, ok := s.tools[params.Name]
		if !ok {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", params.Name))
			return
		}
		result, err := handler(params.Arguments)
		if err != nil {
			s.sendError(req.ID, -32603, fmt.Sprintf("Internal error: %v", err))
			return
		}
		s.sendResult(req.ID, result)

	case "initialize":
		s.sendResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
				"logging":   map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mempalace-go",
				"version": "1.0.0",
			},
		})

	case "notifications/initialized", "notifications/cancelled":
		return

	case "notifications/message":
		// Logging notification from client
		return

	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) sendResult(id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	respBytes, _ := json.Marshal(resp)
	s.stdout.WriteString(string(respBytes) + "\n")
	s.stdout.Flush()
}

func RunStdio() {
	server := NewServerDefault()
	RunServer(server)
}

func RunServer(server *Server) {
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
}
