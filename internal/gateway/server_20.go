package gateway

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func (s *Server) realtimeAudioEvents(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult, connectionSessionKeys map[string]struct{}) ([]protocol.Envelope, bool) {
	provider, ok := s.currentVoiceProvider().(providers.RealtimeVoiceProvider)
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
	providerEvents, err := s.currentVoiceProvider().Cancel(context.Background(), providers.VoiceCancelRequest{
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
	summary.XiaozhiListenToAudioIngressMS = traceDeltaMS(events, "xiaozhi.listen.start", "audio.ingress.buffered")
	summary.XiaozhiOpusDecodeMS = traceDeltaMS(events, "xiaozhi.opus_frame.received", "xiaozhi.opus_frame.decoded")
	summary.ASRFirstPartialMS = firstTraceDelta(
		traceDeltaMS(events, "audio.ingress.buffered", "asr.first_partial"),
		traceDeltaMS(events, "audio.ingress.buffered", "asr.final"),
	)
	llmStartName := "asr.first_partial"
	if !traceHasEvent(events, llmStartName) {
		llmStartName = "asr.final"
	}
	summary.LLMFirstContentMS = traceDeltaMS(events, llmStartName, "provider.first_content")
	summary.TTSFirstAudioMS = traceDeltaMS(events, "provider.first_content", "tts.first_audio")
	summary.AudioDownlinkFirstFrameMS = traceDeltaMS(events, "tts.first_audio", "audio.downlink.first_frame")
	summary.DevicePlaybackStartMS = traceDeltaMS(events, "audio.downlink.first_frame", "device.playback.start")
	summary.AnswerFirstAudioTotalMS = traceDeltaMS(events, "audio.ingress.buffered", "audio.downlink.first_frame")
	return summary
}

func traceDeltaMS(events []TraceEvent, startName string, endName string) *int64 {
	var startAtMS int64
	hasStart := false
	var latestDelta *int64
	for _, event := range events {
		switch {
		case event.Name == startName:
			startAtMS = event.AtMS
			hasStart = true
		case event.Name == endName && hasStart:
			delta := event.AtMS - startAtMS
			if delta < 0 {
				delta = 0
			}
			latestDelta = &delta
			hasStart = false
		}
	}
	return latestDelta
}

func firstTraceDelta(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func traceHasEvent(events []TraceEvent, name string) bool {
	for _, event := range events {
		if event.Name == name {
			return true
		}
	}
	return false
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
	displayState := payload.DisplayState
	s.mu.Unlock()
	if displayState != "" {
		s.recordDeviceDisplayState(event.DeviceID, event.TraceID, event.SessionID, "device_event", string(displayState))
		s.mu.Lock()
		record = s.devices[event.DeviceID]
		s.mu.Unlock()
		return record
	}
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
	roleplayState := s.currentRoleplayDeviceState()
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
	record.CurrentVoiceMode = defaultVoiceMode(s.voiceModeConfig)
	record.CurrentVoiceChainMode = defaultVoiceChainMode(s.voiceChainModeConfig)
	record.CurrentASRProfile = defaultCascadeASRProfile(s.cascadeASRProfileConfig)
	record.CurrentLLMProfile = defaultCascadeLLMProfile(s.cascadeLLMProfileConfig)
	record.CurrentTTSProfile = voiceChainTTSForVoice(defaultFixedTTSProfile(s.fixedTTSProfileConfig), defaultVoiceCloneProfile(s.voiceCloneProfileConfig))
	record.CurrentRealtimeProvider = defaultRealtimeProvider(s.realtimeProviderConfig)
	record.CurrentVoiceCloneProfile = defaultVoiceCloneProfile(s.voiceCloneProfileConfig)
	applyRoleplayDeviceState(&record, roleplayState)
	record.CurrentCloudVoiceProfile = providers.DefaultCloudVoiceProfile(s.cloudVoiceProfileConfig)
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, roleplayDeviceRuntimeEcho(roleplayState))
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

func (s *Server) registerXiaozhiSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex, session *xiaozhiSession, features xiaozhitransport.HelloFeatures) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil || session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.xiaozhiSockets[deviceID] = &xiaozhiDeviceSocket{
		conn:      conn,
		writeMu:   writeMu,
		session:   session,
		features:  features,
		connected: s.now().UnixMilli(),
	}
}

func (s *Server) unregisterXiaozhiSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.xiaozhiSockets[deviceID]
	if socket != nil && socket.conn == conn {
		delete(s.xiaozhiSockets, deviceID)
		nowMS := s.now().UnixMilli()
		record := s.devices[deviceID]
		if record.DeviceID == "" {
			record.DeviceID = deviceID
			record.FirstSeenMS = nowMS
		}
		if record.IdentityStatus == "" {
			record.IdentityStatus = "unknown"
		}
		record.ConnectionStatus = "xiaozhi_ws_disconnected"
		if socket.session != nil {
			record.LastTraceID = socket.session.traceID
			record.LastSessionID = socket.session.sessionID
		}
		record.LastSeenMS = nowMS
		s.devices[deviceID] = record
	}
}

