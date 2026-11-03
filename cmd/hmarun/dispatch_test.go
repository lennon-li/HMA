package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

// --- fake agent-dispatch binary -----------------------------------------
//
// The real agent-dispatch (scripts/agent-dispatch in the operator's shared
// obsidian repo) is a full launcher that can make a real, billed call to a
// real provider. go test must never resolve or execute it (see run.go's
// newRunner: every test here either sets stubDispatch or supplies its own
// agentDispatchBin explicitly, so the PATH lookup that would find the real
// binary installed on this machine never runs inside go test). Instead,
// every test below points hmarun at this small, parameterized fake script,
// so hmarun's own real parsing/verification logic (classifyPreflightOutput,
// stepWorkerDispatch's independent git-diff scope check, and
// stepReviewerDispatch plus the real, unchanged stepReviewImport) is
// exercised honestly, with zero network/API calls.

// fakeAgentDispatchScript is controlled entirely by environment variables
// the test sets before invoking the real dispatch code path:
//
//	FAKE_SCENARIO=preflight-ok          local preflight passes; the
//	                                    simulated real dispatch "completes".
//	FAKE_SCENARIO=preflight-ratelimited local preflight passes; the
//	                                    simulated real dispatch fails with a
//	                                    provider-style rate-limit message
//	                                    and no completion marker.
//	FAKE_SCENARIO=worker-success        commits $FAKE_ALLOWED_FILE.
//	FAKE_SCENARIO=worker-forbidden      commits $FAKE_FORBIDDEN_FILE.
//	FAKE_SCENARIO=worker-fail           exits non-zero, no commit.
//	FAKE_SCENARIO=worker-crash-once     reads/increments a counter at
//	                                    $FAKE_ATTEMPT_FILE; on the first
//	                                    invocation exits non-zero with no
//	                                    commit (simulating a crash), on the
//	                                    second behaves exactly like
//	                                    worker-success. stepWorkerDispatch's
//	                                    retry uses a fresh --run-id per
//	                                    attempt, so both invocations pass
//	                                    the run-id-exclusivity check below;
//	                                    the "second" here is about this
//	                                    counter, not about reusing a run id.
//	FAKE_SCENARIO=worker-crash-once-then-forbidden behaves exactly like
//	                                    worker-crash-once on its first
//	                                    invocation, but commits
//	                                    $FAKE_FORBIDDEN_FILE (not
//	                                    $FAKE_ALLOWED_FILE) on its second --
//	                                    used to exercise the retry ending in
//	                                    an out-of-scope commit rather than a
//	                                    clean success.
//	FAKE_SCENARIO=reviewer-success      reads review_dispatch_id,
//	                                    review_nonce, and the AUDIT.json
//	                                    output path straight out of the
//	                                    packet it receives as --prompt-file
//	                                    (exactly what stepReviewerDispatch's
//	                                    real invocation hands a real
//	                                    reviewer, and exactly what the
//	                                    packet's own Task field instructs
//	                                    it to do), writes AUDIT.json there,
//	                                    and writes receipt.json at
//	                                    $FAKE_RECEIPT_DIR/receipt.json with
//	                                    real launch digests computed the
//	                                    same way internal/repostate's
//	                                    LaunchDigests computes them --
//	                                    launch.go documents that its
//	                                    launchConfig "must stay equal to"
//	                                    agent-run-ledger's own
//	                                    compute_launch_digests, so this is
//	                                    the production algorithm, not an
//	                                    approximation.
//	FAKE_SCENARIO=reviewer-crash-once   reads/increments a counter at
//	                                    $FAKE_ATTEMPT_FILE; on the first
//	                                    invocation exits non-zero with no
//	                                    receipt and no report (simulating a
//	                                    crash); a second invocation with a
//	                                    DIFFERENT --run-id (the only way
//	                                    stepReviewPacket's reissue path
//	                                    calls this scenario a second time --
//	                                    see TestStepReviewPacketReissues-
//	                                    AfterDeadReviewDispatch) behaves
//	                                    exactly like reviewer-success. A
//	                                    second invocation with the SAME
//	                                    --run-id never reaches this
//	                                    counter logic at all: the run-id
//	                                    exclusivity check below (modeling
//	                                    agent-run-ledger's own mkdir
//	                                    exclusivity) rejects it first with
//	                                    exit 73, exactly as the real tool
//	                                    would -- audit-hmarun-step3-4-phil.md
//	                                    F3/F9.
//	FAKE_SCENARIO=reviewer-request-changes behaves exactly like
//	                                    reviewer-success except the
//	                                    AUDIT.json it writes carries verdict
//	                                    "fix-required" (maps to
//	                                    hoststate.ReviewRequestChanges), on
//	                                    its one and only invocation -- used
//	                                    to confirm a real, completed,
//	                                    non-approve verdict is never retried.
//
// packet_sha256 here is sha256 of the packet file's own on-disk bytes,
// which is what hma recorded as ReviewPacketDigest. This now matches a real
// agent-dispatch run too: an earlier version of agent-dispatch
// (obsidian/AgentSystem/scripts/agent-dispatch) hashed `$(cat "$prompt_file")`,
// a bash command substitution that strips the file's trailing newline, so
// in production the two digests would never match even with no wrapper text
// added around the packet; that integration finding was fixed upstream in
// agent-dispatch itself (sentinel-trimmed read instead of a bare command
// substitution) and verified against two real dispatches before this
// comment was written. This fake was never changed to reproduce that
// now-fixed bug -- it was never necessary, since the fake's job is to let
// everything else about the wiring (ledger path, report path, receipt
// shape, stepReviewImport's acceptance) be tested honestly on its own,
// independent of whichever agent-dispatch version happens to be installed.
const fakeAgentDispatchScript = `#!/usr/bin/env bash
set -euo pipefail
agent="$1"; shift
model=""; effort=""; run_id=""; prompt_file=""
while (($#)); do
  case "$1" in
    --model) model="$2"; shift 2;;
    --effort) effort="$2"; shift 2;;
    --run-id) run_id="$2"; shift 2;;
    --review) shift;;
    --prompt) shift 2;;
    --prompt-file) prompt_file="$2"; shift 2;;
    *) shift;;
  esac
done

scenario="${FAKE_SCENARIO:?FAKE_SCENARIO must be set}"
gitcfg=(--no-replace-objects -c core.fileMode=true -c core.ignoreCase=false -c core.fsmonitor=false -c core.untrackedCache=false -c diff.external=)

# Enforce the same run-id exclusivity the real agent-run-ledger enforces
# with its own bare "mkdir \"$run_dir\"" (no -p) before any worker launch: a
# --run-id that has been seen once, ever, in this test's
# AGENT_DISPATCH_STATE_DIR can never be reused, exactly like the real tool's
# run directory. Any hmarun code path that reuses a run id -- the bug this
# property guards against -- now fails here the same way it would fail
# against the real tool, instead of being silently masked by a fake that
# "succeeds on the 2nd call" no matter what run id it was given.
if [[ -n "$run_id" ]]; then
  seen_dir="${AGENT_DISPATCH_STATE_DIR:?AGENT_DISPATCH_STATE_DIR must be set}/fake-seen-run-ids"
  mkdir -p "$seen_dir" >/dev/null 2>&1 || true
  if ! mkdir "$seen_dir/$run_id" 2>/dev/null; then
    echo "Run id already exists: $run_id" >&2
    exit 73
  fi
fi

# counter_next increments and prints the integer counter stored at $1,
# creating it at 0 first if the file does not exist yet. Used by the
# "...-once" scenarios to behave differently on their first vs. later
# invocation, standing in for "the first attempt crashed, the retry
# succeeds".
counter_next() {
  file="$1"
  n=0
  if [[ -f "$file" ]]; then
    n="$(cat "$file")"
  fi
  n=$((n + 1))
  echo "$n" > "$file"
  echo "$n"
}

# do_reviewer_write writes a complete, valid AUDIT.json (report verdict $1,
# sole requirement status $2) and the matching agent-dispatch receipt.json,
# both bound to the packet it received as --prompt-file, exactly as
# reviewer-success's inline body used to before reviewer-crash-once and
# reviewer-request-changes needed to share it.
do_reviewer_write() {
  verdict="$1"
  reqstatus="$2"
  receipt_dir="${FAKE_RECEIPT_DIR:?}"
  mkdir -p "$receipt_dir"
  packet="$prompt_file"
  review_dispatch_id="$(sed -n 's/^review_dispatch_id: //p' "$packet" | head -1)"
  review_nonce="$(sed -n 's/^review_nonce: //p' "$packet" | head -1)"
  audit_path="$(sed -n 's/^Write your AUDIT.json to exactly this absolute path: //p' "$packet" | head -1)"
  packet_sha256="$(sha256sum "$packet" | awk '{print $1}')"
  root="$(git rev-parse --show-toplevel)"
  head="$(git rev-parse HEAD)"
  worktree_sha256="$(
    { git -C "$root" "${gitcfg[@]}" status --porcelain=v2 -z -uall --ignore-submodules=none;
      git -C "$root" "${gitcfg[@]}" diff --binary --no-ext-diff --no-textconv HEAD; } | sha256sum | awk '{print $1}'
  )"
  mkdir -p "$(dirname "$audit_path")"
  cat > "$audit_path" <<JSON
{
  "audit_id": "fake-audit",
  "target": "$agent",
  "modes": ["code"],
  "phase1_frozen": true,
  "limitations": [],
  "verdict": "$verdict",
  "isolation": "isolated",
  "findings": [],
  "requirements": [{"id": "fake-requirement", "status": "$reqstatus", "evidence_refs": ["fake-review"]}],
  "hma_binding": {"packet_sha256": "$packet_sha256", "review_dispatch_id": "$review_dispatch_id", "review_nonce": "$review_nonce"}
}
JSON
  report_sha256="$(sha256sum "$audit_path" | awk '{print $1}')"
  cat > "$receipt_dir/receipt.json" <<JSON
{
  "run_id": "$run_id",
  "agent": "$agent",
  "model": "$model",
  "provider": "openai",
  "effort": "$effort",
  "started_at": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "project_root": "$root",
  "mode": "review",
  "state": "completed",
  "exit_code": 0,
  "packet_sha256": "$packet_sha256",
  "launch_head": "$head",
  "launch_worktree_sha256": "$worktree_sha256",
  "report_files": [{"path": "$audit_path", "sha256": "$report_sha256"}]
}
JSON
  echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
  echo "AGENT_DISPATCH_RECEIPT=$receipt_dir/receipt.json" >&2
  echo "AGENT_DISPATCH_STATE=completed" >&2
}

case "$scenario" in
  preflight-ok)
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "fake hi response"
    echo "AGENT_DISPATCH_STATE=completed" >&2
    exit 0
    ;;
  preflight-ratelimited)
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "simulated provider error: rate limit exceeded (429)" >&2
    exit 1
    ;;
  worker-success)
    f="${FAKE_ALLOWED_FILE:?}"
    echo "fake worker change" >> "$f"
    git add "$f"
    git -c user.name=fake-ming -c user.email=fake-ming@example.invalid commit -qm "fake worker commit"
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "AGENT_DISPATCH_STATE=completed" >&2
    exit 0
    ;;
  worker-forbidden)
    f="${FAKE_FORBIDDEN_FILE:?}"
    mkdir -p "$(dirname "$f")"
    echo "fake forbidden change" >> "$f"
    git add "$f"
    git -c user.name=fake-ming -c user.email=fake-ming@example.invalid commit -qm "fake worker commit touching a forbidden file"
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "AGENT_DISPATCH_STATE=completed" >&2
    exit 0
    ;;
  worker-fail)
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "simulated worker crash: no commit produced" >&2
    exit 1
    ;;
  worker-crash-once)
    n="$(counter_next "${FAKE_ATTEMPT_FILE:?}")"
    if [[ "$n" -eq 1 ]]; then
      echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
      echo "simulated worker crash: no commit produced (attempt $n)" >&2
      exit 1
    fi
    f="${FAKE_ALLOWED_FILE:?}"
    echo "fake worker change (attempt $n)" >> "$f"
    git add "$f"
    git -c user.name=fake-ming -c user.email=fake-ming@example.invalid commit -qm "fake worker commit (attempt $n)"
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "AGENT_DISPATCH_STATE=completed" >&2
    exit 0
    ;;
  worker-crash-once-then-forbidden)
    n="$(counter_next "${FAKE_ATTEMPT_FILE:?}")"
    if [[ "$n" -eq 1 ]]; then
      echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
      echo "simulated worker crash: no commit produced (attempt $n)" >&2
      exit 1
    fi
    f="${FAKE_FORBIDDEN_FILE:?}"
    mkdir -p "$(dirname "$f")"
    echo "fake forbidden change (attempt $n)" >> "$f"
    git add "$f"
    git -c user.name=fake-ming -c user.email=fake-ming@example.invalid commit -qm "fake worker commit touching a forbidden file (attempt $n)"
    echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
    echo "AGENT_DISPATCH_STATE=completed" >&2
    exit 0
    ;;
  reviewer-success)
    do_reviewer_write "pass" "pass"
    exit 0
    ;;
  reviewer-crash-once)
    n="$(counter_next "${FAKE_ATTEMPT_FILE:?}")"
    if [[ "$n" -eq 1 ]]; then
      echo "AGENT_DISPATCH_PREFLIGHT=ok agent=$agent runtime=fake" >&2
      echo "simulated reviewer crash: no receipt or report produced (attempt $n)" >&2
      exit 1
    fi
    do_reviewer_write "pass" "pass"
    exit 0
    ;;
  reviewer-request-changes)
    do_reviewer_write "fix-required" "fail"
    exit 0
    ;;
  *)
    echo "unknown FAKE_SCENARIO: $scenario" >&2
    exit 2
    ;;
esac
`

