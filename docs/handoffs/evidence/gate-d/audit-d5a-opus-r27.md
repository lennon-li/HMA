# HMA Gate D D5a: independent re-audit (dispatch hma-gate-d-audit-20261021-d5ar27m)

**Verdict: FAIL.** The four prior repros (N1 to N4 of round d5ar25m) are now refused. A new bypass of the same class remains. The repository's `.git/commondir` file can point git at an object store outside `.git`. Snapshot and Materialize read the forged store. `VerifyRepositoryObjects` verifies only `.git/objects`, and the fingerprint does not walk the outside store. Through `hma host credit`, I observed a criterion credited PASSED on a target that contains a file the reviewed commit does not have.

## 1. Scope, freeze, independence

- Target: `/home/yeli/repos/HMA`, `main` at `5f6f765517987fae8a573ff8ef6a0f27c3651f29`, plus the working tree. I excluded `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`.
- Freeze at start and end: tracked diff sha256 was `408bd8ab…89974e3` both times, which equals the expected value. `git status --short` gave the same 29 lines both times. The untracked file hashes are identical (`freeze-untracked.txt` = `freeze-end-untracked.txt`). HEAD did not change. I wrote nothing under `/home/yeli/repos`.
- `PHASE1.json` (sha256 `42970e80…87cc4`, in `PHASE1.sha256`) was written before I read the diff. I did not read `docs/handoffs/2026-10-19-gate-d-d5a.md` or any implementer narrative. The closure input was my prior report and repros only.
- Independence: the implementer is Claude Sonnet 5.5 and the auditor is Claude Opus 5.5. They are the **same vendor family**, so there is no provider separation. The context was fresh. Isolation is **partial**: the OS and user account are shared, and read-only access to the repository was kept by conduct, not enforced. All tests ran on copies (`pristine/` for the suite, `work/` for the probes) with GOCACHE and TMPDIR under this directory.
- Sandbox: **bubblewrap 0.9.0 starts in this runtime.** Every credit repro ran its check inside the real sandbox.

What must hold (Gate A v3 §4.3; architecture §7.1, §14; Gate C scope §0.1): a criterion is PASSED only when HMA re-captured a passing check on exactly the independently reviewed target. The adversary can change files in the live repository before and during credit, including `.git`.

## 2. Closure of prior findings (re-run through `hma host credit`)

| Prior | Status | Evidence |
|---|---|---|
| N1 persistent forged subtree + index entry | **Repro closed** | `TestAudit25CreditPersistentForgeryNonGitCheck` gives `pinned git view refused: object verification failed: … pack … is corrupt`, and C1 stays PENDING. The git-check variant is refused the same way. |
| N2 alternates | **Repro closed** for `.git/objects/info/alternates` | `TestAudit25CreditFIFOAlternatesHang` returns at once: `alternate object stores are not supported`. **Not closed in general**: see R1 (alternates inside a commondir store are accepted). |
| N3 non-directory `.git` | **Repro closed** | `TestAudit25CreditSymlinkedGitDirAccepted` gives `.git is not a directory`. `core.worktree` pointing elsewhere is refused by `requireTopLevel` (`TestAudit27CoreWorktree`). **Not closed in general**: see R1 (commondir) and R2 (symlinks inside `.git`). |
| N4 unbounded contexts | **Mostly closed** | Git subprocesses run under ctx: credit 30 min (`credit.go:139`), VerifyBase 10 min (`review.go:614`). One context-free blocking read remains (R3). |

Other questions from the scope:
- No false refusal from HMA's own reads. `TestAudit27OwnReadsKeepFingerprint`: for both a non-git check (`/usr/bin/true`) and a git check (`git status`), the fingerprint is equal before and after a full credit, and C1=PASSED.
- HEAD different from the reviewed commit: `before.Head` must equal the live HEAD (`credit.go:162`), and the artifact digest binds the head, so this is refused (static).
- Replace refs, grafts and shallow: verification runs with a private `GIT_DIR`, so the live replace refs, grafts and shallow file are not consulted. Snapshot uses `--no-replace-objects`. In the private store, `fsck` sees the true parents. No bypass found.
- GIT_* environment: `gitEnv` strips the repository-selecting variables (`snapshot.go:48-61`). The pinned store build and fsck use an explicit environment. Global and system config are still read by Snapshot, but they are host-owned, so this is outside the threat model.
- `.git/agent-dispatch` exemption: alternates there are now unreachable, because `.git/objects/info/alternates` is refused. A relative `commondir` pointing into `agent-dispatch/` is a variant of R1.

