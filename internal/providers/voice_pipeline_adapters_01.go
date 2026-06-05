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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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
	StreamingHelperPath     string
	StreamingPythonPath     string
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
	streamingFactory := options.StreamingSessionFactory
	if streamingFactory == nil {
		streamingFactory = newSherpaStreamingASRSubprocessFactory(sherpaStreamingASRSubprocessOptions{
			HelperPath: strings.TrimSpace(options.StreamingHelperPath),
			PythonPath: strings.TrimSpace(options.StreamingPythonPath),
			ModelDir:   strings.TrimSpace(options.ModelDir),
			Family:     strings.TrimSpace(options.Family),
		})
	}
	return &localSherpaONNXStreamingASRAdapter{
		batch:                   NewLocalSherpaONNXASRAdapter(batchOptions),
		name:                    name,
		streamingSessionFactory: streamingFactory,
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

type sherpaStreamingASRSubprocessOptions struct {
	HelperPath string
	PythonPath string
	ModelDir   string
	Family     string
}

type sherpaStreamingASRSubprocessSession struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	encoder *json.Encoder
	events  chan ASRAdapterEvent
	mu      sync.Mutex
	closed  bool
}

type sherpaStreamingASRCommand struct {
	Type         string `json:"type"`
	Seq          uint64 `json:"seq,omitempty"`
	SampleRateHz int    `json:"sample_rate_hz,omitempty"`
	Channels     int    `json:"channels,omitempty"`
	DurationMS   int    `json:"duration_ms,omitempty"`
	PCM16LEB64   string `json:"pcm16le_b64,omitempty"`
	Mode         string `json:"mode,omitempty"`
	TraceID      string `json:"trace_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	ModelDir     string `json:"model_dir,omitempty"`
	Family       string `json:"family,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type sherpaStreamingASREvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Code string `json:"code,omitempty"`
}

func newSherpaStreamingASRSubprocessFactory(options sherpaStreamingASRSubprocessOptions) StreamingASRSessionFactory {
	helperPath := strings.TrimSpace(options.HelperPath)
	modelDir := strings.TrimSpace(options.ModelDir)
	if helperPath == "" || modelDir == "" {
		return nil
	}
	family := strings.TrimSpace(options.Family)
	if family == "" {
		family = "streaming_zipformer"
	}
	return func(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error) {
		return startSherpaStreamingASRSubprocessSession(ctx, options, family, req)
	}
}

func startSherpaStreamingASRSubprocessSession(ctx context.Context, options sherpaStreamingASRSubprocessOptions, family string, req StreamingASRStartRequest) (StreamingASRSession, error) {
	helperPath := strings.TrimSpace(options.HelperPath)
	modelDir := strings.TrimSpace(options.ModelDir)
	if helperPath == "" || modelDir == "" {
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper is not configured")
	}
	cmdName := helperPath
	args := []string{}
	if pythonPath := strings.TrimSpace(options.PythonPath); pythonPath != "" {
		cmdName = pythonPath
		args = append(args, helperPath)
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	session := &sherpaStreamingASRSubprocessSession{
		cmd:     cmd,
		stdin:   stdin,
		encoder: json.NewEncoder(stdin),
		events:  make(chan ASRAdapterEvent, 8),
	}
	go func() {
		defer session.closeEvents()
		session.readEvents(stdout)
	}()
	go func() {
		_ = cmd.Wait()
	}()
	if err := session.writeCommand(ctx, sherpaStreamingASRCommand{
		Type:         "start",
		SampleRateHz: 16000,
		Channels:     1,
		Mode:         sanitizeVoicePipelineValue(req.Mode, "workmate"),
		TraceID:      "redacted",
		SessionID:    "redacted",
		ModelDir:     modelDir,
		Family:       family,
	}); err != nil {
		session.Cancel(err)
		return nil, fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	return session, nil
}

func (s *sherpaStreamingASRSubprocessSession) AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error {
	if err := validateSherpaStreamingASRFrame(frame); err != nil {
		return err
	}
	return s.writeCommand(ctx, sherpaStreamingASRCommand{
		Type:         "append",
		Seq:          frame.Seq,
		SampleRateHz: frame.SampleRateHz,
		Channels:     frame.Channels,
		DurationMS:   frame.DurationMS,
		PCM16LEB64:   base64.StdEncoding.EncodeToString(frame.PCM16LE),
	})
}

func (s *sherpaStreamingASRSubprocessSession) Events() <-chan ASRAdapterEvent {
	return s.events
}

func (s *sherpaStreamingASRSubprocessSession) Commit(ctx context.Context) error {
	return s.writeCommand(ctx, sherpaStreamingASRCommand{Type: "commit"})
}

func (s *sherpaStreamingASRSubprocessSession) Cancel(cause error) {
	_ = s.writeCommand(context.Background(), sherpaStreamingASRCommand{Type: "cancel", Reason: "cancelled"})
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		_ = s.stdin.Close()
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
	}
	s.mu.Unlock()
}

func (s *sherpaStreamingASRSubprocessSession) writeCommand(ctx context.Context, command sherpaStreamingASRCommand) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("sherpa-onnx streaming ASR helper exited")
	}
	if err := s.encoder.Encode(command); err != nil {
		s.closed = true
		_ = s.stdin.Close()
		return fmt.Errorf("sherpa-onnx streaming ASR helper failed")
	}
	return nil
}

