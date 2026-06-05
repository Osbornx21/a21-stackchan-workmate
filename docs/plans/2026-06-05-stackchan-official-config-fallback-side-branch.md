# StackChan Official Config Fallback Side Branch

Status: side-branch solution candidate.
Created: 2026-06-05.

## Problem

The product device can restore the official StackChan boot, PMIC, launcher,
Setup, and hardware lifecycle, but it can still stop on the official
`Ready to Configure` screen. That is because the official launcher checks
`GetHAL().isAppConfiged()` before showing Home. In the public source this reads
NVS namespace `app_config`, key `is_configed`.

The stock official mobile App path currently fails before it can set that flag:
the App receives BLE `notifyState` type `4`, expects RSA-OAEP(SHA-256)
encrypted device data, and shows `Failed to process device data` when the
public firmware returns the placeholder `hi-stack-chan`.

## Finding

This is not a reason to abandon the official front-end or StackChan hardware
surface. The missing piece is an A21-owned provisioning path that can mark the
official configuration gate complete without depending on the stock App's
closed BLE secret.

The official-compatible NVS path already safely backs up and rewrites the NVS
partition for A21 OTA/WebSocket and optional Wi-Fi credentials. Extending that
same guarded path to write `app_config/is_configed=1` when Wi-Fi credentials
are present unlocks the official Home while preserving:

- official PMIC and power-key lifecycle;
- official launcher, Home, Setup, and up-swipe Home behavior;
- official StackChan hardware apps and body capabilities;
- A21 Gateway routing after the user opens `AI.AGENT`;
- no provider key in firmware;
- no generic `xiaozhi.bin` product flash.

## Candidate Implemented

The side branch updates
`a21-stackchan-official-xiaozhi-compatible-nvs` so that when Wi-Fi credentials
are explicitly written, or when existing Wi-Fi credentials are preserved, the
generated NVS image also includes:

```text
app_config,is_configed,u8,1
```

The provision report summary now exposes
`app_config_marked_configured=true` without printing Wi-Fi passwords or
provider secrets.

## Product Path

1. Use the official-compatible firmware lane to keep the official front-end and
   hardware lifecycle.
2. Use guarded A21 NVS provisioning to set A21 OTA/WebSocket, Wi-Fi
   credentials, and `app_config/is_configed`.
3. Reboot into official Home.
4. User opens official `AI.AGENT`.
5. A21 runtime starts through Gateway `/v1/xiaozhi` for voice and
   `/stackChan/ws` for body/action.

## Longer-Term Device Management

This side branch does not try to reverse the stock App BLE secret. The durable
product direction should be an A21-owned provisioning and device-management
surface:

- local web or desktop provisioner for Wi-Fi, A21 Gateway URL, device ID, and
  mode defaults;
- optional A21 companion mobile page later;
- explicit status that stock official App binding requires official secret
  material and is not a prerequisite for A21 product use.

## Verification

- Focused NVS and official-compatible overlay tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'OfficialXiaozhiCompatible.*NVS|OfficialXiaozhiCompatibleOverlay|StackChanOfficialCandidateContract' -count=1`.
- `git diff --check` passed.
- The previous `internal/audio` Silero help test kill was rerun directly and
  passed:
  `GOMAXPROCS=2 go test ./internal/audio -run TestCheckedInSileroVADRunnerHelpIsExecutable -count=1`.
- Full `make verify` was attempted once; `internal/app` passed, but an
  unrelated `internal/audio` Silero help test was killed by the OS. Rerunning
  the single audio test passed.
- Independent endpoint lab no-flash demo at
  `/Users/jiyurun/Documents/stackchan-xiaozhi-endpoint-lab` also passed as
  separate evidence that an official front-end entry can connect only to A21
  Gateway voice/body surfaces. It is not an A21 product candidate replacement.

## Hardware Write Constraint

The side worktree must not execute the NVS write. The existing T7 control guard
requires a clean foreground hardware-window branch, so the candidate should be
committed here, then cherry-picked or merged into
`codex/a21-hardware-window-*` before running
`a21-stackchan-official-xiaozhi-compatible-nvs-execute`.

## Foreground Toolchain Fix

After the candidate was integrated into the foreground hardware-window branch,
the first guarded NVS execution attempt passed T7 control guard but failed
before reading flash. The report was
`reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260605-140250-1780639370066907000.json`;
`write_executed=false`.

The read log showed ESP-IDF `export.sh` selecting a broken system Python 3.13
whose `_ssl` and `hashlib` modules are blocked by macOS code-signing policy.
This was not a serial, Gateway, Wi-Fi, or firmware-state failure.

The foreground hardware-window branch now supports `A21_IDF_PYTHON` and
`--idf-python` for the official-compatible NVS executor. When supplied, it uses
that Python directly for esptool and the ESP-IDF NVS scripts instead of
sourcing `export.sh`. This keeps the hardware write guarded while avoiding the
broken local Python auto-detection path.

## Remaining Physical Acceptance

- Execute the guarded NVS path with the operator-approved Wi-Fi credentials.
- Confirm reboot reaches official Home instead of `Ready to Configure`.
- Open `AI.AGENT` and confirm A21 voice/body runtime connects through Gateway.
