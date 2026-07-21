# Epistemic Engine demo media package

These frames are captured from the deployed application and the public repositories. Use them as a complete fallback slideshow or as clean cutaways inside a live screen recording.

## Recommended 2:45 edit

| Time | Asset | Screen direction | Narration |
| --- | --- | --- | --- |
| 0:00-0:18 | `00-food-lens-demo.png` | Start full frame, then slowly zoom toward the score and visible signals | "AI can make a recommendation quickly. The hard problem is proving whether an action is justified. Food Lens is our transparent demo application: raw images stay in the browser and the model exposes its inputs, confidence, and limitations." |
| 0:18-0:38 | `01-live-gpt-overview.png` | Crossfade to the live account. Pan from the `openai` workspace label to knowledge coverage and five decisions | "Epistemic Engine adds an evidence control plane between an AI recommendation and a consequential action. This is a real GPT-5.6 run persisted in PostgreSQL, with five decision certificates." |
| 0:38-1:02 | `02-fully-covered-proceed.png` | Zoom from `PR 101` to `PROCEED`, then the human report | "For a fully covered change, the Engine maps each request requirement to direct artifacts. Deterministic policy records human approval and authorizes the action. The model proposes assessments; it never owns authorization." |
| 1:02-1:27 | `03-contradicted-blocked.png` | Hold the `DO NOT PROCEED` status, then zoom to the contradicted claim | "When direct test evidence contradicts the recommendation, the action is blocked even though human approval was granted. Approval cannot override a critical evidence gate." |
| 1:27-1:48 | `04-confidence-only-blocked.png` | Zoom from the title to `0/1 supported` and the verification-pending claim | "High model confidence is not evidence. Without a valid artifact reference, the claim remains open and the Engine refuses the action." |
| 1:48-2:08 | `05-partial-coverage-blocked.png` | Pan across the three claim states: pending, supported, partially supported | "Partial implementation is also visible. The core cursor change exists, but compatibility and migration evidence remain incomplete, so the merge stays blocked." |
| 2:08-2:30 | `06-evidence-graph-and-report.png` | Use a vertical pan: human report, claim cards, then evidence relationship map | "Operators can inspect the readable decision, exact claim graph, bound evidence, support dimensions, remaining gates, and the SHA-256 certificate digest. PostgreSQL restart tests prove this state persists." |
| 2:30-2:40 | `07-codex-proof.png` | Zoom into `worker`, `changed_files`, `approval_recorded`, and `applied: false` | "After explicit approval, the official Codex SDK creates one targeted verification test in a disposable checkout. Network access is disabled, scope is enforced, and the patch is never applied automatically." |
| 2:40-2:45 | `01-live-gpt-overview.png` | Return to the overview and fade to black | "Evidence before autonomous action." |

## Files

- `00-food-lens-demo.png` - transparent demo application and healthy sample result
- `01-live-gpt-overview.png` - live GPT-5.6 account portfolio, knowledge coverage, and five decisions
- `02-fully-covered-proceed.png` - successful PR requirement coverage and authorized action
- `03-contradicted-blocked.png` - contradiction overriding human approval
- `04-confidence-only-blocked.png` - confidence rejected when evidence is missing
- `05-partial-coverage-blocked.png` - incomplete requirement coverage remaining blocked
- `06-evidence-graph-and-report.png` - human report, atomic claims, evidence map, and support explanation
- `07-codex-proof.png` - public Codex SDK proof metadata and bounded patch controls

## Best live-recording inserts

If recording the browser, replace three static sections with live motion:

1. Open the [live GPT-5.6 account](https://epistemic-dashboard-r7zqwwvzgq-ew.a.run.app/?account=acc_cb17e3c15b4bcb1c1dfc27f3), then open the latest decision.
2. Compare the [allowed run](https://epistemic-dashboard-r7zqwwvzgq-ew.a.run.app/run?run=run_42bbb24ccca101aff270ca53) with the [contradicted run](https://epistemic-dashboard-r7zqwwvzgq-ew.a.run.app/run?run=run_213de2d122425564a8a9cd84).
3. Scroll from the human report to the certificate digest and evidence graph. Do not spend video time on setup forms.

## Editing settings

- Canvas: 1920x1080, 30 fps.
- Keep browser content at 100% zoom and crop unused browser chrome.
- Use slow 4-8% zooms rather than rapid cursor movement.
- Use cuts or short 200 ms dissolves; avoid decorative transitions.
- Keep narration louder than any background music. Copyright-free music is optional.
- Add short labels only: `LIVE GPT-5.6`, `DETERMINISTIC POLICY`, `POSTGRESQL`, `CODEX SDK`.
- Export H.264 MP4 at 1080p, then verify the public YouTube version is readable at 720p.
- Keep the final cut below three minutes. Aim for 2:45-2:50.

## Required spoken claims

The published challenge asks the video audio to explain both GPT-5.6 and Codex. Say these points explicitly:

- GPT-5.6 is called through the Responses API with strict structured output.
- Model output is untrusted and cannot authorize an action.
- Codex creates one targeted verification test only after approval.
- Deterministic application policy, human approval, and content-addressed evidence own the final decision.
