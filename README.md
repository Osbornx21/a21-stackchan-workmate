# A21

A21 is a new StackChan/CoreS3 desktop AI workmate project. It is intentionally separate from X21 and V21. V21 is only a professional knowledge adapter target; A21 owns the embodied voice, device, network, provider, observability, and firmware discipline.

The current foundation is Go-first:

- `cmd/a21`: A21 CLI and Gateway entrypoint
- `internal/gateway`: mock/realtime Gateway, simulator, metrics, device registry, V21 professional routing
- `internal/providers`: provider-neutral voice and realtime adapter boundaries
- `internal/firmwarecheck`: firmware manifest, package, artifact, upload, device identity, and flash-plan guards
- `firmware/stackchan`: M5Stack CoreS3/StackChan firmware lane
- `docs/engineering`: current architecture, network, latency, observability, protocol, firmware, and phase docs

## Non-Negotiables

- Do not introduce new X21 names into A21 runtime code.
- Do not reuse V21 internals; call V21 only through the A21 adapter contract.
- Do not store provider API keys in firmware.
- Do not run raw PlatformIO upload. A21 blocks it by design.
- Keep firmware packages A21-named, commit-bound, manifest-backed, and dry-run checked before any future physical flash path exists.
- Keep localhost/LAN/StackChan traffic out of ambient proxies.

## First Commands

```bash
make verify
make namespace-audit
make firmware-tools
make firmware-test
make release-check
```

`make namespace-audit` rejects tracked file paths that introduce X21/V21 runtime identity outside the explicit V21 adapter/docs boundary.

`make firmware-tools` creates the repository-local `.a21-tools/` PlatformIO environment pinned to `platformio==6.1.19`.

`make firmware-current-artifact-check` validates the newest packaged firmware artifact for the current git commit through the release index and per-artifact manifest.

`make latency-bench` writes an ignored `reports/a21-latency-bench-*.json` evidence report for mock Gateway and audio WebSocket timing. The report includes current commit, network fingerprint, and redacted proxy-policy metadata so home, Shanghai office, LAN, and proxy runs can be compared without leaking proxy URLs.

`A21_LAN_TARGET=a21-gateway=127.0.0.1:21080 A21_LAN_SAMPLES=5 make lan-probe` writes an ignored `reports/a21-lan-probe-*.json` direct TCP reachability and jitter report. Use it in Shanghai for Gateway, StackChan-adjacent LAN endpoints, and the A21 V21 adapter boundary on `21121`. It rejects legacy X21/V21 internal ports, does not use proxy settings, and never stores proxy URLs or credentials.

`make v21-adapter-smoke` writes an ignored `reports/a21-v21-adapter-smoke-*.json` readiness report. `A21_V21_ADAPTER_URL=<adapter-boundary> make v21-adapter-smoke-execute` is the explicit query smoke; it never stores query text, answer text, evidence content, full adapter URLs, credentials, proxy URLs, or API keys.

`make release-check` runs Go tests, namespace audit, latency mock benchmarks, firmware tests/build, raw-upload blocker verification, firmware packaging, current-artifact validation, artifact retention planning, office handoff manifest generation, and doctor. It does not flash hardware or delete artifacts.

## Local Gateway

```bash
make gateway
```

Then open:

```text
http://127.0.0.1:21080/simulator
```

The simulator is the current no-hardware development surface for mock turns, professional evidence rendering, audio downlink buffering, and interruption behavior.

## Firmware Safety

```bash
make firmware-build
make firmware-package
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
go run ./cmd/a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --commit <git-sha> --port /dev/cu.usbmodemXXXX
```

`firmware-upload-check` is a dry-run receipt. It sets `flash_allowed: false`. Real flashing is intentionally not implemented.

For physical-device evidence:

```bash
make firmware-device-report
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_DEVICE_REPORT=reports/a21-devices-<timestamp>.json \
A21_DEVICE_ID=stackchan-001 \
make firmware-device-check
```

`/v1/devices` and the simulator Device Registry panel show the latest A21 connection status, mode, and expression for each registered device. They do not store utterance text or V21 evidence bodies.

When a fresh device report and explicit serial port are available, `make firmware-flash-plan` writes `reports/a21-firmware-flash-plan-*.json`. This is the strongest current no-flash receipt and still sets `flash_allowed: false`.

Before taking the project into the Shanghai office, run:

```bash
make office-handoff
```

It writes `reports/a21-office-handoff-*.json` with the current release-ledger-validated firmware artifact, no-delete artifact retention summary, serial inventory, and the next physical acceptance steps. It still sets `flash_allowed: false` and `delete_allowed: false`.

At the office, after `office-preflight` has produced a ready report, cross-check the receipts:

```bash
A21_HANDOFF_REPORT=reports/a21-office-handoff-<timestamp>.json \
A21_OFFICE_PREFLIGHT_REPORT=reports/a21-office-preflight-<timestamp>.json \
make office-acceptance
```

If a no-flash `firmware-flash-plan` report also exists, add `A21_FIRMWARE_FLASH_PLAN=reports/a21-firmware-flash-plan-<timestamp>.json`. The acceptance gate checks commit, artifact, device, and no-flash/no-delete invariants across receipts.

After the device is physically present and Gateway can see it, confirm fresh StackChan identity without claiming audio/screen/motion acceptance:

```bash
A21_OFFICE_ACCEPTANCE_REPORT=reports/a21-office-acceptance-<timestamp>.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-identity-acceptance
```

It writes `reports/a21-stackchan-identity-acceptance-*.json`, checks the office acceptance receipt against a fresh Gateway `/v1/devices` capture and USB serial inventory, and still sets `flash_allowed: false`.

## Required Reading

- [AGENTS.md](AGENTS.md)
- [A21 Codex Masterplan](docs/engineering/A21_CODEX_MASTERPLAN.md)
- [Firmware Release Discipline](docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md)
- [Network](docs/engineering/NETWORK.md)
- [Latency Budget](docs/engineering/LATENCY_BUDGET.md)
- [Protocol](docs/engineering/PROTOCOL.md)
- [V21 Integration](docs/engineering/V21_INTEGRATION.md)
