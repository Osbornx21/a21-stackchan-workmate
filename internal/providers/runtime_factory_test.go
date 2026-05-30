package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGatewayVoiceProviderFromEnvDefaultsToMockDespiteSelectedPrimary(t *testing.T) {
	provider := NewGatewayVoiceProviderFromEnv([]string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	if provider.Name() != "a21-mock-voice" {
		t.Fatalf("provider name = %q, want a21-mock-voice", provider.Name())
	}
}

func TestGatewayVoiceProviderFromEnvDefaultsToMockDespiteDoubaoRealtimePrimary(t *testing.T) {
	provider := NewGatewayVoiceProviderFromEnv([]string{
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	if provider.Name() != "a21-mock-voice" {
		t.Fatalf("provider name = %q, want a21-mock-voice", provider.Name())
	}
}

func TestGatewayVoiceProviderFromEnvUsesSelectedProviderWhenExplicit(t *testing.T) {
	provider := NewGatewayVoiceProviderFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	if provider.Name() != "a21-doubao-realtime-tts" {
		t.Fatalf("provider name = %q, want a21-doubao-realtime-tts", provider.Name())
	}
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured {
		t.Fatalf("health = %#v, want configured healthy provider", health)
	}
	rendered := runtimeProviderMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestGatewayVoiceProviderFromEnvUsesSelectedDoubaoRealtimeWhenExplicit(t *testing.T) {
	provider := NewGatewayVoiceProviderFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	if provider.Name() != "a21-doubao-realtime-voice" {
		t.Fatalf("provider name = %q, want a21-doubao-realtime-voice", provider.Name())
	}
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderDegraded || !health.Configured {
		t.Fatalf("health = %#v, want configured degraded provider", health)
	}
	if !strings.Contains(health.Detail, "execution disabled") {
		t.Fatalf("detail = %q, want explicit execution guard", health.Detail)
	}
	rendered := runtimeProviderMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestGatewayVoiceProviderFromEnvRejectsLegacyRuntimeModeWithoutEchoingValue(t *testing.T) {
	provider := NewGatewayVoiceProviderFromEnv([]string{"A21_GATEWAY_VOICE_PROVIDER=x21-runtime"})
	health, err := provider.Health(context.Background())
	if err == nil {
		t.Fatal("expected legacy runtime mode to be unavailable")
	}
	if health.Provider != "invalid_gateway_voice_provider" || health.Status != VoiceProviderUnavailable {
		t.Fatalf("health = %#v, want invalid gateway runtime unavailable", health)
	}
	rendered := runtimeProviderMustJSON(t, health) + err.Error()
	if strings.Contains(rendered, "x21-runtime") {
		t.Fatalf("runtime mode leaked: %s", rendered)
	}
}

func runtimeProviderMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
