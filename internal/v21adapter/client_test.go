package v21adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientPostsProfessionalQueryContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("path = %q, want /a21/v21/query", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.TraceID != "a21-trace-v21-001" || req.SessionID != "a21-session-v21-001" {
			t.Fatalf("ids = %q / %q", req.TraceID, req.SessionID)
		}
		if req.Mode != "professional" {
			t.Fatalf("mode = %q, want professional", req.Mode)
		}
		if req.Utterance != "查一下语音唤醒误触发" {
			t.Fatalf("utterance = %q", req.Utterance)
		}
		if req.LatencyProfile != "fast_first" || req.AnswerStyle != "voice_first_with_citations" || req.PrivacyScope != "professional_only" {
			t.Fatalf("defaults not applied: %+v", req)
		}
		if req.MaxFirstResponseMS != 1200 {
			t.Fatalf("max_first_response_ms = %d", req.MaxFirstResponseMS)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-v21-001",
			"fast_answer":"历史讨论集中在多人说话和相似音节误唤醒。",
			"confidence":0.82,
			"evidence":[{"title":"语音唤醒体验复盘","type":"meeting","source_id":"v21-doc-001","summary":"提到多人说话导致误唤醒。"}],
			"speech_blocks":["我先说结论。","第一，多人说话是主要场景。"],
			"screen_cards":[{"label":"结论","text":"误唤醒集中在 2 类场景"}],
			"follow_ups":["要不要按车型展开？"]
		}`))
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Query(context.Background(), QueryRequest{
		TraceID:   "a21-trace-v21-001",
		SessionID: "a21-session-v21-001",
		Utterance: "查一下语音唤醒误触发",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.FastAnswer == "" || len(response.Evidence) != 1 || len(response.ScreenCards) != 1 {
		t.Fatalf("response missing professional fields: %+v", response)
	}
}

func TestHTTPClientRejectsResponseMissingProfessionalEvidenceContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-v21-missing-contract",
			"fast_answer":"raw answer text that must not leak",
			"confidence":0.77,
			"speech_blocks":["raw speech block that must not leak"]
		}`))
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(context.Background(), QueryRequest{
		TraceID:   "a21-trace-v21-missing-contract",
		SessionID: "a21-session-v21-missing-contract",
		Utterance: "查一下证据",
	})
	if err == nil {
		t.Fatal("expected missing professional evidence contract to be rejected")
	}
	if !strings.Contains(err.Error(), "professional response contract") {
		t.Fatalf("error = %q, want stable professional response contract code", err.Error())
	}
	for _, forbidden := range []string{"raw answer", "raw speech", "查一下证据"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("response validation error leaked %q: %q", forbidden, err.Error())
		}
	}
}

func TestHTTPClientRejectsNonProfessionalQueryBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(context.Background(), QueryRequest{
		TraceID:      "a21-trace-v21-mode",
		SessionID:    "a21-session-v21-mode",
		Mode:         "workmate",
		Utterance:    "查一下语音唤醒误触发",
		PrivacyScope: "professional_only",
	})
	if err == nil {
		t.Fatal("expected non-professional V21 query mode to be rejected")
	}
	if calls != 0 {
		t.Fatalf("network calls = %d, want 0", calls)
	}
}

func TestValidateProfessionalQueryRequestRejectsUnsafeScopeAndEmptyUtterance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request QueryRequest
	}{
		{
			name: "public privacy",
			request: QueryRequest{
				Mode:         "professional",
				PrivacyScope: "public",
				Utterance:    "查一下语音唤醒误触发",
			},
		},
		{
			name: "empty utterance",
			request: QueryRequest{
				Mode:         "professional",
				PrivacyScope: "professional_only",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateProfessionalQueryRequest(tc.request); err == nil {
				t.Fatal("expected professional query validation error")
			}
		})
	}
}

func TestHTTPClientRejectsLegacyInternalPorts(t *testing.T) {
	_, err := NewHTTPClient("http://127.0.0.1:18080")
	if err == nil {
		t.Fatal("expected legacy V21 internal port to be rejected")
	}
	if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestHTTPClientRejectsURLCredentials(t *testing.T) {
	_, err := NewHTTPClient("http://user:secret-token@127.0.0.1:21121")
	if err == nil {
		t.Fatal("expected adapter URL credentials to be rejected")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked credential: %q", err.Error())
	}
}

func TestSmokeReportRedactsFailureURLs(t *testing.T) {
	report := Smoke(context.Background(), "http://127.0.0.1:21121/a21-adapter", "查一下语音唤醒误触发", true, &http.Client{
		Transport: failingRoundTripper{},
	})
	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	for _, forbidden := range []string{"http://127.0.0.1:21121", "/a21-adapter", "语音唤醒"} {
		if strings.Contains(report.Detail, forbidden) {
			t.Fatalf("detail leaked %q: %+v", forbidden, report)
		}
	}
}

func TestDirectHTTPClientDoesNotUseAmbientProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.invalid:8080")
	client := directHTTPClient(1500 * time.Millisecond)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("direct HTTP client must not inherit ambient proxy env")
	}
}

func TestHTTPClientDoesNotUseAmbientProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.invalid:8080")
	client, err := NewHTTPClient("http://127.0.0.1:21121")
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.httpClient.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("v21 adapter HTTP client must not inherit ambient proxy env")
	}
}

func TestMockClientReturnsDeterministicEvidence(t *testing.T) {
	client := NewMockClient()
	response, err := client.Query(context.Background(), QueryRequest{
		TraceID:   "a21-trace-mock-v21",
		SessionID: "a21-session-mock-v21",
		Utterance: "帮我查 v21",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-mock-v21" {
		t.Fatalf("trace = %q", response.TraceID)
	}
	if response.FastAnswer == "" || len(response.Evidence) == 0 || len(response.SpeechBlocks) == 0 || len(response.ScreenCards) == 0 {
		t.Fatalf("mock response missing fields: %+v", response)
	}
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed for " + req.URL.String())
}
