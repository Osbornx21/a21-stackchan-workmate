package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const xiaozhiPhysicalEvidenceSchemaVersion = "a21.xiaozhi_physical_evidence.v1"
const xiaozhiInstrumentObservationSchemaVersion = "a21.xiaozhi_instrument_observation.v1"
const xiaozhiInstrumentTimingToleranceMS = 5
const xiaozhiInstrumentMaxTimingMS = 60000

type xiaozhiPhysicalEvidenceOptions struct {
	GatewayURL                  string
	DeviceID                    string
	TraceID                     string
	SessionID                   string
	InstrumentObservationReport string
	OutputDir                   string
}

type xiaozhiPhysicalEvidenceReport struct {
	SchemaVersion        string                                `json:"schema_version"`
	GeneratedAtMS        int64                                 `json:"generated_at_ms"`
	ExecutionMode        string                                `json:"execution_mode"`
	Gateway              string                                `json:"gateway,omitempty"`
	DeviceID             string                                `json:"device_id"`
	TraceID              string                                `json:"trace_id"`
	SessionID            string                                `json:"session_id"`
	Profile              string                                `json:"profile"`
	PhysicalDeviceOnline bool                                  `json:"physical_device_online"`
	AudioFrameCount      int                                   `json:"audio_frame_count"`
	Execution            physicalStackChanEvidenceExecution    `json:"execution"`
	StageAvailability    map[string]physicalStackChanMetric    `json:"stage_availability"`
	GatewayMetrics       xiaozhiPhysicalEvidenceGatewayMetrics `json:"gateway_metrics"`
	CanonicalMetrics     physicalStackChanCanonicalMetrics     `json:"canonical_metrics"`
	Mic                  physicalStackChanMicEvidence          `json:"mic"`
	Observation          physicalStackChanObservationEvidence  `json:"observation"`
	Findings             []physicalStackChanEvidenceFinding    `json:"findings"`
	Redaction            physicalStackChanEvidenceRedaction    `json:"redaction"`
	PromotionGate        string                                `json:"promotion_gate"`
	AcceptanceStatus     string                                `json:"acceptance_status"`
	PRDAccepted          bool                                  `json:"prd_accepted"`
	ReportPath           string                                `json:"report_path,omitempty"`
}

type xiaozhiPhysicalEvidenceGatewayMetrics struct {
	GatewayAnswerFirstDownlinkMS physicalStackChanMetric `json:"gateway_answer_first_downlink_ms"`
}

type xiaozhiInstrumentObservationReport struct {
	SchemaVersion                     string                             `json:"schema_version"`
	TraceID                           string                             `json:"trace_id"`
	SessionID                         string                             `json:"session_id"`
	DeviceID                          string                             `json:"device_id"`
	Method                            string                             `json:"method"`
	Instrument                        string                             `json:"instrument"`
	PhysicalSoundObserved             bool                               `json:"physical_sound_observed"`
	ObservedNonzeroAudibleEnergy      bool                               `json:"observed_nonzero_audible_energy"`
	DevicePlaybackObserved            bool                               `json:"device_playback_observed,omitempty"`
	DevicePlaybackObservationSource   string                             `json:"device_playback_observation_source,omitempty"`
	GatewayFirstDownlinkToPlaybackMS  float64                            `json:"gateway_first_downlink_to_device_playback_start_ms,omitempty"`
	GatewayFirstDownlinkToAudibleMS   float64                            `json:"gateway_first_downlink_to_audible_ms"`
	SpeechEndToFirstAudibleResponseMS float64                            `json:"speech_end_to_first_audible_response_ms"`
	AudibleEnergyRMS                  float64                            `json:"audible_energy_rms"`
	NoiseFloorRMS                     float64                            `json:"noise_floor_rms"`
	Redaction                         physicalStackChanEvidenceRedaction `json:"redaction"`
}

func runXiaozhiPhysicalEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiPhysicalEvidenceOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		OutputDir:  "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-physical-evidence --gateway-url http://127.0.0.1:21080 --device-id <device-id> --trace-id <trace-id> --session-id <session-id> [--instrument-observation-report report.json] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown xiaozhi-physical-evidence option %q\n", args[i])
			return 2
		}
	}
	if err := validateXiaozhiPhysicalEvidenceOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical evidence option invalid: %v\n", err)
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical evidence report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildXiaozhiPhysicalEvidenceReport(options)
	if err != nil {
		if err.Error() == "gateway data unsafe" {
			fmt.Fprintln(stderr, "xiaozhi physical evidence gateway data unsafe")
			return 1
		}
		if err.Error() == "instrument observation unsafe" {
			fmt.Fprintln(stderr, "xiaozhi physical evidence instrument observation unsafe")
			return 1
		}
		fmt.Fprintf(stderr, "xiaozhi physical evidence failed: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiPhysicalEvidenceReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintln(stderr, "xiaozhi physical evidence report write failed")
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiPhysicalEvidence(stdout, report); err != nil {
		fmt.Fprintln(stderr, "xiaozhi physical evidence report encode failed")
		return 1
	}
	return 0
}

func validateXiaozhiPhysicalEvidenceOptions(options xiaozhiPhysicalEvidenceOptions) error {
	if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
		return err
	}
	if strings.TrimSpace(options.InstrumentObservationReport) != "" {
		if err := validateA21InputPath(options.InstrumentObservationReport); err != nil {
			return err
		}
	}
	for name, value := range map[string]string{
		"device_id":  options.DeviceID,
		"trace_id":   options.TraceID,
		"session_id": options.SessionID,
	} {
		if !xiaozhiPhysicalSafeID(value) {
			return fmt.Errorf("%s is invalid or unsafe", name)
		}
	}
	return nil
}

