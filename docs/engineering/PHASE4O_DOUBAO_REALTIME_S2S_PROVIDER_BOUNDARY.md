# A21 Phase 4O Doubao Realtime S2S Provider Boundary

## Purpose

Doubao end-to-end realtime speech-to-speech is A21's China-mainland companion fast-path candidate. Phase 4O gives it an A21 `VoiceProvider` boundary so operators can select and inspect it without making a paid or unverified provider call.

This is a safety boundary, not a launch claim.

## Implemented

- `A21_PROVIDER_PRIMARY=doubao_realtime` now returns `a21-doubao-realtime-voice` from `NewVoiceProviderFromEnv`.
- Health reports missing/present configuration using env names only:
  - `A21_DOUBAO_API_KEY`
  - `A21_DOUBAO_APP_ID`
  - `A21_DOUBAO_RESOURCE_ID`
  - `A21_DOUBAO_REALTIME_MODEL`
- When configured, health reports `degraded` with an explicit execution guard instead of claiming the provider is executable.
- Health output never prints API keys, app IDs, resource IDs, model values, auth headers, proxy values, or full provider URLs.
- `Cancel` returns a local A21 cancellation event so runtime orchestration can be tested without provider access.
- Gateway still defaults to `a21-mock-voice`; `A21_GATEWAY_VOICE_PROVIDER=selected` is required before the selected provider is injected.

## Deliberately Blocked

`StartTurn` rejects execution with an explicit error. This prevents A21 from pretending that Doubao S2S is already connected, cancellable, low-latency, or voice-clone capable in this repo.

Unlock requires:

- official API shape confirmed against current Volcengine documentation
- explicit credentials provided by the user
- short-timeout credentialed smoke behind a dedicated command
- cancellation/truncation behavior verified
- latency report captured
- no secrets in reports, logs, traces, simulator UI, firmware, or artifacts

## Firmware Boundary

StackChan firmware remains provider-neutral. It must never store Doubao credentials, call Volcengine, or emit Doubao-native events. Firmware build/package/upload discipline is unchanged: only current-commit A21 artifacts may be considered for flashing, and raw PlatformIO upload remains blocked by the A21 guard.
