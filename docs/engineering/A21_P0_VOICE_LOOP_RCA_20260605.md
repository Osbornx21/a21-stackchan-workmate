# A21 P0 Voice Loop RCA

Status: in progress.
Created: 2026-06-05.
Transition: `T-A21-P0-VOICE-LOOP-RCA-001`.

Internal test 4 remains the recovery baseline. This report does not authorize a
provider switch, prompt rewrite, firmware flash, or NVS write.

## Symptom

User-observed product behavior after internal test 4:

- Wake-word sensitivity is worse than expected.
- The first answer can feel delayed after wake.
- After one response, the device can fall into repeated self-answering.

Separate symptoms, not solved by this RCA:

- No-USB physical power-key cold boot.
- Touch/RGB/vibration/servo amplitude parity after entering `AI.AGENT`.
- Official mobile app device-data parsing.

## Evidence Captured So Far

- `GOMAXPROCS=2 go test -race ./internal/providers -count=1` initially exposed
  a data race in the provider realtime test fake, not in a proven product
  runtime path.
- The race was between concurrent `ReadJSON` and `WriteJSON` access to the
  fake connection's `timeline`, `messages`, and `readIndex` state.
- The fake connection now serializes those fields with a mutex, so provider
  realtime tests can be used as credible RCA evidence again.
- After the test-fake fix:
  - `GOMAXPROCS=2 go test -race ./internal/providers -count=1` passed.
  - `GOMAXPROCS=2 go test ./internal/gateway -run 'Xiaozhi|OfficialStackChan|PowerLifecycle|Playback|Barge|Listen' -count=1`
    passed.

## Current Boundary

The field self-loop is not yet proven to be a provider implementation bug. The
current narrowed boundaries are:

- provider realtime test harness is race-clean after the fake fix;
- current focused Gateway Xiaozhi/official StackChan state-machine tests pass;
- physical product runtime evidence is still required to determine whether the
  loop starts from firmware audio pickup, Gateway listen-state restart,
  keepalive/control-channel behavior, provider realtime streaming, or local
  fallback/placeholder text.

## Forbidden Fixes

- Do not switch providers to hide the symptom.
- Do not rewrite roleplay/personality prompts as a voice-loop fix.
- Do not tune wake-word thresholds without a physical or log-based before/after
  reproduction.
- Do not change normal official `tts.stop` auto-listen behavior unless a trace
  proves it is the loop origin.
- Do not flash a new product artifact until the exact single-variable fix,
  rollback path, and acceptance script are named.

## Next Evidence Required

Capture one product session with these fields aligned by `trace_id`,
`session_id`, and `device_id`:

1. wake event or manual `AI.AGENT` start;
2. `listen.start` and listening mode;
3. first binary audio frame after wake;
4. provider ASR partial/final timing;
5. provider TTS first-byte and `tts.stop` reason;
6. playback-start/playback-stop extension events;
7. any immediately repeated `listen.start` or binary audio after `tts.stop`;
8. whether the repeated answer text comes from provider output or local
   fallback/placeholder handling.

Acceptance for this RCA:

- If trace shows audio from the speaker is being captured as user speech, open
  a firmware/audio gating transition.
- If trace shows Gateway arms the wrong listen state after non-normal output,
  open one minimal Gateway state-machine fix.
- If trace shows provider realtime emits repeated text without new input, open
  one provider adapter fix with race-safe regression coverage.
- If trace shows normal official auto-listen with no repeated audio/text, close
  this as non-reproduced and move to physical wake sensitivity evidence.
