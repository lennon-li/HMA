# HMA Gate D — D4 independent audit

**Dispatch:** `hma-gate-d-audit-20261018-d4r3`  
**Verdict:** **FAIL**  
**Isolation:** isolated conversation; same-host file-copy test environment.  
**Target:** `main` at `a74c9f0901f89903ed0cb2425bc491ece4ea282d` plus the frozen working tree.  
**Tracked diff SHA-256:** `7dd3504a5027f8634706acbd3bba96aa515b15c4e56fdf78882bdca526ddaaed`  
**Phase 1 SHA-256:** `7fc810b3180a396613def4b26dbc4d87c92f3a4b69c2e65c2b3dd44a32bbb01a`

## Human review

Three independently reproduced defects can authorize a launch after revocation. All three findings have high severity. D4 must remain closed.

1. Reconciliation observes a changed policy but does not retain the revocation. Restoring the policy revives the reservation.
2. The hook suppresses a repeated policy-change observation in a new generation. Restoring the policy revives that generation.
3. A review packet issued after a dependency rewind misses that generation’s rewind. Reconciliation authorizes the review launch.

The full repository suite passes in the adjusted copy environment. The race suite and `go vet` pass. The supplied r2 closure tests pass. These results do not resolve the three independent counterexamples.

Prior closure: **9 FIXED, 3 PARTIAL, 0 OPEN**. Requirement coverage: **12 MET, 4 PARTIAL, 0 NOT MET**. PARTIAL means that some required controls pass and a mandatory part still fails.

Observed: start/end HEAD, Git status, tracked diff digest and every standard Git-untracked file hash match. The 259 included input files also match the test copy. No audited repository file was changed. See [FREEZE.json](FREEZE.json), [END-FREEZE.json](END-FREEZE.json), and [SOURCE-MANIFEST.json](SOURCE-MANIFEST.json).

This report grants no stage transition, criterion credit, Gate D overall passage or closure. D5 and D6 remain outside scope.

## Scope and provenance

Authoritative sources:

- Outcome consolidation handoff, 2026-10-15, §5 “Gate D”.
- Approved design: Gate A v5 §1–§2; v6 §2 and §4–§5; v7 §1–§2. Latest revision wins.
- Gate C scope, 2026-10-16, lines 16–35, rows 3 and 4.
- `docs/architecture.md` §7, §9, §14, §18 and §19; `AGENTS.md`.
- User-supplied closure criteria: `/tmp/hma-gate-d-criteria/criteria-d4r3.json`.

The approved design was printed only through the requested `sed` section ranges. Requirements and failure hypotheses were recorded in [REQUIREMENTS.json](REQUIREMENTS.json) before reading the diff. [PHASE1.json](PHASE1.json) was created exclusively and hashed before controlled unblinding. Its bytes remain unchanged.

The auditor did not read implementer narratives or prior auditor reports before that freeze. Source comments and authored tests were inspected as evidence. After the freeze, the auditor read only the permitted r2 test files and the D4 handoff. [R2-TESTS.json](R2-TESTS.json) records the imported test hashes. The r2 source files were copied into the output tree. No prior report was read.

The requested worker/model/effort was `liz / gpt-6.1-sol / high`. This is dispatch provenance supplied by the user. It was not independently authenticated. Jev was unavailable. The auditor used the auditor/human fallback. No subordinate agent or model API was used.

## Requirement coverage

