# Deployment-readiness example

From the repository root:

```bash
go run ./cmd/epistemic evaluate --config examples/deployment-readiness/.epistemic.yaml
```

The failing JUnit artifact becomes a portable `verification.failed` event. The local provider returns `block`, while advise mode reports the decision without failing the shell. Change the fixture to zero failures to observe the portable allow path.
