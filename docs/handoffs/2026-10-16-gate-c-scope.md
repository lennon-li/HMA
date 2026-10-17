# Gate C scope — review provenance and finding import

**Date:** 2026-10-16
**Base:** plan §5 Gate C; design Gate A v3 §4, v4 §5, v5 §1/§3, v6 §1 (latest wins).
**Survey:** Argie (Antigravity, Gemini 4 Argon, read-only), `hma-gate-c-survey-20261016`. Claims spot-checked
by Ming against code and a live receipt.
**Author:** Ming / Anthropic `claude-opus-5-5` / Claude Code CLI.

## 0. Sequencing decision (lead)

Reservations (`hma host attempt`, admission, consumption, launch reconciliation, `valid_until`
generations) all require an active lease (v4 §1, §3; v7 §1). Leases require `hma.plan/v1` and the
acceptance contract, which are Gate D. Reservation work therefore moves to Gate D with the lease. Gate C
keeps everything that does not need a lease. No design rule changes; only the gate that implements it.

### 0.1 Moved requirements and Gate D exit tests (Liz Gate C r1 F12)

Moved from Gate C to Gate D, unchanged in content. Gate D may not be claimed passed, and no
bounded-execution or closure claim may be made, until all of these tests exist and pass:

| Requirement (design) | Gate D exit tests (names from the design, kept) |
|---|---|
| Reservation protocol: replay first, admission, consumption, unknown launch counted (v4 §3) | `TestReservationIdempotent`, `TestConcurrentLastAttempt`, `TestUnknownLaunchCounts`, `TestCountersCarryAcrossRewind`, `TestSameFailureSignatureTwiceRefused`, `TestDifferentRouteRefused`, `TestEscalateToMustBeStop` |
| Final-slot replay and `launch_eligible` (v5 §2, v6 §3) | `TestFinalSlotReplayLaunchesOnce`, `TestFinalSlotReplayAfterUseNotEligible`, `TestNewIDWithoutCapacityRefused` |
| Launch reconciliation, `UNAUTHORIZED_DISPATCH` (v5 §1-§2, v6 §3) | `TestOneReservationPerLaunchNoReuse`, `TestUnlaunchedReservationCharged`, `TestUnreservedDispatchBlocks`, `TestEnvelopeAndAssociationMustMatchAdmission` |
| Validity interval and generations for reservations (v6 §2, v7 §1) | `TestLaunchAtValidUntilUnauthorized`, `TestLaunchAtRevocationTimeUnauthorized`, `TestValidUntilMinOfTwoRevocations`, `TestExpiryThenLaterRevocation`, `TestRenewalDoesNotReviveOldGeneration`, `TestNewGenerationNeedsNewReservation`, `TestLaunchAtEarliestEndUnauthorized`, `TestLaunchBeforeEarliestEndAuthorized` |
| D5 HMA endpoint for the verify-loop authority hook, resume replay (v4 §4, v6 §5) | `test_hma_bound_launch_requires_eligible_true`, `test_resume_replay_uses_same_dispatch_id` against `hma host attempt` |
| Positive criterion credit (v3 §4.3 a+b) | `TestScorecardOnlyIndependentCredit`, `TestPartialNeverVerified` (needs the acceptance contract) |
| BLOCK/WAIVABLE from imported findings (v3 §4.1) | needs a kernel-rule registry and HMA reproduction, and plan budgets; until then every imported finding is ADVISORY |

**Human decision (Lennon, 2026-10-17): ACCEPTED.** This acceptance boundary for Gate C is approved: the rows
above move to Gate D with their exit tests, together with the full fixed review-packet inputs (request,
criterion definitions, non-goals, budgets, required evidence, policy; Jax r3 R3-02/A04) that need the
acceptance contract. No bounded-execution, criterion-credit, or closure claim may rely on them until Gate D
passes those tests.

## 1. C1 — HMA-generated review packet (`hma host review-packet`)

- Input: host envelope, unit, implementation record digest, target (`produced_head`, `diff`,
  `worktree`), reviewer route, review preflight digest, pre-allocated `dispatch_run_id`, issuing reason
  (`per_change` | `checkpoint` | `final`), risk, disputed, final validation, task text.
- HMA generates a 128-bit `review_nonce` and a `review_dispatch_id`, writes the packet file, and computes
  its SHA-256. The packet tells the reviewer to copy `hma_binding` {`packet_sha256`,
  `review_dispatch_id`, `review_nonce`} into the report.
- HMA records the agent-dispatch launch digests it expects (`launch_head`, `launch_worktree_sha256`, same
  definition as `agent-run-ledger compute_launch_digests`).
- HMA appends one review `dispatch` record through the existing dispatch checks (preflight, route,
  independence incl. disputed/final family separation), carrying the new fields. Hand-written review
  dispatches stay accepted for legacy hosts but cannot use review import.

## 2. C2 — receipt-checked review import (`hma host review-import`)

