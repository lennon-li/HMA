package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

type RecordKind string

type DecisionMode string

type PreflightStatus string

type DispatchRole string

type ReviewVerdict string

const (
	KindDecision  RecordKind = "orchestration_decision"
	KindPreflight RecordKind = "route_preflight"
	KindDispatch  RecordKind = "dispatch"
	KindReview    RecordKind = "independent_review"

	DecisionDelegate            DecisionMode = "DELEGATE"
	DecisionDirectCostException DecisionMode = "DIRECT_COST_EXCEPTION"

	PreflightAvailable      PreflightStatus = "AVAILABLE"
	PreflightUnavailable    PreflightStatus = "UNAVAILABLE"
	PreflightIndeterminate  PreflightStatus = "INDETERMINATE"
	PreflightRateLimited    PreflightStatus = "RATE_LIMITED"
	PreflightQuotaExhausted PreflightStatus = "QUOTA_EXHAUSTED"

	RoleImplementation DispatchRole = "IMPLEMENTATION"
	RoleReview         DispatchRole = "REVIEW"

	ReviewApprove ReviewVerdict = "APPROVE"
	ReviewBlock   ReviewVerdict = "BLOCK"
)

const (
	ReasonDecisionMissing            = "HOST_ORCHESTRATION_DECISION_MISSING"
	ReasonDecisionSupersession       = "HOST_DECISION_SUPERSESSION_INVALID"
	ReasonDirectCostNotCheaper       = "HOST_DIRECT_COST_EXCEPTION_NOT_JUSTIFIED"
	ReasonRouteMismatch              = "HOST_ROUTE_MISMATCH"
	ReasonPermissionWidened          = "HOST_PERMISSION_WIDENED"
	ReasonPreflightMissing           = "HOST_PREFLIGHT_MISSING"
	ReasonPreflightUnavailable       = "HOST_PREFLIGHT_UNAVAILABLE"
	ReasonPreflightStale             = "HOST_PREFLIGHT_STALE"
	ReasonImplementationMissing      = "HOST_IMPLEMENTATION_RECORD_MISSING"
	ReasonReviewMissing              = "HOST_INDEPENDENT_REVIEW_MISSING"
	ReasonReviewerNotIndependent     = "HOST_REVIEWER_NOT_INDEPENDENT"
	ReasonReviewPacketMismatch       = "HOST_REVIEW_PACKET_MISMATCH"
	ReasonInvalidHostRecord          = "HOST_RECORD_INVALID"
	ReasonCoreRunMissing             = "HOST_CORE_RUN_MISSING"
	ReasonConcurrentWrite            = "HOST_CONCURRENT_WRITE"
	ReasonHostChainCorrupt           = "HOST_CHAIN_CORRUPT"
	ReasonHostHeadMismatch           = "HOST_HEAD_MISMATCH"
	ReasonImplementationAlreadyBound = "HOST_IMPLEMENTATION_ALREADY_BOUND"
)

// Route is host orchestration telemetry. The portable route-policy identity
// remains the profile/provider/model-family/permission projection; the extra
// fields are credential-free human-facing telemetry required by architecture
// section 12.4.
type Route struct {
	WorkerID            string                 `json:"worker_id"`
	Provider            string                 `json:"provider"`
	ProviderFamily      string                 `json:"provider_family"`
	Model               string                 `json:"model"`
	ModelFamily         string                 `json:"model_family"`
	ProfileDigest       string                 `json:"profile_digest"`
	AccessService       string                 `json:"access_service"`
	AccessServiceDigest string                 `json:"access_service_digest"`
	Runtime             string                 `json:"runtime"`
	ReasoningEffort     string                 `json:"reasoning_effort"`
	ExecutionContext    model.ExecutionContext `json:"execution_context"`
	CapabilityClasses   []string               `json:"capability_classes"`
	PermissionClasses   []string               `json:"permission_classes"`
	RouteApprovalDigest string                 `json:"route_approval_digest"`
}

