package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
)

func TestOpenAIRealtimeVoiceProviderFromEnvReportsConfiguredWithoutLeakingSecrets(t *testing.T) {
	provider := NewOpenAIRealtimeVoiceProviderFromEnv([]string{
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, nil)
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Provider != "a21-openai-realtime-voice" {
		t.Fatalf("provider = %q, want a21-openai-realtime-voice", health.Provider)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want healthy configured realtime", health)
	}
	rendered := providerMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestOpenAIRealtimeVoiceProviderFromEnvReportsMissingCredentials(t *testing.T) {
	provider := NewOpenAIRealtimeVoiceProviderFromEnv([]string{}, nil)
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderUnavailable {
		t.Fatalf("status = %q, want unavailable", health.Status)
	}
	if health.Configured {
		t.Fatal("configured = true, want false")
	}
	if !strings.Contains(health.Detail, "A21_OPENAI_API_KEY") || !strings.Contains(health.Detail, "A21_OPENAI_REALTIME_MODEL") {
		t.Fatalf("detail = %q, want missing env names", health.Detail)
	}
}

func TestOpenAIRealtimeVoiceProviderSessionWritesUpdateAudioCommitAndCancel(t *testing.T) {
	conn := &fakeRealtimeConn{}
	provider := NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{
		APIKey:       "sk-a21-secret",
		Model:        "gpt-realtime-2",
		Instructions: "A21 test realtime voice",
	}, fakeRealtimeDialer{conn: conn})

	session, err := provider.StartRealtimeSession(context.Background(), VoiceSession{
		TraceID:   "a21-trace-openai-provider-001",
		SessionID: "a21-session-openai-provider-001",
		DeviceID:  "stackchan-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString([]byte{1, 2})
	if err := session.SendAudio(context.Background(), protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   payload,
	}); err != nil {
		t.Fatal(err)
	}
	if err := session.CommitAndCreateResponse(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := session.Cancel(context.Background(), VoiceCancelRequest{Reason: CancelBargeIn}); err != nil {
		t.Fatal(err)
	}

	got := realtimeEventTypes(conn.messages)
	want := []string{"session.update", "input_audio_buffer.append", "input_audio_buffer.commit", "response.create", "response.cancel"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("event types = %#v, want %#v", got, want)
	}
	sessionPayload, ok := conn.messages[0]["session"].(map[string]any)
	if !ok {
		t.Fatalf("session payload = %#v, want object", conn.messages[0]["session"])
	}
	if sessionPayload["type"] != "realtime" {
		t.Fatalf("session type = %v, want realtime", sessionPayload["type"])
	}
	if sessionPayload["instructions"] != "A21 test realtime voice" {
		t.Fatalf("instructions = %v", sessionPayload["instructions"])
	}
	for _, message := range conn.messages {
		rendered := providerMustJSON(t, message)
		for _, forbidden := range []string{"sk-a21-secret", "Authorization", "Bearer"} {
			if strings.Contains(rendered, forbidden) {
				t.Fatalf("provider event leaked %q: %s", forbidden, rendered)
			}
		}
	}
}

func TestOpenAIRealtimeVoiceProviderSessionReadsAudioDeltaEvents(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte{7, 8, 9})
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{
			{
				"type":        "response.output_audio.delta",
				"delta":       payload,
				"response_id": "a21-openai-response-001",
			},
		},
	}
	provider := NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{
		APIKey: "sk-a21-secret",
		Model:  "gpt-realtime-2",
	}, fakeRealtimeDialer{conn: conn})

	session, err := provider.StartRealtimeSession(context.Background(), VoiceSession{
		TraceID:   "a21-trace-openai-events",
		SessionID: "a21-session-openai-events",
		DeviceID:  "stackchan-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-session.Events():
		if event.Kind != VoiceEventSpeaking || event.StreamID != "a21-openai-response-001" {
			t.Fatalf("event = %+v", event)
		}
		if event.Audio == nil || event.Audio.DataBase64 != payload || event.Audio.SampleRateHz != 24000 {
			t.Fatalf("audio = %+v", event.Audio)
		}
		if event.Session.TraceID != "a21-trace-openai-events" || event.Session.DeviceID != "stackchan-001" {
			t.Fatalf("session = %+v", event.Session)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for provider audio delta event")
	}
}

func TestOpenAIRealtimeVoiceProviderSessionRequiresConfiguredProvider(t *testing.T) {
	provider := NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{}, fakeRealtimeDialer{conn: &fakeRealtimeConn{}})

	_, err := provider.StartRealtimeSession(context.Background(), VoiceSession{})
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
	if !strings.Contains(err.Error(), "A21_OPENAI_API_KEY") {
		t.Fatalf("err = %v, want missing key detail", err)
	}
}

func TestOpenAIRealtimeVoiceProviderStartTurnDoesNotDialProvider(t *testing.T) {
	dialer := &countingRealtimeDialer{conn: &fakeRealtimeConn{}}
	provider := NewOpenAIRealtimeVoiceProvider(OpenAIRealtimeVoiceProviderConfig{
		APIKey: "sk-a21-secret",
		Model:  "gpt-realtime-2",
	}, dialer)

	_, err := provider.StartTurn(context.Background(), VoiceTurnRequest{
		Session: VoiceSession{TraceID: "a21-trace-no-dial"},
		Text:    "不要意外连外网",
		Mode:    "workmate",
	})
	if err == nil {
		t.Fatal("expected StartTurn to reject text-turn path")
	}
	if !strings.Contains(err.Error(), "StartRealtimeSession") {
		t.Fatalf("err = %v, want StartRealtimeSession guidance", err)
	}
	if dialer.calls != 0 {
		t.Fatalf("dial calls = %d, want 0", dialer.calls)
	}
}

func realtimeEventTypes(messages []map[string]any) []string {
	types := make([]string, 0, len(messages))
	for _, message := range messages {
		eventType, _ := message["type"].(string)
		types = append(types, eventType)
	}
	return types
}

func providerMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

type countingRealtimeDialer struct {
	conn  RealtimeConn
	calls int
}

func (d *countingRealtimeDialer) Dial(_ context.Context, _ string, _ http.Header, _ NetworkPolicy) (RealtimeConn, error) {
	d.calls++
	return d.conn, nil
}
