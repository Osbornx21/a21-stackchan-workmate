package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"a21.local/a21/internal/audio"
)

const (
	defaultVoiceTextMaxTokens = 24
	minVoiceTextMaxTokens     = 8
	maxVoiceTextMaxTokens     = 96
)

type LocalASRRunner func(context.Context, audio.LocalASROptions) (audio.LocalASRResult, error)

type StreamingASRSessionFactory func(context.Context, StreamingASRStartRequest) (StreamingASRSession, error)

type LocalSherpaONNXASRAdapterOptions struct {
	Name       string
	OutputDir  string
	PythonPath string
	ScriptPath string
	ModelDir   string
	Family     string
	TempDir    string
	Runner     LocalASRRunner
}

type LocalSherpaONNXStreamingASRAdapterOptions struct {
	LocalSherpaONNXASRAdapterOptions
	StreamingSessionFactory StreamingASRSessionFactory
}

type localSherpaONNXASRAdapter struct {
	name       string
	outputDir  string
	pythonPath string
	scriptPath string
	modelDir   string
	family     string
	tempDir    string
	runner     LocalASRRunner
}

type localSherpaONNXStreamingASRAdapter struct {
	batch                   ASRAdapter
	name                    string
	streamingSessionFactory StreamingASRSessionFactory
}

func NewLocalSherpaONNXASRAdapter(options LocalSherpaONNXASRAdapterOptions) ASRAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "sherpa_onnx"
	}
	runner := options.Runner
	if runner == nil {
		runner = audio.RunSherpaONNXASR
	}
	return &localSherpaONNXASRAdapter{
		name:       name,
		outputDir:  options.OutputDir,
		pythonPath: options.PythonPath,
		scriptPath: options.ScriptPath,
		modelDir:   options.ModelDir,
		family:     options.Family,
		tempDir:    options.TempDir,
		runner:     runner,
	}
}

func NewLocalSherpaONNXStreamingASRAdapter(options LocalSherpaONNXStreamingASRAdapterOptions) ASRAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "sherpa_onnx_streaming"
	}
	batchOptions := options.LocalSherpaONNXASRAdapterOptions
	batchOptions.Name = name
	return &localSherpaONNXStreamingASRAdapter{
		batch:                   NewLocalSherpaONNXASRAdapter(batchOptions),
		name:                    name,
		streamingSessionFactory: options.StreamingSessionFactory,
	}
}

func (a *localSherpaONNXASRAdapter) Name() string {
	return a.name
}

func (a *localSherpaONNXStreamingASRAdapter) Name() string {
	return a.name
}

func (a *localSherpaONNXStreamingASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	return a.batch.Transcribe(ctx, req)
}

func (a *localSherpaONNXStreamingASRAdapter) StartStreamingASR(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.streamingSessionFactory == nil {
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper is not configured")
	}
	session, err := a.streamingSessionFactory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	return session, nil
}

func (a *localSherpaONNXASRAdapter) Transcribe(ctx context.Context, req ASRAdapterRequest) (<-chan ASRAdapterEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pcm, sampleRateHz, err := pcm16PayloadFromFrames(req.Frames)
	if err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp(a.tempDir, "a21-sherpa-onnx-asr-adapter-*")
	if err != nil {
		return nil, fmt.Errorf("sherpa-onnx ASR adapter failed to allocate temp input")
	}
	wavPath := filepath.Join(tempDir, "a21-asr-input.wav")
	if err := audio.WritePCM16MonoWAV(wavPath, sampleRateHz, pcm); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("sherpa-onnx ASR adapter failed to prepare wav input")
	}

	events := make(chan ASRAdapterEvent, 2)
	go func() {
		defer close(events)
		defer os.RemoveAll(tempDir)
		result, err := a.runner(ctx, audio.LocalASROptions{
			OutputDir:  a.outputDir,
			PythonPath: a.pythonPath,
			ScriptPath: a.scriptPath,
			ModelDir:   a.modelDir,
			Family:     a.family,
			WAVPath:    wavPath,
		})
		if err != nil {
			sendASRAdapterEvent(ctx, events, ASRAdapterEvent{
				Finding: "sherpa-onnx ASR adapter failed",
				Err:     fmt.Errorf("sherpa-onnx ASR adapter failed"),
			})
			return
		}
		if result.Report.Status != "" && result.Report.Status != "passed" {
			sendASRAdapterEvent(ctx, events, ASRAdapterEvent{
				Finding: "sherpa-onnx ASR adapter did not pass",
				Err:     fmt.Errorf("sherpa-onnx ASR adapter did not pass"),
			})
			return
		}
		transcript := strings.TrimSpace(result.Transcript)
		if transcript != "" {
			if !sendASRAdapterEvent(ctx, events, ASRAdapterEvent{Text: transcript}) {
				return
			}
		}
		sendASRAdapterEvent(ctx, events, ASRAdapterEvent{Text: transcript, Final: true})
	}()
	return events, nil
}

