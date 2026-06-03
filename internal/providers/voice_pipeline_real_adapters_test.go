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
	"time"

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

func TestDoubaoRealtimeTTSTTSAdapterStreamsProviderDeltasAsDownlinkChunks(t *testing.T) {
	delta := base64.StdEncoding.EncodeToString(make([]byte, 2880))
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{{
			"type":    "response.audio.delta",
			"item_id": "a21-doubao-tts-stream-001",
			"delta":   delta,
		}},
	}
	provider := NewDoubaoRealtimeTTSProvider(DoubaoRealtimeTTSProviderConfig{
		APIKey:                "sk-a21-secret",
		Model:                 "doubao-tts",
		Voice:                 "zh_female_kailangjiejie_moon_bigtts",
		OutputAudioSampleRate: 24000,
	}, fakeRealtimeDialer{conn: conn})
	adapter := NewDoubaoRealtimeTTSTTSAdapter(DoubaoRealtimeTTSTTSAdapterOptions{
		Name:     "doubao_tts_realtime",
		Provider: provider,
	})
	if _, ok := adapter.(StreamingTTSAdapter); !ok {
		t.Fatalf("adapter %T does not implement StreamingTTSAdapter", adapter)
	}

	chunks, err := adapter.Synthesize(context.Background(), TTSAdapterRequest{
		Session: VoiceSession{TraceID: "a21-trace-doubao-tts-stream", SessionID: "a21-session-doubao-tts-stream", DeviceID: "stackchan-001"},
		Mode:    "workmate",
		Text:    "用户文本不能进报告",
	})
	if err != nil {
		t.Fatal(err)
	}
	first := receiveVoiceAudioChunk(t, chunks)
	if first.Codec != "pcm_s16le" || first.SampleRateHz != 24000 || first.Channels != 1 || first.DurationMS != 60 {
		t.Fatalf("first chunk = %+v, want 24k mono 60ms pcm_s16le", first)
	}
	pcm, err := base64.StdEncoding.DecodeString(first.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 2880 {
		t.Fatalf("first chunk bytes = %d, want 2880", len(pcm))
	}
	_ = collectVoiceChunks(t, chunks)
	if !conn.closed {
		t.Fatal("realtime TTS connection was not closed")
	}
	got := realtimeEventTypes(conn.messages)
	want := []string{"tts_session.update", "input_text.append", "input_text.done"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("event types = %#v, want %#v", got, want)
	}
	for _, message := range conn.messages {
		rendered := mustProviderJSON(t, message)
		for _, forbidden := range []string{"sk-a21-secret", "Authorization", "Bearer"} {
			if strings.Contains(rendered, forbidden) {
				t.Fatalf("provider event leaked %q: %s", forbidden, rendered)
			}
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
	if _, ok := host.ASR.(StreamingASRAdapter); ok {
		t.Fatalf("batch sherpa_onnx adapter unexpectedly implements StreamingASRAdapter")
	}
}

func TestLocalSherpaONNXStreamingASRAdapterStreamsInjectedSession(t *testing.T) {
	factoryCalls := 0
	adapter := NewLocalSherpaONNXStreamingASRAdapter(LocalSherpaONNXStreamingASRAdapterOptions{
		LocalSherpaONNXASRAdapterOptions: LocalSherpaONNXASRAdapterOptions{
			Name: "sherpa_onnx_streaming",
		},
		StreamingSessionFactory: func(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error) {
			factoryCalls++
			if req.Session.TraceID != "a21-trace-sherpa-streaming" || req.Mode != "workmate" {
				t.Fatalf("streaming start request = %+v", req)
			}
			mock := NewMockStreamingASRAdapter("a21-injected-sherpa-session")
			return mock.StartStreamingASR(ctx, req)
		},
	})
	streaming, ok := adapter.(StreamingASRAdapter)
	if !ok {
		t.Fatalf("adapter %T does not implement StreamingASRAdapter", adapter)
	}

	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{
		Session: VoiceSession{TraceID: "a21-trace-sherpa-streaming", SessionID: "a21-session-sherpa-streaming", DeviceID: "stackchan-001"},
		Mode:    "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 {
		t.Fatalf("factory calls = %d, want 1", factoryCalls)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Seq:          1,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    2,
		RMS:          0.2,
		PCM16LE:      []byte{1, 0},
	}); err != nil {
		t.Fatal(err)
	}
	partial := receiveASREvent(t, session.Events())
	if partial.Final || strings.TrimSpace(partial.Text) == "" {
		t.Fatalf("partial = %+v, want non-final text", partial)
	}
	if err := session.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	final := receiveASREvent(t, session.Events())
	if !final.Final || strings.TrimSpace(final.Text) == "" {
		t.Fatalf("final = %+v, want final text", final)
	}
}

func TestLocalSherpaONNXStreamingASRAdapterRequiresConfiguredSession(t *testing.T) {
	adapter := NewLocalSherpaONNXStreamingASRAdapter(LocalSherpaONNXStreamingASRAdapterOptions{})
	streaming, ok := adapter.(StreamingASRAdapter)
	if !ok {
		t.Fatalf("adapter %T does not implement StreamingASRAdapter", adapter)
	}
	_, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{})
	if err == nil {
		t.Fatal("StartStreamingASR succeeded without a configured session factory")
	}
	rendered := err.Error()
	if !strings.Contains(rendered, "sherpa-onnx streaming ASR helper is not configured") {
		t.Fatalf("error = %q, want stable not-configured error", rendered)
	}
	for _, forbidden := range []string{"/Users/", "raw_audio", "transcript", "fixture transcript"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("error leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "helper-commands.jsonl")
	helperPath := writeFakeSherpaStreamingHelper(t, logPath)
	adapter := NewLocalSherpaONNXStreamingASRAdapter(LocalSherpaONNXStreamingASRAdapterOptions{
		LocalSherpaONNXASRAdapterOptions: LocalSherpaONNXASRAdapterOptions{
			Name:     "sherpa_onnx_streaming",
			ModelDir: filepath.Join(t.TempDir(), "a21-streaming-model"),
			Family:   "streaming_zipformer",
			Runner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
				t.Fatal("streaming ASR path must not call batch WAV runner")
				return audio.LocalASRResult{}, nil
			},
		},
		StreamingHelperPath: helperPath,
	})
	streaming, ok := adapter.(StreamingASRAdapter)
	if !ok {
		t.Fatalf("adapter %T does not implement StreamingASRAdapter", adapter)
	}
	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{
		Session: VoiceSession{TraceID: "a21-trace-subprocess", SessionID: "a21-session-subprocess", DeviceID: "stackchan-001"},
		Mode:    "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Seq:          9,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    4,
		RMS:          0.2,
		PCM16LE:      []byte{1, 0, 2, 0},
	}); err != nil {
		t.Fatal(err)
	}
	partial := receiveASREvent(t, session.Events())
	if partial.Final || partial.Text != "partial-from-helper" {
		t.Fatalf("partial = %+v, want helper partial", partial)
	}
	if err := session.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	final := receiveASREvent(t, session.Events())
	if !final.Final || final.Text != "final-from-helper" {
		t.Fatalf("final = %+v, want helper final", final)
	}
	rendered := eventuallyReadFile(t, logPath)
	startIndex := strings.Index(rendered, `"type":"start"`)
	appendIndex := strings.Index(rendered, `"type":"append"`)
	commitIndex := strings.Index(rendered, `"type":"commit"`)
	if startIndex < 0 || appendIndex < 0 || commitIndex < 0 || !(startIndex < appendIndex && appendIndex < commitIndex) {
		t.Fatalf("helper commands out of order:\n%s", rendered)
	}
	for _, forbidden := range []string{".wav", "a21-trace-subprocess", "a21-session-subprocess"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("helper command log leaked or used forbidden value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsSherpaONNXStreamingASR(t *testing.T) {
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming",
	}, VoicePipelineAdapterOptions{
		StreamingASRSessionFactory: func(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error) {
			mock := NewMockStreamingASRAdapter("a21-injected-sherpa-session")
			return mock.StartStreamingASR(ctx, req)
		},
	})

	if adapters.ExecutionMode != "host_local" || adapters.ASR.Name() != "sherpa_onnx_streaming" {
		t.Fatalf("adapters execution/ASR = %q/%q, want host_local/sherpa_onnx_streaming", adapters.ExecutionMode, adapters.ASR.Name())
	}
	streaming, ok := adapters.ASR.(StreamingASRAdapter)
	if !ok {
		t.Fatalf("selected ASR %T does not implement StreamingASRAdapter", adapters.ASR)
	}
	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{
		Session: VoiceSession{TraceID: "a21-trace-selected-streaming", SessionID: "a21-session-selected-streaming", DeviceID: "stackchan-001"},
		Mode:    "workmate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    2,
		RMS:          0.2,
		PCM16LE:      []byte{1, 0},
	}); err != nil {
		t.Fatal(err)
	}
	if event := receiveASREvent(t, session.Events()); event.Final || strings.TrimSpace(event.Text) == "" {
		t.Fatalf("partial = %+v, want non-final text", event)
	}
}

