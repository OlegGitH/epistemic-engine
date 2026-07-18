# ADR-0001: Epistemic Control Plane — Product and Architecture Decisions

- **Date:** 15 July 2026
- **Status:** Accepted
- **Hackathon product:** Epistemic Debugger
- **Long-term category:** Epistemic Control Plane
- **North-star contract:** An agent may act only when its critical claims survive evidence, contradiction, and verification.

## 1. Locked product decisions

| ID | Decision | Choice |
|---|---|---|
| ADR-001 | First domain | Software deployment readiness |
| ADR-002 | Primary user | AI/platform engineer deploying code-changing agents |
| ADR-003 | Primary decision | “Is this change sufficiently verified to deploy?” |
| ADR-004 | Boundary | Ingest traces; do not build another tracing product |
| ADR-005 | Reasoning data | Store observable claims, evidence and explicit justifications; never claim access to hidden chain-of-thought |
| ADR-006 | Confidence | Use an explainable support score, not a model-generated probability of truth |
| ADR-007 | Backend | Go |
| ADR-008 | Agent demo | OpenAI Agents SDK in Python with a thin trace exporter |
| ADR-009 | Model API | OpenAI Responses API through the official `openai-go` SDK |
| ADR-010 | Storage | PostgreSQL with JSONB and relational graph edges |
| ADR-011 | Frontend | Next.js and React Flow |
| ADR-012 | Execution safety | Sandbox or staging only; human approval required before any consequential action |

## 2. MVP user journey

1. A code-changing agent reviews a pull request.
2. The agent recommends: “Safe to deploy.”
3. The control plane imports the trace and repository artifacts.
4. GPT-5.6 decomposes the recommendation into atomic claims.
5. Evidence is attached to each claim.
6. Contradictions, assumptions and unknowns are made explicit.
7. The verification planner selects the smallest critical missing checks.
8. Codex creates or modifies targeted tests.
9. Tests execute in a controlled environment.
10. Policy returns `VERIFIED`, `VERIFIED_WITH_CONDITIONS`, `INSUFFICIENT_EVIDENCE`, `CONTRADICTED`, or `REJECTED`.
11. The system emits an immutable Decision Certificate.

## 3. First-class objects

- Run
- Decision
- Claim
- Evidence
- Relation
- Assumption
- Unknown
- Verification
- Policy
- Proof

### Claim relation types

- `supports`
- `contradicts`
- `qualifies`
- `supersedes`
- `derived_from`
- `verified_by`

### Claim states

- `supported`
- `partially_supported`
- `unsupported`
- `contradicted`
- `stale`
- `verification_pending`
- `externally_verified`
- `rejected`

## 4. Evidence support semantics

The system must not present a support score as “probability the claim is true.”

The explainable support score combines evidence coverage, source quality, source independence, freshness, scope match, direct verification strength, and contradiction burden.

Critical claims with unresolved contradictions or missing required verification block the action regardless of aggregate score.

## 5. Technical architecture

### Experience layer

- Next.js
- React Flow
- Evidence viewer
- Decision timeline
- Decision Certificate

### Go control plane

- Ingestion API
- Claim/evidence service
- Policy engine
- Verification orchestrator
- SSE event stream

### OpenAI integration

- Official `openai-go`
- Responses API
- Structured Outputs
- GPT-5.6 for claim decomposition and verification planning
- Codex for targeted code/test changes
- Python Agents SDK for the hackathon agent runtime and trace export

### Persistence

- PostgreSQL
- JSONB for raw artifacts
- Relational graph edges
- SHA-256 hashes for proof integrity

## 6. Minimal API

```text
POST /v1/runs
POST /v1/runs/{id}/events
POST /v1/runs/{id}/analyze
GET  /v1/runs/{id}/graph

POST /v1/decisions/{id}/plan
POST /v1/verifications/{id}/execute
POST /v1/decisions/{id}/evaluate
GET  /v1/decisions/{id}/certificate
```

## 7. Six-day implementation sequence

### 15 July

- Freeze ADRs.
- Create monorepo.
- Create demo repository and intentional failure.
- Finalize JSON schemas.

### 16 July

- Go HTTP service.
- PostgreSQL schema.
- Trace ingestion.
- Seed run.

### 17 July

- Claim extraction using structured output.
- Evidence ingestion and binding.
- Claim graph endpoint.

### 18 July

- Contradiction analysis.
- Assumption and unknown extraction.
- Verification planning.

### 19 July

- Codex-driven targeted test generation.
- Sandbox test execution.
- Policy engine and decision states.

### 20 July

- Next.js graph.
- Evidence viewer.
- Run timeline.
- Decision Certificate.

### 21 July

- Integration fixes.
- UI polish.
- Deploy.
- Record sub-three-minute demo.
- Submit before the deadline.

## 8. Definition of done

- A real agent trace is imported.
- A recommendation becomes at least four atomic claims.
- Claims display evidence, scope and state.
- A contradiction remains visible.
- A missing critical check is planned.
- Codex produces a targeted verification.
- Verification runs and produces an artifact.
- The initial decision is blocked.
- The corrected decision is allowed with human approval.
- The certificate is reproducible from stored artifacts and hashes.
- The story is understandable in under three minutes.

## 9. Explicit non-goals

- Hidden chain-of-thought inspection.
- Generic observability platform.
- Universal fact checker.
- Universal probability-of-truth score.
- Full enterprise governance.
- Multi-domain ontology.
- Automatic real production deployment.
- Neo4j in the MVP.
- Billing and multi-tenant administration.
