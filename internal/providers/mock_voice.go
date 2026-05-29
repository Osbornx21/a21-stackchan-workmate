package providers

import "context"

type MockVoiceProvider struct{}

func NewMockVoiceProvider() *MockVoiceProvider {
	return &MockVoiceProvider{}
}

func (p *MockVoiceProvider) Name() string {
	return "a21-mock-voice"
}

func (p *MockVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	events := make(chan VoiceEvent, 2)
	defer close(events)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events <- VoiceEvent{
		Session: req.Session,
		Kind:    VoiceEventThinking,
		Text:    "我想一下",
	}
	events <- VoiceEvent{
		Session:  req.Session,
		Kind:     VoiceEventSpeaking,
		Text:     "先说，我在。",
		Final:    true,
		StreamID: "a21-mock-stream-000001",
	}
	return events, nil
}

func (p *MockVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	events := make(chan VoiceEvent, 1)
	defer close(events)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	streamID := req.StreamID
	if streamID == "" {
		streamID = "a21-mock-stream-000001"
	}
	events <- VoiceEvent{
		Session:      req.Session,
		Kind:         VoiceEventCancelled,
		Text:         "好，我听新的。",
		Final:        true,
		StreamID:     streamID,
		CancelReason: req.Reason,
	}
	return events, nil
}

func (p *MockVoiceProvider) Health(ctx context.Context) (VoiceProviderHealth, error) {
	if err := ctx.Err(); err != nil {
		return VoiceProviderHealth{Provider: p.Name(), Status: VoiceProviderUnavailable, Configured: true, Realtime: true}, err
	}
	return VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     VoiceProviderHealthy,
		Configured: true,
		Realtime:   true,
	}, nil
}

func (p *MockVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}
