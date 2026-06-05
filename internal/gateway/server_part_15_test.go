package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiWebSocketVADSpeechEndAutoStopsRealtimeTurn(t *testing.T) {
	server := NewServer()
	runner := newRecordingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
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
		"trace_id":   "a21-trace-xiaozhi-auto-stop",
		"session_id": "a21-session-xiaozhi-auto-stop",
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
	for i := 0; i < 8; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("registry capabilities = %#v", registry["capabilities"])
	}
	if capabilities["microphone"] != "available_xiaozhi_opus_ingress" || capabilities["speaker"] != "available_xiaozhi_opus_downlink" {
		t.Fatalf("registry capabilities = %#v", capabilities)
	}
	if registry["last_event"] != "xiaozhi.tts.opus_frame.downlink" || registry["connection_status"] != "online" {
		t.Fatalf("registry activity = %#v", registry)
	}

	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if len(captured.Frames) < 3 {
		t.Fatalf("captured frames = %d, want speech plus silence frames", len(captured.Frames))
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-auto-stop")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"vad.speech.end", "xiaozhi.listen.auto_stop", "xiaozhi.voice_pipeline.start", "xiaozhi.opus_frame.ignored_not_listening"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketMaxListenDurationAutoStopsAfterSpeech(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiListenMaxDuration: 180 * time.Millisecond,
	})
	runner := newRecordingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	var nowMS int64 = 1000
	server.now = func() time.Time {
		return time.UnixMilli(atomic.AddInt64(&nowMS, 60))
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
		"trace_id":   "a21-trace-xiaozhi-max-listen",
		"session_id": "a21-session-xiaozhi-max-listen",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	for i := 0; i < 4; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if len(captured.Frames) == 0 {
		t.Fatal("captured frames empty")
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-max-listen")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"vad.speech.start", "xiaozhi.listen.max_duration_auto_stop", "xiaozhi.listen.auto_stop", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketProfessionalModeSendsCheckingBeforeDelayedResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(200 * time.Millisecond)
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "帮我查 V21 座舱反馈证据"},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
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
		"trace_id":   "a21-trace-xiaozhi-pro-delayed",
		"session_id": "a21-session-xiaozhi-pro-delayed",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ackCtx, ackCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer ackCancel()
	ttsStart := readXiaozhiJSON(t, ackCtx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" || ttsStart["mode"] != "professional" {
		t.Fatalf("professional tts start = %#v", ttsStart)
	}
	checking := readXiaozhiJSON(t, ackCtx, conn)
	if checking["type"] != "tts" || checking["state"] != "sentence_start" || checking["phase"] != "professional_checking" || checking["mode"] != "professional" {
		t.Fatalf("checking feedback = %#v", checking)
	}
	if !strings.Contains(asString(checking["text"]), "我在查") {
		t.Fatalf("checking text = %#v, want 我在查", checking["text"])
	}
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start after checking feedback")
	}
	if got := v21.lastUtterance(); got != "帮我查 V21 座舱反馈证据" {
		t.Fatalf("v21 utterance = %q, want ASR-derived utterance", got)
	}

	result := readXiaozhiJSON(t, ctx, conn)
	if result["type"] != "tts" || result["state"] != "sentence_start" || result["phase"] != "professional_result" || result["mode"] != "professional" {
		t.Fatalf("professional result = %#v", result)
	}
	resultJSON := mustJSON(t, result)
	for _, forbidden := range []string{"帮我查 V21 座舱反馈证据", "xiaozhi professional voice turn"} {
		if strings.Contains(resultJSON, forbidden) {
			t.Fatalf("professional result leaked forbidden utterance %q: %s", forbidden, resultJSON)
		}
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "professional_result_completed" {
		t.Fatalf("professional stop = %#v", stop)
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-pro-delayed")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	checkingAt, ok := traceEventAtMS(traces.Events, "professional.checking_feedback.sent")
	if !ok {
		t.Fatalf("trace missing professional.checking_feedback.sent: %+v", traces.Events)
	}
	v21StartAt, ok := traceEventAtMS(traces.Events, "v21.query.start")
	if !ok {
		t.Fatalf("trace missing v21.query.start: %+v", traces.Events)
	}
	if checkingAt > v21StartAt || v21StartAt-checkingAt > 1200 {
		t.Fatalf("checking/v21 ordering checking=%d v21_start=%d", checkingAt, v21StartAt)
	}
}

func TestXiaozhiWebSocketVoiceTriggerRoutesDefaultListenToProfessional(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	streamingASR := newTriggerPhraseStreamingASRAdapter("给我证据 座舱报警")
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
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
		"trace_id":   "a21-trace-xiaozhi-voice-trigger",
		"session_id": "a21-session-xiaozhi-voice-trigger",
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
	case <-time.After(150 * time.Millisecond):
		t.Fatal("streaming ASR did not receive xiaozhi audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" || ttsStart["mode"] != "professional" {
		t.Fatalf("voice-trigger tts start = %#v", ttsStart)
	}
	checking := readXiaozhiJSON(t, ctx, conn)
	if checking["type"] != "tts" || checking["state"] != "sentence_start" || checking["phase"] != "professional_checking" || checking["mode"] != "professional" {
		t.Fatalf("voice-trigger checking feedback = %#v", checking)
	}
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start after voice trigger")
	}
	if got := v21.lastUtterance(); got != "给我证据 座舱报警" {
		t.Fatalf("v21 utterance = %q, want streaming final utterance", got)
	}

	result := readXiaozhiJSON(t, ctx, conn)
	if result["type"] != "tts" || result["state"] != "sentence_start" || result["phase"] != "professional_result" || result["mode"] != "professional" {
		t.Fatalf("voice-trigger professional result = %#v", result)
	}
	resultJSON := mustJSON(t, result)
	for _, forbidden := range []string{"给我证据 座舱报警", "RAW_SECRET_EVIDENCE_BODY"} {
		if strings.Contains(resultJSON, forbidden) {
			t.Fatalf("voice-trigger professional result leaked %q: %s", forbidden, resultJSON)
		}
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "professional_result_completed" {
		t.Fatalf("voice-trigger professional stop = %#v", stop)
	}
	select {
	case <-streamingASR.transcribed:
		t.Fatal("triggered professional route ran duplicate batch ASR")
	default:
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-voice-trigger")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	traceBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	traceJSON := string(traceBody)
	for _, forbidden := range []string{"给我证据 座舱报警", "RAW_SECRET_EVIDENCE_BODY"} {
		if strings.Contains(traceJSON, forbidden) {
			t.Fatalf("voice-trigger trace leaked %q: %s", forbidden, traceJSON)
		}
	}
	for _, want := range []string{"professional.voice_trigger.detected", "xiaozhi.professional_route.voice_trigger", "professional.checking_feedback.sent", "v21.query.start"} {
		if !strings.Contains(traceJSON, want) {
			t.Fatalf("voice-trigger trace missing %q: %s", want, traceJSON)
		}
	}
}

func TestXiaozhiWebSocketStockProfessionalRouteUsesRealtimeListenMode(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(100 * time.Millisecond)
	server := NewServerWithOptions(ServerOptions{
		V21Client:                v21,
		XiaozhiStockProfessional: true,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "认真查一下座舱报警证据"},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
		},
	})
	server.setVoiceMode(VoiceModeProfessional)
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
		"trace_id":   "a21-trace-xiaozhi-stock-pro-route",
		"session_id": "a21-session-xiaozhi-stock-pro-route",
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
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}

	ackCtx, ackCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer ackCancel()
	readXiaozhiJSON(t, ackCtx, conn)
	checking := readXiaozhiJSON(t, ackCtx, conn)
	if checking["phase"] != "professional_checking" || checking["mode"] != "professional" || !strings.Contains(asString(checking["text"]), "我在查") {
		t.Fatalf("checking feedback = %#v", checking)
	}
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start after stock route checking feedback")
	}
	if got := v21.lastUtterance(); got != "认真查一下座舱报警证据" {
		t.Fatalf("v21 utterance = %q, want ASR-derived utterance", got)
	}
	result := readXiaozhiJSON(t, ctx, conn)
	if result["phase"] != "professional_result" || result["mode"] != "professional" {
		t.Fatalf("professional result = %#v", result)
	}
	resultJSON := mustJSON(t, result)
	for _, forbidden := range []string{"认真查一下座舱报警证据", "RAW_SECRET_EVIDENCE_BODY", "debug_metrics", "device_events"} {
		if strings.Contains(resultJSON, forbidden) {
			t.Fatalf("stock professional result leaked %q: %s", forbidden, resultJSON)
		}
	}
	readXiaozhiJSON(t, ctx, conn)

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	if capabilities["xiaozhi_profile"] != "stock" {
		t.Fatalf("capabilities = %#v, want stock profile", capabilities)
	}
	for _, forbidden := range []string{"xiaozhi_feature_debug_metrics", "xiaozhi_feature_device_events"} {
		if _, ok := capabilities[forbidden]; ok {
			t.Fatalf("stock route leaked debug feature %q: %#v", forbidden, capabilities)
		}
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stock-pro-route")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	traceBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	traceJSON := string(traceBody)
	for _, forbidden := range []string{"认真查一下座舱报警证据", "RAW_SECRET_EVIDENCE_BODY"} {
		if strings.Contains(traceJSON, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceJSON)
		}
	}
	for _, want := range []string{"xiaozhi.professional_route.stock_override", "professional.checking_feedback.sent", "v21.query.start", "v21.query.first_result"} {
		if !strings.Contains(traceJSON, want) {
			t.Fatalf("trace missing %q: %s", want, traceJSON)
		}
	}
}

