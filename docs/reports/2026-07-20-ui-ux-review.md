# Control Center UI/UX review

**Review date:** 20 July 2026  
**Scope:** Account portfolio and run-level decision review

## Product task

The primary user is a human reviewer deciding whether an AI-assisted action may proceed. The interface must answer, in order:

1. What is the proposed action?
2. May it proceed?
3. Why did the Engine reach that decision?
4. What remains unresolved?
5. What should I do next?
6. Where is the immutable audit proof?

The previous run debugger answered these questions, but in an implementation-centric order: workflow controls, metrics, graph, checks, and only then the human report. That made the most important result discoverable only after substantial scrolling.

## Findings and resolutions

| Severity | Finding | Resolution |
| --- | --- | --- |
| Critical | The claim graph appeared before the human decision, forcing non-specialists to interpret internal relationships before seeing the outcome. | The current decision, review progress, and human report now precede all technical evidence. |
| High | The old three-button workflow did not explain which stages were complete or why a button was disabled. | A four-stage progress model now shows analysis, evidence gaps, human decision, and certificate state with plain-language explanations. |
| High | Persisted approval could conflict visually with an unchecked local checkbox. | Approval state now initializes from the stored certificate; finalized runs no longer present editable controls. |
| High | Empty checks and unknowns appeared as blank or ambiguous panels. | Empty states explain that direct evidence satisfied policy or that no unresolved knowledge gaps remain. |
| High | The portfolio showed metrics but did not prioritize an action. | A recommended-next-step panel routes reviewers to evidence gaps, uncertified AI usage, or disconnected CI. |
| Medium | Portfolio tabs were not reflected in the URL, so views could not be shared and browser navigation was misleading. | `?view=` now identifies the active section and responds to browser back/forward navigation. |
| Medium | Technical labels such as `verification_pending` and evidence types leaked into human-facing text. | Statuses and event names are humanized at the view-model boundary. Machine values remain unchanged in the API and certificate. |
| Medium | The run page was a compressed component with data access, normalization, graph construction, actions, and rendering coupled together. | Domain/view-model logic moved to `run/model.ts`; reusable presentation moved to `run/components.tsx`; `run/page.tsx` now owns orchestration only. |
| Medium | The machine certificate competed visually with the human report. | Machine proof is now a clearly labeled, collapsed audit section below the review content. |
| Medium | Small-screen navigation overflowed and long run IDs could expand the page. | Responsive layouts, wrapping, constrained flex items, and a mobile-safe search control were added. |
| Low | Non-functional notification and avatar controls implied unavailable product behavior. | They were removed from the primary header. |

## Information architecture

The run review now follows a progressive-disclosure model:

```text
Run identity and proposed action
  → current outcome
  → evidence-to-decision progress
  → human decision report
  → evidence and critical claims
  → bounded checks
  → history and unresolved gaps
  → machine certificate
  → optional run comparison
```

The portfolio follows an action-first model:

```text
Recommended next step
  → portfolio health metrics
  → project and evidence summaries
  → recent audit activity
```

## Accessibility and comprehension

- Navigation exposes `aria-current` for the active portfolio view.
- Dialogs expose `role="dialog"`, `aria-modal`, a label, and a named close action.
- Run search has an explicit label and clear loading state.
- Claim selectors expose pressed state and do not require graph interaction.
- Errors use alert semantics.
- Keyboard focus is visible on navigation and primary actions.
- Color is reinforced with text labels; it is not the only status signal.
- Evidence strength is explicitly described as explainable support, not probability.

## Remaining product work

- Add authenticated user identity, account membership, and role-aware actions before public multi-tenant use.
- Add automated browser interaction tests for keyboard order, dialogs, tab history, downloads, and small-screen breakpoints.
- Add a project-level detail route so portfolio rows do not depend only on the run debugger.
- Add user-configurable density if the evidence ledger grows beyond the current demo scale.
