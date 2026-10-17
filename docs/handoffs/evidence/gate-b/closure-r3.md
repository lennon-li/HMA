# Gate B closure audit

**Verdict: FAIL. Gate B is not closed.**

The repository tests, vet, and race tests passed in a source copy. The synthetic dogfood passed 17 checks. Two high-severity safety defects remain. A concurrent `BLOCK` review can be bypassed by a transition. A symlink store path can lose an acknowledged host record under the directory-entry crash model.

**Closure coverage:** 41 IDs: 35 FIXED, 5 PARTIAL, 1 OPEN. The graph-related criteria remain incomplete. This is a Gate B assessment. It is not release approval or host-MVP certification.

**Isolation: compromised.** This was a fresh auditor conversation without implementer chat. I read the permitted Gate A report too broadly. I also explicitly extracted the F/C criteria after the first diff because an earlier display was truncated. I did not read a prohibited prior Gate B review or TODO before freezing Phase 1. These protocol errors prevent an isolated blind-certification claim. The failures below have separate executable evidence.

## Scope and freeze

- Dispatch: `hma-gate-b-closure-blind-20261016-r3`.
- Declared reviewer: Liz / OpenAI `gpt-6.1-sol` / high. Runtime identity was not independently attested. No worker was dispatched.
- Target: `main`, HEAD `67965541b7032a585a68e48f3d10f4a998d5399e`, plus the uncommitted tree.
- Initial binary diff SHA-256: `9ab726fb8ef8a63833ba9410898877a4489bc029aa2256382a9605794291cb5b`.
- Inventory: 120 tracked files and 24 nonignored untracked files, with SHA-256 and modes. `.serena/` and the two `2026-10-16-arch-amendment-*.md` drafts were excluded from substantive review.
- Threat model: trusted host. Accidents, concurrency and drift are in scope. A hostile same-UID writer is out of scope.
- Repository access was read-only by procedure. It was not an OS-enforced read-only mount. All builds, caches, fixtures and probes were under this output directory.
- Objective: [consolidation handoff §5](/home/yeli/repos/HMA/docs/handoffs/2026-10-15-outcome-consolidation-opus-liz.md:82), the specified defect sources, and architecture §§5–7, 10, 12–16, 18–22.
- Phase 1: `2026-10-16T19:42:16.150947+00:00`, SHA-256 `b52def8102b10892d17e77d80867b0a34307b2a4cb095b7107e66286cc4cbbad`. [PHASE1.json](PHASE1.json) was not overwritten.
- End check: [target-verification.json](target-verification.json) confirms unchanged HEAD, branch, status, binary diff, and every inventoried tracked/untracked file and mode. Git-internal optional stat-cache refresh was not established as immutable.

The full report contract, original closure-check text, source hashes, evidence ledger, fix proposals and limitations are in [AUDIT.json](AUDIT.json). Start/end manifests and raw logs remain beside this report.

## Verification

| Check | Observation | Raw evidence |
|---|---|---|
| `go test -count=1 ./...` | PASS, 12 packages | [test.log](test.log) |
| `go vet ./...` | PASS | [vet.log](vet.log) |
| `go test -race -count=1 ./...` | PASS, 12 packages | [race.log](race.log) |
| Dogfood harness | First build failed VCS stamping; unchanged rerun with `GOFLAGS=-buildvcs=false` passed 17 checks | [first](dogfood.log), [rerun](dogfood-rerun.log) |
| Artifact/CI/legacy identity probes | Expected supported controls pass; unsafe states refuse | [command probes](independent-command-probes.log), [expanded probes](independent-expanded-probes.log) |
| Raw conversion/index/root probes | Corrected probes pass; initial object-ID newline fixture errors retained | [corrected raw](independent-raw-corrected.log), [roots](independent-root-probes.log) |
| Reviewer separation | PASS, 30 normal/disputed/final/high/critical cases | [host probes](independent-host.log) |
| Host authority ordering | PASS, 56 order cases; 1 initial decision accepted from 8 competing processes | [authority matrix](independent-authority-matrix.log) |
| Transition vs BLOCK review | FAIL, Verification appended after BLOCK committed | [review-race.log](review-race.log) |
| Storage death/error controls | 170 injected deaths recover valid prefixes; 16 fsync failures never acknowledge; one non-injected ordinal retained | [storage results](storage-expanded-results.json) |
| Symlink ancestry crash model | FAIL, ACK lacked two real-parent fsyncs; permitted loss image hides the record | [syscall trace](storage-probes-expanded/symlink/append.trace) |
| Formatting and diff whitespace | PASS | [format](format.log), [diff check](diff-check.log) |

