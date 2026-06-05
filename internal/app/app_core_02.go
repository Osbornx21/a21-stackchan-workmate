package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/gateway"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/runtimeguard"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func measureGatewayAudioDownlink(serverURL string, seq uint64) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	started := time.Now()
	conn, _, err := websocket.Dial(ctx, latencyBenchWebSocketURL(serverURL, "/ws/audio"), nil)
	if err != nil {
		return 0, err
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 latency bench done")

	traceID := fmt.Sprintf("a21-trace-audio-bench-%06d", seq)
	sessionID := fmt.Sprintf("a21-session-audio-bench-%06d", seq)
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq, traceID, sessionID, latencyBenchPCM16Base64(0)); err != nil {
		return 0, err
	}

	for i := 0; i < 2; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return 0, err
		}
		if event.Kind != protocol.KindControlEvent {
			return 0, fmt.Errorf("audio bench event %d kind %q, want %q", i, event.Kind, protocol.KindControlEvent)
		}
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		return 0, err
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		return 0, fmt.Errorf("audio bench playback kind %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}
	if playback.TraceID != traceID || playback.SessionID != sessionID {
		return 0, fmt.Errorf("audio bench playback trace/session mismatch")
	}
	return time.Since(started), nil
}

func measureGatewayAudioBargeIn(serverURL string, seq uint64) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, latencyBenchWebSocketURL(serverURL, "/ws/audio"), nil)
	if err != nil {
		return 0, err
	}
	defer conn.Close(websocket.StatusNormalClosure, "a21 latency bench done")

	traceID := fmt.Sprintf("a21-trace-audio-barge-bench-%06d", seq)
	sessionID := fmt.Sprintf("a21-session-audio-barge-bench-%06d", seq)
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq*2, traceID, sessionID, latencyBenchPCM16Base64(0)); err != nil {
		return 0, err
	}
	for i := 0; i < 2; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			return 0, err
		}
		if event.Kind != protocol.KindControlEvent {
			return 0, fmt.Errorf("audio barge-in setup event %d kind %q, want %q", i, event.Kind, protocol.KindControlEvent)
		}
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		return 0, err
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		return 0, fmt.Errorf("audio barge-in setup playback kind %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}

	started := time.Now()
	if err := writeLatencyBenchAudioFrame(ctx, conn, seq*2+1, traceID, sessionID, latencyBenchPCM16Base64(12000)); err != nil {
		return 0, err
	}
	var interrupted protocol.Envelope
	if err := wsjson.Read(ctx, conn, &interrupted); err != nil {
		return 0, err
	}
	duration := time.Since(started)
	if interrupted.Kind != protocol.KindControlEvent {
		return 0, fmt.Errorf("audio barge-in event kind %q, want %q", interrupted.Kind, protocol.KindControlEvent)
	}
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(interrupted.Payload, &payload); err != nil {
		return 0, err
	}
	if payload.State != protocol.ExpressionInterrupted {
		return 0, fmt.Errorf("audio barge-in state %q, want %q", payload.State, protocol.ExpressionInterrupted)
	}
	if payload.StreamID == "" {
		return 0, fmt.Errorf("audio barge-in missing stream_id")
	}
	return duration, nil
}

func writeLatencyBenchAudioFrame(ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, dataBase64 string) error {
	payload, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   dataBase64,
	})
	if err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-bench-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   payload,
	})
}

func latencyBenchPCM16Base64(sample int16) string {
	sampleCount := 16000 * 20 / 1000
	data := make([]byte, sampleCount*2)
	for i := 0; i < sampleCount; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func latencyBenchWebSocketURL(serverURL string, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}

func measureGatewayRequest(handler http.Handler, method string, path string, body string) (time.Duration, error) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	started := time.Now()
	handler.ServeHTTP(rec, req)
	duration := time.Since(started)
	if rec.Code != http.StatusOK {
		return 0, fmt.Errorf("%s %s returned status %d: %s", method, path, rec.Code, rec.Body.String())
	}
	return duration, nil
}

func latencySeries(samples []time.Duration) latencyBenchSeries {
	return latencyBenchSeries{
		Samples: len(samples),
		P50MS:   percentileMS(samples, 0.50),
		P95MS:   percentileMS(samples, 0.95),
	}
}

