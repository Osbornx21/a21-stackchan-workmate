package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func (s *Server) cancelRealtimeVoiceTurn(ctx context.Context, req RealtimeSessionCancelRequest) ([]realtimeVoiceOutput, error) {
	started := time.Now()
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.start", s.now().UnixMilli())
	providerEvents, err := s.currentVoiceProvider().Cancel(ctx, providers.VoiceCancelRequest{
		Session:  providers.VoiceSession{TraceID: req.TraceID, SessionID: req.SessionID, DeviceID: req.DeviceID},
		Reason:   req.Reason,
		StreamID: req.StreamID,
	})
	s.metrics.voiceProviderCancelMS.Observe(float64(time.Since(started)) / float64(time.Millisecond))
	if err != nil {
		s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.error", s.now().UnixMilli())
		return nil, err
	}
	outputs := make([]realtimeVoiceOutput, 0, 2)
	for event := range providerEvents {
		outputs = append(outputs, realtimeVoiceOutput{Control: voiceEventToControlPayload(event, req.Mode), Audio: event.Audio})
	}
	s.recordTrace(req.TraceID, req.SessionID, req.DeviceID, "provider.cancel.end", s.now().UnixMilli())
	return outputs, nil
}

func (s *Server) handleMockTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
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
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_turn.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockTurnResponse(req))
}

func (s *Server) handleMockInterrupt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MockTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	traceID, sessionID := s.ids(req.TraceID, req.SessionID)
	req.TraceID = traceID
	req.SessionID = sessionID
	s.recordTrace(traceID, sessionID, req.DeviceID, "http.mock_interrupt.received", s.now().UnixMilli())
	writeJSON(w, http.StatusOK, s.mockInterruptResponse(req))
}

type xiaozhiSession struct {
	mu                             sync.Mutex
	writeMu                        sync.Mutex
	traceID                        string
	sessionID                      string
	deviceID                       string
	features                       xiaozhitransport.HelloFeatures
	currentTurn                    *xiaozhiTurn
	nextTurnID                     uint64
	helloReceived                  bool
	listening                      bool
	listenStartedAtMS              int64
	suppressedListenActive         bool
	binaryProtocolVersion          int
	opusCodec                      *opuscodec.Codec
	opusSampleRateHz               int
	opusChannels                   int
	opusFrameDurationMS            int
	opusFrameCount                 int
	opusByteCount                  int
	opusDecodedFrameCount          int
	opusDecodedSampleCount         int
	opusDecodeErrorCount           int
	opusIngressProcessedFrameCount int
	voicePipelineFrames            []providers.VoicePipelinePCMFrame
	voicePipelineHasSpeech         bool
	ttsStopSent                    bool
	lastDownlinkAtMS               int64
	lastDownlinkTurnID             string
	lastPlaybackStopDoneAtMS       int64
	inputCooldownUntilMS           int64
	inputCooldownReason            string
	officialStackChanState         string
	xiaozhiStateReactionState      string
	streamingASRSession            providers.StreamingASRSession
	streamingASRHasPartial         bool
	streamingASRPartialText        string
	streamingASRHasFinal           bool
	streamingASRFinalText          string
	streamingASRClosed             bool
	streamingASRCommitStarted      bool
	streamingASRAnswerStarted      bool
	wakePrerollFrames              []providers.VoicePipelinePCMFrame
	wakePrerollPayloadBytes        []int
	wakePrerollHasSpeech           bool
	opusIngressQueue               chan xiaozhiOpusIngressFrame
	opusIngressCtx                 context.Context
	opusIngressCancel              context.CancelFunc
}

type xiaozhiTurn struct {
	id           uint64
	ctx          context.Context
	cancel       context.CancelCauseFunc
	pacer        *audio.AudioRateController
	downlink     *xiaozhiTurnDownlinkCodec
	mode         protocol.Mode
	cancelReason string
}

type xiaozhiTurnDownlinkCodec struct {
	sampleRateHz int
	channels     int
	durationMS   int
	codec        *opuscodec.Codec
}

type xiaozhiOpusIngressFrame struct {
	ctx   context.Context
	conn  *websocket.Conn
	frame xiaozhitransport.Frame
	turn  *xiaozhiTurn
	seq   uint64
}

type xiaozhiVoicePipelineRunner interface {
	Run(context.Context, providers.VoicePipelineRequest) (providers.VoicePipelineResult, error)
}

type xiaozhiVoicePipelineStreamer interface {
	RunStream(context.Context, providers.VoicePipelineRequest) (<-chan providers.VoicePipelineStreamEvent, error)
}

