# A21 Codex Audit

Date: 2026-05-29
Workspace: `/Users/jiyurun/Documents/New project`
Branch: `codex/a21-phase1-clean-skeleton`

## Current Repo Structure

```text
AGENTS.md
Makefile
cmd/a21/main.go
docs/a21/
docs/engineering/
docs/superpowers/
go.mod
internal/app/
internal/buildinfo/
internal/protocol/
internal/providers/
internal/runtimeguard/
```

## Language And Tooling

- Primary language: Go.
- Module path: `a21.local/a21`.
- Go version in module: `1.26`.
- Current verification entrypoint: `make verify`.
- Current runtime entrypoint: `go run ./cmd/a21`.
- No Node/pnpm workspace exists yet.
- No firmware tree exists yet.
- No Docker/Compose runtime exists yet.

The external TS/pnpm monorepo proposal remains a future option for simulator/dev-console work, not the current repository foundation.

## Current Commands

```bash
make verify
go run ./cmd/a21 version
go run ./cmd/a21 preflight
go run ./cmd/a21 doctor
```

`doctor` currently emits the same Phase 1 runtime report as `preflight`. Later phases should expand it into a broader dependency, proxy, StackChan, V21, provider, metrics, and firmware diagnostic.

## Current A21 Code

- `internal/buildinfo`: canonical A21 service identity.
- `internal/app`: CLI dispatch for version, preflight, and doctor.
- `internal/runtimeguard`: env, endpoint, cwd, port, and fingerprint guardrails.
- `internal/protocol`: minimal versioned A21 device envelope.
- `internal/providers`: provider-neutral ASR/LLM/TTS/knowledge contracts.

## Namespace Findings

Intentional X21/V21 references exist in:

- runtime guard configuration
- runtime guard tests
- architecture and baseline docs

No new runtime package, command, process, or service name uses X21/V21 identity.

## Legacy Neighbor Risks

Known local legacy territory:

- X21 backend: `8000`
- X21 local FunASR: `10095`
- V21/VKP services: `8080`, `18080`, `4173`, `42173`, `16686`, `16687`

A21 blocks endpoint env vars that point to known legacy ports and blocks A21 reserved port conflicts.

## Proxy And Network Findings

The local environment uses Dragon Cat Lite and DNS mapping into `198.18.0.x`. Phase 1 preflight records:

- default interface
- external DNS probe result
- proxy env variable names, not values

It blocks startup reports when the minimum fingerprint is missing.

## Verification Evidence

Fresh verification after the Phase 1 foundation:

```text
go test -count=1 ./...   PASS
make verify              PASS
go run ./cmd/a21 preflight  PASS in current shell
```

Known defensive checks:

```text
A21_PROVIDER_ENDPOINT=http://127.0.0.1:8000 ... preflight  exits 1
PATH without route/dig ... preflight                         exits 1
```

## Gaps

- No gateway HTTP server yet.
- No audio/control WebSocket yet.
- No simulator yet.
- No OpenTelemetry/Prometheus implementation yet.
- No firmware tree yet.
- No V21 adapter client yet.
- No real provider adapters yet.
- No CI yet.

These gaps are phase boundaries, not Phase 1 regressions.

## Recommended Next Step

Proceed to a narrowly scoped gateway/simulator plan that keeps the Go core intact:

1. Expand protocol around session, trace, audio, control, mode, and expression messages.
2. Add a mock gateway with health, audio/control WebSocket endpoints, and deterministic mock provider.
3. Add simulator only after the protocol and gateway contracts are testable.
4. Add observability hooks before real providers.
