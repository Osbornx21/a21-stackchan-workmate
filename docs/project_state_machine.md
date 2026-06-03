# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-03.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-STACKCHAN-COMPATIBLE-CURRENT-HEAD-NO-WELCOME-REFLASHED`

Active child transitions:

- `T-WAKE-003-ZI-YUE-PHRASE-TUNING`
- `T-ASR-GREEN-LATENCY-001-XIAOZHI-LISTEN-AUTO-STOP`
- `T-ASR-GREEN-LATENCY-002-XIAOZHI-NO-SPEECH-COOLDOWN`
- `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`
- `T-AUDIO-BARE-XIAOZHI-PARITY-001`
- `T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001`
- `T-XIAOZHI-REALTIME-VOICE-PARITY-001`
- `T-XIAOZHI-STREAMING-ASR-001`
- `T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001`
- `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`
- `T-XIAOZHI-STALE-OPUS-INGRESS-SUPPRESSION-001`
- `T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE`
- `T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY`
- `T-XIAOZHI-STOCK-HALF-DUPLEX-ACCEPTANCE-001`
- `T-STACKCHAN-WIFI-PROVISIONING-001-XIAOZHI-STYLE`
- `T-XIAOZHI-HOST-LOCAL-REAL-BASIC-DIALOGUE-SMOKE`
- `T-VOICE-CHAIN-EVIDENCE-001-SELECTED-VOICE-CHAIN-READINESS-INGRESS`
- `T-COSYVOICE-5080-LOCAL-CLONE-CANDIDATE-CHECK`
- `T-CLOUD-VOICE-001-PURE-CLOUD-PROVIDER-MATRIX`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has been flashed in a
foreground hardware window, the latest firmware enters the official Xiaozhi
runtime directly, and the device has now connected to an A21 Gateway over the
stock Xiaozhi WebSocket path. A physical wake/turn produced Gateway uplink,
downlink, and barge-in candidate evidence. The user reports the audible sound
is now materially cleaner after the stock-audio hotfix, but still not at
expected maximum loudness. The 2026-06-02 emergency RCA split the problem into
official audio parity, runtime speaker-volume control, TTS source quality, and
wake-word activation rather than treating it as only loudness. The accepted
3x Gateway gain is now frozen for the contest path. `T-PROVIDER-002` has
landed host-side hot-plug code for the 5080-recommended StepFun text stream
plus Iflytek 16 kHz PCM TTS while preserving stock Xiaozhi firmware/protocol
boundaries; commit `b5a405e` is the current integration baseline. The old
`sherpa_onnx` local voice is now only an emergency/diagnostic fallback for this
contest path; `voice_clone_cli` remains the retained A21 voice-clone seam for
IndexTTS2/CosyVoice/F5-TTS/GPT-SoVITS and must not be removed just because the
immediate fast-dialogue path selects Iflytek. The 2026-06-03 control-tower
worker fan-out refreshed selected-provider readiness without falling back to
`mock`, confirmed the StepFun+Iflytek relay chain remains voice-chain candidate
evidence, and separated the no-flash normal-dialogue observation path from the
optional diagnostic half-duplex counter path. The no-flash self-trigger
observation now has candidate evidence, while custom wake and clone-capable
local TTS remain explicit recovery tasks rather than hidden blockers.
`T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` is now a passed real-model no-audio
smoke: the repo-local canonical helper, sherpa-onnx Python environment, and
streaming Zipformer model cache were discovered automatically, the JSONL
helper loaded the real model, appended one in-memory 60 ms PCM16 frame, and
committed a final event without writing WAV, recording transcript text, storing
raw audio, calling providers/V21, starting Gateway, or touching hardware.
`T-STREAMING-TTS-RUNTIME-PROOF-001` is now recorded as a redacted runtime smoke
tool with truthful no-execute blocker: fake realtime tests prove provider audio
delta to exact 60 ms PCM16 mono chunking before any WAV/file boundary, while the
local default report records `execute_flag_required` and no real provider
runtime pass is claimed.
`T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001` is now a host-side bridge
candidate: stock `/v1/xiaozhi` streaming ASR partial events can start the
workmate LLM/TTS streaming answer before explicit `listen.stop` or `asr.final`,
while the existing final-transcript and batch ASR fallback paths remain
available. This is unit/Gateway/parity evidence only, not real Sherpa runtime,
real provider execution, or physical PRD acceptance.
`T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001` now hardens the
realtime parity gate against a subtler false green: a trace with good event
ordering but missing real streaming ASR/LLM/TTS profile-class markers is
downgraded from `xiaozhi_realtime_candidate` and receives
`xiaozhi_realtime_real_profile_evidence_missing`. Gateway records only
category markers such as `llm.mock_blocked`, `tts.file_boundary_blocked`, or
`tts.real_streaming`; it does not store raw provider names, transcripts,
provider outputs, URLs, credentials, paths, or audio payloads.
`T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001` now moves `listen.stop` and VAD
auto-stop off the blocking ASR commit path: streaming ASR commit/finalization
runs asynchronously, the WebSocket read loop can process abort/barge-in while
commit is pending, and final-driven voice-pipeline startup uses the streaming
ASR final without falling back to batch `Transcribe`.
`T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001` is now a
host-local stock-protocol fidelity cut: A21 emits stock `stt` before `tts`
when streaming ASR partial/final text exists, and buffers a bounded idle
wake pre-roll of decoded Opus frames so wake-adjacent audio can feed the next
`listen/start` turn instead of being dropped. This remains below PRD
acceptance because it is unit/Gateway evidence only, with no real provider
execution, physical stock trace, wake/tap proof, barge-in/touch evidence, or
audible playback.
`T-XIAOZHI-OPUS-INGRESS-QUEUE-001` now adds the missing host-side Opus frame
queue boundary: listening Opus frames are enqueued on a bounded per-session
queue before decode/VAD/streaming-ASR append, so the stock WebSocket control
loop can still read `abort` while ASR append is slow. `listen.stop`
finalization now waits asynchronously for queued ingress to catch up before
committing streaming ASR or starting the voice pipeline. This is still
host-local evidence, not physical Xiaozhi PRD acceptance.
`T-XIAOZHI-STALE-OPUS-INGRESS-SUPPRESSION-001` now closes the next queue
ownership gap: Opus ingress items whose turn context has already been cancelled
are suppressed before decode, audio ingress, VAD, or streaming ASR append. This
keeps abort/wake-as-abort/listen-start barge-in from letting old queued audio
pollute the next listening turn. It is unit/Gateway evidence only, not real
provider execution or physical PRD acceptance.
`T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE` is active as the mode
contraction slice: user-facing product modes converge to `dialogue` and
`professional`; legacy labels such as `workmate`, `companion`, `co_creation`,
and `roleplay` normalize to `dialogue` at product-contract surfaces. The
dialogue readiness report now identifies the static `dialogue_low_latency`
chain and accepts Doubao realtime TTS configuration via env-name-only
`A21_DOUBAO_API_KEY` or `A21_DOUBAO_ACCESS_TOKEN`, while keeping
`professional` on the V21 adapter boundary and `prd_accepted=false`.
`T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY` is active as the main public
transport profile slice: with a valid `A21_PUBLIC_GATEWAY_URL`, `public_wss`
is the default product voice-edge Gateway, while `mac_local` remains
selectable for Mac-local models and local processing. Gateway exposes
`/v1/gateway-profiles`, simulator can select the profile, and
`/xiaozhi/ota/` returns either request-host local `ws`/`wss` or configured
public `ws`/`wss` `/v1/xiaozhi`. The earlier Aliyun SWAS host `101.132.117.182`
proved the experimental chain under systemd/nginx with self-signed `443`, but
the new ECS `47.103.57.217` / `i-uf63f4ymqc2dxtljxz2n` is now running the main
A21 voice-edge Gateway under systemd on `127.0.0.1:21081` behind Caddy public
`80/443`. External HTTP health/profile/OTA checks pass, OTA returns
`ws://47.103.57.217/v1/xiaozhi`, and temporary self-signed HTTPS passes only
with `-k`. Trusted TLS, secure provider-secret injection, real cloud
ASR/LLM/TTS execution, physical StackChan public-WSS evidence, and wake-word
product proof remain open before PRD acceptance.
The 2026-06-03 public-edge provider push moved this slice beyond host-only
reachability: ECS `47.103.57.217` now reads provider configuration from
root-only `/etc/a21/secrets/provider.env`, runs a DashScope CosyVoice wrapper
through A21's existing `voice_clone_cli` TTS seam, and delivered real
DashScope-generated audio to the physical StackChan over stock
`/v1/xiaozhi`. Evidence trace
`a21-trace-public-edge-dashscope-say-1780472526` returned
`status=delivered`, `delivered_transport=xiaozhi_ws`, `audio_chunks=87`,
`tts.first_audio=2159 ms`, `audio.downlink.first_frame=2160 ms`, and
`xiaozhi.say.delivered` for device `44:1b:f6:e2:6a:60`. The matching remote
TTS smoke `/tmp/a21-provider-smoke/a21-local-tts-smoke-20260603-154156.json`
passed with `provider=voice_clone_cli`, `model=cosyvoice_v3_flash`, and a
redacted PCM16 mono WAV quality report. Doubao realtime TTS and the attempted
Iflytek mapping remain blocked by provider `401`; their credentials must not
be treated as A21-ready Realtime/Iflytek credentials. Public host-loopback
bench report `reports/a21-xiaozhi-voice-bench-20260603-154310.624051000.json`
proved virtual answer and abort timing (`barge_in_stop_p95_ms=1`) but stayed
`prd_accepted=false`. Physical half-duplex report
`reports/a21-stackchan-half-duplex-acceptance-20260603-154254.json` correctly
blocked because the stock product firmware is not the diagnostic mic-probe
runtime echo lane. Full PRD remains blocked by operator audible confirmation,
stock physical mic-driven dialogue, normal half-duplex/barge-in proof, and
wake-word product proof.
`T-XIAOZHI-STOCK-HALF-DUPLEX-ACCEPTANCE-001` is active as the product-firmware
half-duplex evidence correction. A new `stackchan-accept --check
xiaozhi-half-duplex` gate now reads stock `/v1/xiaozhi` Gateway traces and
`/v1/audio/recent`, can derive the latest trace/session from `/v1/devices`, and
writes `a21.xiaozhi_half_duplex_acceptance.v1` reports with
`hardware_acceptance_scope=stock_xiaozhi_mic_to_tts_downlink` and
`diagnostic_mic_probe_required=false`. It does not require the diagnostic
mic-probe capability or runtime echo counters, so product stock Xiaozhi
firmware is no longer misclassified by the older diagnostic half-duplex gate.
The first public run against `47.103.57.217` produced
`reports/a21-xiaozhi-half-duplex-acceptance-20260603-161717.013784000.json`:
device `44:1b:f6:e2:6a:60` was online with stock Xiaozhi transport, but the
latest trace was hello-only with zero audio frames, no downlink, no playback
ack, and no barge-in stop evidence, so status correctly remained `blocked` and
`prd_accepted=false`.
The 2026-06-03 post-rebuild/no-sound audit found that the endpoint and
Gateway voice changes were not broadly erased by the clean Xiaozhi refresh:
the current HEAD still contains the wake/listen/VAD, Opus queue, stale ingress
suppression, public Gateway, DashScope downlink, and stock half-duplex
acceptance commits; the latest clean product build/flash still used
`a21-stackchan-official-xiaozhi-compatible.bin` with custom `紫悦` wake config
and A21 control-channel guards. The current runtime break is on the public
Gateway product chain: ECS `47.103.57.217` is still launched with
`--product-chain host_local`, has TTS/text env keys but no ASR env, defaults
missing ASR to `A21_ASR_LOCAL_PROFILE=sherpa_onnx`, and therefore drives real
physical `/v1/xiaozhi` turns into `xiaozhi.voice_pipeline.unavailable` and
`local_fallback`. This keeps physical dialogue and audible acceptance blocked
until the public Gateway has a cloud-compatible ASR/LLM/TTS chain and a fresh
physical trace proves real provider execution.
`T-STACKCHAN-WIFI-PROVISIONING-001-XIAOZHI-STYLE` is active as the endpoint
provisioning contract: A21 no longer treats missing Wi-Fi credentials in the
self-owned firmware state model as local fallback; it enters
`Wi-Fi provisioning` and exposes Xiaozhi's three startup provisioning methods
as `hotspot`, `blufi`, and `acoustic`, with `hotspot` as default. The
official-compatible product overlay now explicitly selects Hotspot/SoftAP
provisioning, keeps BluFi and acoustic disabled but named as build-time
alternatives, points OTA at the main public Gateway
`http://47.103.57.217/xiaozhi/ota/`, and still stores no provider keys or Wi-Fi
credentials in repo/firmware. The product lane was rebuilt no-flash with the
current overlay and produced
`/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
with app SHA-256
`c012542ee8f1837106da91fe934e27487813333eaad124841386706259f48659`; report
`reports/a21-stackchan-official-baseline-20260603-150433-1780470273395485000.json`
is `status=passed`, `official_avatar_action_preserved=true`, and
`official_xiaozhi_start_preserved=true`. This is still below physical
acceptance until flashed and observed through a guarded hardware window.
`T-CLOUD-VOICE-001-PURE-CLOUD-PROVIDER-MATRIX` is active as a side-branch
research/control transition for the newly deployed public Gateway era. It
creates the provider-neutral contract for fully supporting Bailian Qwen-TTS
Realtime and CosyVoice, Doubao realtime/clone voice families, and MiniMax TTS
and clone families without changing A21 defaults. The contract keeps
`cloud_voice_profile` separate from `voice_mode` and `gateway_profile`, keeps
`professional` on the V21 adapter path, and requires frontend/operator
selection, server-side dispatch, A21 latency reports, redacted provider smoke,
and physical StackChan evidence before any cloud voice profile can become
accepted. The side branch now also implements the first no-execute control
surface: `GET/POST /v1/cloud-voice-profiles`, simulator selector/readout,
device-registry `current_cloud_voice_profile`, and `a21 doctor`
`voice.cloud_voice` report. The implemented surface returns only safe profile
IDs, statuses, capabilities, present env names, and missing env names; it does
not execute providers, does not change `voice_mode` or `gateway_profile`, and
does not expose provider keys, model values, voice IDs, URLs, prompt text,
transcripts, or audio payloads.
The fixed
official codec output-volume candidate is already prepared in the repo-owned
Xiaozhi-compatible overlay. The no-write
official-compatible build/report passed with app SHA-256
`2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
The matching candidate was flashed through an operator-approved foreground
hardware window on `/dev/cu.usbmodem1101`; flash report
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
is `status=passed`, `dry_run=false`, `flash_allowed=true`, and
`flash_executed=true`. Gateway `127.0.0.1:21081` remained healthy after flash
and the physical device `44:1b:f6:e2:6a:60` remained registered online. Physical
audible A/B for the volume-92 candidate failed: operator recording
`/Users/jiyurun/Downloads/军民公路259号 7.m4a` measured `-33.4 LUFS` integrated
loudness and `-14.3 dBFS` true peak, materially weaker than the 22:19 reference
sample at `-25.8 LUFS` and `-4.2 dBFS`. Analysis report
`reports/a21-operator-recording-audio-analysis-20260602-2310-stackchan-volume92.json`
sets `post_flash_loudness_accepted=false`. The follow-up operator recording
`/Users/jiyurun/Downloads/军民公路259号 8.m4a`, analyzed from 3 seconds onward,
improved to `-31.78 LUFS` integrated and `-10.56 dBTP` true peak versus
`7.m4a` from 3 seconds at `-35.56 LUFS` and `-14.65 dBTP`, but still leaves
roughly 10 dB peak headroom. Host hotfix tests now pass for
stock MCP speaker-volume delivery, stock physical `listen` reply suppression,
24 kHz downlink TTS chunks, turn-level Opus encoder reuse, foreground
`/v1/xiaozhi/say` delivery, Gateway playback, voice-bench, and
app/gateway/provider/transport focused paths. The live Gateway on
`127.0.0.1:21081` delivered runtime volume `100` through stock MCP and pushed a
long physical TTS turn through `/v1/xiaozhi/say` with trace
`a21-trace-live-long-tts-hotfix`, `text_chars=189`, and `audio_chunks=676`.
Explicit
real/local provider evidence exists, but
generic readiness refreshes can still fall back to `mock` if they omit the
selected provider report/env. It is not yet full PRD accepted because audible
playback observation, broader half-duplex acceptance, and custom wake proof
remain missing. Follow-up recording
`/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` after bounded Gateway
downlink gain measured `-21.65 LUFS` and `-4.72 dBTP` from 3 seconds, with no
obvious clipping in FFmpeg `astats`, but the operator later reported that this
4x gain candidate felt like a slight regression with subtle electrical
interruption and reduced clarity. Follow-up recording
`/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured
`-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds; the active code
candidate now uses a safer 3x max-gain cap while preserving the same stock
protocol, 24 kHz downlink, Opus reuse, and host-say suppression. The 3x code
was relaunched in tmux Gateway session `a21-gateway-21081`, runtime volume
`100` was delivered by trace `a21-trace-stackchan-volume-1780417211`, and
physical `/v1/xiaozhi/say` trace `a21-trace-stackchan-say-1780417217`
delivered `text_chars=176`, `audio_chunks=579`, `answer_first_audio_total_ms=1656`,
`xiaozhi.say.input_suppression_armed=1`, and
`xiaozhi.listen.start.input_suppressed=1`. Operator acceptance recording
`/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and
`-8.6 dBFS` true peak from 3 seconds, and the operator explicitly accepted the
3x audio result. Audio quality is accepted for the current contest path, while
normal dialogue half-duplex remains a separate follow-up and full PRD launch
acceptance is still blocked by custom wake proof. Custom wake-word package
evidence exists, but the 2026-06-03 foreground hardware window proved an
important negative guard: flashing the restored package's bare `xiaozhi.bin`
through `xiaozhi-firmware-flash-execute` overwrote the StackChan avatar/product
app with a plain Xiaozhi UI. That incident was rolled back immediately by
flashing the correct `a21-stackchan-official-xiaozhi-compatible.bin` product
candidate on `/dev/cu.usbmodem1101`; restore flash report
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`
is `status=passed`, `flash_allowed=true`, and `flash_executed=true`.
Post-restore Gateway `127.0.0.1:21081` saw a fresh physical hello from
`44:1b:f6:e2:6a:60`; runtime volume `100` delivered on trace
`a21-trace-recovery-stackchan-volume-1780424386012`; the accepted 5080
StepFun+Iflytek relay WAV delivered through stock `/v1/xiaozhi/say` on trace
`a21-trace-recovery-stackchan-relay-wav-1780424386012` with
`audio_chunks=40`. `T-WAKE-002` rebuilt custom wake inside the
StackChan-compatible product lane instead of the bare `xiaozhi.bin` lane:
plan `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`; build report
`reports/a21-stackchan-official-baseline-20260603-030428-1780427068064697000.json`
is `status=passed`, app SHA-256
`3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`, with
`official_avatar_action_preserved=true`,
`official_xiaozhi_start_preserved=true`, and `minimal_bridge_screen=false`.
Build config proves StackChan board identity, `USE_CUSTOM_WAKE_WORD=true`,
`CUSTOM_WAKE_WORD="zi yue"`, `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`, threshold
`20`, `SR_MN_CN_MULTINET7_QUANT=true`, `USE_AFE_WAKE_WORD=false`, and stock
HiStackChan WakeNet disabled. The overlay intentionally uses a contest-recovery
direct autostart: it sets codec volume and calls `GetHAL().startXiaozhi()`
before the welcome/setup app flow, because the official request path trapped the
physical device on the welcome screen. Retaining official app loading without
showing welcome/setup is now a follow-up transition. No-write flash plan
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030447-1780427087258380000.json`
was clean for `/dev/cu.usbmodem1101`. The first post-commit flash execute was
blocked by the T7 guard because the welcome-recovery hotfix had made the
worktree dirty; commit `7f3225e` then restored a clean control state. Guarded
flash execute
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json`
passed with app `a21-stackchan-official-xiaozhi-compatible.bin`, app offset
`0x20000`, app SHA-256
`3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`, and
control commit `7f3225ee1fe7`. Gateway `127.0.0.1:21081` stayed healthy; the
physical device `44:1b:f6:e2:6a:60` reconnected online at 03:08:51 with trace
`a21-trace-44-1b-f6-e2-6a-60` and speaker volume `100`. Physical wake proof is
now rejected by the operator: saying `紫悦` produced no response. A separate
green-ASR latency problem is also active: live trace reuse showed multiple long
stock Xiaozhi listen windows, including about 25s, 38s, 70s, and 108s before
auto-stop or replacement by a new listen. The current unflashed hotfix candidate
keeps the display identity `紫悦`, adds longer MultiNet command aliases, adds a
Gateway max-listen safety stop with `A21_XIAOZHI_LISTEN_MAX_MS`, and improves
trace summary pairing for reused hardware trace ids. Commit `5242349` was
built and flashed through the official-compatible product lane: build report
`reports/a21-stackchan-official-baseline-20260603-032615-1780428375746120000.json`;
no-write flash plan
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032627-1780428387219884000.json`;
flash execute
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032734-1780428454747199000.json`;
app SHA-256
`e13a6cbd63596cd1388f5a540b488b3e35b449e49a5e31891cf133436561dc6e`.
Gateway `a21-gateway-21081` was restarted from this commit with
`A21_XIAOZHI_LISTEN_MAX_MS=7000`, device `44:1b:f6:e2:6a:60` reconnected
online, and runtime speaker volume `100` was delivered on trace
`a21-trace-wake-asr-hotfix-volume-1780428544`. `T-FLASH-GUARD-001` is now integrated: the
generic `xiaozhi-firmware-flash-*` lane rejects product-looking `xiaozhi.bin`
at app offset `0x20000` unless explicitly marked `--non-product-dev`, while
the official-compatible product flash plan still passes.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `ebc0db4 docs(control): record stackchan flash convergence`
- `18dc553 docs(control): prepare stackchan volume flash gate`
- `f0603f5 fix(firmware): set official xiaozhi codec volume`
- `4613946 fix(firmware): enter official xiaozhi runtime directly`
- `eeacbd3 docs(control): recover hardware network state`
- `e7e9b03 feat(firmware): autostart official xiaozhi candidate`
- `37ef8f3 feat(firmware): add official xiaozhi nvs connection config`
- `6f34091 feat(firmware): add official xiaozhi compatible flash plan`
- `d28560a docs(control): dispatch stackchan volume worker`
- `f49abde docs(audio): plan stackchan volume control`
- `751de08 docs(audio): record stock xiaozhi playback boundary`
- `59f30f4 docs(control): record official flash seam worker dispatch`
- `69c4bbe docs(control): add handoff and state machine workflow`
- `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`
- `060d2bb fix(audio): accept stackchan xiaozhi playback hotfix`
- `b5a405e feat(audio): add hot-pluggable iflytek tts candidate`

