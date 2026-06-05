package gateway

import (
	"context"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/protocol"
	stackchantransport "a21.local/a21/internal/transport/stackchan"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func (s *Server) maybeSendXiaozhiTouchReaction(ctx context.Context, session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) {
	if !s.xiaozhiProductTouchReactionsAllowed(session) {
		return
	}
	id := session.identitySnapshot()
	plans := officialTouchReactionPlans(event)
	if len(plans) == 0 {
		return
	}
	socket, officialDeviceID, ok := s.officialStackChanSocketForXiaozhiDevice(id.deviceID)
	if !ok {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.official_ws_not_connected", s.now().UnixMilli())
		s.deliverXiaozhiTouchMCPReaction(ctx, session, event, "official_ws_disconnected")
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	writeCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	deliveredPackets := 0
	surfaces := map[string]string{
		"last_touch_reaction_transport": "stackchan_official_ws",
		"last_touch_reaction_target":    officialDeviceID,
	}
	socket.writeMu.Lock()
	for _, plan := range plans {
		for _, packet := range plan.Packets {
			if err := socket.conn.Write(writeCtx, websocket.MessageBinary, packet.Bytes()); err != nil {
				socket.writeMu.Unlock()
				s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.official_ws_delivery_error", s.now().UnixMilli())
				s.deliverXiaozhiTouchMCPReaction(ctx, session, event, "official_ws_delivery_error")
				return
			}
			deliveredPackets++
		}
		for key, value := range plan.Metadata.Surfaces {
			surfaces["last_touch_reaction_"+key] = value
		}
	}
	socket.writeMu.Unlock()
	if deliveredPackets == 0 {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.official_ws_empty", s.now().UnixMilli())
		return
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.official_ws.sent", s.now().UnixMilli())
	echo := map[string]string{
		"last_touch_reaction_status":  "delivered",
		"last_touch_reaction_packets": strconv.Itoa(deliveredPackets),
	}
	for key, value := range surfaces {
		echo[key] = value
	}
	s.recordXiaozhiTouchReactionEcho(session, event, echo)
}

func (s *Server) deliverXiaozhiTouchMCPReaction(ctx context.Context, session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent, fallbackReason string) {
	if session == nil {
		return
	}
	id := session.identitySnapshot()
	plans := xiaozhiTouchReactionPlans(session, event)
	if len(plans) == 0 {
		s.recordXiaozhiTouchReactionEcho(session, event, map[string]string{
			"last_touch_reaction_status":          "failed_no_mcp_plan",
			"last_touch_reaction_transport":       "xiaozhi_mcp_sequence",
			"last_touch_reaction_fallback_reason": safeGatewayFallbackToken(fallbackReason, "unknown"),
		})
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	deliveredSteps := 0
	for index, req := range plans {
		delivery, status, _ := s.sendXiaozhiMCPControl(ctx, req)
		if status != 0 {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.xiaozhi_mcp_fallback.failed", s.now().UnixMilli())
			s.recordXiaozhiTouchReactionEcho(session, event, map[string]string{
				"last_touch_reaction_status":          "failed_mcp_" + strconv.Itoa(status),
				"last_touch_reaction_transport":       "xiaozhi_mcp_sequence",
				"last_touch_reaction_fallback_reason": safeGatewayFallbackToken(fallbackReason, "unknown"),
				"last_touch_reaction_steps":           strconv.Itoa(deliveredSteps),
			})
			return
		}
		deliveredSteps++
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		reactionMarker := "xiaozhi.touch_reaction.xiaozhi_mcp_fallback.step" + strconv.Itoa(index+1) + "." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, reactionMarker, s.now().UnixMilli())
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.touch_reaction.xiaozhi_mcp_fallback.sent", s.now().UnixMilli())
	s.recordXiaozhiTouchReactionEcho(session, event, map[string]string{
		"last_touch_reaction_status":          "delivered",
		"last_touch_reaction_transport":       "xiaozhi_mcp_sequence",
		"last_touch_reaction_fallback_reason": safeGatewayFallbackToken(fallbackReason, "unknown"),
		"last_touch_reaction_steps":           strconv.Itoa(deliveredSteps),
	})
}

func officialTouchReactionPlans(event xiaozhitransport.DeviceExtensionEvent) []stackchantransport.OfficialActionPlan {
	events := officialTouchReactionEvents(event)
	if len(events) == 0 {
		return nil
	}
	plans := make([]stackchantransport.OfficialActionPlan, 0, len(events))
	for _, reaction := range events {
		plan, err := stackchantransport.BuildOfficialActionPlan(reaction)
		if err != nil {
			continue
		}
		plans = append(plans, plan)
	}
	return plans
}

func officialTouchReactionEvents(event xiaozhitransport.DeviceExtensionEvent) []xiaozhitransport.DeviceExtensionEvent {
	switch event.Value {
	case "screen_tap":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindFace, Value: "attentive"},
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "look_up", YAngle: 68},
		}
	case "screen_barge_in":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindFace, Value: "attentive"},
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "stop"},
		}
	case "top_tap":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "nod"},
		}
	case "top_swipe_forward":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "look_up", YAngle: 78},
		}
	case "top_swipe_backward":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "shake"},
		}
	case "top_barge_in":
		return []xiaozhitransport.DeviceExtensionEvent{
			{Kind: xiaozhitransport.DeviceEventKindFace, Value: "attentive"},
			{Kind: xiaozhitransport.DeviceEventKindMotion, Value: "stop"},
		}
	default:
		return nil
	}
}

