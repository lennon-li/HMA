# HMA Gate D — D4 independent audit

**Verdict: PASS — D4 only.**

All 8 scoped requirements are MET. All 17 prior finding criteria are FIXED. No new blocking implementation defect was reproduced. The target repository is unchanged.

Full tests, `go vet`, and full race tests passed. The 12 required exit tests passed. The final auditor tests passed under the race detector. The historical r2–r4 checks also passed under the race detector after one disclosed fixture-precondition adjustment.

D5, D6, full Gate D acceptance, and final closure remain outside this verdict. The policy-drift question remains unresolved by instruction.

## Scope, provenance, and isolation

| Item | Recorded value |
|---|---|
| Dispatch | `hma-gate-d-audit-20261018-d4r5` |
| Assigned route | liz / OpenAI / gpt-6.1-sol / high / Codex CLI review mode. Runtime model identity was not independently verified. |
| Target | `/home/yeli/repos/HMA`, `main`, `a74c9f0901f89903ed0cb2425bc491ece4ea282d` plus the frozen working tree |
| Start freeze | `2026-10-19T12:26:36.310050+00:00` |
| End freeze | `2026-10-19T12:38:01.273380+00:00` |
| Tracked binary-diff SHA-256 | `12369f7343f88293f681bf9bf9aabcf0bb5c628b94fa7c92eddd4e7669ac6623` |
| Copied-source manifest SHA-256 | `b6c71346bf689992a7cf884fcd80d9c9a0dd17e3712e99d6a9ad20d71de1ab8b` |
| Phase 1 SHA-256 | `fd5cf31fdc4fa89eabc5b5eb788cd02da3aeb20ba1ff80f73f1e46a12a68aacc` |
| Isolation | Fresh auditor context. Phase 1 was blinded. Execution used a separate source copy on the same host. This was not a separate VM or container. |
| Permissions | Read target. Write audit artifacts and temporary fixtures. No target repository or Git writes. No nested agent CLI or model API. |
| Exclusions | `.serena/`; `docs/handoffs/2026-10-16-arch-amendment-*.md`; later D5/D6 work |
| Tool versions | Go 1.27.1; Git 2.43.0; Python 3.12.3. See `go-version.log` and `tool-versions.json`. |

`freeze-start.json` and `freeze-end.json` include the exact status and SHA-256 of every untracked file. HEAD, branch, status, binary-diff digest, and all untracked digests match. The 259-file source copy has no changed audited file. Auditor tests were added only to that copy.

The original objective is the outcome handoff §5 Gate D. The design comes from Gate A v5 §1–§2, v6 §2 and §4–§5, and v7 §1–§2, with the latest version controlling. Gate C scope lines 16–35 retain the D4 exit tests. Architecture §7, §9, §14, §18, and AGENTS.md constrain the assessment. Exact permitted excerpts are in `normative-excerpts.txt`. Requirements and failure hypotheses were written to `requirements-before-diff.json` before diff inspection.

No approved Jev tool was exposed. The auditor used the documented fallback. No worker dispatch was initiated during this audit. Upstream dispatch preflight was not part of the supplied evidence.

## Requirement coverage

| ID / authoritative source | Observation | Result | Evidence |
|---|---|---|---|
| D4-R1 / v7 §1; v6 §2 | Half-open bounds, expiry, earliest revocation, and conservative ties pass. Whole-chain minima preserve earlier events. | **MET** | `own-host-generations.log`; `exit-tests.log`; `historical-final-race.log` |
| D4-R2 / v7 §1–§2 | Old generations stay revoked. New records use each generation’s own stored plan. Missing generation evidence refuses authorization. | **MET** | `own-host-generations.log`; `exit-tests.log`; `historical-final-race.log` |
| D4-R3 / v5 §1; v6 §4.1–3 | No match, ambiguity, reuse, wrong namespace or mode, and bad receipt identity are rejected. Unlaunched reservations remain charged. | **MET** | `own-host-generations.log`; `historical-final-race.log`; `exit-tests.log` |
| D4-R4 / v6 §4.4–5 | Journal admission and all association fields are checked. Empty journals are included. Unreserved attempts block. | **MET** | `own-cli.log`; `own-host-generations.log`; `closure-addendum.log` |
| D4-R5 / v5 §1; architecture §7, §9 | Unauthorized dispatch persists as an unowned BLOCK. Approving snapshots and later clean scans cannot clear or bypass it. | **MET** | `own-review-cli.log`; `own-cli.log`; `historical-final-race.log` |
| D4-R6 / D4 scope; architecture §7, §14 | The CLI and both progression gates require a clean current scan. New authority, journals, core records, and ledger changes stale it. | **MET** | `own-cli.log`; `own-review-cli.log`; `historical-final-race.log` |
| D4-R7 / v7 §1–§2 | Revocation, host-history, and scan permutations preserve authorization and attempt accounting. | **MET** | `own-host-generations.log`; `historical-final-race.log`; `internal/engine/revocation.go:33` |
| D4-R8 / Gate C §0.1 rows 3–4; D4 scope | All 12 exit tests pass. Regression checks pass. The handoff makes no later-gate completion claim. | **MET** | `exit-tests.log`; `full-test.log`; `vet.log` |

