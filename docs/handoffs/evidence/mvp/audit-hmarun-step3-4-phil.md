# Independent audit: `cmd/hmarun` Steps 3-4 (Phil, 2026-11-03)

**Verdict: PASS WITH FINDINGS (static review only; two required checks could not be run).**

Two checks in the request were **not performed**. This is a failure of my environment, not a pass.

- Check 3 (read `agent-run-ledger` myself): not done. Reads outside the repo were denied.
- Check 8 (build/vet/test): not done. `go` is not on PATH (`command -v go` returned nothing), and probing other locations was denied.

The run-id-reuse fix is therefore **not independently confirmed** against the real tool. See F1.

Nothing was modified under `cmd/hmarun/`. `git status --short` before and after is identical:
```
?? cmd/hmarun/
?? docs/handoffs/evidence/mvp/
```
The only file I added is this report, inside the already-untracked `docs/handoffs/evidence/mvp/`.

## Verification commands

| Command | Result |
|---|---|
| `git status --short` (before/after) | Identical, as above. |
| `command -v go` | No output (go not installed on PATH). |
| `go build ./...`, `go vet ./...` | **NOT RUN** (no `go`). |
| `go test ./cmd/hmarun/... -v -race` | **NOT RUN** (no `go`). |
| `go test ./...` | **NOT RUN** (no `go`). |
| Read `/home/yeli/obsidian/AgentSystem/scripts/agent-run-ledger` | **DENIED** by the tool permission layer. |

Do not treat any claim below as test-verified. All findings come from reading `run.go`, `summary.go`, and `dispatch_test.go`.

## Method

I read these in full: `issuePreflight`, `findLatestValidPreflight`, `classifyPreflightOutput`, `lastFieldValue`, `freshDispatchRunID`, `runAgentDispatch`, `stepWorkerDispatch`, `stepReviewPacket`, `stepReviewerDispatch`, `reviewerDispatchOutputUsable`, `stepReviewImport`, `run()`, `confirm()`, `summary.go`, and the fake script and the two reviewer tests in `dispatch_test.go`. I grepped every `r.summary.` site.

## Findings

### F1 (MEDIUM, unverified claim) The run-id-reuse reasoning is plausible, but I did not confirm it

Observed: the code comments (`run.go` ~1664-1690 and `freshDispatchRunID`) state that `agent-run-ledger` runs a bare `mkdir "$run_dir"`. I could not open that file. The in-repo evidence is only the author's own comments.

Interpretation: the reasoning is internally consistent. A fixed `--run-id` that the ledger already accepted cannot be reused if `mkdir` has no `-p`. I cannot say it is correct against the real tool.

Required action: someone with access must run `grep -n 'mkdir\|already exists' agent-run-ledger` around its `run()` function and attach the lines.

### F2 (HIGH) The escalation remedy is a dead end inside `hmarun`, and a resume re-fails

Evidence:
- `reviewDispatchRunID()` (`run.go:1465`) is deterministic: `hmarun-review-<run>-<unit>`.
- `stepReviewPacket` (`run.go:1511`) returns early when `findReviewDispatch(...) != nil`. It never mints a new packet for the same implementation digest.
- `stepReviewerDispatch` takes the run id from the recorded binding. After any non-usable outcome, including a plain Ctrl-C or kill mid-dispatch, a re-run of `hmarun run` calls `agent-dispatch` with the same run id again.
- If F1's premise holds, that second call fails at once with "Run id already exists". The step re-escalates with the same error.

The error text says to re-issue a packet via `hma host review-packet` and re-run `hmarun run`. But `review-packet-input.json` carries the same deterministic `DispatchRunID`. A human must hand-edit that file. The message does not say so.

Impact: a single reviewer crash or interruption permanently wedges the run in `hmarun`. This is the "no dead-end agents" contract failure, but now documented rather than silent.

Recommendation (not applied): derive the review run id from a per-packet-attempt counter or nonce, and let `stepReviewPacket` re-issue when the latest dispatch has a non-usable ledger run. Alternatively, state in the message exactly which field to change.

### F3 (MEDIUM) The fake harness does not enforce the property the fix depends on

Observed: the fake script never creates or checks a run directory. It does not fail on a repeated `--run-id`. `reviewer-crash-once` "succeeds on the 2nd call" exactly like the old fake that hid the bug. The header comment (lines ~68-80) still documents that semantic.

`TestStepReviewerDispatchCrashEscalatesWithoutRetry` asserts the fake was called once. That protects the no-retry behavior. It cannot detect a reintroduced same-id reuse elsewhere, for example the F2 resume path, because the fake would accept it.

Recommendation: make the fake `mkdir` (no `-p`) a run dir under `$AGENT_DISPATCH_STATE_DIR/<repo_key>/runs/$run_id` and exit 73 with "Run id already exists" on collision. Then add a resume test (F2).

### F4 (MEDIUM, disclosed but material) The harness knowingly differs from production on `packet_sha256`

Evidence (`dispatch_test.go` header): the real `agent-dispatch` hashes `$(cat "$prompt_file")`, which strips the trailing newline. The fake hashes the file bytes. The author states that with the real binary, the digest "would currently NOT match ReviewPacketDigest".

