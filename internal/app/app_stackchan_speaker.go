package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type stackChanSpeakerAcceptanceOptions struct {
	GatewayURL      string
	DeviceID        string
	Commit          string
	WindowMS        int
	MockAudioChunks int
	MinPlayedFrames int
	OutputDir       string
}
type stackChanSpeakerGatewayMetrics struct {
	AudioPlaybackChunkTotal int `json:"gateway_audio_playback_chunk_total"`
}
type stackChanSpeakerAcceptanceThresholds struct {
	MinPlayedFrames  int `json:"min_played_frames"`
	MinGatewayChunks int `json:"min_gateway_chunks"`
}
type stackChanSpeakerAcceptanceReport struct {
	SchemaVersion                    string                               `json:"schema_version"`
	GeneratedAtMS                    int64                                `json:"generated_at_ms"`
	Metadata                         latencyBenchMetadata                 `json:"metadata"`
	DryRun                           bool                                 `json:"dry_run"`
	FlashAllowed                     bool                                 `json:"flash_allowed"`
	DeleteAllowed                    bool                                 `json:"delete_allowed"`
	HardwareAcceptanceScope          string                               `json:"hardware_acceptance_scope"`
	SpeakerAcceptanceStatus          string                               `json:"speaker_acceptance_status"`
	PhysicalSoundObserved            bool                                 `json:"physical_sound_observed"`
	GatewayURL                       string                               `json:"gateway_url"`
	DeviceID                         string                               `json:"device_id"`
	Commit                           string                               `json:"commit"`
	WindowMS                         int                                  `json:"window_ms"`
	MockAudioChunks                  int                                  `json:"mock_audio_chunks"`
	ExpectedAudioDurationMS          int                                  `json:"expected_audio_duration_ms"`
	StreamID                         string                               `json:"stream_id"`
	ControlTraceID                   string                               `json:"control_trace_id,omitempty"`
	ControlSessionID                 string                               `json:"control_session_id,omitempty"`
	WindowStartedAtMS                int64                                `json:"window_started_at_ms,omitempty"`
	WindowEndedAtMS                  int64                                `json:"window_ended_at_ms,omitempty"`
	Firmware                         firmwarecheck.DeviceIdentityFirmware `json:"firmware"`
	Capabilities                     map[string]string                    `json:"capabilities,omitempty"`
	Speaker                          string                               `json:"speaker,omitempty"`
	RuntimeEcho                      map[string]string                    `json:"runtime_echo,omitempty"`
	RuntimeEchoBefore                map[string]string                    `json:"runtime_echo_before,omitempty"`
	PlaybackBufferQueuedChunks       int                                  `json:"playback_buffer_queued_chunks"`
	PlaybackBufferTotalChunks        int                                  `json:"playback_buffer_total_chunks"`
	PlaybackBufferDroppedChunks      int                                  `json:"playback_buffer_dropped_chunks"`
	PlaybackBufferClearCount         int                                  `json:"playback_buffer_clear_count"`
	SpeakerFramesPlayed              int                                  `json:"speaker_frames_played"`
	SpeakerBusyTicks                 int                                  `json:"speaker_busy_ticks"`
	SpeakerDriverErrors              int                                  `json:"speaker_driver_errors"`
	SpeakerLastStreamID              string                               `json:"speaker_last_stream_id"`
	PlaybackBufferTotalChunksDelta   int                                  `json:"playback_buffer_total_chunks_delta"`
	PlaybackBufferDroppedChunksDelta int                                  `json:"playback_buffer_dropped_chunks_delta"`
	PlaybackBufferClearCountDelta    int                                  `json:"playback_buffer_clear_count_delta"`
	SpeakerFramesPlayedDelta         int                                  `json:"speaker_frames_played_delta"`
	SpeakerBusyTicksDelta            int                                  `json:"speaker_busy_ticks_delta"`
	SpeakerDriverErrorsDelta         int                                  `json:"speaker_driver_errors_delta"`
	GatewayAudioPlaybackChunkTotal   int                                  `json:"gateway_audio_playback_chunk_total"`
	GatewayAudioPlaybackChunkDelta   int                                  `json:"gateway_audio_playback_chunk_delta"`
	GatewayMetricsBefore             *stackChanSpeakerGatewayMetrics      `json:"gateway_metrics_before,omitempty"`
	GatewayMetricsAfter              *stackChanSpeakerGatewayMetrics      `json:"gateway_metrics_after,omitempty"`
	Thresholds                       stackChanSpeakerAcceptanceThresholds `json:"thresholds"`
	NextRequiredActions              []string                             `json:"next_required_actions"`
	ReportPath                       string                               `json:"report_path,omitempty"`
	Findings                         []officePreflightFinding             `json:"findings,omitempty"`
}

func runStackChanSpeakerAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanSpeakerAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check speaker --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --commit <git-sha> [--window-ms 1000] [--mock-audio-chunks 50] [--min-played-frames 50] [--output-dir reports]")
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
		case "--mock-audio-chunks":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--mock-audio-chunks")
			if !ok {
				return 2
			}
			if value < 1 || value > stackChanSpeakerProbeMaxChunks {
				fmt.Fprintf(stderr, "--mock-audio-chunks must be between 1 and %d\n", stackChanSpeakerProbeMaxChunks)
				return 2
			}
			options.MockAudioChunks = value
		case "--min-played-frames":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--min-played-frames")
			if !ok {
				return 2
			}
			options.MinPlayedFrames = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-speaker-acceptance option %q\n", args[i])
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
		fmt.Fprintf(stderr, "stackchan speaker acceptance gateway URL invalid: %v\n", err)
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan speaker acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanSpeakerAcceptanceReport(options)
	reportPath, err := writeStackChanSpeakerAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan speaker acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanSpeakerAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan speaker acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan speaker acceptance blocked (instrumented only, no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan speaker acceptance ok (instrumented only, no flash performed)")
	return 0
}
func defaultStackChanSpeakerAcceptanceOptions() stackChanSpeakerAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanSpeakerAcceptanceOptions{
		GatewayURL:      firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:        deviceID,
		Commit:          currentGitCommit(projectRoot),
		WindowMS:        1500,
		MockAudioChunks: 50,
		MinPlayedFrames: 50,
		OutputDir:       "reports",
	}
}
func buildStackChanSpeakerAcceptanceReport(options stackChanSpeakerAcceptanceOptions) stackChanSpeakerAcceptanceReport {
	streamID := "a21-speaker-acceptance-stream"
	report := stackChanSpeakerAcceptanceReport{
		SchemaVersion:           "a21.stackchan_speaker_acceptance.v1",
		GeneratedAtMS:           time.Now().UnixMilli(),
		Metadata:                buildLatencyBenchMetadata(),
		DryRun:                  true,
		FlashAllowed:            false,
		DeleteAllowed:           false,
		HardwareAcceptanceScope: "instrumented_speaker_downlink",
		SpeakerAcceptanceStatus: "confirmed",
		PhysicalSoundObserved:   false,
		GatewayURL:              sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:                options.DeviceID,
		Commit:                  options.Commit,
		WindowMS:                options.WindowMS,
		MockAudioChunks:         options.MockAudioChunks,
		ExpectedAudioDurationMS: options.MockAudioChunks * stackChanSpeakerProbeChunkDurationMS,
		StreamID:                streamID,
		Thresholds: stackChanSpeakerAcceptanceThresholds{
			MinPlayedFrames:  options.MinPlayedFrames,
			MinGatewayChunks: options.MockAudioChunks,
		},
		NextRequiredActions: []string{
			"Keep this report with the Gateway device report before any physical speaker promotion.",
			"Treat this as instrumented downlink and speaker-pump evidence, not human-audible sound acceptance.",
			"Run a later operator-observed speaker acceptance before claiming physical audibility or end-to-end TTS quality.",
		},
	}

	beforeGatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_before_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	beforeDevice, ok := findFirmwareDeviceRecord(beforeGatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_before_missing", "expected StackChan device is missing from Gateway before speaker probe")
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	validateStackChanSpeakerDeviceIdentity(&report, options, beforeDevice)
	report.RuntimeEchoBefore = beforeDevice.RuntimeEcho
	beforePlaybackTotal := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_total_chunks")
	beforePlaybackDropped := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_dropped_chunks")
	beforePlaybackClearCount := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "playback_buffer_clear_count")
	beforeSpeakerFrames := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_frames_played")
	beforeSpeakerBusy := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_busy_ticks")
	beforeSpeakerErrors := stackChanRuntimeEchoInt(&report.Findings, beforeDevice.RuntimeEcho, "speaker_driver_errors")
	if len(report.Findings) > 0 {
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}

	beforeMetrics, err := fetchStackChanSpeakerGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_before_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		return report
	}
	report.GatewayMetricsBefore = &beforeMetrics

	traceID := fmt.Sprintf("a21-trace-speaker-acceptance-%d", report.GeneratedAtMS)
	sessionID := fmt.Sprintf("a21-session-speaker-acceptance-%d", report.GeneratedAtMS)
	control, err := postStackChanSpeakerProbeBatches(options.GatewayURL, options.DeviceID, traceID, sessionID, streamID, options.MockAudioChunks)
	if err != nil {
		report.addFinding("speaker_control_failed", err.Error())
		report.SpeakerAcceptanceStatus = "blocked"
		_, _ = postStackChanSpeakerControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", traceID, sessionID, streamID, 0)
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
			report.addFinding("gateway_device_after_missing", "expected StackChan device is missing from Gateway after speaker probe")
		} else {
			validateStackChanSpeakerDevice(&report, options, afterDevice)
			report.PlaybackBufferTotalChunksDelta = report.PlaybackBufferTotalChunks - beforePlaybackTotal
			report.PlaybackBufferDroppedChunksDelta = report.PlaybackBufferDroppedChunks - beforePlaybackDropped
			report.PlaybackBufferClearCountDelta = report.PlaybackBufferClearCount - beforePlaybackClearCount
			report.SpeakerFramesPlayedDelta = report.SpeakerFramesPlayed - beforeSpeakerFrames
			report.SpeakerBusyTicksDelta = report.SpeakerBusyTicks - beforeSpeakerBusy
			report.SpeakerDriverErrorsDelta = report.SpeakerDriverErrors - beforeSpeakerErrors
			validateStackChanSpeakerRuntimeDeltas(&report, options)
		}
	}

	afterMetrics, err := fetchStackChanSpeakerGatewayMetrics(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_metrics_after_failed", err.Error())
	} else {
		report.GatewayMetricsAfter = &afterMetrics
		report.GatewayAudioPlaybackChunkTotal = afterMetrics.AudioPlaybackChunkTotal
		report.GatewayAudioPlaybackChunkDelta = afterMetrics.AudioPlaybackChunkTotal - beforeMetrics.AudioPlaybackChunkTotal
		validateStackChanSpeakerMetricDeltas(&report, options)
	}

	if _, err := postStackChanSpeakerControl(options.GatewayURL, options.DeviceID, protocol.ExpressionIdle, protocol.ModeWorkmate, "IDLE", report.ControlTraceID, report.ControlSessionID, streamID, 0); err != nil {
		report.addFinding("speaker_idle_control_failed", err.Error())
	}
	if len(report.Findings) > 0 {
		report.SpeakerAcceptanceStatus = "blocked"
	}
	return report
}
func validateStackChanSpeakerDevice(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	validateStackChanSpeakerDeviceIdentity(report, options, device)
	report.RuntimeEcho = device.RuntimeEcho
	report.PlaybackBufferQueuedChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_queued_chunks")
	report.PlaybackBufferTotalChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_total_chunks")
	report.PlaybackBufferDroppedChunks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_dropped_chunks")
	report.PlaybackBufferClearCount = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "playback_buffer_clear_count")
	report.SpeakerFramesPlayed = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_frames_played")
	report.SpeakerBusyTicks = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_busy_ticks")
	report.SpeakerDriverErrors = stackChanRuntimeEchoInt(&report.Findings, device.RuntimeEcho, "speaker_driver_errors")
	report.SpeakerLastStreamID = stackChanRuntimeEchoString(&report.Findings, device.RuntimeEcho, "speaker_last_stream_id")
	if report.SpeakerLastStreamID != report.StreamID {
		report.addFinding("speaker_stream_id_mismatch", "speaker runtime echo does not match the commanded stream id")
	}
}
func validateStackChanSpeakerDeviceIdentity(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions, device firmwarecheck.DeviceIdentityRecord) {
	report.Firmware = device.Firmware
	report.Capabilities = device.Capabilities
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
	if report.Speaker != "available" {
		report.addFinding("speaker_not_available", "device speaker capability is not available")
	}
}
func validateStackChanSpeakerRuntimeDeltas(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions) {
	if report.PlaybackBufferTotalChunksDelta < options.MockAudioChunks {
		report.addFinding("playback_buffer_delta_below_threshold", "playback buffer accepted chunk delta is below commanded mock audio chunks")
	}
	if report.PlaybackBufferDroppedChunksDelta != 0 {
		report.addFinding("playback_buffer_drop_delta_present", "playback buffer dropped chunks during speaker probe")
	}
	if report.SpeakerFramesPlayedDelta < options.MinPlayedFrames {
		report.addFinding("speaker_frames_delta_below_threshold", "speaker played frame delta is below threshold")
	}
	if report.SpeakerDriverErrorsDelta != 0 {
		report.addFinding("speaker_driver_error_delta_present", "speaker driver error counter changed during speaker probe")
	}
}
func validateStackChanSpeakerMetricDeltas(report *stackChanSpeakerAcceptanceReport, options stackChanSpeakerAcceptanceOptions) {
	if report.GatewayAudioPlaybackChunkDelta < options.MockAudioChunks {
		report.addFinding("gateway_playback_chunk_delta_below_threshold", "Gateway playback chunk delta is below commanded mock audio chunks")
	}
}
func fetchStackChanSpeakerGatewayMetrics(gatewayBaseURL string) (stackChanSpeakerGatewayMetrics, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/metrics", nil)
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stackChanSpeakerGatewayMetrics{}, fmt.Errorf("gateway metrics returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return stackChanSpeakerGatewayMetrics{}, err
	}
	return parseStackChanSpeakerGatewayMetrics(string(data))
}
func parseStackChanSpeakerGatewayMetrics(data string) (stackChanSpeakerGatewayMetrics, error) {
	var metrics stackChanSpeakerGatewayMetrics
	seen := false
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		metricName := fields[0]
		if labelsAt := strings.Index(metricName, "{"); labelsAt >= 0 {
			metricName = metricName[:labelsAt]
		}
		if metricName != "a21_audio_playback_chunk_total" {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		metrics.AudioPlaybackChunkTotal = int(value)
		seen = true
	}
	if !seen {
		return stackChanSpeakerGatewayMetrics{}, fmt.Errorf("gateway metrics missing a21_audio_playback_chunk_total")
	}
	return metrics, nil
}
func (report *stackChanSpeakerAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanSpeakerAcceptanceReport(outputDir string, report stackChanSpeakerAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-speaker-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanSpeakerAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanSpeakerAcceptance(writer io.Writer, report stackChanSpeakerAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