## 3. Findings

### R1. HIGH. `.git/commondir` redirects git's object store, refs and config outside the verified and fingerprinted area; a persistent forgery gets credited PASSED (observed)
- **Location:** `internal/evidence/verify.go:16-25` checks only that `.git` is a directory and that `.git/objects/info/alternates` is absent. `verify.go:82` verifies `filepath.Join(gitDir, "objects")`. `internal/repostate/snapshot.go:144-150` (`Output`) uses normal git discovery, which honors `commondir`. `fingerprint_linux.go:39` walks only the paths under the root. Call order: `cmd/hma/credit.go:130,144,155,158,203`.
- **Mechanism:** git reads `$GIT_DIR/commondir` in any git directory, not only in linked worktrees. If `.git/commondir` holds `/elsewhere`, then objects, refs, config and alternates come from `/elsewhere`. HEAD and the index stay in `.git`. `.git/objects` can stay genuine, so verification passes. Snapshot, `changedPaths` and `Materialize` read the forged store. Nothing under `/elsewhere` is fingerprinted.
- **Reproduction:** `repro/credit_audit_d5ar27m_test.go`, `TestAudit27CommonDirForgeryNonGitCheck`. After the review is imported:
  1. Copy objects, refs and config to an outside directory, and write `.git/commondir`.
  2. In the outside store, overwrite `HEAD:s` with a tree that also lists `x`.
  3. Write `s/x`, which is ignored at the reviewed commit, and stage it.

  Live `ls-tree -r HEAD` lists `s/x`; the genuine store does not. Result: `credit err=<nil> … ExitCode:0 … Disposition:PASSED`, `C1=PASSED`, and `requireIndependentCredit` (the outcome gate) also accepts it. The control `TestAudit25CreditControlNoForgery` gives C1=FAILED. More repros of the same class:
  - `TestAudit27CommonDirForgeryGitCheck`: a git check is also credited PASSED.
  - `TestAudit27CommonDirAlternatesNotRefused`: `objects/info/alternates` inside the common dir is accepted (PASSED).
  - `TestAudit27CommonDirFingerprintBlind`: a create-and-delete inside the outside objects directory leaves the fingerprint equal.
  - `TestAudit27CommonDirControl`: commondir alone is accepted, not refused.
- **Preferred fix (prevent, not detect):** after verification, make every git read in Snapshot, `changedPaths` and Materialize use the verified private store. Set `GIT_DIR=<priv>`, `GIT_OBJECT_DIRECTORY=<priv>/objects`, no alternates, `GIT_INDEX_FILE=<private copy of the live index, copied once>`, `GIT_WORK_TREE=<root>`, and pass the verified head explicitly. Then nothing the live `.git` names (commondir, alternates, config includes) is read after verification.
- **Containment (small, can ship first):** in `RequireSelfContainedRepository`:
  - Refuse when `.git/commondir` exists.
  - Refuse when `git rev-parse --git-common-dir` or `--git-path objects`, resolved, differs from `<root>/.git` or `<root>/.git/objects`.
  - Run the alternates check on the resolved objects path.
- **Verification:** all `TestAudit27CommonDir*` tests must refuse credit, and the control must still credit. Rerun `TestAudit27OwnReadsKeepFingerprint` to confirm there are no false refusals.
- **Effort:** containment small; preferred fix medium.

### R2. MEDIUM. A symlink inside a real `.git` (for example `.git/objects`, `.git/index`, `.git/refs`) is accepted, and the fingerprint does not follow it (observed; end-to-end in-window swap inferred)
- **Location:** `verify.go:18` checks only `.git` itself. `fingerprint_linux.go:39-75`: `WalkDir` records a symlink's own lstat, not its target.
- **Reproduction:** `TestAudit27SymlinkedObjectsDir` moves `.git/objects` outside and links it. Result: `change seen by fingerprint=false; credit err=<nil> C1=PASSED`. A persistent forgery is still caught, because verification reads through the link. But the swap-and-restore attack from prior F1 (swap during Materialize, restore before the second fingerprint) is no longer detected for anything behind such a link. I did not run that attack end to end, because there is no hook between Snapshot and Materialize.
- **Fix:** make `Fingerprint` (or `RequireSelfContainedRepository`) refuse any symlink or non-regular, non-directory entry under `.git`, or follow the links and fingerprint their targets. The R1 preferred fix also removes the object-store part of this.
- **Effort:** small.

