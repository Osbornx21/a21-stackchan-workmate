package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
)

func (s *Server) maybeAutoStopXiaozhiTurnOnIngress(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, events []audio.Event) {
	autoStopReason := ""
	if containsAudioIngressEvent(events, audio.EventVADSpeechEnd) {
		autoStopReason = "speech_end"
	} else if s.xiaozhiListenMaxDurationReached(session, s.now().UnixMilli()) {
		autoStopReason = "max_duration"
	}
	if autoStopReason == "" {
		return
	}
	session.mu.Lock()
	if !session.voicePipelineHasSpeech || !session.listening {
		session.mu.Unlock()
		return
	}
	stockPhysicalStop := autoStopReason == "speech_end" && hardwareMACDeviceID(session.deviceID) && xiaozhiClientProfile(session.features) == "stock"
	turn := session.currentTurn
	id := session.identityLocked()
	if stockPhysicalStop {
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.gateway_vad_stop_deferred_stock_physical", s.now().UnixMilli())
		return
	}
	if turn == nil {
		session.mu.Unlock()
		return
	}
	session.listening = false
	session.listenStartedAtMS = 0
	session.mu.Unlock()
	if autoStopReason == "max_duration" {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.max_duration_auto_stop", s.now().UnixMilli())
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.auto_stop", s.now().UnixMilli())
	if s.startXiaozhiStreamingASRCommit(ctx, conn, session, turn) {
		return
	}
	task := s.newXiaozhiTurnTask(session, turn)
	s.startXiaozhiTurnTask(ctx, conn, session, task)
}

func (s *Server) xiaozhiListenMaxDurationReached(session *xiaozhiSession, nowMS int64) bool {
	if s == nil || session == nil || s.xiaozhiListenMaxDurationMS <= 0 || nowMS <= 0 {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.listening || !session.voicePipelineHasSpeech || session.listenStartedAtMS <= 0 {
		return false
	}
	return nowMS-session.listenStartedAtMS >= s.xiaozhiListenMaxDurationMS
}

func (session *xiaozhiSession) xiaozhiUsesStockPhysicalStop() bool {
	if session == nil {
		return false
	}
	id := session.identitySnapshot()
	features := session.featuresSnapshot()
	return hardwareMACDeviceID(id.deviceID) && xiaozhiClientProfile(features) == "stock"
}

func (s *Server) recordXiaozhiAbortMarkers(session *xiaozhiSession, reason string, hadTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.abort.received", now)
	if !hadTurn {
		return
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	if !xiaozhiAbortIsBargeIn(reason) {
		return
	}
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in_detected", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) recordXiaozhiListenBargeInMarkers(session *xiaozhiSession, hadActiveTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.listen.barge_in", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	if hadActiveTurn {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	} else {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop.recent_downlink", now)
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) recordXiaozhiTouchBargeInMarkers(session *xiaozhiSession, hadActiveTurn bool) {
	now := s.now().UnixMilli()
	id := session.identitySnapshot()
	s.metrics.bargeInTotal.Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch.barge_in", now)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "barge_in.detected", now)
	if hadActiveTurn {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.start", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "provider.cancel.end", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.turn.cancel", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "turn_cancelled", now)
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "downlink_queue_cleared", now)
	} else {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop.recent_downlink", now)
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "playback.stop", now)
}

func (s *Server) writeXiaozhiTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	if task.mode == protocol.ModeProfessional {
		s.writeXiaozhiProfessionalTTS(ctx, conn, session, task)
		return
	}
	if professionalVoiceTriggerModeAllowed(task.mode) && professionalVoiceTrigger(task.streamingASRFinalText) {
		task.mode = protocol.ModeProfessional
		session.setCurrentXiaozhiTurnMode(protocol.ModeProfessional)
		now := s.now().UnixMilli()
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.voice_trigger.detected", now)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_route.voice_trigger", now)
		s.writeXiaozhiProfessionalTTS(ctx, conn, session, task)
		return
	}
	if s.writeXiaozhiVoicePipelineTTS(ctx, conn, session, task) {
		return
	}
	if session.shouldAbortXiaozhiTurn(task.turn) {
		return
	}
	s.writeXiaozhiPlaceholderTTS(ctx, conn, session, task)
}

func (s *Server) writeXiaozhiProfessionalTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "professional_tts_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":          "tts",
		"state":         "start",
		"mode":          string(protocol.ModeProfessional),
		"turn_id":       task.turnID,
		"trace_id":      task.traceID,
		"session_id":    task.sessionID,
		"device_id":     task.deviceID,
		"audio_ingress": task.audioIngressSummary("professional_boundary", "checking_then_evidence"),
	}); err != nil {
		return
	}
	receipt := xiaozhiProfessionalCheckingReceipt(task)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":         "tts",
		"state":        "sentence_start",
		"phase":        "professional_checking",
		"mode":         string(protocol.ModeProfessional),
		"turn_id":      task.turnID,
		"trace_id":     task.traceID,
		"session_id":   task.sessionID,
		"device_id":    task.deviceID,
		"text":         receipt.Text,
		"professional": receipt,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_checking_write_error")
		return
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.checking_feedback.sent", s.now().UnixMilli())
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, receipt.Text, "xiaozhi.professional_checking")
	if session.shouldAbortXiaozhiTurn(turn) {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	utterance, err := s.xiaozhiProfessionalASRFinal(turn.ctx, task)
	if err != nil {
		if errors.Is(err, context.Canceled) || session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_asr_unavailable", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_asr_unavailable")
		return
	}
	if strings.TrimSpace(utterance) == "" {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_asr_empty", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_asr_empty")
		return
	}
	workspace := s.selectedProfessionalWorkspace()
	request := v21adapter.QueryRequest{
		TraceID:            task.traceID,
		SessionID:          task.sessionID,
		DeviceID:           task.deviceID,
		UserID:             workspace.UserID,
		WorkspaceID:        workspace.WorkspaceID,
		Mode:               "professional",
		QueryScope:         workspace.QueryScope,
		Utterance:          utterance,
		LatencyProfile:     "fast_first",
		AnswerStyle:        "voice_first_with_citations",
		MaxFirstResponseMS: v21adapter.ProfessionalMaxFirstResponseMS,
		PrivacyScope:       "professional_only",
	}
	utteranceBucket := v21UtteranceLengthBucket(utterance)
	readRecordID := s.startProfessionalReadRecord(request, utteranceBucket)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.workspace.ready", s.now().UnixMilli())
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "professional.query_scope."+workspace.QueryScope, s.now().UnixMilli())
	bindingDecision := s.professionalDeviceBindingDecision(task.deviceID, workspace.UserID, workspace.WorkspaceID, workspace.QueryScope)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, bindingDecision.TraceMarker, s.now().UnixMilli())
	if !bindingDecision.Allowed {
		s.failProfessionalReadRecord(readRecordID, bindingDecision.FailureCode)
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_device_binding_blocked")
		return
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.utterance."+utteranceBucket, s.now().UnixMilli())
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.start", s.now().UnixMilli())
	queryCtx, cancel := context.WithTimeout(turn.ctx, s.v21TTL)
	defer cancel()
	started := time.Now()
	response, err := s.v21.Query(queryCtx, request)
	s.metrics.v21QueryMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		if errors.Is(err, context.Canceled) || session.shouldAbortXiaozhiTurn(turn) {
			s.failProfessionalReadRecord(readRecordID, "suppressed")
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
			return
		}
		s.failProfessionalReadRecord(readRecordID, professionalReadFailureCode(err, queryCtx))
		s.recordV21QueryFailure(task.traceID, task.sessionID, task.deviceID, err, queryCtx)
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_v21_unavailable")
		return
	}
	if session.shouldAbortXiaozhiTurn(turn) {
		s.failProfessionalReadRecord(readRecordID, "suppressed")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	report, err := v21adapter.NewProfessionalBridgeEvidenceReport(response)
	if err != nil {
		s.failProfessionalReadRecord(readRecordID, "contract_invalid")
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.error", s.now().UnixMilli())
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.error.contract_invalid", s.now().UnixMilli())
		s.writeXiaozhiProfessionalFallback(ctx, conn, session, task, "professional_contract_invalid")
		return
	}
	s.completeProfessionalReadRecord(readRecordID, response)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "v21.query.first_result", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":         "tts",
		"state":        "sentence_start",
		"phase":        "professional_result",
		"mode":         string(protocol.ModeProfessional),
		"turn_id":      task.turnID,
		"trace_id":     task.traceID,
		"session_id":   task.sessionID,
		"device_id":    task.deviceID,
		"text":         response.FastAnswer,
		"confidence":   response.Confidence,
		"professional": report,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_result_write_error")
		return
	}
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, response.FastAnswer, "xiaozhi.professional_result")
	if session.shouldAbortXiaozhiTurn(turn) {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.professional_result_suppressed", s.now().UnixMilli())
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "professional_result_completed")
}

