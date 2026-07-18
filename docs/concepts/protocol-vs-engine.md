# Protocol versus Engine

Epistemic Protocol is the portable contract. Epistemic Engine is a compatible implementation.

| Concern | Protocol | Engine |
|---|---|---|
| Event envelope and lifecycle names | Standard | Consumes and produces them |
| Portable allow/block/indeterminate | Standard | Maps deterministic policy verdicts |
| Context propagation and HTTP binding | Standard | Implements the reference facade |
| JSON schemas and canonical proof hashing | Standard | Uses them at its boundary |
| Claim support score | Extension | Engine-specific explainable dimensions |
| PostgreSQL tables and internal IDs | Excluded | Private implementation detail |
| OpenAI analysis and Codex verification | Excluded | Optional adapters |
| Next.js graph and certificate workspace | Excluded | Reference UI |

Applications should instrument against `api/go` or another language SDK. They should not import `apps/control-plane/internal` or depend on engine JSON representations.
