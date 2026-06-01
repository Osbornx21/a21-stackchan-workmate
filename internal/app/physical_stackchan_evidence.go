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

const physicalStackChanEvidenceSchemaVersion = "a21.physical_stackchan_evidence.v1"

type physicalStackChanEvidenceReport struct {
	SchemaVersion     string                               `json:"schema_version"`
	ExecutionMode     string                               `json:"execution_mode"`
	TraceID           string                               `json:"trace_id,omitempty"`
	SessionID         string                               `json:"session_id,omitempty"`
	DeviceID          string                               `json:"device_id,omitempty"`
	FixturePath       string                               `json:"fixture_path,omitempty"`
	Execution         physicalStackChanEvidenceExecution   `json:"execution"`
	StageAvailability map[string]physicalStackChanMetric   `json:"stage_availability"`
	CanonicalMetrics  physicalStackChanCanonicalMetrics    `json:"canonical_metrics"`
	Mic               physicalStackChanMicEvidence         `json:"mic"`
	Observation       physicalStackChanObservationEvidence `json:"observation"`
	Findings          []physicalStackChanEvidenceFinding   `json:"findings"`
	Redaction         physicalStackChanEvidenceRedaction   `json:"redaction"`
	PromotionGate     string                               `json:"promotion_gate"`
	AcceptanceStatus  string                               `json:"acceptance_status"`
	PRDAccepted       bool                                 `json:"prd_accepted"`
	ReportPath        string                               `json:"report_path,omitempty"`
}

type physicalStackChanEvidenceExecution struct {
	ProviderExecuted bool `json:"provider_executed"`
	V21Executed      bool `json:"v21_executed"`
	HardwareExecuted bool `json:"hardware_executed"`
}

type physicalStackChanCanonicalMetrics struct {
	DeviceDownlinkFirstFrameMS        physicalStackChanMetric `json:"device_downlink_first_frame_ms"`
	DevicePlaybackStartMS             physicalStackChanMetric `json:"device_playback_start_ms"`
	SpeechEndToFirstAudibleResponseMS physicalStackChanMetric `json:"speech_end_to_first_audible_response_ms"`
	BargeInDetectedMS                 physicalStackChanMetric `json:"barge_in_detected_ms"`
	BargeInStopMS                     physicalStackChanMetric `json:"barge_in_stop_ms"`
	BargeInPlaybackStopRequestedMS    physicalStackChanMetric `json:"barge_in_playback_stop_requested_ms"`
	BargeInPlaybackStopDoneMS         physicalStackChanMetric `json:"barge_in_playback_stop_done_ms"`
}

type physicalStackChanMetric struct {
	Available bool    `json:"available"`
	ValueMS   float64 `json:"value_ms,omitempty"`
	Source    string  `json:"source,omitempty"`
}

type physicalStackChanMicEvidence struct {
	Available          bool    `json:"available"`
	FramesCaptured     int     `json:"frames_captured,omitempty"`
	FramesDelivered    int     `json:"frames_delivered,omitempty"`
	RMS                float64 `json:"rms,omitempty"`
	DeliveryRatio      float64 `json:"delivery_ratio,omitempty"`
	DriverErrorCount   int     `json:"driver_error_count,omitempty"`
	QueueDropCount     int     `json:"queue_drop_count,omitempty"`
	NonzeroSampleCount int     `json:"nonzero_sample_count,omitempty"`
}

type physicalStackChanObservationEvidence struct {
	Available             bool    `json:"available"`
	PhysicalSoundObserved bool    `json:"physical_sound_observed"`
	OperatorConfirmed     bool    `json:"operator_confirmed"`
	Method                string  `json:"method,omitempty"`
	Instrument            string  `json:"instrument,omitempty"`
	ObservedAudibleMS     float64 `json:"observed_audible_ms,omitempty"`
	ObservedStopMS        float64 `json:"observed_stop_ms,omitempty"`
}

type physicalStackChanEvidenceFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type physicalStackChanEvidenceRedaction struct {
	UserTextStored            bool `json:"user_text_stored"`
	InstructionTextStored     bool `json:"instruction_text_stored"`
	ModelTextStored           bool `json:"model_text_stored"`
	AudioPayloadStored        bool `json:"audio_payload_stored"`
	EncodedAudioPayloadStored bool `json:"encoded_audio_payload_stored"`
	NetworkLocatorStored      bool `json:"network_locator_stored"`
	NetworkRouteStored        bool `json:"network_route_stored"`
	FilesystemLocatorStored   bool `json:"filesystem_locator_stored"`
	SecretMaterialStored      bool `json:"secret_material_stored"`
	InternalThoughtStored     bool `json:"internal_thought_stored"`
}

type physicalStackChanEvidenceFixture struct {
	SchemaVersion                     string                                `json:"schema_version"`
	ExecutionMode                     string                                `json:"execution_mode"`
	TraceID                           string                                `json:"trace_id"`
	SessionID                         string                                `json:"session_id"`
	DeviceID                          string                                `json:"device_id"`
	Execution                         physicalStackChanEvidenceExecution    `json:"execution"`
	DeviceDownlinkFirstFrameMS        *float64                              `json:"device_downlink_first_frame_ms"`
	DevicePlaybackStartMS             *float64                              `json:"device_playback_start_ms"`
	SpeechEndToFirstAudibleResponseMS *float64                              `json:"speech_end_to_first_audible_response_ms"`
	BargeInDetectedMS                 *float64                              `json:"barge_in_detected_ms"`
	BargeInStopMS                     *float64                              `json:"barge_in_stop_ms"`
	BargeInPlaybackStopRequestedMS    *float64                              `json:"barge_in_playback_stop_requested_ms"`
	BargeInPlaybackStopDoneMS         *float64                              `json:"barge_in_playback_stop_done_ms"`
	Mic                               *physicalStackChanMicEvidence         `json:"mic"`
	Observation                       *physicalStackChanObservationEvidence `json:"observation"`
}

func runPhysicalStackChanEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	var fixturePath string
	var outputDir string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 physical-stackchan-evidence --fixture reports/a21-physical-stackchan-fixture.json [--output-dir reports]")
			return 0
		case "--fixture":
			if !readStringOption(args, &i, stderr, "--fixture", &fixturePath) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &outputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown physical-stackchan-evidence option %q\n", args[i])
			return 2
		}
	}
	if strings.TrimSpace(fixturePath) == "" {
		fmt.Fprintln(stderr, "physical-stackchan-evidence requires --fixture")
		return 2
	}
	if err := validateA21InputPath(fixturePath); err != nil {
		fmt.Fprintf(stderr, "physical-stackchan-evidence fixture invalid: %v\n", err)
		return 2
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		fmt.Fprintf(stderr, "physical-stackchan-evidence fixture read failed for %s\n", filepath.Base(fixturePath))
		return 1
	}
	report, err := buildPhysicalStackChanEvidenceReport(filepath.Base(fixturePath), data)
	if err != nil {
		fmt.Fprintf(stderr, "physical-stackchan-evidence fixture invalid: %v\n", err)
		return 1
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "physical-stackchan-evidence report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writePhysicalStackChanEvidenceReport(outputDir, report)
		if err != nil {
			fmt.Fprintln(stderr, "physical-stackchan-evidence report write failed")
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONPhysicalStackChanEvidence(stdout, report); err != nil {
		fmt.Fprintln(stderr, "physical-stackchan-evidence report encode failed")
		return 1
	}
	return 0
}

func buildPhysicalStackChanEvidenceReport(fixtureBase string, data []byte) (physicalStackChanEvidenceReport, error) {
	var fixture physicalStackChanEvidenceFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return physicalStackChanEvidenceReport{}, err
	}
	mode := strings.TrimSpace(fixture.ExecutionMode)
	if mode == "" {
		mode = "physical_stackchan"
	}
	report := physicalStackChanEvidenceReport{
		SchemaVersion:    physicalStackChanEvidenceSchemaVersion,
		ExecutionMode:    mode,
		TraceID:          fixture.TraceID,
		SessionID:        fixture.SessionID,
		DeviceID:         fixture.DeviceID,
		FixturePath:      fixtureBase,
		Execution:        fixture.Execution,
		Redaction:        physicalStackChanEvidenceRedaction{},
		PromotionGate:    "not_production",
		AcceptanceStatus: "blocked",
		PRDAccepted:      false,
	}
	if mode != "physical_stackchan" {
		report.CanonicalMetrics = physicalStackChanCanonicalMetrics{}
		report.StageAvailability = physicalStackChanStageAvailability(report.CanonicalMetrics)
		report.Mic = physicalStackChanMicEvidence{}
		report.Observation = physicalStackChanObservationEvidence{}
		report.AcceptanceStatus = "candidate_host_only"
		report.Findings = append(report.Findings, physicalStackChanFinding("host_loopback_only", "info", "host loopback evidence is not physical StackChan acceptance"))
		return report, nil
	}
	report.CanonicalMetrics = physicalStackChanCanonicalMetrics{
		DeviceDownlinkFirstFrameMS:        physicalStackChanMetricFromValue(fixture.DeviceDownlinkFirstFrameMS, "device_runtime_echo"),
		DevicePlaybackStartMS:             physicalStackChanMetricFromValue(fixture.DevicePlaybackStartMS, "device_runtime_echo"),
		SpeechEndToFirstAudibleResponseMS: physicalStackChanMetricFromValue(fixture.SpeechEndToFirstAudibleResponseMS, "operator_or_instrument"),
		BargeInDetectedMS:                 physicalStackChanMetricFromValue(fixture.BargeInDetectedMS, "gateway_trace"),
		BargeInStopMS:                     physicalStackChanMetricFromValue(fixture.BargeInStopMS, "operator_or_instrument"),
		BargeInPlaybackStopRequestedMS:    physicalStackChanMetricFromValue(fixture.BargeInPlaybackStopRequestedMS, "gateway_trace"),
		BargeInPlaybackStopDoneMS:         physicalStackChanMetricFromValue(fixture.BargeInPlaybackStopDoneMS, "device_runtime_echo"),
	}
	report.StageAvailability = physicalStackChanStageAvailability(report.CanonicalMetrics)
	if fixture.Mic != nil {
		report.Mic = *fixture.Mic
		report.Mic.Available = physicalStackChanMicAvailable(*fixture.Mic)
	}
	if fixture.Observation != nil {
		report.Observation = *fixture.Observation
		report.Observation.Available = physicalStackChanObservationAvailable(*fixture.Observation)
	}
	if physicalStackChanFixtureHasUnsafeContent(data) {
		report.Findings = append(report.Findings, physicalStackChanFinding("unsafe_fixture_content_redacted", "error", "fixture contained forbidden raw content and was redacted from the report"))
	}
	report.Findings = append(report.Findings, physicalStackChanMissingFindings(report)...)
	if len(report.Findings) == 0 {
		report.PromotionGate = "candidate"
		report.AcceptanceStatus = "physical_review_required"
		report.Findings = append(report.Findings, physicalStackChanFinding("physical_review_required", "info", "physical metrics are present but still require explicit review before PRD acceptance"))
	}
	return report, nil
}

func physicalStackChanMetricFromValue(value *float64, source string) physicalStackChanMetric {
	if value == nil || *value <= 0 {
		return physicalStackChanMetric{Available: false}
	}
	return physicalStackChanMetric{Available: true, ValueMS: *value, Source: source}
}

func physicalStackChanStageAvailability(metrics physicalStackChanCanonicalMetrics) map[string]physicalStackChanMetric {
	return map[string]physicalStackChanMetric{
		"device.downlink.first_frame":          metrics.DeviceDownlinkFirstFrameMS,
		"device.playback.start":                metrics.DevicePlaybackStartMS,
		"speech_end_to_first_audible_response": metrics.SpeechEndToFirstAudibleResponseMS,
		"barge_in.detected":                    metrics.BargeInDetectedMS,
		"barge_in.stop":                        metrics.BargeInStopMS,
		"barge_in.playback_stop_requested":     metrics.BargeInPlaybackStopRequestedMS,
		"barge_in.playback_stop_done":          metrics.BargeInPlaybackStopDoneMS,
	}
}

