# D5a independent code and outcome audit

**Verdict: FAIL.** The report-contract JSON encodes this same decision as `fix-required`.

Two high findings remain. A `.git` file makes the fingerprint skip worktree files. The sandbox overlap check omits three directory mounts under `/etc`. Both gaps affect the exact-target prerequisite for criterion credit.

Sandbox runtime behavior is **UNVERIFIED BY YOU**. Bubblewrap cannot create its netlink socket in this runtime. Capture refuses the check. I did not remove the sandbox or run a criterion check unconfined. The observed code defects support a FAIL decision without treating that correct refusal as BLOCKED.

The initial full test and race runs failed. The corrected selected regression and race runs passed with explicit sandbox-dependent exclusions. `go vet ./...` passed. Baseline and target builds passed for `linux/amd64` and `darwin/arm64`. Exact commands, failures and exclusions remain in the linked logs and JSON records.

Prior findings: **21 FIXED; 13 PARTIAL; 0 OPEN**. PARTIAL does not grant closure. It includes remaining runtime verification and the two new counterexamples. Every supplied finding has an individual assessment below.

## Scope, isolation and frozen target

Audit ID: `hma-gate-d-audit-20261019-d5ar20p`. Dispatch metadata: `prof / gpt-6.1-sol / high`, review mode. Isolation: **isolated context**. I received the original objective and neutral closure criteria. I derived requirements before reading the diff. I did not inherit implementer conversation. Source comments and tests are artifacts, not proof. No nested agent, agent CLI or model API was used. Jev was unavailable; I used direct auditor judgment.

Target: branch `main`, HEAD `5f6f765517987fae8a573ff8ef6a0f27c3651f29`, plus the frozen uncommitted tree. Tracked binary-diff SHA-256: `f8a1546f7c9dd8481b9be0632067102b8c52bd0db15fa247c281b4b1d095b32e`. [freeze-start.json](freeze-start.json) contains status and every untracked-file digest. [freeze-end.json](freeze-end.json) and [freeze-comparison.json](freeze-comparison.json) confirm that HEAD, status, tracked diff and all untracked hashes are unchanged.

The audit covers D5a criterion credit and the verified-outcome guard. It checks shared evidence/repository code and D1–D4, Gate B and Gate C regressions. `.serena/` and the two architecture-amendment handoffs were excluded from semantic review. D5b findings disposition and D5c full-packet work remain deferred. This report approves no stage transition and claims no Gate D completion.

Blind findings were frozen in [PHASE1.json](PHASE1.json) before controlled unblinding. SHA-256: `9e0b023cd16d8792d3dc30d887b3a4d9b4adb4808652c29d01cbd941394fa8e5`. [requirements-before-diff.json](requirements-before-diff.json) retains the original requirements and failure hypotheses. Raw tracked changes remain in [diff.binary](diff.binary). The isolated source copy is `worktree/`; auditor-authored tests exist only in that copy. Git writes made by test fixtures remain under the audit output directory. The audited repository was not edited.

Authoritative sources: outcome-consolidation handoff §5 Gate D; Gate A v3 §4 lines 106–139 and §6 row 5; Gate A v5 §3 and v6 §1 review-import amendments; Gate C scope lines 16–35; architecture §§7, 9, 11, 14 and 18; AGENTS.md. Copies of the committed sources are in `baseline/` and `worktree/`. The current implementer handoff was withheld until Phase 1 was frozen.

## Requirement coverage