Toolchain: Go `1.27.1 linux/amd64`. `GOCACHE` and `TMPDIR` were inside this output directory. Tests used copied source without `.git`. All Git mutations made by test fixtures occurred in scratch repositories. No real worker, model API, release, external effect or authentic human approval ceremony was exercised.

## Defect closure matrix

`FIXED` means the scoped defect has supporting source and execution evidence at this dirty target. It does not mean every path in that component is correct. `PARTIAL` means a useful correction exists but the stated closure remains incomplete. `OPEN` means the identified gap remains.

| ID | Status | Observation and evidence |
|---|---|---|
| H1 | **FIXED** | Current stage comes from ProjectStageState. The latest matching approval must follow the latest entry to that stage. Full-suite projected transition/resolution/revisit fixtures passed. Evidence: E03, E06. |
| H2 | **FIXED** | Pilot checks produced head/diff only when ApprovalBindsHeadDiff is true. Grounding and Planning pilot tests passed. Evidence: E03. |
| H3 | **PARTIAL** | Pending intent, atomic record creation and per-run locking work for ordinary paths. Symlink ancestry can still lose an acknowledged record; see R3-02. Evidence: E03, E10. |
| H4 | **FIXED** | All five supported terminal outcomes refuse decision/preflight/dispatch/review. Both commit orders now serialize authorizing host appends with core authority. Evidence: E03, E11. |
| M1 | **FIXED** | Capture applies StoreOutsideRepository, including symlink store controls. Verify-only does not capture. Evidence: E03. |
| M2 | **FIXED** | Resolve uses decodeStrictFile, which rejects trailing documents and unknown fields before a write. Evidence: E03. |
| M3 | **FIXED** | Selected rules cannot drop applicable constraints. All applicable rules are unioned and high/critical separation is enforced independently. Host dispatch explicitly enforces disputed/final flags. Evidence: E03, E07, E08. |
| M4 | **FIXED** | The review-to-Verification gate now calls requireApprovedHostRoute. Core changes also invalidate approval sequence/head. R3-01 separately affects concurrent host review changes. Evidence: E03, E06. |
| M5 | **FIXED** | Canonical scratch location, regular marker bound to store/run and exact untouched seed record are required. This is a trusted-host fixture guard, not hostile same-UID authentication. Evidence: E03, E05. |
| M6 | **FIXED** | Harness fingerprints every store file name and content, excluding lock files, and checks equality on refusals. The harness passed. Evidence: E05. |
| L1 | **FIXED** | Positive and negative RouteAttestation/ExecutionContext tests exist and ran in the full suite. Evidence: E03. |
| L3 | **FIXED** | The workflow invokes bash scripts/dogfood/host-dogfood.sh. The exact local harness passed; hosted Actions was not run. Evidence: E05. |
| N1 | **FIXED** | Live expected dirty content participates in ReviewArtifactDigest even if the optional approval worktree field is absent. Changed head/diff/content and predecessor identities refuse. R3-01 is a separate current-verdict sequencing defect. Evidence: E03, E06, E07. |
| N2 | **FIXED** | Capture snapshots an actual top-level repository, verifies exact base/head/diff, refuses dirt, and rechecks after execution. CI supports clean capture; approved dirty capture is the pilot path. Evidence: E03, E06, E07. |
| B1 | **FIXED** | Unchanged dirty artifacts advance with binding absent or present. Changed artifacts refuse old reviews without core/store mutation. Evidence: E06. |
| B2 | **FIXED** | Snapshot refuses assume-unchanged/skip-worktree. Raw byte checks and post-command checks refuse content or staged mutation before evidence append. Evidence: E03, E06, E07. |
| B3 | **FIXED** | CI refuses inherited selectors for capture. Shared Capture sanitizes selectors and forces the original-object view. Config injection remains available for safe.directory. Evidence: E03, E06. |
| B4 | **FIXED** | CI refuses a populated dirty-content binding even when the current checkout is clean. Absent-binding clean controls pass. Evidence: E03, E06. |
| B5 | **FIXED** | All host writes run under core lock then host lock and bind the checked core head. Full commit-order matrix passed. This does not fix the opposite-direction transition race R3-01. Evidence: E11. |
| B6 | **FIXED** | AppendWith evaluates against locked committed history. Only one conflicting initial decision committed across eight auditor subprocesses. Evidence: E03, E11. |
| B7 | **PARTIAL** | File/directory ordering and ordinary path recovery passed 170 process-death and 16 fsync-error controls. Real symlink target ancestors remain unsynchronized; R3-02 defeats the closure condition. Evidence: E10. |
| B8 | **FIXED** | All applicable rules contribute provider/model-family and capability constraints, including stricter lower-threshold rules. Independent union probes passed. Evidence: E07. |
| B9 | **FIXED** | isSeedRecord compares canonical complete serialized records after clearing only the computed head. Passed, metadata, payload, extra-record, marker and second-seed negatives ran. Evidence: E03, E05. |
| B10 | **PARTIAL** | Every other-unit rewind to Planning or earlier clears old route authority conservatively. Fresh review cannot restore a missing approval. A plan-derived graph and unaffected-unit preservation are still absent. Evidence: E03, E06, E11. |
| B11 | **FIXED** | Raw identity distinguishes owner execute, symlink type/target and absence. Group/other execute equivalence remains supported. Current dirty review controls and legacy expiry ran. Evidence: E03, E06, E07. |
| B12 | **FIXED** | Raw matching compares actual owner-execute to HEAD independently of core.filemode. Changed modes refuse both CI and old review; unchanged controls pass. Evidence: E03, E06. |
| B13 | **FIXED** | Worktree blobs are hashed from raw bytes without Git conversion. Tracked, nested, info and global attribute probes detect mixed-EOL changes. Clean-filter and stat-cache controls ran in the suite. Evidence: E03, E07. |
| B14 | **FIXED** | HEAD or index gitlinks explicitly refuse before capture, including missing submodules. No nested-content support is claimed. Historical gitlink diff policy is separately tested under F04. Evidence: E03, E07. |
| B15 | **FIXED** | Observation forces core.ignoreCase=false. Local/include/environment case aliases are detected. v7 worktree and v2 review domains retire incomplete predecessor identities. Evidence: E03, E06, E07. |
| B16 | **FIXED** | Snapshot compares resolved actual Git top level with the supplied root. Subdirectory/foreign core.worktree refuse. Valid absolute, relative, symlink and linked-worktree roots pass. Evidence: E06, E07. |
| B17 | **FIXED** | Snapshot uses --no-replace-objects. Capture forces GIT_NO_REPLACE_OBJECTS=1. SHA-1/SHA-256 dirt probes refuse and original-object capture succeeds. Predecessor clean/dirty outer identities refuse. Evidence: E03, E06, E07. |
| B18 | **FIXED** | Top-level identity canonicalizes the supplied root to an absolute path before symlink resolution. Relative, absolute, alias and linked-worktree controls pass. Case-insensitive-host aliases were not available. Evidence: E03, E07. |
| F01 | **FIXED** | v7 binds each changed path index mode/object/stage alongside disk state. Auditor staged bytes/mode/deletion/unmerged/intent-to-add and command-induced index mutations are detected. v1-v6 predecessor worktree identities refuse. Evidence: E06, E07. |
| F02 | **FIXED** | The common host append transaction serializes all four host writes with core authority. Unit/whole-run rewinds and all five outcomes passed in both orders. Evidence: E11. |
| F03 | **PARTIAL** | Absolute and relative ancestry are synchronized. The symlink path misses two real parents and permits loss of an acknowledged host chain; R3-02. Evidence: E10. |
| F04 | **FIXED** | The binary base..HEAD diff forces ignore-submodules=none. Historical gitlink changes retain the same nonempty diff across tested local ignore policies; current gitlinks refuse. Evidence: E03, E07. |
| F05 | **PARTIAL** | Conservative invalidation is safe after upstream rewinds, but there is no verified plan graph or demonstrable unaffected U3 preservation. Missing graph authority cannot be treated as completed graph support. Evidence: E06, E11. |
| C01 | **FIXED** | Changed or untracked .gitignore causes enumeration without ignore rules; mutations of hidden content change the digest. Explicit closure_check is null in the criteria source; evaluated against title plus B1/B2 and architecture 14. Evidence: E03, E07. |
| C02 | **FIXED** | Absolute conversion makes ancestor synchronization climb beyond dot. Relative-path syscall control synchronizes real parents. Symlink ancestry is the separate R3-02 defect. Explicit closure_check is null. Evidence: E03, E10. |
| C03 | **FIXED** | Review import validates active same-unit core route under the common core/host transaction. Revocation-first refuses and host-first binds the pre-revocation head. Explicit closure_check is null. Evidence: E03, E11. |
| C04 | **OPEN** | No plan bytes/graph loader is used during route replay. The code explicitly defers plan-derived dependents to Gate D. Explicit closure_check is null; this title-level gap remains open and must not be silently marked closed. Evidence: E06, E11. |

