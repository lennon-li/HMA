# Gate A v7 re-audit

**Gate verdict: PASS — design level only.**  
**Dispatch:** hma-gate-a-audit-20261015-r7. **Date:** 2026-10-15.  
**Isolation:** partial. This is the requested v1–v6-informed re-audit. It is not a blind first pass.  
**Requested reviewer:** Liz / OpenAI / gpt-6.1-sol / high / review mode. Actual runtime identity and coordinator preflight were not independently attested.  
**Modes:** plan, targeted code inspection, reasoning. **Threat model:** T, trusted host.

Observed: HMA remains on main at `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81`. The v6 manifest comparison has zero changed paths, zero removed paths, and one added path: v7. The final target check found no byte, mode, HEAD, status or HMA inventory drift. Obsidian HEAD is `9ba1e74a61ef5133f911903cd5c328666e2170ca`. All three requested Verify Loop artifacts match `9ba1e74a`.

Inference: v7 closes R11 and its dependent R4, A1, A2 and A6 at design level. No new design finding was identified. Code proof remains assigned to later gates. The current controller still lacks the planned authority hook.

The skill report contract forbids `verdict: pass` with partial isolation. `AUDIT.json` therefore records `verdict: conditional-pass` and `gate_verdict: PASS`. The qualification concerns audit isolation. It is not a new product defect. The proposed importer maps `conditional-pass` to no approval. This packet cannot authorize an HMA stage or satisfy final independent acceptance.

## Scope and frozen evidence

The original objective is `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md`, especially §§1–6. The normative sources are AGENTS.md, architecture §§4–7, 9–10, 12–22, Task 13, Task 14 and TODO.md. The evaluated design is v3, as amended by v4, v5, v6 and v7. Earlier auditor reports are continuity evidence. Author verdict claims are not evidence of closure.

| Artifact | SHA-256 / identity |
|---|---|
| HMA start manifest, 108 paths | `fbe57bca95c03df5153096d426a6c9300cf6e65fd4d130b3e2d9c81746a10f46` |
| v7 design bytes | `acf7ca239dd65f014c2512b4f85cf2d92fe0a6e54e60c7c9a82478f560804f4d` |
| Obsidian initial manifest | `e7200a11dc2aa2d73dce05fa0072f4b41b304d70bae4e7eb6a3a3b3bf48164e5` |
| Obsidian canonical-path supplement | `1bbb5195e24ed6d0a8d032d084e22683929e7e50d48d69f8b266fc6fa597df57` |
| Verify Loop controller | `be08b0d5460479e585bd942fb8eafd7c046100b210210d7a4e79a9c89c30f27c` |
| Frozen Phase 1 re-audit | `e8912d0d8e7c1a784104ca64712f2578c709382e18c476b533225380fb81e75f` |

The successful HMA freeze time is `2026-10-15T17:49:12.081567+00:00`. `snapshot/<repo>/blobs/<sha256>` preserves content. Manifests map paths to bytes and modes. `.serena/`, ignored files and Git internals are excluded. Obsidian scope is bounded to the requested artifacts, ledger uniqueness source and audit skill/validator. The initial shorthand test/skill paths were absent. Their canonical `AgentSystem/agent-memory/` paths are frozen in the supplement.

Phase 1 was frozen at `2026-10-15T17:52:26.382810+00:00`. No implementer explanation or controlled unblinding followed it. The report does not claim a blind assessment.

## 1. Findings and closure criteria

| Finding | Status | Observed design change and conclusion |
|---|---|---|
| R11, medium | **CLOSED (design level)** | v7:11–29 defines immutable generation identity and `min(original_expiry, all applicable revocation times)`. It computes this from the whole chain. Renewal cannot revive old authority. All six requested controls are assigned at v7:31–40. |
| R4 | **CLOSED (design level)** | The last residual lifetime ambiguity closes through R11. Full namespace reconciliation, one-to-one launch association, conservative charging and BLOCK on mismatch remain. T requires detection at import, not a tamper-resistant launcher. |
| A1 | **CLOSED (design level)** | Core-approved legal issuance, approved plan/policy/contract, unit/route binding and dependency invalidation remain. New generations cannot adopt old reservations. A host cache grants no authority. |
| A2 | **CLOSED (design level)** | The bridge design has a hook before every launch/resume, exact preflight, stable IDs and bound controller/live/baseline/namespace facts. R11 now governs historical reconciliation. D5 implementation is still future work. |
| A6 | **CLOSED (design level)** | Atomic, lifetime per-unit totals remain monotonic. Unknown and unlaunched reservations stay charged. Renewal cannot reset spending or restore expired authority. Existing final-slot replay has no second charge. |

R1, R6, R7, R8, R9, R10, A4 and A8 retain their earlier design closures. V7 does not amend their operative rules. `AUDIT.json.finding_statuses` retains the prior closure criteria and source references. `code_fixed` is false. The fixed revision is the uncommitted v7 digest, not commit `118ea3a`.

**New findings: none.** No new remedy or extra acceptance criterion is proposed.

