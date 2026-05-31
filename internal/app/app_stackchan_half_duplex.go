package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanHalfDuplexAcceptanceOptions struct {
	GatewayURL        string
	DeviceID          string
	Commit            string
	WindowMS          int
	MinMicFrames      int
	MinPlaybackChunks int
	MinDeliveryRatio  float64
	OutputDir         string
}
type stackChanHalfDuplexAcceptanceThresholds struct {
	MinMicFrames      int     `json:"min_mic_frames"`
	MinPlaybackChunks int     `json:"min_playback_chunks"`
	MinDeliveryRatio  float64 `json:"min_delivery_ratio"`
}
type stackChanHalfDuplexAcceptanceReport struct {
	SchemaVersion                    string                                  `json:"schema_version"`
	GeneratedAtMS                    int64                                   `json:"generated_at_ms"`
	Metadata                         latencyBenchMetadata                    `json:"metadata"`
	DryRun                           bool                                    `json:"dry_run"`
	FlashAllowed                     bool                                    `json:"flash_allowed"`
	DeleteAllowed                    bool                                    `json:"delete_allowed"`
	HardwareAcceptanceScope          string                                  `json:"hardware_acceptance_scope"`
	HalfDuplexAcceptanceStatus       string                                  `json:"half_duplex_acceptance_status"`
	PhysicalSoundObserved            bool                                    `json:"physical_sound_observed"`
	GatewayURL                       string                                  `json:"gateway_url"`
	DeviceID                         string                                  `json:"device_id"`
	Commit                           string                                  `json:"commit"`
	WindowMS                         int                                     `json:"window_ms"`
	ControlTraceID                   string                                  `json:"control_trace_id,omitempty"`
	ControlSessionID                 string                                  `json:"control_session_id,omitempty"`
	WindowStartedAtMS                int64                                   `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                  int64                                   `json:"window_ended_at_ms,omitempty"`
	Firmware                         firmwarecheck.DeviceIdentityFirmware    `json:"firmware"`
	Capabilities                     map[string]string                       `json:"capabilities,omitempty"`
	Microphone                       string                                  `json:"microphone,omitempty"`
	Speaker                          string                                  `json:"speaker,omitempty"`
	RuntimeEcho                      map[string]string                       `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore                map[string]string                       `json:"runtime_echo_before,omitempty"`
	MicFramesCapturedDelta           int                                     `json:"mic_frames_captured_delta"`
	AudioWSSentAudioFramesDelta      int                                     `json:"audio_ws_sent_audio_frames_delta"`
	MicDriverErrorsDelta             int                                     `json:"mic_driver_errors_delta"`
	MicQueueDroppedFramesDelta       int                                     `json:"mic_queue_dropped_frames_delta"`
	PlaybackBufferTotalChunksDelta   int                                     `json:"playback_buffer_total_chunks_delta"`
	PlaybackBufferDroppedChunksDelta int                                     `json:"playback_buffer_dropped_chunks_delta"`
	SpeakerFramesPlayedDelta         int                                     `json:"speaker_frames_played_delta"`
	SpeakerDriverErrorsDelta         int                                     `json:"speaker_driver_errors_delta"`
	AudioWSDeliveryRatio             float64                                 `json:"audio_ws_delivery_ratio,omitempty"`
	GatewayIngressDeliveryRatio      float64                                 `json:"gateway_ingress_delivery_ratio,omitempty"`
	GatewayAudioFrameDelta           int                                     `json:"gateway_audio_frame_delta"`
	GatewayAudioIngressFramesDelta   int                                     `json:"gateway_audio_ingress_frames_delta"`
	GatewayAudioPlaybackChunkDelta   int                                     `json:"gateway_audio_playback_chunk_delta"`
	GatewayVADSpeechDelta            int                                     `json:"gateway_vad_speech_delta"`
	GatewayMetricsBefore             *stackChanMicProbeGatewayMetrics        `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter              *stackChanMicProbeGatewayMetrics        `json:"gateway_metrics_after,omitempty"`
	Thresholds                       stackChanHalfDuplexAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions              []string                                `json:"next_required_actions"`
	ReportPath                       string                                  `json:"report_path,omitempty"`
	Findings                         []officePreflightFinding                `json:"findings,omitempty"`
}

func runStackChanHalfDuplexAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanHalfDuplexAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check half-duplex --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1500] [--min-mic-frames 1] [--min-playback-chunks 1] [--min-delivery-ratio 0.95] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--window-ms":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--window-ms")
			if !ok {
				return 2
			}
			if value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be at most 120000")
				return 2
			}
			options.WindowMS = value
		case "--min-mic-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-mic-frames")
			if !ok {
				return 2
			}
			options.MinMicFrames = value
		case "--min-playback-chunks":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-playback-chunks")
			if !ok {
				return 2
			}
			options.MinPlaybackChunks = value
		case "--min-delivery-ratio":
			value, ok := parseRatioCLIOption(args, &i, stderr, "--min-delivery-ratio")
			if !ok {
				return 2
			}
			options.MinDeliveryRatio = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-half-duplex-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		fmt.Fprintf(stderr, "stackchan half-duplex acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan half-duplex acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanHalfDuplexAcceptanceReport(options)
	reportPath, err := writeStackChanHalfDuplexAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan half-duplex acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanHalfDuplexAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan half-duplex acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan half-duplex acceptance blocked (instrumented only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan half-duplex acceptance ok (instrumented only, no flash performed)")
	return 0
}
func defaultStackChanHalfDuplexAcceptanceOptions() stackChanHalfDuplexAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanHalfDuplexAcceptanceOptions{
		GatewayURL:        firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:          deviceID,
		Commit:            currentGitCommit(projectRoot),
		WindowMS:          1500,
		MinMicFrames:      1,
		MinPlaybackChunks: 1,
		MinDeliveryRatio:  0.95,
		OutputDir:         "reports",
	}
}
func buildStackChanHalfDuplexAcceptanceReport(options stackChanHalfDuplexAcceptanceOptions) stackChanHalfDuplexAcceptanceReport {
	report := stackChanHalfDuplexAcceptanceReport{
		SchemaVersion:              "a21.stackchan_half_duplex_acceptance.v1",
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		DryRun:                     true,
		FlashAllowed:               false,
		DeleteAllowed:              false,
		HardwareAcceptanceScope:    "mic_to_mock_playback",
		HalfDuplexAcceptanceStatus: "confirmed",
		PhysicalSoundObserved:      false,
		GatewayURL:                 sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                   options.DeviceID,
		Commit:                     options.Commit,
		WindowMS:                   options.WindowMS,
		Thresholds: stackChanHalfDuplexAcceptanceThresholds{
			MinMicFrames:      options.MinMicFrames,
			MinPlaybackChunks: options.MinPlaybackChunks,
			MinDeliveryRatio:  options.MinDeliveryRatio,
		},
		NextRequiredActions: []string{
			"Keep this report with the mic-probe flash receipt and Gateway device report.",
			"Treat this as an instrumented half-duplex mock loop, not real ASR/LLM/TTS or full-duplex acceptance.",
			"Run a later operator-observed acceptance before claiming human-audible conversational quality.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before half-duplex probe")
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanHalfDuplexDeviceIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeMicFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_frames_captured")
	beforeAudioWSFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "audio_ws_sent_audio_frames")
	beforeMicErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_driver_errors")
	beforeMicDrops := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_queue_dropped_frames")
	beforePlaybackTotal := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_total_chunks")
	beforePlaybackDrops := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_dropped_chunks")
	beforeSpeakerFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_frames_played")
	beforeSpeakerErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_driver_errors")
	if len(report.Findings) > 0 {
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}

	beforeMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-half-duplex-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-half-duplex-%d", report.GeneratedAtMS)
	control, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionListening, protocol.ModeWorkmate, "HALF DUPLEX", traceID, sessionID, false, true)
	if err != nil {
		report.addFinding("half_duplex_control_failed", err.Error())
		report.HalfDuplexAcceptanceStatus = "blocked"
		return report
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID
	report.WindowStartedAtMS = time.Now().UnixMilli()
	time.Sleep(time.Duration(options.WindowMS) * time.Millisecond)
	report.WindowEndedAtMS = time.Now().UnixMilli()

	afterGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_after_failed", err.Error())
	} else {
		report.GatewayURL = afterGatewayReport.GatewayURL
		afterDevice, ok := findFirmwareDeviceRecord(afterGatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after half-duplex probe")
		} else {
			validateStackChanHalfDuplexDeviceIdentity(&report, options, afterDevice)
			report.RuntimeEcho = afterDevice.RuntimeEcho
			report.MicFramesCapturedDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_frames_captured") - beforeMicFrames
			report.AudioWSSentAudioFramesDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "audio_ws_sent_audio_frames") - beforeAudioWSFrames
			report.MicDriverErrorsDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_driver_errors") - beforeMicErrors
			report.MicQueueDroppedFramesDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "mic_queue_dropped_frames") - beforeMicDrops
			report.PlaybackBufferTotalChunksDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "playback_buffer_total_chunks") - beforePlaybackTotal
			report.PlaybackBufferDroppedChunksDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "playback_buffer_dropped_chunks") - beforePlaybackDrops
			report.SpeakerFramesPlayedDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "speaker_frames_played") - beforeSpeakerFrames
			report.SpeakerDriverErrorsDelta = stackChanRuntimeEchoInt(&report.Findings, afterDevice.RuntimeEcho, "speaker_driver_errors") - beforeSpeakerErrors
		}
	}

	afterMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioFrameDelta = afterMetrics.AudioFrameTotal - beforeMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesDelta = afterMetrics.AudioIngressFramesTotal - beforeMetrics.AudioIngressFramesTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechDelta = afterMetrics.VADSpeechTotal - beforeMetrics.VADSpeechTotal
	}
	report.AudioWSDeliveryRatio = roundedStackChanDiagnosticRatio(report.AudioWSSentAudioFramesDelta, report.MicFramesCapturedDelta)
	report.GatewayIngressDeliveryRatio = roundedStackChanDiagnosticRatio(report.GatewayAudioIngressFramesDelta, report.AudioWSSentAudioFramesDelta)
	validateStackChanHalfDuplexDeltas(&report, options)

	if _, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, false, false); err != nil {
		report.addFinding("half_duplex_idle_control_failed", err.Error())
	}
	if len(report.Findings) > 0 {
		report.HalfDuplexAcceptanceStatus = "blocked"
	}
	return report
}
func validateStackChanHalfDuplexDeviceIdentity(report *stackChanHalfDuplexAcceptanceReport, options stackChanHalfDuplexAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.Microphone = device.Capabilities["microphone"]
	report.Speaker = device.Capabilities["speaker"]
	if device.DeviceID != options.DeviceID {
		report.addFinding("device_id_mismatch", "Gateway device id differs from expected StackChan device")
	}
	if device.IdentityStatus != "ok" {
		report.addFinding("device_identity_not_ok", "Gateway device identity is not ok")
	}
	if device.ConnectionStatus != "" && device.ConnectionStatus != "online" {
		report.addFinding("device_not_online", "Gateway device is not online")
	}
	if device.Firmware.ID != "a21-stackchan" {
		report.addFinding("firmware_id_mismatch", "device firmware id is not a21-stackchan")
	}
	if device.Firmware.Board != "m5stack-cores3" {
		report.addFinding("firmware_board_mismatch", "device firmware board is not m5stack-cores3")
	}
	if device.Firmware.Commit == "" || !sameCLICommit(device.Firmware.Commit, options.Commit) {
		report.addFinding("firmware_commit_mismatch", "device firmware commit differs from expected commit")
	}
	if report.Microphone != firmwarecheck.MicProbeCapabilityStatus {
		report.addFinding("microphone_not_diagnostic_probe", "device microphone capability is not the diagnostic mic-probe status")
	}
	if report.Speaker != "available" {
		report.addFinding("speaker_not_available", "device speaker capability is not available")
	}
}
func validateStackChanHalfDuplexDeltas(report *stackChanHalfDuplexAcceptanceReport, options stackChanHalfDuplexAcceptanceOptions) {
	if report.MicFramesCapturedDelta < options.MinMicFrames {
		report.addFinding("mic_frames_delta_below_threshold", "captured microphone frame delta is below threshold")
	}
	if report.AudioWSSentAudioFramesDelta < options.MinMicFrames {
		report.addFinding("audio_ws_frames_delta_below_threshold", "sent audio WebSocket frame delta is below threshold")
	}
	if report.GatewayAudioIngressFramesDelta < options.MinMicFrames {
		report.addFinding("gateway_audio_ingress_delta_below_threshold", "Gateway audio ingress delta is below threshold")
	}
	if report.GatewayAudioPlaybackChunkDelta < options.MinPlaybackChunks {
		report.addFinding("gateway_playback_chunk_delta_below_threshold", "Gateway playback chunk delta is below threshold")
	}
	if report.PlaybackBufferTotalChunksDelta < options.MinPlaybackChunks {
		report.addFinding("playback_buffer_delta_below_threshold", "playback buffer accepted chunk delta is below threshold")
	}
	if report.SpeakerFramesPlayedDelta < options.MinPlaybackChunks {
		report.addFinding("speaker_frames_delta_below_threshold", "speaker played frame delta is below threshold")
	}
	if report.MicDriverErrorsDelta != 0 {
		report.addFinding("mic_driver_error_delta_present", "microphone driver error counter changed during half-duplex probe")
	}
	if report.MicQueueDroppedFramesDelta != 0 {
		report.addFinding("mic_queue_drop_delta_present", "microphone queue drop counter changed during half-duplex probe")
	}
	if report.PlaybackBufferDroppedChunksDelta != 0 {
		report.addFinding("playback_buffer_drop_delta_present", "playback buffer dropped chunks during half-duplex probe")
	}
	if report.SpeakerDriverErrorsDelta != 0 {
		report.addFinding("speaker_driver_error_delta_present", "speaker driver error counter changed during half-duplex probe")
	}
	if report.MicFramesCapturedDelta > 0 && report.AudioWSDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("audio_ws_delivery_ratio_below_threshold", "audio WebSocket sent frame ratio is below captured microphone frames")
	}
	if report.AudioWSSentAudioFramesDelta > 0 && report.GatewayIngressDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("gateway_ingress_delivery_ratio_below_threshold", "Gateway ingress frame ratio is below audio WebSocket sent frames")
	}
}
func (report *stackChanHalfDuplexAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanHalfDuplexAcceptanceReport(outputDir string, report stackChanHalfDuplexAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-half-duplex-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanHalfDuplexAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanHalfDuplexAcceptance(writer io.Writer, report stackChanHalfDuplexAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
