package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiOpusIngressSkipsCanceledTurnFrame(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-stale-ingress",
		sessionID: "a21-session-xiaozhi-stale-ingress",
	}
	if err := session.configureXiaozhiAudio(xiaozhitransport.AudioParams{
		Format:        "opus",
		SampleRate:    16000,
		Channels:      1,
		FrameDuration: 60,
	}); err != nil {
		t.Fatal(err)
	}
	staleCtx, cancel := context.WithCancel(context.Background())
	cancel()
	packet := xiaozhiTestSpeechOpusPacket(t)

	server.processXiaozhiOpusIngressFrame(staleCtx, nil, session, xiaozhitransport.Frame{
		Kind:      xiaozhitransport.FrameKindOpus,
		Direction: xiaozhitransport.DirectionDeviceToServer,
		DeviceID:  session.deviceID,
		TraceID:   session.traceID,
		SessionID: session.sessionID,
		Opus: &xiaozhitransport.OpusFrame{
			Codec:        "opus",
			Payload:      packet,
			PayloadBytes: len(packet),
		},
	}, 1)

	traces := server.traceEvents("a21-trace-xiaozhi-stale-ingress")
	if !traceContains(traces, "xiaozhi.opus_ingress.stale_frame_suppressed") {
		t.Fatalf("trace missing stale ingress suppression marker: %+v", traces)
	}
	if traceContains(traces, "audio.ingress.buffered") {
		t.Fatalf("stale ingress frame reached audio ingress: %+v", traces)
	}
	if len(session.voicePipelineFrames) != 0 {
		t.Fatalf("voice pipeline frames = %d, want 0 for stale ingress", len(session.voicePipelineFrames))
	}
}

