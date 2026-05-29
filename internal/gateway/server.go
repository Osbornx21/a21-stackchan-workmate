package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Server struct {
	mu      sync.Mutex
	next    uint64
	now     func() time.Time
	metrics *metrics
	voice   providers.VoiceProvider
	devices map[string]DeviceRecord
	traces  map[string][]TraceEvent
}

type ServerOptions struct {
	VoiceProvider providers.VoiceProvider
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

type DeviceFirmwareIdentity struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Board   string `json:"board,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type DeviceRecord struct {
	DeviceID       string                   `json:"device_id"`
	Firmware       DeviceFirmwareIdentity   `json:"firmware,omitempty"`
	IdentityStatus string                   `json:"identity_status"`
	IdentityError  string                   `json:"identity_error,omitempty"`
	LastEvent      protocol.DeviceEventKind `json:"last_event,omitempty"`
	LastSeq        uint64                   `json:"last_seq,omitempty"`
	LastTraceID    string                   `json:"last_trace_id,omitempty"`
	LastSessionID  string                   `json:"last_session_id,omitempty"`
	FirstSeenMS    int64                    `json:"first_seen_ms"`
	LastSeenMS     int64                    `json:"last_seen_ms"`
}

type DeviceRegistryResponse struct {
	Devices []DeviceRecord `json:"devices"`
}

type TraceEvent struct {
	Name      string `json:"name"`
	TraceID   string `json:"trace_id"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	AtMS      int64  `json:"at_ms"`
	OffsetMS  int64  `json:"offset_ms"`
}

type TraceResponse struct {
	TraceID string       `json:"trace_id"`
	Events  []TraceEvent `json:"events"`
}

func NewServer() *Server {
	return NewServerWithOptions(ServerOptions{})
}

func NewServerWithOptions(options ServerOptions) *Server {
	voiceProvider := options.VoiceProvider
	if voiceProvider == nil {
		voiceProvider = providers.NewMockVoiceProvider()
	}
	return &Server{now: time.Now, metrics: newMetrics(), voice: voiceProvider, devices: make(map[string]DeviceRecord), traces: make(map[string][]TraceEvent)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/simulator", s.handleSimulator)
	mux.Handle("/metrics", s.metrics.handler())
	mux.HandleFunc("/v1/devices", s.handleDevices)
	mux.HandleFunc("/v1/traces", s.handleTraces)
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
	writeJSON(w, http.StatusOK, DeviceRegistryResponse{Devices: s.deviceRecords()})
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
	writeJSON(w, http.StatusOK, TraceResponse{TraceID: traceID, Events: s.traceEvents(traceID)})
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
	conn, err := websocket.Accept(w, r, nil)
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
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("audio").Inc()
	defer s.metrics.wsConnections.WithLabelValues("audio").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 audio closed")

	ctx := context.Background()
	for {
		var frame protocol.Envelope
		if err := wsjson.Read(ctx, conn, &frame); err != nil {
			return
		}
		if frame.Kind == protocol.KindAudioFrame {
			s.metrics.audioFrameTotal.Inc()
		}
		traceID, sessionID := s.ids(frame.TraceID, frame.SessionID)
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.frame.received", s.now().UnixMilli())
		events := s.controlSequence(frame.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: protocol.ModeWorkmate, Text: "audio frame accepted"},
		})
		if err := wsjson.Write(ctx, conn, events[0]); err != nil {
			return
		}
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
	case protocol.DeviceEventMockTurn:
		return s.mockTurnResponse(req).Events
	case protocol.DeviceEventInterrupt:
		return s.mockInterruptResponse(req).Events
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

func (s *Server) recordDeviceEvent(event protocol.Envelope, payload protocol.DeviceEventPayload) DeviceRecord {
	nowMS := s.now().UnixMilli()
	firmware := DeviceFirmwareIdentity{
		ID:      payload.FirmwareID,
		Version: payload.FirmwareVersion,
		Board:   payload.FirmwareBoard,
		Commit:  payload.FirmwareCommit,
	}
	status, identityError := validateFirmwareIdentity(firmware)

	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[event.DeviceID]
	if record.DeviceID == "" {
		record.DeviceID = event.DeviceID
		record.FirstSeenMS = nowMS
	}
	record.Firmware = firmware
	record.IdentityStatus = status
	record.IdentityError = identityError
	record.LastEvent = payload.Event
	record.LastSeq = event.Seq
	record.LastTraceID = event.TraceID
	record.LastSessionID = event.SessionID
	record.LastSeenMS = nowMS
	s.devices[event.DeviceID] = record
	return record
}

func (s *Server) deviceRecords() []DeviceRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]DeviceRecord, 0, len(s.devices))
	for _, record := range s.devices {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].DeviceID < records[j].DeviceID
	})
	return records
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

func (s *Server) mockTurnResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.mockTurnTotal.Inc()
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
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
	}
	return events
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(value)
}
