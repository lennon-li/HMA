# HMA Gate D D5b re-audit: findings disposition (hma-gate-d-audit-20261021-d5b3m)

**Verdict: FAIL (fix-required).**

The r1 fixes hold:
- F1–F6 and F8 are closed by running controls.
- A report can no longer create, suppress or clear an HMA BLOCK or WAIVABLE.
- BLOCK always stops the review gate and closure.

The new gate is unsound in one respect, N1 (high). A waiver is keyed only by finding id, and HMA re-raises its budget finding under the same id for every artifact. A waiver granted for a 4-file change therefore passed the review→verification gate for a 25-file change.

There are also three medium findings:
- **N2:** a waiver cannot take effect at the gate without a second independent review.
- **N3:** the dirty-target measurement hangs on an untracked symlink to a FIFO.
- **N4:** a human override can downgrade an HMA-measured WAIVABLE. This is latent.

## Scope, provenance, isolation

- **Target:** HEAD `7c555f67006e4ced6e76f060d74b3a100a76d363` plus the working tree.
  - The tracked diff sha256 is `8f13c4e7…a3e0`, which equals the expected value.
  - Untracked file sha256 values:
    - `cmd/hma/disposition.go`: `98b4309f…294b`
    - `cmd/hma/disposition_test.go`: `39ef6064…a5`
    - `internal/host/disposition_test.go`: `24de58f4…d6`
    - D5b handoff: `f6ab424e…5e`
  - Excluded: `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*`.
- **Freeze:** re-checked at the end. Status, diff hash, HEAD and untracked file hashes were unchanged.
- **PHASE1.json:** sha256 `12869d40de96b50ec36661d6c5e010290367c751bc2129ff9afa62c681f1e42e`. It was written before I read the diff, the D5b code or the D5b handoff.
- **Requirement sources:** Gate A v3 §4.1–4.3 and §6 row 4; architecture §5.2, §5.4, §7.1, §7.2, §9, §14 and §16; Gate C scope §0.1 and §3.
- **Isolation: partial.**
  - Fresh context. I saw no implementer chat.
  - The closure input is my own prior report (round r1), and the dispatch text describes the fixes.
- **Vendor family:** the implementer is Claude Sonnet 5.5 and I am Claude Opus 5.5. This is the same vendor family, with no provider separation.
- **Execution:**
  - All runs used a copy at `/tmp/hma-gate-d-20261021-d5b3m/src`, with GOCACHE and TMPDIR under the output directory. Go is 1.27.1 and git is 2.43.0.
  - No edits and no git writes were made in `/home/yeli/repos`.

## Requirement coverage

| Req | Result | Evidence |
|---|---|---|
| R1 Severity never sets disposition | pass | `TestDispositionFollowsSection9`; r1 A7 shapes |
| R2 BLOCK only for a kernel violation HMA reproduces; unwaivable | pass | A3: `HMA-KERNEL-no_shell` is kept, and the gate refuses. A4 and A8 give ADVISORY. See N5 (resolution divergence). |
| R3 WAIVABLE only for a measured excess | pass | A5 is now WAIVABLE (F6 closed). B7. See N6. |
| R4 A report cannot create, suppress or clear a BLOCK/WAIVABLE | pass | A1 import refused; B7 (prefix shapes); B8 (duplicate keys) |
| R5 A failed criterion check creates no finding | pass | Code inspection |
| R6 Snapshot semantics; stale budget clears | pass | B5. The full suite passes. |
| R7 Unresolved BLOCK/WAIVABLE stops the gate and closure | pass | A2 gate refuses. B11 closure through `runTransition` refuses. B6 ownership. B2 withdraw and expire. |
| R8 Waiver scoped to the exact finding and artifact | **fail** | N1, N4 |
| R9 Facts truthful and bounded | **fail** | N3, N5, N6 |
| R10 `no_shell` is the §7.1/§14 rule | pass | B9. A4: a missing tool is not a shell. See N5. |
| R11 No new false refusal | **fail** | N2. The existing flows are fine (full suite, B5). |
| R12 No regressions | pass | See Q5 |
| R13 No claim on D5c/D6 | pass | The handoff claims neither. |

## Prior findings (r1): closure status

