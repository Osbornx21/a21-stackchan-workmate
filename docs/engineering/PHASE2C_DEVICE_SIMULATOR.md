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
- Sends mock `audio.frame` envelopes
- Displays expression state, mode, trace ID, session ID, and event log
- Shows a simple StackChan face state for `idle`, `listening`, `thinking`, `speaking`, `interrupted`, and `error`

## Boundaries

The simulator is a development surface, not the final A21 device UI. It does not use the browser microphone, does not play gateway audio, and does not emulate firmware timing or servo/RGB behavior yet.

Future simulator work should add:

- microphone capture behind an explicit permission prompt
- mock audio playback
- separate public/private/professional mode indicators
- latency waterfall panel
- disconnect and reconnect scenarios
- visible barge-in timing markers
