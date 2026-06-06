# StackChan Official Xiaozhi Natural Dialogue Repair Plan

Status: active repair plan for the internal-test4 recovery line; first Gateway
transition implemented locally.
Created: 2026-06-06.

## Goal

Correct A21's current product direction by aligning it with the actual M5Stack
StackChan frontend/lifecycle and 78/xiaozhi-esp32 voice state machine, then
repair the natural dialogue and body behavior without weakening the
internal-test4 floor.

This plan is a control-tower repair surface. It is not a flash instruction and
does not authorize PMIC register experiments, NVS writes, provider key movement,
or generic firmware lanes.

## Source Baseline Read

Clean official StackChan export:

- `/tmp/a21-stackchan-official-clean/firmware/main/main.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/apps/app_launcher/app_launcher.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/apps/app_ai_agent/app_ai_agent.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/apps/app_avatar/app_avatar.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/apps/app_setup/app_setup.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/hal/hal.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/hal/hal_ws_avatar.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/hal/hal_mcp.cpp`
- `/tmp/a21-stackchan-official-clean/firmware/main/hal/board/stackchan.cc`

Clean official-compatible Xiaozhi export:

- `/tmp/a21-stackchan-official-clean/firmware/xiaozhi-esp32/main/application.cc`
- `/tmp/a21-stackchan-official-clean/firmware/xiaozhi-esp32/main/protocols/protocol.cc`
- `/tmp/a21-stackchan-official-clean/firmware/xiaozhi-esp32/main/protocols/websocket_protocol.cc`
- `/tmp/a21-stackchan-official-clean/firmware/xiaozhi-esp32/main/mcp_server.cc`

