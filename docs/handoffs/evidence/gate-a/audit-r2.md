# Gate A re-audit

**Dispatch:** hma-gate-a-audit-20261015-r2. **Date:** 2026-10-15T14:14:39.716702+00:00.

**Gate verdict: FAIL.** Report-contract verdict: `fix-required`. This is a design audit. V2 closes 4 of the 10 requested items at design level. Six remain PARTIAL. Five new design findings are recorded below. Four are high severity. No code fix or future test is claimed complete.

**Isolation: partial.** This is a re-audit using my first-round findings. It is not a blind first pass. V2 is the object under audit. No implementer chat or defense was supplied. No controlled unblinding occurred. The audit skill's validator forbids `pass` with partial isolation; the substantive blockers independently require FAIL here. This report cannot serve as final isolated acceptance.

**Scope and authority:** Original objective: `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md`, especially §§0–6. Norms: `AGENTS.md`, architecture, Task 13, Task 14 and TODO. The current repaired external controller is included only to evaluate v2 §3.1. Both repositories remained read-only. Outputs are under this report directory.

**Declared auditor route:** liz / OpenAI / gpt-6.1-sol / high / review mode. Access: current managed Codex tool session. Runtime configuration and prior dispatch preflight were not independently verified. No worker was dispatched. Jev was unavailable; the permitted auditor fallback was used.

## Frozen target and evidence

- Branch `main`; HEAD `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81`.
- V2 SHA-256: `a571fb84d87c8bec4ae846e803d144d42df47b250cdf71135bd86067ddad950e`.
- Start manifest SHA-256: `63f59e86df8228cbeae82432f376422746c1f86b079dcc560f0ccf4ceefec1fb`.
- Tracked diff SHA-256: `ce2c0caf6c85ea3539542511709e1e3e7e96dc7fa299316a53c7ad6b1466b66c`.
- All 102 prior inventoried files match. The only added file is `docs/handoffs/2026-10-15-gate-a-v2.md`. `.serena/` is excluded.
- `snapshot/` preserves the 103 inventoried source files. `external-snapshot/` preserves external inputs and my prior reports. `external-manifest.json` records exact paths and hashes.
- The resolved external paths are `AgentSystem/scripts/verify-loop`, `AgentSystem/agent-memory/boot/test-verify-loop.py`, and `AgentSystem/agent-memory/skills/verify-loop.md`. The shortened boot/skill paths supplied in the dispatch do not exist.

**Checks:** Three independent pure Go probes and 11 selected existing test functions passed in two packages. The probes confirm design incompatibilities; a passing probe does not mean the proposed design works. Two pure controller checks passed. No worker, model API, Git mutation or external effect ran. Full logs: `go-probes.log` and `controller-probes.log`. Exact overlay source: `gate_a_reaudit_probe_test.go`; mapping: `overlay.json`.

Go command:

```sh
GOCACHE=/tmp/hma-gate-a-20261015-r2/go-cache /home/yeli/.local/go/bin/go test -count=1 -v -overlay /tmp/hma-gate-a-20261015-r2/overlay.json -run 'TestReaudit|TestLegacyApprovalSerializationGolden|TestApprovalWorktreeDigestJSONRoundTrip|TestEvaluateResolution' ./internal/engine ./internal/model
```

## 1. Closure of prior findings

“CLOSED” below means design-level closure only. All corresponding code defects remain unfixed at the frozen tree. This re-audit does not demand later-gate code proof at Gate A.

