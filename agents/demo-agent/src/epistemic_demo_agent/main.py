from __future__ import annotations

import argparse
import asyncio
import json
import os
from pathlib import Path

from agents import Agent, RunConfig, Runner, RunState, add_trace_processor, function_tool

from .exporter import ControlPlaneClient
from .tracing import ControlPlaneTraceProcessor


@function_tool(needs_approval=True)
async def propose_targeted_patch(defect: str) -> str:
    """Propose a patch request; execution always pauses for human approval."""
    return f"Approved patch request for one bounded defect: {defect}"


async def run(args: argparse.Namespace) -> None:
    client = ControlPlaneClient(args.control_plane_url)
    trace_exporter = ControlPlaneTraceProcessor(client)
    add_trace_processor(trace_exporter)
    reviewer = Agent(
        name="Deployment reviewer",
        model=os.getenv("OPENAI_MODEL", "gpt-5.6"),
        instructions=(
            "Review the supplied deployment observations. Return a concise final deployment "
            "recommendation with explicit observable reasons. Do not reveal hidden chain-of-thought. "
            "Never claim a patch was applied. If asked to request a patch, call propose_targeted_patch."
        ),
        tools=[propose_targeted_patch] if args.request_patch else [],
    )
    status = "failed" if args.simulate_failure else "passed"
    prompt = (
        f"Repository: {args.repository}\nRevision: {args.revision}\n"
        f"Build status: passed\nUnit tests: passed\nCompatibility test: {status}\n"
        "Should this revision deploy?"
    )
    if args.request_patch:
        prompt += "\nRequest a targeted patch for the first blocking defect."
    run_config = RunConfig(workflow_name="Epistemic deployment reviewer", trace_include_sensitive_data=False)
    if args.resume_state:
        stored = Path(args.resume_state).read_text(encoding="utf-8")
        state = await RunState.from_string(reviewer, stored)
        result = await Runner.run(reviewer, state, run_config=run_config)
    else:
        result = await Runner.run(reviewer, prompt, run_config=run_config)
    while result.interruptions:
        state = result.to_state()
        Path(args.state_path).parent.mkdir(parents=True, exist_ok=True)
        Path(args.state_path).write_text(state.to_string(), encoding="utf-8")
        if args.approval == "pause":
            print(json.dumps({"status": "approval_required", "state_path": args.state_path, "tools": [item.name for item in result.interruptions]}, indent=2))
            return
        for interruption in result.interruptions:
            if args.approval == "approve":
                state.approve(interruption, always_approve=False)
            else:
                state.reject(interruption, rejection_message="The human reviewer rejected patch generation.")
        result = await Runner.run(reviewer, state, run_config=run_config)
    recommendation = str(result.final_output)

    created = client.create_run(
        recommendation=recommendation,
        repository=args.repository,
        revision=args.revision,
    )
    run_id = created["id"]
    trace_exporter.bind(run_id)
    trace_exporter.force_flush()
    client.append_event(run_id, "build.completed", {"status": "passed", "revision": args.revision}, external_id=f"build:{args.revision}")
    client.append_event(run_id, "test.completed", {"suite": "unit", "status": "passed", "revision": args.revision}, external_id=f"unit:{args.revision}")
    client.append_event(
        run_id,
        "test.completed",
        {
            "suite": "compatibility",
            "status": status,
            "revision": args.revision,
            **({"failure": "legacy status value cannot be decoded"} if args.simulate_failure else {}),
        },
        external_id=f"compatibility:{args.revision}",
    )
    if args.simulate_failure:
        client.append_event(run_id, "code.diff", {"file": "orders/service.py", "added": "logger.info('customer_email=%s', customer.email)", "classification": "pii"}, external_id=f"diff:{args.revision}")
    graph = client.analyze(run_id)
    print(json.dumps({"run_id": run_id, "decision_id": graph["decision"]["id"], "recommendation": recommendation}, indent=2))


def cli() -> None:
    parser = argparse.ArgumentParser(description="Run and export the Epistemic Engine demo agent")
    parser.add_argument("--repository", default="example/checkout")
    parser.add_argument("--revision", default="demo-sha")
    parser.add_argument("--control-plane-url", default=os.getenv("CONTROL_PLANE_URL", "http://localhost:8080"))
    parser.add_argument("--simulate-failure", action="store_true")
    parser.add_argument("--request-patch", action="store_true")
    parser.add_argument("--approval", choices=["pause", "approve", "reject"], default="pause")
    parser.add_argument("--state-path", default=".cache/epistemic-review-state.json")
    parser.add_argument("--resume-state")
    asyncio.run(run(parser.parse_args()))


if __name__ == "__main__":
    cli()
