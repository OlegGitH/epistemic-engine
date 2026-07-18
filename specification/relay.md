# Relay pipeline configuration

The optional relay uses protocol-native receivers, processors, and exporters. Its `epistemic.dev/relay/v1alpha1` YAML shape is:

```yaml
api_version: epistemic.dev/relay/v1alpha1
receivers:
  http:
    listen: :8090
  file:
    watch: ./artifacts
processors:
  redact:
    keys: token,password,secret,authorization,api_key
  batch:
    max_size: 100
exporters:
  archive:
    path: .epistemic/relay.jsonl
  engine:
    endpoint: http://localhost:8080
```

The relay validates every event, recursively redacts configured JSON keys, archives JSONL, and retries the compatible engine exporter. The file receiver content-addresses new or changed files and emits `evidence.discovered` without interpreting vendor internals.
