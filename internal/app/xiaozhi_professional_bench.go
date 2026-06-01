package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const xiaozhiProfessionalBenchASRSentinel = "a21 host mock professional ASR sentinel"
const xiaozhiProfessionalBenchPlaceholderUtterance = "xiaozhi professional voice turn"

type xiaozhiProfessionalBenchOptions struct {
	Scenario        string
	FakeV21Delay    time.Duration
	Timeout         time.Duration
	OutputDir       string
	DeviceID        string
	ProtocolVersion int
}

func parseXiaozhiProfessionalBenchPositiveInt(raw string, option string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", option)
	}
	return value, nil
}

type xiaozhiProfessionalBenchReport struct {
	SchemaVersion                   string                            `json:"schema_version"`
	GeneratedAtUnixMS               int64                             `json:"generated_at_unix_ms"`
	SourceProfile                   string                            `json:"source_profile"`
	Scenario                        string                            `json:"scenario"`
	AcceptanceStatus                string                            `json:"acceptance_status"`
	PRDAccepted                     bool                              `json:"prd_accepted"`
	TraceID                         string                            `json:"trace_id"`
	SessionID                       string                            `json:"session_id"`
	DeviceID                        string                            `json:"device_id"`
	CheckingFeedbackObserved        bool                              `json:"checking_feedback_observed"`
	CheckingFeedbackMS              int64                             `json:"checking_feedback_ms"`
	CheckingFeedbackWithin1200      bool                              `json:"checking_feedback_within_1200"`
	ProfessionalResultObserved      bool                              `json:"professional_result_observed"`
	ProfessionalResultAfterChecking bool                              `json:"professional_result_after_checking"`
	AbortStopObserved               bool                              `json:"abort_stop_observed"`
	StaleResultSuppressed           bool                              `json:"stale_result_suppressed"`
	EvidenceCount                   int                               `json:"evidence_count"`
	ScreenCardCount                 int                               `json:"screen_card_count"`
	FollowUpCount                   int                               `json:"follow_up_count"`
	ConfidencePresent               bool                              `json:"confidence_present"`
	NoPlaceholderUtterance          bool                              `json:"no_placeholder_utterance"`
	NoASRTextLeak                   bool                              `json:"no_asr_text_leak"`
	TTSStopObserved                 bool                              `json:"tts_stop_observed"`
	FailureCount                    int                               `json:"failure_count"`
	Execution                       xiaozhiProfessionalBenchExecution `json:"execution"`
	Redaction                       xiaozhiProfessionalBenchRedaction `json:"redaction"`
	Findings                        []xiaozhiProfessionalBenchFinding `json:"findings,omitempty"`
	ReportPath                      string                            `json:"report_path,omitempty"`
}

type xiaozhiProfessionalBenchExecution struct {
	ProviderExecuted bool   `json:"provider_executed"`
	V21Executed      bool   `json:"v21_executed"`
	HardwareExecuted bool   `json:"hardware_executed"`
	GatewayRuntime   string `json:"gateway_runtime"`
}

type xiaozhiProfessionalBenchRedaction struct {
	PayloadsStored       bool `json:"payloads_stored"`
	ASRTextStored        bool `json:"asr_text_stored"`
	EvidenceBodyStored   bool `json:"evidence_body_stored"`
	FullURLStored        bool `json:"full_url_stored"`
	LocalPathStored      bool `json:"local_path_stored"`
	PromptStored         bool `json:"prompt_stored"`
	ProviderOutputStored bool `json:"provider_output_stored"`
}

type xiaozhiProfessionalBenchFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func runXiaozhiProfessionalBench(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiProfessionalBenchOptions{
		Scenario:        "success",
		FakeV21Delay:    100 * time.Millisecond,
		Timeout:         3 * time.Second,
		DeviceID:        "stackchan-virtual-a21-professional-bench-001",
		ProtocolVersion: 1,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-professional-bench [--scenario success|v21_failure] [--fake-v21-delay-ms 100] [--timeout-ms 3000] [--output-dir reports]")
			return 0
		case "--scenario":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--scenario requires a value")
				return 2
			}
			i++
			options.Scenario = strings.TrimSpace(args[i])
		case "--fake-v21-delay-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--fake-v21-delay-ms requires a value")
				return 2
			}
			i++
			delayMS, err := parseXiaozhiProfessionalBenchPositiveInt(args[i], "--fake-v21-delay-ms")
			if err != nil {
				fmt.Fprintf(stderr, "xiaozhi professional bench option invalid: %v\n", err)
				return 2
			}
			options.FakeV21Delay = time.Duration(delayMS) * time.Millisecond
		case "--timeout-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--timeout-ms requires a value")
				return 2
			}
			i++
			timeoutMS, err := parseXiaozhiProfessionalBenchPositiveInt(args[i], "--timeout-ms")
			if err != nil {
				fmt.Fprintf(stderr, "xiaozhi professional bench option invalid: %v\n", err)
				return 2
			}
			options.Timeout = time.Duration(timeoutMS) * time.Millisecond
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-professional-bench option %q\n", args[i])
			return 2
		}
	}
	if err := validateXiaozhiProfessionalBenchOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi professional bench option invalid: %v\n", err)
		return 2
	}
	report := buildXiaozhiProfessionalBenchReport(context.Background(), options)
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "xiaozhi professional bench report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeXiaozhiProfessionalBenchReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write xiaozhi professional bench report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiProfessionalBench(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode xiaozhi professional bench report: %v\n", err)
		return 1
	}
	if report.FailureCount > 0 || report.AcceptanceStatus != "host_mock_ready" {
		return 1
	}
	return 0
}