func (s *Server) maybeSendXiaozhiStateReaction(ctx context.Context, session *xiaozhiSession, state string, reason string, force bool) {
	if !s.xiaozhiProductStateReactionsAllowed(session) {
		return
	}
	id := session.identitySnapshot()
	if xiaozhiStateReactionSuppressed(state, reason) {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.state_reaction.listen_start_suppressed", s.now().UnixMilli())
		s.recordXiaozhiStateReactionEcho(session, state, reason, map[string]string{
			"last_state_reaction_status": "suppressed_listen_start",
		})
		return
	}
	plans := xiaozhiStateReactionPlans(session, state)
	if len(plans) == 0 {
		return
	}
	if !session.claimXiaozhiStateReaction(state, force) {
		return
	}
	for _, req := range plans {
		delivery, status, _ := s.sendXiaozhiMCPControl(ctx, req)
		if status != 0 {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.state_reaction.failed", s.now().UnixMilli())
			s.recordXiaozhiStateReactionEcho(session, state, reason, map[string]string{
				"last_state_reaction_status": "failed_" + strconv.Itoa(status),
			})
			continue
		}
		genericMarker := "xiaozhi.mcp." + delivery.Marker + ".sent"
		reactionMarker := "xiaozhi.state_reaction." + delivery.Marker + ".sent"
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, genericMarker, s.now().UnixMilli())
		s.recordTrace(delivery.Response.TraceID, delivery.Response.SessionID, delivery.Response.DeviceID, reactionMarker, s.now().UnixMilli())
		echo := xiaozhiMCPActivity(delivery.Marker, delivery.Args)
		echo["last_state_reaction_status"] = "delivered"
		echo["last_state_reaction_tool"] = delivery.Marker
		s.recordXiaozhiStateReactionEcho(session, state, reason, echo)
	}
}

func xiaozhiStateReactionSuppressed(state string, reason string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "listening") &&
		strings.EqualFold(strings.TrimSpace(reason), "listen_start")
}

