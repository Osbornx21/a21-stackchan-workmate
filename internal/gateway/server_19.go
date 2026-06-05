package gateway

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func (s *Server) writeXiaozhiTTSStopWithOptions(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string, force bool) bool {
	if conn == nil {
		return false
	}
	session.writeMu.Lock()
	if !force && turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		session.writeMu.Unlock()
		return false
	}
	if !force && !session.claimXiaozhiTTSStop() {
		session.writeMu.Unlock()
		return false
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.tts.stop", s.now().UnixMilli())
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "tts",
		"state":      "stop",
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"reason":     reason,
	}); err != nil {
		session.writeMu.Unlock()
		return false
	}
	session.writeMu.Unlock()
	s.writeXiaozhiOfficialStackChanState(ctx, session, xiaozhiOfficialStateForTTSStop(reason), "tts_stop_"+safeGatewayFallbackToken(reason, "unknown"), true)
	if xiaozhiShouldSuppressInputAfterTTSStop(reason) {
		untilMS := s.now().UnixMilli() + xiaozhiPostTTSInputCooldownMS
		session.suppressXiaozhiInputUntil(untilMS, "post_tts_drain")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.tts.stop.input_suppression_armed", s.now().UnixMilli())
	}
	return true
}

func xiaozhiShouldSuppressInputAfterTTSStop(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" {
		return false
	}
	if strings.Contains(reason, "abort") || strings.Contains(reason, "barge") || strings.Contains(reason, "wake") {
		return false
	}
	return strings.Contains(reason, "local_fallback") ||
		strings.Contains(reason, "placeholder") ||
		strings.Contains(reason, "host_say_complete") ||
		strings.Contains(reason, "degraded") ||
		strings.Contains(reason, "error") ||
		strings.Contains(reason, "unavailable")
}

func (s *Server) writeXiaozhiOpusDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, chunk providers.VoiceAudioChunk) (bool, error) {
	if session == nil || session.shouldAbortXiaozhiTurn(turn) {
		s.recordXiaozhiStaleDownlinkSuppressed(session)
		return false, nil
	}
	if conn == nil {
		return false, fmt.Errorf("xiaozhi downlink websocket is nil")
	}
	if turn.pacer == nil {
		turn.pacer = audio.NewAudioRateController(audio.AudioRateControllerConfig{
			FrameDuration:   60 * time.Millisecond,
			PrebufferFrames: 1,
		})
	}
	pcm, err := xiaozhiDownlinkPCM16(chunk)
	if err != nil {
		return false, err
	}
	codec, err := turn.xiaozhiDownlinkCodec(chunk)
	if err != nil {
		return false, err
	}
	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		return false, err
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "speaking", "opus_downlink", false)
	return turn.pacer.Send(ctx, packet, func(ctx context.Context, frame []byte) error {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordXiaozhiStaleDownlinkSuppressed(session)
			return context.Canceled
		}
		if err := session.writeXiaozhiBinary(ctx, conn, turn, frame); err != nil {
			return err
		}
		nowMS := s.now().UnixMilli()
		session.markXiaozhiDownlink(turn, nowMS)
		s.recordTrace(session.traceID, session.sessionID, session.deviceID, "xiaozhi.tts.opus_frame.downlink", nowMS)
		s.recordXiaozhiDeviceActivity(session, "xiaozhi.tts.opus_frame.downlink", map[string]string{
			"speaker": "available_xiaozhi_opus_downlink",
		})
		return nil
	}, func() bool {
		return session.shouldAbortXiaozhiTurn(turn)
	})
}

func (s *Server) recordXiaozhiStaleDownlinkSuppressed(session *xiaozhiSession) {
	if session == nil {
		return
	}
	s.recordTrace(session.traceID, session.sessionID, session.deviceID, "xiaozhi.tts.stale_frame_suppressed", s.now().UnixMilli())
}

func (turn *xiaozhiTurn) xiaozhiDownlinkCodec(chunk providers.VoiceAudioChunk) (*opuscodec.Codec, error) {
	if turn == nil {
		return nil, fmt.Errorf("%w: nil xiaozhi turn", opuscodec.ErrUnsupportedConfig)
	}
	if turn.downlink != nil &&
		turn.downlink.sampleRateHz == chunk.SampleRateHz &&
		turn.downlink.channels == chunk.Channels &&
		turn.downlink.durationMS == chunk.DurationMS &&
		turn.downlink.codec != nil {
		return turn.downlink.codec, nil
	}
	codec, err := opuscodec.New(chunk.SampleRateHz, chunk.Channels, chunk.DurationMS)
	if err != nil {
		return nil, err
	}
	turn.downlink = &xiaozhiTurnDownlinkCodec{
		sampleRateHz: chunk.SampleRateHz,
		channels:     chunk.Channels,
		durationMS:   chunk.DurationMS,
		codec:        codec,
	}
	return codec, nil
}

const xiaozhiDownlinkPCM16HeadroomPeak = 29490

