# HMA Gate D D1+D2 audit

**Verdict: PASS — scoped to D1+D2 under the trusted-host model.**

All 10 scoped requirement groups are MET. All 16 supplied prior-finding closure criteria are FIXED. No new actionable defect was reproduced. This result does not pass the whole Gate D.

Original tests, vet, and race checks passed. The auditor ran 478 test and subtest invocations in normal mode and again under the race detector. Separate route and addendum checks added 20 race-instrumented invocations. Three deliberately weakened overlays failed the expected behavior checks. Eighteen legacy approvals and eighteen enclosing records retained identical bytes.

## Scope and independence

- Dispatch: `hma-gate-d-audit-20261017-d12r9`.
- Modes: code audit and outcome audit.
- Context isolation: **isolated**. This session received the authoritative request and neutral closure criteria. It did not inherit implementer conversation or read prior audit artifacts.
- Filesystem isolation: a separate copied source snapshot, shared host filesystem, and no continuous repository lock. Repository read-only access was procedural. The source-file sandbox could permit writes; the auditor made none.
- Skill: `/home/yeli/obsidian/AgentSystem/agent-memory/skills/independent-audit/SKILL.md`.
- Requested route label: `liz / gpt-6.1-sol / high`, supplied by the dispatch. Actual provider/model identity was not independently queried.
- Tools: `/usr/bin/git`; `/home/yeli/.local/go/bin/go`, version `go1.27.1 linux/amd64`.
- Permission envelope: read-only inspection of `/home/yeli/repos/HMA`; audit files, copied sources, synthetic repositories, and synthetic stores under this output directory. No nested agent CLI or model API was used. No Jev tool was exposed.
- Exclusions: `.serena/`, `docs/handoffs/2026-10-16-arch-amendment-*.md`, D3-D6 implementation, reservations, launch reconciliation, criterion credit, findings disposition, and the architecture amendment.
- No command launched implementation work under a lease. Pure derivation was tested directly. Runtime integration remains later work.

## Frozen target

- Branch: `main`.
- HEAD: `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`.
- Freeze start UTC: `2026-10-18T00:42:06.595646+00:00`. The local date was 2026-10-17 in America/Toronto.
- Unstaged binary diff SHA-256: `c119b7f95c3cff42c5a2f13037bbef88fefbde6093a55c355a1ccbe64dca19fb`.
- [FREEZE.json](FREEZE.json) records the full short status and the SHA-256 of every untracked file, including excluded files. Excluded files were hashed without reviewing their content.
- [snapshot-manifest.json](snapshot-manifest.json) records 216 copied source and fixture files. The blinded handoff was excluded from the test snapshot.
- [FREEZE-END.json](FREEZE-END.json) matches the start HEAD, branch, short status, binary diff hash, and all untracked hashes. Both repository-to-manifest and snapshot-to-manifest comparisons found zero mismatches.

## Authoritative requirements

Objective: `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md` §5, Gate D. It requires versioned outcome contracts and bounded execution while preserving the human-gated core. The scoped audit checks D1 and pure D2 only.

Design precedence: Gate A v2 §2.2; v3 §2.1–§2.3; v4 §1–§2, with the latest text controlling. Normative constraints: `docs/architecture.md` §§5, 6, 10, and 18, plus `AGENTS.md`. Section 19 controls presentation. [authoritative-excerpts.txt](authoritative-excerpts.txt) preserves the reviewed section bodies. [REQUIREMENTS.json](REQUIREMENTS.json) was written before reading the diff.

