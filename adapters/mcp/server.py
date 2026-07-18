"""Minimal MCP stdio adapter for a compatible Epistemic Protocol server."""
from __future__ import annotations

import json
import os
import sys
from urllib import request

ENDPOINT = os.getenv("EPISTEMIC_ENDPOINT", "http://localhost:8080").rstrip("/")

TOOLS = [
    {"name": "epistemic_get_decision", "description": "Read a portable decision result.", "inputSchema": {"type": "object", "required": ["decision_id"], "properties": {"decision_id": {"type": "string"}}}},
    {"name": "epistemic_get_events", "description": "Read portable decision events.", "inputSchema": {"type": "object", "required": ["decision_id"], "properties": {"decision_id": {"type": "string"}}}},
    {"name": "epistemic_get_certificate", "description": "Read a portable Decision Certificate.", "inputSchema": {"type": "object", "required": ["decision_id"], "properties": {"decision_id": {"type": "string"}}}},
    {
        "name": "epistemic_create_github_pipeline",
        "description": "Generate a vendor-neutral GitHub Actions pipeline that evaluates configured evidence with Epistemic Engine.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "epistemic_action": {"type": "string"},
                "config_path": {"type": "string"},
                "certificate_path": {"type": "string"},
            },
        },
    },
]

def get(path: str):
    with request.urlopen(ENDPOINT + path, timeout=30) as response:
        return json.load(response)

def post(path: str, value: dict):
    body = json.dumps(value).encode("utf-8")
    call = request.Request(ENDPOINT + path, data=body, headers={"Content-Type": "application/json"}, method="POST")
    with request.urlopen(call, timeout=30) as response:
        return json.load(response)

def handle(message: dict):
    method = message.get("method")
    if method == "initialize":
        return {"protocolVersion": "2025-06-18", "capabilities": {"tools": {}}, "serverInfo": {"name": "epistemic-protocol", "version": "0.1.0"}}
    if method == "tools/list": return {"tools": TOOLS}
    if method == "tools/call":
        arguments = message.get("params", {}).get("arguments", {})
        name = message.get("params", {}).get("name")
        if name == "epistemic_create_github_pipeline":
            value = post("/v1/tools/github-actions/pipelines", arguments)
            return {"content": [{"type": "text", "text": json.dumps(value, indent=2)}]}
        decision_id = arguments["decision_id"]
        suffix = {"epistemic_get_decision": "", "epistemic_get_events": "/events", "epistemic_get_certificate": "/certificate"}[name]
        value = get(f"/v1/decisions/{decision_id}{suffix}")
        return {"content": [{"type": "text", "text": json.dumps(value, indent=2)}]}
    raise ValueError(f"unsupported method {method}")

for line in sys.stdin:
    try:
        incoming = json.loads(line)
        result = handle(incoming)
        outgoing = {"jsonrpc": "2.0", "id": incoming.get("id"), "result": result}
    except Exception as exc:
        outgoing = {"jsonrpc": "2.0", "id": incoming.get("id") if "incoming" in locals() else None, "error": {"code": -32603, "message": str(exc)}}
    print(json.dumps(outgoing), flush=True)
