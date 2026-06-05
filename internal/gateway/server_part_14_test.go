package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiWebSocketStockPhysicalProductOrderAfterListenStop(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: singleSentenceTextStreamAdapter{},
			TTS:        segmentChunkTTSAdapter{},
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-product-order",
		"session_id": "a21-session-xiaozhi-product-order",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive the product speech frame")
	}
	time.Sleep(150 * time.Millisecond)

	traces := server.traceEvents("a21-trace-xiaozhi-product-order")
	for _, forbidden := range []string{
		"xiaozhi.listen.stop",
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.tts.stop",
	} {
		if traceContains(traces, forbidden) {
			t.Fatalf("trace should not contain %q before product listen.stop: %+v", forbidden, traces)
		}
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()
	assertXiaozhiProductAnswerSequence(t, ctx, conn)
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)

	traces = server.traceEvents("a21-trace-xiaozhi-product-order")
	for _, want := range []string{
		"xiaozhi.listen.stop",
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.stt.sent",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.voice_pipeline.completed",
		"xiaozhi.tts.stop",
	} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after product answer: %+v", want, traces)
		}
	}
	stopAt, ok := traceEventAtMS(traces, "xiaozhi.listen.stop")
	if !ok {
		t.Fatalf("missing product listen.stop: %+v", traces)
	}
	for _, name := range []string{
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.stt.sent",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.stop",
	} {
		at, ok := traceEventAtMS(traces, name)
		if !ok {
			t.Fatalf("missing %s: %+v", name, traces)
		}
		if at < stopAt {
			t.Fatalf("%s at %d, want after product listen.stop at %d", name, at, stopAt)
		}
	}
}

func TestXiaozhiWebSocketListenStopDoesNotBlockAbortWhileStreamingASRCommitPending(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-nonblocking-commit",
		"session_id": "a21-session-xiaozhi-nonblocking-commit",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		if traceContains(server.traceEvents("a21-trace-xiaozhi-nonblocking-commit"), "xiaozhi.abort.received") {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("abort was not processed while streaming ASR commit was pending: %+v", server.traceEvents("a21-trace-xiaozhi-nonblocking-commit"))
}

func TestXiaozhiWebSocketOpusAppendDoesNotBlockAbortControlFrame(t *testing.T) {
	streamingASR := newBlockingAppendStreamingASRAdapter()
	defer streamingASR.releaseAppend()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-opus-ingress-queue",
		"session_id": "a21-session-xiaozhi-opus-ingress-queue",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appendEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR append did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if traceContains(server.traceEvents("a21-trace-xiaozhi-opus-ingress-queue"), "xiaozhi.abort.received") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("abort was not processed while Opus append was blocked: %+v", server.traceEvents("a21-trace-xiaozhi-opus-ingress-queue"))
}

func TestXiaozhiWebSocketAbortSuppressesQueuedOldTurnOpusFrames(t *testing.T) {
	streamingASR := newBlockingAppendStreamingASRAdapter()
	defer streamingASR.releaseAppend()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-abort-queued-opus",
		"session_id": "a21-session-xiaozhi-abort-queued-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	for i := 0; i < 3; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-streamingASR.appendEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR append did not block on first Opus frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		traces := server.traceEvents("a21-trace-xiaozhi-abort-queued-opus")
		if traceEventCount(traces, "xiaozhi.opus_ingress.stale_frame_suppressed") >= 2 {
			if got := traceEventCount(traces, "audio.ingress.buffered"); got != 1 {
				t.Fatalf("audio ingress buffered = %d, want only first pre-abort frame; traces=%+v", got, traces)
			}
			if traceContains(traces, "xiaozhi.voice_pipeline.start") || traceContains(traces, "xiaozhi.tts.opus_frame.downlink") {
				t.Fatalf("stale queued frames started pipeline/downlink after abort: %+v", traces)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("queued stale Opus frames were not suppressed after abort: %+v", server.traceEvents("a21-trace-xiaozhi-abort-queued-opus"))
}

func TestXiaozhiWebSocketStreamingASRFinalStartsPipelineWithoutBatchFallback(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-streaming-final-no-batch",
		"session_id": "a21-session-xiaozhi-streaming-final-no-batch",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-streaming-final-no-batch")
		if traceContains(traces, "asr.final") && traceContains(traces, "xiaozhi.voice_pipeline.start") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.stream.commit", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after streaming final: %+v", want, traces)
		}
	}
	select {
	case <-streamingASR.transcribe:
		t.Fatalf("batch Transcribe was called despite streaming ASR final: %+v", traces)
	default:
	}
}