const xiaozhiDownlinkPCM16TargetPeak = 24576

const xiaozhiDownlinkPCM16MaxGainMilli = 3000

const xiaozhiDownlinkPCM16NoiseGatePeak = 512

func xiaozhiDownlinkPCM16(chunk providers.VoiceAudioChunk) ([]int16, error) {
	if chunk.Codec != string(protocol.AudioCodecPCMS16LE) {
		return nil, fmt.Errorf("xiaozhi downlink requires pcm_s16le provider audio")
	}
	if (chunk.SampleRateHz != 16000 && chunk.SampleRateHz != 24000 && chunk.SampleRateHz != 48000) || chunk.Channels != 1 || chunk.DurationMS != 60 {
		return nil, fmt.Errorf("xiaozhi downlink requires 16kHz, 24kHz, or 48kHz mono 60ms audio")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(chunk.DataBase64))
	if err != nil {
		return nil, fmt.Errorf("xiaozhi downlink audio must be valid base64")
	}
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("xiaozhi downlink pcm_s16le byte length must be even")
	}
	pcm := make([]int16, len(data)/2)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	applyXiaozhiDownlinkLeveling(pcm)
	return pcm, nil
}

func applyXiaozhiDownlinkLeveling(pcm []int16) {
	maxAbs := 0
	for _, sample := range pcm {
		abs := int(sample)
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	if maxAbs == 0 || maxAbs < xiaozhiDownlinkPCM16NoiseGatePeak {
		return
	}
	targetPeak := maxAbs
	if maxAbs > xiaozhiDownlinkPCM16HeadroomPeak {
		targetPeak = xiaozhiDownlinkPCM16HeadroomPeak
	} else if maxAbs < xiaozhiDownlinkPCM16TargetPeak {
		targetPeak = xiaozhiDownlinkPCM16TargetPeak
		maxBoostedPeak := maxAbs * xiaozhiDownlinkPCM16MaxGainMilli / 1000
		if maxBoostedPeak < targetPeak {
			targetPeak = maxBoostedPeak
		}
		if targetPeak > xiaozhiDownlinkPCM16HeadroomPeak {
			targetPeak = xiaozhiDownlinkPCM16HeadroomPeak
		}
	}
	if targetPeak == maxAbs {
		return
	}
	for i, sample := range pcm {
		pcm[i] = int16(int(sample) * targetPeak / maxAbs)
	}
}

func (s *Server) xiaozhiHelloReply(session *xiaozhiSession) map[string]any {
	audioParams := map[string]any{
		"format":         "opus",
		"sample_rate":    24000,
		"channels":       1,
		"frame_duration": 60,
	}
	reply := map[string]any{
		"type":         "hello",
		"version":      session.binaryProtocolVersion,
		"transport":    "websocket",
		"trace_id":     session.traceID,
		"session_id":   session.sessionID,
		"device_id":    session.deviceID,
		"audio":        audioParams,
		"audio_params": audioParams,
	}
	if session.features.DeviceEvents {
		reply["a21"] = map[string]any{
			"profile":       "debug",
			"device_events": true,
		}
	} else if s.xiaozhiProductPlaybackEventsAllowed(session) || s.xiaozhiProductKeepaliveEventsAllowed(session) || s.xiaozhiProductTouchEventsAllowed(session) || s.xiaozhiProductStateReactionsAllowed(session) {
		a21 := map[string]any{
			"profile": "product",
		}
		if s.xiaozhiProductPlaybackEventsAllowed(session) {
			a21["playback_events"] = true
		}
		if s.xiaozhiProductKeepaliveEventsAllowed(session) {
			a21["keepalive_events"] = true
		}
		if s.xiaozhiProductTouchEventsAllowed(session) {
			a21["touch_events"] = true
		}
		if s.xiaozhiProductTouchReactionsAllowed(session) {
			a21["touch_reactions"] = true
		}
		if s.xiaozhiProductStateReactionsAllowed(session) {
			a21["state_reactions"] = true
		}
		reply["a21"] = a21
	}
	return reply
}

func xiaozhiBinaryProfile(version int) string {
	if version <= 1 {
		return "xiaozhi_binary_v1_raw"
	}
	return fmt.Sprintf("xiaozhi_binary_v%d", version)
}

func (s *Server) xiaozhiBaseReply(session *xiaozhiSession, msgType string, state string, status string, turnID string) map[string]any {
	reply := map[string]any{
		"type":       msgType,
		"state":      state,
		"status":     status,
		"trace_id":   session.traceID,
		"session_id": session.sessionID,
		"device_id":  session.deviceID,
	}
	if turnID != "" {
		reply["turn_id"] = turnID
	}
	return reply
}

func (s *Server) xiaozhiError(session *xiaozhiSession, code string, detail string) map[string]any {
	traceID, sessionID := s.ids(session.traceID, session.sessionID)
	session.traceID = traceID
	session.sessionID = sessionID
	return map[string]any{
		"type":       "error",
		"code":       code,
		"detail":     detail,
		"trace_id":   session.traceID,
		"session_id": session.sessionID,
		"device_id":  session.deviceID,
	}
}

func xiaozhiErrorCode(err error) string {
	switch {
	case errors.Is(err, xiaozhitransport.ErrMalformedJSON):
		return "invalid_json"
	case errors.Is(err, xiaozhitransport.ErrMissingDeviceIdentity), errors.Is(err, xiaozhitransport.ErrLegacyIdentity):
		return "invalid_device_id"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedMessageType):
		return "unsupported_message_type"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedListenState):
		return "unsupported_listen_state"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedHelloVersion):
		return "unsupported_hello_version"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedTransport):
		return "unsupported_transport"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedAudioParams):
		return "unsupported_audio_params"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedBinaryProtocol):
		return "unsupported_binary_protocol_version"
	case errors.Is(err, xiaozhitransport.ErrMalformedBinaryFrame):
		return "malformed_binary_frame"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedBinaryFrameType):
		return "unsupported_binary_frame_type"
	case errors.Is(err, xiaozhitransport.ErrEmptyBinaryPayload):
		return "empty_binary_payload"
	case errors.Is(err, xiaozhitransport.ErrUnexpectedBinaryDirection):
		return "unexpected_binary_frame_direction"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedDeviceEventKind):
		return "unsupported_device_event_kind"
	case errors.Is(err, xiaozhitransport.ErrUnsupportedDeviceEventValue):
		return "unsupported_device_event_value"
	default:
		return "invalid_xiaozhi_message"
	}
}

