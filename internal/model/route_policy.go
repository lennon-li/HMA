package model

import (
	"fmt"
	"sort"
)

// RoutePolicyVersion identifies the normalized route-policy contract.
type RoutePolicyVersion string

const RoutePolicyVersionV1 RoutePolicyVersion = "route-policy/v1"

// Portable capability classes are intentionally vendor- and host-neutral.
const (
	CapabilityPlanning                 = "planning"
	CapabilityArchitecture             = "architecture"
	CapabilityBoundedImplementation    = "bounded_implementation"
	CapabilityDebugging                = "debugging"
	CapabilityRepositoryScanning       = "repository_scanning"
	CapabilitySemanticReview           = "semantic_review"
	CapabilityBrowserRuntimeValidation = "browser_runtime_validation"
	CapabilityRiskTolerance            = "risk_tolerance"
	CapabilityIsolationPermission      = "isolation_permission_enforcement"
	CapabilityTimeoutCancellation      = "timeout_cancellation"
	CapabilityEvidenceOutputCapture    = "evidence_output_capture"
	CapabilityContextOutputBounds      = "context_output_bounds"
	CapabilityCostQuota                = "cost_quota"
)

var portableCapabilityClasses = map[string]struct{}{
	CapabilityPlanning:                 {},
	CapabilityArchitecture:             {},
	CapabilityBoundedImplementation:    {},
	CapabilityDebugging:                {},
	CapabilityRepositoryScanning:       {},
	CapabilitySemanticReview:           {},
	CapabilityBrowserRuntimeValidation: {},
	CapabilityRiskTolerance:            {},
	CapabilityIsolationPermission:      {},
	CapabilityTimeoutCancellation:      {},
	CapabilityEvidenceOutputCapture:    {},
	CapabilityContextOutputBounds:      {},
	CapabilityCostQuota:                {},
}

// Portable permission classes describe the interaction envelope, not a host
// or provider-specific access mechanism.
const (
	PermissionReadOnly         = "read_only"
	PermissionBoundedWrite     = "bounded_write"
	PermissionNoExternalEffect = "no_external_effect"
)

var portablePermissionClasses = map[string]struct{}{
	PermissionReadOnly:         {},
	PermissionBoundedWrite:     {},
	PermissionNoExternalEffect: {},
}

// RouteRiskLevel is the closed risk vocabulary used by route policy rules.
type RouteRiskLevel string

const (
	RouteRiskLow      RouteRiskLevel = "low"
	RouteRiskMedium   RouteRiskLevel = "medium"
	RouteRiskHigh     RouteRiskLevel = "high"
	RouteRiskCritical RouteRiskLevel = "critical"
)

// RouteRiskRule binds a minimum risk classification to required capabilities.
type RouteRiskRule struct {
	ID                        string         `json:"id"`
	MinimumRisk               RouteRiskLevel `json:"minimum_risk"`
	RequiredCapabilityClasses []string       `json:"required_capability_classes"`
}

// RoutePermissionRule describes permission bounds for a policy condition.
// Required and forbidden classes must never overlap.
type RoutePermissionRule struct {
	ID                         string   `json:"id"`
	RequiredPermissionClasses  []string `json:"required_permission_classes"`
	ForbiddenPermissionClasses []string `json:"forbidden_permission_classes"`
}

// RouteIndependenceRule declares the independent-validator requirements at a
// risk level. It contains no provider or model names.
type RouteIndependenceRule struct {
	ID                            string         `json:"id"`
	MinimumRisk                   RouteRiskLevel `json:"minimum_risk"`
	RequiredValidatorCapabilities []string       `json:"required_validator_capabilities"`
	DistinctProviderFamily        bool           `json:"distinct_provider_family"`
	DistinctModelFamily           bool           `json:"distinct_model_family"`
}

// RoutePreferenceRule records portable preference inputs for a future route
// selector. It does not itself select a route.
type RoutePreferenceRule struct {
	ID                         string         `json:"id"`
	Risk                       RouteRiskLevel `json:"risk"`
	RequiredCapabilityClasses  []string       `json:"required_capability_classes"`
	PreferredCapabilityClasses []string       `json:"preferred_capability_classes"`
}