## Module States

| Module | State | Evidence | Next state |
| --- | --- | --- | --- |
| Control workflow | `S1-REPO-CARRIED-CONTROL` | Commit `69c4bbe`; `docs/agent_handoff_log.md`, `docs/project_state_machine.md`, and `docs/plans/` exist | `S2-WORKER-TRANSITION-OPERATING` |
| Gateway `/v1/xiaozhi` | `S3-PHYSICAL-DEVICE-CONNECTED` | Gateway on `127.0.0.1:21081` / LAN port `21081` accepted the physical device via stock Xiaozhi WebSocket; trace `a21-trace-44-1b-f6-e2-6a-60` has Opus uplink, VAD, ASR final, provider first content, TTS first audio, and Opus downlink | `S4-AUDIBLE-PLAYBACK-ACCEPTED` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S5N-NO-WELCOME-IDLE-SOCKET-CANDIDATE` | Commit `a986d6b` request-start physical flash regressed to the welcome/setup trap. Hotfix commit `9ba8bc1` moved the direct Xiaozhi start before visible setup, and `e694550` parked `app_main` after `GetHAL().startXiaozhi()` because `startXiaozhi()` returns and otherwise falls into the Mooncake welcome/setup loop. After the operator reported a welcome regression again, the control tower rebuilt the current HEAD `7906975` product app through the guarded official-compatible lane. Build report `reports/a21-stackchan-official-baseline-20260603-062637-1780439197207182000.json` passed with app SHA-256 `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`; flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062715-1780439235852990000.json` was ready; guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json` passed on `/dev/cu.usbmodem1101` from clean commit `790697518c3e`. Device `44:1b:f6:e2:6a:60` is online on Gateway `21081`; `/v1/devices` reports `last_event=xiaozhi.hello`, `speaker_volume=100`, and trace `a21-trace-44-1b-f6-e2-6a-60`. The operator later confirmed setup is fixed. Recovery trace analysis found the old touch-start episode had only one `xiaozhi.listen.start` and one `xiaozhi.listen.stop`, with `xiaozhi.no_speech.input_suppression_armed=1`, `xiaozhi.listen.start.input_suppressed=1`, and `xiaozhi.listen.start.suppressed_after_no_speech=1`; the latest trace event is `xiaozhi.hello.received`, so current state is idle socket connected rather than infinite listening. Runtime speaker volume `100` was re-delivered on trace `a21-trace-recovery-volume-1780441733`. Physical wake and touch retry remain pending. | `S5O-WAKE-AND-TOUCH-PHYSICAL-ACCEPTED` |
| Device connection/NVS | `S3-LAN-GATEWAY-CONNECTED` | Latest guarded NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json` pointed OTA/WS to the LAN-bound A21 Gateway; reset serial log shows OTA connection to `21081` and activation | `S4-STABLE-RECONNECT-EVIDENCE` |
| Provider hot-plug | `S4A-SELECTED-PROVIDER-READY-VOICE-CHAIN-CANDIDATE` | Read-only worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` confirmed usable explicit provider evidence: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json` passed with `executed=true`, `stream=true`, `repeat=5`, `route_eligible=true`, and `first_content_p95_ms=745.516`; local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json` passed when selected explicitly; readiness `reports/a21-product-readiness-20260602-160841.json` has `provider.real_provider_ready=true` and `server_side.provider_evidence_ready=true`. 5080 report `outbox/A21-VOICE-FULL-REPORT.md` was read through the established `5080lab` outbox lane and recommends StepFun `step-1-8k` for real-time text stream plus Iflytek TTS for real-time 16 kHz PCM synthesis. The source report contained plaintext credentials, so repo docs/logs record only env names and redacted provider identity. Current code lets explicit StepFun/compatibility text-stream profiles run without promoting product route eligibility, exposes `iflytek_tts` through CLI and voice-pipeline TTS selection, and preserves `voice_clone_cli` as the voice-clone path. Worker `019e8954-e65c-72a2-9847-a17a59a0ad6b` unblocked the host chain through 5080 relay: Iflytek TTS report `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json` passed with first audio `100.299 ms`, and StepFun+Iflytek chain report `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json` passed with text first content `229.077 ms` and TTS first audio `87.947 ms`. After deploying the new Gateway handler, relay WAV playback through stock `/v1/xiaozhi/say` delivered `audio_chunks=40` on trace `a21-trace-provider-playback-wav-1780450901`; operator feedback was positive: "好多了". Worker `019e8974-346e-7912-93b2-77cdbb9f3acf` then refreshed selected-provider readiness with route-eligible DeepSeek evidence: `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json` has `provider.selected=deepseek`, `provider.real_provider_ready=true`, and `provider.smoke_status=passed`; `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json` has `provider.ready=true`. StepFun/Iflytek remains voice-chain candidate evidence rather than being forced into the product provider slot. | `S5-VOICE-CHAIN-EVIDENCE-INGRESS-OR-LONG-DIALOGUE` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S3D-STACKCHAN-COMPATIBLE-RESTORED-RELAY-WAV-OK` | Physical stock Xiaozhi path was online; runtime volume `100` and 3x `/v1/xiaozhi/say` were delivered to `44:1b:f6:e2:6a:60`; accepted recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds. On 2026-06-03 the Gateway was restarted from this branch, device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered via stock MCP trace `a21-trace-relay-playback-volume-1780450901`, and relay WAV playback trace `a21-trace-provider-playback-wav-1780450901` delivered `40` stock Xiaozhi Opus downlink chunks. After the mistaken bare wake/Xiaozhi flash, corrective StackChan-compatible app flash restored the product candidate; post-restore runtime volume `100` delivered on trace `a21-trace-recovery-stackchan-volume-1780424386012`, and the accepted relay WAV delivered on trace `a21-trace-recovery-stackchan-relay-wav-1780424386012` with `audio_chunks=40`. Full PRD accepted remains false because custom wake and broader dialogue half-duplex evidence are still pending. | `S4-PRD-PHYSICAL-ACCEPTED` |
| Xiaozhi audio/protocol | `S3D-LIVE-SAY-HOTFIX-RUNNING` | Plan `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`; server/downlink hello restored to stock-compatible 24 kHz while client/uplink remains 16 kHz; stock physical MAC devices no longer receive unsupported server `type=listen` replies; `/v1/xiaozhi/say` delivers TTS lifecycle and Opus binary to the live stock socket for text and now accepts a local 16 kHz mono WAV through `wav_path` with basename-only response metadata; focused Gateway/transport/app tests pass | `S4-PHYSICAL-STOCK-AUDIO-RETESTED` |
| TTS/audio quality | `S7A-RELAY-VOICE-PHYSICAL-OPERATOR-POSITIVE` | Recording `9.m4a` from 3 seconds stayed weak at `-36.99 LUFS` and `-16.05 dBTP`; bounded Gateway PCM leveling at 4x produced `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` at `-21.65 LUFS` and `-4.72 dBTP`, but the operator reported subtle electrical interruption and reduced clarity. Follow-up recording `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured `-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds. Current code uses bounded 3x gain, noise gate, and headroom; live 3x trace `a21-trace-stackchan-say-1780417217` delivered `579` chunks. Operator accepted final recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a`, measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds. Iflytek TTS is now coded as the immediate clearer TTS source; `voice_clone_cli` remains the voice-clone/persona seam; old `sherpa_onnx` is emergency/diagnostic only. Live Mac direct Iflytek smoke extracted the 5080 report credentials in memory without printing them, then failed at WebSocket dial; redacted report `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json` has `status=failed`, `network_mode=direct`, finding `iflytek_tts_websocket_dial_failed`. 5080 relay host-chain WAV `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` was delivered physically through stock `/v1/xiaozhi/say`; operator reported it was much better. Next improvement is selected-provider/readiness evidence and longer normal dialogue acceptance, not additional Gateway gain. | `S7-IFLYTEK-TTS-PHYSICAL-ACCEPTED` |
| Local clone TTS / CosyVoice | `S1-5080-SOURCE-PRESENT-WEIGHTS-MISSING` | Worker `019e897d-d4b4-78c3-9358-ac27a4f61d0d` confirmed 5080 is online and found CosyVoice source plus venv, IndexTTS2 source/venv/runner with CUDA, and traces of F5-TTS/GPT-SoVITS. No clone WAV was produced: CosyVoice venv lacks `torch`/`tqdm` and has no usable `pretrained_models/CosyVoice-*` weights; IndexTTS2 is closest but fails because `checkpoints/qwen0.6bemo4-merge/` is missing or not loadable; F5-TTS/GPT-SoVITS have no confirmed ready checkpoint/run path. Plan `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md` now tracks restoration. | `S2-LOCAL-CLONE-SMOKE-WAV-PRODUCED` |
| StackChan volume/action control | `S5-RUNTIME-VOLUME100-PHYSICAL-ACCEPTED` | `POST /v1/xiaozhi/speaker-volume` delivered official MCP `self.audio_speaker.set_volume` with `volume=100` to live device `44:1b:f6:e2:6a:60`; latest 3x trace is `a21-trace-stackchan-volume-1780417211`. Desktop helper supports both `volume` and `say`; physical loudness is accepted through final operator recording `10.m4a`. | `S6-FROZEN-FOR-CONTEST-FLOW` |
| Half-duplex / echo control | `S3A-NO-FLASH-NORMAL-DIALOGUE-NO-SELF-TRIGGER-CANDIDATE` | `/v1/xiaozhi/say` now arms a short input-suppression window for stock physical devices after host-say completion; focused test proves immediate listen restart and Opus echo are ignored without starting a new voice pipeline. Physical 3x trace `a21-trace-stackchan-say-1780417217` recorded `xiaozhi.say.input_suppression_armed=1` and `xiaozhi.listen.start.input_suppressed=1`. Diagnostic counter reports `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json` and `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` are blocked because current stock firmware lacks A21 identity, diagnostic mic-probe capability, available speaker echo fields, and runtime echo counters. That diagnostic blocker no longer blocks the contest path: no-flash normal-dialogue observation report `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json` delivered the accepted StepFun+Iflytek WAV through stock `/v1/xiaozhi/say` on trace `a21-trace-no-flash-dialogue-observe-1780423245`, waited `20000 ms`, observed `869` trace events, recorded `xiaozhi.listen.start.input_suppressed=1`, and found no `xiaozhi.listen.start`, `provider.start_turn.start`, `provider.realtime_session.start`, or `xiaozhi.voice_pipeline.start` self-trigger events. | `S4-TOUCH-BARGE-IN-OPERATOR-CHECK-OR-DIAGNOSTIC-COUNTERS` |
| Wake word | `S4G-ZI-YUE-PHRASE-TUNED-FLASHED-AWAITING-PROOF` | Custom MultiNet package `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json` and bare flash report `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json` remain incident/background evidence only because `xiaozhi.bin` regressed the product UI. `T-WAKE-002` integrated the requested wake word into the product app lane and flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json` passed, but operator physical verification failed: saying `紫悦` produced no response. `T-WAKE-003` keeps display `紫悦` and updates the custom MultiNet command string to `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue` while staying in the official-compatible product lane. Build report `reports/a21-stackchan-official-baseline-20260603-032615-1780428375746120000.json` passed; flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032734-1780428454747199000.json` passed; device reconnected online. | `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED` |

## Active Transitions

### Active T-WAKE-003: Zi Yue Phrase Tuning

Current state:

- `S4G-ZI-YUE-PHRASE-TUNED-FLASHED-AWAITING-PROOF`

Target state:

- `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED`

Trigger:

- Operator reported that saying `紫悦` produced no response after the guarded
  `T-WAKE-002` product flash.
- Current `zi yue` command is only two syllables and is likely too short for
  robust ESP-SR/MultiNet custom command recognition.

Actions:

- Use
  `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Keep product lane `a21-stackchan-official-xiaozhi-compatible.bin`.
- Keep visible wake identity `紫悦`.
- Tune only the custom MultiNet command list to
  `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue`.

Current result:

- Commit `5242349` passed focused tests and `make verify`.
- Official-compatible build passed with app SHA-256
  `e13a6cbd63596cd1388f5a540b488b3e35b449e49a5e31891cf133436561dc6e`.
- Guarded official-compatible flash execute passed on `/dev/cu.usbmodem1101`.
- Device `44:1b:f6:e2:6a:60` reconnected online.
- Physical wake proof is still pending operator retry.

Acceptance conditions:

- Focused overlay test passes.
- Official-compatible build config proves the tuned command string.
- Guarded official-compatible flash execute passes after clean-worktree checks.
- Operator physically confirms one of the `紫悦` variants wakes the device
  without screen touch.

Failure state:

- `F-WAKE-003-PHYSICAL-WAKE-REJECTED` if the tuned phrases still do not wake.
- `F-WAKE-003-FALSE-WAKE` if the tuned phrases cause unacceptable false wakes.

Rollback path:

- Keep the same product lane and tune threshold/phrases only.
- Reflash the last accepted official-compatible app if UI or audio regresses.

Next state:

- `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED`

### Active T-ASR-GREEN-LATENCY-001: Xiaozhi Listen Auto-Stop

Current state:

- `S-GREEN-ASR-LISTEN-HOTFIX-RUNNING-AWAITING-PHYSICAL-RETEST`

Target state:

- `S-GREEN-ASR-LISTEN-BOUNDED`

Trigger:

- Operator reported a long wait after ASR turns green.
- Live trace `a21-trace-44-1b-f6-e2-6a-60` showed multiple long reused-trace
  listen windows, including approximately 25s, 38s, 70s, and 108s before
  auto-stop or another listen replaced the window.

Actions:

- Use
  `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Add a Gateway max-listen safety stop after speech has been detected.
- Default max listen is 7000 ms and can be overridden with
  `A21_XIAOZHI_LISTEN_MAX_MS`.
- Improve trace summary pairing for reused physical trace ids.

Current result:

- Commit `5242349` passed focused tests and `make verify`.
- Gateway `a21-gateway-21081` was restarted with
  `A21_XIAOZHI_LISTEN_MAX_MS=7000`.
- Device `44:1b:f6:e2:6a:60` reconnected online to the restarted Gateway.
- Runtime volume `100` was redelivered through stock MCP on trace
  `a21-trace-wake-asr-hotfix-volume-1780428544`.
- Operator retest is still required.

Acceptance conditions:

- Focused Gateway test proves `xiaozhi.listen.max_duration_auto_stop`,
  `xiaozhi.listen.auto_stop`, and `xiaozhi.voice_pipeline.start`.
- Focused app test proves env wiring.
- Operator reports the green-light wait is materially shorter after restart or
  flash/runtime refresh.

Failure state:

- `F-ASR-GREEN-LATENCY-STILL-LONG` if green waits still exceed the configured
  bound.
- `F-ASR-GREEN-LATENCY-CUTS-SPEECH` if the bound cuts acceptable normal
  utterances too aggressively.

Rollback path:

- Increase `A21_XIAOZHI_LISTEN_MAX_MS` or revert only this Gateway change.

Next state:

- `S-GREEN-ASR-LISTEN-BOUNDED`

### Active T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001: Official Frontend Without Setup Trap

Current state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-CANDIDATE`

Target state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-READY`

Trigger:

- The operator confirmed the request/start variant trapped the physical device
  on "Welcome! Let's get started", but also asked to preserve official app
  loading so the full StackChan hardware experience is not lost.
- The operator also clarified wake cannot be verified before socket connection:
  before the Xiaozhi socket is established, saying the wake word does nothing;
  tapping the screen opens the socket by entering an unbounded green listening
  state.

Actions:

- Use
  `docs/plans/2026-06-03-boot-idle-socket-and-not-connected-ux.md`.
- Treat `T-BOOT-IDLE-SOCKET-001` as the sub-transition for quiet socket
  preconnect and not-connected feedback.
- Patch the official-compatible product overlay to install official StackChan
  apps first, then set codec volume and request Xiaozhi immediately without
  showing the welcome/setup trap.
- Add A21-named idle control-channel preconnect and `紫悦` connecting/ready
  feedback.
- Add device-side no-speech timeout so a touch-started listen without speech
  exits instead of staying green indefinitely.

Current result:

- Focused overlay tests pass for app-preload autostart, tuned `紫悦` custom
  wake, A21 quiet idle socket contract, and product-candidate reporting.
- The overlay patch passes ordinary `git apply --check` against the official
  source tree and `git diff --check`.
- `make verify` passed after the build-tool change that applies overlays after
  `fetch_repos.py`.
- Official-compatible build passed:
  `reports/a21-stackchan-official-baseline-20260603-040613-1780430773893634000.json`.
- Product app SHA-256:
  `7ff81bb0e564e020128e02068bc7d83b36f90c21de8cbd3d4b89b1dd1d6e9cf3`.
- Generated `sdkconfig.json` proves
  `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`,
  `USE_CUSTOM_WAKE_WORD=true`,
  `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`,
  `CUSTOM_WAKE_WORD_THRESHOLD=20`,
  `SR_MN_CN_MULTINET7_QUANT=true`,
  `USE_AFE_WAKE_WORD=false`,
  `SR_WN_WN9_HISTACKCHAN_TTS3=false`, and
  `SEND_WAKE_WORD_DATA=false`.
- Flash and physical operator validation are still pending.
- The `requestXiaozhiStart()` variant was physically rejected because it showed
  the welcome/setup page; the current hotfix starts Xiaozhi directly after app
  install and before any Mooncake update/setup frame.
- Commit `9ba8bc1` was guarded-flashed successfully through the
  official-compatible product lane:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-041636-1780431396539566000.json`.
- Device `44:1b:f6:e2:6a:60` reconnected `online` after flash, and runtime
  speaker volume `100` was delivered on trace
  `a21-trace-direct-preload-hotfix-volume-1780431400`.
- Operator later reported a welcome/setup regression again after the
  multi-wake flash. Root cause is now corrected in the overlay: official
  `Hal::startXiaozhi()` starts Xiaozhi and returns, so the previous direct
  autostart could still fall through into the Mooncake main loop and let
  `AppLauncher` create `StartupWorker`. The current hotfix parks `app_main`
  after direct `GetHAL().startXiaozhi()` by feeding the watchdog and sleeping,
  preventing AppLauncher/AppSetup from rendering `Welcome! Let's get started`.
- Focused app test now guards this control-flow contract by requiring
  `GetHAL().feedTheDog()` and `GetHAL().delay(1000)` before the Mooncake main
  loop anchor.
- The latest guarded product flash report is
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json`;
  T7 guard recorded clean commit `790697518c3e`, product app
  `a21-stackchan-official-xiaozhi-compatible.bin`, and app SHA-256
  `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`.
- The operator later confirmed setup is fixed. Current live Gateway evidence
  from `/v1/devices` on `127.0.0.1:21081` shows physical device
  `44:1b:f6:e2:6a:60` is `online`, `last_event=xiaozhi.hello`, and
  `speaker_volume=100`.
- Recovery trace analysis for `a21-trace-44-1b-f6-e2-6a-60` shows the old
  touch-start episode had only one `xiaozhi.listen.start`, one
  `xiaozhi.listen.stop`, one `xiaozhi.turn.start`, and no ongoing loop. The
  no-speech cooldown fired once with
  `xiaozhi.no_speech.input_suppression_armed=1`,
  `xiaozhi.listen.start.input_suppressed=1`, and
  `xiaozhi.listen.start.suppressed_after_no_speech=1`; the latest event is
  `xiaozhi.hello.received`.
- Runtime speaker volume `100` was re-delivered through stock MCP on trace
  `a21-trace-recovery-volume-1780441733`.
- Physical wake with `紫悦` variants and touch/no-speech retry remain pending.

Acceptance conditions:

- `make verify` passes.
- Official-compatible build config proves
  `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`.
- Product flash uses only
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- On boot, the device opens the stock Xiaozhi socket without requiring touch
  and without sending `listen.start`.
- The screen shows connecting/ready feedback while the socket is not ready.
- Touch without speech returns from green listening within the device timeout.

Failure state:

- `F-APP-PRELOAD-WELCOME-TRAP` if the welcome/setup screen appears again.
- `F-IDLE-SOCKET-STARTS-LISTENING` if boot preconnect sends `listen.start`.
- `F-TOUCH-GREEN-STILL-INFINITE` if touch still strands the device in green.

Rollback path:

- Revert only this overlay/test/doc transition and reflash the last known
  official-compatible product app if physical boot, audio, or wake behavior
  regresses.

Next state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-READY`

### Active T-AUDIO-BARE-XIAOZHI-PARITY-001: Migrate Bare-Package Audio Advantages

Current state:

- `S-BARE-XIAOZHI-PARITY-FACT-RECORDED`

Target state:

- `S-COMPATIBLE-PRODUCT-AUDIO-PARITY-CHECKLIST-ACTIONABLE`

Trigger:

- The operator-confirmed fact is that the bare `xiaozhi.bin` sounded louder and
  clearer, but it is a non-product incident artifact because it regressed the
  StackChan avatar/product UI.

Actions:

- Keep product lane `a21-stackchan-official-xiaozhi-compatible.bin`; do not
  flash bare `xiaozhi.bin`.
- Split parity into volume/NVS/MCP proof, TTS source mastering, downlink
  PCM/Opus/pacer metrics, and official StackChan app/action initialization.
- Use an independent worker thread for read-only or docs-only parity checklist
  generation.

Current result:

- Worker dispatch is active for this transition.
- No new audio/gain/provider behavior has been claimed in the main thread.

Acceptance conditions:

- A parity checklist identifies which differences are already closed, which are
  host-configurable, and which require physical A/B.
- The checklist preserves the accepted 3x contest path and does not promote
  additional Gateway gain without evidence.

Failure state:

- `F-AUDIO-PARITY-BARE-FLASH-REGRESSION` if a worker or main thread attempts to
  use bare `xiaozhi.bin` as product firmware.
- `F-AUDIO-PARITY-UNVERIFIED-TUNING` if a change claims louder/clearer physical
  output without operator recording or trace evidence.

Rollback path:

- Keep current accepted 3x audio path and runtime volume `100`; defer parity
  tuning until a bounded A/B plan exists.

Next state:

- `S-COMPATIBLE-PRODUCT-AUDIO-PARITY-CHECKLIST-ACTIONABLE`

### Active T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001: Full Xiaozhi Realtime Voice Convergence

Current state:

- `S-XIAOZHI-REALTIME-SEAMS-MAINLINE-NOT-OVERLAPPED`

Trigger:

- The user's review correctly defines Xiaozhi voice as a realtime media system:
  local wake/VAD, long Xiaozhi transport, small Opus frames, streaming ASR,
  streaming LLM, streaming TTS, and paced Opus playback.
- A21 has multiple correct-looking seams, but seams/static gates are not the
  same as real runtime or physical acceptance.

Target state:

- `S-XIAOZHI-FULL-REALTIME-PHYSICAL-CANDIDATE`

Action:

- Use plan
  `docs/plans/2026-06-03-xiaozhi-full-realtime-voice-convergence.md`.
- Keep the product firmware lane on
  `a21-stackchan-official-xiaozhi-compatible`.
- Read-only worker audits for official Xiaozhi protocol/state, CoreS3/audio
  HAL/wake behavior, A21 runtime gaps, and architecture strategy have completed
  again on HEAD `188b341`. They agree that A21 should continue incremental
  convergence on stock Xiaozhi protocol/framework patterns, not wholesale
  server-stack embedding, while keeping full realtime acceptance red.
- Sequence the next implementation transitions as:
  the now-landed host ASR partial-to-LLM bridge, real Sherpa streaming ASR
  unblock, real streaming TTS runtime execution if authorized, physical stock
  `/v1/xiaozhi` realtime parity, local wake/state closure, and official
  StackChan audio/HAL behavior parity.
- Preserve strict evidence boundaries: host/static/mock/provider-shape evidence
  cannot claim physical PRD acceptance.

Acceptance conditions:

- Plan exists and is committed.
- Worker fan-out ids and boundaries are recorded in the plan/handoff.
- `docs/project_state_machine.md` names this convergence transition so future
  models do not treat individual ASR/TTS seams as final success.
- The next code execution worker receives one scoped transition with explicit
  no-flash/no-provider/no-hardware boundaries unless separately authorized.

Failure states:

- `F-XIAOZHI-CONVERGENCE-FAKE-GREEN` if mock, `/say`, host-loopback, static
  readiness, or WAV/file paths are described as Xiaozhi realtime acceptance.
- `F-XIAOZHI-CONVERGENCE-SCOPE-SPRAWL` if one worker silently changes firmware,
  provider execution, wake, gain, and Gateway behavior together.
- `F-XIAOZHI-CONVERGENCE-STOCK-PROFILE-LEAK` if debug/device extensions become
  required in the stock Xiaozhi hello/runtime.

Rollback path:

- Revert the convergence plan/state/log entries. Existing ASR/TTS seams and
  accepted contest audio path remain untouched.

Next state:

- `S-XIAOZHI-CONTINUOUS-TURN-BRIDGE-CANDIDATE-HOST-TESTED`
- Next transition: unblock real Sherpa streaming ASR runtime evidence, run real
  streaming TTS runtime only if explicitly authorized, then collect a physical
  stock `/v1/xiaozhi` parity trace without `/say`, host loopback, mock, or
  WAV/file evidence.

### Active T-XIAOZHI-REALTIME-VOICE-PARITY-001: Xiaozhi Realtime Voice Parity Gate

Current state:

- `S-XIAOZHI-STOCK-OPUS-TRANSPORT-TURN-BUFFERED-HOST`

Trigger:

- User review requires A21 to match Xiaozhi's realtime voice design rather than
  behaving like a normal full-WAV/full-HTTP voice bot.
- Three read-only worker audits on 2026-06-03 agreed that the product device
  lane is stock-shaped at the `/v1/xiaozhi` Opus/WebSocket boundary, but the
  host voice pipeline is still turn-buffered at the ASR boundary.

Target state:

- `S-XIAOZHI-REALTIME-PARITY-GATE-LANDED`

Action:

- Keep product firmware on `a21-stackchan-official-xiaozhi-compatible.bin`.
- Add a trace-only `xiaozhi-realtime-parity` evidence command that reads
  Gateway `/v1/devices` and `/v1/traces` without driving `/say`, synthetic
  host-loopback WebSocket audio, provider execution, V21 execution, flash, NVS,
  or audio playback.
- Classify live traces as `blocked`, `stock_opus_transport_only`,
  `turn_buffered_xiaozhi_candidate`, or `xiaozhi_realtime_candidate`.
- Reject fake path markers such as `/v1/xiaozhi/say`,
  `fast_companion.voice_pipeline.*`, and local fallback control events.

Acceptance conditions:

- Focused tests pass for physical stock Opus traces, streaming-ordering traces,
  and fake `/say` traces.
- The report redacts payloads, credentials, full URLs, local paths,
  transcripts, and raw audio.
- The report never sets `prd_accepted=true` and does not claim local wake,
  audible playback, interruption, or setup-free product acceptance.

Failure states:

- `F-XIAOZHI-REALTIME-PARITY-FAKE-GREEN` if host-loopback, `/say`, or
  fast-companion traces can pass as realtime parity.
- `F-XIAOZHI-REALTIME-PARITY-UNSAFE-REPORT` if report output stores secrets,
  raw audio, transcripts, full URLs, or local paths.
- `F-XIAOZHI-REALTIME-PARITY-BEHAVIOR-REGRESSION` if adding the evidence gate
  changes Gateway or firmware runtime behavior.

Rollback path:

- Remove the additive CLI/report/tests and revert this state/log entry. No
  firmware, provider, Gateway runtime, or NVS rollback is required for Phase 1.

Next state:

- `S-XIAOZHI-REALTIME-PARITY-GATE-LANDED`
- Follow-up transition if the live report remains turn-buffered:
  `T-XIAOZHI-STREAMING-ASR-001`.

### Completed T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001: Realtime Parity Real Profile Evidence

Current state:

- `S-XIAOZHI-REALTIME-PARITY-ORDERING-GATE-HARDENED`

Trigger:

- The parity gate could classify a trace as `xiaozhi_realtime_candidate` based
  on stock Opus transport plus ordered `asr.first_partial`,
  `provider.first_content`, `tts.first_audio`, and downlink markers, even when
  the trace did not prove real streaming ASR/LLM/TTS profile classes.
- User review explicitly warns that A21 must not hide a normal buffered
  question/answer path behind Xiaozhi-shaped protocol markers.

Target state:

- `S-XIAOZHI-REALTIME-PARITY-REQUIRES-REAL-PROFILE-EVIDENCE`

