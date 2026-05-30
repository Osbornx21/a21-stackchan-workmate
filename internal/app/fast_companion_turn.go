package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/protocol"
)

type stackChanFastCompanionTurnReport struct {
	SchemaVersion           string                               `json:"schema_version"`
	GeneratedAtMS           int64                                `json:"generated_at_ms"`
	Metadata                latencyBenchMetadata                 `json:"metadata"`
	Status                  string                               `json:"status"`
	GatewayURL              string                               `json:"gateway_url"`
	DeviceID                string                               `json:"device_id"`
	Repeat                  int                                  `json:"repeat"`
	DeviceOnline            bool                                 `json:"device_online"`
	ListenSource            string                               `json:"listen_source"`
	M3Candidate             bool                                 `json:"m3_candidate"`
	LocalAckFirstAudioP50MS float64                              `json:"local_ack_first_audio_p50_ms,omitempty"`
	LocalAckFirstAudioP95MS float64                              `json:"local_ack_first_audio_p95_ms,omitempty"`
	AnswerFirstAudioP50MS   float64                              `json:"answer_first_audio_total_p50_ms,omitempty"`
	AnswerFirstAudioP95MS   float64                              `json:"answer_first_audio_total_p95_ms,omitempty"`
	BargeInStopP95MS        float64                              `json:"barge_in_stop_p95_ms,omitempty"`
	DeviceFirmware          firmwarecheck.DeviceIdentityFirmware `json:"device_firmware,omitempty"`
	Turns                   []stackChanFastCompanionTurnReceipt  `json:"turns"`
	ReportPath              string                               `json:"report_path,omitempty"`
	Findings                []string                             `json:"findings,omitempty"`
}

type stackChanFastCompanionTurnReceipt struct {
	Turn                    int      `json:"turn"`
	TraceID                 string   `json:"trace_id"`
	SessionID               string   `json:"session_id"`
	ListenSource            string   `json:"listen_source"`
	ListenDetected          bool     `json:"listen_detected"`
	TextStreamProvider      string   `json:"text_stream_provider"`
	TextStreamExecuted      bool     `json:"text_stream_executed"`
	LocalAckFirstAudioMS    float64  `json:"local_ack_first_audio_ms,omitempty"`
	AnswerFirstAudioTotalMS float64  `json:"answer_first_audio_total_ms,omitempty"`
	LocalAckPlaybackChunks  int      `json:"local_ack_playback_chunks,omitempty"`
	LocalAckPlaybackBatches int      `json:"local_ack_playback_batches,omitempty"`
	AnswerPlaybackChunks    int      `json:"answer_playback_chunks,omitempty"`
	AnswerPlaybackBatches   int      `json:"answer_playback_batches,omitempty"`
	PlaybackCleared         bool     `json:"playback_cleared"`
	DeliveredTransport      string   `json:"delivered_transport,omitempty"`
	M3Candidate             bool     `json:"m3_candidate"`
	TraceMarkers            []string `json:"trace_markers,omitempty"`
	Findings                []string `json:"findings,omitempty"`
}

type stackChanFastCompanionTurnOptions struct {
	GatewayURL          string
	DeviceID            string
	Engine              string
	Text                string
	Voice               string
	ModelDir            string
	SpeakerID           int
	ASRProvider         string
	ASRFamily           string
	ASRModelDir         string
	ASRWAVPath          string
	TextProvider        string
	ExecuteTextProvider bool
	ListenSource        string
	Repeat              int
	OutputDir           string
}

type stackChanAudioDelivery struct {
	Chunks             int
	Batches            int
	DeliveredTransport string
}