### R3. LOW. `VerifyRepositoryObjects` reads `.git/config` with `os.ReadFile`, outside the context (observed at function level)
- **Location:** `internal/evidence/verify.go:64`. The same Lstat-then-ReadFile pattern exists for worktree files in `materialize.go:44,73` and `raw.go:145,167`. If a regular file is swapped for a FIFO between those two calls, the read blocks with no context (inferred).
- **Reproduction:** `repro/evidence_audit_d5ar27m_test.go`, `TestAudit27VerifyFIFOConfigIgnoresContext`: with a 2 s context and a FIFO `.git/config`, the call was still blocked 8 s after the context expired.
- **Reachability in credit:** a FIFO placed before credit first stalls `VerifyBase`'s git, which ends at 10 min with a refusal. To reach line 64 blocked, the swap must happen after `rev-parse HEAD` (`credit.go:151`). That window is narrow but reachable with inotify. While blocked, credit holds both run locks indefinitely. This is denial of service, not false credit.
- **Fix:** open with `O_NONBLOCK`, `Fstat`, and require a regular file before reading. Or read the config through `git config --file` under `ctx`, which the code already does for the object-format query. Apply the same open-then-fstat pattern in `materialize.go` and `raw.go`.
- **Effort:** small.

### R4. LOW (informational). `boundedCtx` keeps a redundant timer (observed, static)
- **Location:** `cmd/hma/review.go:622-626`. `context.WithTimeout` already cancels itself at the deadline. The extra `time.AfterFunc(10*time.Minute, cancel)` and the context are held for up to 10 minutes per call, then released. This is not a permanent leak, and it is one call per credit in a short-lived CLI process. `go vet` passes.
- **Fix:** return the cancel function and `defer cancel()` at the call site, as `credit.go:139-140` does.
- **Effort:** trivial.

## 4. Regression (run on the `pristine/` copy)
- `go vet ./...`: exit 0.
- `go test -count=1 -v ./...`: 1041 PASS, 2 FAIL, 2 SKIP.
  - The 2 failures (`TestD5HMABoundLaunchRequiresEligibleTrue`, `TestD5ResumeReplayUsesSameDispatchID`) are **environmental**: `buildHMA` failed with `error obtaining VCS status: exit status 128`. The copy has no `.git`, and Go found a stray empty `/tmp/.git` as the VCS root. With `GOFLAGS=-buildvcs=false`, both PASS, with and without `-race` (`d5hook.log`, `d5hook-race.log`). This is not a product defect. I did not remove `/tmp/.git`, because it is outside scope.
  - Skipped: `TestGateBR2HostChild` (child-process helper) and `TestGateBR2HostPowerLossImage` (documented as unreachable). No D5a test was skipped.
- `go test -race ./cmd/hma ./internal/evidence ./internal/repostate`: 488 PASS, the same 2 environmental failures (both pass when rerun as above), 0 SKIP, no `DATA RACE` reports.
- Logs: `vet.log`, `test.log`, `race.log`, `credit-audit.log`, `c1.log`, `c6.log`, `e2.log`, `evidence-audit.log`.

## 5. Limitations
- The R2 end-to-end in-window swap was not executed. That conclusion rests on the observed fingerprint blindness and the prior F1 mechanics.
- `TestAudit27MidxDivergence` was skipped on purpose. I did not finish a test of whether the private `pack-objects` and live reads choose different packs when the live config sets `core.multiPackIndex=false` and the same object id is stored in two packs. The preferred fix for R1 (all reads through the private store) removes this question.
- Same vendor family; isolation is partial (see §1).

## 6. Next action
Implement R1 containment: refuse `.git/commondir`, and refuse when the resolved git common dir or objects path is not `<root>/.git` or `<root>/.git/objects`. Implement R2 containment: refuse symlinks under `.git`. Then route all of Snapshot and Materialize's git reads through the verified private store, fix R3, and request an independent re-audit with `repro/*_d5ar27m_test.go` as closure input.
