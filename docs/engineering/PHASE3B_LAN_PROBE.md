# A21 Phase 3B LAN Probe

## Purpose

Phase 3B adds an explicit direct LAN reachability receipt for home and Shanghai-office debugging. A21 has to distinguish local path failure from provider/proxy failure before optimizing voice latency.

## Command

```bash
go run ./cmd/a21 lan-probe --target a21-gateway=127.0.0.1:21080 --output-dir reports
A21_LAN_TARGET=a21-gateway=127.0.0.1:21080 make lan-probe
```

Multiple targets can be supplied:

```bash
go run ./cmd/a21 lan-probe \
  --target a21-gateway=127.0.0.1:21080 \
  --target a21-v21-adapter=127.0.0.1:21121 \
  --timeout-ms 1000 \
  --output-dir reports
```

## Report

`lan-probe` writes:

```text
reports/a21-lan-probe-YYYYMMDD-HHMMSS.json
```

The report includes:

- `schema_version: a21.lan_probe.v1`
- current commit
- generated timestamp
- network/DNS fingerprint
- redacted proxy-policy metadata
- target name
- normalized host and port
- direct TCP status
- duration in milliseconds
- coarse error code

It does not include proxy URLs, proxy hosts/ports, proxy credentials, URL credentials, API keys, or raw X21 targets.

## Boundary

This is not a replacement for `doctor`. It is explicit because home development should not fail just because office-only StackChan or adapter endpoints are offline.

This is also not a latency benchmark. It proves TCP reachability and direct-path evidence only. LAN jitter, mDNS, WebSocket protocol health, device identity, and provider latency remain separate checks.