| ID | Requirement | Assessment | Observation / evidence |
|---|---|---|---|
| D4-R01 | Exactly one HMA launch record; no match or two matches is unauthorized. (Gate A v5 §1; v6 §4.1) | **MET** | Implementation and review mutations refuse zero/two matches. Valid controls pass. Evidence: `own-tests-phase1.log: TestR3LaunchMatrix`; `r2-closure.log: TestAuditorD4IndependentAuthorizationMatrix`. |
| D4-R02 | One record cannot authorize two ledger runs. (Gate A v6 §4.2) | **MET** | A receipt/directory mismatch or wrong namespace is refused; a used record cannot authorize again. The implementation sorts scans before consumption. Evidence: `regressions.log: TestOneReservationPerLaunchNoReuse`; `source/internal/host/reconcile.go:186-223`. |
| D4-R03 | Use exact half-open [valid_from,valid_until) and conservative ties. (Gate A v6 §2; v7 §1) | **MET** | One nanosecond before issuance fails. The exact lower bound passes. The upper and revocation ties fail. The hook waits until its next whole-second issuance bound. Evidence: `own-tests-phase1.log: TestR3ExactBoundsAndMonotoneEvents`; `own-tests-phase1.log: TestR3LaunchMatrix`; `regressions.log`. |
| D4-R04 | Effective validity is the monotone minimum of expiry and every applicable rewind, terminal, policy, contract and wall-time event. (Gate A v7 §1) | **PARTIAL** | Recorded event minima, expiry, terminal, review wall-time, and post-dispatch rewinds pass. Policy observations can be lost. A review created after a dependency rewind misses the rewind. Contract rebinding requires the acceptance-criteria path; its rewind is globally revoking. Evidence: `own-tests-phase1.log`; `addendum-policy.log`; `r2-closure.log`; `source/internal/engine/contract_binding.go:25`. |
| D4-R05 | Renewal creates a new generation and never revives old authority. (Gate A v7 §1) | **PARTIAL** | Ordinary renewal and correction controls pass. Repeated policy changes under a new generation are suppressed, so restoration revives that revoked generation. Evidence: `own-tests-phase1.log: TestR3RepeatedPolicyChangeNewGeneration`; `addendum-generation.log`; `addendum-policy.log`. |
| D4-R06 | Use each generation’s approved dependency plan and refuse missing generation/plan evidence. (Gate A v7 §1; closure D4-F02) | **MET** | A two-plan, two-generation fixture preserves the earlier launch and refuses the new launch at its dependency rewind. Missing stored plans and missing generation evidence refuse authorization. Evidence: `own-addendum.log`; `addendum-generation.log`; `r2-closure.log`; `source/cmd/hma/loop.go:399`. |
| D4-R07 | Bind namespace, mode and actual review route, packet and target; maintain review independence. (Gate A v5 §1) | **MET** | Wrong namespace/mode fails. Worker, model, provider, effort, packet, root, head and worktree mutations fail individually. Bound running and completed reviews pass. A receipt alone creates no independent-review record. Evidence: `own-tests-phase1.log: TestR3LaunchMatrix`; `own-addendum.log: TestR3AddendumReviewReceiptMatrixCLI`; `r2-closure.log: TestAuditorD4ReceiptRouteMustMatchReviewAuthority`. |
| D4-R08 | Scan all declared implementation and review namespaces since the first core record. (Gate A v5 §1–§2; v6 §4.1) | **MET** | All admission and review packet namespaces are scanned. A rogue in the separate review ledger becomes BLOCK. Pre-core-start receipts are excluded. Legacy packets lacking a namespace remain outside scan coverage; see limitations. Evidence: `own-tests-phase1.log: TestR3ReviewSnapshotCannotClearBlockCLI`; `own-tests-phase1.log: TestR3ScanAndHostOrder`; `source/cmd/hma/reconcile.go:135`. |
| D4-R09 | Journal envelope, controller, live root, baseline, unit and namespace match exactly one admission. (Gate A v6 §4.4) | **MET** | Each independent journal association mutation fails. The matching control passes. Absent baseline inventory and {} hash equally; a changed inventory differs. Evidence: `own-tests-phase1.log: TestR3JournalMatrix`; `r2-closure.log: TestAuditorD4IndependentJournalBinding`; `own-tests-phase1.log: TestR3JournalBaselineCanonical`. |
| D4-R10 | Every journal dispatch has a reservation; unlaunched reservations remain charged and listed. (Gate A v6 §4.3 and §4.5) | **MET** | Unknown controllers and unreserved attempts produce unauthorized entries. Matching journals pass. Mixed implementation/review scans preserve implementation reservation counts. Unlaunched reservations remain in the result and counters. Evidence: `own-tests-phase1.log: TestR3JournalMatrix`; `regressions.log`; `r2-closure.log`. |
| D4-R11 | UNAUTHORIZED_DISPATCH is persistent unwaivable BLOCK across review snapshots and later scans. (Gate A v5 §1; architecture §7) | **MET** | An approving imported review and findings snapshot preserve the unowned BLOCK. Removing the rogue and rescanning does not reopen the gate. A historical BLOCK still recommends escalation. Evidence: `own-tests-phase1.log: TestR3ReviewSnapshotCannotClearBlockCLI`; `r2-closure.log: TestAuditorD4CleanScanRetainsHistoricalGateBlock`; `regressions.log`. |
| D4-R12 | Progression requires a clean current reconciliation bound to core, relevant host records and ledger inventory. (D4 task scope; architecture §7 and §14) | **PARTIAL** | New receipt, reservation, review authority, journal or core-head changes invalidate old scans. However, the two lost-policy cases produce false clean scans and requireReconciledLaunches returns nil. The separate unobserved policy-file drift question is excluded from closure. Evidence: `own-tests-phase1.log: TestR3NewReviewAuthorityInvalidatesScan`; `own-tests-phase1.log: TestR3NewJournalInvalidatesScan`; `addendum-policy.log`; `source/cmd/hma/reconcile.go:188`. |
| D4-R13 | Hook and reconciliation agree on irreversible generation revocation. (Gate A v5 §2; v7 §1) | **PARTIAL** | The single recorded-policy control and revocation tie pass. Reconciliation-only observation and repeated-policy observation fail to preserve revocation for replay/import. Evidence: `own-tests-phase1.log: TestR3HookRevocationTie`; `r2-closure.log: TestAuditorD4PolicyRestoreDoesNotReviveHook`; `addendum-policy.log`. |
| D4-R14 | Record ingestion and scan order do not change classification or increase valid_until. (Gate A v7 §1–§2) | **MET** | Reversing host ingestion, scan order and dated revocation event order preserves classifications and the earliest applicable end in tested cases. Ordered core-chain generation lookup assumes a valid persisted chain. Evidence: `own-tests-phase1.log: TestR3ScanAndHostOrder`; `r2-closure.log: TestAuditorD4MonotoneEndAllEventOrders`; `addendum-policy.log: TestR3AddendumPolicyOrder`. |
| D4-R15 | Preserve D1–D3, Gate B and Gate C behavior. (Original objective §5 Gate D; user scope) | **MET** | All 12 packages pass normal and race repository suites. Vet passes. Changed Gate C fixtures add namespace/plan inputs and current reconciliation while retaining refusal assertions. New D4 defects remain separate from regression-suite results. Evidence: `regressions.json`; `regressions-race.json`; `final-vet.json`; `start-diff.patch`. |
| D4-R16 | Make no D5 criterion-credit/full-packet or D6 amendment completion claim. (Gate C scope §0.1; user scope) | **MET** | The handoff requests D4 verification/audit and does not claim D5/D6 completion. Reconciliation alone gives no criterion credit. Amendment files were excluded. This audit grants no Gate D overall passage or closure. Evidence: `unblinded/2026-10-18-gate-d-d4.md`; `own-addendum.log: TestR3AddendumReviewReceiptMatrixCLI`. |

