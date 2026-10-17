# Gate C independent audit — FAIL

Audit ID: `hma-gate-c-audit-20261017-final6`. Date: 2026-10-17. Modes: code and outcome.

Two high findings remain. HMA executes three explicit shell entry points. A trusted core terminal append can also land before a loop check starts. The existing test suite passes. The independent negative controls expose both defects.

Target: `main`, `527b7c5ff79090c558f5bbfed5ec87a85f06d1e5`, plus the frozen working tree. Binary diff SHA-256: `1805d9769b3ab5ab23a63b30aacf7a0a148418d1e49a8edabfcd638cbf837916`. All 29 untracked files were hashed. `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md` were excluded from evaluation. The end freeze is unchanged. See [start freeze](freeze-start.json), [end freeze](freeze-end.json), and [copy comparison](copy-source-verification.json).

Isolation: **isolated review context**, fresh user dispatch, with the disclosures below. This was procedural read-only access. The OS permission layer permitted repository writes. No target file or Git state changed. All tests, fixtures, output and Go caches stayed in this output directory.

The single verdict is **FAIL**. `AUDIT.json` uses the report-contract value `fix-required` for the same verdict. It does not authorize a stage transition, Gate C acceptance, execution, publication or closure.

## Scope and authority

The original objective is outcome-consolidation §5 Gate C and §6. Gate A v3 §§3–4 and §6, v4 §§3–5, v5 §§1–3, v6 §1 and §5, and v7 supply the approved design. The architecture, AGENTS and Task 14 supply the normative contract. Only the named v3–v6 sections were interpreted. The human decision dated 2026-10-17 accepts scope §0.1. [Source digests](source-digests.json) and [authorized excerpts](authoritative-design-excerpts.txt) retain provenance.

Reservation admission, consumption, replay, reconciliation and generations remain Gate D work. Positive criterion credit, reproduced BLOCK/WAIVABLE mapping and full acceptance-contract packet inputs also remain Gate D work. This audit checks their recorded exit tests and prevents them from counting as Gate C proof. It does not report these deferred features as implemented.

The review used the existing audit methodology and external interface contracts. It did not replace them. No worker, nested agent CLI or model API was started. Jev was unavailable in the exposed host tools. The auditor used the recorded orchestrator decision fallback.

## Requirement coverage

| ID | Requirement | Result | Evidence |
|---|---|---|---|
| C01 | Generate exact live-target, nonce-bound fixed review packets with approved route/plan, findings and waivers. | **MET** | E02, E06, E11, E12 |
| C02 | Import only receipt-bound current latest one-use review with exact route, artifact, run, launch interval, report hash and nonce. | **MET** | E06, E07, E12 |
| C03 | Enforce independent reviewer across retained implementations and family separation for final/disputed/high risk. | **MET** | E02, E04, E12 |
| C04 | Validate external audit contract without replacing audit methodology; total verdict mapping; final pass requires isolated. | **MET** | E06, E07, E08, E12 |
| C05 | Snapshots preserve blockers on nonapproval, isolate exact unit ownership, reject stale state, and allow authentic current approved resolution. | **MET** | E02, E06, E12 |
| C06 | Maintain baseline finding/waiver compatibility and explicit legacy/manual vs packet-import enforcement. | **MET** | E02, E05, E06, E12 |
| C07 | Re-capture verify-loop checks through exact-target no-shell evidence capture, retain output, refuse closed/missing runs before execution. | **PARTIAL** | E06, E09, E10, E13, E14 |
| C08 | No imported evidence or finding grants criterion credit, stage approval, launch or closure authority. | **MET** | E02, E05, E06, E11, E12 |
| C09 | Failure→linked repair→fresh re-verification fixtures; stale/no-report/self-review refusal; Gate B regression checks. | **MET** | E02, E04, E05, E06 |
| C10 | Record all human-deferred requirements and exit tests; make no Gate C claim depend on them. | **MET** | E11, E16 |

Nine requirements are MET. C07 is PARTIAL. Detailed observations and source mappings remain in [AUDIT.json](AUDIT.json).

## Independent execution evidence

| Check | Observed result | Raw evidence |
|---|---|---|
| `go test -count=1 ./...` | exit 0; 12 packages passed | [go-test.log](go-test.log), [go-test.json](go-test.json) |
| `go vet ./...` | exit 0; passed | [go-vet.log](go-vet.log), [go-vet.json](go-vet.json) |
| `go test -race -count=1 ./...` | exit 0; 12 packages passed | [go-race.log](go-race.log), [go-race.json](go-race.json) |
| `scripts/dogfood/host-dogfood.sh --keep` | exit 0; 17 synthetic CLI checks passed | [host-dogfood.log](host-dogfood.log), [host-dogfood.json](host-dogfood.json) |
| `Auditor suite: go test -race -count=1 -v -run ^TestAuditorFinal6 ./cmd/hma` | exit 1; failed: three loop shell forms and direct Capture shell script; 46 subtests passed | [independent-final.log](independent-final.log), [independent-final.json](independent-final.json) |
| `Core terminal schedule under -race` | exit 1; failed: check marker created after ABORTED; append refused | [core-terminal-schedule.log](core-terminal-schedule.log), [core-terminal-schedule.json](core-terminal-schedule.json) |
| `Host-lock-respecting terminal schedule under -race` | exit 0; passed: terminal writer waits; later capture refuses | [terminal-race.log](terminal-race.log), [terminal-race.json](terminal-race.json) |
| `Interface parity and all-codepoint whitespace` | exit 0; passed without skips | [interface-parity.log](interface-parity.log), [interface-parity.json](interface-parity.json) |

Environment: Go 1.27.1, Git 2.43.0, Python 3.12.3, Linux amd64. `git` resolved to `/usr/bin/git`. [Environment details](environment.json) retain cache paths and copy locations. The dogfood uses synthetic receipts and approvals. It does not prove a real provider launch or human authorization.

The auditor checked 11 retained report candidates with the external Python validator. The JSON checks preserved the existing blocker and closed the host gate for unsafe candidates. The valid report control resolved its own blocker. Case-variant extra keys can be contract-valid because Python ignores them. HMA must also apply its exact binding checks. See [validator results](external-validator-candidates.json) and [candidate bytes](candidates/).

## Findings and proposed remedies

### C-F22 — Executable shell scripts and wrapper aliases bypass the no-shell kernel

Severity: **high**. Confidence: **observed**. Fix status: **proposed**.

loop-evidence executes an absolute executable whose first line is #!/bin/sh, env plus a symlink to sh, and a symlink to env plus sh and a script. Each independent case creates a marker, returns success, and changes both core and host heads. evidence.Capture also directly executes the shell script. RejectShell checks names only, does not inspect the script interpreter, does not resolve wrapper operands, and does not use the resolved wrapper identity. This violates architecture §§7.1 and 14. It requires no same-UID tampering with stored authority.

- Evidence: `independent-final.json; independent-final.log; auditor-final6-suite-frozen.go.txt`. Auditor suite under -race: four failing leaf cases (three loop shell entry points and direct Capture); 46 passing subtests plus standalone checks. No data race warnings. Unmodified product source; independent tests added only to copy.
- Evidence: `internal/evidence/noshell.go:36-55; internal/evidence/capture.go:120-151; cmd/hma/loop.go:143-180`. Shell check uses executable names and argument basenames; no shebang inspection; wrapper operands not symlink-resolved. Successful Capture appends loop evidence.

Preferred fix: In internal/evidence/noshell.go, resolve the actual executable and any explicit command operand of supported wrappers. Reject shell interpreters reached by shebang (including env shebang forms) or symlinks before cmd.Run. Use that shared check for loop admission and Capture. For wrappers whose command operand cannot be identified safely, reject that wrapper at capture admission. Keep ordinary data arguments outside command-position checks.

