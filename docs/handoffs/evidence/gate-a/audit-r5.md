# Gate A v5 re-audit

**Gate verdict: FAIL.** R9 closes at design level. R4, R7 and R8 remain partial. A1, A2, A4, A6 and A8 remain partial. R8 is high severity. Gate B must remain stopped.

Observed: seven controller tests and eight Go test functions passed. The local design models reproduced an expiry endpoint acceptance, a final-slot replay refusal and a stale-report acceptance. These models test the written rules. They do not prove future runtime behavior.

**Dispatch:** `hma-gate-a-audit-20261015-r5`. Declared auditor: Liz / OpenAI / `gpt-6.1-sol` / high / review through this managed Codex session. No worker was dispatched by this auditor.

**Isolation: partial.** This is the requested re-audit of known v1–v4 artifacts. It is not a blind first pass. Prior auditor findings are inputs. V3–v5 are objects of audit. No new implementer explanation was accepted after the Phase 1 freeze.

**Scope:** design-level closure for Gate A. Original objective: one HMA-owned outcome lifecycle with bounded host execution, independent evidence and human-gated closure. Sources: `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:7,15,19-48,50-85`; `docs/architecture.md`; `AGENTS.md`; Task 13; Task 14; `TODO.md`. Threat model T is confirmed by the current user: detect trusted-host accident/drift. Same-UID adversarial-host prevention is outside scope.

## Frozen evidence

| Target | Frozen value |
|---|---|
| HMA branch / HEAD | `main` / `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81` |
| HMA manifest | `31d78a6548c738f8b263cedf2ddb7842466b1c34eea38f17d4feb51165e0218e`; 106 files |
| V5 SHA-256 | `663c1626c0a172786567e52c75f1967feb80b58db397b148a2ecae004e790c38` |
| Obsidian HEAD | `9ba1e74a61ef5133f911903cd5c328666e2170ca` |
| Three-file Obsidian manifest | `97f6bd5c14a6320a9df7f3a659c4e7b04de856d377d96e704f0dfa4088b95228` |
| Phase 1 report | `4b99cec6d2a4072dd087ae2fe44ab3028e8e6b14119a976f030066d4d01dc1cf` |

HMA equals the v4 freeze except for added `docs/handoffs/2026-10-15-gate-a-v5.md`. Obsidian is clean. The current controller, controller tests and Verify Loop skill match `9ba1e74a` byte for byte. The actual test/skill paths are under `AgentSystem/agent-memory/`. See `prior-comparison.json`, both manifests and snapshots. `.serena/` is excluded. Git internals and ignored files are excluded.

## 1. Prior-finding closure

