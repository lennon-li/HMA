# Gate B independent closure audit — r5

**Verdict: FAIL.** Report-contract value: `fix-required`.

HMA accepts a changed tracked directory structure as the same artifact. CI appends revision-labelled evidence for that structure. An old independent review also permits Verification. The optional approval worktree binding does not stop this case.

The supplied suite, vet, race checks, and the 17-check dogfood harness pass. The independent probes expose the blocker. Host/core commit ordering and directory durability pass the stated finite models.

The audit covers 47 defect IDs: 38 FIXED and 9 PARTIAL. Three PARTIAL rows are the human-deferred dependency-graph work. Five PARTIAL rows share the new artifact-identity blocker. M3 is PARTIAL because its final/disputed subcheck remains unproved. No original defect is silently waived.

## Scope and independence

- Dispatch: `hma-gate-b-closure-blind-20261016-r5`. Modes: code and outcome.
- Target: `main`, `67965541b7032a585a68e48f3d10f4a998d5399e`, plus the frozen uncommitted worktree.
- Binary diff SHA-256: `4c3ffeb80e10c136e9321d6c54ef40e512bfa590210c4b8720220cd879847b7a`.
- Phase 1 SHA-256: `58cc9c8285de399757f2ca286800e8609dc726d4506e169515079d2b60f8a818`. Frozen at 2026-10-17T01:06:35.531302+00:00.
- Isolation: **isolated context**. This fresh session had no implementer conversation or prior closure reports. Required original defect sources and source/test comments were treated as evidence.
- Assigned auditor: Liz / gpt-6.1-sol / high, per the user dispatch. Runtime model identity was not independently attested.
- Access: procedural read-only access to the target. The OS sandbox allowed writes, but this auditor made none under `/home/yeli/repos`. All tests and changes ran in the output copy. No nested agents or model APIs ran.
- Exclusions: `.serena/` and the two `docs/handoffs/2026-10-16-arch-amendment-*.md` drafts. Only their requested freeze hashes were recorded.
- Environment: Go 1.27.1 linux/amd64; Git 2.43.0; Python 3.12.3. GOCACHE and both temporary directories are under this packet.
- Threat model: trusted host. A same-UID adversary is outside scope.
- Final freeze: HEAD, branch, status, binary diff hash, and all untracked hashes match. The copy matches all 142 audited source paths. See `FREEZE-start.json`, `FREEZE-end.json`, and `VERIFY-FREEZE.json`.

## Original objective and scope decision

Gate B in `docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md` §5 requires existing defect remediation. It names crash/restart, concurrency, evidence/stage projection, exact approval binding, invalidation, terminal coherence, and host dispatch/review legality. Normative sources are `docs/architecture.md`, `AGENTS.md`, Task 13, and Task 14. `REQUIREMENTS.json` records the derived checks before the diff review.

The source set includes H1–H4, M1–M6, L1, and L3 from the original audit. Only N1/N2/M3 closure criteria were extracted from Gate A JSON. The neutral v5 criteria supply B1–B18 and the later closure IDs. The supplied human decision defers plan-derived dependency graphs to Gate D. Conservative all-unit invalidation must remain in force.

## Requirement coverage

FIXED means the scoped defect has passing executable/static evidence. It does not assert full host-MVP or release conformance. PARTIAL means a closure condition remains unmet or deferred. Evidence keys point to the ledger below.