func (s *Server) xiaozhiSocket(deviceID string) (*xiaozhiDeviceSocket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.xiaozhiSockets[deviceID]
	return socket, socket != nil
}

func (s *Server) registerOfficialStackChanSocket(deviceID string, conn *websocket.Conn, writeMu *sync.Mutex) {
	if !validA21DeviceID(deviceID) || conn == nil || writeMu == nil {
		return
	}
	key := deviceIDLookupKey(deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.officialStackChanSockets[key] = &deviceSocket{conn: conn, writeMu: writeMu, connected: s.now().UnixMilli()}
}

func (s *Server) unregisterOfficialStackChanSocket(deviceID string, conn *websocket.Conn) {
	if deviceID == "" || conn == nil {
		return
	}
	key := deviceIDLookupKey(deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.officialStackChanSockets[key]
	if socket != nil && socket.conn == conn {
		delete(s.officialStackChanSockets, key)
	}
}

func (s *Server) officialStackChanSocket(deviceID string) (*deviceSocket, bool) {
	key := deviceIDLookupKey(deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	socket := s.officialStackChanSockets[key]
	return socket, socket != nil
}

func (s *Server) officialStackChanSocketForXiaozhiDevice(deviceID string) (*deviceSocket, string, bool) {
	key := deviceIDLookupKey(deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if socket := s.officialStackChanSockets[key]; socket != nil {
		return socket, key, true
	}
	if socket := s.officialStackChanSockets[defaultOfficialStackChanDeviceID]; socket != nil {
		return socket, defaultOfficialStackChanDeviceID, true
	}
	return nil, "", false
}

func (s *Server) officialStackChanStatus(deviceID string) OfficialStackChanStatusResponse {
	nowMS := s.now().UnixMilli()
	deviceID = strings.TrimSpace(deviceID)
	key := deviceIDLookupKey(deviceID)
	var record DeviceRecord
	var found bool
	officialDeviceID := ""
	connectedSinceMS := int64(0)
	s.mu.Lock()
	record, found = s.devices[key]
	if socket := s.officialStackChanSockets[key]; socket != nil {
		officialDeviceID = key
		connectedSinceMS = socket.connected
	} else if socket := s.officialStackChanSockets[defaultOfficialStackChanDeviceID]; socket != nil {
		officialDeviceID = defaultOfficialStackChanDeviceID
		connectedSinceMS = socket.connected
	}
	if found && record.RuntimeEcho != nil {
		record.RuntimeEcho = mergeDeviceCapabilities(nil, record.RuntimeEcho)
	}
	s.mu.Unlock()

	connected := connectedSinceMS > 0
	deliveredTransport := "xiaozhi_mcp_fallback_available"
	connectedMS := int64(0)
	if connected {
		deliveredTransport = "stackchan_official_ws"
		connectedMS = nowMS - connectedSinceMS
		if connectedMS < 0 {
			connectedMS = 0
		}
	}
	runtimeEcho := record.RuntimeEcho
	packetCount := officialStackChanStatusPacketCount(runtimeEcho)
	physicalAccepted := officialStackChanStatusPhysicalAccepted(runtimeEcho)
	nextAction := "connect_official_stackchan_ws"
	if connected && physicalAccepted {
		nextAction = "official_relay_ready"
	} else if connected {
		nextAction = "send_official_control_and_collect_physical_acceptance"
	}
	lastEvent := ""
	if found && record.LastEvent != "" {
		lastEvent = string(record.LastEvent)
	}
	return OfficialStackChanStatusResponse{
		SchemaVersion:          "a21.stackchan.official.status.v1",
		DeviceID:               deviceID,
		OfficialDeviceID:       officialDeviceID,
		Connected:              connected,
		ConnectedMS:            connectedMS,
		ConnectedSinceMS:       connectedSinceMS,
		FallbackAvailable:      true,
		DeliveredTransport:     deliveredTransport,
		LastTraceID:            record.LastTraceID,
		LastSessionID:          record.LastSessionID,
		LastEvent:              lastEvent,
		LastAutoState:          runtimeEcho["official_stackchan_auto_state"],
		LastAutoReason:         runtimeEcho["official_stackchan_auto_reason"],
		LastAutoTarget:         runtimeEcho["official_stackchan_auto_target"],
		LastPacketCount:        packetCount,
		PhysicalAccepted:       physicalAccepted,
		OfficialActionSurfaces: officialStackChanStatusSurfaces(runtimeEcho),
		NextAction:             nextAction,
	}
}
