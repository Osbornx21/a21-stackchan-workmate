# ADR 0001: WebSocket Library For A21 Gateway

## Status

Accepted for Phase 2B.

## Context

A21 needs WebSocket transport for low-latency device control and later audio streaming. Go's standard library does not include a production WebSocket server API, and hand-rolling the protocol would violate A21's principle of using good wheels instead of patchwork implementations.

The current A21 gateway is a small Go HTTP server. The WebSocket library must keep that shape intact.

## Decision

Use `github.com/coder/websocket` for the A21 gateway WebSocket boundary.

Primary-source checks on 2026-05-29:

- GitHub README: https://github.com/coder/websocket
- Go package docs: https://pkg.go.dev/github.com/coder/websocket

Reasons:

- minimal and idiomatic API
- first-class `context.Context` support
- zero runtime dependencies
- JSON helpers in `wsjson`
- works with `net/http`
- supports close handshake, ping/pong, and concurrent writes

## Consequences

A21 accepts one narrow production dependency for transport correctness.

The dependency is isolated inside `internal/gateway`. Protocol structs remain owned by `internal/protocol`, and product/session logic must not become tied to library-specific types.

If a later benchmark or firmware spike shows this library is insufficient for audio streaming, replace it behind gateway transport adapters instead of rewriting business logic.
