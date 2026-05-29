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

func (p *CascadeVoiceProvider) Close(ctx context.Context) error {
	var errs []error
	for _, provider := range p.providers {
		if err := provider.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
