# Gate B closure audit

**Verdict: FAIL.** Gate B is not closed.

Target: `main`, `67965541b7032a585a68e48f3d10f4a998d5399e`, plus the frozen working tree.
Diff SHA-256: `1f1eb9ca70958155a25159ccb7cda629b089476ac3643a400535687473006822`.
Phase 1 SHA-256: `c8f0a460c9498fea65cb426992936f6889ed2b138af9a4bd4c0498f940045c32`.

The final freeze matches the start freeze. HEAD, branch, status, binary diff hash and every untracked-file hash are unchanged. The excluded Serena files and two architecture-amendment drafts were hashed but not evaluated. No audited repository or Git state was written.

**Observed coverage:** 29 listed defects are FIXED. Eight are PARTIAL. None is labelled OPEN because each original defect has some remediation. Four actionable residual findings remain open.

**Isolation: partial.** The auditor started with a fresh context. The initial source-bundle read included the whole named Gate A JSON and consolidation handoff. These allowed sources contain non-Gate-B judgments beyond the requested closure fields. The output was truncated. Perfect fact-only blinding is not claimed. No prohibited Gate B review, Gate A v*.md handoff, claude directory or TODO claims was opened before Phase 1 was written and hashed. TODO claims were read only afterward.

## Observed results

| Check | Result | Raw evidence |
|---|---|---|
| Go 1.27.1; go test -count=1 ./... | PASS, 12 packages | [log](logs/go-test.log) |
| go vet ./... | PASS | [log](logs/go-vet.log) |
| go test -count=1 -race ./... | PASS, 12 packages | [log](logs/go-race.log) |
| bash scripts/dogfood/host-dogfood.sh | PASS, 17 checks | [log](logs/dogfood.log) |
| shellcheck scripts/dogfood/host-dogfood.sh | PASS | [log](logs/shellcheck.log) |
| Auditor main CLI probes, including -race | FAIL; hidden-byte progression and revoked-review append | [log](logs/independent-main-race.log) |
| Auditor pilot probes, including -race | FAIL; evidence append after hidden-byte mutation | [log](logs/independent-pilot-race.log) |
| Auditor host fault model, including -race | FAIL; relative-store acknowledgment can disappear | [log](logs/independent-host-race.log) |
| Auditor raw Git snapshot probes | PASS; SHA-1/SHA-256, three alias sources, four attribute sources, three historical gitlink policies | [log](logs/independent-repostate.log) |

The auditor probes fail assertions that express the closure requirements. No data-race warning appears in their race logs. The host model passes 60 absolute-store crash images and seven interrupted-recovery images. It rejects acknowledgment at all 20 reachable directory-fsync failure positions. Twelve concurrent same-run writers and twelve different-run writers retain their records. The relative-store negative still loses its acknowledged record.

## Findings and proposed corrections

### C01 — Uncommitted ignore rules hide changed bytes from review and capture

Severity: **high**. Certainty: **observed**. Disposition: **BLOCK**. Fix: **proposed**.

changedPaths applies worktree .gitignore files to untracked enumeration. It hashes the changed ignore file but omits the files that rule hides. Changing hidden from A to B preserves v7 identity. An existing independent review advances to VERIFICATION in both optional-binding forms. Pilot also appends two records after truncate changes hidden bytes. Committed-ignore exclusions remain a supported control.

Evidence:
- `internal/repostate/raw.go:102`: ls-files --others --exclude-per-directory=.gitignore applies uncommitted ignore content.
- `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix`: Four full transitions accepted changed hidden bytes: new/modified ignore rule, binding absent/present.
- `logs/independent-pilot-race.log#TestClosureR2PilotIgnoredMutation`: Mutating command returned success and appended two core records.

**Preferred correction:** Use ignore rules from the approved committed tree for untracked exclusion. Until that implementation exists, refuse snapshots whose added, modified or staged .gitignore could change the excluded set. Retire incomplete v7 dirty identities before accepting new reviews.

