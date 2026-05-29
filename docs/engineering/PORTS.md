# A21 Ports

## Rule

All A21-owned ports must be documented here before code starts depending on them. A21 must never silently reuse X21/V21 ports.

## Current Phase 1 Reserved Ports

| Port | Owner | Status | Purpose |
| ---: | --- | --- | --- |
| 21080 | A21 Core | reserved | future core HTTP/API/device OTA entry |
| 21081 | A21 Realtime | reserved | future realtime voice or WebSocket entry |
| 21073 | A21 Console | reserved | future dev console UI |
| 21086 | A21 Observability | reserved | future trace/observability UI |
| 21095 | A21 ASR Sidecar | reserved | future local ASR sidecar |
| 21114 | A21 LLM Adapter | reserved | future local model/LLM adapter |
| 21434 | A21 Local Model Bridge | reserved | future Ollama-compatible adapter |

Phase 1 does not bind long-running services yet. Runtime preflight probes these ports on loopback and fails if they are occupied.

## Legacy Ports Treated As Contaminated

| Port | Legacy Owner |
| ---: | --- |
| 8000 | X21 backend |
| 8080 | V21/VKP legacy service |
| 10095 | X21 local FunASR |
| 18080 | V21/VKP backend |
| 4173 | V21/VKP frontend preview |
| 42173 | V21/VKP frontend preview |
| 16686 | tracing UI |
| 16687 | tracing UI alternate |

A21 endpoint env vars pointing at these ports must fail preflight unless a future explicit migration/audit command allows them.

## Future Media Split Candidate

The external master directive proposed:

| Port | Candidate Purpose |
| ---: | --- |
| 21000 | gateway HTTP |
| 21001 | audio WebSocket |
| 21002 | control WebSocket |
| 21003 | dev console |
| 21004 | device simulator |
| 21121 | V21 adapter |
| 21090 | metrics |
| 21091 | trace UI |
| 21099 | health |

Do not adopt this split by stealth. If Phase 2 moves from the current reserved set to this split, write an ADR and update runtimeguard tests first.

## Process Naming

New process names must start with `a21-`, for example:

- `a21-core`
- `a21-gateway`
- `a21-device-simulator`
- `a21-v21-adapter`
- `a21-observability`

Forbidden for new A21-owned processes:

- `x21-*`
- `v21-gateway`
- `voice-gateway`
- `assistant-server`
- ambiguous `stackchan-service`
