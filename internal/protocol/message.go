package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame         Kind = "audio.frame"
	KindAudioPlaybackChunk Kind = "audio.playback.chunk"
	KindDeviceEvent        Kind = "device.event"
	KindAssistantState     Kind = "assistant.state"
	KindScreenExpression   Kind = "screen.expression"
	KindMotionCommand      Kind = "motion.command"
	KindControlEvent       Kind = "control.event"
)

type Envelope struct {
	Protocol  string          `json:"protocol"`
	DeviceID  string          `json:"device_id"`
	Kind      Kind            `json:"kind"`
	Seq       uint64          `json:"seq"`
	TraceID   string          `json:"trace_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	SentAtMS  int64           `json:"sent_at_ms,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Mode string

const (
	ModeDialogue      Mode = "dialogue"
	ModeWorkmate      Mode = "workmate"
	ModeCompanion     Mode = "companion"
	ModeCoCreation    Mode = "co_creation"
	ModeRoleplay      Mode = "roleplay"
	ModeProfessional  Mode = "professional"
	ModeFocus         Mode = "focus"
	ModePublic        Mode = "public"
	ModePrivate       Mode = "private"
	ModeMuted         Mode = "muted"
	ModeLocalFallback Mode = "local_fallback"
	ModeError         Mode = "error"
)

func NormalizeProductMode(mode Mode) Mode {
	switch mode {
	case ModeProfessional:
		return ModeProfessional
	case ModeLocalFallback:
		return ModeLocalFallback
	case ModeError:
		return ModeError
	case ModeDialogue, ModeWorkmate, ModeCompanion, ModeCoCreation, ModeRoleplay, ModeFocus, ModePublic, ModePrivate, ModeMuted, "":
		return ModeDialogue
	default:
		return ModeDialogue
	}
}

func CanonicalProductModes() []Mode {
	return []Mode{ModeDialogue, ModeProfessional}
}

type ExpressionState string

const (
	ExpressionIdle          ExpressionState = "idle"
	ExpressionListening     ExpressionState = "listening"
	ExpressionThinking      ExpressionState = "thinking"
	ExpressionSpeaking      ExpressionState = "speaking"
	ExpressionInterrupted   ExpressionState = "interrupted"
	ExpressionProfessional  ExpressionState = "professional"
	ExpressionLocalFallback ExpressionState = "local_fallback"
	ExpressionError         ExpressionState = "error"
)

type ControlEventPayload struct {
	State                    ExpressionState `json:"state"`
	Mode                     Mode            `json:"mode"`
	Text                     string          `json:"text,omitempty"`
	Final                    bool            `json:"final,omitempty"`
	StreamID                 string          `json:"stream_id,omitempty"`
	DiagnosticToneHz         int             `json:"diagnostic_tone_hz,omitempty"`
	DiagnosticToneDurationMS int             `json:"diagnostic_tone_duration_ms,omitempty"`
	DiagnosticToneVolume     int             `json:"diagnostic_tone_volume,omitempty"`
	Confidence               float64         `json:"confidence,omitempty"`
	Evidence                 []EvidenceItem  `json:"evidence,omitempty"`
	SpeechBlocks             []string        `json:"speech_blocks,omitempty"`
	ScreenCards              []ScreenCard    `json:"screen_cards,omitempty"`
	FollowUps                []string        `json:"follow_ups,omitempty"`
}

type EvidenceItem struct {
	Title    string `json:"title"`
	Type     string `json:"type"`
	SourceID string `json:"source_id"`
	Summary  string `json:"summary"`
	Quote    string `json:"quote,omitempty"`
}

type ScreenCard struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type DeviceEventKind string

const (
	DeviceEventMockTurn              DeviceEventKind = "mock.turn"
	DeviceEventInterrupt             DeviceEventKind = "interrupt"
	DeviceEventTouchWakeOrListen     DeviceEventKind = "touch.wake_or_listen"
	DeviceEventTouchBargeIn          DeviceEventKind = "touch.barge_in"
	DeviceEventTouchTopTap           DeviceEventKind = "touch.top.tap"
	DeviceEventTouchTopSwipeForward  DeviceEventKind = "touch.top.swipe_forward"
	DeviceEventTouchTopSwipeBackward DeviceEventKind = "touch.top.swipe_backward"
	DeviceEventRuntimeEcho           DeviceEventKind = "runtime.echo"
)

type TouchSource string

const (
	TouchSourceScreen    TouchSource = "screen"
	TouchSourceTopSensor TouchSource = "top_sensor"
)

func StackChanHardwareCapabilityKeys() []string {
	return []string{
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
}

type DeviceEventPayload struct {
	Event           DeviceEventKind   `json:"event"`
	Mode            Mode              `json:"mode,omitempty"`
	Text            string            `json:"text,omitempty"`
	TouchSource     TouchSource       `json:"touch_source,omitempty"`
	FirmwareID      string            `json:"firmware_id,omitempty"`
	FirmwareVersion string            `json:"firmware_version,omitempty"`
	FirmwareBoard   string            `json:"firmware_board,omitempty"`
	FirmwareCommit  string            `json:"firmware_commit,omitempty"`
	Capabilities    map[string]string `json:"capabilities,omitempty"`
	RuntimeEcho     map[string]string `json:"runtime_echo,omitempty"`
}

type AudioCodec string

const (
	AudioCodecPCMS16LE AudioCodec = "pcm_s16le"
	AudioCodecOpus     AudioCodec = "opus"
)

type AudioChunk struct {
	Codec              AudioCodec `json:"codec"`
	SampleRateHz       int        `json:"sample_rate_hz"`
	Channels           int        `json:"channels"`
	DurationMS         int        `json:"duration_ms"`
	CaptureStartedAtMS int64      `json:"capture_started_at_ms,omitempty"`
	CaptureEndedAtMS   int64      `json:"capture_ended_at_ms,omitempty"`
	DataBase64         string     `json:"data_base64"`
}

func ValidateAudioChunk(chunk AudioChunk) error {
	return validatePCM16Audio(chunk.Codec, chunk.SampleRateHz, chunk.Channels, chunk.DurationMS, chunk.DataBase64)
}

func ValidateAudioPlaybackChunk(chunk AudioPlaybackChunk) error {
	if chunk.StreamID == "" {
		return fmt.Errorf("audio playback stream_id is required")
	}
	return validatePCM16Audio(chunk.Codec, chunk.SampleRateHz, chunk.Channels, chunk.DurationMS, chunk.DataBase64)
}

func validatePCM16Audio(codec AudioCodec, sampleRateHz int, channels int, durationMS int, dataBase64 string) error {
	if codec != AudioCodecPCMS16LE {
		return fmt.Errorf("audio codec %q is unsupported", codec)
	}
	switch sampleRateHz {
	case 16000, 24000, 48000:
	default:
		return fmt.Errorf("audio sample_rate_hz %d is unsupported", sampleRateHz)
	}
	if channels != 1 {
		return fmt.Errorf("audio channels %d is unsupported", channels)
	}
	switch durationMS {
	case 20, 40:
	default:
		return fmt.Errorf("audio duration_ms %d is unsupported", durationMS)
	}
	if dataBase64 == "" {
		return fmt.Errorf("audio data_base64 is required")
	}
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return fmt.Errorf("audio data_base64 is invalid: %w", err)
	}
	expectedBytes := sampleRateHz * durationMS * channels * 2 / 1000
	if len(data) != expectedBytes {
		return fmt.Errorf("audio pcm byte length %d does not match expected %d", len(data), expectedBytes)
	}
	return nil
}

type AudioPlaybackChunk struct {
	StreamID     string     `json:"stream_id"`
	Codec        AudioCodec `json:"codec"`
	SampleRateHz int        `json:"sample_rate_hz"`
	Channels     int        `json:"channels"`
	DurationMS   int        `json:"duration_ms"`
	DataBase64   string     `json:"data_base64"`
}