**Scope qualification:** This finding covers files hidden only by new or modified worktree ignore rules. Committed-ignore exclusions remain supported. After Phase 1, TODO lines 112-113 claimed a broader ignored-file exclusion under Decision T. The authoritative human approval was not supplied. If that approval explicitly excludes these newly hidden paths, the human must reassess C01 against that exact scope. The reproduced behavior remains unchanged.

**Reason:** Only committed ignore policy may exclude a file from this raw snapshot boundary. A changed rule must not silently widen the unobserved artifact.

Independent acceptance checks:
- Run new/modified/nested ignore probes and pilot mutation with bindings absent/present. Expected: Changed hidden bytes require a different artifact identity or an explicit refusal; no stale transition/evidence append.
- Replay actual stored v7 reviews that omitted hidden, including after removal of the rule; retain committed-ignore unchanged controls. Expected: Incomplete predecessor reviews refuse; current unchanged supported artifacts pass.

**Trade-offs:** Refusal temporarily narrows dirty-worktree support. Reading committed rules requires correct nested and negation semantics. Domain retirement requires fresh captures and reviews.

**Alternative:** Enumerate and bind all untracked files, including ignored files, under an explicitly approved scope policy. This changes build-output behavior and needs contract review.

**Effort:** medium. Affected closure IDs: N1, B1.

### C02 — Relative host store paths do not establish durable ancestry

Severity: **high**. Certainty: **observed**. Disposition: **BLOCK**. Fix: **proposed**.

syncAncestors climbs the lexical store path. For relative-store/host/run it stops at dot. If the working directory has an existing unsynchronized ancestor entry, Append acknowledges while that entry can still disappear. The modeled crash recovers zero records without an error. Absolute-path controls preserve records.

Evidence:
- `internal/host/store.go:316`: filepath.Dir reaches dot; the function returns without visiting absolute ancestors of cwd.
- `logs/independent-host-race.log#TestClosureR2RelativeStoreDurability`: acknowledged=true persisted-files=0 recovered=0 err=<nil> sync-count=10.
- `logs/independent-host-race.log#TestClosureR2DurableCrashImages`: 60 absolute-store crash images pass; seven recovery crash images pass.

**Preferred correction:** Normalize the host store to an absolute physical path before directory initialization and ancestry synchronization. Synchronize the resolved ancestry to the durable boundary, including any newly created suffix. Treat fsync errors as non-acknowledgment.

**Reason:** Durability must hold for accepted relative paths and path aliases, not only absolute spellings.

Independent acceptance checks:
- Run the relative-store crash probe, absolute control, symlink-to-store variants and same/different-run initialization. Expected: Every acknowledged record remains recoverable in the modeled crash image.
- Inject fsync failure at every visited directory and interrupt recovery. Expected: No failed sync acknowledges; recovery returns one verified prefix.

**Trade-offs:** Path resolution can fail or race with host filesystem changes; those cases must refuse. Absolute conversion alone does not establish symlink-target ancestry.

**Alternative:** Refuse relative and unresolved/symlinked store roots until physical-ancestry synchronization is supported. This is smaller containment but breaks accepted relative-path usage.

**Effort:** small. Affected closure IDs: H3, B7, F03.

### C03 — Review import accepts revoked route authority

Severity: **medium**. Certainty: **observed**. Disposition: **BLOCK**. Fix: **proposed**.

runHostReview locks the core and host chains but only calls EvaluateReview(history, review). It ignores the loaded core route state. After an approved same-unit, upstream-unit or whole-run Planning rewind, approvedCoreRouteDigest refuses, yet host review appends and returns CONTINUE_HMA_TRANSITION. The later coding gate still rejects the revoked route; this probe does not demonstrate stage advancement.

Evidence:
- `cmd/hma/host.go:262`: Review callback discards core and only validates host-history review coherence.
- `logs/independent-main-race.log#TestClosureR2ReviewAfterRewind`: All three rewinds still append the independent_review record.
- `logs/independent-main-race.log#TestClosureR2CoreHostOrdering/rewind/review/first=core`: Core-first revocation accepts a review append; terminal-first cases refuse.

**Preferred correction:** Within the locked review callback, resolve the named review dispatch and revalidate its unit route approval against the loaded core snapshot before appending.

