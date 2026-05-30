# A21 Development Mainline

Status: v0.4, aligned with `docs/prd/A21_PRD.md`.

## Source Of Truth

`docs/prd/A21_PRD.md` is the current product-level directive for A21. It supersedes older roadmap ordering where it conflicts with product priority, but it does not repeal verified local engineering facts:

- Go-first A21 Core/Gateway remains the approved foundation.
- StackChan remains a thin device for sensing and expression.
- Firmware build, package, artifact, and guarded flashing discipline remain mandatory.
- A21/X21/V21 naming, process, port, env, log, report, and artifact isolation remains mandatory.
- Provider keys never enter firmware, docs output, reports, traces, logs, screenshots, Git, or device messages.
- External latency labs and procurement experiments stay outside the A21 mainline; only redacted conclusions and profile/env names return.

## Current Mainline

The active product-engineering mainline is now:

```text
StackChan embodied device foundation
  + Provider Spine / text streaming hot plug
  + Local audio front-end boundaries
  -> Fast Companion Hybrid Lane
  -> Professional V21 evidence lane
  -> Controlled realtime voice lane
  -> AgentTask bridge
  -> Product polish and office scenarios
```

This means Provider Spine and mainland-latency-controlled fast companion behavior remain the next software pressure, while the already-built StackChan firmware/link discipline remains a non-regression base.

Current active slice: finish the StackChan hardware mainline before moving deeper into Provider Spine. The slice is deliberately non-destructive: use `stackchan-hardware-mainline` to read Gateway device capability declarations, preserve the no-flash discipline, and lock the next hardware tracks into an ordered diagnostic path.

## Priority Order

### P0. Provider Spine / Text Stream Hot Plug

Goal: make provider selection profile-driven and hot-pluggable without changing Gateway business logic.

Required shape:

- provider families: `text_stream`, `voice_realtime`, `voice_hybrid`, `agent_task`, `local_audio`;
- built-in profile coverage for `siliconflow`, `deepseek`, `stepfun`, `bailian_dashscope`, `moonshot`, `volcengine_ark`, `local_ollama`, `local_vllm`, `openai_realtime`, `doubao_realtime`, `doubao_tts_realtime`, `hermes_agent`, and `mimo_agent`;
- Baidu and Huawei are blocked provider names, not candidates;
- local overrides through `A21_PROVIDER_PROFILES_PATH` are allowed only after secret, legacy-name, and blocked-provider validation;
- provider smoke and doctor output show env names, host, status, timing, and redacted findings only.

### P0. Fast Companion Hybrid Lane

Goal: achieve low perceived latency through local audio front-end plus streaming text provider plus local or streaming TTS.

Default lane:

```text
StackChan mic
  -> Gateway VAD/barge-in
  -> local ASR
  -> OpenAI-compatible streaming text provider
  -> local/streaming TTS
  -> StackChan speaker/screen/servo/RGB/touch state
```

Acceptance target:

- first audible response P50 < 900 ms and P95 < 1500 ms for the candidate chain;
- barge-in local stop/cancel P95 < 300 ms;
- first byte, first content, TTS first audio, downlink first frame, and device playback start are recorded.

### P0. StackChan Physical Acceptance

Goal: keep the embodied device foundation real while Provider Spine advances.

Already preserved:

- A21 firmware identity and guarded flash discipline;
- screen expression, RGB, Y-axis servo, touch semantics, speaker buffer, audio frame uplink, runtime echo, and full hardware capability declaration.

Next accepted work must improve or protect actual product behavior. Planned hardware such as camera, IMU, ambient light, proximity, battery, NFC, infrared, and second servo axis stays `planned_*` until implementation and evidence prove it is not worse than original StackChan behavior.

Active hardware order:

1. IMU read-only diagnostic probe.
2. Ambient light read-only diagnostic probe.
3. Proximity read-only diagnostic probe.
4. Battery read-only diagnostic probe.
5. Second servo axis safety spike.
6. Camera privacy-safe vision spike.
7. NFC explicit opt-in interaction spike.
8. Infrared explicit opt-in interaction spike.

Acceptance entry:

```bash
go run ./cmd/a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
```

The command is a planning and diagnostic gate only. It must not flash firmware, delete artifacts, or promote a planned capability to `available`.

Current progress: the IMU track now has a guarded native runtime boundary and isolated `a21_stackchan_cores3_imu_probe` build lane. It remains diagnostic-only until physical evidence proves posture and motion telemetry improve A21's embodied behavior.

Provider spine progress: DeepSeek is now the first P0 `text_stream` profile with an OpenAI-compatible streaming smoke/parser boundary. `provider-smoke --provider deepseek --stream --repeat N` records redacted first-byte, first-content, total-duration, fallback, trace, and metric evidence while preserving the existing no-network-without-`--execute` rule.

Local audio progress: macOS `say + afconvert` is now the first local TTS boundary for loopback work. `local-tts-smoke` produces redacted evidence and a 16 kHz mono PCM WAV without provider keys, global proxy dependence, firmware changes, or final-voice claims. `local-voice-loopback` now stitches mock VAD, mock ASR, mock OpenAI-compatible text-stream parsing, local TTS, and existing Gateway barge-in bench into one host-side redacted timing receipt.

### P0. Professional V21 Evidence Lane

Goal: keep professional mode auditable and separate from opaque realtime voice.

Rules:

- professional mode calls only the A21 V21 adapter contract;
- companion/private/roleplay content does not automatically become V21 retrieval context;
- public mode does not externalize sensitive evidence;
- every answer keeps `trace_id`, source evidence, confidence, screen cards, and fallback behavior.

### P1. Controlled Realtime Voice Lane

Goal: keep speech-to-speech providers as controlled optional capability, not default architecture.

Rules:

- provider events are converted to A21 provider-neutral events before Gateway or firmware sees them;
- physical StackChan requires one-shot `realtime_on_next_speech` arming before paid realtime provider startup;
- professional mode is explicitly rejected by the realtime lane;
- fixture smoke comes before real network execution.

### P1. AgentTask Bridge

Goal: use mature agent frameworks for long tasks without letting them own A21.

Rules:

- `hermes_agent` and `mimo_agent` are `agent_task` profiles only;
- agent output maps back into A21 semantic events;
- agents cannot write firmware commands, provider env, V21 internals, or Gateway runtime state directly;
- the lane is disabled until explicit `A21_AGENT_PROVIDER_PRIMARY` and smoke pass.

### P2. Product Polish

Goal: turn the working system into a desk workmate people want to talk to.

Work includes:

- personality files instead of one giant prompt;
- office scenario playbooks;
- public/private/focus transitions;
- roleplay and co-creation flows;
- failure copy that preserves A21's presence.

## Non-Regression Rule

No PRD-driven task may regress:

- guarded firmware package/flash flow;
- StackChan capability reporting;
- LAN/proxy directness checks;
- provider report redaction;
- A21 namespace audit;
- V21 adapter boundary;
- Gateway default mock safety.

If a new PRD slice requires changing one of these, it needs an ADR before code.
