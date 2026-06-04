package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	GatewayURL      string
	InputWAV        string
}

func parseXiaozhiProfessionalBenchPositiveInt(raw string, option string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", option)
	}
	return value, nil
}

type xiaozhiProfessionalBenchReport struct {
	SchemaVersion                   string                             `json:"schema_version"`
	GeneratedAtUnixMS               int64                              `json:"generated_at_unix_ms"`
	SourceProfile                   string                             `json:"source_profile"`
	Scenario                        string                             `json:"scenario"`
	AcceptanceStatus                string                             `json:"acceptance_status"`
	PRDAccepted                     bool                               `json:"prd_accepted"`
	Gateway                         string                             `json:"gateway,omitempty"`
	Input                           *xiaozhiVoiceBenchInput            `json:"input_audio,omitempty"`
	TraceID                         string                             `json:"trace_id"`
	SessionID                       string                             `json:"session_id"`
	DeviceID                        string                             `json:"device_id"`
	CheckingFeedbackObserved        bool                               `json:"checking_feedback_observed"`
	CheckingFeedbackMS              int64                              `json:"checking_feedback_ms"`
	CheckingFeedbackWithin1200      bool                               `json:"checking_feedback_within_1200"`
	ProfessionalResultObserved      bool                               `json:"professional_result_observed"`
	ProfessionalResultAfterChecking bool                               `json:"professional_result_after_checking"`
	AbortStopObserved               bool                               `json:"abort_stop_observed"`
	StaleResultSuppressed           bool                               `json:"stale_result_suppressed"`
	EvidenceCount                   int                                `json:"evidence_count"`
	ScreenCardCount                 int                                `json:"screen_card_count"`
	FollowUpCount                   int                                `json:"follow_up_count"`
	ConfidencePresent               bool                               `json:"confidence_present"`
	NoPlaceholderUtterance          bool                               `json:"no_placeholder_utterance"`
	NoASRTextLeak                   bool                               `json:"no_asr_text_leak"`
	V21QueryFirstResultMS           *int64                             `json:"v21_query_first_result_ms,omitempty"`
	ReadRecord                      xiaozhiProfessionalBenchReadRecord `json:"read_record"`
	TTSStopObserved                 bool                               `json:"tts_stop_observed"`
	FailureCount                    int                                `json:"failure_count"`
	Execution                       xiaozhiProfessionalBenchExecution  `json:"execution"`
	Redaction                       xiaozhiProfessionalBenchRedaction  `json:"redaction"`
	Findings                        []xiaozhiProfessionalBenchFinding  `json:"findings,omitempty"`
	ReportPath                      string                             `json:"report_path,omitempty"`
}

type xiaozhiProfessionalBenchExecution struct {
	ProviderExecuted bool   `json:"provider_executed"`
	V21Executed      bool   `json:"v21_executed"`
	HardwareExecuted bool   `json:"hardware_executed"`
	GatewayRuntime   string `json:"gateway_runtime"`
}

type xiaozhiProfessionalBenchReadRecord struct {
	Observed          bool                                     `json:"observed"`
	Completed         bool                                     `json:"completed"`
	RecordCount       int                                      `json:"record_count"`
	Status            string                                   `json:"status,omitempty"`
	RecordID          string                                   `json:"record_id,omitempty"`
	TraceIDMatched    bool                                     `json:"trace_id_matched"`
	SessionIDMatched  bool                                     `json:"session_id_matched"`
	DeviceIDMatched   bool                                     `json:"device_id_matched"`
	QueryScope        string                                   `json:"query_scope,omitempty"`
	PrivacyScope      string                                   `json:"privacy_scope,omitempty"`
	LatencyProfile    string                                   `json:"latency_profile,omitempty"`
	AnswerStyle       string                                   `json:"answer_style,omitempty"`
	UtteranceBucket   string                                   `json:"utterance_bucket,omitempty"`
	SourceScopeCounts map[string]int                           `json:"source_scope_counts,omitempty"`
	WorkspaceStatus   string                                   `json:"workspace_status,omitempty"`
	Redaction         xiaozhiProfessionalBenchReadRecordRedact `json:"redaction"`
}