func percentileMS(samples []time.Duration, quantile float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	rank := int(math.Ceil(quantile*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return float64(sorted[rank].Microseconds()) / 1000
}

func elapsedReportMS(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}

func jitterMS(samples []time.Duration) float64 {
	if len(samples) == 0 {
		return 0
	}
	minimum := samples[0]
	maximum := samples[0]
	for _, sample := range samples[1:] {
		if sample < minimum {
			minimum = sample
		}
		if sample > maximum {
			maximum = sample
		}
	}
	return float64((maximum - minimum).Microseconds()) / 1000
}

func writeJSONReport(writer io.Writer, report runtimeguard.PreflightReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONLANProbe(writer io.Writer, report lanProbeReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func runGateway(args []string, stdout io.Writer, stderr io.Writer) int {
	options := gatewayCLIOptions{Addr: "127.0.0.1:21080"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 gateway --addr 127.0.0.1:21080 [--product-chain host_local|cloud_edge] [--local-ollama-base-url http://127.0.0.1:11434] [--local-ollama-model qwen2.5:0.5b] [--voice-text-max-tokens 32] [--mac-local-gateway-url ws://127.0.0.1:21081/v1/xiaozhi] [--public-gateway-url https://a21.example.com] [--warm-product-chain]")
			return 0
		case "--addr":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--addr requires a value")
				return 2
			}
			i++
			options.Addr = args[i]
		case "--product-chain", "--xiaozhi-product-chain":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintf(stderr, "%s requires a value\n", args[i])
				return 2
			}
			i++
			options.ProductChain = args[i]
		case "--local-ollama-base-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--local-ollama-base-url requires a value")
				return 2
			}
			i++
			options.LocalOllamaBaseURL = args[i]
		case "--local-ollama-model":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--local-ollama-model requires a value")
				return 2
			}
			i++
			options.LocalOllamaModel = args[i]
		case "--voice-text-max-tokens":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--voice-text-max-tokens requires a value")
				return 2
			}
			i++
			options.VoiceTextMaxTokens = args[i]
		case "--public-gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--public-gateway-url requires a value")
				return 2
			}
			i++
			options.PublicGatewayURL = args[i]
		case "--mac-local-gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--mac-local-gateway-url requires a value")
				return 2
			}
			i++
			options.MacLocalGatewayURL = args[i]
		case "--warm-product-chain":
			options.WarmProductChain = true
		default:
			fmt.Fprintf(stderr, "unknown gateway option %q\n", args[i])
			return 2
		}
	}

	env := applyXiaozhiProductChainEnvDefaults(gatewayEnvWithCLIOptions(os.Environ(), options))
	if options.WarmProductChain {
		if err := warmGatewayProductChain(context.Background(), env); err != nil {
			fmt.Fprintf(stderr, "gateway product chain warmup failed: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "a21 gateway listening on %s\n", options.Addr)
	if err := http.ListenAndServe(options.Addr, newGatewayServerFromEnv(env).Handler()); err != nil {
		fmt.Fprintf(stderr, "gateway: %v\n", err)
		return 1
	}
	return 0
}

type gatewayCLIOptions struct {
	Addr               string
	ProductChain       string
	LocalOllamaBaseURL string
	LocalOllamaModel   string
	VoiceTextMaxTokens string
	MacLocalGatewayURL string
	PublicGatewayURL   string
	WarmProductChain   bool
}

func gatewayEnvWithCLIOptions(env []string, options gatewayCLIOptions) []string {
	out := append([]string(nil), env...)
	out = gatewayEnvSetIfNotEmpty(out, "A21_XIAOZHI_PRODUCT_CHAIN", options.ProductChain)
	out = gatewayEnvSetIfNotEmpty(out, "A21_LOCAL_OLLAMA_BASE_URL", options.LocalOllamaBaseURL)
	out = gatewayEnvSetIfNotEmpty(out, "A21_LOCAL_OLLAMA_MODEL", options.LocalOllamaModel)
	out = gatewayEnvSetIfNotEmpty(out, "A21_VOICE_TEXT_MAX_TOKENS", options.VoiceTextMaxTokens)
	out = gatewayEnvSetIfNotEmpty(out, "A21_MAC_LOCAL_GATEWAY_URL", options.MacLocalGatewayURL)
	out = gatewayEnvSetIfNotEmpty(out, "A21_PUBLIC_GATEWAY_URL", options.PublicGatewayURL)
	return out
}

func gatewayEnvSetIfNotEmpty(env []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return env
	}
	prefix := key + "="
	out := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+value)
}

func newGatewayServerFromEnv(env []string) *gateway.Server {
	return gateway.NewServerWithOptions(newGatewayServerOptionsFromEnv(env))
}

