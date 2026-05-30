# A21 StackChan Touch Acceptance

## Scope

StackChan touch is a product surface, not only an input device. A21 must not
claim a gesture is production-ready merely because a firmware callback fired.

## Current Acceptance Levels

Core touch:

- `screen_touch`: screen tap wakes or starts listening.
- `top_tap`: top touch strip tap enters listening.
- `top_barge_in`: top touch while speaking interrupts playback and returns to listening.

Guided directional touch:

- `top_swipe_forward`: top strip swipe from screen/face side toward USB/back side.
- `top_swipe_backward`: top strip swipe from USB/back side toward screen/face side.

Directional top gestures are not core UX until A21 has screen guidance, operator
training, or physical labeling. The current StackChan top surface has no clear
front/middle/back affordance, so a missed directional gesture is a product
affordance gap, not an operator failure.

## Test Command

Use the guided acceptance command instead of ad hoc verbal instructions:

```bash
A21_DEVICE_ID=stackchan-001 A21_TOUCH_CASE=top_tap make stackchan-touch-acceptance
A21_DEVICE_ID=stackchan-001 A21_TOUCH_CASE=top_barge_in make stackchan-touch-acceptance
A21_DEVICE_ID=stackchan-001 A21_TOUCH_CASE=top_swipe_forward make stackchan-touch-acceptance
A21_DEVICE_ID=stackchan-001 A21_TOUCH_CASE=top_swipe_backward make stackchan-touch-acceptance
```

The command sends a prompt to the StackChan screen, watches Gateway device
traces, and writes `reports/a21-stackchan-touch-acceptance-*.json`.

## Product Rule

Do not remove hardware capabilities to simplify testing. If a capability is hard
to operate, add guidance, calibration, physical labeling, or a safer interaction
mapping. Core UX should rely only on gestures a first-time operator can perform
without hidden knowledge.
