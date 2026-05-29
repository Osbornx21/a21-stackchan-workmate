# A21 Phase 4K Gateway Provider Runtime Guard

Phase 4K adds an explicit voice-provider runtime selector for the A21 Gateway.

The default remains safe:

```text
A21_GATEWAY_VOICE_PROVIDER=mock
```

If `A21_GATEWAY_VOICE_PROVIDER` is unset or `mock`, Gateway uses `a21-mock-voice` even when `A21_PROVIDER_PRIMARY` and real provider credentials are present. This prevents a developer shell, Codex session, or Shanghai network experiment from accidentally moving Gateway traffic onto a paid or external provider.

To inject the selected provider object into Gateway runtime, set:

```text
A21_GATEWAY_VOICE_PROVIDER=selected
A21_PROVIDER_PRIMARY=doubao_tts_realtime
```

`selected` means "construct the provider selected by `A21_PROVIDER_PRIMARY`." It does not mean "dial the provider." Current realtime providers still keep ordinary text-turn `StartTurn` disabled. Explicit realtime session APIs, fixture-based smoke commands, timeout controls, and cost-aware operator confirmation must be added before Gateway can perform real provider audio sessions.

## Doctor Semantics

`doctor.voice.provider` reports selected provider local health from `A21_PROVIDER_PRIMARY`.

`doctor.voice.gateway_provider` reports the provider object Gateway would use at startup from `A21_GATEWAY_VOICE_PROVIDER`.

This distinction is intentional. A provider can be configured and ready while Gateway remains on mock runtime.

## Safety Rules

- Gateway defaults to mock.
- `A21_PROVIDER_PRIMARY` alone never changes Gateway runtime.
- Legacy-looking runtime values are redacted to `invalid_gateway_voice_provider`.
- API keys, model values, voice IDs, auth headers, proxy URLs, and full provider URLs are never printed.
- StackChan firmware still sees only A21 protocol events, never provider-specific event names or credentials.
- No new command in this phase performs a real provider network call.

## Verification

Target tests:

```bash
go test ./internal/providers -run 'TestGatewayVoiceProviderFromEnv' -count=1
go test ./internal/app -run 'TestGatewayServerFromEnv|TestGatewayServerSelectedRealtimeProviderKeepsTextTurnGuarded|TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets|TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime' -count=1
```

The tests prove that:

- Gateway remains mock by default even with real provider env configured.
- Gateway uses selected provider only with explicit runtime opt-in.
- ordinary text turns remain guarded for current realtime provider wrappers.
- doctor distinguishes selected provider health from Gateway runtime provider.
- secrets and model/voice values do not appear in Gateway or doctor output.
