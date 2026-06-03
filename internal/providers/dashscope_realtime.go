package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	dashscopeRealtimeDefaultURL = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	dashscopeASRDefaultModel    = "qwen3-asr-flash-realtime"
	dashscopeTTSDefaultModel    = "qwen3-tts-flash-realtime"
	dashscopeTTSReadyTimeout    = 200 * time.Millisecond
)

var dashScopeRealtimeEventSeq atomic.Uint64

type DashScopeRealtimeASRAdapterOptions struct {
	Name   string
	Env    []string
	Dialer RealtimeDialer
}

type DashScopeRealtimeTTSAdapterOptions struct {
	Name   string
	Env    []string
	Dialer RealtimeDialer
}

type dashScopeRealtimeASRAdapter struct {
	name   string
	env    []string
	dialer RealtimeDialer
}

type dashScopeRealtimeTTSAdapter struct {
	name   string
	env    []string
	dialer RealtimeDialer
}

type dashScopeRealtimeASRSession struct {
	session *RealtimeWebSocketSession
	events  chan ASRAdapterEvent
	done    chan struct{}
}

func NewDashScopeRealtimeASRAdapter(options DashScopeRealtimeASRAdapterOptions) ASRAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "dashscope_qwen_asr_realtime"
	}
	return &dashScopeRealtimeASRAdapter{name: name, env: append([]string(nil), options.Env...), dialer: options.Dialer}
}

func NewDashScopeRealtimeTTSAdapter(options DashScopeRealtimeTTSAdapterOptions) TTSAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "dashscope_qwen_tts_realtime"
	}
	return &dashScopeRealtimeTTSAdapter{name: name, env: append([]string(nil), options.Env...), dialer: options.Dialer}
}

func (a *dashScopeRealtimeASRAdapter) Name() string { return a.name }

func (a *dashScopeRealtimeTTSAdapter) Name() string { return a.name }

func (a *dashScopeRealtimeTTSAdapter) StreamingTTSAdapter() {}

func (a *dashScopeRealtimeASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	stream, err := a.StartStreamingASR(ctx, StreamingASRStartRequest{Session: req.Session, Mode: req.Mode})
	if err != nil {
		return nil, err
	}
	go func() {
		for _, frame := range req.Frames {
			if err := stream.AppendFrame(ctx, frame); err != nil {
				stream.Cancel(err)
				return
			}
		}
		if err := stream.Commit(ctx); err != nil {
			stream.Cancel(err)
		}
	}()
	return stream.Events(), nil
}

func (a *dashScopeRealtimeASRAdapter) StartStreamingASR(ctx context.Context, _ StreamingASRStartRequest) (StreamingASRSession, error) {
	key := strings.TrimSpace(envValue(a.env, "A21_DASHSCOPE_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("dashscope realtime ASR provider missing A21_DASHSCOPE_API_KEY")
	}
	model := firstNonEmptyPipelineValue(envValue(a.env, "A21_DASHSCOPE_ASR_MODEL"), dashscopeASRDefaultModel)
	endpoint, err := dashScopeRealtimeURL(model, envValue(a.env, "A21_DASHSCOPE_ASR_REALTIME_URL"))
	if err != nil {
		return nil, err
	}
	policy, _ := NetworkPolicyFromEnv(a.env)
	dialer := a.dialer
	if dialer == nil {
		dialer = coderRealtimeDialer{}
	}
	conn, err := dialer.Dial(ctx, endpoint, dashScopeRealtimeHeaders(key), policy)
	if err != nil {
		return nil, err
	}
	session := &RealtimeWebSocketSession{provider: a.name, conn: conn}
	if err := session.conn.WriteJSON(ctx, map[string]any{
		"event_id": dashScopeRealtimeEventID("asr_session_update"),
		"type":     "session.update",
		"session": map[string]any{
			"modalities":         []string{"text"},
			"input_audio_format": "pcm",
			"sample_rate":        16000,
			"input_audio_transcription": map[string]any{
				"model":    model,
				"language": "zh",
			},
			"turn_detection": nil,
		},
	}); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime ASR session update failed")
	}
	stream := &dashScopeRealtimeASRSession{session: session, events: make(chan ASRAdapterEvent, 8), done: make(chan struct{})}
	go stream.readLoop(ctx)
	return stream, nil
}

func (s *dashScopeRealtimeASRSession) AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error {
	if len(frame.PCM16LE) == 0 {
		return nil
	}
	return s.session.conn.WriteJSON(ctx, map[string]any{
		"event_id": dashScopeRealtimeEventID("asr_audio_append"),
		"type":     "input_audio_buffer.append",
		"audio":    base64.StdEncoding.EncodeToString(frame.PCM16LE),
	})
}

func (s *dashScopeRealtimeASRSession) Events() <-chan ASRAdapterEvent { return s.events }

