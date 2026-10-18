# Gate D D1+D2 independent audit

**Verdict: FAIL.** One observed medium-severity defect remains. An incomplete plan can obtain an active lease. The audit did not demonstrate a broader numeric grant or an unauthorized process launch. The full existing test suite, vet and race checks passed.

Dispatch: `hma-gate-d-audit-20261017-d12r2`. Date: `2026-10-17T23:08:42.751754+00:00`. Modes: code and outcome. Context isolation: **isolated during Phase 1**, with controlled unblinding after its freeze. Test isolation: a source copy on the shared filesystem. Target access was read-only by instruction; it was not a separately verified OS-enforced read-only mount.

## Scope and provenance

The authoritative objective is Gate D in `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:58`. The approved design sources are Gate A v2 §2.2, v3 §2.1–§2.3, and v4 §1–§2. Only those Gate A sections were inspected for requirements. Architecture §5, §6, §10, §18 and §19 and `AGENTS.md` govern this packet. `REQUIREMENTS_PRE_DIFF.json` records the 13 requirements and failure hypotheses written before diff inspection.

Target: branch `main`, HEAD `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`, plus the uncommitted working tree. Binary diff SHA-256: `35d1b1a658336f8774e392181095af456eb646b37f4d4c6228234147afd4de2f`. `FREEZE_START.json` records the exact `git status --short` output and SHA-256 of every untracked file. The start freeze is `2026-10-17T22:56:53.094176+00:00`. The end freeze is `2026-10-17T23:08:42.699396+00:00`. HEAD, branch, status, diff hash and every untracked-file hash are **unchanged**. All 215 copied source files also match their manifest.

The audit excludes `.serena/`, `docs/handoffs/2026-10-16-arch-amendment-*.md`, and D3–D6. Ignored untracked paths were hashed for the freeze; their content was not reviewed. Reservations, durable counter accounting, launch reconciliation, criterion credit, findings disposition, actual process enforcement and the amendment are later work. The unblinded handoff states this split and does not claim full Gate D completion.

No nested agent CLI, model API, subagent, Git write to the audited repository, or repository edit occurred. Go tests created fixture repositories only under the output directory. The Go binary reports `go1.27.1 linux/amd64`. The requested route was `liz / gpt-6.1-sol / high`, review mode. This audit cannot independently attest that runtime model identity. No approved Jev tool was exposed through the permitted host interface. The auditor used the documented fallback.

## Requirement coverage

`MET` means the scoped requirement has satisfactory observed evidence. `PARTIAL` means a demonstrated gap remains. It does not mean the entire feature is absent.

| Requirement | Assessment | Observation and evidence |
| --- | --- | --- |
| D1-01 | PARTIAL | Contract fields and no-result schema exist. Omitted non-goals/deliverables and absent/null argv still parse. `TestAuditContractFieldsDeclared`, `TestAuditExplicitCheckArgv`. |
| D1-02 | MET | Exact criterion-ID binding, Unicode rejection, unique/known keys and semantic digest mutations pass. Canonical whitespace/key-order equivalence also passes. `TestAuditExactJSONAndCanonicalization`, `TestAuditAllDocumentLeavesBound`, `TestClosureF04UnicodeBothDocuments`. |
| D1-03 | MET | Initial exact acceptance, later carry-forward, terminal carry-forward and legal replacement after rewind pass. Omission and swaps refuse. Legacy empty binding remains omitted. `TestAuditContractCarriedAllTransitions`, `TestAuditContractReacceptAfterLegalRewind`, `TestClosureF06InitialContractExactness`. |
| D1-04 | PARTIAL | Plan schema and §10 fields exist. Missing/null max_retries and absent/null verification argv still pass. `TestAuditAllUnitFieldsDeclared`, `TestAuditNullRetryBudget`. |
| D1-05 | MET | Duplicate/unknown/self/cyclic dependencies, contract criteria, route, permission and budget conflicts refuse. Complete positive controls pass. `TestAuditPlanDependencyGraph`, `TestAuditPlanBoundaries`, `TestAuditLeaseRejectsContractCeilingConflicts`. |
| D2-01 | MET | Only the latest legal unit issuer supplies authority. No issuer, wrong unit and illegal-edge controls refuse. Legal validation correction creates a fresh generation. `TestAuditLeaseAdversarialMatrix`, `TestAuditLatestLegalReworkIssuer` and existing legal-edge tests. |
| D2-02 | PARTIAL | Plan digest/run/unit and contract checks work. A plan missing the required retry declaration nevertheless obtains an active lease through real Apply/store calls. `TestAuditMissingRetryCanReachActiveLease`. |
| D2-03 | MET | Leaving authorization, terminal outcomes and relevant rewinds refuse. Legal rework resumes only from a fresh issuer. `TestAuditLeaseAdversarialMatrix`, `TestAuditLeaseAllTerminalOutcomes`, `TestAuditLatestLegalReworkIssuer`. |
| D2-04 | MET | Direct/transitive dependency rewinds and validation corrections refuse. Independent units remain eligible. `TestAuditLeaseAdversarialMatrix`, `TestClosureF02DependentGeneration`. |
| D2-05 | MET | Approval-time interval and attempt/cost limits are enforced. Negative counters refuse. Contract route/permission/budget ceilings refuse conflicts. These are pure checks on trusted consumption inputs, not D3 accounting. `TestClosureF03RealLateApproval`, `TestAuditLeaseAdversarialMatrix`, `TestClosureF01ReadOnlyCeiling`. |
| D2-06 | MET | Current policy digest must match the plan. Contract, issuer and core criterion IDs must match. `TestAuditLeaseAdversarialMatrix`, `TestClosureF07FreshIssuerCriterionSet`. |
| REG-01 | MET | All 12 existing packages pass ordinary and race tests. Vet passes. No existing Gate B/C regression was observed. `corrected-test.log`, `corrected-vet.log`, `corrected-race.log`. |
| SCOPE-01 | MET | Code and the later unblinded handoff reserve host-attempt integration and D3–D6 work for later stages. `internal/engine/lease.go:31`, `UNBLINDED_HANDOFF.md`. |