M3 covers both the 2026-10-14 finding and the stricter Gate A closure criterion. C01–C04 have null detailed closure checks in the supplied criteria file. Their title-based interpretation and overlaps are recorded in AUDIT.json. No hidden Gate A design document was used to invent a new schema.

## Findings and remediation

### R3-01 — A committed BLOCK review can be bypassed by a concurrent coding transition

Severity: **high**. Certainty: **observed**. Fix status: **OPEN**.

runTransition checks requireHostTransitionGate, releases its store locks, then calls transition.Apply. A valid independent BLOCK can append to the host chain in that interval without changing the core chain. Apply checks only core approval/state and therefore commits Verification using the now-superseded APPROVE. The auditor reproduced core sequence 2 -> 3 after BLOCK; a serial gate check of the same BLOCK refused. The scheduling-only overlay changes no acceptance decision.

**Evidence**

- `cmd/hma/transition.go:259-263; internal/transition/apply.go:127`: Host validation and core append are separate transactions; Apply does not reload the host review.
- `review-race.log; probes/race_test.go; race-overlay.json`: BLOCK committed before Apply; serial BLOCK refused; transition returned nil and wrote Verification.

**Proposed fix.** Add a transition transaction that holds the core run lock, then the host run lock, while evaluating the current host implementation/review, evaluating the bound core transition, and appending that core record. Provide locked load/append helpers or a callback API. Do not wrap the existing Apply unchanged in WithRunLock because its nested Load/Append would reacquire the lock.

