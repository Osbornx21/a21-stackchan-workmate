package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Server struct {
	mu                         sync.Mutex
	next                       uint64
	now                        func() time.Time
	metrics                    *metrics
	voice                      providers.VoiceProvider
	v21                        v21adapter.Client
	v21TTL                     time.Duration
	devices                    map[string]DeviceRecord
	traces                     map[string][]TraceEvent
	audioStreams               map[string]string
	activeStreams              map[string]string
	realtimeAudio              map[string]providers.RealtimeVoiceSession
	realtimeAudioCommitAt      map[string]time.Time
	realtimeAudioFirstDownlink map[string]bool
	audioProbeSessions         map[string]bool
	mockPlaybackArmedSessions  map[string]bool
	realtimeArmedSessions      map[string]bool
	audioIngress               *audio.Ingress
	audioSockets               map[string]*deviceSocket
	audioCaptureFrames         []AudioCaptureFrame
}

type ServerOptions struct {
	VoiceProvider providers.VoiceProvider
	V21Client     v21adapter.Client
	V21Timeout    time.Duration
}

type MockTurnRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type MockTurnResponse struct {
	TraceID   string              `json:"trace_id"`
	SessionID string              `json:"session_id"`
	DeviceID  string              `json:"device_id"`
	Events    []protocol.Envelope `json:"events"`
}

type RealtimeSessionRequest struct {
	DeviceID  string        `json:"device_id"`
	Text      string        `json:"text,omitempty"`
	Mode      protocol.Mode `json:"mode,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
}

type RealtimeSessionCancelRequest struct {
	DeviceID  string                      `json:"device_id"`
	Mode      protocol.Mode               `json:"mode,omitempty"`
	TraceID   string                      `json:"trace_id,omitempty"`
	SessionID string                      `json:"session_id,omitempty"`
	StreamID  string                      `json:"stream_id,omitempty"`
	Reason    providers.VoiceCancelReason `json:"reason,omitempty"`
}

type RealtimeSessionResponse struct {
	TraceID   string              `json:"trace_id"`
	SessionID string              `json:"session_id"`
	DeviceID  string              `json:"device_id"`
	Provider  string              `json:"provider"`
	Status    string              `json:"status"`
	Events    []protocol.Envelope `json:"events"`
}

type FastCompanionLocalAudioResult struct {
	ASRProvider          string `json:"asr_provider,omitempty"`
	FirstPartialMS       int64  `json:"first_partial_ms,omitempty"`
	FinalTranscriptChars int    `json:"final_transcript_chars,omitempty"`
}

type FastCompanionTurnRequest struct {
	DeviceID   string                        `json:"device_id"`
	Mode       protocol.Mode                 `json:"mode,omitempty"`
	TraceID    string                        `json:"trace_id,omitempty"`
	SessionID  string                        `json:"session_id,omitempty"`
	LocalAudio FastCompanionLocalAudioResult `json:"local_audio,omitempty"`
}

type FastCompanionTurnResponse struct {
	TraceID            string              `json:"trace_id"`
	SessionID          string              `json:"session_id"`
	DeviceID           string              `json:"device_id"`
	Mode               protocol.Mode       `json:"mode"`
	Status             string              `json:"status"`
	Route              string              `json:"route"`
	AudioFrontend      string              `json:"audio_frontend"`
	TextStreamProvider string              `json:"text_stream_provider"`
	ProviderFamily     string              `json:"provider_family"`
	TextStreamExecuted bool                `json:"text_stream_executed"`
	Events             []protocol.Envelope `json:"events"`
}

type DeviceControlRequest struct {
	DeviceID                     string                        `json:"device_id"`
	State                        protocol.ExpressionState      `json:"state,omitempty"`
	Mode                         protocol.Mode                 `json:"mode,omitempty"`
	Text                         string                        `json:"text,omitempty"`
	TraceID                      string                        `json:"trace_id,omitempty"`
	SessionID                    string                        `json:"session_id,omitempty"`
	StreamID                     string                        `json:"stream_id,omitempty"`
	DiagnosticToneHz             int                           `json:"diagnostic_tone_hz,omitempty"`
	DiagnosticToneDurationMS     int                           `json:"diagnostic_tone_duration_ms,omitempty"`
	DiagnosticToneVolume         int                           `json:"diagnostic_tone_volume,omitempty"`
	MockAudioChunks              int                           `json:"mock_audio_chunks,omitempty"`
	AudioChunks                  []protocol.AudioPlaybackChunk `json:"audio_chunks,omitempty"`
	AudioProbeOnly               bool                          `json:"audio_probe_only,omitempty"`
	MockPlaybackOnNextAudioFrame bool                          `json:"mock_playback_on_next_audio_frame,omitempty"`
	RealtimeOnNextSpeech         bool                          `json:"realtime_on_next_speech,omitempty"`
}

type DeviceControlResponse struct {
	TraceID            string              `json:"trace_id"`
	SessionID          string              `json:"session_id"`
	DeviceID           string              `json:"device_id"`
	Status             string              `json:"status"`
	DeliveredTransport string              `json:"delivered_transport"`
	Events             []protocol.Envelope `json:"events"`
}

type deviceSocket struct {
	conn      *websocket.Conn
	writeMu   *sync.Mutex
	connected int64
}

type realtimeVoiceOutput struct {
	Control protocol.ControlEventPayload
	Audio   *providers.VoiceAudioChunk
}

type DeviceFirmwareIdentity struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Board   string `json:"board,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type DeviceRecord struct {
	DeviceID         string                   `json:"device_id"`
	Firmware         DeviceFirmwareIdentity   `json:"firmware,omitempty"`
	Capabilities     map[string]string        `json:"capabilities,omitempty"`
	RuntimeEcho      map[string]string        `json:"runtime_echo,omitempty"`
	IdentityStatus   string                   `json:"identity_status"`
	IdentityError    string                   `json:"identity_error,omitempty"`
	ConnectionStatus string                   `json:"connection_status,omitempty"`
	DeviceAgeMS      int64                    `json:"device_age_ms,omitempty"`
	CurrentMode      protocol.Mode            `json:"current_mode,omitempty"`
	CurrentExpr      protocol.ExpressionState `json:"current_expression,omitempty"`
	PlaybackStream   string                   `json:"playback_stream_id,omitempty"`
	LastEvent        protocol.DeviceEventKind `json:"last_event,omitempty"`
	LastTouchSource  protocol.TouchSource     `json:"last_touch_source,omitempty"`
	LastSeq          uint64                   `json:"last_seq,omitempty"`
	LastTraceID      string                   `json:"last_trace_id,omitempty"`
	LastSessionID    string                   `json:"last_session_id,omitempty"`
	FirstSeenMS      int64                    `json:"first_seen_ms"`
	LastSeenMS       int64                    `json:"last_seen_ms"`
}

type DeviceRegistryResponse struct {
	SchemaVersion string         `json:"schema_version"`
	Service       string         `json:"service"`
	Devices       []DeviceRecord `json:"devices"`
}

const (
	DeviceRegistrySchemaVersion = "a21.gateway.devices.v1"
	DeviceRegistryServiceName   = "a21-gateway"
	AudioRecentSchemaVersion    = "a21.gateway.audio_recent.v1"
	maxAudioCaptureFrames       = 512
)

type AudioRecentResponse struct {
	SchemaVersion string              `json:"schema_version"`
	DeviceID      string              `json:"device_id,omitempty"`
	TraceID       string              `json:"trace_id,omitempty"`
	SessionID     string              `json:"session_id,omitempty"`
	IncludeAudio  bool                `json:"include_audio"`
	Frames        []AudioCaptureFrame `json:"frames"`
}

