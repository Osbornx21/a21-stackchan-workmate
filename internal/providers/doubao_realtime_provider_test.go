package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDoubaoRealtimeVoiceProviderFromEnvReportsConfiguredWithoutLeakingSecrets(t *testing.T) {
	provider := NewDoubaoRealtimeVoiceProviderFromEnv([]string{
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Provider != "a21-doubao-realtime-voice" {
		t.Fatalf("provider = %q", health.Provider)
	}
	if health.Status != VoiceProviderDegraded || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want degraded configured realtime", health)
	}
	if !strings.Contains(health.Detail, "execution disabled") {
		t.Fatalf("detail = %q, want explicit execution guard", health.Detail)
	}
	rendered := doubaoRealtimeProviderMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestDoubaoRealtimeVoiceProviderFromEnvReportsMissingCredentials(t *testing.T) {
	provider := NewDoubaoRealtimeVoiceProviderFromEnv([]string{})

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderUnavailable || health.Configured {
		t.Fatalf("health = %#v, want unavailable unconfigured", health)
	}
	for _, want := range []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_APP_ID", "A21_DOUBAO_RESOURCE_ID", "A21_DOUBAO_REALTIME_MODEL"} {
		if !strings.Contains(health.Detail, want) {
			t.Fatalf("detail = %q, want %s", health.Detail, want)
		}
	}
}

func TestDoubaoRealtimeVoiceProviderStartTurnDoesNotDialProvider(t *testing.T) {
	provider := NewDoubaoRealtimeVoiceProvider(DoubaoRealtimeVoiceProviderConfig{
		APIKey:     "sk-a21-secret",
		AppID:      "app-a21-secret",
		ResourceID: "resource-a21-secret",
		Model:      "doubao-s2s",
	})

	_, err := provider.StartTurn(context.Background(), VoiceTurnRequest{
		Session: VoiceSession{TraceID: "a21-trace-doubao-s2s-no-dial"},
		Text:    "不要意外连外网",
		Mode:    "companion",
	})
	if err == nil {
		t.Fatal("expected StartTurn to reject unverified S2S path")
	}
	if !strings.Contains(err.Error(), "Doubao realtime speech-to-speech execution is disabled") {
		t.Fatalf("err = %v, want disabled S2S guidance", err)
	}
}

func TestDoubaoRealtimeVoiceProviderCancelReturnsLocalAcknowledgement(t *testing.T) {
	provider := NewDoubaoRealtimeVoiceProvider(DoubaoRealtimeVoiceProviderConfig{
		APIKey:     "sk-a21-secret",
		AppID:      "app-a21-secret",
		ResourceID: "resource-a21-secret",
		Model:      "doubao-s2s",
	})

	events, err := provider.Cancel(context.Background(), VoiceCancelRequest{
		Session: VoiceSession{
			TraceID:   "a21-trace-doubao-s2s-cancel",
			SessionID: "a21-session-doubao-s2s-cancel",
			DeviceID:  "stackchan-001",
		},
		Reason:   CancelBargeIn,
		StreamID: "a21-stream-doubao-s2s-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	event, ok := <-events
	if !ok {
		t.Fatal("expected local cancel acknowledgement event")
	}
	if event.Kind != VoiceEventCancelled || !event.Final || event.CancelReason != CancelBargeIn {
		t.Fatalf("event = %#v, want final cancel acknowledgement", event)
	}
	if event.StreamID != "a21-stream-doubao-s2s-001" || event.Session.DeviceID != "stackchan-001" {
		t.Fatalf("event = %#v, want stream and session propagated", event)
	}
	if _, ok := <-events; ok {
		t.Fatal("expected cancel acknowledgement channel to close")
	}
}

func doubaoRealtimeProviderMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
