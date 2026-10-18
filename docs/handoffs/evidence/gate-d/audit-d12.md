# HMA Gate D D1+D2 audit

**Verdict: FAIL.** Independent tests reproduced invalid active-lease results and an unbound acceptance transition. The report has 6 high, 2 medium and 1 low finding. No repair was made.

The baseline passes `go test ./...`, `go vet ./...` and `go test -race ./...`. The corrected broad test and race runs each pass all 12 packages. These results do not override the independent failing acceptance tests.

The D2 primitive is not wired to a command. This audit demonstrates incorrect derivation results. It does not demonstrate a current host launch bypass. The D1 omission is reproduced through the transition write API.

**Next decision:** return D1/D2 to its implementation owner for correction. The complete next-action contract appears at the end of this report.

## Scope and provenance

- Dispatch: `hma-gate-d-audit-20261017-d12`. Modes: code and outcome.
- Target: branch `main`, HEAD `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`, plus the frozen dirty tree.
- Tracked diff SHA-256: `35d1b1a658336f8774e392181095af456eb646b37f4d4c6228234147afd4de2f`.
- All untracked files were hashed. `.serena/` and the two `2026-10-16-arch-amendment-*` files were not evaluated. The D1/D2 handoff was hash-only before the blind freeze.
- Blind Phase 1 SHA-256: `6bd56fecbbaf1f5a722ea63bc9225df70d64cbb27228e22877b0556e9899b98a`. [Frozen JSON](PHASE1.json), [hash](PHASE1.sha256).
- Isolation: **isolated audit context**. The host filesystem is shared. Tests ran in a copied tree. Repository read-only behavior was procedural, not an OS-enforced read-only mount.
- Declared auditor route: liz / OpenAI / gpt-6.1-sol / high / current Codex host. The exact access-service identifier was not supplied. Runtime model and effort were not independently attested. No worker was dispatched. No nested agent CLI or model API was used.
- Environment: `go1.27.1 linux/amd64`; `go.mod` declares Go 1.21. Git path: `/usr/bin/git`. GOCACHE and both corrected temporary roots are under this output directory.
- Jev: no approved host tool was exposed. The auditor used the documented fallback judgment path.

The authoritative requirements were derived before reading the diff. See [requirements and hypotheses](requirements-derived.json), [failure hypotheses](failure-hypotheses.txt), and [source sections](requirements-sources.txt). The objective is Gate D in the original outcome-consolidation handoff. The design precedence is v2 §2.2, v3 §2.1–§2.3, then v4 §1–§2. Architecture §§5, 6, 10 and 18 and AGENTS.md remain normative.

## Requirement coverage

8 MET; 10 PARTIAL; 0 NOT MET. PARTIAL means some required behavior works and a material part fails. It is encoded as `fail` in the machine report contract. No row is an essential untested unknown.

