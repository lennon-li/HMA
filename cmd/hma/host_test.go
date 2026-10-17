package main

import (
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
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
		WorkerID: worker, Provider: provider, ProviderFamily: providerFamily, ProviderDigest: "provider-digest-" + provider,
		Model: modelName, ModelFamily: modelFamily, ModelDigest: "model-digest-" + modelName, ProfileDigest: "profile-" + worker,
		AccessService: "service-" + provider, AccessServiceDigest: "service-digest-" + provider,
		Runtime: "runtime-" + worker, ReasoningEffort: "medium", ExecutionContext: model.ExecutionContextProxy,
		CapabilityClasses: caps, PermissionClasses: perms, RouteApprovalDigest: approval,
	}
}

func seedRouteAuthorization(t *testing.T, dir, runID, unitID, digest string) *store.Store {
	t.Helper()
	return seedRouteAuthorizationAt(t, dir, runID, unitID, digest, "base")
}

// bindRepository records the run's repository binding.
func bindRepository(t *testing.T, dir, runID, root, identity string) {
	t.Helper()
	input := filepath.Join(t.TempDir(), "bind.json")
	writeJSON(t, input, hostBindRepositoryInput{hostEnvelope: hostEnvelope{RunID: runID, Actor: "fury", Timestamp: "2026-09-16T11:30:00Z"},
		RepositoryRoot: root, RepositoryIdentity: identity})
	if err := run([]string{"host", "bind-repository", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("bind-repository: %v", err)
	}
}

// seedLoopRun creates a repository, a route approval binding its base, and
// the run's repository binding.
func seedLoopRun(t *testing.T, dir string) (string, string) {
	t.Helper()
	root, base, _, _ := ciRepo(t)
	seedRouteAuthorizationAt(t, dir, "run", "U1", "approved-route", base)
	bindRepository(t, dir, "run", root, "repo")
	return root, base
}

// seedRouteAuthorizationAt seeds a route approval that binds base.
func seedRouteAuthorizationAt(t *testing.T, dir, runID, unitID, digest, base string) *store.Store {
	t.Helper()
	s := store.New(dir)
	a := &model.ApprovalBinding{UnitID: unitID, RunID: runID, TransitionDigest: digest,
		CurrentStage: model.StageRouteSelection, ProposedTargetStage: model.StageImplementationAuthorization,
		RepositoryIdentityDigest: "repo", BaseRevisionDigest: base, StageTimeDigest: "head:1",
		AcceptedPlanDigest: "plan", ChallengeNonce: "nonce-route", Approver: "human", Timestamp: "2026-09-16T11:00:00Z"}
	rec := model.Record{UnitID: unitID, Kind: model.KindStageTransition, Version: 1,
		Metadata: model.RecordMetadata{RunID: runID, Sequence: 1, Timestamp: "2026-09-16T11:00:00Z", Actor: "human"},
		Stage:    model.StageRouteSelection, Control: model.ControlApproved, Approval: a}
	if err := s.Append(&rec); err != nil {
		t.Fatalf("seed route authorization: %v", err)
	}
	return s
}

func TestHostCLIEndToEndDelegationAndReview(t *testing.T) {
	dir := t.TempDir()
	seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	input := filepath.Join(t.TempDir(), "host.json")
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)

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
		ArtifactDigest: hoststate.ReviewArtifactDigest("reviewed-head", "reviewed-diff", ""), ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}}
	writeJSON(t, input, reviewDispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewDispatchDigest := history[len(history)-1].HeadAnchor

	review := hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: reviewDispatchDigest,
		ArtifactDigest: hoststate.ReviewArtifactDigest("reviewed-head", "reviewed-diff", ""), ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}}
	writeJSON(t, input, review)
	if err := run([]string{"host", "review", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review result: %v", err)
	}

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff",
	}}); err != nil {
		t.Fatalf("review gate: %v", err)
	}
}

func TestHostCLIFailsClosedOnQuotaExhaustion(t *testing.T) {
	dir := t.TempDir()
	seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	input := filepath.Join(t.TempDir(), "host.json")
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
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

func TestHostTransitionGateRejectsStaleCoreBinding(t *testing.T) {
	dir := t.TempDir()
	coreStore := seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	impl := cliRoute("fury", "openai", "openai", "luna", "gpt", "approved-route", false)
	decision := hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDirectCostException, BoundedJob: "small bounded change",
		EstimatedDirectCost: 1, EstimatedDelegatedCost: 2, CostUnit: "relative", ComparisonBasis: "total cost",
		Rationale: "recorded direct-cost exception", ApprovedPermissionClasses: impl.PermissionClasses,
		WorkContextDigest: "ctx", Route: impl,
	}}
	input := filepath.Join(t.TempDir(), "host.json")
	writeJSON(t, input, decision)
	if err := run([]string{"host", "decision", "--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}

	coreRecords, err := coreStore.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	second := model.Record{
		Kind: model.KindCriterion, Version: 1,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: coreRecords[0].Metadata.HeadAnchor, Timestamp: "2026-09-16T12:01:00Z", Actor: "host"},
		Criteria: []model.Criterion{{ID: "C2", Disposition: model.CriterionPending}},
	}
	if err := store.New(dir).Append(&second); err != nil {
		t.Fatal(err)
	}

	err = requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageImplementationReview,
	}})
	if err == nil || err.Error() != hoststate.ReasonCoreBindingStale {
		t.Fatalf("gate err = %v, want %s", err, hoststate.ReasonCoreBindingStale)
	}
}
