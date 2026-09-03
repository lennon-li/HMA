package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonRouteCoherencePlanUnitMissing           Reason = "ROUTE_COHERENCE_PLAN_UNIT_MISSING"
	ReasonRouteCoherenceRiskInvalid               Reason = "ROUTE_COHERENCE_RISK_INVALID"
	ReasonRouteCoherenceCapabilityUnknown         Reason = "ROUTE_COHERENCE_CAPABILITY_UNKNOWN"
	ReasonRouteCoherencePermissionUnknown         Reason = "ROUTE_COHERENCE_PERMISSION_UNKNOWN"
	ReasonRouteCoherencePolicyDigestMissing       Reason = "ROUTE_COHERENCE_POLICY_DIGEST_MISSING"
	ReasonRouteCoherencePolicyRuleMissing         Reason = "ROUTE_COHERENCE_INDEPENDENCE_RULE_MISSING"
	ReasonRouteCoherenceAccessServiceMissing      Reason = "ROUTE_COHERENCE_ACCESS_SERVICE_MISSING"
	ReasonRouteCoherenceAccessServiceMismatch     Reason = "ROUTE_COHERENCE_ACCESS_SERVICE_MISMATCH"
	ReasonRouteCoherenceProfileDuplicate          Reason = "ROUTE_COHERENCE_PROFILE_DUPLICATE"
	ReasonRouteCoherenceProfileMalformed          Reason = "ROUTE_COHERENCE_PROFILE_MALFORMED"
	ReasonRouteCoherenceProfileUnknown            Reason = "ROUTE_COHERENCE_PROFILE_UNKNOWN"
	ReasonRouteCoherenceProfileIneligible         Reason = "ROUTE_COHERENCE_PROFILE_INELIGIBLE"
	ReasonRouteCoherenceProposalCount             Reason = "ROUTE_COHERENCE_PROPOSAL_COUNT_INVALID"
	ReasonRouteCoherenceProposalMismatch          Reason = "ROUTE_COHERENCE_PROPOSAL_PROFILE_MISMATCH"
	ReasonRouteCoherenceRequiredCapabilityMissing Reason = "ROUTE_COHERENCE_REQUIRED_CAPABILITY_MISSING"
	ReasonRouteCoherencePermissionEscalation      Reason = "ROUTE_COHERENCE_PERMISSION_ESCALATION"
	ReasonRouteCoherenceEscalationNotDistinct     Reason = "ROUTE_COHERENCE_ESCALATION_NOT_DISTINCT"
	ReasonRouteCoherenceValidatorNotIndependent   Reason = "ROUTE_COHERENCE_VALIDATOR_NOT_INDEPENDENT"
	ReasonRouteCoherenceForbiddenFallback         Reason = "ROUTE_COHERENCE_FORBIDDEN_FALLBACK"
	ReasonRouteCoherenceNoEligibleRoute           Reason = "ROUTE_COHERENCE_NO_ELIGIBLE_ROUTE"
)

// RouteCoherenceRequest binds task facts, the policy digest, host-supplied
// profile snapshots, permission envelope, and proposed route roles. Profiles
// and proposals are untrusted inputs; this evaluator only checks consistency.
type RouteCoherenceRequest struct {
	PlanUnitID                  string                `json:"plan_unit_id"`
	Risk                        model.RouteRiskLevel  `json:"risk"`
	RequiredCapabilityClasses   []string              `json:"required_capability_classes"`
	ApprovedPermissionClasses   []string              `json:"approved_permission_classes"`
	Policy                      model.RoutePolicy     `json:"policy"`
	PolicyDigest                string                `json:"policy_digest"`
	IndependenceRuleID          string                `json:"independence_rule_id"`
	ApprovedAccessServiceDigest string                `json:"approved_access_service_digest"`
	Profiles                    []model.RouteProfile  `json:"profiles"`
	PrimaryRoutes               []model.RouteProposal `json:"primary_routes"`
	EscalationRoutes            []model.RouteProposal `json:"escalation_routes"`
	ValidatorRoutes             []model.RouteProposal `json:"validator_routes"`
}

