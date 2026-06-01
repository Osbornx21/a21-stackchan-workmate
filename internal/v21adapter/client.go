package v21adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const QueryPath = "/a21/v21/query"
const HealthPath = "/healthz"
const ProfessionalMaxFirstResponseMS = 1200
const ProfessionalCheckingFeedbackText = "我在查，先把证据和置信度拉出来。"

type QueryFailureClass string

const (
	QueryFailureUnknown         QueryFailureClass = ""
	QueryFailureTimeout         QueryFailureClass = "timeout"
	QueryFailureContractInvalid QueryFailureClass = "contract_invalid"
	QueryFailureUpstreamStatus  QueryFailureClass = "upstream_status"
	QueryFailureNoEvidence      QueryFailureClass = "no_evidence"
	QueryFailureAdapterStatus   QueryFailureClass = "adapter_status"
	QueryFailureTransport       QueryFailureClass = "transport_error"
)

type QueryFailure struct {
	Class       QueryFailureClass
	StatusCode  int
	StatusClass string
}

func (e *QueryFailure) Error() string {
	if e == nil {
		return "v21 adapter query failed"
	}
	switch e.Class {
	case QueryFailureContractInvalid:
		return "v21 adapter professional response contract invalid"
	case QueryFailureTimeout:
		return "v21 adapter query timed out"
	case QueryFailureNoEvidence:
		return "v21 adapter query completed without evidence"
	case QueryFailureUpstreamStatus:
		if e.StatusClass != "" {
			return "v21 adapter upstream status failure: " + e.StatusClass
		}
		return "v21 adapter upstream status failure"
	case QueryFailureAdapterStatus:
		if e.StatusClass != "" {
			return "v21 adapter status failure: " + e.StatusClass
		}
		return "v21 adapter status failure"
	case QueryFailureTransport:
		return "v21 adapter transport failure"
	default:
		return "v21 adapter query failed"
	}
}

func QueryFailureClassOf(err error) QueryFailureClass {
	var failure *QueryFailure
	if errors.As(err, &failure) && failure != nil {
		return failure.Class
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return QueryFailureTimeout
	}
	return QueryFailureUnknown
}

func QueryFailureStatusClassOf(err error) string {
	var failure *QueryFailure
	if errors.As(err, &failure) && failure != nil {
		return failure.StatusClass
	}
	return ""
}

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

