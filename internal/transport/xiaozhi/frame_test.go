package xiaozhi

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseControlFixturesPropagatesA21Identity(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantType    MessageType
		wantTrace   string
		wantSession string
		assert      func(t *testing.T, frame Frame)
	}{
		{
			name:        "hello",
			path:        "testdata/hello.json",
			wantType:    MessageTypeHello,
			wantTrace:   "a21-trace-fixture-hello",
			wantSession: "a21-session-fixture-hello",
			assert: func(t *testing.T, frame Frame) {
				t.Helper()
				if frame.Control.Hello.Version != 1 {
					t.Fatalf("hello version = %d, want 1", frame.Control.Hello.Version)
				}
				if frame.Control.Hello.Transport != "websocket" {
					t.Fatalf("hello transport = %q, want websocket", frame.Control.Hello.Transport)
				}
				if frame.Control.Hello.AudioParams.Format != "opus" {
					t.Fatalf("hello audio format = %q, want opus", frame.Control.Hello.AudioParams.Format)
				}
				if frame.Control.Hello.AudioParams.SampleRate != 16000 {
					t.Fatalf("hello sample rate = %d, want 16000", frame.Control.Hello.AudioParams.SampleRate)
				}
				if frame.Control.Hello.AudioParams.Channels != 1 {
					t.Fatalf("hello channels = %d, want 1", frame.Control.Hello.AudioParams.Channels)
				}
				if frame.Control.Hello.AudioParams.FrameDuration != 60 {
					t.Fatalf("hello frame duration = %d, want 60", frame.Control.Hello.AudioParams.FrameDuration)
				}
				if frame.Control.Hello.AudioParams.BinaryProtocolVersion != 1 {
					t.Fatalf("hello binary protocol = %d, want 1", frame.Control.Hello.AudioParams.BinaryProtocolVersion)
				}
			},
		},
		{
			name:        "listen",
			path:        "testdata/listen.json",
			wantType:    MessageTypeListen,
			wantTrace:   "a21-trace-stackchan-001",
			wantSession: "a21-session-fixture-listen",
			assert: func(t *testing.T, frame Frame) {
				t.Helper()
				if frame.Control.Listen.State != "start" {
					t.Fatalf("listen state = %q, want start", frame.Control.Listen.State)
				}
				if frame.Control.Listen.Mode != "manual" {
					t.Fatalf("listen mode = %q, want manual", frame.Control.Listen.Mode)
				}
			},
		},
		{
			name:        "abort",
			path:        "testdata/abort.json",
			wantType:    MessageTypeAbort,
			wantTrace:   "a21-trace-fixture-abort",
			wantSession: "a21-session-stackchan-001",
			assert: func(t *testing.T, frame Frame) {
				t.Helper()
				if frame.Control.Abort.Reason != "wake_word_detected" {
					t.Fatalf("abort reason = %q, want wake_word_detected", frame.Control.Abort.Reason)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			frame, err := ParseTextFrame(data, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"})
			if err != nil {
				t.Fatal(err)
			}

			if frame.Kind != FrameKindControl {
				t.Fatalf("frame kind = %q, want control", frame.Kind)
			}
			if frame.Direction != DirectionDeviceToServer {
				t.Fatalf("direction = %q, want device_to_server", frame.Direction)
			}
			if frame.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q, want stackchan-001", frame.DeviceID)
			}
			if frame.TraceID != tc.wantTrace {
				t.Fatalf("trace_id = %q, want %q", frame.TraceID, tc.wantTrace)
			}
			if frame.SessionID != tc.wantSession {
				t.Fatalf("session_id = %q, want %q", frame.SessionID, tc.wantSession)
			}
			if frame.Control.Type != tc.wantType {
				t.Fatalf("type = %q, want %q", frame.Control.Type, tc.wantType)
			}
			tc.assert(t, frame)
		})
	}
}

