package gateway

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
)

func (s *Server) writeXiaozhiStreamingVoicePipelineAnswer(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, runner xiaozhiVoicePipelineStreamer, req providers.VoicePipelineRequest, startAtMS int64, markAnswerReady func()) bool {
	events, err := runner.RunStream(turn.ctx, req)
	if err != nil {
		markAnswerReady()
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	answerStarted := false
	lastSegmentSeq := 0
	firstDownlink := true
	stageMarkersRecorded := false
	profileMarkersRecorded := false
	var finalResult providers.VoicePipelineResult
	for event := range events {
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		switch event.Kind {
		case providers.VoicePipelineStreamAudioChunk:
			if !answerStarted || event.SegmentSeq != lastSegmentSeq {
				markAnswerReady()
				answerStarted = true
				lastSegmentSeq = event.SegmentSeq
				if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
					"type":           "tts",
					"state":          "sentence_start",
					"phase":          "answer",
					"turn_id":        task.turnID,
					"trace_id":       task.traceID,
					"session_id":     task.sessionID,
					"device_id":      task.deviceID,
					"voice_pipeline": s.xiaozhiVoicePipelineStreamingSummary(event.Report),
					"text":           "",
				}); err != nil {
					session.cancelXiaozhiTurnContext(turn, "voice_pipeline_answer_write_error")
					return true
				}
			}
			if !stageMarkersRecorded {
				stageMarkersRecorded = true
				s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, event.Timing)
			}
			if !profileMarkersRecorded {
				profileMarkersRecorded = true
				s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, event.Report)
			}
			ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, event.AudioChunk)
			if err != nil {
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.downlink_error", s.now().UnixMilli())
				s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_downlink_error")
				session.cancelXiaozhiTurnContext(turn, "voice_pipeline_downlink_error")
				return true
			}
			if !ok {
				s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_aborted")
				session.cancelXiaozhiTurnContext(turn, "voice_pipeline_aborted")
				return true
			}
			if firstDownlink {
				firstDownlink = false
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
				s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.answer.downlink", s.now().UnixMilli())
			}
		case providers.VoicePipelineStreamDone:
			finalResult = event.Result
			err = event.Err
		}
	}
	s.recordVoicePipelineFallback(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
	if err != nil || finalResult.Status != providers.VoicePipelineStatusCompleted || len(finalResult.AudioChunks) == 0 {
		if session.shouldAbortXiaozhiTurn(turn) || finalResult.Status == providers.VoicePipelineStatusCancelled {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.cancelled", s.now().UnixMilli())
			return true
		}
		if answerStarted {
			markAnswerReady()
			s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
			s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report, finalResult, err)
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed_degraded_after_audio", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed_degraded")
			return true
		}
		markAnswerReady()
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
		s.recordXiaozhiVoicePipelineFailureMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report, finalResult, err)
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.unavailable", s.now().UnixMilli())
		s.writeXiaozhiLocalFallback(ctx, conn, session, turn, task, "voice_pipeline_unavailable_after_fast_ack")
		return true
	}
	if !stageMarkersRecorded {
		s.recordXiaozhiVoicePipelineStageMarkers(task.traceID, task.sessionID, task.deviceID, startAtMS, finalResult.Timing)
	}
	if !profileMarkersRecorded {
		s.recordXiaozhiVoicePipelineProfileMarkers(task.traceID, task.sessionID, task.deviceID, finalResult.Report)
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.voice_pipeline.completed", s.now().UnixMilli())
	s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "voice_pipeline_answer_completed")
	return true
}

func (s *Server) startXiaozhiFastAckBackchannel(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask) func() {
	if !s.xiaozhiFastAckEnabled {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.disabled", s.now().UnixMilli())
		return func() {}
	}
	delay := s.xiaozhiFastAckDelay
	if delay <= 0 {
		s.writeXiaozhiFastAckDownlink(ctx, conn, session, turn, task)
		return func() {}
	}
	answerReady := make(chan struct{})
	var answerReadyOnce sync.Once
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-turn.ctx.Done():
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.cancelled_before_delay", s.now().UnixMilli())
			return
		case <-answerReady:
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.skipped_answer_ready", s.now().UnixMilli())
			return
		case <-timer.C:
		}
		if session.shouldAbortXiaozhiTurn(turn) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.cancelled_after_delay", s.now().UnixMilli())
			return
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.delay_elapsed", s.now().UnixMilli())
		if s.writeXiaozhiFastAckDownlink(ctx, conn, session, turn, task) {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.delayed", s.now().UnixMilli())
		}
	}()
	return func() {
		answerReadyOnce.Do(func() {
			close(answerReady)
		})
	}
}