| r1 | Status | Evidence |
|---|---|---|
| F1 id squat | closed | A1: an `HMA-` id refuses the import (`findings_import.go:73`). Established findings are always appended. |
| F2 approve clears BLOCK | closed | A3, round 2: `U1/HMA-KERNEL-no_shell: BLOCK`; the gate refuses `HOST_FINDINGS_UNRESOLVED`. |
| F3 no gate | closed, with defects | A2 gate refuses. B11 closure refuses. N1, N2, N4 and N7 are defects in the new gate. |
| F4 `evidence_current` | closed | A8 gives `U1/E1: ADVISORY`. |
| F5 missing tool | closed | A4 gives ADVISORY. The resolution divergence is N5. |
| F6 attributes | closed | A5 gives WAIVABLE. |
| F7 dirty content | closed for counting | The new dirty path has N3. |
| F8 assorted | (a), (b) closed; (c) unchanged and accepted; (d) documented | See N9. |

r1 controls that now "FAIL" for non-defect reasons:
- **A1:** it calls `t.Fatal` because the import is refused, which is the intended fix.
- **A2:** `outcomeGate` calls `requireIndependentCredit` alone. The new check lives in `runTransition`, and B11 covers it.
- **A3:** it asserts the old id `U1/F1`. The BLOCK is now `U1/HMA-KERNEL-no_shell`.
- **A7 combined:** a string-shaped `hma_ref` refuses the import.

## Findings (ordered by impact)

### N1 — HIGH, observed: a waiver survives a change of the measured artifact

- **Location:**
  - `cmd/hma/disposition.go:187`: the lookup is `ActiveWaivers[{finding, f.ID}]` only.
  - `internal/host/findings_import.go`, `AddEstablishedFindings`: the id is the stable `HMA-BUDGET-<name>`.
  - `engine/resolution.go:44-90`: nothing expires a waiver when the artifact changes.
- **Reproduction (`TestAuditB3WaiverSurvivesNewArtifact`):**
  1. A 4-file change against a 3-file budget, with an approving review, gives `U1/HMA-BUDGET-max_files: WAIVABLE`.
  2. `hma resolve` GRANT on that id.
  3. A new implementation commits 20 more files. A new packet and an approving review follow, then the findings import.
  4. Result: the same id is re-raised, and `gate()` **passes** for 25 files against the budget of 3.
- **Why it matters:**
  - §7.2: a waiver "expires only when its scoped artifacts or criteria change", and it is a human decision on a specific excess.
  - The dispatch rule is "WAIVABLE without an active waiver for that exact finding refuses". This finding is a new measurement on a new artifact.
- **Fix:**
  - For findings whose source id starts with `HMA-`, require the GRANT to bind `head` and `diff_digest`. `hma resolve` already supports these fields.
  - In `requireFindingsResolved`, compare that binding with the unit's current reviewed artifact. Treat a mismatch as unwaived.
- **Alternative:** include the review artifact digest in the established finding (an id suffix or a field), so that a new artifact is a new finding.
- **Verify:** B3 refuses. The B1 re-review on the same artifact, with a bound waiver, passes.
- **Effort:** small.

### N2 — MEDIUM, observed: a waiver cannot take effect at the review gate without a new independent review

- **Location:** `cmd/hma/transition.go:364-367`. `requireOnlyOwnSnapshotsSince` accepts only findings snapshots after the review. The `waiver_operation` record makes the review stale.
- **Reproduction (`TestAuditB1WaiverThenGate`):**
  - The unwaived gate refuses with `HOST_FINDINGS_UNRESOLVED`. After a GRANT, `requireFindingsResolved` passes, but `gate()` refuses with `HOST_CORE_BINDING_STALE`.
  - After a new packet, review and import, the gate passes. It passes only because of N1 (same id).
- **Test gap:** the shipped `TestWaivableNeedsHumanWaiver` checks `requireFindingsResolved`, never the gate after a waiver (`disposition_test.go:186-189`).
- **Fix:** allow GRANT `waiver_operation` records scoped to a finding of this unit after the review. Alternatively, make re-review the documented waiver path and add a gate-level test.
- **Verify:** B1 passes without re-review. The Gate C stale-binding tests still refuse other records.
- **Effort:** small.

### N3 — MEDIUM, observed: the dirty-target measurement follows untracked symlinks without bounds

- **Location:** `cmd/hma/disposition.go:146-159`, where `os.ReadFile` runs on each untracked path.
- **Reproduction (`TestAuditB10Measure`):**
  - An untracked symlink to a FIFO blocks `measureChange` for more than 5 s. In findings import this happens under the host run lock, and `boundedCtx` does not bound it.
  - An untracked symlink to a 5000-line file outside the repository counts 5002 lines. Committed, the same symlink counts 3.
  - Inferred, not run: a symlink to `/dev/zero` grows memory without bound.
- **Fix:**
  - `Lstat` each path.
  - Count a symlink by its link text (1 line), as git does.
  - Skip or refuse non-regular files.
  - Stream-count newlines with a byte cap, stopping once the budget is exceeded.