| ID | Decision | Evidence | Observation |
|---|---|---|---|
| H1 | FIXED | E01, E13 | CI projects current stage and rejects approval packets from earlier visits. Supplied transition/resolution cases pass. |
| H2 | FIXED | E01, E11 | Grounding and Planning captures leave head/diff approval fields empty and succeed. |
| H3 | FIXED | E01, E09, E10, E12 | Host locking, pending recovery, crash images, ancestor-fsync refusal, and same/different-run writers pass. |
| H4 | FIXED | E01, E08, E12 | All five supported terminal outcomes refuse serial host paths. Valid core API ABORTED ordering passes for all four host commands. |
| M1 | FIXED | E01, E13 | Capture checks store isolation, including symlinks into the checkout. |
| M2 | FIXED | E01, E13 | Resolve uses strict EOF decoder; multiple documents and garbage refuse without record mutation. |
| M3 | PARTIAL | E01, E11, E13 | Lower-tier and tied-rule bypasses are closed. All applicable constraints are conjoined. The literal final/disputed subcheck has no independently demonstrated entry point or explicit purpose input. |
| M4 | FIXED | E01, E08, E13 | Second coding gate checks current core route approval. Revoked review import refuses. |
| M5 | FIXED | E01, E04, E13 | Marker binds canonical scratch store and run. Exact seed JSON excludes all changed payload/metadata. Marker and repeat-seeding negatives pass. |
| M6 | FIXED | E04, E13 | Harness hashes all non-lock store filenames and bytes on every refusal. |
| L1 | FIXED | E01, E13 | RouteAttestation structural positive/negative tests cover fields, classes, optional absence, and execution vocabulary. |
| L3 | FIXED | E04, E13 | CI workflow invokes scripts/dogfood/host-dogfood.sh. |
| N1 | PARTIAL | E06, E07 | Byte/mode/index changes refuse old reviews, but ignored parent-directory symlink replacement preserves identity and advances in both binding forms. |
| N2 | PARTIAL | E01, E06, E07, E08 | Repository, root, head/diff and ordinary mutation refusals pass; parent-directory symlink changes are still labelled clean and CI appends evidence. |
| B1 | PARTIAL | E01, E06, E07 | Ordinary dirty A-to-B and optional-binding cases pass; changed parent directory structure still authorizes through an A review. |
| B2 | PARTIAL | E01, E06, E08 | Index flags and normal command mutation refuse; pre-existing and command-induced parent symlink substitution still append revision-labelled evidence. |
| B3 | FIXED | E01, E08, E11 | CI refuses selectors. Shared Capture strips selectors, pins original object view, and preserves non-selector configuration. |
| B4 | FIXED | E01, E13 | A populated approval worktree binding refuses a now-clean tree. |
| B5 | FIXED | E01, E08, E12 | Host transaction holds core lock through durable commit; terminal-first writes refuse, host-first writes bind the preceding head. |
| B6 | FIXED | E01, E12 | Conflicting initial decisions serialize against committed history; supplied 16-writer decision test accepts exactly one. |
| B7 | FIXED | E01, E09, E10, E12 | Own directory-entry crash images and fsync failure model pass; negative control detects missing durability. Existing recovery and writer controls pass. |
| B8 | FIXED | E01, E11 | Tied thresholds and non-risk-ordered capabilities cannot weaken the conjunction. Valid separated controls pass. |
| B9 | FIXED | E01, E13 | Guard compares every serialized seed field with seedRecord after clearing only the computed head anchor. |
| B10 | FIXED | E01, E08, E13 | Conservative all-unit invalidation blocks old dependent authority after early rewind; review import cannot revive it. Plan-derived precision is deferred by the supplied human decision. |
| B11 | PARTIAL | E05, E06, E07 | Owner execute, leaf symlink/type, deletion and byte cases pass. Intermediate directory symlinks are followed and can preserve exact artifact identity. |
| B12 | FIXED | E01, E05, E06 | Raw owner execute check detects core.filemode=false changes; full old-review cases refuse in both optional-binding forms. |
| B13 | FIXED | E01, E05, E08 | Raw-byte observation detects normalization-hidden tracked content and command mutation. Attribute fixtures in supplied suite pass. |
| B14 | FIXED | E01, E13 | Current HEAD/index gitlinks explicitly refuse, independent of submodule ignore configuration. This is the allowed unsupported-repository path. |
| B15 | FIXED | E01, E05, E13, E14 | Case aliases are enumerated with ignoreCase=false. Worktree domain v7 and review schema v2 retire predecessor incomplete identities; independently reconstructed predecessor review schema refuses. |
| B16 | FIXED | E01, E05 | Subdirectory and foreign core.worktree roots refuse; relative, symlink and linked-worktree controls pass. |
| B17 | FIXED | E01, E05, E08, E11, E14 | SHA-1/SHA-256 replacement bytes are dirty; Capture and Pilot read original objects. Predecessor review-schema controls refuse. |
| B18 | FIXED | E01, E05 | Absolute, relative and symlink top-level roots pass; subdirectory/foreign-root refusals remain. |
| F01 | FIXED | E01, E05, E06, E08 | Index blob/mode/deletion/unmerged entries affect v7 dirty identity; staged command mutation refuses. |
| F02 | FIXED | E08, E12, E13 | All four host writes serialize with core authority through actual durable host append; real BLOCK-review CLI ordering passes. |
| F03 | FIXED | E09, E10, E12 | Existing unsynced ancestors are included in every append; same/different-run initialization and failure controls pass the modeled checks. |
| F04 | FIXED | E01, E05, E13 | Historical gitlink diff is nonempty and stable under local ignore settings; supplied historical controls also pass. |
| F05 | PARTIAL (Gate D) | E01, E13 | Verified dependency graph is not implemented. Human scope note defers it to Gate D; all-unit invalidation remains active. |
| C01 | FIXED | E01, E05, E06, E08 | Changed or new .gitignore causes complete untracked enumeration; raw and full-review negatives pass. New finding R5-01 concerns tracked parent structure under committed ignores. |
| C02 | FIXED | E09, E13 | Relative store ancestry reaches the absolute filesystem root; 94 relative-path crash images pass. |
| C03 | FIXED | E01, E08 | Review import rejects revoked core route authority; valid core API rewind-first case passes. |
| C04 | PARTIAL (Gate D) | E01, E13 | Graph precision remains deferred; conservative invalidation remains active. |
| R3-01 | FIXED | E12, E13 | A committed BLOCK stops valid CLI transition. A later BLOCK waits through gate plus successful core append. |
| R3-02 | FIXED | E09, E10 | Symlink and target ancestry are synchronized; crash model and fault tests pass. |
| R3-03 | PARTIAL (Gate D) | E01, E13 | Transitive graph precision remains deferred; conservative invalidation remains active. |
| R3-04 | FIXED | E13 | Task 14 describes conservative early-gate all-unit route invalidation and the Gate D graph boundary. |
| R4-01 | FIXED | E08, E12, E13 | Own valid transition.Apply terminal/rewind tests cover all host operations, with actual callback commit inside core lock. Host-first and revocation-first behavior pass. |
| R4-02 | FIXED | E09, E10, E13 | Intermediate link parents and absolute/relative targets are included; multihop model survives 102 crash images and injected parent failures refuse. |