type OrchestrationDecision struct {
	UnitID                    string       `json:"unit_id"`
	Mode                      DecisionMode `json:"mode"`
	BoundedJob                string       `json:"bounded_job"`
	EstimatedDirectCost       float64      `json:"estimated_direct_cost"`
	EstimatedDelegatedCost    float64      `json:"estimated_delegated_cost"`
	CostUnit                  string       `json:"cost_unit"`
	ComparisonBasis           string       `json:"comparison_basis"`
	Rationale                 string       `json:"rationale"`
	ApprovedPermissionClasses []string     `json:"approved_permission_classes"`
	WorkContextDigest         string       `json:"work_context_digest"`
	Route                     Route        `json:"route"`
	SupersedesDecisionDigest  string       `json:"supersedes_decision_digest,omitempty"`
}

type RoutePreflight struct {
	UnitID         string          `json:"unit_id"`
	Role           DispatchRole    `json:"role"`
	Route          Route           `json:"route"`
	Status         PreflightStatus `json:"status"`
	ResponseDigest string          `json:"response_digest,omitempty"`
	ExpiresAt      string          `json:"expires_at"`
}

type Dispatch struct {
	UnitID                     string               `json:"unit_id"`
	Role                       DispatchRole         `json:"role"`
	Route                      Route                `json:"route"`
	PreflightRecordDigest      string               `json:"preflight_record_digest"`
	BoundedTask                string               `json:"bounded_task"`
	WorkContextDigest          string               `json:"work_context_digest"`
	ImplementationRecordDigest string               `json:"implementation_record_digest,omitempty"`
	ArtifactDigest             string               `json:"artifact_digest,omitempty"`
	ReviewPacketDigest         string               `json:"review_packet_digest,omitempty"`
	Risk                       model.RouteRiskLevel `json:"risk,omitempty"`
}

type IndependentReview struct {
	UnitID                     string        `json:"unit_id"`
	ImplementationRecordDigest string        `json:"implementation_record_digest"`
	ReviewDispatchDigest       string        `json:"review_dispatch_digest"`
	ArtifactDigest             string        `json:"artifact_digest"`
	ReviewPacketDigest         string        `json:"review_packet_digest"`
	ReviewerID                 string        `json:"reviewer_id"`
	Verdict                    ReviewVerdict `json:"verdict"`
	FindingsDigest             string        `json:"findings_digest,omitempty"`
}

type Record struct {
	Version         int                    `json:"version"`
	Kind            RecordKind             `json:"kind"`
	RunID           string                 `json:"run_id"`
	Sequence        int64                  `json:"sequence"`
	PredecessorHash string                 `json:"predecessor_hash"`
	HeadAnchor      string                 `json:"head_anchor"`
	CoreHeadAnchor  string                 `json:"core_head_anchor"`
	Timestamp       string                 `json:"timestamp"`
	Actor           string                 `json:"actor"`
	Decision        *OrchestrationDecision `json:"decision,omitempty"`
	Preflight       *RoutePreflight        `json:"preflight,omitempty"`
	Dispatch        *Dispatch              `json:"dispatch,omitempty"`
	Review          *IndependentReview     `json:"review,omitempty"`
}

func validDecisionMode(v DecisionMode) bool {
	return v == DecisionDelegate || v == DecisionDirectCostException
}

func validRole(v DispatchRole) bool { return v == RoleImplementation || v == RoleReview }

func validPreflightStatus(v PreflightStatus) bool {
	switch v {
	case PreflightAvailable, PreflightUnavailable, PreflightIndeterminate, PreflightRateLimited, PreflightQuotaExhausted:
		return true
	}
	return false
}

func validVerdict(v ReviewVerdict) bool { return v == ReviewApprove || v == ReviewBlock }