type AudioCaptureFrame struct {
	DeviceID            string  `json:"device_id"`
	TraceID             string  `json:"trace_id"`
	SessionID           string  `json:"session_id"`
	Seq                 uint64  `json:"seq"`
	SentAtMS            int64   `json:"sent_at_ms,omitempty"`
	ReceivedAtMS        int64   `json:"received_at_ms"`
	SampleRateHz        int     `json:"sample_rate_hz"`
	Channels            int     `json:"channels"`
	DurationMS          int     `json:"duration_ms"`
	CaptureStartedAtMS  int64   `json:"capture_started_at_ms,omitempty"`
	CaptureEndedAtMS    int64   `json:"capture_ended_at_ms,omitempty"`
	DataBytes           int     `json:"data_bytes"`
	DataBase64          string  `json:"data_base64,omitempty"`
	RMS                 float64 `json:"rms"`
	VADDetector         string  `json:"vad_detector"`
	SpeechDetected      bool    `json:"speech_detected"`
	SpeechActive        bool    `json:"speech_active"`
	DroppedFrames       int     `json:"dropped_frames,omitempty"`
	DroppedFrameDelta   int     `json:"dropped_frame_delta,omitempty"`
	IngressBufferFrames int     `json:"ingress_buffer_frames"`
}

type TraceEvent struct {
	Name      string `json:"name"`
	TraceID   string `json:"trace_id"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	AtMS      int64  `json:"at_ms"`
	OffsetMS  int64  `json:"offset_ms"`
}

type TraceLatencySummary struct {
	EventCount                   int    `json:"event_count"`
	LastOffsetMS                 int64  `json:"last_offset_ms"`
	AudioFrameToPlaybackMS       *int64 `json:"audio_frame_to_playback_ms,omitempty"`
	V21QueryFirstResultMS        *int64 `json:"v21_query_first_result_ms,omitempty"`
	BargeInStopMS                *int64 `json:"barge_in_stop_ms,omitempty"`
	ProviderCommitToFirstAudioMS *int64 `json:"provider_commit_to_first_audio_ms,omitempty"`
}

type TraceResponse struct {
	TraceID string              `json:"trace_id"`
	Events  []TraceEvent        `json:"events"`
	Summary TraceLatencySummary `json:"summary"`
}

func NewServer() *Server {
	return NewServerWithOptions(ServerOptions{})
}

func NewServerWithOptions(options ServerOptions) *Server {
	voiceProvider := options.VoiceProvider
	if voiceProvider == nil {
		voiceProvider = providers.NewMockVoiceProvider()
	}
	v21Client := options.V21Client
	if v21Client == nil {
		v21Client = v21adapter.NewMockClient()
	}
	v21TTL := options.V21Timeout
	if v21TTL <= 0 {
		v21TTL = 3 * time.Second
	}
	return &Server{
		now:                        time.Now,
		metrics:                    newMetrics(),
		voice:                      voiceProvider,
		v21:                        v21Client,
		v21TTL:                     v21TTL,
		devices:                    make(map[string]DeviceRecord),
		traces:                     make(map[string][]TraceEvent),
		audioStreams:               make(map[string]string),
		activeStreams:              make(map[string]string),
		realtimeAudio:              make(map[string]providers.RealtimeVoiceSession),
		realtimeAudioCommitAt:      make(map[string]time.Time),
		realtimeAudioFirstDownlink: make(map[string]bool),
		audioProbeSessions:         make(map[string]bool),
		mockPlaybackArmedSessions:  make(map[string]bool),
		realtimeArmedSessions:      make(map[string]bool),
		audioIngress:               audio.NewIngress(audio.DefaultIngressConfig()),
		audioSockets:               make(map[string]*deviceSocket),
		audioCaptureFrames:         make([]AudioCaptureFrame, 0, maxAudioCaptureFrames),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/simulator", s.handleSimulator)
	mux.Handle("/metrics", s.metrics.handler())
	mux.HandleFunc("/v1/devices", s.handleDevices)
	mux.HandleFunc("/v1/devices/control", s.handleDeviceControl)
	mux.HandleFunc("/v1/audio/recent", s.handleAudioRecent)
	mux.HandleFunc("/v1/traces", s.handleTraces)
	mux.HandleFunc("/v1/providers/voice/health", s.handleVoiceProviderHealth)
	mux.HandleFunc("/v1/realtime/session", s.handleRealtimeSessionStart)
	mux.HandleFunc("/v1/realtime/session/cancel", s.handleRealtimeSessionCancel)
	mux.HandleFunc("/v1/fast-companion/turn", s.handleFastCompanionTurn)
	mux.HandleFunc("/v1/mock-turn", s.handleMockTurn)
	mux.HandleFunc("/v1/mock-interrupt", s.handleMockInterrupt)
	mux.HandleFunc("/ws/control", s.handleControlWS)
	mux.HandleFunc("/ws/audio", s.handleAudioWS)
	return mux
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, DeviceRegistryResponse{
		SchemaVersion: DeviceRegistrySchemaVersion,
		Service:       DeviceRegistryServiceName,
		Devices:       s.deviceRecords(),
	})
}

func (s *Server) handleDeviceControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.State == "" {
		req.State = protocol.ExpressionListening
	}
	if req.StreamID == "" && req.State == protocol.ExpressionSpeaking {
		req.StreamID = "a21-device-command-stream-000001"
	}
	if req.MockAudioChunks < 0 || req.MockAudioChunks > 8 {
		http.Error(w, "mock_audio_chunks must be between 0 and 8", http.StatusBadRequest)
		return
	}
	if len(req.AudioChunks) > 8 {
		http.Error(w, "audio_chunks must contain at most 8 chunks", http.StatusBadRequest)
		return
	}
	if (req.DiagnosticToneHz != 0 || req.DiagnosticToneDurationMS != 0 || req.DiagnosticToneVolume != 0) &&
		(req.DiagnosticToneHz < 50 || req.DiagnosticToneHz > 8000 ||
			req.DiagnosticToneDurationMS < 1 || req.DiagnosticToneDurationMS > 10000 ||
			req.DiagnosticToneVolume < 1 || req.DiagnosticToneVolume > 255) {
		http.Error(w, "diagnostic tone must set hz 50-8000, duration 1-10000ms, and volume 1-255", http.StatusBadRequest)
		return
	}
	for i := range req.AudioChunks {
		if req.AudioChunks[i].StreamID == "" {
			req.AudioChunks[i].StreamID = req.StreamID
		}
		if req.AudioChunks[i].StreamID != req.StreamID {
			http.Error(w, "audio_chunks stream_id must match request stream_id", http.StatusBadRequest)
			return
		}
		if err := protocol.ValidateAudioPlaybackChunk(req.AudioChunks[i]); err != nil {
			http.Error(w, "invalid audio_chunks payload", http.StatusBadRequest)
			return
		}
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.setAudioProbeOnly(req.DeviceID, traceID, sessionID, req.AudioProbeOnly)
	s.setMockPlaybackOnNextAudioFrame(req.DeviceID, traceID, sessionID, req.MockPlaybackOnNextAudioFrame)
	s.setRealtimeOnNextSpeech(req.DeviceID, traceID, sessionID, req.RealtimeOnNextSpeech)
	socket, ok := s.audioSocket(req.DeviceID)
	if !ok {
		http.Error(w, "device audio websocket is not connected", http.StatusConflict)
		return
	}

	events := s.deviceControlEvents(req)
	for _, event := range events {
		if err := writeAudioEnvelope(r.Context(), socket.conn, socket.writeMu, event); err != nil {
			http.Error(w, "device command delivery failed", http.StatusBadGateway)
			return
		}
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "device.command.delivered", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, DeviceControlResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "audio_ws",
		Events:             events,
	})
}

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}
	events := s.traceEvents(traceID)
	writeJSON(w, http.StatusOK, TraceResponse{TraceID: traceID, Events: events, Summary: traceLatencySummary(events)})
}

func (s *Server) handleAudioRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !loopbackHTTPRemote(r.RemoteAddr) {
		http.Error(w, "audio capture is available only from loopback", http.StatusForbidden)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if deviceID != "" && !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if traceID == "" && sessionID == "" {
		http.Error(w, "trace_id or session_id is required", http.StatusBadRequest)
		return
	}
	limit := 64
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > maxAudioCaptureFrames {
			http.Error(w, "limit must be between 1 and 512", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	includeAudio := r.URL.Query().Get("include_audio") == "1"
	writeJSON(w, http.StatusOK, AudioRecentResponse{
		SchemaVersion: AudioRecentSchemaVersion,
		DeviceID:      deviceID,
		TraceID:       traceID,
		SessionID:     sessionID,
		IncludeAudio:  includeAudio,
		Frames:        s.recentAudioFrames(deviceID, traceID, sessionID, limit, includeAudio),
	})
}

func loopbackHTTPRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "a21-gateway",
		"status":  "ok",
		"version": buildinfo.Version,
	})
}

func (s *Server) handleVoiceProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	health, err := s.voice.Health(ctx)
	status := http.StatusOK
	if err != nil || health.Status == providers.VoiceProviderUnavailable {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, health)
}

func (s *Server) handleRealtimeSessionStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RealtimeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		http.Error(w, "professional mode must use the professional path with V21 evidence", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.metrics.realtimeSessionTotal.Inc()
	s.recordTrace(traceID, sessionID, req.DeviceID, "realtime.session.start.received", s.now().UnixMilli())

	outputs, err := s.startRealtimeVoiceTurn(r.Context(), req)
	status := "completed"
	code := http.StatusOK
	if err != nil {
		status = "error"
		code = http.StatusBadGateway
		outputs = []realtimeVoiceOutput{{Control: protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true}}}
	}
	events := s.realtimeOutputSequence(req.DeviceID, traceID, sessionID, outputs)
	writeJSON(w, code, RealtimeSessionResponse{
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  req.DeviceID,
		Provider:  s.voice.Name(),
		Status:    status,
		Events:    events,
	})
}

func (s *Server) startRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.start", s.now().UnixMilli())
	providerEvents, err := s.voice.StartTurn(ctx, providers.VoiceTurnRequest{
		Session: providers.VoiceSession{TraceID: req.TraceID, SessionID: req.SessionID, DeviceID: req.DeviceID},
		Text:    req.Text,
		Mode:    string(req.Mode),
	})
	s.metrics.voiceProviderStartTurnMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.error", s.now().UnixMilli())
		return nil, err
	}
	outputs := make([]realtimeVoiceOutput, 0, 4)
	firstEvent := true
	for event := range providerEvents {
		if firstEvent {
			s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.first_event", s.now().UnixMilli())
			firstEvent = false
		}
		payload := voiceEventToControlPayload(event, req.Mode)
		if payload.State == protocol.ExpressionSpeaking && payload.StreamID != "" {
			s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, payload.StreamID)
		}
		outputs = append(outputs, realtimeVoiceOutput{Control: payload, Audio: event.Audio})
	}
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.end", s.now().UnixMilli())
	return outputs, nil
}

func (s *Server) handleRealtimeSessionCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RealtimeSessionCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Reason == "" {
		req.Reason = providers.CancelBargeIn
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	if req.StreamID == "" {
		req.StreamID = s.clearActiveStream(traceID, sessionID, req.DeviceID)
	} else {
		s.clearActiveStream(traceID, sessionID, req.DeviceID)
	}
	s.metrics.realtimeSessionCancelTotal.Inc()
	s.recordTrace(traceID, sessionID, req.DeviceID, "realtime.session.cancel.received", s.now().UnixMilli())

	outputs, err := s.cancelRealtimeVoiceTurn(r.Context(), req)
	status := "cancelled"
	code := http.StatusOK
	if err != nil {
		status = "error"
		code = http.StatusBadGateway
		outputs = []realtimeVoiceOutput{{Control: protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: req.Mode, Text: "好，我听新的。", Final: true, StreamID: req.StreamID}}}
	}
	events := s.realtimeOutputSequence(req.DeviceID, traceID, sessionID, outputs)
	writeJSON(w, code, RealtimeSessionResponse{
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  req.DeviceID,
		Provider:  s.voice.Name(),
		Status:    status,
		Events:    events,
	})
}

func (s *Server) cancelRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionCancelRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.start", s.now().UnixMilli())
	providerEvents, err := s.voice.Cancel(ctx, providers.VoiceCancelRequest{
		Session:  providers.VoiceSession{TraceID: req.TraceID, SessionID: req.SessionID, DeviceID: req.DeviceID},
		Reason:   req.Reason,
		StreamID: req.StreamID,
	})
	s.metrics.voiceProviderCancelMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.error", s.now().UnixMilli())
		return nil, err
	}
	outputs := make([]realtimeVoiceOutput, 0, 2)
	for event := range providerEvents {
		outputs = append(outputs, realtimeVoiceOutput{Control: voiceEventToControlPayload(event, req.Mode), Audio: event.Audio})
	}
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.end", s.now().UnixMilli())
	return outputs, nil
}

func (s *Server) handleMockTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_turn.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockTurnResponse(req))
}

func (s *Server) handleMockInterrupt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_interrupt.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockInterruptResponse(req))
}

func (s *Server) handleControlWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("control").Inc()
	defer s.metrics.wsConnections.WithLabelValues("control").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 control closed")

	ctx := context.Background()
	for {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return
		}
		events := s.controlEventsForDeviceEvent(event)
		for _, control := range events {
			if err := wsjson.Write(ctx, conn, control); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleAudioWS(w http.ResponseWriter, r *http.Request) {
	queryDeviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if queryDeviceID != "" && !validA21DeviceID(queryDeviceID) {
		http.Error(w, "invalid device_id", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("audio").Inc()
	defer s.metrics.wsConnections.WithLabelValues("audio").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 audio closed")
	realtimeAudioKeys := make(map[string]struct{})
	defer s.closeRealtimeAudioSessions(context.Background(), realtimeAudioKeys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writeMu := &sync.Mutex{}
	registeredDeviceID := queryDeviceID
	if registeredDeviceID != "" {
		s.registerAudioSocket(registeredDeviceID, conn, writeMu)
	}
	defer func() {
		if registeredDeviceID != "" {
			s.unregisterAudioSocket(registeredDeviceID, conn)
		}
	}()
	for {
		var frame protocol.Envelope
		if err := wsjson.Read(ctx, conn, &frame); err != nil {
			return
		}
		if registeredDeviceID == "" && validA21DeviceID(frame.DeviceID) {
			registeredDeviceID = frame.DeviceID
			s.registerAudioSocket(registeredDeviceID, conn, writeMu)
		}
		if frame.Kind == protocol.KindAudioFrame {
			s.metrics.audioFrameTotal.Inc()
		}
		traceID, sessionID := s.ids(frame.TraceID, frame.SessionID)
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.frame.received", s.now().UnixMilli())
		ingress, validAudio := s.observeAudioIngress(frame, traceID, sessionID)
		if frame.Kind == protocol.KindAudioFrame && !validAudio {
			events := s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: "invalid audio frame", Final: true},
			})
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if s.shouldBargeIn(frame, traceID, sessionID, ingress) {
			events := s.audioBargeInEvents(frame, traceID, sessionID)
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if frame.Kind == protocol.KindAudioFrame && s.audioProbeOnly(frame.DeviceID, traceID, sessionID) {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.probe.frame.accepted", s.now().UnixMilli())
			continue
		}
		if events, handled := s.realtimeAudioEvents(ctx, conn, writeMu, frame, traceID, sessionID, ingress, realtimeAudioKeys); handled {
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
		}
		if !s.shouldEmitMockAudioPlayback(frame, traceID, sessionID, ingress) {
			continue
		}
		streamID := s.mockAudioStreamID(frame, traceID, sessionID)
		events := s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "audio frame accepted"},
			{State: protocol.ExpressionSpeaking, Mode: protocol.ModeWorkmate, Text: "mock playback chunk", Final: true, StreamID: streamID},
		})
		for _, event := range events {
			if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
				return
			}
		}
		playback := s.mockAudioPlaybackChunk(frame, traceID, sessionID, streamID)
		if err := writeAudioEnvelope(ctx, conn, writeMu, playback); err != nil {
			return
		}
		s.setActiveStream(traceID, sessionID, frame.DeviceID, streamID)
	}
}

func (s *Server) shouldEmitMockAudioPlayback(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) bool {
	if frame.Kind != protocol.KindAudioFrame {
		return true
	}
	if !physicalStackChanDeviceID(frame.DeviceID) {
		return true
	}
	if s.consumeMockPlaybackOnNextAudioFrame(frame.DeviceID, traceID, sessionID) {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.armed", s.now().UnixMilli())
		return true
	}
	if containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechStart) {
		return true
	}
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.suppressed", s.now().UnixMilli())
	return false
}

func (s *Server) observeAudioIngress(frame protocol.Envelope, traceID string, sessionID string) (audio.IngressResult, bool) {
	if frame.Kind != protocol.KindAudioFrame {
		return audio.IngressResult{}, true
	}
	var chunk protocol.AudioChunk
	if err := json.Unmarshal(frame.Payload, &chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.invalid", s.now().UnixMilli())
		return audio.IngressResult{}, false
	}
	if err := protocol.ValidateAudioChunk(chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.invalid", s.now().UnixMilli())
		return audio.IngressResult{}, false
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     frame.DeviceID,
		TraceID:      traceID,
		SessionID:    sessionID,
		Seq:          frame.Seq,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	s.recordAudioCaptureFrame(frame, chunk, result, traceID, sessionID)
	s.metrics.audioIngressFramesTotal.Inc()
	if result.DroppedFrameDelta > 0 {
		s.metrics.audioIngressDroppedTotal.Add(float64(result.DroppedFrameDelta))
	}
	s.metrics.audioIngressBufferDepth.Set(float64(result.BufferedFrames))
	s.metrics.audioIngressRMS.Set(result.RMS)
	s.metrics.vadDetectorDecisions.WithLabelValues(vadDetectorLabel(result.VADDetector), vadDecisionLabel(result.SpeechDetected)).Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.ingress.buffered", s.now().UnixMilli())
	for _, event := range result.Events {
		switch event {
		case audio.EventVADSpeechStart:
			s.metrics.vadSpeechStartTotal.Inc()
		case audio.EventVADSpeechEnd:
			s.metrics.vadSpeechEndTotal.Inc()
		}
		s.recordTrace(traceID, sessionID, frame.DeviceID, string(event), s.now().UnixMilli())
	}
	return result, true
}

func (s *Server) recordAudioCaptureFrame(frame protocol.Envelope, chunk protocol.AudioChunk, ingress audio.IngressResult, traceID string, sessionID string) {
	dataBytes := 0
	if data, err := base64.StdEncoding.DecodeString(chunk.DataBase64); err == nil {
		dataBytes = len(data)
	}
	capture := AudioCaptureFrame{
		DeviceID:            frame.DeviceID,
		TraceID:             traceID,
		SessionID:           sessionID,
		Seq:                 frame.Seq,
		SentAtMS:            frame.SentAtMS,
		ReceivedAtMS:        s.now().UnixMilli(),
		SampleRateHz:        chunk.SampleRateHz,
		Channels:            chunk.Channels,
		DurationMS:          chunk.DurationMS,
		CaptureStartedAtMS:  chunk.CaptureStartedAtMS,
		CaptureEndedAtMS:    chunk.CaptureEndedAtMS,
		DataBytes:           dataBytes,
		DataBase64:          chunk.DataBase64,
		RMS:                 ingress.RMS,
		VADDetector:         ingress.VADDetector,
		SpeechDetected:      ingress.SpeechDetected,
		SpeechActive:        ingress.SpeechActive,
		DroppedFrames:       ingress.DroppedFrames,
		DroppedFrameDelta:   ingress.DroppedFrameDelta,
		IngressBufferFrames: ingress.BufferedFrames,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audioCaptureFrames = append(s.audioCaptureFrames, capture)
	if len(s.audioCaptureFrames) > maxAudioCaptureFrames {
		s.audioCaptureFrames = s.audioCaptureFrames[len(s.audioCaptureFrames)-maxAudioCaptureFrames:]
	}
}

func (s *Server) recentAudioFrames(deviceID string, traceID string, sessionID string, limit int, includeAudio bool) []AudioCaptureFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	matches := make([]AudioCaptureFrame, 0, limit)
	for i := len(s.audioCaptureFrames) - 1; i >= 0 && len(matches) < limit; i-- {
		frame := s.audioCaptureFrames[i]
		if deviceID != "" && frame.DeviceID != deviceID {
			continue
		}
		if traceID != "" && frame.TraceID != traceID {
			continue
		}
		if sessionID != "" && frame.SessionID != sessionID {
			continue
		}
		if !includeAudio {
			frame.DataBase64 = ""
		}
		matches = append(matches, frame)
	}
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}
	return matches
}

func vadDetectorLabel(detector string) string {
	if detector == "" {
		return "unknown"
	}
	return detector
}

func vadDecisionLabel(speechDetected bool) string {
	if speechDetected {
		return "speech"
	}
	return "silence"
}

func (s *Server) realtimeAudioEvents(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult, connectionSessionKeys map[string]struct{}) ([]protocol.Envelope, bool) {
	provider, ok := s.voice.(providers.RealtimeVoiceProvider)
	if !ok || frame.Kind != protocol.KindAudioFrame {
		return nil, false
	}
	key := streamStateKey(traceID, sessionID, frame.DeviceID)
	if s.shouldSuppressUnarmedPhysicalRealtimeAudio(frame, traceID, sessionID, ingress) {
		return nil, true
	}
	if containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechEnd) {
		session := s.realtimeAudioSession(traceID, sessionID, frame.DeviceID)
		if session == nil {
			return nil, true
		}
		connectionSessionKeys[key] = struct{}{}
		if err := session.CommitAndCreateResponse(ctx); err != nil {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.commit.error", s.now().UnixMilli())
			return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
			}), true
		}
		s.metrics.realtimeAudioCommitTotal.Inc()
		s.markRealtimeAudioCommit(traceID, sessionID, frame.DeviceID, s.now())
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.commit", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionThinking, Mode: protocol.ModeWorkmate, Text: "我在想"},
		}), true
	}
	if !ingress.SpeechActive || !ingress.SpeechDetected {
		return nil, true
	}

	var chunk protocol.AudioChunk
	if err := json.Unmarshal(frame.Payload, &chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append.error", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: "invalid audio frame", Final: true},
		}), true
	}
	session := s.realtimeAudioSession(traceID, sessionID, frame.DeviceID)
	started := false
	if session == nil {
		created, err := provider.StartRealtimeSession(ctx, providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: frame.DeviceID})
		if err != nil {
			s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_session.error", s.now().UnixMilli())
			return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
				{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
			}), true
		}
		session = created
		s.setRealtimeAudioSession(traceID, sessionID, frame.DeviceID, session)
		connectionSessionKeys[key] = struct{}{}
		s.startRealtimeAudioDownlinkPump(ctx, conn, writeMu, frame.DeviceID, traceID, sessionID, session)
		started = true
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_session.start", s.now().UnixMilli())
	} else {
		connectionSessionKeys[key] = struct{}{}
	}
	if err := session.SendAudio(ctx, chunk); err != nil {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append.error", s.now().UnixMilli())
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true},
		}), true
	}
	s.metrics.realtimeAudioUplinkFrames.Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.audio.append", s.now().UnixMilli())
	if started {
		return s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "我在听"},
		}), true
	}
	return nil, true
}

func (s *Server) shouldSuppressUnarmedPhysicalRealtimeAudio(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) bool {
	if !physicalStackChanDeviceID(frame.DeviceID) {
		return false
	}
	if s.realtimeAudioSession(traceID, sessionID, frame.DeviceID) != nil {
		return false
	}
	if !ingress.SpeechDetected {
		return false
	}
	if s.consumeRealtimeOnNextSpeech(frame.DeviceID, traceID, sessionID) {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_audio.physical_armed", s.now().UnixMilli())
		return false
	}
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.realtime_audio.physical_suppressed", s.now().UnixMilli())
	return true
}

func (s *Server) startRealtimeAudioDownlinkPump(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, deviceID string, traceID string, sessionID string, session providers.RealtimeVoiceSession) {
	events := session.Events()
	if events == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				outputTraceID := firstNonEmpty(event.Session.TraceID, traceID)
				outputSessionID := firstNonEmpty(event.Session.SessionID, sessionID)
				outputDeviceID := firstNonEmpty(event.Session.DeviceID, deviceID)
				payload := voiceEventToControlPayload(event, protocol.ModeWorkmate)
				if payload.State == protocol.ExpressionSpeaking && payload.StreamID != "" {
					s.setActiveStream(outputTraceID, outputSessionID, outputDeviceID, payload.StreamID)
				}
				s.metrics.realtimeAudioDownlinkEvents.Inc()
				eventAt := s.now()
				s.recordTrace(outputTraceID, outputSessionID, outputDeviceID, "provider.audio.downlink", eventAt.UnixMilli())
				if event.Audio != nil {
					s.observeRealtimeFirstAudioDownlink(outputTraceID, outputSessionID, outputDeviceID, eventAt)
				}
				envelopes := s.realtimeOutputSequence(outputDeviceID, outputTraceID, outputSessionID, []realtimeVoiceOutput{
					{Control: payload, Audio: event.Audio},
				})
				for _, envelope := range envelopes {
					if err := writeAudioEnvelope(ctx, conn, writeMu, envelope); err != nil {
						return
					}
				}
			}
		}
	}()
}

func (s *Server) shouldBargeIn(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) bool {
	if frame.Kind != protocol.KindAudioFrame {
		return false
	}
	if !containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechStart) {
		return false
	}
	return s.activeStream(traceID, sessionID, frame.DeviceID) != ""
}

func containsAudioIngressEvent(events []audio.Event, want audio.Event) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}

func (s *Server) audioBargeInEvents(frame protocol.Envelope, traceID string, sessionID string) []protocol.Envelope {
	streamID := s.clearActiveStream(traceID, sessionID, frame.DeviceID)
	s.metrics.bargeInTotal.Inc()
	now := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "barge_in.detected", now)
	s.recordTrace(traceID, sessionID, frame.DeviceID, "playback.stop", now)
	s.recordTrace(traceID, sessionID, frame.DeviceID, "provider.cancel", now)
	payloads := make([]protocol.ControlEventPayload, 0, 2)
	providerEvents, err := s.voice.Cancel(context.Background(), providers.VoiceCancelRequest{
		Session:  providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: frame.DeviceID},
		Reason:   providers.CancelBargeIn,
		StreamID: streamID,
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{
			State:    protocol.ExpressionInterrupted,
			Mode:     protocol.ModeWorkmate,
			Text:     "好，我听新的。",
			StreamID: streamID,
		})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, protocol.ModeWorkmate))
		}
	}
	payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "你说。"})
	return s.controlSequence(frame.DeviceID, traceID, sessionID, payloads)
}

func a21WebSocketAcceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		Subprotocols:   []string{"arduino"},
		OriginPatterns: []string{"file://"},
	}
}

func (s *Server) controlEventsForDeviceEvent(event protocol.Envelope) []protocol.Envelope {
	var payload protocol.DeviceEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return s.errorEvents(event, "invalid device event")
	}
	traceID, sessionID := s.ids(event.TraceID, event.SessionID)
	event.TraceID = traceID
	event.SessionID = sessionID
	s.recordTrace(traceID, sessionID, event.DeviceID, "device."+string(payload.Event)+".received", s.now().UnixMilli())
	record := s.recordDeviceEvent(event, payload)
	if record.IdentityStatus == "invalid" {
		s.metrics.deviceIdentityInvalidTotal.Inc()
		return s.errorEvents(event, "invalid device firmware identity: "+record.IdentityError)
	}
	req := MockTurnRequest{
		DeviceID:  event.DeviceID,
		Text:      payload.Text,
		Mode:      payload.Mode,
		TraceID:   event.TraceID,
		SessionID: event.SessionID,
	}
	switch payload.Event {
	case protocol.DeviceEventMockTurn, protocol.DeviceEventTouchWakeOrListen:
		return s.mockTurnResponse(req).Events
	case protocol.DeviceEventInterrupt, protocol.DeviceEventTouchBargeIn:
		return s.mockInterruptResponse(req).Events
	case protocol.DeviceEventTouchTopTap:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "我在。", Final: true},
		})
	case protocol.DeviceEventTouchTopSwipeForward:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionThinking, Mode: protocol.ModeCoCreation, Text: "往前推一步。", Final: true},
		})
	case protocol.DeviceEventTouchTopSwipeBackward:
		return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionIdle, Mode: protocol.ModeFocus, Text: "我先收一收。", Final: true},
		})
	case protocol.DeviceEventRuntimeEcho:
		return nil
	default:
		return s.errorEvents(event, "unsupported device event")
	}
}

func (s *Server) recordTrace(traceID string, sessionID string, deviceID string, name string, atMS int64) {
	if traceID == "" || name == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.traces[traceID]
	offset := int64(0)
	if len(events) > 0 {
		offset = atMS - events[0].AtMS
		if offset < 0 {
			offset = 0
		}
	}
	event := TraceEvent{
		Name:      name,
		TraceID:   traceID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		AtMS:      atMS,
		OffsetMS:  offset,
	}
	s.traces[traceID] = append(events, event)
}

func (s *Server) traceEvents(traceID string) []TraceEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := append([]TraceEvent(nil), s.traces[traceID]...)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].OffsetMS < events[j].OffsetMS
	})
	return events
}

func traceLatencySummary(events []TraceEvent) TraceLatencySummary {
	summary := TraceLatencySummary{EventCount: len(events)}
	if len(events) == 0 {
		return summary
	}
	for _, event := range events {
		if event.OffsetMS > summary.LastOffsetMS {
			summary.LastOffsetMS = event.OffsetMS
		}
	}
	summary.AudioFrameToPlaybackMS = traceDeltaMS(events, "audio.frame.received", "audio.playback.chunk.sent")
	summary.V21QueryFirstResultMS = traceDeltaMS(events, "v21.query.start", "v21.query.first_result")
	summary.BargeInStopMS = traceDeltaMS(events, "barge_in.detected", "playback.stop")
	summary.ProviderCommitToFirstAudioMS = traceDeltaMS(events, "provider.audio.commit", "provider.audio.first_downlink")
	return summary
}

func traceDeltaMS(events []TraceEvent, startName string, endName string) *int64 {
	var startAtMS int64
	hasStart := false
	for _, event := range events {
		switch {
		case event.Name == startName && !hasStart:
			startAtMS = event.AtMS
			hasStart = true
		case event.Name == endName && hasStart:
			delta := event.AtMS - startAtMS
			if delta < 0 {
				delta = 0
			}
			return &delta
		}
	}
	return nil
}

func (s *Server) recordDeviceEvent(event protocol.Envelope, payload protocol.DeviceEventPayload) DeviceRecord {
	nowMS := s.now().UnixMilli()
	firmware := DeviceFirmwareIdentity{
		ID:      payload.FirmwareID,
		Version: payload.FirmwareVersion,
		Board:   payload.FirmwareBoard,
		Commit:  payload.FirmwareCommit,
	}
	status, identityError := validateFirmwareIdentity(firmware)
	capabilities, capabilityError := sanitizeDeviceStringMap(payload.Capabilities, "device capabilities")
	if capabilityError != "" {
		status = "invalid"
		if identityError != "" {
			identityError += "; " + capabilityError
		} else {
			identityError = capabilityError
		}
	}
	runtimeEcho, runtimeEchoError := sanitizeDeviceStringMap(payload.RuntimeEcho, "device runtime echo values")
	if runtimeEchoError != "" {
		status = "invalid"
		if identityError != "" {
			identityError += "; " + runtimeEchoError
		} else {
			identityError = runtimeEchoError
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[event.DeviceID]
	if record.DeviceID == "" {
		record.DeviceID = event.DeviceID
		record.FirstSeenMS = nowMS
	}
	record.Firmware = firmware
	record.Capabilities = capabilities
	if runtimeEcho != nil {
		record.RuntimeEcho = runtimeEcho
	}
	record.IdentityStatus = status
	record.IdentityError = identityError
	record.LastEvent = payload.Event
	if payload.TouchSource != "" {
		record.LastTouchSource = payload.TouchSource
	}
	record.LastSeq = event.Seq
	record.LastTraceID = event.TraceID
	record.LastSessionID = event.SessionID
	record.LastSeenMS = nowMS
	s.devices[event.DeviceID] = record
	return record
}

func sanitizeDeviceStringMap(input map[string]string, label string) (map[string]string, string) {
	if len(input) == 0 {
		return nil, ""
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		cleanKey := strings.TrimSpace(key)
		cleanValue := strings.TrimSpace(value)
		if cleanKey == "" || cleanValue == "" {
			continue
		}
		if strings.Contains(strings.ToLower(cleanKey), "x21") ||
			strings.Contains(strings.ToLower(cleanKey), "v21") ||
			strings.Contains(strings.ToLower(cleanValue), "x21") ||
			strings.Contains(strings.ToLower(cleanValue), "v21") {
			return nil, label + " contain forbidden legacy identity"
		}
		output[cleanKey] = cleanValue
	}
	return output, ""
}

func (s *Server) recordDeviceControl(deviceID string, traceID string, sessionID string, payload protocol.ControlEventPayload, atMS int64) {
	if deviceID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = atMS
		record.IdentityStatus = "unknown"
	}
	if payload.Mode != "" {
		record.CurrentMode = payload.Mode
	}
	if payload.State != "" {
		record.CurrentExpr = payload.State
	}
	if payload.State != "" {
		streamKey := streamStateKey(traceID, sessionID, deviceID)
		if payload.State == protocol.ExpressionSpeaking {
			if payload.StreamID != "" {
				record.PlaybackStream = payload.StreamID
				s.activeStreams[streamKey] = payload.StreamID
			}
		} else {
			record.PlaybackStream = ""
			delete(s.activeStreams, streamKey)
		}
	}
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = atMS
	s.devices[deviceID] = record
}

func (s *Server) registerAudioSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audioSockets[deviceID] = &deviceSocket{conn: conn, writeMu: writeMu, connected: s.now().UnixMilli()}
}

func (s *Server) unregisterAudioSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.audioSockets[deviceID]
	if socket != nil && socket.conn == conn {
		delete(s.audioSockets, deviceID)
	}
}

func (s *Server) audioSocket(deviceID string) (*deviceSocket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.audioSockets[deviceID]
	return socket, socket != nil
}

func (s *Server) deviceRecords() []DeviceRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	nowMS := s.now().UnixMilli()
	records := make([]DeviceRecord, 0, len(s.devices))
	for _, record := range s.devices {
		records = append(records, withDeviceFreshness(record, nowMS))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].DeviceID < records[j].DeviceID
	})
	return records
}

const deviceOnlineWindowMS int64 = 300_000

func withDeviceFreshness(record DeviceRecord, nowMS int64) DeviceRecord {
	if record.LastSeenMS <= 0 {
		record.ConnectionStatus = "unknown"
		return record
	}
	ageMS := nowMS - record.LastSeenMS
	if ageMS < 0 {
		ageMS = 0
	}
	record.DeviceAgeMS = ageMS
	if ageMS <= deviceOnlineWindowMS {
		record.ConnectionStatus = "online"
	} else {
		record.ConnectionStatus = "stale"
	}
	return record
}

var a21FirmwareCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)
var a21FirmwareVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$`)

