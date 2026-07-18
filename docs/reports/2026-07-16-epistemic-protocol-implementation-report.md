# Epistemic Protocol v0.1 implementation report

**Report date:** 16 July 2026
**Implementation status:** Complete in the repository
**Release status:** Ready for organization naming, hosted CI, packaging, and tag creation

## Executive summary

The repository now contains a vendor-neutral Epistemic Protocol v0.1 and keeps the Epistemic Engine behind a reference-implementation boundary. The delivered implementation covers the full attached backlog: normative specifications, versioned schemas, portable APIs, provider SDKs, an Engine HTTP facade, CLI and artifact ingestion, adapters, an optional relay, conformance suites, examples, documentation, and release metadata.

The protocol core has no dependency on OpenTelemetry, OpenAI, the Engine's internal packages, PostgreSQL entities, or UI models. Implementations exchange portable events, decision requests/results, context, structured errors, capabilities, and reproducible SHA-256 proofs.

## Delivered product structure

| Area | Delivered implementation |
|---|---|
| Normative contract | `specification/` contains scope, lifecycle, compatibility, context propagation, HTTP/JSON, relay behavior, semantic conventions, OpenAPI, and v0.1 schemas. |
| Stable Go API | `api/go/` contains portable types, validation, canonical JSON, hashing, the Provider contract, and a small Client facade. |
| Providers | Remote HTTP, File JSONL, No-op, deterministic Local, and Composite fan-out providers are under `sdk/go/providers/`. |
| Language SDKs | TypeScript and Python SDKs provide types/validation, canonical hashing, and Remote/File/No-op providers. |
| Reference server | `apps/control-plane/` exposes all eight public endpoints and normalizes protocol data into the existing Engine service without leaking internal entities. |
| CLI | `cmd/epistemic/` reads `.epistemic.yaml`, converts supported artifacts to privacy-safe events, evaluates decisions, writes results/certificates, and implements stable exit codes. |
| Relay | `cmd/epistemic-relay/` validates, redacts, batches, retries, archives, exports remotely, and watches a content-addressed artifact folder. |
| Adapters | GitHub Action, OpenAI Agents lifecycle mapper, MCP stdio adapter, JUnit, SARIF, generic artifact, and React components are under `adapters/`. |
| Conformance | `conformance/` contains valid/invalid, duplicate, ordering, parent, evaluation, proof, provider, schema, and cross-language fixtures. |
| Examples and docs | Deployment readiness, agent tool execution, research evidence, and an alternate compatible server are reproducible from `examples/`; quickstarts, concepts, RFCs, badge, and release metadata are included. |

## Public protocol surface

The Engine reference facade implements:

- `POST /v1/events`
- `POST /v1/events:batch`
- `POST /v1/decisions:evaluate`
- `GET /v1/decisions/{id}`
- `GET /v1/decisions/{id}/events`
- `GET /v1/decisions/{id}/certificate`
- `GET /.well-known/epistemic`
- `GET /v1/stream`

It supports all 30 event names in the eight specified families, version validation, additive unknown fields, context header propagation, event and idempotency-key deduplication, parent validation, partition ordering, portable errors, synchronous allow/block evaluation, lifecycle event generation, SSE, and independently verifiable certificates.

Engine storage trace keys are scoped by portable decision ID. This permits multiple decisions to share one protocol run ID without colliding with the Engine store's unique external-trace constraint, while preserving the original portable run context on the wire.

## CLI behavior

The compiled CLI exit contract is:

| Exit | Meaning |
|---:|---|
| 0 | Allowed, or a block reported in `observe`/`advise` mode |
| 2 | Blocked in `enforce` mode |
| 3 | Evaluation/provider failure |
| 4 | Configuration or invocation failure |

The deployment-readiness example deliberately uses `advise`; its failed JUnit evidence produces a portable block and certificate while returning exit 0. Unit coverage confirms the same evidence returns exit 2 in `enforce` mode.

## Verification evidence

The final release-gate pass completed successfully on 16 July 2026:

| Verification | Result |
|---|---|
| Root Go protocol, CLI, relay, provider, and conformance tests | Pass |
| Root Go vet | Pass |
| Nested Engine unit/integration-boundary tests | Pass |
| Nested Engine Go vet | Pass |
| TypeScript SDK build and conformance | Pass |
| Python SDK tests | Pass |
| React adapter TypeScript build | Pass |
| OpenAI Agents and MCP Python compilation | Pass |
| JSON Schema and release manifest parsing | Pass |
| Next.js production build | Pass |
| Codex worker TypeScript build | Pass |
| Demo agent compilation | Pass |
| Docker Compose configuration validation | Pass |
| Compiled offline CLI deployment example | Pass |

The canonical fixture digest is identical in Go, TypeScript, and Python. A live Remote-provider evaluation against the Engine facade also returned a portable block, emitted the expected lifecycle history, and produced a certificate whose digest was independently reproduced by the Python SDK.

## Release and operational boundaries

The implementation is complete locally, but these release actions require the publishing organization's authority and infrastructure:

1. Publish the configured repository/package coordinates and maintain stable release tags.
2. Run the matrix on a clean hosted CI runner, including the PostgreSQL service job.
3. Publish Go, npm, Python, CLI binary, and schema artifacts.
4. Create and publish the `v0.1.0` Git tag.
5. Grant the compatibility badge only to implementations that pass the claimed conformance profile.

The portable facade ledger and live SSE subscriber registry are process-local in v0.1; normalized Engine runs/events use the configured Engine repository. Durable portable replay across server restarts is an operational hardening item rather than part of the vendor-neutral wire contract. Docker Compose syntax was validated, but the final pass did not start containers because no active Docker daemon was available.

## Conclusion

Every implementation item in the attached v0.1 backlog has a corresponding repository artifact and automated or build-level verification. What remains is release administration and optional production hardening, not protocol design or product scaffolding.
