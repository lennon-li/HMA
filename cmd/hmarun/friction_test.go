package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

// TestRunSummaryCleanRun confirms a clean, --stub-dispatch run to terminal
// PARTIAL records a nonzero number of approval requests and exactly zero
// blocked operations of either kind.
func TestRunSummaryCleanRun(t *testing.T) {
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)

	cfg := runConfig{
		storeDir: storeDir, repoRoot: repo, runID: "run", unit: "U1",
		contractPath: contractPath, planPath: planPath, hmaBin: hmaBin,
		autoApprove: true, stubDispatch: true,
	}
	r, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("hmarun run: %v", err)
	}

	if r.summary.approvals == 0 {
		t.Error("summary.approvals = 0, want at least one approval request over a full run")
	}
	if r.summary.blockedRecoverable != 0 || r.summary.blockedUnrecoverable != 0 {
		t.Errorf("summary = %+v, want zero blocked operations for a clean run", r.summary)
	}
	if r.summary.recoveryAttempts != 0 || r.summary.recoverySucceeded != 0 {
		t.Errorf("summary = %+v, want zero recovery attempts for a clean run", r.summary)
	}
	if r.summary.outcome != "PARTIAL" {
		t.Errorf("summary.outcome = %q, want PARTIAL", r.summary.outcome)
	}
}

// TestRunSummaryWorkerRetryIsOneRecoverableBlock confirms that a worker
// dispatch whose first attempt produces no commit, then succeeds on its one
// automatic retry, is counted as exactly one recoverable block and one
// successful recovery attempt -- never an unrecoverable one.
func TestRunSummaryWorkerRetryIsOneRecoverableBlock(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-crash-once")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	t.Setenv("FAKE_ATTEMPT_FILE", filepath.Join(t.TempDir(), "attempts"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}

	if r.summary.blockedRecoverable != 1 {
		t.Errorf("summary.blockedRecoverable = %d, want 1", r.summary.blockedRecoverable)
	}
	if r.summary.blockedUnrecoverable != 0 {
		t.Errorf("summary.blockedUnrecoverable = %d, want 0", r.summary.blockedUnrecoverable)
	}
	if r.summary.recoveryAttempts != 1 || r.summary.recoverySucceeded != 1 {
		t.Errorf("summary recovery counts = attempts=%d succeeded=%d, want 1 and 1", r.summary.recoveryAttempts, r.summary.recoverySucceeded)
	}
}

// TestRunSummaryForbiddenFileIsOneUnrecoverableBlock confirms that a worker
// dispatch which commits a forbidden file is counted as exactly one
// unrecoverable block, with no automatic recovery attempted, and that the
// step itself still fails (no completion).
func TestRunSummaryForbiddenFileIsOneUnrecoverableBlock(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-forbidden")
	t.Setenv("FAKE_FORBIDDEN_FILE", filepath.Join(r.cfg.repoRoot, "docs"))
	err := r.stepWorkerDispatch(ctx)
	if err == nil {
		t.Fatal("stepWorkerDispatch: want an error for a forbidden-file commit (no completion), got nil")
	}

	if r.summary.blockedUnrecoverable != 1 {
		t.Errorf("summary.blockedUnrecoverable = %d, want 1", r.summary.blockedUnrecoverable)
	}
	if r.summary.blockedRecoverable != 0 {
		t.Errorf("summary.blockedRecoverable = %d, want 0", r.summary.blockedRecoverable)
	}
	if r.summary.recoveryAttempts != 0 {
		t.Errorf("summary.recoveryAttempts = %d, want 0 (forbidden-file hard stop is never retried)", r.summary.recoveryAttempts)
	}
}

// TestRunSummaryRetryEndingOutOfScopeStillCountsAsAnAttempt is an
// independent-review finding (audit-hmarun-step3-4-phil.md, F8.1): an
// earlier version only credited recoveryAttempts when the one bounded
// worker retry ended in success or in final no-commit exhaustion, so a
// retry that actually ran but committed an out-of-scope file (or exited
// non-zero, or failed to launch) was blocked but silently never counted as
// an attempted recovery. This confirms the retry is now counted exactly
// once no matter which of those outcomes it hits.
func TestRunSummaryRetryEndingOutOfScopeStillCountsAsAnAttempt(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-crash-once-then-forbidden")
	t.Setenv("FAKE_FORBIDDEN_FILE", filepath.Join(r.cfg.repoRoot, "docs"))
	t.Setenv("FAKE_ATTEMPT_FILE", filepath.Join(t.TempDir(), "attempts"))
	err := r.stepWorkerDispatch(ctx)
	if err == nil {
		t.Fatal("stepWorkerDispatch: want an error (the retry committed an out-of-scope file), got nil")
	}

	if r.summary.recoveryAttempts != 1 {
		t.Errorf("summary.recoveryAttempts = %d, want 1: the retry did happen, even though it ended out-of-scope", r.summary.recoveryAttempts)
	}
	if r.summary.recoverySucceeded != 0 {
		t.Errorf("summary.recoverySucceeded = %d, want 0: an out-of-scope commit is not a successful recovery", r.summary.recoverySucceeded)
	}
	if r.summary.blockedUnrecoverable != 1 {
		t.Errorf("summary.blockedUnrecoverable = %d, want 1", r.summary.blockedUnrecoverable)
	}
	if r.summary.blockedRecoverable != 0 {
		t.Errorf("summary.blockedRecoverable = %d, want 0: this block was never actually resolved", r.summary.blockedRecoverable)
	}
}