func validateXiaozhiProfessionalBenchOptions(options xiaozhiProfessionalBenchOptions) error {
	switch options.Scenario {
	case "success", "v21_failure":
	default:
		return fmt.Errorf("--scenario must be success or v21_failure")
	}
	if options.FakeV21Delay < 0 || options.FakeV21Delay > 5*time.Second {
		return fmt.Errorf("--fake-v21-delay-ms must be between 0 and 5000")
	}
	if options.Timeout <= 0 {
		return fmt.Errorf("--timeout-ms must be positive")
	}
	if xiaozhiVoiceBenchContainsLegacy(options.DeviceID) {
		return fmt.Errorf("--device-id must use A21/StackChan identity")
	}
	return nil
}

func buildXiaozhiProfessionalBenchReport(ctx context.Context, options xiaozhiProfessionalBenchOptions) xiaozhiProfessionalBenchReport {
	generatedAtMS := time.Now().UnixMilli()
	traceID := fmt.Sprintf("a21-trace-xiaozhi-professional-bench-%d", generatedAtMS)
	sessionID := fmt.Sprintf("a21-session-xiaozhi-professional-bench-%d", generatedAtMS)
	report := xiaozhiProfessionalBenchReport{
		SchemaVersion:     "a21.xiaozhi_professional_bench.v1",
		GeneratedAtUnixMS: generatedAtMS,
		SourceProfile:     "host_mock",
		Scenario:          options.Scenario,
		AcceptanceStatus:  "host_mock_blocked",
		PRDAccepted:       false,
		TraceID:           traceID,
		SessionID:         sessionID,
		DeviceID:          options.DeviceID,
		Execution: xiaozhiProfessionalBenchExecution{
			ProviderExecuted: false,
			V21Executed:      false,
			HardwareExecuted: false,
			GatewayRuntime:   "httptest_in_process",
		},
		Redaction: xiaozhiProfessionalBenchRedaction{
			PayloadsStored:       false,
			ASRTextStored:        false,
			EvidenceBodyStored:   false,
			FullURLStored:        false,
			LocalPathStored:      false,
			PromptStored:         false,
			ProviderOutputStored: false,
		},
	}
	v21 := &xiaozhiProfessionalBenchV21Client{delay: options.FakeV21Delay, fail: options.Scenario == "v21_failure"}
	adapters := providers.VoicePipelineAdapters{
		ASR:        xiaozhiProfessionalBenchASRAdapter{text: xiaozhiProfessionalBenchASRSentinel},
		TextStream: xiaozhiProfessionalBenchTextStreamAdapter{},
		TTS:        providers.NewMockTTSAdapter("a21-host-mock-tts"),
	}
	server := gateway.NewServerWithOptions(gateway.ServerOptions{
		V21Client:                    v21,
		XiaozhiVoicePipelineAdapters: &adapters,
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	turn := runXiaozhiProfessionalBenchTurn(ctx, options, httpServer.URL, traceID, sessionID)
	report.CheckingFeedbackObserved = turn.checkingObserved
	report.CheckingFeedbackMS = turn.checkingMS
	report.CheckingFeedbackWithin1200 = turn.checkingObserved && turn.checkingMS <= int64(v21adapter.ProfessionalMaxFirstResponseMS)
	report.ProfessionalResultObserved = turn.resultObserved
	report.ProfessionalResultAfterChecking = turn.resultObserved && turn.resultIndex > turn.checkingIndex && turn.checkingIndex >= 0
	abortTurn := runXiaozhiProfessionalBenchAbortTurn(ctx, options)
	report.AbortStopObserved = abortTurn.abortStopObserved
	report.StaleResultSuppressed = abortTurn.staleResultSuppressed
	report.EvidenceCount = turn.evidenceCount
	report.ScreenCardCount = turn.screenCardCount
	report.FollowUpCount = turn.followUpCount
	report.ConfidencePresent = turn.confidencePresent
	report.TTSStopObserved = turn.ttsStopObserved
	report.NoPlaceholderUtterance = v21.lastUtterance() != "" && v21.lastUtterance() != xiaozhiProfessionalBenchPlaceholderUtterance
	if !report.CheckingFeedbackObserved {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "checking_feedback_missing", Message: "professional checking feedback was not observed"})
	} else if !report.CheckingFeedbackWithin1200 {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "checking_feedback_slow", Message: "professional checking feedback exceeded 1200 ms"})
	}
	if !report.ProfessionalResultObserved {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "professional_result_missing", Message: "professional result was not observed"})
	}
	if !report.ProfessionalResultAfterChecking && report.ProfessionalResultObserved {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "professional_result_order_invalid", Message: "professional result was not observed after checking feedback"})
	}
	if !report.TTSStopObserved {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "tts_stop_missing", Message: "professional TTS stop was not observed"})
	}
	if !report.AbortStopObserved {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "abort_stop_missing", Message: "professional abort stop was not observed"})
	}
	if !report.StaleResultSuppressed {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "stale_result_not_suppressed", Message: "professional result was observed after abort"})
	}
	if !report.NoPlaceholderUtterance {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "placeholder_utterance_used", Message: "professional query did not use ASR-derived utterance"})
	}
	report.FailureCount = len(report.Findings)
	if report.FailureCount == 0 {
		report.AcceptanceStatus = "host_mock_ready"
	}
	report.NoASRTextLeak = xiaozhiProfessionalBenchReportOmitsSensitiveText(report)
	if !report.NoASRTextLeak {
		report.FailureCount++
		report.AcceptanceStatus = "host_mock_blocked"
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "redaction_failed", Message: "professional bench report contained unsafe content"})
	}
	return report
}