func xiaozhiProfessionalCheckingReceipt(task xiaozhiTurnTask) v21adapter.ProfessionalBridgeReceipt {
	return v21adapter.ProfessionalBridgeReceipt{
		SchemaVersion:      "a21.v21_professional_bridge_receipt.v1",
		TraceID:            task.traceID,
		SessionID:          task.sessionID,
		Mode:               "professional",
		Status:             "checking",
		Text:               v21adapter.ProfessionalCheckingFeedbackText,
		MaxFirstResponseMS: v21adapter.ProfessionalMaxFirstResponseMS,
		EvidenceCompleted:  false,
	}
}

func (s *Server) xiaozhiProfessionalASRFinal(ctx context.Context, task xiaozhiTurnTask) (string, error) {
	if finalText := strings.TrimSpace(task.streamingASRFinalText); finalText != "" {
		return finalText, nil
	}
	if s.xiaozhiProfessionalASR == nil {
		return "", fmt.Errorf("professional ASR adapter unavailable")
	}
	events, err := s.xiaozhiProfessionalASR.Transcribe(ctx, providers.ASRAdapterRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode:   string(protocol.ModeProfessional),
		Frames: task.voicePipelineFrames,
	})
	if err != nil {
		return "", err
	}
	finalText := ""
	for event := range events {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Final {
			finalText = strings.TrimSpace(event.Text)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return finalText, nil
}

func (s *Server) writeXiaozhiProfessionalFallback(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, reason string) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	text := "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。"
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "professional_unavailable",
		"mode":       string(protocol.ModeProfessional),
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       text,
	}); err != nil {
		session.cancelXiaozhiTurnContext(turn, "professional_fallback_write_error")
		return
	}
	s.writeXiaozhiProfessionalAudioDownlink(ctx, conn, session, task, text, "xiaozhi.professional_fallback")
	if session.shouldAbortXiaozhiTurn(turn) {
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, reason)
}

func (s *Server) writeXiaozhiProfessionalAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, text string, marker string) bool {
	chunks, ok := s.writeXiaozhiTextAudioDownlink(ctx, conn, session, task, text, protocol.ModeProfessional, marker)
	return ok && chunks > 0
}

