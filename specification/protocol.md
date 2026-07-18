# Epistemic Protocol v0.1

Epistemic Protocol is a vendor-neutral contract for exchanging observable claims, evidence, verification, and portable workflow decisions. Epistemic Engine is one reference implementation.

## Boundary

The protocol has no required OpenTelemetry, OpenAI, database, UI, or policy-engine dependency. It does not standardize hidden reasoning. Implementations may expose proprietary scoring only through namespaced extensions; portable clients must rely on status, reasons, conditions, events, and proofs.

## Messages

- Universal `Event` envelope.
- `DecisionRequest` and `DecisionResult`.
- Immutable `Certificate` with canonical SHA-256 proof.
- Structured protocol `Error`.
- Capability discovery document.

Every message carries `spec_version: "0.1"`. Event data is a JSON value whose semantics are selected by the event type. Vendor additions belong in reverse-domain-named `extensions` keys.

## Invariants

1. Event IDs are globally unique within a producer namespace and are idempotency keys by default.
2. Duplicate event IDs must not create duplicate effects.
3. Sequence values are monotonic within an ordering partition when present.
4. Parent IDs must refer to an earlier or externally known event.
5. Correlation, run, and decision context survives every transport hop.
6. Unknown event types from a later minor version are retained or ignored safely; they never imply approval.
7. `allow` does not imply `action_allowed: true` without implementation-required approval.
8. Certificates are immutable. Revocation and replacement are expressed with protocol events.

Canonical schemas under `schemas/v0.1/` are normative. This prose explains their intended semantics.
