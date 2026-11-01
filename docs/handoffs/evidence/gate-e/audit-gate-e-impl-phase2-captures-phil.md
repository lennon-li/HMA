# Gate E Phase 2 (new captures) — independent code-behavior review

- Reviewer: Phil, OpenAI `gpt-6-luna`, GitHub Copilot CLI, verification/review mode.
- Date: 2026-11-01. HEAD: `0e3ae82` (unchanged). Worktree: uncommitted.
- Toolchain: `/home/yeli/.local/go/bin/go`, `go version` = go1.27.3 linux/amd64.
- Scope: `git diff internal/model/contract.go`; new `internal/model/checkpoint_policy.go`, `checkpoint_policy_test.go`, `checkpoint_fields_test.go`.
- References: draft §3.2/§3.3; live `docs/architecture.md` §9, §10, §16 "Checkpoint policy".

## Answer

PASS WITH FINDINGS. The code does what the architecture text specifies. The explicit-zero check runs, catches the case `requireAllKeys` cannot, and covers every plan unit. Digests are byte-identical to HEAD digests (verified empirically). One MINOR finding: the two in-repo digest-compat tests prove internal consistency only.

## Command results

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0; all 12 packages `ok` (`cmd/hma` 111.8s; `internal/model` cached) |
| `go test -race -count=1 ./internal/model/...` | `ok` 1.167s |
| `git status --short` | `M internal/model/contract.go`; `??` the three new files. Nothing else. |
| `git rev-parse --short HEAD` | `0e3ae82` (no commit) |

## Specific checks

### 1. `refuseExplicitZero` is wired and effective — PASS
- `ParseAcceptanceContract` and `ParsePlan` call it after `decodeCanonicalInput` and before `Validate*` (diff lines read directly).
- The refusal message in my probe was `optional field may not be set to its zero value; omit the key instead`. That text comes only from `walkExplicitZero`, so the new check, not `requireAllKeys`, caused it.
- Reasoning: `omitempty` drops the zero field from the typed re-marshal, so `requireAllKeys` never looks for the key. Explicit and absent decode identically without the new check. The new check reads the raw document, so it can tell them apart.
- Tests `TestExplicitZeroOptionalFieldRefused` cover contract `""`, contract `null`, plan `false`, plan `null`. All pass, and each asserts the fixture edit applied.

### 2. `requireAllKeys` not weakened — PASS
The diff hunks are: two struct-field additions, two call-site insertions, and a new block after `requireAllKeys`. The function body is not in any hunk. All other fields stay required and non-null.

### 3. `"[]"` iterates every unit — PASS (empirical)
- Hand trace: in `walkExplicitZero`, `"[]"` loops `for _, elem := range arr` and returns on the first error. It does not stop at element 0 or keep only the last.
- Throwaway probe (temp copy of HEAD plus the new files, outside the repo): a plan with 4 units. I injected `"material_milestone":false` into each unit in turn. All 4 injections were refused with the explicit-zero error (units 0, 1, 2, 3).
- The existing test injects into unit 0 only (`strings.Replace(..., 1)`). It does not prove all-unit iteration. See finding M-2.

### 4. Digest compatibility — PASS empirically, with a test-strength finding
- The two named tests (`TestContractDigestUnchangedWithoutPolicyDigest`, `TestPlanDigestUnchangedWithoutMilestone`) do **not** compare against a pre-Gate-E fixture. They compare two fresh computations in the same run. The test comments say so. They prove internal consistency, not backward compatibility. See finding M-1.
- Independent check: I ran the same probe (`ParseAcceptanceContract(mustJSON(testContract()))`, then `ParsePlan(mustJSON(testPlan(cd)))`) on a `git archive HEAD` copy and on HEAD plus the diff:

| Document | HEAD digest | Phase 2 digest |
| --- | --- | --- |
| contract | `3891a798704b77d765126b51c6affa21239e741ba5ec4ed4a82319a6675ace36` | identical |
| plan | `8882eff28fcbe9e40e746993eee29e9a2b4fb6899a2fa6002f1577090be6b24d` | identical |

- Conclusion: true backward compatibility holds for these fixtures. The in-repo tests do not guard it.

### 5. Catch-all-ordinary refusal narrowness — PASS
- Code: `r.Glob == "**" && len(r.Classes) == 1 && r.Classes[0] == "ordinary"`. This matches §16 exactly.
- Refused case: tested (`catch-all`, `**`, `[ordinary]`).
- Near-miss tested: `**` with `[ordinary, security]` parses. 
- Near-miss not tested: other globs such as `*`, `**/*`, `docs/**` with only `ordinary`. `testCheckpointPolicy` has `docs/**` → `ordinary` and parses in the valid-case test, so that one is covered indirectly. `*` and `**/*` are untested, though the code compares the glob string exactly, so they cannot trigger. Also untested: `["ordinary","ordinary"]` on `**` (length 2, not refused; matches the "only class" wording loosely). See M-3.

### 6. Duplicate/empty rule ID refusal — fine
Not stated in §3.2/§16, but it does not contradict them. It is cheap, deterministic, and makes later reporting by rule id unambiguous. I classify it as a harmless implementation choice. It is a NOTE only so the architecture text can mention it if the owner wants the contract to be complete (§16 says only "an `id`").

### 7. Scope — PASS
Only `internal/model/contract.go` is modified and three new files are in `internal/model/`. No other file and no commit. This evidence file is the single new file outside the model package.

## Findings

| ID | Severity | Finding |
| --- | --- | --- |
| M-1 | MINOR | `TestContractDigestUnchangedWithoutPolicyDigest` and `TestPlanDigestUnchangedWithoutMilestone` compare two fresh digests, not a recorded pre-Gate-E digest. They would not catch a future change that alters the digest of existing documents. Compatibility holds today (check 4), but is unguarded. Recommendation: pin the HEAD digests above as literals for the test fixtures, if fixture stability allows. |
| M-2 | MINOR | The explicit-zero plan test injects into unit 0 only. The `"[]"` iteration is correct (traced and probed on 4 units) but unguarded by an in-repo test. |
| M-3 | MINOR | Catch-all near-miss coverage is partial (check 5). Only the extra-class case is tested directly. |
| N-1 | NOTE | Duplicate/empty rule ID refusal is an unspecified addition; judged reasonable and harmless. |
| N-2 | NOTE | `ParseCheckpointPolicy` strictness rests on the shared `decodeCanonicalInput`; `TestCheckpointPolicyFieldsRequired` and `...DecodingIsStrict` pass. The test comment "duplicate/case-variant keys" refers to cases the test does not exercise (only `unknown-field` and `trailing`). Documentation nit only. |
| N-3 | NOTE | Whitespace-only `checkpoint_policy_digest` (e.g. `" "`) is not refused by `isZeroString`. Semantic form checks, if any, belong to later phases; not in the §9 text read. Not a defect against the accepted contract. |

No BLOCKING or MAJOR findings.

## Limits
- I did not run a mutation of `requireAllKeys`-only behavior (removing the new call) to show acceptance; the error-message provenance and the reasoning cover it.
- The probe ran in a temp directory (`/tmp/tmp.*`, outside the repo). I could not delete it (`rm` denied); it holds only `git archive` copies and probe tests. No repo file was changed by the probe.

## Verdict

**PASS WITH FINDINGS** (3 MINOR, 3 NOTE; 0 BLOCKING, 0 MAJOR).

Next action: the owner decides whether to harden the tests (M-1 to M-3) before Phase 3.