func noDuplicates(values []string) bool {
	seen := map[string]bool{}
	for _, v := range values {
		if v == "" || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func subset(values, allowed []string) bool {
	for _, v := range values {
		if !contains(allowed, v) {
			return false
		}
	}
	return true
}

func validatePermissionClasses(values []string) error {
	if len(values) == 0 || !noDuplicates(values) {
		return errors.New("permission classes must be non-empty and unique")
	}
	for _, p := range values {
		if !model.ValidRoutePolicyPermission(p) {
			return fmt.Errorf("unknown permission class %q", p)
		}
	}
	return nil
}

func validateCapabilityClasses(values []string) error {
	if len(values) == 0 || !noDuplicates(values) {
		return errors.New("capability classes must be non-empty and unique")
	}
	for _, c := range values {
		if !model.ValidRoutePolicyCapability(c) {
			return fmt.Errorf("unknown capability class %q", c)
		}
	}
	return nil
}

func validateRoute(r Route) error {
	required := []struct {
		name, value string
	}{
		{"worker_id", r.WorkerID}, {"provider", r.Provider}, {"provider_family", r.ProviderFamily},
		{"model", r.Model}, {"model_family", r.ModelFamily}, {"profile_digest", r.ProfileDigest},
		{"access_service", r.AccessService}, {"access_service_digest", r.AccessServiceDigest},
		{"runtime", r.Runtime}, {"reasoning_effort", r.ReasoningEffort}, {"route_approval_digest", r.RouteApprovalDigest},
	}
	for _, f := range required {
		if f.value == "" {
			return fmt.Errorf("route missing %s", f.name)
		}
	}
	if !model.ValidExecutionContext(r.ExecutionContext) {
		return fmt.Errorf("invalid execution_context %q", r.ExecutionContext)
	}
	if err := validateCapabilityClasses(r.CapabilityClasses); err != nil {
		return err
	}
	return validatePermissionClasses(r.PermissionClasses)
}

func canonicalRoute(r Route) Route {
	r.CapabilityClasses = append([]string(nil), r.CapabilityClasses...)
	r.PermissionClasses = append([]string(nil), r.PermissionClasses...)
	sort.Strings(r.CapabilityClasses)
	sort.Strings(r.PermissionClasses)
	return r
}

func RouteDigest(r Route) (string, error) {
	if err := validateRoute(r); err != nil {
		return "", err
	}
	b, err := json.Marshal(canonicalRoute(r))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func sameRoute(a, b Route) bool {
	da, errA := RouteDigest(a)
	db, errB := RouteDigest(b)
	return errA == nil && errB == nil && da == db
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("missing timestamp")
	}
	return time.Parse(time.RFC3339, value)
}

func validateDecision(d *OrchestrationDecision) error {
	if d == nil {
		return errors.New("missing decision")
	}
	if d.UnitID == "" || d.BoundedJob == "" || d.CostUnit == "" || d.ComparisonBasis == "" || d.Rationale == "" || d.WorkContextDigest == "" {
		return errors.New("decision missing required field")
	}
	if !validDecisionMode(d.Mode) {
		return fmt.Errorf("invalid decision mode %q", d.Mode)
	}
	if d.EstimatedDirectCost < 0 || d.EstimatedDelegatedCost < 0 {
		return errors.New("estimated costs must be non-negative")
	}
	if d.Mode == DecisionDirectCostException && d.EstimatedDirectCost >= d.EstimatedDelegatedCost {
		return errors.New(ReasonDirectCostNotCheaper)
	}
	if err := validatePermissionClasses(d.ApprovedPermissionClasses); err != nil {
		return err
	}
	if err := validateRoute(d.Route); err != nil {
		return err
	}
	if !subset(d.Route.PermissionClasses, d.ApprovedPermissionClasses) {
		return errors.New(ReasonPermissionWidened)
	}
	if !contains(d.Route.CapabilityClasses, model.CapabilityBoundedImplementation) {
		return errors.New("implementation route lacks bounded_implementation capability")
	}
	return nil
}

func validatePreflight(p *RoutePreflight) error {
	if p == nil || p.UnitID == "" || !validRole(p.Role) || !validPreflightStatus(p.Status) {
		return errors.New("invalid preflight")
	}
	if err := validateRoute(p.Route); err != nil {
		return err
	}
	if p.Status == PreflightAvailable && p.ResponseDigest == "" {
		return errors.New("available preflight missing response_digest")
	}
	if _, err := parseTime(p.ExpiresAt); err != nil {
		return errors.New("invalid preflight expires_at")
	}
	return nil
}

func validateDispatch(d *Dispatch) error {
	if d == nil || d.UnitID == "" || !validRole(d.Role) || d.PreflightRecordDigest == "" || d.BoundedTask == "" || d.WorkContextDigest == "" {
		return errors.New("invalid dispatch")
	}
	if err := validateRoute(d.Route); err != nil {
		return err
	}
	if d.Role == RoleImplementation {
		if d.ImplementationRecordDigest != "" || d.ArtifactDigest != "" || d.ReviewPacketDigest != "" || d.Risk != "" {
			return errors.New("implementation dispatch contains review-only fields")
		}
		if !contains(d.Route.CapabilityClasses, model.CapabilityBoundedImplementation) {
			return errors.New("implementation route lacks bounded_implementation capability")
		}
	} else {
		if d.ImplementationRecordDigest == "" || d.ArtifactDigest == "" || d.ReviewPacketDigest == "" || !model.ValidRouteRiskLevel(d.Risk) {
			return errors.New("review dispatch missing binding or risk")
		}
		if !contains(d.Route.CapabilityClasses, model.CapabilitySemanticReview) {
			return errors.New("review route lacks semantic_review capability")
		}
		if !contains(d.Route.PermissionClasses, model.PermissionReadOnly) || contains(d.Route.PermissionClasses, model.PermissionBoundedWrite) {
			return errors.New("review route is not read-only")
		}
	}
	return nil
}

func validateReview(r *IndependentReview) error {
	if r == nil || r.UnitID == "" || r.ImplementationRecordDigest == "" || r.ReviewDispatchDigest == "" ||
		r.ArtifactDigest == "" || r.ReviewPacketDigest == "" || r.ReviewerID == "" || !validVerdict(r.Verdict) {
		return errors.New("invalid independent review")
	}
	return nil
}

func ValidateRecord(r *Record, requireAnchor bool) error {
	if r == nil || r.Version != 1 || r.RunID == "" || r.Sequence < 1 || r.CoreHeadAnchor == "" || r.Timestamp == "" || r.Actor == "" {
		return errors.New(ReasonInvalidHostRecord)
	}
	if _, err := parseTime(r.Timestamp); err != nil {
		return errors.New(ReasonInvalidHostRecord)
	}
	if r.Sequence == 1 && r.PredecessorHash != "" || r.Sequence > 1 && r.PredecessorHash == "" {
		return errors.New(ReasonInvalidHostRecord)
	}
	if requireAnchor && r.HeadAnchor == "" {
		return errors.New(ReasonInvalidHostRecord)
	}
	set := 0
	if r.Decision != nil {
		set++
	}
	if r.Preflight != nil {
		set++
	}
	if r.Dispatch != nil {
		set++
	}
	if r.Review != nil {
		set++
	}
	if set != 1 {
		return errors.New(ReasonInvalidHostRecord)
	}
	switch r.Kind {
	case KindDecision:
		if r.Decision == nil {
			return errors.New(ReasonInvalidHostRecord)
		}
		return validateDecision(r.Decision)
	case KindPreflight:
		if r.Preflight == nil {
			return errors.New(ReasonInvalidHostRecord)
		}
		return validatePreflight(r.Preflight)
	case KindDispatch:
		if r.Dispatch == nil {
			return errors.New(ReasonInvalidHostRecord)
		}
		return validateDispatch(r.Dispatch)
	case KindReview:
		if r.Review == nil {
			return errors.New(ReasonInvalidHostRecord)
		}
		return validateReview(r.Review)
	default:
		return errors.New(ReasonInvalidHostRecord)
	}
}

func latestDecision(records []Record, unitID string) *Record {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Kind == KindDecision && records[i].Decision != nil && records[i].Decision.UnitID == unitID {
			return &records[i]
		}
	}
	return nil
}

func recordByDigest(records []Record, digest string) *Record {
	for i := range records {
		if records[i].HeadAnchor == digest {
			return &records[i]
		}
	}
	return nil
}

func failedPreflightAfter(records []Record, unitID string, afterSequence int64) bool {
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Sequence <= afterSequence {
			break
		}
		if r.Kind == KindPreflight && r.Preflight != nil && r.Preflight.UnitID == unitID && r.Preflight.Role == RoleImplementation && r.Preflight.Status != PreflightAvailable {
			return true
		}
	}
	return false
}

