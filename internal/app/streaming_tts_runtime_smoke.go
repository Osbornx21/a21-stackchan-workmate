package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/providers"
)

const streamingTTSRuntimeSmokeSchemaVersion = "a21.audio.streaming_tts_runtime_smoke.v1"
const streamingTTSRuntimeSmokeText = "A21 realtime TTS runtime smoke."

var streamingTTSRuntimeSmokeDialer providers.RealtimeDialer

type streamingTTSRuntimeSmokeOptions struct {
	Execute   bool
	OutputDir string
	Timeout   time.Duration
	Env       []string
	Dialer    providers.RealtimeDialer
}

type streamingTTSRuntimeSmokeReport struct {
	SchemaVersion                   string   `json:"schema_version"`
	GeneratedAtMS                   int64    `json:"generated_at_ms"`
	Status                          string   `json:"status"`
	Provider                        string   `json:"provider"`
	EvidenceMode                    string   `json:"evidence_mode"`
	ExecuteRequested                bool     `json:"execute_requested"`
	Executed                        bool     `json:"executed"`
	ProviderConfigured              bool     `json:"provider_configured"`
	ProfileEnv                      string   `json:"profile_env"`
	APIKeyEnv                       string   `json:"api_key_env"`
	ModelEnv                        string   `json:"model_env"`
	VoiceEnv                        string   `json:"voice_env"`
	SampleRateEnv                   string   `json:"sample_rate_env"`
	MissingEnv                      []string `json:"missing_env,omitempty"`
	SampleRateHz                    int      `json:"sample_rate_hz"`
	Channels                        int      `json:"channels"`
	TargetChunkDurationMS           int      `json:"target_chunk_duration_ms"`
	SessionUpdateSent               bool     `json:"session_update_sent"`
	TextAppendSent                  bool     `json:"text_append_sent"`
	TextDoneSent                    bool     `json:"text_done_sent"`
	FirstProviderAudioDeltaObserved bool     `json:"first_provider_audio_delta_observed"`
	FirstAudioDeltaBeforeEOF        bool     `json:"first_audio_delta_before_eof"`
	ProviderEOFObserved             bool     `json:"provider_eof_observed"`
	Exact60MSPCM16MonoChunks        int      `json:"exact_60ms_pcm16_mono_chunks"`
	InvalidAudioDeltaEvents         int      `json:"invalid_audio_delta_events"`
	DurationMS                      float64  `json:"duration_ms,omitempty"`
	FirstAudioDeltaMS               float64  `json:"first_audio_delta_ms,omitempty"`
	WAVFileBoundary                 bool     `json:"wav_file_boundary"`
	TextPolicy                      string   `json:"text_policy"`
	ProviderOutputPolicy            string   `json:"provider_output_policy"`
	AudioPayloadPolicy              string   `json:"audio_payload_policy"`
	CredentialPolicy                string   `json:"credential_policy"`
	NetworkLocatorPolicy            string   `json:"network_locator_policy"`
	LocalPathPolicy                 string   `json:"local_path_policy"`
	Findings                        []string `json:"findings,omitempty"`
	ReportPath                      string   `json:"report_path,omitempty"`
}

func runStreamingTTSRuntimeSmokeCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	return runStreamingTTSRuntimeSmoke(clean, execute, stdout, stderr)
}

func runStreamingTTSRuntimeSmoke(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := streamingTTSRuntimeSmokeOptions{
		Execute:   execute,
		OutputDir: "reports",
		Timeout:   20 * time.Second,
		Env:       os.Environ(),
		Dialer:    streamingTTSRuntimeSmokeDialer,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 streaming-tts-runtime-smoke [--execute] [--timeout-ms 20000] [--output-dir reports]")
			return 0
		case "--timeout-ms":
			raw := ""
			if !readStringOption(args, &i, stderr, "--timeout-ms", &raw) {
				return 2
			}
			timeoutMS, err := parsePositiveIntOption(raw, "--timeout-ms")
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
			options.Timeout = time.Duration(timeoutMS) * time.Millisecond
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown streaming-tts-runtime-smoke option %q\n", args[i])
			return 2
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "streaming TTS runtime smoke report dir invalid: %v\n", err)
		return 1
	}
	report := buildStreamingTTSRuntimeSmokeReport(context.Background(), options)
	reportPath, err := writeStreamingTTSRuntimeSmokeReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write streaming TTS runtime smoke report: %v\n", err)
		return 1
	}
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONStreamingTTSRuntimeSmoke(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode streaming TTS runtime smoke report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}

