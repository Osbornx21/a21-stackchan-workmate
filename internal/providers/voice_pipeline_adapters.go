package providers

import (
	"bufio"
	"bytes"
	"context"
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

func (a *localSherpaONNXASRAdapter) Name() string {
	return a.name
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

func sendTextStreamEvent(ctx context.Context, out chan<- TextStreamEvent, event TextStreamEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- event:
		return true
	}
}

type LocalTTSSynthesizer func(context.Context, audio.LocalTTSOptions) (audio.LocalTTSReport, error)

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

func (a *localTTSAdapter) Name() string {
	return a.name
}

func (a *localTTSAdapter) Synthesize(ctx context.Context, req TTSAdapterRequest) (<-chan VoiceAudioChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	options := a.baseOptions
	options.Text = req.Text
	options.OutputSampleRateHz = 48000
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
	chunks, err := audio.ReadPCM16MonoWAVChunksForSampleRate(report.OutputPath, 60, 48000)
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

type VoicePipelineAdapterOptions struct {
	ASRRunner      LocalASRRunner
	ASRModelDir    string
	ASRFamily      string
	TextHTTPClient *http.Client
	TextMaxTokens  int
	TTSSynthesizer LocalTTSSynthesizer
	TTSOptions     audio.LocalTTSOptions
}

func VoicePipelineAdaptersFromEnv(env []string, optionList ...VoicePipelineAdapterOptions) VoicePipelineAdapters {
	var options VoicePipelineAdapterOptions
	if len(optionList) > 0 {
		options = optionList[0]
	}
	selection := VoicePipelineSelectionFromEnv(env)
	textMaxTokens := voiceTextMaxTokensFromEnv(env, options.TextMaxTokens)
	adapters := VoicePipelineAdapters{
		ASR:           NewMockASRAdapter("mock-local-asr"),
		TextStream:    NewMockTextStreamAdapter("mock-text-stream"),
		TTS:           NewMockTTSAdapter("mock-fast-tts"),
		Selection:     selection,
		ExecutionMode: "fixture",
	}
	if isLocalSherpaASRProfile(selection.ASRProfile) {
		adapters.ASR = NewLocalSherpaONNXASRAdapter(LocalSherpaONNXASRAdapterOptions{
			Name:     selection.ASRProfile,
			ModelDir: options.ASRModelDir,
			Family:   options.ASRFamily,
			Runner:   options.ASRRunner,
		})
		adapters.ExecutionMode = "host_local"
	}
	if isOpenAITextStreamProfile(selection.LLMProfile) {
		adapters.TextStream = NewOpenAICompatibleTextStreamAdapter(OpenAICompatibleTextStreamAdapterOptions{
			Name:         selection.LLMProfile,
			ProviderName: selection.LLMProfile,
			Env:          env,
			Client:       options.TextHTTPClient,
			MaxTokens:    textMaxTokens,
		})
		adapters.ExecutionMode = "host_local"
	}
	if isOllamaTextStreamProfile(selection.LLMProfile) {
		adapters.TextStream = NewOllamaTextStreamAdapter(OllamaTextStreamAdapterOptions{
			Name:      selection.LLMProfile,
			Env:       env,
			Client:    options.TextHTTPClient,
			MaxTokens: textMaxTokens,
		})
		adapters.ExecutionMode = "host_local"
	}
	if isLocalTTSProfile(selection.TTSProfile) {
		synthesizer := options.TTSSynthesizer
		if synthesizer == nil && normalizePipelineProfile(selection.TTSProfile) == "macos_say" {
			synthesizer = audio.SynthesizeMacOSSay
		}
		adapters.TTS = NewLocalTTSAdapter(LocalTTSAdapterOptions{
			Name:        selection.TTSProfile,
			BaseOptions: options.TTSOptions,
			Synthesizer: synthesizer,
		})
		adapters.ExecutionMode = "host_local"
	}
	return adapters
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

func isOpenAITextStreamProfile(profile string) bool {
	profile = normalizePipelineProfile(profile)
	if profile == "" || profile == "mock" || profile == "mock_text_stream" || profile == "mock-text-stream" {
		return false
	}
	builtin, _, ok := ProviderProfileByNameFromEnv(nil, profile)
	return ok && builtin.Family == ProviderFamilyTextStream && builtin.Protocol == "openai_chat_completions"
}

func isOllamaTextStreamProfile(profile string) bool {
	profile = normalizePipelineProfile(profile)
	if profile != "local_ollama" {
		return false
	}
	builtin, _, ok := ProviderProfileByNameFromEnv(nil, profile)
	return ok && builtin.Family == ProviderFamilyTextStream && builtin.Protocol == "ollama_chat"
}

func isLocalTTSProfile(profile string) bool {
	switch normalizePipelineProfile(profile) {
	case "sherpa_onnx", "local_sherpa_onnx", "macos_say":
		return true
	default:
		return false
	}
}

func normalizePipelineProfile(profile string) string {
	return strings.ToLower(strings.TrimSpace(profile))
}
