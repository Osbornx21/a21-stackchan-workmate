package providers

import (
	"context"
	"fmt"
	"strings"
)

func NewVoiceProviderFromEnv(env []string) VoiceProvider {
	rawPrimary := strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	if rawPrimary == "" {
		rawPrimary = "mock"
	}
	primary := strings.ToLower(rawPrimary)
	switch primary {
	case "mock":
		return NewMockVoiceProvider()
	case "openai_realtime":
		return NewOpenAIRealtimeVoiceProviderFromEnv(env, nil)
	case "doubao_realtime":
		return NewDoubaoRealtimeVoiceProviderFromEnv(env)
	case "doubao_tts_realtime":
		return NewDoubaoRealtimeTTSProviderFromEnv(env, nil)
	default:
		if containsLegacyProviderIdentity(primary) {
			return unavailableVoiceProvider{
				name:   "invalid_legacy_provider",
				detail: "provider target rejected",
			}
		}
		if providerProfileIsAgentTask(primary) {
			return unavailableVoiceProvider{
				name:   "invalid_agent_task_primary",
				detail: "agent-task profiles are not A21 voice providers",
			}
		}
		if knownProvider(primary) {
			return unavailableVoiceProvider{
				name:   primary,
				detail: "selected provider does not have an A21 voice adapter yet",
			}
		}
		return unavailableVoiceProvider{
			name:   "unknown_provider",
			detail: "provider target is unknown",
		}
	}
}

type unavailableVoiceProvider struct {
	name   string
	detail string
}

func (p unavailableVoiceProvider) Name() string {
	if p.name == "" {
		return "unknown_provider"
	}
	return p.name
}

func (p unavailableVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
	if err := ctx.Err(); err != nil {
		return VoiceProviderHealth{
			Provider:   p.Name(),
			Status:     VoiceProviderUnavailable,
			Configured: false,
			Realtime:   false,
			Detail:     "context cancelled",
		}, err
	}
	detail := p.detail
	if detail == "" {
		detail = "provider unavailable"
	}
	return VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     VoiceProviderUnavailable,
		Configured: false,
		Realtime:   false,
		Detail:     detail,
	}, fmt.Errorf("%w: %s", ErrVoiceProviderUnavailable, detail)
}

func (p unavailableVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}
