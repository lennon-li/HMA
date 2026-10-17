# Gate B closure audit — 2026-10-16 r4

**Verdict: FAIL.** Gate B is not ready for closure. Two high-severity findings remain. No machine stage transition or human approval is issued.

Target: `main` at `67965541b7032a585a68e48f3d10f4a998d5399e` plus the frozen working tree. Binary diff SHA-256: `c4c0130fb71f906feaee12face6dbb07602cf8bc5fb590ac529702df73346fc7`. The start and end inventories match. All 24 untracked files were hashed. The three excluded paths were not evaluated.

**Isolation: compromised.** The first diff read accidentally included the forbidden Gate B section of `TODO.md` before phase-1 findings were written. The leak was disclosed. It is recorded in [isolation.json](isolation.json). This session had no implementer chat, but this pass cannot be called blind or fully independent. No earlier review directory or optional historical explanation was opened.

Phase-1 file: [PHASE1.json](PHASE1.json). SHA-256: `4273b262685f45a1a6dc90f93b9f5fc928b45e4b52b1bbb97fb4102a4618307b`. The file was written once. No optional unblinding followed. Requirements were derived and hashed before the diff in [requirements-before-diff.json](requirements-before-diff.json).

**Observed verification.** The final baseline suite passed 12 tested packages. It recorded 717 passing test/subtest results and two intentional skips. Vet, build, race, shell syntax, whitespace checks, and the 17-check dogfood harness passed. The two initial runs with default `TMPDIR` failed one location-dependent scratch test. Their logs are retained. Controlled `TMPDIR` runs passed. The initial commentary count of 13 tested packages was incorrect; the raw log count is 12.

| Check | Result | Evidence |
| --- | --- | --- |
| Standard suite | PASS; 717 pass events, 2 skips | [logs/go-test-controlled.jsonl](logs/go-test-controlled.jsonl) |
| Vet | PASS | [go-vet.log](logs/go-vet.log) |
| Race suite | PASS; no race report | [go-race-controlled.jsonl](logs/go-race-controlled.jsonl) |
| Build / shell syntax / diff whitespace | PASS | [build](logs/go-build.log), [diff check](logs/diff-check.log) |
| Dogfood | PASS; 17 checks | [dogfood.log](logs/dogfood.log) |
| Full artifact matrix | PASS; 26 subcases | [logs/artifact-matrix-final.log](logs/artifact-matrix-final.log) |
| Independent raw-snapshot controls | PASS | [logs/snapshot-probes.log](logs/snapshot-probes.log) |
| Independent pilot/capture controls | PASS | [logs/pilot-probes.log](logs/pilot-probes.log) |
| Process-exit recovery | PASS; 215 interrupted checkpoints | [logs/crash-matrix.log](logs/crash-matrix.log) |
| Approved API/core versus host commit | FAIL; 2 approved API cases | [logs/approved-api-probe.log](logs/approved-api-probe.log) |
| Intermediate symlink durability model | FAIL | [probe log](logs/independent-probes.log) |
| Selected independent probes under race detector | Same two failures; no data race report | [independent-race.log](logs/independent-race.log) |

**Human scope decision.** The approved criteria note defers F05, C04 and R3-03 to Gate D. Their closure status remains PARTIAL. Conservative all-unit invalidation is required now and is present. The deferral does not authorize a safety waiver.

**Requirement coverage:** 36 FIXED; 9 PARTIAL; 0 OPEN. Three PARTIAL entries are explicitly deferred. Six PARTIAL entries are affected by the two blocking findings. FIXED means the named defect check is satisfied within the stated verification limits. It does not claim full host-MVP or release conformance.

