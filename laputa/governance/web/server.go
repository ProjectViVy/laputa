// Package web exposes Laputa state over HTTP for human inspection.
//
// Routes:
//   GET  /                     -> dashboard listing all 14 sections
//   GET  /s/{name}             -> per-section HTML view
//   GET  /api/sections         -> JSON snapshot index
//   GET  /api/sections/{name}  -> JSON section data
//   GET  /api/snapshot         -> full Laputa snapshot as JSON
//   GET  /healthz              -> liveness probe
//
// Write endpoints (added for multi-session sharing — sessions need to
// update shared sections over HTTP rather than race on the JSON files):
//   POST   /api/sections/{name}  -> full replace, body = section JSON
//   PATCH  /api/sections/{name}  -> JSON Patch-style update at {path}
//   DELETE /api/sections/{name}  -> remove {path} from section
//
// Cross-process locking is enforced inside FileStore.Write (flock),
// so concurrent POST/PATCH from multiple sessions on the same section
// serialize cleanly rather than last-writer-wins.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	laputa "github.com/dashimaki/laputa/governance"
)

// Server is a read-only HTTP view onto a Laputa Engine.
type Server struct {
	engine *laputa.Engine
	addr   string

	indexTmpl *template.Template
	detailTmpl *template.Template
	started time.Time
}

// New creates a Server bound to engine.
func New(engine *laputa.Engine, addr string) (*Server, error) {
	idx, err := template.New("index").Parse(indexHTML)
	if err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	det, err := template.New("detail").Parse(detailHTML)
	if err != nil {
		return nil, fmt.Errorf("parse detail: %w", err)
	}
	return &Server{
		engine:     engine,
		addr:       addr,
		indexTmpl:  idx,
		detailTmpl: det,
		started:    time.Now(),
	}, nil
}

// ListenAndServe blocks; cancel ctx to shut down.
func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.recover(s.handleIndex))
	mux.HandleFunc("/s/", s.recover(s.handleDetail))
	mux.HandleFunc("/api/sections", s.recover(s.handleSectionsJSON))
	mux.HandleFunc("/api/sections/", s.recover(s.handleSectionWriteOrRead))
	mux.HandleFunc("/api/snapshot", s.recover(s.handleSnapshotJSON))
	mux.HandleFunc("/healthz", s.recover(s.handleHealthz))

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// recover wraps an http.HandlerFunc with panic recovery so a single bad request
// can't bring down the whole server.
func (s *Server) recover(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				_ = rec
			}
		}()
		h(w, r)
	}
}

