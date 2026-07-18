from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any
from urllib import error, request


@dataclass(frozen=True)
class ControlPlaneClient:
    base_url: str = "http://localhost:8080"

    def create_run(self, *, recommendation: str, repository: str, revision: str) -> dict[str, Any]:
        return self._post(
            "/v1/runs",
            {
                "title": f"{repository} deployment readiness",
                "source": "openai-agents-sdk-demo",
                "recommendation": recommendation,
                "raw": {"repository": repository, "revision": revision},
            },
        )

    def append_event(
        self,
        run_id: str,
        event_type: str,
        payload: dict[str, Any],
        *,
        external_id: str | None = None,
        sequence: int = 0,
    ) -> dict[str, Any]:
        return self._post(
            f"/v1/runs/{run_id}/events",
            {
                "type": event_type,
                "external_id": external_id or "",
                "sequence": sequence,
                "source": "demo-agent",
                "occurred_at": datetime.now(timezone.utc).isoformat(),
                "payload": payload,
            },
        )

    def analyze(self, run_id: str) -> dict[str, Any]:
        return self._post(f"/v1/runs/{run_id}/analyze", None)

    def _post(self, path: str, payload: dict[str, Any] | None) -> dict[str, Any]:
        body = b"" if payload is None else json.dumps(payload).encode("utf-8")
        call = request.Request(
            f"{self.base_url.rstrip('/')}{path}",
            data=body,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        try:
            with request.urlopen(call, timeout=60) as response:
                return json.load(response)
        except error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"control plane returned HTTP {exc.code}: {detail}") from exc