| ID | Decision | Observation | Evidence |
| --- | --- | --- | --- |
| H1 | FIXED | Projected current stage and only a packet from the current visit determine CI eligibility. Source-stage and previous-visit packets refuse. | `logs/go-test-controlled.jsonl#TestGateBR2CIProjectedTransitions`; `cmd/hma/ci.go:85` |
| H2 | FIXED | Grounding and Planning captures accept approvals without head/diff fields. Later stages retain required bindings. | `logs/go-test-controlled.jsonl#TestGateBEarlyStagePilot`; `logs/go-test-controlled.jsonl#TestGateBR2PlanningPilot`; `internal/pilot/run.go:94` |
| H3 | FIXED | Pending-intent recovery, head-drift negatives and separate-process concurrent writers pass. New crash matrix recovers verified prefixes at 215 interrupted checkpoints. Ancestor durability remains separately partial under B7/F03/R3-02. | `logs/go-test-controlled.jsonl#TestGateBHostConcurrentAppends`; `logs/go-test-controlled.jsonl#TestGateBR2HostSeparateProcessWriters`; `logs/crash-matrix.log` |
| H4 | FIXED | All five terminal outcomes refuse each serial host command and coding gates. Concurrent core API commit ordering remains separately partial under B5/F02. | `logs/go-test-controlled.jsonl#TestGateBR2AllTerminalHostPaths`; `logs/go-test-controlled.jsonl#TestGateBTerminalRunClosesTransitionGate` |
| M1 | FIXED | Evidence capture refuses in-repository and symlinked in-repository stores. Verification-only behavior remains read-only in intent. | `logs/go-test-controlled.jsonl#TestGateBCIInsideRepositoryStore`; `logs/go-test-controlled.jsonl#TestGateBR2CIStoreSymlink`; `cmd/hma/ci.go:65` |
| M2 | FIXED | Trailing JSON and trailing garbage refuse. Whitespace-only suffix is accepted. Refusals retain stored bytes. | `logs/go-test-controlled.jsonl#TestGateBR2ResolveStrict`; `cmd/hma/host.go:56` |
| M3 | FIXED | Every applicable rule contributes constraints. HIGH and CRITICAL enforce provider and model-family separation. Lower-tier and same-tier weak selection cannot drop constraints. | `logs/go-test-controlled.jsonl#TestGateBCriticalWeakRule`; `logs/go-test-controlled.jsonl#TestGateBR2IndependenceTiedRule`; `logs/go-test-controlled.jsonl#TestGateBR2IndependenceRiskControls`; `internal/engine/route_coherence.go:274` |
| M4 | FIXED | The second coding gate replays current core route approval and compares the active implementation route digest. | `logs/go-test-controlled.jsonl#TestHostRouteApprovalInvalidatedByEarlierStageRewind`; `cmd/hma/transition.go:73`; `logs/artifact-matrix-final.log` |
| M5 | FIXED | Canonical scratch/run marker and exact seed-record comparison reject copied/symlinked/foreign/nonseed cases. A valid untouched seed succeeds. | `logs/go-test-controlled.jsonl#TestSyntheticStoreAcceptsSeededRun`; `logs/go-test-controlled.jsonl#TestSyntheticStoreRefusesCopiedMarker`; `logs/go-test-controlled.jsonl#TestSeedRouteApprovalRefusesAlteredSeed`; `scripts/dogfood/hostinput/main.go:185` |
| M6 | FIXED | Refusals compare file-name and byte fingerprints for core and host files, excluding lock files, and pass in the executed harness. | `scripts/dogfood/host-dogfood.sh:64`; `logs/dogfood.log` |
| L1 | FIXED | Direct valid/invalid structural and through-record fixtures execute successfully. | `logs/go-test-controlled.jsonl#TestRouteAttestationStructure`; `logs/go-test-controlled.jsonl#TestGateBR2AttestationThroughRecord`; `internal/model/route_test.go:26` |
| L3 | FIXED | Workflow runs the harness. The harness succeeds locally with 17 checks. Hosted GitHub execution was not performed. | `.github/workflows/ci.yml:51`; `logs/dogfood.log` |
| N1 | FIXED | Full CLI artifact matrix rejects each changed artifact in both optional-binding forms and accepts unchanged dirty controls. | `logs/artifact-matrix-final.log`; `cmd/hma/transition.go:103` |
| N2 | FIXED | CI tests refuse empty/foreign/wrong/dirty/mutated checkouts. Independent pilot mutation probes append no evidence. Clean and exact dirty controls pass. | `logs/go-test-controlled.jsonl#TestGateBCINonRepositoryEvidence`; `logs/go-test-controlled.jsonl#TestGateBR2CISymlinkAndWorktree`; `logs/pilot-probes.log`; `logs/snapshot-probes.log` |
| B1 | FIXED | Both optional-binding forms bind the independent review to the live dirty digest. Changed bytes refuse with unchanged store bytes. | `logs/artifact-matrix-final.log#bytes`; `cmd/hma/transition.go:97` |
| B2 | FIXED | Assume-unchanged and skip-worktree are explicit unsupported-state refusals. Clean controls remain usable. | `logs/go-test-controlled.jsonl#TestGateBR2CIHiddenDirty`; `logs/go-test-controlled.jsonl#TestGateBR2CIMutationHiddenByIndex`; `internal/repostate/snapshot.go:417` |
| B3 | FIXED | CI refuses inherited selectors. Capture and pilot remove selectors and force original Git objects. Approved config injection is retained. | `logs/go-test-controlled.jsonl#TestGateBR2CIGitEnvironment`; `logs/go-test-controlled.jsonl#TestGateBR2EvidenceEnvControl`; `logs/pilot-probes.log#TestAuditorReplacementCapturePilot`; `internal/repostate/snapshot.go:61` |
| B4 | FIXED | CI explicitly rejects populated dirty-worktree approval bindings. Absent-binding clean checkout controls pass. | `logs/go-test-controlled.jsonl#TestGateBR2CIBoundWorktreeGone`; `cmd/hma/ci.go:187` |
| B5 | PARTIAL | Serial and CLI transition serialization pass. An approved concurrent transition.Apply commits ABORTED before the host commit; host dispatch still records and returns START_WORKER. See R4-01. | `logs/approved-api-probe.log`; `logs/independent-probes.log#TestAuditorCoreHostCommitWindow` |
| B6 | FIXED | Decision evaluation and append share the host run lock. Concurrent conflicting initial decisions cannot both commit without supersession. | `logs/go-test-controlled.jsonl#TestGateBR2ConcurrentHostDecisions`; `internal/host/store.go:256` |
| B7 | PARTIAL | Ordered record/head/intent fsync and process-exit recovery pass. Multi-hop symlink resolution omits an intermediate symlink parent from fsync. See R4-02. | `logs/crash-matrix.log`; `logs/independent-probes.log#TestAuditorIntermediateSymlinkDurability`; `internal/host/store.go:343` |
| B8 | FIXED | Effective constraints union capability and separation requirements across every applicable rule rather than trusting the selected rule alone. | `logs/go-test-controlled.jsonl#TestGateBR2IndependenceTiedRule`; `internal/engine/route_coherence.go:293` |
| B9 | FIXED | Seeding compares complete marshaled seed record except computed head. Altered disposition, foreign seed, directory/symlink marker and repeated seeding refuse. | `logs/go-test-controlled.jsonl#TestGateBR2SyntheticForeignSeed`; `logs/go-test-controlled.jsonl#TestGateBR2SyntheticSecondSeed`; `logs/go-test-controlled.jsonl#TestGateBR2SyntheticMarkerDirectory`; `scripts/dogfood/hostinput/main.go:199` |
| B10 | FIXED | The approved human scope decision permits conservative invalidation. Any unit return to Planning or earlier invalidates all old routes. A fresh Route Selection approval can establish new authority. Plan-derived precision is deferred. | `logs/go-test-controlled.jsonl#TestGateBR2DependentRouteInvalidation`; `cmd/hma/transition.go:205`; `requirements-before-diff.json#criteria.note` |
| B11 | FIXED | Raw file type and owner-execute changes alter identity and refuse prior reviews. Group-execute equivalence and clean controls pass. | `logs/artifact-matrix-final.log#owner-mode-hidden`; `logs/artifact-matrix-final.log#symlink`; `logs/snapshot-probes.log#TestAuditorSnapshotControls`; `internal/repostate/snapshot.go:202` |
| B12 | FIXED | Raw comparison detects owner-execute regardless of Git filemode settings. CI and both full review paths refuse changed modes. | `logs/go-test-controlled.jsonl#TestGateBR4CIHiddenOwnerExecute`; `logs/go-test-controlled.jsonl#TestGateBR4CIHiddenOwnerExecuteMutation`; `logs/artifact-matrix-final.log#owner-mode-hidden` |
| B13 | FIXED | Raw hashing detects normalized byte differences for tracked, nested, info and global attributes. Clean LF controls pass. No conversion-based clean claim survives. | `logs/snapshot-probes.log#TestAuditorNormalizedAttributes`; `logs/go-test-controlled.jsonl#TestGateBR5CINormalizedTrackedBytes`; `internal/repostate/raw.go:177` |
| B14 | FIXED | HEAD or index gitlinks refuse before capture, including missing worktree cases. Sparse checkout also refuses. Ignoring submodules cannot hide the raw gitlink check. | `logs/snapshot-probes.log#TestAuditorRootFormsAndSubmoduleRefusals`; `logs/go-test-controlled.jsonl#TestSnapshotRefusesSubmoduleAndSparse`; `internal/repostate/raw.go:66` |
| B15 | FIXED | Local/include/environment ignorecase controls cannot hide a distinct case alias. Current v7 dirty digest and review-v2 domains reject predecessor identities. Historical r6 binary fixtures were not loaded. | `logs/snapshot-probes.log#case-local`; `logs/snapshot-probes.log#case-include`; `logs/snapshot-probes.log#case-env`; `logs/artifact-matrix-final.log#legacy-v6`; `internal/repostate/snapshot.go:182` |
| B16 | FIXED | Subdirectory and foreign core.worktree roots refuse before capture. Absolute, relative, symlink and linked-worktree controls remain supported. | `logs/snapshot-probes.log#TestAuditorRootFormsAndSubmoduleRefusals`; `logs/go-test-controlled.jsonl#TestGateBR2CISymlinkAndWorktree`; `internal/repostate/raw.go:219` |
| B17 | FIXED | SHA-1 and SHA-256 snapshots ignore replace refs. Capture and pilot read original objects even with inherited selectors. Current review-v2 schema rejects a synthesized predecessor clean review. | `logs/snapshot-probes.log#TestAuditorReplacementObjectFormats`; `logs/pilot-probes.log#TestAuditorReplacementCapturePilot`; `logs/artifact-matrix-final.log#legacy-clean-schema`; `internal/host/host.go:643` |
| B18 | FIXED | Absolute, relative and symlink root forms pass. Wrong-root controls refuse. Case-insensitive-host aliases were unavailable on this Linux host. | `logs/snapshot-probes.log#TestAuditorRootFormsAndSubmoduleRefusals`; `logs/go-test-controlled.jsonl#TestGateBR2CISymlinkAndWorktree` |
| F01 | FIXED | Full review matrix refuses staged bytes/modes/deletion in both binding forms. Capture-induced staging refuses with no append. Unapproved unmerged stages refuse. A reconstructed v6 dirty review expires. | `logs/artifact-matrix-final.log#index-bytes`; `logs/artifact-matrix-final.log#index-mode`; `logs/artifact-matrix-final.log#index-deletion`; `logs/artifact-matrix-final.log#legacy-v6`; `logs/pilot-probes.log`; `internal/repostate/snapshot.go:186` |
| F02 | PARTIAL | The core lock protects record construction, then releases before host persistence. Approved API abort/rewind commits in that gap; stale dispatch still succeeds. See R4-01. | `logs/approved-api-probe.log`; `cmd/hma/host.go:118`; `internal/host/store.go:265` |
| F03 | PARTIAL | Absolute and relative ancestor-fsync error controls refuse without acknowledgement. Intermediate symlink ancestors are still omitted. See R4-02. | `logs/independent-probes.log#TestAuditorAncestorSyncFailures`; `logs/independent-probes.log#TestAuditorIntermediateSymlinkDurability` |
| F04 | FIXED | Nonempty historical gitlink diff remains identical across local ignore policies. Canonical diff forces --ignore-submodules=none. | `logs/snapshot-probes.log#TestAuditorHistoricalGitlinkPolicy`; `logs/go-test-controlled.jsonl#TestSnapshotDiffIncludesHistoricalGitlink`; `internal/repostate/snapshot.go:254` |
| F05 | PARTIAL (Gate D) | Human explicitly deferred graph precision to Gate D on 2026-10-16. The required conservative all-unit invalidation is present. This deferral is not a waiver of safety. | `logs/go-test-controlled.jsonl#TestGateBR2DependentRouteInvalidation`; `cmd/hma/transition.go:205`; `requirements-before-diff.json#criteria.note` |
| C01 | FIXED | Changed, added and deleted .gitignore disable untracked exclusion. Independent hidden-file A/B changes alter the digest. Full review rejects the changed artifact. | `logs/snapshot-probes.log#ignore-changed`; `logs/snapshot-probes.log#ignore-added`; `logs/snapshot-probes.log#ignore-deleted`; `logs/artifact-matrix-final.log#dirty-ignore`; `internal/repostate/raw.go:123` |
| C02 | FIXED | Ancestor synchronization climbs absolute paths. Relative-store positive and injected-fsync-failure controls execute successfully. | `logs/go-test-controlled.jsonl#TestGateBSyncAncestorsRelative`; `logs/independent-probes.log#TestAuditorAncestorSyncFailures`; `internal/host/store.go:346` |
| C03 | FIXED | Review import checks current core route authority before evaluating or appending the review. A serial revoked-route import refuses. | `logs/go-test-controlled.jsonl#TestGateBClosureReviewImportRefusesRevokedRoute`; `cmd/hma/host.go:267` |
| C04 | PARTIAL (Gate D) | Graph precision is explicitly deferred to Gate D. Missing graph cannot preserve an independent unit because current policy invalidates all old units conservatively. | `requirements-before-diff.json#criteria.note`; `cmd/hma/transition.go:205` |
| R3-01 | FIXED | Transition CLI holds the host lock from gate evaluation through core append. A BLOCK already committed refuses both full CLI binding forms. Host append cannot interleave inside that CLI section. | `logs/go-test-controlled.jsonl#TestGateBClosureTransitionHoldsHostLock`; `logs/artifact-matrix-final.log#block-current`; `cmd/hma/transition.go:277` |
| R3-02 | PARTIAL | Lexical and final resolved paths are synced, but an intermediate link parent is skipped. The modeled lost link makes Load return an empty run after acknowledgement. See R4-02. | `logs/independent-probes.log#TestAuditorIntermediateSymlinkDurability`; `internal/host/store.go:350` |
| R3-03 | PARTIAL (Gate D) | The same human Gate D deferral applies. Conservative invalidation remains mandatory and is implemented. | `requirements-before-diff.json#criteria.note`; `cmd/hma/transition.go:205` |
| R3-04 | PARTIAL | Conservative invalidation documentation is corrected. The statement that host commands check and append under both locks remains false. See R4-01. | `docs/task14-host-orchestration.md:157`; `docs/task14-host-orchestration.md:192`; `logs/approved-api-probe.log` |

