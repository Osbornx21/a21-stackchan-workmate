package gateway

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func sanitizedXiaozhiOTAHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" ||
		strings.ContainsAny(host, "/\\?#% \t\r\n") ||
		strings.Contains(host, "@") ||
		strings.Contains(strings.ToLower(host), "token") ||
		strings.Contains(strings.ToLower(host), "secret") {
		return ""
	}
	return host
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
	if req.MockAudioChunks != nil && (*req.MockAudioChunks < 0 || *req.MockAudioChunks > 8) {
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
	socket, ok := s.audioSocket(req.DeviceID)
	if !ok {
		http.Error(w, "device audio websocket is not connected", http.StatusConflict)
		return
	}

	events := s.deviceControlEvents(req)
	s.applyDeviceControlStreamState(req)
	for _, event := range events {
		if err := r.Context().Err(); err != nil {
			s.clearDeviceControlStreamState(req)
			http.Error(w, "device command delivery failed", http.StatusBadGateway)
			return
		}
		if err := writeAudioEnvelope(r.Context(), socket.conn, socket.writeMu, event); err != nil {
			s.clearDeviceControlStreamState(req)
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

func (s *Server) handleXiaozhiDeviceControl(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "invalid xiaozhi device event", http.StatusBadRequest)
		return
	}
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindState, xiaozhitransport.DeviceEventKindFace, xiaozhitransport.DeviceEventKindDisplay, xiaozhitransport.DeviceEventKindMotion:
	default:
		http.Error(w, "unsupported xiaozhi device control event", http.StatusBadRequest)
		return
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	if !socket.features.DeviceEvents {
		http.Error(w, "xiaozhi device events require debug profile negotiation", http.StatusConflict)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payload, err := xiaozhitransport.BuildDeviceExtensionEvent(xiaozhitransport.Identity{
		DeviceID:  req.DeviceID,
		TraceID:   traceID,
		SessionID: sessionID,
	}, socket.features, xiaozhitransport.DeviceExtensionProfileDebug, event)
	if err != nil {
		http.Error(w, "invalid xiaozhi device event", http.StatusBadRequest)
		return
	}
	socket.writeMu.Lock()
	err = socket.conn.Write(r.Context(), websocket.MessageText, payload)
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "xiaozhi device command delivery failed", http.StatusBadGateway)
		return
	}
	s.recordXiaozhiDeviceControlDelivered(req.DeviceID, traceID, sessionID, event)
	writeJSON(w, http.StatusOK, XiaozhiDeviceControlResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_ws",
		Event:              string(event.Kind),
		Value:              event.Value,
	})
}

const (
	xiaozhiSpeakerVolumeToolName          = "self.audio_speaker.set_volume"
	xiaozhiMCPGetDeviceStatusToolName     = "self.get_device_status"
	xiaozhiMCPScreenSetBrightnessToolName = "self.screen.set_brightness"
	xiaozhiMCPScreenSetThemeToolName      = "self.screen.set_theme"
	xiaozhiMCPScreenGetInfoToolName       = "self.screen.get_info"
	xiaozhiMCPRobotGetHeadAnglesToolName  = "self.robot.get_head_angles"
	xiaozhiMCPRobotSetHeadAnglesToolName  = "self.robot.set_head_angles"
	xiaozhiMCPRobotSetLEDColorToolName    = "self.robot.set_led_color"
)

