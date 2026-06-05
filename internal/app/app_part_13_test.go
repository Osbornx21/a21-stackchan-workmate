package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
)

func TestNormalizeLocalTTSEngineSupportsIflytekAliases(t *testing.T) {
	for _, raw := range []string{"iflytek_tts", "iflytek-tts", "iflytek", "xfyun", "xfyun_tts"} {
		got, err := normalizeLocalTTSEngine(raw)
		if err != nil {
			t.Fatalf("normalize %q: %v", raw, err)
		}
		if got != "iflytek_tts" {
			t.Fatalf("normalize %q = %q, want iflytek_tts", raw, got)
		}
	}
}

func TestRunLocalTTSSmokeRejectsLegacyReportDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-tts-smoke", "--output-dir", filepath.Join(t.TempDir(), "v21-reports")}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestRunLocalASRSmokeWritesRedactedReport(t *testing.T) {
	original := runSherpaONNXASRSmoke
	t.Cleanup(func() { runSherpaONNXASRSmoke = original })
	runSherpaONNXASRSmoke = func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRReport, error) {
		if options.ModelDir != "/tmp/a21-asr-model" || options.Family != "paraformer" || options.WAVPath != "/tmp/a21-asr-model/test_wavs/private.wav" {
			t.Fatalf("asr options = %+v", options)
		}
		return audio.LocalASRReport{
			SchemaVersion:    "a21.audio.local_asr.v1",
			GeneratedAtMS:    time.Now().UnixMilli(),
			Status:           "passed",
			Provider:         "sherpa_onnx",
			Engine:           "paraformer",
			ModelDir:         "a21-asr-model",
			WAVName:          "private.wav",
			InputDurationMS:  1200,
			DecodeDurationMS: 90,
			RealTimeFactor:   0.075,
			TextChars:        8,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-asr-smoke", "--engine", "sherpa_onnx", "--family", "paraformer", "--model-dir", "/tmp/a21-asr-model", "--wav", "/tmp/a21-asr-model/test_wavs/private.wav", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.audio.local_asr.v1"`,
		`"status": "passed"`,
		`"provider": "sherpa_onnx"`,
		`"engine": "paraformer"`,
		`"model_dir": "a21-asr-model"`,
		`"wav_name": "private.wav"`,
		`"text_chars": 8`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-asr-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private transcript", "/tmp/a21-asr-model", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("local ASR smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalASRSmokeRejectsLegacyReportDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-asr-smoke", "--output-dir", filepath.Join(t.TempDir(), "v21-reports")}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestWriteLocalASRSmokeReportUsesUniqueNamesForRapidWrites(t *testing.T) {
	dir := t.TempDir()
	report := audio.LocalASRReport{
		SchemaVersion:    "a21.audio.local_asr.v1",
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "passed",
		Provider:         "sherpa_onnx",
		Engine:           "paraformer",
		TranscriptPolicy: "transcript_not_recorded",
	}

	first, err := writeLocalASRSmokeReport(dir, report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeLocalASRSmokeReport(dir, report)
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatalf("rapid report writes used the same path: %s", first)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-asr-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("reports = %d, want 2: %v", len(matches), matches)
	}
}

func TestRunLocalVoiceLoopbackHelpListsLocalOllama(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{"mock_text_stream", "deepseek", "local_ollama"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q: %s", want, stdout.String())
		}
	}
}

func TestFastCompanionVoicePreviewKeepsFirstSpeechShort(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "sentence boundary",
			in:   "先稳住。后面这句不要进首段语音。",
			want: "先稳住",
		},
		{
			name: "rune cap",
			in:   "这是一个很长很长的中文回复内容",
			want: "这是一个很长很长的中文回",
		},
		{
			name: "ascii tail",
			in:   "这是来自本地 Ollama 的回复",
			want: "这是来自本地",
		},
		{
			name: "ascii only",
			in:   "A21 loopback response",
			want: "A21 loopback response",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fastCompanionVoicePreview(tc.in); got != tc.want {
				t.Fatalf("preview = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFastCompanionTextStreamPromptUsesPersonalityRuntimeAssets(t *testing.T) {
	prompt := fastCompanionTextStreamPrompt("a21 mock transcript")

	for _, want := range []string{
		"Core Identity",
		"Tone Rules",
		"Workmate Mode",
		"不超过12个字",
		"a21 mock transcript",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{
		"Professional Mode",
		"Companion Mode",
		"Post-Meeting Playbook",
		"Failure Overlay",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt included unselected asset %q:\n%s", forbidden, prompt)
		}
	}
}

func TestFastCompanionTextStreamPromptIncludesBoundedMemoryHints(t *testing.T) {
	t.Setenv("A21_MEMORY_USER_PREFERENCES", "偏好短句")
	t.Setenv("A21_MEMORY_SESSION_NOTES", "当前任务是座舱 PRD 评审")

	prompt := fastCompanionTextStreamPrompt("a21 mock transcript")

	for _, want := range []string{
		"Workmate Mode",
		"Memory Hints",
		"user_preference:user_preference_1",
		"偏好短句",
		"session_memory:session_memory_1",
		"当前任务是座舱 PRD 评审",
		"a21 mock transcript",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRunLocalVoiceLoopbackWritesRedactedReport(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      10,
			TTSFirstAudioMS: 10,
			AudioQuality: &audio.PCMQualityReport{
				Status:       "passed",
				Codec:        "pcm_s16le",
				SampleRateHz: 16000,
				Channels:     1,
				PeakAbs:      1600,
				RMSDBFS:      -28.2,
			},
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text", "真实输入不要进报告", "--repeat", "2", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.audio.local_voice_loopback.v1"`,
		`"status": "passed"`,
		`"vad_status": "mock_only"`,
		`"asr_provider": "mock_asr"`,
		`"text_stream_provider": "mock_text_stream"`,
		`"tts_provider": "macos_say"`,
		`"local_ack_audio_quality"`,
		`"tts_audio_quality"`,
		`"repeat": 2`,
		`"tts_first_audio_p50_ms"`,
		`"first_audio_total_p95_ms"`,
		`"barge_in_status": "benchmarked"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"真实输入不要进报告", "A21 loopback response", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackSupportsVoiceClonePersonaReport(t *testing.T) {
	original := synthesizeVoiceCloneCLI
	t.Cleanup(func() { synthesizeVoiceCloneCLI = original })
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ttsInputs []string
	synthesizeVoiceCloneCLI = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInputs = append(ttsInputs, options.Text)
		if options.VoiceCloneCommand != "/a21/bin/a21-index-tts2-wrapper" ||
			options.VoiceCloneModel != "Index-TTS2" ||
			options.VoiceCloneReferenceAudioPath != refAudio ||
			options.VoiceCloneReferenceText != "参考文本不能进报告" ||
			options.VoiceClonePersona != "A21 Workmate" ||
			options.VoiceCloneStyle != "Warm-Pro" {
			t.Fatalf("clone options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-voice-clone-loopback-%d.wav", len(ttsInputs)))
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "voice_clone_cli",
			Engine:          "voice_clone_cli",
			Voice:           "a21_workmate",
			Model:           "index_tts2",
			VoicePersona:    "a21_workmate",
			StyleProfile:    "warm_pro",
			ReferenceAudio:  "a21-persona-reference.wav",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      14,
			TTSFirstAudioMS: 14,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"local-voice-loopback",
		"--engine", "voice_clone_cli",
		"--text", "用户原文不要进报告",
		"--clone-command", "/a21/bin/a21-index-tts2-wrapper",
		"--clone-model", "Index-TTS2",
		"--clone-ref-audio", refAudio,
		"--clone-ref-text", "参考文本不能进报告",
		"--voice-persona", "A21 Workmate",
		"--voice-style", "Warm-Pro",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if len(ttsInputs) != 2 || ttsInputs[0] != "嗯，我在。" || ttsInputs[1] != "A21 loopback response" {
		t.Fatalf("tts inputs = %#v, want local ack and answer preview", ttsInputs)
	}
	for _, want := range []string{
		`"tts_provider": "voice_clone_cli"`,
		`"tts_model": "index_tts2"`,
		`"tts_voice": "a21_workmate"`,
		`"tts_voice_persona": "a21_workmate"`,
		`"tts_style_profile": "warm_pro"`,
		`"tts_reference_audio": "a21-persona-reference.wav"`,
		`"local_ack_tts_provider": "voice_clone_cli"`,
		`"local_ack_tts_model": "index_tts2"`,
		`"local_ack_tts_voice_persona": "a21_workmate"`,
		`"local_ack_tts_style_profile": "warm_pro"`,
		`"local_ack_tts_reference_audio": "a21-persona-reference.wav"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"用户原文不要进报告", "A21 loopback response", "嗯，我在", "参考文本不能进报告", refAudio, "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("voice clone loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseDeepSeekTextStreamWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-deepseek-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MaxTokens int `json:"max_tokens"`
			Messages  []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.MaxTokens != 12 {
			t.Fatalf("max_tokens = %d, want 12", body.MaxTokens)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning":"先识别情绪"}}]}`,
			`data: {"choices":[{"delta":{"content":"收到我会帮你稳住"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DEEPSEEK_BASE_URL", server.URL)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "deepseek", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "收到我会帮你稳住" {
		t.Fatalf("tts input = %q, want provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "deepseek"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 1`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat", "用户原文不要进报告", "收到我会帮你稳住", "先识别情绪", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseCompatibilityTextStreamWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-compat-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "Qwen/Qwen3.5-9B" {
			t.Fatalf("model = %q, want Qwen/Qwen3.5-9B", body.Model)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"content":"收到我会帮你稳住"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_LAB_SILICONFLOW_API_KEY", "sk-a21-siliconflow-secret")
	t.Setenv("A21_SILICONFLOW_MODEL", "Qwen/Qwen3.5-9B")
	t.Setenv("A21_SILICONFLOW_BASE_URL", server.URL)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "siliconflow", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "收到我会帮你稳住" {
		t.Fatalf("tts input = %q, want provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "siliconflow"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"tts_provider": "macos_say"`,
		`local text provider is compatibility-only; product route eligibility is unchanged`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sk-a21-siliconflow-secret", "Qwen/Qwen3.5-9B", "用户原文不要进报告", "收到我会帮你稳住", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseLocalOllamaTextStreamWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-ollama-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path = %q, want /api/chat", r.URL.Path)
		}
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Options struct {
				NumPredict int `json:"num_predict"`
			} `json:"options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "qwen2.5:0.5b" || body.Options.NumPredict != 12 {
			t.Fatalf("ollama body = %+v", body)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		_, _ = w.Write([]byte(strings.Join([]string{
			`{"message":{"content":"收到我会帮你稳住"}}`,
			`{"done":true}`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	t.Setenv("A21_PROVIDER_PRIMARY", "local_ollama")
	t.Setenv("A21_LOCAL_OLLAMA_BASE_URL", server.URL)
	t.Setenv("A21_LOCAL_OLLAMA_MODEL", "qwen2.5:0.5b")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "local_ollama", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "收到我会帮你稳住" {
		t.Fatalf("tts input = %q, want local Ollama voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "local_ollama"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 0`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"qwen2.5:0.5b", "用户原文不要进报告", "收到我会帮你稳住", server.URL, "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseHotPlugTextStreamProfileWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-hotplug-test.wav")
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     8,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      12,
			TTSFirstAudioMS: 12,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		var body struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			Messages  []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "hidden-hotplug-model" || body.MaxTokens != 12 {
			t.Fatalf("provider request body = %+v", body)
		}
		if len(body.Messages) != 1 || !strings.Contains(body.Messages[0].Content, "12个字") || !strings.Contains(body.Messages[0].Content, "a21 mock transcript") {
			t.Fatalf("fast companion prompt not applied: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"reasoning":"不要泄露推理"}}]}`,
			`data: {"choices":[{"delta":{"content":"热插拔已接入"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	t.Cleanup(server.Close)
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_loopback_vendor",
		"label": "A21 loopback vendor",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_LOOPBACK_VENDOR_API_KEY",
		"model_env": "A21_LOOPBACK_VENDOR_MODEL",
		"base_url_env": "A21_LOOPBACK_VENDOR_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_PROVIDER_PRIMARY", "a21_loopback_vendor")
	t.Setenv("A21_LOOPBACK_VENDOR_API_KEY", "sk-a21-hotplug-secret")
	t.Setenv("A21_LOOPBACK_VENDOR_MODEL", "hidden-hotplug-model")
	t.Setenv("A21_LOOPBACK_VENDOR_BASE_URL", server.URL+"/v1")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "a21_loopback_vendor", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "热插拔已接入" {
		t.Fatalf("tts input = %q, want hotplug provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "a21_loopback_vendor"`,
		`"text_stream_executed": true`,
		`"text_stream_content_delta_count": 1`,
		`"text_stream_reasoning_delta_count": 1`,
		`"tts_provider": "macos_say"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-local-voice-loopback-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports = %v, %v", matches, err)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{profilePath, filepath.Dir(profilePath), server.URL, "sk-a21-hotplug-secret", "hidden-hotplug-model", "用户原文不要进报告", "热插拔已接入", "不要泄露推理", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}
