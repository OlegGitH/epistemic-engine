# Conformance

- `protocol/` validates public types, eight event families, shared fixtures, canonical hashing, and portable examples.
- `providers/` exports a reusable Provider contract suite and runs it against Remote, File, No-op, and Local providers.
- `fixtures/` covers valid, invalid, duplicate, ordering, parent, evaluation, and proof cases.
- The reference Engine runs facade conformance cases in `apps/control-plane/internal/httpapi/protocol_test.go`.

Passing the claimed v0.1 suites is required before displaying `docs/compatibility-badge.svg`.
