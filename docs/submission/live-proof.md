# Live GPT-5.6 and Codex proof

This procedure creates judge-safe artifacts without storing credentials or hidden reasoning.

## Prerequisites

- Set `OPENAI_API_KEY` in the current shell. Never write it to a file or commit it.
- Install Go 1.25, Node.js 22 or newer, npm and Git.
- Stop any process using port `8081`.

## Run

```powershell
$env:OPENAI_API_KEY = "set-this-locally"
./scripts/run-submission-proof.ps1
```

The script starts an isolated memory-backed control plane on port `8081` with `ANALYZER_MODE=openai` and `OPENAI_MODEL=gpt-5.6`, creates a real run with typed evidence, invokes GPT-5.6 claim analysis, evaluates the decision, downloads its human report, and then invokes the official Codex SDK worker after explicit approval.

Outputs are stored under `.cache/submission/`:

- `openai-proof.json` — model, graph, certificate and proof identifiers
- `openai-certificate-report.md` — human-readable result
- `codex-proof.json` — Codex thread ID, bounded changed files and patch digest
- `control-plane.stderr.log` — credential-free server diagnostics

Before publishing, inspect each artifact for repository-specific data. Do not publish prompts, API keys, hidden reasoning, user content or sensitive trace payloads.
