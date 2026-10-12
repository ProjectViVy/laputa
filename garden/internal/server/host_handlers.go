package server

import (
	"errors"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"net/http"
	"strings"
)

func (s *Server) handleHostProjectionPreview(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var in agentapi.HostProjectionRequest
	if err := decodeJSON(w, r, 8192, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	if in.BindingID != "" && in.BindingID != r.PathValue("id") {
		writeErrorWithCode(w, 400, "invalid_binding", errors.New("binding mismatch"))
		return
	}
	in.BindingID = r.PathValue("id")
	plan, err := s.Installation.PreviewHostProjection(r.Context(), in)
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, plan)
}

func writeHostResult(w http.ResponseWriter, status agentapi.HostStatus, err error) {
	if err != nil {
		var domain *agentapi.Error
		if errors.As(err, &domain) && status.State != "" {
			domain.Details["host_status"] = status
		}
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, status)
}
func (s *Server) handleHostProjectionStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	status, err := s.Installation.HostProjectionStatus(r.Context(), r.PathValue("id"))
	writeHostResult(w, status, err)
}
func (s *Server) handleHostProjectionApply(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var plan agentapi.HostFilePlan
	if err := decodeJSON(w, r, 8<<20, &plan); err != nil {
		writeRequestError(w, err)
		return
	}
	if plan.BindingID != r.PathValue("id") {
		writeErrorWithCode(w, 400, "invalid_binding", errors.New("plan binding mismatch"))
		return
	}
	status, err := s.Installation.ApplyHostProjection(r.Context(), plan)
	writeHostResult(w, status, err)
}
func (s *Server) handleHostProjectionRevoke(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	status, err := s.Installation.RevokeHostProjection(r.Context(), r.PathValue("id"))
	writeHostResult(w, status, err)
}
func (s *Server) handleHostProjectionLoaded(w http.ResponseWriter, r *http.Request) {
	// Owner cookie, claimed profile and generic hook success are not load proof.
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
	if binding.BindingID != r.PathValue("id") {
		writeErrorWithCode(w, 403, "principal_forbidden", errors.New("load evidence binding mismatch"))
		return
	}
	session := strings.TrimSpace(r.Header.Get("X-Garden-Session"))
	if session == "" {
		writeErrorWithCode(w, 400, "invalid_binding", errors.New("authenticated host session required"))
		return
	}
	var in struct {
		Hash string `json:"hash"`
	}
	if err = decodeJSON(w, r, 1024, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	status, err := s.Installation.AcknowledgeHostLoaded(r.Context(), binding.BindingID, session, in.Hash)
	writeHostResult(w, status, err)
}
