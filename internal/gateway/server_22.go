package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

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
	if req.Mode != protocol.ModeRoleplay && req.Mode != protocol.ModeWorkmate && req.Mode != protocol.ModeCompanion {
		http.Error(w, "fast companion mode must be roleplay, companion, or workmate", http.StatusBadRequest)
		return
	}
	selectedVoiceMode := s.selectedVoiceMode()
	if !voiceModeAvailableForFastCompanion(selectedVoiceMode) {
		http.Error(w, plannedVoiceModeError(selectedVoiceMode), http.StatusConflict)
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
	frames, err := fastCompanionVoicePipelineFrames(req.LocalAudio.Frames)
	if err != nil {
		http.Error(w, "invalid local_audio.frames", http.StatusBadRequest)
		return
	}
	roleplay, _, err := s.roleplayRuntimeSummary(req.Roleplay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	receivedAtMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.turn.received", receivedAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.profile.ready", receivedAtMS)
	if roleplay.MemoryPromptInputReady {
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.memory.ready", receivedAtMS)
	}
	writeJSON(w, http.StatusOK, s.fastCompanionTurnResponse(r.Context(), req, receivedAtMS, frames, roleplay))
}

func (s *Server) fastCompanionTurnResponse(ctx context.Context, req FastCompanionTurnRequest, receivedAtMS int64, frames []providers.VoicePipelinePCMFrame, roleplay RoleplayRuntimeSummary) FastCompanionTurnResponse {
	if len(frames) > 0 {
		return s.fastCompanionVoicePipelineTurnResponse(ctx, req, receivedAtMS, frames, roleplay)
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	asrFirstPartialAtMS := receivedAtMS + req.LocalAudio.FirstPartialMS
	placeholderStartAtMS := asrFirstPartialAtMS + 1
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.local_audio.frontend.accepted", receivedAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "asr.first_partial", asrFirstPartialAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.text_stream.route.placeholder", placeholderStartAtMS)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_byte", placeholderStartAtMS+1)
	s.recordTrace(traceID, sessionID, req.DeviceID, "provider.first_content", placeholderStartAtMS+2)
	s.recordTrace(traceID, sessionID, req.DeviceID, "tts.first_audio", placeholderStartAtMS+3)
	s.recordTrace(traceID, sessionID, req.DeviceID, "audio.downlink.first_frame", placeholderStartAtMS+4)
	s.recordTrace(traceID, sessionID, req.DeviceID, "device.playback.start", placeholderStartAtMS+5)
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
		Roleplay:           roleplay,
		Events:             events,
	}
}

