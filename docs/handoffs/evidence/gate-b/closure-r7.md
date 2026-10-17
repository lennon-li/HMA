# Gate B independent closure audit

**Verdict: PASS — Gate B scope only.**

Observed: 47 listed defects are FIXED. Three dependency-graph items remain PARTIAL under the human Gate D deferral. No new blocking defect was found.

Target: `main`, HEAD `67965541b7032a585a68e48f3d10f4a998d5399e`, plus the frozen working tree. The repository is unchanged. All 29 untracked files have matching start/end SHA-256 hashes. All 145 scoped files match the start copy in bytes, mode and symlink form.

Tracked diff SHA-256: `2a2a487a251bdad81c6aee721fe201494c19f409ddcc4b30698192be5343ee56`.

Phase-1 SHA-256: `937647764d20cbf668bfba4f6781eb1ca1cc404bd59b8c7db1d60ac94d283392`. Phase 1 was frozen before retained fixtures or Gate B TODO claims were read.

**Isolation: isolated review context.** The repository was not mounted read-only. The auditor ran tests in a scratch copy. Start/end evidence verifies read-only conduct. Repository author comments were visible as audited content. No prior review report or verdict was read.

The authoritative objective is the consolidation handoff §5. The normative sources are `docs/architecture.md`, `AGENTS.md`, Task 13 and Task 14. The defect inputs are the 2026-10-14 audit, only the N1/N2/M3 closure criteria from Gate A, and neutral `criteria-v7.json`. `.serena/` and the two architecture-amendment drafts were excluded from evaluation. Their untracked hashes were still preserved.

The configured reviewer route was Liz / OpenAI / gpt-6.1-sol / high. That identity came from the dispatch. The audit has no separate runtime identity attestation. Jev was unavailable. The auditor used the documented fallback.

## Verification evidence

| Check | Observation | Evidence |
| --- | --- | --- |
| Required suite | `go test -count=1 ./...` passed all 12 packages. | [test.log](test.log) |
| Static check | `go vet ./...` passed. | [vet.log](vet.log) |
| Required race suite | `go test -count=1 -race ./...` passed all 12 packages. | [race.log](race.log) |
| Dogfood | Real host/transition CLI harness passed all 17 synthetic checks. | [dogfood.log](dogfood.log) |
| Independent artifact matrix | 26 cases × 2 optional-binding forms passed. Fresh clean/dirty controls advance. Changed artifacts and old domains refuse. | [probes-cmd.log](probes-cmd.log) |
| Independent race probes | Artifact, CLI independence, commit-order and BLOCK-review probes passed under `-race`. | [probes-cmd-race.log](probes-cmd-race.log) |
| Independent persistence probes | 51 first-append and 47 successor process-exit points recovered valid prefixes. Ancestor and fsync-failure controls passed. | [probes-host.log](probes-host.log) |
| Retained power-image model | First/successor/recovery completed after 55/97/109 iterations, including the final acknowledged control. All-loss directory images preserved acknowledged records. | [retained-host.log](retained-host.log) |
| Retained syscall matrices | Original script: 35 injected deaths. Expanded script: 175 injected deaths and 2 non-injected controls. Both passed all 17 fsync-failure positions. | [original](storage-results.json), [expanded](storage-expanded-results.json) |
| Process concurrency | Each of 5 rounds accepted exactly 1 of 16 conflicting initial decisions. | [raw-concurrency-results.json](raw-concurrency-results.json) |
| R6-01 historical replay | Actual predecessor observations and fixed v2 review serialization refuse both clean replay branches. Fresh v3 controls pass. | [retained-cmd.log](retained-cmd.log), [adaptation](retained/r6-adaptation.diff) |
| R6-01 negative control | Restoring only the v2 domain makes both legacy-clean tests fail as expected. | [negative-review.log](negative-review.log) |
| Repository freeze | HEAD, branch, diff, status, all untracked hashes and 145 scoped file states match. | [freeze-verification.json](freeze-verification.json) |

