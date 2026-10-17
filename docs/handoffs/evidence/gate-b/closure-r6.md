# Gate B independent closure audit

**Verdict: FAIL.** One high-severity defect remains. A predecessor clean review can still authorize Verification after an internal tracked directory changes from a symlink to a real directory.

Target: `main` at `67965541b7032a585a68e48f3d10f4a998d5399e`, plus the frozen working tree. The final head, branch, status, diff digest and all untracked hashes match the initial freeze. The source repository was not modified.

Coverage: **44 FIXED; 5 PARTIAL; 0 OPEN** across 49 listed IDs. N1 and R5-01 remain partial because of the same observed defect. F05, C04 and R3-03 remain partial and are explicitly deferred to Gate D. They do not block this Gate B audit. Conservative all-unit invalidation remains in force.

**Isolation: isolated blind phase.** This is a fresh auditor session. No implementer chat or earlier Gate B review was read before Phase 1. Repository comments, tests and Task 14 were treated as artifacts. Review was read-only by conduct. The workspace sandbox was not an enforced read-only mount.

Phase 1 SHA-256: `3e9cb0ada2b650cc12fd0d00df1ea92bafe5068099d8f746946cf29ce799facc`. The immutable findings are in [PHASE1.json](/tmp/hma-gate-b-closure-20261016-r6/PHASE1.json). The hash was recorded before reading TODO.md. The final JSON uses report-contract `fix-required`; `gate_verdict: FAIL` is the same verdict in the requested vocabulary.

## Observed verification

| Check | Result | Evidence |
| --- | --- | --- |
| Untouched-source test suite | PASS | [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| Vet | PASS | [vet.log](/tmp/hma-gate-b-closure-20261016-r6/vet.log) |
| Untouched-source race suite | PASS | [race-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/race-isolated.log) |
| Synthetic CLI dogfood | PASS, 17 checks | [dogfood.log](/tmp/hma-gate-b-closure-20261016-r6/dogfood.log) |
| Independent full artifact matrix | 24/26 pass; 2 legacy clean replays fail | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log) |
| Core API authority ordering | PASS, 24 cases | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| Ordinary/final/disputed/high/critical review boundary | PASS, 15 cases | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| Crash and directory-loss model | PASS, 241 checkpoint runs | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| Intermediate symlink parent fsync faults | PASS | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| Raw byte/index/root/replacement probes | PASS | [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log), [probes-index-capture.log](/tmp/hma-gate-b-closure-20261016-r6/probes-index-capture.log), [probes-pilot.log](/tmp/hma-gate-b-closure-20261016-r6/probes-pilot.log) |
| Concurrent initialization, two runs | PASS under race detector | [probes-concurrency.log](/tmp/hma-gate-b-closure-20261016-r6/probes-concurrency.log) |

The first suite and race runs failed one environment-dependent dogfood unit test. It assumed the source directory lay outside `os.TempDir()`. The source copy was under `/tmp`. Reruns with a dedicated `TMPDIR` passed without source changes. The initial results remain in [test.log](/tmp/hma-gate-b-closure-20261016-r6/test.log) and [race.log](/tmp/hma-gate-b-closure-20261016-r6/race.log).

## R6-01: predecessor clean review authority does not expire

**Severity: high. Certainty: observed. Affected checks: N1 and R5-01.**

The independent reproduction uses this sequence:

1. Commit `dir/tracked` and ignore the directory path and an alternate copy.
2. Replace `dir` with a symlink to the alternate directory. Keep the tracked leaf bytes equal.
3. Run a controlled predecessor observation. It uses leaf `Lstat` and worktree domain v7. It reports a clean worktree because it follows the parent symlink.
4. Record an independent review with the predecessor `hma.host.review-artifact/v2` digest.
5. Restore a real directory with equal tracked bytes. Keep the core chain and recorded review fixed.
6. Present a valid fresh human approval for the live checkout. The current CLI appends `IMPLEMENTATION_REVIEW -> VERIFICATION`.

Expected result: refusal until a fresh independent review binds the corrected observation contract. Observed result: acceptance and a new core transition. The phase-1 reproduction appears at [probes-artifact-final.log:102](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log:102). A post-freeze challenge hard-codes the predecessor v2 schema instead of calling the current digest function. It reproduces the same failure: [post-phase1-legacy-confirmation.log](/tmp/hma-gate-b-closure-20261016-r6/post-phase1-legacy-confirmation.log).

