# Gate D D1+D2 audit — FAIL

**Audit ID:** `hma-gate-d-audit-20261017-d12r3`  
**Target:** `main`, HEAD `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`, plus the frozen working tree.  
**Date:** 2026-10-17. **Modes:** code and outcome. **Threat model:** trusted host.  
**Isolation:** `compromised`. See the disclosure below.  
**JSON verdict mapping:** `fix-required` is the report-contract spelling of this single `FAIL` verdict.

One new high-severity defect blocks D1+D2 acceptance. A human-approved return to `GROUNDING` can leave an unrelated unit’s old lease active. The existing repository tests pass. The independently written reproduction fails in normal and race modes.

The audit assessed 12 requirements. Ten are `MET`. Two are `PARTIAL`. All 10 supplied prior finding closure criteria are `FIXED` within their stated scope. They do not cover the new global-rewind bypass.

## Evidence and boundary

Observed facts precede the recommendation. The audit did not change production files. The final freeze matches the start freeze. HEAD, branch, status, tracked binary diff and all untracked file digests are unchanged.

| Frozen item | Value |
|---|---|
| Tracked binary diff SHA-256 | `35d1b1a658336f8774e392181095af456eb646b37f4d4c6228234147afd4de2f` |
| Phase 1 SHA-256 | `382cd74718ac5caf49a456b163e1e6ad090259c6c23ef48ef9b3d056f73f80ef` |
| Start and end manifests | [freeze-start.json](freeze-start.json), [freeze-end.json](freeze-end.json) |
| Source/testdata copy | [snapshot-manifest.json](snapshot-manifest.json), 134 unchanged copied files |
| Go and Git | `go1.27.1 linux/amd64`; module declares Go `1.21`; `/usr/bin/git` 2.43.0 |

Tests ran from `snapshot/`. `GOCACHE`, `GOTMPDIR` and `TMPDIR` were under this audit directory. Existing tests created temporary Git fixture repositories there. No Git write targeted `/home/yeli/repos/HMA`. The test PATH resolves Git to `/usr/bin/git`.

D3-D6 are excluded. The audit does not credit reservations, launch reconciliation, criterion credit, finding disposition, or the architecture amendment. `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md` were hashed for stability and excluded from evaluation.

## F-D12R3-01 — a unit tag hides a global rewind

**Severity:** high. **Certainty:** observed. **Fix status:** proposed. **Effort:** small.

1. The fixture accepts an exact contract and plan for units `A` and independent `D`.
2. A fresh human approval authorizes unit `A`. Its lease is active.
3. `transition.Apply` accepts the legal return from `IMPLEMENTATION_AUTHORIZATION` to `GROUNDING`, with unit `D` on the approval.
4. `store.Load` verifies the resulting chain. `ProjectStageState` reports `GROUNDING`.
5. `DeriveLease` returns `ReasonNone` for `A`. It returns the same old lease identity.

| Grounding transition unit | Core projected stage | A lease result |
|---|---|---|
| empty | `GROUNDING` | refused: `LEASE_VOID_AFTER_REWIND` |
| `A` | `GROUNDING` | refused: `LEASE_VOID_AFTER_REWIND` |
| unrelated `D` | `GROUNDING` | **active**: empty reason / `ReasonNone` |

The raw [D case](rewind-reproduction/unit-D/result.json) includes separate [core records](rewind-reproduction/unit-D/records.json), [plan](rewind-reproduction/unit-D/plan.json), [contract](rewind-reproduction/unit-D/contract.json), and [returned lease](rewind-reproduction/unit-D/lease.json). The [normal log](auditor-transition.log) and [race log](auditor-race.log) contain the controls and failure.

**Diagnosis.** `internal/engine/lease.go:154-175` filters later transitions by unit scope before it classifies the target gate. A transition tagged with an unrelated unit reaches no invalidation branch. `transition.Apply` does not reject that scope. This bypass needs no forged chain or host lease record.

**Requirement.** Architecture §5.4 says changed Grounding inputs invalidate downstream packets. Gate A v3 §2.2-§2.3 forbids leases after a relevant whole-run rewind. The existing deterministic invalidation table assigns Grounding to `ScopeWholeRun` (`internal/engine/invalidation.go:70`). That engine also refuses narrowing a whole-run trigger to one unit (`:246`). The persisted transition and new lease scan do not preserve this rule.

**Preferred fix, not validated.** Classify later approved returns to `GROUNDING` and `ACCEPTANCE_CRITERIA` as whole-run rewinds before filtering by unit or dependency. Keep old generations void until a new legal issuer. Reject nonempty unit scope for whole-run grounding/criteria rewinds at the transition boundary. Keep genuine Planning and Route selection changes scoped by the approved graph.

**Verification.** Rerun the three observed controls in normal and race modes. Each must reject invalid scope before append or refuse the old A lease after append. Add the legal Release-and-closure to Acceptance-criteria variant. Check that later unrelated progress cannot revive an old generation. Check that a fresh legal A issuer can resume. Retain direct/transitive dependency and unrelated-correction controls. Then rerun the full checks and obtain a fresh isolated audit.