func TestParseHelloAcceptsAudioAliasAndRejectsUnsafeParams(t *testing.T) {
	frame, err := ParseTextFrame([]byte(`{
		"type": "hello",
		"version": 3,
		"device_id": "stackchan-001",
		"audio": {
			"format": "opus",
			"sample_rate": 16000,
			"channels": 1,
			"frame_duration": 60
		}
	}`), DirectionDeviceToServer, Identity{})
	if err != nil {
		t.Fatal(err)
	}
	if frame.Control.Hello.Transport != "websocket" {
		t.Fatalf("transport = %q, want websocket default", frame.Control.Hello.Transport)
	}
	if frame.Control.Hello.Version != 3 || frame.Control.Hello.AudioParams.BinaryProtocolVersion != 3 {
		t.Fatalf("version = %d binary = %d, want 3/3", frame.Control.Hello.Version, frame.Control.Hello.AudioParams.BinaryProtocolVersion)
	}

	_, err = ParseTextFrame([]byte(`{
		"type": "hello",
		"device_id": "stackchan-001",
		"audio": {
			"format": "opus",
			"sample_rate": 24000,
			"channels": 1,
			"frame_duration": 60
		}
	}`), DirectionDeviceToServer, Identity{})
	if !errors.Is(err, ErrUnsupportedAudioParams) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedAudioParams)
	}
}

func TestParseHelloCapturesFeatureProfile(t *testing.T) {
	frame, err := ParseTextFrame([]byte(`{
		"type": "hello",
		"device_id": "stackchan-001",
		"features": {
			"mcp": true,
			"aec": true,
			"device_events": true,
			"playback_events": true,
			"debug_metrics": true
		},
		"audio": {
			"format": "opus",
			"sample_rate": 16000,
			"channels": 1,
			"frame_duration": 60
		}
	}`), DirectionDeviceToServer, Identity{})
	if err != nil {
		t.Fatal(err)
	}
	features := frame.Control.Hello.Features
	if !features.MCP || !features.AEC || !features.DeviceEvents || !features.PlaybackEvents || !features.DebugMetrics {
		t.Fatalf("features = %+v, want all advertised flags captured", features)
	}

	frame, err = ParseTextFrame([]byte(`{
		"type": "hello",
		"device_id": "stackchan-002",
		"audio": {
			"format": "opus",
			"sample_rate": 16000,
			"channels": 1,
			"frame_duration": 60
		}
	}`), DirectionDeviceToServer, Identity{})
	if err != nil {
		t.Fatal(err)
	}
	if frame.Control.Hello.Features != (HelloFeatures{}) {
		t.Fatalf("features = %+v, want zero-value stock feature set", frame.Control.Hello.Features)
	}
}

func TestParseControlUsesDeviceIDFromMessageWhenConnectionIdentityIsAbsent(t *testing.T) {
	frame, err := ParseTextFrame([]byte(`{"type":"listen","state":"stop","device_id":"stackchan-message-01"}`), DirectionDeviceToServer, Identity{})
	if err != nil {
		t.Fatal(err)
	}
	if frame.DeviceID != "stackchan-message-01" {
		t.Fatalf("device_id = %q, want stackchan-message-01", frame.DeviceID)
	}
	if frame.TraceID != "a21-trace-stackchan-message-01" {
		t.Fatalf("trace_id = %q, want derived A21 trace", frame.TraceID)
	}
	if frame.SessionID != "a21-session-stackchan-message-01" {
		t.Fatalf("session_id = %q, want derived A21 session", frame.SessionID)
	}
}

