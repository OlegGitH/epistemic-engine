from __future__ import annotations

import hashlib
import json
from pathlib import Path
from typing import Any
from urllib import request

SPEC_VERSION = "0.1"
EVENT_TYPES = {"decision.started","decision.requested","decision.evaluated","decision.blocked","decision.approved","decision.completed","claim.declared","claim.updated","claim.supported","claim.contradicted","claim.superseded","claim.rejected","evidence.discovered","evidence.attached","evidence.expired","evidence.invalidated","assumption.declared","assumption.resolved","unknown.declared","unknown.resolved","contradiction.detected","contradiction.resolved","verification.requested","verification.approved","verification.started","verification.completed","verification.failed","proof.issued","proof.revoked","proof.superseded"}

def validate_event(event: dict[str, Any]) -> None:
    if event.get("spec_version") != SPEC_VERSION:
        raise ValueError("unsupported spec_version")
    if not event.get("id") or event.get("type") not in EVENT_TYPES:
        raise ValueError("invalid event identity or type")
    if not event.get("source", {}).get("name") or not event.get("subject", {}).get("type") or not event.get("subject", {}).get("id") or not event.get("time") or "data" not in event:
        raise ValueError("invalid protocol event")

def canonical_json(value: Any) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)

def hash_value(value: Any) -> str:
    return hashlib.sha256(canonical_json(value).encode("utf-8")).hexdigest()

class RemoteProvider:
    def __init__(self, endpoint: str): self.endpoint = endpoint.rstrip("/")
    def emit(self, event: dict[str, Any]) -> None: validate_event(event); self._post("/v1/events", event)
    def evaluate(self, decision: dict[str, Any]) -> dict[str, Any]: return self._post("/v1/decisions:evaluate", decision)
    def flush(self) -> None: pass
    def shutdown(self) -> None: pass
    def _post(self, path: str, value: Any) -> dict[str, Any]:
        call = request.Request(self.endpoint + path, data=json.dumps(value).encode(), method="POST", headers={"Content-Type":"application/json"})
        with request.urlopen(call, timeout=30) as response: return json.load(response)

class FileProvider:
    def __init__(self, path: str): self.path = Path(path)
    def emit(self, event: dict[str, Any]) -> None: validate_event(event); self._write("event", event)
    def evaluate(self, decision: dict[str, Any]) -> dict[str, Any]:
        self._write("decision_request", decision)
        return {"spec_version":SPEC_VERSION,"decision_id":decision.get("decision_id","offline"),"status":"indeterminate","action_allowed":False,"reasons":[{"code":"provider_offline","message":"File provider does not evaluate."}],"conditions":[],"evaluated_at":"1970-01-01T00:00:00Z"}
    def flush(self) -> None: pass
    def shutdown(self) -> None: pass
    def _write(self, kind: str, value: Any) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self.path.open("a", encoding="utf-8") as stream: stream.write(json.dumps({"kind":kind,"value":value}, separators=(",",":")) + "\n")

class NoopProvider:
    def emit(self, event: dict[str, Any]) -> None: pass
    def evaluate(self, decision: dict[str, Any]) -> dict[str, Any]: return {"spec_version":SPEC_VERSION,"decision_id":decision.get("decision_id","noop"),"status":"indeterminate","action_allowed":False,"reasons":[{"code":"provider_disabled","message":"Epistemic evaluation is disabled."}],"conditions":[],"evaluated_at":"1970-01-01T00:00:00Z"}
    def flush(self) -> None: pass
    def shutdown(self) -> None: pass
