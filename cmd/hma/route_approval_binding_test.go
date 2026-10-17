package main

import (
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/store"
)

func routeApprovalFixture(t *testing.T, supersede bool) string {
	t.Helper()
	dir := t.TempDir()
	seedRouteAuthorization(t, dir, "run", "U1", "core-route-1")
	if supersede {
		s := store.New(dir)
		prior, err := s.Load("run")
		if err != nil {
			t.Fatal(err)
		}
		a := &model.ApprovalBinding{
			UnitID: "U1", RunID: "run", TransitionDigest: "return-to-route",
			CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: model.StageRouteSelection,
			RepositoryIdentityDigest: "repo", BaseRevisionDigest: "base",
			StageTimeDigest: prior[0].Metadata.HeadAnchor + ":2", AcceptedPlanDigest: "plan",
			ChallengeNonce: "nonce-return", Approver: "human", Timestamp: "2026-09-16T11:01:00Z",
		}
		r := model.Record{
			UnitID: "U1", Kind: model.KindStageTransition, Version: 1,
			Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: prior[0].Metadata.HeadAnchor, Timestamp: "2026-09-16T11:01:00Z", Actor: "human"},
			Stage:    model.StageImplementationAuthorization, Control: model.ControlApproved, Approval: a,
		}
		if err := s.Append(&r); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestHostRouteApprovalClaimsBindToCoreTransition(t *testing.T) {
	t.Run("approved route is accepted", func(t *testing.T) {
		dir := routeApprovalFixture(t, false)
		if err := validateRouteApprovalClaim(dir, "run", "U1", "core-route-1"); err != nil {
			t.Fatalf("approved route rejected: %v", err)
		}
	})
	t.Run("forged digest is rejected", func(t *testing.T) {
		dir := routeApprovalFixture(t, false)
		if err := validateRouteApprovalClaim(dir, "run", "U1", "forged-digest"); err == nil || err.Error() != "HOST_ROUTE_APPROVAL_MISMATCH" {
			t.Fatalf("claim error = %v, want HOST_ROUTE_APPROVAL_MISMATCH", err)
		}
	})
	t.Run("different unit is rejected", func(t *testing.T) {
		dir := routeApprovalFixture(t, false)
		if err := validateRouteApprovalClaim(dir, "run", "U2", "core-route-1"); err == nil {
			t.Fatal("route approval from another unit was accepted")
		}
	})
	t.Run("superseded route is rejected", func(t *testing.T) {
		dir := routeApprovalFixture(t, true)
		if err := validateRouteApprovalClaim(dir, "run", "U1", "core-route-1"); err == nil {
			t.Fatal("stale route claim was accepted")
		}
	})
}

// An approved escalation to an earlier gate invalidates prior route approval,
// even when no explicit return to ROUTE_SELECTION was recorded.
func TestHostRouteApprovalInvalidatedByEarlierStageRewind(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unit   string
		target model.Stage
	}{
		{"unit planning rewind", "U1", model.StagePlanning},
		{"unit grounding rewind", "U1", model.StageGrounding},
		{"whole-run planning rewind", "", model.StagePlanning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := routeApprovalFixture(t, false)
			s := store.New(dir)
			records, err := s.Load("run")
			if err != nil {
				t.Fatal(err)
			}
			a := &model.ApprovalBinding{
				UnitID: tc.unit, RunID: "run", TransitionDigest: "human-rewind",
				CurrentStage: model.StageImplementationAuthorization, ProposedTargetStage: tc.target,
				RepositoryIdentityDigest: "repo", BaseRevisionDigest: "base",
				StageTimeDigest: "rewind:2", AcceptedPlanDigest: "plan",
				ChallengeNonce: "nonce-rewind", Approver: "human", Timestamp: "2026-09-16T11:02:00Z",
			}
			rec := model.Record{
				UnitID: tc.unit, Kind: model.KindStageTransition, Version: 1,
				Metadata: model.RecordMetadata{RunID: "run", Sequence: 2, PredecessorHash: records[0].Metadata.HeadAnchor,
					Timestamp: "2026-09-16T11:02:00Z", Actor: "human"},
				Stage: model.StageImplementationAuthorization, Control: model.ControlApproved, Approval: a,
			}
			if err := s.Append(&rec); err != nil {
				t.Fatal(err)
			}
			if err := validateRouteApprovalClaim(dir, "run", "U1", "core-route-1"); err == nil {
				t.Fatal("stale route approval remained valid after an approved rewind")
			}
		})
	}
}

// Every host entrypoint must reject an unapproved digest before it can append
// a decision, preflight, or dispatch record to the host chain.
func TestHostCLIRejectsForgedRouteApprovalsAtEveryStep(t *testing.T) {
	dir := t.TempDir()
	seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
	input := filepath.Join(t.TempDir(), "host.json")
	approved := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	forged := approved
	forged.RouteApprovalDigest = "forged-digest"
	envelope := hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:00:00Z"}
	decision := hostDecisionInput{hostEnvelope: envelope, Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "total", Rationale: "delegate", ApprovedPermissionClasses: approved.PermissionClasses,
		WorkContextDigest: "ctx", Route: forged,
	}}
	check := func(op string, payload any) {
		t.Helper()
		before, err := hoststate.NewStore(dir).Load("run")
		if err != nil {
			t.Fatal(err)
		}
		writeJSON(t, input, payload)
		err = run([]string{"host", op, "--input", input, "--store", dir})
		if err == nil || err.Error() != hoststate.ReasonRouteApprovalMismatch {
			t.Fatalf("%s error = %v, want %s", op, err, hoststate.ReasonRouteApprovalMismatch)
		}
		after, err := hoststate.NewStore(dir).Load("run")
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) {
			t.Fatalf("%s appended an invalid record", op)
		}
	}
	check("decision", decision)
	decision.Decision.Route = approved
	writeJSON(t, input, decision)
	if err := run([]string{"host", "decision", "--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}
	preflight := hostPreflightInput{hostEnvelope: envelope, Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: forged,
		Status: hoststate.PreflightAvailable, ResponseDigest: "hi", ExpiresAt: "2026-09-16T12:05:00Z",
	}}
	check("preflight", preflight)
	preflight.Preflight.Route = approved
	writeJSON(t, input, preflight)
	if err := run([]string{"host", "preflight", "--input", input, "--store", dir}); err != nil {
		t.Fatal(err)
	}
	history, err := hoststate.NewStore(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	dispatch := hostDispatchInput{hostEnvelope: envelope, Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: forged,
		PreflightRecordDigest: history[len(history)-1].HeadAnchor,
		BoundedTask:           "one bounded unit", WorkContextDigest: "ctx",
	}}
	check("dispatch", dispatch)
}
