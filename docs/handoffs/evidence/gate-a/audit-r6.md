# Gate A v6 re-audit

**Dispatch:** `hma-gate-a-audit-20261015-r6`  
**Date:** 2026-10-15  
**Gate verdict: FAIL.** Machine report verdict: `fix-required`.  
**Isolation: partial.** This is the requested re-audit informed by the v1–v5 auditor reports. It is not a blind first pass. The auditor is separate from the implementer. No clean-room independence claim is made.

V6 closes the stated R7, R8, R10, A8 and dependent A4 design gaps. Its half-open interval also fixes the v5 R4 endpoint error. One import-contract conflict remains. A later revocation must never extend an earlier expiry or revocation. V6 says to set the end time to the revoking event time. Its complete reconciliation list then uses that end time. This permits conflicting implementations of the same design.

**New finding R11 is medium severity.** It is a specification conflict. It is not a demonstrated runtime bypass of a complete HMA importer. The required correction is small: preserve the earliest end for each authority generation. Gate B remains stopped pending Gate A acceptance. Future implementation is not required to close this design finding.

## Scope, provenance and freeze

The authoritative objective is [the original consolidation handoff](hma-snapshot/docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md), especially §§0–6 and its seven minimum proofs. The normative constraints are [architecture §§4–7, 9–10, 12–22](hma-snapshot/docs/architecture.md), [AGENTS.md](hma-snapshot/AGENTS.md), [Task 13](hma-snapshot/docs/task13-terminal-outcomes.md), [Task 14](hma-snapshot/docs/task14-host-orchestration.md) and [TODO.md](hma-snapshot/TODO.md). The approved threat model is **T: trusted host**. HMA must detect host accident or drift at import and block closure. Same-UID hostile-host prevention is outside this audit.

The evaluated amendment chain is [v3](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v3.md) → [v4](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v4.md) → [v5](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v5.md) → [v6](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v6.md). Earlier auditor reports are preserved under [prior-reports](prior-reports/). Author assertions in the amendments were objects of audit, not acceptance evidence.

| Target | Observed freeze |
|---|---|
| HMA branch and HEAD | `main`, `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81` |
| HMA freeze time | `2026-10-15T17:36:33.087502+00:00` |
| HMA inventory | 107 tracked/nonignored untracked paths; `.serena/` excluded |
| HMA manifest SHA-256 | `5e6f59a43d7acfec5f1b47cb8a48d1efb20e8a6f779bc8736f11b4d698253e37` |
| V6 SHA-256 | `95508464f4e640fb5fda7aeb8f4069120f932edd866c916c310c465bdf7bd410` |
| Change from v5 freeze | Only `docs/handoffs/2026-10-15-gate-a-v6.md` added; no changed or removed audited paths |
| Obsidian HEAD | `9ba1e74a61ef5133f911903cd5c328666e2170ca` |
| Obsidian manifest SHA-256 | `ab81d47a6fc8f7f320081de5d2eeeaa37e594c0dd4b4ac01a1191feb7bf97ed7` |
| Obsidian checked files | `AgentSystem/scripts/verify-loop`, `AgentSystem/agent-memory/boot/test-verify-loop.py`, `AgentSystem/agent-memory/skills/verify-loop.md` |
| Repair correspondence | All three file hashes match committed `9ba1e74a` |

The [HMA manifest](hma-start-manifest.json), [Obsidian manifest](obsidian-start-manifest.json), source snapshots, [initial diff](hma-start.diff), status files and [prior comparison](prior-comparison.json) preserve the evidence. The current `agent-run-ledger` source is separately frozen in [supplemental-manifest.json](supplemental-manifest.json). It was inspected for exact dispatch-ID uniqueness. The final [target verification](target-verification.json) reports no change to either target or the supplemental source.

The user declares the route as `liz / OpenAI / gpt-6.1-sol / high / review`. The managed runtime and original `hi` preflight were not independently attested. This auditor dispatched no worker. No nested agent CLI, model API, Git write or repository edit ran. All audit writes and test caches stayed under this output directory.