| Finding | Status | Reason |
|---|---|
| R4 | PARTIAL | V5 scans beyond the lease window and checks issuance, expiry and revocation. That closes the principal v4 omission under T. The interval is closed at its end: t == valid_until is authorized. The inherited lease rule requires now < expiry. Equal timestamps at a revoke/launch boundary cannot establish safe order. This is ordinary clock granularity/drift detection, not a same-UID adversary claim. |
| R7 | PARTIAL | Replay-first and one-time envelope admission remain repaired. V5 adds current launch eligibility, but defines it through active lease. V3 defines active as counters below ceilings. Reserving the final slot makes that false. A lost final-reservation response then replays with launch_eligible:false even before launch, time expiry, rewind or policy change. Existing-reservation validity must be separate from admission capacity. R4 also retains an endpoint issue. |
| R8 | PARTIAL | Exact expected run/namespace, review mode, completed exit 0 and total verdict mapping are now specified. The prior wrong-mode/failed-worker/unknown-verdict holes close. Mtime and modified-file hashing do not establish that the report belongs to this packet: a wrapper can reserialize an old pass/resolution after start and the current reviewer can complete without replacing it. All named receipt predicates then hold. No clean run-owned absent output or report-body/current-packet equality is required. This remains an accidental stale-output path under T. |
| R9 | CLOSED (design level) | V5 classifies implementation and review launches separately. Reviews match HMA-issued review dispatches, retain stage/independence checks and do not debit implementation attempts. Missing/double matches and write-mode review receipts block. The literal controls authorize one implementation plus one legitimate review and charge one implementation attempt. Future implementation must bind the two named dispatch IDs unambiguously. |
| A1 | PARTIAL | Initial and rework issuance remain tied to authentic legal human-approved transitions and the accepted plan. No initial host self-issued lease path was found. Contract/policy/dependency invalidation remain required. R4 leaves the endpoint unresolved; R7 conflates already-reserved authority with new capacity. V5 grants no new stage authority. |
| A2 | PARTIAL | Stable IDs, before-each-launch D5 hook, pinned live/clone/ledger identity, fresh preflight and HMA live recapture remain required. The current controller supports the planned seam but has no HMA hook yet and normalizes away provider/profile/access service. That is later-gate implementation work, not alone a Gate A defect. R4/R7 remain. The replacement reconciliation prose also omits the retained envelope/admission check (R10); the existing proof row still requires it. |
| A4 | PARTIAL | Cumulative unresolved snapshots, explicit independent per-ID resolutions, same-head evidence repair, reopen counting, kernel disposition and final isolation checks remain required. The total verdict map is repaired. R8 still permits a stale report with apparently fresh file metadata. A stale explicit F1 resolution can therefore be imported as current and hide F1 BLOCK under the named predicates. |
| A6 | PARTIAL | Componentwise admission, atomic shared counters, monotonic spending across renewal, conservative unknown reservations and unmetered-route refusal remain required. R9 closes the review/implementation counting conflict. R7 now refuses recovery of the final reserved slot; R4 retains the time endpoint gap. No unlimited lease renewal or double-debit path was found in the retained counter rules. |
| A8 | PARTIAL | V5 adds useful late-launch, review-class and review-state controls at the assigned later gates. It omits equal-endpoint, capacity-exhausted replay and modified-stale-body controls. V3 still asserts journal/reservation set equality although unlaunched reservations must remain counted; only the launch subset/bijection to consumed launch records is correct. The ./<pkg> command placeholder remains unresolved. These are test-plan gaps, not demands for Gate A runtime implementation. |
| R1 | CLOSED (design level) | Unchanged and retained. V4 uses one target-based issuance rule over the two legal incoming edges. Review/Verification rework goes through human-approved Route selection; Independent validation can return directly to Implementation authorization. The retained active-stage, invalidation and monotonic-counter rules prevent old approvals from reviving after a rewind. No new stage or edge is introduced. |
| R6 | CLOSED (design level) | Unchanged and retained. The required effective-policy digest now has a named location in hma.plan/v1. The existing accepted_plan_digest binds that document to a human approval. Current-policy mismatch voids the lease. Legacy runs without the new plan cannot obtain a lease under the retained fail-closed rule. Later code must canonicalize the effective hierarchy; the existing route-policy normalizer alone does not prove that full hierarchy is already implemented. |

These statuses close design only. `code_fixed` is false and `fixed_revision` is null. The full original closure criteria and source references are preserved per ID in `AUDIT.json`. Runtime proof belongs to the assigned later gates.

## 2. Approval compatibility

**Answer: no intrinsic conflict.** The concrete amendment must preserve §6 authority and serialization semantics.

- V3 drops execution_lease_digest. The only new ApprovalBinding field is optional acceptance_contract_digest with omitempty. V4 policy_digest belongs in accepted hma.plan/v1; findings_snapshot is an additive record flag. V5 changes none of these fields.
- Preserve old field order/serialization and omit absent fields. The 2026-09-08 worktree amendment supplies the compatible optional-binding precedent; populated fields must participate in equality, hash and freshness checks. Old approvals remain valid for their original semantics. Legacy runs cannot derive new leases without an approved new plan/contract.
- A contract run requires its digest on criteria approval and downstream approvals; missing or mismatched contract must fail closed. Do not retroactively require the new field for an old run. New readers may write populated fields that old strict readers reject; version/reader requirements must be documented.
- Eight existing selected Go test functions passed, including legacy golden, worktree JSON/approval binding and legal-edge controls. Those tests do not prove serialization of future fields. D1 direction permits drafting the exact normative amendment; it is not a substitute for the required exact semantic authority.

## 3. Concrete host sequences

