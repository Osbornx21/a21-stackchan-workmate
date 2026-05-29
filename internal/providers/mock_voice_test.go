package providers

import (
	"context"
	"testing"
)

func TestMockVoiceProviderStartTurnStreamsDeterministicEvents(t *testing.T) {
	provider := NewMockVoiceProvider()
	events, err := provider.StartTurn(context.Background(), VoiceTurnRequest{
		Session: VoiceSession{
			TraceID:   "a21-trace-000001",
			SessionID: "a21-session-000001",
			DeviceID:  "stackchan-sim-001",
		},
		Text: "先说，我在",
		Mode: "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := drainVoiceEvents(events)
	if len(got) != 2 {
		t.Fatalf("events = %d, want 2", len(got))
	}
	if got[0].Kind != VoiceEventThinking {
		t.Fatalf("first kind = %q, want thinking", got[0].Kind)
	}
	if got[1].Kind != VoiceEventSpeaking {
		t.Fatalf("second kind = %q, want speaking", got[1].Kind)
	}
	if got[1].Text != "先说，我在。" {
		t.Fatalf("speaking text = %q", got[1].Text)
	}
	if got[1].Session.TraceID != "a21-trace-000001" {
		t.Fatalf("trace = %q", got[1].Session.TraceID)
	}
}

func TestMockVoiceProviderCancelStreamsCancelledEvent(t *testing.T) {
	provider := NewMockVoiceProvider()
	events, err := provider.Cancel(context.Background(), VoiceCancelRequest{
		Session: VoiceSession{TraceID: "a21-trace-000009", SessionID: "a21-session-000009"},
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
	if got[0].Session.SessionID != "a21-session-000009" {
		t.Fatalf("session = %q", got[0].Session.SessionID)
	}
}

func drainVoiceEvents(events <-chan VoiceEvent) []VoiceEvent {
	var out []VoiceEvent
	for event := range events {
		out = append(out, event)
	}
	return out
}