func pcm16PayloadFromFrames(frames []VoicePipelinePCMFrame) ([]byte, int, error) {
	if len(frames) == 0 {
		return nil, 0, fmt.Errorf("pcm frame payload is required")
	}
	var sampleRateHz int
	var pcm []byte
	for _, frame := range frames {
		if strings.ToLower(strings.TrimSpace(frame.Codec)) != "pcm_s16le" {
			return nil, 0, fmt.Errorf("pcm frame codec must be pcm_s16le")
		}
		if frame.Channels != 1 {
			return nil, 0, fmt.Errorf("pcm frame channels must be mono")
		}
		if frame.SampleRateHz <= 0 {
			return nil, 0, fmt.Errorf("pcm frame sample rate is required")
		}
		if sampleRateHz == 0 {
			sampleRateHz = frame.SampleRateHz
		}
		if frame.SampleRateHz != sampleRateHz {
			return nil, 0, fmt.Errorf("pcm frame sample rates must match")
		}
		if len(frame.PCM16LE) == 0 {
			return nil, 0, fmt.Errorf("pcm frame payload is required")
		}
		if len(frame.PCM16LE)%2 != 0 {
			return nil, 0, fmt.Errorf("pcm frame payload must contain 16-bit samples")
		}
		pcm = append(pcm, frame.PCM16LE...)
	}
	if len(pcm) == 0 {
		return nil, 0, fmt.Errorf("pcm frame payload is required")
	}
	return pcm, sampleRateHz, nil
}

func sendASRAdapterEvent(ctx context.Context, out chan<- ASRAdapterEvent, event ASRAdapterEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- event:
		return true
	}
}

type OpenAICompatibleTextStreamAdapterOptions struct {
	Name         string
	ProviderName string
	Env          []string
	Client       *http.Client
	MaxTokens    int
}

type OllamaTextStreamAdapterOptions struct {
	Name      string
	Env       []string
	Client    *http.Client
	MaxTokens int
}

type FallbackTextStreamAdapterOptions struct {
	Name             string
	Primary          TextStreamAdapter
	Fallback         TextStreamAdapter
	FallbackProvider string
	Reason           string
}

type ollamaTextStreamAdapter struct {
	name      string
	env       []string
	client    *http.Client
	maxTokens int
}

func NewOllamaTextStreamAdapter(options OllamaTextStreamAdapterOptions) TextStreamAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "local_ollama"
	}
	return &ollamaTextStreamAdapter{
		name:      name,
		env:       append([]string(nil), options.Env...),
		client:    options.Client,
		maxTokens: options.MaxTokens,
	}
}

func (a *ollamaTextStreamAdapter) Name() string {
	return a.name
}

func (a *ollamaTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := RunTextStreamCompletionFromEnv(ctx, a.env, TextStreamCompletionOptions{
		ProviderName: "local_ollama",
		Prompt:       req.Text,
		MaxTokens:    a.maxTokens,
		Client:       a.client,
	})
	if err != nil {
		return nil, err
	}
	events := make(chan TextStreamEvent, 2)
	defer close(events)
	if strings.TrimSpace(result.ContentText) != "" {
		events <- TextStreamEvent{Kind: TextStreamDeltaContent, Text: result.ContentText}
	}
	events <- TextStreamEvent{Kind: TextStreamDeltaDone}
	return events, nil
}

type openAICompatibleTextStreamAdapter struct {
	name         string
	providerName string
	env          []string
	client       *http.Client
	maxTokens    int
}