func buildXiaozhiPhysicalEvidenceReport(options xiaozhiPhysicalEvidenceOptions) (xiaozhiPhysicalEvidenceReport, error) {
	deviceReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		return xiaozhiPhysicalEvidenceReport{}, err
	}
	device, ok := findFirmwareDeviceRecord(deviceReport.Devices, options.DeviceID)
	if !ok {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("device not found in gateway registry")
	}
	trace, err := fetchGatewayTrace(options.GatewayURL, options.TraceID)
	if err != nil {
		return xiaozhiPhysicalEvidenceReport{}, err
	}
	audioRecent, err := fetchXiaozhiPhysicalRecentAudio(options)
	if err != nil {
		return xiaozhiPhysicalEvidenceReport{}, err
	}
	if xiaozhiPhysicalGatewayDataUnsafe(device, trace, audioRecent) {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("gateway data unsafe")
	}
	report := xiaozhiPhysicalEvidenceReport{
		SchemaVersion:        xiaozhiPhysicalEvidenceSchemaVersion,
		GeneratedAtMS:        time.Now().UnixMilli(),
		ExecutionMode:        "physical_xiaozhi_gateway",
		Gateway:              productSurfaceLabel(options.GatewayURL, ""),
		DeviceID:             options.DeviceID,
		TraceID:              options.TraceID,
		SessionID:            options.SessionID,
		Profile:              firstNonEmpty(strings.TrimSpace(device.Capabilities["xiaozhi_profile"]), "unknown"),
		PhysicalDeviceOnline: xiaozhiPhysicalDeviceOnline(device),
		AudioFrameCount:      len(audioRecent.Frames),
		Execution: physicalStackChanEvidenceExecution{
			ProviderExecuted: false,
			V21Executed:      false,
			HardwareExecuted: false,
		},
		Redaction:        physicalStackChanEvidenceRedaction{},
		PromotionGate:    "not_production",
		AcceptanceStatus: "candidate_gateway_downlink",
		PRDAccepted:      false,
	}
	report.GatewayMetrics = xiaozhiPhysicalGatewayMetricsFromTrace(trace)
	report.CanonicalMetrics = physicalStackChanCanonicalMetrics{
		DeviceDownlinkFirstFrameMS:        physicalStackChanMetric{},
		DevicePlaybackStartMS:             physicalStackChanMetric{},
		SpeechEndToFirstAudibleResponseMS: physicalStackChanMetric{},
		BargeInDetectedMS:                 xiaozhiPhysicalTracePresenceMetric(trace, "barge_in.detected", "gateway_trace"),
		BargeInStopMS: physicalStackChanMetricFromNonNegativeInt64(firstNonNilInt64(
			trace.Summary.BargeInStopMS,
			xiaozhiPhysicalTraceDeltaMS(trace, "barge_in.detected", "playback.stop"),
		), "gateway_trace"),
		BargeInPlaybackStopRequestedMS: physicalStackChanMetricFromNonNegativeInt64(
			xiaozhiPhysicalTraceDeltaMS(trace, "barge_in.detected", "playback.stop"),
			"gateway_trace",
		),
		BargeInPlaybackStopDoneMS: physicalStackChanMetric{},
	}
	report.Mic = xiaozhiPhysicalMicEvidence(audioRecent.Frames)
	report.Observation = physicalStackChanObservationEvidence{}
	var instrumentFindings []physicalStackChanEvidenceFinding
	if strings.TrimSpace(options.InstrumentObservationReport) != "" {
		observation, findings, err := loadXiaozhiInstrumentObservationReport(options, trace, report.GatewayMetrics.GatewayAnswerFirstDownlinkMS)
		if err != nil {
			return xiaozhiPhysicalEvidenceReport{}, err
		}
		instrumentFindings = append(instrumentFindings, findings...)
		if observation != nil {
			report.CanonicalMetrics.DeviceDownlinkFirstFrameMS = report.GatewayMetrics.GatewayAnswerFirstDownlinkMS
			report.CanonicalMetrics.DevicePlaybackStartMS = physicalStackChanMetricFromInt64(trace.Summary.DevicePlaybackStartMS, "device_runtime_echo")
			if !report.CanonicalMetrics.DevicePlaybackStartMS.Available && xiaozhiInstrumentPlaybackObservationValid(*observation) {
				report.CanonicalMetrics.DevicePlaybackStartMS = physicalStackChanMetric{
					Available: true,
					ValueMS:   observation.GatewayFirstDownlinkToPlaybackMS,
					Source:    "trusted_runtime_observation",
				}
			}
			if observation.DevicePlaybackObserved && !report.CanonicalMetrics.DevicePlaybackStartMS.Available {
				instrumentFindings = append(instrumentFindings, physicalStackChanFinding("xiaozhi_physical_instrument_playback_timing_invalid", "error", "instrument playback observation timing was invalid"))
			}
			report.CanonicalMetrics.SpeechEndToFirstAudibleResponseMS = physicalStackChanMetric{
				Available: true,
				ValueMS:   observation.SpeechEndToFirstAudibleResponseMS,
				Source:    "instrument_observation",
			}
			report.Observation = physicalStackChanObservationEvidence{
				Available:             true,
				PhysicalSoundObserved: true,
				OperatorConfirmed:     false,
				Method:                observation.Method,
				Instrument:            observation.Instrument,
				ObservedAudibleMS:     observation.SpeechEndToFirstAudibleResponseMS,
			}
		}
	}
	if report.CanonicalMetrics.DevicePlaybackStartMS.Available && xiaozhiPhysicalObservationAvailable(report.Observation) {
		report.PromotionGate = "candidate"
		report.AcceptanceStatus = "physical_review_required"
	}
	report.StageAvailability = xiaozhiPhysicalStageAvailability(report, trace)
	report.Findings = xiaozhiPhysicalFindings(report, trace)
	report.Findings = append(report.Findings, instrumentFindings...)
	return report, nil
}

func xiaozhiPhysicalDeviceOnline(device firmwarecheck.DeviceIdentityRecord) bool {
	return device.DeviceID != "" &&
		(device.ConnectionStatus == "" || device.ConnectionStatus == "online") &&
		!strings.Contains(strings.ToLower(device.DeviceID), "sim") &&
		strings.TrimSpace(device.Capabilities["xiaozhi_transport"]) == "websocket"
}

func xiaozhiPhysicalGatewayMetricsFromTrace(trace gateway.TraceResponse) xiaozhiPhysicalEvidenceGatewayMetrics {
	return xiaozhiPhysicalEvidenceGatewayMetrics{
		GatewayAnswerFirstDownlinkMS: physicalStackChanMetricFromInt64(firstNonNilInt64(
			xiaozhiPhysicalSpeechEndToFirstDownlinkMS(trace),
			trace.Summary.AnswerFirstAudioTotalMS,
		), "gateway_trace"),
	}
}

