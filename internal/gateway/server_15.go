package gateway

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
)

func (s *Server) handleXiaozhiBinary(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, data []byte) bool {
	if !session.helloReceivedSnapshot() {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, "hello_required", "hello is required before binary audio"))
		return true
	}
	frame, err := xiaozhitransport.ParseBinaryFrameVersion(data, xiaozhitransport.DirectionDeviceToServer, session.identity(), session.binaryProtocolVersionSnapshot())
	if err != nil {
		_ = session.writeXiaozhiJSON(ctx, conn, nil, s.xiaozhiError(session, xiaozhiErrorCode(err), xiaozhiErrorDetail(err)))
		return true
	}
	session.adoptFrame(frame)
	session.mu.Lock()
	listening := session.listening
	suppressedListenActive := session.suppressedListenActive
	id := session.identityLocked()
	session.mu.Unlock()
	if !listening {
		if suppressedListenActive {
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.ignored_suppressed_listen", s.now().UnixMilli())
			return true
		}
		if session.shouldBufferXiaozhiWakePreroll(s.now().UnixMilli()) {
			return s.bufferXiaozhiWakePreroll(session, frame)
		}
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.ignored_not_listening", s.now().UnixMilli())
		return true
	}
	session.mu.Lock()
	session.opusFrameCount++
	session.opusByteCount += frame.Opus.PayloadBytes
	seq := uint64(session.opusFrameCount)
	id = session.identityLocked()
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.received", s.now().UnixMilli())
	turn := session.currentXiaozhiTurn()
	if s.enqueueXiaozhiOpusIngressFrame(ctx, conn, session, frame, turn, seq) {
		return true
	}
	s.processXiaozhiOpusIngressFrame(ctx, conn, session, frame, seq)
	return true
}

func (s *Server) startXiaozhiOpusIngressQueue(ctx context.Context, session *xiaozhiSession) {
	if session == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	queueCtx, cancel := context.WithCancel(ctx)
	queue := make(chan xiaozhiOpusIngressFrame, maxXiaozhiOpusIngressQueueFrames)
	session.mu.Lock()
	if session.opusIngressCancel != nil {
		session.opusIngressCancel()
	}
	session.opusIngressCtx = queueCtx
	session.opusIngressCancel = cancel
	session.opusIngressQueue = queue
	session.mu.Unlock()
	go s.runXiaozhiOpusIngressQueue(queueCtx, session, queue)
}

func (s *Server) cancelXiaozhiOpusIngressQueue(session *xiaozhiSession, reason string) {
	if session == nil {
		return
	}
	session.mu.Lock()
	cancel := session.opusIngressCancel
	session.opusIngressCancel = nil
	session.opusIngressCtx = nil
	session.opusIngressQueue = nil
	session.mu.Unlock()
	if cancel != nil {
		cancel()
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queue_cancelled."+safeGatewayFallbackToken(reason, "unknown"), s.now().UnixMilli())
	}
}

func (s *Server) enqueueXiaozhiOpusIngressFrame(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, frame xiaozhitransport.Frame, turn *xiaozhiTurn, seq uint64) bool {
	queue := session.xiaozhiOpusIngressQueue()
	if queue == nil {
		return false
	}
	item := xiaozhiOpusIngressFrame{
		ctx:   firstNonNilContext(xiaozhiTurnContext(turn), ctx),
		conn:  conn,
		frame: frame,
		turn:  turn,
		seq:   seq,
	}
	select {
	case queue <- item:
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queued", s.now().UnixMilli())
	default:
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.queue_dropped", s.now().UnixMilli())
	}
	return true
}

func (s *Server) runXiaozhiOpusIngressQueue(ctx context.Context, session *xiaozhiSession, queue <-chan xiaozhiOpusIngressFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-queue:
			s.processXiaozhiOpusIngressFrame(item.ctx, item.conn, session, item.frame, item.seq)
		}
	}
}