func buildStreamingTTSRuntimeSmokeReport(ctx context.Context, options streamingTTSRuntimeSmokeOptions) streamingTTSRuntimeSmokeReport {
	env := options.Env
	if env == nil {
		env = os.Environ()
	}
	report := streamingTTSRuntimeSmokeReport{
		SchemaVersion:         streamingTTSRuntimeSmokeSchemaVersion,
		GeneratedAtMS:         time.Now().UnixMilli(),
		Status:                "blocked",
		Provider:              "doubao_tts_realtime",
		EvidenceMode:          "runtime_realtime_tts_smoke",
		ExecuteRequested:      options.Execute,
		ProfileEnv:            "A21_TTS_FAST_PROFILE",
		APIKeyEnv:             "A21_DOUBAO_API_KEY",
		ModelEnv:              "A21_DOUBAO_TTS_MODEL",
		VoiceEnv:              "A21_DOUBAO_TTS_VOICE",
		SampleRateEnv:         "A21_DOUBAO_TTS_SAMPLE_RATE_HZ",
		SampleRateHz:          streamingTTSRuntimeSmokeSampleRate(env),
		Channels:              1,
		TargetChunkDurationMS: 60,
		TextPolicy:            "fixed_a21_probe_text_not_recorded",
		ProviderOutputPolicy:  "provider_output_text_not_recorded",
		AudioPayloadPolicy:    "payload_bytes_not_recorded",
		CredentialPolicy:      "credential_values_not_recorded",
		NetworkLocatorPolicy:  "network_locators_not_recorded",
		LocalPathPolicy:       "local_paths_not_recorded",
		WAVFileBoundary:       false,
	}
	if !options.Execute {
		report.Findings = append(report.Findings, "execute_flag_required")
		return report
	}
	report.MissingEnv = streamingTTSRuntimeSmokeMissingEnv(env)
	if len(report.MissingEnv) > 0 {
		report.Findings = append(report.Findings, "missing_required_env")
		return report
	}
	report.ProviderConfigured = true
	report.Executed = true
	if options.Timeout <= 0 {
		options.Timeout = 20 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	started := time.Now()
	if err := executeStreamingTTSRuntimeSmoke(runCtx, env, options.Dialer, &report, started); err != nil {
		report.DurationMS = elapsedStreamingTTSRuntimeSmokeMS(started)
		report.Status = streamingTTSRuntimeSmokeStatusForFinding(err.Error())
		report.Findings = append(report.Findings, err.Error())
		return report
	}
	report.DurationMS = elapsedStreamingTTSRuntimeSmokeMS(started)
	report.Status = "passed"
	return report
}

func executeStreamingTTSRuntimeSmoke(ctx context.Context, env []string, dialer providers.RealtimeDialer, report *streamingTTSRuntimeSmokeReport, started time.Time) error {
	provider := providers.NewDoubaoRealtimeTTSProviderFromEnv(env, dialer)
	session, err := provider.StartRealtimeTTSSession(ctx, providers.VoiceSession{
		TraceID:   "a21-trace-streaming-tts-runtime-smoke",
		SessionID: "a21-session-streaming-tts-runtime-smoke",
		DeviceID:  "a21-host-runtime-smoke",
	})
	if err != nil {
		return fmt.Errorf("provider_runtime_failed")
	}
	defer session.Close(context.Background())
	report.SessionUpdateSent = true
	report.SampleRateHz = session.OutputSampleRateHz()
	if err := session.SendText(ctx, streamingTTSRuntimeSmokeText); err != nil {
		return fmt.Errorf("provider_runtime_failed")
	}
	report.TextAppendSent = true
	if err := session.TextDone(ctx); err != nil {
		return fmt.Errorf("provider_runtime_failed")
	}
	report.TextDoneSent = true
	chunker := newStreamingTTSRuntimeSmokeChunker(report.SampleRateHz)
	for {
		event, ok, err := session.ReadVoiceEvent(ctx)
		if err != nil {
			if err == io.EOF {
				report.ProviderEOFObserved = true
				break
			}
			if ctx.Err() != nil {
				return fmt.Errorf("provider_runtime_timeout")
			}
			return fmt.Errorf("provider_runtime_failed")
		}
		if !ok || event.Audio == nil {
			continue
		}
		report.FirstProviderAudioDeltaObserved = true
		if report.FirstAudioDeltaMS == 0 {
			report.FirstAudioDeltaMS = elapsedStreamingTTSRuntimeSmokeMS(started)
		}
		chunks, err := chunker.AppendBase64(event.Audio.DataBase64)
		if err != nil {
			report.InvalidAudioDeltaEvents++
			return fmt.Errorf("provider_audio_delta_invalid")
		}
		for _, chunk := range chunks {
			if chunk.exact60MSPCM16Mono {
				report.Exact60MSPCM16MonoChunks++
				report.FirstAudioDeltaBeforeEOF = true
			}
		}
		if report.Exact60MSPCM16MonoChunks > 0 {
			return nil
		}
	}
	if !report.FirstProviderAudioDeltaObserved {
		return fmt.Errorf("provider_audio_delta_missing")
	}
	return fmt.Errorf("exact_60ms_pcm16_mono_chunk_missing")
}

func streamingTTSRuntimeSmokeMissingEnv(env []string) []string {
	var missing []string
	if strings.TrimSpace(streamingTTSRuntimeSmokeEnvValue(env, "A21_TTS_FAST_PROFILE")) != "doubao_tts_realtime" {
		missing = append(missing, "A21_TTS_FAST_PROFILE")
	}
	for _, name := range []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_TTS_MODEL", "A21_DOUBAO_TTS_VOICE"} {
		if strings.TrimSpace(streamingTTSRuntimeSmokeEnvValue(env, name)) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func streamingTTSRuntimeSmokeEnvValue(env []string, name string) string {
	prefix := name + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func streamingTTSRuntimeSmokeSampleRate(env []string) int {
	switch strings.TrimSpace(streamingTTSRuntimeSmokeEnvValue(env, "A21_DOUBAO_TTS_SAMPLE_RATE_HZ")) {
	case "24000":
		return 24000
	case "48000":
		return 48000
	default:
		return 16000
	}
}

func streamingTTSRuntimeSmokeStatusForFinding(finding string) string {
	switch finding {
	case "provider_runtime_timeout", "provider_runtime_failed", "provider_audio_delta_invalid":
		return "failed"
	default:
		return "blocked"
	}
}

type streamingTTSRuntimeSmokeChunk struct {
	exact60MSPCM16Mono bool
}

type streamingTTSRuntimeSmokeChunker struct {
	sampleRateHz int
	frameBytes   int
	buffer       []byte
}

func newStreamingTTSRuntimeSmokeChunker(sampleRateHz int) *streamingTTSRuntimeSmokeChunker {
	if sampleRateHz == 0 {
		sampleRateHz = 16000
	}
	return &streamingTTSRuntimeSmokeChunker{
		sampleRateHz: sampleRateHz,
		frameBytes:   sampleRateHz * 60 / 1000 * 2,
	}
}

func (c *streamingTTSRuntimeSmokeChunker) AppendBase64(value string) ([]streamingTTSRuntimeSmokeChunk, error) {
	pcm, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil || len(pcm) == 0 {
		return nil, fmt.Errorf("invalid audio delta")
	}
	if len(pcm)%2 != 0 {
		return nil, fmt.Errorf("invalid audio delta")
	}
	c.buffer = append(c.buffer, pcm...)
	var chunks []streamingTTSRuntimeSmokeChunk
	for len(c.buffer) >= c.frameBytes {
		c.buffer = c.buffer[c.frameBytes:]
		chunks = append(chunks, streamingTTSRuntimeSmokeChunk{exact60MSPCM16Mono: true})
	}
	return chunks, nil
}

func writeStreamingTTSRuntimeSmokeReport(outputDir string, report streamingTTSRuntimeSmokeReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-streaming-tts-runtime-smoke-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.OpenFile(reportPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(filepath.Clean(reportPath))
	if err := writeJSONStreamingTTSRuntimeSmoke(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStreamingTTSRuntimeSmoke(writer io.Writer, report streamingTTSRuntimeSmokeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func elapsedStreamingTTSRuntimeSmokeMS(started time.Time) float64 {
	return float64(time.Since(started).Microseconds()) / 1000
}