**Reason:** F02 explicitly requires review writes to refuse when revocation commits first; an atomic lock does not replace the missing authority predicate.

Independent acceptance checks:
- Run ReviewAfterRewind and both-order CoreHostOrdering for review, unit/upstream/whole-run revocations and all five terminal outcomes. Expected: Revocation-first returns route-approval refusal and preserves stored bytes. Host-first binds the pre-revocation core head.
- Run unchanged valid review import and the second coding gate. Expected: Current authorized review remains usable; stale reviews cannot regain authority.

**Trade-offs:** Historical results may need a separate nonauthorizing telemetry path if the product must retain them after revocation. Do not return an execution/progression action for such telemetry.

**Alternative:** Persist late reviews as explicitly nonauthorizing observations with no CONTINUE_HMA_TRANSITION action. This requires a separate documented record contract.

**Effort:** small. Affected closure IDs: F02, B10.

### C04 — Dependency invalidation has no verified plan graph

Severity: **medium**. Certainty: **observed**. Disposition: **BLOCK**. Fix: **proposed**.

approvedCoreRouteDigestIn clears every unit route after another unit returns to Planning or earlier. It does not load AcceptedPlanDigest, verify a graph or compute transitive dependents. In the U2->U1->U0 plus independent U3 scenario, all old routes are conservatively refused. A new trusted approval can restore U1 even when its plan digest names no resolved artifact. Safety containment exists, but B10/F05 plan-derived closure is incomplete.

Evidence:
- `cmd/hma/transition.go:192`: Source explicitly defers narrowing conservative invalidation with plan-derived dependents.
- `logs/independent-main-race.log#TestClosureR2PlanReplay`: U0/U1/U2/U3 all refuse after U0 rewind; fresh-route approval with unresolved plan is accepted.
- `internal/engine/invalidation.go:117`: Pure engine consumes caller-supplied DependentUnitIDs; route replay has no verified graph input.

**Preferred correction:** Add a trusted host resolver for the accepted immutable plan digest. Validate unit IDs, dependencies and cycles. Compute the transitive affected set from that verified graph and apply it to route replay. Keep conservative refusal when resolution or validation fails.

**Reason:** B10/F05 require plan-derived transitive invalidation and fail-closed behavior for missing or corrupt plans.

Independent acceptance checks:
- Use a verified U2->U1->U0 graph and independent U3, then rewind U0. Expected: Old U0/U1/U2 authority refuses; U3 may survive only when the verified graph proves independence.
- Remove/corrupt the plan, attempt review re-attestation, perform a whole-run rewind, then establish a fresh approved route with a verified current plan. Expected: Missing/corrupt plans refuse. Review cannot revive invalidated authority. A fresh valid approval establishes new authority.

**Trade-offs:** Graph persistence/resolution may need a scoped contract addition. Conservative refusal remains the safe containment while that work is approved.

**Alternative:** Retain whole-run conservative invalidation and explicitly obtain a human-approved change to the closure criterion. The auditor cannot change that criterion.

**Effort:** medium. Affected closure IDs: B10, F05.

## Closure matrix

FIXED means the scoped mechanism has supporting source and execution evidence. PARTIAL means remediation exists but the authoritative closure requirement remains unsatisfied. The JSON preserves all supplied B/F closure variants. Historical helpers were not silently treated as executed.

