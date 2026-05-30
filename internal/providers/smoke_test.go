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
	for _, want := range []string{"A21_LAB_DEEPSEEK_API_KEY"} {
		if !stringSliceContains(report.MissingEnv, want) {
			t.Fatalf("missing env lacks %q: %#v", want, report.MissingEnv)
		}
	}
	if stringSliceContains(report.MissingEnv, "A21_DEEPSEEK_MODEL") {
		t.Fatalf("model should use DeepSeek default for P0 smoke: %#v", report.MissingEnv)
	}
}

func TestProviderSmokeDryRunReportsReadyWithoutLeakingSecrets(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
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
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat"} {
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
		sawModel = body.Model == "deepseek-chat"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer server.Close()

	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
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
	if strings.Contains(string(data), "sk-a21-secret") || strings.Contains(string(data), "deepseek-chat") {
		t.Fatalf("smoke report leaked secret or model: %s", data)
	}
}

func TestProviderSmokeExecutesOpenAICompatibleStreamingRequest(t *testing.T) {
	var sawStream bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		sawStream = body.Stream && body.Model == "deepseek-chat"
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning":"先想"}}]}`,
			`data: {"choices":[{"delta":{"content":"OK"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	defer server.Close()

	report := ProviderSmokeFromEnvWithOptions(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_DEEPSEEK_BASE_URL=" + server.URL,
	}, ProviderSmokeOptions{
		ProviderName: "deepseek",
		Execute:      true,
		Stream:       true,
		Repeat:       2,
		Client:       server.Client(),
	})

	if report.Status != ProviderSmokePassed {
		t.Fatalf("status = %q, detail = %q", report.Status, report.Detail)
	}
	if !sawStream {
		t.Fatal("server did not receive stream request with configured model")
	}
	if report.Family != string(ProviderFamilyTextStream) || !report.Stream || report.Repeat != 2 {
		t.Fatalf("family/stream/repeat = %q/%v/%d", report.Family, report.Stream, report.Repeat)
	}
	if len(report.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(report.Attempts))
	}
	for _, attempt := range report.Attempts {
		if attempt.FirstByteMS <= 0 || attempt.FirstContentMS <= 0 || attempt.TotalDurationMS <= 0 {
			t.Fatalf("attempt timings not populated: %+v", attempt)
		}
		if attempt.ContentDeltaCount != 1 || attempt.ReasoningDeltaCount != 1 || !attempt.Done {
			t.Fatalf("attempt stream counts = %+v", attempt)
		}
	}
	if report.TimingSummary == nil || report.TimingSummary.FirstByteP50MS <= 0 || report.TimingSummary.FirstContentP95MS <= 0 {
		t.Fatalf("timing summary not populated: %+v", report.TimingSummary)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(data)
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat", "先想", "OK"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("stream report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestProviderSmokeStreamingHTTPFailureReportsFallbackTraceMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `bad key sk-a21-secret for deepseek-chat`, http.StatusUnauthorized)
	}))
	defer server.Close()

	report := ProviderSmokeFromEnvWithOptions(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-secret",
		"A21_DEEPSEEK_BASE_URL=" + server.URL,
	}, ProviderSmokeOptions{
		ProviderName: "deepseek",
		Execute:      true,
		Stream:       true,
		Repeat:       1,
		Client:       server.Client(),
	})

	if report.Status != ProviderSmokeFailed {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	if report.Fallback == nil || !report.Fallback.Activated || report.Fallback.Provider != "mock" {
		t.Fatalf("fallback = %+v, want activated mock", report.Fallback)
	}
	if report.TraceID == "" || len(report.TraceMarkers) == 0 || len(report.Metrics) == 0 {
		t.Fatalf("trace/metrics missing: trace_id=%q markers=%+v metrics=%+v", report.TraceID, report.TraceMarkers, report.Metrics)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(data)
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat", "bad key"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("failure report leaked %q: %s", forbidden, rendered)
		}
	}
	if !strings.Contains(report.Detail, "body_sha256=") {
		t.Fatalf("detail missing redacted body hash: %q", report.Detail)
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

func TestProviderSmokeDoesNotExposeRealtimeProvidersDuringP0(t *testing.T) {
	report := ProviderSmokeFromEnv(context.Background(), []string{
		"A21_PROVIDER_PRIMARY=openai_realtime",
		"A21_OPENAI_API_KEY=sk-a21-secret",
		"A21_OPENAI_REALTIME_MODEL=gpt-realtime",
	}, "openai_realtime", true, nil)

	if report.Provider != "unknown_provider" || report.Status != ProviderSmokeFailed {
		t.Fatalf("provider/status = %q/%q, want unknown_provider/failed", report.Provider, report.Status)
	}
	if report.Executed {
		t.Fatal("executed = true, want false for non-P0 provider")
	}
}