## Findings and remedies

All findings below were frozen in Phase 1. Each fix is proposed. No remediation was made or validated.

### D4R3-F01 — Reconciliation observes policy drift without persisting revocation

**Severity:** high. **Certainty:** observed. **Mode:** code and outcome. **Status:** open.

**Observed failure.** After issuance, reconcile changed policy with no launch. Restore policy. Hook returns LaunchEligible:true; a later post-change receipt is Authorized with no Unauthorized entry.

**Primary evidence.** `cmd/hma/reconcile.go:72; internal/host/reconcile.go:365`. Run `TestR3ReconcilePolicyObservationPersists`. See [own-tests-phase1.log](own-tests-phase1.log).

**Post-freeze confirmation.** [addendum-policy.log](addendum-policy.log) repeats this case with a launch timestamp captured after the policy observation and before import. The hook returns `LaunchEligible:true`. Reconciliation returns `Authorized:[d-1]` or `Authorized:[d-3]`, with no unauthorized entry. `requireReconciledLaunches` returns nil. This shows a false clean reconciliation can satisfy the launch gate. It does not show a completed human-approved stage transition.

**Diagnosis.** Only hook recordPolicyChange persists dated revocations. Reconciliation stores PolicyDigest but PolicyRevokedAt ignores reconciliation records.

