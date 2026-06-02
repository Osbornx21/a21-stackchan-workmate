# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-03.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-STACKCHAN-COMPATIBLE-ZI-YUE-TUNED-FLASHED-ASR-LATENCY-HOTFIX-RUNNING`

Active child transitions:

- `T-WAKE-003-ZI-YUE-PHRASE-TUNING`
- `T-ASR-GREEN-LATENCY-001-XIAOZHI-LISTEN-AUTO-STOP`
- `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`
- `T-AUDIO-BARE-XIAOZHI-PARITY-001`
- `T-VOICE-CHAIN-EVIDENCE-001-SELECTED-VOICE-CHAIN-READINESS-INGRESS`
- `T-COSYVOICE-5080-LOCAL-CLONE-CANDIDATE-CHECK`

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
| Firmware candidate | `S5H-STACKCHAN-COMPATIBLE-APP-PRELOAD-QUIET-SOCKET-CANDIDATE` | Previous direct-start product flash recovered from the welcome/setup trap but skipped official app preload. Current unbuilt overlay candidate changes the product lane to install official StackChan apps first, set codec volume `92`, immediately `requestXiaozhiStart()`, add `A21_STACKCHAN_KEEP_CONTROL_CHANNEL`, open the stock Xiaozhi WebSocket quietly while idle, show `紫悦` connecting/ready copy, and add a 7s no-speech device-side listen timeout. Focused app tests and ordinary `git apply --check` pass; build/flash/physical proof are still pending. | `S5I-APP-PRELOAD-NO-WELCOME-FLASHED-WAKE-READY` |
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

- `S-DIRECT-XIAOZHI-START-FLASHED-WAKE-SOCKET-COUPLED`

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
  `10fb2896d6096ab12beb81519166f0cb790226894e6d9451222904d2ff9f65f0`.
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

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Audible playback is accepted for the 3x foreground path, relay WAV playback received positive operator feedback, and no-flash self-trigger observation passed; custom wake, device playback timing, barge-in/touch operator proof, and final physical evidence regeneration are still missing | Collect custom wake proof, device playback timing or trusted playback-start evidence, and barge-in/touch proof, then regenerate `xiaozhi-physical-evidence`. |
| T-PRD-001: Declare full PRD physical acceptance | Audio path is accepted, selected-provider readiness is refreshed, and no-flash self-trigger observation passed, but PRD accepted remains false because custom wake, continuous voice pipeline/host voice evidence, V21 professional execution, and final physical evidence regeneration are still pending | Close custom wake proof, collect continuous voice/host voice evidence and V21 professional evidence, regenerate physical evidence, and rerun product readiness. |
| T-HALF-DUPLEX-DIAG-001: Instrumented half-duplex counter acceptance | Latest online diagnostic-counter run `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` is blocked because current stock firmware lacks A21 identity, diagnostic mic-probe capability, available speaker echo fields, and runtime echo counters; this is no longer a contest-path blocker because no-flash self-trigger observation passed | Create a separate guarded diagnostic-capability firmware plan only if machine-verifiable counters are required. |

## Next Candidate Transitions

1. `T-WAKE-003: Zi Yue Phrase Tuning`
   - Current phase: `紫悦` physical proof failed for the two-syllable `zi yue`
     command.
   - Next action: build and guarded-flash the tuned official-compatible product
     app, then physically retry `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦`.

2. `T-ASR-GREEN-LATENCY-001: Xiaozhi Listen Auto-Stop`
   - Current phase: Gateway hotfix is running on `21081` with
     `A21_XIAOZHI_LISTEN_MAX_MS=7000`.
   - Next action: ask the operator to test green-light wait and capture the
     resulting trace.

3. `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001: Official Frontend Without Setup Trap`
   - Current phase: active overlay/test candidate preloads official apps,
     requests Xiaozhi immediately, adds quiet idle socket readiness, and adds
     `紫悦` not-connected feedback plus no-speech listen timeout.
   - Next action: run `make verify`, build the official-compatible product app,
     inspect `sdkconfig.json`, then guarded-flash from a clean worktree.

4. `T-AUDIO-BARE-XIAOZHI-PARITY-001: Migrate Bare-Package Audio Advantages`
   - Current phase: active worker dispatched for a bounded parity checklist;
     no product audio behavior has been changed in this main-thread slice.
   - Next action: integrate the worker checklist only if it is read-only/docs
     safe, then plan any A/B before changing provider, gain, or codec behavior.

5. `T-VOICE-CHAIN-EVIDENCE-001: Selected Voice-Chain Readiness Ingress`
   - Current phase: StepFun+Iflytek relay evidence is the best operator
     accepted voice-chain candidate, but existing product readiness provider
     slots correctly accept only route-eligible provider-smoke evidence.
   - Next action: either add a narrow redacted voice-chain evidence ingestion
     surface, or run the existing host voice/continuous pipeline report shape
     with the selected relay chain without changing provider route eligibility.