func runStackChanFastCompanionTurn(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanFastCompanionTurnOptions{
		GatewayURL:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:     firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		Engine:       strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx")),
		Voice:        strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE")),
		ModelDir:     strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR")),
		SpeakerID:    parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21),
		ASRProvider:  strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_ASR_PROVIDER"), "mock_asr")),
		ASRFamily:    strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY")),
		ASRModelDir:  strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR")),
		ASRWAVPath:   strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_WAV")),
		TextProvider: strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TEXT_PROVIDER"), "mock_text_stream")),
		ListenSource: "host_fixture",
		Text:         "A21 fast companion turn.",
		Repeat:       1,
		OutputDir:    "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-fast-companion-turn [--gateway-url http://127.0.0.1:21080] [--device-id stackchan-001] [--engine sherpa_onnx|macos_say] [--asr-provider mock_asr|sherpa_onnx] [--text-provider mock_text_stream|deepseek] [--execute-text-provider] [--listen-source host_fixture|stackchan_mic] [--repeat 3] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--engine":
			if !readStringOption(args, &i, stderr, "--engine", &options.Engine) {
				return 2
			}
		case "--text":
			if !readStringOption(args, &i, stderr, "--text", &options.Text) {
				return 2
			}
		case "--voice":
			if !readStringOption(args, &i, stderr, "--voice", &options.Voice) {
				return 2
			}
		case "--model-dir":
			if !readStringOption(args, &i, stderr, "--model-dir", &options.ModelDir) {
				return 2
			}
		case "--speaker-id":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--speaker-id")
			if !ok {
				return 2
			}
			options.SpeakerID = value
		case "--asr-provider":
			if !readStringOption(args, &i, stderr, "--asr-provider", &options.ASRProvider) {
				return 2
			}
		case "--asr-family":
			if !readStringOption(args, &i, stderr, "--asr-family", &options.ASRFamily) {
				return 2
			}
		case "--asr-model-dir":
			if !readStringOption(args, &i, stderr, "--asr-model-dir", &options.ASRModelDir) {
				return 2
			}
		case "--asr-wav":
			if !readStringOption(args, &i, stderr, "--asr-wav", &options.ASRWAVPath) {
				return 2
			}
		case "--text-provider":
			if !readStringOption(args, &i, stderr, "--text-provider", &options.TextProvider) {
				return 2
			}
		case "--execute-text-provider":
			options.ExecuteTextProvider = true
		case "--listen-source":
			if !readStringOption(args, &i, stderr, "--listen-source", &options.ListenSource) {
				return 2
			}
		case "--repeat":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--repeat")
			if !ok {
				return 2
			}
			options.Repeat = value
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown stackchan-fast-companion-turn option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan fast companion turn report dir invalid: %v\n", err)
		return 1
	}
	report, buildErr := buildStackChanFastCompanionTurnReport(context.Background(), options)
	if buildErr != nil && report.SchemaVersion == "" {
		fmt.Fprintf(stderr, "stackchan fast companion turn failed: %v\n", buildErr)
		return 1
	}
	reportPath, err := writeStackChanFastCompanionTurnReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan fast companion turn report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanFastCompanionTurn(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan fast companion turn report: %v\n", err)
		return 1
	}
	if buildErr != nil {
		fmt.Fprintf(stderr, "stackchan fast companion turn failed: %v\n", buildErr)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func readStringOption(args []string, index *int, stderr io.Writer, name string, target *string) bool {
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		fmt.Fprintf(stderr, "%s requires a value\n", name)
		return false
	}
	*index = *index + 1
	*target = args[*index]
	return true
}

