# A21 Ports

Status: active registry entrypoint.
Date: 2026-06-01.

This file exists because A21 agents are required to check an explicit port
registry before adding or running services. The detailed network and proxy
policy lives in `docs/engineering/NETWORK.md`; this file is the compact index.

## Reserved A21 Ports

| Port | Owner | Status |
| --- | --- | --- |
| `21080` | A21 Gateway HTTP/WebSocket | active default |
| `21081` | A21 realtime lane | reserved |
| `21073` | A21 local console | reserved |
| `21086` | A21 observability | reserved |
| `21095` | A21 ASR sidecar | reserved |
| `21114` | A21 LLM adapter | reserved |
| `21121` | A21 V21 adapter boundary | optional local boundary |
| `21130` | A21 Agent I/O bridge | optional host-only bridge |
| `21434` | A21 local-model bridge | reserved |

Adding a new service port requires updating this file, `NETWORK.md`, and the
runtime guard that checks reserved ports.

## Forbidden Defaults

A21 must not adopt known legacy/internal ports as A21 defaults or endpoint env
targets: `8000`, `8080`, `10095`, `18080`, `4173`, `42173`, `16686`, and
`16687`.

V21 may be reached only through the A21 adapter boundary, normally `21121`, not
through V21 internals.

## Runtime Rule

Localhost, LAN, `.local`, StackChan, and V21 adapter traffic must stay direct
and must not silently inherit global proxy settings. Cloud provider egress may
use only explicit A21 provider proxy configuration.
