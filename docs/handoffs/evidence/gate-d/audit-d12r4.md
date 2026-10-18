# Gate D D1+D2 audit

**Verdict: FAIL.** The report-contract value is `fix-required`.

The final normal suite, vet, and race suite pass. All 12 packages pass in each full test run.
The whole-run rewind finding is FIXED. Ten prior findings are FIXED. F01 is PARTIAL.
The remaining gap concerns selected-route identity and resolved permissions before an active lease.
This audit does not demonstrate an unauthorized launch. D3 launch admission remains outside scope.

Audit ID: `hma-gate-d-audit-20261017-d12r4`. Date: `2026-10-17T23:34:13.053322+00:00`.
Target: main, `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`, plus the frozen working tree.
Tracked binary diff SHA-256: `35d1b1a658336f8774e392181095af456eb646b37f4d4c6228234147afd4de2f`.
Phase 1 SHA-256: `eab409c00bd57a18c60ae0c9bb814342d3efdd44cb409c87df3141a8b25f50a1`.
Isolation: **isolated fresh auditor context**, with a filesystem copy for tests. This is not a separate VM.
Repository access stayed read-only. HEAD, status, diff digest, and all untracked hashes are unchanged.
No nested agent CLI, model API, worker dispatch, network fetch, commit, or push was used.
Requested auditor route: liz / OpenAI gpt-6.1-sol / high / current Codex host. These are dispatch metadata. The host model was not independently attested.

## Coverage

Requirements were derived before reading the diff. Source excerpts and the derived requirements are retained.

| ID | Requirement | Assessment | Independent evidence |
|---|---|---|---|
| D1-01 | Contract schema and canonical digest | MET | Strict typed parsing rejects malformed Unicode, missing/null keys, duplicate/unknown keys and trailing data. Valid scalar text survives. Material changes alter canonical digests. (E03) |
| D1-02 | Core criterion ID equality | MET | Exact core/contract ID-set checks reject renamed, missing, extra and duplicate IDs; disposition-only change preserves the ID check. (E03, E04, E05) |
| D1-03 | Optional approval binding compatibility | MET | One optional acceptance_contract_digest field is omitted empty. Independently specified legacy approval JSON is byte-identical. (E05, E06, E08) |
| D1-04 | Transition contract continuity | MET | Apply refuses presented-unbound initial contract, mismatch, later drop and swap. Both-absent legacy succeeds. Replacement succeeds only on legal return to acceptance, and prior contract use cannot be dropped there. (E05, E06, E08) |
| D1-05 | Plan schema and bounded units | MET | All declared fields are present and non-null; valid zero retry and empty argv/dependencies pass. Invalid bounded declarations and dependency cycles/unknown dependencies are rejected by code and suite. (E03, E04) |
| D2-01 | Legal and latest approval issuer | MET | Only latest applicable approved Route selection or Independent validation transition into Implementation authorization issues; illegal review edge and absent issuer refuse. (E04, E05) |
| D2-02 | Plan/contract/route/permission binding | PARTIAL | Recomputed plan/contract digests, run, unit, policy, plan route names, declared permissions and budget ceilings are checked. Selected-route identity/permission coherence cannot be established from this API; explicit F01 identity closure is unmet. (E03, E04, E11) |
| D2-03 | Stage, terminal and rewind invalidation | MET | Global Grounding/Acceptance targets void all old generations regardless of unit tag. Dependency correction/route/plan rewinds void direct/transitive leases. Genuinely independent scoped rewind preserves unrelated unit. New issuer is necessary to resume. (E04, E05, E08, E09) |
| D2-04 | Wall time, counters, policy | MET | Uses human approval timestamp; only [approval, expiry) active. Negative and equal-ceiling counters refuse; below ceilings pass; current policy mismatch refuses. (E04, E05) |
| D2-05 | Deterministic lease identity and purity | MET | Reference digest matches documented transition_digest/unit_id/plan_digest envelope; every identity component changes digest; same input is deterministic and not mutated; planless input refuses. (E04) |
| REG-01 | Existing core/Gate B/Gate C safety | MET | Full existing core, Gate B and Gate C suites, vet and race pass in isolated copy. No repository edit. (E05, E06, E07, E08) |
| SCOPE-01 | D1/D2 boundary | MET | Code implements pure derivation and transition/document binding. Counters are supplied by trusted host; no reservation, launch reconciliation, criterion credit or disposition feature is credited as complete. (E11) |

