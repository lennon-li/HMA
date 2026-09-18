# HMA

Human-gated evidence and stage control for coding agents.

## Status

HMA v1 is released and tagged `v1.0.0`. It ships one standard-library-only Go
binary that captures revision-bound evidence, verifies challenge-bound human
approvals, records human resolutions, and verifies a CI runner against an
approved revision.

Read the boundaries below before relying on it. The six stage-control and
route evaluators are now reachable from the CLI through the read-only
`hma eval` family; see [Reachable surface](#reachable-surface).

## Purpose

Coding agents routinely advance on unsupported claims: they implement before grounding, declare success without fresh verification, overengineer beyond the approved scope, audit forever, retry failures without learning, and avoid saying `FAILED`, `BLOCKED`, or `UNKNOWN`.

HMA is a standalone, vendor-neutral gatekeeper for Git repository coding tasks. It does not automate coding. It assembles evidence, enforces deterministic safety rules, proposes agents and models, and asks a human to authorize every stage transition.

> No evidence, no transition.

## Core contract

- Humans approve every stage transition.
- HMA does not dispatch workers in v1; after approval it emits route and permission records, and a human or host starts execution.
- An interactive CLI agent is orchestration-first: it delegates coding unless
  direct execution is demonstrably cheaper for the bounded job and records
  that rationale before acting.
- Before dispatch, the host sends the approved route a minimal `hi` preflight
  and confirms that the exact route is reachable and not reporting an exhausted
  quota or rate limit; failure returns to the approved escalation route or a
  human gate, never a silent substitute.
- Machines gather evidence, enforce unwaivable rules, detect drift, and propose one next action.
- Interactive hosts use Jev, when configured and available, as an advisory decision tool for eligible-route choice, result sufficiency, and next-step selection; deterministic HMA policy and human approval remain authoritative.
- Agent prose is not proof.
- Every coding change receives independent review. The implementing agent,
  including an orchestrator using the direct-execution exception, cannot
  approve or validate its own work.
- `FAILED`, `BLOCKED`, `UNKNOWN`, `PARTIAL`, and `ABORTED` are honest supported outcomes.
- Auditing stops when approved criteria are resolved and no closure-blocking finding remains.
- Project policy may tighten the safety kernel but cannot weaken it.

## Install

```
go build -o hma ./cmd/hma
```

Requires the Go toolchain version in `go.mod`. There are no module dependencies.

## Commands

```
hma pilot  --input <file> --store <directory>
hma transition --input <file> --store <directory>
hma resolve --input <file> --store <directory>
hma eval [transition|invalidation|validator-contract
         |route-verification|route-coherence|route-policy] --input <file>
hma show --store <directory> --run_id <id>
hma ci github --store <directory> --run_id <id>
              [--verify-only | --repo-root <dir> --exec <prog> [--arg <a>]...]
hma inventory --allowlist <file> --repo-root <dir> --out <file>
```

`hma pilot` runs the host-trusted local pilot boundary: it verifies that the
repository is on the approved revision with a clean worktree, checks a
single-use challenge-bound approval against the live chain position, runs one
explicit evidence command with no shell, confirms the repository did not change
during capture, and appends the approval and evidence records.

A dirty worktree is refused unless the host sets both `dirty_worktree_approved`
and `expected_worktree_digest` and HMA's recomputed digest matches, so
approving a dirty capture approves one exact worktree state rather than
whatever is on disk at capture time. Such evidence records a `worktree_digest`
and is not revision-reproducible; see
[docs/task10-deferred-schema-decisions.md](docs/task10-deferred-schema-decisions.md).
An approval may also bind the worktree digest itself (`worktree_digest`,
optional at any stage); a transition or capture presenting such an approval is
refused with `STALE_WORKTREE_BINDING` when the current worktree differs from
the approved digest, including when the worktree is now clean.

`hma transition` records one human-approved stage transition. It is the only
command that changes a run's stage, and it changes one only because a human
already decided to: the decision arrives as a single-use, challenge-bound
approval binding, and the command refuses it unless the run is really in the
stage the approval was raised from, the edge is one the architecture lists,
the approval is fresh and matches the host's independently computed expected
binding exactly, its nonce has not been spent anywhere in the chain, and every
plan-declared required evidence digest is present in the chain and was
captured at the revision the approval binds. Recording a human's decision is
not making one: the result always reports `machine_advanced: false`.

Only the four stages whose approvals bind a produced head and diff —
implementation review, verification, independent validation, and release and
closure — are checked against the live head and diff. A packet is stale only
when a field it actually binds changes, so a commit landing after a grounding
approval does not invalidate it, and a commit landing after a review approval
does. A `worktree_digest` an approval itself binds is checked against the live
worktree at every stage; see
[docs/task12-schema-amendments.md](docs/task12-schema-amendments.md).

A run whose chain is empty is in `GROUNDING`, so `hma transition` is also how
a run starts; there is no separate `init`. A run that has recorded a terminal
outcome accepts no further transition.

Phase A1 terminal outcomes are reachable through `hma transition`: human-initiated
`ABORTED`, or human-confirmed `PARTIAL` when the committed criterion snapshot
contains at least one `PASSED`/`WAIVED` criterion and at least one
`PENDING`/`FAILED` criterion. The approval names exactly one stage or outcome
target. Outcome transitions append an `outcome` record and report
`to_outcome`, with `machine_advanced: false`. See
[docs/task13-terminal-outcomes.md](docs/task13-terminal-outcomes.md).

`hma resolve` records one human resolution — a waiver grant, expiry, or
withdrawal, or a finding-disposition override. It replays the committed chain
first, so an already-granted waiver cannot be granted again, a withdrawal with
no active waiver is refused, a `BLOCK` finding cannot be waived or overridden,
and a challenge nonce already used in the run is rejected as a replay. Setting
`expected_predecessor_head` binds the operation to the exact chain tail the
human approved against, and an operation may optionally declare
`repository_identity`, `base_revision`, `head`, and `diff_digest`, each
enforced against the live value the host reports when present: a waiver bound
to the produced head or diff it was granted against is refused with
`STALE_HEAD_BINDING` or `STALE_DIFF_BINDING` once the repository has moved on.

`hma eval` calls exactly one deterministic evaluator on one host input
document and prints its result. It is strictly read-only: it takes no store
argument, appends no record, advances no stage, grants no approval, selects no
route, and dispatches nothing. A `LEGAL_PENDING_HUMAN_APPROVAL` result is a
statement that a proposal *may be shown to a human* — never that anything
advanced. Any resulting record is the host's to write. Input is decoded
strictly: an unknown field or a second document in the file is refused.

`hma show` prints the projection of a committed chain — the stage the run is
in, whether it is terminal, and the criteria, findings, active waiver scopes,
and spent challenge nonces the chain already says. It replays the chain and writes nothing. Waiver scopes and
nonces are sorted, so repeated invocations over an unchanged chain are
byte-identical.

`hma ci github` verifies that the GitHub Actions runner is on the approved
repository and revision. It writes an evidence record only when `--exec`
actually runs a command and produces an exit code and an output digest; it
never manufactures evidence from environment variables. Use `--verify-only` to
check the revision without recording anything. The host must bind the
approval's repository identity to the `owner/repo` slug for this adapter,
because that is the only identity the runner can derive independently.

`hma inventory` captures host-local machine truth: it runs only the commands
named in a strict `machine-truth-allowlist/v1` file, each by absolute path,
with no shell, a per-command timeout, and only the environment the allowlist
supplies. It writes the digested, freshness-bounded inventory to `--out` and
prints a summary. The allowlist and the output must both resolve outside
`--repo-root`, through symlinks too. An inventory is availability evidence
only; nothing consumes it for route selection or eligibility.

## Reachable surface

Implemented and reachable from the CLI:

- host-trusted local pilot with approval binding and evidence capture
- recording a human-approved stage transition, via `hma transition`
- human resolution: waivers and finding-disposition overrides
- GitHub Actions revision verification and CI evidence capture
- the chained JSONL run store
- read-only evaluation of six deterministic evaluators, via `hma eval`: stage
  transitions, approval invalidation, validator contract, route attestation,
  route coherence, and the versioned route-policy contract
- the stage and resolution projection of a chain, via `hma show`
- host-local machine-truth inventory from an allowlist, via `hma inventory`

`hma eval` is not the whole evaluator surface. The remaining deterministic
evaluators are reached through the command that owns their side effects, not
through `eval`: approval binding and resolution through `hma pilot` and
`hma resolve`, CI verification through `hma ci github`. `EvaluateWaiverChange`
has no command of its own.

Deliberately **not** reachable from any command, and not planned:

- approval granting, route selection, or worker dispatch — HMA never
  manufactures the human decision it requires, and never starts a worker
- machine stage advancement — no command moves a run on its own judgment.
  `hma transition` moves a run only by recording a challenge-bound decision a
  human already made, and verifying that decision is still valid against the
  chain and the repository

Deferred to later phases:

- recording `BLOCKED`, `UNKNOWN`, or `FAILED` (Phase A2), or verified terminal
  closure (Phase B); the Phase A1 write path explicitly refuses these outcomes

`hma eval` classifies a proposed transition without recording anything;
`hma transition` records one a human approved. Neither supplies the decision.

## Shape

- One standalone Go binary
- Standard-library-first implementation, no module dependencies
- Content-addressed JSON records; the portable JSON Schema the
  architecture calls for is specified but not yet published in this repository
- Host-managed, detection-based run store: an append-only JSONL chain with a
  per-run head anchor, an exclusive per-run lock, and a two-phase commit that
  recovers an interrupted write instead of leaving the run unreadable
- Fresh command/evidence capture bound to exact Git revisions, with the
  executable pinned to a resolved absolute path and no shell
- Human challenge-bound, single-use approvals
- Primary agent/model route plus one explicit escalation route
- Independent semantic validation
- Local gating plus CI verification through independent validation and a release request; human release approval follows passing CI

## Trust boundary

The run store is **host-trusted**. Its head anchor and predecessor chain detect
accidental truncation, rollback, and corruption, and the per-run lock prevents
concurrent writers from losing each other's records. None of this is a
tamper-resistance claim against an adversary running as the same user, who can
rewrite the store and its anchors together. Approval nonces, revision binding,
and evidence freshness are enforced within that boundary, not against it.

The GitHub Actions actor is unauthenticated environment data and is recorded
with a `github-actions:` provenance prefix rather than as a human identity.

## Documentation

- [Architecture and product contract](docs/architecture.md)
- [Jev decision-tool host integration](docs/jev-decision-tool.md)
- [Task 9 — CLI surface](docs/task9-cli-surface-proposal.md)
- [Task 10 — deferred schema decisions](docs/task10-deferred-schema-decisions.md)
- [Task 11 — the stage-transition write path](docs/task11-stage-transition-path.md)
- [Task 12 — schema amendments: waiver and worktree binding](docs/task12-schema-amendments.md)

## Deliberate non-goals for v1

- Autonomous coding orchestration
- A daemon, database, TUI, or plugin framework
- Provider-specific model SDKs
- Generic support for non-Git tasks
- Silent model fallback, automatic publication, or self-approval

## License

MIT. See [LICENSE](LICENSE).