## 1. V5 closure criteria

“CLOSED” below means **design-level closure**. It does not certify future code. The exact prior criteria and evidence references are retained in [AUDIT.json](AUDIT.json), `finding_statuses`.

| Finding | Status | Observed result and reason |
|---|---|---|
| R4 | **PARTIAL** | V6 §2 fixes the inclusive endpoint. A launch exactly at expiry or revocation is unauthorized. Single-event boundary controls pass. The end-time update still lacks monotone composition across multiple revocations; see R11. This is a different residual from the corrected v5 endpoint. |
| R7 | **CLOSED (design level)** | V6 §3 separates existing-reservation validity from unused capacity. A lost final-slot response returns the same reservation without another charge. Its exact ID may launch once before expiry/revocation if no ledger run exists. A new ID still needs capacity. |
| R8 | **CLOSED (design level)** | V6 §1 requires body equality for packet digest, review-dispatch ID and a fresh 128-bit nonce. Rewriting an old report no longer passes. A completed reviewer without a matching report imports nothing. Retained receipt, target, independence and verdict checks still apply. |
| R10 | **CLOSED (design level)** | V6 §4 restores journal/admission envelope equality and controller/live-root/baseline/namespace association. Each launch has one authority record. Each authority record has at most one observed launch. Unlaunched reservations stay charged. |
| A8 | **CLOSED (design level)** | V6 §5 replaces impossible set equality and assigns endpoint, final-slot replay, stale-body and association controls to concrete HMA/Obsidian packages at Gate C. Retained rows cover the other proofs and Gate F. R11 requires an additional bounded control; it does not undo these specific matrix repairs. |
| A1 | **PARTIAL** | Legal human-bound issuance, approved plan/contract/policy, route equality and dependency invalidation remain specified. R7 closes. R11 leaves historical lifetime reconciliation ambiguous. |
| A2 | **PARTIAL** | The before-launch/resume hook, exact-route preflight, pinned association, stop-on-escalation and HMA live recapture remain required. R7/R10 close. R11 still affects historical launch reconciliation. |
| A4 | **CLOSED (design level)** | Cumulative unresolved snapshots, explicit independent per-ID resolutions, same-head evidence repair, disposition mapping, total verdict mapping and final-isolation rejection remain required. The new body binding closes the stale-resolution path. An advisory-only import cannot erase an unresolved BLOCK. |
| A6 | **PARTIAL** | Admission ceilings, atomic monotonic counters, conservative unknown reservations, direct/nested counting and review exclusion remain specified. R7 recovers the last slot. R11 can make the literal time reducer miss an out-of-authority launch. |

R1, R6 and R9 retain their prior design-level closure. V6 introduces no new stage or issuance edge. The only legal lease issuers remain the latest applicable approved transition into `IMPLEMENTATION_AUTHORIZATION` from Route selection or Independent validation. These closures do not mark the existing Gate B code defects fixed.

## 2. Approval and digest compatibility

**No intrinsic conflict was found.** Compatibility requires the stated serialization and enforcement rules. V6 adds no approval field.

V3 removed `execution_lease_digest`. The proposed `ApprovalBinding` addition is the optional `acceptance_contract_digest`, omitted when empty. V4 puts `policy_digest` inside `hma.plan/v1`, which is bound through the existing `accepted_plan_digest`. The additive `findings_snapshot` flag belongs to records, not approvals. [Sources: v3 §§2.1, 4.2; v4 §2.](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v3.md)

Architecture §6 already permits an optional exact worktree binding under the 2026-09-08 amendment. That is a compatible precedent. It does not itself authorize a new semantic amendment. The concrete contract-binding amendment must preserve these properties:

- Preserve the exact legacy JSON bytes when new fields are absent. Existing field order and omission matter because hashes cover serialized bytes.
- Include populated new fields in equality, hash inputs and freshness checks. An omitted or mismatched contract on a contract run must fail closed.
- Require the contract digest at `ACCEPTANCE_CRITERIA -> PLANNING` and propagate it through later approvals. Changed checks or thresholds need the applicable human reauthorization.
- Keep legacy approvals valid for their original semantics. A legacy run without an approved new plan cannot acquire a new lease.
- Document reader/version requirements. An old strict reader may reject populated new fields.

