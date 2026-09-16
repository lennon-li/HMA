# HMA TODO

This file tracks implementation work that has not yet been folded into the normative architecture contract. `docs/architecture.md` remains the product-contract source of truth.

## Interactive orchestration and independent review

These tasks implement `docs/architecture.md` §12.4. They belong to the host
integration layer; HMA core remains non-dispatching in v1. The HMA-side P1
contract is implemented by `hma host ...`; individual CLI clients still need
the P2 adapter hooks before they are automatically HMA-conforming.

### P1 — HMA-side contract complete; required before an interactive CLI host is HMA-conforming

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
