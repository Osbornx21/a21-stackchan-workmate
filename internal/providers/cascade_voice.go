package providers

import (
	"context"
	"errors"
)

type CascadeVoiceProvider struct {
	providers []VoiceProvider
}

func NewCascadeVoiceProvider(providers ...VoiceProvider) *CascadeVoiceProvider {
	return &CascadeVoiceProvider{providers: providers}
}

func (p *CascadeVoiceProvider) Name() string {
	return "a21-cascade-voice"
}

func (p *CascadeVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	var errs []error
	for _, provider := range p.providers {
		events, err := provider.StartTurn(ctx, req)
		if err == nil {
			return events, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(append([]error{ErrVoiceProviderUnavailable}, errs...)...)
}

func (p *CascadeVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	var errs []error
	for _, provider := range p.providers {
		events, err := provider.Cancel(ctx, req)
		if err == nil {
			return events, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(append([]error{ErrVoiceProviderUnavailable}, errs...)...)
}

func (p *CascadeVoiceProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
	var errs []error
	var degraded *VoiceProviderHealth
	for _, provider := range p.providers {
		health, err := provider.Health(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if health.Status == VoiceProviderHealthy {
			return VoiceProviderHealth{
				Provider:       p.Name(),
				Status:         VoiceProviderHealthy,
				Configured:     true,
				Realtime:       health.Realtime,
				ActiveProvider: health.Provider,
			}, nil
		}
		if health.Status == VoiceProviderDegraded && degraded == nil {
			copy := health
			degraded = &copy
		}
	}
	if degraded != nil {
		return VoiceProviderHealth{
			Provider:       p.Name(),
			Status:         VoiceProviderDegraded,
			Configured:     true,
			Realtime:       degraded.Realtime,
			ActiveProvider: degraded.Provider,
			Detail:         degraded.Detail,
		}, nil
	}
	return VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     VoiceProviderUnavailable,
		Configured: len(p.providers) > 0,
		Detail:     "no healthy voice provider",
	}, errors.Join(append([]error{ErrVoiceProviderUnavailable}, errs...)...)
}

func (p *CascadeVoiceProvider) Close(ctx context.Context) error {
	var errs []error
	for _, provider := range p.providers {
		if err := provider.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