func xiaozhiStateReactionPlans(session *xiaozhiSession, state string) []XiaozhiMCPControlRequest {
	if session == nil {
		return nil
	}
	id := session.identitySnapshot()
	base := XiaozhiMCPControlRequest{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiReactionInt(red)
		req.Green = xiaozhiReactionInt(green)
		req.Blue = xiaozhiReactionInt(blue)
		return req
	}
	head := func(yaw *int, pitch int, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = yaw
		req.Pitch = xiaozhiReactionInt(pitch)
		req.Speed = xiaozhiReactionInt(speed)
		return req
	}
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "idle":
		return []XiaozhiMCPControlRequest{
			led(20, 20, 40),
			head(xiaozhiReactionInt(0), 24, 160),
		}
	case "listening":
		return []XiaozhiMCPControlRequest{
			led(0, 72, 168),
			head(xiaozhiReactionInt(0), 30, 180),
		}
	case "thinking":
		return []XiaozhiMCPControlRequest{
			led(90, 0, 168),
			head(xiaozhiReactionInt(0), 36, 160),
		}
	case "speaking":
		return []XiaozhiMCPControlRequest{
			led(0, 120, 90),
			head(xiaozhiReactionInt(8), 28, 180),
		}
	case "error", "fatal_error":
		return []XiaozhiMCPControlRequest{
			led(168, 24, 0),
			head(xiaozhiReactionInt(0), 20, 220),
		}
	default:
		return nil
	}
}

func xiaozhiTouchReactionPlans(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) []XiaozhiMCPControlRequest {
	if session == nil {
		return nil
	}
	id := session.identitySnapshot()
	base := XiaozhiMCPControlRequest{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
	led := func(red, green, blue int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetLEDColorToolName
		req.Red = xiaozhiReactionInt(red)
		req.Green = xiaozhiReactionInt(green)
		req.Blue = xiaozhiReactionInt(blue)
		return req
	}
	head := func(yaw *int, pitch int, speed int) XiaozhiMCPControlRequest {
		req := base
		req.ToolName = xiaozhiMCPRobotSetHeadAnglesToolName
		req.Yaw = yaw
		req.Pitch = xiaozhiReactionInt(pitch)
		req.Speed = xiaozhiReactionInt(speed)
		return req
	}
	switch event.Value {
	case "screen_tap":
		return []XiaozhiMCPControlRequest{
			led(0, 120, 168),
			head(nil, 62, 560),
		}
	case "screen_barge_in":
		return []XiaozhiMCPControlRequest{
			led(168, 40, 0),
			head(xiaozhiReactionInt(0), 38, 720),
		}
	case "top_tap":
		return []XiaozhiMCPControlRequest{
			led(60, 0, 168),
			head(nil, 68, 680),
		}
	case "top_swipe_forward":
		return []XiaozhiMCPControlRequest{
			led(0, 120, 90),
			head(xiaozhiReactionInt(45), 70, 780),
		}
	case "top_swipe_backward":
		return []XiaozhiMCPControlRequest{
			led(120, 60, 0),
			head(xiaozhiReactionInt(-45), 45, 780),
		}
	case "top_barge_in":
		return []XiaozhiMCPControlRequest{
			led(168, 24, 0),
			head(xiaozhiReactionInt(0), 35, 720),
		}
	default:
		return nil
	}
}

func xiaozhiReactionInt(value int) *int {
	return &value
}

func (s *Server) recordXiaozhiTouchReactionEcho(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent, echo map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	deviceEvent, source, ok := xiaozhiTouchEventKindAndSource(event)
	if !ok {
		return
	}
	nowMS := s.now().UnixMilli()
	reactionEcho := map[string]string{
		"last_touch_reaction_event":  safeGatewayFallbackToken(event.Value, "unknown"),
		"last_touch_reaction_source": string(source),
	}
	for key, value := range echo {
		reactionEcho[key] = value
	}
	s.mu.Lock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastEvent = deviceEvent
	record.LastTouchEvent = deviceEvent
	record.LastTouchSource = source
	record.LastTouchTraceID = id.traceID
	record.LastTouchSessionID = id.sessionID
	record.LastTouchSeenMS = nowMS
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, reactionEcho)
	s.devices[id.deviceID] = record
	s.mu.Unlock()
}

