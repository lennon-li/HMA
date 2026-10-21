# HMA Gate D D5a: independent re-audit (dispatch hma-gate-d-audit-20261021-d5ar28m)

**Verdict: FAIL.** The R1, R2 and R3 repros from round d5ar27m are now refused. A new bypass in the same class remains. HMA's own git reads (Snapshot, changedPaths, Materialize) still read the live object store under the live repository configuration, and `VerifyRepositoryObjects` does not. When `core.multiPackIndex=false` is set, the live reads choose a forged pack copy of an object id that verification read from the genuine pack. Through `hma host credit`, I observed C1 credited PASSED on a target that holds a file the reviewed commit does not have. The same attack works when the setting sits in an include file under the exempt `.git/agent-dispatch`.

## 1. Scope, freeze, independence

- Target: `/home/yeli/repos/HMA`, `main` at `5f6f765517987fae8a573ff8ef6a0f27c3651f29`, plus the working tree. I excluded `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`.
- Freeze: the tracked diff sha256 was `408bd8ab…89974e3` at start and at end, which equals the expected value. `git status --short` was identical at start and end. The sha256 of all 15 untracked files was identical at start and end (`freeze-start-*` = `freeze-end-*`). HEAD did not change. I wrote nothing under `/home/yeli/repos` and made no git writes there.
- Phase 1: `PHASE1.json` (sha256 `4c7759a9…91ce5`, in `PHASE1.sha256`) was written before I read the tracked diff. **Deviation:** I wrote it after I read `verify.go`, `credit.go`, `fingerprint_linux.go` and the first part of `snapshot.go`. The file records this. I did not read `docs/handoffs/2026-10-19-gate-d-d5a.md` or any implementer narrative. My closure inputs were my prior report (d5ar27m) and its repro tests.
- Independence: the implementer is Claude Sonnet 5.5 and the auditor is Claude Opus 5.5. They are the **same vendor family**, so there is no provider separation. The context was fresh, but the dispatch named my prior round. Isolation is **partial**: the OS and user account are shared, and I kept the repository read-only by conduct, not by enforcement. All tests ran on copies (`pristine/` for the suite, `work/` for probes) with GOCACHE and TMPDIR under this directory.
- Sandbox: **bubblewrap 0.9.0 starts in this runtime** (`bwrap --unshare-all … true` succeeded). The sandbox tests ran and passed. Every credit probe ran its check through the real sandbox path.

What must hold (Gate A v3 §4.3; architecture §7, §9, §14; Gate C scope §0.1): a criterion is PASSED only when HMA re-captured a passing check on exactly the independently reviewed target. No state outside the fingerprinted tree or the verified store may decide what HMA reads (PHASE1 M1 to M7).

## 2. Closure of prior findings (re-run through `hma host credit`, `credit-prior-repros.log`)

| Prior | Status | What the repro now does |
|---|---|---|
| R1 commondir (high) | **Closed** | `TestAudit27CommonDirForgeryNonGitCheck`, `…GitCheck`, `…Control` and `…AlternatesNotRefused` all fail with `.git/commondir redirects git outside .git`. C1 stays PENDING. (`TestAudit27CommonDirFingerprintBlind` still "fails". It is a unit probe of `Fingerprint` alone, which by design does not walk an outside directory. Credit refuses commondir before that matters.) |
| R2 symlinks in .git (medium) | **Closed** | `TestAudit27SymlinkedObjectsDir` gives `a link or special file under .git: …/.git/objects`. C1 stays PENDING. `TestAudit25CreditSymlinkedGitDirAccepted` gives `.git is not a directory`. |
| R3 blocking config read (low) | **Repro closed, residual** | `TestAudit27CreditFIFOConfigHang` and `TestAudit27VerifyFIFOConfigIgnoresContext` return within 5 ms with "special file under .git". A swap race remains possible in principle (S3). |
| R4 redundant timer (info) | Open, unchanged | `cmd/hma/review.go:622-624` still uses `time.AfterFunc`. This is informational only. |
| Earlier N1/N2/N3 | Still closed | Forged loose object: fsck gives `… is corrupt`. Alternates: refused. |