func (s *Server) fastCompanionVoicePipelineTurnResponse(ctx context.Context, req FastCompanionTurnRequest, receivedAtMS int64, frames []providers.VoicePipelinePCMFrame, roleplay RoleplayRuntimeSummary) FastCompanionTurnResponse {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.local_audio.frontend.accepted", receivedAtMS)
	startAtMS := s.now().UnixMilli()
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.start", startAtMS)
	newRunner := s.currentXiaozhiVoicePipelineRunnerFactory()
	if newRunner == nil {
		newRunner = defaultXiaozhiVoicePipelineRunner
	}
	runner := newRunner()
	request := providers.VoicePipelineRequest{
		Session: providers.VoiceSession{
			TraceID:   traceID,
			SessionID: sessionID,
			DeviceID:  req.DeviceID,
		},
		Mode:              string(req.Mode),
		Frames:            append([]providers.VoicePipelinePCMFrame(nil), frames...),
		VoiceCloneProfile: roleplay.VoiceCloneProfile,
	}
	if prompt, err := s.roleplayPromptInput(req.Roleplay, "我在。"); err == nil && strings.TrimSpace(prompt) != "" {
		request.TextPrompt = prompt
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.prompt_input.used", s.now().UnixMilli())
	}
	if validVoiceCloneProfile(roleplay.VoiceCloneProfile) {
		s.recordTrace(traceID, sessionID, req.DeviceID, "roleplay.voice_clone_profile.used", s.now().UnixMilli())
	}
	result, err := runner.Run(ctx, request)
	s.recordVoicePipelineFallback(traceID, sessionID, req.DeviceID, result.Report)
	fallbackUsed, fallbackProvider, fallbackReason := voicePipelineFallbackResponse(result.Report)
	if err != nil || result.Status != providers.VoicePipelineStatusCompleted || len(result.AudioChunks) == 0 {
		s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.unavailable", s.now().UnixMilli())
		s.recordLocalFallback(traceID, sessionID, req.DeviceID, "voice_pipeline_unavailable")
		events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
			{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
			{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "本地语音链路暂时没有给出可播放答案。"},
			s.localFallbackPayload(),
		})
		return FastCompanionTurnResponse{
			TraceID:                    traceID,
			SessionID:                  sessionID,
			DeviceID:                   req.DeviceID,
			Mode:                       protocol.ModeLocalFallback,
			Status:                     "local_fallback",
			Route:                      "fast_companion_hybrid",
			AudioFrontend:              "local_audio",
			TextStreamProvider:         voicePipelineTextStreamProvider(result.Report),
			ProviderFamily:             string(providers.ProviderFamilyTextStream),
			TextStreamExecuted:         result.Report.Output.LLMContentChars > 0,
			TextStreamFallbackUsed:     fallbackUsed,
			TextStreamFallbackProvider: fallbackProvider,
			TextStreamFallbackReason:   fallbackReason,
			Roleplay:                   roleplay,
			Events:                     events,
		}
	}
	s.recordXiaozhiVoicePipelineStageMarkers(traceID, sessionID, req.DeviceID, startAtMS, result.Timing)
	s.recordTrace(traceID, sessionID, req.DeviceID, "fast_companion.voice_pipeline.completed", s.now().UnixMilli())
	streamID := "a21-fast-companion-voice-pipeline"
	events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
		{State: protocol.ExpressionThinking, Mode: req.Mode, Text: "我把本地语音接到文本流。"},
		{State: protocol.ExpressionSpeaking, Mode: req.Mode, Final: true, StreamID: streamID},
	})
	sentAt := s.now().UnixMilli()
	for _, chunk := range result.AudioChunks {
		firstChunk := len(events) == 3
		chunkAtMS := sentAt + int64(len(events)-3)
		if firstChunk {
			s.recordTrace(traceID, sessionID, req.DeviceID, "audio.downlink.first_frame", sentAt)
			s.recordTrace(traceID, sessionID, req.DeviceID, "device.playback.start", chunkAtMS+1)
		}
		chunk := chunk
		events = append(events, s.voiceAudioPlaybackChunk(req.DeviceID, traceID, sessionID, uint64(len(events)+1), chunkAtMS, streamID, &chunk))
	}
	return FastCompanionTurnResponse{
		TraceID:                    traceID,
		SessionID:                  sessionID,
		DeviceID:                   req.DeviceID,
		Mode:                       req.Mode,
		Status:                     "pipeline_completed",
		Route:                      "fast_companion_hybrid",
		AudioFrontend:              "local_audio",
		TextStreamProvider:         voicePipelineTextStreamProvider(result.Report),
		ProviderFamily:             string(providers.ProviderFamilyTextStream),
		TextStreamExecuted:         result.Report.Output.LLMContentChars > 0,
		TextStreamFallbackUsed:     fallbackUsed,
		TextStreamFallbackProvider: fallbackProvider,
		TextStreamFallbackReason:   fallbackReason,
		Roleplay:                   roleplay,
		Events:                     events,
	}
}

