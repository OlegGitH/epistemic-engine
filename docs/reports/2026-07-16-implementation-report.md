# Epistemic Control Plane — implementation report

**Report date:** 16 July 2026
**Product:** Epistemic Debugger
**Long-term category:** Epistemic Control Plane
**Implementation status:** Repository-deliverable P0/P1 scope complete
**Source decisions:** ADR-0001 and the 16 July 2026 implementation plan

## 1. Executive summary

Epistemic Engine now implements an end-to-end deployment-readiness control plane. The product accepts an agent recommendation and observable run events, decomposes the recommendation into atomic claims, binds typed and content-addressed evidence, exposes contradictions and missing evidence, proposes bounded verification, applies deterministic policy, and produces an immutable Decision Certificate.

The north-star contract is enforced throughout the implementation:

> An agent may act only when its critical claims survive evidence, contradiction, and verification.

The completed vertical slice covers the full critical path:

```text
unsafe code change → agent trace → event ingestion → claim graph
→ evidence gaps and contradictions → approved verification
→ sandbox or recorded result → policy decision → certificate
```

All Go tests, production TypeScript builds, schema checks, golden evaluations, and the live HTTP lifecycle pass. The six-scenario evaluation reports perfect results against the current deterministic fixture set. Live Docker and PostgreSQL integration on this machine remains an environment check because Docker Desktop was not running; the migration and integration test are included in CI.

## 2. Delivered product capabilities

### Trace and event ingestion

- Creates a run and associated deployment decision.
- Accepts model, tool, guardrail, CI, repository, and custom observations.
- Preserves producer event IDs and correlation IDs.
- Assigns ordered sequences transactionally when the producer omits one.
- Makes retries idempotent through unique external event identities.
- Publishes lifecycle updates through server-sent events.
- Emits structured request logs connected by correlation and run identifiers.

### Claim and evidence analysis

- Produces five deterministic deployment claims covering build, tests, compatibility, privacy, and rollback readiness.
- Supports an optional OpenAI Responses API analyzer with a strict 3–7 claim schema.
- Rejects incomplete responses, refusals, invalid JSON, invalid states, missing critical claims, and model timeouts.
- Prevents model output from declaring a critical claim externally verified.
- Normalizes build results, test results, migrations, diffs, logs, traces, and custom artifacts into typed evidence.
- Computes SHA-256 content hashes for evidence artifacts.
- Binds evidence to claims through explicit `supports`, `contradicts`, and related graph edges.
- Keeps assumptions and critical unknowns as first-class objects instead of flattening them into prose.

### Explainable support scoring

Each claim exposes an evidence-support score made from:

- evidence coverage;
- source quality;
- source independence;
- freshness;
- scope match;
- direct verification strength;
- contradiction burden.

The score is explicitly described as evidence support, not a probability that a claim is true. Hard policy gates override aggregate scoring.

### Verification planning and execution

- Creates the smallest targeted check for every unresolved critical claim.
- Produces executable compatibility and PII test specifications for the demo repository.
- Falls back to a bounded human verification when no safe executable mapping exists.
- Requires explicit approval before every verification execution.
- Supports deterministic recorded artifacts for reliable demos.
- Includes a Docker runner with:
  - an image allowlist;
  - no network;
  - read-only repository mounts;
  - read-only container filesystem;
  - CPU, memory, and PID limits;
  - execution deadlines;
  - bounded stdout and stderr capture.
- Maps passed and failed results back to their claims.
- Adds a `verified_by` graph relation and hashes the result artifact.

### Codex integration

The TypeScript Codex worker:

- requires an explicit `--approved` flag;
- copies the repository into a disposable workspace;
- disables network access and web search;
- receives one bounded claim and verification specification;
- permits modifications only under `tests/`;
- rejects changes to product code, dependencies, configuration, or CI;
- returns an unapplied patch and its SHA-256 hash;
- deletes its disposable workspace after completion.

### Agents SDK integration

