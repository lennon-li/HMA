# HMA Gate D D3 audit

**Verdict: PASS.** All scoped D3 requirements are MET. All six prior findings are FIXED. No blocking defect was observed.

Dispatch: `hma-gate-d-audit-20261018-d3r3`. This is a code and outcome audit. The verdict applies to D3 only. It does not accept D4 or D5.

## Evaluated target

- Branch: `main`. HEAD: `0d90eed831741db297c220ee83f123e2554e06a0`.
- Tracked binary-diff SHA-256: `70b827844a42b0d90e6d0a7a6984f8ef73e2798e8f7de69d12a6d2c59828e072`.
- All untracked-file hashes are in [freeze-start.json](freeze-start.json). The ignored scope paths are `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`.
- [freeze-end.json](freeze-end.json) matches HEAD, branch, complete short status, binary diff, and every untracked-file hash. The repository is unchanged.
- Dirty source bytes are retained in `snapshot/` and `tracked-working-tree.patch`. The verdict is bound to this working tree, not HEAD alone.

## Independence and method

Isolation: **isolated auditor context, instruction-enforced read-only repository use**. This session received the original task and primary sources. It received no implementer conversation. The host filesystem is shared and technically writable. This audit does not claim an OS-enforced read-only mount or independently attested model identity. The supplied route metadata names Jax / OpenAI / gpt-6.1-sol / high / review. That backend identity is not independently verified.

The auditor read the independent-audit skill in code and outcome modes. No worker was dispatched. No nested agent CLI or model API was used. An exact-route preflight was therefore not required. No approved Jev tool was exposed. The auditor used the documented fallback decision path.

Requirements and failure hypotheses were recorded in [requirements-derived.json](requirements-derived.json) before implementation inspection. Phase 1 was frozen before reading the prior closure criteria or implementer handoff.

PHASE1 SHA-256: `a2950186c47bd70c49889225956fff70099bd8b179137f3e27a76ebe8fd04f39`. [PHASE1.json](PHASE1.json) remains unchanged. The final report adds controlled-unblinding closure tests. The handoff makes no D4 or D5 completion claim.

## Verification evidence

| Check | Result | Evidence |
|---|---|---|
| Full repository tests | Exit 0 | [go-test.log](go-test.log) |
| Go vet | Exit 0 | [go-vet.log](go-vet.log) |
| Full repository race tests | Exit 0 | [go-race.log](go-race.log) |
| 11 blind auditor test functions | Exit 0; 36 PASS entries with subtests | [auditor-tests.log](auditor-tests.log) |
| Blind auditor tests with race detector | Exit 0 | [auditor-race.log](auditor-race.log) |
| 6 closure test functions | Exit 0; 33 PASS entries with subtests | [closure-tests.log](closure-tests.log) |
| Closure tests with race detector | Exit 0 | [closure-race.log](closure-race.log) |
| Reservation write-cut recovery | Exit 0; before/after record controls | [write-cut.log](write-cut.log) |
| Fresh consumption after stage, contract, policy, lease-time, and route changes | Exit 0 with race detector | [fresh-denials.log](fresh-denials.log) |
| Three injected fault controls | All failed by the intended assertion | `mutation-checks.json`, `mutant-*.log` |

Go version: `go1.27.1 linux/amd64`. `GOCACHE` is under this output directory. Commands, parameters, exits, source hashes, and complete raw logs remain available in [COMMANDS.md](COMMANDS.md), [EVIDENCE.json](EVIDENCE.json), and [environment.json](environment.json).

The fault controls bypassed the ledger check, lost the attempt counter, and bypassed replay conflict handling. The independent tests detected each fault. The ordinary code and race checks used the original source. Only the fault and write-cut checks used altered source overlays.

The write-cut test injects a returned error before or after the durable reservation record write. A retry recovered one charged reservation. This is a deterministic commit-error experiment. It is not a power-loss experiment.

## Requirement coverage

| Requirement | Assessment | Independent evidence |
|---|---|---|
| D3-01 Replay identical triple and request digest first; return unchanged reservation without recharging even inactive or exhausted. Conflict on changed digest. | MET | `auditor-tests.log:TestAuditorAdmissionConsumesNothingAndUnknownStaysCharged`, `auditor-tests.log:TestAuditorReplayConflictAndRevocation` |
| D3-02 First controller admission binds envelope and exact route; requires active lease, stop escalation, ceilings within remaining attempts/time/spend; no consumption on admission. | MET | `auditor-tests.log:TestAuditorAdmissionNegativeControls`, `auditor-tests.log:TestAuditorAdmissionConsumesNothingAndUnknownStaysCharged` |
| D3-03 Each fresh reservation requires matching admission, active lease, capacity, allowed failure signature; exactly one durable charge under per-run lock. Unknown/unlaunched reservations stay charged. | MET | `auditor-tests.log:TestAuditorFailurePersistsAcrossController`, `auditor-tests.log:TestAuditorCostMonotonicAndCeiling` |
| D3-04 Counters remain monotonic across rewind and new leases. Concurrent callers cannot share last slot. | MET | `auditor-tests.log:TestAuditorConcurrentLastSlot`, `auditor-tests.log:TestAuditorReplayConflictAndRevocation` |
| D3-05 Replay eligibility requires live lease by time/stage/policy/contract and unused dispatch ID; capacity not rechecked. Old generations cannot revive slots. | MET | `auditor-tests.log:TestAuditorExpiredReplayAndLastSlot`, `auditor-tests.log:TestAuditorUsedAndUnreadableLedgerRefused` |
| D3-06 Bind hook request to authoritative controller state, actual envelope, HMA store/run/unit, root, baseline, ledger namespace and exact route. Refuse wrong route and non-stop escalation. | MET | `auditor-tests.log:TestAuditorRequestBindingNegativeControls`, `auditor-tests.log:TestAuditorAdmissionNegativeControls` |
| D3-07 Endpoint works with verify-loop contract, launch needs literal eligible true, resume reuses dispatch ID and charges once. | MET | `auditor-tests.log:TestAuditorActualControllerAuthorityContract`, `go-test.log` |
| D3-08 No regression of D1/D2 lease/counter safety, Gate B kernel/approval safety, Gate C evidence integration. Do not claim deferred D4 reconciliation or D5 credit/disposition. | MET | `go-test.log`, `go-vet.log` |

The detailed JSON report has 23 D3 subchecks. There are 19 independent test functions across the audit overlays. The endpoint granted a recovered final slot without recharging it. It refused used, expired, revoked, other-generation, wrong-route, over-capacity, repeated-signature, and non-stop cases. Eight competing callers obtained exactly one final-slot grant. Admission consumed nothing. Unknown launches retained their charge.

The real controller test used `authority_request`, `invoke_authority_hook`, and `authorize_launch` from the installed verify-loop script. It preserved the dispatch ID and reservation through saved-state replay. A used slot failed the controller’s literal `launch_eligible is True` check. No real worker was launched.

## Prior finding closure

| Finding | Status | Observation and evidence |
|---|---|---|
| D3J-01 | FIXED | Eight controls hold either host or core lock for new consumption or replay. The clock is not sampled while the lock is held. Expired calls grant no launch; in-window controls grant. Existing reservations remain charged. Independent write-cut tests also conserve charges across commit errors. Evidence: `closure-tests.log:TestClosureClockWaitBothLocksNewAndReplay`. |
| D3J-02 | FIXED | Wrong namespace refuses before admission. Lexical and controller-directory symlink aliases resolve to the same used dispatch. Canonical absent-path control grants once. Used and unreadable paths refuse launch. Evidence: `closure-tests.log:TestClosureNamespaceAliasesAndUsedReplay`. |
| D3J-03 | FIXED | Same-unit repeated signature refuses across controllers and within one controller. Distinct signature is accepted. A synthetic other-unit host reservation does not block U1; this is a guard-isolation control, not a full two-unit workflow. Evidence: `auditor-tests.log:TestAuditorFailurePersistsAcrossController`. |
| D3J-04 | FIXED | Matching missing/default, empty, and nonempty inventories are accepted. Incorrect digest refuses before admission. A changed inventory invalidates the old request and preserves the existing charge. Evidence: `closure-tests.log:TestClosureBaselineInventory`. |
| D3R2-01 | FIXED | Contradictory phase, route including cost, signature omitted or changed, attempt index, and dispatch ID refuse without reservation. The actual repeated signature is still refused. Historical replay and the real controller authority_request/invoke_authority_hook/authorize_launch boundary work. Last-slot replay charges once. Evidence: `closure-tests.log:TestClosurePendingContradictionsAndHistoricalReplay`. |
| D3R2-02 | FIXED | Fresh requests with directory, file, symlink, dangling symlink, or unreadable runs path refuse before admission. An absent path grants one charge. Replay, unknown launch, and commit-error recovery preserve exactly one reservation. Evidence: `closure-tests.log:TestClosureFreshLedgerPathKinds`. |

## Findings and remediation

No actionable defect, risk, or gap remains in the scoped assessment. No code remedy is proposed. No material redesign opportunity is recommended. The synthetic other-unit failure record tests guard isolation. It does not prove a complete two-unit workflow.

Observed regression evidence: full repository tests, vet, and race tests passed. The production diff adds the endpoint and host record types. It does not change the safety kernel. This supports the scoped regression conclusion; it is not proof of all possible executions.

## Limits and omissions

- D4 launch reconciliation, `UNAUTHORIZED_DISPATCH`, revocation minima, and import generation rules were not accepted here. D5 criterion credit and findings disposition were not accepted here.
- Auditor tests reuse existing accepted-core and repository fixture constructors. The test inputs, assertions, controller-boundary test, lock-wait tests, and injected faults were authored independently.
- The existing untracked `hma` binary was hashed and its Go build metadata was inspected. Runtime endpoint tests built fresh binaries. The existing binary was not independently matched byte for byte to source.
- The first added controller test expected an incorrect literal error message. The auditor corrected that expectation. The failed ordinary log remains in `initial-auditor-tests.log`. The corresponding failed race log was overwritten by the final successful run. This omission does not conceal a reproduced product failure.
- Raw logs are complete on disk. Host-rendered tool output may be compressed. No evidence is taken from an implementer explanation alone.
- Dirty-worktree evidence is snapshot-bound. Hostile-host protection, real power loss, real external worker execution, review import, and outcome credit are not claimed.

## Single next action — architecture §18

**Recommendation:** The human approver should review and accept D3 for this frozen snapshot.

**Current outcome:** PASS. All scoped requirements are MET. Six prior findings are FIXED. The repository freeze is unchanged.

**Rationale:** Independent runtime checks and fault controls found no blocking D3 defect.

**Owner:** Human approver. **Agent/model route:** NOTHING. **Permission envelope:** Read this packet. No repository write, worker launch, stage transition, commit, push, or release is authorized by the audit.

**Exact approval required:** Explicit human D3 acceptance. D4 needs its own authorized dispatch.

**Stop and escalation conditions:** Stop if the snapshot changes or new critical/high evidence contradicts this packet. Return the changed target for independent review.

**Next action:** Human review and accept D3.
