# HMA

Human-gated evidence and stage control for coding agents.

## Status

HMA is in architecture bootstrap. This repository currently defines the product contract and MVP boundaries; it does not yet ship a runnable CLI.

## Purpose

Coding agents routinely advance on unsupported claims: they implement before grounding, declare success without fresh verification, overengineer beyond the approved scope, audit forever, retry failures without learning, and avoid saying `FAILED`, `BLOCKED`, or `UNKNOWN`.

HMA is a standalone, vendor-neutral gatekeeper for Git repository coding tasks. It does not automate coding. It assembles evidence, enforces deterministic safety rules, proposes agents and models, and asks a human to authorize every stage transition.

> No evidence, no transition.

## Core contract

- Humans approve every stage transition.
- HMA does not dispatch workers in v1; after approval it emits route and permission records, and a human or host starts execution.
- Machines gather evidence, enforce unwaivable rules, detect drift, and propose one next action.
- Agent prose is not proof.
- The implementing agent cannot approve or validate its own work.
- `FAILED`, `BLOCKED`, `UNKNOWN`, `PARTIAL`, and `ABORTED` are honest supported outcomes.
- Auditing stops when approved criteria are resolved and no closure-blocking finding remains.
- Project policy may tighten the safety kernel but cannot weaken it.

## Planned shape

- One standalone Go binary
- Standard-library-first implementation
- Content-addressed JSON records and portable JSON Schema
- Host-managed, detection-based append-only run store with chained records and a per-run head anchor
- Fresh command/evidence capture bound to exact Git revisions
- Human challenge-bound, single-use approvals
- Primary agent/model route plus one explicit escalation route
- Independent semantic validation
- Local gating plus CI verification through independent validation and a release request; human release approval follows passing CI

## Documentation

- [Architecture and product contract](docs/architecture.md)

## Deliberate non-goals for v1

- Autonomous coding orchestration
- A daemon, database, TUI, or plugin framework
- Provider-specific model SDKs
- Generic support for non-Git tasks
- Silent model fallback, automatic publication, or self-approval
- A claim of product readiness before fixture and real-repository pilots pass

## Current milestone

This bootstrap commit records the architecture agreed during the design interview. Implementation, packaging, installation, and live-agent integration remain unstarted and unverified.
