package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/providers"
)

const providerLatencyBenchSchemaVersion = "a21.provider_latency_bench.v1"
const providerLatencyFixtureSchemaVersion = "a21.provider_latency_fixture.v1"
const providerLatencyFixtureSidecarMaxBytes = 64 * 1024
const providerLatencyBenchPlaceholderReason = "report_contract_shape_only_no_provider_v21_gateway_or_hardware_execute"

type providerLatencyBenchReport struct {
	SchemaVersion     string                                  `json:"schema_version"`
	ExecutionMode     string                                  `json:"execution_mode"`
	BaselineScope     string                                  `json:"baseline_scope"`
	Iterations        int                                     `json:"iterations"`
	TraceID           string                                  `json:"trace_id"`
	SessionID         string                                  `json:"session_id"`
	DeviceID          string                                  `json:"device_id"`
	Provider          providerLatencyBenchProvider            `json:"provider"`
	Fixture           *providerLatencyBenchFixture            `json:"fixture,omitempty"`
	Metadata          latencyBenchMetadata                    `json:"metadata"`
	Network           providers.NetworkReport                 `json:"network"`
	MetricTerms       []providerLatencyBenchMetricTerm        `json:"metric_terms"`
	StageAvailability []providerLatencyBenchStageAvailability `json:"stage_availability"`
	Samples           []providerLatencyBenchSample            `json:"samples"`
	Summary           providerLatencyBenchSummary             `json:"summary"`
	CanonicalMetrics  map[string]providerLatencyCanonical     `json:"canonical_metrics"`
	Counts            providerLatencyBenchCounts              `json:"counts"`
	PromotionGate     string                                  `json:"promotion_gate"`
	Execution         providerLatencyBenchExecution           `json:"execution"`
	Redaction         providerLatencyBenchRedaction           `json:"redaction"`
	Findings          []providerLatencyBenchFinding           `json:"findings,omitempty"`
	ReportPath        string                                  `json:"report_path,omitempty"`
}

type providerLatencyBenchProvider struct {
	Profile       string   `json:"profile"`
	Label         string   `json:"label"`
	Family        string   `json:"family"`
	Protocol      string   `json:"protocol"`
	RouteEligible bool     `json:"route_eligible"`
	APIKeyEnv     string   `json:"api_key_env,omitempty"`
	ModelEnv      string   `json:"model_env,omitempty"`
	BaseURLEnv    string   `json:"base_url_env,omitempty"`
	RequiredEnv   []string `json:"required_env,omitempty"`
}

type providerLatencyBenchFixture struct {
	FixtureID string                               `json:"fixture_id"`
	Stored    bool                                 `json:"stored"`
	Metadata  *providerLatencyBenchFixtureMetadata `json:"metadata,omitempty"`
}

type providerLatencyBenchFixtureMetadata struct {
	SchemaVersion string                                    `json:"schema_version"`
	Identity      string                                    `json:"identity"`
	Audio         providerLatencyBenchFixtureAudioMetadata  `json:"audio"`
	Sample        providerLatencyBenchFixtureSampleMetadata `json:"sample"`
	Window        providerLatencyBenchFixtureWindowMetadata `json:"window"`
}

type providerLatencyBenchFixtureAudioMetadata struct {
	Format       string `json:"format"`
	SampleRateHz int    `json:"sample_rate_hz"`
	Channels     int    `json:"channels"`
	DurationMS   int    `json:"duration_ms"`
}

type providerLatencyBenchFixtureSampleMetadata struct {
	SampleCount int `json:"sample_count"`
}

type providerLatencyBenchFixtureWindowMetadata struct {
	WindowMS    int `json:"window_ms"`
	WindowCount int `json:"window_count"`
}

type providerLatencyBenchFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type providerLatencyBenchMetricTerm struct {
	Term            string `json:"term"`
	A21Stage        string `json:"a21_stage"`
	CanonicalMetric string `json:"canonical_metric"`
	Meaning         string `json:"meaning"`
}

