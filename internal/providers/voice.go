package providers

import (
	"context"
	"errors"

	"a21.local/a21/internal/protocol"
)

var ErrVoiceProviderUnavailable = errors.New("a21 voice provider unavailable")

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
	Session  VoiceSession
	Reason   VoiceCancelReason
	StreamID string
}

type VoiceEventKind string

const (
	VoiceEventThinking  VoiceEventKind = "thinking"
	VoiceEventSpeaking  VoiceEventKind = "speaking"
	VoiceEventCancelled VoiceEventKind = "cancelled"
	VoiceEventError     VoiceEventKind = "error"
)

type VoiceEvent struct {
	Session      VoiceSession
	Kind         VoiceEventKind
	Text         string
	Final        bool
	StreamID     string
	Audio        *VoiceAudioChunk
	CancelReason VoiceCancelReason
}

type VoiceAudioChunk struct {
	Codec        string
	SampleRateHz int
	Channels     int
	DurationMS   int
	DataBase64   string
	Finding      string `json:"finding,omitempty"`
	Err          error  `json:"-"`
}

type VoiceProviderHealthStatus string

const (
	VoiceProviderHealthy     VoiceProviderHealthStatus = "healthy"
	VoiceProviderDegraded    VoiceProviderHealthStatus = "degraded"
	VoiceProviderUnavailable VoiceProviderHealthStatus = "unavailable"
)

type VoiceProviderHealth struct {
	Provider       string                    `json:"provider"`
	Status         VoiceProviderHealthStatus `json:"status"`
	Configured     bool                      `json:"configured"`
	Realtime       bool                      `json:"realtime"`
	ActiveProvider string                    `json:"active_provider,omitempty"`
	Detail         string                    `json:"detail,omitempty"`
}

type VoiceProvider interface {
	Name() string
	StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error)
	Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error)
	Health(ctx context.Context) (VoiceProviderHealth, error)
	Close(ctx context.Context) error
}

type RealtimeVoiceProvider interface {
	StartRealtimeSession(ctx context.Context, session VoiceSession) (RealtimeVoiceSession, error)
}

type RealtimeVoiceSession interface {
	SendAudio(ctx context.Context, chunk protocol.AudioChunk) error
	CommitAndCreateResponse(ctx context.Context) error
	Cancel(ctx context.Context, req VoiceCancelRequest) error
	Events() <-chan VoiceEvent
	Close(ctx context.Context) error
}
