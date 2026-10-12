package server

import (
	"errors"
	"github.com/ProjectViVy/laputa/garden/agentapi"
	"net/http"
	"strings"
)

// boundRoute creates an immutable request view. Owner selection never mutates
// the listener. Shutdown drains admitted requests before closing authority.
func (s *Server) boundRoute(op agentapi.Operation, optional bool, f func(*Server, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Installation == nil {
			f(s, w, r)
			return
		}
		p, ok := s.requirePrincipal(w, r, PrincipalUser, PrincipalAgent)
		if !ok {
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v2/admin/") && p != PrincipalUser {
			writeErrorWithCode(w, 503, "unavailable", errors.New("installation-wide admin projections require owner access"))
			return
		}
		if optional {
			writeErrorWithCode(w, http.StatusServiceUnavailable, "unavailable", errors.New("route unavailable in multi-persona installation"))
			return
		}
		session := r.Header.Get("X-Garden-Session")
		if session == "" {
			session = "garden-http-read"
		}
		profile, workspace := r.Header.Get("X-Garden-Profile"), r.Header.Get("X-Garden-Workspace")
		var domain *agentapi.Service
		var binding agentapi.Binding
		var err error
		if p == PrincipalUser {
			if profile == "" {
				profiles, e := s.Installation.ListProfiles(r.Context())
				if e != nil {
					writeMaterialsReadError(w, e)
					return
				}
				if len(profiles) == 1 {
					profile = profiles[0].ProfileID
				}
			}
			if profile == "" {
				writeErrorWithCode(w, 400, "invalid_binding", errors.New("select a profile"))
				return
			}
			c, e := s.Installation.BindProfile(profile, agentapi.ClientConfig{AgentID: "garden-console", Platform: "http", WorkspaceID: workspace, Principal: agentapi.PrincipalUser})
			if e != nil {
				writeMaterialsReadError(w, e)
				return
			}
			defer c.Close()
			domain, binding, err = c.AdmittedService(session)
		} else {
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			host, e := s.Installation.AuthenticateHost(r.Context(), strings.TrimSpace(parts[1]))
			if e != nil {
				writeMaterialsReadError(w, e)
				return
			}
			if (profile != "" && profile != host.ProfileID) || (workspace != "" && workspace != host.WorkspaceID) {
				writeErrorWithCode(w, 403, "invalid_binding", errors.New("request conflicts with host binding"))
				return
			}
			b, e := s.Installation.BindHostSession(host.BindingID, session)
			if e != nil {
				writeMaterialsReadError(w, e)
				return
			}
			domain, binding = b.AdmittedService()
		}
		if err != nil {
			writeMaterialsReadError(w, err)
			return
		}
		if err = domain.Check(r.Context(), agentapi.Principal(p), binding, op); err != nil {
			writeMaterialsReadError(w, err)
			return
		}
		if s.Owner == nil {
			writeErrorWithCode(w, 503, "unavailable", errors.New("installation owner unavailable"))
			return
		}
		subject, err := s.Owner.Subject(r.Context(), binding.ProfileID)
		if err != nil {
			writeMaterialsReadError(w, err)
			return
		}
		unlock, err := s.Owner.Enter()
		if err != nil {
			writeMaterialsReadError(w, err)
			return
		}
		defer unlock()
		v := *s
		v.AgentAPI = domain
		v.ProfileID = binding.ProfileID
		v.binding = binding
		v.Persona = subject.Persona
		v.Actmem = subject.Actmem
		v.ActmemWorkspace = binding.WorkspaceID
		v.Ingestions = nil
		v.Activity = nil
		v.TraceStore = nil
		f(&v, w, r)
	}
}
func (s *Server) requestBinding(sessionID string) agentapi.Binding {
	if s.Installation != nil {
		b := s.binding
		if sessionID != "" {
			b.SessionID = sessionID
		}
		return b
	}
	return agentapi.Binding{ProfileID: s.ProfileID, AgentID: "garden-http", Platform: "http", SessionID: sessionID}
}