The result is 10 MET and 3 PARTIAL requirements. No requirement was marked MET from an implementer success claim alone.

## Observed defect: D12R2-01

**Required document fields can be omitted or null. Severity: medium. Confidence: observed. Fix status: proposed.**

`internal/model/contract.go:318` accepts nil `Check.Argv`. The contract list check at line 367 checks entries but does not require the non-goal or deliverable arrays. The retry check at line 481 accepts zero. Typed JSON decoding turns a missing/null retry value into zero. It turns a missing/null argument vector into nil. The exact-key decoder does not enforce required-key presence.

Independent tests deleted each unit field and each budget field from a complete plan. Missing `max_retries` was the remaining unit-budget omission that parsed and leased. `max_retries:null` also parsed as zero. Contract and plan checks accepted missing/null `argv`. Contract parsing accepted omitted `non_goals` and `deliverables`. These outcomes conflict with the required document declarations in Gate A v2 §2.2, v3 §2.2 and architecture §10.

The stronger reproduction used actual `transition.Apply` calls and a verified `store.Load` result. It presented a plan without `max_retries`. The accepted-plan digest was human-bound in the transition fixture. `DeriveLease` returned `ReasonNone` and lease digest `635cd326da24cf92bdee55a172bf302b98f6ba7c33655c96660d7bc8e0b8010e`. See `TestAuditMissingRetryCanReachActiveLease` in `independent_transition_test.go` and `independent-complete-v2.log`. The same defect reproduced in the race run. It does not show that zero retries grants extra retries. It shows that an incomplete required plan declaration is accepted as lease authority.

**Preferred repair:** enforce required-key presence and JSON type before typed decoding. Require a present, non-null integer for `max_retries`. Require present arrays for `argv`, `non_goals` and `deliverables`. Allow explicit `[]` where an empty list is valid. Allow explicit `max_retries:0`. Reject nil argv in `validCheck`. Keep canonical digests unchanged for documents that already meet the contract.

**Verification:** the supplied missing/null controls must pass by observing refusal. Explicit zero and explicit empty-array controls must still pass where valid. `TestAuditMissingRetryCanReachActiveLease` must stop at parsing. Then run the full existing test, vet and race commands and obtain independent review of the frozen repair.

**Alternative:** decode into pointer/raw-message fields, check presence/type, then convert to the public model. This needs more conversion code but preserves the public value types. **Trade-off:** fixtures with null arrays need explicit declarations. Changing null to `[]` changes a document digest. Do not silently rebind an existing approval. Obtain the applicable fresh approval. **Effort:** small. Resolve this boundary defect before D1/D2 acceptance. No remedy was implemented or validated in this audit.

## Prior finding closure

The neutral criteria file was read only after Phase 1 was hashed. Its SHA-256 is `8461eb5c1414e25ef4bd8b44d5d9449ecfe0ce7bc491275ccc7e94ffc13efa7a`. A retained copy is `CLOSURE_CRITERIA.json`. No prior status or verdict was supplied in that file.

| Prior ID | Closure | Independent observation |
| --- | --- | --- |
| F01 | PARTIAL | The in-scope plan route and permission ceilings are enforced. Outside routes and bounded_write against a read_only contract refuse. The returned lease retains the plan implementer route. The additional criterion asks for an attempted/approved-route identity mismatch while preserving the plan. LeaseRequest has no independent attempted-route/profile/permission input. No such mismatch rejection was reproduced through the current pure API. Actual requested-route enforcement is not yet wired and belongs to D3. This is an incomplete closure check, not a demonstrated D1/D2 route-identity bypass. |
| F02 | FIXED | Direct and transitive dependency rewinds and corrections refuse. An independent unit stays active. A fresh legal issuing transition resumes the dependent with a different generation; the old issuer is not restored. |
| F03 | FIXED | A real Apply/store case accepts an otherwise legal two-minute-old approval for a one-minute plan, then refuses its expired lease. The request just before expiry succeeds; exact expiry and time before approval refuse. |
| F04 | FIXED | Contracts and plans reject both malformed UTF-8 byte variants and lone surrogate variants before returning a digest. Valid non-ASCII text, surrogate pairs and U+FFFD pass. Independent leaf mutations change the digest or refuse. |
| F05 | PARTIAL | Most required unit fields now refuse when deleted. Missing max_retries still parses and leases; null max_retries also parses as zero. Missing/null verification argv also parses. A complete explicit plan is a passing control. |
| F06 | FIXED | Initial presented-but-unbound contracts and wrong digests refuse. A both-absent legacy control and an exact document/digest control pass. Later and terminal transitions also reject omission or swapping. |
| F07 | FIXED | A fresh issuer does not hide renamed, missing, extra or duplicate criterion IDs. All refuse. A disposition-only change with the same ID set remains eligible for this ID-set check. |
| F08 | FIXED | A separately constructed JSON envelope hashed with SHA-256 equals the lease digest. The digest uses Approval.TransitionDigest, unit and plan digest. Repeated input is deterministic. Changing the record anchor alone does not change it; identity-field changes do. |
| F09 | FIXED | Each negative signed counter refuses. Zero and below-ceiling controls pass. Equality refuses. A legal rework issuer does not erase the counters supplied by the trusted caller. Durable counter derivation is D3. |

