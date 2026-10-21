# HMA Gate D D5a — independent re-audit (dispatch hma-gate-d-audit-20261021-d5ar25m)

**Verdict: FAIL.** F2 is closed. F3 is closed for preparation, with a low residual. **F1 is not closed.** The fingerprint now catches an in-window swap in `.git`, which was the prior repro. But a forgery that is never swapped inside the window still gets a non-git check credited PASSED on a target that differs from the reviewed commit. I reproduced this end to end through `hma host credit`. The regression run passed (vet, test and -race all exit 0).

## 1. Scope, freeze, independence

- Target: `/home/yeli/repos/HMA`, `main` at `5f6f765517987fae8a573ff8ef6a0f27c3651f29` plus the working tree. I ignored `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md`.
- Freeze at start: the tracked diff sha256 was `765068ef1c7f4ba0a145267914dcdadc06a6ad750e3fbb9d2bf3190aced69d55`, which equals the expected value. The untracked file hashes are in `freeze-start-untracked.sha`. The end check is in §7.
- `PHASE1.json` (sha256 `d034d869…c0c04`) was written before I read the diff. It lists hypotheses G1–G10. I did not read `docs/handoffs/2026-10-19-gate-d-d5a.md` or any implementer results. I did read the implementer's tests in `fingerprint_test.go`, as artifacts.
- Independence: the auditor is Claude Opus 5.5 and the implementer is Claude Sonnet 5.5. They are the **same vendor family**, so there is no provider separation. My context was fresh except for my own prior report and repros, which were the permitted closure input. Isolation is **partial**: the OS is shared, and read-only access to `/home/yeli/repos` was kept by conduct. All tests ran on copies under this directory, with GOCACHE and TMPDIR set here.
- Sandbox: **bubblewrap 0.9.0 starts in this runtime.** Every credit repro below ran the check inside the real sandbox.

Principle (Gate A v3 §4.3; architecture §7.1, §14): a criterion is PASSED only when HMA re-captured a passing check on exactly the independently reviewed target.

## 2. Closure of prior findings

| Prior | Status | Evidence |
|---|---|---|
| F2 symlinked root | **Closed** | `fingerprint_linux.go:31-38` resolves the root with EvalSymlinks and refuses a non-directory. My `TestAuditFingerprintSymlinkedRoot` PASS. The implementer's `TestFingerprintResolvesSymlinkedRoot` PASS. |
| F1 swap-and-restore in `.git` during Materialize | **Closed for that repro; not closed as a finding** | Re-run of `TestAuditForgedSubtreeInjectsUnreviewedFile`: `fingerprint equal=false`, so credit would now refuse. Re-run of `TestAuditExcludeToggleHidesReviewedFile`: `fingerprint equal=false`. The file was copied either way, because `ls-files --others` does not read `info/exclude`. New bypasses are listed as N1–N3 below. |
| F3 preparation without a deadline | **Closed for preparation**, residual N4 | `TestAudit25PrepareFIFOAlternatesEnds`: PrepareGitExecution returned after 2m0s with `pinned git view refused`. `gitsafe.go:58,94-95,138-139` |

Side changes:
- (a) `GIT_OPTIONAL_LOCKS=0` (`snapshot.go:120`) only stops opportunistic index refresh. Snapshot hashes raw bytes (`raw.go:143-174`) and does not trust the stat cache, so the snapshot is not weakened. `TestAudit25NoFalseRefusalOwnReads` PASS: the fingerprint is stable across repeated Snapshot and Materialize runs.
- (b) `buildPinnedStore` (`gitsafe.go:189`) uses the live store only as an alternate for `rev-list` and `pack-objects`. `fsck` and `read-tree` run with `env[:5]` plus the private GIT_DIR and object directory, so no alternate is involved (`:213,219`). The private store has no `info/alternates`, refs, grafts or shallow file. All prior pinned probes still PASS: forged subtree, forged loose-over-pack, forged object via alternates, replace refs and grafts, and sha256. `TestAudit25CreditPersistentForgeryGitCheck` was refused by fsck. **Neither side change weakens the pinned view or the snapshot.**