func fastCompanionVoicePipelineFrames(raw []FastCompanionLocalAudioFrame) ([]providers.VoicePipelinePCMFrame, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > 64 {
		return nil, fmt.Errorf("too many local audio frames")
	}
	frames := make([]providers.VoicePipelinePCMFrame, 0, len(raw))
	for i, frame := range raw {
		codec := strings.ToLower(strings.TrimSpace(frame.Codec))
		if codec == "" {
			codec = string(protocol.AudioCodecPCMS16LE)
		}
		if codec != string(protocol.AudioCodecPCMS16LE) {
			return nil, fmt.Errorf("local audio frame codec must be pcm_s16le")
		}
		if frame.SampleRateHz <= 0 || frame.Channels != 1 || frame.DurationMS <= 0 {
			return nil, fmt.Errorf("local audio frame audio shape invalid")
		}
		if frame.RMS < 0 {
			return nil, fmt.Errorf("local audio frame rms invalid")
		}
		pcm, err := base64.StdEncoding.DecodeString(strings.TrimSpace(frame.DataBase64))
		if err != nil || len(pcm) == 0 || len(pcm)%2 != 0 {
			return nil, fmt.Errorf("local audio frame payload invalid")
		}
		if frame.ByteCount > 0 && frame.ByteCount != len(pcm) {
			return nil, fmt.Errorf("local audio frame byte count mismatch")
		}
		seq := frame.Seq
		if seq == 0 {
			seq = uint64(i + 1)
		}
		frames = append(frames, providers.VoicePipelinePCMFrame{
			Seq:          seq,
			Codec:        codec,
			SampleRateHz: frame.SampleRateHz,
			Channels:     frame.Channels,
			DurationMS:   frame.DurationMS,
			ByteCount:    len(pcm),
			RMS:          frame.RMS,
			PCM16LE:      pcm,
		})
	}
	return frames, nil
}