Imports only if every check holds; else refuse with a named reason and append nothing:

1. review dispatch record with `review_dispatch_id` exists, is HMA-generated, and its implementation is
   still the unit's current one;
2. receipt `run_id` == `dispatch_run_id`; `mode` == `review`; `state` == `completed`; `exit_code` == 0;
3. receipt `packet_sha256` == recorded packet digest; `launch_head` and `launch_worktree_sha256` == the
   recorded expected values;
4. receipt `agent`, `model` == dispatch route worker and model (independence was checked at dispatch);
5. report bytes hash to a `report_files` entry; report `hma_binding` equals the packet's three values
   (a previous or rewritten report carries another nonce and is refused);
6. report `isolation` == `isolated` when the dispatch is `final` or `final_validation`.

Verdict map (total): `pass` → `APPROVE`; `fix-required` → `REQUEST_CHANGES`; anything else (incl.
`conditional-pass`, `blocked`, `not-evaluated`, `unknown`, missing) → `ESCALATE`, next action
`ESCALATE_TO_HUMAN`. Only `APPROVE` satisfies the review gate. The record binds receipt, report, and
findings digests.

## 3. C3 — imported findings and §9 disposition

- Core `finding` records gain additive `findings_snapshot` (v3 §4.2). With it, the list replaces the
  previous snapshot, even if empty. An empty snapshot is accepted only when bound to an imported `APPROVE`
  review of the exact current target. Without the flag, behaviour is unchanged.
- Import mapping (v3 §4.1, applies to imported audit findings only; the validator contract keeps
  architecture §16 rules): severity never sets disposition; a kernel-rule reference that HMA reproduces →
  `BLOCK`; a complexity-budget reference → `WAIVABLE`; everything else → `ADVISORY`. A failed criterion
  check creates no finding class; the criterion is not credited.
- No import path credits an acceptance criterion `PASSED` (survey Q4: none exists today; tests keep it so).

## 4. C4 — fixtures

Audit failure → linked remediation → re-verification; stale packet; rewritten old report; same-route
reviewer; reviewer that completed without a report imports nothing; verify-loop evidence never yields
criterion PASS.

## 4a. Gate C r1 remediation

Review packets bind the artifact HMA observes (`repostate.Snapshot` of root/base, optional approved dirty
content) and the canonical project root; the host may not supply the artifact digest. A packet is valid
for launch in `[dispatch time, +1h)`. Receipt agent, model, provider, effort, and project root must match.
Only the unit's latest packet is answerable, once. Once a run has any packet, host-written review dispatches
and reviews are refused run-wide. Review import and findings import require the current core head (one
snapshot per review). A final audit needs provider- and model-family separation. The reviewer must be
independent of every implementation of the unit. Reports must be one JSON document with a findings array;
a pass inconsistent with the report contract (non-isolated, high/critical finding, non-passing or missing
requirements, an HMA citation) is escalated. An imported review counts at the gate only after its findings
snapshot is recorded. A snapshot never weakens or drops another unit's findings, and a non-approving one
never weakens this unit's. `hma host loop-evidence` imports an HMA-bound verify-loop state, re-runs its
targeted check without a shell, and records it as core evidence; it credits nothing.

## 4b. No-shell boundary for loop re-capture (limit, recorded)

Evidence capture refuses: a shell executable (by name or through symlinks); any command wrapper (env, xargs,
nice, ...) as the executable or through symlinks; a script whose `#!` interpreter chain reaches a shell or a
wrapper (so `#!/usr/bin/env python3` scripts are refused; run `python3 script.py`); and a shell followed by a
command-string flag anywhere in argv. A shell name used as plain data is allowed. **Limit:** a compiled program
that starts a shell itself cannot be distinguished from any other program by inspection. Under the trusted-host
model this is bounded in Gate D, where the re-captured argv must equal a check approved in the accepted plan
(`hma.plan/v1`); until then the argv comes from an HMA-bound verify-loop envelope only.

## 4c. Repository binding

Each run records its one repository with `hma host bind-repository` (canonical top-level root and the
repository identity every approved core transition binds), once, before any review packet or loop evidence; it
is never taken from a packet or journal (`HOST_REPOSITORY_BINDING_MISSING`, `HOST_REPOSITORY_BINDING_EXISTS`).
A review packet must name that root and the base of the unit's active route approval; this is checked before
the packet is written, under the locks at dispatch, and at review and findings import
(`HOST_REVIEW_REPOSITORY_MISMATCH`). Loop evidence must run in that root at that base, and the journal's
`repo.root` must name it (`HOST_REPOSITORY_MISMATCH`). Legacy runs without a binding cannot import until it is
recorded. Imported finding ids may not contain `/`, and a snapshot with a duplicate global id is refused.
**Trust note:** the binding is host-written (trusted-host model); Gate D ties it to the accepted plan.

## 5. Next action (§18)

Implement C1–C4 with tests, run full verification, then Liz blind audit of Gate C.
