package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const xiaozhiHalfDuplexAcceptanceSchemaVersion = "a21.xiaozhi_half_duplex_acceptance.v1"

type xiaozhiHalfDuplexAcceptanceReport struct {
	SchemaVersion                 string                               `json:"schema_version"`
	GeneratedAtMS                 int64                                `json:"generated_at_ms"`
	Metadata                      latencyBenchMetadata                 `json:"metadata"`
	Source                        string                               `json:"source"`
	HardwareAcceptanceScope       string                               `json:"hardware_acceptance_scope"`
	HalfDuplexAcceptanceStatus    string                               `json:"half_duplex_acceptance_status"`
	DiagnosticMicProbeRequired    bool                                 `json:"diagnostic_mic_probe_required"`
	PhysicalSoundObserved         bool                                 `json:"physical_sound_observed"`
	Gateway                       string                               `json:"gateway,omitempty"`
	DeviceID                      string                               `json:"device_id"`
	TraceID                       string                               `json:"trace_id"`
	SessionID                     string                               `json:"session_id"`
	Profile                       string                               `json:"profile"`
	PhysicalDeviceOnline          bool                                 `json:"physical_device_online"`
	AudioFrameCount               int                                  `json:"audio_frame_count"`
	MicAvailable                  bool                                 `json:"mic_available"`
	DownlinkAvailable             bool                                 `json:"downlink_available"`
	PlaybackAckAvailable          bool                                 `json:"playback_ack_available"`
	BargeInDetectedAvailable      bool                                 `json:"barge_in_detected_available"`
	BargeInStopAvailable          bool                                 `json:"barge_in_stop_available"`
	BargeInStopDoneAvailable      bool                                 `json:"barge_in_stop_done_available"`
	StageAvailability             map[string]physicalStackChanMetric   `json:"stage_availability"`
	CanonicalMetrics              physicalStackChanCanonicalMetrics    `json:"canonical_metrics"`
	Mic                           physicalStackChanMicEvidence         `json:"mic"`
	Observation                   physicalStackChanObservationEvidence `json:"observation"`
	Redaction                     physicalStackChanEvidenceRedaction   `json:"redaction"`
	PRDAccepted                   bool                                 `json:"prd_accepted"`
	NextRequiredActions           []string                             `json:"next_required_actions"`
	SourcePhysicalEvidenceSummary xiaozhiPhysicalEvidenceSummary       `json:"source_physical_evidence_summary"`
	Findings                      []physicalStackChanEvidenceFinding   `json:"findings,omitempty"`
	ReportPath                    string                               `json:"report_path,omitempty"`
}

type xiaozhiPhysicalEvidenceSummary struct {
	SchemaVersion    string `json:"schema_version"`
	AcceptanceStatus string `json:"acceptance_status"`
	PromotionGate    string `json:"promotion_gate"`
	PRDAccepted      bool   `json:"prd_accepted"`
}

func runXiaozhiHalfDuplexAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiPhysicalEvidenceOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		OutputDir:  "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-accept --check xiaozhi-half-duplex --gateway-url http://127.0.0.1:21080 --device-id <device-id> --trace-id <trace-id> --session-id <session-id> [--instrument-observation-report report.json] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if !readStringOption(args, &i, stderr, "--gateway-url", &options.GatewayURL) {
				return 2
			}
		case "--device-id":
			if !readStringOption(args, &i, stderr, "--device-id", &options.DeviceID) {
				return 2
			}
		case "--trace-id":
			if !readStringOption(args, &i, stderr, "--trace-id", &options.TraceID) {
				return 2
			}
		case "--session-id":
			if !readStringOption(args, &i, stderr, "--session-id", &options.SessionID) {
				return 2
			}
		case "--instrument-observation-report":
			if !readStringOption(args, &i, stderr, "--instrument-observation-report", &options.InstrumentObservationReport) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-half-duplex option %q\n", args[i])
			return 2
		}
	}
	if err := validateXiaozhiPhysicalEvidenceOptions(options); err != nil {
		if strings.TrimSpace(options.TraceID) == "" || strings.TrimSpace(options.SessionID) == "" {
			if err := populateXiaozhiHalfDuplexLatestDeviceTrace(&options); err != nil {
				fmt.Fprintf(stderr, "xiaozhi half-duplex latest trace unavailable: %v\n", err)
				return 1
			}
		}
	}
	if err := validateXiaozhiPhysicalEvidenceOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi half-duplex option invalid: %v\n", err)
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "xiaozhi half-duplex report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildXiaozhiHalfDuplexAcceptanceReport(options)
	if err != nil {
		if err.Error() == "gateway data unsafe" {
			fmt.Fprintln(stderr, "xiaozhi half-duplex gateway data unsafe")
			return 1
		}
		if err.Error() == "instrument observation unsafe" {
			fmt.Fprintln(stderr, "xiaozhi half-duplex instrument observation unsafe")
			return 1
		}
		fmt.Fprintf(stderr, "xiaozhi half-duplex failed: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiHalfDuplexAcceptanceReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintln(stderr, "xiaozhi half-duplex report write failed")
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiHalfDuplexAcceptance(stdout, report); err != nil {
		fmt.Fprintln(stderr, "xiaozhi half-duplex report encode failed")
		return 1
	}
	if report.HalfDuplexAcceptanceStatus == "blocked" {
		fmt.Fprintln(stdout, "stackchan stock Xiaozhi half-duplex acceptance blocked")
		return 1
	}
	if report.HalfDuplexAcceptanceStatus == "physical_review_required" {
		fmt.Fprintln(stdout, "stackchan stock Xiaozhi half-duplex acceptance needs physical review")
		return 0
	}
	fmt.Fprintln(stdout, "stackchan stock Xiaozhi half-duplex trace candidate recorded")
	return 0
}

func populateXiaozhiHalfDuplexLatestDeviceTrace(options *xiaozhiPhysicalEvidenceOptions) error {
	if options == nil || !xiaozhiPhysicalSafeID(options.DeviceID) {
		return fmt.Errorf("device_id is invalid or unsafe")
	}
	deviceReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		return err
	}
	device, ok := findFirmwareDeviceRecord(deviceReport.Devices, options.DeviceID)
	if !ok {
		return fmt.Errorf("device not found in gateway registry")
	}
	if strings.TrimSpace(options.TraceID) == "" {
		options.TraceID = strings.TrimSpace(device.LastTraceID)
	}
	if strings.TrimSpace(options.SessionID) == "" {
		options.SessionID = strings.TrimSpace(device.LastSessionID)
	}
	if strings.TrimSpace(options.TraceID) == "" || strings.TrimSpace(options.SessionID) == "" {
		return fmt.Errorf("latest trace/session missing from gateway registry")
	}
	return nil
}

