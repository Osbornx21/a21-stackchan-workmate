package app

import (
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var requiredStackChanCapabilities = []string{
	"microphone",
	"speaker",
	"screen",
	"screen_touch",
	"top_touch",
	"servo_y",
	"rgb",
}

func isRequiredStackChanCapability(capability string) bool {
	for _, required := range requiredStackChanCapabilities {
		if capability == required {
			return true
		}
	}
	return false
}

func runStackChanAccept(args []string, stdout io.Writer, stderr io.Writer) int {
	var check string
	passThrough := make([]string, 0, len(args))
	showHelp := len(args) == 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			if check == "" {
				showHelp = true
			} else {
				passThrough = append(passThrough, args[i])
			}
		case "--check":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--check requires a value")
				return 2
			}
			i++
			check = args[i]
		default:
			passThrough = append(passThrough, args[i])
		}
	}
	if check == "" {
		if showHelp {
			fmt.Fprintln(stdout, "a21 stackchan-accept --check identity|physical-evidence|capability|mic-probe|imu-probe|sensor-probe|half-duplex|xiaozhi-half-duplex|xiaozhi-prd-review|speaker|touch|hardware-mainline [check options]")
			return 0
		}
		fmt.Fprintln(stderr, "--check requires a value")
		return 2
	}
	return dispatchStackChanAccept(check, passThrough, stdout, stderr)
}

func runDeprecatedStackChanAcceptAlias(args []string, stdout io.Writer, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	check, ok := stackChanAcceptAliasCheck(args[0])
	if !ok {
		return 0, false
	}
	return dispatchStackChanAccept(check, args[1:], stdout, stderr), true
}

func stackChanAcceptAliasCheck(command string) (string, bool) {
	switch command {
	case "stackchan-identity-acceptance":
		return "identity", true
	case "stackchan-physical-evidence":
		return "physical-evidence", true
	case "stackchan-capability-acceptance":
		return "capability", true
	case "stackchan-mic-probe-acceptance":
		return "mic-probe", true
	case "stackchan-imu-probe-acceptance":
		return "imu-probe", true
	case "stackchan-sensor-probe-acceptance":
		return "sensor-probe", true
	case "stackchan-half-duplex-acceptance":
		return "half-duplex", true
	case "xiaozhi-half-duplex-acceptance", "stackchan-xiaozhi-half-duplex-acceptance":
		return "xiaozhi-half-duplex", true
	case "stackchan-speaker-acceptance":
		return "speaker", true
	case "stackchan-touch-acceptance":
		return "touch", true
	case "stackchan-hardware-mainline":
		return "hardware-mainline", true
	default:
		return "", false
	}
}

