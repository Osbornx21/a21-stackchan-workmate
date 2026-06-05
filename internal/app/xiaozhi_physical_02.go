package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/gateway"
)

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
	client := *a21DirectHTTPClient(3 * time.Second)
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
	// This report never serializes the full device runtime echo. Scanning every
	// echo key here rejects safe product booleans such as
	// roleplay_prompt_text_stored=false while adding no redaction value.
	for key, value := range xiaozhiPhysicalConsumedRuntimeEcho(device.RuntimeEcho) {
		if xiaozhiPhysicalUnsafeString(key) || xiaozhiPhysicalUnsafeString(value) {
			return true
		}
	}
	for _, event := range trace.Events {
		if xiaozhiPhysicalTraceEventNameUnsafe(event.Name) ||
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

func xiaozhiPhysicalTraceEventNameUnsafe(name string) bool {
	switch strings.TrimSpace(name) {
	case "roleplay.prompt_input.used":
		return false
	default:
		return xiaozhiPhysicalUnsafeString(name)
	}
}

func xiaozhiPhysicalConsumedRuntimeEcho(echo map[string]string) map[string]string {
	if len(echo) == 0 {
		return nil
	}
	consumed := make(map[string]string)
	for _, key := range []string{
		"last_state_reaction_status",
		"last_state_reaction_state",
		"last_state_reaction_reason",
		"last_state_reaction_tool",
		"robot_led_red",
		"robot_led_green",
		"robot_led_blue",
		"robot_head_yaw",
		"robot_head_pitch",
		"robot_head_speed",
		"xiaozhi_mcp_tool",
	} {
		if value, ok := echo[key]; ok {
			consumed[key] = value
		}
	}
	return consumed
}

func xiaozhiPhysicalEvidenceMatchesTarget(options xiaozhiPhysicalEvidenceOptions, device firmwarecheck.DeviceIdentityRecord, trace gateway.TraceResponse, audioRecent gateway.AudioRecentResponse) bool {
	targetDeviceID := strings.TrimSpace(options.DeviceID)
	targetTraceID := strings.TrimSpace(options.TraceID)
	targetSessionID := strings.TrimSpace(options.SessionID)
	deviceSessionID := xiaozhiPhysicalDeviceRuntimeSessionID(targetDeviceID)
	if strings.TrimSpace(device.DeviceID) != targetDeviceID ||
		strings.TrimSpace(trace.TraceID) != targetTraceID ||
		strings.TrimSpace(audioRecent.DeviceID) != targetDeviceID ||
		strings.TrimSpace(audioRecent.TraceID) != targetTraceID ||
		!xiaozhiPhysicalTargetSessionMatch(audioRecent.SessionID, targetSessionID, deviceSessionID) {
		return false
	}
	if !xiaozhiPhysicalOptionalTargetMatch(device.LastTraceID, targetTraceID) ||
		!xiaozhiPhysicalTargetSessionMatch(device.LastSessionID, targetSessionID, deviceSessionID) ||
		!xiaozhiPhysicalTraceHasTargetSession(trace, targetSessionID) {
		return false
	}
	for _, event := range trace.Events {
		if !xiaozhiPhysicalOptionalTargetMatch(event.DeviceID, targetDeviceID) ||
			!xiaozhiPhysicalOptionalTargetMatch(event.TraceID, targetTraceID) ||
			!xiaozhiPhysicalTargetSessionMatch(event.SessionID, targetSessionID, deviceSessionID) {
			return false
		}
	}
	for _, frame := range audioRecent.Frames {
		if !xiaozhiPhysicalOptionalTargetMatch(frame.DeviceID, targetDeviceID) ||
			!xiaozhiPhysicalOptionalTargetMatch(frame.TraceID, targetTraceID) ||
			!xiaozhiPhysicalTargetSessionMatch(frame.SessionID, targetSessionID, deviceSessionID) {
			return false
		}
	}
	return true
}

func xiaozhiPhysicalDeviceRuntimeSessionID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ""
	}
	return "a21-session-" + strings.ReplaceAll(deviceID, ":", "-")
}

func xiaozhiPhysicalTargetSessionMatch(value string, target string, deviceRuntimeSession string) bool {
	value = strings.TrimSpace(value)
	target = strings.TrimSpace(target)
	deviceRuntimeSession = strings.TrimSpace(deviceRuntimeSession)
	return value == "" || value == target || (deviceRuntimeSession != "" && value == deviceRuntimeSession)
}

func xiaozhiPhysicalTraceHasTargetSession(trace gateway.TraceResponse, targetSessionID string) bool {
	targetSessionID = strings.TrimSpace(targetSessionID)
	if targetSessionID == "" {
		return false
	}
	for _, event := range trace.Events {
		if strings.TrimSpace(event.SessionID) == targetSessionID {
			return true
		}
	}
	return false
}

func xiaozhiPhysicalOptionalTargetMatch(value string, target string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == target
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
