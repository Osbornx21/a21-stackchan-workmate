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
			if err := audio.WritePCM16MonoWAV(path, 24000, make([]byte, 3000)); err != nil {
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
