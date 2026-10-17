package main

import (
	"context"
	"fmt"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGateBR2ReviewExactArtifact(t *testing.T) {
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
		ArtifactDigest: hoststate.ReviewArtifactDigest("reviewed-head", "reviewed-diff", "approved-dirty"), ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}}
	writeJSON(t, input, reviewDispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewDispatchDigest := history[len(history)-1].HeadAnchor

	review := hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: reviewDispatchDigest,
		ArtifactDigest: hoststate.ReviewArtifactDigest("reviewed-head", "reviewed-diff", "approved-dirty"), ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}}
	writeJSON(t, input, review)
	if err := run([]string{"host", "review", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review result: %v", err)
	}

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff", WorktreeDigest: "approved-dirty",
	}, DirtyWorktreeApproved: true, ExpectedWorktreeDigest: "approved-dirty"}); err != nil {
		t.Fatalf("review gate: %v", err)
	}

	for _, tc := range []struct{ h, d, w string }{{"OTHER-HEAD", "reviewed-diff", "approved-dirty"}, {"reviewed-head", "OTHER-DIFF", "approved-dirty"}, {"reviewed-head", "reviewed-diff", "OTHER-DIRTY"}, {"reviewed-head", "reviewed-diff", ""}} {
		err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification, ProducedHeadDigest: tc.h, DiffDigest: tc.d, WorktreeDigest: tc.w}, DirtyWorktreeApproved: tc.w != "", ExpectedWorktreeDigest: tc.w})
		if err == nil || err.Error() != hoststate.ReasonReviewArtifactMismatch {
			t.Fatalf("changed artifact %+v refusal: %v", tc, err)
		}
	}
}

func TestGateBR2ReviewLegacyDirtyHole(t *testing.T) {
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
		ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff", WorktreeDigest: "",
	}}); err != nil {
		t.Fatalf("review gate: %v", err)
	}

	for _, dirty := range []string{"UNREVIEWED-DIRTY-A", "UNREVIEWED-DIRTY-B"} {
		in := transition.Input{RunID: "run", DirtyWorktreeApproved: true, ExpectedWorktreeDigest: dirty, ExpectedApproval: model.ApprovalBinding{UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification, ProducedHeadDigest: "reviewed-head", DiffDigest: "reviewed-diff"}}
		if err := requireHostTransitionGate(dir, in); err == nil {
			t.Errorf("same review accepted exact dirty content %s with optional approval worktree binding absent", dirty)
		}
	}
}

func TestGateBR2ReviewDirtyFullTransition(t *testing.T) {
	dir := t.TempDir()
	root, base, head, diff := ciRepo(t)
	st := seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	core, _ := st.Load("run")
	a := *core[0].Approval
	a.CurrentStage = model.StageImplementationAuthorization
	a.ProposedTargetStage = model.StageImplementationReview
	a.ChallengeNonce = "enter-review"
	a.StageTimeDigest = core[0].Metadata.HeadAnchor + ":2"
	a.RepositoryIdentityDigest = "owner/repo"
	a.BaseRevisionDigest = base
	entry := model.Record{Kind: model.KindStageTransition, Version: 1, UnitID: "U1", Stage: a.CurrentStage, Control: model.ControlApproved, Approval: &a, Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: core[0].Metadata.HeadAnchor, Timestamp: a.Timestamp, Actor: "human"}}
	if err := st.Append(&entry); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a"), []byte("REVIEWED-DIRTY-A"), 0600)

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
		ArtifactDigest: hoststate.ReviewArtifactDigest(head, diff, ""), ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}}
	writeJSON(t, input, reviewDispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewDispatchDigest := history[len(history)-1].HeadAnchor

	review := hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: reviewDispatchDigest,
		ArtifactDigest: hoststate.ReviewArtifactDigest(head, diff, ""), ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}}
	writeJSON(t, input, review)
	if err := run([]string{"host", "review", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review result: %v", err)
	}

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: head, DiffDigest: diff, WorktreeDigest: "",
	}}); err != nil {
		t.Fatalf("review gate: %v", err)
	}

	// The independently reviewed dirty bytes change. Keep the core and review unchanged.
	os.WriteFile(filepath.Join(root, "a"), []byte("UNREVIEWED-DIRTY-B"), 0600)
	snap, err := repostate.Snapshot(context.Background(), root, base, true)
	if err != nil {
		t.Fatal(err)
	}
	core, err = store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	last := core[len(core)-1]
	now := time.Now().UTC().Truncate(time.Second)
	approval := model.ApprovalBinding{RunID: "run", UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification, TransitionDigest: "human-approved-verification-B", RepositoryIdentityDigest: "owner/repo", BaseRevisionDigest: base, ProducedHeadDigest: head, DiffDigest: diff, AcceptedPlanDigest: "plan", StageTimeDigest: fmt.Sprintf("%s:%d", last.Metadata.HeadAnchor, last.Metadata.Sequence+1), ChallengeNonce: "verify-B", Approver: "human", Timestamp: now.Format(time.RFC3339)}
	in := transition.Input{RunID: "run", RepositoryRoot: root, RepositoryIdentity: "owner/repo", BaseRevision: base, DirtyWorktreeApproved: true, ExpectedWorktreeDigest: snap.Worktree, ExpectedApproval: approval, Approval: approval, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
	writeJSON(t, input, in)
	err = runTransition([]string{"--store", dir, "--input", input})
	if err == nil {
		t.Error("FULL CLI transition advanced to VERIFICATION for UNREVIEWED-DIRTY-B using review of A")
	}
}
