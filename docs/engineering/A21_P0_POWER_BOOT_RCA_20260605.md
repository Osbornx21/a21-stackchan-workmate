# A21 P0 Power Boot RCA

Status: current P0 root-cause boundary.
Created: 2026-06-05.
Transition: `T-A21-P0-POWER-BOOT-RCA-001`.

This report supersedes earlier speculative PMIC-register repair notes. It does
not authorize a new flash. Internal test 4 remains the recovery baseline.

## Symptom

User-observed no-USB behavior:

- Pressing the upper-left physical power key flashes the screen.
- The red LED also flashes.
- The device then does not reach a usable official StackChan front-end.

Separate symptoms, not solved by this RCA:

- Voice wake sensitivity, delayed reply, and self-reply loop.
- Touch/RGB/vibration/servo parity after entering `AI.AGENT`.
- Official mobile app `Failed to process device data`.

## Source References Read

- A21 product overlay:
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- A21 overlay guard tests:
  `internal/app/official_stackchan_test.go`
- M5Stack StackChan source reference:
  `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan`
- M5Stack StackChan clean export:
  `/Users/jiyurun/Documents/stackchan-xiaozhi-endpoint-lab/workspace/official-stackchan-clean`
- 78 Xiaozhi ESP32 source reference:
  `/Users/jiyurun/Documents/小马暴力/sources/xiaozhi-esp32`

Reference state:

- `m5stack/StackChan` local reference is at `da156e1` but dirty. Treat its
  checked-out files as local reference material, not pristine Git proof.
- `78/xiaozhi-esp32` local reference is on
  `codex/x21-stackchan-firmware` and dirty. Use `origin/main` for mature
  Xiaozhi state-machine comparison.
- The clean StackChan export is not an independent Git repository.

## State-Machine Comparison

| Segment | Official M5Stack StackChan | Current A21 product lane | Finding |
| --- | --- | --- | --- |
| Physical key to PMIC hold | Hardware PWRKEY must make AXP2101 hold rails before ESP32 firmware runs. | Same hardware path. A21 app code is not running yet. | If there is no serial boot/app log, Gateway and AI.AGENT cannot be the cause. |
| PMIC init after app boot | `M5StackCoreS3Board` initializes I2C, AXP2101, power-save timer, IO expander, display, camera, touch. | Same sequence plus read-only PMIC diagnostic snapshot. | USB boot evidence proves this works only when external power is present. |
| Official front-end | `app_main()` installs Launcher, AI.AGENT, Avatar, Setup, and keeps Mooncake running until `AI.AGENT` requests Xiaozhi. | Current overlay tests forbid patching `main.cpp`, direct autostart, or parking `app_main`. | A21 currently preserves official front-end entry before AI.AGENT. |
| AI.AGENT entry | Opening `AI.AGENT` calls `GetHAL().requestXiaozhiStart()`, then official main loop starts Xiaozhi. | A21 routes Xiaozhi OTA/WebSocket and body relay to A21 Gateway after entry. | This is after boot; it cannot explain no-USB pre-boot failure. |
| 78 Xiaozhi voice state | Manual start uses `kListeningModeManualStop`; wake word opens channel from idle and then enters default listening mode. | A21 overlay adds keepalive, playback/touch events, and custom wake. | Useful for voice-loop RCA, not for physical power-key cold boot. |

## Evidence

- Stock official no-overlay diagnostic flash passed:
  `reports/a21-stackchan-official-baseline-flash-20260605-161057-1780647057743653000.json`
  - branch: `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
  - commit: `a923ac17b5ce`
  - app artifact: `stack-chan.bin`
  - app SHA-256:
    `a0cd9129b9e5f4718893d4fa672cb62e57088d5585057a1e4fa8ec835018135e`
  - flash executed: `true`

- A21 product restore flash passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-163318-1780648398612612000.json`
  - branch: `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
  - commit: `998a4bba6e9d`
  - app artifact: `a21-stackchan-official-xiaozhi-compatible.bin`
  - app SHA-256:
    `5968211923f788666e08bca51740e691dd17ae36d2535d8c265ced73d3abbf23`
  - flash executed: `true`

- NVS provisioning passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260605-141249-1780639969993980000.json`
  - write executed: `true`
  - preserved entries: `25`
  - mutated entries: `6`
  - servo calibration present: `true`
  - Wi-Fi credentials written: `true`
  - `app_config/is_configed` marked configured: `true`
  - OTA host: `47.103.57.217`
  - WebSocket path: `/v1/xiaozhi`