| Defect | Status | Observation and evidence |
|---|---|---|
| H1 | FIXED | Projected stage replaces lastRecord.Stage; approvals are selected since current-stage entry. TestCIUsesProjectedStageNotLastRecord and TestGateBR2CIProjectedTransitions pass. Evidence: `cmd/hma/ci.go:88`, `cmd/hma/main_test.go`, `logs/go-test.log` |
| H2 | FIXED | Grounding and Planning pilots accept absent head/diff bindings and retain actual head evidence. Evidence: `internal/pilot/run.go:99`, `logs/independent-pilot-race.log#TestClosureR2PilotEarlyStages` |
| H3 | PARTIAL | Locking, pending recovery and absolute-path durability work. Relative stores can lose an acknowledged record. C02 blocks closure. Evidence: `internal/host/store.go:215`, `logs/independent-host-race.log#TestClosureR2RelativeStoreDurability` |
| H4 | FIXED | All four host operations serialize with all five supported terminal outcomes, with both commit orders. Terminal-first operations refuse without stored-byte changes. Evidence: `cmd/hma/host.go:116`, `logs/independent-main-race.log#TestClosureR2CoreHostOrdering` |
| M1 | FIXED | CI capture checks outside-repository store placement; direct and symlinked inside stores refuse. Evidence: `cmd/hma/ci.go:73`, `cmd/hma/gateb_r2_test.go#TestGateBR2CIStoreSymlink`, `logs/go-test.log` |
| M2 | FIXED | Resolution uses the strict single-document decoder; concatenated JSON and unknown-field regressions pass. Evidence: `cmd/hma/resolve.go:47`, `cmd/hma/gateb_r2_test.go#TestGateBR2ResolveStrict`, `logs/go-test.log` |
| M3 | FIXED | Risk-tier selection, union of all applicable constraints, and HIGH/CRITICAL kernel family separation are enforced. Host disputed/final dispatch controls independently reject same families. Evidence: `internal/engine/route_coherence.go:274`, `logs/go-test.log`, `logs/independent-main-race.log#TestClosureR2DisputedFinalIndependence` |
| M4 | FIXED | The second coding gate rechecks active core route and implementation route claim. Review import has the distinct C03 gap. Evidence: `cmd/hma/transition.go:62`, `cmd/hma/route_approval_binding_test.go`, `logs/go-test.log` |
| M5 | FIXED | Scratch path, regular marker, canonical store/run content and exact untouched seed-record equality are checked; positive and negative tests pass. Evidence: `scripts/dogfood/hostinput/main.go:201`, `logs/go-test.log`, `logs/dogfood.log` |
| M6 | FIXED | Every harness refusal checks the cryptographic fingerprint of core and host files, excluding lock files. Harness passes 17 checks. Evidence: `scripts/dogfood/host-dogfood.sh:56`, `logs/dogfood.log`, `logs/shellcheck.log` |
| L1 | FIXED | Positive/negative attestation structure and through-record tests now exist and pass. Evidence: `internal/model/route_test.go`, `internal/model/gateb_r2_test.go`, `logs/go-test.log` |
| L3 | FIXED | CI includes the hardened host dogfood shell step; local invocation passes. Hosted GitHub Actions was not run. Evidence: `.github/workflows/ci.yml:51`, `logs/dogfood.log` |
| N1 | PARTIAL | Ordinary byte/mode/type/index changes refuse stale reviews. Uncommitted ignore rules still allow changed bytes to retain review identity and progress. C01. Evidence: `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix`, `internal/repostate/raw.go:102` |
| N2 | FIXED | CI validates actual root/base/head/diff and clean policy before and after capture. Wrong/dirty/mutated inputs refuse. CI deliberately supports clean checkout capture only. Evidence: `cmd/hma/ci.go:136`, `logs/go-test.log`, `logs/independent-main-race.log#TestClosureR2CIIndexAndRoots` |
| B1 | PARTIAL | Optional binding remains optional; ordinary A-to-B changes refuse. Hidden dirty bytes still advance in both binding forms. C01. Evidence: `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix` |
| B2 | FIXED | Hidden-index flags explicitly refuse; raw tracked-byte comparison sees configuration-hidden changes. CI mutation regressions pass. Evidence: `internal/repostate/snapshot.go:244`, `cmd/hma/gateb_r2_test.go`, `logs/go-test.log` |
| B3 | FIXED | CI capture refuses inherited repository selectors; Capture sanitizes selectors and forces original-object reads. Evidence: `cmd/hma/ci.go:70`, `internal/evidence/capture.go:145`, `logs/go-test.log`, `logs/independent-pilot-race.log#TestClosureR2PilotReplacementObjectView` |
| B4 | FIXED | CI refuses a populated dirty-worktree approval binding even after checkout becomes clean; absent-binding clean controls pass. Evidence: `cmd/hma/ci.go:190`, `cmd/hma/gateb_r2_test.go#TestGateBR2CIBoundWorktreeGone`, `logs/go-test.log` |
| B5 | FIXED | Core lock spans each host authorization check and host append. The five terminal conditions pass both-order tests for decision/preflight/dispatch/review. Evidence: `cmd/hma/host.go:110`, `logs/independent-main-race.log#TestClosureR2CoreHostOrdering` |
| B6 | FIXED | Host history evaluation and append share the host lock under the core lock. Conflicting unsuperseded initial decisions have one winner; nonconflicting store appends survive. Evidence: `cmd/hma/host.go:120`, `cmd/hma/gateb_r2_test.go#TestGateBR2ConcurrentHostDecisions`, `logs/go-test.log`, `logs/independent-host-race.log#TestClosureR2ConcurrentInitialization` |
| B7 | PARTIAL | Absolute first/successor and interrupted recovery crash images pass; fsync failures do not acknowledge. Relative store ancestry remains unsynchronized. C02. Evidence: `logs/independent-host-race.log`, `internal/host/store.go:316` |
| B8 | FIXED | Union of all applicable independence constraints prevents tied-rule and lower-tier weakening; stricter capabilities are retained. Evidence: `internal/engine/route_coherence.go:292`, `internal/engine/gateb_r2_test.go`, `logs/go-test.log` |
| B9 | FIXED | Seed comparison serializes all fields except the store-computed head, rejecting altered disposition, metadata and payload; marker and repeated-seed controls pass. Evidence: `scripts/dogfood/hostinput/main.go:203`, `scripts/dogfood/hostinput/gateb_r2_test.go`, `logs/go-test.log` |
| B10 | PARTIAL | Upstream early rewinds conservatively clear every other unit route. Graph resolution is absent; review re-attestation still appends after revocation. C03/C04. Evidence: `cmd/hma/transition.go:192`, `logs/independent-main-race.log#TestClosureR2PlanReplay`, `logs/independent-main-race.log#TestClosureR2ReviewAfterRewind` |
| B11 | FIXED | Current identity includes file type, symlink target, absence and owner execute; full CLI negatives refuse and unchanged controls pass. v1-v6 identities expire. Evidence: `internal/repostate/snapshot.go:177`, `logs/go-test.log`, `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix`, `logs/independent-main-race.log#TestClosureR2OldDomainsExpire` |
| B12 | FIXED | Owner-execute comparison bypasses core.filemode; mode-hidden CI and full-review regressions pass. Evidence: `internal/repostate/raw.go:158`, `logs/go-test.log`, `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix` |
| B13 | FIXED | Raw bytes detect conversion-hidden changes; tracked, nested, info and global attribute sources independently refuse clean capture. Evidence: `internal/repostate/raw.go:169`, `logs/independent-repostate.log#TestClosureR2AttributeSources`, `logs/go-test.log` |
| B14 | FIXED | Any HEAD or index gitlink refuses before capture/review. Historical gitlink deletion remains in the canonical diff under three ignore policies. Evidence: `internal/repostate/raw.go:67`, `internal/repostate/raw.go:94`, `logs/go-test.log`, `logs/independent-repostate.log#TestClosureR2HistoricalGitlinkPolicies` |
| B15 | FIXED | Forced case-sensitive enumeration finds aliases from local/include/environment policies. Full-review alias changes refuse. v7 dirty and v2 review domains retire predecessor identities. Evidence: `internal/repostate/snapshot.go:114`, `logs/independent-repostate.log#TestClosureR2AliasConfigSources`, `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix`, `logs/independent-main-race.log#TestClosureR2OldDomainsExpire` |
| B16 | FIXED | Actual top-level equality rejects subdirectories and foreign core.worktree before capture. Symlink and linked-worktree controls pass. Evidence: `internal/repostate/raw.go:195`, `logs/independent-main-race.log#TestClosureR2CIIndexAndRoots`, `logs/go-test.log` |
| B17 | FIXED | Snapshot reads original Git objects for SHA-1/SHA-256. Pilot output digest equals the original object with and without replacements and inherited GIT_NO_REPLACE_OBJECTS=0. Old clean review v1 refuses. Evidence: `logs/independent-repostate.log#TestClosureR2SHAReplacement`, `logs/independent-pilot-race.log#TestClosureR2PilotReplacementObjectView`, `logs/independent-main-race.log#TestClosureR2OldDomainsExpire` |
| B18 | FIXED | Absolute/relative/symlink top-level roots pass CI; subdirectory/foreign roots refuse. Linked worktrees pass existing tests. Evidence: `logs/independent-main-race.log#TestClosureR2CIIndexAndRoots`, `logs/go-test.log` |
| F01 | FIXED | Index content/mode/deletion/unmerged entries change identity and refuse old reviews in both binding forms. Pilot and CI refuse index mutation before append. v6 domain replay refuses. Evidence: `internal/repostate/snapshot.go:439`, `logs/independent-main-race.log#TestClosureR2FullArtifactMatrix`, `logs/independent-main-race.log#TestClosureR2CIIndexMutation`, `logs/independent-pilot-race.log#TestClosureR2PilotIndexMutation` |
| F02 | PARTIAL | Core/host lock ordering serializes both commit orders. Review import does not revalidate a revoked route and still appends. C03. Evidence: `logs/independent-main-race.log#TestClosureR2CoreHostOrdering`, `cmd/hma/host.go:262` |
| F03 | PARTIAL | Absolute store ancestry is synchronized; relative ancestry stops at dot and loses an acknowledged record in a crash image. C02. Evidence: `logs/independent-host-race.log#TestClosureR2RelativeStoreDurability`, `internal/host/store.go:316` |
| F04 | FIXED | Explicit --ignore-submodules=none preserves the nonempty historical gitlink diff across local diff, local submodule and tracked .gitmodules policy. Evidence: `internal/repostate/snapshot.go:261`, `logs/independent-repostate.log#TestClosureR2HistoricalGitlinkPolicies` |
| F05 | PARTIAL | Conservative invalidation stops U0/U1/U2/U3; no verified plan graph distinguishes the independent U3. New approval restores authority without resolving its plan. C04. Evidence: `cmd/hma/transition.go:179`, `logs/independent-main-race.log#TestClosureR2PlanReplay` |