Reason: The kernel must classify the explicit execution entry point, including its interpreter and wrapper command, rather than trust a filename.

- Verification: Run TestAuditorFinal6ShellEntrypoints and TestAuditorFinal6CaptureShellScript with -race; include #!/bin/sh, #!/usr/bin/env sh, symlinked command operands and aliased env. Fingerprint both chains and marker before request. Expected: All shell entry points refuse before marker creation and before either append; printf %s sh, a normal executable, and an allowed non-shell interpreter remain usable.

Trade-offs: Shell-based verification scripts must be replaced with explicit non-shell checks. Wrapper command parsing needs an explicit supported grammar. This is not a general sandbox for arbitrary descendant processes.

Alternative: Use a verified allowlist of native executables and explicit non-shell interpreters, and refuse command wrappers and executable scripts until their entry points can be classified. This is simpler but reduces compatibility.

Effort: medium. Priority: 1. Related prior findings: C-F19, C-F21. No remedy was implemented or validated.

### C-F23 — A concurrent trusted core terminal append can precede loop check execution

Severity: **high**. Confidence: **observed**. Fix status: **proposed**.

runHostLoopEvidence calls core Load, checks Terminal, then executes outside the core lock. The host lock serializes CLI transitions but does not serialize the trusted direct core append used in Gate B regression tests. An unchanged production path paused at its first snapshot Git call accepts a core ABORTED record. When resumed, it executes touch and creates the marker. Only afterwards does s.Append reject record chain discontinuity. No evidence or host record is accepted, but the check starts on a closed run. This fails A08 and the Gate B core/host ordering guarantee.

- Evidence: `core-terminal-schedule.json; core-terminal-schedule.log; probes/cmd/hma/auditor_final6_test.go:237`. Unmodified production path: Git helper pauses first snapshot Git call after core Load. Trusted bTerminal core append records ABORTED. Marker absent before release. After release the check creates marker; append fails record chain discontinuity; zero host records.
- Evidence: `cmd/hma/loop.go:115-143,162-169; cmd/hma/host.go:117-151; cmd/hma/gateb_r2_test.go:276-287; internal/store/jsonl.go:154-173,247-268`. Loop releases core lock after Load and holds only host lock during execution; other host append paths hold both locks; Gate B already tests trusted concurrent core appends.

Preferred fix: Add a core-store transaction API that loads and appends under one existing core run lock. In runHostLoopEvidence, acquire host then core lock; check terminal/route, capture, retain output, append core evidence, and commit the host binding before releasing the core lock. Use a locked append callback; do not call the current locking Append recursively.

Reason: The terminal check and execution admission must share the lock used by every trusted core append. A final optimistic predecessor check protects accepted records but cannot undo executed argv.

- Verification: Run TestAuditorFinal6CoreTerminalBeforeExecution under -race with the scheduling helper; also run TestAuditorFinal6TerminalAdmissionRace and an already-terminal/missing-run matrix. Expected: If terminal authority commits first, no marker is created. If capture admission locks first, the terminal append waits until capture and binding finish. Valid open-run capture completes without deadlock.
- Verification: Repeat the schedule with an approved route rewind instead of ABORTED, and run all Gate B and Gate C tests, vet and dogfood. Expected: A core revocation cannot land between authorization check and argv start; both chains remain coherent.

Trade-offs: Holding the core lock across a check delays terminal and resolution writes; enforce bounded command duration and cancellation in the existing execution envelope. A transaction API is needed to avoid recursive-lock deadlock. Cross-chain crash recovery remains a separate concern; this finding does not claim new corruption.

Alternative: Introduce a durable execution-admission operation that every core authority-revoking writer must honor. This can reduce lock hold time but adds state and is larger than the existing host-then-core transaction pattern.

Effort: medium. Priority: 1. Related prior findings: A08. No remedy was implemented or validated.

The shell reproducer creates a harmless marker outside the fixture repository. It also records core evidence and host loop evidence. It does not need forged receipts or altered stored records. The direct Capture reproducer proves that the shared evidence boundary has the same gap.

The terminal reproducer does not change production code. A helper executable pauses the first snapshot Git call and forwards the original arguments and output to `/usr/bin/git`. The test records `ABORTED` with the same trusted core append primitive used by Gate B. It confirms that the marker is absent. It then releases Git. The check creates the marker. The later evidence append fails. The separate host-lock control passes. This isolates the missing core lock from ordinary CLI transition serialization.

## Prior finding closure

| Prior ID | Status | Independent observation | Evidence |
|---|---|---|---|
| F01 | **FIXED** | Nonapproval preserves the stronger prior same-ID and omitted BLOCK/WAIVABLE. Safe snapshots may append the preserved result rather than reject the whole report. | E06, E12 |
| F02 | **FIXED** | Independent pre-review and pre-snapshot core finding, criterion, waiver and rewind drift refuse without changing heads. | E06 |
| F03 | **FIXED** | Latest one-use packet rule and concurrent import control prevent replay overriding later review. | E02, E06 |
| F04 | **FIXED** | HMA observes artifact; wrong declared artifact and wrong receipt root fail; live imports compare head/diff/worktree. | E02, E06, E12 |
| F05 | **FIXED** | Final issuing reason enters strong provider/model-family rule even without final_validation. | E02, E12 |
| F06 | **FIXED** | Own wrong/missing provider and effort matrix refuses; exact control imports. | E06 |
| F07 | **FIXED** | Contract-invalid candidates do not approve or clear blockers; malformed documents refuse. | E06, E07, E08 |
| F08 | **DEFERRED-BY-HUMAN** | Unverified references remain advisory and cited passes escalate. Positive reproduced BLOCK and measured WAIVABLE paths are approved Gate D work. | E02, E11, E12 |
| F09 | **FIXED** | Loop bridge independently recaptures passing/failing checks, binds state/attempts, retains output, and credits no criterion. Remaining runner violations are C-F22/C-F23. | E02, E06 |
| F10 | **FIXED** | Half-open review launch TTL enforced; own before-start/at-expiry controls refuse; valid receipt imports. Generation-wide reconciliation remains Gate D. | E02, E06, E11 |
| F11 | **FIXED** | Every implementation contribution is checked; full-suite retained-history tests pass. | E02, E12 |
| F12 | **DEFERRED-BY-HUMAN** | Reservations, replay, namespace association, revocation and generation rules retain Gate D exit tests. | E11 |
| F13 | **FIXED** | Run-wide packet mode rejects new manual overrides; compatibility remains explicitly legacy_manual. | E02, E06, E12 |
| A01 | **FIXED** | Own invalid report candidates preserve blocker and close host gate; external validator used. | E06, E07, E08 |
| A02 | **FIXED** | Independent tracked/untracked live drift refuses current snapshot; exact target control clears own blocker. | E06 |
| A03 | **FIXED** | UnitID ownership preserves U1/sub blocker when U1 snapshot imports; unowned records remain. | E06, E12 |
| A04 | **DEFERRED-BY-HUMAN** | Gate C packet includes core head, plan/route, findings and waivers; full acceptance-contract packet inputs are explicitly accepted Gate D deferral. | E02, E11, E12 |
| A05 | **FIXED** | Legacy source ownership is conservative; ambiguous records and active waiver preserved in own probe; valid approving review clears unambiguous own legacy finding. | E02, E06 |
| A06 | **DEFERRED-BY-HUMAN** | Reproduction-based disposition and exact-target independent criterion credit remain unchanged Gate D prerequisites. | E11 |
| A07 | **DEFERRED-BY-HUMAN** | Current envelope/store hash and hook checks work; reservation/namespace/reconciliation matrix moves to Gate D. | E02, E11 |
| A08 | **PARTIAL** | Already-terminal and missing runs refuse; CLI host-lock race control passes. Trusted core append race still starts check after ABORTED (C-F23). | E06, E09, E10 |
| A09 | **FIXED** | legacy_manual and packet_import mode exposed; manual caller packet refuses in both independent controls. | E02, E06, E12 |
| R3-01 | **FIXED** | New pending packet refuses old snapshot; own pending final closes gate and fresh completed/imported/snapshotted final reopens it. Concurrent duplicate import is one-use under -race. | E02, E06 |
| R3-02 | **DEFERRED-BY-HUMAN** | Full immutable request, criterion definitions, non-goals, budgets, required evidence and policy packet inputs accepted for Gate D. Existing core snapshot remains required. | E11, E12 |
| R3-03 | **DEFERRED-BY-HUMAN** | Lennon 2026-10-17 explicitly accepted the scope boundary. Deferred rows retain prerequisites and exit tests; no bounded-execution or closure proof claimed. | E11 |
| C-F14 | **FIXED** | Own legacy and post-failure packet-mode controls reject caller Packet before changing either chain or blocker. Genuine generated/imported/snapshot control works. | E06, E12 |
| C-F15 | **FIXED** | Own random-token stdout, distinct stderr, exit 7 and >1 MiB output are retained with explicit truncation and matching framed digest. | E06 |
| C-F16 | **FIXED** | Exact-key decoder ignores case variants; own verdict/long-s/binding and receipt matrix preserves blockers; full suite includes nested exact-key tests. | E02, E06, E08 |
| C-F17 | **FIXED** | Own separator-only candidate escalates; all 1114112 code points agree with Python isspace (29 whitespace code points). | E06, E07, E08 |
| C-F18 | **FIXED** | Own empty/partial/symlink artifact residue refuses before appends and preserves residue; intact control reused. Atomic temporary-write/link path inspected. | E06, E12 |
| C-F19 | **PARTIAL** | Named shells, direct shell symlink and forbidden loop operations refuse in existing tests. Three further explicit shell entry points still run and append (C-F22). | E02, E06, E13 |
| C-F20 | **FIXED** | Own invalid UTF-8 report and receipt refuse without clearing blocker; exact decoder rejects UTF-8 before parsing report/receipt/loop state. | E02, E06, E08 |
| C-F21 | **PARTIAL** | Plain printf %s sh succeeds. Full shell-entry-point negative control remains incomplete through C-F22. | E06, E13 |

