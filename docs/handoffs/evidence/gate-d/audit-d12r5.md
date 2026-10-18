# HMA Gate D D1+D2 audit

**Verdict: FAIL.** The report-contract value is `fix-required`. Both labels mean the same decision.

Observed: full normal tests, vet and race tests pass in the isolated copy. The added waiver test fails in normal and race modes. Three bound-waiver cases keep a lease active after its issuer is invalidated. The unbound control passes.

Assessment: all 12 supplied prior-finding closure criteria are **FIXED**. One new high-severity finding, **D12R5-01**, blocks D1+D2 acceptance. D3-D6 remain outside this audit.

## Scope, provenance and isolation

- Dispatch: `hma-gate-d-audit-20261017-d12r5`.
- Target: `git:bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0 + diff:sha256:c119b7f95c3cff42c5a2f13037bbef88fefbde6093a55c355a1ccbe64dca19fb + FREEZE_START.json untracked manifest`.
- Branch: `main`. Start freeze: `2026-10-17T23:40:35.805544+00:00`. See [FREEZE_START.json](FREEZE_START.json).
- Start and end HEAD, branch, status, binary-diff digest and every untracked-file digest are equal. All 10 changed/new code and test copies match the target. See [FREEZE_VERIFICATION.json](FREEZE_VERIFICATION.json).
- Mode: code and outcome audit. Context isolation: **isolated**. Execution isolation: local filesystem copy; same user and kernel; no container or VM. Target read-only access was a task constraint, not an OS-enforced mount.
- The user declared the auditor route as liz / gpt-6.1-sol / high. OpenAI is inferred from the model label. Access was through this Codex host. The route was not independently attested. No onward agent dispatch or nested model call occurred.
- Authoritative sources: outcome consolidation §5 Gate D; Gate A v2 §2.2; v3 §2.1-2.3; v4 §1-2; architecture §§5, 6, 10, 18; AGENTS.md. Latest approved Gate A design wins. Gate A content was printed only in the requested section ranges.
- Requirements and failure hypotheses were saved in [REQUIREMENTS.json](REQUIREMENTS.json) before diff inspection. The verdict-free closure criteria were the only supplied prior-finding context.
- Phase 1 SHA-256: `bfdf47fffdf5f7944e030dd50087f47e5ca0bb39ee310f6709c4a1b970452e09`. See [PHASE1.json](PHASE1.json) and [PHASE1.sha256](PHASE1.sha256). The file was created exclusively, then hashed. It was not rewritten.
- The implementer handoff was read only after that hash was recorded. No prohibited prior-audit directory or `/tmp/claude-*` file was read. `.serena/` and the architecture-amendment files were excluded from substantive review. Their untracked-file hashes remain in the freeze inventory.

## Requirement coverage

MET means the scoped requirement has satisfactory evidence. PARTIAL means part of the requirement fails. NOT MET means its required behavior is absent. These are audit assessments, not HMA terminal outcomes.

