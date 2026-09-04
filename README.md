# HMA

Human-gated evidence and stage control for coding agents.

## Status

HMA v1 is released and tagged `v1.0.0`. It ships one standard-library-only Go
binary that captures revision-bound evidence, verifies challenge-bound human
approvals, records human resolutions, and verifies a CI runner against an
approved revision.

Read the boundaries below before relying on it. Every deterministic evaluator
is now reachable from the CLI through the read-only `hma eval` family; see
[Reachable surface](#reachable-surface).

## Purpose

Coding agents routinely advance on unsupported claims: they implement before grounding, declare success without fresh verification, overengineer beyond the approved scope, audit forever, retry failures without learning, and avoid saying `FAILED`, `BLOCKED`, or `UNKNOWN`.

HMA is a standalone, vendor-neutral gatekeeper for Git repository coding tasks. It does not automate coding. It assembles evidence, enforces deterministic safety rules, proposes agents and models, and asks a human to authorize every stage transition.

> No evidence, no transition.

## Core contract

- Humans approve every stage transition.
- HMA does not dispatch workers in v1; after approval it emits route and permission records, and a human or host starts execution.
- Machines gather evidence, enforce unwaivable rules, detect drift, and propose one next action.
- Agent prose is not proof.
- The implementing agent cannot approve or validate its own work.
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
hma resolve --input <file> --store <directory>
hma eval [transition|invalidation|validator-contract
         |route-verification|route-coherence|route-policy] --input <file>
hma show --store <directory> --run_id <id>
hma ci github --store <directory> --run_id <id>
              [--verify-only | --repo-root <dir> --exec <prog> [--arg <a>]...]
```

`hma pilot` runs the host-trusted local pilot boundary: it verifies that the
repository is on the approved revision with a clean worktree, checks a
single-use challenge-bound approval against the live chain position, runs one
explicit evidence command with no shell, confirms the repository did not change
during capture, and appends the approval and evidence records.

`hma resolve` records one human resolution — a waiver grant, expiry, or
withdrawal, or a finding-disposition override. It replays the committed chain
first, so an already-granted waiver cannot be granted again, a withdrawal with
no active waiver is refused, a `BLOCK` finding cannot be waived or overridden,
and a challenge nonce already used in the run is rejected as a replay. Setting
`expected_predecessor_head` binds the operation to the exact chain tail the
human approved against.

`hma eval` calls exactly one deterministic evaluator on one host input
document and prints its result. It is strictly read-only: it takes no store
argument, appends no record, advances no stage, grants no approval, selects no
route, and dispatches nothing. A `LEGAL_PENDING_HUMAN_APPROVAL` result is a
statement that a proposal *may be shown to a human* — never that anything
advanced. Any resulting record is the host's to write. Input is decoded
strictly: an unknown field or a second document in the file is refused.

`hma show` prints the resolution projection of a committed chain — the
criteria, findings, active waiver scopes, and spent challenge nonces the chain
already says. It replays the chain and writes nothing. Waiver scopes and
nonces are sorted, so repeated invocations over an unchanged chain are
byte-identical.

`hma ci github` verifies that the GitHub Actions runner is on the approved
repository and revision. It writes an evidence record only when `--exec`
actually runs a command and produces an exit code and an output digest; it
never manufactures evidence from environment variables. Use `--verify-only` to
check the revision without recording anything. The host must bind the
approval's repository identity to the `owner/repo` slug for this adapter,
because that is the only identity the runner can derive independently.

## Reachable surface

Implemented and reachable from the CLI:

- host-trusted local pilot with approval binding and evidence capture
- human resolution: waivers and finding-disposition overrides
- GitHub Actions revision verification and CI evidence capture
- the chained JSONL run store
- read-only evaluation of every deterministic evaluator, via `hma eval`:
  stage transitions, approval invalidation, validator contract, route
  attestation, route coherence, and the versioned route-policy contract
- the resolution projection of a chain, via `hma show`

Deliberately **not** reachable from any command, and not planned:

- stage advancement, approval granting, route selection, or worker dispatch —
  a human performs every stage transition, and no HMA command advances state
  on its own

`hma eval` closes the gap where the binary could not express a stage
transition at all. It still does not perform one: it classifies a proposal and
prints the classification.

## Shape

- One standalone Go binary
- Standard-library-first implementation, no module dependencies
- Content-addressed JSON records and portable JSON Schema
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

## Deliberate non-goals for v1

- Autonomous coding orchestration
- A daemon, database, TUI, or plugin framework
- Provider-specific model SDKs
- Generic support for non-Git tasks
- Silent model fallback, automatic publication, or self-approval

## License

MIT. See [LICENSE](LICENSE).
