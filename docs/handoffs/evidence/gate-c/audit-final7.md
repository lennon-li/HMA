# HMA Gate C independent code and outcome audit

**Verdict: FAIL.** One high-severity finding remains open.

Target: `main` at `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree.
Tracked diff SHA-256: `86e454008c29c90537b8834918baca65db165cbabca7f00bcb0c9a95f35c7e7f`.
Every untracked file has a hash in `FREEZE.json`. The final five freeze checks all match.

**Observed:** A review of a different base can clear the current unit blocker. This also works with a different repository and base. The ordinary product tests pass. The independent negative control fails.

**Interpretation:** Packet and receipt consistency is insufficient when packet generation selects a base that differs from the core approval. Gate C must return for repair. This report does not establish any stage advance or closure.

Isolation: **isolated review context**, with operationally read-only repository access. Tests ran in `/tmp/hma-gate-c-20261017-final7/snapshot`. This is not an OS-enforced read-only mount. The report uses the independent-audit skill. The host identity requested by the dispatch is Liz / OpenAI / gpt-6.1-sol / high. No nested agent, model API, or worker dispatch was used.

## Scope and evidence freeze

The original objective is the consolidation checkpoint in the outcome handoff §5 and its deliverables/proof requirements in §6. The permitted Gate A sections define design details. Architecture, AGENTS and task14 remain normative. Lennon’s 2026-10-17 acceptance of scope §0.1 controls the Gate D boundary.

Requirements were derived before the diff. The initial requirement inventory is `REQUIREMENTS-before-diff.json`. Scope contents were displayed only with `sed -n '1,86p'`. No optional unblinding occurred. TODO Gate C content and previous review outputs were not read. Excluded files were hashed for the freeze but were not audited.

Phase 1 is `PHASE1.json`, SHA-256 `e9df472bcd7d1ceb978af7ff59c398d01004cfe75bbbfc993b097e402d073d95`. It was validated with the external report validator. It was not overwritten. `AUDIT.json` uses the report-contract value `fix-required` for the single user-facing verdict **FAIL**.

## Requirement coverage

MET means satisfactory scoped evidence. PARTIAL means implemented protection has a demonstrated gap. Deferred behavior is not counted as implemented success.

| ID | In-scope requirement | Status | Evidence |
|---|---|---|---|
| C1.1 | HMA generates and records fixed packet, random 128-bit nonce, dispatch ID, packet digest and launch digests. | MET | E03, E09, E13 |
| C1.2 | Packet target must correspond to the run and its approved repository/base; wrong supplied target must refuse. | PARTIAL | E08, E10 |
| C1.3 | Packet carries available core head, accepted plan/route, projected criteria, unit/unowned findings and waivers; input/core drift refuses. | MET | E03, E12, E13 |
| C1.4 | Dispatch checks current implementation, read-only review route, fresh preflight and independence across unit contribution history; final reason enforces family separation. | MET | E03, E13 |
| C1.5 | Only review-packet creates Packet provenance; manual dispatch cannot impersonate generation. | MET | E03, E06, E13 |
| C2.1 | Receipt names exact run, completed review mode, zero exit, packet/launch digests, root and worker/provider/model/effort. | MET | E03, E05, E09 |
| C2.2 | Report bytes match receipt entry and packet digest/dispatch/nonce; missing, rewritten and other reports cannot import. | MET | E03, E05, E06, E09 |
| C2.3 | Review imports only once for current implementation/latest packet and exact core head; pending newer packet closes old gate/snapshot. | MET | E03, E05, E06, E12 |
| C2.4 | Live packet target is unchanged at report and findings import; tracked/untracked drift refuses. | MET | E03, E05, E09 |
| C2.5 | Reviewer launch is inside fixed one-hour half-open validity interval; rewind/current-core drift refuses. | MET | E03, E05, E09 |
| C2.6 | Report contract, Unicode whitespace, exact case-sensitive keys and UTF-8 agree with independent-audit interface; malformed/inconsistent pass cannot approve. | MET | E03, E05, E06, E09 |
| C2.7 | Verdict mapping is total and conservative; only valid pass approves; final approval requires isolated context. | MET | E03, E05, E09 |
| C3.1 | Additive snapshot flag replaces findings even when empty; no-flag behavior and old digests remain compatible. | MET | E03, E06, E07, E12 |
| C3.2 | Non-approving imports preserve all prior BLOCK/WAIVABLE findings and same-ID strength; only authentic current approval can resolve. | PARTIAL | E03, E05, E06, E08, E12 |
| C3.3 | Unit ownership, slash/prefix cases and legacy finding migration preserve unrelated/unowned findings and waiver references. | MET | E03, E05, E06, E07, E12 |
| C3.4 | Severity and unsupported kernel/budget labels do not create blockers; imported findings remain ADVISORY until Gate D registry/budgets. | MET | E03, E05, E12 |
| C3.5 | Review import cannot satisfy host Verification gate before findings snapshot; other core records invalidate the review. | MET | E03, E05, E06, E12 |
| C4.1 | Verify-loop state binds canonical envelope hash, HMA run/unit/store, hook and stop policy; changed journal/association refuses. | MET | E03, E05, E06, E11 |
| C4.2 | HMA independently reruns targeted argv without shell; unsafe shell/script/wrapper entry points refuse while plain data works. | MET | E03, E06, E11 |
| C4.3 | Missing/terminal runs refuse before execution; host and core locks exclude terminal/core appends throughout recapture and append. | MET | E03, E05, E11 |
| C4.4 | Recapture preserves actual exit/output, framed digest, truncation and durable immutable output reference; mutation/corrupt residue refuses evidence append. | MET | E03, E05, E06, E11 |
| C4.5 | No import credits criteria, grants reservations/leases, changes core stage or manufactures verified success. | MET | E03, E04, E05, E06, E11, E12 |
| C5.1 | Failure-to-linked-remediation-to-fresh-review fixtures and missing/self/stale review controls execute. | MET | E03, E04, E05, E06 |
| C5.2 | Legacy runs remain readable and visibly labelled; migration refuses later manual review overrides. | MET | E03, E06, E07, E13 |
| C5.3 | Gate B approval, stage, terminal, chain integrity, host concurrency and artifact protections retain passing regression evidence. | MET | E03, E04, E05, E06 |
| BOUNDARY | All human-deferred requirements remain recorded with Gate D exit tests/prerequisites; Gate C makes no bounded-execution, positive-credit or closure claim from them. | MET | E02, E03, E04, E11, E12, E13 |

Totals: **24 MET; 2 PARTIAL; 0 NOT MET**. The two partial rows concern C-F24.

## C-F24 — Review with another base resolves current findings

**High severity. Observed. OPEN.** Related prior finding: F04.

review-packet accepts caller-selected repository_root and base_revision, and derives artifact_digest when the caller omits it. corePacketState checks active route and accepted plan, but never compares this base with the core approval BaseRevisionDigest or establishes that the root belongs to that approved repository. A nonce-bound receipt/report then proves consistency with this wrongly selected packet. findings-import treats that APPROVE as sufficient to replace the current unit findings. Two independent reproductions clear A-only-blocker: base changed to HEAD in the same root, and another root/base. No receipt rewriting, same-UID adversary, core tampering or manual provenance override is needed after the synthetic approved fixture is created through transition.Apply. This violates architecture §§6 and 7.1 and leaves prior F04 only partly closed. The reproduction demonstrates blocker removal and host APPROVE, not an authentic human stage transition or verified closure.

| Independent case | Observed behavior | Expected behavior |
|---|---|---|
| Exact approved base and root | APPROVE and own blocker resolution | Accept |
| Same root; base changed to produced HEAD | APPROVE and empty snapshot clear blocker | Refuse before any authority append |
| Different root and base | APPROVE and empty snapshot clear blocker | Refuse before any authority append |
| Explicit A artifact against B | HOST_REVIEW_ARTIFACT_MISMATCH | Refuse |

The decisive reproduction uses four ordinary `transition.Apply` edges. All approvals are labelled `SYNTHETIC-AUDIT-FIXTURE (not human approval)`. It then uses normal host decision, preflight, implementation dispatch, review-packet, review-import and findings-import paths. No audited source is changed.

The same-root reproduction has core approved base `10db4b7ce8ce827346b7154577f5df433f776e3a` and reviewed base `9a67b66cd64fcbf5dfd59a30375969850ed017cd`. The different-repository reproduction has core approved base `709a82c423c5b66d813e3f5a42a8bd4b741c6cfb` and reviewed base `8d26faccfe597c262277cccf32342500593ca52e`.

Source: `cmd/hma/review.go:86`, `:187`, `:192`, `:224`, and `:477`. The packet derives its target from the supplied root/base. A caller may omit `artifact_digest`. The core packet checks the plan and route but omits the approved-base comparison. Later checks compare the repository against this packet.

Raw evidence: `repository-confirmation.log`. Retained exact packet, report, receipt and core/host chains are under `reproductions/current-base-control/`, `reproductions/different-base-same-root/`, and `reproductions/different-repository/`. Historical paths in receipts still name the original temporary fixtures. The retained report copies preserve the original bytes and digests.

Reproduce from the external snapshot:

```bash
PATH="$HOME/.local/go/bin:$PATH" \
GOCACHE=/tmp/hma-gate-c-20261017-final7/gocache \
TMPDIR=/tmp/hma-gate-c-20261017-final7/runtime \
go test -race -count=1 -v -run '^TestAuditor7OtherRepository' ./cmd/hma
```

Expected result after repair: all four cases pass their assertions. Current result: two negative cases fail and both controls pass.

**Preferred repair — proposed:** Before writing a usable packet or dispatch, resolve the unit's active core approval and require packet BaseRevision to equal its BaseRevisionDigest. Bind the canonical repository root/identity to the run from authoritative grounding rather than from this review request; refuse findings resolution if this association is unavailable or differs. Carry these bindings in the fixed packet and recheck them under the existing locks at dispatch, review-import and findings-import. Do not let omitted artifact_digest substitute a new approved repository/base. Preserve the legacy compatibility label and require explicit migration of records lacking the authoritative association.

**Why:** Nonce, receipt and live-target equality currently establish only packet self-consistency. They do not establish that the packet reviews the repository/base whose findings it may resolve. The existing approved base is already available without the deferred Gate D acceptance contract.

**Trade-offs:** This may refuse old packet-generation callers that selected another base or lack a recorded authoritative root. Migration must preserve old chain bytes and all findings/waivers. The smallest containment is approved-base comparison; that alone cannot prove repository identity when unrelated roots share a base commit.

**Alternative:** Require a separately authoritative target association before enabling findings resolution, and keep all current blockers while this binding is missing. A mandatory caller artifact digest provides additional drift detection but is insufficient by itself if the same caller can choose both root and digest.

**Effort and sequence:** Medium. Apply approved-base containment first. Establish the authoritative repository association. Independently rerun the two failing negative controls and the current-target controls. Then run the ordinary Gate B/C checks. No proposed fix was implemented or validated by this audit.

## Prior-finding closure

| ID | Status | Independent assessment | Evidence |
|---|---|---|---|
| F01 | FIXED | Same-ID strength/omission controls pass for non-approving verdicts. Authentic-current authorization still has the separate F04/C-F24 target-binding gap. | E03, E05, E12 |
| F02 | FIXED | Core findings/criteria/waivers after packet or review invalidate import. Own snapshot is the narrow gate exception. | E03, E06, E12 |
| F03 | FIXED | Old answered/superseded report cannot be replayed; two concurrent imports yield exactly one append. | E03, E05, E06 |
| F04 | PARTIAL | Caller-supplied wrong artifact and receipt root drift refuse. Omitting artifact_digest permits a different core-approved base or another repository/base and its report clears the blocker; see C-F24. | E03, E08, E10 |
| F05 | FIXED | Final reason participates in strong family rule even without FinalValidation. | E03, E13 |
| F06 | FIXED | Missing and wrong provider/effort, agent/model and root all refuse without chain append. | E03, E05, E09 |
| F07 | FIXED | Report-contract failures refuse/escalate and preserve blockers. Full Python validator and parser controls pass. | E03, E05, E06, E09 |
| F08 | DEFERRED-BY-HUMAN | Negative label-only controls are fixed. Real kernel reproduction and measured budget positive paths remain Gate D with exit tests; no Gate C authority relies on them. | E02, E03, E05, E12 |
| F09 | FIXED | State bridge re-captures actual pass/fail checks, retains bindings and credits no criterion. Failure/remediation/review fixture passes. | E03, E05, E11 |
| F10 | FIXED | Review launch interval boundaries refuse/accept correctly; core drift invalidates. Reservation-generation reconciliation remains explicitly deferred. | E02, E03, E05 |
| F11 | FIXED | Code includes all recorded unit contributions; multi-route and independent route tests pass. This is not a claim about Gate D unrecorded launches. | E03, E13 |
| F12 | DEFERRED-BY-HUMAN | Reservation and reconciliation matrix is preserved in scope §0.1 and must pass Gate D before execution/closure authority. | E02, E11 |
| F13 | FIXED | First generated packet switches entire run to packet_import and refuses manual override; legacy_manual status remains explicit. | E03, E06, E13 |
| A01 | FIXED | Contract-invalid report tests preserve blocker and keep host gate closed. | E03, E05, E06, E09 |
| A02 | FIXED | Tracked/untracked live target changes refuse both imports; unchanged control works. This does not establish initial target-to-core binding (C-F24). | E03, E05 |
| A03 | FIXED | Ownership uses UnitID rather than a slash prefix; other-unit/subunit and unowned blockers remain. | E03, E05, E12 |
| A04 | DEFERRED-BY-HUMAN | Contract-dependent full packet inputs are explicitly moved to Gate D. In-scope current core head/plan/route/findings/waivers are present and core drift checks remain. | E02, E03, E12, E13 |
| A05 | FIXED | Baseline-written record loads byte-identically, own finding can resolve, active waiver remains, and ambiguous source owner stays unowned. | E06, E07, E12 |
| A06 | DEFERRED-BY-HUMAN | Real-rule/budget classification, criterion recheck association and independent positive credit require Gate D contract; named exit tests preserved. | E02, E11, E12 |
| A07 | DEFERRED-BY-HUMAN | In-scope canonical hash/store/hook/stop controls pass. Full namespace/reservation/generation reconciliation remains Gate D. | E02, E03, E11 |
| A08 | FIXED | Missing/closed runs refuse before marker execution; open check independently probes and observes both locks held. | E03, E05, E11 |
| A09 | FIXED | Legacy and migrated enforcement labels are explicit; migrated manual overrides refuse. | E03, E06, E13 |
| R3-01 | FIXED | New unanswered packet prevents older snapshot and approval; current report/snapshot control restores gate. Host lock serializes packet/snapshot/gate and concurrent replay tests pass under race. | E03, E05, E06, E12 |
| R3-02 | DEFERRED-BY-HUMAN | Full acceptance-contract packet inputs move to Gate D by accepted §0.1. Existing core-state packet fields do not claim standalone final closure sufficiency. | E02, E13 |
| R3-03 | DEFERRED-BY-HUMAN | Human approval now explicitly resolves the boundary-authority question. Original design rows remain in Gate D with exit tests and prerequisites. No Gate D behavior is claimed passed here. | E02, E11, E12, E13 |
| C-F14 | FIXED | Manual Packet refuses before append in legacy and packet modes; manual APPROVE after failed import refuses. | E03, E06, E13 |
| C-F15 | FIXED | Fresh-token stdout/stderr, exit 9, framed digest, durable reference and over-cap truncation verified after command exit. | E03, E05, E11 |
| C-F16 | FIXED | ASCII and Unicode folded variants cannot override exact fields; receipts and nested JSON use same exact-key decoder. | E03, E05, E09 |
| C-F17 | FIXED | U+001C..U+001F report rejection preserves blocker; full code-point whitespace parity test passes. | E03, E05, E09 |
| C-F18 | FIXED | Empty, partial and symlink output residues refuse without changing chains/residue; intact artifact reuse succeeds. Publication uses exclusive link after sync. | E03, E06, E11 |
| C-F19 | FIXED | Shell, wrappers, forbidden git push/curl/rm argv refuse before execution/appends; direct Capture shell controls refuse. | E03, E06, E11 |
| C-F20 | FIXED | Invalid UTF-8 report, receipt and loop state refuse and preserve blocker/chains. | E03, E06, E09 |
| C-F21 | FIXED | printf %s sh runs; shell entry points refuse. | E03, E06, E11 |
| C-F22 | FIXED | Shell shebang, env shebang, symlink shell interpreter and wrapper aliases refuse in loop and direct Capture. Residual compiled-program behavior is disclosed in code and limitations. | E03, E06, E11 |
| C-F23 | FIXED | Independent check nonblockingly probes core and host lock files; both are held. Core Transact spans admission, execution and append. | E03, E05, E11 |

Totals: **27 FIXED; 1 PARTIAL; 0 OPEN; 7 DEFERRED-BY-HUMAN** among the 35 supplied prior findings. C-F24 is a new OPEN finding. A deferred status applies only to the accepted Gate D portion. The notes retain the result of each in-scope negative control.

## Verification

| Command group | Exit | Observation |
|---|---|---|
| `go test -count=1 ./...` | 1 | Initial default TMPDIR; one scratch-directory assertion fails in /tmp source copy. Log: `go-test.log`. |
| `go test -race -count=1 ./...` | 1 | Same initial environment assertion. Log: `go-race.log`. |
| `go test -count=1 ./...` | 0 | TMPDIR under output/runtime; 12 packages pass. Log: `go-test-isolated.log`. |
| `go vet ./...` | 0 | No diagnostics. Log: `go-vet.log`. |
| `go test -race -count=1 ./...` | 0 | TMPDIR under output/runtime; 12 packages pass. Log: `go-race-isolated.log`. |
| `scripts/dogfood/host-dogfood.sh --keep` | 0 | 17 checks pass; TMPDIR under audit output. Log: `dogfood.log`. |
| `go test -race -count=1 -v -run '^TestAuditor7' ./cmd/hma` | 0 | Executed before the extra/repository test files were added; 7 functions, 51 named subtests pass. Log: `auditor-tests.log`. |
| `go test -race -count=1 -v -run '^TestAuditor7(Baseline|Concurrent|Shell|Artifact|Invalid|Core|Manual)' ./cmd/hma` | 0 | 7 additional functions, 25 named subtests pass. Log: `auditor-extra.log`. |
| `go test -count=1 -v -run '^TestAuditor7OtherRepository' ./cmd/hma` | 1 | Initial different-repository negative control fails. Log: `repository-probe.log`. |
| `go test -race -count=1 -v -run '^TestAuditor7OtherRepository' ./cmd/hma` | 1 | Two failing negatives and two passing controls. Log: `repository-controls.log`. |
| `go test -race -count=1 -v -run '^TestAuditor7OtherRepository' ./cmd/hma` | 1 | Confirmed with four ordinary transition.Apply edges and retained core/host/report/receipt evidence. Log: `repository-confirmation.log`. |
| `go test -count=1 -v -run 'TestGateCR3ReportContractParity|TestGateCR6StripMatchesPython|TestGateCR3CanonicalHashMatchesPython|TestLaunchDigestsMatchLedger' ./internal/host ./internal/repostate` | 0 | Verbose rerun confirms the external interface tests execute, not skip. Log: `interface-parity.log`. |

All commands, working directories and environment adjustments are recorded in `COMMANDS.json`. Tool versions: Go 1.27.1, Git 2.43.0, Python 3.12.3. Go resolves from `~/.local/go/bin`; Git resolves to `/usr/bin/git`. GOCACHE is beneath this audit output.

The 14 independent passing test functions include 76 passing named subtests. The final repository/base reproduction adds 2 passing controls and 2 failing negative cases under `-race`. Existing product tests and vet pass. These facts do not override the failing independent test.

The initial full test and race runs failed one scratch-directory assertion because the source copy is under default `/tmp`. The source copy was outside the dedicated `TMPDIR` on the passing runs. Product code was unchanged. Both failure logs remain available.

## Evidence ledger

| ID | Artifact | Observation |
|---|---|---|
| E01 | FREEZE.json; FREEZE-check-phase1.json | main at 527b7c5ff79090c558f5bbfed5ec87a85f06d1e5; tracked diff SHA-256 86e454008c29c90537b8834918baca65db165cbabca7f00bcb0c9a95f35c7e7f; every untracked file hashed; all five freeze comparisons match. |
| E02 | REQUIREMENTS-before-diff.json; docs/handoffs/2026-10-16-gate-c-scope.md:1-86 | Requirements derived before diff. Scope read only through sed lines 1-86. Human acceptance moves the named requirements unchanged to Gate D with exit tests. |
| E03 | go-test-isolated.log; go-vet.log; go-race-isolated.log | go test -count=1 ./..., go vet ./..., and go test -race -count=1 ./... exit 0 in external copy. No product files edited. |
| E04 | dogfood.log; hma-host-dogfood.fy7uYV/ | scripts/dogfood/host-dogfood.sh --keep exits 0, PASS 17 checks. Real local binaries; synthetic host telemetry and approval fixtures only. Missing human approval still refuses core transition. |
| E05 | auditor-tests.log; snapshot/cmd/hma/auditor_final7_test.go | Seven independent test functions pass under -race, with 51 passing named subtests: invalid reports, receipt drift, half-open timing, latest packet/replay, tracked/untracked live drift, finding strength/ownership, missing/closed/open admission, both locks held inside check, retained output and truncation. |
| E06 | auditor-extra.log; snapshot/cmd/hma/auditor_final7_extra_test.go | Seven additional independent functions pass under -race, with 25 passing named subtests: baseline compatibility, concurrent single-use import, shell entry points/direct Capture, artifact residue, invalid UTF-8, stale core findings/criteria/waivers, and manual overrides. |
| E07 | baseline-writer.log; baseline-writer/; baseline-records/ | Baseline model/store source extracted with /usr/bin/git show from HEAD. Baseline executable writes legacy findings and active waiver. Current loader preserves bytes/head, imports own finding resolution, preserves waiver, and keeps ambiguous findings. Baseline head 18ac3ca184256c8d5060beaf58983322271aa3df68434954316a0db370e4a9ab. |
| E08 | repository-confirmation.log; snapshot/cmd/hma/auditor_final7_repository_test.go; reproductions/ | Independent -race reproduction has two failures and two valid controls. A same-root changed base and a different repository/base each obtain APPROVE and an empty finding snapshot that removes A-only-blocker. Current approved-base control succeeds; explicitly supplying A artifact against B refuses. Core chain was generated through four ordinary transition.Apply edges with clearly labelled synthetic approvals. |
| E09 | internal/host/review_import.go:117; internal/host/exact_json.go:26; internal/host/report_contract.go:20; internal/host/gatec_r3_parity_test.go | Reviewed receipt/nonce/current-dispatch checks, exact JSON keys and UTF-8 enforcement, Python report-validation port. Existing tests include validator parity and exhaustive Python whitespace comparison over 0x110000 code points; all ran in the passing suite with external scripts available. |
| E10 | cmd/hma/review.go:86; cmd/hma/review.go:187; cmd/hma/review.go:224; cmd/hma/review.go:477 | Packet computes target from caller-selected root/base and allows omitted artifact_digest. corePacketState checks route/plan but does not compare packet base to active core approval base. Import rechecks live state against that packet, not against core approved repository/base. |
| E11 | cmd/hma/loop.go:116; internal/store/jsonl.go:251; internal/evidence/noshell.go:67; cmd/hma/loop.go:299 | Loop holds host then core lock through terminal/route checks, capture, and appends. Outputs use temporary-file sync and exclusive hard-link publication; corrupt/symlink residue refuses. Entry points, wrappers, and shell interpreters are checked. |
| E12 | internal/host/findings_import.go:68; internal/host/findings_import.go:101; internal/engine/resolution.go:47; cmd/hma/transition.go:85 | Nonpass snapshots preserve prior blocking strength. Explicit unit ownership preserves other units and unowned findings. Snapshot flag is additive. Host gate requires the imported snapshot and latest dispatch. Old no-flag projection remains unchanged. |
| E13 | internal/host/host.go:289; internal/host/host.go:771; cmd/hma/host.go:240; cmd/hma/host.go:267; docs/task14-host-orchestration.md | Run migration is one-way to packet_import; manual Packet/provenance and manual approvals refuse. Final reason enforces family separation. All implementation contributions enter independence checks. Legacy mode is exposed. |
| E14 | go-test.log; go-race.log; scripts/dogfood/hostinput/main_test.go:66 | First full test and race runs exit 1 only because the source copy is beneath default /tmp and a scratch-directory test expects source to be outside os.TempDir. Repeating unchanged code with TMPDIR=output/runtime passes. This is a test-environment adjustment, not a product fix. |
| E15 | interface-parity.log | Post-freeze verbose confirmation of canonical JSON hash, external report validator, exhaustive whitespace and real agent-run-ledger launch-digest parity. Four tests pass without skips. No assessment changed. |

## Gate D boundary

Scope §0.1 retains the reservation replay/admission/consumption rules, final-slot eligibility, namespace reconciliation, half-open launch windows, monotone generations, D5 hook replay, independent positive criterion credit, and reproduced kernel/budget disposition. It lists their exit tests. The accepted paragraph also moves the full contract-dependent fixed packet inputs. Gate C imports neither criterion PASSED nor execution authority. The report makes no claim that Gate D or closure has passed.

C-F24 concerns the existing core-approved base and current finding resolution. It does not require the deferred acceptance contract, lease or reservation layer.

## Remediation roadmap and opportunities

1. Return the C-F24 unit for human-authorized repair. Preserve current findings and exact evidence.
2. After the repair, an independent read-only auditor must rerun the retained negative tests against the exact repaired target.
3. Require passing scoped regression checks before another Gate C acceptance decision.

No unrelated redesign or optional implementation is recommended. Strengthening test coverage at the approved-base boundary is part of the C-F24 repair.

## Limitations and isolation

- This is a fresh audit conversation. No implementer conversation or previous Gate C verdict was supplied or read. Permitted design files/tests contain author commentary and Gate A history; these were treated as artifacts, not as proof. Session isolation is observed locally, not a cryptographic attestation of upstream dispatch.
- No optional unblinding occurred. TODO.md contents, scope sections after line 86, prior audit output directories, .serena and Gate D architecture-amendment files were not inspected. Excluded untracked files were hashed only for the requested freeze.
- Repository access was operationally read-only and all tests/negative fixtures ran in the external copy. The host workspace itself permits writes; this is not an OS-enforced read-only mount or a separate machine. Tests create and commit only disposable Git fixture repositories.
- Genuine worker/model execution and model APIs were forbidden and not used. Receipts are synthetic fixtures shaped against the real ledger interface. Dogfood is explicitly synthetic; no claim of Gate F operational completion is made.
- Full Gate D reservation/namespace/generation/lease reconciliation, positive criterion credit, kernel reproduction/budget disposition and full acceptance-contract packet inputs remain human-deferred. Their implementation was not audited as Gate C success.
- No-shell checks inspect entry points, symlinks, wrappers and interpreters. Arbitrary compiled programs can spawn a shell internally; this residual is disclosed in noshell.go and requires trusted approved command selection. This report does not claim arbitrary-program sandboxing.
- Initial default-TMPDIR test/race runs failed one environment-dependent scratch-directory assertion. Unchanged product code passed with a dedicated scratch TMPDIR; initial failure logs are retained.
- Finite tests and static inspection do not prove all possible interleavings. Negative failure cases are reproducible source tests with retained synthetic stores; ordinary passing fixture temporary directories are removed by Go tests.
- Jev was unavailable in the host tool inventory. The auditor made bounded audit judgments directly and grants no stage/waiver/release authority.

No post-freeze explanation changed the findings. The additional interface tests confirmed execution without skips. The final repository state matches the starting freeze. Source and external-contract hashes are in `AUDITED-SOURCES.json`. Output hashes are in `ARTIFACTS.sha256`.

## Required next action (§18)

**RETURN_FOR_REPAIR.** Lennon should authorize one bounded C-F24 repair unit. Ming is the proposed implementer. The proposed design route is Anthropic `claude-opus-5-5` through Claude Code CLI. Effort is not specified in the authoritative design and must be selected through the eligible route process. The exact-route `hi` preflight must pass before dispatch.

Permission envelope: the repository/base binding repair and its focused HMA tests only. No commit, push, publication, credential change or stage advance is authorized by this audit. The human must approve the repair unit, route and envelope. Stop for unresolved repository authority, permission or scope drift, failed preflight, repeated material failure, or failed independent retest.

**Next action: return Gate C to Lennon for authorization of the bounded C-F24 repair.**
