package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRealtimeWebSocketPlanFromEnvBuildsOpenAIPlanWithoutLeakingSecrets(t *testing.T) {
	report := RealtimeWebSocketPlanFromEnv([]string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, "openai_realtime")

	if report.Provider != "openai_realtime" {
		t.Fatalf("provider = %q, want openai_realtime", report.Provider)
	}
	if report.Status != ProviderSmokeReady {
		t.Fatalf("status = %q, want ready", report.Status)
	}
	if !report.Configured {
		t.Fatal("configured = false, want true")
	}
	if report.EndpointHost != "api.openai.com" {
		t.Fatalf("endpoint host = %q, want api.openai.com", report.EndpointHost)
	}
	if report.NetworkMode != NetworkModeDirect {
		t.Fatalf("network mode = %q, want direct", report.NetworkMode)
	}
	rendered := realtimeMustJSON(t, report)
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("plan leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRealtimeWebSocketPlanFromEnvSkipsWhenCredentialsMissing(t *testing.T) {
	report := RealtimeWebSocketPlanFromEnv([]string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
	}, "openai_realtime")

	if report.Status != ProviderSmokeSkipped {
		t.Fatalf("status = %q, want skipped", report.Status)
	}
	for _, want := range []string{"A21_OPENAI_API_KEY", "A21_OPENAI_REALTIME_MODEL"} {
		if !stringSliceContains(report.MissingEnv, want) {
			t.Fatalf("missing env lacks %q: %#v", want, report.MissingEnv)
		}
	}
}

func TestRealtimeWebSocketPlanFromEnvRejectsLegacyProviderName(t *testing.T) {
	report := RealtimeWebSocketPlanFromEnv([]string{
		"A21_PROVIDER_PRIMARY=x21_realtime",
	}, "x21_realtime")

	if report.Provider != "invalid_legacy_provider" {
		t.Fatalf("provider = %q, want invalid_legacy_provider", report.Provider)
	}
	if report.Status != ProviderSmokeFailed {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	rendered := realtimeMustJSON(t, report)
	if strings.Contains(rendered, "x21_realtime") {
		t.Fatalf("legacy provider name leaked: %s", rendered)
	}
}

func TestRealtimeWebSocketSessionSendsSessionUpdateAndCancel(t *testing.T) {
	conn := &fakeRealtimeConn{}
	adapter := NewRealtimeWebSocketAdapter(RealtimeWebSocketConfig{
		Provider: "openai_realtime",
		URL:      "wss://api.openai.com/v1/realtime?model=gpt-realtime-2",
		Headers:  map[string]string{"Authorization": "Bearer sk-a21-secret"},
	}, fakeRealtimeDialer{conn: conn})

	session, err := adapter.Connect(context.Background(), map[string]any{
		"type":         "realtime",
		"instructions": "A21 test session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Cancel(context.Background(), VoiceCancelRequest{Reason: CancelBargeIn}); err != nil {
		t.Fatal(err)
	}

	if len(conn.messages) != 2 {
		t.Fatalf("messages = %#v, want 2 events", conn.messages)
	}
	if got, want := conn.messages[0]["type"], "session.update"; got != want {
		t.Fatalf("first event type = %v, want %s", got, want)
	}
	sessionPayload, ok := conn.messages[0]["session"].(map[string]any)
	if !ok {
		t.Fatalf("session payload = %#v, want object", conn.messages[0]["session"])
	}
	if got, want := sessionPayload["type"], "realtime"; got != want {
		t.Fatalf("session type = %v, want %s", got, want)
	}
	if got, want := conn.messages[1]["type"], "response.cancel"; got != want {
		t.Fatalf("second event type = %v, want %s", got, want)
	}
	if got, want := conn.messages[1]["reason"], string(CancelBargeIn); got != want {
		t.Fatalf("cancel reason = %v, want %s", got, want)
	}
}

func TestRealtimeWebSocketAdapterPassesHeadersToDialerOnly(t *testing.T) {
	dialer := &recordingRealtimeDialer{conn: &fakeRealtimeConn{}}
	adapter := NewRealtimeWebSocketAdapter(RealtimeWebSocketConfig{
		Provider: "openai_realtime",
		URL:      "wss://api.openai.com/v1/realtime?model=gpt-realtime-2",
		Headers: map[string]string{
			"Authorization":              "Bearer sk-a21-secret",
			"OpenAI-Safety-Identifier":   "a21-user-hash",
			"X-A21-Provider-Trace-Guard": "a21-test",
		},
	}, dialer)

	_, err := adapter.Connect(context.Background(), map[string]any{"type": "realtime"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dialer.url, "wss://api.openai.com/v1/realtime?model=gpt-realtime-2"; got != want {
		t.Fatalf("dial URL = %q, want %q", got, want)
	}
	if got := dialer.header.Get("Authorization"); got != "Bearer sk-a21-secret" {
		t.Fatalf("authorization header = %q, want secret passed to dialer", got)
	}

	report := adapter.Report()
	rendered := realtimeMustJSON(t, report)
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("adapter report leaked %q: %s", forbidden, rendered)
		}
	}
}

type fakeRealtimeConn struct {
	messages []map[string]any
	closed   bool
}

func (c *fakeRealtimeConn) WriteJSON(_ context.Context, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var message map[string]any
	if err := json.Unmarshal(encoded, &message); err != nil {
		return err
	}
	c.messages = append(c.messages, message)
	return nil
}

func (c *fakeRealtimeConn) Close(_ context.Context) error {
	c.closed = true
	return nil
}

type fakeRealtimeDialer struct {
	conn RealtimeConn
}

func (d fakeRealtimeDialer) Dial(_ context.Context, _ string, _ http.Header, _ NetworkPolicy) (RealtimeConn, error) {
	return d.conn, nil
}

type recordingRealtimeDialer struct {
	conn   RealtimeConn
	url    string
	header http.Header
}

func (d *recordingRealtimeDialer) Dial(_ context.Context, url string, header http.Header, _ NetworkPolicy) (RealtimeConn, error) {
	d.url = url
	d.header = header.Clone()
	return d.conn, nil
}

func realtimeMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
