package protocol

import "encoding/json"

const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
	KindControlEvent     Kind = "control.event"
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

type ExpressionState string

const (
	ExpressionIdle         ExpressionState = "idle"
	ExpressionListening    ExpressionState = "listening"
	ExpressionThinking     ExpressionState = "thinking"
	ExpressionSpeaking     ExpressionState = "speaking"
	ExpressionInterrupted  ExpressionState = "interrupted"
	ExpressionProfessional ExpressionState = "professional"
	ExpressionError        ExpressionState = "error"
)

type ControlEventPayload struct {
	State        ExpressionState `json:"state"`
	Mode         Mode            `json:"mode"`
	Text         string          `json:"text,omitempty"`
	Final        bool            `json:"final,omitempty"`
	StreamID     string          `json:"stream_id,omitempty"`
	Confidence   float64         `json:"confidence,omitempty"`
	Evidence     []EvidenceItem  `json:"evidence,omitempty"`
	SpeechBlocks []string        `json:"speech_blocks,omitempty"`
	ScreenCards  []ScreenCard    `json:"screen_cards,omitempty"`
	FollowUps    []string        `json:"follow_ups,omitempty"`
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
	DeviceEventMockTurn          DeviceEventKind = "mock.turn"
	DeviceEventInterrupt         DeviceEventKind = "interrupt"
	DeviceEventTouchWakeOrListen DeviceEventKind = "touch.wake_or_listen"
	DeviceEventTouchBargeIn      DeviceEventKind = "touch.barge_in"
)

type TouchSource string

const (
	TouchSourceScreen    TouchSource = "screen"
	TouchSourceTopSensor TouchSource = "top_sensor"
)

type DeviceEventPayload struct {
	Event           DeviceEventKind `json:"event"`
	Mode            Mode            `json:"mode,omitempty"`
	Text            string          `json:"text,omitempty"`
	TouchSource     TouchSource     `json:"touch_source,omitempty"`
	FirmwareID      string          `json:"firmware_id,omitempty"`
	FirmwareVersion string          `json:"firmware_version,omitempty"`
	FirmwareBoard   string          `json:"firmware_board,omitempty"`
	FirmwareCommit  string          `json:"firmware_commit,omitempty"`
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
