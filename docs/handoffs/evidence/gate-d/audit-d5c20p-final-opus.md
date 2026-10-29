# D5c final audit — PASS

**Dispatch:** `hma-gate-d-d5c20p-final-audit-20261029-ming`
**Date:** 2026-10-29 (report written 2026-10-29T17:26Z)
**Target:** `main` at `b848a39ade9aa55b8ace6cf5b531bad83289e657`
**Parent (diff base):** `e65ce7272be758dafa4db523582e144332ad1e29`
**Scope:** the one commit, 7 files, +574 / −11. Unit diff SHA-256 (`git diff e65ce72 b848a39`): `85d242e4a44ec226bdc68c4e3624740c7e1a4b341a7988de642847b72c5bc833`.
**Mode:** code review. Read, execute and test only. No commit, push or history change.

## Human review layer

Machine verdict: **PASS**.

- All four findings from the prior audit (D5C20P-F01..F04) are fixed at `b848a39`.
- I reproduced F01, F02 and F03 on the parent `e65ce72` with my own probes, independent of the retained tests. The same probes pass on the target.
- I built three single-hunk mutants, each reverting one fix. Every mutant is caught by both the retained tests and my probes.
- `internal/repostate/raw.go` is byte-identical to the parent.
- This commit introduces no new defect of the same shape.
- vet, the full test suite, the race suite, gofmt and the darwin/windows cross-builds all pass.
- There are two ADVISORY notes and no BLOCK or WAIVABLE findings.

| Finding | Severity | Observed result | Disposition |
|---|---|---|---|
| D5C20P-F01 (prior) | High | Fixed. A case-distinct untracked file is captured under inherited `core.ignoreCase=true` from four config sources. | Resolved |
| D5C20P-F02 (prior) | High | Fixed. The clean manifest and the clean budget measurement refuse a transient HEAD swap with `HOST_REVIEW_TARGET_CHANGED`. | Resolved |
| D5C20P-F03 (prior) | Medium | Fixed. The `staged` flag follows the captured index when the index is unstaged or restaged after capture, in both directions. | Resolved |
| D5C20P-F04 (prior) | Medium | Fixed. The handoff now states that non-Linux refusal covers clean and dirty targets. See A01 for one imprecision. | Resolved |
| D5C20PF-A01 | Low | Handoff names the error `dirty target observation requires Linux`. A clean-bound live-target check on non-Linux surfaces `HOST_REVIEW_TARGET_CHANGED` instead. The refusal claim is correct; only the named reason is imprecise. | ADVISORY |
| D5C20PF-A02 | Info | Pre-existing, outside this commit: `ExpectedLaunchWorktree` comes from a second live read (`LaunchDigests`) after the capture. It fails closed at receipt comparison. | ADVISORY |

### Auditor identity and independence (disclosure)

- **Agent:** Ming (role: senior consultant).
- **Runtime/access service:** Claude Code, built on the Claude Agent SDK.
- **Provider:** Anthropic.
- **Model:** Claude Opus 5.5 (`claude-opus-5-5`). This is self-reported from the runtime system prompt. I have no independent model telemetry.
- **Separation from the implementer:** genuine. The implementer liz runs on OpenAI `gpt-6.1-terra`. I am Anthropic Claude, a different provider and model family. I had fresh context and no memory of how the commit was produced.
- **Not separated from the orchestrator:** the commit trailer reads `Co-Authored-By: Claude Sonnet 5`, and the handoff names a Claude orchestrator. Both are Anthropic, the same provider family as this auditor. §13 requires a different provider/model family for final validation. Read against the implementer, that requirement is met. Whether orchestrator/commit-author family overlap disqualifies this report is Lennon's decision.
- **Blinding:** partial. As the dispatch directed, I read the commit message and the prior audit report (`/tmp/hma-gate-d-20261029-d5c20p/AUDIT.md` and `AUDIT.json`). I wrote all probes and mutants myself. I did not use the pre-existing artifacts (see below).

## Answers to the dispatch questions

### Q1. Does `check-ignore` use the capture's `core.ignoreCase` pin? (F01)

