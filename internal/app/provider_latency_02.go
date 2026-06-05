package app

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func providerLatencyHostLoopbackAudioQualityFinding(source string) providerLatencyBenchFinding {
	switch source {
	case "local_ack":
		return providerLatencyBenchFinding{
			Code:    "host_loopback_local_ack_audio_quality_failed",
			Message: "host-loopback local acknowledgement audio quality guard reported a warning or failure",
		}
	case "tts":
		return providerLatencyBenchFinding{
			Code:    "host_loopback_tts_audio_quality_failed",
			Message: "host-loopback TTS audio quality guard reported a warning or failure",
		}
	default:
		return providerLatencyBenchFinding{
			Code:    "host_loopback_audio_quality_failed",
			Message: "host-loopback audio quality guard reported a warning or failure",
		}
	}
}

func providerLatencyHostLoopbackExecution(source providerLatencyBenchExecution) providerLatencyBenchExecution {
	execution := providerLatencyBenchExecution{
		ProviderExecuted:           false,
		V21Executed:                false,
		HardwareExecuted:           false,
		VoicePipelineObserved:      source.VoicePipelineObserved,
		VoicePipelineExecutionMode: providerLatencySafeExecutionMode(source.VoicePipelineExecutionMode),
		ASRProfile:                 providerLatencySafeIdentifier(source.ASRProfile, false),
		ASRProfileEnv:              providerLatencySafeIdentifier(source.ASRProfileEnv, true),
		LLMProfile:                 providerLatencySafeIdentifier(source.LLMProfile, false),
		LLMProfileEnv:              providerLatencySafeIdentifier(source.LLMProfileEnv, true),
		TTSProfile:                 providerLatencySafeIdentifier(source.TTSProfile, false),
		TTSProfileEnv:              providerLatencySafeIdentifier(source.TTSProfileEnv, true),
	}
	if execution.VoicePipelineExecutionMode == "" {
		execution.VoicePipelineExecutionMode = "unknown"
	}
	if execution.VoicePipelineExecutionMode == "host_local" {
		execution.HostLocalASRExecuted = source.HostLocalASRExecuted && execution.ASRProfile != ""
		execution.HostLocalTextExecuted = source.HostLocalTextExecuted && execution.LLMProfile != ""
		execution.HostLocalTTSExecuted = source.HostLocalTTSExecuted && execution.TTSProfile != ""
	}
	if execution.VoicePipelineExecutionMode == "cloud_edge" {
		execution.ProviderExecuted = source.ProviderExecuted &&
			execution.ASRProfile != "" &&
			execution.LLMProfile != "" &&
			execution.TTSProfile != ""
	}
	return execution
}

func providerLatencySafeExecutionMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "fixture":
		return "fixture"
	case "host_local":
		return "host_local"
	case "cloud_edge":
		return "cloud_edge"
	case "mixed":
		return "mixed"
	default:
		return "unknown"
	}
}

func providerLatencySafeIdentifier(value string, requireA21Env bool) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 || providerLatencyContainsUnsafeIdentifier(value) {
		return ""
	}
	if requireA21Env && !strings.HasPrefix(value, "A21_") {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return ""
	}
	return value
}

