package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func xiaozhiOfficialMCPFallbackPlans(deviceID string, event xiaozhitransport.DeviceExtensionEvent) (string, []XiaozhiMCPControlRequest, error) {
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState:
		preset := map[string]string{
			"idle":      "reset_idle",
			"listening": "listening",
			"thinking":  "thinking",
			"speaking":  "speaking",
			"error":     "reset_idle",
		}[event.Value]
		if preset == "" {
			return "", nil, errors.New("official state has no safe xiaozhi mcp fallback")
		}
		name, plans, err := xiaozhiBodyPresetPlans(deviceID, preset)
		return "body_preset:" + name, plans, err
	case xiaozhitransport.DeviceEventKindFace:
		preset := map[string]string{
			"idle":      "reset_idle",
			"attentive": "listening",
			"thinking":  "thinking",
			"speaking":  "speaking",
			"happy":     "celebrate",
			"error":     "reset_idle",
		}[event.Value]
		if preset == "" {
			return "", nil, errors.New("official face has no safe xiaozhi mcp fallback")
		}
		name, plans, err := xiaozhiBodyPresetPlans(deviceID, preset)
		return "body_preset:" + name, plans, err
	case xiaozhitransport.DeviceEventKindMotion:
		name, plans, err := xiaozhiBodyMotionPlans(deviceID, event.Value)
		return "body_motion:" + name, plans, err
	default:
		return "", nil, errors.New("official event has no safe xiaozhi mcp fallback")
	}
}

func (s *Server) deliverOfficialStackChanMCPFallback(ctx context.Context, deviceID string, traceID string, sessionID string, event xiaozhitransport.DeviceExtensionEvent, metadata stackchantransport.OfficialActionMetadata) (XiaozhiDeviceControlResponse, int, string) {
	fallbackName, plans, err := xiaozhiOfficialMCPFallbackPlans(deviceID, event)
	if err != nil {
		return XiaozhiDeviceControlResponse{}, http.StatusBadRequest, err.Error()
	}
	if len(plans) == 0 {
		return XiaozhiDeviceControlResponse{}, http.StatusBadRequest, "official stackchan fallback has no safe xiaozhi mcp steps"
	}
	valueToken := safeGatewayFallbackToken(event.Value, "value")
	for index, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(ctx, plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, deviceID, "stackchan.official_mcp_fallback.failed", s.now().UnixMilli())
			return XiaozhiDeviceControlResponse{}, status, message
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		fallbackMarker := "stackchan.official_mcp_fallback." + string(event.Kind) + "." + valueToken + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, fallbackMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["official_stackchan_fallback"] = "xiaozhi_mcp_sequence"
		activity["official_stackchan_fallback_name"] = fallbackName
		activity["official_stackchan_fallback_status"] = "delivered"
		activity["official_stackchan_fallback_step"] = strconv.Itoa(index + 1)
		activity["official_stackchan_event"] = string(event.Kind)
		activity["official_stackchan_value"] = event.Value
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, fallbackMarker, activity)
	}

	surfaces := officialStackChanMCPFallbackSurfaces(metadata.Surfaces, fallbackName, len(plans), metadata.PacketCount)
	s.recordOfficialStackChanMCPFallbackDelivered(deviceID, traceID, sessionID, event, metadata, fallbackName, len(plans), surfaces)
	accepted := false
	return XiaozhiDeviceControlResponse{
		TraceID:                        traceID,
		SessionID:                      sessionID,
		DeviceID:                       deviceID,
		Status:                         "fallback_delivered",
		DeliveredTransport:             "xiaozhi_mcp_sequence",
		Event:                          string(event.Kind),
		Value:                          event.Value,
		OfficialActionPhysicalAccepted: &accepted,
		OfficialActionSurfaces:         surfaces,
		OfficialActionFallbackReason:   "official_stackchan_ws_disconnected",
	}, 0, ""
}

