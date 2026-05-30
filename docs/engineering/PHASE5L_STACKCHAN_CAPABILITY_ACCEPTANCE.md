# Phase 5L: StackChan Capability Acceptance

Status: implemented as a no-flash evidence gate.

## Goal

Move StackChan acceptance beyond identity-only confirmation without pretending that a firmware declaration is a physical test. The Gateway device registry can say which hardware surfaces the firmware exposes; `stackchan-capability-acceptance` requires separate per-capability evidence before those surfaces are accepted.

## Evidence Template Command

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-physical-evidence
```

This writes:

```text
reports/a21-stackchan-physical-evidence-YYYYMMDD-HHMMSS.json
```

The template starts with each capability as `pending`. When observations are already available, generate a passed evidence file with explicit entries:

```bash
go run ./cmd/a21 stackchan-physical-evidence \
  --identity-acceptance reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --pass microphone=gateway_audio_frame \
  --pass speaker=audible_playback \
  --pass screen=operator_visible_state \
  --pass screen_touch=touch_event \
  --pass top_touch=touch_event \
  --pass servo_y=servo_clamped_motion \
  --pass rgb=operator_visible_state \
  --output-dir reports
```

Gateway-observable evidence can be derived from A21 runtime state:

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
A21_DERIVE_GATEWAY_EVIDENCE=1 \
make stackchan-physical-evidence
```

This mode reads the A21 Gateway directly with no ambient proxy. It can mark `microphone` when the latest trace has `audio.frame.received`, `speaker` when Gateway sent `audio.playback.chunk`, `screen_touch` or `top_touch` when the latest touch event includes the source, and `screen`/`servo_y`/`rgb` when the latest device registry includes firmware `runtime_echo` values. If `runtime_echo.screen` is missing, `screen` can still fall back to `gateway_render_state`.

`runtime_echo` is stronger than Gateway intent because it is emitted by firmware after applying screen, motion, and RGB state. It is still not the same as human-visible or instrument-measured proof: physical audibility, visibility, servo movement, and LED output remain separate observations for Shanghai office acceptance.

## Acceptance Command

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT=reports/a21-stackchan-physical-evidence-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-capability-acceptance
```

Direct CLI:

```bash
go run ./cmd/a21 stackchan-capability-acceptance \
  --identity-acceptance reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
  --evidence reports/a21-stackchan-physical-evidence-YYYYMMDD-HHMMSS.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --output-dir reports
```

## Evidence Contract

The physical evidence file must be A21-scoped:

```json
{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "<expected-git-sha>",
  "artifact_sha256": "<identity-artifact-sha256>",
  "observations": [
    {
      "capability": "microphone",
      "status": "passed",
      "evidence_type": "gateway_audio_frame",
      "observed_at_ms": 1780000001000
    }
  ]
}
```

Required capabilities:

- `microphone`
- `speaker`
- `screen`
- `screen_touch`
- `top_touch`
- `servo_y`
- `rgb`

Each capability must be declared `available` in the identity acceptance report and must have a `passed` evidence observation with a non-empty `evidence_type` and `observed_at_ms`.

## Guardrails

- Output report schema: `a21.stackchan_capability_acceptance.v1`
- Success status: `capability_acceptance_status=confirmed`
- Blocked status: `capability_acceptance_status=blocked`
- Always sets `flash_allowed=false`
- Always sets `delete_allowed=false`
- Rejects input/output paths containing forbidden X21/V21 identity before reading them
- Rejects legacy identity inside capability/evidence fields without echoing the polluted value

## Non-Claims

This gate does not claim OTA, production full-duplex, AEC, provider latency, or physical flashing. It is the acceptance ledger for hardware surfaces only.
