# PR requirement coverage

The rules analyzer supports `action_type: code_change_review` for evidence-bound review of pull requests, patches, plans, and other text-described changes.

The integration emits three event types in sequence:

1. `requirements.declared` contains the original request and an array of atomic requirements (`id`, `text`, `critical`, and `required_evidence_types`).
2. `change.artifact.observed` records each supplied diff, test, migration, log, trace, or custom artifact under a stable `id` and valid `kind`.
3. `requirement.assessed` records one review outcome (`passed`, `partial`, `missing`, or `failed`), a confidence value, concise rationale, and artifact IDs in `evidence_refs`.

The Engine creates one claim per declared requirement and binds it only to observed artifacts. A passed assessment supports a claim only when confidence is at least 0.80, at least one referenced artifact exists, and all declared evidence kinds are present. The support score remains an explainable evidence score, not a probability that the code is correct.

| Observation | Claim state | Policy effect |
| --- | --- | --- |
| Passed, confidence gate met, all evidence kinds present | `supported` | May proceed after approval |
| Passed but an evidence kind is missing, or explicitly partial | `partially_supported` | `VERIFIED_WITH_CONDITIONS` |
| Missing assessment, low confidence, or no valid artifact reference | `verification_pending` plus a critical unknown | `INSUFFICIENT_EVIDENCE` |
| Failed or contradicted assessment | `contradicted` | `CONTRADICTED`, even with approval |

This separation is intentional: a model may propose coverage and express confidence, but it cannot make its own unsupported statement authoritative. Deterministic policy and human approval remain outside the model.