func officialStackChanMCPFallbackSurfaces(metadata map[string]string, fallbackName string, stepCount int, plannedPackets int) map[string]string {
	surfaces := map[string]string{
		"official_relay":         "disconnected",
		"fallback":               "xiaozhi_mcp_sequence",
		"fallback_name":          fallbackName,
		"fallback_steps":         strconv.Itoa(stepCount),
		"planned_packet_count":   strconv.Itoa(plannedPackets),
		"physical_accepted":      "false",
		"packet_delivery_status": "not_sent_no_official_ws",
	}
	for key, value := range metadata {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		surfaces["planned_"+key] = value
	}
	return surfaces
}

func xiaozhiBodyScenePlans(deviceID string, scene string) (string, []XiaozhiMCPControlRequest, error) {
	scene = strings.ToLower(strings.TrimSpace(scene))
	if scene == "" {
		return "", nil, errors.New("body_scene is required")
	}
	base := XiaozhiMCPControlRequest{DeviceID: deviceID}
	brightness := func(value int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPScreenSetBrightnessToolName
		req.Brightness = xiaozhiPresetInt(value)
		return req
	}
	theme := func(value string) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPScreenSetThemeToolName
		req.Theme = value
		return req
	}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiPresetInt(red)
		req.Green = xiaozhiPresetInt(green)
		req.Blue = xiaozhiPresetInt(blue)
		return req
	}
	head := func(yaw, pitch, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = xiaozhiPresetInt(yaw)
		req.Pitch = xiaozhiPresetInt(pitch)
		req.Speed = xiaozhiPresetInt(speed)
		return req
	}
	switch scene {
	case "showtime":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(72),
			led(168, 80, 0),
			head(-18, 36, 260),
			led(0, 168, 80),
			head(18, 36, 260),
			head(0, 24, 220),
			led(0, 36, 96),
		}, nil
	case "focus":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(62),
			led(120, 72, 0),
			head(-12, 28, 180),
		}, nil
	case "reset":
		return scene, []XiaozhiMCPControlRequest{
			theme("auto"),
			brightness(55),
			led(0, 0, 32),
			head(0, 18, 200),
		}, nil
	case "full_check":
		return scene, []XiaozhiMCPControlRequest{
			theme("dark"),
			brightness(72),
			led(168, 80, 0),
			head(-18, 36, 260),
			led(0, 168, 80),
			head(18, 36, 260),
			head(0, 24, 220),
			led(0, 36, 96),
			theme("dark"),
			brightness(62),
			led(120, 72, 0),
			head(-12, 28, 180),
			theme("auto"),
			brightness(55),
			led(0, 0, 32),
			head(0, 18, 200),
		}, nil
	default:
		return "", nil, errors.New("body_scene must be showtime, focus, reset, or full_check")
	}
}

func xiaozhiPresetInt(value int) *int {
	return &value
}

func xiaozhiMCPAllowedTools() []string {
	return []string{
		xiaozhiSpeakerVolumeToolName,
		xiaozhiMCPGetDeviceStatusToolName,
		xiaozhiMCPScreenSetBrightnessToolName,
		xiaozhiMCPScreenSetThemeToolName,
		xiaozhiMCPScreenGetInfoToolName,
		xiaozhiMCPRobotGetHeadAnglesToolName,
		xiaozhiMCPRobotSetHeadAnglesToolName,
		xiaozhiMCPRobotSetLEDColorToolName,
	}
}

func xiaozhiMCPBlockedToolClasses() []string {
	return []string{
		"reboot",
		"firmware_upgrade",
		"camera_photo",
		"screen_snapshot",
		"camera_stream_video",
		"nfc",
		"infrared",
		"power_shutdown",
		"power_sleep",
		"app_lifecycle",
	}
}

func xiaozhiMCPHasAnyArgument(req XiaozhiMCPControlRequest) bool {
	return xiaozhiMCPHasAnyArgumentExcept(req)
}

