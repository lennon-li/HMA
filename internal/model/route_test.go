package model

import "testing"

func validAttestation() RouteAttestation {
	return RouteAttestation{
		ProviderFamily: "openai", ModelFamily: "gpt", ExecutionContext: ExecutionContextLocalRuntime,
		ProfileDigest: "profile", CapabilityClasses: []string{CapabilitySemanticReview},
		PermissionClasses: []string{PermissionBoundedWrite}, RouteApprovalDigest: "route-approval",
	}
}

func TestValidExecutionContextClosedVocabulary(t *testing.T) {
	for _, c := range []ExecutionContext{ExecutionContextDirectAPI, ExecutionContextProxy, ExecutionContextLocalRuntime, ExecutionContextCIRunner} {
		if !ValidExecutionContext(c) {
			t.Errorf("%q rejected", c)
		}
	}
	for _, c := range []ExecutionContext{"", "shell", "DIRECT_API"} {
		if ValidExecutionContext(c) {
			t.Errorf("%q accepted", c)
		}
	}
}

func TestRouteAttestationStructure(t *testing.T) {
	ok := validAttestation()
	if err := validateRouteAttestation(&ok); err != nil {
		t.Fatalf("valid attestation rejected: %v", err)
	}
	// The attestation is optional on evidence (EvidenceRef.Route omitempty);
	// absent is not malformed.
	if err := validateRouteAttestation(nil); err != nil {
		t.Errorf("absent attestation rejected: %v", err)
	}
	cases := map[string]func(*RouteAttestation){
		"provider family":   func(r *RouteAttestation) { r.ProviderFamily = "" },
		"model family":      func(r *RouteAttestation) { r.ModelFamily = "" },
		"execution context": func(r *RouteAttestation) { r.ExecutionContext = "" },
		"unknown context":   func(r *RouteAttestation) { r.ExecutionContext = "shell" },
		"profile digest":    func(r *RouteAttestation) { r.ProfileDigest = "" },
		"no capabilities":   func(r *RouteAttestation) { r.CapabilityClasses = nil },
		"empty capability":  func(r *RouteAttestation) { r.CapabilityClasses = []string{CapabilitySemanticReview, ""} },
		"no permissions":    func(r *RouteAttestation) { r.PermissionClasses = nil },
		"empty permission":  func(r *RouteAttestation) { r.PermissionClasses = []string{""} },
		"route approval":    func(r *RouteAttestation) { r.RouteApprovalDigest = "" },
	}
	for name, mutate := range cases {
		r := validAttestation()
		mutate(&r)
		if err := validateRouteAttestation(&r); err == nil {
			t.Errorf("%s: invalid attestation accepted", name)
		}
	}
}
