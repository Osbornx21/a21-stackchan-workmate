# Sherpa Streaming ASR Runtime Helper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `sherpa_onnx_streaming` from a selectable seam to a real long-lived streaming ASR helper/session boundary that can receive Xiaozhi PCM frames incrementally without writing WAV.

**Architecture:** A21 Gateway keeps the stock Xiaozhi WebSocket and decoded PCM frame flow. The provider layer starts one subprocess JSONL helper per ASR session, sends `start` then `append` frames as base64 PCM16LE, receives `partial` events before commit, sends `commit`, and receives `final`. Tests use a fake helper process only; real Sherpa/model execution remains a later guarded runtime proof.

**Tech Stack:** Go provider adapters, `os/exec`, JSONL over stdin/stdout, Python helper script, fake-helper Go tests, existing `providers.StreamingASRAdapter`.

---

## Background And Problem

The user reviewed Xiaozhi correctly: Xiaozhi behaves like a realtime media
system, not a request/response chatbot. A21 has already restored the device
transport shape: stock `/v1/xiaozhi` WebSocket, Opus 60 ms frames, server TTS
lifecycle, and paced downlink. The current ASR provider path is still short of
Xiaozhi parity:

- `sherpa_onnx` remains a batch/WAV adapter.
- `sherpa_onnx_streaming` can be selected and is classified separately by the
  readiness gate, but it only starts through an injected test factory.
- No default runtime helper exists to accept `AppendFrame` from the live
  Gateway streaming ASR session.

This transition closes the next real architecture gap without pretending full
PRD acceptance is done.

## Current System State

- `internal/providers/voice_pipeline.go` defines `StreamingASRAdapter`,
  `StreamingASRSession`, `StreamingASRStartRequest`, and `VoicePipelinePCMFrame`.
- `internal/gateway/server.go` can start and feed a streaming ASR session when
  the selected ASR adapter implements `StreamingASRAdapter`.
- `internal/providers/voice_pipeline_adapters.go` defines
  `LocalSherpaONNXStreamingASRAdapter`, but `StartStreamingASR` returns
  `sherpa-onnx streaming ASR helper is not configured` unless an injected
  factory is present.
- `scripts/a21_sherpa_onnx_asr_smoke.py` is batch/WAV and must not be reused
  as the streaming helper.
- `internal/app/xiaozhi_streaming_provider_readiness.go` treats
  `sherpa_onnx_streaming` as implemented only when helper/model env is present,
  but this is static/no-execute evidence, not runtime proof.

## Target State

- Add a repo-owned helper script:
  `scripts/a21_sherpa_onnx_streaming_asr_session.py`.
- Add a Go subprocess-backed `StreamingASRSessionFactory` used by
  `sherpa_onnx_streaming` when both helper path and model dir are configured.
- Preserve the existing injected factory for tests and future non-subprocess
  implementations.
- Prove with fake helper tests that:
  - frames are sent as JSONL `append` commands before `commit`;
  - partial events can arrive before final;
  - the streaming path writes no WAV;
  - errors are stable and redacted.
- Keep `sherpa_onnx` batch/WAV unchanged and still blocked by the strict
  Xiaozhi realtime readiness gate.
- Keep readiness honest: helper configured means ASR shape can be statically
  ready, but no real model/provider/physical acceptance is claimed by this
  transition.

## Non-Goals

- Do not flash firmware.
- Do not start, stop, or restart Gateway.
- Do not run real Sherpa, Doubao, Iflytek, 5080, V21, or any provider network.
- Do not play audio.
- Do not change wake word, no-welcome firmware, codec volume, TTS gain,
  downlink pacer, or state machine behavior outside ASR session wiring.
- Do not remove the batch Sherpa adapter.
- Do not mark `prd_accepted=true`.

## Impact Scope

