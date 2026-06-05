package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
)

func fetchXiaozhiProfessionalBenchReadRecord(ctx context.Context, gatewayURL string, traceID string, sessionID string, deviceID string) xiaozhiProfessionalBenchReadRecord {
	query := make(url.Values)
	query.Set("trace_id", traceID)
	endpoint, _, err := firmwareGatewayEndpoint(gatewayURL, "/v1/professional-read-records", query)
	if err != nil {
		return xiaozhiProfessionalBenchReadRecord{}
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xiaozhiProfessionalBenchReadRecord{}
	}
	client := *a21DirectHTTPClient(2 * time.Second)
	response, err := client.Do(request)
	if err != nil {
		return xiaozhiProfessionalBenchReadRecord{}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return xiaozhiProfessionalBenchReadRecord{}
	}
	var records gateway.ProfessionalReadRecordsResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 32768)).Decode(&records); err != nil {
		return xiaozhiProfessionalBenchReadRecord{}
	}
	if records.SchemaVersion != gateway.ProfessionalReadRecordsSchemaVersion || records.Status != "ok" {
		return xiaozhiProfessionalBenchReadRecord{
			RecordCount: len(records.Records),
			Redaction:   xiaozhiProfessionalBenchReadRecordRedactionFromGateway(records.Redaction),
		}
	}
	summary := xiaozhiProfessionalBenchReadRecord{
		Observed:    len(records.Records) > 0,
		RecordCount: len(records.Records),
		Redaction:   xiaozhiProfessionalBenchReadRecordRedactionFromGateway(records.Redaction),
	}
	if len(records.Records) != 1 {
		return summary
	}
	record := records.Records[0]
	summary.Completed = record.Status == "completed"
	summary.Status = record.Status
	summary.RecordID = record.RecordID
	summary.TraceIDMatched = record.TraceID == traceID
	summary.SessionIDMatched = record.SessionID == sessionID
	summary.DeviceIDMatched = record.DeviceID == deviceID
	summary.QueryScope = record.QueryScope
	summary.PrivacyScope = record.PrivacyScope
	summary.LatencyProfile = record.LatencyProfile
	summary.AnswerStyle = record.AnswerStyle
	summary.UtteranceBucket = record.UtteranceBucket
	summary.SourceScopeCounts = copyStringIntMap(record.SourceScopeCounts)
	summary.WorkspaceStatus = record.WorkspaceStatus
	summary.Redaction = xiaozhiProfessionalBenchReadRecordRedactionFromGateway(record.Redaction)
	return summary
}

func xiaozhiProfessionalBenchReadRecordRedactionFromGateway(redaction gateway.ProfessionalWorkspaceRedaction) xiaozhiProfessionalBenchReadRecordRedact {
	return xiaozhiProfessionalBenchReadRecordRedact{
		DocumentTextStored:    redaction.DocumentTextStored,
		QueryTextStored:       redaction.QueryTextStored,
		RetrievedTextStored:   redaction.RetrievedTextStored,
		FullURLStored:         redaction.FullURLStored,
		LocalPathStored:       redaction.LocalPathStored,
		CredentialValueStored: redaction.CredentialValueStored,
		ProviderOutputStored:  redaction.ProviderOutputStored,
		VoiceTextStored:       redaction.VoiceTranscriptStored,
	}
}

func xiaozhiProfessionalBenchOpusPackets(options xiaozhiProfessionalBenchOptions) ([][]byte, xiaozhiVoiceBenchInput, error) {
	packets, input, err := xiaozhiVoiceBenchOpusPackets(xiaozhiVoiceBenchOptions{InputWAV: options.InputWAV})
	if err != nil {
		return nil, xiaozhiVoiceBenchInput{}, err
	}
	if input.Source == "synthetic_sine" {
		input.Source = "synthetic_opus"
	}
	return packets, input, nil
}

func xiaozhiProfessionalBenchFloatField(values map[string]any, key string) float64 {
	if values == nil {
		return 0
	}
	value, _ := values[key].(float64)
	return value
}

func xiaozhiProfessionalBenchBoolField(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, _ := values[key].(bool)
	return value
}

func xiaozhiProfessionalBenchReportOmitsSensitiveText(report xiaozhiProfessionalBenchReport) bool {
	data, err := json.Marshal(report)
	if err != nil {
		return false
	}
	rendered := string(data)
	for _, forbidden := range []string{
		xiaozhiProfessionalBenchASRSentinel,
		xiaozhiProfessionalBenchPlaceholderUtterance,
		"RAW_SECRET",
		"http://",
		"https://",
		"data_base64",
		"provider output",
		"reasoning",
	} {
		if strings.Contains(rendered, forbidden) {
			return false
		}
	}
	return true
}

func writeXiaozhiProfessionalBenchReport(outputDir string, report xiaozhiProfessionalBenchReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-professional-bench-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiProfessionalBench(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiProfessionalBench(writer io.Writer, report xiaozhiProfessionalBenchReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

type xiaozhiProfessionalBenchASRAdapter struct {
	text string
}

func (a xiaozhiProfessionalBenchASRAdapter) Name() string {
	return "a21-host-mock-professional-asr"
}

func (a xiaozhiProfessionalBenchASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.ASRAdapterEvent{Text: a.text, Final: true}:
		}
	}()
	return out, nil
}

type xiaozhiProfessionalBenchTextStreamAdapter struct{}

func (xiaozhiProfessionalBenchTextStreamAdapter) Name() string {
	return "a21-host-mock-professional-text-stream"
}

func (xiaozhiProfessionalBenchTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type xiaozhiProfessionalBenchV21Client struct {
	delay    time.Duration
	fail     bool
	mu       sync.Mutex
	requests []v21adapter.QueryRequest
}

func (c *xiaozhiProfessionalBenchV21Client) lastUtterance() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		return ""
	}
	return c.requests[len(c.requests)-1].Utterance
}

func (c *xiaozhiProfessionalBenchV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.mu.Lock()
	c.requests = append(c.requests, request)
	c.mu.Unlock()
	if c.delay > 0 {
		timer := time.NewTimer(c.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return v21adapter.QueryResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
	if c.fail {
		return v21adapter.QueryResponse{}, errors.New("a21 host mock professional path failed")
	}
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "结论：需要按可引用证据复核。",
		Confidence: 0.77,
		Evidence: []v21adapter.Evidence{{
			Title:    "raw evidence title",
			Type:     "meeting",
			SourceID: "v21-doc-secret-raw",
			Summary:  "RAW_SECRET_EVIDENCE_BODY",
			Quote:    "RAW_SECRET_QUOTE",
		}},
		SpeechBlocks:      []string{"RAW_SECRET_SPEECH_BLOCK"},
		ScreenCards:       []v21adapter.ScreenCard{{Label: "结论", Text: "RAW_SECRET_CARD_TEXT"}},
		FollowUps:         []string{"RAW_SECRET_FOLLOW_UP"},
		SourceScopeCounts: map[string]int{"public": 1},
		WorkspaceStatus:   v21adapter.WorkspaceSearchable,
	}, nil
}