Totals: 23 FIXED; 3 PARTIAL; 7 DEFERRED-BY-HUMAN; 0 OPEN. DEFERRED-BY-HUMAN is a scope decision. It is not implementation success. Mixed rows describe the fixed Gate C subset and the deferred Gate D subset explicitly. A08 remains PARTIAL through C-F23. C-F19 and C-F21 remain PARTIAL through C-F22.

## Remediation roadmap

| Sequence | Owner | Work | Exit test |
|---|---|---|---|
| 1 | Human implementation owner | Approve a bounded correction for C-F22 and C-F23. Keep loop-evidence out of accepted verification until corrected. | Exact correction unit and routes receive human approval. |
| 2 | Eligible implementer | Correct explicit execution-entry classification and core-lock transaction. | All shell and terminal negative controls refuse before execution; valid controls work. |
| 3 | Independent reviewer | Re-run these probes, full tests, vet, race and dogfood against the exact corrected target. | No unresolved high finding; fresh evidence; source freeze unchanged during review. |

No additional redesign opportunity is proposed. The two corrections target existing kernel rules.

## Phase 1 freeze and controlled unblinding

Requirements and hypotheses were recorded before the diff in [requirements-before-diff.json](requirements-before-diff.json). The initial combined output was truncated. Relevant source sections were then read in bounded calls.

