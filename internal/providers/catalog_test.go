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
	if !deepseek.Selected {
		t.Fatal("deepseek should be selected")
	}
	if deepseek.Family != string(ProviderFamilyTextStream) {
		t.Fatalf("deepseek family = %q, want %q", deepseek.Family, ProviderFamilyTextStream)
	}
	if deepseek.Protocol != "openai_chat_completions" {
		t.Fatalf("deepseek protocol = %q", deepseek.Protocol)
	}
	if !stringSliceContains(deepseek.Capabilities, "text_stream") {
		t.Fatalf("deepseek capabilities lack text_stream: %#v", deepseek.Capabilities)
	}
	if deepseek.Configured {
		t.Fatal("deepseek should require the lab API key before it is configured")
	}
	if !stringSliceContains(deepseek.MissingEnv, "A21_LAB_DEEPSEEK_API_KEY") {
		t.Fatalf("deepseek missing env lacks lab key: %#v", deepseek.MissingEnv)
	}
	if stringSliceContains(deepseek.MissingEnv, "A21_DEEPSEEK_MODEL") {
		t.Fatalf("deepseek model should use a default for P0 smoke: %#v", deepseek.MissingEnv)
	}
}

func TestProviderCatalogBlocksBaiduAndHuaweiPrimaryWithoutEchoingValue(t *testing.T) {
	for _, primary := range []string{"baidu_qianfan", "huawei_pangu"} {
		report := ProviderCatalogFromEnv([]string{"A21_PROVIDER_PRIMARY=" + primary})
		if report.Primary != "blocked_provider" {
			t.Fatalf("primary = %q, want blocked_provider for %q", report.Primary, primary)
		}
		if len(report.Findings) != 1 || report.Findings[0].Code != "provider_blocked" {
			t.Fatalf("findings = %#v, want provider_blocked", report.Findings)
		}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), primary) {
			t.Fatalf("catalog leaked blocked provider value: %s", data)
		}
	}
}

func TestProviderCatalogP0BuiltInsStayNarrow(t *testing.T) {
	report := ProviderCatalogFromEnv([]string{"A21_ENV=development"})
	var names []string
	for _, provider := range report.Providers {
		names = append(names, provider.Name)
	}
	for _, want := range []string{"mock", "deepseek"} {
		if !stringSliceContains(names, want) {
			t.Fatalf("providers = %#v, missing %q", names, want)
		}
	}
	for _, forbidden := range []string{"siliconflow", "stepfun", "bailian_dashscope", "moonshot", "volcengine_ark", "local_ollama", "local_vllm", "openai_realtime", "doubao_realtime", "doubao_tts_realtime", "hermes_agent", "mimo_agent"} {
		if stringSliceContains(names, forbidden) {
			t.Fatalf("P0 catalog horizontally expanded to %q: %#v", forbidden, names)
		}
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