Scope questions:
- **Order and TOCTOU (static and observed):** the walk runs first (`credit.go:130`), then `fpStart` (`:144`), then `rev-parse HEAD` (`:151`), then the walk again inside `VerifyRepositoryObjects` (`verify.go:63`, called from `credit.go:155`), and then Snapshot (`:158`). The second walk is inside the fingerprint window. An entry added after it and removed later changes the `.git` directory's ctime, so `fpAfter` refuses it. One exception exists: changes inside `.git/agent-dispatch` are neither walked nor fingerprinted (S2, observed).
- **Other .git files:** a symlinked HEAD or ref is refused by the walk. `packed-refs`, `shallow`, `info/grafts`, `worktrees/` and `modules/` are inside the fingerprint. Grafts and shallow change parents only. Snapshot's `diff base..head` and `ls-tree` are tree reads, so parents do not matter. Replace refs are disabled by `--no-replace-objects` (Output) and `GIT_NO_REPLACE_OBJECTS` (check). Hooks do not run for read-only commands, and the check uses `core.hooksPath=/dev/null`. A commit-graph is not used by `ls-tree` or `diff` to pick trees. **Configuration is the open channel:** `.git/config` and its includes (`include.path`, `includeIf`) are honored by every `repostate.Output` call (`snapshot.go:144-155`). Any include target outside the walk is unverified (S1, S2). `core.worktree` elsewhere is refused (`TestAudit27CoreWorktree`). Sparse checkout and submodules are refused by `changedPaths`.
- **Files outside .git:** `changedPaths` compares raw bytes, so `.gitattributes` filters and eol settings cannot hide a change. `.gitmodules` with a gitlink is refused (mode 160000). A nested untracked repository is listed as `dir/` and refused as not a regular file. Ignored ones are not copied.
- **Environment:** `Output` inherits the host environment minus the repository-selecting list (`snapshot.go:48-61`). It keeps `GIT_CONFIG_*` and reads the host's global and system config. The threat model trusts the host, so this is not a finding. It is another way live reads can differ from the fixed-config verification.

## 3. Findings

### S1. HIGH. Live git reads and object verification take different copies of an object id (observed)
- **Location:** `internal/repostate/snapshot.go:144-155` (`Output`: normal discovery, live `.git/config` with includes), used by `raw.go:48` (`changedPaths` ls-tree), `materialize.go:26` (Materialize ls-tree) and `snapshot.go:279-291`. `internal/evidence/gitsafe.go:188-216` (`buildPinnedStore`: fixed config, live objects as alternate). Call order: `cmd/hma/credit.go:155,158,203`.
- **Mechanism:** verification shows that a correct copy of every object reachable from HEAD can be read from the store **under fixed config**. HMA's own reads then read the same store **under live config**. With two packs holding the same id, the multi-pack index sends verification to the genuine pack. `core.multiPackIndex=false` sends live reads to the newer pack, which holds the forged tree. Git does not check the hash of a tree read during tree traversal (it only checks the object named directly).
- **Reproduction:** `repro/credit_audit_d5ar28m_test.go`, `TestAudit28MidxDivergenceNonGitCheck` (helper `a28MidxForgery`; a manual version is in `repro/midxdbg*.sh` and `forgeidx.py`). After the review is imported:
  1. `repack -a -d` makes genuine pack A.
  2. Build pack B (HEAD commit, root tree, and a tree listing `s/x`, with the idx renamed to the id of `HEAD:s`). B is newer than A.
  3. `multi-pack-index write --preferred-pack=A`, then `git config core.multiPackIndex false`.
  4. Stage `s/x` (ignored at the reviewed commit). The subtree `s` is shared by base and head, so the diff digest does not change.

  Live `ls-tree -r HEAD` = `a s/.gitignore s/k s/x`, and via the midx = `a s/.gitignore s/k`. Result: `credit err=<nil> … ExitCode:0 ReportStatus:pass Disposition:PASSED`, `C1=PASSED`, and `requireIndependentCredit` (the outcome gate) also accepts it. Control `TestAudit28MidxControl` (same packs, live config unchanged) is refused. The git-check variant `TestAudit28MidxDivergenceGitCheck` is refused (`git repository configuration refused`) because of the config allowlist.
