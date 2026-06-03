# 2026-06-03 - Xiaozhi Realtime Parity Real Profile Evidence

Status: completed host-local evidence-hardening transition.
Owner: A21 control tower.
Transition: `T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001`.

## Problem

A21's `xiaozhi-realtime-parity` gate already checks stock `/v1/xiaozhi` Opus
transport and event ordering. That is necessary but not enough. A trace with
mock text/TTS, local fallback, or a WAV/file-boundary TTS path can still look
like a Xiaozhi-style realtime sequence if it emits the same generic trace
markers.

The user requirement is stricter: A21 must behave like Xiaozhi's realtime voice
system, not a buffered question/answer system hidden behind a compatible
socket. Realtime candidate evidence therefore needs explicit proof that the
turn used real streaming ASR, real streaming LLM, and real streaming TTS path
classes.

## Current State

- `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001` is landed.
- `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` is landed as a host-local no-audio
  real-model ASR smoke.
- `xiaozhi-realtime-parity` checks ASR partial ordering, provider first
  content, TTS first audio, Opus downlink, and fake path markers.
- Trace events do not yet expose whether the selected ASR/LLM/TTS profiles were
  real streaming implementations or mocks/file-boundary adapters.

## Target State

- Gateway records profile-class trace markers for Xiaozhi voice-pipeline turns:
  real streaming ASR, real streaming LLM, real streaming TTS, and blockers for
  mock or file-boundary stages.
- `xiaozhi-realtime-parity` requires the real streaming stage markers before it
  can classify a trace as `xiaozhi_realtime_candidate`.
- Mock/text fixture/file-boundary traces remain below realtime parity even when
  their generic timing markers are ordered correctly.
- Reports remain redacted and do not store transcripts, provider outputs, raw
  audio, URLs, credentials, proxy values, or absolute paths.

## Scope

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Non-Goals

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say` acceptance.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.
- No migration to MQTT+UDP in this transition.

## Acceptance

- Red tests first show the parity gate currently accepts realtime-looking mock
  traces too strongly.
- Gateway tests prove mock voice-pipeline turns emit blocker markers rather
  than real-stage markers.
- App tests prove `xiaozhi-realtime-parity` keeps mock/file-boundary traces out
  of `xiaozhi_realtime_candidate`.
- Existing realtime-ordering tests continue to pass when the trace contains
  explicit real streaming profile markers.
- `git diff --check`, focused Go tests, and `make verify` pass.

## Failure States

- `F-REALTIME-PARITY-MOCK-GREEN` if a mock text/TTS path can still reach
  `xiaozhi_realtime_candidate`.
- `F-TRACE-SECRET-LEAK` if profile evidence records raw profile values,
  credentials, URLs, paths, transcripts, or audio payloads.
- `F-STOCK-TRANSPORT-REGRESSION` if stock `/v1/xiaozhi` parsing, Opus ingress,
  or downlink tests regress.

## Rollback

Revert the trace marker additions, parity-gate checks, tests, and state/log
entries. Existing ASR partial bridge, Sherpa smoke, streaming TTS adapter seam,
and stock `/v1/xiaozhi` transport remain intact.

## Next State

If accepted, the next transition is a runtime evidence step: either authorized
real streaming TTS execution with complete env, or a physical stock
`/v1/xiaozhi` parity report that honestly remains blocked until real profile
markers appear in the live trace.
