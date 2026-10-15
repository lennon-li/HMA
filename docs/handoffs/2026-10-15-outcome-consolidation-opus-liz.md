# HMA consolidation and outcome-driven execution — implementation handoff

**Date:** 2026-10-15  
**Repository:** `lennon-li/HMA`, `main`  
**Lead implementer:** Opus 5.5 (high effort)  
**Independent auditor:** Liz — Sol 6.1 High  
**Goal:** One coherent, evidence-backed outcome lifecycle, not parallel orchestration, verify-loop, and audit-loop systems.

## 0. Read and reconcile existing truth first

Read `AGENTS.md`, `README.md`, `TODO.md`, `docs/architecture.md` (especially §§12.4, 13, 18, 19), `docs/task13-terminal-outcomes.md`, `docs/task14-host-orchestration.md`, current code and tests. Locate the canonical Obsidian **verify loop skill** and **audit skill** in the current agent-memory/skills installation; inspect their *actual, current* contracts and implementations before proposing reuse. Do not claim these skills are integrated based on their names.

Inventory every open architectural, functional, safety, host, test, and documentation gap and reconcile findings against current code. In particular investigate task/route/approval bindings, direct-cost exception enforcement, end-to-end host execution, evidence freshness, crash recovery, verified-success closure, route-coherence and projection integrity. These are candidate checks, not assertions that every item is broken. Check whether the recent host dogfood harness is committed or still local, and preserve any valid existing work. Produce a gap/evidence matrix with reproducible checks.

**Important boundary:** Existing HMA v1 is a *human-gated, non-dispatching* Go safety kernel; it never grants approvals or dispatches agents. Human approval is mandatory for every HMA stage transition. Do not silently convert it into an autonomous executor. Implement extended execution leases only in a host/orchestrator layer within preapproved scope. Any change in human-approval semantics is a proposed architecture change requiring explicit human authorization, not an assumption.

## 1. Outcome contract — establish a single source of truth

Define a versioned, machine-readable outcome contract with: stable outcome/run ID; user intent and scope; observable deliverables; explicit acceptance tests and thresholds; baseline/revision; risk; permitted tools/routes/permission envelope; stage gates; dependencies; time/token/cost/retry budgets; evidence requirements; escalation conditions; human decisions; and immutable policy/contract digests. An agent may propose clarifications but cannot change approved success criteria, audit policy, scope, risk, or permissions unilaterally. A contract revision invalidates affected approval/evidence and requires human reauthorization where existing HMA policy demands it.

Define **two distinct completion measures**: (1) implementer/test-reported completion and (2) independently verified acceptance completion. Count only independently verified criteria in the primary outcome completion rate. Track reopened findings, repeated failed attempts, cost and elapsed time per verified outcome. Do not silently treat tests passing, a review approval, or a completed checklist as final success.

## 2. One lifecycle; separate responsibilities

Use one HMA-owned run/evidence/finding/decision state model. Do not create a second task tracker or a second gate state machine.

- **HMA core:** deterministic legality, evidence/provenance, route and independence constraints, challenge-bound human approvals, stage/outcome projection and closure rules.
- **Host/orchestrator:** long-leash work execution, bounded dispatch, retries, notifications, and adapters. HMA v1 remains non-dispatching.
- **Verify-loop skill:** execution-level test/diagnosis/remediation/retest methodology. It supplies evidence; it does not mark an HMA outcome accepted.
- **Audit skill:** independent, adversarial evaluation against frozen artifacts/contract, findings with severity and reproducible evidence; no self-certification.
- **Liz / independent reviewer:** independently challenges architecture, fixes, criteria coverage, evidence freshness, and final closure. Liz must not implement work she audits.
- **Human:** approves stage transitions and changes to risk/scope/permissions per the existing normative contract.

Flow: approved outcome and permitted envelope → implement within leash → tests/verify loop → freeze review packet → independent audit at risk-based checkpoint → remediate and reverify findings → stage approval as required → final adversarial audit → approved final closure or explicit FAILED/BLOCKED/UNKNOWN/PARTIAL/ABORTED. No unsupported auto-advancement. An audit failure creates linked findings and rework, not a fresh unrelated task.

## 3. Meaningful independent audit policy

Define a deterministic, versioned checkpoint rule driven by **risk and changes to outcome semantics**, not an arbitrary number of commits. Mandatory checkpoint triggers: modified acceptance criterion or permission boundary; high/critical-risk changes; public interface or persisted schema/protocol change; irreversible/external effect; security/approval/routing/evidence/closure logic; completion of an approved material milestone; and final outcome closure. A checkpoint's frozen packet must bind exact revision and diff, scope, criteria, evidence digests, reviewer identity/model/provider/context separation, and policy version. Non-triggering low-risk edits may be batched between checkpoints with ongoing local tests.

Require independent review for **every coding change**, consistent with existing HMA requirements; a meaningful checkpoint is an additional *stage-level adversarial audit*, not a loophole allowing unreviewed coding changes. Preserve high-risk provider/model-family separation. No implementer self-review, no reviewer silently replaced by the same route, no evidence-free signoff.

Specify a configurable *more stringent* policy (e.g. independent stage audit after each implementation batch) but never permit configuration to weaken HMA safety invariants. Any relaxation of human approval or baseline review requirements must be surfaced explicitly to the human.

## 4. Execution leash, convergence, and stopping