Seven prior findings are FIXED. Two are PARTIAL. F01 is not counted as a demonstrated D1/D2 identity bypass. Its separate attempted-route check remains unexecuted because that input is outside this pure API. A later host-integration review must compare the attempted route and permission classes with the approved lease/plan and existing core route approval before launch. This audit does not approve that later work.

## Verification and raw evidence

All commands ran in `snapshot/`. `PATH` preferred `~/.local/go/bin`. `GOCACHE` was the output directory’s `gocache/`. Corrected runs used the same output directory’s `test-tmp/` for both `GOTMPDIR` and `TMPDIR`. Source files under the audited repository were not changed.

| Check | Observed result | Raw record |
| --- | --- | --- |
| `go test -count=1 ./...` | PASS, all 12 packages | `corrected-test.log` |
| `go vet ./...` | PASS | `corrected-vet.log` |
| `go test -race -count=1 ./...` | PASS, all 12 packages | `corrected-race.log` |
| Independent overlay tests, ordinary and race | Five top-level tests fail on the required-field defect. Other independent controls pass. No data-race report appeared. | `independent-complete-v2.log`, `independent-race.log` |
| Closure-specific race tests | F05 field-removal test fails on max_retries. Other closure tests pass. | `closure-tests.log` |
| Temporary policy-check mutation | Policy negative controls fail, as expected | `mutation-policy.log` |
| Temporary UTF-8-check mutation | Malformed Unicode negative control fails, as expected | `mutation-unicode.log` |
| Temporary contract-binding mutation | Criterion drift negative control fails, as expected | `mutation-binding.log` |

Mutation overlays changed temporary copies only. They are test-sensitivity controls. They are not repairs or evidence about an unmodified failing path. The initial `go test ./...` and `go test -race ./...` runs used different temporary-directory settings. Synthetic-store tests rejected those fixture paths. Those failures came from the audit environment. The corrected full runs passed. Initial logs and both result manifests are retained; the failures were not erased.

`VERIFICATION_RESULTS.json`, `BASELINE_RESULTS.json`, `INDEPENDENT_COMPLETE_V2_RESULT.json`, `CLOSURE_RESULT.json` and `MUTATION_RESULTS.json` retain commands, exit codes and source hashes where recorded. `overlay.json` and `overlay-closure.json` map external tests onto the source copy. The `independent_*_test.go` and `closure_*_test.go` files retain all reproductions. Some pure engine tests reuse the repository fixture constructor. The strongest incomplete-plan and late-approval cases use real transition/store calls.

## Controlled unblinding and limits

`PHASE1.json` remains unchanged. Its SHA-256 is `6a3527880f2eb7583dcee662209937ee8bf1c66ee035eafc8e9e17a86a00c69c`. After that freeze, the audit read only the neutral closure criteria and the previously withheld D1/D2 handoff from the prohibited initial-context set. It did not read earlier audit directories, verdicts or `/tmp/claude-*` content. Closure-specific tests confirmed the blind defect. No blind finding was removed or rewritten.

This is a trusted-host audit. Core records, current policy digest, counters and time are trusted caller inputs. No hostile-host or cryptographic-authenticity claim is made. Pure lease eligibility does not prove actual host launch enforcement. Durable counters and route-at-launch comparison remain later work. No material improvement opportunity beyond the smallest required repair was identified in this scope.

## Required next action — architecture §18

**Return the failed D1/D2 packet to the implementer for the required-field repair and independent retest.**

Owner: the human gate owner. Rationale: an incomplete required plan can obtain an active lease. The immediate action is a human handoff. Any later coding dispatch must use its approved implementer route and exact-route preflight. This audit selects and launches no agent.

Permission envelope: read and return this packet. This audit authorizes no repository write, dispatch, commit, push, release, approval or stage advancement. No stage-transition approval is requested here. The human must make the gate decision. Any later transition needs its own fresh, exact challenge-bound approval. Stop D1/D2 acceptance while D12R2-01 remains open. Escalate any proposed criteria, scope or permission expansion.
