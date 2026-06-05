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
	AcceptanceStatus  string                                  `json:"acceptance_status"`
	PRDAccepted       bool                                    `json:"prd_accepted"`
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
	Stage             string  `json:"stage"`
	SourceTraceMarker string  `json:"source_trace_marker"`
	Samples           int     `json:"samples"`
	P50MS             float64 `json:"p50_ms"`
	P95MS             float64 `json:"p95_ms"`
	P99MS             float64 `json:"p99_ms"`
	Available         bool    `json:"available"`
	Placeholder       bool    `json:"placeholder"`
	PlaceholderReason string  `json:"placeholder_reason"`
}

type providerLatencyBenchSample struct {
	Index                      int     `json:"index"`
	TransportIngressMS         float64 `json:"transport_ingress_ms"`
	CodecDecodeMS              float64 `json:"codec_decode_ms"`
	ASRFirstPartialMS          float64 `json:"asr_first_partial_ms"`
	ASRFinalMS                 float64 `json:"asr_final_ms"`
	LLMFirstContentMS          float64 `json:"llm_first_content_ms"`
	ProviderFirstByteMS        float64 `json:"provider_first_byte_ms"`
	ProviderFirstContentMS     float64 `json:"provider_first_content_ms"`
	TTSFirstAudioMS            float64 `json:"tts_first_audio_ms"`
	DownlinkFirstFrameMS       float64 `json:"downlink_first_frame_ms"`
	AudioDownlinkFirstFrameMS  float64 `json:"audio_downlink_first_frame_ms"`
	DevicePlaybackStartMS      float64 `json:"device_playback_start_ms"`
	BargeInDetectedMS          float64 `json:"barge_in_detected_ms"`
	BargeInStopMS              float64 `json:"barge_in_stop_ms"`
	ProviderCancelMS           float64 `json:"provider_cancel_ms"`
	ProviderCancelDoneMS       float64 `json:"provider_cancel_done_ms"`
	PlaybackStopMS             float64 `json:"playback_stop_ms"`
	PlaybackStopDoneMS         float64 `json:"playback_stop_done_ms"`
	AnswerFirstAudioMS         float64 `json:"answer_first_audio_ms"`
	Placeholder                bool    `json:"placeholder"`
	bargeInDetectedObserved    bool
	bargeInStopObserved        bool
	providerCancelObserved     bool
	providerCancelDoneObserved bool
	playbackStopObserved       bool
	playbackStopDoneObserved   bool
}