func newGatewayServerOptionsFromEnv(env []string) gateway.ServerOptions {
	env = applyXiaozhiProductChainEnvDefaults(env)
	xiaozhiVoicePipelineAdapters := providers.VoicePipelineAdaptersFromEnv(env)
	options := gateway.ServerOptions{
		VoiceProvider:                providers.NewGatewayVoiceProviderFromEnv(env),
		XiaozhiVoicePipelineAdapters: &xiaozhiVoicePipelineAdapters,
		AudioIngressConfig:           newAudioIngressConfigFromEnv(env),
		XiaozhiStockProfessional:     appEnvBool(env, "A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE"),
		XiaozhiProductPlaybackEvents: appEnvBool(env, "A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS"),
		XiaozhiProductTouchEvents:    appEnvBool(env, "A21_XIAOZHI_PRODUCT_TOUCH_EVENTS"),
		XiaozhiProductTouchReactions: appEnvBool(env, "A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS"),
		XiaozhiProductStateReactions: appEnvBool(env, "A21_XIAOZHI_PRODUCT_STATE_REACTIONS"),
		XiaozhiSTTScreenPolicy:       appEnvValue(env, "A21_XIAOZHI_STT_SCREEN_POLICY"),
		BodySceneStepDelay:           180 * time.Millisecond,
		MacLocalGatewayURL:           appEnvValue(env, "A21_MAC_LOCAL_GATEWAY_URL"),
		PublicGatewayURL:             appEnvValue(env, "A21_PUBLIC_GATEWAY_URL"),
		CloudVoiceProfile:            appEnvValue(env, "A21_CLOUD_VOICE_PROFILE"),
		CloudVoiceEnv:                append([]string(nil), env...),
		WorkspaceDocumentStoreDir:    appEnvValue(env, "A21_WORKSPACE_DOCUMENT_STORE_DIR"),
	}
	if listenMaxMS, err := strconv.Atoi(strings.TrimSpace(appEnvValue(env, "A21_XIAOZHI_LISTEN_MAX_MS"))); err == nil && listenMaxMS > 0 {
		options.XiaozhiListenMaxDuration = time.Duration(listenMaxMS) * time.Millisecond
	}
	if bodySceneStepDelayMS, err := strconv.Atoi(strings.TrimSpace(appEnvValue(env, "A21_BODY_SCENE_STEP_DELAY_MS"))); err == nil && bodySceneStepDelayMS > 0 {
		options.BodySceneStepDelay = time.Duration(bodySceneStepDelayMS) * time.Millisecond
	}
	if uploadMaxBytes, err := strconv.ParseInt(strings.TrimSpace(appEnvValue(env, "A21_WORKSPACE_DOCUMENT_MAX_BYTES")), 10, 64); err == nil && uploadMaxBytes > 0 {
		options.WorkspaceDocumentMaxBytes = uploadMaxBytes
	}
	if adapterURL := strings.TrimSpace(appEnvValue(env, "A21_V21_ADAPTER_URL")); adapterURL != "" {
		client, err := v21adapter.NewHTTPClient(adapterURL)
		if err != nil {
			options.V21Client = v21ConfigurationErrorClient{err: err}
		} else {
			options.V21Client = client
		}
	}
	return options
}

func applyXiaozhiProductChainEnvDefaults(env []string) []string {
	switch gatewayProductChainMode(env) {
	case "cloud_edge", "cloud-edge", "public_cloud", "public-cloud":
		return applyXiaozhiCloudEdgeProductChainEnvDefaults(env)
	case "host_local", "host-local", "product", "real":
	default:
		return applyXiaozhiFastAckDelayEnvDefaults(env)
	}
	out := append([]string(nil), env...)
	out = applyXiaozhiFastAckDelayEnvDefaults(out)
	if strings.TrimSpace(appEnvValue(out, "A21_ASR_LOCAL_PROFILE")) == "" {
		out = append(out, "A21_ASR_LOCAL_PROFILE=sherpa_onnx")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TTS_FAST_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_BALANCED_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_QUALITY_PROFILE")) == "" {
		if voiceCloneCommandFromEnv(out) != "" &&
			strings.TrimSpace(appEnvValue(out, "A21_VOICE_CLONE_REF_AUDIO")) != "" {
			out = append(out, "A21_TTS_FAST_PROFILE=voice_clone_cli")
		} else {
			out = append(out, "A21_TTS_FAST_PROFILE=sherpa_onnx_tts")
		}
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TEXT_STREAM_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_PROVIDER_PRIMARY")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LOCAL_OLLAMA_BASE_URL")) != "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LOCAL_OLLAMA_MODEL")) != "" {
		out = append(out, "A21_TEXT_STREAM_PROFILE=local_ollama")
	}
	return out
}