func TestXiaozhiWebSocketUsesStockHandshakeHeaders(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Device-Id":        []string{"stackchan-header-001"},
			"Protocol-Version": []string{"3"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version": 3,
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "hello" || reply["device_id"] != "stackchan-header-001" || reply["version"] != float64(3) {
		t.Fatalf("hello reply = %#v", reply)
	}
	if _, ok := reply["audio_params"].(map[string]any); !ok {
		t.Fatalf("hello reply missing stock audio_params: %#v", reply)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["device_id"] != "stackchan-header-001" {
		t.Fatalf("registry = %#v, want header device id", registry)
	}
}

func TestXiaozhiWebSocketListenDecodesOpusIngressTelemetry(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-raw-opus",
		"session_id": "a21-session-xiaozhi-raw-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	startAck := readXiaozhiJSON(t, ctx, conn)
	if startAck["type"] != "listen" || startAck["state"] != "start" || startAck["status"] != "accepted" {
		t.Fatalf("listen start ack = %#v", startAck)
	}
	packet := xiaozhiTestOpusPacket(t)
	if err := conn.Write(ctx, websocket.MessageBinary, packet); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, packet); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	summary, ok := ttsStart["audio_ingress"].(map[string]any)
	if !ok {
		t.Fatalf("audio summary = %#v", ttsStart["audio_ingress"])
	}
	if summary["codec"] != "opus" || summary["decode_status"] != XiaozhiOpusDecodedPCMState || summary["frame_count"] != float64(2) || summary["byte_count"] != float64(len(packet)*2) {
		t.Fatalf("audio summary = %#v", summary)
	}
	if summary["decoded_frame_count"] != float64(2) || summary["decoded_sample_count"] != float64(1920) || summary["decoded_duration_ms"] != float64(120) {
		t.Fatalf("audio summary = %#v", summary)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" {
		t.Fatalf("sentence start = %#v", sentence)
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-raw-opus")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.opus_frame.received") || !traceContains(traces.Events, "xiaozhi.opus_frame.decoded") || !traceContains(traces.Events, "xiaozhi."+XiaozhiOpusDecodedPCMState) {
		t.Fatalf("trace missing xiaozhi opus markers: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketKeepsBadOpusDecodeHonest(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-bad-opus",
		"session_id": "a21-session-xiaozhi-bad-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, bytes.Repeat([]byte{0x7f}, opuscodec.MaxOpusPacketBytes+1)); err != nil {
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
	if summary["decode_status"] != XiaozhiOpusDecodeErrorState || summary["frame_count"] != float64(1) || summary["decode_error_count"] != float64(1) {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketDecodedOpusFeedsAudioIngress(t *testing.T) {
	server := NewServer()
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
		"trace_id":   "a21-trace-xiaozhi-audio-ingress",
		"session_id": "a21-session-xiaozhi-audio-ingress",
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
		t.Fatalf("downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	readXiaozhiJSON(t, ctx, conn)

	req := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-xiaozhi-audio-ingress", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"sample_rate_hz":16000`,
		`"duration_ms":60`,
		`"data_bytes":1920`,
		`"speech_detected":true`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("recent response missing %q: %s", want, rec.Body.String())
		}
	}
	if strings.Contains(rec.Body.String(), `"data_base64"`) {
		t.Fatalf("default recent response leaked decoded audio: %s", rec.Body.String())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-audio-ingress", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "audio.ingress.buffered") || !traceContains(traces.Events, string(audio.EventVADSpeechStart)) {
		t.Fatalf("trace missing decoded ingress markers: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus(t *testing.T) {
	server := NewServer()
	runner := newRecordingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	profileReq, err := http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL+"/v1/roleplay-profile", bytes.NewBufferString(`{"scenario":"desk_mouthpiece","voice_clone_profile":"a21_voice_clone_default","memory_hints":["真实语音也要短句角色感"]}`))
	if err != nil {
		t.Fatal(err)
	}
	profileReq.Header.Set("Content-Type", "application/json")
	profileResp, err := http.DefaultClient.Do(profileReq)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = profileResp.Body.Close() })
	if profileResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(profileResp.Body)
		t.Fatalf("roleplay profile status = %d: %s", profileResp.StatusCode, string(body))
	}
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	server.xiaozhiFastAckTTS = providers.NewMockTTSAdapter("mock-fast-tts")

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-pipeline",
		"session_id": "a21-session-xiaozhi-pipeline",
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
	if pipeline["schema_version"] != "a21.voice_pipeline.fast_ack.v1" || pipeline["execution_mode"] != "host_local" || pipeline["status"] != "running" || pipeline["stage"] != "fast_ack" {
		t.Fatalf("voice pipeline = %#v", pipeline)
	}
	selection, ok := pipeline["selection"].(map[string]any)
	if !ok || selection["tts_profile"] != "voice_clone_cli" {
		t.Fatalf("voice pipeline selection = %#v, want voice_clone_cli", pipeline["selection"])
	}
	startPayload := mustJSON(t, ttsStart)
	for _, forbidden := range []string{"data_base64", "raw_audio", "provider output", "fixture transcript", "真实语音也要短句角色感", "http://", "https://", "/Users/"} {
		if strings.Contains(startPayload, forbidden) {
			t.Fatalf("tts start leaked %q: %s", forbidden, startPayload)
		}
	}

	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["phase"] != "fast_ack" || sentence["placeholder"] == true {
		t.Fatalf("fast ack sentence start = %#v", sentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("fast ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	sentence = readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["phase"] != "answer" || sentence["placeholder"] == true {
		t.Fatalf("answer sentence start = %#v", sentence)
	}
	answerPipeline, ok := sentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", sentence["voice_pipeline"])
	}
	if answerPipeline["schema_version"] != "a21.voice_pipeline.fixture.v1" || answerPipeline["stage"] != "answer" || answerPipeline["audio_chunk_count"] != float64(1) {
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
	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if len(captured.Frames) != 1 {
		t.Fatalf("captured frames = %d, want 1", len(captured.Frames))
	}
	if len(captured.Frames[0].PCM16LE) != captured.Frames[0].ByteCount || len(captured.Frames[0].PCM16LE) == 0 {
		t.Fatalf("captured frame PCM bytes = %d, byte_count = %d", len(captured.Frames[0].PCM16LE), captured.Frames[0].ByteCount)
	}
	if !strings.Contains(captured.TextPrompt, "Memory Hints") ||
		!strings.Contains(captured.TextPrompt, "session_memory:session_memory_1") ||
		!strings.Contains(captured.TextPrompt, "真实语音也要短句角色感") {
		t.Fatalf("xiaozhi voice pipeline request missing roleplay prompt input")
	}
	if captured.VoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("captured voice clone profile = %q, want selected clone", captured.VoiceCloneProfile)
	}

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-pipeline")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.voice_pipeline.start",
		"asr.first_partial",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"roleplay.prompt_input.used",
		"roleplay.voice_clone_profile.used",
		"xiaozhi.voice_pipeline.completed",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if strings.Contains(mustJSON(t, traces), "真实语音也要短句角色感") {
		t.Fatalf("trace leaked roleplay prompt hint")
	}
}

func TestXiaozhiWebSocketWakePrerollOpusFeedsNextTurn(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-wake-preroll",
		"session_id": "a21-session-xiaozhi-wake-preroll",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}

	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if len(captured.Frames) != 1 {
		t.Fatalf("captured frames = %d, want wake pre-roll frame carried into next turn", len(captured.Frames))
	}
	traces := server.traceEvents("a21-trace-xiaozhi-wake-preroll")
	for _, want := range []string{"xiaozhi.wake_preroll.opus_frame.buffered", "xiaozhi.wake_preroll.attached", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
	if traceContains(traces, "xiaozhi.opus_frame.ignored_not_listening") {
		t.Fatalf("wake pre-roll should not be discarded as not-listening: %+v", traces)
	}
}

func TestXiaozhiWebSocketStreamingASRStartsBeforeListenStop(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-streaming-asr",
		"session_id": "a21-session-xiaozhi-streaming-asr",
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

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-streaming-asr")
		if traceContains(traces, "asr.first_partial") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.stream.start", "asr.audio.append", "asr.first_partial"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace before listen stop missing %q: %+v", want, traces)
		}
	}
	if traceContains(traces, "xiaozhi.listen.stop") {
		t.Fatalf("streaming ASR should start before listen stop: %+v", traces)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" {
		t.Fatalf("post-stop stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-streaming-asr")
		if traceContains(traces, "asr.stream.commit") && traceContains(traces, "asr.final") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.stream.commit", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace after listen stop missing %q: %+v", want, traces)
		}
	}
	partialAt, ok := traceEventAtMS(traces, "asr.first_partial")
	if !ok {
		t.Fatalf("trace missing asr.first_partial: %+v", traces)
	}
	stopAt, ok := traceEventAtMS(traces, "xiaozhi.listen.stop")
	if !ok {
		t.Fatalf("trace missing xiaozhi.listen.stop: %+v", traces)
	}
	if partialAt >= stopAt {
		t.Fatalf("asr.first_partial at %d, want before listen.stop at %d", partialAt, stopAt)
	}
}