func NewOpenAICompatibleTextStreamAdapter(options OpenAICompatibleTextStreamAdapterOptions) TextStreamAdapter {
	providerName := strings.TrimSpace(options.ProviderName)
	if providerName == "" {
		providerName = strings.TrimSpace(options.Name)
	}
	if providerName == "" {
		providerName = "deepseek"
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = providerName
	}
	return &openAICompatibleTextStreamAdapter{
		name:         name,
		providerName: providerName,
		env:          append([]string(nil), options.Env...),
		client:       options.Client,
		maxTokens:    options.MaxTokens,
	}
}

func (a *openAICompatibleTextStreamAdapter) Name() string {
	return a.name
}

func (a *openAICompatibleTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	httpReq, client, err := a.newRequest(ctx, req.Text)
	if err != nil {
		return nil, err
	}
	events := make(chan TextStreamEvent, 8)
	go a.streamEvents(ctx, client, httpReq, events)
	return events, nil
}

func (a *openAICompatibleTextStreamAdapter) newRequest(ctx context.Context, text string) (*http.Request, *http.Client, error) {
	env := a.env
	rawProvider := strings.TrimSpace(a.providerName)
	if rawProvider == "" {
		rawProvider = strings.TrimSpace(envValue(env, "A21_TEXT_STREAM_PROFILE"))
	}
	if rawProvider == "" {
		rawProvider = strings.TrimSpace(envValue(env, "A21_PROVIDER_PRIMARY"))
	}
	if rawProvider == "" {
		rawProvider = "deepseek"
	}
	provider := strings.ToLower(rawProvider)
	if containsLegacyProviderIdentity(provider) || containsBlockedProviderIdentity(provider) {
		return nil, nil, fmt.Errorf("text stream provider target rejected")
	}
	profile, _, ok := ProviderProfileByNameFromEnv(env, provider)
	if !ok {
		return nil, nil, fmt.Errorf("text stream provider target is unknown")
	}
	spec := providerSmokeSpecFromProfile(profile)
	configured, missing := providerSmokeConfigured(env, spec)
	if !configured {
		return nil, nil, fmt.Errorf("text stream provider missing env: %s", strings.Join(missing, ","))
	}
	if !spec.Executable {
		return nil, nil, fmt.Errorf("text stream provider protocol is not executable")
	}
	endpoint, err := providerSmokeEndpoint(env, spec)
	if err != nil {
		return nil, nil, err
	}
	prompt := strings.TrimSpace(text)
	if prompt == "" {
		prompt = "A21 local voice loopback. Reply briefly."
	}
	maxTokens := a.maxTokens
	if maxTokens <= 0 {
		maxTokens = 48
	}
	body := map[string]any{
		"model": providerSmokeModel(env, spec),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": maxTokens,
		"stream":     true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, err
	}
	if spec.APIKeyEnv != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(envValue(env, spec.APIKeyEnv)))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("User-Agent", "a21-text-stream-adapter/0.1")
	client := a.client
	if client == nil {
		policy, _ := NetworkPolicyFromEnv(env)
		client, err = NewProviderHTTPClient(policy)
		if err != nil {
			return nil, nil, err
		}
	}
	return httpReq, client, nil
}

func (a *openAICompatibleTextStreamAdapter) streamEvents(ctx context.Context, client *http.Client, req *http.Request, out chan<- TextStreamEvent) {
	defer close(out)
	resp, err := client.Do(req)
	if err != nil {
		sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream adapter failed", Err: fmt.Errorf("text stream adapter failed")})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream adapter HTTP status was not successful", Err: fmt.Errorf("text stream adapter HTTP status was not successful")})
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		parsedEvents, done, err := parseOpenAICompatibleTextStreamLine(strings.TrimSpace(scanner.Text()))
		if err != nil {
			sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream adapter parse failed", Err: fmt.Errorf("text stream adapter parse failed")})
			return
		}
		if done {
			if !sendTextStreamEvent(ctx, out, TextStreamEvent{Kind: TextStreamDeltaDone}) {
				return
			}
			continue
		}
		for _, event := range parsedEvents {
			if event.Text == "" {
				continue
			}
			if !sendTextStreamEvent(ctx, out, event) {
				return
			}
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream adapter read failed", Err: fmt.Errorf("text stream adapter read failed")})
	}
}

