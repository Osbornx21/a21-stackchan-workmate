package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func xiaozhiOpusDecodeStatusFromCounts(decodedFrameCount int, decodeErrorCount int) string {
	if decodeErrorCount > 0 && decodedFrameCount > 0 {
		return XiaozhiOpusPartialDecodeErrorState
	}
	if decodeErrorCount > 0 {
		return XiaozhiOpusDecodeErrorState
	}
	if decodedFrameCount > 0 {
		return XiaozhiOpusDecodedPCMState
	}
	return XiaozhiOpusNoFramesState
}

func (session *xiaozhiSession) xiaozhiDecodedDurationMS() int {
	if session == nil {
		return 0
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.xiaozhiDecodedDurationMSLocked()
}

func (session *xiaozhiSession) xiaozhiDecodedDurationMSLocked() int {
	if session.opusSampleRateHz <= 0 {
		return 0
	}
	return session.opusDecodedSampleCount * 1000 / session.opusSampleRateHz
}

func copyXiaozhiVoicePipelineFrames(frames []providers.VoicePipelinePCMFrame) []providers.VoicePipelinePCMFrame {
	if len(frames) == 0 {
		return nil
	}
	copied := append([]providers.VoicePipelinePCMFrame(nil), frames...)
	for i := range copied {
		copied[i].PCM16LE = append([]byte(nil), copied[i].PCM16LE...)
	}
	return copied
}

func (s *Server) startXiaozhiStreamingASR(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, mode protocol.Mode) {
	if s == nil || session == nil {
		return
	}
	streamingASR := s.currentXiaozhiVoicePipelineASR()
	if streamingASR == nil {
		return
	}
	streaming, ok := streamingASR.(providers.StreamingASRAdapter)
	if !ok {
		return
	}
	id := session.identitySnapshot()
	stream, err := streaming.StartStreamingASR(ctx, providers.StreamingASRStartRequest{
		Session: providers.VoiceSession{
			TraceID:   id.traceID,
			SessionID: id.sessionID,
			DeviceID:  id.deviceID,
		},
		Mode: string(mode),
	})
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.unavailable", s.now().UnixMilli())
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.start_failed."+xiaozhiStreamingASRErrorCode(err), s.now().UnixMilli())
		return
	}
	session.mu.Lock()
	session.streamingASRSession = stream
	session.streamingASRHasPartial = false
	session.streamingASRPartialText = ""
	session.streamingASRHasFinal = false
	session.streamingASRFinalText = ""
	session.streamingASRClosed = false
	session.streamingASRCommitStarted = false
	session.streamingASRAnswerStarted = false
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.start", s.now().UnixMilli())
	go s.consumeXiaozhiStreamingASREvents(ctx, conn, session, stream)
}

