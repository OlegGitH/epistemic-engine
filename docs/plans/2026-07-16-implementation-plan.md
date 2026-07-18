# Epistemic Control Plane — implementation record

- Plan date: 16 July 2026
- Source ADR: `docs/adr/0001-epistemic-control-plane.md`
- Internal freeze: 21 July 2026, 18:00 CEST
- Official deadline: 21 July 2026, 17:00 PDT / 22 July 2026, 02:00 CEST

## Objective

Ship one vertical slice that turns a real agent run into atomic claims, typed evidence, visible contradictions and assumptions, an executable verification plan, sandbox results, a deterministic policy decision, and an immutable Decision Certificate.

Critical path:

```text
unsafe PR → Agents SDK trace → Go ingestion → claim graph → evidence gaps
→ Codex check → Docker result → policy → certificate
```

## Delivery status

“Implemented” means the capability is present in this repository. “Release step” requires a person or external service and cannot be completed by repository code alone.

| ID | Priority | Deliverable | Status | Repository evidence |
|---|---:|---|---|---|
| FND-01 | P0 | Monorepo and developer environment | Implemented | Compose, Makefile, service Dockerfiles |
| FND-02 | P0 | Versioned domain types and JSON schemas | Implemented | `internal/domain`, `apps/control-plane/api/schemas`, `schemas` |
| FND-03 | P0 | PostgreSQL migration and repository | Implemented | migration and transactional pgx adapter |
| FND-04 | P0 | Unsafe demo pull request | Implemented | compatibility and PII defects plus corrected patch |
| FND-05 | P0 | Ordered/idempotent event ingestion | Implemented | event constraints, row locking, duplicate identity |
| FND-06 | P0 | Agents SDK reviewer and trace exporter | Implemented | Python agent, trace processor, HITL state |
| FND-07 | P0 | Seeded claim graph | Implemented | seed CLI and React Flow workspace |
| ANA-01 | P0 | Strict Structured Output client | Implemented | `openai-go` Responses adapter with refusal/error handling |
| ANA-02 | P0 | 3–7 atomic claims | Implemented | rules and OpenAI analyzers |
| ANA-03 | P0 | Typed evidence collectors and hashes | Implemented | diff/test/migration/log/build/trace normalization |
| ANA-04 | P0 | Evidence-to-claim binding | Implemented | typed relations and claim evidence IDs |
| ANA-05 | P0 | Contradiction and scope analysis | Implemented | explicit contradiction edges, assumptions, unknowns |
| ANA-06 | P1 | Six-scenario golden dataset | Implemented | `evals/fixtures/scenarios.json` |
| VER-01 | P0 | Bounded verification planner | Implemented | targeted compatibility/PII checks; human fallback |
| VER-02 | P0 | Codex test-generation worker | Implemented | disposable, test-only, network-disabled SDK worker |
| VER-03 | P0 | Isolated Docker runner | Implemented | allowlist, no network, limits, read-only mounts, timeout |
| VER-04 | P0 | Results mapped to claims | Implemented | state update and `verified_by` relation |
| VER-05 | P0 | Explicit approval gate | Implemented | API and Agents SDK tool approval gates |
| VER-06 | P1 | Deterministic fallback fixtures | Implemented | recorded failure artifacts and recorded execution mode |
| DEC-01 | P0 | Deterministic policy engine | Implemented | hard claim/unknown gates and human action approval |
| DEC-02 | P0 | Explainable support scoring | Implemented | seven named dimensions and contradiction burden |
| DEC-03 | P0 | Immutable Decision Certificate | Implemented | policy/artifact hashes and reproducible SHA-256 proof |
| DEC-04 | P0 | Lifecycle API | Implemented | graph, plan, execute, evaluate, certificate, SSE |
| DEC-05 | P0 | Policy/proof/idempotency tests | Implemented | Go unit and HTTP lifecycle tests |
| DEC-06 | P1 | Structured logs and correlation IDs | Implemented | request JSON logs, trace/run/certificate identifiers |
| UX-01 | P0 | Single-screen decision workspace | Implemented | timeline, graph, inspector, decision state |
| UX-02 | P0 | Claim/contradiction interactions | Implemented | selectable claims and support/gap details |
| UX-03 | P0 | Verification console and progress | Implemented | approval, outcome, hash, SSE refresh |
| UX-04 | P0 | Decision Certificate view | Implemented | distinct certificate and JSON export |
| UX-05 | P0 | Before/after mode | Implemented | two-run comparison |
| UX-06 | P1 | Golden metrics | Implemented | eval CLI and expected metrics |
| SHP-01 | P0 | Clean-machine end-to-end run | Ready; environment verification pending | Compose/CI path implemented; requires running Docker daemon |
| SHP-02 | P0 | Bound latency and fallback | Implemented | request/runner timeouts, idempotence, recorded mode |
| SHP-03 | P0 | README, architecture, setup | Implemented | root README and `docs/` |
| SHP-04 | P0 | Record sub-three-minute demo | Release step | storyboard below; recording requires a human |
| SHP-05 | P0 | Submit to Devpost | Release step | requires account access and final project URL/video |
| SHP-06 | P1 | Final visual/copy polish | Implemented baseline | responsive single-screen workspace |

## Demo storyboard (under three minutes)

1. Seed `unsafe`; show the deployment recommendation decomposed into five claims.
2. Select compatibility and privacy claims; show failed migration evidence and the PII contradiction.
3. Evaluate to show `CONTRADICTED` and blocked action.
4. Seed `pending`; plan two bounded checks and show explicit per-check approvals.
5. Record or run passing compatibility/PII checks; show `verified_by` edges and artifact hashes.
6. Approve the consequential action, evaluate, and export `VERIFIED_WITH_CONDITIONS` certificate.
7. Compare unsafe and corrected run IDs in before/after mode.

## Release gates

- Gate A: trace ingestion, durable seed graph, and reproducible unsafe PR.
- Gate B: critical claim extraction, visible contradictions, Codex check artifact, Docker runner.
- Gate C: blocked initial decision, corrected verified decision, certificate in one-screen UI.

All three gates are represented in code. Gate execution involving Docker/OpenAI depends on the corresponding local credentials and daemon.

## Metrics

The eval command reports scenario count, critical-claim recall, contradiction recall, and decision accuracy. The API and tests additionally cover verification-plan executability, initial/corrected decision behavior, certificate reproducibility, and idempotent reprocessing.

## Application-owned controls

Policy outcomes, support computation, hashes, permissions, sandbox limits, approval, persistence, and certificates are application-owned. OpenAI integrations propose structured claims, run the demo agent, or generate a bounded test patch; they do not own authorization or the final verdict.
