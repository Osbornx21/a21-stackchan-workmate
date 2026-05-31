package v21adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const QueryPath = "/a21/v21/query"
const HealthPath = "/healthz"

type Client interface {
	Query(ctx context.Context, request QueryRequest) (QueryResponse, error)
}

type QueryRequest struct {
	TraceID            string `json:"trace_id"`
	SessionID          string `json:"session_id"`
	Mode               string `json:"mode"`
	Utterance          string `json:"utterance"`
	LatencyProfile     string `json:"latency_profile"`
	AnswerStyle        string `json:"answer_style"`
	MaxFirstResponseMS int    `json:"max_first_response_ms"`
	PrivacyScope       string `json:"privacy_scope"`
}

type Evidence struct {
	Title    string `json:"title"`
	Type     string `json:"type"`
	SourceID string `json:"source_id"`
	Summary  string `json:"summary"`
	Quote    string `json:"quote,omitempty"`
}

type ScreenCard struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type QueryResponse struct {
	TraceID      string       `json:"trace_id"`
	FastAnswer   string       `json:"fast_answer"`
	Confidence   float64      `json:"confidence"`
	Evidence     []Evidence   `json:"evidence"`
	SpeechBlocks []string     `json:"speech_blocks"`
	ScreenCards  []ScreenCard `json:"screen_cards"`
	FollowUps    []string     `json:"follow_ups"`
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) (*HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("v21 adapter endpoint must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("v21 adapter endpoint host is required")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("v21 adapter endpoint must not include credentials")
	}
	if endpointLooksUnsafe(parsed) {
		return nil, fmt.Errorf("v21 adapter endpoint must target the A21 adapter boundary, not a legacy internal service")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return &HTTPClient{baseURL: parsed.String(), httpClient: directHTTPClient(0)}, nil
}

func (c *HTTPClient) Query(ctx context.Context, request QueryRequest) (QueryResponse, error) {
	request = withDefaults(request)
	body, err := json.Marshal(request)
	if err != nil {
		return QueryResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+QueryPath, bytes.NewReader(body))
	if err != nil {
		return QueryResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return QueryResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return QueryResponse{}, fmt.Errorf("v21 adapter query failed with status %d", resp.StatusCode)
	}
	var response QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return QueryResponse{}, err
	}
	if response.TraceID == "" {
		response.TraceID = request.TraceID
	}
	return response, nil
}

func ProbeHealth(ctx context.Context, baseURL string, httpClient *http.Client) error {
	client, err := NewHTTPClient(baseURL)
	if err != nil {
		return err
	}
	if httpClient == nil {
		httpClient = directHTTPClient(1500 * time.Millisecond)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+HealthPath, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("v21 adapter health failed with status %d", resp.StatusCode)
	}
	return nil
}

func directHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: nil},
	}
}

type MockClient struct{}

func NewMockClient() MockClient {
	return MockClient{}
}

func (MockClient) Query(ctx context.Context, request QueryRequest) (QueryResponse, error) {
	request = withDefaults(request)
	return QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "历史讨论主要集中在多人说话、相似音节误唤醒和连续对话残留监听三个场景。",
		Confidence: 0.82,
		Evidence: []Evidence{
			{
				Title:    "语音唤醒体验复盘",
				Type:     "meeting",
				SourceID: "v21-doc-mock-001",
				Summary:  "提到多人说话和相似音节会导致误唤醒。",
			},
		},
		SpeechBlocks: []string{
			"我先说结论。历史讨论主要集中在三个场景。",
			"第一，多人说话会提高误触发风险。",
			"第二，相似音节和连续对话残留监听也需要重点看。",
		},
		ScreenCards: []ScreenCard{
			{Label: "结论", Text: "误唤醒集中在 3 类场景"},
			{Label: "来源", Text: "会议纪要 / PPI / 打点表"},
		},
		FollowUps: []string{
			"要不要按车型展开？",
			"要不要查对应埋点口径？",
		},
	}, nil
}

func withDefaults(request QueryRequest) QueryRequest {
	request.Mode = "professional"
	if request.LatencyProfile == "" {
		request.LatencyProfile = "fast_first"
	}
	if request.AnswerStyle == "" {
		request.AnswerStyle = "voice_first_with_citations"
	}
	if request.MaxFirstResponseMS == 0 {
		request.MaxFirstResponseMS = 1200
	}
	if request.PrivacyScope == "" {
		request.PrivacyScope = "professional_only"
	}
	return request
}

func endpointLooksUnsafe(parsed *url.URL) bool {
	lower := strings.ToLower(parsed.String())
	if strings.Contains(lower, "x21") {
		return true
	}
	switch parsed.Port() {
	case "8000", "8080", "10095", "18080", "4173", "42173", "16686", "16687":
		return true
	default:
		return false
	}
}
