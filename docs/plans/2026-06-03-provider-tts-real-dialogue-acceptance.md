# A21 Provider/TTS Real Dialogue Acceptance Plan

Status: active plan, code-ready with Mac-direct TTS blocked  
Date: 2026-06-03  
Owner: A21 control tower  
Transition: `T-PROVIDER-002-REAL-TTS-LLM-SELECTION`

## Background And Problem Definition

The physical StackChan audio hotfix is accepted for loudness and clarity with
bounded Gateway gain at 3x, runtime speaker volume `100`, and stock Xiaozhi
protocol parity. The remaining operator complaint is no longer only loudness:
the current local TTS voice is not acceptable for the contest conversation
experience.

5080lab returned `outbox/A21-VOICE-FULL-REPORT.md` through the established
file-based lane. The report recommends:

- real-time LLM: StepFun `step-1-8k`, OpenAI-compatible streaming endpoint;
- real-time TTS: Xfyun/Iflytek TTS, 16 kHz PCM output over WebSocket;
- fallback/backup: DashScope for Chinese text quality and an A21-owned local
  fallback path after a separate bakeoff;
- not the immediate contest baseline: Volcengine async TTS and 5080 IndexTTS2.
  This does not remove A21 voice cloning. `voice_clone_cli` remains the
  product-facing clone seam for IndexTTS2, CosyVoice, F5-TTS, or GPT-SoVITS.

The report contains plaintext provider credentials. Those values must not be
copied into source, docs, reports, logs, handoff entries, or shell history
fragments recorded in the repo. A21 must consume only environment variable
names and redacted evidence.

## Current System State

- Branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Last integrated state: `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`.
- Gateway-side foreground TTS playback to physical StackChan is accepted with
  recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a`.
- `local-voice-loopback` and `stackchan-fast-companion-turn` already accept an
  explicit text-stream provider through `--text-provider` and
  `--execute-text-provider`.
- Built-in StepFun text-stream catalog support exists, but it is a
  compatibility candidate rather than product-readiness route-eligible by
  default.
- Host-side TTS engines cover `sherpa_onnx`, `macos_say`,
  `voice_clone_cli`, and now `iflytek_tts`. The old `sherpa_onnx` voice is
  retained only as an emergency/diagnostic path for this contest window.
- `voice_clone_cli` is a separate voice-clone capability, not the old local
  TTS. It must remain hot-pluggable while the immediate fast-dialogue path uses
  Iflytek.
- Global Mac/Codex proxy settings are present and must not be changed. A21
  runtime commands may set process-scoped `NO_PROXY` / `no_proxy` only.

## Target State

`S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

The contest dialogue path uses a selected real mainland-friendly text provider
plus a clearer real-time TTS source, without firmware changes:

- `A21_TEXT_STREAM_PROFILE=stepfun` with `A21_STEPFUN_MODEL=step-1-8k` and
  `A21_LAB_STEPFUN_API_KEY` supplied from a local secret environment only.
- `A21_LOCAL_TTS_ENGINE=iflytek_tts` with
  `A21_IFLYTEK_TTS_APP_ID`, `A21_IFLYTEK_TTS_API_KEY`, and
  `A21_IFLYTEK_TTS_API_SECRET` supplied from local secret environment only.
- `A21_TTS_FAST_PROFILE=iflytek_tts` when the Gateway voice pipeline, not only
  CLI smoke commands, should select Iflytek.
- `A21_TTS_FAST_PROFILE=voice_clone_cli` or
  `A21_LOCAL_TTS_ENGINE=voice_clone_cli` remain valid when the user selects a
  cloned/persona voice and provides `A21_VOICE_CLONE_*` configuration.
- StackChan continues to receive stock Xiaozhi `tts` lifecycle JSON plus Opus
  audio frames; firmware remains unaware of provider identity.
- Evidence stays redacted: no raw text, transcript, provider output, audio
  base64, provider credentials, full URL, proxy URL, or local path.

## Non-Goals

- Do not modify global macOS proxy, Codex proxy, shell profile proxy, or network
  service configuration.
- Do not flash firmware, write NVS, or touch wake-word firmware in this
  transition.
- Do not promote StepFun to product route-eligible readiness in the provider
  catalog without a separate promotion decision and evidence.
- Do not hard-code provider credentials from the 5080 report.
- Do not store the raw 5080 voice report in the A21 repository.
- Do not reopen the accepted 3x Gateway gain unless physical playback clips or
  becomes harsh with the new TTS source.
- Do not remove, downgrade, or reinterpret `voice_clone_cli` as the rejected
  old local TTS. Voice cloning remains a retained A21 capability.