| ID | Requirement | D5a result | Observation and evidence |
|---|---|---|---|
| R1 | Credit requires HMA recapture passed AND isolated independent pass review listing criterion verified at exact approved target. | **NOT MET** | The conjunction rule exists, but exact-target recapture is not assured. F01 admits transient materialized bytes. F02 leaves live inputs exposed for repositories in omitted mount roots. Evidence: `auditor-gitfile.log`, `auditor-gitfile-race.log`, `auditor-negative.log`, `cmd/hma/credit.go:180`. |
| R2 | Credit and verified outcome bind current contract, lease generation, repository/base/head/diff/worktree; stale or mixed-target credit refused. | **PARTIAL** | Independent seeded guard controls pass. Target drift, mixed-target credit and CLI contract/lease reauthorization are refused. Fresh successful capture and fresh review after reauthorization remain unverified here. Evidence: `auditor-credit-corrected.log`, `auditor-additional.log`, `cmd/hma/credit.go:451`. |
| R3 | Self reports, missing/partial/escalated/stale/other-target or other-contract reviews and substituted report bytes cannot grant credit. | **MET** | Before-capture refusals cover partial/compromised review, blocked/unknown/conditional/fix-required verdicts, substituted report bytes and changed contract/generation. Latest-review, target and receipt checks were traced. Evidence: `auditor-credit-corrected.log`, `internal/host/review_import.go:158`, `internal/host/findings_import.go:129`, `cmd/hma/credit.go:100`. |
| R4 | Checks execute safe explicit argv at reviewed bytes; unreviewed inputs, git helpers/config/index/ref races, external paths, transient writes cannot manufacture PASSED. | **NOT MET** | Explicit Git forms, private validated config and pinned objects/index/ref controls pass. The gitfile copy-window and omitted mounted-directory cases fail. Evidence: `auditor-negative.log`, `auditor-race.log`, `auditor-gitfile.log`, `auditor-gitfile-race.log`. |
| R5 | Credit does not advance stages or broaden permission; outcome gate demands all current independently verified criteria and existing D1-D4/B/C safety. | **PARTIAL** | Selected full regression suite passes with explicit sandbox-dependent exclusions. Credit holds host then core lock and appends no stage approval. Verified terminal outcomes still fail closed in the core with OUTCOME_NOT_IMPLEMENTED. This audit does not prove full closure or filesystem-writer exclusion. Evidence: `regressions.log`, `regressions-race.log`, `internal/engine/stage_state.go:216`, `cmd/hma/credit.go:85`. |
| R6 | Evidence records actual argv/cwd/results/identity/digests/changed paths and integrity, and response has exactly one next action. | **PARTIAL** | Direct Capture cwd/output consistency and pure disposition/action controls pass. ChangedFiles and private-directory assignments were inspected. Persisted successful sandboxed credit evidence could not be independently produced. Evidence: `auditor-additional.log`, `auditor-credit-corrected.log`, `internal/evidence/capture.go:244`, `cmd/hma/credit.go:375`. |
| R7 | D5a claims only criterion-credit scope; findings disposition and full packet remain deferred; complexity justified by boundary. | **MET** | Code adds criterion credit and a closure guard. The handoff explicitly leaves D5b and D5c for later. It does not claim Gate D completed. No new dispatch or authority expansion is supplied. Evidence: `implementer-handoff.md:1`, `cmd/hma/credit.go`, `internal/host/host.go:265`. |

The core still returns `OUTCOME_NOT_IMPLEMENTED` for verified terminal outcomes (`internal/engine/stage_state.go:216`). The new host guard is additional refusal logic. A passing seeded guard test does not prove a full approved closure. A Go race-detector pass also does not prove that filesystem snapshots are atomic.

## Independent findings

### D5AR20P-F01 — A .git file stops the fingerprint walk and admits transient copy bytes

Severity: **HIGH**. Confidence: observed defect. Fix status: proposed, not validated.

Fingerprint returns filepath.SkipDir for .git without checking whether it is a directory. For a Git gitdir file, WalkDir skips the remaining entries in its parent directory. Gitfile repositories are accepted for non-git criterion checks. A writer can edit an existing tracked file after fpBefore, let Materialize copy that file, and restore it before the final Snapshot and Fingerprint. Both guards then equal their initial values while the private copy contains unreviewed bytes. This violates the exact-reviewed-target prerequisite for positive criterion credit. Host and core locks do not lock repository writers.

- `internal/repostate/fingerprint_linux.go:42`: The .git case returns SkipDir for every file type.
- `cmd/hma/credit.go:173,180,191,198`: The only live-copy consistency checks are before/after Snapshot and Fingerprint around captureOnTarget.
- `auditor-gitfile.log; auditor-gitfile-race.log`: All 3 ordinary and all 5 race-enabled gitfile repetitions report snapshot_equal=true fingerprint_equal=true materialized="UNREVIEWED\n". Every .git-directory control reports fingerprint_equal=false.
- `auditor-gitfile-host.log`: The real host fixture binds a gitfile repository, imports an isolated pass report, and reaches the sandbox probe for a non-git check.

