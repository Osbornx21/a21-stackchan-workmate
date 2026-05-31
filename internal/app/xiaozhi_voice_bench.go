package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio/opuscodec"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type xiaozhiVoiceBenchOptions struct {
	GatewayURL      string
	Profile         string
	DeviceID        string
	ProtocolVersion int
	Repeat          int
	TimeoutMS       int
	OutputDir       string
}

type xiaozhiVoiceBenchReport struct {
	SchemaVersion   string                     `json:"schema_version"`
	GeneratedAtMS   int64                      `json:"generated_at_ms"`
	ExecutionMode   string                     `json:"execution_mode"`
	BaselineScope   string                     `json:"baseline_scope"`
	Gateway         string                     `json:"gateway"`
	Profile         string                     `json:"profile"`
	ProtocolVersion int                        `json:"protocol_version"`
	DeviceID        string                     `json:"device_id"`
	Repeat          int                        `json:"repeat"`
	AnswerTurns     []xiaozhiVoiceBenchTurn    `json:"answer_turns"`
	BargeInTurns    []xiaozhiVoiceBenchTurn    `json:"barge_in_turns"`
	Summary         xiaozhiVoiceBenchSummary   `json:"summary"`
	Counts          xiaozhiVoiceBenchCounts    `json:"counts"`
	Acceptance      string                     `json:"acceptance_status"`
	PRDAccepted     bool                       `json:"prd_accepted"`
	Execution       xiaozhiVoiceBenchExecution `json:"execution"`
	Redaction       xiaozhiVoiceBenchRedaction `json:"redaction"`
	Limitations     []string                   `json:"limitations,omitempty"`
	Findings        []xiaozhiVoiceBenchFinding `json:"findings,omitempty"`
	ReportPath      string                     `json:"report_path,omitempty"`
}

type xiaozhiVoiceBenchTurn struct {
	Turn                 int      `json:"turn"`
	Kind                 string   `json:"kind"`
	TraceID              string   `json:"trace_id"`
	SessionID            string   `json:"session_id"`
	HelloAccepted        bool     `json:"hello_accepted"`
	ListenAck            bool     `json:"listen_ack"`
	BinaryDownlinkFrames int      `json:"binary_downlink_frames"`
	FirstAudioMS         *int64   `json:"first_audio_ms,omitempty"`
	TTSStopReceived      bool     `json:"tts_stop_received"`
	AbortSent            bool     `json:"abort_sent"`
	AbortStopMS          *int64   `json:"abort_stop_ms,omitempty"`
	MetricsObserved      bool     `json:"metrics_observed"`
	Status               string   `json:"status"`
	Findings             []string `json:"findings,omitempty"`
}

type xiaozhiVoiceBenchSummary struct {
	AnswerFirstAudioP50MS float64 `json:"answer_first_audio_total_p50_ms"`
	AnswerFirstAudioP95MS float64 `json:"answer_first_audio_total_p95_ms"`
	BargeInStopP50MS      float64 `json:"barge_in_stop_p50_ms"`
	BargeInStopP95MS      float64 `json:"barge_in_stop_p95_ms"`
}

type xiaozhiVoiceBenchCounts struct {
	AnswerTurnCount  int `json:"answer_turn_count"`
	BargeInTurnCount int `json:"barge_in_turn_count"`
	FailureCount     int `json:"failure_count"`
}

type xiaozhiVoiceBenchExecution struct {
	ProviderExecuted bool `json:"provider_executed"`
	V21Executed      bool `json:"v21_executed"`
	HardwareExecuted bool `json:"hardware_executed"`
}

