package audio

import (
	"encoding/base64"
	"testing"
)

func TestIngressBufferCapsDepthAndCountsDrops(t *testing.T) {
	ingress := NewIngress(IngressConfig{
		MaxBufferedFrames: 3,
		SpeechThreshold:   0.02,
		SilenceHangover:   2,
	})

	var result IngressResult
	for seq := uint64(1); seq <= 5; seq++ {
		result = ingress.Push(Frame{
			DeviceID:     "stackchan-sim-001",
			TraceID:      "a21-trace-audio-ingress",
			SessionID:    "a21-session-audio-ingress",
			Seq:          seq,
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   pcm16Base64WithSample(0),
		})
	}

	if result.BufferedFrames != 3 {
		t.Fatalf("buffered frames = %d, want 3", result.BufferedFrames)
	}
	if result.DroppedFrames != 2 {
		t.Fatalf("dropped frames = %d, want 2", result.DroppedFrames)
	}
}

func TestIngressDetectsSpeechStartAndEndWithHangover(t *testing.T) {
	ingress := NewIngress(IngressConfig{
		MaxBufferedFrames: 8,
		SpeechThreshold:   0.02,
		SilenceHangover:   2,
	})

	start := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-vad",
		SessionID:    "a21-session-vad",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if start.RMS <= 0.02 {
		t.Fatalf("rms = %.4f, want above speech threshold", start.RMS)
	}
	if len(start.Events) != 1 || start.Events[0] != EventVADSpeechStart {
		t.Fatalf("start events = %#v, want vad.speech.start", start.Events)
	}

	firstSilence := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-vad",
		SessionID:    "a21-session-vad",
		Seq:          2,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if len(firstSilence.Events) != 0 {
		t.Fatalf("first silence events = %#v, want no hangover end yet", firstSilence.Events)
	}

	secondSilence := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-vad",
		SessionID:    "a21-session-vad",
		Seq:          3,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if len(secondSilence.Events) != 1 || secondSilence.Events[0] != EventVADSpeechEnd {
		t.Fatalf("second silence events = %#v, want vad.speech.end", secondSilence.Events)
	}
}

func TestIngressUsesConfiguredVADDetector(t *testing.T) {
	detector := &scriptedVADDetector{
		decisions: []VADDecision{
			{SpeechDetected: true, Score: 0.73, Detector: "a21-scripted-vad"},
			{SpeechDetected: false, Score: 0.01, Detector: "a21-scripted-vad"},
			{SpeechDetected: false, Score: 0.01, Detector: "a21-scripted-vad"},
		},
	}
	ingress := NewIngress(IngressConfig{
		MaxBufferedFrames: 8,
		SilenceHangover:   2,
		VADDetector:       detector,
	})

	start := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		SessionID:    "a21-session-custom-vad",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})
	if !start.SpeechDetected || start.RMS != 0.73 || start.VADDetector != "a21-scripted-vad" {
		t.Fatalf("start = %+v, want scripted speech decision", start)
	}
	if len(start.Events) != 1 || start.Events[0] != EventVADSpeechStart {
		t.Fatalf("start events = %#v, want vad.speech.start", start.Events)
	}

	firstSilence := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		SessionID:    "a21-session-custom-vad",
		Seq:          2,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if firstSilence.SpeechDetected {
		t.Fatalf("first silence = %+v, want scripted non-speech despite loud input", firstSilence)
	}
	if len(firstSilence.Events) != 0 {
		t.Fatalf("first silence events = %#v, want hangover", firstSilence.Events)
	}

	secondSilence := ingress.Push(Frame{
		DeviceID:     "stackchan-sim-001",
		SessionID:    "a21-session-custom-vad",
		Seq:          3,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if len(secondSilence.Events) != 1 || secondSilence.Events[0] != EventVADSpeechEnd {
		t.Fatalf("second silence events = %#v, want vad.speech.end", secondSilence.Events)
	}
	if detector.calls != 3 {
		t.Fatalf("detector calls = %d, want 3", detector.calls)
	}
}

func pcm16Base64WithSample(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}

type scriptedVADDetector struct {
	decisions []VADDecision
	calls     int
}

func (d *scriptedVADDetector) Detect(frame Frame) VADDecision {
	d.calls++
	if len(d.decisions) == 0 {
		return VADDecision{Detector: "a21-scripted-vad"}
	}
	decision := d.decisions[0]
	d.decisions = d.decisions[1:]
	return decision
}