Current A21 Gateway source:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/official_stackchan_test.go`

## Official State Machine Facts

### Boot and Frontend

Official StackChan does not boot directly into voice. It:

1. Initializes HAL and NVS.
2. Installs Mooncake apps:
   `AppLauncher`, `AppAiAgent`, `AppAvatar`, `AppSetup`, `AppDance`, and
   others.
3. Runs `GetMooncake().update()` until `GetHAL().isXiaozhiStartRequested()`.
4. Only after AI.AGENT is opened does it uninstall all Mooncake apps, destroy
   Mooncake, and call `GetHAL().startXiaozhi()`.

Implication for A21:

- "Official before AI.AGENT, A21 after AI.AGENT" is the official-compatible
  product path.
- Reintroducing boot-time auto-start is a regression unless an ADR explicitly
  changes the frontend product shape.
- Manual builds from the dirty local official tree containing X21 auto-start
  code are unsafe.

### AppAvatar and Body Relay

`AppAvatar` owns the official `/stackChan/ws` avatar/motion/camera/call/text
relay. It starts `WebsocketAvatarWorker` as a Mooncake ability and maps binary
packets to:

- `ControlAvatar`
- `ControlMotion`
- `DanceSequence`
- camera stream start/stop
- call and text-message surfaces

When AI.AGENT starts Xiaozhi, official `main.cpp` uninstalls all apps. Therefore
the Avatar app's relay and Home lifecycle are not a reliable runtime body
surface after AI.AGENT.

Implication for A21:

- In AI.AGENT/Xiaozhi runtime, body behavior must be provided by the Xiaozhi
  runtime's local StackChan bridge and MCP tools.
- Gateway `/stackChan/ws` remains valid for official Avatar mode and diagnostics
  but must not be the only product body path after AI.AGENT.

### Xiaozhi Voice State

Xiaozhi's mature voice state machine is:

- `Idle`: wake word enabled, voice processing off.
- `Connecting`: opens audio channel.
- `Listening`: sends `listen.start`, enables voice processing, disables wake
  word unless AFE listening wake-word is configured.
- `Speaking`: disables voice processing unless realtime mode is active; only
  AFE wake-word may remain available.
- `tts.stop`: if manual-stop mode, returns to `Idle`; otherwise returns to
  `Listening`.

Implication for A21:

- Do not globally force manual-stop or break normal official auto-listen just
  to hide a feedback loop.
- The product bug is likely at the Gateway/device boundary after playback:
  speaker tail, stale audio, fallback/placeholder speech, or late downlink must
  not become a fresh user turn.

### Power-Key Boundary

If no-USB PWRKEY produces only a screen flash/red LED flash and no boot/app log,
the failing boundary is before Gateway, provider, AI.AGENT, or voice state can
participate. The current product line must preserve the stock PMIC startup
parity restored in HEAD and must not reintroduce speculative AXP2101 register
writes without a stock/A21 A/B trace or hardware measurement.

## A21 Target Natural Dialogue Flow

### Frontend Phase

State: `OfficialHome`.

- Use official Launcher/Setup/Home behavior.
- Official Wi-Fi/config/account screens remain the first-run path.
- AI.AGENT is the user-selected entry into A21 runtime.
- A21 does not steal boot before this point.

### A21 Runtime Phase

State: `A21Idle`.

- Wake word enabled.
- Voice processing off.
- Face/body idle motion remains alive through StackChan local update task.
- Touch feedback is local and immediate, but a casual screen/top tap does not
  automatically start a voice turn.

State: `A21WakeAck`.

- Wake word detected.
- Device plays a short popup/attention sound and moves/LEDs lightly.
- Gateway opens/continues `/v1/xiaozhi` channel.

State: `A21Listening`.

- Voice processing on.
- User speech starts only after new VAD speech evidence.
- If no speech arrives, return to idle without speaking a fallback line that can
  feed back into the microphone.

State: `A21Thinking`.

- Voice processing off or guarded.
- Optional fast ack after the configured delay only if provider latency exceeds
  the threshold and the ack is known not to trigger post-TTS self-listening.

State: `A21Speaking`.

- Downlink Opus is paced.
- Device playback start/stop_done events are recorded when negotiated.
- Touch/wake barge-in aborts playback and arms a short input drain cooldown.

State: `A21PostPlaybackGrace`.

- Official Xiaozhi may move the device UI back to Listening after normal
  `tts.stop`.
- Gateway must not start a new answer from immediate tail audio.
- A fresh turn requires a new VAD speech_start after playback stop_done plus a
  short guard window, or an explicit wake/touch/manual start outside the guard.

## Repair Transitions

### T-A21-NATURAL-DIALOGUE-POST-PLAYBACK-GUARD-001

Current state:

- Current Gateway intentionally accepts stock physical auto-listen after normal
  `voice_pipeline_answer_completed`.
- It suppresses input after host-say, placeholder/no-speech, touch abort,
  degraded, error, and unavailable paths, but not after a normal completed
  answer.

Target state:

- Official auto-listen UI is preserved.
- Normal completed-answer `tts.stop` arms a product-only post-playback guard for
  stock physical MAC clients when playback events are available.
- During the guard, Gateway may accept `listen.start` but must not form a new
  voice pipeline turn from tail frames until it observes fresh speech after
  playback `stop_done + guard_ms`.
- Trace markers distinguish `post_playback_guard_armed`,
  `post_playback_tail_suppressed`, and `post_playback_fresh_speech_accepted`.

Acceptance:

- Existing fallback/no-speech/degraded suppression tests still pass.
- New tests prove normal auto-listen is not destroyed, but immediate post-answer
  tail audio does not create a second answer.
- Physical acceptance requires a stock `/v1/xiaozhi` trace where the device
  answers once, returns to listening, and remains silent without self-loop.

Forbidden:

- Do not force all normal answers to manual stop.
- Do not disable official wake/listen semantics globally.
- Do not hide the loop by suppressing every post-answer follow-up.

### T-A21-XIAOZHI-NATIVE-BODY-SEQUENCE-001

Current state:

- A21 has local touch RGB/servo/sound feedback and Gateway MCP fallback.
- The feedback is one-step and lower fidelity than official Avatar/Dance
  sequences.

Target state:

- Add a Xiaozhi-runtime local body sequence helper for StackChan:
  attention, listening, thinking, speaking, screen_tap, top_tap,
  top_swipe_forward, top_swipe_backward, and barge_in.
- Use official primitives already present in the runtime:
  `GetStackChan().motion().moveWithSpeed`, `goHome`, neon light color, vibration
  sound, and MCP tools.
- Motion should be multi-step where needed: move, hold, return/home, with
  bounded amplitude and speed.

Acceptance:

- Overlay tests prove the helper is present and touch handlers use it.
- No official AppAvatar relay is required for AI.AGENT body feedback.
- Physical acceptance compares visible amplitude and responsiveness against the
  official Servo/RGB test path.

Forbidden:

- Do not depend on `/stackChan/ws` being connected in AI.AGENT.
- Do not block audio state transitions on body animation completion.
- Do not move beyond official servo limits.

### T-A21-OFFICIAL-FRONTEND-MODE-ENTRY-ADR-001

Current state:

- Roleplay/professional mode switching exists in Gateway global state and HTTP
  control surfaces, not as a user-visible official frontend selector.

Target state:

- Produce an ADR for the least risky UX:
  keep official Home/Setup, add/repurpose a mode entry only before entering
  AI.AGENT, and keep A21 mode state in Gateway.
- Do not attempt to return from Xiaozhi to Home in this transition because
  official Xiaozhi app is designed as a never-return path after Mooncake
  teardown.

Acceptance:

- ADR states whether mode selection is:
  Gateway web/app only,
  official Setup menu extension,
  or a future Mooncake-native A21 app.
- No code changes until the UX route is chosen.

### T-A21-POWER-KEY-STOCK-AB-EVIDENCE-001

Current state:

- No-USB power behavior remains physically unresolved.
- Current HEAD restored stock PMIC startup parity and forbids old speculative
  writes.

Target state:

- Collect a timed stock-official versus A21 product A/B matrix:
  USB boot, no-USB short press, no-USB long press, serial/log reachability,
  PMIC snapshot when app is reached, and battery/external power state.

Acceptance:

- If stock official fails the same way, route to hardware/battery/charging
  diagnosis.
- If stock official succeeds and A21 fails before app logs, isolate bootloader,
  partition, sdkconfig, or PMIC init delta.
- If both reach app logs but A21 later fails, only then inspect app-level power
  save/shutdown state.

Forbidden:

- No PMIC register write restoration without this A/B evidence.
- No Gateway/provider changes for no-USB PWRKEY symptoms.

## Immediate Recommended Order

1. Implement and test `T-A21-NATURAL-DIALOGUE-POST-PLAYBACK-GUARD-001` in
   Gateway. Completed locally on 2026-06-06; physical evidence is still
   pending.
2. Implement `T-A21-XIAOZHI-NATIVE-BODY-SEQUENCE-001` in the official-compatible
   overlay with focused build tests.
3. Only after those pass, build the product artifact and ask for one foreground
   physical run.
4. Run `T-A21-POWER-KEY-STOCK-AB-EVIDENCE-001` separately; do not mix it with
   voice/body code changes.

## 2026-06-06 Local Implementation Update

Implemented:

- `T-A21-NATURAL-DIALOGUE-POST-PLAYBACK-GUARD-001`.

Code behavior:

- Normal completed-answer `tts.stop` now arms a product-only post-playback guard
  only when the session is a stock physical hardware MAC and product
  `playback_events` allowance is negotiated.
- Device `playback=stop_done` extends that guard so it is anchored to actual
  playback completion.
- `listen.start` is still accepted, preserving official Xiaozhi auto-listen UI.
- Opus frames inside the guard are dropped before decode/VAD/ASR and traced as
  post-playback tail suppression.
- After the guard releases, the first fresh VAD speech is traced as accepted and
  can form the next turn normally.

Verification:

- Focused Gateway post-playback/auto-listen/no-speech/degraded/touch-abort tests
  passed.
- `git diff --check` passed.
- `GOMAXPROCS=2 make verify` passed.

Still pending:

- No ECS deploy, product flash, or physical trace occurred in this transition.
- `T-A21-XIAOZHI-NATIVE-BODY-SEQUENCE-001` remains the next code transition for
  touch/body amplitude and native StackChan motion parity.

## 2026-06-06 Local Body Sequence Implementation Update

Implemented:

- `T-A21-XIAOZHI-NATIVE-BODY-SEQUENCE-001`.

Code behavior:

- The official-compatible Xiaozhi overlay now contains a local
  `A21BodySequenceStep`/`A21BodySequence` helper for attention, listening,
  thinking, speaking, screen tap, top tap, top swipe forward, top swipe
  backward, and barge-in names.
- Product touch handlers now start the sequence helper with
  `xTaskCreatePinnedToCore(a21_body_sequence_task, ...)`, so touch RGB/servo
  animation does not block audio state transitions.
- Touch/barge feedback uses StackChan-native neon RGB,
  `GetStackChan().motion().moveWithSpeed`, `motion.goHome`, and the existing
  vibration sound.
- Swipe/barge amplitudes are now visibly larger while staying within the
  official StackChan servo range used by setup/diagnostic paths.

Verification:

- Focused official overlay tests passed.
- `git diff --check` passed before docs update.
- Tail whitespace scan passed before docs update.
- `GOMAXPROCS=2 make verify` passed.
- `make a21-stackchan-official-xiaozhi-compatible-build` passed. Product app
  artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`;
  SHA-256:
  `50d1c963ae890837546199a96d44554f0226d395aae443091f418f48c2009e82`.

