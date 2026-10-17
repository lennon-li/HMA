package main

import (
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
	"os"
	"path/filepath"
	"testing"
)

func TestGateBReviewGateUnreviewedArtifact(t *testing.T) {
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

	// Keep the valid review and unchanged core chain. Ask to verify other bytes.
	for _, head := range []string{"UNREVIEWED-HEAD-A", "UNREVIEWED-HEAD-B"} {
		err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
			UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
			ProducedHeadDigest: head, DiffDigest: "UNREVIEWED-DIFF",
		}})
		if err == nil {
			t.Errorf("accepted review gate for %s / UNREVIEWED-DIFF although review artifact is literal artifact", head)
		}
	}

	// M4: a human-approved rewind of the unit to Route selection supersedes
	// the route; the second gate must then refuse the earlier review.
	s := store.New(dir)
	core, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	last := core[len(core)-1]
	rw := &model.ApprovalBinding{UnitID: "U1", RunID: "run", TransitionDigest: "rewind-route",
		CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageRouteSelection,
		RepositoryIdentityDigest: "repo", BaseRevisionDigest: "base", StageTimeDigest: last.Metadata.HeadAnchor + ":2",
		AcceptedPlanDigest: "plan", ChallengeNonce: "nonce-rewind-route", Approver: "human", Timestamp: "2026-09-16T12:10:00Z",
		ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff"}
	rec := model.Record{UnitID: "U1", Kind: model.KindStageTransition, Version: 1, Stage: model.StageImplementationReview, Control: model.ControlApproved, Approval: rw,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: last.Metadata.Sequence + 1, PredecessorHash: last.Metadata.HeadAnchor, Timestamp: "2026-09-16T12:10:00Z", Actor: "human"}}
	if err := s.Append(&rec); err != nil {
		t.Fatal(err)
	}
	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff",
	}}); err == nil {
		t.Fatal("review gate accepted after the route was superseded by an approved rewind")
	}
}

func TestGateBResolveTrailingJSON(t *testing.T) {
	dir := t.TempDir()
	seedCriterion(t, dir, "run")
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, waiverInput("run", "audit-nonce"))
	b, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(input, append(b, []byte("\n{\"ignored\":true}\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err = runResolve([]string{"--input", input, "--store", dir}); err == nil {
		t.Error("accepted two JSON documents and appended waiver")
	}
}
func TestGateBTerminalHostDecision(t *testing.T) {
	dir := t.TempDir()
	s := seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	prior, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	a := *prior[0].Approval
	a.CurrentStage = model.StageImplementationAuthorization
	a.ProposedTargetStage = ""
	a.ProposedTargetOutcome = model.OutcomeAborted
	a.ChallengeNonce = "audit-abort"
	a.StageTimeDigest = prior[0].Metadata.HeadAnchor + ":2"
	rec := model.Record{Kind: model.KindOutcome, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Outcome: model.OutcomeAborted, Approval: &a, Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: prior[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if err = s.Append(&rec); err != nil {
		t.Fatal(err)
	}
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	in := hostDecisionInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "host", Timestamp: "2026-09-16T12:00:00Z"}, Decision: hoststate.OrchestrationDecision{UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "bounded", EstimatedDirectCost: 2, EstimatedDelegatedCost: 1, CostUnit: "relative", ComparisonBasis: "total", Rationale: "delegate", ApprovedPermissionClasses: impl.PermissionClasses, WorkContextDigest: "ctx", Route: impl}}
	input := filepath.Join(t.TempDir(), "in.json")
	writeJSON(t, input, in)
	if err = runHostDecision([]string{"--input", input, "--store", dir}); err == nil {
		t.Error("host decision accepted and appended after ABORTED")
	}
}
func TestGateBCINonRepositoryEvidence(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	seedApprovedVerification(t, dir, "run", "owner/repo", "approved-sha")
	setGitHubEnv(t, "owner/repo", "approved-sha")
	if err := runCIGithub([]string{"--store", dir, "--run_id", "run", "--repo-root", root, "--exec", "/bin/true"}); err == nil {
		t.Error("CI captured evidence labelled approved-sha in a non-Git empty directory")
	}
}
func TestGateBCIInsideRepositoryStore(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "worker-writable-store")
	seedApprovedVerification(t, dir, "run", "owner/repo", "approved-sha")
	setGitHubEnv(t, "owner/repo", "approved-sha")
	if err := runCIGithub([]string{"--store", dir, "--run_id", "run", "--repo-root", root, "--exec", "/bin/true"}); err == nil {
		t.Error("CI appended evidence to store inside the claimed repository")
	}
}

// H4: a terminal core run also closes the host transition gate.
func TestGateBTerminalRunClosesTransitionGate(t *testing.T) {
	dir := t.TempDir()
	s := seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	prior, err := s.Load("run")
	if err != nil {
		t.Fatal(err)
	}
	a := *prior[0].Approval
	a.CurrentStage, a.ProposedTargetStage, a.ProposedTargetOutcome = model.StageImplementationAuthorization, "", model.OutcomeAborted
	a.ChallengeNonce, a.StageTimeDigest = "gate-abort", prior[0].Metadata.HeadAnchor+":2"
	rec := model.Record{Kind: model.KindOutcome, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Outcome: model.OutcomeAborted, Approval: &a,
		Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: prior[0].Metadata.HeadAnchor, Actor: "human", Timestamp: a.Timestamp}}
	if err := s.Append(&rec); err != nil {
		t.Fatal(err)
	}
	err = requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageImplementationReview,
	}})
	if err == nil || err.Error() != hoststate.ReasonCoreRunTerminal {
		t.Fatalf("gate on terminal run: %v, want %s", err, hoststate.ReasonCoreRunTerminal)
	}
}
