# Gate A independent audit — 2026-10-15

**Gate verdict: FAIL.** Machine-readable verdict: `fix-required`.

The existing Go tests, vet and 17-check synthetic dogfood run pass. Eight independent safety assertions fail. The packet preserves a non-dispatching core in principle. It does not yet define enough authority, artifact, skill-integration and acceptance binding to authorize Gate B against the complete plan.

**Before Gate B:** revise the Gate A design for A1–A8; add N1/N2 to the defect work; resolve the exact human architecture/scope/commit decisions needed by that work. H1–H4 and the other existing code defects are work for Gate B. They do not have to be fixed before the gate whose purpose is to fix them.

Risk is high because the scope includes approval, routing and architecture. No run-specific active-waiver packet or approval challenge was supplied. No waiver or stage authorization is assumed. No paid model or external-effect operation ran. The final target check confirms unchanged HEAD, status, tracked diff and all 102 file hashes (`target-verification.json`).

## Scope, isolation and frozen target

**Isolation: isolated context; procedural read-only repository access.** I did not author the packet. I received no implementer chat or defense. I derived requirements before reading the baseline and reference audit as audited claims. This is not a claim of an OS-enforced read-only mount or perfect blinding to comments in source.

- Dispatch: `hma-gate-a-audit-20261015`. Assigned reviewer: Liz / `gpt-6.1-sol` / high / review mode. Runtime model identity was supplied by the dispatch; no separate model endpoint was queried.
- Objective: `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md`, especially §§0–6. Norms: `AGENTS.md`, `docs/architecture.md`, Task 13, Task 14 and `TODO.md`.
- Branch: `main`. HEAD: `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81`.
- Starting `git diff | sha256sum`: `ce2c0caf6c85ea3539542511709e1e3e7e96dc7fa299316a53c7ad6b1466b66c`.
- Baseline packet SHA-256: `a90913fea56132d2f412cbef318079fd4c0369d749e4324b94129fed20333125`.
- Full file-hash manifest SHA-256: `3e1c0dfc56e135fabb9e0923b49fdf617d401e7410675b402305a12500e13cd4`. This adds untracked bytes omitted by `git diff`.
- Start status is reproduced below. Raw status, tracked diff and all 102 inventoried file bytes are retained in `start-status.txt`, `start.diff`, `start-manifest.json` and `snapshot/`. `.serena/` was ignored.
- Blind Phase 1 findings froze at `2026-10-15T13:36:13.913606+00:00` in `PHASE1.json`, SHA-256 `f2988561c8f8e9881c6cb40007b58fd994a3dfdb7ba817a22d4f2cd2bf1858a9`. No controlled unblinding followed.

```text
 M TODO.md
 M cmd/hma/host.go
 M cmd/hma/host_test.go
 M cmd/hma/transition.go
 M docs/task14-host-orchestration.md
 M internal/host/host.go
?? .serena/
?? cmd/hma/route_approval_binding_test.go
?? docs/audit-2026-10-14.md
?? docs/handoffs/2026-10-15-gate-a-baseline.md
?? scripts/
```

External skill snapshot: `external-snapshot/manifest.json`. Verify-loop script SHA-256 `80dbbab3d0e49f41d3573d45dac787e20d1d4ddb8414ca8c164243accb27d821`. Verify-loop skill SHA-256 `be76149b99c9f2b74a9d44ba0fc1f8fcff81b4290f1aa15aa90e9754ef3e810d`. These external files changed during inspection. This report uses the final captured version, which runs workers and checks in a private clone. The HMA target did not change in the checks performed.

## Evidence and reproduction

| Check | Result | Raw evidence |
|---|---|---|
| Go toolchain | `go1.27.1 linux/amd64` | captured command output |
| `go test -count=1 ./...` | PASS; 11 tested packages, one package with no tests | `go-test.log`, `go-test.result.json` |
| `go vet ./...` | PASS | `go-vet.log`, `go-vet.result.json` |
| `bash scripts/dogfood/host-dogfood.sh` | PASS; 17 synthetic checks | `dogfood.log`, `dogfood.result.json` |
| Independent probes | FAIL as expected: 8 tests expose safety/contract defects | `probes.log`, `*_audit_probe_test.go`, `overlay.json` |

Reproduce the probes from the frozen target with:

```sh
GOCACHE=/tmp/hma-gate-a-20261015/go-cache PATH="$HOME/.local/go/bin:$PATH" \
  go test -overlay /tmp/hma-gate-a-20261015/overlay.json \
  -run '^TestAudit' -count=1 -v \
  ./cmd/hma ./internal/engine ./internal/pilot ./internal/host
```

The overlay adds test source from `/tmp`. It does not edit HMA files. Fixture stores and repositories are scratch data. `probes-initial.log` retains an initial unused-import build failure. `probes-cache-refusal.log` retains a default-cache filesystem refusal. The final probe run fixes the import and uses `/tmp` for its Go cache. No approval escalation was requested.

## Direct answer to audit question 1

| ID | Independent status | Correction or qualification |
|---|---|---|
| H1 | Confirmed by source | Projected stage and matching approval selection both need attention. |
| H2 | Confirmed by source and probe | Valid early-stage capture fails. |
| H3 | Confirmed crash recovery gap; probe reconstructs crash bytes | O_EXCL guards same-sequence overwrite. Do not claim a reproduced concurrency data-loss exploit. |
| H4 | Confirmed by source and terminal decision probe | No terminal guard on host execution entrypoints. |
| M1 | Confirmed by source and probe | Add canonical outside-repo guard. N2 is a separate evidence-identity defect. |
| M2 | Confirmed by source and probe | A trailing document is accepted. |
| M3 | Confirmed by source and negative-control probe | Raise severity from medium to high for the independence bypass. |
| M4 | Confirmed structural omission; exploitation unknown | The baseline is right to say unproven. Fresh core-head checks block simple stale reuse. |
| M5 | Confirmed by source | Synthetic fixture guard accepts spoofed marker forms and arbitrary nonempty runs. No hostile same-UID security claim. |
| M6 | Confirmed by source | Equal counts do not establish unchanged bytes. |
| L1 | Confirmed coverage gap | No direct model attestation cases found; no runtime defect inferred from coverage alone. |
| L2 | Confirmed deferred capability | High impact for this outcome, not a new regression. Must resolve D3. |
| L3 | Confirmed workflow omission | Add only after harness hardening/review. |
| N1 | Missing from baseline | Host review is not bound to the transition artifact. |
| N2 | Missing from baseline | CI can label checks from a non-Git directory as the approved revision. |