R11's independent checks used these exact cases:

| Case | Observed model result |
|---|---|
| G1 issue10/end100, legal rewinds40/60, visible delayed launch50 | End40; launch unauthorized. The literal v6 assignment accepts this negative control. |
| All six orders of revocations40/60/80 | End40 in every order. |
| Expiry20, delayed launch30, later policy revocation40 | End20; launch unauthorized. |
| 625 bounded expiry/event combinations | Adding an event never increases the end. |
| G2 approval70, old G1 reservation launch75 | Refused. G2 cannot revive or adopt it. A fresh G2 reservation admits the valid control. |
| Launch exactly at earliest end40 | Refused. Launch39 is accepted when other predicates hold. |
| Review original TTL25, later revocations40/60 | End25. The original TTL is preserved. |

These are literal specification models in `probes.py`. They are not tests of an implemented HMA importer. They meet the requested Gate A design-check boundary. Gate C must execute the named import controls. Gate D must prove generation integration with real approvals.

## 2. Approval and digest compatibility

No intrinsic conflict with architecture §6 was found. V7 adds no approval field. V3:28–36 removes `execution_lease_digest` and retains only optional `acceptance_contract_digest`. V4:44–49 puts `policy_digest` inside the plan already bound by `accepted_plan_digest`. `findings_snapshot` is an additive record flag.

Architecture:192–200 permits an optional exact worktree binding. Task 13:37–40 and `internal/model/types.go:163–183` show additive omitted-field precedent. `internal/model/terminal_outcomes_test.go:51–87` protects exact legacy approval and record bytes. `internal/store/jsonl.go:40–49` hashes marshalled record bytes. Equivalent JSON meaning is insufficient if serialization changes.

Required implementation conditions remain:

- Omit empty new fields. Preserve old field order, bytes and hashes. Do not impose a new contract on legacy approvals. Legacy runs without the required plan cannot derive a new lease.
- For contract runs, require the contract digest at Acceptance criteria → Planning and every later approval. Include populated fields in equality, transition/record hashes and freshness checks. Changed statements, checks or thresholds require reauthorization.
- Preserve every existing single-use, nonce, repository/base, stage-time, plan, evidence, waiver and applicable produced-artifact binding. No new legal edge or automatic transition is allowed.
- Prepare and authorize the exact normative amendment before code depends on the changed semantics. D1 approves direction. The 2026-09-08 precedent is not blanket amendment authority.
- Document reader/version compatibility. Old strict readers may reject records that contain populated new fields.

Nine selected existing model/engine Go test functions passed. This verifies present legacy, worktree, approval, legal-edge, no-auto-advance and resolution controls. It does not validate fields that do not exist yet.

## 3. Attack sequences under T

No sequence was found that both follows the v7 design and permits the requested closure bypass. A trusted host can omit a call accidentally and launch before detection. T explicitly requires detection at import. It does not require HMA to prevent a same-UID hostile host.

| Attempt | Concrete sequence | Required result under v7 |
|---|---|---|
| Self-issue lease | Write host lease L without a valid unit core issuer; alternatively copy another unit/plan approval. | Derivation fails. No authority. A visible unreserved launch creates `UNAUTHORIZED_DISPATCH` BLOCK at import. |
| Count attempts wrongly | Reserve last slot I; lose reply; replay; then reuse after ledger I exists or request new ID; finally renew. | I charges once. Valid unused replay can launch once. Used/new exhausted IDs are refused. Renewal keeps spent totals. Unknown/unlaunched slots stay charged. Legitimate reviews are separate. |
| Revive old lease | Issue G1 at10/end100; rewind40; omit hook and visibly launch50; rewind60; approve G2 at70; import. | G1 end40. Launch50 creates BLOCK. G2 cannot authorize it retrospectively. |
| Import stale review | Complete current review but reserialize old PASS/F1-resolution and include its correct file hash in the current receipt. | Old packet digest/dispatch ID/nonce fails body equality. F1 stays open; criterion credit stays zero. |
| Import self review | Use an implementation ledger run or same worker/model/context; for high/disputed/final work use the same provider/model family. | Required identity, role and independence checks refuse acceptance. |
| Hide BLOCK | Import F1 BLOCK, then F2 ADVISORY; then submit stale or unauthenticated empty snapshot. | Cumulative import retains F1. Only a fresh eligible explicit independent resolution can remove it. Kernel downgrade stays refused. |

The current controller at `9ba1e74a` remains a future integration dependency. `validate_envelope` at lines363–433 accepts an HMA marker without a hook. Route normalization drops provider/profile/access-service keys. `dispatch_agent` at lines831–861 does not call a supplied false hook. The audit observed that path with `Popen` fully mocked. No worker ran. D5 must implement start refusal, hook-before-launch/resume and `launch_eligible:true` handling. These gaps were already assigned to Gate C.

The controller uses a private `dispatch-ledger` namespace at line854. The future admission/import must bind that namespace. The ledger uses atomic directory creation to reject duplicate exact launch IDs (`agent-run-ledger`:738–747). The audit inspected this source but did not run the dispatcher.