| ID | Requirement | Assessment | Observation and evidence |
| --- | --- | --- | --- |
| D1-01 | Immutable versioned acceptance contract includes required intent, scope, non-goals, criteria/checks/threshold/evidence, risk, deliverables, budget, route and permission ceilings. Strict unambiguous decoding; material semantic change changes digest. Source: Gate A v2 §2.2; v3 §2.1 | **MET** | Strict byte parser rejects malformed Unicode and absent/null declarations. Valid scalar Unicode round-trips. Material changes change the canonical digest. Evidence: internal/model/contract.go:142-184,190-377,396-438; independent-final.log: TestAuditorRequiredDeclarations and TestAuditorUnicodeAndDigest; control-declarations.log |
| D1-02 | Core criterion IDs exactly match contract IDs; result dispositions remain outside contract. Source: Gate A v2 §2.2 | **MET** | Exact criterion-ID matching refuses renamed, missing, extra and duplicate sets. A disposition-only change preserves the ID condition. Evidence: internal/engine/contract_binding.go:60-149; internal/engine/lease.go:158-162; independent-final.log: TestAuditorLeaseBoundaries/criterion_* and disposition_only; TestAuditorApplyContractChoices |
| D1-03 | Optional acceptance_contract_digest preserves legacy approval serialization; initial presented contract cannot be dropped; all later approvals carry same digest; mismatch refuses. Contract replacement only after human rewind to acceptance criteria. Source: Gate A v3 §2.1; architecture §6 | **MET** | Real Apply accepts exact contract and both-absent legacy control. It refuses presented-unbound, wrong digest and changed document. Replacement succeeds after legal human rewind. Empty additive field preserves golden legacy bytes. Evidence: internal/model/types.go:193-197; internal/transition/apply.go:147-157; internal/engine/contract_binding.go:60-108; independent-final.log: TestAuditorApplyContractChoices, TestAuditorLegacyApprovalBytes, TestAuditorReplaceContractAfterRewind |
| D1-04 | hma.plan/v1 has every architecture §10 declaration, max_wall_minutes, cost ceiling, policy_digest, dependency validation and canonical accepted_plan_digest. Missing or ambiguous required declarations refuse. Source: Gate A v3 §2.2-2.3; v4 §2; architecture §10 | **MET** | Every nested document declaration is tested missing and null. Explicit zero retry and empty argv/non-goals succeed. Unknown/self/cyclic dependencies and contract conflicts refuse. Evidence: internal/model/contract.go:443-599; independent-final.log: TestAuditorRequiredDeclarations and TestAuditorContractCeilings; control-declarations.log |
| D2-01 | Only latest approved legal per-unit transition targeting Implementation authorization issues lease; no host record grants authority; no issuer or legacy plan refuses. Source: Gate A v4 §1 | **MET** | Only legal unit issuers qualify. No issuer, wrong unit and illegal-only issuer refuse. Legal Independent validation rework succeeds with retained route approval. Evidence: internal/engine/lease.go:92-108,223-232; independent-final.log: TestAuditorLeaseBoundaries/illegal_issuer_only, legal_rework, unit_mismatch; TestAuditorRewinds |
| D2-02 | Lease binds exact approved plan, unit, approval transition digest and contract. Stable digest uses documented identity fields. Source: Gate A v3 §2.2 | **MET** | Reference SHA-256 agrees with the specified transition_digest/unit_id/plan_digest envelope. Each identity field changes the digest; a changed record anchor does not. Changed plan/contract bytes refuse. Evidence: internal/engine/lease.go:110-151; independent-final.log: TestAuditorLeaseDigest, TestAuditorLeaseDigestAllFields, plan_mismatch, contract_mismatch |
| D2-03 | Lease only active at projected Implementation authorization and nonterminal run. Any later global/unit/dependency rewind voids old generation; transitive graph derives from approved plan; fresh legal issuer can resume. Source: Gate A v3 §2.2-2.3; architecture §5 | **MET** | Pure and real Apply controls void all old A generations after Grounding or Acceptance criteria returns, for empty/A/D tags. Real D release preserves A until the global rewind. Direct/transitive dependency corrections refuse; unrelated correction stays active. A fresh issuer resumes. Evidence: internal/engine/lease.go:154-189,246-262; independent-final.log: TestAuditorRewinds and TestAuditorRealApplyGlobalRewinds; control-rewind.log |
| D2-04 | Wall time is in [approval time, approval time + max_wall_minutes), not append time; counters nonnegative and strictly below ceilings. Source: Gate A v3 §2.2 | **MET** | Approval timestamp controls expiry despite late append. Exact expiry and pre-approval Now refuse. Negative counters and equality at either ceiling refuse. Below-ceiling controls pass. Evidence: internal/engine/lease.go:133-138,207-219; independent-final.log: TestAuditorLeaseBoundaries and TestAuditorRealApplyLateApproval |
| D2-05 | Current policy equals plan policy; contract risk/route/permission/budget ceilings constrain unit. Selected core route identity agrees with unit and exact approval, including legal rework. Source: Gate A v2 §2.2; v3 §2.2; v4 §1-2 | **MET** | Out-of-contract routes, permissions, budgets and criteria refuse even after rebinding accepted_plan_digest to the changed plan. Missing/mismatched route projections and changed policy refuse. All exact host route fields affect identity except the approval reference. Evidence: internal/model/contract.go:555-599; internal/engine/lease.go:194-220; internal/host/host.go:459-470; independent-final.log: TestAuditorContractCeilings, TestAuditorLeaseBoundaries, TestAuditorExactRouteIdentity; control-route.log |
| REG-01 | Preserve transition kernel, Gate B and Gate C behavior and legacy binding bytes; do not claim D3-D6 reservations, reconciliation, criterion credit, finding disposition or architecture amendment. Source: Original objective; architecture §5-6 | **PARTIAL** | All 12 packages pass full normal/race checks and vet. Golden legacy binding and Gate B/C suite pass. New pure lease authority does not preserve the existing bound-waiver invalidation rule. Post-freeze inspection confirms the handoff explicitly defers D3-D6. No later-part completion is claimed. Evidence: go-test-final.log; go-vet-final.log; go-race-final.log; waiver-reproduction.log; waiver-race.log |
| D2-06 | An approval invalidated by a change to a waiver it actually bound cannot continue to grant a lease. Regranting the waiver cannot restore that old approval. Source: docs/architecture.md §5.4 final paragraph and §6 final binding rule; AGENTS.md safety kernel; refinement of initial REG-01 | **NOT MET** | Real Apply issues an active lease whose approval binds a recorded waiver grant digest. Legal WITHDRAW and EXPIRE operations are evaluated and appended. EvaluateWaiverChange reports the issuer invalidated, while DeriveLease still returns ReasonNone. Withdrawal followed by regrant also keeps the old lease active. The unbound control stays active. Evidence: waiver_audit_test.go:13-113; waiver-reproduction.log; waiver-race.log; internal/engine/lease.go:154-220; internal/engine/invalidation.go:315-361 |

