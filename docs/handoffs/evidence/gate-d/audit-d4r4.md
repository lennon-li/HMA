# HMA Gate D D4 audit

**Verdict: FAIL.** The report-contract value is `fix-required`. These are the same verdict.

An imported journal with an unreserved attempt can pass the full host transition gate without reconciliation. A journal with zero attempts can reconcile clean without a matching admission. The audit found one high defect and one medium gap.

The final full repository test suite, race suite, and vet passed. All 33 permitted r2/r3 auditor tests passed. Those checks do not invalidate the independent counterexamples. Thirteen prior findings are FIXED. Two are PARTIAL.

## Scope and isolation

- Dispatch: `hma-gate-d-audit-20261018-d4r4`. Modes: code and outcome.
- Target: branch `main`; HEAD `a74c9f0901f89903ed0cb2425bc491ece4ea282d`; uncommitted worktree.
- Tracked diff SHA-256: `12369f7343f88293f681bf9bf9aabcf0bb5c628b94fa7c92eddd4e7669ac6623`.
- Context: fresh isolated session. Requirements preceded the diff. No prior verdict or implementer discussion was initial context.
- Execution: 259 source files copied under this output directory. Tests used temporary stores. There was no separate OS account or container.
- Repository access stayed read-only by conduct. The sandbox did not enforce that boundary. No Git write targeted the audited repository.
- Start and end HEAD, status, diff digest, and all untracked-file hashes match. The copied original source files still match the repository.
- Exclusions: `.serena/`, the specified architecture amendment files, D5, and D6. The D4 handoff was content-blinded until Phase 1 was frozen.

The authoritative objective is `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md` §5. The approved design is v5 §1–§2, v6 §2 and §4–§5, and v7 §1–§2. Latest terms win. Gate C scope lines 16–35, rows 3 and 4, define the D4 exit tests. Architecture §7, §9, §14, and §18 remain normative. The threat model is a trusted host.

## Requirement coverage

| ID | Requirement | Result | Evidence |
|---|---|---|---|
| R01 | All namespace receipts since first core record match exactly one reservation or review dispatch; authority cannot be reused. | **MET** | prior-auditor-tests.log: TestR3LaunchMatrix, TestAuditorD4IndependentAuthorizationMatrix; baseline-final.log: full repository suite |
| R02 | Half-open interval; full timestamp precision; earliest applicable event wins independent of ingestion order. | **MET** | independent-final.log: TestR4IndependentValidityAndScanOrder; prior-auditor-tests.log: TestR3ExactBoundsAndMonotoneEvents, TestAuditorD4MonotoneEndAllEventOrders |
| R03 | Every record belongs to one approval generation and its generation-bound plan; renewal never revives it. | **MET** | independent-final.log: TestR4IndependentRenewedReviewControl; prior-auditor-tests.log: TestR3AddendumTwoGenerationPlans, TestR3AddendumRenewedReviewIndependent, TestR3AddendumReviewMissingGenerationOrPlan |
| R04 | Rewind of unit/dependency, terminal, policy/contract drift and generation wall end monotonically revoke both launch classes. | **MET** | independent-final.log: TestR4IndependentPolicyRestoreAcrossGenerations; prior-auditor-tests.log: TestR3ReconcilePolicyObservationPersists, TestR3RepeatedPolicyChangeNewGeneration, TestR3AddendumPolicyObservedRestoreMatrix, TestR3HookRevocationTie; targeted-addendum-corrected.log: planning/correction dependency CLI controls |
| R05 | Envelope, controller ID, live root, clone baseline and namespace equal admission; every journal dispatch has reservation. | **PARTIAL** | independent-final.log: TestR4IndependentEmptyJournalMustBindAdmission; independent-addendum-race.log: TestR4IndependentEmptyJournalCLIMustBindAdmission, TestR4IndependentEmptyMatchingJournalControl; prior-auditor-tests.log: TestR3JournalMatrix, TestR3JournalBaselineCanonical |
| R06 | Unauthorized launches produce BLOCK; review snapshots and subsequent scans cannot clear or bypass it. | **MET** | independent-final.log: TestR4IndependentBlockSurvivesSnapshot; prior-auditor-tests.log: TestR3ReviewSnapshotCannotClearBlockCLI, TestAuditorD4CleanScanRetainsHistoricalGateBlock |
| R07 | Gate requires clean reconciliation covering current core/host/ledger state; reject missing or stale coverage. | **NOT MET** | independent-final.log: TestR4IndependentUnadmittedJournalCannotBypassGate, TestR4IndependentExplicitScanBlocksUnadmittedJournal, TestR4IndependentJournalInvalidatesPriorScan; targeted-addendum-corrected.log: TestR4IndependentUnadmittedJournalAfterEmptyCleanScan; prior-auditor-tests.log: TestR3NewReviewAuthorityInvalidatesScan, TestR3NewJournalInvalidatesScan |
| R08 | Unlaunched reservations remain charged; valid review receipts must match namespace, review mode, route, packet and target; no implementation credit. | **MET** | prior-auditor-tests.log: TestR3ScanAndHostOrder, TestR3AddendumReviewReceiptMatrixCLI, TestAuditorD4ReceiptRouteMustMatchReviewAuthority; baseline-final.log: accounting and review-import regression suite |
| R09 | D1-D3, Gate B, Gate C preserved; D5 and D6 not claimed. | **MET** | baseline-final.log; race-final.log; vet-final.log; docs/handoffs/2026-10-18-gate-d-d4.md (read only after Phase 1) |

