#!/usr/bin/env bash
set -euo pipefail

: "${GCP_PROJECT_ID:?Set GCP_PROJECT_ID to the target Google Cloud project}"
REGION="${GCP_REGION:-europe-west1}"
PREFIX="${EPISTEMIC_RESOURCE_PREFIX:-epistemic}"
REPOSITORY="${PREFIX}-containers"
SQL_INSTANCE="${PREFIX}-postgres"
DATABASE="epistemic"
DATABASE_USER="epistemic"
DATABASE_SECRET="${PREFIX}-database-url"
RUNTIME_SA="${PREFIX}-runtime@${GCP_PROJECT_ID}.iam.gserviceaccount.com"
TAG="${GITHUB_SHA:-$(git rev-parse --short=12 HEAD)}"
REGISTRY="${REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${REPOSITORY}"
CONTROL_IMAGE="${REGISTRY}/control-plane:${TAG}"
WEB_IMAGE="${REGISTRY}/dashboard:${TAG}"
MIGRATE_IMAGE="${REGISTRY}/migrate:${TAG}"

gcloud config set project "$GCP_PROJECT_ID" >/dev/null
gcloud services enable artifactregistry.googleapis.com cloudbuild.googleapis.com run.googleapis.com sqladmin.googleapis.com secretmanager.googleapis.com iamcredentials.googleapis.com
PROJECT_NUMBER="$(gcloud projects describe "$GCP_PROJECT_ID" --format='value(projectNumber)')"
LEGACY_CLOUD_BUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"
DEFAULT_COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

if ! gcloud artifacts repositories describe "$REPOSITORY" --location "$REGION" >/dev/null 2>&1; then
  gcloud artifacts repositories create "$REPOSITORY" --repository-format docker --location "$REGION" --description "Epistemic Engine containers"
fi
if ! gcloud iam service-accounts describe "$RUNTIME_SA" >/dev/null 2>&1; then
  gcloud iam service-accounts create "${PREFIX}-runtime" --display-name "Epistemic Cloud Run runtime"
fi
# IAM can briefly lag behind service-account creation on a new project.
for attempt in {1..12}; do
  if gcloud iam service-accounts describe "$RUNTIME_SA" >/dev/null 2>&1; then
    break
  fi
  if [[ "$attempt" == 12 ]]; then
    echo "Runtime service account did not become available: $RUNTIME_SA" >&2
    exit 1
  fi
  sleep 5
done
for role in roles/cloudsql.client roles/secretmanager.secretAccessor; do
  gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" --member "serviceAccount:${RUNTIME_SA}" --role "$role" --quiet >/dev/null
done

if gcloud iam service-accounts describe "$LEGACY_CLOUD_BUILD_SA" >/dev/null 2>&1; then
  CLOUD_BUILD_SA="$LEGACY_CLOUD_BUILD_SA"
else
  CLOUD_BUILD_SA="$DEFAULT_COMPUTE_SA"
fi
gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" --member "serviceAccount:${CLOUD_BUILD_SA}" --role roles/artifactregistry.writer --quiet >/dev/null

if ! gcloud sql instances describe "$SQL_INSTANCE" >/dev/null 2>&1; then
  gcloud sql instances create "$SQL_INSTANCE" --database-version POSTGRES_17 --edition enterprise --tier db-f1-micro --region "$REGION" --storage-type SSD --storage-size 10GB --availability-type zonal
fi
if ! gcloud sql databases describe "$DATABASE" --instance "$SQL_INSTANCE" >/dev/null 2>&1; then
  gcloud sql databases create "$DATABASE" --instance "$SQL_INSTANCE"
fi

CONNECTION_NAME="$(gcloud sql instances describe "$SQL_INSTANCE" --format='value(connectionName)')"
if ! gcloud secrets describe "$DATABASE_SECRET" >/dev/null 2>&1; then
  DATABASE_PASSWORD="$(python3 -c 'import secrets; print(secrets.token_urlsafe(32))')"
  gcloud sql users create "$DATABASE_USER" --instance "$SQL_INSTANCE" --password "$DATABASE_PASSWORD"
  ENCODED_PASSWORD="$(python3 -c 'import sys,urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$DATABASE_PASSWORD")"
  ENCODED_SOCKET="$(python3 -c 'import sys,urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "/cloudsql/${CONNECTION_NAME}")"
  DATABASE_URL="postgresql://${DATABASE_USER}:${ENCODED_PASSWORD}@/${DATABASE}?host=${ENCODED_SOCKET}"
  gcloud secrets create "$DATABASE_SECRET" --replication-policy automatic
  printf '%s' "$DATABASE_URL" | gcloud secrets versions add "$DATABASE_SECRET" --data-file=-
fi

gcloud builds submit . --config deploy/gcp/cloudbuild-control-plane.yaml --substitutions "_IMAGE=${CONTROL_IMAGE}"
gcloud builds submit . --config deploy/gcp/cloudbuild-migrate.yaml --substitutions "_IMAGE=${MIGRATE_IMAGE}"

if gcloud run jobs describe "${PREFIX}-migrate" --region "$REGION" >/dev/null 2>&1; then
  gcloud run jobs update "${PREFIX}-migrate" --region "$REGION" --image "$MIGRATE_IMAGE" --service-account "$RUNTIME_SA" --set-cloudsql-instances "$CONNECTION_NAME" --set-secrets "DATABASE_URL=${DATABASE_SECRET}:latest" --max-retries 1 --task-timeout 10m
else
  gcloud run jobs create "${PREFIX}-migrate" --region "$REGION" --image "$MIGRATE_IMAGE" --service-account "$RUNTIME_SA" --set-cloudsql-instances "$CONNECTION_NAME" --set-secrets "DATABASE_URL=${DATABASE_SECRET}:latest" --max-retries 1 --task-timeout 10m
fi
gcloud run jobs execute "${PREFIX}-migrate" --region "$REGION" --wait

gcloud run deploy "${PREFIX}-control-plane" --region "$REGION" --image "$CONTROL_IMAGE" --service-account "$RUNTIME_SA" --add-cloudsql-instances "$CONNECTION_NAME" --set-secrets "DATABASE_URL=${DATABASE_SECRET}:latest" --set-env-vars "ANALYZER_MODE=rules,EXECUTION_MODE=recorded,REQUIRE_DURABLE_STORAGE=true" --port 8080 --memory 512Mi --cpu 1 --min-instances 0 --max-instances 5 --timeout 3600 --allow-unauthenticated
CONTROL_PLANE_URL="$(gcloud run services describe "${PREFIX}-control-plane" --region "$REGION" --format='value(status.url)')"

gcloud builds submit . --config deploy/gcp/cloudbuild-web.yaml --substitutions "_IMAGE=${WEB_IMAGE},_CONTROL_PLANE_URL=${CONTROL_PLANE_URL}"
gcloud run deploy "${PREFIX}-dashboard" --region "$REGION" --image "$WEB_IMAGE" --service-account "$RUNTIME_SA" --port 3000 --memory 512Mi --cpu 1 --min-instances 0 --max-instances 3 --allow-unauthenticated
DASHBOARD_URL="$(gcloud run services describe "${PREFIX}-dashboard" --region "$REGION" --format='value(status.url)')"

printf '\nEpistemic Engine deployed.\nCONTROL_PLANE_URL=%s\nDASHBOARD_URL=%s\nCLOUD_SQL_INSTANCE=%s\n' "$CONTROL_PLANE_URL" "$DASHBOARD_URL" "$CONNECTION_NAME"