func (s *Server) mockTurnResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.mockTurnTotal.Inc()
	if req.Mode == "" {
		req.Mode = protocol.ModeWorkmate
	}
	if req.Mode == protocol.ModeProfessional {
		return s.professionalTurnResponse(req)
	}
	if professionalVoiceTriggerModeAllowed(req.Mode) && professionalVoiceTrigger(req.Text) {
		traceID, sessionID := s.ids(req.TraceID, req.SessionID)
		req.TraceID = traceID
		req.SessionID = sessionID
		req.Mode = protocol.ModeProfessional
		s.recordTrace(traceID, sessionID, req.DeviceID, "professional.voice_trigger.detected", s.now().UnixMilli())
		return s.professionalTurnResponse(req)
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	payloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: req.Mode, Text: "我在听"},
	}
	providerEvents, err := s.currentVoiceProvider().StartTurn(context.Background(), providers.VoiceTurnRequest{
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
	response, err := s.executeProfessionalQuery(ProfessionalQueryRequest{
		DeviceID:  req.DeviceID,
		Text:      req.Text,
		TraceID:   req.TraceID,
		SessionID: req.SessionID,
	})
	if err != nil {
		traceID, sessionID := s.ids(req.TraceID, req.SessionID)
		events := s.controlSequence(req.DeviceID, traceID, sessionID, []protocol.ControlEventPayload{{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "专业模式参数不完整。我先停在证据边界。",
			Final: true,
		}})
		return MockTurnResponse{TraceID: traceID, SessionID: sessionID, DeviceID: req.DeviceID, Events: events}
	}
	return MockTurnResponse{TraceID: response.TraceID, SessionID: response.SessionID, DeviceID: response.DeviceID, Events: response.Events}
}

func (s *Server) executeProfessionalQuery(req ProfessionalQueryRequest) (ProfessionalQueryResponse, error) {
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	deviceID := strings.TrimSpace(req.DeviceID)
	utterance := professionalQueryUtterance(req)
	workspace, err := s.professionalWorkspaceRuntime(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
		QueryScope:  req.QueryScope,
	})
	if err != nil {
		return ProfessionalQueryResponse{}, err
	}
	preQueryPayloads := []protocol.ControlEventPayload{
		{State: protocol.ExpressionListening, Mode: protocol.ModeProfessional, Text: "我在听"},
		{State: protocol.ExpressionProfessional, Mode: protocol.ModeProfessional, Text: "进入专业模式。情绪先放旁边，现在只看证据。"},
		{State: protocol.ExpressionThinking, Mode: protocol.ModeProfessional, Text: "我在查，先把证据和置信度拉出来。"},
	}
	events := s.controlSequence(deviceID, traceID, sessionID, preQueryPayloads)
	s.recordTrace(traceID, sessionID, deviceID, "professional.query.started", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.checking_feedback.sent", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.workspace.ready", s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "professional.query_scope."+workspace.QueryScope, s.now().UnixMilli())
	utteranceBucket := v21UtteranceLengthBucket(utterance)
	queryRequest := v21adapter.QueryRequest{
		TraceID:            traceID,
		SessionID:          sessionID,
		DeviceID:           deviceID,
		UserID:             workspace.UserID,
		WorkspaceID:        workspace.WorkspaceID,
		Mode:               "professional",
		QueryScope:         workspace.QueryScope,
		Utterance:          utterance,
		LatencyProfile:     "fast_first",
		AnswerStyle:        "voice_first_with_citations",
		MaxFirstResponseMS: 1200,
		PrivacyScope:       "professional_only",
	}
	readRecordID := s.startProfessionalReadRecord(queryRequest, utteranceBucket)
	bindingDecision := s.professionalDeviceBindingDecision(deviceID, workspace.UserID, workspace.WorkspaceID, workspace.QueryScope)
	deviceBinding := ProfessionalQueryDeviceBinding{
		Policy:      bindingDecision.Policy,
		Status:      bindingDecision.Status,
		BindingID:   bindingDecision.BindingID,
		FailureCode: bindingDecision.FailureCode,
	}
	baseResponse := ProfessionalQueryResponse{
		SchemaVersion:          ProfessionalQuerySchemaVersion,
		Service:                DeviceRegistryServiceName,
		Status:                 "started",
		TraceID:                traceID,
		SessionID:              sessionID,
		DeviceID:               deviceID,
		Mode:                   VoiceModeProfessional,
		Route:                  "professional_query",
		AdapterContractVersion: ProfessionalAdapterContractVersion,
		Workspace:              workspace,
		DeviceBinding:          deviceBinding,
		ReadRecordID:           readRecordID,
		ReadRecordStatus:       "started",
		Events:                 events,
		Redaction:              professionalWorkspaceRedaction(),
	}
	s.recordTrace(traceID, sessionID, deviceID, bindingDecision.TraceMarker, s.now().UnixMilli())
	if !bindingDecision.Allowed {
		s.failProfessionalReadRecord(readRecordID, bindingDecision.FailureCode)
		events = append(events, s.controlSequenceFrom(deviceID, traceID, sessionID, uint64(len(events)+1), []protocol.ControlEventPayload{{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "这台设备还没绑定到当前专业工作区。我先停在证据边界，避免把个人资料查错地方。",
			Final: true,
		}})...)
		baseResponse.Status = "failed"
		baseResponse.FailureCode = bindingDecision.FailureCode
		baseResponse.ReadRecordStatus = "failed"
		baseResponse.Events = events
		baseResponse.DeviceBinding = deviceBinding
		return baseResponse, nil
	}
	s.recordTrace(traceID, sessionID, deviceID, "v21.query.utterance."+utteranceBucket, s.now().UnixMilli())
	s.recordTrace(traceID, sessionID, deviceID, "v21.query.start", s.now().UnixMilli())
	queryCtx, cancel := context.WithTimeout(context.Background(), s.v21TTL)
	defer cancel()
	started := time.Now()
	response, err := s.v21.Query(queryCtx, queryRequest)
	s.metrics.v21QueryMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	postQueryPayloads := make([]protocol.ControlEventPayload, 0, 1)
	if err != nil {
		s.failProfessionalReadRecord(readRecordID, professionalReadFailureCode(err, queryCtx))
		s.recordV21QueryFailure(traceID, sessionID, deviceID, err, queryCtx)
		postQueryPayloads = append(postQueryPayloads, protocol.ControlEventPayload{
			State: protocol.ExpressionError,
			Mode:  protocol.ModeProfessional,
			Text:  "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。",
			Final: true,
		})
		baseResponse.Status = "failed"
		baseResponse.FailureCode = professionalReadFailureCode(err, queryCtx)
		baseResponse.ReadRecordStatus = "failed"
	} else {
		report, reportErr := v21adapter.NewProfessionalBridgeEvidenceReport(response)
		if reportErr != nil {
			s.failProfessionalReadRecord(readRecordID, "contract_invalid")
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.error", s.now().UnixMilli())
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.error.contract_invalid", s.now().UnixMilli())
			postQueryPayloads = append(postQueryPayloads, protocol.ControlEventPayload{
				State: protocol.ExpressionError,
				Mode:  protocol.ModeProfessional,
				Text:  "V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。",
				Final: true,
			})
			baseResponse.Status = "failed"
			baseResponse.FailureCode = "contract_invalid"
			baseResponse.ReadRecordStatus = "failed"
		} else {
			s.completeProfessionalReadRecord(readRecordID, response)
			s.recordTrace(traceID, sessionID, deviceID, "v21.query.first_result", s.now().UnixMilli())
			answer := protocol.ControlEventPayload{
				State:        protocol.ExpressionSpeaking,
				Mode:         protocol.ModeProfessional,
				Text:         response.FastAnswer,
				Final:        true,
				Confidence:   response.Confidence,
				Evidence:     v21EvidenceToProtocol(response.Evidence),
				SpeechBlocks: response.SpeechBlocks,
				ScreenCards:  v21ScreenCardsToProtocol(response.ScreenCards),
				FollowUps:    response.FollowUps,
			}
			postQueryPayloads = append(postQueryPayloads, answer)
			baseResponse.Status = "completed"
			baseResponse.ReadRecordStatus = "completed"
			baseResponse.Answer = &answer
			baseResponse.EvidenceReport = &report
		}
	}
	events = append(events, s.controlSequenceFrom(deviceID, traceID, sessionID, uint64(len(events)+1), postQueryPayloads)...)
	baseResponse.Events = events
	return baseResponse, nil
}