func providerLatencyContainsUnsafeIdentifier(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"x21", "v21", "http://", "https://", "/", "\\", ":", "@", "key", "token", "secret", "proxy", "prompt", "transcript", "output"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func providerLatencyHostLoopbackSamples(report providerLatencyBenchHostLoopbackReport, physicalEvidence bool) []providerLatencyBenchSample {
	if providerLatencyHostLoopbackIsVirtualXiaozhi(report) {
		return providerLatencyVirtualXiaozhiSamples(report)
	}
	samples := make([]providerLatencyBenchSample, 0, len(report.AnswerTurns)+len(report.BargeInTurns)+len(report.Samples))
	for _, trace := range report.Samples {
		samples = append(samples, providerLatencyHostLoopbackSampleFromTrace(len(samples)+1, trace, physicalEvidence))
	}
	for _, turn := range report.AnswerTurns {
		if turn.TraceSummary == nil {
			continue
		}
		sample := providerLatencyHostLoopbackSampleFromTrace(len(samples)+1, *turn.TraceSummary, physicalEvidence)
		if turn.FirstAudioMS != nil && sample.AnswerFirstAudioMS == 0 {
			sample.AnswerFirstAudioMS = *turn.FirstAudioMS
		}
		samples = append(samples, sample)
	}
	for _, turn := range report.BargeInTurns {
		if turn.TraceSummary == nil {
			continue
		}
		samples = append(samples, providerLatencyHostLoopbackSampleFromTrace(len(samples)+1, *turn.TraceSummary, physicalEvidence))
	}
	return samples
}

func providerLatencyVirtualXiaozhiSamples(report providerLatencyBenchHostLoopbackReport) []providerLatencyBenchSample {
	count := len(report.FirstAudioSamplesMS)
	if len(report.AbortStopSamplesMS) > count {
		count = len(report.AbortStopSamplesMS)
	}
	samples := make([]providerLatencyBenchSample, 0, count)
	for index := 0; index < count; index++ {
		sample := providerLatencyBenchSample{
			Index:       index + 1,
			Placeholder: false,
		}
		if index < len(report.FirstAudioSamplesMS) {
			sample.AnswerFirstAudioMS = report.FirstAudioSamplesMS[index]
		}
		if index < len(report.AbortStopSamplesMS) {
			sample.BargeInStopMS = report.AbortStopSamplesMS[index]
			sample.PlaybackStopMS = report.AbortStopSamplesMS[index]
			sample.PlaybackStopDoneMS = report.AbortStopSamplesMS[index]
			sample.bargeInStopObserved = true
			sample.playbackStopObserved = true
			sample.playbackStopDoneObserved = true
		}
		samples = append(samples, sample)
	}
	return samples
}

func providerLatencyHostLoopbackSampleFromTrace(index int, trace providerLatencyBenchHostLoopbackTrace, physicalEvidence bool) providerLatencyBenchSample {
	sample := providerLatencyBenchSample{
		Index:       index,
		Placeholder: false,
	}
	sample.TransportIngressMS = firstProviderLatencyValue(trace.TransportIngressMS, trace.XiaozhiListenToAudioIngressMS)
	sample.CodecDecodeMS = firstProviderLatencyValue(trace.CodecDecodeMS, trace.XiaozhiOpusDecodeMS)
	sample.ASRFirstPartialMS = providerLatencyValue(trace.ASRFirstPartialMS)
	sample.ASRFinalMS = providerLatencyValue(trace.ASRFinalMS)
	sample.LLMFirstContentMS = firstProviderLatencyValue(trace.LLMFirstContentMS, trace.ProviderFirstContentMS)
	sample.ProviderFirstByteMS = providerLatencyValue(trace.ProviderFirstByteMS)
	sample.ProviderFirstContentMS = firstProviderLatencyValue(trace.ProviderFirstContentMS, trace.LLMFirstContentMS)
	sample.TTSFirstAudioMS = providerLatencyValue(trace.TTSFirstAudioMS)
	audioDownlink := firstProviderLatencyValue(trace.AudioDownlinkFirstFrameMS, trace.DownlinkFirstFrameMS)
	sample.DownlinkFirstFrameMS = audioDownlink
	sample.AudioDownlinkFirstFrameMS = audioDownlink
	if physicalEvidence {
		sample.DevicePlaybackStartMS = providerLatencyValue(trace.DevicePlaybackStartMS)
	}
	sample.BargeInDetectedMS = providerLatencyValue(trace.BargeInDetectedMS)
	sample.BargeInStopMS = providerLatencyValue(trace.BargeInStopMS)
	sample.ProviderCancelMS = providerLatencyValue(trace.ProviderCancelMS)
	sample.ProviderCancelDoneMS = providerLatencyValue(trace.ProviderCancelDoneMS)
	sample.PlaybackStopMS = providerLatencyValue(trace.PlaybackStopMS)
	sample.PlaybackStopDoneMS = providerLatencyValue(trace.PlaybackStopDoneMS)
	sample.bargeInDetectedObserved = trace.BargeInDetectedMS != nil
	sample.bargeInStopObserved = trace.BargeInStopMS != nil
	sample.providerCancelObserved = trace.ProviderCancelMS != nil
	sample.providerCancelDoneObserved = trace.ProviderCancelDoneMS != nil
	sample.playbackStopObserved = trace.PlaybackStopMS != nil
	sample.playbackStopDoneObserved = trace.PlaybackStopDoneMS != nil
	sample.AnswerFirstAudioMS = firstProviderLatencyValue(trace.AnswerFirstAudioMS, trace.AnswerFirstAudioTotalMS)
	return sample
}

func firstProviderLatencyValue(values ...*float64) float64 {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return 0
}

func providerLatencyValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func providerLatencyHostLoopbackMissingFindings(summary providerLatencyBenchSummary, physicalEvidence bool) []providerLatencyBenchFinding {
	required := []struct {
		code   string
		series providerLatencyBenchSeries
	}{
		{code: "host_loopback_transport_ingress_unavailable", series: summary.TransportIngressMS},
		{code: "host_loopback_codec_decode_unavailable", series: summary.CodecDecodeMS},
		{code: "host_loopback_asr_first_partial_unavailable", series: summary.ASRFirstPartialMS},
		{code: "host_loopback_asr_final_unavailable", series: summary.ASRFinalMS},
		{code: "host_loopback_llm_first_content_unavailable", series: summary.LLMFirstContentMS},
		{code: "host_loopback_tts_first_audio_unavailable", series: summary.TTSFirstAudioMS},
		{code: "host_loopback_audio_downlink_first_frame_unavailable", series: summary.AudioDownlinkFirstFrameMS},
		{code: "host_loopback_answer_first_audio_unavailable", series: summary.AnswerFirstAudioMS},
		{code: "host_loopback_barge_in_stop_unavailable", series: summary.BargeInStopMS},
	}
	findings := make([]providerLatencyBenchFinding, 0)
	for _, stage := range required {
		if stage.series.Samples == 0 {
			findings = append(findings, providerLatencyBenchFinding{
				Code:    stage.code,
				Message: "host-loopback report is missing a required redacted timing stage",
			})
		}
	}
	if !physicalEvidence || summary.DevicePlaybackStartMS.Samples == 0 {
		findings = append(findings, providerLatencyBenchFinding{
			Code:    "physical_device_playback_start_unavailable",
			Message: "physical StackChan playback evidence is unavailable",
		})
	}
	return findings
}

func providerLatencyHostLoopbackCandidate(summary providerLatencyBenchSummary, physicalEvidence bool, findingCount int) bool {
	if findingCount > 1 {
		return false
	}
	return !physicalEvidence &&
		summary.AnswerFirstAudioMS.Samples >= 3 &&
		summary.BargeInStopMS.Samples >= 3 &&
		summary.AnswerFirstAudioMS.P95MS > 0 &&
		summary.AnswerFirstAudioMS.P95MS < 1500 &&
		summary.BargeInStopMS.P95MS >= 0 &&
		summary.BargeInStopMS.P95MS < 300
}

type providerLatencyBenchFixtureSidecar struct {
	SchemaVersion string                                    `json:"schema_version"`
	Identity      string                                    `json:"identity"`
	Audio         providerLatencyBenchFixtureAudioMetadata  `json:"audio"`
	Sample        providerLatencyBenchFixtureSampleMetadata `json:"sample"`
	Window        providerLatencyBenchFixtureWindowMetadata `json:"window"`
}

func loadProviderLatencyBenchFixtureMetadata(fixturePath string) (*providerLatencyBenchFixtureMetadata, []providerLatencyBenchFinding) {
	if strings.ToLower(filepath.Ext(fixturePath)) != ".json" {
		return nil, nil
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	if len(data) > providerLatencyFixtureSidecarMaxBytes {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	var sidecar providerLatencyBenchFixtureSidecar
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sidecar); err != nil {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	if !validProviderLatencyFixtureSidecar(sidecar) {
		return nil, []providerLatencyBenchFinding{invalidProviderLatencyFixtureFinding()}
	}
	metadata := providerLatencyBenchFixtureMetadata{
		SchemaVersion: providerLatencyFixtureSchemaVersion,
		Identity:      strings.TrimSpace(sidecar.Identity),
		Audio: providerLatencyBenchFixtureAudioMetadata{
			Format:       strings.ToLower(strings.TrimSpace(sidecar.Audio.Format)),
			SampleRateHz: sidecar.Audio.SampleRateHz,
			Channels:     sidecar.Audio.Channels,
			DurationMS:   sidecar.Audio.DurationMS,
		},
		Sample: sidecar.Sample,
		Window: sidecar.Window,
	}
	return &metadata, nil
}

func invalidProviderLatencyFixtureFinding() providerLatencyBenchFinding {
	return providerLatencyBenchFinding{
		Code:    "fixture_sidecar_invalid",
		Message: "fixture metadata sidecar is invalid or unsafe",
	}
}

func validProviderLatencyFixtureSidecar(sidecar providerLatencyBenchFixtureSidecar) bool {
	if sidecar.SchemaVersion != providerLatencyFixtureSchemaVersion {
		return false
	}
	if !safeProviderLatencyFixtureString(sidecar.Identity) || strings.TrimSpace(sidecar.Identity) == "" {
		return false
	}
	if !safeProviderLatencyFixtureString(sidecar.Audio.Format) || strings.TrimSpace(sidecar.Audio.Format) == "" {
		return false
	}
	return sidecar.Audio.SampleRateHz > 0 &&
		sidecar.Audio.Channels > 0 &&
		sidecar.Audio.DurationMS > 0 &&
		sidecar.Sample.SampleCount > 0 &&
		sidecar.Window.WindowMS > 0 &&
		sidecar.Window.WindowCount > 0
}

func safeProviderLatencyFixtureString(value string) bool {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(trimmed, "://") ||
		strings.Contains(trimmed, "/") ||
		strings.Contains(trimmed, "\\") ||
		strings.Contains(trimmed, "..") {
		return false
	}
	return true
}

func providerLatencyFixtureContainsForbiddenKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenProviderLatencyFixtureKey(key) || providerLatencyFixtureContainsForbiddenKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if providerLatencyFixtureContainsForbiddenKey(child) {
				return true
			}
		}
	}
	return false
}