The root cause is visible in [internal/host/host.go:643](/home/yeli/repos/HMA/internal/host/host.go:643). The clean review still uses artifact schema v2. The worktree v8 change in [internal/repostate/snapshot.go:178](/home/yeli/repos/HMA/internal/repostate/snapshot.go:178) only affects nonempty dirty digests. An empty clean worktree digest carries no new observation version.

Fresh clean and dirty controls pass. Current byte, executable mode, parent type, staged content, staged mode, staged deletion, unmerged stages, ignore-rule and case-alias changes refuse the old review. Reconstructed predecessor v7 dirty reviews also refuse. The missing migration is specific to predecessor clean reviews.

The reproduction runs both requested optional-binding branches. Both serialize an absent `WorktreeDigest` because the observed worktree is clean. These are two runs of the clean case. They are not evidence of a populated clean binding. Separate populated dirty-binding controls pass.

**Proposed fix:** advance the review artifact observation domain after the parent-component correction. For example, use `hma.host.review-artifact/v3`. Require a fresh independent review for predecessor v2 artifacts. Keep stored history append-only. Add a regression that calculates the predecessor v2 digest explicitly.

**Verification:** the predecessor clean replay must refuse and preserve all record and head bytes. A new independent review of the supported real-directory checkout must pass with valid human approval. Current dirty controls must pass with the optional approval binding absent and populated. Repeat the affected test, vet, race and dogfood checks.

**Trade-off:** a domain bump also expires valid v2 reviews. They require another independent review. **Alternative:** persist and check a separate observation-version field in the immutable review contract. That is a larger compatibility change. The old clean values do not identify which v2 observations were incomplete. **Effort:** small. **Fix state:** proposed; no implementation was changed or validated.

## Requirement coverage

FIXED means the inspected implementation and independently executed evidence satisfy the bounded closure behavior. It does not claim exhaustive platform or filesystem coverage. PARTIAL identifies an observed remaining subcheck or the explicit Gate D deferral.

