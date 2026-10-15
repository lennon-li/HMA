# Gate A v3 re-audit

**Dispatch:** hma-gate-a-audit-20261015-r3. **Frozen assessment:** 2026-10-15T14:29:40.493526+00:00.

**Gate verdict: FAIL.** Report-contract verdict: `fix-required`. Four of the 11 requested findings are CLOSED at design level. Seven remain PARTIAL. Initial lease issuance, dependent invalidation, escalation stopping, disposition rules and empty-snapshot representation improved. Renewal, launch accounting and review authentication still prevent design acceptance.

**Isolation: partial.** This is a re-audit using my v1 and v2 findings. It is not a blind first pass. No implementer explanation or defense was supplied. V3 is the object of audit. The skill validator cannot issue `pass` with partial isolation. The substantive high findings independently require FAIL.

**Authority and scope:** The original objective is `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md`, especially §§0–6. Norms are architecture, AGENTS, Task 13, Task 14 and TODO. Gate A needs a coherent design. Code proof belongs to later gates. This audit does not treat unimplemented future tests or the absent future hook as separate Gate A defects.

**Route and permissions:** Declared auditor: liz / OpenAI / gpt-6.1-sol / high / review mode. Access: current managed Codex tool session. Actual runtime settings and earlier dispatch preflight were not verified. No subagent or nested agent CLI was used. Both repositories remained read-only. Jev was unavailable; the specified auditor fallback was used.

## Frozen target and checks

- `main`, HEAD `118ea3ae4daf0fb9a7c7ae2f3590a40feea19d81`.
- V3 SHA-256: `574dbfe98c6a0aa5b7c928e65873d9ed50fffd9905b23f4cba67a04566c8268c`.
- Start manifest SHA-256: `5421a92a8b59ac9e601b45a13b14cdb83d446200ad1a513b1524c471e1d4e7a6`.
- Tracked diff SHA-256: `ce2c0caf6c85ea3539542511709e1e3e7e96dc7fa299316a53c7ad6b1466b66c`.
- All 103 prior v2 files match. V3 is the only addition. The snapshot contains 104 files. `.serena/` is excluded.
- `snapshot/` preserves source bytes. `external-snapshot/` preserves the current controller inputs and prior auditor reports. `supplemental-snapshot/` preserves later-inspected ledger/dispatch and skill-contract sources. Manifests contain complete hashes.

The resolved external paths are `AgentSystem/scripts/verify-loop`, `AgentSystem/agent-memory/boot/test-verify-loop.py`, and `AgentSystem/agent-memory/skills/verify-loop.md`. These are the actual boot/skill locations. The current uncommitted repair was inspected. The controller SHA-256 is `be08b0d5460479e585bd942fb8eafd7c046100b210210d7a4e79a9c89c30f27c`.

**Executed checks:** 15 top-level Go test functions passed: 2 new pure probes and 13 existing functions. Three mocked controller checks passed. Two literal design traces expose reservation contradictions. A passing probe confirms the observed behavior; it does not certify the proposed design. No real worker, model API or Git-writing test ran. See `go-probes.log`, `controller-probes.log`, the probe sources and `overlay.json`.

```sh
GOCACHE=/tmp/hma-gate-a-20261015-r3/go-cache GOTMPDIR=/tmp/hma-gate-a-20261015-r3/go-tmp /home/yeli/.local/go/bin/go test -count=1 -v -overlay /tmp/hma-gate-a-20261015-r3/overlay.json -run 'TestGateAV3|TestLegacyApprovalSerializationGolden|TestApprovalWorktreeDigestJSONRoundTrip|TestEvaluateApprovalBinding|TestEvaluateResolution' ./internal/engine ./internal/model
PYTHONDONTWRITEBYTECODE=1 python3 /tmp/hma-gate-a-20261015-r3/controller_probes.py
```

The existing controller preserves an unknown `authority_hook` field but does not call it. The mocked launch proceeds despite `/bin/false`. Its route normalizer drops added profile/provider/access-service fields. With `escalate_to=stop`, it does stop after two unsuccessful interventions. These observations constrain D5. They are not claims that the planned hook already exists.

## 1. Prior finding closure

CLOSED means **design level only**. No affected code is fixed in this unchanged target. The exact inherited closure criteria remain in `AUDIT.json` and the frozen v2 report. V3 is read charitably as a delta that retains v2 cumulative unresolved findings, explicit independent resolutions, reopen IDs, same-head evidence repair, full section 17 rules and contract ceilings. Those retained rules should be explicit in the next normative draft. A standalone reading of v3 that drops them would not close A4/R5.

| ID | Status | Assessment |
|---|---|---|

| A1 | **PARTIAL** | The plan digest now supplies a non-circular human-bound source for initial unit authority. The approved dependency graph and fail-closed legacy rule close the previous optional-list gap. Initial route equality is required. The claimed approval-bound policy has no defined source (R6). Renewal still uses an illegal edge and an inconsistent issuance rule (R1). Launch-time authority remains incomplete (R4). Source: `docs/handoffs/2026-10-15-gate-a-v3.md:32-65,71-98`. |

