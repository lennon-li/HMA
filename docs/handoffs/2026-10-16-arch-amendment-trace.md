# Architecture amendment trace — Gate A approved design

**Date:** 2026-10-16
**Dispatch ID:** hma-arch-amendment-draft-20261016-r2
**Scope:** paragraph-level provenance mapping for `docs/handoffs/2026-10-16-arch-amendment-draft.md`.

| Paragraph | Source packet | Source section | Notes |
|---|---|---|---|
| §6 — `acceptance_contract_digest` omitted when empty | v3 | §2.1 | Additive field preserving legacy serialization |
| §6 — required on `ACCEPTANCE_CRITERIA -> PLANNING` and carried by later approvals | v3 | §2.1 | D1 approved |
| §6 — mismatch refuses transition | v3 | §2.1 | D1 approved |
| §9 — versioned `hma.acceptance-contract/v1` | v3 | §2.1 | D1 approved |
| §9 — contract binds criteria, non-goals, risk, routes, permission envelope, policy digest | outcome consolidation | §1 | Also v3 §2.1 route/permission ceilings |
| §9 — contract revision invalidates affected approvals/evidence | outcome consolidation | §1 | Precedence rule |
| §9 — contract digest bound to approvals as `acceptance_contract_digest` | v3 | §2.1 | Cross-reference to §6 |
| §10 — `hma.plan/v1` canonical JSON document | v3 | §2.2 | Lease source |
| §10 — plan digest is existing `accepted_plan_digest` | v3 | §2.2 | No new approval field |
| §10 — plan contains `policy_digest`, per-unit ceilings, dependency graph | v3 | §2.2; v4 | §2 adds policy_digest |
| §10 — lease derives from latest approved transition targeting `IMPLEMENTATION_AUTHORIZATION` | v4 | §1 | R1 closure |
| §10 — legal incoming edges are `ROUTE_SELECTION` and `INDEPENDENT_VALIDATION` | v4 | §1 | R1 closure |
| §10 — host presents plan; HMA recomputes digest; host cache grants nothing | v3 | §2.2 | A1 closure |
| §10 — active-lease conditions (stage, rewind, wall time, counters, policy, contract) | v3 | §2.2; v4 | §2 adds policy digest |
| §10 — new lease generation; spent counters carry forward; old reservations not revived | v7 | §1 | R11 closure |
| §10 — each record has `valid_from` and `valid_until` | v6 | §2; v7 | §1 R11 |
| §10 — `valid_until = min(original_expiry, all applicable revocation times)` | v7 | §1 | R11 closure |
| §10 — applicable revocation events list | v7 | §1 | R11 closure |
| §10 — computed from whole chain; only decreases; ingestion-order independent | v7 | §1 | R11 closure |
| §10 — half-open interval `valid_from <= t < valid_until` | v6 | §2; v7 | §1 |
| §12.4 — reservation hook before each launch/resume | v4 | §4; v3 | §3.2 mandatory hook |
| §12.4 — hook request fields (controller run, attempt, dispatch_run_id, envelope, route, failure signature, root, baseline, namespace) | v4 | §4 | R4 closure |
| §12.4 — replay-first rule and `HOST_RESERVATION_CONFLICT` | v4 | §3 | R7 closure |
| §12.4 — admission rule (`escalate_to == stop`, route equality, envelope ≤ remaining) | v4 | §3 | R7 closure |
| §12.4 — consumption rule (active lease, attempts ≥ 1, failure-signature rule) | v4 | §3; v3 | §3.2 same-failure twice |
| §12.4 — unknown launch counted, no cancellation, monotonic counters | v3 | §3.2 | R4/A6 closure |
| §12.4 — `reservation_id` and `launch_eligible` response semantics | v4 | §4; v5 | §2; v6 §3 final-slot replay |
| §12.4 — launch classes (implementation vs review) | v5 | §1 | R9 closure |
| §12.4 — `review_dispatch` record contents | v5 | §1 | R9 closure |
| §12.4 — no match / double match → `UNAUTHORIZED_DISPATCH` (BLOCK) | v5 | §1; v6 | §4 |
| §12.4 — review write mode or duplicate ID → unauthorized | v5 | §1 | R9 closure |
| §12.4 — full namespace scan at import and validity interval check | v5 | §2; v6 | §4; v7 §1 monotone end |
| §12.4 — unlaunched reservation stays charged | v6 | §4 | R10/A8 closure |
| §12.4 — verify-loop journal/admission equality checks | v6 | §4 | R10 closure |
| §14 — HMA generates review packet and records digest before dispatch | v4 | §5 | R8 closure |
| §14 — packet contains `review_nonce` and `review_dispatch_id` | v6 | §1 | R8 closure |
| §14 — review import preconditions (receipt target, mode, exit 0, namespace, report hashes) | v4 | §5; v5 | §3 R8/R9 closure |
| §14 — report JSON must contain matching `packet_sha256`, `review_dispatch_id`, `review_nonce` | v6 | §1 | R8 closure |
| §14 — reviewer not an implementation reservation | v5 | §1 | R9 closure |
| §14 — total verdict map (`pass`, `fix-required`, everything else → escalate) | v5 | §3 | R8 closure |
| §16 — additive `findings_snapshot` flag and replacement semantics | v3 | §4.2 | R5 closure |
| §16 — empty snapshot only from authentic independent review | v3 | §4.2; v4 | §5 |
| §16 — severity never sets disposition | v3 | §4.1 | R2 closure |
| §16 — §9-exact disposition mapping (kernel → BLOCK, criterion fail → no credit, complexity → WAIVABLE, else ADVISORY) | v3 | §4.1 | R2 closure |
| §16 — fabricated/non-reproducing criterion link → ADVISORY | v3 | §4.1 | R2 closure |
| §16 — checkpoint trigger list and unclassifiable-path trigger | v3 | §5; outcome consolidation | §3 |
| §16 — disputed validation uses high-risk independence rule | v3 | §5 | A5 closure |
| §20 — findings snapshot applied and no unresolved BLOCK/WAIVABLE remains | v3 | §4.2 | Snapshot interaction with closure |
| §22 — trusted-host assumption (decision T) | v4 | §0 | Detects accident/drift, not same-UID adversary |
| §22 — host omission detected at import, not prevented at launch | v4 | §0; v5 | §2 |
| §22 — receipts trusted for drift/accident detection only | v4 | §0; v5 | §3 |

## Open questions

| Open question | Origin | Reason it is unresolved |
|---|---|---|
| JSON Schema shape for `hma.acceptance-contract/v1` and `hma.plan/v1` | Draft author | Schema publication deferred |
| Derivation of live root, clone baseline, ledger namespace in hook | v4 §4 | Controller/adapter detail |
| `review_nonce` entropy source and rotation | v6 §1 | Implementation detail |
| Plan ceiling to verify-loop envelope mapping | v3 §3.2, v4 §3 | Adapter detail |
| `valid_until` chain computation and tie resolution | v7 §1 | Implementation detail |
| Deterministic checkpoint-trigger classifier | v3 §5, outcome consolidation §3 | Implementation detail |
| D5/D6 Obsidian adapter contracts | v4 §0, §4, §5 | Out of architecture scope |