| Question | Result under v5 |
|---|---|
| Self-issue an initial lease? | No valid design path found. A host cache grants nothing. A legal human-approved transition and exact plan digest remain required. |
| Count attempts wrongly? | Review counting conflict is repaired. Existing monotonic counters prevent reset. The final reserved attempt loses replay eligibility through its own debit (R7). No double-debit or unlimited-launch path is claimed. |
| Import stale/self review? | Self-review remains forbidden by distinct worker/model/context and risk rules. Wrong mode/run/state now refuses. Reserialized stale output can still satisfy the named predicates (R8). |
| Hide a BLOCK? | Yes, through R8: an old explicit F1 resolution can be exposed as current output. Ordinary advisory-only import cannot erase F1 because cumulative explicit-resolution rules remain required. |

See R4/R7/R8 below for numbered sequences. These are accident/drift sequences under T. No receipt forgery or hidden ledger is required. The predicates model provenance acceptance; later gates must also prove criterion recapture and final closure checks.

## 4. Regressions and actionable findings

V5 repairs the main v4 launch-window omission, the review-accounting conflict, receipt state/mode/run eligibility and incomplete verdict mapping. It adds a replay-capacity conflict (R7). Its replacement scan also conflicts with retained tests (R10). No new illegal core edge or approval field is proposed.

### R8 — Report mtime does not bind report production to the current review

Severity: **high**. Status: **PARTIAL**. Certainty: observed specification/model result. Fix: proposed; no implementation validated.

V5 repairs receipt eligibility but uses post-start modification as a substitute for fresh current-review report production. The receipt can accurately bind the current prompt and current checkout while the declared report still contains an older pass and finding resolution. The import predicates never require a report-body or result-manifest binding to the current HMA packet.

Evidence:

- `docs/handoffs/2026-10-15-gate-a-v5.md:49-58`: Requires current run, terminal review success and post-start file/entry timestamps; does not require absent-at-launch output or body/packet equality.
- `docs/handoffs/2026-10-15-gate-a-v4.md:86-95`: Receipt packet/target and report hashes are checked as separate fields.
- `probes.py / probes.log`: A pre-existing old-target pass/resolution is reserialized after start, producing changed bytes and mtime. All named v5 receipt/mtime checks pass. Failed, running, wrong-run, wrong-mode and untouched-old-file controls fail as expected.

Reproduction / attack sequence: 1. F1 is open against packet P1. A previously valid P0 report says pass and explicitly resolves F1. 2. A trusted wrapper reuses its output path and reformats or rewrites the cached JSON after the P1 review starts. This changes both bytes and mtime. 3. The independent reviewer completes with exit 0 but emits only stdout and does not replace the declared file. 4. D6 truthfully hashes the modified file for the correct P1 run. 5. Import checks P1 receipt, current target, correct namespace/mode/state, new hash and timestamps. Every named predicate passes. 6. The old pass/resolution is treated as current. No forged receipt, hidden ledger or hostile host is required. This is a specification counterexample; no real importer executed.

Expected: Only a report produced for the expected review of the current packet may approve, resolve F1 or credit criteria. Cached old output must be refused even if its file metadata changes during the current run.

Actual: Untouched stale files are refused. Rewritten stale files are not excluded by the written predicates. A new file hash establishes byte integrity, not current packet authorship.

Closure criterion: Reserialize a previous pass/resolution after current start while current completed reviewer produces no report: refuse import, keep F1 and zero criterion credit. A fresh exact-packet report succeeds; wrong packet/body, altered bytes and failed receipt refuse.

Preferred fix: Require a fresh run-owned output directory with no declared report at launch, or capture canonical reviewer output directly. Require the structured report/result manifest to name the HMA-generated packet digest and exact target; compare those fields on import. Preserve receipt run/mode/terminal checks and cumulative explicit resolutions. Do not infer production from mtime alone.

Independent verification:

- Reuse and reformat a prior P0 pass/resolution during a P1 run; reviewer exits 0 without new report → Refuse; F1 remains open; no approval or criterion credit
- Produce a new report bound to P1 and target, then vary only body packet digest or receipt hash → Valid control accepted; wrong-body and tampered variants refused

Trade-offs: Needs an adapter/report binding field and a clean output lifecycle. Valid reviews that fail to emit a bound report must retry or reconcile. This does not claim to prove semantic honesty of a reviewer.

Alternative: Make manual file imports evidence-only. Allow approval/resolution only from a trusted adapter that captures the exact reviewer output and binds it to the current packet.

