package app

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type xiaozhiVoiceBenchOptions struct {
	GatewayURL          string
	Profile             string
	DeviceID            string
	ProtocolVersion     int
	InputWAV            string
	Repeat              int
	TimeoutMS           int
	OutputDir           string
	RequireProductChain bool
}

type xiaozhiVoiceBenchInput struct {
	Source         string `json:"source"`
	WAVName        string `json:"wav_name,omitempty"`
	OpusFrameCount int    `json:"opus_frame_count"`
}

type xiaozhiVoiceBenchReport struct {
	SchemaVersion   string                     `json:"schema_version"`
	GeneratedAtMS   int64                      `json:"generated_at_ms"`
	ExecutionMode   string                     `json:"execution_mode"`
	BaselineScope   string                     `json:"baseline_scope"`
	Gateway         string                     `json:"gateway"`
	Profile         string                     `json:"profile"`
	ProtocolVersion int                        `json:"protocol_version"`
	Input           xiaozhiVoiceBenchInput     `json:"input_audio"`
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
	Turn                   int                            `json:"turn"`
	Kind                   string                         `json:"kind"`
	TraceID                string                         `json:"trace_id"`
	SessionID              string                         `json:"session_id"`
	HelloAccepted          bool                           `json:"hello_accepted"`
	ListenAck              bool                           `json:"listen_ack"`
	BinaryDownlinkFrames   int                            `json:"binary_downlink_frames"`
	DownlinkAudioQuality   *audio.PCMQualityReport        `json:"downlink_audio_quality,omitempty"`
	FirstAudioMS           *int64                         `json:"first_audio_ms,omitempty"`
	TTSStopReceived        bool                           `json:"tts_stop_received"`
	AbortSent              bool                           `json:"abort_sent"`
	AbortStopMS            *int64                         `json:"abort_stop_ms,omitempty"`
	MetricsObserved        bool                           `json:"metrics_observed"`
	TraceSummary           *xiaozhiVoiceBenchTraceSummary `json:"trace_summary,omitempty"`
	VoicePipelineExecution xiaozhiVoiceBenchExecution     `json:"-"`
	Status                 string                         `json:"status"`
	Findings               []string                       `json:"findings,omitempty"`
}

type xiaozhiVoiceBenchTraceSummary struct {
	EventCount                    int    `json:"event_count"`
	LastOffsetMS                  int64  `json:"last_offset_ms"`
	BargeInStopMS                 *int64 `json:"barge_in_stop_ms,omitempty"`
	XiaozhiListenToAudioIngressMS *int64 `json:"xiaozhi_listen_to_audio_ingress_ms,omitempty"`
	XiaozhiOpusDecodeMS           *int64 `json:"xiaozhi_opus_decode_ms,omitempty"`
	ASRFirstPartialMS             *int64 `json:"asr_first_partial_ms,omitempty"`
	ASRFinalMS                    *int64 `json:"asr_final_ms,omitempty"`
	LLMFirstContentMS             *int64 `json:"llm_first_content_ms,omitempty"`
	TTSFirstAudioMS               *int64 `json:"tts_first_audio_ms,omitempty"`
	AudioDownlinkFirstFrameMS     *int64 `json:"audio_downlink_first_frame_ms,omitempty"`
	DevicePlaybackStartMS         *int64 `json:"device_playback_start_ms,omitempty"`
	AnswerFirstAudioTotalMS       *int64 `json:"answer_first_audio_total_ms,omitempty"`
}

