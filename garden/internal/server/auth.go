package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
)

// Principal is the authenticated local capability class. Actor labels are
// intentionally separate and never participate in this decision.
type Principal string

const (
	PrincipalRead      Principal = "read"
	PrincipalUser      Principal = "user"
	PrincipalAgent     Principal = "agent"
	PrincipalAutodream Principal = "autodream"
	PrincipalOperator  Principal = "operator"
)

// CapabilityConfig maps local bearer capabilities to principals. The token
// values are compared by digest in constant time and are never logged.
type CapabilityConfig struct {
	ReadToken      string
	UserToken      string
	AgentToken     string
	AutodreamToken string
	OperatorToken  string
}

// CapabilityTokens is an intentionally discoverable alias for integrations.
type CapabilityTokens = CapabilityConfig

var (
	errAuthenticationRequired = errors.New("capability token required")
	errInvalidCapability      = errors.New("invalid capability token")
)

func capabilityConfigFromEnv() CapabilityConfig {
	return CapabilityConfig{
		ReadToken:      strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_READ_TOKEN")),
		UserToken:      strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_USER_TOKEN")),
		AgentToken:     strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_AGENT_TOKEN")),
		AutodreamToken: strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_AUTODREAM_TOKEN")),
		OperatorToken:  strings.TrimSpace(os.Getenv("GARDEN_CAPABILITY_OPERATOR_TOKEN")),
	}
}

func (c CapabilityConfig) configured() bool {
	return c.ReadToken != "" || c.UserToken != "" || c.AgentToken != "" || c.AutodreamToken != "" || c.OperatorToken != ""
}

func (c CapabilityConfig) principalForToken(token string) (Principal, error) {
	provided := sha256.Sum256([]byte(token))
	checks := []struct {
		principal Principal
		token     string
	}{
		{PrincipalRead, c.ReadToken},
		{PrincipalUser, c.UserToken},
		{PrincipalAgent, c.AgentToken},
		{PrincipalAutodream, c.AutodreamToken},
		{PrincipalOperator, c.OperatorToken},
	}
	var matched Principal
	for _, check := range checks {
		if check.token == "" {
			continue
		}
		expected := sha256.Sum256([]byte(check.token))
		if subtle.ConstantTimeCompare(provided[:], expected[:]) == 1 {
			if matched != "" {
				return "", errInvalidCapability
			}
			matched = check.principal
		}
	}
	if matched == "" {
		return "", errInvalidCapability
	}
	return matched, nil
}

func (s *Server) capabilityConfig() CapabilityConfig {
	if s.Capabilities.configured() {
		return s.Capabilities
	}
	return capabilityConfigFromEnv()
}

func (s *Server) principalForRequest(r *http.Request) (Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		if r.Method == http.MethodGet && isLoopbackRequest(r) {
			return PrincipalRead, nil
		}
		return "", errAuthenticationRequired
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errInvalidCapability
	}
	return s.capabilityConfig().principalForToken(strings.TrimSpace(parts[1]))
}

// principalForReadRequest applies the documented local read policy to both
// GET resources and read-only POST query endpoints. It never broadens a write
// route: callers must opt into this helper only for read-only operations.
func (s *Server) principalForReadRequest(r *http.Request) (Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" && isLoopbackRequest(r) {
		return PrincipalRead, nil
	}
	return s.principalForRequest(r)
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) requirePrincipal(w http.ResponseWriter, r *http.Request, allowed ...Principal) (Principal, bool) {
	principal, err := s.principalForRequest(r)
	if err != nil {
		if errors.Is(err, errAuthenticationRequired) {
			writeErrorWithCode(w, http.StatusUnauthorized, "authentication_required", err)
		} else {
			writeErrorWithCode(w, http.StatusUnauthorized, "invalid_capability_token", err)
		}
		return "", false
	}
	for _, candidate := range allowed {
		if principal == candidate {
			return principal, true
		}
	}
	writeErrorWithCode(w, http.StatusForbidden, "principal_forbidden", errors.New("principal is not permitted for this operation"))
	return "", false
}

func (s *Server) requireReadPrincipal(w http.ResponseWriter, r *http.Request, allowed ...Principal) (Principal, bool) {
	principal, err := s.principalForReadRequest(r)
	if err != nil {
		if errors.Is(err, errAuthenticationRequired) {
			writeErrorWithCode(w, http.StatusUnauthorized, "authentication_required", err)
		} else {
			writeErrorWithCode(w, http.StatusUnauthorized, "invalid_capability_token", err)
		}
		return "", false
	}
	for _, candidate := range allowed {
		if principal == candidate {
			return principal, true
		}
	}
	writeErrorWithCode(w, http.StatusForbidden, "principal_forbidden", errors.New("principal is not permitted for this operation"))
	return "", false
}

func auditLabel(r *http.Request, principal Principal) string {
	if actor := strings.TrimSpace(r.Header.Get("X-Garden-Actor")); actor != "" {
		return actor
	}
	if principal == "" {
		return "user_request"
	}
	return string(principal)
}