Yes. `internal/repostate/paths.go:307-309` now builds the argument vector as follows:

1. It starts from `append([]string{}, observationConfig...)`. That is the same slice `Output` uses (`internal/repostate/snapshot.go:128-140,148`). It includes `-c core.ignoreCase=false` at `snapshot.go:137`.
2. It appends `-c core.excludesFile=/dev/null`, an explicit `--git-dir=<tmp>/.git --work-tree=<tmp>`, and `check-ignore --no-index -z --stdin`.

The copy (`append([]string{}, …)`) does not alias the package-level slice.

Evidence:

- **Probe `TestMingF01IgnoreCaseInherited`:** `.gitignore` = `SECRET` and `exact.txt`. Untracked files `secret` (150 lines) and `exact.txt`. Inherited `ignoreCase=true` was set four ways: global file, `GIT_CONFIG_COUNT`, `GIT_CONFIG_PARAMETERS`, and repo config.
  - Target: `secret` is in `capture.Changed`. The manifest has `secret` with `Added:150`. `exact.txt` is still excluded, which shows ignore evaluation is case-pinned rather than disabled. All four subtests pass.
  - Parent: `capture.Changed=[]` for global file, `GIT_CONFIG_COUNT` and `GIT_CONFIG_PARAMETERS`, so F01 reproduces. The repo-config case passed even on the parent because the scratch repository does not read the target's config.
- **Retained tests** `TestAudit20IgnoreCaseParity`, `TestAudit20IgnoreCasePacketAndFindings` and `TestAudit20IgnoreSemanticsParity` (6 ignore-semantics cases) pass on the target. The first two fail on the parent.
- **Mutant M2** (`args := []string{}`, i.e. the pin removed) fails 3 retained tests and 3 probe subtests.

### Q2. Does the `staged` flag come from the captured index bytes? (F03)

Yes. These lines carry the fix:

- `repostate.Capture` gains `Index string` (`snapshot.go:309`).
- `capture()` stores the one `ls-files --stage -z --` read (`snapshot.go:357,372`). That same read also produces `State.Staged` and the per-path staged entries in the worktree digest.
- `observeCaptured` uses `capture.Index` whenever a capture is supplied (`cmd/hma/observe.go:133-141`). It falls back to a fresh `ls-files -s -z` only when there is no capture.
- `-s` and `--stage` produce the same format, and `parseIndex` keeps stage-0 records. So the captured and fresh forms parse identically.

Production callers always supply a capture:

- review packet: `review.go:403`, which takes `&captured`
- findings import: `review.go:642,652`, which takes `live` from `captureLiveTarget`, non-nil on success

`measureChange`, `buildChangedFileManifest` and `observeTarget(Limit)` (the nil-capture wrappers) have no non-test callers.

Evidence:

- **Probe `TestMingF03StagedFromCapturedIndex`:** I captured, then flipped the index (unstage, or stage), then observed with the original capture.
  - Target: the manifest `staged` equals the captured value in both directions.
  - Parent: it reports the later index in both directions (`staged=false`, captured `true`; and the reverse).
  - A re-capture detects the index change in both directions (`digestChanged=true`). So the packet path's post-manifest capture (`review.go:408-414`) is a second barrier.
- **Retained `TestAudit20IndexMetadataSwapRestore`** (unstage before the manifest, restore during the worktree walk) passes on the target and fails on the parent.
- **Mutant M3** (always read a fresh index) fails this retained test and both probe subtests.

### Q3. Does the corrected handoff text match the code? (F04)

Mostly yes. I checked each claim against the code:

- "Captured HEAD, index, and worktree reads bind their results to the supplied capture" (handoff line ~41-42) is **true**. HEAD equality is checked at `observe.go:122-124`. The index comes from the capture at `observe.go:134-135`. Each worktree read is checked against `capture.Entries` by `checkObserved` (`observe.go:219`).
- "Other platforms refuse live-target capture … for clean and dirty targets" is **true** for `captureTarget`. It returns `errDirtyObservationUnsupported` before `CaptureTarget` regardless of `dirty` (`observe.go:74-79`; error text at `observe_other.go:14`).
- "Review-packet generation uses that capture for both contract-bound and unbound runs" is **true**. `captureTarget` runs at `review.go:348`, before the `state.Contract != nil` branch at `review.go:399`. Retained `TestAudit20FollowupCleanUnsupported` and my probe confirm a clean packet is refused with the platform error.
- "Live-target checks during review and findings import also use it when a packet binding exists. These paths refuse on macOS and Windows" is **true**: `requireLiveTarget` and `captureLiveTarget` (`review.go:530,642,659,734-760`) call `captureTarget`. One imprecision: for a clean binding (`dirty=false`), `review.go:753-757` maps the platform error to `HOST_REVIEW_TARGET_CHANGED`. My probe observed `liveClean=HOST_REVIEW_TARGET_CHANGED` and `liveDirty=<platform sentinel>`. See A01.
- "Direct committed-object observation does not require the anchored worktree reader" is **true**. The clean branch of `observeCaptured` reads only git objects, and the probe confirms it with the platform guard set. With the current call graph this path is reached only after a successful capture, so on non-Linux it is not reachable from a CLI command. The sentence does not claim otherwise.
- I did not re-verify "Credit and the sandbox are Linux-only already". That text is unchanged from the parent.

### Q4. Is the clean target bound to the captured HEAD? (F02)

Yes. The HEAD check moved out of the dirty branch. It now runs unconditionally whenever a capture is supplied: right after `rev-parse HEAD` and before any tree, blob, index or worktree read (`observe.go:117-124`).

After the check, every read is by immutable object id:

- `listTree(head)` with the verified commit id
- `listTree(base)` with the verified full base id
- `cat-file blob <oid>`

So a HEAD swap after the check cannot redirect any read.

Evidence:

- **`TestMingF02CleanHeadSwapBeforeObserve`:** a `git` wrapper on PATH swaps HEAD just before `observeCaptured`'s `rev-parse HEAD` and restores it at the first `ls-tree`. The target refuses for both the manifest and the measurement. On the parent it was accepted with the other revision's bytes (`Added:200`, `lines=201` against the approved 3-line change).
- **`TestMingF02CleanHeadMovedNoRestore`:** target refuses. Parent accepted.
- **`TestMingF02CleanHeadSwapAfterCheckStaysBound`:** the swap happens after the check and is restored at the first `cat-file`. The manifest is the approved HEAD's (`SHA256 = sha256(approved body)`, `Added:3`), so the binding holds.
- **`TestMingF02DirtyHeadSwapBeforeObserve`:** the dirty path still refuses. This guards against a regression from moving the check.
- **Retained tests** `TestAudit20CleanCaptureHeadBinding`, `TestAudit20CleanPacketHeadSwapRestore` and `TestAudit20FollowupCleanFindingsSwapRestore/swap_restore` pass on the target and fail on the parent. On the parent, the findings import produced `findings=map[]` and dropped the 150-line budget excess.
- **Mutant M1** (check removed) fails 3 retained tests and 4 probe tests, including the dirty-path guard.

### Q5. Regressions and scope creep

- **`raw.go` untouched:**
  - `git diff --quiet e65ce72 b848a39 -- internal/repostate/raw.go` exits 0.
  - The blob id is identical: `00c5a701e17971f6d1ee3ab2e84b835b3e305a3c` at both commits.
  - File SHA-256 is `03a02c9adffab8f04a0fd9f1d4016c62a787f9b28395b0fe7fa9b354e5a702cc`, the same value the prior audit recorded for the D5b target.