Go version: `go1.27.1 linux/amd64`. Git version: `2.43.0`. `bash -n`, ShellCheck and `git diff --check` also passed. Exact test argv, environments and exit codes are in [COMMANDS.json](COMMANDS.json). The full source inventory is in [source-manifest.json](source-manifest.json). The report passed the skill’s structural validator; this validates form only. See [report-validation.log](report-validation.log).

## Defect-by-defect closure

`FIXED` applies to this frozen target. `PARTIAL` is not a claim that the deferred graph feature exists. The report-contract requirement status is `pass` only for the accepted Gate B scope.

| ID | Closure | Observation and evidence |
| --- | --- | --- |
| H1 | **FIXED** | CI projects the current stage. It selects an approval for leaving that stage after the latest entry. Projected-stage, stageless-record and earlier-visit controls pass. [test.log](test.log), [coverage-detail.log](coverage-detail.log) |
| H2 | **FIXED** | Grounding and Planning pilot captures pass with empty head/diff approval fields. Binding-stage stale head/diff inputs refuse. [probes-pilot.log](probes-pilot.log) |
| H3 | **FIXED** | Host appends serialize. Pending intent recovery retains verified prefixes. Process-exit, power-image, fsync-failure and concurrent-writer checks pass. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| H4 | **FIXED** | All five supported terminal outcomes refuse host writes in the serial suite. Actual core ABORTED API calls serialize against all four host commands in both commit orders. [test.log](test.log), [coverage-detail.log](coverage-detail.log) |
| M1 | **FIXED** | CI refuses stores inside the tested repository, including symlink path controls. [test.log](test.log), [coverage-detail.log](coverage-detail.log) |
| M2 | **FIXED** | Resolve uses the shared strict decoder. A second document and trailing garbage refuse. Whitespace remains accepted. [test.log](test.log), [coverage-detail.log](coverage-detail.log) |
| M3 | **FIXED** | HIGH/CRITICAL cannot select weaker policy separation. Applicable capabilities are conjoined. Final/disputed host review dispatches require both family separations. Normal-work controls pass. [probes-engine.log](probes-engine.log), [probes-atomicity.log](probes-atomicity.log) |
| M4 | **FIXED** | The second coding gate revalidates current route approval, review approval, core head and exact live artifact. [probes-atomicity.log](probes-atomicity.log), [coverage-detail.log](coverage-detail.log) |
| M5 | **FIXED** | Only a regular marker naming the canonical scratch path and run passes. The sole prior seed record must equal seedCore output. Validly hashed PASSED, actor, timestamp, unit and control alterations refuse. [test.log](test.log), [probes-seed.log](probes-seed.log) |
| M6 | **FIXED** | Every dogfood refusal compares a SHA-256 fingerprint of store file names and bytes. Core and host record/head files are covered. Empty lock files are excluded. [dogfood.log](dogfood.log) |
| L1 | **FIXED** | Direct structural route-attestation tests exercise all required fields and execution-context vocabulary. Record-boundary validation controls pass. [test.log](test.log), [coverage-detail.log](coverage-detail.log) |
| L3 | **FIXED** | The CI workflow invokes the hardened dogfood harness. A local real-CLI run passes all 17 checks. [dogfood.log](dogfood.log) |
| N1 | **FIXED** | Full CLI controls review A before mutation. An otherwise valid current B approval cannot reuse the A review. Current clean/dirty artifacts advance. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| N2 | **FIXED** | Empty/foreign roots, stale head/diff, unapproved dirt and checkout mutation refuse. Real clean and explicitly bound dirty pilot captures pass. CI supports clean revision capture. [probes-pilot.log](probes-pilot.log), [probes-cmd.log](probes-cmd.log) |
| B1 | **FIXED** | Full dirty-artifact transitions reject stale reviews with the optional approval worktree field absent or populated. Unchanged controls pass. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| B2 | **FIXED** | Hidden-index flags refuse. Raw tracked-byte comparison detects dirt independently of Git stat caches. Command-induced content, mode and index mutations refuse evidence append. [probes-cmd.log](probes-cmd.log), [test.log](test.log) |
| B3 | **FIXED** | CI refuses inherited repository selectors. Capture strips selectors and forces original-object Git reads. Snapshot and capture use the explicit checkout. [probes-cmd.log](probes-cmd.log), [test.log](test.log) |
| B4 | **FIXED** | CI refuses any populated worktree binding for a clean checkout. The absent-binding clean control passes. [probes-cmd.log](probes-cmd.log) |
| B5 | **FIXED** | Host commits execute inside the core lock. A committed termination wins before decision, preflight, dispatch and review; host-first commands bind the pre-terminal core head. [test.log](test.log), [retained-cmd.log](retained-cmd.log) |
| B6 | **FIXED** | The deterministic host-history check runs inside the host transaction. Five real-process rounds each accept 1 of 16 conflicting initial decisions. Accepted ordinary concurrent store appends are retained. [test.log](test.log), [raw-concurrency-results.json](raw-concurrency-results.json) |
| B7 | **FIXED** | Acknowledged records survive the current all-loss directory model. Absolute, relative and multi-hop link ancestry is synchronized. Concurrent same/different-run initialization and fsync failures pass. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| B8 | **FIXED** | Every applicable rule contributes required separation and validator capabilities. Same-tier and lower-tier stricter capability controls reject the weaker selection. [probes-engine.log](probes-engine.log) |
| B9 | **FIXED** | Synthetic seeding compares the full normalized seed record after clearing only its computed head anchor. Validly hashed foreign metadata/disposition/control and repeated seeding refuse. [test.log](test.log), [probes-seed.log](probes-seed.log) |
| B10 | **FIXED** | An approved rewind to Planning or earlier invalidates every existing unit route. Re-attestation cannot revive revoked authority. A fresh human-approved route can establish new authority. Graph narrowing is deferred. [test.log](test.log), [retained-plan.log](retained-plan.log) |
| B11 | **FIXED** | Worktree v8 distinguishes raw bytes, type, owner execute, absence and staged entry. Group/other execute equivalence remains supported. Older dirty domains refuse. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| B12 | **FIXED** | Raw mode checks and forced Git mode settings detect owner-execute changes under core.filemode=false. CI pre-existing and command-induced chmod negatives refuse. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| B13 | **FIXED** | Raw blob comparison detects EOL changes from tracked, nested, info and global attributes. Conversion cannot label changed bytes as clean. Supported unchanged cases pass. [test.log](test.log), [retained-snapshot.log](retained-snapshot.log) |
| B14 | **FIXED** | HEAD/index gitlinks and sparse checkouts refuse explicitly. Ordinary repositories remain supported. Historical gitlink diffs are not suppressed by ignore policy. [test.log](test.log), [retained-snapshot.log](retained-snapshot.log) |
| B15 | **FIXED** | Local/include/environment ignorecase settings cannot hide a case alias. Worktree v8 expires predecessor v5 identities. Full current case-alias review negatives refuse. [probes-cmd-race.log](probes-cmd-race.log), [retained-snapshot.log](retained-snapshot.log) |
| B16 | **FIXED** | Snapshot requires the actual top-level directory. Subdirectory and configured foreign-root cases refuse. Root aliases and linked worktrees remain supported. [test.log](test.log), [retained-snapshot.log](retained-snapshot.log) |
| B17 | **FIXED** | Snapshot disables replacement objects. Capture sets GIT_NO_REPLACE_OBJECTS=1. SHA-1/SHA-256 and inherited-variable controls pass. Review v3 expires earlier clean observation identities. [test.log](test.log), [retained-snapshot.log](retained-snapshot.log) |
| B18 | **FIXED** | Absolute, relative and symlink forms of the real top level pass. Mismatched roots continue to refuse. [probes-cmd.log](probes-cmd.log), [retained-snapshot.log](retained-snapshot.log) |
| F01 | **FIXED** | Index content, mode, deletion and unmerged stages change artifact identity. A command that changes the index cannot append clean CI evidence. Historical dirty domains expire. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| F02 | **FIXED** | Host/core commit ordering is serialized for all host operations. Actual core API abort, unit rewind and whole-run rewind pass both orders. Accepted records are retained. [test.log](test.log), [retained-cmd.log](retained-cmd.log) |
| F03 | **FIXED** | Every append synchronizes existing ancestry. Absolute, relative and link ancestry failure controls refuse acknowledgment. Power-image recovery and concurrent initialization pass. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| F04 | **FIXED** | Historical gitlink removal has a nonempty stable canonical diff under local diff.ignoreSubmodules and submodule ignore settings. [test.log](test.log), [retained-snapshot.log](retained-snapshot.log) |
| F05 | **PARTIAL** | Plan-derived selective invalidation is absent. Conservative all-unit invalidation passes. The human deferred graph-derived scope to Gate D. [test.log](test.log), [retained-plan.log](retained-plan.log) |
| C01 | **FIXED** | Added, changed and deleted .gitignore cases include affected untracked bytes. Changing hidden A to B changes identity and refuses old review. [probes-cmd-race.log](probes-cmd-race.log), [retained-snapshot.log](retained-snapshot.log) |
| C02 | **FIXED** | Relative store paths are made absolute before ancestor synchronization. Relative ancestry and injected sync-failure controls pass. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| C03 | **FIXED** | Review import checks current core route authority under both locks. Revoked-route imports refuse without record/head changes. [test.log](test.log), [retained-cmd.log](retained-cmd.log) |
| C04 | **PARTIAL** | There is no validated plan graph on the production replay boundary. The accepted Gate B behavior invalidates all units. Graph validation is deferred to Gate D. [test.log](test.log), [retained-plan.log](retained-plan.log) |
| R3-01 | **FIXED** | The transition CLI holds the host lock from gate check through core append. The ordering regression passes. A committed BLOCK review independently prevents Verification without store changes. [probes-atomicity.log](probes-atomicity.log), [test.log](test.log) |
| R3-02 | **FIXED** | Physical and lexical link ancestry are collected and synchronized. Both link directions and failure controls pass. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| R3-03 | **PARTIAL** | Selective graph closure remains unimplemented. Conservative invalidation and fresh-route controls pass. The human deferred selective closure to Gate D. [test.log](test.log), [retained-plan.log](retained-plan.log) |
| R3-04 | **FIXED** | Task 14 now states conservative all-unit invalidation, the Gate D boundary and host-then-core lock ordering. These match inspected code and observed replay behavior. [test.log](test.log), [retained-plan.log](retained-plan.log) |
| R4-01 | **FIXED** | Transact calls commit inside WithRunLock. Actual core transition API tests prove core withdrawal cannot commit before durable host append in the host-first order. [test.log](test.log), [retained-cmd.log](retained-cmd.log) |
| R4-02 | **FIXED** | Per-component path resolution records intermediate symlink lookup parents. Relative and absolute multi-hop targets pass sync coverage and failure injection. [storage-results.json](storage-results.json), [storage-expanded-results.json](storage-expanded-results.json) |
| R5-01 | **FIXED** | Tracked leaves under a link or non-directory parent are absent to raw comparison and hashing. CI refuses both structures. Old dirty identities expire. Real-directory controls pass. [retained-cmd.log](retained-cmd.log), [probes-cmd-race.log](probes-cmd-race.log) |
| R5-02 | **FIXED** | The supported host CLI boundary carries final_validation/disputed on review dispatch. It refuses either same provider family or same model family. Ordinary low-risk and separated routes pass. [probes-atomicity.log](probes-atomicity.log), [test.log](test.log) |
| R6-01 | **FIXED** | A predecessor observation helper reproduces the old clean parent-symlink omission. Persisted review-artifact/v2 identities refuse both full replay branches. Fresh v3 clean/dirty controls advance. Reverting only the review domain to v2 makes both legacy-clean tests fail. [negative-review.log](negative-review.log), [retained/r6-adaptation.diff](retained/r6-adaptation.diff) |