Primary evidence: [architecture §6](hma-snapshot/docs/architecture.md), [ApprovalBinding fields](hma-snapshot/internal/model/types.go), [legacy golden serialization](hma-snapshot/internal/model/terminal_outcomes_test.go) and [approval equality/worktree checks](hma-snapshot/internal/engine/approval_binding.go). Existing golden and worktree tests passed in [go-checks.log](go-checks.log). They do not test fields that have not been implemented.

## 3. Host-drift and attack checks

**Self-issued lease:** No conforming path was found. A host cache grants no authority. The source is an authentic legal core transition whose approved plan digest matches the presented plan. Wrong unit, changed plan, changed policy, invalidated dependency or terminal state must refuse. [Sources: v3 §§2–3; v4 §§1–3.](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v4.md)

**Wrong attempt count:** No new reset, replay double charge or review-as-implementation debit path was found. Reservations charge immediately and remain charged without a launch. Replay returns the same record. Unit totals carry across renewal. Reviews have separate records and do not debit implementation attempts. The current canonical ledger uses an exclusive run-directory creation for an exact ID; an existing ID exits 73. [Source: agent-run-ledger:738–747.](supplemental-snapshot/AgentSystem/scripts/agent-run-ledger) Concurrency and integrated replay still need production tests.

**Stale or self review:** The v5 stale-body sequence is rejected by v6. Reformatting a P0 report during a P1 review retains the old packet digest/dispatch ID/nonce. A truthful current receipt does not repair those body mismatches. Self-review also remains forbidden by distinct-worker, different-model and separate-context requirements. High-risk/final review requires provider/model-family separation. A nonce does not prove reasoning quality. Under T, it supplies the missing current-packet binding. [Sources: v3 §4.3 and §5; v4 §5; v5 §3; v6 §1.](hma-snapshot/docs/handoffs/2026-10-15-gate-a-v6.md)

**Existing BLOCK concealment:** An unresolved F1 must remain in the cumulative snapshot when an advisory F2 arrives. Only an eligible independent exact-target explicit resolution can remove F1. Empty-snapshot semantics do not waive this condition. Stale or missing reports cannot resolve F1 or credit a criterion. The pure model preserves F1. Runtime importer proof remains Gate C/E work.

**Missed new BLOCK:** R11 shows a conflicting import rule that can fail to create `UNAUTHORIZED_DISPATCH` for an expired/revoked launch. It does not erase an already recorded F1. The following is a literal-rule counterexample, not a claim that all retained rules permit it.

1. At time 10, approved authority generation G1 expires at 100. Reserve dispatch I and charge it once.
2. At 40, a human-approved rewind to `ROUTE_SELECTION` revokes G1.
3. At 50, a delayed host scheduler accidentally launches I without a fresh successful hook. The declared ledger records the launch accurately.
4. At 60, the human approves a further rewind to `PLANNING`. This is legal from Route selection. A reducer that applies the written “sets” rule to every affected record changes I's end from 40 to 60.
5. Later, authentic plan/route/authorization approvals allow subsequent work and a fresh review. Import at a nonterminal checkpoint scans the whole namespace.
6. The literal interval `10 <= 50 < 60` accepts I. The required earliest-end interval rejects I and records `UNAUTHORIZED_DISPATCH` BLOCK.

No hostile host, hidden ledger or forged receipt is needed. A second case is expiry at 20, delayed launch at 30 and a later policy event at 40. An unconditional replacement accepts the late launch. A minimum projection refuses it. The retained no-revival rule conflicts with that replacement. An implementation honoring that rule rejects both cases.

## 4. New finding R11

**Title:** Later revocation can extend the written import-validity interval.  
**Severity:** medium. **Kind:** gap. **Confidence:** observed specification conflict. **Status:** OPEN. **Fixed revision:** none.

