# A21 Phase 4L Realtime Fixture Smoke

Phase 4L adds an offline realtime fixture smoke command:

```bash
go run ./cmd/a21 provider-realtime-fixture --provider doubao_tts_realtime --execute
go run ./cmd/a21 provider-realtime-fixture --provider openai_realtime --execute
```

Make wrapper:

```bash
A21_PROVIDER=doubao_tts_realtime make provider-realtime-fixture
```

This smoke reuses the existing realtime provider wrappers and injects a local fake WebSocket dialer. It validates that provider-owned session setup and first client events can be generated without leaking secrets and without opening a network connection.

## What It Proves

- required environment names are sufficient to construct the provider wrapper
- OpenAI realtime can create a session update, append a tiny audio frame, commit, create a response, and cancel through the adapter boundary
- Doubao realtime TTS can create a `tts_session.update`, append text, and mark text done through the adapter boundary
- endpoint host reporting is redacted to host only
- API keys, model values, voice IDs, auth headers, proxy URLs, and full provider URLs stay out of command output
- legacy-looking provider names are redacted

## What It Does Not Prove

- real provider connectivity
- paid provider correctness
- provider latency
- audio quality
- real StackChan speaker output
- full-duplex capture/playback

Those require later explicit smoke or integration commands with short timeouts, cost-aware operator intent, and trace/metric output.

## Safety

`provider-realtime-fixture` is separate from `provider-realtime-plan` and from future real network smoke commands. `provider-realtime-plan` remains dry-run and rejects `--execute`; `provider-realtime-fixture --execute` only uses the local fake connection.