// EvaluateDecision validates a decision and prevents a route/mode change from
// silently replacing an earlier decision. A superseding decision must name the
// exact previous decision; a route change after an unavailable preflight is the
// approved escalation case, while any other supersession is a fresh human
// route decision represented by a new route_approval_digest.
func EvaluateDecision(records []Record, d OrchestrationDecision) error {
	if err := validateDecision(&d); err != nil {
		return err
	}
	prev := latestDecision(records, d.UnitID)
	if prev == nil {
		if d.SupersedesDecisionDigest != "" {
			return errors.New(ReasonDecisionSupersession)
		}
		return nil
	}
	changed := prev.Decision.Mode != d.Mode || !sameRoute(prev.Decision.Route, d.Route)
	if !changed {
		if d.SupersedesDecisionDigest != "" && d.SupersedesDecisionDigest != prev.HeadAnchor {
			return errors.New(ReasonDecisionSupersession)
		}
		return nil
	}
	if d.SupersedesDecisionDigest != prev.HeadAnchor {
		return errors.New(ReasonDecisionSupersession)
	}
	if d.Route.RouteApprovalDigest == prev.Decision.Route.RouteApprovalDigest && !failedPreflightAfter(records, d.UnitID, prev.Sequence) {
		return errors.New(ReasonDecisionSupersession)
	}
	return nil
}

