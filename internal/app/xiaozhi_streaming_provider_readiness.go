package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/providers"
)

const xiaozhiStreamingProviderReadinessSchemaVersion = "a21.xiaozhi_streaming_provider_readiness.v1"

type xiaozhiStreamingProviderReadinessOptions struct {
	OutputDir string
	Env       []string
}

type xiaozhiStreamingProviderReadinessReport struct {
	SchemaVersion        string                                 `json:"schema_version"`
	GeneratedAtMS        int64                                  `json:"generated_at_ms"`
	ExecutionMode        string                                 `json:"execution_mode"`
	ProductMode          string                                 `json:"product_mode"`
	ChainMode            string                                 `json:"chain_mode"`
	ProfessionalBoundary string                                 `json:"professional_boundary"`
	Selection            providers.VoicePipelineSelection       `json:"selection"`
	ASR                  xiaozhiStreamingProviderReadinessStage `json:"asr"`
	LLM                  xiaozhiStreamingProviderReadinessStage `json:"llm"`
	TTS                  xiaozhiStreamingProviderReadinessStage `json:"tts"`
	GateStatus           string                                 `json:"gate_status"`
	PRDAccepted          bool                                   `json:"prd_accepted"`
	Findings             []string                               `json:"findings"`
	Redaction            xiaozhiVoiceBenchRedaction             `json:"redaction"`
	NextRequired         []string                               `json:"next_required"`
	ReportPath           string                                 `json:"report_path,omitempty"`
}

type xiaozhiStreamingProviderReadinessStage struct {
	Profile              string `json:"profile"`
	ProfileEnv           string `json:"profile_env,omitempty"`
	Adapter              string `json:"adapter"`
	Capability           string `json:"capability"`
	Ready                bool   `json:"ready"`
	RealProvider         bool   `json:"real_provider"`
	Streaming            bool   `json:"streaming"`
	UsesMock             bool   `json:"uses_mock"`
	UsesFileBoundary     bool   `json:"uses_file_boundary"`
	UsesWAVBoundary      bool   `json:"uses_wav_boundary"`
	ImplementedInGateway bool   `json:"implemented_in_gateway"`
}

func runXiaozhiStreamingProviderReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiStreamingProviderReadinessOptions{
		OutputDir: "reports",
		Env:       os.Environ(),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 xiaozhi-streaming-provider-readiness [--output-dir reports]")
			return 0
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-streaming-provider-readiness option %q\n", args[i])
			return 2
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "xiaozhi streaming provider readiness report dir invalid: %v\n", err)
			return 1
		}
	}
	report := buildXiaozhiStreamingProviderReadinessReport(options.Env)
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiStreamingProviderReadinessReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write xiaozhi streaming provider readiness report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintf(stderr, "encode xiaozhi streaming provider readiness report: %v\n", err)
		return 1
	}
	if report.GateStatus != "passed" {
		return 1
	}
	return 0
}

func buildXiaozhiStreamingProviderReadinessReport(env []string) xiaozhiStreamingProviderReadinessReport {
	selection := providers.VoicePipelineSelectionFromEnv(env)
	asr := classifyXiaozhiStreamingASR(env, selection)
	llm := classifyXiaozhiStreamingLLM(env, selection)
	tts := classifyXiaozhiStreamingTTS(env, selection)
	findings := append([]string{}, asrFindings(asr)...)
	findings = append(findings, llmFindings(llm)...)
	findings = append(findings, ttsFindings(tts)...)
	status := "blocked"
	if asr.Ready && llm.Ready && tts.Ready {
		status = "passed"
	}
	next := []string{}
	if !asr.Ready {
		next = append(next, "implement a real providers.StreamingASRAdapter for stock /v1/xiaozhi")
	}
	if !llm.Ready {
		next = append(next, "select a real text_stream LLM profile")
	}
	if !tts.Ready {
		next = append(next, "implement streaming TTS that emits PCM/Opus chunks before complete WAV/file synthesis")
	}
	return xiaozhiStreamingProviderReadinessReport{
		SchemaVersion:        xiaozhiStreamingProviderReadinessSchemaVersion,
		GeneratedAtMS:        time.Now().UnixMilli(),
		ExecutionMode:        "static_no_execute_provider_capability_gate",
		ProductMode:          "dialogue",
		ChainMode:            "dialogue_low_latency",
		ProfessionalBoundary: "v21_adapter_only",
		Selection:            selection,
		ASR:                  asr,
		LLM:                  llm,
		TTS:                  tts,
		GateStatus:           status,
		PRDAccepted:          false,
		Findings:             uniqueXiaozhiStreamingFindings(findings),
		Redaction: xiaozhiVoiceBenchRedaction{
			PayloadsStored:         false,
			CredentialValuesStored: false,
			FullURLsStored:         false,
			LocalPathsStored:       false,
		},
		NextRequired: next,
	}
}

