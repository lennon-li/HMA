# HMA Gate D D5a — independent audit (dispatch hma-gate-d-audit-20261021-d5ar24m)

**Verdict: FAIL** for the D5a changes in scope (2 high findings). Regression run: **PASS**.

## 1. Executive assessment

- Target: `/home/yeli/repos/HMA`, `main` at `5f6f765517987fae8a573ff8ef6a0f27c3651f29` plus the uncommitted working tree.
  `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md` were ignored.
- Freeze: the tracked diff sha256 was `f8a1546f7c9dd8481b9be0632067102b8c52bd0db15fa247c281b4b1d095b32e` at start and at end. This equals the expected value.
  The untracked-file hashes and `git status --short` were identical at start and end (`freeze-start-*.txt` and `freeze-end-*.txt`).
- PHASE1.json sha256: `6a96aa34f04ad6fed16a4a5b0812f4b708ce5b7f094c6a8b610a6be296ca0153`. It was written before I read the diff, the D5a handoff or the implementer results.
- Independence: the auditor is Claude Opus 5.5 (ming) and the implementer is Claude Sonnet 5.5. Both are from the same vendor family, so there is no provider-family separation.
  The auditor had a fresh context. The isolation level is **partial**: the OS is shared, and read-only access to `/home/yeli/repos` was kept by conduct only.
  All tests ran on copies under the output directory, with GOCACHE and TMPDIR set there.
- I did not read `docs/handoffs/2026-10-19-gate-d-d5a.md`. I did not read the /tmp/hma-gate-* or /tmp/claude-* directories, except for the one implementer results file. I read that file after PHASE1.
- Sandbox: **bubblewrap 0.9.0 started in this runtime.** I verified the sandbox behavior at runtime myself (see Q3). It is not UNVERIFIED.

Principle tested: a criterion is PASSED only when HMA re-captured a passing check on exactly the independently reviewed target (Gate A v3 §4.3; architecture §7.1 "authentic, current, revision-bound evidence"; §14).

## 2. Requirement coverage

| Question | Result | Evidence |
|---|---|---|
| Q1 private store, index and config differ from the reviewed commit without refusal? | **No bypass found** | Five probes refused or stayed correct (table below) |
| Q2 Fingerprint: other names or types hide siblings; swap-and-restore unseen? | **FAIL** | F2 (symlinked root). F1 (a .git swap-and-restore during Materialize is unseen) |
| Q3 another bind, env var or tool tree exposes live content? | **No exposure found** (runtime verified) | `TestAuditSandboxHidesLive` |
| Q4 regressions in captureOnTarget / Materialize | **FAIL** (F1: copy is not verified). No test regressions | regression run |
| Regression: `go vet ./...`, `go test ./...`, `go test -race ./cmd/hma ./internal/evidence ./internal/repostate` | **PASS** (exit 0, 0, 0; **no tests skipped**, checked with `-v`) | `vet.log`, `test.log`, `race.log` |

Q1 probes (`repro/evidence_audit_d5ar24m_test.go`, all PASS, so the code behaved correctly):
- Forged subtree under the reviewed id in the live store → refused by fsck.
- Forged loose object shadowing a packed one → the private store holds the reviewed bytes.
- Forged object served through live `objects/info/alternates` → refused.
- Live `refs/replace` and `info/grafts` → the pinned index is the reviewed tree. The private view has no replace/info/shallow/alternates.
- sha256 repository → works.

Not executed (inferred): commit-graph/MIDX lies and the race between rev-list and pack-objects. The private store has no commit-graph, and fsck checks every object hash and connectivity from the pinned HEAD. A lie therefore yields a missing or corrupt object, which means refusal.

Q2 entry types (`TestAuditFingerprintEntryTypes`, PASS): a `.git` file, a `.git` symlink to a directory, a `.git` FIFO, and a nested `.git` directory. None of them hid a sibling swap.

## 3. Findings

### F1 — HIGH — The private copy is built from unverified live `.git` state and never checked against the reviewed target

- **Certainty:** observed at the Materialize level; inferred end-to-end through `hma host credit`.
- **Location:**
  - `internal/repostate/materialize.go:22-26` (`changedPaths`, `ls-tree -r ... head`) and `internal/repostate/raw.go:48,80,89,102`, which read live objects and the live index.
  - `internal/repostate/fingerprint_linux.go:42-48`, which skips `.git`.
  - `cmd/hma/credit.go:356-378`, where Materialize output goes straight to Capture with no comparison to `before`/`rv.ArtifactDigest`.
- **Diagnosis:** `git ls-tree -r` hash-checks only the top tree object. Subtrees are read without hash verification (observed: `scripts/forge_subtree.sh` lists the forged subtree, while `fsck` reports a hash mismatch). The fsck in `buildPinnedStore` protects only the private git view. It does not protect the path list that Materialize copies, which was computed earlier from the live store.
  A writer can do the following during Materialize:
  1. Substitute a loose subtree object under its reviewed id.
  2. Optionally rewrite `.git/index`.
  3. Restore both before `pack-objects`.

  The copy then gains an ignored, unreviewed worktree file. Alternatively, with index tampering, it loses a tracked file that a committed `.gitignore` matches. The `before`/`after` Snapshots are equal, and the Fingerprint is equal because it skips `.git`. The check runs on a target the review never saw, and a pass is credited.
  The same unverified read sets the evidence record's `ChangedFiles` (`credit.go:360`, `diff base head`).