The explicit D2-06 row refines the original REG-01 safety-kernel requirement. It comes from architecture §5.4 and §6 already in the authoritative source set. It does not add a new product feature or deferred findings-disposition requirement.

## New finding D12R5-01

**High severity. Observed. Open.** A lease stays active after a bound waiver invalidates its issuing approval.

The reproduction follows these steps:

1. Create a temporary Git repository and a validated core chain. Record criterion `C`.
2. Evaluate and append a human `GRANT` for `C`. Keep the grant record digest.
3. Use real `transition.Apply` calls to accept the contract, select the route and authorize unit `A`. Each tested bound approval includes the grant digest in `ActiveWaivers`.
4. Confirm that `DeriveLease` returns an active A lease.
5. Evaluate and append a legal `WITHDRAW` or `EXPIRE` for the same scope. Confirm that the resolution projection no longer has an active waiver.
6. Call `EvaluateWaiverChange` with the bound grant digest. It reports the issuing approval as invalidated.
7. Call `DeriveLease` against the current validated chain. It still returns `ReasonNone` and an active lease. Withdrawal followed by a fresh grant also keeps the old lease active.

The control uses an issuer that did not bind the grant. It stays active after withdrawal. This confirms the required scoped behavior.

Evidence: [waiver_audit_test.go](waiver_audit_test.go), [normal output](waiver-reproduction.log), [race output](waiver-race.log), and [overlay map](waiver-overlay.json). Three bound cases fail in each mode. The unbound control passes in each mode.

Cause: `internal/engine/lease.go:163-189` skips every record except approved stage transitions during invalidation replay. Its later checks cover route, time, counters and policy. They do not apply the bound-waiver invalidation rule in `internal/engine/invalidation.go:340-342`. The `ActiveWaivers` set on the selected issuer is never checked.

Requirement: architecture §5.4 says a waiver operation invalidates approvals bound to the affected criterion, finding or artifact. Section 6 says a change to an actually bound field invalidates the approval. A lease cannot retain authority from that invalidated approval. The host in this reproduction is truthful. No core record is forged.

Impact: this is a demonstrated invalid lease in the new pure interface. No production command consumes it in this candidate. The finding blocks accepting D2 for later consumption. It does not claim that a real worker was launched.

Proposed preferred fix: In pure lease derivation, replay waiver applicability changes from the validated core chain against the selected issuer's bound waiver-operation digests. Resolve each bound grant to its canonical scope. Refuse the issuer after a later withdrawal, expiration, replacement or other applicable change. Preserve that invalidated-generation fact even if the same scope is granted again. Share the deterministic projection/predicate with EvaluateWaiverChange rather than relying on a host-supplied active flag. Add a distinct refusal reason.