func (s *Server) writeXiaozhiFastAckDownlink(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask) bool {
	if session.shouldAbortXiaozhiTurn(turn) {
		return false
	}
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "fast_ack",
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       "",
	}); err != nil {
		return false
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
		Mode: string(protocol.ModeWorkmate),
		Text: "我在",
	})
	if err != nil {
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.unavailable", s.now().UnixMilli())
		return false
	}
	for chunk := range chunks {
		ok, err := s.writeXiaozhiOpusDownlink(ctx, conn, session, turn, chunk)
		if err != nil {
			s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.downlink_error", s.now().UnixMilli())
			s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "fast_ack_downlink_error")
			return false
		}
		if !ok {
			return false
		}
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "audio.downlink.first_frame", s.now().UnixMilli())
		s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.downlink", s.now().UnixMilli())
		return true
	}
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.fast_ack.unavailable", s.now().UnixMilli())
	return false
}

func (s *Server) xiaozhiVoicePipelineInitialTTSStatus() string {
	if s.xiaozhiFastAckEnabled && s.xiaozhiFastAckDelay > 0 {
		return "delayed_fast_ack_then_answer"
	}
	if s.xiaozhiFastAckEnabled {
		return "fast_ack_then_answer"
	}
	return "answer_only"
}

func (s *Server) xiaozhiVoicePipelineInitialSummary() map[string]any {
	summary := s.xiaozhiVoicePipelineFastAckSummary()
	summary["fast_ack_enabled"] = s.xiaozhiFastAckEnabled
	if s.xiaozhiFastAckEnabled && s.xiaozhiFastAckDelay > 0 {
		summary["schema_version"] = "a21.voice_pipeline.delayed_fast_ack.v1"
		summary["stage"] = "answer_pending"
		summary["fast_ack_delay_ms"] = s.xiaozhiFastAckDelay.Milliseconds()
	} else if !s.xiaozhiFastAckEnabled {
		summary["schema_version"] = "a21.voice_pipeline.answer_only.v1"
		summary["stage"] = "answer_pending"
	}
	return summary
}

func (s *Server) xiaozhiVoicePipelineFastAckSummary() map[string]any {
	meta := s.currentXiaozhiVoicePipelineMeta()
	if isZeroGatewayVoicePipelineSelection(meta.Selection) {
		meta.Selection = providers.VoicePipelineSelectionFromEnv(nil)
	}
	executionMode := strings.TrimSpace(meta.ExecutionMode)
	if executionMode == "" {
		executionMode = "fixture"
	}
	return map[string]any{
		"schema_version": "a21.voice_pipeline.fast_ack.v1",
		"status":         "running",
		"stage":          "fast_ack",
		"execution_mode": executionMode,
		"selection": map[string]any{
			"asr_mode":        meta.Selection.ASRMode,
			"asr_profile":     meta.Selection.ASRProfile,
			"asr_profile_env": meta.Selection.ASRProfileEnv,
			"llm_profile":     meta.Selection.LLMProfile,
			"llm_profile_env": meta.Selection.LLMProfileEnv,
			"tts_mode":        meta.Selection.TTSMode,
			"tts_profile":     meta.Selection.TTSProfile,
			"tts_profile_env": meta.Selection.TTSProfileEnv,
		},
	}
}

