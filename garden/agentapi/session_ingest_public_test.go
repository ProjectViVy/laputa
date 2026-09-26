package agentapi_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/dashimaki/garden/agentapi"
)

// A package outside agentapi can compile against the public request/receipt
// without importing garden/internal or relying on Go's internal import rules.
var _ func(*agentapi.Service, context.Context, agentapi.Principal, agentapi.SessionSubmitRequest) (agentapi.SessionAccepted, error) = (*agentapi.Service).SubmitSession

func TestSessionSubmitPublicSignatureDoesNotExposeInternalTypes(t *testing.T) {
	method, ok := reflect.TypeOf((*agentapi.Service)(nil)).MethodByName("SubmitSession")
	if !ok {
		t.Fatal("public method unavailable")
	}
	if strings.Contains(method.Type.String(), "/internal/") {
		t.Fatalf("internal DTO leaked: %s", method.Type)
	}
}