The baseline honestly labels inherited items. Its inherited status is not itself a false claim of direct verification. Its strongest overstatement would be treating H3 concurrency harm or M4 route exploitation as demonstrated; this report does neither.

## Architecture and host authority — question 2

Observed: §4.0 keeps dispatch in the host and says that a lease cannot write a core transition. These statements agree with architecture §§1,4,12.4. The existing transition path still checks legal stage, expected/presented approval, nonce, sequence and live revision where bound.

Inference: the proposal cannot yet establish that the host is unable to forge or bypass a stage decision. A host that controls both expected and presented approval JSON is inside the existing trusted-host boundary. A worker must not receive that authority or store access. The lease needs a trusted issuance path and exact approved bindings. A human-approved lease cannot substitute for a new challenge-bound approval on an actual stage edge. Repair after independent validation may require a human-approved rewind before coding resumes.

A1 defines the authority gap. N1 defines a reproduced independent-review gate gap. A3, A5 and A6 define contract, checkpoint and accounting gaps. No finding asks for cryptographic protection against a fully hostile same-user host, which the current threat model excludes.

## Skills and one lifecycle — question 3

Observed: the actual verify-loop owns a durable execution journal, launch, recovery, red-check validation and controller-run checks. It does not provide independent semantic acceptance. Its green result is `bounded-checks-only`. The final captured version uses a private clone and copies only checked in-scope changes back. Its dispatch function calls `agent-dispatch` directly. Its no-progress handler can choose the envelope escalation route. A completed-result import cannot enforce HMA authority before those operations.

Recommendation: keep the loop journal as subordinate raw execution evidence. HMA must own approved contract, stage, finding, decision and acceptance state. Define one per-attempt adapter that checks authority and returns receipts. Do not create another task or gate machine. Import controller evidence with exact command/output/definition/clone-result bindings and independent reproduction or an approved trusted capture boundary. Agent-submitted ledger JSON remains a lead under architecture §14.

The audit skill report cannot be mapped to HMA by severity alone. Its validator checks form only. The report has no native HMA finding disposition or per-criterion acceptance receipt. HMA must validate provenance, target and independence. It must preserve BLOCK/WAIVABLE/ADVISORY semantics and all prior findings. See A2, A4 and A6.

## P-1–P-6 and D1–D4 — question 4

| Item | Assessment |
|---|---|
| P-1 | Correct: authority/contract amendments need explicit human approval. Direction approval must be followed by a concrete versioned amendment. |
| P-2 | Treat the configured model route as a claim until exact-route preflight/telemetry. A model-inventory omission is not proof of unavailability or ineligibility. This dispatch names the route; the auditor did not invoke a route or inspect credentials. |
| P-3 | Correct goal, incomplete mechanism. Current core criterion records do not contain the semantic acceptance definition. See A3. |
| P-4 | PATH absence does not establish absence of an MCP/SDK/HTTP tool. No Jev capability was exposed in this auditor host inventory. Record unavailable here and use the auditor fallback. Exact-route hi belongs to the dispatching host. |
| P-5 | Correct cost concern. Declared costs need an enforceable maximum. Token metering and cumulative attempt semantics remain undefined. See A6. |
| P-6 | Correctly identifies the Phase B dependency. PARTIAL/ABORTED is truthful stopping but cannot satisfy the original verified-closure outcome without an explicit human scope reduction. See A7/L2. |
| D1 | Needed for exact normative changes and compatibility behavior. The proposed direction alone leaves safety bindings undecided. |
| D2 | Current explicit dispatch authorizes this audit. Do not reinterpret it as blanket approval of future routes, artifacts, gates or commits. |
| D3 | Required scope decision before claiming a complete executable plan. State the affected minimum proofs. |
| D4 | Commit authority must be explicit. Resolve the dirty-start/verify-loop sequence before Gate B execution. A passing audit does not itself authorize a commit. |

Missing Gate A blockers: trusted lease issuance and stage/rework legality; semantic criteria authority; exact artifact/review binding; per-attempt loop integration and evidence provenance; finding import/disposition/reopen semantics; complete checkpoint inputs and reviewer separation; cumulative budgets and retry classification; and the proof matrix. Existing route facts, dependent-unit invalidation, direct-cost conditions and adapter lifecycle must be assigned to explicit units with exit criteria. Listing them as open product items does not establish when the approved outcome will satisfy them.

## Minimum proof coverage — question 5

