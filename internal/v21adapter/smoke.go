package v21adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var smokeURLPattern = regexp.MustCompile(`https?://[^\s"']+`)

type SmokeReport struct {
	SchemaVersion    string  `json:"schema_version"`
	Adapter          string  `json:"adapter"`
	Protocol         string  `json:"protocol"`
	Status           string  `json:"status"`
	Configured       bool    `json:"configured"`
	Executed         bool    `json:"executed"`
	EndpointHost     string  `json:"endpoint_host,omitempty"`
	QueryPath        string  `json:"query_path"`
	HealthPath       string  `json:"health_path"`
	DurationMS       float64 `json:"duration_ms,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
	EvidenceCount    int     `json:"evidence_count,omitempty"`
	SpeechBlockCount int     `json:"speech_block_count,omitempty"`
	ScreenCardCount  int     `json:"screen_card_count,omitempty"`
	FollowUpCount    int     `json:"follow_up_count,omitempty"`
	ReportPath       string  `json:"report_path,omitempty"`
	Detail           string  `json:"detail,omitempty"`
}

func Smoke(ctx context.Context, adapterURL string, utterance string, execute bool, httpClient *http.Client) SmokeReport {
	report := SmokeReport{
		SchemaVersion: "a21.v21_adapter_smoke.v1",
		Adapter:       "a21-v21-adapter",
		Protocol:      "a21_v21_query",
		Status:        "failed",
		QueryPath:     QueryPath,
		HealthPath:    HealthPath,
	}
	adapterURL = strings.TrimSpace(adapterURL)
	if adapterURL == "" {
		report.Status = "skipped"
		report.Detail = "v21 adapter smoke skipped because adapter URL is missing"
		return report
	}
	report.Configured = true
	if host := smokeEndpointHost(adapterURL); host != "" {
		report.EndpointHost = host
	}
	client, err := NewHTTPClient(adapterURL)
	if err != nil {
		report.Detail = err.Error()
		return report
	}
	if httpClient == nil {
		httpClient = directHTTPClient(3 * time.Second)
	}
	client.httpClient = httpClient
	if !execute {
		report.Status = "ready"
		report.Detail = "v21 adapter smoke is configured; pass --execute to query the adapter boundary"
		return report
	}
	if strings.TrimSpace(utterance) == "" {
		utterance = "查一下语音唤醒误触发"
	}
	request := QueryRequest{
		TraceID:   "a21-trace-v21-smoke",
		SessionID: "a21-session-v21-smoke",
		Utterance: utterance,
	}
	report.Executed = true
	started := time.Now()
	response, err := client.Query(ctx, request)
	report.DurationMS = float64(time.Since(started).Microseconds()) / 1000
	if err != nil {
		report.Status = "failed"
		report.Detail = redactSmokeDetail(err.Error())
		return report
	}
	report.Status = "passed"
	report.Confidence = response.Confidence
	report.EvidenceCount = len(response.Evidence)
	report.SpeechBlockCount = len(response.SpeechBlocks)
	report.ScreenCardCount = len(response.ScreenCards)
	report.FollowUpCount = len(response.FollowUps)
	report.Detail = "v21 adapter query smoke succeeded"
	return report
}

func smokeEndpointHost(adapterURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(adapterURL))
	if err != nil || parsed.Host == "" {
		return "invalid_adapter_url"
	}
	return parsed.Host
}

func redactSmokeDetail(detail string) string {
	if strings.TrimSpace(detail) == "" {
		return ""
	}
	detail = smokeURLPattern.ReplaceAllString(detail, "<redacted-url>")
	lower := strings.ToLower(detail)
	if strings.Contains(lower, "user:") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
		return "v21 adapter smoke failed with redacted detail"
	}
	return fmt.Sprintf("%s", detail)
}