**Preferred fix.** Persist each applicable policy observation under the host/core locks before reconciliation, with generation-scoped identity. Derive hook and import validity from the same observations.

**Independent verification.** Changed-policy reconcile, restore, replay and fresh issuance must refuse old generation. Post-change receipt must become BLOCK. Before-change control and separately approved new generation must remain eligible.

**Trade-offs.** Avoid retroactively revoking launches before the first observed change; unseen filesystem changes remain outside trusted-host evidence.

**Alternative.** Treat historical launch_reconciliation policy digests as revocation observations with their timestamps and core heads.

**Effort and priority.** medium effort; priority 1. Contain by keeping D4 closed. Then apply the smallest fix and obtain an independent re-test.

### D4R3-F02 — Global policy-pair deduplication loses revocation in later generations

**Severity:** high. **Certainty:** observed. **Mode:** code and outcome. **Status:** open.

**Observed failure.** Observe P→Q under G1; restore P; approve G2 and reserve; observe P→Q again; restore P. G2 replay is LaunchEligible:true and post-change G2 receipt is Authorized.

**Primary evidence.** `cmd/hma/attempt.go:443; internal/host/reconcile.go:387`. Run `TestR3RepeatedPolicyChangeNewGeneration`. See [own-tests-phase1.log](own-tests-phase1.log).

**Post-freeze confirmation.** [addendum-policy.log](addendum-policy.log) repeats this case with a launch timestamp captured after the policy observation and before import. The hook returns `LaunchEligible:true`. Reconciliation returns `Authorized:[d-1]` or `Authorized:[d-3]`, with no unauthorized entry. `requireReconciledLaunches` returns nil. This shows a false clean reconciliation can satisfy the launch gate. It does not show a completed human-approved stage transition.

**Diagnosis.** recordPolicyChange deduplicates solely by policy digest pair; PolicyRevokedAt correctly ignores G1 observations before G2, leaving G2 without its own event.

**Preferred fix.** Deduplicate only when an existing dated observation applies to each active generation. Record repeated policy changes at a current core head when a new generation is not covered.

**Independent verification.** Repeat identical policy pair across two approved generations. Both old generations remain revoked after restoration; a third independently approved generation remains eligible. Reverse event ingestion must preserve the earliest applicable end.

**Trade-offs.** Append more observations; migrations must retain earlier event times and old-generation applicability.

**Alternative.** Store explicit affected issuer sequence and head per revocation; permits precise per-generation deduplication.

**Effort and priority.** small effort; priority 1. Contain by keeping D4 closed. Then apply the smallest fix and obtain an independent re-test.

### D4R3-F03 — Review issued after a dependency rewind bypasses generation revocation

**Severity:** high. **Certainty:** observed. **Mode:** code and outcome. **Status:** open.

**Observed failure.** Generation plan has U1 depending on U2. U2 returns from Implementation authorization to Route selection. The CLI issues a fresh U1 review packet and reconciliation authorizes its later receipt.

