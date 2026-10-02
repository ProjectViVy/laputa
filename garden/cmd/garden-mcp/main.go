// garden-mcp: MCP (Model Context Protocol) server that exposes Garden's
// memory capabilities (Persona, Recall/Memories) to MCP clients such as
// OpenClaw. It bridges to the running Garden HTTP API over loopback.
//
// Transport: stdio (per MCP spec for local servers).
// Usage: garden-mcp [--base-url http://127.0.0.1:7373]
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	toolPersonaStatus      = "persona_status"
	toolPersonaGet         = "persona_get"
	toolPersonaPropose     = "persona_propose"
	toolPersonaP16         = "persona_p16"
	toolActmemRead         = "actmem_read"
	toolActmemQuery        = "actmem_query"
	toolActmemAppend       = "actmem_append"
	toolIndexHealth        = "index_health"
	toolMemorySearch       = "memory_search"
	toolEvidenceSearch     = "evidence_search"
	toolEvidenceGet        = "evidence_get"
	toolEvidenceCollection = "evidence_collections"
	toolActivityRecent     = "activity_recent"
	toolRecordSignal       = "record_signal"
)

// Client is a thin HTTP client over the Garden /v2 API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type HTTPError struct {
	Method string
	Path   string
	Status int
	Body   string
}

var errBackendTransport = errors.New("garden backend transport failure")

type backendErrorEnvelope struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Legacy    string         `json:"error"`
	Retryable bool           `json:"retryable"`
	RequestID string         `json:"request_id"`
	Details   map[string]any `json:"details"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("garden %s %s -> %d %s", e.Method, e.Path, e.Status, http.StatusText(e.Status))
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_AGENT_TOKEN")),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// get performs a GET request and decodes the JSON response into out.
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// post performs a POST request with a JSON body and decodes the response.
func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	return c.doWithHeaders(ctx, method, path, body, nil, out)
}

func (c *Client) doWithHeaders(ctx context.Context, method, path string, body any, headers map[string]string, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", errBackendTransport, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return &HTTPError{Method: method, Path: path, Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) postWithHeaders(ctx context.Context, path string, body any, headers map[string]string, out any) error {
	return c.doWithHeaders(ctx, http.MethodPost, path, body, headers, out)
}

// --- Persona types (mirror of /v2/persona/status response) ---

type personaStatusResponse struct {
	Status    string            `json:"status"`
	Documents []personaFileView `json:"documents"`
}

type personaFileView struct {
	Kind         string  `json:"kind"`
	FileName     string  `json:"file_name"`
	Exists       bool    `json:"exists"`
	Valid        bool    `json:"valid"`
	Reason       *string `json:"reason"`
	Revision     int     `json:"revision"`
	UpdatedAt    *string `json:"updated_at"`
	PendingCount int     `json:"pending_count"`
	ContentLimit int     `json:"content_limit"`
	FrozenLimit  *int    `json:"frozen_limit"`
	Required     bool    `json:"required"`
	ToolOnly     bool    `json:"tool_only"`
}

type personaDocumentResponse struct {
	Kind         string  `json:"kind"`
	FileName     string  `json:"file_name"`
	Exists       bool    `json:"exists"`
	Valid        bool    `json:"valid"`
	Content      string  `json:"content"`
	Revision     int     `json:"revision"`
	ContentHash  string  `json:"content_hash"`
	UpdatedAt    *string `json:"updated_at"`
	PendingCount int     `json:"pending_count"`
}

type personaWriteResponse struct {
	Document personaDocumentResponse `json:"document"`
	Changed  bool                    `json:"changed"`
}

// --- Recall types (mirror of /v2/recall/fast response) ---

type fastRecallRequest struct {
	Query       string `json:"query"`
	BudgetChars int    `json:"budget_chars"`
}

type fastRecallResponse struct {
	TraceID  string   `json:"trace_id"`
	Mode     string   `json:"mode"`
	Cards    []rcard  `json:"cards"`
	Context  string   `json:"context"`
	Degraded bool     `json:"degraded"`
	Warnings []string `json:"warnings"`
}

type rcard struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary"`
	SourceRef string  `json:"source_ref"`
	Revision  int     `json:"revision"`
	Status    string  `json:"status"`
	HeatScore float64 `json:"heat_score"`
}

// --- Memories types (mirror of /v2/memories response) ---

type memoriesResponse struct {
	Items []memoryItem `json:"items"`
}

