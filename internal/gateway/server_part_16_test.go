package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiWebSocketProfessionalModeSendsStructuredRedactedResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-structured",
		"session_id": "a21-session-xiaozhi-pro-structured",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	result := readXiaozhiJSON(t, ctx, conn)
	professional, ok := result["professional"].(map[string]any)
	if !ok {
		t.Fatalf("professional report = %#v", result["professional"])
	}
	for key, want := range map[string]float64{
		"evidence_count":     1,
		"screen_card_count":  1,
		"follow_up_count":    1,
		"speech_block_count": 1,
	} {
		if professional[key] != want {
			t.Fatalf("professional[%s] = %#v, want %.0f", key, professional[key], want)
		}
	}
	if result["text"] != "结论：需要按可引用证据复核。" || result["confidence"] != 0.77 {
		t.Fatalf("professional conclusion/confidence = %#v", result)
	}
	encoded := mustJSON(t, result)
	for _, forbidden := range []string{"RAW_SECRET_EVIDENCE_BODY", "RAW_SECRET_QUOTE", "RAW_SECRET_CARD_TEXT", "v21-doc-secret-raw"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("professional result leaked %q: %s", forbidden, encoded)
		}
	}
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketProfessionalAbortSuppressesStaleResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(-1)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-abort",
		"session_id": "a21-session-xiaozhi-pro-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	v21.release()
	select {
	case <-v21.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("professional V21 query did not observe abort cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketProfessionalNewTurnSuppressesStaleResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(-1)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-new-turn",
		"session_id": "a21-session-xiaozhi-pro-new-turn",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "barge_in" {
		t.Fatalf("new turn stop = %#v", stop)
	}
	newTurnAck := readXiaozhiJSON(t, ctx, conn)
	if newTurnAck["type"] != "listen" || newTurnAck["state"] != "start" || newTurnAck["status"] != "accepted" {
		t.Fatalf("new turn ack = %#v", newTurnAck)
	}
	v21.release()
	select {
	case <-v21.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("professional V21 query did not observe new-turn cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketSendsFastAckBeforeVoicePipelineCompletes(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-fast-ack",
		"session_id": "a21-session-xiaozhi-fast-ack",
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

	ackCtx, ackCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer ackCancel()
	ttsStart := readXiaozhiJSON(t, ackCtx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	audioIngress, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio ingress = %#v", ttsStart["audio_ingress"])
	}
	if audioIngress["asr_status"] != "pipeline_running" || audioIngress["tts_status"] != "fast_ack_then_answer" {
		t.Fatalf("audio ingress = %#v, want running fast ack status", audioIngress)
	}
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	if pipeline["status"] != "running" || pipeline["stage"] != "fast_ack" {
		t.Fatalf("voice pipeline = %#v, want fast_ack running", pipeline)
	}
	for _, forbidden := range []string{"data_base64", "raw_audio", "provider output", "fast ack", "http://", "https://", "/Users/"} {
		if strings.Contains(mustJSON(t, ttsStart), forbidden) {
			t.Fatalf("tts start leaked %q: %s", forbidden, mustJSON(t, ttsStart))
		}
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
	case <-runner.released:
		t.Fatal("answer pipeline completed before fast ack was read")
	default:
	}

	runner.releaseAnswer()
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	if answerPipeline["stage"] != "answer" || answerPipeline["audio_chunk_count"] != float64(1) {
		t.Fatalf("answer voice pipeline = %#v", answerPipeline)
	}
	messageType, data, err = conn.Read(ctx)
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
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestXiaozhiWebSocketFastAckDisabledWaitsForAnswer(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		CloudVoiceEnv: []string{"A21_XIAOZHI_FAST_ACK_ENABLED=false"},
	})
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
		"trace_id":   "a21-trace-xiaozhi-fast-ack-disabled",
		"session_id": "a21-session-xiaozhi-fast-ack-disabled",
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

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	audioIngress, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio ingress = %#v", ttsStart["audio_ingress"])
	}
	if audioIngress["asr_status"] != "pipeline_running" || audioIngress["tts_status"] != "answer_only" {
		t.Fatalf("audio ingress = %#v, want running answer-only status", audioIngress)
	}
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	if pipeline["stage"] != "answer_pending" || pipeline["fast_ack_enabled"] != false {
		t.Fatalf("voice pipeline = %#v, want fast ack disabled answer pending", pipeline)
	}
	time.Sleep(80 * time.Millisecond)
	traceReqBeforeAnswer := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fast-ack-disabled", nil)
	traceRecBeforeAnswer := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRecBeforeAnswer, traceReqBeforeAnswer)
	if traceRecBeforeAnswer.Code != http.StatusOK {
		t.Fatalf("trace status before answer = %d: %s", traceRecBeforeAnswer.Code, traceRecBeforeAnswer.Body.String())
	}
	var tracesBeforeAnswer TraceResponse
	if err := json.NewDecoder(traceRecBeforeAnswer.Body).Decode(&tracesBeforeAnswer); err != nil {
		t.Fatal(err)
	}
	if !traceContains(tracesBeforeAnswer.Events, "xiaozhi.fast_ack.disabled") {
		t.Fatalf("trace missing fast ack disabled marker before answer: %+v", tracesBeforeAnswer.Events)
	}
	if traceContains(tracesBeforeAnswer.Events, "xiaozhi.fast_ack.downlink") {
		t.Fatalf("trace must not include fast ack downlink before answer: %+v", tracesBeforeAnswer.Events)
	}

	runner.releaseAnswer()
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

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fast-ack-disabled", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if traceContains(traces.Events, "xiaozhi.fast_ack.downlink") {
		t.Fatalf("trace must not include fast ack downlink when disabled: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketDelayedFastAckWaitsForThreshold(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		CloudVoiceEnv: []string{
			"A21_XIAOZHI_FAST_ACK_ENABLED=true",
			"A21_XIAOZHI_FAST_ACK_DELAY_MS=80",
		},
	})
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
		"trace_id":   "a21-trace-xiaozhi-delayed-fast-ack",
		"session_id": "a21-session-xiaozhi-delayed-fast-ack",
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

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	audioIngress, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio ingress = %#v", ttsStart["audio_ingress"])
	}
	if audioIngress["tts_status"] != "delayed_fast_ack_then_answer" {
		t.Fatalf("audio ingress = %#v, want delayed fast ack status", audioIngress)
	}
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	if pipeline["schema_version"] != "a21.voice_pipeline.delayed_fast_ack.v1" ||
		pipeline["stage"] != "answer_pending" ||
		pipeline["fast_ack_delay_ms"] != float64(80) {
		t.Fatalf("voice pipeline = %#v, want delayed fast ack summary", pipeline)
	}
	time.Sleep(30 * time.Millisecond)
	tracesBeforeDelay := server.traceEvents("a21-trace-xiaozhi-delayed-fast-ack")
	if traceContains(tracesBeforeDelay, "xiaozhi.fast_ack.downlink") {
		t.Fatalf("trace must not include fast ack downlink before threshold: %+v", tracesBeforeDelay)
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

	runner.releaseAnswer()
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	readXiaozhiJSON(t, ctx, conn)

	traces := server.traceEvents("a21-trace-xiaozhi-delayed-fast-ack")
	for _, want := range []string{"xiaozhi.fast_ack.delay_elapsed", "xiaozhi.fast_ack.delayed", "xiaozhi.fast_ack.downlink"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
}

func TestXiaozhiWebSocketDelayedFastAckSkipsWhenAnswerReady(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		CloudVoiceEnv: []string{
			"A21_XIAOZHI_FAST_ACK_ENABLED=true",
			"A21_XIAOZHI_FAST_ACK_DELAY_MS=200",
		},
	})
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
		"trace_id":   "a21-trace-xiaozhi-delayed-fast-ack-skip",
		"session_id": "a21-session-xiaozhi-delayed-fast-ack-skip",
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
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v, want answer without fast ack", answerSentence)
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
	assertNoXiaozhiMessage(t, conn, 120*time.Millisecond)

	var traces []TraceEvent
	for i := 0; i < 10; i++ {
		traces = server.traceEvents("a21-trace-xiaozhi-delayed-fast-ack-skip")
		if traceContains(traces, "xiaozhi.fast_ack.skipped_answer_ready") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !traceContains(traces, "xiaozhi.fast_ack.skipped_answer_ready") {
		t.Fatalf("trace missing delayed fast ack skip marker: %+v", traces)
	}
	if traceContains(traces, "xiaozhi.fast_ack.downlink") {
		t.Fatalf("trace must not include fast ack downlink when answer is ready: %+v", traces)
	}
}

