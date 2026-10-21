# HMA Gate D D5a: independent re-audit (dispatch hma-gate-d-audit-20261021-d5ar29m)

**Verdict: PASS for the scoped fix (C1).** The prior high finding (S1, live config vs. fixed verification config) and the medium ledger-include finding (S2) are closed: every repro is now refused with `local git configuration is not plain`, and C1 stays PENDING. No allowlisted key, casing trick, second config source in the repository, or after-check config change gave a bypass. Regression is green. One adversarial line was **not executed** (H-1 below, a hypothesis); this verdict does not cover it.

## 1. Scope, freeze, independence
- Target: `/home/yeli/repos/HMA` `main` @ `5f6f765`, plus working tree; `.serena/` and `docs/handoffs/2026-10-16-arch-amendment-*.md` excluded.
- Freeze: tracked diff sha256 `408bd8ab…89974e3` at start and end (equals expected). `git status --short` and sha256 of all 19 untracked files identical start/end (`freeze-*-start/end.txt`). HEAD unchanged. No writes or git writes under `/home/yeli/repos`.
- Since my prior round only `internal/evidence/verify.go`, `verify_test.go`, `cmd/hma/gated_credit_test.go` changed (untracked-hash comparison).
- Independence: implementer Claude Sonnet 5.5; auditor Claude Opus 5.5. **Same vendor family.** Isolation **partial**: fresh session, but I authored the prior round's findings and repros and read them. I read `verify.go`, `gitsafe.go`, `snapshot.go`, `fingerprint_linux.go`, and part of `credit.go` before writing `PHASE1.json` (sha256 in `PHASE1.sha256`). I did not read `docs/handoffs/2026-10-19-gate-d-d5a.md`.
- **Interruption:** part of this session's work was stopped by a safety classifier. I did not continue that line. H-1 is therefore unexecuted.
- Runtime: bubblewrap starts here (`bwrap --unshare-all --ro-bind / / true` OK); sandbox tests ran and passed, not skipped.

## 2. What must hold
Gate A v3 §4.3: a criterion is independently verified only if HMA re-captured its check at the bound target and it passed. Architecture §22: local OS and host are trusted; project-local content is adversarial. So every object HMA's own reads use must equal the object in the verified store, and nothing in the repository outside the fingerprint may decide what they read.

## 3. Prior findings, re-run through `hma host credit` (`prior-repros.raw`)
| Repro | Now |
|---|---|
| `TestAudit28MidxDivergenceNonGitCheck` (S1) | refused: `local git configuration is not plain`, C1=PENDING |
| `TestAudit28MidxDivergenceGitCheck` | refused (same message) |
| `TestAudit28MidxViaLedgerInclude` (S2) | refused (`include.path` not allowlisted) |
| `TestAudit28LedgerIncludeInvisible` | refused before the check runs |
| `TestAudit28MidxControl` | refused: `dirty worktree policy violation` |
| `TestAudit25CreditPersistentForgery*` | refused by fsck in private store |
| `TestAudit28NoFalseRefusalMaintained` (tags, packed refs, gc, commit-graph, midx, linked worktree, hook) | C1=PASSED |

## 4. C1 questions (analysis = inferred from code; probes = observed)
- **Allowlisted `core.*` keys.** `core.filemode/symlinks/autocrlf/ignorecase` are overridden by `observationConfig` `-c` flags (`snapshot.go:127-139`) on every `Output` call, so their live values cannot change HMA's reads. `core.logallrefupdates` affects writes only. `core.precomposeunicode` acts only on macOS. `core.bare=true` would make worktree commands fail, which refuses (fail closed). Inferred.
- **Values that matter.** `extensions.objectformat` / `core.repositoryformatversion`: verification reads the same value from the same private copy and builds a matching store (`verify.go:150-155`). A wrong format makes git fail on both sides, so it fails closed. Inferred.
- **`remote.*`, `branch.*`, `user.*`.** No effect on `rev-parse HEAD`, `diff base..head`, `ls-tree`, `ls-files`. Inferred.
- **Spelling/casing.** Names come from git's own parser, which returns canonical lowercase section/key names. `gitConfigNameAllowed` lowercases again (`gitsafe.go:228-239`). Observed: `Core.FileMode` allowed, `core.multipackindex` refused. Multi-dot subsections (`remote.a.b.url`) map to `remote.*.url`, which is the same key class, so this is harmless.
- **Second sources in the repository.** `.git/config.worktree` is read only with `extensions.worktreeConfig`, which is refused (observed). Includes are refused. The ledger can no longer feed config.
- **Host sources** (`~/.gitconfig`, XDG, system, inherited `GIT_CONFIG_*`): `Output` keeps them (`snapshot.go:46-47, 149`); verification ignores them. This is a structural gap but out of threat model (§22 trusts the host). Informational I-1.
- **Change after the check.** `requirePlainConfig` runs at `credit.go:130` (before the fingerprint) and again inside `VerifyRepositoryObjects` (`verify.go:115`, called at `credit.go:155`), after `fpStart` (`:144`). Any later write or rename of `.git/config` changes its ctime/inode, and `fpAfter` (`:221`) refuses. Confirmed by `TestAudit28LedgerIncludeInvisible` class and the existing swap-and-restore tests. Residual: `requireApprovedRepository` (`:133`) reads under live config outside the fingerprint window. Its result does not define the target (Snapshot re-reads inside the window), so impact is low. Inferred.