Verification limit: No criterion executable was run unconfined. Materialize and the acceptance guards were reproduced directly. A PASSED append exploiting this gap is inferred from caller wiring, not observed, because bubblewrap cannot start here.

Preferred fix: In Fingerprint, return filepath.SkipDir only when the .git entry is a directory. Return nil for the excluded .git file or symlink so traversal continues to its siblings. Add regression coverage for accepted gitdir-file repositories and transient changes during the materialization window.

Reason: WalkDir uses SkipDir on a non-directory to skip its remaining siblings. Continuing the walk restores the file-metadata evidence on which credit relies.

Acceptance checks:

- Run TestAuditorGitfileTransientCopy repeatedly and under -race after the change. Add a no-writer reviewed-byte copy control. Expected: Every transient gitfile edit changes the fingerprint or refuses credit. The no-writer copy contains REVIEWED bytes and its guard remains equal.
- On a host with bubblewrap, bind a gitfile repository and import a pass review for a check that only succeeds on transient unreviewed tracked bytes. Schedule the writer around Materialize. Expected: No run records PASSED. An unchanged reviewed target can still credit.

Trade-offs: The minimal fix preserves Git gitdir-file support. Metadata-based detection still depends on the supported filesystem semantics. It does not prevent writers.

Alternative: Refuse non-directory .git entries before all criterion checks, including non-git commands. This is a simpler containment policy but rejects linked worktrees and separate-git-dir checkouts. A content-verified immutable materialization would provide a stronger boundary at greater implementation cost.

Effort: **small**. Repair within the existing D5a boundary, then freeze the new target for independent retest.

### D5AR20P-F02 — Overlap refusal omits directory mounts in systemFiles

Severity: **HIGH**. Confidence: observed defect. Fix status: proposed, not validated.

sandboxCommand rejects live roots under systemDirs, then binds all existing systemFiles entries without that check. systemFiles contains directories: /etc/ssl, /etc/alternatives, and /etc/ld.so.conf.d. A repository rooted within one of these directories would remain readable beside the private target. A read-only bind prevents writes but still exposes ignored or other unreviewed live inputs. The generic live-path string check does not cover aliases or computed paths.

- `internal/evidence/sandbox.go:27,30,52,57,67`: The overlap list is systemDirs, while the bind list also includes systemFiles directories.
- `auditor-negative.log; auditor-race.log`: /usr/share is rejected as a system-directory overlap. Each of the 3 /etc directory cases reaches the probe instead; all 3 race-enabled repetitions reproduce the omission.

Verification limit: The policy function was tested against existing directory paths. No repository was created in /etc and no system file was changed. Actual mount visibility and false credit on a functioning sandbox are inferred from bind arguments and remain unverified here.

Preferred fix: Validate repository overlap against the resolved source of every planned bind mount before the probe. Include systemFiles entries that are directories. Reject a mount whose directory contains the live root or exposes a subtree of it. Keep the executable-tree overlap check and test symlink spellings.

Reason: All host paths exposed by the sandbox must be subject to the same live-repository separation rule; the variable name systemFiles is not a filesystem-type check.

Acceptance checks:

- Extend TestAuditorMountedDirectoryOverlap with the existing /etc directories, a live /etc ancestor case, and symlink aliases. Keep /usr/share and an ordinary home/tmp checkout control. Expected: Every live-directory overlap is refused before the bubblewrap probe. The ordinary checkout reaches the probe.
- On a functioning sandbox host, create an isolated fixture under an approved writable mounted-source directory and attempt to read an ignored live input through an alias. Expected: Policy refuses before any criterion starts. No ignored live input produces PASSED.

Trade-offs: Refusal limits repositories in runtime dependency directories. Replacing whole-directory binds with a minimal runtime image is broader and may affect toolchain compatibility.