MET means the scoped checks passed. It is not proof for all possible input states. R05 is PARTIAL because zero-attempt journals skip admission binding. R07 is NOT MET because an unadmitted journal can bypass the clean-reconciliation gate.

## Findings and proposed remedies

### D4R4-F01 — An unadmitted journal bypasses reconciliation at the transition gate

HIGH. Observed. Fix status: proposed.

The loop-evidence CLI accepts an HMA-bound journal containing an unreserved dispatch attempt. Namespaces considers only loop admissions and review packets. With neither present, requireReconciledLaunches returns nil before examining the journal or requiring reconciliation. After a current-core-bound implementation dispatch, requireHostTransitionGate also returns nil. An explicit reconciliation does detect the unknown admission and attempt, but the transition gate does not require it.

Evidence:
- `phase1-tests/cmd/hma/auditor_r4_test.go:13`: TestR4IndependentUnadmittedJournalCannotBypassGate fails: full host transition gate opened with an unreserved journal attempt, no reconciliation and no BLOCK.
- `independent-final.log`: TestR4IndependentExplicitScanBlocksUnadmittedJournal passes; explicit scan introduces BLOCK and closes the gate.
- `cmd/hma/reconcile.go:218; internal/host/reconcile.go:73; cmd/hma/loop.go:208`: Empty namespace shortcut ignores imported journal evidence; import records the journal without enforcing admission.
- `targeted-addendum-corrected.log: TestR4IndependentUnadmittedJournalAfterEmptyCleanScan`: A prior clean scan with zero namespaces remains accepted after an unadmitted journal is imported.

**Preferred fix:** Include HMA-bound loop-evidence namespaces in mandatory reconciliation coverage, or refuse journal import unless its controller and attempts bind an admission and reservations. Require a current clean reconciliation whenever an HMA-bound journal exists, including when no admission or review packet exists. Preserve the legacy no-launch path only when there is no HMA-bound launch or journal evidence.

**Reason:** The presence of an unadmitted journal must close the gate rather than opt the run out of reconciliation.

**Verification:**
- Run TestR4IndependentUnadmittedJournalCannotBypassGate with no prior scan, then after an earlier clean no-namespace scan. Expected: Import refuses the journal or the full gate requires reconciliation; no progression append.
- Run explicit scan, admitted matching journal, and legacy no-journal controls. Expected: Unknown journal produces BLOCK; matching admitted journal reconciles clean; allowed legacy behavior remains available.

**Trade-offs:** Existing pre-admission loop journals may require an explicit migration or refusal. Namespace coverage can grow; scanning more journals may reveal historical violations.

**Alternative:** Perform and persist full journal reconciliation atomically during loop-evidence import, and have the gate require that proof. This requires wider import changes.

**Effort and priority:** small. P1; contain D4 progression first, then repair gate/coverage, then independent re-test.

### D4R4-F02 — A journal without dispatch attempts skips controller admission binding

MEDIUM. Observed. Fix status: proposed.

reconcileJournals checks len(matched) != 1 only when at least one attempt has a nonempty dispatch_run_id. An unknown-controller HMA-bound journal with no attempts therefore yields zero unauthorized entries. Gate A v6 §4 item 4 applies journal association equality to HMA-bound verify-loop runs without an attempt-count exception. No false launch authorization was demonstrated by this case alone.

Evidence:
- `phase1-tests/internal/host/auditor_r4_test.go:31`: TestR4IndependentEmptyJournalMustBindAdmission fails: unknown HMA-bound journal without attempts accepted with Unauthorized: [].
- `internal/host/reconcile.go:250`: withAttempts guards missing-admission finding; field equality runs only with exactly one admission.
- `independent-addendum-race.log: TestR4IndependentEmptyJournalCLIMustBindAdmission`: The real loop-evidence CLI imports an unknown-controller zero-attempt journal; reconcile returns clean. Matching empty-journal positive control passes.