| ID | Requirement and observed result | Assessment | Evidence |
| --- | --- | --- | --- |
| D1-01 | Versioned canonical immutable acceptance contract with all v2 fields and v3 route/permission ceilings; exact core criterion IDs; no result dispositions. Contract definitions and ceilings are present. The schema contains no result dispositions. Exact criterion-ID comparison is enforced. | **MET** | `internal/model/contract.go:20-130`; `probes.log:TestAuditAllDeclaredFields`; `probes.log:TestAuditMaterialChangesAndGraphs` |
| D1-02 | Strict document interpretation: reject ambiguous/malformed data; material changes alter digest. Invalid UTF-8, lone surrogates, duplicate keys, case aliases, unknown keys, trailing documents, missing keys, and null declarations refuse. Valid scalar Unicode survives. Material definition changes change the canonical digest. | **MET** | `internal/model/contract.go:190-378`; `probes.log:TestAuditUnicodeAndAmbiguity`; `mutant-unicode.log`; `mutant-declarations.log` |
| D1-03 | Optional omitted-empty approval field preserves legacy bytes; initial presented contract binds; later approvals carry exact contract; swaps and drops refuse. Real Apply calls refuse an unbound presented contract, wrong digest, missing initial document, wrong run, wrong criteria, later drop, later swap, and changed threshold. Both-absent legacy and matching controls pass. | **MET** | `internal/model/types.go:197`; `internal/engine/contract_binding.go:60-109`; `internal/transition/apply.go:147-157`; `probes.log:TestAuditApplyContractBinding`; `legacy-comparison.json` |
| D1-04 | hma.plan/v1 contains every architecture §10 declaration, wall/cost ceilings and required effective policy digest; validate unit/dependency graph. Every declared key was deleted and set to null separately. Incomplete plans refuse at parsing and lease derivation. Explicit zero retries, empty argv, empty contract non-goals, and empty dependency declarations pass. Cycles, duplicate units, unknown dependencies, and wall overflow refuse. | **MET** | `internal/model/contract.go:442-599`; `probes.log:TestAuditAllDeclaredFields`; `probes.log:TestAuditMaterialChangesAndGraphs`; `probes.log:TestAuditContractCeilings` |
| D2-01 | Only latest applicable approved legal transition into implementation authorization issues lease; deterministic digest binds Approval.TransitionDigest, unit and plan. No issuer and illegal-edge controls refuse. Initial and legal correction issuers pass. Independent reference hashes match. Changing transition digest, unit, or plan changes identity. A record-anchor change does not change lease identity. | **MET** | `internal/engine/lease.go:90-151`; `probes.log:TestAuditLeaseIdentityReference`; `probes.log:TestAuditReworkRetainsValidRoute`; `addendum-probes.log:TestAuditIdentityFieldsIndependently` |
| D2-02 | Presented valid plan and contract match issuer bindings, run and unit; exact current core criterion IDs; selected route identity and permission stay coherent. Plan, contract, run, unit, criterion IDs, selected route identity, selected permissions, and route approval reference must match. Fresh-issuer criterion drift refuses. A disposition-only change preserves the ID check. | **MET** | `internal/engine/lease.go:106-129`; `internal/engine/lease.go:157-163`; `internal/engine/lease.go:207-226`; `probes.log:TestAuditLeaseBoundaries`; `probes.log:TestAuditMissingCriterionAndFreshReapproval`; `probes.log:TestAuditContractCeilings`; `route-probe.log`; `addendum-probes.log:TestAuditFreshIssuerCriteriaDrift` |
| D2-03 | Terminal state, wrong stage, same-unit/run/direct/transitive dependency rewinds void old generation; unrelated unit does not; new legal issuer may resume. Unit, whole-run, direct dependency, transitive dependency, and dependency correction invalidations refuse. Grounding and Acceptance-criteria rewinds invalidate under every tested unit tag. Real Apply histories cannot reuse an old route after an affected rewind. Unrelated-unit and fresh-route controls pass. | **MET** | `internal/engine/lease.go:164-204`; `probes.log:TestAuditDependencyAndWholeRunRewinds`; `probes.log:TestAuditRealApplyRewindRoute`; `probes.log:TestAuditRealApplyChangedPlanRoute`; `mutant-rewind.log` |
| D2-04 | Approval-based wall interval [approval,expiry), nonnegative counters strictly below per-unit and contract ceilings; current computed policy match. The wall interval uses the approval timestamp. A real transition with a two-minute-old approval and one-minute wall ceiling refuses a lease. Exact expiry refuses; one nanosecond before expiry passes. Negative counters, equal ceilings, and policy drift refuse. Valid counters below ceilings pass. | **MET** | `internal/engine/lease.go:130-153`; `internal/engine/lease.go:228-243`; `probes.log:TestAuditRealApplyWallTime`; `probes.log:TestAuditLeaseBoundaries` |
| D2-05 | Bound waiver changes invalidate approval including reused route approval; no old generation revival or inactive grant issuance. Withdrawn, expired, replaced, missing, malformed, and duplicate bound waiver references refuse. Old issuers do not revive after regrant. Correction issuers cannot reuse a waiver-invalidated route. Unbound, unrelated-scope, multiple-live-grant, and fresh-current-grant controls pass. | **MET** | `internal/engine/lease.go:246-328`; `probes.log:TestAuditWaiverGenerations`; `probes.log:TestAuditRealApplyWaivers` |
| D12-REG | Existing transitions, Gate B/C, legacy approval digests remain correct; no claims of D3-D6 implementation. All existing packages pass test, vet, and race checks. Legacy approval and record serialization matches bc5acb7. The handoff explicitly defers D3-D6 and full Gate D passage. | **MET** | `test.result.json`; `vet.result.json`; `race.result.json`; `probes-race.result.json`; `legacy-comparison.json`; `docs/handoffs/2026-10-17-gate-d-d1-d2.md:5-8,38-40 (read after Phase 1 freeze)` |