type fallbackTextStreamAdapter struct {
	name             string
	primary          TextStreamAdapter
	fallback         TextStreamAdapter
	fallbackProvider string
	reason           string
}

func NewFallbackTextStreamAdapter(options FallbackTextStreamAdapterOptions) TextStreamAdapter {
	name := strings.TrimSpace(options.Name)
	primaryName := ""
	if options.Primary != nil {
		primaryName = options.Primary.Name()
	}
	fallbackProvider := strings.TrimSpace(options.FallbackProvider)
	if fallbackProvider == "" && options.Fallback != nil {
		fallbackProvider = options.Fallback.Name()
	}
	if name == "" {
		name = primaryName
		if fallbackProvider != "" {
			name = strings.TrimSpace(primaryName + "->" + fallbackProvider)
		}
	}
	reason := strings.TrimSpace(options.Reason)
	if reason == "" {
		reason = "primary_failed"
	}
	return &fallbackTextStreamAdapter{
		name:             name,
		primary:          options.Primary,
		fallback:         options.Fallback,
		fallbackProvider: fallbackProvider,
		reason:           reason,
	}
}

func (a *fallbackTextStreamAdapter) Name() string {
	return a.name
}

func (a *fallbackTextStreamAdapter) StreamText(ctx context.Context, req TextStreamAdapterRequest) (<-chan TextStreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.primary == nil {
		return a.startFallback(ctx, req, "primary_unavailable"), nil
	}
	primaryEvents, err := a.primary.StreamText(ctx, req)
	if err != nil {
		return a.startFallback(ctx, req, a.reason), nil
	}
	out := make(chan TextStreamEvent, 8)
	go func() {
		defer close(out)
		contentSeen := false
		for event := range primaryEvents {
			if event.Err != nil && !contentSeen {
				a.forwardFallback(ctx, req, out, a.reason)
				return
			}
			if event.Kind == TextStreamDeltaDone && !contentSeen {
				a.forwardFallback(ctx, req, out, "primary_empty")
				return
			}
			if event.Kind == TextStreamDeltaContent && strings.TrimSpace(event.Text) != "" {
				contentSeen = true
			}
			if !contentSeen && event.Kind == TextStreamDeltaReasoning {
				continue
			}
			if !sendTextStreamEvent(ctx, out, event) {
				return
			}
			if event.Err != nil {
				return
			}
		}
		if !contentSeen {
			a.forwardFallback(ctx, req, out, "primary_empty")
		}
	}()
	return out, nil
}

func (a *fallbackTextStreamAdapter) startFallback(ctx context.Context, req TextStreamAdapterRequest, reason string) <-chan TextStreamEvent {
	out := make(chan TextStreamEvent, 8)
	go func() {
		defer close(out)
		a.forwardFallback(ctx, req, out, reason)
	}()
	return out
}

func (a *fallbackTextStreamAdapter) forwardFallback(ctx context.Context, req TextStreamAdapterRequest, out chan<- TextStreamEvent, reason string) {
	provider := safeVoicePipelineProfileName(a.fallbackProvider)
	if provider == "" || provider == "unknown_provider" {
		provider = "fallback"
	}
	if !sendTextStreamEvent(ctx, out, TextStreamEvent{
		Finding: "provider_fallback_used",
		Fallback: &TextStreamFallbackEvent{
			Activated: true,
			Provider:  provider,
			Reason:    sanitizeVoicePipelineValue(reason, "primary_failed"),
		},
	}) {
		return
	}
	if a.fallback == nil {
		sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream fallback adapter missing", Err: fmt.Errorf("text stream fallback adapter missing")})
		return
	}
	fallbackEvents, err := a.fallback.StreamText(ctx, req)
	if err != nil {
		sendTextStreamEvent(ctx, out, TextStreamEvent{Finding: "text stream fallback adapter failed", Err: fmt.Errorf("text stream fallback adapter failed")})
		return
	}
	for event := range fallbackEvents {
		if !sendTextStreamEvent(ctx, out, event) {
			return
		}
	}
}

func sendTextStreamEvent(ctx context.Context, out chan<- TextStreamEvent, event TextStreamEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- event:
		return true
	}
}

type LocalTTSSynthesizer func(context.Context, audio.LocalTTSOptions) (audio.LocalTTSReport, error)

