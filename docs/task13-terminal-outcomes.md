# Task 13: Phase A1 and A2 terminal outcomes

Phase A1 extends `hma transition` to record `ABORTED` and `PARTIAL` through
the existing challenge-bound human approval path. It implements the
[architecture §6](architecture.md#6-human-approval-contract) amendment and
the stopping edges in §5.4. This supersedes the terminal-outcome limitation
described in the historical [Task 11 write-path notes](task11-stage-transition-path.md).

## Approval and command

Use the existing command:

```sh
hma transition --input /host/transition.json --store /host/run-store
```

The host input retains the existing repository, freshness, expected approval,
and presented approval fields. In both approval bindings, name exactly one
target. For example, an abort from grounding uses these target fields:

```json
{
  "current_stage": "GROUNDING",
  "proposed_target_stage": "",
  "proposed_target_outcome": "ABORTED"
}
```

This fragment is not a complete approval: all universal §6 bindings, including
the transition digest covering the populated target, remain required.
The trusted host prepares the expected binding; HMA compares the presented
binding field for field. HMA does not manufacture a human approval or choose
its target. Required evidence, repository/base, stage-time, nonce, expiry,
and optional worktree binding checks remain in force. The four later stages,
including `RELEASE_AND_CLOSURE`, still require the live produced head and diff.

`proposed_target_outcome` is additive with `omitempty`; an empty outcome
does not change existing approval JSON or record hashing. The existing
`proposed_target_stage` field retains its original serialization, including
an empty string for an outcome approval.

## Outcome rules

- `ABORTED` is human-initiated from any nonterminal stage. It needs no
  outcome-specific supporting evidence or criterion disposition.
- `PARTIAL` is derived from the committed criterion snapshot and confirmed
  by the human: at least one criterion must be `PASSED` or `WAIVED`, and
  at least one must be `PENDING` or `FAILED`. An empty, wholly unresolved,
  or wholly resolved set is refused. An all-passed run cannot use PARTIAL
  to bypass verified closure.
- The criterion snapshot is the latest non-empty `criteria` list in the
  committed chain, matching the existing resolution projection. Lists are
  snapshots, not per-ID patches; a later snapshot replaces the earlier one.
  The host-trusted chain carries the approved criteria and dispositions.
  The transition input cannot supply a replacement list. Phase A1 does not
  change criterion approval or waiver resolution semantics.
- `BLOCKED`, `UNKNOWN`, and `FAILED` require the matching live classification
  projection described below. `VERIFIED_SUCCESS` and `VERIFIED_WITH_WAIVERS`
  remain deferred to Phase B and are refused by this write path.
  The pure edge classifier continues to describe the full architecture.

## Record and result

A successful outcome transition appends a `KindOutcome` record at the current
stage with `APPROVED` control, the matching outcome approval, and the current
criterion snapshot. It does not claim that skipped stages were completed.
The result adds `to_outcome`; `to_stage` remains present as an empty string.
Stage transition results omit `to_outcome`, preserving their prior shape.

The chain projection marks the run terminal, retains its last stage, and
consumes the approval nonce. Any later transition, including an outcome replay,
is refused. Every result retains `machine_advanced: false`: recording the
human's decision is not making one.

Store loading verifies record hashes, chain continuity, and the host head
before evaluation. Corrupted target fields or head anchors are refused.
The store remains host-trusted and detection-based; this phase adds no trust
roots or protection against a same-UID adversary rewriting the entire store.

## Verification and scope

Tests cover both outcomes from `GROUNDING` through the write path and CLI,
criterion boundary cases, all deferred outcome refusals, target mismatches,
stale bindings, spent nonces, post-terminal refusal, and tampered chains.
Golden legacy approval and record serialization protect existing digest inputs.

No release gate, publication, dispatch, stage waiver,
or validator-independence change is included. The execution packet assigns
independent review to Ming (Claude Code), a different model family from the
Codex implementer; implementation tests do not substitute for that review.


## Phase A2 classification records

Use `hma transition --input <file> --store <directory>` with the same host
and approval bindings, plus this structured input:

```json
{
  "classification": {"outcome": "BLOCKED", "finding_id": "F1"}
}
```

Both `expected_approval` and `approval` must additionally contain
`"unit_id": "U1"` and the identical `classification` object, with
`proposed_target_outcome` matching the classification and an empty target stage.
All existing repository, evidence, freshness, stage-time, and single-use nonce
checks apply. The host's transition digest covers the classification decision;
the engine also compares the structured payload and unit to the expected binding.
The allowed outcomes encode permission/safety block (`BLOCKED`), unknown failure
(`UNKNOWN`), and material failure (`FAILED`). Finding and unit IDs are required;
free-text outcome attestations are rejected. The human/host has already matched
material failures; the engine does not count attempts or infer route identity.

This writes `KindClassification`, with approved control and the standard approval,
without moving the stage or making the run terminal. The result's `to_outcome`
reports the classified outcome. Projection exposes `FailureOutcome` separately
from the committed terminal `Outcome`.

A later human-approved state change with exactly the same record `unit_id`
supersedes the classification, regardless of record kind. Approved stage changes
(including rewinds) take their unit from `approval.unit_id`; `hma resolve` accepts
an optional `waiver_operation.unit_id` or `finding_override.unit_id` inside the
human-approved payload. The record unit must match that payload; resolutions without
a unit cannot supersede classifications. Top-level `unit_id` input is rejected.
Other units, earlier changes, and unapproved records do not clear classifications.
A new classification on the same unit replaces the previous one.

Across live units, projection selects `BLOCKED > UNKNOWN > FAILED`.
To commit that terminal outcome, submit a fresh standard outcome transition
without a classification payload. Its target must match the projection.
A live failure rejects `PARTIAL`, while its existing mixed-criteria rule remains
unchanged. Human-initiated `ABORTED` retains precedence. Recorded terminal outcomes
continue to refuse further transitions.