## Verification evidence

All paths below are relative to this audit directory. The source snapshot is `snapshot/`. Auditor tests are in [snapshot/internal/auditprobe](snapshot/internal/auditprobe). They use independently defined documents and histories. They do not import old auditor fixtures. Real Apply tests use synthetic Git repositories and verified stores. Synthetic pure-interface tests also exercise invalid or mismatched input directly.

| Check | Observed result | Raw evidence |
| --- | --- | --- |
| `go test -count=1 ./...` | Exit 0. All 12 original packages passed. | [test.log](test.log), [test.result.json](test.result.json) |
| `go vet ./...` | Exit 0. | [vet.log](vet.log), [vet.result.json](vet.result.json) |
| `go test -race -count=1 ./...` | Exit 0. All original packages passed. The auditor package existed before its tests were added, so this run reports no tests there. | [race.log](race.log), [race.result.json](race.result.json) |
| Independent probes, normal mode | Exit 0. 15 test groups; 478 test and subtest invocations. | [probes.log](probes.log) |
| Independent probes, race mode | Exit 0. The same 478 invocations passed. | [probes-race.log](probes-race.log), [probes-race.result.json](probes-race.result.json) |
| Independent probe vet | Exit 0. | [probes-vet.log](probes-vet.log), [probes-vet.result.json](probes-vet.result.json) |
| Route projection identity and unit binding | Exit 0 with race detection. 14 invocations. | [route-probe.log](route-probe.log) |
| Identity-field isolation and fresh-issuer criterion drift | Exit 0 with race detection. 6 invocations. Added after Phase 1 freeze. | [addendum-probes.log](addendum-probes.log) |
| Disable retained-route rewind scanning with `-overlay` | Exit 1 as expected. Eight affected rewind cases produced forbidden active leases. The unrelated Planning control passed. This was a behavioral failure, not a build error. | [mutant-rewind.log](mutant-rewind.log), [mutants/rewind.json](mutants/rewind.json) |
| Disable required-key checking with `-overlay` | Exit 1 as expected. Twelve missing/null declaration cases failed. This was a behavioral failure, not a build error. | [mutant-declarations.log](mutant-declarations.log), [mutants/declarations.json](mutants/declarations.json) |
| Disable Unicode checks with `-overlay` | Exit 1 as expected. Twelve malformed Unicode cases failed. Distinct invalid bytes normalized to identical digests in the mutant. This was a behavioral failure, not a build error. | [mutant-unicode.log](mutant-unicode.log), [mutants/unicode.json](mutants/unicode.json) |
| Legacy serialization against committed bc5acb7 model source | All 36 serialized artifacts are byte-identical. These cover 9 stages, with and without the optional worktree digest, for both approvals and records. | [legacy-comparison.json](legacy-comparison.json), [legacy-before.log](legacy-before.log), [legacy-after.log](legacy-after.log) |