Define delegated autonomy as a **bounded lease**, not blanket permission: exact approved outcome/subtask scope, routes, tool and filesystem/network permissions, max retries, cost/token/time budget, allowed reversible changes, and explicit forbidden actions. The host may iterate implementation→verification→remediation without asking for step-by-step human instructions *within* the lease. The lease cannot approve HMA stage transitions, enlarge its permissions, change the acceptance contract, bypass independent review, silently switch models/routes, or release/deploy. Expired, stale, contradictory, exhausted or indeterminate authority fails closed and escalates.

Define convergence by reduction of unresolved closure-blocking findings and progress on independently verified criteria. Set stop conditions: all acceptance criteria independently verified and eligible for human closure approval; budget/retry exhausted; no progress after bounded attempts; unsafe change; missing evidence; route unavailable; scope change; or required human decision. Emit explicit truthful status rather than repeating audit cycles indefinitely.

## 5. Ordered work and mandatory independent checkpoints

**Gate A — architecture audit, before coding:** Opus documents current behavior, actual gaps, proposed state/contract changes, compatibility impact, skills mapping and a test plan. Liz independently critiques against `docs/architecture.md`. Resolve blockers and obtain needed human approval for product-contract changes.

**Gate B — existing defect remediation:** Opus fixes demonstrable defects first; test crash/restart, concurrent writes, evidence/stage projection, exact approval binding, invalidation, terminal-state coherence, and host dispatch/review legality as applicable. Run verify loop. Freeze evidence; Liz independently audits high-risk safety changes before proceeding.

**Gate C — consolidation:** Integrate existing verify-loop and audit-skill interfaces into HMA's evidence/finding model *without copying or replacing their independent methodology*. Build test fixtures for audit failure→linked remediation→re-verification and review separation. Liz audits integration and migration/compatibility.

**Gate D — outcome layer and leash:** Implement versioned outcome contracts, completion metrics, bounded host execution and explicit escalation. Preserve the HMA human-gated core. Liz checks that the host cannot forge a stage decision or broaden an approved envelope.

**Gate E — checkpoint policy:** Implement deterministic meaningful-change detection, immutable audit packets, mandatory per-coding-change review, risk-based audits and final audit enforcement. Test refusal of stale/missing/self-review and false completion. Liz validates.

**Gate F — final end-to-end dogfood:** Use this HMA project itself as the real case, with Opus implementing and Liz independently auditing. Run actual binaries/commands and negative fixtures, including induced audit finding, remediation, repeat audit, interrupted execution, stale approval, altered criteria, expired lease, quota failure, and honest non-success terminal results. Require reproducible traces rather than narrative success. Liz performs final adversarial audit of exact final commit and evidence; only the human can grant required closure authorization.

After any failed gate, Opus addresses findings, reruns verification, and Liz reviews the affected evidence. Never rebaseline criteria to make the failure disappear.

## 6. Deliverables and acceptance

Deliver: (a) gap baseline and architecture decision record; (b) updated normative architecture and user-facing command/host contract; (c) minimal core/host changes with tests; (d) versioned outcome/checkpoint/lease schemas with migration rules; (e) skill integration mapping, avoiding duplicate state; (f) executable end-to-end dogfood harness; (g) reproducible raw evidence and separate Liz audit reports at each gate; (h) final outcome scorecard and short human-readable decision packet.

Minimum proof:
1. An outcome can be expressed with objective acceptance criteria and budgets.
2. A host can execute multiple permitted implementation/test/fix iterations without human micro-management **but cannot advance HMA stages without authentic bound approval**.
3. Every coding change gets an independent reviewer, meaningful changes trigger stage audits, and final closure requires independent final audit.
4. Rejection or findings block completion, link to repairs and require fresh verification; stale packets and self-review are rejected.
5. True completion, partial completion and failure states are distinguishable; no manufactured VERIFIED_SUCCESS.
6. A real reproducible end-to-end dogfood run and negative tests pass; the independent auditor confirms the exact final revision.
7. No duplicative verify/audit state machine, undocumented safety downgrade, or unapproved external side effect.

## 7. Instructions to the two agents

**Opus (lead):** Start with inspection, not an assumed rewrite. Execute work in ordered gates. Delegate bounded mechanical subtasks if policy permits, but remain responsible for integration. Invoke existing verify-loop and audit skills where available. Produce exact commands, outputs, commit SHAs, evidence digests and a checkpoint packet for Liz. Work on the existing main/master per repository norms; **do not open a PR or create an unnecessary branch**. Do not claim Liz reviewed anything until her distinct model/session actually did. Commit/push only under valid authority; do not treat this document as a blanket approval bypassing HMA gates.

**Liz — Sol 6.1 High (independent auditor):** Review against current normative docs and code, not Opus's summary. Challenge architecture, security invariants, outcome metrics, independence binding, meaningful-stage detection, safety-gate semantics, failures and dogfood evidence. Reproduce tests and try adversarial/negative scenarios. Record each finding with severity, exact reproduction, expected versus actual, closure criterion and fixed revision. Report PASS / FAIL / BLOCKED / UNKNOWN per gate; never silently waive a finding or implement the fix yourself. Final approval is a recommendation to the human, not a machine stage transition.

**Escalate to the human only for:** revised authority/approval semantics, changes to approved success criteria or risk, permission/cost budget expansion, incompatible product decisions, unresolvable safety conflicts, and final human acceptance where required. Other routine implementation choices are delegated within the approved envelope.

**Final deliverable:** "Outcome in → bounded autonomous execution → tests and independent evidence → human-gated verified closure," using **one** HMA system and existing skills, not a new fourth framework.