**Findings, ordered by impact.**

**R4-01 — HIGH — The core lock ends before the host authorization commits.**

appendHostRecord acquires the core run lock inside the build callback passed to host.Store.AppendWith. WithRunLock returns at cmd/hma/host.go:148, releasing that lock. host.Store.appendLocked then validates, hashes and persists the returned record at internal/host/store.go:269-309. A scheduling-only overlay pauses immediately after build returns. A valid challenge-bound transition.Apply abort or Planning rewind commits first. runHostDispatch then commits against the earlier core head and returns dispatch_allowed=true with next_action=START_WORKER. No stored bytes are rewritten by the test. The ordinary hma transition CLI now takes the host lock and serializes with host writes; the demonstrated conflict uses the callable core transition API. The existing route-revocation regression starts a contender but does not prove the host record commits before it.

Evidence:
- `cmd/hma/host.go:116`: The core-lock callback constructs a record; actual host persistence occurs after its return.
- `internal/store/jsonl.go:166`: The core lock is deferred-unlocked when WithRunLock returns.
- `internal/host/store.go:265`: build returns before record validation and the durable append.
- `logs/approved-api-probe.log`: Both ABORTED and PLANNING approved transition.Apply controls commit before host append; dispatch still returns START_WORKER.
- `logs/independent-probes.log#TestAuditorCoreHostCommitWindow`: 15 post-build terminal interleavings succeed incorrectly: 3 host operations times 5 terminal outcomes.
- `docs/task14-host-orchestration.md:192`: The documented promise that host commands append under both locks is not implemented.