Action:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md`.
- Gateway voice-pipeline turns now emit redacted category markers for profile
  classes:
  `xiaozhi.voice_pipeline.asr.real_streaming`,
  `xiaozhi.voice_pipeline.llm.real_streaming`,
  `xiaozhi.voice_pipeline.tts.real_streaming`, or blocker markers such as
  `asr.mock_blocked`, `asr.batch_blocked`, `llm.mock_blocked`,
  `tts.mock_blocked`, and `tts.file_boundary_blocked`.
- `xiaozhi-realtime-parity` now counts those markers and requires all three
  real streaming stage markers, with no profile blockers, before it can return
  `xiaozhi_realtime_candidate`.

Acceptance conditions:

- Red app test first proved an ordered realtime-looking trace without real
  profile markers was incorrectly accepted as `xiaozhi_realtime_candidate`.
- Red Gateway test first proved mock partial-bridge turns lacked blocker
  markers.
- Focused app/Gateway tests pass and reports stay redacted.

Failure states:

- `F-REALTIME-PARITY-MOCK-GREEN` if mock or file-boundary profile classes can
  still reach `xiaozhi_realtime_candidate`.
- `F-TRACE-SECRET-LEAK` if profile evidence stores provider values, URLs,
  paths, transcripts, credentials, proxy values, or raw audio.

Rollback path:

- Revert the profile marker additions, parity-gate checks, tests, plan, and
  state/log entries. Existing stock `/v1/xiaozhi` transport, ASR partial
  bridge, Sherpa smoke, and TTS adapter seam remain intact.

Next state:

- `S-XIAOZHI-REALTIME-PARITY-REQUIRES-REAL-PROFILE-EVIDENCE`
- Next transition: non-blocking turn reducer/queue hardening so streaming ASR
  commit/final handling cannot stall the Xiaozhi WebSocket read loop.

### Completed T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001: Xiaozhi Nonblocking ASR Commit

Current state:

- `S-XIAOZHI-ASR-COMMIT-NONBLOCKING-HOST-TESTED`

Trigger:

- Runtime read-only audit found `commitXiaozhiStreamingASR` was called
  synchronously from `listen.stop` and VAD auto-stop, which could hold the
  `/v1/xiaozhi` WebSocket read loop while commit/final polling waited.
- Xiaozhi-style voice requires the control channel to remain responsive to
  abort/barge-in while ASR finalization is pending.

Target state:

- `S-XIAOZHI-ASR-COMMIT-ASYNC-CONTROL-LOOP-RESPONSIVE`

Action:

- Added plan `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md`.
- Replaced synchronous stop/auto-stop ASR commit with an async commit task.
- The async task records `asr.stream.commit`, waits briefly for a streaming
  ASR final, starts the voice-pipeline task from the streaming final when no
  partial-driven answer already started, and records truthful timeout/error
  markers instead of silently promoting missing final evidence.
- Removed the unused synchronous commit helper to avoid future regression.

Acceptance conditions:

- Red test first proved abort was not processed while streaming ASR commit was
  pending.
- New Gateway tests prove abort is recorded during pending commit and that a
  streaming ASR final starts the pipeline without calling batch `Transcribe`.
- Existing Xiaozhi streaming ASR, partial bridge, voice-pipeline, and parity
  tests continue to pass.

Failure states:

- `F-XIAOZHI-ASR-COMMIT-BLOCKS-WS` if abort/barge-in cannot be read during ASR
  commit.
- `F-XIAOZHI-ASR-COMMIT-BATCH-REGRESSION` if a streaming ASR final path calls
  batch `Transcribe` or writes a WAV.
- `F-XIAOZHI-ASR-COMMIT-DOUBLE-TURN` if partial-driven and final-driven paths
  start duplicate voice-pipeline tasks.

Rollback path:

- Revert the async commit helper, stop/auto-stop call-site changes, tests, plan,
  and state/log entries. Existing stock `/v1/xiaozhi` transport, ASR partial
  bridge, real-profile gate, Sherpa smoke, and TTS adapter seam remain intact.

Next state:

- `S-XIAOZHI-ASR-COMMIT-ASYNC-CONTROL-LOOP-RESPONSIVE`
- Next transition: feed the real Sherpa streaming ASR session through a stock
  `/v1/xiaozhi` host-local trace, then keep realtime parity blocked until real
  streaming LLM/TTS profile evidence and physical playback are present.

### Completed T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001: Stock STT And Wake Preroll

Current state:

- `S-XIAOZHI-STOCK-STT-WAKE-PREROLL-HOST-TESTED`

Trigger:

- Source-read workers compared upstream Xiaozhi protocol/audio behavior with
  A21 and found two narrow host-side fidelity gaps that did not require live
  providers, Gateway lifecycle changes, firmware, hardware, or audio playback.
- Stock Xiaozhi expects server `stt` display messages before TTS when
  transcripts exist, and wake-adjacent Opus can arrive before/around
  `listen/detect`.

Target state:

- `S-XIAOZHI-STOCK-STT-AND-IDLE-PREROLL-PRESERVED`

Action:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md`.
- Emitted stock `stt` WebSocket messages before `tts/start` when a streaming
  ASR partial or final transcript is available, without recording transcript
  text in traces.
- Added bounded idle wake pre-roll buffering for decoded Opus frames received
  before `listen/start`, attaching those frames to the next turn and feeding
  them into streaming ASR when active.
- Kept cooldown and current-turn suppression intact so post-placeholder or
  host-say Opus remains `ignored_not_listening` instead of becoming false wake
  evidence.

Acceptance conditions:

- Red tests first proved TTS began before stock `stt` and pre-listen Opus was
  dropped from the next voice pipeline request.
- Focused Gateway tests prove stock `stt` precedes TTS, wake pre-roll feeds the
  next turn, streaming ASR/final/partial paths still work, and paced Opus TTS
  downlink still works.
- Full Gateway package and focused app realtime parity/readiness tests pass.

Failure states:

- `F-XIAOZHI-STT-ORDER-REGRESSION` if TTS starts before stock `stt` when a
  transcript exists.
- `F-XIAOZHI-WAKE-PREROLL-DROPPED` if idle pre-listen Opus cannot feed the next
  streaming ASR/voice-pipeline turn.
- `F-XIAOZHI-COOLDOWN-FALSE-WAKE` if cooldown or current-turn Opus is
  misclassified as wake pre-roll.

Rollback path:

- Revert the stock STT helper, wake pre-roll buffering/attach logic, tests,
  plan, and state/log entries. Existing nonblocking ASR commit, partial bridge,
  real-profile gate, Sherpa smoke, and realtime TTS seam remain intact.

Next state:

- `S-XIAOZHI-STOCK-STT-AND-IDLE-PREROLL-PRESERVED`
- Next transition: with authorization, collect a physical stock `/v1/xiaozhi`
  trace or execute the real streaming TTS provider path; keep realtime PRD
  acceptance false until real streaming ASR/LLM/TTS profile evidence, physical
  playback, barge-in/touch, and idle recovery are proven.

### Completed T-XIAOZHI-OPUS-INGRESS-QUEUE-001: Xiaozhi Opus Ingress Queue

Current state:

- `S-XIAOZHI-OPUS-INGRESS-QUEUED-HOST-TESTED`

Trigger:

- The full realtime objective explicitly requires an Opus frame queue between
  local wake/VAD capture and streaming ASR.
- Source-read workers confirmed upstream Xiaozhi separates protocol callbacks
  from encode/decode/playback queues, while A21 still decoded Opus, ran
  audio-ingress/VAD, and appended to streaming ASR inline on the WebSocket read
  loop.

Target state:

- `S-XIAOZHI-OPUS-FRAME-QUEUE-CONTROL-LOOP-RESPONSIVE`

Action:

- Added plan `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md`.
- Added a bounded per-session Opus ingress queue for listening frames.
- `handleXiaozhiBinary` now parses/adopts incoming Opus, records receipt,
  enqueues the frame, and returns to the WebSocket read loop without waiting
  for decode, VAD/audio ingress, or streaming ASR append.
- The queue worker preserves frame order, decodes Opus, pushes audio ingress
  evidence, appends to streaming ASR, and records queue drops explicitly.
- `listen.stop` now starts an async finalize path that waits briefly for
  queued ingress to catch up before committing streaming ASR or starting the
  voice pipeline, keeping the control loop free to process abort.

Acceptance conditions:

- Red test first proved a blocking streaming ASR `AppendFrame` prevented
  `abort` from being processed.
- Focused Gateway tests prove `abort` is recorded while ASR append remains
  blocked and that wake pre-roll, stock STT, streaming final, nonblocking
  commit, partial bridge, and paced Opus downlink still work.
- Full Gateway package, focused app parity/readiness tests, `git diff --check`,
  and `make verify` pass.

Failure states:

- `F-XIAOZHI-OPUS-INGRESS-BLOCKS-CONTROL` if audio decode/VAD/ASR append can
  block reading `abort`.
- `F-XIAOZHI-OPUS-INGRESS-EMPTY-STOP` if `listen.stop` starts a voice pipeline
  before queued frames are processed.
- `F-XIAOZHI-OPUS-INGRESS-HIDDEN-DROP` if queue overflow loses frames without
  trace evidence.

Rollback path:

- Revert the queue helper code, async stop-finalize path, tests, plan, and
  state/log entries. Existing stock STT, wake pre-roll, nonblocking ASR commit,
  partial bridge, real-profile gate, Sherpa smoke, and realtime TTS seam remain
  intact.

Next state:

- `S-XIAOZHI-OPUS-FRAME-QUEUE-CONTROL-LOOP-RESPONSIVE`
- Next transition: prove wake-as-abort/playback-drain ordering in a host stock
  WebSocket test, then move to authorized physical stock `/v1/xiaozhi` trace or
  real streaming TTS runtime proof. Full PRD acceptance remains false until
  real streaming ASR/LLM/TTS profile evidence, physical playback, wake/tap,
  barge-in/touch, and idle recovery are proven.

### Active T-XIAOZHI-STREAMING-ASR-001: Stock Xiaozhi Streaming ASR Session

Current state:

- `S-XIAOZHI-STREAMING-ASR-SESSION-SEAM-IMPLEMENTED-FAKE-ONLY`

Trigger:

- User review requires A21 to match Xiaozhi's realtime chain:
  local wake/VAD, long socket, Opus frames, streaming ASR, streaming LLM,
  streaming TTS, and paced Opus playback.
- Read-only workers confirmed current `/v1/xiaozhi` was previously
  turn-buffered at the ASR boundary: decoded PCM frames were accumulated, then
  ASR ran after stop/VAD end.

Target state:

- `S-XIAOZHI-STREAMING-ASR-PROVIDER-READY`

Action:

- Use `docs/plans/2026-06-03-xiaozhi-streaming-asr.md`.
- Add optional `providers.StreamingASRAdapter` and `StreamingASRSession`
  alongside existing batch `ASRAdapter`.
- Start a streaming ASR session on stock `/v1/xiaozhi` listen start when the
  selected ASR adapter supports it.
- Feed decoded Opus PCM frames into the streaming ASR session in
  `observeXiaozhiDecodedIngress`.
- Commit the streaming ASR session on `listen.stop` or VAD auto-stop.
- Reuse streaming ASR final text in the voice pipeline so batch ASR is not
  re-run when streaming final is available.
- Keep existing accumulated `voicePipelineFrames` and batch ASR fallback when no
  streaming session/final exists.

Acceptance conditions:

- Gateway test proves `asr.first_partial` appears before `xiaozhi.listen.stop`
  and before `xiaozhi.voice_pipeline.start`.
- Existing listen-stop and VAD auto-stop tests still pass.
- Provider test proves mock streaming ASR emits partial on frame append and
  final only on commit.
- `xiaozhi-realtime-parity` counts `asr.stream.start`,
  `asr.audio.append`, and `asr.stream.commit`, and requires stream start/append
  for `xiaozhi_realtime_candidate`.
- This phase is not real provider acceptance; it is a session seam and fake
  adapter proof.

Failure states:

- `F-XIAOZHI-STREAMING-ASR-BATCH-REGRESSION` if non-streaming ASR fallback no
  longer works.
- `F-XIAOZHI-STREAMING-ASR-FAKE-PROVIDER-GREEN` if fake streaming ASR is
  recorded as real provider readiness.
- `F-XIAOZHI-STREAMING-ASR-GOROUTINE-LEAK` if abort/socket close does not cancel
  sessions.
- `F-XIAOZHI-STREAMING-ASR-TRACE-ORDERING-REGRESSION` if partial/final markers
  occur only after listen stop.

Rollback path:

- Revert the additive provider interfaces, Gateway session lifecycle hooks,
  tests, and this state entry. No firmware, NVS, provider credential, or audio
  gain rollback is involved.

Next state:

- `S-XIAOZHI-STREAMING-ASR-PROVIDER-READY`
- Next transition:
  `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`.

### Active T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001: Sherpa Streaming ASR Runtime Helper

Current state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`

Trigger:

- The full Xiaozhi parity objective requires real streaming ASR, not a batch
  WAV runner or a selectable profile name.
- `T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001` added profile selection and an
  injected factory seam, but no default runtime helper/session process.

Target state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`

Action:

- Use `docs/plans/2026-06-03-sherpa-streaming-asr-runtime-helper.md`.
- Add a JSONL subprocess helper script for Sherpa streaming ASR sessions.
- Add a Go subprocess-backed `StreamingASRSessionFactory` for
  `sherpa_onnx_streaming` when helper path and model dir are configured.
- Prove with fake helper tests that `AppendFrame` can produce partial ASR
  before `Commit`, and `Commit` can produce final ASR without writing WAV.
- Preserve batch `sherpa_onnx` and keep readiness/PRD acceptance red until real
  model/provider/physical evidence exists.

Acceptance conditions:

- Plan exists and is committed: `13e9e82`.
- Worker runs in a scoped worktree with explicit no-firmware/no-provider/no-audio
  boundaries.
- Focused provider/app tests, `git diff --check`, and `make verify` pass before
  integration.
- Handoff logs state that this is runtime-helper candidate evidence only, not
  full Xiaozhi realtime acceptance.
- Worker branch
  `codex/a21-sherpa-streaming-asr-runtime-manual-20260603` adds the JSONL
  helper and subprocess session; fake-helper tests prove `AppendFrame` produces
  partial and `Commit` produces final without calling the batch WAV runner.
- Mainline fast-forward integration commit: `041ad69`.
- Mainline test hardening commit: `987a532`.
- Mainline focused tests and `make verify` pass after increasing the
  subprocess-helper test event wait to tolerate full-repo package parallelism.

Failure states:

- `F-SHERPA-STREAMING-HELPER-WAV-REGRESSION` if the streaming path writes WAV.
- `F-SHERPA-STREAMING-HELPER-SECRET-LEAK` if helper errors expose local paths,
  transcripts, provider output, URLs, proxy values, or credentials.
- `F-SHERPA-STREAMING-HELPER-HANG` if subprocess cancellation leaves pipes or
  child processes alive.
- `F-SHERPA-STREAMING-ASR-FALSE-GREEN` if static helper env is recorded as real
  model/physical PRD acceptance.

Rollback path:

- Revert helper script, subprocess factory, env wiring, tests, and docs. The
  adapter seam and batch Sherpa path remain available.

Next state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`
- Next transition: real no-audio model smoke, then stock `/v1/xiaozhi`
  operator-triggered physical realtime parity.

### Completed T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001: Sherpa Streaming ASR Real-Model No-Audio Smoke

Current state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`
- Local outcome: `passed` with `real_model_no_audio_streaming_smoke`.
- Report:
  `reports/a21-local-asr-streaming-smoke-20260603-085636-1780448196060587000.json`
  stores helper/model basenames, env names, booleans, event counts, and
  redaction policies only.

Trigger:

- The Sherpa streaming helper is mainline candidate, but fake helper tests do
  not prove local real-model streaming ASR can start or commit.
- Xiaozhi realtime parity requires ASR runtime proof before a physical
  `/v1/xiaozhi` turn can be classified as more than a seam/static candidate.

Target state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`

Action:

- Use plan `docs/plans/2026-06-03-sherpa-realmodel-no-audio-smoke.md`.
- Add or reuse a redacted no-audio smoke report for
  `sherpa_onnx_streaming`.
- Run the JSONL helper against real local model files if present; otherwise
  record a stable model/package blocker.
- Do not write WAV, play audio, capture mic, call providers/V21, start/stop
  Gateway, flash firmware, or write NVS.
- Completed by adding `a21 local-asr-streaming-smoke` and a Make target.
  Later control integration found the repo-local canonical model cache and
  taught the smoke/provider runtime/readiness paths to discover it when explicit
  env is absent. `make local-asr-streaming-smoke` now passes locally with the
  real helper/model and redacted evidence.

Acceptance conditions:

- Smoke outcome is recorded as either real-model pass or truthful blocker.
- Report stores no transcript, raw audio, raw helper payload, credential, full
  URL, or absolute local path.
- Focused tests, `git diff --check`, and `make verify` pass before commit.
- State/log make clear this remains below physical PRD acceptance.
- Current smoke outcome is a host-local real-model pass, not PRD or physical
  realtime parity acceptance.

Failure states:

- `F-SHERPA-REALMODEL-SMOKE-WAV-BOUNDARY` if the smoke writes or reads WAV.
- `F-SHERPA-REALMODEL-SMOKE-SECRET-LEAK` if reports leak path/transcript/audio
  or secret values.
- `F-SHERPA-REALMODEL-SMOKE-FAKE-GREEN` if missing model files or missing
  sherpa-onnx package are treated as pass.

Rollback path:

- Revert the smoke CLI/report/tests/docs. Keep the streaming helper seam.

Next state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`
- Next transition: authorized real streaming TTS runtime execution or physical
  stock `/v1/xiaozhi` parity trace after the selected runtime env is complete.

### Completed T-STREAMING-TTS-RUNTIME-PROOF-001: Streaming TTS Runtime Proof

Current state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED-BLOCKED-NO-EXECUTE`

Target state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED`

Trigger:

- `T-XIAOZHI-STREAMING-TTS-ADAPTER-001` added a selectable
  `doubao_tts_realtime` streaming TTS adapter seam, but no real runtime smoke
  has proven or truthfully blocked provider session start, text append, first
  audio delta, 60 ms PCM chunk framing, and session close.
