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
	ModeProfessional  Mode = "professional"
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
	State    ExpressionState `json:"state"`
	Mode     Mode            `json:"mode"`
	Text     string          `json:"text,omitempty"`
	Final    bool            `json:"final,omitempty"`
	StreamID string          `json:"stream_id,omitempty"`
}