Phase 1 was frozen at 2026-10-17T15:14:42.070759+00:00. SHA-256: `bae21e4de28174c927c58626c47fedcdd4023718d8ad82284e0cfbd49699d4d3`. [PHASE1.json](PHASE1.json) remains unchanged. [PHASE1.sha256](PHASE1.sha256) records the hash.

After that freeze, the auditor read scope §4a onward and only the Gate C section of TODO.md. Those sections contain implementer summaries and earlier Gate C verdict labels. They were treated as claims. The no-shell claim is contradicted by C-F22. No new execution evidence changed either finding. No forbidden prior audit directory was opened.

## Evidence ledger

| ID | Source | Observation |
|---|---|---|
| E01 | `freeze-start.json; start-status.txt; start-diff.binary` | Frozen main at 527b7c5ff79090c558f5bbfed5ec87a85f06d1e5; dirty tracked diff and every untracked file hashed. |
| E02 | `go-test.json; go-test.log` | go test -count=1 ./... exited 0 in audit copy (12 packages). |
| E03 | `go-vet.json; go-vet.log` | go vet ./... exited 0. |
| E04 | `go-race.json; go-race.log` | go test -race -count=1 ./... exited 0. |
| E05 | `host-dogfood.json; host-dogfood.log` | Synthetic CLI host dogfood exited 0; 17 checks; no actual worker or authentic human approval. |
| E06 | `independent-final.json; independent-final.log; auditor-final6-suite-frozen.go.txt` | Auditor suite under -race: four failing leaf cases (three loop shell entry points and direct Capture); 46 passing subtests plus standalone checks. No data race warnings. Unmodified product source; independent tests added only to copy. |
| E07 | `interface-parity.json; interface-parity.log` | External audit validator parity, Python canonical hash, all-codepoint whitespace parity, and actual ledger launch-digest parity passed without skips. |
| E08 | `external-validator-candidates.json; candidates/` | 11 retained report candidates independently checked with external validator. Verdict case variant and binding-only variant remain contract-valid, but exact HMA binding/verdict checks reject unsafe authority. Invalid UTF-8 and trailing JSON fail load. |
| E09 | `terminal-race.json; terminal-race.log` | Host-lock-respecting terminal controller waits for admitted check; later capture refuses. Positive control for shared host locking. |
| E10 | `core-terminal-schedule.json; core-terminal-schedule.log; probes/cmd/hma/auditor_final6_test.go:237` | Unmodified production path: Git helper pauses first snapshot Git call after core Load. Trusted bTerminal core append records ABORTED. Marker absent before release. After release the check creates marker; append fails record chain discontinuity; zero host records. |
| E11 | `authoritative-design-excerpts.txt; source-digests.json; docs/handoffs/2026-10-16-gate-c-scope.md:9-37` | Authorized design excerpts and human acceptance boundary. Deferred reservation/generation, criterion-credit, disposition-reproduction and full acceptance-contract packet inputs retained as Gate D prerequisites. |
| E12 | `cmd/hma/review.go:86-119,274-450; internal/host/review_import.go:116-230; internal/host/findings_import.go:68-151; cmd/hma/transition.go:43-114` | Static trace: latest/single-use packet, live target, core-head binding, exact route/receipt/report checks; same-review snapshot exception; other/unowned finding preservation; no import criterion credit. |
| E13 | `internal/evidence/noshell.go:36-55; internal/evidence/capture.go:120-151; cmd/hma/loop.go:143-180` | Shell check uses executable names and argument basenames; no shebang inspection; wrapper operands not symlink-resolved. Successful Capture appends loop evidence. |
| E14 | `cmd/hma/loop.go:115-143,162-169; cmd/hma/host.go:117-151; cmd/hma/gateb_r2_test.go:276-287; internal/store/jsonl.go:154-173,247-268` | Loop releases core lock after Load and holds only host lock during execution; other host append paths hold both locks; Gate B already tests trusted concurrent core appends. |
| E15 | `freeze-end.json; end-status.txt; copy-source-verification.json` | Target unchanged at end. Main and full HEAD match; status digest, binary diff digest and all 29 untracked hashes match. Baseline and probe product source bytes match target. |
| E16 | `docs/handoffs/2026-10-16-gate-c-scope.md:87-end; TODO.md:Gate C (read after PHASE1 sha256 freeze)` | No change to findings. Deferred Gate D rows stay recorded; scope still claims no-shell recapture, contradicted by independent tests. |