- USB PMIC snapshots were captured only while externally powered. In the
  recorded snapshots, AXP2101 `r01` does not prove battery discharging. The
  stock Xiaozhi `Axp2101` helper decodes `r01` direction bits as
  `1=charging` and `2=discharging`.

## Root-Cause Boundary

The first failing boundary is currently:

`physical PWRKEY -> AXP2101 battery/rail hold -> ESP32 bootloader/app reaches PMIC init`

The failure is not currently bounded to:

- A21 Gateway;
- provider chain;
- V21;
- AI.AGENT runtime;
- roleplay/professional mode;
- official Launcher code after `app_main()`;
- Xiaozhi voice state machine after network activation.

Reasoning:

- A21 firmware code can only change PMIC registers after ESP32 app boot reaches
  the board constructor.
- The observed standalone symptom can happen before that point.
- Project state records that a stock-official A/B still showed the same no-USB
  screen/red-LED flash symptom. If that observation is correct, the problem is
  below A21 code and below the restored official front-end.
- Earlier A21 PMIC writes for `REG10`, `REG22`, `REG24`, and `REG27=0x10`
  were tried and later removed because they did not close the no-USB symptom
  and diverged from stock official startup behavior.

## Current Hypothesis

H1 is the active hypothesis:

The device's standalone battery/PMIC rail path is not sustaining cold boot from
the current hardware state. Firmware can still boot under USB/external power,
which is why USB flash/serial evidence looks healthy while unplugged boot does
not.

H2 remains open but unproven:

A21 product assets or runtime increase early load enough to expose a marginal
battery/rail condition. This requires stock official to pass no-USB while A21
fails, which is not the current recorded evidence.

H3 remains open but unproven:

The physical power-key press/hold window differs from the expected hardware
behavior. This needs a timed press matrix against a known-good stock-official
baseline.

## Forbidden Fixes

- Do not reintroduce speculative AXP2101 register writes without new A/B or
  hardware measurement evidence.
- Do not patch `firmware/main/main.cpp` to autostart Xiaozhi before official
  Launcher/Home.
- Do not flash generic `xiaozhi.bin` as the product artifact.
- Do not treat USB-online Gateway status as no-USB power acceptance.
- Do not mix this P0 with voice-loop, body parity, or mobile app binding fixes.

## Next Evidence Required

Run the next step as evidence capture, not a code patch:

1. Charge the unit from USB until the official battery indicator and PMIC
   heartbeat show non-low battery state.
2. Power it off completely.
3. Disconnect USB and press the upper-left power key with a timed matrix:
   `250 ms`, `1 s`, `3 s`, `6 s`, `12 s`.
4. Record for each press whether it reaches:
   `no visible power`, `screen/red LED flash only`, `BootROM/serial`, official
   `Ready to configure`, official Home, or `AI.AGENT`.
5. If stock official no-overlay can be temporarily restored for diagnostics,
   repeat the same timed matrix with stock official, then restore the internal
   test 4 A21 product artifact.

Acceptance for this RCA:

- If stock official and A21 both fail no-USB, escalate to battery/PMIC/base
  hardware inspection instead of firmware churn.
- If stock official passes but A21 fails, open one single-variable firmware
  transition against the exact delta found in source/build/asset comparison.
- If A21 reaches official Home but fails after opening `AI.AGENT`, close this
  RCA and move to the separate voice/body runtime RCA.
