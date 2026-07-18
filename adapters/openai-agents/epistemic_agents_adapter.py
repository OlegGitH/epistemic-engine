"""Map observable OpenAI Agents SDK lifecycle metadata to Epistemic Protocol.

This adapter intentionally accepts plain metadata dictionaries. It does not
depend on the protocol core and never serializes model input/output or hidden
reasoning.
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Any


def trace_event(*, event_id: str, event_type: str, subject_type: str, subject_id: str,
                decision_id: str, run_id: str, correlation_id: str,
                data: dict[str, Any], parent_id: str | None = None,
                sequence: int = 0) -> dict[str, Any]:
    allowed = {
        "tool.started": "verification.started",
        "tool.completed": "verification.completed",
        "tool.failed": "verification.failed",
        "guardrail.blocked": "decision.blocked",
        "approval.requested": "verification.requested",
        "approval.approved": "verification.approved",
    }
    protocol_type = allowed.get(event_type, "evidence.discovered")
    safe_data = {key: value for key, value in data.items() if key in {
        "trace_id", "span_id", "tool_name", "status", "error_type",
        "workflow_name", "approval_required", "duration_ms"
    }}
    return {
        "spec_version": "0.1",
        "id": event_id,
        "type": protocol_type,
        "source": {"name": "openai-agents-sdk-adapter", "version": "0.1"},
        "subject": {"type": subject_type, "id": subject_id},
        "time": datetime.now(timezone.utc).isoformat(),
        "context": {
            "decision_id": decision_id,
            "run_id": run_id,
            "correlation_id": correlation_id,
            **({"parent_id": parent_id} if parent_id else {}),
        },
        "ordering": {"sequence": sequence, "partition": run_id},
        "idempotency_key": event_id,
        "data": safe_data,
    }