## Limitations

- Fresh user-dispatched auditor context; no implementer conversation or prior Gate C verdict was read. Only status-stripped prior criteria were used before Phase 1. Authorized v7 contains incidental historical Gate A judgments; primary source/test comments also contain author interpretations, which were not treated as proof.
- Repository was writable at the OS permission layer. Read-only was an audited behavior, not an OS-enforced mount. All audit writes and tests ran under /tmp/hma-gate-c-20261017-final6. Final HEAD, branch, status, binary diff SHA-256 and hashes of all 29 untracked files equal the start freeze. All 186 evaluated source/supplied-test files in audit copies match the source; see copy-source-verification.json.
- Tests use synthetic host receipts, routes and approvals. No worker CLI, nested agent, model API or live production approval was invoked. External interfaces were inspected as contracts; no real end-to-end worker launch was performed.
- Core terminal schedule uses the same trusted core Append fixture primitive as Gate B and a Git helper executable that only pauses and forwards unchanged arguments/output to /usr/bin/git. It does not alter production source. Host-lock-respecting CLI terminalization passes the separate positive control.
- Gate D deferred behavior was not tested as implemented. Gate C does not certify bounded execution, positive criterion credit, reproduced finding dispositions or closure.
- No approved Jev tool was exposed. Tool metadata search found none. Auditor/orchestrator decision fallback was recorded; no other model was presented as Jev.
- Initial combined source outputs were truncated by the tool. Relevant authoritative excerpts and code were re-read in bounded calls; raw authoritative excerpts and source hashes are retained. Full TODO Gate C and scope §4a onward remain unread before this freeze.
- Audit copies do not preserve real external side effects, production quotas or provider identity attestations. The trusted-host model excludes malicious same-UID forgery. Passing tests do not prove absence of all bugs.

The Git-helper terminal test covers a deterministic valid schedule. It does not estimate the probability of that schedule in production. Its failure uses direct trusted core append, which Gate B already treats as a supported concurrency boundary. The host-lock-respecting CLI path passed.

The final source file includes the terminal scheduling helper. [Frozen earlier suite source](auditor-final6-suite-frozen.go.txt) matches the source hash in `independent-final.json`. [Current probe source](probes/cmd/hma/auditor_final6_test.go) matches the source hash in `core-terminal-schedule.json`. Both tested source versions remain available.

## Required next action — REMEDIATE

Current outcome: **FAIL**. Rationale: two reproduced high kernel defects prevent Gate C acceptance. Proposed owner: Ming / human implementation owner. Proposed route: Proposed original implementer route: Ming / Anthropic / claude-opus-5-5. Effort, access service and eligibility require host verification before any new dispatch. No dispatch authorized by this report.

Permission envelope: A separately human-approved bounded change to evidence shell-entry classification and loop/core execution transaction, with tests. This audit remains read-only on /home/yeli/repos.

Exact approval required: Human approval of the specific corrective plan unit and exact implementer/reviewer routes before coding; no Gate C acceptance or stage advance follows from this audit.

Stop and escalation conditions: Stop on permission widening, stale approval, unavailable eligible independent reviewer, or repeat material failure. Escalate to the human.

**Next action: the human implementation owner must approve the bounded corrective unit for C-F22 and C-F23.**