func buildStackChanFastCompanionTurnReport(ctx context.Context, options stackChanFastCompanionTurnOptions) (stackChanFastCompanionTurnReport, error) {
	if options.Repeat <= 0 {
		options.Repeat = 1
	}
	options.ListenSource = normalizeFastCompanionListenSource(options.ListenSource)
	generatedAtMS := time.Now().UnixMilli()
	report := stackChanFastCompanionTurnReport{
		SchemaVersion: "a21.stackchan_fast_companion_turn.v1",
		GeneratedAtMS: generatedAtMS,
		Metadata:      buildLatencyBenchMetadata(),
		Status:        "failed",
		GatewayURL:    sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:      options.DeviceID,
		Repeat:        options.Repeat,
		ListenSource:  options.ListenSource,
		M3Candidate:   false,
	}
	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.Findings = append(report.Findings, "gateway device report failed")
		return report, err
	}
	device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
	if !ok {
		report.Findings = append(report.Findings, "device missing from Gateway")
		return report, nil
	}
	report.DeviceFirmware = device.Firmware
	report.DeviceOnline = device.ConnectionStatus == "" || device.ConnectionStatus == "online"
	if !report.DeviceOnline {
		report.Findings = append(report.Findings, "device is not online")
		return report, nil
	}
	if options.ListenSource == "stackchan_mic" {
		report.Findings = append(report.Findings, "stackchan_mic listen source requires physical mic turn evidence")
		return report, nil
	}

	ackSamples := make([]time.Duration, 0, options.Repeat)
	answerSamples := make([]time.Duration, 0, options.Repeat)
	bargeSamples := make([]time.Duration, 0, options.Repeat)
	allPassed := true
	for turn := 1; turn <= options.Repeat; turn++ {
		receipt, err := runStackChanFastCompanionSingleTurn(ctx, options, turn, generatedAtMS)
		if err != nil {
			allPassed = false
			report.Findings = append(report.Findings, fmt.Sprintf("turn %d failed", turn))
			report.Turns = append(report.Turns, receipt)
			return report, err
		}
		if len(receipt.Findings) > 0 {
			allPassed = false
		}
		report.Turns = append(report.Turns, receipt)
		if receipt.LocalAckFirstAudioMS > 0 {
			ackSamples = append(ackSamples, time.Duration(receipt.LocalAckFirstAudioMS*1000)*time.Microsecond)
		}
		if receipt.AnswerFirstAudioTotalMS > 0 {
			answerSamples = append(answerSamples, time.Duration(receipt.AnswerFirstAudioTotalMS*1000)*time.Microsecond)
		}
	}
	if len(ackSamples) > 0 {
		report.LocalAckFirstAudioP50MS = percentileMS(ackSamples, 0.50)
		report.LocalAckFirstAudioP95MS = percentileMS(ackSamples, 0.95)
	}
	if len(answerSamples) > 0 {
		report.AnswerFirstAudioP50MS = percentileMS(answerSamples, 0.50)
		report.AnswerFirstAudioP95MS = percentileMS(answerSamples, 0.95)
	}
	if len(bargeSamples) > 0 {
		report.BargeInStopP95MS = percentileMS(bargeSamples, 0.95)
	}
	if options.ListenSource != "stackchan_mic" {
		report.Findings = append(report.Findings, "not an M3 candidate: listen source is not stackchan_mic")
	}
	report.M3Candidate = allPassed && options.ListenSource == "stackchan_mic" && report.Repeat >= 3 && report.AnswerFirstAudioP95MS > 0 && report.AnswerFirstAudioP95MS < 1500
	report.Status = "passed"
	return report, nil
}

func runStackChanFastCompanionSingleTurn(ctx context.Context, options stackChanFastCompanionTurnOptions, turn int, generatedAtMS int64) (stackChanFastCompanionTurnReceipt, error) {
	traceID := fmt.Sprintf("a21-trace-fast-companion-%d-%02d", generatedAtMS, turn)
	sessionID := fmt.Sprintf("a21-session-fast-companion-%d-%02d", generatedAtMS, turn)
	receipt := stackChanFastCompanionTurnReceipt{
		Turn:           turn,
		TraceID:        traceID,
		SessionID:      sessionID,
		ListenSource:   options.ListenSource,
		ListenDetected: options.ListenSource == "stackchan_mic",
		TraceMarkers: []string{
			"fast_companion.turn.start",
			"local_ack.playback.start",
			"answer.playback.start",
		},
	}
	loopback, err := buildLocalVoiceLoopbackReport(ctx, localTTSRuntimeOptions{
		Engine:    options.Engine,
		Text:      options.Text,
		Voice:     options.Voice,
		ModelDir:  options.ModelDir,
		SpeakerID: options.SpeakerID,
		OutputDir: options.OutputDir,
	}, 1, localVoiceLoopbackTextStreamOptions{
		Provider: options.TextProvider,
		Execute:  options.ExecuteTextProvider,
		Env:      os.Environ(),
	}, localVoiceLoopbackASROptions{
		Provider:  options.ASRProvider,
		Family:    options.ASRFamily,
		ModelDir:  options.ASRModelDir,
		WAVPath:   options.ASRWAVPath,
		OutputDir: options.OutputDir,
	})
	if err != nil {
		receipt.Findings = append(receipt.Findings, "local loopback failed")
		return receipt, err
	}
	if loopback.Status != "passed" {
		receipt.Findings = append(receipt.Findings, "local loopback did not pass")
		return receipt, nil
	}
	receipt.TextStreamProvider = loopback.TextStreamProvider
	receipt.TextStreamExecuted = loopback.TextStreamExecuted
	receipt.LocalAckFirstAudioMS = loopback.LocalAckFirstAudioTotalMS
	receipt.AnswerFirstAudioTotalMS = loopback.AnswerFirstAudioP95MS
	ackDelivery, err := deliverStackChanAudioFile(ctx, options.GatewayURL, options.DeviceID, traceID, sessionID, fmt.Sprintf("a21-fast-companion-ack-%d-%02d", generatedAtMS, turn), loopback.LocalAckAudioPath)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "local ack playback delivery failed")
		return receipt, err
	}
	answerDelivery, err := deliverStackChanAudioFile(ctx, options.GatewayURL, options.DeviceID, traceID, sessionID, fmt.Sprintf("a21-fast-companion-answer-%d-%02d", generatedAtMS, turn), loopback.TTSAudioPath)
	if err != nil {
		receipt.Findings = append(receipt.Findings, "answer playback delivery failed")
		return receipt, err
	}
	receipt.LocalAckPlaybackChunks = ackDelivery.Chunks
	receipt.LocalAckPlaybackBatches = ackDelivery.Batches
	receipt.AnswerPlaybackChunks = answerDelivery.Chunks
	receipt.AnswerPlaybackBatches = answerDelivery.Batches
	receipt.DeliveredTransport = firstNonEmpty(answerDelivery.DeliveredTransport, ackDelivery.DeliveredTransport)
	if _, err := postStackChanSpeakerControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", traceID, sessionID, fmt.Sprintf("a21-fast-companion-answer-%d-%02d", generatedAtMS, turn), 0); err != nil {
		receipt.Findings = append(receipt.Findings, "playback idle delivery failed")
		return receipt, err
	}
	receipt.PlaybackCleared = true
	if options.ListenSource != "stackchan_mic" {
		receipt.Findings = append(receipt.Findings, "host fixture listen source is not M3 evidence")
	}
	receipt.M3Candidate = options.ListenSource == "stackchan_mic" && receipt.LocalAckPlaybackChunks > 0 && receipt.AnswerPlaybackChunks > 0 && receipt.AnswerFirstAudioTotalMS < 1500
	return receipt, nil
}