Preferred fix (proposed): Restructure the host transaction API so the host lock is taken first and the core run lock remains held through the actual host record persistence and final directory sync. Provide a continuation or explicit locked-append operation rather than returning a record from a callback whose core guard has ended. Update the task14 lock guarantee to describe the tested implementation.

Why: The current nested callback only guards record construction. Extending the guard through commit makes terminal/revocation order and bound core head agree.

Independent verification:
- Run TestAuditorApprovedTransitionAPICommitWindow and decision/preflight/dispatch/review races with valid core API terminal/rewind inputs in both commit orders. Expected: If core authority is withdrawn first, host adds no record and emits no execution action. If host commits first, the core contender waits until host durability is complete.
- Repeat the BLOCK-review CLI race, independent concurrent decisions, all five terminal outcomes and normal unchanged full CLI controls. Expected: No deadlock, stale authorization, lost accepted record, or widened permission. The guarded interval includes the actual append, not only build.

Trade-offs: Adds an explicit transaction/lock-lifetime interface. All callers must preserve the single host-then-core lock order. Holding the core lock through fsync increases contention. A post-commit check alone cannot undo an already committed authorizing record.

Alternative: Route all authority-changing API writers through a common host-then-core transaction boundary and make bypassing entrypoints unavailable. This is broader and requires compatibility analysis; merely documenting callers as trusted does not prevent accidental concurrency.

