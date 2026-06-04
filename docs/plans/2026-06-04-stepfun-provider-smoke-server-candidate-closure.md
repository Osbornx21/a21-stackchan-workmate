# A21 StepFun Provider Smoke Server Candidate Closure

## Transition

`T-STEPFUN-PROVIDER-SMOKE-SERVER-CANDIDATE-CLOSURE-001`

## Current State

After `T-ROLEPLAY-VOICE-RUNTIME-PROBE-CLOSURE-001`, local no-hardware
server-side readiness had all software/runtime evidence except real provider
smoke. The local shell had no active `A21_` env, but
`.a21-run/provider.env` existed with A21-namespaced provider key variables.

## Target State

- Run an executed provider smoke for the launch-policy text provider.
- Feed the provider report into `server-side-readiness-bundle`.
- Reach `server_side_candidate_ready` while preserving the physical PRD gate.

## Execution

- Sourced `.a21-run/provider.env` without printing secret values.
- Used `A21_PROVIDER_PRIMARY=stepfun`,
  `A21_TEXT_STREAM_PROFILE=stepfun`, and
  `A21_STEPFUN_MODEL=step-1-8k`.
- Ran a dry provider smoke first; it reported `configured=true`.
- Ran executed streaming provider smoke:
  - report: `reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json`
  - status: `passed`
  - executed: `true`
  - repeat: `3`
  - HTTP status: `200`
  - first-content p50: `467.607ms`
  - first-content p95: `2235.073ms`
- Started local Gateway with the same launch-policy provider env and ran:
  `server-side-readiness-bundle --provider-smoke-report <report>
  --use-latest-reports --require-candidate`.
  - report: `reports/a21-server-side-readiness-bundle-20260604-153129.json`
  - status: `server_side_candidate_ready`
  - `candidate_ready=true`
  - `launch_ready=false`
  - physical gate remains required.

## Out Of Scope

- No provider keys were printed, stored in firmware, or committed.
- No ECS/root-secret change, firmware build, flash, serial, NVS, or physical
  hardware action.
- This does not prove physical StackChan playback, wake acceptance, roleplay
  physical expression, or full PRD launch.

## Next State

`S-INTERNAL-TEST4-SERVER-SIDE-CANDIDATE-READY-PHYSICAL-STACKCHAN-PENDING`