## Evidence ledger

Raw logs are complete on disk. Some tool displays were truncated. The report does not use truncated display text as its sole evidence. The replay script retains expected failures as failures. An expected failing negative control is not a passing product test.

| Evidence | Artifact | Observation |
|---|---|---|
| E00 | [FREEZE-start.json](FREEZE-start.json) | Start freeze: main, full HEAD, git status --short, binary diff SHA-256, every untracked file SHA-256. |
| E01 | [test-final.log](test-final.log) | Supplied suite passes all 12 packages with -count=1 -skip ^TestAuditor ./...; copy uses a common TMPDIR and GOTMPDIR. |
| E02 | [vet.log](vet.log) | go vet ./... exits 0. |
| E03 | [race-final.log](race-final.log) | Supplied suite passes with -race; all 12 packages. |
| E04 | [dogfood-final.log](dogfood-final.log) | Real CLI synthetic dogfood passes 17 checks; no human approval or real worker dispatch. |
| E05 | [probe-raw.log](probe-raw.log) | Independent raw matrix passes 16 mutations, SHA-1/SHA-256 replacement controls, historical gitlink, staged-under-dirty and root forms. |
| E06 | [probe-directory-cli.log](probe-directory-cli.log) | 28 full review cases pass their expected acceptance/refusal; 2 parent-symlink stale reviews incorrectly reach Verification; 2 CI parent-symlink cases incorrectly append evidence. |
| E07 | [probe-directory-type.log](probe-directory-type.log) | Independent clean snapshot reproducer: tracked directory replaced by an ignored symlink retains identical Head/Diff/empty Worktree; Git reports deletion. |
| E08 | [probe-ordering-ci.log](probe-ordering-ci.log) | 12 valid core-API/host ordering cases pass. Five independent Python capture mutations refuse. Capture reads original Git object view with replacement refs. |
| E09 | [probe-durability.log](probe-durability.log) | Directory-entry crash model passes 388 images: 94 absolute, 94 relative, 98 symlink, 102 multihop-relative; 12 directory-fsync failures refuse acknowledgement. |
| E10 | [probe-durability-negative.log](probe-durability-negative.log) | Expected failing negative control: omit intermediate hop parent from sync; acknowledged record disappears at modeled checkpoint 52. |
| E11 | [probe-independence-pilot.log](probe-independence-pilot.log) | Six independence conjunction fixtures pass. Pilot original-object capture passes Grounding, Planning and Implementation review. |
| E12 | [probe-ordering-race.log](probe-ordering-race.log) | Race-enabled valid core API ordering, real BLOCK-review CLI ordering, and 24 same/different-run initialization appends pass. |
| E13 | [SOURCE-SHA256.json](SOURCE-SHA256.json) | Primary-source digests. Static wiring inspected for CI projection, route replay, strict decoding, guard identity, lock scope, durable ancestors, review digest and raw snapshot. |
| E14 | [probe-legacy.log](probe-legacy.log) | Reconstructed predecessor review-artifact/v1 clean and dirty JSON identities refuse in both optional-binding forms; fresh current review controls advance. |

