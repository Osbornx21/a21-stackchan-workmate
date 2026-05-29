package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeJSONUsesA21Protocol(t *testing.T) {
	msg := Envelope{
		Protocol: ProtocolVersion,
		DeviceID: "stackchan-home-01",
		Kind:     KindAudioFrame,
		Seq:      42,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"protocol":"a21.device.v1","device_id":"stackchan-home-01","kind":"audio.frame","seq":42}` {
		t.Fatalf("json = %s", data)
	}
}

func TestMessageKindsCoverCoreDeviceLoop(t *testing.T) {
	tests := map[string]Kind{
		"audio.frame":       KindAudioFrame,
		"device.event":      KindDeviceEvent,
		"assistant.state":   KindAssistantState,
		"screen.expression": KindScreenExpression,
		"motion.command":    KindMotionCommand,
	}

	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("kind = %q, want %q", got, want)
		}
	}
}

func TestEnvelopeSupportsTraceSessionAndPayload(t *testing.T) {
	payload := json.RawMessage(`{"state":"listening","mode":"workmate"}`)
	msg := Envelope{
		Protocol:  ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      KindControlEvent,
		Seq:       1,
		TraceID:   "a21-trace-000001",
		SessionID: "a21-session-000001",
		SentAtMS:  1234,
		Payload:   payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"trace_id":"a21-trace-000001"`) {
		t.Fatalf("json missing trace_id: %s", data)
	}
	if !strings.Contains(string(data), `"payload":{"state":"listening","mode":"workmate"}`) {
		t.Fatalf("json missing payload: %s", data)
	}
}

func TestControlEventPayloadStates(t *testing.T) {
	event := ControlEventPayload{
		State: ExpressionListening,
		Mode:  ModeWorkmate,
		Text:  "我在听",
		Final: false,
	}
	if event.State != "listening" {
		t.Fatalf("State = %q, want listening", event.State)
	}
	if event.Mode != "workmate" {
		t.Fatalf("Mode = %q, want workmate", event.Mode)
	}
}

func TestModesCoverA21OfficeAndProductStates(t *testing.T) {
	tests := map[string]Mode{
		"workmate":       ModeWorkmate,
		"companion":      ModeCompanion,
		"co_creation":    ModeCoCreation,
		"roleplay":       ModeRoleplay,
		"professional":   ModeProfessional,
		"focus":          ModeFocus,
		"public":         ModePublic,
		"private":        ModePrivate,
		"muted":          ModeMuted,
		"local_fallback": ModeLocalFallback,
		"error":          ModeError,
	}
	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("mode = %q, want %q", got, want)
		}
	}
}

func TestDeviceEventPayloadKinds(t *testing.T) {
	event := DeviceEventPayload{
		Event:           DeviceEventMockTurn,
		Mode:            ModeWorkmate,
		Text:            "先说，我在",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	}
	if event.Event != "mock.turn" {
		t.Fatalf("Event = %q, want mock.turn", event.Event)
	}
	if event.Mode != "workmate" {
		t.Fatalf("Mode = %q, want workmate", event.Mode)
	}
	if event.FirmwareID != "a21-stackchan" {
		t.Fatalf("FirmwareID = %q, want a21-stackchan", event.FirmwareID)
	}
}

func TestDeviceEventPayloadKindsCoverTouchSemantics(t *testing.T) {
	tests := map[DeviceEventKind]string{
		DeviceEventTouchWakeOrListen: "touch.wake_or_listen",
		DeviceEventTouchBargeIn:      "touch.barge_in",
	}
	for got, want := range tests {
		if string(got) != want {
			t.Fatalf("touch event = %q, want %q", got, want)
		}
	}

	event := DeviceEventPayload{
		Event:       DeviceEventTouchWakeOrListen,
		Mode:        ModeWorkmate,
		Text:        "先说，我在",
		TouchSource: TouchSourceScreen,
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"touch_source":"screen"`) {
		t.Fatalf("json missing touch source: %s", data)
	}
}

func TestAudioChunkPayloadShape(t *testing.T) {
	chunk := AudioChunk{
		Codec:              AudioCodecPCMS16LE,
		SampleRateHz:       16000,
		Channels:           1,
		DurationMS:         20,
		CaptureStartedAtMS: 1000,
		CaptureEndedAtMS:   1020,
		DataBase64:         "AAAA",
	}
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"codec":"pcm_s16le"`) {
		t.Fatalf("json missing codec: %s", data)
	}
	if chunk.Channels != 1 {
		t.Fatalf("Channels = %d, want 1", chunk.Channels)
	}
}
