package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestVoiceProviderFromEnvDefaultsToMock(t *testing.T) {
	provider := NewVoiceProviderFromEnv([]string{})
	if provider.Name() != "a21-mock-voice" {
		t.Fatalf("provider name = %q, want a21-mock-voice", provider.Name())
	}
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured {
		t.Fatalf("health = %#v, want healthy configured mock", health)
	}
}

func TestVoiceProviderFromEnvSelectsDoubaoRealtimeTTSWithoutLeakingSecrets(t *testing.T) {
	provider := NewVoiceProviderFromEnv([]string{
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
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want healthy configured realtime", health)
	}
	rendered := factoryMustJSON(t, health)
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestVoiceProviderFromEnvSelectsOpenAIRealtime(t *testing.T) {
	provider := NewVoiceProviderFromEnv([]string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime-2",
	})
	if provider.Name() != "a21-openai-realtime-voice" {
		t.Fatalf("provider name = %q, want a21-openai-realtime-voice", provider.Name())
	}
	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != VoiceProviderHealthy || !health.Configured || !health.Realtime {
		t.Fatalf("health = %#v, want healthy configured realtime", health)
	}
}

func TestVoiceProviderFromEnvRejectsLegacyPrimaryWithoutEchoingValue(t *testing.T) {
	provider := NewVoiceProviderFromEnv([]string{"A21_PROVIDER_PRIMARY=x21_voice"})
	health, err := provider.Health(context.Background())
	if err == nil {
		t.Fatal("expected legacy provider health to return unavailable error")
	}
	if health.Provider != "invalid_legacy_provider" || health.Status != VoiceProviderUnavailable {
		t.Fatalf("health = %#v, want invalid legacy unavailable", health)
	}
	rendered := factoryMustJSON(t, health) + err.Error()
	if strings.Contains(rendered, "x21_voice") {
		t.Fatalf("legacy provider leaked: %s", rendered)
	}
}

func factoryMustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
