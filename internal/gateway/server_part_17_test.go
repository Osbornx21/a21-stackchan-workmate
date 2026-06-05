package gateway

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiWebSocketAbortAfterFastAckSuppressesAnswerFrames(t *testing.T) {
	server := NewServer()
	runner := newSlowAnswerXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.releaseAnswer)

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
		"trace_id":   "a21-trace-xiaozhi-fast-ack-abort",
		"session_id": "a21-session-xiaozhi-fast-ack-abort",
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
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("voice pipeline did not start")
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("slow answer pipeline did not observe cancellation")
	}
	runner.releaseAnswer()
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketContinuesAnswerWhenFastAckUnavailable(t *testing.T) {
	server := NewServer()
	server.xiaozhiFastAckTTS = failingXiaozhiTTSAdapter{}
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return fallbackReportingXiaozhiPipelineRunner{}
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
		"trace_id":   "a21-trace-xiaozhi-fast-ack-unavailable",
		"session_id": "a21-session-xiaozhi-fast-ack-unavailable",
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

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_answer_completed" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fast-ack-unavailable", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.fast_ack.unavailable", "provider.first_content", "tts.first_audio", "audio.downlink.first_frame", "xiaozhi.voice_pipeline.completed"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestNewServerWithOptionsUsesConfiguredXiaozhiVoicePipelineAdapters(t *testing.T) {
	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("a21-test-asr"),
		TextStream: providers.NewMockTextStreamAdapter("a21-test-text"),
		TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
		Selection: providers.VoicePipelineSelection{
			ASRMode:    "local",
			ASRProfile: "a21-test-asr",
			LLMProfile: "a21-test-text",
			TTSMode:    "fast",
			TTSProfile: "a21-test-tts",
		},
		ExecutionMode: "host_local",
	}
	server := NewServerWithOptions(ServerOptions{XiaozhiVoicePipelineAdapters: &adapters})
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
		"trace_id":   "a21-trace-xiaozhi-configured-pipeline",
		"session_id": "a21-session-xiaozhi-configured-pipeline",
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

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	for _, want := range []string{
		`"execution_mode":"host_local"`,
		`"schema_version":"a21.voice_pipeline.fast_ack.v1"`,
		`"stage":"fast_ack"`,
		`"status":"running"`,
		`"asr_profile":"a21-test-asr"`,
		`"llm_profile":"a21-test-text"`,
		`"tts_profile":"a21-test-tts"`,
	} {
		if !strings.Contains(mustJSON(t, pipeline), want) {
			t.Fatalf("voice pipeline missing %q: %#v", want, pipeline)
		}
	}
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("fast ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
}

