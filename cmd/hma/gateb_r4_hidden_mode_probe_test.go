package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
	"github.com/lennon-li/HMA/internal/repostate"
	"github.com/lennon-li/HMA/internal/store"
	"github.com/lennon-li/HMA/internal/transition"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func r4FullDirtyReview(t *testing.T, beforeMode, afterMode os.FileMode, bind bool, domain string, staleHost bool, staleApproval bool) {
	dir := t.TempDir()
	root, base, head, diff := ciRepo(t)
	bGit(t, root, "config", "core.filemode", "false")
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
	if e := os.WriteFile(filepath.Join(root, "dirty-evidence"), []byte("REVIEWED-DIRTY-A"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(filepath.Join(root, "a"), beforeMode); e != nil {
		t.Fatal(e)
	}
	reviewedSnap, e := repostate.Snapshot(context.Background(), root, base, true)
	if e != nil {
		t.Fatal(e)
	}
	if domain != "v3" {
		reviewedSnap.Worktree = r4LegacyDigest(head, domain, beforeMode)
	}

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
		ArtifactDigest: hoststate.ReviewArtifactDigest(head, diff, reviewedSnap.Worktree), ReviewPacketDigest: "packet", Risk: model.RouteRiskHigh,
	}}
	writeJSON(t, input, reviewDispatch)
	if err := run([]string{"host", "dispatch", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review dispatch: %v", err)
	}
	history, _ = hoststate.NewStore(dir).Load("run")
	reviewDispatchDigest := history[len(history)-1].HeadAnchor

	review := hostReviewInput{hostEnvelope: hostEnvelope{RunID: "run", Actor: "fury", Timestamp: "2026-09-16T12:05:00Z"}, Review: hoststate.IndependentReview{
		UnitID: "U1", ImplementationRecordDigest: implementation, ReviewDispatchDigest: reviewDispatchDigest,
		ArtifactDigest: hoststate.ReviewArtifactDigest(head, diff, reviewedSnap.Worktree), ReviewPacketDigest: "packet", ReviewerID: "ming", Verdict: hoststate.ReviewApprove,
	}}
	writeJSON(t, input, review)
	if err := run([]string{"host", "review", "--input", input, "--store", dir}); err != nil {
		t.Fatalf("review result: %v", err)
	}

	if err := requireHostTransitionGate(dir, transition.Input{RunID: "run", ExpectedApproval: model.ApprovalBinding{
		UnitID: "U1", CurrentStage: model.StageImplementationReview, ProposedTargetStage: model.StageVerification,
		ProducedHeadDigest: head, DiffDigest: diff, WorktreeDigest: "",
	}, DirtyWorktreeApproved: true, ExpectedWorktreeDigest: reviewedSnap.Worktree}); err != nil {
		t.Fatalf("review gate: %v", err)
	}

	// The independently reviewed dirty bytes change. Keep the core and review unchanged.
	if e := os.Chmod(filepath.Join(root, "a"), afterMode); e != nil {
		t.Fatal(e)
	}
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
	if bind {
		approval.WorktreeDigest = snap.Worktree
	}
	if staleApproval {
		approval.WorktreeDigest = r4LegacyDigest(head, domain, beforeMode)
	}
	t.Logf("git_status=%q forced_mode_diff=%s", bGit(t, root, "status", "--porcelain"), bGit(t, root, "-c", "core.filemode=true", "diff", "--summary"))
	t.Logf("domain=%s modes=%#o->%#o review=%s live=%s bound=%v staleHost=%v staleApproval=%v Git=%s", domain, beforeMode, afterMode, reviewedSnap.Worktree, snap.Worktree, bind, staleHost, staleApproval, bGit(t, root, "diff", "--summary"))
	in := transition.Input{RunID: "run", RepositoryRoot: root, RepositoryIdentity: "owner/repo", BaseRevision: base, DirtyWorktreeApproved: true, ExpectedWorktreeDigest: snap.Worktree, ExpectedApproval: approval, Approval: approval, ApprovalNow: now, ApprovalMaxAgeSeconds: 300}
	if staleHost {
		in.ExpectedWorktreeDigest = reviewedSnap.Worktree
		if bind {
			in.ExpectedApproval.WorktreeDigest = reviewedSnap.Worktree
			in.Approval.WorktreeDigest = reviewedSnap.Worktree
		}
	}
	before := bStoreBytes(t, dir)
	writeJSON(t, input, in)
	err = runTransition([]string{"--store", dir, "--input", input})
	shouldPass := domain == "v3" && beforeMode&0100 == afterMode&0100 && !staleHost && !staleApproval
	if shouldPass {
		if err != nil {
			t.Fatalf("valid current artifact refused: %v", err)
		}
		rs, e := store.New(dir).Load("run")
		if e != nil || len(rs) != 3 {
			t.Fatalf("control records=%d err=%v", len(rs), e)
		}
	} else {
		if err == nil {
			t.Fatal("stale artifact advanced")
		}
		if bStoreBytes(t, dir) != before {
			t.Fatal("refusal changed store bytes")
		}
		if staleHost {
			if !strings.Contains(strings.ToLower(err.Error()), "stale") {
				t.Fatalf("expected kernel stale refusal, got %v", err)
			}
		} else if err.Error() != hoststate.ReasonReviewArtifactMismatch {
			t.Fatalf("unexpected first refusal: %v", err)
		}
		t.Logf("REFUSED %v; store unchanged", err)
	}
}
func r4LegacyDigest(head, domain string, mode os.FileMode) string {
	h := sha256.New()
	h.Write([]byte("hma-worktree-" + domain + "\x00" + head + "\x00"))
	h.Write([]byte("path\x00"))
	frame := func(b []byte) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	frame([]byte("a"))
	if domain == "v1" {
		h.Write([]byte("content\x00"))
	} else if mode&0111 != 0 {
		h.Write([]byte("executable\x00"))
	} else {
		h.Write([]byte("regular\x00"))
	}
	frame([]byte("REVIEWED-DIRTY-A"))
	return hex.EncodeToString(h.Sum(nil))
}
func TestGateBR4HiddenOwnerExecuteFullCLI(t *testing.T) {
	for _, bind := range []bool{false, true} {
		t.Run(fmt.Sprintf("bind=%v", bind), func(t *testing.T) { r4FullDirtyReview(t, 0600, 0700, bind, "v3", false, false) })
	}
}
