# Independent review: cmd/hmarun Steps 1–2

Reviewer: Phil (separate context). Date: 2026-11-02. Subject: untracked `cmd/hmarun/` (main, init, answers, build, interactive, run + tests).

## Verdict: FAIL (resumability) — safety-kernel posture otherwise sound

One blocking defect (F1): a run interrupted at any point after the worker commit can never be completed. It fails closed, so it is a liveness defect and not a safety hole. Two further findings (F2, F3) should be fixed or accepted explicitly. Items 3, 7 and 8 are clean.

## Verification commands (run by me)

| Command | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | all packages `ok` (cmd/hma 121s, cmd/hmarun 3.1s, internal/* ok) |
| `go test ./cmd/hmarun/... -v -race` | PASS (`TestRunReachesPartial`, `TestRunResumesAfterPartialProgress`, build/init tests) |
| `git diff --stat -- internal/engine internal/transition cmd/hma` | empty (item 8 confirmed) |

`git status --short` shows `cmd/hmarun/` and an unrelated untracked `ap-acceptance-20261101-01-prof-hma/` (contains `marker.txt`; not mine, not examined further).

Method for items 3, 4, 6, 7: a temporary in-package probe test (`zz_phil_probe_test.go`) calling the real runner against the real `hma` binary built from this tree. A shell wrapper around `hma` injected a failure on the Nth `hma` call, either before the real call ("before") or after it succeeded ("after"). Each case was then resumed with a fresh `runRun`, and compared to a clean run. The probe deleted itself afterward. No existing source file was edited. `cmd/hmarun/` now holds only the original 9 files (verified).

Clean-run reference: core=12 records, host=9 records, 8 approval nonces, 1 distinct `route_approval_digest` (appearing 5 times), final state PARTIAL.

## Findings

### F1 (HIGH, item 4) — `base` is re-derived per process; resume wedges after the worker commit
`newRunner` sets `r.base = git rev-parse HEAD` (run.go:273) on every invocation. The comment says it is "fixed for the lifetime of the run", but it is only fixed within one process. The worker step creates a commit, so HEAD moves. A resumed process therefore uses a different `base_revision` than the original one.

Observed with crash injection (fresh `runRun` after the crash):

- Crash in the `implementation-review` transition, `bind-repository`, `review-preflight` or `review-packet` (before and after variants): resume fails with `HOST_REVIEW_REPOSITORY_MISMATCH`.
- Crash in `review-import`, `findings-import`, `reconcile` or `verification`: resume fails at `verification` with `HOST_REVIEW_ARTIFACT_MISMATCH`.
- Crash in `independent-validation` or `credit`: resume fails at `partial` with `HOST_CREDIT_REFUSED`.
- Crashes at hma calls 1–7 (before the worker commit) resume correctly, in both modes. Only a crash at call 18 ("after", already terminal) also passes.
- The `after n=8`…`n=17` and `before n=8`…`n=18` cases all fail to finish (about 21 of 36 injected crash points).

Fail-closed: the kernel refuses each time, nothing is bypassed, and no record is duplicated. But the run is permanently stuck.

The existing test `TestRunResumesAfterPartialProgress` stops before the worker commit, which is why it passes.

Recommended fix: when the core chain already has records, take base from the chain (the first approval's `BaseRevisionDigest`) instead of the current HEAD; use HEAD only for a fresh run. Add a resume test that stops after `stepWorkerStub`.

### F2 (MEDIUM, item 5) — stub review looks like a real review in the chain; approval prompts do not say so
Stub output is labeled in stdout (`STUB: ...` lines), in code, and by route names (`hmarun-stub-reviewer`, `stub-provider-b`, `stub-model-b`). The receipt and AUDIT carry no false real-model names. However:

- AUDIT.json asserts `"verdict":"pass"`, `"isolation":"isolated"` (no isolation happened), `"limitations":[]`, and a requirement `status:"pass"` with `evidence_refs:["hmarun-stub"]`.
- `hma host review-import` and `hma host credit` accept it, so the chain gets an `independent_review` record and criterion C1 = PASSED. Nothing in the chain's kernel-visible fields marks this as non-evidence, except the string names above.
- The approval prompts for `verification`, `independent-validation`, and `PARTIAL` read "Present unit … reviewed implementation…" with no stub warning. `host credit` has no human prompt at all.
- Stubs run even without `--auto-approve`. A human typing a name approves transitions resting on a verdict no human or model rendered. (Probe: interactive stdin run completes with `approvers={alice:8}`.)

Recommended: set `isolation` to a non-claiming value if the schema allows, put `STUB` in `audit_id`/`limitations`, prefix the stub-dependent approval summaries with "STUB REVIEW (not real evidence)", and make a run that used stubs refuse to be labelled as anything but test-mode until the real dispatch replaces them.

### F3 (MEDIUM, item 6) — EOF on stdin causes an unbounded busy loop
`prompter.readLine` swallows `io.EOF` and returns `("", nil)`. `askRequired` then loops forever printing "… is required." With empty stdin and no `--auto-approve`, `hmarun run` spun for the 3s probe window and emitted about 931 MB of output (about 300 MB/s). A second probe (one line, then EOF) behaved the same.

Safety result: it never approves. Core records stayed at 0 and no approval existed. It does not auto-approve, but it also does not "refuse"; it hangs and floods. Fix: return an error on EOF in `readLine`/`askRequired`.

Note: the repository-identity prompt silently accepts its default on EOF (before the loop). Low risk, but with the loop fix it should also error on EOF.

### F4 (LOW, item 4) — stub worker is not atomic
`stepWorkerStub` appends a marker, then `git add`/`git commit`. It skips only when the latest commit message equals the stub message. A crash between the write and the commit leaves a dirty file, and a resume appends a second marker line. Also, it commits to whatever `--repo` points at; there is no "throwaway repo" guard. Not exercised by my wrapper (which only intercepts `hma`); this is from code reading.

### F5 (LOW) — conflicting contract on resume
Resuming with a contract whose criterion IDs differ from the chain baseline is rejected by the kernel (`ACCEPTANCE_CONTRACT_CRITERIA_MISMATCH`), and the run is stuck. This is the correct fail-closed behavior. The error text could say "use a new run_id".

## Item-by-item

| # | Result |
|---|---|
| 3 ensureCriteriaBaseline | PASS. See below. |
| 4 idempotency/resume | FAIL for post-worker interruption (F1). Pre-worker resume and re-run of a completed run: no duplicate records, no reused nonce (every resumed case: nonces unique, count equals clean run, 1 distinct route digest). |
| 5 stub labeling | PASS WITH FINDINGS (F2). |
| 6 auto-approve | Default off; explicit flag required; approver string is `AUTO-APPROVE-TEST-MODE (not a human approval)`. No path approves without the flag or typed input. EOF loops (F3). With typed input, approver is the typed string. |
| 7 route digest | PASS. Digest is re-read from the core chain on every use (run.go:~340), never cached. In all 36 injected-crash cases and the clean run, exactly 1 distinct `route_approval_digest` appeared in host records. |
| 8 untouched kernel | PASS (empty diff). |

## `ensureCriteriaBaseline` (item 3)

Observed behavior, by probe:

- Empty chain: no write (0 baselines).
- At ACCEPTANCE_CRITERIA, three consecutive calls: exactly 1 baseline, `[{C1 PENDING} {C2 PENDING}]`.
- A runner for a different contract (X1/X2) against the same store: writes no second baseline; the kernel rejects the transition with `ACCEPTANCE_CONTRACT_CRITERIA_MISMATCH`.
- Two concurrent runners, first baseline: one succeeds, the other gets `record chain discontinuity` from `store.Append`. Result: 1 baseline.

Code guarantees: writes only when `len(state.Criteria)==0`, stage is ACCEPTANCE_CRITERIA, chain non-empty; dispositions hard-coded `PENDING`; IDs from `ContractCriterionIDs` of the contract it then presents. I could not make it write a second baseline, a non-PENDING one, or a mismatched-ID one. No finding.

Soundness of the design: sound as implemented. It is the same bootstrap the kernel's own test fixtures use, it is idempotent, append-only through `store.Append` (sequence and predecessor enforced), and the kernel still independently validates IDs against the presented contract. Two residual notes: (a) the baseline is appended before the human approves the contract, so a declined approval leaves a harmless PENDING record; (b) it is attributed to `hmarun:core-bootstrap`, not a human. A kernel-side `hma` command for baseline creation would be the cleaner long-term answer, but is out of scope here.

## Confirmations

- I did not edit any existing source file. I created one temporary probe file (self-deleted, absent now) and this report.
- Nothing committed.

## Next action

Fix F1 (derive base from the chain on resume), then F3 and F2. Add a resume test that stops after `stepWorkerStub`. Send for re-review.

## Re-review after hardening (2026-11-02)

**Verdict: PASS.** F1, F2, and F3 are each closed. F4 and F5 were out of scope and are not re-assessed.

### Evidence

| Check | Observed |
|---|---|
| `go build ./...`, `go vet ./...` | Clean. |
| `go test ./...` | All 13 packages `ok` (`cmd/hma` 114.7 s, `cmd/hmarun` 9.4 s). |
| `go test ./cmd/hmarun/... -v -race` | `PASS` (10.5 s). |
| `git diff --stat -- internal/engine internal/transition cmd/hma` | Empty. |

### F1 — closed
I rebuilt my original method. A shell wrapper around the real `hma` failed the Nth call before it ran, or after it succeeded. A fresh `runRun` then resumed. I covered n=1..18 in both modes (36 crash points, including after `review-packet`, `reconcile`, and `credit`). In all 36 cases:
- the resume returned no error;
- the run reached terminal `PARTIAL`;
- the core chain had 12 records and the store had 25 JSON files, equal to the clean run;
- no nonce appeared twice.

Before the fix, about 21 of the 36 cases wedged. `newRunner` now reads `base_revision` from the first core record when the chain is non-empty. I did not trust the new tests alone. The wrapper run is independent evidence.

### F3 — closed
`hmarun run` with empty stdin and no `--auto-approve` returned `input ended before an answer was given` in under 1 s. Output was 176 bytes (one prompt). Nothing was written to the core or host chain. The repository-identity prompt no longer takes its default on EOF. The `cmd/hmarun` tests for `readLine`/`askRequired`/`ask` EOF pass.

### F2 — closed
- `hma host show` prints `"actor":"hmarun STUB REVIEW (no real evidence)"` on the review dispatch, `independent_review`, and `criterion_credit` records. `actor` survives import. The marker appeared 4 times in the dump.
- The four approval prompts print the marker as a prefix, observed in real output:
  - `STUB REVIEW (no real evidence) — Present unit "U1"'s reviewed implementation for verification.`
  - `… — Advance unit "U1" to independent validation.`
  - `… — Credit unit "U1"'s criterion "C1" from the imported independent review.`
  - `… — Record unit "U1"'s run "run" as PARTIAL: …`
- `stepCredit` now has a confirmation prompt.

### Residual notes (not blocking)
- The `launch_reconciliation` record carries no marker. It does not depend on review content, so I do not require one.
- Under `--auto-approve` the prefix is printed but no human reads it. That is expected for test mode.

### Housekeeping
I added a temporary probe, `cmd/hmarun/zz_phil2_probe_test.go`. The sandbox denied `rm`, so I could not delete it. Both tests in it are now `t.Skip`. **The human should delete this file before any commit.** I edited no other source file. I committed nothing. `/tmp/phil-answers.json`, `/tmp/hmarun-p`, and `/tmp/hma-p` are scratch files outside the repo.

### Next action
Delete `cmd/hmarun/zz_phil2_probe_test.go`, then proceed to the next stage gate under `docs/architecture.md` §18.
