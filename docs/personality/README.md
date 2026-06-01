# A21 Personality Assets

These files are the canonical copywriting and personality source assets for
A21. They satisfy PRD section 8 by splitting personality into small,
reviewable parts instead of one giant prompt.

## Runtime Composition Rule

Build a live instruction set with:

```text
core_identity + tone_rules + one mode + optional one scenario + failure overlay
```

Rules:

- Load exactly one `mode_prompts/*.md` file for the current A21 mode.
- Load at most one `scenario_playbooks/*.md` file when a scenario is active.
- Add `mode_prompts/failure.md` only as an overlay when execution fails or confidence is too low.
- Never concatenate all modes or all scenarios into one runtime prompt.
- Keep professional mode evidence in structured V21 fields and cards, not generic chat text.

## Asset Map

- `core_identity.md`: stable A21 identity, boundaries, and default stance.
- `tone_rules.md`: style rules, banned customer-service phrasing, failure tone, and privacy limits.
- `mode_prompts/`: mode-specific deltas only.
- `scenario_playbooks/`: situational behavior patterns and short sample lines.