type xiaozhiProfessionalBenchTurn struct {
	checkingObserved  bool
	checkingMS        int64
	checkingIndex     int
	resultObserved    bool
	resultIndex       int
	evidenceCount     int
	screenCardCount   int
	followUpCount     int
	confidencePresent bool
	ttsStopObserved   bool
}

func runXiaozhiProfessionalBenchTurn(ctx context.Context, options xiaozhiProfessionalBenchOptions, gatewayURL string, traceID string, sessionID string) xiaozhiProfessionalBenchTurn {
	turn := xiaozhiProfessionalBenchTurn{checkingIndex: -1, resultIndex: -1}
	turnCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	wsURL, err := xiaozhiVoiceBenchWebSocketURL(gatewayURL)
	if err != nil {
		return turn
	}
	conn, _, err := websocket.Dial(turnCtx, wsURL, nil)
	if err != nil {
		return turn
	}
	defer conn.Close(websocket.StatusNormalClosure, "xiaozhi professional bench complete")
	voiceOptions := xiaozhiVoiceBenchOptions{
		Profile:         "xiaozhi",
		DeviceID:        options.DeviceID,
		ProtocolVersion: options.ProtocolVersion,
		TimeoutMS:       int(options.Timeout / time.Millisecond),
	}
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchHello(voiceOptions, traceID, sessionID)); err != nil {
		return turn
	}
	if _, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); !ok {
		return turn
	}
	startListen := xiaozhiVoiceBenchListen(voiceOptions, traceID, sessionID, "start")
	startListen["mode"] = "professional"
	if err := wsjson.Write(turnCtx, conn, startListen); err != nil {
		return turn
	}
	if _, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); !ok {
		return turn
	}
	packet, err := xiaozhiVoiceBenchSyntheticOpusPacket()
	if err != nil {
		return turn
	}
	if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
		return turn
	}
	stopListen := xiaozhiVoiceBenchListen(voiceOptions, traceID, sessionID, "stop")
	stopListen["mode"] = "professional"
	stopSent := time.Now()
	if err := wsjson.Write(turnCtx, conn, stopListen); err != nil {
		return turn
	}
	for index := 0; index < 8; index++ {
		message, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn)
		if !ok {
			return turn
		}
		if message["type"] != "tts" {
			continue
		}
		phase := xiaozhiVoiceBenchStringField(message, "phase")
		state := xiaozhiVoiceBenchStringField(message, "state")
		switch {
		case phase == "professional_checking":
			turn.checkingObserved = true
			turn.checkingIndex = index
			turn.checkingMS = time.Since(stopSent).Milliseconds()
		case phase == "professional_result":
			turn.resultObserved = true
			turn.resultIndex = index
			professional, _ := message["professional"].(map[string]any)
			turn.evidenceCount = int(xiaozhiProfessionalBenchFloatField(professional, "evidence_count"))
			turn.screenCardCount = int(xiaozhiProfessionalBenchFloatField(professional, "screen_card_count"))
			turn.followUpCount = int(xiaozhiProfessionalBenchFloatField(professional, "follow_up_count"))
			turn.confidencePresent = xiaozhiProfessionalBenchBoolField(professional, "confidence_present")
		case state == "stop":
			turn.ttsStopObserved = true
			return turn
		}
	}
	return turn
}