type memoryMutationResponse struct {
	Memory     memoryItem `json:"memory"`
	IndexState string     `json:"index_state"`
	IndexJobID *string    `json:"index_job_id"`
}

type memoryItem struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Content    string   `json:"content"`
	Status     string   `json:"status"`
	Version    int      `json:"version"`
	Lifecycle  string   `json:"lifecycle"`
	Collection string   `json:"collection"`
	Tags       []string `json:"tags"`
	UpdatedAt  string   `json:"updated_at"`
	Source     struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id,omitempty"`
		EventID   string `json:"event_id,omitempty"`
		URI       string `json:"uri,omitempty"`
	} `json:"source"`
}

type actmemResponse struct {
	Revision  uint64 `json:"revision"`
	UpdatedAt string `json:"updated_at"`
	Markdown  string `json:"markdown"`
	Truncated bool   `json:"truncated"`
}

type actmemQueryResponse struct {
	Revision uint64 `json:"revision"`
	Items    []struct {
		Section     string  `json:"section"`
		WorkSection *string `json:"work_section"`
		LineIndex   int     `json:"line_index"`
		Excerpt     string  `json:"excerpt"`
	} `json:"items"`
	ReturnedChars int  `json:"returned_chars"`
	Truncated     bool `json:"truncated"`
}

type actmemWriteResponse struct {
	Result struct {
		Changed  bool   `json:"changed"`
		Revision uint64 `json:"revision"`
	} `json:"result"`
}

type personaReviewResponse struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	BaseRevision uint64 `json:"base_revision"`
	BaseHash     string `json:"base_hash"`
	Actor        string `json:"actor"`
	Reason       string `json:"reason"`
	State        string `json:"state"`
	CreatedAt    string `json:"created_at"`
}

type indexHealthResponse struct {
	Status               string   `json:"status"`
	Reasons              []string `json:"reasons"`
	CanonicalActiveCount int      `json:"canonical_active_count"`
	VectorActiveCount    int      `json:"vector_active_count"`
	VectorPhysicalCount  int      `json:"vector_physical_count"`
	BM25Count            int      `json:"bm25_count"`
	PendingJobs          int      `json:"pending_jobs"`
	FailedJobs           int      `json:"failed_jobs"`
}

// --- Evidence types (mirror of /v2/materials/cards and .../evidence) ---

type evidenceCardsResponse struct {
	Cards      []evidenceCard `json:"cards"`
	NextCursor any            `json:"next_cursor"`
	Source     string         `json:"source"`
}

type evidenceCard struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Collection string   `json:"collection"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	SourceRef  string   `json:"source_ref"`
	Revision   int      `json:"revision"`
	Status     string   `json:"status"`
	Tags       []string `json:"tags"`
	HeatScore  float64  `json:"heat_score"`
	ValidFrom  string   `json:"valid_from"`
}

type evidenceGetResponse struct {
	CardID    string             `json:"card_id"`
	Fragments []evidenceFragment `json:"fragments"`
	Source    string             `json:"source"`
}

type evidenceFragment struct {
	CardID      string `json:"card_id"`
	MaterialRef string `json:"material_ref"`
	SourceRev   string `json:"source_rev"`
	Excerpt     string `json:"excerpt"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	ContentHash string `json:"content_hash"`
	Validity    string `json:"validity"`
}

type evidenceCollectionsResponse struct {
	Collections []evidenceCollection `json:"collections"`
	Source      string               `json:"source"`
}

type evidenceCollection struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// --- Activity types (mirror of /v2/activity/sessions/{id}) ---

type activityEventsResponse struct {
	SessionID string          `json:"session_id"`
	Events    []activityEvent `json:"events"`
}

type activityEvent struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
}

// --- Write types (POST /v2/memories, POST /v2/activity/events) ---