**Primary evidence.** `internal/host/reconcile.go:357; cmd/hma/transition.go:222`. Run `TestR3ReviewIssuedAfterDependencyRewind; TestR3ReviewAfterDependencyRewindCLI`. See [own-tests-phase1.log](own-tests-phase1.log).

**CLI trace.** The fixture approves a plan where U1 depends on U2. U2 moves from Route selection to Implementation authorization. U2 then returns to Route selection at `2026-09-16T12:05:00Z`. HMA issues a U1 review packet at `12:05:20Z`. The receipt starts at `12:05:50Z`. Reconciliation returns `Authorized:[after-dependency-rewind]` and `Unauthorized:[]`. The pure-function negative control fails in the same way.

**Diagnosis.** Review revocation scan starts at dispatch CoreHeadAnchor rather than generation issuer; the cross-unit route check does not reject another unit returning to Route selection.

**Preferred fix.** Compute review generation validity from the lease issuer for all applicable dependency and global revocations. Preserve permitted own-unit forward progress. Refuse packet issuance when its generation is already revoked.

**Independent verification.** Dependency rewind before packet issuance refuses the packet or produces BLOCK for its launch. Test dependency planning, route selection, and correction; before-event control and independent-unit control remain authorized; fresh approved generation works.

**Trade-offs.** Do not use implementation RevokedAt unchanged: own forward move into review must remain legal.

**Alternative.** Require an explicit stored review generation binding and validate it at both issuance and reconciliation.

**Effort and priority.** medium effort; priority 1. Contain by keeping D4 closed. Then apply the smallest fix and obtain an independent re-test.

## Prior finding closure

| Prior ID | Closure | Evidence-backed reason |
|---|---|---|
| D4-F01 | **FIXED** | New ledger receipts, review authority and journals invalidate the scan. A clean unchanged control passes. Evidence: `own-tests-phase1.log: TestR3NewReviewAuthorityInvalidatesScan`; `own-tests-phase1.log: TestR3NewJournalInvalidatesScan`; `regressions.log: TestGateBindsReconciledLedger`. |
| D4-F02 | **FIXED** | Generation-bound retained plans are used; both old and new plan fixtures behave correctly around dependency rewinds. Evidence: `own-addendum.log: TestR3AddendumTwoGenerationPlans`; `addendum-generation.log`; `regressions.log: TestReconcileUsesGenerationPlan`. |
| D4-F03 | **PARTIAL** | Old review IDs are revoked at rewinds/corrections and fresh renewed reviews pass. A new dispatch issued after a generation dependency rewind is still authorized (D4R3-F03). Evidence: `r2-closure.log`; `addendum-generation.log`; `own-tests-phase1.log: TestR3ReviewAfterDependencyRewindCLI`. |
| D4-F04 | **FIXED** | Packet namespace binds import; separate review ledger is scanned; reversed mixed-mode scans keep both valid launches and reservation accounting. Evidence: `regressions.log: TestReviewImportUsesPacketLedger`; `own-tests-phase1.log: TestR3ScanAndHostOrder`; `own-tests-phase1.log: TestR3ReviewSnapshotCannotClearBlockCLI`. |
| D4-F05 | **FIXED** | Exact sub-second lower bounds and exclusive upper/revocation ties pass; no truncation widening remains. Evidence: `own-tests-phase1.log: TestR3ExactBoundsAndMonotoneEvents`; `own-tests-phase1.log: TestR3LaunchMatrix`. |
| D4-F06 | **FIXED** | Unknown controller/unreserved attempt cases become unauthorized. Correct controller/reservation control passes. Evidence: `own-tests-phase1.log: TestR3JournalMatrix`; `r2-closure.log: TestAuditorD4IndependentJournalBinding`. |
| D4-F07 | **FIXED** | Baseline and root mismatches become unauthorized. Empty/absent baseline controls agree. Evidence: `own-tests-phase1.log: TestR3JournalMatrix`; `own-tests-phase1.log: TestR3JournalBaselineCanonical`; `r2-closure.log`. |
| D4-F08 | **PARTIAL** | First hook policy observation, before-change control and tie work. Reconciliation observations and repeated digest pairs are lost (D4R3-F01/F02). Evidence: `regressions.log: TestReconcilePolicyRevocation`; `own-tests-phase1.log: TestR3HookRevocationTie`; `addendum-policy.log`. |
| D4-F09 | **FIXED** | Later clean scans retain BLOCK, close the gate and recommend escalation; no-history clean control can continue. Evidence: `regressions.log: TestReconcileNextActionKeepsHistoricalBlock`; `own-tests-phase1.log: TestR3ReviewSnapshotCannotClearBlockCLI`; `r2-closure.log`. |
| D4R2-F01 | **FIXED** | The specified policy, correction-generation, wall-end and missing-policy cases now refuse. Before-event and forward-progress controls pass. Missing generation/plan refuses; fresh renewed review is independently eligible. The older dependency-rewind issue remains under D4-F03. Evidence: `r2-closure.log`; `own-addendum.log`; `addendum-generation.log`; `addendum-policy.log: TestR3AddendumPolicyOrder`. |
| D4R2-F02 | **PARTIAL** | Single observed-change restoration refuses replay and new issuance. Restoring policy revives authority after reconciliation-only observation or repeated changes in a later generation. Evidence: `r2-closure.log: TestAuditorD4PolicyRestoreDoesNotReviveHook`; `regressions.log: TestPolicyRestoreDoesNotReviveGeneration`; `addendum-policy.log`. |
| D4R2-F03 | **FIXED** | Actual implementation worker is rejected at reconciliation before report import. Every receipt route/packet/target field is independently varied. Bound running/completed launches pass without granting review credit. Evidence: `r2-closure.log: TestAuditorD4ReceiptRouteMustMatchReviewAuthority`; `own-addendum.log: TestR3AddendumReviewReceiptMatrixCLI`. |