func deliverStackChanAudioFile(ctx context.Context, gatewayURL string, deviceID string, traceID string, sessionID string, streamID string, wavPath string) (stackChanAudioDelivery, error) {
	if strings.TrimSpace(wavPath) == "" {
		return stackChanAudioDelivery{}, fmt.Errorf("audio path is required")
	}
	pcmChunks, err := audio.ReadPCM16MonoWAVChunks(wavPath, 20)
	if err != nil {
		return stackChanAudioDelivery{}, err
	}
	delivery := stackChanAudioDelivery{Chunks: len(pcmChunks)}
	for offset := 0; offset < len(pcmChunks); offset += stackChanSpeakerProbeBatchChunks {
		end := int(math.Min(float64(offset+stackChanSpeakerProbeBatchChunks), float64(len(pcmChunks))))
		batch := make([]protocol.AudioPlaybackChunk, 0, end-offset)
		for _, chunk := range pcmChunks[offset:end] {
			batch = append(batch, protocol.AudioPlaybackChunk{
				StreamID:     streamID,
				Codec:        protocol.AudioCodecPCMS16LE,
				SampleRateHz: chunk.SampleRateHz,
				Channels:     chunk.Channels,
				DurationMS:   chunk.DurationMS,
				DataBase64:   chunk.DataBase64,
			})
		}
		response, err := postStackChanAudioPlaybackBatch(gatewayURL, deviceID, traceID, sessionID, streamID, batch)
		if err != nil {
			return delivery, err
		}
		delivery.Batches++
		delivery.DeliveredTransport = firstNonEmpty(response.DeliveredTransport, delivery.DeliveredTransport)
		select {
		case <-ctx.Done():
			return delivery, ctx.Err()
		case <-time.After(time.Duration((end-offset)*20) * time.Millisecond):
		}
	}
	return delivery, nil
}

func normalizeFastCompanionListenSource(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "stackchan_mic":
		return "stackchan_mic"
	default:
		return "host_fixture"
	}
}

func writeStackChanFastCompanionTurnReport(outputDir string, report stackChanFastCompanionTurnReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-fast-companion-turn-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanFastCompanionTurn(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStackChanFastCompanionTurn(writer io.Writer, report stackChanFastCompanionTurnReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
