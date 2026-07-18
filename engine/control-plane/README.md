# Reference Engine control plane

The active Go reference implementation remains at `apps/control-plane` to preserve the existing product and Compose paths. It is a separate Go module and imports the vendor-neutral public API only through its protocol HTTP adapter.

This directory reserves the protocol repository layout boundary. A future repository extraction can move the nested Engine module here without changing any protocol package or SDK import.