func classifyXiaozhiStreamingASR(env []string, selection providers.VoicePipelineSelection) xiaozhiStreamingProviderReadinessStage {
	profile := normalizeXiaozhiStreamingProfile(selection.ASRProfile)
	stage := xiaozhiStreamingProviderReadinessStage{
		Profile:    profile,
		ProfileEnv: selection.ASRProfileEnv,
		Adapter:    profile,
		Capability: "streaming_asr_session",
	}
	switch profile {
	case "a21_fixture_streaming_asr":
		stage.Ready = true
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
	case "sherpa_onnx_streaming", "local_sherpa_onnx_streaming", "streaming_zipformer":
		stage.Adapter = "local_sherpa_onnx_streaming_asr"
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
		if xiaozhiSherpaStreamingASRConfigured(env) {
			stage.Ready = true
		}
	case "sherpa_onnx", "local_sherpa_onnx":
		stage.Adapter = "local_sherpa_onnx_asr"
		stage.UsesFileBoundary = true
		stage.UsesWAVBoundary = true
		stage.ImplementedInGateway = true
	case "mock", "mock_local_asr", "mock-local-asr", "":
		stage.Adapter = "mock_local_asr"
		stage.UsesMock = true
	default:
		if strings.Contains(profile, "iflytek") || strings.Contains(profile, "iat") || strings.Contains(profile, "xfyun") {
			stage.Adapter = "missing_iflytek_iat_streaming_adapter"
			stage.RealProvider = true
		} else {
			stage.Adapter = "unknown_asr_adapter"
		}
	}
	return stage
}

func classifyXiaozhiStreamingLLM(env []string, selection providers.VoicePipelineSelection) xiaozhiStreamingProviderReadinessStage {
	profile := normalizeXiaozhiStreamingProfile(selection.LLMProfile)
	stage := xiaozhiStreamingProviderReadinessStage{
		Profile:    profile,
		ProfileEnv: selection.LLMProfileEnv,
		Adapter:    profile,
		Capability: "text_delta_stream",
	}
	if profile == "a21_fixture_text_stream" {
		stage.Ready = true
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
		return stage
	}
	if profile == "" || profile == "mock" || profile == "mock_text_stream" || profile == "mock-text-stream" {
		stage.Adapter = "mock_text_stream"
		stage.UsesMock = true
		return stage
	}
	if xiaozhiStreamingTextProfileConfigured(env, profile) {
		stage.Ready = true
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
		return stage
	}
	stage.Adapter = "unconfigured_text_stream_profile"
	return stage
}

func classifyXiaozhiStreamingTTS(env []string, selection providers.VoicePipelineSelection) xiaozhiStreamingProviderReadinessStage {
	profile := normalizeXiaozhiStreamingProfile(selection.TTSProfile)
	stage := xiaozhiStreamingProviderReadinessStage{
		Profile:    profile,
		ProfileEnv: selection.TTSProfileEnv,
		Adapter:    profile,
		Capability: "incremental_tts_audio_stream",
	}
	switch profile {
	case "a21_fixture_streaming_tts":
		stage.Ready = true
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
	case "doubao_tts_realtime", "doubao_realtime_tts":
		stage.Adapter = "doubao_realtime_tts_adapter"
		stage.RealProvider = true
		stage.Streaming = true
		stage.ImplementedInGateway = true
		if xiaozhiDoubaoRealtimeTTSConfigured(env) {
			stage.Ready = true
		}
	case "iflytek_tts", "iflytek", "xfyun", "xfyun_tts":
		stage.Adapter = "iflytek_tts_via_local_wav_adapter"
		stage.RealProvider = true
		stage.UsesFileBoundary = true
		stage.UsesWAVBoundary = true
		stage.ImplementedInGateway = true
	case "sherpa_onnx", "sherpa_onnx_tts", "local_sherpa_onnx", "local_sherpa_onnx_tts", "macos_say", "voice_clone_cli":
		stage.Adapter = "local_tts_wav_adapter"
		stage.UsesFileBoundary = true
		stage.UsesWAVBoundary = true
		stage.ImplementedInGateway = true
	case "mock", "mock_fast_tts", "mock-fast-tts", "":
		stage.Adapter = "mock_fast_tts"
		stage.UsesMock = true
	default:
		stage.Adapter = "unknown_tts_adapter"
	}
	return stage
}