| Plan §6 proof | What §7 supplies | Missing acceptance checks |
|---|---|---|
| 1: objective criteria and budgets | Lease negatives; contract-change approval | Round-trip semantic criteria/tests/thresholds; canonical digest and migration; meter support; budget equality and overflow; immutable policy/spec bindings. |
| 2: permitted iterations; authentic stage approval | Generic host-cannot-write-transition assertion | At least two real permitted repair/test iterations; lease issuance; actual hi/dispatch receipts; no host self-approval; nonce replay and wrong unit/plan/repo; required rewind refusal; crash/resume accounting; direct-cost path. |
| 3: every coding change; meaningful/final audit | Trigger table, self-review, missing final audit | Coding artifact coverage across multiple edits; exact-model and family separation; disputed/final cases; missing/unavailable reviewer; metadata-only/irreversible/final triggers; immutable full packet; wrong/stale policy and evidence at unchanged head. |
| 4: rejection links repairs; fresh verification | Finding-repair fixture; stale packet | Multiple finding snapshot preservation; disposition mapping; BLOCK cannot disappear/waive; reopened finding; report shape versus truth; repair rejection and independent retest; new bytes with unchanged core head; same-head evidence-only repair. |
| 5: true/partial/failure; no false success | Terminal lease case; unspecified Gate F negatives | Real VERIFIED_SUCCESS and VERIFIED_WITH_WAIVERS through §21; all required CI/final audit; unresolved and stale criteria; waiver accounting; live failure precedence; scorecard denominators and stale/duplicate/reopened evidence; explicit PARTIAL/FAILED/BLOCKED/UNKNOWN/ABORTED. |
| 6: real reproducible dogfood; exact final audit | Reference to plan Gate F scenarios | Actual host adapter and real worker returns; approved real stage decisions; induced finding/repair/re-audit; interruption, quota, stale approval, changed criteria, expired lease and honest stopping; raw traces; exact final commit plus packet/evidence hashes. Synthetic fixtures remain separate. |
| 7: one state; no downgrade/external effect | No explicit ownership/provenance test | One authoritative reducer/owner; subordinate loop journal; trusted approval-store isolation; no actual external-effect launch; migration fail-closed behavior; portable/host boundary; route facts/dependent invalidation/direct-cost enforcement. |

All seven are future proof obligations. None is established in full by this Gate A packet. This does not require all later tests to exist before coding. It requires a complete, consistent plan for producing them.

## Findings and closure criteria

Severity is impact on this approved outcome. `observed` includes directly inspected plan gaps; it does not mean an unimplemented proposal was executed. All remedies are proposed. No fixed revision exists. Each finding retains expected versus actual behavior, source, remedy, verification and trade-offs in `AUDIT.json`.

### A1 — Lease and contract approval authority is underspecified

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:87-110,141-144; docs/architecture.md §§4-6,10,12.4,15`.
**Actual:** The proposed initial lease contains no human decision reference or approval binding. Active checks cover expiry, counters, current route/contract and nonterminal status, but not an approved unit/plan, legal execution phase, exact permission facts or dependent-unit invalidation. Only widening is explicitly called human-approved. The packet acknowledges that existing route approval stores no approved route facts, yet gives that digest authority over the lease.
**Expected:** A host execution lease must derive authority from an authentic trusted human decision for exact approved scope. It cannot authorize a new HMA stage or bypass the human gate on rework.
**Proposed fix:** Specify the trusted issuance/revocation path, exact human-bound lease/contract/policy digests, approved plan/unit/route roles and permission facts, legal phase conditions, dependency invalidation and replay rules. Keep worker access outside the approval/store boundary. Make all host execution entrypoints consume this checked authority; retain the existing transition approval path.
**Closure criterion:** Reject an initial self-issued lease, copied approval, another unit/plan/repository, stale policy/contract, invalidated dependent unit, unauthorized route/permission and rework requiring an unapproved rewind. A valid in-stage bounded lease may run permitted work but cannot advance a stage.
**Trade-off:** This requires a concrete architecture amendment and persisted approval facts. Merely accepting D1 direction does not define the binding. No new cryptographic identity system is required within the existing trusted-host threat model.
**Alternative:** Defer lease dispatch and retain manual trusted-host execution until the authority schema is approved.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A2 — Completed verify-loop import does not integrate its execution authority

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:129-137; external-snapshot/scripts/verify-loop:819-850,1255-1273,1437-1448; external-snapshot/agent-memory/skills/verify-loop.md:3-13,44-74`.
**Actual:** The actual controller owns attempts, deadlines, recovery and worker launch through agent-dispatch. It can select an escalation route after failed interventions. The proposed mapping only imports a completed result. It defines no before-each-attempt HMA eligibility, exact-route preflight, dispatch receipt, authority/stop check, or approved escalation bridge. Host counting of dispatch records will miss these inner attempts unless that bridge exists. The current skill also runs workers/checks in a private clone and copies changed paths back; the packet does not bind clone evidence to the returned HMA artifact.
**Expected:** Use one authoritative HMA run and approval model. Retain the controller execution journal as subordinate evidence. Every actual dispatch must obey HMA policy, including resume and escalation.
**Proposed fix:** Choose and document one integration seam before Gate C. Prefer a controller callback/adapter that reserves each attempt under the HMA lease, validates the exact approved route, performs real preflight, records dispatch before launch, and checks stop/rewind conditions. Bind controller run/envelope/check-definition hashes and clone/result artifact digests on import. HMA owns findings/acceptance; the loop owns only its execution journal.
**Closure criterion:** Trace two inner repair attempts, interruption/resume and an escalation. Require one HMA-bound reservation/preflight/dispatch per actual attempt. Revocation, missing preflight and required human rewind must prevent the next launch. Reproduce imported checks under HMA or use an explicitly trusted capture adapter; never count worker prose as proof.
**Trade-off:** A final JSON import can be useful evidence but cannot retroactively govern dispatch. Changing the external controller needs its own authorized scope and review; do not silently edit AgentSystem.
**Alternative:** Run the loop as a separately approved bounded controller job and explicitly prove each inner dispatch conforms; a narrow evidence-only pilot must disclose that it does not prove full execution integration.
**Effort:** large. **Fix status:** proposed. **Fixed revision:** none.

