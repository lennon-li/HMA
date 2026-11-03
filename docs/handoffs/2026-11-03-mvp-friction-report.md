# MVP friction report (2026-11-03)

Scope: `cmd/hmarun` (built this session) plus one real, non-synthetic
dogfood run against a throwaway repo, real `agent-dispatch` calls (ming =
`claude-sonnet-5`, phil = `gpt-5.3-codex`), and the real `hma` binary. Full
evidence: `docs/handoffs/evidence/mvp/dogfood-run-2026-11-03.md`. This
report is the honest accounting the plan asked for: what the real run
actually cost in approvals and friction, what turned out to be redundant,
and what a later policy revision could safely consolidate — not a success
narrative.

## Approval count (the real run, before it stopped short)

7 distinct `Type your name to approve` checkpoints were answered, in order:
accept-criteria, planning, route-selection, route-approval, decision
(delegate to ming), implementation-review (present for review), and one
attempt at verification (present for verification) that then failed inside
`hma transition` itself after the approval was already recorded. One
additional prompt (binding repository identity) is not counted as an
"approval" by `hmarun`'s own summary counter, but behaves enough like one
in practice that it is worth counting here too — see below.

None of these 7 were redundant in the sense of asking the same question
twice: each corresponds to a distinct real stage transition with its own
digest, and `hmarun run`'s resumability correctly skipped re-asking any of
them across three separate process restarts (crash-equivalent resumes).
**The repetition that did happen was not in the approval count — it was in
real dispatch cost**, below.

## What was genuinely redundant: repeated real dispatch after an
unrecoverable, non-code failure

`stepWorkerDispatch`'s one bounded retry assumes the failure it is retrying
past is something a second attempt might fix (a flaky run, a fixable bug).
Across this session's dogfood work, the real implementer's commit was
denied by the same global permission policy **six times** (two attempts ×
three separate `hmarun run` invocations before the actual blocker was
understood) — every one of those dispatches cost real API time and money,
and every one produced the identical, predictable denial. A retry cannot
fix a global permission policy. This is a real design gap worth closing:
**improvement**, not blocker (below).

## What a later policy revision could safely consolidate