func forbiddenProviderLatencyFixtureKey(key string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "raw_pcm", "raw_audio", "pcm_bytes", "data_base64", "audio_base64", "base64_audio",
		"prompt", "transcript", "provider_output", "reasoning", "credential_values",
		"api_key", "access_token", "token", "full_url", "url", "proxy_url", "local_path", "path":
		return true
	default:
		return false
	}
}

func normalizeProviderLatencyBenchMode(mode string, fixturePath string) string {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		if strings.TrimSpace(fixturePath) != "" {
			return "fixture"
		}
		return "mock"
	}
	switch normalized {
	case "mock", "fixture":
		return normalized
	case "host_loopback", "host-loopback":
		return "host_loopback"
	default:
		return ""
	}
}

func buildProviderLatencyBenchSamples(iterations int, mode string) []providerLatencyBenchSample {
	samples := make([]providerLatencyBenchSample, 0, iterations)
	modeOffset := 0
	switch mode {
	case "fixture":
		modeOffset = 7
	case "host_loopback":
		modeOffset = 3
	}
	for i := 0; i < iterations; i++ {
		n := float64(i + modeOffset)
		samples = append(samples, providerLatencyBenchSample{
			Index:                     i + 1,
			TransportIngressMS:        35 + n,
			CodecDecodeMS:             48 + n,
			ASRFirstPartialMS:         120 + n,
			ASRFinalMS:                170 + n,
			LLMFirstContentMS:         260 + n,
			ProviderFirstByteMS:       210 + n,
			ProviderFirstContentMS:    260 + n,
			TTSFirstAudioMS:           420 + n,
			DownlinkFirstFrameMS:      460 + n,
			AudioDownlinkFirstFrameMS: 460 + n,
			DevicePlaybackStartMS:     540 + n,
			BargeInDetectedMS:         30 + n,
			BargeInStopMS:             180 + n,
			ProviderCancelMS:          40 + n,
			ProviderCancelDoneMS:      40 + n,
			PlaybackStopMS:            180 + n,
			PlaybackStopDoneMS:        180 + n,
			Placeholder:               true,
		})
	}
	return samples
}