The independent probe sources are in `target/cmd/hma/auditor_r5_test.go`, `target/internal/repostate/auditor_r5_test.go`, `target/internal/engine/auditor_r5_test.go`, `target/internal/pilot/auditor_r5_test.go`, and `target/internal/host/auditor_r5_*test.go`. `host-store-instrumented.go` changes only observation/fault seams. `host-store-negative-control.go` intentionally omits one synchronization parent. Each has an explicit Go overlay manifest.

## Findings and remedies

### R5-01 — A symlink in a tracked path parent preserves clean evidence and old review authority

**HIGH; observed; BLOCK.

worktreeMatches Lstat/ReadFile the full leaf path. These operations follow symlinks in earlier components. A committed ignore rule can suppress enumeration of the replacement parent symlink. With equal target bytes and leaf mode, changedPaths is empty (or unchanged when another dirty file exists). CI appends revision-labelled evidence before or after the substitution. The full CLI advances Implementation review to Verification using an earlier review, with optional approval worktree binding absent and present.

Evidence:
- `internal/repostate/raw.go:69,143-168`: The leaf comparison never checks that each tracked path parent is a real directory.
- `probe-directory-type.log`: Git reports D dir/tracked; pre/post HMA State values are equal and Worktree stays empty.
- `probe-directory-cli.log:122-129,250-257`: Both old-review paths append sequence 3 Verification records with machine_advanced=false; the human approval is valid for the current computed binding but the review artifact is stale in reality.
- `probe-directory-cli.log:290-301`: Pre-existing and command-induced substitution both append clean CI evidence with an empty WorktreeDigest.

**Proposed fix.** Before reading a tracked leaf, Lstat every relative parent component from the canonical repository root. Refuse a symlink or non-directory parent. Apply the same rule when hashing changed paths, including staged-only paths. Retire existing incomplete clean review identities and dirty worktree identities with new observation/review schema domains; require a fresh independent review.

**Reason.** Git trees require directories for these parent components. Checking only the final leaf silently substitutes a different structure. Schema expiry prevents an old observation of the unsupported structure from authorizing the restored structure.

**Verification.**
- Run TestAuditorR5TrackedDirectoryType, both TestAuditorR5CIDirectoryType cases, and FullArtifactMatrix parent-symlink with optional binding false/true. Expected: Snapshot refuses explicitly or identifies the changed structure. CI and old-review progression append no records and preserve store bytes.
- Review a symlink-parent observation using current v2/v7 identities, restore a real directory with equal bytes, then replay that stored review after the fix. Expected: Old incomplete identities refuse. A fresh review of a supported real-directory checkout succeeds.
- Retain clean/dirty, root-symlink, relative-root, linked-worktree, tracked leaf-symlink, committed-ignore and ordinary staged controls. Expected: Supported unchanged cases succeed. Root aliases remain supported; a root alias is distinct from a symlink replacing a tracked internal directory.