func (s *Server) startXiaozhiListenStopAfterIngressDrain(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, turn *xiaozhiTurn) {
	if session == nil || turn == nil {
		return
	}
	go func() {
		waitCtx, cancel := context.WithTimeout(firstNonNilContext(turn.ctx, ctx), 300*time.Millisecond)
		defer cancel()
		if !s.waitXiaozhiOpusIngressDrained(waitCtx, session) {
			id := session.identitySnapshot()
			s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.drain_timeout", s.now().UnixMilli())
		}
		if session.shouldAbortXiaozhiTurn(turn) {
			return
		}
		if s.startXiaozhiStreamingASRCommit(ctx, conn, session, turn) {
			return
		}
		task := s.newXiaozhiTurnTask(session, turn)
		s.startXiaozhiTurnTask(ctx, conn, session, task)
	}()
}

func (s *Server) waitXiaozhiOpusIngressDrained(ctx context.Context, session *xiaozhiSession) bool {
	for {
		session.mu.Lock()
		received := session.opusFrameCount
		processed := session.opusIngressProcessedFrameCount
		session.mu.Unlock()
		if processed >= received {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func xiaozhiTurnContext(turn *xiaozhiTurn) context.Context {
	if turn == nil {
		return nil
	}
	return turn.ctx
}

func firstNonNilContext(contexts ...context.Context) context.Context {
	for _, ctx := range contexts {
		if ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (s *Server) processXiaozhiOpusIngressFrame(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, frame xiaozhitransport.Frame, seq uint64) {
	defer func() {
		session.mu.Lock()
		session.opusIngressProcessedFrameCount++
		session.mu.Unlock()
	}()
	if ctx != nil && ctx.Err() != nil {
		id := session.identitySnapshot()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_ingress.stale_frame_suppressed", s.now().UnixMilli())
		return
	}
	session.mu.Lock()
	if session.opusCodec == nil {
		session.opusDecodeErrorCount++
		id := session.identityLocked()
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decode_error", s.now().UnixMilli())
		return
	}
	pcm, err := session.opusCodec.DecodePCM16(frame.Opus.Payload)
	if err != nil {
		session.opusDecodeErrorCount++
		id := session.identityLocked()
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decode_error", s.now().UnixMilli())
		return
	}
	session.opusDecodedFrameCount++
	session.opusDecodedSampleCount += len(pcm)
	id := session.identityLocked()
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.opus_frame.decoded", s.now().UnixMilli())
	s.recordXiaozhiDeviceActivity(session, "xiaozhi.opus_frame.decoded", map[string]string{
		"microphone": "available_xiaozhi_opus_ingress",
	})
	s.observeXiaozhiDecodedIngress(ctx, conn, session, pcm, seq)
}

func (s *Server) bufferXiaozhiWakePreroll(session *xiaozhiSession, frame xiaozhitransport.Frame) bool {
	if session == nil || frame.Opus == nil {
		return true
	}
	id := session.identitySnapshot()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.received", s.now().UnixMilli())
	session.mu.Lock()
	if session.opusCodec == nil {
		session.mu.Unlock()
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.decode_error", s.now().UnixMilli())
		return true
	}
	sampleRateHz := session.opusSampleRateHz
	channels := session.opusChannels
	pcm, err := session.opusCodec.DecodePCM16(frame.Opus.Payload)
	session.mu.Unlock()
	if err != nil {
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.decode_error", s.now().UnixMilli())
		return true
	}
	if len(pcm) == 0 || sampleRateHz <= 0 || channels <= 0 {
		return true
	}
	durationMS := len(pcm) * 1000 / sampleRateHz / channels
	chunk := protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   pcm16Base64(pcm),
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     id.deviceID,
		TraceID:      id.traceID,
		SessionID:    id.sessionID,
		Seq:          0,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	pipelineFrame := providers.VoicePipelinePCMFrame{
		Codec:        string(chunk.Codec),
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		ByteCount:    len(pcm) * 2,
		RMS:          result.RMS,
		PCM16LE:      pcm16Bytes(pcm),
	}
	session.mu.Lock()
	seq := uint64(len(session.wakePrerollFrames) + 1)
	pipelineFrame.Seq = seq
	session.wakePrerollFrames = append(session.wakePrerollFrames, pipelineFrame)
	session.wakePrerollPayloadBytes = append(session.wakePrerollPayloadBytes, frame.Opus.PayloadBytes)
	if len(session.wakePrerollFrames) > maxXiaozhiWakePrerollFrames {
		session.wakePrerollFrames = session.wakePrerollFrames[len(session.wakePrerollFrames)-maxXiaozhiWakePrerollFrames:]
		session.wakePrerollPayloadBytes = session.wakePrerollPayloadBytes[len(session.wakePrerollPayloadBytes)-maxXiaozhiWakePrerollFrames:]
	}
	if result.SpeechDetected || result.SpeechActive || containsAudioIngressEvent(result.Events, audio.EventVADSpeechStart) {
		session.wakePrerollHasSpeech = true
	}
	session.mu.Unlock()
	s.recordAudioCaptureFrame(protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  id.deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
		SentAtMS:  s.now().UnixMilli(),
	}, chunk, result, id.traceID, id.sessionID)
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.opus_frame.buffered", s.now().UnixMilli())
	return true
}

func (s *Server) attachXiaozhiWakePreroll(ctx context.Context, session *xiaozhiSession) {
	if session == nil {
		return
	}
	session.mu.Lock()
	if len(session.wakePrerollFrames) == 0 {
		session.mu.Unlock()
		return
	}
	id := session.identityLocked()
	frames := append([]providers.VoicePipelinePCMFrame(nil), session.wakePrerollFrames...)
	payloadBytes := append([]int(nil), session.wakePrerollPayloadBytes...)
	for i := range frames {
		frames[i].PCM16LE = append([]byte(nil), session.wakePrerollFrames[i].PCM16LE...)
	}
	hasSpeech := session.wakePrerollHasSpeech
	session.wakePrerollFrames = nil
	session.wakePrerollPayloadBytes = nil
	session.wakePrerollHasSpeech = false
	session.voicePipelineFrames = append(session.voicePipelineFrames, frames...)
	for i, frame := range frames {
		session.opusFrameCount++
		if i < len(payloadBytes) {
			session.opusByteCount += payloadBytes[i]
		}
		session.opusDecodedFrameCount++
		session.opusDecodedSampleCount += frame.ByteCount / 2
	}
	if hasSpeech {
		session.voicePipelineHasSpeech = true
	}
	session.mu.Unlock()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "xiaozhi.wake_preroll.attached", s.now().UnixMilli())
	for _, frame := range frames {
		s.appendXiaozhiStreamingASRFrame(ctx, session, frame)
	}
}

func (s *Server) observeXiaozhiDecodedIngress(ctx context.Context, conn *websocket.Conn, session *xiaozhiSession, pcm []int16, seq uint64) {
	session.mu.Lock()
	id := session.identityLocked()
	sampleRateHz := session.opusSampleRateHz
	channels := session.opusChannels
	session.mu.Unlock()
	if len(pcm) == 0 || sampleRateHz <= 0 || channels <= 0 {
		return
	}
	durationMS := len(pcm) * 1000 / sampleRateHz / channels
	chunk := protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: sampleRateHz,
		Channels:     channels,
		DurationMS:   durationMS,
		DataBase64:   pcm16Base64(pcm),
	}
	frame := protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  id.deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
		SentAtMS:  s.now().UnixMilli(),
	}
	result := s.audioIngress.Push(audio.Frame{
		DeviceID:     id.deviceID,
		TraceID:      id.traceID,
		SessionID:    id.sessionID,
		Seq:          frame.Seq,
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		DataBase64:   chunk.DataBase64,
	})
	pipelineFrame := providers.VoicePipelinePCMFrame{
		Seq:          frame.Seq,
		Codec:        string(chunk.Codec),
		SampleRateHz: chunk.SampleRateHz,
		Channels:     chunk.Channels,
		DurationMS:   chunk.DurationMS,
		ByteCount:    len(pcm) * 2,
		RMS:          result.RMS,
		PCM16LE:      pcm16Bytes(pcm),
	}
	session.mu.Lock()
	session.voicePipelineFrames = append(session.voicePipelineFrames, pipelineFrame)
	if result.SpeechDetected || result.SpeechActive || containsAudioIngressEvent(result.Events, audio.EventVADSpeechStart) {
		session.voicePipelineHasSpeech = true
	}
	session.mu.Unlock()
	s.appendXiaozhiStreamingASRFrame(ctx, session, pipelineFrame)
	s.recordAudioCaptureFrame(frame, chunk, result, id.traceID, id.sessionID)
	s.metrics.audioIngressFramesTotal.Inc()
	if result.DroppedFrameDelta > 0 {
		s.metrics.audioIngressDroppedTotal.Add(float64(result.DroppedFrameDelta))
	}
	s.metrics.audioIngressBufferDepth.Set(float64(result.BufferedFrames))
	s.metrics.audioIngressRMS.Set(result.RMS)
	s.metrics.vadDetectorDecisions.WithLabelValues(vadDetectorLabel(result.VADDetector), vadDecisionLabel(result.SpeechDetected)).Inc()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "audio.ingress.buffered", s.now().UnixMilli())
	for _, event := range result.Events {
		switch event {
		case audio.EventVADSpeechStart:
			s.metrics.vadSpeechStartTotal.Inc()
		case audio.EventVADSpeechEnd:
			s.metrics.vadSpeechEndTotal.Inc()
		}
		s.recordTrace(id.traceID, id.sessionID, id.deviceID, string(event), s.now().UnixMilli())
	}
	s.maybeAutoStopXiaozhiTurnOnIngress(ctx, conn, session, result.Events)
}