func validateFirmwareIdentity(identity DeviceFirmwareIdentity) (string, string) {
	if identity.ID == "" && identity.Version == "" && identity.Board == "" && identity.Commit == "" {
		return "unknown", ""
	}
	values := []string{identity.ID, identity.Version, identity.Board, identity.Commit}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), "x21") || strings.Contains(strings.ToLower(value), "v21") {
			return "invalid", "firmware identity contains forbidden legacy identity"
		}
	}
	if identity.ID != "a21-stackchan" {
		return "invalid", "firmware_id must be a21-stackchan"
	}
	if !a21FirmwareVersionPattern.MatchString(identity.Version) {
		return "invalid", "firmware_version must be semver-like"
	}
	if identity.Board != "m5stack-cores3" {
		return "invalid", "firmware_board must be m5stack-cores3"
	}
	if !a21FirmwareCommitPattern.MatchString(identity.Commit) {
		return "invalid", "firmware_commit must be a git sha"
	}
	return "ok", ""
}

func validA21DeviceID(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	lower := strings.ToLower(deviceID)
	return !strings.Contains(lower, "x21") && !strings.Contains(lower, "v21")
}

func (s *Server) deviceControlEvents(req DeviceControlRequest) []protocol.Envelope {
	payload := protocol.ControlEventPayload{
		State:                    req.State,
		Mode:                     req.Mode,
		Text:                     req.Text,
		Final:                    req.State != protocol.ExpressionListening && req.State != protocol.ExpressionThinking,
		StreamID:                 req.StreamID,
		DiagnosticToneHz:         req.DiagnosticToneHz,
		DiagnosticToneDurationMS: req.DiagnosticToneDurationMS,
		DiagnosticToneVolume:     req.DiagnosticToneVolume,
	}
	events := s.controlSequence(req.DeviceID, req.TraceID, req.SessionID, []protocol.ControlEventPayload{payload})
	if len(req.AudioChunks) > 0 {
		s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, req.StreamID)
		sentAt := s.now().UnixMilli()
		for i, chunk := range req.AudioChunks {
			events = append(events, s.voiceAudioPlaybackChunk(
				req.DeviceID,
				req.TraceID,
				req.SessionID,
				uint64(len(events)+1),
				sentAt+int64(i+1),
				chunk.StreamID,
				&providers.VoiceAudioChunk{
					Codec:        string(chunk.Codec),
					SampleRateHz: chunk.SampleRateHz,
					Channels:     chunk.Channels,
					DurationMS:   chunk.DurationMS,
					DataBase64:   chunk.DataBase64,
				},
			))
		}
		return events
	}
	if req.MockAudioChunks <= 0 || req.StreamID == "" {
		return events
	}
	s.setActiveStream(req.TraceID, req.SessionID, req.DeviceID, req.StreamID)
	sentAt := s.now().UnixMilli()
	for i := 0; i < req.MockAudioChunks; i++ {
		events = append(events, s.voiceAudioPlaybackChunk(
			req.DeviceID,
			req.TraceID,
			req.SessionID,
			uint64(len(events)+1),
			sentAt+int64(i+1),
			req.StreamID,
			&providers.VoiceAudioChunk{
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: 16000,
				Channels:     1,
				DurationMS:   20,
				DataBase64:   mockPCM16SquareWaveBase64(16000, 20),
			},
		))
	}
	return events
}

