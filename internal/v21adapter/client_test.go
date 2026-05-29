package v21adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestHTTPClientRejectsLegacyInternalPorts(t *testing.T) {
	_, err := NewHTTPClient("http://127.0.0.1:18080")
	if err == nil {
		t.Fatal("expected legacy V21 internal port to be rejected")
	}
	if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("error = %q", err.Error())
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