func (s *sherpaStreamingASRSubprocessSession) readEvents(stdout io.Reader) {
	decoder := json.NewDecoder(stdout)
	for {
		var event sherpaStreamingASREvent
		if err := decoder.Decode(&event); err != nil {
			if err != io.EOF {
				sendASRAdapterEvent(context.Background(), s.events, ASRAdapterEvent{
					Finding: "sherpa-onnx streaming ASR helper protocol error",
					Err:     fmt.Errorf("sherpa-onnx streaming ASR helper protocol error"),
				})
			}
			return
		}
		switch strings.ToLower(strings.TrimSpace(event.Type)) {
		case "ready":
			continue
		case "partial":
			sendASRAdapterEvent(context.Background(), s.events, ASRAdapterEvent{Text: event.Text})
		case "final":
			sendASRAdapterEvent(context.Background(), s.events, ASRAdapterEvent{Text: event.Text, Final: true})
		case "error":
			sendASRAdapterEvent(context.Background(), s.events, ASRAdapterEvent{
				Finding: "sherpa-onnx streaming ASR helper failed",
				Err:     fmt.Errorf("sherpa-onnx streaming ASR helper failed"),
			})
			return
		default:
			sendASRAdapterEvent(context.Background(), s.events, ASRAdapterEvent{
				Finding: "sherpa-onnx streaming ASR helper protocol error",
				Err:     fmt.Errorf("sherpa-onnx streaming ASR helper protocol error"),
			})
			return
		}
	}
}

func (s *sherpaStreamingASRSubprocessSession) closeEvents() {
	s.mu.Lock()
	s.closed = true
	_ = s.stdin.Close()
	s.mu.Unlock()
	close(s.events)
}

func validateSherpaStreamingASRFrame(frame VoicePipelinePCMFrame) error {
	if strings.ToLower(strings.TrimSpace(frame.Codec)) != "pcm_s16le" {
		return fmt.Errorf("sherpa-onnx streaming ASR frame codec must be pcm_s16le")
	}
	if frame.SampleRateHz <= 0 || frame.Channels != 1 || frame.DurationMS <= 0 {
		return fmt.Errorf("sherpa-onnx streaming ASR frame metadata invalid")
	}
	if len(frame.PCM16LE) == 0 || len(frame.PCM16LE)%2 != 0 {
		return fmt.Errorf("sherpa-onnx streaming ASR frame payload invalid")
	}
	return nil
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