func TestXiaozhiWebSocketStreamsFirstAnswerSegmentBeforeTextStreamDone(t *testing.T) {
	textStream := newBlockingSegmentTextStreamAdapter()
	tts := segmentChunkTTSAdapter{}
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: textStream,
			TTS:        tts,
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	t.Cleanup(textStream.releaseFinal)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-stream-answer",
		"session_id": "a21-session-xiaozhi-stream-answer",
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
	case <-textStream.firstSegmentSent:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("text stream did not emit first segment")
	}

	answerCtx, answerCancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer answerCancel()
	answerSentence := readXiaozhiJSON(t, answerCtx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	if answerPipeline["stage"] != "answer" || answerPipeline["streaming"] != true {
		t.Fatalf("answer voice pipeline = %#v, want streaming answer", answerPipeline)
	}
	messageType, data, err = conn.Read(answerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-textStream.finalReleased:
		t.Fatal("final text segment was released before first answer binary")
	default:
	}

	textStream.releaseFinal()
	_ = readXiaozhiJSON(t, ctx, conn)
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("final answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_answer_completed" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
}

func TestXiaozhiWebSocketStreamingAnswerErrorAfterAudioDoesNotFallbackLoop(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: singleSentenceTextStreamAdapter{},
			TTS:        chunkThenErrorTTSAdapter{},
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
		"trace_id":   "a21-trace-xiaozhi-stream-degraded",
		"session_id": "a21-session-xiaozhi-stream-degraded",
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
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_answer_completed_degraded" {
		t.Fatalf("tts stop after degraded streaming audio = %#v", ttsStop)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-stream-degraded", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.voice_pipeline.completed_degraded_after_audio") {
		t.Fatalf("trace missing degraded-after-audio marker: %+v", traces.Events)
	}
	if traceContains(traces.Events, "xiaozhi.voice_pipeline.unavailable") || traceContains(traces.Events, "xiaozhi.local_fallback.sent") {
		t.Fatalf("streaming audio already played; must not local fallback loop: %+v", traces.Events)
	}
	if !traceContains(traces.Events, "xiaozhi.tts.stop.input_suppression_armed") {
		t.Fatalf("trace missing post-tts input suppression: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketAbortDuringStreamingAnswerSuppressesStaleSegments(t *testing.T) {
	textStream := newBlockingSegmentTextStreamAdapter()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: textStream,
			TTS:        segmentChunkTTSAdapter{},
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	t.Cleanup(textStream.releaseFinal)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-stream-abort",
		"session_id": "a21-session-xiaozhi-stream-abort",
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
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	select {
	case <-textStream.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("streaming text adapter did not observe cancellation")
	}
	textStream.releaseFinal()
	assertNoXiaozhiWebSocketMessage(t, conn, 150*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stream-abort")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.turn.cancel",
		"downlink_queue_cleared",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}