- Xiaozhi realtime parity still requires evidence that selected TTS can emit
  audio before a complete WAV/file or provider EOF exists.

Actions:

- Use plan `docs/plans/2026-06-03-streaming-tts-runtime-proof.md`.
- Add a scoped app smoke such as `a21 streaming-tts-runtime-smoke` plus Make
  target.
- Default to no-provider execution and report `execute_flag_required` unless
  `--execute` is explicit.
- With missing env, report stable blocker names instead of failing obscurely or
  echoing secret values.
- With explicit `--execute` and complete env, start only the selected realtime
  TTS session, send a short A21-owned test text, observe first provider audio
  delta, count valid 60 ms PCM chunks, close the session, and write redacted
  timing/count evidence.
- Worker result: added `a21 streaming-tts-runtime-smoke` and
  `make streaming-tts-runtime-smoke`. Default local runtime report
  `reports/a21-streaming-tts-runtime-smoke-20260603-074721-1780444041851710000.json`
  is `status=blocked` with finding `execute_flag_required`; no real provider
  call, Gateway call, ASR/LLM/V21 call, `/v1/xiaozhi/say`, physical device
  path, audio playback, firmware build, flash, or NVS write occurred.

Acceptance conditions:

- Focused app/provider tests prove no-execute blocker, missing-env blocker,
  fake realtime first-audio-before-EOF behavior, no WAV/file boundary, and
  redaction.
- Runtime report stores no user transcript, provider output text, raw/base64
  audio, credentials, full URL, proxy value, or absolute path.
- If real provider execution is not authorized or fails, the transition records
  a truthful blocked/failed status and remains below PRD acceptance.
- `git diff --check` and relevant focused tests pass before integration.

Failure states:

- `F-STREAMING-TTS-RUNTIME-SECRET-LEAK` if output/report leaks credentials,
  raw/base64 audio, full URL, proxy value, user text, or absolute path.
- `F-STREAMING-TTS-RUNTIME-WAV-BOUNDARY` if the smoke writes/reads WAV or waits
  for a complete file before first chunk.
- `F-STREAMING-TTS-RUNTIME-FAKE-GREEN` if static env/config or fake provider
  tests are promoted as real provider/runtime or physical Xiaozhi acceptance.

Next candidate transition:

- Operator-authorized real Doubao realtime TTS runtime execution with
  `--execute` and complete env, still below physical `/v1/xiaozhi` acceptance.

Rollback path:

- Revert the smoke command, Make target, tests, and docs. Keep the existing
  static adapter seam and accepted physical 3x contest audio path.

Next state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED`
- Next transition: stock `/v1/xiaozhi` realtime parity proof after ASR and TTS
  runtime evidence are both available.

### Active T-PROVIDER-002b: Iflytek/Real-TTS Live Chain Unblock

Current state:

- `S4A-SELECTED-PROVIDER-READY-VOICE-CHAIN-CANDIDATE`
- `S7A-RELAY-VOICE-PHYSICAL-OPERATOR-POSITIVE`

Target state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

Trigger:

- `T-PROVIDER-002` integrated host-side StepFun and Iflytek hot-plug code in
  commit `b5a405e`, but live Mac direct Iflytek WebSocket dial failed and
  StepFun loopback was unstable on the Mac direct path.
- The operator rejected the old local TTS source for contest dialogue quality.

Actions:

- Use `docs/plans/2026-06-03-provider-tts-live-chain-unblock.md`.
- Worker `019e8954-e65c-72a2-9847-a17a59a0ad6b` used 5080 relay first instead
  of adding a WebSocket adapter.
- Keep provider credentials host-side, reports redacted, global proxy
  unchanged, and firmware/NVS out of scope.

Current result:

- 5080 relay Iflytek TTS passed:
  `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json`,
  first audio `100.299 ms`.
- 5080 relay StepFun `step-1-8k` plus Iflytek TTS chain passed:
  `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json`,
  text first content `229.077 ms`, TTS first audio `87.947 ms`.
- Candidate WAVs were imported under `reports/provider-tts-candidate/`.
- Physical relay WAV playback was later completed through stock
  `/v1/xiaozhi/say` on trace
  `a21-trace-provider-playback-wav-1780450901`; operator feedback was
  positive: "好多了".
- Selected-provider readiness was refreshed without falling back to `mock`:
  `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json`
  selects route-eligible `deepseek` with `real_provider_ready=true`; bundle
  `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json`
  has `provider.ready=true`.
- StepFun remains an explicit compatibility voice-chain candidate, not product
  route-eligible provider readiness evidence.

Acceptance conditions:

- Physical StackChan playback uses the relay-generated StepFun+Iflytek WAV or
  equivalent live chain through the stock Xiaozhi path.
- Operator accepts or rejects the new voice with a concrete listening reason.
- StepFun remains an explicit compatibility candidate until a separate
  readiness promotion.
- Product selected-provider readiness remains pinned to a route-eligible real
  provider report unless an explicit ADR/transition promotes StepFun.

Failure state:

- `F-PROVIDER-002B-SECRETS-LEAKED` if credentials, auth URL/query, transcript,
  provider output, raw/base64 audio, proxy value, or local path enter reports
  or docs.
- `F-PROVIDER-002B-PHYSICAL-STALE` while no fresh physical socket is available
  for playback.
- `F-PROVIDER-002B-PHYSICAL-REJECTED` if operator rejects the relay voice after
  stock Xiaozhi playback.

Rollback path:

- Keep accepted 3x Gateway gain and stock firmware.
- Use `sherpa_onnx` only as emergency/diagnostic fallback, or `voice_clone_cli`
  when a configured clone wrapper is selected.

Next state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

### Active T-PROVIDER-PLAYBACK-001: Stock Xiaozhi Relay WAV Playback

Current state:

- `S-PROVIDER-PLAYBACK-STOCK-WAV-DELIVERED-OPERATOR-POSITIVE`

Target state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

Trigger:

- The 5080 relay produced a candidate StepFun+Iflytek WAV, but the legacy
  `/v1/devices/control` playback path is not valid evidence for stock Xiaozhi
  firmware.

Actions:

- Use
  `docs/plans/2026-06-03-stock-xiaozhi-relay-wav-playback.md`.
- Worker PLAYBACK added TDD-covered `/v1/xiaozhi/say` `wav_path` support for
  local A21-compatible 16 kHz mono PCM WAV files.
- Keep response/report surfaces basename-only and exclude raw/base64 audio,
  transcript/provider output, credentials, proxy values, and full local paths.

Current result:

- Focused Gateway test first failed RED with `400: text is required`.
- Minimal Gateway implementation now sends WAV chunks through the same stock
  TTS lifecycle, Opus downlink path, and post-say input-suppression path as
  text say.
- Gateway `21081` was restarted from this branch, the target device reconnected
  online/fresh, and runtime speaker volume `100` was delivered through stock
  MCP trace `a21-trace-relay-playback-volume-1780450901`.
- Foreground `/v1/xiaozhi/say` `wav_path` delivery succeeded for relay WAV
  basename `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` with
  trace `a21-trace-provider-playback-wav-1780450901`, `audio_chunks=40`,
  `delivered_transport=xiaozhi_ws`, `audio_source=wav_file`, and no full path
  in the response.
- Trace `a21-trace-provider-playback-wav-1780450901` recorded
  `xiaozhi.say.start=1`, `tts.first_audio=1`,
  `audio.downlink.first_frame=1`, `xiaozhi.tts.opus_frame.downlink=40`,
  `xiaozhi.say.input_suppression_armed=1`, and `xiaozhi.say.delivered=1`.
- Operator feedback after hearing it: "好多了".

Acceptance conditions:

- `/v1/devices` shows device `44:1b:f6:e2:6a:60` online/fresh.
- The relay WAV basename `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`
  is delivered through `/v1/xiaozhi/say` `wav_path`.
- Operator accepts or rejects the voice with a concrete listening reason.

Failure state:

- `F-PROVIDER-PLAYBACK-PHYSICAL-STALE` while no fresh physical socket is
  available for playback.
- `F-PROVIDER-PLAYBACK-LIVE-BINARY-OLD` if the live Gateway process is still
  running a pre-`wav_path` build and rejects the request before delivery.
- `F-PROVIDER-PLAYBACK-SECRET-LEAK` if a response, report, or doc captures
  full WAV paths, raw/base64 audio, transcript/provider output, credentials,
  proxy values, or other local path values beyond safe basenames.

Rollback path:

- Revert the `wav_path` handler/test/doc changes only; keep accepted 3x gain,
  stock firmware, provider hot-plug code, wake package, and half-duplex state.

Next state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

### Active T-HALF-DUPLEX-001: Normal Dialogue Echo Suppression Acceptance

Current state:

- `S2C-NORMAL-DIALOGUE-ACCEPTANCE-IN-PROGRESS`

Target state:

- `S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

Trigger:

- Host-say suppression is physically observed, but normal dialogue after a real
  user turn has not proven no self-trigger after assistant TTS.

Actions:

- Use `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`.
- Worker `019e8956-7c31-78e3-bd96-f04b94f52d0b` owns safe readiness and probe
  work only.

Current result:

- Gateway `127.0.0.1:21081` is healthy.
- Device `44:1b:f6:e2:6a:60` is registered but stale, with
  `identity_status=unknown`, empty firmware identity, microphone
  `available_xiaozhi_opus_ingress`, and speaker
  `available_xiaozhi_opus_downlink`.
- `stackchan-half-duplex-acceptance` was not run because the device was not
  online/fresh and does not expose the diagnostic mic-probe capability required
  by the plan.

Acceptance conditions:

- Device is online/fresh and exposes the needed speaker/microphone evidence.
- `stackchan-half-duplex-acceptance` or normal-dialogue trace proves no
  unwanted self-trigger after TTS while preserving touch/barge-in.

Failure state:

- `F-HALF-DUPLEX-001-DEVICE-STALE` if device cannot be refreshed for live
  acceptance.
- `F-HALF-DUPLEX-001-NO-DIAGNOSTIC-MIC` if current firmware lacks the required
  mic-probe counters for instrumented acceptance.
- `F-HALF-DUPLEX-001-OVER-SUPPRESSED` if a fix makes touch/barge-in feel deaf.

Rollback path:

- Revert only half-duplex suppression tuning if it blocks user interruption.
- Keep accepted 3x gain, stock protocol, provider selection, wake firmware, and
  NVS unchanged.

Next state:

- `S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

### Rejected T-WAKE-002: Zi Yue Wake In StackChan-Compatible App

Current state:

- `S4F-ZI-YUE-PHYSICAL-WAKE-REJECTED`

Target state:

- `S4-ZI-YUE-PHYSICAL-WAKE-PROOF`

Trigger:

- Custom wake package existed only in the bare `xiaozhi.bin` lane, but that lane
  regressed the physical product UI to plain Xiaozhi.
- The live physical firmware still needs custom wake that preserves the
  StackChan avatar/action surface.
- The operator requested the wake word be changed to `紫悦`.

Actions:

- Use `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`.
- Configure the official-compatible product overlay for custom MultiNet wake:
  `zi yue` / `紫悦`, threshold `20`, MultiNet7, no wake-word data send, AFE
  WakeNet disabled, and HiStackChan WakeNet disabled.
- Preserve `a21-stackchan-official-xiaozhi-compatible.bin`, official StackChan
  app surface, and `GetHAL().startXiaozhi()`.
- Build the product candidate, inspect `sdkconfig.json`, run no-write flash
  plan, then use only the official-compatible guarded flash execute path.

Current result:

- Focused app tests passed for the official-compatible overlay, `紫悦` wake
  contract, and product flash-lane guard.
- `make verify` passed.
- First ESP-IDF attempt failed at Python 3.13 `_csv` dynamic-library system
  policy during `gen_crt_bundle.py`; manual replay and a second build passed,
  so this is recorded as an environment hiccup, not a firmware/config failure.
- Build report
  `reports/a21-stackchan-official-baseline-20260603-030428-1780427068064697000.json`
  is `status=passed`; app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  has SHA-256
  `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`.
- Build config inspection proved:
  `BOARD_TYPE_M5STACK_STACK_CHAN=true`,
  `BOARD_TYPE_M5STACK_CORE_S3=false`,
  `USE_CUSTOM_WAKE_WORD=true`,
  `CUSTOM_WAKE_WORD="zi yue"`,
  `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`,
  `CUSTOM_WAKE_WORD_THRESHOLD=20`,
  `SEND_WAKE_WORD_DATA=false`,
  `WAKE_WORD_DETECTION_IN_LISTENING=false`,
  `USE_AFE_WAKE_WORD=false`,
  `SR_MN_CN_MULTINET7_QUANT=true`, and
  `SR_WN_WN9_HISTACKCHAN_TTS3=false`.
- No-write flash plan
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030447-1780427087258380000.json`
  is `status=ready`, `dry_run=true`, `flash_allowed=false`,
  `flash_executed=false`, app `a21-stackchan-official-xiaozhi-compatible.bin`,
  offset `0x20000`, port `/dev/cu.usbmodem1101`.
- Guarded flash execute
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json`
  passed on `/dev/cu.usbmodem1101` with control commit `7f3225ee1fe7`,
  clean worktree, app `a21-stackchan-official-xiaozhi-compatible.bin`, app
  offset `0x20000`, and app SHA-256
  `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`.
- Post-flash Gateway `127.0.0.1:21081` remained healthy; device
  `44:1b:f6:e2:6a:60` reconnected online at 03:08:51 with volume `100`, trace
  `a21-trace-44-1b-f6-e2-6a-60`.
- Operator physical verification failed: saying `紫悦` produced no response.

Acceptance conditions:

- Guarded flash execute passes with app
  `a21-stackchan-official-xiaozhi-compatible.bin` at offset `0x20000`.
- Physical device reboots into the StackChan-compatible avatar/action UI, not
  plain Xiaozhi.
- Operator says `紫悦` and the device wakes without screen touch. This was not
  met.
- If `紫悦` misses or false-wakes, the next transition tunes only custom wake
  phrase/threshold while keeping the product app lane.

Failure state:

- `F-WAKE-002-UNGUARDED-WRITE` if any worker performs a background flash/NVS
  write.
- `F-WAKE-002-BARE-XIAOZHI-REGRESSION` if product hardware is flashed with
  `xiaozhi.bin` again.
- `F-WAKE-002-STACKCHAN-UI-REGRESSION` if the flashed app shows plain Xiaozhi
  rather than StackChan-compatible UI.
- `F-WAKE-002-PHYSICAL-WAKE-REJECTED` if the operator cannot wake it by saying
  `紫悦`.

Rollback path:

- Reflash the last accepted official-compatible app artifact through
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- If only wake sensitivity is wrong, keep the same product app lane and tune
  `CONFIG_CUSTOM_WAKE_WORD_THRESHOLD` from `20` toward `35`.
- Preserve NVS connection settings unless a rollback plan explicitly requires
  reprovisioning.

Next state:

- `S4F-ZI-YUE-PHYSICAL-WAKE-REJECTED`

### Completed T-WAKE-INTEGRATE-001: Custom Wake Bare Flash Guard And Recovery

Current state:

- `S2D-CUSTOM-WAKE-PACKAGE-BARE-FLASH-INVALID-ROLLBACK-DONE`

Target state:

- `S2E-CUSTOM-WAKE-BARE-FLASH-GUARDED`

Trigger:

- Custom wake package existed, but the live physical firmware used stock
  Xiaozhi WakeNet/touch activation and the operator reported weak wake
  response.
- A foreground attempt to flash the restored bare wake build succeeded
  technically but restored plain Xiaozhi UI, proving that raw `xiaozhi.bin`
  is the wrong product artifact for StackChan.

Actions:

- Use `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`.
- Worker `019e8993-0987-7601-9b8a-3aa4e88ebfaa` owns only the small flash-lane
  guard hotfix; it must not run hardware writes.

Current result:

- Wake package integrity passed below activation:
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`.
- Artifact exists and matches SHA-256
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`.
- Bare wake flash execution
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`
  passed but is now incident evidence only because it flashed app file
  `xiaozhi.bin`.
- Corrective StackChan-compatible flash execution
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`
  restored app file `a21-stackchan-official-xiaozhi-compatible.bin`.
- Post-restore volume/TTS evidence passed on traces
  `a21-trace-recovery-stackchan-volume-1780424386012` and
  `a21-trace-recovery-stackchan-relay-wav-1780424386012`.

Acceptance conditions:

- No future product StackChan runbook uses `xiaozhi-firmware-flash-execute` or
  app file `xiaozhi.bin`.
- Generic `xiaozhi-firmware-flash-*` rejects product-looking `xiaozhi.bin`
  unless explicitly marked `--non-product-dev`.
- Corrective StackChan-compatible flash path is documented and verified.

Failure state:

- `F-FW-003-UNGUARDED-WRITE` if any background flash/NVS write is attempted.
- `F-FW-003-BARE-XIAOZHI-PRODUCT-FLASH` if a product StackChan device is
  flashed again with `xiaozhi.bin`.
- `F-FW-003-PACKAGE-STALE` if future wake assets cannot be reconciled with the
  accepted audio/official-compatible baseline.

Rollback path:

- Reflash the last accepted official-compatible app artifact through a guarded
  foreground path if custom wake regresses behavior.
- Preserve NVS connection settings unless a rollback plan explicitly requires
  reprovisioning.

Next state:

- `S2E-CUSTOM-WAKE-BARE-FLASH-GUARDED`

### Completed T-PROVIDER-002: Real Provider/TTS Hot-Plug Candidate

Current state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`
- `S3C-STEPFUN-IFLYTEK-HOTPLUG-CODED`

