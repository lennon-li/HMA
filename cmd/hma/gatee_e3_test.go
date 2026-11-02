package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
)

// TestNonContractRunCannotClaimSuccess is Decision D10 / architecture
// section 20: a non-contract run cannot record a success-claiming terminal
// outcome, even when it has nothing to credit (the vacuous-credit gap D10
// closes for PARTIAL, §2.4). The refusal runs inside runTransition's closure
// handling, before transition.Apply, so it fires before any stage-legality
// or credit check would.
func TestNonContractRunCannotClaimSuccess(t *testing.T) {
	for _, outcome := range []model.TerminalOutcome{model.OutcomeVerifiedSuccess, model.OutcomeVerifiedWithWaivers, model.OutcomePartial} {
		t.Run(string(outcome), func(t *testing.T) {
			repo, dir := gitRepo(t), t.TempDir()
			cmd := exec.Command("git", "-C", repo, "-c", "user.name=HMA Test", "-c", "user.email=hma@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git commit: %v: %s", err, out)
			}
			snap, err := repostate.Snapshot(context.Background(), repo, "HEAD", false)
			if err != nil {
				t.Fatal(err)
			}
			if outcome == model.OutcomePartial {
				// No PASSED criterion either: this confirms the refusal does
				// not depend on there being anything to credit.
				seed := model.Record{Kind: model.KindCriterion, Version: 1,
					Metadata: model.RecordMetadata{RunID: "run", Sequence: 1, Actor: "human", Timestamp: "2026-09-16T11:00:00Z"},
					Criteria: []model.Criterion{{ID: "C1", Disposition: model.CriterionWaived}, {ID: "C2", Disposition: model.CriterionFailed}},
				}
				if err := store.New(dir).Append(&seed); err != nil {
					t.Fatal(err)
				}
			}
			records, err := store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			state := engine.ProjectStageState(records)
			now := time.Now().UTC().Truncate(time.Second)
			a := model.ApprovalBinding{RunID: "run", TransitionDigest: "target-" + string(outcome), CurrentStage: state.CurrentStage,
				ProposedTargetOutcome: outcome, RepositoryIdentityDigest: "repo", BaseRevisionDigest: snap.Head,
				StageTimeDigest: fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1), AcceptedPlanDigest: "plan",
				ChallengeNonce: "outcome", Approver: "human", Timestamp: now.Format(time.RFC3339)}
			in := transition.Input{RunID: "run", RepositoryRoot: repo, RepositoryIdentity: "repo", BaseRevision: snap.Head,
				ExpectedApproval: a, Approval: a, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
			input := filepath.Join(t.TempDir(), "input.json")
			writeJSON(t, input, in)
			if err := run([]string{"transition", "--input", input, "--store", dir}); err == nil || err.Error() != reasonContractRequiredForSuccessClaim {
				t.Fatalf("%s on a non-contract run: got %v, want %s", outcome, err, reasonContractRequiredForSuccessClaim)
			}
			records, err = store.New(dir).Load("run")
			if err != nil {
				t.Fatal(err)
			}
			if len(records) > 0 {
				if tail := records[len(records)-1]; tail.Kind == model.KindOutcome {
					t.Fatalf("outcome recorded despite refusal: %+v", tail)
				}
			}
		})
	}
}

