package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/lennon-li/HMA/internal/model"
)

const (
	ReasonRoutePolicyInvalidVersion      Reason = "ROUTE_POLICY_VERSION_UNSUPPORTED"
	ReasonRoutePolicyMissingField        Reason = "ROUTE_POLICY_FIELD_MISSING"
	ReasonRoutePolicyUnknownCapability   Reason = "ROUTE_POLICY_CAPABILITY_UNKNOWN"
	ReasonRoutePolicyUnknownPermission   Reason = "ROUTE_POLICY_PERMISSION_UNKNOWN"
	ReasonRoutePolicyDuplicateIdentifier Reason = "ROUTE_POLICY_IDENTIFIER_DUPLICATE"
	ReasonRoutePolicyInvalidRisk         Reason = "ROUTE_POLICY_RISK_INVALID"
	ReasonRoutePolicyMalformedReference  Reason = "ROUTE_POLICY_REFERENCE_MALFORMED"
	ReasonRoutePolicyContradictoryBounds Reason = "ROUTE_POLICY_BOUNDS_CONTRADICTORY"
	ReasonRoutePolicyDigestMismatch      Reason = "ROUTE_POLICY_DIGEST_MISMATCH"
	ReasonRoutePolicyCanonicalization    Reason = "ROUTE_POLICY_CANONICALIZATION_FAILED"
)

// RoutePolicyRequest asks only for deterministic validation, normalization,
// and digesting. ExpectedDigest can be used by a caller to detect policy drift.
type RoutePolicyRequest struct {
	Policy         model.RoutePolicy `json:"policy"`
	ExpectedDigest string            `json:"expected_digest,omitempty"`
}

// RoutePolicyResult contains a policy projection and digest, never a route
// choice or an approval. A valid result remains pending human approval.
type RoutePolicyResult struct {
	Decision                   Decision                    `json:"decision"`
	Reason                     Reason                      `json:"reason,omitempty"`
	Projection                 model.RoutePolicyProjection `json:"projection,omitempty"`
	Digest                     string                      `json:"digest,omitempty"`
	RequiresFreshHumanApproval bool                        `json:"requires_fresh_human_approval"`
	MachineAdvanced            bool                        `json:"machine_advanced"`
}

func routePolicyReason(err error) Reason {
	var validationErr *model.RoutePolicyValidationError
	if !errors.As(err, &validationErr) {
		return ReasonRoutePolicyCanonicalization
	}
	switch validationErr.Kind {
	case model.RoutePolicyInvalidVersion:
		return ReasonRoutePolicyInvalidVersion
	case model.RoutePolicyMissingField:
		return ReasonRoutePolicyMissingField
	case model.RoutePolicyUnknownCapability:
		return ReasonRoutePolicyUnknownCapability
	case model.RoutePolicyUnknownPermission:
		return ReasonRoutePolicyUnknownPermission
	case model.RoutePolicyDuplicateIdentifier:
		return ReasonRoutePolicyDuplicateIdentifier
	case model.RoutePolicyInvalidRisk:
		return ReasonRoutePolicyInvalidRisk
	case model.RoutePolicyMalformedReference:
		return ReasonRoutePolicyMalformedReference
	case model.RoutePolicyContradictoryBounds:
		return ReasonRoutePolicyContradictoryBounds
	default:
		return ReasonRoutePolicyCanonicalization
	}
}

// RoutePolicyDigest returns the content digest of a normalized projection.
// The projection uses only structs and sorted slices, so encoding/json's
// deterministic struct-field order is the canonical representation.
func RoutePolicyDigest(projection model.RoutePolicyProjection) (string, error) {
	canonical, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// EvaluateRoutePolicy validates and normalizes route policy data. It never
// selects a route, executes discovery, changes state, widens permissions,
// grants approval, or advances a stage.
func EvaluateRoutePolicy(req RoutePolicyRequest) RoutePolicyResult {
	projection, err := model.NormalizeRoutePolicy(req.Policy)
	if err != nil {
		return RoutePolicyResult{Decision: DecisionRejected, Reason: routePolicyReason(err), RequiresFreshHumanApproval: true, MachineAdvanced: false}
	}
	digest, err := RoutePolicyDigest(projection)
	if err != nil {
		return RoutePolicyResult{Decision: DecisionRejected, Reason: ReasonRoutePolicyCanonicalization, RequiresFreshHumanApproval: true, MachineAdvanced: false}
	}
	if req.ExpectedDigest != "" && req.ExpectedDigest != digest {
		return RoutePolicyResult{Decision: DecisionRejected, Reason: ReasonRoutePolicyDigestMismatch, Projection: projection, Digest: digest, RequiresFreshHumanApproval: true, MachineAdvanced: false}
	}
	return RoutePolicyResult{Decision: DecisionLegalPendingApproval, Projection: projection, Digest: digest, RequiresFreshHumanApproval: true, MachineAdvanced: false}
}
