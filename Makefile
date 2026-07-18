.PHONY: dev down bootstrap test test-protocol test-go test-web test-worker test-agent test-sdk test-react protocol-example eval seed-unsafe seed-pending seed-corrected run-api run-web run-relay

dev:
	docker compose up --build

down:
	docker compose down

bootstrap:
	cd apps/web && npm ci
	cd workers/codex-worker && npm ci
	cd sdk/typescript && npm ci
	cd adapters/react && npm ci

test: test-protocol test-go test-web test-worker test-agent test-sdk test-react

test-protocol:
	go test ./...

test-go:
	cd apps/control-plane && go test ./...

test-web:
	cd apps/web && npm run build

test-worker:
	cd workers/codex-worker && npm run build

test-agent:
	cd agents/demo-agent && python -m compileall -q src
	python -m compileall -q adapters/openai-agents adapters/mcp

test-sdk:
	cd sdk/typescript && npm test
	cd sdk/python && python -m unittest discover -s tests

test-react:
	cd adapters/react && npm run build

protocol-example:
	go run ./cmd/epistemic evaluate --config examples/deployment-readiness/.epistemic.yaml

eval:
	cd apps/control-plane && go run ./cmd/eval

seed-unsafe:
	cd apps/control-plane && go run ./cmd/seed --scenario unsafe

seed-pending:
	cd apps/control-plane && go run ./cmd/seed --scenario pending

seed-corrected:
	cd apps/control-plane && go run ./cmd/seed --scenario corrected

run-api:
	cd apps/control-plane && go run ./cmd/server

run-web:
	cd apps/web && npm run dev

run-relay:
	go run ./cmd/epistemic-relay --config deploy/relay.example.yaml