- **Verify:** both B10 cases return promptly, and the symlink counts 1 line.
- **Effort:** small.

### N4 — MEDIUM, inferred (latent): a human override downgrades an HMA-measured WAIVABLE

- **Location:** `internal/engine/resolution.go:240-256` refuses only BLOCK overrides.
- **Reproduction (`TestAuditB4OverrideDowngradesWaivable`):**
  - `hma resolve` with `finding_override` WAIVABLE→ADVISORY on `U1/HMA-BUDGET-max_files` is accepted.
  - `requireFindingsResolved(..., "")` (closure) then passes with no waiver.
- **Why the review gate is not bypassed:** the gate refuses `HOST_CORE_BINDING_STALE`, and a re-review re-raises the WAIVABLE.
- **Why it is latent at closure:** the core refuses verified outcomes today.
- **Architecture §9:** v3 §4.1 refuses human downgrade only for BLOCK. Architecture §5.2 resolves a WAIVABLE only by "waived, withdrawn, or corrected". §7.2 requires a waiver to stay visible and forbids `VERIFIED_SUCCESS` while one is active. An override of HMA's own measurement is none of these three. It also lets a measured excess close as `VERIFIED_SUCCESS`. I judge this not intended, but it is a design decision for Lennon.
- **Fix (recommended):** refuse a `FindingOverride` on ids whose source part starts with `HMA-`. Keep overrides for report-sourced findings.
- **Verify:** B4 is refused.
- **Effort:** small.

### N5 — LOW, observed: `no_shell` resolves the plan executable differently from the capture

- **Location:** `disposition.go:58-66` (`lookPathAbs`) uses `os.Stat` only. It also joins names that contain `/` with PATH directories. The capture's own resolution is `evidence/capture.go:100-112`: `exec.LookPath`, with relative paths refused.
- **Reproduction (`TestAuditB9ShellResolution`):** a non-executable `#!/bin/sh` file named `true`, earlier on PATH, makes plan command `true` an unwaivable BLOCK. The capture would run `/usr/bin/true`. Also, `./script` resolves through PATH directories.
- **Other B9 results:**
  - These match the capture policy (true): a symlink to bash, `#!/bin/sh`, `#!/usr/bin/env bash`, `busybox`, `env`, `timeout`, `find`.
  - These are false: `python3 -c`, a `#!/usr/bin/python3` script, `git -c alias=!sh`. This is the documented limit of inspection.
- **Fix:** export and reuse `resolveExecutable`. On a resolution error, apply `RejectShell` to the name only.
- **Effort:** small.

### N6 — LOW, observed: live config can also over-count, and runs a clean filter

- **Reproduction (`TestAuditB12CleanFilter`, dirty target):** `.git/info/attributes` contains `* filter=x`, and `.git/config` sets `filter.x.clean`.
  - With `seq 1 1000` as the filter, the count goes from 4 to 2001 lines.
  - With an emptying filter, it falls to 1.
  - The filter command runs.
- **The stated limit is wrong:** the handoff says this exposure "can only under-count". Over-counting gives a false WAIVABLE, never a BLOCK.
- **No effect:** `diff.relative`, `diff.algorithm` and `core.bigFileThreshold` change nothing.
- **Fix:** correct the stated limit, or measure dirty content without filters. One way: `hash-object --no-filters` into a temporary index, then diff the base against that tree.
- **Effort:** medium.

### N7 — LOW, observed (latent): the closure findings check allows `VERIFIED_SUCCESS` while a waiver is active

- **Reproduction:** in B11, after a GRANT, `VERIFIED_SUCCESS` passes the findings check and goes on to the credit check.
- **Why the model does not catch it:** validation checks only the terminal record's own `Waivers` (`model/validate.go:329-333`).
- **Fix:** in the closure branch, refuse `VERIFIED_SUCCESS` when `ActiveWaivers` is non-empty.
- **Effort:** small.

### N8 — LOW, inferred: a native (legacy-manual) review never computes HMA facts

- **Location:**
  - `findingFacts` runs only at findings import (`review.go:466-475`).
  - `EvaluateReview` accepts a native APPROVE when no packet exists in the run (`host/host.go:1018`).
- **Effect:** a budget excess or a shell plan command then passes review→verification. Credit still needs an imported review, so verified closure stays blocked.
- **Fix:** refuse native review in a run with a lease plan, or compute the facts at the gate.
- **Status:** not run, because no fixture combines a plan with native review.

### N9 — LOW, observed (opportunity): measurement definitions