**Trade-offs.** A host that sends a unit tag on a whole-run rewind will need a corrected exact human approval. A lease-side guard must still handle already persisted scoped global rewinds. A blanket ban whenever the run’s global stage differs could disable valid independent unit work.

**Alternative.** Enforce a strict chain invariant at Apply and require every lease caller to use that validator. This creates more coupling. It also needs an explicit policy for already persisted scoped rewinds. The target-first derivation guard is the smaller containment.

## Requirement coverage

Each row traces to the authoritative source recorded in [requirements-derived.json](requirements-derived.json). Full observations and evidence references remain in [AUDIT.json](AUDIT.json).

| ID | Requirement | Assessment | Primary independent check |
|---|---|---|---|
| D1-01 | Contract fields and immutable definitions | **MET** | EveryFieldDeclared; explicit empty controls |
| D1-02 | Exact parsing, Unicode and digest integrity | **MET** | UnicodeAndExactJSON; parser mutation control |
| D1-03 | Approval field, transition binding and legacy bytes | **MET** | ApplyContractMatrix; ApplyReacceptAndCarry; LegacyApprovalExactBytes |
| D1-04 | Exact criterion ID set | **MET** | Criteria: renamed, missing, extra, duplicate and disposition controls |
| D1-05 | Plan format, complete units and graph validation | **MET** | EveryFieldDeclared; invalid dependency graphs |
| D2-01 | Legal issuer and active stage | **PARTIAL** | LeaseBoundaries; DependencyGenerations; WholeRunRewindWithUnitID |
| D2-02 | Presented plan and contract binding | **MET** | LeaseBoundaries: missing/mismatched documents and issuer |
| D2-03 | Route, permission, policy and budget ceilings | **MET** | RouteCeilings; policy drift controls |
| D2-04 | Unit, dependency and whole-run invalidation | **PARTIAL** | DependencyGenerations; new global rewind failure |
| D2-05 | Wall time, counters and terminal conditions | **MET** | LeaseBoundaries; RealApplyLateApproval |
| D2-06 | Deterministic lease identity | **MET** | LeaseIdentity reference SHA-256 and field variation |
| REG-01 | Transition, Gate B/C regression and scope claims | **MET** | Full tests/vet/race; exact legacy bytes; post-freeze handoff check |

`D2-01` and `D2-04` are partial because of F-D12R3-01. Legal issuance, ordinary stage progression, direct/transitive dependency correction, unrelated correction and fresh-generation controls pass. The failure concerns global rewind scope.

## Prior finding closure

| Prior ID | Status | Evidence |
|---|---|---|
| F01 | **FIXED** | `auditor-engine.log:TestAuditorD12RouteCeilings` |
| F02 | **FIXED** | `auditor-engine.log:TestAuditorD12DependencyGenerations` |
| F03 | **FIXED** | `auditor-transition.log:TestAuditorD12RealApplyLateApproval`; `auditor-engine.log:TestAuditorD12LeaseBoundaries` |
| F04 | **FIXED** | `auditor-engine.log:TestAuditorD12UnicodeAndExactJSON` |
| F05 | **FIXED** | `auditor-engine.log:TestAuditorD12EveryFieldDeclared` |
| F07 | **FIXED** | `auditor-engine.log:TestAuditorD12Criteria` |
| F06 | **FIXED** | `auditor-transition.log:TestAuditorD12ApplyContractMatrix` |
| F08 | **FIXED** | `auditor-engine.log:TestAuditorD12LeaseIdentity` |
| F09 | **FIXED** | `auditor-engine.log:TestAuditorD12LeaseBoundaries` |
| D12R2-01 | **FIXED** | `auditor-engine.log:TestAuditorD12EveryFieldDeclared`; `negative-control-field-guard.log`; `go-test.log`; `go-vet.log`; `go-race.log`; `auditor-legacy.log` |

F01 includes out-of-contract routes, permission broadening and a changed approved plan identity with the old plan preserved. F02 includes direct/transitive correction and route rewinds, an unrelated control, and fresh issuance. F03 includes a real late-approved Apply operation and nanosecond boundaries. F04 covers two malformed UTF-8 bytes, high/low lone surrogates and valid Unicode controls for both formats.

F05 and D12R2-01 use recursive removal/null tests for every declared field in both documents, all four plan units, budgets and checks. The normal field suite has 328 omission/null cases plus an explicit empty/zero control. It refuses incomplete plans at parsing and derivation. A deliberate Go overlay removes only `requireAllKeys`; 12 leaf cases then fail as expected. These include retry budgets and contract arrays. This is a negative control, not a defect in the audited bytes.

F07 checks renamed/missing/extra/duplicate IDs and ID-preserving dispositions. F06 includes presented-unbound refusal and the both-absent legacy control through Apply. F08 uses an independently constructed reference JSON envelope. F09 tests every signed counter below zero and at/below its ceiling.