### A3 — The criteria digest does not bind objective criterion definitions

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:70-73,91-99,122-127; internal/model/types.go:142-146; internal/engine/stage_state.go:47-49,78-80`.
**Actual:** Criterion records contain ID and disposition only. A digest over those records does not bind acceptance test definitions, thresholds, non-goals or evidence requirements. It also conflates mutable result disposition with the approved definition. The proposal does not specify which approved snapshot is canonical or how a criterion maps to independent acceptance. Risk, routes, permissions and budgets appear in multiple places without a precedence rule.
**Expected:** One immutable approved semantic acceptance contract must define objective success. Result projections must not redefine it or double-count duplicate evidence.
**Proposed fix:** Define one canonical approved specification containing criteria definitions, tests/thresholds, non-goals, dependencies, required evidence and policy version. Persist/reference its digest from core approval and the outcome record. Keep criterion dispositions and independent evidence links as separate projections. Specify precedence over lease/envelope copies and exact invalidation rules for every contract field.
**Closure criterion:** Change a threshold while preserving criterion IDs/dispositions and require invalidation plus human reauthorization. Record PASS without independent evidence and require zero independently verified credit. Test stale/duplicate/reopened evidence, waivers and zero criteria.
**Trade-off:** A specification may live as a content-addressed document; extending every portable record is unnecessary. Existing records need explicit migration semantics.
**Alternative:** Publish the complete criterion specification as a single host contract, with core approvals binding its digest; do not duplicate definitions in the host and core.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A4 — Audit import can lose findings and lacks acceptance/disposition rules

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:134; internal/model/types.go:148-155; internal/engine/resolution.go:42-49; external-snapshot/agent-memory/skills/independent-audit/references/report-contract.md`.
**Actual:** The audit report uses severity and verdict, while HMA findings use BLOCK/WAIVABLE/ADVISORY. The mapping supplies no deterministic disposition/reference policy or verified per-criterion acceptance. It says each finding becomes a core finding record; current projection replaces the whole nonempty Findings snapshot, so naive one-finding-per-record imports hide earlier findings. Closure at a newer head is an extra condition even when fresh evidence at the same head resolves a non-code gap. A report digest and phase1_frozen field do not establish reviewer independence.
**Expected:** Preserve all linked findings and their approved invariant/criterion references. Severity alone cannot invent a blocker. Only a properly bound independent review can support verified acceptance. Freshness must follow actually bound inputs.
**Proposed fix:** Specify report verdict/disposition mapping, allowed invariant/criterion references, complete snapshot or explicit event-reducer semantics, stable IDs and reopen/supersession links. Define authenticated host provenance for reviewer/model/provider/context separation and the frozen target. Require independent confirmation of repairs at the exact current artifact; require a new head only when the repair changes committed content.
**Closure criterion:** Import BLOCK F1 followed by ADVISORY F2 and prove F1 remains open. Reject self-review, partial/compromised isolation for final independent acceptance, stale target, fabricated phase1 claim and out-of-criteria blockers. Retest repairs and show same-head fresh evidence can resolve an evidence-only gap without manufactured commits.
**Trade-off:** Preserve the skill method and raw report. Structural report validation checks shape, not truth. BLOCK findings must not be silently waived or overwritten.
**Alternative:** Store the full immutable report and a validated cumulative HMA finding snapshot with explicit source links, rather than importing isolated rows.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A5 — Checkpoint policy omits final/irreversible triggers and packet bindings

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:112-120; approved handoff §§3,5; docs/architecture.md §§12.4,13,16,21`.
**Actual:** The proposed detector input has paths, risk, contract/interface delta and unit completion, but no explicit final-closure or irreversible/external-effect event. Packet binding is described only as packet digest plus head/diff. It omits the required scope/criteria/evidence/policy and reviewer-context bindings. Same-route refusal alone does not enforce different exact model at normal risk or provider/model-family separation for high, disputed and final validation.
**Expected:** Every mandatory meaningful-change trigger and final audit must be deterministic and unavoidable. Every coding change still needs independent review.
**Proposed fix:** List the full trigger table from plan §3, with explicit final-closure, external/irreversible effect and approved milestone events. Define a versioned immutable packet manifest containing exact revision/diff/dirty bytes, scope, criteria, required evidence digests, reviewer identity/model/provider/context and policy. Enforce §13 independently of route equality.
**Closure criterion:** Use a same-path metadata-only criteria/permission change, an external-effect event without code diff, final closure with no new diff, and every mandatory trigger. Reject changed policy/evidence/criteria at unchanged head and same-model/different-worker or same-family final reviewers.
**Trade-off:** Policy may add triggers but cannot weaken the compiled floor. Conservative unknown classification must stop or request inspection, not silently skip the audit.
**Alternative:** Initially require an independent stage audit after every approved batch, while retaining per-change review and final-audit bindings.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A6 — Budget comparison is reversed and attempt authority is incomplete

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:77-79,103-110,133; docs/architecture.md §§10,17; external-snapshot/agent-memory/skills/verify-loop.md:25-33`.
**Actual:** The packet says lease budgets >= envelope budgets are refused, then says the envelope must fit inside the lease. The inequality is reversed. Counting host dispatch rows does not define direct-cost work, inner loop attempts, review dispatches, retries/reroutes, crash reservations or repeated-failure stopping. Token ceilings have no enforcement mechanism. The proposal omits the §17 classification and same-material-failure-twice stop rule.
**Expected:** Reject an envelope that exceeds any remaining lease ceiling. Count all implementation attempts across corrections/reroutes. Do not count a reviewer launch as a new implementation attempt. Fail closed when a claimed hard ceiling is unmeasurable.
**Proposed fix:** Write componentwise envelope <= remaining lease constraints, including equality. Define an append-only attempt reservation/result identity and monotonic counters across restart and lease supersession. Account for direct work and nested attempts. Integrate §17 classification/human confirmation and stop after two material failures on a route. Define reliable wall/token/cost enforcement, or refuse unsupported hard budgets.
**Closure criterion:** For lease ceiling 5 accept envelope ceilings 4 and 5, refuse 6. Exercise crash before/after launch, resume, direct work, nested attempts, reviews, reroute and lease renewal. Assert no reset of spent authority, no third blind same-failure attempt, and refusal of unmetered paid/token routes.
**Trade-off:** Declared spend is a reservation, not observed provider billing. The exact maximum must be independently enforceable. Separate the arithmetic typo from the larger accounting design gap.
**Alternative:** Use only explicitly zero incremental-cost routes and finite runtime/attempt budgets for the first approved pilot; disclose unsupported token metrics.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A7 — PARTIAL/ABORTED is not an equivalent completion of the approved outcome

