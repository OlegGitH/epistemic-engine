# Epistemic CLI

```bash
epistemic evaluate --config .epistemic.yaml
epistemic version
```

Exit codes for a compiled CLI are stable: `0` success/non-enforcing report, `2` blocked or unapproved enforcement, `3` indeterminate enforcement, and `4` configuration/protocol/runtime failure. `go run` reports non-zero program exits through Go's wrapper and may itself return `1`; use a built binary in CI enforcement.