- **Preferred fix (proposed):** read the target from the verified store. Run every git read in Snapshot, changedPaths and Materialize, and the `rev-parse HEAD` at `credit.go:151`, with `GIT_DIR=<priv>`, `GIT_OBJECT_DIRECTORY=<priv>/objects` (no alternates), `GIT_INDEX_FILE=<private copy of the live index, read once>`, `GIT_WORK_TREE=<root>`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL=/dev/null` and the fixed config that `buildPinnedStore` already writes. Build the store once per credit and reuse it for the check. `base` must be in it: it is an ancestor of the HEAD that `rev-list --objects` packed, or HMA must pack it explicitly and refuse otherwise.
- **Containment (small):** in `RequireSelfContainedRepository` or credit, read `.git/config` once (`git config --file <copy> --list --name-only`, so includes are not expanded) and refuse any key outside a credit allowlist. The allowlist excludes `include.*`, `includeif.*`, `core.multipackindex`, `core.commitgraph` and `pack.*`. Alternatively, refuse a repository with `objects/pack/multi-pack-index` or with an object id present in two packs. That alternative is narrower and more fragile.
- **Verification:** `TestAudit28MidxDivergenceNonGitCheck` and `TestAudit28MidxViaLedgerInclude` must end refused or with C1 FAILED. `TestAudit28NoFalseRefusalMaintained` and `TestAudit27OwnReadsKeepFingerprint` must stay PASSED.
- **Trade-offs:** the preferred fix adds one store build to each credit, which is already done once. The containment refuses some real repositories with custom config. The current `gitConfigAllowed` already does this for git checks.
- **Effort:** containment small; preferred fix medium.

### S2. MEDIUM. `.git/agent-dispatch` is a hiding place for git configuration (observed)
- **Location:** `internal/evidence/verify.go:37-44` (walk skips the ledger), `internal/repostate/fingerprint_linux.go:56-58` (fingerprint skips it). Git resolves a relative `include.path` against `.git`.
- **Reproduction:** `TestAudit28LedgerIncludeInvisible`: `include.path = agent-dispatch/x.cfg`. `x.cfg` is rewritten during the check (`creditCheckHook`), and credit gives `err=<nil> C1=PASSED`, so the change is not detected. `TestAudit28MidxViaLedgerInclude`: the S1 attack with the setting only in the ledger include also gives **C1=PASSED**.
- **Fix:** the S1 containment (refuse `include*` keys) closes this. The S1 preferred fix (fixed config) makes the ledger irrelevant to HMA's reads. More durable option: keep the host ledger outside `.git`, or have the walk refuse any regular file under `agent-dispatch` that live config names.
- **Effort:** small.

### S3. LOW. Lstat-then-read windows without a context remain (inferred; not observed to hang)
- **Location:** `verify.go:90` (`os.ReadFile(.git/config)` after the walk at `:38`), `gitsafe.go:76,111`, `materialize.go:73`, `raw.go:167`, `snapshot.go:230`.
- **Evidence:** `TestAudit28SwapRaceWorktree`: 60 credit attempts while tracked file `a` was swapped with a FIFO in a loop. There was no hang (the errors were dirty or ENOENT). `TestAudit28SwapRaceConfig` (×4): credit blocked for more than 15 s each time, but the goroutine dumps show the block was in the git subprocess `rev-parse --verify <base>^{commit}` (VerifyBase, under the 10-minute `boundedCtx`), not in an unbounded Go read (`credit-a28-swapcfg.log`). So the observed blocks are bounded at 10 minutes while both run locks are held. An unbounded `os.ReadFile` on a swapped-in FIFO stays possible in principle.
- **Fix:** open with `O_RDONLY|O_NONBLOCK|O_NOFOLLOW`, `fstat` the descriptor, require a regular file, then read. In credit, use a shorter bound for VerifyBase.
- **Effort:** small.

### No false refusal observed
`TestAudit28NoFalseRefusalMaintained` covers an annotated and a lightweight tag, `pack-refs --all`, `gc` with commit-graph, `commit-graph write`, `multi-pack-index write`, a registered linked worktree (`.git/worktrees/<n>/{commondir,gitdir}` are accepted because only top-level ones are refused), an executable hook and the sample hooks. Result: `credit err=<nil> C1=PASSED`. `TestAudit27OwnReadsKeepFingerprint` gives PASSED with a stable fingerprint for both the non-git and the git check. Expected refusals: a linked worktree or `--separate-git-dir` checkout as the target (`.git` is not a directory), `clone --shared` (alternates), and a repository with submodules (mode 160000). These are refused by static reading and by prior repros. I did not run a new end-to-end test for them.

## 4. Regression (pristine copy)

- `go vet ./...`: exit 0.
- `go test -count=1 ./...`: 1043 PASS, 2 FAIL, 2 SKIP. The two failures (`TestD5HMABoundLaunchRequiresEligibleTrue`, `TestD5ResumeReplayUsesSameDispatchID`) are environmental: `error obtaining VCS status: exit status 128` comes from a stray `/tmp/.git` above the copy. With `GOFLAGS=-buildvcs=false`, the result is **1045 PASS, 0 FAIL, 2 SKIP**.
- Skipped: `internal/host.TestGateBR2HostChild` (child-process helper) and `internal/host.TestGateBR2HostPowerLossImage`. No D5a, evidence or repostate test was skipped.
- `go test -race -count=1 ./cmd/hma ./internal/evidence ./internal/repostate` (`-buildvcs=false`): 492 PASS, 0 FAIL, 0 SKIP, no `DATA RACE`.
- Logs: `vet.log`, `test.json`, `test-novcs.json`, `race.json`, `credit-*.log`.

## 5. Limitations
- Same vendor family, partial isolation, and PHASE1 was written after reading part of the code (§1).
- S1 was shown with git 2.43.0. Other git versions may order packs differently. The root cause (live config vs. fixed config over the same store) does not depend on the version.
- I did not exhaustively list other config keys that change object selection. The fix should not depend on such a list.
- The S3 unbounded-read race was not observed.

## 6. Next action
Route Snapshot, changedPaths, Materialize and the credit `rev-parse HEAD` through the verified private store under fixed config (S1 preferred fix). Meanwhile, refuse credit when live `.git/config` has any key outside an allowlist, which also closes S2. Then request an independent re-audit with `repro/credit_audit_d5ar28m_test.go` as the closure input.