func TestXiaozhiWebSocketAbortCancelsBlockedTurnTaskWithinBargeInBudget(t *testing.T) {
	server := NewServer()
	runner := newBlockingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.unblock)

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
		"trace_id":   "a21-trace-xiaozhi-blocked-abort",
		"session_id": "a21-session-xiaozhi-blocked-abort",
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
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("xiaozhi turn task did not enter blocking pipeline")
	}

	abortAt := time.Now()
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	stopAfter := time.Since(abortAt)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	if stopAfter >= 300*time.Millisecond {
		t.Fatalf("abort stop latency = %s, want <300ms", stopAfter)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("blocked xiaozhi pipeline did not observe turn cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-blocked-abort", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"barge_in.detected",
		"provider.cancel.end",
		"playback.stop",
		"xiaozhi.turn.cancel",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traces.Summary.BargeInStopMS == nil || *traces.Summary.BargeInStopMS >= 300 {
		t.Fatalf("barge-in summary = %v, want <300ms", traces.Summary.BargeInStopMS)
	}
	t.Logf("xiaozhi abort stop latency=%s trace_barge_in_stop_ms=%d", stopAfter, *traces.Summary.BargeInStopMS)
}

func TestXiaozhiWebSocketListenStartBargeInStopsActiveTTS(t *testing.T) {
	server := NewServer()
	runner := newBlockingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.unblock)

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
		"trace_id":   "a21-trace-xiaozhi-listen-barge",
		"session_id": "a21-session-xiaozhi-listen-barge",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	firstStart := readXiaozhiJSON(t, ctx, conn)
	firstTurnID, ok := firstStart["turn_id"].(string)
	if !ok || firstTurnID == "" {
		t.Fatalf("first listen ack turn_id = %#v", firstStart["turn_id"])
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	if messageType, data, err := conn.Read(ctx); err != nil {
		t.Fatal(err)
	} else if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("xiaozhi turn task did not enter blocking pipeline")
	}

	bargeAt := time.Now()
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	stopAfter := time.Since(bargeAt)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "barge_in" {
		t.Fatalf("listen/start barge stop = %#v", stop)
	}
	if stop["turn_id"] != firstTurnID {
		t.Fatalf("listen/start barge stop turn_id = %#v, want %q", stop["turn_id"], firstTurnID)
	}
	if stopAfter >= 300*time.Millisecond {
		t.Fatalf("listen/start barge stop latency = %s, want <300ms", stopAfter)
	}
	nextStart := readXiaozhiJSON(t, ctx, conn)
	if nextStart["type"] != "listen" || nextStart["state"] != "start" || nextStart["status"] != "accepted" {
		t.Fatalf("next listen ack = %#v", nextStart)
	}
	nextTurnID, ok := nextStart["turn_id"].(string)
	if !ok || nextTurnID == "" || nextTurnID == firstTurnID {
		t.Fatalf("next turn_id = %#v, first=%q", nextStart["turn_id"], firstTurnID)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("blocked xiaozhi pipeline did not observe listen/start barge-in cancellation")
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-listen-barge", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.listen.barge_in",
		"barge_in.detected",
		"provider.cancel.end",
		"playback.stop",
		"xiaozhi.turn.cancel",
		"downlink_queue_cleared",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traceContains(traces.Events, "xiaozhi.abort.received") {
		t.Fatalf("listen/start barge-in must not be mislabeled as abort: %+v", traces.Events)
	}
	if traces.Summary.BargeInStopMS == nil || *traces.Summary.BargeInStopMS >= 300 {
		t.Fatalf("barge-in summary = %v, want <300ms", traces.Summary.BargeInStopMS)
	}
}

func TestXiaozhiWebSocketAcceptsProtocolVersion3BinaryFrames(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version":    3,
		"trace_id":   "a21-trace-xiaozhi-v3",
		"session_id": "a21-session-xiaozhi-v3",
		"device_id":  "stackchan-001",
	})
	hello := readXiaozhiJSON(t, ctx, conn)
	if hello["version"] != float64(3) {
		t.Fatalf("hello version = %#v, want 3", hello["version"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	payload := xiaozhiTestOpusPacket(t)
	wire := make([]byte, 4+len(payload))
	wire[0] = 0
	binary.BigEndian.PutUint16(wire[2:4], uint16(len(payload)))
	copy(wire[4:], payload)
	if err := conn.Write(ctx, websocket.MessageBinary, wire); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["profile"] != "xiaozhi_binary_v3" || summary["frame_count"] != float64(1) || summary["byte_count"] != float64(len(payload)) || summary["decode_status"] != XiaozhiOpusDecodedPCMState {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketAcceptsProtocolVersion2BinaryFrames(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Device-Id":        []string{"stackchan-v2-001"},
			"Protocol-Version": []string{"2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version":    2,
		"trace_id":   "a21-trace-xiaozhi-v2",
		"session_id": "a21-session-xiaozhi-v2",
	})
	hello := readXiaozhiJSON(t, ctx, conn)
	if hello["version"] != float64(2) || hello["device_id"] != "stackchan-v2-001" {
		t.Fatalf("hello version/device = %#v/%#v, want v2 header device", hello["version"], hello["device_id"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	payload := xiaozhiTestOpusPacket(t)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiProtocol2Wire(payload, 240)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["profile"] != "xiaozhi_binary_v2" || summary["frame_count"] != float64(1) || summary["byte_count"] != float64(len(payload)) || summary["decode_status"] != XiaozhiOpusDecodedPCMState {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketAbortStopsPlaceholderTTSAndPreventsStaleBinary(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-abort",
		"session_id": "a21-session-xiaozhi-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "wake_word_detected"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	if _, ok := stop["turn_id"].(string); !ok {
		t.Fatalf("abort stop turn_id = %#v, want cancelled turn id", stop["turn_id"])
	}
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x04, 0x05}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestXiaozhiWebSocketRejectsLegacyIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id": "x21-device",
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "error" || reply["code"] != "invalid_device_id" {
		t.Fatalf("reply = %#v, want invalid_device_id", reply)
	}
}

func TestControlWebSocketRegistersFirmwareIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       7,
		TraceID:   "a21-trace-device-000007",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "先说，我在",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.DeviceID != "stackchan-001" {
		t.Fatalf("device id = %q", device.DeviceID)
	}
	if device.IdentityStatus != "ok" {
		t.Fatalf("identity status = %q, want ok", device.IdentityStatus)
	}
	if device.Firmware.ID != "a21-stackchan" || device.Firmware.Version != "0.1.0" || device.Firmware.Board != "m5stack-cores3" || device.Firmware.Commit != "082eb938b713" {
		t.Fatalf("firmware identity = %+v", device.Firmware)
	}
	if device.LastEvent != "mock.turn" {
		t.Fatalf("last event = %q, want mock.turn", device.LastEvent)
	}
	if device.LastTraceID != "a21-trace-device-000007" {
		t.Fatalf("last trace = %q", device.LastTraceID)
	}
}

func TestControlWebSocketRegistersStackChanCapabilities(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       8,
		TraceID:   "a21-trace-device-000008",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "能力上报",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		Capabilities: map[string]string{
			"microphone":    "available",
			"speaker":       "available",
			"screen":        "available",
			"screen_touch":  "available",
			"top_touch":     "available",
			"servo_y":       "available",
			"servo_x":       "planned_continuous_rotation_axis",
			"rgb":           "available",
			"camera":        "planned_core_s3_camera",
			"imu":           "planned_9_axis_imu",
			"ambient_light": "planned_ambient_light_sensor",
			"proximity":     "planned_proximity_sensor",
			"battery":       "planned_550mah_battery",
			"nfc":           "planned_nfc",
			"infrared":      "planned_infrared_tx_rx",
		},
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	capabilities := registry.Devices[0].Capabilities
	wantCapabilities := map[string]string{
		"microphone":    "available",
		"speaker":       "available",
		"screen":        "available",
		"screen_touch":  "available",
		"top_touch":     "available",
		"servo_y":       "available",
		"servo_x":       "planned_continuous_rotation_axis",
		"rgb":           "available",
		"camera":        "planned_core_s3_camera",
		"imu":           "planned_9_axis_imu",
		"ambient_light": "planned_ambient_light_sensor",
		"proximity":     "planned_proximity_sensor",
		"battery":       "planned_550mah_battery",
		"nfc":           "planned_nfc",
		"infrared":      "planned_infrared_tx_rx",
	}
	for key, want := range wantCapabilities {
		if capabilities[key] != want {
			t.Fatalf("capability %s = %q, want %q; all=%#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestControlWebSocketRegistersRuntimeEchoWithoutAssistantReply(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       11,
		TraceID:   "a21-trace-runtime-echo",
		SessionID: "a21-session-runtime-echo",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":  "speaking",
			"servo_y": "48deg",
			"rgb":     "#002430",
		},
	})

	ctxShort, cancelShort := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelShort()
	var reply protocol.Envelope
	if err := wsjson.Read(ctxShort, conn, &reply); err == nil {
		t.Fatalf("runtime echo produced unexpected assistant reply: %+v", reply)
	}

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.LastEvent != protocol.DeviceEventRuntimeEcho {
		t.Fatalf("last event = %q, want runtime echo", device.LastEvent)
	}
	for key, want := range map[string]string{
		"screen":  "speaking",
		"servo_y": "48deg",
		"rgb":     "#002430",
	} {
		if device.RuntimeEcho[key] != want {
			t.Fatalf("runtime echo %s = %q, want %q; all=%#v", key, device.RuntimeEcho[key], want, device.RuntimeEcho)
		}
	}
}

func TestControlWebSocketRejectsLegacyRuntimeEchoIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      12,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
		RuntimeEcho: map[string]string{
			"screen": "x21-render",
		},
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}
	if strings.Contains(payload.Text, "x21-render") {
		t.Fatalf("error leaked forbidden echo value: %q", payload.Text)
	}
}
