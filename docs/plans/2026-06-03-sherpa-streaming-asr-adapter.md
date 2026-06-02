# 2026-06-03 - Sherpa Streaming ASR Adapter

Status: active.
Owner: A21 control tower plus scoped worker.
Transition: `T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001`.

## Background And Problem

`T-XIAOZHI-STREAMING-ASR-PROVIDER-001a` added a truthful static gate. It
correctly blocks the current selected chain because `sherpa_onnx` ASR still
accumulates PCM, writes a temporary WAV, and then invokes the local Sherpa smoke
runner. The model family can be `streaming_zipformer`, but the A21 adapter path
is still batch/file-boundary.

Gateway already has a stock `/v1/xiaozhi` streaming ASR seam:

`listen.start -> StreamingASRAdapter.StartStreamingASR -> AppendFrame on decoded
Opus PCM -> Commit -> asr.final`.

The missing provider step is a selectable real-progress local Sherpa streaming
adapter/profile that implements `providers.StreamingASRAdapter` without
relabeling the existing batch `sherpa_onnx` path.

## Current System State

- `internal/providers/voice_pipeline.go` already defines
  `StreamingASRAdapter` and `StreamingASRSession`.
- `internal/gateway/server.go` can start and feed a streaming ASR session when
  the selected ASR adapter implements that interface.
- `internal/providers/voice_pipeline_adapters.go` currently selects
  `NewLocalSherpaONNXASRAdapter` for `sherpa_onnx` and `local_sherpa_onnx`;
  that adapter writes a temporary WAV and only then emits partial/final events.
- `internal/app/xiaozhi_streaming_provider_readiness.go` currently labels
  `sherpa_onnx` as `asr_batch_wav_boundary_not_xiaozhi_streaming`.
- The static gate must remain red until ASR, LLM, and TTS are all truly
  streaming. This transition can improve the ASR stage but must not mark TTS or
  physical PRD acceptance green.

## Target State

- Add a distinct streaming Sherpa ASR profile, for example
  `sherpa_onnx_streaming` and `local_sherpa_onnx_streaming`.
- The new profile selects an adapter that implements both `ASRAdapter`
  fallback and `StreamingASRAdapter`.
- The streaming session path uses an injected/session-factory seam in tests so
  `AppendFrame -> partial -> Commit -> final` is proven without writing WAV.
- The existing `sherpa_onnx` batch profile remains unchanged and continues to
  be classified as WAV/batch.
- The readiness gate distinguishes:
  - batch `sherpa_onnx`: blocked with WAV boundary;
  - streaming `sherpa_onnx_streaming` without helper/model proof: blocked, no
    WAV boundary, finding says streaming helper/model proof is missing;
  - streaming `sherpa_onnx_streaming` with explicit helper/model configuration:
    ASR stage can be ready, while total gate still blocks if TTS remains WAV.
- Reports remain redaction-safe: no transcripts, raw audio, provider outputs,
  credentials, full URLs, or absolute local paths.

## Non-Goals

- Do not flash firmware.
- Do not start, stop, or restart Gateway.
- Do not run real Sherpa, provider, V21, or audio playback in this transition.
- Do not implement streaming TTS here.
- Do not change wake word, setup/no-welcome, TTS gain, codec volume, or firmware
  overlay.
- Do not claim physical Xiaozhi realtime parity or PRD acceptance.

## Impact Scope

- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Optional future helper file, only if implemented in this transition:

- `scripts/a21_sherpa_onnx_streaming_asr_session.py`

## Worker Task

Worker executes only `T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001`.

### Boundaries

- Allowed: edit Go provider adapter, app readiness gate, tests, and docs in the
  worker worktree.
- Allowed: run focused Go tests, `git diff --check`, and `make verify`.
- Forbidden: firmware changes, flash, service restart, real provider/ASR/V21
  execution, audio playback, TTS refactor, wake changes, Gateway behavior
  changes outside profile selection.

### Required Implementation Shape

1. Add `LocalSherpaONNXStreamingASRAdapterOptions` with at least:
   - `Name`;
   - batch fallback options or embedded batch adapter;
   - `StreamingSessionFactory` or equivalent injected factory.
2. Add `NewLocalSherpaONNXStreamingASRAdapter`.
3. The adapter must implement:
   - `Name() string`;
   - `Transcribe(...)` by delegating to the existing batch adapter as fallback;
   - `StartStreamingASR(...)` by opening the injected streaming session factory
     and returning a `providers.StreamingASRSession`.
4. If no streaming factory/helper is configured, `StartStreamingASR` must return
   a stable redacted error such as `sherpa-onnx streaming ASR helper is not
   configured`; it must not leak local paths or model text.
5. Extend `VoicePipelineAdapterOptions` with a streaming ASR factory hook for
   tests.
6. Extend `VoicePipelineAdaptersFromEnv` so only streaming profiles select the
   new adapter:
   - `sherpa_onnx_streaming`;
   - `local_sherpa_onnx_streaming`;
   - optionally `streaming_zipformer` as an alias if the worker documents it.
7. Keep `sherpa_onnx` and `local_sherpa_onnx` on the existing WAV/batch adapter.
8. Update the readiness gate:
   - batch Sherpa remains WAV-boundary blocked;
   - streaming Sherpa has `ImplementedInGateway=true`, `Streaming=true`,
     `UsesWAVBoundary=false`, `UsesFileBoundary=false`;
   - streaming Sherpa is ASR-ready only when explicit helper/model proof is
     configured by env or a future report seam;
   - missing helper/model returns a new stable finding, for example
     `asr_sherpa_streaming_helper_or_model_missing`.

## Acceptance Criteria

- Provider tests prove the new streaming adapter:
  - implements `StreamingASRAdapter`;
  - streams frames through an injected fake session;
  - emits partial before final;
  - does not create a temp WAV in the streaming test path.
- Env-selection tests prove:
  - `A21_ASR_LOCAL_PROFILE=sherpa_onnx` still selects the batch adapter;
  - `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming` selects an adapter that
    implements `StreamingASRAdapter`.
- Readiness tests prove:
  - `sherpa_onnx` still reports `asr_batch_wav_boundary_not_xiaozhi_streaming`;
  - `sherpa_onnx_streaming` without helper/model reports
    `asr_sherpa_streaming_helper_or_model_missing`, not WAV boundary;
  - `sherpa_onnx_streaming` with helper/model can make ASR stage ready, while
    total gate still blocks if TTS remains WAV.
- Commands pass:
  - `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1`
  - `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`
  - `git diff --check`
  - `make verify`

## Rollback

- Revert the provider adapter/profile/gate/docs changes.
- Existing `sherpa_onnx` batch fallback remains untouched, so rollback should
  not affect the last accepted audio or no-welcome firmware state.

## Risks

- A selectable streaming profile without actual helper/model runtime proof can
  be misread as product ready. The readiness gate must keep it blocked until
  explicit proof is present.
- Python-helper subprocess lifecycle, backpressure, cancellation, and stderr
  redaction are real runtime risks; if not implemented now, keep them recorded
  as the next transition rather than faking readiness.
- This improves ASR architecture only. TTS remains a WAV/file boundary and
  full Xiaozhi realtime parity remains blocked.

## Manual Confirmation Points

- None in this transition. Physical proof resumes only after ASR and TTS
  streaming gates are both green enough to run a real operator-triggered
  `/v1/xiaozhi` turn.

## Worker Return Format

Worker must return only:

- This stage did what;
- Files changed;
- Tests run and results;
- Whether it deviated from plan;
- Remaining blockers;
- Recommended next step.