## 3. Findings

### N1 — HIGH — a forgery planted before the first fingerprint gets a non-git check credited on an unreviewed target (observed, end to end)
- **Location:** `cmd/hma/credit.go:134,173,180,198`; `internal/repostate/materialize.go:22-26` (reads live, unverified objects and index); `internal/evidence/capture.go:156-186` (the pinned store and its fsck exist only when the check is git).
- **Diagnosis:** the fix detects *changes* inside the window but does not verify *content*. The attack has four steps, all done after the review is imported and before credit, with nothing restored:
  1. Write an ignored file `s/x`.
  2. Replace the loose object for `HEAD:s` with a tree that also lists `x`.
  3. Add `s/x` to the index.
  4. `s` is the same object at base and head, so `git diff base head` never reads it.

  The before Snapshot then equals the reviewed artifact digest (`reviewed==before=true`). The fingerprint is equal before and after, because nothing changed in the window. Materialize copies `s/x`. A non-git check never builds or fscks a pinned store. The same applies to a change made between the Snapshot (`:134`) and the first fingerprint (`:173`), and between criteria.
- **Reproduction:** `repro/credit_audit_d5ar25m_test.go`:
  - `TestAudit25CreditPersistentForgeryNonGitCheck` → `credit err=<nil> … C1=PASSED`. `requireIndependentCredit` (the outcome gate) also accepts it.
  - Control `TestAudit25CreditControlNoForgery` → `C1=FAILED`.
  - Materialize-level version: `TestAudit25PersistentForgeryBeforeFirstFingerprint`.
- **Preferred fix:** prevent; do not detect.
  1. For every sandboxed credit, git or not, build the fsck-verified private store **before** the before Snapshot. It must contain the base and head closure.
  2. Copy the live index once into a private file.
  3. Run Snapshot, `changedPaths` and Materialize's `ls-tree` and `diff` with `GIT_OBJECT_DIRECTORY=<priv>/objects`, no alternates, and `GIT_INDEX_FILE=<private index copy>`.
  4. Require that Snapshot to equal `rv.ArtifactDigest`.
  5. After copying, digest the copy in the Snapshot domain and require equality.

  With a verified tree, a forged index entry shows up as a staged change, so a clean credit is refused.
- **Alternative:** always build the pinned store and add `git fsck --connectivity-only` plus full object verification of the live head closure at the start of credit. This is weaker, because it still leaves a time-of-check to time-of-use (TOCTOU) gap before Materialize.
- **Verification:** the N1 test must refuse or give FAILED. The control must stay FAILED. `TestAudit25CreditNoFalseRefusalGitCheck` and the existing suite must stay green under `-race`.
- **Effort:** medium. **Fix status:** proposed.

### N2 — HIGH — the fingerprint does not cover object stores outside `.git`, or the exempt ledger (observed at Materialize level, end-to-end inferred)
- **Location:** `fingerprint_linux.go:56-58` (the `.git/agent-dispatch` SkipDir). The walk never follows `objects/info/alternates`.
- **Diagnosis:** suppose the repository already uses an alternate object directory, through a `clone --shared` or `--reference` layout, or one an attacker set up earlier. The `alternates` file is static, so the fingerprint is stable. A swap of the subtree object in that directory during Materialize, then a restore, is unseen. The restore happens before `buildPinnedStore`, so fsck passes even for a git check. An alternate under `.git/agent-dispatch` is unseen in the same way. A writer to the ledger can therefore hide object state.
- **Reproduction:** `TestAudit25SwapInExternalAlternate` and `TestAudit25SwapInAgentDispatchAlternate` both give `snapshot equal=true fingerprint equal=true fsck after restore=true unreviewed file in copy=true`.
- **Fix:** the N1 fix removes this class, because content is verified and no live store is read after verification. Short-term containment: refuse credit when `objects/info/alternates` exists, recursively. Refuse `.git/agent-dispatch` unless it is a plain directory that no alternates file or include points into.
- **Verification:** both tests refuse. **Effort:** small for containment; covered by N1 otherwise.