type xiaozhiVoiceBenchTraceEvent struct {
	Name string `json:"name"`
	AtMS int64  `json:"at_ms"`
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
	ProviderExecuted           bool   `json:"provider_executed"`
	V21Executed                bool   `json:"v21_executed"`
	HardwareExecuted           bool   `json:"hardware_executed"`
	VoicePipelineObserved      bool   `json:"voice_pipeline_observed"`
	VoicePipelineExecutionMode string `json:"voice_pipeline_execution_mode"`
	ASRProfile                 string `json:"asr_profile,omitempty"`
	ASRProfileEnv              string `json:"asr_profile_env,omitempty"`
	LLMProfile                 string `json:"llm_profile,omitempty"`
	LLMProfileEnv              string `json:"llm_profile_env,omitempty"`
	TTSProfile                 string `json:"tts_profile,omitempty"`
	TTSProfileEnv              string `json:"tts_profile_env,omitempty"`
	HostLocalASRExecuted       bool   `json:"host_local_asr_executed"`
	HostLocalTextExecuted      bool   `json:"host_local_text_executed"`
	HostLocalTTSExecuted       bool   `json:"host_local_tts_executed"`
	HostProductChainReady      bool   `json:"host_product_chain_ready"`
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
			fmt.Fprintln(stdout, "a21 xiaozhi-voice-bench [--gateway-url http://127.0.0.1:21080] [--profile xiaozhi|a21-debug] [--device-id stackchan-virtual-a21-bench-001] [--protocol-version 1|2|3] [--input-wav fixture.wav] [--repeat 3] [--timeout-ms 5000] [--require-product-chain] [--output-dir reports]")
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
		case "--require-product-chain":
			options.RequireProductChain = true
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
	inputWAV := strings.TrimSpace(options.InputWAV)
	if inputWAV != "" {
		if xiaozhiVoiceBenchContainsLegacy(inputWAV) {
			return fmt.Errorf("input wav contains legacy identity")
		}
		if strings.ToLower(filepath.Ext(inputWAV)) != ".wav" {
			return fmt.Errorf("--input-wav must point to a wav file")
		}
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
			ProviderExecuted:           false,
			V21Executed:                false,
			HardwareExecuted:           false,
			VoicePipelineExecutionMode: "unknown",
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
	packets, input, err := xiaozhiVoiceBenchOpusPackets(options)
	if err != nil {
		report.Findings = append(report.Findings, xiaozhiVoiceBenchFinding{Code: "opus_fixture_unavailable", Message: "Opus uplink fixture could not be generated"})
		report.Counts.FailureCount = len(report.Findings)
		return report
	}
	report.Input = input
	for turn := 1; turn <= options.Repeat; turn++ {
		traceID := fmt.Sprintf("a21-trace-xiaozhi-bench-%d-answer-%02d", generatedAtMS, turn)
		sessionID := fmt.Sprintf("a21-session-xiaozhi-bench-%d-answer-%02d", generatedAtMS, turn)
		receipt := runXiaozhiVoiceBenchTurn(ctx, options, wsURL, packets, turn, "answer", traceID, sessionID, false)
		attachXiaozhiVoiceBenchTraceSummary(ctx, options.GatewayURL, &receipt)
		report.AnswerTurns = append(report.AnswerTurns, receipt)
	}
	for turn := 1; turn <= options.Repeat; turn++ {
		traceID := fmt.Sprintf("a21-trace-xiaozhi-bench-%d-barge-%02d", generatedAtMS, turn)
		sessionID := fmt.Sprintf("a21-session-xiaozhi-bench-%d-barge-%02d", generatedAtMS, turn)
		receipt := runXiaozhiVoiceBenchTurn(ctx, options, wsURL, packets, turn, "barge_in", traceID, sessionID, true)
		attachXiaozhiVoiceBenchTraceSummary(ctx, options.GatewayURL, &receipt)
		report.BargeInTurns = append(report.BargeInTurns, receipt)
	}
	report.Counts.AnswerTurnCount = len(report.AnswerTurns)
	report.Counts.BargeInTurnCount = len(report.BargeInTurns)
	report.Summary = summarizeXiaozhiVoiceBench(report.AnswerTurns, report.BargeInTurns)
	report.Execution = summarizeXiaozhiVoiceBenchExecution(report.AnswerTurns, report.BargeInTurns)
	if options.RequireProductChain && !xiaozhiVoiceBenchProductChainReady(report.Execution) {
		report.Findings = append(report.Findings, xiaozhiVoiceBenchFinding{
			Code:    "product_chain_not_executed",
			Message: "xiaozhi voice bench did not observe non-fixture ASR, text stream, and TTS stages in one product chain",
		})
	}
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

func runXiaozhiVoiceBenchTurn(ctx context.Context, options xiaozhiVoiceBenchOptions, wsURL string, packets [][]byte, turn int, kind string, traceID string, sessionID string, abortAfterFirstAudio bool) xiaozhiVoiceBenchTurn {
	receipt := xiaozhiVoiceBenchTurn{
		Turn:      turn,
		Kind:      kind,
		TraceID:   traceID,
		SessionID: sessionID,
		Status:    "failed",
	}
	turnCtx, cancel := context.WithTimeout(ctx, time.Duration(options.TimeoutMS)*time.Millisecond)
	defer cancel()
	conn, resp, err := websocket.Dial(turnCtx, wsURL, &websocket.DialOptions{
		HTTPClient: a21DirectHTTPClient(time.Duration(options.TimeoutMS) * time.Millisecond),
	})
	if err != nil {
		receipt.Findings = append(receipt.Findings, xiaozhiVoiceBenchWebSocketFailureFinding(resp, err))
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
	for _, packet := range packets {
		if err := conn.Write(turnCtx, websocket.MessageBinary, wrapXiaozhiVoiceBenchOpus(packet, options.ProtocolVersion)); err != nil {
			receipt.Findings = append(receipt.Findings, "opus_uplink_send_failed")
			return receipt
		}
	}
	if err := wsjson.Write(turnCtx, conn, xiaozhiVoiceBenchListen(options, traceID, sessionID, "stop")); err != nil {
		receipt.Findings = append(receipt.Findings, "listen_stop_send_failed")
		return receipt
	}
	downlinkCodec, err := opuscodec.New(24000, 1, 60)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "downlink_opus_decoder_unavailable")
		return receipt
	}
	var downlinkPCM []byte
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
			pcm, err := downlinkCodec.DecodePCM16(data)
			if err != nil {
				receipt.Findings = append(receipt.Findings, "downlink_opus_decode_failed")
				return receipt
			}
			downlinkPCM = appendPCM16LE(downlinkPCM, pcm)
			if quality, err := audio.AnalyzePCM16LEQuality("pcm_s16le", 24000, 1, 0, downlinkPCM); err == nil {
				quality.Codec = "opus_decoded_pcm_s16le"
				receipt.DownlinkAudioQuality = &quality
			}
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

func appendPCM16LE(out []byte, pcm []int16) []byte {
	for _, sample := range pcm {
		out = binary.LittleEndian.AppendUint16(out, uint16(sample))
	}
	return out
}

func summarizeXiaozhiVoiceBenchExecution(answerTurns []xiaozhiVoiceBenchTurn, bargeInTurns []xiaozhiVoiceBenchTurn) xiaozhiVoiceBenchExecution {
	summary := xiaozhiVoiceBenchExecution{VoicePipelineExecutionMode: "unknown"}
	for _, turn := range append(append([]xiaozhiVoiceBenchTurn(nil), answerTurns...), bargeInTurns...) {
		execution := turn.VoicePipelineExecution
		if !execution.VoicePipelineObserved {
			continue
		}
		if !summary.VoicePipelineObserved {
			summary = execution
			continue
		}
		summary.HostProductChainReady = summary.HostProductChainReady || execution.HostProductChainReady
		summary.HostLocalASRExecuted = summary.HostLocalASRExecuted || execution.HostLocalASRExecuted
		summary.HostLocalTextExecuted = summary.HostLocalTextExecuted || execution.HostLocalTextExecuted
		summary.HostLocalTTSExecuted = summary.HostLocalTTSExecuted || execution.HostLocalTTSExecuted
		summary.ProviderExecuted = summary.ProviderExecuted || execution.ProviderExecuted || execution.HostLocalTextExecuted
		if summary.VoicePipelineExecutionMode != execution.VoicePipelineExecutionMode {
			summary.VoicePipelineExecutionMode = "mixed"
		}
		if summary.ASRProfile == "" {
			summary.ASRProfile = execution.ASRProfile
		}
		if summary.ASRProfileEnv == "" {
			summary.ASRProfileEnv = execution.ASRProfileEnv
		}
		if summary.LLMProfile == "" {
			summary.LLMProfile = execution.LLMProfile
		}
		if summary.LLMProfileEnv == "" {
			summary.LLMProfileEnv = execution.LLMProfileEnv
		}
		if summary.TTSProfile == "" {
			summary.TTSProfile = execution.TTSProfile
		}
		if summary.TTSProfileEnv == "" {
			summary.TTSProfileEnv = execution.TTSProfileEnv
		}
	}
	summary.V21Executed = false
	summary.HardwareExecuted = false
	return summary
}

func xiaozhiVoiceBenchExecutionFromPipeline(pipeline map[string]any) xiaozhiVoiceBenchExecution {
	execution := xiaozhiVoiceBenchExecution{
		VoicePipelineObserved:      true,
		VoicePipelineExecutionMode: xiaozhiVoiceBenchSafeExecutionMode(xiaozhiVoiceBenchStringField(pipeline, "execution_mode")),
	}
	if selection, ok := pipeline["selection"].(map[string]any); ok {
		execution.ASRProfile = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "asr_profile"), false)
		execution.ASRProfileEnv = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "asr_profile_env"), true)
		execution.LLMProfile = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "llm_profile"), false)
		execution.LLMProfileEnv = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "llm_profile_env"), true)
		execution.TTSProfile = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "tts_profile"), false)
		execution.TTSProfileEnv = xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(selection, "tts_profile_env"), true)
	}
	if stage := xiaozhiVoiceBenchSafeIdentifier(xiaozhiVoiceBenchStringField(pipeline, "stage"), false); stage != "" && stage != "answer" {
		return execution
	}
	if execution.VoicePipelineExecutionMode == "host_local" {
		execution.HostLocalASRExecuted = xiaozhiVoiceBenchNonMockStageProfile(execution.ASRProfile)
		execution.HostLocalTextExecuted = xiaozhiVoiceBenchNonMockStageProfile(execution.LLMProfile)
		execution.HostLocalTTSExecuted = xiaozhiVoiceBenchNonMockStageProfile(execution.TTSProfile)
		execution.ProviderExecuted = execution.HostLocalTextExecuted
		execution.HostProductChainReady = xiaozhiVoiceBenchProductChainReady(execution)
	} else if execution.VoicePipelineExecutionMode == "cloud_edge" {
		execution.ProviderExecuted = xiaozhiVoiceBenchNonMockStageProfile(execution.ASRProfile) &&
			xiaozhiVoiceBenchNonMockStageProfile(execution.LLMProfile) &&
			xiaozhiVoiceBenchNonMockStageProfile(execution.TTSProfile)
		execution.HostProductChainReady = execution.ProviderExecuted
	}
	return execution
}

