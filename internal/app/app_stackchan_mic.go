package app

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/protocol"
)

type stackChanMicProbeAcceptanceOptions struct {
	GatewayURL          string
	DeviceID            string
	Commit              string
	WindowMS            int
	MinFrames           int
	MinAbsPeak          int
	MinNonzeroSamples   int
	MinGatewayRMS       float64
	MinGatewayVADSpeech int
	MinDeliveryRatio    float64
	OutputDir           string
}
type stackChanMicProbeGatewayMetrics struct {
	AudioFrameTotal         int     `json:"gateway_audio_frame_total"`
	AudioIngressFramesTotal int     `json:"gateway_audio_ingress_frames_total"`
	AudioIngressRMS         float64 `json:"gateway_audio_ingress_rms"`
	AudioPlaybackChunkTotal int     `json:"gateway_audio_playback_chunk_total"`
	VADSpeechTotal          int     `json:"gateway_vad_speech_total"`
}
type stackChanMicProbeAcceptanceReport struct {
	SchemaVersion                  string                                `json:"schema_version"`
	GeneratedAtMS                  int64                                 `json:"generated_at_ms"`
	Metadata                       latencyBenchMetadata                  `json:"metadata"`
	DryRun                         bool                                  `json:"dry_run"`
	FlashAllowed                   bool                                  `json:"flash_allowed"`
	DeleteAllowed                  bool                                  `json:"delete_allowed"`
	HardwareAcceptanceScope        string                                `json:"hardware_acceptance_scope"`
	MicProbeAcceptanceStatus       string                                `json:"mic_probe_acceptance_status"`
	ProductionCapabilityPromoted   bool                                  `json:"production_capability_promoted"`
	GatewayURL                     string                                `json:"gateway_url"`
	DeviceID                       string                                `json:"device_id"`
	Commit                         string                                `json:"commit"`
	WindowMS                       int                                   `json:"window_ms,omitempty"`
	ControlTraceID                 string                                `json:"control_trace_id,omitempty"`
	ControlSessionID               string                                `json:"control_session_id,omitempty"`
	WindowStartedAtMS              int64                                 `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                int64                                 `json:"window_ended_at_ms,omitempty"`
	Firmware                       firmwarecheck.DeviceIdentityFirmware  `json:"firmware"`
	Capabilities                   map[string]string                     `json:"capabilities,omitempty"`
	Microphone                     string                                `json:"microphone,omitempty"`
	RuntimeEcho                    map[string]string                     `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore              map[string]string                     `json:"runtime_echo_before,omitempty"`
	MicFramesCaptured              int                                   `json:"mic_frames_captured"`
	AudioWSSentAudioFrames         int                                   `json:"audio_ws_sent_audio_frames"`
	MicDriverErrors                int                                   `json:"mic_driver_errors"`
	MicQueueDepth                  int                                   `json:"mic_queue_depth"`
	MicQueueDroppedFrames          int                                   `json:"mic_queue_dropped_frames"`
	MicLastAbsPeak                 int                                   `json:"mic_last_abs_peak"`
	MicLastNonzeroSamples          int                                   `json:"mic_last_nonzero_samples"`
	MicCaptureRateHz               float64                               `json:"mic_capture_rate_hz,omitempty"`
	AudioWSSentRateHz              float64                               `json:"audio_ws_sent_rate_hz,omitempty"`
	GatewayAudioIngressRateHz      float64                               `json:"gateway_audio_ingress_rate_hz,omitempty"`
	AudioWSDeliveryRatio           float64                               `json:"audio_ws_delivery_ratio,omitempty"`
	GatewayIngressDeliveryRatio    float64                               `json:"gateway_ingress_delivery_ratio,omitempty"`
	GatewayAudioFrameTotal         int                                   `json:"gateway_audio_frame_total"`
	GatewayAudioIngressFramesTotal int                                   `json:"gateway_audio_ingress_frames_total"`
	GatewayAudioIngressRMS         float64                               `json:"gateway_audio_ingress_rms"`
	GatewayAudioPlaybackChunkTotal int                                   `json:"gateway_audio_playback_chunk_total"`
	GatewayVADSpeechTotal          int                                   `json:"gateway_vad_speech_total"`
	MicFramesCapturedDelta         int                                   `json:"mic_frames_captured_delta"`
	AudioWSSentAudioFramesDelta    int                                   `json:"audio_ws_sent_audio_frames_delta"`
	MicDriverErrorsDelta           int                                   `json:"mic_driver_errors_delta"`
	MicQueueDroppedFramesDelta     int                                   `json:"mic_queue_dropped_frames_delta"`
	GatewayAudioFrameDelta         int                                   `json:"gateway_audio_frame_delta"`
	GatewayAudioIngressFramesDelta int                                   `json:"gateway_audio_ingress_frames_delta"`
	GatewayAudioPlaybackChunkDelta int                                   `json:"gateway_audio_playback_chunk_delta"`
	GatewayVADSpeechDelta          int                                   `json:"gateway_vad_speech_delta"`
	GatewayMetricsBefore           *stackChanMicProbeGatewayMetrics      `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter            *stackChanMicProbeGatewayMetrics      `json:"gateway_metrics_after,omitempty"`
	Thresholds                     stackChanMicProbeAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions            []string                              `json:"next_required_actions"`
	ReportPath                     string                                `json:"report_path,omitempty"`
	Findings                       []officePreflightFinding              `json:"findings,omitempty"`
}
type stackChanMicProbeAcceptanceThresholds struct {
	MinFrames           int     `json:"min_frames"`
	MinAbsPeak          int     `json:"min_abs_peak"`
	MinNonzeroSamples   int     `json:"min_nonzero_samples"`
	MinGatewayRMS       float64 `json:"min_gateway_rms"`
	MinGatewayVADSpeech int     `json:"min_gateway_vad_speech"`
	MinDeliveryRatio    float64 `json:"min_delivery_ratio"`
}

func runStackChanMicProbeAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanMicProbeAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check mic-probe --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 0] [--min-frames 90] [--min-abs-peak 1] [--min-nonzero-samples 1] [--min-gateway-rms 0] [--min-vad-speech 0] [--min-delivery-ratio 0.95] [--output-dir reports]")
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
		case "--min-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-frames")
			if !ok {
				return 2
			}
			options.MinFrames = value
		case "--min-abs-peak":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-abs-peak")
			if !ok {
				return 2
			}
			options.MinAbsPeak = value
		case "--min-nonzero-samples":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-nonzero-samples")
			if !ok {
				return 2
			}
			options.MinNonzeroSamples = value
		case "--min-gateway-rms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--min-gateway-rms requires a value")
				return 2
			}
			i++
			value, err := strconv.ParseFloat(args[i], 64)
			if err != nil || value < 0 {
				fmt.Fprintln(stderr, "--min-gateway-rms must be a non-negative number")
				return 2
			}
			options.MinGatewayRMS = value
		case "--min-delivery-ratio":
			value, ok := parseRatioCLIOption(args, &i, stderr, "--min-delivery-ratio")
			if !ok {
				return 2
			}
			options.MinDeliveryRatio = value
		case "--min-vad-speech":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-vad-speech")
			if !ok {
				return 2
			}
			options.MinGatewayVADSpeech = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-mic-probe-acceptance option %q\n", args[i])
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
		fmt.Fprintf(stderr, "stackchan mic probe acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan mic probe acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanMicProbeAcceptanceReport(options)
	reportPath, err := writeStackChanMicProbeAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan mic probe acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanMicProbeAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan mic probe acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan mic probe acceptance blocked (diagnostic only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan mic probe acceptance ok (diagnostic only, no flash performed)")
	return 0
}
func defaultStackChanMicProbeAcceptanceOptions() stackChanMicProbeAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanMicProbeAcceptanceOptions{
		GatewayURL:          firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:            deviceID,
		Commit:              currentGitCommit(projectRoot),
		MinFrames:           90,
		MinAbsPeak:          1,
		MinNonzeroSamples:   1,
		MinGatewayRMS:       0,
		MinGatewayVADSpeech: 0,
		MinDeliveryRatio:    0.95,
		OutputDir:           "reports",
	}
}
func buildStackChanMicProbeAcceptanceReport(options stackChanMicProbeAcceptanceOptions) stackChanMicProbeAcceptanceReport {
	report := stackChanMicProbeAcceptanceReport{
		SchemaVersion:                "a21.stackchan_mic_probe_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "diagnostic_microphone_only",
		MicProbeAcceptanceStatus:     "confirmed",
		ProductionCapabilityPromoted: false,
		GatewayURL:                   sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		WindowMS:                     options.WindowMS,
		Thresholds: stackChanMicProbeAcceptanceThresholds{
			MinFrames:           options.MinFrames,
			MinAbsPeak:          options.MinAbsPeak,
			MinNonzeroSamples:   options.MinNonzeroSamples,
			MinGatewayRMS:       options.MinGatewayRMS,
			MinGatewayVADSpeech: options.MinGatewayVADSpeech,
			MinDeliveryRatio:    options.MinDeliveryRatio,
		},
		NextRequiredActions: []string{
			"Keep this report with the mic-probe flash receipt and Gateway device report.",
			"Do not treat this as production microphone, speaker, AEC, full-duplex, provider, or latency acceptance.",
			"Promote microphone capability only through a later ADR and release-firmware acceptance gate.",
		},
	}

	if options.WindowMS > 0 {
		runStackChanMicProbeAcceptanceWindow(&report, options)
		if len(report.Findings) > 0 {
			report.MicProbeAcceptanceStatus = "blocked"
		}
		return report
	}

	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
	} else {
		report.GatewayURL = gatewayReport.GatewayURL
		device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("gateway_device_missing", "expected StackChan device is missing from Gateway")
		} else {
			validateStackChanMicProbeDevice(&report, options, device)
		}
	}

	metrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_failed", err.Error())
	} else {
		report.GatewayAudioFrameTotal = metrics.AudioFrameTotal
		report.GatewayAudioIngressFramesTotal = metrics.AudioIngressFramesTotal
		report.GatewayAudioIngressRMS = metrics.AudioIngressRMS
		report.GatewayAudioPlaybackChunkTotal = metrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechTotal = metrics.VADSpeechTotal
		validateStackChanMicProbeMetrics(&report, options)
	}

	if len(report.Findings) > 0 {
		report.MicProbeAcceptanceStatus = "blocked"
	}
	return report
}
func runStackChanMicProbeAcceptanceWindow(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		return
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before probe window")
		return
	}
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforeMicFramesCaptured := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_frames_captured")
	beforeAudioWSSentAudioFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "audio_ws_sent_audio_frames")
	beforeMicDriverErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_driver_errors")
	beforeMicQueueDroppedFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "mic_queue_dropped_frames")

	beforeMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		return
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-mic-probe-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-mic-probe-%d", report.GeneratedAtMS)
	control, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionListening, protocol.ModeWorkmate, "MIC PROBE", traceID, sessionID, true, false)
	if err != nil {
		report.addFinding("mic_probe_control_failed", err.Error())
		return
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
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after probe window")
		} else {
			validateStackChanMicProbeDevice(report, options, afterDevice)
			report.MicFramesCapturedDelta = report.MicFramesCaptured - beforeMicFramesCaptured
			report.AudioWSSentAudioFramesDelta = report.AudioWSSentAudioFrames - beforeAudioWSSentAudioFrames
			report.MicDriverErrorsDelta = report.MicDriverErrors - beforeMicDriverErrors
			report.MicQueueDroppedFramesDelta = report.MicQueueDroppedFrames - beforeMicQueueDroppedFrames
			validateStackChanMicProbeRuntimeDeltas(report, options)
		}
	}

	afterMetrics, err := fetchStackChanMicProbeGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioFrameTotal = afterMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesTotal = afterMetrics.AudioIngressFramesTotal
		report.GatewayAudioIngressRMS = afterMetrics.AudioIngressRMS
		report.GatewayAudioPlaybackChunkTotal = afterMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechTotal = afterMetrics.VADSpeechTotal
		report.GatewayAudioFrameDelta = afterMetrics.AudioFrameTotal - beforeMetrics.AudioFrameTotal
		report.GatewayAudioIngressFramesDelta = afterMetrics.AudioIngressFramesTotal - beforeMetrics.AudioIngressFramesTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		report.GatewayVADSpeechDelta = afterMetrics.VADSpeechTotal - beforeMetrics.VADSpeechTotal
		validateStackChanMicProbeMetricDeltas(report, options)
	}
	populateStackChanMicProbeWindowQuality(report)
	validateStackChanMicProbeDeliveryRatios(report, options)

	if _, err := postStackChanMicProbeControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, false, false); err != nil {
		report.addFinding("mic_probe_idle_control_failed", err.Error())
	}
}
func validateStackChanMicProbeDevice(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
	report.RuntimeEcho = device.RuntimeEcho
	report.Microphone = device.Capabilities["microphone"]
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
	report.MicFramesCaptured = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_frames_captured")
	report.AudioWSSentAudioFrames = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "audio_ws_sent_audio_frames")
	report.MicDriverErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_driver_errors")
	report.MicQueueDepth = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_queue_depth")
	report.MicQueueDroppedFrames = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_queue_dropped_frames")
	report.MicLastAbsPeak = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_last_abs_peak")
	report.MicLastNonzeroSamples = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "mic_last_nonzero_samples")

	if report.MicFramesCaptured < options.MinFrames {
		report.addFinding("mic_frames_below_threshold", "captured microphone frame count is below threshold")
	}
	if report.AudioWSSentAudioFrames < options.MinFrames {
		report.addFinding("audio_ws_frames_below_threshold", "sent audio WebSocket frame count is below threshold")
	}
	if report.MicDriverErrors != 0 {
		report.addFinding("mic_driver_errors_present", "microphone driver reported errors")
	}
	if report.MicQueueDroppedFrames != 0 {
		report.addFinding("mic_queue_drops_present", "microphone queue dropped frames")
	}
	if report.MicLastAbsPeak < options.MinAbsPeak {
		report.addFinding("mic_peak_below_threshold", "latest microphone peak is below threshold")
	}
	if report.MicLastNonzeroSamples < options.MinNonzeroSamples {
		report.addFinding("mic_nonzero_samples_below_threshold", "latest microphone frame has too few non-zero samples")
	}
}
func (report *stackChanMicProbeAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