Effort: medium. Priority: before Gate A acceptance. Code proof stays at the assigned later gates.

Regression classification: Residual prior closure gap.

### R4 — Validity check accepts launches at the expiry or revocation boundary

Severity: **medium**. Status: **PARTIAL**. Certainty: observed specification/model result. Fix: proposed; no implementation validated.

The v5 comparison uses t > valid_until. It therefore authorizes t == valid_until, contrary to the inherited strict wall-time bound. Revocation and launch can also share a timestamp. Trusted host clock values do not establish causal order within that timestamp.

Evidence:

- `docs/handoffs/2026-10-15-gate-a-v5.md:28-43`: The interval check rejects below valid_from and above valid_until only.
- `docs/handoffs/2026-10-15-gate-a-v3.md:47-50`: Active lease requires wall time since approval below max_wall_minutes.
- `obsidian-snapshot/AgentSystem/scripts/verify-loop:66-67`: Controller timestamps use whole-second UTC strings.
- `probes.log`: A launch with start=20 and valid_until=20 is authorized; start=21 is refused.

Reproduction / attack sequence: 1. Reserve D before expiry E. 2. Controller or scheduling delay carries launch to E after its prior hook response. 3. Ledger started_at equals E. V5 authorizes equality. Variant: rewind/terminal and launch share the same timestamp, although launch occurs after revocation. This uses ordinary timing/granularity, not clock forgery.

Expected: Expired/revoked or temporally indeterminate launch authority must fail closed at import under T.

Actual: The principal v4 before/after-window omission is repaired. The exact endpoint and equal-timestamp ordering remain accepted or unspecified.

Closure criterion: For each expiry/rewind/terminal cause test t just before, exactly equal to and after the end; permit only provably valid launches. Same-timestamp ambiguous ordering must not become approval evidence.

Preferred fix: Define half-open validity [valid_from, valid_until). Use t >= valid_until as refusal. Define how core-event/launch ordering is established; conservatively reject ambiguous equal-timestamp revocation ordering. Keep dynamic contract/policy/dependency invalidation and full-lifetime scans.

Independent verification:

- Launch at expiry and at a revoking record timestamp → UNAUTHORIZED_DISPATCH BLOCK; no closure
- Launch provably before end and launch after end → Valid control authorized; late launch blocked

Trade-offs: Conservative timestamp handling can reject a legitimate launch at coarse clock resolution. Use sufficiently precise times and causal/core anchors where practical. Same-UID clock tampering stays out of scope.

Alternative: Use a trusted launch adapter with serialized causal launch authorization and core anchor checks, retaining import reconciliation as a second check.

Effort: small. Priority: before Gate A acceptance. Code proof stays at the assigned later gates.

Regression classification: Residual R4 with a new explicit inclusive-end comparison.

### R7 — Final-slot reservation replay loses launch eligibility through its own charge

Severity: **medium**. Status: **PARTIAL**. Certainty: observed specification/model result. Fix: proposed; no implementation validated.

V5 uses active lease as the replay launch-eligibility test. V3 includes counters below ceilings in active lease. V4 consumption charges a reservation immediately. The final reservation thus exhausts capacity before its launch, so an identical lost-response replay is now ineligible even with unchanged time, stage and policy.

Evidence:

- `docs/handoffs/2026-10-15-gate-a-v5.md:29-31`: Replay adds launch_eligible:false when lease no longer active.
- `docs/handoffs/2026-10-15-gate-a-v3.md:47-50`: Counters below ceilings is an active-lease condition.
- `docs/handoffs/2026-10-15-gate-a-v4.md:55-66`: Replay precedes capacity checks; reservation is counted immediately; exhausted replay must return historical record.
- `probes.log`: Limit=1, spent 0 -> reserve -> spent 1 -> replay active false, launch_eligible false.

Reproduction / attack sequence: 1. Lease has one remaining attempt. 2. HMA appends the reservation and charges it. 3. Hook response is lost before controller launches. 4. Repeat exact key and request digest before expiry/revocation. 5. Historical reservation returns, but active lease is false solely because that reservation spent the final slot. 6. Controller refuses the already-reserved launch. No worker or unknown launch occurred in this model.

Expected: Replay must preserve already-reserved valid launch authority without a second debit. Exhaustion must refuse a new reservation, not retroactively invalidate the reservation that consumed the final slot.

