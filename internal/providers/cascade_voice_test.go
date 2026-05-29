package providers

import (
	"context"
	"errors"
	"testing"
)

func TestCascadeVoiceProviderFallsBackOnStartTurnError(t *testing.T) {
	cascade := NewCascadeVoiceProvider(
		errorVoiceProvider{name: "primary"},
		scriptedProvider{name: "fallback", text: "fallback speaking"},
	)
	events, err := cascade.StartTurn(context.Background(), VoiceTurnRequest{
		Session: VoiceSession{TraceID: "a21-trace-000001"},
		Text:    "hello",
		Mode:    "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := drainVoiceEvents(events)
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].Text != "fallback speaking" {
		t.Fatalf("text = %q, want fallback speaking", got[0].Text)
	}
}

func TestCascadeVoiceProviderCancelUsesFirstSuccessfulProvider(t *testing.T) {
	cascade := NewCascadeVoiceProvider(
		errorVoiceProvider{name: "primary"},
		scriptedProvider{name: "fallback", text: "cancelled"},
	)
	events, err := cascade.Cancel(context.Background(), VoiceCancelRequest{
		Session: VoiceSession{SessionID: "a21-session-000001"},
		Reason:  CancelBargeIn,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := drainVoiceEvents(events)
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].Kind != VoiceEventCancelled {
		t.Fatalf("kind = %q, want cancelled", got[0].Kind)
	}
}

func TestCascadeVoiceProviderReturnsErrorWhenAllProvidersFail(t *testing.T) {
	cascade := NewCascadeVoiceProvider(errorVoiceProvider{name: "primary"}, errorVoiceProvider{name: "fallback"})
	_, err := cascade.StartTurn(context.Background(), VoiceTurnRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrVoiceProviderUnavailable) {
		t.Fatalf("err = %v, want ErrVoiceProviderUnavailable", err)
	}
}

type errorVoiceProvider struct {
	name string
}

func (p errorVoiceProvider) Name() string {
	return p.name
}

func (p errorVoiceProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	return nil, ErrVoiceProviderUnavailable
}

func (p errorVoiceProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	return nil, ErrVoiceProviderUnavailable
}

func (p errorVoiceProvider) Close(ctx context.Context) error {
	return nil
}

type scriptedProvider struct {
	name string
	text string
}

func (p scriptedProvider) Name() string {
	return p.name
}

func (p scriptedProvider) StartTurn(ctx context.Context, req VoiceTurnRequest) (<-chan VoiceEvent, error) {
	events := make(chan VoiceEvent, 1)
	defer close(events)
	events <- VoiceEvent{Session: req.Session, Kind: VoiceEventSpeaking, Text: p.text, Final: true}
	return events, nil
}

func (p scriptedProvider) Cancel(ctx context.Context, req VoiceCancelRequest) (<-chan VoiceEvent, error) {
	events := make(chan VoiceEvent, 1)
	defer close(events)
	events <- VoiceEvent{Session: req.Session, Kind: VoiceEventCancelled, Text: p.text, Final: true}
	return events, nil
}

func (p scriptedProvider) Close(ctx context.Context) error {
	return nil
}
