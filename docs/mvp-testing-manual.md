# HMA MVP controlled testing manual

This manual exercises `hma` (the run engine) and `hmarun` (the operator CLI) in a disposable Git repository. It does not require model accounts, use live dispatch, or authorize a release. Use the two test flags together only in this fixture: `--stub-dispatch` substitutes local test workers and reviewers, and `--auto-approve` supplies synthetic approvals. **Do not omit `--stub-dispatch`: real dispatch is the default.** Do not use either flag to test a live run or a real approval.

The expected result is `PARTIAL`: the first criterion passes, and the second remains pending. This is a terminal test result, not verified success. The test does not qualify live dispatch, verified resume, or finalization for production.

## 1. Build two matching binaries

Start in a checkout of the HMA revision you plan to test. Use a POSIX shell on Linux, Git, and the Go version required by `go.mod`. Keep this shell open for the remaining steps. These commands build a separate copy of each binary and do not replace anything on your `PATH`.

```sh
scratch=$(mktemp -d "${TMPDIR:-/tmp}/hma-mvp.XXXXXXXX")
mkdir "$scratch/bin" "$scratch/repo" "$scratch/store"
git rev-parse HEAD > "$scratch/source-revision.txt"
git status --short > "$scratch/source-status.txt"
go build -o "$scratch/bin/hma" ./cmd/hma
go build -o "$scratch/bin/hmarun" ./cmd/hmarun
printf 'Test files: %s\n' "$scratch"
```

Stop if either build fails. Use those two binaries together. Save the source status when reporting a result, since a commit ID alone does not identify a build made from a modified checkout.

## 2. Create the fixture and its plan

The next commands commit a seed file in the **disposable repository only**. The stub worker later makes a second local commit there. No remote is configured for this repository.

```sh
git -C "$scratch/repo" init -q
printf 'one\n' > "$scratch/repo/a"
git -C "$scratch/repo" add -- a
git -C "$scratch/repo" -c user.name='HMA test fixture' \
  -c user.email=fixture@example.invalid commit -qm 'fixture base'

cat > "$scratch/answers.json" <<'JSON'
{
  "run_id": "mvp-smoke",
  "intent": "Test HMA with local stubs",
  "risk": "low",
  "non_goals": ["No production action"],
  "criteria": [
    {"id":"C1","statement":"Fixture check passes","threshold":"exit 0",
     "check":{"executable":"true","argv":[]}},
    {"id":"C2","statement":"Second criterion stays pending","threshold":"exit 0",
     "check":{"executable":"true","argv":[]}}
  ],
  "units": [
    {"id":"U1","allowed_files":["a"],"forbidden_files":["docs"],
     "criteria_advanced":["C1"]},
    {"id":"U2","allowed_files":["b"],"forbidden_files":["docs"],
     "criteria_advanced":["C2"]}
  ]
}
JSON

"$scratch/bin/hmarun" init --answers "$scratch/answers.json" \
  --out-dir "$scratch/plan"
```

`init` writes `contract.json` and `plan.json`. Use `U1` only for this test. `U2` is present to give the pending criterion its own planned unit.

## 3. Run, inspect, and repeat

```sh
run_smoke() {
  "$scratch/bin/hmarun" run --stub-dispatch --auto-approve \
    --hma-bin "$scratch/bin/hma" \
    --store "$scratch/store" --repo "$scratch/repo" \
    --run_id mvp-smoke --unit U1 \
    --contract "$scratch/plan/contract.json" \
    --plan "$scratch/plan/plan.json"
}

run_smoke > "$scratch/first.log" 2>&1

grep 'reached terminal PARTIAL' "$scratch/first.log"
"$scratch/bin/hma" show --store "$scratch/store" --run_id mvp-smoke
```

Stop if `run_smoke` fails or the expected line is missing. Read `first.log` before retrying. A successful run reports `PARTIAL` and leaves the fixture commit on the repository's current branch. To check a completed retry, repeat the same command against the same run ID, store, repository, contract and plan:

```sh
before=$(git -C "$scratch/repo" rev-parse HEAD)
run_smoke > "$scratch/retry.log" 2>&1
test "$(git -C "$scratch/repo" rev-parse HEAD)" = "$before"
"$scratch/bin/hma" show --store "$scratch/store" --run_id mvp-smoke
```

The repeat should exit successfully, keep the same Git HEAD and still show `PARTIAL`. To test from a fresh start, create a new scratch directory and run the manual again. Do not delete records, edit the plan for an existing run, or change the run ID to get around a refusal.

## If the test fails

| Symptom | Action |
|---|---|
| Build or `init` fails | Check the Go toolchain and source revision. If answers validation failed, use the fixture exactly as shown. |
| The run stops or reports a different outcome | Keep `first.log`, the plan and contract, the Git HEAD, and the store. Check that you selected `U1` and kept both test flags. |
| The run attempts real dispatch or touches a real repository | Stop immediately. Check which binaries and paths you used. Do not rerun against that store or repository. |
| The retry fails or HEAD changes | Keep both logs, the store and the fixture repository. Report the exact paths and source revision. |

When reporting results, include the source revision and status, Go version, exit status and observed outcome. Review logs before sharing them. No production setup or deployment is part of this manual.

For command reference, see the [MVP usage guide](mvp-usage-guide.md).