func physicalStackChanMicAvailable(mic physicalStackChanMicEvidence) bool {
	return mic.FramesCaptured > 0 && mic.FramesDelivered > 0 && mic.RMS > 0 && mic.DeliveryRatio > 0 && mic.NonzeroSampleCount > 0
}

func physicalStackChanObservationAvailable(observation physicalStackChanObservationEvidence) bool {
	return observation.PhysicalSoundObserved && observation.OperatorConfirmed &&
		strings.TrimSpace(observation.Method) != "" && strings.TrimSpace(observation.Instrument) != "" &&
		observation.ObservedAudibleMS > 0 && observation.ObservedStopMS > 0
}

func physicalStackChanMissingFindings(report physicalStackChanEvidenceReport) []physicalStackChanEvidenceFinding {
	var findings []physicalStackChanEvidenceFinding
	if !report.CanonicalMetrics.DeviceDownlinkFirstFrameMS.Available {
		findings = append(findings, physicalStackChanFinding("physical_downlink_receipt_missing", "error", "physical device downlink first-frame evidence is missing"))
	}
	if !report.CanonicalMetrics.DevicePlaybackStartMS.Available {
		findings = append(findings, physicalStackChanFinding("physical_playback_start_missing", "error", "physical device playback-start evidence is missing"))
	}
	if !report.CanonicalMetrics.SpeechEndToFirstAudibleResponseMS.Available {
		findings = append(findings, physicalStackChanFinding("physical_audible_response_missing", "error", "physical first audible response evidence is missing"))
	}
	if !report.CanonicalMetrics.BargeInStopMS.Available ||
		!report.CanonicalMetrics.BargeInDetectedMS.Available ||
		!report.CanonicalMetrics.BargeInPlaybackStopRequestedMS.Available ||
		!report.CanonicalMetrics.BargeInPlaybackStopDoneMS.Available {
		findings = append(findings, physicalStackChanFinding("physical_barge_in_stop_missing", "error", "physical barge-in stop evidence is missing"))
	}
	if !report.Mic.Available {
		findings = append(findings, physicalStackChanFinding("physical_mic_counters_rms_missing", "error", "physical microphone counters and RMS evidence are missing"))
	}
	if !report.Observation.Available {
		findings = append(findings, physicalStackChanFinding("physical_operator_instrument_observation_missing", "error", "operator or instrument observation evidence is missing"))
	}
	return findings
}

func physicalStackChanFinding(code string, severity string, message string) physicalStackChanEvidenceFinding {
	return physicalStackChanEvidenceFinding{Code: code, Severity: severity, Message: message}
}

func physicalStackChanFixtureHasUnsafeContent(data []byte) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	return physicalStackChanValueUnsafe(value)
}

func physicalStackChanValueUnsafe(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if physicalStackChanUnsafeKey(key) || physicalStackChanValueUnsafe(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if physicalStackChanValueUnsafe(child) {
				return true
			}
		}
	case string:
		return physicalStackChanUnsafeString(typed)
	}
	return false
}

func physicalStackChanUnsafeKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "transcript", "prompt", "provider_output", "raw_audio", "data_base64", "audio_base64", "base64_audio", "url", "proxy", "local_path", "reasoning":
		return true
	default:
		return false
	}
}

func physicalStackChanUnsafeString(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"http://", "https://", "secret-token", "api_key", "sk-", "proxy.local", "/users/", "/private/", "raw_audio", "data_base64",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func writePhysicalStackChanEvidenceReport(outputDir string, report physicalStackChanEvidenceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-physical-stackchan-evidence-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONPhysicalStackChanEvidence(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONPhysicalStackChanEvidence(writer io.Writer, report physicalStackChanEvidenceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
