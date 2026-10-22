# HMA Gate D D5b audit: findings disposition (hma-gate-d-audit-20261021-d5b1m)

**Verdict: FAIL (fix-required).**

D5b maps a report's `hma_ref` against HMA-computed facts, and report shapes cannot forge a BLOCK. Two parts fail:
- Report text can still suppress an HMA-measured WAIVABLE, or clear an HMA-reproduced BLOCK.
- No gate acts on an imported BLOCK or WAIVABLE, so architecture §5.2/§7.2/§9 is not met.

The handoff also states that `evidence_current` is not registered. The code registers it, and the rule fires.

## Scope, provenance, isolation

- **Target:** HEAD `7c555f67006e4ced6e76f060d74b3a100a76d363` plus the uncommitted working tree. The tracked diff sha256 is `0e80b3fc…c46ff`, which equals the expected value.
- **Untracked file sha256 values:**
  - `cmd/hma/disposition.go`: `85312652…c8c5`
  - `cmd/hma/disposition_test.go`: `f6234337…c65d`
  - `internal/host/disposition_test.go`: `08d3e3fe…57f9`
  - D5b handoff: `4f32ab41…be`
- **Freeze:** re-checked at the end of the audit. Status, diff hash and file hashes were unchanged.
- **Excluded:** `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*`.
- **Requirements:** derived from Gate A v3 §4.1–4.3 and §6 row 4; architecture §5.2, §7, §9 and §14 (lines 670–671); Gate C scope §0.1 and §3; the D4 handoff.
- **PHASE1.json:** sha256 `28ad4374f245a30c18b330c7711eedb5530b1a44826c77c919a2fc7bd1a18bf4`. It was written before I read the diff or the D5b handoff.
- **Isolation:**
  - Fresh context. I saw no implementer chat.
  - The coordinator's prompt framed the questions.
  - I read the D5b handoff only after PHASE1.json existed and controls A1–A7 were written. The handoff prompted control A8, which tests one of its claims.
- **Vendor family:** the implementer is Claude Sonnet 5.5 and I am Claude Opus 5.5. This is the same vendor family. The context is isolated, but there is no provider separation.
- **Execution:** all tests ran on a copy at `/tmp/hma-gate-d-20261021-d5b1m/src`, with GOCACHE and TMPDIR set under that directory. Go is 1.27.1. No edits or git writes were made in `/home/yeli/repos`.

## Requirement coverage

| Req | Result | Evidence |
|---|---|---|
| R1 Severity never sets disposition | pass | `MapAuditFindings` reads no severity field. Exit test and A7 (per-shape probe) confirm this. |
| R2 BLOCK only for a violation HMA reproduces; unwaivable | **fail** | F2: an approving review drops a BLOCK whose predicate still holds. F4: `evidence_current` fires on a routine repair. F5: `no_shell` fires when a tool is missing from PATH. |
| R3 WAIVABLE only for a measured budget excess | **fail** | F1: report text squats the measured id. F6: `.gitattributes` hides lines from the measurement. |
| R4 Everything else ADVISORY; no invented blocker | pass | A7 shapes stay ADVISORY: case variants, extra keys, a key in wrong case, a full-width id, a forged `UNAUTHORIZED_DISPATCH:` id. A string-shaped `hma_ref` refuses the whole import (fail closed). |
| R5 Failed criterion check creates no finding | pass | Nothing implements a criterion link (code inspection). |
| R6 Snapshot keeps earlier BLOCK on non-approve; unowned UD survives | pass (unchanged) | `SnapshotFindings` is unchanged (`findings_import.go:133-158`). An approve still replaces unit findings; see F2. |
| R7 Unresolved BLOCK/WAIVABLE blocks closure; WAIVABLE needs a human waiver | **fail** | F3, from runtime control A2. |
| R8 Measurements bound to the reviewed target | partial | `requireLiveTarget` runs before `findingFacts` (`review.go:458-461`, then `:466`). Dirty content is not measured (F7). Attribute and config influence: F6. |
| R9 Exit test `TestDispositionFollowsSection9` | pass | Passes with and without `-race`. |
| R10 No claim on D5c/D6 | pass | The handoff and code claim neither. |
| R11 No regressions | pass | `go vet ./...` is clean. `go test -count=1 ./...` passes: 1052 PASS, 0 FAIL. `-race` on the four packages passes. Details below. |

## Findings (ordered by impact)

### F1 — HIGH, observed: report id squats `HMA-BUDGET-<name>` and suppresses the measured WAIVABLE

