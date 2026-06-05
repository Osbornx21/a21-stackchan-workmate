# A21 Internal Test 4 Recovery Baseline

Status: protected recovery baseline.
Last updated: 2026-06-05.

This document freezes the internal test 4 release as the project recovery
baseline. It is the point to return to if stabilization work becomes unsafe,
unclear, or destructive.

## Protected Baseline

- Release: `a21-internal-test4`
- Release URL:
  `https://github.com/Osbornx21/a21-stackchan-workmate/releases/tag/a21-internal-test4`
- Commit: `1387d58f364f6ae1c7258487fb5a9863567adc74`
- Commit subject: `chore(release): prepare internal test 4 package`
- Source branch:
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
- Product firmware app artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`
- Product firmware app SHA-256:
  `4d181ed2119bd31f8dc798e15d30865fdcc2d3980b3981a36e7b23d7e87f4350`

Release assets:

- `a21-internal-test4-source-1387d58f364f.tar.gz`
- `a21-internal-test4-firmware-bundle-1387d58f364f.tar.gz`
- `a21-internal-test4-1387d58f364f.sha256`
- `a21-internal-test4-firmware-files-1387d58f364f.sha256`

## Verified Before Freeze

- `go test ./internal/gateway ./internal/personality -count=1` passed.
- `GOMAXPROCS=2 make verify` passed.
- GitHub release assets were uploaded against commit `1387d58`.
- The main repository worktree was clean before the stabilization branch was
  created.
- Git loose-object/gc warning was resolved before this baseline was frozen.

## Product Lane Rules

- Product StackChan app flashes must use the official-compatible product lane
  and app artifact `a21-stackchan-official-xiaozhi-compatible.bin`.
- The generic `xiaozhi.bin` lane remains non-product/dev evidence only.
- No provider key, Wi-Fi credential, private transcript, or local secret may be
  added to repository files, release notes, reports, or logs.
- NVS writes are allowed only as explicit hardware-window actions. The current
  NVS writer has single-slot Wi-Fi semantics, so a new explicit Wi-Fi write
  replaces the previously stored hotspot entry.

## Known Unaccepted Areas

These are not reasons to discard the internal test 4 baseline. They are the
first stabilization targets after the freeze:

- No-cable physical power-key boot remains user-reported as unresolved.
- Voice UX has user-reported regression: lower wake sensitivity, delayed first
  response, and possible self-reply loop.
- StackChan body parity is not product-accepted: touch feedback, RGB reaction,
  vibration, and servo amplitude still need stock-vs-A21 evidence.
- User self-service provisioning from no preloaded Wi-Fi is not accepted.
- The official app/device-data failure needs an isolated root-cause report.
- Provider realtime race validation is not credible until the test fake or
  adapter loop is made race-safe.
- One old temporary firmware worktree remains dirty and must be archived or
  removed only after its diff is classified.

## Recovery Rule

If any stabilization branch makes the device worse, stop feature work and
restore from this baseline first. Do not mass-revert unknown commits in the
main worktree. Do not merge side-branch experiments into the product lane
without a narrow transition, acceptance evidence, and a rollback path.