The Python reviewer agent:

- uses the OpenAI Agents SDK for the demo workflow;
- exports observable trace and span metadata through a custom processor;
- excludes prompt, generation, and tool contents from tracing;
- disables sensitive trace data;
- exposes an approval-gated targeted-patch tool;
- serializes interrupted runs for human review;
- supports approval, rejection, and resumable execution;
- never represents a patch request as an applied code change.

### Policy and Decision Certificates

The deterministic deployment policy returns one of:

- `VERIFIED`;
- `VERIFIED_WITH_CONDITIONS`;
- `INSUFFICIENT_EVIDENCE`;
- `CONTRADICTED`;
- `REJECTED`.

Critical contradictions, unsupported critical claims, missing verification, unresolved critical unknowns, and absent analysis block the action. Even a verified decision requires separate human approval before `action_allowed` becomes true.

The immutable Decision Certificate contains:

- decision and run identities;
- recommendation and verdict;
- action and approval state;
- policy version;
- conditions;
- claim summaries and support details;
- verification results;
- sorted evidence and verification artifact hashes;
- issuance timestamp;
- reproducible SHA-256 proof.

Certificate creation is idempotent: a repeated evaluation returns the previously stored certificate for the decision.

## 3. User experience

The Next.js workspace provides a single-screen decision view containing:

- deployment recommendation and current verdict;
- workflow controls for analysis, planning, verification, and certification;
- run, claim, verification, evidence, and open-gate metrics;
- an interactive React Flow claim graph;
- selectable claim and evidence inspection;
- explainable scoring dimensions;
- explicit required-evidence and contradiction details;
- an approval-gated verification console;
- live lifecycle refresh through SSE;
- a visually distinct Decision Certificate;
- certificate JSON export;
- event timeline and unresolved unknowns;
- before-and-after comparison using two run IDs.

The frontend communicates exclusively through the public API and does not access PostgreSQL directly.

## 4. Persistence and API

### PostgreSQL model

The durable schema includes:

- `runs`;
- `run_events`;
- `decisions`;
- `claims`;
- `evidence`;
- `claim_evidence`;
- `relations`;
- `assumptions`;
- `unknowns`;
- `verifications`;
- `policies`;
- `proofs`.

JSONB stores source artifacts, specifications, support dimensions, policy definitions, and certificates. Relational tables retain stable graph identities and edges. The pgx repository implements transactional run creation, ordered event insertion, analysis persistence, verification updates, policy decisions, and proof retrieval.

### Public lifecycle API

| Method | Route | Responsibility |
|---|---|---|
| `POST` | `/v1/runs` | Create a run and decision |
| `POST` | `/v1/runs/{id}/events` | Append an idempotent observation |
| `POST` | `/v1/runs/{id}/analyze` | Build the epistemic graph |
| `GET` | `/v1/runs/{id}/graph` | Retrieve graph and lifecycle state |
| `GET` | `/v1/runs/{id}/events/stream` | Stream lifecycle updates |
| `POST` | `/v1/decisions/{id}/verification-plan` | Plan bounded checks |
| `POST` | `/v1/verifications/{id}/execute` | Execute or record an approved check |
| `POST` | `/v1/decisions/{id}/evaluate` | Apply policy and create a certificate |
| `GET` | `/v1/decisions/{id}/certificate` | Retrieve the immutable proof |

The complete OpenAPI document, reusable JSON Schemas, and request examples are stored under `apps/control-plane/api/`. Release snapshots are exported under `schemas/`.

## 5. Demo implementation

The canonical unsafe orders change contains two intentional defects:

1. A renamed status enum no longer accepts the persisted legacy `processing` value.
2. The order service writes a customer email address into application logs.

The repository includes tests that expose both problems, recorded failure artifacts, a mechanically applicable corrected patch, and three seed scenarios:

- `unsafe`: compatibility and PII claims are contradicted;
- `pending`: build and unit tests pass while compatibility and privacy require verification;
- `corrected`: all primary observations pass.

