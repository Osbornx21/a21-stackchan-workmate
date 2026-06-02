package providers

import (
	"context"
	"fmt"
	"strings"
)

type DoubaoRealtimeTTSProviderConfig struct {
	APIKey                string
	Model                 string
	Voice                 string
	OutputAudioFormat     string
	OutputAudioSampleRate int
	NetworkPolicy         NetworkPolicy
}

type DoubaoRealtimeTTSProvider struct {
	config DoubaoRealtimeTTSProviderConfig
	dialer RealtimeDialer
}

type DoubaoRealtimeTTSProviderSession struct {
	session          *RealtimeWebSocketSession
	voice            VoiceSession
	outputSampleRate int
}

func NewDoubaoRealtimeTTSProvider(config DoubaoRealtimeTTSProviderConfig, dialer RealtimeDialer) *DoubaoRealtimeTTSProvider {
	return &DoubaoRealtimeTTSProvider{config: config, dialer: dialer}
}

func NewDoubaoRealtimeTTSProviderFromEnv(env []string, dialer RealtimeDialer) *DoubaoRealtimeTTSProvider {
	policy, _ := NetworkPolicyFromEnv(env)
	return NewDoubaoRealtimeTTSProvider(DoubaoRealtimeTTSProviderConfig{
		APIKey:                strings.TrimSpace(envValue(env, "A21_DOUBAO_API_KEY")),
		Model:                 strings.TrimSpace(envValue(env, "A21_DOUBAO_TTS_MODEL")),
		Voice:                 strings.TrimSpace(envValue(env, "A21_DOUBAO_TTS_VOICE")),
		OutputAudioFormat:     strings.TrimSpace(envValue(env, "A21_DOUBAO_TTS_OUTPUT_FORMAT")),
		OutputAudioSampleRate: doubaoTTSOutputSampleRateFromEnv(env),
		NetworkPolicy:         policy,
	}, dialer)
}

func (p *DoubaoRealtimeTTSProvider) Name() string {
	return "a21-doubao-realtime-tts"
}

func (p *DoubaoRealtimeTTSProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
	if err := ctx.Err(); err != nil {
		return VoiceProviderHealth{
			Provider:   p.Name(),
			Status:     VoiceProviderUnavailable,
			Configured: p.configured(),
			Realtime:   true,
			Detail:     "context cancelled",
		}, err
	}
	if missing := p.missingEnv(); len(missing) > 0 {
		return VoiceProviderHealth{
			Provider:   p.Name(),
			Status:     VoiceProviderUnavailable,
			Configured: false,
			Realtime:   true,
			Detail:     "missing " + strings.Join(missing, ","),
		}, nil
	}
	return VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     VoiceProviderHealthy,
		Configured: true,
		Realtime:   true,
	}, nil
}

func (p *DoubaoRealtimeTTSProvider) StartRealtimeTTSSession(ctx context.Context, voiceSession VoiceSession) (*DoubaoRealtimeTTSProviderSession, error) {
	if missing := p.missingEnv(); len(missing) > 0 {
		return nil, fmt.Errorf("doubao realtime TTS provider missing %s", strings.Join(missing, ","))
	}
	endpoint, err := doubaoRealtimeURL(p.config.Model, "A21_DOUBAO_TTS_MODEL")
	if err != nil {
		return nil, err
	}
	dialer := p.dialer
	if dialer == nil {
		dialer = coderRealtimeDialer{}
	}
	conn, err := dialer.Dial(ctx, endpoint, realtimeHeaders(map[string]string{
		"Authorization": "Bearer " + p.config.APIKey,
	}), p.config.NetworkPolicy)
	if err != nil {
		return nil, err
	}
	session := &RealtimeWebSocketSession{
		provider: "doubao_tts_realtime",
		conn:     conn,
	}
	sampleRate := p.config.OutputAudioSampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	if err := DoubaoRealtimeTTSSendSessionUpdate(ctx, session, DoubaoRealtimeTTSConfig{
		Model:                 p.config.Model,
		Voice:                 p.config.Voice,
		OutputAudioFormat:     p.config.OutputAudioFormat,
		OutputAudioSampleRate: sampleRate,
	}); err != nil {
		_ = conn.Close(ctx)
		return nil, err
	}
	return &DoubaoRealtimeTTSProviderSession{session: session, voice: voiceSession, outputSampleRate: sampleRate}, nil
}

func (p *DoubaoRealtimeTTSProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("doubao realtime TTS text StartTurn is disabled; use StartRealtimeTTSSession for explicit TTS sessions")
}

func (p *DoubaoRealtimeTTSProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	events := make(chan VoiceEvent, 1)
	defer close(events)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events <- VoiceEvent{
		Session:      req.Session,
		Kind:         VoiceEventCancelled,
		Final:        true,
		StreamID:     req.StreamID,
		CancelReason: req.Reason,
	}
	return events, nil
}

func (p *DoubaoRealtimeTTSProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

func (s *DoubaoRealtimeTTSProviderSession) SendText(ctx context.Context, delta string) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("doubao realtime TTS provider session is not connected")
	}
	return DoubaoRealtimeTTSSendText(ctx, s.session, delta)
}

func (s *DoubaoRealtimeTTSProviderSession) TextDone(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("doubao realtime TTS provider session is not connected")
	}
	return DoubaoRealtimeTTSSendTextDone(ctx, s.session)
}

func (s *DoubaoRealtimeTTSProviderSession) Close(ctx context.Context) error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Close(ctx)
}

func (s *DoubaoRealtimeTTSProviderSession) OutputSampleRateHz() int {
	if s == nil || s.outputSampleRate == 0 {
		return 16000
	}
	return s.outputSampleRate
}

func (s *DoubaoRealtimeTTSProviderSession) ReadVoiceEvent(ctx context.Context) (VoiceEvent, bool, error) {
	if s == nil || s.session == nil {
		return VoiceEvent{}, false, fmt.Errorf("doubao realtime TTS provider session is not connected")
	}
	raw, err := s.session.ReadEvent(ctx)
	if err != nil {
		return VoiceEvent{}, false, err
	}
	return s.ServerEventToVoiceEvent(raw)
}

func (s *DoubaoRealtimeTTSProviderSession) ServerEventToVoiceEvent(raw map[string]any) (VoiceEvent, bool, error) {
	if s == nil {
		return VoiceEvent{}, false, fmt.Errorf("doubao realtime TTS provider session is not connected")
	}
	return DoubaoRealtimeTTSServerEventToVoiceEvent(s.voice, s.outputSampleRate, raw)
}

func (p *DoubaoRealtimeTTSProvider) configured() bool {
	return len(p.missingEnv()) == 0
}

func (p *DoubaoRealtimeTTSProvider) missingEnv() []string {
	var missing []string
	if strings.TrimSpace(p.config.APIKey) == "" {
		missing = append(missing, "A21_DOUBAO_API_KEY")
	}
	if strings.TrimSpace(p.config.Model) == "" {
		missing = append(missing, "A21_DOUBAO_TTS_MODEL")
	}
	if strings.TrimSpace(p.config.Voice) == "" {
		missing = append(missing, "A21_DOUBAO_TTS_VOICE")
	}
	return missing
}

func doubaoTTSOutputSampleRateFromEnv(env []string) int {
	switch strings.TrimSpace(envValue(env, "A21_DOUBAO_TTS_SAMPLE_RATE_HZ")) {
	case "24000":
		return 24000
	case "48000":
		return 48000
	default:
		return 16000
	}
}