func (s *Server) handleFastCompanionTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FastCompanionTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		http.Error(w, "professional mode must use the professional path with V21 evidence", http.StatusBadRequest)
		return
	}
	if req.Mode != protocol.ModeWorkmate && req.Mode != protocol.ModeCompanion {
		http.Error(w, "fast companion mode must be companion or workmate", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.LocalAudio.ASRProvider) == "" {
		http.Error(w, "local_audio.asr_provider is required", http.StatusBadRequest)
		return
	}
	if req.LocalAudio.FirstPartialMS < 0 || req.LocalAudio.FinalTranscriptChars < 0 {
		http.Error(w, "local_audio timing and transcript counts must be non-negative", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.turn.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.fastCompanionTurnResponse(req))
}

func (s *Server) fastCompanionTurnResponse(req FastCompanionTurnRequest) FastCompanionTurnResponse {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	now := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.local_audio.frontend.accepted", now)
	s.recordTrace(traceID, sessionID, req.DeviceID, "asr.first_partial", now+1)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.text_stream.route.placeholder", now+2)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_byte", now+3)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_content", now+4)
	s.recordTrace(traceID, sessionID, req.DeviceID, "tts.first_audio", now+5)
	s.recordTrace(traceID, sessionID, req.DeviceID, "audio.downlink.first_frame", now+6)
	s.recordTrace(traceID, sessionID, req.DeviceID, "device.playback.start", now+7)
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
		{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "我把本地语音结果接到文本流边界。"},
		{State: protocol.ExpressionSpeaking, Mode: req.Mode, Text: "先走文本流占位边界，不启动真实 provider。", Final: true, StreamID: "a21-fast-companion-placeholder-stream"},
	})
	return FastCompanionTurnResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Mode:               req.Mode,
		Status:             "boundary_ready",
		Route:              "fast_companion_hybrid",
		AudioFrontend:      "local_audio",
		TextStreamProvider: "mock_text_stream",
		ProviderFamily:     string(providers.ProviderFamilyTextStream),
		TextStreamExecuted: false,
		Events:             events,
	}
}