**Preferred fix:** Require exactly one matching controller admission for every imported HMA-bound journal, including zero-attempt journals. Compare unit, envelope, namespace, live root and baseline only after that mandatory match.

**Reason:** Journal association is independent of the number of launched attempts.

**Verification:**
- Run TestR4IndependentEmptyJournalMustBindAdmission; add zero-attempt matching-admission and mismatched-association controls. Expected: Unknown or ambiguous controller is refused or BLOCK; exact matching empty journal remains clean.

**Trade-offs:** Historical journals imported before admission will be rejected or block progression; define their explicit migration treatment.

**Alternative:** Reject all zero-attempt journals at import. This narrows useful evidence import and is less compatible.

**Effort and priority:** small. P2; repair admission binding with F01, then independent re-test.

No production transition was approved or appended in these counterexamples. The first failure is at the full host gate used by the transition CLI. An explicit reconciliation is a positive control: it records the unreserved nonempty journal as BLOCK and closes the gate.

## Prior finding closure

| Prior ID | Status | Independent assessment | Evidence |
|---|---|---|---|
| D4-F01 | **PARTIAL** | Ledger changes, new review authority and journals close the gate for admitted namespaces. A journal after a clean empty-namespace scan still bypasses it. | prior-auditor-tests.log: TestR3NewReviewAuthorityInvalidatesScan, TestR3NewJournalInvalidatesScan; baseline-final.log: TestGateBindsReconciledLedger; targeted-addendum-corrected.log: TestR4IndependentUnadmittedJournalAfterEmptyCleanScan |
| D4-F02 | **FIXED** | Each generation uses its retained approved plan. Missing evidence fails closed; earlier valid launches remain authorized. | prior-auditor-tests.log: TestR3AddendumTwoGenerationPlans; baseline-final.log: TestReconcileUsesGenerationPlan |
| D4-F03 | **FIXED** | Review bounds include applicable rewind and lease generation. Fresh reviews under new generations remain valid. | independent-final.log: TestR4IndependentRenewedReviewControl; prior-auditor-tests.log: TestR3AddendumRenewedReviewIndependent |
| D4-F04 | **FIXED** | Review namespaces are bound and scanned. Shared-namespace scan order preserves both launch classes and attempt accounting. | prior-auditor-tests.log: TestR3ScanAndHostOrder, TestR3ReviewSnapshotCannotClearBlockCLI; baseline-final.log: TestReviewImportUsesPacketLedger |
| D4-F05 | **FIXED** | Exact lower bound and upper tie are enforced; just-inside controls pass. | independent-final.log: TestR4IndependentValidityAndScanOrder; prior-auditor-tests.log: TestR3ExactBoundsAndMonotoneEvents |
| D4-F06 | **PARTIAL** | Explicit reconciliation blocks unknown controllers with attempts and unreserved dispatch IDs. The gate can avoid that scan, and empty unknown-controller journals still reconcile clean. | independent-final.log: TestR4IndependentExplicitScanBlocksUnadmittedJournal, TestR4IndependentUnadmittedJournalCannotBypassGate, TestR4IndependentEmptyJournalMustBindAdmission; prior-auditor-tests.log: TestR3JournalMatrix |
| D4-F07 | **FIXED** | Baseline and live-root association survive import and are checked; absent baseline and {} canonicalize equally. | prior-auditor-tests.log: TestR3JournalMatrix, TestR3JournalBaselineCanonical; baseline-final.log: TestJournalBaselineMustMatchAdmission |
| D4-F08 | **FIXED** | Observed policy change cuts launch validity. Ties fail. Restoring bytes does not revive generation authority. | independent-final.log: TestR4IndependentPolicyRestoreAcrossGenerations; prior-auditor-tests.log: TestR3HookRevocationTie |
| D4-F09 | **FIXED** | Later clean scans preserve historic BLOCK and escalation. Clean controls still allow continuation. | prior-auditor-tests.log: TestAuditorD4CleanScanRetainsHistoricalGateBlock; baseline-final.log: TestReconcileNextActionKeepsHistoricalBlock |
| D4R2-F01 | **FIXED** | Reviews enforce policy, correction generation, wall end and required policy/plan evidence. Forward controls remain valid. | prior-auditor-tests.log: TestAuditorD4RejectReviewAfterPolicyRevocation, TestAuditorD4RejectOldReviewAfterCorrectionGeneration, TestAuditorD4ReviewWallEnd, TestAuditorD4ReviewPolicyNotPresented, TestR3AddendumReviewMissingGenerationOrPlan |
| D4R2-F02 | **FIXED** | Restored policy cannot revive replay or fresh issuance under an observed revoked generation. New approved generations have a valid control. | prior-auditor-tests.log: TestAuditorD4PolicyRestoreDoesNotReviveHook, TestR3RepeatedPolicyChangeNewGeneration, TestR3AddendumPolicyObservedRestoreMatrix; baseline-final.log: TestPolicyRestoreDoesNotReviveGeneration |
| D4R2-F03 | **FIXED** | Receipt route, packet and target must match review authority. Running and completed valid launches pass and grant no review credit alone. | prior-auditor-tests.log: TestR3AddendumReviewReceiptMatrixCLI, TestAuditorD4ReceiptRouteMustMatchReviewAuthority |
| D4R3-F01 | **FIXED** | Reconciliation policy observations persist and revoke restored-policy replay, fresh issuance and subsequent launches. | prior-auditor-tests.log: TestR3ReconcilePolicyObservationPersists, TestR3AddendumPolicyObservedRestoreMatrix; baseline-final.log: TestReconcilePolicyObservationPersists |
| D4R3-F02 | **FIXED** | Repeated policy pairs under later generations are recorded and remain revoked on restore. | prior-auditor-tests.log: TestR3RepeatedPolicyChangeNewGeneration, TestR3AddendumPolicyObservedRestoreMatrix; independent-final.log: TestR4IndependentPolicyRestoreAcrossGenerations |
| D4R3-F03 | **FIXED** | Dependency revocation applies from generation issuance. Route and correction CLI cases block the launch; planning CLI refuses preflight. Before-event, independent-unit and fresh-generation controls pass. | prior-auditor-tests.log: TestR3ReviewAfterDependencyRewindCLI, TestR3ReviewIssuedAfterDependencyRewind, TestR3AddendumRenewedReviewIndependent; targeted-addendum-corrected.log: TestR4AddendumDependency_planning_CLI, TestR4AddendumDependency_correction_CLI; independent-final.log: TestR4IndependentValidityAndScanOrder |

