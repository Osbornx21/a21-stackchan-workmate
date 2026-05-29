# A21 Protocol

## Current Phase 1 Contract

The current Go protocol package defines a minimal device envelope:

```go
const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
)

type Envelope struct {
	Protocol string `json:"protocol"`
	DeviceID string `json:"device_id"`
	Kind     Kind   `json:"kind"`
	Seq      uint64 `json:"seq"`
}
```

This is intentionally small. It establishes the A21 namespace and the first wire families without pretending to solve all device media concerns.

## Protocol Rules

- Every message family must be versioned.
- Device messages must include `device_id`.
- Runtime paths must be ready to carry `trace_id` and `session_id` before Phase 2 gateway work.
- Audio and control traffic may share an envelope but should have separate transport surfaces when real-time behavior requires it.
- StackChan firmware should receive semantic commands, not provider-specific events.
- V21 evidence must not be encoded as generic chat text when in professional mode; it needs explicit evidence/card fields in later contracts.

## Future Envelope Candidate

Phase 2 may expand the envelope to:

```ts
type A21Envelope<TType extends string, TPayload> = {
  version: "a21.protocol.v1";
  type: TType;
  trace_id: string;
  session_id: string;
  device_id: string;
  seq: number;
  sent_at_ms: number;
  payload: TPayload;
};
```

This is a candidate, not current code. If adopted, update the Go protocol first and keep simulator/firmware/generated schemas aligned.

## Audio Chunk Direction

Phase 2 should define:

- codec: `pcm_s16le` first, Opus later if needed
- sample rates: 16000, 24000, or 48000 Hz
- mono audio for first StackChan path
- device frame duration: 20 ms or 40 ms for LAN responsiveness
- provider aggregation: adapter-specific, often 100-200 ms

## Control Event Direction

Future control events should cover:

- state update
- expression update
- subtitle delta
- playback start
- playback stop
- mode update
- device status
- trace marker

## Barge-In Requirements

Interrupt is not just stop audio. It must coordinate:

- local playback stop
- mouth/speaking state stop
- provider cancel/truncate/reset
- conversation state update
- trace marker
- transition back to listening

## Compatibility Rule

No public A21 protocol field may contain X21 or V21 naming unless the message is explicitly part of a V21 adapter contract.