func writeFakeAgentDispatch(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent-dispatch")
	if err := os.WriteFile(path, []byte(fakeAgentDispatchScript), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// newRealDispatchRunner builds a *runner configured for real dispatch
// against the fake binary (not --stub-dispatch), against a fresh throwaway
// repository and store.
func newRealDispatchRunner(t *testing.T, fakeBin string) *runner {
	t.Helper()
	hmaBin := buildHMABinary(t)
	repo := hmarunTestRepo(t)
	storeDir := t.TempDir()
	docsDir := t.TempDir()
	contractPath, planPath := hmarunTestDocs(t, docsDir)
	cfg := runConfig{
		storeDir: storeDir, repoRoot: repo, runID: "run", unit: "U1",
		contractPath: contractPath, planPath: planPath, hmaBin: hmaBin,
		agentDispatchBin: fakeBin, autoApprove: true,
	}
	r, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// runSteps runs each step in order and fails the test immediately on the
// first error.
func runSteps(t *testing.T, ctx context.Context, steps ...func(context.Context) error) {
	t.Helper()
	for _, step := range steps {
		if err := step(ctx); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
}

// throughImplDispatch is the fixed prefix every test below needs before it
// can exercise the worker or reviewer step: every core stage transition up
// to and including the (host-record-only, no agent-dispatch call) real
// implementation dispatch record.
func throughImplDispatch(r *runner) []func(context.Context) error {
	return []func(context.Context) error{
		r.stepAcceptanceCriteria, r.stepPlanning, r.stepRouteSelection, r.stepRouteApproval,
		r.stepDecision, r.stepImplPreflight, r.stepImplDispatch,
	}
}

// --- pure unit tests ------------------------------------------------------

func TestRepoKeyForMatchesAgentDispatchLedgerHelper(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available to cross-check against agent-run-ledger's own repo_key_for")
	}
	dir := t.TempDir()
	got, err := repoKeyFor(dir)
	if err != nil {
		t.Fatal(err)
	}
	// This is scripts/agent-run-ledger's repo_key_for, verbatim, run exactly
	// as that script runs it, so this test fails the moment the two
	// implementations drift apart.
	out, err := exec.Command("python3", "-c", `
import hashlib, os, sys
root = os.path.realpath(sys.argv[1])
name = os.path.basename(root).replace("/", "-") or "repo"
digest = hashlib.sha256(root.encode("utf-8")).hexdigest()[:16]
print(f"{name}-{digest}")
`, dir).CombinedOutput()
	if err != nil {
		t.Fatalf("python3 cross-check: %v: %s", err, out)
	}
	want := strings.TrimSpace(string(out))
	if got != want {
		t.Fatalf("repoKeyFor(%q) = %q, want %q (agent-dispatch's own repo_key_for)", dir, got, want)
	}
}

func TestClassifyPreflightOutput(t *testing.T) {
	cases := []struct {
		name   string
		output string
		exit   int
		want   hoststate.PreflightStatus
	}{
		{"clean success", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nhi\nAGENT_DISPATCH_STATE=completed\n", 0, hoststate.PreflightAvailable},
		{"local preflight failure", "AGENT_DISPATCH_PREFLIGHT=failed reason=missing-command command=claude agent=ming\n", 69, hoststate.PreflightUnavailable},
		{"rate limited after local ok", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nError: rate limit exceeded (429)\n", 1, hoststate.PreflightRateLimited},
		{"quota exhausted after local ok", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nerror: insufficient_quota\n", 1, hoststate.PreflightQuotaExhausted},
		{"unexplained failure after local ok", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nsomething else broke\n", 1, hoststate.PreflightIndeterminate},
		// audit-hmarun-step3-4-phil.md F7: a hex run id or digest that
		// happens to contain the digits "429" embedded in a longer token
		// must not be misread as an HTTP 429 rate-limit signal.
		{"hex token containing 429 is not a rate-limit signal", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nAGENT_DISPATCH_RUN_ID=a1b429c2deadbeef\nsomething else broke\n", 1, hoststate.PreflightIndeterminate},
		{"exit 0 but no usable result", "AGENT_DISPATCH_PREFLIGHT=ok agent=ming runtime=claude\nAGENT_DISPATCH_STATE=completed_without_result\n", 0, hoststate.PreflightIndeterminate},
		{"no recognizable preflight line", "usage: agent-dispatch ...\n", 2, hoststate.PreflightIndeterminate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, digest := classifyPreflightOutput(c.output, c.exit)
			if got != c.want {
				t.Errorf("classifyPreflightOutput(...) status = %s, want %s", got, c.want)
			}
			if digest != sha256Hex(c.output) {
				t.Errorf("classifyPreflightOutput(...) digest is not sha256 of the exact output bytes")
			}
		})
	}
}

// --- fake-binary scenario tests --------------------------------------------

// TestStepImplPreflightCleanSuccess is scenario (1): a clean preflight
// success writes an AVAILABLE host record with a real response_digest.
func TestStepImplPreflightCleanSuccess(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, r.stepAcceptanceCriteria, r.stepPlanning, r.stepRouteSelection, r.stepRouteApproval, r.stepDecision)

	if err := r.stepImplPreflight(ctx); err != nil {
		t.Fatalf("stepImplPreflight: %v", err)
	}
	history, err := hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, rec := range history {
		if rec.Kind == hoststate.KindPreflight && rec.Preflight != nil && rec.Preflight.Role == hoststate.RoleImplementation {
			found = true
			if rec.Preflight.Status != hoststate.PreflightAvailable {
				t.Errorf("preflight status = %s, want AVAILABLE", rec.Preflight.Status)
			}
			if rec.Preflight.ResponseDigest == "" || rec.Preflight.ResponseDigest == stubPreflightResponseDigest {
				t.Errorf("response_digest = %q, want a real digest of the actual agent-dispatch output", rec.Preflight.ResponseDigest)
			}
		}
	}
	if !found {
		t.Fatal("no implementation preflight record was written")
	}
}

// TestStepImplPreflightRateLimitedStopsCleanly is scenario (2): a
// rate-limited preflight must stop `hmarun run` cleanly, must not reach
// dispatch, and must record the real observed status, never a silently
// promoted AVAILABLE.
func TestStepImplPreflightRateLimitedStopsCleanly(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ratelimited")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, r.stepAcceptanceCriteria, r.stepPlanning, r.stepRouteSelection, r.stepRouteApproval, r.stepDecision)

	err := r.stepImplPreflight(ctx)
	if err == nil {
		t.Fatal("stepImplPreflight: want an error for a RATE_LIMITED preflight, got nil")
	}
	if !strings.Contains(err.Error(), "RATE_LIMITED") || !strings.Contains(err.Error(), "stops here") {
		t.Fatalf("error does not clearly name the stop reason (no dead-end agents contract): %v", err)
	}

	history, err := hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	var preflights, dispatches int
	for _, rec := range history {
		if rec.Kind == hoststate.KindPreflight && rec.Preflight != nil && rec.Preflight.Role == hoststate.RoleImplementation {
			preflights++
			if rec.Preflight.Status != hoststate.PreflightRateLimited {
				t.Errorf("recorded preflight status = %s, want RATE_LIMITED: the real status, never silently promoted", rec.Preflight.Status)
			}
		}
		if rec.Kind == hoststate.KindDispatch {
			dispatches++
		}
	}
	if preflights != 1 {
		t.Fatalf("implementation preflight records = %d, want exactly 1", preflights)
	}
	if dispatches != 0 {
		t.Fatalf("dispatch records = %d, want 0: a non-AVAILABLE preflight must never let hmarun proceed to dispatch", dispatches)
	}
}

// TestStepWorkerDispatchSuccess is scenario (3): a worker dispatch whose
// fake output includes a real commit inside the repository confirms hmarun
// reads the produced head and proceeds.
func TestStepWorkerDispatchSuccess(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	headBefore, err := r.repoHead(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FAKE_SCENARIO", "worker-success")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}

	headAfter, err := r.repoHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if headAfter == headBefore {
		t.Fatal("stepWorkerDispatch did not move HEAD")
	}
	out, err := exec.Command("git", "-C", r.cfg.repoRoot, "diff", "--name-only", headBefore, headAfter).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "a" {
		t.Fatalf("changed files = %q, want %q", got, "a")
	}
}

// TestStepWorkerDispatchForbiddenFileStopsImmediately is scenario (4): a
// worker dispatch that touches a forbidden file must be caught by hmarun
// itself from the real git diff, not by trusting the worker, and must stop
// with a clear, specific message -- never retried automatically.
func TestStepWorkerDispatchForbiddenFileStopsImmediately(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	headBefore, err := r.repoHead(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FAKE_SCENARIO", "worker-forbidden")
	t.Setenv("FAKE_FORBIDDEN_FILE", filepath.Join(r.cfg.repoRoot, "docs"))
	err = r.stepWorkerDispatch(ctx)
	if err == nil {
		t.Fatal("stepWorkerDispatch: want an error when the worker commits a forbidden file, got nil")
	}
	if !strings.Contains(err.Error(), "outside unit") || !strings.Contains(err.Error(), "docs") {
		t.Fatalf("error does not clearly name the scope violation: %v", err)
	}
	headAfter, err := r.repoHead(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if headAfter == headBefore {
		t.Fatal("expected the forbidden commit to still be present (hmarun does not auto-revert); HEAD did not move at all")
	}
}

// TestStepReviewerDispatchAcceptedByReviewImport is scenario (5): a
// reviewer dispatch whose fake AUDIT.json/receipt are then actually
// accepted by the real, unchanged stepReviewImport/EvaluateReviewImport
// path, end to end.
func TestStepReviewerDispatchAcceptedByReviewImport(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-success")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}
	if err := r.stepImplementationReview(ctx); err != nil {
		t.Fatalf("stepImplementationReview: %v", err)
	}
	if err := r.stepBindRepository(ctx); err != nil {
		t.Fatalf("stepBindRepository: %v", err)
	}

	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	if err := r.stepReviewPreflight(ctx); err != nil {
		t.Fatalf("stepReviewPreflight: %v", err)
	}
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket: %v", err)
	}

	ledgerDir, err := r.ledgerDir()
	if err != nil {
		t.Fatal(err)
	}
	receiptDir := filepath.Join(ledgerDir, "runs", r.reviewDispatchRunID())
	t.Setenv("FAKE_SCENARIO", "reviewer-success")
	t.Setenv("FAKE_RECEIPT_DIR", receiptDir)
	if err := r.stepReviewerDispatch(ctx); err != nil {
		t.Fatalf("stepReviewerDispatch: %v", err)
	}

	if err := r.stepReviewImport(ctx); err != nil {
		t.Fatalf("stepReviewImport did not accept the real reviewer dispatch's receipt/report: %v", err)
	}

	history, err := hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit {
			found = true
			if rec.Review.Verdict != hoststate.ReviewApprove {
				t.Errorf("imported review verdict = %s, want APPROVE", rec.Review.Verdict)
			}
			if rec.Review.ReceiptDigest == "" || rec.Review.ReportDigest == "" {
				t.Error("imported review is missing receipt_digest/report_digest: it was not really imported")
			}
		}
	}
	if !found {
		t.Fatal("no independent review record was written")
	}

	// Regression for the real dogfood run's HOST_UNAUTHORIZED_DISPATCH
	// finding (see the dispatch report): agentDispatchStateDir() is the one
	// directory the reviewer's packet declares as a reconciled namespace
	// (ledgerDir derives from it). hma host reconcile's rule is "every run
	// physically present in a declared namespace must match a reservation
	// or packet", so every preflight and real implementer dispatch above
	// must land somewhere else -- unreconciledDispatchStateDir() -- or
	// reconcile would flag them as unauthorized launches and block
	// stepVerification regardless of how clean the implementation and
	// review actually were. The fake binary's run-id exclusivity check
	// (above in this file) marks each run id it sees under
	// "$AGENT_DISPATCH_STATE_DIR/fake-seen-run-ids/<run-id>", so those
	// marker directories show, directly, which AGENT_DISPATCH_STATE_DIR
	// each real call actually received -- not just that the two directory
	// values differ as constants, which would be true even if a call site
	// still passed the wrong one.
	seenIn := func(dir string) []string {
		entries, err := os.ReadDir(filepath.Join(dir, "fake-seen-run-ids"))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			t.Fatal(err)
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return names
	}
	unreconciledSeen := seenIn(r.unreconciledDispatchStateDir())
	reconciledSeen := seenIn(r.agentDispatchStateDir())
	// Two IMPLEMENTATION preflights (throughImplDispatch issues one;
	// TestStepReviewerDispatchAcceptedByReviewImport's own setup does not
	// issue a second, so exactly one) plus the worker dispatch, plus the
	// REVIEW preflight: all four real non-reviewer calls.
	if len(unreconciledSeen) != 3 {
		t.Fatalf("unreconciledDispatchStateDir() saw %d run id(s), want 3 (impl preflight, worker dispatch, review preflight): %v",
			len(unreconciledSeen), unreconciledSeen)
	}
	if len(reconciledSeen) != 1 {
		t.Fatalf("agentDispatchStateDir() saw %d run id(s), want exactly 1 (the real reviewer dispatch): %v",
			len(reconciledSeen), reconciledSeen)
	}
	if reconciledSeen[0] != r.reviewDispatchRunID() {
		t.Fatalf("agentDispatchStateDir() saw run id %q, want the reviewer's own %q", reconciledSeen[0], r.reviewDispatchRunID())
	}
	for _, id := range unreconciledSeen {
		if id == r.reviewDispatchRunID() {
			t.Fatalf("the reviewer's run id %q leaked into unreconciledDispatchStateDir()", id)
		}
	}
}

// TestStepReviewerDispatchCrashEscalatesWithoutRetry replaces an earlier
// version of stepReviewerDispatch's behavior that retried a crashed
// reviewer dispatch with the same --run-id. That retry cannot work against
// the real agent-dispatch/agent-run-ledger tools: agent-run-ledger creates
// the run directory with a bare `mkdir` (no -p) before any worker launch,
// so a run id that has already reached that point -- true of essentially
// every real crash, since the crash represents the worker failing after
// the ledger accepted the run -- can never be reused, and the review
// packet's own DispatchRunID is baked into its hashed bytes, so a fresh
// attempt needs a fresh packet, not a retried flag. This test uses the
// "...-once" fake, which would silently succeed on a second invocation (so
// it is exactly the fake that would have masked the old bug): it proves
// stepReviewerDispatch calls agent-dispatch exactly once, returns an
// actionable error naming why a retry can't work, and records the failure
// as unrecoverable rather than as a resolved recovery.
func TestStepReviewerDispatchCrashEscalatesWithoutRetry(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-success")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}
	if err := r.stepImplementationReview(ctx); err != nil {
		t.Fatalf("stepImplementationReview: %v", err)
	}
	if err := r.stepBindRepository(ctx); err != nil {
		t.Fatalf("stepBindRepository: %v", err)
	}

	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	if err := r.stepReviewPreflight(ctx); err != nil {
		t.Fatalf("stepReviewPreflight: %v", err)
	}
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket: %v", err)
	}

	ledgerDir, err := r.ledgerDir()
	if err != nil {
		t.Fatal(err)
	}
	receiptDir := filepath.Join(ledgerDir, "runs", r.reviewDispatchRunID())
	attemptFile := filepath.Join(t.TempDir(), "attempts")
	t.Setenv("FAKE_SCENARIO", "reviewer-crash-once")
	t.Setenv("FAKE_RECEIPT_DIR", receiptDir)
	t.Setenv("FAKE_ATTEMPT_FILE", attemptFile)
	err = r.stepReviewerDispatch(ctx)
	if err == nil {
		t.Fatal("stepReviewerDispatch: want an error from a crashed reviewer dispatch, got nil")
	}
	for _, want := range []string{"cannot be retried", "same run id", "mint a fresh packet", binding(r).DispatchRunID} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}

	if r.summary.recoveryAttempts != 0 || r.summary.recoverySucceeded != 0 {
		t.Errorf("summary recovery counts = attempts=%d succeeded=%d, want 0 and 0: this is not a retry", r.summary.recoveryAttempts, r.summary.recoverySucceeded)
	}
	if r.summary.blockedRecoverable != 0 || r.summary.blockedUnrecoverable != 1 {
		t.Errorf("summary blocks = recoverable=%d unrecoverable=%d, want 0 and 1", r.summary.blockedRecoverable, r.summary.blockedUnrecoverable)
	}

	attempts, err := os.ReadFile(attemptFile)
	if err != nil {
		t.Fatalf("reading attempt counter: %v", err)
	}
	if got := strings.TrimSpace(string(attempts)); got != "1" {
		t.Errorf("fake agent-dispatch was invoked %s times, want exactly 1 (no same-run-id retry)", got)
	}
}

// binding returns the review dispatch's packet binding, for tests that need
// to assert on its fields (e.g. the error message naming the run id).
func binding(r *runner) *hoststate.ReviewPacketBinding {
	history, err := r.hostHistory()
	if err != nil {
		panic(err)
	}
	implementationDigest, err := hoststate.RequireImplementation(history, r.cfg.unit)
	if err != nil {
		panic(err)
	}
	dispatch := findReviewDispatch(history, r.cfg.unit, implementationDigest)
	if dispatch == nil {
		panic("no review dispatch found")
	}
	return dispatch.Dispatch.Packet
}

// TestStepReviewPacketReissuesAfterDeadReviewDispatch is the fix for the
// finding TestStepReviewerDispatchCrashEscalatesWithoutRetry's error message
// promises a remedy for: that a dead review dispatch (crashed, never
// imported) must not wedge a resumed run forever. Before this fix,
// stepReviewPacket's own idempotency check ("a review dispatch already
// exists for this unit, skip") meant a resumed `hmarun run` kept finding the
// same dead dispatch record and resuming the same permanently-consumed run
// id, so the escalation error's own suggested fix ("re-run `hmarun run`")
// was false -- a real dead end, exactly what this project's "no dead-end
// agents" requirement forbids. This test proves the promise is now true:
// after a crash, calling stepReviewPacket again (as a resumed run's step
// walk would) mints a genuinely new packet at a new run id, and a
// subsequent reviewer dispatch at that new id succeeds and imports cleanly.
func TestStepReviewPacketReissuesAfterDeadReviewDispatch(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-success")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}
	if err := r.stepImplementationReview(ctx); err != nil {
		t.Fatalf("stepImplementationReview: %v", err)
	}
	if err := r.stepBindRepository(ctx); err != nil {
		t.Fatalf("stepBindRepository: %v", err)
	}

	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	if err := r.stepReviewPreflight(ctx); err != nil {
		t.Fatalf("stepReviewPreflight: %v", err)
	}
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket (first issuance): %v", err)
	}
	firstRunID := binding(r).DispatchRunID
	if firstRunID != r.reviewDispatchRunID() {
		t.Fatalf("first issuance run id = %q, want the deterministic %q", firstRunID, r.reviewDispatchRunID())
	}

	ledgerDir, err := r.ledgerDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_SCENARIO", "reviewer-crash-once")
	t.Setenv("FAKE_RECEIPT_DIR", filepath.Join(ledgerDir, "runs", firstRunID))
	t.Setenv("FAKE_ATTEMPT_FILE", filepath.Join(t.TempDir(), "attempts"))
	if err := r.stepReviewerDispatch(ctx); err == nil {
		t.Fatal("stepReviewerDispatch: want an error from the crashed first attempt")
	}

	// A resumed run's step walk calls stepReviewPacket again before
	// stepReviewerDispatch; this is that call.
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket (reissue): %v", err)
	}
	secondRunID := binding(r).DispatchRunID
	if secondRunID == firstRunID {
		t.Fatalf("stepReviewPacket did not reissue: run id is still %q", secondRunID)
	}

	// The new run id has never been used, so a plain success (not
	// "...-once") is realistic here -- nothing has crashed at this id.
	t.Setenv("FAKE_SCENARIO", "reviewer-success")
	t.Setenv("FAKE_RECEIPT_DIR", filepath.Join(ledgerDir, "runs", secondRunID))
	if err := r.stepReviewerDispatch(ctx); err != nil {
		t.Fatalf("stepReviewerDispatch (after reissue): %v", err)
	}
	if err := r.stepReviewImport(ctx); err != nil {
		t.Fatalf("stepReviewImport did not accept the reissued dispatch's receipt/report: %v", err)
	}

	history, err := hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit {
			found = true
			if rec.Review.Verdict != hoststate.ReviewApprove {
				t.Errorf("imported review verdict = %s, want APPROVE", rec.Review.Verdict)
			}
		}
	}
	if !found {
		t.Fatal("no independent review record was written")
	}

	// A further stepReviewPacket call (as a later resume would make) must
	// now be a no-op: the dispatch is imported, so it must never reissue
	// again.
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket (post-import): %v", err)
	}
	if got := binding(r).DispatchRunID; got != secondRunID {
		t.Errorf("stepReviewPacket reissued again after import: run id = %q, want unchanged %q", got, secondRunID)
	}
}

// TestStepReviewerDispatchNeverRetriesRealVerdict is Part A's second
// required test: a reviewer dispatch whose fake agent-dispatch binary
// produces a complete receipt and report carrying a real, non-approve
// verdict (REQUEST_CHANGES) on its one and only invocation must never be
// retried -- stepReviewerDispatch hands that result straight to
// stepReviewImport, which must surface it as-is.
func TestStepReviewerDispatchNeverRetriesRealVerdict(t *testing.T) {
	fakeBin := writeFakeAgentDispatch(t)
	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	r := newRealDispatchRunner(t, fakeBin)
	ctx := context.Background()
	runSteps(t, ctx, throughImplDispatch(r)...)

	t.Setenv("FAKE_SCENARIO", "worker-success")
	t.Setenv("FAKE_ALLOWED_FILE", filepath.Join(r.cfg.repoRoot, "a"))
	if err := r.stepWorkerDispatch(ctx); err != nil {
		t.Fatalf("stepWorkerDispatch: %v", err)
	}
	if err := r.stepImplementationReview(ctx); err != nil {
		t.Fatalf("stepImplementationReview: %v", err)
	}
	if err := r.stepBindRepository(ctx); err != nil {
		t.Fatalf("stepBindRepository: %v", err)
	}

	t.Setenv("FAKE_SCENARIO", "preflight-ok")
	if err := r.stepReviewPreflight(ctx); err != nil {
		t.Fatalf("stepReviewPreflight: %v", err)
	}
	if err := r.stepReviewPacket(ctx); err != nil {
		t.Fatalf("stepReviewPacket: %v", err)
	}

	ledgerDir, err := r.ledgerDir()
	if err != nil {
		t.Fatal(err)
	}
	receiptDir := filepath.Join(ledgerDir, "runs", r.reviewDispatchRunID())
	t.Setenv("FAKE_SCENARIO", "reviewer-request-changes")
	t.Setenv("FAKE_RECEIPT_DIR", receiptDir)
	if err := r.stepReviewerDispatch(ctx); err != nil {
		t.Fatalf("stepReviewerDispatch: %v (a complete dispatch with a real verdict must not be treated as a failure)", err)
	}

	if r.summary.recoveryAttempts != 0 || r.summary.blockedRecoverable != 0 || r.summary.blockedUnrecoverable != 0 {
		t.Errorf("summary = %+v, want no blocks or retries: a real completed verdict is not a crash", r.summary)
	}

	if err := r.stepReviewImport(ctx); err != nil {
		t.Fatalf("stepReviewImport did not accept the real, non-approve verdict: %v", err)
	}

	history, err := hoststate.NewStore(r.cfg.storeDir).Load(r.cfg.runID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, rec := range history {
		if rec.Kind == hoststate.KindReview && rec.Review != nil && rec.Review.UnitID == r.cfg.unit {
			found = true
			if rec.Review.Verdict != hoststate.ReviewRequestChanges {
				t.Errorf("imported review verdict = %s, want REQUEST_CHANGES (a real verdict, never retried)", rec.Review.Verdict)
			}
		}
	}
	if !found {
		t.Fatal("no independent review record was written")
	}
}
