# A21 Doctor

## Current Command

```bash
go run ./cmd/a21 doctor
go run ./cmd/a21 doctor --output-dir reports
make doctor
```

Phase 3B `doctor` reuses the preflight report and also writes a timestamped JSON file:

```text
reports/a21-doctor-YYYYMMDD-HHMMSS.json
```

It checks what the current foundation can truthfully check:

- legacy env variable prefixes
- A21 endpoint env vars pointing at known legacy X21/V21 ports
- legacy working directories
- A21 reserved port conflicts
- minimum network/DNS fingerprint
- proxy env variable names, without values

## Exit Codes

Current behavior:

- `0`: required Phase 1 checks pass
- `1`: required Phase 1 check fails
- `2`: CLI usage error, such as unknown command or invalid doctor flag

Warnings-only exit code is reserved for a later expanded doctor.

## Future Expansion

The full doctor should eventually add human-readable table output and cover:

- project identity and namespace
- Go/Node/Python/Firmware tooling
- A21 ports
- proxy and no-proxy coverage
- StackChan LAN reachability
- V21 adapter health
- provider connectivity and credential presence
- mock audio loop
- metrics/trace availability
- firmware compile status

Do not add fake checks. A doctor item should be implemented only when the repository has the subsystem it claims to check.
