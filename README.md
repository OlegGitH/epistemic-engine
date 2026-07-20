# Epistemic Protocol and Engine

**Epistemic Protocol** is the vendor-neutral interoperability contract. **Epistemic Engine** is the reference implementation for software deployment readiness. Instrument once, send to any compatible provider, use any engine, display in any UI, and gate any workflow.

The protocol core has no OpenAI, OpenTelemetry, PostgreSQL, or UI dependency. The Engine turns an agent recommendation into atomic claims, binds typed evidence, keeps contradictions and assumptions visible, runs bounded verification, applies deterministic policy, and emits an immutable Decision Certificate.

> An agent may act only when its critical claims survive evidence, contradiction, and verification.

## OpenAI Build Week 2026

Epistemic Engine is a **Developer Tools** entry built during the July 13–21, 2026 submission period. It addresses a specific operational problem: AI agents can recommend consequential changes faster than teams can establish whether the assumptions behind those recommendations are actually supported.

The Engine converts a recommendation into atomic claims, binds each claim to observable evidence, preserves contradictions and unknowns, permits only approval-bounded verification, applies deterministic policy, and emits a content-addressed Decision Certificate with a plain-language report.

### How OpenAI technology is used

- **GPT-5.6 through the Responses API** proposes a strict structured decomposition of a recommendation into 3–7 observable claims. Model output is treated as an untrusted proposal; it cannot approve an action or determine the final verdict.
- **Codex SDK** creates one targeted verification test inside a disposable repository copy after explicit human approval. Network access is disabled, edits outside `tests/` are rejected, and the worker emits an unapplied patch plus its SHA-256 digest.
- **Codex collaboration during development** accelerated the protocol and API scaffolding, conformance fixtures, PostgreSQL persistence, dashboard implementation, CI scenario matrix, GCP packaging, test generation, and UI/UX refinement. Human decisions defined the vendor-neutral boundary, deterministic policy ownership, privacy boundary, approval model, and product positioning.

The dated commit history records the Build Week implementation. The [submission packet](docs/submission/README.md) distinguishes implementation evidence, judge instructions, the demo narrative, and final external items.

### Fast judge path