func (s *Server) recordV21QueryFailure(traceID string, sessionID string, deviceID string, err error, queryCtx context.Context) {
	for _, marker := range v21QueryFailureMarkers(err, queryCtx) {
		s.recordTrace(traceID, sessionID, deviceID, marker, s.now().UnixMilli())
	}
}

func v21QueryFailureMarkers(err error, queryCtx context.Context) []string {
	if errors.Is(err, context.DeadlineExceeded) || (queryCtx != nil && errors.Is(queryCtx.Err(), context.DeadlineExceeded)) {
		return []string{"v21.query.timeout", "v21.query.error.timeout"}
	}
	markers := []string{"v21.query.error"}
	if class := v21adapter.QueryFailureClassOf(err); class != "" {
		markers = append(markers, "v21.query.error."+string(class))
	}
	if statusClass := v21adapter.QueryFailureStatusClassOf(err); statusClass != "" {
		markers = append(markers, "v21.query.error."+statusClass)
	}
	return markers
}

func v21UtteranceLengthBucket(utterance string) string {
	length := len([]rune(strings.TrimSpace(utterance)))
	switch {
	case length <= 0:
		return "length_empty"
	case length <= 16:
		return "length_1_16"
	case length <= 64:
		return "length_17_64"
	case length <= 160:
		return "length_65_160"
	default:
		return "length_gt_160"
	}
}

func (s *Server) mockInterruptResponse(req MockTurnRequest) MockTurnResponse {
	s.metrics.bargeInTotal.Inc()
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	mode := req.Mode
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	payloads := make([]protocol.ControlEventPayload, 0, 2)
	providerEvents, err := s.currentVoiceProvider().Cancel(context.Background(), providers.VoiceCancelRequest{
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
	return s.controlSequenceFrom(deviceID, traceID, sessionID, 1, payloads)
}

func (s *Server) controlSequenceFrom(deviceID string, traceID string, sessionID string, startSeq uint64, payloads []protocol.ControlEventPayload) []protocol.Envelope {
	events := make([]protocol.Envelope, 0, len(payloads))
	sentAt := s.now().UnixMilli()
	for i, payload := range payloads {
		data, _ := json.Marshal(payload)
		events = append(events, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  deviceID,
			Kind:      protocol.KindControlEvent,
			Seq:       startSeq + uint64(i),
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
