# 2026-06-03 - Streaming TTS Runtime Proof

Status: active.
Owner: A21 control tower plus scoped worker.
Transition: `T-STREAMING-TTS-RUNTIME-PROOF-001`.

## Background And Problem Definition

A21 now has a selectable `doubao_tts_realtime` adapter seam and static
readiness can distinguish it from WAV/file TTS paths. That is necessary but not
enough for Xiaozhi realtime parity. The user's target is not a channel-shaped
API; it is a voice media system where provider audio begins flowing before a
complete TTS file or full answer exists.

The remaining gap is runtime proof:

- local/Iflytek/voice-clone TTS still synthesize a complete WAV/file before A21
  reads chunks;
- `doubao_tts_realtime` has fake-connection tests and static readiness, but no
  redacted runtime smoke proving session start, text append, first audio delta,
  60 ms chunk framing, and close/failure behavior;
- no report currently proves or truthfully blocks real streaming TTS execution.

## Current System State

- `providers.StreamingTTSAdapter` exists.
- `A21_TTS_FAST_PROFILE=doubao_tts_realtime` selects a streaming TTS adapter.
- Fake realtime-conn provider tests prove provider `response.audio.delta`
  becomes downlink-ready `pcm_s16le` 60 ms chunks without WAV.
- `xiaozhi-streaming-provider-readiness` can mark the static TTS shape ready
  when required Doubao env exists, but it does not execute the provider.
- `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` is completed as a truthful blocker
  with `model_dir_missing`; ASR real-model proof remains separate.

## Target State

- A new redacted runtime smoke command exists for streaming TTS.
- By default, the command performs no provider network call and records a stable
  blocker if `--execute` is absent or required env is missing.
- With `--execute` and complete env, the command starts the realtime TTS
  session, sends a short synthetic non-user test text, records first provider
  audio delta timing, drains at least one 60 ms PCM chunk, closes the session,
  and writes a redacted report.
- The report can be used by future Xiaozhi parity gates as TTS runtime evidence,
  while still not claiming physical PRD acceptance.

## Non-Goals

- Do not change firmware, wake, NVS, gain, speaker volume, or Gateway service
  lifecycle.
- Do not call ASR, LLM, V21, `/v1/xiaozhi/say`, or physical device audio.
- Do not play audio or write WAV.
- Do not store user text, provider output text, raw/base64 audio, credentials,
  full URLs, proxy values, or absolute local paths.
- Do not mark full Xiaozhi realtime parity or PRD accepted.
- Do not remove local/Iflytek/voice-clone TTS; keep their boundary classified
  honestly.

## Impact Scope

- `internal/app/streaming_tts_runtime_smoke.go` or equivalent new app file
- `internal/app/app_plan_execute.go`
- `internal/app/*streaming_tts*_test.go`
- `internal/providers/doubao_realtime_tts*.go` only if a minimal helper is
  missing for safe runtime smoke
- `Makefile`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Phased Execution Steps

### Phase 1 - Report Contract And No-Execute Blocker

1. Add `a21 streaming-tts-runtime-smoke`.
2. Add `make streaming-tts-runtime-smoke`.
3. Default behavior without `--execute` returns nonzero and writes a report
   with `status=blocked`, finding `execute_flag_required`.
4. Missing Doubao env returns `status=blocked` with stable missing-env names.
5. Reports use only env names, provider name, booleans, counts, durations,
   basenames if needed, and redaction policies.

Acceptance:

- Focused app tests prove no-execute and missing-env blockers.
- Output/report do not contain credential, raw/base64 audio, full URL, path, or
  user text.

### Phase 2 - Fake Realtime Runtime Proof

1. Use an injected fake realtime dialer in tests.
2. Prove the command observes:
   - session update sent;
   - text append sent;
   - text done sent;
   - first `response.audio.delta` received;
   - at least one 60 ms PCM chunk emitted before provider EOF.
3. Prove errors are stable and redacted.

Acceptance:

- Tests fail if the adapter waits for EOF before first audio.
- Tests prove no WAV/file boundary.

### Phase 3 - Guarded Real Provider Execution

1. Allow `--execute` only when env is complete:
   - `A21_TTS_FAST_PROFILE=doubao_tts_realtime`
   - `A21_DOUBAO_API_KEY`
   - `A21_DOUBAO_TTS_MODEL`
   - `A21_DOUBAO_TTS_VOICE`
   - optional `A21_DOUBAO_TTS_SAMPLE_RATE_HZ`
2. Use a short fixed test text owned by A21, not user transcript.
3. Respect A21 network policy and direct-connect/proxy rules without changing
   Codex proxy settings.
4. Runtime success requires first audio delta and at least one valid 60 ms PCM
   chunk; provider failure is recorded as stable `provider_runtime_failed`.

Acceptance:

- If provider execution is not authorized or env is missing, the transition can
  complete as a truthful blocker.
- If executed, report records first-audio timing and chunk count only.

## Rollback

- Revert the smoke command, tests, Make target, and docs.
- Existing static adapter seam, local TTS, Iflytek TTS, voice clone, accepted
  3x physical audio path, and no-welcome firmware remain untouched.

## Risks

- A report-only runtime smoke can be mistaken for physical `/v1/xiaozhi`
  acceptance. State machine and readiness must keep this separate.
- Provider API shape or credentials may be stale. Treat this as a blocker or
  runtime failure, not a reason to fake readiness.
- Audio delta sample rate or chunk size may differ. The command must validate
  and count only valid downlink-ready chunks.
- Executing provider calls may cost money. Keep `--execute` explicit and
  default to no-execute blocker.

## Required Manual Confirmation

- Whether the operator authorizes a real provider call with current env.
- Physical listening acceptance remains a later transition using stock
  `/v1/xiaozhi`, not this smoke.

## Worker Execution Task

Implement `T-STREAMING-TTS-RUNTIME-PROOF-001` in an isolated worker branch.

Worker boundaries:

- Allowed: app/provider tests, app command, Make target, state/handoff docs.
- Forbidden: firmware, flash, NVS, Gateway start/stop, physical device,
  audio playback, ASR/LLM/V21 execution, `/v1/xiaozhi/say`, provider execution
  unless the command is explicitly run with `--execute`.
- Must not leak secrets, text, raw/base64 audio, full URLs, proxy values, or
  absolute paths.

Worker return format:

- Branch/HEAD/dirty
- What this stage did
- Files changed
- Tests run/results
- Whether it deviated from plan
- Runtime result: passed/blocked/failed and stable finding
- Remaining blockers
- Recommended next transition