func (s *Server) writeXiaozhiTextAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, text string, mode protocol.Mode, marker string) (int, bool) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return 0, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	tts := s.xiaozhiFastAckTTS
	if tts == nil {
		tts = providers.NewMockTTSAdapter("mock-fast-tts")
	}
	chunks, err := tts.Synthesize(turn.ctx, providers.TTSAdapterRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode: string(mode),
		Text: text,
	})
	if err != nil {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	firstAudio := true
	wrote := false
	audioChunks := 0
	for chunk := range chunks {
		if session.shouldAbortXiaozhiTurn(turn) {
			return audioChunks, false
		}
		if firstAudio {
			firstAudio = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "tts.first_audio", s.now().UnixMilli())
		}
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_error", s.now().UnixMilli())
			session.cancelXiaozhiTurnContext(turn, marker+"_downlink_error")
			return audioChunks, false
		}
		if !ok {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_aborted", s.now().UnixMilli())
			return audioChunks, false
		}
		audioChunks++
		if !wrote {
			wrote = true
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink", s.now().UnixMilli())
		}
	}
	if !wrote {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
	}
	return audioChunks, wrote
}

func (s *Server) writeXiaozhiWAVAudioDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask, chunks []providers.VoiceAudioChunk, marker string) (int, bool) {
	turn := task.turn
	if session.shouldAbortXiaozhiTurn(turn) {
		return 0, false
	}
	if len(chunks) == 0 {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".tts_unavailable", s.now().UnixMilli())
		return 0, false
	}
	firstAudio := true
	wrote := false
	audioChunks := 0
	for _, chunk := range chunks {
		if session.shouldAbortXiaozhiTurn(turn) {
			return audioChunks, false
		}
		if firstAudio {
			firstAudio = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "tts.first_audio", s.now().UnixMilli())
		}
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_error", s.now().UnixMilli())
			session.cancelXiaozhiTurnContext(turn, marker+"_downlink_error")
			return audioChunks, false
		}
		if !ok {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink_aborted", s.now().UnixMilli())
			return audioChunks, false
		}
		audioChunks++
		if !wrote {
			wrote = true
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, marker+".downlink", s.now().UnixMilli())
		}
	}
	return audioChunks, wrote
}