- **Location:** `internal/host/findings_import.go:97-109`. `AddMeasuredFindings` skips its WAIVABLE when the imported list already has the id (`!have[id]`, line 104). `MapAuditFindings` does not reserve the `HMA-` prefix (line 73).
- **Reproduction (A1, `TestAuditD5bMeasuredIDSquat`):** a 4-file change against a 3-file budget, plus the finding `{"id":"HMA-BUDGET-max_files"}` with no `hma_ref`. Result: `U1/HMA-BUDGET-max_files: ADVISORY`, and no WAIVABLE is recorded.
- **The shipped test encodes this behaviour:** `internal/host/disposition_test.go:47-51` expects an exceeded `max_diff_lines` to stay ADVISORY under the squatted id.
- **Fix:**
  - Refuse imported ids that start with `HMA-` in `MapAuditFindings`, with `HOST_FINDINGS_IMPORT_INVALID`.
  - In `AddMeasuredFindings`, raise the disposition of any same-id entry to WAIVABLE, or always append.
  - Invert the host test.
- **Verify:** A1 shows WAIVABLE or the import is refused. The A6 control still shows WAIVABLE.
- **Alternative:** namespace HMA findings outside the unit prefix (for example `HMA/U1/budget/max_files`, owned by the unit). This is larger, because ids then need another separator rule.
- **Effort:** small.

### F2 — HIGH, observed: an approving review clears a unit BLOCK while HMA's predicate still holds

- **Location:**
  - `findings_import.go:133-158`: on approve, the snapshot replaces the unit's findings.
  - `cmd/hma/disposition.go:51-57`: the `no_shell` fact is computed on every import but becomes a finding only if an auditor cites it.
- **Reproduction (A3, `TestAuditD5bApproveClearsHeldBlock`):**
  1. The plan command is `/bin/sh -c true`. A fix-required review cites `no_shell`, giving `U1/F1: BLOCK`.
  2. A new packet gets a `pass` report with no findings. The core findings become `map[]`.
  3. `gate()` (reconcile, then IMPLEMENTATION_REVIEW→VERIFICATION) **passes**.
- **Why it matters:** an unwaivable kernel BLOCK that HMA reproduced is removed by the auditor leaving it out. A human downgrade of the same BLOCK is refused (`resolution.go:218,253`). This breaks "human downgrade refused" and "a validator cannot silently create (or remove) a blocker".
- **Fix:**
  - Mirror `AddMeasuredFindings` for kernel rules: for every registered rule whose predicate holds, add an HMA-owned `HMA-KERNEL-<rule>` BLOCK regardless of citation.
  - In review import, escalate (never approve) while any registered predicate holds.
  - Optionally, have `SnapshotFindings` keep a prior BLOCK on approve when its rule still holds.
- **Verify:** in A3, round 2 keeps a BLOCK and `gate()` refuses. A pass with no rule holding still clears (control: `TestGateCAuditFailureRemediationReverification`).
- **Trade-off:** `authority` would then add a second BLOCK beside the unowned `UNAUTHORIZED_DISPATCH` BLOCK. That is harmless, but it is redundant.
- **Effort:** small to medium.

### F3 — HIGH, observed: BLOCK and WAIVABLE dispositions reach no gate

- **Location:**
  - `cmd/hma/transition.go:37-116`. The review→verification gate checks for an APPROVE review, a snapshot and the artifact. It does not check finding dispositions. It refuses BLOCKs only through `HasUnauthorizedDispatch` (`reconcile.go:481`).
  - Closure (`transition.go:299-307` `requireIndependentCredit`; `model/validate.go:329-347`) checks criteria and waivers, not findings.
  - The engine (`engine/transitions.go`) never reads findings.
- **Reproduction (A2, `TestAuditD5bWaivableReachesGate`):**
  1. A 4-file change gets an approving review, and the findings import records `U1/HMA-BUDGET-max_files: WAIVABLE`.
  2. `gate()` passes, with no waiver.
  3. `outcomeGate()` (the verified-outcome credit check) also passes.
- **BLOCK path:** a BLOCK reaches the gate only through a non-approving verdict, which the gate already refuses for its own reason. With F2, an approving review removes it.
- **Requirement:** architecture §5.2 (lines 113–116): "an unresolved `WAIVABLE` finding blocks closure until it is waived". Lines 670–671: "no unresolved `BLOCK` or `WAIVABLE` finding remains".
- **Scope note:** this gap may predate D5b. D5b is the first change that produces imported BLOCK and WAIVABLE findings, and the D5b handoff neither claims nor schedules enforcement.
- **Fix:**
  - In `requireHostTransitionGateIn` (review→verification), and in the closure path for `VERIFIED_SUCCESS`/`VERIFIED_WITH_WAIVERS`, refuse while the current core findings hold any BLOCK.
  - Refuse while they hold any WAIVABLE without an active `ScopeFinding` waiver (`ProjectResolutionState(core).ActiveWaivers`). Use a new reason such as `HOST_UNRESOLVED_FINDING`.
  - Optionally, have the model reject `VERIFIED_SUCCESS` while any WAIVABLE exists.