func (s *Server) mockTurnResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.mockTurnTotal.Inc()
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		return s.professionalTurnResponse(req)
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
	}
	providerEvents, err := s.voice.StartTurn(context.Background(), providers.VoiceTurnRequest{
		Session: providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID},
		Text:    req.Text,
		Mode:    string(req.Mode),
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: err.Error(), Final: true})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, req.Mode))
		}
	}
	events := s.controlSequence(req.DeviceID, traceID, sessionID, payloads)
	return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
}

func (s *Server) professionalTurnResponse(req MockTurnRequest) MockTurnResponse {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: protocol.ModeProfessional, Text: "我在听"},
		{State: protocol.ExpressionProfessional, Mode: protocol.ModeProfessional, Text: "进入专业模式。情绪先放旁边，现在只看证据。"},
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "v21.query.start", s.now().UnixMilli())
	queryCtx, cancel := context.WithTimeout(context.Background(), s.v21TTL)
	defer cancel()
	started := time.Now()
	response, err := s.v21.Query(queryCtx, v21adapter.QueryRequest{
		TraceID:   traceID,
		SessionID: sessionID,
		Utterance: req.Text,
	})
	s.metrics.v21QueryMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		marker := "v21.query.error"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(queryCtx.Err(), context.DeadlineExceeded) {
			marker = "v21.query.timeout"
		}
		s.recordTrace(traceID, sessionID, req.DeviceID, marker, s.now().UnixMilli())
		payloads = append(payloads, protocol.ControlEventPayload{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。",
			Final: true,
		})
	} else {
		s.recordTrace(traceID, sessionID, req.DeviceID, "v21.query.first_result", s.now().UnixMilli())
		payloads = append(payloads, protocol.ControlEventPayload{
			State:        protocol.ExpressionSpeaking,
			Mode:         protocol.ModeProfessional,
			Text:         response.FastAnswer,
			Final:        true,
			Confidence:   response.Confidence,
			Evidence:     v21EvidenceToProtocol(response.Evidence),
			SpeechBlocks: response.SpeechBlocks,
			ScreenCards:  v21ScreenCardsToProtocol(response.ScreenCards),
			FollowUps:    response.FollowUps,
		})
	}
	events := s.controlSequence(req.DeviceID, traceID, sessionID, payloads)
	return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
}

