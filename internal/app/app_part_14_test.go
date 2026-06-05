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

func TestRunLocalVoiceLoopbackFallsBackToConfiguredTextProviderWithoutLeakingContent(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-fallback-test.wav")
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
	var sawPrimary bool
	var sawFallback bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/primary/"):
			sawPrimary = true
			http.Error(w, `primary failed sk-a21-primary-secret`, http.StatusServiceUnavailable)
		case strings.HasPrefix(r.URL.Path, "/fallback/"):
			sawFallback = true
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(strings.Join([]string{
				`data: {"choices":[{"delta":{"reasoning":"fallback reasoning must not leak"}}]}`,
				`data: {"choices":[{"delta":{"content":"兜底已接管"}}]}`,
				`data: [DONE]`,
				``,
			}, "\n")))
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_loopback_fallback",
		"label": "A21 loopback fallback",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_LOOPBACK_FALLBACK_API_KEY",
		"model_env": "A21_LOOPBACK_FALLBACK_MODEL",
		"base_url_env": "A21_LOOPBACK_FALLBACK_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-primary-secret")
	t.Setenv("A21_DEEPSEEK_BASE_URL", server.URL+"/primary")
	t.Setenv("A21_LOOPBACK_FALLBACK_API_KEY", "sk-a21-fallback-secret")
	t.Setenv("A21_LOOPBACK_FALLBACK_MODEL", "hidden-fallback-model")
	t.Setenv("A21_LOOPBACK_FALLBACK_BASE_URL", server.URL+"/fallback")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--text-provider", "deepseek", "--fallback-text-provider", "a21_loopback_fallback", "--execute-text-provider", "--text", "用户原文不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawPrimary || !sawFallback {
		t.Fatalf("saw primary/fallback = %v/%v, want both", sawPrimary, sawFallback)
	}
	if ttsInput != "兜底已接管" {
		t.Fatalf("tts input = %q, want fallback provider voice preview", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"text_stream_provider": "a21_loopback_fallback"`,
		`"text_stream_fallback_used": true`,
		`"text_stream_fallback_provider": "a21_loopback_fallback"`,
		`"text_stream_fallback_reason": "primary_failed"`,
		`"text_stream_executed": true`,
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
	for _, forbidden := range []string{profilePath, filepath.Dir(profilePath), server.URL, "sk-a21-primary-secret", "sk-a21-fallback-secret", "hidden-fallback-model", "用户原文不要进报告", "兜底已接管", "fallback reasoning", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("loopback fallback leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackRecordsLocalAckSeparatelyFromProviderAnswer(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	var ttsInputs []string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInputs = append(ttsInputs, options.Text)
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-local-voice-loopback-ack-test-%d.wav", len(ttsInputs)))
		if err := os.WriteFile(outputPath, []byte("RIFF-a21"), 0o644); err != nil {
			return audio.LocalTTSReport{}, err
		}
		firstAudioMS := float64(13)
		if len(ttsInputs) == 1 {
			firstAudioMS = 7
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
			DurationMS:      firstAudioMS,
			TTSFirstAudioMS: firstAudioMS,
		}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"content":"这是正式回答"}}]}`,
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
	if len(ttsInputs) != 2 {
		t.Fatalf("tts calls = %d, want local ack + provider answer: %#v", len(ttsInputs), ttsInputs)
	}
	if !strings.Contains(ttsInputs[0], "我在") || ttsInputs[0] == "这是正式回答" {
		t.Fatalf("first TTS input should be local ack, got %q", ttsInputs[0])
	}
	if ttsInputs[1] != "这是正式回答" {
		t.Fatalf("second TTS input = %q, want provider answer", ttsInputs[1])
	}
	for _, want := range []string{
		`"local_ack_enabled": true`,
		`"local_ack_tts_first_audio_ms": 7`,
		`"local_ack_first_audio_total_ms"`,
		`"answer_first_audio_total_p95_ms"`,
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
	var report localVoiceLoopbackReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatal(err)
	}
	if report.AnswerFirstAudioP95MS < 20 {
		t.Fatalf("answer first audio total = %.3f, want local ack gate + answer TTS", report.AnswerFirstAudioP95MS)
	}
	for _, forbidden := range []string{"sk-a21-secret", "用户原文不要进报告", "这是正式回答", "我在", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunLocalVoiceLoopbackCanUseSherpaASRWithoutLeakingTranscript(t *testing.T) {
	originalTTS := synthesizeMacOSSay
	originalASR := runSherpaONNXASR
	t.Cleanup(func() {
		synthesizeMacOSSay = originalTTS
		runSherpaONNXASR = originalASR
	})
	runSherpaONNXASR = func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
		if options.ModelDir != "/tmp/a21-asr-model" || options.Family != "paraformer" || options.WAVPath != "/tmp/a21-asr.wav" {
			t.Fatalf("asr options = %+v", options)
		}
		return audio.LocalASRResult{
			Transcript: "真实转写不要进报告",
			Report: audio.LocalASRReport{
				SchemaVersion:    "a21.audio.local_asr.v1",
				GeneratedAtMS:    time.Now().UnixMilli(),
				Status:           "passed",
				Provider:         "sherpa_onnx",
				Engine:           "paraformer",
				ModelDir:         "a21-asr-model",
				WAVName:          "a21-asr.wav",
				InputDurationMS:  1200,
				DecodeDurationMS: 88,
				RealTimeFactor:   0.073,
				TextChars:        9,
				TranscriptPolicy: "transcript_not_recorded",
			},
		}, nil
	}
	var ttsInput string
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		ttsInput = options.Text
		outputPath := filepath.Join(options.OutputDir, "a21-local-voice-loopback-asr-test.wav")
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
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"local-voice-loopback", "--engine", "macos_say", "--asr-provider", "sherpa_onnx", "--asr-family", "paraformer", "--asr-model-dir", "/tmp/a21-asr-model", "--asr-wav", "/tmp/a21-asr.wav", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if ttsInput != "A21 loopback response" {
		t.Fatalf("tts input = %q, want mock text stream response", ttsInput)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"asr_provider": "sherpa_onnx"`,
		`"asr_engine": "paraformer"`,
		`"asr_model_dir": "a21-asr-model"`,
		`"asr_wav_name": "a21-asr.wav"`,
		`"asr_text_chars": 9`,
		`"asr_transcript_policy": "transcript_not_recorded"`,
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
	for _, forbidden := range []string{"真实转写不要进报告", "/tmp/a21-asr-model", "/tmp/a21-asr.wav", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("loopback report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanLocalTTSPlaybackSendsRedactedAudioChunks(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			DeviceID    string `json:"device_id"`
			Text        string `json:"text"`
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.DeviceID != "stackchan-001" {
			t.Fatalf("request = %+v", request)
		}
		if len(request.AudioChunks) > 0 && request.Text != "A21 LOCAL TTS" {
			t.Fatalf("audio request text = %q", request.Text)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      33,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{`"schema_version": "a21.stackchan_local_tts_playback.v1"`, `"status": "passed"`, `"tts_provider": "sherpa_onnx"`, `"playback_chunks": 2`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "不要写进报告") {
		t.Fatalf("playback report leaked text: %s", stdout.String())
	}
}

func TestRunStackChanLocalTTSPlaybackSupportsVoiceClonePersonaReport(t *testing.T) {
	original := synthesizeVoiceCloneCLI
	t.Cleanup(func() { synthesizeVoiceCloneCLI = original })
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			DeviceID    string `json:"device_id"`
			Text        string `json:"text"`
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.DeviceID != "stackchan-001" {
			t.Fatalf("request = %+v", request)
		}
		if len(request.AudioChunks) > 0 && request.Text != "A21 LOCAL TTS" {
			t.Fatalf("audio request text = %q", request.Text)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeVoiceCloneCLI = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		if options.Text != "不要写进报告" ||
			options.VoiceCloneCommand != "/a21/bin/a21-index-tts2-wrapper" ||
			options.VoiceCloneModel != "Index-TTS2" ||
			options.VoiceCloneReferenceAudioPath != refAudio ||
			options.VoiceCloneReferenceText != "参考文本不能进报告" ||
			options.VoiceClonePersona != "A21 Workmate" ||
			options.VoiceCloneStyle != "Warm-Pro" {
			t.Fatalf("clone options = %+v", options)
		}
		outputPath := filepath.Join(options.OutputDir, "a21-voice-clone-playback-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
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
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      29,
			TTSFirstAudioMS: 29,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"stackchan-local-tts-playback",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--engine", "voice_clone_cli",
		"--text", "不要写进报告",
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
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{
		`"tts_provider": "voice_clone_cli"`,
		`"tts_engine": "voice_clone_cli"`,
		`"tts_model": "index_tts2"`,
		`"tts_voice": "a21_workmate"`,
		`"tts_voice_persona": "a21_workmate"`,
		`"tts_style_profile": "warm_pro"`,
		`"tts_reference_audio": "a21-persona-reference.wav"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{server.URL, "不要写进报告", "参考文本不能进报告", refAudio, "Authorization", "Bearer", "raw_audio", "data_base64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("voice clone playback report leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunStackChanLocalTTSPlaybackPrerollsInitialAudioBuffer(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var batchSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			batchSizes = append(batchSizes, len(request.AudioChunks))
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-preroll-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*10))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     6400,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      33,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if got, want := fmt.Sprint(batchSizes), "[8 8]"; got != want {
		t.Fatalf("audio batch sizes = %s, want %s", got, want)
	}
}

func TestRunStackChanLocalTTSPlaybackKeepsSteadyBatchesLargeEnoughForSpeakerPump(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var batchSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			batchSizes = append(batchSizes, len(request.AudioChunks))
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-steady-batch-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*20))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     12800,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      400,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if got, want := fmt.Sprint(batchSizes), "[8 8 8]"; got != want {
		t.Fatalf("audio batch sizes = %s, want %s", got, want)
	}
}

func TestRunStackChanLocalTTSPlaybackPacesOfficialCodecBridge(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.AudioChunks) > 0 {
			requestTimes = append(requestTimes, time.Now())
		}
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, "a21-sherpa-playback-prebuffer-test.wav")
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 320*40))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "sherpa_onnx",
			Engine:          "vits_icefall_zh_aishell3",
			Voice:           "sid_21",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     25600,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      800,
			TTSFirstAudioMS: 33,
		}, nil
	}
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "sherpa_onnx", "--text", "不要写进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if len(requestTimes) < 5 {
		t.Fatalf("audio request count = %d, want at least 5", len(requestTimes))
	}
	for i := 1; i <= 2; i++ {
		if gap := requestTimes[i].Sub(requestTimes[i-1]); gap < 120*time.Millisecond {
			t.Fatalf("official codec bridge pacing gap %d = %s, want >= 120ms", i, gap)
		}
	}
}

