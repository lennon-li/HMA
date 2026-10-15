# Gate A — baseline, gap matrix, and proposed contract changes

**Date:** 2026-10-15
**Plan:** `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md` (the "plan")
**Reference audit:** `docs/audit-2026-10-14.md` (the "audit")
**Author:** Ming (Claude Code, Opus 5.5), lead implementer
**Status:** DRAFT for independent Gate A critique by Liz. Not reviewed. No code changed.

This packet is the plan's Gate A deliverable (a): current behavior, actual gaps,
proposed state/contract changes, compatibility impact, skill mapping, and test
plan. It is not an approval. It does not amend `docs/architecture.md`.

## 1. Baseline (observed 2026-10-15)

| Item | Value |
|---|---|
| Branch / HEAD | `main` / `118ea3a`, equal to `origin/main` after fetch |
| Uncommitted tracked edits | `TODO.md`, `cmd/hma/host.go`, `cmd/hma/host_test.go`, `cmd/hma/transition.go`, `docs/task14-host-orchestration.md`, `internal/host/host.go` (135+/12-) |
| Untracked | `cmd/hma/route_approval_binding_test.go`, `docs/audit-2026-10-14.md`, `scripts/dogfood/{host-dogfood.sh,hostinput/main.go}`, `.serena/` |
| Toolchain | `go1.27.1 linux/amd64` at `~/.local/go/bin` (audit recorded Go 1.24.0) |
| `go vet ./...` | PASS |
| `go test -count=1 ./...` | PASS, 616 tests, 12 packages |
| `gofmt -l .` | clean |
| `bash scripts/dogfood/host-dogfood.sh` | PASS (17 checks) |

The uncommitted edits are the route-approval binding subset recorded in `TODO.md`
(host decision/preflight/dispatch/implementation gate bound to the latest approved
`ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION` digest) and the synthetic dogfood
harness. The audit reviewed this tree. The plan requires preserving valid existing
work; Gate B will commit it together with its fixes, after review.

## 2. Audit findings — re-verified against source

Status words: `confirmed` = re-read in source this session; `inherited` = taken
from the audit, not re-checked.

| ID | Status | Evidence (this session) | Gate |
|---|---|---|---|
| H1 | confirmed | `cmd/hma/ci.go` uses `records[len-1].Stage` as `CurrentStage`, not `engine.ProjectStageState` | B |
| H2 | confirmed | `internal/pilot/run.go` requires `expected.ProducedHeadDigest == before.Head` and `DiffDigest == before.Diff` unconditionally; `model/validate.go` forbids those fields before binding stages, so early-stage pilot capture always fails | B |
| H3 | confirmed | `internal/host/store.go` `Append` writes the numbered record, then `head`; no pending intent, no lock. Core `internal/store/jsonl.go` already has `.head.pending` + `.lock` recovery; host store does not | B |
| H4 | confirmed | `runHostDecision/Preflight/Dispatch/Review` never load core outcome state; `approvedCoreRouteDigest` iterates only `stage_transition` records | B |
| M1 | confirmed | `cmd/hma/ci.go` has no `repostate.StoreOutsideRepository` call | B |
| M2 | confirmed | `cmd/hma/resolve.go` decodes once without the trailing `io.EOF` check used in `host.go`, `eval.go`, `main.go`, `transition.go` | B |
| M3 | confirmed | `engine/route_coherence.go` selects the rule by caller ID; only checks `risk >= MinimumRisk` | B |
| M4 | inherited | structural; exploitability unproven | B (test first) |
| M5, M6 | inherited | dogfood harness hardening | B |
| L1 | inherited | route attestation unit coverage | B |
| L2 | inherited | §21 release checks absent | D/E boundary, document |
| L3 | confirmed | `.github/workflows/ci.yml` has no `dogfood` step | B (after M5/M6) |