## Evidence and reproduction

The authoritative sources and their SHA-256 digests are recorded in [requirements-derived.json](requirements-derived.json). That file was written before diff inspection. The target freeze is in [freeze-start.json](freeze-start.json) and [freeze-end.json](freeze-end.json). Raw binary diff bytes are retained in `diff-start.binary`. The frozen source copy is `target-copy/`. All probe sources are in `probes/`. Overlay maps bind them to the copied packages. [commands.jsonl](commands.jsonl) records argument vectors, working directories, start times, durations, log paths and exit codes. `GOCACHE` and `TMPDIR` were under this output directory.

Example independent reproduction from this output directory:

```sh
cd target-copy
export PATH="$HOME/.local/go/bin:$PATH"
export GOCACHE=/tmp/hma-gate-b-closure-20261016-r2/gocache
export TMPDIR=/tmp/hma-gate-b-closure-20261016-r2/tmp
go test -count=1 -race -v -overlay=../overlay.json ./cmd/hma -run "^TestClosureR2"
go test -count=1 -race -v -overlay=../host-overlay.json ./internal/host -run "^TestClosureR2"
go test -count=1 -race -v -overlay=../pilot-overlay.json ./internal/pilot -run "^TestClosureR2"
go test -count=1 -v -overlay=../repostate-overlay.json ./internal/repostate -run "^TestClosureR2"
```