- **Scope:** all four non-test hunks map to F01–F04. One addition goes beyond the finding: explicit `--git-dir`/`--work-tree` on the scratch `check-ignore`. It is hardening inside the same invocation, already implied by `cmd.Dir = tmp` plus `gitEnv` stripping `GIT_DIR`/`GIT_WORK_TREE`. It does not change behaviour on the probes (6 ignore-semantics parity cases pass). The three new test files are regression tests only.
- **New same-shape defects introduced:** none found. I traced every fact `observeCaptured` puts into the manifest or budget when a capture is supplied:

  | Fact | Source |
  |---|---|
  | head | live, then checked equal to `capture.State.Head` |
  | base and head trees | immutable ids |
  | index | the capture |
  | changed set | the capture |
  | worktree bytes | live, then checked equal to `capture.Entries` |
  | blob bytes | immutable ids |

  None comes from an unchecked second live read. Reusing `capture.Index` is consistent with `capture.State.Head`, because both come from the same `capture()` call that the post-manifest re-capture vouches for.
- **Pre-existing, outside this commit:** `ExpectedLaunchWorktree` (`review.go:358-364`) comes from a separate live `LaunchDigests` read, and only its head is compared to the capture. See A02.

## Findings

### D5C20PF-A01 — ADVISORY (low)

- **Criterion:** §16 "contradictions between claims and evidence" (documentation precision).
- **Evidence:** handoff line ~111, `review.go:751-757`, and probe `TestMingF04UnsupportedPlatformBehaviour` (`liveClean=HOST_REVIEW_TARGET_CHANGED`).
- **Observation:** the handoff names `dirty target observation requires Linux` as the refusal for live-target capture "for clean and dirty targets". For a clean packet binding, `captureLiveTarget` reports `HOST_REVIEW_TARGET_CHANGED` instead.
- **Impact:** the refusal is correct (fail-closed). Only the reported reason can mislead an operator on macOS or Windows. The code involved predates this commit.
- **Suggested wording:** "(…; a clean-bound live-target check reports `HOST_REVIEW_TARGET_CHANGED`)". Not required for acceptance.

### D5C20PF-A02 — ADVISORY (info, pre-existing, outside this commit)

- **Criterion:** the dispatch's "same-shape" sweep.
- **Evidence:** `review.go:358-364,375`.
- **Observation:** `ExpectedLaunchWorktree` is bound into the packet from a second live read after the capture. Only `head` is cross-checked. Unbound runs have no post-manifest re-capture.
- **Impact:** it is a receipt comparison value, not part of the artifact digest. A transient state recorded there causes a later launch-receipt mismatch (refusal), not acceptance of a different target.
- **Why ADVISORY:** this commit did not introduce or change it, and §16 forbids demanding unrelated cleanup. Recorded for the gate owner.

## Commands and results

Authoritative gates ran in `/home/yeli/repos/HMA`:

- HEAD was `b848a39`, and `git status --porcelain` was empty before and after.
- Toolchain: `go1.27.3 linux/amd64` from `~/.local/go/bin`. The `/usr/local/go` install is go1.11, too old for this module, so I did not use it.
- git 2.34.1, Linux 7.1.1.

| Command | Result | Log |
|---|---|---|
| `go vet ./...` | exit 0 | `ming/logs/final-vet.log` |
| `go test -count=1 ./...` | exit 0. 12/12 packages ok (cmd/hma 111.7s) | `ming/logs/final-test.log` |
| `go test -race -count=1 ./cmd/hma ./internal/evidence ./internal/repostate ./internal/host ./internal/engine` | exit 0. 5/5 ok (cmd/hma 136.7s) | `ming/logs/final-race.log` |
| `gofmt -l` on the 6 changed `.go` files | no output, exit 0 | `ming/logs/final-gofmt.log` |
| `GOOS=darwin go build ./...`; `GOOS=windows go build ./...`; `GOOS=darwin go vet ./cmd/hma` | all ok | `ming/logs/final-cross.log` |
| Probes + retained `TestAudit20*` on target (`go test -count=1 -v -run 'TestMing\|TestAudit20' ./cmd/hma`) | exit 0. All pass | `ming/logs/probes-target.log` |
| Same on parent `e65ce72` (retained tests copied in; the one `c.Index` assertion removed so it compiles) | exit 1. F01, F02 and F03 reproduce; F04 control and the stable controls pass | `ming/logs/probes-parent.log` |
| Mutants M1 (HEAD check removed), M2 (ignoreCase pin removed), M3 (fresh index) | each exit 1. Caught by retained tests and probes | `ming/logs/mutant-m{1,2,3}.log` |