func pcm16Base64(pcm []int16) string {
	data := pcm16Bytes(pcm)
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func pcm16Bytes(pcm []int16) []byte {
	if len(pcm) == 0 {
		return nil
	}
	data := make([]byte, len(pcm)*2)
	for i, sample := range pcm {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return data
}

func (s *Server) recordXiaozhiDeviceSeen(frame xiaozhitransport.Frame) {
	session := &xiaozhiSession{
		deviceID:  frame.DeviceID,
		traceID:   frame.TraceID,
		sessionID: frame.SessionID,
	}
	capabilities := map[string]string(nil)
	if frame.Control != nil && frame.Control.Hello != nil {
		session.features = frame.Control.Hello.Features
		capabilities = s.xiaozhiFeatureCapabilities(frame.Control.Hello.Features, session)
	}
	s.recordXiaozhiDeviceActivity(session, "xiaozhi.hello", capabilities)
}

func (s *Server) recordXiaozhiDeviceActivity(session *xiaozhiSession, event string, capabilities map[string]string) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
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
	if len(capabilities) > 0 {
		record.Capabilities = mergeDeviceCapabilities(record.Capabilities, capabilities)
	}
	record.ConnectionStatus = "online"
	if cleanEvent := strings.TrimSpace(event); cleanEvent != "" {
		record.LastEvent = protocol.DeviceEventKind(cleanEvent)
	}
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	s.devices[id.deviceID] = record
}

func (s *Server) recordDeviceDisplayState(deviceID string, traceID string, sessionID string, source string, rawState string) bool {
	deviceID = strings.TrimSpace(deviceID)
	rawState = strings.TrimSpace(rawState)
	if deviceID == "" || rawState == "" {
		return false
	}
	source = normalizeDisplayStateSource(source)
	state := protocol.NormalizeOfficialDisplayState(rawState)
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	record := s.devices[deviceID]
	if record.DeviceID == "" {
		record.DeviceID = deviceID
		record.FirstSeenMS = nowMS
	}
	if record.IdentityStatus == "" {
		record.IdentityStatus = "unknown"
	}
	changed := record.DisplayState != state ||
		record.DisplayStateSource != source ||
		record.DisplayStateTraceID != traceID ||
		record.DisplayStateSessionID != sessionID
	record.DisplayState = state
	record.DisplayStateSource = source
	record.DisplayStateTraceID = traceID
	record.DisplayStateSessionID = sessionID
	record.DisplayStateUpdatedAtMS = nowMS
	record.DisplayStateAccepted = false
	record.LastTraceID = traceID
	record.LastSessionID = sessionID
	record.LastSeenMS = nowMS
	s.devices[deviceID] = record
	s.mu.Unlock()
	if !changed {
		return false
	}
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.received", nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.normalized", nowMS)
	s.recordTrace(traceID, sessionID, deviceID, "stackchan.display_state.registry_updated", nowMS)
	return true
}

func normalizeDisplayStateSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "xiaozhi", "device_event", "official_stackchan":
		return strings.ToLower(strings.TrimSpace(source))
	default:
		return "unknown"
	}
}

func (s *Server) recordXiaozhiPlaybackStart(session *xiaozhiSession, streamID string) {
	s.recordXiaozhiPlaybackEvent(session, "device.playback.start", streamID)
}

func (s *Server) recordXiaozhiPlaybackStopDone(session *xiaozhiSession, streamID string) {
	s.recordXiaozhiPlaybackEvent(session, "device.playback.stop_done", streamID)
}

func (s *Server) recordXiaozhiHeartbeat(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, "device.heartbeat", nowMS)
	runtimeEcho := xiaozhiHeartbeatRuntimeEcho(event)
	if len(runtimeEcho) == 0 {
		s.recordXiaozhiDeviceActivity(session, "device.heartbeat", nil)
		return
	}
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
	record.LastEvent = protocol.DeviceEventKind("device.heartbeat")
	record.LastTraceID = id.traceID
	record.LastSessionID = id.sessionID
	record.LastSeenMS = nowMS
	record.RuntimeEcho = mergeDeviceCapabilities(record.RuntimeEcho, runtimeEcho)
	s.devices[id.deviceID] = record
}

func xiaozhiHeartbeatRuntimeEcho(event xiaozhitransport.DeviceExtensionEvent) map[string]string {
	if event.Kind != xiaozhitransport.DeviceEventKindHeartbeat || len(event.RuntimeEcho) == 0 {
		return nil
	}
	echo := map[string]string{}
	for key, value := range event.RuntimeEcho {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		echo[key] = value
	}
	if len(echo) == 0 {
		return nil
	}
	return echo
}

