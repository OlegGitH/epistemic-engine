# Compatibility policy

Protocol versions use `major.minor` semantics.

- A major change may remove fields, change meaning, or make a previously valid message invalid.
- A minor change may add optional fields, event types, reason codes, and capabilities.
- Consumers must ignore unknown optional fields while preserving them when acting as a relay.
- Unknown event types must never cause an action to be allowed. Enforcing consumers return `indeterminate` unless their policy explicitly treats the event as irrelevant.
- Required fields are never added in a minor release to an existing schema.
- A deprecated field remains valid for at least the rest of its major series and is annotated in the schema and changelog.
- Extensions use reverse-domain keys, for example `com.example.score`, and cannot override standard fields.

A compatible v0.1 implementation exposes `0.1` through discovery and passes the protocol conformance fixtures for its claimed features.
