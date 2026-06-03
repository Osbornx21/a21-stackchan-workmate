package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	dashscopeRealtimeDefaultURL = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	dashscopeASRDefaultModel    = "qwen3-asr-flash-realtime"
	dashscopeTTSDefaultModel    = "qwen3-tts-flash-realtime"
)

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
		"type": "session.update",
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
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(frame.PCM16LE),
	})
}

func (s *dashScopeRealtimeASRSession) Events() <-chan ASRAdapterEvent { return s.events }

func (s *dashScopeRealtimeASRSession) Commit(ctx context.Context) error {
	if err := s.session.conn.WriteJSON(ctx, map[string]any{"type": "input_audio_buffer.commit"}); err != nil {
		return err
	}
	return s.session.conn.WriteJSON(ctx, map[string]any{"type": "session.finish"})
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
	if err := session.conn.WriteJSON(ctx, dashScopeTTSUpdateEvent(a.env)); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS session update failed")
	}
	if err := session.conn.WriteJSON(ctx, map[string]any{"type": "input_text_buffer.append", "text": req.Text}); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS text append failed")
	}
	if err := session.conn.WriteJSON(ctx, map[string]any{"type": "input_text_buffer.commit"}); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS text commit failed")
	}
	if err := session.conn.WriteJSON(ctx, map[string]any{"type": "session.finish"}); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("dashscope realtime TTS session finish failed")
	}
	out := make(chan VoiceAudioChunk, 4)
	go func() {
		defer close(out)
		defer session.Close(context.Background())
		chunker := newPCM16Mono60MSChunker(24000)
		for {
			raw, err := session.ReadEvent(ctx)
			if err != nil {
				if err == io.EOF {
					for _, chunk := range chunker.Flush() {
						sendVoiceAudioChunk(ctx, out, chunk)
					}
				}
				return
			}
			eventType, _ := raw["type"].(string)
			if eventType == "response.audio.delta" {
				for _, chunk := range mustChunkDashScopeAudio(chunker, raw) {
					if !sendVoiceAudioChunk(ctx, out, chunk) {
						return
					}
				}
			}
			if eventType == "response.done" || eventType == "session.finished" {
				for _, chunk := range chunker.Flush() {
					sendVoiceAudioChunk(ctx, out, chunk)
				}
				return
			}
		}
	}()
	return out, nil
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
	return map[string]any{"type": "session.update", "session": session}
}

func mustChunkDashScopeAudio(chunker *pcm16Mono60MSChunker, raw map[string]any) []VoiceAudioChunk {
	chunks, err := chunker.AppendBase64(firstStringField(raw, "delta"))
	if err != nil {
		return nil
	}
	return chunks
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
