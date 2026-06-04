# Body Scene Physical Acceptance Surface

Status: active local transition.
Date: 2026-06-05 CST.

## Goal

Close the product loop after paced `full_check`: let an operator record that
screen, RGB, and servo motion were visibly observed, without changing firmware,
voice protocol, provider execution, V21, camera, NFC, or IR behavior.

## Transition

- Current state: `POST /v1/xiaozhi/body-scene` delivers paced `full_check`
  machine evidence with `physical_accepted=false`.
- Target state: `/workspace` exposes an explicit physical acceptance action
  for the latest hardware scene, and Gateway records a redacted
  body-scene physical acceptance marker tied to the scene trace/session.
- Acceptance: focused tests prove the endpoint rejects missing scene evidence,
  accepts the latest `full_check` trace with screen/RGB/servo observations, and
  updates device registry/trace without raw notes or broad PRD acceptance.

## Boundaries

- Do not build or flash firmware.
- Do not write NVS.
- Do not execute providers or V21.
- Do not expose camera/NFC/IR/reboot/OTA/app-lifecycle actions.
- Do not reopen internal-test3 voice acceptance.