func (s *Server) recordXiaozhiTouchEvent(session *xiaozhiSession, event xiaozhitransport.DeviceExtensionEvent) bool {
	id := session.identitySnapshot()
	if strings.TrimSpace(id.deviceID) == "" {
		return false
	}
	deviceEvent, source, ok := xiaozhiTouchEventKindAndSource(event)
	if !ok {
		return false
	}
	nowMS := s.now().UnixMilli()
	traceName := "device." + string(deviceEvent) + ".received"
	s.recordTrace(id.traceID, id.sessionID, id.deviceID, traceName, nowMS)
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
	s.devices[id.deviceID] = record
	s.mu.Unlock()
	return true
}

func xiaozhiTouchEventKindAndSource(event xiaozhitransport.DeviceExtensionEvent) (protocol.DeviceEventKind, protocol.TouchSource, bool) {
	var deviceEvent protocol.DeviceEventKind
	var source protocol.TouchSource
	switch event.Value {
	case "screen_tap":
		deviceEvent = protocol.DeviceEventTouchWakeOrListen
		source = protocol.TouchSourceScreen
	case "screen_barge_in":
		deviceEvent = protocol.DeviceEventTouchBargeIn
		source = protocol.TouchSourceScreen
	case "top_tap":
		deviceEvent = protocol.DeviceEventTouchTopTap
		source = protocol.TouchSourceTopSensor
	case "top_swipe_forward":
		deviceEvent = protocol.DeviceEventTouchTopSwipeForward
		source = protocol.TouchSourceTopSensor
	case "top_swipe_backward":
		deviceEvent = protocol.DeviceEventTouchTopSwipeBackward
		source = protocol.TouchSourceTopSensor
	case "top_barge_in":
		deviceEvent = protocol.DeviceEventTouchBargeIn
		source = protocol.TouchSourceTopSensor
	default:
		return "", "", false
	}
	if event.Source == string(protocol.TouchSourceScreen) {
		source = protocol.TouchSourceScreen
	} else if event.Source == string(protocol.TouchSourceTopSensor) {
		source = protocol.TouchSourceTopSensor
	}
	return deviceEvent, source, true
}

func xiaozhiTouchEventIsBargeIn(event xiaozhitransport.DeviceExtensionEvent) bool {
	switch event.Value {
	case "screen_barge_in", "top_barge_in":
		return true
	default:
		return false
	}
}