func applyXiaozhiCloudEdgeProductChainEnvDefaults(env []string) []string {
	out := append([]string(nil), env...)
	if strings.TrimSpace(appEnvValue(out, "A21_XIAOZHI_FAST_ACK_ENABLED")) == "" {
		out = append(out, "A21_XIAOZHI_FAST_ACK_ENABLED=true")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_XIAOZHI_FAST_ACK_DELAY_MS")) == "" {
		out = append(out, "A21_XIAOZHI_FAST_ACK_DELAY_MS=700")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_XIAOZHI_STT_SCREEN_POLICY")) == "" {
		out = append(out, "A21_XIAOZHI_STT_SCREEN_POLICY=status_only")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_ASR_PROFILE")) == "" {
		out = append(out, "A21_ASR_PROFILE=cloud")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_ASR_CLOUD_PROFILE")) == "" {
		if strings.TrimSpace(appEnvValue(out, "A21_DASHSCOPE_API_KEY")) != "" {
			out = append(out, "A21_ASR_CLOUD_PROFILE=dashscope_qwen_asr_realtime")
		} else {
			out = append(out, "A21_ASR_CLOUD_PROFILE=doubao_asr_realtime")
		}
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TEXT_STREAM_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_PROVIDER_PRIMARY")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LAB_STEPFUN_API_KEY")) != "" {
		out = append(out, "A21_TEXT_STREAM_PROFILE=stepfun")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TEXT_STREAM_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_PROVIDER_PRIMARY")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_LAB_DEEPSEEK_API_KEY")) != "" {
		out = append(out, "A21_TEXT_STREAM_PROFILE=deepseek")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_TTS_FAST_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_BALANCED_PROFILE")) == "" &&
		strings.TrimSpace(appEnvValue(out, "A21_TTS_QUALITY_PROFILE")) == "" {
		if strings.TrimSpace(appEnvValue(out, "A21_DASHSCOPE_API_KEY")) != "" {
			out = append(out, "A21_TTS_FAST_PROFILE=dashscope_qwen_tts_realtime")
		} else {
			out = append(out, "A21_TTS_FAST_PROFILE=doubao_tts_realtime")
		}
	}
	return out
}

func applyXiaozhiFastAckDelayEnvDefaults(env []string) []string {
	out := append([]string(nil), env...)
	if strings.TrimSpace(appEnvValue(out, "A21_XIAOZHI_FAST_ACK_ENABLED")) == "" {
		out = append(out, "A21_XIAOZHI_FAST_ACK_ENABLED=true")
	}
	if strings.TrimSpace(appEnvValue(out, "A21_XIAOZHI_FAST_ACK_DELAY_MS")) == "" {
		out = append(out, "A21_XIAOZHI_FAST_ACK_DELAY_MS=700")
	}
	return out
}

func gatewayProductChainMode(env []string) string {
	return strings.ToLower(strings.TrimSpace(firstNonEmpty(appEnvValue(env, "A21_XIAOZHI_PRODUCT_CHAIN"), appEnvValue(env, "A21_PRODUCT_CHAIN"))))
}