Impact: every real reviewer dispatch would likely fail `review-import`. The tests pass in spite of that. This is an unresolved integration defect, not a harness-honesty lapse, because it is disclosed. I did not verify the claim against the script.

### F5 (LOW) Worker retry: fresh run id is correct; two residual risks

Answer to check 4: `stepWorkerDispatch` builds a fresh run id per attempt (`freshDispatchRunID` with `freshToken`, 128 bits). It cannot hit the same-id collision. I found no parallel issue in the run-id handling.

Residual risks, not run-id related:
- Attempt 2 starts on top of any uncommitted working-tree debris from attempt 1. Nothing cleans it. Whether `agent-dispatch` rejects a dirty tree is unverified.
- The `headBefore != r.base` early return treats a moved HEAD as a done worker. It does not check scope or exit status of that earlier commit on resume. Scope is checked only in the invocation that made the commit.

### F6 (LOW) Other run-id / argument reuse (check 2)

- Preflight (`issuePreflight`): fresh token per call. Safe.
- Reviewer: the only fixed run id. See F2.
- `--prompt-file` for the worker is `workerPromptPath(attempt)`, rewritten each attempt. Safe.
- `AGENT_DISPATCH_STATE_DIR` is fixed per run. That is intentional, and the ledger path depends on it.

No other reuse of an `agent-dispatch` argument found.

### F7 (LOW) `classifyPreflightOutput`

Answer to check 5: it does not silently proceed past a bad preflight.
- A clean `AVAILABLE` requires `AGENT_DISPATCH_PREFLIGHT=ok`, exit 0, and `AGENT_DISPATCH_STATE=completed` (last occurrence).
- `failed` gives UNAVAILABLE. Missing line gives INDETERMINATE.
- Non-completed runs map to RATE_LIMITED or QUOTA_EXHAUSTED by keyword, else INDETERMINATE.
- `issuePreflight` writes the real status to the host chain, then returns an error for anything not AVAILABLE.

Defect: the match `strings.Contains(lower, "429")` runs over the whole output, which includes the run id printed by hmarun's own tooling and any hex digests. A hex token containing `429` can label a non-rate-limit failure RATE_LIMITED. It is still non-AVAILABLE, so the safety outcome is unchanged. Only the label and the advice are wrong. Tighten to a word-boundary match.

Also: if the host-chain write (`execHMA ... preflight`) fails when status is non-AVAILABLE, the function returns that error first and does not count a block (see F8).

### F8 (MEDIUM) Friction summary under-reports in two places beyond the disclosed gap

Counter derivation is honest where instrumented: `approvals` is incremented in `confirm()` for every approval request, including `--auto-approve`. It counts per invocation, so a resumed run does not carry earlier approvals. `summary.go` says "one invocation", so that is disclosed.

Disclosed gap check: the `summary.` sites are at run.go lines 589, 872, 894, 1333, 1348, 1357, 1367-1369, 1382-1383, 1732, 1740 (plus outcome at 497/502). No `summary.` use appears in `doTransition` or in `stepDecision`, `stepBindRepository`, `stepReviewPacket`, `stepReviewImport`, `stepFindingsImport`, `stepReconcile`, `stepCredit`. That matches the disclosure.

Undisclosed gaps:
1. **`recoveryAttempts` under-count.** In `stepWorkerDispatch`, attempt 2 can end by a launch error (1333), an out-of-scope commit (1348), or an in-scope commit with non-zero exit (1357). Each increments `blockedUnrecoverable` but not `recoveryAttempts`, although a retry occurred. Only success (1367) and no-commit exhaustion (1382) count it.
2. **Blocked count can be 0 on a blocked run.** The `outcome` field says "blocked at step X", but `blocked` shows 0 for any failure in the uninstrumented steps. This includes a `host review-import` rejection and the host-chain write failure in F7. The header comment in `summary.go` partly discloses this for I/O errors but not for review-import and reconcile rejections, which are real operational blocks.
3. **Stale comment.** The `summary.go` header says "stepReviewerDispatch's new one" retry. That retry was removed.
4. The `blockedRecoverable` count in `stepWorkerDispatch` is correct for the first-fail/second-succeed case (+1 block, +1 attempt, +1 success).

### F9 (LOW) Stale harness text

The `reviewer-crash-once` header comment says the second call "behaves exactly like reviewer-success". That is still literally true of the fake. It now reads as a design intent rather than a deliberate trap. Add a note that the production tool would refuse the second call.

## Answers to the numbered checks

| # | Answer |
|---|---|
| 1 | Mostly honest paths, with two gaps: F3 (no run-dir collision) and F4 (known `packet_sha256` divergence). |
| 2 | No other reuse in `run.go`. The same-class problem reappears on resume: F2. |
| 3 | Not performed (access denied). F1. |
| 4 | Fresh id per attempt; no run-id issue. Residual dirty-tree risk F5. |
| 5 | Correct and fail-closed. Cosmetic `429` defect F7. |
| 6 | Counters derive from real events. Under-reports found: F8. |
| 7 | F2, F4. |
| 8 | Not performed (no `go`). |
| 9 | Confirmed; no files changed except this report. |

