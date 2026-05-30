# Phase 5L: StackChan Capability Acceptance

Status: implemented as a no-flash evidence gate.

## Goal

Move StackChan acceptance beyond identity-only confirmation without pretending that a firmware declaration is a physical test. The Gateway device registry can say which hardware surfaces the firmware exposes; `stackchan-capability-acceptance` requires separate per-capability evidence before those surfaces are accepted.

## Command

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT=reports/a21-stackchan-physical-evidence.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-capability-acceptance
```

Direct CLI:

```bash
go run ./cmd/a21 stackchan-capability-acceptance \
  --identity-acceptance reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
  --evidence reports/a21-stackchan-physical-evidence.json \
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
