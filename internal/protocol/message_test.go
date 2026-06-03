package protocol

import (
	"encoding/base64"
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
		"audio.frame":          KindAudioFrame,
		"audio.playback.chunk": KindAudioPlaybackChunk,
		"device.event":         KindDeviceEvent,
		"assistant.state":      KindAssistantState,
		"screen.expression":    KindScreenExpression,
		"motion.command":       KindMotionCommand,
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

func TestExpressionStatesCoverLocalFallback(t *testing.T) {
	tests := map[string]ExpressionState{
		"idle":           ExpressionIdle,
		"listening":      ExpressionListening,
		"thinking":       ExpressionThinking,
		"speaking":       ExpressionSpeaking,
		"interrupted":    ExpressionInterrupted,
		"professional":   ExpressionProfessional,
		"local_fallback": ExpressionLocalFallback,
		"error":          ExpressionError,
	}
	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("expression = %q, want %q", got, want)
		}
	}
}

func TestModesCoverA21OfficeAndProductStates(t *testing.T) {
	tests := map[string]Mode{
		"dialogue":       ModeDialogue,
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

func TestProductModesConvergeToRoleplayAndProfessional(t *testing.T) {
	tests := map[Mode]Mode{
		"":                ModeRoleplay,
		ModeDialogue:      ModeRoleplay,
		ModeWorkmate:      ModeRoleplay,
		ModeCompanion:     ModeRoleplay,
		ModeCoCreation:    ModeRoleplay,
		ModeRoleplay:      ModeRoleplay,
		ModeFocus:         ModeRoleplay,
		ModePublic:        ModeRoleplay,
		ModePrivate:       ModeRoleplay,
		ModeMuted:         ModeRoleplay,
		ModeProfessional:  ModeProfessional,
		ModeLocalFallback: ModeLocalFallback,
		ModeError:         ModeError,
	}
	for input, want := range tests {
		if got := NormalizeProductMode(input); got != want {
			t.Fatalf("NormalizeProductMode(%q) = %q, want %q", input, got, want)
		}
	}
	catalog := CanonicalProductModes()
	if len(catalog) != 2 || catalog[0] != ModeRoleplay || catalog[1] != ModeProfessional {
		t.Fatalf("canonical product modes = %#v, want roleplay/professional", catalog)
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
		Capabilities: map[string]string{
			"microphone":    "available",
			"speaker":       "available",
			"screen":        "available",
			"screen_touch":  "available",
			"top_touch":     "available",
			"servo_y":       "available",
			"servo_x":       "planned_continuous_rotation_axis",
			"rgb":           "available",
			"camera":        "planned_core_s3_camera",
			"imu":           "planned_9_axis_imu",
			"ambient_light": "planned_ambient_light_sensor",
			"proximity":     "planned_proximity_sensor",
			"battery":       "planned_550mah_battery",
			"nfc":           "planned_nfc",
			"infrared":      "planned_infrared_tx_rx",
		},
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
	if event.Capabilities["screen_touch"] != "available" {
		t.Fatalf("screen_touch capability = %q, want available", event.Capabilities["screen_touch"])
	}
	if event.Capabilities["camera"] != "planned_core_s3_camera" {
		t.Fatalf("camera capability = %q, want planned_core_s3_camera", event.Capabilities["camera"])
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"camera":"planned_core_s3_camera"`) {
		t.Fatalf("json missing capabilities: %s", data)
	}
}

func TestStackChanHardwareCapabilityKeysCoverCoreS3AndRobotBody(t *testing.T) {
	want := []string{
		"microphone",
		"speaker",
		"screen",
		"screen_touch",
		"top_touch",
		"servo_y",
		"servo_x",
		"rgb",
		"camera",
		"imu",
		"ambient_light",
		"proximity",
		"battery",
		"nfc",
		"infrared",
	}
	keys := StackChanHardwareCapabilityKeys()
	got := map[string]bool{}
	for _, key := range keys {
		got[key] = true
	}
	for _, key := range want {
		if !got[key] {
			t.Fatalf("StackChanHardwareCapabilityKeys missing %q; keys=%v", key, keys)
		}
	}
}

func TestDeviceEventPayloadKindsCoverTouchSemantics(t *testing.T) {
	tests := map[DeviceEventKind]string{
		DeviceEventTouchWakeOrListen:     "touch.wake_or_listen",
		DeviceEventTouchBargeIn:          "touch.barge_in",
		DeviceEventTouchTopTap:           "touch.top.tap",
		DeviceEventTouchTopSwipeForward:  "touch.top.swipe_forward",
		DeviceEventTouchTopSwipeBackward: "touch.top.swipe_backward",
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

func TestDeviceEventPayloadKindsCoverRuntimeEcho(t *testing.T) {
	event := DeviceEventPayload{
		Event: DeviceEventRuntimeEcho,
		Mode:  ModeWorkmate,
		RuntimeEcho: map[string]string{
			"screen":  "speaking",
			"servo_y": "48deg",
			"rgb":     "#002430",
		},
	}
	if string(event.Event) != "runtime.echo" {
		t.Fatalf("runtime echo event = %q, want runtime.echo", event.Event)
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"event":"runtime.echo"`,
		`"runtime_echo"`,
		`"screen":"speaking"`,
		`"servo_y":"48deg"`,
		`"rgb":"#002430"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("json missing %q: %s", want, data)
		}
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

func TestValidateAudioChunkAcceptsFullPCM16Frame(t *testing.T) {
	chunk := AudioChunk{
		Codec:        AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   testPCM16Base64WithSample(0),
	}
	if err := ValidateAudioChunk(chunk); err != nil {
		t.Fatalf("ValidateAudioChunk() error = %v", err)
	}
}

func TestValidateAudioChunkRejectsShortPCMFrame(t *testing.T) {
	chunk := AudioChunk{
		Codec:        AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   "AAAA",
	}
	if err := ValidateAudioChunk(chunk); err == nil {
		t.Fatal("ValidateAudioChunk() error = nil, want short PCM rejection")
	}
}

func TestAudioPlaybackChunkPayloadShape(t *testing.T) {
	chunk := AudioPlaybackChunk{
		StreamID:     "a21-mock-stream-001",
		Codec:        AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   "AAAA",
	}
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"stream_id":"a21-mock-stream-001"`,
		`"codec":"pcm_s16le"`,
		`"sample_rate_hz":16000`,
		`"channels":1`,
		`"duration_ms":20`,
		`"data_base64":"AAAA"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("playback chunk json missing %q: %s", want, text)
		}
	}
}

func TestValidateAudioPlaybackChunkAcceptsPCM16Playback(t *testing.T) {
	chunk := AudioPlaybackChunk{
		StreamID:     "a21-playback-stream-001",
		Codec:        AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   testPCM16Base64WithSample(1024),
	}

	if err := ValidateAudioPlaybackChunk(chunk); err != nil {
		t.Fatalf("ValidateAudioPlaybackChunk() error = %v", err)
	}
}

func TestValidateAudioPlaybackChunkRejectsShortPCM(t *testing.T) {
	chunk := AudioPlaybackChunk{
		StreamID:     "a21-playback-stream-001",
		Codec:        AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   "AAAA",
	}

	if err := ValidateAudioPlaybackChunk(chunk); err == nil {
		t.Fatal("ValidateAudioPlaybackChunk() error = nil, want short PCM rejection")
	}
}

func testPCM16Base64WithSample(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}
