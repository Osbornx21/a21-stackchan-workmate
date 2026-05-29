package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderSmokeSkipsMissingDeepSeekCredentials(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{"A21_PROVIDER_PRIMARY=deepseek"}, "deepseek", false, nil)

	if report.Provider != "deepseek" {
		t.Fatalf("provider = %q, want deepseek", report.Provider)
	}
	if report.Status != ProviderSmokeSkipped {
		t.Fatalf("status = %q, want skipped", report.Status)
	}
	if report.Configured {
		t.Fatal("configured = true, want false")
	}
	for _, want := range []string{"A21_DEEPSEEK_API_KEY", "A21_DEEPSEEK_MODEL"} {
		if !stringSliceContains(report.MissingEnv, want) {
			t.Fatalf("missing env lacks %q: %#v", want, report.MissingEnv)
		}
	}
}

func TestProviderSmokeDryRunReportsReadyWithoutLeakingSecrets(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_DEEPSEEK_MODEL=deepseek-v4-flash",
	}, "deepseek", false, nil)

	if report.Status != ProviderSmokeReady {
		t.Fatalf("status = %q, want ready", report.Status)
	}
	if !report.Configured || report.Executed {
		t.Fatalf("configured/executed = %v/%v, want true/false", report.Configured, report.Executed)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(data)
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-v4-flash"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("smoke report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestProviderSmokeExecutesOpenAICompatibleRequest(t *testing.T) {
	var sawAuth bool
	var sawModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		sawAuth = r.Header.Get("Authorization") == "Bearer sk-a21-secret"
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		sawModel = body.Model == "deepseek-v4-flash"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer server.Close()

	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_DEEPSEEK_MODEL=deepseek-v4-flash",
		"A21_DEEPSEEK_BASE_URL=" + server.URL,
	}, "deepseek", true, server.Client())

	if report.Status != ProviderSmokePassed {
		t.Fatalf("status = %q, detail = %q", report.Status, report.Detail)
	}
	if !report.Executed || report.HTTPStatus != 200 {
		t.Fatalf("executed/http = %v/%d", report.Executed, report.HTTPStatus)
	}
	if !sawAuth || !sawModel {
		t.Fatalf("saw auth/model = %v/%v, want true/true", sawAuth, sawModel)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-a21-secret") || strings.Contains(string(data), "deepseek-v4-flash") {
		t.Fatalf("smoke report leaked secret or model: %s", data)
	}
}

func TestProviderSmokeRedactsLegacyProviderName(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{"A21_PROVIDER_PRIMARY=x21_voice"}, "x21_voice", false, nil)

	if report.Provider != "invalid_legacy_provider" {
		t.Fatalf("provider = %q, want invalid_legacy_provider", report.Provider)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "x21_voice") {
		t.Fatalf("smoke report leaked legacy provider: %s", data)
	}
}

func TestProviderSmokeMarksRealtimeProvidersUnsupportedForHTTP(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime",
	}, "openai_realtime", true, nil)

	if report.Status != ProviderSmokeUnsupported {
		t.Fatalf("status = %q, want unsupported", report.Status)
	}
	if report.Executed {
		t.Fatal("executed = true, want false for realtime websocket provider")
	}
}
