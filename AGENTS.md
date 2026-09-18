# Agent Instructions

HMA is human-gated. Repository instructions never weaken `docs/architecture.md`,
the deterministic safety kernel, approval binding, route coherence, or
independent-review requirements.

## Orchestration-first

An interactive CLI agent is an orchestrator by default. Delegate bounded coding
work unless the recorded direct-cost exception is satisfied. Before dispatch,
perform the required exact-route `hi` preflight. Report the worker, provider,
model, reasoning/effort level, access service, task, and permission envelope.

## Jev decision tool

When an approved Jev decision tool is available through the host (MCP, SDK, or
HTTP adapter), the interactive orchestrator MUST use Jev for soft, bounded
judgments at these checkpoints:

1. **Pre-dispatch:** choose among already-eligible worker routes and classify
   task complexity. Jev may rank or choose only routes that deterministic HMA
   policy has already declared eligible.
2. **Post-worker:** judge whether the returned artifact appears sufficient for
   the approved acceptance criteria, with choices such as `accept`, `retry`,
   or `escalate`.
3. **Pre-next-step:** choose among `continue`, `retry`, `replan`,
   `escalate`, or `stop` after the current evidence and findings are
   assembled.
4. **Discretionary implementation triage:** when the agent itself proposes an
   optional change not explicitly requested by the human, use Jev to classify
   it as `implement_now`, `prototype`, `defer`, or `reject` before adding
   it to the plan.

Ask narrow typed questions over explicit state. Prefer Jev `Choice`, `Noul`,
and `Score` primitives with explicit criteria and a `none`/fallback option
where the choices may be incomplete. Do not ask vague questions such as "is
this good?".

Jev is advisory intelligence, not authority:

- deterministic HMA policy runs before Jev and may remove forbidden routes or
  actions from the choice set;
- Jev never grants permission, widens a permission envelope, waives a rule,
  approves a stage transition, publishes, commits, pushes, or releases;
- Jev never replaces the mandatory independent review of coding work;
- Jev output is a decision trace, not proof that acceptance criteria passed;
- human decisions and challenge-bound approvals remain authoritative;
- if Jev is unavailable, record that condition and fall back to the
  orchestrator/human decision path. Do not silently pretend another model is
  Jev.

Until a host has calibrated thresholds from local outcomes, treat low-confidence
or close decisions as escalation signals rather than forcing an automatic
choice. The host owns thresholds; portable HMA core must not hard-code
provider-specific confidence cutoffs.

See `docs/jev-decision-tool.md` for the decision contract and examples.
