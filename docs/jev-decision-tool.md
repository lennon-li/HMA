# Jev as an HMA decision tool

Status: host-integration guidance

Jev is the preferred default **soft-decision provider** for Lennon's interactive
HMA workflow. It belongs to the host-integration layer, not the portable HMA
core. HMA remains vendor-neutral and deterministic where authority or safety is
involved.

TypeSafe's System One API accepts one structured state plus focused typed
questions. A `Choice` returns one option with probabilities/confidence, a
`Noul` returns a yes/no probability, and a `Score` returns a position on an
ordered rubric. That shape fits routing and gate recommendations without asking
a general-purpose reasoning model to spend tokens on every micro-decision.

Official SDKs:
- JavaScript/TypeScript: `@typesafe-ai/sdk`
- Python: `typesafe-sdk`

Credentials such as `TYPESAFE_API_KEY` are host-local secrets and MUST NOT be
written to portable HMA records or committed to the repository.

## Authority order

For any decision:

1. deterministic safety and route-policy rules;
2. explicit human instructions and HMA approval bindings;
3. Jev advisory judgment over the remaining legal choices;
4. orchestrator adjudication when the Jev result is unavailable, ambiguous, or
   below the host's configured confidence policy.

Jev never expands the legal choice set. If deterministic policy says a route is
ineligible, it is not presented to Jev.

## Required checkpoints

### 1. Route decision

State should include the bounded task, required capabilities, risk class,
permission envelope, eligible runtime profiles, current availability/preflight
facts, and relevant cost/latency information.

Ask separately:

- Which eligible worker is the best fit?
- How difficult is the bounded task?
- Is specialist/deeper reasoning materially required?

Example shape:

```json
{
  "state": {
    "task": "Fix failing CI after Python-version expansion",
    "risk": "normal",
    "eligible_workers": {
      "jax": "strong coding/debugging",
      "wei": "fast general implementation",
      "phil": "deep architecture/review"
    }
  },
  "questions": {
    "worker": {
      "type": "choice",
      "instructions": "Which eligible worker best fits this bounded task?",
      "criteria": {
        "jax": "General coding and debugging",
        "wei": "Fast bounded implementation",
        "phil": "Architecture-heavy or difficult reasoning",
        "none": "No listed worker is a clear fit"
      }
    },
    "complexity": {
      "type": "score",
      "instructions": "How difficult is the implementation?",
      "criteria": ["small/bounded", "moderate", "hard/high-uncertainty"]
    }
  }
}
```

The host applies the returned probabilities/confidence to its configured
thresholds. A weak or close result escalates to the orchestrator/human rather
than forcing a route.

### 2. Result-sufficiency decision

After a worker returns, provide the immutable request, acceptance criteria,
bounded task, changed artifact/diff summary, test evidence, known findings, and
declared omissions.

Ask a finite choice such as:

- `accept`: appears sufficient to proceed to the normal verification/review
  path;
- `retry`: same worker should correct a bounded deficiency;
- `escalate`: another worker, reviewer, or human should inspect it;
- `unknown`: state is insufficient to judge.

An `accept` recommendation does **not** mark any HMA criterion passed and does
not replace independent review.

### 3. Next-step decision

After each bounded unit, ask Jev to choose among only currently legal next
actions, for example:

- `continue`
- `retry`
- `replan`
- `escalate`
- `stop`

Deterministic transition legality is evaluated first. Jev may recommend one of
the legal options; only the HMA/human authority model can authorize a stage
transition.

### 4. Discretionary idea triage

Use this only for changes proposed by the agents themselves, not to veto a
human's explicit request.

Provide expected benefit, implementation effort, maintenance burden, overlap,
risk, and project priority. Ask:

- `implement_now`
- `prototype`
- `defer`
- `reject`

This is intended to reduce scope creep and unnecessary architecture.

## Decision trace

A host integration should retain a compact decision trace containing:

- decision-provider identity (`jev`);
- actual model identifier returned by the provider when available;
- timestamp;
- digest of the state supplied to Jev;
- question identifiers and allowed choices/rubrics;
- selected answer(s), probabilities/confidence, and any score;
- configured threshold/policy version;
- resulting host action;
- whether the orchestrator or human overrode the recommendation and why.

Do not put credentials, raw private runtime profiles, secrets, or unnecessary
user data into the trace.

## Failure behavior

Jev is optional infrastructure, not a single point of failure.

If Jev is missing, unavailable, rate-limited, or returns an unusable/ambiguous
answer:

- record `JEV_UNAVAILABLE` or `JEV_INDETERMINATE`;
- preserve the deterministic legal/illegal decision already made by HMA;
- return the soft judgment to the orchestrator or human;
- do not silently substitute another decision model under the Jev identity;
- do not weaken review, safety, permission, or approval requirements.

## Integration boundary

Portable HMA core should expose a provider-neutral decision contract. A host
adapter may bind that contract to Jev through an approved MCP server, the
official SDK, or the official HTTP API. Provider-specific credentials,
endpoints, retries, quota handling, and local aliases stay outside the portable
core.