**Trade-offs.** Failing closed can reject unusual worktrees that replace tracked directories with links. Normal top-level aliases and tracked leaf symlinks must retain their current behavior. Schema expiry requires fresh captures/reviews.

**Alternative.** Represent every relevant parent component and link target in the artifact identity, including ignore-resistant enumeration. This is larger and risks admitting content outside the Git tree; explicit refusal is smaller.

**Effort and status.** medium; `proposed`. Priority 1.

### R5-02 — M3 final and disputed validation closure subcheck is not demonstrated

**MEDIUM; observed; BLOCK for full M3 closure; not an observed final-validation bypass.

The allowed Gate A M3 closure criterion explicitly requests final and disputed coverage. Supplied tests and independent probes demonstrate high/critical risk-rule and capability conjunction behavior. No final/disputed validation purpose is represented in RouteCoherenceRequest or tested as a host-derived context. This audit cannot treat those two purpose-specific checks as passed. This is an evidence/scope gap, not proof that final validation is currently accepted unsafely.

Evidence:
- `docs/handoffs/evidence/gate-a/audit-r1.json M3.closure_criterion (only this field read)`: The criterion ends: Cover final and disputed validation.
- `internal/engine/route_coherence.go:33-46,301-305`: The request has Risk, but no final/disputed purpose. The kernel separation floor applies to high/critical risk.
- `probe-independence-pilot.log`: All six independently constructed high/critical conjunction fixtures pass; they do not establish purpose-to-risk translation.

**Proposed fix.** Provide the supported trusted-host entry points and deterministic mapping for final/disputed validation, then add independent fixtures that exercise those contexts with weak rule IDs and same-family routes. If those entry points remain deferred, obtain an explicit human scope decision for this M3 subcheck and keep its absence visible.

**Reason.** The audit must distinguish the repaired rule-selection defect from the remaining literal closure requirement. A label on a high-risk unit test does not test purpose-derived risk.

**Verification.**
- Exercise final and disputed contexts through their supported host/API boundary with every applicable lower-risk rule, plus a valid separated control. Expected: Same-family routes refuse regardless of selected rule. Valid independent routes remain eligible. If the surface is unsupported, it refuses explicitly and the scope decision records that limitation.

**Trade-offs.** The minimum may be an evidence fixture and an explicit scope decision. Adding a new product context requires approved contract/binding design; do not widen permissions to create the test.

**Alternative.** Explicitly defer only the purpose-specific subcheck by human decision while retaining all repaired high/critical kernel constraints. No such decision was present in the allowed inputs.

**Effort and status.** unknown; `remediation_blocked`. Priority 2.

### Minimal reproduction for R5-01

1. Commit `dir/tracked` with bytes `tracked`. Commit an ignore rule `dir`. Force-add the tracked file.
2. Capture a clean snapshot, or capture dirty artifact A with a separate unchanged dirty file. Record an independent review of A.
3. Replace `dir` with a symlink to an external directory that has an equal `tracked` file and leaf mode.
4. Observe `git status --porcelain`: `D dir/tracked`. HMA returns the same State.
5. Run CI capture. It appends clean evidence. Or present a fresh valid human transition approval for the current computed binding. The old A review permits Verification in both optional-binding forms.

This is a tracked path failure. The rule that permits committed ignores for untracked content does not authorize a change to a tracked file's parent structure. No store tampering or same-UID attack is needed.

## Remediation roadmap

| Priority | Owner | Minimal action | Independent exit criterion |
|---|---|---|---|
| 1 | Approved implementer | Refuse symlink/non-directory tracked parents; expire incomplete observation identities. | All R5-01 negatives refuse before append; unchanged controls pass; old observations cannot authorize restored structures. |
| 2 | Implementer and human coordinator | Supply M3 final/disputed entry-point evidence or obtain an explicit scope decision. | Literal closure subchecks pass or the approved deferral is recorded. |
| 3 | Independent auditor | Review the fixed packet after verification. | Frozen target, complete probes, no unwaivable blocker. |

No unrelated redesign is proposed. There are no additional discretionary implementation opportunities in this packet.

Replay completed after the frozen blind pass. All nine command groups returned their expected exit codes. See `RERUN-RESULT.log` and `rerun-logs/`. Final source hashes still match the start freeze. The JSON report passes the skill report-contract validator. That validation checks form, not finding truth.

## Controlled unblinding

Phase 1 remained unchanged after its recorded hash. Only then did the auditor read the target TODO Gate B section and TODO diff. Those are implementer claims. They did not overturn the reproductions. The TODO marks N1/N2 and the relevant artifact fixes complete. This audit cannot confirm full closure. A historical F02 bullet states the old reversed lock order; the later R3-01 bullet and actual code use host then core. No prior closure report or prohibited temporary review directory was opened.

## Limits and uncertainty

- Context isolation is fresh and blinded to prior closure reviews and implementer discussions. Required original defect-source documents contain historical assessments; they were used only as authoritative requirements. Source comments and supplied tests were inspected as artifacts.
- Read-only source access was procedural, not an OS-enforced read-only mount. All test, probe and fault-injection changes are under this audit output directory. No files under /home/yeli/repos were edited; final freeze verification is reported separately.
- The test target is a copied dirty worktree on main at HEAD 67965541b7032a585a68e48f3d10f4a998d5399e. .serena and the two architecture-amendment drafts were excluded from inspection and the copy. Their untracked hashes are retained only as requested freeze metadata.
- The initial copy-under-/tmp suite exposed a harness test assumption. A second attempt used mismatched TMPDIR/GOTMPDIR. Both failed logs are retained. Final supplied-suite/race/harness runs align both variables and pass. These setup failures are not counted as product safety defects.
- The storage model discards all unsynchronized directory entries at each checkpoint. It covers first/successor append and recovery of each crash image, not real hardware power loss or every possible subset/order of persisted entries. It assumes successfully file-synced immutable data persists. Source ordering and an intentionally broken negative control support the model.
- Earlier named 40/55-checkpoint and r6/r7 reproduction scripts were not loaded before Phase 1. Independent equivalents were constructed. Exact historical binary replay was not performed; predecessor schema/digest expiry also has static domain-separation evidence. Case-insensitive-host execution was unavailable on this Linux case-sensitive filesystem.
- Concurrency probes cover finite schedules. All five supported terminal outcomes have serial coverage in supplied tests; independent valid core API ordering uses ABORTED and Planning rewind. They do not exhaust every outcome, every unit rewind, every operation, and every timing combination.
- Plan-derived dependency precision F05/C04/R3-03 is deferred to Gate D by the supplied human scope note. These rows remain PARTIAL and are not Gate B blockers. Conservative all-units invalidation is mandatory and was checked.
- Jev is unavailable in the host tool inventory. Auditor judgment was used. No nested agents, model APIs, worker CLIs or external communications were used.
- Passing synthetic telemetry, unit tests and this audit cannot grant stage approval, prove full host MVP conformance, or authorize release. The same-UID adversary is outside the supplied trusted-host threat model.

The initial legacy probe used a generic predecessor projection. The later `probe-legacy.log` replay uses reconstructed `hma.host.review-artifact/v1` schema JSON. Exact historical binary fixtures remain outside the evidence claim. The initial durability-model relative-path failure came from the probe failing to canonicalize a relative sync path. The corrected model passes. `probe-durability-model-draft.log` retains that draft failure.

## Required next action

**Return Gate B to the approved implementer for bounded remediation, then obtain a fresh independent closure audit.** Owner: Opus, coordinated by the human. Use the already eligible approved route. Record its provider, model, effort, service, permission envelope, and exact-route `hi` before dispatch. Limit proposed writes to artifact observation, digest migration, related tests, and the M3 evidence gap. This report issues no approval challenge. It grants no commit, push, release, stage transition, or broader permission. Stop if remediation needs changed authority semantics, wider scope, or removal of conservative invalidation. A final/disputed scope change requires an explicit human decision.