Disclosed anomaly from an earlier, superseded run:

- I first ran the full suite in an isolated clone under `/tmp` (`ming/logs/test.log`). It failed once in `scripts/dogfood/hostinput` (`TestSyntheticStoreRefusesNonScratchDir`).
- That test asserts that the cwd (the repository) is *not* under `os.TempDir()`. A clone under `/tmp` breaks that assumption, so this is an environment artifact.
- The same package passes in the project tree.
- That clone also picked up my probe file partway through. Its results (`ming/logs/{vet,test,race,gofmt}.log`) are superseded by the `final-*` logs above.

## Limitations and process disclosures

- The dispatch text reached me in compressed form. A retrieval of the full text was denied by tool permissions, so I worked from the compressed text. All named questions were legible and are answered above.
- The output directory already held partial artifacts from another session (`gocache/`, `logs/`, `probes/`, `repo/`, `repo-probe/`, `tmp/`, timestamped 10:54–11:00). I did not use, modify or delete them. My work is under `ming/`.
- My probe file remains as an untracked file in the scratch clone `ming/target/`, because deletion was denied. It is not in the project repository.
- I did not run Jev, the implementer's `host-dogfood.sh`, or a macOS/Windows runtime. Non-Linux behaviour was exercised on Linux by setting `errDirtyObservationUnsupported` to a sentinel, the same technique the retained tests use.
- Model identity is self-reported (see the disclosure above).

## Artifacts

- `ming/probes/ming_probe_test.go`: SHA-256 `e2f4dc3bd44727bfb013946892b6b8a1d67322f216d1c1ef91b32e01582cb448`
- `ming/logs/final-test.log`: `f1285a5d22b7794c34cdf83422971d916949646ae3523e70a79ee00452a8ab37`
- `ming/logs/final-race.log`: `353210ee950dbc9f41d86fe65e51776dbb50d633dcae92902e456e43d9daf7c9`
- `ming/logs/probes-target.log`: `d21433fb1cfa8bc18bc12449d15097aff54a9083235bbdbbb02fae8791d0cf96`
- `ming/logs/probes-parent.log`: `38719d6dcd1b5ba77d9956b67ed22bee18fd4b0995471703ca14eb8c4c996142`
- Machine-readable report: [AUDIT.json](AUDIT.json)

## Next action (§18)

**Exactly one recommended next action:** `SUBMIT_D5C_FOR_HUMAN_ACCEPTANCE`. Lennon decides whether to accept D5c at `b848a39`, based on this PASS.

- **Current outcome:** PASS. F01–F04 are resolved. F01–F03 were reproduced on the parent and shown fixed on the target. All three fixes are mutant-proven. Gates are green. `raw.go` is unchanged. There are two ADVISORY notes and no BLOCK.
- **Rationale:** the code evidence supports acceptance. The one open question is independence, and that is a policy judgment rather than a code defect. This auditor (Anthropic) is a different provider/model family from the implementer (OpenAI). It is the same family as the orchestrator and commit co-author (Claude).
- **Proposed owner:** Lennon.
- **Agent/model route:** none. This is a human decision; no execution is required.
- **Permission envelope:** read this report and decide. Nothing is authorized by this report: no commit, push, branch, release, stage transition or waiver.
- **Exact approval required:** Lennon's explicit statement accepting D5c at commit `b848a39ade9aa55b8ace6cf5b531bad83289e657`. The statement should also say whether an Anthropic-family auditor satisfies §13 final-validation independence, given the Claude orchestrator. If it does not, the result is `BLOCKED` pending an auditor from a third provider family, per §13.
- **Stop and escalation conditions:**
  - Stop and escalate if Lennon rules the orchestrator-family overlap disqualifying (then `BLOCKED`, not FAIL).
  - Stop if any later audit reproduces a defect in the F01–F03 classes at `b848a39`.
  - Stop if `main` moves past `b848a39` before acceptance; re-audit the delta.
  - A01 and A02 do not block acceptance. Act on them only by separate instruction.
