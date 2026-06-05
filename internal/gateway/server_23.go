package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func (s *Server) mockAudioPlaybackChunk(frame protocol.Envelope, traceID string, sessionID string, streamID string, seq uint64) protocol.Envelope {
	payload := protocol.AudioPlaybackChunk{
		StreamID:     streamID,
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   mockAudioPlaybackBase64(frame.DeviceID, 16000, 20),
	}
	data, _ := json.Marshal(payload)
	sentAt := s.now().UnixMilli()
	s.metrics.audioPlaybackChunkTotal.Inc()
	s.recordTrace(traceID, sessionID, frame.DeviceID, "audio.playback.chunk.sent", sentAt)
	return protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  frame.DeviceID,
		Kind:      protocol.KindAudioPlaybackChunk,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		SentAtMS:  sentAt,
		Payload:   data,
	}
}

func mockAudioPlaybackBase64(deviceID string, sampleRateHz int, durationMS int) string {
	if physicalStackChanDeviceID(deviceID) {
		return mockPCM16SquareWaveBase64(sampleRateHz, durationMS)
	}
	return mockPCM16SilenceBase64(sampleRateHz, durationMS)
}

func physicalStackChanDeviceID(deviceID string) bool {
	return strings.HasPrefix(deviceID, "stackchan-") &&
		!strings.HasPrefix(deviceID, "stackchan-sim-") &&
		!strings.HasPrefix(deviceID, "stackchan-bench-")
}

func mockPCM16SilenceBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	byteCount := sampleRateHz * durationMS * 2 / 1000
	return base64.StdEncoding.EncodeToString(make([]byte, byteCount))
}

func mockPCM16SquareWaveBase64(sampleRateHz int, durationMS int) string {
	if sampleRateHz <= 0 || durationMS <= 0 {
		return ""
	}
	samples := sampleRateHz * durationMS / 1000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(9000)
		if (i/8)%2 == 1 {
			sample = -9000
		}
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Server) setActiveStream(traceID string, sessionID string, deviceID string, streamID string) {
	if streamID == "" {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeStreams[key] = streamID
}

func (s *Server) activeStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeStreams[key]
}

func (s *Server) clearActiveStream(traceID string, sessionID string, deviceID string) string {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	streamID := s.activeStreams[key]
	delete(s.activeStreams, key)
	return streamID
}

func (s *Server) setAudioProbeOnly(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.audioProbeSessions[key] = true
		return
	}
	delete(s.audioProbeSessions, key)
}

func (s *Server) audioProbeOnly(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioProbeSessions[key]
}

func (s *Server) setMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string, enabled bool, mockAudioChunks *int) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		chunks := 1
		if mockAudioChunks != nil {
			chunks = *mockAudioChunks
		}
		s.mockPlaybackArmedSessions[key] = chunks
		return
	}
	delete(s.mockPlaybackArmedSessions, key)
}

func (s *Server) consumeMockPlaybackOnNextAudioFrame(deviceID string, traceID string, sessionID string) (int, bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	chunks, armed := s.mockPlaybackArmedSessions[key]
	if !armed {
		return 0, false
	}
	delete(s.mockPlaybackArmedSessions, key)
	return chunks, true
}

func (s *Server) setRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string, enabled bool) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.realtimeArmedSessions[key] = true
		return
	}
	delete(s.realtimeArmedSessions, key)
}

func (s *Server) consumeRealtimeOnNextSpeech(deviceID string, traceID string, sessionID string) bool {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.realtimeArmedSessions[key] {
		return false
	}
	delete(s.realtimeArmedSessions, key)
	return true
}

func (s *Server) realtimeAudioSession(traceID string, sessionID string, deviceID string) providers.RealtimeVoiceSession {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realtimeAudio[key]
}

func (s *Server) setRealtimeAudioSession(traceID string, sessionID string, deviceID string, session providers.RealtimeVoiceSession) {
	if session == nil {
		return
	}
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudio[key] = session
}

func (s *Server) markRealtimeAudioCommit(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realtimeAudioCommitAt[key] = at
	delete(s.realtimeAudioFirstDownlink, key)
}

func (s *Server) observeRealtimeFirstAudioDownlink(traceID string, sessionID string, deviceID string, at time.Time) {
	key := streamStateKey(traceID, sessionID, deviceID)
	s.mu.Lock()
	commitAt, hasCommit := s.realtimeAudioCommitAt[key]
	alreadyObserved := s.realtimeAudioFirstDownlink[key]
	if hasCommit && !alreadyObserved {
		s.realtimeAudioFirstDownlink[key] = true
	}
	s.mu.Unlock()
	if !hasCommit || alreadyObserved {
		return
	}
	durationMS := float64(at.Sub(commitAt)) / float64(time.Millisecond)
	if durationMS < 0 {
		durationMS = 0
	}
	s.metrics.realtimeFirstAudioMS.Observe(durationMS)
	s.recordTrace(traceID, sessionID, deviceID, "provider.audio.first_downlink", at.UnixMilli())
}

func (s *Server) closeRealtimeAudioSessions(ctx context.Context, keys map[string]struct{}) {
	for key := range keys {
		session := s.clearRealtimeAudioSessionByKey(key)
		if session != nil {
			_ = session.Close(ctx)
		}
	}
}

func (s *Server) clearRealtimeAudioSessionByKey(key string) providers.RealtimeVoiceSession {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.realtimeAudio[key]
	delete(s.realtimeAudio, key)
	delete(s.realtimeAudioCommitAt, key)
	delete(s.realtimeAudioFirstDownlink, key)
	return session
}

func (s *Server) mockAudioStreamID(frame protocol.Envelope, traceID string, sessionID string) string {
	streamSeq := frame.Seq
	if streamSeq == 0 {
		streamSeq = 1
	}
	key := traceID
	if key == "" {
		key = sessionID
	}
	if key == "" {
		return fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.audioStreams[key]; existing != "" {
		return existing
	}
	streamID := fmt.Sprintf("a21-audio-stream-%06d", streamSeq)
	s.audioStreams[key] = streamID
	return streamID
}

func streamStateKey(traceID string, sessionID string, deviceID string) string {
	switch {
	case sessionID != "":
		return sessionID
	case traceID != "":
		return traceID
	case deviceID != "":
		return deviceID
	default:
		return "a21-audio-stream-default"
	}
}

func writeAudioEnvelope(ctx context.Context, conn *websocket.Conn, mu *sync.Mutex, event protocol.Envelope) error {
	mu.Lock()
	defer mu.Unlock()
	return wsjson.Write(ctx, conn, event)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(value)
}
