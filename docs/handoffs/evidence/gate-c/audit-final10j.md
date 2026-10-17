# HMA Gate C audit — final10j

**Verdict: FAIL.** Two output-contract defects remain. Both have low severity. The tested authority controls passed.

The auditor isolation is **compromised**. This session started fresh. Two Gate A v3 range reads printed forbidden §5 and the start of §7. This report does not claim isolated independent acceptance.

The repository stayed at `main`, `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree. The tracked diff SHA-256 is `2abaadf6cbdf68e314bfdbb5f95db7e2f1eb610d25f91d84fca2ed6570fefab0`. See [FREEZE-START.json](FREEZE-START.json) and [FREEZE-END.json](FREEZE-END.json). Every untracked file has a digest. `.serena/` and the two architecture-amendment handoffs were excluded from the audit.

The authoritative objective is the 2026-10-15 consolidation handoff §5 Gate C and §6. Approved design sections and normative contracts are traced in [REQUIREMENTS.json](REQUIREMENTS.json) and [AUDIT.json](AUDIT.json). The Lennon 2026-10-17 scope boundary is accepted. This audit does not reopen it.

## Evidence and limits

- The unmodified source copy passed `go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...`.
- The synthetic host dogfood passed 17 checks. Its initial copied-tree build failed at VCS stamping. The rerun used `GOFLAGS=-buildvcs=false`. Both logs remain available.
- Auditor controls checked receipt drift, invalid reports, core/live-target staleness, exact ownership, same-ID downgrades, concurrent import, pending packets, first-packet clone rejection, terminal runs, core locking, retained output, truncation, corrupt artifacts, shell entry points and executable aliases.
- A Python-contract-normalized verify-loop journal carried a failed check and repair link. HMA re-captured a passing check and left an existing criterion `PENDING`.
- The new binary loaded six host records written by baseline `527b7c5`. It reported `legacy_manual`. Baseline-written core findings and active waiver scopes also loaded without byte changes.
- No real worker, model API, provider `hi`, launch reservation, Gate D criterion credit, or verified product closure was exercised. Tests used scratch repositories and synthetic approvals. The original repository was not edited.

Raw logs and exact command results are linked below. Initial auditor fixture errors are retained. They were corrected in the audit copy. The two presentation failures are product observations.

| Check | Exit | Evidence |
|---|---:|---|
| `auditor-consolidated` | 1 | [auditor-consolidated.log](auditor-consolidated.log); [auditor-consolidated.result.json](auditor-consolidated.result.json) |
| `auditor-extra` | 0 | [auditor-extra.log](auditor-extra.log); [auditor-extra.result.json](auditor-extra.result.json) |
| `auditor-first-and-journal` | 0 | [auditor-first-and-journal.log](auditor-first-and-journal.log); [auditor-first-and-journal.result.json](auditor-first-and-journal.result.json) |
| `auditor-negative-v2` | 0 | [auditor-negative-v2.log](auditor-negative-v2.log); [auditor-negative-v2.result.json](auditor-negative-v2.result.json) |
| `auditor-negative` | 1 | [auditor-negative.log](auditor-negative.log); [auditor-negative.result.json](auditor-negative.result.json) |
| `auditor-waiver-rewind-v2` | 0 | [auditor-waiver-rewind-v2.log](auditor-waiver-rewind-v2.log); [auditor-waiver-rewind-v2.result.json](auditor-waiver-rewind-v2.result.json) |
| `auditor-waiver-rewind` | 1 | [auditor-waiver-rewind.log](auditor-waiver-rewind.log); [auditor-waiver-rewind.result.json](auditor-waiver-rewind.result.json) |
| `baseline-migration-v2` | 0 | [baseline-migration-v2.log](baseline-migration-v2.log); [baseline-migration-v2.result.json](baseline-migration-v2.result.json) |
| `baseline-migration` | 1 | [baseline-migration.log](baseline-migration.log); [baseline-migration.result.json](baseline-migration.result.json) |
| `go-race` | 0 | [go-race.log](go-race.log); [go-race.result.json](go-race.result.json) |
| `go-test` | 0 | [go-test.log](go-test.log); [go-test.result.json](go-test.result.json) |
| `go-vet` | 0 | [go-vet.log](go-vet.log); [go-vet.result.json](go-vet.result.json) |
| `host-dogfood-vcs-disabled` | 0 | [host-dogfood-vcs-disabled.log](host-dogfood-vcs-disabled.log); [host-dogfood-vcs-disabled.result.json](host-dogfood-vcs-disabled.result.json) |
| `host-dogfood` | 1 | [host-dogfood.log](host-dogfood.log); [host-dogfood.result.json](host-dogfood.result.json) |
| `interface-parity` | 0 | [interface-parity.log](interface-parity.log); [interface-parity.result.json](interface-parity.result.json) |
| `presentation-contract` | 1 | [presentation-contract.log](presentation-contract.log); [presentation-contract.result.json](presentation-contract.result.json) |

The consolidated auditor result counts each top-level test and subtest: `{"pass": 97, "fail": 3, "skip": 0, "failed_tests": ["TestAuditor10jPresentationContract/bind-repository-next-action", "TestAuditor10jPresentationContract/loop-evidence-enforcement-label", "TestAuditor10jPresentationContract"]}`.

## Requirement coverage

MET means sufficient evidence for this bounded Gate C audit. It is not a proof over all inputs. PARTIAL identifies a remaining observed contract defect.

| ID | Gate C requirement | Status | Observation / evidence |
|---|---|---|---|
| C01 | Integrate existing interfaces without replacing independent audit or duplicating verify state. Re-capture checks as evidence only. | **MET** | The bridge reads the verify-loop interface and re-captures argv. A Python-contract-normalized journal preserves two synthetic repair links. HMA records its own result and leaves an existing PENDING criterion unchanged. No second verify controller or semantic auditor was introduced. See `auditor-first-and-journal.log`, `auditor-negative-v2.log`, `cmd/hma/loop.go:87`. |
| C02 | HMA alone generates packet, nonce and dispatch binding to exact current implementation, approved repository/base and observed target. | **MET** | HMA generates a 128-bit nonce and packet digest. Manual packet injection, first-packet clone/unbound cases, wrong base/root and wrong artifact controls refuse. A repository binding is an explicit one-time host step. See `auditor-first-and-journal.log`, `auditor-extra.log`, `auditor-negative-v2.log`, `cmd/hma/review.go:159`. |
| C03 | Receipt binds run, mode, completion, route including provider/effort, packet, launch target, report hash and nonce. Failed/missing/stale/other report grants nothing. | **MET** | Provider, effort, worker, model, completion, mode, exit, packet, target, report-file and nonce checks fail closed. Invalid/missing required fields, case variants, information separators, trailing JSON and invalid UTF-8 cannot approve. The external Python validator agrees on the auditor report matrix. See `auditor-negative-v2.log`, `internal/host/review_import.go:114`, `internal/host/report_contract.go:20`. |
| C04 | Review independence covers all unit implementations; final/disputed/high require provider/model family separation and isolated final acceptance. | **MET** | The source checks all implementation contributions. Existing runtime tests exercise prior contributor and final-reason family separation. Normal and stronger isolation rules preserve the different-model/provider-family boundary. See `go-test.log`, `go-race.log`, `internal/host/gatec_r2_test.go:12`, `cmd/hma/gatec_r2_test.go:97`, `internal/host/host.go:810`. |
| C05 | Current review has short half-open validity; core changes, target drift, new pending packet and replay cannot reuse approval. Reservation generations deferred. | **MET** | Auditor probes accept the issuance instant and one nanosecond before expiry. They refuse before issuance, at expiry and after expiry. Exactly one concurrent import succeeds. A new pending packet closes the old gate and snapshot; a current new report/snapshot restores it. Core and live target drift refuse. See `auditor-negative-v2.log`, `auditor-waiver-rewind-v2.log`, `internal/host/host.go:898`. |
| C06 | Snapshots replace exact unit findings; nonapproval preserves BLOCK/WAIVABLE; empty clearing requires exact latest imported approval; no global ID collisions. | **MET** | Failed and escalating snapshots retain same-ID and omitted BLOCK/WAIVABLE findings. Fresh approval clears only the exact owner. Legacy own/unowned findings and active waiver scopes remain readable. Imported slash IDs and duplicate global IDs are refused. See `auditor-negative-v2.log`, `auditor-extra.log`, `baseline-migration-v2.log`, `internal/host/findings_import.go:74`, `cmd/hma/review.go:453`. |
| C07 | Imported unsupported citations/severity do not create blockers; imports never credit criteria. Positive mapping/credit deferred unchanged. | **MET** | Unsupported kernel/budget citations and severity alone remain ADVISORY. A pass carrying a citation escalates. Neither imported report snapshots nor loop evidence writes criterion credit. Positive reproduced disposition/credit remains Gate D. See `go-test.log`, `auditor-negative-v2.log`, `auditor-first-and-journal.log`, `internal/host/review_import.go:205`, `internal/host/findings_import.go:26`. |
| C08 | Check execution refuses terminal/missing/stale route or wrong repository before launch; host/core locks protect check through append; capture uses argv without shell/forbidden ops and retains inspectable output. | **MET** | Missing/terminal runs and wrong root/base/journal repository refuse before argv. The core lock is held inside the check. Shell/script/interpreter/wrapper entry points and forbidden executable aliases refuse. Stdout/stderr, a nonzero exit, truncation, framed digest, intact reuse and corrupt-residue refusal were reproduced. See `auditor-negative-v2.log`, `auditor-extra.log`, `cmd/hma/loop.go:125`, `internal/evidence/noshell.go:48`, `internal/store/jsonl.go:251`. |
| C09 | Existing hash chains and legacy records remain readable with no-flag semantics; migration is visible and cannot reopen manual review after packet enforcement. | **PARTIAL** | Baseline-written core and six host records remain readable. Status exposes legacy_manual; packet migration blocks later manual reviews. Two new write results break the documented human-facing result contract (J10-01 and J10-02). The binding prerequisite is documented in scope §4c, so Phase 1 J10-03 is withdrawn. See `baseline-migration-v2.log`, `legacy-load.log`, `legacy-mode.log`, `presentation-contract.log`, `documentation-contract.json`, `UNBLINDING-scope.txt`. |
| C10 | Reproducible failure -> remediation -> fresh review fixtures and Gate B regression checks pass. | **MET** | The unmodified copied tree passes go test, go vet, go test -race, and synthetic host dogfood (17 checks). Auditor safety negative controls pass separately. Two added presentation assertions fail reproducibly; these failures are retained as findings, not converted into passing tests. See `go-test.result.json`, `go-vet.result.json`, `go-race.result.json`, `host-dogfood-vcs-disabled.result.json`, `presentation-contract.result.json`. |
| C11 | Each deferred original requirement has preserved Gate D exit tests and no Gate C authority, criterion or closure claim depends on it. | **MET** | Accepted scope §0.1 preserves the deferred requirement rows and original design proof matrix. Scope §4b/§4c records current no-shell and repository-binding limits. TODO leaves reservations and Gate C independent audit/commit open. Imports grant neither launch authority, positive criterion credit, nor closure. See `docs/handoffs/2026-10-16-gate-c-scope.md:16`, `REQUIREMENTS.json`, `cmd/hma/loop.go:80`, `docs/task14-host-orchestration.md:198`, `UNBLINDING-scope.txt`, `UNBLINDING-todo.txt`. |

## Findings

### J10-01 — bind-repository returns an empty next action

Severity: **low**. Confidence: **observed**. Fix status: **proposed**.

The successful bind-repository handler encodes hostResultFor directly. That helper does not set NextAction. The real binary returns next_action:"", contrary to architecture §18. The binding succeeds; no authority bypass was observed.

- `PRESENTATION-bind.json`: Successful repository_binding record has review_enforcement:"legacy_manual" and next_action:"".
- `cmd/hma/repository.go:93`: The handler returns hostResultFor without assigning NextAction.
- `presentation-contract.log`: TestAuditor10jPresentationContract/bind-repository-next-action fails against the compiled current binary.

Preferred fix: Assign one explicit next action after successful repository binding before encoding the result. Define that action in the command contract; use NOTHING only if stopping is the intended action.

Reason: Every gate result must tell the host one next action under architecture §18.

Verification: Run the real bind-repository binary command and parse its JSON result; retain wrong-identity/rebinding refusals. Expected: Successful output has one documented nonempty next_action. Refused calls append nothing.

Trade-offs: Additive presentation change. Keep dispatch_allowed false and do not imply stage or launch approval.

Alternative: Teach hostResultFor to accept a required action parameter. This has a wider call-site cost.

Effort: small. The audit did not implement or validate this remedy.

### J10-02 — loop-evidence omits the review enforcement label

Severity: **low**. Confidence: **observed**. Fix status: **proposed**.

loop-evidence builds hostResult directly and never fills ReviewEnforcement. Its successful JSON omits review_enforcement. task14 states that every write result reports legacy_manual or packet_import. host show --mode remains correct, so this is a result-contract gap, not an observed enforcement bypass.

- `docs/task14-host-orchestration.md:158`: Every write result and host show --mode must report review_enforcement.
- `cmd/hma/loop.go:210`: The result literal does not populate ReviewEnforcement.
- `PRESENTATION-loop.json`: The real binary records loop_evidence with next_action:EVIDENCE_RECORDED but no review_enforcement member.

Preferred fix: Fill ReviewEnforcement from the committed host history while the existing host lock is held, or construct the result after releasing locks through the shared result helper. Do not invoke its locking Load inside the host transaction.

Reason: The result must disclose the same enforcement mode that the persisted run uses without creating a recursive-lock deadlock.

Verification: Run loop-evidence on legacy_manual and packet_import runs with passing and failing recaptures. Compare each result with host show --mode under a timeout. Expected: Every successful write reports the correct mode and one next action. Commands complete without deadlock; no criterion is credited.

Trade-offs: No record schema or authority change. Avoid a second unlocked read if exact response-time mode matters.

Alternative: Build all host write results through one helper that accepts the already loaded committed history.

Effort: small. The audit did not implement or validate this remedy.

## Prior finding closure

All 40 supplied prior findings are assessed below. FIXED is bounded to the cited unsafe case and the accepted Gate C boundary. DEFERRED-BY-HUMAN preserves the Gate D positive requirement. It does not mean implemented.

| Prior ID | Status | Evidence / qualification |
|---|---|---|
| F01 — A failed review can downgrade an existing blocker by ID | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-extra.log`, `auditor-negative-v2.log`. |
| F02 — Imports accept stale core bindings and can erase newer blockers | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `auditor-waiver-rewind-v2.log`. |
| F03 — Replaying an old passing run overrides a newer failed review | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `cmd/hma/gatec_r1_test.go:14`. |
| F04 — Generated packets do not bind the observed repository artifact | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-extra.log`, `auditor-first-and-journal.log`, `cmd/hma/gatec_r2_test.go:70`. |
| F05 — Final-audit reason does not enforce final-review family separation | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `cmd/hma/gatec_r2_test.go:97`, `go-race.log`. |
| F06 — Receipt import ignores provider and effort drift | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| F07 — Invalid audit reports can approve and clear findings | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `go-race.log`. |
| F08 — Unverified reference labels create new unwaivable blockers | **DEFERRED-BY-HUMAN** | Unsupported links are advisory today. Reproduced kernel/budget positive paths are explicitly human-deferred to Gate D; they are not claimed fixed. Evidence: `auditor-negative-v2.log`, `docs/handoffs/2026-10-16-gate-c-scope.md:29`. |
| F09 — The verify-loop evidence bridge is absent | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-first-and-journal.log`, `auditor-negative-v2.log`, `cmd/hma/gatec_findings_test.go:58`. |
| F10 — Review dispatches have no short validity interval | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `auditor-waiver-rewind-v2.log`. |
| F11 — Reviewer independence checks only the latest implementation | **FIXED** | Executed the repository prior-contributor matrix and independently inspected the all-contribution loop. No real worker dispatch or semantic independence proof was attempted. Evidence: `internal/host/gatec_r2_test.go:12`, `internal/host/host.go:810`, `go-race.log`. |
| F12 — Reservation checks assigned to Gate C are not implemented | **DEFERRED-BY-HUMAN** | Reservation/launch reconciliation and monotone-generation proof matrix remains Gate D under the accepted boundary. Evidence: `docs/handoffs/2026-10-16-gate-c-scope.md:23`. |
| F13 — New legacy manual-review writes bypass the consolidated import path | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `cmd/hma/gatec_r1_test.go:47`, `go-race.log`. |
| A01 — Invalid audit reports approve, clear blockers, and satisfy the host Verification gate | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| A02 — An old live-target review can clear a blocker after the worktree changes | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| A03 — Finding namespace prefixes allow a review to clear another unit blocker | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `auditor-extra.log`. |
| A04 — HMA-generated review packets omit the contract and findings snapshot | **DEFERRED-BY-HUMAN** | Current core head, accepted plan digest, criteria projection, own/unowned findings and active waiver scopes are bound. Full request/definitions/non-goals/budgets/evidence/policy inputs are human-deferred. Evidence: `cmd/hma/gatec_r3_test.go:108`, `docs/handoffs/2026-10-16-gate-c-scope.md:31`. |
| A05 — Existing unprefixed core findings have no import migration | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `baseline-migration-v2.log`, `internal/host/findings_import.go:109`. |
| A06 — Required disposition, criterion recheck and independent-credit integration are absent | **DEFERRED-BY-HUMAN** | Negative no-credit/no-invented-blocker controls pass. Positive disposition/recheck/independent-credit integration is human-deferred. Evidence: `auditor-first-and-journal.log`, `docs/handoffs/2026-10-16-gate-c-scope.md:28`. |
| A07 — Gate C launch-reconciliation integration is deferred in full | **DEFERRED-BY-HUMAN** | Wrong envelope digest/store/hook/escalation refuses. Namespace/reservation/replay/revocation enforcement is human-deferred, not claimed implemented. Evidence: `cmd/hma/gatec_r3_test.go:136`, `docs/handoffs/2026-10-16-gate-c-scope.md:27`. |
| A08 — loop-evidence executes verify argv before checking terminal state | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `auditor-extra.log`. |
| A09 — Legacy compatibility runs do not expose an enforcement-mode label | **FIXED** | The prior status/migration/manual-override closure check is fixed. J10-02 separately records the newer loop write-result label omission. Evidence: `legacy-mode.log`, `cmd/hma/gatec_r3_test.go:170`, `go-race.log`. |
| R3-01 — A newer unanswered packet does not invalidate an older approving review | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `cmd/hma/gatec_r4_test.go:11`. |
| R3-02 — The fixed review packet still omits mandatory review inputs | **DEFERRED-BY-HUMAN** | Full fixed review packet inputs remain Gate D by explicit human acceptance, with original criteria preserved. Evidence: `docs/handoffs/2026-10-16-gate-c-scope.md:31`. |
| R3-03 — The broader Gate C acceptance-boundary deferral is not approved and leaves design requirements incomplete | **DEFERRED-BY-HUMAN** | The previously unapproved boundary now has explicit Lennon 2026-10-17 approval. Deferred work remains open at Gate D, not Gate C success. Evidence: `docs/handoffs/2026-10-16-gate-c-scope.md:16`. |
| C-F14 — Legacy dispatch input can impersonate an HMA-generated review packet | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `cmd/hma/host.go:244`. |
| C-F15 — Loop recapture discards the verification output | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `auditor-extra.log`. |
| C-F16 — Go reads case-variant JSON keys that the Python contract ignores | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `internal/host/exact_json.go:21`, `go-race.log`. |
| C-F17 — Go and Python disagree on required nonempty report fields | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `internal/host/gatec_r3_parity_test.go:126`, `go-race.log`. |
| C-F18 — Interrupted output-artifact writes are trusted as complete on retry | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| C-F19 — loop-evidence executes a shell command from an imported journal | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| C-F20 — Invalid UTF-8 report passes Go checks but fails the Python validator | **FIXED** | Invalid UTF-8 report reproduced independently. Receipt and loop state share the inspected UTF-8 guard; their complete malformed-document cross-product was not independently enumerated. Evidence: `auditor-negative-v2.log`, `internal/host/exact_json.go:21`. |
| C-F21 — The shell filter refuses a shell name used as plain data | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`. |
| C-F22 — Executable shell scripts and wrapper aliases bypass the no-shell kernel | **FIXED** | The enumerated shell/script/interpreter/wrapper entry points refuse. An arbitrary native executable can internally spawn a shell; this residual is documented in source and remains a trusted approved-check limit. No universal no-shell proof is claimed. Evidence: `auditor-negative-v2.log`, `internal/evidence/noshell.go:44`. |
| C-F23 — A concurrent trusted core terminal append can precede loop check execution | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-extra.log`, `cmd/hma/loop.go:125`. |
| C-F24 — A review with a different approved base can clear the current unit blocker | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-extra.log`. |
| J8-01 — First generated packet can adopt another checkout and clear the current blocker | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-first-and-journal.log`, `cmd/hma/gatec_r6_test.go:333`. |
| J8-02 — Slash-separated finding IDs collide across exact unit owners | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `cmd/hma/gatec_r6_test.go:380`, `cmd/hma/review.go:453`, `go-race.log`. |
| J8-03 — Loop recapture can attach another repository artifact to an anchored unit | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-extra.log`. |
| J9-01 — Executable aliases bypass the loop recapture operation filter | **FIXED** | The supplied unsafe closure case is closed by the cited checks and inspected guard, within the trusted-host and accepted Gate C boundary. This is bounded evidence, not a proof over every possible input. Evidence: `auditor-negative-v2.log`, `cmd/hma/gatec_r6_test.go:419`, `go-race.log`. |

The complete original closure checks remain in AUDIT.json. Counts: {'FIXED': 33, 'DEFERRED-BY-HUMAN': 7}.

## Controlled unblinding addendum

Phase 1 is immutable: [PHASE1.json](PHASE1.json), SHA-256 `c1e6ac1e3978230a1c1530b7e4606c83f9e03dd972488dc79e89d1ded444eb1b`. The report contract validator accepted its form before later notes were read.

- Scope §4c documents the one-time bind-repository prerequisite, immutable root/identity, named refusals and legacy import limitation. The final assessment withdraws J10-03. The entry-point CLI documentation could still link this material, but that is an optional documentation improvement.
- Scope §4b explicitly discloses arbitrary native executable internals and defers accepted-plan argv binding to Gate D. This matches the inspected no-shell limit.
- TODO Gate C leaves reservation work and independent audit/commit unchecked. It records the human acceptance date. No Gate C completed closure, launch-authority or positive criterion-credit claim was found in those sections.
- J10-01 and J10-02 are unchanged because later notes do not change either observed binary result.

J10-03 is **withdrawn** in this addendum. Its original diagnosis remains in PHASE1.json and the JSON addendum. The active findings are J10-01 and J10-02. Linking scope §4c from task14 remains optional.

## Audit limitations

- Isolation is compromised. The session started fresh and inherited no implementer conversation. Two v3 re-reads printed forbidden sections: lines 140-144 (§5), then lines 167-175 (start of §7). This exceeded the named-section limit. The context cannot be reset in this session. No claim of a fully blinded independent PASS is made.
- Before Phase 1, the scope note was displayed only via sed -n 1,86p. TODO Gate C, scope §4a onward, prior audit directories and verdicts were not read. The only prior-review input was the authorized criteria-final10j.json. Authorized v7 header metadata itself names a prior Gate A verdict.
- Tests ran in a copied source tree under the audit directory, not the audited repository. The baseline writer also ran in that directory. No worker/model API was invoked. Tests use synthetic host telemetry and approvals; they grant no real HMA authority.
- Initial host-dogfood build failed because a copied tree could not obtain VCS status. GOFLAGS=-buildvcs=false fixed the environment; both logs are retained. Auditor test construction errors (Go module version Context call, waiver fixture unit_id and baseline waiver shape) are retained and corrected only in the audit copy.
- No end-to-end real agent-dispatch model run, provider reachability hi, lease/reservation authority, Gate D positive criterion credit, or final product closure was exercised or claimed. A same-UID adversary, hidden ledger forgery and arbitrary executable internals are outside the declared trusted-host scope.
- The original repository code/tests are unchanged. Only auditor tests were added under the copied tree. Source inspection and selected negative controls cannot prove the absence of every bypass or every operational/filesystem failure.
- No approved host Jev tool is exposed. Advisory decisions were made by this auditor without pretending another model was Jev.

The before-Phase-1 restriction statement above describes that phase only. Later scope notes and TODO Gate C were read after the recorded hash. The original isolation incident remains disclosed.

The source files, auditor tests, baseline writer, command results, manifests and raw logs remain under this output directory. [EVIDENCE-SHA256.json](EVIDENCE-SHA256.json) records final artifact digests. Form validation does not prove the findings or independence.

## Remediation and next action

The smallest proposed repair changes two command outputs. Keep repository binding and all authority checks intact. Retest the two binary assertions in both review modes. Then use a fresh blinded reviewer for the exact corrected target.

**One next action:** Return J10-01 and J10-02 to the lead for one bounded output-only remediation, followed by a fresh blinded review of the corrected target.

Owner: Lead implementer (Ming) and coordinating human. Route: Ming / Anthropic claude-opus-5-5 / Claude Code CLI, with effort and exact profile from the human-approved host route; proposed only, no dispatch in this audit.

Permission envelope: Current dispatch: read-only repository; writes only under this audit directory. Proposed follow-up: cmd/hma/repository.go and cmd/hma/loop.go output construction, any needed command-contract wording and focused tests. No weakening of repository binding, run locks, report checks or stage authority; no commit/push/release from this audit.

Exact approval required: A new bounded implementation authorization if writes are not already authorized by the human; current read-only dispatch grants no write permission. Exact-route hi and independent review remain required.

Stop and escalation conditions: Stop on any required permission or scope expansion, changed target during audit, failed safety regression, or inability to establish the required review isolation.