| ID | Closure status | Observation | Evidence |
| --- | --- | --- | --- |
| H1 | FIXED | CI projects current stage and searches for the current visit approval. Transition/resolution regressions pass. | [cmd/hma/ci.go:85](/home/yeli/repos/HMA/cmd/hma/ci.go:85), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| H2 | FIXED | Grounding and Planning pilot captures accept absent head/diff bindings. Later-stage stale bindings still refuse. | [internal/pilot/run.go:94](/home/yeli/repos/HMA/internal/pilot/run.go:94), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| H3 | FIXED | Host run lock, pending intent, atomic publication and directory sync recover verified prefixes. Independent first/successor/recovery checkpoints, concurrency and injected ancestry sync failures pass. | [internal/host/store.go:232](/home/yeli/repos/HMA/internal/host/store.go:232), [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| H4 | FIXED | All four host write commands refuse the five supported terminal outcomes. Core API abort/write interleavings serialize. Host show remains an inspection operation. | [cmd/hma/host.go:111](/home/yeli/repos/HMA/cmd/hma/host.go:111), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| M1 | FIXED | CI capture enforces outside-repository store isolation, including symlink store aliases. | [cmd/hma/ci.go:72](/home/yeli/repos/HMA/cmd/hma/ci.go:72), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| M2 | FIXED | Resolve uses the shared strict decoder; trailing JSON and unknown-field refusals pass. | [cmd/hma/resolve.go:46](/home/yeli/repos/HMA/cmd/hma/resolve.go:46), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| M3 | FIXED | Route coherence enforces all applicable constraints and high/critical family floors. Final/disputed host dispatch enforces family separation independently of risk-tier rule selection. | [internal/engine/route_coherence.go:275](/home/yeli/repos/HMA/internal/engine/route_coherence.go:275), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| M4 | FIXED | Second coding gate revalidates current route approval before artifact comparison. Revoked review import and core API rewinds refuse. | [cmd/hma/transition.go:74](/home/yeli/repos/HMA/cmd/hma/transition.go:74), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| M5 | FIXED | Synthetic marker binds canonical temporary store/run, must be regular, and approval seeding accepts only the exact untouched seed. Copied/symlink/directory markers and altered seeds refuse. | [scripts/dogfood/hostinput/main.go:147](/home/yeli/repos/HMA/scripts/dogfood/hostinput/main.go:147), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| M6 | FIXED | Dogfood refusal fingerprint covers store file names and bytes, excluding lock files; all 17 harness checks pass. | [scripts/dogfood/host-dogfood.sh:62](/home/yeli/repos/HMA/scripts/dogfood/host-dogfood.sh:62), [dogfood.log](/tmp/hma-gate-b-closure-20261016-r6/dogfood.log) |
| L1 | FIXED | Direct attestation positive/negative fixtures cover required fields and closed execution-context vocabulary. | [internal/model/route_test.go](/home/yeli/repos/HMA/internal/model/route_test.go), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| L3 | FIXED | CI workflow now runs the same dogfood harness that passed independently. Hosted GitHub runner execution itself was not observed. | [.github/workflows/ci.yml:51](/home/yeli/repos/HMA/.github/workflows/ci.yml:51), [dogfood.log](/tmp/hma-gate-b-closure-20261016-r6/dogfood.log) |
| N1 | PARTIAL | Current dirty artifact changes refuse old reviews, with optional binding absent/present. A predecessor clean v2 review survives the parent-symlink correction. See R6-01. | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log) |
| N2 | FIXED | Real repository/base/head/diff, unapproved dirt, directory type and mutating-command negatives pass. Valid clean, dirty-bound, root-alias and linked-worktree controls pass. | [cmd/hma/main_test.go:442](/home/yeli/repos/HMA/cmd/hma/main_test.go:442), [probes-cli.log](/tmp/hma-gate-b-closure-20261016-r6/probes-cli.log) |
| B1 | FIXED | Live host-declared dirty digest binds the review even when the approval field is absent. Both optional-binding forms pass unchanged controls and reject current artifact mutations. | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log) |
| B2 | FIXED | Hidden index flags explicitly refuse; raw tracked bytes bypass Git status/config conversion. Mutation and mode capture negatives preserve store bytes. | [cmd/hma/gateb_r2_test.go:77](/home/yeli/repos/HMA/cmd/hma/gateb_r2_test.go:77), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B3 | FIXED | CI refuses inherited repository selectors; Capture strips selectors and forces original Git objects. Replacement-view direct and pilot probes pass. | [internal/repostate/snapshot.go:64](/home/yeli/repos/HMA/internal/repostate/snapshot.go:64), [probes-cli.log](/tmp/hma-gate-b-closure-20261016-r6/probes-cli.log) |
| B4 | FIXED | CI refuses a populated dirty-worktree binding even after the checkout becomes clean; absent clean binding remains accepted. | [cmd/hma/ci.go:189](/home/yeli/repos/HMA/cmd/hma/ci.go:189), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| B5 | FIXED | Host check and durable host commit share the same core lock. Core API abort either precedes host refusal or follows completed host commit. | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| B6 | FIXED | Host history evaluation and commit share the host run lock. Conflicting unsuperseded initial decisions serialize and later conflicts refuse. | [cmd/hma/gateb_r2_test.go:315](/home/yeli/repos/HMA/cmd/hma/gateb_r2_test.go:315), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| B7 | FIXED | Directory syncing precedes acknowledgment. Independent model retains fsynced directory entries and loses unsynchronized entries. First/successor/recovery prefixes and acknowledgments survive 241 checkpoints. | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| B8 | FIXED | Effective independence is the union of applicable capabilities and separation constraints, including tied and lower-tier rules. | [internal/engine/route_coherence.go:287](/home/yeli/repos/HMA/internal/engine/route_coherence.go:287), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| B9 | FIXED | Synthetic seed comparison marshals every record field after removing only the computed head anchor. Nonseed criterion/metadata/payload records refuse. | [scripts/dogfood/hostinput/main.go:196](/home/yeli/repos/HMA/scripts/dogfood/hostinput/main.go:196), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| B10 | FIXED | In Gate B scope, any unit rewind to Planning or earlier conservatively clears other units route approvals. Revoked review import refuses. Plan-derived preservation is deferred by the supplied human scope decision. | [cmd/hma/transition.go:224](/home/yeli/repos/HMA/cmd/hma/transition.go:224), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| B11 | FIXED | Owner-execute, raw bytes, type, absence and staged entry identity are bound. Current mode/type artifact controls and predecessor dirty-domain retirement pass. | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| B12 | FIXED | core.filemode=false cannot hide owner-execute from raw comparison. CI preexisting/mutation negatives and full review controls pass. | [cmd/hma/gateb_r4_ci_hidden_mode_test.go](/home/yeli/repos/HMA/cmd/hma/gateb_r4_ci_hidden_mode_test.go), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B13 | FIXED | Raw blob comparison detects normalization-hidden bytes independently of attributes. Info/nested and capture regressions pass. | [internal/repostate/raw.go:153](/home/yeli/repos/HMA/internal/repostate/raw.go:153), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B14 | FIXED | Current HEAD and index gitlinks explicitly refuse; ignore policy cannot suppress an accepted submodule snapshot. Historical removal diff stays bound. | [internal/repostate/raw.go:65](/home/yeli/repos/HMA/internal/repostate/raw.go:65), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B15 | FIXED | Observation forces core.ignoreCase=false. Case-alias current CI/raw/full-review changes are detected; dirty digest domain retires earlier incomplete identities. | [internal/repostate/snapshot.go:119](/home/yeli/repos/HMA/internal/repostate/snapshot.go:119), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B16 | FIXED | Snapshot requires resolved actual top level. Subdirectory/foreign configured root refuse; valid symlink and relative root controls pass. | [internal/repostate/raw.go:228](/home/yeli/repos/HMA/internal/repostate/raw.go:228), [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| B17 | FIXED | Snapshot ignores replacement refs; Capture and pilot force GIT_NO_REPLACE_OBJECTS=1. Review artifact v2 retires earlier v1 clean identities. SHA-1/SHA-256 and inherited-variable checks pass. | [internal/repostate/snapshot.go:64](/home/yeli/repos/HMA/internal/repostate/snapshot.go:64), [probes-cli.log](/tmp/hma-gate-b-closure-20261016-r6/probes-cli.log) |
| B18 | FIXED | Absolute, relative, root symlink and linked-worktree controls pass, while invalid nested roots refuse. | [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| F01 | FIXED | Independent full CLI variants bind staged content, mode, deletion and unmerged index stages. Index-mutating capture commands refuse without store mutation. | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log), [probes-index-capture.log](/tmp/hma-gate-b-closure-20261016-r6/probes-index-capture.log) |
| F02 | FIXED | All host authorization operations run check plus durable commit while holding core authority lock. Valid core API abort and unit/whole rewinds serialize in both orders. | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| F03 | FIXED | Every ancestry lookup directory is synchronized, even if it existed earlier. Fsync faults never acknowledge. Independent same/different-run initialization and process/directory-loss recovery pass. | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log), [probes-concurrency.log](/tmp/hma-gate-b-closure-20261016-r6/probes-concurrency.log) |
| F04 | FIXED | Canonical diff explicitly sets --ignore-submodules=none; historical gitlink removal diff stays nonempty under diff.ignoreSubmodules=all. | [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log) |
| F05 | PARTIAL (Gate D) | Plan-derived dependency graph remains unimplemented and is explicitly deferred to Gate D by the 2026-10-16 human scope decision. Conservative all-unit invalidation is retained and tested. Not a Gate B blocker. | [requirements-neutral.json:note](/tmp/hma-gate-b-closure-20261016-r6/requirements-neutral.json), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| C04 | PARTIAL (Gate D) | Plan-derived dependency graph remains unimplemented and is explicitly deferred to Gate D by the 2026-10-16 human scope decision. Conservative all-unit invalidation is retained and tested. Not a Gate B blocker. | [requirements-neutral.json:note](/tmp/hma-gate-b-closure-20261016-r6/requirements-neutral.json), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| R3-03 | PARTIAL (Gate D) | Plan-derived dependency graph remains unimplemented and is explicitly deferred to Gate D by the 2026-10-16 human scope decision. Conservative all-unit invalidation is retained and tested. Not a Gate B blocker. | [requirements-neutral.json:note](/tmp/hma-gate-b-closure-20261016-r6/requirements-neutral.json), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| C01 | FIXED | An added or changed .gitignore causes unfiltered untracked enumeration. Hidden-file byte mutation changes identity and refuses old review. | [probes-raw.log](/tmp/hma-gate-b-closure-20261016-r6/probes-raw.log), [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log) |
| C02 | FIXED | Relative store paths are made absolute before complete ancestry traversal. Relative-store regression and fsync probes pass. | [internal/host/store.go:369](/home/yeli/repos/HMA/internal/host/store.go:369), [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| C03 | FIXED | Review import revalidates current core route authority under the same core lock used through commit. Revoked-route and core-first API tests refuse. | [cmd/hma/host.go:264](/home/yeli/repos/HMA/cmd/hma/host.go:264), [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| R3-01 | FIXED | CLI transition holds the host lock from review gate through core append, serializing an imported BLOCK review. Retained scheduling test passes. | [cmd/hma/transition.go:333](/home/yeli/repos/HMA/cmd/hma/transition.go:333), [test-isolated.log](/tmp/hma-gate-b-closure-20261016-r6/test-isolated.log) |
| R3-02 | FIXED | Symlink store ancestry includes lexical/link-target lookup parents. Independent mixed absolute/relative intermediate-hop fixture and fsync failures pass. | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| R3-04 | FIXED | Task 14 now describes conservative cross-unit earlier-gate invalidation and its Gate D boundary. | [docs/task14-host-orchestration.md:154](/home/yeli/repos/HMA/docs/task14-host-orchestration.md:154) |
| R4-01 | FIXED | Core lock covers actual host commit, not only record construction. Own valid API race covers all four operations, abort and both rewind scopes, both orders. | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |
| R4-02 | FIXED | Intermediate symlink parents are collected while resolving each hop. Injected fsync failure at every collected directory refuses acknowledgment. Directory-loss model passes. | [probes-fs-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-fs-final.log) |
| R5-01 | PARTIAL | Current symlink/non-directory parent changes are detected, and v7 dirty reviews expire. Clean v2 predecessor review identity is unchanged and advances after restoring a real directory. See R6-01. | [probes-artifact-final.log](/tmp/hma-gate-b-closure-20261016-r6/probes-artifact-final.log), [probes-cli.log](/tmp/hma-gate-b-closure-20261016-r6/probes-cli.log) |
| R5-02 | FIXED | Supported host dispatch exposes disputed/final flags and applies provider/model-family separation at low risk. Own command-boundary same-provider/same-model negatives and separated positives pass. | [probes-authority.log](/tmp/hma-gate-b-closure-20261016-r6/probes-authority.log) |

The JSON report retains every evidence pointer and the full remediation checks. The accepted Gate D deferral comes from the supplied neutral criteria note, not from later TODO claims.

## Evidence and reproduction

Freeze records: [freeze-start.json](/tmp/hma-gate-b-closure-20261016-r6/freeze-start.json), [freeze-end.json](/tmp/hma-gate-b-closure-20261016-r6/freeze-end.json). Binary diff SHA-256: `49745200d9d7fd651ee9e178f806b1a9410c6b1df04fdb39085c974bf91a9ee3`. The freeze includes hashes of all untracked files. `.serena/` and the two architecture amendment drafts were excluded from substantive review and test copies.

The untouched-source suite ran in `baseline/`. Independent probes ran in `repo/`. Persistent probe sources and overlays are in `probes/`. The filesystem tests require `fs-overlay.json`; they instrument only the audit copy. The helper reconstructs the specified predecessor behavior. It is not an authenticated historical binary.

Use `~/.local/go/bin` first in `PATH`. Set `GOCACHE=/tmp/hma-gate-b-closure-20261016-r6/gocache`. For the untouched-source suite, set `TMPDIR=/tmp/hma-gate-b-closure-20261016-r6/scratch`. Run from `baseline/`:

```text
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

The exact independent probe commands are in `AUDIT.json` under `evidence_ledger`. Probe commands run from `repo/`. [probes/auditor_r6_test.go](/tmp/hma-gate-b-closure-20261016-r6/probes/auditor_r6_test.go) preserves the phase-1 probe. [probes/auditor_r6_postphase_test.go](/tmp/hma-gate-b-closure-20261016-r6/probes/auditor_r6_postphase_test.go) preserves the stronger hard-coded predecessor challenge used by the final `repo/` copy. [evidence-manifest.json](/tmp/hma-gate-b-closure-20261016-r6/evidence-manifest.json) hashes the retained logs, reports and probe sources.

## Controlled unblinding

After the Phase 1 hash was recorded, TODO.md was read only as an audited artifact. Its N1/R5-01 completion claims do not resolve the frozen executable failure. Its Gate D deferral agrees with the authoritative neutral scope note. No prior Gate B review directory or Gate A narrative was opened. The unblinding did not change the verdict.

A post-freeze challenge replaced the review digest call in the legacy probe with an explicit predecessor v2 schema calculation. The clean replay still passes incorrectly. The predecessor dirty and fresh clean/dirty controls pass their expected checks.

One Phase 1 limitation needed a wording correction. Persistent artifacts are under this output directory. Some independent Go probes used automatically removed `/tmp/Test...` fixture directories. This correction is recorded in [unblinding-addendum.json](/tmp/hma-gate-b-closure-20261016-r6/unblinding-addendum.json). The frozen Phase 1 file was not changed.

## Limits and omitted checks

- Fresh session with only the user dispatch and neutral/authoritative requirement sources. No implementer conversation or earlier Gate B reviews were read. Source comments, implementation tests and the changed Task 14 document were inspected as artifacts; their explanations are not proof.
- Read-only by conduct and unchanged final source freeze, not an OS-enforced read-only mount. All persistent audit artifacts and caches are under this output directory. Independent Go probes also used automatically removed default /tmp/Test... fixture directories. The dogfood harness created and removed its /tmp scratch store. No source files or Git state were written.
- Linux/amd64, Go 1.27.1. No Windows, macOS, case-insensitive filesystem, actual power cut, or hosted GitHub Actions run. Filesystem model assumes successful file fsync persists file data, and models loss of unsynchronized directory entries. It tests the all-loss image and ordinary process restart at each checkpoint, not every arbitrary subset/reordering or physical device behavior.
- The precise earlier named probe scripts and their 40/55 checkpoints were not read or executed because earlier review directories were withheld. Independent equivalent probes were built instead; 51 first-append, 89 successor and 101 recovery checkpoint runs passed. Initial malformed auditor test files failed to compile before correction; those setup failures are not product defects. Final probe logs contain corrected executions. Initial auditor-only compile errors remain in the session tool trace.
- Initial test and race runs failed one test because the test assumed its repository source directory lay outside os.TempDir. Reruns of the untouched source copy with TMPDIR set to a dedicated scratch directory passed. The initial failures remain in test.log and race.log.
- The predecessor helper reconstructs only the specified v7 behavior: old raw leaf Lstat that follows parent symlinks, and worktree domain v7. No earlier implementation tree or earlier review packet was read. This is an executable controlled reconstruction, not an authenticated historical binary.
- Clean observations have an empty Worktree digest. The legacy-clean reproduction runs both requested binding branches, but both have an absent serialized optional WorktreeDigest. Populated dirty binding cases are separate and pass.
- Exact worker/model/effort route was supplied by the dispatch as liz/gpt-6.1-sol/high. The host does not expose a separate cryptographic runtime model attestation. No subagent, nested agent CLI, model API, external messages, git writes, or source edits were used.
- Jev is unavailable in the exposed host tool inventory. The auditor used the orchestrator decision fallback. No Jev result was fabricated.
- F05, C04 and R3-03 are PARTIAL but nonmandatory for Gate B under the supplied human scope decision. This audit does not claim full host MVP conformance or release readiness.

No additional redesign or optional implementation is recommended. The evidence supports one bounded compatibility correction.

## Required next action

**Recommendation: return Gate B to the human owner for a bounded R6-01 remediation assignment and independent re-audit.**

Rationale: the stale clean review still authorizes a coding transition. Proposed owner: the human Gate B owner. Proposed execution route: the original Opus implementer, followed by a distinct eligible Liz review route. Exact provider/model/access-service routes require recorded human authorization and the exact-route `hi` preflight before any new dispatch.

Permission envelope: review-observation identity/version and its regression tests only. Preserve the safety kernel, optional approval binding, route coherence and append-only history. No commit, push or release is authorized by this audit. Exact approval required: the human authorizes the bounded remediation unit and eligible implementation/review routes. Keep Gate B open. Stop and escalate on permission denial, route unavailability, budget exhaustion, source drift or a remaining failed replay.