Effort: medium. Priority: 1. Fix status: proposed; no remediation was implemented or validated.

**R4-02 — HIGH — An intermediate symlink parent is absent from durable store ancestry.**

syncAncestors syncs the lexical absolute parent chain and the final EvalSymlinks parent chain. These chains omit directories containing intermediate symlinks. The independent fixture uses store-alias -> unsynced-bridge/second-hop -> durable-real. Append acknowledges a record, but no syncDir call targets unsynced-bridge. Under a directory-entry crash model, the unsynchronized second-hop entry can be lost. Removing only that entry to materialize the permitted crash image makes Load through the configured store path return zero records with no error. The canonical target still contains the acknowledged record. This is a storage-crash availability/durability failure in the trusted-host model. It is not a same-UID adversary claim. Real power loss was not performed.

Evidence:
- `internal/host/store.go:343`: Only lexical and final resolved path chains are synchronized.
- `logs/independent-probes.log#TestAuditorIntermediateSymlinkDurability`: Intermediate symlink parent sync count is 0 after acknowledged append. Materializing loss of the unsynced hop yields configured-store records=0 with err=nil; canonical target records=1.
- `logs/independent-race.log#TestAuditorIntermediateSymlinkDurability`: The same omission and modeled record loss reproduce under -race.
- `logs/crash-matrix.log`: 215 ordinary process-exit checkpoints recover correctly. Those controls do not establish storage-crash durability for omitted path dependencies.

