// internal/engine/route_verification.go
package engine

import (
	"github.com/lennon-li/HMA/internal/model"
)

// AllowedRouteEntry is one host-declared, out-of-repository allowed-routes
// manifest entry: the provider/model/context a given profile_digest is
// authorized to assert, and the capability/permission classes it may assert
// under that profile. The manifest is part of the trusted computing base,
// not the portable record stream (architecture.md §12.2, §22).
type AllowedRouteEntry struct {
	ProviderFamily    string
	ModelFamily       string
	ExecutionContext  model.ExecutionContext
	CapabilityClasses []string
	PermissionClasses []string
}

// RouteAttestationRequest is one route attestation to verify against the
// host-declared allowed-routes manifest and the plan unit's active Route
// Selection approval.
type RouteAttestationRequest struct {
	Attestation                 model.RouteAttestation
	AllowedRoutes               map[string]AllowedRouteEntry // keyed by profile_digest
	ApprovedRouteApprovalDigest string
	ApprovedCapabilityClasses   []string
	ApprovedPermissionClasses   []string
}

// Route attestation rejection reasons (architecture.md Task 6 design §4).
// Every rejection is unwaivable; none of these codes are WAIVABLE.
const (
	ReasonRouteAttestationMissing    Reason = "ROUTE_ATTESTATION_MISSING"
	ReasonRouteAttestationIncomplete Reason = "ROUTE_ATTESTATION_INCOMPLETE"
	ReasonRouteApprovalMismatch      Reason = "ROUTE_APPROVAL_MISMATCH"
	ReasonRouteApprovalStale         Reason = "ROUTE_APPROVAL_STALE"
	ReasonRouteProfileUnknown        Reason = "ROUTE_PROFILE_UNKNOWN"
	ReasonRouteManifestMismatch      Reason = "ROUTE_MANIFEST_MISMATCH"
	ReasonRouteCapabilityEscalation  Reason = "ROUTE_CAPABILITY_ESCALATION"
	ReasonRoutePermissionEscalation  Reason = "ROUTE_PERMISSION_ESCALATION"
	ReasonRouteUnitScopeViolation    Reason = "ROUTE_UNIT_SCOPE_VIOLATION"
)

// EvaluateRouteAttestation checks a route attestation, in order: structural
// completeness, approval binding, manifest membership, manifest
// consistency, and capability/permission containment. It is pure: it reads
// req, returns a value, and changes no state.
//
// Success (DecisionLegalPendingApproval) means the attestation is evidence
// eligible to support a transition packet, not an autonomous advance —
// machine authority here is limited to checking record shape, freshness,
// and binding; it never authorizes on its own. This is a trust-root
// consistency check against host-declared state, not cryptographic proof
// that a specific vendor executed the call; see design §3.3.
//
// ROUTE_ATTESTATION_MISSING and ROUTE_UNIT_SCOPE_VIOLATION are gate-policy
// and unit-scoping checks made by the caller before/around this evaluator,
// not by this function: RouteAttestationRequest always carries a concrete
// Attestation value (no "absent route" case), and carries no plan-unit
// identifier to scope against. ROUTE_APPROVAL_STALE is likewise not
// distinguishable from ROUTE_APPROVAL_MISMATCH from this function's inputs
// alone -- the caller supplies only the currently active approval digest,
// so any non-matching digest surfaces as ROUTE_APPROVAL_MISMATCH. All nine
// reason codes are declared here to keep the reason vocabulary closed and
// exact for callers that produce the other codes at the gate-policy layer.
func EvaluateRouteAttestation(req RouteAttestationRequest) (Decision, Reason) {
	a := req.Attestation

	if a.ProviderFamily == "" || a.ModelFamily == "" || a.ExecutionContext == "" ||
		a.ProfileDigest == "" || len(a.CapabilityClasses) == 0 ||
		len(a.PermissionClasses) == 0 || a.RouteApprovalDigest == "" {
		return DecisionRejected, ReasonRouteAttestationIncomplete
	}
	if !model.ValidExecutionContext(a.ExecutionContext) {
		return DecisionRejected, ReasonRouteAttestationIncomplete
	}

	if a.RouteApprovalDigest != req.ApprovedRouteApprovalDigest {
		return DecisionRejected, ReasonRouteApprovalMismatch
	}

	entry, ok := req.AllowedRoutes[a.ProfileDigest]
	if !ok {
		return DecisionRejected, ReasonRouteProfileUnknown
	}

	if entry.ProviderFamily != a.ProviderFamily ||
		entry.ModelFamily != a.ModelFamily ||
		entry.ExecutionContext != a.ExecutionContext {
		return DecisionRejected, ReasonRouteManifestMismatch
	}

	if !isSubset(a.CapabilityClasses, entry.CapabilityClasses) ||
		!isSubset(a.CapabilityClasses, req.ApprovedCapabilityClasses) {
		return DecisionRejected, ReasonRouteCapabilityEscalation
	}
	if !isSubset(a.PermissionClasses, entry.PermissionClasses) ||
		!isSubset(a.PermissionClasses, req.ApprovedPermissionClasses) {
		return DecisionRejected, ReasonRoutePermissionEscalation
	}

	return DecisionLegalPendingApproval, ReasonNone
}

// isSubset reports whether every element of sub appears in super.
func isSubset(sub, super []string) bool {
	set := make(map[string]bool, len(super))
	for _, s := range super {
		set[s] = true
	}
	for _, s := range sub {
		if !set[s] {
			return false
		}
	}
	return true
}