Why: Architecture §5.4 invalidates approvals that bind changed waivers. A lease derives authority from the approval and cannot outlive that invalidation. Current stage alone cannot represent scoped packet invalidation.

Verification:

- Run the preserved waiver-overlay reproduction in normal and race modes. Expected: All four subtests pass: bound withdrawal, expiration and withdrawal/regrant refuse; the unbound control remains active.
- Add separate controls for unrelated scope, a new legal issuer bound to the current waiver generation, multiple grants, and missing/malformed references to actually bound grants. Expected: Unrelated changes preserve the lease. Only the fresh valid issuer can resume. Ambiguous bound provenance fails closed; no older generation revives.
- Rerun full test, vet, race, route/rewind/legacy-byte controls and obtain a fresh independent audit. Expected: All regressions pass and the new invalidation test passes without relaxing the original contract.

Trade-offs: Mapping a bound operation digest to canonical scope requires precise chain provenance. Overbroad invalidation can stop unrelated units. Rejecting every waiver-bearing run is safe containment but restricts previously permitted work. No stage transition or architecture amendment is needed for this proposed fix.

Alternative: As temporary containment, refuse all lease derivation for issuers with a nonempty ActiveWaivers set. This is small and fails closed but sacrifices scoped availability. A durable alternative is a shared immutable approval-validity projection used by both lease and waiver engines; it is a larger integration change.

Effort: **medium**. Priority: contain before lease use, then implement and independently retest. Fix status: **proposed**. No audited code was edited and no remedy was validated.

## Prior-finding closure decisions

| ID | Status | Independent closure evidence |
| --- | --- | --- |
| F01 | **FIXED** | TestAuditorContractCeilings; TestAuditorLeaseBoundaries/profile_identity and permission_broadening; TestAuditorExactRouteIdentity |
| F02 | **FIXED** | TestAuditorRewinds (direct/transitive dependency corrections, route and plan returns, unrelated correction and fresh issuer) |
| F03 | **FIXED** | TestAuditorRealApplyLateApproval; TestAuditorLeaseBoundaries/nanosecond_before_expiry, exact_expiry, before_approval |
| F04 | **FIXED** | TestAuditorUnicodeAndDigest |
| F05 | **FIXED** | TestAuditorRequiredDeclarations; TestAuditorContractCeilings |
| F07 | **FIXED** | TestAuditorLeaseBoundaries/criterion_rename, criterion_missing, criterion_extra, criterion_duplicate, disposition_only |
| F06 | **FIXED** | TestAuditorApplyContractChoices |
| F08 | **FIXED** | TestAuditorLeaseDigest; TestAuditorLeaseDigestAllFields |
| F09 | **FIXED** | TestAuditorLeaseBoundaries/negative_attempts, negative_cost, below_ceilings, attempt_ceiling, cost_ceiling |
| D12R2-01 | **FIXED** | TestAuditorRequiredDeclarations; TestAuditorLegacyApprovalBytes; go-test-final.log; go-vet-final.log; go-race-final.log |
| F-D12R3-01 | **FIXED** | TestAuditorRewinds; TestAuditorRealApplyGlobalRewinds (including active A while D reaches release, tagged global rewinds, continued progress, fresh A issuer); go-race-final.log |
| D12R4-01 | **FIXED** | TestAuditorLeaseBoundaries; TestAuditorExactRouteIdentity; TestAuditorRealApplyGlobalRewinds; go-test-final.log; go-vet-final.log; go-race-final.log |

All named auditor tests above are in [independent-final.log](independent-final.log), except the new waiver reproduction. These closure decisions apply only to the supplied criteria. They do not resolve D12R5-01.

## Commands and raw evidence

Run from `/tmp/hma-gate-d-20261017-d12r5/worktree`. Prefix each Go command with:

```sh
export PATH="$HOME/.local/go/bin:$PATH"
export GOCACHE=/tmp/hma-gate-d-20261017-d12r5/gocache
export TMPDIR=/tmp/hma-gate-d-20261017-d12r5/testtmp
```

Environment: Go `go1.27.1 linux/amd64`; module language version `1.21`; Git `/usr/bin/git`. Test-created repositories live under the output temporary root. No Git write targeted `/home/yeli/repos/HMA`.