- **Reproduction:** `TestAuditForgedSubtreeInjectsUnreviewedFile` (`repro/repostate_audit_d5ar24m_test.go`). Output: `materialize err=<nil> snapshot equal=true fingerprint equal=true unreviewed file in copy=true`. It also fails under `-race`.
- **Precondition:** a committed ignore rule that matches a file the check reads (injection), or a tracked file that matches an ignore rule (omission). Both are within the stated threat model (a writer to the live repository).
- **Preferred fix:**
  1. Build and fsck the pinned store first.
  2. Compute Materialize's path list against that verified store (`GIT_OBJECT_DIRECTORY=<priv>/objects`). Use the index bytes from the same read that the `before` Snapshot digested.
  3. After copying, digest the copy (path set, bytes, file type, execute bit) with the Snapshot digest function and require equality with `before` (and so with `rv.ArtifactDigest`) before Capture.
  4. Compute `ChangedFiles` from the verified store.
- **Alternative:** extend Fingerprint to `.git` (objects, index, info, config, refs, packed-refs). This is cheaper but noisy, because git rewrites the index when it refreshes. It detects the attack but does not prevent it.
- **Verification:** the repro test must refuse (error from credit/Materialize), and the existing D5a tests must stay green. Add an omission variant: forge the subtree and the index, with a tracked file that matches `.gitignore`.
- **Effort:** medium. **Fix status:** proposed, not validated.

### F2 — HIGH — Fingerprint is blind when the project root is a symlink

- **Certainty:** observed for `Fingerprint`; inferred for credit.
- **Location:** `internal/repostate/fingerprint_linux.go:30`. Credit passes the unresolved `p.ProjectRoot` (`cmd/hma/credit.go:173,198`). `sameStore` (`cmd/hma/loop.go:308`) and `captureOnTarget` (`credit.go:334`) accept a symlinked root.
- **Diagnosis:** `filepath.WalkDir` does not follow a symlinked root. It emits one row for the link and walks nothing. A swap-and-restore of any worktree file while Materialize copies it is therefore unseen. The `after` Snapshot equals `before` once the bytes are restored.
- **Reproduction:** `TestAuditFingerprintSymlinkedRoot`. The digest is unchanged after a file is replaced with a new inode and new bytes.
- **Preferred fix:** in Fingerprint, `root = filepath.EvalSymlinks(root)` before walking. Also refuse a root whose `Lstat` is not a directory. Alternatively, credit can pass `rroot`. Do both.
- **Verification:** the repro test must report a changed digest. Add a credit-level swap-and-restore test with a symlinked ProjectRoot.
- **Effort:** small. **Fix status:** proposed.

### F3 — LOW — Git calls in preparation have no context or timeout; one inherits the host environment

- **Certainty:** hypothesis (not executed).
- **Location:** `internal/evidence/gitsafe.go:88,131,179,192,200,206`. These use `exec.Command` without ctx. Line 131 runs `git config --file` with the inherited environment.
- **Diagnosis:** a live writer can replace `objects/info/alternates` with a FIFO and block `rev-list`, so credit hangs. This is a denial of service, not a false credit.
- **Fix:** use `exec.CommandContext(ctx, ...)` with the Capture ctx, plus the fixed environment at line 131.
- **Verification:** with a FIFO as alternates, credit returns an error within the deadline.

## 4. Improvement opportunities (non-blocking)

- Coarse ctime granularity: Fingerprint relies on ctime changing. Kernel 7.0 multigrain timestamps (ext4, xfs, btrfs, tmpfs) make this sound once the file has been stat'ed. On other filesystems (NFS, FUSE), a swap-and-restore inside one tick is a theoretical gap. Consider refusing an unknown filesystem type or adding inode-generation checks.
- `Materialize`'s symlink check is lexical. A link chain such as `l2 -> .` plus `l1 -> l2/..` resolves outside the copy root. Inside the sandbox this reaches only the empty tmpfs, so it is harmless, but the comment "refused" overstates it. Resolve link targets inside the copy instead.

## 5. Remediation roadmap

1. F2 (small): resolve the root in Fingerprint and add the repro as a regression test.
2. F1 (medium): verify the copy against the reviewed digest using the fsck-verified pinned store. Add the injection and omission repro tests.
3. F3 (small): thread ctx through the preparation git calls.
4. Re-audit F1 and F2 independently, preferably with a different provider family. Exit criterion: the repro tests refuse, and the regression suite stays green under `-race`.

## 6. Limitations

- The end-to-end credit attack for F1 and F2 was not run through `hma host credit`. A deterministic run would need a hook inside Materialize, which does not exist. The step from Materialize to credit is inferred from `credit.go:356-378`.
- Not executed: commit-graph/MIDX forgery, a timed race between rev-list and pack-objects, and a FIFO denial of service.
- The auditor and implementer are from the same vendor family. The isolation is partial (shared OS, read-only by conduct).
- Implementer evidence (`host-sandbox-results.txt`, all PASS) was consistent with my regression run. It was not used as proof.

## 7. Single next action

The implementer fixes F2 and then F1, with my repro tests (`repro/*.go`) added as regression tests. Then a fresh independent re-audit of those two fixes follows.
