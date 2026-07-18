# Epistemic Protocol RFC process

Semantic additions start as an RFC under this directory.

1. Copy `0000-template.md` and assign the next available number.
2. State the interoperability problem and why an extension is insufficient.
3. Include compatibility, privacy, security, schema, and conformance impact.
4. Gather implementation feedback from at least two independent consumers or providers.
5. Mark the RFC Accepted, Rejected, or Deferred.
6. Accepted changes update specification, schemas, SDKs, fixtures, and changelog together.

Minor versions cannot add required fields to existing messages. Major changes require a migration document and a new schema directory.