func buildXiaozhiHalfDuplexAcceptanceReport(options xiaozhiPhysicalEvidenceOptions) (xiaozhiHalfDuplexAcceptanceReport, error) {
	physical, err := buildXiaozhiPhysicalEvidenceReport(options)
	if err != nil {
		return xiaozhiHalfDuplexAcceptanceReport{}, err
	}
	report := xiaozhiHalfDuplexAcceptanceReport{
		SchemaVersion:              xiaozhiHalfDuplexAcceptanceSchemaVersion,
		GeneratedAtMS:              time.Now().UnixMilli(),
		Metadata:                   buildLatencyBenchMetadata(),
		Source:                     "xiaozhi_physical_evidence",
		HardwareAcceptanceScope:    "stock_xiaozhi_mic_to_tts_downlink",
		HalfDuplexAcceptanceStatus: "blocked",
		DiagnosticMicProbeRequired: false,
		PhysicalSoundObserved:      physical.Observation.PhysicalSoundObserved,
		Gateway:                    physical.Gateway,
		DeviceID:                   physical.DeviceID,
		TraceID:                    physical.TraceID,
		SessionID:                  physical.SessionID,
		Profile:                    physical.Profile,
		PhysicalDeviceOnline:       physical.PhysicalDeviceOnline,
		AudioFrameCount:            physical.AudioFrameCount,
		MicAvailable:               physical.Mic.Available,
		DownlinkAvailable:          physical.StageAvailability["xiaozhi.tts.downlink"].Available || physical.CanonicalMetrics.DeviceDownlinkFirstFrameMS.Available || physical.GatewayMetrics.GatewayAnswerFirstDownlinkMS.Available,
		PlaybackAckAvailable:       physical.StageAvailability["device.playback.ack"].Available || physical.CanonicalMetrics.DevicePlaybackStartMS.Available,
		BargeInDetectedAvailable:   physical.CanonicalMetrics.BargeInDetectedMS.Available,
		BargeInStopAvailable:       physical.CanonicalMetrics.BargeInStopMS.Available,
		BargeInStopDoneAvailable:   physical.CanonicalMetrics.BargeInPlaybackStopDoneMS.Available,
		StageAvailability:          xiaozhiHalfDuplexStageAvailability(physical),
		CanonicalMetrics:           physical.CanonicalMetrics,
		Mic:                        physical.Mic,
		Observation:                physical.Observation,
		Redaction:                  physical.Redaction,
		PRDAccepted:                false,
		SourcePhysicalEvidenceSummary: xiaozhiPhysicalEvidenceSummary{
			SchemaVersion:    physical.SchemaVersion,
			AcceptanceStatus: physical.AcceptanceStatus,
			PromotionGate:    physical.PromotionGate,
			PRDAccepted:      physical.PRDAccepted,
		},
		NextRequiredActions: []string{
			"Keep this as stock Xiaozhi physical trace evidence; do not merge it with diagnostic mic-probe acceptance.",
			"Collect operator audible confirmation or instrument observation before treating it as product half-duplex review evidence.",
			"Run a separate wake-word product proof before claiming full PRD green.",
		},
	}
	report.Findings = xiaozhiHalfDuplexFindings(report)
	if len(report.Findings) == 0 {
		report.HalfDuplexAcceptanceStatus = "physical_review_required"
	} else if xiaozhiHalfDuplexMachineTraceCandidate(report) {
		report.HalfDuplexAcceptanceStatus = "candidate_gateway_trace"
	}
	return report, nil
}

