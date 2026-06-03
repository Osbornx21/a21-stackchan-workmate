# A21 Cloud Voice Provider Matrix

Status: branch research and integration contract.
Date: 2026-06-03.
Branch: `codex/a21-pure-cloud-voice-matrix`.

## Purpose

A21 now has a deployed public Gateway path and needs to try multiple pure-cloud
voice options quickly without turning vendor choice into architecture. This
document maps the current Bailian/DashScope, Doubao/Volcengine, and MiniMax
voice APIs onto A21's own provider spine, frontend selection, dispatch, latency,
redaction, and physical StackChan gates.

The target is full support for:

- Bailian Qwen-TTS realtime, including cloned/custom voice IDs where the model
  supports them.
- Bailian CosyVoice realtime families and CosyVoice voice creation/cloning.
- Doubao realtime TTS and Doubao voice-clone model families.
- MiniMax realtime/synchronous TTS and MiniMax voice-clone model families.

Full support means each provider can be listed, configured by the operator UI,
selected explicitly, dispatched server-side without firmware keys, benchmarked
with A21 report fields, and promoted only after real runtime plus physical
StackChan evidence. It does not mean every visible catalog row is immediately
runtime-ready.

## Existing A21 Anchors

Prior A21 work already gives this branch a stable landing zone:

- `voice_mode` is only `dialogue` or `professional`. Cloud voice provider
  selection must not create hidden product modes.
- `gateway_profile` is separate from `voice_mode` and selects `public_wss` or
  `mac_local`; provider selection must not change where StackChan connects.
- `A21_PROVIDER_PROFILES_PATH` already supports A21-namespaced hot-plug text
  profiles for OpenAI-compatible text-stream providers.
- `A21_TTS_FAST_PROFILE` is the current host voice-pipeline TTS selector.
- `A21_GATEWAY_VOICE_PROVIDER=selected` is the explicit runtime opt-in before
  Gateway uses a selected realtime voice provider.
- `doubao_tts_realtime` already has an A21 streaming TTS adapter seam and fake
  runtime smoke coverage. Real provider execution remains separate.
- `voice_clone_cli` remains the A21 retained clone/persona TTS seam for local
  or lab-hosted clone engines; cloud clone APIs should use the same redaction
  and evidence discipline.

## Source Snapshot

Official sources checked for this branch:

- Alibaba Cloud Model Studio Qwen-TTS-Realtime:
  `https://help.aliyun.com/zh/model-studio/qwen-tts-realtime`
- Alibaba Cloud Model Studio CosyVoice realtime WebSocket API:
  `https://help.aliyun.com/zh/model-studio/cosyvoice-websocket-api`
- Alibaba Cloud Model Studio custom voice / voice cloning:
  `https://help.aliyun.com/zh/model-studio/customize-voice-models`
- Alibaba Cloud Model Studio Qwen Omni Realtime:
  `https://help.aliyun.com/zh/model-studio/qwen-omni-realtime`
- Volcengine Doubao realtime TTS overview and WebSocket API:
  `https://www.volcengine.com/docs/6561/1354869`
  and
  `https://www.volcengine.com/docs/6561/1329505`
- Volcengine Doubao voice clone / ICL model docs:
  `https://www.volcengine.com/docs/6561/163025`
  and
  `https://www.volcengine.com/docs/6561/79823`
- MiniMax realtime TTS WebSocket and T2A docs:
  `https://www.minimax.io/platform/document/realtime-speech-synthesis`
  and
  `https://www.minimax.io/platform/document/T2A%20V2?key=66701c7bbc8dd929d08f9a51`
- MiniMax voice clone:
  `https://www.minimax.io/platform/document/Voice%20Clone?key=66719005a427f0c8a5701643`

These sources are API-shape inputs only. A21 promotion still requires A21-owned
runtime reports, redacted latency measurements, and physical StackChan evidence.

## Prior Conversation API Inventory

Memory and repo recovery found these relevant prior decisions:

| Area | Existing fact | A21 implication |
| --- | --- | --- |
| Provider spine | Text providers include DeepSeek, StepFun, DashScope, SiliconFlow, MiniMax text candidates, and local fallback. StepFun/Iflytek relay was the best accepted voice-chain candidate in earlier 5080 work. | Cloud voice work must reuse provider-lane vocabulary instead of introducing a vendor router. |
| Explicit selection | The user previously required front-end/user selection and no silent routing. | Add a frontend-visible cloud voice profile selector; do not infer provider from prompt intent. |
| Doubao realtime TTS | `doubao_tts_realtime` exists in A21 as a streaming TTS adapter seam with fake connection tests and a static readiness gate. | Keep it as the first reusable streaming TTS adapter shape, but reconcile the endpoint/auth/event contract against current Volcengine docs before real execution. |
| CosyVoice / IndexTTS2 | `voice_clone_cli` is retained; a 5080 plan found CosyVoice source but no usable local weights yet, and IndexTTS2 had missing checkpoint assets at that time. | Cloud CosyVoice voice creation can complement local clone work, but it must not silently replace the active contest voice until smoke and listening evidence pass. |
| MiniMax | Prior 5080 text-provider retest had `MiniMax-M2.7-highspeed` as not acceptable for speech-first text default. | That does not reject MiniMax TTS/voice-clone APIs; treat MiniMax voice as a separate TTS/clone lane with its own evidence. |

## A21 Capability Model

Add a new frontend/operator concept: `cloud_voice_profile`.

It is separate from:

- `voice_mode`: product behavior, only `dialogue` or `professional`.
- `gateway_profile`: device transport target, `public_wss` or `mac_local`.
- `A21_PROVIDER_PRIMARY`: text or realtime provider spine selection.

Minimum profile fields:

```json
{
  "schema_version": "a21.cloud_voice_profile.v1",
  "id": "a21_bailian_qwen_tts_realtime",
  "label": "Bailian Qwen-TTS Realtime",
  "vendor": "bailian",
  "family": "voice_hybrid",
  "protocol": "bailian_qwen_tts_realtime_ws",
  "status": "catalog_only",
  "selected": false,
  "configured": false,
  "dispatchable": false,
  "required_env": ["A21_DASHSCOPE_API_KEY", "A21_BAILIAN_QWEN_TTS_MODEL"],
  "present_env": [],
  "missing_env": ["A21_DASHSCOPE_API_KEY", "A21_BAILIAN_QWEN_TTS_MODEL"],
  "safe_config_fields": ["voice_id", "sample_rate_hz", "audio_format", "latency_class"],
  "forbidden_report_fields": ["text", "transcript", "provider_output", "raw_audio", "data_base64", "secret_values", "full_urls"]
}
```

Allowed `dispatch_status` values:

| Status | Meaning |
| --- | --- |
| `catalog_only` | Visible in UI and docs, but no runtime adapter exists yet. |
| `static_ready` | Adapter exists and required env names are present, but no real provider execution has passed. |
| `runtime_smoke_passed` | Real provider smoke passed with redacted A21 report and no physical claim. |
| `candidate` | Host/Gateway candidate evidence exists with p50/p95 metrics. |
| `accepted` | Physical StackChan evidence and A21 readiness gates accept the profile for the scoped path. |

The frontend must show the selected cloud voice profile as an explicit operator
choice. The device registry may expose safe profile labels and statuses, but
StackChan firmware must never receive API keys, provider URLs, clone reference
text, raw audio, or provider-specific event bodies.

## Current No-Execute Control Surface

This branch implements the first control-plane slice without provider
execution:

- `GET /v1/cloud-voice-profiles` returns schema
  `a21.gateway.cloud_voice_profiles.v1`, service `a21-gateway`, the selected
  profile ID, and safe rows for Bailian Qwen/Qwen3/CosyVoice/Omni, Doubao, and
  MiniMax.
- `POST` or `PUT /v1/cloud-voice-profiles` accepts only known safe
  `cloud_voice_profile` IDs. Unknown, legacy, or blocked values are rejected
  without echoing the submitted value.
- Device registry rows expose only `current_cloud_voice_profile`.
- The simulator exposes a compact cloud voice selector and readout alongside
  the existing `voice_mode` and `gateway_profile` selectors.
- `a21 doctor` includes `voice.cloud_voice` using schema
  `a21.cloud_voice_profiles.v1`, reporting selected profile, status, present
  env names, and missing env names only.
- `a21_doubao_tts_realtime` may become `static_ready` when its existing
  adapter seam and required env names are present. Other configured profiles
  stay `catalog_only` until their adapters and fixture tests land.

## Provider Mapping