type xiaozhiTurnTask struct {
	turn                      *xiaozhiTurn
	turnID                    string
	traceID                   string
	sessionID                 string
	deviceID                  string
	audioIngressBase          map[string]any
	voicePipelineFrames       []providers.VoicePipelinePCMFrame
	voicePipelineHasSpeech    bool
	streamingASRPartialText   string
	streamingASRPartialDriven bool
	streamingASRFinalText     string
	streamingASRUsed          bool
	mode                      protocol.Mode
}

type xiaozhiSessionIdentity struct {
	deviceID  string
	traceID   string
	sessionID string
}

func defaultXiaozhiVoicePipelineRunner() xiaozhiVoicePipelineRunner {
	return providers.NewVoicePipelineRunner(providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
	})
}

func (session *xiaozhiSession) identitySnapshot() xiaozhiSessionIdentity {
	if session == nil {
		return xiaozhiSessionIdentity{}
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.identityLocked()
}

func (session *xiaozhiSession) identityLocked() xiaozhiSessionIdentity {
	return xiaozhiSessionIdentity{
		deviceID:  session.deviceID,
		traceID:   session.traceID,
		sessionID: session.sessionID,
	}
}

func (session *xiaozhiSession) identity() xiaozhitransport.Identity {
	id := session.identitySnapshot()
	return xiaozhitransport.Identity{
		DeviceID:  id.deviceID,
		TraceID:   id.traceID,
		SessionID: id.sessionID,
	}
}

func (session *xiaozhiSession) adoptFrame(frame xiaozhitransport.Frame) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.deviceID = frame.DeviceID
	session.traceID = frame.TraceID
	session.sessionID = frame.SessionID
}

func (session *xiaozhiSession) featuresSnapshot() xiaozhitransport.HelloFeatures {
	if session == nil {
		return xiaozhitransport.HelloFeatures{}
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.features
}

func (session *xiaozhiSession) helloReceivedSnapshot() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.helloReceived
}

func (session *xiaozhiSession) binaryProtocolVersionSnapshot() int {
	if session == nil {
		return 1
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.binaryProtocolVersion
}

func (session *xiaozhiSession) setXiaozhiListening(listening bool, startedAtMS int64) {
	if session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.listening = listening
	if listening {
		session.listenStartedAtMS = startedAtMS
	} else {
		session.listenStartedAtMS = 0
	}
}

func (session *xiaozhiSession) xiaozhiListening() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.listening
}

func (session *xiaozhiSession) startXiaozhiTurn(parent context.Context, mode protocol.Mode) *xiaozhiTurn {
	if parent == nil {
		parent = context.Background()
	}
	if mode == "" {
		mode = protocol.ModeWorkmate
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.cancelCurrentXiaozhiTurnLocked("new_turn")
	session.nextTurnID++
	ctx, cancel := context.WithCancelCause(parent)
	turn := &xiaozhiTurn{
		id:     session.nextTurnID,
		ctx:    ctx,
		cancel: cancel,
		mode:   mode,
		pacer: audio.NewAudioRateController(audio.AudioRateControllerConfig{
			FrameDuration:   60 * time.Millisecond,
			PrebufferFrames: 1,
		}),
	}
	session.currentTurn = turn
	return turn
}

func (session *xiaozhiSession) setCurrentXiaozhiTurnMode(mode protocol.Mode) {
	if mode == "" {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn != nil {
		session.currentTurn.mode = mode
	}
}

func (session *xiaozhiSession) currentXiaozhiTurn() *xiaozhiTurn {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.currentTurn
}

func (session *xiaozhiSession) currentXiaozhiTurnID() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return xiaozhiTurnID(session.currentTurn)
}

func (session *xiaozhiSession) cancelCurrentXiaozhiTurn(reason string) *xiaozhiTurn {
	session.mu.Lock()
	turn := session.cancelCurrentXiaozhiTurnLocked(reason)
	session.mu.Unlock()
	return turn
}

func (session *xiaozhiSession) cancelCurrentXiaozhiTurnLocked(reason string) *xiaozhiTurn {
	if session.currentTurn == nil {
		return nil
	}
	turn := session.currentTurn
	turn.cancelReason = strings.TrimSpace(reason)
	turn.cancel(xiaozhiTurnCancelCause(reason))
	session.currentTurn = nil
	return turn
}

func (session *xiaozhiSession) prepareXiaozhiListenStartBargeIn(reason string, nowMS int64, recentWindowMS int64) (xiaozhiTurnTask, bool) {
	session.mu.Lock()
	if session.ttsStopSent {
		session.currentTurn = nil
		session.mu.Unlock()
		return xiaozhiTurnTask{}, false
	}
	turn := session.cancelCurrentXiaozhiTurnLocked(reason)
	turnID := xiaozhiTurnID(turn)
	if turnID == "" && session.lastDownlinkTurnID != "" && nowMS-session.lastDownlinkAtMS >= 0 && nowMS-session.lastDownlinkAtMS <= recentWindowMS {
		if session.lastPlaybackStopDoneAtMS >= session.lastDownlinkAtMS && session.lastPlaybackStopDoneAtMS <= nowMS {
			session.mu.Unlock()
			return xiaozhiTurnTask{}, false
		}
		turnID = session.lastDownlinkTurnID
	}
	if turnID == "" {
		session.mu.Unlock()
		return xiaozhiTurnTask{}, false
	}
	task := xiaozhiTurnTask{
		turn:      turn,
		turnID:    turnID,
		traceID:   session.traceID,
		sessionID: session.sessionID,
		deviceID:  session.deviceID,
		mode:      xiaozhiTurnMode(turn),
	}
	session.mu.Unlock()
	return task, true
}

func (session *xiaozhiSession) suppressXiaozhiInputUntil(untilMS int64, reason string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if untilMS > session.inputCooldownUntilMS {
		session.inputCooldownUntilMS = untilMS
		session.inputCooldownReason = safeGatewayFallbackToken(reason, "cooldown")
	}
}

func (session *xiaozhiSession) xiaozhiInputSuppression(nowMS int64) (bool, string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.inputCooldownUntilMS > 0 && nowMS >= 0 && nowMS < session.inputCooldownUntilMS {
		return true, firstNonEmpty(session.inputCooldownReason, "cooldown")
	}
	return false, ""
}

func (session *xiaozhiSession) xiaozhiSuppressedListenActive() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.suppressedListenActive
}