func xiaozhiVoiceBenchProductChainReady(execution xiaozhiVoiceBenchExecution) bool {
	if execution.HostProductChainReady {
		return true
	}
	return execution.VoicePipelineObserved &&
		((execution.VoicePipelineExecutionMode == "host_local" &&
			execution.HostLocalASRExecuted &&
			execution.HostLocalTextExecuted &&
			execution.HostLocalTTSExecuted) ||
			(execution.VoicePipelineExecutionMode == "cloud_edge" &&
				execution.ProviderExecuted))
}

func xiaozhiVoiceBenchNonMockStageProfile(profile string) bool {
	profile = strings.ToLower(strings.TrimSpace(profile))
	return profile != "" &&
		profile != "mock" &&
		!strings.HasPrefix(profile, "mock-") &&
		!strings.HasPrefix(profile, "mock_")
}

func xiaozhiVoiceBenchStringField(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func xiaozhiVoiceBenchSafeExecutionMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "fixture":
		return "fixture"
	case "host_local":
		return "host_local"
	case "cloud_edge":
		return "cloud_edge"
	default:
		return "unknown"
	}
}

func xiaozhiVoiceBenchSafeIdentifier(value string, requireA21Env bool) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 || xiaozhiVoiceBenchContainsUnsafeIdentifier(value) {
		return ""
	}
	if requireA21Env && !strings.HasPrefix(value, "A21_") {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return ""
	}
	return value
}