| ID | Requirement | Assessment | Observation and evidence |
| --- | --- | --- | --- |
| D1-01 | Versioned immutable canonical acceptance contract contains run, intent, scope, non-goals, criteria/check/threshold/evidence, risk, deliverables, budget/route/permission ceilings; excludes result dispositions. Source: v2 §2.2; v3 §2.1. | **MET** | Schema represents combined intent/scope text, non-goals, criterion definitions and ceiling fields. Results remain in core criteria. Empty lists may be meaningful; mandatory scalar/plan completeness defects are assessed separately. `internal/model/contract.go:20`, `internal/model/contract.go:39` |
| D1-02 | Every material contract byte/semantic change changes digest; parse rejects ambiguous, invalid or noncanonical input. Source: v2 §2.2 canonical JSON; audit integrity focus. | **PARTIAL** | Known fields are hashed after decoding; duplicate/case/unknown/trailing keys are refused. Invalid UTF-8 and lone UTF-16 surrogates are accepted and normalized to the same replacement character/digest. `independent-model.log`, `independent-controls.log`, `F04` |
| D1-03 | Core criterion ID set exactly equals contract criterion ID set. Source: v2 §2.2. | **PARTIAL** | Acceptance and later transitions check exact criterion ID sets. Lease derivation does not recheck the current core criterion set. `internal/engine/contract_binding.go:75`, `independent-transition-corrected.log`, `F07` |
| D1-04 | Optional acceptance_contract_digest omitted empty; legacy serialization/digests unchanged. Source: v3 §2.1. | **MET** | Independent exact-byte legacy serialization control passes. Nonempty new binding appears in JSON. Existing chain tests pass. `independent-controls.log`, `go-test-corrected.log` |
| D1-05 | Acceptance-to-planning approval binds contract; every later approval carries it; mismatch/drop/swap refused; contract revision requires human rewind to acceptance. Source: v2 §2.2; v3 §2.1; architecture §6. | **PARTIAL** | Existing tests refuse threshold changes, swapped/dropped bound digests and missing documents. Initial acceptance silently ignores a supplied document when its digest is omitted. `internal/transition/contract_test.go:101`, `independent-transition-corrected.log`, `F06` |
| D1-06 | hma.plan/v1 canonical plan records all architecture §10 unit fields, wall minutes, declared cost ceiling, policy digest; digest is accepted_plan_digest. Source: v3 §2.2; v4 §2; architecture §10. | **PARTIAL** | Version, wall-time, cost and policy digest fields exist and accepted_plan_digest is checked. Complexity budget is absent; many required section 10 fields need not be populated or present. `internal/model/contract.go:65`, `internal/model/contract.go:297`, `F05` |
| D1-07 | Plan validates IDs, criterion references, dependency graph/prior-unit constraints, ceilings and required fields; malformed or unavailable plan fails closed. Source: v3 §2.3; architecture §10. | **PARTIAL** | Unique IDs, unknown/self dependency refusal, acyclic graph, criterion references and three ceiling comparisons exist. Missing plan fields still permit a lease. No stricter array-order rule for the term prior units was inferred. `internal/model/contract_test.go:124`, `independent-model.log`, `independent-engine.log`, `F05` |
| D2-01 | Lease derives only from latest applicable approved unit transition to IMPLEMENTATION_AUTHORIZATION on either legal issuer edge. No host cache grants authority. Source: v4 §1. | **MET** | Independent controls refuse no issuer, unapproved record, illegal issuer edge, different approval unit and missing plan/unit. Legal correction selects the new issuer. Input is assumed to be the verified core chain, not a forged slice. `independent-controls.log`, `independent-engine.log` |
| D2-02 | Presented plan digest equals issuer accepted_plan_digest; unit exists; contract digest/run/identity binding valid. Source: v2 §2.2; v3 §2.2; architecture §6. | **PARTIAL** | Plan/run/contract digest mismatches fail closed under independent controls. Current core criterion IDs are not checked during lease derivation. `independent-controls.log`, `independent-transition-corrected.log`, `F07` |
| D2-03 | Terminal run or projected unit stage outside IMPLEMENTATION_AUTHORIZATION cannot keep lease. Source: v3 §2.2; v4 §1. | **MET** | Independent unit review and terminal controls refuse the lease. Whole-run rewind also refuses. This is per-unit replay, not a demand that another unit cannot move the global run stage. `independent-engine.log` |
| D2-04 | Later approved rewind of unit, run or transitive dependency invalidates lease, using approved plan graph. Source: v3 §2.3; v4 §1. | **PARTIAL** | Route/Planning and transitive dependency rewinds invalidate leases. Independent-validation correction of a dependency to implementation authorization leaves dependent leases active. `independent-engine.log`, `F02` |
| D2-05 | Elapsed approval wall time strictly below max_wall_minutes; future/invalid timestamps fail closed. Source: v3 §2.2; audit time focus. | **PARTIAL** | Exact expiry boundary is refused. Expiry is computed from record time, not approval time; a real Apply chain grants an already-expired approval a new wall interval. Future-time requests are accepted. `independent-engine.log`, `independent-transition-corrected.log`, `F03` |
| D2-06 | All applicable per-unit counters below ceilings; lease grant within contract ceilings and route/permission identity; conflicts fail closed. Source: v2 §2.2; v3 §2.2; architecture §10. | **PARTIAL** | Attempts and cost compare at equality, and unit ceilings cannot exceed contract numeric ceilings. Contract permitted routes and permission envelope do not constrain the derived grant. Negative counters are accepted; D3-derived accounting is out of scope. `independent-controls.log`, `independent-engine.log`, `F01`, `F09` |
| D2-07 | Current computed policy digest equals approved plan policy_digest; drift voids lease and escalates to human. Source: v4 §2. | **MET** | Independent policy-drift control refuses mismatched policy digest. The trusted host supplies the current computed digest; recomputing it inside this pure function is not required. Host escalation wiring is outside D1/D2. `independent-engine.log`, `internal/engine/lease.go:166` |
| D2-08 | Lease digest deterministically binds issuer transition digest, unit and plan digest. Source: v3 §2.2. | **PARTIAL** | Digest is deterministic and bound to authenticated record anchor, unit and plan. Design names the approval transition digest; code uses record HeadAnchor. This is a protocol interpretation discrepancy, not demonstrated authority forgery. `independent-controls.log`, `independent-digest.log`, `F08` |
| D2-09 | Independent validation correction approval can issue fresh lease; illegal review-to-authorization remains refused; later lease uses carried counters supplied by trusted host. Source: v3 §2.2; v4 §1. | **MET** | Independent correction test reissues on the legal edge and refuses carried counters at the attempt ceiling. Existing legal-edge controls refuse implementation-review to authorization. Counter persistence is deferred to D3. `independent-engine.log`, `internal/engine/lease_test.go:225`, `go-test-corrected.log` |
| REG-01 | Existing transition human approval, exact binding, legal edges, Gate B and Gate C behavior preserved. Source: architecture §5-§6; original objective. | **MET** | All 12 packages pass corrected baseline tests and race tests. Vet passes. This is regression evidence, not proof of every possible Gate B/C behavior. `go-test-corrected.log`, `go-race-corrected.log`, `go-vet.log` |
| SCOPE-01 | D1/D2 expose pure derivation only; do not claim reservations, reconciliation, credit, disposition, or architecture amendment completion. Source: audit scope. | **MET** | Blind code inspection finds pure derivation, with reservation-derived counters explicitly deferred to D3. No launch, credit, disposition or amendment implementation is claimed by these code changes. Handoff narrative remains unread until after Phase 1. After the frozen blind pass, the handoff explicitly confirmed D3-D6 deferral and no command wiring. `internal/engine/lease.go:29`, `start-status.txt`, `unblinding.json` |

