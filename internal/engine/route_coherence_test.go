package engine

import (
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

type routeCoherenceFixtureCase struct {
	Name    string                           `json:"name"`
	Rule    string                           `json:"rule"`
	Request RouteCoherenceRequest            `json:"request"`
	Expect  routeCoherenceFixtureExpectation `json:"expect"`
}

type routeCoherenceFixtureExpectation struct {
	Decision                Decision `json:"decision"`
	Reason                  Reason   `json:"reason"`
	PrimaryProfileDigest    string   `json:"primary_profile_digest,omitempty"`
	EscalationProfileDigest string   `json:"escalation_profile_digest,omitempty"`
	ValidatorProfileDigest  string   `json:"validator_profile_digest,omitempty"`
}

var requiredRouteCoherenceRules = []string{"V1", "I1", "I2", "I3", "I4", "I5", "I6", "I7", "I8", "I9", "I10", "I11"}

func assertRouteCoherenceRuleCoverage(t *testing.T, cases []routeCoherenceFixtureCase) {
	t.Helper()
	seen := map[string]bool{}
	for _, tc := range cases {
		if tc.Rule == "" {
			t.Fatalf("fixture case %q declares no rule", tc.Name)
		}
		if seen[tc.Rule] {
			t.Fatalf("fixture rule %s is duplicated", tc.Rule)
		}
		seen[tc.Rule] = true
	}
	for _, rule := range requiredRouteCoherenceRules {
		if !seen[rule] {
			t.Errorf("fixture matrix rule %s has no case", rule)
		}
		delete(seen, rule)
	}
	for rule := range seen {
		t.Errorf("fixture declares rule %s outside the approved matrix", rule)
	}
}

func assertExpectedRouteProfile(t *testing.T, role string, routes []model.RouteProposal, expected string) {
	t.Helper()
	if expected == "" {
		return
	}
	if len(routes) != 1 {
		t.Fatalf("expected one %s route for profile %q, got %d", role, expected, len(routes))
	}
	if routes[0].ProfileDigest != expected {
		t.Fatalf("%s route profile = %q, want %q", role, routes[0].ProfileDigest, expected)
	}
}

func TestRouteCoherenceFixtures(t *testing.T) {
	var cases []routeCoherenceFixtureCase
	loadFixture(t, "../../testdata/engine/route-coherence.json", &cases)
	if len(cases) == 0 {
		t.Fatal("route coherence fixture is empty")
	}
	assertRouteCoherenceRuleCoverage(t, cases)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req := tc.Request
			if req.PolicyDigest == "" {
				req.PolicyDigest = EvaluateRoutePolicy(RoutePolicyRequest{Policy: req.Policy}).Digest
			}
			got := EvaluateRouteCoherence(req)
			if got.Decision != tc.Expect.Decision || got.Reason != tc.Expect.Reason {
				t.Fatalf("EvaluateRouteCoherence() = (%s, %s), want (%s, %s)", got.Decision, got.Reason, tc.Expect.Decision, tc.Expect.Reason)
			}
			if got.MachineAdvanced || !got.RequiresFreshHumanApproval {
				t.Fatalf("authority flags = advanced:%v fresh:%v", got.MachineAdvanced, got.RequiresFreshHumanApproval)
			}
			assertExpectedRouteProfile(t, "primary", req.PrimaryRoutes, tc.Expect.PrimaryProfileDigest)
			assertExpectedRouteProfile(t, "escalation", req.EscalationRoutes, tc.Expect.EscalationProfileDigest)
			assertExpectedRouteProfile(t, "validator", req.ValidatorRoutes, tc.Expect.ValidatorProfileDigest)
		})
	}
}

func coherencePolicy() model.RoutePolicy {
	return model.RoutePolicy{
		Version: model.RoutePolicyVersionV1,
		CapabilityClasses: []string{
			model.CapabilityPlanning,
			model.CapabilitySemanticReview,
			model.CapabilityRepositoryScanning,
		},
		PermissionClasses: []string{model.PermissionReadOnly, model.PermissionBoundedWrite},
		IndependenceRules: []model.RouteIndependenceRule{{
			ID:                            "independent-high",
			MinimumRisk:                   model.RouteRiskHigh,
			RequiredValidatorCapabilities: []string{model.CapabilitySemanticReview},
			DistinctProviderFamily:        true,
			DistinctModelFamily:           true,
		}},
	}
}

func coherenceProfiles() []model.RouteProfile {
	return []model.RouteProfile{
		{ProfileDigest: "sha256:primary", ProviderFamily: "provider-a", ModelFamily: "model-a", ExecutionContext: model.ExecutionContextDirectAPI, AccessServiceDigest: "sha256:access", CapabilityClasses: []string{model.CapabilityPlanning, model.CapabilitySemanticReview}, PermissionClasses: []string{model.PermissionReadOnly}, Eligible: true},
		{ProfileDigest: "sha256:escalation", ProviderFamily: "provider-b", ModelFamily: "model-b", ExecutionContext: model.ExecutionContextDirectAPI, AccessServiceDigest: "sha256:access", CapabilityClasses: []string{model.CapabilityPlanning, model.CapabilitySemanticReview}, PermissionClasses: []string{model.PermissionReadOnly}, Eligible: true},
		{ProfileDigest: "sha256:validator", ProviderFamily: "provider-c", ModelFamily: "model-c", ExecutionContext: model.ExecutionContextDirectAPI, AccessServiceDigest: "sha256:access", CapabilityClasses: []string{model.CapabilitySemanticReview}, PermissionClasses: []string{model.PermissionReadOnly}, Eligible: true},
	}
}