The first three commands are expected to fail on the frozen target. Each failure has its assertion and observed output in the cited log. The host overlay adds filesystem observation wrappers. It preserves production syscall order and return values except at explicit fsync-fault positions. It changes no repository file. Earlier probe-development logs are retained. Final v3/race logs supersede the enum and digest fixture errors described below.

## Controlled unblinding addendum

The auditor froze Phase 1 at `c8f0a460c9498fea65cb426992936f6889ed2b138af9a4bd4c0498f940045c32` before reading `TODO.md:67-129`. The auditor did not open prior Gate B reviews. No Phase 1 finding changed. The final report adds the scope qualification for C01; Phase 1 remains unchanged.

TODO claims committed-ignore-only exclusion at line 108. C01 falsifies that claim. TODO claims F02/F03 completion at lines 123-126. C02 and C03 identify residual failures. TODO openly defers F05 to Gate D at lines 127-128. The authoritative audit scope includes F05. A TODO deferral does not change the closure criteria.

## Limits and unresolved questions

- Isolation is partial. This is a fresh audit context with no implementer conversation. The initial source-bundle command read the whole named Gate A report and consolidation handoff instead of only closure fields and section 5. Those allowed files contain prior non-Gate-B assessments. Output was truncated. This prevents a claim of perfect fact-only blinding. No prohibited prior Gate B directory, Gate A v*.md handoff, claude directory or TODO Gate B claims was opened before this freeze.
- Linux amd64, Go 1.27.1. No case-insensitive filesystem or non-Linux fsync/locking behavior was tested.
- All executions used a copied target and scratch repositories/stores under this output directory. No network, real provider dispatch, model API or human production approval was exercised.
- The durability model drops all unsynchronized directory entries and retains synchronized file bytes by inode. It covers 60 append and seven interrupted-recovery images, 20 reachable directory-fsync failure positions and 12-writer same/different-run tests. It is a model, not a physical power-loss test or exhaustive enumeration of every persistence subset.
- The named old external helper scripts and the exact historical 40/55 process-exit checkpoints were not read or executed before Phase 1. Auditor-authored equivalent probes and current repository regressions were used. No historical review verdict is treated as proof.
- Original-domain dirty replay probes construct v1-v6 framing for a regular untracked file; they do not execute every historical binary. Current schema separation, exact clean v1 construction and source inspection support the expiry result.
- CI clean checkout capture is supported; dirty CI capture is deliberately refused. Hosted GitHub Actions, release closure, portable schema publication and full host-MVP conformance are outside this Gate B verdict.
- Initial development probe logs contain corrected auditor fixture errors: uppercase enum literals in the first host probe and an omitted evidence-digest domain prefix in the first pilot probe. Final v3/race logs use corrected fixtures; the original logs remain available.
- No approved Jev tool was exposed by the host. Advisory decisions used the auditor path. No worker dispatch was performed; the supplied liz/model/effort identity is coordinator metadata, not a runtime route attestation.