Open product-contract items from the audit and `TODO.md` (not regressions):
route identity/capability/permission persistence in core approvals; dependent-unit
invalidation scope; full §12.4 direct-cost exception; end-to-end host flow; Phase B
`VERIFIED_SUCCESS` closure and §21 release gate.

## 3. Plan audit (Ming, not independent)

Observed: the plan is internally consistent with `docs/architecture.md` §4
(human authority) and §12.4 (host layer). It keeps HMA core non-dispatching.

Findings on the plan itself:

1. **P-1 (scope):** Gates D/E introduce new normative contracts (outcome contract,
   lease, checkpoint policy). `STOP_RULES` and the plan both require human approval
   for product-contract changes. Coding for D/E cannot start on this packet alone.
2. **P-2 (route):** "Sol 6.1 High" resolves to Liz's Codex default
   `gpt-6.1-sol` (`~/.codex-liz/config.toml`), effort `high`. It is not in
   `skills/model-inventory.md`. Inferred, not runtime-verified until `hi` preflight.
3. **P-3 (duplication risk):** §1 lists acceptance tests and stage gates inside the
   outcome contract. Core already owns criteria (`criterion` records, §9) and stages
   (§5). Proposal: the outcome contract references the core criteria digest; it does
   not restate criteria.
4. **P-4 (tooling):** `hi` and Jev are not on `PATH` on asgard
   (`command -v hi jev` empty). `agent-route` exists. Per `AGENTS.md`, Jev
   unavailability is recorded; decisions fall back to orchestrator/human.
5. **P-5 (measurement):** cost/token budgets need a meter. The verify-loop
   controller enforces only zero-cost or declared per-attempt costs. The lease must
   use the same rule: unmetered paid routes fail closed.
6. **P-6 (Phase B):** minimum proof 5 needs `VERIFIED_SUCCESS` closure, which
   `CURRENT_STATE` defers to Phase B with the §21 release gate. Gate F needs either
   Phase B in scope or an explicit statement that closure ends at
   human-approved PARTIAL/ABORTED plus a "eligible for closure" status.

## 4. Proposed state and contract changes

Principle: one HMA state model. New items are host-layer records in the existing
host chain (`internal/host`), bound by digest to core records. Core gains no
dispatch, no lease authority, and no automatic transition.

### 4.1 Outcome contract (host record `outcome_contract`, schema `hma.host.outcome-contract/v1`)

Fields: `run_id`; `outcome_id`; intent/scope text digest; deliverables;
`criteria_digest` (digest over the core `criterion` records approved at
`ACCEPTANCE_CRITERIA`); risk; baseline revision; permitted routes and permission
envelope; budgets (attempts, wall time, tokens, cost model as in verify-loop);
escalation conditions; `contract_digest`. A later contract supersedes by predecessor
digest; any change to criteria digest, permissions, or risk is a checkpoint trigger
(§4.3) and requires a new human approval.

### 4.2 Execution lease (host record `lease`, schema `hma.host.lease/v1`)

Bound to `run_id`, `unit_id`, `route_approval_digest`, `contract_digest`,
`issued_at`, `expires_at`, `max_attempts`, budget ceilings, `forbidden_actions`.
Host `decision`, `preflight`, and `dispatch` require one active lease. Active means:
not expired, not exhausted (attempts counted from host dispatch records), contract
and route digests current, and the core run not terminal (fixes H4 class). A lease
never records a core transition and cannot be widened; a wider lease is a new
human-approved lease. Expired, exhausted, contradictory, or indeterminate leases
fail closed with a named reason and next action `ESCALATE_TO_HUMAN`.

### 4.3 Checkpoint policy (`hma.host.checkpoint-policy/v1`)

Deterministic function: (changed paths, path classes, risk, contract delta,
interface/schema delta, unit completion) → `{per_change_review: always,
stage_audit: required|not_required, reasons[]}`. Per-change independent review stays
mandatory (§12.4). Kernel triggers are compiled in. A repository policy file may add
triggers only; a policy that removes a kernel trigger is rejected. Stage audit
records bind an immutable audit-packet digest; a stale packet (head/diff mismatch)
or same-route reviewer is refused.

