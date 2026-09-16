package host

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/model"
)

func testRoute(worker, provider, providerFamily, modelName, modelFamily, approval string, review bool) Route {
	caps := []string{model.CapabilityBoundedImplementation, model.CapabilityEvidenceOutputCapture}
	perms := []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	if review {
		caps = []string{model.CapabilitySemanticReview, model.CapabilityEvidenceOutputCapture}
		perms = []string{model.PermissionReadOnly, model.PermissionNoExternalEffect}
	}
	return Route{
		WorkerID: worker, Provider: provider, ProviderFamily: providerFamily,
		Model: modelName, ModelFamily: modelFamily, ProfileDigest: "profile-" + worker,
		AccessService: "service-" + provider, AccessServiceDigest: "service-digest-" + provider,
		Runtime: "runtime-" + worker, ReasoningEffort: "medium", ExecutionContext: model.ExecutionContextProxy,
		CapabilityClasses: caps, PermissionClasses: perms, RouteApprovalDigest: approval,
	}
}

func decisionFor(route Route, mode DecisionMode) OrchestrationDecision {
	d := OrchestrationDecision{
		UnitID: "U1", Mode: mode, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "bounded comparison",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect},
		WorkContextDigest:         "ctx-implementer", Route: route,
	}
	if mode == DecisionDirectCostException {
		d.EstimatedDirectCost = 3
		d.EstimatedDelegatedCost = 7
	}
	return d
}

func record(seq int64, head string, kind RecordKind) Record {
	return Record{Version: 1, Kind: kind, RunID: "run", Sequence: seq, HeadAnchor: head, CoreHeadAnchor: "core", Timestamp: "2026-09-16T12:00:00Z", Actor: "fury"}
}

func TestDirectCostExceptionMustActuallyBeCheaper(t *testing.T) {
	d := decisionFor(testRoute("fury", "google", "google", "flash", "gemini", "route-a", false), DecisionDirectCostException)
	d.EstimatedDirectCost = d.EstimatedDelegatedCost
	if err := EvaluateDecision(nil, d); err == nil || err.Error() != ReasonDirectCostNotCheaper {
		t.Fatalf("err = %v, want %s", err, ReasonDirectCostNotCheaper)
	}
}

func TestImplementationPreflightMustMatchDecision(t *testing.T) {
	impl := testRoute("jax", "openai", "openai", "luna", "gpt", "route-a", false)
	d := decisionFor(impl, DecisionDelegate)
	r := record(1, "decision", KindDecision)
	r.Decision = &d
	other := impl
	other.Model = "terra"
	p := RoutePreflight{UnitID: "U1", Role: RoleImplementation, Route: other, Status: PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:05:00Z"}
	if err := EvaluatePreflight([]Record{r}, p, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)); err == nil || err.Error() != ReasonRouteMismatch {
		t.Fatalf("err = %v, want %s", err, ReasonRouteMismatch)
	}
}

func TestDispatchFailsClosedForUnavailableAndStalePreflight(t *testing.T) {
	impl := testRoute("jax", "openai", "openai", "luna", "gpt", "route-a", false)
	d := decisionFor(impl, DecisionDelegate)
	dr := record(1, "decision", KindDecision)
	dr.Decision = &d
	for _, tc := range []struct {
		name      string
		status    PreflightStatus
		dispatch  time.Time
		expiresAt string
		want      string
	}{
		{"quota", PreflightQuotaExhausted, time.Date(2026, 9, 16, 12, 1, 0, 0, time.UTC), "2026-09-16T12:05:00Z", ReasonPreflightUnavailable},
		{"stale", PreflightAvailable, time.Date(2026, 9, 16, 12, 6, 0, 0, time.UTC), "2026-09-16T12:05:00Z", ReasonPreflightStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := RoutePreflight{UnitID: "U1", Role: RoleImplementation, Route: impl, Status: tc.status, ResponseDigest: "response", ExpiresAt: tc.expiresAt}
			pr := record(2, "preflight", KindPreflight)
			pr.Preflight = &p
			dispatch := Dispatch{UnitID: "U1", Role: RoleImplementation, Route: impl, PreflightRecordDigest: "preflight", BoundedTask: d.BoundedJob, WorkContextDigest: "ctx-worker"}
			err := EvaluateDispatch([]Record{dr, pr}, dispatch, tc.dispatch)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
		})
	}
}

