# Final requirements and coverage audit

**Audit date:** 21 July 2026  
**Product:** Epistemic Engine  
**Submission category:** Developer Tools  
**Source contracts:** ADR-0001, the control-plane implementation plan, Epistemic Protocol v0.1, and the OpenAI Build Week requirements

## Executive decision

The tool is implemented, deployed, and demonstrably working. Its core safety contract is covered: model output proposes evidence-bound assessments, deterministic policy owns authorization, missing or contradictory critical evidence blocks the action, PostgreSQL persists the run and certificate, and the dashboard exposes both machine and human-readable results.

The software is ready to submit. The submission itself is **not yet complete** because three owner-only Devpost actions remain:

1. upload a public YouTube demo shorter than three minutes;
2. obtain the primary Codex `/feedback` session ID;
3. complete and submit the Devpost form before the deadline.

No additional product feature is required to satisfy the published challenge requirements.

## Official Build Week requirements

The official challenge requires a working project built with Codex and GPT-5.6, a category, description, public demo video, testable repository, README/setup/sample data, explanation of Codex and GPT-5.6 usage, a `/feedback` session ID, and—in the Developer Tools category—installation instructions, supported platforms, and a judge-ready demo or sandbox. Source: [OpenAI Build Week requirements](https://openai.devpost.com/#requirements).

| Requirement | Status | Evidence or remaining action |
| --- | --- | --- |
| Working project using Codex and GPT-5.6 | ✅ Covered | Public GCP deployment, live GPT-5.6 run, official Codex SDK worker, green CI |
| Category | ✅ Covered | Developer Tools |
| Project description | ✅ Covered | `docs/submission/devpost-copy.md` |
| Public code repository | ✅ Covered | `OlegGitH/epistemic-engine`, public |
| Relevant license | ✅ Covered | MIT |
| README and setup instructions | ✅ Covered | Root README, Docker Compose, GCP guide, protocol quickstart |
| Sample data/scenarios | ✅ Covered | Deployment fixtures, Food Lens samples, four lifecycle paths, five PR-coverage paths |
| Codex contribution explained | ✅ Covered | README, submission copy, bounded worker, published proof patch |
| GPT-5.6 use explained and demonstrated | ✅ Covered | Official Go Responses API analyzer plus live PR-review Responses API execution |
| Developer-tool installation and platforms | ✅ Covered | Windows/macOS/Linux, Docker path, hosted judge path |
| Judge can test without rebuilding | ✅ Covered | Public dashboard/API and recorded deterministic paths |
| Public demo video under three minutes | ⛔ Owner action | Script is ready; record and upload to YouTube |
| Codex `/feedback` session ID | ⛔ Owner action | Run `/feedback` in the primary build thread and paste the ID into Devpost |
| Devpost entry submitted | ⛔ Owner action | Paste prepared copy, links, video, and session ID; review and submit |

## ADR definition-of-done coverage

| ADR outcome | Status | Evidence | Qualification |
| --- | --- | --- | --- |
| Import a real agent trace | ⚠️ Implemented | Python Agents SDK exporter, trace-event ingestion, adapter tests | Show this path in the video if claiming a live Agents SDK trace; the latest public live matrix uses the PR reviewer path |
| Recommendation becomes at least four atomic claims | ✅ Covered | Deployment analysis produces five claims; PR fixtures produce one claim per declared requirement |
| Claims show evidence, scope, and state | ✅ Covered | Run graph, evidence inspector, claim details, API graph |
| Contradiction remains visible | ✅ Covered | Privacy and PR contradiction scenarios retain `contradicts` relations and `CONTRADICTED` results |
| Missing critical check is planned | ✅ Covered | Verification-plan endpoint and bounded-verification scenario |
| Codex produces a targeted verification | ✅ Covered | Official Codex SDK worker, test-only patch, thread metadata, SHA-256 patch digest |
| Verification runs and emits an artifact | ✅ Covered | Approved sandbox result, artifact hash, CI fixtures, runner safety tests |
| Initial decision is blocked | ✅ Covered | Missing, contradicted, confidence-only, and approval-pending scenarios |
| Corrected decision is allowed after human approval | ✅ Covered | Supported release and fully-covered PR scenarios |
| Certificate reproduces from stored artifacts/hashes | ✅ Covered | Cross-language canonical hashes, idempotence tests, report/certificate digest checks |
| Story is understandable in under three minutes | ⚠️ Prepared | Timed 2:45 script and public judge path | Must still be recorded and validated by a human viewer |

## What each execution path proves

| Path | What it gives us | What it does not prove |
| --- | --- | --- |
| `npm test` in Food Lens | Deterministic classifier behavior, raw-image privacy boundary, PR reviewer validation, API integration | Engine persistence or live model behavior |
| `npm run smoke` and `npm run evidence` | A runnable application check plus content-addressed CI evidence | Epistemic policy behavior by itself |
| `npm run test:engine` | Public protocol discovery, ingestion, idempotency, ordering, policy, verification gates, certificates, authentication, dashboard aggregation, SSE | Live OpenAI variability or restart durability unless pointed at the PostgreSQL deployment |
| `npm run test:scenario` | One branch-selected deployment story with a precise expected certificate and action result | The other branch outcomes |
| Recorded `npm run test:pr-review` | Repeatable five-case regression for full, partial, missing, contradicted, and confidence-only request coverage | A real model request; reviewer outputs are recorded fixtures |
| Live `PR_REVIEW_PROVIDER=openai` | Real GPT-5.6 Responses API Structured Output, artifact-reference validation, model variability, deterministic downstream safety gate | Deterministic repeatability of the exact epistemic label |
| Cloud Run Job `epistemic-pr-review-live` | Secret Manager boundary, private on-demand compute, live OpenAI-to-Engine-to-PostgreSQL-to-dashboard flow | A continuously running service; the job intentionally has no idle instance |
| PostgreSQL restart test in CI | Accounts, runs, graphs, certificates, reports, and digests survive an Engine restart | Disaster recovery or multi-region failover |
| `make test` in Epistemic Engine | Go protocol/control-plane tests, web and worker builds, Python/TypeScript SDK conformance, adapter compilation | External credentials and human release actions |
| Protocol conformance suite | Vendor-neutral event, provider, ordering, error, proof, and cross-language hashing compatibility | Product-specific scoring or UI behavior |
| Agents SDK demo | Observable trace export, sensitive-data exclusion, interruption and human approval flow | Hidden reasoning; the product explicitly never requests it |
| Codex proof workflow | Approved, test-only, disposable-workspace patch generation with no network and an unapplied digest | Automatic product-code modification or production deployment |
| Public dashboard | Judge-ready inspection of accounts, projects, AI systems, runs, claims, evidence, reports, and certificates | Authentication/tenant administration, which is outside the MVP |

## Scenario coverage

### Deployment lifecycle branches

| Branch | Expected result | Capability exercised |
| --- | --- | --- |
| `scenario/supported-release` | `VERIFIED`, allowed | Complete evidence and approval |
| `scenario/insufficient-evidence` | `INSUFFICIENT_EVIDENCE`, blocked | Critical unknowns |
| `scenario/privacy-contradiction` | `CONTRADICTED`, blocked | Contradiction overrides confidence and approval |
| `scenario/bounded-verification` | `VERIFIED_WITH_CONDITIONS`, blocked | Approved sandbox checks plus separate action approval |

### PR requirement-coverage branches

| Branch | Expected safety outcome | Capability exercised |
| --- | --- | --- |
| `scenario/pr-fully-covered` | Allowed | Every declared requirement has direct evidence |
| `scenario/pr-partially-covered` | Blocked | Core change exists but regression/migration evidence is incomplete |
| `scenario/pr-missing-coverage` | Blocked | A critical security requirement is absent |
| `scenario/pr-contradicted` | Blocked | Test evidence contradicts the compatibility claim |
| `scenario/pr-confidence-only` | Blocked | High model confidence without a valid artifact reference |

Each dedicated PR branch selects exactly one fixture through `epistemic-scenario.json`; `main` runs the complete matrix.

## Latest verified live evidence

- Live OpenAI execution: `epistemic-pr-review-live-z7csc`, completed successfully in 1 minute 10 seconds.
- Live account: [GPT-5.6 PR coverage dashboard](https://epistemic-dashboard-r7zqwwvzgq-ew.a.run.app/?account=acc_cb17e3c15b4bcb1c1dfc27f3).
- Result: five scenarios, one allowed and four blocked.
- Persistence: PostgreSQL with `durable: true`.
- Integrity: all five human-report digests matched their machine certificates.
- Availability: all five direct run pages returned HTTP 200.
- Cost boundary: API and dashboard minimum instances are zero; the private job had zero active executions after completion. Cloud SQL remains the durable component and does not scale to zero.

The live model labeled the fully-covered case `VERIFIED_WITH_CONDITIONS` because it considered malformed-cursor evidence partially supported. Human approval allowed the action, so the required safety result passed. This is useful calibration evidence: exact model labels may vary, but deterministic policy and the action gate remain authoritative.

## Known limitations and honest non-goals

- The hosted demo is judge-oriented and has no full multi-tenant administration or enterprise identity layer.
- The protocol facade's portable replay ledger and SSE subscribers are process-local in v0.1; core Engine data is durable in PostgreSQL.
- The OpenAI reviewer is advisory and variable. It cannot authorize an action or invent an accepted evidence identifier.
- The demo never performs a real production deployment.
- The system does not request, store, or claim access to hidden chain-of-thought.
- Docker sandbox execution is covered by implementation and safety tests; the public PR matrix records bounded evidence rather than applying a generated patch.
- Cloud SQL incurs baseline cost even while Cloud Run services and jobs scale to zero.

## Final submission checklist

1. Open every public repository and GCP link in a signed-out browser.
2. Record the prepared 2:45 demo and upload it publicly to YouTube.
3. Run Codex `/feedback` in the primary implementation thread and save the returned session ID.
4. Paste `devpost-copy.md`, repository URL, live dashboard, video URL, and session ID into Devpost.
5. Confirm Developer Tools category, MIT license, supported platforms, and judge instructions.
6. Submit before 21 July 2026 at 5:00 PM PDT (22 July at 2:00 AM CEST).
7. Keep the GCP project and Secret Manager configuration available through judging; monitor Cloud SQL billing.

## Final verdict

**Tool requirements: covered for the hackathon MVP.**  
**Submission requirements: blocked only by the video, `/feedback` session ID, and final Devpost form submission.**
