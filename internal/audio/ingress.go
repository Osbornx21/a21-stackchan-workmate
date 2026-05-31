package audio

import (
	"context"
	"encoding/base64"
	"errors"
	"math"
	"sync"
	"time"
)

type Event string

const (
	EventVADSpeechStart Event = "vad.speech.start"
	EventVADSpeechEnd   Event = "vad.speech.end"
)

type IngressConfig struct {
	MaxBufferedFrames int
	SpeechThreshold   float64
	SilenceHangover   int
	VADDetector       VADDetector
	VADPreference     VADDetectorPreference
	SileroRunner      SileroVADRunner
	VADTimeout        time.Duration
}

type Frame struct {
	DeviceID     string
	TraceID      string
	SessionID    string
	Seq          uint64
	SampleRateHz int
	Channels     int
	DurationMS   int
	DataBase64   string
}

type IngressResult struct {
	BufferedFrames    int
	DroppedFrames     int
	DroppedFrameDelta int
	RMS               float64
	VADDetector       string
	VADStatus         string
	VADFinding        string
	SpeechDetected    bool
	SpeechActive      bool
	Events            []Event
}

type VADDecision struct {
	SpeechDetected bool
	Score          float64
	Detector       string
	Status         string
	Finding        string
}

type VADDetector interface {
	Detect(ctx context.Context, frame Frame) VADDecision
}

type SileroVADRunner interface {
	Detect(ctx context.Context, frame Frame) (VADDecision, error)
}

type Ingress struct {
	mu     sync.Mutex
	config IngressConfig
	vad    VADDetector
	state  map[string]*streamState
}

type RMSVADDetector struct {
	threshold float64
}

type SileroVADAdapter struct {
	runner   SileroVADRunner
	fallback RMSVADDetector
	timeout  time.Duration
}

type streamState struct {
	buffer        []Frame
	dropped       int
	speechActive  bool
	silenceFrames int
}

const (
	VADDetectorRMS               = "a21-rms-vad"
	VADDetectorSilero            = "a21-silero-vad"
	VADDetectorSileroFallbackRMS = "a21-silero-vad-fallback-rms"

	VADDetectorPreferenceRMS    VADDetectorPreference = "rms"
	VADDetectorPreferenceSilero VADDetectorPreference = "silero"

	VADStatusRMSBaseline          = "rms_baseline"
	VADStatusSileroAvailable      = "silero_available"
	VADStatusSileroUnavailable    = "silero_unavailable"
	VADFindingRMSBaseline         = "rms_baseline_dev_only"
	VADFindingSileroAvailable     = "silero_runner_available"
	VADFindingSileroNotSelected   = "silero_not_selected"
	VADFindingSileroRunnerEmpty   = "silero_runner_missing_rms_fallback"
	VADFindingSileroRunnerError   = "silero_runner_error_rms_fallback"
	VADFindingSileroRunnerAbort   = "silero_runner_canceled_rms_fallback"
	VADFindingSileroRunnerTimeout = "silero_runner_timeout_rms_fallback"

	DefaultSileroVADTimeout = 75 * time.Millisecond
)

type VADDetectorPreference string

type VADDetectorConfig struct {
	Preference        VADDetectorPreference
	SileroRunner      SileroVADRunner
	FallbackThreshold float64
	Timeout           time.Duration
}

type SileroVADAdapterOptions struct {
	Runner            SileroVADRunner
	FallbackThreshold float64
	Timeout           time.Duration
}

func DefaultIngressConfig() IngressConfig {
	return IngressConfig{
		MaxBufferedFrames: 8,
		SpeechThreshold:   0.02,
		SilenceHangover:   2,
	}
}

func NewIngress(config IngressConfig) *Ingress {
	defaults := DefaultIngressConfig()
	if config.MaxBufferedFrames <= 0 {
		config.MaxBufferedFrames = defaults.MaxBufferedFrames
	}
	if config.SpeechThreshold <= 0 {
		config.SpeechThreshold = defaults.SpeechThreshold
	}
	if config.SilenceHangover <= 0 {
		config.SilenceHangover = defaults.SilenceHangover
	}
	vad := config.VADDetector
	if vad == nil {
		vad = NewVADDetector(VADDetectorConfig{
			Preference:        config.VADPreference,
			SileroRunner:      config.SileroRunner,
			FallbackThreshold: config.SpeechThreshold,
			Timeout:           config.VADTimeout,
		})
	}
	return &Ingress{config: config, vad: vad, state: make(map[string]*streamState)}
}

func (i *Ingress) Push(frame Frame) IngressResult {
	return i.PushContext(context.Background(), frame)
}