| Provider family | Official API shape | A21 profile IDs | First A21 adapter target | Notes |
| --- | --- | --- | --- | --- |
| Bailian Qwen-TTS Realtime | WebSocket realtime TTS. Qwen-TTS-Realtime uses `session.update`, `input_text_buffer.append`, and either server-commit or manual commit mode. Current docs distinguish system voices for Qwen TTS from custom voice IDs on Qwen3-TTS-VC-Realtime. | `a21_bailian_qwen_tts_realtime`, `a21_bailian_qwen3_tts_vc_realtime` | `StreamingTTSAdapter` producing 60 ms `pcm_s16le` mono chunks. | Treat custom voice IDs as server-side safe config, not model values in reports. |
| Bailian CosyVoice realtime | DashScope WebSocket inference API. CosyVoice families support realtime synthesis and custom voice flows; newer CosyVoice 3.5 variants are Beijing-region only in current docs and are custom-voice oriented. | `a21_bailian_cosyvoice_realtime`, `a21_bailian_cosyvoice_clone_tts` | `StreamingTTSAdapter` plus a voice-clone management command. | Must store only voice-id labels and basenames. No reference text or audio payloads in reports. |
| Bailian Qwen Omni Realtime | WebSocket speech-to-speech / multimodal realtime session. | `a21_bailian_qwen_omni_realtime` | `RealtimeVoiceProvider` behind explicit opt-in. | Never routes professional mode; requires one-shot realtime arming and cancellation proof. |
| Doubao realtime TTS | Volcengine realtime / bidirectional TTS APIs with streaming text input and audio output. Existing A21 code uses `tts_session.update`, `input_text.append`, `input_text.done`, and `response.audio.delta` shape. | `a21_doubao_tts_realtime` | Reconcile existing `DoubaoRealtimeTTSProvider` against current endpoint/auth docs, then real smoke. | This is the current reusable streaming TTS seam. |
| Doubao voice clone | Volcengine voice clone models include ICL-style clone and generated voice IDs used by TTS. | `a21_doubao_voice_clone_tts` | Clone command/report surface, then TTS profile dispatch. | Clone source/reference consent and redaction must be explicit. |
| MiniMax realtime/sync TTS | MiniMax WebSocket T2A sends `task_start`, `task_continue`, and `task_finish`; HTTP T2A v2 also exists. | `a21_minimax_t2a_ws`, `a21_minimax_t2a_http` | WebSocket TTS adapter if it proves audio chunks before final; otherwise keep HTTP as quality/fallback TTS, not strict realtime. | MiniMax output format defaults in docs are often compressed audio; A21 needs PCM/Opus conversion proof before StackChan downlink promotion. |
| MiniMax voice clone | Upload file, obtain `file_id`, call voice clone with A21-owned `voice_id`, then use that voice in TTS. | `a21_minimax_voice_clone_tts` | Clone management command plus MiniMax TTS adapter. | `file_id` and `voice_id` are sensitive operational IDs; reports should store only safe labels or hashed/basename references. |

## Proposed Environment Names

All names remain A21-owned:

| Provider | Env names |
| --- | --- |
| Bailian Qwen-TTS | `A21_DASHSCOPE_API_KEY`, `A21_BAILIAN_QWEN_TTS_MODEL`, `A21_BAILIAN_QWEN_TTS_VOICE_ID`, `A21_BAILIAN_QWEN_TTS_SAMPLE_RATE_HZ`, `A21_BAILIAN_QWEN_TTS_FORMAT` |
| Bailian CosyVoice | `A21_DASHSCOPE_API_KEY`, `A21_BAILIAN_COSYVOICE_MODEL`, `A21_BAILIAN_COSYVOICE_VOICE_ID`, `A21_BAILIAN_COSYVOICE_SAMPLE_RATE_HZ`, `A21_BAILIAN_COSYVOICE_FORMAT` |
| Bailian Omni | `A21_DASHSCOPE_API_KEY`, `A21_BAILIAN_OMNI_REALTIME_MODEL`, `A21_BAILIAN_OMNI_REALTIME_VOICE_ID` |
| Doubao | Existing `A21_DOUBAO_API_KEY` or `A21_DOUBAO_ACCESS_TOKEN`, `A21_DOUBAO_TTS_MODEL`, `A21_DOUBAO_TTS_VOICE`, `A21_DOUBAO_TTS_SAMPLE_RATE_HZ`; add `A21_DOUBAO_TTS_ENDPOINT_PROFILE` only if official endpoints diverge by model family. |
| MiniMax | `A21_MINIMAX_API_KEY`, `A21_MINIMAX_GROUP_ID`, `A21_MINIMAX_TTS_MODEL`, `A21_MINIMAX_VOICE_ID`, `A21_MINIMAX_TTS_SAMPLE_RATE_HZ`, `A21_MINIMAX_TTS_FORMAT` |

Reports may include env names, endpoint host labels, provider family, latency
statistics, and safe profile IDs. Reports must not include env values, model
values, raw voice IDs when they are operational secrets, prompt text,
transcripts, provider output, raw/base64 audio, full URLs, proxy values, local
paths, or reference text.

## Frontend And Dispatch Contract

Add these server-side surfaces in a later implementation slice:

| Endpoint | Purpose | No-secret rule |
| --- | --- | --- |
| `GET /v1/cloud-voice-profiles` | List available cloud voice profiles, statuses, capabilities, and missing env names. | Never returns values, full URLs, voice clone reference text, or payloads. |
| `POST /v1/cloud-voice-profiles` | Select an explicit profile for `dialogue` TTS or realtime test. | Accepts only safe profile IDs and safe config labels. |
| `GET /v1/cloud-voice-profiles/plan` | Planned dry-run readiness plan for the selected profile. | Same redaction as doctor/provider-realtime-plan. |
| `POST /v1/cloud-voice-profiles/dispatch` | Planned runtime dispatch for an already configured profile. | Requires explicit execute flag or operator window; not a hidden prompt router. |

The UI must present:

- `voice_mode`: `dialogue` or `professional`.
- `gateway_profile`: `public_wss` or `mac_local`.
- `cloud_voice_profile`: the selected ASR/TTS/S2S/clone voice profile.
- safe status: `catalog_only`, `static_ready`, `runtime_smoke_passed`,
  `candidate`, or `accepted`.
- missing env names and next required evidence.

`professional` remains V21-only. A cloud voice profile can speak professional
feedback after the V21 adapter returns speech blocks, but it must not become the
professional retrieval route.

## Latency And Evidence Gates

Each cloud voice adapter must feed the existing A21 benchmark contract:

- `tts_request_to_first_audio_ms`
- `first_llm_token_to_first_tts_audio_ms`
- `provider_commit_to_first_audio_ms`
- `gateway_downlink_first_frame_ms`
- `device_downlink_first_frame_ms`
- `device_playback_start_ms`
- `speech_end_to_first_audible_response_ms`
- `barge_in_stop_ms`
- `provider_cancel_ms`

Required promotion sequence:

1. Static plan: catalog row plus missing/present env names.
2. Fixture test: fake provider events prove parsing, chunking, cancel, and
   redaction.
3. Runtime smoke: real provider execution on approved lab host, no audio
   payload persisted, no physical claim.
4. Host/Gateway candidate: `/v1/xiaozhi` or voice-pipeline trace with real
   streaming ASR/text/TTS profile markers.
5. Physical StackChan evidence: device downlink receipt, playback start,
   audible/operator acceptance, barge-in stop, and no secret leakage.

## Implementation Slices

Recommended order:

1. `T-CLOUD-VOICE-001`: Add the catalog/profile schema, docs, no-execute
   `GET/POST /v1/cloud-voice-profiles`, simulator selector, and doctor report.
2. `T-CLOUD-VOICE-002`: Add provider-neutral streaming TTS adapter tests and
   a common chunker contract reusable across Qwen/CosyVoice/Doubao/MiniMax.
3. `T-CLOUD-VOICE-003`: Reconcile and harden Doubao realtime TTS against
   current Volcengine docs, then run real smoke only on an approved lab host.
4. `T-CLOUD-VOICE-004`: Add Bailian Qwen-TTS realtime adapter and fixture
   smoke.
5. `T-CLOUD-VOICE-005`: Add Bailian CosyVoice adapter and voice-clone command
  /report contract.
6. `T-CLOUD-VOICE-006`: Add MiniMax TTS and voice-clone adapters, with
   compressed-output-to-PCM conversion proof if needed.
7. `T-CLOUD-VOICE-007`: Add safe dispatch state; do not put provider keys in
   browser/localStorage/firmware.
8. `T-CLOUD-VOICE-008`: Run 5080/mainland runtime smoke matrix and ingest
   redacted reports.
9. `T-CLOUD-VOICE-009`: Run physical StackChan A/B, barge-in, and quality
   acceptance.

## Open Questions

- Which clone reference voices are consented for lab use, and where should they
  live outside Git?
- Should A21 prefer PCM output from every provider, or accept provider Opus/MP3
  and normalize in Gateway? PCM is simpler for downlink timing evidence; Opus
  may reduce egress bandwidth if provider support is clean.
- Should Qwen Omni realtime be exposed in the same frontend selector as TTS
  profiles or behind a separate one-shot realtime arming control?
- Which profile becomes the first 5080 real-smoke target after Doubao: Bailian
  Qwen-TTS realtime, CosyVoice, or MiniMax?

## Hard Red Lines

- No provider API key in firmware, frontend persistent storage, docs, reports,
  traces, logs, screenshots, or Git.
- No hidden auto-router that changes cloud voice provider from prompt content.
- No cloud provider event body reaches StackChan; Gateway converts to A21
  semantic state and 60 ms media chunks.
- No professional/V21 route through opaque speech-to-speech provider.
- No `accepted` status without A21 runtime evidence plus physical StackChan
  evidence for the claimed behavior.
