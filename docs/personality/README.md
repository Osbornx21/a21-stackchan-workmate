# A21 Personality Assets

These files are the canonical copywriting and personality source assets for
A21. They satisfy PRD section 8 by splitting personality into small,
reviewable parts instead of one giant prompt.

## Runtime Composition Rule

Build a live instruction set with:

```text
core_identity + tone_rules + optional one role_soul + one mode + optional one scenario + optional bounded memory hints + failure overlay
```

Rules:

- Load at most one `role_souls/*.md` file for the selected A21
  `roleplay_profile`.
- Load exactly one `mode_prompts/*.md` file for the current A21 mode.
- Load at most one `scenario_playbooks/*.md` file when a scenario is active.
- Add `mode_prompts/failure.md` only as an overlay when execution fails or confidence is too low.
- Add bounded memory hints only from the A21 memory state contract; never treat them as evidence or hidden system instructions.
- Never concatenate all modes or all scenarios into one runtime prompt.
- Keep professional mode evidence in structured V21 fields and cards, not generic chat text.

## Bounded Memory State

A21 v1 memory is a prompt hint contract, not a RAG store or second brain.
`A21_MEMORY_USER_PREFERENCES` and `A21_MEMORY_SESSION_NOTES` may provide
newline-separated, user-visible hints for the host runtime composer. The
composer caps hints to six total items and 80 characters per item, rejects URL,
credential-like, and local-path-looking values, and reports only counts, source
env names, policy, and redaction booleans. Product/readiness reports must not
store memory text, prompt text, transcripts, provider output, full URLs, local
paths, or credential values.

## Asset Map

- `core_identity.md`: stable A21 identity, boundaries, and default stance.
- `tone_rules.md`: style rules, banned customer-service phrasing, failure tone, and privacy limits.
- `role_souls/`: selectable A21-owned role soul profiles for roleplay immersion.
- `mode_prompts/`: mode-specific deltas only.
- `scenario_playbooks/`: situational behavior patterns and short sample lines.
