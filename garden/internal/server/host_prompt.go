package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ProjectViVy/laputa/garden/agentapi"
)

func (s *Server) handleNativeHostConfig(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var cfg agentapi.NativeBindingConfig
	if err := decodeJSON(w, r, 4096, &cfg); err != nil {
		writeRequestError(w, err)
		return
	}
	status, err := s.Installation.ConfigureNativeHost(r.Context(), r.PathValue("id"), cfg)
	writeHostResult(w, status, err)
}
func (s *Server) handleNativeHostRetryPreview(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var empty struct{}
	if err := decodeJSON(w, r, 1024, &empty); err != nil {
		writeRequestError(w, err)
		return
	}
	preview, err := s.Installation.PreviewHostRetry(r.Context(), r.PathValue("id"))
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, preview)
}
func (s *Server) handleNativeHostRetryConfirm(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var body struct {
		PreviewID string `json:"preview_id"`
	}
	if err := decodeJSON(w, r, 1024, &body); err != nil {
		writeRequestError(w, err)
		return
	}
	status, err := s.Installation.ConfirmHostRetry(r.Context(), r.PathValue("id"), body.PreviewID)
	writeHostResult(w, status, err)
}
func (s *Server) handleNativeHostPrompt(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimSpace(r.Header.Get("Authorization")), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeErrorWithCode(w, 401, "authentication_required", errAuthenticationRequired)
		return
	}
	binding, err := s.Installation.AuthenticateHost(r.Context(), strings.TrimSpace(parts[1]))
	if err != nil {
		writeErrorWithCode(w, 401, "authentication_required", errAuthenticationRequired)
		return
	}
	if binding.BindingID != r.PathValue("id") || (r.Header.Get("X-Garden-Profile") != "" && r.Header.Get("X-Garden-Profile") != binding.ProfileID) || (r.Header.Get("X-Garden-Workspace") != "" && r.Header.Get("X-Garden-Workspace") != binding.WorkspaceID) {
		writeErrorWithCode(w, 403, "principal_forbidden", errors.New("prompt request conflicts with authenticated binding"))
		return
	}
	var req agentapi.NativePromptRequest
	if err = decodeJSON(w, r, 4096, &req); err != nil {
		writeRequestError(w, err)
		return
	}
	session := r.Header.Get("X-Garden-Session")
	if session == "" || session != req.SessionID {
		writeErrorWithCode(w, 400, "invalid_binding", errors.New("exact authenticated host session required"))
		return
	}
	projection, err := s.Installation.PrepareHostPrompt(r.Context(), binding.BindingID, req)
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, projection)
}
