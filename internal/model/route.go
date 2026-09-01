// internal/model/route.go
package model

import (
	"errors"
	"fmt"
)

// ExecutionContext is the closed vocabulary for how a route attestation's
// model call was made. A proxied route has a different trust boundary than
// a direct one, so this is not a free string.
type ExecutionContext string

const (
	ExecutionContextDirectAPI    ExecutionContext = "direct_api"
	ExecutionContextProxy        ExecutionContext = "proxy"
	ExecutionContextLocalRuntime ExecutionContext = "local_runtime"
	ExecutionContextCIRunner     ExecutionContext = "ci_runner"
)

// ValidExecutionContext reports whether c is an approved execution context.
func ValidExecutionContext(c ExecutionContext) bool {
	switch c {
	case ExecutionContextDirectAPI, ExecutionContextProxy, ExecutionContextLocalRuntime, ExecutionContextCIRunner:
		return true
	}
	return false
}

// RouteAttestation binds an evidence capture to the route that produced it:
// which provider and model family, through which execution context, under
// which host-local profile, asserting which capability and permission
// classes, and bound to which Route Selection approval. It contains no
// credential, private path, profile name, or provider-specific secret; see
// the portable capability taxonomy (architecture.md §12.1) and host-local
// profile boundary (§12.2).
type RouteAttestation struct {
	ProviderFamily      string           `json:"provider_family"`
	ModelFamily         string           `json:"model_family"`
	ExecutionContext    ExecutionContext `json:"execution_context"`
	ProfileDigest       string           `json:"profile_digest"`
	CapabilityClasses   []string         `json:"capability_classes"`
	PermissionClasses   []string         `json:"permission_classes"`
	RouteApprovalDigest string           `json:"route_approval_digest"`
}

// validateRouteAttestation structurally validates a RouteAttestation: every
// field non-empty, execution_context drawn from the closed vocabulary, and
// both class sets non-empty with no empty-string member. It never advances
// a transition. Call sites: validateEvidence, when EvidenceRef.Route is
// non-nil.
func validateRouteAttestation(r *RouteAttestation) error {
	if r == nil {
		return nil
	}
	if r.ProviderFamily == "" {
		return errors.New("route attestation missing provider_family")
	}
	if r.ModelFamily == "" {
		return errors.New("route attestation missing model_family")
	}
	if r.ExecutionContext == "" {
		return errors.New("route attestation missing execution_context")
	}
	if !ValidExecutionContext(r.ExecutionContext) {
		return fmt.Errorf("route attestation execution_context %q is not an approved execution context", r.ExecutionContext)
	}
	if r.ProfileDigest == "" {
		return errors.New("route attestation missing profile_digest")
	}
	if len(r.CapabilityClasses) == 0 {
		return errors.New("route attestation missing capability_classes")
	}
	for i, c := range r.CapabilityClasses {
		if c == "" {
			return fmt.Errorf("route attestation capability_classes[%d] is empty", i)
		}
	}
	if len(r.PermissionClasses) == 0 {
		return errors.New("route attestation missing permission_classes")
	}
	for i, c := range r.PermissionClasses {
		if c == "" {
			return fmt.Errorf("route attestation permission_classes[%d] is empty", i)
		}
	}
	if r.RouteApprovalDigest == "" {
		return errors.New("route attestation missing route_approval_digest")
	}
	return nil
}