func delegatedImplementationHistory() ([]Record, Route, string) {
	impl := testRoute("jax", "openai", "openai", "luna", "gpt", "route-a", false)
	d := decisionFor(impl, DecisionDelegate)
	dr := record(1, "decision", KindDecision)
	dr.Decision = &d
	p := RoutePreflight{UnitID: "U1", Role: RoleImplementation, Route: impl, Status: PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:05:00Z"}
	pr := record(2, "preflight", KindPreflight)
	pr.Preflight = &p
	dispatch := Dispatch{UnitID: "U1", Role: RoleImplementation, Route: impl, PreflightRecordDigest: "preflight", BoundedTask: d.BoundedJob, WorkContextDigest: "ctx-worker"}
	ir := record(3, "implementation", KindDispatch)
	ir.Dispatch = &dispatch
	return []Record{dr, pr, ir}, impl, ir.HeadAnchor
}

func TestReviewIndependenceRules(t *testing.T) {
	history, impl, implementation := delegatedImplementationHistory()
	baseReview := testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)

	cases := []struct {
		name string
		edit func(*Route, *Dispatch)
		want string
	}{
		{"self-review", func(r *Route, _ *Dispatch) { r.WorkerID = impl.WorkerID }, ReasonReviewerNotIndependent},
		{"same-model", func(r *Route, _ *Dispatch) { r.Model = impl.Model }, ReasonReviewerNotIndependent},
		{"same-provider-high-risk", func(r *Route, _ *Dispatch) { r.ProviderFamily = impl.ProviderFamily }, ReasonReviewerNotIndependent},
		{"same-context", func(_ *Route, d *Dispatch) { d.WorkContextDigest = "ctx-worker" }, ReasonReviewerNotIndependent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reviewRoute := baseReview
			p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: reviewRoute, Status: PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:10:00Z"}
			pr := record(4, "review-preflight", KindPreflight)
			pr.Timestamp = "2026-09-16T12:02:00Z"
			pr.Preflight = &p
			d := Dispatch{UnitID: "U1", Role: RoleReview, Route: reviewRoute, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review fixed packet", WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh}
			tc.edit(&reviewRoute, &d)
			d.Route = reviewRoute
			pr.Preflight.Route = reviewRoute
			err := EvaluateDispatch(append(history, pr), d, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC))
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestValidIndependentReviewFlow(t *testing.T) {
	history, _, implementation := delegatedImplementationHistory()
	reviewRoute := testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)
	p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: reviewRoute, Status: PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:10:00Z"}
	pr := record(4, "review-preflight", KindPreflight)
	pr.Timestamp = "2026-09-16T12:02:00Z"
	pr.Preflight = &p
	if err := EvaluatePreflight(history, p, time.Date(2026, 9, 16, 12, 2, 0, 0, time.UTC)); err != nil {
		t.Fatalf("review preflight: %v", err)
	}
	history = append(history, pr)
	d := Dispatch{UnitID: "U1", Role: RoleReview, Route: reviewRoute, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review fixed packet", WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh}
	if err := EvaluateDispatch(history, d, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC)); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	rd := record(5, "review-dispatch", KindDispatch)
	rd.Timestamp = "2026-09-16T12:03:00Z"
	rd.Dispatch = &d
	history = append(history, rd)
	review := IndependentReview{UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: rd.HeadAnchor, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: ReviewApprove}
	if err := EvaluateReview(history, review); err != nil {
		t.Fatalf("review result: %v", err)
	}
	rr := record(6, "review", KindReview)
	rr.Review = &review
	history = append(history, rr)
	if err := RequireApprovedReview(history, "U1"); err != nil {
		t.Fatalf("approval gate: %v", err)
	}
}

func TestFailedPreflightCanBeFollowedByExplicitSupersedingDecision(t *testing.T) {
	primary := testRoute("jax", "openai", "openai", "luna", "gpt", "route-a", false)
	d1 := decisionFor(primary, DecisionDelegate)
	r1 := record(1, "decision-a", KindDecision)
	r1.Decision = &d1
	p := RoutePreflight{UnitID: "U1", Role: RoleImplementation, Route: primary, Status: PreflightQuotaExhausted, ResponseDigest: "quota", ExpiresAt: "2026-09-16T12:05:00Z"}
	r2 := record(2, "preflight-a", KindPreflight)
	r2.Preflight = &p
	escalation := testRoute("wei", "google", "google", "flash", "gemini", "route-b", false)
	d2 := decisionFor(escalation, DecisionDelegate)
	d2.SupersedesDecisionDigest = r1.HeadAnchor
	if err := EvaluateDecision([]Record{r1, r2}, d2); err != nil {
		t.Fatalf("approved escalation rejected: %v", err)
	}
}

func TestDirectCostExceptionCanBeIndependentlyReviewed(t *testing.T) {
	impl := testRoute("fury", "google", "google", "flash", "gemini", "route-direct", false)
	d := decisionFor(impl, DecisionDirectCostException)
	dr := record(1, "direct-decision", KindDecision)
	dr.Decision = &d
	history := []Record{dr}
	implementation, err := RequireImplementation(history, "U1")
	if err != nil || implementation != dr.HeadAnchor {
		t.Fatalf("direct implementation = %q, %v", implementation, err)
	}
	reviewer := testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)
	p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: reviewer, Status: PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:10:00Z"}
	if err := EvaluatePreflight(history, p, time.Date(2026, 9, 16, 12, 2, 0, 0, time.UTC)); err != nil {
		t.Fatalf("direct review preflight: %v", err)
	}
	pr := record(2, "review-preflight", KindPreflight)
	pr.Timestamp = "2026-09-16T12:02:00Z"
	pr.Preflight = &p
	history = append(history, pr)
	rdPayload := Dispatch{UnitID: "U1", Role: RoleReview, Route: reviewer, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review direct work", WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskMedium}
	if err := EvaluateDispatch(history, rdPayload, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC)); err != nil {
		t.Fatalf("direct review dispatch: %v", err)
	}
	rd := record(3, "review-dispatch", KindDispatch)
	rd.Timestamp = "2026-09-16T12:03:00Z"
	rd.Dispatch = &rdPayload
	history = append(history, rd)
	review := IndependentReview{UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: rd.HeadAnchor, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: ReviewApprove}
	if err := EvaluateReview(history, review); err != nil {
		t.Fatalf("direct review: %v", err)
	}
}

func TestReviewResultMustMatchFixedPacket(t *testing.T) {
	history, _, implementation := delegatedImplementationHistory()
	reviewer := testRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)
	p := RoutePreflight{UnitID: "U1", Role: RoleReview, Route: reviewer, Status: PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:10:00Z"}
	pr := record(4, "review-preflight", KindPreflight)
	pr.Timestamp = "2026-09-16T12:02:00Z"
	pr.Preflight = &p
	history = append(history, pr)
	d := Dispatch{UnitID: "U1", Role: RoleReview, Route: reviewer, PreflightRecordDigest: pr.HeadAnchor, BoundedTask: "review", WorkContextDigest: "ctx-review", ImplementationRecordDigest: implementation, ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskMedium}
	rd := record(5, "review-dispatch", KindDispatch)
	rd.Dispatch = &d
	history = append(history, rd)
	review := IndependentReview{UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: rd.HeadAnchor, ArtifactDigest: "artifact", ReviewPacketDigest: "different-packet", ReviewerID: "ming", Verdict: ReviewApprove}
	if err := EvaluateReview(history, review); err == nil || err.Error() != ReasonReviewPacketMismatch {
		t.Fatalf("err = %v, want %s", err, ReasonReviewPacketMismatch)
	}
}

func TestStoreDetectsHostRecordTampering(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	d := decisionFor(testRoute("jax", "openai", "openai", "luna", "gpt", "route-a", false), DecisionDelegate)
	r := Record{Version: 1, Kind: KindDecision, RunID: "run", CoreHeadAnchor: "core", Timestamp: "2026-09-16T12:00:00Z", Actor: "fury", Decision: &d}
	if err := s.Append(&r); err != nil {
		t.Fatal(err)
	}
	path := dir + "/host/run/00000000000000000001.json"
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), "change one bounded unit", "changed behind HMA", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("run"); err == nil || !strings.Contains(err.Error(), ReasonHostChainCorrupt) {
		t.Fatalf("tamper load = %v", err)
	}
}
