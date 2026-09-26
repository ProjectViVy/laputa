// Package agentapi defines Garden's public transport-neutral Agent contract.
package agentapi

// Principal is supplied by trusted authentication, never by Binding or actor metadata.
type Principal string

const (
	PrincipalRead      Principal = "read"
	PrincipalUser      Principal = "user"
	PrincipalAgent     Principal = "agent"
	PrincipalAutodream Principal = "autodream"
	PrincipalOperator  Principal = "operator"
)

// Operation names are a closed policy vocabulary, not URL paths.
type Operation string

const (
	OpBootstrap      Operation = "bootstrap"
	OpSearch         Operation = "search"
	OpExpand         Operation = "expand"
	OpCapture        Operation = "capture"
	OpRemember       Operation = "remember"
	OpPersonaGet     Operation = "persona_get"
	OpPersonaPropose Operation = "persona_propose"
	OpPersonaP16     Operation = "persona_p16"
	OpPersonaReview  Operation = "persona_review"
	OpPersonaRepair  Operation = "persona_repair"
	OpActmemRead     Operation = "actmem_read"
	OpActmemQuery    Operation = "actmem_query"
	OpActmemWrite    Operation = "actmem_write"
	OpActmemMaintain Operation = "actmem_maintain"
)

// Authorize checks a trusted principal and exact server-owned profile ID.
// Authentication and loopback detection belong to the adapter, not this policy.
func Authorize(configuredProfile string, binding Binding, principal Principal, operation Operation) *Error {
	if configuredProfile == "" || binding.ProfileID == "" || binding.ProfileID != configuredProfile {
		return policyError("profile_mismatch", "profile does not match configured profile")
	}
	if principal == "" {
		return policyError("authentication_required", "capability token required")
	}
	allowed := false
	switch operation {
	case OpBootstrap, OpSearch, OpExpand, OpPersonaGet:
		allowed = principal == PrincipalRead || principal == PrincipalUser || principal == PrincipalAgent || principal == PrincipalAutodream || principal == PrincipalOperator
	case OpCapture:
		allowed = principal == PrincipalUser || principal == PrincipalAgent || principal == PrincipalAutodream
	case OpRemember, OpActmemWrite, OpActmemMaintain:
		allowed = principal == PrincipalUser || principal == PrincipalAgent
	case OpPersonaPropose:
		allowed = principal == PrincipalAgent || principal == PrincipalAutodream
	case OpPersonaP16:
		allowed = principal == PrincipalUser || principal == PrincipalAgent || principal == PrincipalAutodream
	case OpPersonaReview:
		allowed = principal == PrincipalUser
	case OpPersonaRepair:
		allowed = principal == PrincipalUser || principal == PrincipalOperator
	case OpActmemRead, OpActmemQuery:
		allowed = principal == PrincipalRead || principal == PrincipalUser || principal == PrincipalAgent || principal == PrincipalOperator
	}
	if !allowed {
		return policyError("principal_forbidden", "principal is not permitted for this operation")
	}
	return nil
}

func policyError(code, message string) *Error {
	return &Error{Code: code, Message: message, LegacyError: message, Details: map[string]any{}}
}