## Prior finding closure

| ID | Status | Evidence |
|---|---|---|
| F01 | PARTIAL | Plan-to-contract route and permission ceilings fixed. Selected-route identity and resolved permissions are not inputs to derivation; the required mismatch check remains absent. (E03, E04, E11) |
| F02 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F03 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F04 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F05 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F07 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F06 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F08 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F09 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| D12R2-01 | FIXED | Independent reproductions and source checks satisfy the scoped closure criteria. (E03, E04, E05, E06, E07, E08) |
| F-D12R3-01 | FIXED | Six real legal Apply rewind cases pass in normal and race modes. Old A remains void after D progress. Fresh A issuer resumes. Mutation removing the global check reproduces failure for unrelated D. (E04, E05, E06, E07, E08, E09) |

## D12R4-01 — Selected-route coherence is absent from the pure lease interface

Severity: **high**. Kind: **gap**. Confidence: **observed interface gap**. Prior item: **F01**.

Plan-to-contract route and declared-permission ceilings now work. The requested remaining F01 check is not implementable at DeriveLease: no input supplies the selected route identity, resolved route permissions, or an approval-bound projection of that route. The core approval carries an opaque transition digest; DeriveLease never resolves its route payload. Therefore D2 cannot establish that selected route == approved plan route, or that selected route permissions fit the grant, before reporting an active lease. This is an observed interface/validation gap and a mandatory closure failure. It is not a reproduced unauthorized dispatch. Launch admission remains D3–D6 scope.

Observed facts:

- `internal/engine/lease.go:38` defines the complete request. It has core records, unit, plan, contract, policy, counters, and time. It has no selected-route identity or resolved permission projection.
- `internal/model/contract.go:544` compares route names and permissions declared by the plan with contract ceilings. The auditor negative cases confirm those checks work.
- `internal/engine/lease.go:141` copies the plan implementer route into the returned lease. It does not compare that route with a separately selected route.
- The existing core route approval uses an opaque transition digest. `cmd/hma/transition.go:204` checks that reference for existing host actions. It does not resolve a route payload for pure lease derivation.
- The supplied F01 closure criterion requires identity mismatch and permission broadening to refuse before an active lease. The identity half has no executable input in this API.

Inference: green tests of plan declarations do not establish the selected-route mismatch invariant. This gap blocks the requested F01 closure. Existing Gate B host-route tests still pass.

**Preferred fix (proposed):** Define route strings as stable identities and extend the pure request with a trusted host projection of the selected route plus its approval-bound identity/permission grant. Verify that the projection references the active core route approval and matches the selected plan unit; compare resolved permissions with both unit and contract ceilings before ReasonNone. Refuse an absent projection for contract-backed leases. Preserve legal Independent validation correction issuers by retaining the preceding valid route binding.

**Why:** The comparison needs both sides: the approved plan declaration and the route actually selected under a core approval. The current request has only the first side.

**Verification:**

- Keep contract, plan bytes and accepted_plan_digest fixed. Supply matching selected profile and approval as a positive control. Then vary profile identity, permission class, approval reference and unit independently. Expected: Matching selection succeeds. Each mismatch and missing projection refuses before the lease is active.
- Use a selected bounded_write route under a read_only unit/contract, then restore read_only; include legal Independent validation -> Implementation authorization rework. Expected: Broadening refuses. Matching read_only and legal rework succeed with the proper route binding.
- Rerun existing full test/vet/race and independent lease/rewind/legacy-byte controls. Expected: All pass; no illegal issuer, scope widening or legacy serialization change.

**Trade-offs:** A required route projection changes callers of the new D2 API and needs an explicit identity convention. It must not add another authority source or treat host telemetry as human approval. This remedy is proposed; no implementation was made.

**Alternative:** Keep DeriveLease explicitly as a route-scoped grant and move selected-route admission to a separate deterministic API. That can be sound only after the human authoritatively amends the present F01 before-active closure criterion; the auditor cannot waive it.

Effort: **medium**. Priority: **1**. The proposed fix has not been implemented or validated.

## Verification and reproducibility

Run commands from `worktree/`. Complete commands and exit codes are in `commands.json`.

