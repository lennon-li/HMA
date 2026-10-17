package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/transition"
)

// reviewArtifactV2 is the retired hma.host.review-artifact/v2 digest, as a
// review recorded before the tracked-parent correction would carry it.
func reviewArtifactV2(head, diff, worktree string) string {
	b, _ := json.Marshal(struct {
		Schema       string `json:"schema"`
		ProducedHead string `json:"produced_head_digest"`
		Diff         string `json:"diff_digest"`
		Worktree     string `json:"worktree_digest,omitempty"`
	}{"hma.host.review-artifact/v2", head, diff, worktree})
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// R6-01: a predecessor clean v2 review cannot authorize Verification; a fresh
// v3 review of the same clean artifact can.
func TestGateBClosureR6PredecessorCleanReviewRetired(t *testing.T) {
	for _, tc := range []struct {
		name    string
		digest  func(head, diff string) string
		refused bool
	}{
		{"legacy-v2-clean", func(h, d string) string { return reviewArtifactV2(h, d, "") }, true},
		{"fresh-v3-clean", func(h, d string) string { return hoststate.ReviewArtifactDigest(h, d, "") }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			_, _, head, diff := ciRepo(t)
			seedRouteAuthorization(t, dir, "run", "U1", "approved-route")
			artifact := tc.digest(head, diff)
			recordReviewedImplementation(t, dir, artifact)
			err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
				UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
				ProducedHeadDigest: head, DiffDigest: diff,
			}})
			if tc.refused && (err == nil || err.Error() != hoststate.ReasonReviewArtifactMismatch) {
				t.Fatalf("predecessor clean review: err = %v, want %s", err, hoststate.ReasonReviewArtifactMismatch)
			}
			if !tc.refused && err != nil {
				t.Fatalf("fresh review refused: %v", err)
			}
		})
	}
}

// recordReviewedImplementation records a delegated implementation and an
// approving independent review that carries artifact.
func recordReviewedImplementation(t *testing.T, dir, artifact string) {
	t.Helper()
	input := filepath.Join(t.TempDir(), "host.json")
	impl := cliRoute("jax", "openai", "openai", "luna", "gpt", "approved-route", false)
	reviewer := cliRoute("ming", "anthropic", "anthropic", "sonnet", "claude", "approved-route", true)
	step := func(cmd string, v any) string {
		t.Helper()
		writeJSON(t, input, v)
		if err := run([]string{"host", cmd, "--input", input, "--store", dir}); err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		h, err := hoststate.NewStore(dir).Load("run")
		if err != nil {
			t.Fatal(err)
		}
		return h[len(h)-1].HeadAnchor
	}
	env := func(ts string) hostEnvelope { return hostEnvelope{RunID: "run", Actor: "fury", Timestamp: ts} }
	step("decision", hostDecisionInput{hostEnvelope: env("2026-09-16T12:00:00Z"), Decision: hoststate.OrchestrationDecision{
		UnitID: "U1", Mode: hoststate.DecisionDelegate, BoundedJob: "change one bounded unit",
		EstimatedDirectCost: 8, EstimatedDelegatedCost: 4, CostUnit: "relative",
		ComparisonBasis: "dispatch plus execution plus review", Rationale: "delegation is cheaper",
		ApprovedPermissionClasses: []string{model.PermissionBoundedWrite, model.PermissionNoExternalEffect}, WorkContextDigest: "ctx-fury", Route: impl,
	}})
	pf := step("preflight", hostPreflightInput{hostEnvelope: env("2026-09-16T12:01:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-impl", ExpiresAt: "2026-09-16T12:06:00Z",
	}})
	implementation := step("dispatch", hostDispatchInput{hostEnvelope: env("2026-09-16T12:02:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleImplementation, Route: impl, PreflightRecordDigest: pf,
		BoundedTask: "change one bounded unit", WorkContextDigest: "ctx-jax",
	}})
	rpf := step("preflight", hostPreflightInput{hostEnvelope: env("2026-09-16T12:03:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:08:00Z",
	}})
	rd := step("dispatch", hostDispatchInput{hostEnvelope: env("2026-09-16T12:04:00Z"), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: rpf,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-ming", ImplementationRecordDigest: implementation,
		ArtifactDigest: artifact, ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}})
	step("review", hostReviewInput{hostEnvelope: env("2026-09-16T12:05:00Z"), Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: rd,
		ArtifactDigest: artifact, ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}})
}
