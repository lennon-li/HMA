# HMA TODO

This file tracks implementation work that has not yet been folded into the normative architecture contract. `docs/architecture.md` remains the product-contract source of truth.

## Interactive orchestration and independent review

These tasks implement `docs/architecture.md` §12.4. They belong to the host
integration layer; HMA core remains non-dispatching in v1. `hma host ...`
provides host-local records and partial transition gates. It does not complete
interactive-host MVP conformance: the host still needs to establish approved
route/task bindings and perform the orchestration sequence, and HMA does not
enforce every direct-cost exception condition in §12.4.

### P1 — HMA-side record and gate subset (implemented; MVP conformance remains open)

- [x] **Add a versioned orchestration-decision record.** Record `DELEGATE` or
  `DIRECT_COST_EXCEPTION`, the bounded job, estimated direct and delegated total
  costs, comparison basis, approved permission envelope, decision timestamp,
  and actor. Reject an unrecorded direct-coding path.
- [x] **Define and record the route preflight handshake.** Before each dispatch,
  send `hi` to the exact approved profile through the approved access service
  and bind the response to profile/provider/model/access-service digests and a
  freshness timestamp. HMA records and verifies the host-observed handshake; it
  does not send the network/client message itself.
- [x] **Fail closed on unavailable or indeterminate routes.** An absent response,
  route mismatch, rate limit, or exhausted quota must prevent dispatch and
  return to an explicit superseding route decision or human route decision.
  Never substitute silently.
- [x] **Emit dispatch telemetry before work starts.** Report worker identity,
  provider, model, reasoning/effort level, access service, runtime, bounded
  task, and permission envelope.
- [x] **Add a mandatory independent-review binding for coding work.** Source,
  test, executable-configuration, build, and CI changes cannot progress from
  implementation review to verification without a reviewer record from a
  distinct agent and separate review context. Treat direct work by the
  orchestrator as implementation for this rule.
- [x] **Enforce independence by risk.** Require a different model for normal
  coding work and the §13 provider/model-family separation for high-risk and
  critical work; prohibit self-review and silent reviewer substitution.
- [x] **Add conformance fixtures.** Cover delegated work, a valid direct-cost
  exception, an unjustified direct path, successful and failed `hi` preflights,
  quota exhaustion, route mismatch, escalation, orchestrator self-review,
  same-model review, high-risk same-provider review, fixed-packet mismatch,
  valid independent review, coding-stage enforcement, and host-record tamper
  detection.

See `docs/task14-host-orchestration.md` for the executable contract and first
Fury pilot sequence.

Open before interactive-host MVP conformance:

- [ ] Bind recorded unit, task, route, permissions, and route approval to the
  actual approved HMA plan/route state. A non-empty approval digest alone does
  not establish that binding.
- [ ] Enforce every §12.4 direct-cost exception condition: bounded/small task,
  approved permission envelope, route eligibility, and a total-cost comparison
  that includes setup, duplicated context, execution, and review coordination.
- [ ] Complete and exercise the host flow around the HMA records: route choice
  after deterministic eligibility checks, real exact-route preflight before
  every dispatch, artifact return, fixed review packet, and explicit human
  escalation when a route or result is indeterminate.
- [x] Preserve freshness across core-run changes. Transition gates reject
  host evidence bound to an earlier core-chain head.
- [ ] Add client-neutral adapter hooks and conformance coverage for an
  interactive CLI host; the HMA core must remain non-dispatching.


### Post-MVP — learned decision-provider integration (optional)

This work is intentionally outside the MVP acceptance boundary. The MVP host
may make soft route/result/next-step judgments itself or return them to a human,
provided deterministic HMA legality, approval, permission, and independence
rules remain authoritative. Keep the provider-neutral seam so a learned
decision provider can be added later without changing portable core semantics.

- [ ] **Add a provider-neutral decision trace schema.** Bind provider/model
  identity, state digest, typed question contract, probabilities/confidence,
  threshold-policy version, selected host action, and any override rationale.
- [ ] **Implement an optional Jev host adapter.** Support an approved Jev MCP/SDK/HTTP
  path without adding provider-specific credentials or endpoints to portable
  HMA core.
- [ ] **Optionally use Jev at the three execution checkpoints.** Pre-dispatch route choice,
  post-worker result sufficiency, and pre-next-step continue/retry/replan/
  escalate/stop. Deterministic HMA eligibility and legality checks run first.
- [ ] **Add discretionary-idea triage.** Agent-generated optional implementation
  ideas should be classified as implement/prototype/defer/reject before they can
  expand scope; an explicit human request is not vetoed by this gate.
- [ ] **Calibrate confidence policy from recorded outcomes.** Until calibrated,
  low-confidence or close Jev decisions escalate rather than force an automatic
  soft judgment.