func (s *Server) mockInterruptResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.bargeInTotal.Inc()
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	payloads := make([]protocol.ControlEventPayload, 0, 2)
	providerEvents, err := s.voice.Cancel(context.Background(), providers.VoiceCancelRequest{
		Session: providers.VoiceSession{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID},
		Reason:  providers.CancelBargeIn,
	})
	if err != nil {
		payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: mode, Text: "好，我听新的。"})
	} else {
		for event := range providerEvents {
			payloads = append(payloads, voiceEventToControlPayload(event, mode))
		}
	}
	payloads = append(payloads, protocol.ControlEventPayload{State: protocol.ExpressionListening, Mode: mode, Text: "你说。"})
	events := s.controlSequence(req.DeviceID, traceID, sessionID, payloads)
	return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
}

func voiceEventToControlPayload(event providers.VoiceEvent, mode protocol.Mode) protocol.ControlEventPayload {
	switch event.Kind {
	case providers.VoiceEventThinking:
		return protocol.ControlEventPayload{State: protocol.ExpressionThinking, Mode: mode, Text: event.Text}
	case providers.VoiceEventSpeaking:
		return protocol.ControlEventPayload{State: protocol.ExpressionSpeaking, Mode: mode, Text: event.Text, Final: event.Final, StreamID: event.StreamID}
	case providers.VoiceEventCancelled:
		return protocol.ControlEventPayload{State: protocol.ExpressionInterrupted, Mode: mode, Text: event.Text, Final: event.Final, StreamID: event.StreamID}
	default:
		return protocol.ControlEventPayload{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: event.Text, Final: true}
	}
}