func xiaozhiErrorDetail(err error) string {
	if err == nil {
		return "invalid xiaozhi message"
	}
	return err.Error()
}

func (s *Server) handleOfficialStackChanWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := officialStackChanDeviceID(r)
	if !validA21DeviceID(deviceID) {
		http.Error(w, "valid device_id is required", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	writeMu := &sync.Mutex{}
	s.registerOfficialStackChanSocket(deviceID, conn, writeMu)
	s.recordOfficialStackChanConnected(deviceID)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer s.unregisterOfficialStackChanSocket(deviceID, conn)
	defer conn.Close(websocket.StatusNormalClosure, "official stackchan websocket closed")

	go s.writeOfficialStackChanHeartbeat(ctx, conn, writeMu)

	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if messageType == websocket.MessageBinary && len(data) > 0 && data[0] == stackchantransport.DataTypeHeartbeatPong {
			s.recordTrace("", "", deviceID, "stackchan.official_ws.heartbeat_pong", s.now().UnixMilli())
		}
	}
}

func officialStackChanDeviceID(r *http.Request) string {
	query := r.URL.Query()
	deviceID := firstNonEmpty(query.Get("device_id"), query.Get("deviceId"), query.Get("id"))
	if strings.TrimSpace(deviceID) == "" {
		return defaultOfficialStackChanDeviceID
	}
	return strings.TrimSpace(deviceID)
}

func (s *Server) writeOfficialStackChanHeartbeat(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	packet := stackchantransport.OfficialPacket{Type: stackchantransport.DataTypeHeartbeatPing}.Bytes()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			writeMu.Lock()
			err := conn.Write(ctx, websocket.MessageBinary, packet)
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
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
		if frame.Kind == protocol.KindDeviceEvent {
			events := s.controlEventsForDeviceEvent(frame)
			for _, event := range events {
				if err := writeAudioEnvelope(ctx, conn, writeMu, event); err != nil {
					return
				}
			}
			continue
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
		mockPlaybackChunks, shouldEmitMockAudioPlayback := s.mockAudioPlaybackChunkCount(frame, traceID, sessionID, ingress)
		if !shouldEmitMockAudioPlayback {
			continue
		}
		if mockPlaybackChunks <= 0 {
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
		for i := 0; i < mockPlaybackChunks; i++ {
			playback := s.mockAudioPlaybackChunk(frame, traceID, sessionID, streamID, uint64(len(events)+i+1))
			if err := writeAudioEnvelope(ctx, conn, writeMu, playback); err != nil {
				return
			}
		}
		s.setActiveStream(traceID, sessionID, frame.DeviceID, streamID)
	}
}

func (s *Server) mockAudioPlaybackChunkCount(frame protocol.Envelope, traceID string, sessionID string, ingress audio.IngressResult) (int, bool) {
	if frame.Kind != protocol.KindAudioFrame {
		return 1, true
	}
	if !physicalStackChanDeviceID(frame.DeviceID) {
		return 1, true
	}
	if chunks, armed := s.consumeMockPlaybackOnNextAudioFrame(frame.DeviceID, traceID, sessionID); armed {
		s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.armed", s.now().UnixMilli())
		return chunks, true
	}
	if containsAudioIngressEvent(ingress.Events, audio.EventVADSpeechStart) {
		return 1, true
	}
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.mock_physical.suppressed", s.now().UnixMilli())
	return 0, false
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
		VADStatus:           ingress.VADStatus,
		VADFinding:          ingress.VADFinding,
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
