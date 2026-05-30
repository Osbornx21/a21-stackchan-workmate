package audio

import (
	"encoding/base64"
	"math"
	"sync"
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
	SpeechDetected    bool
	SpeechActive      bool
	Events            []Event
}

type Ingress struct {
	mu     sync.Mutex
	config IngressConfig
	state  map[string]*streamState
}

type streamState struct {
	buffer        []Frame
	dropped       int
	speechActive  bool
	silenceFrames int
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
	return &Ingress{config: config, state: make(map[string]*streamState)}
}

func (i *Ingress) Push(frame Frame) IngressResult {
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

	rms := pcm16RMS(frame.DataBase64)
	speechDetected := rms >= i.config.SpeechThreshold
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
		RMS:               rms,
		SpeechDetected:    speechDetected,
		SpeechActive:      state.speechActive,
		Events:            events,
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