func coherenceProposal(profile model.RouteProfile) model.RouteProposal {
	return model.RouteProposal{
		ProfileDigest:     profile.ProfileDigest,
		ProviderFamily:    profile.ProviderFamily,
		ModelFamily:       profile.ModelFamily,
		ExecutionContext:  profile.ExecutionContext,
		CapabilityClasses: append([]string(nil), profile.CapabilityClasses...),
		PermissionClasses: append([]string(nil), profile.PermissionClasses...),
	}
}

func validCoherenceRequest() RouteCoherenceRequest {
	profiles := coherenceProfiles()
	policy := coherencePolicy()
	policyResult := EvaluateRoutePolicy(RoutePolicyRequest{Policy: policy})
	return RouteCoherenceRequest{
		PlanUnitID:                  "U1",
		Risk:                        model.RouteRiskHigh,
		RequiredCapabilityClasses:   []string{model.CapabilitySemanticReview},
		ApprovedPermissionClasses:   []string{model.PermissionReadOnly},
		Policy:                      policy,
		PolicyDigest:                policyResult.Digest,
		IndependenceRuleID:          "independent-high",
		ApprovedAccessServiceDigest: "sha256:access",
		Profiles:                    profiles,
		PrimaryRoutes:               []model.RouteProposal{coherenceProposal(profiles[0])},
		EscalationRoutes:            []model.RouteProposal{coherenceProposal(profiles[1])},
		ValidatorRoutes:             []model.RouteProposal{coherenceProposal(profiles[2])},
	}
}

func TestEvaluateRouteCoherence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RouteCoherenceRequest)
		want   Reason
	}{
		{name: "valid proposal", want: ReasonNone},
		{name: "multiple primary routes", mutate: func(req *RouteCoherenceRequest) { req.PrimaryRoutes = append(req.PrimaryRoutes, req.PrimaryRoutes[0]) }, want: ReasonRouteCoherenceProposalCount},
		{name: "same escalation profile", mutate: func(req *RouteCoherenceRequest) { req.EscalationRoutes[0] = req.PrimaryRoutes[0] }, want: ReasonRouteCoherenceEscalationNotDistinct},
		{name: "validator reuses implementer", mutate: func(req *RouteCoherenceRequest) { req.ValidatorRoutes[0] = req.PrimaryRoutes[0] }, want: ReasonRouteCoherenceValidatorNotIndependent},
		{name: "access service substitution", mutate: func(req *RouteCoherenceRequest) { req.Profiles[1].AccessServiceDigest = "sha256:other-access" }, want: ReasonRouteCoherenceAccessServiceMismatch},
		{name: "permission escalation", mutate: func(req *RouteCoherenceRequest) {
			req.Profiles[0].PermissionClasses = []string{model.PermissionBoundedWrite}
			req.PrimaryRoutes[0].PermissionClasses = []string{model.PermissionBoundedWrite}
		}, want: ReasonRouteCoherencePermissionEscalation},
		{name: "required capability missing", mutate: func(req *RouteCoherenceRequest) {
			req.PrimaryRoutes[0].CapabilityClasses = []string{model.CapabilityPlanning}
			req.Profiles[0].CapabilityClasses = []string{model.CapabilityPlanning}
		}, want: ReasonRouteCoherenceRequiredCapabilityMissing},
		{name: "no eligible route", mutate: func(req *RouteCoherenceRequest) {
			req.Profiles = nil
			req.PrimaryRoutes = nil
			req.EscalationRoutes = nil
			req.ValidatorRoutes = nil
		}, want: ReasonRouteCoherenceNoEligibleRoute},
		{name: "forbidden fallback", mutate: func(req *RouteCoherenceRequest) {
			req.Policy.ForbiddenFallbacks = []model.RouteFallbackRule{{ID: "no-review-substitution", FromCapabilityClass: model.CapabilitySemanticReview, ToCapabilityClass: model.CapabilityRepositoryScanning}}
			req.PolicyDigest = EvaluateRoutePolicy(RoutePolicyRequest{Policy: req.Policy}).Digest
			req.PrimaryRoutes[0].CapabilityClasses = []string{model.CapabilityRepositoryScanning}
			req.Profiles[0].CapabilityClasses = []string{model.CapabilityRepositoryScanning}
		}, want: ReasonRouteCoherenceForbiddenFallback},
		{name: "validator provider not independent", mutate: func(req *RouteCoherenceRequest) {
			req.Profiles[2].ProviderFamily = req.Profiles[0].ProviderFamily
			req.ValidatorRoutes[0].ProviderFamily = req.Profiles[0].ProviderFamily
		}, want: ReasonRouteCoherenceValidatorNotIndependent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCoherenceRequest()
			if tt.mutate != nil {
				tt.mutate(&req)
			}
			got := EvaluateRouteCoherence(req)
			if got.Reason != tt.want {
				t.Fatalf("EvaluateRouteCoherence() reason = %s, want %s", got.Reason, tt.want)
			}
			if got.MachineAdvanced || !got.RequiresFreshHumanApproval {
				t.Fatalf("authority flags = advanced:%v fresh:%v", got.MachineAdvanced, got.RequiresFreshHumanApproval)
			}
			if tt.want == ReasonNone && got.Decision != DecisionLegalPendingApproval {
				t.Fatalf("decision = %s, want %s", got.Decision, DecisionLegalPendingApproval)
			}
		})
	}
}

func TestEvaluateRouteCoherenceRejectsStalePolicyDigest(t *testing.T) {
	req := validCoherenceRequest()
	req.PolicyDigest = "sha256:stale"
	got := EvaluateRouteCoherence(req)
	if got.Decision != DecisionRejected || got.Reason != ReasonRoutePolicyDigestMismatch {
		t.Fatalf("result = %#v", got)
	}
}