func xiaozhiVoiceBenchContainsUnsafeIdentifier(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"x21", "v21", "http://", "https://", "/", "\\", ":", "@", "key", "token", "secret", "proxy", "prompt", "transcript", "output"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func readXiaozhiVoiceBenchJSON(ctx context.Context, conn *websocket.Conn) (map[string]any, bool) {
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return nil, false
		}
		if messageType == websocket.MessageBinary {
			continue
		}
		if messageType != websocket.MessageText {
			return nil, false
		}
		var message map[string]any
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, false
		}
		return message, true
	}
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
		if pipeline, ok := message["voice_pipeline"].(map[string]any); ok {
			receipt.MetricsObserved = true
			receipt.VoicePipelineExecution = xiaozhiVoiceBenchExecutionFromPipeline(pipeline)
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

func xiaozhiVoiceBenchOpusPackets(options xiaozhiVoiceBenchOptions) ([][]byte, xiaozhiVoiceBenchInput, error) {
	if strings.TrimSpace(options.InputWAV) != "" {
		packets, err := xiaozhiVoiceBenchOpusPacketsFromWAV(options.InputWAV)
		if err != nil {
			return nil, xiaozhiVoiceBenchInput{}, err
		}
		return packets, xiaozhiVoiceBenchInput{
			Source:         "wav_fixture",
			WAVName:        filepath.Base(filepath.Clean(options.InputWAV)),
			OpusFrameCount: len(packets),
		}, nil
	}
	packet, err := xiaozhiVoiceBenchSyntheticOpusPacket()
	if err != nil {
		return nil, xiaozhiVoiceBenchInput{}, err
	}
	return [][]byte{packet}, xiaozhiVoiceBenchInput{
		Source:         "synthetic_sine",
		OpusFrameCount: 1,
	}, nil
}

func xiaozhiVoiceBenchOpusPacketsFromWAV(path string) ([][]byte, error) {
	chunks, err := audio.ReadPCM16MonoWAVChunks(path, 60)
	if err != nil {
		return nil, err
	}
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		return nil, err
	}
	packets := make([][]byte, 0, len(chunks))
	for _, chunk := range chunks {
		data, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			return nil, err
		}
		if len(data)%2 != 0 {
			return nil, fmt.Errorf("wav pcm chunk must contain 16-bit samples")
		}
		pcm := make([]int16, len(data)/2)
		for i := range pcm {
			pcm[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		}
		packet, err := codec.EncodePCM16(pcm)
		if err != nil {
			return nil, err
		}
		packets = append(packets, packet)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("wav fixture produced no Opus frames")
	}
	return packets, nil
}

