# Task 12 — Schema Amendments: Waiver and Worktree Binding

Status: implemented. Approved by Lennon on 2026-09-08.

The planning packet numbered this work Task 11; by the time it was approved,
the stage-transition write path had already taken that number
([docs/task11-stage-transition-path.md](task11-stage-transition-path.md)), so
the amendment is recorded here as Task 12.

Task 10 left exactly two bindings deliberately unbound, each requiring an
architecture amendment rather than an implementation decision: a waiver bound
to a produced head or diff digest, and an approval binding a worktree digest
([docs/task10-deferred-schema-decisions.md](task10-deferred-schema-decisions.md),
"What remains unbound"). This is that amendment, approved and implemented. The
architecture contract is amended in sections 6, 7.2, and 10.

## 1. Waiver binding (section 7.2)

A `WaiverOperation` may now optionally bind `head` (a commit SHA) and
`diff_digest`, alongside the `repository_identity` and `base_revision` Task 10
made bindable. When a binding is populated, `EvaluateResolution` compares it
against the live value the host supplies and rejects the operation with
`STALE_HEAD_BINDING` or `STALE_DIFF_BINDING` on mismatch.

Properties the fixtures pin down:

- the binding is optional: a waiver that declares neither stays valid, even
  when the host reports live head and diff values;
- it is fail-closed: a host that cannot report the live value cannot prove a
  bound waiver fresh, so the operation is refused rather than accepted on
  missing evidence;
- a clean-worktree head mismatch behaves like any other mismatch;
- only a waiver operation carries these fields: a finding override is never
  invalidated by live head or diff values, because it cannot bind them.

`hma resolve` accepts `head` and `diff_digest` as the live host-observed values
and surfaces the rejections with their reason and a non-zero exit, writing
nothing. The binding persists in the recorded `waiver_operation` record.

## 2. Approval worktree binding (sections 6 and 10)

An `ApprovalBinding` may now optionally bind `worktree_digest`. The binding is
optional at every stage, not only the four that bind head and diff, because
authorizing work against uncommitted content is most relevant at the earlier
unit gates. When populated, the transition is refused if the current worktree
digest differs from the approved digest — including when the worktree is now
clean, since a bound digest is never empty.

Enforcement lives in the engine and is wired from the two paths that hold a
live repository snapshot:

- `EvaluateApprovalBinding` refuses with `STALE_WORKTREE_BINDING` when the
  presented approval binds a digest that differs from the live
  `WorktreeDigest` on its request; `EvaluateStageTransition` carries the same
  live value through, so the stage-transition evaluator enforces the rule
  directly;
- `hma transition` (`transition.Apply`) checks the approval's binding against
  the live snapshot before the host-level declaration, so a drift refusal
  names the binding that failed rather than a generic staleness;
- `hma pilot` (`pilot.Run`) applies the same check and wires the live digest
  into the approval-binding evaluation.

When the approval binds no worktree digest, nothing changes: a dirty worktree
is still governed by the explicit dirty-worktree approval of the exact content
(`dirty_worktree_approved` plus `expected_worktree_digest`), exactly as in
Task 10. An approval that binds no digest is not newly invalidated by a dirty
worktree it never bound.

## What the fixtures prove

- `internal/engine`: a mismatched head or diff digest rejects a waiver
  operation (`STALE_HEAD_BINDING`, `STALE_DIFF_BINDING`); matching bindings
  pass; unbound operations and finding overrides are unaffected.
- `internal/engine`: a mismatched worktree digest refuses the stage transition
  (`STALE_WORKTREE_BINDING`), including the clean-worktree case; a matching
  binding passes; an unbound approval is unaffected by a live worktree digest.
- `internal/transition`, end to end against a real Git repository: a
  worktree-bound approval records against the exact approved uncommitted
  content, and after the worktree drifts the transition is refused with
  `STALE_WORKTREE_BINDING` and writes nothing.
- `cmd/hma`: `hma resolve` records a waiver bound to the live head and diff,
  and surfaces `STALE_HEAD_BINDING` / `STALE_DIFF_BINDING` on the command
  surface with a non-zero exit, writing nothing.

## Incidental repair

Verifying this task on macOS exposed a pre-existing defect unrelated to the
amendment: `repostate.resolveExisting` resolved a symlink's target without
re-resolving the target's own components, so on a system whose temporary
directory lives behind a link (macOS `/var` → `/private/var`) a store
symlinked into the repository compared as outside it, and
`TestStoreOutsideRepositoryRejectsSymlinkedStore` failed on a pristine
checkout. The walk now resolves to a bounded fixpoint; the test passes on
macOS and is unchanged on Linux.
