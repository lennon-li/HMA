# Audit: Gate E Phase 1 (defect G-R fix) — code-behavior review

- Date: 2026-10-31
- Reviewer: Phil (Copilot CLI, verification/review mode; model as dispatched: `gpt-6-luna`)
- Reviewed: uncommitted worktree diff of `cmd/hma/review.go`, `cmd/hma/host.go`, `internal/host/host.go`, plus untracked `cmd/hma/gatee_g_r_test.go`. HEAD `b845b73` unchanged.
- References: `docs/handoffs/2026-10-31-gate-e-checkpoint-triggers-draft.md` §2.3 (G-R), §3.5 E0 steps 4–5, §5.2.
- Go toolchain: go1.27.3 linux/amd64 (`/home/yeli/.local/go/bin`, put on PATH for the session; the absolute-path invocation form was denied by the sandbox, PATH form worked).

## Command results

| Command | Result |
|---|---|
| `go build ./...` | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| `go test ./...` | PASS — all 12 packages ok (`cmd/hma` 108.9s) |
| `go test -race -count=1 ./cmd/hma/... ./internal/host/...` | PASS — `cmd/hma` 136.3s, `internal/host` 3.3s |

## Red/green verification — LIMITATION

`git stash` was **denied by the sandbox** (rule `shell(git stash:*)`). I did not perform the revert/re-run, and I did not work around the denial. The red/green claim is therefore **not independently executed**. Evidence instead:

- Green: both named tests, and the extra `TestReviewPacketNonContractRunRiskCheckSkipped`, pass as part of the full `go test ./...` and `-race` runs above, with the fix present.
- Red, by reading: without the diff, `hoststate.ReasonReviewRiskBelowRequired` does not exist, so `gatee_g_r_test.go` would not compile (the `cmd/hma` package fails). Even with the constant, the packet path has no risk comparison, and `runHostDispatch` has no contract-bound review refusal. Both tests assert an exact error string and an unchanged host-record count. They would not pass against the old code. This is inference, not observation.

## Fix vs. §3.5 E0 steps 4–5 (read from the diff)

- Risk check uses `state.Contract.Risk` only; no floor. Matches (floor is Phase 4). Guarded by `state.Contract != nil`.
- Placement: inside the `appendHostRecord` closure, after the core-head check (`review.go:449`), before `validateRouteApprovalClaimIn` and before `EvaluateDispatch` (`review.go:465`). Matches.
- Packet-less review on a contract-bound run: `runHostDispatch` closure returns existing `ReasonReviewImportRequired` when `Role == RoleReview && engine.BoundContractDigest(core) != ""`. Placed first in the closure, before `EvaluateDispatch` (`host.go:256`). Reuses the existing reason. Matches.
- New reason name is exactly `HOST_REVIEW_RISK_BELOW_REQUIRED`. Matches the draft's PROPOSED name.

## Regression check

The diff to the three source files is additive only (24 added lines, none removed). No existing test file is modified. The only test change is the new untracked file. No assertion was loosened, skipped, or deleted. The new file adds a non-contract control test, so the check is shown not to fire on runs without a bound contract.

## Scope check

`git status --short` shows exactly: `M cmd/hma/host.go`, `M cmd/hma/review.go`, `M internal/host/host.go`, `?? cmd/hma/gatee_g_r_test.go`. HEAD is `b845b73`. No commit was made. This evidence file is the only file I added.

## Findings

1. **NOTE** — `riskRank` in `cmd/hma/review.go` is byte-for-byte equivalent (same switch, values 0/1/2/3/4) to the unexported `routeRiskRank` in `internal/engine/route_coherence.go`. Duplication is acceptable here because Phase 1 was scoped away from `internal/engine`. Phase 4 needs this ordering again for `RequiredRisk`. A later phase should consider exporting one helper (or placing it on `model.RouteRiskLevel`) and removing the copy, so the two cannot drift.
2. **NOTE** — Red/green was not executed (`git stash` denied; see above). Recommend the orchestrator or a human re-run the stash-based check if the empirical red result is required.
3. **NOTE** — An unrecognized risk ranks 0, below `low`. A dispatch with an invalid or empty `Risk` on a contract-bound run is therefore refused with `HOST_REVIEW_RISK_BELOW_REQUIRED`. This is fail-closed and consistent with the comment. No test covers the empty-risk case. Not a defect.

No BLOCKING, MAJOR, or MINOR findings.

## Verdict

**PASS WITH FINDINGS** (NOTE-level only; red/green is by inference, not execution).