func EvaluatePreflight(records []Record, p RoutePreflight, at time.Time) error {
	if err := validatePreflight(&p); err != nil {
		return err
	}
	expires, _ := parseTime(p.ExpiresAt)
	if !expires.After(at) {
		return errors.New(ReasonPreflightStale)
	}
	if p.Role == RoleImplementation {
		d := latestDecision(records, p.UnitID)
		if d == nil || d.Decision.Mode != DecisionDelegate {
			return errors.New(ReasonDecisionMissing)
		}
		if !sameRoute(d.Decision.Route, p.Route) {
			return errors.New(ReasonRouteMismatch)
		}
	} else {
		if _, err := RequireImplementation(records, p.UnitID); err != nil {
			return err
		}
		if !contains(p.Route.PermissionClasses, model.PermissionReadOnly) || contains(p.Route.PermissionClasses, model.PermissionBoundedWrite) {
			return errors.New(ReasonPermissionWidened)
		}
	}
	return nil
}

func implementationRoute(records []Record, digest string) (Route, string, error) {
	r := recordByDigest(records, digest)
	if r == nil {
		return Route{}, "", errors.New(ReasonImplementationMissing)
	}
	if r.Kind == KindDecision && r.Decision != nil && r.Decision.Mode == DecisionDirectCostException {
		return r.Decision.Route, r.Decision.WorkContextDigest, nil
	}
	if r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Role == RoleImplementation {
		return r.Dispatch.Route, r.Dispatch.WorkContextDigest, nil
	}
	return Route{}, "", errors.New(ReasonImplementationMissing)
}

