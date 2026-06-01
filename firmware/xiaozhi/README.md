# A21 Xiaozhi Firmware Overlays

This directory stores A21-owned overlays for externally checked-out xiaozhi
firmware source trees. The source tree remains outside this repo; overlays are
the reviewable A21 artifact.

## Debug Playback Ack Overlay

`overlays/a21-debug-playback-ack.patch` adds firmware-side debug playback
runtime echoes for physical instrumentation. It is disabled by default with
`CONFIG_A21_DEBUG_DEVICE_EVENTS=n`.
The patch is generated with zero context to keep A21 whitespace gates clean;
apply it to the external firmware source with `git apply --unidiff-zero`.

When enabled in a debug build, the client advertises `features.device_events`.
It sends redacted `type=device`, `kind=playback` events only after the Gateway
server hello includes the A21 allowance `a21.profile=debug` and
`a21.device_events=true`.

The overlay emits:

- `playback=start` after decoded PCM reaches the firmware audio output task.
- `playback=stop_done` after a server TTS stop has moved the device out of
  speaking state and any auto-stop playback queue wait has completed.

Stock xiaozhi profile builds must leave the option disabled. No flash command
is introduced here. Build and flash planning must continue through the existing
guarded `xiaozhi-firmware-flash-plan` and `xiaozhi-firmware-flash-execute`
lanes.