type providerLatencyBenchSummary struct {
	TransportIngressMS        providerLatencyBenchSeries `json:"transport_ingress_ms"`
	CodecDecodeMS             providerLatencyBenchSeries `json:"codec_decode_ms"`
	ASRFirstPartialMS         providerLatencyBenchSeries `json:"asr_first_partial_ms"`
	ASRFinalMS                providerLatencyBenchSeries `json:"asr_final_ms"`
	LLMFirstContentMS         providerLatencyBenchSeries `json:"llm_first_content_ms"`
	ProviderFirstByteMS       providerLatencyBenchSeries `json:"provider_first_byte_ms"`
	ProviderFirstContentMS    providerLatencyBenchSeries `json:"provider_first_content_ms"`
	TTSFirstAudioMS           providerLatencyBenchSeries `json:"tts_first_audio_ms"`
	DownlinkFirstFrameMS      providerLatencyBenchSeries `json:"downlink_first_frame_ms"`
	AudioDownlinkFirstFrameMS providerLatencyBenchSeries `json:"audio_downlink_first_frame_ms"`
	DevicePlaybackStartMS     providerLatencyBenchSeries `json:"device_playback_start_ms"`
	BargeInDetectedMS         providerLatencyBenchSeries `json:"barge_in_detected_ms"`
	BargeInStopMS             providerLatencyBenchSeries `json:"barge_in_stop_ms"`
	ProviderCancelMS          providerLatencyBenchSeries `json:"provider_cancel_ms"`
	ProviderCancelDoneMS      providerLatencyBenchSeries `json:"provider_cancel_done_ms"`
	PlaybackStopMS            providerLatencyBenchSeries `json:"playback_stop_ms"`
	PlaybackStopDoneMS        providerLatencyBenchSeries `json:"playback_stop_done_ms"`
	AnswerFirstAudioMS        providerLatencyBenchSeries `json:"answer_first_audio_ms"`
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
	ProviderExecuted           bool   `json:"provider_executed"`
	V21Executed                bool   `json:"v21_executed"`
	HardwareExecuted           bool   `json:"hardware_executed"`
	VoicePipelineObserved      bool   `json:"voice_pipeline_observed"`
	VoicePipelineExecutionMode string `json:"voice_pipeline_execution_mode"`
	ASRProfile                 string `json:"asr_profile,omitempty"`
	ASRProfileEnv              string `json:"asr_profile_env,omitempty"`
	LLMProfile                 string `json:"llm_profile,omitempty"`
	LLMProfileEnv              string `json:"llm_profile_env,omitempty"`
	TTSProfile                 string `json:"tts_profile,omitempty"`
	TTSProfileEnv              string `json:"tts_profile_env,omitempty"`
	HostLocalASRExecuted       bool   `json:"host_local_asr_executed"`
	HostLocalTextExecuted      bool   `json:"host_local_text_executed"`
	HostLocalTTSExecuted       bool   `json:"host_local_tts_executed"`
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
		Metadata:         buildLatencyBenchMetadata(),
		Network:          network,
		MetricTerms:      providerLatencyBenchMetricTerms(),
		PromotionGate:    "not_production",
		AcceptanceStatus: "not_accepted",
		PRDAccepted:      false,
		Execution: providerLatencyBenchExecution{
			ProviderExecuted:           false,
			V21Executed:                false,
			HardwareExecuted:           false,
			VoicePipelineExecutionMode: "unknown",
		},
		Redaction: providerLatencyBenchRedaction{
			PayloadsStored:         false,
			CredentialValuesStored: false,
			FullURLsStored:         false,
			LocalPathsStored:       false,
		},
	}
	hostLoopbackIngested := false
	hostLoopbackPhysicalEvidence := false
	hostLoopbackInvalid := false
	if options.FixturePath != "" {
		report.Fixture = &providerLatencyBenchFixture{
			FixtureID: filepath.Base(options.FixturePath),
			Stored:    false,
		}
		if mode == "host_loopback" {
			evidence, fixtureFindings := loadProviderLatencyBenchHostLoopbackReport(options.FixturePath)
			report.Findings = append(report.Findings, fixtureFindings...)
			if evidence.Valid {
				report.Samples = evidence.Samples
				hostLoopbackIngested = true
				hostLoopbackPhysicalEvidence = evidence.PhysicalEvidence
				report.Execution = providerLatencyHostLoopbackExecution(evidence.Execution)
				report.Findings = append(report.Findings, evidence.AudioQualityFindings...)
				if len(evidence.Samples) > 0 {
					report.Iterations = len(evidence.Samples)
				}
			} else if len(fixtureFindings) > 0 {
				hostLoopbackInvalid = true
			}
		} else {
			fixtureMetadata, fixtureFindings := loadProviderLatencyBenchFixtureMetadata(options.FixturePath)
			report.Fixture.Metadata = fixtureMetadata
			report.Findings = append(report.Findings, fixtureFindings...)
		}
	}
	if len(report.Samples) == 0 && !hostLoopbackInvalid {
		report.Samples = buildProviderLatencyBenchSamples(iterations, mode)
	}
	report.Summary = summarizeProviderLatencyBenchSamples(report.Samples)
	if hostLoopbackIngested {
		report.Findings = append(report.Findings, providerLatencyHostLoopbackMissingFindings(report.Summary, hostLoopbackPhysicalEvidence)...)
		if providerLatencyHostLoopbackCandidate(report.Summary, hostLoopbackPhysicalEvidence, len(report.Findings)) {
			report.AcceptanceStatus = "candidate_host_only"
		}
	}
	placeholder := !hostLoopbackIngested
	report.StageAvailability = buildProviderLatencyBenchStageAvailability(report.Summary, placeholder, hostLoopbackPhysicalEvidence)
	report.CanonicalMetrics = buildProviderLatencyCanonicalMetrics(report.Summary, placeholder, hostLoopbackPhysicalEvidence)
	report.Counts.FailureCount = len(report.Findings)
	return report, nil
}

