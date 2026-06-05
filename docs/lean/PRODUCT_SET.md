# A21 Lean Product Set

Generated from frozen HEAD `090eca6` on `main-lean`.

Command:

```bash
go list -deps ./cmd/a21 | rg 'a21\.local/a21' | sort
```

Current product package set:

```text
a21.local/a21/cmd/a21
a21.local/a21/internal/agentplan
a21.local/a21/internal/app
a21.local/a21/internal/audio
a21.local/a21/internal/audio/opuscodec
a21.local/a21/internal/buildinfo
a21.local/a21/internal/firmwarecheck
a21.local/a21/internal/gateway
a21.local/a21/internal/personality
a21.local/a21/internal/protocol
a21.local/a21/internal/providers
a21.local/a21/internal/runtimeguard
a21.local/a21/internal/transport/stackchan
a21.local/a21/internal/transport/xiaozhi
a21.local/a21/internal/v21adapter
```

Current lean boundary:

- Product `cmd/a21` keeps runtime, doctor, provider, V21 adapter, protocol,
  transport, guarded firmware discipline, and product StackChan commands.
- Lab-only demo/bench/evidence/professional acceptance commands are dispatched
  through `cmd/a21-lab`.
- Product Gateway no longer exposes the simulator route by default.
- Product readiness no longer treats mock demo or simulator proof as
  product-ready.
- Redline assets remain in the product set only when they are real runtime
  boundaries, not self-proof scaffolding.

Verification status, 2026-06-05:

- `go list -deps ./cmd/a21` contains no `bench`, `demo`, `evidence`, or
  `professional` package path matches.
- A temporary `cmd/a21` product binary had no `lab` or `simulator` symbol
  matches under `go tool nm`.
- Source package implementation files are still Go-package co-located where
  that preserves the current Go-first foundation; the enforced product
  boundary is CLI dispatch plus product binary reachability.
