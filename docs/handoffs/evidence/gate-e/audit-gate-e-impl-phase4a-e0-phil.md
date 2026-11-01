# Gate E Phase 4a (E0 wiring): independent code-behavior review

- Reviewer: Phil, via GitHub Copilot CLI, verification/review mode.
- Model: Claude Sonnet 5.5 (`claude-sonnet-5.5`). This is a same-family, separate-instance review (no shared context with the implementer). It is not cross-family. Lennon accepted this.
- Target: uncommitted worktree on HEAD `186461b`. Modified: `cmd/hma/review.go`, `internal/host/host.go`. Untracked: `cmd/hma/gatee_e0_test.go`, `internal/host/gatee_e0_checkpoint_binding_test.go`.
- Spec: `docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md` §3.4, §3.5 E0, §3.5a.
- Toolchain: go1.27.3 (`/home/yeli/.local/go/bin`).

## Verdict: PASS (no BLOCKING or MAJOR findings; 2 MINOR, 2 NOTE)

## Command results

| Command | Result |
|---|---|
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test ./...` | All 13 packages ok (`cmd/hma` 110.8s) |
| `go test -race -count=1 ./cmd/hma/... ./internal/host/...` | ok (`cmd/hma` 136.3s, `internal/host` 3.3s) |
| `go test ./cmd/hma/... -run TestD5C5PBoundInputSwapsAfterParse -v` | PASS (current code) |
| Same test, with reads moved after the hook (my probe) | **FAIL**: `HOST_EVIDENCE_ARTIFACT_CORRUPT` (`d5c5p_regression_test.go:137`) |

## TOCTOU ordering verification (D5c)

1. Code read. In `runHostReviewPacket` (`cmd/hma/review.go`), the block that calls `loadCheckpointPolicy` (line ~574) and `checkpointReviewBaseline` (line ~578) comes directly after `corePacketStateWithPlan`. It sits in its own `if state.Contract != nil` block. The next block calls `reviewPacketBeforeManifestHook()`. Both reads run before the hook. Confirmed.
2. Current code: `TestD5C5PBoundInputSwapsAfterParse` passes.
3. Fail-then-revert probe. I saved the diff, then moved the two calls to just after the hook invocation, inside the hook's `if` block. I re-ran the test. It failed with `HOST_EVIDENCE_ARTIFACT_CORRUPT`. I restored `review.go` from a byte copy. `git diff` before and after the probe is byte-identical (`cmp` OK). The ordering matters and the fix is real.

Interpretation: the failure proves the test guards this ordering. It does not prove the policy file itself is untrusted after the hook; the retained-plan re-read in `checkpointReviewBaseline` is the likely trigger.

## Other checks

### 1. Diff against §3.5 E0 and §3.5a

- §3.5a fields: all 11 present on `ReviewPacketBinding` with the specified JSON tags, all `omitempty`. `RiskFloor` is `low` or `high` when the schema is set, so it is never empty there. Types match (`*engine.CheckpointBaseline` for covers).
- `validatePacketBinding`: refuses any checkpoint field without `checkpoint_schema` (covers all 10 other fields, including the `CheckpointCovers != nil` test). It refuses an unknown schema and an invalid `minimum_issuing_reason`. Legacy bindings return early and marshal as before (`TestLegacyHostRecordAnchorUnchanged`).
- E0 step 2: refusal is `rank(in.IssuingReason) < rank(decision.Level)`. A host may meet or exceed the level, never go below. An unknown reason ranks 0 and is refused. The error text carries the derived level. Correct.
- E0 step 3: `checkpointPacketHeader` prints the fields in the packet header (so the digest covers them). Skipped when no schema.
- E0 step 4: G-R now uses `riskRank(decision.RequiredRisk)`, in the locked closure after the head check. It is not `RiskFloor` alone and not `contract.Risk` alone. `binding.ContractRisk`, `RiskFloor`, `BoundContractDigest`, `FamilySeparation` are recorded.
- `EvaluateDispatch`: adds `(d.Packet != nil && d.Packet.FamilySeparation)`. All four prior terms (high/critical risk, `Disputed`, `FinalValidation`, `IssueFinal`) remain. Confirmed in diff (one `-` line changed, `||` appended).
- E0 step 5: the packet-less `runHostDispatch` refusal is already in HEAD (`cmd/hma/host.go:249-251`). It is not part of this diff. No action needed.

### 2. Baseline / Current derivation (§3.4)

- `checkpointReviewBaseline` scans `history` from the end backward. It takes the first approved review for the unit whose packet has a schema, non-nil `CheckpointCovers`, and `IssuingReason` checkpoint or final. That is the latest such review. Correct.
- Fallback: `firstContractAcceptance` returns the first approved `ACCEPTANCE_CRITERIA -> PLANNING` transition's contract digest. `firstLeasePermissionClasses` returns the classes from the plan of the first record for which `engine.IssuesLease` matches the unit (loop returns on the first match). Both are first, not latest. Correct. Classes are sorted for determinism.
- `Current` = bound contract digest plus sorted current unit classes. Correct per §3.5a.
- Gate on `Baseline`/`Current` and the policy: `Contract == nil` skips the classifier; policy is nil when the contract has no digest (all paths unclassified, per the code comment). A digest mismatch refuses with `ReasonPolicyNotCurrent`.

### 3. Regression check on existing tests

- `git ls-files -m` lists only the two source files. No existing test file was modified, so no assertion was loosened.
- The only removed source lines are the old risk check, the old `strong` expression tail, and the packet template's blank line (replaced by `%s` plus an extra header). No behavior removed.

### 4. Scope

- Only the two source files and two new test files. Nothing else touched. Nothing committed (HEAD `186461b`). The only other file I wrote is this evidence file.

## Findings

| # | Severity | Finding |
|---|---|---|
| 1 | MINOR | New tests (`gatee_e0_test.go`) cover too-low reason, below-floor risk, populated fields, non-contract absence. I found no test that exercises the baseline derivation (latest checkpoint-grade review vs. first-acceptance/first-lease fallback). That is the §3.4 logic most prone to a latest-vs-first mistake. I verified it by reading only. |
| 2 | MINOR | `validatePacketBinding` accepts `checkpoint_schema` with empty `minimum_issuing_reason`, `risk_floor`, `checkpoint_covers`, etc. §3.5a says these are "only with schema" and `RiskFloor` is "always set" there. Absence under a schema is not rejected. Not reachable from `review-packet`; only a hand-built binding. |
| 3 | NOTE | The packet template changed from `valid_until: %s\n\nReport` to `valid_until: %s\n%s\nReport`. By reading, an empty header yields the identical legacy bytes. A checkpoint header ends in `\n`, so it is followed by the blank line. I did not diff golden packet bytes. |
| 4 | NOTE | `checkpointReviewBaseline` and `firstLeasePermissionClasses` call `loadPlans`, a disk read. Safe only while they stay before the hook (§ above). A comment in the code states this; the D5c test enforces it. |

## Unresolved / not checked

- I did not run the new tests individually beyond the full suite (they pass in it).
- Finding 3 was not tested against a golden packet.

Next action: Lennon decides whether to accept the PASS and commit Phase 4a, with findings 1 and 2 recorded for follow-up.
