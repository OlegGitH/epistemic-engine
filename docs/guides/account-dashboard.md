# Account dashboard

The Epistemic Control Center aggregates project knowledge, declared AI usage, verification activity, and Decision Certificates inside an account workspace.

## What the dashboard measures

- **Knowledge coverage:** supported or externally verified claims divided by all analyzed claims.
- **AI systems:** declared model usages, including provider, model, purpose, data classes, tools, and owner.
- **Valid certificates:** issued decisions whose policy evaluation permits the proposed action.
- **Attention items:** contradictions, stale claims, unresolved unknowns, and AI systems without a valid certificate.

A certificate applies to a specific decision and declared AI usage under a named policy. It is not a universal endorsement of a model.

## Start locally

```bash
docker compose up --build
```

Open <http://localhost:3000>. The first screen creates an account workspace. The account ID is kept in browser storage; it can also be supplied at build time as `NEXT_PUBLIC_EPISTEMIC_ACCOUNT_ID` or in a shareable local URL as `?account=ACCOUNT_ID`.

## Register through the API

Create an account:

```bash
curl -X POST http://localhost:8080/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme AI","slug":"acme-ai"}'
```

Register a project using the returned account ID:

```bash
curl -X POST http://localhost:8080/v1/accounts/ACCOUNT_ID/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"Food Lens","repository":"acme/food-lens","owner":"Applied AI"}'
```

Register a model usage using the returned project ID:

```bash
curl -X POST http://localhost:8080/v1/projects/PROJECT_ID/ai-systems \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Food image analyzer",
    "provider":"OpenAI",
    "model":"gpt-5.6",
    "purpose":"Assess visible food evidence and explain uncertainty",
    "data_classes":["user_image","metadata"],
    "tools":[],
    "owner":"Applied AI"
  }'
```

## Link evidence runs

Include the account, project, and AI system when creating a run:

```json
{
  "account_id": "ACCOUNT_ID",
  "project_id": "PROJECT_ID",
  "ai_system_id": "AI_SYSTEM_ID",
  "title": "Food Lens release 0.1",
  "recommendation": "Deploy the food image analysis service",
  "source": "github-actions",
  "risk_level": "high"
}
```

Claims, evidence, unknowns, verifications, and certificates produced by that run are then included in the account dashboard at:

```http
GET /v1/accounts/ACCOUNT_ID/dashboard
```

The original run-level claim graph remains available under **Run debugger** or at <http://localhost:3000/run>.

## Connect GitHub Actions

The dashboard can create a project-scoped connection token and a ready-to-paste workflow step. From the UI, choose **Connect project**. Through the API:

```bash
curl -X POST http://localhost:8080/v1/projects/PROJECT_ID/connections \
  -H "Content-Type: application/json" \
  -d '{
    "provider":"github-actions",
    "repository":"acme/food-lens",
    "endpoint":"https://epistemic.example.com"
  }'
```

The response contains:

- a connection record;
- a plaintext `epk_...` token that is returned only once;
- the GitHub Actions step configured for dashboard publishing.

Store the token as the repository secret `EPISTEMIC_INGEST_TOKEN`. Store the public Control Center URL as the repository variable `EPISTEMIC_ENDPOINT`. The publishing-capable action configuration is:

```yaml
- name: Epistemic quality gate
  uses: OlegGitH/epistemic-engine/adapters/github-action@v0.2
  with:
    config: .epistemic.yaml
    certificate: .epistemic/certificate.json
    report: .epistemic/project-quality.json
    endpoint: ${{ vars.EPISTEMIC_ENDPOINT }}
    token: ${{ secrets.EPISTEMIC_INGEST_TOKEN }}
    ai_system: AI_SYSTEM_ID
```

After local evaluation, the action posts the project report, commit and workflow context, and portable certificate to `POST /v1/ingest`. The API:

1. hashes the supplied token and resolves its active project connection;
2. ensures the optional AI system belongs to that project;
3. independently reproduces and verifies the certificate SHA-256 digest;
4. stores the report once per workflow attempt;
5. stores the certificate once per project and digest;
6. updates connection activity and account dashboard status.

Publishing is optional. Without `endpoint` and `token`, the action continues to operate as a local-only quality gate and uploads its certificate as a GitHub artifact.

## PostgreSQL migration

New installations apply `000001_init.up.sql`, `000002_portfolio.up.sql`, and `000003_project_connections.up.sql` automatically. For an existing database, apply both portfolio migrations with your normal migration process before starting the updated control plane.

## Security boundary

The current API scopes portfolio data by account ID but does not yet authenticate users or CI callers. Do not expose it as a multi-tenant public service until account membership, RBAC, and workload authentication are enabled. The recommended production path is user SSO plus GitHub Actions OIDC for certificate ingestion; long-lived repository tokens should be a fallback only.
