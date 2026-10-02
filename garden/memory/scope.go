package memory

import (
	"encoding/hex"
	"strings"

	"github.com/dashimaki/laputa/evolution"
)

// Scope encoding for backend records (spec section 6.2): versioned,
// unambiguous, exact-match only. Subject/workspace IDs are opaque host
// identifiers, so they are hex-encoded — no delimiter can ever collide.
// Records with an empty or undecodable scope value are unclassified: they
// stay stored but invisible to every scoped reader until the owner
// classifies them.
const scopeVersion = "scope/v1"

// EncodeScope renders a validated scope for canonical storage.
func EncodeScope(scope evolution.Scope) string {
	subject := hex.EncodeToString([]byte(scope.SubjectID))
	if scope.Kind == evolution.ScopeWorkspace {
		return scopeVersion + ":workspace:" + subject + ":" + hex.EncodeToString([]byte(scope.WorkspaceID))
	}
	return scopeVersion + ":personal:" + subject
}

// DecodeScope parses a stored scope encoding; anything else is unclassified.
func DecodeScope(raw string) (evolution.Scope, error) {
	parts := strings.Split(raw, ":")
	if len(parts) < 3 || parts[0] != scopeVersion {
		return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
	}
	subject, err := hex.DecodeString(parts[2])
	if err != nil || len(subject) == 0 {
		return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
	}
	scope := evolution.Scope{SubjectID: string(subject)}
	switch parts[1] {
	case "personal":
		if len(parts) != 3 {
			return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
		}
		scope.Kind = evolution.ScopePersonal
	case "workspace":
		if len(parts) != 4 {
			return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
		}
		workspace, err := hex.DecodeString(parts[3])
		if err != nil || len(workspace) == 0 {
			return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
		}
		scope.Kind = evolution.ScopeWorkspace
		scope.WorkspaceID = string(workspace)
	default:
		return evolution.Scope{}, &evolution.ContractError{Code: evolution.ErrInvalidScope, Message: "unclassified scope encoding"}
	}
	if err := scope.Validate(); err != nil {
		return evolution.Scope{}, err
	}
	return scope, nil
}

// ScopeVisible reports whether a stored scope string decodes to one of the
// admitted read scopes (exact tuple match, never prefix).
func ScopeVisible(raw string, admitted []evolution.Scope) bool {
	if raw == "" {
		return false
	}
	decoded, err := DecodeScope(raw)
	if err != nil {
		return false
	}
	for _, scope := range admitted {
		if decoded.Equal(scope) {
			return true
		}
	}
	return false
}