func (s *Server) writeXiaozhiVoicePipelineTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) bool {
	turn := task.turn
	hasStreamingFinal := strings.TrimSpace(task.streamingASRFinalText) != ""
	if session.shouldAbortXiaozhiTurn(turn) || len(task.voicePipelineFrames) == 0 || (!task.voicePipelineHasSpeech && !hasStreamingFinal) {
		return false
	}
	startAtMS := s.now().UnixMilli()
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.start", startAtMS)
	newRunner := s.currentXiaozhiVoicePipelineRunnerFactory()
	if newRunner == nil {
		newRunner = defaultXiaozhiVoicePipelineRunner
	}
	runner := newRunner()
	asrTranscript := task.streamingASRFinalText
	asrTranscriptSource := ""
	if task.streamingASRPartialDriven && strings.TrimSpace(task.streamingASRPartialText) != "" {
		asrTranscript = task.streamingASRPartialText
		asrTranscriptSource = providers.VoicePipelineASRTranscriptSourcePartial
	} else if strings.TrimSpace(asrTranscript) != "" {
		asrTranscriptSource = providers.VoicePipelineASRTranscriptSourceFinal
	}
	if strings.TrimSpace(asrTranscript) != "" {
		if !s.writeXiaozhiSTT(ctx, conn, session, turn, task, asrTranscript, asrTranscriptSource) {
			return true
		}
	}
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "voice_pipeline_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":           "tts",
		"state":          "start",
		"turn_id":        task.turnID,
		"trace_id":       task.traceID,
		"session_id":     task.sessionID,
		"device_id":      task.deviceID,
		"audio_ingress":  task.audioIngressSummary("pipeline_running", s.xiaozhiVoicePipelineInitialTTSStatus()),
		"voice_pipeline": s.xiaozhiVoicePipelineInitialSummary(),
	}); err != nil {
		return true
	}
	request := providers.VoicePipelineRequest{
		Session: providers.VoiceSession{
			TraceID:   task.traceID,
			SessionID: task.sessionID,
			DeviceID:  task.deviceID,
		},
		Mode:                string(protocol.ModeWorkmate),
		Frames:              append([]providers.VoicePipelinePCMFrame(nil), task.voicePipelineFrames...),
		VoiceCloneProfile:   s.currentVoiceCloneProfile(),
		ASRTranscript:       asrTranscript,
		ASRTranscriptSource: asrTranscriptSource,
	}
	if s.selectedVoiceMode() == VoiceModeRoleplay {
		if prompt, err := s.roleplayPromptInput(RoleplayProfileSelectionRequest{}, roleplayPromptUserText(asrTranscript)); err == nil && strings.TrimSpace(prompt) != "" {
			request.TextPrompt = prompt
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "roleplay.prompt_input.used", s.now().UnixMilli())
		}
		if voiceCloneProfileUsesClone(request.VoiceCloneProfile) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "roleplay.voice_clone_profile.used", s.now().UnixMilli())
		}
	}
	markAnswerReady := s.startXiaozhiFastAckBackchannel(ctx, conn, session, turn, task)
	defer markAnswerReady()
	if streamer, ok := runner.(xiaozhiVoicePipelineStreamer); ok {
		return s.writeXiaozhiStreamingVoicePipelineAnswer(ctx, conn, session, turn, task, streamer, request, startAtMS, markAnswerReady)
	}
	result, err := runner.Run(turn.ctx, request)
	s.recordVoicePipelineFallback(task.traceID, task.sessionID, task.deviceID, result.Report)
	if err != nil || result.Status != providers.VoicePipelineStatusCompleted || len(result.AudioChunks) == 0 {
		if session.shouldAbortXiaozhiTurn(turn) || result.Status == providers.VoicePipelineStatusCancelled {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		markAnswerReady()
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, result.Report)
		s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, result.Report, result, err)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, result.Timing)
	markAnswerReady()
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":           "tts",
		"state":          "sentence_start",
		"phase":          "answer",
		"turn_id":        task.turnID,
		"trace_id":       task.traceID,
		"session_id":     task.sessionID,
		"device_id":      task.deviceID,
		"voice_pipeline": xiaozhiVoicePipelineSummary(result.Report),
		"text":           "",
	}); err != nil {
		return true
	}
	firstDownlink := true
	for _, chunk := range result.AudioChunks {
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.downlink_error", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_downlink_error")
			return true
		}
		if !ok {
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_aborted")
			return true
		}
		if firstDownlink {
			firstDownlink = false
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
		}
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed", s.now().UnixMilli())
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed")
	return true
}

func (s *Server) writeXiaozhiSTT(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, text string, source string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	displayText, displayPolicy, sendDisplay := xiaozhiSTTDisplayText(text, s.xiaozhiSTTScreenPolicy)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt.display."+displayPolicy, s.now().UnixMilli())
	if !sendDisplay {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt.suppressed", s.now().UnixMilli())
		if source != "" {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt."+safeGatewayFallbackToken(source, "streaming"), s.now().UnixMilli())
		}
		return true
	}
	payload := map[string]any{
		"type":       "stt",
		"session_id": task.sessionID,
		"text":       displayText,
	}
	if err := session.writeXiaozhiJSON(ctx, conn, turn, payload); err != nil {
		session.cancelXiaozhiTurnContext(turn, "stt_write_error")
		return false
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt.sent", s.now().UnixMilli())
	if source != "" {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.stt."+safeGatewayFallbackToken(source, "streaming"), s.now().UnixMilli())
	}
	return true
}

func xiaozhiSTTDisplayText(text string, policy string) (string, string, bool) {
	switch normalizeXiaozhiSTTScreenPolicy(policy) {
	case xiaozhiSTTScreenPolicyStatusOnly:
		return xiaozhiSTTStatusOnlyText, xiaozhiSTTScreenPolicyStatusOnly, true
	case xiaozhiSTTScreenPolicyOff:
		return "", xiaozhiSTTScreenPolicyOff, false
	default:
		return text, xiaozhiSTTScreenPolicyRaw, true
	}
}