| A2 | **PARTIAL** | The bridge now carries stable attempt IDs, envelope digest and route JSON. It serializes reservation checks, retains unknown reservations and re-captures live checks. It still enforces hook omission only at import. It does not bind the live/clone baseline and ledger namespace to a one-launch authorization or define expiry/cancellation across the hook-to-launch interval (R4). The current route normalizer drops profile/provider/access-service facts; D5 must address actual route resolution, not only insert a call. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:71-104`. |

| A4 | **PARTIAL** | The reference-based disposition map, explicit empty snapshots, independent criterion credit, and verdict table address major prior gaps. Cumulative unresolved findings, explicit resolutions, reopen IDs and same-head evidence repair remain obligations from v2, read as retained by this delta. Receipt existence and eligible model/provider do not authenticate the report output, exact packet or context lineage (R8). A substituted resolution could therefore erase an earlier BLOCK. The verdict table also needs explicit handling for conditional-pass and not-evaluated and a fail-closed default. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:108-138; v2:103-116`. |

| A5 | **CLOSED (design level)** | The inherited checkpoint table includes metadata, external effects, final closure, semantic changes, policy/evidence/criteria bindings and a compiled minimum. V3 adds disputed-validation separation, fail-closed unknown classification, distinct worker and separate context. This closes the checkpoint-policy criterion at design level. Import authentication remains a separate A4/R8 gap; no runtime enforcement is claimed. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:140-144; v2:120-135`. |

| A6 | **PARTIAL** | Componentwise budget ceilings, atomic shared reservations, monotonic spending across renewal, unknown-launch counting and stop-on-escalation are stated. Direct work remains counted under inherited v2 rules. The literal check order defeats last-attempt idempotence, and comparing the full immutable envelope with remaining counters rejects a second intended attempt (R7). Actual launch identity/lifetime, direct-work reservation and cancellation remain incomplete under R4. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:71-102; v2:137-146`. |

| A8 | **PARTIAL** | The matrix now assigns Ming/Liz ownership, lists seven proofs, adds many refusal controls, and names the real Gate F harness. It is a design test plan, not executed code proof. The command still uses an unresolved ./<pkg> placeholder; more materially, the R4 closure schedules (lost response, reservation-to-launch revocation, conflicting-ID replay, hook omission before dispatch) and report-output substitution are not assigned explicit checks. Add these to the planned matrix; execute only at the assigned later gates. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:146-165`. |

| R1 | **PARTIAL** | Initial issuance on ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION is legal, before work, and avoids the v2 digest cycle. The walkthrough then proposes IMPLEMENTATION_REVIEW -> IMPLEMENTATION_AUTHORIZATION, an unlisted edge. It says a rewind approval issues a fresh lease although the sole issuance edge remains route selection -> authorization. Initial issuance is repaired; renewal is not coherent. The independent Go probe confirms the edge mismatch. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:42-58`. |

| R2 | **CLOSED (design level)** | Unlinked low/high observations are advisory. Severity does not create blockers. Reproduced kernel violations remain unwaivable BLOCK; measurable complexity exceedance is WAIVABLE; fabricated/non-reproducing criterion links are advisory. A failed criterion check prevents criterion credit. These satisfy the specific v2 closure controls without adding acceptance requirements. Preserve semantic-validator scope under architecture sections 9 and 16; the importer must not claim that a green check proves all semantics. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:108-117,137-138,160`. |

| R3 | **CLOSED (design level)** | The lease authorizes only the approved implementer route. Every different route requires human classification confirmation and a new approved transition. HMA-bound envelopes must use escalate_to=stop. The current controller actually stops on that value after two unsuccessful interventions. Same-route retries still inherit the complete section 17 classification and human-confirmation requirements; this finding does not grant automatic classification authority. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:69-77,157`. |

