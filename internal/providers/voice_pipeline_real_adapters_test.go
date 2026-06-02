package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/audio"
)

func TestVoicePipelinePCMFramePayloadIsJSONRedacted(t *testing.T) {
	frame := VoicePipelinePCMFrame{
		Seq:          7,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    4,
		PCM16LE:      []byte{1, 2, 3, 4},
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(data)
	for _, forbidden := range []string{"PCM16LE", "AQIDBA==", "payload", "raw_audio"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("frame JSON leaked %q: %s", forbidden, rendered)
		}
	}
	if !strings.Contains(rendered, `"ByteCount":4`) {
		t.Fatalf("frame JSON missing safe metadata: %s", rendered)
	}
}

func TestLocalSherpaONNXASRAdapterBuildsTempWAVAndEmitsPartialFinal(t *testing.T) {
	var wavPath string
	adapter := NewLocalSherpaONNXASRAdapter(LocalSherpaONNXASRAdapterOptions{
		Name:     "sherpa_onnx",
		ModelDir: filepath.Join(t.TempDir(), "a21-asr-model"),
		Runner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
			if err := ctx.Err(); err != nil {
				return audio.LocalASRResult{}, err
			}
			wavPath = options.WAVPath
			data, err := os.ReadFile(options.WAVPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(data[:12], []byte("WAVE")) {
				t.Fatalf("temp wav header = %q, want WAVE", string(data[:12]))
			}
			if !bytes.Contains(data, []byte{1, 0, 2, 0, 3, 0, 4, 0}) {
				t.Fatalf("temp wav does not contain PCM payload")
			}
			return audio.LocalASRResult{
				Report:     audio.LocalASRReport{Status: "passed", TextChars: 4},
				Transcript: "t",
			}, nil
		},
	})

	events, err := adapter.Transcribe(context.Background(), ASRAdapterRequest{
		Session: VoiceSession{TraceID: "a21-trace-asr", SessionID: "a21-session-asr", DeviceID: "stackchan-sim-001"},
		Mode:    "workmate",
		Frames: []VoicePipelinePCMFrame{{
			Seq:          1,
			Codec:        "pcm_s16le",
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   60,
			ByteCount:    8,
			PCM16LE:      []byte{1, 0, 2, 0, 3, 0, 4, 0},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	collected := collectASREvents(t, events)
	if len(collected) != 2 {
		t.Fatalf("events = %#v, want partial and final", collected)
	}
	if collected[0].Final || !collected[1].Final || collected[1].Text != "t" {
		t.Fatalf("events = %#v, want partial then final transcript", collected)
	}
	if wavPath == "" {
		t.Fatal("runner did not receive a wav path")
	}
	if _, err := os.Stat(wavPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp wav stat err = %v, want cleaned temp file", err)
	}
}

func TestLocalSherpaONNXASRAdapterRedactsRunnerErrors(t *testing.T) {
	adapter := NewLocalSherpaONNXASRAdapter(LocalSherpaONNXASRAdapterOptions{
		Runner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
			return audio.LocalASRResult{}, errors.New("failed at a21-input.wav with marker")
		},
	})

	events, err := adapter.Transcribe(context.Background(), ASRAdapterRequest{
		Frames: []VoicePipelinePCMFrame{{
			Codec:        "pcm_s16le",
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   60,
			ByteCount:    2,
			PCM16LE:      []byte{1, 0},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	collected := collectASREvents(t, events)
	if len(collected) != 1 || collected[0].Err == nil {
		t.Fatalf("events = %#v, want one redacted error event", collected)
	}
	rendered := collected[0].Err.Error() + collected[0].Finding
	for _, forbidden := range []string{"marker", "/Users/", "a21-input.wav"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("ASR error leaked %q: %s", forbidden, rendered)
		}
	}
	if !strings.Contains(rendered, "sherpa-onnx ASR adapter failed") {
		t.Fatalf("ASR error = %q, want stable adapter failure", rendered)
	}
}

func TestOpenAICompatibleTextStreamAdapterEmitsContentDeltasWithFakeTransport(t *testing.T) {
	var sawAuth bool
	var sawPrompt bool
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sawAuth = strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		sawPrompt = strings.Contains(string(body), `"p"`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"choices":[{"delta":{"reasoning":"r"}}]}`,
				`data: {"choices":[{"delta":{"content":"a"}}]}`,
				`data: {"choices":[{"delta":{"content":"b"}}]}`,
				`data: [DONE]`,
				``,
			}, "\n"))),
			Request: req,
		}, nil
	})}
	adapter := NewOpenAICompatibleTextStreamAdapter(OpenAICompatibleTextStreamAdapterOptions{
		Name:         "deepseek",
		Env:          []string{"A21_PROVIDER_PRIMARY=deepseek", "A21_LAB_DEEPSEEK_API_KEY=configured-token"},
		ProviderName: "deepseek",
		MaxTokens:    16,
		Client:       client,
	})

	events, err := adapter.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
	if err != nil {
		t.Fatal(err)
	}
	collected := collectTextEvents(t, events)
	var content strings.Builder
	var reasoningCount int
	var done bool
	for _, event := range collected {
		if event.Err != nil {
			t.Fatalf("unexpected text stream error: %v", event.Err)
		}
		switch event.Kind {
		case TextStreamDeltaContent:
			content.WriteString(event.Text)
		case TextStreamDeltaReasoning:
			reasoningCount++
		case TextStreamDeltaDone:
			done = true
		}
	}
	if !sawAuth || !sawPrompt {
		t.Fatalf("saw auth/prompt = %v/%v, want true/true", sawAuth, sawPrompt)
	}
	if content.String() != "ab" || reasoningCount != 1 || !done {
		t.Fatalf("events = %#v, content=%q reasoning=%d done=%v", collected, content.String(), reasoningCount, done)
	}
}

func TestLocalTTSAdapterConvertsWAVToDownlinkReady24KChunks(t *testing.T) {
	adapter := NewLocalTTSAdapter(LocalTTSAdapterOptions{
		Name: "local_sherpa_onnx",
		Synthesizer: func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
			if options.OutputSampleRateHz != 24000 {
				t.Fatalf("output sample rate = %d, want 24000", options.OutputSampleRateHz)
			}
			if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(options.OutputDir, "a21-tts.wav")
			if err := audio.WritePCM16MonoWAV(path, 24000, make([]byte, 4000)); err != nil {
				t.Fatal(err)
			}
			return audio.LocalTTSReport{Status: "passed", OutputPath: path}, nil
		},
	})

	chunks, err := adapter.Synthesize(context.Background(), TTSAdapterRequest{Text: "t"})
	if err != nil {
		t.Fatal(err)
	}
	collected := collectVoiceChunks(t, chunks)
	if len(collected) != 2 {
		t.Fatalf("chunks = %d, want 2", len(collected))
	}
	for _, chunk := range collected {
		if chunk.Codec != "pcm_s16le" || chunk.SampleRateHz != 24000 || chunk.Channels != 1 || chunk.DurationMS != 60 {
			t.Fatalf("chunk = %+v, want pcm_s16le 24k mono 60ms", chunk)
		}
		pcm, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		if len(pcm) != 2880 {
			t.Fatalf("chunk bytes = %d, want 2880", len(pcm))
		}
	}
}

func TestVoicePipelineAdaptersFromEnvDefaultsMockAndSelectsHostLocal(t *testing.T) {
	defaults := VoicePipelineAdaptersFromEnv(nil)
	if defaults.ExecutionMode != "fixture" {
		t.Fatalf("default execution mode = %q, want fixture", defaults.ExecutionMode)
	}
	if defaults.ASR.Name() != "mock-local-asr" || defaults.TextStream.Name() != "mock-text-stream" || defaults.TTS.Name() != "mock-fast-tts" {
		t.Fatalf("default adapters = %s/%s/%s, want mocks", defaults.ASR.Name(), defaults.TextStream.Name(), defaults.TTS.Name())
	}

	host := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx",
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_TTS_FAST_PROFILE=sherpa_onnx",
	}, VoicePipelineAdapterOptions{
		ASRRunner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
			return audio.LocalASRResult{Report: audio.LocalASRReport{Status: "passed"}, Transcript: "ok"}, nil
		},
		TextHTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\n")), Request: req}, nil
		})},
		TTSSynthesizer: func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
			path := filepath.Join(options.OutputDir, "a21-tts.wav")
			_ = os.MkdirAll(options.OutputDir, 0o755)
			_ = audio.WritePCM16MonoWAV(path, 24000, make([]byte, 2880))
			return audio.LocalTTSReport{Status: "passed", OutputPath: path}, nil
		},
	})
	if host.ExecutionMode != "host_local" {
		t.Fatalf("host execution mode = %q, want host_local", host.ExecutionMode)
	}
	if host.ASR.Name() != "sherpa_onnx" || host.TextStream.Name() != "deepseek" || host.TTS.Name() != "sherpa_onnx" {
		t.Fatalf("host adapters = %s/%s/%s", host.ASR.Name(), host.TextStream.Name(), host.TTS.Name())
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsVoiceCloneCLI(t *testing.T) {
	refAudio := filepath.Join(t.TempDir(), "a21-persona-reference.wav")
	if err := os.WriteFile(refAudio, []byte("RIFF-a21-reference"), 0o644); err != nil {
		t.Fatal(err)
	}
	refTextPath := filepath.Join(t.TempDir(), "a21-reference.txt")
	if err := os.WriteFile(refTextPath, []byte("参考文本不能进报告"), 0o600); err != nil {
		t.Fatal(err)
	}
	var captured audio.LocalTTSOptions
	outputDir := t.TempDir()
	command := "/usr/bin/ssh -i /a21-lab/secrets/a21_5080_fixture_ed25519 21@192.168.1.6 powershell -NoProfile -File D:/a21-mainland-latency-lab/outbox/a21-index-tts2-wrapper.ps1"
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_TTS_FAST_PROFILE=voice_clone_cli",
		"A21_VOICE_CLONE_COMMAND=" + command,
		"A21_VOICE_CLONE_MODEL=Index-TTS2",
		"A21_VOICE_CLONE_REF_AUDIO=" + refAudio,
		"A21_VOICE_CLONE_REF_TEXT_FILE=" + refTextPath,
		"A21_VOICE_PERSONA=A21 Workmate",
		"A21_VOICE_STYLE=Warm-Pro",
	}, VoicePipelineAdapterOptions{
		TTSOptions: audio.LocalTTSOptions{OutputDir: outputDir},
		TTSSynthesizer: func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
			captured = options
			if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(options.OutputDir, "a21-voice-clone-adapter.wav")
			if err := audio.WritePCM16MonoWAV(path, 24000, make([]byte, 2880)); err != nil {
				t.Fatal(err)
			}
			return audio.LocalTTSReport{Status: "passed", OutputPath: path}, nil
		},
	})

	if adapters.ExecutionMode != "host_local" || adapters.TTS.Name() != "voice_clone_cli" {
		t.Fatalf("adapters execution/TTS = %q/%q", adapters.ExecutionMode, adapters.TTS.Name())
	}
	chunks, err := adapters.TTS.Synthesize(context.Background(), TTSAdapterRequest{Text: "用户原文不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	collected := collectVoiceChunks(t, chunks)
	if len(collected) != 1 || collected[0].SampleRateHz != 24000 || collected[0].DurationMS != 60 {
		t.Fatalf("chunks = %+v, want one 24k 60ms chunk", collected)
	}
	if captured.Text != "用户原文不进报告" ||
		captured.OutputSampleRateHz != 24000 ||
		captured.VoiceCloneCommand != command ||
		captured.VoiceCloneModel != "Index-TTS2" ||
		captured.VoiceCloneReferenceAudioPath != refAudio ||
		captured.VoiceCloneReferenceText != "" ||
		captured.VoiceCloneReferenceTextPath != refTextPath ||
		captured.VoiceClonePersona != "A21 Workmate" ||
		captured.VoiceCloneStyle != "Warm-Pro" {
		t.Fatalf("captured TTS options = %+v", captured)
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsLocalOllama(t *testing.T) {
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_PROVIDER_PRIMARY=local_ollama",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_LOCAL_OLLAMA_BASE_URL=http://127.0.0.1:11434",
		"A21_LOCAL_OLLAMA_MODEL=qwen2.5:0.5b",
	})

	if adapters.ExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local", adapters.ExecutionMode)
	}
	if adapters.TextStream.Name() != "local_ollama" {
		t.Fatalf("text stream adapter = %s, want local_ollama", adapters.TextStream.Name())
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsHotPlugOpenAITextStreamProfile(t *testing.T) {
	profilePath := writeProviderProfileFile(t, `{
		"name": "a21_voice_lab_text",
		"label": "A21 voice lab text stream",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_VOICE_LAB_TEXT_API_KEY",
		"model_env": "A21_VOICE_LAB_TEXT_MODEL",
		"base_url_env": "A21_VOICE_LAB_TEXT_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`)
	var body map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", req.URL.Path)
		}
		body = readJSONRequestBody(t, req)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\ndata: [DONE]\n")),
			Request:    req,
		}, nil
	})}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_PROVIDER_PROFILES_PATH=" + profilePath,
		"A21_PROVIDER_PRIMARY=a21_voice_lab_text",
		"A21_TEXT_STREAM_PROFILE=a21_voice_lab_text",
		"A21_VOICE_LAB_TEXT_API_KEY=configured-token",
		"A21_VOICE_LAB_TEXT_MODEL=hidden-model",
		"A21_VOICE_LAB_TEXT_BASE_URL=https://a21-provider.invalid/v1",
	}, VoicePipelineAdapterOptions{
		TextHTTPClient: client,
		TextMaxTokens:  15,
	})

	if adapters.ExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local", adapters.ExecutionMode)
	}
	if adapters.TextStream.Name() != "a21_voice_lab_text" {
		t.Fatalf("text stream adapter = %s, want a21_voice_lab_text", adapters.TextStream.Name())
	}
	events, err := adapters.TextStream.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = collectTextEvents(t, events)

	got, ok := body["max_tokens"].(float64)
	if !ok || int(got) != 15 {
		t.Fatalf("max_tokens = %#v, want 15", body["max_tokens"])
	}
}

