package providers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProviderCatalogDefaultsToMockPrimary(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_ENV=development"})

	if report.Primary != "mock" {
		t.Fatalf("primary = %q, want mock", report.Primary)
	}
	mock := providerReadinessByName(t, report, "mock")
	if !mock.Selected || !mock.Configured || !mock.Realtime {
		t.Fatalf("mock readiness = %+v", mock)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want none", report.Findings)
	}
}

func TestProviderCatalogReportsSelectedDoubaoMissingCredentials(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_PROVIDER_PRIMARY=doubao_realtime"})

	doubao := providerReadinessByName(t, report, "doubao_realtime")
	if !doubao.Selected {
		t.Fatal("expected doubao to be selected")
	}
	if doubao.Configured {
		t.Fatal("doubao should not be configured without required env")
	}
	for _, want := range []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_APP_ID", "A21_DOUBAO_RESOURCE_ID", "A21_DOUBAO_REALTIME_MODEL"} {
		if !stringSliceContains(doubao.MissingEnv, want) {
			t.Fatalf("doubao missing env lacks %q: %#v", want, doubao.MissingEnv)
		}
	}
}

func TestProviderCatalogReportsDoubaoTTSRealtimeReadiness(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})

	doubao := providerReadinessByName(t, report, "doubao_tts_realtime")
	if !doubao.Selected || !doubao.Configured || !doubao.Realtime {
		t.Fatalf("doubao tts readiness = %+v", doubao)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(data)
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("catalog leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestProviderCatalogMarksOpenAIConfiguredWithoutLeakingSecret(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime",
	})

	openai := providerReadinessByName(t, report, "openai_realtime")
	if !openai.Selected || !openai.Configured || !openai.Realtime {
		t.Fatalf("openai readiness = %+v", openai)
	}
	if len(openai.MissingEnv) != 0 {
		t.Fatalf("openai missing env = %#v", openai.MissingEnv)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-a21-secret") {
		t.Fatalf("catalog leaked API key: %s", data)
	}
	if !strings.Contains(string(data), "A21_OPENAI_API_KEY") {
		t.Fatalf("catalog should expose env names only: %s", data)
	}
}

func TestProviderCatalogFlagsLegacyPrimaryIdentity(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_PROVIDER_PRIMARY=x21_voice"})

	if report.Primary == "x21_voice" {
		t.Fatalf("primary leaked legacy provider value: %#v", report)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	if report.Findings[0].Code != "provider_legacy_identity" {
		t.Fatalf("code = %q, want provider_legacy_identity", report.Findings[0].Code)
	}
	if strings.Contains(report.Findings[0].Detail, "x21_voice") {
		t.Fatalf("finding detail leaked legacy provider value: %#v", report.Findings[0])
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "x21_voice") {
		t.Fatalf("catalog leaked legacy provider value: %s", data)
	}
}

func TestProviderCatalogRedactsUnknownPrimary(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_PROVIDER_PRIMARY=experimental-provider"})

	if report.Primary != "unknown_provider" {
		t.Fatalf("primary = %q, want unknown_provider", report.Primary)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	if report.Findings[0].Code != "provider_unknown" {
		t.Fatalf("code = %q, want provider_unknown", report.Findings[0].Code)
	}
	for _, readiness := range report.Providers {
		if readiness.Selected {
			t.Fatalf("unexpected selected provider for unknown primary: %+v", readiness)
		}
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "experimental-provider") {
		t.Fatalf("catalog leaked unknown provider value: %s", data)
	}
}

func TestProviderCatalogMarksDeepSeekAsTextStreamFamily(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_PROVIDER_PRIMARY=deepseek"})

	deepseek := providerReadinessByName(t, report, "deepseek")
	if deepseek.Family != string(ProviderFamilyTextStream) {
		t.Fatalf("deepseek family = %q, want %q", deepseek.Family, ProviderFamilyTextStream)
	}
	if deepseek.Protocol != "openai_chat_completions" {
		t.Fatalf("deepseek protocol = %q", deepseek.Protocol)
	}
	if !stringSliceContains(deepseek.Capabilities, "text_stream") {
		t.Fatalf("deepseek capabilities lack text_stream: %#v", deepseek.Capabilities)
	}
}

func providerReadinessByName(t *testing.T, report ProviderCatalogReport, name string) ProviderReadiness {
	t.Helper()
	for _, readiness := range report.Providers {
		if readiness.Name == name {
			return readiness
		}
	}
	t.Fatalf("provider %q not found in %#v", name, report.Providers)
	return ProviderReadiness{}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