Primary implementation locations: `internal/engine/revocation.go:33`, `internal/host/reconcile.go:127`, `internal/host/reconcile.go:238`, `internal/host/reconcile.go:303`, `internal/host/reconcile.go:428`, `cmd/hma/reconcile.go:39`, `cmd/hma/reconcile.go:209`, `cmd/hma/attempt.go:439`, `cmd/hma/loop.go:87`, and `internal/host/findings_import.go:71`. The copied sources retain exact target bytes.

## Executed evidence

All Go commands used `/home/yeli/.local/go/bin/go`. `GOCACHE` and `TMPDIR` were under this output directory. Tests ran from `source/`. Each check has a `.json` command record and a complete `.log` file. The command records preserve argv, working directory, environment overrides, timestamps, exit code, and detected FAIL lines. Logs combine stdout and stderr without truncation.

| Check | Result | Raw evidence |
|---|---|---|
| Initial `go test ./...` | Failed in two nested builds: VCS status exit 128. Preserved. | `baseline-test.json`, `baseline-test.log` |
| `go test ./... -count=1`, with `GOFLAGS=-buildvcs=false` | Passed in all 12 packages. | `full-test.json`, `full-test.log` |
| `go vet ./...` | Passed. | `vet.json`, `vet.log` |
| `go test -race ./... -count=1` | Passed in all 12 packages. | `race.json`, `race.log` |
| Final auditor tests with `-race`, `-run ^TestR5` | 12 top-level tests and 45 subtests passed. | `own-final-race.json`, `own-final-race.log` |
| Required Gate C rows 3–4 exit tests | All 12 named tests passed. | `exit-tests.json`, `exit-tests.log` |
| Original historical r2–r4 tests | 40 top-level tests passed. One failed an obsolete fixture precondition. The policy-drift probe was excluded. | `historical-original.json`, `historical-original.log` |
| Historical tests with r4 precondition overlay and `-race` | 41 top-level tests and 62 subtests passed. | `historical-final-race.json`, `historical-final-race.log` |
| Post-freeze journal closure controls with `-race` | Matching empty journal passed. Changed baseline blocked with and without attempts. Wrong root was refused. Four subtests passed. | `closure-addendum.json`, `closure-addendum.log` |

The initial nested-build failure was resolved by disabling VCS stamping in the source copy. This did not change production bytes. No executable failed to start. No execution produced a Permission denied error. The audit inspected FAIL lines as well as exit status.

Independent negative controls deliberately disabled three checks through Go overlays. Disabling interval validation caused four boundary failures. Disabling revocation caused the minimum-end test to fail. Disabling ledger inventory validation caused the new-launch freshness test to fail. See `control-half-open-valid.log`, `control-revocation.log`, and `control-gate.log`. These expected failures are not target defects. The first interval mutant failed to compile because of unused variables; it was corrected before the meaningful control run. Its original log is retained as `control-half-open.log`.

## Prior finding closure

FIXED means the supplied closure behavior passed against this frozen target. It does not mean all future behaviors are proven. Historical methods that were not supplied as permitted files were checked with the auditor’s own cases and current repository tests.

