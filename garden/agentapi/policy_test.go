package agentapi

import "testing"

func TestAuthorizeRoleMatrix(t *testing.T) {
	binding := Binding{ProfileID: "default", AgentID: "host", SessionID: "session"}
	cases := []struct {
		op      Operation
		allowed []Principal
	}{
		{OpBootstrap, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator}},
		{OpSearch, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator}},
		{OpExpand, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator}},
		{OpCapture, []Principal{PrincipalUser, PrincipalAgent, PrincipalAutodream}},
		{OpRemember, []Principal{PrincipalUser, PrincipalAgent}},
		{OpPersonaGet, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator}},
		{OpPersonaPropose, []Principal{PrincipalAgent, PrincipalAutodream}},
		{OpPersonaP16, []Principal{PrincipalUser, PrincipalAgent, PrincipalAutodream}},
		{OpPersonaReview, []Principal{PrincipalUser}},
		{OpPersonaRepair, []Principal{PrincipalUser, PrincipalOperator}},
		{OpActmemRead, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalOperator}},
		{OpActmemQuery, []Principal{PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalOperator}},
		{OpActmemWrite, []Principal{PrincipalUser, PrincipalAgent}},
		{OpActmemMaintain, []Principal{PrincipalUser, PrincipalAgent}},
	}
	for _, row := range cases {
		for _, role := range []Principal{"", PrincipalRead, PrincipalUser, PrincipalAgent, PrincipalAutodream, PrincipalOperator, "admin"} {
			err := Authorize("default", binding, role, row.op)
			want := false
			for _, p := range row.allowed {
				if role == p {
					want = true
				}
			}
			if (err == nil) != want {
				t.Errorf("%s/%s allowed=%v err=%v", row.op, role, want, err)
			}
		}
	}
}

func TestAuthorizeFailsClosedForProfileAndUnknownOperation(t *testing.T) {
	for _, tc := range []struct{ configured, supplied string }{
		{"", "default"}, {"default", ""}, {"default", "other"}, {"default", " default "},
	} {
		if err := Authorize(tc.configured, Binding{ProfileID: tc.supplied}, PrincipalAgent, OpBootstrap); err == nil || err.Code != "profile_mismatch" {
			t.Errorf("profile %q/%q: %v", tc.configured, tc.supplied, err)
		}
	}
	if err := Authorize("default", Binding{ProfileID: "default"}, PrincipalAgent, Operation("approve_all")); err == nil || err.Code != "principal_forbidden" {
		t.Errorf("unknown operation: %v", err)
	}
	if err := Authorize("default", Binding{ProfileID: "default"}, PrincipalAgent, OpPersonaReview); err == nil || err.Code != "principal_forbidden" {
		t.Errorf("actor must not elevate principal: %v", err)
	}
	if err := Authorize("default", Binding{ProfileID: "default"}, "", OpBootstrap); err == nil || err.Code != "authentication_required" {
		t.Errorf("no implicit loopback privilege: %v", err)
	}
}