type providerLatencyBenchStageAvailability struct {
	Stage             string `json:"stage"`
	Available         bool   `json:"available"`
	Placeholder       bool   `json:"placeholder"`
	PlaceholderReason string `json:"placeholder_reason"`
}

type providerLatencyBenchSample struct {
	Index                     int     `json:"index"`
	ASRFirstPartialMS         float64 `json:"asr_first_partial_ms"`
	ProviderFirstByteMS       float64 `json:"provider_first_byte_ms"`
	ProviderFirstContentMS    float64 `json:"provider_first_content_ms"`
	TTSFirstAudioMS           float64 `json:"tts_first_audio_ms"`
	DownlinkFirstFrameMS      float64 `json:"downlink_first_frame_ms"`
	AudioDownlinkFirstFrameMS float64 `json:"audio_downlink_first_frame_ms"`
	DevicePlaybackStartMS     float64 `json:"device_playback_start_ms"`
	BargeInStopMS             float64 `json:"barge_in_stop_ms"`
	ProviderCancelMS          float64 `json:"provider_cancel_ms"`
	PlaybackStopMS            float64 `json:"playback_stop_ms"`
	Placeholder               bool    `json:"placeholder"`
}

type providerLatencyBenchSummary struct {
	ASRFirstPartialMS         providerLatencyBenchSeries `json:"asr_first_partial_ms"`
	ProviderFirstByteMS       providerLatencyBenchSeries `json:"provider_first_byte_ms"`
	ProviderFirstContentMS    providerLatencyBenchSeries `json:"provider_first_content_ms"`
	TTSFirstAudioMS           providerLatencyBenchSeries `json:"tts_first_audio_ms"`
	DownlinkFirstFrameMS      providerLatencyBenchSeries `json:"downlink_first_frame_ms"`
	AudioDownlinkFirstFrameMS providerLatencyBenchSeries `json:"audio_downlink_first_frame_ms"`
	DevicePlaybackStartMS     providerLatencyBenchSeries `json:"device_playback_start_ms"`
	BargeInStopMS             providerLatencyBenchSeries `json:"barge_in_stop_ms"`
	ProviderCancelMS          providerLatencyBenchSeries `json:"provider_cancel_ms"`
	PlaybackStopMS            providerLatencyBenchSeries `json:"playback_stop_ms"`
}

type providerLatencyBenchSeries struct {
	Samples int     `json:"samples"`
	P50MS   float64 `json:"p50_ms"`
	P95MS   float64 `json:"p95_ms"`
	P99MS   float64 `json:"p99_ms"`
}

type providerLatencyCanonical struct {
	SourceStage       string  `json:"source_stage"`
	Samples           int     `json:"samples"`
	P50MS             float64 `json:"p50_ms"`
	P95MS             float64 `json:"p95_ms"`
	P99MS             float64 `json:"p99_ms"`
	Available         bool    `json:"available"`
	Placeholder       bool    `json:"placeholder"`
	PlaceholderReason string  `json:"placeholder_reason"`
}

type providerLatencyBenchCounts struct {
	FallbackCount int `json:"fallback_count"`
	FailureCount  int `json:"failure_count"`
}

type providerLatencyBenchExecution struct {
	ProviderExecuted bool `json:"provider_executed"`
	V21Executed      bool `json:"v21_executed"`
	HardwareExecuted bool `json:"hardware_executed"`
}

