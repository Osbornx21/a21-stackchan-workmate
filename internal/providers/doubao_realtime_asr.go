package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	doubaoRealtimeASRSessionUpdate      = "transcription_session.update"
	doubaoRealtimeASRInputAudioAppend   = "input_audio_buffer.append"
	doubaoRealtimeASRInputAudioCommit   = "input_audio_buffer.commit"
	doubaoRealtimeASRTranscriptResult   = "conversation.item.input_audio_transcription.result"
	doubaoRealtimeASRTranscriptDelta    = "conversation.item.input_audio_transcription.delta"
	doubaoRealtimeASRTranscriptComplete = "conversation.item.input_audio_transcription.completed"
)

type DoubaoRealtimeASRProviderConfig struct {
	APIKey        string
	Model         string
	InputFormat   string
	InputRateHz   int
	NetworkPolicy NetworkPolicy
}

type DoubaoRealtimeASRAdapterOptions struct {
	Name   string
	Env    []string
	Dialer RealtimeDialer
}

type doubaoRealtimeASRAdapter struct {
	name   string
	config DoubaoRealtimeASRProviderConfig
	dialer RealtimeDialer
}

type doubaoRealtimeASRSession struct {
	session *RealtimeWebSocketSession
	events  chan ASRAdapterEvent
	done    chan struct{}
}

func NewDoubaoRealtimeASRAdapter(options DoubaoRealtimeASRAdapterOptions) ASRAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "doubao_asr_realtime"
	}
	policy, _ := NetworkPolicyFromEnv(options.Env)
	return &doubaoRealtimeASRAdapter{
		name: name,
		config: DoubaoRealtimeASRProviderConfig{
			APIKey:        doubaoRealtimeTTSAPIKeyFromEnv(options.Env),
			Model:         strings.TrimSpace(envValue(options.Env, "A21_DOUBAO_ASR_MODEL")),
			InputFormat:   strings.TrimSpace(envValue(options.Env, "A21_DOUBAO_ASR_INPUT_FORMAT")),
			InputRateHz:   doubaoASRInputSampleRateFromEnv(options.Env),
			NetworkPolicy: policy,
		},
		dialer: options.Dialer,
	}
}

func (a *doubaoRealtimeASRAdapter) Name() string {
	if a == nil || strings.TrimSpace(a.name) == "" {
		return "doubao_asr_realtime"
	}
	return a.name
}

func (a *doubaoRealtimeASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
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

func (a *doubaoRealtimeASRAdapter) StartStreamingASR(ctx context.Context, _ StreamingASRStartRequest) (StreamingASRSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if missing := a.missingEnv(); len(missing) > 0 {
		return nil, fmt.Errorf("doubao realtime ASR provider missing %s", strings.Join(missing, ","))
	}
	endpoint, err := doubaoRealtimeURL(a.config.Model, "A21_DOUBAO_ASR_MODEL")
	if err != nil {
		return nil, err
	}
	dialer := a.dialer
	if dialer == nil {
		dialer = coderRealtimeDialer{}
	}
	conn, err := dialer.Dial(ctx, endpoint, realtimeHeaders(map[string]string{
		"Authorization": "Bearer " + a.config.APIKey,
	}), a.config.NetworkPolicy)
	if err != nil {
		return nil, err
	}
	session := &RealtimeWebSocketSession{provider: a.Name(), conn: conn}
	if err := DoubaoRealtimeASRSendSessionUpdate(ctx, session, a.config); err != nil {
		_ = conn.Close(ctx)
		return nil, err
	}
	stream := &doubaoRealtimeASRSession{
		session: session,
		events:  make(chan ASRAdapterEvent, 8),
		done:    make(chan struct{}),
	}
	go stream.readLoop(ctx)
	return stream, nil
}

func (a *doubaoRealtimeASRAdapter) missingEnv() []string {
	var missing []string
	if strings.TrimSpace(a.config.APIKey) == "" {
		missing = append(missing, "A21_DOUBAO_API_KEY")
	}
	if strings.TrimSpace(a.config.Model) == "" {
		missing = append(missing, "A21_DOUBAO_ASR_MODEL")
	}
	return missing
}

func DoubaoRealtimeASRSessionUpdateEvent(config DoubaoRealtimeASRProviderConfig) (map[string]any, error) {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		return nil, fmt.Errorf("doubao realtime ASR model is required")
	}
	format := strings.TrimSpace(config.InputFormat)
	if format == "" {
		format = "pcm"
	}
	rate := config.InputRateHz
	if rate == 0 {
		rate = 16000
	}
	if rate != 16000 && rate != 24000 {
		return nil, fmt.Errorf("doubao realtime ASR sample rate %d is unsupported", rate)
	}
	return map[string]any{
		"type": doubaoRealtimeASRSessionUpdate,
		"session": map[string]any{
			"input_audio_format":      format,
			"input_audio_codec":       "raw",
			"input_audio_sample_rate": rate,
			"input_audio_bits":        16,
			"input_audio_channel":     1,
			"input_audio_transcription": map[string]any{
				"model": model,
			},
		},
	}, nil
}

