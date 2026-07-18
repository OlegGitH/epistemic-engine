# Architecture

Epistemic Protocol is an isolated interoperability layer. Epistemic Engine is a modular reference implementation around a small, deterministic domain core. OpenAI, PostgreSQL, Docker, HTTP, and the browser are adapters; none owns the portable contract or final policy decision.

```text
Application instrumentation
          |
          v
 Epistemic Protocol API + SDK providers
   | remote | file | noop | local | composite
          |
          v
 Protocol HTTP facade / optional relay
          |
          v
 Epistemic Engine normalization adapter
          |
          v
 Private Engine domain, analysis, persistence, policy, UI
```

The public `api/go` module cannot import Engine packages. The Engine nested module depends on the protocol module through a local replace during development. Protocol schemas contain no Engine database entities, support-score dimensions, model configuration, or UI types.

```text
Agents SDK trace / CI artifacts / repository diff
                       |
                       v
          ordered + idempotent ingestion
                       |
             +---------+----------+
             |                    |
             v                    v
   claim decomposition      typed evidence collector
   + assumptions/unknowns   + SHA-256 content hashes
             |                    |
             +---------+----------+
                       v
          claim/evidence relation graph
                       |
                       v
       bounded verification planner --approval--> Codex test patch
                       |                              |
                       +--------- Docker sandbox <---+
                                      |
                                      v
                           verified_by relation
                                      |
                                      v
                    deterministic policy evaluation
                                      |
                                      v
                 immutable certificate + SHA-256 proof
```

## Trust boundaries

1. Imported trace content and artifacts are untrusted observations.
2. Structured model output proposes claims; it cannot externally verify a critical claim.
3. Support is a transparent evidence-quality aggregate, never a probability of truth.
4. Critical contradictions, absent evidence, and unresolved critical unknowns are hard gates.
5. Every verification needs explicit approval and runs only in `sandbox` or `staging`.
6. Docker execution uses an image allowlist, no network, read-only repository mount, CPU/memory/PID limits, and a deadline.
7. Codex works in a disposable copy, has no network, can change only `tests/`, and outputs an unapplied patch.
8. Consequential action still requires a separate human approval during policy evaluation.

## Runtime components

- `internal/domain`: versioned JSON vocabulary and invariants.
- `internal/analysis`: deterministic rules and strict Responses API adapter.
- `internal/store`: in-memory test adapter and transactional pgx repository.
- `internal/verification`: bounded plans and restricted Docker execution.
- `internal/policy`: pure, deterministic hard gates.
- `internal/service`: lifecycle orchestration and reproducible proof hashing.
- `internal/httpapi`: validated HTTP routes, structured logs, correlation IDs, and SSE.
- `agents/demo-agent`: Agents SDK tools, resumable HITL state, and privacy-safe tracing.
- `workers/codex-worker`: approved targeted-test generation through the Codex SDK.

## Persistence and immutability

PostgreSQL stores first-class runs, decisions, claims, evidence, relations, assumptions, unknowns, verifications, policies, and proofs. Event ordering is serialized by locking the owning run; `(run_id, sequence)` and producer `external_id` are unique. Evidence and verification artifacts are SHA-256 addressed. A proof is unique per decision and evaluation returns the stored certificate on retries.

SSE is an operational convenience, not the source of truth: clients receive a durable graph snapshot first, followed by in-process lifecycle notifications.