type providerLatencyBenchRedaction struct {
	PayloadsStored         bool `json:"payloads_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
}

func runProviderLatencyBench(args []string, stdout io.Writer, stderr io.Writer) int {
	options := providerLatencyBenchOptions{
		Provider:   "mock",
		Iterations: 5,
	}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-latency-bench [--provider <provider>] [--fixture <path>] [--mode mock|fixture|host_loopback] [--iterations 5] [--output-dir reports]")
			return 0
		case "--provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--provider requires a value")
				return 2
			}
			i++
			options.Provider = args[i]
		case "--fixture":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--fixture requires a value")
				return 2
			}
			i++
			options.FixturePath = args[i]
		case "--mode":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--mode requires a value")
				return 2
			}
			i++
			options.ExecutionMode = args[i]
		case "--iterations":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--iterations requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 || value > 1000 {
				fmt.Fprintln(stderr, "--iterations must be an integer between 1 and 1000")
				return 2
			}
			options.Iterations = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		case "--execute":
			fmt.Fprintln(stderr, "provider-latency-bench does not support --execute; this scaffold is mock/fixture only")
			return 2
		default:
			fmt.Fprintf(stderr, "unknown provider-latency-bench option %q\n", args[i])
			return 2
		}
	}
	report, err := buildProviderLatencyBenchReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "provider latency bench failed: %v\n", err)
		return 1
	}
	if outputDir != "" {
		if err := validateA21ReportDir(outputDir); err != nil {
			fmt.Fprintf(stderr, "provider latency bench report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeProviderLatencyBenchReport(outputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write provider latency bench report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONProviderLatencyBench(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode provider latency bench report: %v\n", err)
		return 1
	}
	return 0
}

type providerLatencyBenchOptions struct {
	Provider      string
	ExecutionMode string
	FixturePath   string
	Iterations    int
}

func buildProviderLatencyBenchReport(options providerLatencyBenchOptions) (providerLatencyBenchReport, error) {
	providerName := strings.TrimSpace(options.Provider)
	if providerName == "" {
		providerName = "mock"
	}
	profile, findings, ok := providers.ProviderProfileByNameFromEnv(os.Environ(), strings.ToLower(providerName))
	if !ok || len(findings) > 0 {
		return providerLatencyBenchReport{}, fmt.Errorf("provider profile is not available for latency bench")
	}
	mode := normalizeProviderLatencyBenchMode(options.ExecutionMode, options.FixturePath)
	if mode == "" {
		return providerLatencyBenchReport{}, fmt.Errorf("--mode must be mock, fixture, or host_loopback")
	}
	iterations := options.Iterations
	if iterations <= 0 {
		iterations = 5
	}
	_, network := providers.NetworkPolicyFromEnv(os.Environ())
	report := providerLatencyBenchReport{
		SchemaVersion: providerLatencyBenchSchemaVersion,
		ExecutionMode: mode,
		BaselineScope: "host_only",
		Iterations:    iterations,
		TraceID:       "a21-trace-provider-latency-bench-000001",
		SessionID:     "a21-session-provider-latency-bench-000001",
		DeviceID:      "none_host_fixture",
		Provider: providerLatencyBenchProvider{
			Profile:       profile.Name,
			Label:         profile.Label,
			Family:        string(profile.Family),
			Protocol:      profile.Protocol,
			RouteEligible: profile.RouteEligible,
			APIKeyEnv:     profile.APIKeyEnv,
			ModelEnv:      profile.ModelEnv,
			BaseURLEnv:    profile.BaseURLEnv,
			RequiredEnv:   append([]string(nil), profile.RequiredEnv...),
		},
		Metadata:      buildLatencyBenchMetadata(),
		Network:       network,
		MetricTerms:   providerLatencyBenchMetricTerms(),
		PromotionGate: "not_production",
		Execution: providerLatencyBenchExecution{
			ProviderExecuted: false,
			V21Executed:      false,
			HardwareExecuted: false,
		},
		Redaction: providerLatencyBenchRedaction{
			PayloadsStored:         false,
			CredentialValuesStored: false,
			FullURLsStored:         false,
			LocalPathsStored:       false,
		},
	}
	if options.FixturePath != "" {
		fixtureMetadata, fixtureFindings := loadProviderLatencyBenchFixtureMetadata(options.FixturePath)
		report.Fixture = &providerLatencyBenchFixture{
			FixtureID: filepath.Base(options.FixturePath),
			Stored:    false,
			Metadata:  fixtureMetadata,
		}
		report.Findings = append(report.Findings, fixtureFindings...)
	}
	report.StageAvailability = buildProviderLatencyBenchStageAvailability()
	report.Samples = buildProviderLatencyBenchSamples(iterations, mode)
	report.Summary = summarizeProviderLatencyBenchSamples(report.Samples)
	report.CanonicalMetrics = buildProviderLatencyCanonicalMetrics(report.Summary)
	report.Counts.FailureCount = len(report.Findings)
	return report, nil
}

func providerLatencyBenchMetricTerms() []providerLatencyBenchMetricTerm {
	return []providerLatencyBenchMetricTerm{
		{
			Term:            "TTFS",
			A21Stage:        "asr_first_partial_ms",
			CanonicalMetric: "speech_end_to_final_asr_ms",
			Meaning:         "speech end to first or final ASR signal; this scaffold records first partial placeholder only",
		},
		{
			Term:            "TTFT",
			A21Stage:        "provider_first_content_ms",
			CanonicalMetric: "llm_request_to_first_token_ms",
			Meaning:         "text provider request to first content token",
		},
		{
			Term:            "FTTS",
			A21Stage:        "tts_first_audio_ms",
			CanonicalMetric: "first_llm_token_to_first_tts_audio_ms",
			Meaning:         "first provider content to first playable TTS audio",
		},
		{
			Term:            "TTFA",
			A21Stage:        "tts_first_audio_ms",
			CanonicalMetric: "tts_request_to_first_audio_ms",
			Meaning:         "TTS request to first playable audio",
		},
	}
}

func buildProviderLatencyBenchStageAvailability() []providerLatencyBenchStageAvailability {
	stages := []string{
		"asr_first_partial_ms",
		"provider_first_byte_ms",
		"provider_first_content_ms",
		"tts_first_audio_ms",
		"audio_downlink_first_frame_ms",
		"device_playback_start_ms",
		"barge_in_stop_ms",
		"provider_cancel_ms",
		"playback_stop_ms",
	}
	availability := make([]providerLatencyBenchStageAvailability, 0, len(stages))
	for _, stage := range stages {
		availability = append(availability, providerLatencyBenchStageAvailability{
			Stage:             stage,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: providerLatencyBenchPlaceholderReason,
		})
	}
	return availability
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
			ASRFirstPartialMS:         120 + n,
			ProviderFirstByteMS:       210 + n,
			ProviderFirstContentMS:    260 + n,
			TTSFirstAudioMS:           420 + n,
			DownlinkFirstFrameMS:      460 + n,
			AudioDownlinkFirstFrameMS: 460 + n,
			DevicePlaybackStartMS:     540 + n,
			BargeInStopMS:             180 + n,
			ProviderCancelMS:          40 + n,
			PlaybackStopMS:            180 + n,
			Placeholder:               true,
		})
	}
	return samples
}

func summarizeProviderLatencyBenchSamples(samples []providerLatencyBenchSample) providerLatencyBenchSummary {
	asr := make([]time.Duration, 0, len(samples))
	firstByte := make([]time.Duration, 0, len(samples))
	firstContent := make([]time.Duration, 0, len(samples))
	tts := make([]time.Duration, 0, len(samples))
	downlink := make([]time.Duration, 0, len(samples))
	audioDownlink := make([]time.Duration, 0, len(samples))
	playback := make([]time.Duration, 0, len(samples))
	bargeIn := make([]time.Duration, 0, len(samples))
	cancel := make([]time.Duration, 0, len(samples))
	playbackStop := make([]time.Duration, 0, len(samples))
	for _, sample := range samples {
		asr = append(asr, msDuration(sample.ASRFirstPartialMS))
		firstByte = append(firstByte, msDuration(sample.ProviderFirstByteMS))
		firstContent = append(firstContent, msDuration(sample.ProviderFirstContentMS))
		tts = append(tts, msDuration(sample.TTSFirstAudioMS))
		downlink = append(downlink, msDuration(sample.DownlinkFirstFrameMS))
		audioDownlink = append(audioDownlink, msDuration(sample.AudioDownlinkFirstFrameMS))
		playback = append(playback, msDuration(sample.DevicePlaybackStartMS))
		bargeIn = append(bargeIn, msDuration(sample.BargeInStopMS))
		cancel = append(cancel, msDuration(sample.ProviderCancelMS))
		playbackStop = append(playbackStop, msDuration(sample.PlaybackStopMS))
	}
	return providerLatencyBenchSummary{
		ASRFirstPartialMS:         providerLatencySeries(asr),
		ProviderFirstByteMS:       providerLatencySeries(firstByte),
		ProviderFirstContentMS:    providerLatencySeries(firstContent),
		TTSFirstAudioMS:           providerLatencySeries(tts),
		DownlinkFirstFrameMS:      providerLatencySeries(downlink),
		AudioDownlinkFirstFrameMS: providerLatencySeries(audioDownlink),
		DevicePlaybackStartMS:     providerLatencySeries(playback),
		BargeInStopMS:             providerLatencySeries(bargeIn),
		ProviderCancelMS:          providerLatencySeries(cancel),
		PlaybackStopMS:            providerLatencySeries(playbackStop),
	}
}

func buildProviderLatencyCanonicalMetrics(summary providerLatencyBenchSummary) map[string]providerLatencyCanonical {
	return map[string]providerLatencyCanonical{
		"asr_first_partial_ms": providerLatencyCanonicalFromSeries(
			"asr_first_partial_ms",
			summary.ASRFirstPartialMS,
		),
		"provider_first_byte_ms": providerLatencyCanonicalFromSeries(
			"provider_first_byte_ms",
			summary.ProviderFirstByteMS,
		),
		"provider_first_content_ms": providerLatencyCanonicalFromSeries(
			"provider_first_content_ms",
			summary.ProviderFirstContentMS,
		),
		"tts_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
		),
		"audio_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
		),
		"device_playback_start_ms": providerLatencyCanonicalFromSeries(
			"device_playback_start_ms",
			summary.DevicePlaybackStartMS,
		),
		"barge_in_stop_ms": providerLatencyCanonicalFromSeries(
			"barge_in_stop_ms",
			summary.BargeInStopMS,
		),
		"provider_cancel_ms": providerLatencyCanonicalFromSeries(
			"provider_cancel_ms",
			summary.ProviderCancelMS,
		),
		"playback_stop_ms": providerLatencyCanonicalFromSeries(
			"playback_stop_ms",
			summary.PlaybackStopMS,
		),
		"speech_end_to_final_asr_ms": providerLatencyCanonicalFromSeries(
			"asr_first_partial_ms",
			summary.ASRFirstPartialMS,
		),
		"speech_end_to_first_llm_token_ms": providerLatencyCanonicalFromSeries(
			"provider_first_content_ms",
			summary.ProviderFirstContentMS,
		),
		"llm_request_to_first_token_ms": providerLatencyCanonicalFromSeries(
			"provider_first_content_ms",
			summary.ProviderFirstContentMS,
		),
		"first_llm_token_to_first_tts_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
		),
		"tts_request_to_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
		),
		"provider_commit_to_first_audio_ms": providerLatencyCanonicalFromSeries(
			"tts_first_audio_ms",
			summary.TTSFirstAudioMS,
		),
		"gateway_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
		),
		"device_downlink_first_frame_ms": providerLatencyCanonicalFromSeries(
			"audio_downlink_first_frame_ms",
			summary.AudioDownlinkFirstFrameMS,
		),
		"speech_end_to_first_audible_response_ms": providerLatencyCanonicalFromSeries(
			"device_playback_start_ms",
			summary.DevicePlaybackStartMS,
		),
	}
}

func providerLatencyCanonicalFromSeries(sourceStage string, series providerLatencyBenchSeries) providerLatencyCanonical {
	return providerLatencyCanonical{
		SourceStage:       sourceStage,
		Samples:           series.Samples,
		P50MS:             series.P50MS,
		P95MS:             series.P95MS,
		P99MS:             series.P99MS,
		Available:         false,
		Placeholder:       true,
		PlaceholderReason: providerLatencyBenchPlaceholderReason,
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
	report.ReportPath = reportPath
	if err := writeJSONProviderLatencyBench(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONProviderLatencyBench(writer io.Writer, report providerLatencyBenchReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