func warmGatewayProductChain(ctx context.Context, env []string) error {
	switch gatewayProductChainMode(env) {
	case "cloud_edge", "cloud-edge", "public_cloud", "public-cloud":
	case "host_local", "host-local", "product", "real":
	default:
		return nil
	}
	warmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if strings.Contains(strings.ToLower(appEnvValue(env, "A21_ASR_LOCAL_PROFILE")), "sherpa") {
		result, err := audio.RunSherpaONNXASR(warmCtx, audio.LocalASROptions{})
		if err != nil {
			return fmt.Errorf("ASR warmup failed")
		}
		if result.Report.Status != "passed" {
			return fmt.Errorf("ASR warmup did not pass")
		}
	}
	textProfile := firstNonEmpty(appEnvValue(env, "A21_TEXT_STREAM_PROFILE"), appEnvValue(env, "A21_PROVIDER_PRIMARY"))
	if xiaozhiVoiceBenchNonMockStageProfile(textProfile) {
		if _, err := providers.RunTextStreamCompletionFromEnv(warmCtx, env, providers.TextStreamCompletionOptions{
			ProviderName: textProfile,
			Prompt:       "A21 warmup. Reply briefly.",
			MaxTokens:    8,
		}); err != nil {
			return fmt.Errorf("text stream warmup failed")
		}
	}
	ttsProfile := firstNonEmpty(appEnvValue(env, "A21_TTS_FAST_PROFILE"), appEnvValue(env, "A21_TTS_BALANCED_PROFILE"), appEnvValue(env, "A21_TTS_QUALITY_PROFILE"))
	normalizedTTSProfile := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(ttsProfile)), "-", "_")
	if strings.Contains(normalizedTTSProfile, "sherpa") {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		report, err := audio.SynthesizeSherpaONNX(warmCtx, audio.LocalTTSOptions{
			Text:      "A21 warmup",
			OutputDir: outputDir,
			SpeakerID: 21,
		})
		if err != nil {
			return fmt.Errorf("TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("TTS warmup did not pass")
		}
	} else if normalizedTTSProfile == "voice_clone_cli" {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		clone := voiceCloneRuntimeOptionsFromEnv(env)
		report, err := audio.SynthesizeVoiceCloneCLI(warmCtx, audio.LocalTTSOptions{
			Text:                         "A21 warmup",
			OutputDir:                    outputDir,
			VoiceCloneCommand:            clone.Command,
			VoiceCloneModel:              clone.Model,
			VoiceCloneReferenceAudioPath: clone.ReferenceAudioPath,
			VoiceCloneReferenceText:      clone.ReferenceText,
			VoiceCloneReferenceTextPath:  clone.ReferenceTextPath,
			VoiceClonePersona:            clone.Persona,
			VoiceCloneStyle:              clone.Style,
		})
		if err != nil {
			return fmt.Errorf("voice clone TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("voice clone TTS warmup did not pass")
		}
	} else if normalizedTTSProfile == "iflytek_tts" || normalizedTTSProfile == "iflytek" || normalizedTTSProfile == "xfyun" || normalizedTTSProfile == "xfyun_tts" {
		outputDir := filepath.Join(os.TempDir(), "a21-product-chain-warmup")
		_ = os.RemoveAll(outputDir)
		defer os.RemoveAll(outputDir)
		report, err := audio.SynthesizeIflytekTTS(warmCtx, audio.LocalTTSOptions{
			Text:      "A21 warmup",
			OutputDir: outputDir,
		})
		if err != nil {
			return fmt.Errorf("Iflytek TTS warmup failed")
		}
		if report.Status != "passed" {
			return fmt.Errorf("Iflytek TTS warmup did not pass")
		}
	}
	return nil
}

func newAudioIngressConfigFromEnv(env []string) audio.IngressConfig {
	config := audio.DefaultIngressConfig()
	switch strings.ToLower(strings.TrimSpace(appEnvValue(env, "A21_VAD_PREFERENCE"))) {
	case "silero":
		config.VADPreference = audio.VADDetectorPreferenceSilero
		config.SileroRunner = audio.NewCommandSileroVADRunner(audio.CommandSileroVADRunnerConfig{
			CommandPath: appEnvValue(env, "A21_SILERO_VAD_COMMAND"),
			ModelPath:   appEnvValue(env, "A21_SILERO_VAD_MODEL"),
		})
	case "rms", "":
		config.VADPreference = audio.VADDetectorPreferenceRMS
	default:
		config.VADPreference = audio.VADDetectorPreferenceRMS
	}
	if timeoutMS, err := strconv.Atoi(strings.TrimSpace(appEnvValue(env, "A21_SILERO_VAD_TIMEOUT_MS"))); err == nil && timeoutMS > 0 {
		config.VADTimeout = time.Duration(timeoutMS) * time.Millisecond
	}
	return config
}

type v21ConfigurationErrorClient struct {
	err error
}

func (c v21ConfigurationErrorClient) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	if c.err == nil {
		return v21adapter.QueryResponse{}, fmt.Errorf("v21 adapter configuration invalid")
	}
	return v21adapter.QueryResponse{}, fmt.Errorf("v21 adapter configuration invalid: %w", c.err)
}

func appEnvValue(env []string, want string) string {
	prefix := want + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func appEnvBool(env []string, want string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnvValue(env, want))) {
	case "1", "true", "yes", "on", "professional":
		return true
	default:
		return false
	}
}
