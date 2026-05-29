package providers

import "context"

type VoiceSession struct {
	TraceID   string
	SessionID string
	DeviceID  string
}

type VoiceTurnRequest struct {
	Session VoiceSession
	Text    string
	Mode    string
}

type VoiceCancelReason string

const (
	CancelBargeIn VoiceCancelReason = "barge_in"
	CancelError   VoiceCancelReason = "error"
)

type VoiceCancelRequest struct {
	Session VoiceSession
	Reason  VoiceCancelReason
}

type VoiceEventKind string

const (
	VoiceEventThinking  VoiceEventKind = "thinking"
	VoiceEventSpeaking  VoiceEventKind = "speaking"
	VoiceEventCancelled VoiceEventKind = "cancelled"
	VoiceEventError     VoiceEventKind = "error"
)

type VoiceEvent struct {
	Session  VoiceSession
	Kind     VoiceEventKind
	Text     string
	Final    bool
	StreamID string
}

type VoiceProvider interface {
	Name() string
	StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error)
	Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error)
	Close(ctx context.Context) error
}
