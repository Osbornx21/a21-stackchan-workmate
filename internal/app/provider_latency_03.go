package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

func buildProviderLatencyCanonicalMetrics(summary providerLatencyBenchSummary, placeholder bool, physicalEvidence bool) map[string]providerLatencyCanonical {
	return map[string]providerLatencyCanonical{
		"transport_ingress_ms": providerLatencyCanonicalFromSeries(
			"transport_ingress_ms",
			summary.TransportIngressMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"codec_decode_ms": providerLatencyCanonicalFromSeries(
			"codec_decode_ms",
			summary.CodecDecodeMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"asr_first_partial_ms": providerLatencyCanonicalFromSeries(
			"asr_first_partial_ms",
			summary.ASRFirstPartialMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"asr_final_ms": providerLatencyCanonicalFromSeries(
			"asr_final_ms",
			summary.ASRFinalMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"llm_first_content_ms": providerLatencyCanonicalFromSeries(
			"llm_first_content_ms",
			summary.LLMFirstContentMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"provider_first_byte_ms": providerLatencyCanonicalFromSeries(
			"provider_first_byte_ms",
			summary.ProviderFirstByteMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"provider_first_content_ms": providerLatencyCanonicalFromSeries(
			"provider_first_content_ms",
			summary.ProviderFirstContentMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"tts_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"downlink_first_frame_ms",
			summary.DownlinkFirstFrameMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"audio_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"device_playback_start_ms": providerLatencyCanonicalFromSeries(
			"device_playback_start_ms",
			summary.DevicePlaybackStartMS,
			placeholder,
			true,
			physicalEvidence,
		),
		"barge_in_detected_ms": providerLatencyCanonicalFromSeries(
			"barge_in_detected_ms",
			summary.BargeInDetectedMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"barge_in_stop_ms": providerLatencyCanonicalFromSeries(
			"barge_in_stop_ms",
			summary.BargeInStopMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"provider_cancel_ms": providerLatencyCanonicalFromSeries(
			"provider_cancel_ms",
			summary.ProviderCancelMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"provider_cancel_done_ms": providerLatencyCanonicalFromSeries(
			"provider_cancel_done_ms",
			summary.ProviderCancelDoneMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"playback_stop_ms": providerLatencyCanonicalFromSeries(
			"playback_stop_ms",
			summary.PlaybackStopMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"playback_stop_done_ms": providerLatencyCanonicalFromSeries(
			"playback_stop_done_ms",
			summary.PlaybackStopDoneMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"answer_first_audio_ms": providerLatencyCanonicalFromSeries(
			"answer_first_audio_ms",
			summary.AnswerFirstAudioMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"speech_end_to_final_asr_ms": providerLatencyCanonicalFromSeries(
			"asr_final_ms",
			summary.ASRFinalMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"speech_end_to_first_llm_token_ms": providerLatencyCanonicalFromSeries(
			"llm_first_content_ms",
			summary.LLMFirstContentMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"llm_request_to_first_token_ms": providerLatencyCanonicalFromSeries(
			"llm_first_content_ms",
			summary.LLMFirstContentMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"first_llm_token_to_first_tts_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"tts_request_to_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"provider_commit_to_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"gateway_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
			placeholder,
			false,
			physicalEvidence,
		),
		"device_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
			placeholder,
			true,
			physicalEvidence,
		),
		"speech_end_to_first_audible_response_ms": providerLatencyCanonicalFromSeries(
			"device_playback_start_ms",
			summary.DevicePlaybackStartMS,
			placeholder,
			true,
			physicalEvidence,
		),
		"answer_first_audio_p95_ms": providerLatencyCanonicalFromP95(
			"answer_first_audio_ms",
			summary.AnswerFirstAudioMS,
			placeholder,
		),
		"barge_in_stop_p95_ms": providerLatencyCanonicalFromP95(
			"barge_in_stop_ms",
			summary.BargeInStopMS,
			placeholder,
		),
	}
}

func providerLatencyCanonicalFromSeries(sourceStage string, series providerLatencyBenchSeries, placeholder bool, physicalOnly bool, physicalEvidence bool) providerLatencyCanonical {
	available := !placeholder && series.Samples > 0
	if physicalOnly && !physicalEvidence {
		available = false
	}
	return providerLatencyCanonical{
		SourceStage:       sourceStage,
		Samples:           series.Samples,
		P50MS:             series.P50MS,
		P95MS:             series.P95MS,
		P99MS:             series.P99MS,
		Available:         available,
		Placeholder:       placeholder,
		PlaceholderReason: providerLatencyPlaceholderReason(placeholder),
	}
}

func providerLatencyCanonicalFromP95(sourceStage string, series providerLatencyBenchSeries, placeholder bool) providerLatencyCanonical {
	available := !placeholder && series.Samples >= 3
	return providerLatencyCanonical{
		SourceStage:       sourceStage,
		Samples:           series.Samples,
		P50MS:             series.P95MS,
		P95MS:             series.P95MS,
		P99MS:             series.P95MS,
		Available:         available,
		Placeholder:       placeholder,
		PlaceholderReason: providerLatencyPlaceholderReason(placeholder),
	}
}

func msDuration(value float64) time.Duration {
	return time.Duration(value * float64(time.Millisecond))
}

func providerLatencySeries(samples []time.Duration) providerLatencyBenchSeries {
	return providerLatencyBenchSeries{
		Samples: len(samples),
		P50MS:   percentileMS(samples, 0.50),
		P95MS:   percentileMS(samples, 0.95),
		P99MS:   percentileMS(samples, 0.99),
	}
}

func writeProviderLatencyBenchReport(outputDir string, report providerLatencyBenchReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-provider-latency-bench-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONProviderLatencyBench(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONProviderLatencyBench(writer io.Writer, report providerLatencyBenchReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