// RouteFallbackRule lists a forbidden capability substitution. Both ends are
// portable capability classes, never client or provider identifiers.
type RouteFallbackRule struct {
	ID                  string `json:"id"`
	FromCapabilityClass string `json:"from_capability_class"`
	ToCapabilityClass   string `json:"to_capability_class"`
}

// RoutePolicy is the portable, versioned input to route-policy evaluation.
// It contains no host-local profiles, availability claims, credentials,
// private paths, provider choices, or execution results.
type RoutePolicy struct {
	Version            RoutePolicyVersion      `json:"version"`
	CapabilityClasses  []string                `json:"capability_classes"`
	PermissionClasses  []string                `json:"permission_classes"`
	RiskRules          []RouteRiskRule         `json:"risk_rules"`
	PermissionRules    []RoutePermissionRule   `json:"permission_rules"`
	IndependenceRules  []RouteIndependenceRule `json:"independence_rules"`
	PreferenceRules    []RoutePreferenceRule   `json:"preference_rules"`
	ForbiddenFallbacks []RouteFallbackRule     `json:"forbidden_fallbacks"`
}

// RoutePolicyProjection is the canonical normalized representation whose
// JSON bytes are hashed by the engine.
type RoutePolicyProjection = RoutePolicy

// RoutePolicyValidationKind identifies a deterministic policy validation
// failure for the engine's closed reason vocabulary.
type RoutePolicyValidationKind string

const (
	RoutePolicyInvalidVersion      RoutePolicyValidationKind = "invalid_version"
	RoutePolicyMissingField        RoutePolicyValidationKind = "missing_field"
	RoutePolicyUnknownCapability   RoutePolicyValidationKind = "unknown_capability"
	RoutePolicyUnknownPermission   RoutePolicyValidationKind = "unknown_permission"
	RoutePolicyDuplicateIdentifier RoutePolicyValidationKind = "duplicate_identifier"
	RoutePolicyInvalidRisk         RoutePolicyValidationKind = "invalid_risk"
	RoutePolicyMalformedReference  RoutePolicyValidationKind = "malformed_reference"
	RoutePolicyContradictoryBounds RoutePolicyValidationKind = "contradictory_bounds"
)

// RoutePolicyValidationError carries stable classification without exposing
// arbitrary validation prose as a machine reason.
type RoutePolicyValidationError struct {
	Kind  RoutePolicyValidationKind
	Field string
}

func (e *RoutePolicyValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("route policy validation failed: %s", e.Kind)
	}
	return fmt.Sprintf("route policy validation failed: %s (%s)", e.Kind, e.Field)
}

func policyError(kind RoutePolicyValidationKind, field string) error {
	return &RoutePolicyValidationError{Kind: kind, Field: field}
}

// ValidRoutePolicyCapability reports whether c is in the portable taxonomy.
func ValidRoutePolicyCapability(c string) bool {
	_, ok := portableCapabilityClasses[c]
	return ok
}

// ValidRoutePolicyPermission reports whether p is in the portable permission
// vocabulary.
func ValidRoutePolicyPermission(p string) bool {
	_, ok := portablePermissionClasses[p]
	return ok
}

// ValidRouteRiskLevel reports whether risk is in the closed risk vocabulary.
func ValidRouteRiskLevel(risk RouteRiskLevel) bool {
	switch risk {
	case RouteRiskLow, RouteRiskMedium, RouteRiskHigh, RouteRiskCritical:
		return true
	default:
		return false
	}
}

func sortedUnique(values []string, kind RoutePolicyValidationKind, field string) ([]string, error) {
	result := append([]string{}, values...)
	for i, value := range result {
		if value == "" {
			return nil, policyError(RoutePolicyMissingField, fmt.Sprintf("%s[%d]", field, i))
		}
	}
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i] == result[i-1] {
			return nil, policyError(kind, field)
		}
	}
	return result, nil
}

func normalizeCapabilities(values []string, field string) ([]string, error) {
	result, err := sortedUnique(values, RoutePolicyDuplicateIdentifier, field)
	if err != nil {
		return nil, err
	}
	for _, value := range result {
		if !ValidRoutePolicyCapability(value) {
			return nil, policyError(RoutePolicyUnknownCapability, field)
		}
	}
	return result, nil
}