Actual: Counting is conservative and no double debit occurs. The replay repair loses its intended final-slot recovery behavior. Import lease-end semantics also need to distinguish exhausted admission capacity from the existing reservation validity period.

Closure criterion: Lose final response before launch and replay while time/stage/policy remain valid: same reservation, no second charge, one exact-ID launch may proceed. New ID refuses. Expired/revoked or already-used authority must not start another worker.

Preferred fix: Separate new-attempt admission capacity from the validity of a reservation already charged. Compute replay launch eligibility from its own lease generation, validity, current stage/policy/contract, and exact-run launch/reconciliation state. Retain exactly-once canonical dispatch IDs and keep unknown launches charged; do not restore budget.

Independent verification:

- Reserve final slot, lose response, replay before expiry with no launch → Same reservation and charge; reserved launch eligible
- Replay after expiry/revocation, and request a distinct new ID at exhausted capacity → No launch; no new reservation; no counter reset

Trade-offs: Requires explicit reserved/observed/unknown launch state. Unknown launch cannot authorize a fresh ID. A safe permanent refusal is conservative but does not close the previously requested final-response recovery criterion.

Alternative: Explicitly abandon final-slot replay recovery and report a consumed-but-unlaunched attempt. That is a human-visible acceptance change and does not meet the current closure criterion.

Effort: small. Priority: before Gate A acceptance. Code proof stays at the assigned later gates.

Regression classification: New v5 launch_eligible condition reintroduces the exhausted-replay liveness defect.

### A8 — Proof matrix still contains contradictory and incomplete acceptance checks

Severity: **medium**. Status: **PARTIAL**. Certainty: observed specification/model result. Fix: proposed; no implementation validated.

The matrix adds the requested ordinary late-launch and receipt-state negatives. It still lacks controls for the residuals above. Its retained journal/reservation equality row conflicts with conservative unlaunched reservations. Package commands remain placeholders.

Evidence:

- `docs/handoffs/2026-10-15-gate-a-v3.md:148-164`: Command is ./<pkg>; Gate F says journal/reservation sets equal.
- `docs/handoffs/2026-10-15-gate-a-v4.md:66`: Unknown launch stays counted, no cancellation.
- `docs/handoffs/2026-10-15-gate-a-v5.md:42,60-71`: No-ledger reservations stay counted; new matrix does not replace the old equality assertion.

Reproduction / attack sequence: Reserve I, then crash before any launch. The retained authority design keeps I counted. The retained Gate F equality assertion then fails because the launch/journal set is empty but the reservation set contains I. Separately, the new tests can all pass while equal-endpoint, final-capacity replay and reserialized stale-body cases are absent.

Expected: The matrix must describe falsifiable and mutually consistent checks for all seven proofs at their assigned gates.

Actual: The ordinary new tests are useful. The complete closure matrix remains inconsistent and incomplete.

Closure criterion: Replace equality with exactly one reservation per observed implementation launch, no reuse, and separately accounted unlaunched/unknown reservations. Assign endpoint, final-slot replay, stale-body and preserved-envelope checks to concrete packages/adapters and later gates.

Preferred fix: Amend the retained Gate F row to use observed-launch subset/unique mapping and separate conservative reservations. Add named controls for R4/R7/R8 and clarify the R10 invariants. Resolve command package/adapter locations when assigning implementation units; freeze commands, exits and raw results at the assigned gates.

Independent verification:

- Run a crash-before-launch model and one implementation plus one review → Crash reservation remains counted; review excluded; no impossible equality requirement
- Trace proofs 1-7 to package/adapter, gate, owner, command and expected rejection → No material acceptance placeholder or uncovered residual

Trade-offs: Adds small test-plan work now; runtime execution remains later-gate work. Exact test function names alone do not prove coverage.

Alternative: Use a machine-readable acceptance manifest with package/adapter and gate fields instead of prose tables; this is optional and a larger change.

Effort: small. Priority: before Gate A acceptance. Code proof stays at the assigned later gates.

Regression classification: Inherited A8 gaps remain after v5 additions.

### R10 — Replacement reconciliation prose omits retained envelope and cardinality invariants

Severity: **low**. Status: **OPEN**. Certainty: observed specification/model result. Fix: proposed; no implementation validated.