func asrFindings(stage xiaozhiStreamingProviderReadinessStage) []string {
	if stage.Ready {
		return nil
	}
	if stage.UsesMock {
		return []string{"asr_mock_fixture_not_realtime"}
	}
	if stage.UsesWAVBoundary {
		return []string{"asr_batch_wav_boundary_not_xiaozhi_streaming"}
	}
	if stage.Adapter == "local_sherpa_onnx_streaming_asr" {
		return []string{"asr_sherpa_streaming_helper_or_model_missing"}
	}
	if strings.Contains(stage.Adapter, "iflytek") {
		return []string{"asr_iflytek_iat_streaming_adapter_missing"}
	}
	return []string{"asr_streaming_adapter_missing"}
}

func llmFindings(stage xiaozhiStreamingProviderReadinessStage) []string {
	if stage.Ready {
		return nil
	}
	if stage.UsesMock {
		return []string{"llm_mock_text_stream_not_product"}
	}
	return []string{"llm_text_stream_profile_unconfigured"}
}

func ttsFindings(stage xiaozhiStreamingProviderReadinessStage) []string {
	if stage.Ready {
		return nil
	}
	if stage.UsesMock {
		return []string{"tts_mock_fixture_not_realtime"}
	}
	if stage.UsesWAVBoundary {
		return []string{"tts_wav_file_boundary_not_xiaozhi_streaming"}
	}
	if stage.Adapter == "doubao_realtime_tts_adapter" {
		return []string{"tts_doubao_realtime_config_missing"}
	}
	return []string{"tts_streaming_adapter_missing"}
}

func xiaozhiStreamingTextProfileConfigured(env []string, profile string) bool {
	profile = normalizeXiaozhiStreamingProfile(profile)
	if profile == "" {
		return false
	}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(key, "PROVIDER") && strings.Contains(value, profile) {
			return true
		}
	}
	switch profile {
	case "deepseek", "stepfun", "local_ollama":
		return true
	default:
		return false
	}
}

func xiaozhiSherpaStreamingASRConfigured(env []string) bool {
	options := applyLocalASRStreamingSmokeDefaults(localASRStreamingSmokeOptions{
		HelperPath: strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_STREAMING_HELPER")),
		ModelDir:   strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_ASR_MODEL_DIR")),
	})
	return pathExists(options.HelperPath) &&
		dirExists(options.ModelDir) &&
		len(missingSherpaStreamingModelFiles(options.ModelDir)) == 0
}

func xiaozhiDoubaoRealtimeTTSConfigured(env []string) bool {
	return xiaozhiDoubaoRealtimeTTSCredentialConfigured(env) &&
		strings.TrimSpace(appEnvValue(env, "A21_DOUBAO_TTS_MODEL")) != "" &&
		strings.TrimSpace(appEnvValue(env, "A21_DOUBAO_TTS_VOICE")) != ""
}

func xiaozhiDoubaoRealtimeTTSCredentialConfigured(env []string) bool {
	return strings.TrimSpace(appEnvValue(env, "A21_DOUBAO_API_KEY")) != "" ||
		strings.TrimSpace(appEnvValue(env, "A21_DOUBAO_ACCESS_TOKEN")) != ""
}

func normalizeXiaozhiStreamingProfile(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func uniqueXiaozhiStreamingFindings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func writeXiaozhiStreamingProviderReadinessReport(outputDir string, report xiaozhiStreamingProviderReadinessReport) (string, error) {
	now := time.Now()
	path := filepath.Join(outputDir, fmt.Sprintf("a21-xiaozhi-streaming-provider-readiness-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", err
	}
	return path, nil
}
