package app

import (
	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/providers"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type localVoiceLoopbackReport struct {
	SchemaVersion              string               `json:"schema_version"`
	GeneratedAtMS              int64                `json:"generated_at_ms"`
	Metadata                   latencyBenchMetadata `json:"metadata"`
	Status                     string               `json:"status"`
	Repeat                     int                  `json:"repeat"`
	InputTextBytes             int                  `json:"input_text_bytes"`
	VADStatus                  string               `json:"vad_status"`
	VADDetector                string               `json:"vad_detector"`
	VADSpeechStartEvents       int                  `json:"vad_speech_start_events"`
	VADSpeechEndEvents         int                  `json:"vad_speech_end_events"`
	ASRProvider                string               `json:"asr_provider"`
	ASREngine                  string               `json:"asr_engine,omitempty"`
	ASRModelDir                string               `json:"asr_model_dir,omitempty"`
	ASRWAVName                 string               `json:"asr_wav_name,omitempty"`
	ASRFirstPartialMS          float64              `json:"asr_first_partial_ms"`
	ASRInputDurationMS         float64              `json:"asr_input_duration_ms,omitempty"`
	ASRDecodeDurationMS        float64              `json:"asr_decode_duration_ms,omitempty"`
	ASRRealTimeFactor          float64              `json:"asr_real_time_factor,omitempty"`
	ASRTextChars               int                  `json:"asr_text_chars,omitempty"`
	ASRTranscriptPolicy        string               `json:"asr_transcript_policy,omitempty"`
	TextStreamProvider         string               `json:"text_stream_provider"`
	TextStreamFamily           string               `json:"text_stream_family"`
	TextStreamExecuted         bool                 `json:"text_stream_executed"`
	TextStreamFallbackUsed     bool                 `json:"text_stream_fallback_used,omitempty"`
	TextStreamFallbackProvider string               `json:"text_stream_fallback_provider,omitempty"`
	TextStreamFallbackReason   string               `json:"text_stream_fallback_reason,omitempty"`
	TextStreamEndpointHost     string               `json:"text_stream_endpoint_host,omitempty"`
	TextStreamFirstContentMS   float64              `json:"text_stream_first_content_ms"`
	TextStreamContentDeltas    int                  `json:"text_stream_content_delta_count"`
	TextStreamReasoningDeltas  int                  `json:"text_stream_reasoning_delta_count"`
	TextStreamDone             bool                 `json:"text_stream_done"`
	AnswerVoicePreviewChars    int                  `json:"answer_voice_preview_chars,omitempty"`
	LocalAckEnabled            bool                 `json:"local_ack_enabled"`
	LocalAckStatus             string               `json:"local_ack_status,omitempty"`
	LocalAckTTSProvider        string               `json:"local_ack_tts_provider,omitempty"`
	LocalAckTTSFirstAudioMS    float64              `json:"local_ack_tts_first_audio_ms,omitempty"`
	LocalAckFirstAudioTotalMS  float64              `json:"local_ack_first_audio_total_ms,omitempty"`
	LocalAckAudioPath          string               `json:"local_ack_audio_path,omitempty"`
	TTSProvider                string               `json:"tts_provider"`
	TTSVoice                   string               `json:"tts_voice"`
	TTSOutputFormat            string               `json:"tts_output_format"`
	TTSAudioPath               string               `json:"tts_audio_path,omitempty"`
	TTSFirstAudioMS            float64              `json:"tts_first_audio_ms"`
	TTSFirstAudioP50MS         float64              `json:"tts_first_audio_p50_ms,omitempty"`
	TTSFirstAudioP95MS         float64              `json:"tts_first_audio_p95_ms,omitempty"`
	AnswerFirstAudioP50MS      float64              `json:"answer_first_audio_total_p50_ms,omitempty"`
	AnswerFirstAudioP95MS      float64              `json:"answer_first_audio_total_p95_ms,omitempty"`
	FirstAudioTotalP50MS       float64              `json:"first_audio_total_p50_ms,omitempty"`
	FirstAudioTotalP95MS       float64              `json:"first_audio_total_p95_ms,omitempty"`
	TotalDurationMS            float64              `json:"total_duration_ms"`
	BargeInStatus              string               `json:"barge_in_status"`
	BargeInStopP95MS           float64              `json:"barge_in_stop_p95_ms,omitempty"`
	ReportPath                 string               `json:"report_path,omitempty"`
	Findings                   []string             `json:"findings,omitempty"`
}

