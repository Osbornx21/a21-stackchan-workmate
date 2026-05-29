# A21 Protocol

## Current Contract

The current Go protocol package defines a small device envelope with trace/session metadata and raw JSON payloads:

```go
const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
	KindControlEvent     Kind = "control.event"
)

type Envelope struct {
	Protocol  string          `json:"protocol"`
	DeviceID  string          `json:"device_id"`
	Kind      Kind            `json:"kind"`
	Seq       uint64          `json:"seq"`
	TraceID   string          `json:"trace_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	SentAtMS  int64           `json:"sent_at_ms,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}
```

This remains intentionally small. It establishes the A21 namespace, control events, device events, and mock audio chunks without pretending to solve all device media concerns.

## Protocol Rules

- Every message family must be versioned.
- Device messages must include `device_id`.
- Runtime paths carry `trace_id` and `session_id`.
- Audio and control traffic may share an envelope but should have separate transport surfaces when real-time behavior requires it.
- StackChan firmware should receive semantic commands, not provider-specific events.
- V21 evidence must not be encoded as generic chat text when in professional mode; it needs explicit evidence/card fields in later contracts.

## Modes

A21 mode values are semantic product and office-state signals, not provider names:

- `workmate`
- `companion`
- `co_creation`
- `roleplay`
- `professional`
- `focus`
- `public`
- `private`
- `muted`
- `local_fallback`
- `error`

Only `professional` is allowed to trigger the V21 adapter path. `public`, `private`, and `muted` are office visibility/privacy states and must remain visible to the user without silently becoming professional retrieval context.

## Audio Chunk

Current mock audio payload:

- codec: `pcm_s16le` first, Opus later if needed
- sample rates: 16000, 24000, or 48000 Hz
- mono audio for first StackChan path
- device frame duration: 20 ms or 40 ms for LAN responsiveness
- provider aggregation: adapter-specific, often 100-200 ms

## Control And Device Events

Phase 2B supports:

- device event `mock.turn` -> control events `listening`, `thinking`, `speaking`
- device event `interrupt` -> control events `interrupted`, `listening`
- device event `touch.wake_or_listen` -> same turn path as `mock.turn`, with optional `touch_source`
- device event `touch.barge_in` -> same interruption path as `interrupt`, with optional `touch_source`
- audio frame -> mock listening ack

Firmware-originated `device.event` payloads may also carry build identity:

- `firmware_id`: must be `a21-stackchan` when present
- `firmware_version`: semver-like A21 firmware version
- `firmware_board`: must be `m5stack-cores3` when present
- `firmware_commit`: git SHA embedded by the A21 PlatformIO pre-build script
- `touch_source`: `screen` or `top_sensor` for semantic touch-origin events

Gateway records this identity in the device registry and rejects events whose firmware identity contains forbidden X21/V21 naming or mismatched A21 board/firmware fields.

Professional `control.event` payloads can now include explicit evidence fields:

- `confidence`
- `evidence[]`
- `speech_blocks[]`
- `screen_cards[]`
- `follow_ups[]`

This keeps V21 professional evidence visible to the client without pretending it is ordinary chat text.

Future control events should cover subtitle deltas, playback start/stop, explicit mode update event kinds, device status, and trace markers.

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