Alternative: Use a fixed minimal runtime image or selectively hide the live subtree after binding dependencies. This avoids some location restrictions but adds mount-order and dependency complexity. Fail-closed overlap refusal is the smaller change.

Effort: **small**. Repair within the existing D5a boundary, then freeze the new target for independent retest.

## Reproduction and evidence ledger

Run from the output directory. `run-check.py` uses `/home/yeli/.local/go/bin/go`, `/usr/bin/git` via its fixed PATH, and audit-local cache, temp and HOME directories. Corrected runs set `GOFLAGS=-buildvcs=false` because the copied tree has no `.git` metadata. The local verify-loop script was copied into the isolated HOME for the hook tests. No network or external dispatch was used.

```sh
python3 run-check.py reproduce-gitfile /home/yeli/.local/go/bin/go test -v -count=3 ./internal/repostate -run '^TestAuditorGitfileTransientCopy$'
python3 run-check.py reproduce-mounts /home/yeli/.local/go/bin/go test -v -count=1 ./internal/evidence -run '^TestAuditorMountedDirectoryOverlap$'
```

Both commands should fail on this target. The gitfile control must detect changed metadata for a normal `.git` directory. The mount control must refuse `/usr/share` before the probe. These commands exercise Materialize and policy guards; they do not disable the sandbox.

| Evidence | Result | Qualification |
|---|---|---|
| [go-test.log](go-test.log); [go-test.json](go-test.json) | FAIL | Full original suite. Sandbox refusal, source-copy VCS build errors, and temp-directory mismatch are retained. |
| [go-race.log](go-race.log); [go-race.json](go-race.json) | FAIL | Full original race run. Same environment failures. No DATA RACE warning observed. |
| [go-vet.log](go-vet.log); [go-vet.json](go-vet.json) | PASS | All production packages. |
| [regressions.log](regressions.log); [regressions.json](regressions.json) | PASS | First corrected selected run. Its exact exclusions are in the JSON. Later auditor tests were added after this run compiled. |
| [regressions-race.log](regressions-race.log); [regressions-race.json](regressions-race.json) | PASS | Corrected selected race run. Excludes 15 sandbox-dependent credit tests, one sandbox-runtime test, and the separately executed auditor tests. |
| [regressions-final.log](regressions-final.log); [regressions-final.json](regressions-final.json) | PASS | Final selected ordinary run, with the same exclusions as the selected race run. |
| [auditor-negative.log](auditor-negative.log); [auditor-negative.json](auditor-negative.json) | FAIL | Pinned-object/index/ref controls, Git forms/config, materialization/fingerprint and fail-closed control pass. Three omitted mount-directory cases fail. |
| [auditor-race.log](auditor-race.log); [auditor-race.json](auditor-race.json) | FAIL | Independent mount omission repeats 3 times. Other selected auditor controls pass. |
| [auditor-gitfile.log](auditor-gitfile.log); [auditor-gitfile.json](auditor-gitfile.json) | FAIL | Gitfile transient-copy gap repeats 3 times; normal-directory controls detect the edit. |
| [auditor-gitfile-race.log](auditor-gitfile-race.log); [auditor-gitfile-race.json](auditor-gitfile-race.json) | FAIL | Same gap repeats 5 times under -race. |
| [auditor-gitfile-host.log](auditor-gitfile-host.log); [auditor-gitfile-host.json](auditor-gitfile-host.json) | PASS | Gitfile root and isolated pass review reach the actual host sandbox probe; no successful capture claimed. |
| [auditor-credit-corrected.log](auditor-credit-corrected.log); [auditor-credit-corrected.json](auditor-credit-corrected.json) | PASS | Before-capture review/report refusals, CLI contract reauthorization, seeded guard and pure action controls. |
| [auditor-additional.log](auditor-additional.log); [auditor-additional.json](auditor-additional.json) | PASS | Mixed-target seeded guard and direct actual-cwd evidence controls. |
| `cross-{linux,darwin}-{baseline,target}.json` | PASS | Both target and immutable baseline compile for both requested platforms. |

The full outputs are retained without summarizer truncation. Some program output is governed by HMA’s own output limit. `checks-inventory.json` lists actual test FAIL markers separately from words such as the valid disposition `FAILED`. One initial auditor CLI fixture wrongly populated head/diff fields at an earlier stage. It was corrected. The invalid fixture failure remains in `auditor-credit.log`; it is not a product finding.

