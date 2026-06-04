# Server-Side Professional Ritual Execution Gate

Transition: `T-SERVER-SIDE-PROFESSIONAL-RITUAL-EXECUTION-GATE-001`

Current state:

- Server-side readiness accepts V21 adapter smoke as professional evidence.
- `a21 xiaozhi-professional-bench` already produces external Gateway runtime evidence for the professional checking ritual and result ordering.
- Adapter smoke and professional ritual evidence are not separated in the launch gate.

Target state:

- Server-side candidate readiness requires a distinct `professional_ritual_ready` gate.
- The gate only accepts `a21.xiaozhi_professional_bench.v1` external Gateway evidence with checking feedback, professional result ordering, stale-result suppression, abort stop, V21 execution, and redaction checks.
- Adapter smoke remains useful as V21 adapter evidence, but it cannot by itself satisfy the professional ritual gate.

Action:

- Add `professional_ritual_ready` and source report fields to product readiness.
- Add a `professional_ritual` evidence block and `professional_ritual_execution` collection step to server-side readiness bundle.
- Prefer existing Xiaozhi professional bench evidence over newer adapter smoke when resolving latest professional reports.
- Update protocol/control handoff with the new gate and run focused plus full verification.

Acceptance:

- Product/server-side readiness blocks when adapter smoke exists but professional ritual evidence is missing.
- Candidate readiness passes when provider, V21/professional ritual, host voice, roleplay voice, wake-word, and voice-chain gates are all ready.
- Bundle collection never runs V21/professional external execution unless `--execute-v21-smoke` is supplied.
- `git diff --check` and `GOMAXPROCS=2 make verify` pass.

Rollback path:

- Revert this transition commit only. Do not revert internal-test3 audio/protocol changes or prior roleplay voice readiness commits.

Forbidden actions:

- No firmware build, flash, serial, NVS, provider key persistence in firmware, prune/gc, destructive git, or physical hardware action in this code-only transition.
