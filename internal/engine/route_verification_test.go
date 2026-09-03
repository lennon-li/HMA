package engine

import (
	"testing"
)

type routeVerificationCase struct {
	Name           string                  `json:"name"`
	Request        RouteAttestationRequest `json:"request"`
	ExpectDecision Decision                `json:"expect_decision"`
	ExpectReason   Reason                  `json:"expect_reason"`
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
		decision, reason := EvaluateRouteAttestation(tc.Request)
		if decision != tc.ExpectDecision || reason != tc.ExpectReason {
			t.Errorf("%s: EvaluateRouteAttestation() = (%s, %s), want (%s, %s)", tc.Name, decision, reason, tc.ExpectDecision, tc.ExpectReason)
		}
		seen[string(reason)] = true
	}
	for _, rule := range requiredRouteVerificationRules {
		if !seen[rule] {
			t.Errorf("missing fixture for route verification reason %s", rule)
		}
	}
}
