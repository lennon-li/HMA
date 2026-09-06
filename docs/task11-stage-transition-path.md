# Task 11 — the stage-transition write path

Status: implemented

## The gap

Before this change, `ProposedTargetStage` was validated in three places and
consumed in none. The four record writes in the binary were `hma pilot` (an
approval and an evidence record, both at the run's *current* stage), `hma
resolve` (a waiver or override), and `hma ci github` (evidence). `hma eval
transition` classified a proposed edge and, by the Task 9 read-only design,
recorded nothing.

So HMA could prove a transition legal and refuse an illegal one, but no code
path moved a run from one stage to the next. The nine-stage model was fully
specified, fully evaluated, and unreachable. "No evidence, no transition" was
the product, and the binary had no transition at all.

`hma transition` closes that gap.

## Decisions

### The machine records a decision; it does not make one

`hma transition` writes a record only when a human has already decided. The
decision arrives as the single-use, challenge-bound `ApprovalBinding` the
architecture already specifies, and the command's whole job is to refuse it
unless it is still valid: raised from the stage the run is actually in, naming
an edge §5.4 lists, matching the host's independently computed expected binding
field for field, fresh, with an unspent nonce, and with its plan-declared
required evidence really in the chain.

`machine_advanced` is structurally `false` in every result, as it is
everywhere else in the engine.

### `Control` describes the stage the record is filed under

The first implementation filed an APPROVED record under the stage the run was
moving *to*. The existing fixtures said otherwise, and they were right:
`StageControlState` is a *stage*-control vocabulary, so `Stage: VERIFICATION,
Control: APPROVED` means "the verification stage is approved", and the run's
new position is that approval's `ProposedTargetStage`.

Both readings replay unambiguously, so this was not a correctness tiebreak —
it was a question of whose contract governs. The vocabulary's own definition
does, so the implementation changed to match rather than redefining the
vocabulary by implementation. Two fixtures that had drifted from their own
approvals were corrected in the same pass, and `ValidateRecord` now enforces
the rule: a `stage_transition` is filed under its approval's current stage, and
an APPROVED one must carry the approval that proves it.

### A run with an empty chain is in GROUNDING

A run has not left its first stage until a human approved leaving it, so an
empty chain projects to `GROUNDING` and `hma transition` is also how a run
starts. This keeps the Task 9 decision to have no `hma init`: there is still
exactly one way state enters a store, and it is a human decision.

### Only bound fields make a packet stale

Architecture §6 binds a produced head and diff for implementation review,
verification, independent validation, and release and closure — and for no
other stage. `hma transition` therefore compares the live head and diff only
for those four. Comparing them everywhere would invent a staleness rule the
contract does not have, and a grounding approval would be invalidated by a
commit it never bound. `model.ApprovalBindsHeadDiff` is now exported so the
rule has one definition rather than two.

### Required evidence must be present *and* fresh

An approval's `RequiredEvidenceDigests` are satisfied only by evidence in the
chain whose output digest matches *and* whose `BaseRevision` is the revision
the approval binds (and whose `HeadRevision` matches when the approval binds a
produced head). Existence alone would let evidence gathered before the code
changed satisfy a requirement after it changed, which is the "stale evidence
reused after a revision changes" failure in §2. Missing and stale are reported
as distinct reasons.

### One definition of a spent nonce

`ProjectStageState` collects challenge nonces from every record family, not
just approvals, and `hma pilot` now uses it. Previously a nonce burned on a
waiver could be replayed as a pilot approval nonce, because pilot built its own
narrower set.

### `internal/repostate`

`snapshotRepo`, the worktree-content digest, and the porcelain status parsing
moved out of `internal/pilot` into `internal/repostate`, shared with
`internal/transition`. Duplicating security-critical digest framing into a
second package was the alternative and is worse. `VerifyBase` moved with it and
now has its own tests, including the case that an abbreviated or symbolic
revision is not an acceptable spelling of an exact base.

## Deliberately deferred: terminal outcomes

`hma transition` moves a run between stages. It cannot end one.

`EvaluateTransition` already classifies a proposed `TargetOutcome`, but
`ApprovalBinding.ProposedTargetStage` is typed as a `Stage` and
`validateApproval` requires it to be a valid one. Approving a run into
`FAILED`, `BLOCKED`, `ABORTED`, `PARTIAL`, `VERIFIED_WITH_WAIVERS`, or
`VERIFIED_SUCCESS` therefore cannot be expressed in the approval contract as
written.

Widening that contract is a change to architecture §6, not an implementation
detail, and closure additionally requires CI and human release approval that
this command does not model. It is left open rather than guessed at.

Until it is decided, a run can be driven from `GROUNDING` to
`RELEASE_AND_CLOSURE` and stops there.

## Verification

Every new test was run against a deliberately broken implementation first and
confirmed to fail — the standing check in this repository, after vacuous tests
appeared three times in two days. The mutations exercised were: projection
ignoring `Control`, required evidence never checked, stage mismatch never
checked, nonces collected from approvals only, the head/diff comparison
removed, every stage treated as head-binding, the record never appended, and
the record invariant disabled.

`TestApplyWalksEveryStage` drives a real Git repository through all eight
ordinary edges from `GROUNDING` to `RELEASE_AND_CLOSURE`, one separate human
approval per edge, and asserts the chain replays to the final stage.