**Sources:** v6:21–25 and 38–45; v3:47–50; v5:28–40; [probes.py](probes.py), [probes.log](probes.log) and [PHASE1-addendum.json](PHASE1-addendum.json).

**Expected:** The effective end is the earliest original expiry and applicable revocation time for that authority generation. It never increases. A new human-approved generation cannot retroactively authorize a launch under G1.

**Actual:** V6 corrects the interval endpoint but does not state the composition rule. A literal replacement can widen it. Retained active-lease and replay rules prohibit revival. The operative import rule and that invariant need one consistent answer.

**Diagnosis:** The design mixes two operations: shorten existing authority and assign a new endpoint. The suspected implementation risk is a reducer that treats each later event as an overwrite. No future reducer was executed.

**Preferred fix, proposed:** State the rule in v6 §2:

```text
effective_valid_until = min(original_valid_until,
                            prior_effective_valid_until,
                            applicable_revocation_time)
```

Expiry and revocation remain permanent for that generation. Renewal creates separately approved authority. Add `TestLaterRevocationCannotExtendValidity` and `TestExpiryNotExtendedByLaterRevocation` to `internal/host` import at Gate C. Verify generation behavior at Gate D.

**Closure criterion:** State this minimum rule or an equivalent first-revocation terminal rule. Add the two-rewind and expiry-then-later-revocation controls. Check chronological and reverse event ingestion. Keep the equality, valid-before-earliest-end and final-slot replay controls. The design amendment and assigned tests are enough for Gate A. Actual execution belongs to Gate C/D.

**Independent verification:** For the sequence above, I remains cut off at 40 and produces BLOCK. Expiry 20 remains 20 after event 40. A launch just before the earliest end remains valid; equality and later launches refuse. A final-slot reservation with no invalidation still recovers without a second charge. The literal minimum negative controls pass; production code is untested.

**Trade-offs:** This is a small design and reducer correction. It stays conservative on time ties. It needs the original generation's expiry, not a later lease's expiry. It adds no authority field or tamper-resistant launcher.

**Alternative:** Give each generation a terminal revoked state after the first applicable revocation. Preserve immutable original expiry. This works but is larger than a minimum projection over existing times.

**Priority and effort:** Small. Clarify the design before Gate A acceptance. Implement and independently test at the assigned later gates.

**Regression classification:** Newly identified inherited ambiguity. V5 already said revocation sets the end. V6 repeats that rule and makes its interval the complete reconciliation time predicate. No new illegal edge, approval channel or serialization regression was found. R7 and R8 are repairs, not regressions. This report does not claim a proved bypass of all retained requirements.

## 5. Human decisions before Gate B

**The recorded decisions cover the identified product-decision categories. They do not provide every run-specific approval required to start Gate B now.**

V4 §0 records T, D1 architecture direction, D3 Phase B scope, D4 conditional commits to `main`, D5 hook changes and D6 receipt changes. The current request confirms T and supplies the audit dispatch authority covered by D2. No additional D7 product decision is needed for the minimum-end clarification.

Gate A must first pass. Each bounded Gate B unit still needs its approved plan, route, permission envelope, budget and authentic challenge-bound stage approvals. No such concrete packet was supplied for this audit. A report verdict cannot manufacture it.

D1 approves amendment direction. The exact normative text must still be prepared, reviewed and receive the required semantic authority before code relies on changed semantics. This does not require all later documentation to precede routine existing-defect remediation. D4 permits commits only after the applicable audit and full tests pass. Push remains separately approved. The original D1/D3/D4/D5/D6 dialogue was not supplied; this audit inspected the recorded decisions.

## Verification and evidence ledger