### N3 — MEDIUM — a symlinked `.git`, or a gitfile from a linked worktree, makes the fingerprint blind to the real git dir; non-git credit accepts it (observed)
- **Location:** `fingerprint_linux.go:40` (WalkDir does not follow `.git`). Only `PrepareGitExecution` (`gitsafe.go:61-63`) refuses a non-directory `.git`, and that runs only for git checks.
- **Reproduction:** `TestAudit25CreditSymlinkedGitDirAccepted` → `index swap seen by fingerprint=false; credit err=<nil> C1=PASSED`. With such a layout, the prior in-window F1 attack is open again for non-git checks (inferred).
- **Fix:** at the start of credit, refuse a ProjectRoot whose `.git` fails `Lstat().IsDir()`, the same rule as PrepareGitExecution. Alternatively, resolve the git dir and common dir and fingerprint them too.
- **Verification:** the test refuses. **Effort:** small.

### N4 — LOW — credit can still hang on a FIFO alternates file, before preparation (observed)
- **Location:** `cmd/hma/credit.go:133` (`context.Background()`); `repostate.Output` (`snapshot.go:144-155`). Credit holds both run locks while it waits.
- **Reproduction:** `TestAudit25CreditFIFOAlternatesHang`. Credit was still blocked after 30 s. It completed once the FIFO was opened for writing. A manual check gave `git ls-tree -r HEAD` exit 124 after 5 s.
- **Fix:** give credit a bounded context, for example `context.WithTimeout` covering Snapshot, Fingerprint and Materialize, and set `cmd.WaitDelay`. Also refuse a non-regular `objects/info/alternates`. This is a denial of service, not false credit.
- **Effort:** small.

### Negative results (no bypass found)
- `TestAudit25LedgerIncludeToggle`: `.git/config` included a file under `.git/agent-dispatch` that toggled `core.worktree`. Result: 0 hits in 200 tries. With repository discovery, git ignored the included `core.worktree`, which I checked by hand.
- The pinned view is unaffected by replace refs, grafts or alternates. See the side changes in §2.
- No false refusal from HMA's own reads. A concurrent user `git status` (optional locks on) does change the fingerprint, which leads to a fail-closed refusal. That is acceptable but should be documented for operators.

## 4. Regression
- `go vet ./...`: exit 0.
- `go test -count=1 -v ./...`: exit 0. There were 1039 PASS and 0 FAIL. **2 skipped**, both in `internal/host/gateb_r2_test.go`:
  - `TestGateBR2HostChild`: a child-process helper, "child only".
  - `TestGateBR2HostPowerLossImage`: documented as unreachable after the fix.

  No D5a test skipped. The sandbox tests ran.
- `go test -race ./cmd/hma ./internal/evidence ./internal/repostate`: exit 0 (no data race reported, no skips).
- The suite ran on a clean copy (`pristine/`) with the same diff hash. The probes ran on a separate copy (`work/`).

## 5. Limitations
- N2 end to end through `hma host credit` was not run, because there is no hook inside Materialize. The step from Materialize to credit is inferred from `credit.go:173-198` and `captureOnTarget`.
- Ctime granularity on NFS or FUSE filesystems was not tested.
- The auditor and implementer are from the same vendor family, and isolation is partial.

## 6. Single next action
The implementer fixes N1, which subsumes N2: build the fsck-verified private store and a private index copy for every credit, and run Snapshot and Materialize against them. Then add N3 and N4 containment, and add `repro/*_d5ar25m_test.go` as regression tests. After that, a fresh independent re-audit, preferably by a different provider family.

## 7. End freeze
HEAD unchanged (5f6f765). `git status --short`, the tracked diff sha256 (`765068ef…69d55`) and all untracked-file hashes were identical at start and end (`freeze-end-*`). No file under /home/yeli/repos was written.
