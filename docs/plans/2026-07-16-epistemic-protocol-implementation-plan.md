# Epistemic Protocol v0.1 — implementation record

**Plan date:** 16 July 2026
**Contract:** Instrument once. Send to any compatible provider. Use any engine. Display in any UI. Gate any workflow.
**Status:** Repository implementation complete; release tag/publication pending

## Architectural boundaries

- No OpenTelemetry or OpenAI dependency in the core protocol.
- No public PostgreSQL, Engine service, or UI entity.
- No UI dependency in schemas or SDK types.
- No standardized hidden reasoning.
- Portable result states with implementation-specific scoring kept behind extensions.

## Delivered structure

```text
specification/                 normative protocol, lifecycle, compatibility, HTTP, schemas
api/go/                        stable vendor-neutral Go types, Client, Provider, hashing
sdk/go/providers/              remote, file, noop, local, composite
sdk/typescript/                types, validation, hashing, providers
sdk/python/                    validation, hashing, providers
cmd/epistemic/                 YAML/artifact evaluation CLI
cmd/epistemic-relay/           validate/redact/archive/retry relay
adapters/                      GitHub, Agents SDK, JUnit, SARIF, MCP, React
apps/control-plane/            Epistemic Engine reference implementation and facade
conformance/                   protocol/provider suites and cross-language fixtures
examples/                      deployment, agent tool, research, alternate server
```

## Backlog completion

| ID | Priority | Result |
|---|---:|---|
| P0-01 | P0 | v0.1 scope and non-goals frozen in the normative specification |
| P0-02 | P0 | Public module separated from the Engine nested module |
| P0-03 | P0 | Major/minor, additive-field, deprecation, extension, and unknown-event rules documented |
| SP-01 | P0 | Universal envelope implemented in schemas and all SDKs |
| SP-02 | P0 | Decision/run/correlation/parent propagation implemented in JSON, Go context, and HTTP header |
| SP-03 | P0 | Eight families and 30 lifecycle event names implemented |
| SP-04 | P0 | Portable request/result/status/reason/condition/error messages implemented |
| SP-05 | P0 | Single, batch, evaluate, query, certificate, discovery, and SSE binding implemented |
| SP-06 | P1 | Status, reason, risk, evidence, and relation registry published |
| SC-01 | P0 | Canonical v0.1 JSON Schemas created |
| SC-02 | P0 | Isolated Go protocol types and validation created |
| SC-03 | P1 | TypeScript and Python types/validation created |
| SC-04 | P0 | Canonical JSON and SHA-256 fixture reproduced in Go, TypeScript, and Python |
| SV-01 | P0 | Engine protocol ingestion facade implemented |
| SV-02 | P0 | Standard synchronous evaluation maps Engine policy to portable results |
| SV-03 | P0 | Capability discovery implemented |
| SV-04 | P0 | Portable events normalize to internal run observations without exposing internal types |
| SV-05 | P1 | Batch ingestion and global portable SSE stream implemented |
| SDK-01 | P0 | Stable Go API exposes Client, Provider, events, decisions, and proof types |
| SDK-02 | P0 | Remote HTTP provider implemented with context propagation and structured errors |
| SDK-03 | P0 | File and No-op providers implemented |
| SDK-04 | P1 | Deterministic in-process Local provider implemented |
| SDK-05 | P1 | Composite fan-out provider implemented |
| CLI-01 | P0 | `.epistemic.yaml` v1alpha1 and schema implemented |
| CLI-02 | P0 | `epistemic evaluate` implements certificate output and stable exit codes |
| CLI-03 | P0 | Diff, JUnit, SARIF, build, migration, log, trace, JSON, and custom artifacts supported |
| CLI-04 | P1 | Observe, advise, and enforce adoption modes implemented |
| AD-01 | P0 | Composite GitHub Action evaluates and uploads the certificate |
| AD-02 | P0 | Privacy-safe OpenAI Agents SDK lifecycle mapper implemented |
| AD-03 | P1 | MCP stdio adapter exposes decision, event, and proof inspection |
| AD-04 | P1 | Portable React decision badge and certificate view implemented |
| RL-01 | P1 | Relay pipeline configuration documented and parsed |
| RL-02 | P1 | HTTP receiver and content-addressed artifact-folder watcher implemented |
| RL-03 | P1 | Validation, recursive redaction, batch limits, and retry implemented |
| RL-04 | P1 | Compatible Engine and JSONL archive exporters implemented |
| CF-01 | P0 | Protocol and Engine facade conformance cases automated |
| CF-02 | P0 | Reusable provider suite covers all P0 Go providers |
| CF-03 | P1 | Deployment, tool execution, research, and alternate-server examples published |
| CF-04 | P1 | RFC process and template published |
| REL-01 | P0 | Quickstart and concepts documentation published in-repository |
| REL-02 | P0 | Aligned v0.1 release manifest prepared; Git tag/publication remains an external release action |
| REL-03 | P1 | Compatibility badge asset created; use requires passing the claimed conformance profile |

## Release gate status

- Go Remote provider emits valid events to the Engine facade.
- File provider captures the same event objects as JSONL.
- The alternate sample server compiles against only the public API and Local provider.
- CLI evaluates YAML and JUnit artifacts and writes a reproducible certificate.
- Duplicate IDs and idempotency keys do not duplicate effects.
- Ordering collisions and invalid parent relationships are rejected.
- Correlation, run, decision, and parent context survive transport.
- Engine responses use portable statuses and standard blocking reasons.
- Canonical hash fixture matches across Go, TypeScript, and Python.
- Protocol, Engine, provider, CLI, and language SDK checks are automated.
- The offline quickstart completes without Docker or vendor credentials.

## External release actions

1. Replace placeholder module/repository coordinates with the publishing organization.
2. Run the CI matrix from a clean hosted runner.
3. Publish Go module, npm package, Python package, CLI binaries, and schemas.
4. Create Git tag `v0.1.0` from the final release commit.
5. Publish the compatibility badge only for conformance-tested implementations.
