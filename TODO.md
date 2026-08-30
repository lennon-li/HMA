# HMA TODO

This file tracks implementation work that has not yet been folded into the normative architecture contract. `docs/architecture.md` remains the product-contract source of truth.

## Reverse-skill-inspired routing hardening

Reference reviewed: `zhaoxuya520/reverse-skill` (`AGENTS.md`, 2026-08-30 review). Useful patterns are its routing single source of truth, routing regression benchmark, routing-coherence checks, generated tool inventory, and client-neutral adapters. HMA should adopt the patterns, not the security-specific skill framework or bootstrap behavior.

These items extend HMA's existing route-selection, host-local profile, evidence, independence, and deterministic-fixture contracts. They should not create a second routing system.

### P1 — before route selection is considered implemented

- [ ] **Define one versioned route-policy data contract / normalized projection as the routing SSoT.** It should operate on HMA's portable capability taxonomy and policy/risk inputs, not hard-code provider-specific client names, credentials, private paths, or local aliases. A route-policy change must produce a new digest.
- [ ] **Add table-driven routing conformance fixtures.** Each row should bind plan-unit/task facts, risk, required capabilities, available/eligible host profiles, permission envelope, and relevant policy to exactly one expected primary route, one escalation route, validator-independence requirement, and forbidden fallbacks. Include ambiguous, unavailable-capability, high-risk, and no-eligible-route cases.
- [ ] **Add a deterministic route-coherence validator.** At minimum verify: referenced capabilities exist; exactly one primary and one escalation route are proposed; escalation is independently eligible; implementer cannot validate itself; required model/provider-family independence is satisfied; permission bounds are not widened; and no silent fallback or access-service substitution is possible.
- [ ] **Implement a host-local machine-truth inventory from allowlisted discovery only.** Capture executable presence, version, model-list/availability metadata, timestamp/freshness, and a digest. Keep it credential-free and outside portable repository artifacts. Preserve the architecture rule that availability is evidence only and is not proof of eligibility, trust, isolation, quota, or capability.

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