- Do not make CosyVoice the active contest fallback without a separate local
  bakeoff; it is the preferred future ordinary local fallback / clone-model
  candidate, not an untested replacement in this transition.

## Impact Scope

- `internal/audio`: add an env-driven host-side Iflytek TTS synthesizer that
  writes 16 kHz mono PCM WAV and returns the existing redacted
  `a21.audio.local_tts.v1` report shape.
- `internal/app`: expose `iflytek_tts` through existing local TTS CLI/runtime
  selection for `local-tts-smoke`, `local-voice-loopback`,
  `stackchan-local-tts-playback`, and `stackchan-fast-companion-turn`.
- `internal/providers`: allow explicitly selected text-stream candidates such
  as StepFun to run in the host-local voice pipeline without promoting them to
  product route eligibility, and expose `iflytek_tts` as a host-local TTS
  adapter while keeping `voice_clone_cli`.
- `docs/engineering/PROTOCOL.md` / `docs/engineering/NETWORK.md`: document
  provider-neutral host TTS selection and process-scoped direct/LAN proxy
  handling if code changes introduce new env vars.
- `docs/project_state_machine.md` and `docs/agent_handoff_log.md`: record
  state transition and verification.

## Phased Execution

### Phase 1: Confirm Report And Runtime Boundary

Actions:

- Read 5080 voice report through `5080lab` outbox or the local 5080 lane.
- Confirm selected real-time combo: StepFun text stream plus Iflytek TTS.
- Confirm that StepFun can be invoked through the existing OpenAI-compatible
  text-stream path with explicit env.
- Confirm that Iflytek TTS needs only a host-side TTS adapter.

Acceptance:

- Selected combo and non-goals are written in this plan.
- Plaintext report credentials are not copied into repo documents.
- `docs/project_state_machine.md` points at this transition.

### Phase 2: Implement Iflytek Host TTS Adapter

Worker task:

- Add `iflytek_tts` as a local TTS engine.
- Implement HMAC-authenticated WebSocket request using env-provided app id,
  API key, and API secret.
- Request 16 kHz 16-bit mono PCM output and write an A21 WAV via existing audio
  helpers.
- Include only endpoint host, provider/engine labels, voice label, timing,
  file basename, and aggregate PCM quality in reports.
- Add tests with a local fake WebSocket server or injected dialer/client seam;
  tests must prove redaction and missing-env behavior.

Acceptance:

- `go test ./internal/audio ./internal/app -run 'Iflytek|LocalTTS|VoiceLoopback|StackChan.*TTS|FastCompanion' -count=1` passes.
- Reports never contain credentials, Authorization/auth URL query, raw text,
  provider output, base64 audio, full URL, proxy value, or local path.
- Existing `sherpa_onnx`, `macos_say`, and `voice_clone_cli` behavior is
  unchanged; `voice_clone_cli` remains the voice-clone path.

### Phase 3: Host-Only Smoke With Selected Combo

Actions:

- Run `local-tts-smoke --engine iflytek_tts` with env from a local secret file
  or unrecorded shell export.
- Run `local-voice-loopback --engine iflytek_tts --text-provider stepfun
  --execute-text-provider` with process-scoped direct/LAN `NO_PROXY`.
- Keep StepFun as a compatibility-executed provider unless a separate
  promotion profile is explicitly loaded.

Acceptance:

- Iflytek TTS smoke passes with redacted output.
- StepFun text stream produces a redacted timing receipt.
- No global proxy mutation occurred.

### Phase 4: Physical StackChan Foreground Playback

Actions:

- Ensure Gateway is healthy on the selected LAN port.
- Ensure physical StackChan is currently online, not only stale.
- Send a long foreground TTS through `stackchan-local-tts-playback` or
  `/v1/xiaozhi/say` using the new Iflytek-generated WAV/TTS path.
- Operator records and accepts/rejects clarity.

Acceptance:

- Physical device receives stock TTS lifecycle and Opus chunks.
- Runtime speaker volume remains `100`.
- Operator accepts the voice quality for contest baseline conversation, or the
  transition records the exact rejection reason and rolls back to accepted 3x
  local TTS.

### Phase 5: Real Dialogue Candidate

Actions:

- Use `stackchan-fast-companion-turn` with `--text-provider stepfun
  --execute-text-provider --engine iflytek_tts` once the TTS smoke is accepted.
- Keep `listen_source=host_fixture` unless physical mic capture is explicitly
  opened.

Acceptance:

- Redacted report shows real text stream and real host-side TTS in one chain.
- Physical StackChan plays the answer clearly.
- Full PRD remains blocked until wake-word and normal dialogue half-duplex are
  separately accepted.

