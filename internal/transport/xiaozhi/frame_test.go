package xiaozhi

import (
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
		"device_id": "stackchan-001",
		"audio": {
			"format": "opus",
			"sample_rate": 16000,
			"channels": 1,
			"frame_duration": 60,
			"binary_protocol_version": 1
		}
	}`), DirectionDeviceToServer, Identity{})
	if err != nil {
		t.Fatal(err)
	}
	if frame.Control.Hello.Transport != "websocket" {
		t.Fatalf("transport = %q, want websocket default", frame.Control.Hello.Transport)
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

func TestParseBinaryFrameRejectsUnsupportedBinaryVersion(t *testing.T) {
	_, err := ParseBinaryFrameVersion([]byte{0x01}, DirectionDeviceToServer, Identity{DeviceID: "stackchan-001"}, 2)
	if !errors.Is(err, ErrUnsupportedBinaryProtocol) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedBinaryProtocol)
	}
}