func EvaluateDispatch(records []Record, d Dispatch, at time.Time) error {
	if err := validateDispatch(&d); err != nil {
		return err
	}
	pr := recordByDigest(records, d.PreflightRecordDigest)
	if pr == nil || pr.Kind != KindPreflight || pr.Preflight == nil || pr.Preflight.UnitID != d.UnitID || pr.Preflight.Role != d.Role {
		return errors.New(ReasonPreflightMissing)
	}
	if !sameRoute(pr.Preflight.Route, d.Route) {
		return errors.New(ReasonRouteMismatch)
	}
	if pr.Preflight.Status != PreflightAvailable {
		return errors.New(ReasonPreflightUnavailable)
	}
	preflightAt, err := parseTime(pr.Timestamp)
	if err != nil || at.Before(preflightAt) {
		return errors.New(ReasonPreflightStale)
	}
	expires, _ := parseTime(pr.Preflight.ExpiresAt)
	if at.After(expires) {
		return errors.New(ReasonPreflightStale)
	}

	if d.Role == RoleImplementation {
		decision := latestDecision(records, d.UnitID)
		if decision == nil || decision.Decision.Mode != DecisionDelegate {
			return errors.New(ReasonDecisionMissing)
		}
		if !sameRoute(decision.Decision.Route, d.Route) {
			return errors.New(ReasonRouteMismatch)
		}
		if d.BoundedTask != decision.Decision.BoundedJob {
			return errors.New("HOST_BOUNDED_TASK_MISMATCH")
		}
		if !subset(d.Route.PermissionClasses, decision.Decision.ApprovedPermissionClasses) {
			return errors.New(ReasonPermissionWidened)
		}
		return nil
	}

	currentImplementation, err := RequireImplementation(records, d.UnitID)
	if err != nil {
		return err
	}
	if currentImplementation != d.ImplementationRecordDigest {
		return errors.New(ReasonImplementationAlreadyBound)
	}
	implRoute, implContext, err := implementationRoute(records, d.ImplementationRecordDigest)
	if err != nil {
		return err
	}
	if d.Route.WorkerID == implRoute.WorkerID || d.Route.Model == implRoute.Model || d.WorkContextDigest == implContext {
		return errors.New(ReasonReviewerNotIndependent)
	}
	if d.Risk == model.RouteRiskHigh || d.Risk == model.RouteRiskCritical {
		if d.Route.ProviderFamily == implRoute.ProviderFamily || d.Route.ModelFamily == implRoute.ModelFamily {
			return errors.New(ReasonReviewerNotIndependent)
		}
	}
	return nil
}

func EvaluateReview(records []Record, review IndependentReview) error {
	if err := validateReview(&review); err != nil {
		return err
	}
	currentImplementation, err := RequireImplementation(records, review.UnitID)
	if err != nil {
		return err
	}
	if currentImplementation != review.ImplementationRecordDigest {
		return errors.New(ReasonImplementationAlreadyBound)
	}
	dispatchRecord := recordByDigest(records, review.ReviewDispatchDigest)
	if dispatchRecord == nil || dispatchRecord.Kind != KindDispatch || dispatchRecord.Dispatch == nil || dispatchRecord.Dispatch.Role != RoleReview {
		return errors.New(ReasonReviewMissing)
	}
	d := dispatchRecord.Dispatch
	if d.UnitID != review.UnitID || d.ImplementationRecordDigest != review.ImplementationRecordDigest || d.ArtifactDigest != review.ArtifactDigest || d.ReviewPacketDigest != review.ReviewPacketDigest || d.Route.WorkerID != review.ReviewerID {
		return errors.New(ReasonReviewPacketMismatch)
	}
	return nil
}

// RequireImplementation returns the record digest that represents the current
// unit's implementation path. For a direct-cost exception the decision itself
// is the implementation identity; for delegated work it is the latest matching
// implementation dispatch after the current decision.
func RequireImplementation(records []Record, unitID string) (string, error) {
	decision := latestDecision(records, unitID)
	if decision == nil {
		return "", errors.New(ReasonDecisionMissing)
	}
	if decision.Decision.Mode == DecisionDirectCostException {
		return decision.HeadAnchor, nil
	}
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Sequence <= decision.Sequence {
			break
		}
		if r.Kind == KindDispatch && r.Dispatch != nil && r.Dispatch.Role == RoleImplementation && r.Dispatch.UnitID == unitID && sameRoute(r.Dispatch.Route, decision.Decision.Route) {
			return r.HeadAnchor, nil
		}
	}
	return "", errors.New(ReasonImplementationMissing)
}

func RequireApprovedReview(records []Record, unitID string) error {
	implementation, err := RequireImplementation(records, unitID)
	if err != nil {
		return err
	}
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Kind == KindReview && r.Review != nil && r.Review.UnitID == unitID && r.Review.ImplementationRecordDigest == implementation {
			if r.Review.Verdict == ReviewApprove {
				return nil
			}
			return errors.New(ReasonReviewMissing)
		}
	}
	return errors.New(ReasonReviewMissing)
}
