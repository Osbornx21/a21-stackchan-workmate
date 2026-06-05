# StackChan Power State Machine A/B RCA

Date: 2026-06-05

## Transition

`T-STACKCHAN-POWER-STATE-MACHINE-AB-RCA-001`

## Current Symptom

- USB/external-power boot reaches official StackChan Launcher/Home.
- The flashed product candidate initializes PMIC, display, camera, touch, MCP,
  head touch, IO expander, RTC, IMU, and servos on USB.
- No-USB cold boot is still not accepted: pressing the upper-left power key
  flashes the screen/red LED, then power does not stay up long enough to reach
  the official front-end.

## Evidence Already Captured

- Latest successful product flash:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-153312-1780644792918047000.json`.
- Latest flashed app SHA-256:
  `a99ac8b1311a122865136ff52f06160c659e2e89465361cefd26eacfc7322684`.
- USB boot PMIC snapshot:
  `r00=38,r01=35,r10=34,r12=00,r14=65,r20=04,r21=20,r22=06,r23=3f,r24=00,r26=08,r27=00,r30=3f,r61=05,r62=0d,r63=15,r64=03,r80=05,r82=12,r90=3f,r91=00,r92=0d,r94=1c,r95=1c,r97=1c,r99=18,a4=64,a5=00`.
- Official source export reference:
  `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483`.
- The local official source worktree is dirty, so comparisons must use
  `git show HEAD:...` or clean export trees, not the dirty source files.
- Clean stock-official no-overlay build passed without flashing:
  `reports/a21-stackchan-official-baseline-20260605-155153-1780645913655503000.json`.
- Stock-official no-overlay app artifact:
  `/tmp/a21-stackchan-official-stock-build/stack-chan.bin`, SHA-256
  `a0cd9129b9e5f4718893d4fa672cb62e57088d5585057a1e4fa8ec835018135e`.
- Stock-official diagnostic flash plan passed without flashing:
  `reports/a21-stackchan-official-baseline-flash-20260605-160124-1780646484411288000.json`.

## State Machine Comparison

| Layer | Official StackChan HEAD | Current A21 product candidate | Finding |
| --- | --- | --- | --- |
| `app_main` | `GetHAL().init()` then installs Launcher, AI.AGENT, Avatar, Setup, and other apps; main loop runs Mooncake until `isXiaozhiStartRequested()`; only then starts Xiaozhi. | Same clean-HEAD lifecycle. No direct Xiaozhi autostart and no parked main loop. | Current no-USB failure is not caused by a replaced official app lifecycle. |
| Launcher/Home | If `app_config/is_configed` is false, Setup worker runs; otherwise Launcher/Home is created. | Same lifecycle; A21 NVS provision marks `app_config/is_configed=1` after preserving Wi-Fi and calibration entries. | Ready-to-configure trap was fixed separately; it is not the current no-USB symptom. |
| Board init order | NVS, board init, MCP, head touch, IO expander, RTC, IMU, servo, LVGL. Board constructor initializes I2C, PMIC, power-save timer, IO expander, display, camera, touch. | Same order, plus PMIC diagnostics and A21 Gateway routing after AI.AGENT. | USB logs prove this layer runs when external power is present. No-USB failure appears before this evidence can be produced. |
| StackChan PMIC writes | Official StackChan writes rail/backlight config and `REG27=0x00`. | A21 product currently adds `REG10 |= 0x04`, `REG22=0b110`, `REG24=0x00`, keeps `REG27=0x00`, and logs PMIC snapshot. | These writes occur after ESP32 app boot. They cannot explain or fix a power drop that happens before firmware reaches PMIC init. |
| Xiaozhi `m5stack-core-s3` reference | Initializes CoreS3 PMIC similarly to StackChan; no `REG10/22/24/27` power-key hunk. | A21 is not using this firmware directly. | Copying register sets from unrelated Xiaozhi boards is not justified for StackChan/CoreS3. |
| Other Xiaozhi AXP2101 boards | Kevin/Waveshare boards set `REG22=0b110`, often `REG27=0x10`, plus charger/rail registers. | A21 tried conservative parts of this profile, but physical no-USB behavior did not change. | These are working examples for their boards, not proof for StackChan. Treat as hypotheses only. |

## Binary/Configuration A/B Notes

- Stock-official `flash_args` and A21 product `flash_args` use the same
  offsets: bootloader `0x0`, app `0x20000`, partition table `0x8000`,
  OTA data `0xd000`, assets `0xa00000`.
- Stock-official app artifact: `stack-chan.bin`, about 3.6 MB.
- A21 product app artifact: `a21-stackchan-official-xiaozhi-compatible.bin`,
  about 3.6 MB.
- Stock-official assets artifact: about 2.2 MB.
- A21 product assets artifact: about 4.5 MB, and the partition table expands
  `assets` from 4 MB to 5 MB.
- A21 product sdkconfig changes include A21 OTA/Gateway URL, keepalive,
  playback/touch feature gates, custom wake word, and MultiNet7. These are
  product requirements and voice/body features, but they make the boot binary
  set materially different from stock official.

## Root-Cause Boundary

The first failing boundary is currently:

`physical upper-left PWRKEY -> AXP2101 battery/PMIC rail hold -> ESP32 bootloader/app reaches PMIC init`

not:

`Gateway -> provider -> voice -> AI.AGENT -> official Launcher/Home`

The decisive observation is that firmware PMIC writes happen only after the
ESP32 app is alive. If the detached device only flashes screen/red LED and never
reaches serial/app evidence, firmware register patches may be too late to
affect the failing cold-start window.

## Next A/B Required

1. Build a clean stock-official no-overlay baseline in an isolated work/build
   directory. This is no-flash evidence and does not affect the product device.
2. Compare the stock-official generated source and flash arguments against the
   current A21 product candidate.
3. Only after the no-flash evidence is recorded, decide the physical A/B:
   - If a stock-official temporary flash is allowed for diagnostic evidence,
     use only the guarded `stackchan-official-baseline-flash-execute`
     diagnostic lane. It accepts only the official no-overlay `stack-chan.bin`
     artifact, never the A21 product artifact or generic `xiaozhi.bin`.
   - If stock-official flashing is not allowed by release discipline, the next
     product-lane test must be a single-variable A21 candidate with an explicit
     hypothesis and no unrelated overlay churn.

## Hypotheses

- H1: The device battery/base/PMIC rail path cannot sustain cold start from the
  current hardware state. Stock official firmware would also fail no-USB cold
  boot. Evidence needed: stock-official physical A/B or external battery/rail
  measurement.
- H2: The A21 product candidate increases startup load or changes persisted PMIC
  shutdown state enough that cold boot fails before app PMIC init. Evidence
  needed: stock-official physical A/B succeeds while A21 fails.
- H3: The operator interaction window differs from official expectation
  (press/hold duration), but the current symptom has not been instrumented
  enough to prove this. Evidence needed: timed press/hold matrix against a known
  stock-official baseline.

## Forbidden Actions

- Do not flash generic `xiaozhi.bin` as the A21 product image.
- Do not apply additional PMIC register patches before an A/B result or a
  single explicit hypothesis is recorded.
- Do not roll back internal-test3 voice/protocol changes.
- Do not reset Wi-Fi/NVS unless a separate guarded NVS transition is opened.
- Do not run Git prune/gc.
