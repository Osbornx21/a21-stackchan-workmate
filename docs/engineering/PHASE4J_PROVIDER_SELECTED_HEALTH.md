# A21 Phase 4J Provider Selected Health

## Purpose

Phase 4J makes `doctor` voice health reflect the selected `A21_PROVIDER_PRIMARY` instead of always reporting the deterministic mock provider.

This improves operator truthfulness: when `A21_PROVIDER_PRIMARY=doubao_tts_realtime`, the health section now reports the Doubao realtime TTS provider object and its local configuration state. It does not dial the provider and it does not switch the Gateway runtime to a paid or external provider.

## Implemented Boundary

Code:

- `internal/providers/factory.go`
- `internal/providers/factory_test.go`
- `internal/app/doctor.go`
- `internal/app/app_test.go`

Provider factory behavior:

- empty or `mock` -> `a21-mock-voice`
- `openai_realtime` -> `a21-openai-realtime-voice`
- `doubao_tts_realtime` -> `a21-doubao-realtime-tts`
- legacy-looking primary -> `invalid_legacy_provider`
- unknown primary -> `unknown_provider`
- known but not-yet-wrapped provider -> unavailable provider with a redacted detail

## Safety Rules

- Provider factory construction must not dial external services.
- `doctor` health is local configuration health, not provider connectivity proof.
- Gateway still defaults to the mock provider unless `A21_GATEWAY_VOICE_PROVIDER=selected` is explicitly configured.
- Raw legacy provider values, API keys, model values, voice IDs, auth headers, proxy URLs, and full provider URLs must not appear in doctor output.

## Test Coverage

Covered by:

```bash
go test ./internal/providers -run 'TestVoiceProviderFromEnv' -count=1
go test ./internal/app -run 'TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets|TestRunDoctorBlocksLegacyProviderPrimaryWithoutEchoingValue|TestRunDoctorIncludesProviderCatalogWithoutSecrets' -count=1
```

The tests cover default mock selection, Doubao TTS selection, OpenAI selection, legacy provider redaction, and doctor selected-health output without secret leakage.

## Next Step

Phase 4K adds the explicit Gateway provider-selection guard and records the distinction in `doctor.voice.gateway_provider`.