func xiaozhiPhysicalSpeechEndToFirstDownlinkMS(trace gateway.TraceResponse) *int64 {
	var speechEndAt int64
	speechEndSeen := false
	for _, event := range trace.Events {
		if event.Name != "vad.speech.end" {
			continue
		}
		speechEndAt = event.AtMS
		speechEndSeen = true
		break
	}
	if !speechEndSeen {
		return nil
	}
	for _, event := range trace.Events {
		if event.AtMS < speechEndAt {
			continue
		}
		if event.Name != "audio.downlink.first_frame" && event.Name != "xiaozhi.tts.opus_frame.downlink" {
			continue
		}
		delta := event.AtMS - speechEndAt
		if delta < 0 {
			return nil
		}
		return &delta
	}
	return nil
}

func xiaozhiPhysicalTraceDeltaMS(trace gateway.TraceResponse, startName string, endName string) *int64 {
	var startAt int64
	startSeen := false
	for _, event := range trace.Events {
		if event.Name != startName {
			continue
		}
		startAt = event.AtMS
		startSeen = true
		break
	}
	if !startSeen {
		return nil
	}
	for _, event := range trace.Events {
		if event.Name != endName || event.AtMS < startAt {
			continue
		}
		delta := event.AtMS - startAt
		return &delta
	}
	return nil
}

func xiaozhiPhysicalTracePresenceMetric(trace gateway.TraceResponse, name string, source string) physicalStackChanMetric {
	if !xiaozhiTraceHasEvent(trace, name) {
		return physicalStackChanMetric{}
	}
	return physicalStackChanMetric{Available: true, Source: source}
}

func xiaozhiPhysicalStageAvailability(report xiaozhiPhysicalEvidenceReport, trace gateway.TraceResponse) map[string]physicalStackChanMetric {
	return map[string]physicalStackChanMetric{
		"physical_xiaozhi.online":      xiaozhiPhysicalBoolMetric(report.PhysicalDeviceOnline, "gateway_device_registry"),
		"xiaozhi.profile.stock":        xiaozhiPhysicalBoolMetric(report.Profile == "stock", "gateway_device_registry"),
		"xiaozhi.opus.decode":          xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "xiaozhi.opus_frame.decoded"), "gateway_trace"),
		"audio.ingress.pcm":            xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "audio.ingress.buffered"), "gateway_trace"),
		"vad.speech.end":               xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "vad.speech.end"), "gateway_trace"),
		"xiaozhi.listen.auto_stop":     xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "xiaozhi.listen.auto_stop"), "gateway_trace"),
		"xiaozhi.tts.downlink":         xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "xiaozhi.tts.opus_frame.downlink"), "gateway_trace"),
		"answer.first_downlink":        report.GatewayMetrics.GatewayAnswerFirstDownlinkMS,
		"device.playback.ack":          xiaozhiPhysicalBoolMetric(xiaozhiTraceHasEvent(trace, "device.playback.start") || report.CanonicalMetrics.DevicePlaybackStartMS.Available, "device_runtime_echo"),
		"operator.audible_observation": xiaozhiPhysicalBoolMetric(xiaozhiPhysicalObservationAvailable(report.Observation), "instrument_observation"),
	}
}