V5 expressly replaces v4 import detection and writes only per-ledger matching and time predicates. V4 also required journal/admission envelope equality and at most one ledger run per reservation. Those operative clauses are omitted, while the v4 proof row still requires envelope mismatch BLOCK. This is an internal specification conflict. The retained proof prevents a claim that the full design positively permits the bypass.

Evidence:

- `docs/handoffs/2026-10-15-gate-a-v5.md:4-5,33-40`: Replacement scan does not mention journal envelope or reverse reservation cardinality.
- `docs/handoffs/2026-10-15-gate-a-v4.md:78-81`: Old scan explicitly requires both invariants.
- `docs/handoffs/2026-10-15-gate-a-v4.md:104`: Retained TestEnvelopeDigestMismatchBlocks still expects BLOCK.
- `obsidian-snapshot/AgentSystem/scripts/verify-loop:217-219,1107-1112`: Controller verifies its own envelope hash, not equality to an independent HMA admission record.
- `probes.log`: Replacement scan alone authorizes in-window I despite Q1 journal vs Q0 admission; no such check is evaluated.

Reproduction / attack sequence: Admit Q0 and reserve I. Submit a self-consistent but wrong/reissued controller journal Q1 while I has a correctly timed ledger row. An importer implementing only the new scan classifies I authorized. An importer honoring the retained v4 test must BLOCK. Both readings cannot be the complete operative rule. This is a specification ambiguity, not a demonstrated bypass of all retained requirements.

Expected: State all reconciliation invariants in one operative contract, with tests that agree.

Actual: New operative scan and retained proof matrix disagree on their complete acceptance conditions.

Closure criterion: Explicitly retain journal envelope == admission envelope, exact controller/live/baseline/namespace association, and each implementation reservation maps to at most one observed launch. Unlaunched reservations remain charged.

Preferred fix: Add the retained equality and reverse-cardinality clauses to v5 reconciliation, or explicitly say the replacement changes only the time/class filter and retains the remaining v4 predicates. Clarify review_dispatch_id versus dispatch_run_id at the same time.

Independent verification:

- Use a self-consistent wrong controller envelope against Q0 admission → UNAUTHORIZED_DISPATCH BLOCK despite valid run ID/time
- Try two observed launches for one reservation; then reserve with no launch → Duplicate refused; unlaunched reservation remains conservatively counted

Trade-offs: This is a small clarification. Canonical dispatcher ID uniqueness can supply part of the cardinality evidence, but HMA still needs to check namespace and association.

Alternative: Publish one consolidated normative reconciliation section and retire inherited delta prose to avoid amendment-selection mistakes.

Effort: small. Priority: before Gate A acceptance. Code proof stays at the assigned later gates.

Regression classification: New v5 replacement-text omission; retained v4 test limits severity.

## 5. Human decisions

**Answer: the recorded set covers the identified product-decision categories. It does not authorize Gate B by itself.**

- T is confirmed in this user request. D1/D3/D4/D5/D6 are recorded in v4 section 0. Their original dialogue is outside this audit evidence.
- D3 includes Phase B. D4 conditions commit-to-main on passing audit and full tests; push remains separately approved. This auditor has no Git write authority.
- Gate A must actually pass before Gate B. Each Gate B bounded unit still needs approved plan/route/permission/budget and authentic challenge-bound stage transitions. No such run-specific packet was supplied here.
- D1 exact architecture text is deferred. Prepare and audit it within authorized docs work; secure exact required human authority before implementation that depends on changed semantics. This is not a demand to halt every routine defect fix until all later-gate documentation exists.
- D5/D6 cover the hook/receipt direction. They do not establish that D5/D6 are implemented, tested or reviewed. Current controller repair 9ba1e74a is present; HMA hook support is not.

## Requirement coverage