```sh
TMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOTMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOCACHE=/tmp/hma-gate-d-20261017-d12r4/gocache /home/yeli/.local/go/bin/go test -count=1 ./...
TMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOTMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOCACHE=/tmp/hma-gate-d-20261017-d12r4/gocache /home/yeli/.local/go/bin/go vet ./...
TMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOTMPDIR=/tmp/hma-gate-d-20261017-d12r4/runtime GOCACHE=/tmp/hma-gate-d-20261017-d12r4/gocache /home/yeli/.local/go/bin/go test -race -count=1 ./...
```

The toolchain is Go 1.27.1 on linux/amd64. The module declares Go 1.21.

The auditor authored 13 top-level tests in three files under `worktree/internal/{model,engine,transition}/auditor_d12_test.go`.
They cover required fields, Unicode, canonical digests, plan ceilings, legal issuers, time and counter boundaries, policy, contract continuity, criterion IDs, terminal outcomes, dependencies, generations, and legacy bytes.
The real Apply matrix contains six global rewind cases. It tests Grounding and Release-and-closure → Acceptance criteria, each with empty/A/D unit tags.
Each case also tests later D progress and a fresh A issuer. The same cases ran in the full race suite.

Two mutation controls fail as expected:

- `negative-control-rewind.log`: removing the global-target guard reopens old A leases for both unrelated-D global rewind cases.
- `negative-control-fields.log`: removing required-field enforcement accepts missing/null fields. The auditor tests detect it.

The initial copy-location runs failed a scratch-store test. The second environment had different TMPDIR and GOTMPDIR values and failed scratch positive controls.
The final environment sets both to `runtime/`. The copied worktree lies outside that directory. Final normal/race/vet checks pass.
These are test-environment corrections. No audited source was changed. The failed logs remain available.

## Evidence ledger

| Evidence | Source | Observation |
|---|---|---|
| E01 | `freeze-start.json; copy-manifest.json` | HEAD bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0 on main; dirty diff frozen; all 215 copied target files match source hashes. |
| E02 | `requirements-derived.json; requirements-source-excerpts.txt` | Requirements derived before diff inspection. Gate A sections restricted to v2 §2.2, v3 §2.1–2.3 and v4 §1–2. |
| E03 | `auditor-model-engine.log; worktree/internal/model/auditor_d12_test.go` | Independent recursive omission/null tests, valid empty list/zero controls, malformed/valid Unicode, material digest changes and ceiling negatives passed. |
| E04 | `auditor-engine-final.log; worktree/internal/engine/auditor_d12_test.go` | Independent legal/illegal issuer, plan/contract mismatch, terminal, counter/time/policy boundaries, lease reference digest, purity, criterion drift, direct/transitive dependency and new-generation controls passed. |
| E05 | `auditor-transition.log; worktree/internal/transition/auditor_d12_test.go` | Real Apply global rewind tests passed for Grounding and Release-and-closure -> Acceptance criteria with unit tags empty/A/D; old A stayed void after D progress; only fresh A issuer resumed. Late-approval expiry, contract continuity and exact legacy serialization controls passed. |
| E06 | `go-test-final.log` | TMPDIR=GOTMPDIR=<output>/runtime GOCACHE=<output>/gocache ~/.local/go/bin/go test -count=1 ./... exited 0; all 12 packages passed, including auditor tests. |
| E07 | `go-vet-final.log` | Same environment; ~/.local/go/bin/go vet ./... exited 0. |
| E08 | `go-race-final.log` | Same environment; ~/.local/go/bin/go test -race -count=1 ./... exited 0; all 12 packages passed, including real Apply rewind tests. |
| E09 | `negative-control-rewind-overlay.json; negative-control-rewind.log` | Removing only the new global target check caused auditor real Apply tests to fail for Grounding/D and Acceptance criteria/D; exit 1 as expected. |
| E10 | `negative-control-fields-overlay.json; negative-control-fields.log` | Removing requireAllKeys caused independent missing/null tests to fail for contract non_goals/deliverables and plan max_retries; exit 1 as expected. |
| E11 | `internal/engine/lease.go:38–47,86–198; internal/model/contract.go:544–580; internal/model/types.go:173–198; cmd/hma/transition.go:204–249` | LeaseRequest has no selected/approved route projection. DeriveLease compares route labels in plan with contract labels, then copies plan ImplementerRoute into Lease. It does not compare a selected profile or its permissions with the plan grant. Core route binding exposes an opaque transition digest, not the route payload. |
| E12 | `freeze-end.json; PHASE1.sha256` | All required start/end freeze checks equal. PHASE1 bytes retain their recorded digest after optional unblinding. |

