# Public APIs

This folder contains the existing Epistemic Engine product API used by its web client. The vendor-neutral interoperability contract is separately versioned under `specification/openapi-v0.1.yaml` and `specification/schemas/v0.1/`.

- `openapi/epistemic-control-plane.yaml` describes the HTTP API.
- `schemas/` contains reusable JSON Schema documents for trace events and certificates.
- `examples/` contains payloads that can be posted directly to a local server.

When the control plane is running, browse the interactive Swagger UI at <http://localhost:8080/docs/> or read the raw contract at <http://localhost:8080/openapi.yaml>. The binary embeds the OpenAPI document and referenced schemas, so these endpoints also work in the container image.

`GET /v1/tools` lists Engine tools. `POST /v1/tools/github-actions/pipelines` generates a vendor-neutral GitHub Actions pipeline that evaluates the evidence configured in `.epistemic.yaml`. It returns file content for an authorized caller to apply; it does not write to GitHub directly.

The API accepts observations and explicit justifications. It has no field for hidden chain-of-thought. Verification execution accepts only results from a controlled `sandbox` or `staging` environment.

New third-party instrumentation should target Epistemic Protocol rather than importing these Engine graph representations.

See the [pipeline tool guide](../../../docs/guides/github-actions-pipeline.md) for complete REST, PowerShell, GitHub Actions, configuration, evidence, and MCP examples.
