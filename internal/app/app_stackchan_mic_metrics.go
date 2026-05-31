package app

import (
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

func fetchStackChanMicProbeGatewayMetrics(gatewayBaseURL string) (stackChanMicProbeGatewayMetrics, error) {
	endpoint, _, err := firmwareGatewayEndpoint(gatewayBaseURL, "/metrics", nil)
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	client := http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stackChanMicProbeGatewayMetrics{}, fmt.Errorf("gateway metrics returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return stackChanMicProbeGatewayMetrics{}, err
	}
	return parseStackChanMicProbeGatewayMetrics(string(data))
}
func parseStackChanMicProbeGatewayMetrics(data string) (stackChanMicProbeGatewayMetrics, error) {
	var metrics stackChanMicProbeGatewayMetrics
	seen := map[string]bool{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		name := fields[0]
		metricName := name
		if labelsAt := strings.Index(metricName, "{"); labelsAt >= 0 {
			metricName = metricName[:labelsAt]
		}
		switch metricName {
		case "a21_audio_frame_total":
			metrics.AudioFrameTotal = int(value)
			seen["a21_audio_frame_total"] = true
		case "a21_audio_ingress_frames_total":
			metrics.AudioIngressFramesTotal = int(value)
			seen["a21_audio_ingress_frames_total"] = true
		case "a21_audio_ingress_rms":
			metrics.AudioIngressRMS = value
			seen["a21_audio_ingress_rms"] = true
		case "a21_audio_playback_chunk_total":
			metrics.AudioPlaybackChunkTotal = int(value)
			seen["a21_audio_playback_chunk_total"] = true
		case "a21_vad_detector_decisions_total":
			if strings.Contains(name, `result="speech"`) {
				metrics.VADSpeechTotal += int(value)
				seen["a21_vad_detector_decisions_total_speech"] = true
			}
		}
	}
	for _, required := range []string{
		"a21_audio_frame_total",
		"a21_audio_ingress_frames_total",
		"a21_audio_ingress_rms",
		"a21_audio_playback_chunk_total",
	} {
		if !seen[required] {
			return stackChanMicProbeGatewayMetrics{}, fmt.Errorf("gateway metrics missing %s", required)
		}
	}
	return metrics, nil
}
func validateStackChanMicProbeMetrics(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.GatewayAudioFrameTotal < options.MinFrames {
		report.addFinding("gateway_audio_frames_below_threshold", "Gateway audio frame metric is below threshold")
	}
	if report.GatewayAudioIngressFramesTotal < options.MinFrames {
		report.addFinding("gateway_audio_ingress_frames_below_threshold", "Gateway audio ingress metric is below threshold")
	}
	if report.GatewayAudioIngressRMS < options.MinGatewayRMS {
		report.addFinding("gateway_audio_rms_below_threshold", "Gateway audio ingress RMS is below threshold")
	}
	if report.GatewayAudioPlaybackChunkTotal != 0 {
		report.addFinding("gateway_playback_chunks_present", "Gateway emitted playback chunks during audio-probe-only acceptance")
	}
	if report.GatewayVADSpeechTotal < options.MinGatewayVADSpeech {
		report.addFinding("gateway_vad_speech_below_threshold", "Gateway VAD speech decisions are below threshold")
	}
}
func validateStackChanMicProbeRuntimeDeltas(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.MicFramesCapturedDelta < options.MinFrames {
		report.addFinding("mic_frames_delta_below_threshold", "captured microphone frame delta is below threshold")
	}
	if report.AudioWSSentAudioFramesDelta < options.MinFrames {
		report.addFinding("audio_ws_frames_delta_below_threshold", "sent audio WebSocket frame delta is below threshold")
	}
	if report.MicDriverErrorsDelta != 0 {
		report.addFinding("mic_driver_error_delta_present", "microphone driver error counter changed during probe window")
	}
	if report.MicQueueDroppedFramesDelta != 0 {
		report.addFinding("mic_queue_drop_delta_present", "microphone queue drop counter changed during probe window")
	}
}
func validateStackChanMicProbeMetricDeltas(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.GatewayAudioFrameDelta < options.MinFrames {
		report.addFinding("gateway_audio_frame_delta_below_threshold", "Gateway audio frame delta is below threshold")
	}
	if report.GatewayAudioIngressFramesDelta < options.MinFrames {
		report.addFinding("gateway_audio_ingress_delta_below_threshold", "Gateway audio ingress delta is below threshold")
	}
	if report.GatewayAudioIngressRMS < options.MinGatewayRMS {
		report.addFinding("gateway_audio_rms_below_threshold", "Gateway audio ingress RMS is below threshold")
	}
	if report.GatewayAudioPlaybackChunkDelta != 0 {
		report.addFinding("gateway_playback_chunk_delta_present", "Gateway playback chunk counter changed during audio-probe-only window")
	}
	if report.GatewayVADSpeechDelta < options.MinGatewayVADSpeech {
		report.addFinding("gateway_vad_speech_delta_below_threshold", "Gateway VAD speech decision delta is below threshold")
	}
}
func populateStackChanMicProbeWindowQuality(report *stackChanMicProbeAcceptanceReport) {
	elapsedMS := report.WindowEndedAtMS - report.WindowStartedAtMS
	if elapsedMS > 0 {
		report.MicCaptureRateHz = roundedStackChanDiagnosticValue(float64(report.MicFramesCapturedDelta) * 1000 / float64(elapsedMS))
		report.AudioWSSentRateHz = roundedStackChanDiagnosticValue(float64(report.AudioWSSentAudioFramesDelta) * 1000 / float64(elapsedMS))
		report.GatewayAudioIngressRateHz = roundedStackChanDiagnosticValue(float64(report.GatewayAudioIngressFramesDelta) * 1000 / float64(elapsedMS))
	}
	report.AudioWSDeliveryRatio = roundedStackChanDiagnosticRatio(report.AudioWSSentAudioFramesDelta, report.MicFramesCapturedDelta)
	report.GatewayIngressDeliveryRatio = roundedStackChanDiagnosticRatio(report.GatewayAudioIngressFramesDelta, report.AudioWSSentAudioFramesDelta)
}
func validateStackChanMicProbeDeliveryRatios(report *stackChanMicProbeAcceptanceReport, options stackChanMicProbeAcceptanceOptions) {
	if report.MicFramesCapturedDelta > 0 && report.AudioWSDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("audio_ws_delivery_ratio_below_threshold", "audio WebSocket sent frame ratio is below captured microphone frames")
	}
	if report.AudioWSSentAudioFramesDelta > 0 && report.GatewayIngressDeliveryRatio < options.MinDeliveryRatio {
		report.addFinding("gateway_ingress_delivery_ratio_below_threshold", "Gateway ingress frame ratio is below audio WebSocket sent frames")
	}
}
func writeStackChanMicProbeAcceptanceReport(outputDir string, report stackChanMicProbeAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-mic-probe-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanMicProbeAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanMicProbeAcceptance(writer io.Writer, report stackChanMicProbeAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