## Controlled unblinding

`PHASE1.json` was frozen and hashed before reading the current implementer handoff. Its digest is `eab409c00bd57a18c60ae0c9bb814342d3efdd44cb409c87df3141a8b25f50a1`.
The current handoff confirms that command wiring and launch consumption are deferred to D3. It supplies no route projection that closes F01.
It does not claim D3–D6 or full Gate D completion. The frozen findings and verdict are unchanged.
No prior audit report or forbidden temporary audit directory was read.

## Remediation roadmap and opportunities

Return F01 for a bounded route-binding correction. Preserve D1 legacy serialization and the repaired global/dependency rewind behavior.
The exit criterion is the specified matching-route positive control and identity/permission/approval-reference negative controls, followed by full verification and a fresh independent audit.
The owner is the human-assigned implementer. This audit grants no execution permission.

Opportunity: retain the auditor test and mutation artifacts for the independent retest. No extra redesign is recommended.

## Limits

- Fresh audit context contains the user dispatch and allowed neutral closure criteria. No implementer narrative or prior audit verdict was read before this freeze. Code comments and author-written tests were inspected as artifacts, not accepted as proof.
- Isolation is a separate fresh auditor context and a filesystem copy. It is not a separate VM or container trust boundary. The OS, host-managed chain and caller-provided current policy/counters are trusted, as scoped.
- Only D1 and pure D2 are evaluated. No operational admission, reservation accounting, reconciliation, criterion credit, finding disposition, architecture amendment, release or full Gate D completion is certified.
- The F01 selected-route mismatch is an observed API gap. No valid runtime mismatch test can be passed into the current request type. No unauthorized launch is claimed.
- Pure dependency and counter tests use independently authored synthetic records. Six whole-run rewind scenarios also traverse the real Apply/store path with fresh approvals. Real Apply tests reuse existing low-level Git fixture and approval-construction helpers; acceptance assertions and test sequences are auditor-authored.
- Initial full normal/race runs failed the scratch-store location assertion because the copy was under /tmp. A second environment placed GOTMPDIR outside TMPDIR and failed scratch positive controls. Final runs set both to <output>/runtime, outside the worktree, and passed. All failed logs are retained.
- No Jev decision tool is available. Auditor judgments use the documented direct fallback. No worker/model API or nested agent was dispatched.
- Go toolchain is go1.27.1 linux/amd64; module declares go 1.21. No other Go version or platform was tested.
- The repository was read-only by audit conduct and sandbox protection of Git internals, not mounted as an immutable snapshot. Start/end HEAD, status, binary diff digest and every untracked hash match; this does not prove no transient concurrent change occurred. The tested copied target bytes also match the source manifest.
- Displayed tool output was sometimes truncated. Complete test logs, the copied source and the independently authored test files remain in the output directory. An initial auditor fixture draft had a syntax error; it was corrected before acceptance runs. No product finding relies on that draft failure.

## Required next action — architecture §18

**Human owner reviews this packet and returns F01 for a bounded route-binding correction before accepting D1+D2.**

- Current outcome: D1 requirements MET; D2 selected-route coherence PARTIAL; 10 prior findings FIXED and F01 PARTIAL. Full checks pass.
- Rationale: The supplied mandatory F01 before-active mismatch criterion is not established.
- Owner: Human HMA gate owner; implementation owner to be assigned through the approved route process.
- Route: No execution route selected by this audit. This next action is human review; any later coding dispatch needs an eligible human-approved route and exact-route hi preflight.
- Permission envelope: Audit stays read-only on /home/yeli/repos/HMA; artifacts only under /tmp/hma-gate-d-20261017-d12r4. No commit, push or stage transition.
- Exact approval: Human challenge-bound approval of the concrete correction packet and its route/envelope before any HMA stage transition; this audit grants none.
- Stop: Do not accept D1+D2 or advance on this packet while F01 remains partial.
- Escalate: Escalate to the human if the team proposes moving the explicit F01 criterion to D3 or changing its meaning.

Next action: human owner reviews this packet and returns F01 for the bounded correction before accepting D1+D2.
