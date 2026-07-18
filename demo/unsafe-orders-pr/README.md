# Unsafe orders pull request

This fixture intentionally contains two deployment-blocking defects:

1. `migrate_status()` no longer accepts the legacy `processing` value.
2. `process_order()` writes the customer's email address into application logs.

Run the bounded checks directly:

```bash
python -m unittest discover -s tests -v
```

Both checks must fail in the unsafe state. Apply `../corrected-orders.patch` to a disposable copy to demonstrate the corrected, verified state.
