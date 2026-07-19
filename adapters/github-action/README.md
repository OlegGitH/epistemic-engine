# GitHub Actions adapter

```yaml
- uses: OlegGitH/epistemic-engine/adapters/github-action@v0.2
  with:
    config: .epistemic.yaml
    certificate: .epistemic/certificate.json
```

In `observe` and `advise` modes the step reports without blocking. In `enforce` mode exit code `2` blocks the workflow and exit code `3` reports an indeterminate decision. The certificate is uploaded even when enforcement blocks.

## Generate a standardized pipeline

The Engine exposes a vendor-neutral generator rather than requiring every repository to hand-author the Epistemic gate:

```http
POST /v1/tools/github-actions/pipelines
Content-Type: application/json

{
  "epistemic_action": "OlegGitH/epistemic-engine/adapters/github-action@v0.1",
  "config_path": ".epistemic.yaml",
  "certificate_path": ".epistemic/certificate.json"
}
```

The response contains `.github/workflows/epistemic-ci.yml` plus the required GitHub secrets and variables. The generator never writes to a repository or broadens GitHub permissions; an authorized caller decides whether to apply the returned file.

The same operation is available through MCP as `epistemic_create_github_pipeline`. Any CI producer can use the generic `tool` evidence contract: `{ "tool": "...", "status": "passed|failed|pending" }`. Tool-specific execution belongs in project configuration or a separate adapter, not in the pipeline generator.

## Publish to the account dashboard

Create a project connection in the Control Center, store the returned one-time token as the `EPISTEMIC_INGEST_TOKEN` repository secret, and add the optional publishing inputs:

```yaml
- name: Epistemic quality gate
  uses: OlegGitH/epistemic-engine/adapters/github-action@v0.2
  with:
    config: .epistemic.yaml
    certificate: .epistemic/certificate.json
    report: .epistemic/project-quality.json
    endpoint: ${{ vars.EPISTEMIC_ENDPOINT }}
    token: ${{ secrets.EPISTEMIC_INGEST_TOKEN }}
    ai_system: ais_registered_system_id
```

Publishing is skipped when `endpoint` or `token` is absent, so the action remains usable as a local-only gate. The dashboard endpoint authenticates the project token, verifies the certificate digest, and deduplicates repeated workflow attempts.

See the [pipeline tool guide](../../docs/guides/github-actions-pipeline.md) for setup, Swagger, cURL, PowerShell, evidence, and troubleshooting examples.