func (session *xiaozhiSession) clearSuppressedXiaozhiListen() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.suppressedListenActive {
		return false
	}
	session.suppressedListenActive = false
	return true
}

func (session *xiaozhiSession) cancelXiaozhiTurnContext(turn *xiaozhiTurn, reason string) {
	if turn == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	turn.cancelReason = strings.TrimSpace(reason)
	turn.cancel(xiaozhiTurnCancelCause(reason))
	if turn.pacer != nil {
		turn.pacer.Reset()
	}
}

func (session *xiaozhiSession) xiaozhiTurnCancelReason(turn *xiaozhiTurn) string {
	if session == nil || turn == nil {
		return ""
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return strings.TrimSpace(turn.cancelReason)
}

func (session *xiaozhiSession) shouldAbortXiaozhiTurn(turn *xiaozhiTurn) bool {
	if turn == nil {
		return true
	}
	session.mu.Lock()
	currentTurn := session.currentTurn
	session.mu.Unlock()
	return currentTurn != turn || turn.ctx.Err() != nil
}

func xiaozhiUserInterruptReason(reason string) (string, bool) {
	reason = safeGatewayFallbackToken(reason, "")
	if reason == "" {
		return "", false
	}
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "error") || strings.Contains(lower, "unavailable") {
		return "", false
	}
	if strings.Contains(lower, "barge") ||
		strings.Contains(lower, "wake") ||
		strings.Contains(lower, "abort") ||
		strings.Contains(lower, "interrupt") {
		return reason, true
	}
	return "", false
}

func (session *xiaozhiSession) completeXiaozhiTurn(turn *xiaozhiTurn) {
	if turn == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn == turn {
		session.currentTurn = nil
	}
}

func xiaozhiTurnID(turn *xiaozhiTurn) string {
	if turn == nil || turn.id == 0 {
		return ""
	}
	return fmt.Sprintf("a21-xiaozhi-turn-%06d", turn.id)
}

func xiaozhiTurnMode(turn *xiaozhiTurn) protocol.Mode {
	if turn == nil || turn.mode == "" {
		return protocol.ModeWorkmate
	}
	return turn.mode
}

func (session *xiaozhiSession) resetXiaozhiTTSStop() {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.ttsStopSent = false
}

func (session *xiaozhiSession) claimXiaozhiTTSStop() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.ttsStopSent {
		return false
	}
	session.ttsStopSent = true
	return true
}

func (session *xiaozhiSession) claimOfficialStackChanState(state string, force bool) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !force && session.officialStackChanState == state {
		return false
	}
	session.officialStackChanState = state
	return true
}