type xiaozhiProfessionalBenchAbortTurn struct {
	abortStopObserved     bool
	staleResultSuppressed bool
}

func runXiaozhiProfessionalBenchAbortTurn(ctx context.Context, options xiaozhiProfessionalBenchOptions) xiaozhiProfessionalBenchAbortTurn {
	result := xiaozhiProfessionalBenchAbortTurn{staleResultSuppressed: true}
	traceID := fmt.Sprintf("a21-trace-xiaozhi-professional-bench-abort-%d", time.Now().UnixMilli())
	sessionID := fmt.Sprintf("a21-session-xiaozhi-professional-bench-abort-%d", time.Now().UnixMilli())
	v21 := &xiaozhiProfessionalBenchV21Client{delay: 500 * time.Millisecond}
	adapters := providers.VoicePipelineAdapters{
		ASR:        xiaozhiProfessionalBenchASRAdapter{text: xiaozhiProfessionalBenchASRSentinel},
		TextStream: xiaozhiProfessionalBenchTextStreamAdapter{},
		TTS:        providers.NewMockTTSAdapter("a21-host-mock-tts"),
	}
	server := gateway.NewServerWithOptions(gateway.ServerOptions{
		V21Client:                    v21,
		XiaozhiVoicePipelineAdapters: &adapters,
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	turnCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	wsURL, err := xiaozhiVoiceBenchWebSocketURL(httpServer.URL)
	if err != nil {
		return result
	}
	conn, _, err := websocket.Dial(turnCtx, wsURL, nil)
	if err != nil {
		return result
	}
	defer conn.Close(websocket.StatusNormalClosure, "xiaozhi professional abort bench complete")
	voiceOptions := xiaozhiVoiceBenchOptions{
		Profile:         "xiaozhi",
		DeviceID:        options.DeviceID,
		ProtocolVersion: options.ProtocolVersion,
		TimeoutMS:       int(options.Timeout / time.Millisecond),
	}
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchHello(voiceOptions, traceID, sessionID)); err != nil {
		return result
	}
	if _, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); !ok {
		return result
	}
	startListen := xiaozhiVoiceBenchListen(voiceOptions, traceID, sessionID, "start")
	startListen["mode"] = "professional"
	if err := wsjson.Write(turnCtx, conn, startListen); err != nil {
		return result
	}
	if _, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); !ok {
		return result
	}
	packet, err := xiaozhiVoiceBenchSyntheticOpusPacket()
	if err != nil {
		return result
	}
	if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
		return result
	}
	stopListen := xiaozhiVoiceBenchListen(voiceOptions, traceID, sessionID, "stop")
	stopListen["mode"] = "professional"
	if err := wsjson.Write(turnCtx, conn, stopListen); err != nil {
		return result
	}
	for index := 0; index < 4; index++ {
		message, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn)
		if !ok {
			return result
		}
		if xiaozhiVoiceBenchStringField(message, "phase") == "professional_checking" {
			if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchAbort(voiceOptions, traceID, sessionID)); err != nil {
				return result
			}
			break
		}
	}
	for index := 0; index < 4; index++ {
		message, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn)
		if !ok {
			return result
		}
		if xiaozhiVoiceBenchStringField(message, "phase") == "professional_result" {
			result.staleResultSuppressed = false
		}
		if message["type"] == "tts" && xiaozhiVoiceBenchStringField(message, "state") == "stop" {
			result.abortStopObserved = true
			break
		}
	}
	staleCtx, staleCancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer staleCancel()
	if message, ok := readXiaozhiVoiceBenchJSON(staleCtx, conn); ok && xiaozhiVoiceBenchStringField(message, "phase") == "professional_result" {
		result.staleResultSuppressed = false
	}
	return result
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
		SpeechBlocks: []string{"RAW_SECRET_SPEECH_BLOCK"},
		ScreenCards:  []v21adapter.ScreenCard{{Label: "结论", Text: "RAW_SECRET_CARD_TEXT"}},
		FollowUps:    []string{"RAW_SECRET_FOLLOW_UP"},
	}, nil
}