func (s *Server) xiaozhiVoicePipelineStreamingSummary(reports ...providers.VoicePipelineReport) map[string]any {
	summary := s.xiaozhiVoicePipelineFastAckSummary()
	summary["schema_version"] = "a21.voice_pipeline.streaming_answer.v1"
	summary["stage"] = "answer"
	summary["streaming"] = true
	if len(reports) == 0 {
		return summary
	}
	report := reports[0]
	if report.Fallback != nil && report.Fallback.Activated {
		summary["fallback"] = map[string]any{
			"activated": true,
			"provider":  safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
			"reason":    safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated"),
		}
		summary["selection"] = map[string]any{
			"asr_mode":                 report.Selection.ASRMode,
			"asr_profile":              report.Selection.ASRProfile,
			"asr_profile_env":          report.Selection.ASRProfileEnv,
			"llm_profile":              report.Selection.LLMProfile,
			"llm_profile_env":          report.Selection.LLMProfileEnv,
			"llm_fallback_profile":     report.Selection.LLMFallbackProfile,
			"llm_fallback_profile_env": report.Selection.LLMFallbackProfileEnv,
			"tts_mode":                 report.Selection.TTSMode,
			"tts_profile":              report.Selection.TTSProfile,
			"tts_profile_env":          report.Selection.TTSProfileEnv,
		}
	}
	return summary
}

func (s *Server) recordXiaozhiVoicePipelineStageMarkers(traceID string, sessionID string, deviceID string, startAtMS int64, timing providers.VoicePipelineTiming) {
	if timing.ASRFirstPartialMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "asr.first_partial", startAtMS+timing.ASRFirstPartialMS)
	}
	if timing.ASRFinalMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "asr.final", startAtMS+timing.ASRFinalMS)
	}
	if timing.LLMFirstContentMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "provider.first_content", startAtMS+timing.LLMFirstContentMS)
	}
	if timing.TTSFirstAudioMS >= 0 {
		s.recordTrace(traceID, sessionID, deviceID, "tts.first_audio", startAtMS+timing.TTSFirstAudioMS)
	}
}

func (s *Server) recordXiaozhiVoicePipelineProfileMarkers(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport) {
	now := s.now().UnixMilli()
	for _, marker := range xiaozhiVoicePipelineProfileMarkers(report) {
		s.recordTrace(traceID, sessionID, deviceID, marker, now)
	}
}

func (s *Server) recordXiaozhiVoicePipelineFailureMarkers(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport, result providers.VoicePipelineResult, err error) {
	now := s.now().UnixMilli()
	markers := xiaozhiVoicePipelineFailureMarkers(report, result, err)
	for _, marker := range markers {
		s.recordTrace(traceID, sessionID, deviceID, marker, now)
	}
}

func xiaozhiVoicePipelineFailureMarkers(report providers.VoicePipelineReport, result providers.VoicePipelineResult, err error) []string {
	markers := make([]string, 0, 4)
	if report.Status != "" && report.Status != string(providers.VoicePipelineStatusCompleted) {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.status_"+safeGatewayFallbackToken(report.Status, "failed"))
	}
	if result.Status != "" && result.Status != providers.VoicePipelineStatusCompleted {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.result_"+safeGatewayFallbackToken(string(result.Status), "failed"))
	}
	for _, finding := range report.Findings {
		marker := xiaozhiVoicePipelineFindingMarker(finding)
		if marker != "" && !gatewayStringSliceHas(markers, marker) {
			markers = append(markers, marker)
		}
	}
	if result.Timing.LLMFirstContentMS < 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.no_llm_first_content")
	}
	if result.Timing.TTSFirstAudioMS < 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.no_tts_first_audio")
	}
	if len(result.AudioChunks) == 0 {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.empty_audio")
	}
	if err != nil {
		markers = append(markers, "xiaozhi.voice_pipeline.failed.err")
	}
	return markers
}