FIXED applies to the stated defect and closure checks. It does not claim that D4 as a whole passes. D4-F01 and D4-F06 remain PARTIAL because the new counterexamples defeat their broader acceptance conditions.

## Verification and evidence ledger

Go: `go1.27.1 linux/amd64`. Git: `2.43.0`. Auditor Git commands used `/usr/bin/git`. Test fixture Git also resolved to `/usr/bin/git`. GOCACHE is inside this output directory. Final settings are in `verification-environment-final.json`.

Final tests use `GOFLAGS=-buildvcs=false` because the source copy has no Git metadata. `TMPDIR` and `GOTMPDIR` both name the output runtime directory. The final baseline and race suites exclude `TestR4Independent*` and `TestAuditorD4GateMustDetectPolicyDrift`. Those probes run separately.

| Log | Check | Exit | Result |
|---|---|---|---|
| [independent-final.log](independent-final.log) | Blind independent checks | 1 | expected audit assertions failed; inspect raw log |
| [prior-auditor-tests.log](prior-auditor-tests.log) | Permitted r2/r3 closure checks | 0 | passed |
| [open-policy-probe.log](open-policy-probe.log) | Excluded open contract probe | 1 | expected audit assertions failed; inspect raw log |
| [baseline-final.log](baseline-final.log) | Final full repository baseline; excludes independent probes and open contract probe | 0 | passed |
| [race-final.log](race-final.log) | Final full repository race; same exclusions | 0 | passed |
| [vet-final.log](vet-final.log) | Final vet | 0 | passed |
| [independent-addendum-race.log](independent-addendum-race.log) | Independent checks with CLI empty-journal addendum under race | 1 | expected audit assertions failed; inspect raw log |
| [targeted-addendum-corrected.log](targeted-addendum-corrected.log) | Corrected planning/correction CLI controls and stale empty-scan negative check | 1 | expected audit assertions failed; inspect raw log |

The blind independent run had eight tests: six passed and two failed. Post-freeze checks reproduce both findings through the CLI and under race instrumentation. The extra stale-empty-scan check also fails. The corrected planning and correction dependency CLI controls pass. No executable-start failure was observed. No race warning was observed.

The initial full runs failed because the copied source was under the default temporary root and VCS stamping could not resolve a repository. The first environment adjustment still gave Go tests and `os.TempDir` different roots. Raw logs remain in `go-test.log`, `go-race.log`, `baseline-adjusted.log`, and `race-adjusted.log`. The final settings fix the test environment without changing audited code.