## 5. False refusals (observed, `allowlist.log`)
- Accepted: a real `git clone --no-local` with tags and a second branch; the same clone with `core.fileMode=false`.
- Refused: `pull.rebase=true` (observed end to end). By name check, these are also refused: `push.default`, `gc.auto`, `color.ui`, `init.defaultbranch`, `receive.*`, `remote.*.pushurl/tagopt/prune`, `branch.*.rebase`, `core.hookspath` (husky), `core.editor`, `core.excludesfile`, `core.sshcommand`, `filter.lfs.*`/`lfs.*` (`git lfs install --local`), `maintenance.*` (`git maintenance start`), `commit.gpgsign`, `user.signingkey`, `gpg.format`, `submodule.*`, `core.sparsecheckout`.
- Judgement: this is acceptable as fail-closed for v1 credit. The cost is real for husky, LFS, maintenance, and signing users. It becomes a problem only because the refusal does not name the key (L-1).

## 6. Findings
### L-1. LOW. Refusal does not name the offending key (observed)
- `verify.go:65,97-98`: one generic message for every key.
- Repro: `TestAudit29FreshCloneAccepted` → `pull.rebase=true` gives `…local git configuration is not plain`.
- Fix: return the first refused canonical key name in the error. Do not echo values, which can carry credentials in URLs. Verify: the same probe shows `pull.rebase`.

### H-1. HYPOTHESIS, unrated, NOT EXECUTED. Non-config `.git` state may still split live reads from verification
- Basis: with config pinned, git still consults default-on auxiliary data in the live `.git` (e.g. commit-graph) during HMA's live reads. `buildPinnedStore` reads the live store as an alternate and fscks only the private copy (`gitsafe.go:182-224`). I did not determine whether any such auxiliary file can make the two disagree. That probe line was interrupted (§1).
- Next check: an independent auditor should probe this with fixtures, or the implementer should take the structural fix below, which removes the question.

### I-1. INFO. Host-level git config reaches live reads only (inferred)
Out of threat model; listed for completeness. The structural fix removes it.

## 7. Structural option: is it still needed?
The allowlist closes the **config** sub-class: no repository-local key outside the list survives, and no listed key changes object selection. It does **not** close the general class "HMA's live reads consult state the verifier did not consult". Host config (I-1) and possibly auxiliary object data (H-1) still differ between the two. **Recommendation:** adopt the single verified store as durable hardening. Build it once per credit; run `rev-parse HEAD`, Snapshot, changedPaths, Materialize, and VerifyBase against it with `GIT_DIR`/`GIT_OBJECT_DIRECTORY` private, no alternates, fixed config; ensure `base` objects are included. This also lets you relax the allowlist for HMA's own reads, cutting the §5 false refusals. It is not required to close the S1/S2 findings, which are closed.

## 8. Regression (copy under `work/`, `GOFLAGS=-buildvcs=false`, GOCACHE/TMPDIR under output dir)
- `go vet ./...`: exit 0.
- `go test -count=1 ./...`: 1061 PASS, 0 FAIL, 2 SKIP (`internal/host.TestGateBR2HostChild`, `TestGateBR2HostPowerLossImage`). Counts include my repro tests.
- `go test -race -count=1 ./cmd/hma ./internal/evidence ./internal/repostate`: 508 PASS, 0 FAIL, 0 SKIP, no `DATA RACE`.
- No evidence/repostate/D5a test skipped. Sandbox tests passed under real bwrap.

## 9. Limitations
Same vendor family; partial isolation; H-1 unexecuted; git 2.43.0 only; §4 items marked inferred were not run end to end.

## 10. Next action
Run an independent fixture probe for H-1 (auxiliary `.git` data such as commit-graph vs. the verified store), or implement the single verified store (§7). Make L-1 a one-line error-message change in the same pass.