These were measured in B10 and B9:
- A pure 60-line rename counts 2 files and 120 lines.
- A 1 MiB zero-byte file and a 1 MiB one-line file each count 1 line.
- Base == HEAD gives 0/0.
- A non-ancestor base gives the tree difference.
- An untracked file ignored only by `.git/info/exclude` is counted. This is consistent with `repostate/raw.go:102`.
- `find` and `timeout` as plan commands give an unwaivable BLOCK, consistent with the capture policy.

Document these definitions in the plan contract.

## Answers to the dispatch questions

1. **Is the gate sound?**
   - **BLOCK: sound.**
     - HMA re-raises it on every import (A3).
     - It stops the unit gate and closure (B11).
     - Another unit's BLOCK does not stop U1, but closure sees it (B6).
     - An unowned BLOCK stops U1 (B6).
   - **WAIVABLE: sound for these cases:**
     - an unwaived finding (A2, B11);
     - a withdrawn or expired waiver (B2);
     - a finding replaced under a different id (a new id needs a new waiver);
     - a finding replaced as BLOCK (BLOCK always refuses).
   - **WAIVABLE: unsound for the same id across artifacts (N1).**
   - **Override:** WAIVABLE→ADVISORY is accepted. It is blocked at the review gate (stale binding, then re-raise) but passes the closure form (N4). §5.2/§7.2 do not support it for HMA-measured findings, so Lennon should decide.
   - **Order of snapshots:**
     - An approving review that arrives after remediation clears a stale budget finding (B5).
     - A fix-required review in between keeps the stale WAIVABLE until then, which is acceptable.
   - **`unit=""` closure form:** it checks all findings and refuses through `runTransition` (B11).
2. **Can a report create, suppress or clear an HMA BLOCK/WAIVABLE?** No.
   - `HMA-`, `HMA-BUDGET-…` and `HMA-KERNEL-…` refuse the import.
   - A NUL suffix also refuses the import.
   - Case variants (`hma-`, `Hma-`), a leading space, a zero-width space and a full-width hyphen stay ordinary ids. They never displace the HMA finding. An id becomes WAIVABLE only when the cited budget is truly exceeded (B7).
   - Duplicate keys: the last value wins. An `HMA-` id as the last value refuses the import. The established findings survive every variant (B8).
   - `hma_ref` shapes stay ADVISORY or refuse the import (A7).
   - Optional: reserve the prefix case-insensitively, so ids such as `hma-BUDGET-…` cannot mislead a human reader.
3. **False refusals:**
   - No existing flow regressed (1056 PASS). An ordinary advisory-only approving review passes.
   - A stale `HMA-BUDGET` clears after remediation and an approving review (B5).
   - New refusal sources:
     - N2: the waiver path needs a re-review.
     - N5: host PATH artifacts.
     - N3 and N6: over-counts.
4. **Are the facts truthful?**
   - `RejectShell`: see N5 and B9.
   - Measurement: see N3, N6 and N9. Renames, base == HEAD, a non-ancestor base, binaries and huge files were measured. Merges and submodules were checked by code reading only.
   - Live config: it can over-count as well as under-count (N6). It cannot forge a BLOCK.
5. **Regressions:**
   - `go vet ./...` is clean.
   - `go test -count=1 ./...`: all packages ok, 1056 PASS, 0 FAIL.
   - Skipped: `TestGateBR2HostChild` (child helper) and `TestGateBR2HostPowerLossImage` (B7 marked unreachable).
   - `go test -race` passes on `./cmd/hma ./internal/evidence ./internal/repostate ./internal/host`.
   - The bubblewrap sandbox tests ran and passed.

## Controls (copy only)

- `src/cmd/hma/zz_audit_d5b3_test.go`, sha256 `5975be98…29c7`: controls B1–B12.
- `src/cmd/hma/zz_audit_d5b_r1_test.go`, sha256 `98d79058…16f8`: the r1 controls A1–A9, re-run.

Logs: `vet.log`, `test.log`, `verbose.log`, `race.log`, `neg_all.log`, `neg_b*.log`, `neg_a.log`.

## Limitations

- The closure path was exercised only up to the findings check, because the core refuses verified outcomes.
- N8 was not run.
- The `/dev/zero` case is inferred.
- Merges and submodules were checked by code reading only.
- The fixes are proposed and not validated.
- The vendor family is the same as the implementer's, and isolation is partial.

## Next action

Ming fixes N1 (bind HMA-finding waivers to the reviewed artifact), N2 and N3, and adds B1, B3 and B10 (fifo, symlink) as permanent tests. Lennon decides N4 (whether a WAIVABLE may be overridden). Then dispatch an independent re-audit, preferably from another provider family.