- Create: `scripts/a21_sherpa_onnx_streaming_asr_session.py`
- Modify: `internal/providers/voice_pipeline_adapters.go`
- Modify: `internal/providers/voice_pipeline_real_adapters_test.go`
- Modify if needed: `internal/app/xiaozhi_streaming_provider_readiness.go`
- Modify if needed: `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- Modify: `docs/project_state_machine.md`
- Modify: `docs/agent_handoff_log.md`

## Helper Protocol

All messages are newline-delimited JSON on stdin/stdout. The helper must never
print transcripts, local paths, model paths, provider output, proxy values, or
secrets to stderr. Stable error codes are allowed.

Go to helper commands:

```json
{"type":"start","sample_rate_hz":16000,"channels":1,"mode":"workmate","trace_id":"redacted","session_id":"redacted","model_dir":"...","family":"streaming_zipformer"}
{"type":"append","seq":1,"sample_rate_hz":16000,"channels":1,"duration_ms":60,"pcm16le_b64":"..."}
{"type":"commit"}
{"type":"cancel","reason":"barge_in"}
```

Helper to Go events:

```json
{"type":"ready"}
{"type":"partial","text":"..."}
{"type":"final","text":"..."}
{"type":"error","code":"sherpa_streaming_helper_failed"}
```

The Go report path must redact text and paths; ASR text may flow internally to
the voice pipeline, but reports/findings must not store it.

## Phased Execution

### Phase 1: Subprocess Factory And Fake Helper Test

Acceptance:

- A provider test creates a fake helper script in `t.TempDir()`.
- Starting `sherpa_onnx_streaming` with helper/model env starts the subprocess.
- `AppendFrame` writes an `append` command before `Commit`.
- The fake helper emits `partial` after append and `final` after commit.
- The test proves no temp WAV path is created by making the old batch runner
  fail if called.

Rollback:

- Revert the subprocess factory and fake helper test. The injected factory seam
  remains usable.

### Phase 2: Python Helper Script

Acceptance:

- `scripts/a21_sherpa_onnx_streaming_asr_session.py --help` or import-safe
  parsing does not require sherpa-onnx.
- In fake mode, the helper can read JSONL commands and emit deterministic
  `ready`, `partial`, and `final` events for tests/manual dry checks.
- Real mode validates model dir/family and returns stable error codes if
  sherpa-onnx or model files are missing.

Rollback:

- Remove the script and leave Go configured-helper selection blocked.

### Phase 3: Env Wiring And Redaction

Acceptance:

- `VoicePipelineAdaptersFromEnv` wires the subprocess factory only for
  `sherpa_onnx_streaming`, `local_sherpa_onnx_streaming`, and
  `streaming_zipformer`.
- Required env:
  - `A21_SHERPA_ONNX_STREAMING_HELPER`
  - `A21_SHERPA_ONNX_ASR_MODEL_DIR`
- Optional env:
  - `A21_SHERPA_ONNX_STREAMING_PYTHON`
  - `A21_SHERPA_ONNX_ASR_FAMILY`
- Batch profiles `sherpa_onnx` and `local_sherpa_onnx` still select the batch
  WAV adapter.
- Errors from helper start/read/write are stable:
  - `sherpa-onnx streaming ASR helper failed`
  - `sherpa-onnx streaming ASR helper exited`
  - `sherpa-onnx streaming ASR helper protocol error`

Rollback:

- Revert env wiring and keep readiness gate blocked as missing helper/model.

### Phase 4: State And Handoff

Acceptance:

- `docs/project_state_machine.md` records this transition as runtime-helper
  candidate evidence, not product acceptance.
- `docs/agent_handoff_log.md` records exact files, commands, reports, tests,
  and remaining blockers.

Rollback:

- Revert docs update if implementation is reverted.

## Required Tests And Commands

Focused tests:

```bash
go test ./internal/providers -run 'TestLocalSherpaONNX.*Streaming|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1
go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1
```

Final checks:

```bash
git diff --check
make verify
```

Forbidden in this transition:

```bash
make a21-stackchan-official-xiaozhi-compatible-flash-execute
go run ./cmd/a21 gateway
go run ./cmd/a21 provider-smoke-execute
go run ./cmd/a21 v21-adapter-smoke-execute
```

## Risks

- A subprocess helper can leak or hang if stdin/stdout goroutines are not
  canceled. The session must close pipes and kill the process on `Cancel`.
- Scanner defaults can truncate large lines. Use `json.Decoder` or a scanner
  buffer large enough for 60 ms PCM base64.
- Static helper env can be misread as real ASR proof. Reports and state machine
  must keep runtime/model and physical acceptance separate.
- Real sherpa-onnx streaming recognizer APIs may vary by model family; fake
  tests must not pretend real model execution happened.

## Manual Confirmation Points

- None. This transition is host-only and no-audio.
- Physical testing resumes only after ASR helper runtime, streaming TTS runtime,
  and no-welcome/wake state are acceptable enough to run an operator-triggered
  stock `/v1/xiaozhi` turn.

## Worker Execution Instructions

Worker should implement only this plan in a scoped worktree.

Boundaries:

- Allowed: edit listed files, create helper script, run focused tests,
  `git diff --check`, and `make verify`.
- Forbidden: firmware, flash, NVS, service restarts, provider/V21 execution,
  hardware, audio playback, unrelated refactors.

Worker return format:

- This stage did what.
- Files changed.
- Tests run and results.
- Whether it deviated from plan.
- Remaining blockers.
- Recommended next step.