func xiaozhiVoiceBenchSyntheticOpusPacket() ([]byte, error) {
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

func xiaozhiVoiceBenchWebSocketFailureFinding(resp *http.Response, err error) string {
	const base = "gateway_websocket_unavailable"
	if resp != nil && resp.StatusCode > 0 {
		return fmt.Sprintf("%s_http_%d", base, resp.StatusCode)
	}
	if err == nil {
		return base
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "proxy") || strings.Contains(message, "tunnel"):
		return base + "_proxy_or_tunnel"
	case strings.Contains(message, "connection refused"):
		return base + "_connection_refused"
	case strings.Contains(message, "operation not permitted"):
		return base + "_operation_not_permitted"
	case strings.Contains(message, "deadline exceeded") || strings.Contains(message, "i/o timeout"):
		return base + "_timeout"
	case strings.Contains(message, "no such host") || strings.Contains(message, "lookup"):
		return base + "_dns"
	case strings.Contains(message, "expected handshake response status code"):
		return base + "_bad_status"
	case strings.Contains(message, "sec-websocket"):
		return base + "_bad_handshake"
	default:
		return base + "_dial_error"
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

func attachXiaozhiVoiceBenchTraceSummary(ctx context.Context, gatewayURL string, receipt *xiaozhiVoiceBenchTurn) {
	if receipt == nil || receipt.TraceID == "" {
		return
	}
	summary, err := fetchXiaozhiVoiceBenchTraceSummary(ctx, gatewayURL, receipt.TraceID)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "trace_summary_unavailable")
		return
	}
	receipt.TraceSummary = &summary
	if receipt.Kind == "barge_in" {
		if summary.BargeInStopMS == nil && receipt.AbortStopMS != nil {
			summary.BargeInStopMS = receipt.AbortStopMS
			receipt.TraceSummary = &summary
		}
		if !xiaozhiVoiceBenchBargeInTraceMetricsPresent(summary) {
			receipt.Findings = append(receipt.Findings, "trace_barge_in_metrics_missing")
		}
		return
	}
	if !xiaozhiVoiceBenchCoreTraceMetricsPresent(summary) {
		receipt.Findings = append(receipt.Findings, "trace_core_stage_metrics_missing")
	}
}

