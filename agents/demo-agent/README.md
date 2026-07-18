# Demo reviewer agent

This thin OpenAI Agents SDK runtime reviews deployment observations, requests an approval-gated bounded patch when asked, and exports observable trace metadata and CI artifacts into the Go control plane.

It does not export prompts, model/tool contents, or hidden chain-of-thought. `trace_include_sensitive_data` is disabled and the custom processor emits identifiers, span types, completion state, and error type only.

```powershell
python -m venv .venv
.venv\Scripts\Activate.ps1
pip install -e .
$env:OPENAI_API_KEY = "..."
epistemic-demo-agent --repository demo/unsafe-orders-pr --revision demo-sha --simulate-failure
```

To demonstrate human-in-the-loop pause/resume:

```powershell
epistemic-demo-agent --request-patch --state-path .cache/reviewer-state.json
epistemic-demo-agent --request-patch --resume-state .cache/reviewer-state.json --approval approve
```

Use `--approval reject` to resume with a rejection. A paused run does not call or simulate patch application. The agent only proposes a bounded request; the separate Codex worker creates an unapplied test patch after its own explicit approval.