type ProfessionalBridgeReceipt struct {
	SchemaVersion      string `json:"schema_version"`
	TraceID            string `json:"trace_id,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	Mode               string `json:"mode"`
	Status             string `json:"status"`
	Text               string `json:"text"`
	MaxFirstResponseMS int    `json:"max_first_response_ms"`
	EvidenceCompleted  bool   `json:"evidence_completed"`
}

type ProfessionalBridgeEvidenceReport struct {
	SchemaVersion     string                            `json:"schema_version"`
	TraceID           string                            `json:"trace_id,omitempty"`
	Status            string                            `json:"status"`
	EvidenceCompleted bool                              `json:"evidence_completed"`
	ConfidencePresent bool                              `json:"confidence_present"`
	EvidenceCount     int                               `json:"evidence_count"`
	SpeechBlockCount  int                               `json:"speech_block_count"`
	ScreenCardCount   int                               `json:"screen_card_count"`
	FollowUpCount     int                               `json:"follow_up_count"`
	EvidenceCards     []ProfessionalBridgeEvidenceCard  `json:"evidence_cards,omitempty"`
	ScreenCards       []ProfessionalBridgeScreenCard    `json:"screen_cards,omitempty"`
	Redaction         ProfessionalBridgeRedactionStatus `json:"redaction"`
}

type ProfessionalBridgeEvidenceCard struct {
	TitlePresent    bool   `json:"title_present"`
	Type            string `json:"type,omitempty"`
	SourceIDPresent bool   `json:"source_id_present"`
	SummaryPresent  bool   `json:"summary_present"`
	QuotePresent    bool   `json:"quote_present"`
}

type ProfessionalBridgeScreenCard struct {
	LabelPresent bool `json:"label_present"`
	TextPresent  bool `json:"text_present"`
}

type ProfessionalBridgeRedactionStatus struct {
	FastAnswerStored     bool `json:"fast_answer_stored"`
	EvidenceBodyStored   bool `json:"evidence_body_stored"`
	ScreenTextStored     bool `json:"screen_text_stored"`
	SpeechBlocksStored   bool `json:"speech_blocks_stored"`
	FollowUpsStored      bool `json:"follow_ups_stored"`
	FullURLStored        bool `json:"full_url_stored"`
	LocalPathStored      bool `json:"local_path_stored"`
	ProviderOutputStored bool `json:"provider_output_stored"`
}

type ProfessionalBridgeReadiness struct {
	SchemaVersion             string `json:"schema_version"`
	Configured                bool   `json:"configured"`
	ContractReady             bool   `json:"contract_ready"`
	Status                    string `json:"status"`
	QueryPath                 string `json:"query_path"`
	HealthPath                string `json:"health_path"`
	MaxFirstResponseMS        int    `json:"max_first_response_ms"`
	CheckingFeedbackSupported bool   `json:"checking_feedback_supported"`
	EvidenceContractReady     bool   `json:"evidence_contract_ready"`
	QueryExecuted             bool   `json:"query_executed"`
	Detail                    string `json:"detail,omitempty"`
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
	if err := ValidateProfessionalQueryRequest(request); err != nil {
		return QueryResponse{}, &QueryFailure{Class: QueryFailureContractInvalid}
	}
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
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return QueryResponse{}, &QueryFailure{Class: QueryFailureTimeout}
		}
		return QueryResponse{}, &QueryFailure{Class: QueryFailureTransport}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return QueryResponse{}, queryFailureForStatus(resp.StatusCode)
	}
	var response QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return QueryResponse{}, &QueryFailure{Class: QueryFailureContractInvalid}
	}
	if response.TraceID == "" {
		response.TraceID = request.TraceID
	}
	if err := ValidateProfessionalQueryResponse(response); err != nil {
		return QueryResponse{}, &QueryFailure{Class: QueryFailureContractInvalid}
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

func NewProfessionalBridgeReceipt(request QueryRequest) (ProfessionalBridgeReceipt, error) {
	if err := ValidateProfessionalQueryRequest(request); err != nil {
		return ProfessionalBridgeReceipt{}, err
	}
	request = withDefaults(request)
	return ProfessionalBridgeReceipt{
		SchemaVersion:      "a21.v21_professional_bridge_receipt.v1",
		TraceID:            request.TraceID,
		SessionID:          request.SessionID,
		Mode:               request.Mode,
		Status:             "checking",
		Text:               ProfessionalCheckingFeedbackText,
		MaxFirstResponseMS: request.MaxFirstResponseMS,
		EvidenceCompleted:  false,
	}, nil
}

func NewProfessionalBridgeEvidenceReport(response QueryResponse) (ProfessionalBridgeEvidenceReport, error) {
	if err := ValidateProfessionalQueryResponse(response); err != nil {
		return ProfessionalBridgeEvidenceReport{}, err
	}
	report := ProfessionalBridgeEvidenceReport{
		SchemaVersion:     "a21.v21_professional_bridge_evidence.v1",
		TraceID:           response.TraceID,
		Status:            "evidence_completed",
		EvidenceCompleted: true,
		ConfidencePresent: true,
		EvidenceCount:     len(response.Evidence),
		SpeechBlockCount:  len(response.SpeechBlocks),
		ScreenCardCount:   len(response.ScreenCards),
		FollowUpCount:     len(response.FollowUps),
		Redaction: ProfessionalBridgeRedactionStatus{
			FastAnswerStored:     false,
			EvidenceBodyStored:   false,
			ScreenTextStored:     false,
			SpeechBlocksStored:   false,
			FollowUpsStored:      false,
			FullURLStored:        false,
			LocalPathStored:      false,
			ProviderOutputStored: false,
		},
	}
	for _, item := range response.Evidence {
		report.EvidenceCards = append(report.EvidenceCards, ProfessionalBridgeEvidenceCard{
			TitlePresent:    strings.TrimSpace(item.Title) != "",
			Type:            redactedEvidenceType(item.Type),
			SourceIDPresent: strings.TrimSpace(item.SourceID) != "",
			SummaryPresent:  strings.TrimSpace(item.Summary) != "",
			QuotePresent:    strings.TrimSpace(item.Quote) != "",
		})
	}
	for _, card := range response.ScreenCards {
		report.ScreenCards = append(report.ScreenCards, ProfessionalBridgeScreenCard{
			LabelPresent: strings.TrimSpace(card.Label) != "",
			TextPresent:  strings.TrimSpace(card.Text) != "",
		})
	}
	return report, nil
}

func NewProfessionalBridgeReadiness(adapterURL string) ProfessionalBridgeReadiness {
	report := ProfessionalBridgeReadiness{
		SchemaVersion:             "a21.v21_professional_bridge_readiness.v1",
		QueryPath:                 QueryPath,
		HealthPath:                HealthPath,
		MaxFirstResponseMS:        ProfessionalMaxFirstResponseMS,
		CheckingFeedbackSupported: true,
		EvidenceContractReady:     true,
		QueryExecuted:             false,
	}
	if strings.TrimSpace(adapterURL) == "" {
		report.Status = "disabled"
		return report
	}
	report.Configured = true
	if _, err := NewHTTPClient(adapterURL); err != nil {
		report.Status = "misconfigured"
		report.Detail = redactSmokeDetail(err.Error())
		return report
	}
	report.Status = "configured"
	report.ContractReady = true
	return report
}

func directHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: nil},
	}
}

func queryFailureForStatus(status int) *QueryFailure {
	statusClass := statusClass(status)
	class := QueryFailureAdapterStatus
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		class = QueryFailureContractInvalid
	case http.StatusFailedDependency:
		class = QueryFailureNoEvidence
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		class = QueryFailureUpstreamStatus
	}
	return &QueryFailure{Class: class, StatusCode: status, StatusClass: statusClass}
}

func statusClass(status int) string {
	switch {
	case status >= 100 && status < 200:
		return "status_1xx"
	case status >= 200 && status < 300:
		return "status_2xx"
	case status >= 300 && status < 400:
		return "status_3xx"
	case status >= 400 && status < 500:
		return "status_4xx"
	case status >= 500 && status < 600:
		return "status_5xx"
	default:
		return ""
	}
}

type MockClient struct{}

func NewMockClient() MockClient {
	return MockClient{}
}

func (MockClient) Query(ctx context.Context, request QueryRequest) (QueryResponse, error) {
	if err := ValidateProfessionalQueryRequest(request); err != nil {
		return QueryResponse{}, err
	}
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

func ValidateProfessionalQueryRequest(request QueryRequest) error {
	if strings.TrimSpace(request.Utterance) == "" {
		return fmt.Errorf("v21 professional query utterance is required")
	}
	if mode := strings.TrimSpace(request.Mode); mode != "" && mode != "professional" {
		return fmt.Errorf("v21 adapter accepts only professional mode")
	}
	if privacyScope := strings.TrimSpace(request.PrivacyScope); privacyScope != "" && privacyScope != "professional_only" {
		return fmt.Errorf("v21 adapter accepts only professional_only privacy scope")
	}
	return nil
}

func ValidateProfessionalQueryResponse(response QueryResponse) error {
	switch {
	case strings.TrimSpace(response.FastAnswer) == "":
		return fmt.Errorf("v21 adapter professional response contract invalid: missing fast_answer")
	case response.Confidence < 0 || response.Confidence > 1:
		return fmt.Errorf("v21 adapter professional response contract invalid: confidence out of range")
	case len(response.Evidence) == 0:
		return fmt.Errorf("v21 adapter professional response contract invalid: missing evidence")
	case len(response.SpeechBlocks) == 0:
		return fmt.Errorf("v21 adapter professional response contract invalid: missing speech_blocks")
	case len(response.ScreenCards) == 0:
		return fmt.Errorf("v21 adapter professional response contract invalid: missing screen_cards")
	case len(response.FollowUps) == 0:
		return fmt.Errorf("v21 adapter professional response contract invalid: missing follow_ups")
	default:
		return nil
	}
}

func withDefaults(request QueryRequest) QueryRequest {
	if request.Mode == "" {
		request.Mode = "professional"
	}
	if request.LatencyProfile == "" {
		request.LatencyProfile = "fast_first"
	}
	if request.AnswerStyle == "" {
		request.AnswerStyle = "voice_first_with_citations"
	}
	if request.MaxFirstResponseMS == 0 {
		request.MaxFirstResponseMS = ProfessionalMaxFirstResponseMS
	}
	if request.PrivacyScope == "" {
		request.PrivacyScope = "professional_only"
	}
	return request
}

func redactedEvidenceType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if len(value) > 48 {
		value = value[:48]
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "other"
	}
	return value
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