func xiaozhiVoiceBenchBargeInTraceMetricsPresent(summary xiaozhiVoiceBenchTraceSummary) bool {
	return summary.BargeInStopMS != nil &&
		summary.AnswerFirstAudioTotalMS != nil
}

func xiaozhiVoiceBenchCoreTraceMetricsPresent(summary xiaozhiVoiceBenchTraceSummary) bool {
	return summary.ASRFirstPartialMS != nil &&
		summary.ASRFinalMS != nil &&
		summary.LLMFirstContentMS != nil &&
		summary.TTSFirstAudioMS != nil &&
		summary.AudioDownlinkFirstFrameMS != nil &&
		summary.AnswerFirstAudioTotalMS != nil
}

func fetchXiaozhiVoiceBenchTraceSummary(ctx context.Context, gatewayURL string, traceID string) (xiaozhiVoiceBenchTraceSummary, error) {
	endpoint, err := xiaozhiVoiceBenchTraceURL(gatewayURL, traceID)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	client := *a21DirectHTTPClient(2 * time.Second)
	response, err := client.Do(request)
	if err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return xiaozhiVoiceBenchTraceSummary{}, fmt.Errorf("trace summary unavailable")
	}
	var decoded struct {
		Events  []xiaozhiVoiceBenchTraceEvent `json:"events"`
		Summary xiaozhiVoiceBenchTraceSummary `json:"summary"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return xiaozhiVoiceBenchTraceSummary{}, err
	}
	if decoded.Summary.EventCount <= 0 {
		return xiaozhiVoiceBenchTraceSummary{}, fmt.Errorf("trace summary empty")
	}
	if decoded.Summary.ASRFinalMS == nil {
		decoded.Summary.ASRFinalMS = xiaozhiVoiceBenchTraceDeltaMS(decoded.Events, "audio.ingress.buffered", "asr.final")
	}
	return decoded.Summary, nil
}

func xiaozhiVoiceBenchTraceDeltaMS(events []xiaozhiVoiceBenchTraceEvent, startName string, endName string) *int64 {
	var startAtMS int64
	hasStart := false
	for _, event := range events {
		switch {
		case event.Name == startName && !hasStart:
			startAtMS = event.AtMS
			hasStart = true
		case event.Name == endName && hasStart:
			delta := event.AtMS - startAtMS
			if delta < 0 {
				delta = 0
			}
			return &delta
		}
	}
	return nil
}

func xiaozhiVoiceBenchTraceURL(gatewayURL string, traceID string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid gateway URL")
	}
	parsed.Path = "/v1/traces"
	query := url.Values{}
	query.Set("trace_id", traceID)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
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