## Prior-finding closure decisions

FIXED means that inspected code and the cited independent execution support the bounded code closure. It does not mean that this auditor ran a functioning sandbox. PARTIAL means the full supplied closure check remains incomplete or a counterexample remains. No implementer pass statement is used as independent proof.

| Prior ID | Decision | Evidence and remaining limit |
|---|---|---|
| `D5A-F01` | **PARTIAL** | CLI changed-contract reauthorization refuses old review and seeded old credit. Fresh-review credit control remains unverified. Evidence: `auditor-credit-corrected.log`. |
| `D5A-F02` | **PARTIAL** | Seeded unchanged guard passes; stale target and mixed-target criteria refuse. Full capture controls remain unverified. Evidence: `auditor-credit-corrected.log`, `auditor-additional.log`. |
| `D5A-F03` | **FIXED** | Resolved-directory containment refuses external/dangling links and accepts internal/plain directories; direct cwd matches pwd. Evidence: `auditor-additional.log`, `regressions.log`. |
| `D5A-F04` | **PARTIAL** | Pure passing/failing/unlisted disposition and one-action rules pass. Successful persisted credit response remains unverified. Evidence: `auditor-credit-corrected.log`, `cmd/hma/credit.go:270`. |
| `D5AR2-F05` | **FIXED** | Only harmless exact global options precede a listed Git form; repository/config selectors refuse. Evidence: `auditor-negative.log`, `regressions.log`. |
| `D5AR3-F06` | **FIXED** | Aliases, mutating and unlisted subcommands refuse. Safe listed built-ins retain controls. Evidence: `auditor-negative.log`, `regressions.log`. |
| `D5AR4-F07` | **FIXED** | Exact option spellings reject abbreviations, write/program options and unlisted bundles. Evidence: `internal/evidence/gitforms.go`, `regressions.log`. |
| `D5AR4-F08` | **FIXED** | Patch forms require both helper-disabling flags; private config/env also remove launchers. Evidence: `internal/evidence/gitforms.go`, `regressions.log`. |
| `D5-GITFORMS` | **FIXED** | One approved-form implementation is shared by Capture, host credit and loop evidence. Evidence: `cmd/hma/loop.go:491`, `internal/evidence/capture.go:163`, `regressions.log`. |
| `D5AR5-F09` | **PARTIAL** | Direct unsafe-config refusal and helper prevention pass. Successful host credit control is unverified. Evidence: `auditor-negative.log`, `regressions.log`. |
| `D5AR6-F10` | **FIXED** | All -C check forms refuse; canonical direct-capture controls and nested-repo refusal pass. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go`. |
| `D5AR6-F11` | **FIXED** | Git itself parses copied config key names; comment-backslash attacks refuse, with standard config controls. Evidence: `auditor-negative.log`, `regressions.log`. |
| `D5AR6-F12` | **FIXED** | Direct Capture rejects filter configuration before the filter runs. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:35`. |
| `AUD-D5A-01` | **PARTIAL** | Directory-.git transient controls detect edits; gitfile transient materialization bypasses both guards (new F01). Evidence: `auditor-gitfile.log`, `auditor-gitfile-race.log`. |
| `AUD-D5A-02` | **FIXED** | Direct Capture applies approved forms and refuses -C, --git-dir and -c. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:116`. |
| `AUD-D5A-03` | **FIXED** | Canonical Git detection, form/config enforcement and recorded canonical executable are exercised by direct alias tests. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:116`. |
| `D5AR9-F01` | **PARTIAL** | Root entry create/remove detection exists. Empty-directory changes below a gitfile root are not covered because traversal stops (new F01). Evidence: `internal/repostate/fingerprint_linux.go:42`, `regressions.log`. |
| `D5AR9-F02` | **FIXED** | The config bytes checked are the private config bytes used; direct race-hook and control tests pass. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:175`. |
| `D5AR10P-F01` | **PARTIAL** | Ignore rules do not narrow the walk in normal repos. The gitfile traversal defect still skips ignored and unignored subtrees (new F01). Evidence: `auditor-negative.log`, `auditor-gitfile.log`. |
| `D5AR11P-F01` | **PARTIAL** | Materialize omits ignored inputs and preserves dirty bytes. Actual sandbox behavior is unverified; mounted-root omission still allows ignored live inputs (new F02). Evidence: `auditor-negative.log`, `auditor-race.log`. |
| `D5AR12P-F01` | **PARTIAL** | Absolute/climbing links and literal live-root argv refuse. General sandbox read/write containment remains unverified, with new F02. Evidence: `regressions.log`, `internal/repostate/materialize.go`, `auditor-negative.log`. |
| `D5AR12P-F02` | **PARTIAL** | Actual private cwd and ChangedFiles assignments exist; direct evidence cwd checks pass. Persisted sandbox-credit pwd/subdirectory/dirty variants remain unverified. Evidence: `auditor-additional.log`, `cmd/hma/credit.go:375`. |
| `D5AR13P-F01` | **PARTIAL** | Sandbox design addresses aliases but runtime remains unverified; new F02 is a live-root exposure case. Evidence: `internal/evidence/sandbox.go`, `auditor-negative.log`. |
| `D5AR13P-F02` | **PARTIAL** | Composed symlink containment relies on the OS sandbox; actual sandbox behavior is unverified. Absolute/climbing links have static refusal. Evidence: `internal/repostate/materialize.go`, `internal/evidence/sandbox.go`. |
| `D5AR13P-F03` | **FIXED** | Pinned view has detached HEAD and empty refs; an added live branch is absent in independently queried private view. Evidence: `auditor-negative.log`. |
| `D5AR14P-F01` | **FIXED** | Private object store contains only objects reachable from the pinned commit; independently added unrelated object is absent while HEAD:a succeeds. Evidence: `auditor-negative.log`. |
| `D5AR14P-F02` | **FIXED** | Private index comes from reviewed HEAD; live staged phantom is absent while reviewed path succeeds. Evidence: `auditor-negative.log`. |
| `D5AR14P-F03` | **FIXED** | Both baseline and target compile for darwin/arm64 and linux/amd64. Non-Linux Fingerprint returns an explicit unsupported error in source. Evidence: `cross-darwin-baseline.json`, `cross-darwin-target.json`, `cross-linux-baseline.json`, `cross-linux-target.json`, `internal/repostate/fingerprint_other.go`. |
| `D5AR15P-F01` | **FIXED** | Sandboxed Git refuses StagedTarget before preparation or probe. Snapshot binds staged entries. Unsupported staged checks fail closed. Evidence: `regressions.log`, `internal/evidence/capture.go:178`. |
| `D5AR15P-F02` | **FIXED** | Pinned config is fixed; filemode-mode-change control remains different after local config changes. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:269`. |
| `D5AR16P-F01` | **FIXED** | Preparation uses private config and no live-index preparation command; post-validation live-helper mutation test passes. Evidence: `regressions.log`, `internal/evidence/gitsafe_test.go:305`. |
| `D5AR17P-F01` | **FIXED** | Capture uses the bound snapshot Staged flag, not a preparation-time live index read. Direct staged-target refusal is tested. Evidence: `regressions.log`, `cmd/hma/credit.go:180`, `internal/evidence/capture.go:178`. |
| `D5AR18P-F01` | **FIXED** | Staged and staged digest entries use one final index output. Added/removed/mode/object/conflict cases and snapshot controls pass. Earlier path discovery still has separate reads; no new index bypass was demonstrated. Evidence: `regressions.log`, `internal/repostate/snapshot.go:285`, `internal/repostate/materialize_test.go:111`. |
| `D5AR19P-F01` | **PARTIAL** | Listed /usr,/bin,/lib directory roots are refused, but directory sources in systemFiles are omitted (new F02). Evidence: `auditor-negative.log`, `auditor-race.log`. |