The neutral criteria are retained in [closure-criteria.json](closure-criteria.json). The original r1 test bodies were not supplied through an allowed path. The auditor used the neutral closure criteria and independent equivalent checks. The two permitted r2 source files contain 12 closure test functions. All 12 pass. `TestAuditorD4GateMustDetectPolicyDrift` was run separately and is excluded from closure.

## Verification and raw evidence

Go version: `go1.27.1 linux/amd64`. Direct Go path: `/home/yeli/.local/go/bin/go`. Git resolves to `/usr/bin/git`. Tool hashes are in [ENVIRONMENT.json](ENVIRONMENT.json).

Tests ran from `/tmp/hma-gate-d-20261018-d4r3/source`. The copy has no Git metadata. Successful adjusted runs use these environment values:

```text
GOCACHE=/tmp/hma-gate-d-20261018-d4r3/gocache
TMPDIR=/tmp/hma-gate-d-20261018-d4r3/test-tmp
GOFLAGS=-buildvcs=false
```

The table shows Go arguments after the absolute executable path. Full argument vectors, start/end timestamps, exit codes, output lengths and hashes are in [EVIDENCE.json](EVIDENCE.json). Per-command `.json` files retain the execution environment. Raw `.log` files contain complete stdout/stderr.

| Check | Arguments | Result | Raw evidence |
|---|---|---|---|
| Repository regression | `test -count=1 ./... -skip ^TestR3` | Exit 0; all 12 packages pass; no FAIL lines. | [regressions.log](regressions.log) |
| Repository race regression | `test -count=1 -race ./... -skip ^TestR3` | Exit 0; all 12 packages pass; no race report. | [regressions-race.log](regressions-race.log) |
| Static checks | `vet ./...` | Exit 0. | [final-vet.log](final-vet.log) |
| Independent blind tests | `test -count=1 -v ./internal/host ./cmd/hma -run ^TestR3` | 10 test functions pass; 4 fail. The four failures demonstrate three defects. | [own-tests-phase1.log](own-tests-phase1.log) |
| Independent race checks | `test -count=1 -race ./internal/host ./cmd/hma -run ^TestR3` | Same four failing functions; no race report. | [own-race.log](own-race.log) |
| Supplied r2 closure tests | `test -count=1 -v ./internal/host ./cmd/hma -run ^TestAuditorD4 -skip ^TestAuditorD4GateMustDetectPolicyDrift$` | 12 test functions pass. | [r2-closure.log](r2-closure.log) |
| Receipt/plan follow-up controls | `test -count=1 -v ./internal/host ./cmd/hma -run ^TestR3Addendum` | 4 functions and 10 receipt-mutation subtests pass at this run. | [own-addendum.log](own-addendum.log) |
| Renewed generation controls | `test -count=1 -v ./internal/host -run ^TestR3Addendum` | 3 functions pass at this run. | [addendum-generation.log](addendum-generation.log) |
| Exact-time policy and gate checks | `test -count=1 -v ./internal/host ./cmd/hma -run ^TestR3Addendum(Policy\|Renewed\|TwoGeneration\|ReviewMissing)` | Four host controls pass. Both policy-restoration subtests fail and show the reconciliation gate opens. | [addendum-policy.log](addendum-policy.log) |
| Open contract probe | `test -count=1 -v ./cmd/hma -run ^TestAuditorD4GateMustDetectPolicyDrift$` | Fails; excluded from closure and blocking findings. | [open-policy-gate-question.log](open-policy-gate-question.log) |