| R4 | **PARTIAL** | Stable IDs, a per-run lock, immediate conservative counting, monotonic unit counters, and optional result receipts are substantial corrections. A mandatory prelaunch admission path, request/response schema, ledger/baseline binding, launch validity window, conflicting duplicate handling and terminal receipt reconciliation are still missing. Import-time detection cannot govern a launch that already happened. R7 records two additional contradictions introduced by the reservation algorithm. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:79-104`. |

| R5 | **CLOSED (design level)** | An omitted-false additive findings_snapshot flag distinguishes an intentional empty replacement from an omitted snapshot. An independent exact-target import is required for empty replacement. With the retained cumulative/explicit-resolution rule, this closes absent-versus-empty projection semantics. Old record serialization and reducer behavior must remain unchanged when the flag is absent. Authentication of the supposed independent import remains A4/R8, not proof that the flag itself is wrong. Source: `docs/handoffs/2026-10-15-gate-a-v3.md:119-124,160; v2:109-116`. |

The earlier design closures A3, A7, N1 and N2 are not reopened as demands for Gate A code proof. Gate B still needs the original A-versus-B review/artifact controls and nonrepository/wrong-checkout/dirty-content/mutating-command capture controls. D4 must preserve the reviewed clean-start sequence described below.

## 2. Approval fields and digest compatibility

**Observed:** Architecture §6 permits the optional exact worktree binding from 2026-09-08. Task 13 uses the same additive, omitted-empty approach for terminal targets. `ApprovalBinding` is serialized into hashed records. Expected/presented approvals are compared field for field. The legacy approval golden test and worktree round-trip test passed in this re-audit.

**Assessment:** The proposed optional `acceptance_contract_digest` does not inherently conflict with §6. D1 must explicitly amend the normative approval contract. `omitempty` can preserve existing JSON bytes and digests when the field is empty. V3 drops `execution_lease_digest`. Deriving the lease from `{approval transition digest, unit, plan digest}` is non-circular because the derived lease is not itself hashed into that approval. Using `accepted_plan_digest` to bind a canonical plan is compatible in principle.

The additive `findings_snapshot` flag can also preserve old bytes when false/absent. The reducer must retain legacy nonempty-snapshot behavior and distinguish an explicit new empty snapshot. This is not permission for ordinary old empty records to clear findings.

**Qualifications:** These fields are not implemented. Keep field order, JSON tags, nil/empty semantics, legacy defaults and hash projections stable. Do not populate defaults while reloading historical records. Test populated mismatch, omitted-field goldens and legacy chain/bundle loading. Old strict readers need not accept new populated fields; define the new-record reader/version boundary. Contract-use runs must enforce the new binding without retroactively invalidating valid legacy history. Contract revision must follow the approved criteria rewind and never resurrect an old approval. Exact runtime/profile/private-path data must remain host-local; portable plans may bind opaque attestations.

**Remaining defect:** V3 compares current policy to an approval-bound policy without naming its stored source (R6). The worktree precedent establishes neither that missing binding nor a new stage edge. Compatibility of the additive field does not close R1/R6.

## 3. Adversarial sequences

| Question | Assessment | Concrete sequence or refusal |
|---|---|---|

| Self-issue initial lease? | Explicitly refused at design level. | Write arbitrary lease L; present no authentic applicable route approval or a plan whose hash differs. V3 recomputes the plan and derives authority from the real unit approval; the cache grants nothing. Copying another run/unit/repository approval cannot meet universal binding. No compliant initial self-issuance bypass was established. Policy and renewal gaps are R6/R1. |

| Use a stale dependent lease? | The previous list-omission attack is closed at design level. | Approve B dependent on A; rewind A. Derive transitive dependents from the approved plan and void B. Missing or invalid plan voids all leases. Legacy runs without the new plan cannot acquire a lease. |

| Launch without a hook? | Import-time detection is too late (R4). | Omit the optional controller field, execute, then import. Import records unauthorized-dispatch BLOCK, but does not prevent earlier execution. A trusted HMA-bound admission flag and before-launch refusal are required. Current-code mocked Popen confirms no existing hook enforcement; future enforcement remains to be designed. |

| Reuse a delayed reservation? | Not excluded (R4). | Reserve while valid, pause, then expire/revoke authority before launch. A cached reservation has no stated consumption/deadline/cancellation rule. No actual HMA exploit is claimed. |

| Miscount attempts? | Duplicate namespace and replay schedules remain (R4/R7). | Two controller state/ledger namespaces reuse the same tuple. HMA can return one reservation while each ledger launches once. Separately, a lost response at the last permitted attempt is rejected before the idempotency lookup. A full five-attempt immutable envelope can also fail the second remaining-budget comparison. |

| Change route automatically? | Explicitly refused; R3 closed. | P fails twice. escalate_to=stop makes the current controller stop. An alternate route E, even in the proposed plan, requires human-confirmed classification and a new approved transition. The hook must reject E under P authority. |

| Import stale review? | Target-digest rule is sound; receipt substitution remains R8. | A genuine old report for A cannot satisfy current head/diff/worktree/evidence/policy/contract comparisons. But attaching its genuine eligible receipt to a new forged report B is not excluded by receipt-exists/model checks unless output and launched packet are bound. |

| Import self review? | Direct self-review is forbidden; identity borrowing remains R8. | Same worker/model/context is refused by preserved rules. An implementer can instead submit self-authored B with an unrelated genuine independent receipt. Trusting asserted isolated context and current target does not bind B to that worker output. Contribution-aware review must also cover all unreviewed contributors in multi-attempt work. |

| Hide BLOCK? | Cumulative rule forbids ordinary overwrite; forged resolution still matters. | F1 BLOCK followed by advisory F2 must retain F1. A genuine exact-target explicit independent resolution may clear it with the new flag. Under the R8 substitution schedule, a fabricated resolution can borrow an independent receipt and request an empty snapshot. The importer must reject that payload and retain F1. |

These are bounded design counterexamples. They do not assume the host may forge a human stage approval, rewrite trusted store history undetected, or bypass the OS. The proposed importer and attempt hook are not implemented. No current-code exploit of those future components was reproduced.

## 4. Actionable findings and new regressions

R1 and R4 below are residual prior findings. R6–R8 are newly numbered. R6/R8 make prior incomplete authority/provenance obligations precise. R7 is a new contradiction introduced by v3. The direct review-to-authorization walkthrough in R1 is also new and wrong. No new acceptance criteria are invented.

### R1 — Lease renewal still names an unlisted edge and a second issuance rule

**HIGH · observed · PARTIAL.** Residual R1; initial v2 stage defect repaired, new v3 renewal walkthrough is wrong.

**Source/evidence:** `docs/handoffs/2026-10-15-gate-a-v3.md:42-58; docs/architecture.md sections 5.4,6; internal/engine/transitions.go:65-92,139-153; go-probes.log:TestGateAV3IssuanceAndRenewalEdges`. The pure Go probe accepts initial route issuance, rejects review -> authorization with UNLISTED_EDGE, and accepts review -> route -> authorization and independent-validation -> authorization.

**Actual:** V3 repairs initial issuance, then walks directly from IMPLEMENTATION_REVIEW to IMPLEMENTATION_AUTHORIZATION. Architecture section 5.4 and IsLegalEdge reject that edge. Lines 53-54 also say a rewind approval itself issues a fresh lease, whereas lines 42-46 make the route-selection approval the only issuer. A legal INDEPENDENT_VALIDATION -> IMPLEMENTATION_AUTHORIZATION rework transition needs an explicit lease treatment as well.

**Expected:** Use only legal challenge-bound edges. Define exactly which fresh approval activates renewed work. Preserve spent counters and never reactivate invalidated authority.

**Reproduction or attack:** Approve a route, execute work, and advance to implementation review. Follow v3 line 57 to authorize a correction directly. The core rejects the edge. Bypassing that rejection violates the kernel. Taking the legal return to route selection instead needs a fresh route approval; the old rewind approval is not the issuer defined at line 42.

**Preferred fix (proposed):** Correct the walkthrough to IMPLEMENTATION_REVIEW -> ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION. Specify that only the new final edge issues the fresh derived lease. State how the already-legal validation -> authorization edge is handled: either explicitly bind a permitted new derivation there or require the existing legal route-selection path before work. Do not add an implicit stage edge.

**Closure checks:**
- Trace initial issuance, review rejection, legal route return and renewed issuance → No launch before new applicable issuance; all legal edges have fresh human approvals; counters persist.
- Exercise legal validation rework and dependency/whole-run rewind → One documented legal activation rule; no old approval becomes active again.

**Trade-offs:** An extra route-selection gate adds a human step. It preserves the current architecture and avoids a new approval field. A new issuance edge is a larger D1 amendment.

**Alternative:** Define a separately approved preimage for lease activation on the existing legal validation rework edge, while keeping review rework on the route-selection path.

**Effort:** small. **Priority:** Before Gate A design acceptance; code proof at assigned later gates. **Fixed revision:** none. This audit did not implement or validate the remedy.

### R4 — The reservation seam still detects missing authority after execution

**HIGH · inferred · PARTIAL.** Incomplete R4/A2/A6 repair; v3 adds some controls but does not complete the prior closure criterion.

**Source/evidence:** `docs/handoffs/2026-10-15-gate-a-v3.md:81-104; v2:88-98; AgentSystem/scripts/verify-loop:363-434,830-897,1091-1126,1264-1299; AgentSystem/agent-memory/skills/verify-loop.md:43-68; controller-probes.log`. Mocked dispatch_agent reaches Popen with authority_hook=/bin/false; route validation removes profile_digest/provider/access_service. The current controller uses a per-state-directory dispatch-ledger. The ledger prevents duplicate IDs within one ledger directory, not across arbitrary namespaces. V3 has no future protocol that binds this namespace or baseline to the reservation.

**Actual:** V3 makes hook presence mandatory at evidence import, after worker effects. It gives no trusted admission rule that marks an HMA-bound job independently of its optional hook field and refuses launch without authorization. The request omits live/clone baseline and ledger namespace. The response has no stated launch validity, consumption or cancellation contract. A reservation alone does not prove which launch it authorized. The current controller has no hook call; this is an expected future change, but the proposed D5 scope must define all required enforcement.

**Expected:** Every actual launch, including direct work and resume, must consume current HMA authority before effects. One reservation must authorize at most one exact launch. Reconciliation cannot issue new authority.

**Reproduction or attack:** Sequence 1: create an HMA-intended envelope without authority_hook; launch through the controller; import later. V3 rejects import and records BLOCK, but work already ran without the hook. Sequence 2: obtain a reservation while the lease is active, pause before Popen, then expire/revoke the lease or change policy and launch on the returned record. No reservation expiry or launch-time consumption rule is specified. Sequence 3: two controller instances in distinct state/ledger namespaces reuse the same controller/attempt/dispatch tuple. The host returns the same reservation, and each private ledger can launch that run ID once. Import matching by dispatch_run_id does not distinguish two launches. These are missing-protocol counterexamples, not reproduced exploits in nonexistent HMA code.

**Preferred fix (proposed):** Add a versioned HMA job admission record that pins run/unit/lease, actual live and clone identities, envelope/check definitions, ledger namespace, and resolved exact profile/route. Require a successful typed reservation before launching any HMA-bound job. Bind immutable request content to the idempotency key and reject conflicting replays. Define one-launch consumption, a validity deadline bounded by the lease, atomic revocation/launch ordering, timeout/cancellation, and exact terminal receipt reconciliation. Apply the same attempt abstraction to direct work. Keep import checks as defense in depth.

**Closure checks:**
- Try an HMA-bound job with omitted hook, invalid hook, timeout and unclassified/unsafe failure → No worker Popen occurs; record refusal and the appropriate next action.
- Reserve, delay, revoke/expire, then attempt launch → No launch under invalid authority; unknown launch remains spent and cannot be reused.
- Replay the same IDs in two ledger namespaces or change immutable payload under one key → Second conflicting launch is refused; identical recovery reads return the same status without a new launch.
- Trace clone/live baseline, actual resolved route, preflight and canonical result → All records name the same bound job and launch; unverified route fields cannot satisfy equality.

**Trade-offs:** This is a trusted-host protocol, not OS protection against a hostile same-UID store rewrite. Fail-closed unknown reservations can consume unused budget. Do not promise journal/reservation set equality when a crash can leave a reservation without a launch; require every launch to have one reservation and separately account for unlaunched reservations.

**Alternative:** Serialize the first integration through one HMA-managed controller/ledger namespace, prohibit concurrent/nested jobs and direct execution, and use escalate_to=stop. Stable consumption, expiry and prelaunch admission are still required.

**Effort:** medium. **Priority:** Before Gate A design acceptance; code proof at assigned later gates. **Fixed revision:** none. This audit did not implement or validate the remedy.

### R6 — The approval-bound policy has no defined persisted source

**HIGH · observed · OPEN.** Newly articulated residual A1 gap; not a claim that optional approval fields are incompatible.

**Source/evidence:** `docs/handoffs/2026-10-15-gate-a-v3.md:30-50; v2:42-54; internal/model/types.go:163-186; internal/engine/approval_binding.go:39-69; go-probes.log:TestGateAV3ApprovalPolicyFieldAbsent`. The pure reflection probe confirms no direct policy_digest field today. The issue is the missing design location, not an assertion that a direct field is the only solution.

**Actual:** The active-lease predicate compares current policy with the approval-bound policy. ApprovalBinding has no policy field. V3 adds only acceptance_contract_digest; its specified acceptance-contract and plan field lists do not name a policy digest or a typed required-evidence reference that carries it. The design states a desired comparison without defining the human-bound value to compare. A policy hash in an audit packet is not the earlier execution approval binding.

**Expected:** Pin the exact effective policy before execution through an existing human-bound digest or a specifically approved new binding. Refuse absent/stale policy without selecting a permissive default.

**Reproduction or attack:** Approve plan P under effective policy Q0. Change the current policy to Q1 while leaving the specified plan/contract bytes unchanged. At the hook, ask for the approval-bound policy hash. No field or derivation is defined. A strict implementation cannot activate any lease. An implementation that fills the missing value from current policy or an unbound host cache can accept Q1 under the Q0 approval. The latter is a design counterexample, not a reproduced current-code bypass.

**Preferred fix (proposed):** Add an effective-policy digest to the canonical approved plan (or explicitly to the acceptance contract) and define the digest over the effective hierarchy and relevant route/checkpoint policy versions. Require equality with the current deterministic projection and the human-approved document. Name this field and its invalidation rule in D1. No extra ApprovalBinding field is necessary if the existing accepted_plan_digest covers it.

**Closure checks:**
- Approve Q0, then change effective policy at the same head/plan unit → Next launch refused until applicable fresh human authority binds the new policy.
- Omit policy data, submit a host-only cache, or reorder semantically identical policy input → Missing/unbound policy refused; equivalent normalized policy yields the same digest.
- Load a legacy approval without a new plan/contract → No retroactive requirement for old transitions; no new lease is derived without the new required authority document.

**Trade-offs:** A new policy-aware plan version needs documented canonicalization and migration. Legacy approval bytes can remain intact. The kernel still dominates all policy layers.

**Alternative:** Bind a typed policy-evidence artifact through RequiredEvidenceDigests, with explicit source and projection rules. This uses existing approval fields but needs equally clear validation.

**Effort:** small. **Priority:** Before Gate A design acceptance; code proof at assigned later gates. **Fixed revision:** none. This audit did not implement or validate the remedy.

### R7 — The new check order breaks replay and a second intended attempt

**HIGH · observed · OPEN.** New v3 algorithm contradiction; affects A2/A6/R4.

**Source/evidence:** `docs/handoffs/2026-10-15-gate-a-v3.md:47-50,81-98,157-158; AgentSystem/scripts/verify-loop:213-217,952-989,1258-1286; controller-probes.log; controller_probes.py`. A literal executable model of v3 step order reproduces both contradictions. It is explicitly labeled a design trace; it is not an implementation of hma host attempt. With ceiling 1, replay is rejected as exhausted. With ceiling 5 and an immutable five-attempt envelope, the second distinct attempt is rejected because 5 > 4.

**Actual:** Step 1 checks an active lease and remaining counters before step 2 looks up an existing reservation. Reserving the final permitted attempt exhausts the counters. A replay after a lost response is therefore refused before the promised idempotent result can be returned. Also, the rule that envelope ceilings must fit remaining counters is applied at the per-attempt seam without distinguishing admission from consumption. The current controller pins max_attempts in one immutable envelope. After one reservation, the original full envelope no longer fits the remaining attempt budget.

**Expected:** Identical recovery reads must be idempotent even after capacity is fully reserved. New attempts must fit the remaining unit budget. An already-admitted loop must not reserve its full original limit again for every attempt.

**Reproduction or attack:** Approve one attempt; reserve K; lose the success response before launch; retry K. Step 1 sees spent=1 and limit=1, so it refuses before step 2. Separately, admit max_attempts=5 under remaining=5; reserve attempt 1; call the hook for attempt 2 with the same immutable envelope. The written ceiling rule compares 5 to remaining 4 and refuses. Neither schedule permits the intended recovery/multi-iteration behavior.

**Preferred fix (proposed):** Separate read/reconciliation of an existing immutable reservation from permission for a new launch. Under the lock, check key/payload equality first; return existing status even when exhausted, but mark whether launch remains permitted under current authority. Apply full envelope ceilings once at admission. For later new reservations, compare the incremental attempt/cost/time requirement and the unconsumed loop allowance against remaining unit totals. Preserve conservative spending.

**Closure checks:**
- Lose the response for the final reservation and repeat an identical callback → Same reservation/status returned with no second charge; no new or stale launch authority.
- Run two distinct attempts from an admitted five-attempt envelope under total ceiling five → Both valid attempts reserve once; attempt six refused; another controller cannot overspend.
- Create an unlaunched reservation and end the run → Reservation remains accounted separately; launch IDs are a subset of reservation IDs rather than an unjustified equality assertion.

**Trade-offs:** Retrieving a historical reservation must not authorize a launch after expiry. Tracking per-loop admitted allowance adds bookkeeping but avoids modifying immutable envelopes. This is liveness/accounting repair, not permission to expand budgets.

**Alternative:** Use a single controller admission that reserves the full declared loop allowance once, then suballocate stable attempts within that reservation. Unknown subattempts remain consumed; unused allowance needs an explicit conservative release policy.

**Effort:** medium. **Priority:** Before Gate A design acceptance; code proof at assigned later gates. **Fixed revision:** none. This audit did not implement or validate the remedy.

### R8 — Receipt existence does not bind the reviewer output to the review packet

**HIGH · inferred · OPEN.** Residual A4 provenance gap in the new v3 receipt adapter; also affects safe empty-snapshot use.

**Source/evidence:** `docs/handoffs/2026-10-15-gate-a-v3.md:121-138,142-144; v2:103-116,131-135; internal/host/host.go:567-607; AgentSystem/scripts/agent-run-ledger:271-365; AgentSystem/scripts/verify-loop:697-716`. The inspected ledger payload has run_id/agent/model/effort/mode/project_root and output paths/counts. It does not contain the packet, dirty-worktree, output-content or account-home identity assumed by the import description. Verify Loop separately hashes canonical receipt/output files; those hashes still need a trusted HMA review-dispatch-to-packet binding for audit imports.

**Actual:** V3 checks that a supplied agent-dispatch receipt exists and that its model/provider meets independence rules. It does not define equality between the actual launched review packet/context, the current report bytes, and the canonical worker output. The current ledger receipt has mode, project root, head, paths and byte counts, but no packet/worktree/content-output hashes, provider or account_home field. Account home is not evidence of a separate review context. Preserving host distinct-worker/context checks is necessary, but those asserted fields do not authenticate an imported report by themselves.

**Expected:** Only the exact successful independent review of the frozen current packet may approve or resolve its findings. Resolve provenance from trusted launch records and captured output, not from report route/isolation strings or unrelated receipt existence.

**Reproduction or attack:** Obtain a genuine eligible reviewer receipt for target A or an unrelated prompt at the same Git head. Submit implementer-authored report B that claims pass, isolated context, the current target and resolved F1, while attaching that genuine receipt digest. Its model/provider and ledger existence pass the described receipt checks. Unless the importer also binds dispatched packet and captured report output, B can receive APPROVE and an empty snapshot can remove F1 BLOCK. A self-authored report can thus borrow an independent identity without forging the ledger. Multi-writer variant: A writes one part, B repairs another, then A reviews the aggregate using only B as the latest implementer; the plan must ensure A's own contribution already has an eligible review. These are unexcluded design attacks, not a reproduced importer implementation.

**Preferred fix (proposed):** Bind an HMA review dispatch to packet/target/evidence/policy/contract digests, approved read-only route, actual separate context, and all relevant implementation contributions. Pin the canonical ledger namespace and exact terminal run. Capture report bytes or an output manifest under trusted host control and require digest equality to that dispatch result before importing. Check terminal success and review mode. Derive provider/model-family facts from an authenticated approved profile, not nonexistent receipt fields. Retain cumulative findings and require explicit per-ID independent resolution. Complete the verdict map for conditional-pass/not-evaluated and reject unknown verdicts by default; define REQUEST_CHANGES as a non-approving persisted result without changing legacy BLOCK semantics.

**Closure checks:**
- Attach a genuine receipt for another packet/prompt, another namespace, altered report bytes, wrong mode or failed worker → Import refused; F1 BLOCK remains; no criterion credit.
- Use the correct packet/output but same worker/model/context or a contributing implementer → Independence refusal according to normal/high/disputed/final risk; valid independent control succeeds.
- Import F1 BLOCK, then advisory F2, then a genuine exact-target explicit resolution with empty snapshot → F1 survives F2; only the authenticated resolution clears it; stale or fabricated empty imports cannot clear it.
- Exercise every skill report-contract verdict and an unknown string → Only an eligible authenticated pass can approve; all other cases have explicit non-approving results.

**Trade-offs:** This can be implemented with a host-owned receipt/output manifest and existing HMA dispatch records; no new identity service or trust in worker prose is needed. Family separation and separate contexts remain mandatory. Ledger output hashes alone do not prove semantic correctness or actual blinding.

**Alternative:** Initially accept reports only through a trusted local review adapter that launches the frozen packet and directly captures its output, while preserving manual import as evidence-only and never approval-capable.

**Effort:** medium. **Priority:** Before Gate A design acceptance; code proof at assigned later gates. **Fixed revision:** none. This audit did not implement or validate the remedy.

## 5. Human decisions before Gate B

**D1, D3, D4 and D5 are the right four product/execution decision categories. As written, they are not a complete concrete authorization packet.** No separate new product decision is needed merely because routine fixes are required. The existing per-unit and per-transition approvals remain additional instances of existing authority, not a newly invented D6.

- **D1:** Correct R1/R4/R6/R7/R8 before asking for design acceptance. State the exact approval/plan/policy/import additions and migration behavior. Preserve §§5–7, 9, 12–17 and 20–21. The current wording does not authorize an unlisted edge or loosen independence. Clarify that complete §17 failure classification and human confirmation remain required; preapproved same-route scope is not agent self-classification.
- **D3:** Include Phase B verified-success/waiver/release work, or obtain an explicit reduction of the original acceptance criteria. A recommendation is not a decision. PARTIAL does not satisfy the original verified-closure objective.
- **D4:** Name the exact commit scope. The first code commit must contain only preserved work plus Gate B fixes that received coding review and the required tests. A Gate A design audit and green tests do not review all dirty code. Preserve v2’s first-clean-commit path before starting the controller; the current controller refuses a dirty baseline. Later loop output needs its own frozen review before Gate B completes. A docs-only Gate A commit can be separately scoped. Push still needs explicit approval.
- **D5:** Name the three exact external paths and the typed admission/reservation/result protocol. Include exact-route resolution, preflight, omission/timeout/cancellation behavior, escalate_to=stop, independent review and cross-project tests. The present narrow “field + call site” label is insufficient for the authority and provenance work. If implementation needs a ledger-format change rather than a host-owned wrapper, obtain that exact additional external-file scope first.

Before Gate B work, each unit still needs authentic approved plan/scope, route eligibility, permission and budget facts. Every core transition needs its own fresh challenge-bound human decision. D4 is not coding or dispatch authorization. D5 is not permission to edit arbitrary Obsidian files. This audit grants none of them.

## 6. Requirement coverage and remediation order

| Requirement | Result | Evidence |
|---|---|---|
| Q1: requested prior closure | FAIL: 4 closed, 7 partial | §1 and exact inherited criteria in AUDIT.json |
| Q2: additive binding compatibility | PASS in principle, qualified; not code proof | §2, legacy/worktree tests |
| Q3: authority, accounting, review and finding attacks | FAIL | §3; R1/R4/R6/R7/R8 |
| Q4: new wrong behavior/regression | FAIL | illegal renewal walkthrough; replay/envelope contradictions |
| Q5: complete pre-B decision packet | FAIL as written; four categories remain sufficient | §5 |

Recommended corrective sequence, subject to the human escalation below:
1. Define one legal issuance/renewal rule and bind effective policy in an approved document. Exit: R1/R6 design traces have explicit human-bound values and legal edges.
2. Define mandatory admission, launch consumption, identity namespace, replay/admission arithmetic and cancellation. Exit: R4/R7 schedules have explicit refusal or recovery behavior.
3. Bind exact review dispatch, frozen packet and captured report output. Exit: R8 substitution controls preserve F1 and reject self/stale reviews.
4. Update the proof matrix and D1/D3/D4/D5 packet. Obtain the bounded human decision, then re-audit any revised design before dependent Gate B code. Later gates must supply execution proof.

**Improvement opportunities:** None separate from the minimum corrective work. No speculative redesign is recommended.

## 7. Limitations and integrity

- Isolation is partial. This is the same auditor re-auditing v1/v2 findings, not a blind first pass. Prior auditor reports are explicit permitted inputs. V3 is the object of audit; no implementer chat/defense or controlled unblinding was supplied.

- This is Gate A design review. CLOSED (design level) does not mean code-fixed, integration-tested, human-approved or ready for Gate B. Existing code defects and the absence of a bridge are expected later-gate work, not alone reasons to fail this gate.

- Both repositories were read-only by procedure; they were not mounted read-only. No HMA/Obsidian file, Git state, worker, model API or network endpoint was mutated/invoked by the audit. All created files and caches are under the output directory.

- The HMA inventory preserves 104 tracked/nonignored files including dirty bytes. .serena/, Git internals and ignored files are excluded. Exact controller/skill/prior-report inputs are preserved separately. Supporting ledger/dispatch and audit-contract files were captured later and labeled supplemental.

- Only two new pure Go probes, thirteen selected preexisting Go test functions, three mocked controller checks, and two literal design traces ran. No full Go/controller suite or real end-to-end worker test ran. The controller suite contains agent-dispatch and Git-writing fixtures forbidden by this dispatch.

- Design attack sequences describe missing or contradictory rules in proposed code. They are not demonstrated runtime exploits in an HMA importer/attempt hook that does not yet exist. Trusted-host store compromise is outside the asserted protection.

- The declared auditor route is liz / OpenAI / gpt-6.1-sol / high / review. Access is the current managed Codex tool session. Actual runtime/provider configuration and prior coordinator hi preflight were not independently verified. No subagent dispatch was performed.

- Jev was absent from the exposed callable tool inventory. Auditor/orchestrator judgment was used as the specified fallback. No other model was represented as Jev.

- The initially attempted python executable was absent; successful scripts used python3. Some exploratory displays were truncated. Material findings use targeted untruncated reads, preserved bytes and logs. One exploratory ledger filename did not exist; rg located agent-run-ledger.

- The skill report validator forbids pass under partial isolation. Substantive unresolved high findings independently require FAIL; this verdict is not caused only by the isolation rule.

Phase 1 was frozen at `2026-10-15T14:29:40.493526+00:00` in `PHASE1.json`, SHA-256 `7b6241df0c38b961658b15605e23528a8cb451b08afd16fc982f7d7fb8d6ba59`. This is a re-audit assessment freeze, not a blind assessment. No controlled unblinding followed. The phase file is retained unchanged. Structural validation is recorded in `report-validation.log`; it validates form, not truth. End-of-audit target checks are recorded in `target-verification.json`. Raw commands, sources and full probe outputs remain in this directory.

## Exactly one next action (§18)

**ESCALATE_TO_HUMAN — Lennon reviews this failed Gate A authority-design packet. Keep Gate B stopped.**

**Current outcome:** Gate A FAIL; 4 requested findings CLOSED (design level), 7 PARTIAL; 3 newly numbered findings R6-R8 plus residual R1/R4. **Rationale:** The third audit still finds authority/provenance gaps. More unbounded design iteration or code without the required human decisions would not satisfy the gate.

**Owner/route:** Lennon; human decision. No worker dispatch is required for this action. Any later execution route needs its own eligibility and exact-route `hi`.

**Permission envelope:** Read/review the frozen audit packet. This report grants no coding, Git write, cross-project write, budget expansion, stage transition, push or release authority.

**Exact approval required:** No approval is granted here. Before dependent execution, Lennon must accept a corrected concrete D1 authority design, choose D3 scope, authorize the bounded D4 commit envelope and D5 external changes. Each Gate B plan unit/route/permission/budget and every stage transition still require the existing authentic bound authority. Routine fixes do not create an invented D6.

**Stop/escalation conditions:** Keep Gate B stopped while high design findings or required authority remain unresolved. Stop for unavailable eligible independent review, missing evidence, unknown authority or a safety conflict.
