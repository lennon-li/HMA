package model

import (
	"reflect"
	"testing"
)

func testRoutePolicy() RoutePolicy {
	return RoutePolicy{
		Version:           RoutePolicyVersionV1,
		CapabilityClasses: []string{CapabilitySemanticReview, CapabilityBoundedImplementation, CapabilityRepositoryScanning},
		PermissionClasses: []string{PermissionBoundedWrite, PermissionReadOnly},
		RiskRules: []RouteRiskRule{{
			ID:                        "risk-high",
			MinimumRisk:               RouteRiskHigh,
			RequiredCapabilityClasses: []string{CapabilitySemanticReview},
		}},
		PermissionRules: []RoutePermissionRule{{
			ID:                         "permissions-default",
			RequiredPermissionClasses:  []string{PermissionReadOnly},
			ForbiddenPermissionClasses: []string{PermissionBoundedWrite},
		}},
		IndependenceRules: []RouteIndependenceRule{{
			ID:                            "independent-high",
			MinimumRisk:                   RouteRiskHigh,
			RequiredValidatorCapabilities: []string{CapabilitySemanticReview},
			DistinctProviderFamily:        true,
			DistinctModelFamily:           true,
		}},
		PreferenceRules: []RoutePreferenceRule{{
			ID:                         "prefer-review",
			Risk:                       RouteRiskMedium,
			RequiredCapabilityClasses:  []string{CapabilityRepositoryScanning},
			PreferredCapabilityClasses: []string{CapabilitySemanticReview},
		}},
		ForbiddenFallbacks: []RouteFallbackRule{{
			ID:                  "no-review-substitution",
			FromCapabilityClass: CapabilitySemanticReview,
			ToCapabilityClass:   CapabilityRepositoryScanning,
		}},
	}
}

func TestNormalizeRoutePolicySortsWithoutMutatingInput(t *testing.T) {
	input := testRoutePolicy()
	original := input

	got, err := NormalizeRoutePolicy(input)
	if err != nil {
		t.Fatalf("NormalizeRoutePolicy() error = %v", err)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatal("NormalizeRoutePolicy mutated its input")
	}
	if want := []string{CapabilityBoundedImplementation, CapabilityRepositoryScanning, CapabilitySemanticReview}; !reflect.DeepEqual(got.CapabilityClasses, want) {
		t.Fatalf("capabilities = %#v, want %#v", got.CapabilityClasses, want)
	}
	if want := []string{PermissionBoundedWrite, PermissionReadOnly}; !reflect.DeepEqual(got.PermissionClasses, want) {
		t.Fatalf("permissions = %#v, want %#v", got.PermissionClasses, want)
	}
	if got.RiskRules[0].ID != "risk-high" || got.PermissionRules[0].ID != "permissions-default" {
		t.Fatalf("normalized rule order was not stable: %#v %#v", got.RiskRules, got.PermissionRules)
	}
}

func TestNormalizeRoutePolicyEmptyNoRoutePolicyIsValid(t *testing.T) {
	got, err := NormalizeRoutePolicy(RoutePolicy{Version: RoutePolicyVersionV1})
	if err != nil {
		t.Fatalf("NormalizeRoutePolicy() error = %v", err)
	}
	if got.CapabilityClasses == nil || got.PermissionClasses == nil || got.RiskRules == nil {
		t.Fatal("normalized empty policy did not produce canonical empty arrays")
	}
}

func TestNormalizeRoutePolicyRejectsContradictoryPermissionBounds(t *testing.T) {
	policy := testRoutePolicy()
	policy.PermissionRules[0].ForbiddenPermissionClasses = []string{PermissionReadOnly}
	if _, err := NormalizeRoutePolicy(policy); err == nil {
		t.Fatal("NormalizeRoutePolicy accepted overlapping permission bounds")
	} else {
		validationErr, ok := err.(*RoutePolicyValidationError)
		if !ok || validationErr.Kind != RoutePolicyContradictoryBounds {
			t.Fatalf("error = %v, want contradictory bounds", err)
		}
	}
}