func (s *dashScopeRealtimeASRSession) Commit(ctx context.Context) error {
	if err := s.session.conn.WriteJSON(ctx, map[string]any{"event_id": dashScopeRealtimeEventID("asr_audio_commit"), "type": "input_audio_buffer.commit"}); err != nil {
		return err
	}
	return s.session.conn.WriteJSON(ctx, map[string]any{"event_id": dashScopeRealtimeEventID("asr_session_finish"), "type": "session.finish"})
}

func (s *dashScopeRealtimeASRSession) Cancel(err error) {
	_ = s.session.Close(context.Background())
}

func (s *dashScopeRealtimeASRSession) readLoop(ctx context.Context) {
	defer close(s.done)
	defer close(s.events)
	defer s.session.Close(context.Background())
	for {
		raw, err := s.session.ReadEvent(ctx)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				s.events <- ASRAdapterEvent{Finding: "dashscope realtime ASR read failed", Err: fmt.Errorf("dashscope realtime ASR read failed")}
			}
			return
		}
		eventType, _ := raw["type"].(string)
		switch eventType {
		case "conversation.item.input_audio_transcription.text":
			if text := firstStringField(raw, "text", "transcript", "delta"); text != "" {
				s.events <- ASRAdapterEvent{Text: text}
			}
		case "conversation.item.input_audio_transcription.completed":
			text := firstStringField(raw, "transcript", "text", "delta")
			if text != "" {
				s.events <- ASRAdapterEvent{Text: text, Final: true}
			}
			return
		case "session.finished":
			return
		case "error":
			s.events <- ASRAdapterEvent{Finding: "dashscope realtime ASR provider error", Err: fmt.Errorf("dashscope realtime ASR provider error")}
			return
		}
	}
}

