# HMA Gate D D1+D2 audit

**Verdict: FAIL.** D1 is MET. D2 is PARTIAL. One high-severity defect remains.

A new correction approval can obtain an active lease through an older route approval that a waiver operation invalidated. The new approval can bind the exact current waiver set. This is a lease-derivation failure in D2. It does not depend on D3 command-launch behavior.

Dispatch: `hma-gate-d-audit-20261017-d12r7`. Target: `main`, HEAD `bc5acb7c0bdc634f31a0644ee17d0f610d7f4be0`, plus the frozen working tree. Tracked diff SHA-256: `c119b7f95c3cff42c5a2f13037bbef88fefbde6093a55c355a1ccbe64dca19fb`. The full untracked inventory is in [freeze-start.json](freeze-start.json).

Isolation: **isolated auditor context; same-host copy execution**. This session received the original sources and the user-authorized closure criteria. It did not receive implementation chat or prior audit reports. The copies are not a separate VM or an OS-enforced read-only mount. Repository read-only access was an operating constraint. All writes and temporary fixture repositories stayed under this output directory.

The blind findings are immutable in [PHASE1.json](PHASE1.json). SHA-256: `477e7d292fea64f470618eaa20cdf5f492d95bc1b9f1ef6761b23c50f45ef61f`. Requirements were recorded before diff inspection in [requirements-before-diff.json](requirements-before-diff.json).

## Requirement coverage

Sources: outcome-consolidation §5 Gate D; Gate A v2 §2.2; v3 §2.1–§2.3; v4 §1–§2, with latest revisions taking precedence; architecture §5, §6, §10, §18; and AGENTS.md. The trusted-host threat model applies.

| Requirement | Assessment | Observation and evidence |
| --- | --- | --- |
| D1-01 — Immutable canonical acceptance contract with all required fields; material content changes change digest; reject ambiguous JSON and malformed Unicode. | MET | Strict parser rejects missing/null fields, malformed Unicode, duplicate/case/unknown keys and trailing documents. Independent edits to statements, checks and thresholds change the digest. E03, E04 |
| D1-02 — Contract criterion IDs exactly equal current core criterion IDs. | MET | Acceptance compares exact ID sets. Lease derivation refuses renamed/extra/duplicate IDs and preserves disposition-only changes. sameIDs also checks cardinality. E03, E07 |
| D1-03 — Optional approval field omitted when empty preserves legacy digests; presented contract cannot be dropped, swapped or accepted unbound; later approvals carry same digest until human approved rewind. | MET | Real Apply refuses presented-unbound and wrong-digest contracts. Exact and both-absent legacy controls pass. Existing carried-digest, drop/swap and legacy-byte checks pass. E03, E04, E07 |
| D1-04 — hma.plan/v1 canonical document includes every §10 declaration plus policy, wall time and cost; validates graph and budgets. | MET | Every serialized document field is checked for omission/null, including zero retries. Incomplete unit and budget fields refuse DeriveLease. Plan graph, policy, contract, route and budget validation are present. E03, E04, E07 |
| D2-01 — Lease uses latest legal human-approved issuer targeting IMPLEMENTATION_AUTHORIZATION; no record or illegal edge grants authority; identity binds approval transition digest, unit and plan digest. | MET | Only Route selection and Independent validation issuers are accepted. Reference lease digest agrees with transition/unit/plan fields; unit, transition and plan identity changes alter it. E03, E04 |
| D2-02 — Plan digest equals accepted plan; contract and criterion IDs match; grants stay inside contract and selected approved route and permissions. | PARTIAL | Plan, contract, criterion, route identity, permission and approval-reference checks pass. The retained route approval can nevertheless be invalidated by its bound waiver and still authorize a correction lease. E03, E05, E06 |
| D2-03 — Lease active only in authorized projected stage and nonterminal run; unit, whole-run and transitive dependency rewind invalidate old issuers without revival. | MET | Unit and transitive dependency corrections/route/plan rewinds refuse old leases. Whole-run Grounding/Acceptance rewinds refuse for empty/self/unrelated unit tags. Actual legal rework and fresh-issuer controls pass; all terminal outcomes refuse. E03 |
| D2-04 — Wall time uses human approval timestamp and [approval,expiry) interval; counters are nonnegative and strictly below ceilings. | MET | Real Apply accepts a two-minute-old approval within the approval freshness window, but a one-minute lease refuses. Before/equal expiry and negative/zero/below/equal counters behave as required. E03 |
| D2-05 — Current effective policy equals plan-bound policy; bound approval inputs including active waiver generation cannot become stale or inactive. | PARTIAL | Policy changes refuse. Current issuer waiver references must be live and unique, including withdrawal/expiry/regrant controls. The retained route approval receives no corresponding validity check. E03, E05, E06 |
| D2-06 — Pure derivation fails closed for absent, malformed or legacy plan; existing transitions, Gate B/C, legacy bytes remain valid; D3–D6 not claimed. | MET | Absent/malformed plans fail closed. Full original test/vet/race passes on the frozen source copy. Code describes D3 consumption derivation as future work; no launch/credit/closure completeness is inferred. E02, E03, E07 |