func summarizeProviderLatencyBenchSamples(samples []providerLatencyBenchSample) providerLatencyBenchSummary {
	transportIngress := make([]time.Duration, 0, len(samples))
	codecDecode := make([]time.Duration, 0, len(samples))
	asr := make([]time.Duration, 0, len(samples))
	asrFinal := make([]time.Duration, 0, len(samples))
	llmFirstContent := make([]time.Duration, 0, len(samples))
	firstByte := make([]time.Duration, 0, len(samples))
	firstContent := make([]time.Duration, 0, len(samples))
	tts := make([]time.Duration, 0, len(samples))
	downlink := make([]time.Duration, 0, len(samples))
	audioDownlink := make([]time.Duration, 0, len(samples))
	playback := make([]time.Duration, 0, len(samples))
	bargeInDetected := make([]time.Duration, 0, len(samples))
	bargeIn := make([]time.Duration, 0, len(samples))
	cancel := make([]time.Duration, 0, len(samples))
	cancelDone := make([]time.Duration, 0, len(samples))
	playbackStop := make([]time.Duration, 0, len(samples))
	playbackStopDone := make([]time.Duration, 0, len(samples))
	answerFirstAudio := make([]time.Duration, 0, len(samples))
	for _, sample := range samples {
		appendProviderLatencyPositiveMS(&transportIngress, sample.TransportIngressMS)
		appendProviderLatencyPositiveMS(&codecDecode, sample.CodecDecodeMS)
		appendProviderLatencyPositiveMS(&asr, sample.ASRFirstPartialMS)
		appendProviderLatencyPositiveMS(&asrFinal, sample.ASRFinalMS)
		appendProviderLatencyPositiveMS(&llmFirstContent, sample.LLMFirstContentMS)
		appendProviderLatencyPositiveMS(&firstByte, sample.ProviderFirstByteMS)
		appendProviderLatencyPositiveMS(&firstContent, sample.ProviderFirstContentMS)
		appendProviderLatencyPositiveMS(&tts, sample.TTSFirstAudioMS)
		appendProviderLatencyPositiveMS(&downlink, sample.DownlinkFirstFrameMS)
		appendProviderLatencyPositiveMS(&audioDownlink, sample.AudioDownlinkFirstFrameMS)
		appendProviderLatencyPositiveMS(&playback, sample.DevicePlaybackStartMS)
		appendProviderLatencyObservedMS(&bargeInDetected, sample.BargeInDetectedMS, sample.bargeInDetectedObserved)
		appendProviderLatencyObservedMS(&bargeIn, sample.BargeInStopMS, sample.bargeInStopObserved)
		appendProviderLatencyObservedMS(&cancel, sample.ProviderCancelMS, sample.providerCancelObserved)
		appendProviderLatencyObservedMS(&cancelDone, sample.ProviderCancelDoneMS, sample.providerCancelDoneObserved)
		appendProviderLatencyObservedMS(&playbackStop, sample.PlaybackStopMS, sample.playbackStopObserved)
		appendProviderLatencyObservedMS(&playbackStopDone, sample.PlaybackStopDoneMS, sample.playbackStopDoneObserved)
		appendProviderLatencyPositiveMS(&answerFirstAudio, sample.AnswerFirstAudioMS)
	}
	return providerLatencyBenchSummary{
		TransportIngressMS:        providerLatencySeries(transportIngress),
		CodecDecodeMS:             providerLatencySeries(codecDecode),
		ASRFirstPartialMS:         providerLatencySeries(asr),
		ASRFinalMS:                providerLatencySeries(asrFinal),
		LLMFirstContentMS:         providerLatencySeries(llmFirstContent),
		ProviderFirstByteMS:       providerLatencySeries(firstByte),
		ProviderFirstContentMS:    providerLatencySeries(firstContent),
		TTSFirstAudioMS:           providerLatencySeries(tts),
		DownlinkFirstFrameMS:      providerLatencySeries(downlink),
		AudioDownlinkFirstFrameMS: providerLatencySeries(audioDownlink),
		DevicePlaybackStartMS:     providerLatencySeries(playback),
		BargeInDetectedMS:         providerLatencySeries(bargeInDetected),
		BargeInStopMS:             providerLatencySeries(bargeIn),
		ProviderCancelMS:          providerLatencySeries(cancel),
		ProviderCancelDoneMS:      providerLatencySeries(cancelDone),
		PlaybackStopMS:            providerLatencySeries(playbackStop),
		PlaybackStopDoneMS:        providerLatencySeries(playbackStopDone),
		AnswerFirstAudioMS:        providerLatencySeries(answerFirstAudio),
	}
}

func appendProviderLatencyPositiveMS(samples *[]time.Duration, value float64) {
	if value <= 0 {
		return
	}
	*samples = append(*samples, msDuration(value))
}

func appendProviderLatencyObservedMS(samples *[]time.Duration, value float64, observed bool) {
	if observed {
		if value < 0 {
			return
		}
		*samples = append(*samples, msDuration(value))
		return
	}
	appendProviderLatencyPositiveMS(samples, value)
}