func (s *Server) recordXiaozhiStateReactionEcho(session *xiaozhiSession, state string, reason string, echo map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	reactionEcho := map[string]string{
		"last_state_reaction_state":  safeGatewayFallbackToken(state, "unknown"),
		"last_state_reaction_reason": safeGatewayFallbackToken(reason, "unknown"),
	}
	for key, value := range echo {
		reactionEcho[key] = value
	}
	s.mu.Lock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, reactionEcho)
	s.devices[id.deviceID] = record
	s.mu.Unlock()
}

func (s *Server) recordXiaozhiPlaybackEvent(session *xiaozhiSession, event string, streamID string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	if event == "device.playback.stop_done" {
		session.mu.Lock()
		session.lastPlaybackStopDoneAtMS = nowMS
		session.mu.Unlock()
	}
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, event, nowMS)
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.devices[id.deviceID]
	if record.DeviceID == "" {
		record.DeviceID = id.deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	record.ConnectionStatus = "online"
	record.LastEvent = protocol.DeviceEventKind(event)
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	if streamID != "" {
		record.PlaybackStream = streamID
	}
	s.devices[id.deviceID] = record
}

func (s *Server) xiaozhiFeatureCapabilities(features xiaozhitransport.HelloFeatures, session *xiaozhiSession) map[string]string {
	capabilities := map[string]string{
		"xiaozhi_profile":   xiaozhiClientProfile(features),
		"xiaozhi_transport": "websocket",
		"xiaozhi_audio":     "opus_16000hz_mono_60ms",
	}
	if features.MCP {
		capabilities["xiaozhi_feature_mcp"] = "true"
		if s.xiaozhiProductStateReactionsAllowed(session) {
			capabilities["xiaozhi_product_state_reactions"] = "true"
		}
	}
	if features.AEC {
		capabilities["xiaozhi_feature_aec"] = "true"
	}
	if features.DeviceEvents {
		capabilities["xiaozhi_feature_device_events"] = "true"
	}
	if features.PlaybackEvents {
		capabilities["xiaozhi_feature_playback_events"] = "true"
		if s.xiaozhiProductPlaybackEventsAllowed(session) {
			capabilities["xiaozhi_product_playback_events"] = "true"
		}
	}
	if features.KeepaliveEvents {
		capabilities["xiaozhi_feature_keepalive_events"] = "true"
		if s.xiaozhiProductKeepaliveEventsAllowed(session) {
			capabilities["xiaozhi_product_keepalive_events"] = "true"
		}
	}
	if features.TouchEvents {
		capabilities["xiaozhi_feature_touch_events"] = "true"
		if s.xiaozhiProductTouchEventsAllowed(session) {
			capabilities["xiaozhi_product_touch_events"] = "true"
		}
		if s.xiaozhiProductTouchReactionsAllowed(session) {
			capabilities["xiaozhi_product_touch_reactions"] = "true"
		}
	}
	if features.DebugMetrics {
		capabilities["xiaozhi_feature_debug_metrics"] = "true"
	}
	if xiaozhiClientProfile(features) == "debug" {
		capabilities["xiaozhi_debug_extension_isolated"] = "true"
	}
	return capabilities
}

func (s *Server) xiaozhiProductPlaybackEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductPlaybackEvents &&
		features.PlaybackEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductKeepaliveEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductPlaybackEvents &&
		features.KeepaliveEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductTouchEventsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductTouchEvents &&
		features.TouchEvents &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func (s *Server) xiaozhiProductTouchReactionsAllowed(session *xiaozhiSession) bool {
	return s != nil &&
		s.xiaozhiProductTouchReactions &&
		s.xiaozhiProductTouchEventsAllowed(session)
}

func (s *Server) xiaozhiProductStateReactionsAllowed(session *xiaozhiSession) bool {
	features := session.featuresSnapshot()
	id := session.identitySnapshot()
	return s != nil &&
		s.xiaozhiProductStateReactions &&
		features.MCP &&
		!features.DeviceEvents &&
		!features.DebugMetrics &&
		hardwareMACDeviceID(id.deviceID)
}

