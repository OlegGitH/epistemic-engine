# Development guide

## Toolchain

- Go 1.25
- Node.js 20+ and npm
- Python 3.10+
- Docker Desktop with Compose for PostgreSQL and sandbox execution

Install JavaScript dependencies with `make bootstrap`; Go dependencies are resolved from `apps/control-plane/go.mod`. Install the optional agent in a virtual environment with `pip install -e agents/demo-agent`.

## Fast offline path

Run the protocol, provider, and conformance suites from the repository root:

```bash
go test ./...
go run ./cmd/epistemic evaluate --config examples/deployment-readiness/.epistemic.yaml
```

The root Go module is vendor-neutral. `apps/control-plane` is a separate nested Engine module joined through `go.work` for local development.

Run the API from `apps/control-plane`:

```bash
go run ./cmd/server
```

Run the web app from `apps/web`:

```bash
npm run dev
```

Seed and analyze a durable story while the API is running:

```bash
cd apps/control-plane
go run ./cmd/seed --scenario unsafe
go run ./cmd/seed --scenario pending
```

Without `DATABASE_URL`, the API uses the in-memory adapter for short-lived development. Set `REQUIRE_DURABLE_STORAGE=true` to make startup fail instead of silently accepting that fallback. The Compose and GCP deployment paths set both `DATABASE_URL` and `REQUIRE_DURABLE_STORAGE=true`.

Verify the active backend before a persistence test:

```bash
curl http://localhost:8080/health
```

A durable response contains `{"status":"ok","storage":"postgresql","durable":true}`. Account-linked runs, claim/evidence state, decisions, proofs, and published certificates then survive API restarts.

The portable CLI can write a human-readable companion to its immutable JSON certificate:

```yaml
outputs:
  result: .epistemic/result.json
  certificate: .epistemic/certificate.json
  report: .epistemic/certificate-report.md
```

## Analyzer modes

- `ANALYZER_MODE=rules`: deterministic, offline, and used by tests/evals.
- `ANALYZER_MODE=openai`: official `openai-go` Responses API with a strict schema for 3–7 claims.

Model refusals, incomplete responses, invalid JSON, invalid states, missing critical claims, and request timeouts are rejected. Critical claims cannot become externally verified from model output.

## Verification modes

- `EXECUTION_MODE=recorded`: default; an approved caller records a sandbox artifact. This is the deterministic demo fallback.
- `EXECUTION_MODE=docker`: the API executes the generated bounded specification using the allowlisted image under restricted Docker flags.

Configure `VERIFICATION_ROOT` and comma-separated `VERIFICATION_IMAGES`. The root must contain the planned repository; the default local layout expects `../../demo/unsafe-orders-pr` when the API starts from `apps/control-plane`.

The Codex worker is intentionally separate:

```bash
cd workers/codex-worker
npm run build
node dist/main.js --approved --repository ../../demo/unsafe-orders-pr --specification ../../demo/verification-spec.json --output ../../demo/recorded/codex-patch.json
```

It requires `OPENAI_API_KEY`, creates a disposable checkout, disables network/web search, permits only `tests/` changes, and writes an unapplied JSON patch artifact.

## Demo agent

The Agents SDK reviewer exports observable events only. If its approval-gated patch tool interrupts, the CLI serializes resumable state; resume with `--state-file ... --approve` or `--reject`. See `agents/demo-agent/README.md` for commands.

## Verification commands

```bash
make test
make eval
```

To verify the demo fixture manually, first confirm its tests fail, apply `demo/corrected-orders.patch` inside a disposable copy, and confirm the same tests pass. Never apply the corrected patch to the canonical unsafe fixture if you want to preserve the demo.

PostgreSQL initialization scripts run only for an empty volume. Use `docker compose down -v` after a migration rewrite in local development.