type xiaozhiProfessionalBenchReadRecordRedact struct {
	DocumentTextStored    bool `json:"document_text_stored"`
	QueryTextStored       bool `json:"query_text_stored"`
	RetrievedTextStored   bool `json:"retrieved_text_stored"`
	FullURLStored         bool `json:"full_url_stored"`
	LocalPathStored       bool `json:"local_path_stored"`
	CredentialValueStored bool `json:"credential_value_stored"`
	ProviderOutputStored  bool `json:"provider_output_stored"`
	VoiceTextStored       bool `json:"voice_text_stored"`
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
			fmt.Fprintln(stdout, "a21 xiaozhi-professional-bench [--gateway-url http://127.0.0.1:21080] [--scenario success|v21_failure] [--fake-v21-delay-ms 100] [--device-id stackchan-virtual-a21-professional-bench-001] [--protocol-version 1|2|3] [--input-wav fixture.wav] [--timeout-ms 3000] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--input-wav":
			if !readStringOption(args, &i, stderr, "--input-wav", &options.InputWAV) {
				return 2
			}
		case "--protocol-version":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--protocol-version")
			if !ok {
				return 2
			}
			options.ProtocolVersion = value
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
	if report.FailureCount > 0 || (report.AcceptanceStatus != "host_mock_ready" && report.AcceptanceStatus != "external_gateway_ready") {
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
	switch options.ProtocolVersion {
	case 1, 2, 3:
	default:
		return fmt.Errorf("--protocol-version must be 1, 2, or 3")
	}
	if xiaozhiVoiceBenchContainsLegacy(options.DeviceID) {
		return fmt.Errorf("--device-id must use A21/StackChan identity")
	}
	if strings.TrimSpace(options.GatewayURL) != "" {
		if xiaozhiVoiceBenchContainsLegacy(options.GatewayURL) {
			return fmt.Errorf("--gateway-url contains legacy identity")
		}
		if _, err := xiaozhiVoiceBenchWebSocketURL(options.GatewayURL); err != nil {
			return fmt.Errorf("--gateway-url is invalid")
		}
	}
	if strings.TrimSpace(options.InputWAV) != "" {
		if xiaozhiVoiceBenchContainsLegacy(options.InputWAV) {
			return fmt.Errorf("--input-wav contains legacy identity")
		}
		if strings.ToLower(filepath.Ext(options.InputWAV)) != ".wav" {
			return fmt.Errorf("--input-wav must point to a wav file")
		}
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
	packets, input, err := xiaozhiProfessionalBenchOpusPackets(options)
	if err != nil {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "opus_fixture_unavailable", Message: "professional Opus uplink fixture could not be generated"})
		return finalizeXiaozhiProfessionalBenchReport(report)
	}
	report.Input = &input
	if strings.TrimSpace(options.GatewayURL) != "" {
		report.SourceProfile = "external_gateway"
		report.AcceptanceStatus = "external_gateway_blocked"
		report.Gateway = xiaozhiVoiceBenchGatewayLabel(options.GatewayURL)
		report.Execution.GatewayRuntime = "external_gateway"
		turn := runXiaozhiProfessionalBenchTurn(ctx, options, options.GatewayURL, traceID, sessionID, packets)
		report.CheckingFeedbackObserved = turn.checkingObserved
		report.CheckingFeedbackMS = turn.checkingMS
		report.CheckingFeedbackWithin1200 = turn.checkingObserved && turn.checkingMS <= int64(v21adapter.ProfessionalMaxFirstResponseMS)
		report.ProfessionalResultObserved = turn.resultObserved
		report.ProfessionalResultAfterChecking = turn.resultObserved && turn.resultIndex > turn.checkingIndex && turn.checkingIndex >= 0
		report.EvidenceCount = turn.evidenceCount
		report.ScreenCardCount = turn.screenCardCount
		report.FollowUpCount = turn.followUpCount
		report.ConfidencePresent = turn.confidencePresent
		report.TTSStopObserved = turn.ttsStopObserved
		trace, traceErr := fetchXiaozhiProfessionalBenchTraceEvidence(ctx, options.GatewayURL, traceID)
		if traceErr != nil {
			report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "external_gateway_trace_unavailable", Message: "external Gateway professional trace was unavailable"})
		}
		report.Execution.V21Executed = trace.V21QueryStarted && trace.V21QueryFirstResult
		report.NoPlaceholderUtterance = trace.UtteranceLengthObserved
		report.V21QueryFirstResultMS = trace.V21QueryFirstResultMS
		report.ReadRecord = fetchXiaozhiProfessionalBenchReadRecord(ctx, options.GatewayURL, traceID, sessionID, options.DeviceID)
		abortTurn := runXiaozhiProfessionalBenchAbortTurn(ctx, options, options.GatewayURL, packets)
		report.AbortStopObserved = abortTurn.abortStopObserved
		report.StaleResultSuppressed = abortTurn.staleResultSuppressed
		return finalizeXiaozhiProfessionalBenchReport(report)
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

	turn := runXiaozhiProfessionalBenchTurn(ctx, options, httpServer.URL, traceID, sessionID, packets)
	report.CheckingFeedbackObserved = turn.checkingObserved
	report.CheckingFeedbackMS = turn.checkingMS
	report.CheckingFeedbackWithin1200 = turn.checkingObserved && turn.checkingMS <= int64(v21adapter.ProfessionalMaxFirstResponseMS)
	report.ProfessionalResultObserved = turn.resultObserved
	report.ProfessionalResultAfterChecking = turn.resultObserved && turn.resultIndex > turn.checkingIndex && turn.checkingIndex >= 0
	abortTurn := runXiaozhiProfessionalBenchAbortTurn(ctx, options, "", packets)
	report.AbortStopObserved = abortTurn.abortStopObserved
	report.StaleResultSuppressed = abortTurn.staleResultSuppressed
	report.EvidenceCount = turn.evidenceCount
	report.ScreenCardCount = turn.screenCardCount
	report.FollowUpCount = turn.followUpCount
	report.ConfidencePresent = turn.confidencePresent
	report.TTSStopObserved = turn.ttsStopObserved
	report.NoPlaceholderUtterance = v21.lastUtterance() != "" && v21.lastUtterance() != xiaozhiProfessionalBenchPlaceholderUtterance
	report.ReadRecord = fetchXiaozhiProfessionalBenchReadRecord(ctx, httpServer.URL, traceID, sessionID, options.DeviceID)
	return finalizeXiaozhiProfessionalBenchReport(report)
}

