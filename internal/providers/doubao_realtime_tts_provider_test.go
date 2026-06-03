package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDoubaoRealtimeTTSProviderFromEnvReportsConfiguredWithoutLeakingSecrets(t *testing.T) {
	provider := NewDoubaoRealtimeTTSProviderFromEnv([]string{
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	}, nil)

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Provider != "a21-doubao-realtime-tts" {
		t.Fatalf("provider = %q", health.Provider)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want healthy configured realtime", health)
	}
	rendered := doubaoTTSProviderMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestDoubaoRealtimeTTSProviderFromEnvAcceptsAccessTokenAlias(t *testing.T) {
	provider := NewDoubaoRealtimeTTSProviderFromEnv([]string{
		"A21_DOUBAO_ACCESS_TOKEN=access-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_SECRET_KEY=secret-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=voice-secret",
	}, nil)

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want healthy configured realtime", health)
	}
	rendered := doubaoTTSProviderMustJSON(t, health)
	for _, forbidden := range []string{"access-a21-secret", "app-a21-secret", "secret-a21-secret", "doubao-tts", "voice-secret", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestDoubaoRealtimeTTSProviderFromEnvReportsMissingCredentials(t *testing.T) {
	provider := NewDoubaoRealtimeTTSProviderFromEnv([]string{}, nil)

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderUnavailable || health.Configured {
		t.Fatalf("health = %#v, want unavailable unconfigured", health)
	}
	for _, want := range []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_TTS_MODEL", "A21_DOUBAO_TTS_VOICE"} {
		if !strings.Contains(health.Detail, want) {
			t.Fatalf("detail = %q, want %s", health.Detail, want)
		}
	}
}

func TestDoubaoRealtimeTTSProviderSessionWritesUpdateTextAndDone(t *testing.T) {
	conn := &fakeRealtimeConn{}
	provider := NewDoubaoRealtimeTTSProvider(DoubaoRealtimeTTSProviderConfig{
		APIKey:                "sk-a21-secret",
		Model:                 "doubao-tts",
		Voice:                 "zh_female_kailangjiejie_moon_bigtts",
		OutputAudioSampleRate: 16000,
	}, fakeRealtimeDialer{conn: conn})

	session, err := provider.StartRealtimeTTSSession(context.Background(), VoiceSession{
		TraceID:   "a21-trace-doubao-tts-provider-001",
		SessionID: "a21-session-doubao-tts-provider-001",
		DeviceID:  "stackchan-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.SendText(context.Background(), "你好"); err != nil {
		t.Fatal(err)
	}
	if err := session.TextDone(context.Background()); err != nil {
		t.Fatal(err)
	}

	got := realtimeEventTypes(conn.messages)
	want := []string{"tts_session.update", "input_text.append", "input_text.done"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("event types = %#v, want %#v", got, want)
	}
	for _, message := range conn.messages {
		rendered := doubaoTTSProviderMustJSON(t, message)
		for _, forbidden := range []string{"sk-a21-secret", "Authorization", "Bearer"} {
			if strings.Contains(rendered, forbidden) {
				t.Fatalf("provider event leaked %q: %s", forbidden, rendered)
			}
		}
	}
}

func TestDoubaoRealtimeTTSProviderStartTurnDoesNotDialProvider(t *testing.T) {
	dialer := &countingRealtimeDialer{conn: &fakeRealtimeConn{}}
	provider := NewDoubaoRealtimeTTSProvider(DoubaoRealtimeTTSProviderConfig{
		APIKey: "sk-a21-secret",
		Model:  "doubao-tts",
		Voice:  "zh_female_kailangjiejie_moon_bigtts",
	}, dialer)

	_, err := provider.StartTurn(context.Background(), VoiceTurnRequest{
		Session: VoiceSession{TraceID: "a21-trace-doubao-no-dial"},
		Text:    "不要意外连外网",
		Mode:    "professional",
	})
	if err == nil {
		t.Fatal("expected StartTurn to reject ordinary text-turn path")
	}
	if !strings.Contains(err.Error(), "StartRealtimeTTSSession") {
		t.Fatalf("err = %v, want StartRealtimeTTSSession guidance", err)
	}
	if dialer.calls != 0 {
		t.Fatalf("dial calls = %d, want 0", dialer.calls)
	}
}

type doubaoTTSCountingRealtimeDialer struct {
	conn  RealtimeConn
	calls int
}

func (d *doubaoTTSCountingRealtimeDialer) Dial(_ context.Context, _ string, _ http.Header, _ NetworkPolicy) (RealtimeConn, error) {
	d.calls++
	return d.conn, nil
}

func doubaoTTSProviderMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