The intended demonstration begins with a blocked unsafe decision, verifies a pending corrected run, issues a `VERIFIED_WITH_CONDITIONS` certificate after approval, and compares the two runs side by side.

## 6. Verification evidence

| Check | Result |
|---|---|
| Go unit and HTTP lifecycle tests | Passed |
| Go static analysis with `go vet` | Passed |
| Golden scenarios | 6/6 passed |
| Critical-claim recall | 1.00 |
| Contradiction recall | 1.00 |
| Decision accuracy | 1.00 |
| Certificate proof reproduction | Passed |
| Event reprocessing idempotence | Passed |
| Analysis reprocessing idempotence | Passed |
| Verification approval enforcement | Passed |
| Docker allowlist and path-containment tests | Passed |
| Next.js production build and type checking | Passed |
| Codex worker TypeScript build | Passed |
| Agents SDK imports and current API signatures | Passed |
| Unsafe fixture failure | Reproduced as intended |
| Corrected patch application | Passed |
| Corrected fixture tests | Passed |
| JSON documents and schema synchronization | Passed |
| OpenAPI YAML parsing | Passed |
| Root and deployment Compose configuration | Passed |
| Live HTTP pending-to-certified workflow | Passed |
| Rendered Next.js response | HTTP 200 with expected workspace content |

The live workflow produced:

- two planned and approved checks;
- two externally verified critical claims;
- `VERIFIED_WITH_CONDITIONS`;
- `action_allowed: true` after human approval;
- a reproducible SHA-256 certificate proof.

## 7. Safety assessment

The implementation preserves the ADR safety boundary:

- It does not request or store hidden chain-of-thought.
- Imported content and model output are treated as untrusted proposals.
- Models cannot authorize actions or create externally verified claim states.
- The control plane never performs a real production deployment.
- Verification is restricted to sandbox or staging environments.
- Every bounded check requires approval.
- Codex cannot modify product code in the worker path.
- Final consequential action requires a separate human approval.
- Content hashes and immutable certificates make the final decision auditable.

## 8. Known constraints and outstanding release work

### Environment-dependent checks

- Docker Desktop was not running during the final local review. Compose configuration passed, but the complete PostgreSQL container and Docker verification execution must still be exercised once the daemon is available.
- PostgreSQL round-trip integration coverage is present and configured in CI using a real PostgreSQL service.
- OpenAI calls require `OPENAI_API_KEY`; the OpenAI adapters were compiled and their SDK interfaces validated without spending a live model request during final verification.
- In-app browser automation was unavailable. The production frontend build, live rendered response, and complete API lifecycle were verified, but click-level visual regression automation was not captured.

### Human release steps

- Start Docker Desktop and run the clean-machine Compose path.
- Run one live Agents SDK review and one approved Codex patch generation with project credentials.
- Record the sub-three-minute product demonstration.
- Publish the repository and deployment URL.
- Complete the Devpost entry and evidence checklist before the internal freeze.

These steps require external credentials, accounts, or human presentation and are intentionally not automated by the repository.

## 9. Operating instructions

Start the complete local stack:

```bash
docker compose up --build
```

Seed the primary demo:

```bash
cd apps/control-plane
go run ./cmd/seed --scenario unsafe
go run ./cmd/seed --scenario pending
```

Run the complete local verification suite:

```bash
make bootstrap
make test
make eval
```

The web workspace is available at `http://localhost:3000` and the API health endpoint at `http://localhost:8080/healthz`.

## 10. Conclusion

The implementation satisfies the ADR definition of done at the repository level. A real trace can be ingested, a recommendation becomes an inspectable claim graph, contradictions remain visible, missing checks are planned, verification results update claim state, deterministic policy blocks or permits the action, and a reproducible Decision Certificate records the outcome.

The product is ready for the final environment run, live OpenAI demonstration, video recording, and submission workflow.
