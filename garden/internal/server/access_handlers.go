package server

import (
	"net/http"

	"github.com/ProjectViVy/laputa/garden/agentapi"
)

func (s *Server) mountAccess(mux *http.ServeMux) {
	if s.Installation == nil {
		return
	}
	mux.HandleFunc("POST /v2/pairing", s.handlePairing)
	mux.HandleFunc("GET /v2/session", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]bool{"paired": s.Pairing.owner(r)})
	})
	mux.HandleFunc("GET /v2/profiles", s.handleProfiles)
	mux.HandleFunc("POST /v2/profiles", s.handleProfiles)
	mux.HandleFunc("GET /v2/shared-spaces", s.handleSharedSpaces)
	mux.HandleFunc("POST /v2/shared-spaces", s.handleSharedSpaces)
	mux.HandleFunc("GET /v2/shared-spaces/{id}/members", s.handleSharedMembers)
	mux.HandleFunc("PUT /v2/shared-spaces/{id}/members/{profile_id}", s.handleSharedMembers)
	mux.HandleFunc("DELETE /v2/shared-spaces/{id}/members/{profile_id}", s.handleSharedMembers)
	mux.HandleFunc("GET /v2/host-bindings", s.handleHostBindings)
	mux.HandleFunc("POST /v2/host-bindings", s.handleHostBindings)
	mux.HandleFunc("POST /v2/host-bindings/{id}/projection/preview", s.handleHostProjectionPreview)
	mux.HandleFunc("GET /v2/host-bindings/{id}/projection", s.handleHostProjectionStatus)
	mux.HandleFunc("PUT /v2/host-bindings/{id}/projection", s.handleHostProjectionApply)
	mux.HandleFunc("DELETE /v2/host-bindings/{id}/projection", s.handleHostProjectionRevoke)
	mux.HandleFunc("POST /v2/host-bindings/{id}/projection/loaded", s.handleHostProjectionLoaded)
	mux.HandleFunc("PUT /v2/host-bindings/{id}/projection/native-config", s.handleNativeHostConfig)
	mux.HandleFunc("POST /v2/host-bindings/{id}/projection/prompt", s.handleNativeHostPrompt)
	mux.HandleFunc("POST /v2/host-bindings/{id}/projection/retry-preview", s.handleNativeHostRetryPreview)
	mux.HandleFunc("POST /v2/host-bindings/{id}/projection/retry-confirm", s.handleNativeHostRetryConfirm)
	mux.HandleFunc("PUT /v2/host-bindings/{id}/controls", s.handleHostControls)
	mux.HandleFunc("DELETE /v2/host-bindings/{id}", s.handleHostControls)
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	if r.Method == "GET" {
		profiles, err := s.Installation.ListProfiles(r.Context())
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, 200, profiles)
		return
	}
	var in agentapi.ProfileRegistration
	if err := decodeJSON(w, r, 4096, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	p, err := s.Installation.RegisterProfile(r.Context(), in)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, 201, p)
}
func (s *Server) handleSharedSpaces(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	if r.Method == "GET" {
		spaces, err := s.Installation.ListSharedSpaces(r.Context())
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, 200, spaces)
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, 1024, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	space, err := s.Installation.CreateSharedSpace(r.Context(), in.Name)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, 201, space)
}
func (s *Server) handleSharedMembers(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	space := r.PathValue("id")
	if r.Method == "GET" {
		members, err := s.Installation.ListSharedMembers(r.Context(), space)
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, 200, members)
		return
	}
	var err error
	if r.Method == "DELETE" {
		err = s.Installation.RemoveSharedMember(r.Context(), space, r.PathValue("profile_id"))
	} else {
		var in struct {
			Access string `json:"access"`
		}
		if err = decodeJSON(w, r, 1024, &in); err != nil {
			writeRequestError(w, err)
			return
		}
		err = s.Installation.SetSharedMember(r.Context(), space, r.PathValue("profile_id"), in.Access)
	}
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (s *Server) handleHostBindings(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	if r.Method == "GET" {
		bindings, err := s.Installation.ListHostBindings(r.Context())
		if err != nil {
			writeHandlerError(w, err)
			return
		}
		writeJSON(w, 200, bindings)
		return
	}
	var in agentapi.HostBinding
	if err := decodeJSON(w, r, 8192, &in); err != nil {
		writeRequestError(w, err)
		return
	}
	binding, credential, err := s.Installation.CreateHostBinding(r.Context(), in)
	if err != nil {
		writeHandlerError(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"binding": binding, "credential": credential})
}
func (s *Server) handleHostControls(w http.ResponseWriter, r *http.Request) {
	if !s.requireOwner(w, r) {
		return
	}
	var err error
	if r.Method == "DELETE" {
		err = s.Installation.RevokeHostBinding(r.Context(), r.PathValue("id"))
	} else {
		var in agentapi.HostControls
		if err = decodeJSON(w, r, 1024, &in); err != nil {
			writeRequestError(w, err)
			return
		}
		err = s.Installation.SetHostControls(r.Context(), r.PathValue("id"), in)
	}
	if err != nil {
		writeMaterialsReadError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