The repository regression runs occurred before r2 test files were added to the copy. They include the repository’s D1–D3, Gate B, Gate C and D4 tests. `-skip ^TestR3` excludes only the independent additions. The r2 closure tests were then run separately. No existing test assertion was removed by the auditor.

**Initial environment failures.** The initial full and race runs failed. Two D5 hook tests could not build the nested HMA binary because Go VCS stamping failed in the copy. One dogfood scratch-directory test assumed that the source directory lies outside the temporary-directory root. The adjusted `GOFLAGS` and `TMPDIR` values resolved these conditions. [baseline-test.log](baseline-test.log) and [baseline-race.log](baseline-race.log) retain those failures. They are not classified as product defects. No executable failed to start.

**Audit-test corrections.** An early snapshot test attempted to import a review bound to the pre-finding core head. HMA correctly refused it as `HOST_CORE_BINDING_STALE`. The auditor then issued a fresh packet and confirmed the snapshot retains BLOCK. An early CLI dependency fixture omitted `stage_time_digest`; it was corrected before the Phase-1 run. Earlier logs remain available. These harness failures are not findings.

## Controlled unblinding and open question

Controlled unblinding did not change the three Phase-1 findings. The r2 tests pass, but they do not test repeated policy observations or a review issued after a dependency rewind. The handoff’s restoration claim is therefore narrower than its wording suggests.

`TestAuditorD4GateMustDetectPolicyDrift` returns a nil gate error after an unobserved policy-file edit. The user identified this as an open contract question. This audit preserves that classification. It is not a fourth blocker and does not change a prior closure status. The actual blockers concern events already observed by HMA or retained in the core chain.

## Remediation roadmap

| Priority | Work | Independent exit criterion |
|---|---|---|
| 1 | Keep D4 closed. Persist generation-aware policy observations for F01/F02. | Neither restoration counterexample authorizes replay, fresh issuance under the revoked generation, or a post-observation launch. |
| 1 | Compute review generation revocation from the lease issuer. Refuse packets from a revoked generation for F03. | Pre-packet dependency rewinds refuse the packet or its launch. Valid own forward progress and new approved generations still pass. |
| 2 | Re-audit the corrected frozen target. | All negative controls refuse, all positive controls pass, and closure tests, full suite, vet and race pass. |

No optional redesign is proposed. The smallest shared revocation fix is preferred. No audited repository changes were made during this audit.