func TestVoicePipelineAdaptersFromEnvWiresSherpaStreamingSubprocessHelper(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "helper-commands.jsonl")
	helperPath := writeFakeSherpaStreamingHelper(t, logPath)
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming",
		"A21_SHERPA_ONNX_STREAMING_HELPER=" + helperPath,
		"A21_SHERPA_ONNX_ASR_MODEL_DIR=" + filepath.Join(t.TempDir(), "a21-streaming-model"),
		"A21_SHERPA_ONNX_ASR_FAMILY=streaming_zipformer",
	}, VoicePipelineAdapterOptions{
		ASRRunner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
			t.Fatal("streaming helper env must not call batch WAV runner")
			return audio.LocalASRResult{}, nil
		},
	})
	streaming, ok := adapters.ASR.(StreamingASRAdapter)
	if adapters.ExecutionMode != "host_local" || adapters.ASR.Name() != "sherpa_onnx_streaming" || !ok {
		t.Fatalf("ASR adapter = %T/%s mode=%s, want streaming sherpa host_local", adapters.ASR, adapters.ASR.Name(), adapters.ExecutionMode)
	}
	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{Mode: "workmate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Seq:          1,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    2,
		PCM16LE:      []byte{1, 0},
	}); err != nil {
		t.Fatal(err)
	}
	if event := receiveASREvent(t, session.Events()); event.Final || event.Text != "partial-from-helper" {
		t.Fatalf("partial = %+v, want helper partial", event)
	}
}

