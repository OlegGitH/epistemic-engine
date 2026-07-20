# Codex verification worker

This worker gives the official Codex SDK one approved, bounded verification specification. It copies the repository to a disposable workspace, disables network access, permits writes only inside that copy, rejects changes outside `tests/`, enforces the requested test file when one is specified, and emits a patch artifact without applying it.

```bash
npm install
npm run build
node dist/main.js \
  --approved \
  --repository ../../demo/unsafe-orders-pr \
  --specification ../../demo/verification-spec.json \
  --output ../../demo/recorded/codex-patch.json
```

Omitting `--approved` is a hard failure. Applying the emitted patch is a separate consequential action and is never performed by this worker.