func xiaozhiVoicePipelineFindingMarker(finding string) string {
	finding = strings.ToLower(strings.TrimSpace(finding))
	switch finding {
	case "asr adapter failed":
		return "xiaozhi.voice_pipeline.failed.asr_adapter_failed"
	case "text stream adapter failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_adapter_failed"
	case "text stream adapter http status was not successful":
		return "xiaozhi.voice_pipeline.failed.text_stream_http_status"
	case "text stream adapter parse failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_parse_failed"
	case "text stream adapter read failed":
		return "xiaozhi.voice_pipeline.failed.text_stream_read_failed"
	case "tts adapter failed":
		return "xiaozhi.voice_pipeline.failed.tts_adapter_failed"
	case "tts adapter no audio":
		return "xiaozhi.voice_pipeline.failed.tts_no_audio"
	case "tts adapter read failed":
		return "xiaozhi.voice_pipeline.failed.tts_read_failed"
	case "tts adapter provider error":
		return "xiaozhi.voice_pipeline.failed.tts_provider_error"
	case "tts adapter invalid audio delta":
		return "xiaozhi.voice_pipeline.failed.tts_invalid_audio_delta"
	case "tts adapter missing configuration":
		return "xiaozhi.voice_pipeline.failed.tts_missing_configuration"
	case "tts adapter connect failed":
		return "xiaozhi.voice_pipeline.failed.tts_connect_failed"
	case "tts adapter session update failed":
		return "xiaozhi.voice_pipeline.failed.tts_session_update_failed"
	case "tts adapter text append failed":
		return "xiaozhi.voice_pipeline.failed.tts_text_append_failed"
	case "tts adapter text commit failed":
		return "xiaozhi.voice_pipeline.failed.tts_text_commit_failed"
	case "tts adapter session finish failed":
		return "xiaozhi.voice_pipeline.failed.tts_session_finish_failed"
	case "provider_fallback_used":
		return "xiaozhi.voice_pipeline.provider_fallback_used"
	case "streaming_asr_partial_reused":
		return "xiaozhi.voice_pipeline.streaming_asr_partial_reused"
	case "streaming_asr_final_reused":
		return "xiaozhi.voice_pipeline.streaming_asr_final_reused"
	default:
		return ""
	}
}

func xiaozhiVoicePipelineProfileMarkers(report providers.VoicePipelineReport) []string {
	markers := make([]string, 0, 3)
	if xiaozhiVoicePipelineASRRealStreaming(report) {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.real_streaming")
	} else if strings.Contains(report.Selection.ASRProfile, "mock") {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.mock_blocked")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.asr.batch_blocked")
	}
	if xiaozhiVoicePipelineLLMRealStreaming(report.Selection.LLMProfile) {
		markers = append(markers, "xiaozhi.voice_pipeline.llm.real_streaming")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.llm.mock_blocked")
	}
	if xiaozhiVoicePipelineTTSRealStreaming(report.Selection.TTSProfile) {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.real_streaming")
	} else if strings.Contains(report.Selection.TTSProfile, "mock") {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.mock_blocked")
	} else {
		markers = append(markers, "xiaozhi.voice_pipeline.tts.file_boundary_blocked")
	}
	return markers
}

func xiaozhiVoicePipelineASRRealStreaming(report providers.VoicePipelineReport) bool {
	profile := strings.ToLower(strings.TrimSpace(report.Selection.ASRProfile))
	return (profile == "sherpa_onnx_streaming" ||
		profile == "local_sherpa_onnx_streaming" ||
		profile == "streaming_zipformer" ||
		profile == "dashscope_qwen_asr_realtime" ||
		profile == "doubao_asr_realtime" ||
		profile == "qwen_asr_realtime") &&
		(gatewayStringSliceHas(report.Findings, "streaming_asr_partial_reused") || gatewayStringSliceHas(report.Findings, "streaming_asr_final_reused"))
}

func xiaozhiVoicePipelineLLMRealStreaming(profile string) bool {
	profile = strings.ToLower(strings.TrimSpace(profile))
	return profile != "" && !strings.Contains(profile, "mock")
}

func xiaozhiVoicePipelineTTSRealStreaming(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "doubao_tts_realtime", "doubao_realtime_tts", "dashscope_qwen_tts_realtime", "dashscope_tts_realtime", "qwen_tts_realtime", "qwen3_tts_realtime":
		return true
	default:
		return false
	}
}

func gatewayStringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (s *Server) xiaozhiAudioIngressSummary(session *xiaozhiSession, asrStatus string, ttsStatus string) map[string]any {
	session.mu.Lock()
	id := session.identityLocked()
	decodeStatus := xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
	decodedDurationMS := session.xiaozhiDecodedDurationMSLocked()
	binaryProtocolVersion := session.binaryProtocolVersion
	opusSampleRateHz := session.opusSampleRateHz
	opusChannels := session.opusChannels
	opusFrameDurationMS := session.opusFrameDurationMS
	opusFrameCount := session.opusFrameCount
	opusByteCount := session.opusByteCount
	opusDecodedFrameCount := session.opusDecodedFrameCount
	opusDecodedSampleCount := session.opusDecodedSampleCount
	opusDecodeErrorCount := session.opusDecodeErrorCount
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi."+decodeStatus, s.now().UnixMilli())
	return map[string]any{
		"codec":                "opus",
		"profile":              xiaozhiBinaryProfile(binaryProtocolVersion),
		"sample_rate_hz":       opusSampleRateHz,
		"channels":             opusChannels,
		"frame_duration_ms":    opusFrameDurationMS,
		"decode_status":        decodeStatus,
		"frame_count":          opusFrameCount,
		"byte_count":           opusByteCount,
		"decoded_frame_count":  opusDecodedFrameCount,
		"decoded_sample_count": opusDecodedSampleCount,
		"decoded_duration_ms":  decodedDurationMS,
		"decode_error_count":   opusDecodeErrorCount,
		"asr_status":           asrStatus,
		"tts_status":           ttsStatus,
	}
}

func xiaozhiVoicePipelineSummary(report providers.VoicePipelineReport) map[string]any {
	summary := map[string]any{
		"schema_version":    report.SchemaVersion,
		"status":            report.Status,
		"stage":             "answer",
		"execution_mode":    report.ExecutionMode,
		"audio_chunk_count": report.Output.AudioChunkCount,
		"llm_segment_count": report.Output.LLMSegmentCount,
		"streaming":         report.Output.StreamingAnswer,
		"selection": map[string]any{
			"asr_mode":                 report.Selection.ASRMode,
			"asr_profile":              report.Selection.ASRProfile,
			"asr_profile_env":          report.Selection.ASRProfileEnv,
			"llm_profile":              report.Selection.LLMProfile,
			"llm_profile_env":          report.Selection.LLMProfileEnv,
			"llm_fallback_profile":     report.Selection.LLMFallbackProfile,
			"llm_fallback_profile_env": report.Selection.LLMFallbackProfileEnv,
			"tts_mode":                 report.Selection.TTSMode,
			"tts_profile":              report.Selection.TTSProfile,
			"tts_profile_env":          report.Selection.TTSProfileEnv,
		},
		"timing": map[string]any{
			"asr_first_partial_ms":             report.Timing.ASRFirstPartialMS,
			"asr_final_ms":                     report.Timing.ASRFinalMS,
			"llm_first_content_ms":             report.Timing.LLMFirstContentMS,
			"tts_first_audio_ms":               report.Timing.TTSFirstAudioMS,
			"audio_downlink_first_frame_ms":    report.Timing.AudioDownlinkFirstMS,
			"speech_end_to_final_asr_ms":       report.Timing.SpeechEndToFinalASRMS,
			"speech_end_to_first_llm_token_ms": report.Timing.SpeechEndToFirstTokenMS,
		},
	}
	if report.Fallback != nil && report.Fallback.Activated {
		summary["fallback"] = map[string]any{
			"activated": true,
			"provider":  safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
			"reason":    safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated"),
		}
	}
	return summary
}

func (s *Server) recordVoicePipelineFallback(traceID string, sessionID string, deviceID string, report providers.VoicePipelineReport) {
	if report.Fallback == nil || !report.Fallback.Activated {
		return
	}
	s.metrics.providerFailoverTotal.Inc()
	s.metrics.fallbackTotal.Inc()
	atMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "fallback.used", atMS)
	s.recordTrace(traceID, sessionID, deviceID, "provider.failover", atMS)
}

func (s *Server) localFallbackPayload() protocol.ControlEventPayload {
	return protocol.ControlEventPayload{
		State: protocol.ExpressionLocalFallback,
		Mode:  protocol.ModeLocalFallback,
		Text:  localFallbackText,
		Final: true,
	}
}