| Check | Observed result | Evidence and limit |
|---|---|---|
| Freeze and v5 comparison | Expected HEAD; only v6 added | Manifests, snapshots, `freeze.py`, `prior-comparison.json` |
| Verify Loop repair identity | Three files match `9ba1e74a` | `obsidian-start-manifest.json`; actual committed bytes compared |
| Selected Go model/engine checks | Exit 0; 9 top-level test functions in 2 packages | [go-checks.log](go-checks.log); Go `go1.27.1 linux/amd64`; no full suite |
| Existing controller pure/mocked tests | Exit 0; 7 tests | [controller-unit-checks.log](controller-unit-checks.log); no worker launched |
| Independent models/current-code observations | Exit 0; 40 assertions | [probes.log](probes.log), [machine results](probe-results.json); modeled cases include expected rejection and the R11 counterexample |
| Target unchanged at completion | All unchanged | [target-verification.json](target-verification.json); includes supplemental ledger source |
| Report contract | See `report-validation.log` | Exit 0; contract VALID (form only); no authentication of findings or fixes |

The controller was inspected at its current commit. It already has durable exact attempt IDs, a private ledger namespace, pinned envelope checks, clone execution and unknown-run reconciliation. It does **not** implement D5 yet. It accepts an HMA marker without requiring a hook. Its route normalizer drops provider/profile/access-service fields. Mocked dispatch ignores an injected false hook. These observations establish remaining integration work. They are not separate Gate A failures. D6 receipt fields also remain planned. The current ledger receipt shape does not supply the proposed packet/report digests.

Exact reproduction commands are in [AUDIT.json](AUDIT.json), `verification.reproduction_commands`. The audit used `/home/yeli/.local/go/bin/go` because `go` was absent from PATH. All Go caches and test temporary paths were below this output directory. Python tests ran against snapshots with bytecode writes disabled.

## Freeze, self-challenge and limitations

[PHASE1.json](PHASE1.json) was frozen at `2026-10-15T17:41:20.940648+00:00`. Its SHA-256 is `737c27d0a89e2f66836ccb376e877822e21c1149a1900d06c883f74ddf1f7238`. No implementer unblinding followed. The later addendum is an auditor self-challenge. It preserves the initial terminal-event illustration and replaces it with two legal nonterminal rewinds. A terminal-then-import sequence alone cannot establish a complete bypass because terminal host-write refusal may intervene. The classification and verdict remain a design-conflict assessment.

This audit did not run the full test suite, `-race`, a real worker, real quota/cost metering, integrated concurrent resume, full migration, final closure or end-to-end dogfood. It did not audit all other Obsidian files. No missing future implementation was used alone to fail Gate A. Literal models are not production proof. A correct implementation preserving first revocation would refuse the counterexample.

No approved Jev tool was exposed in the host tool inventory. **JEV_UNAVAILABLE** is recorded. Auditor judgment is the fallback; no model was presented as Jev. The full limitations remain in [AUDIT.json](AUDIT.json). No unrelated improvement is proposed.

## Single next action — REPLAN

**Current outcome:** Gate A FAIL. R4/A1/A2/A6 remain partial through R11. R7/R8/R10/A8/A4 close at design level.

**Owner:** Ming. **Action:** Amend the end-time reducer and add the two multi-event controls. Then submit that amendment for the same read-only Gate A re-audit. Keep Gate B stopped until Gate A passes.

**Rationale:** One small clarification makes import-time detection agree with the retained permanent-revocation rule.

**Proposed route:** Ming / Anthropic / `claude-opus-5-5` / high / Claude Code CLI through the previously approved access-service profile. Revalidate the exact route and service before any worker dispatch. This report dispatches nobody.

**Permission envelope:** Authorized Gate A design text and proof-matrix correction only. This audit grants no repository edit to this auditor, Gate B source change, Git write, push, release, scope expansion or added budget.

**Exact approval required:** No new product decision is identified for this bounded clarification under T/D1. Gate A acceptance and all applicable exact unit/route/stage approvals remain required. Exact normative semantic authority must precede dependent code. No approval challenge was supplied or issued here.

**Stop and escalate:** Refer to Lennon if the repair changes acceptance, threat model, risk, permissions, cost or approval semantics beyond recorded directions, or conflicts with the deterministic kernel.