func xiaozhiClientProfile(features xiaozhitransport.HelloFeatures) string {
	if features.DeviceEvents || features.DebugMetrics {
		return "debug"
	}
	return "stock"
}

func (s *Server) xiaozhiListenMode(raw string, features xiaozhitransport.HelloFeatures) protocol.Mode {
	if strings.TrimSpace(strings.ToLower(raw)) == string(protocol.ModeProfessional) {
		return protocol.ModeProfessional
	}
	if s.xiaozhiStockProfessionalRouteSelected(raw, features) {
		return protocol.ModeProfessional
	}
	return protocol.ModeWorkmate
}

func (s *Server) xiaozhiStockProfessionalRouteSelected(raw string, features xiaozhitransport.HelloFeatures) bool {
	return s.xiaozhiStockProfessional &&
		s.selectedVoiceMode() == VoiceModeProfessional &&
		xiaozhiClientProfile(features) == "stock" &&
		strings.TrimSpace(strings.ToLower(raw)) != string(protocol.ModeProfessional)
}

func mergeDeviceCapabilities(existing map[string]string, additions map[string]string) map[string]string {
	if len(existing) == 0 {
		merged := make(map[string]string, len(additions))
		for key, value := range additions {
			merged[key] = value
		}
		return merged
	}
	merged := make(map[string]string, len(existing)+len(additions))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range additions {
		merged[key] = value
	}
	return merged
}

func (s *Server) newXiaozhiTurnTask(session *xiaozhiSession, turn *xiaozhiTurn) xiaozhiTurnTask {
	session.mu.Lock()
	id := session.identityLocked()
	decodeStatus := xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
	decodedDurationMS := session.xiaozhiDecodedDurationMSLocked()
	streamingASRPartialText := session.streamingASRPartialText
	streamingASRFinalText := session.streamingASRFinalText
	streamingASRUsed := session.streamingASRHasFinal && strings.TrimSpace(session.streamingASRFinalText) != ""
	binaryProtocolVersion := session.binaryProtocolVersion
	opusSampleRateHz := session.opusSampleRateHz
	opusChannels := session.opusChannels
	opusFrameDurationMS := session.opusFrameDurationMS
	opusFrameCount := session.opusFrameCount
	opusByteCount := session.opusByteCount
	opusDecodedFrameCount := session.opusDecodedFrameCount
	opusDecodedSampleCount := session.opusDecodedSampleCount
	opusDecodeErrorCount := session.opusDecodeErrorCount
	voicePipelineFrames := copyXiaozhiVoicePipelineFrames(session.voicePipelineFrames)
	voicePipelineHasSpeech := session.voicePipelineHasSpeech
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi."+decodeStatus, s.now().UnixMilli())
	return xiaozhiTurnTask{
		turn:      turn,
		turnID:    xiaozhiTurnID(turn),
		traceID:   id.traceID,
		sessionID: id.sessionID,
		deviceID:  id.deviceID,
		mode:      xiaozhiTurnMode(turn),
		audioIngressBase: map[string]any{
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
		},
		voicePipelineFrames:     voicePipelineFrames,
		voicePipelineHasSpeech:  voicePipelineHasSpeech,
		streamingASRPartialText: streamingASRPartialText,
		streamingASRFinalText:   streamingASRFinalText,
		streamingASRUsed:        streamingASRUsed,
	}
}

func (task xiaozhiTurnTask) audioIngressSummary(asrStatus string, ttsStatus string) map[string]any {
	summary := make(map[string]any, len(task.audioIngressBase)+2)
	for key, value := range task.audioIngressBase {
		summary[key] = value
	}
	summary["asr_status"] = asrStatus
	summary["tts_status"] = ttsStatus
	return summary
}

func (s *Server) startXiaozhiTurnTask(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, task xiaozhiTurnTask) {
	go func() {
		defer session.completeXiaozhiTurn(task.turn)
		s.writeXiaozhiTTS(ctx, conn, session, task)
	}()
}