**Reason.** The review verdict used for the gate must be current at the same ordering point as the core commit. Use the existing core-then-host lock order.

**Independent acceptance check**

- Rerun TestR3ReviewBlockInterleaving with a real BLOCK commit between the initial check and append. Add implementation replacement and both commit orders. Expected: A BLOCK/replacement committed first refuses before a new core record. A transition committed first must bind the same checked core/host snapshot. Unchanged APPROVE and optional dirty binding controls still pass.
- Run all 56 host-authority order cases and repository test/vet/race checks. Expected: No deadlock, permission widening, lost host record, or stale-core authorization.

**Trade-offs.** The transition transaction will hold both run locks. It needs a small API refactor to avoid recursive flock acquisition. Repository snapshot work can precede the transaction, but its bound identity and approval sequence must still be checked correctly.

**Alternative.** Use a cross-chain generation check/CAS under the same serialization boundary. Merely rechecking the host immediately before Append leaves another race window.

**Effort:** medium. The fix is proposed. The auditor did not implement it.

### R3-02 — Symlink store ancestry can lose an acknowledged host record

Severity: **high**. Certainty: **observed**. Fix status: **OPEN**.

syncAncestors applies filepath.Abs and filepath.Dir to the supplied spelling. It does not resolve symlinks. For alias -> new-parent/deep/store, the writer fsyncs store, store/host and the lexical alias parent, but never new-parent or deep. strace observed ACK with both parent syncs absent. Under the stated directory-entry crash model, loss of deep removes the acknowledged chain. Reconstructing that allowed image made record_visible false and Load returned an empty run. Actual hardware power loss was not induced; data loss is modeled from observed syscall ordering.