Result:

- Commit `b5a405e` added host-side `iflytek_tts`, StepFun explicit
  compatibility selection, voice-pipeline TTS selection, and retained
  `voice_clone_cli`.
- Focused tests and `make verify` passed before this follow-up transition.
- Mac direct Iflytek TTS remained blocked, which created active
  `T-PROVIDER-002b`.

Next state:

- `S3D-STEPFUN-IFLYTEK-LIVE-UNBLOCK-IN-PROGRESS`

### Completed T-INTEGRATE-001: Review And Integrate Accepted Audio Hotfixes

Current state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

Target state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

Trigger:

- Operator accepted the 3x physical audio result after recording
  `/Users/jiyurun/Downloads/军民公路259号 10.m4a`.

Actions:

- Review the T-AUDIO-002 and T-AUDIO-003 hotfix diff for stock-protocol,
  audio-leveling, session/trace, and half-duplex boundaries.
- Update project state and handoff docs with accepted recording evidence.
- Run focused Go tests, desktop helper syntax checks, `git diff --check`, and
  live Gateway/device status checks.
- Commit the accepted hotfixes into the current integration branch if review
  finds no blocking issue.

Acceptance conditions:

- No blocking review findings.
- Verification commands pass.
- Commit captures code, tests, plans, protocol docs, state machine, and handoff
  log.
- Integration commit has been created on the current branch.

Failure state:

- `F-INTEGRATE-001-BLOCKING-REVIEW-FINDING` if review finds a protocol,
  session, safety, or acceptance-labeling issue.
- `F-INTEGRATE-001-VERIFY-FAILED` if tests or syntax/diff checks fail.

Rollback path:

- Do not commit; keep the dirty branch available for targeted fixes.

Next state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

### Previous T-AUDIO-003: Bounded TTS Gain And Host-Say Input Suppression

Current state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

Target state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

Trigger:

- Operator recording `9.m4a` after `/v1/xiaozhi/say` remained too quiet:
  from 3 seconds it measured `-36.99 LUFS`, `-16.05 dBTP`; best early
  10-second window measured `-34.79 LUFS`, `-16.05 dBTP`.
- Sidecar analysis found the Gateway downlink PCM path only attenuated
  full-scale frames and never lifted quiet TTS with available headroom.
- Half-duplex sidecar found `/v1/xiaozhi/say` did not arm input suppression,
  so speaker playback could be captured by the microphone and retrigger listen
  or voice-pipeline flow.

Actions:

- Use `docs/plans/2026-06-02-stackchan-tts-gain-half-duplex-hotfix.md`.
- Replace pure headroom limiting with bounded downlink PCM leveling:
  tiny noise below gate is unchanged, quiet non-silent TTS is lifted toward a
  target peak with max-gain cap, and hot frames remain below headroom.
- Keep the live stock protocol unchanged.
- Add host-say-only short input suppression for stock physical devices.
- Keep normal dialogue half-duplex acceptance separate.

Acceptance conditions:

- Focused tests prove quiet TTS frames are boosted, tiny noise is not boosted,
  and full-scale frames remain capped.
- Focused test proves immediate listen restart plus speech Opus after host-say
  is ignored for stock physical devices without starting a new voice pipeline.
- Physical recording after the gain hotfix improves materially without obvious
  clipping, but listening quality must remain acceptable. 4x candidate evidence
  `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` measured `-21.65 LUFS` and
  `-4.72 dBTP` from 3 seconds but was rejected by operator listening feedback.
  The current 3x candidate is expected to preserve clarity with less peak
  stress; follow-up recording
  `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured
  `-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds.
- Final 3x Gateway trace `a21-trace-stackchan-say-1780417217` delivered
  `579` TTS chunks to the live physical stock socket after volume `100` trace
  `a21-trace-stackchan-volume-1780417211`.
- Operator accepted final recording
  `/Users/jiyurun/Downloads/军民公路259号 10.m4a`, which measured
  `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds.

Failure state:

- `F-AUDIO-003-CLIPPING-OR-HARSHNESS` if operator reports harshness,
  electrical interruption, or the next recording shows clipping.
- `F-AUDIO-003-STILL-QUIET` if physical loudness remains unacceptable after
  bounded Gateway gain.
- `F-AUDIO-003-SELF-TRIGGER` if physical host-say still retriggers voice flow.

Rollback path:

- Remove or lower the bounded PCM leveling constants while leaving stock
  protocol, firmware, provider, and V21 untouched.
- Remove the host-say suppression helper if it blocks legitimate operator
  foreground tests.
- If safe 3x Gateway gain is insufficient, move to guarded firmware codec-gain
  or persistent volume work instead of stacking more host gain.

Next state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

### Previous T-AUDIO-002: Stock Xiaozhi Audio Parity And Runtime Volume Hotfix

Current state:

- `S3-LIVE-HOTFIX-DEPLOYED-AND-LONG-TTS-SENT`

Target state:

- `S4-PHYSICAL-STOCK-AUDIO-RETESTED`

Trigger:

- The operator recording after the volume-92 flash still sounded blurred,
  intermittent, and unclear.
- Parallel read-only workers found that official StackChan Xiaozhi behavior is
  not just "same protocol": client/uplink is 16 kHz, server/downlink/CoreS3
  output is 24 kHz, stock firmware does not accept `listen` replies, official
  volume is exposed through MCP, and A21 was rebuilding Opus downlink encoders
  per frame.

Actions:

- Use `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`.
- Keep client/uplink validation at 16 kHz while restoring server/downlink TTS
  to 24 kHz mono 60 ms.
- Suppress unsupported `listen` replies for stock physical MAC-address devices.
- Reuse the turn-level Opus encoder for contiguous same-format downlink frames
  and reduce initial prebuffer from five frames to one frame.
- Add `POST /v1/xiaozhi/speaker-volume` as a stock MCP
  `self.audio_speaker.set_volume` delivery path.
- Add `POST /v1/xiaozhi/say` as an operator foreground path that sends stock
  TTS lifecycle plus Opus binary downlink over the live `/v1/xiaozhi` socket.
- Update the desktop StackChan control helper to call the runtime volume
  endpoint and host-say endpoint.
- Do not flash firmware, write NVS, execute V21, or claim physical acceptance
  in this transition.

Acceptance conditions:

- Focused Gateway/provider/transport/app tests pass.
- `git diff --check` passes.
- Desktop helper syntax checks pass.
- A Gateway built from this tree can set volume `100` on the live stock
  Xiaozhi socket, then play a long TTS turn for physical recording. This has
  been demonstrated by trace `a21-trace-live-long-tts-hotfix`.
- Physical acceptance remains red until the operator/instrument recording is
  analyzed and logged.

Failure state:

- `F-AUDIO-002-MCP-NOT-SUPPORTED` if the live device does not advertise
  `features.mcp=true` or does not react to the official volume tool.
- `F-AUDIO-002-PHYSICAL-STILL-BROKEN` if post-hotfix audio remains blurred or
  intermittent, shifting focus to TTS model/profile and device-side playback
  instrumentation.
- `F-AUDIO-002-REGRESSION` if focused playback/barge-in tests fail.

Rollback path:

- Revert the Gateway/provider/transport patches and keep the already flashed
  firmware unchanged.
- If runtime volume delivery behaves unexpectedly, stop using
  `/v1/xiaozhi/speaker-volume` and resume the guarded firmware volume-100
  candidate path.

Next state:

- `S4-PHYSICAL-STOCK-AUDIO-RETESTED`

### Candidate T-HW-VOLUME-002: Raise StackChan Fixed Output After Volume92 A/B Failed

Current state:

- `S3B-VOLUME92-A-B-FAILED`

Target state:

- `S4-VOLUME100-CANDIDATE-BUILD-FLASH-A-B`

Trigger:

- The user clarified the target is StackChan device loudness, not macOS system
  volume.
- The latest phone recording is stronger than the previous one but still has
  low sustained loudness and narrow active speech spectrum.
- Current stock `/v1/xiaozhi` has no Gateway runtime speaker-volume setter, so
  the fastest honest path is a guarded firmware-side output-gain candidate.
- Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` has prepared the fixed
  official codec output-volume patch for main-thread review.
- Main-thread no-write build/report has passed for the patched official
  Xiaozhi-compatible candidate.
- Main-thread no-write flash-plan report is ready, and read-only flash-gate
  worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed the artifact hash,
  USB port, dry-run safety fields, and rollback baseline.
- The operator explicitly opened the foreground hardware window and supplied the
  guarded execute command with confirmation token.
- Main-thread flash execute passed with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`.
- Post-flash recording `/Users/jiyurun/Downloads/军民公路259号 7.m4a`
  measured `-33.4 LUFS` integrated loudness and `-14.3 dBFS` true peak,
  weaker than the 22:19 reference at `-25.8 LUFS` and `-4.2 dBFS`.
- The analysis report
  `reports/a21-operator-recording-audio-analysis-20260602-2310-stackchan-volume92.json`
  records `post_flash_loudness_accepted=false` and no clipping evidence.

Actions:

- Use `docs/plans/2026-06-02-stackchan-volume-action-control.md`.
- Keep the main thread as architecture/control only.
- Keep the failed volume-92 flash execute report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
  as the rollback/comparison evidence.
- Dispatch a scoped worker to raise the official Xiaozhi-compatible fixed
  output candidate to `SetOutputVolume(100)` and verify the official codec
  `SetOutputVolume` ordering without touching hardware.
- Main thread reviews the worker commit, runs no-write build and flash-plan,
  then opens a foreground flash window only after operator approval.
- After any volume-100 flash, trigger a real stock Xiaozhi turn, collect a
  post-flash phone recording, and compare it against both `7.m4a` and the
  22:19 reference sample.
- Do not add runtime volume protocol, do not use macOS volume as evidence, do
  not use legacy local audio playback, do not write NVS, do not execute
  provider/V21, and do
  not promote diagnostic tone or host-only evidence to PRD acceptance.

Acceptance conditions:

- The official Xiaozhi-compatible overlay explicitly sets official codec output
  volume `100` before entering the Xiaozhi runtime.
- Existing `GetHAL().startXiaozhi()` behavior remains preserved.
- Focused guard/test or build-report evidence proves the volume setting exists
  in the candidate artifact path.
- A no-write official-compatible build/report passes for the volume-100
  candidate before any flash is requested.
- A no-write flash plan is ready and keeps `flash_allowed=false` until the
  operator explicitly runs the guarded execute command with the confirmation
  token.
- Foreground flash execute and before/after audible A/B are recorded before
  physical loudness is accepted.
- `git diff --check` passes, and any touched Go guard tests pass.
- State and handoff docs record that physical before/after audibility remains
  unaccepted until foreground flash plus phone/instrument A/B.

Failure state:

- `F-HW-VOLUME-001-NO-OFFICIAL-CODEC-SEAM` if the worker cannot locate a safe
  official codec setting point.
- `F-HW-VOLUME-001-SCOPE-DRIFT` if runtime protocol, Gateway behavior,
  provider/V21, NVS, flash, or macOS audio is touched outside the plan.
- `F-HW-VOLUME-001-PHYSICAL-OVERCLAIM` if the code candidate is treated as
  product voice acceptance without foreground physical evidence.
- `F-HW-VOLUME-002-NO-AUDIBLE-GAIN` if volume 100 also fails to improve the
  physical recording, in which case control returns to `T-AUDIO-001` TTS/model
  and playback-chain RCA.

Rollback path:

- Revert the overlay/test/doc patch if the candidate build or guard fails.
- Keep the currently flashed firmware and NVS route unchanged until an explicit
  hardware window approves flash.
- If a flashed volume trial sounds worse, flash back to the last accepted
  official-compatible app SHA recorded in the firmware flash reports.

Next state:

- `S4-VOLUME100-CANDIDATE-BUILD-FLASH-A-B`

### T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback

Current state:

- `S1C-STOCK-XIAOZHI-OPERATOR-RECORDING-PENDING`

Target state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

Trigger:

- Physical Gateway downlink is now proven as a candidate path, but the user
  reports that the audible sound is still wrong and likely TTS-related.
- Previous audio/TTS optimization commits are present, but they only prove
  host-side PCM/downlink guardrails, not physical audible quality.

Actions:

- Use `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`.
- Keep the main thread as control tower.
- Dispatch a worker for host downlink objective isolation.
- Integrate the worker result so `xiaozhi-voice-bench` can report decoded
  Opus/downlink PCM quality without storing raw audio.
- Run host-only product-chain bench against current A21 Gateways to verify
  post-Opus quality before touching physical firmware or TTS profile routing.
- Avoid hardware writes, NVS writes, provider/V21 execution, unapproved Mac
  audio playback, and PRD overclaiming.
- After worker return, run a foreground physical A/B only if the operator
  approves any Gateway/TTS profile change.
- For the current stock Xiaozhi physical connection, do not use
  `stackchan-local-tts-playback` as evidence: it requires the legacy A21 audio
  WebSocket and returned `409 device audio websocket is not connected` during
  the 2026-06-02 foreground recording window.
- Collect the next physical audible sample by triggering a real stock Xiaozhi
  listen/audio turn on the device, or first create a separate planned
  transition for a stock-safe TTS injection seam.

Acceptance conditions:

- The project can classify the bad sound as TTS/model, Opus/downlink, firmware
  playback, or stock-control compatibility with concrete evidence.
- Host-only and physical evidence remain labeled separately.
- Any follow-up fix has focused tests and does not reintroduce the old PCM
  bridge as a product path.

Failure state:

- `F-AUDIO-001-UNISOLATED` if reports still cannot distinguish TTS generation
  from post-Opus/downlink quality.
- `F-AUDIO-001-PHYSICAL-OVERCLAIM` if host-only evidence is promoted to PRD
  acceptance.
- `F-AUDIO-001-SCOPE-DRIFT` if a worker touches firmware, NVS, provider/V21,
  or Mac audio outside the plan.

Rollback path:

- Keep the current `sherpa_onnx_tts` host-local route as baseline.
- Revert any host-only report additions if they destabilize tests.
- Do not mutate firmware/NVS in this transition, so hardware rollback should
  not be needed.

Next state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

### T-HW-002: Recover Network/Relay And Collect Physical Evidence

Current state:

- `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

Target state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

Trigger:

- The official Xiaozhi-compatible candidate was flashed through the T7
  foreground guard.
- The latest firmware no longer enters the setup/QR or watchdog failure path.
- The old temporary relay returned `503`; a LAN-bound A21 Gateway on port
  `21081` was verified by `/healthz` and `/xiaozhi/ota/`.
- A foreground guarded NVS update pointed OTA/WS to the LAN Gateway and the
  device connected through stock Xiaozhi WebSocket after wake.

Actions:

- Complete audible playback or trusted device playback ack evidence for the
  physical Xiaozhi turn.
- Preserve the physical Gateway trace and serial evidence without storing raw
  audio or transcript bodies.
- Close real provider smoke on the host side, then rerun readiness.
- Keep custom wake as blocked until guarded wake firmware flash and physical
  custom wake proof are recorded.

Acceptance conditions:

- Device connects to A21 Gateway using stock-compatible Xiaozhi protocol.
- Real microphone input, Gateway downlink, barge-in stop, official
  avatar/action, wake behavior, and provider rotation evidence are recorded
  without leaking keys or debug-only protocol fields.
- Audible playback observation or trusted device playback ack is present.
- Host/mock/candidate evidence remains labeled separately from physical
  acceptance.

Failure state:

- `F-HW-002-STALE-RELAY` if the recorded temporary relay no longer resolves or
  forwards OTA/WS.
- `F-HW-002-WIFI-NOT-CONFIGURED` if the device remains in AP config mode.
- `F-HW-002-UNCONFIRMED-WRITE` if any worker attempts background NVS/flash
  writes.
- `F-HW-002-NO-PHYSICAL-EVIDENCE` if the device connects but evidence is not
  recorded.
- `F-HW-002-AUDIBLE-ACK-MISSING` if Gateway downlink exists but physical
  audible playback or trusted device playback ack remains unproven.

Rollback path:

- Preserve boot/NVS evidence before changing connection settings.
- Restore the previous known-good official StackChan package through a guarded
  foreground flash path if the A21 candidate must be reverted.

Next state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

## Completed Transitions

