package engine

import (
	"reflect"
	"testing"

	"github.com/lennon-li/HMA/internal/model"
)

type routePolicyFixtureCase struct {
	Name           string             `json:"name"`
	Request        RoutePolicyRequest `json:"request"`
	ExpectDecision Decision           `json:"expect_decision"`
	ExpectReason   Reason             `json:"expect_reason"`
}

func TestRoutePolicyFixtures(t *testing.T) {
	var cases []routePolicyFixtureCase
	loadFixture(t, "../../testdata/engine/route-policy.json", &cases)
	if len(cases) == 0 {
		t.Fatal("route policy fixture is empty")
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := EvaluateRoutePolicy(tc.Request)
			if got.Decision != tc.ExpectDecision || got.Reason != tc.ExpectReason {
				t.Fatalf("EvaluateRoutePolicy() = (%s, %s), want (%s, %s)", got.Decision, got.Reason, tc.ExpectDecision, tc.ExpectReason)
			}
			if got.MachineAdvanced {
				t.Fatal("route-policy evaluation reported machine advancement")
			}
			if !got.RequiresFreshHumanApproval {
				t.Fatal("route-policy evaluation did not require fresh human approval")
			}
			if tc.ExpectDecision == DecisionLegalPendingApproval && got.Digest == "" {
				t.Fatal("valid route policy returned no digest")
			}
		})
	}
}

func TestRoutePolicyEquivalentOrderingHasSameDigest(t *testing.T) {
	first := model.RoutePolicy{
		Version:           model.RoutePolicyVersionV1,
		CapabilityClasses: []string{model.CapabilitySemanticReview, model.CapabilityPlanning},
		PermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionReadOnly},
		RiskRules: []model.RouteRiskRule{{
			ID: "r", MinimumRisk: model.RouteRiskHigh,
			RequiredCapabilityClasses: []string{model.CapabilitySemanticReview, model.CapabilityPlanning},
		}},
	}
	second := first
	second.CapabilityClasses = []string{model.CapabilityPlanning, model.CapabilitySemanticReview}
	second.PermissionClasses = []string{model.PermissionReadOnly, model.PermissionBoundedWrite}
	second.RiskRules[0].RequiredCapabilityClasses = []string{model.CapabilityPlanning, model.CapabilitySemanticReview}

	firstResult := EvaluateRoutePolicy(RoutePolicyRequest{Policy: first})
	secondResult := EvaluateRoutePolicy(RoutePolicyRequest{Policy: second})
	if firstResult.Decision != DecisionLegalPendingApproval || secondResult.Decision != DecisionLegalPendingApproval {
		t.Fatalf("equivalent policies were rejected: %#v %#v", firstResult, secondResult)
	}
	if firstResult.Digest != secondResult.Digest {
		t.Fatalf("equivalent policy digests differ: %s != %s", firstResult.Digest, secondResult.Digest)
	}
	if !reflect.DeepEqual(firstResult.Projection, secondResult.Projection) {
		t.Fatal("equivalent policies produced different projections")
	}
}

func TestRoutePolicyMaterialChangeChangesDigest(t *testing.T) {
	policy := model.RoutePolicy{
		Version:           model.RoutePolicyVersionV1,
		CapabilityClasses: []string{model.CapabilityPlanning},
		PermissionClasses: []string{model.PermissionReadOnly},
	}
	first := EvaluateRoutePolicy(RoutePolicyRequest{Policy: policy})
	policy.CapabilityClasses = append(policy.CapabilityClasses, model.CapabilityDebugging)
	second := EvaluateRoutePolicy(RoutePolicyRequest{Policy: policy})
	if first.Digest == second.Digest {
		t.Fatalf("material policy change preserved digest %s", first.Digest)
	}
}

func TestRoutePolicyExpectedDigestDetectsDrift(t *testing.T) {
	policy := model.RoutePolicy{Version: model.RoutePolicyVersionV1}
	current := EvaluateRoutePolicy(RoutePolicyRequest{Policy: policy})
	got := EvaluateRoutePolicy(RoutePolicyRequest{Policy: policy, ExpectedDigest: "sha256:stale"})
	if current.Digest == "" || got.Digest == "" {
		t.Fatal("expected digests")
	}
	if got.Decision != DecisionRejected || got.Reason != ReasonRoutePolicyDigestMismatch {
		t.Fatalf("drift result = %#v", got)
	}
}
