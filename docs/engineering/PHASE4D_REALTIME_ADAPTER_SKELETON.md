# A21 Phase 4D Realtime Adapter Skeleton

## Purpose

Phase 4D adds a narrow realtime WebSocket transport skeleton behind `internal/providers`.

This is not a production OpenAI or Doubao voice integration yet. It is a controlled adapter foundation that lets A21 test connection planning, session update events, cancel events, redaction, and provider network policy before any realtime audio semantics are connected to Gateway.

## Source Baseline

OpenAI's current Realtime documentation says server-to-server integrations can connect to Realtime over WebSocket with a backend-held API key, and that `/v1/realtime` voice-agent sessions exchange JSON client/server events over the socket.

The same docs show session initialization through `session.update` and audio conversations through client events such as `input_audio_buffer.append`, plus server events such as `response.output_audio.delta`.

Volcengine's current realtime AI voice docs expose a broad AI audio/video interaction stack, including end-to-end realtime voice model entry points, interruption, latency reduction, embedded hardware integration, RAG, MCP, and server OpenAPI flows. A21 therefore keeps Doubao realtime as a first-class candidate, but it must get a provider-specific adapter instead of being squeezed into the OpenAI event shape.

## Implemented Boundary

Code:

- `internal/providers/realtime.go`
- `internal/providers/realtime_test.go`

Capabilities:

- Builds a redacted OpenAI realtime WebSocket plan from `A21_OPENAI_API_KEY` and `A21_OPENAI_REALTIME_MODEL`.
- Reports endpoint host only; no API key, model value, auth header, or full URL is exposed.
- Adds the redacted plan to `doctor` under `voice.realtime_plan`.
- Rejects legacy-looking provider names before they enter the realtime path.
- Uses the existing A21 provider network policy.
- Wraps the existing `github.com/coder/websocket` dependency behind `RealtimeDialer`.
- Sends a generic `session.update` event after connect.
- Sends a generic `response.cancel` event for barge-in/cancel scaffolding.

## Non-Goals

Phase 4D does not:

- stream microphone audio to OpenAI or Doubao
- parse provider output audio deltas
- perform a paid network smoke by default
- expose provider credentials to firmware, simulator, or Gateway routes
- implement Doubao RTC/OpenAPI protocol details
- claim a real provider can deliver low-latency A21 voice yet

## Safety Rules

- Provider realtime code stays in `internal/providers`.
- Firmware never stores provider keys and never dials provider endpoints.
- Gateway must use A21 provider interfaces only.
- Realtime reports never include full URLs when query parameters can contain model or account-specific values.
- Ambient `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY` are ignored by default; provider egress must be explicit through `A21_PROVIDER_PROXY_URL`.
- Doubao-specific implementation requires a separate doc/readiness pass against official Volcengine docs.

## Next Step

The next implementation slice should add a provider-specific OpenAI realtime voice adapter around this transport, still using mock/local tests first:

1. Map A21 audio chunks to `input_audio_buffer.append`.
2. Map provider audio deltas into A21 downlink audio events.
3. Map barge-in to provider cancel and local playback stop spans.
4. Add a dry-run CLI that prints a redacted realtime plan only.
5. Add an explicit `--execute` smoke that is skipped unless credentials and operator intent are both present.