## Blocking finding

### F-D12R7-01 — A correction lease reuses an invalid route approval

**Severity: high. Confidence: observed. Fix status: proposed.**

Observed sequence:

1. A human approves `ROUTE_SELECTION -> IMPLEMENTATION_AUTHORIZATION` for unit A. The route approval binds live artifact-waiver grant G.
2. Approved transitions reach `INDEPENDENT_VALIDATION`. These approvals also carry G.
3. A legal human waiver operation withdraws or expires G. The replacement case withdraws G and grants H.
4. A human approves the legal `INDEPENDENT_VALIDATION -> IMPLEMENTATION_AUTHORIZATION` correction. It binds the current waiver set: empty after withdrawal/expiry, or H after replacement.
5. The host presents the exact approved plan, contract, and route projection. `DeriveLease` returns `ReasonNone`, which means the lease is active. It retains the route approval that bound G.

All three refusal assertions fail in normal and race modes. A fresh legal route approval passes as the positive control. The tests use actual `Apply` calls, `Store.Load`-validated records, `EvaluateResolution`-accepted operations, and real `host.RouteIdentity` / `host.RouteGrant` projections. The strong case does not supply a stale waiver on the current issuer.

Evidence: [current-waivers.log](current-waivers.log), [current-waivers-race.log](current-waivers-race.log), [independent test](snapshot/internal/transition/auditor_d12r7_test.go), and [complete withdrawal replay input](reproductions/current-waivers/current-waivers-withdraw.json). The replay packet contains the actual validated records and document bytes. The `current-waivers/` packets reflect the final race run.