const (
	fastCompanionTextStreamMaxTokens        = 12
	fastCompanionAnswerVoicePreviewMaxRunes = 12
)

func runLocalVoiceLoopback(args []string, stdout io.Writer, stderr io.Writer) int {
	inputText := "A21 本地语音 loopback 测试。"
	engine := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TTS_ENGINE"), "sherpa_onnx"))
	voice := strings.TrimSpace(os.Getenv("A21_LOCAL_TTS_VOICE"))
	modelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_MODEL_DIR"))
	speakerID := parsePositiveIntOrDefault(os.Getenv("A21_SHERPA_ONNX_SPEAKER_ID"), 21)
	asrProvider := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_ASR_PROVIDER"), "mock_asr"))
	asrFamily := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_FAMILY"))
	asrModelDir := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_MODEL_DIR"))
	asrWAVPath := strings.TrimSpace(os.Getenv("A21_SHERPA_ONNX_ASR_WAV"))
	textProvider := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TEXT_PROVIDER"), "mock_text_stream"))
	fallbackTextProvider := strings.TrimSpace(firstNonEmpty(os.Getenv("A21_LOCAL_TEXT_FALLBACK_PROVIDER"), os.Getenv("A21_TEXT_STREAM_FALLBACK_PROFILE")))
	executeTextProvider := false
	repeat := parsePositiveIntOrDefault(os.Getenv("A21_LOCAL_VOICE_LOOPBACK_REPEAT"), 1)
	outputDir := "reports"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 local-voice-loopback [--engine sherpa_onnx|macos_say] [--asr-provider mock_asr|sherpa_onnx] [--asr-family paraformer|sense_voice|streaming_zipformer] [--asr-model-dir <dir>] [--asr-wav <path>] [--text-provider mock_text_stream|deepseek|local_ollama|<A21_PROVIDER_PROFILES_PATH route-eligible profile>] [--fallback-text-provider local_ollama|<route-eligible profile>] [--execute-text-provider] [--text <text>] [--voice Tingting] [--model-dir <dir>] [--speaker-id 21] [--repeat 3] [--output-dir reports]")
			return 0
		case "--engine":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--engine requires a value")
				return 2
			}
			i++
			engine = args[i]
		case "--text":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text requires a value")
				return 2
			}
			i++
			inputText = args[i]
		case "--voice":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice requires a value")
				return 2
			}
			i++
			voice = args[i]
		case "--model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--model-dir requires a value")
				return 2
			}
			i++
			modelDir = args[i]
		case "--speaker-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--speaker-id requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--speaker-id must be a positive integer")
				return 2
			}
			speakerID = value
		case "--text-provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--text-provider requires a value")
				return 2
			}
			i++
			textProvider = args[i]
		case "--fallback-text-provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--fallback-text-provider requires a value")
				return 2
			}
			i++
			fallbackTextProvider = args[i]
		case "--asr-provider":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-provider requires a value")
				return 2
			}
			i++
			asrProvider = args[i]
		case "--asr-family":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-family requires a value")
				return 2
			}
			i++
			asrFamily = args[i]
		case "--asr-model-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-model-dir requires a value")
				return 2
			}
			i++
			asrModelDir = args[i]
		case "--asr-wav":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--asr-wav requires a value")
				return 2
			}
			i++
			asrWAVPath = args[i]
		case "--execute-text-provider":
			executeTextProvider = true
		case "--repeat":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--repeat requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				fmt.Fprintln(stderr, "--repeat must be a positive integer")
				return 2
			}
			repeat = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			outputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown local-voice-loopback option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(outputDir); err != nil {
		fmt.Fprintf(stderr, "local voice loopback report dir invalid: %v\n", err)
		return 1
	}
	report, err := buildLocalVoiceLoopbackReport(context.Background(), localTTSRuntimeOptions{
		Engine:    engine,
		Text:      inputText,
		Voice:     voice,
		ModelDir:  modelDir,
		SpeakerID: speakerID,
		OutputDir: outputDir,
	}, repeat, localVoiceLoopbackTextStreamOptions{
		Provider:         textProvider,
		FallbackProvider: fallbackTextProvider,
		Execute:          executeTextProvider,
		Env:              os.Environ(),
	}, localVoiceLoopbackASROptions{
		Provider:  asrProvider,
		Family:    asrFamily,
		ModelDir:  asrModelDir,
		WAVPath:   asrWAVPath,
		OutputDir: outputDir,
	})
	if err != nil {
		fmt.Fprintf(stderr, "local voice loopback failed: %v\n", err)
		return 1
	}
	reportPath, err := writeLocalVoiceLoopbackReport(outputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write local voice loopback report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONLocalVoiceLoopback(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode local voice loopback report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

type localVoiceLoopbackTextStreamOptions struct {
	Provider         string
	FallbackProvider string
	Execute          bool
	Env              []string
	Client           *http.Client
}
type localVoiceLoopbackASROptions struct {
	Provider  string
	Family    string
	ModelDir  string
	WAVPath   string
	OutputDir string
}

func buildLocalVoiceLoopbackReport(ctx context.Context, ttsOptions localTTSRuntimeOptions, repeat int, textOptions localVoiceLoopbackTextStreamOptions, asrOptions localVoiceLoopbackASROptions) (localVoiceLoopbackReport, error) {
	if repeat <= 0 {
		repeat = 1
	}
	start := time.Now()
	frontEnd := audio.RunMockFrontEndEval()
	report := localVoiceLoopbackReport{
		SchemaVersion:        "a21.audio.local_voice_loopback.v1",
		GeneratedAtMS:        time.Now().UnixMilli(),
		Metadata:             buildLatencyBenchMetadata(),
		Status:               "failed",
		Repeat:               repeat,
		InputTextBytes:       len([]byte(ttsOptions.Text)),
		VADStatus:            frontEnd.Status,
		VADDetector:          frontEnd.Detector,
		VADSpeechStartEvents: frontEnd.SpeechStartEvents,
		VADSpeechEndEvents:   frontEnd.SpeechEndEvents,
		ASRProvider:          "mock_asr",
		TextStreamProvider:   "mock_text_stream",
		TextStreamFamily:     string(providers.ProviderFamilyTextStream),
		BargeInStatus:        "not_run",
	}

	transcript, err := runLocalVoiceLoopbackASR(ctx, asrOptions, &report)
	if err != nil {
		return report, err
	}
	if strings.TrimSpace(transcript) == "" {
		report.Findings = append(report.Findings, "ASR did not produce text")
		return report, nil
	}

	textResultCh := make(chan localVoiceLoopbackTextResult, 1)
	go func() {
		textReport := localVoiceLoopbackReport{}
		ttsText, textErr := runLocalVoiceLoopbackTextStream(ctx, transcript, textOptions, &textReport)
		textResultCh <- localVoiceLoopbackTextResult{text: ttsText, report: textReport, err: textErr}
	}()
	if err := runLocalVoiceLoopbackLocalAck(ctx, ttsOptions, &report); err != nil {
		return report, err
	}
	textResult := <-textResultCh
	mergeLocalVoiceLoopbackTextReport(&report, textResult.report)
	if textResult.err != nil {
		return report, textResult.err
	}
	ttsText := fastCompanionVoicePreview(textResult.text)
	if ttsText == "" {
		report.Findings = append(report.Findings, "text stream produced no content")
		return report, nil
	}
	report.AnswerVoicePreviewChars = len([]rune(ttsText))

	ttsSamples := make([]time.Duration, 0, repeat)
	firstAudioTotalSamples := make([]time.Duration, 0, repeat)
	for sample := 0; sample < repeat; sample++ {
		ttsOptions.Text = ttsText
		ttsReport, err := synthesizeLocalTTS(ctx, ttsOptions)
		if err != nil {
			report.Findings = append(report.Findings, "local TTS failed")
			return report, err
		}
		report.TTSProvider = ttsReport.Provider
		report.TTSVoice = ttsReport.Voice
		report.TTSOutputFormat = ttsReport.OutputFormat
		report.TTSAudioPath = ttsReport.OutputPath
		report.TTSFirstAudioMS = ttsReport.TTSFirstAudioMS
		if ttsReport.Status != "passed" {
			report.Findings = append(report.Findings, "local TTS did not pass")
			return report, nil
		}
		ttsSamples = append(ttsSamples, time.Duration(ttsReport.TTSFirstAudioMS*1000)*time.Microsecond)
		answerStartGateMS := math.Max(report.TextStreamFirstContentMS, report.LocalAckTTSFirstAudioMS)
		firstAudioTotalSamples = append(firstAudioTotalSamples, time.Duration((report.ASRFirstPartialMS+answerStartGateMS+ttsReport.TTSFirstAudioMS)*1000)*time.Microsecond)
	}
	report.TTSFirstAudioP50MS = percentileMS(ttsSamples, 0.50)
	report.TTSFirstAudioP95MS = percentileMS(ttsSamples, 0.95)
	report.AnswerFirstAudioP50MS = percentileMS(firstAudioTotalSamples, 0.50)
	report.AnswerFirstAudioP95MS = percentileMS(firstAudioTotalSamples, 0.95)
	report.FirstAudioTotalP50MS = report.AnswerFirstAudioP50MS
	report.FirstAudioTotalP95MS = report.AnswerFirstAudioP95MS

	bench, err := runMockLatencyBench(1)
	if err != nil {
		report.Findings = append(report.Findings, "barge-in latency bench failed")
		return report, err
	}
	report.BargeInStatus = "benchmarked"
	report.BargeInStopP95MS = bench.Summary.AudioWSBargeInMS.P95MS
	report.TotalDurationMS = elapsedReportMS(start)
	report.Status = "passed"
	return report, nil
}

type localVoiceLoopbackTextResult struct {
	text   string
	report localVoiceLoopbackReport
	err    error
}

func runLocalVoiceLoopbackLocalAck(ctx context.Context, options localTTSRuntimeOptions, report *localVoiceLoopbackReport) error {
	report.LocalAckEnabled = true
	ackOptions := options
	ackOptions.Text = localCompanionAckText()
	ackReport, err := synthesizeLocalTTS(ctx, ackOptions)
	if err != nil {
		report.LocalAckStatus = "failed"
		report.Findings = append(report.Findings, "local ack TTS failed")
		return err
	}
	report.LocalAckTTSProvider = ackReport.Provider
	report.LocalAckTTSFirstAudioMS = ackReport.TTSFirstAudioMS
	report.LocalAckAudioPath = ackReport.OutputPath
	report.LocalAckFirstAudioTotalMS = report.ASRFirstPartialMS + ackReport.TTSFirstAudioMS
	if ackReport.Status != "passed" {
		report.LocalAckStatus = "failed"
		report.Findings = append(report.Findings, "local ack TTS did not pass")
		return nil
	}
	report.LocalAckStatus = "passed"
	return nil
}
func localCompanionAckText() string {
	return "嗯，我在。"
}
func mergeLocalVoiceLoopbackTextReport(report *localVoiceLoopbackReport, textReport localVoiceLoopbackReport) {
	report.TextStreamProvider = textReport.TextStreamProvider
	report.TextStreamFamily = textReport.TextStreamFamily
	report.TextStreamExecuted = textReport.TextStreamExecuted
	report.TextStreamFallbackUsed = textReport.TextStreamFallbackUsed
	report.TextStreamFallbackProvider = textReport.TextStreamFallbackProvider
	report.TextStreamFallbackReason = textReport.TextStreamFallbackReason
	report.TextStreamEndpointHost = textReport.TextStreamEndpointHost
	report.TextStreamFirstContentMS = textReport.TextStreamFirstContentMS
	report.TextStreamContentDeltas = textReport.TextStreamContentDeltas
	report.TextStreamReasoningDeltas = textReport.TextStreamReasoningDeltas
	report.TextStreamDone = textReport.TextStreamDone
	if len(textReport.Findings) > 0 {
		report.Findings = append(report.Findings, textReport.Findings...)
	}
}
func runLocalVoiceLoopbackASR(ctx context.Context, options localVoiceLoopbackASROptions, report *localVoiceLoopbackReport) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(firstNonEmpty(options.Provider, "mock_asr")))
	provider = strings.ReplaceAll(provider, "-", "_")
	switch provider {
	case "", "mock", "mock_asr":
		asrStart := time.Now()
		report.ASRProvider = "mock_asr"
		report.ASRFirstPartialMS = elapsedReportMS(asrStart)
		return "a21 mock transcript", nil
	case "sherpa", "sherpa_onnx":
		result, err := runSherpaONNXASR(ctx, audio.LocalASROptions{
			OutputDir: options.OutputDir,
			ModelDir:  options.ModelDir,
			Family:    options.Family,
			WAVPath:   options.WAVPath,
		})
		if err != nil {
			report.Findings = append(report.Findings, "sherpa-onnx ASR failed")
			return "", err
		}
		report.ASRProvider = result.Report.Provider
		report.ASREngine = result.Report.Engine
		report.ASRModelDir = result.Report.ModelDir
		report.ASRWAVName = result.Report.WAVName
		report.ASRInputDurationMS = result.Report.InputDurationMS
		report.ASRDecodeDurationMS = result.Report.DecodeDurationMS
		report.ASRRealTimeFactor = result.Report.RealTimeFactor
		report.ASRTextChars = result.Report.TextChars
		report.ASRTranscriptPolicy = result.Report.TranscriptPolicy
		report.ASRFirstPartialMS = result.Report.DecodeDurationMS
		if result.Report.Status != "passed" {
			report.Findings = append(report.Findings, "sherpa-onnx ASR did not pass")
			return "", nil
		}
		return result.Transcript, nil
	default:
		report.Findings = append(report.Findings, "unsupported local ASR provider")
		return "", fmt.Errorf("unsupported local ASR provider")
	}
}
func runLocalVoiceLoopbackTextStream(ctx context.Context, prompt string, options localVoiceLoopbackTextStreamOptions, report *localVoiceLoopbackReport) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(options.Provider))
	provider = strings.ReplaceAll(provider, "-", "_")
	switch provider {
	case "", "mock", "mock_text_stream":
		return runMockLocalVoiceLoopbackTextStream(report)
	}
	profile, _, ok := providers.ProviderProfileByNameFromEnv(options.Env, provider)
	if !ok || profile.Family != providers.ProviderFamilyTextStream {
		report.Findings = append(report.Findings, "unsupported local text provider")
		return "", fmt.Errorf("unsupported local text provider")
	}
	if !profile.RouteEligible {
		report.Findings = append(report.Findings, "local text provider is not route eligible")
		return "", fmt.Errorf("unsupported local text provider")
	}
	if profile.Protocol != "openai_chat_completions" && profile.Protocol != "ollama_chat" {
		report.Findings = append(report.Findings, "unsupported local text provider protocol")
		return "", fmt.Errorf("unsupported local text provider")
	}
	if !options.Execute {
		report.Findings = append(report.Findings, provider+" text stream not executed; mock text stream used")
		return runMockLocalVoiceLoopbackTextStream(report)
	}
	result, err := runLocalVoiceLoopbackTextStreamCompletion(ctx, prompt, provider, options)
	if err != nil {
		fallbackProvider := normalizeLocalVoiceLoopbackProvider(options.FallbackProvider)
		if fallbackProvider == "" || fallbackProvider == provider {
			report.Findings = append(report.Findings, provider+" text stream failed")
			return "", err
		}
		if fallbackErr := validateLocalVoiceLoopbackTextProvider(options.Env, fallbackProvider); fallbackErr != nil {
			report.Findings = append(report.Findings, "local text fallback provider is not executable")
			return "", err
		}
		report.Findings = append(report.Findings, "provider_fallback_used")
		report.TextStreamFallbackUsed = true
		report.TextStreamFallbackProvider = fallbackProvider
		report.TextStreamFallbackReason = "primary_failed"
		result, err = runLocalVoiceLoopbackTextStreamCompletion(ctx, prompt, fallbackProvider, options)
		if err != nil {
			report.Findings = append(report.Findings, fallbackProvider+" text stream failed")
			return "", err
		}
	}
	applyLocalVoiceLoopbackTextStreamResult(report, result)
	return result.ContentText, nil
}

func normalizeLocalVoiceLoopbackProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return strings.ReplaceAll(provider, "-", "_")
}

func validateLocalVoiceLoopbackTextProvider(env []string, provider string) error {
	profile, _, ok := providers.ProviderProfileByNameFromEnv(env, provider)
	if !ok || profile.Family != providers.ProviderFamilyTextStream {
		return fmt.Errorf("unsupported local text provider")
	}
	if !profile.RouteEligible {
		return fmt.Errorf("unsupported local text provider")
	}
	if profile.Protocol != "openai_chat_completions" && profile.Protocol != "ollama_chat" {
		return fmt.Errorf("unsupported local text provider")
	}
	return nil
}

func runLocalVoiceLoopbackTextStreamCompletion(ctx context.Context, prompt string, provider string, options localVoiceLoopbackTextStreamOptions) (providers.TextStreamCompletionResult, error) {
	return providers.RunTextStreamCompletionFromEnv(ctx, options.Env, providers.TextStreamCompletionOptions{
		ProviderName: provider,
		Prompt:       fastCompanionTextStreamPrompt(prompt),
		MaxTokens:    fastCompanionTextStreamMaxTokens,
		Client:       options.Client,
	})
}

func applyLocalVoiceLoopbackTextStreamResult(report *localVoiceLoopbackReport, result providers.TextStreamCompletionResult) {
	report.TextStreamProvider = result.Provider
	report.TextStreamFamily = string(result.Family)
	report.TextStreamExecuted = true
	report.TextStreamEndpointHost = result.EndpointHost
	report.TextStreamFirstContentMS = result.FirstContentMS
	report.TextStreamContentDeltas = result.ContentDeltaCount
	report.TextStreamReasoningDeltas = result.ReasoningDeltaCount
	report.TextStreamDone = result.Done
}
func fastCompanionTextStreamPrompt(transcript string) string {
	cleaned := strings.TrimSpace(transcript)
	if cleaned == "" {
		cleaned = "我在。"
	}
	return "你是 A21 桌面伙伴。用中文不超过12个字自然回应，不要解释，不要列点。用户说：" + cleaned
}