| Transition | Result | Notes |
| --- | --- | --- |
| T-FW-001: Freeze external X21 Xiaozhi builds | Completed | Commit `f7c95f0`; protects A21 from consuming external X21 Xiaozhi build dirs. |
| T-GW-001: Sync Xiaozhi turns to official StackChan | Completed | Commit `a36206f`; supports official StackChan turn synchronization. |
| T-GW-002: Relay official StackChan avatar actions | Completed | Commit `ea51c67`; maps Gateway state/action to official StackChan relay. |
| T-TR-001: Map A21 events to official StackChan frames | Completed | Commit `cdabe89`; keeps screen/action relay on official StackChan packet shapes. |
| T-FW-002: Add official Xiaozhi-compatible StackChan build | Completed host/build candidate | Commit `987bbb0`; candidate build lane exists, physical flash still pending. |
| T-GOV-001: Establish repo-carried workflow state | Completed | Commit `69c4bbe`; adds handoff log, state machine, and plan discipline. |
| T-VERIFY-001: Integrated host verification after governance merge | Completed | `make verify` passed; mainline official candidate rebuild passed with app SHA-256 `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`. |
| T-FW-004: Add official compatible candidate flash seam | Completed | Commit `6f34091`; no-write plan passed for `/dev/cu.usbmodem1101` with `dry_run=true`, `flash_allowed=false`, and app offset `0x20000`. |
| T-FW-005: Add official compatible NVS connection config | Completed | Commit `37ef8f3`; T7 NVS write report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json` preserved servo calibration and Wi-Fi credentials while updating OTA/WS route. |
| T-FW-006: Autostart official Xiaozhi candidate | Superseded | Commit `e7e9b03`; initial autostart removed setup gate but hit a setup-uninstall watchdog path during field testing. |
| T-FW-007: Enter official Xiaozhi runtime directly | Completed | Commit `4613946`; latest app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`; flashed through report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`. |
| T-GOV-002: Recover hardware network state docs | Completed | Commit `eeacbd3`; reconciled collapsed control thread, active plan, state machine, and handoff log before foreground NVS execution. |
| T-HW-002a: Refresh connection route and prove physical Gateway downlink | Completed candidate | Latest NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json`; serial logs `reports/a21-stackchan-direct-xiaozhi-serial-reset-20260602-2128.log` and `reports/a21-stackchan-physical-wake-serial-20260602-2130.log`; physical evidence report `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json`; readiness report `reports/a21-product-readiness-20260602-213204.json`. |
| T-AUDIO-000: Read-only Xiaozhi audio/protocol audit | Completed | Worker thread `019e888d-f57d-7922-8e48-24b00255a122` found audio optimization landed, the physical audio path is stock-profile Xiaozhi Opus through A21 Gateway rather than old PCM bridge, and the most likely bad-sound boundary is TTS generation before Opus/downlink. |
| T-AUDIO-001a: Host downlink objective isolation | Completed | Worker thread `019e8895-43a6-7e23-a4f3-601f0451ab50`; `xiaozhi-voice-bench` now decodes captured binary downlink Opus frames and reports redacted aggregate `downlink_audio_quality` while preserving host-only candidate semantics. |
| T-AUDIO-001b: Host/Gateway post-Opus quality run | Completed host-only candidate | `reports/a21-xiaozhi-voice-bench-20260602-220325.719331000.json` passed 3-repeat host product-chain bench with `sherpa_onnx_tts`, answer p95 397 ms, and post-Opus quality passed; `reports/a21-xiaozhi-voice-bench-20260602-220344.175240000.json` passed a one-round check on the physical LAN Gateway; readiness `reports/a21-product-readiness-20260602-220447.json` and bundle `reports/a21-server-side-readiness-bundle-20260602-220501.json` still correctly block launch. |
| T-AUDIO-001c: Foreground long-TTS push attempt | Blocked, evidence-preserving | macOS output volume was set to 100 after explicit operator request; physical device `44:1b:f6:e2:6a:60` was online on stock Xiaozhi Opus via Gateway `21081`; `go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --engine sherpa_onnx ...` returned `409 device audio websocket is not connected`, so no valid physical StackChan playback was claimed. |
| T-OPS-001: Desktop StackChan control boundary helper | Completed | Added `tools/desktop/a21-stackchan-control.command` and copied it to `/Users/jiyurun/Desktop/A21-StackChan-Control.command`; status check passed against Gateway `21081`; action probes and diagnostic tone correctly reported current stock-session 409 blockers instead of claiming control. |
| T-HW-VOLUME-001a: Fixed official codec volume candidate prep/build | Completed no-hardware candidate | Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` prepared official codec `SetOutputVolume(92)` in the Xiaozhi-compatible overlay before `GetHAL().startXiaozhi()` and added a Go guard test; main branch integrated the patch, focused Go tests, official StackChan Go subset, clean official `HEAD` patch-apply check, `git diff --check`, and no-write build report `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json` passed. App SHA-256: `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. |
| T-HW-VOLUME-001b: Fixed official codec volume flash gate | Completed no-write gate | No-write flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json` is `status=ready`, `port=/dev/cu.usbmodem1101`, `dry_run=true`, `flash_allowed=false`, `flash_executed=false`, and app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`; read-only worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed it is enough to request an operator-approved hardware window, not enough for physical acceptance. Read-only worker `019e88d6-ad61-7513-b379-aa21ae7db150` prepared the A/B evidence runbook and confirmed `stackchan-local-tts-playback`, Mac volume, diagnostic tone, host-only bench, and dry-run reports are not valid physical loudness acceptance. Rollback baseline: `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`, app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`. |
| T-HW-VOLUME-001c: Foreground fixed codec volume flash | Completed hardware write, A/B pending | Operator supplied guarded execute command; flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json` is `status=passed`, `port=/dev/cu.usbmodem1101`, `dry_run=false`, `flash_allowed=true`, `flash_executed=true`, app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. Post-flash Gateway `127.0.0.1:21081` health returned ok and device `44:1b:f6:e2:6a:60` remained registered online. This is not physical loudness acceptance. |
| T-FW-003a: Bare wake flash incident and StackChan-compatible recovery | Completed recovery | Bare wake flash execution `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json` passed generic T7 but flashed app `xiaozhi.bin`, reverting the physical device to plain Xiaozhi UI. Corrective no-write plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021802-1780424282862523000.json` and execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json` restored `a21-stackchan-official-xiaozhi-compatible.bin` with app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. Post-restore runtime volume and relay WAV playback succeeded on traces `a21-trace-recovery-stackchan-volume-1780424386012` and `a21-trace-recovery-stackchan-relay-wav-1780424386012`. |
| T-FLASH-GUARD-001: Product artifact-lane guard | Completed | Generic `xiaozhi-firmware-flash-*` now rejects product-looking app `xiaozhi.bin` at `0x20000` unless explicitly marked `--non-product-dev`; rejection points to `a21-stackchan-official-xiaozhi-compatible-flash-execute`. Real incident build no-write plan now exits nonzero with the guard message. Correct product no-write plan still passes in `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-023008-1780425008542994000.json`. Focused app tests, `git diff --check`, and `make verify` passed. |
| T-PROTOCOL-001a: Stock Xiaozhi listen warning plan-first audit | Completed plan | Worker `019e88dd-1517-77c2-a002-7db771ec016a` created `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`; strongest hypothesis is Gateway sends server-to-device `type=listen` ack frames that stock firmware does not accept, while binary Opus/TTS downlink still reaches `speaking`. |
| T-PROVIDER-001a: Provider readiness truth audit | Completed read-only audit | Worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` found `T-PROVIDER-001` stale as a missing-smoke blocker. Usable explicit provider evidence exists: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`, local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json`, provider-ready readiness `reports/a21-product-readiness-20260602-160841.json`, and server-side bundle `reports/a21-server-side-readiness-bundle-20260602-160841.json`. Newer generic readiness reports that selected `mock` are invocation/context regressions, not absence of provider evidence. |
| T-PROVIDER-001b: Selected provider readiness refresh | Completed, server-side still blocked by non-provider gates | Worker `019e8974-346e-7912-93b2-77cdbb9f3acf` pinned readiness to route-eligible DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`; generated `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json` with `provider.selected=deepseek`, `real_provider_ready=true`, and `smoke_status=passed`; generated `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json` with `provider.ready=true`. Reports remain `server_side_blocked` because V21, host voice/continuous pipeline, physical StackChan, and wake-word gates are still open. |
| T-AUDIO-002: Stock Xiaozhi audio parity and runtime volume hotfix | Completed physical candidate | Restored 24 kHz downlink while keeping 16 kHz uplink, suppressed unsupported stock physical `listen` replies, reused turn-level Opus encoder, reduced prebuffer to one frame, added stock MCP speaker volume endpoint, and added foreground `/v1/xiaozhi/say`; focused Gateway/provider/transport/app tests passed. |
| T-AUDIO-003: Bounded 3x TTS gain and host-say suppression | Completed accepted | 4x gain was rejected by operator listening feedback; final 3x gain delivered `a21-trace-stackchan-say-1780417217`, final accepted recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds, and host-say suppression markers were observed. |
| T-AUDIO-004a: Bare Xiaozhi audio and whole-device parity read-only audit | Completed read-only audit | Sidecar thread `019e899d-e8c2-71e0-a10c-9e7e34ac4cff` reported the operator-confirmed fact that bare `xiaozhi.bin` was louder and clearer, while keeping it classified as a non-product/incident artifact. The explanation is not protocol alone: the key differences are official CoreS3 codec/HAL behavior, source TTS/mastering, runtime volume/NVS/MCP state, bounded leveling, gopus downlink, pacer/encoder state, and official StackChan app/action initialization. It recommends migrating parity conditions into `a21-stackchan-official-xiaozhi-compatible`, not returning to the bare `xiaozhi.bin` or the old M5Unified PCM bridge. |
| T-HALF-DUPLEX-002: No-flash normal dialogue self-trigger observation | Completed candidate | Main thread used the online stock Xiaozhi device without firmware flash or operator click, delivered relay WAV `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` through `/v1/xiaozhi/say`, and generated `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json` with `status=candidate_passed_no_self_trigger`, trace `a21-trace-no-flash-dialogue-observe-1780423245`, `audio_chunks=40`, `wait_after_say_ms=20000`, `event_count=869`, empty `self_trigger_event_names`, and `input_suppressed_count=1`. Diagnostic half-duplex counters remain a separate optional firmware path. |
| T-ASR-GREEN-LATENCY-001: Bound Xiaozhi listen from firmware | Flashed, physical validation pending | Physical trace `a21-trace-44-1b-f6-e2-6a-60` showed `xiaozhi.listen.start=110`, `xiaozhi.opus_frame.received=8034`, repeated `listen.stop -> listen.start`, and wake disabled while listening. Root cause candidate: official `HandleStartListeningEvent()` forced `kListeningModeManualStop`, while the A21 no-speech timeout only armed for `kListeningModeAutoStop`. The overlay now routes StartListening through `GetDefaultListeningMode()`, starts the no-speech timer for non-realtime listening, and keeps VAD silence auto-stop scoped to AutoStop after speech. Focused tests, `make verify`, and product build passed; build report `reports/a21-stackchan-official-baseline-20260603-042940-1780432180247638000.json`, app SHA-256 `ca0877d09eecfb9c69f2279ce14c95942a2cf966e05f118e82551e92c14665b9`. No-write plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043116-1780432276384588000.json` passed, and guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043222-1780432342953949000.json` passed on `/dev/cu.usbmodem1101` from clean commit `88cb8a92069d`. Post-flash Gateway evidence: device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered by stock MCP trace `a21-trace-listen-bound-volume-1780432365`, and post-flash trace events since `1780432350000` contained only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-WAKE-003b: Split Zi Yue MultiNet command list | Flashed, physical validation pending | Source inspection found `CUSTOM_WAKE_WORD` is documented as one pinyin command, while `CustomWakeWord::Initialize()` adds each command through `esp_mn_commands_add`. The earlier `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue` config therefore likely registered as one invalid/overlong command instead of four alternatives. The overlay now splits `CONFIG_CUSTOM_WAKE_WORD` on `|`, trims each command, and registers each as a separate wake command with display/greeting `紫悦`. Focused tests, `make verify`, and official-compatible build passed; build report `reports/a21-stackchan-official-baseline-20260603-043851-1780432731137364000.json`; app SHA-256 `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`. Binary strings include `Loaded %d A21 custom wake command(s) for %s` and the multi-phrase config. Commit `8e4df0b` was flashed through no-write report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044128-1780432888876682000.json` and guarded execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044234-1780432954645477000.json` on `/dev/cu.usbmodem1101`; T7 guard saw clean commit `8e4df0b30c0f`, app SHA `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`. Device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` delivered by trace `a21-trace-multi-wake-volume-1780432980`, and post-flash trace since `1780432954000` showed only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001b: Park after direct Xiaozhi start | Current HEAD reflashed, visual validation pending | Operator reported the welcome/setup page reappeared after the multi-wake product flash even though the binary contained the direct autostart log. Source inspection proved the false assumption: `Hal::startXiaozhi()` starts Xiaozhi tasks and returns, so `main.cpp` could continue into the Mooncake main loop and AppLauncher setup worker. The overlay now parks `app_main` after direct `GetHAL().startXiaozhi()` with `GetHAL().feedTheDog()` and `GetHAL().delay(1000)`, before the Mooncake main loop. Focused app tests, `git diff --check`, and `make verify` passed in the original hotfix. Product build report `reports/a21-stackchan-official-baseline-20260603-054837-1780436917458986000.json` passed with app SHA-256 `e3758cffdc294e74220f5288a33306d4aace164279bea6acb3a2cea5af17e1c3`; guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-054954-1780436994114555000.json` passed on `/dev/cu.usbmodem1101` from clean commit `e694550f1739`. After later provider commits, the control tower rebuilt and flashed current HEAD `7906975`: build `reports/a21-stackchan-official-baseline-20260603-062637-1780439197207182000.json`, flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062715-1780439235852990000.json`, and execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json` all passed; app SHA-256 is `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`. Device `44:1b:f6:e2:6a:60` reconnected online, runtime speaker volume `100` was delivered by trace `a21-trace-current-head-volume-1780439333`, and post-flash trace events since `1780439333539` contain only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-ASR-GREEN-LATENCY-002: Gateway no-speech cooldown | Completed runtime hotfix, physical validation pending | Live trace before the fix showed `xiaozhi.listen.start=333`, `audio.ingress.buffered=25115`, `stackchan.official_auto.not_connected=1025`, and repeated `listen.stop -> placeholder tts.stop -> listen.start` loops. Gateway now arms a short stock-physical input suppression after `placeholder_no_asr_tts`, recording `xiaozhi.no_speech.input_suppression_armed` and reason-specific `xiaozhi.listen.start.suppressed_after_no_speech` so an immediate restart is ignored instead of opening a new turn. Focused Gateway tests passed, `make verify` passed, Gateway `a21-gateway-21081` was restarted from this worktree, `/healthz` returned `service=a21-gateway,status=ok`, device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered by `a21-trace-no-speech-cooldown-final-volume-1780434593`, and live trace `a21-trace-44-1b-f6-e2-6a-60` later showed the cooldown firing once: `xiaozhi.no_speech.input_suppression_armed=1`, `xiaozhi.listen.start.input_suppressed=1`, `xiaozhi.listen.start.suppressed_after_no_speech=1`, with only one `xiaozhi.turn.start`. |
| T-XIAOZHI-STREAMING-ASR-PROVIDER-001a: Static provider readiness gate | Completed truthful blocker | Plan `docs/plans/2026-06-03-xiaozhi-streaming-provider-readiness.md` defines the strict provider requirements for Xiaozhi realtime parity. New CLI `a21 xiaozhi-streaming-provider-readiness` is static/no-execute and blocks mock, batch WAV, and file-boundary paths. Default report `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055502.json` is `gate_status=blocked` with mock ASR/LLM/TTS findings. Selected StepFun+Iflytek check `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055512.json` is also `gate_status=blocked`: LLM text stream is ready, but Sherpa ASR is `asr_batch_wav_boundary_not_xiaozhi_streaming` and Iflytek TTS is `tts_wav_file_boundary_not_xiaozhi_streaming`. This prevents mock `/say`, `streaming_zipformer` name-only, or WAV TTS evidence from being promoted to Xiaozhi realtime parity. |
| T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001: Selectable Sherpa streaming ASR seam | Completed adapter seam, runtime helper not proven | Plan `docs/plans/2026-06-03-sherpa-streaming-asr-adapter.md` scoped the work. Added `sherpa_onnx_streaming` / `local_sherpa_onnx_streaming` / `streaming_zipformer` selection for a `providers.StreamingASRAdapter` wrapper that keeps the old batch Sherpa adapter as fallback and starts streaming only through a configured session factory. Existing `sherpa_onnx` remains batch/WAV and is still blocked as `asr_batch_wav_boundary_not_xiaozhi_streaming`. New readiness reports prove the distinction: `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117185327000.json` blocks missing streaming helper/model without WAV-boundary findings, while `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117326052000.json` marks ASR+LLM ready when helper/model env are present but still blocks on `tts_wav_file_boundary_not_xiaozhi_streaming`. Focused provider/app tests, `git diff --check`, and `make verify` passed. No firmware, provider execution, service restart, flash, or audio playback occurred. |
| T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001: Sherpa streaming ASR real-model no-audio smoke | Completed host-local real-model pass | `local-asr-streaming-smoke` now discovers the repo-local canonical helper, sherpa-onnx Python env, and `sherpa-onnx-streaming-zipformer-zh-int8-2025-06-30` model cache when explicit env is absent. `make local-asr-streaming-smoke` generated `reports/a21-local-asr-streaming-smoke-20260603-085636-1780448196060587000.json` with `status=passed`, `evidence_mode=real_model_no_audio_streaming_smoke`, `model_files_present=true`, `frames_appended=1`, `ready_events=1`, `final_events=1`, `transcript_policy=transcript_not_recorded`, and `audio_payload_policy=raw_audio_not_recorded`. Static readiness report `reports/a21-xiaozhi-streaming-provider-readiness-20260603-085641-1780448201880789000.json` marks ASR ready from the same canonical cache while still blocking mock LLM/TTS. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host loopback, WAV/file acceptance path, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-STREAMING-TTS-ADAPTER-001: Doubao realtime TTS adapter seam | Completed adapter seam, runtime provider not executed | Plan `docs/plans/2026-06-03-xiaozhi-streaming-tts-adapter.md` scoped the work. Added an explicit `providers.StreamingTTSAdapter` marker, a Doubao realtime TTS pipeline adapter selected by `A21_TTS_FAST_PROFILE=doubao_tts_realtime`, and a PCM16 mono chunker that turns provider audio deltas into exact 60 ms downlink-ready chunks without writing or reading a WAV. Local/Iflytek/voice-clone TTS remain classified as WAV/file boundaries. New readiness reports prove the distinction: `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061849-1780438729911945000.json` blocks missing Doubao TTS config, `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730222971000.json` marks TTS ready but still blocks ASR helper proof, and `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730359984000.json` passes the static provider-shape gate when ASR helper env, StepFun, and Doubao realtime TTS env are all present. `prd_accepted` remains false because no real provider execution or physical `/v1/xiaozhi` trace was captured. Focused provider/app tests, `git diff --check`, and `make verify` passed. |
| T-STREAMING-TTS-RUNTIME-PROOF-001: Streaming TTS runtime smoke | Completed truthful blocker | Added redacted `a21 streaming-tts-runtime-smoke` plus `make streaming-tts-runtime-smoke`. Without `--execute`, local report `reports/a21-streaming-tts-runtime-smoke-20260603-074721-1780444041851710000.json` is `status=blocked`, finding `execute_flag_required`. Tests use a fake realtime dialer/session to prove `tts_session.update`, `input_text.append`, and `input_text.done` are sent, the first provider audio delta is observed while the stream is open, and at least one exact 60 ms PCM16 mono chunk is counted without WAV/file boundary. No real provider execution or physical Xiaozhi/StackChan path was used. |
| T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001: ASR partial to LLM realtime bridge | Completed host-side candidate | Added a partial transcript source for `VoicePipelineRequest`, a stock `/v1/xiaozhi` partial bridge that starts exactly one workmate streaming answer from the first ASR partial before `listen.stop`/`asr.final`, and parity gates that require ordered `asr.stream.commit` while blocking host-loopback fake markers. Focused provider/Gateway/app tests passed. This does not execute real providers/V21, start Gateway, flash firmware, play audio, or claim physical PRD acceptance. |
| T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001: Realtime parity real profile evidence | Completed evidence hardening | Plan `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md` scoped the no-execute/no-hardware cut. Gateway now records redacted profile-class markers for Xiaozhi voice-pipeline turns, distinguishing real streaming ASR/LLM/TTS from mock, batch, and file-boundary stages without storing raw provider names, transcripts, provider outputs, URLs, credentials, paths, or audio payloads. `xiaozhi-realtime-parity` now requires all three real streaming profile markers and no profile blockers before returning `xiaozhi_realtime_candidate`; ordered traces without those markers downgrade to `turn_buffered_xiaozhi_candidate` with `xiaozhi_realtime_real_profile_evidence_missing`. Focused app/Gateway tests passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host loopback runtime, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001: Xiaozhi nonblocking ASR commit | Completed host-local control-loop hardening | Plan `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md` scoped the cut. `listen.stop` and VAD auto-stop now start async streaming-ASR commit/final handling instead of blocking the `/v1/xiaozhi` WebSocket read loop. Focused Gateway tests prove abort can be processed while commit remains pending and that a streaming ASR final starts the voice pipeline without calling batch `Transcribe`. This keeps A21 closer to Xiaozhi's responsive control/media state machine while remaining below full PRD acceptance. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001: Stock STT and wake preroll | Completed host-local stock fidelity cut | Plan `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md` scoped source-read workers plus a narrow red/green implementation. A21 now sends stock `stt` before `tts/start` when streaming ASR text exists, and buffers up to five true-idle wake pre-roll Opus frames for the next `listen/start` while preserving cooldown/current-turn `ignored_not_listening` behavior. Focused Gateway tests, full Gateway package tests, and focused app realtime parity/readiness tests passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime acceptance, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-OPUS-INGRESS-QUEUE-001: Xiaozhi Opus ingress queue | Completed host-local control-loop hardening | Plan `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md` scoped the cut. Listening Opus frames now enter a bounded per-session queue before decode/VAD/streaming-ASR append, and `listen.stop` finalization waits asynchronously for queued ingress to catch up. Focused Gateway tests, full Gateway package tests, focused app parity/readiness tests, `git diff --check`, and `make verify` passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime acceptance, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE: Dialogue-first product mode and readiness gate | Active host-local convergence | Plan `docs/plans/2026-06-03-dialogue-first-low-latency-prd-convergence.md` scopes the cut. Product mode surfaces converge on `dialogue` plus `professional`; `dialogue` is the low-latency Xiaozhi/ASR/text/TTS chain and `professional` remains V21-only. Doubao realtime TTS static readiness accepts env-name-only access-token configuration without storing credential/model/voice values. No provider/V21 execution, Gateway restart, firmware, flash, hardware, or audio playback occurred in this transition. |
| T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY: Main public Gateway profile | Active main-edge running | Plan `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md` scopes the cut. Gateway now separates `gateway_profile` from `voice_mode`: valid `A21_PUBLIC_GATEWAY_URL` selects `public_wss` as the main product public path, while `mac_local` remains available for Mac/local-model switching. Product deployment targets trusted `443`/`wss`; IP-only bring-up can use public `http/ws`. `/v1/gateway-profiles`, simulator selection, env `A21_PUBLIC_GATEWAY_URL`, CLI `--public-gateway-url`, and OTA public URL behavior are implemented. The earlier `101.132.117.182` SWAS path is experimental/backup. New ECS `47.103.57.217` is active behind Caddy, returns `ws://47.103.57.217/v1/xiaozhi` from OTA, and passed remote `make verify`; host-only bench remains blocked below PRD because no real provider/V21/hardware execution was injected. |
| T-VOICE-CHAIN-SELECTOR-001-CASCADE-REALTIME-HOTSWITCH: Product voice-chain selector | Completed host-local selector cut | Plan `docs/plans/2026-06-03-voice-chain-product-selector-hot-switch.md` scoped the cut. Gateway now exposes `GET/POST/PUT /v1/voice-chain-profiles` for the frontend to choose `cascade` or `realtime` independently from `voice_mode`, `gateway_profile`, and catalog-only `cloud_voice_profile`. Cascade exposes ASR and LLM choices, recommends StepFun, keeps DeepSeek as fallback, and fixes default TTS at DashScope realtime TTS unless voice clone maps the effective TTS to `voice_clone_cli`. Realtime selection updates the existing `A21_GATEWAY_VOICE_PROVIDER=selected` gate plus `A21_PROVIDER_PRIMARY`. Simulator and device registry now display chain mode, ASR, LLM, effective TTS, realtime provider, and voice/clone profile. Focused Gateway tests passed; no provider execution, deployment, firmware, hardware, or audio playback occurred. |
| T-XIAOZHI-FAST-ACK-CONTINUITY-001: Fast ack cannot block main answer | Completed host-local runtime fix | Public bench against `47.103.57.217` showed the chain reached Opus ingress/decode, streaming ASR append/commit, ASR partial/final, and stock `stt`, but then stopped at `xiaozhi.fast_ack.unavailable` without running the full answer pipeline. Gateway now treats fast ack as optional: if fast-ack TTS fails and the turn is not aborted, the full ASR -> LLM -> TTS answer pipeline continues. Focused Gateway tests, related package tests, `git diff --check`, and `make verify` passed. This is not yet deployed/bench-verified in the public Gateway at the time of this state entry, and it does not claim physical PRD acceptance. |
| T-XIAOZHI-PROVIDER-STATE-MACHINE-REBUILD-001: DashScope realtime TTS lifecycle | Completed host-local provider fix | Plan `docs/plans/2026-06-03-xiaozhi-provider-state-machine-rebuild.md` scoped the official-state-machine rebuild. DashScope realtime TTS now starts the read loop before text append/commit, waits for `session.updated` when available, emits chunks on `response.audio.delta`, sends `session.finish` only after audio completion and successful text commit, and closes without finish on cancellation. Focused DashScope/provider tests, related Gateway/App/Provider tests, `git diff --check`, and `make verify` passed. This is not yet public-bench or physical StackChan acceptance; next action is deploy to `47.103.57.217`, run `xiaozhi-voice-bench --require-product-chain`, then physical wake/dialogue/barge-in proof. |