func v21EvidenceToProtocol(evidence []v21adapter.Evidence) []protocol.EvidenceItem {
	items := make([]protocol.EvidenceItem, 0, len(evidence))
	for _, item := range evidence {
		items = append(items, protocol.EvidenceItem{
			Title:    item.Title,
			Type:     item.Type,
			SourceID: item.SourceID,
			Summary:  item.Summary,
			Quote:    item.Quote,
		})
	}
	return items
}

func v21ScreenCardsToProtocol(cards []v21adapter.ScreenCard) []protocol.ScreenCard {
	result := make([]protocol.ScreenCard, 0, len(cards))
	for _, card := range cards {
		result = append(result, protocol.ScreenCard{Label: card.Label, Text: card.Text})
	}
	return result
}

func (s *Server) errorEvents(event protocol.Envelope, text string) []protocol.Envelope {
	traceID, sessionID := s.ids(event.TraceID, event.SessionID)
	return s.controlSequence(event.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionError, Mode: protocol.ModeError, Text: text, Final: true},
	})
}

func (s *Server) ids(traceID string, sessionID string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if traceID == "" || sessionID == "" {
		s.next++
	}
	if traceID == "" {
		traceID = fmt.Sprintf("a21-trace-%06d", s.next)
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("a21-session-%06d", s.next)
	}
	return traceID, sessionID
}

func (s *Server) controlSequence(deviceID string, traceID string, sessionID string, payloads []protocol.ControlEventPayload) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(payloads))
	sentAt := s.now().UnixMilli()
	for i, payload := range payloads {
		data, _ := json.Marshal(payload)
		events = append(events, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  deviceID,
			Kind:      protocol.KindControlEvent,
			Seq:       uint64(i + 1),
			TraceID:   traceID,
			SessionID: sessionID,
			SentAtMS:  sentAt + int64(i),
			Payload:   data,
		})
		s.recordTrace(traceID, sessionID, deviceID, "control."+string(payload.State)+".sent", sentAt+int64(i))
		s.recordDeviceControl(deviceID, traceID, sessionID, payload, sentAt+int64(i))
	}
	return events
}

