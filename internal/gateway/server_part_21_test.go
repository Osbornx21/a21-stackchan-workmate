package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
)

func newBlockingProfessionalASRAdapter() *blockingProfessionalASRAdapter {
	return &blockingProfessionalASRAdapter{
		started:  make(chan struct{}),
		released: make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (a *blockingProfessionalASRAdapter) Name() string {
	return "a21-blocking-professional-asr"
}

func (a *blockingProfessionalASRAdapter) release() {
	a.releaseOnce.Do(func() {
		close(a.released)
	})
}

func (a *blockingProfessionalASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent, 1)
	a.startOnce.Do(func() {
		close(a.started)
	})
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			a.cancelOnce.Do(func() {
				close(a.canceled)
			})
		case <-a.released:
			select {
			case <-ctx.Done():
				a.cancelOnce.Do(func() {
					close(a.canceled)
				})
			case out <- providers.ASRAdapterEvent{Text: "释放后的专业查询", Final: true}:
			}
		}
	}()
	return out, nil
}

type passthroughTextStreamAdapter struct{}

func (passthroughTextStreamAdapter) Name() string {
	return "a21-passthrough-text-stream"
}

func (passthroughTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type blockingSegmentTextStreamAdapter struct {
	firstSegmentSent chan struct{}
	release          chan struct{}
	finalReleased    chan struct{}
	canceled         chan struct{}
	firstOnce        sync.Once
	releaseOnce      sync.Once
	finalOnce        sync.Once
	cancelOnce       sync.Once
}

func newBlockingSegmentTextStreamAdapter() *blockingSegmentTextStreamAdapter {
	return &blockingSegmentTextStreamAdapter{
		firstSegmentSent: make(chan struct{}),
		release:          make(chan struct{}),
		finalReleased:    make(chan struct{}),
		canceled:         make(chan struct{}),
	}
}

func (a *blockingSegmentTextStreamAdapter) Name() string {
	return "a21-blocking-segment-text-stream"
}

func (a *blockingSegmentTextStreamAdapter) releaseFinal() {
	a.releaseOnce.Do(func() {
		close(a.release)
	})
}

func (a *blockingSegmentTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent)
	go func() {
		defer close(out)
		cancel := func() {
			a.cancelOnce.Do(func() {
				close(a.canceled)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第一句。"}:
			a.firstOnce.Do(func() {
				close(a.firstSegmentSent)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-a.release:
			a.finalOnce.Do(func() {
				close(a.finalReleased)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第二句。"}:
		}
		select {
		case <-ctx.Done():
			cancel()
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type segmentChunkTTSAdapter struct{}

func (segmentChunkTTSAdapter) Name() string {
	return "a21-segment-chunk-tts"
}

func (segmentChunkTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	out := make(chan providers.VoiceAudioChunk, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- xiaozhiTestVoiceAudioChunk():
		}
	}()
	return out, nil
}

type gatedSecondChunkTTSAdapter struct {
	release <-chan struct{}
}

func (g gatedSecondChunkTTSAdapter) Name() string {
	return "a21-gated-second-chunk-tts"
}

func (g gatedSecondChunkTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	out := make(chan providers.VoiceAudioChunk)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- xiaozhiTestVoiceAudioChunk():
		}
		if g.release != nil {
			<-g.release
		}
		select {
		case out <- xiaozhiTestVoiceAudioChunk():
		case <-time.After(500 * time.Millisecond):
		}
	}()
	return out, nil
}

type singleSentenceTextStreamAdapter struct{}

func (singleSentenceTextStreamAdapter) Name() string {
	return "a21-single-sentence-text-stream"
}

func (singleSentenceTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 2)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第一句。"}:
		}
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type chunkThenErrorTTSAdapter struct{}

func (chunkThenErrorTTSAdapter) Name() string {
	return "a21-chunk-then-error-tts"
}

func (chunkThenErrorTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	out := make(chan providers.VoiceAudioChunk, 2)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- xiaozhiTestVoiceAudioChunk():
		}
		select {
		case <-ctx.Done():
		case out <- providers.VoiceAudioChunk{Err: errors.New("provider closed after first audio")}:
		}
	}()
	return out, nil
}

type gatewayFallbackTextStreamAdapter struct{}

func (gatewayFallbackTextStreamAdapter) Name() string {
	return "a21-gateway-fallback-text-stream"
}

func (gatewayFallbackTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 3)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{
			Finding: "provider_fallback_used",
			Fallback: &providers.TextStreamFallbackEvent{
				Activated: true,
				Provider:  "a21_voice_fallback",
				Reason:    "primary_failed",
			},
		}:
		}
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "fallback voice answer。"}:
		}
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type recordingXiaozhiPipelineRunner struct {
	requests chan providers.VoicePipelineRequest
	delegate xiaozhiVoicePipelineRunner
}

func newRecordingXiaozhiPipelineRunner() *recordingXiaozhiPipelineRunner {
	return &recordingXiaozhiPipelineRunner{
		requests: make(chan providers.VoicePipelineRequest, 1),
		delegate: providers.NewVoicePipelineRunner(providers.VoicePipelineAdapters{
			ASR:        providers.NewMockASRAdapter("mock-local-asr"),
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		}),
	}
}

func (r *recordingXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	captured := req
	captured.Frames = append([]providers.VoicePipelinePCMFrame(nil), req.Frames...)
	for i := range captured.Frames {
		captured.Frames[i].PCM16LE = append([]byte(nil), req.Frames[i].PCM16LE...)
	}
	select {
	case r.requests <- captured:
	default:
	}
	return r.delegate.Run(ctx, req)
}

type fallbackReportingXiaozhiPipelineRunner struct{}

func (fallbackReportingXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	select {
	case <-ctx.Done():
		return providers.VoicePipelineResult{Status: providers.VoicePipelineStatusCancelled}, ctx.Err()
	default:
	}
	return providers.VoicePipelineResult{
		Status: providers.VoicePipelineStatusCompleted,
		Timing: providers.VoicePipelineTiming{
			ASRFirstPartialMS:       10,
			ASRFinalMS:              20,
			LLMFirstContentMS:       30,
			TTSFirstAudioMS:         40,
			AudioDownlinkFirstMS:    50,
			SpeechEndToFinalASRMS:   20,
			SpeechEndToFirstTokenMS: 30,
		},
		AudioChunks: []providers.VoiceAudioChunk{{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 24000,
			Channels:     1,
			DurationMS:   60,
			DataBase64:   base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11, 0x00}, 1440)),
		}},
		Report: providers.VoicePipelineReport{
			SchemaVersion: "a21.voice_pipeline.fixture.v1",
			Status:        string(providers.VoicePipelineStatusCompleted),
			TraceID:       req.Session.TraceID,
			SessionID:     req.Session.SessionID,
			DeviceID:      req.Session.DeviceID,
			Mode:          req.Mode,
			ExecutionMode: "fixture",
			Selection: providers.VoicePipelineSelection{
				LLMProfile:            "deepseek",
				LLMProfileEnv:         "A21_PROVIDER_PRIMARY",
				LLMFallbackProfile:    "a21_voice_fallback",
				LLMFallbackProfileEnv: "A21_TEXT_STREAM_FALLBACK_PROFILE",
			},
			Output: providers.VoicePipelineOutputReport{
				AudioChunkCount: 1,
				AudioCodec:      string(protocol.AudioCodecPCMS16LE),
				SampleRateHz:    24000,
				Channels:        1,
				ChunkDurationMS: 60,
				LLMContentChars: 21,
			},
			Fallback: &providers.VoicePipelineFallbackReport{
				Activated: true,
				Provider:  "a21_voice_fallback",
				Reason:    "primary_failed",
			},
			Findings: []string{"provider_fallback_used"},
		},
	}, nil
}

type unavailableXiaozhiPipelineRunner struct{}

func (unavailableXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	return unavailableXiaozhiPipelineResult(req), errors.New("a21 provider unavailable")
}

func (unavailableXiaozhiPipelineRunner) RunStream(ctx context.Context, req providers.VoicePipelineRequest) (<-chan providers.VoicePipelineStreamEvent, error) {
	events := make(chan providers.VoicePipelineStreamEvent, 1)
	go func() {
		defer close(events)
		select {
		case <-ctx.Done():
			return
		case events <- providers.VoicePipelineStreamEvent{
			Kind:   providers.VoicePipelineStreamDone,
			Result: unavailableXiaozhiPipelineResult(req),
			Err:    errors.New("a21 provider unavailable"),
		}:
		}
	}()
	return events, nil
}

func unavailableXiaozhiPipelineResult(req providers.VoicePipelineRequest) providers.VoicePipelineResult {
	return providers.VoicePipelineResult{
		Status: providers.VoicePipelineStatusFailed,
		Timing: providers.VoicePipelineTiming{
			ASRFirstPartialMS:       -1,
			ASRFinalMS:              -1,
			LLMFirstContentMS:       -1,
			TTSFirstAudioMS:         -1,
			AudioDownlinkFirstMS:    -1,
			ProviderCancelMS:        -1,
			BargeInStopMS:           -1,
			SpeechEndToFinalASRMS:   -1,
			SpeechEndToFirstTokenMS: -1,
		},
		Report: providers.VoicePipelineReport{
			SchemaVersion: "a21.voice_pipeline.fixture.v1",
			Status:        string(providers.VoicePipelineStatusFailed),
			TraceID:       req.Session.TraceID,
			SessionID:     req.Session.SessionID,
			DeviceID:      req.Session.DeviceID,
			Mode:          req.Mode,
			ExecutionMode: "fixture",
			Selection: providers.VoicePipelineSelection{
				LLMProfile:    "deepseek",
				LLMProfileEnv: "A21_PROVIDER_PRIMARY",
			},
			Findings: []string{"text stream adapter failed"},
		},
	}
}

type failingXiaozhiTTSAdapter struct{}

func (failingXiaozhiTTSAdapter) Name() string {
	return "a21-failing-tts"
}

func (failingXiaozhiTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	return nil, errors.New("a21 tts unavailable")
}

type slowAnswerXiaozhiPipelineRunner struct {
	entered      chan struct{}
	canceled     chan struct{}
	release      chan struct{}
	released     chan struct{}
	enteredOnce  sync.Once
	canceledOnce sync.Once
	releaseOnce  sync.Once
}

func newSlowAnswerXiaozhiPipelineRunner() *slowAnswerXiaozhiPipelineRunner {
	return &slowAnswerXiaozhiPipelineRunner{
		entered:  make(chan struct{}),
		canceled: make(chan struct{}),
		release:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

func (r *slowAnswerXiaozhiPipelineRunner) releaseAnswer() {
	r.releaseOnce.Do(func() {
		close(r.release)
		close(r.released)
	})
}

func (r *slowAnswerXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	r.enteredOnce.Do(func() {
		close(r.entered)
	})
	select {
	case <-ctx.Done():
		r.canceledOnce.Do(func() {
			close(r.canceled)
		})
		return providers.VoicePipelineResult{
			Status:       providers.VoicePipelineStatusCancelled,
			CancelReason: providers.CancelBargeIn,
			Timing: providers.VoicePipelineTiming{
				ASRFirstPartialMS:       -1,
				ASRFinalMS:              -1,
				LLMFirstContentMS:       -1,
				TTSFirstAudioMS:         -1,
				AudioDownlinkFirstMS:    -1,
				ProviderCancelMS:        1,
				BargeInStopMS:           1,
				SpeechEndToFinalASRMS:   -1,
				SpeechEndToFirstTokenMS: -1,
			},
			Report: providers.VoicePipelineReport{
				SchemaVersion: "a21.voice_pipeline.fixture.v1",
				Status:        string(providers.VoicePipelineStatusCancelled),
				TraceID:       req.Session.TraceID,
				SessionID:     req.Session.SessionID,
				DeviceID:      req.Session.DeviceID,
				Mode:          req.Mode,
				ExecutionMode: "fixture",
			},
		}, nil
	case <-r.release:
		return providers.VoicePipelineResult{
			Status: providers.VoicePipelineStatusCompleted,
			Timing: providers.VoicePipelineTiming{
				ASRFirstPartialMS:       10,
				ASRFinalMS:              20,
				LLMFirstContentMS:       30,
				TTSFirstAudioMS:         40,
				AudioDownlinkFirstMS:    40,
				ProviderCancelMS:        -1,
				BargeInStopMS:           -1,
				SpeechEndToFinalASRMS:   20,
				SpeechEndToFirstTokenMS: 30,
			},
			AudioChunks: []providers.VoiceAudioChunk{xiaozhiTestVoiceAudioChunk()},
			Report: providers.VoicePipelineReport{
				SchemaVersion: "a21.voice_pipeline.fixture.v1",
				Status:        string(providers.VoicePipelineStatusCompleted),
				TraceID:       req.Session.TraceID,
				SessionID:     req.Session.SessionID,
				DeviceID:      req.Session.DeviceID,
				Mode:          req.Mode,
				ExecutionMode: "fixture",
				Output: providers.VoicePipelineOutputReport{
					AudioChunkCount: 1,
				},
				Redaction: providers.VoicePipelineRedactionPolicies{
					TranscriptPolicy:     "transcript_not_recorded",
					ProviderOutputPolicy: "provider_output_not_recorded",
					AudioPayloadPolicy:   "audio_payload_not_recorded",
					URLPolicy:            "full_url_not_recorded",
					ProxyPolicy:          "proxy_value_not_recorded",
					LocalPathPolicy:      "local_path_not_recorded",
				},
			},
		}, nil
	}
}

func xiaozhiTestVoiceAudioChunk() providers.VoiceAudioChunk {
	return providers.VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   base64.StdEncoding.EncodeToString(make([]byte, 2880)),
	}
}

func xiaozhiTestOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := codec.EncodePCM16(make([]int16, codec.FrameSamples()))
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestSpeechOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]int16, codec.FrameSamples())
	for i := range pcm {
		if (i/10)%2 == 0 {
			pcm[i] = 12000
		} else {
			pcm[i] = -12000
		}
	}
	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestPCM16Base64(sampleRate int, durationMS int, sample int16) string {
	sampleCount := sampleRate * durationMS / 1000
	data := make([]byte, sampleCount*2)
	for i := 0; i < sampleCount; i++ {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return base64.StdEncoding.EncodeToString(data)
}

func maxAbsPCM16(pcm []int16) int {
	maxAbs := 0
	for _, sample := range pcm {
		abs := int(sample)
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	return maxAbs
}

func readXiaozhiJSON(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if messageType == websocket.MessageBinary {
			continue
		}
		if messageType != websocket.MessageText {
			t.Fatalf("xiaozhi message type=%v, want text JSON", messageType)
		}
		var message map[string]any
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("failed to read JSON message: %v", err)
		}
		return message
	}
}

func readXiaozhiBinary(t *testing.T, ctx context.Context, conn *websocket.Conn) []byte {
	t.Helper()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("xiaozhi binary message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	return data
}

type xiaozhiObservedFrame struct {
	messageType websocket.MessageType
	json        map[string]any
	byteCount   int
}

func readXiaozhiFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) xiaozhiObservedFrame {
	t.Helper()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType == websocket.MessageBinary {
		if len(data) == 0 {
			t.Fatal("xiaozhi binary frame was empty")
		}
		return xiaozhiObservedFrame{messageType: messageType, byteCount: len(data)}
	}
	if messageType != websocket.MessageText {
		t.Fatalf("xiaozhi message type=%v, want text JSON or binary Opus", messageType)
	}
	var message map[string]any
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatalf("failed to parse xiaozhi JSON frame: %v", err)
	}
	return xiaozhiObservedFrame{messageType: messageType, json: message, byteCount: len(data)}
}

func assertXiaozhiProductAnswerSequence(t *testing.T, ctx context.Context, conn *websocket.Conn) {
	t.Helper()
	want := []struct {
		kind  string
		state string
		phase string
	}{
		{kind: "stt"},
		{kind: "tts", state: "start"},
		{kind: "tts", state: "sentence_start", phase: "fast_ack"},
		{kind: "binary"},
		{kind: "tts", state: "sentence_start", phase: "answer"},
		{kind: "binary"},
		{kind: "tts", state: "stop"},
	}
	for i, expected := range want {
		frame := readXiaozhiFrame(t, ctx, conn)
		if expected.kind == "binary" {
			if frame.messageType != websocket.MessageBinary || frame.byteCount == 0 {
				t.Fatalf("frame %d = %s, want non-empty binary Opus", i, xiaozhiFrameSummary(frame))
			}
			continue
		}
		if frame.messageType != websocket.MessageText {
			t.Fatalf("frame %d = %s, want JSON %s", i, xiaozhiFrameSummary(frame), expected.kind)
		}
		gotKind, _ := frame.json["type"].(string)
		if gotKind != expected.kind {
			t.Fatalf("frame %d type = %q, want %q", i, gotKind, expected.kind)
		}
		if expected.state != "" {
			gotState, _ := frame.json["state"].(string)
			if gotState != expected.state {
				t.Fatalf("frame %d state = %q, want %q for %s", i, gotState, expected.state, expected.kind)
			}
		}
		if expected.phase != "" {
			gotPhase, _ := frame.json["phase"].(string)
			if gotPhase != expected.phase {
				t.Fatalf("frame %d phase = %q, want %q for %s/%s", i, gotPhase, expected.phase, expected.kind, expected.state)
			}
		}
		if expected.kind == "stt" {
			text, _ := frame.json["text"].(string)
			if strings.TrimSpace(text) == "" {
				t.Fatalf("frame %d stt text was empty", i)
			}
		}
	}
}