Legacy-output SHA-256: `90bdca5c805ba913879b240547a10ba04b5400460d3f59f588ed8532ad68e0f1`.

`run-checks.py` and `run-extra.py` preserve commands and environment. Tests used `PATH=/home/yeli/.local/go/bin:/usr/bin:/bin`, `GOTOOLCHAIN=local`, `GOCACHE` under this audit directory, and matching `TMPDIR` and `GOTMPDIR` under `tmp/`. Production files in the snapshot still match the frozen source manifest. Overlays alter only test-time compiler inputs.

## Prior-finding closure

FIXED means the current code meets the supplied closure criteria under this scope. It does not certify later integration. The prior test artifacts were not read or reused. [criteria-d12r9.json](criteria-d12r9.json) preserves the supplied criteria. Every result below has independent replacement tests in this packet.

| Prior ID | Subject | Status | Independent evidence |
| --- | --- | --- | --- |
| F01 | Contract route and permission ceilings do not constrain leases | **FIXED** | `TestAuditContractCeilings`; `TestAuditLeaseBoundaries`; `TestAuditRouteProjectionCoherence` |
| F02 | Dependency correction does not invalidate dependent leases | **FIXED** | `TestAuditDependencyAndWholeRunRewinds`; `TestAuditReworkRetainsValidRoute` |
| F03 | Lease wall time starts at record append instead of human approval | **FIXED** | `TestAuditRealApplyWallTime`; `TestAuditLeaseBoundaries` |
| F04 | Unicode decoding loses document content before hashing | **FIXED** | `TestAuditUnicodeAndAmbiguity`; `mutant-unicode.log` |
| F05 | Plan validation accepts units missing required bounded-work fields | **FIXED** | `TestAuditAllDeclaredFields`; `TestAuditMaterialChangesAndGraphs` |
| F07 | Lease derivation ignores current core criterion ID drift | **FIXED** | `TestAuditFreshIssuerCriteriaDrift`; `TestAuditMissingCriterionAndFreshReapproval` |
| F06 | Initial acceptance can silently drop a presented contract | **FIXED** | `TestAuditApplyContractBinding` |
| F08 | Lease digest uses record anchor instead of the specified transition field | **FIXED** | `TestAuditLeaseIdentityReference`; `TestAuditIdentityFieldsIndependently` |
| F09 | Negative counters pass the lease ceiling check | **FIXED** | `TestAuditLeaseBoundaries` |
| D12R2-01 | Required document fields can be omitted or null, including a plan that obtains an active lease | **FIXED** | `TestAuditAllDeclaredFields`; `mutant-declarations.log`; `legacy-comparison.json`; `test.log`; `vet.log`; `race.log` |
| F-D12R3-01 | A unit tag can hide a whole-run Grounding rewind from lease invalidation | **FIXED** | `TestAuditDependencyAndWholeRunRewinds`; `TestAuditRealApplyRewindRoute` |
| D12R4-01 | Selected-route coherence is absent from the pure lease interface | **FIXED** | `TestAuditLeaseBoundaries`; `TestAuditRouteProjectionCoherence`; `TestAuditReworkRetainsValidRoute` |
| D12R5-01 | A lease stays active after a bound waiver invalidates its issuing approval | **FIXED** | `TestAuditWaiverGenerations`; `TestAuditRealApplyWaivers` |
| D12R6-01 | A new issuer can bind an inactive or replaced waiver grant and obtain a lease | **FIXED** | `TestAuditWaiverGenerations`; `TestAuditRealApplyWaivers` |
| F-D12R7-01 | Correction lease reuses a route approval invalidated by a bound waiver | **FIXED** | `TestAuditWaiverGenerations`; `TestAuditRealApplyWaivers` |
| F-D12R8-01 | Correction issuer reuses a route approval invalidated by a plan or whole-run rewind | **FIXED** | `TestAuditReworkRetainsValidRoute`; `TestAuditRealApplyRewindRoute`; `TestAuditRealApplyChangedPlanRoute`; `mutant-rewind.log` |

## Findings and remediation