## Rollback Plan

- Set `A21_LOCAL_TTS_ENGINE=sherpa_onnx` only as an emergency/diagnostic
  rollback to the accepted 3x local path; it is not the desired contest voice.
- Set `A21_LOCAL_TTS_ENGINE=voice_clone_cli` or
  `A21_TTS_FAST_PROFILE=voice_clone_cli` to test the retained voice-clone path
  when a wrapper/reference voice is configured.
- Keep `A21_TEXT_STREAM_PROFILE` unset or set to the previous selected provider.
- Stop using any temporary StepFun route-eligible profile file if one was
  created for local testing.
- Reuse the accepted runtime volume and Gateway downlink gain; do not flash
  firmware as part of rollback.

## Risks

- The 5080 report included plaintext credentials; local secrets should be
  treated carefully, and rotation may be needed outside this repo.
- Iflytek TTS can fail if local clock skew breaks HMAC signing.
- Process-scoped `NO_PROXY` protects LAN/Gateway/StackChan traffic but does not
  itself guarantee a provider egress path through mainland routing.
- The Iflytek WebSocket client is direct by default and deliberately ignores
  ambient `HTTP_PROXY` / `HTTPS_PROXY`; explicit provider-proxy support for
  WebSocket TTS requires a later adapter transition.
- StepFun is not route-eligible in the built-in catalog; using it for contest
  listening is acceptable as an explicit candidate, but product readiness
  promotion needs a separate transition.
- New cloud TTS may have different loudness, so the accepted 3x Gateway gain
  might need a listening-only check for clipping or harshness.

## Human Confirmation Points

- Confirm whether exposed 5080 credentials should be rotated outside the repo.
- Confirm physical listening acceptance after Iflytek TTS playback.
- Confirm whether StepFun should remain a contest-only selected candidate or be
  promoted through a separate route-eligible profile transition after evidence.

## 2026-06-03 Execution Update

- Code path is implemented and tested: `iflytek_tts` is available through
  `local-tts-smoke`, `local-voice-loopback`, `stackchan-local-tts-playback`,
  `stackchan-fast-companion-turn`, and the Gateway voice-pipeline TTS adapter.
- Explicit StepFun/compatibility text-stream profiles can execute without being
  promoted to product `route_eligible=true`.
- `voice_clone_cli` remains fully retained as the voice-clone seam; worker
  review confirmed IndexTTS2/5080 bridge, CosyVoice/F5/GPT-SoVITS docs,
  readiness checks, and smoke/loopback/playback tests still exist.
- Live StepFun text-stream smoke passed on Mac direct:
  `reports/provider-tts-candidate/a21-provider-smoke-20260603-010652-957877000.json`
  has `status=passed`, `network_mode=direct`, `route_eligible=false`, and
  `repeat=2`.
- Live Iflytek TTS smoke on Mac direct extracted credentials from the 5080
  report only in process memory, printed no secret values, and failed at
  WebSocket dial. Redacted report:
  `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json`
  has `status=failed`, `network_mode=direct`, and finding
  `iflytek_tts_websocket_dial_failed`.
- `local-voice-loopback --text-provider stepfun --execute-text-provider` was
  retried twice and failed on the Mac direct path with HTTP header timeout then
  TLS handshake timeout. Treat Mac direct real-time loopback as unstable.

## 2026-06-03 CosyVoice/Clone Candidate Update

- A separate 5080 worker checked local clone-capable TTS assets without
  touching A21 code, firmware, NVS, provider secrets, global proxy, or
  StackChan hardware.
- CosyVoice source and a `.venv` exist on 5080, but the CosyVoice venv lacks
  `torch` and `tqdm`, and no usable `pretrained_models/CosyVoice-*` weights
  were found.
- CosyVoice classes can be imported from the IndexTTS venv and CUDA is
  available there, but no CosyVoice model weights are ready for generation.
- IndexTTS2 is closest to usable: source, venv, runner, CUDA, and partial
  checkpoints exist, but inference currently fails because
  `checkpoints/qwen0.6bemo4-merge/` is missing or not loadable.
- F5-TTS and GPT-SoVITS source traces exist, but no ready checkpoint/run path
  was confirmed.
- No clone WAV was produced. Clone-capable local TTS remains a retained
  `voice_clone_cli` candidate, not the current contest default.
- Recovery is now tracked under
  `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md`.

Next worker should execute `T-PROVIDER-002b`: use the fastest real egress path
for TTS, preferably 5080/Alibaba relay if direct Mac remains blocked, then run
Iflytek smoke, StepFun+TTS loopback, and physical StackChan playback.