func xiaozhiHalfDuplexStageAvailability(physical xiaozhiPhysicalEvidenceReport) map[string]physicalStackChanMetric {
	return map[string]physicalStackChanMetric{
		"physical_xiaozhi.online":      physical.StageAvailability["physical_xiaozhi.online"],
		"xiaozhi.profile.stock":        physical.StageAvailability["xiaozhi.profile.stock"],
		"xiaozhi.profile.debug":        physical.StageAvailability["xiaozhi.profile.debug"],
		"xiaozhi.opus.decode":          physical.StageAvailability["xiaozhi.opus.decode"],
		"audio.ingress.pcm":            physical.StageAvailability["audio.ingress.pcm"],
		"vad.speech.end":               physical.StageAvailability["vad.speech.end"],
		"xiaozhi.listen.auto_stop":     physical.StageAvailability["xiaozhi.listen.auto_stop"],
		"xiaozhi.tts.downlink":         physical.StageAvailability["xiaozhi.tts.downlink"],
		"device.playback.ack":          physical.StageAvailability["device.playback.ack"],
		"barge_in.detected":            physical.CanonicalMetrics.BargeInDetectedMS,
		"barge_in.stop":                physical.CanonicalMetrics.BargeInStopMS,
		"device.playback.stop_done":    physical.CanonicalMetrics.BargeInPlaybackStopDoneMS,
		"operator.audible_observation": physical.StageAvailability["operator.audible_observation"],
	}
}

func xiaozhiHalfDuplexFindings(report xiaozhiHalfDuplexAcceptanceReport) []physicalStackChanEvidenceFinding {
	var findings []physicalStackChanEvidenceFinding
	for _, stage := range []struct {
		available bool
		code      string
		message   string
	}{
		{report.PhysicalDeviceOnline, "xiaozhi_half_duplex_device_offline", "physical Xiaozhi device is not online in the A21 Gateway registry"},
		{report.StageAvailability["xiaozhi.profile.stock"].Available || report.StageAvailability["xiaozhi.profile.debug"].Available, "xiaozhi_half_duplex_profile_missing", "stock or isolated debug Xiaozhi profile evidence is missing"},
		{report.MicAvailable, "xiaozhi_half_duplex_mic_ingress_missing", "stock Xiaozhi microphone Opus/PCM ingress evidence is missing"},
		{report.DownlinkAvailable, "xiaozhi_half_duplex_downlink_missing", "stock Xiaozhi TTS downlink evidence is missing"},
		{report.PlaybackAckAvailable, "xiaozhi_half_duplex_playback_ack_missing", "device playback start or trusted runtime playback observation is missing"},
		{report.BargeInDetectedAvailable, "xiaozhi_half_duplex_barge_in_detected_missing", "barge-in detection evidence is missing"},
		{report.BargeInStopAvailable, "xiaozhi_half_duplex_barge_in_stop_missing", "barge-in stop evidence is missing"},
		{report.Observation.Available && report.Observation.PhysicalSoundObserved, "xiaozhi_half_duplex_audible_observation_missing", "operator audible confirmation or instrumented audible observation is missing"},
	} {
		if !stage.available {
			findings = append(findings, physicalStackChanFinding(stage.code, "error", stage.message))
		}
	}
	if report.BargeInStopAvailable && !report.BargeInStopDoneAvailable {
		findings = append(findings, physicalStackChanFinding("xiaozhi_half_duplex_stop_done_not_available", "info", "stock firmware may not expose device playback stop_done; Gateway barge-in stop evidence is present"))
	}
	return findings
}

func xiaozhiHalfDuplexMachineTraceCandidate(report xiaozhiHalfDuplexAcceptanceReport) bool {
	return report.PhysicalDeviceOnline &&
		report.MicAvailable &&
		report.DownlinkAvailable &&
		report.BargeInDetectedAvailable &&
		report.BargeInStopAvailable
}

func writeXiaozhiHalfDuplexAcceptanceReport(outputDir string, report xiaozhiHalfDuplexAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-half-duplex-acceptance-"+time.Now().Format("20060102-150405.000000000")+".json")
	report.ReportPath = filepath.Base(reportPath)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONXiaozhiHalfDuplexAcceptance(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiHalfDuplexAcceptance(writer io.Writer, report xiaozhiHalfDuplexAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