// RouteCoherenceResult reports only whether the proposal is coherent enough
// to present for human approval. It never selects, dispatches, or approves a
// route.
type RouteCoherenceResult struct {
	Decision                   Decision `json:"decision"`
	Reason                     Reason   `json:"reason,omitempty"`
	PolicyDigest               string   `json:"policy_digest,omitempty"`
	RequiresFreshHumanApproval bool     `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool     `json:"machine_advanced"`
}

func routeClassSet(values []string, capability bool) (map[string]struct{}, Reason) {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return nil, ReasonRouteCoherenceProfileMalformed
		}
		valid := model.ValidRoutePolicyCapability(value)
		if !capability {
			valid = model.ValidRoutePolicyPermission(value)
		}
		if !valid {
			if capability {
				return nil, ReasonRouteCoherenceCapabilityUnknown
			}
			return nil, ReasonRouteCoherencePermissionUnknown
		}
		if _, exists := set[value]; exists {
			return nil, ReasonRouteCoherenceProfileMalformed
		}
		set[value] = struct{}{}
	}
	return set, ReasonNone
}

func routeSetEqual(left, right []string, capability bool) (bool, Reason) {
	leftSet, reason := routeClassSet(left, capability)
	if reason != ReasonNone {
		return false, reason
	}
	rightSet, reason := routeClassSet(right, capability)
	if reason != ReasonNone {
		return false, reason
	}
	if len(leftSet) != len(rightSet) {
		return false, ReasonRouteCoherenceProposalMismatch
	}
	for value := range leftSet {
		if _, exists := rightSet[value]; !exists {
			return false, ReasonRouteCoherenceProposalMismatch
		}
	}
	return true, ReasonNone
}

func routeSubset(sub, super []string, capability bool) (bool, Reason) {
	subSet, reason := routeClassSet(sub, capability)
	if reason != ReasonNone {
		return false, reason
	}
	superSet, reason := routeClassSet(super, capability)
	if reason != ReasonNone {
		return false, reason
	}
	for value := range subSet {
		if _, exists := superSet[value]; !exists {
			if capability {
				return false, ReasonRouteCoherenceRequiredCapabilityMissing
			}
			return false, ReasonRouteCoherencePermissionEscalation
		}
	}
	return true, ReasonNone
}

func routeHas(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func routeRiskRank(risk model.RouteRiskLevel) int {
	switch risk {
	case model.RouteRiskLow:
		return 1
	case model.RouteRiskMedium:
		return 2
	case model.RouteRiskHigh:
		return 3
	case model.RouteRiskCritical:
		return 4
	default:
		return 0
	}
}

func findProfile(profiles map[string]model.RouteProfile, proposal model.RouteProposal) (model.RouteProfile, Reason) {
	if proposal.ProfileDigest == "" {
		return model.RouteProfile{}, ReasonRouteCoherenceProposalMismatch
	}
	profile, ok := profiles[proposal.ProfileDigest]
	if !ok {
		return model.RouteProfile{}, ReasonRouteCoherenceProfileUnknown
	}
	if !profile.Eligible {
		return model.RouteProfile{}, ReasonRouteCoherenceProfileIneligible
	}
	if proposal.ProviderFamily != profile.ProviderFamily || proposal.ModelFamily != profile.ModelFamily || proposal.ExecutionContext != profile.ExecutionContext {
		return model.RouteProfile{}, ReasonRouteCoherenceProposalMismatch
	}
	if equal, reason := routeSetEqual(proposal.CapabilityClasses, profile.CapabilityClasses, true); !equal {
		return model.RouteProfile{}, reason
	}
	if equal, reason := routeSetEqual(proposal.PermissionClasses, profile.PermissionClasses, false); !equal {
		return model.RouteProfile{}, reason
	}
	return profile, ReasonNone
}

func validateProfile(profile model.RouteProfile) Reason {
	if profile.ProfileDigest == "" || profile.ProviderFamily == "" || profile.ModelFamily == "" || profile.AccessServiceDigest == "" || !model.ValidExecutionContext(profile.ExecutionContext) {
		return ReasonRouteCoherenceProfileMalformed
	}
	if _, reason := routeClassSet(profile.CapabilityClasses, true); reason != ReasonNone {
		return reason
	}
	if _, reason := routeClassSet(profile.PermissionClasses, false); reason != ReasonNone {
		return reason
	}
	return ReasonNone
}

func validateProposal(proposal model.RouteProposal, profileMap map[string]model.RouteProfile, requiredCapabilities, approvedPermissions []string, approvedAccessService string, forbiddenFallbacks []model.RouteFallbackRule) (model.RouteProfile, Reason) {
	profile, reason := findProfile(profileMap, proposal)
	if reason != ReasonNone {
		return model.RouteProfile{}, reason
	}
	if profile.AccessServiceDigest != approvedAccessService {
		return model.RouteProfile{}, ReasonRouteCoherenceAccessServiceMismatch
	}
	for _, fallback := range forbiddenFallbacks {
		if routeHas(requiredCapabilities, fallback.FromCapabilityClass) &&
			!routeHas(proposal.CapabilityClasses, fallback.FromCapabilityClass) &&
			routeHas(proposal.CapabilityClasses, fallback.ToCapabilityClass) {
			return model.RouteProfile{}, ReasonRouteCoherenceForbiddenFallback
		}
	}
	if ok, reason := routeSubset(requiredCapabilities, proposal.CapabilityClasses, true); !ok {
		return model.RouteProfile{}, reason
	}
	if ok, reason := routeSubset(proposal.PermissionClasses, approvedPermissions, false); !ok {
		return model.RouteProfile{}, reason
	}
	return profile, ReasonNone
}

func hasEligibleRouteProfile(profiles []model.RouteProfile, requiredCapabilities, approvedPermissions []string, approvedAccessService string) bool {
	for _, profile := range profiles {
		if !profile.Eligible || profile.AccessServiceDigest != approvedAccessService {
			continue
		}
		if ok, _ := routeSubset(requiredCapabilities, profile.CapabilityClasses, true); !ok {
			continue
		}
		if ok, _ := routeSubset(profile.PermissionClasses, approvedPermissions, false); !ok {
			continue
		}
		return true
	}
	return false
}

// EvaluateRouteCoherence checks route-role cardinality, profile/proposal
// consistency, capability and permission bounds, validator independence,
// forbidden fallback rules, and approved access-service binding. It performs
// no route selection, I/O, discovery, dispatch, approval, or stage advancement.
func EvaluateRouteCoherence(req RouteCoherenceRequest) RouteCoherenceResult {
	result := RouteCoherenceResult{RequiresFreshHumanApproval: true, MachineAdvanced: false}
	if req.PlanUnitID == "" {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherencePlanUnitMissing
		return result
	}
	if !model.ValidRouteRiskLevel(req.Risk) {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceRiskInvalid
		return result
	}
	if req.PolicyDigest == "" {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherencePolicyDigestMissing
		return result
	}
	policyResult := EvaluateRoutePolicy(RoutePolicyRequest{Policy: req.Policy, ExpectedDigest: req.PolicyDigest})
	if policyResult.Decision != DecisionLegalPendingApproval {
		result.Decision, result.Reason, result.PolicyDigest = DecisionRejected, policyResult.Reason, policyResult.Digest
		return result
	}
	result.PolicyDigest = policyResult.Digest
	if _, reason := routeClassSet(req.RequiredCapabilityClasses, true); reason != ReasonNone {
		result.Decision, result.Reason = DecisionRejected, reason
		return result
	}
	if len(req.ApprovedPermissionClasses) == 0 {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherencePermissionUnknown
		return result
	}
	if _, reason := routeClassSet(req.ApprovedPermissionClasses, false); reason != ReasonNone {
		result.Decision, result.Reason = DecisionRejected, reason
		return result
	}
	if req.ApprovedAccessServiceDigest == "" {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceAccessServiceMissing
		return result
	}
	var independenceRule *model.RouteIndependenceRule
	for i := range policyResult.Projection.IndependenceRules {
		if policyResult.Projection.IndependenceRules[i].ID == req.IndependenceRuleID {
			independenceRule = &policyResult.Projection.IndependenceRules[i]
			break
		}
	}
	if independenceRule == nil || routeRiskRank(req.Risk) < routeRiskRank(independenceRule.MinimumRisk) {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherencePolicyRuleMissing
		return result
	}
	profileMap := make(map[string]model.RouteProfile, len(req.Profiles))
	for _, profile := range req.Profiles {
		if reason := validateProfile(profile); reason != ReasonNone {
			result.Decision, result.Reason = DecisionRejected, reason
			return result
		}
		if _, exists := profileMap[profile.ProfileDigest]; exists {
			result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceProfileDuplicate
			return result
		}
		profileMap[profile.ProfileDigest] = profile
	}
	if len(req.PrimaryRoutes) != 1 || len(req.EscalationRoutes) != 1 || len(req.ValidatorRoutes) != 1 {
		if !hasEligibleRouteProfile(req.Profiles, req.RequiredCapabilityClasses, req.ApprovedPermissionClasses, req.ApprovedAccessServiceDigest) {
			result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceNoEligibleRoute
			return result
		}
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceProposalCount
		return result
	}
	primary, reason := validateProposal(req.PrimaryRoutes[0], profileMap, req.RequiredCapabilityClasses, req.ApprovedPermissionClasses, req.ApprovedAccessServiceDigest, policyResult.Projection.ForbiddenFallbacks)
	if reason != ReasonNone {
		result.Decision, result.Reason = DecisionRejected, reason
		return result
	}
	escalation, reason := validateProposal(req.EscalationRoutes[0], profileMap, req.RequiredCapabilityClasses, req.ApprovedPermissionClasses, req.ApprovedAccessServiceDigest, policyResult.Projection.ForbiddenFallbacks)
	if reason != ReasonNone {
		result.Decision, result.Reason = DecisionRejected, reason
		return result
	}
	if primary.ProfileDigest == escalation.ProfileDigest {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceEscalationNotDistinct
		return result
	}
	validator, reason := validateProposal(req.ValidatorRoutes[0], profileMap, independenceRule.RequiredValidatorCapabilities, req.ApprovedPermissionClasses, req.ApprovedAccessServiceDigest, policyResult.Projection.ForbiddenFallbacks)
	if reason != ReasonNone {
		result.Decision, result.Reason = DecisionRejected, reason
		return result
	}
	if validator.ProfileDigest == primary.ProfileDigest ||
		(independenceRule.DistinctProviderFamily && validator.ProviderFamily == primary.ProviderFamily) ||
		(independenceRule.DistinctModelFamily && validator.ModelFamily == primary.ModelFamily) {
		result.Decision, result.Reason = DecisionRejected, ReasonRouteCoherenceValidatorNotIndependent
		return result
	}
	result.Decision = DecisionLegalPendingApproval
	return result
}