- **Verify:** A2 refuses. After a granted waiver on that finding, the gate passes, and closure is allowed only as `VERIFIED_WITH_WAIVERS`. BLOCK never passes.
- **Alternative:** enforce only at closure, as architecture's wording requires, and leave review→verification unchanged. This is smaller but lets a run reach verification with known blockers.
- **Effort:** medium.

### F4 — MEDIUM, observed: `evidence_current` is registered, contrary to the handoff, and fires on a routine repair

- **Location:** `findings_import.go:30` (`KernelRules` includes `evidence_current`) and `disposition.go:58-72`.
- **Handoff claim:** "only `authority` and `no_shell` … `evidence_current` … stays unregistered", and "cannot hold at import today".
- **Reproduction (A8, `TestAuditD5bEvidenceCurrentReachable`):**
  1. Credit review 1.
  2. Create a new implementation and commit a repair.
  3. Issue a new packet. A fix-required report cites `evidence_current`.
  4. Result: `U1/E1: BLOCK`.
- **Why the handoff argument fails:** the predicate uses the unit's latest credit from any earlier implementation, and the live-target check binds only the current packet.
- **Why this is not a kernel violation:** a stale credit is already void at the gate (D5a `TestCreditVoidAfterNewImplementation`). §4.1 says this case is "not credited", not a new blocker.
- **Fix:** remove `evidence_current` from `KernelRules`, as the handoff states. Optionally, delete its computation.
- **Verify:** A8 gives ADVISORY.
- **Effort:** small.

### F5 — MEDIUM, observed: the `no_shell` predicate holds for non-shell, environment-dependent reasons

- **Location:** `disposition.go:53-57` uses `unsafeLoopArgv` (`loop.go:463`). That function returns true when `exec.LookPath` fails, and for curl, ssh, gh pr, disallowed git forms, and empty arguments.
- **Reproduction (A4, `TestAuditD5bNoShellMissingTool`):** the plan executable `hma-audit-no-such-tool` with a cited `no_shell` gives an unwaivable `U1/F1: BLOCK`.
- **Why it matters:** a missing tool on the host's PATH is not the §7.1 rule "commands never through a shell". Another named class may apply (external effect), but not this rule.
- **Fix:**
  - Use `evidence.RejectShell(argv0, canonical, args) != ""` for `no_shell`.
  - On a resolution failure, refuse the import (unknown), or do not hold the rule.
  - If the network and publication cases are wanted, register them as a separate rule (`external_effect`).
- **Verify:** A4 gives ADVISORY or a refused import. `/bin/sh -c` still gives BLOCK.
- **Effort:** small.

### F6 — MEDIUM, observed: repository attributes hide lines from the budget measurement

- **Location:** `disposition.go:79`. `git diff --numstat` honours `-diff`/`binary` attributes from the worktree `.gitattributes`, `.git/info/attributes` and `core.attributesFile`, and also `core.bigFileThreshold` from the live config. Binary rows count 0 lines (lines 90–95).
- **Reproduction (A5, `TestAuditD5bGitattributesHidesLines`):** a committed `.gitattributes` with `* -diff` plus a 500-line file, against a 100-line budget, gives F1 ADVISORY and no `HMA-BUDGET-max_diff_lines`. Control A6, the same change without the attribute, gives WAIVABLE.
- **Who controls the input:** the implementing worker controls the committed `.gitattributes`. It is visible in the reviewed diff but not in the measurement.
- **Same class as the D5a rounds:** `findingFacts` reads under the live `.git/config`. `observationConfig` does not neutralise attributes or `bigFileThreshold`. The impact here is limited to under-counting (a missing WAIVABLE). No BLOCK can be forged.
- **Fix:** add `--text` to the numstat call, or neutralise attributes with `-c core.attributesFile=/dev/null` plus `--attr-source=<empty tree>` (git ≥ 2.40). Also add `-c core.bigFileThreshold=<large>`. Count lines of true binaries separately, for example by blob size, if wanted.
- **Verify:** A5 gives WAIVABLE. A6 is unchanged.
- **Effort:** small.

### F7 — LOW, inferred: dirty content of a dirty-approved review target is not measured

- **Location:** `disposition.go:75-79` measures `base..HEAD` only. The handoff discloses this. A review bound to a worktree digest can exceed the budget in uncommitted files without a WAIVABLE.
- **Fix:** when `p.WorktreeDigest != ""`, measure `git diff --numstat <base>` against the worktree, plus untracked paths from the snapshot.
- **Effort:** small.

