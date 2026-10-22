# HMA Gate D D5b re-audit, round 3: findings disposition (hma-gate-d-audit-20261021-d5b4m)

**Verdict: PASS.** N1, N3, N4, N5 and N7 are closed; my controls verify each one end to end or at the unit
level. N2, N6, N8 and N9 can stay open for D5b. Three new LOW findings exist (P1–P3). None breaks a D5b
requirement. P1 and P3 are latent until the core allows verified outcomes, and they must be fixed before
that happens.

## Scope, provenance, isolation

- **Target:** HEAD `7c555f67006e4ced6e76f060d74b3a100a76d363` plus the working tree. The tracked diff sha256 is
  `ed85aadde0ff036a3833d527544800136f79a047e3d22c741e4482c2f74f0261`, which equals the expected value.
  `freeze-start-*` and `freeze-end-*` match byte for byte: the status, the diff hash and the sha256 of every
  untracked file. `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*` were excluded.
- **PHASE1.json:** sha256 `ae5a186b9e48ebce260c572f7916511516495d53a774645e7bd996613e3a7553`. I wrote it from
  the sources (v3 §4.1, §6 row 4; arch §5.2, §7.1, §7.2, §9, §14; gate-c §0.1) before reading the handoff
  `2026-10-21-gate-d-d5b.md` and before reading the diff in detail. Before writing it, I ran only the freeze
  and `git diff --stat`.
- **Isolation: partial.**
  - The implementer is Claude Sonnet 5.5 and I am Claude Opus 5.5. That is the same vendor family, so there
    is no provider separation.
  - I wrote rounds 1 and 2 of this audit, and the dispatch summarised the implementer's fixes. My context
    therefore carries both my earlier findings and the implementer's own claims.
  - This is a fresh session.
- **Execution:**
  - Every test ran on copies, `copy/` and `copy2/` (controls), under the output directory, with GOCACHE and
    TMPDIR inside it. Go came from `~/.local/go/bin`.
  - I made no edits and no git writes in `/home/yeli/repos`.

## Regression

| Check | Result |
|---|---|
| `go vet ./...` | exit 0 |
| `go test -count=1 ./...` | all packages ok (`test-buildvcsfalse.log`) |
| `go test -race` on cmd/hma, evidence, repostate, host, engine | all ok, 0 DATA RACE (`regression.log`, `race-cmdhma.log`) |
| Skipped | `TestGateBR2HostChild` (child helper) and `TestGateBR2HostPowerLossImage` |

The first run had two failures: `TestD5HMABoundLaunchRequiresEligibleTrue` and
`TestD5ResumeReplayUsesSameDispatchID`. The cause was "error obtaining VCS status", because my copy has no
`.git` while those tests build the binary. This is an environment artifact. With `GOFLAGS=-buildvcs=false`
both tests pass, and so do the full suite and the cmd/hma race run.

## Requirement coverage

| Req | Result | Evidence |
|---|---|---|
| R1 Severity never sets disposition | pass | `TestDispositionFollowsSection9`; `MapAuditFindings` reads no severity (`findings_import.go:77-84`) |
| R2 BLOCK only for a reproduced kernel violation; unwaivable; downgrade refused | pass | `resolution.go:220,253-255`; established BLOCK is re-raised on every import |
| R3 WAIVABLE only for a measured budget excess | pass | D1: `U1/HMA-BUDGET-max_files-<tag>` exists only when the measured excess exists |
| R4 Report cannot create, suppress or clear BLOCK/WAIVABLE | pass | `HMA-` prefix and `/` refused at import (`findings_import.go:73`); a `pass` that cites `hma_ref` escalates (observed in D1 draft: `HOST_INDEPENDENT_REVIEW_MISSING`) |
| R5 Unresolved BLOCK/WAIVABLE stops the gate | pass | D1, D3; `disposition.go:197-218` |
| R6 Waiver scoped to the finding and its artifact | pass | D1 (N1 closed); P1 note |
| R7 No `VERIFIED_SUCCESS` with an active waiver | pass (latent) | D3 (N7) |
| R8 Facts measured, bounded; `no_shell` judges the program the capture runs | pass | D5 (N3), D6 (N5), D8 |
| R9 No new false refusal | pass with P2 | full suite green; P2 is an edge case |
| R10 Gate C §0.1 row: registry, reproduction, budgets; v3 §6 row 4 test | pass | registry `no_shell`; `MeasuredBudgets`; proof-row test passes |

## Closure of N1–N9