type DoubaoRealtimeTTSTTSAdapterOptions struct {
	Name     string
	Env      []string
	Provider *DoubaoRealtimeTTSProvider
	Dialer   RealtimeDialer
}

type LocalTTSAdapterOptions struct {
	Name        string
	BaseOptions audio.LocalTTSOptions
	Synthesizer LocalTTSSynthesizer
}

type localTTSAdapter struct {
	name        string
	baseOptions audio.LocalTTSOptions
	synthesizer LocalTTSSynthesizer
}

type doubaoRealtimeTTSTTSAdapter struct {
	name     string
	env      []string
	provider *DoubaoRealtimeTTSProvider
	dialer   RealtimeDialer
}

func NewLocalTTSAdapter(options LocalTTSAdapterOptions) TTSAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "sherpa_onnx"
	}
	synthesizer := options.Synthesizer
	if synthesizer == nil {
		synthesizer = audio.SynthesizeSherpaONNX
	}
	return &localTTSAdapter{name: name, baseOptions: options.BaseOptions, synthesizer: synthesizer}
}

func NewDoubaoRealtimeTTSTTSAdapter(options DoubaoRealtimeTTSTTSAdapterOptions) TTSAdapter {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "doubao_tts_realtime"
	}
	return &doubaoRealtimeTTSTTSAdapter{
		name:     name,
		env:      append([]string(nil), options.Env...),
		provider: options.Provider,
		dialer:   options.Dialer,
	}
}

func (a *localTTSAdapter) Name() string {
	return a.name
}

func (a *doubaoRealtimeTTSTTSAdapter) Name() string {
	return a.name
}

func (a *doubaoRealtimeTTSTTSAdapter) StreamingTTSAdapter() {}

func (a *localTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	options := a.baseOptions
	options.Text = req.Text
	options.OutputSampleRateHz = 24000
	cleanupDir := ""
	if strings.TrimSpace(options.OutputDir) == "" {
		tempDir, err := os.MkdirTemp("", "a21-local-tts-adapter-*")
		if err != nil {
			return nil, fmt.Errorf("local TTS adapter failed to allocate temp output")
		}
		options.OutputDir = tempDir
		cleanupDir = tempDir
	}
	if cleanupDir != "" {
		defer os.RemoveAll(cleanupDir)
	}
	report, err := a.synthesizer(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("local TTS adapter failed")
	}
	if report.Status != "" && report.Status != "passed" {
		return nil, fmt.Errorf("local TTS adapter did not pass")
	}
	if strings.TrimSpace(report.OutputPath) == "" {
		return nil, fmt.Errorf("local TTS adapter output wav missing")
	}
	chunks, err := audio.ReadPCM16MonoWAVChunksForSampleRate(report.OutputPath, 60, 24000)
	if err != nil {
		return nil, fmt.Errorf("local TTS adapter output wav invalid")
	}
	out := make(chan VoiceAudioChunk, len(chunks))
	defer close(out)
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out <- VoiceAudioChunk{
			Codec:        "pcm_s16le",
			SampleRateHz: chunk.SampleRateHz,
			Channels:     chunk.Channels,
			DurationMS:   chunk.DurationMS,
			DataBase64:   chunk.DataBase64,
		}
	}
	return out, nil
}

func (a *doubaoRealtimeTTSTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider := a.provider
	if provider == nil {
		provider = NewDoubaoRealtimeTTSProviderFromEnv(a.env, a.dialer)
	}
	session, err := provider.StartRealtimeTTSSession(ctx, req.Session)
	if err != nil {
		return nil, fmt.Errorf("doubao realtime TTS adapter failed to start")
	}
	if err := session.SendText(ctx, req.Text); err != nil {
		_ = session.Close(ctx)
		return nil, fmt.Errorf("doubao realtime TTS adapter failed to send text")
	}
	if err := session.TextDone(ctx); err != nil {
		_ = session.Close(ctx)
		return nil, fmt.Errorf("doubao realtime TTS adapter failed to finish text")
	}
	out := make(chan VoiceAudioChunk, 4)
	go func() {
		defer close(out)
		defer session.Close(context.Background())
		chunker := newPCM16Mono60MSChunker(session.outputSampleRate)
		for {
			raw, err := session.session.ReadEvent(ctx)
			if err != nil {
				if err == io.EOF {
					for _, chunk := range chunker.Flush() {
						if !sendVoiceAudioChunk(ctx, out, chunk) {
							return
						}
					}
				}
				return
			}
			event, ok, err := session.ServerEventToVoiceEvent(raw)
			if err != nil || !ok || event.Audio == nil {
				continue
			}
			chunks, err := chunker.AppendBase64(event.Audio.DataBase64)
			if err != nil {
				return
			}
			for _, chunk := range chunks {
				if !sendVoiceAudioChunk(ctx, out, chunk) {
					return
				}
			}
		}
	}()
	return out, nil
}