func TestXiaozhiWebSocketLateStreamingASRFinalStillStartsPipeline(t *testing.T) {
	streamingASR := newLateFinalStreamingASRAdapter(250 * time.Millisecond)
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-late-final",
		"session_id": "a21-session-xiaozhi-late-final",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}

	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" || stt["text"] != "late streaming final after commit timeout" {
		t.Fatalf("late final stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("late final tts start = %#v", ttsStart)
	}
	traces := server.traceEvents("a21-trace-xiaozhi-late-final")
	for _, want := range []string{"asr.stream.commit", "asr.stream.final_timeout", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
	if got := traceEventCount(traces, "xiaozhi.voice_pipeline.start"); got != 1 {
		t.Fatalf("voice pipeline start count = %d, want exactly one late-final answer: %+v", got, traces)
	}
}

func TestXiaozhiWebSocketStreamingASRFinalStartsPipelineWhenGatewayVADMisses(t *testing.T) {
	streamingASR := providers.NewMockStreamingASRAdapter("mock-streaming-asr")
	asr, ok := streamingASR.(providers.ASRAdapter)
	if !ok {
		t.Fatal("mock streaming ASR adapter must also satisfy batch ASR fallback")
	}
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        asr,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-vad-miss-final",
		"session_id": "a21-session-xiaozhi-vad-miss-final",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" {
		t.Fatalf("stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	traces := server.traceEvents("a21-trace-xiaozhi-vad-miss-final")
	for _, want := range []string{"asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after streaming final despite VAD miss: %+v", want, traces)
		}
	}
	if traceContains(traces, "vad.speech.start") {
		t.Fatalf("test must prove VAD-miss path, got vad.speech.start: %+v", traces)
	}
}

func TestXiaozhiWebSocketStreamingASRFinalSendsStockSTTBeforeTTS(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-streaming-final-stt",
		"session_id": "a21-session-xiaozhi-streaming-final-stt",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()

	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" || stt["text"] != "streaming final after async commit" {
		t.Fatalf("first post-ASR message = %#v, want stock stt text before tts", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start after stt = %#v", ttsStart)
	}
	tracePayload := mustJSON(t, server.traceEvents("a21-trace-xiaozhi-streaming-final-stt"))
	if strings.Contains(tracePayload, "streaming final after async commit") {
		t.Fatalf("trace leaked ASR transcript text: %s", tracePayload)
	}
}

func TestXiaozhiWebSocketSTTScreenPolicyStatusOnlyRedactsDeviceTranscript(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		CloudVoiceEnv: []string{"A21_XIAOZHI_STT_SCREEN_POLICY=status_only"},
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-stt-status-only",
		"session_id": "a21-session-xiaozhi-stt-status-only",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()

	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" || stt["text"] != xiaozhiSTTStatusOnlyText {
		t.Fatalf("status-only stt = %#v", stt)
	}
	if stt["text"] == "streaming final after async commit" {
		t.Fatalf("status-only stt leaked raw transcript: %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start after redacted stt = %#v", ttsStart)
	}
	traces := server.traceEvents("a21-trace-xiaozhi-stt-status-only")
	for _, want := range []string{"xiaozhi.stt.display.status_only", "xiaozhi.stt.sent", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
	tracePayload := mustJSON(t, traces)
	if strings.Contains(tracePayload, "streaming final after async commit") {
		t.Fatalf("trace leaked ASR transcript text: %s", tracePayload)
	}
}