- **N1 closed (observed, D1 `TestD4N1EndToEnd`).**
  - The tag is the first 16 hex characters of `rr.Review.ArtifactDigest`. That value is the sha256 of the
    reviewed head, diff and worktree (`host.go:1060-1068`), set at packet time (`review.go:206-210`). It is
    the right digest.
  - Before the import measures, `requireLiveTarget` binds the live tree to the reviewed target
    (`review.go:458`).
  - The same artifact keeps the same id across re-reviews (`…-6cff1833936ecbe1` both times). After a waiver
    and a fresh same-artifact review, the gate passes.
  - Adding 20 files gives a new id (`…-b57fbf0a4950dbfe`), and the gate refuses with
    `HOST_FINDINGS_UNRESOLVED`. The old-artifact finding is dropped by the snapshot. Its waiver stays active:
    the gate ignores it, but closure does not (P1).
  - Collisions need a 64-bit prefix match. A second preimage against a waived artifact costs about 2^64. A
    pre-computed colliding pair costs about 2^32, but only a party that controls both artifacts can build
    one, which the trusted-host threat model excludes (opportunity O1).
- **N2: acceptable, not a defect.**
  - The behaviour fails closed. §7.2 does not require a waiver to take effect without re-review. The re-raised
    id is stable for the same artifact.
  - `TestWaivableNeedsHumanWaiver` now walks the gate (`disposition_test.go:213-229`), and D1 reproduces the
    path.
  - The cost is one extra independent review for each waiver (opportunity O2).
- **N3 closed (observed, D5).** Counts:

  | Path | Lines |
  |---|---|
  | Symlink (to `/dev/zero`) | 1 |
  | Directory | 1 |
  | FIFO (no block) | 1 |
  | Empty file | 0 |
  | 20 MiB file | 16384 (cap; still far above any budget) |
  | Unreadable file | 0 lines (D8), but `repostate.Snapshot` refuses a dirty target that holds one (`permission denied`), so no packet can bind it |

- **N4 closed for established ids (observed, D4 through `hma resolve`).** The override is refused with
  `CANNOT_OVERRIDE_HMA_ESTABLISHED_FINDING`, and report-sourced `U1/F1` stays overridable. A report source id
  containing "/" (and therefore "/HMA-") cannot exist (`findings_import.go:73`; `TestGateCR11SlashFindingIDRefused`
  passes). False refusal for unit ids: see P2.
- **N5 closed (observed, D6).** A non-executable `#!/bin/sh` decoy named `true` early on PATH now gives false.
  `sh -c`, `/bin/bash` and `env sh -c` give true. A missing tool gives false.
- **N7 closed (observed, D3).**
  - With an active waiver, `VERIFIED_SUCCESS` is refused with `HOST_FINDINGS_UNRESOLVED`.
  - `VERIFIED_WITH_WAIVERS` and non-verified outcomes pass the findings stage. The next refusal comes from
    `transition.Apply` because my input was incomplete.
  - This is latent: the core still returns `OUTCOME_NOT_IMPLEMENTED`.
- **N6: acceptable open for D5b.**
  - The effect is an over-count, which gives a false WAIVABLE. It never creates a BLOCK and never hides an
    excess. The host is trusted, and a human can waive a false WAIVABLE.
  - The handoff now states the limit correctly (`d5b.md:86-87`).
  - **Required before Gate D is claimed passed:** measure with filters neutralised, as round 2 proposed:
    `hash-object --no-filters` into a temporary index, or `-c filter.<x>.clean=` for all configured filters.
- **N8: acceptable open for D5b, with a condition (see P3).**
  - It does not weaken the review gate for packet-import reviews. A native review still passes
    review→verification without facts.
  - **Required before verified closure is enabled:** P3's fix.
- **N9: acceptable.**
  - These are documented measurement definitions. Renames count as delete plus add, merges as a tree diff,
    binary content as text lines.
  - Each fails toward WAIVABLE, not toward BLOCK.

## New findings

### P1 — LOW, observed (latent): a waiver for a superseded artifact stays active and blocks `VERIFIED_SUCCESS`

- **Location:** `engine/resolution.go:55-75`. Nothing expires a finding-scoped waiver when its finding leaves
  the snapshot. `transition.go:315-322` counts every active waiver.
- **Reproduction (`TestD4StaleWaiverAtClosure`):**
  1. Grant a waiver on `…-c2ac7da8389376c0`.
  2. Remediate below budget and re-review. The findings become `map[]`, and the gate passes.
  3. `VERIFIED_SUCCESS` is refused with `HOST_FINDINGS_UNRESOLVED` while ActiveWaivers still holds the gone id.
  4. WITHDRAW succeeds, and the findings stage then passes.
- **Why it matters:**
  - §7.2 says a waiver "expires only when its scoped artifacts … change". The scoped artifact has changed,
    so HMA leaves an expired waiver active.
  - The result fails closed: it never over-claims. But it reports `VERIFIED_WITH_WAIVERS` for a run that has
    no waived finding, unless a human withdraws the waiver.
- **Fix:** in findings import, when the new snapshot drops a finding-scoped id that holds an active waiver,
  append an EXPIRE `waiver_operation` with actor HMA and reason "artifact superseded". An alternative is to
  have the closure count only waivers whose target is still present, but that would conflict with the
  `VERIFIED_WITH_WAIVERS` precondition.
- **Verify:** D2 makes `VERIFIED_SUCCESS` pass the findings stage without a manual withdraw. D1 still refuses
  artifact B.
- **Effort:** small. Must be fixed before verified outcomes are enabled.

### P2 — LOW, observed (engine level): override refused for report findings of a unit whose id starts `HMA-`

- **Location:** `engine/resolution.go:258`. The test inspects the whole id. Plan unit ids are free-form
  (`model/contract.go:456`).
- **Reproduction (D4):** for a report-sourced WAIVABLE, `EvaluateResolution` returns
  `CANNOT_OVERRIDE_HMA_ESTABLISHED_FINDING` for `HMA-1/F1` and for `proj/HMA-2/F1`. It returns
  `LEGAL_PENDING_HUMAN_APPROVAL` for `U1/F1`.
- **Reachability:** inferred. It needs a plan unit named `HMA-…`, which is plausible in this repository.
- **Fix:** test the source part, not the whole id. Use `strings.HasPrefix(strings.TrimPrefix(id, f.UnitID+"/"), "HMA-")`,
  or better, record a structural `Source: "hma"` marker on established findings. Alternatively, refuse plan
  unit ids that start with `HMA-` or contain `/`.
- **Verify:** D4 accepts `HMA-1/F1` and still refuses `U1/HMA-BUDGET-…`.
- **Effort:** small.

### P3 — LOW, inferred (latent): closure credit does not require the credited review's findings import

- **Location:** `requireIndependentCredit` (`cmd/hma/credit.go:474`) never checks for a findings snapshot.
  Only the review gate requires one (`transition.go:103,374`).
- **Reproduction (`TestD4NoFindingsImport`):**
  1. Use an over-budget artifact with an approving imported review and no findings import.
  2. The review gate refuses with `HOST_FINDINGS_SNAPSHOT_MISSING`, which is correct.
  3. `outcomeGate` returns nil, and the `VERIFIED_SUCCESS` closure passes the findings and credit stages; it
     stops only in `transition.Apply`.
- **Why it matters:** combined with N8 (a native review at the review gate), a run could reach closure with
  no HMA fact ever measured. Today the core refuses verified outcomes.
- **Fix:** at closure, require a findings snapshot whose `FindingsSource.ReviewRecordDigest` is the credited
  review. A simpler alternative is to compute `findingFacts` for the closure artifact directly.
- **Verify:** the D7 closure is refused with `HOST_FINDINGS_SNAPSHOT_MISSING`.
- **Effort:** small. Required together with N8 before verified closure is enabled.

## Opportunities (non-blocking)

- **O1:** use the full 64-hex digest in the budget finding id instead of 16 characters. This removes the
  2^32 pre-computed-pair margin at no cost.
- **O2:** let a GRANT `waiver_operation` for a finding of the same unit pass the stale-binding allowlist.
  This saves one independent review per waiver. Optional.

## Controls

The controls live in copy2 only:

- `copy2/cmd/hma/zz_audit_d5b4_test.go`, sha256 `3ce19e66…984f`
- the helper, sha256 `aadf9d83…0507`

Logs: `controls*.log`, `regression.log`, `test-buildvcsfalse.log`, `race-cmdhma.log`.

## Limitations

- The closure path is exercised only up to `transition.Apply`; the core refuses verified outcomes.
- P2's reachability through a real plan was not run.
- My fixes are proposed and not validated.
- Same vendor family; isolation is partial (see above).

## Next action

Ming fixes P1, P2 and P3 together with the N8 condition, and adds D2, D4 (unit `HMA-1`) and D7 as permanent
tests before verified outcomes are enabled. D5b itself can be committed now.