func TestXiaozhiWebSocketStockProfessionalRouteSendsProfessionalOpusDownlink(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{
		V21Client:                v21,
		XiaozhiStockProfessional: true,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "认真查一下热管理证据"},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
		},
	})
	server.setVoiceMode(VoiceModeProfessional)
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
		"trace_id":   "a21-trace-xiaozhi-stock-pro-downlink",
		"session_id": "a21-session-xiaozhi-stock-pro-downlink",
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
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" || ttsStart["mode"] != "professional" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	checking := readXiaozhiJSON(t, ctx, conn)
	if checking["phase"] != "professional_checking" || checking["mode"] != "professional" || !strings.Contains(asString(checking["text"]), "我在查") {
		t.Fatalf("checking feedback = %#v", checking)
	}
	readXiaozhiBinary(t, ctx, conn)

	result := readXiaozhiJSON(t, ctx, conn)
	if result["phase"] != "professional_result" || result["mode"] != "professional" {
		t.Fatalf("professional result = %#v", result)
	}
	readXiaozhiBinary(t, ctx, conn)
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["reason"] != "professional_result_completed" {
		t.Fatalf("professional stop = %#v", stop)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	if capabilities["speaker"] != "available_xiaozhi_opus_downlink" {
		t.Fatalf("capabilities = %#v, want xiaozhi speaker downlink", capabilities)
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stock-pro-downlink")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tts.first_audio", "audio.downlink.first_frame", "xiaozhi.tts.opus_frame.downlink"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketStockProfessionalRouteFallbackDoesNotQueryV21OnEmptyASR(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{
		V21Client:                v21,
		XiaozhiStockProfessional: true,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "   "},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
		},
	})
	server.setVoiceMode(VoiceModeProfessional)
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
		"trace_id":   "a21-trace-xiaozhi-stock-pro-empty",
		"session_id": "a21-session-xiaozhi-stock-pro-empty",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "auto"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "auto"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	checking := readXiaozhiJSON(t, ctx, conn)
	if checking["phase"] != "professional_checking" {
		t.Fatalf("checking feedback = %#v", checking)
	}
	fallback := readXiaozhiJSON(t, ctx, conn)
	if fallback["phase"] != "professional_unavailable" || !strings.Contains(asString(fallback["text"]), "V21 现在没接上") {
		t.Fatalf("fallback = %#v", fallback)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["reason"] != "professional_asr_empty" {
		t.Fatalf("stop = %#v", stop)
	}
	select {
	case <-v21.started:
		t.Fatal("V21 query started after stock route empty ASR final text")
	default:
	}
}

func TestXiaozhiStockProfessionalRouteDoesNotApplyToDebugProfile(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{XiaozhiStockProfessional: true})
	if got := server.xiaozhiListenMode("realtime", xiaozhitransport.HelloFeatures{MCP: true, AEC: true}); got != protocol.ModeWorkmate {
		t.Fatalf("default roleplay stock mode = %q, want workmate", got)
	}
	server.setVoiceMode(VoiceModeProfessional)
	if got := server.xiaozhiListenMode("realtime", xiaozhitransport.HelloFeatures{MCP: true, AEC: true}); got != protocol.ModeProfessional {
		t.Fatalf("professional stock mode = %q, want professional", got)
	}
	if got := server.xiaozhiListenMode("realtime", xiaozhitransport.HelloFeatures{DeviceEvents: true}); got != protocol.ModeWorkmate {
		t.Fatalf("debug mode = %q, want workmate", got)
	}
}

func TestXiaozhiWebSocketProfessionalModeDoesNotUsePlaceholderUtterance(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "认真查一下电池续航证据"},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
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
		"trace_id":   "a21-trace-xiaozhi-pro-asr-query",
		"session_id": "a21-session-xiaozhi-pro-asr-query",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
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
	checking := readXiaozhiJSON(t, ctx, conn)
	if checking["phase"] != "professional_checking" {
		t.Fatalf("checking feedback = %#v", checking)
	}
	result := readXiaozhiJSON(t, ctx, conn)
	if result["phase"] != "professional_result" {
		t.Fatalf("professional result = %#v", result)
	}
	if got := v21.lastUtterance(); got != "认真查一下电池续航证据" {
		t.Fatalf("v21 utterance = %q, want ASR-derived utterance", got)
	}
	if got := v21.lastUtterance(); got == "xiaozhi professional voice turn" {
		t.Fatal("v21 query used old placeholder utterance")
	}
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketProfessionalASREmptyFallsBackWithoutV21Query(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        scriptedProfessionalASRAdapter{text: "   "},
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
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
		"trace_id":   "a21-trace-xiaozhi-pro-asr-empty",
		"session_id": "a21-session-xiaozhi-pro-asr-empty",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
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
	fallback := readXiaozhiJSON(t, ctx, conn)
	if fallback["type"] != "tts" || fallback["phase"] != "professional_unavailable" {
		t.Fatalf("professional ASR fallback = %#v", fallback)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "professional_asr_empty" {
		t.Fatalf("professional ASR fallback stop = %#v", stop)
	}
	select {
	case <-v21.started:
		t.Fatal("V21 query started after empty ASR final text")
	default:
	}
}

func TestXiaozhiWebSocketProfessionalAbortDuringSlowASRSuppressesStaleResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	asr := newBlockingProfessionalASRAdapter()
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21,
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        asr,
			TextStream: passthroughTextStreamAdapter{},
			TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
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
		"trace_id":   "a21-trace-xiaozhi-pro-asr-abort",
		"session_id": "a21-session-xiaozhi-pro-asr-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
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
	select {
	case <-asr.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional ASR did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	asr.release()
	select {
	case <-asr.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("professional ASR did not observe abort cancellation")
	}
	select {
	case <-v21.started:
		t.Fatal("V21 query started after ASR-stage abort")
	default:
	}
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-pro-asr-abort")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.turn.cancel", "xiaozhi.professional_result_suppressed"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}