The model permits loss of unsynchronized directory entries. It does not simulate physical hardware or prove every filesystem persistence ordering. Several named old tests live in blinded prior-review directories. They were replaced with disclosed auditor-authored probes, not claimed as executed. The table rests on current source, executed tests and explicit inferences from domain separation. No release readiness or full host-MVP claim follows from this result.

## Remediation sequence and exit criteria

Address C01 and C02 first. They affect artifact authorization and acknowledged-record durability. Address C03 in the same bounded host-authority review. Resolve C04 against the approved plan contract. Retain the unchanged controls and refusal-byte checks. A fresh independent audit must reproduce all four corrections before Gate B can close. No proposed correction has been implemented or validated by this audit. The report validator passed for `AUDIT.json`; this validates form only. See `logs/report-validation.log`.

No additional redesign opportunity is proposed.

## Required next action

**RETURN_GATE_B_TO_IMPLEMENTATION.**

Owner: the human Gate B orchestrator. Rationale: two observed high-severity failures remain, and the supplied closure checks are not all satisfied. Proposed route: the existing human-approved Gate B implementer route. Exact provider, model and effort are unavailable in this audit packet. The host must establish and report that route and perform its exact-route `hi` before dispatch. Proposed permission envelope: bounded remediation of C01-C04 in the cited snapshot, host-store and route-review paths, with tests. No commit, push, release or permission expansion is authorized by this audit. Required approval: a fresh human-approved remediation unit and eligible implementer/reviewer routes. Stop and escalate if scope, permissions or contract semantics must widen. Require fresh independent verification before Gate B closure.