| ID | Status | Reason and source |
|---|---|---|
| A1 | **PARTIAL** | The initial lease must match a challenge-bound human-approved digest. This closes the direct self-issuance hole in principle. The named issuance edge is illegal (R1). Approved route facts and current policy comparison remain unspecified. The contract field list omits route/permission ceilings although the precedence rule requires them. Dependent invalidation uses an optional submitted list and accepts old records as named-unit-only; it does not define derivation from approved dependencies or fail-closed legacy handling. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:26-76; TODO.md:51-56; cmd/hma/transition.go:131-164`. |
| A2 | **PARTIAL** | The before-launch seam and live-checkout recapture are substantial corrections. The current repaired controller has stable per-attempt dispatch IDs and recovery. The proposed hook does not bind those IDs, immutable envelope, clone baseline, exact selected route or launch receipt to a host reservation. Optional-hook fail-closed activation, idempotence and concurrency are unspecified (R4). Permitted escalation membership does not satisfy the human escalation gate (R3). Source: `docs/handoffs/2026-10-15-gate-a-v2.md:80-101; external controller:830-862,1258-1299,1450-1462`. |
| A3 | **CLOSED (design level)** | The immutable semantic contract binds statements, check argv, thresholds, non-goals and evidence requirements. Its digest is human-approved. Result dispositions remain separate. A threshold change with unchanged IDs must invalidate verification and require reauthorization. Independent result provenance and scorecard edge cases remain obligations under A4/A8; this is not a claim that those implementations exist. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:34-54,152,156`. |
| A4 | **PARTIAL** | Cumulative unresolved snapshots, same-head evidence repair and reopened IDs address major errors. New disposition rules violate the architecture and cannot use the claimed downgrade path (R2). The last finding cannot disappear via an empty snapshot with the current reducer (R5). There is no report-verdict-to-host-verdict table, independently verified per-criterion mapping, or trusted launch/context provenance rule beyond asserted route/isolation fields. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:103-116; internal/engine/resolution.go:42-49,239-258; internal/host/host.go:567-607`. |
| A5 | **PARTIAL** | Final closure, external effects, metadata, immutable evidence/policy/contract bindings and a compiled floor are now explicit. The independence table omits disputed validation. A packet binds a reviewer route, but not explicit reviewer launch/context provenance. Import must preserve the existing separate-context and distinct-worker checks. The detector does not state fail-closed behavior for unknown interface/path classification. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:105-108,120-135; docs/architecture.md sections 12.4,13,16`. |
| A6 | **PARTIAL** | The inequality is fixed. Direct work, inner attempts, reroutes, crash accounting, unmetered refusal and the two-failure stop rule are stated. Stable HMA reservation/result identity, atomic shared counters, monotonic spending across renewal, time-of-launch authorization and exact route binding remain absent (R4). Merely citing section 17 does not reconcile the automatic escalation design (R3). Source: `docs/handoffs/2026-10-15-gate-a-v2.md:88-98,137-146`. |
| A7 | **CLOSED (design level)** | D3 explicitly includes verified closure/release work or requires an explicit acceptance reduction. PARTIAL is no longer offered as equivalent completion. D4 supplies the prior audit alternative: independently review preserved work plus Gate B fixes, authorize a commit, then start from a clean tree. Gate B must still freeze and independently review the later loop result before proceeding. D3/D4 are proposed human decisions, not approvals already received. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:156-172; original handoff section 5 Gate B`. |
| A8 | **PARTIAL** | All seven minimum proofs now have a gate and candidate test names. This is useful design evidence, not code proof. The matrix still omits concrete command/owner/expected-refusal assignments and key controls: renewal/concurrent reservation/replay, report provenance/verdict and criterion credit, disputed validation, N2 nonrepository/mutating-command cases, and empty-final-findings clearing. A test name such as TestNoSecondStageProjection is not a check of all duplicate-state behavior. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:148-160; prior AUDIT.json A8 closure_criterion`. |
| N1 | **CLOSED (design level)** | The revised design explicitly binds the review to head/diff/worktree, and packet changes make it stale. Gate B is assigned the fix and TestReviewBoundToArtifact. Retain the original A-versus-B, dirty-byte and valid-A controls. Code remains unchanged and the reproduced defect remains open until Gate B. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:19,105-106,131-132,154`. |
| N2 | **CLOSED (design level)** | The revised design explicitly requires a snapshot of tested Git content and comparison with the approval. It also requires HMA recapture rather than trusting loop green. This is a sufficient design obligation when read with the original N2 closure criterion. Gate B must still test nonrepositories, wrong checkouts, unapproved dirt and before/after mutation; v2 does not supply those tests. Source: `docs/handoffs/2026-10-15-gate-a-v2.md:20,94-96; prior AUDIT.json N2 closure_criterion`. |

## 2. Approval fields and compatibility

**Observed:** Architecture §6 lists universal bindings and optional worktree binding. It does not forbid all further additive bindings. `ApprovalBinding` uses optional JSON fields. `EvaluateApprovalBinding` compares expected and presented bindings with `reflect.DeepEqual`. Store hashes depend on serialized record bytes. `TestLegacyApprovalSerializationGolden` and `TestApprovalWorktreeDigestJSONRoundTrip` passed in this re-audit. Sources: `internal/model/types.go:156-188`, `internal/model/types_test.go:505-533`, `internal/model/terminal_outcomes_test.go:51-86`, `internal/engine/approval_binding.go:39-64`.

**Assessment:** The two optional digest fields are compatible in principle. They need the explicit D1 normative amendment. Omission when empty can preserve all existing serialized bytes and digests. This does not prove compatibility of fields that have not yet been implemented. Keep field order, tags, legacy defaults and hashing projection stable. Round-trip old records without populating defaults. Add expected/presented mismatch tests for each populated field. Preserve nonce and chain-position checks. New records need updated readers; compatibility does not imply that an old strict reader will understand new populated fields.

**Conflict:** The proposed lease stage is illegal (R1). The worktree amendment precedent cannot authorize a new stage. It also cannot justify retroactively requiring new populated fields on historical approvals. Carrying the contract digest forward must distinguish an outcome-contract run from a legacy run, define clearing on an approved rewind, and never reactivate an invalidated approval. Current code contains neither new field. Future golden/migration tests remain required.

**Open design issue:** If the lease is moved onto the existing route approval, its `route_approval_digest` must not hash the same lease digest recursively. Use a clearly defined preimage or prior route-facts document. This is a design obligation, not proof that every possible implementation is circular.

## 3. Adversarial sequences and limits

| Question | Assessment | Concrete sequence or refusal |
|---|---|---|
| Self-issue an initial lease? | **Explicitly forbidden at design level.** No compliant direct issuance bypass was established. | Host creates arbitrary lease L; its digest differs from the authentic core approval; §2.3 must refuse it. Copying another unit/run approval also fails universal binding. R1 currently makes legitimate issuance impossible. Rewriting trusted store/expected approvals is outside the claimed protection, not a new attack proof. |
| Replay or reuse stale unit authority? | **Residual gap under A1.** | Approve dependent unit B; rewind dependency A using a legacy record with no `invalidated_units`; §2.3 treats it as invalidating A only. Existing route lookup ignores records for A when checking B. Unless approved dependency replay independently invalidates B, B retains stale authority. New list omission must not choose the invalidation set. A whole-run stage check may refuse a simple immediate replay; the design must also cover returning to B's allowed phase without reapproving B. |
| Miscount attempts? | **Yes, the design does not exclude concrete reservation/replay schedules.** | With one attempt left, two hook callers both pass remaining checks before either reserves. Both append/launch unless the operation is atomic. A lost callback response can also reserve twice for one durable controller dispatch ID. Reissuing a lease can reset spending unless counters are unit-cumulative. R4 describes controls. These are design counterexamples, not measured runtime races. |
| Change to an approved escalation route? | **V2 appears to allow an unauthorized sequence.** | P fails twice; controller selects allowlisted E; hook sees allowed E and remaining budget; E starts without a new human route transition. The mock confirms selection only. The missing HMA stop is a proposed-design defect (R3). |
| Import a stale review? | **Explicitly forbidden by v2 §§3.2/4.1.** | Review artifact A; alter head/diff/dirty content, policy, contract or bound evidence; the comparison must refuse the old packet. The original N1 code exploit remains in the unchanged code, but is closed as a design requirement. Do not present that old exploit as proof that the new design permits it. |
| Import a self review? | **Not permitted; provenance enforcement remains incomplete.** | Direct same-model or partial/compromised review is explicitly refused. But an implementer-authored report can assert another route and `isolation=isolated`; route text and a report digest do not prove a launch or separate context. The importer must resolve trusted review-dispatch/receipt/context provenance. Also trace all contributing implementers in multi-attempt work: A writes one part, B repairs another, A reviews the aggregate against only latest implementer B. Existing singular-latest comparison is insufficient unless A's contribution already has independent review. These are acceptance gaps, not a claim that a trusted host cannot lie under the existing threat model. |
| Hide a BLOCK finding with a later advisory report? | **That old attack is explicitly closed at design level.** | F1 BLOCK then F2 ADVISORY must produce a cumulative snapshot containing F1. Omitting F1 violates §3.2. There is no demonstrated compliant overwrite path. However, report verdict is unmapped: a FAIL/blocked report with no rows could yield no imported blockers if an adapter derives APPROVE only from row count. Stable source-report links and a verdict table must prevent concealed failures. An explicit empty snapshot has the opposite failure today: it retains a resolved final blocker (R5). |

Only the initial self-issued/stale-packet rules are stated here as refusals. Missing protocols are not treated as proven runtime exploits. No claim is made that v2 permits bypassing an authentic human stage approval by itself.

## 4. New findings and regressions

### R1 — Lease issuance names a nonexistent stage and unavailable approval

**HIGH · observed · New v2 regression.**

**Source:** `docs/handoffs/2026-10-15-gate-a-v2.md:36-37,60-66; docs/architecture.md sections 5.2,5.4; go-probes.log:TestReauditUndefinedImplementationStage`.

**Actual:** V2 permits execution_lease_digest only on IMPLEMENTATION_AUTHORIZATION -> IMPLEMENTATION, and activates a lease only in IMPLEMENTATION or IMPLEMENTATION_REVIEW. IMPLEMENTATION is not a legal core stage. Current execution begins after ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION. No valid issuance approval can satisfy v2 literally. Replacing only the target with IMPLEMENTATION_REVIEW would bind authority after the implementation it is supposed to authorize.

**Expected:** A valid existing human-approved edge must authorize work before it starts, without adding a second stage machine or an implicit transition.

**Sequence:** Prepare a lease and attempt to obtain the only permitted approval. Model validation rejects proposed_target_stage IMPLEMENTATION. If a host bypasses that rejection it violates the kernel; no compliant issuance path remains.

**Preferred fix:** Choose an existing pre-execution edge, normally ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION, and bind the unit lease there. Update active-stage and latest-applicable lookup rules consistently. Preserve no inheritance across rewind. Explain how the lease references the route approval without a self-referential digest if both are issued in one transition.

**Closure criterion:** Design-trace a real unit from route approval through execution and review, then rewind. Require zero launches before the authentic lease approval, valid in-stage attempts after it, and rejection after rewind. Keep ValidStage/IsLegalEdge negative controls.

**Trade-offs:** Issuing lease and route authority together needs a non-circular digest scheme. A separate new stage would require a larger architecture amendment.

**Alternative:** Bind a proposed lease digest earlier on an approved plan/route document, while a later existing approval activates it. Do not introduce an unapproved IMPLEMENTATION stage.

**Effort:** small. **Priority:** before Gate A acceptance. **Fix status:** proposed. **Fixed revision:** none.

### R2 — Import fabricates blockers and promises a forbidden downgrade

**HIGH · observed · New v2 regression.**

**Source:** `docs/handoffs/2026-10-15-gate-a-v2.md:109-115; docs/architecture.md sections 3,7.2,9,16,20; internal/engine/resolution.go:212-216,253-254; go-probes.log:TestReauditBlockCannotBeReclassified`.

**Actual:** Every critical/high finding becomes BLOCK even without a criterion or invariant. Every unlinked medium/low finding also becomes BLOCK until a human reclassifies it. Section 9 requires out-of-criteria observations to be ADVISORY except kernel violations or declared complexity exceedance. Current resolution refuses both waiver and override of BLOCK. V2 invents blocking acceptance conditions and its escape path does not exist.

**Expected:** Classify by approved criterion/invariant and policy-defined waiver class. Severity expresses impact and cannot create an acceptance requirement. Keep actual unwaivable blockers unwaivable.

**Sequence:** Import a low-severity unrelated cleanup observation. V2 makes it BLOCK. Ask the human to assign ADVISORY through the advertised override. EvaluateResolution returns CANNOT_OVERRIDE_BLOCK_FINDING. Implementing the advertised override by weakening that check would instead create a safety regression.

**Preferred fix:** Validate finding references before import. Map out-of-criteria observations to ADVISORY; map measurable approved complexity exceedance to policy-eligible WAIVABLE; reserve BLOCK for a documented unsatisfied approved criterion or unwaivable invariant. Route unresolved semantic disputes to the existing human process without downgrading kernel failures.

**Closure criterion:** Import unlinked low and high cleanup recommendations: both remain advisory and cannot block valid closure. Import a real kernel violation: it blocks and human downgrade is rejected. Exercise an eligible complexity waiver and out-of-criteria fabricated links.

**Trade-offs:** Invalid references may require a stopped import or an explicit unresolved semantic issue. Conservative safety handling must not become new acceptance criteria.

**Alternative:** Admit only reports already carrying valid HMA references/dispositions, and retain other observations in the immutable report as advisory.

**Effort:** small. **Priority:** before Gate A acceptance. **Fix status:** proposed. **Fixed revision:** none.

### R3 — Permitted escalation routes are treated as sufficient authority

**HIGH · observed · New v2 authority regression / incomplete A2,A6 repair.**

**Source:** `docs/handoffs/2026-10-15-gate-a-v2.md:70-72,88-98,141-145; docs/architecture.md sections 4.1,5.4,17; external controller:1450-1462; controller-probes.log`.

**Actual:** V2 explicitly runs the hook before any escalation route and requires that route only to belong to the lease. It allows repair dispatches without new approval and escalates to a human only when rework needs a rewind. The repaired controller automatically selects escalate_to after two unsuccessful interventions. Section 17 requires human confirmation of failure classification and a new human-approved transition for escalation. An initial lease cannot replace that decision.

**Expected:** After failure classification requires human confirmation or route escalation, stop before the next worker. Resume only with the fresh applicable human-approved transition and matching route/lease.

**Sequence:** Approve a lease containing routes P and E. P fails twice. The controller sets active_route=E. The proposed hook sees E in the lease and sufficient counters and launches E without a new core route transition. The pure mock reproduces automatic selection; actual HMA launch is future code, not reproduced.

**Preferred fix:** Keep the current section 17 semantics. Require the hook to reject escalation selection until a fresh human route transition is present; invalidate the old lease as appropriate. Record observed failure classification and its human confirmation. D5 must include this behavior, not just a callable hook.

**Closure criterion:** Mock two failed interventions with an allowlisted alternate route. Assert no alternate launch before human confirmation and a fresh route transition. Then approve the transition and verify exactly one attempt under the newly applicable authority. Unknown/safety failures must stop.

**Trade-offs:** Long-leash iteration remains possible for genuinely permitted same-route work, but escalation still pauses for a human. A change to this policy would require a separate explicit section 17 authority amendment.

**Alternative:** Configure escalate_to=stop for the first integration and let HMA propose the new human route decision.

**Effort:** small. **Priority:** before Gate A acceptance. **Fix status:** proposed. **Fixed revision:** none.

### R4 — Hook bridge lacks a stable reservation and launch protocol

**HIGH · inferred · New seam gap / incomplete A2,A6 repair.**

**Source:** `docs/handoffs/2026-10-15-gate-a-v2.md:58-68,88-98,137-146; external controller:223-257,830-862,1258-1299; controller-probes.log`.

**Actual:** The hook argv carries run/unit/lease/route but no controller run, stable attempt ID, dispatch_run_id, envelope digest, reservation result or receipt identity. It appends a dispatch before launch, but the design does not specify atomic check-and-reserve, exact dynamic route transmission, idempotent resume, consumed-attempt identity, concurrent callers, renewal accounting, or prevention of an HMA-bound controller job omitting the optional hook. These omissions prevent the stated one-actual-attempt/one-HMA-count guarantee. The repaired controller already has stable IDs that the bridge could reuse.

**Expected:** Each actual launch must consume one globally counted unit attempt under an atomic current authority check. Resume and duplicate callbacks must not double-count. Renewed leases must not reset spent unit authority.

**Sequence:** Two authorized controller jobs both observe one remaining attempt. Both hook processes pass the non-atomic remaining check, then append and launch. Or restart after HMA append but before durable controller acknowledgement: a second hook call consumes a second row for the same stable controller attempt. These are missing-protocol counterexamples, not demonstrated races in unimplemented code.

**Preferred fix:** Define a typed versioned request/response using controller run ID plus dispatch_run_id as an idempotency key. Bind envelope/check definitions, actual selected route, live/clone baseline and lease/core/policy digests. Atomically check authority and reserve shared unit counters. Require an HMA reservation to launch; attach canonical receipt/result; never release ambiguous launches. Require the hook for HMA-bound jobs and define renewal, direct-work, timeout and cancellation behavior.

**Closure criterion:** Run duplicate callbacks, a crash after reservation before launch, a lost launch response, resume, two concurrent controllers with one attempt left, a route change, omission of hook and lease renewal after prior spending. Require at most the approved number of launches and exactly one count per observed launch; unknown launch stays reserved.

**Trade-offs:** A callback is the correct seam, but it must be a protocol rather than only a success exit code. Strictly reserve declared cost; it is not provider billing. OS protection against a hostile trusted host remains outside this audit.

**Alternative:** Initially serialize all work through one HMA controller with escalate_to=stop and prohibit renewal/concurrent nested jobs; still persist stable attempt IDs and receipts.

**Effort:** medium. **Priority:** before Gate A acceptance. **Fix status:** proposed. **Fixed revision:** none.

### R5 — Empty cumulative findings cannot clear the last resolved blocker

**MEDIUM · observed · Newly identified compatibility gap in v2 import design.**

**Source:** `docs/handoffs/2026-10-15-gate-a-v2.md:109-115; internal/engine/resolution.go:42-49; internal/model/types.go:299-307; go-probes.log:TestReauditEmptySnapshotDoesNotClearLastBlock`.

**Actual:** V2 removes resolved findings from the cumulative snapshot. When the last finding resolves, the snapshot is empty. The current reducer updates only when len(Findings)>0, so it retains the old BLOCK. A valid independent repair therefore cannot finish closure through the proposed format. V2 notices replacement semantics but not the absent-versus-empty case.

**Expected:** An explicit authoritative empty snapshot must mean no unresolved findings. Omitted findings must mean no new snapshot. Resolution must retain its independent evidence trail.

**Sequence:** Import F1 BLOCK; independently resolve F1; append the now-empty cumulative snapshot. ProjectResolutionState still returns F1 BLOCK. The pure overlay probe reproduces this byte-state projection.

**Preferred fix:** Add an explicit findings-snapshot marker/kind/version that can represent an empty authoritative set. Make projection distinguish omission from intentional empty state. Only the validated independent import may perform that replacement; retain resolution IDs and report links.

**Closure criterion:** Import BLOCK F1, import an independent exact-target resolution with an explicit empty snapshot, and require zero open findings. Append unrelated records with omitted Findings and require no state change. Reject an unauthenticated or stale empty replacement.

**Trade-offs:** This is a persisted-schema change. Preserve the legacy meaning of omitted/empty ordinary records and existing digests; do not make every legacy record clear all findings.

**Alternative:** Use append-only resolved-finding events with an explicit reducer instead of deleting rows, while retaining cumulative compatibility views.

**Effort:** medium. **Priority:** before Gate A acceptance. **Fix status:** proposed. **Fixed revision:** none.

## 5. Are D1, D3, D4 and D5 complete before Gate B?

**No, as written.** These are useful decision categories. They do not yet identify a coherent and complete decision surface.

- D1 names amendments only to §§6 and 12.4. V2 also conflicts with §§5, 9, 13, 16 and 17. Correct the design to preserve those invariants. Do not ask the human to waive the kernel or authorize silent escalation through a general direction approval. If any genuine authority change remains, state the exact changed clause, behavior and scope before requesting a decision.
- D3 correctly requires either Phase B/release closure or an explicit reduction of the original outcome. Preserve the original criteria until the human chooses. “Recommended” is not an approved decision.
- D4 is a proposed commit envelope. It does not authorize coding, host dispatch, stage transitions, push or release. Code review before a first commit can establish clean start; the later Verify Loop result still needs frozen evidence and review before Gate B completes.
- D5 must name the exact external files, typed hook/receipt protocol, failure/timeout behavior, human escalation stop, independent review and cross-project integration tests. A general hook label does not resolve R3/R4. No external edits are authorized by this audit dispatch.
- Each Gate B unit still needs the applicable authentic plan/unit/route/permission/budget authority and every core transition needs its own challenge-bound approval. This is existing architecture, not a newly invented D6. Prior D2 authorizes this re-audit as dispatched; this report does not infer unlimited later audit dispatch authority.

**Recommendation:** Close the design defects first. Then present corrected D1/D3/D4/D5 with exact bounded execution envelopes. Routine defect remediation does not need a new product decision merely because an auditor found it. Human approval should apply to a concrete valid design, not repair ambiguity by implication.

## 6. Remediation and verification order

1. Correct R1–R3 and preserve the existing stage, finding and escalation rules. Exit: a legal pre-execution approval path, reference-based finding policy and fresh human authority before escalation.
2. Specify R4 and R5, route/dependency authority, report provenance, disputed validation and independent criterion credit. Exit: typed replay/concurrency/import examples with explicit refusal conditions and compatible persisted semantics.
3. Complete A8's command/owner/refusal matrix. Add the counterexamples above and original N1/N2 controls. Freeze a new Gate A packet and obtain re-audit before Gate B. Later code proof remains assigned to later gates.

No optional redesign is recommended. An explicit no-escalation, serialized first integration is a smaller containment option if it meets the approved outcome. It must not silently remove acceptance criteria.

## 7. Audit limits and report integrity

- Partial isolation: this is a re-audit using the auditor first-round report and closure criteria. No claim of a blind first pass or final independent acceptance. No implementer chat or defense was supplied; v2 is the object under audit. No controlled unblinding occurred.
- Read-only by procedure and sandbox scope, not an OS-enforced read-only mount. Both shared trees are dirty. The target snapshot preserves all 103 tracked/nonignored file bytes except .serena; ignored artifacts and Git internals are excluded. Start/end identity and hashes are recorded.
- Gate A is a design audit. CLOSED (design level) does not mean implemented, tested in integration, human-approved, or code-fixed. All original code defects remain at the same revision.
- No nested agent CLI, worker dispatch, model API, network request, human approval ceremony, Git mutation, release or external effect was performed. Controller checks use pure functions/mocks. The full external suite invokes agent-dispatch and Git-writing scratch fixtures, so it was not run under this dispatch.
- Only focused pure Go probes and existing model/resolution tests were rerun. The previous full-suite results were not rerun or treated as fresh results. Future race/crash/renewal/import attacks are labeled design counterexamples, not demonstrated exploits in nonexistent code.
- The dispatch names liz / OpenAI gpt-6.1-sol / high / read-only. Access is the current managed Codex tool session. Runtime/provider configuration and the prior coordinator preflight were not independently verified. No subagent dispatch was needed for this read-only audit.
- Jev was not exposed in the callable host tool inventory. The auditor used the specified orchestrator/human fallback and did not substitute another model as Jev.
- The initially supplied shortened external boot/skill paths were absent; rg located the actual files under AgentSystem/agent-memory/. Both exact resolved paths and hashes are recorded. Initial python commands failed because python is absent; all successful scripting used python3.
- Some exploratory tool displays were truncated. Frozen source copies and full probe logs remain available. Findings rely on inspected targeted source ranges and untruncated probe results, not omitted display text.

Phase 1 was frozen at `2026-10-15T14:14:39.716702+00:00` in `PHASE1.json`, SHA-256 `4664706fe1ebab89a50c6b95c22df4a35bce115f2693ab50a89fe060a9701ee9`. No later implementer unblinding occurred. The immutable freeze records the initial re-audit assessment; it does not establish blind isolation. Packaging correction: the log contains 14 top-level Go tests: 3 independent probes and 11 preexisting functions. PHASE1.json records 12 preexisting functions. That counting error is corrected here; findings and verdict are unchanged. The freeze remains byte-for-byte intact. Controller reproduction: `PYTHONDONTWRITEBYTECODE=1 python3 /tmp/hma-gate-a-20261015-r2/controller_probes.py`. Machine-readable details, inherited closure criteria and remedies remain in `AUDIT.json`. Structural validation is reported in `report-validation.log`; it is not proof of findings. End-of-audit source verification is in `target-verification.json`.

## Exactly one next action (§18)

**Ming revises the Gate A design to resolve the remaining findings and submits a frozen revision for re-audit.**

**Current outcome:** Gate A FAIL (design audit). **Rationale:** Illegal issuance stage, blocker fabrication, escalation authority and unresolved attempt/import protocols prevent approval of the full coding plan.

**Owner and proposed route:** Ming; Anthropic / Opus 5.5 / high, through the existing approved host route. Record exact access service/runtime and perform exact-route hi before any dispatch.

**Permission envelope:** Proposed documentation-only Gate A revision. No coding, Git writes, cross-project edits, permission expansion or release authority is granted by this audit.

**Exact approval required:** No approval is granted here. Obtain exact human authorization for corrected D1, chosen D3, D4 commit envelope and D5 external changes before dependent execution. D1 must preserve existing stage, finding and escalation invariants unless the human separately authorizes an explicit permissible amendment. Each future HMA transition retains its own bound approval.

**Stop/escalation conditions:** Do not start Gate B while Gate A blockers remain. Stop for unavailable eligible review, an unresolved safety conflict, or missing required human authority.