No new actionable finding was reproduced. No remediation is required for this scoped result. No code was changed in the audited repository.

Non-defect opportunity: the external auditor probes and mutation overlays are reusable regression evidence for later D3 work. They remain external. This audit proposes no additional implementation change.

## Phase 1 freeze and controlled unblinding

[PHASE1.json](PHASE1.json) was created and hashed before the implementation handoff was read. Its SHA-256 is `6fd060bb6a8a1a9aa654884926beb896b8dbafeeab3d4c35280843254abcef17`. The file remains unchanged. [PHASE1.sha256](PHASE1.sha256) records the digest.

Only after that freeze, the auditor read `docs/handoffs/2026-10-17-gate-d-d1-d2.md`. The handoff separates D1+D2 from D3–D6. It says lease derivation is not wired to a command. It does not claim full Gate D passage. The narrative agrees with the scoped result. Its remediation statements remain author claims; tests and source inspection support this audit. No prior audit directory was opened.

The post-freeze addendum tests strengthen two checks. They hold the transition digest fixed while independently varying the unit or plan digest. They also test criterion drift against a fresh issuer. Both passed under race detection. They did not change the blind verdict. [UNBLINDING.json](UNBLINDING.json) records the addendum.

## Limits and assumptions

- Scope is D1 and D2 only. D3-D6, reservation accounting, dispatch enforcement, launch reconciliation, criterion credit, finding disposition, and architecture amendment are not validated.
- Trusted host supplies an authenticated validated core chain, current computed policy digest, current route projection, wall clock, and counters. Pure derivation does not authenticate host assertions or enforce OS process confinement.
- Same live filesystem as the target. No implementer conversation inherited. User-supplied prior closure criteria intentionally informed adversarial checks; prior verdicts and test artifacts were not read.
- Only synthetic repositories and stores under the output directory were mutated. The audited repository received no writes. Tests necessarily perform git writes in synthetic repositories; no audited-repository git write was issued.
- No complete malicious-host or exhaustive concurrency/state-space proof is claimed. Existing race tests and independent probes passed.
- Heading-only rg indexed the specified design files to find section line ranges. Only the permitted Gate A section bodies were printed with sed and used as design requirements.
- The full race run saw the auditor package before its tests were added and reports [no tests to run] for that package. All 478 auditor invocations ran separately under -race. The later route projection test is recorded separately.
- No Jev tool was exposed in the tool catalog. No model API or nested agent CLI was used.
- Go 1.27.1 linux/amd64 was used with module go directive 1.21; Go 1.21 runtime compatibility was not separately executed.
- Repository read-only access was procedural. The host sandbox allowed source-file writes, while .git was separately read-protected. Start/end hashes and the 216-file snapshot comparison demonstrate no observed change; they are not a continuous filesystem lock.
- Some interactive tool output was truncated. Full test logs, mutation logs, source snapshot bytes, and manifests are retained in the audit directory. No test-log file was truncated.

The host must supply a validated core chain, the current computed policy digest, the actual selected-route projection, the current wall clock, and carried counters. These are trusted-host assumptions. This audit does not claim process confinement or resistance to a host that fabricates these inputs. Only attempt and cost counters are inputs to the current pure interface. Future reservation and launch integration must enforce its own approved requirements.

## Human gate and single next action

Current outcome: PASS for D1+D2. Evidence: all scoped requirements are MET; all 16 prior criteria are FIXED; no reproduced blocking finding; unchanged target freeze. Full Gate D passage remains outside this result.

**Next action: the human gate owner reviews this evidence packet and decides whether to authorize the separate Gate D D3 task.**

Rationale: the D1+D2 checks support progression to the next scoped human gate. Proposed owner: the human gate owner. Proposed execution route: none is dispatched by this audit; a D3 worker route requires separate selection and exact-route preflight. Permission envelope: this audit remains read-only on `/home/yeli/repos`; artifacts remain under this output directory. Exact approval required: explicit human authorization of the D3 task, acceptance criteria, worker route, and permission envelope. This report creates no approval challenge and grants no execution authority. Stop on target digest drift, a new blocking finding, or absent human authorization.