func TestXiaozhiWebSocketStreamingASRRecordsSanitizedProviderError(t *testing.T) {
	streamingASR := newErrorEventStreamingASRAdapter("dashscope realtime ASR provider error model_or_profile", errors.New("dashscope realtime ASR provider error: model_or_profile"))
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
		"trace_id":   "a21-trace-xiaozhi-streaming-asr-error",
		"session_id": "a21-session-xiaozhi-streaming-asr-error",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-streaming-asr-error")
		if traceContains(traces, "asr.stream.error.model_or_profile") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.stream.start", "asr.stream.error", "asr.stream.error.model_or_profile"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
	for _, forbidden := range []string{"sk-", "qwen3-asr-secret-model", "InvalidModel"} {
		if traceContains(traces, forbidden) {
			t.Fatalf("trace leaked provider detail %q: %+v", forbidden, traces)
		}
	}
}

func TestXiaozhiWebSocketASRPartialDoesNotSpeakBeforeListenStop(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-partial-bridge",
		"session_id": "a21-session-xiaozhi-partial-bridge",
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

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-partial-bridge")
		if traceContains(traces, "asr.first_partial") && traceContains(traces, "xiaozhi.voice_pipeline.partial_prewarm_deferred") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.first_partial", "xiaozhi.voice_pipeline.partial_prewarm_deferred"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace before listen stop missing %q: %+v", want, traces)
		}
	}
	for _, forbidden := range []string{
		"xiaozhi.listen.stop",
		"vad.speech.end",
		"asr.final",
		"xiaozhi.voice_pipeline.start",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.voice_pipeline.llm.real_streaming",
		"xiaozhi.voice_pipeline.tts.real_streaming",
	} {
		if traceContains(traces, forbidden) {
			t.Fatalf("trace should not contain %q before explicit stop: %+v", forbidden, traces)
		}
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-partial-bridge")
		if traceContains(traces, "asr.final") && traceContains(traces, "xiaozhi.voice_pipeline.completed") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"xiaozhi.listen.stop", "asr.stream.commit", "asr.final", "xiaozhi.voice_pipeline.start", "provider.first_content", "tts.first_audio", "xiaozhi.voice_pipeline.completed"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace after listen stop missing %q: %+v", want, traces)
		}
	}
	pipelineStartAt, ok := traceEventAtMS(traces, "xiaozhi.voice_pipeline.start")
	if !ok {
		t.Fatalf("missing xiaozhi.voice_pipeline.start: %+v", traces)
	}
	stopAt, ok := traceEventAtMS(traces, "xiaozhi.listen.stop")
	if !ok {
		t.Fatalf("missing xiaozhi.listen.stop: %+v", traces)
	}
	providerAt, ok := traceEventAtMS(traces, "provider.first_content")
	if !ok {
		t.Fatalf("missing provider.first_content: %+v", traces)
	}
	finalAt, ok := traceEventAtMS(traces, "asr.final")
	if !ok {
		t.Fatalf("missing asr.final: %+v", traces)
	}
	ttsAt, ok := traceEventAtMS(traces, "tts.first_audio")
	if !ok {
		t.Fatalf("missing tts.first_audio: %+v", traces)
	}
	completedAt, ok := traceEventAtMS(traces, "xiaozhi.voice_pipeline.completed")
	if !ok {
		t.Fatalf("missing xiaozhi.voice_pipeline.completed: %+v", traces)
	}
	if pipelineStartAt < stopAt {
		t.Fatalf("pipeline start at %d, want after listen.stop at %d", pipelineStartAt, stopAt)
	}
	if providerAt < finalAt {
		t.Fatalf("provider.first_content at %d, want after asr.final at %d", providerAt, finalAt)
	}
	if ttsAt >= completedAt {
		t.Fatalf("tts.first_audio at %d, want before pipeline completed at %d", ttsAt, completedAt)
	}
	if got := traceEventCount(traces, "xiaozhi.voice_pipeline.start"); got != 1 {
		t.Fatalf("voice pipeline start count = %d, want exactly one final-driven task: %+v", got, traces)
	}
}

