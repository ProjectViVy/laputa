package server

import (
	"errors"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"net/http"
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