func (a *dashScopeRealtimeTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	key := strings.TrimSpace(envValue(a.env, "A21_DASHSCOPE_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("dashscope realtime TTS provider missing A21_DASHSCOPE_API_KEY")
	}
	model := firstNonEmptyPipelineValue(envValue(a.env, "A21_DASHSCOPE_TTS_MODEL"), envValue(a.env, "A21_BAILIAN_QWEN_TTS_MODEL"), dashscopeTTSDefaultModel)
	endpoint, err := dashScopeRealtimeURL(model, envValue(a.env, "A21_DASHSCOPE_TTS_REALTIME_URL"))
	if err != nil {
		return nil, err
	}
	policy, _ := NetworkPolicyFromEnv(a.env)
	dialer := a.dialer
	if dialer == nil {
		dialer = coderRealtimeDialer{}
	}
	conn, err := dialer.Dial(ctx, endpoint, dashScopeRealtimeHeaders(key), policy)
	if err != nil {
		return nil, err
	}
	session := &RealtimeWebSocketSession{provider: a.name, conn: conn}
	out := make(chan VoiceAudioChunk, 4)
	ready := make(chan error, 1)
	commitResult := make(chan bool, 1)
	go dashScopeRealtimeTTSReadLoop(ctx, session, out, ready, commitResult)
	if err := session.conn.WriteJSON(ctx, dashScopeTTSUpdateEvent(a.env)); err != nil {
		signalDashScopeTTSCommitResult(commitResult, false)
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS session update failed")
	}
	select {
	case err := <-ready:
		if err != nil {
			signalDashScopeTTSCommitResult(commitResult, false)
			_ = conn.Close(ctx)
			return nil, fmt.Errorf("dashscope realtime TTS session update failed")
		}
	case <-ctx.Done():
		signalDashScopeTTSCommitResult(commitResult, false)
		_ = conn.Close(context.Background())
		return nil, ctx.Err()
	case <-time.After(dashscopeTTSReadyTimeout):
	}
	if err := session.conn.WriteJSON(ctx, map[string]any{"event_id": dashScopeRealtimeEventID("tts_text_append"), "type": "input_text_buffer.append", "text": req.Text}); err != nil {
		signalDashScopeTTSCommitResult(commitResult, false)
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS text append failed")
	}
	if err := session.conn.WriteJSON(ctx, map[string]any{"event_id": dashScopeRealtimeEventID("tts_text_commit"), "type": "input_text_buffer.commit"}); err != nil {
		signalDashScopeTTSCommitResult(commitResult, false)
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS text commit failed")
	}
	signalDashScopeTTSCommitResult(commitResult, true)
	return out, nil
}

func dashScopeRealtimeTTSReadLoop(ctx context.Context, session *RealtimeWebSocketSession, out chan<- VoiceAudioChunk, ready chan<- error, commitResult <-chan bool) {
	defer close(out)
	defer session.Close(context.Background())
	chunker := newPCM16Mono60MSChunker(24000)
	audioSeen := false
	readySent := false
	signalReady := func(err error) {
		if readySent {
			return
		}
		readySent = true
		select {
		case ready <- err:
		default:
		}
	}
	for {
		raw, err := session.ReadEvent(ctx)
		if err != nil {
			if err == io.EOF {
				signalReady(fmt.Errorf("dashscope realtime TTS ended before session update"))
				for _, chunk := range chunker.Flush() {
					audioSeen = true
					if !sendVoiceAudioChunk(ctx, out, chunk) {
						return
					}
				}
				if !audioSeen && ctx.Err() == nil {
					sendDashScopeTTSError(ctx, out, "tts adapter no audio", "dashscope realtime TTS produced no audio")
				}
				return
			}
			if ctx.Err() == nil {
				signalReady(fmt.Errorf("dashscope realtime TTS read failed"))
				sendDashScopeTTSError(ctx, out, "tts adapter read failed", "dashscope realtime TTS read failed")
			}
			return
		}
		eventType, _ := raw["type"].(string)
		switch eventType {
		case "session.created":
			continue
		case "session.updated":
			signalReady(nil)
		case "response.created":
			signalReady(nil)
		case "response.audio.delta":
			signalReady(nil)
			chunks, err := chunkDashScopeAudio(chunker, raw)
			if err != nil {
				sendDashScopeTTSError(ctx, out, "tts adapter invalid audio delta", "dashscope realtime TTS audio delta invalid")
				return
			}
			for _, chunk := range chunks {
				audioSeen = true
				if !sendVoiceAudioChunk(ctx, out, chunk) {
					return
				}
			}
		case "response.audio.done":
			signalReady(nil)
		case "response.done":
			signalReady(nil)
			for _, chunk := range chunker.Flush() {
				audioSeen = true
				if !sendVoiceAudioChunk(ctx, out, chunk) {
					return
				}
			}
			if !audioSeen {
				sendDashScopeTTSError(ctx, out, "tts adapter no audio", "dashscope realtime TTS produced no audio")
				return
			}
			if ctx.Err() == nil && waitDashScopeTTSCommitted(ctx, commitResult) {
				_ = session.conn.WriteJSON(ctx, map[string]any{"event_id": dashScopeRealtimeEventID("tts_session_finish"), "type": "session.finish"})
			}
			return
		case "session.finished":
			signalReady(nil)
			for _, chunk := range chunker.Flush() {
				audioSeen = true
				if !sendVoiceAudioChunk(ctx, out, chunk) {
					return
				}
			}
			if !audioSeen {
				sendDashScopeTTSError(ctx, out, "tts adapter no audio", "dashscope realtime TTS produced no audio")
			}
			return
		case "error":
			err := fmt.Errorf("dashscope realtime TTS provider error")
			signalReady(err)
			sendDashScopeTTSError(ctx, out, "tts adapter provider error", "dashscope realtime TTS provider error")
			return
		}
	}
}

func signalDashScopeTTSCommitResult(commitResult chan<- bool, ok bool) {
	select {
	case commitResult <- ok:
	default:
	}
}

func waitDashScopeTTSCommitted(ctx context.Context, commitResult <-chan bool) bool {
	select {
	case ok := <-commitResult:
		return ok
	case <-ctx.Done():
		return false
	}
}

func dashScopeTTSUpdateEvent(env []string) map[string]any {
	session := map[string]any{
		"mode":            "commit",
		"response_format": "pcm",
		"sample_rate":     24000,
	}
	if voice := strings.TrimSpace(firstNonEmptyPipelineValue(envValue(env, "A21_DASHSCOPE_TTS_VOICE"), envValue(env, "A21_BAILIAN_QWEN_TTS_VOICE_ID"))); voice != "" {
		session["voice"] = voice
	}
	return map[string]any{"event_id": dashScopeRealtimeEventID("tts_session_update"), "type": "session.update", "session": session}
}

func dashScopeRealtimeEventID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "event"
	}
	return "a21_" + prefix + "_" + strconv.FormatUint(dashScopeRealtimeEventSeq.Add(1), 10)
}

func chunkDashScopeAudio(chunker *pcm16Mono60MSChunker, raw map[string]any) ([]VoiceAudioChunk, error) {
	chunks, err := chunker.AppendBase64(firstStringField(raw, "delta"))
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

func sendDashScopeTTSError(ctx context.Context, out chan<- VoiceAudioChunk, finding string, message string) bool {
	finding = strings.TrimSpace(finding)
	if finding == "" {
		finding = "tts adapter failed"
	}
	return sendVoiceAudioChunk(ctx, out, VoiceAudioChunk{
		Finding: finding,
		Err:     fmt.Errorf("%s", message),
	})
}

func dashScopeRealtimeURL(model string, override string) (string, error) {
	raw := strings.TrimSpace(override)
	if raw == "" {
		raw = dashscopeRealtimeDefaultURL
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("model", strings.TrimSpace(model))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func dashScopeRealtimeHeaders(key string) http.Header {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
	header.Set("OpenAI-Beta", "realtime=v1")
	return header
}
