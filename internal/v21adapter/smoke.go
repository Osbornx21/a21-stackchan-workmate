package v21adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var smokeURLPattern = regexp.MustCompile(`https?://[^\s"']+`)

const SmokeSchemaVersion = "a21.v21_adapter_smoke.v1"

type SmokeReport struct {
	SchemaVersion      string  `json:"schema_version"`
	GeneratedAtMS      int64   `json:"generated_at_ms"`
	Adapter            string  `json:"adapter"`
	Protocol           string  `json:"protocol"`
	Status             string  `json:"status"`
	Configured         bool    `json:"configured"`
	Executed           bool    `json:"executed"`
	EndpointHost       string  `json:"endpoint_host,omitempty"`
	Mode               string  `json:"mode,omitempty"`
	LatencyProfile     string  `json:"latency_profile,omitempty"`
	AnswerStyle        string  `json:"answer_style,omitempty"`
	PrivacyScope       string  `json:"privacy_scope,omitempty"`
	MaxFirstResponseMS int     `json:"max_first_response_ms,omitempty"`
	QueryPath          string  `json:"query_path"`
	HealthPath         string  `json:"health_path"`
	DurationMS         float64 `json:"duration_ms,omitempty"`
	Confidence         float64 `json:"confidence,omitempty"`
	EvidenceCount      int     `json:"evidence_count,omitempty"`
	SpeechBlockCount   int     `json:"speech_block_count,omitempty"`
	ScreenCardCount    int     `json:"screen_card_count,omitempty"`
	FollowUpCount      int     `json:"follow_up_count,omitempty"`
	FailureClass       string  `json:"failure_class,omitempty"`
	StatusClass        string  `json:"status_class,omitempty"`
	RedactionOK        bool    `json:"redaction_ok"`
	ReportPath         string  `json:"report_path,omitempty"`
	Detail             string  `json:"detail,omitempty"`
}

func Smoke(ctx context.Context, adapterURL string, utterance string, execute bool, httpClient *http.Client) SmokeReport {
	report := SmokeReport{
		SchemaVersion: SmokeSchemaVersion,
		GeneratedAtMS: time.Now().UnixMilli(),
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
		return finalizedSmokeReport(report)
	}
	report.Configured = true
	if host := smokeEndpointHost(adapterURL); host != "" {
		report.EndpointHost = host
	}
	client, err := NewHTTPClient(adapterURL)
	if err != nil {
		report.Detail = err.Error()
		return finalizedSmokeReport(report)
	}
	if httpClient == nil {
		httpClient = directHTTPClient(3 * time.Second)
	}
	client.httpClient = httpClient
	if !execute {
		report.Status = "ready"
		report.Detail = "v21 adapter smoke is configured; pass --execute to query the adapter boundary"
		return finalizedSmokeReport(report)
	}
	if strings.TrimSpace(utterance) == "" {
		utterance = "查一下语音唤醒误触发"
	}
	request := withDefaults(QueryRequest{
		TraceID:   "a21-trace-v21-smoke",
		SessionID: "a21-session-v21-smoke",
		Utterance: utterance,
	})
	report.Mode = request.Mode
	report.LatencyProfile = request.LatencyProfile
	report.AnswerStyle = request.AnswerStyle
	report.PrivacyScope = request.PrivacyScope
	report.MaxFirstResponseMS = request.MaxFirstResponseMS
	report.Executed = true
	started := time.Now()
	response, err := client.Query(ctx, request)
	report.DurationMS = float64(time.Since(started).Microseconds()) / 1000
	if err != nil {
		report.Status = "failed"
		report.FailureClass = string(QueryFailureClassOf(err))
		report.StatusClass = QueryFailureStatusClassOf(err)
		report.Detail = redactSmokeDetail(err.Error())
		return finalizedSmokeReport(report)
	}
	report.Status = "passed"
	report.Confidence = response.Confidence
	report.EvidenceCount = len(response.Evidence)
	report.SpeechBlockCount = len(response.SpeechBlocks)
	report.ScreenCardCount = len(response.ScreenCards)
	report.FollowUpCount = len(response.FollowUps)
	report.Detail = "v21 adapter query smoke succeeded"
	return finalizedSmokeReport(report)
}

func finalizedSmokeReport(report SmokeReport) SmokeReport {
	finalizeSmokeReport(&report)
	return report
}

func finalizeSmokeReport(report *SmokeReport) {
	if report == nil {
		return
	}
	if report.GeneratedAtMS <= 0 {
		report.GeneratedAtMS = time.Now().UnixMilli()
	}
	report.RedactionOK = smokeReportRedactionOK(*report)
}

func smokeReportRedactionOK(report SmokeReport) bool {
	encoded, err := json.Marshal(report)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(encoded))
	for _, forbidden := range []string{
		"查一下语音唤醒误触发",
		"历史讨论",
		"raw retrieved evidence",
		"raw prompt text",
		"raw transcript text",
		"raw provider output",
		"raw reasoning text",
		"http://",
		"https://",
		"/users/",
		"api_key",
		"secret-token",
		"bearer ",
		"provider output",
		"reasoning",
	} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
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
