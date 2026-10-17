package engine

import (
	"github.com/lennon-li/HMA/internal/model"
	"testing"
)

func TestGateBCriticalWeakRule(t *testing.T) {
	req := validCoherenceRequest()
	req.Risk = model.RouteRiskCritical
	req.Policy.IndependenceRules = append(req.Policy.IndependenceRules, model.RouteIndependenceRule{ID: "weak-low", MinimumRisk: model.RouteRiskLow, RequiredValidatorCapabilities: []string{model.CapabilitySemanticReview}})
	req.PolicyDigest = EvaluateRoutePolicy(RoutePolicyRequest{Policy: req.Policy}).Digest
	req.Profiles[2].ProviderFamily = req.Profiles[0].ProviderFamily
	req.Profiles[2].ModelFamily = req.Profiles[0].ModelFamily
	req.ValidatorRoutes[0] = coherenceProposal(req.Profiles[2])
	req.IndependenceRuleID = "independent-high"
	if got := EvaluateRouteCoherence(req); got.Decision != DecisionRejected {
		t.Fatalf("control did not reject: %+v", got)
	}
	req.IndependenceRuleID = "weak-low"
	if got := EvaluateRouteCoherence(req); got.Decision != DecisionRejected {
		t.Errorf("CRITICAL same-family validator accepted by weak rule: %+v", got)
	}
}
