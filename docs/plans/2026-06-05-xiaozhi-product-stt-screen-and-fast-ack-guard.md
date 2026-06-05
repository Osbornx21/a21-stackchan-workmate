# A21 Xiaozhi Product STT Screen And Fast-Ack Guard

Status: executed in main control thread.
Date: 2026-06-05 CST.

## Transition

`T-XIAOZHI-PRODUCT-STT-SCREEN-AND-FAST-ACK-GUARD-001`

## Current State

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` identified that the
  product voice path still needed stronger evidence around fast-ack behavior
  and raw STT transcript display.
- Current code sends stock `type=stt` frames with raw ASR text before
  `tts.start`, which is correct for official Xiaozhi compatibility but risky
  as a product default in an office companion.
- Cloud-edge product chain defaults still enable the fast acknowledgement
  backchannel unless explicitly disabled.

## Target State

- Preserve official stock STT order and raw text compatibility for fixture,
  lab, and explicit compatibility runs.
- Make cloud-edge product defaults wait for the real answer instead of playing
  a fast acknowledgement placeholder.
- Add an explicit A21 product STT screen policy so ASR text can still drive the
  voice pipeline without forcing raw transcript text onto the device display.

## Trigger

The user reported unstable voice behavior and asked the control thread to apply
all review findings before more product acceptance.

## Action

- Add `A21_XIAOZHI_STT_SCREEN_POLICY` with values:
  - `raw`: stock-compatible raw STT frame text.
  - `status_only`: send a non-sensitive status phrase in `stt.text`.
  - `off`: suppress the device STT display frame while preserving ASR input for
    the answer pipeline.
- Keep default Gateway fixture behavior as `raw`.
- Set cloud-edge product-chain default to
  `A21_XIAOZHI_FAST_ACK_ENABLED=false`.
- Set cloud-edge product-chain default to
  `A21_XIAOZHI_STT_SCREEN_POLICY=status_only`.
- Record trace markers:
  `xiaozhi.stt.display.raw`, `xiaozhi.stt.display.status_only`, or
  `xiaozhi.stt.display.off`.

## Acceptance

- Stock STT before TTS tests still prove raw compatibility by default.
- Product status-only policy test proves raw ASR transcript is not written into
  device `stt.text` or traces.
- App env tests prove cloud-edge defaults are product-safe while preserving
  explicit override capability.
- Review-related Gateway race subset and default gates remain green.

## Failure State

- If stock STT ordering regresses, revert the product default only and keep the
  protocol-compatible raw path.
- If cloud-edge latency becomes worse in physical evidence, re-enable fast ack
  explicitly with a non-speaking or device-local policy in a separate
  transition.

## Rollback Path

- Set `A21_XIAOZHI_FAST_ACK_ENABLED=true` and
  `A21_XIAOZHI_STT_SCREEN_POLICY=raw` for lab runs.
- Revert this transition if it breaks stock Xiaozhi interoperability tests.

## Next State

`S-XIAOZHI-PRODUCT-STT-SCREEN-AND-FAST-ACK-GUARD-READY`

