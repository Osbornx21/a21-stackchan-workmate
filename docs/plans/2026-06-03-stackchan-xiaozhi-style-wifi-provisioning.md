# StackChan Xiaozhi-Style Wi-Fi Provisioning Plan

Goal: make A21 StackChan product firmware follow Xiaozhi's startup Wi-Fi
provisioning model instead of requiring hardcoded Wi-Fi credentials.

Current state: A21 public Gateway is the main voice-edge entry at
`47.103.57.217`, Mac Gateway remains selectable, and the correct product
firmware lane is `a21-stackchan-official-xiaozhi-compatible.bin`. A21 self-owned
firmware still enters local fallback when Wi-Fi credentials are missing.

Target state: first boot or missing Wi-Fi credentials enter a device-side Wi-Fi
provisioning state. The official-compatible product overlay explicitly preserves
Xiaozhi's three provisioning-method contract: stored NVS credentials first,
Hotspot/SoftAP captive portal by default, with BluFi and acoustic provisioning
kept as build-time alternatives. No provider key is ever stored in firmware.

Trigger: user requested Xiaozhi-like startup provisioning, specifically the
three Xiaozhi provisioning methods, and said not to invent a new A21-only
configuration flow.

Actions:

- Add firmware-native tests proving missing Wi-Fi enters provisioning rather
  than local fallback, and that Hotspot/BluFi/Acoustic methods are named,
  redacted, and callable by the runtime seam.
- Add the minimal A21 self-owned firmware provisioning state/method/runtime
  seam needed to pass those tests.
- Add Go overlay-contract tests proving the official-compatible overlay
  explicitly enables Hotspot provisioning, keeps BluFi and acoustic as named
  alternatives, and does not hardcode Wi-Fi credentials.
- Patch the official-compatible overlay `sdkconfig.defaults` hunk with the
  product default provisioning flags.
- Update firmware docs, handoff log, and project state machine with the exact
  boundary: endpoint Wi-Fi provisioning is in product lane; physical flash and
  acceptance still require a guarded hardware window.

Acceptance:

- Focused firmware-native tests pass.
- Focused official overlay Go test passes.
- `make verify` passes.
- Existing staged work stays staged; no previous staged changes are reverted.

Failure state: if native PlatformIO tooling is unavailable, keep the tests and
code staged but report the missing local toolchain. If overlay tests fail,
do not claim provisioning is product-lane ready.

Rollback path: revert only this plan plus the new provisioning edits in
`firmware/stackchan/*`, `firmware/stackchan-official/overlays/*`,
`internal/app/official_stackchan_test.go`, and the two control documents.

Next state: after build/package verification, schedule a guarded physical flash
of the product lane and observe first-boot/no-NVS behavior on the real
StackChan.