func assertXiaozhiProductAnswerAudioUntilStop(t *testing.T, ctx context.Context, conn *websocket.Conn) {
	t.Helper()
	seenStart := false
	seenAnswerSentence := false
	binaryFrames := 0
	for i := 0; i < 12; i++ {
		frame := readXiaozhiFrame(t, ctx, conn)
		if frame.messageType == websocket.MessageBinary {
			if frame.byteCount == 0 {
				t.Fatalf("frame %d = %s, want non-empty binary Opus", i, xiaozhiFrameSummary(frame))
			}
			binaryFrames++
			continue
		}
		if frame.messageType != websocket.MessageText {
			t.Fatalf("frame %d = %s, want JSON or binary Opus", i, xiaozhiFrameSummary(frame))
		}
		kind, _ := frame.json["type"].(string)
		state, _ := frame.json["state"].(string)
		phase, _ := frame.json["phase"].(string)
		if kind == "tts" && state == "start" {
			seenStart = true
		}
		if kind == "tts" && state == "sentence_start" && phase == "answer" {
			seenAnswerSentence = true
		}
		if kind == "tts" && state == "stop" {
			if !seenStart || !seenAnswerSentence || binaryFrames < 2 {
				t.Fatalf("answer before stop incomplete: seen_start=%v seen_answer=%v binary_frames=%d", seenStart, seenAnswerSentence, binaryFrames)
			}
			return
		}
	}
	t.Fatalf("answer did not reach tts stop: seen_start=%v seen_answer=%v binary_frames=%d", seenStart, seenAnswerSentence, binaryFrames)
}

func xiaozhiFrameSummary(frame xiaozhiObservedFrame) string {
	if frame.messageType == websocket.MessageBinary {
		return fmt.Sprintf("binary bytes=%d", frame.byteCount)
	}
	msgType, _ := frame.json["type"].(string)
	state, _ := frame.json["state"].(string)
	phase, _ := frame.json["phase"].(string)
	return fmt.Sprintf("json type=%q state=%q phase=%q", msgType, state, phase)
}

func xiaozhiProtocol2Wire(payload []byte, timestampMS uint32) []byte {
	wire := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint16(wire[0:2], 2)
	binary.BigEndian.PutUint16(wire[2:4], 0)
	binary.BigEndian.PutUint32(wire[8:12], timestampMS)
	binary.BigEndian.PutUint32(wire[12:16], uint32(len(payload)))
	copy(wire[16:], payload)
	return wire
}

func assertOfficialStackChanState(t *testing.T, ctx context.Context, conn *websocket.Conn, state string) {
	t.Helper()
	seenAvatar := false
	seenMotion := false
	for i := 0; i < 2; i++ {
		messageType, frame, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if messageType != websocket.MessageBinary {
			t.Fatalf("official stackchan message type = %v, want binary", messageType)
		}
		if len(frame) < 5 {
			t.Fatalf("official stackchan frame too short: %#v", frame)
		}
		if got := binary.BigEndian.Uint32(frame[1:5]); got != uint32(len(frame)-5) {
			t.Fatalf("official stackchan frame length = %d, want %d", got, len(frame)-5)
		}
		switch frame[0] {
		case 0x03:
			seenAvatar = true
			var payload map[string]map[string]int
			if err := json.Unmarshal(frame[5:], &payload); err != nil {
				t.Fatal(err)
			}
			wantMouthWeight := 0
			if state == "speaking" {
				wantMouthWeight = 82
			}
			if payload["mouth"]["weight"] != wantMouthWeight {
				t.Fatalf("official avatar payload = %#v, want %s mouth weight %d", payload, state, wantMouthWeight)
			}
		case 0x04:
			seenMotion = true
			var payload map[string]map[string]int
			if err := json.Unmarshal(frame[5:], &payload); err != nil {
				t.Fatal(err)
			}
			wantAngle := map[string]int{
				"listening": 620,
				"thinking":  700,
				"speaking":  580,
			}[state]
			if wantAngle == 0 {
				wantAngle = 450
			}
			if payload["pitchServo"]["angle"] != wantAngle {
				t.Fatalf("official motion payload = %#v, want %s pitch %d", payload, state, wantAngle)
			}
		default:
			t.Fatalf("official stackchan frame type = %#x, want avatar or motion", frame[0])
		}
	}
	if !seenAvatar || !seenMotion {
		t.Fatalf("official stackchan state %s frames missing avatar=%v motion=%v", state, seenAvatar, seenMotion)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func fetchSingleWorkspaceSource(t *testing.T, handler http.Handler, sourceID string) WorkspaceSource {
	t.Helper()
	var response WorkspaceSourcesResponse
	body := fetchWorkspaceSourcesBody(t, handler, sourceID)
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Sources) != 1 {
		t.Fatalf("workspace sources = %+v, want one source", response)
	}
	return response.Sources[0]
}

func fetchWorkspaceSourcesBody(t *testing.T, handler http.Handler, sourceID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/workspace-sources?source_id="+url.QueryEscape(sourceID), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("workspace sources status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func multipartWorkspaceDocumentBody(t *testing.T, fields map[string]string, filename string, content string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, strings.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}