func providerLatencyBenchMetricTerms() []providerLatencyBenchMetricTerm {
	return []providerLatencyBenchMetricTerm{
		{
			Term:            "TTFS",
			A21Stage:        "asr_final_ms",
			CanonicalMetric: "speech_end_to_final_asr_ms",
			Meaning:         "speech end to final ASR signal; this scaffold records placeholder timing only",
		},
		{
			Term:            "TTFT",
			A21Stage:        "llm_first_content_ms",
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

type providerLatencyBenchStageSpec struct {
	Stage             string
	SourceTraceMarker string
	Series            providerLatencyBenchSeries
	PhysicalOnly      bool
}

func buildProviderLatencyBenchStageAvailability(summary providerLatencyBenchSummary, placeholder bool, physicalEvidence bool) []providerLatencyBenchStageAvailability {
	stages := providerLatencyBenchStageSpecs(summary)
	availability := make([]providerLatencyBenchStageAvailability, 0, len(stages))
	for _, stage := range stages {
		available := !placeholder && stage.Series.Samples > 0
		if stage.PhysicalOnly && !physicalEvidence {
			available = false
		}
		availability = append(availability, providerLatencyBenchStageAvailability{
			Stage:             stage.Stage,
			SourceTraceMarker: stage.SourceTraceMarker,
			Samples:           stage.Series.Samples,
			P50MS:             stage.Series.P50MS,
			P95MS:             stage.Series.P95MS,
			P99MS:             stage.Series.P99MS,
			Available:         available,
			Placeholder:       placeholder,
			PlaceholderReason: providerLatencyPlaceholderReason(placeholder),
		})
	}
	return availability
}

func providerLatencyBenchStageSpecs(summary providerLatencyBenchSummary) []providerLatencyBenchStageSpec {
	return []providerLatencyBenchStageSpec{
		{Stage: "transport_ingress_ms", SourceTraceMarker: "audio.frame.received", Series: summary.TransportIngressMS},
		{Stage: "codec_decode_ms", SourceTraceMarker: "xiaozhi.opus_frame.decoded", Series: summary.CodecDecodeMS},
		{Stage: "asr_first_partial_ms", SourceTraceMarker: "asr.first_partial", Series: summary.ASRFirstPartialMS},
		{Stage: "asr_final_ms", SourceTraceMarker: "asr.final", Series: summary.ASRFinalMS},
		{Stage: "llm_first_content_ms", SourceTraceMarker: "provider.first_content", Series: summary.LLMFirstContentMS},
		{Stage: "provider_first_byte_ms", SourceTraceMarker: "provider.first_byte", Series: summary.ProviderFirstByteMS},
		{Stage: "provider_first_content_ms", SourceTraceMarker: "provider.first_content", Series: summary.ProviderFirstContentMS},
		{Stage: "tts_first_audio_ms", SourceTraceMarker: "tts.first_audio", Series: summary.TTSFirstAudioMS},
		{Stage: "downlink_first_frame_ms", SourceTraceMarker: "audio.downlink.first_frame", Series: summary.DownlinkFirstFrameMS},
		{Stage: "audio_downlink_first_frame_ms", SourceTraceMarker: "audio.downlink.first_frame", Series: summary.AudioDownlinkFirstFrameMS},
		{Stage: "device_playback_start_ms", SourceTraceMarker: "device.playback.start", Series: summary.DevicePlaybackStartMS, PhysicalOnly: true},
		{Stage: "barge_in_detected_ms", SourceTraceMarker: "barge_in.detected", Series: summary.BargeInDetectedMS},
		{Stage: "barge_in_stop_ms", SourceTraceMarker: "playback.stop", Series: summary.BargeInStopMS},
		{Stage: "provider_cancel_ms", SourceTraceMarker: "provider.cancel.end", Series: summary.ProviderCancelMS},
		{Stage: "provider_cancel_done_ms", SourceTraceMarker: "provider.cancel.end", Series: summary.ProviderCancelDoneMS},
		{Stage: "playback_stop_ms", SourceTraceMarker: "playback.stop", Series: summary.PlaybackStopMS},
		{Stage: "playback_stop_done_ms", SourceTraceMarker: "playback.stop", Series: summary.PlaybackStopDoneMS},
		{Stage: "answer_first_audio_ms", SourceTraceMarker: "answer.first_audio.host_loopback", Series: summary.AnswerFirstAudioMS},
	}
}

func providerLatencyPlaceholderReason(placeholder bool) string {
	if !placeholder {
		return ""
	}
	return providerLatencyBenchPlaceholderReason
}

type providerLatencyBenchHostLoopbackEvidence struct {
	Valid                bool
	Samples              []providerLatencyBenchSample
	PhysicalEvidence     bool
	Execution            providerLatencyBenchExecution
	AudioQualityFindings []providerLatencyBenchFinding
}

type providerLatencyBenchHostLoopbackReport struct {
	SchemaVersion        string                                  `json:"schema_version"`
	Schema               string                                  `json:"schema"`
	ExecutionMode        string                                  `json:"execution_mode"`
	BaselineScope        string                                  `json:"baseline_scope"`
	DeviceID             string                                  `json:"device_id"`
	AnswerTurns          []providerLatencyBenchHostLoopbackTurn  `json:"answer_turns"`
	BargeInTurns         []providerLatencyBenchHostLoopbackTurn  `json:"barge_in_turns"`
	Samples              []providerLatencyBenchHostLoopbackTrace `json:"samples"`
	FirstAudioSamplesMS  []float64                               `json:"first_audio_samples_ms"`
	AbortStopSamplesMS   []float64                               `json:"abort_stop_samples_ms"`
	FirstAudioP95MS      *float64                                `json:"first_audio_p95_ms"`
	AbortStopP95MS       *float64                                `json:"abort_stop_p95_ms"`
	HostCandidate        bool                                    `json:"host_candidate"`
	PRDAccepted          bool                                    `json:"prd_accepted"`
	Execution            providerLatencyBenchExecution           `json:"execution"`
	AudioQuality         *providerLatencyBenchAudioQualityReport `json:"audio_quality,omitempty"`
	LocalAckAudioQuality *providerLatencyBenchAudioQualityReport `json:"local_ack_audio_quality,omitempty"`
	TTSAudioQuality      *providerLatencyBenchAudioQualityReport `json:"tts_audio_quality,omitempty"`
}

type providerLatencyBenchAudioQualityReport struct {
	Status   string   `json:"status"`
	Findings []string `json:"findings,omitempty"`
}

type providerLatencyBenchHostLoopbackTurn struct {
	TraceSummary *providerLatencyBenchHostLoopbackTrace `json:"trace_summary"`
	FirstAudioMS *float64                               `json:"first_audio_ms,omitempty"`
}

type providerLatencyBenchHostLoopbackTrace struct {
	TransportIngressMS            *float64 `json:"transport_ingress_ms,omitempty"`
	XiaozhiListenToAudioIngressMS *float64 `json:"xiaozhi_listen_to_audio_ingress_ms,omitempty"`
	CodecDecodeMS                 *float64 `json:"codec_decode_ms,omitempty"`
	XiaozhiOpusDecodeMS           *float64 `json:"xiaozhi_opus_decode_ms,omitempty"`
	ASRFirstPartialMS             *float64 `json:"asr_first_partial_ms,omitempty"`
	ASRFinalMS                    *float64 `json:"asr_final_ms,omitempty"`
	LLMFirstContentMS             *float64 `json:"llm_first_content_ms,omitempty"`
	ProviderFirstByteMS           *float64 `json:"provider_first_byte_ms,omitempty"`
	ProviderFirstContentMS        *float64 `json:"provider_first_content_ms,omitempty"`
	TTSFirstAudioMS               *float64 `json:"tts_first_audio_ms,omitempty"`
	DownlinkFirstFrameMS          *float64 `json:"downlink_first_frame_ms,omitempty"`
	AudioDownlinkFirstFrameMS     *float64 `json:"audio_downlink_first_frame_ms,omitempty"`
	DevicePlaybackStartMS         *float64 `json:"device_playback_start_ms,omitempty"`
	BargeInDetectedMS             *float64 `json:"barge_in_detected_ms,omitempty"`
	BargeInStopMS                 *float64 `json:"barge_in_stop_ms,omitempty"`
	ProviderCancelMS              *float64 `json:"provider_cancel_ms,omitempty"`
	ProviderCancelDoneMS          *float64 `json:"provider_cancel_done_ms,omitempty"`
	PlaybackStopMS                *float64 `json:"playback_stop_ms,omitempty"`
	PlaybackStopDoneMS            *float64 `json:"playback_stop_done_ms,omitempty"`
	AnswerFirstAudioMS            *float64 `json:"answer_first_audio_ms,omitempty"`
	AnswerFirstAudioTotalMS       *float64 `json:"answer_first_audio_total_ms,omitempty"`
}

func loadProviderLatencyBenchHostLoopbackReport(fixturePath string) (providerLatencyBenchHostLoopbackEvidence, []providerLatencyBenchFinding) {
	if strings.ToLower(filepath.Ext(fixturePath)) != ".json" {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	if len(data) > providerLatencyFixtureSidecarMaxBytes {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	var hostReport providerLatencyBenchHostLoopbackReport
	decoder = json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&hostReport); err != nil {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	if !validProviderLatencyHostLoopbackReport(hostReport) {
		return providerLatencyBenchHostLoopbackEvidence{}, []providerLatencyBenchFinding{invalidProviderLatencyHostLoopbackFinding()}
	}
	evidence := providerLatencyBenchHostLoopbackEvidence{
		Valid:                true,
		PhysicalEvidence:     providerLatencyHostLoopbackHasPhysicalEvidence(hostReport),
		Execution:            hostReport.Execution,
		AudioQualityFindings: providerLatencyHostLoopbackAudioQualityFindings(hostReport),
	}
	evidence.Samples = providerLatencyHostLoopbackSamples(hostReport, evidence.PhysicalEvidence)
	return evidence, nil
}

func invalidProviderLatencyHostLoopbackFinding() providerLatencyBenchFinding {
	return providerLatencyBenchFinding{
		Code:    "host_loopback_report_invalid",
		Message: "host-loopback report is invalid or unsafe",
	}
}

func validProviderLatencyHostLoopbackReport(report providerLatencyBenchHostLoopbackReport) bool {
	switch providerLatencyHostLoopbackSchema(report) {
	case "a21.xiaozhi_voice_bench.v1", "a21.audio.local_voice_loopback.v1", "a21.provider_latency_host_loopback.v1", "a21.virtual_xiaozhi_harness.v1":
	default:
		return false
	}
	mode := strings.TrimSpace(report.ExecutionMode)
	return mode == "" || mode == "host_loopback"
}

func providerLatencyHostLoopbackSchema(report providerLatencyBenchHostLoopbackReport) string {
	if strings.TrimSpace(report.SchemaVersion) != "" {
		return strings.TrimSpace(report.SchemaVersion)
	}
	return strings.TrimSpace(report.Schema)
}

func providerLatencyHostLoopbackIsVirtualXiaozhi(report providerLatencyBenchHostLoopbackReport) bool {
	return providerLatencyHostLoopbackSchema(report) == "a21.virtual_xiaozhi_harness.v1"
}

func providerLatencyHostLoopbackHasPhysicalEvidence(report providerLatencyBenchHostLoopbackReport) bool {
	if report.Execution.HardwareExecuted {
		return true
	}
	return strings.TrimSpace(report.BaselineScope) == "physical_stackchan"
}

func providerLatencyHostLoopbackAudioQualityFindings(report providerLatencyBenchHostLoopbackReport) []providerLatencyBenchFinding {
	checks := []struct {
		source  string
		quality *providerLatencyBenchAudioQualityReport
	}{
		{source: "", quality: report.AudioQuality},
		{source: "local_ack", quality: report.LocalAckAudioQuality},
		{source: "tts", quality: report.TTSAudioQuality},
	}
	findings := make([]providerLatencyBenchFinding, 0)
	for _, check := range checks {
		if !providerLatencyAudioQualityBlocksHostCandidate(check.quality) {
			continue
		}
		findings = append(findings, providerLatencyHostLoopbackAudioQualityFinding(check.source))
	}
	return findings
}

func providerLatencyAudioQualityBlocksHostCandidate(quality *providerLatencyBenchAudioQualityReport) bool {
	if quality == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(quality.Status)) {
	case "warning", "failed":
		return true
	}
	for _, finding := range quality.Findings {
		if providerLatencyBlockingAudioQualityFinding(finding) {
			return true
		}
	}
	return false
}

func providerLatencyBlockingAudioQualityFinding(code string) bool {
	switch strings.TrimSpace(code) {
	case "audio_quality_clipping_detected",
		"audio_quality_low_headroom",
		"audio_quality_near_silence",
		"audio_quality_dc_offset_detected",
		"audio_quality_invalid_payload",
		"audio_quality_invalid_format",
		"audio_quality_unsupported_codec",
		"audio_quality_unavailable",
		"audio_quality_format_mismatch":
		return true
	default:
		return false
	}
}
