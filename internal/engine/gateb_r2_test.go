package engine

import (
	"github.com/lennon-li/HMA/internal/model"
	"testing"
)

func TestGateBR2IndependenceTiedRule(t *testing.T) {
	r := validCoherenceRequest()
	strict := r.Policy.IndependenceRules[0]
	strict.ID = "strict-same-risk"
	strict.RequiredValidatorCapabilities = append(strict.RequiredValidatorCapabilities, model.CapabilityRepositoryScanning)
	r.Policy.IndependenceRules = append(r.Policy.IndependenceRules, strict)
	r.PolicyDigest = EvaluateRoutePolicy(RoutePolicyRequest{Policy: r.Policy}).Digest
	r.IndependenceRuleID = strict.ID
	if got := EvaluateRouteCoherence(r); got.Decision != DecisionRejected {
		t.Fatalf("strict control: %+v", got)
	}
	r.IndependenceRuleID = "independent-high"
	if got := EvaluateRouteCoherence(r); got.Decision != DecisionRejected {
		t.Errorf("same-risk weaker rule accepted despite missing mandatory capability: %+v", got)
	}
}
func TestGateBR2IndependenceRiskControls(t *testing.T) {
	for _, risk := range []model.RouteRiskLevel{model.RouteRiskHigh, model.RouteRiskCritical} {
		for _, weak := range []model.RouteRiskLevel{model.RouteRiskLow, model.RouteRiskMedium} {
			r := validCoherenceRequest()
			r.Risk = risk
			r.Policy.IndependenceRules = append(r.Policy.IndependenceRules, model.RouteIndependenceRule{ID: "weak", MinimumRisk: weak, RequiredValidatorCapabilities: []string{model.CapabilitySemanticReview}})
			r.PolicyDigest = EvaluateRoutePolicy(RoutePolicyRequest{Policy: r.Policy}).Digest
			r.IndependenceRuleID = "weak"
			if got := EvaluateRouteCoherence(r); got.Decision != DecisionRejected {
				t.Errorf("%s weak %s: %+v", risk, weak, got)
			}
			r.IndependenceRuleID = "independent-high"
			if got := EvaluateRouteCoherence(r); got.Decision != DecisionLegalPendingApproval {
				t.Errorf("valid separated route: %+v", got)
			}
		}
	}
}
