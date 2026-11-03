# Independent audit: `cmd/hmarun` smoke-test fixes 1-4 (Phil, 2026-11-03)

**Verdict: PASS WITH FINDINGS.** Fixes 1-3 are correct. Fix 4 is correct for every claim it makes, but it is incomplete: the prompt omits the required `fix` object for actionable findings (F1).

Not performed: I could not read `/home/yeli/obsidian/AgentSystem/scripts/agent-run-ledger` or `agent-dispatch`. The permission layer denied it. The claims that `--report-file` is required, and that `claude-sonnet-5` / `gpt-5.3-codex` are accepted and report providers `anthropic` / `openai`, rest on the requester's real-dispatch evidence. I did not independently confirm them.

Nothing under `cmd/hmarun/` was edited. `git status --short` before and after is identical:
```
?? cmd/hmarun/
?? docs/handoffs/evidence/mvp/
```
My only file is this report. Throwaway check files were in `/tmp` and used `go run -overlay`; nothing was written into the repo.

## Verification commands

| Command | Result |
|---|---|
| `go version` | go1.27.3 linux/amd64 (`/home/yeli/.local/go/bin`) |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./cmd/hmarun/... -v -race` | `ok ... cmd/hmarun 18.161s`; 35 `--- PASS`, 0 `--- FAIL` (top-level lines) |
| `go test ./...` | all packages `ok` (`cmd/hma` 119.0s, `cmd/hmarun` 17.2s, `internal/host` 1.06s, others cached) |
| Throwaway: worked example through `host.ValidateAuditReport` | returned `[]` (no violations) |
| Throwaway: report with a `defect` finding and no `fix` | `[findings[0].fix: required for actionable finding]` |

## Per-fix result

| Fix | Result | Location |
|---|---|---|
| 1. Valid `--model` values | Correct | `run.go` `dispatchConfigFor` (~l.437-440) |
| 2. phil `openai` / `gpt-5.3-codex` | Correct (logic verified; receipt values taken from requester) | same; `host.go` ~l.1025-1037; `review_import.go` l.55-60 |
| 3. `--report-file` | Correct | `run.go` ~l.1913-1923 |
| 4. `reviewTaskText()` | Partially correct (F1, F2) | `run.go` l.1596-1665 |

## Findings

### Fix 1 - correct
ming uses `claude-sonnet-5`, phil uses `gpt-5.3-codex`; both non-empty, so `hma host decision` no longer rejects "route missing model". `withModelFlag` (l.446) is used at all three dispatch sites: preflight (l.906), worker (l.1386), reviewer (l.1913). Each passes the same `cfg.Model` that the route record binds, so the receipt model can match `Route.Model`. Value validity against the real tool is unverified by me (see top).

### Fix 2 - correct; I re-derived both checks
- `ReviewLaunchMatches` (`review_import.go` l.55-60) requires `l.Model == d.Route.Model` and `l.Provider == d.Route.Provider`, exactly.
- `EvaluateDispatch` (`host.go` ~l.1019-1036): `strong` is true when `IssuingReason == IssueFinal`. For every implementation contribution the review is refused if `WorkerID` or `Model` is equal, or, when strong, if `ProviderFamily` **or** `ModelFamily` is equal.
- Config: ming = `claude-sonnet-5` / `anthropic` / `claude-family`; phil = `gpt-5.3-codex` / `openai` / `gpt-family`. All five compared fields differ, so independence passes.
- Receipt equality holds only if phil's real receipt says `openai` and `gpt-5.3-codex`. That is the requester's observed evidence, not mine.
- Residual risk (inference): `Provider` is a hand-kept label. If agent-dispatch's `derive_provider` changes, import fails closed with `HOST_REVIEW_RECEIPT_MISMATCH`. It fails closed, not open.

### Fix 3 - correct
`--report-file r.reportPath()` is appended to the reviewer args only (l.1923). `reportPath()` (l.1976) is the same path `stepReviewImport` sends as `ReportPath` (l.~2005), and `review_import.go` l.192-196 compares `f.Path == in.ReportPath` plus the SHA-256. The worker needs no such flag: the worker writes no bound report, and the import code never reads `report_files` for it. Whether the real tool requires an absolute path is not verified by me; `artifactsDir` is the existing path used elsewhere.

### Fix 4 - prompt versus contract
Checked against `auditReport` (`review_import.go` l.77-83) and `ValidateAuditReport` (`report_contract.go`):

| Prompt claim | Contract | Match |
|---|---|---|
| keys `audit_id`, `target` nonempty strings | l.31-35 | yes |
| `modes` nonempty, from plan/code/reasoning/outcome | l.29, 36-47 | yes |
| `phase1_frozen` boolean `true` | boolean required l.54; `true` needed for pass l.~178 | yes |
| `limitations` array of strings, `[]` ok | l.57-66 | yes |
| `verdict` lowercase `pass`/`fix-required` | `MapReportVerdict` l.~100; others escalate | yes |
| `isolation` isolated/partial/compromised | contract l.51; import switch | yes |
| `hma_binding` three fields | `ReportBinding` | yes |
| `requirements` id + status pass/fail/unknown + `evidence_refs` required for pass/fail | l.67-92 | yes |
| finding fields, enum sets, `evidence` shape | l.100-135 | yes |
| worked example | valid JSON, `[]` violations (throwaway check, rendered for unit `U1`, criterion `C1`) | yes |

**F1 (medium) - `fix` object omitted from the finding schema.** `ValidateAuditReport` requires a `fix` object (`action`, `why`, `tradeoffs`, `alternative`, `effort`, `status`, nonempty `verification[{method,expected}]`) for every finding of kind `defect`, `risk` or `gap`. Only `opportunity` may omit it. The prompt does not mention it. Evidence: a report with a `defect` finding built exactly to the prompt returned `findings[0].fix: required for actionable finding`. Effect (`review_import.go` ~l.226-229): `contractValid` is false, so any non-approve verdict is forced to `ReviewEscalate`. A reviewer that correctly finds a defect and says `fix-required` is therefore escalated instead of mapped to `RequestChanges`. The clean-pass path is unaffected, which is why the real smoke test did not hit it. Fail-safe, but it degrades the request-changes loop. Recommend adding the `fix` schema, with a second worked example.

**F2 (low) - wording stricter than the contract.** "EXACTLY these top-level keys ... no others" is not enforced: the validator and `auditReport` ignore extra keys. The statement "exactly one entry per criterion" is also not enforced by any validator. Harmless, but the prompt claims more than the contract. The prompt also does not state finding-id rules enforced by `MapAuditFindings` (no `/`, no `HMA-` prefix, unique), and it omits that `pass` needs `isolation: isolated` and zero actionable findings (`report_contract.go` ~l.187-194). The last point is conveyed only loosely by "no blocking finding".

**F3 (info) - no unit test pins the prompt.** `grep` finds no test calling `reviewTaskText` or asserting `--report-file`, `claude-sonnet-5` or `gpt-5.3-codex` in `cmd/hmarun/*_test.go`. The fake binary cannot catch regressions here. Recommend a test that feeds the rendered example to `hoststate.ValidateAuditReport`, and one that asserts the reviewer args contain `--report-file`.

## Assumptions and open items
- Assumption: the requester's receipt evidence (model/provider values, `report_files` behaviour) is accurate.
- Open: real-tool contract files unread (permission denied).
- Open: F1 unresolved.

**Next action:** human decides whether to fix F1 (add the `fix` schema to `reviewTaskText`) before the next real run; no other fix blocks.