func xiaozhiPhysicalFindings(report xiaozhiPhysicalEvidenceReport, trace gateway.TraceResponse) []physicalStackChanEvidenceFinding {
	var findings []physicalStackChanEvidenceFinding
	if !report.StageAvailability["physical_xiaozhi.online"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_device_offline", "error", "physical Xiaozhi device is not online in the A21 Gateway registry"))
	}
	if !report.StageAvailability["xiaozhi.profile.stock"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_stock_profile_missing", "error", "stock Xiaozhi profile evidence is missing"))
	}
	for _, stage := range []struct {
		key  string
		code string
		msg  string
	}{
		{"xiaozhi.opus.decode", "xiaozhi_physical_opus_decode_missing", "Xiaozhi Opus decode trace evidence is missing"},
		{"audio.ingress.pcm", "xiaozhi_physical_pcm_ingress_missing", "PCM ingress trace evidence is missing"},
		{"vad.speech.end", "xiaozhi_physical_vad_speech_end_missing", "VAD speech-end trace evidence is missing"},
		{"xiaozhi.listen.auto_stop", "xiaozhi_physical_auto_stop_missing", "Xiaozhi listen auto-stop trace evidence is missing"},
		{"xiaozhi.tts.downlink", "xiaozhi_physical_tts_downlink_missing", "Xiaozhi TTS downlink trace evidence is missing"},
		{"answer.first_downlink", "xiaozhi_physical_answer_first_downlink_missing", "Gateway answer first-downlink timing is missing"},
	} {
		if !report.StageAvailability[stage.key].Available {
			findings = append(findings, physicalStackChanFinding(stage.code, "error", stage.msg))
		}
	}
	if report.PhysicalDeviceOnline && report.GatewayMetrics.GatewayAnswerFirstDownlinkMS.Available && xiaozhiTraceHasEvent(trace, "xiaozhi.tts.opus_frame.downlink") {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_gateway_downlink_candidate", "info", "stock Xiaozhi physical device reached Gateway downlink, but this is not audible playback acceptance"))
	}
	if !report.StageAvailability["device.playback.ack"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_device_playback_ack_missing", "error", "missing device playback ack such as device.playback.start or trusted runtime echo"))
	}
	if !report.StageAvailability["operator.audible_observation"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_operator_observation_missing", "error", "missing operator audible observation or instrumented first audible playback evidence"))
	}
	if report.StageAvailability["operator.audible_observation"].Available {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_instrument_observation_present", "info", "instrument observed nonzero audible energy after Gateway first downlink"))
	}
	if report.PromotionGate == "candidate" && (!report.CanonicalMetrics.BargeInStopMS.Available ||
		!report.CanonicalMetrics.BargeInDetectedMS.Available ||
		!report.CanonicalMetrics.BargeInPlaybackStopRequestedMS.Available ||
		!report.CanonicalMetrics.BargeInPlaybackStopDoneMS.Available) {
		findings = append(findings, physicalStackChanFinding("xiaozhi_physical_barge_in_stop_missing", "error", "physical barge-in stop evidence is missing"))
	}
	return findings
}