## 4. Regressions and decision completeness

No new wrong design was found. The minimum reducer narrows authority and preserves original review TTL. Generation identity prevents renewal from extending an old record. The half-open interval preserves refusal at ties. Replay preserves the spent final slot. No stage, approval channel, finding downgrade or host dispatch power is added.

The recorded decisions cover all identified product-decision categories before Gate B: T, D1 amendment direction, D3 Phase B, D4 conditional main commits, D5 hook work and D6 receipt work. This dispatch supplies D2 review authority. V7 needs no new D7 category.

These records are not a complete concrete Gate B authority packet. The exact unit, route, permission envelope and each stage transition still need authentic bound approvals. Exact normative text must be prepared, reviewed and authorized before implementation depends on changed semantics. Authorized existing-defect remediation need not wait for all Gate D documentation. D4 permits commits only after the corresponding gate audit and full tests. Push needs explicit approval.

## Requirement coverage and verification

| Question | Result within scope | Evidence |
|---|---|---|
| Q1 closure | PASS, design only | v7:9–40; finding table; `probes.log` |
| Q2 compatibility | PASS, with stated implementation obligations | architecture §6; model/engine/store sources; `go-checks.log` |
| Q3 bypasses under T | No conforming bypass found | retained v3–v6 import/authority rules; attack table; pure/mocked checks |
| Q4 regressions | None identified | v3–v7 comparison; current controller inspection; prior manifest comparison |
| Q5 decisions | Categories sufficient; exact authority still required | v4 §0; architecture §§4–6; current dispatch |

| Executed check | Result | Raw evidence |
|---|---|---|
| Selected Go model/engine controls | 9 top-level test functions passed; 2 packages | `go-checks.log`, exact argv/environment in `go-checks.json` |
| Current-controller pure/mocked tests | 7 passed | `controller-unit-checks.log`, `controller_unit_checks.py` |
| Independent specification/mocked observations | 42 checks passed | `probes.log`, `probe-results.json`, `probes.py` |
| Three controller artifacts vs commit9ba1e74a | All byte-identical | `obsidian-commit-comparison.json` |
| Final target and snapshot integrity | All unchanged | `target-verification.json`, `verify_target.py` |

The commands and evidence inventory are in `AUDIT.json.verification` and `evidence_inventory`. Helpers, logs, snapshots and manifests remain under this output directory. JSON report validation checks form only; see `report-validation.log` for its actual result.

## Limits and later-gate exit criteria

Both repositories were read-only by procedure. No Git write, nested agent CLI or model API ran. The first `python` attempt failed because that command is absent; `python3` completed the freeze. Canonical controller test/skill paths were located and snapshotted after the shorthand paths were absent. Complete relevant source bytes remain available despite truncated broad terminal displays.

No full suite, production crash/concurrency integration, real dispatch, quota/billing measurement, final release gate or end-to-end dogfood ran. Gate A does not establish runtime correctness. Original Gate B findings H1–H4, M1–M6, L1/L3 and N1/N2 retain their original qualifications. This report does not certify them fixed. Phase B closure/release work remains in scope through D3.

The original human dialogue behind recorded D1/D3/D4/D5/D6 was not supplied. Current T is directly confirmed by this request. The actual reviewer runtime and coordinator preflight were not attested. Partial isolation prevents use of this report for final independent acceptance. No approved Jev tool was exposed; auditor judgment is the recorded fallback.

The nonce binds the current packet under T. It does not establish good reasoning or defeat a malicious same-UID receipt writer. Snapshot hashes establish integrity, not authority or blinding. No claim extends to a hostile OS or hidden hostile-host ledger.

No design remediation remains from this re-audit. The implementation roadmap is unchanged: Gate B fixes baseline defects; Gate C proves reservation/import/D5/D6 controls; Gate D proves lease/contract/generation integration; Gate E proves checkpoint and final acceptance enforcement; Gate F supplies exact-revision real dogfood and eligible final independent review. Each gate retains its original evidence and human approval requirements.

## Single next action — architecture §18

**CONTINUE. Owner: Ming.** Prepare the first bounded Gate B existing-defect remediation unit and its exact approval packet. Execute it only under applicable authentic approvals.

Rationale: v7 meets the last open Gate A design closure criterion. Proposed route: Ming / Anthropic / `claude-opus-5-5` / high / Claude Code CLI through the existing approved access-service profile. Confirm exact eligible route/service and run `hi` before dispatch.

Permission envelope: approved Gate B defect scope and the concrete per-unit file/tool/cost limits. This audit grants no source or Git-write authority. Exact approval: human Gate A design acceptance where required by the run, and the unit/route/stage challenge-bound approvals in architecture §§4–6. Existing valid approvals need not be requested again. Authorize concrete normative text before code depends on changed semantics.

Stop and escalate to Lennon for missing/stale authority, open kernel findings, unavailable/indeterminate routes, changes to acceptance/risk/permissions/budget, or conflicts with architecture. This report makes no HMA transition.
