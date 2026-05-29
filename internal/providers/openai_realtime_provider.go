package providers

import (
	"context"
	"fmt"
	"strings"

	"a21.local/a21/internal/protocol"
)

type OpenAIRealtimeVoiceProviderConfig struct {
	APIKey           string
	Model            string
	SafetyIdentifier string
	Instructions     string
	NetworkPolicy    NetworkPolicy
}

type OpenAIRealtimeVoiceProvider struct {
	config OpenAIRealtimeVoiceProviderConfig
	dialer RealtimeDialer
}

type OpenAIRealtimeProviderSession struct {
	session *RealtimeWebSocketSession
	voice   VoiceSession
}

func NewOpenAIRealtimeVoiceProvider(config OpenAIRealtimeVoiceProviderConfig, dialer RealtimeDialer) *OpenAIRealtimeVoiceProvider {
	return &OpenAIRealtimeVoiceProvider{config: config, dialer: dialer}
}

func NewOpenAIRealtimeVoiceProviderFromEnv(env []string, dialer RealtimeDialer) *OpenAIRealtimeVoiceProvider {
	policy, _ := NetworkPolicyFromEnv(env)
	return NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{
		APIKey:           strings.TrimSpace(envValue(env, "A21_OPENAI_API_KEY")),
		Model:            strings.TrimSpace(envValue(env, "A21_OPENAI_REALTIME_MODEL")),
		SafetyIdentifier: strings.TrimSpace(envValue(env, "A21_OPENAI_SAFETY_IDENTIFIER")),
		Instructions:     strings.TrimSpace(envValue(env, "A21_OPENAI_REALTIME_INSTRUCTIONS")),
		NetworkPolicy:    policy,
	}, dialer)
}

func (p *OpenAIRealtimeVoiceProvider) Name() string {
	return "a21-openai-realtime-voice"
}

func (p *OpenAIRealtimeVoiceProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
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

func (p *OpenAIRealtimeVoiceProvider) StartRealtimeSession(ctx context.Context, voiceSession VoiceSession) (*OpenAIRealtimeProviderSession, error) {
	if missing := p.missingEnv(); len(missing) > 0 {
		return nil, fmt.Errorf("openai realtime provider missing %s", strings.Join(missing, ","))
	}
	endpoint, err := openAIRealtimeURL(p.config.Model)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Authorization": "Bearer " + p.config.APIKey,
	}
	if p.config.SafetyIdentifier != "" {
		headers["OpenAI-Safety-Identifier"] = p.config.SafetyIdentifier
	}
	realtime := NewRealtimeWebSocketAdapter(RealtimeWebSocketConfig{
		Provider:      "openai_realtime",
		URL:           endpoint,
		Headers:       headers,
		NetworkPolicy: p.config.NetworkPolicy,
	}, p.dialer)
	sessionPayload := map[string]any{
		"type": "realtime",
	}
	if p.config.Instructions != "" {
		sessionPayload["instructions"] = p.config.Instructions
	}
	session, err := realtime.Connect(ctx, sessionPayload)
	if err != nil {
		return nil, err
	}
	return &OpenAIRealtimeProviderSession{session: session, voice: voiceSession}, nil
}

func (p *OpenAIRealtimeVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("openai realtime text StartTurn is disabled; use StartRealtimeSession for explicit realtime audio sessions")
}

func (p *OpenAIRealtimeVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
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

func (p *OpenAIRealtimeVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

func (s *OpenAIRealtimeProviderSession) SendAudio(ctx context.Context, chunk protocol.AudioChunk) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("openai realtime provider session is not connected")
	}
	return OpenAIRealtimeSendAudioChunk(ctx, s.session, chunk)
}

func (s *OpenAIRealtimeProviderSession) CommitAndCreateResponse(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("openai realtime provider session is not connected")
	}
	return OpenAIRealtimeCommitAndCreateResponse(ctx, s.session)
}

func (s *OpenAIRealtimeProviderSession) Cancel(ctx context.Context, req VoiceCancelRequest) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("openai realtime provider session is not connected")
	}
	return s.session.Cancel(ctx, req)
}

func (s *OpenAIRealtimeProviderSession) Close(ctx context.Context) error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Close(ctx)
}

func (p *OpenAIRealtimeVoiceProvider) configured() bool {
	return len(p.missingEnv()) == 0
}

func (p *OpenAIRealtimeVoiceProvider) missingEnv() []string {
	var missing []string
	if strings.TrimSpace(p.config.APIKey) == "" {
		missing = append(missing, "A21_OPENAI_API_KEY")
	}
	if strings.TrimSpace(p.config.Model) == "" {
		missing = append(missing, "A21_OPENAI_REALTIME_MODEL")
	}
	return missing
}