func xiaozhiStreamingASRErrorCode(err error) string {
	message := strings.ToLower(strings.TrimSpace(errString(err)))
	switch {
	case message == "":
		return "unknown"
	case strings.Contains(message, "missing"):
		return "missing_env"
	case strings.Contains(message, "dial") || strings.Contains(message, "websocket"):
		return "dial_failed"
	case strings.Contains(message, "session") || strings.Contains(message, "update"):
		return "session_update_failed"
	default:
		return "provider_failed"
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) consumeXiaozhiStreamingASREvents(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, stream providers.StreamingASRSession) {
	for event := range stream.Events() {
		if event.Err != nil {
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.error", s.now().UnixMilli())
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.error."+xiaozhiStreamingASREventErrorCode(event), s.now().UnixMilli())
			continue
		}
		partialText := strings.TrimSpace(event.Text)
		if partialText != "" {
			recordPartial := false
			session.mu.Lock()
			if !session.streamingASRHasPartial {
				session.streamingASRHasPartial = true
				session.streamingASRPartialText = partialText
				recordPartial = true
			}
			session.mu.Unlock()
			if recordPartial {
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.first_partial", s.now().UnixMilli())
				s.startXiaozhiPartialVoicePipeline(ctx, conn, session, partialText)
			}
		}
		if event.Final {
			session.mu.Lock()
			session.streamingASRHasFinal = true
			session.streamingASRFinalText = event.Text
			if strings.TrimSpace(event.Text) != "" {
				session.voicePipelineHasSpeech = true
			}
			session.mu.Unlock()
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.final", s.now().UnixMilli())
			s.maybeStartXiaozhiStreamingASRFinalAnswer(ctx, conn, session)
		}
	}
	session.mu.Lock()
	if session.streamingASRSession == stream {
		session.streamingASRClosed = true
	}
	session.mu.Unlock()
}

func xiaozhiStreamingASREventErrorCode(event providers.ASRAdapterEvent) string {
	message := strings.ToLower(strings.TrimSpace(event.Finding + " " + errString(event.Err)))
	switch {
	case message == "":
		return "unknown"
	case strings.Contains(message, "missing"):
		return "missing_env"
	case strings.Contains(message, "401") ||
		strings.Contains(message, "403") ||
		strings.Contains(message, "auth") ||
		strings.Contains(message, "api key") ||
		strings.Contains(message, "apikey") ||
		strings.Contains(message, "bearer"):
		return "auth_failed"
	case strings.Contains(message, "model"):
		return "model_or_profile"
	case strings.Contains(message, "audio") ||
		strings.Contains(message, "pcm") ||
		strings.Contains(message, "sample") ||
		strings.Contains(message, "format"):
		return "audio_format"
	case strings.Contains(message, "quota") ||
		strings.Contains(message, "rate") ||
		strings.Contains(message, "limit"):
		return "rate_limited_or_quota"
	case strings.Contains(message, "session") ||
		strings.Contains(message, "update"):
		return "session_update_failed"
	case strings.Contains(message, "commit") ||
		strings.Contains(message, "finish"):
		return "commit_failed"
	case strings.Contains(message, "dial") ||
		strings.Contains(message, "websocket") ||
		strings.Contains(message, "network"):
		return "network"
	case strings.Contains(message, "read") ||
		strings.Contains(message, "eof"):
		return "read_failed"
	case strings.Contains(message, "invalid") ||
		strings.Contains(message, "parameter"):
		return "invalid_request"
	default:
		return "provider_failed"
	}
}

func (s *Server) startXiaozhiPartialVoicePipeline(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, partialText string) {
	if s == nil || session == nil || conn == nil || strings.TrimSpace(partialText) == "" {
		return
	}
	session.mu.Lock()
	if session.streamingASRAnswerStarted || !session.listening || session.currentTurn == nil || len(session.voicePipelineFrames) == 0 {
		session.mu.Unlock()
		return
	}
	session.voicePipelineHasSpeech = true
	session.mu.Unlock()

	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.voice_pipeline.partial_prewarm_deferred", s.now().UnixMilli())
}

func (session *xiaozhiSession) claimXiaozhiStreamingASRAnswer(turn *xiaozhiTurn) bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if turn == nil || session.currentTurn != turn || session.streamingASRAnswerStarted {
		return false
	}
	session.streamingASRAnswerStarted = true
	return true
}

