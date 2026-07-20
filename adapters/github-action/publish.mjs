import fs from "node:fs/promises";
import { existsSync } from "node:fs";

const endpoint = required("EPISTEMIC_ENDPOINT").replace(/\/+$/, "");
const token = required("EPISTEMIC_INGEST_TOKEN");
const certificatePath = required("EPISTEMIC_CERTIFICATE_PATH");
const reportPath = process.env.EPISTEMIC_REPORT_PATH ?? "";

const certificate = JSON.parse(await fs.readFile(certificatePath, "utf8"));
const report = reportPath && existsSync(reportPath)
  ? JSON.parse(await fs.readFile(reportPath, "utf8"))
  : undefined;
const serverURL = process.env.GITHUB_SERVER_URL ?? "https://github.com";
const repository = process.env.GITHUB_REPOSITORY ?? "";
const runID = process.env.GITHUB_RUN_ID ?? "local";
const runAttempt = process.env.GITHUB_RUN_ATTEMPT ?? "1";

const response = await fetch(`${endpoint}/v1/ingest`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    external_id: `${runID}-${runAttempt}`,
    ai_system_id: process.env.EPISTEMIC_AI_SYSTEM_ID || undefined,
    context: {
      repository,
      commit_sha: process.env.GITHUB_SHA ?? "",
      branch: process.env.GITHUB_REF_NAME ?? "",
      workflow: process.env.GITHUB_WORKFLOW ?? "",
      run_url: repository ? `${serverURL}/${repository}/actions/runs/${runID}` : "",
    },
    report,
    certificate,
  }),
});

const body = await response.text();
if (!response.ok) {
  throw new Error(`Epistemic publish failed (${response.status}): ${body}`);
}
console.log(`Published Epistemic report and certificate: ${body}`);

function required(name) {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
}
