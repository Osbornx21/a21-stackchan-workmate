package providers

import (
	"context"
	"fmt"
	"strings"
)

type DoubaoRealtimeVoiceProviderConfig struct {
	APIKey        string
	AppID         string
	ResourceID    string
	Model         string
	NetworkPolicy NetworkPolicy
}

type DoubaoRealtimeVoiceProvider struct {
	config DoubaoRealtimeVoiceProviderConfig
}

func NewDoubaoRealtimeVoiceProvider(config DoubaoRealtimeVoiceProviderConfig) *DoubaoRealtimeVoiceProvider {
	return &DoubaoRealtimeVoiceProvider{config: config}
}

func NewDoubaoRealtimeVoiceProviderFromEnv(env []string) *DoubaoRealtimeVoiceProvider {
	policy, _ := NetworkPolicyFromEnv(env)
	return NewDoubaoRealtimeVoiceProvider(DoubaoRealtimeVoiceProviderConfig{
		APIKey:        strings.TrimSpace(envValue(env, "A21_DOUBAO_API_KEY")),
		AppID:         strings.TrimSpace(envValue(env, "A21_DOUBAO_APP_ID")),
		ResourceID:    strings.TrimSpace(envValue(env, "A21_DOUBAO_RESOURCE_ID")),
		Model:         strings.TrimSpace(envValue(env, "A21_DOUBAO_REALTIME_MODEL")),
		NetworkPolicy: policy,
	})
}

func (p *DoubaoRealtimeVoiceProvider) Name() string {
	return "a21-doubao-realtime-voice"
}

func (p *DoubaoRealtimeVoiceProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
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
		Status:     VoiceProviderDegraded,
		Configured: true,
		Realtime:   true,
		Detail:     "execution disabled pending verified Doubao realtime speech-to-speech adapter",
	}, nil
}

func (p *DoubaoRealtimeVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("Doubao realtime speech-to-speech execution is disabled until the A21 adapter has verified official API shape, credentialed smoke, cancellation, and latency behavior")
}

func (p *DoubaoRealtimeVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
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

func (p *DoubaoRealtimeVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

func (p *DoubaoRealtimeVoiceProvider) configured() bool {
	return len(p.missingEnv()) == 0
}

func (p *DoubaoRealtimeVoiceProvider) missingEnv() []string {
	var missing []string
	if strings.TrimSpace(p.config.APIKey) == "" {
		missing = append(missing, "A21_DOUBAO_API_KEY")
	}
	if strings.TrimSpace(p.config.AppID) == "" {
		missing = append(missing, "A21_DOUBAO_APP_ID")
	}
	if strings.TrimSpace(p.config.ResourceID) == "" {
		missing = append(missing, "A21_DOUBAO_RESOURCE_ID")
	}
	if strings.TrimSpace(p.config.Model) == "" {
		missing = append(missing, "A21_DOUBAO_REALTIME_MODEL")
	}
	return missing
}