func (s *Server) handleXiaozhiSay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiSayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(req.Text)
	wavPath := strings.TrimSpace(req.WAVPath)
	if text == "" && wavPath == "" {
		http.Error(w, "text or wav_path is required", http.StatusBadRequest)
		return
	}
	if text != "" && wavPath != "" {
		http.Error(w, "provide either text or wav_path, not both", http.StatusBadRequest)
		return
	}
	var wavChunks []providers.VoiceAudioChunk
	audioSource := ""
	audioBasename := ""
	if wavPath != "" {
		chunks, err := audio.ReadPCM16MonoWAVChunksForSampleRate(wavPath, 60, 16000)
		if err != nil || len(chunks) == 0 {
			http.Error(w, "wav_path must reference an A21-compatible 16 kHz mono PCM WAV", http.StatusBadRequest)
			return
		}
		audioSource = "wav_file"
		audioBasename = filepath.Base(wavPath)
		wavChunks = make([]providers.VoiceAudioChunk, 0, len(chunks))
		for _, chunk := range chunks {
			wavChunks = append(wavChunks, providers.VoiceAudioChunk{
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: chunk.SampleRateHz,
				Channels:     chunk.Channels,
				DurationMS:   chunk.DurationMS,
				DataBase64:   chunk.DataBase64,
			})
		}
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok || socket.session == nil {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	session := socket.session
	session.deviceID = req.DeviceID
	session.traceID = traceID
	session.sessionID = sessionID
	turn := session.startXiaozhiTurn(r.Context(), mode)
	session.resetXiaozhiTTSStop()
	task := xiaozhiTurnTask{
		turn:      turn,
		turnID:    xiaozhiTurnID(turn),
		traceID:   traceID,
		sessionID: sessionID,
		deviceID:  req.DeviceID,
		mode:      mode,
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.start", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(r.Context(), socket.conn, turn, map[string]any{
		"type":       "tts",
		"state":      "start",
		"phase":      "host_say",
		"turn_id":    task.turnID,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
	}); err != nil {
		http.Error(w, "xiaozhi say start delivery failed", http.StatusBadGateway)
		return
	}
	if err := session.writeXiaozhiJSON(r.Context(), socket.conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "host_say",
		"turn_id":    task.turnID,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
		"text":       "",
	}); err != nil {
		http.Error(w, "xiaozhi say sentence delivery failed", http.StatusBadGateway)
		return
	}
	var audioChunks int
	if wavPath != "" {
		audioChunks, ok = s.writeXiaozhiWAVAudioDownlink(r.Context(), socket.conn, session, task, wavChunks, "xiaozhi.say")
	} else {
		audioChunks, ok = s.writeXiaozhiTextAudioDownlink(r.Context(), socket.conn, session, task, text, mode, "xiaozhi.say")
	}
	if !ok {
		if interruptReason, interrupted := xiaozhiUserInterruptReason(session.xiaozhiTurnCancelReason(turn)); interrupted {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.interrupted", s.now().UnixMilli())
			writeJSON(w, http.StatusOK, XiaozhiSayResponse{
				TraceID:            traceID,
				SessionID:          sessionID,
				DeviceID:           req.DeviceID,
				Status:             "interrupted",
				DeliveredTransport: "xiaozhi_ws",
				TextChars:          len([]rune(text)),
				AudioChunks:        audioChunks,
				InterruptReason:    interruptReason,
				AudioSource:        audioSource,
				AudioBasename:      audioBasename,
			})
			return
		}
		s.writeXiaozhiTTSStop(r.Context(), socket.conn, session, turn, task, "host_say_unavailable")
		http.Error(w, "xiaozhi say audio delivery failed", http.StatusBadGateway)
		return
	}
	s.writeXiaozhiTTSStop(r.Context(), socket.conn, session, turn, task, "host_say_complete")
	s.suppressXiaozhiInputAfterHostSay(session, task)
	session.completeXiaozhiTurn(turn)
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.say.delivered", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, XiaozhiSayResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_ws",
		TextChars:          len([]rune(text)),
		AudioChunks:        audioChunks,
		AudioSource:        audioSource,
		AudioBasename:      audioBasename,
	})
}

func (s *Server) handleXiaozhiSpeakerVolume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiSpeakerVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	if req.Volume < 0 || req.Volume > 100 {
		http.Error(w, "volume must be 0..100", http.StatusBadRequest)
		return
	}
	socket, ok := s.xiaozhiSocket(req.DeviceID)
	if !ok {
		http.Error(w, "xiaozhi websocket is not connected", http.StatusConflict)
		return
	}
	if !socket.features.MCP {
		http.Error(w, "xiaozhi speaker volume requires stock MCP support", http.StatusConflict)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mcpID := "a21-mcp-speaker-volume-" + safeGatewayFallbackToken(traceID, "trace")
	payload, err := xiaozhitransport.BuildMCPToolsCallRequest(mcpID, xiaozhiSpeakerVolumeToolName, map[string]any{
		"volume": req.Volume,
	})
	if err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp request", http.StatusBadRequest)
		return
	}
	var payloadObject map[string]any
	if err := json.Unmarshal(payload, &payloadObject); err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp request", http.StatusBadRequest)
		return
	}
	message, err := json.Marshal(map[string]any{
		"type":       "mcp",
		"payload":    payloadObject,
		"trace_id":   traceID,
		"session_id": sessionID,
		"device_id":  req.DeviceID,
	})
	if err != nil {
		http.Error(w, "invalid xiaozhi speaker volume mcp message", http.StatusBadRequest)
		return
	}
	socket.writeMu.Lock()
	err = socket.conn.Write(r.Context(), websocket.MessageText, message)
	socket.writeMu.Unlock()
	if err != nil {
		http.Error(w, "xiaozhi speaker volume delivery failed", http.StatusBadGateway)
		return
	}
	s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.mcp.speaker_volume.sent", s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(&xiaozhiSession{deviceID: req.DeviceID, traceID: traceID, sessionID: sessionID}, "xiaozhi.mcp.speaker_volume.sent", map[string]string{
		"speaker_volume": strconv.Itoa(req.Volume),
	})
	writeJSON(w, http.StatusOK, XiaozhiSpeakerVolumeResponse{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp",
		ToolName:           xiaozhiSpeakerVolumeToolName,
		Volume:             req.Volume,
		MCPID:              mcpID,
	})
}

func (s *Server) handleXiaozhiDeviceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiDeviceStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:  req.DeviceID,
		ToolName:  xiaozhiMCPGetDeviceStatusToolName,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiScreenBrightness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiScreenBrightnessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:   req.DeviceID,
		ToolName:   xiaozhiMCPScreenSetBrightnessToolName,
		Brightness: req.Brightness,
		TraceID:    req.TraceID,
		SessionID:  req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiScreenTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiScreenThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validXiaozhiNamedScreenTheme(req.Theme) {
		http.Error(w, "theme must be light, dark, or auto", http.StatusBadRequest)
		return
	}
	response, status, message := s.deliverXiaozhiMCPControl(r.Context(), XiaozhiMCPControlRequest{
		DeviceID:  req.DeviceID,
		ToolName:  xiaozhiMCPScreenSetThemeToolName,
		Theme:     req.Theme,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleXiaozhiBodyPreset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodyPresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	preset, plans, err := xiaozhiBodyPresetPlans(req.DeviceID, req.Preset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	steps := make([]XiaozhiBodyPresetStepResponse, 0, len(plans))
	for _, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_preset."+preset+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		presetMarker := "xiaozhi.body_preset." + preset + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, presetMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_body_preset"] = preset
		activity["last_body_preset_status"] = "delivered"
		activity["last_body_preset_tool"] = delivery.Marker
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, presetMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	writeJSON(w, http.StatusOK, XiaozhiBodyPresetResponse{
		SchemaVersion:      "a21.gateway.xiaozhi_body_preset.v1",
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp_sequence",
		Preset:             preset,
		Steps:              steps,
		ResultRedacted:     true,
		PhysicalAccepted:   false,
	})
}

func (s *Server) handleXiaozhiBodyMotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req XiaozhiBodyMotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validA21DeviceID(req.DeviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	motion, plans, err := xiaozhiBodyMotionPlans(req.DeviceID, req.Motion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	steps := make([]XiaozhiBodyPresetStepResponse, 0, len(plans))
	for index, plan := range plans {
		plan.TraceID = traceID
		plan.SessionID = sessionID
		delivery, status, message := s.sendXiaozhiMCPControl(r.Context(), plan)
		if status != 0 {
			s.recordTrace(traceID, sessionID, req.DeviceID, "xiaozhi.body_motion."+motion+".failed", s.now().UnixMilli())
			http.Error(w, message, status)
			return
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		motionMarker := "xiaozhi.body_motion." + motion + ".step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, motionMarker, s.now().UnixMilli())
		activity := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		activity["last_body_motion"] = motion
		activity["last_body_motion_status"] = "delivered"
		activity["last_body_motion_step"] = strconv.Itoa(index + 1)
		activity["last_body_motion_tool"] = delivery.Marker
		s.recordXiaozhiDeviceActivity(&xiaozhiSession{
			deviceID:  delivery.Response.DeviceID,
			traceID:   delivery.Response.TraceID,
			sessionID: delivery.Response.SessionID,
		}, motionMarker, activity)
		steps = append(steps, XiaozhiBodyPresetStepResponse{
			ToolName:  delivery.Response.ToolName,
			MCPID:     delivery.Response.MCPID,
			Marker:    delivery.Marker,
			Arguments: delivery.Response.Arguments,
		})
	}
	writeJSON(w, http.StatusOK, XiaozhiBodyMotionResponse{
		SchemaVersion:      "a21.gateway.xiaozhi_body_motion.v1",
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           req.DeviceID,
		Status:             "delivered",
		DeliveredTransport: "xiaozhi_mcp_sequence",
		Motion:             motion,
		Steps:              steps,
		ResultRedacted:     true,
		PhysicalAccepted:   false,
	})
}