| Check | Exact command | Exit | Evidence |
| --- | --- | --- | --- |
| T01 | `go test ./...` | 0 | [go-test-final.log](go-test-final.log). 12 packages pass; some unchanged packages cached. |
| T02 | `go vet ./...` | 0 | [go-vet-final.log](go-vet-final.log). No diagnostics. |
| T03 | `go test -race ./...` | 0 | [go-race-final.log](go-race-final.log). 12 packages pass, including auditor normal controls and real global rewind tests. |
| T04 | `go test -v ./internal/engine ./internal/transition ./internal/host -run '^TestAuditor'` | 0 | [independent-final.log](independent-final.log). 13 top-level auditor tests; 414 PASS entries including top-level and subtests. This is a test-entry count, not a count of independent requirements. |
| T05 | `go test -v -overlay=/tmp/hma-gate-d-20261017-d12r5/waiver-overlay.json ./internal/transition -run '^TestAuditorBoundWaiver'` | 1 | [waiver-reproduction.log](waiver-reproduction.log). Three bound-waiver cases fail; unbound control passes. |
| T06 | `go test -race -v -overlay=/tmp/hma-gate-d-20261017-d12r5/waiver-overlay.json ./internal/transition -run '^TestAuditorBoundWaiver'` | 1 | [waiver-race.log](waiver-race.log). Same three bound-waiver failures under race; unbound control passes. |
| T07 | `go test -overlay=/tmp/hma-gate-d-20261017-d12r5/controls/route.json -run '^TestAuditorLeaseBoundaries/profile_identity$' -v ./internal/engine` | 1 | [control-route.log](control-route.log). Expected mutation-control failure: removed route identity comparison admits wrong route. |
| T08 | `go test -overlay=/tmp/hma-gate-d-20261017-d12r5/controls/rewind.json -run '^TestAuditorRewinds/DGROUNDING$' -v ./internal/engine` | 1 | [control-rewind.log](control-rewind.log). Expected mutation-control failure: narrowed global rewind admits old A lease. |
| T09 | `go test -overlay=/tmp/hma-gate-d-20261017-d12r5/controls/declarations.json -run '^TestAuditorRequiredDeclarations' -v ./internal/engine` | 1 | [control-declarations.log](control-declarations.log). Expected mutation-control failures: missing/null declarations become accepted. |

T07-T09 are sensitivity controls. Their overlays remove one safety check from the copied source. Their failures are expected. They do not modify the audited source. T05-T06 are different: they expose the unmodified implementation defect.

The independent passing set contains 13 top-level tests and 414 PASS entries including subtests. This count is not proof of coverage. The requirement table states what was tested. Full-suite logs cover 12 packages, including existing Gate B and Gate C tests. Some unchanged packages are cached in the final normal run. The race run includes the real global rewind tests.

The first full normal and race runs failed the scratch-directory test because the copied repository was under the then-default `/tmp` temporary root. The corrected environment sets `TMPDIR` to `testtmp`, which separates the source copy from scratch stores. The corrected and final runs pass. An early auditor-only route test used a nonexistent capability constant. That test was corrected before final checks. The failed logs are retained. Neither failure is attributed to the implementation.

## Controlled unblinding addendum

Phase 1 was frozen at SHA-256 `bfdf47fffdf5f7944e030dd50087f47e5ca0bb39ee310f6709c4a1b970452e09` before reading `docs/handoffs/2026-10-17-gate-d-d1-d2.md`. The handoff digest is `37494c978e00f9059b44c002415653a1eafd9a88e3c629ac9fdb1c8650f9f1b3`.

- The handoff explicitly defers D3 reservations, D4 launch reconciliation, D5 criterion credit/findings disposition/full packet, and D6 architecture amendment. It states that Gate D as a whole is not passed until all parts pass.
- The handoff states that DeriveLease is not wired to a command yet and D3 will consume it. Static call-site inspection agrees: no production caller of DeriveLease is added in this candidate. The new defect is therefore demonstrated in the pure interface, not as a completed real worker-launch exploit.
- The handoff does not discuss bound-waiver invalidation. Its completeness claims do not refute the executable failure. No Phase 1 finding, prior closure decision or verdict changed.

## Limits and unresolved assumptions

