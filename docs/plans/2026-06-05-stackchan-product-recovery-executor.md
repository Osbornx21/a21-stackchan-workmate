# StackChan Product Recovery Executor

Status: executed in main control thread.
Date: 2026-06-05 CST.

## Transition

`T-STACKCHAN-PRODUCT-RECOVERY-EXECUTOR-001`

## Current State

- Product Gateway deployment is live, but the product device is still absent
  from `/v1/devices`.
- Official StackChan relay status remains `connected=false`.
- The latest guarded product flash log reports
  `Failed to connect to ESP32-S3: No serial data received`.
- Operators currently have to run product recovery precheck, wait-ROM product
  flash, and post-flash Gateway checks as separate commands during a tight
  physical BOOT/RESET window.

## Target State

- Keep `stackchan-product-recovery` as a read-only precheck by default.
- Add an explicit `--execute-flash` mode that:
  - requires the existing product confirmation token,
  - skips flashing if the product is already online,
  - uses only the existing official-compatible product flash implementation,
  - forces wait-ROM behavior with `--esptool-before no_reset`,
  - writes a product recovery execution report, and
  - performs a post-flash Gateway/official relay check.
- Add Makefile wrappers so the hardware window has a single correct command
  and does not drift to the generic Xiaozhi flash lane.

## Trigger

The user asked for rapid full remediation and allowed provider, Gateway,
hardware, ECS, and flash work, while still requiring protection of the
official-compatible product lane.

## Action

- Extend `a21 stackchan-product-recovery` with `--execute-flash`.
- Add schema `a21.stackchan_product_recovery_execution.v1`.
- Add Make targets:
  - `stackchan-product-recovery`
  - `stackchan-product-recovery-execute`
- Add tests for confirmation-token enforcement, online-product flash skipping,
  wait-ROM official-compatible flash execution, and postcheck status.

## Acceptance

- Focused product recovery and official-compatible flash tests pass.
- Read-only recovery still emits `a21.stackchan_product_recovery.v1`.
- Execution recovery emits `a21.stackchan_product_recovery_execution.v1`.
- No generic `xiaozhi.bin` or generic Xiaozhi flash lane is used.
- The command cannot write flash without both the product confirmation token
  and the existing hardware control guard.

## Failure State

- If ROM/download is not entered, the executor fails at the existing wait-ROM
  flash step and records the no-serial evidence.
- If flash succeeds but the product does not reconnect, the executor records
  `flash_passed_reconnect_pending` rather than physical acceptance.

## Rollback Path

- Use read-only `stackchan-product-recovery`.
- Use the existing direct
  `a21-stackchan-official-xiaozhi-compatible-flash-execute` product lane.

## Next State

`S-STACKCHAN-PRODUCT-RECOVERY-EXECUTOR-READY`