func DoubaoRealtimeASRInputAudioAppendEvent(frame VoicePipelinePCMFrame) (map[string]any, error) {
	if len(frame.PCM16LE) == 0 {
		return nil, fmt.Errorf("doubao realtime ASR input audio frame is empty")
	}
	return map[string]any{
		"type":  doubaoRealtimeASRInputAudioAppend,
		"audio": base64.StdEncoding.EncodeToString(frame.PCM16LE),
	}, nil
}

func DoubaoRealtimeASRInputAudioCommitEvent() map[string]any {
	return map[string]any{"type": doubaoRealtimeASRInputAudioCommit}
}

func DoubaoRealtimeASRServerEventToASREvent(raw map[string]any) (ASRAdapterEvent, bool, error) {
	eventType, _ := raw["type"].(string)
	switch eventType {
	case doubaoRealtimeASRTranscriptResult, doubaoRealtimeASRTranscriptDelta:
		text := firstStringField(raw, "delta", "transcript", "text")
		if strings.TrimSpace(text) == "" {
			return ASRAdapterEvent{}, false, nil
		}
		return ASRAdapterEvent{Text: text}, true, nil
	case doubaoRealtimeASRTranscriptComplete:
		text := firstStringField(raw, "transcript", "text", "delta")
		if strings.TrimSpace(text) == "" {
			return ASRAdapterEvent{}, false, nil
		}
		return ASRAdapterEvent{Text: text, Final: true}, true, nil
	default:
		return ASRAdapterEvent{}, false, nil
	}
}

func DoubaoRealtimeASRSendSessionUpdate(ctx context.Context, session *RealtimeWebSocketSession, config DoubaoRealtimeASRProviderConfig) error {
	event, err := DoubaoRealtimeASRSessionUpdateEvent(config)
	if err != nil {
		return err
	}
	return doubaoRealtimeASRWriteEvent(ctx, session, event)
}

func DoubaoRealtimeASRSendAudio(ctx context.Context, session *RealtimeWebSocketSession, frame VoicePipelinePCMFrame) error {
	event, err := DoubaoRealtimeASRInputAudioAppendEvent(frame)
	if err != nil {
		return err
	}
	return doubaoRealtimeASRWriteEvent(ctx, session, event)
}

func DoubaoRealtimeASRSendCommit(ctx context.Context, session *RealtimeWebSocketSession) error {
	return doubaoRealtimeASRWriteEvent(ctx, session, DoubaoRealtimeASRInputAudioCommitEvent())
}

func (s *doubaoRealtimeASRSession) AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("doubao realtime ASR session is not connected")
	}
	return DoubaoRealtimeASRSendAudio(ctx, s.session, frame)
}

func (s *doubaoRealtimeASRSession) Events() <-chan ASRAdapterEvent {
	return s.events
}

func (s *doubaoRealtimeASRSession) Commit(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("doubao realtime ASR session is not connected")
	}
	return DoubaoRealtimeASRSendCommit(ctx, s.session)
}

func (s *doubaoRealtimeASRSession) Cancel(err error) {
	if s == nil {
		return
	}
	select {
	case <-s.done:
		return
	default:
	}
	if err != nil {
		select {
		case s.events <- ASRAdapterEvent{Finding: "doubao realtime ASR session cancelled", Err: err}:
		default:
		}
	}
	_ = s.session.Close(context.Background())
}

func (s *doubaoRealtimeASRSession) readLoop(ctx context.Context) {
	defer close(s.done)
	defer close(s.events)
	defer s.session.Close(context.Background())
	for {
		raw, err := s.session.ReadEvent(ctx)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				s.events <- ASRAdapterEvent{Finding: "doubao realtime ASR read failed", Err: fmt.Errorf("doubao realtime ASR read failed")}
			}
			return
		}
		event, ok, err := DoubaoRealtimeASRServerEventToASREvent(raw)
		if err != nil {
			s.events <- ASRAdapterEvent{Finding: "doubao realtime ASR event parse failed", Err: err}
			return
		}
		if !ok {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case s.events <- event:
		}
		if event.Final {
			return
		}
	}
}

func doubaoRealtimeASRWriteEvent(ctx context.Context, session *RealtimeWebSocketSession, event map[string]any) error {
	if session == nil || session.conn == nil {
		return fmt.Errorf("doubao realtime ASR session is not connected")
	}
	if event["type"] == "" {
		return fmt.Errorf("doubao realtime ASR event type is required")
	}
	return session.conn.WriteJSON(ctx, event)
}

func doubaoASRInputSampleRateFromEnv(env []string) int {
	switch strings.TrimSpace(envValue(env, "A21_DOUBAO_ASR_SAMPLE_RATE_HZ")) {
	case "24000":
		return 24000
	default:
		return 16000
	}
}

func firstStringField(values map[string]any, names ...string) string {
	for _, name := range names {
		if value, _ := values[name].(string); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