type xiaozhiVoiceBenchRedaction struct {
	PayloadsStored         bool `json:"payloads_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
}

type xiaozhiVoiceBenchFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func runXiaozhiVoiceBench(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiVoiceBenchOptions{
		GatewayURL:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		Profile:         "xiaozhi",
		DeviceID:        "stackchan-virtual-a21-bench-001",
		ProtocolVersion: 1,
		Repeat:          3,
		TimeoutMS:       5000,
		OutputDir:       "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-voice-bench [--gateway-url http://127.0.0.1:21080] [--profile xiaozhi|a21-debug] [--device-id stackchan-virtual-a21-bench-001] [--protocol-version 1|2|3] [--repeat 3] [--timeout-ms 5000] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--profile":
			if !readStringOption(args, &i, stderr, "--profile", &options.Profile) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--protocol-version":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--protocol-version")
			if !ok {
				return 2
			}
			options.ProtocolVersion = value
		case "--repeat":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--repeat")
			if !ok {
				return 2
			}
			options.Repeat = value
		case "--timeout-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--timeout-ms")
			if !ok {
				return 2
			}
			options.TimeoutMS = value
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-voice-bench option %q\n", args[i])
			return 2
		}
	}
	if err := validateXiaozhiVoiceBenchOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi voice bench option invalid: %v\n", err)
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "xiaozhi voice bench report dir invalid: %v\n", err)
		return 1
	}
	report := buildXiaozhiVoiceBenchReport(context.Background(), options)
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiVoiceBenchReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write xiaozhi voice bench report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiVoiceBench(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode xiaozhi voice bench report: %v\n", err)
		return 1
	}
	if report.Acceptance != "candidate_host_only" {
		return 1
	}
	return 0
}

func validateXiaozhiVoiceBenchOptions(options xiaozhiVoiceBenchOptions) error {
	switch strings.TrimSpace(options.Profile) {
	case "xiaozhi", "a21-debug":
	default:
		return fmt.Errorf("--profile must be xiaozhi or a21-debug")
	}
	switch options.ProtocolVersion {
	case 1, 2, 3:
	default:
		return fmt.Errorf("--protocol-version must be 1, 2, or 3")
	}
	if options.Repeat <= 0 || options.Repeat > 30 {
		return fmt.Errorf("--repeat must be between 1 and 30")
	}
	if options.TimeoutMS <= 0 || options.TimeoutMS > 30000 {
		return fmt.Errorf("--timeout-ms must be between 1 and 30000")
	}
	if xiaozhiVoiceBenchContainsLegacy(options.DeviceID) {
		return fmt.Errorf("device id contains legacy identity")
	}
	return nil
}

func buildXiaozhiVoiceBenchReport(ctx context.Context, options xiaozhiVoiceBenchOptions) xiaozhiVoiceBenchReport {
	generatedAtMS := time.Now().UnixMilli()
	report := xiaozhiVoiceBenchReport{
		SchemaVersion:   "a21.xiaozhi_voice_bench.v1",
		GeneratedAtMS:   generatedAtMS,
		ExecutionMode:   "host_loopback",
		BaselineScope:   "host_only",
		Gateway:         xiaozhiVoiceBenchGatewayLabel(options.GatewayURL),
		Profile:         strings.TrimSpace(options.Profile),
		ProtocolVersion: options.ProtocolVersion,
		DeviceID:        options.DeviceID,
		Repeat:          options.Repeat,
		Acceptance:      "blocked",
		PRDAccepted:     false,
		Execution: xiaozhiVoiceBenchExecution{
			ProviderExecuted: false,
			V21Executed:      false,
			HardwareExecuted: false,
		},
		Redaction: xiaozhiVoiceBenchRedaction{
			PayloadsStored:         false,
			CredentialValuesStored: false,
			FullURLsStored:         false,
			LocalPathsStored:       false,
		},
		Limitations: []string{
			"host loopback candidate only; physical StackChan microphone, playback start, and barge-in stop evidence are still required",
			"barge-in timing is measured from the host WebSocket stop lifecycle and must be rechecked with multi-frame TTS plus physical playback",
		},
	}
	wsURL, err := xiaozhiVoiceBenchWebSocketURL(options.GatewayURL)
	if err != nil {
		report.Findings = append(report.Findings, xiaozhiVoiceBenchFinding{Code: "gateway_url_invalid", Message: "gateway URL is invalid"})
		report.Counts.FailureCount = len(report.Findings)
		return report
	}
	packet, err := xiaozhiVoiceBenchOpusPacket()
	if err != nil {
		report.Findings = append(report.Findings, xiaozhiVoiceBenchFinding{Code: "opus_fixture_unavailable", Message: "synthetic Opus fixture could not be generated"})
		report.Counts.FailureCount = len(report.Findings)
		return report
	}
	for turn := 1; turn <= options.Repeat; turn++ {
		traceID := fmt.Sprintf("a21-trace-xiaozhi-bench-%d-answer-%02d", generatedAtMS, turn)
		sessionID := fmt.Sprintf("a21-session-xiaozhi-bench-%d-answer-%02d", generatedAtMS, turn)
		receipt := runXiaozhiVoiceBenchTurn(ctx, options, wsURL, packet, turn, "answer", traceID, sessionID, false)
		report.AnswerTurns = append(report.AnswerTurns, receipt)
	}
	for turn := 1; turn <= options.Repeat; turn++ {
		traceID := fmt.Sprintf("a21-trace-xiaozhi-bench-%d-barge-%02d", generatedAtMS, turn)
		sessionID := fmt.Sprintf("a21-session-xiaozhi-bench-%d-barge-%02d", generatedAtMS, turn)
		receipt := runXiaozhiVoiceBenchTurn(ctx, options, wsURL, packet, turn, "barge_in", traceID, sessionID, true)
		report.BargeInTurns = append(report.BargeInTurns, receipt)
	}
	report.Counts.AnswerTurnCount = len(report.AnswerTurns)
	report.Counts.BargeInTurnCount = len(report.BargeInTurns)
	report.Summary = summarizeXiaozhiVoiceBench(report.AnswerTurns, report.BargeInTurns)
	report.Counts.FailureCount = xiaozhiVoiceBenchFailureCount(report.AnswerTurns) + xiaozhiVoiceBenchFailureCount(report.BargeInTurns) + len(report.Findings)
	if report.Counts.FailureCount == 0 &&
		xiaozhiVoiceBenchHasAnswerSamples(report.AnswerTurns) &&
		xiaozhiVoiceBenchHasBargeInSamples(report.BargeInTurns) &&
		report.Summary.AnswerFirstAudioP95MS < 1500 &&
		report.Summary.BargeInStopP95MS < 300 {
		report.Acceptance = "candidate_host_only"
	}
	return report
}

func runXiaozhiVoiceBenchTurn(ctx context.Context, options xiaozhiVoiceBenchOptions, wsURL string, packet []byte, turn int, kind string, traceID string, sessionID string, abortAfterFirstAudio bool) xiaozhiVoiceBenchTurn {
	receipt := xiaozhiVoiceBenchTurn{
		Turn:      turn,
		Kind:      kind,
		TraceID:   traceID,
		SessionID: sessionID,
		Status:    "failed",
	}
	turnCtx, cancel := context.WithTimeout(ctx, time.Duration(options.TimeoutMS)*time.Millisecond)
	defer cancel()
	conn, _, err := websocket.Dial(turnCtx, wsURL, nil)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "gateway_websocket_unavailable")
		return receipt
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	startAt := time.Now()
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchHello(options, traceID, sessionID)); err != nil {
		receipt.Findings = append(receipt.Findings, "hello_send_failed")
		return receipt
	}
	if message, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); ok {
		markXiaozhiVoiceBenchJSON(&receipt, message)
	}
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchListen(options, traceID, sessionID, "start")); err != nil {
		receipt.Findings = append(receipt.Findings, "listen_start_send_failed")
		return receipt
	}
	if message, ok := readXiaozhiVoiceBenchJSON(turnCtx, conn); ok {
		markXiaozhiVoiceBenchJSON(&receipt, message)
	}
	if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
		receipt.Findings = append(receipt.Findings, "opus_uplink_send_failed")
		return receipt
	}
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchListen(options, traceID, sessionID, "stop")); err != nil {
		receipt.Findings = append(receipt.Findings, "listen_stop_send_failed")
		return receipt
	}
	var abortAt time.Time
	for {
		messageType, data, err := conn.Read(turnCtx)
		if err != nil {
			receipt.Findings = append(receipt.Findings, "turn_read_failed")
			return receipt
		}
		switch messageType {
		case websocket.MessageText:
			var message map[string]any
			if err := json.Unmarshal(data, &message); err == nil {
				markXiaozhiVoiceBenchJSON(&receipt, message)
				if receipt.AbortSent && receipt.TTSStopReceived && !abortAt.IsZero() {
					value := int64(time.Since(abortAt) / time.Millisecond)
					receipt.AbortStopMS = &value
					receipt.Status = "passed"
					return receipt
				}
				if !abortAfterFirstAudio && receipt.TTSStopReceived {
					receipt.Status = "passed"
					return receipt
				}
			}
		case websocket.MessageBinary:
			receipt.BinaryDownlinkFrames++
			if receipt.FirstAudioMS == nil {
				value := int64(time.Since(startAt) / time.Millisecond)
				receipt.FirstAudioMS = &value
			}
			if abortAfterFirstAudio && !receipt.AbortSent {
				if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchAbort(options, traceID, sessionID)); err != nil {
					receipt.Findings = append(receipt.Findings, "abort_send_failed")
					return receipt
				}
				receipt.AbortSent = true
				abortAt = time.Now()
			}
		}
	}
}

func readXiaozhiVoiceBenchJSON(ctx context.Context, conn *websocket.Conn) (map[string]any, bool) {
	var message map[string]any
	if err := wsjson.Read(ctx, conn, &message); err != nil {
		return nil, false
	}
	return message, true
}

func markXiaozhiVoiceBenchJSON(receipt *xiaozhiVoiceBenchTurn, message map[string]any) {
	if receipt == nil {
		return
	}
	if message["type"] == "hello" && message["transport"] == "websocket" {
		receipt.HelloAccepted = true
	}
	if message["type"] == "listen" && message["state"] == "start" && message["status"] == "accepted" {
		receipt.ListenAck = true
	}
	if message["type"] == "tts" && message["state"] == "stop" {
		receipt.TTSStopReceived = true
	}
	if message["type"] == "tts" {
		if _, ok := message["audio_ingress"].(map[string]any); ok {
			receipt.MetricsObserved = true
		}
		if _, ok := message["voice_pipeline"].(map[string]any); ok {
			receipt.MetricsObserved = true
		}
	}
}

func xiaozhiVoiceBenchHello(options xiaozhiVoiceBenchOptions, traceID string, sessionID string) map[string]any {
	features := map[string]bool{
		"mcp": true,
		"aec": true,
	}
	if options.Profile == "a21-debug" {
		features["device_events"] = true
		features["debug_metrics"] = true
	}
	audioParams := map[string]any{
		"format":                  "opus",
		"sample_rate":             16000,
		"channels":                1,
		"frame_duration":          60,
		"binary_protocol_version": options.ProtocolVersion,
	}
	return map[string]any{
		"type":         "hello",
		"version":      options.ProtocolVersion,
		"transport":    "websocket",
		"device_id":    options.DeviceID,
		"trace_id":     traceID,
		"session_id":   sessionID,
		"features":     features,
		"audio":        audioParams,
		"audio_params": audioParams,
	}
}

func xiaozhiVoiceBenchListen(options xiaozhiVoiceBenchOptions, traceID string, sessionID string, state string) map[string]any {
	return map[string]any{
		"type":       "listen",
		"state":      state,
		"device_id":  options.DeviceID,
		"trace_id":   traceID,
		"session_id": sessionID,
	}
}

func xiaozhiVoiceBenchAbort(options xiaozhiVoiceBenchOptions, traceID string, sessionID string) map[string]any {
	return map[string]any{
		"type":       "abort",
		"reason":     "barge_in",
		"device_id":  options.DeviceID,
		"trace_id":   traceID,
		"session_id": sessionID,
	}
}

func xiaozhiVoiceBenchOpusPacket() ([]byte, error) {
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		return nil, err
	}
	pcm := make([]int16, codec.FrameSamples())
	for i := range pcm {
		pcm[i] = int16(math.Sin(2*math.Pi*440*float64(i)/16000) * 12000)
	}
	return codec.EncodePCM16(pcm)
}

func wrapXiaozhiVoiceBenchOpus(packet []byte, protocolVersion int) []byte {
	switch protocolVersion {
	case 2:
		wrapped := make([]byte, 16+len(packet))
		binary.BigEndian.PutUint16(wrapped[0:2], 2)
		binary.BigEndian.PutUint32(wrapped[8:12], uint32(time.Now().UnixMilli()))
		binary.BigEndian.PutUint32(wrapped[12:16], uint32(len(packet)))
		copy(wrapped[16:], packet)
		return wrapped
	case 3:
		wrapped := make([]byte, 4+len(packet))
		binary.BigEndian.PutUint16(wrapped[2:4], uint16(len(packet)))
		copy(wrapped[4:], packet)
		return wrapped
	default:
		return append([]byte(nil), packet...)
	}
}

func summarizeXiaozhiVoiceBench(answerTurns []xiaozhiVoiceBenchTurn, bargeInTurns []xiaozhiVoiceBenchTurn) xiaozhiVoiceBenchSummary {
	answerDurations := make([]time.Duration, 0, len(answerTurns))
	for _, turn := range answerTurns {
		if turn.FirstAudioMS != nil {
			answerDurations = append(answerDurations, time.Duration(*turn.FirstAudioMS)*time.Millisecond)
		}
	}
	bargeDurations := make([]time.Duration, 0, len(bargeInTurns))
	for _, turn := range bargeInTurns {
		if turn.AbortStopMS != nil {
			bargeDurations = append(bargeDurations, time.Duration(*turn.AbortStopMS)*time.Millisecond)
		}
	}
	return xiaozhiVoiceBenchSummary{
		AnswerFirstAudioP50MS: percentileMS(answerDurations, 0.50),
		AnswerFirstAudioP95MS: percentileMS(answerDurations, 0.95),
		BargeInStopP50MS:      percentileMS(bargeDurations, 0.50),
		BargeInStopP95MS:      percentileMS(bargeDurations, 0.95),
	}
}

func xiaozhiVoiceBenchFailureCount(turns []xiaozhiVoiceBenchTurn) int {
	failures := 0
	for _, turn := range turns {
		if turn.Status != "passed" || !turn.HelloAccepted || !turn.ListenAck || turn.BinaryDownlinkFrames <= 0 || !turn.MetricsObserved || len(turn.Findings) > 0 {
			failures++
			continue
		}
		if turn.Kind == "answer" && turn.FirstAudioMS == nil {
			failures++
		}
		if turn.Kind == "barge_in" && turn.AbortStopMS == nil {
			failures++
		}
	}
	return failures
}

func xiaozhiVoiceBenchHasAnswerSamples(turns []xiaozhiVoiceBenchTurn) bool {
	for _, turn := range turns {
		if turn.Kind == "answer" && turn.FirstAudioMS != nil {
			return true
		}
	}
	return false
}

func xiaozhiVoiceBenchHasBargeInSamples(turns []xiaozhiVoiceBenchTurn) bool {
	for _, turn := range turns {
		if turn.Kind == "barge_in" && turn.AbortStopMS != nil {
			return true
		}
	}
	return false
}

func xiaozhiVoiceBenchWebSocketURL(gatewayURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid gateway URL")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported gateway URL scheme")
	}
	parsed.Path = "/v1/xiaozhi"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func xiaozhiVoiceBenchGatewayLabel(gatewayURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "invalid_gateway"
	}
	scope := "remote"
	switch parsed.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		scope = "loopback"
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scope + ":" + port + "/v1/xiaozhi"
}

func xiaozhiVoiceBenchContainsLegacy(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func writeXiaozhiVoiceBenchReport(outputDir string, report xiaozhiVoiceBenchReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-voice-bench-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiVoiceBench(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiVoiceBench(writer io.Writer, report xiaozhiVoiceBenchReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