func (i *Ingress) PushContext(ctx context.Context, frame Frame) IngressResult {
	if ctx == nil {
		ctx = context.Background()
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	key := frameKey(frame)
	state := i.state[key]
	if state == nil {
		state = &streamState{}
		i.state[key] = state
	}

	state.buffer = append(state.buffer, frame)
	droppedDelta := 0
	if len(state.buffer) > i.config.MaxBufferedFrames {
		state.buffer = state.buffer[len(state.buffer)-i.config.MaxBufferedFrames:]
		state.dropped++
		droppedDelta = 1
	}

	decision := i.vad.Detect(ctx, frame)
	speechDetected := decision.SpeechDetected
	events := make([]Event, 0, 1)
	if speechDetected {
		state.silenceFrames = 0
		if !state.speechActive {
			state.speechActive = true
			events = append(events, EventVADSpeechStart)
		}
	} else if state.speechActive {
		state.silenceFrames++
		if state.silenceFrames >= i.config.SilenceHangover {
			state.speechActive = false
			state.silenceFrames = 0
			events = append(events, EventVADSpeechEnd)
		}
	}

	return IngressResult{
		BufferedFrames:    len(state.buffer),
		DroppedFrames:     state.dropped,
		DroppedFrameDelta: droppedDelta,
		RMS:               decision.Score,
		VADDetector:       decision.Detector,
		VADStatus:         decision.Status,
		VADFinding:        decision.Finding,
		SpeechDetected:    speechDetected,
		SpeechActive:      state.speechActive,
		Events:            events,
	}
}

func NewVADDetector(config VADDetectorConfig) VADDetector {
	if config.Preference == VADDetectorPreferenceSilero {
		return NewSileroVADAdapterWithOptions(SileroVADAdapterOptions{
			Runner:            config.SileroRunner,
			FallbackThreshold: config.FallbackThreshold,
			Timeout:           config.Timeout,
		})
	}
	return NewRMSVADDetector(config.FallbackThreshold)
}

func NewRMSVADDetector(threshold float64) RMSVADDetector {
	if threshold <= 0 {
		threshold = DefaultIngressConfig().SpeechThreshold
	}
	return RMSVADDetector{threshold: threshold}
}

func (d RMSVADDetector) Detect(_ context.Context, frame Frame) VADDecision {
	score := pcm16RMS(frame.DataBase64)
	return VADDecision{
		SpeechDetected: score >= d.threshold,
		Score:          score,
		Detector:       VADDetectorRMS,
		Status:         VADStatusRMSBaseline,
		Finding:        VADFindingRMSBaseline,
	}
}

func NewSileroVADAdapter(runner SileroVADRunner, fallbackThreshold float64) SileroVADAdapter {
	return NewSileroVADAdapterWithOptions(SileroVADAdapterOptions{
		Runner:            runner,
		FallbackThreshold: fallbackThreshold,
	})
}

func NewSileroVADAdapterWithOptions(options SileroVADAdapterOptions) SileroVADAdapter {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultSileroVADTimeout
	}
	return SileroVADAdapter{
		runner:   options.Runner,
		fallback: NewRMSVADDetector(options.FallbackThreshold),
		timeout:  timeout,
	}
}

func (d SileroVADAdapter) Detect(ctx context.Context, frame Frame) VADDecision {
	if d.runner == nil {
		return d.fallbackDecision(frame, VADFindingSileroRunnerEmpty)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx := ctx
	cancel := func() {}
	if d.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, d.timeout)
	}
	defer cancel()

	type runnerResult struct {
		decision VADDecision
		err      error
	}
	resultCh := make(chan runnerResult, 1)
	go func() {
		decision, err := d.runner.Detect(runCtx, frame)
		resultCh <- runnerResult{decision: decision, err: err}
	}()

	var result runnerResult
	select {
	case result = <-resultCh:
	case <-runCtx.Done():
		return d.fallbackDecision(frame, sileroFallbackFinding(runCtx.Err()))
	}
	if result.err != nil {
		return d.fallbackDecision(frame, sileroFallbackFinding(result.err))
	}
	decision := result.decision
	decision.Detector = VADDetectorSilero
	decision.Status = VADStatusSileroAvailable
	decision.Finding = VADFindingSileroAvailable
	return decision
}

func (d SileroVADAdapter) fallbackDecision(frame Frame, finding string) VADDecision {
	decision := d.fallback.Detect(context.Background(), frame)
	decision.Detector = VADDetectorSileroFallbackRMS
	decision.Status = VADStatusSileroUnavailable
	decision.Finding = finding
	return decision
}

func sileroFallbackFinding(err error) string {
	switch {
	case err == nil:
		return VADFindingSileroRunnerError
	case errors.Is(err, context.Canceled):
		return VADFindingSileroRunnerAbort
	case errors.Is(err, context.DeadlineExceeded):
		return VADFindingSileroRunnerTimeout
	default:
		return VADFindingSileroRunnerError
	}
}

func frameKey(frame Frame) string {
	switch {
	case frame.SessionID != "":
		return frame.SessionID
	case frame.TraceID != "":
		return frame.TraceID
	case frame.DeviceID != "":
		return frame.DeviceID
	default:
		return "a21-audio-ingress-default"
	}
}

func pcm16RMS(dataBase64 string) float64 {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil || len(data) < 2 {
		return 0
	}
	samples := len(data) / 2
	var sum float64
	for i := 0; i < samples; i++ {
		raw := uint16(data[i*2]) | uint16(data[i*2+1])<<8
		value := int16(raw)
		normalized := float64(value) / 32768.0
		sum += normalized * normalized
	}
	return math.Sqrt(sum / float64(samples))
}