func TestXiaozhiWebSocketStockPhysicalDefersGatewayVADStopUntilListenStop(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-stock-vad-defer",
		"session_id": "a21-session-xiaozhi-stock-vad-defer",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-stock-vad-defer")
		if traceContains(traces, "vad.speech.end") && traceContains(traces, "xiaozhi.listen.gateway_vad_stop_deferred_stock_physical") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"vad.speech.end", "xiaozhi.listen.gateway_vad_stop_deferred_stock_physical", "asr.first_partial"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace before listen stop missing %q: %+v", want, traces)
		}
	}
	for _, forbidden := range []string{"xiaozhi.listen.auto_stop", "xiaozhi.voice_pipeline.start", "provider.first_content", "tts.first_audio", "xiaozhi.tts.opus_frame.downlink"} {
		if traceContains(traces, forbidden) {
			t.Fatalf("stock physical trace should not contain %q before device listen.stop: %+v", forbidden, traces)
		}
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" {
		t.Fatalf("post-stop stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("post-stop tts start = %#v", ttsStart)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-stock-vad-defer")
		if traceContains(traces, "asr.final") && traceContains(traces, "xiaozhi.voice_pipeline.start") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"xiaozhi.listen.stop", "asr.stream.commit", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace after listen stop missing %q: %+v", want, traces)
		}
	}
}
