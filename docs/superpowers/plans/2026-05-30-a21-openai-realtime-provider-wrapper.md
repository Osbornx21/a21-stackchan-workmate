# A21 OpenAI Realtime Provider Wrapper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap the tested OpenAI Realtime event mapper in an A21 provider boundary that can be configured from env, health-checked without leaking secrets, and exercised with an injected WebSocket connection.

**Architecture:** Keep the provider wrapper inside `internal/providers`. It builds on the existing `RealtimeWebSocketAdapter` and Phase 4E mapper, uses `A21_` env only, and does not perform real provider network I/O in tests or doctor. Gateway and firmware continue to see only A21 provider/session types.

**Tech Stack:** Go, `internal/providers`, injected `RealtimeDialer`, table-driven tests.

---

### Task 1: Env Config and Health

**Files:**
- Create: `internal/providers/openai_realtime_provider_test.go`
- Create: `internal/providers/openai_realtime_provider.go`

- [ ] **Step 1: Write the failing test**

```go
func TestOpenAIRealtimeVoiceProviderFromEnvReportsConfiguredWithoutLeakingSecrets(t *testing.T) {
	provider := NewOpenAIRealtimeVoiceProviderFromEnv([]string{
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, nil)
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v", health)
	}
	rendered := mustJSON(t, health)
	if strings.Contains(rendered, "sk-a21-secret") || strings.Contains(rendered, "gpt-realtime-2") {
		t.Fatalf("health leaked secret/model: %s", rendered)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestOpenAIRealtimeVoiceProviderFromEnvReportsConfiguredWithoutLeakingSecrets -count=1`

Expected: FAIL because `NewOpenAIRealtimeVoiceProviderFromEnv` is undefined.

- [ ] **Step 3: Write minimal implementation**

Create `OpenAIRealtimeVoiceProvider`, env config parsing, and `Health`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers -run TestOpenAIRealtimeVoiceProviderFromEnvReportsConfiguredWithoutLeakingSecrets -count=1`

Expected: PASS.

### Task 2: Injected Session Lifecycle

**Files:**
- Modify: `internal/providers/openai_realtime_provider_test.go`
- Modify: `internal/providers/openai_realtime_provider.go`

- [ ] **Step 1: Write the failing test**

```go
func TestOpenAIRealtimeVoiceProviderSessionWritesUpdateAudioCommitAndCancel(t *testing.T) {
	conn := &fakeRealtimeConn{}
	provider := NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{
		APIKey: "sk-a21-secret",
		Model:  "gpt-realtime-2",
	}, fakeRealtimeDialer{conn: conn})
	session, err := provider.StartRealtimeSession(context.Background(), VoiceSession{
		TraceID: "a21-trace",
		SessionID: "a21-session",
		DeviceID: "stackchan-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString([]byte{1, 2})
	if err := session.SendAudio(context.Background(), protocol.AudioChunk{Codec: protocol.AudioCodecPCMS16LE, SampleRateHz: 24000, Channels: 1, DurationMS: 20, DataBase64: payload}); err != nil {
		t.Fatal(err)
	}
	if err := session.CommitAndCreateResponse(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := session.Cancel(context.Background(), VoiceCancelRequest{Reason: CancelBargeIn}); err != nil {
		t.Fatal(err)
	}
	if got := eventTypes(conn.messages); strings.Join(got, ",") != "session.update,input_audio_buffer.append,input_audio_buffer.commit,response.create,response.cancel" {
		t.Fatalf("event types = %#v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestOpenAIRealtimeVoiceProviderSessionWritesUpdateAudioCommitAndCancel -count=1`

Expected: FAIL because realtime session wrapper is undefined.

- [ ] **Step 3: Write minimal implementation**

Implement `StartRealtimeSession`, `OpenAIRealtimeProviderSession.SendAudio`, `CommitAndCreateResponse`, `Cancel`, and `Close`.

- [ ] **Step 4: Run provider tests**

Run: `go test ./internal/providers`

Expected: PASS.

### Task 3: Docs and Verification

**Files:**
- Create: `docs/engineering/PHASE4F_OPENAI_REALTIME_PROVIDER_WRAPPER.md`
- Modify: `docs/engineering/PHASE4E_OPENAI_REALTIME_EVENT_MAPPING.md`

- [ ] **Step 1: Document boundaries**

Document that Phase 4F is a configured provider/session wrapper only: no real network smoke, no Gateway switch, no firmware secret exposure.

- [ ] **Step 2: Run verification**

Run:

```bash
make verify
```

Expected: PASS.
