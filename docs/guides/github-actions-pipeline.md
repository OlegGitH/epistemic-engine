# GitHub Actions pipeline tool

The pipeline tool creates a small, reviewable GitHub Actions workflow that runs an Epistemic quality gate. It does not connect to GitHub, commit files, or choose project-specific test and analysis tools for you. The caller receives the proposed file content and decides where and how to apply it.

## What you need

- A running Epistemic control plane.
- An `.epistemic.yaml` file in the target repository.
- Access to the Epistemic GitHub Action, either from the target repository or through a published action reference.
- Permission to add a file under `.github/workflows/` in the target repository.

## 1. Start the API and open Swagger

From the repository root, start only the API:

```bash
make run-api
```

Or start the complete Docker Compose environment:

```bash
docker compose up --build
```

Open <http://localhost:8080/docs/>. The Swagger page lists the endpoints, request schemas, examples, and an interactive **Try it out** button. The raw OpenAPI document is available at <http://localhost:8080/openapi.yaml>.

## 2. Discover the available tools

```bash
curl --fail http://localhost:8080/v1/tools
```

The catalog currently includes `github-actions-pipeline`. Clients should use the catalog instead of assuming a tool is installed.

## 3. Generate a workflow

Use an action published by the organization that hosts Epistemic Engine:

```bash
curl --fail-with-body \
  --request POST \
  --header "Content-Type: application/json" \
  --data '{
    "name": "Epistemic CI",
    "epistemic_action": "OlegGitH/epistemic-engine/adapters/github-action@v0.1",
    "config_path": ".epistemic.yaml",
    "certificate_path": ".epistemic/certificate.json"
  }' \
  http://localhost:8080/v1/tools/github-actions/pipelines
```

`OlegGitH/epistemic-engine/adapters/github-action@v0.1` is an example. Replace it with an action reference your GitHub organization can access. If the adapter is stored in the same target repository, use `./adapters/github-action` instead.

All request properties are optional. An empty JSON object uses the local-action and standard-path defaults:

```bash
curl --fail-with-body \
  --request POST \
  --header "Content-Type: application/json" \
  --data '{}' \
  http://localhost:8080/v1/tools/github-actions/pipelines
```

The response has this shape:

```json
{
  "tool_id": "github-actions-pipeline",
  "files": [
    {
      "path": ".github/workflows/epistemic-ci.yml",
      "content": "name: \"Epistemic CI\"\n..."
    }
  ],
  "required_secrets": [],
  "required_variables": []
}
```

## 4. Review and apply the returned file

The API deliberately does not write to a repository. Review the returned path, content, secrets, and variables before applying them.

With Bash and `jq`:

```bash
mkdir -p .github/workflows
curl --fail-with-body \
  --request POST \
  --header "Content-Type: application/json" \
  --data '{"epistemic_action":"OlegGitH/epistemic-engine/adapters/github-action@v0.1"}' \
  http://localhost:8080/v1/tools/github-actions/pipelines \
  | jq -r '.files[] | select(.path == ".github/workflows/epistemic-ci.yml") | .content' \
  > .github/workflows/epistemic-ci.yml
```

With PowerShell:

```powershell
$body = @{
  epistemic_action = "OlegGitH/epistemic-engine/adapters/github-action@v0.1"
} | ConvertTo-Json

$result = Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/v1/tools/github-actions/pipelines" `
  -ContentType "application/json" `
  -Body $body

$file = $result.files | Where-Object path -eq ".github/workflows/epistemic-ci.yml"
New-Item -ItemType Directory -Force ".github/workflows" | Out-Null
Set-Content -Path $file.path -Value $file.content -NoNewline
```

Commit the workflow only after reviewing it:

```bash
git diff -- .github/workflows/epistemic-ci.yml
git add .github/workflows/epistemic-ci.yml .epistemic.yaml
git commit -m "Add Epistemic quality gate"
```

## 5. Configure the quality gate

The generated workflow delegates policy and evidence selection to `.epistemic.yaml`. A minimal local configuration is:

```yaml
api_version: epistemic.dev/v1alpha1
mode: enforce
decision:
  id: deployment-readiness
  recommendation: The service is ready to deploy.
  action_type: software_deployment
  subject_type: repository
  subject_id: my-service
  risk_level: high
  approved: true
  approved_by: ci-policy
provider:
  type: local
requirements:
  - id: project-quality
    description: The configured project quality checks pass.
    critical: true
    evidence_types: [tool]
sources:
  - type: tool
    name: project-quality
    path: .epistemic/project-quality.json
outputs:
  result: .epistemic/result.json
  certificate: .epistemic/certificate.json
```

The `certificate` path must match the `certificate_path` supplied to the generator so the action can upload it.

### Supplying evidence from any tool

A project-specific step or adapter can write the configured evidence file before the Epistemic action runs:

```json
{
  "tool": "project-quality",
  "status": "passed",
  "exit_code": 0,
  "summary": "All configured checks passed"
}
```

Valid statuses are `passed`, `failed`, `error`, `pending`, and `indeterminate`. This contract allows a test runner, scanner, build system, or internal tool to supply evidence without coupling the pipeline generator to that product.

For example, insert the producer step before `Evaluate configured evidence` in the generated workflow:

```yaml
- name: Produce project quality evidence
  shell: bash
  run: |
    mkdir -p .epistemic
    if ./scripts/project-quality-check; then
      status=passed
      exit_code=0
    else
      exit_code=$?
      status=failed
    fi
    printf '{"tool":"project-quality","status":"%s","exit_code":%s}\n' \
      "$status" "$exit_code" > .epistemic/project-quality.json
```

The generator intentionally does not add this command because each repository owns its toolchain. Standard JUnit and SARIF sources can also be listed directly in `.epistemic.yaml`.

## Gate behavior

The mode in `.epistemic.yaml` controls whether the workflow blocks:

| Mode | Behavior |
| --- | --- |
| `observe` | Produce a result and certificate without blocking. |
| `advise` | Report the decision without blocking. |
| `enforce` | Block failed or unapproved decisions; report indeterminate decisions separately. |

The action always attempts to upload the Decision Certificate when the configured file exists.

## MCP usage

The same operation is exposed as the MCP tool `epistemic_create_github_pipeline`. Set `EPISTEMIC_ENDPOINT` for the MCP adapter when the API is not at `http://localhost:8080`:

```bash
EPISTEMIC_ENDPOINT=https://epistemic.example.com python adapters/mcp/server.py
```

The MCP caller receives the same generated-file response as the REST API and must apply it through its own authorized repository workflow.

## Troubleshooting

- **The action cannot be found:** verify that `epistemic_action` is a reachable published action reference or a valid local path in the target repository.
- **The evidence file cannot be found:** ensure the producer step runs before the Epistemic action and that its output matches the `sources[].path` value.
- **The certificate is not uploaded:** make `outputs.certificate` and the generator's `certificate_path` identical.
- **The gate reports blocked:** inspect the JSON result and certificate under `.epistemic/`, then check failed or contradictory evidence.
- **Swagger is blank:** the HTML is served locally, while its Swagger UI JavaScript and CSS are loaded from `unpkg.com`; allow that domain or use the raw OpenAPI document with another OpenAPI client.