**Evidence**

- `internal/host/store.go:220-226,292-330`: Both ancestry functions walk lexical paths; syncAncestors makes them absolute but does not canonicalize the real ancestry.
- `storage-expanded-results.json#ancestry.symlink; storage-probes-expanded/symlink/append.trace`: ACK=true with two missing real-parent fsyncs. The allowed loss image removed the acknowledged record. Absolute/relative controls had no missing syncs.

**Proposed fix.** After creating the run directory, resolve its complete existing real path and synchronize every real ancestor through the durable boundary. Use that canonical path for the relevant commit/locking operations, or use directory descriptors with a clearly defined real-ancestry durability procedure.

**Reason.** The parent directory that contains an entry must be synchronized. Synchronizing a different lexical ancestor of a symlink does not establish the target entry.

**Independent acceptance check**

- Rerun storage_expanded_probe.py with a symlink at each store-path component, plus relative symlink spellings and concurrently created real ancestors. Expected: Every required real parent appears in successful fsync ordering before ACK. No allowed directory-entry-loss image loses an acknowledged record.
- Retain all 170 process-death positions, recovery/corruption controls and inject fsync errors at real ancestors. Expected: Acknowledged records survive. Interrupted attempts recover a valid prefix. A sync error never acknowledges.

**Trade-offs.** Canonicalization may change path aliases used for locks and needs care for not-yet-created directories. Same-UID hostile symlink retargeting remains outside this audit threat model. Additional ancestor fsyncs cost I/O.

**Alternative.** Explicitly refuse store paths containing symlink components before recording anything. This is a narrower support policy and would remove currently accepted aliases; document and test that compatibility cost.

**Effort:** small. The fix is proposed. The auditor did not implement it.

### R3-03 — Dependency graph closure remains incomplete

Severity: **medium**. Certainty: **observed**. Fix status: **PARTIAL**.

Route replay now invalidates every unit on another unit rewind to Planning or earlier. That prevents the observed stale-dependent authorization. It does not validate AcceptedPlanDigest against plan bytes or derive U2 -> U1 -> U0 while preserving independent U3. The implementation explicitly postpones that graph to Gate D. C04 has no detailed closure_check in the supplied criteria; its title-level missing-graph condition is still present. The conservative safety remedy is observed; complete graph functionality is not.

**Evidence**

- `cmd/hma/transition.go:193-201`: All routes are cleared conservatively; comment says Gate D narrows this with plan-derived dependents.
- `independent-command-probes.log; independent-authority-matrix.log`: Same/other/whole-run rewinds revoke old route authority; graph-based independent-unit preservation cannot be exercised because no graph loader exists.

**Proposed fix.** Bind host-replay plan input to the approval AcceptedPlanDigest, validate the declared dependency graph, and derive transitive affected units. Retain conservative invalidation for missing/corrupt/legacy plans. If this functionality is intentionally outside Gate B, obtain an explicit human scope/closure decision rather than marking these IDs fixed.

**Reason.** A conservative guard closes the stale-dependent safety path but does not supply the plan-derived behavior requested by the graph findings.

**Independent acceptance check**

- Use a digest-verified U2 -> U1 -> U0 plan plus independent U3; rewind U0; retry review import/dispatch; then approve a fresh legal route. Repeat with absent/corrupt plan and whole-run rewind. Expected: U1/U2 old authority refuses; U3 is preserved only with a verified graph. Missing/corrupt plans refuse conservatively. Fresh human-approved route restores only newly approved authority.

**Trade-offs.** Requires an immutable host plan source/schema and explicit compatibility behavior. A human may choose to retain coarse invalidation, but this auditor cannot authorize a closure-criteria change.