type pcm16Mono60MSChunker struct {
	sampleRateHz int
	frameBytes   int
	buffer       []byte
}

func newPCM16Mono60MSChunker(sampleRateHz int) *pcm16Mono60MSChunker {
	if sampleRateHz == 0 {
		sampleRateHz = 24000
	}
	return &pcm16Mono60MSChunker{
		sampleRateHz: sampleRateHz,
		frameBytes:   sampleRateHz * 60 / 1000 * 2,
	}
}

func (c *pcm16Mono60MSChunker) AppendBase64(value string) ([]VoiceAudioChunk, error) {
	pcm, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(pcm) == 0 {
		return nil, fmt.Errorf("streaming TTS adapter received invalid PCM delta")
	}
	if len(pcm)%2 != 0 {
		return nil, fmt.Errorf("streaming TTS adapter received odd PCM delta")
	}
	c.buffer = append(c.buffer, pcm...)
	return c.drain(false), nil
}

func (c *pcm16Mono60MSChunker) Flush() []VoiceAudioChunk {
	return c.drain(true)
}

func (c *pcm16Mono60MSChunker) drain(final bool) []VoiceAudioChunk {
	var chunks []VoiceAudioChunk
	for len(c.buffer) >= c.frameBytes {
		chunks = append(chunks, c.chunkFromPCM(c.buffer[:c.frameBytes]))
		c.buffer = c.buffer[c.frameBytes:]
	}
	if final && len(c.buffer) > 0 {
		padded := make([]byte, c.frameBytes)
		copy(padded, c.buffer)
		chunks = append(chunks, c.chunkFromPCM(padded))
		c.buffer = nil
	}
	return chunks
}

func (c *pcm16Mono60MSChunker) chunkFromPCM(pcm []byte) VoiceAudioChunk {
	return VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: c.sampleRateHz,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   base64.StdEncoding.EncodeToString(pcm),
	}
}

func sendVoiceAudioChunk(ctx context.Context, out chan<- VoiceAudioChunk, chunk VoiceAudioChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- chunk:
		return true
	}
}

type VoicePipelineAdapterOptions struct {
	ASRRunner                  LocalASRRunner
	ASRModelDir                string
	ASRFamily                  string
	StreamingASRSessionFactory StreamingASRSessionFactory
	TextHTTPClient             *http.Client
	TextMaxTokens              int
	TTSSynthesizer             LocalTTSSynthesizer
	TTSRealtimeDialer          RealtimeDialer
	TTSOptions                 audio.LocalTTSOptions
}