func (session *xiaozhiSession) claimXiaozhiStateReaction(state string, force bool) bool {
	state = strings.TrimSpace(state)
	if state == "" {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !force && session.xiaozhiStateReactionState == state {
		return false
	}
	session.xiaozhiStateReactionState = state
	return true
}

func (session *xiaozhiSession) markXiaozhiDownlink(turn *xiaozhiTurn, atMS int64) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.lastDownlinkAtMS = atMS
	session.lastDownlinkTurnID = xiaozhiTurnID(turn)
}

func xiaozhiTurnCancelCause(reason string) error {
	if xiaozhiAbortIsBargeIn(reason) {
		return providers.ErrVoicePipelineBargeIn
	}
	return context.Canceled
}

func xiaozhiAbortIsBargeIn(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	return reason == "" || strings.Contains(reason, "barge") || strings.Contains(reason, "wake")
}

func (session *xiaozhiSession) writeXiaozhiJSON(ctx context.Context, conn *websocket.Conn, turn *xiaozhiTurn, value any) error {
	if conn == nil {
		return fmt.Errorf("xiaozhi websocket is nil")
	}
	session.writeMu.Lock()
	defer session.writeMu.Unlock()
	if turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		return context.Canceled
	}
	return wsjson.Write(ctx, conn, value)
}

func (session *xiaozhiSession) writeXiaozhiBinary(ctx context.Context, conn *websocket.Conn, turn *xiaozhiTurn, frame []byte) error {
	if conn == nil {
		return fmt.Errorf("xiaozhi websocket is nil")
	}
	session.writeMu.Lock()
	defer session.writeMu.Unlock()
	if turn != nil && session.shouldAbortXiaozhiTurn(turn) {
		return context.Canceled
	}
	return conn.Write(ctx, websocket.MessageBinary, frame)
}

func (session *xiaozhiSession) configureXiaozhiAudio(params xiaozhitransport.AudioParams) error {
	codec, err := opuscodec.New(params.SampleRate, params.Channels, params.FrameDuration)
	if err != nil {
		return err
	}
	session.mu.Lock()
	session.opusCodec = codec
	session.opusSampleRateHz = params.SampleRate
	session.opusChannels = params.Channels
	session.opusFrameDurationMS = params.FrameDuration
	stream, shouldCancel := session.resetXiaozhiOpusIngressLocked()
	session.mu.Unlock()
	if shouldCancel {
		stream.Cancel(context.Canceled)
	}
	return nil
}

func (session *xiaozhiSession) resetXiaozhiOpusIngress() {
	session.mu.Lock()
	stream, shouldCancel := session.resetXiaozhiOpusIngressLocked()
	session.mu.Unlock()
	if shouldCancel {
		stream.Cancel(context.Canceled)
	}
}

func (session *xiaozhiSession) resetXiaozhiOpusIngressLocked() (providers.StreamingASRSession, bool) {
	session.opusFrameCount = 0
	session.opusByteCount = 0
	session.opusDecodedFrameCount = 0
	session.opusDecodedSampleCount = 0
	session.opusDecodeErrorCount = 0
	session.opusIngressProcessedFrameCount = 0
	session.voicePipelineFrames = nil
	session.voicePipelineHasSpeech = false
	stream := session.streamingASRSession
	shouldCancel := stream != nil && !session.streamingASRClosed
	if session.streamingASRSession != nil && !session.streamingASRClosed {
		session.streamingASRClosed = true
	}
	session.streamingASRSession = nil
	session.streamingASRHasPartial = false
	session.streamingASRPartialText = ""
	session.streamingASRHasFinal = false
	session.streamingASRFinalText = ""
	session.streamingASRClosed = false
	session.streamingASRAnswerStarted = false
	return stream, shouldCancel
}

func (session *xiaozhiSession) xiaozhiOpusIngressQueue() chan xiaozhiOpusIngressFrame {
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.opusIngressQueue
}

func (session *xiaozhiSession) resetXiaozhiWakePreroll() {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.wakePrerollFrames = nil
	session.wakePrerollPayloadBytes = nil
	session.wakePrerollHasSpeech = false
}

func (session *xiaozhiSession) shouldBufferXiaozhiWakePreroll(nowMS int64) bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.currentTurn != nil {
		return false
	}
	if session.inputCooldownUntilMS > 0 && nowMS >= 0 && nowMS < session.inputCooldownUntilMS {
		return false
	}
	return true
}

func (session *xiaozhiSession) xiaozhiOpusDecodeStatus() string {
	if session == nil {
		return XiaozhiOpusDecodeErrorState
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return xiaozhiOpusDecodeStatusFromCounts(session.opusDecodedFrameCount, session.opusDecodeErrorCount)
}