func TestVoicePipelineAdaptersFromEnvDiscoversCanonicalSherpaStreamingCache(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	logPath := filepath.Join(root, "helper-commands.jsonl")
	helperPath := filepath.Join(root, "scripts", "a21_sherpa_onnx_streaming_asr_session.py")
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		t.Fatal(err)
	}
	helper := `#!/bin/sh
set -eu
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + logPath + `"
  case "$line" in
    *'"type":"start"'*) printf '%s\n' '{"type":"ready"}' ;;
    *'"type":"append"'*) printf '%s\n' '{"type":"partial","text":"partial-from-canonical-helper"}' ;;
    *'"type":"commit"'*) printf '%s\n' '{"type":"final","text":"final-from-canonical-helper"}'; exit 0 ;;
  esac
done
`
	if err := os.WriteFile(helperPath, []byte(helper), 0o755); err != nil {
		t.Fatal(err)
	}
	pythonPath := filepath.Join(root, ".a21-tools", "sherpa-onnx-venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(pythonPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pythonPath, []byte("#!/bin/sh\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	modelDir := filepath.Join(root, ".a21-tools", "sherpa-onnx-asr-models", "sherpa-onnx-streaming-zipformer-zh-int8-2025-06-30")
	createProviderStreamingASRModelFiles(t, modelDir)
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming",
	}, VoicePipelineAdapterOptions{
		ASRRunner: func(ctx context.Context, options audio.LocalASROptions) (audio.LocalASRResult, error) {
			t.Fatal("canonical streaming ASR runtime must not call batch WAV runner")
			return audio.LocalASRResult{}, nil
		},
	})
	streaming, ok := adapters.ASR.(StreamingASRAdapter)
	if adapters.ExecutionMode != "host_local" || adapters.ASR.Name() != "sherpa_onnx_streaming" || !ok {
		t.Fatalf("ASR adapter = %T/%s mode=%s, want streaming sherpa host_local", adapters.ASR, adapters.ASR.Name(), adapters.ExecutionMode)
	}
	session, err := streaming.StartStreamingASR(context.Background(), StreamingASRStartRequest{Mode: "workmate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.AppendFrame(context.Background(), VoicePipelinePCMFrame{
		Seq:          1,
		Codec:        "pcm_s16le",
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   60,
		ByteCount:    2,
		PCM16LE:      []byte{1, 0},
	}); err != nil {
		t.Fatal(err)
	}
	if event := receiveASREvent(t, session.Events()); event.Final || event.Text != "partial-from-canonical-helper" {
		t.Fatalf("partial = %+v, want canonical helper partial", event)
	}
	if err := session.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if event := receiveASREvent(t, session.Events()); !event.Final || event.Text != "final-from-canonical-helper" {
		t.Fatalf("final = %+v, want canonical helper final", event)
	}
	rendered := eventuallyReadFile(t, logPath)
	for _, forbidden := range []string{root, ".wav"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("helper command log leaked or used forbidden value %q:\n%s", forbidden, rendered)
		}
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

func createProviderStreamingASRModelFiles(t *testing.T, modelDir string) {
	t.Helper()
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"encoder.int8.onnx", "decoder.onnx", "joiner.int8.onnx", "tokens.txt"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("a21"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsDoubaoRealtimeTTS(t *testing.T) {
	conn := &fakeRealtimeConn{
		serverMessages: []map[string]any{{
			"type":  "response.audio.delta",
			"delta": base64.StdEncoding.EncodeToString(make([]byte, 2880)),
		}},
	}
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_TTS_FAST_PROFILE=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
		"A21_DOUBAO_TTS_SAMPLE_RATE_HZ=24000",
	}, VoicePipelineAdapterOptions{
		TTSRealtimeDialer: fakeRealtimeDialer{conn: conn},
	})
	if adapters.ExecutionMode != "host_local" || adapters.TTS.Name() != "doubao_tts_realtime" {
		t.Fatalf("adapters execution/TTS = %q/%q", adapters.ExecutionMode, adapters.TTS.Name())
	}
	if _, ok := adapters.TTS.(StreamingTTSAdapter); !ok {
		t.Fatalf("selected TTS %T does not implement StreamingTTSAdapter", adapters.TTS)
	}
	chunks, err := adapters.TTS.Synthesize(context.Background(), TTSAdapterRequest{Text: "文本不进报告"})
	if err != nil {
		t.Fatal(err)
	}
	first := receiveVoiceAudioChunk(t, chunks)
	if first.SampleRateHz != 24000 || first.DurationMS != 60 {
		t.Fatalf("first chunk = %+v, want 24k 60ms", first)
	}
}

func TestVoicePipelineAdaptersFromEnvSelectsIflytekTTS(t *testing.T) {
	var captured audio.LocalTTSOptions
	adapters := VoicePipelineAdaptersFromEnv([]string{
		"A21_TTS_FAST_PROFILE=iflytek_tts",
	}, VoicePipelineAdapterOptions{
		TTSOptions: audio.LocalTTSOptions{OutputDir: t.TempDir()},
		TTSSynthesizer: func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
			captured = options
			if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(options.OutputDir, "a21-iflytek-adapter.wav")
			if err := audio.WritePCM16MonoWAV(path, 24000, make([]byte, 2880)); err != nil {
				t.Fatal(err)
			}
			return audio.LocalTTSReport{Status: "passed", OutputPath: path}, nil
		},
	})

	if adapters.ExecutionMode != "host_local" || adapters.TTS.Name() != "iflytek_tts" {
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
	if captured.Text != "用户原文不进报告" || captured.OutputSampleRateHz != 24000 {
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

func TestVoicePipelineAdaptersFromEnvSelectsCompatibilityTextStreamProfile(t *testing.T) {
	var body map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "a21-stepfun.invalid" || req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("request URL = %s, want a21-stepfun.invalid/v1/chat/completions", req.URL.String())
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
		"A21_PROVIDER_PRIMARY=stepfun",
		"A21_TEXT_STREAM_PROFILE=stepfun",
		"A21_LAB_STEPFUN_API_KEY=configured-token",
		"A21_STEPFUN_MODEL=step-1-8k",
		"A21_STEPFUN_BASE_URL=https://a21-stepfun.invalid/v1",
	}, VoicePipelineAdapterOptions{
		TextHTTPClient: client,
		TextMaxTokens:  9,
	})

	if adapters.ExecutionMode != "host_local" {
		t.Fatalf("execution mode = %q, want host_local", adapters.ExecutionMode)
	}
	if adapters.TextStream.Name() != "stepfun" {
		t.Fatalf("text stream adapter = %s, want stepfun", adapters.TextStream.Name())
	}
	events, err := adapters.TextStream.StreamText(context.Background(), TextStreamAdapterRequest{Text: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = collectTextEvents(t, events)

	if got, ok := body["max_tokens"].(float64); !ok || int(got) != 9 {
		t.Fatalf("max_tokens = %#v, want 9", body["max_tokens"])
	}
	if got := body["model"]; got != "step-1-8k" {
		t.Fatalf("model = %#v, want step-1-8k", got)
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

func writeFakeSherpaStreamingHelper(t *testing.T, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a21-fake-sherpa-streaming-helper.sh")
	script := `#!/bin/sh
set -eu
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + logPath + `"
  case "$line" in
    *'"type":"start"'*) printf '%s\n' '{"type":"ready"}' ;;
    *'"type":"append"'*) printf '%s\n' '{"type":"partial","text":"partial-from-helper"}' ;;
    *'"type":"commit"'*) printf '%s\n' '{"type":"final","text":"final-from-helper"}'; exit 0 ;;
    *'"type":"cancel"'*) exit 0 ;;
  esac
done
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func eventuallyReadFile(t *testing.T, path string) string {
	t.Helper()
	var lastErr error
	for i := 0; i < 20; i++ {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("read %s: %v", path, lastErr)
	return ""
}

func receiveASREvent(t *testing.T, events <-chan ASRAdapterEvent) ASRAdapterEvent {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("ASR event channel closed before event")
		}
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ASR event")
	}
	return ASRAdapterEvent{}
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

func receiveVoiceAudioChunk(t *testing.T, chunks <-chan VoiceAudioChunk) VoiceAudioChunk {
	t.Helper()
	select {
	case chunk, ok := <-chunks:
		if !ok {
			t.Fatal("voice audio chunk channel closed before chunk")
		}
		return chunk
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for voice audio chunk")
	}
	return VoiceAudioChunk{}
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