- Steps 1-4 (accept-criteria through decision) happen before any real
  dispatch and never depend on anything dispatch-related. For a run whose
  contract/plan were just produced by `hmarun init` in the same sitting (so
  there is no meaningful time gap for a human to reconsider), these four
  could reasonably be a single combined confirmation without weakening
  anything — the same human, the same sitting, four digests shown together
  instead of one at a time. This is explicitly **not implemented now** (per
  the plan's own deferral), but is a legitimate, low-risk consolidation
  candidate for later.
- Steps 5 and 6 (present-for-review, present-for-verification) should stay
  separate: they bracket the one thing in this whole sequence a human
  cannot cheaply redo if wrong — the real independent reviewer's actual
  judgment.

## Findings, ranked by how much they would block real use

### 1. Headless `agent-dispatch` workers can never self-commit here (structural, not an `hmarun`/HMA bug)

`~/.claude/settings.json` lists `Bash(git commit:*)`/`Bash(git push:*)`
under `"ask"`, globally. A headless dispatch has no one to answer an "ask"
prompt, so every real worker's own `git commit` is denied, every time,
deterministically — confirmed six times across this dogfood's real
dispatches. A project-local `.claude/settings.local.json` `allow` override
was tried and does **not** work (a local `allow` cannot override a
user-level `ask` rule), and introduces a second problem: it leaves an
untracked file in the worktree that HMA's own stricter dirty-worktree check
(which deliberately ignores host-local ignore sources) correctly flags.

This is the single most consequential finding of the whole exercise: as
currently configured, `cmd/hmarun`'s real-dispatch worker path can produce
a correct, tested implementation but can never deliver it without a human
completing the commit by hand. That human step is not currently a
first-class part of the flow — it happened this session by me stepping in
as the acting orchestrator, which worked, but is not documented anywhere as
an expected, designed path.

**Not fixed this session** (deliberately: it is a real policy tradeoff
outside `cmd/hmarun`'s remit, not a code bug). See TODO.md.

### 2. `HOST_UNAUTHORIZED_DISPATCH` on every real-dispatch run (hmarun bug — fixed, reviewed)

Every real preflight call and the real implementer's own dispatch attempts
shared the exact same ledger directory the reviewer's real `review-packet`
call declares as a reconciled namespace. `hma host reconcile`'s own
correct rule — every run physically present in a declared namespace needs a
matching reservation or packet — then flagged all of them as unauthorized,
blocking `stepVerification` on every real-dispatch run, independent of
finding 1. Never caught before because no earlier test or smoke run drove
the full pipeline through `reconcile` against real dispatch; this dogfood
run was the first.

**Fixed**: `cmd/hmarun/run.go` now routes preflight and the real
implementer dispatch through a separate, never-declared
`unreconciledDispatchStateDir()`; only the real reviewer dispatch still
uses the directory `ledgerDir()`/`review-packet` depend on.
Regression-tested (`TestStepReviewerDispatchAcceptedByReviewImport`,
proven via revert → fail → restore → pass). Independently reviewed:
`docs/handoffs/evidence/mvp/audit-hmarun-ledger-isolation-fix-phil.md`
(PASS).

### 3. The repository-identity prompt doesn't behave like the others (operational, minor)

`resolveRepositoryIdentity` accepts an empty answer to keep its default
(the repo path); every other checkpoint (`confirm()`) requires a typed,
non-empty answer that becomes the approver. Piping one approver string to
every prompt in a supervised batch — the natural operating pattern — silently
overrides this one default too. Harmless in practice (it is a label, not
used for file resolution), but it is the one inconsistent prompt in an
otherwise uniform sequence, and worth either fixing (require an explicit
`--yes` style flag to skip it, or make it also default-preserving under
`--auto-approve`-style batching) or at minimum documenting distinctly (done
now, in the usage guide).

### 4. The "recoverable failure" demonstration didn't fire the way it was designed to

The dogfood task was deliberately shaped (naive byte-reversal vs. UTF-8
runes) to give a real implementer a plausible chance of an initial,
genuinely-fixable failure. The real `claude-sonnet-5` dispatch got it right
immediately every time, so the retry-with-failure-context path was
exercised only by the permission-wall failures (finding 1), which no retry
can actually fix. The retry mechanism itself is confirmed correct (one
bounded attempt, failure context carried forward, clean exhaustion
message) — this is not a defect, just a note that the dogfood's own bait
didn't catch a capable model, and the retry logic's only live exercise this
session was against a failure class it cannot help with.

## What worked without any friction

- The real worker correctly self-limited at the designed `CHANGELOG.md`
  scope conflict, citing the plan's own `escalation_conditions` text back —
  no kernel enforcement was even needed to catch a violation, because the
  prompt text alone was sufficient.
- The real reviewer (`gpt-5.3-codex`, genuinely cross-family from
  `claude-sonnet-5`) produced a fully schema-conformant `AUDIT.json` on its
  first try, including the `fix` object for its one `gap` finding — direct
  validation of `reviewTaskText()`'s schema documentation (fixed earlier
  this session after an independent review caught the gap) against a real,
  previously-unseen model.
- `stepReviewImport`, `stepFindingsImport`, and the reconciliation step's
  own logic (once given a correctly-isolated namespace) worked exactly as
  designed: C1 credited PASSED from real evidence, C2 left genuinely
  PENDING from a real, evidenced gap finding — not fabricated, not
  silently dropped.
- Resumability: three separate `hmarun run` invocations against the same
  store, interrupted by real failures and a code fix in between, never
  duplicated a record or re-asked an already-answered approval.

## Next, if resourced

A second, fresh real dogfood run (new throwaway repo, new run id) would
exercise the `HOST_UNAUTHORIZED_DISPATCH` fix against a real dispatch end
to end, reaching an actual terminal `PARTIAL` record. It was not done this
session (Lennon's call, given the real dispatch cost already spent and that
the fix is independently reviewed and regression-tested against the exact
mechanism). It would still hit finding 1 (the commit-permission wall)
identically, needing the same one manual-commit step, unless that is
addressed first.