## Findings and deferred work

No current actionable defect was reproduced in the scoped changed code. No discretionary redesign is recommended. This is empirical closure evidence. It is not proof against every possible interleaving or environment.

**F05, C04 and R3-03 remain PARTIAL.** The neutral criteria note records Lennon’s 2026-10-16 decision to defer plan-derived dependent invalidation to Gate D. The current implementation invalidates every unit route after any unit returns to Planning or earlier. The retained replay test confirms this behavior. It also confirms that a fresh approved route can establish authority without a resolved plan graph.

The future remedy is to bind replay to a validated plan graph and compute transitive dependents from it. Missing or corrupt graph data must retain conservative invalidation. Verification must use `U2 -> U1 -> U0` and independent `U3`. Rewinding `U0` must revoke `U1/U2`. `U3` may survive only with a verified graph. Review re-attestation must not revive revoked authority.

Proposed effort: medium, in Gate D. Trade-off: selective invalidation adds persisted plan/provenance and migration work. The safe alternative is to retain all-unit invalidation. The graph remedy is proposed. It was not implemented or validated by this audit.

## Controlled unblinding

The immutable [PHASE1.json](PHASE1.json) has SHA-256 `937647764d20cbf668bfba4f6781eb1ca1cc404bd59b8c7db1d60ac94d283392`. After freezing it, the auditor read TODO Gate B as an artifact and selected retained test sources. No earlier verdict/report was read. Retained checks did not change the phase-1 findings. They added evidence.