func TestParseTextFrameErrorsAreClear(t *testing.T) {
	tests := []struct {
		name string
		data string
		want error
	}{
		{name: "malformed json", data: `{`, want: ErrMalformedJSON},
		{name: "missing device identity", data: `{"type":"hello"}`, want: ErrMissingDeviceIdentity},
		{name: "unsupported message type", data: `{"type":"goodbye"}`, want: ErrUnsupportedMessageType},
		{name: "unsupported listen state", data: `{"type":"listen","state":"pause","device_id":"stackchan-001"}`, want: ErrUnsupportedListenState},
		{name: "legacy identity", data: `{"type":"listen","state":"start","device_id":"x21-device"}`, want: ErrLegacyIdentity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseTextFrame([]byte(tc.data), DirectionDeviceToServer, Identity{})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want.Error()) {
				t.Fatalf("err text = %v, want clear %q", err, tc.want.Error())
			}
		})
	}
}

func TestParseBinaryFrameRecognizesOpusWithoutDecoding(t *testing.T) {
	payload := []byte{0xf8, 0xff, 0xfe, 0x01}

	frame, err := ParseBinaryFrame(payload, DirectionDeviceToServer, Identity{
		DeviceID:  "stackchan-001",
		TraceID:   "a21-trace-opus-001",
		SessionID: "a21-session-opus-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 0x00

	if frame.Kind != FrameKindOpus {
		t.Fatalf("frame kind = %q, want opus", frame.Kind)
	}
	if frame.Opus == nil {
		t.Fatal("opus frame is nil")
	}
	if frame.Opus.Codec != "opus" {
		t.Fatalf("codec = %q, want opus", frame.Opus.Codec)
	}
	if frame.Opus.BinaryVersion != 1 {
		t.Fatalf("binary version = %d, want 1", frame.Opus.BinaryVersion)
	}
	if frame.Opus.PayloadBytes != 4 {
		t.Fatalf("payload bytes = %d, want 4", frame.Opus.PayloadBytes)
	}
	if frame.Opus.Payload[0] != 0xf8 {
		t.Fatalf("payload was not copied before caller mutation: %#v", frame.Opus.Payload)
	}
	if frame.TraceID != "a21-trace-opus-001" || frame.SessionID != "a21-session-opus-001" || frame.DeviceID != "stackchan-001" {
		t.Fatalf("identity = trace:%q session:%q device:%q", frame.TraceID, frame.SessionID, frame.DeviceID)
	}
}

func TestParseBinaryFrameVersion2UnwrapsTimestampedOpusPayload(t *testing.T) {
	payload := []byte{0x11, 0x22, 0x33}
	wire := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint16(wire[0:2], 2)
	binary.BigEndian.PutUint16(wire[2:4], 0)
	binary.BigEndian.PutUint32(wire[8:12], 1234)
	binary.BigEndian.PutUint32(wire[12:16], uint32(len(payload)))
	copy(wire[16:], payload)

	frame, err := ParseBinaryFrameVersion(wire, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Opus.BinaryVersion != 2 || frame.Opus.TimestampMS != 1234 || frame.Opus.PayloadBytes != len(payload) {
		t.Fatalf("opus metadata = %+v, want version=2 timestamp=1234 bytes=%d", frame.Opus, len(payload))
	}
	if string(frame.Opus.Payload) != string(payload) {
		t.Fatalf("payload = %#v, want %#v", frame.Opus.Payload, payload)
	}
}

func TestParseBinaryFrameVersion3UnwrapsCompactOpusPayload(t *testing.T) {
	payload := []byte{0xaa, 0xbb}
	wire := make([]byte, 4+len(payload))
	wire[0] = 0
	wire[1] = 0
	binary.BigEndian.PutUint16(wire[2:4], uint16(len(payload)))
	copy(wire[4:], payload)

	frame, err := ParseBinaryFrameVersion(wire, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Opus.BinaryVersion != 3 || frame.Opus.PayloadBytes != len(payload) || string(frame.Opus.Payload) != string(payload) {
		t.Fatalf("opus = %+v, want version=3 bytes=%d payload=%#v", frame.Opus, len(payload), payload)
	}
}

func TestParseBinaryFrameRejectsEmptyPayloadAndUnexpectedDirection(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		direction Direction
		want      error
	}{
		{name: "empty payload", payload: nil, direction: DirectionDeviceToServer, want: ErrEmptyBinaryPayload},
		{name: "unexpected direction", payload: []byte{0x01}, direction: Direction("sideways"), want: ErrUnexpectedBinaryDirection},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBinaryFrame(tc.payload, tc.direction, Identity{DeviceID: "stackchan-001"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestParseBinaryFrameRejectsMalformedWrappedPayload(t *testing.T) {
	_, err := ParseBinaryFrameVersion([]byte{0x00, 0x02}, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 2)
	if !errors.Is(err, ErrMalformedBinaryFrame) {
		t.Fatalf("short v2 err = %v, want %v", err, ErrMalformedBinaryFrame)
	}

	wire := make([]byte, 5)
	wire[0] = 1
	binary.BigEndian.PutUint16(wire[2:4], 1)
	wire[4] = 0xff
	_, err = ParseBinaryFrameVersion(wire, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 3)
	if !errors.Is(err, ErrUnsupportedBinaryFrameType) {
		t.Fatalf("unsupported type err = %v, want %v", err, ErrUnsupportedBinaryFrameType)
	}
}

func TestParseBinaryFrameRejectsUnsupportedBinaryVersion(t *testing.T) {
	_, err := ParseBinaryFrameVersion([]byte{0x01}, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 4)
	if !errors.Is(err, ErrUnsupportedBinaryProtocol) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedBinaryProtocol)
	}
}

func TestBuildServerHelloKeepsStockProfileFreeOfDebugExtensions(t *testing.T) {
	data, err := BuildServerHello(Identity{
		DeviceID:  "stackchan-001",
		TraceID:   "a21-trace-server-hello",
		SessionID: "a21-session-server-hello",
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(data)
	for _, forbidden := range []string{"debug_metrics", "device_events"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("server hello leaked %q: %s", forbidden, payload)
		}
	}

	var hello map[string]any
	if err := json.Unmarshal(data, &hello); err != nil {
		t.Fatal(err)
	}
	if hello["type"] != "hello" || hello["transport"] != "websocket" || hello["version"] != float64(3) {
		t.Fatalf("hello = %#v, want stock xiaozhi server hello", hello)
	}
	audio, ok := hello["audio_params"].(map[string]any)
	if !ok {
		t.Fatalf("audio_params = %#v, want object", hello["audio_params"])
	}
	if audio["format"] != "opus" || audio["sample_rate"] != float64(24000) || audio["frame_duration"] != float64(60) {
		t.Fatalf("audio_params = %#v, want 24kHz mono Opus 60ms", audio)
	}
}

func TestBuildAndParseMCPJSONRPCEnvelope(t *testing.T) {
	initData, err := BuildMCPInitializeRequest("a21-mcp-001", "a21-xiaozhi-transport")
	if err != nil {
		t.Fatal(err)
	}
	initEnvelope, err := ParseMCPEnvelope(initData)
	if err != nil {
		t.Fatal(err)
	}
	if initEnvelope.JSONRPC != "2.0" || initEnvelope.Method != MCPMethodInitialize {
		t.Fatalf("initialize envelope = %+v, want jsonrpc 2.0 initialize", initEnvelope)
	}

	listData, err := BuildMCPToolsListRequest("a21-mcp-002")
	if err != nil {
		t.Fatal(err)
	}
	listEnvelope, err := ParseMCPEnvelope(listData)
	if err != nil {
		t.Fatal(err)
	}
	if listEnvelope.Method != MCPMethodToolsList {
		t.Fatalf("tools/list method = %q, want %q", listEnvelope.Method, MCPMethodToolsList)
	}

	callData, err := BuildMCPToolsCallRequest("a21-mcp-003", "display.set_emotion", map[string]any{
		"emotion":       "listening",
		"api_key":       "sk-a21-secret",
		"transcript":    "raw office words",
		"audio_base64":  "AAAA",
		"safe_metadata": "visible",
	})
	if err != nil {
		t.Fatal(err)
	}
	callPayload := string(callData)
	for _, leaked := range []string{"sk-a21-secret", "raw office words", "AAAA"} {
		if strings.Contains(callPayload, leaked) {
			t.Fatalf("tools/call leaked sensitive argument value %q: %s", leaked, callPayload)
		}
	}
	if !strings.Contains(callPayload, `"safe_metadata":"visible"`) {
		t.Fatalf("tools/call dropped safe metadata: %s", callPayload)
	}
	callEnvelope, err := ParseMCPEnvelope(callData)
	if err != nil {
		t.Fatal(err)
	}
	if callEnvelope.Method != MCPMethodToolsCall || callEnvelope.ToolName != "display.set_emotion" {
		t.Fatalf("tools/call envelope = %+v, want sanitized tool call", callEnvelope)
	}
}

func TestMCPRejectsInvalidToolNamesAndLegacyIdentity(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want error
	}{
		{name: "empty", tool: "", want: ErrInvalidMCPToolName},
		{name: "space", tool: "display set", want: ErrInvalidMCPToolName},
		{name: "legacy x21", tool: "x21.display.set", want: ErrLegacyIdentity},
		{name: "legacy v21", tool: "v21.query", want: ErrLegacyIdentity},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildMCPToolsCallRequest("a21-mcp-bad", tc.tool, map[string]any{"emotion": "idle"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}

	_, err := ParseMCPEnvelope([]byte(`{"jsonrpc":"2.0","id":"a21-mcp-bad","method":"tools/call","params":{"name":"x21.display","arguments":{}}}`))
	if !errors.Is(err, ErrLegacyIdentity) {
		t.Fatalf("parse err = %v, want %v", err, ErrLegacyIdentity)
	}
}

func TestBuildLLMEmotionMessageMapsA21StatesWithoutLegacyNames(t *testing.T) {
	tests := map[string]string{
		"idle":         "idle",
		"listening":    "listening",
		"thinking":     "thinking",
		"speaking":     "speaking",
		"interrupted":  "interrupted",
		"professional": "professional",
		"error":        "error",
		"":             "idle",
	}
	for state, wantEmotion := range tests {
		t.Run("state_"+state, func(t *testing.T) {
			data, err := BuildLLMEmotionMessage(Identity{
				DeviceID:  "stackchan-001",
				TraceID:   "a21-trace-emotion",
				SessionID: "a21-session-emotion",
			}, state)
			if err != nil {
				t.Fatal(err)
			}
			payload := string(data)
			if strings.Contains(strings.ToLower(payload), "x21") || strings.Contains(strings.ToLower(payload), "v21") {
				t.Fatalf("emotion payload leaked legacy naming: %s", payload)
			}
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatal(err)
			}
			if msg["type"] != "llm" || msg["emotion"] != wantEmotion {
				t.Fatalf("message = %#v, want llm emotion %q", msg, wantEmotion)
			}
		})
	}

	_, err := BuildLLMEmotionMessage(Identity{DeviceID: "x21-stackchan"}, "listening")
	if !errors.Is(err, ErrLegacyIdentity) {
		t.Fatalf("err = %v, want %v", err, ErrLegacyIdentity)
	}
}

func TestClampYAngleKeepsServoContractInsideStockRange(t *testing.T) {
	tests := map[int]int{
		-30: 5,
		0:   5,
		5:   5,
		42:  42,
		85:  85,
		120: 85,
	}
	for input, want := range tests {
		if got := ClampYAngle(input); got != want {
			t.Fatalf("ClampYAngle(%d) = %d, want %d", input, got, want)
		}
	}
	params := BuildMotionParams(120)
	if params["y_angle"] != 85 {
		t.Fatalf("motion params = %#v, want y_angle clamped to 85", params)
	}
}