## Limitations

- Conversation isolation is isolated: no implementer narrative or prior auditor report was read before PHASE1 freeze. Code and tests contain unavoidable author comments. This is not process/VM or identity isolation.
- Tests used a copied dirty worktree on the same trusted host. All approvals, receipts and dispatches in the negative fixtures are synthetic. No external worker was launched. No full live human-approved end-to-end outcome is claimed.
- Initial full and race suites failed in the copy: two nested go-build VCS stamping errors and one scratch-directory assumption. GOFLAGS=-buildvcs=false and TMPDIR under the output directory resolved these test-environment issues. The raw failures remain available.
- No executable failed to start. Searches for Permission denied, fork/exec and DATA RACE in retained test logs found none. Missing-file reads are not test results.
- The supplied r1 test names are criteria only. Only the two explicitly allowed r2 auditor source files were imported. Equivalent independent checks support the r1 closure judgments; the original r1 test bodies were not read or run.
- Partial source/log displays in tool output were truncated. Retained test logs and copied primary source bytes are complete. Early mistaken reads of absent filenames and a Go ELF header did not provide evidence and were not relied on.
- Legacy pre-D4 review packets without ledger_namespace are not scanned. This audit applies authorization/namespace exit tests to D4-generated records; it does not certify a legacy migration. No evidence here establishes a legacy packet can legitimately open a D4 gate.
- User designated TestAuditorD4GateMustDetectPolicyDrift an open contract question. It fails, but it is excluded from closure and the blocking findings. The three defects independently demonstrate failures after an observed event or retained core event.
- D5 criterion credit, findings disposition, full acceptance packets and D6 amendments remain outside this audit. Excluded paths: .serena/ and docs/handoffs/2026-10-16-arch-amendment-*.md. Excluded untracked bytes were hashed for freeze integrity only.
- Jev was unavailable through host tool discovery. Auditor/human fallback was used. No model API, nested agent CLI, subagent dispatch, audited-repository edit or Git write, publication or live approval occurred. Tests initialized and committed temporary synthetic Git fixtures only under /tmp.
- Read-only audit was procedural; the host technically exposed worktree write capability. The repository .git was read-restricted. Start/end status, HEAD, tracked diff and all standard Git-untracked hashes match. The copied input has 259 files and no byte mismatch.
- The skill validator accepts lowercase fix-required, while the dispatch requires FAIL. AUDIT.json uses the dispatch value. AUDIT.contract.json is a deterministic compatibility projection that changes only verdict to fix-required; validation of that projection is structural only.

## Report format

`AUDIT.json` uses the user-required verdict `FAIL`. The skill validator uses the equivalent label `fix-required`. `AUDIT.contract.json` changes only that value for structural compatibility. Validation logs and the exact transformation are retained in `VALIDATION.json`. A structural check does not validate the findings.

## Required next action — architecture §18

**RETURN_D4_FOR_REMEDIATION.** The human coordinator should return D4 to the implementer with D4R3-F01–F03. The evidence shows unauthorized launches can be accepted. D4 must remain closed.

**Proposed route:** Ming / Anthropic / `claude-opus-5-5` / high effort / Claude Code CLI. This is a proposal from post-freeze implementer provenance. Eligibility and the exact-route `hi` preflight remain required before dispatch.

**Permission envelope:** Proposed bounded writes for the D4 revocation fixes and tests. No stage change, fabricated approval, commit, push, publication or external effect. This read-only audit grants no implementation authority.

**Exact approval required:** None to receive this audit. Before implementation dispatch, the coordinator must bind the eligible route and exact task/envelope under HMA policy. Each stage transition needs its own human challenge-bound approval.

**Stop and escalation conditions:** Keep D4 closed while a high finding remains. Stop if the permission envelope, preflight or independent re-audit cannot be satisfied. An executable-start failure during retest requires BLOCKED.

**Single next action: RETURN_D4_FOR_REMEDIATION.**