### 4.4 Completion metrics (read-only `hma host scorecard`)

Computed from host + core chains: criteria total; criteria with implementer/test
evidence; criteria independently verified; open closure-blocking findings;
reopened findings; attempts; elapsed time; declared cost. Primary completion rate
uses independently verified criteria only.

## 5. Skill mapping (no duplicate state)

| Skill | Actual contract (read this session) | HMA mapping |
|---|---|---|
| verify-loop (`AgentSystem/scripts/verify-loop`, `skills/verify-loop.md`) | Own ledger under git common dir; envelope pins scope, checks, attempts, deadline, route, cost; result `bounded-checks-only`; refuses dirty start | Host imports a completed ledger result as one core `evidence` record (argv, exit, output digest, ledger run ID + digest). It is implementer evidence, never criterion PASS by itself. HMA does not mirror ledger state. Lease budgets ≥ envelope budgets are refused (envelope must fit inside the lease). |
| independent-audit (`skills/independent-audit/SKILL.md`) | Fresh session; read-only; `prepare_packet.py` freezes target and hashes; `validate_report.py` checks report shape; withholds implementer narrative | Audit packet digest + report digest become a host `review` record (existing kind, extended with packet digest). Each report finding becomes a core `finding` record linked by ID. Rework references finding IDs; closure needs each blocking finding resolved by fresh evidence at a newer head. |

Caveat: the verify-loop refuses a dirty start; the current HMA tree is dirty. Gate B
must commit the preserved work before the verify-loop can run on it.

## 6. Compatibility

Existing core records and host records remain valid. New host kinds are additive;
older host chains without a lease remain readable but cannot dispatch after the
lease rule ships (fail closed). This is a behavior change for existing host users and
needs explicit human approval (Decision D1).

## 7. Test plan (summary)

- Gate B: one failing regression test per finding before each fix (H1–H4, M1–M6, L1);
  crash-injection between record and head for host store; concurrent writers;
  `-race`; dogfood harness byte-fingerprint refusals; CI dogfood step.
- Gate C: fixtures — audit finding → linked remediation → re-verification; stale
  packet; same-route reviewer; verify-loop evidence never yields criterion PASS.
- Gate D: lease expired/exhausted/widened/terminal-run; contract change requires new
  approval; host cannot write a core transition.
- Gate E: policy weakening rejected; trigger table tests; missing final audit blocks
  closure; self-review refused.
- Gate F: HMA-on-HMA dogfood with real binaries and the negative scenarios listed in
  plan §5, traces kept under `docs/handoffs/evidence/`.

## 8. Decisions required from the human

- **D1:** Approve §4 as the direction for an architecture amendment (new host-layer
  outcome contract, lease, checkpoint policy) and the fail-closed lease rule in §6.
- **D2:** Approve dispatch of Liz (`liz` / `gpt-6.1-sol` / high, read-only) for each
  gate audit.
- **D3:** Scope of Phase B (`VERIFIED_SUCCESS` closure, §21 release gate) for Gate F;
  see P-6.
- **D4:** Commit authority: commit on `main` after each gate passes Liz's audit, or
  hold all commits for explicit approval.

## 9. Next action (§18)

- Recommended action: dispatch Liz for an independent Gate A critique of this packet
  against `docs/architecture.md` and the source tree at `118ea3a` plus the listed
  uncommitted files.
- Owner: Ming. Route: `liz` / OpenAI `gpt-6.1-sol` / high / Codex CLI via
  `agent-dispatch --review`. Permission: read-only repo, write one report file under
  the run directory; no git writes, no nested agents.
- Approval required: D2.
- Stop if Liz reports a blocker against `docs/architecture.md` §4 or §12.4.
