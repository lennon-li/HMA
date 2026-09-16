package main

import (
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/transition"
)

func cliRoute(worker, provider, providerFamily, modelName, modelFamily, approval string, review bool) hoststate.Route {
	caps := []string{model.CapabilityBoundedImplementation, model.CapabilityEvidenceOutputCapture}
	perms := []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}
	if review {
		caps = []string{model.CapabilitySemanticReview, model.CapabilityEvidenceOutputCapture}
		perms = []string{model.PermissionReadOnly, model.PermissionNoExternalEffect}
	}
	return hoststate.Route{
		WorkerID: worker, Provider: provider, ProviderFamily: providerFamily,
		Model: modelName, ModelFamily: modelFamily, ProfileDigest: "profile-" + worker,
		AccessService: "service-" + provider, AccessServiceDigest: "service-digest-" + provider,
		Runtime: "runtime-" + worker, ReasoningEffort: "medium", ExecutionContext: model.ExecutionContextProxy,
		CapabilityClasses: caps, PermissionClasses: perms, RouteApprovalDigest: approval,
	}
}

func TestHostCLIEndToEndDelegationAndReview(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(t.TempDir(), "host.json")
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "route-impl", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "route-review", true)

	decision := hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "delegation is cheaper",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: impl,
	}}
	writeJSON(t, input, decision)
	if err := run([]string{"host", "decision", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("decision: %v", err)
	}

	preflight := hostPreflightInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:01:00Z"}, Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-impl", ExpiresAt: "2026-09-16T12:06:00Z",
	}}
	writeJSON(t, input, preflight)
	if err := run([]string{"host", "preflight", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("implementation preflight: %v", err)
	}
	history, err := hoststate.NewStore(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	implPreflight := history[len(history)-1].HeadAnchor

	dispatch := hostDispatchInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:02:00Z"}, Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: implPreflight,
		BoundedTask: "change one bounded unit", WorkContextDigest: "ctx-jax",
	}}
	writeJSON(t, input, dispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("implementation dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	implementation := history[len(history)-1].HeadAnchor

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageImplementationReview,
	}}); err != nil {
		t.Fatalf("implementation gate: %v", err)
	}

	reviewPreflight := hostPreflightInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:03:00Z"}, Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:08:00Z",
	}}
	writeJSON(t, input, reviewPreflight)
	if err := run([]string{"host", "preflight", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review preflight: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewPreflightDigest := history[len(history)-1].HeadAnchor

	reviewDispatch := hostDispatchInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:04:00Z"}, Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: reviewPreflightDigest,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: implementation,
		ArtifactDigest: "artifact", ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}}
	writeJSON(t, input, reviewDispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewDispatchDigest := history[len(history)-1].HeadAnchor

	review := hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: reviewDispatchDigest,
		ArtifactDigest: "artifact", ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}}
	writeJSON(t, input, review)
	if err := run([]string{"host", "review", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review result: %v", err)
	}

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
	}}); err != nil {
		t.Fatalf("review gate: %v", err)
	}
}

func TestHostCLIFailsClosedOnQuotaExhaustion(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(t.TempDir(), "host.json")
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "route-impl", false)
	decision := hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative", ComparisonBasis: "cost", Rationale: "delegate",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: impl,
	}}
	writeJSON(t, input, decision)
	if err := run([]string{"host", "decision", "--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}
	preflight := hostPreflightInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:01:00Z"}, Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, Status: hoststate.PreflightQuotaExhausted, ResponseDigest: "quota", ExpiresAt: "2026-09-16T12:06:00Z",
	}}
	writeJSON(t, input, preflight)
	if err := run([]string{"host", "preflight", "--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}
	history, _ := hoststate.NewStore(dir).Load("run")
	dispatch := hostDispatchInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:02:00Z"}, Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: history[len(history)-1].HeadAnchor,
		BoundedTask: "change one bounded unit", WorkContextDigest: "ctx-jax",
	}}
	writeJSON(t, input, dispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err == nil || err.Error() != hoststate.ReasonPreflightUnavailable {
		t.Fatalf("dispatch err = %v, want %s", err, hoststate.ReasonPreflightUnavailable)
	}
}

func TestHostTransitionGateRejectsUnrecordedCodingPath(t *testing.T) {
	dir := t.TempDir()
	in := transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageImplementationReview,
	}}
	if err := requireHostTransitionGate(dir, in); err == nil || err.Error() != hoststate.ReasonDecisionMissing {
		t.Fatalf("gate err = %v, want %s", err, hoststate.ReasonDecisionMissing)
	}
}
