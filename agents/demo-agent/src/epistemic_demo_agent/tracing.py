from __future__ import annotations

import threading
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any

from agents.tracing import TracingProcessor

from .exporter import ControlPlaneClient


@dataclass
class _QueuedEvent:
    external_id: str
    event_type: str
    payload: dict[str, Any]


class ControlPlaneTraceProcessor(TracingProcessor):
    """Exports observable trace metadata without generation/tool contents.

    The SDK invokes processor hooks synchronously, so hooks only enqueue small
    records. Network work happens in ``force_flush`` after the agent run.
    """

    def __init__(self, client: ControlPlaneClient) -> None:
        self._client = client
        self._run_id: str | None = None
        self._events: list[_QueuedEvent] = []
        self._lock = threading.Lock()

    def bind(self, run_id: str) -> None:
        self._run_id = run_id

    def on_trace_start(self, trace: Any) -> None:
        self._enqueue(
            f"trace-start:{trace.trace_id}",
            "trace.started",
            {"trace_id": trace.trace_id, "workflow_name": getattr(trace, "name", "agent-workflow")},
        )

    def on_trace_end(self, trace: Any) -> None:
        self._enqueue(f"trace-end:{trace.trace_id}", "trace.completed", {"trace_id": trace.trace_id})

    def on_span_start(self, span: Any) -> None:
        span_data = getattr(span, "span_data", None)
        self._enqueue(
            f"span-start:{span.span_id}",
            f"{getattr(span_data, 'type', 'custom')}.started",
            {"trace_id": span.trace_id, "span_id": span.span_id, "parent_id": span.parent_id},
        )

    def on_span_end(self, span: Any) -> None:
        span_data = getattr(span, "span_data", None)
        error = getattr(span, "error", None)
        self._enqueue(
            f"span-end:{span.span_id}",
            f"{getattr(span_data, 'type', 'custom')}.completed",
            {
                "trace_id": span.trace_id,
                "span_id": span.span_id,
                "status": "failed" if error else "completed",
                "error_type": type(error).__name__ if error else None,
            },
        )

    def force_flush(self) -> None:
        if not self._run_id:
            return
        with self._lock:
            pending, self._events = self._events, []
        for sequence, event in enumerate(pending, start=1):
            self._client.append_event(
                self._run_id,
                event.event_type,
                event.payload,
                external_id=event.external_id,
                sequence=sequence,
            )

    def shutdown(self) -> None:
        self.force_flush()

    def _enqueue(self, external_id: str, event_type: str, payload: dict[str, Any]) -> None:
        payload["observed_at"] = datetime.now(timezone.utc).isoformat()
        with self._lock:
            self._events.append(_QueuedEvent(external_id, event_type, payload))
