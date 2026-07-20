import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";

const endpoint = (process.env.EPISTEMIC_ENDPOINT || "http://127.0.0.1:8081").replace(/\/$/, "");
const model = process.env.OPENAI_MODEL || "gpt-5.6";
const stamp = `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;

const health = await api("/healthz");
assert.equal(health.status, "ok");
const account = await api("/v1/accounts", { method:"POST", expected:201, body:{ name:`Build Week live proof ${stamp}`, slug:`build-week-proof-${stamp}` } });
const project = await api(`/v1/accounts/${account.id}/projects`, { method:"POST", expected:201, body:{ name:"Epistemic Engine", repository:"OlegGitH/epistemic-engine", owner:"OlegGitH" } });
const aiSystem = await api(`/v1/projects/${project.id}/ai-systems`, { method:"POST", expected:201, body:{ name:"GPT-5.6 claim analyzer", provider:"OpenAI", model, purpose:"Propose observable deployment-readiness claims from typed evidence", data_classes:["ci_metadata", "test_results"], tools:["responses_api"], owner:"OlegGitH" } });
const run = await api("/v1/runs", { method:"POST", expected:201, body:{ account_id:account.id, project_id:project.id, ai_system_id:aiSystem.id, external_trace_id:`submission-${stamp}`, title:"OpenAI Build Week live proof", source:"submission-proof", recommendation:"Release the evidenced Epistemic Engine revision.", action_type:"software_deployment", subject:"OlegGitH/epistemic-engine", risk_level:"high" } });

const events = [
  ["build.completed", { status:"passed", artifact:"epistemic-engine" }],
  ["test.completed", { status:"passed", suite:"Go, Node, Python and conformance" }],
  ["compatibility.test.completed", { status:"passed", suite:"protocol compatibility" }],
  ["privacy.test.completed", { status:"passed", suite:"trace redaction boundary" }],
  ["rollback.check.completed", { status:"ready", target:"previous Cloud Run revision" }]
];
for (const [index, [type, payload]] of events.entries()) {
  await api(`/v1/runs/${run.id}/events`, { method:"POST", expected:202, body:{ external_id:`submission-${index + 1}-${stamp}`, sequence:index + 1, type, source:"submission-proof", correlation_id:stamp, payload } });
}

const graph = await api(`/v1/runs/${run.id}/analyze`, { method:"POST" });
assert.ok(graph.claims.length >= 3 && graph.claims.length <= 7, `GPT-5.6 returned ${graph.claims.length} claims; expected 3–7`);
assert.ok(graph.claims.every(claim => claim.id && claim.statement && claim.state), "analysis returned an incomplete claim");
const certificate = await api(`/v1/decisions/${graph.decision.id}/evaluate`, { method:"POST", body:{ human_approved:true } });
assert.match(certificate.proof.digest, /^[a-f0-9]{64}$/);
const humanReport = await api(`/v1/decisions/${graph.decision.id}/certificate/report`);
assert.equal(humanReport.proof.digest, certificate.proof.digest);

const proof = {
  schema_version:"epistemic-build-week-proof/v1",
  created_at:new Date().toISOString(),
  provider:"OpenAI",
  model,
  api:"Responses API",
  account_id:account.id,
  project_id:project.id,
  ai_system_id:aiSystem.id,
  run_id:run.id,
  decision_id:graph.decision.id,
  claim_count:graph.claims.length,
  unknown_count:graph.unknowns.length,
  verdict:certificate.verdict,
  action_allowed:certificate.action_allowed,
  certificate_digest:certificate.proof.digest,
  claims:graph.claims.map(({id, statement, state, critical}) => ({id, statement, state, critical}))
};
await mkdir(".cache/submission", { recursive:true });
await Promise.all([
  writeFile(".cache/submission/openai-proof.json", JSON.stringify(proof, null, 2) + "\n"),
  writeFile(".cache/submission/openai-certificate-report.md", humanReport.markdown)
]);
console.log(`openai-submission-proof-ok: ${model}, ${proof.claim_count} claims, ${proof.verdict}`);
console.log(`certificate: ${proof.certificate_digest}`);

async function api(path, { method="GET", body, expected=200 } = {}) {
  const response = await fetch(`${endpoint}${path}`, { method, headers:body === undefined ? {} : { "Content-Type":"application/json" }, body:body === undefined ? undefined : JSON.stringify(body) });
  const text = await response.text();
  let parsed = {};
  if (text) {
    try { parsed = JSON.parse(text); } catch { parsed = { raw:text }; }
  }
  assert.equal(response.status, expected, `${method} ${path}: ${response.status} ${text}`);
  return parsed;
}