Preferred fix (proposed): Resolve the store path component by component. Collect and fsync the parent directory of every symlink encountered, including intermediate targets, as well as the final canonical store ancestry before acknowledging. Return any resolution or fsync failure. Add a bounded resolution loop and keep relative and ordinary symlink stores usable.

Why: Every directory entry needed to reach the acknowledged store must be durable, including intermediate symlink entries. Syncing only the starting spelling and final destination does not cover them.

Independent verification:
- Run the intermediate-symlink fixture with absolute and relative multi-hop targets, then inject failure at each collected link parent. Expected: Every necessary parent is synced before acknowledgement. Each injected failure returns an error without acknowledgement.
- Use a directory-entry crash model that loses any unsynchronized link or ancestor; retain first/successor/interrupted recovery and concurrent same/different-run controls. Expected: Every acknowledged record remains reachable through the configured durable store identity. Unacknowledged attempts recover one verified prefix.

Trade-offs: More path-resolution and fsync work is required. The trust model need not defend hostile link replacement during the operation, but ordinary path-resolution errors must fail closed. The current probe demonstrates a permissible crash image, not the behavior of a specific physical drive.

Alternative: Explicitly refuse symlink ancestry before any store append, with a supported canonical absolute store location. This is simpler containment but removes currently accepted path forms and needs a documented compatibility decision.

Effort: medium. Priority: 2. Fix status: proposed; no remediation was implemented or validated.