func xiaozhiMCPHasAnyArgumentExcept(req XiaozhiMCPControlRequest, allowed ...string) bool {
	allowedSet := map[string]bool{}
	for _, name := range allowed {
		allowedSet[name] = true
	}
	if req.Brightness != nil && !allowedSet["brightness"] {
		return true
	}
	if req.Volume != nil && !allowedSet["volume"] {
		return true
	}
	if strings.TrimSpace(req.Theme) != "" && !allowedSet["theme"] {
		return true
	}
	if req.Yaw != nil && !allowedSet["yaw"] {
		return true
	}
	if req.Pitch != nil && !allowedSet["pitch"] {
		return true
	}
	if req.Speed != nil && !allowedSet["speed"] {
		return true
	}
	if req.Red != nil && !allowedSet["red"] {
		return true
	}
	if req.Green != nil && !allowedSet["green"] {
		return true
	}
	if req.Blue != nil && !allowedSet["blue"] {
		return true
	}
	return false
}

func validXiaozhiScreenTheme(theme string) bool {
	if theme == "" || len(theme) > 32 {
		return false
	}
	for _, ch := range theme {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func validXiaozhiNamedScreenTheme(theme string) bool {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case "light", "dark", "auto":
		return true
	default:
		return false
	}
}

func (s *Server) suppressXiaozhiInputAfterHostSay(session *xiaozhiSession, task xiaozhiTurnTask) {
	if session == nil || !hardwareMACDeviceID(session.deviceID) || xiaozhiClientProfile(session.features) == "debug" {
		return
	}
	untilMS := s.now().UnixMilli() + xiaozhiHostSayInputCooldownMS
	session.suppressXiaozhiInputUntil(untilMS, "after_host_say")
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.say.input_suppression_armed", s.now().UnixMilli())
}

func (s *Server) suppressXiaozhiInputAfterNoSpeechPlaceholder(session *xiaozhiSession, task xiaozhiTurnTask) {
	if session == nil || !hardwareMACDeviceID(session.deviceID) || xiaozhiClientProfile(session.features) == "debug" {
		return
	}
	untilMS := s.now().UnixMilli() + xiaozhiNoSpeechInputCooldownMS
	session.suppressXiaozhiInputUntil(untilMS, "after_no_speech")
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.no_speech.input_suppression_armed", s.now().UnixMilli())
}

func (s *Server) handleOfficialStackChanControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiDeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	event, err := xiaozhiDeviceControlEventFromRequest(req)
	if err != nil {
		http.Error(w, "invalid official stackchan event", http.StatusBadRequest)
		return
	}
	plan, err := stackchantransport.BuildOfficialActionPlan(event)
	if err != nil {
		http.Error(w, "unsupported official stackchan event", http.StatusBadRequest)
		return
	}
	event = plan.Event
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	socket, ok := s.officialStackChanSocket(req.DeviceID)
	if !ok {
		if req.AllowMCPFallback {
			response, status, message := s.deliverOfficialStackChanMCPFallback(r.Context(), req.DeviceID, traceID, sessionID, event, plan.Metadata)
			if status != 0 {
				http.Error(w, message, status)
				return
			}
			writeJSON(w, http.StatusOK, response)
			return
		}
		http.Error(w, "official stackchan websocket is not connected", http.StatusConflict)
		return
	}
	socket.writeMu.Lock()
	for _, packet := range plan.Packets {
		err = socket.conn.Write(r.Context(), websocket.MessageBinary, packet.Bytes())
		if err != nil {
			break
		}
	}
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "official stackchan command delivery failed", http.StatusBadGateway)
		return
	}
	s.recordOfficialStackChanControlDelivered(req.DeviceID, traceID, sessionID, event, plan.Metadata)
	writeJSON(w, http.StatusOK, XiaozhiDeviceControlResponse{
		TraceID:                        traceID,
		SessionID:                      sessionID,
		DeviceID:                       req.DeviceID,
		Status:                         "delivered",
		DeliveredTransport:             "stackchan_official_ws",
		Event:                          string(event.Kind),
		Value:                          event.Value,
		PacketCount:                    plan.Metadata.PacketCount,
		OfficialActionPhysicalAccepted: boolValue(plan.Metadata.PhysicalAccepted),
		OfficialActionSurfaces:         plan.Metadata.Surfaces,
	})
}

func (s *Server) handleOfficialStackChanStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		deviceID = defaultOfficialStackChanDeviceID
	}
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.officialStackChanStatus(deviceID))
}

func xiaozhiDeviceControlEventFromRequest(req XiaozhiDeviceControlRequest) (xiaozhitransport.DeviceExtensionEvent, error) {
	kind := strings.TrimSpace(strings.ToLower(firstNonEmpty(req.Event, req.Kind)))
	if kind == "" {
		switch {
		case strings.TrimSpace(req.State) != "":
			kind = string(xiaozhitransport.DeviceEventKindState)
		case strings.TrimSpace(firstNonEmpty(req.Emotion, req.Face)) != "":
			kind = string(xiaozhitransport.DeviceEventKindFace)
		case strings.TrimSpace(firstNonEmpty(req.Slot, req.Display)) != "":
			kind = string(xiaozhitransport.DeviceEventKindDisplay)
		case strings.TrimSpace(firstNonEmpty(req.Name, req.Motion)) != "":
			kind = string(xiaozhitransport.DeviceEventKindMotion)
		}
	}
	value := ""
	switch xiaozhitransport.DeviceEventKind(kind) {
	case xiaozhitransport.DeviceEventKindState:
		value = req.State
	case xiaozhitransport.DeviceEventKindFace:
		value = firstNonEmpty(req.Emotion, req.Face)
	case xiaozhitransport.DeviceEventKindDisplay:
		value = firstNonEmpty(req.Slot, req.Display)
	case xiaozhitransport.DeviceEventKindMotion:
		value = firstNonEmpty(req.Name, req.Motion)
	}
	return xiaozhitransport.NormalizeDeviceExtensionEvent(xiaozhitransport.DeviceExtensionEvent{
		Kind:   xiaozhitransport.DeviceEventKind(kind),
		Value:  value,
		YAngle: req.YAngle,
		Text:   req.Text,
		Reason: req.Reason,
	})
}

func (s *Server) applyDeviceControlStreamState(req DeviceControlRequest) {
	s.setAudioProbeOnly(req.DeviceID, req.TraceID, req.SessionID, req.AudioProbeOnly)
	s.setMockPlaybackOnNextAudioFrame(req.DeviceID, req.TraceID, req.SessionID, req.MockPlaybackOnNextAudioFrame, req.MockAudioChunks)
	s.setRealtimeOnNextSpeech(req.DeviceID, req.TraceID, req.SessionID, req.RealtimeOnNextSpeech)
}

func (s *Server) clearDeviceControlStreamState(req DeviceControlRequest) {
	s.setAudioProbeOnly(req.DeviceID, req.TraceID, req.SessionID, false)
	s.setMockPlaybackOnNextAudioFrame(req.DeviceID, req.TraceID, req.SessionID, false, nil)
	s.setRealtimeOnNextSpeech(req.DeviceID, req.TraceID, req.SessionID, false)
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
	health, err := s.currentVoiceProvider().Health(ctx)
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
		Provider:  s.currentVoiceProvider().Name(),
		Status:    status,
		Events:    events,
	})
}

func (s *Server) startRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.start_turn.start", s.now().UnixMilli())
	providerEvents, err := s.currentVoiceProvider().StartTurn(ctx, providers.VoiceTurnRequest{
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
		Provider:  s.currentVoiceProvider().Name(),
		Status:    status,
		Events:    events,
	})
}
