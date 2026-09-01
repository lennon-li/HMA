package engine

import (
	"encoding/json"
	"testing"
)

type routeVerificationCase struct {
	Name    string                 `json:"name"`
	Rule    string                 `json:"rule"`
	Request RouteAttestation       `json:"request"`
	Expect  RouteVerificationResult `json:"expect"`
}

var requiredRouteVerificationRules = []string{
	"ROUTE_ATTESTATION_INCOMPLETE",
	"ROUTE_APPROVAL_MISMATCH",
	"ROUTE_PROFILE_UNKNOWN",
	"ROUTE_MANIFEST_MISMATCH",
	"ROUTE_CAPABILITY_ESCALATION",
	"ROUTE_PERMISSION_ESCALATION",
}

func TestRouteVerificationReasonCoverage(t *testing.T) {
	var cases []routeVerificationCase
	loadFixture(t, "../../testdata/engine/route-verification.json", &cases)

	seen := make(map[string]bool)
	for _, tc := range cases {
		seen[tc.Expect.Reason] = true
	}
	for _, rule := range requiredRouteVerificationRules {
		if !seen[rule] {
			t.Errorf("missing fixture for route verification reason %s", rule)
		}
	}
}