func TestVoicePipelineRunnerFallsBackToConfiguredTextStreamProfile(t *testing.T) {
	profilePath := writeProviderProfileFile(t, `{
		"name": "a21_voice_fallback",
		"label": "A21 voice fallback text stream",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "streaming_text", "text_stream"],
		"api_key_env": "A21_VOICE_FALLBACK_API_KEY",
		"model_env": "A21_VOICE_FALLBACK_MODEL",
		"base_url_env": "A21_VOICE_FALLBACK_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`)
	var sawPrimary bool
	var sawFallback bool
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "primary.invalid":
			sawPrimary = true
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":"primary down sk-a21-primary-secret"}`)),
				Request:    req,
			}, nil
		case "fallback.invalid":
			sawFallback = true
			body := readJSONRequestBody(t, req)
			if got, ok := body["max_tokens"].(float64); !ok || int(got) != 16 {
				t.Fatalf("fallback max_tokens = %#v, want 16", body["max_tokens"])
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(strings.Join([]string{
					`data: {"choices":[{"delta":{"reasoning":"fallback reasoning must not be stored"}}]}`,
					`data: {"choices":[{"delta":{"content":"fallback voice answer"}}]}`,
					`data: [DONE]`,
					``,
				}, "\n"))),
				Request: req,
			}, nil
		default:
			t.Fatalf("unexpected provider host = %q", req.URL.Host)
			return nil, nil
		}
	})}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_PROVIDER_PROFILES_PATH=" + profilePath,
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_TEXT_STREAM_FALLBACK_PROFILE=a21_voice_fallback",
		"A21_LAB_DEEPSEEK_API_KEY=sk-a21-primary-secret",
		"A21_DEEPSEEK_BASE_URL=https://primary.invalid/v1",
		"A21_VOICE_FALLBACK_API_KEY=sk-a21-fallback-secret",
		"A21_VOICE_FALLBACK_MODEL=hidden-fallback-model",
		"A21_VOICE_FALLBACK_BASE_URL=https://fallback.invalid/v1",
	}, VoicePipelineAdapterOptions{
		TextHTTPClient: client,
		TextMaxTokens:  16,
	})
	result, err := NewVoicePipelineRunner(adapters).Run(context.Background(), VoicePipelineRequest{
		Session: VoiceSession{TraceID: "a21-trace-fallback", SessionID: "a21-session-fallback", DeviceID: "stackchan-test"},
		Mode:    "workmate",
	})

	if err != nil {
		t.Fatal(err)
	}
	if !sawPrimary || !sawFallback {
		t.Fatalf("saw primary/fallback = %v/%v, want both", sawPrimary, sawFallback)
	}
	if result.Status != VoicePipelineStatusCompleted {
		t.Fatalf("status = %q, want completed: %#v", result.Status, result.Report.Findings)
	}
	if result.Report.Fallback == nil || !result.Report.Fallback.Activated || result.Report.Fallback.Provider != "a21_voice_fallback" {
		t.Fatalf("fallback report = %+v, want activated a21_voice_fallback", result.Report.Fallback)
	}
	if result.Report.Selection.LLMFallbackProfile != "a21_voice_fallback" ||
		result.Report.Selection.LLMFallbackProfileEnv != "A21_TEXT_STREAM_FALLBACK_PROFILE" {
		t.Fatalf("selection fallback = %+v", result.Report.Selection)
	}
	rendered, err := json.Marshal(result.Report)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"fallback"`, `"provider":"a21_voice_fallback"`, `"provider_fallback_used"`} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("fallback report missing %q: %s", want, rendered)
		}
	}
	for _, forbidden := range []string{profilePath, "primary.invalid", "fallback.invalid", "sk-a21-primary-secret", "sk-a21-fallback-secret", "hidden-fallback-model", "fallback voice answer", "fallback reasoning", "fixture transcript"} {
		if strings.Contains(string(rendered), forbidden) {
			t.Fatalf("fallback report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestVoicePipelineAdaptersFromEnvAppliesVoiceTextMaxTokensToOpenAICompatibleRequests(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int
	}{
		{name: "default", want: 24},
		{name: "env override", env: "A21_VOICE_TEXT_MAX_TOKENS=40", want: 40},
		{name: "invalid defaults", env: "A21_VOICE_TEXT_MAX_TOKENS=not-a-number", want: 24},
		{name: "too low clamps", env: "A21_VOICE_TEXT_MAX_TOKENS=2", want: 8},
		{name: "too high clamps", env: "A21_VOICE_TEXT_MAX_TOKENS=120", want: 96},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]any
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body = readJSONRequestBody(t, req)
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body:       io.NopCloser(strings.NewReader("data: [DONE]\n")),
					Request:    req,
				}, nil
			})}
			env := []string{
				"A21_PROVIDER_PRIMARY=deepseek",
				"A21_TEXT_STREAM_PROFILE=deepseek",
				"A21_LAB_DEEPSEEK_API_KEY=configured-token",
			}
			if tt.env != "" {
				env = append(env, tt.env)
			}
			adapters := VoicePipelineAdaptersFromEnv(env, VoicePipelineAdapterOptions{TextHTTPClient: client})

			events, err := adapters.TextStream.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
			if err != nil {
				t.Fatal(err)
			}
			_ = collectTextEvents(t, events)

			got, ok := body["max_tokens"].(float64)
			if !ok || int(got) != tt.want {
				t.Fatalf("max_tokens = %#v, want %d", body["max_tokens"], tt.want)
			}
		})
	}
}

func TestVoicePipelineAdaptersFromEnvAppliesVoiceTextMaxTokensToOllamaRequests(t *testing.T) {
	var body map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body = readJSONRequestBody(t, req)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/x-ndjson"}},
			Body:       io.NopCloser(strings.NewReader("{\"done\":true}\n")),
			Request:    req,
		}, nil
	})}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_PROVIDER_PRIMARY=local_ollama",
		"A21_TEXT_STREAM_PROFILE=local_ollama",
		"A21_LOCAL_OLLAMA_BASE_URL=http://127.0.0.1:11434",
		"A21_LOCAL_OLLAMA_MODEL=qwen2.5:0.5b",
		"A21_VOICE_TEXT_MAX_TOKENS=36",
	}, VoicePipelineAdapterOptions{TextHTTPClient: client})

	events, err := adapters.TextStream.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = collectTextEvents(t, events)

	options, ok := body["options"].(map[string]any)
	if !ok {
		t.Fatalf("options = %#v, want object", body["options"])
	}
	got, ok := options["num_predict"].(float64)
	if !ok || int(got) != 36 {
		t.Fatalf("num_predict = %#v, want 36", options["num_predict"])
	}
}

func TestVoicePipelineAdaptersFromEnvTextMaxTokenOptionOverridesEnv(t *testing.T) {
	var body map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body = readJSONRequestBody(t, req)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: [DONE]\n")),
			Request:    req,
		}, nil
	})}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_PROVIDER_PRIMARY=deepseek",
		"A21_TEXT_STREAM_PROFILE=deepseek",
		"A21_LAB_DEEPSEEK_API_KEY=configured-token",
		"A21_VOICE_TEXT_MAX_TOKENS=96",
	}, VoicePipelineAdapterOptions{
		TextHTTPClient: client,
		TextMaxTokens:  18,
	})

	events, err := adapters.TextStream.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = collectTextEvents(t, events)

	got, ok := body["max_tokens"].(float64)
	if !ok || int(got) != 18 {
		t.Fatalf("max_tokens = %#v, want 18", body["max_tokens"])
	}
}

func collectASREvents(t *testing.T, events <-chan ASRAdapterEvent) []ASRAdapterEvent {
	t.Helper()
	var collected []ASRAdapterEvent
	for event := range events {
		collected = append(collected, event)
	}
	return collected
}

func collectTextEvents(t *testing.T, events <-chan TextStreamEvent) []TextStreamEvent {
	t.Helper()
	var collected []TextStreamEvent
	for event := range events {
		collected = append(collected, event)
	}
	return collected
}

func collectVoiceChunks(t *testing.T, chunks <-chan VoiceAudioChunk) []VoiceAudioChunk {
	t.Helper()
	var collected []VoiceAudioChunk
	for chunk := range chunks {
		collected = append(collected, chunk)
	}
	return collected
}

func readJSONRequestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