## Verification evidence

| Check | Exit | Evidence |
| --- | --- | --- |
| `/home/yeli/.local/go/bin/go test ./...` | 1 | [go-test.log](go-test.log); command metadata: [go-test.json](go-test.json) |
| `/home/yeli/.local/go/bin/go vet ./...` | 0 | [go-vet.log](go-vet.log); command metadata: [go-vet.json](go-vet.json) |
| `/home/yeli/.local/go/bin/go test -race ./...` | 1 | [go-race.log](go-race.log); command metadata: [go-race.json](go-race.json) |
| `/home/yeli/.local/go/bin/go test -overlay /tmp/hma-gate-d-20261017-d12/baseline-overlay.json ./...` | 0 | [go-test-corrected.log](go-test-corrected.log); command metadata: [go-test-corrected.json](go-test-corrected.json) |
| `/home/yeli/.local/go/bin/go test -race -overlay /tmp/hma-gate-d-20261017-d12/baseline-overlay.json ./...` | 0 | [go-race-corrected.log](go-race-corrected.log); command metadata: [go-race-corrected.json](go-race-corrected.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/engine -run ^TestAuditorD12 -count=1` | 1 | [independent-engine.log](independent-engine.log); command metadata: [independent-engine.json](independent-engine.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/model -run ^TestAuditorD12 -count=1` | 1 | [independent-model.log](independent-model.log); command metadata: [independent-model.json](independent-model.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/transition -run ^TestAuditorD12 -count=1` | 1 | [independent-transition.log](independent-transition.log); command metadata: [independent-transition.json](independent-transition.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/transition -run ^TestAuditorD12 -count=1` | 1 | [independent-transition-corrected.log](independent-transition-corrected.log); command metadata: [independent-transition-corrected.json](independent-transition-corrected.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/engine ./internal/model -run ^TestAuditorD12(IndependentBindings|LeaseDigestBinding|StrictJSONControls|LegacyApprovalSerialization)$ -count=1` | 0 | [independent-controls.log](independent-controls.log); command metadata: [independent-controls.json](independent-controls.json) |
| `/home/yeli/.local/go/bin/go test -v ./internal/engine -run ^TestAuditorD12SpecifiedLeaseDigest$ -count=1` | 1 | [independent-digest.log](independent-digest.log); command metadata: [independent-digest.json](independent-digest.json) |
| `/home/yeli/.local/go/bin/go test -json ./internal/model ./internal/engine ./internal/transition -run ^TestAuditorD12 -count=1` | 1 | [independent-all.log](independent-all.log); command metadata: [independent-all.json](independent-all.json) |

The final auditor suite has 47 leaf cases: 31 pass and 16 fail. Each failure is preserved in [structured test events](independent-all.log). These failures include the qualified F08 protocol interpretation and F09 invalid-input check. They are not 16 separate high-severity findings.

The initial broad test and race runs failed because the auditor set GOTMPDIR and TMPDIR to different roots. The dogfood tests require fixture stores to be under `os.TempDir()`. After those roots were aligned, both broad runs passed. Original logs are retained. This was an audit-harness failure, not a repository regression.

The baseline overlay removes only the three auditor-added test files. It does not change implementation files. [Overlay](baseline-overlay.json), [runner](run-check.py), [source manifest](snapshot-manifest.json), and [snapshot check](snapshot-check.json) are retained. Some tool displays were truncated; the underlying logs were not truncated.

Run the retained runner with a new log name to reproduce. It sets cache and temporary paths to this output directory. Its working directory is the copied snapshot:

```sh
python3 /tmp/hma-gate-d-20261017-d12/run-check.py rerun-auditor /home/yeli/.local/go/bin/go test -v ./internal/model ./internal/engine ./internal/transition -run ^TestAuditorD12 -count=1
python3 /tmp/hma-gate-d-20261017-d12/run-check.py rerun-baseline /home/yeli/.local/go/bin/go test -overlay /tmp/hma-gate-d-20261017-d12/baseline-overlay.json ./...
python3 /tmp/hma-gate-d-20261017-d12/run-check.py rerun-race /home/yeli/.local/go/bin/go test -race -overlay /tmp/hma-gate-d-20261017-d12/baseline-overlay.json ./...
python3 /tmp/hma-gate-d-20261017-d12/run-check.py rerun-vet /home/yeli/.local/go/bin/go vet -overlay /tmp/hma-gate-d-20261017-d12/baseline-overlay.json ./...
```

All source locators below refer to the frozen repository bytes. Those files are also available under `snapshot/`. Auditor tests are [engine](snapshot/internal/engine/auditor_d12_test.go), [model](snapshot/internal/model/auditor_d12_test.go), and [transition](snapshot/internal/transition/auditor_d12_test.go).

## Findings and proposed remedies

Every remedy below is proposed. No remedy has been implemented or validated.

### F01 — Contract route and permission ceilings do not constrain leases

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: medium.

Observed diagnosis: ValidatePlanAgainstContract checks contract/run/policy identity, criterion references, attempts, wall time and cost. It never consults PermittedRoutes or PermissionEnvelope. DeriveLease copies ImplementerRoute from the plan and has no approved route/permission projection to compare. A human-approved plan with an implementer route outside the separately accepted contract returns ReasonNone.

Evidence: `internal/model/contract.go:366` — No route or permission ceiling comparison exists in this validator.

Evidence: `internal/engine/lease.go:130` — Returned route comes directly from unit.ImplementerRoute.

Evidence: `independent-engine.log:TestAuditorD12RouteCeiling` — Contract permits route-approved/route-escalate/route-validator; plan names unapproved-provider-route; derivation returns active lease.

Qualification: This is a conflicting human-approved input, not a forged approval. The test proves the route-ceiling failure; the permission failure is the inspected absence of a comparison, not a separately executed launch.

Preferred fix, proposed: Check every applicable plan route against the contract route ceiling. Bind or present the existing approved route projection to derivation and verify its route identity and permission classes against the plan and contract; fail closed when unavailable or conflicting.

Reason: A later route approval must not broaden an immutable contract ceiling. A digest comparison alone does not establish route or permission compatibility.

Independent acceptance check:

- Run TestAuditorD12RouteCeiling with the out-of-contract route. Expected: Derivation refuses; an in-ceiling route still succeeds.
- Present a route with bounded_write against a read_only contract and vary approved route identity while preserving the plan. Expected: Both permission broadening and identity mismatch refuse before a lease becomes active.

Trade-offs and limits: Requires an explicit projection/interface for existing route approval data. Preserve legacy runs without contracts and existing route-coherence checks; do not invent an authorization field.

Alternative: Validate route/permission compatibility at plan approval and persist a kernel-validated projection that derivation can verify. This moves validation earlier but still needs replayable evidence.

### F02 — Dependency correction does not invalidate dependent leases

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: medium.

Observed diagnosis: The dependency branch only invalidates targets before IMPLEMENTATION_AUTHORIZATION. The legal INDEPENDENT_VALIDATION -> IMPLEMENTATION_AUTHORIZATION correction is a rewind of that dependency, but does not satisfy isEarlierGate. With C depending on B depending on A, a later correction approval for A leaves C active.

Evidence: `internal/engine/lease.go:154` — Dependency invalidation tests isEarlierGate only.

Evidence: `internal/engine/lease.go:174` — Implementation authorization is absent from the earlier-gate predicate.

Evidence: `independent-engine.log:TestAuditorD12LeaseControls/dependency_correction` — ReasonNone after A correction; route rewind of A correctly refuses C; unrelated Z correction keeps C active.

Preferred fix, proposed: Treat an approved dependency correction from independent validation to implementation authorization as invalidating each direct and transitive dependent lease issued before that correction. Keep new issuance for the corrected unit separate from dependent invalidation.

Reason: v3 section 2.3 voids leases of every transitive dependent when a unit rewinds; v4 section 1 explicitly makes this correction legal.

Independent acceptance check:

- Run dependency_correction, dependency_route_rewind and unrelated_correction controls. Expected: Direct and transitive dependents refuse; unrelated unit stays active.
- Approve a new legal issuing transition for an invalidated dependent. Expected: Only that new generation can resume; old generation never revives.

Trade-offs and limits: Must distinguish correction from ordinary forward transition. Do not globally invalidate unrelated units.

Alternative: Use a shared approved-transition invalidation predicate across lease replay and the existing route replay path, with dependency closure supplied by the accepted plan.

### F03 — Lease wall time starts at record append instead of human approval

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: small.

Observed diagnosis: DeriveLease parses ir.Metadata.Timestamp instead of Approval.Timestamp. Apply writes Metadata.Timestamp at append time. A still-admissible two-minute-old approval for a one-minute plan therefore obtains a fresh minute. The lower wall-time bound is also absent, so Now before issuance succeeds.

Evidence: `internal/engine/lease.go:122` — IssuedAt uses record timestamp; only Now.Before(until) is checked.

Evidence: `internal/transition/apply.go:171` — Append timestamp is time.Now(), independent of the approved timestamp.

Evidence: `independent-transition-corrected.log:TestAuditorD12ApplyLateApprovalExtendsLease` — Approval 2026-10-17T22:39:31Z; record 22:41:31.877466698Z; request 22:41:32Z active until 22:42:31.877466698Z.

Evidence: `independent-engine.log:TestAuditorD12LeaseControls/future_approval` — Request before issuer time returns ReasonNone.

Qualification: Future Now is an invalid trusted-host input or clock rollback case; the delayed-append defect is reproduced through the actual transition API with admissible approval inputs.

Preferred fix, proposed: Use the issuing approval timestamp for IssuedAt and expiry. Reject invalid/future approval time, reject Now before issuance, and validate the max-wall conversion before constructing time.Duration.

Reason: The design bounds elapsed wall time since approval. Append latency must consume, rather than renew, that bound.

Independent acceptance check:

- Run the real Apply late-approval test with a one-minute plan and two-minute-old approval. Expected: Transition may remain otherwise legal, but derivation refuses expired lease.
- Test one nanosecond before expiry, exact expiry, and Now before approval. Expected: Only the request inside [approval, expiry) succeeds.

Trade-offs and limits: Pending approvals can consume their entire lease before append, as the approved design requires. Duration overflow validation may reject formerly accepted extreme budgets.

Alternative: If append time is desired instead, obtain a design change and human approval for that changed timing contract; do not silently reinterpret the existing approval.

### F04 — Unicode decoding loses document content before hashing

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: medium.

Observed diagnosis: The new decoder checks keys and unknown fields but not UTF-8 or lone-surrogate integrity. Go replaces invalid UTF-8 and unpaired Unicode escapes with U+FFFD. Documents with 0xff, 0xfe, \ud800 and \ud801 in intent all parse to the same value and digest. Their original bytes cannot be recovered from the accepted canonical representation.

Evidence: `internal/model/contract.go:171` — decodeCanonicalInput uses encoding/json after key checking without an encoding-integrity check.

Evidence: `independent-model.log:TestAuditorD12UnicodeIntegrity` — Four distinct inputs decode to replacement character and digest 14bdbce57426415198e65ef79111a57df5bf125923b0e4c9cf2cc3ffee64ff13.

Evidence: `internal/host/exact_json.go:26` — Existing host decoder already rejects invalid UTF-8; the new document path omits that guard.

Preferred fix, proposed: Reject non-UTF-8 raw input and reject unpaired UTF-16 surrogate escapes before decoding values. Apply the same strict value validation to both plan and contract parsing. Preserve valid surrogate pairs and legitimate U+FFFD text.

Reason: Canonicalization may normalize JSON syntax, but must not silently replace data that different consumers reject or read differently.

Independent acceptance check:

- Run invalid_utf8, invalid_utf8_second, lone_surrogate and other_lone_surrogate tests for contracts and plans. Expected: Each malformed input is rejected before any digest is returned.
- Round-trip valid non-ASCII text, valid paired surrogates and explicit U+FFFD. Expected: Valid scalar Unicode survives and material statement/check changes change the digest.

Trade-offs and limits: A raw UTF-8 check alone does not reject escaped lone surrogates. Avoid rejecting valid international text or escaped pairs. Ordinary whitespace/key-order equivalence is not a defect.

Alternative: Require exact canonical input bytes and compare against a separately specified valid-Unicode serializer. This is stricter and may break pretty-printed documents.

### F05 — Plan validation accepts units missing required bounded-work fields

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: medium.

Observed diagnosis: ValidatePlan only requires behavior, implementer/validator routes, commands and three positive lease ceilings. A unit with no criteria advanced, allowed/forbidden scope, pre-change evidence, minimum implementation, reuse analysis, escalation route, independence rule, required evidence or stop/rollback/escalation conditions still obtains a lease. The schema has no complexity budget. ParsePlan also accepts a missing acceptance_contract_digest, although derivation later rejects that mismatch.

Evidence: `internal/model/contract.go:65` — UnitBudgets has no complexity budget field.

Evidence: `internal/model/contract.go:297` — Most architecture section 10 fields have no required-presence/value checks.

Evidence: `independent-engine.log:TestAuditorD12MissingPlanFields` — Removing those fields still returns ReasonNone.

Evidence: `independent-model.log:TestAuditorD12PlanMissingRequiredFields` — A minimal plan missing contract digest and most unit fields parses successfully.

Preferred fix, proposed: Define required fields and their explicit empty-list semantics for hma.plan/v1. Validate mandatory scalar fields, criterion coverage, scope/actions, all budget dimensions including complexity, escalation and independence, required evidence and stop/rollback/escalation declarations. Require the contract digest for the contract-backed format.

Reason: Architecture section 10 defines the unit authorization envelope; struct fields alone do not make missing declarations valid.

Independent acceptance check:

- Delete one required field at a time from an otherwise complete unit and run ParsePlan and DeriveLease. Expected: Each missing declaration is rejected; explicit empty lists only pass where the documented schema permits them.
- Present an explicit complete unit with all required declarations. Expected: Parsing and derivation continue to succeed.

Trade-offs and limits: Do not require every forbidden/non-goal/dependency list to be nonempty. Distinguish a deliberate empty declaration from omission. Adds compatibility constraints for new-format documents, not legacy approval JSON.

Alternative: Keep a separate draft-plan type and only construct an immutable validated plan when the full unit contract is present.

### F07 — Lease derivation ignores current core criterion ID drift

**HIGH · defect · observed**. Priority: before D1/D2 acceptance. Effort: small.

Observed diagnosis: Transition binding checks the current core criterion ID set, but DeriveLease checks only contract/plan digests. A fresh legitimate Apply chain followed by a structurally valid criterion snapshot replacing C1 with OTHER still returns an active lease under the old contract.

Evidence: `internal/engine/contract_binding.go:98` — Transitions call criteriaMatchBound against the current core snapshot.

Evidence: `internal/engine/lease.go:115` — Lease contract branch never compares ProjectStageState(...).Criteria to the contract.

Evidence: `independent-transition-corrected.log:TestAuditorD12CoreCriteriaMismatchLease` — Store.Append and Load accept the criterion snapshot; projected IDs=[OTHER]; derivation returns ReasonNone.

Qualification: Fresh approval is used in the corrected reproduction. It does not depend on the separate clock defect. Criterion credit and disposition remain out of scope.

Preferred fix, proposed: Compare the current projected core criterion ID set with ContractCriterionIDs before returning an active lease. Also verify it against the set at the latest acceptance transition, reusing the strict set comparison.

Reason: The immutable criteria-set contract must apply when work authority is derived, not only at the next stage transition.

Independent acceptance check:

- Run the fresh-approval criterion mismatch reproduction; test renamed, missing, extra and duplicate IDs. Expected: Each mismatch refuses a lease.
- Change only a criterion disposition while preserving exact IDs. Expected: The ID-set check remains satisfied; separate disposition policy still applies.

Trade-offs and limits: The test uses the core Store API, not an identified CLI criterion-edit command. The host is trusted; this is a missing deterministic consistency guard on a valid stored input, not malicious host forgery.

Alternative: Prevent criterion-ID mutations in the store unless a human-approved criteria rewind has been recorded, while retaining the derivation check as defense against historical inconsistent chains.

### F06 — Initial acceptance can silently drop a presented contract

**MEDIUM · defect · observed**. Priority: before D1/D2 acceptance. Effort: small.

Observed diagnosis: At initial ACCEPTANCE_CRITERIA -> PLANNING, an empty approval contract digest returns ReasonNone before checking whether a document was supplied. Apply has already parsed that document, but records a legacy approval and BoundContractDigest remains empty.

Evidence: `internal/engine/contract_binding.go:63` — Empty-digest legacy branch does not check doc != nil.

Evidence: `independent-transition-corrected.log:TestAuditorD12ApplyDropsPresentedContract` — Apply accepts sequence 3 with a supplied contract; bound digest is empty.

Preferred fix, proposed: Reject a nonnil presented contract when its approving acceptance_contract_digest is empty, including the first acceptance. Preserve the both-absent legacy case.

Reason: The approval must explicitly bind any contract being accepted. Silent downgrade obscures the actual accepted scope.

Independent acceptance check:

- Run ApplyDropsPresentedContract and a both-absent legacy control. Expected: Presented-unbound contract refuses; legacy transition still succeeds.
- Run a document with its correct digest and one with a different digest. Expected: Only the exact matching contract is accepted.

Trade-offs and limits: Callers that supplied ignored documents must add explicit binding. This defect alone does not produce a lease; no bound contract still refuses derivation.

Alternative: Reject inconsistent document/binding presence at the transition input boundary before evaluating contract state.

### F08 — Lease digest uses record anchor instead of the specified transition field

**MEDIUM · defect · observed**. Priority: before D1/D2 acceptance. Effort: small.

Observed diagnosis: The design describes a lease digest over approval transition digest, unit and plan digest. The implementation hashes ir.Metadata.HeadAnchor instead of a.TransitionDigest. An auditor test holds the existing JSON envelope constant and changes only that input; the resulting hashes differ.

Evidence: `internal/engine/lease.go:127` — Hash input is {ir.Metadata.HeadAnchor, UnitID, planDigest}.

Evidence: `independent-digest.log:TestAuditorD12SpecifiedLeaseDigest` — Actual 5f6f020e36530b112b9f3990cd1ed508e05e370816c10dffc1f6e30b8adb7c7e; transition-field input gives f9487876f12207a120a0681c64b4b7818115f6b311fc61bd8d60e9c4b0a33fe5.

Qualification: Protocol interpretation risk. The test establishes the different input, not a cryptographic collision or unauthorized lease.

Preferred fix, proposed: Specify the versioned lease-digest envelope and use Approval.TransitionDigest for the design-named input. If record-anchor identity is also required, include it explicitly under an approved clarified schema.

Reason: Independent consumers need one reproducible definition of the same cache identity.

Independent acceptance check:

- Compute a reference digest from the documented envelope and Approval.TransitionDigest, unit and accepted plan digest. Expected: Reference and implementation agree byte for byte.
- Vary each documented identity field independently. Expected: Every identity change alters the lease digest; identical input remains deterministic.

Trade-offs and limits: The authenticated record anchor already transitively binds the approval. This is not a demonstrated safety bypass. The report treats the design phrase as the existing ApprovalBinding.TransitionDigest field; that terminology could instead be clarified by the design owner.

Alternative: Explicitly amend the design to define approval transition digest as the record anchor, and publish a canonical test vector before D3 caches depend on it.

### F09 — Negative counters pass the lease ceiling check

**LOW · risk · observed**. Priority: before D3 integration. Effort: small.

Observed diagnosis: Attempts and CostSpent are signed values and only upper bounds are checked. Negative counts return ReasonNone. This weakens the pure function input contract, but no legitimate D3 reservation derivation producing negatives was demonstrated.

Evidence: `internal/engine/lease.go:163` — Only >= ceiling is checked.

Evidence: `independent-engine.log:TestAuditorD12LeaseControls/negative_counters` — Attempts=-1 and CostSpent=-20 return ReasonNone.

Preferred fix, proposed: Reject negative Attempts and CostSpent before comparing ceilings. Document that D3 must carry cumulative nonnegative consumption across generations.

Reason: Invalid counters should fail closed at the derivation boundary instead of creating artificial headroom.

Independent acceptance check:

- Run negative_counters separately for each signed counter and zero/below/equal ceiling controls. Expected: Every negative input refuses; zero and valid below-ceiling inputs pass; equality refuses.

Trade-offs and limits: Trusted-host threat model limits adversarial significance. This does not audit reservation persistence, aggregate run accounting or reconciliation.

Alternative: Use validated nonnegative counter constructors or unsigned accounting types with checked arithmetic in D3.

## Controlled unblinding

The auditor wrote and hashed Phase 1 before opening `docs/handoffs/2026-10-17-gate-d-d1-d2.md`. The frozen hash is `6bd56fecbbaf1f5a722ea63bc9225df70d64cbb27228e22877b0556e9899b98a`. The original blind assessment remains unchanged.

The handoff explicitly defers D3–D6. It says DeriveLease is pure and is not yet wired to a command. This confirms SCOPE-01. It limits the operational claim to function behavior for D2. It does not remove D1/D2 contract, time, dependency or validity obligations. No finding was withdrawn or changed in severity. The final combined run repeats the same auditor assertions after unblinding. [Unblinding record](unblinding.json).

## Remediation sequence and opportunities

First correct the high-severity derivation and contract-validation defects F01–F05 and F07. Correct F06 before accepting another contract-backed transition. Resolve the F08 digest definition before D3 records depend on it. Add the F09 counter guard before D3 integration. Each change needs scoped verification and independent re-test under the original criteria. Do not rebaseline the criteria to erase a failure.

Opportunity: retain this adversarial control matrix as acceptance regression evidence. This is a test-evidence improvement. It does not require a redesign or imply that the deferred Gate D parts are complete.

## Limitations

- Fresh audit context contains the neutral dispatch and authoritative requirements, with no implementer chat or prior audit findings. Source comments and tests are naturally visible as artifacts, not accepted as judgments.
- Context isolation is isolated. Execution isolation is a copied source tree in a shared host filesystem, not a container or OS-enforced read-only mount. Repository read-only behavior is procedural; sandbox technically permits repository writes.
- Threat model is a trusted host. DeriveLease is assumed to receive an ordered, validated core chain, accurate current policy digest and accurate counters. Forged core slices, altered anchors, malicious clocks and host model/API attacks are not claimed as in-scope exploits.
- No nested agents, agent CLIs, model APIs or external communications were used. No approved Jev tool is exposed in host metadata; judgment fell back to this auditor. No worker dispatch was performed, so route hi preflight is not applicable.
- All test artifacts, fixture repositories, cache and temporary files are under the audit output directory. Tests use fixture git writes only; no git write command targeted /home/yeli/repos. Git resolves to /usr/bin/git in the test environment.
- Initial broad tests/race tests failed only in scripts/dogfood/hostinput because the auditor set GOTMPDIR and TMPDIR to different roots. Corrected aligned-temp runs pass. Both original and corrected logs are retained.
- Auditor negative controls intentionally fail to show defects. Positive/mismatch/legacy controls pass. Full raw files are retained even where tool displays were truncated.
- D3-D6 reservations, launcher reconciliation, criterion credit, findings disposition and architecture amendment are excluded. The excluded .serena and architecture-amendment files are hashed for freeze identity only.
- The optional implementer D1/D2 handoff has not been read before this Phase 1 freeze. No /tmp/hma-gate-* directory other than this output directory and no /tmp/claude-* directory was read.
- Current live HMA run state, approval challenge and implementer route are not part of this audit target. This report neither supplies approval nor advances a stage.
- D2-08 has an explicit design-terminology interpretation. F08 does not establish forgery. Negative counters are invalid trusted-host input; F09 is a nonblocking boundary risk, not proof of broken future D3 accounting.
- After Phase 1, only docs/handoffs/2026-10-17-gate-d-d1-d2.md was unblinded. It confirms that DeriveLease is not wired to host commands. D2 findings are reproduced function-level authority errors, not demonstrated current launch bypasses. D1 contract omission is reproduced through Apply. No finding changed.

No production route, host launch, reservation, live policy configuration, external worker or human approval was created or exercised. Gate B/C regression coverage is the passing repository suite and race run. It is not a new audit of every prior gate requirement.

## Freeze verification

**UNCHANGED.** The end check matches HEAD, branch, exact status bytes, binary diff digest, every untracked file digest, and all 215 copied original source-file digests. See [start freeze](freeze-start.json), [end freeze](freeze-end.json), and [comparison](freeze-verification.json). No repository file or Git state was changed by this audit. Fixture Git writes occurred only below the output directory.

Machine report encoding: `gate_verdict: FAIL` and report-contract `verdict: fix-required` express the same decision. The skill validator checks form and verdict consistency only. See `report-validation.log` for the executed validation result. It does not prove the findings or proposed fixes.

## Required next action — architecture §18

Current outcome: FAIL

Rationale: Deterministic guards permit leases outside the approved contract or after invalidating changes. Green baseline checks do not satisfy the failing acceptance cases.

Proposed owner: Ming / the human-designated Opus implementation orchestrator (author attribution observed after unblinding)

Proposed route: Handoff only. No worker execution is requested by this report. A later repair dispatch must use its human-approved exact route and hi preflight; provider/model/access service were not supplied here.

Permission envelope: Read the audit packet. This report grants no repository edits, launch, git write, stage change, commit, push or release.

Exact approval required: None for receipt of this audit. Any repair execution or stage transition must use a still-valid existing approval or a fresh challenge bound to its exact current HMA inputs; this audit creates none.

Stop conditions: Do not declare D1/D2 accepted or proceed through the failed gate until corrective work passes scoped verification and independent re-audit. Stop if source freeze changes or required route/permission approval is absent.

Escalation conditions: Escalate design ambiguities, wider scope, route or permission changes, and unresolved high-severity findings to the human.

**Single recommended next action: Return failed Gate D D1/D2 to the implementation owner for correction.**