func dispatchStackChanAccept(check string, args []string, stdout io.Writer, stderr io.Writer) int {
	switch normalizeStackChanAcceptCheck(check) {
	case "identity":
		return runStackChanIdentityAcceptance(args, stdout, stderr)
	case "physical-evidence", "physical":
		return runStackChanPhysicalEvidence(args, stdout, stderr)
	case "capability":
		return runStackChanCapabilityAcceptance(args, stdout, stderr)
	case "mic-probe", "microphone":
		return runStackChanMicProbeAcceptance(args, stdout, stderr)
	case "imu-probe", "imu":
		return runStackChanIMUProbeAcceptance(args, stdout, stderr)
	case "sensor-probe", "sensor", "sensors":
		return runStackChanSensorProbeAcceptance(args, stdout, stderr)
	case "half-duplex":
		return runStackChanHalfDuplexAcceptance(args, stdout, stderr)
	case "xiaozhi-half-duplex", "stock-half-duplex", "xiaozhi-physical-half-duplex":
		return runXiaozhiHalfDuplexAcceptance(args, stdout, stderr)
	case "xiaozhi-prd-review", "xiaozhi-physical-prd-review":
		return runXiaozhiPhysicalPRDReview(args, stdout, stderr)
	case "speaker":
		return runStackChanSpeakerAcceptance(args, stdout, stderr)
	case "touch":
		return runStackChanTouchAcceptance(args, stdout, stderr)
	case "hardware-mainline", "mainline":
		return runStackChanHardwareMainline(args, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown stackchan acceptance check %q\n", check)
		return 2
	}
}

func normalizeStackChanAcceptCheck(check string) string {
	normalized := strings.ToLower(strings.TrimSpace(check))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.TrimPrefix(normalized, "stackchan-")
	normalized = strings.TrimSuffix(normalized, "-acceptance")
	return normalized
}

func stackChanRuntimeEchoInt(findings *[]officePreflightFinding, runtimeEcho map[string]string, key string) int {
	value := strings.TrimSpace(runtimeEcho[key])
	if value == "" {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_missing", Message: "runtime echo missing " + key})
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_invalid", Message: "runtime echo value is not an integer for " + key})
		return 0
	}
	return parsed
}
func roundedStackChanDiagnosticRatio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return roundedStackChanDiagnosticValue(float64(numerator) / float64(denominator))
}
func roundedStackChanDiagnosticValue(value float64) float64 {
	return math.Round(value*1000) / 1000
}
func stackChanAbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func stackChanRuntimeEchoString(findings *[]officePreflightFinding, runtimeEcho map[string]string, key string) string {
	value := strings.TrimSpace(runtimeEcho[key])
	if value == "" {
		*findings = append(*findings, officePreflightFinding{Code: "runtime_echo_missing", Message: "runtime echo missing " + key})
	}
	return value
}
func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
func postStackChanMicProbeControl(gatewayBaseURL string, deviceID string, state protocol.ExpressionState, mode protocol.Mode, text string, traceID string, sessionID string, audioProbeOnly bool, mockPlaybackOnNextAudioFrame bool) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:                     deviceID,
		State:                        state,
		Mode:                         mode,
		Text:                         text,
		TraceID:                      traceID,
		SessionID:                    sessionID,
		AudioProbeOnly:               audioProbeOnly,
		MockPlaybackOnNextAudioFrame: mockPlaybackOnNextAudioFrame,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := *a21DirectHTTPClient(3 * time.Second)
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}
func postStackChanSpeakerControl(gatewayBaseURL string, deviceID string, state protocol.ExpressionState, mode protocol.Mode, text string, traceID string, sessionID string, streamID string, mockAudioChunks int) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	mockAudioChunksPtr := mockAudioChunks
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           state,
		Mode:            mode,
		Text:            text,
		TraceID:         traceID,
		SessionID:       sessionID,
		StreamID:        streamID,
		MockAudioChunks: &mockAudioChunksPtr,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := *a21DirectHTTPClient(3 * time.Second)
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}
func postStackChanAudioPlaybackBatch(gatewayBaseURL string, deviceID string, traceID string, sessionID string, streamID string, chunks []protocol.AudioPlaybackChunk) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	request := gateway.DeviceControlRequest{
		DeviceID:    deviceID,
		State:       protocol.ExpressionSpeaking,
		Mode:        protocol.ModeWorkmate,
		Text:        "A21 LOCAL TTS",
		TraceID:     traceID,
		SessionID:   sessionID,
		StreamID:    streamID,
		AudioChunks: chunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := *a21DirectHTTPClient(15 * time.Second)
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}
func postStackChanSpeakerProbeBatches(gatewayBaseURL string, deviceID string, traceID string, sessionID string, streamID string, totalChunks int) (gateway.DeviceControlResponse, error) {
	remaining := totalChunks
	var firstControl gateway.DeviceControlResponse
	for remaining > 0 {
		consumedChunks := stackChanSpeakerProbeBatchChunks
		if remaining < consumedChunks {
			consumedChunks = remaining
		}
		control, err := postStackChanSpeakerControl(gatewayBaseURL, deviceID, protocol.ExpressionSpeaking, protocol.ModeWorkmate, "SPEAKER PROBE", traceID, sessionID, streamID, stackChanSpeakerProbeBatchChunks)
		if err != nil {
			return firstControl, err
		}
		if firstControl.TraceID == "" {
			firstControl = control
		}
		remaining -= consumedChunks
		if remaining > 0 {
			time.Sleep(time.Duration(stackChanSpeakerProbeBatchChunks*stackChanSpeakerProbeChunkDurationMS) * time.Millisecond)
		}
	}
	return firstControl, nil
}
func postStackChanTouchControl(gatewayBaseURL string, deviceID string, spec stackChanTouchCaseSpec) (gateway.DeviceControlResponse, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/v1/devices/control", nil)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	mockAudioChunksPtr := spec.MockAudioChunks
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           spec.ControlState,
		Mode:            spec.ControlMode,
		Text:            spec.DevicePrompt,
		TraceID:         "a21-trace-touch-acceptance-" + spec.Name,
		SessionID:       "a21-session-touch-acceptance",
		StreamID:        spec.StreamID,
		MockAudioChunks: &mockAudioChunksPtr,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := *a21DirectHTTPClient(3 * time.Second)
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response gateway.DeviceControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	if containsLegacyIdentity(response.TraceID) || containsLegacyIdentity(response.SessionID) || containsLegacyIdentity(response.DeviceID) {
		return gateway.DeviceControlResponse{}, fmt.Errorf("gateway device control contains forbidden legacy identity")
	}
	return response, nil
}
func candidateStackChanTouchTraceIDs(lastTraceID string) []string {
	lastTraceID = strings.TrimSpace(lastTraceID)
	if lastTraceID == "" {
		return nil
	}
	ids := []string{lastTraceID}
	const prefix = "a21-trace-device-"
	if !strings.HasPrefix(lastTraceID, prefix) {
		return ids
	}
	suffix := strings.TrimPrefix(lastTraceID, prefix)
	value, err := strconv.Atoi(suffix)
	if err != nil {
		return ids
	}
	for candidate := value - 1; candidate >= 0 && candidate >= value-6; candidate-- {
		ids = append(ids, fmt.Sprintf("%s%0*d", prefix, len(suffix), candidate))
	}
	return ids
}