The R6 matrix needed one semantic preservation: historical parent-review cases must retain `hma.host.review-artifact/v2`. Calling the new helper would create a current review and would not test historical replay. The saved adaptation changes only those historical serializer calls and the helper location. The predecessor raw/snapshot implementation was built separately. The current target was not replaced by predecessor code during its final transition checks.

The sandbox initially denied ptrace. Those two failed attempts remain in the logs. Automatic review then allowed ptrace-enabled scratch execution. Both syscall matrices completed successfully. No automatic-review rejection remains unresolved.

## Limits

- The repository was not mounted read-only. Read-only behavior was procedural and verified by start/end hashes and a 145-file byte/mode/symlink comparison against the start copy. All tests, helper builds, stores and probe changes were under this output directory.
- Isolation is conversational blinding in a fresh session. It is not a fresh machine or a hardware isolation claim. Repository code/test comments remained visible as audited artifacts. No prior verdict/report was needed or read; retained test sources and TODO Gate B claims were read only after PHASE1.json was frozen.
- The storage evidence uses process death, syscall fault injection and a directory-entry loss model. It does not prove behavior on all hardware/filesystems or physically cut device power.
- Linux case-sensitive filesystem only. Case-insensitive host behavior was not exercised. Windows and other unsupported platform locking paths were not exercised.
- Go 1.27.1 linux/amd64 and Git 2.43.0 were used. The earlier audit cited Go 1.24.0; that toolchain was not separately reproduced.
- Named historical checks were rerun where retained fixtures were available and relevant. The report does not claim every old test name was replayed verbatim. Current equivalent matrices and source analysis supply the listed coverage. Dynamic syscall counts differ from historical 40/55 counts because the current protocol and environment differ.
- Repository helpers supply fixture creation and JSON serialization in some independent probes. Assertions and new negative matrices were auditor-written. The explicit retained-fixture adaptations and provenance are available.
- The synthetic dogfood run does not dispatch a real worker, perform an actual route hi handshake, or provide genuine human approval. It validates CLI enforcement, not full host MVP conformance.
- F05, C04 and R3-03 remain PARTIAL. The authoritative criteria note defers plan-derived invalidation to Gate D. A fresh route may be recorded without resolving a plan artifact; conservative all-unit rewind invalidation remains the Gate B boundary.
- Full approved provider/model/permission fact binding, Gate C integration, Gate D leases/graph semantics, successful release outcomes and release/publication authority are outside this Gate B audit.
- Same-UID coherent store rewriting is out of scope under the trusted-host threat model. No external APIs, nested agent CLIs, model calls, audited-repository Git writes or audited-file edits were used. Git fixture creation occurred only in scratch test directories. Jev was unavailable; the auditor used the recorded fallback.

## Single next action

**Lennon reviews this frozen packet and records the Gate B acceptance decision.**

Rationale: the current Gate B checks pass. The accepted Gate D deferral remains explicit. Owner: Lennon. Route: human review. Permission envelope: review and record the decision only. Exact approval: explicit acceptance bound to the HEAD, diff digest, freeze manifest and audit report above. No stage challenge was issued. Stop acceptance if target bytes change or a safety finding remains unresolved. Escalate scope or approval-semantic changes to the human.
