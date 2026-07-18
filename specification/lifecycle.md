# Lifecycle

Events use `<family>.<verb>` names. Producers emit only observable state transitions.

| Family | v0.1 events |
|---|---|
| decision | started, requested, evaluated, blocked, approved, completed |
| claim | declared, updated, supported, contradicted, superseded, rejected |
| evidence | discovered, attached, expired, invalidated |
| assumption | declared, resolved |
| unknown | declared, resolved |
| contradiction | detected, resolved |
| verification | requested, approved, started, completed, failed |
| proof | issued, revoked, superseded |

No sequence is mandatory beyond the semantic constraints of a particular workflow. Consumers must tolerate missing optional stages, duplicated delivery, interleaving partitions, and late evidence. A terminal event cannot erase earlier contradictions; resolution requires an explicit resolving or superseding event.