Observed source: [lease.go:157](snapshot/internal/engine/lease.go#L157) checks waiver validity only for the latest lease issuer. [lease.go:194](snapshot/internal/engine/lease.go#L194) compares the selected route reference with a digest from [routeApprovalFor](snapshot/internal/engine/lease.go#L298). That helper does not check the retained approval for invalidation. Architecture §5.4 invalidates approvals that bind the changed waiver. Architecture §6 invalidates an approval when an actually bound field changes. Gate A v2 §2.2 leaves route identity authority with the core route approval.

**Diagnosis:** a correction lease depends on two approvals: the current implementation issuer and the retained route approval. The code validates waiver history for only the first. A new implementation approval does not revive the invalidated route approval.

**Preferred fix:** retain the selected route approval record index. Check its validity through the full current history before activating the lease. Reuse the waiver-validity logic already applied to the current issuer. Refuse either invalid approval. Do not search backward for another usable route approval. Resume only after a fresh legal route-selection approval. This is a proposed fix; the auditor applied no repair.

**Verification:** run `TestAuditorD12CurrentWaiversDoNotRestoreOldRoute` with and without `-race`. Withdrawal, expiry, and replacement must refuse. `fresh_route_control` must remain active. Retain the live, unrelated, unbound-route, direct-issuer, selected-route mismatch, rewind, and legacy-byte controls. Then run the full test, vet, and race checks and obtain a fresh independent audit.

**Trade-offs:** the retained-approval check must see records after the correction issuer too. Otherwise later withdrawal remains invisible. Unbound and unrelated waiver scopes must remain usable. A second history replay has a small cost. Existing correction leases with invalid retained routes must stop for human route reapproval.

**Alternative containment:** require new route approval for every correction whose retained route approval bound any waiver. This refuses some valid corrections. Copying old waiver references to the new issuer is not a fix; those references can already be invalid. Effort: small. Priority: gate-blocking.

## Prior finding closure

Each status below answers the supplied closure criteria. It does not assert complete route-approval safety. The new composition defect remains F-D12R7-01. The original prior audit overlays were not opened or reused. Independent equivalents were written.

| Prior finding | Status | Evidence |
| --- | --- | --- |
| F01 | FIXED | Independent out-of-contract route, permission and budget tests refuse; matching route passes. |
| F02 | FIXED | Direct/transitive correction, route and plan rewind tests refuse; unrelated and new-issuer controls pass. |
| F03 | FIXED | Actual Apply late approval and [approval,expiry) boundaries pass. |
| F04 | FIXED | Contract/plan malformed UTF-8 and surrogate controls refuse; valid Unicode and material digests pass. |
| F05 | FIXED | Every unit/budget declaration omission/null refuses parsing and derivation; complete positive fixture passes. |
| F07 | FIXED | Current exact-ID comparison and independent drift/disposition controls pass. Missing-ID cardinality is checked in sameIDs and existing acceptance tests. |
| F06 | FIXED | Actual Apply presented-unbound/wrong-digest refuse; exact and both-absent legacy controls pass. |
| F08 | FIXED | Independent reference digest and independent transition/unit/plan identity mutations pass. |
| F09 | FIXED | Each negative counter refuses; zero/below pass and equality refuses. |
| D12R2-01 | FIXED | All serialized document fields are exhaustively removed/nullified. Missing/null zero-retry declarations refuse, including DeriveLease. Empty argv/non-goals and zero retries pass. Full regressions and literal legacy bytes pass. |
| F-D12R3-01 | FIXED | Actual Apply whole-run Grounding and legal Release -> Acceptance rewinds refuse for all three tags. Progress does not revive the old issuer; a new legal issuer succeeds. Dependency/unrelated controls pass in normal/race. |
| D12R4-01 | FIXED | The supplied identity/permission/approval-reference/unit/missing-projection and legal-correction closure checks pass. This is a narrow closure; route-approval validity is separately blocked by F-D12R7-01. |
| D12R5-01 | FIXED | The supplied checks for waiver changes to the lease issuer pass, with unrelated/unbound/new-issuer/multiple/malformed controls. The distinct retained-route-approval case is F-D12R7-01. |
| D12R6-01 | FIXED | Actual Apply issuer cases withdrawn/expired/old-after-regrant refuse at derivation; current-after-regrant passes. Direct issuer live/duplicate/missing/after-change controls and full regressions pass. |

## Verification and reproducibility

Environment: Go `go1.27.1 linux/amd64`; Git `2.43.0`. `git` resolves to `/usr/bin/git`. `GOCACHE`, `TMPDIR`, and `GOTMPDIR` are under this output directory. `TMPDIR` and `GOTMPDIR` are equal. No dependency installation or nested agent/model call was used.

| Check | Result | Raw evidence |
| --- | --- | --- |
| `go test -count=1 ./...` on manifest-matched source without auditor tests | Exit 0; 12 packages pass | [regression-test.log](regression-test.log) |
| `go vet ./...` on the same source | Exit 0 | [regression-vet.log](regression-vet.log) |
| `go test -race -count=1 ./...` on the same source | Exit 0; 12 packages pass | [regression-race.log](regression-race.log) |
| Independent normal and race suites | Only the retained-route refusal assertions fail | [final-normal.log](final-normal.log), [final-race.log](final-race.log) |
| Strong reproduction with truthful current waiver bindings, normal and race | Three refusal assertions fail; fresh-route control passes | [current-waivers-checks.json](current-waivers-checks.json) |
| Incomplete plan derivation, digest identity, and literal legacy-byte checks | Pass in normal and race modes | [independent-extra-checks.json](independent-extra-checks.json) |
| Guard-removal negative controls: issuer waiver, Unicode surrogate, contract binding | Each independently written test detects its removed guard; expected exit 1 | [negative-controls.json](negative-controls.json) |
| Final target freeze comparison | All unchanged | [freeze-verification.json](freeze-verification.json) |

The test copies retain exact audited production bytes. [snapshot-manifest.json](snapshot-manifest.json) identifies them. Auditor tests exist only in `snapshot/`; `regression/` contains no auditor tests. Exact commands and exit codes are in the `*-checks.json` files. Overlay files and full logs are retained.

To reproduce the strong failure:

```bash
cd /tmp/hma-gate-d-20261017-d12r7/snapshot
export PATH="$HOME/.local/go/bin:/usr/bin:/bin"
export GOCACHE=/tmp/hma-gate-d-20261017-d12r7/gocache
export TMPDIR=/tmp/hma-gate-d-20261017-d12r7/tmp
export GOTMPDIR="$TMPDIR"
go test -count=1 -v ./internal/transition -run TestAuditorD12CurrentWaiversDoNotRestoreOldRoute
go test -race -count=1 -v ./internal/transition -run TestAuditorD12CurrentWaiversDoNotRestoreOldRoute
```

Current expected result: exit 1 for both commands, with the three blocked cases returning an active lease. After a valid repair, both commands must exit 0.

## Controlled unblinding and scope

After PHASE1 SHA-256 `477e7d292fea64f470618eaa20cdf5f492d95bc1b9f1ef6761b23c50f45ef61f` was recorded, the auditor read only the previously withheld implementation handoff `docs/handoffs/2026-10-17-gate-d-d1-d2.md`. It explicitly says D2 is pure and not wired to commands. It assigns D3–D6 to later parts. It makes no claim that this part provides reservations, launch reconciliation, criterion credit, findings disposition, or the architecture amendment. Its issuer-only waiver description agrees with the inspected code. No finding or verdict changed. No prior audit directory was read.

## Limits and remediation path

- Isolation is fresh model context with the user-authorized objective and stripped closure criteria. No implementation chat, prior audit reports, or prohibited temporary directories were read. The auditor cannot independently attest the model/session identity supplied by the dispatch prompt.
- Execution uses source copies and temporary fixture repositories on the same trusted host. This is not a separate VM or an OS-enforced read-only mount. The host sandbox permits repository writes; the auditor performed none. Test fixture Git writes occur only beneath the output directory.
- The dirty/untracked implementation is not an immutable commit. Freeze hashes plus the source manifest define the audited target; final equality verification is required. Excluded .serena and architecture-amendment files are fingerprinted only, not assessed.
- Some setup helpers from existing model and transition tests are reused. Engine fixtures, requirement reconstruction, adversarial mutations, expected results, literal byte reference and new failure assertions are auditor-authored.
- D3–D6 reservations, reconciliation, criterion credit, findings disposition and architecture amendment are outside this verdict. No command-launch integration or complete Gate D safety is claimed.
- Jev is absent from the exposed host tool inventory. Auditor judgment is the recorded fallback.
- The initial independent run over-specified the refusal code after Release -> Acceptance. The oracle was corrected to require refusal; this was an auditor-test error, not a product defect. Earlier logs are retained.
- The audit is finite testing and source review, not a proof. No fix was applied or validated.
- Some initial combined source listings were truncated by the tool display. Relevant code was inspected in smaller spans. Full executed test logs, request packets, manifests, and source copies remain on disk without that display truncation.

No separate improvement project is proposed. The smallest useful change is to validate both authority dependencies through the same approval-validity logic. First contain use of correction leases through invalid routes. Then implement the targeted check. Keep the negative controls and obtain a fresh independent re-test. This audit grants no permission or stage transition.

## Required next action

**Return D1+D2 to the human gate owner for bounded remediation of F-D12R7-01.**

Current outcome: FAIL, supported by the normal and race reproductions above. Owner: the human gate owner; the approved implementer receives the bounded repair only after that decision. No model execution is needed for this review decision. The orchestrator must select and preflight an eligible exact route before repair dispatch; this audit does not invent its provider, model, effort, or access service.

Permission envelope: review these artifacts only. Exact approval required: authorize the retained-route validity repair and its regression tests. The repair then requires a fresh independent audit. Stop progression to D3 on this packet. Escalate a proposed product-contract change or wider permission request to the human.