func (s *Server) appendXiaozhiStreamingASRFrame(ctx context.Context, session *xiaozhiSession, frame providers.VoicePipelinePCMFrame) {
	if session == nil {
		return
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	closed := session.streamingASRClosed
	session.mu.Unlock()
	if stream == nil || closed {
		return
	}
	if err := stream.AppendFrame(ctx, frame); err != nil {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.audio.append_error", s.now().UnixMilli())
		return
	}
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.audio.append", s.now().UnixMilli())
}

func (s *Server) startXiaozhiStreamingASRCommit(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn) bool {
	if session == nil || conn == nil || turn == nil {
		return false
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	closed := session.streamingASRClosed
	started := session.streamingASRCommitStarted
	if stream != nil && !closed && !started {
		session.streamingASRCommitStarted = true
	}
	session.mu.Unlock()
	if stream == nil || closed {
		return false
	}
	if started {
		return true
	}
	go s.finishXiaozhiStreamingASRCommit(ctx, conn, session, turn, stream)
	return true
}

func (s *Server) finishXiaozhiStreamingASRCommit(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, stream providers.StreamingASRSession) {
	commitCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.commit", s.now().UnixMilli())
	if err := stream.Commit(commitCtx); err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.commit_error", s.now().UnixMilli())
		return
	}
	if !s.waitXiaozhiStreamingASRFinal(session, 200*time.Millisecond) {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.final_timeout", s.now().UnixMilli())
		return
	}
	s.maybeStartXiaozhiStreamingASRFinalAnswer(ctx, conn, session)
}

func (s *Server) maybeStartXiaozhiStreamingASRFinalAnswer(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession) bool {
	if s == nil || session == nil || conn == nil {
		return false
	}
	session.mu.Lock()
	turn := session.currentTurn
	listening := session.listening
	commitStarted := session.streamingASRCommitStarted
	hasFinal := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
	session.mu.Unlock()
	if listening || !commitStarted || !hasFinal || turn == nil || session.shouldAbortXiaozhiTurn(turn) {
		return false
	}
	if !session.claimXiaozhiStreamingASRAnswer(turn) {
		return false
	}
	task := s.newXiaozhiTurnTask(session, turn)
	if strings.TrimSpace(task.streamingASRFinalText) == "" {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.final_empty", s.now().UnixMilli())
		return false
	}
	s.startXiaozhiTurnTask(ctx, conn, session, task)
	return true
}

func (s *Server) waitXiaozhiStreamingASRFinal(session *xiaozhiSession, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		session.mu.Lock()
		hasFinal := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
		closed := session.streamingASRClosed
		session.mu.Unlock()
		if hasFinal {
			return true
		}
		if closed || time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (s *Server) cancelXiaozhiStreamingASR(session *xiaozhiSession, reason string) {
	if session == nil {
		return
	}
	session.mu.Lock()
	stream := session.streamingASRSession
	if stream == nil || session.streamingASRClosed {
		session.mu.Unlock()
		return
	}
	session.streamingASRClosed = true
	session.mu.Unlock()
	stream.Cancel(context.Canceled)
	if strings.TrimSpace(reason) != "" {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "asr.stream.cancelled", s.now().UnixMilli())
	}
}

func (s *Server) handleXiaozhiWS(w http.ResponseWriter, r *http.Request) {
	queryDeviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	headerDeviceID := strings.TrimSpace(r.Header.Get("Device-Id"))
	deviceID := firstNonEmpty(queryDeviceID, headerDeviceID)
	if deviceID != "" && !validA21DeviceID(deviceID) {
		http.Error(w, "invalid device_id", http.StatusBadRequest)
		return
	}
	protocolVersion, err := parseXiaozhiProtocolVersionHeader(r.Header.Get("Protocol-Version"))
	if err != nil {
		http.Error(w, "invalid Protocol-Version", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
	if err != nil {
		return
	}
	s.metrics.wsConnections.WithLabelValues("xiaozhi").Inc()
	defer s.metrics.wsConnections.WithLabelValues("xiaozhi").Dec()
	defer conn.Close(websocket.StatusNormalClosure, "a21 xiaozhi closed")

	ctx := context.Background()
	session := &xiaozhiSession{
		deviceID:              deviceID,
		binaryProtocolVersion: protocolVersion,
	}
	defer func() {
		s.cancelXiaozhiStreamingASR(session, "socket_closed")
		s.cancelXiaozhiOpusIngressQueue(session, "socket_closed")
		id := session.identitySnapshot()
		s.unregisterXiaozhiSocket(id.deviceID, conn)
	}()
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		switch messageType {
		case websocket.MessageText:
			if !s.handleXiaozhiText(ctx, conn, session, data) {
				return
			}
		case websocket.MessageBinary:
			if !s.handleXiaozhiBinary(ctx, conn, session, data) {
				return
			}
		}
	}
}

func parseXiaozhiProtocolVersionHeader(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1, nil
	}
	version, err := strconv.Atoi(raw)
	if err != nil || !xiaozhitransport.SupportedBinaryProtocolVersion(version) {
		return 0, fmt.Errorf("unsupported xiaozhi protocol version")
	}
	return version, nil
}

func (s *Server) handleXiaozhiText(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if xiaozhiTextMessageType(data) == "device" {
		return s.handleXiaozhiDeviceExtension(ctx, conn, session, data)
	}
	frame, err := xiaozhitransport.ParseTextFrame(data, xiaozhitransport.DirectionDeviceToServer, session.identity())
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	session.adoptFrame(frame)
	if frame.Control == nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_message_type", "unsupported xiaozhi message"))
		return true
	}
	switch frame.Control.Type {
	case xiaozhitransport.MessageTypeHello:
		if err := session.configureXiaozhiAudio(frame.Control.Hello.AudioParams); err != nil {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_audio_params", err.Error()))
			return true
		}
		session.mu.Lock()
		session.helloReceived = true
		session.listening = false
		session.listenStartedAtMS = 0
		session.suppressedListenActive = false
		session.ttsStopSent = false
		session.wakePrerollFrames = nil
		session.wakePrerollPayloadBytes = nil
		session.wakePrerollHasSpeech = false
		session.binaryProtocolVersion = frame.Control.Hello.AudioParams.BinaryProtocolVersion
		session.features = frame.Control.Hello.Features
		id := session.identityLocked()
		features := session.features
		session.mu.Unlock()
		s.registerXiaozhiSocket(id.deviceID, conn, &session.writeMu, session, features)
		s.recordXiaozhiDeviceSeen(frame)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.hello.received", s.now().UnixMilli())
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiHelloReply(session))
	case xiaozhitransport.MessageTypeListen:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before listen"))
			return true
		}
		rawListenMode := frame.Control.Listen.Mode
		features := session.featuresSnapshot()
		mode := s.xiaozhiListenMode(rawListenMode, features)
		switch frame.Control.Listen.State {
		case "start":
			nowMS := s.now().UnixMilli()
			if suppressed, reason := session.xiaozhiInputSuppression(nowMS); suppressed {
				session.setXiaozhiListening(false, 0)
				session.mu.Lock()
				session.suppressedListenActive = true
				session.mu.Unlock()
				session.resetXiaozhiOpusIngress()
				session.resetXiaozhiWakePreroll()
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start.input_suppressed", nowMS)
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start.suppressed_"+reason, nowMS)
				s.writeXiaozhiListenReply(ctx, conn, session, "start", "ignored", "")
				return true
			}
			bargeTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn("barge_in", nowMS, xiaozhiPlaybackInterruptWindowMS)
			if shouldStopPlayback {
				s.recordXiaozhiListenBargeInMarkers(session, bargeTask.turn != nil)
				s.writeXiaozhiTTSStopForce(ctx, conn, session, bargeTask.turn, bargeTask, "barge_in")
			}
			turn := session.startXiaozhiTurn(ctx, mode)
			session.setXiaozhiListening(true, nowMS)
			session.resetXiaozhiOpusIngress()
			s.startXiaozhiOpusIngressQueue(ctx, session)
			s.startXiaozhiStreamingASR(ctx, conn, session, mode)
			s.attachXiaozhiWakePreroll(ctx, session)
			session.resetXiaozhiTTSStop()
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.start", s.now().UnixMilli())
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.start", s.now().UnixMilli())
			s.writeXiaozhiOfficialStackChanState(ctx, session, "listening", "listen_start", true)
			if s.xiaozhiStockProfessionalRouteSelected(rawListenMode, features) {
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.professional_route.stock_override", s.now().UnixMilli())
			}
			s.writeXiaozhiListenReply(ctx, conn, session, "start", "accepted", xiaozhiTurnID(turn))
		case "detect":
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.detect", s.now().UnixMilli())
			s.writeXiaozhiListenReply(ctx, conn, session, "detect", "accepted", session.currentXiaozhiTurnID())
		case "stop":
			if !session.xiaozhiListening() {
				if session.clearSuppressedXiaozhiListen() {
					nowMS := s.now().UnixMilli()
					session.suppressXiaozhiInputUntil(nowMS+xiaozhiSuppressedListenDrainMS, "after_suppressed_listen")
					id := session.identitySnapshot()
					s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.suppressed_session_ended", nowMS)
					s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.suppressed_session_drain_armed", nowMS)
				}
				id := session.identitySnapshot()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop.ignored", s.now().UnixMilli())
				s.writeXiaozhiListenReply(ctx, conn, session, "stop", "ignored", "")
				return true
			}
			session.setXiaozhiListening(false, 0)
			if mode == protocol.ModeProfessional {
				session.setCurrentXiaozhiTurnMode(mode)
			}
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.stop", s.now().UnixMilli())
			s.startXiaozhiListenStopAfterIngressDrain(ctx, conn, session, session.currentXiaozhiTurn())
		}
	case xiaozhitransport.MessageTypeAbort:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before abort"))
			return true
		}
		session.setXiaozhiListening(false, 0)
		abortReason := frame.Control.Abort.Reason
		s.cancelXiaozhiStreamingASR(session, "abort")
		abortTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn(abortReason, s.now().UnixMilli(), xiaozhiPlaybackInterruptWindowMS)
		s.recordXiaozhiAbortMarkers(session, abortReason, shouldStopPlayback)
		if strings.TrimSpace(abortReason) == "" {
			session.suppressXiaozhiInputUntil(s.now().UnixMilli()+xiaozhiTouchBargeInInputCooldownMS, "after_barge")
		}
		if shouldStopPlayback {
			s.writeXiaozhiTTSStopForce(ctx, conn, session, abortTask.turn, abortTask, "abort")
		}
	case xiaozhitransport.MessageTypeMCP:
		if !session.helloReceivedSnapshot() {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before mcp"))
			return true
		}
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.mcp.response.received", s.now().UnixMilli())
		s.recordXiaozhiDeviceActivity(session, "xiaozhi.mcp.response.received", map[string]string{
			"xiaozhi_mcp_response": "received_redacted",
		})
	}
	return true
}

