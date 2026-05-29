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
- Displays audio input state, frame count, RMS, and playback state
- Plays a short local mock playback tick when a speaking control event arrives and playback is enabled
- Displays expression state, mode, trace ID, session ID, and event log
- Displays office visibility badges for `PRIVATE`, `PUBLIC`, `PRO`, `MUTED`, `LISTENING`, screen state, and output state
- Displays Gateway `/v1/devices` registry status for the simulator device
- Displays Gateway `/v1/traces` waterfall events for the active trace
- Displays professional evidence, confidence, screen cards, and follow-ups returned by the V21 adapter mock path
- Shows a simple StackChan face state for `idle`, `listening`, `thinking`, `speaking`, `interrupted`, and `error`

## Boundaries

The simulator is a development surface, not the final A21 device UI. It does not play gateway audio and does not emulate firmware timing or servo/RGB behavior yet.

Current microphone support is a simulator aid only: it uses the browser microphone API when the user grants permission, encodes short PCM-style frames, and sends them through the existing A21 `audio.frame` envelope. It is not calibrated ASR input, does not prove real StackChan microphone capture, and does not prove echo cancellation.

Current playback support is local mock feedback only. It does not decode or play real Gateway/Provider TTS audio chunks yet.

The simulator uses a synthetic firmware commit `0000000` so Gateway identity validation can be exercised without pretending the simulator is a real packaged firmware artifact.

Future simulator work should add:

- richer latency waterfall segments and p50/p95 summaries
- disconnect and reconnect scenarios
- visible barge-in timing markers
- wav upload fixtures
- real downlink audio playback once the Gateway emits audio chunks