**Remediation roadmap.** First, extend the transaction guard through the durable host append. Second, close the intermediate-symlink durability gap. Keep the ordinary CLI BLOCK-review lock protection, artifact controls and conservative invalidation intact. Then freeze the repair and obtain a new blind audit. Each finding includes a falsifiable exit check. No discretionary redesign is needed.

**Verification limits and provenance.**

- Isolation is compromised. A target.diff read accidentally displayed TODO.md Gate B implementer claims before PHASE1.json existed. The leak was disclosed immediately and recorded in isolation.json. A fresh blind auditor is still required for an independent closure claim.
- Fresh session provenance is limited to the supplied dispatch and current conversation. Requested route is liz / gpt-6.1-sol / high. No runtime model/effort/provider attestation or exact-route hi preflight evidence was supplied to this auditor. No worker was dispatched.
- No prior /tmp/hma-gate-* reviews, /tmp/claude-* directories, Gate A v*.md handoffs, or excluded architecture drafts were opened. No optional historical unblinding was performed. The permitted criteria file and this output directory were used.
- Tests ran in a separate filesystem copy with copied .git metadata, not in the read-only target. Scheduling/fault hooks and new test files were supplied with Go overlays. Fixtures partly reuse repository setup helpers; the added mutations and assertions were authored during this audit.
- Storage-crash evidence is a directory-entry fault model plus fsync call trace. No actual power loss or hardware durability certification was performed. Process exits were tested separately at 215 checkpoints.
- The atomicity finding is reproduced through a correctly approved core transition.Apply API call. The ordinary transition CLI now holds the host lock and prevents that same CLI-to-CLI interleaving. This distinction limits the demonstrated attack/accident surface.
- The initial default-TMPDIR suite and race run each failed TestSyntheticStoreRefusesNonScratchDir because the test copy is under /tmp. Setting TMPDIR to output/tmp makes the repository copy external to the configured scratch root; both complete suites then pass. Initial failing logs are retained.
- Historical helper filenames and actual r6/r7 binaries named in criteria were not loaded. Equivalent current-path probes, a reconstructed v6 dirty identity, a synthesized predecessor clean schema and source domain checks were used. No exhaustive historical binary compatibility claim is made.
- Linux amd64, Git 2.43.0 and Go 1.27.1 were used. Case-insensitive filesystem alias behavior was unavailable. No hosted GitHub CI job, real model dispatch, external effect or verified-success release path was executed.
- F05, C04 and R3-03 are PARTIAL and deferred to Gate D by the explicit human scope note. The in-scope conservative all-unit invalidation is retained. No approval, waiver, commit, push or release was performed.
- Jev was unavailable in the host tool inventory. Orchestrator judgment was recorded instead. No replacement model was labeled Jev.
- Some initial terminal output was truncated. The complete target diff, raw test logs, source snapshot, probe sources and hashes remain available. Structured reports do not convert missing execution into proof.

**Evidence inventory.** [AUDIT.json](AUDIT.json) contains the report-contract fields and the full requirement matrix. [freeze-start.json](freeze-start.json), [freeze-end.json](freeze-end.json), [source-digests.json](source-digests.json), [target.diff](target.diff), and the `snapshot/`, `logs/`, and `probes/` directories retain target bytes, commands and reproductions. `probes/scheduling-overlay.json` changes only the test copy through Go overlays. `reproduce.sh` gives the rerun commands. The raw logs take precedence over this presentation.

**One recommended next action: RETURN_GATE_B_FOR_BOUNDED_REMEDIATION.**

Owner: the designated Gate B implementer, Opus, coordinated by the human/host. Use the existing approved exact implementation route. Its provider/model/effort/access-service attestation was not available in this audit. The host must report and preflight that route before dispatch. Proposed write scope is `cmd/hma/host.go`, `internal/host/store.go`, directly related tests, and `docs/task14-host-orchestration.md`. This report grants no write authority.

Approval required: the exact challenge-bound HMA approval for the current bounded remediation unit. No approval challenge was supplied or issued here. Human Gate B acceptance and progression to Gate C remain pending. Stop and escalate if the repair requires revised authority semantics, wider permissions, changed criteria, or an incompatible path-support decision. Exit requires both blocking reproductions and the in-scope controls to pass, followed by a fresh blind closure audit.