type createMemoryRequest struct {
	Content string   `json:"content"`
	Kind    string   `json:"kind,omitempty"`
	Scope   string   `json:"scope,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Source  any      `json:"source,omitempty"`
}

type createActivityRequest struct {
	SessionID string         `json:"session_id"`
	EventID   string         `json:"event_id"`
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
}

type createActivityResponse struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

func main() {
	baseURL := flag.String("base-url", envOr("GARDEN_MCP_BASE_URL", "http://127.0.0.1:7373"), "Garden HTTP API base URL")
	flag.Parse()

	if err := server.ServeStdio(newMCPServer(NewClient(*baseURL))); err != nil {
		fmt.Fprintf(os.Stderr, "garden-mcp: %v\n", err)
		os.Exit(1)
	}
}

func newMCPServer(client *Client) *server.MCPServer {
	srv := server.NewMCPServer(
		"garden",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Garden MemoryOS provider. Persona protected edits create reviews; ACTMEM is explicit; memories write through canonical Mentle; index health is read-only."),
	)

	srv.AddTool(handlePersonaStatus(client))
	srv.AddTool(handlePersonaGet(client))
	srv.AddTool(handlePersonaPropose(client))
	srv.AddTool(handlePersonaP16(client))
	srv.AddTool(handleActmemRead(client))
	srv.AddTool(handleActmemQuery(client))
	srv.AddTool(handleActmemAppend(client))
	srv.AddTool(handleIndexHealth(client))
	srv.AddTool(handleMemorySearch(client))
	srv.AddTool(handleEvidenceSearch(client))
	srv.AddTool(handleEvidenceGet(client))
	srv.AddTool(handleEvidenceCollections(client))
	srv.AddTool(handleActivityRecent(client))
	srv.AddTool(handleRecordSignal(client))
	return srv
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- tool handlers ---

func handlePersonaStatus(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolPersonaStatus,
		mcp.WithDescription("Get the Garden persona status: profile state (ready/incomplete/uninitialized) plus each of the seven persona Markdown files (IDENTITY, RELATIONSHIP, REDLINE, USER, DREAM, DARK, WORLD) with existence, validity, revision, content/frozen limits, and required flags."),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp personaStatusResponse
		if err := client.get(ctx, "/v2/persona/documents", &resp); err != nil {
			return errorResult(err), nil
		}
		text := formatPersonaStatus(resp)
		return okResult(text), nil
	}
	return tool, handler
}

func handlePersonaGet(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolPersonaGet,
		mcp.WithDescription("Read a single Garden persona Markdown file by kind (identity, relationship, redline, user, dream, dark, world). Returns the full Markdown content plus revision and content hash."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Persona file kind: identity, relationship, redline, user, dream, dark, or world")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind, _ := req.Params.Arguments.(map[string]any)["kind"].(string)
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind == "" {
			return errorResult(fmt.Errorf("missing required argument: kind")), nil
		}
		var resp personaDocumentResponse
		if err := client.get(ctx, "/v2/persona/documents/"+urlPathEscape(kind), &resp); err != nil {
			return errorResult(err), nil
		}
		text := formatPersonaDocument(resp)
		return okResult(text), nil
	}
	return tool, handler
}

func handlePersonaPropose(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolPersonaPropose,
		mcp.WithDescription("Submit a protected Persona Markdown proposal for user review. This creates a review request and never applies the change directly."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Persona kind: identity, relationship, redline, user, dream, dark, or world")),
		mcp.WithNumber("base_revision", mcp.Required(), mcp.Description("Revision read before preparing the proposal")),
		mcp.WithString("base_hash", mcp.Required(), mcp.Description("Content hash read for the base revision")),
		mcp.WithString("proposed_markdown", mcp.Required(), mcp.Description("Proposed Markdown body")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("Why the proposal is needed")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		kind, _ := args["kind"].(string)
		baseHash, _ := args["base_hash"].(string)
		proposed, _ := args["proposed_markdown"].(string)
		reason, _ := args["reason"].(string)
		baseRevision, ok := integerArgument(args["base_revision"])
		if strings.TrimSpace(kind) == "" || !ok || strings.TrimSpace(baseHash) == "" || strings.TrimSpace(proposed) == "" || strings.TrimSpace(reason) == "" {
			return errorResult(fmt.Errorf("kind, base_revision, base_hash, proposed_markdown and reason are required")), nil
		}
		var resp personaReviewResponse
		if err := client.post(ctx, "/v2/persona/reviews", map[string]any{
			"kind": kind, "base_revision": baseRevision, "base_hash": baseHash,
			"proposed_markdown": proposed, "reason": reason,
		}, &resp); err != nil {
			return errorResult(err), nil
		}
		return okResult(fmt.Sprintf("Persona proposal %s created\nkind=%s base_revision=%d state=%s\nreason: %s", resp.ID, resp.Kind, resp.BaseRevision, resp.State, resp.Reason)), nil
	}
	return tool, handler
}

func handlePersonaP16(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolPersonaP16,
		mcp.WithDescription("Apply a legal direct Persona P16 write using the MCP agent capability: DREAM, DARK, or USER Observations only. Protected Persona documents must use persona_propose instead."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Persona kind: dream, dark, or user")),
		mcp.WithNumber("base_revision", mcp.Required(), mcp.Description("Exact revision read before the direct write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Dream, dark, or USER Observations Markdown body")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("Why the direct P16 write is needed")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		kind, _ := args["kind"].(string)
		kind = strings.ToLower(strings.TrimSpace(kind))
		content, _ := args["content"].(string)
		reason, _ := args["reason"].(string)
		baseRevision, revisionOK := integerArgument(args["base_revision"])
		if kind != "dream" && kind != "dark" && kind != "user" {
			return errorResult(fmt.Errorf("persona_p16 kind must be dream, dark, or user")), nil
		}
		if !revisionOK || baseRevision <= 0 || strings.TrimSpace(content) == "" || strings.TrimSpace(reason) == "" {
			return errorResult(fmt.Errorf("kind, positive base_revision, content and reason are required")), nil
		}
		scope := "document"
		if kind == "user" {
			scope = "observations"
		}
		var resp personaWriteResponse
		if err := client.put(ctx, "/v2/persona/documents/"+urlPathEscape(kind), map[string]any{
			"base_revision": baseRevision,
			"content":       content,
			"scope":         scope,
			"reason":        reason,
		}, &resp); err != nil {
			return errorResult(err), nil
		}
		return okResult(fmt.Sprintf("Persona P16 write changed=%t kind=%s revision=%d", resp.Changed, resp.Document.Kind, resp.Document.Revision)), nil
	}
	return tool, handler
}

func handleActmemRead(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolActmemRead,
		mcp.WithDescription("Explicitly read the bounded ACTMEM activity memory. ACTMEM is tool-only and is not part of bootstrap or automatic context."),
		mcp.WithNumber("max_chars", mcp.Description("Maximum returned characters (default 1200)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		path := "/v2/actmem"
		value, err := optionalInteger(args, "max_chars", 1200, 1, 1200)
		if err != nil {
			return errorResult(err), nil
		}
		if value != 1200 {
			path += fmt.Sprintf("?max_chars=%d", value)
		}
		var resp actmemResponse
		if err := client.get(ctx, path, &resp); err != nil {
			return errorResult(err), nil
		}
		return okResult(fmt.Sprintf("ACTMEM revision=%d updated_at=%s truncated=%t\n---\n%s", resp.Revision, resp.UpdatedAt, resp.Truncated, resp.Markdown)), nil
	}
	return tool, handler
}

func handleActmemQuery(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolActmemQuery,
		mcp.WithDescription("Explicitly query ACTMEM sections. Results are bounded excerpts and never feed automatic context assembly."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Text to find in ACTMEM")),
		mcp.WithArray("sections", mcp.WithStringItems(), mcp.Description("Optional sections: Pulse, Recap, or Work")),
		mcp.WithNumber("max_hits", mcp.Description("Maximum hits")),
		mcp.WithNumber("max_chars", mcp.Description("Maximum returned characters")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return errorResult(fmt.Errorf("missing required argument: query")), nil
		}
		sections := []string{}
		if raw, ok := args["sections"].([]any); ok {
			for _, item := range raw {
				if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
					sections = append(sections, strings.TrimSpace(value))
				}
			}
		}
		body := map[string]any{"query": query, "sections": sections}
		maxHits, err := optionalInteger(args, "max_hits", 20, 1, 50)
		if err != nil {
			return errorResult(err), nil
		}
		maxChars, err := optionalInteger(args, "max_chars", 1200, 1, 1200)
		if err != nil {
			return errorResult(err), nil
		}
		body["max_hits"] = maxHits
		body["max_chars"] = maxChars
		var resp actmemQueryResponse
		if err := client.post(ctx, "/v2/actmem/query", body, &resp); err != nil {
			return errorResult(err), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "ACTMEM query %q revision=%d hits=%d truncated=%t\n", query, resp.Revision, len(resp.Items), resp.Truncated)
		for _, item := range resp.Items {
			section := item.Section
			if item.WorkSection != nil && *item.WorkSection != "" {
				section += "/" + *item.WorkSection
			}
			fmt.Fprintf(&b, "- [%s line %d] %s\n", section, item.LineIndex, item.Excerpt)
		}
		if len(resp.Items) == 0 {
			b.WriteString("(no ACTMEM matches)\n")
		}
		return okResult(b.String()), nil
	}
	return tool, handler
}

func handleActmemAppend(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolActmemAppend,
		mcp.WithDescription("Append a bounded activity pulse to ACTMEM through the authorized maintenance endpoint. The caller scope is derived from the authenticated principal, never supplied. This is a direct ACTMEM write, not a Persona or Mentle memory write."),
		mcp.WithString("session_key", mcp.Required(), mcp.Description("Stable session key")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Pulse content")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		sessionKey, _ := args["session_key"].(string)
		content, _ := args["content"].(string)
		if strings.TrimSpace(sessionKey) == "" || strings.TrimSpace(content) == "" {
			return errorResult(fmt.Errorf("session_key and content are required")), nil
		}
		var resp actmemWriteResponse
		if err := client.post(ctx, "/v2/actmem/maintenance", map[string]any{
			"operation": "system_append",
			"entry":     map[string]any{"section": "pulse", "session_id": sessionKey, "body": content},
		}, &resp); err != nil {
			return errorResult(err), nil
		}
		return okResult(fmt.Sprintf("ACTMEM pulse appended changed=%t revision=%d", resp.Result.Changed, resp.Result.Revision)), nil
	}
	return tool, handler
}

func handleIndexHealth(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolIndexHealth,
		mcp.WithDescription("Read live Mentle canonical-vs-derived index health. This is diagnostic only; repair/rebuild remains an operator action."),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp indexHealthResponse
		if err := client.get(ctx, "/v2/admin/index-health", &resp); err != nil {
			return errorResult(fmt.Errorf("index health unavailable: %w", err)), nil
		}
		return okResult(formatIndexHealth(resp)), nil
	}
	return tool, handler
}

func handleMemorySearch(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolMemorySearch,
		mcp.WithDescription("Search Garden memory using query-scoped Fast Recall plus canonical memories whose fields literally match the query. Results include sources; no unfiltered recent tail is presented as a match."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default 5)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return errorResult(fmt.Errorf("missing required argument: query")), nil
		}
		maxResults, err := optionalInteger(args, "max_results", 5, 1, 100)
		if err != nil {
			return errorResult(err), nil
		}

		var recall fastRecallResponse
		recallErr := client.post(ctx, "/v2/recall/fast", fastRecallRequest{
			Query:       query,
			BudgetChars: 6000,
		}, &recall)

		var mems memoriesResponse
		memErr := client.get(ctx, "/v2/memories?limit=100", &mems)

		if recallErr != nil && memErr != nil {
			recallFailure := errorResult(recallErr).StructuredContent.(map[string]any)
			memoryFailure := errorResult(memErr).StructuredContent.(map[string]any)
			payload := map[string]any{
				"code":       "backend_error",
				"message":    "memory search backends unavailable",
				"retryable":  recallFailure["retryable"] == true || memoryFailure["retryable"] == true,
				"request_id": "",
				"details":    map[string]any{"recall": recallFailure, "memories": memoryFailure},
			}
			return &mcp.CallToolResult{
				Content:           []mcp.Content{mcp.NewTextContent("error [backend_error]: memory search backends unavailable")},
				StructuredContent: payload,
				IsError:           true,
			}, nil
		}

		text := formatMemorySearch(query, maxResults, &recall, recallErr, &mems, memErr)
		return okResult(text), nil
	}
	return tool, handler
}

func handleEvidenceSearch(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolEvidenceSearch,
		mcp.WithDescription("Search Garden Mentle evidence cards by query. Returns matching cards with id, kind, title, summary, source, tags, and heat score."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of cards (default 5)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return errorResult(fmt.Errorf("missing required argument: query")), nil
		}
		limit, err := optionalInteger(args, "limit", 5, 1, 100)
		if err != nil {
			return errorResult(err), nil
		}
		var resp evidenceCardsResponse
		if err := client.get(ctx, fmt.Sprintf("/v2/materials/cards?query=%s&limit=%d", urlQueryEscape(query), limit), &resp); err != nil {
			return errorResult(err), nil
		}
		text := formatEvidenceCards(resp)
		return okResult(text), nil
	}
	return tool, handler
}

func handleEvidenceGet(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolEvidenceGet,
		mcp.WithDescription("Get the evidence fragments for a specific Garden Mentle card by card id."),
		mcp.WithString("card_id", mcp.Required(), mcp.Description("Card id, e.g. mem_...")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		cardID, _ := args["card_id"].(string)
		cardID = strings.TrimSpace(cardID)
		if cardID == "" {
			return errorResult(fmt.Errorf("missing required argument: card_id")), nil
		}
		var resp evidenceGetResponse
		if err := client.get(ctx, "/v2/materials/cards/"+urlPathEscape(cardID)+"/evidence", &resp); err != nil {
			return errorResult(err), nil
		}
		text := formatEvidenceGet(resp)
		return okResult(text), nil
	}
	return tool, handler
}

func handleEvidenceCollections(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolEvidenceCollection,
		mcp.WithDescription("List Garden Mentle evidence collections with their card counts."),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp evidenceCollectionsResponse
		if err := client.get(ctx, "/v2/materials/collections", &resp); err != nil {
			return errorResult(err), nil
		}
		var b strings.Builder
		if len(resp.Collections) == 0 {
			fmt.Fprintf(&b, "(no collections)\n")
		} else {
			for _, c := range resp.Collections {
				fmt.Fprintf(&b, "- %s (%d cards)\n", c.Name, c.Count)
			}
		}
		return okResult(b.String()), nil
	}
	return tool, handler
}

func handleActivityRecent(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolActivityRecent,
		mcp.WithDescription("Get recent Garden activity events for a session. Returns event id, type, and data."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session id, e.g. sess_...")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of events (default 20)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		sessionID, _ := args["session_id"].(string)
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			return errorResult(fmt.Errorf("missing required argument: session_id")), nil
		}
		limit, err := optionalInteger(args, "limit", 20, 1, 200)
		if err != nil {
			return errorResult(err), nil
		}
		var resp activityEventsResponse
		if err := client.get(ctx, fmt.Sprintf("/v2/activity/sessions/%s?limit=%d", urlPathEscape(sessionID), limit), &resp); err != nil {
			return errorResult(err), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Session %s (%d events)\n", resp.SessionID, len(resp.Events))
		if len(resp.Events) == 0 {
			fmt.Fprintf(&b, "(no events)\n")
		}
		for _, e := range resp.Events {
			data := ""
			if e.Data != nil {
				db, err := json.Marshal(e.Data)
				if err == nil {
					data = " " + string(db)
				}
			}
			fmt.Fprintf(&b, "- [%s] %s%s\n", e.ID, e.Type, data)
		}
		return okResult(b.String()), nil
	}
	return tool, handler
}

func handleRecordSignal(client *Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool(
		toolRecordSignal,
		mcp.WithDescription("Record a durable signal/memory into Garden as a persistent memory entry (writes via POST /v2/memories). Use when the agent discovers a fact worth persisting. This is a WRITE operation."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The signal/fact content to persist")),
		mcp.WithString("kind", mcp.Description("Memory kind (default 'note'; valid: note, fact, preference, decision, session_digest, source_artifact, semantic_unit)")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Tags to attach")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		content, _ := args["content"].(string)
		content = strings.TrimSpace(content)
		if content == "" {
			return errorResult(fmt.Errorf("missing required argument: content")), nil
		}
		kind, _ := args["kind"].(string)
		if kind == "" {
			kind = "note"
		}
		var tags []string
		if rawTags, ok := args["tags"].([]any); ok {
			for _, t := range rawTags {
				if s, ok := t.(string); ok && strings.TrimSpace(s) != "" {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
		}
		body := createMemoryRequest{
			Content: content,
			Kind:    kind,
			Tags:    tags,
			Source:  map[string]any{"type": "agent"},
		}
		var resp memoryMutationResponse
		encoded, _ := json.Marshal(body)
		key := fmt.Sprintf("agent-signal-%x", sha256.Sum256(encoded))
		if err := client.postWithHeaders(ctx, "/v2/memories", body, map[string]string{"Idempotency-Key": key}, &resp); err != nil {
			return errorResult(err), nil
		}
		text := fmt.Sprintf("recorded memory %s (lifecycle=%s, collection=%s, index_state=%s)\ncontent: %s", resp.Memory.ID, resp.Memory.Lifecycle, resp.Memory.Collection, resp.IndexState, resp.Memory.Content)
		return okResult(text), nil
	}
	return tool, handler
}

// --- formatting helpers ---

func formatEvidenceCards(resp evidenceCardsResponse) string {
	var b strings.Builder
	if len(resp.Cards) == 0 {
		fmt.Fprintf(&b, "(no matching evidence cards)\n")
		return b.String()
	}
	for _, c := range resp.Cards {
		fmt.Fprintf(&b, "- [%s] (%s/%s) %s\n", c.ID, c.Kind, c.Collection, c.Title)
		if c.Summary != "" && c.Summary != c.Title {
			fmt.Fprintf(&b, "  %s\n", c.Summary)
		}
		if len(c.Tags) > 0 {
			fmt.Fprintf(&b, "  tags=[%s]\n", strings.Join(c.Tags, ","))
		}
		fmt.Fprintf(&b, "  source=%s rev=%d status=%s heat=%.3f\n", c.SourceRef, c.Revision, c.Status, c.HeatScore)
	}
	return b.String()
}

func formatEvidenceGet(resp evidenceGetResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Card %s (%d fragments)\n", resp.CardID, len(resp.Fragments))
	for _, f := range resp.Fragments {
		fmt.Fprintf(&b, "- %s\n  excerpt: %s\n  ref: %s (rev %s)\n", f.MaterialRef, f.Excerpt, f.CardID, f.SourceRev)
	}
	return b.String()
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

func urlPathEscape(s string) string {
	return url.PathEscape(s)
}

func formatPersonaStatus(resp personaStatusResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Persona profile: %s\n\n", resp.Status)
	// stable order
	order := []string{"identity", "relationship", "redline", "user", "dream", "dark", "world"}
	for _, kind := range order {
		var f *personaFileView
		for i := range resp.Documents {
			if resp.Documents[i].Kind == kind {
				f = &resp.Documents[i]
				break
			}
		}
		if f == nil {
			continue
		}
		state := "missing"
		if f.Exists {
			state = "exists"
		}
		if f.Exists && f.Valid {
			state = "ready"
		}
		if f.Exists && !f.Valid {
			state = fmt.Sprintf("invalid (%s)", orNil(f.Reason))
		}
		fmt.Fprintf(&b, "- %s [%s] rev=%d pending=%d limit=%d%s\n",
			f.FileName, state, f.Revision, f.PendingCount, f.ContentLimit, toolOnlySuffix(*f))
	}
	return b.String()
}

func formatPersonaDocument(resp personaDocumentResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)\n", resp.FileName, resp.Kind)
	if !resp.Exists {
		fmt.Fprintf(&b, "Not created (optional file)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "revision: %d\n", resp.Revision)
	if resp.ContentHash != "" {
		fmt.Fprintf(&b, "content_hash: %s\n", resp.ContentHash)
	}
	fmt.Fprintf(&b, "---\n%s\n", resp.Content)
	return b.String()
}

func formatMemorySearch(query string, maxResults int, recall *fastRecallResponse, recallErr error, mems *memoriesResponse, memErr error) string {
	if maxResults < 1 {
		maxResults = 1
	}
	var b strings.Builder
	matchedMemories := []memoryItem(nil)
	if memErr == nil && mems != nil {
		matchedMemories = filterMemoryItems(mems.Items, query)
	}
	degraded := recallErr != nil || memErr != nil
	resultCount := 0
	if recallErr == nil && recall != nil {
		resultCount += len(recall.Cards)
	}
	resultCount += len(matchedMemories)
	state := "ok"
	if degraded {
		state = "degraded"
	} else if resultCount == 0 {
		state = "empty"
	}
	fmt.Fprintf(&b, "Memory search: %q (status=%s)\n\n", query, state)

	if recallErr == nil && recall != nil {
		fmt.Fprintf(&b, "=== Fast Recall (mode=%s, trace=%s) ===\n", recall.Mode, recall.TraceID)
		if len(recall.Cards) == 0 {
			fmt.Fprintf(&b, "(no recall cards)\n")
		} else {
			n := maxResults
			if len(recall.Cards) < n {
				n = len(recall.Cards)
			}
			for _, c := range recall.Cards[:n] {
				fmt.Fprintf(&b, "- [%s] %s (%s) heat=%.3f\n  %s\n", c.ID, c.Title, c.SourceRef, c.HeatScore, c.Summary)
			}
		}
		if recall.Context != "" {
			fmt.Fprintf(&b, "\ncontext: %s\n", recall.Context)
		}
	} else if recallErr != nil {
		fmt.Fprintf(&b, "(recall unavailable: %v)\n", recallErr)
	}

	if memErr == nil && len(matchedMemories) > 0 {
		fmt.Fprintf(&b, "\n=== Canonical memories matching query ===\n")
		n := maxResults
		if len(matchedMemories) < n {
			n = len(matchedMemories)
		}
		for _, m := range matchedMemories[:n] {
			tags := ""
			if len(m.Tags) > 0 {
				tags = " tags=[" + strings.Join(m.Tags, ",") + "]"
			}
			fmt.Fprintf(&b, "- [%s] (%s/%s)%s\n  %s\n", m.ID, m.Lifecycle, m.Collection, tags, m.Content)
		}
	} else if memErr != nil {
		fmt.Fprintf(&b, "(memories unavailable: %v)\n", memErr)
	} else if resultCount == 0 {
		fmt.Fprintf(&b, "(no query-matched results)\n")
	}

	return b.String()
}

func filterMemoryItems(items []memoryItem, query string) []memoryItem {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil
	}
	matched := make([]memoryItem, 0, len(items))
	for _, item := range items {
		values := []string{item.Content, item.Kind, item.Lifecycle, item.Collection, item.Source.Type, item.Source.SessionID, item.Source.EventID, item.Source.URI}
		values = append(values, item.Tags...)
		for _, value := range values {
			if strings.Contains(strings.ToLower(value), needle) {
				matched = append(matched, item)
				break
			}
		}
	}
	return matched
}

func formatIndexHealth(resp indexHealthResponse) string {
	status := resp.Status
	if status == "" {
		status = "unknown"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Mentle index health: %s\n", status)
	fmt.Fprintf(&b, "canonical_active=%d vector_active=%d vector_physical=%d bm25=%d\n", resp.CanonicalActiveCount, resp.VectorActiveCount, resp.VectorPhysicalCount, resp.BM25Count)
	fmt.Fprintf(&b, "jobs=pending:%d failed:%d\n", resp.PendingJobs, resp.FailedJobs)
	if len(resp.Reasons) > 0 {
		fmt.Fprintf(&b, "reasons: %s\n", strings.Join(resp.Reasons, ", "))
	}
	return b.String()
}

func integerArgument(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int64:
		return int(number), true
	case float64:
		return int(number), number == float64(int(number))
	case json.Number:
		parsed, err := number.Int64()
		return int(parsed), err == nil
	default:
		return 0, false
	}
}

func optionalInteger(args map[string]any, key string, fallback, minValue, maxValue int) (int, error) {
	if args == nil {
		return fallback, nil
	}
	raw, exists := args[key]
	if !exists || raw == nil {
		return fallback, nil
	}
	value, ok := integerArgument(raw)
	if !ok || value < minValue || value > maxValue {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", key, minValue, maxValue)
	}
	return value, nil
}

func orNil(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toolOnlySuffix(f personaFileView) string {
	if f.ToolOnly {
		return " (tool-only)"
	}
	return ""
}

func okResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(text)},
	}
}

func errorResult(err error) *mcp.CallToolResult {
	payload := map[string]any{"code": "invalid_request", "message": err.Error(), "retryable": false, "request_id": "", "details": map[string]any{}}
	var httpErr *HTTPError
	switch {
	case errors.As(err, &httpErr):
		payload["status"] = httpErr.Status
		payload["code"] = "backend_error"
		payload["message"] = http.StatusText(httpErr.Status)
		payload["retryable"] = httpErr.Status == http.StatusTooManyRequests || httpErr.Status >= http.StatusInternalServerError
		var envelope backendErrorEnvelope
		if json.Unmarshal([]byte(httpErr.Body), &envelope) == nil && envelope.Code != "" {
			payload["code"] = envelope.Code
			if envelope.Message != "" {
				payload["message"] = envelope.Message
			} else if envelope.Legacy != "" {
				payload["message"] = envelope.Legacy
			}
			payload["retryable"] = envelope.Retryable
			payload["request_id"] = envelope.RequestID
			if envelope.Details != nil {
				payload["details"] = envelope.Details
			}
		}
	case errors.Is(err, errBackendTransport):
		payload["code"] = "backend_unavailable"
		payload["message"] = "Garden backend unavailable"
		payload["retryable"] = true
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{mcp.NewTextContent(fmt.Sprintf("error [%s]: %s", payload["code"], payload["message"]))},
		StructuredContent: payload,
		IsError:           true,
	}
}
