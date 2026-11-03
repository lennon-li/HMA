# `hmarun` usage guide

`cmd/hmarun` is a thin wrapper around the real `hma` binary and the real
`agent-dispatch` tool. It never reimplements a safety check; it only
assembles correct input for the existing ones and walks the 20-step
sequence from `GROUNDING` to a terminal outcome, one real human checkpoint
at a time. It is resumable: a crashed or interrupted run picks back up from
the first undone step on the next invocation, against the same `--store`
and `--run_id`.

## 1. `hmarun init` — build the contract and plan

You need a `hma.acceptance-contract/v1` and `hma.plan/v1` pair before
`hmarun run` can do anything. `hmarun init` builds both from either an
interactive Q&A or a non-interactive answers file.

```
hmarun init --out-dir ./mytask
```

prompts for: intent, non-goals, risk, one or more acceptance criteria (each
a statement, a threshold, and a verification command — an explicit
executable + argv, never a shell string), and one or more plan units (each
an `allowed_files`/`forbidden_files` pair, and which criteria it advances).
It writes `mytask/contract.json` and `mytask/plan.json`, self-validated by
round-tripping them through `internal/model`'s own parsers before writing
anything.

For a scripted or repeated setup, write an answers file instead (see
`cmd/hmarun/answers.go`'s doc comment for the full schema and defaults) and
pass `--answers path/to/answers.json`.

**Design the plan unit's scope deliberately.** If a criterion requires
touching a file the unit's `forbidden_files` excludes, that is a real,
designed tension — the unit's own `escalation_conditions` text says so
explicitly, and a capable implementer will self-limit and report the gap
rather than violate scope (confirmed in the 2026-11-03 dogfood: see
`docs/handoffs/evidence/mvp/dogfood-run-2026-11-03.md`). That is a
legitimate way to produce a genuine, evidenced `PENDING` criterion rather
than a fabricated one.

## 2. `hmarun run` — walk the sequence

```
hmarun run --store ./mytask-store --repo /path/to/target/repo \
  --run_id my-task-001 --contract ./mytask/contract.json \
  --plan ./mytask/plan.json [--unit U1]
```

- `--store` is this run's own HMA store (core + host records + artifacts).
  Keep it separate per run.
- `--repo` is the repository the work actually happens in. It is never
  `hmarun`'s own source tree.
- Real dispatch is the default. `agent-dispatch` must be on `PATH`, or pass
  `--agent-dispatch-bin <path>`. Use `--stub-dispatch` only for a dry run
  with no real worker/reviewer (useful to prove the sequence before
  spending real dispatch budget — see "Dry-run first" below).
- `go build -o <path> ./cmd/hma` first, and pass `--hma-bin <path>` or put
  it on `PATH`.

### Approving checkpoints

At every real stage transition, `hmarun run` prints a short summary (stage,
target, route, risk) and asks:

```
Type your name to approve:
```

Whatever you type becomes the real `Approver` field on that transition's
approval record — this is the actual human-gating mechanism, not a
cosmetic prompt. For a supervised batch of checkpoints you have already
decided to approve, it is reasonable to pipe the same approver string to
all of them (e.g. `yes "Your Name (reason, date)" | hmarun run ...`) — but
see the one exception below.

**One prompt behaves differently.** The very first thing `hmarun run` asks,
binding the repository identity, shows a default in brackets and accepts an
empty answer to keep it:

```
Repository identity to bind (this cannot change later) [/path/to/repo]:
```

Press Enter (or don't feed it a line) to keep the repository path as the
identity. If you pipe the same approver string to *every* prompt
indiscriminately, it will silently override this one too, binding the
approver's name as the "repository identity" instead of the path — harmless
(it is a label, not used for file resolution) but worth avoiding.

### What a real run actually asks you to approve, in order

1. Accept the acceptance contract (digest shown).
2. Accept the plan (digest shown).
3. Approve the unit's route (implementer identity/provider/model shown).
4. Delegate the unit's implementation to that route (vs. direct cost).
5. Present the implementation for review, once the real worker has
   produced an in-scope commit.
6. Present the reviewed implementation for verification, once the real
   reviewer's report has been imported.

Steps 1-4 happen before any real dispatch; 5-6 happen after the real worker
and reviewer, respectively, have actually run.

### Inspecting status

- `hma show --store <store> --run_id <id>` — the core chain's current
  stage/outcome.
- `hma host show --store <store> --run_id <id>` — the full host record
  history: preflights, dispatches, the review packet, the imported review,
  reconciliation.
- Every real `agent-dispatch` run's receipt and report live under
  `<store>/agent-dispatch-state[-unreconciled]/<repo-key>/runs/<run-id>/`.

### Recovery behavior

- **No commit from the real implementer:** `hmarun run` retries once,
  automatically, carrying the failed attempt's command and output forward
  as context for the retry. If the retry also produces no commit, it stops
  with a named, actionable error (which run id, which receipt, what a human
  should do next) — never a bare block.
- **An out-of-scope commit:** if the real worker commits something outside
  the unit's `allowed_files` or inside `forbidden_files`, `hmarun run`
  detects it from the actual `git diff` (never from the worker's own
  self-report) and stops immediately, naming the offending path and the
  unit's approved scope. It does not revert or retry automatically — a
  human decides whether to revert, widen scope, or re-dispatch.
- **A dead reviewer dispatch** (crashed before producing a report): the next
  `hmarun run` invocation mints a fresh review packet with a fresh run id
  automatically; it never retries the same `--run-id` (the real
  `agent-run-ledger` can never reuse one).
- **A permission wall your environment imposes on the real dispatch itself**
  (for example: this machine's global Claude Code settings currently put
  `git commit`/`git push` under an "ask" permission a headless dispatch can
  never answer) is not something `hmarun` can route around, and it
  shouldn't try to. It will retry once, then stop with the real denial
  message. A human then decides: relax the policy (a real tradeoff, not
  `hmarun`'s call), or complete the one blocked step by hand.

### Dry-run first

Before spending real dispatch budget, prove the sequence against a scratch
store with `--stub-dispatch --auto-approve`: every checkpoint auto-approves
with a clearly-labeled synthetic approver, and the worker/reviewer steps use
local stand-ins instead of real dispatch. This is fast, free, and catches
contract/plan wiring mistakes before they cost anything real.

## 3. What `PARTIAL` means here

If a criterion genuinely cannot be advanced within the approved unit's scope
(the deliberate-tension case above), the run can still close `PARTIAL`: the
criteria that did pass are credited for real, from the real imported
review, and the ones that didn't stay honestly `PENDING` — not silently
dropped, not fabricated as passed. `VERIFIED_SUCCESS`/`WITH_WAIVERS` closure
is out of scope for this MVP; `PARTIAL` is the real, evidenced ceiling.