Open the [live Food Lens submission dashboard](https://epistemic-dashboard-r7zqwwvzgq-ew.a.run.app/?account=acc_2574df3bf09361265b5fefcf). It contains five authenticated CI reports and proof-verified certificates covering supported release, insufficient evidence, privacy contradiction, and bounded verification. The [public API health check](https://epistemic-control-plane-r7zqwwvzgq-ew.a.run.app/health) confirms durable PostgreSQL storage.

Supported platforms are Windows, macOS, and Linux with Docker Desktop and Docker Compose.

```bash
git clone https://github.com/OlegGitH/epistemic-engine.git
cd epistemic-engine
docker compose up --build
```

Open <http://localhost:3000>, seed a scenario with `cd apps/control-plane && go run ./cmd/seed --scenario unsafe`, and inspect the printed run ID at <http://localhost:3000/run>. The standalone [Food Lens demo](https://github.com/OlegGitH/epistemic-engine-demo) exercises supported, insufficient-evidence, contradicted, and bounded-verification paths in CI.

## What is implemented

- Ordered, idempotent trace ingestion with correlation IDs and live SSE lifecycle events.
- Deterministic local analysis plus an optional OpenAI Responses API Structured Outputs analyzer.
- Requirement-level PR and text-change review that catches partial, missing, contradicted, and confidence-without-evidence outcomes.
- Typed evidence for diffs, tests, builds, migrations, logs, and traces with SHA-256 content hashes.
- Claim/evidence/contradiction graph, assumptions, unknowns, and explainable support dimensions.
- Approval-gated verification planning, recorded fallback artifacts, and a restricted Docker runner.
- A narrowly scoped Codex SDK worker that may edit only `tests/` in a disposable checkout.
- Deterministic deployment policy and content-addressed Decision Certificates.
- PostgreSQL persistence, Next.js/React Flow workspace, six golden scenarios, and an unsafe demo PR.
- A vendor-neutral GitHub pipeline generator exposed through HTTP and MCP, with configured tool evidence feeding the Epistemic quality gate.
- An account portfolio dashboard for projects, registered AI usage, knowledge health, certificate history, and evidence activity.
- Project-scoped GitHub Actions connections that publish authenticated reports and proof-verified certificates into the dashboard.
- Account-scoped PostgreSQL persistence for runs, evidence state, decisions, and certificates, plus a plain-language certificate report in the UI and CLI.

## Repository map

```text
specification/            Protocol v0.1, lifecycle, compatibility, schemas, HTTP
api/go/                   Stable vendor-neutral Go API
sdk/                      Go providers plus TypeScript and Python SDKs
cmd/epistemic/            YAML and artifact evaluation CLI
cmd/epistemic-relay/      Optional validation/redaction/archive relay
adapters/                 GitHub, Agents SDK, JUnit, SARIF, MCP, React
conformance/              Protocol/provider suites and portable fixtures
examples/                 Cross-domain examples and alternate server
agents/demo-agent/       Python Agents SDK reviewer, HITL pause/resume, trace processor
apps/control-plane/      Go Engine plus its OpenAPI contract, schemas, and examples
apps/web/                Account control center plus run-level decision debugger
db/migrations/           PostgreSQL schema and versioned policy seed
demo/                    Unsafe orders PR, corrected patch, recorded artifacts
deploy/                  Deployment entry points
docs/                    ADR, architecture, development, and implementation plan
evals/                   Six golden scenarios and expected metrics
schemas/                 Exported language-neutral schema snapshots
workers/codex-worker/    Bounded Codex SDK verification-test generator
```

## Protocol quickstart

No Docker or vendor credentials are needed:

```bash
go run ./cmd/epistemic evaluate --config examples/deployment-readiness/.epistemic.yaml
```

The example converts JUnit into a portable event, evaluates it with the Local provider, and writes a Decision Certificate. Continue with the [15-minute protocol quickstart](docs/quickstart/protocol.md).

## Start the complete local stack

Prerequisites: Docker Desktop with Compose. From the repository root:

```bash
docker compose up --build
```

This starts PostgreSQL on `5432`, the Go API on `8080`, and the workspace on `3000`. The API waits for PostgreSQL health and requires the durable repository in the Compose stack. `GET /healthz` reports `storage: "postgresql"` and `durable: true` so deployments and tests can verify that persistence is active.

Open the account control center at <http://localhost:3000> and the interactive API documentation at <http://localhost:8080/docs/>. See the [dashboard guide](docs/guides/account-dashboard.md) to register an account, projects, and AI usage. To generate and install a reusable GitHub Actions quality gate, follow the [pipeline tool guide](docs/guides/github-actions-pipeline.md).

Seed a scenario in another terminal:

```bash
cd apps/control-plane
go run ./cmd/seed --scenario unsafe
```

Open the run debugger at <http://localhost:3000/run> and paste the printed run ID. Available scenarios are `unsafe`, `pending`, and `corrected`.

Each evaluated decision exposes both representations:

- `GET /v1/decisions/DECISION_ID/certificate` returns the immutable machine certificate.
- `GET /v1/decisions/DECISION_ID/certificate/report` returns a structured human report; add `?format=markdown` to download it.

The human report explains whether to proceed, why, which critical claims were supported, remaining conditions, and the certificate digest. It is derived from the certificate and stored evidence; it does not modify the signed machine result.

If an older database volume predates the current migration, recreate it once with `docker compose down -v` before starting the stack.

## Deploy to Google Cloud

The [GCP deployment package](deploy/gcp/README.md) builds the API and dashboard in Artifact Registry, provisions Cloud SQL and Secret Manager, runs versioned migrations through a Cloud Run Job, and deploys both services to Cloud Run. It also includes keyless GitHub OIDC setup and a bootstrap script that connects the Food Lens repository pipeline to the deployed dashboard.

Deployment is manual by design because Cloud SQL and Cloud Run can create billable resources.

## Validate locally

```bash
make bootstrap
make test
make eval
```

The tests under `demo/unsafe-orders-pr` intentionally fail until `demo/corrected-orders.patch` is applied. CI asserts both the unsafe failure and the corrected pass.

## Optional OpenAI paths

The offline `rules` analyzer is the default. Set `ANALYZER_MODE=openai`, `OPENAI_API_KEY`, and `OPENAI_MODEL` to use strict Structured Outputs. The Python reviewer and Codex worker are separate, approval-gated integrations; see their local READMEs.

No hidden chain-of-thought is requested or stored. Model output is treated as a structured proposal; policy, permissions, hashes, persistence, and certificates remain application-owned.

See the [protocol specification](specification/protocol.md), [protocol versus Engine boundary](docs/concepts/protocol-vs-engine.md), [PR requirement coverage contract](docs/concepts/pr-requirement-coverage.md), [architecture](docs/architecture.md), [development](docs/development.md), the [accepted Engine ADR](docs/adr/0001-epistemic-control-plane.md), the [protocol implementation record](docs/plans/2026-07-16-epistemic-protocol-implementation-plan.md), the [Protocol implementation report](docs/reports/2026-07-16-epistemic-protocol-implementation-report.md), the [Engine implementation report](docs/reports/2026-07-16-implementation-report.md), and the [Control Center UI/UX review](docs/reports/2026-07-20-ui-ux-review.md).