Still pending:

- No product flash or physical body acceptance occurred in this transition.
- The helper has state names for attention/listening/thinking/speaking, but
  this transition only wires touch/barge. Voice lifecycle body hooks should be
  handled separately so the already verified voice guard is not disturbed.
- No-USB power-key behavior remains unresolved and requires the separate
  stock/A21 A/B evidence transition.

## 2026-06-06 Endpoint Voice Feel Implementation Update

Implemented:

- Follow-up endpoint repair for wake sensitivity, post-TTS breath, RGB
  ownership, and touch/barge-in ordering.

Code behavior:

- Product custom wake threshold is now `16`, down from `20`.
- Normal completed TTS stop arms a 4200 ms post-TTS breath window when playback
  events are negotiated. Fresh VAD speech cancels the timer; silence returns to
  Idle through the normal stop-listening path so wake-word detection is
  restored.
- TTS start, VAD speech, and abort/barge/wake stop paths cancel or skip the
  breath window.
- Speaking-state screen/top barge-in now sends the touch/abort path before local
  RGB/servo feedback.
- Idle screen tap starts A21 listening through `StartA21TouchListening` using
  the default listen mode instead of the manual-stop handler.
- Local body sequences now use lower touch RGB brightness and call the official
  LED state refresh after sequences while Listening/Speaking.

Verification:

- Focused official overlay tests passed.
- `GOMAXPROCS=2 make verify` passed.
- `git diff --check` passed.
- Tail whitespace scan passed.
- `make a21-stackchan-official-xiaozhi-compatible-build` passed. Product app
  artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`;
  SHA-256:
  `67e29577ab8910b1c3377c414f0f1d493698bb9530d28cc871aae5cc04ee94b7`.

Still pending:

- No product flash or physical acceptance occurred in this transition.
- Next physical run should verify first wake, post-TTS re-wake after silence,
  continuous follow-up inside the 4200 ms breath, RGB/ASR color ownership,
  screen tap start latency, and screen/top barge-in latency.

## Summary Format For Workers

- Transition:
- What changed:
- Files changed:
- Tests run and results:
- Runtime or physical evidence:
- Deviations from plan:
- Remaining issues:
- Next suggested action:
- Forbidden actions avoided:
