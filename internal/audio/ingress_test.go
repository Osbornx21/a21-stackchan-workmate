package audio

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestNewVADDetectorPrefersSileroWhenConfigured(t *testing.T) {
	runner := &scriptedSileroVADRunner{
		decision: VADDecision{
			SpeechDetected: true,
			Score:          0.87,
			Detector:       "runner-detail",
		},
	}
	detector := NewVADDetector(VADDetectorConfig{
		Preference:        VADDetectorPreferenceSilero,
		SileroRunner:      runner,
		FallbackThreshold: 0.02,
	})

	decision := detector.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-silero-selection",
		SessionID:    "a21-session-silero-selection",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})

	if !decision.SpeechDetected || decision.Score != 0.87 {
		t.Fatalf("decision = %+v, want Silero runner decision", decision)
	}
	if decision.Detector != VADDetectorSilero || decision.Status != VADStatusSileroAvailable || decision.Finding != VADFindingSileroAvailable {
		t.Fatalf("decision labels = %+v, want stable Silero labels", decision)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestNewVADDetectorKeepsRMSWhenSileroNotConfigured(t *testing.T) {
	runner := &scriptedSileroVADRunner{
		decision: VADDecision{SpeechDetected: true, Score: 0.99, Detector: "runner-detail"},
	}
	detector := NewVADDetector(VADDetectorConfig{
		Preference:        VADDetectorPreferenceRMS,
		SileroRunner:      runner,
		FallbackThreshold: 0.02,
	})

	decision := detector.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-rms-selection",
		SessionID:    "a21-session-rms-selection",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})

	if decision.SpeechDetected || decision.Detector != VADDetectorRMS || decision.Status != VADStatusRMSBaseline {
		t.Fatalf("decision = %+v, want RMS silence decision", decision)
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

func TestSileroVADAdapterUsesRunnerDecisionWithStableLabel(t *testing.T) {
	runner := &scriptedSileroVADRunner{
		decision: VADDecision{
			SpeechDetected: true,
			Score:          0.91,
			Detector:       "runner-model-detail",
		},
	}
	adapter := NewSileroVADAdapter(runner, 0.02)

	result := adapter.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-silero-vad",
		SessionID:    "a21-session-silero-vad",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(0),
	})

	if !result.SpeechDetected || result.Score != 0.91 {
		t.Fatalf("result = %+v, want runner speech decision", result)
	}
	if result.Detector != "a21-silero-vad" {
		t.Fatalf("Detector = %q, want a21-silero-vad", result.Detector)
	}
	if result.Status != VADStatusSileroAvailable || result.Finding != VADFindingSileroAvailable {
		t.Fatalf("status/finding = %q/%q, want stable Silero availability", result.Status, result.Finding)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestSileroVADAdapterFallsBackToRMSWhenRunnerUnavailableWithStableStatus(t *testing.T) {
	runner := &scriptedSileroVADRunner{
		err: errors.New("runner unavailable: /private/a21/model.onnx raw audio should stay private"),
	}
	adapter := NewSileroVADAdapter(runner, 0.02)

	result := adapter.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-silero-vad-fallback",
		SessionID:    "a21-session-silero-vad-fallback",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})

	if !result.SpeechDetected || result.Score <= 0.02 {
		t.Fatalf("result = %+v, want RMS fallback speech decision", result)
	}
	if result.Detector != "a21-silero-vad-fallback-rms" {
		t.Fatalf("Detector = %q, want a21-silero-vad-fallback-rms", result.Detector)
	}
	if result.Status != VADStatusSileroUnavailable || result.Finding != VADFindingSileroRunnerError {
		t.Fatalf("status/finding = %q/%q, want stable unavailable fallback", result.Status, result.Finding)
	}
	for _, field := range []string{result.Detector, result.Status, result.Finding} {
		for _, forbidden := range []string{"runner unavailable", "model", "private", "raw audio"} {
			if strings.Contains(field, forbidden) {
				t.Fatalf("fallback field %q leaked %q", field, forbidden)
			}
		}
	}
}

func TestSileroVADAdapterTimesOutAndCancelsRunner(t *testing.T) {
	runner := &blockingSileroVADRunner{done: make(chan struct{})}
	adapter := NewSileroVADAdapterWithOptions(SileroVADAdapterOptions{
		Runner:            runner,
		FallbackThreshold: 0.02,
		Timeout:           10 * time.Millisecond,
	})

	started := time.Now()
	result := adapter.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-silero-timeout",
		SessionID:    "a21-session-silero-timeout",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	elapsed := time.Since(started)

	if elapsed > 150*time.Millisecond {
		t.Fatalf("Detect elapsed %s, want timeout fallback within 150ms", elapsed)
	}
	if !result.SpeechDetected || result.Detector != VADDetectorSileroFallbackRMS || result.Finding != VADFindingSileroRunnerTimeout {
		t.Fatalf("result = %+v, want timeout RMS fallback speech", result)
	}
	select {
	case <-runner.done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("runner did not observe context cancellation")
	}
}

func TestCommandSileroVADRunnerRedactsUnavailableCommandDetails(t *testing.T) {
	runner := NewCommandSileroVADRunner(CommandSileroVADRunnerConfig{
		CommandPath: "/private/a21/silero-vad-runner",
		ModelPath:   "/private/a21/models/silero.onnx",
	})

	_, err := runner.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-command-silero",
		SessionID:    "a21-session-command-silero",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if err == nil {
		t.Fatal("Detect() error = nil, want unavailable command error")
	}
	for _, forbidden := range []string{"/private", "silero.onnx", "a21/silero-vad-runner", "raw", "AAAA"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("error %q leaked %q", err.Error(), forbidden)
		}
	}
}

func TestCommandSileroVADRunnerUsesLocalCommandDecision(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "a21-silero-vad-runner")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\ncat >/dev/null\nprintf '{\"speech_detected\":true,\"score\":0.82}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := NewCommandSileroVADRunner(CommandSileroVADRunnerConfig{
		CommandPath: commandPath,
		ModelPath:   filepath.Join(dir, "model.onnx"),
	})

	decision, err := runner.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-command-silero-success",
		SessionID:    "a21-session-command-silero-success",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !decision.SpeechDetected || decision.Score != 0.82 || decision.Detector != "" || decision.Status != "" || decision.Finding != "" {
		t.Fatalf("decision = %+v, want raw command score before adapter labeling", decision)
	}
}

func TestCommandSileroVADRunnerRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "a21-silero-vad-runner")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\ncat >/dev/null\nprintf 'not-json /private/model.onnx'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := NewCommandSileroVADRunner(CommandSileroVADRunnerConfig{
		CommandPath: commandPath,
		ModelPath:   filepath.Join(dir, "model.onnx"),
	})

	_, err := runner.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-command-silero-malformed",
		SessionID:    "a21-session-command-silero-malformed",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if !errors.Is(err, ErrSileroVADRunnerUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrSileroVADRunnerUnavailable", err)
	}
	for _, forbidden := range []string{dir, "model.onnx", "not-json"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("error %q leaked %q", err.Error(), forbidden)
		}
	}
}

func TestCommandSileroVADRunnerReportsCommandFailureAsUnavailable(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "a21-silero-vad-runner")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\ncat >/dev/null\nprintf 'local path should stay private' >&2\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := NewCommandSileroVADRunner(CommandSileroVADRunnerConfig{
		CommandPath: commandPath,
		ModelPath:   filepath.Join(dir, "model.onnx"),
	})

	_, err := runner.Detect(context.Background(), Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-command-silero-failure",
		SessionID:    "a21-session-command-silero-failure",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if !errors.Is(err, ErrSileroVADRunnerUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrSileroVADRunnerUnavailable", err)
	}
	for _, forbidden := range []string{dir, "model.onnx", "local path"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("error %q leaked %q", err.Error(), forbidden)
		}
	}
}

func TestCommandSileroVADRunnerReturnsContextCancellation(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "a21-silero-vad-runner")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\ncat >/dev/null\nsleep 5\nprintf '{\"speech_detected\":true,\"score\":0.99}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := NewCommandSileroVADRunner(CommandSileroVADRunnerConfig{
		CommandPath: commandPath,
		ModelPath:   filepath.Join(dir, "model.onnx"),
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runner.Detect(ctx, Frame{
		DeviceID:     "stackchan-sim-001",
		TraceID:      "a21-trace-command-silero-cancel",
		SessionID:    "a21-session-command-silero-cancel",
		Seq:          1,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Detect() error = %v, want context.Canceled", err)
	}
}

func TestCheckedInSileroVADRunnerHelpIsExecutable(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "a21_silero_vad.py")
	python := testPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, python, scriptPath, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("script help failed: %v", err)
	}
	output := string(out)
	for _, want := range []string{"--sample-rate", "--channels", "--duration-ms", "--model"} {
		if !strings.Contains(output, want) {
			t.Fatalf("script help missing %q in %q", want, output)
		}
	}
	for _, forbidden := range []string{"pcm_s16le_base64", "/private", "A21_SILERO_VAD_MODEL"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("script help leaked %q in %q", forbidden, output)
		}
	}
}

func TestCheckedInSileroVADRunnerReportsUnavailableWithFinitePCM(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "a21_silero_vad.py")
	python := testPython(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, scriptPath, "--sample-rate", "16000", "--channels", "1", "--duration-ms", "20", "--model", "/private/a21/model.onnx")
	cmd.Stdin = strings.NewReader(strings.Repeat("\x00", 640))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("script succeeded without optional model/runtime, want unavailable exit")
	}
	output := string(out)
	if !strings.Contains(output, "a21 silero vad unavailable") {
		t.Fatalf("script output = %q, want fixed unavailable message", output)
	}
	for _, forbidden := range []string{"/private", "model.onnx", "pcm_s16le_base64"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("script output leaked %q in %q", forbidden, output)
		}
	}
}

func TestSileroVADAdapterLowCardinalityLabels(t *testing.T) {
	labels := map[string]bool{}
	for _, runner := range []SileroVADRunner{
		&scriptedSileroVADRunner{decision: VADDecision{SpeechDetected: true, Score: 0.88, Detector: "a21-runner-variant-a"}},
		&scriptedSileroVADRunner{decision: VADDecision{SpeechDetected: false, Score: 0.03, Detector: "a21-runner-variant-b"}},
		&scriptedSileroVADRunner{err: errors.New("runner unavailable with model detail")},
		nil,
	} {
		adapter := NewSileroVADAdapter(runner, 0.02)
		decision := adapter.Detect(context.Background(), Frame{
			DeviceID:     "stackchan-sim-001",
			TraceID:      "a21-trace-low-cardinality",
			SessionID:    "a21-session-low-cardinality",
			Seq:          1,
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   pcm16Base64WithSample(0),
		})
		labels[decision.Detector] = true
	}

	if len(labels) != 2 || !labels["a21-silero-vad"] || !labels["a21-silero-vad-fallback-rms"] {
		t.Fatalf("labels = %#v, want only stable silero/fallback labels", labels)
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

func testPython(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"python3", "/usr/bin/python3"} {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = exec.CommandContext(ctx, path, "-c", "pass").Run()
		cancel()
		if err == nil {
			return path
		}
	}
	t.Skip("python3 is unavailable or not responsive")
	return ""
}

type scriptedVADDetector struct {
	decisions []VADDecision
	calls     int
}

func (d *scriptedVADDetector) Detect(_ context.Context, frame Frame) VADDecision {
	d.calls++
	if len(d.decisions) == 0 {
		return VADDecision{Detector: "a21-scripted-vad"}
	}
	decision := d.decisions[0]
	d.decisions = d.decisions[1:]
	return decision
}

type scriptedSileroVADRunner struct {
	decision VADDecision
	err      error
	calls    int
}

func (r *scriptedSileroVADRunner) Detect(_ context.Context, frame Frame) (VADDecision, error) {
	r.calls++
	if r.err != nil {
		return VADDecision{}, r.err
	}
	return r.decision, nil
}

type blockingSileroVADRunner struct {
	done  chan struct{}
	calls int
}

func (r *blockingSileroVADRunner) Detect(ctx context.Context, frame Frame) (VADDecision, error) {
	r.calls++
	<-ctx.Done()
	close(r.done)
	return VADDecision{}, ctx.Err()
}
