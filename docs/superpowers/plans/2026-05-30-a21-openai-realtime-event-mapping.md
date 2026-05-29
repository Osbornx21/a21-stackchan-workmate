# A21 OpenAI Realtime Event Mapping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a provider-specific OpenAI Realtime event mapper that translates A21 audio chunks, response triggers, cancellation, and provider audio deltas without executing real network calls.

**Architecture:** Keep OpenAI event names inside `internal/providers`; Gateway and firmware continue to speak A21 protocol/events only. The mapper is pure and testable, so later realtime adapters can call it from the WebSocket transport without mixing provider JSON shapes into Gateway or firmware code.

**Tech Stack:** Go, `internal/providers`, `internal/protocol`, table-driven tests, OpenAI Realtime WebSocket event docs.

---

### Task 1: A21 Audio to OpenAI Input Events

**Files:**
- Create: `internal/providers/openai_realtime_test.go`
- Create: `internal/providers/openai_realtime.go`

- [ ] **Step 1: Write the failing test**

```go
func TestOpenAIRealtimeInputAudioAppendEventMapsA21Chunk(t *testing.T) {
	event, err := OpenAIRealtimeInputAudioAppendEvent(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if event["type"] != "input_audio_buffer.append" {
		t.Fatalf("type = %v", event["type"])
	}
	if event["audio"] == "" {
		t.Fatal("audio is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestOpenAIRealtimeInputAudioAppendEventMapsA21Chunk -count=1`

Expected: FAIL because `OpenAIRealtimeInputAudioAppendEvent` is undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/providers/openai_realtime.go` with a pure mapper that validates PCM S16LE, mono, supported sample rate, duration, and base64 data, returning `input_audio_buffer.append`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers -run TestOpenAIRealtimeInputAudioAppendEventMapsA21Chunk -count=1`

Expected: PASS.

### Task 2: Response and Cancel Events

**Files:**
- Modify: `internal/providers/openai_realtime_test.go`
- Modify: `internal/providers/openai_realtime.go`

- [ ] **Step 1: Write the failing test**

```go
func TestOpenAIRealtimeManualTurnEvents(t *testing.T) {
	events := OpenAIRealtimeManualTurnEvents()
	if events[0]["type"] != "input_audio_buffer.commit" {
		t.Fatalf("first type = %v", events[0]["type"])
	}
	if events[1]["type"] != "response.create" {
		t.Fatalf("second type = %v", events[1]["type"])
	}
	cancel := OpenAIRealtimeCancelEvent(VoiceCancelRequest{Reason: CancelBargeIn})
	if cancel["type"] != "response.cancel" {
		t.Fatalf("cancel = %#v", cancel)
	}
	if _, ok := cancel["reason"]; ok {
		t.Fatalf("cancel leaked A21-local reason into provider event: %#v", cancel)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestOpenAIRealtimeManualTurnEvents -count=1`

Expected: FAIL because turn/cancel helpers are undefined.

- [ ] **Step 3: Write minimal implementation**

Add `OpenAIRealtimeManualTurnEvents` and `OpenAIRealtimeCancelEvent`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers -run TestOpenAIRealtimeManualTurnEvents -count=1`

Expected: PASS.

### Task 3: OpenAI Output Delta to A21 Voice Event

**Files:**
- Modify: `internal/providers/voice.go`
- Modify: `internal/providers/openai_realtime_test.go`
- Modify: `internal/providers/openai_realtime.go`
- Modify: `docs/engineering/PHASE4D_REALTIME_ADAPTER_SKELETON.md`

- [ ] **Step 1: Write the failing test**

```go
func TestOpenAIRealtimeServerAudioDeltaMapsToVoiceEvent(t *testing.T) {
	event, ok, err := OpenAIRealtimeServerEventToVoiceEvent(VoiceSession{
		TraceID: "a21-trace",
		SessionID: "a21-session",
		DeviceID: "stackchan-001",
	}, map[string]any{
		"type": "response.output_audio.delta",
		"response_id": "resp_123",
		"delta": base64.StdEncoding.EncodeToString([]byte{0, 1}),
	})
	if err != nil || !ok {
		t.Fatalf("ok/err = %v/%v", ok, err)
	}
	if event.Kind != VoiceEventSpeaking || event.Audio == nil {
		t.Fatalf("event = %#v", event)
	}
	if event.StreamID != "resp_123" {
		t.Fatalf("stream = %q", event.StreamID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers -run TestOpenAIRealtimeServerAudioDeltaMapsToVoiceEvent -count=1`

Expected: FAIL because output mapping and `VoiceEvent.Audio` are missing.

- [ ] **Step 3: Write minimal implementation**

Add `VoiceAudioChunk` to `voice.go`, map `response.output_audio.delta` to a speaking event with a PCM S16LE 24 kHz mono audio chunk, and ignore unrelated provider events with `ok=false`.

- [ ] **Step 4: Run full provider tests**

Run: `go test ./internal/providers`

Expected: PASS.

### Task 4: Verification

**Files:**
- Modify: `docs/engineering/PHASE4D_REALTIME_ADAPTER_SKELETON.md`

- [ ] **Step 1: Run release gate**

Run: `make verify`

Expected: PASS.

- [ ] **Step 2: Run full release gate**

Run: `make release-check`

Expected: PASS and produce a new A21 firmware artifact for the current commit after committing.