### F8 — LOW, observed: assorted

- (a) `review_import.go:85-87`: the comment says "HMA cannot yet reproduce a cited kernel rule", which is now stale. The escalate behaviour itself is correct (control A9: a citing pass escalates, its BLOCK stands, and the gate refuses).
- (b) `authority` holds run-wide. Any finding that cites it becomes BLOCK while any unit has an unauthorized dispatch. This is redundant with the existing gate.
- (c) `currentPlanUnit` (`disposition.go:17-38`) fails closed for a missing or unreadable plan or a missing unit entry, which is acceptable. As a result, findings import is impossible for any run without a lease plan, which is a behaviour change from Gate C.
- (d) A rename counts as 2 files plus full line counts (`--no-renames`). A merge is measured as the base→HEAD tree diff. A submodule pointer counts as 1 file and 2 lines. These are acceptable, but undocumented.

## Answers to the dispatch questions

1. **Can report content create a disposition HMA did not establish?**
   - No forged BLOCK or WAIVABLE: the A7 shapes, case variants, unicode, extra keys, `UNAUTHORIZED_DISPATCH:` ids (namespaced to `U1/…`, so `HasPrefix` misses them), duplicate ids (refused) and wrong-shape `hma_ref` (refused) cannot.
   - Report content can **suppress** an HMA-established result (F1) and **remove** one (F2).
   - A pass carrying an `hma_ref` is escalated (A9).
2. **Are the facts truthful?**
   - `authority`: truthful, but run-wide.
   - `no_shell`: over-broad (F5).
   - `evidence_current`: not a kernel fact (F4).
   - Budgets: can be gamed through attributes and config (F6). Dirty content is omitted (F7).
   - The live-target check precedes `findingFacts` (confirmed at `review.go:458`). The plan is the current generation's (last lease issuer). A missing plan, unit or unreadable plan is refused.
   - `findingFacts` is exposed to the live-config class. It matters only as under-counting.
3. **Can a later import weaken or clear an earlier finding?**
   - `SnapshotFindings` is unchanged. A non-approving import keeps BLOCK and WAIVABLE (A9; Gate C tests).
   - An approving review clears both. For WAIVABLE, `AddMeasuredFindings` re-raises a still-exceeded budget, unless the id is squatted (F1). For BLOCK nothing re-raises it (F2).
4. **Does the disposition reach the gate?** No (F3, verified by running A2 and A3). The only enforced BLOCK class is `UNAUTHORIZED_DISPATCH:` (D4).
5. **Regressions?**
   - `go vet ./...` is clean.
   - `go test -count=1 ./...`: all packages ok, 1052 PASS, 0 FAIL.
   - Skipped tests: `TestGateBR2HostChild` (child helper only) and `TestGateBR2HostPowerLossImage` (B7, marked unreachable).
   - `go test -race` on `./cmd/hma ./internal/evidence ./internal/repostate ./internal/host` passes.
   - Bubblewrap starts in this runtime (`/usr/bin/bwrap`). The sandbox tests ran and passed (`TestSandboxShowsOnlyTheCopy`, `TestSandboxRefusesRepositoryInsideSystemDirectory`).
   - My controls were also run under `-race`.

## Negative controls (auditor-written, copy only)

The controls are in `src/cmd/hma/zz_audit_d5b_test.go` (sha256 `98d79058…16f8`).

| Control | Result |
|---|---|
| A1 | FAIL, confirms F1 |
| A2 | FAIL, confirms F3 |
| A3 | FAIL, confirms F2 |
| A4 | FAIL, confirms F5 |
| A5 | FAIL, confirms F6 |
| A6 | PASS (control for A5) |
| A7 shapes (`TestAuditD5bShapesEach`) | PASS |
| A7 combined (`TestAuditD5bReportShapes`) | FAIL, not a defect: the string-shaped `hma_ref` refuses the whole import |
| A8 | FAIL, confirms F4 |
| A9 | PASS |

Logs: `neg_race.log`, `neg.log`, `test.log`, `race.log`, `vet.log`, `verbose.log`.

## Limitations

- There is no end-to-end run of `authority` producing a BLOCK. That path was checked by code reading and the host unit test.
- Renames, merges and submodules were checked by code reading, not runtime tests.
- The proposed fixes are not validated.
- The vendor family is the same as the implementer's.

## Next action

Ming fixes F1 to F4 as the minimum set for D5b exit. F3 can instead be scheduled explicitly as a named Gate D exit item with Lennon's approval. Ming should also fix F5 and F6, add A1 to A5 and A8 as permanent tests, and re-dispatch an independent re-audit.