func (s *Server) recordLocalFallback(traceID string, sessionID string, deviceID string, reason string) {
	s.metrics.fallbackTotal.Inc()
	atMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, deviceID, "fallback.used", atMS)
	s.recordTrace(traceID, sessionID, deviceID, "local_fallback.entered", atMS)
	if reason != "" {
		s.recordTrace(traceID, sessionID, deviceID, "local_fallback."+safeGatewayFallbackToken(reason, "unavailable"), atMS)
	}
	s.recordDeviceControl(deviceID, traceID, sessionID, s.localFallbackPayload(), atMS)
}

func (s *Server) writeXiaozhiLocalFallback(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	s.recordLocalFallback(task.traceID, task.sessionID, task.deviceID, reason)
	s.recordTrace(task.traceID, task.sessionID, task.deviceID, "xiaozhi.local_fallback.sent", s.now().UnixMilli())
	if err := session.writeXiaozhiJSON(ctx, conn, turn, map[string]any{
		"type":       "tts",
		"state":      "sentence_start",
		"phase":      "local_fallback",
		"mode":       string(protocol.ModeLocalFallback),
		"turn_id":    task.turnID,
		"trace_id":   task.traceID,
		"session_id": task.sessionID,
		"device_id":  task.deviceID,
		"text":       localFallbackText,
	}); err != nil {
		return false
	}
	return s.writeXiaozhiTTSStop(ctx, conn, session, turn, task, "local_fallback")
}

func voicePipelineTextStreamProvider(report providers.VoicePipelineReport) string {
	if report.Fallback != nil && report.Fallback.Activated {
		if provider := safeGatewayFallbackToken(report.Fallback.Provider, "fallback"); provider != "" {
			return provider
		}
	}
	return firstNonEmpty(safeGatewayFallbackToken(report.Selection.LLMProfile, ""), "unknown")
}

func voicePipelineFallbackResponse(report providers.VoicePipelineReport) (bool, string, string) {
	if report.Fallback == nil || !report.Fallback.Activated {
		return false, "", ""
	}
	return true,
		safeGatewayFallbackToken(report.Fallback.Provider, "fallback"),
		safeGatewayFallbackToken(report.Fallback.Reason, "fallback_activated")
}

var gatewayFallbackTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,80}$`)

func safeGatewayFallbackToken(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "http") ||
		strings.Contains(lower, "bearer") ||
		strings.Contains(lower, "sk-") ||
		strings.Contains(lower, "x21") ||
		strings.Contains(lower, "v21") ||
		strings.Contains(value, "/") ||
		strings.Contains(value, "\\") ||
		!gatewayFallbackTokenPattern.MatchString(value) {
		return fallback
	}
	return value
}

func (s *Server) writeXiaozhiPlaceholderTTS(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	s.writeXiaozhiOfficialStackChanState(ctx, session, "thinking", "placeholder_tts_start", false)
	if err := session.writeXiaozhiJSON(ctx, conn, task.turn, map[string]any{
		"type":          "tts",
		"state":         "start",
		"turn_id":       task.turnID,
		"trace_id":      task.traceID,
		"session_id":    task.sessionID,
		"device_id":     task.deviceID,
		"audio_ingress": task.audioIngressSummary("not_connected", "placeholder_only"),
	}); err != nil {
		return
	}
	if err := session.writeXiaozhiJSON(ctx, conn, task.turn, map[string]any{
		"type":        "tts",
		"state":       "sentence_start",
		"turn_id":     task.turnID,
		"trace_id":    task.traceID,
		"session_id":  task.sessionID,
		"device_id":   task.deviceID,
		"placeholder": true,
		"text":        "",
	}); err != nil {
		return
	}
	s.writeXiaozhiTTSStop(ctx, conn, session, task.turn, task, "placeholder_no_asr_tts")
	s.suppressXiaozhiInputAfterNoSpeechPlaceholder(session, task)
}

func (s *Server) writeXiaozhiTTSStop(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	return s.writeXiaozhiTTSStopWithOptions(ctx, conn, session, turn, task, reason, false)
}

func (s *Server) writeXiaozhiTTSStopForce(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn, task xiaozhiTurnTask, reason string) bool {
	return s.writeXiaozhiTTSStopWithOptions(ctx, conn, session, turn, task, reason, true)
}
