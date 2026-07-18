# Epistemic Protocol quickstart

This quickstart instruments and evaluates a decision in under 15 minutes. It requires Go 1.25. Docker and OpenAI credentials are not required.

## 1. Run the offline example

From the repository root:

```bash
go run ./cmd/epistemic evaluate --config examples/deployment-readiness/.epistemic.yaml
```

The CLI reads a failing JUnit artifact, emits `verification.failed`, evaluates with the local provider, prints a portable `block`, and writes `.epistemic/example-certificate.json`.

## 2. Start the reference engine

```bash
cd apps/control-plane
go run ./cmd/server
```

Capability discovery is available at:

```bash
curl http://localhost:8080/.well-known/epistemic
```

## 3. Use the Remote provider

Copy `.epistemic.example.yaml` to `.epistemic.yaml`, keep `provider.type: remote`, and run:

```bash
go run ./cmd/epistemic evaluate --config .epistemic.yaml
```

The same events now cross HTTP, are normalized into the reference engine, and return a portable result/certificate. Switch to `provider.type: file` to capture the same instrumentation as JSONL without an engine.

## 4. Adoption modes

- `observe`: collect and evaluate without failing the workflow.
- `advise`: report the result without blocking.
- `enforce`: exit `2` for a blocked/unapproved decision, `3` for indeterminate, and `4` for protocol/configuration failure.

The protocol core does not import OpenAI, OpenTelemetry, PostgreSQL, or the Engine UI.