- [ ] **Add Jev failure fixtures.** Missing tool, timeout/rate limit,
  indeterminate answer, low confidence, and attempted illegal-choice expansion
  must preserve hard policy and return control to the orchestrator/human.
- [ ] **Keep any learned provider advisory.** A provider result cannot approve a transition, waive a
  rule, satisfy evidence, replace independent review, widen permissions, or
  authorize commit/push/release.


### P2 — host adapters

- [ ] **Implement client-neutral adapter hooks** for orchestration decision,
  preflight, dispatch reporting, artifact return, and independent-review return.
  Codex, Claude Code, OpenCode, Cursor, Hermes/Fury, and other clients may
  translate these hooks but may not change their semantics.
- [ ] **Measure estimated versus observed dispatch cost** so the direct-cost
  exception can be calibrated without becoming a convenience bypass.

## Reverse-skill-inspired routing hardening

Reference reviewed: `zhaoxuya520/reverse-skill` (`AGENTS.md`, 2026-08-30 review). Useful patterns are its routing single source of truth, routing regression benchmark, routing-coherence checks, generated tool inventory, and client-neutral adapters. HMA should adopt the patterns, not the security-specific skill framework or bootstrap behavior.

These items extend HMA's existing route-selection, host-local profile, evidence, independence, and deterministic-fixture contracts. They should not create a second routing system.

### P1 — before route selection is considered implemented

- [x] **Define one versioned route-policy data contract / normalized projection as the routing SSoT.** It should operate on HMA's portable capability taxonomy and policy/risk inputs, not hard-code provider-specific client names, credentials, private paths, or local aliases. A route-policy change must produce a new digest.
- [x] **Add table-driven routing conformance fixtures.** Each row should bind plan-unit/task facts, risk, required capabilities, available/eligible host profiles, permission envelope, and relevant policy to exactly one expected primary route, one escalation route, validator-independence requirement, and forbidden fallbacks. Include ambiguous, unavailable-capability, high-risk, and no-eligible-route cases.
- [x] **Add a deterministic route-coherence validator.** At minimum verify: referenced capabilities exist; exactly one primary and one escalation route are proposed; escalation is independently eligible; implementer cannot validate itself; required model/provider-family independence is satisfied; permission bounds are not widened; and no silent fallback or access-service substitution is possible.
- [x] **Implement a host-local machine-truth inventory from allowlisted discovery only.** (`hma inventory`; not yet consumed by route verification.) Capture executable presence, version, model-list/availability metadata, timestamp/freshness, and a digest. Keep it credential-free and outside portable repository artifacts. Preserve the architecture rule that availability is evidence only and is not proof of eligibility, trust, isolation, quota, or capability.

### P2 — after the deterministic evaluator/harness nucleus exists

- [ ] **Make routing regression part of the deterministic conformance suite and CI.** A routing-policy change should fail existing expected-route fixtures unless the changed expectations are reviewed explicitly.
- [ ] **Define a client-adapter boundary for Codex, Claude Code, OpenCode, Cursor, and other hosts.** Adapters may translate HMA's approved route/permission attestation into host-specific invocation, but client behavior must not become routing policy or alter evaluator semantics.
- [ ] **Add route reproducibility checks.** The same approved harness contract + policy digest + machine-truth snapshot must produce the same route proposal and rationale classification.
- [ ] **Add negative fixtures for stale or contradictory machine truth.** Missing/stale discovery, profile-policy mismatch, or unavailable independent validator must produce an explicit `BLOCKED`/`UNKNOWN`/human-decision path rather than a guessed or silent substitute.

### Integration with existing HMA work

- Treat routing conformance as an extension of the deterministic evaluator and end-to-end decision-table work identified in audit Findings 12, 15, and 16; do not implement it as a separate prompt-driven router.
- Reuse Section 12's portable capability taxonomy and host-local profiles instead of creating a second tool/profile schema.
- Reuse HMA's existing Evidence -> Finding -> Next Action -> Verification/Validation chain rather than importing reverse-skill's case/evidence object model.
- Keep human approval and HMA's authority boundaries authoritative. Routing may recommend; it must not dispatch, widen permission, waive independence, or advance a stage in v1.

### Explicit non-goals

- [ ] Do **not** import reverse-skill's security-specific skill catalog, ACT authorization semantics, global/bootstrap prompt injection, or automatic tool installation into HMA core.
- [ ] Do **not** make a generated tool index or model list a trusted eligibility source.
- [ ] Do **not** duplicate routing truth across Markdown instructions, adapters, and executable policy data.

### Done when

Routing hardening is complete for the MVP when table-driven fixtures prove that a fixed harness/policy/machine-truth input deterministically yields one route proposal, one escalation path, the required independent validator, bounded permissions, and an explicit no-route outcome when eligibility cannot be proven.
