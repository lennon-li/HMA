# Gate C independent audit

**Verdict: FAIL.** Audit ID: `hma-gate-c-audit-20261017-final3`. Date: 2026-10-17.

The auditor reproduced two defects. An invalid report can approve and clear an existing `BLOCK`. An interrupted output write can leave an empty artifact that a later capture accepts. The complete repository test suites pass. Those results do not override the independent failures.

## Scope and isolation

Code and outcome audit of `main` at `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, including the frozen working tree. The diff SHA-256 is `746a8568b14f9339ca3f42f07a57ff2fb075b4820a3d187958904c2b217c045c`. The audit excludes `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`.

Isolation: **isolated context**, with the limitations below. The auditor received the original objective, selected design sections, normative contracts, accepted scope decision, and sanitized prior closure criteria. Requirements and failure hypotheses were recorded before diff inspection. The requested route was liz / gpt-6.1-sol / high. No dispatch receipt independently attests the actual host model/runtime. No nested agent CLI or model API was used.

The target repository stayed read-only by audit procedure. It was not mounted read-only. Tests ran in a copy under this output directory. New probes used Go overlays. The copy has 183 source files that match the target. Go was `go1.27.1 linux/amd64`. The cache is local to this output directory. Git reads used `/usr/bin/git`. Existing tests create synthetic Git fixtures outside the target.

The authoritative scope is original objective §5 Gate C and §6, the specified Gate A amendments, architecture, AGENTS, and task14. Lennon accepted scope §0.1 on 2026-10-17. Gate D reservations, leases, criterion credit, reproduced kernel/budget mapping, and full contract-dependent packet inputs remain deferred. This audit does not award Gate D, Gate F, stage advancement, or closure.

## Verification

| Check | Result | Raw evidence |
|---|---|---|
| `go test -count=1 ./...` with separate `TMPDIR` | PASS, 12 packages | [log](go-test-isolated.log) |
| `go vet ./...` | PASS | [log](go-vet.log) |
| `go test -race -count=1 ./...` with separate `TMPDIR` | PASS, 12 packages, no race reports | [log](go-race-isolated.log) |
| `scripts/dogfood/host-dogfood.sh --keep` | PASS, 17 synthetic checks | [log](host-dogfood.log) |
| Auditor report, receipt, finding, target, concurrency and recapture probes | FAIL only on four C-F17 whitespace subcases; six other top-level probes pass | [normal](auditor-probes.log), [race](auditor-probes-race.log) |
| Auditor baseline migration, caller-packet boundary, terminal race and loop-key probes | PASS under `-race` | [log](auditor-supplement.log) |
| Auditor interrupted-output fixture | FAIL under `-race`, C-F18 | [log](auditor-output-crash.log) |
| Delivered interface parity and retained-contributor independence tests | PASS; no matching repostate tests in this filter | [log](interface-parity.log) |

The first normal and race suites failed `TestSyntheticStoreRefusesNonScratchDir`. The unchanged helper assumes its repository directory is outside `os.TempDir`. This audit copy was under `/tmp`. Both full suites passed after `TMPDIR` was set to a separate child directory. The [initial normal log](go-test.log) and [initial race log](go-race.log) remain available. No test was removed or weakened.

Commands, exit codes, parameters and log hashes are in [execution-ledger.json](execution-ledger.json). The probes reuse repository setup helpers. Their inputs and assertions are auditor-written. Sources are [auditor_probes_test.go](auditor_probes_test.go), [auditor_supplement_test.go](auditor_supplement_test.go), and [auditor_output_crash_test.go](auditor_output_crash_test.go).

## Gate C requirement coverage

`PARTIAL` maps to `fail` in the machine report because a mandatory requirement remains unsatisfied.

| ID and source | Status | Observation | Evidence |
|---|---|---|---|
| C01: scope §1; v4 §5; v6 §1 | MET | HMA creates and hashes exact-target packets and unique nonce bindings. Caller Packet injection is rejected on both legacy and migrated runs. | E02, E05 |
| C02: scope §2; v4 §5; v5 §3; normative task14 | MET | Receipt identity, provider/effort, root, nonce, exact report bytes, half-open launch interval, latest dispatch, current implementation, core head and live-target checks hold in exercised cases. | E02, E03, E05 |
| C03: v3 §4.3; v5 §3; architecture §§12-13,16 | PARTIAL | Normal invalid reports escalate/refuse. U+001C..U+001F-only required fields pass Go but fail Python; these reports approve and open the host gate. Final family/isolation guards otherwise hold. | E01, E03 |
| C04: v3 §4.1-4.2; architecture §§7,9 | PARTIAL | Failed/escalated snapshots preserve strong findings and exact ownership. The C-F17 invalid approving report still clears an existing own-unit BLOCK. | E01, E03, E05 |
| C05: original §5 Gate C compatibility; scope §1; task14 | MET | Records generated using the baseline 527b7c5 model/store load with unchanged anchors. Own legacy findings resolve; ambiguous findings and active waivers remain. Packet mode blocks new manual approvals. | E05, E06 |
| C06: original §5 Gate C; v3 §3.2; architecture §14; accepted scope §0.1 | PARTIAL | Bridge rechecks argv and target, binds envelope/run/unit/store and retains normal/truncated streams without criterion credit. C-F18 accepts an incomplete existing output artifact. | E03, E04, E05 |
| C07: architecture §§5-7,14-15; task14 | MET | Gate B suite and dogfood pass. Missing/terminal runs refuse before argv. A queued recapture after terminal commit refuses under the host-lock protocol. No stage approval is generated. | E02, E03, E05 |
| C08: original §5 Gate C, §6 minimum proofs 4,6,7; scope §4 | PARTIAL | Delivered failure/remediation/reverification fixtures pass. Independent negative controls expose two unresolved defects, so the full Gate C acceptance set does not pass. | E01, E02, E03, E04 |
| C09: accepted human scope §0.1; criteria note | MET | Accepted scope §0.1 records Gate D prerequisites and exit tests. Gate C imports grant no criterion credit, launch authority, or terminal closure. Full contract packet inputs and reservation matrix are deferred by the human. | E07 |

The matrix contains 5 MET and 4 PARTIAL requirements. No missing Gate D feature is treated as a Gate C defect.

## Observed findings and proposed remedies

### C-F17 — Go and Python disagree on required nonempty report fields; invalid reports can approve and clear BLOCK

Severity: **high**. Certainty: **observed**. Priority: 1. Fix status: **proposed**.

internal/host/report_contract.go:22 uses strings.TrimSpace. Python validate_report.py uses str.strip, which additionally treats U+001C through U+001F as whitespace. A current, receipt-bound pass report with audit_id, target, requirement id and evidence reference consisting only of each such character fails the authoritative Python validator but has no Go violations. review-import records APPROVE; findings-import removes PRIOR/BLOCK; requireHostTransitionGate returns nil. This satisfies the host gate only; no actual stage transition or human approval was fabricated.

Evidence:
- `auditor-probes.log:10-65; auditor-probes-race.log:10-65`: Four subcases reproduce Python exit 1, Go violations=[], APPROVE, empty findings, and gate=<nil>.
- `internal/host/report_contract.go:22; internal/host/review_import.go:183-206; cmd/hma/review.go:427-430`: The nonempty predicate is reused across contract fields. Its false acceptance feeds the approval map and approving findings replacement.
- `invalid-report-U001C.json; whitespace-result-U001C.json`: Retained exact invalid report plus host/core records show the prior BLOCK and its replacement.

Preferred fix: In internal/host/report_contract.go, use one nonempty predicate with the authoritative Python strip whitespace set, including U+001C..U+001F. Extend parity tests across all nonempty fields and route approval tests through each invalid report. Keep invalid reports ESCALATE or refused, and preserve blockers.

Why: The imported report contract must have the same field semantics in Go and Python before pass can authorize an approving snapshot.

Independent exit checks:
- Run auditor TestAuditorReportWhitespaceParity for U+001C..U+001F, with the existing valid current report control. Expected: Every Python-invalid report refuses or escalates, retains PRIOR/BLOCK, and leaves the host gate closed. Valid current report still approves.
- Differentially test nonempty fields using ASCII whitespace, U+0085, U+00A0, U+2007, U+202F, the four controls, and mixed real text. Expected: Go and Python accept and reject the same inputs.

Trade-offs: A report with previously accepted control-only fields will no longer approve. This is a small compatibility restriction on invalid reports. No schema or stage semantics need to change.

Alternative: Call the canonical Python validator before approval. This avoids the port gap but adds a runtime dependency to the standalone Go integration.

Effort: small. Repair before Gate C acceptance. The auditor did not implement or validate the proposed fix.

### C-F18 — Interrupted output-artifact writes are trusted as complete on retry

Severity: **medium**. Certainty: **observed**. Priority: 2. Fix status: **proposed**.

retainOutput creates the final digest-named path with O_EXCL before writing bytes. An interrupted process can leave that final path empty or partial. On retry, os.ErrExist returns its reference without reading or validating it. An auditor fixture places an empty file at the exact expected path, representing a crash between OpenFile and Write. loop-evidence then reports EVIDENCE_RECORDED and appends both records with a reference to 0 bytes instead of the required 99-byte output artifact. No same-UID adversary is needed for this failure state. The test simulates crash residue; it does not claim to have killed a process at that instruction.

Evidence:
- `auditor-output-crash.log:2-6; interrupted-output-result.json`: loop-evidence accepts with exit 0; output_ref resolves to 0 bytes, expected 99; OutputDigest=c7d386429b24fec97b446d8e2b50a23942057ca5ab8b8a9839776f0ca443fd16 cannot be reconstructed from that artifact.
- `cmd/hma/loop.go:288-299`: The final name is created before Write/Sync, and the existing-file branch returns without validating content.

Preferred fix: In retainOutput, validate the exact bytes/digest and regular-file type of an existing artifact before returning a reference. Publish new artifacts only after a complete temporary file is written and synced, with atomic no-clobber publication and directory sync. Refuse a corrupt existing artifact before either evidence append; preserve it for diagnosis.

Why: A final digest-named path must identify complete bytes, including after interruption. Filename existence alone does not prove content integrity.

Independent exit checks:
- Run TestAuditorOutputInterruptedArtifact with empty, truncated and wrong-byte existing artifacts, plus a valid existing-artifact control. Expected: Incomplete/corrupt artifacts never produce accepted evidence. Valid existing artifacts reproduce both streams and OutputDigest.
- Interrupt artifact publication before and after sync/publish, restart and repeat capture. Expected: Every accepted output_ref is complete and verifiable. No partial final artifact is trusted; both chains remain unchanged on corrupt-artifact refusal.

Trade-offs: Existing-artifact reuse adds a bounded read/hash. Atomic no-clobber publication needs platform-aware handling. Recovery must not silently rewrite evidence history.

Alternative: Keep exclusive final-path writes but validate existing bytes and fail closed on mismatch; this contains false evidence while leaving manual recovery for crash residue.

Effort: small. Repair before Gate C acceptance. The auditor did not implement or validate the proposed fix.

## Prior finding closure

Each status is limited to the scoped defect. A FIXED finding does not erase the new defects. A DEFERRED-BY-HUMAN status records the accepted Gate D boundary. It does not mean its implementation passed.

| Prior ID | Status | Independent observation | Evidence |
|---|---|---|---|
| F01 | FIXED | Nonapproving snapshot mapping preserves same-ID BLOCK/WAIVABLE strength and omitted strong findings. The safe snapshot may append retained findings; it does not apply a downgrade. | E03 |
| F02 | FIXED | Both import paths bind the exact core head; any intervening record fails stale binding, with only the own imported snapshot allowed at the host gate. | E02, E03 |
| F03 | FIXED | One-use latest-dispatch checks prevent old review replay. Concurrent import admits exactly one; a newer unanswered packet blocks older snapshots/gates. | E02, E03 |
| F04 | FIXED | Packet generation observes and derives the artifact; wrong declared artifact and receipt root/head/worktree refuse; later tracked/untracked/dirty-content changes refuse findings import. | E02, E03 |
| F05 | FIXED | Final issuing reason invokes strong family separation even without FinalValidation; normal and independent controls pass. | E02, E06 |
| F06 | FIXED | Missing/wrong provider and effort refuse with both chains unchanged; exact route controls pass. | E03 |
| F07 | PARTIAL | Common malformed/inconsistent candidates refuse or escalate. C-F17 still approves a Python-invalid report and clears BLOCK. | E01, E03 |
| F08 | DEFERRED-BY-HUMAN | Positive reproduced-kernel and measured-budget mappings are Gate D under accepted §0.1. Negative fake/unreproduced references stay ADVISORY; cited pass escalates. | E02, E07 |
| F09 | FIXED | The bridge exists and independently recaptures rather than trusting green; failing/passing checks, envelope drift and mutation controls pass without criterion or launch credit. Output-retention failure is tracked separately as C-F18/C-F15. | E02, E03, E04 |
| F10 | FIXED | Launch interval is frozen and half-open. At start and just-before-end controls pass; before, at end, after and missing time refuse. Core-head staleness prevents post-rewind reuse. Lease generation work remains deferred. | E02, E03, E07 |
| F11 | FIXED | Independence loops over every retained implementation contribution for the unit. Earlier-worker and earlier-model cases refuse; independent-of-both passes. | E06 |
| F12 | DEFERRED-BY-HUMAN | Reservations, charged unlaunched attempts, replay eligibility and lease generations move unchanged to Gate D with exit tests. No Gate C execution/closure claim uses them. | E07 |
| F13 | FIXED | Packet mode is sticky for the run. New manual review/dispatch paths refuse after migration; legacy compatibility remains labelled. | E02, E05 |
| A01 | PARTIAL | Standard contract failures preserve BLOCK and keep gate closed, but C-F17 reproduces the same unsafe outcome with control-only required text. | E01, E03 |
| A02 | FIXED | Both imports re-snapshot live artifact; own tracked, untracked, and dirty-content cases retain blocker and chains. Current exact target control passes. | E03 |
| A03 | FIXED | Ownership uses UnitID rather than ID prefix. U1/sub and unowned blockers survive U1 approval; nonapproving same-ID cases preserve strength. | E03 |
| A04 | DEFERRED-BY-HUMAN | Known blocker, bound core head, route, plan digest and projected state are present. Full contract-dependent packet inputs are explicitly deferred to Gate D by the human. | E02, E07 |
| A05 | FIXED | A genuine baseline-written record loads with its original anchor. Unit-owned legacy finding resolution works; ambiguous IDs and active finding waiver remain visible. | E05 |
| A06 | DEFERRED-BY-HUMAN | Positive disposition/criterion-credit integration needs Gate D registries, budgets and approved acceptance contract. Gate C has no credit path. | E07 |
| A07 | DEFERRED-BY-HUMAN | Gate C envelope hash/store/hook/stop checks are exercised and pass. Namespace/reservation reconciliation matrix is Gate D by accepted scope. | E03, E05, E07 |
| A08 | FIXED | Missing/terminal admission refuses before marker execution; a terminal writer holding host lock wins before queued recapture and no argv starts. Open control captures output without deadlock. | E03, E05 |
| A09 | FIXED | Write results and host show --mode expose legacy_manual versus packet_import. New manual approvals are refused after a packet. | E02, E05 |
| R3-01 | FIXED | Concurrent import allows one result. A newer pending packet prevents old snapshot and gate acceptance; its current valid report+snapshot reopens the host gate. | E03 |
| R3-02 | DEFERRED-BY-HUMAN | Full fixed request/criteria/non-goals/budgets/evidence/policy inputs depend on the acceptance contract and are explicitly moved to Gate D. Current blocker/state binding remains in scope and implemented. | E07 |
| R3-03 | DEFERRED-BY-HUMAN | Lennon accepted §0.1 on 2026-10-17. Preserve the full original design exit matrix for Gate D; this audit makes no reservation, independent-credit or closure claim. | E07 |
| C-F14 | FIXED | Caller-supplied Dispatch.Packet refuses before append on both legacy and packet_import runs. Genuine packet/import controls pass. | E03, E05 |
| C-F15 | PARTIAL | Normal and capped stdout/stderr remain accessible and reproduce framed digest, including nonzero exits. C-F18 accepts an empty artifact left at the final name by an interrupted write. | E03, E04 |
| C-F16 | FIXED | Exact key decoding ignores ASCII and Unicode case variants in reports, receipts, nested bindings and loop state; safe report verdict is retained. Whitespace parity in C-F17 is a separate defect. | E03, E05, E06 |

Totals: **18 FIXED, 3 PARTIAL, 7 DEFERRED-BY-HUMAN, 0 OPEN**. The two new findings remain unresolved.

## Evidence inventory

| ID | Files and source references | Observation |
|---|---|---|
| E00 | `freeze-start.json`, `start-status.txt`, `start-diff.binary`, `requirements-derived.json` | Neutral target freeze and requirement reconstruction before diff inspection. |
| E01 | `auditor-probes.log`, `auditor-probes-race.log`, `invalid-report-U001C.json`, `invalid-report-U001D.json`, `invalid-report-U001E.json`, `invalid-report-U001F.json`, `whitespace-result-U001C.json` | Independent Python/Go report parity counterexample; invalid approval, BLOCK clearance and host gate acceptance. |
| E02 | `go-test-isolated.log`, `go-race-isolated.log`, `go-vet.log`, `host-dogfood.log` | Repository suites pass in isolated copy; synthetic host dogfood passes 17 checks. Initial environmental failures retained in go-test.log/go-race.log. |
| E03 | `auditor_probes_test.go`, `overlay.json`, `auditor-probes.log`, `auditor-probes-race.log` | Auditor matrices for report safety, receipt route/window, exact ownership, concurrent import/pending packet, live target mutation, closed/missing runs, retained and capped output. |
| E04 | `auditor_output_crash_test.go`, `overlay-output.json`, `auditor-output-crash.log`, `interrupted-output-result.json` | Independent simulated interrupted-artifact test accepts a zero-byte referenced output artifact. |
| E05 | `auditor_supplement_test.go`, `overlay-supplement.json`, `auditor-supplement.log`, `legacy-store/legacy.jsonl`, `baseline/scripts/auditorlegacy/main.go` | Genuine baseline record load/migration, exact waiver retention, both-mode manual-packet rejection, queued terminal race and loop aliases pass under -race. |
| E06 | `interface-parity.log`, `internal/host/gatec_r3_parity_test.go`, `internal/host/gatec_r2_test.go` | Delivered Python/Go canonical-hash and report corpus parity tests pass; earlier-implementer independence tests pass. Corpus does not include C-F17. |
| E07 | `docs/handoffs/2026-10-16-gate-c-scope.md:16-35`, `/tmp/hma-gate-c-criteria/criteria-final3.json`, `docs/task14-host-orchestration.md:137-237` | Human-approved Gate D boundary retains original exit tests and prohibits reliance for Gate C credit/launch/closure. |

The full working-tree diff is retained in [start-diff.binary](start-diff.binary). Source digests and copy equality are in [source-inventory.json](source-inventory.json). Every untracked file hash is in the freeze manifests, including excluded files for change detection only. Excluded file contents were not audited.

## Controlled unblinding

Phase 1 was written once to [PHASE1.json](PHASE1.json). Its SHA-256 is `5cf5fe74fcabd798070ec139cfd37a50d49c4c8f018a58a1b47f22f23271781f`. The [hash record](PHASE1.sha256) was recorded before unblinding.

After that freeze, the auditor read scope §4a and later and the Gate C TODO section. Those sections describe implementation claims and historical findings. They do not change either reproduction. TODO leaves the independent Gate C audit and commit open. It records the accepted reservation deferral. No new factual claim changed the frozen findings or verdict. Other prior-review directories remained unread.

## Limits and unresolved conditions

- Fresh audit context contains original objective, approved source sections and the sanitized closure criteria only. The required v7 source header contains a historical Gate A verdict; source artifacts and comments naturally contain author claims. No prior Gate C verdict or implementer conversation was read before Phase 1. This is context isolation, not an OS-enforced read-only mount or proof of hidden system routing.
- All execution used a copied repository or Go overlays under this output directory. Audit permissions, rather than an OS-enforced read-only filesystem, protected the original repository. No target edits, Git mutations on the target, nested agent CLIs or model APIs were used. Existing tests create and mutate synthetic Git fixtures outside the target.
- Fixtures use synthetic receipts, route approvals and host preflight telemetry. No live worker, exact-route hi, authentic human stage transition, or final outcome closure was executed. This is the Gate C integration audit, not Gate F real-worker dogfood.
- Gate D reservations, leases/generations, independent criterion credit, reproduced kernel/budget dispositions and full contract-dependent packet inputs are excluded by explicit human decision. Their exit tests remain required before later claims.
- The first full normal/race suites failed one unchanged dogfood-helper test because the repository copy was under os.TempDir. Both complete suites passed after giving tests a separate TMPDIR; both initial logs remain available.
- Independent probes reuse repository fixture setup helpers but define their own test inputs, adverse cases and assertions. Tests are finite; passed suites are not proof over all possible concurrency schedules or filesystems.
- Crash residue was simulated at the artifact filename boundary. No real power loss or storage-failure injection was performed.
- No approved Jev tool is exposed. Auditor decisions use the permitted fallback; no model/API substitution was made.

No optional redesign is recommended. The two concrete repairs are enough to address the observed gaps.

## Remediation order and exit criteria

1. Repair C-F17 first. The four Python-invalid reports must never approve, clear `BLOCK`, or satisfy the host gate. Keep the valid report control working.
2. Repair C-F18. Corrupt or incomplete existing artifacts must refuse before evidence append, or recover through a verified atomic publication path. Retained bytes must reproduce both streams and the framed output digest.
3. Run an independent read-only retest of the exact repaired target. Rerun the two counterexamples, affected parity/recapture checks, and the required repository checks. Preserve the original acceptance boundary.

## Final integrity and report contract

The [end freeze](freeze-end.json) equals the [start freeze](freeze-start.json) for HEAD, branch, status hash, diff hash, and every untracked file hash. The audited repository is unchanged. The machine report is [AUDIT.json](AUDIT.json). Its `fix-required` value is the independent-audit contract spelling of the single Gate C verdict **FAIL**. Structural validation output is retained in [report-validation.log](report-validation.log); validation proves report form only.

## Required next action — architecture §18

**RETURN_TO_IMPLEMENTATION.** Owner: the human Gate C coordinator. Rationale: C-F17 and C-F18 remain observed failures. The human must authorize a bounded repair unit and an eligible exact implementer route. The proposed permission envelope covers report predicate parity and output-artifact integrity with regression probes. This report grants no write permission, stage approval, commit, push, or release authority. Stop and escalate if repair needs changed criteria, wider permissions, or design changes. Gate C must stay unaccepted until an independent retest passes the repaired target.
