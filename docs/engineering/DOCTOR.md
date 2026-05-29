# A21 Doctor

## Current Command

```bash
go run ./cmd/a21 doctor
go run ./cmd/a21 doctor --output-dir reports
go run ./cmd/a21 serial-list
make doctor
```

`doctor` emits the preflight report, A21 firmware/tooling status, and also writes a timestamped JSON file:

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
- StackChan firmware manifest identity
- repository-local PlatformIO venv path
- repository-local PlatformIO core path
- validated firmware artifact count
- current git commit firmware artifact match
- local `/dev/cu.*` serial device inventory
- serial device USB-modem classification
- serial device process ownership via `lsof`

The firmware section intentionally checks repository-local paths under `.a21-tools/`. This keeps A21 firmware tooling isolated from X21/V21 and from global PlatformIO state.

`serial-list` emits just the serial inventory portion for physical-device prep. It does not flash, provision, reset, or open a serial monitor.

## Exit Codes

Current behavior:

- `0`: required checks pass
- `1`: a blocking check fails
- `2`: CLI usage error, such as unknown command or invalid doctor flag

Warnings-only exit code is reserved for a later expanded doctor.

## Future Expansion

The full doctor should eventually add human-readable table output and cover:

- project identity and namespace
- Go/Node/Python tooling
- A21 ports
- proxy and no-proxy coverage
- StackChan LAN reachability
- V21 adapter health
- provider connectivity and credential presence
- mock audio loop
- metrics/trace availability
- firmware compile status and guarded physical-upload readiness

Do not add fake checks. A doctor item should be implemented only when the repository has the subsystem it claims to check.