| Requirement | Result | Observation | Evidence |
|---|---|---|---|
| Q1-prior-closure | fail | R9 CLOSED (design level); R4/R7/R8 and A1/A2/A4/A6/A8 PARTIAL. R1/R6 remain closed. | `finding_statuses`; `probes.log`; `docs/handoffs/2026-10-15-gate-a-v5.md:12-58` |
| Q2-approval-compatibility | pass | No intrinsic conflict if the concrete normative amendment preserves absent fields and enforces populated contract fields. New-field implementation remains later-gate proof. | `docs/architecture.md:170-202`; `internal/model/types.go:156-183`; `internal/model/terminal_outcomes_test.go:53-87`; `go-checks.log`; `docs/handoffs/2026-10-15-gate-a-v3.md:32-36`; `docs/handoffs/2026-10-15-gate-a-v4.md:44-49` |
| Q3-trusted-host-drift-detection | fail | No initial self-issued lease or legal self-review route found. Stale reserialized output can pass named predicates and hide an old BLOCK resolution. Exact-end launch handling remains unsafe; final-slot replay is refused. | `probes.py`; `probes.log`; `docs/handoffs/2026-10-15-gate-a-v5.md:28-58` |
| Q4-regressions | fail | R7 adds an exhausted-replay liveness regression. R10 is a new low-severity specification conflict. No new illegal core edge or serialization change is proposed. | `R7`; `R10`; `docs/handoffs/2026-10-15-gate-a-v3.md:47-50`; `docs/handoffs/2026-10-15-gate-a-v4.md:55-66`; `docs/handoffs/2026-10-15-gate-a-v5.md:29-40` |
| Q5-human-decisions | pass | No further product-decision category identified. Gate A has not passed. Recorded directions do not replace exact approval of semantic normative text or each unit/stage transition. | `docs/handoffs/2026-10-15-gate-a-v4.md:10-19`; `docs/architecture.md:58-85,136-140,170-202`; `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:50-64,81-85` |

## Verification and evidence ledger

| Evidence | Method | Result and limits |
|---|---|---|
| Freeze | `python3 freeze.py`; read-only Git inventory; SHA-256 snapshots | 106 HMA files; only v5 added since r4. Three current Obsidian files match repair commit. |
| `probes.py`, `probes.log` | Mock current-controller Popen and ledger; literal v5 predicates with positive/negative controls | Exit 0. No real launcher. Models expose R4/R7/R8. R9 controls pass. R10 is a text/test conflict. |
| `controller-unit-checks.log` | `PYTHONDONTWRITEBYTECODE=1 TMPDIR=/tmp/hma-gate-a-20261015-r5 python3 controller_unit_checks.py` | Exit 0; 7 tests. Pure or mocked budget, unsafe-argv, stop and exact-run reconciliation checks. |
| `go-checks.log` | Exact command below, in frozen HMA snapshot | Exit 0; 8 top-level test functions, 2 packages. Existing compatibility and edge controls only. |
| `PHASE1.json` | Frozen before report production; no controlled unblinding | Immutable Phase 1 observations retained. Go completion recorded in final report. |
| `target-verification.json` | Rehash both live target inventories, snapshots, heads and status | All unchanged. |
| `report-validation.log` | Skill `validate_report.py AUDIT.json` | Exit 0; report contract valid, form only. |

```sh
GOCACHE=/tmp/hma-gate-a-20261015-r5/gocache GOTMPDIR=/tmp/hma-gate-a-20261015-r5 TMPDIR=/tmp/hma-gate-a-20261015-r5 GOTOOLCHAIN=local /home/yeli/.local/go/bin/go test -count=1 -v -run '^(TestLegacyApprovalSerializationGolden|TestApprovalWorktreeDigestJSONRoundTrip|TestValidStages|TestLegalEdgeSetIsExactlyTheApprovedTable|TestNoInputAdvancesAStage|TestEvaluateApprovalBinding|TestEvaluateApprovalBindingEnforcesWorktreeBinding|TestEvaluateStageTransitionAcceptsWorktreeBoundApproval)$' ./internal/model ./internal/engine
```

Toolchain: `go1.27.1 linux/amd64`. No full suite was run. Go and controller tests do not prove D5/D6 or the future importer. The current controller still accepts an HMA envelope without a hook, ignores a false hook in the mocked call, and drops provider/profile/access-service fields during normalization. These are explicit later-gate integration requirements, not separate Gate A implementation failures.

## Remediation roadmap