// ---- handlers ----

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot, err := s.engine.Snapshot(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sections := sectionsMap(snapshot)
	rows := make([]indexRow, 0, len(laputa.AllSections))
	for _, name := range laputa.AllSections {
		rs := laputa.SectionRegistry[name]
		info, ok := sections[string(name)].(map[string]any)
		if !ok {
			continue
		}
		data := dataMap(info)
		rows = append(rows, indexRow{
			Name:        string(name),
			Number:      numberPrefix(string(name)),
			Status:      asString(info["status"]),
			WriteAuth:   asString(info["write_authority"]),
			SchemaOwner: rs.SchemaOwner,
			Size:        len(data),
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.indexTmpl.Execute(w, indexPage{
		Started: s.started,
		Rows:    rows,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	sectionName := strings.TrimPrefix(r.URL.Path, "/s/")
	if sectionName == "" {
		http.NotFound(w, r)
		return
	}
	section := laputa.SectionName(sectionName)
	if _, ok := laputa.SectionRegistry[section]; !ok {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	data, err := s.engine.GetSection(ctx, section)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	rs := laputa.SectionRegistry[section]
	pretty, _ := json.MarshalIndent(data, "", "  ")

	updatedAt := ""
	if m, ok := data["_meta"].(map[string]any); ok {
		updatedAt = asString(m["updated_at"])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.detailTmpl.Execute(w, detailPage{
		Name:        string(section),
		Number:      numberPrefix(string(section)),
		Status:      rs.Status,
		WriteAuth:   string(rs.WriteAuth),
		SchemaOwner: rs.SchemaOwner,
		JSON:        string(pretty),
		UpdatedAt:   updatedAt,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSectionsJSON(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.engine.Snapshot(ctx)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	sections := sectionsMap(snapshot)
	out := make([]sectionInfoJSON, 0, len(laputa.AllSections))
	for _, name := range laputa.AllSections {
		info, ok := sections[string(name)].(map[string]any)
		if !ok {
			continue
		}
		data := dataMap(info)
		rs := laputa.SectionRegistry[name]
		out = append(out, sectionInfoJSON{
			Name:        string(name),
			Status:      rs.Status,
			WriteAuth:   string(rs.WriteAuth),
			SchemaOwner: rs.SchemaOwner,
			Bytes:       len(data),
			HasMeta:     data["_meta"] != nil,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sections": out,
		"count":    len(out),
	})
}

func (s *Server) handleSectionJSON(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/sections/")
	section := laputa.SectionName(name)
	if _, ok := laputa.SectionRegistry[section]; !ok {
		writeJSONError(w, fmt.Errorf("unknown section: %s", name))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	data, err := s.engine.GetSection(ctx, section)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": name,
		"data": data,
	})
}

func (s *Server) handleSnapshotJSON(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snapshot, err := s.engine.Snapshot(ctx)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"started": s.started,
		"uptime":  time.Since(s.started).Round(time.Second).String(),
	})
}

// handleSectionWriteOrRead dispatches /api/sections/{name} by HTTP method.
// GET keeps the original handleSectionJSON behavior; POST/PATCH/DELETE add
// shared-write APIs so multiple sessions can update sections without
// racing on the JSON files (cross-process locking is enforced inside
// FileStore.Write via flock).
func (s *Server) handleSectionWriteOrRead(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleSectionJSON(w, r)
	case http.MethodPost:
		s.handleSectionWrite(w, r)
	case http.MethodPatch:
		s.handleSectionPatch(w, r)
	case http.MethodDelete:
		s.handleSectionDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) resolveSection(w http.ResponseWriter, r *http.Request) (laputa.SectionName, bool) {
	name := strings.TrimPrefix(r.URL.Path, "/api/sections/")
	section := laputa.SectionName(name)
	if _, ok := laputa.SectionRegistry[section]; !ok {
		writeJSONError(w, fmt.Errorf("unknown section: %s", name))
		return "", false
	}
	return section, true
}

// handleSectionWrite replaces the entire section body.
//
//	POST /api/sections/{name}
//	Body: full JSON object representing the section (excluding _meta,
//	      which the server sets to keep timestamps authoritative).
//
// Use PATCH for surgical updates; use POST for full replaces.
func (s *Server) handleSectionWrite(w http.ResponseWriter, r *http.Request) {
	section, ok := s.resolveSection(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, fmt.Errorf("read body: %w", err))
		return
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		writeJSONError(w, fmt.Errorf("invalid json: %w", err))
		return
	}
	// Strip _meta: server is authoritative for timestamps.
	delete(data, "_meta")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.engine.SetSection(ctx, section, data); err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"section": string(section),
		"action":  "write",
	})
}

// handleSectionPatch updates a single dot-notation path inside a section.
//
//	PATCH /api/sections/{name}?path=role
//	Body: JSON-encoded value (string, number, object, array, etc.)
func (s *Server) handleSectionPatch(w http.ResponseWriter, r *http.Request) {
	section, ok := s.resolveSection(w, r)
	if !ok {
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSONError(w, fmt.Errorf("missing ?path= query parameter"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, fmt.Errorf("read body: %w", err))
		return
	}
	// Allow empty body to mean "set to null"
	var value any = nil
	if len(body) > 0 {
		if err := json.Unmarshal(body, &value); err != nil {
			writeJSONError(w, fmt.Errorf("invalid json value: %w", err))
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.engine.UpdateSection(ctx, section, path, value); err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"section": string(section),
		"path":    path,
		"action":  "patch",
	})
}

// handleSectionDelete removes a dot-notation path inside a section.
//
//	DELETE /api/sections/{name}?path=capabilities.0
func (s *Server) handleSectionDelete(w http.ResponseWriter, r *http.Request) {
	section, ok := s.resolveSection(w, r)
	if !ok {
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSONError(w, fmt.Errorf("missing ?path= query parameter"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.engine.DeleteSectionPath(ctx, section, path); err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"section": string(section),
		"path":    path,
		"action":  "delete",
	})
}

// ---- helpers ----

type indexRow struct {
	Name        string
	Number      string
	Status      string
	WriteAuth   string
	SchemaOwner string
	Size        int
}

type indexPage struct {
	Started time.Time
	Rows    []indexRow
}

type detailPage struct {
	Name        string
	Number      string
	Status      string
	WriteAuth   string
	SchemaOwner string
	JSON        string
	UpdatedAt   string
}

type sectionInfoJSON struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	WriteAuth   string `json:"write_authority"`
	SchemaOwner string `json:"schema_owner"`
	Bytes       int    `json:"bytes"`
	HasMeta     bool   `json:"has_meta"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func numberPrefix(name string) string {
	if len(name) >= 3 && name[2] == '-' {
		return name[:2]
	}
	return ""
}

func sectionsMap(snapshot map[string]any) map[string]any {
	if s, ok := snapshot["sections"].(map[string]any); ok {
		return s
	}
	return map[string]any{}
}

func dataMap(info map[string]any) map[string]any {
	if d, ok := info["data"].(map[string]any); ok {
		return d
	}
	return map[string]any{}
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Laputa Dashboard</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; margin: 0; background: #0e1116; color: #c9d1d9; }
  header { background: #161b22; padding: 16px 24px; border-bottom: 1px solid #30363d; }
  header h1 { margin: 0; font-size: 18px; color: #58a6ff; }
  header .meta { color: #8b949e; font-size: 12px; margin-top: 4px; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 10px 16px; text-align: left; border-bottom: 1px solid #21262d; font-size: 14px; }
  th { background: #161b22; color: #8b949e; font-weight: 500; text-transform: uppercase; font-size: 11px; }
  a { color: #58a6ff; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .status-stable { color: #3fb950; }
  .status-tbd    { color: #d29922; }
  .auth-user_only { color: #f85149; }
  .auth-report_system { color: #58a6ff; }
  .auth-agent_self { color: #8b949e; }
</style>
</head>
<body>
<header>
  <h1>Laputa Governance</h1>
  <div class="meta">{{len .Rows}} sections · started {{.Started.Format "2006-01-02T15:04:05Z"}}</div>
</header>
<table>
<thead><tr>
  <th>#</th><th>Section</th><th>Status</th><th>Write Authority</th><th>Schema</th><th>Bytes</th>
</tr></thead>
<tbody>
{{range .Rows}}
<tr>
  <td>{{.Number}}</td>
  <td><a href="/s/{{.Name}}">{{.Name}}</a></td>
  <td class="status-{{.Status}}">{{.Status}}</td>
  <td class="auth-{{.WriteAuth}}">{{.WriteAuth}}</td>
  <td>{{.SchemaOwner}}</td>
  <td>{{.Size}}</td>
</tr>
{{end}}
</tbody>
</table>
</body>
</html>`

const detailHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Laputa / {{.Name}}</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; margin: 0; background: #0e1116; color: #c9d1d9; }
  header { background: #161b22; padding: 16px 24px; border-bottom: 1px solid #30363d; }
  header a { color: #58a6ff; text-decoration: none; font-size: 12px; }
  header h1 { margin: 8px 0; font-size: 18px; color: #c9d1d9; }
  header .meta { color: #8b949e; font-size: 12px; }
  pre { margin: 0; padding: 16px 24px; background: #161b22; color: #c9d1d9; font-family: ui-monospace, monospace; font-size: 12px; overflow-x: auto; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; margin-right: 6px; }
  .badge-stable { background: #1c3a25; color: #3fb950; }
  .badge-tbd    { background: #4a3500; color: #d29922; }
  .badge-auth   { background: #21262d; color: #58a6ff; }
</style>
</head>
<body>
<header>
  <a href="/">← back</a>
  <h1>{{.Number}} · {{.Name}}</h1>
  <div class="meta">
    <span class="badge badge-{{.Status}}">{{.Status}}</span>
    <span class="badge badge-auth">write={{.WriteAuth}}</span>
    <span class="badge badge-auth">schema={{.SchemaOwner}}</span>
    {{if .UpdatedAt}}<span class="badge badge-auth">updated={{.UpdatedAt}}</span>{{end}}
  </div>
</header>
<pre>{{.JSON}}</pre>
</body>
</html>`