- Context isolation is isolated: fresh audit inputs contained original requirements, neutral target metadata and verdict-free closure criteria. No implementer summary or prior audit result was read before Phase 1. Source code, tests and comments were audited artifacts; they necessarily contain author explanations.
- Execution isolation is a local filesystem copy, not a VM/container or a separate trust domain. Audited repository access was logically read-only, not OS-enforced read-only. Tests ran under the same user and kernel, with caches and temporary repositories under this output directory. No Git writes targeted the audited repository.
- The target is a dirty working tree over bc5acb7, not an immutable committed candidate. Start/end hashes detect changed endpoints, not transient changes between reads. Trusted-host inputs (current chain, route projection, policy digest and counters) are assumed truthful; D3 persistence/counter derivation and launch enforcement are outside scope.
- No D3-D6 acceptance is granted: reservations, launch reconciliation, criterion credit, findings disposition and the architecture amendment remain deferred. Per-coding-change mandatory review and later Gate E/F outcomes are not certified.
- Jev was not available in the host tool inventory. The auditor made typed requirement and evidence judgments directly. No onward worker dispatch, nested agent CLI or model API was used.
- Initial normal and race full-suite runs failed TestSyntheticStoreRefusesNonScratchDir because the copied repository lived under /tmp, the then-default temporary root. Setting TMPDIR to output/testtmp restored the test's required separation. Final normal/vet/race runs passed.
- An early auditor-only host identity test referenced a nonexistent capability constant. It was corrected to the existing repository_scanning constant before final checks. Initial failed logs are retained. This was not an audited implementation defect.
- Canonical digest means SHA-256 over the typed Go JSON serialization. Whitespace, object-key ordering and equivalent Unicode escaping may normalize to the same digest by design. This audit tests material semantic changes, not raw-byte uniqueness or cryptographic collision resistance.
- The new failing waiver reproduction uses real Apply, EvaluateResolution and Store.Append interfaces with synthetic local human-confirmation fixtures. It does not launch a worker or claim a production exploit/dispatch was executed. Proposed remedies have not been implemented or validated.

No material optional redesign is recommended. The smallest useful change is to enforce the existing bound-approval invalidation rule in the pure lease path.

## Remediation sequence

1. Keep the new lease interface from being consumed as accepted D2 authority while D12R5-01 remains open. Exit condition: Coordinator records the blocker; no acceptance or further lease-use claim relies on this audit as PASS.
2. Implement the minimal pure bound-waiver invalidation projection or explicitly bounded fail-closed containment. Exit condition: Preserved four-case reproduction passes; old generations stay void after regrant; new valid issuer can resume.
3. Run the full regression commands and request a fresh independent audit of the changed bytes. Exit condition: No high/critical finding remains; full required checks and new negative controls pass.

## Required next action — architecture §18

**Current outcome:** FAIL. D1+D2 acceptance remains refused. Evidence: one high finding with normal and race reproductions. All prior closure criteria are fixed.

**Requested transition:** NONE. This audit does not advance a stage.

**Recommendation:** Return D12R5-01 to the Gate D implementer for a minimal bound-waiver invalidation fix and a new independent audit.

**Rationale:** A lease currently keeps authority after an approval input changes. The failed negative control blocks D1+D2 acceptance despite 12 prior closures.

**Owner:** Gate D coordinator / Ming (Opus implementer, as identified by the post-freeze handoff)

**Proposed route:** Proposed prior implementer route: Ming / Anthropic / claude-opus-5-5 / effort not provided / Claude Code CLI, from the post-freeze handoff. Eligibility and exact-route hi preflight must be checked before dispatch. No route is authorized by this audit.

**Permission envelope:** Proposed next work: bounded code/test changes for pure approval validity and lease derivation. This audit itself remains read-only on the target. No commit, push, release, launch, permission expansion or stage advancement is authorized.

**Exact approval required:** No stage decision is requested or granted by this audit. Remediation must stay within existing human-approved Gate D scope; any later stage transition requires its own fresh exact-input challenge-bound human approval. No approval challenge was generated in this audit.

**Stop and escalation conditions:** Keep Gate D D1+D2 closed while D12R5-01 remains open. Escalate if remediation requires a product-contract change, broader permission or an unavailable worker route. Stop for any failing regression or remaining invalidated-issuer lease.

**Single next action:** Return D12R5-01 to the Gate D implementer for remediation and a fresh independent audit.