`AUDIT.json` preserves every original closure criterion alongside its assessment. The isolated guard tests seed trusted credit metadata. They do not simulate successful capture. The CLI reauthorization test uses synthetic approval fixture actors and checks the real transition command; those actors are not human approvals.

## Controlled unblinding

After hashing Phase 1, I read only the permitted current handoff and host-result text. Copies are [implementer-handoff.md](implementer-handoff.md) and [implementer-host-sandbox-results.txt](implementer-host-sandbox-results.txt). The handoff confirms that D5b and D5c are deferred. The host text lists passing sandbox/credit tests. It has no command, frozen target digest or environment manifest in the supplied text. I label it implementer evidence, not my evidence and not proof. It does not exercise either new finding. The blind findings and FAIL decision are unchanged.

## Remediation sequence and limits

Repair F01 first. Restore fingerprint traversal or refuse gitfile repositories for every credit check. Repair F02 next. Derive overlap checks from every mount source. Then freeze the corrected target and obtain an independent retest. Include a host where bubblewrap can start. Retain the new negative controls and the unchanged-target positive controls. D5b and D5c remain outside this task.

No unrelated redesign or additional feature is recommended. The smallest boundary repairs above address the observed defects. Proposed fixes have not been applied or validated.

- Sandbox runtime behavior is UNVERIFIED BY YOU. Bubblewrap fails closed: NETLINK_ROUTE socket Operation not permitted. No sandbox was removed and no sandboxed criterion was rerun unconfined.
- No complete successful host-credit capture, false-PASSED exploit, or approved verified terminal outcome was produced by this auditor. The core currently refuses verified terminal outcomes with OUTCOME_NOT_IMPLEMENTED.
- The gitfile reproduction deterministically schedules a writer during the real Materialize window, outside a complete sandboxed Capture. It demonstrates unreviewed copied bytes and equal actual guards. The resulting false credit is an inference from caller wiring.
- Mounted-directory overlap tests call the real sandbox policy with existing /etc directories as hypothetical live roots. They do not create or mutate /etc repositories or observe a successful mount namespace.
- Outcome-guard positive controls seed trusted credit metadata. They test the guard, not recapture provenance. Synthetic approval actors are fixture data, not human approvals.
- The initial full test/race runs failed. Besides bubblewrap refusals, copied-source builds lacked VCS metadata and GOTMPDIR differed from TMPDIR. Corrected regression runs use GOFLAGS=-buildvcs=false and matching temp directories. The logs retain initial failures.
- One initial auditor fixture supplied head/diff fields at a stage that forbids them. The fixture was corrected. auditor-credit.log retains that failure; auditor-credit-corrected.log is the valid result.
- The host evidence was read only after Phase 1 was frozen. It is implementer evidence, not this auditor's result and not proof. It lacks a command, target digest and environment manifest in the supplied text.
- No external service, agent CLI, model API or nested reviewer was invoked. Jev was unavailable. The configured prof / gpt-6.1-sol / high route is user-supplied dispatch metadata, not independently introspected provider provenance.
- The ignored .serena directory and architecture-amendment handoffs were not semantically audited. Their untracked hashes are recorded for state verification. The current D5a handoff was withheld until controlled unblinding.

The report JSON was checked with the independent-audit `validate_report.py` script. [report-validation.log](report-validation.log) records the structural result. Structural validity does not prove findings, isolation or remediation. [artifact-manifest.json](artifact-manifest.json) records report and evidence digests.

## One required next action — architecture §18

**RETURN_D5A_FOR_REMEDIATION**. Owner: HMA implementer / human coordinator. Two observed high defects violate exact-target criterion credit. This audit grants no Gate D completion or stage approval.

Route: Use the existing human-approved implementation route. Resolve worker/provider/model/effort/access service and run the exact-route hi preflight before any later dispatch; no new dispatch is authorized or performed by this audit.

Permission envelope: A later implementer may change only the approved D5a scope and add regressions. Preserve the deterministic kernel, approval binding, route coherence and independent review. This audit itself remains repository-read-only.

Exact approval required: No transition, commit, push, release or scope expansion approval is requested. Human authorization is required for any change beyond the existing implementation envelope.

Stop and escalation conditions: Stop any out-of-envelope change or unavailable exact route. After correction, require an independent retest of the frozen new target. Do not claim Gate D complete while D5b/D5c remain deferred.
