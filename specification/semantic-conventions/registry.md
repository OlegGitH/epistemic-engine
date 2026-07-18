# Semantic conventions registry

## Decision status

- `allow`: portable requirements passed; local approval may still be required.
- `block`: a standard blocking reason is active.
- `indeterminate`: available information cannot safely produce allow or block.
- `error`: evaluation did not complete.

## Standard reason codes

- `contradicted`
- `insufficient_evidence`
- `verification_required`
- `rejected`
- `approval_required`
- `policy_error`
- `provider_offline`
- `provider_disabled`

## Risk

`low`, `medium`, `high`, `critical`.

## Evidence types

`build`, `test`, `migration`, `diff`, `log`, `trace`, `sarif`, `junit`, `tool`, `human_attestation`, `custom`.

## Relations

`supports`, `contradicts`, `qualifies`, `supersedes`, `derived_from`, `verified_by`.

New standard values require an RFC. Vendor-specific values must be namespaced.