| Priority | Work | Independent exit criterion |
|---|---|---|
| 1 | Bind current review output to current packet and prevent stale output reuse (R8) | Rewritten stale-file negative refuses; fresh exact-packet control imports. |
| 2 | Separate reserved authority from admission capacity; define validity endpoint/order (R7/R4) | Final-slot replay works once; exact-end and revoked launches block. |
| 3 | Consolidate retained reconciliation rules and correct the matrix (R10/A8) | Operative predicates and tests agree; no impossible set equality. |
| 4 | Re-audit the amended frozen design | All scoped Gate A design blockers close; runtime proof remains scheduled. |

No optional redesign is recommended. The proposed remedies are unvalidated design corrections.

## Limits and controlled unblinding

- Partial isolation: this is an expressly requested re-audit, with prior v1-v4 auditor reports available. It is not a blind first pass. V3-v5 contain author explanations/success claims as objects of audit. No new implementer explanation was requested or accepted after the Phase 1 freeze.
- The managed auditor route is declared by the user as liz / OpenAI / gpt-6.1-sol / high / review. Host/runtime identity and dispatch preflight were not independently attested. No subagent, nested agent CLI, model API or real worker was launched.
- Both repositories were read-only by procedure, not by an OS read-only mount. No repository edits or Git writes ran. All snapshots, helper scripts, probe files, reports and Go caches are below this output directory. Source .serena, Git internals and ignored files are excluded from the HMA audited inventory.
- Gate A is design-level only. HMA reservation/importer, D5 authority hook and D6 receipt fields remain future implementation. No missing future code was used alone as a Gate A blocker. No proposed remedy is labelled runtime-validated.
- Bounded tests only: eight existing Go test functions in model/engine, seven pure or mocked controller tests, a mocked dispatch observation and literal design models. No full suite, real dispatch, end-to-end billing/token meter, concurrency integration, migration, final closure or CI release ran.
- Literal model counterexamples test the written predicates. They do not demonstrate runtime exploitation of a future importer. The stale-file sequence requires an accidental wrapper rewrite/reuse and current completed worker omission; an untouched stale file or failed receipt is correctly refused.
- R10 is a specification conflict, not a proved bypass of the full inherited matrix. The retained envelope-mismatch test still requires BLOCK. Report wording does not elevate that omission to a high-severity authority failure.
- T is confirmed by the current user. The underlying human dialogue for D1/D3/D4/D5/D6 was not inspected; they are evaluated as recorded decision claims. The Obsidian repair commit and its three relevant current file bytes were independently verified.
- JEV_UNAVAILABLE: exposed tool inventory contained no approved callable Jev decision tool. Auditor judgment is the recorded fallback. No model was presented as Jev.
- Initial freeze invocation used unavailable python, then succeeded with python3. One exploratory ledger path was absent; no claim depends on it. Some broad tool displays were truncated; material sources were reread in targeted ranges and preserved in snapshots.
- The report validator checks structure only and disallows pass under partial isolation. Gate FAIL is independently supported by the unresolved high R8 finding. No gate PASS is inferred from test passes.

No post-freeze implementer explanation was supplied. No blind-pass claim is made. The exact phase freeze, report validator result and final target-integrity check are recorded in `AUDIT.json`. The validator checks form, not evidence truth.

## Required next action — REPLAN

Amend the frozen Gate A design to resolve R4/R7/R8/A8 and clarify R10, then submit the amendment for the same read-only re-audit. Keep Gate B stopped.

Current outcome: Gate A FAIL. R9 closes at design level; R4/R7/R8 and A1/A2/A4/A6/A8 remain partial.

Rationale: The current design still admits stale report output under T and has boundary/replay/test-plan defects. These are bounded corrections to the existing objective.

Owner: Ming. Route: Ming / Anthropic / claude-opus-5-5 / high as in the original lead assignment, via the approved Claude Code CLI host profile. Revalidate exact route and access service before any worker dispatch; this report dispatches nobody.

Permission envelope: Authorized Gate A design text only. No Gate B source changes, Git write, push, release, scope expansion or added budget is authorized by this audit.

Exact approval required: No new product-decision category is identified for these repairs. Use recorded D1/D5/D6 directions within their envelope. Exact normative authority/compatibility amendment and every applicable unit/stage transition still need the authentic human authority required by the architecture. A passing audit cannot provide it.

Stop/escalation conditions: Keep Gate B stopped until Gate A passes. Escalate if repair changes success criteria, threat model, risk, permissions, cost or approval semantics beyond recorded directions, or conflicts with the deterministic kernel.
