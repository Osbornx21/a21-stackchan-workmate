# 2026-06-03 - Sherpa Real-Model No-Audio Smoke

Status: ready for scoped worker.
Owner: A21 control tower plus one implementation worker.
Transition: `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001`.

## Background And Problem Definition

`sherpa_onnx_streaming` now has a mainline JSONL subprocess helper and Go
`StreamingASRSession` adapter. That proves the A21 interface can stream
`AppendFrame -> partial -> Commit -> final` with fake helper tests, but it does
not prove a real Sherpa streaming recognizer can be started from the local
model cache.

The next honest step is a no-audio/no-hardware smoke that validates the helper
against real model files if they are present, or records a stable missing-model
blocker if they are not.

## Current System State

- Mainline has `scripts/a21_sherpa_onnx_streaming_asr_session.py`.
- Mainline has `NewLocalSherpaONNXStreamingASRAdapter` with optional
  `A21_SHERPA_ONNX_STREAMING_HELPER`,
  `A21_SHERPA_ONNX_STREAMING_PYTHON`,
  `A21_SHERPA_ONNX_ASR_MODEL_DIR`, and
  `A21_SHERPA_ONNX_ASR_FAMILY`.
- Fake helper tests and `make verify` pass.
- No real model execution, provider execution, hardware execution, service
  restart, or audio playback is accepted yet.

## Target State

`S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`

One of two truthful outcomes is acceptable:

1. Real model smoke passes:
   - helper starts real Sherpa streaming recognizer;
   - accepts a tiny generated PCM16LE frame or silence frame;
   - commits cleanly and emits either empty final or stable recognized text;
   - report stores no transcript/audio/raw payload/full path/secret.
2. Real model smoke is blocked:
   - report records stable blocker such as `model_dir_missing`,
     `model_files_missing`, or `sherpa_onnx_unavailable`;
   - readiness remains red and no fake green is claimed.

## Non-Goals

- Do not play audio.
- Do not capture microphone.
- Do not call any cloud provider.
- Do not call V21.
- Do not start or stop Gateway.
- Do not flash firmware or write NVS.
- Do not use a WAV file in the streaming path.
- Do not store transcripts, raw audio, provider outputs, credentials, full
  URLs, or absolute local paths in reports.

## Impact Scope

Preferred smallest implementation:

- `internal/app/app.go`
  - Add a CLI command such as `a21 local-asr-streaming-smoke` only if no
    suitable command exists.
- `internal/app/local_asr_streaming_smoke.go`
  - Create a redacted no-audio smoke report for `sherpa_onnx_streaming`.
- `internal/app/local_asr_streaming_smoke_test.go`
  - Unit tests for pass, missing model, and redaction.
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

If a suitable report command already exists, the worker should reuse it instead
of creating a new CLI.

## Execution Steps

1. Confirm branch/HEAD/dirty and read this plan plus `AGENTS.md`,
   `docs/project_state_machine.md`, and the latest `docs/agent_handoff_log.md`.
2. Search for existing ASR smoke commands before adding anything:
   - `rg -n "local-asr|asr-smoke|sherpa|streaming" internal/app cmd Makefile`
3. Inspect local model-cache conventions, but do not download models.
4. Add or reuse a redacted smoke report that:
   - reads env names but not secret values;
   - validates model dir basename/file presence without storing full path;
   - starts the helper only when model files appear valid;
   - sends generated PCM16LE silence/tiny frame through JSONL;
   - commits the session;
   - classifies pass/block/fail without storing transcript or audio payload.
5. Add focused tests with fake helper/process seams where needed.
6. Run:
   - focused app tests for the smoke command;
   - `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1`;
   - `git diff --check`;
   - `make verify` if the focused tests pass.
7. Update state machine and handoff log.
8. Commit only if tests pass.

## Acceptance Criteria

- The worker returns branch/HEAD/dirty and a commit hash if committed.
- The smoke report distinguishes:
  - helper fake/unit success;
  - real model success;
  - missing model/config block;
  - helper protocol/runtime failure.
- No WAV file is written or read in the streaming smoke path.
- No real audio playback/capture/provider/V21/hardware action occurs.
- Static provider readiness and PRD acceptance remain red unless a later
  physical/runtime transition proves the full chain.

## Rollback

- Revert the smoke CLI/report/tests/docs. The main Sherpa streaming helper seam
  remains available.

## Risks

- Local model layout may not match `encoder.int8.onnx`, `decoder.onnx`,
  `joiner.int8.onnx`, and `tokens.txt`; this must become a truthful
  `model_files_missing` blocker.
- Python/sherpa-onnx availability may differ by shell; report should name the
  missing package class, not a full local path.
- Any transcript emitted by a real recognizer is sensitive enough to avoid in
  reports, even if generated silence is used.

## Worker Return Format

- Branch/HEAD/dirty.
- This stage did what.
- Files changed.
- Tests run and results.
- Whether it deviated from this plan.
- Remaining blockers.
- Recommended next transition.
