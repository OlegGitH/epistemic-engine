# Deploy Epistemic Engine to Google Cloud

This package deploys the control plane and dashboard to Cloud Run, uses Cloud SQL for PostgreSQL state, stores the database URL in Secret Manager, runs schema migrations as a Cloud Run Job, and stores images in Artifact Registry.

## One-time local deployment

Requirements: `gcloud`, a billing-enabled project, project-owner-equivalent permissions, Bash, Python 3, and `jq`/`gh` for demo bootstrap.

```bash
export GCP_PROJECT_ID="your-project-id"
export GCP_REGION="europe-west1"
bash deploy/gcp/deploy.sh
```

The script is idempotent and prints `CONTROL_PLANE_URL` and `DASHBOARD_URL`. It creates billable resources, most notably Cloud SQL. Review regional pricing before execution.

The default deployment is a public demonstration: Cloud Run allows unauthenticated access while project publishing remains protected by project-scoped ingest tokens. Before production use, add identity-aware access, restrict CORS, configure backups/high availability, and select an appropriate Cloud SQL tier.

## Connect the Food Lens pipeline

After deployment, copy the printed URLs and run:

```bash
export EPISTEMIC_ENDPOINT="https://epistemic-control-plane-...run.app"
export EPISTEMIC_DASHBOARD_URL="https://epistemic-dashboard-...run.app"
bash deploy/gcp/bootstrap-demo.sh
```

The bootstrap creates a dashboard account, project, AI-system registration, and ingest connection. It writes `EPISTEMIC_ENDPOINT`, `EPISTEMIC_AI_SYSTEM_ID`, and `EPISTEMIC_INGEST_TOKEN` to `OlegGitH/epistemic-engine-demo`, then triggers the real Food Lens GitHub Actions workflow. Its commit SHA, report, and certificate will appear in the deployed dashboard.

## GitHub deployment workflow

For keyless deployments, run `setup-github-oidc.sh` once as a GCP administrator. Store its outputs and the project ID as repository variables:

- `GCP_PROJECT_ID`
- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_DEPLOYER_SERVICE_ACCOUNT`

Then run **Deploy Epistemic Engine to GCP** from GitHub Actions. The workflow is manual-only to prevent accidental infrastructure cost.
