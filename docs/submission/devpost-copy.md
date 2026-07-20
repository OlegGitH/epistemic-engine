# Devpost submission copy

## Project name

Epistemic Engine

## Tagline

Evidence before autonomous action.

## Category

Developer Tools

## Inspiration

AI agents can propose code changes, deployments and operational actions faster than teams can verify the assumptions behind them. Existing observability shows what happened; it rarely establishes whether a recommended action is justified. We built Epistemic Engine to place an evidence boundary between an agent recommendation and a consequential action.

## What it does

Epistemic Engine turns a recommendation into atomic observable claims, binds typed evidence to those claims, keeps contradictions, assumptions and unknowns visible, runs only approval-bounded verification, applies deterministic policy and issues a content-addressed Decision Certificate.

The dashboard gives operators two views of the same result: a detailed evidence graph for investigation and a plain-language report that says whether to proceed, why, what remains unresolved and how to verify the immutable certificate digest.

Food Lens demonstrates the complete lifecycle. Its branch lab intentionally produces four outcomes: a supported release, insufficient evidence, a privacy contradiction and verification with remaining approval conditions. Each branch is successful only when the Engine reaches the expected result and persists it across a PostgreSQL restart.

## How we built it

The product combines a vendor-neutral protocol and SDKs, a Go control plane, PostgreSQL, a Next.js dashboard, GitHub Actions adapters, a restricted Docker verifier, an OpenAI Responses API analyzer, and an approval-gated Codex SDK worker.

GPT-5.6 proposes a strict structured decomposition into observable claims. Its output is treated as untrusted input. Deterministic application policy—not the model—decides whether an action is allowed. The Codex worker operates in a disposable checkout with network access disabled, can edit only `tests/`, and emits an unapplied patch and digest after explicit approval.

## How we used Codex

Codex helped translate the original product decisions and ADR into a working multi-language implementation. It accelerated scaffolding, API and schema alignment, test creation, CI diagnosis, database persistence, GCP packaging, scenario design, and the final UI/UX refinement. We kept human ownership over product scope, protocol boundaries, approval rules, privacy constraints and deployment decisions.

The most valuable part of the collaboration was not raw code generation; it was maintaining a continuous loop from architectural claim to executable evidence. When persistence, run inspection or CI behavior did not match the intended product promise, Codex traced the failure, changed the narrowest component and reran the full path.

## Challenges

The hardest design problem was preventing the model integration from becoming the authority it was supposed to audit. We separated proposal, evidence, verification and policy. GPT-5.6 can suggest claims, but it cannot invent support. Codex can create a test patch, but it cannot apply it. Human approval is recorded but cannot override a contradiction.

We also had to make blocked outcomes understandable. The same machine certificate now drives a human report, while the dashboard preserves the evidence graph and certificate digest for deeper inspection.

## Accomplishments

- Portable ordered and idempotent evidence protocol
- Explicit claim/evidence/contradiction graph
- Approval-bounded verification with hashed artifacts
- Deterministic policy and immutable certificates
- Durable account, project, run and certificate state
- Human-readable decision reports
- Four branch-specific outcomes plus a full-scope CI harness
- OpenAI Responses API and Codex SDK integrations with narrow authority
- Complete local Docker stack and Google Cloud deployment package

## What we learned

AI safety in developer tooling becomes more practical when expressed as ordinary engineering constraints: explicit inputs, typed evidence, content hashes, bounded permissions, deterministic policies and reviewable outputs. The model is most useful when it helps structure uncertainty—not when it hides uncertainty behind confidence.

## What's next

Next we will add provider conformance certification, identity-aware production access, richer policy packs, long-term evidence retention controls and integrations for additional CI, agent and observability systems.

## Links

- Source: https://github.com/OlegGitH/epistemic-engine
- Food Lens demo: https://github.com/OlegGitH/epistemic-engine-demo
- Live dashboard: add after deployment
- Demo video: add after upload