func finalizeXiaozhiProfessionalBenchReport(report xiaozhiProfessionalBenchReport) xiaozhiProfessionalBenchReport {
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
	if report.SourceProfile == "external_gateway" && !xiaozhiProfessionalBenchReadRecordReady(report.ReadRecord) {
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "professional_read_record_missing", Message: "professional read-record ledger did not complete for the bench trace"})
	}
	report.FailureCount = len(report.Findings)
	if report.FailureCount == 0 {
		if report.SourceProfile == "external_gateway" {
			report.AcceptanceStatus = "external_gateway_ready"
		} else {
			report.AcceptanceStatus = "host_mock_ready"
		}
	}
	report.NoASRTextLeak = xiaozhiProfessionalBenchReportOmitsSensitiveText(report)
	if !report.NoASRTextLeak {
		report.FailureCount++
		report.AcceptanceStatus = "host_mock_blocked"
		report.Findings = append(report.Findings, xiaozhiProfessionalBenchFinding{Code: "redaction_failed", Message: "professional bench report contained unsafe content"})
	}
	return report
}

func xiaozhiProfessionalBenchReadRecordReady(record xiaozhiProfessionalBenchReadRecord) bool {
	return record.Observed &&
		record.Completed &&
		record.RecordCount == 1 &&
		record.Status == "completed" &&
		record.RecordID != "" &&
		record.TraceIDMatched &&
		record.SessionIDMatched &&
		record.DeviceIDMatched &&
		v21adapter.ValidQueryScope(record.QueryScope) &&
		record.PrivacyScope == "professional_only" &&
		record.LatencyProfile == "fast_first" &&
		record.AnswerStyle == "voice_first_with_citations" &&
		validProductV21WorkspaceStatus(record.WorkspaceStatus) &&
		validProductV21SourceScopeCounts(record.SourceScopeCounts) &&
		xiaozhiProfessionalBenchReadRecordRedactionOK(record.Redaction)
}