func (s *Server) realtimeOutputSequence(deviceID string, traceID string, sessionID string, outputs []realtimeVoiceOutput) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(outputs)*2)
	sentAt := s.now().UnixMilli()
	var seq uint64 = 1
	for _, output := range outputs {
		if output.Control.State != "" {
			data, _ := json.Marshal(output.Control)
			eventAt := sentAt + int64(seq-1)
			events = append(events, protocol.Envelope{
				Protocol:  protocol.ProtocolVersion,
				DeviceID:  deviceID,
				Kind:      protocol.KindControlEvent,
				Seq:       seq,
				TraceID:   traceID,
				SessionID: sessionID,
				SentAtMS:  eventAt,
				Payload:   data,
			})
			s.recordTrace(traceID, sessionID, deviceID, "control."+string(output.Control.State)+".sent", eventAt)
			s.recordDeviceControl(deviceID, traceID, sessionID, output.Control, eventAt)
			seq++
		}
		if output.Audio != nil {
			streamID := output.Control.StreamID
			if streamID == "" {
				streamID = "a21-provider-audio-stream"
			}
			eventAt := sentAt + int64(seq-1)
			events = append(events, s.voiceAudioPlaybackChunk(deviceID, traceID, sessionID, seq, eventAt, streamID, output.Audio))
			seq++
		}
	}
	return events
}

func (s *Server) voiceAudioPlaybackChunk(deviceID string, traceID string, sessionID string, seq uint64, sentAt int64, streamID string, audio *providers.VoiceAudioChunk) protocol.Envelope {
	codec := protocol.AudioCodec(audio.Codec)
	if codec == "" {
		codec = protocol.AudioCodecPCMS16LE
	}
	sampleRateHz := audio.SampleRateHz
	if sampleRateHz == 0 {
		sampleRateHz = 16000
	}
	channels := audio.Channels
	if channels == 0 {
		channels = 1
	}
	durationMS := audio.DurationMS
	if durationMS == 0 {
		durationMS = 20
	}
	payload := protocol.AudioPlaybackChunk{
		StreamID:     streamID,
		Codec:        codec,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   audio.DataBase64,
	}
	data, _ := json.Marshal(payload)
	s.metrics.audioPlaybackChunkTotal.Inc()
	s.recordTrace(traceID, sessionID, deviceID, "audio.playback.chunk.sent", sentAt)
	return protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  deviceID,
		Kind:      protocol.KindAudioPlaybackChunk,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		SentAtMS:  sentAt,
		Payload:   data,
	}
}

func (s *Server) mockAudioPlaybackChunk(frame protocol.Envelope, traceID string, sessionID string, streamID string) protocol.Envelope {
	payload := protocol.AudioPlaybackChunk{
		StreamID:     streamID,
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   mockAudioPlaybackBase64(frame.DeviceID, 16000, 20),
	}
	data, _ := json.Marshal(payload)
	sentAt := s.now().UnixMilli()
	s.metrics.audioPlaybackChunkTotal.Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.playback.chunk.sent", sentAt)
	return protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  frame.DeviceID,
		Kind:      protocol.KindAudioPlaybackChunk,
		Seq:       frame.Seq + 1,
		TraceID:   traceID,
		SessionID: sessionID,
		SentAtMS:  sentAt,
		Payload:   data,
	}
}

func mockAudioPlaybackBase64(deviceID string, sampleRateHz int, durationMS int) string {
	if physicalStackChanDeviceID(deviceID) {
		return mockPCM16SquareWaveBase64(sampleRateHz, durationMS)
	}
	return mockPCM16SilenceBase64(sampleRateHz, durationMS)
}

func physicalStackChanDeviceID(deviceID string) bool {
	return strings.HasPrefix(deviceID, "stackchan-") &&
		!strings.HasPrefix(deviceID, "stackchan-sim-") &&
		!strings.HasPrefix(deviceID, "stackchan-bench-")
}

func mockPCM16SilenceBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	byteCount := sampleRateHz * durationMS * 2 / 1000
	return base64.StdEncoding.EncodeToString(make([]byte, byteCount))
}

func mockPCM16SquareWaveBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	samples := sampleRateHz * durationMS / 1000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(9000)
		if (i/8)%2 == 1 {
			sample = -9000
		}
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Server) setActiveStream(traceID string, sessionID string, deviceID string, streamID string) {
	if streamID == "" {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeStreams[key] = streamID
}

func (s *Server) activeStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeStreams[key]
}

func (s *Server) clearActiveStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	streamID := s.activeStreams[key]
	delete(s.activeStreams, key)
	return streamID
}

func (s *Server) setAudioProbeOnly(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.audioProbeSessions[key] = true
		return
	}
	delete(s.audioProbeSessions, key)
}

func (s *Server) audioProbeOnly(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioProbeSessions[key]
}

func (s *Server) setMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.mockPlaybackArmedSessions[key] = true
		return
	}
	delete(s.mockPlaybackArmedSessions, key)
}

func (s *Server) consumeMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.mockPlaybackArmedSessions[key] {
		return false
	}
	delete(s.mockPlaybackArmedSessions, key)
	return true
}

func (s *Server) setRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.realtimeArmedSessions[key] = true
		return
	}
	delete(s.realtimeArmedSessions, key)
}

func (s *Server) consumeRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.realtimeArmedSessions[key] {
		return false
	}
	delete(s.realtimeArmedSessions, key)
	return true
}

func (s *Server) realtimeAudioSession(traceID string, sessionID string, deviceID string) providers.RealtimeVoiceSession {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realtimeAudio[key]
}

func (s *Server) setRealtimeAudioSession(traceID string, sessionID string, deviceID string, session providers.RealtimeVoiceSession) {
	if session == nil {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudio[key] = session
}

func (s *Server) markRealtimeAudioCommit(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudioCommitAt[key] = at
	delete(s.realtimeAudioFirstDownlink, key)
}

func (s *Server) observeRealtimeFirstAudioDownlink(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	commitAt, hasCommit := s.realtimeAudioCommitAt[key]
	alreadyObserved := s.realtimeAudioFirstDownlink[key]
	if hasCommit && !alreadyObserved {
		s.realtimeAudioFirstDownlink[key] = true
	}
	s.mu.Unlock()
	if !hasCommit || alreadyObserved {
		return
	}
	durationMS := float64(at.Sub(commitAt)) / float64(time.Millisecond)
	if durationMS < 0 {
		durationMS = 0
	}
	s.metrics.realtimeFirstAudioMS.Observe(durationMS)
	s.recordTrace(traceID, sessionID, deviceID, "provider.audio.first_downlink", at.UnixMilli())
}

func (s *Server) closeRealtimeAudioSessions(ctx context.Context, keys map[string]struct{}) {
	for key := range keys {
		session := s.clearRealtimeAudioSessionByKey(key)
		if session != nil {
			_ = session.Close(ctx)
		}
	}
}

func (s *Server) clearRealtimeAudioSessionByKey(key string) providers.RealtimeVoiceSession {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.realtimeAudio[key]
	delete(s.realtimeAudio, key)
	delete(s.realtimeAudioCommitAt, key)
	delete(s.realtimeAudioFirstDownlink, key)
	return session
}

func (s *Server) mockAudioStreamID(frame protocol.Envelope, traceID string, sessionID string) string {
	streamSeq := frame.Seq
	if streamSeq == 0 {
		streamSeq = 1
	}
	key := traceID
	if key == "" {
		key = sessionID
	}
	if key == "" {
		return fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.audioStreams[key]; existing != "" {
		return existing
	}
	streamID := fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	s.audioStreams[key] = streamID
	return streamID
}

func streamStateKey(traceID string, sessionID string, deviceID string) string {
	switch {
	case sessionID != "":
		return sessionID
	case traceID != "":
		return traceID
	case deviceID != "":
		return deviceID
	default:
		return "a21-audio-stream-default"
	}
}

func writeAudioEnvelope(ctx context.Context, conn *websocket.Conn, mu *sync.Mutex, event protocol.Envelope) error {
	mu.Lock()
	defer mu.Unlock()
	return wsjson.Write(ctx, conn, event)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(value)
}
