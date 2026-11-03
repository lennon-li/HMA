# Independent audit: `cmd/hmarun` ledger isolation fix (Phil, 2026-11-03)

**Verdict: PASS.** I independently reproduced the failure mode and the fix behavior, and the current wiring isolates unreconciled real dispatches from the reviewer packet namespace exactly as required.

## Verification commands

| Command | Result |
|---|---|
| `export PATH="/home/yeli/.local/go/bin:$PATH" && go build ./...` | Pass (`go version go1.27.3 linux/amd64`; command exit 0). |
| `export PATH="/home/yeli/.local/go/bin:$PATH" && go vet ./...` | Pass (no vet findings; command exit 0). |
| `export PATH="/home/yeli/.local/go/bin:$PATH" && go test ./cmd/hmarun/... -v -race` | Pass (`ok github.com/lennon-li/HMA/cmd/hmarun 17.707s`; all listed tests passed, including `TestStepReviewerDispatchAcceptedByReviewImport`). |
| `export PATH="/home/yeli/.local/go/bin:$PATH" && go test ./...` | Pass (all packages `ok`; includes `cmd/hma 117.332s`, `cmd/hmarun 16.611s`, `internal/host 1.137s`; command exit 0). |
| `go test -overlay=/tmp/hmarun-overlay.json ./cmd/hmarun -run TestStepReviewerDispatchAcceptedByReviewImport -count=1 -v` | **Expected fail reproduced** with reverted call-site behavior: `unreconciledDispatchStateDir() saw 0 run id(s), want 3 ...`; command exit 1. |
| `go test ./cmd/hmarun -run TestStepReviewerDispatchAcceptedByReviewImport -count=1 -v` | Pass restored on current source; command exit 0. |

## Findings with evidence

### F1 — Wiring correctness of isolation fix: **confirmed**

Evidence from current `cmd/hmarun/run.go`:

- `runAgentDispatch` now takes `stateDir string` and sets `AGENT_DISPATCH_STATE_DIR` from that argument (`run.go:1059+`).
- `agentDispatchStateDir()` returns `<store>/agent-dispatch-state` (`run.go:1078+`).
- `unreconciledDispatchStateDir()` returns `<store>/agent-dispatch-state-unreconciled` (`run.go:1095+`).
- Updated call sites:
  - `issuePreflight` uses `r.unreconciledDispatchStateDir()` (`run.go:909`).
  - `stepWorkerDispatch` uses `r.unreconciledDispatchStateDir()` (`run.go:1413`).
  - `stepReviewerDispatch` uses `r.agentDispatchStateDir()` (`run.go:1964`).

Only three real `runAgentDispatch` call sites exist in `cmd/hmarun` (confirmed by search), and all are correctly assigned.

### F2 — `ledgerDir()` vs reviewer dispatch namespace: **confirmed aligned**

`ledgerDir()` remains derived from `agentDispatchStateDir()` via `repoKeyFor` (`run.go:1531-1539`). `stepReviewerDispatch` is the only real dispatch that still runs with `agentDispatchStateDir()` (`run.go:1964`), and `stepReviewImport` also reads receipt/report from `ledgerDir()`. This keeps packet binding, receipt lookup, and reconcile namespace on the same physical directory.

### F3 — `stepReviewPreflight` path correctness: **confirmed through `issuePreflight`**

`stepReviewPreflight` calls `r.issuePreflight(ctx, hoststate.RoleReview, reviewer)` (`run.go:1501-1515`). That means review preflight uses the same updated `issuePreflight` path and therefore writes to `unreconciledDispatchStateDir()`, not to reviewer ledger namespace.

### F4 — Independent regression reproduction (revert → fail → restore → pass): **confirmed**

To satisfy the "do not edit `cmd/hmarun`" constraint, I used a Go build overlay (temporary `/tmp` files only) that replaced the two unreconciled call-site arguments with `r.agentDispatchStateDir()` while leaving the repository untouched:

- Overlay run failed exactly as expected:
  - `dispatch_test.go:694: unreconciledDispatchStateDir() saw 0 run id(s), want 3 ...`
- Immediate run on current source passed.

This independently validates the test catches the bug and that the present fix resolves it.

### F5 — Same-class overlap risk elsewhere in `cmd/hmarun`: **no additional instance found**

Search results show:

- No other real `runAgentDispatch` invocations beyond the three audited sites.
- No other writer of `AGENT_DISPATCH_STATE_DIR` in runtime code besides `runAgentDispatch`.
- No alternate preflight path bypassing `issuePreflight`.

I did not find another path where a non-review real dispatch can silently overlap the review packet's reconciled namespace.

## Git status check

`git --no-pager status --short --untracked-files=all` before report creation listed existing untracked `cmd/hmarun/*` and prior audit files. After this audit, status is unchanged except for this new file:

- `docs/handoffs/evidence/mvp/audit-hmarun-ledger-isolation-fix-phil.md`

No edits were made under `cmd/hmarun/`. No commit/push performed.

## Next action

Proceed to the human gate decision for this fix (accept/reject) under `docs/architecture.md` §18 using this evidence packet.