// TestClosureWithPerChangeReviewOnlyRefused is G-F at the host-gate level
// (architecture section 16 E3): a credited PASSED criterion backed only by a
// per_change-level review is refused at closure with HOST_FINAL_AUDIT_MISSING,
// whatever its risk or family separation (this fixture's review is already
// family-separated and high-risk; the only thing it lacks is being final).
func TestClosureWithPerChangeReviewOnlyRefused(t *testing.T) {
	cf := newCreditFixtureReason(t, hoststate.IssuePerChange, "true", []string{}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	if got := cf.criteria()["C1"]; got != model.CriterionPassed {
		t.Fatalf("C1 = %v, want PASSED", got)
	}
	if err := cf.outcomeGate(); err == nil || err.Error() != reasonFinalAuditMissing {
		t.Fatalf("closure credit backed by a per_change review: got %v, want %s", err, reasonFinalAuditMissing)
	}
}

// TestClosureSameFamilyNormalRiskRefused is G-F's normal-risk path (the live
// gap the design names, §2.3): dispatch risk low, a same-family reviewer
// (allowed, because low risk/non-disputed/non-final/non-family-separation
// packet never sets EvaluateDispatch's "strong" independence requirement),
// and issuing_reason per_change. Before Gate E Phase 4c this credit closed a
// run with no final audit at all; E3 now refuses it.
func TestClosureSameFamilyNormalRiskRefused(t *testing.T) {
	cb := seedContractBoundRun(t, model.RouteRiskLow)
	impl := cliRoute("jax", "openai", "openai", "gpt4", "gpt", "approved-route", false)
	reviewer := cliRoute("review2", "openai", "openai", "gpt5", "gpt", "approved-route", true)
	implementation := cbImplement(t, cb, impl)
	reviewPreflight := cbStep(t, cb, "preflight", hostPreflightInput{hostEnvelope: fxEnv("2026-09-16T12:03:00Z"), Preflight: hoststate.RoutePreflight{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, Status: hoststate.PreflightAvailable, ResponseDigest: "hi-review", ExpiresAt: "2026-09-16T12:08:00Z",
	}})

	ledger := t.TempDir()
	runID := "20261016T000000Z-review2-review-1"
	packetTS := "2026-09-16T12:04:00Z"
	input := filepath.Join(t.TempDir(), "packet.json")
	packetPath := filepath.Join(t.TempDir(), "packet.md")
	writeJSON(t, input, hostReviewPacketInput{hostEnvelope: fxEnv(packetTS), Dispatch: hoststate.Dispatch{
		UnitID: "U1", Role: hoststate.RoleReview, Route: reviewer, PreflightRecordDigest: reviewPreflight, ImplementationRecordDigest: implementation,
		BoundedTask: "review fixed packet", WorkContextDigest: "ctx-review2",
		ArtifactDigest: hoststate.ReviewArtifactDigest(cb.head, cb.diff, ""), Risk: model.RouteRiskLow,
	}, DispatchRunID: runID, IssuingReason: hoststate.IssuePerChange,
		RepositoryRoot: cb.root, BaseRevision: cb.base, Task: "Review the change.",
		ContractPath: cb.contractPath, PolicyPath: cb.policy, LedgerDir: ledger})
	if err := run([]string{"host", "review-packet", "--input", input, "--store", cb.store, "--out", packetPath}); err != nil {
		t.Fatalf("review-packet: %v", err)
	}

	h, err := hoststate.NewStore(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	p := h[len(h)-1].Dispatch
	if p.Packet.FamilySeparation {
		t.Fatal("fixture packet unexpectedly requires family separation; the scenario needs a same-family review to be allowed")
	}
	// A reviewFixture wrapper around this hand-built packet, so the existing
	// report/receipt/import machinery can be reused unchanged.
	f := &reviewFixture{t: t, store: cb.store, root: cb.root, head: cb.head, diff: cb.diff, ledger: ledger,
		packetSHA: p.ReviewPacketDigest, rdID: p.Packet.ReviewDispatchID, runID: runID, packetNonce: p.Packet.ReviewNonce,
		reviewWorker: "review2", launchHead: p.Packet.ExpectedLaunchHead, launchWT: p.Packet.ExpectedLaunchWorktree,
		implementation: implementation, packetTS: packetTS, contractPath: cb.contractPath, policy: cb.policy}
	report := f.report("pass", "isolated", func(r map[string]any) {
		r["requirements"] = []any{map[string]any{"id": "C1", "status": "pass", "evidence_refs": []any{"e1"}}}
	})
	f.receipt(report, func(r map[string]any) { r["model"] = "gpt5"; r["provider"] = "openai" })
	if err := f.importReport(report); err != nil {
		t.Fatalf("import: %v", err)
	}

	creditInput := filepath.Join(t.TempDir(), "credit.json")
	writeJSON(t, creditInput, hostCreditInput{hostEnvelope: fxEnv("2026-09-16T12:07:00Z"), ReviewRecord: f.latestReview(),
		ReportPath: report, ContractPath: cb.contractPath})
	if _, err := captureStdout(t, func() error { return run([]string{"host", "credit", "--input", creditInput, "--store", cb.store}) }); err != nil {
		t.Fatalf("credit: %v", err)
	}

	history, err := hoststate.NewStore(cb.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	snap, err := repostate.Snapshot(context.Background(), cb.root, cb.base, true)
	if err != nil {
		t.Fatal(err)
	}
	artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
	if err := requireIndependentCredit(cb.store, "run", artifact, history); err == nil || err.Error() != reasonFinalAuditMissing {
		t.Fatalf("closure credit backed by a same-family, normal-risk, per_change review: got %v, want %s", err, reasonFinalAuditMissing)
	}
}

// partialTransitionInput builds the transition.Input for a human-approved
// PARTIAL outcome on cf's run, at its current projected stage (PARTIAL is
// legal from any nonterminal stage, architecture section 5.4).
func partialTransitionInput(t *testing.T, cf *creditFixture, nonce string) transition.Input {
	t.Helper()
	core, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state := engine.ProjectStageState(core)
	now := time.Now().UTC().Truncate(time.Second)
	a := model.ApprovalBinding{RunID: "run", TransitionDigest: "target-" + nonce, CurrentStage: state.CurrentStage,
		ProposedTargetOutcome: model.OutcomePartial, RepositoryIdentityDigest: "repo", BaseRevisionDigest: cf.base,
		StageTimeDigest: fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1), AcceptedPlanDigest: cf.planDigest,
		AcceptanceContractDigest: engine.BoundContractDigest(core),
		ChallengeNonce:           nonce, Approver: "human", Timestamp: now.Format(time.RFC3339)}
	// cf.credit (cmd/hma/credit.go) captures the closure artifact with
	// DirtyWorktreeApproved false (hostCreditInput never sets it); this must
	// match so runTransition's own snapshot computes the same artifact.
	return transition.Input{RunID: "run", RepositoryRoot: cf.root, RepositoryIdentity: "repo", BaseRevision: cf.base,
		ExpectedApproval: a, Approval: a, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
}

// TestPartialWithPerChangeCreditRefused is the live G-F red test (Decision
// D7 (2)): on the unmodified tree this is reachable and accepted (PARTIAL's
// closure branch never called requireIndependentCredit, and even once it
// does, requireIndependentCredit never checked for a final-level review).
// After Gate E Phase 4c, recording PARTIAL over a per_change-backed credit is
// refused with HOST_FINAL_AUDIT_MISSING.
func TestPartialWithPerChangeCreditRefused(t *testing.T) {
	cf := newCreditFixtureReason(t, hoststate.IssuePerChange, "true", []string{}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	in := partialTransitionInput(t, cf, "partial-per-change")
	input := filepath.Join(t.TempDir(), "partial.json")
	writeJSON(t, input, in)
	if err := run([]string{"transition", "--input", input, "--store", cf.store}); err == nil || err.Error() != reasonFinalAuditMissing {
		t.Fatalf("PARTIAL recorded over a per_change-backed credit: got %v, want %s", err, reasonFinalAuditMissing)
	}
}

// TestPartialWithStaleCreditRefused is Decision D7's compatibility check: an
// artifact that drifted after credit still refuses with the existing
// HOST_CREDIT_REFUSED, unaffected by the new final-gate check (the artifact
// mismatch is caught before the final-gate check runs).
func TestPartialWithStaleCreditRefused(t *testing.T) {
	cf := newCreditFixture(t, "true")
	if _, err := cf.credit(cf.approve("pass")); err != nil {
		t.Fatalf("credit: %v", err)
	}
	if err := cf.outcomeGate(); err != nil {
		t.Fatalf("control: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cf.root, "a"), []byte("UNREVIEWED"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cf.outcomeGate(); err == nil || err.Error() != hoststate.ReasonCreditRefused {
		t.Fatalf("stale credit after artifact drift: got %v, want %s", err, hoststate.ReasonCreditRefused)
	}
}

// TestPartialWaivedOnlyNeedsNoReview is Decision D7 (2): PARTIAL with only
// WAIVED and PENDING criteria (no PASSED) needs no review at all, because
// requireIndependentCredit's loop only visits PASSED criteria, and WAIVED is
// a human operation that needs nothing more.
func TestPartialWaivedOnlyNeedsNoReview(t *testing.T) {
	cf := newCreditFixtureC2(t, "true", model.CriterionWaived)
	if got := cf.criteria(); got["C1"] != model.CriterionPending || got["C2"] != model.CriterionWaived {
		t.Fatalf("criteria: %v", got)
	}
	if err := cf.outcomeGate(); err != nil {
		t.Fatalf("requireIndependentCredit with no PASSED criterion: %v", err)
	}
	in := partialTransitionInput(t, cf, "partial-waived-only")
	input := filepath.Join(t.TempDir(), "partial.json")
	writeJSON(t, input, in)
	if err := run([]string{"transition", "--input", input, "--store", cf.store}); err != nil {
		t.Fatalf("PARTIAL with WAIVED/PENDING criteria only: %v", err)
	}
	records, err := store.New(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	if tail := records[len(records)-1]; tail.Kind != model.KindOutcome || tail.Outcome != model.OutcomePartial {
		t.Fatalf("tail = %+v", tail)
	}
}

// mutatedReviewDispatchPacket returns a copy of history with the one review
// dispatch's packet binding replaced by edit's result, leaving every other
// record (and every other field of that record) byte-identical. It isolates
// one condition of requireIndependentCredit's final-gate check from the
// other three it is paired with via ||.
func mutatedReviewDispatchPacket(t *testing.T, history []hoststate.Record, edit func(hoststate.ReviewPacketBinding) hoststate.ReviewPacketBinding) []hoststate.Record {
	t.Helper()
	out := append([]hoststate.Record(nil), history...)
	found := false
	for i, r := range out {
		if r.Kind == hoststate.KindDispatch && r.Dispatch != nil && r.Dispatch.Role == hoststate.RoleReview && r.Dispatch.Packet != nil {
			if found {
				t.Fatal("more than one review dispatch packet in history")
			}
			p := edit(*r.Dispatch.Packet)
			d := *r.Dispatch
			d.Packet = &p
			out[i].Dispatch = &d
			found = true
		}
	}
	if !found {
		t.Fatal("no review dispatch packet found to mutate")
	}
	return out
}

// TestClosureRefusedOnBoundContractDigestMismatch is Phil's F2: the final
// gate's packet.BoundContractDigest != contract condition, isolated from the
// other three its check is paired with via ||.
//
// Through any real dispatch-import-credit-closure path this condition cannot
// fire while everything else about the credit still qualifies: review.go
// sets a packet's BoundContractDigest from the contract bound at dispatch
// time; credit.go's reviewBinding (credit.go:434) refuses credit unless that
// same digest is still the one bound at credit time; and
// requireIndependentCredit's own ContractDigest check (credit.go:520) already
// refuses (with the plain HOST_CREDIT_REFUSED, before this packet check ever
// runs) a credit whose recorded contract no longer matches the one bound at
// closure. Chaining these: whenever requireIndependentCredit reaches the
// packet check at all, packet.BoundContractDigest already equals contract.
// This test can only exercise the condition by editing a copy of a real
// packet record in memory (never written back to the store) after a normal
// credit succeeds; it is regression coverage for that chain of invariants,
// not a path reachable through dispatch, import, or credit as they exist
// today.
func TestClosureRefusedOnBoundContractDigestMismatch(t *testing.T) {
	cf := newCreditFixtureArgs(t, "true", []string{}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	history, err := hoststate.NewStore(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	mutated := mutatedReviewDispatchPacket(t, history, func(p hoststate.ReviewPacketBinding) hoststate.ReviewPacketBinding {
		p.BoundContractDigest = p.BoundContractDigest + "-mismatched"
		return p
	})
	snap, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
	if err != nil {
		t.Fatal(err)
	}
	artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
	if err := requireIndependentCredit(cf.store, "run", artifact, mutated); err == nil || err.Error() != reasonFinalAuditMissing {
		t.Fatalf("closure credit backed by a mismatched-contract-digest packet: got %v, want %s", err, reasonFinalAuditMissing)
	}
}

// TestClosureRefusedOnMissingCheckpointSchema is Phil's F3: the final gate's
// packet.CheckpointSchema == "" condition, isolated from the other three its
// check is paired with via ||.
//
// Unlike F2, this is not provably dead: CheckpointSchema (added in Gate E
// Phase 4a) is zero on every pre-Phase-4a stored dispatch record (see
// internal/host's TestLegacyHostRecordAnchorUnchanged), and credit.go's own
// checks never look at it. A review dispatched and imported before Phase 4a
// existed, then credited after HMA is upgraded to code that includes this
// gate, produces exactly this shape: a real, successfully credited PASSED
// criterion whose packet has every other field this check wants
// (IssuingReason final, BoundContractDigest matching) but no
// CheckpointSchema, because nothing set it at the time. This test edits a
// copy of a real (post-Phase-4a) packet record in memory to the same shape,
// which is the same simulation internal/host's legacy-record tests already
// use elsewhere in this codebase, rather than replaying an actual old store.
func TestClosureRefusedOnMissingCheckpointSchema(t *testing.T) {
	cf := newCreditFixtureArgs(t, "true", []string{}, model.CriterionPending)
	report := cf.approve("pass")
	if _, err := cf.credit(report); err != nil {
		t.Fatalf("credit: %v", err)
	}
	history, err := hoststate.NewStore(cf.store).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	mutated := mutatedReviewDispatchPacket(t, history, func(p hoststate.ReviewPacketBinding) hoststate.ReviewPacketBinding {
		p.CheckpointSchema = ""
		return p
	})
	snap, err := repostate.Snapshot(context.Background(), cf.root, cf.base, true)
	if err != nil {
		t.Fatal(err)
	}
	artifact := hoststate.ReviewArtifactDigest(snap.Head, snap.Diff, snap.Worktree)
	if err := requireIndependentCredit(cf.store, "run", artifact, mutated); err == nil || err.Error() != reasonFinalAuditMissing {
		t.Fatalf("closure credit backed by a schema-less (legacy) packet: got %v, want %s", err, reasonFinalAuditMissing)
	}
}

// TestStoppingOutcomesUnaffectedByE3 confirms FAILED, BLOCKED, and UNKNOWN
// (via a human-approved classification) and ABORTED still need no review and
// no bound contract: these are exercised end to end on a non-contract run by
// the existing TestTransitionCLIPhaseA1 (ABORTED, PARTIAL) and
// TestTransitionCLIPhaseA2 (BLOCKED, UNKNOWN, FAILED) CLI tests, both of
// which still pass unchanged after this phase (they are not in the
// success-claiming set runTransition's closure branch now gates, and
// PARTIAL there seeds no PASSED criterion either, so it reaches no new
// check). This test only pins that FAILED itself needs no bound contract,
// as the narrowest direct check not already covered above.
func TestStoppingOutcomesUnaffectedByE3(t *testing.T) {
	dir := t.TempDir()
	repo := gitRepo(t)
	cmd := exec.Command("git", "-C", repo, "-c", "user.name=HMA Test", "-c", "user.email=hma@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	snap, err := repostate.Snapshot(context.Background(), repo, "HEAD", false)
	if err != nil {
		t.Fatal(err)
	}
	records, err := store.New(dir).Load("run")
	if err != nil {
		t.Fatal(err)
	}
	state := engine.ProjectStageState(records)
	now := time.Now().UTC().Truncate(time.Second)
	a := model.ApprovalBinding{RunID: "run", TransitionDigest: "target-failed", CurrentStage: model.StageGrounding,
		ProposedTargetOutcome: model.OutcomeFailed, RepositoryIdentityDigest: "repo", BaseRevisionDigest: snap.Head,
		StageTimeDigest: fmt.Sprintf("%s:%d", state.PredecessorHead, state.Sequence+1), AcceptedPlanDigest: "plan",
		ChallengeNonce: "outcome", Approver: "human", Timestamp: now.Format(time.RFC3339), UnitID: "U1"}
	in := transition.Input{RunID: "run", RepositoryRoot: repo, RepositoryIdentity: "repo", BaseRevision: snap.Head,
		ExpectedApproval: a, Approval: a, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
	in.Classification = &model.Classification{Outcome: model.OutcomeFailed, FindingID: "F1"}
	in.ExpectedApproval.Classification = in.Classification
	in.Approval = in.ExpectedApproval
	input := filepath.Join(t.TempDir(), "input.json")
	writeJSON(t, input, in)
	if err := run([]string{"transition", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("FAILED on a non-contract run (not success-claiming, no E3): %v", err)
	}
}