**HIGH · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:80-83,166-169,26-30,136-137; approved handoff §§5-6`.
**Actual:** P-6 offers PARTIAL/ABORTED plus eligible-for-closure as an alternative for Gate F. The approved goal requires human-gated verified closure and minimum proofs 3-6. Honest partial stopping is supported, but it does not meet that original outcome. D3 is unresolved. Gate B also needs a concrete clean-start strategy: the packet promises to commit preserved work with fixes after review, but requires a clean tree before the verify-loop runs.
**Expected:** Any scope reduction must be an explicit human change to acceptance criteria. A gate verdict or generic direction approval does not grant commit, release or future dispatch authority.
**Proposed fix:** Rewrite D3 as an explicit scope decision with affected proof coverage. State either the authorized Phase B/release implementation or the approved incomplete deliverable. Resolve the Gate B review/commit/clean-loop order and D4 authority before execution. Preserve existing bytes and keep each commit within approved authority.
**Closure criterion:** Trace every minimum proof to a concrete phase and executable check. Refuse to label PARTIAL or eligible-for-closure as VERIFIED_SUCCESS. Demonstrate the chosen Gate B workflow can start the actual pinned verify-loop without unapproved commits or overwriting preexisting edits.
**Trade-off:** Scope resolution may change delivery time. It must not be hidden in a status label. Current audit dispatch satisfies current D2; it does not authorize all future work.
**Alternative:** Perform static/regression defect work first under explicit scope, independently review it, obtain authorized commit authority, then run a clean bounded loop and re-audit its exact result.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### H1 — CI reads the record stage instead of the projected stage

**HIGH · observed · code · Gate B.**

**Evidence:** `cmd/hma/ci.go:72-90,130; internal/engine/stage_state.go:62-87`.
**Actual:** CI uses the last record Stage. Approved transition records store the source stage. Resolution records can omit Stage. CI can reject a permitted current stage or evaluate a stage the run has left.
**Expected:** CI must evaluate the committed current stage and select a corresponding applicable approval.
**Proposed fix:** Use ProjectStageState for CI legality and evidence-record stage. Select the bound approval by the intended CI check, rather than the latest arbitrary Approval pointer.
**Closure criterion:** Run CI after IMPLEMENTATION_AUTHORIZATION -> IMPLEMENTATION_REVIEW and after an appended resolution. Also test VERIFICATION -> INDEPENDENT_VALIDATION. Assert projected-stage behavior and no fallback to an unrelated approval.
**Trade-off:** A projected stage alone does not supply a produced-head approval where the preceding source-stage approval never bound one.
**Alternative:** Require an explicit validated CI request bound to the projected stage. This changes more CLI surface.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### H2 — Pilot rejects valid early-stage approvals

**HIGH · observed · code · Gate B.**

**Evidence:** `internal/pilot/run.go:90-95; internal/model/validate.go:175-188; probes.log:TestAuditEarlyStagePilot`.
**Actual:** A Grounding approval without produced head/diff is rejected as expected approval binding is stale. Model validation forbids those fields at early stages. The independent probe reproduced the conflict.
**Expected:** Pilot must enforce head/diff only at stages whose approval binds them. Its live capture snapshot still binds the actual evidence.
**Proposed fix:** Reuse model.ApprovalBindsHeadDiff for the expected-approval comparison, as transition.Apply already does. Keep live repository and exact dirty-content checks.
**Closure criterion:** Pass valid Grounding and Planning captures without approval head/diff. Reject stale head/diff at Implementation Review and a changed approved dirty worktree.
**Trade-off:** Do not remove live snapshot checks when narrowing approval comparisons.
**Alternative:** Limit pilot to later stages and introduce a separate early-stage evidence path. This is a larger compatibility change.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### H3 — Host append has no interrupted-write recovery

**HIGH · observed · code · Gate B.**

**Evidence:** `internal/host/store.go:85-135,138-192; probes.log:TestAuditHostInterruptedHead`.
**Actual:** Append writes a numbered record before the head. The crash-window byte state (durable record, missing head) fails Load with HOST_HEAD_MISMATCH. The independent probe reconstructed that state; it did not kill a process. There is no pending transaction or per-run lock. O_EXCL does prevent two writers from overwriting the same numbered file; lost writes from concurrency were not demonstrated.
**Expected:** Interrupted appends must recover or roll back deterministically without losing a committed record. Concurrent writes must preserve one valid chain.
**Proposed fix:** Use a per-run transaction lock and durable pending commit protocol comparable to the core store. Recover before Load/Append and synchronize directory changes where required.
**Closure criterion:** Inject failure after creation, partial record write, fsync, pending intent and head replacement. Restart and assert one valid committed prefix. Exercise separate-process writers and verify every accepted append remains in the chain.
**Trade-off:** Recovery needs an explicit commit point. Blindly trusting the final numbered record could accept an incomplete write.
**Alternative:** Store host events through a shared proven transaction primitive while preserving separate host record semantics.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### H4 — Host operations accept a terminal run

**HIGH · observed · code · Gate B.**

**Evidence:** `cmd/hma/host.go:76-121,131-253; cmd/hma/transition.go:131-164; probes.log:TestAuditTerminalHostDecision`.
**Actual:** A core ABORTED outcome does not clear the route digest. A new host decision succeeds and appends against that terminal core head. The same outcome checks are absent from the other host mutation entrypoints.
**Expected:** A terminal run must refuse new execution-authorizing host operations. Reading historical telemetry can remain allowed.
**Proposed fix:** Project core stage/outcome before all host execution mutations. Reject terminal execution and bind the check and append to a consistent core snapshot. Define any allowed post-terminal reporting operation explicitly.
**Closure criterion:** After each supported terminal outcome, refuse decision, preflight and dispatch without byte changes. Test any permitted read/report path separately.
**Trade-off:** A future lease check does not repair the present defect until it is implemented on every path.
**Alternative:** Apply the terminal guard at a shared host boundary and keep historical reads available.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### L2 — Verified closure and the §21 release gate remain deferred

**HIGH · observed · code · Resolve scope before Gate B; implement at approved later gate.**

**Evidence:** `docs/task13-terminal-outcomes.md:41-46; internal/engine/ci_verification.go:24-53; internal/engine/stage_state.go:ReasonOutcomeNotImplemented; README.md deferred phases`.
**Actual:** The write path refuses VERIFIED_SUCCESS and VERIFIED_WITH_WAIVERS. The CI evaluator checks stage and identity only; it does not implement the §21 release contract. The baseline is correct that this is deferred work, not a new regression. It is high impact relative to the approved consolidation outcome.
**Expected:** The final requested verified closure must have independently verified criteria, final validation, required CI and authentic human closure authority.
**Proposed fix:** Resolve D3 explicitly. Include the minimal Phase B verified-closure and §21 work in the authorized plan, or obtain explicit human approval to reduce the original deliverable and label it incomplete.
**Closure criterion:** Exercise successful closure only from RELEASE_AND_CLOSURE with all required proof. Reject missing final audit, required CI, stale evidence, unresolved BLOCK/WAIVABLE and missing human approval. Test waivers yield VERIFIED_WITH_WAIVERS.
**Trade-off:** This is broader than defect-only Gate B. It must have an explicit phase and approval boundary.
**Alternative:** Deliver an explicitly human-approved partial scope. That alternative cannot be called completion of the original plan.
**Effort:** large. **Fix status:** proposed. **Fixed revision:** none.

### M3 — Caller can select weak independence for critical risk

**HIGH · observed · code · Gate B.**

**Evidence:** `internal/engine/route_coherence.go:263-273,308-325; probes.log:TestAuditCriticalWeakRule`.
**Actual:** A CRITICAL request with a same-provider/model-family validator is rejected with the high-risk rule, then accepted when only independence_rule_id changes to a low-risk rule. The policy contains both rules. The independent probe supplies this negative control. The reference audit labels this medium; high is appropriate because the §13 independence requirement is bypassed in the route evaluator.
**Expected:** Caller selection cannot weaken the mandatory high-risk, disputed or final independence floor.
**Proposed fix:** Derive the effective independence constraints from risk and all applicable policy rules. Enforce the kernel floor independently of the caller rule ID. Explicitly represent disputed/final conditions when this evaluator is used for those decisions.
**Closure criterion:** For CRITICAL and HIGH, select every lower-risk rule in turn and assert same-family rejection. Preserve valid separated routes and normal-work controls. Cover final and disputed validation.
**Trade-off:** Combining constraints requires a deterministic documented precedence or union rule.
**Alternative:** Reject every noncanonical rule selection for the risk, then enforce the kernel floor separately.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### N1 — Independent review is not bound to the transition artifact

**HIGH · observed · code · Gate B.**

**Evidence:** `cmd/hma/transition.go:54-74; internal/host/host.go:118-141,589-608,635-649; probes.log:TestAuditReviewGateUnreviewedArtifact`.
**Actual:** The same valid review lets requireHostTransitionGate accept two different ProducedHeadDigest values with an unrelated DiffDigest. The gate never resolves ArtifactDigest/ReviewPacketDigest to the transition repository, head, diff or exact dirty bytes. The probe demonstrates the host-gate defect, not a complete stage transition without human approval.
**Expected:** Only an independent review of the exact artifact being progressed may satisfy a coding gate (architecture §§6,12.4,14).
**Proposed fix:** Define an immutable artifact manifest with repository/base/head/diff and optional exact worktree digest, plus the core/host binding. Resolve and compare it at review import and the transition gate. Make any later artifact change require a fresh independent review.
**Closure criterion:** Review artifact A, change the artifact to B without changing the core chain, and present otherwise valid approval for B. Reject progression until a fresh review of B is recorded. Include dirty bytes and a valid A control.
**Trade-off:** An opaque digest proves equality only between fields that are actually compared. Do not replace the human approval or expand the same-UID threat model.
**Alternative:** Persist explicit artifact binding fields on review records and compare them directly; still retain immutable packet bytes.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### N2 — CI labels evidence with a revision it never checks locally

**HIGH · observed · code · Gate B.**

**Evidence:** `cmd/hma/ci.go:85-115; internal/evidence/capture.go:98-160; cmd/hma/main_test.go:256-287; probes.log:TestAuditCINonRepositoryEvidence`.
**Actual:** CI accepts matching environment strings and an approval, runs /bin/true in an empty non-Git directory, and writes evidence labelled approved-sha with the approval diff. The independent probe reproduced this. The existing positive CI test also uses a non-Git directory. No before/after Git snapshot or dirty-content check establishes what was tested.
**Expected:** Fresh required evidence must describe the exact tested repository content. Environment metadata alone is insufficient (architecture §§7.1,14,21).
**Proposed fix:** Validate the actual --repo-root repository, base, head, diff and approved dirty-content state before capture. Capture metadata from that snapshot. Verify unchanged content after the command before appending evidence.
**Closure criterion:** Reject an empty directory, another checkout, wrong head/diff, unapproved dirt and a command that mutates the checkout. Accept only a real matching clean/explicitly bound dirty fixture.
**Trade-off:** GitHub environment identity still has host-trust limits; this fix establishes actual local content, not actor authentication.
**Alternative:** Delegate capture to the established pilot snapshot boundary with CI-specific identity checks.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### A8 — Summary test plan does not establish all seven minimum proofs

**MEDIUM · observed · plan · Before Gate B plan acceptance.**

**Evidence:** `docs/handoffs/2026-10-15-gate-a-baseline.md:146-158; approved handoff §6 minimum proofs 1-7`.
**Actual:** The test summary names selected negative cases but supplies no acceptance-to-command/evidence matrix, real multi-iteration host test, approval forgery/replay isolation test, semantic criterion-drift test, deterministic finding/scorecard tests or complete release/terminal tests. It repeats the plan Gate F scenarios without an executable host adapter or explicit authority model for the real run.
**Expected:** Each approved minimum proof needs a positive case, material negative controls, exact revision/evidence and independent acceptance. Synthetic fixtures must remain labelled.
**Proposed fix:** Add the seven-row proof matrix in this report to the Gate A plan. Name test ownership, command, expected first refusal, frozen input/output digests and gate. Add N1/N2 and the full authority/retry/import/checkpoint negatives before declaring Gate A ready.
**Closure criterion:** Review the expanded matrix against proofs 1-7. Execute its bounded checks at their assigned gates and freeze raw results. Require final independent audit of the exact final revision; do not treat a synthetic 17-check run as proof of actual dispatch or human decisions.
**Trade-off:** Gate A can approve a sufficient executable plan before later implementation tests exist. It cannot claim those future proofs have passed.
**Alternative:** Narrow the deliverable only through an explicit human-approved acceptance revision, and mark every removed proof as out of the new scope.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### M1 — CI can write a store inside the claimed repository

**MEDIUM · observed · code · Gate B.**

**Evidence:** `cmd/hma/ci.go:40-58,104-134; probes.log:TestAuditCIInsideRepositoryStore`.
**Actual:** CI appends evidence to a store under --repo-root. It does not call StoreOutsideRepository. The probe reproduced this with a claimed root; actual Git identity is independently missing under N2.
**Expected:** The canonical store must remain outside the worker-writable project tree (architecture §15).
**Proposed fix:** Apply the canonical, symlink-aware outside-repository store guard before CI evidence execution or mutation.
**Closure criterion:** Reject a store inside an actual scratch Git repository and one that resolves inside through a symlink. Allow a separate scratch store.
**Trade-off:** Use canonical paths so a lexical prefix check cannot miss symlinks.
**Alternative:** Require the host to provide an already validated external store handle; this needs a wider API change.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### M2 — Resolve accepts trailing JSON documents

**MEDIUM · observed · code · Gate B.**

**Evidence:** `cmd/hma/resolve.go:52-57; probes.log:TestAuditResolveTrailingJSON`.
**Actual:** A valid waiver JSON followed by a second JSON object is accepted and appends the waiver. The independent probe reproduced this.
**Expected:** One CLI input must contain exactly one strictly decoded document.
**Proposed fix:** Use the existing strict decoder helper, including a trailing io.EOF check, before any store mutation.
**Closure criterion:** Reject two documents and trailing malformed tokens. Accept whitespace after one valid document. Assert unchanged chain/head bytes on refusal.
**Trade-off:** Clients relying on ignored trailing bytes will fail. That is the intended boundary.
**Alternative:** Add an equivalent local EOF check if sharing the helper would create package coupling.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### M4 — Review gate omits direct route revalidation; exploit remains unproven

**MEDIUM · inferred · code · Gate B.**

**Evidence:** `cmd/hma/transition.go:35-74; cmd/hma/host.go:229-248`.
**Actual:** The second coding gate checks an APPROVE and the review record core head, but does not call requireApprovedHostRoute. Core-head freshness rejects simple old-review reuse after a rewind. No legal end-to-end bypass due only to this omission was reproduced. The baseline accurately calls it structural and unproven.
**Expected:** Do not describe this as a demonstrated route bypass. Determine whether the second gate must enforce an active route under the approved contract.
**Proposed fix:** First add a route-supersession/rewind test through both legal coding edges, including attempted review re-attestation. Revalidate the current route and implementation at the shared gate if needed; retain freshness checks.
**Closure criterion:** Attempt stale, superseded, dependent-invalidated and re-attested review paths with authentic fixture transitions. Report the actual first refusal. Close as a documented non-defect if all legal paths enforce the invariant elsewhere.
**Trade-off:** A redundant check can improve defense but must not invent an approval requirement.
**Alternative:** Prove the invariant from the full transition path and retain the omission with that proof.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### M5 — Synthetic-store marker does not establish fixture ownership

**MEDIUM · observed · code · Gate B.**

**Evidence:** `scripts/dogfood/hostinput/main.go:112-116,144-170`.
**Actual:** requireSyntheticStore uses os.Stat only. A copied marker, symlink or directory with that name passes. Approval seeding accepts any nonempty run, without checking the seeded criterion identity, run contents or prior approvals. This is a fixture guard defect, not a same-UID tamper-resistance claim.
**Expected:** Synthetic approval-shaped records must be confined to an identified scratch fixture, not appended to an arbitrary marked store.
**Proposed fix:** Validate canonical scratch ownership, a regular non-symlink marker with expected identity, the exact seed run and permitted record sequence before seeding. Refuse existing approvals and foreign records.
**Closure criterion:** Reject marker directories/symlinks, copied markers on foreign stores, mismatched runs, foreign seed records and second approval seeding. Keep the labelled fresh-scratch positive fixture.
**Trade-off:** Filesystem marker checks cannot protect against a fully hostile same-user writer. Keep the trust claim bounded.
**Alternative:** Eliminate the standalone seeding command and create all fixture state inside a scratch-only test harness.
**Effort:** medium. **Fix status:** proposed. **Fixed revision:** none.

### M6 — Refusal harness compares record counts, not stored bytes

**MEDIUM · observed · code · Gate B.**

**Evidence:** `scripts/dogfood/host-dogfood.sh:69-80`.
**Actual:** expect_refusal checks host record count and core line count. An in-place change of a record or anchor can preserve both counts. The baseline correctly retains this finding.
**Expected:** A refusal that promises no mutation must preserve all relevant chain and head bytes.
**Proposed fix:** Hash a sorted path/content manifest of core stream, core head/pending state and host record/head files before and after refusal. Compare additions, deletions and content.
**Closure criterion:** Use a deliberately mutating refusal double that preserves counts but alters one stored byte. The harness must fail. Unchanged genuine refusals must pass.
**Trade-off:** Snapshot only the controlled store; stdout logs and unrelated scratch outputs should not affect the comparison.
**Alternative:** Compare complete copied store trees byte-for-byte in scratch instead of hashes.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### L1 — Route attestation lacks direct structural test coverage

**LOW · observed · code · Gate B.**

**Evidence:** `internal/model/route.go:51-103; internal/model/*_test.go (rg for RouteAttestation/validateRouteAttestation returned no matches)`.
**Actual:** The attestation validator exists, but the model tests do not directly exercise its structural field boundaries. The prior low-priority coverage observation is correct. Missing coverage alone is not a demonstrated production failure.
**Expected:** The portable attestation field contract needs positive and negative structural fixtures.
**Proposed fix:** Add a valid attestation and table cases for missing required fields, invalid execution context, empty class sets and empty members through ValidateRecord.
**Closure criterion:** Assert acceptance of the valid control and refusal of each malformed attestation. Test the intentional nil-attestation compatibility path.
**Trade-off:** Tests should encode the approved schema, not silently add policy semantics to structural validation.
**Alternative:** Cover these cases in portable schema conformance tests when those schemas ship.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

### L3 — CI does not execute the shell dogfood harness

**LOW · observed · code · Gate B.**

**Evidence:** `.github/workflows/ci.yml:verify steps; scripts/dogfood/host-dogfood.sh`.
**Actual:** The workflow runs formatting, vet, build, tests, race and determinism. It has no dogfood step. Go test does not execute the shell harness. The baseline status is correct.
**Expected:** The hardened executable harness must run in the conformance workflow before its result is used as release evidence.
**Proposed fix:** After M5/M6 review, add a CI step for the harness with declared Go/jq dependencies and retained failure evidence.
**Closure criterion:** Run the workflow command in a scratch store and prove an induced harness failure fails the job.
**Trade-off:** A green synthetic harness remains synthetic evidence; it does not prove real host dispatch.
**Alternative:** Move the same bounded CLI checks into Go integration tests that CI already executes.
**Effort:** small. **Fix status:** proposed. **Fixed revision:** none.

## Remediation sequence and limits

1. Revise the Gate A packet. Close A1–A8 at the design level. Resolve concrete D1/D3/D4 decisions before dependent work. Preserve the current tree and account for the two new code findings.
2. Under a valid Gate B coding authorization, write the targeted regressions and fix H1–H4, M1–M3, M5/M6 and N1/N2. Investigate M4 before prescribing a fix. Review the preserved route-binding subset. Harden and then integrate the harness. Keep the deferred route/closure obligations assigned to explicit later gates.
3. Freeze exact post-fix evidence and obtain independent review through the eligible route. Implement later gates only within their approved scope. Final outcome acceptance still needs authentic human authority.

No optional redesign is recommended. The smallest useful improvement is to reuse the core store transaction discipline and existing snapshot/approval helpers while keeping host telemetry separate.

Audit limitations:

- Fresh auditor context. No implementer conversation, prior model reasoning or defense was supplied. The approved plan, baseline packet and reference audit are inherently author-written objects and were read as claims, after independent requirements reconstruction.
- Isolation is isolated at the context level. Repository access was read-only by audit procedure, not an OS-enforced read-only mount. The worktree is shared and dirty. No HMA file or Git state was edited by this auditor. Start/end status, diff and per-file manifests are retained.
- External AgentSystem verify-loop files changed during inspection: the later skill adds private-clone execution and the script gained matching code. The final external snapshot and hashes define the audited skill version. Earlier reads are not treated as a stable version. No AgentSystem dispatch, model API or network call was run.
- Existing test/vet/dogfood checks passed. Eight independent audit probes intentionally failed expected safety assertions. The first probe build had an unused import, then the default Go cache refused writes. Both failed attempts are retained; corrected probes ran with GOCACHE in /tmp.
- H3 reproduces a crash-window byte state, not a killed process or power-loss durability test. Concurrent writer corruption was not independently reproduced. M4 exploitability remains unproven.
- N1 is independently reproduced at the host gate, not a complete unauthenticated stage transition. Existing transition.Apply still requires its trusted expected/presented approval inputs and legal current stage. No claim of cryptographic protection against a hostile trusted host or same-UID store rewrite is made.
- No actual worker dispatch, route hi, real human challenge ceremony, release CI, token/billing metering, external effects or final verified-success flow was exercised. These would exceed this read-only audit or are not implemented. Baseline remote-origin equality and historical toolchain/test-count claims were not independently verified.
- The HMA snapshot includes all 102 inventoried tracked/nonignored untracked files except .serena. Ignored artifacts, Git metadata and historical logs are not snapshotted. Directory modes and symlink metadata are not preserved by this byte snapshot; the targeted files are regular files.
- Jev was not exposed through the host tool inventory. The auditor used the orchestrator/auditor decision path and did not substitute another model under the Jev identity.

Report packaging check: `validate_report.py` passed for `AUDIT.json` (form only). N1 source locators were corrected to `internal/host/host.go:589-608,635-649` after Phase 1 freeze. No diagnosis, severity or verdict changed. The frozen `PHASE1.json` remains unchanged.

## Exactly one recommended next action (§18)

Ming revises the Gate A packet to resolve A1-A8 and include N1/N2 in Gate B, then submits the frozen revised packet for Liz review.

**Current outcome:** Gate A FAIL. **Rationale:** unresolved authority, lifecycle and proof-plan gaps prevent safe approval of the complete coding plan.
**Owner and proposed route:** Ming; Anthropic / Opus 5.5 / high, through the existing approved host route; exact access-service/runtime identity must be recorded before dispatch.
**Permission envelope:** Proposed documentation-only Gate A revision; no coding, Git writes, release or permission expansion from this audit.
**Exact approval required:** Human decisions on exact architecture/authority changes (D1), Phase B scope or explicit acceptance reduction (D3), and commit authority (D4) before dependent execution. Current dispatch authorizes only this Gate A audit.
**Stop/escalation conditions:** Do not start Gate B while Gate A blockers remain. Stop for unavailable eligible review route, unresolved safety conflict or absent required human decision.