func fastCompanionVoicePreview(text string) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if cleaned == "" {
		return ""
	}
	if idx := firstSentenceBoundary(cleaned); idx > 0 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	runes := []rune(cleaned)
	if len(runes) <= fastCompanionAnswerVoicePreviewMaxRunes {
		return cleaned
	}
	if !containsNonASCII(cleaned) {
		return cleaned
	}
	preview := string(runes[:fastCompanionAnswerVoicePreviewMaxRunes])
	preview = strings.TrimRight(preview, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	return strings.TrimSpace(preview)
}

func containsNonASCII(text string) bool {
	for _, r := range text {
		if r > 127 {
			return true
		}
	}
	return false
}

func firstSentenceBoundary(text string) int {
	first := -1
	for _, boundary := range []string{"。", "！", "？", "!", "?"} {
		if idx := strings.Index(text, boundary); idx >= 0 && (first < 0 || idx < first) {
			first = idx
		}
	}
	return first
}

func runMockLocalVoiceLoopbackTextStream(report *localVoiceLoopbackReport) (string, error) {
	providerStart := time.Now()
	streamResult, err := providers.ParseOpenAICompatibleTextStream(strings.NewReader(strings.Join([]string{
		`data: {"choices":[{"delta":{"reasoning":"classify loopback"}}]}`,
		`data: {"choices":[{"delta":{"content":"A21 loopback response"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")))
	if err != nil {
		report.Findings = append(report.Findings, "mock text stream parse failed")
		return "", err
	}
	report.TextStreamProvider = "mock_text_stream"
	report.TextStreamFamily = string(providers.ProviderFamilyTextStream)
	report.TextStreamExecuted = false
	report.TextStreamFirstContentMS = elapsedReportMS(providerStart)
	report.TextStreamContentDeltas = streamResult.ContentDeltaCount
	report.TextStreamReasoningDeltas = streamResult.ReasoningDeltaCount
	report.TextStreamDone = streamResult.Done
	return streamResult.ContentText(), nil
}
func writeLocalVoiceLoopbackReport(outputDir string, report localVoiceLoopbackReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-local-voice-loopback-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONLocalVoiceLoopback(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONLocalVoiceLoopback(writer io.Writer, report localVoiceLoopbackReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