func (s *Server) writeXiaozhiListenReply(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, state string, status string, turnID string) {
	if !session.shouldSendXiaozhiListenReply() {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen."+state+".reply_suppressed_stock_physical", s.now().UnixMilli())
		return
	}
	_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiBaseReply(session, "listen", state, status, turnID))
}

func (session *xiaozhiSession) shouldSendXiaozhiListenReply() bool {
	if session == nil {
		return false
	}
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	if xiaozhiClientProfile(features) == "debug" {
		return true
	}
	return !hardwareMACDeviceID(id.deviceID)
}

func (s *Server) handleXiaozhiDeviceExtension(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if !session.helloReceivedSnapshot() {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before device events"))
		return true
	}
	features := session.featuresSnapshot()
	productPlaybackEvents := s.xiaozhiProductPlaybackEventsAllowed(session)
	productKeepaliveEvents := s.xiaozhiProductKeepaliveEventsAllowed(session)
	productTouchEvents := s.xiaozhiProductTouchEventsAllowed(session)
	if !features.DeviceEvents && !productPlaybackEvents && !productKeepaliveEvents && !productTouchEvents {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "device_events_disabled", "device events require debug profile negotiation or product allowance"))
		return true
	}
	event, err := xiaozhitransport.ParseDeviceExtensionEvent(data)
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	if !features.DeviceEvents {
		switch event.Kind {
		case xiaozhitransport.DeviceEventKindPlayback:
			if !productPlaybackEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product keepalive allowance does not accept playback acknowledgements"))
				return true
			}
		case xiaozhitransport.DeviceEventKindHeartbeat:
			if !productKeepaliveEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product playback allowance does not accept keepalive heartbeats"))
				return true
			}
		case xiaozhitransport.DeviceEventKindTouch:
			if !productTouchEvents {
				_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product allowance does not accept touch events"))
				return true
			}
		default:
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "product allowance only accepts playback acknowledgements, keepalive heartbeats, and touch events"))
			return true
		}
	}
	switch event.Kind {
	case xiaozhitransport.DeviceEventKindPlayback:
		switch event.Value {
		case "start":
			s.recordXiaozhiPlaybackStart(session, event.StreamID)
			return true
		case "stop_done":
			s.recordXiaozhiPlaybackStopDone(session, event.StreamID)
			return true
		default:
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi device event"))
			return true
		}
	case xiaozhitransport.DeviceEventKindHeartbeat:
		s.recordXiaozhiHeartbeat(session, event)
		return true
	case xiaozhitransport.DeviceEventKindTouch:
		if !s.recordXiaozhiTouchEvent(session, event) {
			_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi touch event"))
			return true
		}
		if xiaozhiTouchEventIsBargeIn(event) {
			nowMS := s.now().UnixMilli()
			bargeTask, shouldStopPlayback := session.prepareXiaozhiListenStartBargeIn("touch_barge_in", nowMS, xiaozhiPlaybackInterruptWindowMS)
			if shouldStopPlayback {
				s.recordXiaozhiTouchBargeInMarkers(session, bargeTask.turn != nil)
				s.writeXiaozhiTTSStopForce(ctx, conn, session, bargeTask.turn, bargeTask, "touch_barge_in")
			}
		}
		s.maybeSendXiaozhiTouchReaction(ctx, session, event)
		return true
	default:
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "unsupported_device_event", "unsupported xiaozhi device event"))
		return true
	}
}

func xiaozhiTextMessageType(data []byte) string {
	var common struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &common); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(common.Type))
}
