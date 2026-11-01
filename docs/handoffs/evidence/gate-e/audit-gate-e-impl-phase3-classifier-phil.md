# Gate E Phase 3 — checkpoint classifier: independent code-behavior review (Phil)

- Reviewer: Phil via GitHub Copilot CLI. Self-reported model: Claude Sonnet 5.5 (`claude-sonnet-5.5`).
- Independence: same-family, separate-instance review (no shared context with the implementer). It is **not** cross-family. Lennon accepted this for now.
- Date: 2026-11-01. HEAD: `c49bd22` (unchanged). Go: go1.27.3 (`/home/yeli/.local/go/bin`).
- Subject: untracked `internal/engine/checkpoint.go` (330 lines), `internal/engine/checkpoint_test.go` (537 lines).
- References: draft §3.4, §5.1; merged `docs/architecture.md` §16 "Checkpoint triggers", "Checkpoint policy", "Checkpoint classifier" (lines 700-747). Merged text is authoritative.

## Verdict

**PASS WITH FINDINGS.** No BLOCKING or MAJOR finding. Two MINOR and two NOTE items below.

## Command results

| Command | Result |
|---|---|
| `go build ./...` | exit 0, no output |
| `go vet ./...` | no output |
| `go test ./internal/engine/... -v -run "TestCheckpoint\|TestClassifyCheckpoint"` | 23 tests, all PASS |
| `go test ./...` | all 13 packages `ok` (`cmd/hma` 112s) |
| `go test -race -count=1 ./internal/engine/...` | ok |
| `git status --short` | only the two `??` files; no tracked file modified |
| `git log -1` | `c49bd22`; nothing committed |

§5.1 name coverage: all 22 engine-side names are present and pass (`T2ContractHigh`/`Critical`, `T3PublicInterface`/`Schema`/`Protocol` are the split forms of the slash rows). Two rows (`PolicyCatchAllOrdinaryRefused`, `PolicyUnknownClassRefused`) are intentionally absent. They exist in `internal/model/checkpoint_policy_test.go` (Phase 2), and the file header says so. I accept this: they test `model.ValidateCheckpointPolicy`, not the classifier.

## Rule-by-rule check against §3.4 and merged §16

| Rule | Code | Match |
|---|---|---|
| T1 | `Current.ContractDigest != Baseline.ContractDigest` or `!stringSetEqual(permission classes)`; set compare dedupes and ignores order | yes |
| T2 | `maxRouteRisk(contractRisk, HostRisk, riskFloor) >= high` | yes |
| T3 | any path class in `public_interface`, `persisted_schema`, `protocol` | yes |
| T4 | `t4Observed \|\| t4Allowed`; allowed = `no_external_effect` absent from `Unit.PermissionClasses`; observed = path class `external_effect`/`irreversible` | yes |
| T5 | `security`, `approval`, `routing`, `evidence`, `closure` | yes |
| T6 | `Unit.MaterialMilestone && Stage == INDEPENDENT_VALIDATION` | yes |
| T7 | `FinalAuditOutcomes[TargetOutcome] \|\| Stage == RELEASE_AND_CLOSURE`; set = VERIFIED_SUCCESS, VERIFIED_WITH_WAIVERS, PARTIAL | yes (matches merged §16 and D7) |
| T8 | `len(unclassified) > 0` (no policy ⇒ every path unclassified; empty class set ⇒ unclassified; `ordinary` is a class so it is classified) | yes |
| Level | T7 ⇒ `final`; else IV and any trigger ⇒ `checkpoint`; else `per_change` | yes |
| RiskFloor | `high` iff `t3 \|\| t4Observed \|\| t5`; else `low`; allowed-only T4 and T8 do not raise it | yes |
| FamilySeparation | `combined >= high \|\| Disputed \|\| t7` | yes |
| RequiredRisk | `maxRouteRisk(contractRisk, riskFloor)` | yes |
| Triggers/UnclassifiedPaths | sorted; trigger set is a map, deduped | yes |

Glob matcher: segment DP, `*` = one segment, `**` = zero or more. Matches the merged policy text. Rules are unioned per path; order cannot matter.

## Specific subtleties