## Next action

1. Re-run this audit's checks 3 and 8 on a host that has `go` and read access to `agent-run-ledger`.
2. Fix F2 before any real-tool run. Then fix F3 and F4.

## Re-review (2026-11-03)

**Verdict: PASS.** No new findings blocking. Two items deferred by the orchestrator are accepted as deferrable (below).

### 1. Executed checks (go1.27.3, `/home/yeli/.local/go/bin/go`)

| Command | Result |
|---|---|
| `go build ./...` | exit 0, no output |
| `go vet ./...` | no output |
| `go test ./cmd/hmarun/... -v -race` | PASS, `ok cmd/hmarun 18.060s`, no `--- FAIL` lines |
| `go test ./...` | all packages `ok` (cmd/hma 116.9s, cmd/hmarun 16.8s, internal/* incl. host, repostate, store, transition) |

### 2. Fix status (diffs read directly; files are untracked, so no `git diff` exists — I read the current source)

| ID | Status | Evidence |
|---|---|---|
| F2 | **Fixed** | `run.go` `reviewDispatchNeedsReissue` (~L1503-1535): returns false if a review record with a ReportDigest exists for the dispatch, or if the receipt/report is usable; otherwise true. `stepReviewPacket` (~L1568-1585) then mints a fresh packet at `freshDispatchRunID("review-reissue")`. `findReviewDispatch` scans history backwards, so the newest packet wins and the loop cannot re-select the dead one. `TestStepReviewPacketReissuesAfterDeadReviewDispatch` (dispatch_test.go L766+) asserts: crash, then new run id ≠ first, dispatch at new id succeeds, import yields APPROVE, and a later `stepReviewPacket` leaves the run id unchanged. It passed under `-race`. |
| F3 | **Fixed** | `dispatch_test.go` L150-157: `mkdir "$seen_dir/$run_id"` (no `-p` on the leaf), exit 73 + "Run id already exists" on collision. It sits after argument parsing and before `case "$scenario"` (L235), so it runs for every scenario. It is skipped only when `--run-id` is empty. Faithful model of mkdir exclusivity. |
| F4 | **Fixed** | Comment at ~L100-123 now says the digest mismatch was fixed upstream (sentinel-trimmed read) and describes the fake as unchanged. It no longer describes a live bug. I could not verify the upstream agent-dispatch claim myself (see §4). |
| F7 | **Fixed** | `run.go` L924-931 `standalone429 = \b429\b`, used at L955. Test case "hex token containing 429 is not a rate-limit signal" (dispatch_test.go L425) uses `a1b429c2deadbeef`, which has no word boundary around 429, and expects Indeterminate. The existing "(429)" case still expects RateLimited. |
| F8 | **Fixed (counting); disclosure honest** | `run.go` L1335-1339 increments `recoveryAttempts` once at the top of the loop when `attempt > 1`. `TestRunSummaryRetryEndingOutOfScopeStillCountsAsAnAttempt` (friction_test.go L113+, scenario `worker-crash-once-then-forbidden`) asserts 1. Other tests still assert 0 for no-retry cases (friction_test L39, L100) and 1/1 for a successful retry (L71). `summary.go` header discloses that `review-import`/`reconcile` rejections are not counted as blocks, and no longer refers to a reviewer retry. |

F9 (stale scenario comments) is also fixed: the `reviewer-crash-once` text (~L78-98) now states that a second call must use a different run id and that the same id is rejected by the exclusivity check first.

### 3. Deferred items — reasonable to defer

- **F5 residual LOWs** (dirty tree debris carried into retry attempt 2; `headBefore != r.base` early return not re-checking scope on resume): agree. They are independent of run-id reuse and low severity. They need a design decision. I recommend recording them as known limitations in the handoff.
- **F8 uninstrumented review-import/reconcile rejections**: agree. The outcome and error text still record the stop, and the summary now says so. Instrumentation is outside this pass.

### 4. Unconfirmed claim and F2 confidence

- I still could not read `/home/yeli/obsidian/AgentSystem/scripts/agent-run-ledger` (sandbox boundary). The claim that the real tool refuses a reused run id via a bare `mkdir` is **not independently confirmed by me**. I rely on the orchestrator's direct citation of that file. The same applies to the upstream agent-dispatch trim fix in F4.
- **F2:** I can confirm the hmarun side works. An executable, race-clean test shows a dead review dispatch gets a new run id, the new dispatch imports, and no second reissue occurs. The fake now rejects any run-id reuse. If the orchestrator's citation of the real tool is correct, the fix is sound end to end. It has not been exercised against the real `agent-run-ledger` in my review.

### 5. Working tree

`git status --short` before and after my work: `?? cmd/hmarun/` and `?? docs/handoffs/evidence/mvp/` (unchanged; the only edit I made is appending to this report). Nothing committed or pushed.

**Next action:** the human decides whether to proceed with the MVP step 3-4 gate; consider one real-tool smoke run of a crashed-then-reissued review.
