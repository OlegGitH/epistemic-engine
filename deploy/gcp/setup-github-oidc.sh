#!/usr/bin/env bash
set -euo pipefail
: "${GCP_PROJECT_ID:?Set GCP_PROJECT_ID}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-OlegGitH/epistemic-engine}"
POOL="github-actions"
PROVIDER="github"
DEPLOYER="epistemic-deployer"
PROJECT_NUMBER="$(gcloud projects describe "$GCP_PROJECT_ID" --format='value(projectNumber)')"

gcloud services enable iamcredentials.googleapis.com sts.googleapis.com
gcloud iam service-accounts describe "${DEPLOYER}@${GCP_PROJECT_ID}.iam.gserviceaccount.com" >/dev/null 2>&1 || gcloud iam service-accounts create "$DEPLOYER" --display-name "Epistemic GitHub deployer"
for role in roles/viewer roles/run.admin roles/cloudsql.admin roles/artifactregistry.admin roles/cloudbuild.builds.editor roles/storage.admin roles/secretmanager.admin roles/iam.serviceAccountAdmin roles/iam.serviceAccountUser roles/resourcemanager.projectIamAdmin roles/serviceusage.serviceUsageAdmin; do
  gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" --member "serviceAccount:${DEPLOYER}@${GCP_PROJECT_ID}.iam.gserviceaccount.com" --role "$role" --quiet >/dev/null
done
gcloud iam workload-identity-pools describe "$POOL" --location global >/dev/null 2>&1 || gcloud iam workload-identity-pools create "$POOL" --location global --display-name "GitHub Actions"
gcloud iam workload-identity-pools providers describe "$PROVIDER" --workload-identity-pool "$POOL" --location global >/dev/null 2>&1 || gcloud iam workload-identity-pools providers create-oidc "$PROVIDER" --workload-identity-pool "$POOL" --location global --issuer-uri https://token.actions.githubusercontent.com --attribute-mapping 'google.subject=assertion.sub,attribute.repository=assertion.repository' --attribute-condition "assertion.repository=='${GITHUB_REPOSITORY}'"
gcloud iam service-accounts add-iam-policy-binding "${DEPLOYER}@${GCP_PROJECT_ID}.iam.gserviceaccount.com" --role roles/iam.workloadIdentityUser --member "principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/attribute.repository/${GITHUB_REPOSITORY}"

printf 'GCP_WORKLOAD_IDENTITY_PROVIDER=projects/%s/locations/global/workloadIdentityPools/%s/providers/%s\nGCP_DEPLOYER_SERVICE_ACCOUNT=%s@%s.iam.gserviceaccount.com\n' "$PROJECT_NUMBER" "$POOL" "$PROVIDER" "$DEPLOYER" "$GCP_PROJECT_ID"