## Reproducible checks

All commands ran with `/home/yeli/.local/go/bin` on PATH and the environment stored in the matching `.json` file. Use `python3 run-check.py NAME /home/yeli/.local/go/bin/go ...` from this directory to repeat them. Audit-only tests are in [engine test source](snapshot/internal/engine/auditor_d12r3_test.go) and [transition test source](snapshot/internal/transition/auditor_d12r3_test.go).

| Command after `go` | Exit | Evidence |
|---|---:|---|
| `test -count=1 ./...` | 0 | [go-test.log](go-test.log) |
| `vet ./...` | 0 | [go-vet.log](go-vet.log) |
| `test -race -count=1 ./...` | 0 | [go-race.log](go-race.log) |
| `test -count=1 -v ./internal/engine -run ^TestAuditorD12` | 0 | [auditor-engine.log](auditor-engine.log), 443 named test/subtest runs |
| `test -count=1 -v ./internal/transition -run ^TestAuditorD12` | 1 | [auditor-transition.log](auditor-transition.log), only the new rewind case fails |
| `test -race -count=1 -v ./internal/engine ./internal/transition -run ^TestAuditorD12` | 1 | [auditor-race.log](auditor-race.log), same rewind failure; no race warning |
| `test -count=1 -v ./internal/engine -run ^TestAuditorD12LegacyApprovalExactBytes$` | 0 | [auditor-legacy.log](auditor-legacy.log) |
| `test -overlay=negative-control-overlay.json -count=1 -v ./internal/engine -run ^TestAuditorD12EveryFieldDeclared$` | 1, expected | [negative-control-field-guard.log](negative-control-field-guard.log), 12 mutant leaf failures |
| `test -count=1 -v ./internal/host -run ^TestGateC` | 0 | [gatec-detail.log](gatec-detail.log), Python parity checks run without skips |

The full checks ran before audit-only tests were added. The later focused checks exercise those audit additions. Copied production bytes did not change. Full raw logs are available; some tool previews were truncated. The report validator checks form only. It does not prove findings, fixes, or independence.

## Controlled unblinding and limitations

Phase 1 was created once and hashed before reading the D1+D2 handoff. Its digest is `382cd74718ac5caf49a456b163e1e6ad090259c6c23ef48ef9b3d056f73f80ef`. [PHASE1.json](PHASE1.json) remains unchanged. The later handoff read confirms that D3-D6 and live host enforcement remain deferred. No finding or verdict changed. No earlier Gate D audit directory or `/tmp/claude-*` directory was read.

**Isolation disclosure.** An early `sed` command read Gate A v3 lines 1-28 before the requested §2.1-§2.3. This exposed historical Gate A verdicts and an author change summary outside the allowed sections. It did not expose earlier Gate D verdicts or the D1+D2 handoff. This report records `isolation_compromised=true`. It does not claim full independent blinding. A fresh isolated auditor should review the eventual fix. The failure evidence remains directly reproducible.

The requested auditor route was Liz / OpenAI / `gpt-6.1-sol` / high / Codex review session. This is coordinator-supplied metadata. Runtime model and effort were not independently attested. No worker was dispatched. No nested agent CLI or model API was used. No callable Jev adapter was exposed. The audit used the required auditor fallback.

The audit used the [independent-audit skill](/home/yeli/obsidian/AgentSystem/agent-memory/skills/independent-audit/SKILL.md). It assessed code and observable outcomes. It did not implement or validate a fix. It did not test the proposed Acceptance-criteria variant. The trusted-host scope excludes same-UID tampering claims. The dirty tree has no single immutable Git revision; the manifests identify its evaluated bytes.

No separate redesign opportunity is proposed. The smallest useful change is the global rewind scope fix and its regression controls.

## Remediation task and required next action (§18)

**Current outcome:** FAIL. The new high-severity rewind bypass remains open.  
**Rationale:** A global rewind must remove stale lease authority.  
**Proposed owner:** human gate owner.  
**Proposed execution route:** return the bounded task to the recorded author, Ming / Anthropic / `claude-opus-5-5` / effort not recorded / Claude Code CLI. This is a proposal. It is not a dispatch or eligibility claim. Deterministic route eligibility, an explicit effort setting, and exact-route `hi` preflight are required before dispatch.  
**Permission envelope:** only global rewind invalidation, transition scope refusal, and regression tests. No automatic stage changes, commits, pushes or releases. This audit retains repository read-only access.  
**Exact approval required:** an explicit human decision bound to the eventual remediation task, route, effort, frozen target and permission envelope. This packet grants no approval. There is no core transition challenge to answer in this audit.  
**Stop and escalation conditions:** hold D1+D2 acceptance and D3 progression. Stop on changed target bytes, failed controls, disputed scope, or missing route approval. The exit criterion is passing checks plus fresh isolated review of the fix.

**Single next action:** the human gate owner reviews F-D12R3-01 and approves one bounded remediation dispatch.