**Alternative.** Keep coarse invalidation as the supported Gate B behavior under an explicit human-approved scope clarification. Retain C04 as deferred/open until graph functionality is actually implemented.

**Effort:** medium. The fix is proposed. The auditor did not implement it.

### R3-04 — Task 14 describes pre-fix route invalidation behavior

Severity: **low**. Certainty: **observed**. Fix status: **OPEN**.

Task 14 still states that current digest lookup cannot detect dependent-unit scope and describes review artifact identity as approval fields. The code now conservatively revokes other-unit routes after upstream rewinds and uses ExpectedWorktreeDigest independently of the optional approval field. The document can mislead a host integrator about these safety boundaries.

**Evidence**

- `docs/task14-host-orchestration.md:155-160,185-192; cmd/hma/transition.go:85-89,193-201`: Normative host notes and executable conservative invalidation/live worktree comparison do not describe the same behavior.

**Proposed fix.** Update Task 14 to state conservative invalidation without graph precision, live expected-worktree artifact binding, and the current observation/review identity versions. Keep all deferred broader route-fact binding limitations explicit.

**Reason.** The host contract must describe the actual refusal boundary and optional approval semantics.

**Independent acceptance check**

- Compare the revised paragraphs with the existing dirty review, dependency rewind and legacy identity probes. Expected: Documentation predicts the observed pass/refusal behavior without claiming a verified graph or stronger human route attestation.

**Trade-offs.** Documentation-only correction. It does not resolve R3-01/R3-02 or supply graph support.

**Alternative.** Remove the stale implementation-specific paragraph and link to a precise current conformance table.

**Effort:** small. The fix is proposed. The auditor did not implement it.

## Reproduction and evidence handling

Run Go probes from the saved `snapshot/` with these settings. The commands intentionally fail only for the transition race.

```sh
export PATH="$HOME/.local/go/bin:$PATH"
export GOCACHE=/tmp/hma-gate-b-closure-20261016-r3/gocache
export TMPDIR=/tmp/hma-gate-b-closure-20261016-r3/tmp
export GOFLAGS=-buildvcs=false
cd /tmp/hma-gate-b-closure-20261016-r3/snapshot
go test -v -count=1 ./cmd/hma ./internal/engine ./internal/host ./internal/repostate -run "^TestR3"
go test -v -count=1 -overlay ../race-overlay.json ./cmd/hma -run "^TestR3ReviewBlockInterleaving$"
```

The overlay inserts one callback after the host gate check. The callback uses the real host review command to append `BLOCK`. It verifies a serial gate refusal, then lets the original transition proceed. It does not alter approval, review or append decisions.

Storage reproduction is [probes/storage_expanded_probe.py](probes/storage_expanded_probe.py), using the saved `host-probe.test` binary. The exact per-case argv is in storage-expanded-results.json. It requires ptrace permission. It uses no-clobber case directories; run it in a fresh audit-output copy or change its output directory names before repeating. Retain prior evidence. The test locks its goroutine to one OS thread for deterministic fsync injection. One remaining summed ordinal did not inject and is explicitly excluded from the 170 count.

The filesystem model permits loss of a directory entry before its real parent is synchronized. The fixture creates real ancestors without syncing them. The syscall trace records which parents the host writer actually syncs. The modeled loss then removes one unsynchronized entry. This is evidence of a missing durability step. It is not a hardware power-cut test.

## Controlled unblinding

Only after the Phase 1 hash was recorded did I inspect `git diff -- TODO.md`. TODO states that graph precision remains deferred as F05/C04. It also marks implementation batches checked while keeping independent re-audit and commit pending. These are implementer claims. They do not authorize a closure-criteria change. The current MVP gap paragraph is stale relative to the new conservative invalidation. No blind finding or verdict changed. No previous Gate B report was opened.

## Limitations