The first adapted dependency fixtures also failed. One helper treated safe preflight refusal as a test failure. The correction fixture omitted required approval artifact bindings. Only auditor-owned tests were corrected. Both original and corrected logs remain available.

Complete raw command output is retained. `AUDIT.json` lists command vectors, exits, failure lines, and SHA-256 digests. `source-evidence.txt` retains complete numbered source for the main paths. `audited-file-manifest.json` retains path, content digest, and mode for every copied original file. `freeze-start.json`, `freeze-end.json`, and `freeze-verification.json` retain the state checks.

## Controlled unblinding

Phase 1 SHA-256: `7164be7593abf95f44618168a799f7a8ee5692905b1c92db6ab0f39d9a925ba8`.

`PHASE1.json` is unchanged. `PHASE1.sha256` records its digest. `phase1-tests/` retains the exact blind test sources. `unblinding-provenance.json` records all six copied r2/r3 test files and their digests. The prior auditor reports were not read. The D4 implementer handoff was read after the freeze. No blind finding was withdrawn.

The user-designated policy-drift probe fails. A policy edit after a clean scan remains unseen by the gate until a hook or reconciliation observes it. This is an open contract question. It is not a closure failure in this audit. See `open-policy-probe.log`.

## Remediation roadmap

1. Contain D4 progression. Require an explicit reconciliation for imported HMA-bound journals until the gate enforces coverage.
2. Repair the empty-namespace shortcut and mandatory journal admission binding. Preserve matching empty journals and valid admitted launch controls.
3. Obtain independent re-test of both findings. Keep D4 unpassed until all scoped acceptance checks pass.

These steps describe one bounded repair action and its exit conditions. No implementation improvement beyond the two defects is proposed.

## Limits

- Trusted-host scope. Tests seed authority chains in temporary stores; they do not prove malicious-host resistance. No human-approved production transition or live worker dispatch was performed.
- The audit uses a dirty worktree snapshot, not a clean committed revision. freeze-start.json records the tracked diff digest and every untracked-file digest. audited-file-manifest.json identifies the 259 evaluated copied files. The end freeze is unchanged.
- Read-only behavior on the original repository was voluntary. The sandbox allowed writes, but none were issued against it. Test fixtures used Git only in ephemeral repositories under the output directory. No Git write command targeted the audited repository.
- The source copy excludes .serena/, the specified architecture amendment files, and the blinded D4 handoff until post-freeze inspection. Initial hashing of excluded untracked files exposed no contents.
- Initial full test/race runs contain environment failures and are preserved. A first adjustment still had GOTMPDIR and TMPDIR mismatch. Final runs align both, set GOFLAGS=-buildvcs=false for the non-Git source copy, and pass. They exclude intentionally failing auditor checks and the user-designated open contract probe, each run separately.
- Command logs preserve argv, working directory, exits and complete output without intentional truncation. Precise per-process start/end wall timestamps were not instrumented; log file times and tool-call sequence remain available. Some interactive source output was truncated; source-evidence.txt preserves complete numbered source for the main audited paths.
- Context isolation is a session-level claim, not externally attested process provenance. No prior reports or implementer rationale were initial inputs. Authorized closure criteria and naturally occurring source/test comments can influence attention.
- D5 criterion credit, findings disposition/full packet, and D6 amendment are excluded. Their completion is not claimed. No discretionary implementation improvement was added.
- No approved Jev tool was exposed. Advisory checkpoint decisions used the auditor path. No worker route was dispatched or independently preflighted.
- The known policy-drift probe remains an explicitly excluded open contract question. Green baseline and race results do not settle it or override the four failing assertions that reproduce the two findings.

## Single next action — REMEDIATE_D4_FINDINGS

Current outcome: FAIL. Two journal checks violate the D4 contract.

Owner: the human coordinator assigns a bounded implementation worker. The worker must repair D4R4-F01 and D4R4-F02. No worker is dispatched by this audit.

Route: the human must select a deterministically eligible implementation route and run its exact-route `hi` preflight. No approved Jev tool was exposed for advisory ranking.

Permission envelope: proposed future changes are limited to reconciliation, journal import, transition-gate logic, and regression tests. No commit, push, publication, or stage approval is authorized by this packet. The auditor retains read-only repository access.

Approval: the human must assign the repair route and its bounded implementation permission envelope. Any stage decision still requires the exact HMA challenge-bound approval.

Stop conditions: do not claim D4 passed before independent re-test closes both findings. Report BLOCKED if an executable cannot start. Escalate any proposed contract change to the human. Do not weaken acceptance criteria.
