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
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
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
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           state,
		Mode:            mode,
		Text:            text,
		TraceID:         traceID,
		SessionID:       sessionID,
		StreamID:        streamID,
		MockAudioChunks: mockAudioChunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
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
	client := http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
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
	request := gateway.DeviceControlRequest{
		DeviceID:        deviceID,
		State:           spec.ControlState,
		Mode:            spec.ControlMode,
		Text:            spec.DevicePrompt,
		TraceID:         "a21-trace-touch-acceptance-" + spec.Name,
		SessionID:       "a21-session-touch-acceptance",
		StreamID:        spec.StreamID,
		MockAudioChunks: spec.MockAudioChunks,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return gateway.DeviceControlResponse{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
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