func loadXiaozhiInstrumentObservationReport(options xiaozhiPhysicalEvidenceOptions, trace gateway.TraceResponse, gatewayFirstDownlink physicalStackChanMetric) (*xiaozhiInstrumentObservationReport, []physicalStackChanEvidenceFinding, error) {
	path := strings.TrimSpace(options.InstrumentObservationReport)
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || physicalStackChanValueUnsafe(raw) {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	var observation xiaozhiInstrumentObservationReport
	if err := json.Unmarshal(data, &observation); err != nil {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	if !xiaozhiInstrumentObservationSafe(observation) {
		return nil, nil, fmt.Errorf("instrument observation unsafe")
	}
	if observation.TraceID != options.TraceID || observation.SessionID != options.SessionID || observation.DeviceID != options.DeviceID {
		return nil, []physicalStackChanEvidenceFinding{
			physicalStackChanFinding("xiaozhi_physical_instrument_observation_mismatch", "error", "instrument observation trace/session/device did not match Gateway evidence"),
		}, nil
	}
	if !xiaozhiInstrumentObservationTimingValid(observation, trace, gatewayFirstDownlink) {
		return nil, []physicalStackChanEvidenceFinding{
			physicalStackChanFinding("xiaozhi_physical_instrument_observation_timing_invalid", "error", "instrument observation timing did not follow Gateway first downlink"),
		}, nil
	}
	return &observation, nil, nil
}

func xiaozhiInstrumentObservationSafe(observation xiaozhiInstrumentObservationReport) bool {
	if observation.SchemaVersion != xiaozhiInstrumentObservationSchemaVersion ||
		!productPhysicalStackChanRedactionOK(observation.Redaction) {
		return false
	}
	for _, value := range []string{
		observation.TraceID,
		observation.SessionID,
		observation.DeviceID,
		observation.Method,
		observation.Instrument,
	} {
		if !xiaozhiPhysicalSafeID(value) {
			return false
		}
	}
	if strings.TrimSpace(observation.DevicePlaybackObservationSource) != "" && !xiaozhiPhysicalSafeID(observation.DevicePlaybackObservationSource) {
		return false
	}
	return observation.PhysicalSoundObserved &&
		observation.ObservedNonzeroAudibleEnergy &&
		observation.AudibleEnergyRMS > 0 &&
		observation.NoiseFloorRMS > 0 &&
		observation.AudibleEnergyRMS > observation.NoiseFloorRMS
}

func xiaozhiInstrumentPlaybackObservationValid(observation xiaozhiInstrumentObservationReport) bool {
	if !observation.DevicePlaybackObserved ||
		strings.TrimSpace(observation.DevicePlaybackObservationSource) == "" ||
		!xiaozhiPlausibleInstrumentTiming(observation.GatewayFirstDownlinkToPlaybackMS) {
		return false
	}
	if observation.GatewayFirstDownlinkToPlaybackMS > observation.GatewayFirstDownlinkToAudibleMS+xiaozhiInstrumentTimingToleranceMS {
		return false
	}
	return true
}

func xiaozhiInstrumentObservationTimingValid(observation xiaozhiInstrumentObservationReport, trace gateway.TraceResponse, gatewayFirstDownlink physicalStackChanMetric) bool {
	if !gatewayFirstDownlink.Available ||
		!xiaozhiPlausibleInstrumentTiming(observation.GatewayFirstDownlinkToAudibleMS) ||
		!xiaozhiPlausibleInstrumentTiming(observation.SpeechEndToFirstAudibleResponseMS) {
		return false
	}
	if !xiaozhiTraceHasEvent(trace, "xiaozhi.tts.opus_frame.downlink") && !xiaozhiTraceHasEvent(trace, "audio.downlink.first_frame") {
		return false
	}
	expectedAudibleMS := gatewayFirstDownlink.ValueMS + observation.GatewayFirstDownlinkToAudibleMS
	return xiaozhiTimingWithinTolerance(expectedAudibleMS, observation.SpeechEndToFirstAudibleResponseMS, xiaozhiInstrumentTimingToleranceMS)
}

func xiaozhiPlausibleInstrumentTiming(value float64) bool {
	return value > 0 && value <= xiaozhiInstrumentMaxTimingMS
}

func xiaozhiTimingWithinTolerance(a float64, b float64, tolerance float64) bool {
	delta := a - b
	if delta < 0 {
		delta = -delta
	}
	return delta <= tolerance
}

func xiaozhiPhysicalObservationAvailable(observation physicalStackChanObservationEvidence) bool {
	return observation.Available &&
		observation.PhysicalSoundObserved &&
		strings.TrimSpace(observation.Method) != "" &&
		strings.TrimSpace(observation.Instrument) != "" &&
		observation.ObservedAudibleMS > 0
}

func xiaozhiPhysicalMicEvidence(frames []gateway.AudioCaptureFrame) physicalStackChanMicEvidence {
	if len(frames) == 0 {
		return physicalStackChanMicEvidence{}
	}
	totalBytes := 0
	delivered := 0
	rmsTotal := 0.0
	nonzeroSamples := 0
	for _, frame := range frames {
		totalBytes += frame.DataBytes
		if frame.DataBytes > 0 {
			delivered++
			nonzeroSamples += frame.DataBytes / 2
		}
		rmsTotal += frame.RMS
	}
	mic := physicalStackChanMicEvidence{
		FramesCaptured:     len(frames),
		FramesDelivered:    delivered,
		RMS:                roundedStackChanDiagnosticValue(rmsTotal / float64(len(frames))),
		DeliveryRatio:      roundedStackChanDiagnosticRatio(delivered, len(frames)),
		NonzeroSampleCount: nonzeroSamples,
	}
	mic.Available = mic.FramesCaptured > 0 && mic.FramesDelivered > 0 && mic.RMS > 0 && totalBytes > 0
	return mic
}

func fetchXiaozhiPhysicalRecentAudio(options xiaozhiPhysicalEvidenceOptions) (gateway.AudioRecentResponse, error) {
	query := url.Values{}
	query.Set("device_id", options.DeviceID)
	query.Set("trace_id", options.TraceID)
	query.Set("session_id", options.SessionID)
	query.Set("limit", "64")
	endpoint, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/audio/recent", query)
	if err != nil {
		return gateway.AudioRecentResponse{}, err
	}
	client := http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Get(endpoint)
	if err != nil {
		return gateway.AudioRecentResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return gateway.AudioRecentResponse{}, fmt.Errorf("gateway audio recent returned status %d", resp.StatusCode)
	}
	var response gateway.AudioRecentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.AudioRecentResponse{}, err
	}
	if response.SchemaVersion != gateway.AudioRecentSchemaVersion {
		return gateway.AudioRecentResponse{}, fmt.Errorf("gateway audio recent has unexpected schema")
	}
	return response, nil
}

func xiaozhiPhysicalGatewayDataUnsafe(device firmwarecheck.DeviceIdentityRecord, trace gateway.TraceResponse, audioRecent gateway.AudioRecentResponse) bool {
	for _, value := range []string{
		device.DeviceID,
		device.Firmware.ID,
		device.Firmware.Version,
		device.Firmware.Board,
		device.Firmware.Commit,
		device.LastTraceID,
		device.LastSessionID,
		trace.TraceID,
		audioRecent.DeviceID,
		audioRecent.TraceID,
		audioRecent.SessionID,
	} {
		if xiaozhiPhysicalUnsafeString(value) {
			return true
		}
	}
	for key, value := range device.Capabilities {
		if xiaozhiPhysicalUnsafeString(key) || xiaozhiPhysicalUnsafeString(value) {
			return true
		}
	}
	for key, value := range device.RuntimeEcho {
		if xiaozhiPhysicalUnsafeString(key) || xiaozhiPhysicalUnsafeString(value) {
			return true
		}
	}
	for _, event := range trace.Events {
		if xiaozhiPhysicalUnsafeString(event.Name) ||
			xiaozhiPhysicalUnsafeString(event.TraceID) ||
			xiaozhiPhysicalUnsafeString(event.SessionID) ||
			xiaozhiPhysicalUnsafeString(event.DeviceID) {
			return true
		}
	}
	for _, frame := range audioRecent.Frames {
		if frame.DataBase64 != "" ||
			xiaozhiPhysicalUnsafeString(frame.DeviceID) ||
			xiaozhiPhysicalUnsafeString(frame.TraceID) ||
			xiaozhiPhysicalUnsafeString(frame.SessionID) ||
			xiaozhiPhysicalUnsafeString(frame.VADDetector) ||
			xiaozhiPhysicalUnsafeString(frame.VADStatus) ||
			xiaozhiPhysicalUnsafeString(frame.VADFinding) {
			return true
		}
	}
	return false
}

func xiaozhiPhysicalUnsafeString(value string) bool {
	if physicalStackChanUnsafeString(value) {
		return true
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"transcript", "prompt", "provider output", "provider_output", "reasoning", "secret", "token"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func xiaozhiPhysicalSafeID(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 160 && !containsLegacyIdentity(value) && !xiaozhiPhysicalUnsafeString(value)
}

func xiaozhiTraceHasEvent(trace gateway.TraceResponse, name string) bool {
	for _, event := range trace.Events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func xiaozhiPhysicalBoolMetric(available bool, source string) physicalStackChanMetric {
	if !available {
		return physicalStackChanMetric{Available: false}
	}
	return physicalStackChanMetric{Available: true, Source: source}
}

func physicalStackChanMetricFromInt64(value *int64, source string) physicalStackChanMetric {
	if value == nil || *value <= 0 {
		return physicalStackChanMetric{Available: false}
	}
	return physicalStackChanMetric{Available: true, ValueMS: float64(*value), Source: source}
}

func physicalStackChanMetricFromNonNegativeInt64(value *int64, source string) physicalStackChanMetric {
	if value == nil || *value < 0 {
		return physicalStackChanMetric{Available: false}
	}
	return physicalStackChanMetric{Available: true, ValueMS: float64(*value), Source: source}
}

func firstNonNilInt64(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func writeXiaozhiPhysicalEvidenceReport(outputDir string, report xiaozhiPhysicalEvidenceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-xiaozhi-physical-evidence-"+time.Now().Format("20060102-150405.000000000")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONXiaozhiPhysicalEvidence(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONXiaozhiPhysicalEvidence(writer io.Writer, report xiaozhiPhysicalEvidenceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