func VoicePipelineAdaptersFromEnv(env []string, optionList ...VoicePipelineAdapterOptions) VoicePipelineAdapters {
	var options VoicePipelineAdapterOptions
	if len(optionList) > 0 {
		options = optionList[0]
	}
	ttsOptions := localTTSOptionsFromEnv(env, options.TTSOptions)
	selection := VoicePipelineSelectionFromEnv(env)
	textMaxTokens := voiceTextMaxTokensFromEnv(env, options.TextMaxTokens)
	adapters := VoicePipelineAdapters{
		ASR:           NewMockASRAdapter("mock-local-asr"),
		TextStream:    NewMockTextStreamAdapter("mock-text-stream"),
		TTS:           NewMockTTSAdapter("mock-fast-tts"),
		Selection:     selection,
		ExecutionMode: "fixture",
	}
	if isLocalSherpaStreamingASRProfile(selection.ASRProfile) {
		adapters.ASR = NewLocalSherpaONNXStreamingASRAdapter(LocalSherpaONNXStreamingASRAdapterOptions{
			LocalSherpaONNXASRAdapterOptions: LocalSherpaONNXASRAdapterOptions{
				Name:     selection.ASRProfile,
				ModelDir: options.ASRModelDir,
				Family:   options.ASRFamily,
				Runner:   options.ASRRunner,
			},
			StreamingSessionFactory: options.StreamingASRSessionFactory,
		})
		adapters.ExecutionMode = "host_local"
	} else if isLocalSherpaASRProfile(selection.ASRProfile) {
		adapters.ASR = NewLocalSherpaONNXASRAdapter(LocalSherpaONNXASRAdapterOptions{
			Name:     selection.ASRProfile,
			ModelDir: options.ASRModelDir,
			Family:   options.ASRFamily,
			Runner:   options.ASRRunner,
		})
		adapters.ExecutionMode = "host_local"
	}
	if textStream, ok := textStreamAdapterForPipelineProfile(env, selection.LLMProfile, options.TextHTTPClient, textMaxTokens); ok {
		if fallback, fallbackOK := textStreamAdapterForPipelineProfile(env, selection.LLMFallbackProfile, options.TextHTTPClient, textMaxTokens); fallbackOK {
			textStream = NewFallbackTextStreamAdapter(FallbackTextStreamAdapterOptions{
				Primary:          textStream,
				Fallback:         fallback,
				FallbackProvider: selection.LLMFallbackProfile,
				Reason:           "primary_failed",
			})
		}
		adapters.TextStream = textStream
		adapters.ExecutionMode = "host_local"
	}
	if isDoubaoRealtimeTTSProfile(selection.TTSProfile) {
		adapters.TTS = NewDoubaoRealtimeTTSTTSAdapter(DoubaoRealtimeTTSTTSAdapterOptions{
			Name:   selection.TTSProfile,
			Env:    env,
			Dialer: options.TTSRealtimeDialer,
		})
		adapters.ExecutionMode = "host_local"
	} else if isLocalTTSProfile(selection.TTSProfile) {
		synthesizer := options.TTSSynthesizer
		if synthesizer == nil && normalizePipelineProfile(selection.TTSProfile) == "macos_say" {
			synthesizer = audio.SynthesizeMacOSSay
		}
		if synthesizer == nil && normalizePipelineProfile(selection.TTSProfile) == "voice_clone_cli" {
			synthesizer = audio.SynthesizeVoiceCloneCLI
		}
		if synthesizer == nil && isIflytekTTSProfile(selection.TTSProfile) {
			synthesizer = audio.SynthesizeIflytekTTS
		}
		adapters.TTS = NewLocalTTSAdapter(LocalTTSAdapterOptions{
			Name:        selection.TTSProfile,
			BaseOptions: ttsOptions,
			Synthesizer: synthesizer,
		})
		adapters.ExecutionMode = "host_local"
	}
	return adapters
}

func textStreamAdapterForPipelineProfile(env []string, profileName string, client *http.Client, maxTokens int) (TextStreamAdapter, bool) {
	if profile, ok := openAITextStreamProfileFromEnv(env, profileName); ok {
		return NewOpenAICompatibleTextStreamAdapter(OpenAICompatibleTextStreamAdapterOptions{
			Name:         profile.Name,
			ProviderName: profile.Name,
			Env:          env,
			Client:       client,
			MaxTokens:    maxTokens,
		}), true
	}
	if profile, ok := ollamaTextStreamProfileFromEnv(env, profileName); ok {
		return NewOllamaTextStreamAdapter(OllamaTextStreamAdapterOptions{
			Name:      profile.Name,
			Env:       env,
			Client:    client,
			MaxTokens: maxTokens,
		}), true
	}
	return nil, false
}

func voiceTextMaxTokensFromEnv(env []string, override int) int {
	if override > 0 {
		return clampVoiceTextMaxTokens(override)
	}
	raw := strings.TrimSpace(envValue(env, "A21_VOICE_TEXT_MAX_TOKENS"))
	if raw == "" {
		return defaultVoiceTextMaxTokens
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVoiceTextMaxTokens
	}
	return clampVoiceTextMaxTokens(value)
}

func clampVoiceTextMaxTokens(value int) int {
	switch {
	case value < minVoiceTextMaxTokens:
		return minVoiceTextMaxTokens
	case value > maxVoiceTextMaxTokens:
		return maxVoiceTextMaxTokens
	default:
		return value
	}
}

func isLocalSherpaASRProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "sherpa_onnx", "local_sherpa_onnx":
		return true
	default:
		return false
	}
}

func isLocalSherpaStreamingASRProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "sherpa_onnx_streaming", "local_sherpa_onnx_streaming", "streaming_zipformer":
		return true
	default:
		return false
	}
}

func openAITextStreamProfileFromEnv(env []string, profile string) (ProviderProfile, bool) {
	return textStreamProfileFromEnv(env, profile, "openai_chat_completions")
}

func ollamaTextStreamProfileFromEnv(env []string, profile string) (ProviderProfile, bool) {
	return textStreamProfileFromEnv(env, profile, "ollama_chat")
}

func textStreamProfileFromEnv(env []string, profile string, protocol string) (ProviderProfile, bool) {
	profile = normalizePipelineProfile(profile)
	if profile == "" || profile == "mock" || profile == "mock_text_stream" || profile == "mock-text-stream" {
		return ProviderProfile{}, false
	}
	found, _, ok := ProviderProfileByNameFromEnv(env, profile)
	return found, ok && found.Family == ProviderFamilyTextStream && found.Protocol == protocol
}

func isLocalTTSProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "sherpa_onnx", "sherpa_onnx_tts", "local_sherpa_onnx", "local_sherpa_onnx_tts", "macos_say", "voice_clone_cli", "iflytek_tts", "iflytek", "xfyun", "xfyun_tts":
		return true
	default:
		return false
	}
}

func isDoubaoRealtimeTTSProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "doubao_tts_realtime", "doubao_realtime_tts":
		return true
	default:
		return false
	}
}

func isIflytekTTSProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "iflytek_tts", "iflytek", "xfyun", "xfyun_tts":
		return true
	default:
		return false
	}
}

func localTTSOptionsFromEnv(env []string, base audio.LocalTTSOptions) audio.LocalTTSOptions {
	out := base
	if strings.TrimSpace(out.Voice) == "" {
		out.Voice = strings.TrimSpace(envValue(env, "A21_LOCAL_TTS_VOICE"))
	}
	if strings.TrimSpace(out.ModelDir) == "" {
		out.ModelDir = strings.TrimSpace(envValue(env, "A21_SHERPA_ONNX_MODEL_DIR"))
	}
	if out.SpeakerID <= 0 {
		if value, err := strconv.Atoi(strings.TrimSpace(envValue(env, "A21_SHERPA_ONNX_SPEAKER_ID"))); err == nil && value > 0 {
			out.SpeakerID = value
		}
	}
	if strings.TrimSpace(out.VoiceCloneCommand) == "" {
		out.VoiceCloneCommand = strings.TrimSpace(envValue(env, "A21_VOICE_CLONE_COMMAND"))
	}
	if strings.TrimSpace(out.VoiceCloneModel) == "" {
		out.VoiceCloneModel = strings.TrimSpace(envValue(env, "A21_VOICE_CLONE_MODEL"))
	}
	if strings.TrimSpace(out.VoiceCloneReferenceAudioPath) == "" {
		out.VoiceCloneReferenceAudioPath = strings.TrimSpace(envValue(env, "A21_VOICE_CLONE_REF_AUDIO"))
	}
	if strings.TrimSpace(out.VoiceCloneReferenceText) == "" {
		out.VoiceCloneReferenceText = strings.TrimSpace(envValue(env, "A21_VOICE_CLONE_REF_TEXT"))
	}
	if strings.TrimSpace(out.VoiceCloneReferenceTextPath) == "" {
		out.VoiceCloneReferenceTextPath = strings.TrimSpace(envValue(env, "A21_VOICE_CLONE_REF_TEXT_FILE"))
	}
	if strings.TrimSpace(out.VoiceClonePersona) == "" {
		out.VoiceClonePersona = strings.TrimSpace(envValue(env, "A21_VOICE_PERSONA"))
	}
	if strings.TrimSpace(out.VoiceCloneStyle) == "" {
		out.VoiceCloneStyle = strings.TrimSpace(envValue(env, "A21_VOICE_STYLE"))
	}
	return out
}

func normalizePipelineProfile(profile string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(profile)), "-", "_")
}