1. **T4 two forms — correct.** Allowed form sets only the trigger. Only `t4Observed` feeds `riskFloor`. Empirical probe: unit with `bounded_write`, no paths, IV stage gave `level=checkpoint, fam=false, floor=low, required=low, triggers=[IRREVERSIBLE_OR_EXTERNAL_EFFECT]`. `TestCheckpointT4AllowedEffect` and `TestCheckpointT4PathClass` cover both sides.
2. **HostRisk in family separation — correct.** `combinedRisk := maxRouteRisk(contractRisk, in.HostRisk, riskFloor)` drives both T2 and `familySeparation`. Probe: contract `low`, HostRisk `high`, no paths ⇒ `fam=true`, T2 fires.
3. **RequiredRisk excludes HostRisk — correct.** Same probe: `required=low`, `floor=low` while HostRisk is `high`.
4. **T8 `Contract == nil` clause omitted — agree, not a gap.** Merged §16 line 739 reads "Any changed path cannot be classified (above)." The "above" definition (line 706) covers only no-matching-rule or no-policy. Line 726 says "The classifier runs only on contract-bound runs." Neither carries the draft clause, which was tied to D10 option (2). The code tolerates `Contract == nil` safely (risk and policy digest default empty). With nil contract and a nil policy, every path is still T8. Nothing is under-protected for a contract-bound run.
5. **Known-gap test direction — correct.** `TestCheckpointMisclassifiedPathKnownGap` classes `cmd/hma/**` as `ordinary` and asserts T3 is **absent** and `UnclassifiedPaths` is empty. If someone patched the classifier to close the gap (e.g. content/path heuristics raising T3), the first assertion fails. If someone made `ordinary` count as unclassified, the second fails. It does not assert "no triggers at all", so an unrelated trigger would not break it; this is acceptable.

## Determinism and non-mutation

- `TestCheckpointDeterministic` shuffles once (seed 42) and also varies permission-class order (baseline vs current) and duplicate classes. One shuffle is a weak sample (see MINOR 1).
- My own probe (run via `go test -overlay`, source kept outside the repo; nothing written to the worktree): 5,000 random shuffles of a 6-entry manifest (including a duplicate path with different statuses) and 5 rules (overlapping, multi-class, `ordinary`), plus reordered/duplicated permission classes. All 5,000 `reflect.DeepEqual` against the baseline output, including `ManifestDigest`. No diffs.
- Code reading: output order comes from `sort.Slice`/`sort.Strings`; map iteration only feeds sorted slices or booleans. `manifestDigest` sorts a copy with a total order.
- `TestClassifyCheckpointDoesNotMutateInput` marshals the whole `CheckpointInput` (following `Contract`, `Policy` pointers, `Policy.Rules` entries, `Manifest` entries, `PermissionClasses` slices) before and after, and compares bytes. It does exercise slices and pointers. The classifier never writes to input, so this passes. Limit: JSON marshalling does not see unexported fields or capacity aliasing; neither exists in these types.

## Findings

| # | Severity | Finding |
|---|---|---|
| 1 | MINOR | `TestCheckpointDeterministic` uses one fixed shuffle. A defect that depends on a specific order could slip through. Suggest many iterations or all permutations of the 4-entry manifest. Probe above shows the code is order-independent today. |
| 2 | MINOR | `**` matches zero segments, so glob `a/**` matches the bare path `a`. The merged text says `**` matches "any number of segments", so this is consistent. It does mean a file named `a` is matched by `a/**`. No test pins this either way. Low safety impact (over-classifies, never under-classifies). |
| 3 | NOTE | `manifestDigest` returns `""` on marshal error. `ChangedPath` is a plain struct, so the error cannot occur. Phase 4 should not treat `""` as a valid digest. |
| 4 | NOTE | `Level` returns `checkpoint` only at `INDEPENDENT_VALIDATION`. Outside IV, triggers can fire while level stays `per_change` (D8 (1), spec-conformant). Phase 4 callers must read `Triggers` and `FamilySeparation`, not `Level` alone, before IV. |

## Scope confirmation

Only `internal/engine/checkpoint.go` and `checkpoint_test.go` are new. No tracked file changed. No commit made. This evidence file is the only file I wrote in the repo.

## Next action (§18)

Human decision: accept this review and authorize Phase 4 (wiring into packet issuance and gates), or request fixes for the MINOR items first.