- Isolation is compromised, not fully blind. Fresh conversation had no implementer chat, but the initial broad read of the permitted Gate A audit-r1.json exposed historical findings/limitations beyond the requested N1/N2/M3 closure criteria. This is a protocol error. No prohibited Gate B prior review, implementer summary or TODO Gate B section was read before Phase 1 freeze.
- The criteria file first appeared in a truncated batched tool display. F01-F05 and C01-C04 were explicitly extracted after the first code diff. Initial invariant reconstruction covered their main safety themes, but exact-source ordering was not fully satisfied. No clean-context restart was attempted under the ban on nested agent CLIs/model APIs. Do not characterize this report as an isolated blind certification.
- The target is shared and dirty, not an OS-enforced read-only mount. Repository content was read-only by procedure. All source-copy probes, builds, caches, fixtures and induced crashes were under the assigned output. No source edit, commit, reset, checkout, index update or ref write was performed on the target. Initial git status did not explicitly disable optional Git metadata refresh; Git-internal stat-cache immutability is not established.
- Tests ran in a source copy without .git. Dogfood required GOFLAGS=-buildvcs=false for build stamping. This changes build provenance metadata only. Each behavioral Git fixture was initialized in scratch storage.
- Storage crash evidence combines actual syscall tracing/process death and a directory-entry durability model. It is not a power-cut test or a complete filesystem proof. Earlier ptrace was denied in the sandbox; the same bounded scratch tracing was then approved through automatic escalation. An initial syscall harness also overcounted per-thread positions; those attempts remain preserved. Final results identify 170 injected deaths, one non-injected ordinal, and 16 EIO refusals.
- The transition race uses a one-hook scheduling overlay. It changes only when the already-permitted BLOCK is imported. Actual host/core functions, records, approval validation and append code remain in use. No probability or production frequency estimate is claimed.
- Initial independent raw probes failed because auditor fixtures retained trailing Git output newlines. Corrected probes passed. Failed attempts and corrected output remain available.
- Closure criteria C01-C04 contain null closure_check/expected/requirement fields. Their assessments explicitly use titles, overlapping B/F criteria and the normative architecture; no new exact schema requirement is invented.
- Historical named probes in forbidden /tmp review directories were not run or read before Phase 1. Equivalent auditor probes were written here. The original 40/55 process-exit scripts were not inspected; the new 170-position matrix is a separate reproduction, not a claim to have rerun those exact scripts.
- Some source tests skip subprocess-only helpers and a superseded power-loss image. The auditor supplied a distinct storage crash/fault matrix rather than accepting the skipped test as proof. The full cross-product of every artifact type, legacy domain, binding form and filesystem was not exhaustively exercised.
- No case-insensitive host, actual worker dispatch/hi, real human challenge ceremony, hosted GitHub Actions run, release gate, final verified-success implementation, token/billing metering or same-UID hostile rewrite test occurred. Those are outside this read-only Gate B audit. The dispatch-declared reviewer/model/effort was not independently runtime-attested.
- No approved Jev tool was exposed. Auditor fallback decisions were used; no other model was presented as Jev.

No additional redesign opportunity is proposed. Correct the observed gaps first.

## Remediation order

1. Serialize host review validation and the core transition commit.
2. Establish real store ancestry durability, or explicitly refuse unsupported aliases.
3. Resolve the graph closure gap and correct the host contract text.
4. Freeze the result and obtain a fresh isolated independent audit.

## Single next action

Opus remediates R3-01 and R3-02, resolves the dependency-graph closure gap, and submits one frozen Gate B packet for a fresh isolated independent audit.

- **Current outcome:** FAIL. Evidence: E09, E10 and E11.
- **Owner and route:** Opus / lead implementer. Existing human-approved Opus 5.5 high route. Exact worker/provider/access-service/runtime must be attested and preflighted before any dispatch; this auditor does not dispatch it.
- **Permission envelope:** Proposed bounded Gate B fixes and associated tests/docs only, under valid existing coding authority. No audit authority to commit, push, publish, release, widen permissions or advance a stage.
- **Exact approval:** No approval is requested for this completed read-only audit. Any change to graph scope, acceptance criteria or human approval semantics needs explicit human authorization; subsequent stage advancement needs its own fresh challenge-bound approval. No challenge was minted by this audit.
- **Stop and escalation conditions:** Stop for absent valid implementation envelope, unavailable eligible independent route, unresolved authority conflict, criteria/scope change or permission expansion.
