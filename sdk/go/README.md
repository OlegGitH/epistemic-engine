# Go SDK

Applications import `github.com/OlegGitH/epistemic-engine/api/go` for stable protocol types, `Client`, and `Provider`. Provider implementations live under `sdk/go/providers`: `remote`, `file`, `noop`, `local`, and `composite`.

```go
provider := remote.New("http://localhost:8080")
client := epistemic.NewClient(provider)
result, err := client.Evaluate(ctx, request)
```

The public API has no dependency on Epistemic Engine or its persistence model.
