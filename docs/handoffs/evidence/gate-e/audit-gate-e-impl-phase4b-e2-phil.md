# Gate E Phase 4b (E2 checkpoint gate): independent code-behavior review (Phil)

- Reviewer: Phil, GitHub Copilot CLI, same-family separate-instance review (not cross-family; per Lennon's prior acceptance).
- Target: uncommitted worktree. HEAD `98af25b`. `cmd/hma/transition.go` (modified), `cmd/hma/gatee_e2_test.go` (new, 412 lines).
- Go: go1.27.3 (`/home/yeli/.local/go/bin/go`).
- Date: 2026-11-01.

## Answer first

**PASS WITH FINDINGS.** The E2 logic matches the architecture text and the draft. Precedence is correct and nothing is dead. One MAJOR finding: the new edge makes `HOST_UNIT_ID_REQUIRED` fire for non-contract runs (and any run whose approval has no `unit_id`) on `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE`. That contradicts the D10 non-contract exemption. I reproduced it with a probe (check 1, F1).

## Command results

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | all 13 packages `ok` (`cmd/hma` 112.3s) |
| `go test -race -count=1 ./cmd/hma/...` | `ok` (139.6s) |
| Probe `TestZZNonContractNoUnit` (temporary) | `HOST_UNIT_ID_REQUIRED` on a non-contract run, empty unit ID |
| Probe `TestZZMultiCycle` (temporary) | before 2nd IV entry: `nil`; after: `HOST_CHECKPOINT_REVIEW_MISSING` |

## Check 1: scope creep and flag threading

(a) Both flags are optional. `fs.String("contract", "", ...)` and `fs.String("checkpoint-policy", "", ...)` default to `""`. Confirmed.

(b) `requireHostTransitionGate` forwards `"", ""`. Its callers are all in `_test.go` files (`gateb_*`, `host_test.go`, `gatec_review_import_test.go`, `gatee_e2_test.go`). None of them pass the checkpoint edge except the new E2 tests, and the non-checkpoint edges return before the new code runs. Confirmed unaffected.

(c) Other call sites. `requireHostTransitionGateIn` is called only from `runTransition` (production) and from `gatee_e2_test.go`. `runTransition` is called from `main.go` and four existing tests (`gateb_r2_test.go:545`, `gateb_r2_review_test.go:315`, `gateb_r4_hidden_mode_probe_test.go:162`). None needed a signature-aware update; the full suite passes. Confirmed.

Design judgment (assumption/recommendation, not proof):
- Given the constraint that `internal/transition/` was off-limits, flags on `hma transition` are the smallest correct way to thread the documents. The flags are read only after the run is known contract-bound, and review-packet issuance already takes the same two inputs through the same `corePacketState`/`loadCheckpointPolicy` path. So this is consistent with existing practice, not a new pattern.
- It is still a mild design smell. The host input file (`--input`) is the normal carrier of transition inputs, and the CLI now has a second channel. It also makes every contract-bound IV exit require `--contract` even when no trigger would fire, because the contract must be proven before triggers can be computed. This fails closed, so it is safe, but it is an operator-visible change that needs documentation.
- The three new reason constants live in `cmd/hma/transition.go`, not `internal/host`. The file comment says so and says why.
- Recommendation: acceptable to merge. Record in the roadmap that a later phase may move the two paths into `transition.Input` and the reasons into `internal/host`.

## Check 2: `HostRisk`/`Disputed` left at zero

Facts from `internal/engine/checkpoint.go`:
- `HostRisk` is used in exactly one expression: `combinedRisk := maxRouteRisk(contractRisk, in.HostRisk, riskFloor)`.
- `combinedRisk` feeds T2 (`>= high`) and `FamilySeparation`. `RequiredRisk` is `max(contractRisk, riskFloor)` and does not use `HostRisk`.
- `Disputed` feeds only `FamilySeparation`. It does not feed any trigger or `RequiredRisk`.
- The gate reads only `decision.Triggers`, `decision.RequiredRisk`, and `decision.PolicyDigest`. It never reads `FamilySeparation`.

Reasoning:
1. `Disputed` cannot change any value the gate reads. Zero is exactly equivalent.
2. `HostRisk` can only change the gate input through T2. `maxRouteRisk` is monotone, so gate triggers with `HostRisk=0` are a subset of gate triggers with any `HostRisk`. Zero can only remove T2, never add a trigger.
3. T2 is therefore dropped at gate time only when `max(contract risk, risk floor)` is below `high`. If `max(contract risk, risk floor)` is `high`, T2 fires at gate time too, with or without `HostRisk`.
4. The recorded-triggers-superset check (`recordedTriggers ⊇ decision.Triggers`) is one-directional. A recorded packet that includes T2 from a real `HostRisk` while gate time has no T2 satisfies the superset check trivially. That never rejects a review. The extra recorded trigger only makes the recorded set larger, which cannot cause a false refusal or a false pass on any trigger the gate computed.
5. So the answer to the question asked: yes, this can only make the gate's bar lower, never silently waive a check among the triggers the gate computes. The recorded superset is harmless.

The one real effect, stated plainly (observed from the code, not from a test): a contract-bound unit whose *only* trigger would be T2-from-host-risk (contract risk low, no floor-raising path class, no other trigger) will pass E2 with no checkpoint review, even if an earlier dispatch declared `HostRisk=high`. At E0, the same unit with that dispatch would have been classified `checkpoint` at stage IV. This is a lowering, bounded to that narrow case. It is consistent with D8 (1), because the architecture's E2 text says "recomputes the classification at gate time on the live target" and no dispatch is live at a gate. `HostRisk` is also host-written and untrusted, while contract risk and floor are trusted. If a host did dispatch a high-risk review in IV, that review's own packet records T2 and E2 uses it normally. I classify this as a NOTE, with an open question for the architecture owner: should gate time use the latest review's dispatch risk as `HostRisk`? That would be circular (the review under test would define its own bar), so I recommend keeping zero.

## Check 3: "latest" `ivApprovalIndex`

Confirmed intentional and correct. The loop has no `break`, so it ends at the last matching approved `VERIFICATION -> INDEPENDENT_VALIDATION` record for the unit. A unit that re-enters IV after a rewind must be audited against its latest entry; an earlier entry would let a pre-rewind review stand in.

The existing suite has no multi-cycle test (the only IV-ordering test is `TestCheckpointPerChangeReviewCannotStandIn`, a single cycle). I wrote a temporary probe (`TestZZMultiCycle`). It seeded the standard IV fixture, confirmed a qualifying review passes (`nil`), then appended a second approved `VERIFICATION -> INDEPENDENT_VALIDATION` record after the review's dispatch anchor. The gate then returned `HOST_CHECKPOINT_REVIEW_MISSING`. A first-match loop would have returned `nil`. Correct.

Related detail, correct: `dispatchIndex < ivApprovalIndex` allows equality, which is the normal case where the dispatch's core head is the IV approval record itself.

## Check 4: refusal precedence

Order in `requireCheckpointGate`, top to bottom:

1. Run-open, bound-contract exemption (returns `nil` for non-contract), manifest, `corePacketState` (route/contract/plan proofs; fires `HOST_ROUTE_APPROVAL_MISSING` and similar), policy load, baseline.
2. No triggers: `nil` (D8 (1)).
3. `requireFindingsResolved`.
4. `MISSING`: no latest review, review not answering the latest dispatch, not APPROVE, not isolated, no packet, no `checkpoint_schema`, wrong `issuing_reason`, no dispatch record, no IV approval, dispatch before the IV approval.
5. Findings snapshot (`requireOnlyOwnSnapshotsSince` or `ReasonFindingsSnapshotMissing`).
6. `TRIGGERS_UNCOVERED`: recorded triggers not a superset, then policy digest mismatch.
7. `STALE`: bound contract digest mismatch, then dispatch risk below `RequiredRisk`.
8. `ReasonReviewArtifactMismatch`: worktree digest, then artifact digest.

This matches the implementer's claim and the draft's list. No check is dead: each reason has a test that reaches it (`MISSING`, `TRIGGERS_UNCOVERED`, `STALE` both conditions via `TestCheckpointReviewStaleFieldMismatchReachable`, artifact mismatch is shared with E1 and was not reached by an E2-specific test; see F3). The architecture text does not rank the reasons, so the implemented order is a reasonable, documented choice. One observation: unresolved findings (step 3) with no review at all returns the findings reason, not `MISSING`. Acceptable, since both refuse.

## Check 5: rewind-vs-gate-refusal claim

Read both tests. The claim is accurate.
- `checkpointReacceptContract` appends a second `ACCEPTANCE_CRITERIA -> PLANNING` approval with a different contract digest. `TestCheckpointStaleAfterContractReaccepted` and `TestCheckpointStaleAfterPolicyChange` both assert `ReasonRouteApprovalMissing`, with a comment giving the reason (route invalidation inside `corePacketState`, before E2's own staleness checks).
- The assertions are exact string equality on the refusal, so they are not loosened versions of a STALE assertion. A weaker alternative (accepting any error) was not used.
- `TestCheckpointReviewStaleFieldMismatchReachable` independently drives each STALE condition (contract digest mismatch, risk below required) through the `Dispatch.Risk` and `BoundContractDigest` fields, with both subtests asserting `reasonCheckpointReviewStale`. So STALE is reachable and tested, not rationalized.
- Limitation (observed): no test reaches STALE via a *real* contract re-acceptance, because the route-approval check always wins first. This means the STALE contract-digest branch is defense in depth behind the route check in practice. That is acceptable and the tests state it honestly.

## Check 6 and 7: build, scope, no loosened tests

- Build, vet, full suite, and race suite pass (table above).
- `git status`: `M cmd/hma/transition.go`, `?? cmd/hma/gatee_e2_test.go`. HEAD still `98af25b`. Nothing committed.
- The diff to `transition.go` is purely additive apart from the two signature/forwarding edits and the added edge condition. No existing test file is modified (`git status` shows no `M` on any `_test.go`), so no existing assertion was loosened.

## Findings

| ID | Severity | Finding |
| --- | --- | --- |
| F1 | MAJOR | The new edge is added to the guarded-edge condition before the `UnitID == ""` check and `requireCoreRunOpen`. A non-contract run, or any contract run whose approval has no `unit_id`, now returns `HOST_UNIT_ID_REQUIRED` on `INDEPENDENT_VALIDATION -> RELEASE_AND_CLOSURE`. At HEAD, this edge returned `nil` before reaching those checks. `unit_id` is optional on approvals (`types.go`: "empty means no unit supersession"). Reproduced by probe `TestZZNonContractNoUnit`. This contradicts the non-contract exemption (D10, architecture §16) and is a behavior change for existing non-contract runs. For a contract-bound run, refusing on an empty unit is correct fail-closed. Fix direction (for the implementer, not done here): evaluate contract-bound status before requiring a unit ID or run-open on the checkpoint edge, and add a test that a non-contract run with no unit ID still passes. `TestCheckpointGateIgnoresNonContractRun` misses this because it sets `UnitID: "U1"`. |
| F2 | MINOR | Because `requireCheckpointGate` returns before `requireReconciledLaunches`, the E2 edge does not run the unreconciled-launch check that the other two edges run. The architecture's E2 text does not require it, so this is a design choice. Decide explicitly and record it. |
| F3 | MINOR | No E2-specific test reaches `ReasonReviewArtifactMismatch`. The branch is a copy of the E1 rule but is untested at the E2 edge. |
| F4 | NOTE | Check 2: a unit whose only trigger is T2-from-host-risk passes E2 with no checkpoint review. Narrow, spec-consistent, only lowers the bar. See Check 2. |
| F5 | NOTE | Every contract-bound IV exit now requires `--contract`, and `--checkpoint-policy` when the contract carries a policy digest, even when no trigger fires. Fails closed. Needs operator documentation. |
| F6 | NOTE | The three reason constants live in `cmd/hma`, not `internal/host`. Documented in a code comment. |
| F7 | NOTE | Process: my temporary probe file `cmd/hma/zz_probe_test.go` could not be removed. The `rm` command was denied by the tool permission rules, and I did not work around the denial. **The file is still untracked in the worktree and must be deleted by the human or the next session before any commit.** `transition.go` and `gatee_e2_test.go` were not modified by me. |

No BLOCKING findings.

## Verdict

**PASS WITH FINDINGS.** F1 should be fixed (or explicitly accepted by Lennon) before merge. F7 must be cleaned up before commit.

Next action: Lennon decides whether F1 is fixed in Phase 4b before merge, and removes `cmd/hma/zz_probe_test.go`.