func normalizePermissions(values []string, field string) ([]string, error) {
	result, err := sortedUnique(values, RoutePolicyDuplicateIdentifier, field)
	if err != nil {
		return nil, err
	}
	for _, value := range result {
		if !ValidRoutePolicyPermission(value) {
			return nil, policyError(RoutePolicyUnknownPermission, field)
		}
	}
	return result, nil
}

func validateReferences(values, declared []string, field string) error {
	set := make(map[string]struct{}, len(declared))
	for _, value := range declared {
		set[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := set[value]; !ok {
			return policyError(RoutePolicyMalformedReference, field)
		}
	}
	return nil
}

func registerRuleID(ids map[string]struct{}, id, field string) error {
	if id == "" {
		return policyError(RoutePolicyMissingField, field+".id")
	}
	if _, exists := ids[id]; exists {
		return policyError(RoutePolicyDuplicateIdentifier, field+".id")
	}
	ids[id] = struct{}{}
	return nil
}

// NormalizeRoutePolicy validates and produces the canonical route-policy
// projection. It performs no route selection and has no side effects.
func NormalizeRoutePolicy(policy RoutePolicy) (RoutePolicyProjection, error) {
	if policy.Version == "" {
		return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, "version")
	}
	if policy.Version != RoutePolicyVersionV1 {
		return RoutePolicyProjection{}, policyError(RoutePolicyInvalidVersion, "version")
	}

	capabilities, err := normalizeCapabilities(policy.CapabilityClasses, "capability_classes")
	if err != nil {
		return RoutePolicyProjection{}, err
	}
	permissions, err := normalizePermissions(policy.PermissionClasses, "permission_classes")
	if err != nil {
		return RoutePolicyProjection{}, err
	}
	ids := make(map[string]struct{})

	riskRules := make([]RouteRiskRule, len(policy.RiskRules))
	for i, rule := range policy.RiskRules {
		field := fmt.Sprintf("risk_rules[%d]", i)
		if err := registerRuleID(ids, rule.ID, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		if !ValidRouteRiskLevel(rule.MinimumRisk) {
			return RoutePolicyProjection{}, policyError(RoutePolicyInvalidRisk, field+".minimum_risk")
		}
		classes, err := normalizeCapabilities(rule.RequiredCapabilityClasses, field+".required_capability_classes")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		if len(classes) == 0 {
			return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, field+".required_capability_classes")
		}
		if err := validateReferences(classes, capabilities, field+".required_capability_classes"); err != nil {
			return RoutePolicyProjection{}, err
		}
		riskRules[i] = RouteRiskRule{ID: rule.ID, MinimumRisk: rule.MinimumRisk, RequiredCapabilityClasses: classes}
	}
	sort.Slice(riskRules, func(i, j int) bool { return riskRules[i].ID < riskRules[j].ID })

	permissionRules := make([]RoutePermissionRule, len(policy.PermissionRules))
	for i, rule := range policy.PermissionRules {
		field := fmt.Sprintf("permission_rules[%d]", i)
		if err := registerRuleID(ids, rule.ID, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		required, err := normalizePermissions(rule.RequiredPermissionClasses, field+".required_permission_classes")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		forbidden, err := normalizePermissions(rule.ForbiddenPermissionClasses, field+".forbidden_permission_classes")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		if len(required) == 0 && len(forbidden) == 0 {
			return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, field)
		}
		if err := validateReferences(required, permissions, field+".required_permission_classes"); err != nil {
			return RoutePolicyProjection{}, err
		}
		if err := validateReferences(forbidden, permissions, field+".forbidden_permission_classes"); err != nil {
			return RoutePolicyProjection{}, err
		}
		for _, requiredPermission := range required {
			for _, forbiddenPermission := range forbidden {
				if requiredPermission == forbiddenPermission {
					return RoutePolicyProjection{}, policyError(RoutePolicyContradictoryBounds, field)
				}
			}
		}
		permissionRules[i] = RoutePermissionRule{ID: rule.ID, RequiredPermissionClasses: required, ForbiddenPermissionClasses: forbidden}
	}
	sort.Slice(permissionRules, func(i, j int) bool { return permissionRules[i].ID < permissionRules[j].ID })

	independenceRules := make([]RouteIndependenceRule, len(policy.IndependenceRules))
	for i, rule := range policy.IndependenceRules {
		field := fmt.Sprintf("independence_rules[%d]", i)
		if err := registerRuleID(ids, rule.ID, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		if !ValidRouteRiskLevel(rule.MinimumRisk) {
			return RoutePolicyProjection{}, policyError(RoutePolicyInvalidRisk, field+".minimum_risk")
		}
		classes, err := normalizeCapabilities(rule.RequiredValidatorCapabilities, field+".required_validator_capabilities")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		if len(classes) == 0 && !rule.DistinctProviderFamily && !rule.DistinctModelFamily {
			return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, field)
		}
		if err := validateReferences(classes, capabilities, field+".required_validator_capabilities"); err != nil {
			return RoutePolicyProjection{}, err
		}
		independenceRules[i] = RouteIndependenceRule{ID: rule.ID, MinimumRisk: rule.MinimumRisk, RequiredValidatorCapabilities: classes, DistinctProviderFamily: rule.DistinctProviderFamily, DistinctModelFamily: rule.DistinctModelFamily}
	}
	sort.Slice(independenceRules, func(i, j int) bool { return independenceRules[i].ID < independenceRules[j].ID })

	preferenceRules := make([]RoutePreferenceRule, len(policy.PreferenceRules))
	for i, rule := range policy.PreferenceRules {
		field := fmt.Sprintf("preference_rules[%d]", i)
		if err := registerRuleID(ids, rule.ID, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		if !ValidRouteRiskLevel(rule.Risk) {
			return RoutePolicyProjection{}, policyError(RoutePolicyInvalidRisk, field+".risk")
		}
		required, err := normalizeCapabilities(rule.RequiredCapabilityClasses, field+".required_capability_classes")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		preferred, err := normalizeCapabilities(rule.PreferredCapabilityClasses, field+".preferred_capability_classes")
		if err != nil {
			return RoutePolicyProjection{}, err
		}
		if len(preferred) == 0 {
			return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, field+".preferred_capability_classes")
		}
		if err := validateReferences(required, capabilities, field+".required_capability_classes"); err != nil {
			return RoutePolicyProjection{}, err
		}
		if err := validateReferences(preferred, capabilities, field+".preferred_capability_classes"); err != nil {
			return RoutePolicyProjection{}, err
		}
		preferenceRules[i] = RoutePreferenceRule{ID: rule.ID, Risk: rule.Risk, RequiredCapabilityClasses: required, PreferredCapabilityClasses: preferred}
	}
	sort.Slice(preferenceRules, func(i, j int) bool { return preferenceRules[i].ID < preferenceRules[j].ID })

	forbiddenFallbacks := make([]RouteFallbackRule, len(policy.ForbiddenFallbacks))
	for i, rule := range policy.ForbiddenFallbacks {
		field := fmt.Sprintf("forbidden_fallbacks[%d]", i)
		if err := registerRuleID(ids, rule.ID, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		if rule.FromCapabilityClass == "" || rule.ToCapabilityClass == "" {
			return RoutePolicyProjection{}, policyError(RoutePolicyMissingField, field)
		}
		if rule.FromCapabilityClass == rule.ToCapabilityClass {
			return RoutePolicyProjection{}, policyError(RoutePolicyContradictoryBounds, field)
		}
		if err := validateReferences([]string{rule.FromCapabilityClass, rule.ToCapabilityClass}, capabilities, field); err != nil {
			return RoutePolicyProjection{}, err
		}
		forbiddenFallbacks[i] = rule
	}
	sort.Slice(forbiddenFallbacks, func(i, j int) bool { return forbiddenFallbacks[i].ID < forbiddenFallbacks[j].ID })

	return RoutePolicyProjection{
		Version:            policy.Version,
		CapabilityClasses:  capabilities,
		PermissionClasses:  permissions,
		RiskRules:          riskRules,
		PermissionRules:    permissionRules,
		IndependenceRules:  independenceRules,
		PreferenceRules:    preferenceRules,
		ForbiddenFallbacks: forbiddenFallbacks,
	}, nil
}
