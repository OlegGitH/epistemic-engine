# HTTP/JSON binding

Servers accept UTF-8 `application/json` and return protocol `Error` objects for non-2xx responses.

| Method | Endpoint | Success |
|---|---|---|
| POST | `/v1/events` | `202` event accepted or deduplicated |
| POST | `/v1/events:batch` | `202` per-event acceptance results |
| POST | `/v1/decisions:evaluate` | `200` portable decision result |
| GET | `/v1/decisions/{id}` | `200` portable decision result |
| GET | `/v1/decisions/{id}/events` | `200` ordered event collection |
| GET | `/v1/decisions/{id}/certificate` | `200` immutable certificate |
| GET | `/.well-known/epistemic` | `200` capabilities |
| GET | `/v1/stream` | `200` server-sent protocol events |

Batch requests contain `{"events": [...]}` and may contain at most the advertised discovery limit. The response contains `accepted`, `duplicate`, and `errors` arrays. SSE messages use the protocol event type as the SSE `event` field and the complete event envelope as `data`.

Stable error codes include `invalid_message`, `unsupported_version`, `unsupported_event_type`, `not_found`, `conflict`, `limit_exceeded`, `evaluation_failed`, and `temporarily_unavailable`.
