# Independent Gate C audit — FAIL

The repository suite, full race suite, vet, and 17-check host dogfood pass. Independent controls preserved blockers and rejected stale imports. They did not produce an approval, criterion-credit, stage-transition, or closed-run execution bypass. Two findings-import response defects remain. The response can recommend continuation after a failed review. It also omits its documented enforcement label.

Audit: `hma-gate-c-audit-20261017-final11`. Date: 2026-10-17. Modes: code and outcome. Auditor assignment: liz / gpt-6.1-sol / high / review mode, as supplied by the dispatch. No new worker was dispatched. No model API was called. Jev was unavailable in the callable host inventory; orchestrator judgment was recorded.

Target: branch `main`, HEAD `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree. Full `git diff --binary` SHA-256: `2abaadf6cbdf68e314bfdbb5f95db7e2f1eb610d25f91d84fca2ed6570fefab0`. All 30 untracked files were hashed, including excluded paths for freeze comparison. The full status, diff digest, branch, HEAD and untracked map match at the end. [Start freeze](freeze-start.json), [end freeze](freeze-end.json), [comparison](freeze-comparison.json), and [audited file manifest](audited-file-manifest.json) preserve the evidence.

Isolation: **isolated evaluator context**, with the execution and blinding limits below. Phase 1 was frozen before unblinding. SHA-256: `bb9fccd7a352a4f39767244ddc348c60883ffd62550e6d4228fb034945753855`. [PHASE1.json](PHASE1.json) remains unchanged. This is a scoped Gate C FAIL. It is not a Gate D, release, or product-readiness verdict.

## Objective and accepted scope

The authoritative objective is the consolidation gate in the outcome handoff, §5 Gate C and §6. It requires integration of the existing verify-loop and independent-audit interfaces, failure/remediation/re-verification fixtures, review separation, and migration/compatibility review. The approved design sections were v3 §3–§4/§6, v4 §3–§5, v5 §1–§3, v6 §1/§5, and v7. Latest applicable amendments win. Normative contracts were architecture.md, AGENTS.md and task14-host-orchestration.md.

Lennon accepted the scope note §0.1 boundary on 2026-10-17. Leases, reservations, launch reconciliation, positive criterion credit, reproduced kernel/budget import dispositions, D5 admission, and full acceptance-contract packet inputs remain Gate D requirements. Their recorded exit criteria were assessed. Their implementation is not credited here. The audit excludes `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`. [Derived requirements](derived-requirements.json), [allowed design excerpts](requirements-sources.txt), and [interface hashes](interface-manifest.json) retain provenance.

## Requirement coverage

Ten requirements are MET. Two are PARTIAL. No requirement is marked MET solely because tests exist.

| ID | Requirement | Result | Independent observation and evidence |
|---|---|---|---|
| C01 | Generated immutable packet binds independently observed approved repository root/base/head/diff/worktree; exact plan/unit/core head/criteria/findings/waivers present within accepted boundary | **MET** | Packets derive the artifact from Snapshot; they require the explicit one-time repository binding, the approved base, current core state, plan digest, projected criteria, owned/unowned findings, and active waivers. Wrong root/base and packet artifact controls pass. (E-CODE, E-BASE, E-TEST) |
| C02 | Fresh random 128-bit nonce, dispatch/run ID and packet digest must bind current independent read-only reviewer dispatch with valid exact preflight and half-open short launch interval | **MET** | crypto/rand supplies 16-byte nonce and dispatch identity. Preflight, read-only capability, all-contribution independence, final family separation and half-open launch time checks are enforced. (E-CODE, E-TEST) |
| C03 | Receipt completed exit-zero review matches run, packet, launch artifact, canonical project root, worker/provider/model/effort and report bytes; no stale/other report imports | **MET** | Receipt mismatch controls refuse before append. They include wrong run/provider/effort/root, case variant mode, unlisted report, pre-issuance/expiry times, old nonce, invalid UTF-8, and missing report. Exact controls approve. (E-AUDITOR, E-TEST) |
| C04 | Total verdict mapping and independent-audit contract validation prevent invalid reports approving; final isolated only; independent against all retained implementations incl final family separation | **PARTIAL** | Review import maps and validates verdicts correctly. Independent validity parity covers 82 new cases; repository parity tests also pass. Final isolation and independence controls pass. J11-01: a subsequent findings snapshot loses the non-approving verdict in its next-action recommendation. (E-AUDITOR, E-CONTRACT, E-RESPONSE) |
| C05 | Latest packet/current implementation only, single import and snapshot, exact live target/core binding; newer packets/failures invalidate prior approval; concurrency serializes | **MET** | Live tracked/untracked mutations refuse both import phases without store changes. Exactly one of two concurrent imports succeeds. A newer unanswered packet closes the gate and refuses the older snapshot. Any unrelated core append makes the review stale. (E-AUDITOR, E-TEST) |
| C06 | Additive snapshot semantics preserve legacy behavior; nonapproval cannot remove/weaken BLOCK or WAIVABLE; authentic current APPROVE may resolve exact own-unit findings; preserve other/unowned findings and waivers | **MET** | Non-approving snapshots preserve BLOCK/WAIVABLE, including omitted IDs. Exact owners prevent cross-unit/prefix clearing. Unowned blockers survive. A current approval resolves its own findings. Baseline-written records load with unchanged anchors and migrate. (E-AUDITOR, E-BASE, E-TEST) |
| C07 | Imported labels/severity never fabricate blockers or acceptance credit; no import grants criterion PASSED or stages/launch/closure authority | **MET** | Imports never write criterion dispositions or stage-transition records. Unsupported kernel/budget labels remain advisory; a pass carrying a citation escalates. Criterion sentinels remain PENDING through import, snapshot, capture and migration. (E-AUDITOR, E-LIFECYCLE, E-CODE) |
| C08 | verify-loop bridge verifies canonical envelope/run/unit/store/root and hook/stop; independently captures approved check without shell at bound pre/post target with retained stdout/stderr/truncation/digest | **MET** | Canonical envelope and run/unit/store/root checks are present. HMA captures the executable itself and observes the pre/post snapshot. Independent shell/alias/shebang/residue controls pass. Both retained streams re-hash correctly; nonzero exit and the 1 MiB cap/truncation remain explicit. (E-LIFECYCLE, E-OUTPUT, E-TEST) |
| C09 | Closed/missing runs, wrong unit/route, wrong root/base and review-run reuse refuse before executing checks or writing evidence; core/host locking prevents terminal race | **MET** | Host then core locking spans terminal/route checks, execution, evidence append and host commit. Independent paused check serializes a concurrent ABORTED append after capture; a second call on the closed run executes nothing and changes no store bytes. Wrong root/base and review-run association controls pass. (E-AUDITOR, E-TEST) |
| C10 | Existing records/digests project compatibly; legacy mode explicit; first packet migrates permanently to imported reviews and refuses host-created packet/manual approval bypass | **PARTIAL** | Baseline core/host chains load with the original anchors. legacy_manual is visible and the first generated packet permanently moves the run to packet_import. Manual approval/packet bypass controls refuse. J11-02: findings-import omits the documented review_enforcement result field. (E-BASE, E-RESPONSE, E-TEST) |
| C11 | Executable failure→linked remediation→fresh re-verification plus stale/self-review/no-report controls and Gate B guarantees remain intact | **MET** | go test, go vet and the full race suite pass on copied source. The 17-check host dogfood passes. An independently written failure→BUG finding→failed re-capture→scratch repair commit→new implementation→fresh passing re-capture→new review/snapshot fixture passes without criterion credit or stage advancement. (E-TEST, E-LIFECYCLE, E-DOGFOOD) |
| C12 | Accepted Gate D rows remain recorded with exit tests; no Gate C bounded-execution/positive-credit/closure claim relies on deferred functionality | **MET** | The accepted §0.1 scope retains Gate D requirements and named exit tests/reproduction conditions. Imported disposition, positive credit, reservations, lease/generation reconciliation, D5 admission and full contract inputs are not treated as implemented authority. This is Gate C evidence/import integration only. (E-SCOPE, E-CODE) |

Each row retains its authoritative section citation in [AUDIT.json](AUDIT.json). “PARTIAL” maps to report-contract status `fail`; “MET” maps to `pass`. The top-level contract verdict is `fix-required`, which is the requested **FAIL** verdict.

## Findings

### J11-01 — A non-approving findings snapshot recommends stage continuation

Severity: **medium**. Confidence: observed. Status: OPEN. Fix status: proposed.

After a receipt-bound fix-required review or blocked report (ESCALATE), findings-import returns next_action CONTINUE_HMA_TRANSITION when no BLOCK/WAIVABLE findings remain. The same run still fails the host Verification gate. runHostFindingsImport selects its next action only from finding dispositions and discards rr.Review.Verdict. Imported findings are advisory under the accepted boundary, so a normal failed audit can reach this output. This violates architecture §18 and the total-verdict escalation contract. The deterministic gate remains closed; this reproduction does not show an approval or stage bypass.

- `cmd/hma/review.go:482`: Default continuation is replaced only when a finding is non-advisory; the review verdict is not considered.
- `logs/result-contract.log:21`: fix-required snapshot returned CONTINUE_HMA_TRANSITION; the independent gate check failed.
- `logs/result-contract.log:31`: blocked/ESCALATE snapshot returned CONTINUE_HMA_TRANSITION; the independent gate check failed.

Preferred fix: In runHostFindingsImport, retain the latest imported review verdict and choose the result next_action from it first: ESCALATE returns ESCALATE_TO_HUMAN and REQUEST_CHANGES returns RECORD_FINDINGS_AND_REMEDIATE. For APPROVE, account for remaining blockers before recommending continuation. Keep all existing gate checks.

Why: The review verdict is the authoritative host review result; advisory-only snapshot contents cannot turn a failed review into a continuation recommendation.

Independent verification:

- Run TestAuditNonApprovingSnapshotNextAction for fix-required and blocked, including empty and advisory-only snapshots. Expected: REQUEST_CHANGES recommends remediation; ESCALATE recommends human escalation; both gates remain closed.
- Run a current pass plus snapshot with no residual blocker. Expected: Continuation remains recommended and the host gate passes; no criterion or stage is written.

Trade-off: Response guidance changes for callers that followed the incorrect continuation token. Persisted records and review provenance need no migration.

Alternative: Return only a neutral FINDINGS_RECORDED action and require another status command to compute the next step; this adds a host round trip and needs a documented contract change.

Effort: small. Priority: 1. No product fix was applied or validated by this auditor.

### J11-02 — findings-import omits its documented review enforcement label

Severity: **low**. Confidence: observed. Status: OPEN. Fix status: proposed.

The real binary successfully writes a findings snapshot but its result has no review_enforcement member. host show --mode returns packet_import for the same run. findingsImportResult is a separate response type with no such field. The normative task14 contract says every write result reports review_enforcement. Import provenance and gate checks still enforce packet_import; this is an output-contract omission.

- `docs/task14-host-orchestration.md:158`: Every write result and host show --mode must report the run review_enforcement.
- `cmd/hma/review.go:380`: findingsImportResult contains recorded, core_sequence, core_head_anchor, findings and next_action only.
- `logs/result-contract.log:9`: Successful real-binary findings-import result omitted review_enforcement; the subsequent show returned packet_import.

Preferred fix: Add review_enforcement to findingsImportResult. Populate it from hoststate.ReviewEnforcementMode(history) while the findings import holds the host run lock.

Why: Use the same validated host history that authorized the snapshot so every successful write result exposes the documented mode.

Independent verification:

- Run TestAuditFindingsResultModeContract against the real hma binary; compare its result with host show --mode. Expected: The successful response includes review_enforcement=packet_import and agrees with show. Existing bind-repository and loop-evidence response checks still pass.

Trade-off: This is an additive response field; strict downstream JSON consumers may need to accept it. It does not alter core/host record digests.

Alternative: Amend the task14 contract to explicitly exclude findings-import, with human agreement; emitting the field is the smaller implementation change.

Effort: small. Priority: 2. No product fix was applied or validated by this auditor.

## Prior finding closure

All 42 prior criteria were assessed. Thirty-five findings are FIXED. Seven are DEFERRED-BY-HUMAN. No prior finding is OPEN or PARTIAL within the accepted Gate C boundary. The two new response findings remain OPEN. Mixed deferred findings retain their implemented negative controls in the notes. “DEFERRED-BY-HUMAN” does not mean implemented or waived. Exact supplied closure checks remain in AUDIT.json.

| Prior ID | Status | Assessment | Evidence |
|---|---|---|---|
| F01 | FIXED | Strength matrix preserves BLOCK and WAIVABLE on same-ID downgrades and omissions for REQUEST_CHANGES/ESCALATE. Legitimate failed-review snapshots may append; no downgrade is applied. | E-AUDITOR |
| F02 | FIXED | Imports compare exact current core anchor. Findings import uses expected sequence/predecessor on append; a later core record refuses stale import. | E-TEST |
| F03 | FIXED | Only latest packet is answerable once. Concurrent duplicate imports accept exactly one; newer packet prevents older snapshot/gate reuse. | E-AUDITOR |
| F04 | FIXED | Snapshot-derived target and approved root/base binding reject wrong artifacts and other repositories. | E-TEST |
| F05 | FIXED | IssueFinal independently triggers provider/model-family separation even with FinalValidation=false. | E-TEST |
| F06 | FIXED | Receipt provider and effort must be present and exact; wrong/missing variants refuse without store change. | E-AUDITOR |
| F07 | FIXED | Invalid reports refuse or escalate. Failed/unknown requirements, missing evidence, unfrozen phase1 and malformed findings cannot approve. | E-CONTRACT |
| F08 | DEFERRED-BY-HUMAN | Negative labels are safe now: unreproduced references remain advisory and cited pass escalates. Reproduced kernel/budget positive paths require Gate D registry and budgets. | E-SCOPE |
| F09 | FIXED | Independent failure→finding→actual scratch repair→fresh re-capture→new review fixture passes; bridge grants no criterion, launch or stage authority. | E-LIFECYCLE |
| F10 | FIXED | Launch time must fall in the fixed packet half-open interval; before issuance, exact expiry and after expiry refuse. Core rewinds also invalidate bindings. | E-AUDITOR |
| F11 | FIXED | Every historical implementation contribution is checked, including distinct-worker/model/family and context requirements. Multi-route history controls pass. | E-TEST |
| F12 | DEFERRED-BY-HUMAN | Reservation/admission/consumption/replay/reconciliation and v6/v7 validity tests are explicitly retained for Gate D; no execution-leash authority claimed. | E-SCOPE |
| F13 | FIXED | Actual HEAD-written chain loads; first packet permanently requires imported reviews; manual dispatch/approval provenance bypasses refuse. Unmigrated mode is visible. | E-BASE |
| A01 | FIXED | Independent invalid report matrix preserves existing BLOCK/WAIVABLE and keeps the host gate closed; valid own resolution control succeeds. | E-AUDITOR |
| A02 | FIXED | Tracked and untracked live changes before either import refuse, preserve store bytes and preserve blockers. | E-AUDITOR |
| A03 | FIXED | Explicit UnitID ownership preserves U1/sub blockers on U1 approval; prefix is not used as ownership. | E-AUDITOR |
| A04 | DEFERRED-BY-HUMAN | Current packet carries plan digest, criteria, core anchor, owned/unowned findings and waivers. Contract-dependent full fixed packet inputs remain human-deferred with R3-02 exit criteria. | E-SCOPE |
| A05 | FIXED | Baseline-written unprefixed own finding is resolved only by current own-unit approval. Ambiguous last-source ownership stays unowned; imports do not drop waiver records. | E-BASE |
| A06 | DEFERRED-BY-HUMAN | Reproduced kernel/budget disposition and positive independent criterion credit require accepted Gate D prerequisites. Current imports keep unsupported labels advisory and give no credit. | E-SCOPE |
| A07 | DEFERRED-BY-HUMAN | Current envelope/run/unit/store/stop/hook checks pass. Lease namespace/replay/revocation reconciliation is human-deferred unchanged to Gate D. | E-SCOPE |
| A08 | FIXED | Independent concurrent terminal append serializes after a paused live check; subsequent closed-run capture executes nothing. Missing/terminal fixtures also pass. | E-AUDITOR |
| A09 | FIXED | Legacy status and migration mode are visible in host show --mode; migration refuses manual override. Findings-import output omission is separately J11-02. | E-BASE |
| R3-01 | FIXED | A newer unanswered packet prevents an earlier snapshot or gate approval. Host locking serializes single-use imports and dispatch updates. | E-AUDITOR |
| R3-02 | DEFERRED-BY-HUMAN | Immutable request, full criterion definitions/non-goals/budgets/policy/evidence inputs need the acceptance contract. Human accepted Gate D deferral; existing R3-02/A04 reproduction criteria are retained. | E-SCOPE |
| R3-03 | DEFERRED-BY-HUMAN | Lennon explicitly accepted §0.1 on 2026-10-17. Required deferred rows and exit checks remain Gate D obligations, not Gate C success claims. | E-SCOPE |
| C-F14 | FIXED | host dispatch refuses every caller-supplied Packet before append; manual review cannot supply import provenance; authentic review-packet/import controls pass. | E-TEST |
| C-F15 | FIXED | Both streams persist after child exit, frame hashes recompute, exit/truncation match; mutating checks refuse append in repository controls. | E-OUTPUT |
| C-F16 | FIXED | Exact-key decoder ignores case/Unicode fold variants recursively; independent trailing Verdict and isolation variants cannot approve or clear blockers. | E-AUDITOR |
| C-F17 | FIXED | Information-separator strings cannot approve; repository full-code-point whitespace parity passes. | E-CONTRACT |
| C-F18 | FIXED | Independent partial artifact at the exact expected path refuses without store change. Repository empty/symlink/intact reuse controls pass; final filename uses complete synced bytes. | E-LIFECYCLE |
| C-F19 | FIXED | Shells, wrappers, aliases and forbidden operation aliases refuse before append/execution; direct Capture negative controls pass in repository suite. | E-LIFECYCLE |
| C-F20 | FIXED | Exact decoder and report contract both reject invalid UTF-8; report/receipt/state paths share the decoder. Existing blocker-preservation fixture passes. | E-TEST |
| C-F21 | FIXED | printf %s sh runs, retains sh output and grants no criterion. Shell/wrapper entry points still refuse. | E-LIFECYCLE |
| C-F22 | FIXED | Independent shell-shebang, shell alias, env alias and xargs controls refuse. Repository nested/shebang/interpreter-alias controls pass. Arbitrary permitted programs spawning a shell remain a disclosed trusted-check limit. | E-LIFECYCLE |
| C-F23 | FIXED | Core lock is held during capture; direct terminal append is serialized after check and evidence. No check runs on the subsequently closed run. | E-AUDITOR |
| C-F24 | FIXED | Approved root and exact approved base are required; alternate HEAD-as-base, other repository and clone roots refuse before usable packet. | E-TEST |
| J8-01 | FIXED | Explicit one-time host repository binding precedes the first packet; unbound/second-binding/clone controls refuse. Host supplies initial identity mapping; this is not independent root attestation. | E-TEST |
| J8-02 | FIXED | Slash source IDs are rejected and final snapshot duplicate global IDs are rejected before append; exact owner field remains authoritative. | E-TEST |
| J8-03 | FIXED | Loop root, approved base, repository identity and journal repo.root must agree with the explicit run binding before execution. | E-TEST |
| J9-01 | FIXED | Canonical executable is judged and executed. Independent git/rm aliases refuse; repository curl/direct/absolute aliases also pass. | E-LIFECYCLE |
| J10-01 | FIXED | Real binary bind-repository emits nonempty GENERATE_REVIEW_PACKET_OR_IMPORT_LOOP_EVIDENCE. | E-AUDITOR |
| J10-02 | FIXED | Real binary loop-evidence emits review_enforcement and agrees with host show --mode. J11-02 is a different command. | E-AUDITOR |

## Evidence and reproduction

All source Git operations used `/usr/bin/git`. Go was `/home/yeli/.local/go/bin/go`, version `go1.27.1 linux/amd64`. GOCACHE and TMPDIR were under this output directory. [COMMANDS.json](COMMANDS.json) records commands, working directories, logs, and exit codes. [Auditor test source](auditor_negative_test.go) contains the independent mutations and assertions. The baseline test source is under `baseline/cmd/hma/auditor_export_test.go`.

| Evidence | Source | Observed result |
|---|---|---|
| E-FREEZE | `freeze-start.json`; `freeze-end.json`; `freeze-comparison.json`; `audited-file-manifest.json` | main at 527b7c5ff79090c558f5bbfed5ec87a85f06d1e5; status, full binary diff digest and all 30 untracked file hashes unchanged. |
| E-SCOPE | `requirements-sources.txt`; `derived-requirements.json`; `docs/handoffs/2026-10-16-gate-c-scope.md:1-86`; `/tmp/hma-gate-c-criteria/criteria-final11.json` | Authoritative objective and allowed design sections derived before diff inspection. Gate D boundary accepted by Lennon on 2026-10-17. |
| E-CODE | `audited-tracked.patch`; `cmd/hma/review.go`; `cmd/hma/loop.go`; `cmd/hma/repository.go`; `internal/host/review_import.go`; `internal/host/findings_import.go`; `internal/host/report_contract.go`; `internal/evidence/noshell.go`; `cmd/hma/transition.go` | Static path and authority review; source references use the frozen working tree, not bare HEAD. |
| E-TEST | `logs/go-test-copy.log`; `logs/go-vet.result.json`; `logs/go-race.log`; `logs/go-race.result.json` | All 12 packages pass go test -count=1 and go test -race -count=1. go vet exits 0. This is the repository suite, before addition of the auditor tests. |
| E-DOGFOOD | `logs/dogfood.log`; `logs/dogfood.result.json` | Real HMA and hostinput binaries; 17 synthetic checks pass. No worker/model starts, no actual hi, no authentic human approval. |
| E-AUDITOR | `auditor_negative_test.go`; `logs/auditor.log` | 9 independent top-level tests pass under -race: adverse report matrix, receipt/time/root/live target matrix, strength matrix, concurrent single-use import/newer packet, concurrent core terminal serialization, output, real-binary result controls, baseline migration and 82-case external parity. |
| E-BASE | `baseline/cmd/hma/auditor_export_test.go`; `logs/baseline-export.log`; `baseline-metadata.json`; `baseline-original-run.jsonl`; `logs/auditor.log` | Code exported from HEAD 527b7c5 writes synthetic baseline records. Current code loads their original core and host head anchors, labels legacy mode, migrates via generated packet, refuses manual override, resolves its own legacy finding and retains PENDING criterion. |
| E-CONTRACT | `logs/auditor.log`; `internal/host/gatec_r3_parity_test.go`; `logs/go-test-copy.log` | 82 independently selected required-field/type cases agree with the actual Python validator. Repository tests additionally cover full-range Python whitespace parity, canonical hashes, exact JSON keys and invalid UTF-8. |
| E-RESPONSE | `logs/result-contract.log`; `logs/result-contract.exit` | Exit 1: exactly two top-level response-contract tests fail, with three failing cases. No approval, blocker-clear, criterion-credit or terminal-execution bypass is demonstrated by these failures. |
| E-LIFECYCLE | `logs/lifecycle-shell.log`; `logs/lifecycle-shell.exit` | 8 shell/alias/corrupt-output/plain-data cases and independent code-repair lifecycle pass under -race. The journal green claim does not override re-capture failure. |
| E-OUTPUT | `logs/retained-output.log`; `retained-output-37.json`; `retained-evidence-37.json`; `retained-output-1048629.json`; `retained-evidence-1048629.json` | After the child check exits, raw stdout/stderr remain in retained artifacts. Length-framed digests match, exit is 7, and output above 1 MiB is explicitly truncated. |
| E-REPRO | `reproduce.sh`; `logs/reproduce.log`; `/tmp/hma-gate-c-20261017-final11/repro.6QkhnP/baseline.log`; `/tmp/hma-gate-c-20261017-final11/repro.6QkhnP/auditor.log` | Fresh baseline export and all 13 final auditor top-level tests re-executed under -race. Eleven pass. Only the two frozen response-contract tests fail; three failing cases. No DATA RACE diagnostic. |

The repository suite passes in all 12 packages. Full race tests also pass in all 12 packages. The final combined auditor reproduction uses a new scratch baseline and executes all 13 top-level auditor tests under `-race`. Eleven pass. Only J11-01 and J11-02 tests fail, with three failing cases. No race diagnostic appears.

```bash
/tmp/hma-gate-c-20261017-final11/reproduce.sh
```

This creates a new scratch directory under the output directory. Expected exit on the audited product is `1`. Final checked reproduction: `/tmp/hma-gate-c-20261017-final11/repro.6QkhnP`. The script never edits `/home/yeli/repos/HMA`. Synthetic approvals are test fixtures, not human authorization.

The retained output fixtures have 37-byte and 1,048,576-byte stdout respectively, plus 41-byte stderr. The second command emitted 1,048,629 stdout bytes, so truncation is explicit. Exit code is 7. The exported JSON artifacts and evidence records permit digest re-computation after the child check and Go test processes end. Initial auditor fixture mistakes and their corrected reruns are preserved in the logs; they are not product findings.

## Controlled unblinding

The first-phase SHA-256 was recorded as `bb9fccd7a352a4f39767244ddc348c60883ffd62550e6d4228fb034945753855`. Only then were scope §4a onward and TODO Gate C read. The later text disclosed the same no-shell and initial root-binding limits. It left reservations and the independent Gate C audit/commit open. It supplied no new execution evidence. It did not change either finding, the requirement ratings, the deferrals, or the verdict. No forbidden prior-audit temporary directory was opened.

## Remediation roadmap

| Priority | Change | Exit criterion |
|---|---|---|
| 1 | Preserve non-approving review verdict in findings-import guidance. | Both non-approving response cases return remediation or escalation; their gates remain closed. |
| 2 | Emit findings-import review_enforcement. | Real-binary result equals host show --mode. |
| 3 | Independently retest the bounded correction. | The two failing tests pass; safety, baseline migration, and repository checks remain satisfactory. |

No optional redesign is proposed. The smallest correction concerns response construction. Gate D work remains outside this correction.

## Limits and unresolved assumptions

- Context isolation is isolated for Gate C: fresh evaluator context, no implementer conversation and no prior Gate C verdicts. The user-authorized criteria file was read before Phase 1. Primary source/test comments and the allowed Gate A v7 document contain historical reviewer names and Gate A judgments; these are disclosed incidental author text, not evidence of Gate C correctness.
- Review access was procedural, not enforced by an operating-system read-only mount. No audited repository edits or Git writes were performed. The first test command mistakenly used the source cwd, with cache/temp paths in the output directory; its suite completed successfully. The required suite was independently repeated in the private copy. Start/end status, binary-diff SHA-256 and all untracked hashes match.
- Only the source content permitted for the blind pass was shown to the evaluator. Snapshot copying and freeze hashing handled file bytes mechanically. Before Phase 1, scope note content was displayed only with sed -n 1,86p; TODO Gate C and scope §4a+ were not inspected. No forbidden prior-audit temporary directory was read.
- Execution uses Linux and Go 1.27.1. Cross-platform behavior and actual model/reviewer reasoning were not tested. Receipts, route approvals and worker results are synthetic fixtures. No nested agent CLI or model API was invoked. The dogfood harness explicitly uses synthetic approvals and preflights.
- Trusted-host threat model applies. These checks detect accident and drift, not a malicious same-UID host. The initial root/opaque-repository-identity mapping is asserted by the host binding step; packet checks enforce consistency with it. Arbitrary permitted executables that internally launch shells cannot be recognized by argv-name inspection.
- Gate D leases/reservations/admission/namespace reconciliation/positive criterion credit/full contract packet inputs and Gate E/F final closure enforcement are not validated or claimed complete. Kernel/budget positive dispositions remain deferred. The report does not authorize stage progression or release.
- Auditor test setup errors were corrected without changing product code: an incorrect constant, an assumed sha256: prefix on the raw evidence digest, and nil-vs-empty stderr artifact bytes. Earlier execution logs are retained. Only corrected passing controls and the independently reproduced product-response failures support the verdict.
- Tests sample behavior and cannot prove the absence of all bypasses. Race checks found no data race in their exercised paths. Output paths created by t.TempDir are removed after tests; durable raw retained-output artifacts, logs, source and the reproduction script are separately kept.
- Controlled unblinding occurred only after the Phase 1 digest was recorded. Later scope/TODO assertions did not change the frozen findings. No prior audit directory was opened.

No critical or high finding was demonstrated. The two lower-severity defects still fail mandatory response contracts. This audit does not claim that a green fixture suite proves all safety properties.

## Required next action

**Lennon: authorize one bounded correction unit for J11-01 and J11-02.** The proposed envelope covers findings-import response construction in `cmd/hma/review.go` and focused tests. The human must approve the exact implementation and independent-review routes. This audit grants no approval. Stop and return to the human if the correction needs a change to provenance, reviewer eligibility, blocker handling, criterion credit, terminal checks, or broader scope.

