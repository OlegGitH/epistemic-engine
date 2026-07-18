# Context propagation

The portable context contains `decision_id`, `run_id`, `correlation_id`, and `parent_id`. Identifiers are opaque UTF-8 strings and must not contain credentials or personal data.

## In process

SDKs carry the `Context` value explicitly on events and decision requests. Child events copy decision, run, and correlation identifiers and set `parent_id` to the immediate causal event.

## HTTP

The JSON body is authoritative. Clients may additionally send:

```text
Epistemic-Context: decision=<id>;run=<id>;correlation=<id>;parent=<id>
```

Servers fill absent body context from this header but never overwrite explicit body fields. Responses return `Epistemic-Context` when a decision or run context is known. Proxies must forward the header without interpreting identifiers.