func xiaozhiProfessionalBenchReadRecordRedactionOK(redaction xiaozhiProfessionalBenchReadRecordRedact) bool {
	return !redaction.DocumentTextStored &&
		!redaction.QueryTextStored &&
		!redaction.RetrievedTextStored &&
		!redaction.FullURLStored &&
		!redaction.LocalPathStored &&
		!redaction.CredentialValueStored &&
		!redaction.ProviderOutputStored &&
		!redaction.VoiceTextStored
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

func runXiaozhiProfessionalBenchTurn(ctx context.Context, options xiaozhiProfessionalBenchOptions, gatewayURL string, traceID string, sessionID string, packets [][]byte) xiaozhiProfessionalBenchTurn {
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
	for _, packet := range packets {
		if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
			return turn
		}
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

func runXiaozhiProfessionalBenchAbortTurn(ctx context.Context, options xiaozhiProfessionalBenchOptions, gatewayURL string, packets [][]byte) xiaozhiProfessionalBenchAbortTurn {
	result := xiaozhiProfessionalBenchAbortTurn{staleResultSuppressed: true}
	traceID := fmt.Sprintf("a21-trace-xiaozhi-professional-bench-abort-%d", time.Now().UnixMilli())
	sessionID := fmt.Sprintf("a21-session-xiaozhi-professional-bench-abort-%d", time.Now().UnixMilli())
	if strings.TrimSpace(gatewayURL) == "" {
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
		gatewayURL = httpServer.URL
	}
	turnCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	wsURL, err := xiaozhiVoiceBenchWebSocketURL(gatewayURL)
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
	for _, packet := range packets {
		if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
			return result
		}
	}
	stopListen := xiaozhiVoiceBenchListen(voiceOptions, traceID, sessionID, "stop")
	stopListen["mode"] = "professional"
	if err := wsjson.Write(turnCtx, conn, stopListen); err != nil {
		return result
	}
	abortDeadline := time.Now().Add(options.Timeout)
	for time.Now().Before(abortDeadline) {
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
	stopDeadline := time.Now().Add(options.Timeout)
	for time.Now().Before(stopDeadline) {
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

type xiaozhiProfessionalBenchTraceEvidence struct {
	V21QueryStarted         bool
	V21QueryFirstResult     bool
	UtteranceLengthObserved bool
	V21QueryFirstResultMS   *int64
}

func fetchXiaozhiProfessionalBenchTraceEvidence(ctx context.Context, gatewayURL string, traceID string) (xiaozhiProfessionalBenchTraceEvidence, error) {
	endpoint, err := xiaozhiVoiceBenchTraceURL(gatewayURL, traceID)
	if err != nil {
		return xiaozhiProfessionalBenchTraceEvidence{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xiaozhiProfessionalBenchTraceEvidence{}, err
	}
	client := *a21DirectHTTPClient(2 * time.Second)
	response, err := client.Do(request)
	if err != nil {
		return xiaozhiProfessionalBenchTraceEvidence{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return xiaozhiProfessionalBenchTraceEvidence{}, fmt.Errorf("trace unavailable")
	}
	var decoded struct {
		Events  []xiaozhiVoiceBenchTraceEvent `json:"events"`
		Summary struct {
			V21QueryFirstResultMS *int64 `json:"v21_query_first_result_ms,omitempty"`
		} `json:"summary"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return xiaozhiProfessionalBenchTraceEvidence{}, err
	}
	evidence := xiaozhiProfessionalBenchTraceEvidence{V21QueryFirstResultMS: decoded.Summary.V21QueryFirstResultMS}
	for _, event := range decoded.Events {
		switch {
		case event.Name == "v21.query.start":
			evidence.V21QueryStarted = true
		case event.Name == "v21.query.first_result":
			evidence.V21QueryFirstResult = true
		case strings.HasPrefix(event.Name, "v21.query.utterance.length_"):
			evidence.UtteranceLengthObserved = true
		}
	}
	return evidence, nil
}

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
