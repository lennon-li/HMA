# Task 9 — Evaluator CLI Surface Proposal

Status: **proposal only, not implemented.** Adding a CLI command is an explicit
stop condition in the Task 8 packet, so this document requests approval; it
does not authorize the work it describes.

## 1. Problem

v1.0.0 ships one binary whose CLI exposes `pilot`, `resolve`, and `ci github`.
The deterministic evaluators HMA is built around are reachable only as Go
packages:

| Evaluator | Package | CLI |
| --- | --- | --- |
| stage transitions | `internal/engine.EvaluateTransition` | none |
| approval invalidation | `internal/engine.EvaluateInvalidation` | none |
| validator contract | `internal/engine.EvaluateValidatorContract` | none |
| route attestation | `internal/engine.EvaluateRouteVerification` | none |
| route coherence | `internal/engine.EvaluateRouteCoherence` | none |
| route policy | `internal/engine.EvaluateRoutePolicy` | none |

A host running the shipped binary therefore cannot propose a stage transition,
evaluate invalidation, check validator independence, or evaluate a route
policy. The product contract says "no evidence, no transition"; the binary
currently cannot express a transition at all. Bootstrapping a run store is only
possible through `hma pilot`.

This is a completeness gap, not a defect in the evaluators themselves: each is
tested and behaves as specified when called as a library.

## 2. Bounded scope proposed

In scope:

- one read-only `hma eval <evaluator>` command family that reads a host input
  document, calls exactly one existing pure evaluator, and prints its result;
- one `hma show` command that prints the projected state of a run store;
- host input schemas for each evaluator, mirroring the existing `pilot` and
  `resolve` input pattern (`DisallowUnknownFields`, one document per file);
- table-driven CLI tests reusing the existing engine fixtures.

Out of scope:

- any change to evaluator semantics, record schemas, or approval binding;
- writing records from `hma eval` — the evaluators are pure and must stay
  observation-only at the CLI;
- stage advancement, dispatch, publication, release, or route selection;
- new dependencies or a non-standard-library runtime.

## 3. Open decisions requiring approval

1. Whether `hma eval` may append a record for a proposed transition, or must
   remain strictly read-only with the host doing any writing. Read-only is the
   safer default and is what this proposal recommends.
2. Whether a `hma init` command should exist to bootstrap a run store, or
   whether `pilot` remains the only entry point.
3. Whether `hma show` prints the resolution projection only, or a fuller run
   summary including terminal-outcome derivation.
4. Output contract: one JSON document per invocation on stdout, non-zero exit
   on `REJECTED`, matching the existing commands' behaviour.

## 4. Verification gates

Unchanged from Task 8: `gofmt -l` empty, `go vet`, `go build`,
`go test -count=1 ./...`, `go test -race`, fixture determinism across repeated
runs, `git diff --check`, and a clean worktree. These now run in CI on every
push and pull request (`.github/workflows/ci.yml`).

## 5. Stop conditions

Stop and return for clarification if the work requires changing evaluator
semantics, record schemas, approval binding, stage authority, or the
standard-library-only constraint; or if any command would advance a stage,
grant approval, select a route, or dispatch a worker.
