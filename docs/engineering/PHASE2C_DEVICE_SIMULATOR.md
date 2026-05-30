# A21 Phase 2C Device Simulator

## Purpose

Phase 2C makes A21 usable without physical StackChan hardware by serving a built-in browser simulator from the gateway.

## Run

```bash
make gateway
```

Open:

```text
http://127.0.0.1:21080/simulator
```

## Current Capabilities

- Connects to `/ws/control`
- Connects to `/ws/audio`
- Sends `mock.turn` device events
- Sends `interrupt` device events
- Sends simulated A21 firmware identity in device events
- Sends mock `audio.frame` envelopes
- Sends mock audio bursts over `/ws/audio`
- Can request browser microphone permission and stream short PCM mock frames over `/ws/audio`
- Receives Gateway `audio.playback.chunk` downlink envelopes and displays downlink count, buffered chunk count, active `stream_id`, and scheduled PCM chunk count
- Displays audio input state, frame count, RMS, playback state, and downlink buffer state
- Decodes and schedules Gateway mock `pcm_s16le` playback chunks through WebAudio when playback is enabled
- Stops scheduled WebAudio sources when an interrupt/error/non-speaking state clears playback
- Plays a short local mock playback tick when a speaking control event arrives and playback is enabled
- Displays expression state, mode, trace ID, session ID, and event log
- Displays office visibility badges for `PRIVATE`, `PUBLIC`, `PRO`, `MUTED`, `LISTENING`, screen state, and output state
- Displays Gateway `/v1/devices` registry status and connection freshness for the simulator device
- Displays Gateway `/v1/traces` waterfall events for the active trace
- Displays Gateway trace latency summary for audio frame to playback, V21 first result, barge-in stop, and provider commit to first audio
- Displays professional evidence, confidence, screen cards, and follow-ups returned by the V21 adapter mock path
- Shows a simple StackChan face state for `idle`, `listening`, `thinking`, `speaking`, `interrupted`, and `error`

## Boundaries

The simulator is a development surface, not the final A21 device UI. It does not emulate firmware timing, calibrated speaker output, or servo/RGB hardware behavior yet.

Current microphone support is a simulator aid only: it uses the browser microphone API when the user grants permission, encodes short PCM-style frames, and sends them through the existing A21 `audio.frame` envelope. It is not calibrated ASR input, does not prove real StackChan microphone capture, and does not prove echo cancellation.

Current playback support decodes Gateway mock `pcm_s16le` downlink chunks and schedules them through WebAudio so the simulator can exercise a real client-side playback queue and barge-in cancellation boundary. The Gateway still emits deterministic 20 ms silence chunks, so this does not prove real Provider TTS audio, real StackChan speaker output, mouth sync, full-duplex capture, or acoustic echo cancellation.

The simulator uses a synthetic firmware commit `0000000` so Gateway identity validation can be exercised without pretending the simulator is a real packaged firmware artifact.

The simulator's Device Registry panel now mirrors Gateway acceptance fields: device identity, firmware identity, connection status, device age, current mode, and current expression. This is deliberately shallow state. It proves the operator can see whether A21 is live, stale, listening, speaking, professional, muted, private, public, local, or in error without storing the user's utterance or V21 evidence body in the registry.

Future simulator work should add:

- p50/p95 summaries across repeated turns
- disconnect and reconnect scenarios
- visible barge-in timing markers
- wav upload fixtures
- real provider TTS fixture playback once the Gateway emits non-silent provider chunks
- simulator-visible active playback source count and stop latency markers