| Prior ID | Result | Verification |
|---|---|---|
| D4-F01 | **FIXED** | `TestR5IndependentGateFreshness`; `TestR5IndependentNewReviewDispatchStalesScan`; `TestR3NewJournalInvalidatesScan` |
| D4-F02 | **FIXED** | `TestR5IndependentTwoGenerationPlans`; `TestR3AddendumTwoGenerationPlans` |
| D4-F03 | **FIXED** | `TestR5IndependentTwoGenerationPlans`; `TestAuditorD4RejectOldReviewAfterCorrectionGeneration`; `TestR4IndependentRenewedReviewControl` |
| D4-F04 | **FIXED** | `TestR5IndependentScanOrderAndReviewBinding`; `TestR3ScanAndHostOrder`; `TestR5IndependentReviewSnapshotCannotClearDispatchBlock`; `TestReviewImportUsesPacketLedger` |
| D4-F05 | **FIXED** | `TestR5IndependentClassificationMatrix`; `TestR3ExactBoundsAndMonotoneEvents`; `TestR4IndependentValidityAndScanOrder` |
| D4-F06 | **FIXED** | `TestR5IndependentJournalMatrix`; `TestR5IndependentUnknownJournalGate`; `TestR5IndependentMatchedJournalCLI` |
| D4-F07 | **FIXED** | `TestR5IndependentJournalMatrix`; `TestR5AddendumEmptyJournalAndBaselineCLI`; `TestR3JournalBaselineCanonical` |
| D4-F08 | **FIXED** | `TestR3HookRevocationTie`; `TestR3AddendumPolicyOrder`; `TestReconcilePolicyRevocation` |
| D4-F09 | **FIXED** | `TestR5IndependentBlockSurvivesLaterCleanCLI`; `TestAuditorD4CleanScanRetainsHistoricalGateBlock` |
| D4R2-F01 | **FIXED** | `TestAuditorD4RejectReviewAfterPolicyRevocation`; `TestAuditorD4ReviewPolicyNotPresented`; `TestAuditorD4ReviewWallEnd`; `TestR3AddendumReviewMissingGenerationOrPlan`; `TestR3AddendumRenewedReviewIndependent` |
| D4R2-F02 | **FIXED** | `TestAuditorD4PolicyRestoreDoesNotReviveHook`; `TestPolicyRestoreDoesNotReviveGeneration`; `TestR3HookRevocationTie` |
| D4R2-F03 | **FIXED** | `TestAuditorD4ReceiptRouteMustMatchReviewAuthority`; `TestR3AddendumReviewReceiptMatrixCLI`; `TestR5IndependentScanOrderAndReviewBinding` |
| D4R3-F01 | **FIXED** | `TestR3ReconcilePolicyObservationPersists`; `TestR3AddendumPolicyObservedRestoreMatrix` |
| D4R3-F02 | **FIXED** | `TestR3RepeatedPolicyChangeNewGeneration`; `TestR3AddendumPolicyObservedRestoreMatrix`; `TestR4IndependentPolicyRestoreAcrossGenerations` |
| D4R3-F03 | **FIXED** | `TestR3ReviewIssuedAfterDependencyRewind`; `TestR3ReviewAfterDependencyRewindCLI`; `TestReviewIssuedAfterDependencyRewind`; `TestR5IndependentTwoGenerationPlans` |
| D4R4-F01 | **FIXED** | `TestR4IndependentUnadmittedJournalCannotBypassGate (updated namespace-count precondition only)`; `TestR5IndependentUnknownJournalGate`; `TestR5IndependentGateFreshness` |
| D4R4-F02 | **FIXED** | `TestR4IndependentEmptyJournalMustBindAdmission`; `TestR5IndependentJournalMatrix`; `TestR5AddendumEmptyJournalAndBaselineCLI`; `TestEveryJournalBindsAdmission` |

The exact supplied closure criteria, verification names, and evidence pointers are also retained in `AUDIT.json`. No prior status or verdict was used as initial evidence.

## Findings and remediation

No new blocking D4 implementation finding was reproduced. No D4 corrective patch is proposed. No discretionary implementation change was made. There is no corrective-work roadmap to execute from this audit.

## Controlled-unblinding addendum

Phase 1 was frozen at `2026-10-19T12:34:37.459607+00:00` with SHA-256 `fd5cf31fdc4fa89eabc5b5eb788cd02da3aeb20ba1ff80f73f1e46a12a68aacc`. Its bytes remain unchanged. The file was hashed before any prior auditor test or the implementation handoff was read.

After the freeze, the auditor copied only the permitted r2, r3, and r4 test files. `unblind-manifest.json` records their source paths and SHA-256 values. The original copies remain in `historical-tests/`. The implementation handoff is retained as `handoff-after-phase1.md`. No other prior audit reports or `/tmp/claude-*` files were read.

The original r4 gate test asserted that an imported unadmitted journal declared zero namespaces. The fix makes that assertion obsolete. The unmodified run failed there, before its gate assertion. The overlay changes only this precondition to require one namespace. It retains all gate assertions. The adapted test passed under the race detector. The independently written Phase 1 cases had already passed with no prior scan, with an earlier clean scan, and with both empty and nonempty unknown journals.

One Phase 1 qualification changed: D4-R8 was PARTIAL only because the withheld handoff had not been checked for later-gate claims. That inspection found no D5, D6, full Gate D, or closure-completion claim. D4-R8 is now MET. No Phase 1 finding was removed or rewritten.

The handoff has older descriptions alongside later remediation sections. Those statements were treated as claims. Code and executable checks determined behavior. In particular, clone baseline and review namespace behavior were directly tested.

## Limits and unresolved questions

- Isolation describes fresh audit context and blinded Phase 1. Execution used a separate source copy on the same trusted host, not a separate VM or container. Read-only repository access was enforced by instruction; the environment allowed worktree writes but none were made.
- The assigned worker/model/effort were supplied by the user. Runtime model identity and upstream exact-route preflight were not independently introspected. No auditor-originated worker dispatch, nested agent CLI or model API occurred.
- All conclusions assume the approved trusted-host and host-clock threat model. No hostile administrator or filesystem-authenticity guarantee is asserted. Go race tests detect Go data races; they do not prove atomic external-ledger snapshots against arbitrary concurrent filesystem writers.
- Tests run against copied dirty bytes. They are bound to HEAD, tracked binary-diff digest and per-untracked-file digests. They are not a clean-revision CI attestation.
- The initial full suite failed in two nested builds with error obtaining VCS status, exit status 128. GOFLAGS=-buildvcs=false resolved this in the source copy. The original failure is retained. No executable failed to start and no Permission denied execution failure occurred.
- Original r4 test execution failed its obsolete zero-namespace precondition. The overlay changes only that precondition to require one declared journal namespace. All original behavior assertions remain. The original test and failure log are retained.
- The first half-open mutation control did not compile due to unused variables. A corrected mutation compiled and failed the intended assertions. Mutation failures are expected controls, not findings against the audited target.
- Auditor-authored tests reuse repository fixture builders. Their cases and expected results were independently derived. Full tests and full race runs compiled before the last auditor-only additions; final own tests and addendum tests were separately run with -race.
- TestAuditorD4GateMustDetectPolicyDrift is deliberately excluded. A policy edit after a clean scan is an open contract question, not a closure check. This audit does not resolve that question.
- No approved Jev decision tool was exposed. The auditor used the documented orchestrator fallback.
- D1-D3 and Gates B/C received regression coverage, not a fresh full re-audit. D5 criterion credit, findings disposition, full packet, D6 amendment, full Gate D acceptance and final closure are outside this verdict.
- The implementation handoff contains older descriptions alongside later remediation sections. Its claims were not proof. Primary code and executable results govern this assessment. The excluded architecture-amendment files and .serena content were hashed for freeze verification but not evaluated.
- Raw output logs are complete combined stdout/stderr streams. Tool display excerpts were sometimes truncated; the saved logs were not truncated.

The report-contract validator checks report form and verdict consistency only. Its output is retained as `report-validation.log`. `AUDIT.json` uses `verdict: pass` for that schema and `overall_verdict: PASS` for the same single human verdict. Evidence-file digests are retained in its `evidence_inventory` and in `EVIDENCE.sha256`.

## Required next action — architecture §18

**Lennon reviews this D4 audit packet and records the human D4 decision.** The scoped evidence supports acceptance. The proposed owner is Lennon. No agent/model route is required for this human review. The permission envelope permits inspection of this packet only. Explicit human acceptance of D4 is required before scheduling D5. No HMA approval challenge was generated. This report grants no stage-transition, commit, push, publication, or release authority. Stop if the target digest changes or a new blocking finding or executable-start failure appears. Escalate disputed validity rules and the separate policy-drift contract question to the human.
