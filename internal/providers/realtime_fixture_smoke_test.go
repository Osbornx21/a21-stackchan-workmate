package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRealtimeFixtureSmokeDoubaoTTSExecutesWithoutLeakingSecrets(t *testing.T) {
	report := RealtimeFixtureSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	}, "doubao_tts_realtime", true)

	if report.Provider != "doubao_tts_realtime" || report.Protocol != "websocket_realtime_fixture" {
		t.Fatalf("provider/protocol = %q/%q", report.Provider, report.Protocol)
	}
	if report.Status != ProviderSmokePassed || !report.Executed || !report.Configured {
		t.Fatalf("report = %#v, want passed executed configured", report)
	}
	if report.EndpointHost != "ai-gateway.vei.volces.com" {
		t.Fatalf("endpoint host = %q", report.EndpointHost)
	}
	rendered := realtimeFixtureSmokeJSON(t, report)
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("fixture smoke leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRealtimeFixtureSmokeOpenAIExecutesWithoutLeakingSecrets(t *testing.T) {
	report := RealtimeFixtureSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, "openai_realtime", true)

	if report.Provider != "openai_realtime" || report.Status != ProviderSmokePassed || !report.Executed {
		t.Fatalf("report = %#v, want openai passed executed", report)
	}
	if report.EndpointHost != "api.openai.com" {
		t.Fatalf("endpoint host = %q", report.EndpointHost)
	}
	rendered := realtimeFixtureSmokeJSON(t, report)
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("fixture smoke leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRealtimeFixtureSmokeRequiresExecute(t *testing.T) {
	report := RealtimeFixtureSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	}, "openai_realtime", false)

	if report.Status != ProviderSmokeReady || report.Executed {
		t.Fatalf("report = %#v, want ready without execution", report)
	}
	if !strings.Contains(report.Detail, "--execute") {
		t.Fatalf("detail = %q, want execute guidance", report.Detail)
	}
}

func TestRealtimeFixtureSmokeRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	report := RealtimeFixtureSmokeFromEnv(context.Background(), []string{"A21_PROVIDER_PRIMARY=x21_realtime"}, "x21_realtime", true)

	if report.Provider != "invalid_legacy_provider" || report.Status != ProviderSmokeFailed {
		t.Fatalf("report = %#v, want legacy failure", report)
	}
	rendered := realtimeFixtureSmokeJSON(t, report)
	if strings.Contains(rendered, "x21_realtime") {
		t.Fatalf("fixture smoke leaked legacy provider: %s", rendered)
	}
}

func realtimeFixtureSmokeJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
