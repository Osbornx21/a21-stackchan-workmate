package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	path := filepath.Join(t.TempDir(), "a21-fake-sherpa-streaming-helper.py")
	quotedLogPath, err := json.Marshal(logPath)
	if err != nil {
		t.Fatal(err)
	}
	script := `#!/usr/bin/env python3
import json
import sys

log_path = ` + string(quotedLogPath) + `

with open(log_path, "a", encoding="utf-8") as log:
    for line in sys.stdin:
        log.write(line)
        log.flush()
        try:
            command = json.loads(line)
        except Exception:
            print(json.dumps({"type": "error", "code": "bad_command"}), flush=True)
            continue
        command_type = command.get("type")
        if command_type == "start":
            print(json.dumps({"type": "ready"}), flush=True)
        elif command_type == "append":
            print(json.dumps({"type": "partial", "text": "partial-from-helper"}), flush=True)
        elif command_type == "commit":
            print(json.dumps({"type": "final", "text": "final-from-helper"}), flush=True)
            sys.exit(0)
        elif command_type == "cancel":
            sys.exit(0)
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