func TestXiaozhiVoicePipelineRecordsProviderFallbackObservability(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: gatewayFallbackTextStreamAdapter{},
			TTS:        segmentChunkTTSAdapter{},
			Selection: providers.VoicePipelineSelection{
				ASRMode:               "local",
				ASRProfile:            "a21-gateway-final-asr",
				ASRProfileEnv:         "A21_ASR_LOCAL_PROFILE",
				LLMProfile:            "deepseek",
				LLMProfileEnv:         "A21_PROVIDER_PRIMARY",
				LLMFallbackProfile:    "a21_voice_fallback",
				LLMFallbackProfileEnv: "A21_TEXT_STREAM_FALLBACK_PROFILE",
				TTSMode:               "fast",
				TTSProfile:            "a21-segment-chunk-tts",
				TTSProfileEnv:         "A21_TTS_FAST_PROFILE",
			},
			ExecutionMode: "fixture",
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-fallback",
		"session_id": "a21-session-xiaozhi-fallback",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("fast ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	fallback, ok := answerPipeline["fallback"].(map[string]any)
	if !ok || fallback["activated"] != true || fallback["provider"] != "a21_voice_fallback" || fallback["reason"] != "primary_failed" {
		t.Fatalf("fallback summary = %#v", answerPipeline["fallback"])
	}
	answerPayload := mustJSON(t, answerSentence)
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(answerPayload, forbidden) {
			t.Fatalf("xiaozhi fallback summary leaked %q: %s", forbidden, answerPayload)
		}
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	readXiaozhiJSON(t, ctx, conn)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fallback", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fallback.used", "provider.failover", "xiaozhi.voice_pipeline.completed"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_provider_failover_total 1", "a21_fallback_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestXiaozhiVoicePipelineUnavailableEmitsLocalFallbackState(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return unavailableXiaozhiPipelineRunner{}
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-local-fallback",
		"session_id": "a21-session-xiaozhi-local-fallback",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiBinary(t, ctx, conn)
	fallback := readXiaozhiJSON(t, ctx, conn)
	if fallback["type"] != "tts" ||
		fallback["state"] != "sentence_start" ||
		fallback["phase"] != "local_fallback" ||
		fallback["mode"] != "local_fallback" ||
		!strings.Contains(asString(fallback["text"]), "外部大脑连不上") ||
		!strings.Contains(asString(fallback["text"]), "我还在") {
		t.Fatalf("fallback sentence = %#v", fallback)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "local_fallback" {
		t.Fatalf("fallback stop = %#v", stop)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-local-fallback", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{
		"xiaozhi.voice_pipeline.llm.real_streaming",
		"xiaozhi.voice_pipeline.failed.text_stream_adapter_failed",
		"xiaozhi.voice_pipeline.failed.no_llm_first_content",
		"xiaozhi.voice_pipeline.failed.no_tts_first_audio",
		"xiaozhi.voice_pipeline.failed.empty_audio",
		"xiaozhi.voice_pipeline.unavailable",
		"local_fallback.entered",
		"xiaozhi.local_fallback.sent",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_fallback_total 1") {
		t.Fatalf("metrics missing local fallback count:\n%s", metricsRec.Body.String())
	}
	if strings.Contains(metricsRec.Body.String(), "a21_provider_failover_total 1") {
		t.Fatalf("local fallback incremented provider failover count:\n%s", metricsRec.Body.String())
	}

	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(devicesRec, devicesReq)
	for _, want := range []string{`"current_mode":"local_fallback"`, `"current_expression":"local_fallback"`} {
		if !strings.Contains(devicesRec.Body.String(), want) {
			t.Fatalf("devices missing %q: %s", want, devicesRec.Body.String())
		}
	}
}