| T-XIAOZHI-SECOND-READONLY-CROSSCHECK-001: Protocol/endpoint/runtime/strategy cross-check | Completed read-only audit | Four strict read-only workers on HEAD `188b341` returned structured final reports. Protocol thread `019e8ac3-c9f3-7cc3-b8a1-c27cc2748168` confirmed WebSocket/Opus parity is enough for the immediate product lane but MQTT+UDP must remain a planned Xiaozhi transport gap. Endpoint thread `019e8ac3-c9f7-7721-9f6c-1bce1e69af4c` identified custom wake vs official AFE/WakeNet and parked direct-Xiaozhi app lifecycle as the highest product-lane parity risks. Runtime thread `019e8ac3-c9f6-7350-a66e-e51dcdc8109e` identified the host chain blocker: ASR partials do not yet drive LLM/TTS before ASR final/listen stop. Strategy thread `019e8ac3-c9fa-7ed0-8b61-625a418a84c2` recommends incremental A21 convergence using Xiaozhi firmware/protocol/audio-service patterns, with ADR-backed B-lite voice-engine adapter only if phased physical evidence fails. No worker edited files, built, flashed, started services, called providers/V21, or touched audio/hardware. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Audible playback is accepted for the 3x foreground path, relay WAV playback received positive operator feedback, and no-flash self-trigger observation passed; custom wake, device playback timing, barge-in/touch operator proof, and final physical evidence regeneration are still missing | Collect custom wake proof, device playback timing or trusted playback-start evidence, and barge-in/touch proof, then regenerate `xiaozhi-physical-evidence`. |
| T-PRD-001: Declare full PRD physical acceptance | Audio path is accepted, selected-provider readiness is refreshed, and no-flash self-trigger observation passed, but PRD accepted remains false because custom wake, continuous voice pipeline/host voice evidence, V21 professional execution, and final physical evidence regeneration are still pending | Close custom wake proof, collect continuous voice/host voice evidence and V21 professional evidence, regenerate physical evidence, and rerun product readiness. |
| T-HALF-DUPLEX-DIAG-001: Instrumented half-duplex counter acceptance | Latest online diagnostic-counter run `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` is blocked because current stock firmware lacks A21 identity, diagnostic mic-probe capability, available speaker echo fields, and runtime echo counters; this is no longer a contest-path blocker because no-flash self-trigger observation passed | Create a separate guarded diagnostic-capability firmware plan only if machine-verifiable counters are required. |

## Next Candidate Transitions

1. `T-WAKE-003: Zi Yue Phrase Tuning`
   - Current phase: multi-phrase splitter candidate flashed so `紫悦`,
     `紫悦紫悦`, `你好紫悦`, and `小紫悦` are registered as separate MultiNet
     commands instead of one `|`-joined string.
   - Next action: physically retry all four wake variants from idle and capture
     trace/serial evidence; if all fail, add CustomWakeWord init/feed logging
     or tune threshold next.

2. `T-ASR-GREEN-LATENCY-001: Xiaozhi Listen Auto-Stop`
   - Current phase: product app flashed, firmware no-speech timer is present,
     and Gateway now suppresses immediate no-speech placeholder listen restarts
     for stock physical devices.
   - Next action: ask the operator to tap once and confirm no-speech green
     listening exits instead of looping, then test wake from idle with `紫悦`,
     `紫悦紫悦`, `你好紫悦`, and `小紫悦`.

3. `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001: Official Frontend Without Setup Trap`
   - Current phase: no-welcome/direct Xiaozhi path is physically useful, but
     the latest read-only audit says full official AppAvatar/AppDance/AppSetup
     lifecycle parity is not proven because the Mooncake app update path is
     bypassed.
   - Next action: keep current contest package unless welcome regresses; plan a
     separate official-app-lifecycle parity transition after wake/tap validation.

4. `T-AUDIO-BARE-XIAOZHI-PARITY-001: Migrate Bare-Package Audio Advantages`
   - Current phase: read-only parity audit confirms the device side is official
     Xiaozhi Opus/AudioService, while the remaining audio gap is mainly host
     provider/TTS PCM/WAV source quality, chunking, leveling, and re-encoding.
   - Next action: keep 3x accepted gain frozen; plan TTS streaming/mastering A/B
     before changing provider, gain, codec, or firmware.

5. `T-VOICE-CHAIN-EVIDENCE-001: Selected Voice-Chain Readiness Ingress`
   - Current phase: StepFun+Iflytek relay evidence is the best operator
     accepted voice-chain candidate, but existing product readiness provider
     slots correctly accept only route-eligible provider-smoke evidence.
   - Next action: either add a narrow redacted voice-chain evidence ingestion
     surface, or run the existing host voice/continuous pipeline report shape
     with the selected relay chain without changing provider route eligibility.

6. `T-XIAOZHI-HOST-LOCAL-REAL-BASIC-DIALOGUE-SMOKE`
   - Current phase: read-only Gateway/provider audit confirmed `/v1/xiaozhi`
     is the closest product path: real Opus ingress, VAD buffering, voice
     pipeline, and Opus downlink, while ASR/TTS are still not fully streaming.
   - Next action: after an operator-triggered real `/v1/xiaozhi` turn, run
     `xiaozhi-realtime-parity` against the live trace. Do not use
     `/v1/xiaozhi/say`, `xiaozhi-voice-bench`, or `fast-companion-turn` as
     physical parity evidence.

7. `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`
   - Current phase: completed host-side candidate in scoped worker.
   - Next action: keep this evidence below PRD acceptance, then use it as the
     ordering gate for real Sherpa ASR runtime proof, authorized realtime TTS
     execution, and a physical stock `/v1/xiaozhi` trace.

8. `T-XIAOZHI-STREAMING-ASR-001`
   - Current phase: host-side streaming ASR session, partial-to-LLM bridge,
     nonblocking commit, stock `stt` ordering, and true-idle wake pre-roll
     buffering are implemented and covered by Gateway tests. Listening Opus
     frames now pass through a bounded per-session ingress queue so decode,
     VAD, and ASR append do not block the WebSocket control loop. Cancelled
     queue items are now suppressed before audio ingress so old-turn audio
     cannot leak into a fresh listening turn. Full runtime proof is still
     missing because no physical stock `/v1/xiaozhi` trace has shown real
     streaming ASR/LLM/TTS profile markers plus playback.
   - Next action: in an approved runtime/hardware window, collect an
     operator-triggered physical stock `/v1/xiaozhi` trace and run
     `xiaozhi-realtime-parity`; do not promote host-loopback, `/say`,
     voice-bench, or file-boundary evidence.

9. `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`
   - Current phase: public Gateway `cloud_edge` runtime bridge is implemented
     in code. `sherpa_onnx_streaming` remains the local streaming ASR seam,
     `doubao_asr_realtime` and `dashscope_qwen_asr_realtime` are now
     selectable cloud `StreamingASRAdapter` profiles, and
     `doubao_tts_realtime` plus `dashscope_qwen_tts_realtime` are selectable
     `StreamingTTSAdapter` profiles. `A21_XIAOZHI_PRODUCT_CHAIN=cloud_edge`
     now defaults the public product chain away from local Sherpa and toward
     cloud ASR + text-stream LLM + realtime TTS. With `A21_DASHSCOPE_API_KEY`
     present and no explicit ASR/TTS override, the cloud-edge default selects
     DashScope ASR/TTS. With `A21_LAB_STEPFUN_API_KEY` present and no explicit
     text override, the cloud-edge default now selects StepFun before falling
     back to DeepSeek. Static readiness can pass for configured cloud ASR,
     configured StepFun or DeepSeek text stream, and configured realtime TTS,
     while `prd_accepted` remains false.
   - Current deployment note: main ECS `47.103.57.217` is running the verified
     cloud-edge build behind Caddy, and remote readiness currently reports
     `dashscope_qwen_asr_realtime + deepseek + dashscope_qwen_tts_realtime`
     because `/etc/a21/secrets/provider.env` contains no StepFun key/model
     entries. The remaining correction for the intended `stepfun` LLM path is
     root-only injection of `A21_LAB_STEPFUN_API_KEY` and
     `A21_STEPFUN_MODEL=step-1-8k`, then service restart and a fresh provider
     bench. Credentialed live provider execution and physical trace proof are
     still pending.
   - Next action: inject StepFun secrets into the ECS root-only provider env if
     available, rerun `xiaozhi-streaming-provider-readiness` to prove
     `llm_profile=stepfun`, then collect an operator-triggered physical stock
     `/v1/xiaozhi` trace and run `xiaozhi-realtime-parity`.

10. `T-WAKE-004-AFE-VS-CUSTOM-PARITY`
   - Current phase: candidate from read-only audio HAL/wake worker
     `019e8a8a-7263-7b20-94b9-9b2847aa741d`.
   - Next action: plan whether to restore official AFE WakeNet behavior or
     harden custom MultiNet `紫悦`; acceptance must be physical wake from idle,
     not screen tap.

11. `T-STACKCHAN-APP-LIFECYCLE-PARITY`
    - Current phase: candidate from read-only audio HAL/wake worker.
    - Next action: plan how to preserve official StackChan AppLauncher,
      AppAiAgent, AppAvatar, AppDance, AppSetup/Mooncake lifecycle while still
      avoiding the welcome/setup trap.

12. `T-HAL-AUDIO-CONFIG-PARITY`
    - Current phase: candidate from read-only audio HAL/wake worker.
    - Next action: record and, if evidence supports it, align CoreS3/StackChan
      codec constants and init order such as ES7210 input gain, AFE/AEC/VAD,
      AW88298 output, and runtime MCP/NVS volume behavior.
