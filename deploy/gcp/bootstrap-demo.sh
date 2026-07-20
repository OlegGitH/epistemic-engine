#!/usr/bin/env bash
set -euo pipefail

: "${EPISTEMIC_ENDPOINT:?Set EPISTEMIC_ENDPOINT to the deployed control-plane URL}"
: "${EPISTEMIC_DASHBOARD_URL:?Set EPISTEMIC_DASHBOARD_URL to the deployed dashboard URL}"
DEMO_REPOSITORY="${DEMO_REPOSITORY:-OlegGitH/epistemic-engine-demo}"
STAMP="$(date +%s)"

post() { curl --fail --silent --show-error -H 'Content-Type: application/json' -d "$2" "${EPISTEMIC_ENDPOINT}$1"; }
ACCOUNT="$(post /v1/accounts "$(jq -nc --arg slug "food-lens-${STAMP}" '{name:"Food Lens Demo",slug:$slug}')")"
ACCOUNT_ID="$(jq -r .id <<<"$ACCOUNT")"
PROJECT="$(post "/v1/accounts/${ACCOUNT_ID}/projects" "$(jq -nc --arg repository "$DEMO_REPOSITORY" '{name:"Food Lens",slug:"food-lens",repository:$repository,owner:"OlegGitH"}')")"
PROJECT_ID="$(jq -r .id <<<"$PROJECT")"
AI_SYSTEM="$(post "/v1/projects/${PROJECT_ID}/ai-systems" '{"name":"Food Lens Classifier","provider":"deterministic-demo","model":"food-lens-demo-v0.1","purpose":"Classify food images as healthy, mixed, or less healthy for an educational demonstration.","data_classes":["food-image-derived-features"],"tools":["browser-canvas","epistemic-engine"],"owner":"OlegGitH"}')"
AI_SYSTEM_ID="$(jq -r .id <<<"$AI_SYSTEM")"
CONNECTION="$(post "/v1/projects/${PROJECT_ID}/connections" "$(jq -nc --arg repository "$DEMO_REPOSITORY" --arg endpoint "$EPISTEMIC_ENDPOINT" '{provider:"github-actions",repository:$repository,endpoint:$endpoint}')")"
TOKEN="$(jq -r .token <<<"$CONNECTION")"

gh variable set EPISTEMIC_ENDPOINT --repo "$DEMO_REPOSITORY" --body "$EPISTEMIC_ENDPOINT"
gh variable set EPISTEMIC_AI_SYSTEM_ID --repo "$DEMO_REPOSITORY" --body "$AI_SYSTEM_ID"
gh secret set EPISTEMIC_INGEST_TOKEN --repo "$DEMO_REPOSITORY" --body "$TOKEN"
gh workflow run "Food Lens CI" --repo "$DEMO_REPOSITORY" --ref main

printf 'ACCOUNT_ID=%s\nPROJECT_ID=%s\nAI_SYSTEM_ID=%s\nDASHBOARD_URL=%s/?account=%s\n' "$ACCOUNT_ID" "$PROJECT_ID" "$AI_SYSTEM_ID" "${EPISTEMIC_DASHBOARD_URL%/}" "$ACCOUNT_ID"
