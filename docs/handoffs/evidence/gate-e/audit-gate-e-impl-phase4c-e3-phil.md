# Gate E Phase 4c (E3 final gate) — independent code-behavior review

- Reviewer: Phil (Copilot CLI, verification mode). Same-family, separate-instance review (not cross-family).
- HEAD: `4a34b09` (unchanged). Reviewed: uncommitted worktree diff.
- Toolchain: go1.27.3 (`/home/yeli/.local/go/bin`).
- Scope of my writes: this file only. I made temporary edits to `cmd/hma/credit.go` for mutation probes. I restored them and confirmed the `git diff` was byte-identical to the starting diff (`cmp`). I deleted my `/tmp/phil_*` scratch files (`rm` was denied; `os.remove` via Python worked). No stray files remain.

## Command results

| Command | Result |
|---|---|
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test ./...` | all packages ok (`cmd/hma` 116.7 s) |
| `go test -race -count=1 ./cmd/hma/...` | ok (144.9 s) |

## Checks

### 1. Non-contract success-claim refusal — PASS
- Observed: `runTransition` condition is now `VERIFIED_SUCCESS || VERIFIED_WITH_WAIVERS || PARTIAL`. Inside it, the first action after the host gate is `engine.BoundContractDigest(core) == ""` → `HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM`.
- This runs before the `repostate.Snapshot`, the credit check, and the active-waivers check. Zero PASSED criteria therefore cannot bypass it (the old vacuous path).
- Only `requireHostTransitionGateIn` runs earlier. It returns `nil` for every edge except the two per-unit edges and the checkpoint edge. On the checkpoint edge, a non-contract run is allowed through per D10. So no other reason precedes this one for a non-contract success claim.
- `FAILED`, `BLOCKED`, `UNKNOWN`, `ABORTED` do not match the condition. No new check is reachable for them. `TestStoppingOutcomesUnaffectedByE3` covers `FAILED` only. The other three rely on `TestTransitionCLIPhaseA1`/`A2`, which pass.

### 2. E3 checks in `requireIndependentCredit` — PASS (with test-coverage findings)
- Observed: for every PASSED criterion's credit, after `RequireLatestImportedReview`, the code looks up the dispatch packet via `packetFor`. It refuses with `HOST_FINAL_AUDIT_MISSING` unless the packet is non-nil, `CheckpointSchema != ""`, `IssuingReason == final`, and `BoundContractDigest == contract`. A nil packet is refused.
- The loop skips non-PASSED dispositions, so `PARTIAL`'s "only PASSED needs credit; WAIVED needs nothing more" needed no structural change. `TestPartialWaivedOnlyNeedsNoReview` passes.
- Red/green mutation (see 5) shows the `IssuingReason` branch is enforced by tests.
- Mutation probes show **two of the four conditions are not enforced by any test**:
  - Dropping `|| packet.BoundContractDigest != contract`: the full `cmd/hma` suite still passes.
  - Dropping `packet.CheckpointSchema == ""`: the full `cmd/hma` suite still passes.
- Inference: the contract-digest check is partly redundant. The credit's own `ContractDigest != contract` check runs first, and import binds the review to the contract at dispatch. I did not prove full redundancy. The `CheckpointSchema` check has no test with a legacy (no-schema) final-reason packet.

### 3. Inner active-waivers check — PASS
- Read directly in the diff: the `ActiveWaivers > 0` check is still under `if o == model.OutcomeVerifiedSuccess`. The only change is that `core` is now loaded once in the outer branch instead of inside it. It is not widened to `PARTIAL` or `VERIFIED_WITH_WAIVERS`.

### 4. Test-file modifications

Overall: the changes match §4's prediction (the two named files, narrowly). None is loosened, except as noted in 4b.

- **4a. `newCreditFixtureArgs` default `IssueFinal`.** Strengthened, not loosened. A `newCreditFixtureReason` helper exists for other levels. Callers still run the same credit/transition flows, now with a packet that satisfies E3, so they keep testing their original intent. The full suite passes. The G-F negative tests use `newCreditFixtureReason` or hand-built per_change packets (`TestClosureWithPerChangeReviewOnlyRefused`, `TestClosureSameFamilyNormalRiskRefused`, `TestPartialWithPerChangeCreditRefused`).
- **4b. `TestCreditRefusals` non-isolated case — real coverage loss (MAJOR-leaning MINOR).**
  - Observed: under the final-level fixture, import itself refuses the `partial`-isolation APPROVE (`HOST_REVIEW_ISOLATION_INSUFFICIENT`). The test now takes `continue` before `credit()` is called.
  - So `credit.go`'s own `rv.Isolation != "isolated"` refusal (credit.go:117) lost its only exercise. I mutated it out (removed that condition). The full `cmd/hma` suite still passed. `grep` of `isolation` in the tests finds no other credit-path test. `gateb`/`gatec` isolation tests run at import only. The `gatee_e2` fixtures build records directly and never call credit.
  - The check is not provably dead. A per_change or checkpoint-level partial-isolation APPROVE can still be imported, and credit does not require final level. Mitigation: E3 still blocks closure on such a credit, but the credit record itself could now be created without the isolation guard, unnoticed by tests.
  - The fixture comment discloses the redirect. It does not disclose the lost coverage. The test's `continue` branch is a valid (still-asserted) import-refusal check, so the assertion was adjusted, not weakened. The coverage loss is a side effect.
- **4c. `TestTransitionCLIPhaseA1` `PARTIAL`.** The new assertion (`HOST_CONTRACT_REQUIRED_FOR_SUCCESS_CLAIM`) replaces a vacuous success: a non-contract run with no PASSED criterion used to record `PARTIAL` with no error. This closes the D10 gap and does not paper over a meaningful assertion. The `return` leaves the post-checks of that subtest unexecuted for `PARTIAL` only; the other outcomes run them unchanged.

### 5. Red/green for `TestPartialWithPerChangeCreditRefused` — CONFIRMED
- First attempt (reverting `credit.go`/`transition.go` wholesale) fails to build: the new tests reference the reason constants defined in those files. This is not a red result.
- Second attempt: I replaced only the E3 condition in `requireIndependentCredit` with `if false && packet != nil {`. Result:
  - `TestPartialWithPerChangeCreditRefused`: FAIL, `PARTIAL recorded over a per_change-backed credit: got <nil>, want HOST_FINAL_AUDIT_MISSING`.
  - `TestClosureWithPerChangeReviewOnlyRefused` and `TestClosureSameFamilyNormalRiskRefused`: FAIL, `got <nil>`.
  - Stale-credit, waived-only, and stopping-outcome tests: PASS.
- The report's claim holds: without the E3 block, `PARTIAL` over a per_change credit is accepted with `err=nil`. I restored the file; `git diff` is byte-identical to the start.
- Note: my first neutralization attempt had an operator-precedence mistake and left the `IssuingReason` branch live, so the tests passed. I discarded that result and redid it correctly.

### 6. Scoped-out `TestFinalReviewSatisfiesBothE2AndE3` — acceptable as disclosed, not blocking
- Judgment: the safety-relevant half (E3 refuses a per_change-backed credit; E2 refuses per_change at checkpoint) is covered by separate red/green tests. The unverified part is positive: one final-level review accepted at both the checkpoint gate and the credit gate. That is a liveness/usability property (no false refusal), not a safety bypass.
- The existing fixtures run credit under a final-level packet and the checkpoint tests use final reviews, so each side is exercised on its own. I did not see a composed run.
- Risk: a hidden mismatch (for example, the E2 review record differing from the credit's `ReviewRecord`) would only show as a false refusal in a real contract-bound closure. Recommend tracking it as an open item before the E-series end-to-end/dogfood step. It should not block this merge.

### 7. Build and test — see table. All green.

### 8. Scope — PASS
- `git status`: `M credit.go`, `M gated_credit_test.go`, `M transition.go`, `M transition_outcomes_test.go`, `?? gatee_e3_test.go`. Nothing else. HEAD `4a34b09`. Nothing committed.
- Note: this evidence file is the one additional file I wrote.

## Findings

| # | Severity | Finding |
|---|---|---|
| F1 | MAJOR (non-blocking, judgment) | `credit.go:117` non-isolated refusal lost its only test coverage when the fixture default changed. Mutation survives the full suite. The check is reachable and not provably redundant. Restore a test (credit at per_change via `newCreditFixtureReason` with a `partial`-isolation APPROVE). |
| F2 | MINOR | `packet.BoundContractDigest != contract` is untested. Mutation survives. The check may be partly redundant with the credit's own contract check. |
| F3 | MINOR | `packet.CheckpointSchema == ""` is untested (no legacy/no-schema final-reason packet case). Mutation survives. |
| F4 | MINOR | `TestStoppingOutcomesUnaffectedByE3` directly covers only `FAILED`. `BLOCKED`/`UNKNOWN`/`ABORTED` rely on the unchanged A1/A2 CLI tests. The test comment states this. |
| F5 | NOTE | `TestFinalReviewSatisfiesBothE2AndE3` not built (see 6). Disclosed. Not blocking. Positive-path composition is unverified. |
| F6 | NOTE | Fixture comment discloses the import-time redirect but not the credit-coverage loss in F1. |

No BLOCKING findings. Source behavior matches the draft §3.5/§4/D7/D10 and architecture §16/§20.

## Verdict: PASS WITH FINDINGS

Next action (architecture §18): the human decides whether to accept F1–F3 as follow-ups or to require tests first, then approve or reject the E3 commit. I have not committed.