func TestRunStackChanLocalTTSPlaybackCanSendExistingA21WAV(t *testing.T) {
	original := synthesizeSherpaONNX
	t.Cleanup(func() { synthesizeSherpaONNX = original })
	synthesizeSherpaONNX = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		t.Fatal("existing WAV playback must not synthesize TTS")
		return audio.LocalTTSReport{}, nil
	}
	var receivedChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices/control" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			AudioChunks []struct {
				DataBase64 string `json:"data_base64"`
			} `json:"audio_chunks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		receivedChunks += len(request.AudioChunks)
		fmt.Fprint(w, `{"trace_id":"a21-trace-local","session_id":"a21-session-local","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "a21-existing-playback.wav")
	writeAppTestWAV(t, wavPath, 16000, bytes.Repeat([]byte{1, 0}, 320*3))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-local-tts-playback", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--wav", wavPath, "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if receivedChunks != 8 {
		t.Fatalf("received chunks = %d, want 8 padded speaker-batch chunks", receivedChunks)
	}
	for _, want := range []string{`"tts_provider": "wav_file"`, `"tts_audio_path": "a21-existing-playback.wav"`, `"playback_chunks": 3`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanFastCompanionTurnHelpListsLocalOllama(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{"mock_text_stream", "deepseek", "local_ollama"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanFastCompanionTurnDeliversAckAndAnswerWithoutLeakingText(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-fast-companion-%d.wav", time.Now().UnixNano()))
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	var audioRequests int
	var idleRequests int
	var totalChunks int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/devices/control":
			var request struct {
				DeviceID    string `json:"device_id"`
				State       string `json:"state"`
				Text        string `json:"text"`
				AudioChunks []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.DeviceID != "stackchan-001" {
				t.Fatalf("device id = %q", request.DeviceID)
			}
			if len(request.AudioChunks) > 0 {
				audioRequests++
				totalChunks += len(request.AudioChunks)
				if request.Text != "A21 LOCAL TTS" {
					t.Fatalf("audio request text = %q", request.Text)
				}
			} else if request.State == "idle" {
				idleRequests++
			}
			fmt.Fprint(w, `{"trace_id":"a21-trace-fast","session_id":"a21-session-fast","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--direct-source-ip", "127.0.0.1", "--device-id", "stackchan-001", "--engine", "macos_say", "--text", "用户输入不要进报告", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if audioRequests < 2 || totalChunks < 4 {
		t.Fatalf("audio requests/chunks = %d/%d, want ack and answer playback", audioRequests, totalChunks)
	}
	if idleRequests != 1 {
		t.Fatalf("idle requests = %d, want final playback clear", idleRequests)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "passed"`,
		`"direct_source_ip": "127.0.0.1"`,
		`"device_online": true`,
		`"local_ack_playback_chunks": 2`,
		`"answer_playback_chunks": 2`,
		`"playback_cleared": true`,
		`"m3_candidate": false`,
		`"listen_source": "host_fixture"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
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
	for _, forbidden := range []string{"用户输入不要进报告", "A21 loopback response", "嗯，我在", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("fast companion report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanFastCompanionTurnWritesFailedReportWhenGatewayUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		http.Error(w, "gateway not ready", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--text", "不要泄漏这句话", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "failed"`,
		`"device_online": false`,
		`"m3_candidate": false`,
		`"gateway device report failed"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
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
	for _, forbidden := range []string{"不要泄漏这句话", "Authorization", "Bearer", "sk-"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("failed report leaked %q: stdout=%s stderr=%s report=%s", forbidden, stdout.String(), stderr.String(), reportData)
		}
	}
}
