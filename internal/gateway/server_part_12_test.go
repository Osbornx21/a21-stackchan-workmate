package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestXiaozhiSayDeliversWAVAsStockTTSDownlink(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	wavPath := filepath.Join(t.TempDir(), "a21-relay-candidate.wav")
	if err := audio.WritePCM16MonoWAV(wavPath, 16000, make([]byte, 1920)); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-say-wav",
		"session_id": "a21-session-say-wav",
	})
	readXiaozhiJSON(t, ctx, conn)

	type sayFrames struct {
		start    map[string]any
		sentence map[string]any
		packet   []byte
		stop     map[string]any
	}
	framesCh := make(chan sayFrames, 1)
	readErrCh := make(chan error, 1)
	readMap := func() (map[string]any, error) {
		var message map[string]any
		if err := wsjson.Read(ctx, conn, &message); err != nil {
			return nil, err
		}
		return message, nil
	}
	go func() {
		start, err := readMap()
		if err != nil {
			readErrCh <- err
			return
		}
		sentence, err := readMap()
		if err != nil {
			readErrCh <- err
			return
		}
		messageType, packet, err := conn.Read(ctx)
		if err != nil {
			readErrCh <- err
			return
		}
		if messageType != websocket.MessageBinary || len(packet) == 0 {
			readErrCh <- fmt.Errorf("say wav downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(packet))
			return
		}
		stop, err := readMap()
		if err != nil {
			readErrCh <- err
			return
		}
		framesCh <- sayFrames{start: start, sentence: sentence, packet: packet, stop: stop}
	}()

	payload, err := json.Marshal(map[string]string{
		"device_id":  "44:1b:f6:e2:6a:60",
		"wav_path":   wavPath,
		"trace_id":   "a21-trace-say-wav",
		"session_id": "a21-session-say-wav",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
		httpServer.URL+"/v1/xiaozhi/say",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("say wav status = %d: %s", resp.StatusCode, string(body))
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response["delivered_transport"] != "xiaozhi_ws" || response["audio_source"] != "wav_file" || response["audio_basename"] != "a21-relay-candidate.wav" {
		t.Fatalf("say wav response = %+v", response)
	}
	if response["text_chars"] != float64(0) || response["audio_chunks"] != float64(1) {
		t.Fatalf("say wav counts = %+v", response)
	}
	forbiddenResponse := string(body)
	for _, forbidden := range []string{wavPath, filepath.Dir(wavPath), "data:", "base64", "transcript", "provider", "proxy"} {
		if strings.Contains(forbiddenResponse, forbidden) {
			t.Fatalf("say wav response leaked %q: %s", forbidden, forbiddenResponse)
		}
	}

	var frames sayFrames
	select {
	case err := <-readErrCh:
		t.Fatal(err)
	case frames = <-framesCh:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if frames.start["type"] != "tts" || frames.start["state"] != "start" || frames.start["phase"] != "host_say" {
		t.Fatalf("say wav start = %#v", frames.start)
	}
	if frames.sentence["type"] != "tts" || frames.sentence["state"] != "sentence_start" || frames.sentence["phase"] != "host_say" {
		t.Fatalf("say wav sentence = %#v", frames.sentence)
	}
	if frames.stop["type"] != "tts" || frames.stop["state"] != "stop" || frames.stop["reason"] != "host_say_complete" {
		t.Fatalf("say wav stop = %#v", frames.stop)
	}
	if len(frames.packet) == 0 {
		t.Fatal("say wav binary packet is empty")
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-say-wav", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.say.start", "xiaozhi.tts.opus_frame.downlink", "xiaozhi.say.input_suppression_armed", "xiaozhi.say.delivered"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiSaySuppressesImmediateListenRestartForStockPhysical(t *testing.T) {
	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("mock-local-asr"),
		TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
		TTS:        segmentChunkTTSAdapter{},
		Selection:  providers.VoicePipelineSelectionFromEnv(nil),
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
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-say-suppress",
		"session_id": "a21-session-say-suppress",
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
			httpServer.URL+"/v1/xiaozhi/say",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","text":"短播放后抑制回声。","trace_id":"a21-trace-say-suppress","session_id":"a21-session-say-suppress"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiBinary(t, ctx, conn)
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "host_say_complete" {
		t.Fatalf("say stop = %#v", stop)
	}
	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("say status = %d: %s", resp.StatusCode, string(body))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "listen",
		"state":      "start",
		"trace_id":   "a21-trace-say-suppress",
		"session_id": "a21-session-say-suppress",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 120*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-say-suppress", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.say.input_suppression_armed", "xiaozhi.listen.start.input_suppressed", "xiaozhi.opus_frame.ignored_suppressed_listen"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traceContains(traces.Events, "xiaozhi.wake_preroll.opus_frame.buffered") {
		t.Fatalf("suppressed host-say echo must not enter wake preroll: %+v", traces.Events)
	}
	for _, forbidden := range []string{"xiaozhi.turn.start", "audio.ingress.buffered", "xiaozhi.voice_pipeline.start"} {
		if traceContains(traces.Events, forbidden) {
			t.Fatalf("trace unexpectedly contains %q: %+v", forbidden, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketStockPhysicalAcceptsOfficialAutoListenAfterAnswer(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        reusableStreamingASRAdapter{name: "mock-streaming-asr"},
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
		"trace_id":   "a21-trace-xiaozhi-product-post-answer-drain",
		"session_id": "a21-session-xiaozhi-product-post-answer-drain",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	assertXiaozhiProductAnswerAudioUntilStop(t, ctx, conn)

	beforeDrain := traceEventCount(server.traceEvents("a21-trace-xiaozhi-product-post-answer-drain"), "xiaozhi.voice_pipeline.start")
	beforeDownlink := traceEventCount(server.traceEvents("a21-trace-xiaozhi-product-post-answer-drain"), "xiaozhi.tts.opus_frame.downlink")
	if beforeDrain != 1 || beforeDownlink != 2 {
		t.Fatalf("initial answer trace counts pipeline=%d downlink=%d, want 1/2", beforeDrain, beforeDownlink)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "listen",
		"state":      "start",
		"mode":       "realtime",
		"trace_id":   "a21-trace-xiaozhi-product-post-answer-drain",
		"session_id": "a21-session-xiaozhi-product-post-answer-drain",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "listen",
		"state":      "stop",
		"mode":       "realtime",
		"trace_id":   "a21-trace-xiaozhi-product-post-answer-drain",
		"session_id": "a21-session-xiaozhi-product-post-answer-drain",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	assertXiaozhiProductAnswerAudioUntilStop(t, ctx, conn)

	traces := server.traceEvents("a21-trace-xiaozhi-product-post-answer-drain")
	for _, want := range []string{
		"xiaozhi.listen.start",
		"xiaozhi.opus_frame.received",
		"audio.ingress.buffered",
		"xiaozhi.voice_pipeline.start",
	} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after official auto-listen restart: %+v", want, traces)
		}
	}
	if got := traceEventCount(traces, "xiaozhi.voice_pipeline.start"); got != beforeDrain+1 {
		t.Fatalf("voice pipeline start count = %d, want %d after official auto-listen restart", got, beforeDrain+1)
	}
	if got := traceEventCount(traces, "xiaozhi.tts.opus_frame.downlink"); got != beforeDownlink+2 {
		t.Fatalf("downlink count = %d, want %d after official auto-listen restart", got, beforeDownlink+2)
	}
	for _, forbidden := range []string{
		"xiaozhi.listen.start.input_suppressed",
		"xiaozhi.listen.start.suppressed_post_tts_drain",
		"xiaozhi.opus_frame.ignored_suppressed_listen",
	} {
		if traceContains(traces, forbidden) {
			t.Fatalf("trace unexpectedly contains %q after official auto-listen restart: %+v", forbidden, traces)
		}
	}
}

func TestXiaozhiWebSocketManualAbortCancelsTurnWithoutBargeInMarkers(t *testing.T) {
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
		"device_id":  "stackchan-001",
		"trace_id":   "a21-trace-xiaozhi-manual-abort",
		"session_id": "a21-session-xiaozhi-manual-abort",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "manual"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("manual abort stop = %#v", stop)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-manual-abort", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.abort.received", "xiaozhi.turn.cancel"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	for _, forbidden := range []string{"barge_in.detected", "playback.stop"} {
		if traceContains(traces.Events, forbidden) {
			t.Fatalf("trace unexpectedly contains %q: %+v", forbidden, traces.Events)
		}
	}
	if traces.Summary.BargeInStopMS != nil {
		t.Fatalf("manual abort barge-in summary = %v, want nil", traces.Summary.BargeInStopMS)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	if strings.Contains(metricsRec.Body.String(), "a21_barge_in_total 1") {
		t.Fatalf("manual abort incremented barge-in metric:\n%s", metricsRec.Body.String())
	}
}

func TestXiaozhiWebSocketTouchAbortSuppressesImmediateListenRestart(t *testing.T) {
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
		"device_id":  "stackchan-001",
		"trace_id":   "a21-trace-xiaozhi-touch-abort-cooldown",
		"session_id": "a21-session-xiaozhi-touch-abort-cooldown",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("touch abort stop = %#v", stop)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	suppressed := readXiaozhiJSON(t, ctx, conn)
	if suppressed["type"] != "listen" || suppressed["state"] != "start" || suppressed["status"] != "ignored" {
		t.Fatalf("immediate listen restart = %#v, want ignored", suppressed)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-touch-abort-cooldown", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.listen.start.suppressed_after_barge") {
		t.Fatalf("trace missing listen cooldown suppression: %+v", traces.Events)
	}
	if countTraceEvents(traces.Events, "xiaozhi.turn.start") != 1 {
		t.Fatalf("turn starts = %+v, want only the pre-abort turn", traces.Events)
	}
}

func TestXiaozhiWebSocketNoSpeechPlaceholderSuppressesImmediateListenRestartForStockPhysical(t *testing.T) {
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
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-xiaozhi-no-speech-cooldown",
		"session_id": "a21-session-xiaozhi-no-speech-cooldown",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["placeholder"] != true {
		t.Fatalf("placeholder sentence = %#v", sentence)
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "placeholder_no_asr_tts" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "listen",
		"state":      "start",
		"trace_id":   "a21-trace-xiaozhi-no-speech-cooldown",
		"session_id": "a21-session-xiaozhi-no-speech-cooldown",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "listen",
		"state":      "stop",
		"trace_id":   "a21-trace-xiaozhi-no-speech-cooldown",
		"session_id": "a21-session-xiaozhi-no-speech-cooldown",
		"device_id":  "44:1b:f6:e2:6a:60",
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-no-speech-cooldown", nil)
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
		"xiaozhi.no_speech.input_suppression_armed",
		"xiaozhi.listen.start.input_suppressed",
		"xiaozhi.listen.start.suppressed_after_no_speech",
		"xiaozhi.opus_frame.ignored_suppressed_listen",
		"xiaozhi.listen.stop.suppressed_session_drain_armed",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traceContains(traces.Events, "xiaozhi.wake_preroll.opus_frame.buffered") {
		t.Fatalf("suppressed listen audio must not enter wake preroll: %+v", traces.Events)
	}
	if countTraceEvents(traces.Events, "xiaozhi.turn.start") != 1 {
		t.Fatalf("turn starts = %+v, want only the no-speech turn", traces.Events)
	}
}

func TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-downlink",
		sessionID: "a21-session-xiaozhi-downlink",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test done")
		ok, err := server.writeXiaozhiOpusDownlink(context.Background(), conn, session, turn, providers.VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 24000,
			Channels:     1,
			DurationMS:   60,
			DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 6000),
		})
		if err != nil || !ok {
			t.Errorf("downlink = ok:%v err:%v", ok, err)
		}
	}))
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })
	messageType, packet, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary {
		t.Fatalf("message type = %v, want binary", messageType)
	}
	codec, err := opuscodec.New(24000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := codec.DecodePCM16(packet)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != codec.FrameSamples() {
		t.Fatalf("decoded samples = %d, want %d", len(pcm), codec.FrameSamples())
	}
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("pacer sent frames = %d, want 1", turn.pacer.SentFrames())
	}
	if !traceContains(server.traceEvents("a21-trace-xiaozhi-downlink"), "xiaozhi.tts.opus_frame.downlink") {
		t.Fatalf("trace missing downlink marker: %+v", server.traceEvents("a21-trace-xiaozhi-downlink"))
	}
}

func TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames(t *testing.T) {
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-downlink-codec",
		sessionID: "a21-session-xiaozhi-downlink-codec",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	chunk := xiaozhiTestVoiceAudioChunk()

	first, err := turn.xiaozhiDownlinkCodec(chunk)
	if err != nil {
		t.Fatal(err)
	}
	second, err := turn.xiaozhiDownlinkCodec(chunk)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("downlink encoder was rebuilt for a contiguous same-format turn")
	}

	next, err := turn.xiaozhiDownlinkCodec(providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 48000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(48000, 60, 6000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if next == first {
		t.Fatal("downlink encoder should rebuild when the audio format changes")
	}
}

func TestWriteXiaozhiOpusDownlinkAccepts48KMono60MS(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-downlink-48k",
		sessionID: "a21-session-xiaozhi-downlink-48k",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, a21WebSocketAcceptOptions())
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test done")
		ok, err := server.writeXiaozhiOpusDownlink(context.Background(), conn, session, turn, providers.VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 48000,
			Channels:     1,
			DurationMS:   60,
			DataBase64:   xiaozhiTestPCM16Base64(48000, 60, 6000),
		})
		if err != nil || !ok {
			t.Errorf("downlink 48k = ok:%v err:%v", ok, err)
		}
	}))
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })
	messageType, packet, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary {
		t.Fatalf("message type = %v, want binary", messageType)
	}
	codec, err := opuscodec.New(48000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := codec.DecodePCM16(packet)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 2880 {
		t.Fatalf("decoded samples = %d, want 2880", len(pcm))
	}
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("pacer sent frames = %d, want 1", turn.pacer.SentFrames())
	}
}

func TestXiaozhiDownlinkPCM16AppliesHeadroomToHotTTSFrames(t *testing.T) {
	for _, tt := range []struct {
		name   string
		sample int16
	}{
		{name: "positive full scale", sample: 32767},
		{name: "negative full scale", sample: -32768},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pcm, err := xiaozhiDownlinkPCM16(providers.VoiceAudioChunk{
				Codec:        string(protocol.AudioCodecPCMS16LE),
				SampleRateHz: 24000,
				Channels:     1,
				DurationMS:   60,
				DataBase64:   xiaozhiTestPCM16Base64(24000, 60, tt.sample),
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := maxAbsPCM16(pcm); got > xiaozhiDownlinkPCM16HeadroomPeak {
				t.Fatalf("pcm peak = %d, want <= %d", got, xiaozhiDownlinkPCM16HeadroomPeak)
			}
			for _, sample := range pcm {
				if sample == 32767 || sample == -32768 {
					t.Fatalf("pcm retained clipped full-scale sample %d", sample)
				}
			}
		})
	}
}

func TestXiaozhiDownlinkPCM16BoostsQuietTTSFramesWithinHeadroom(t *testing.T) {
	pcm, err := xiaozhiDownlinkPCM16(providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 6000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := maxAbsPCM16(pcm); got != 18000 {
		t.Fatalf("quiet TTS pcm peak = %d, want bounded 3x boost to 18000", got)
	}
}

func TestXiaozhiDownlinkPCM16DoesNotBoostTinyNoise(t *testing.T) {
	pcm, err := xiaozhiDownlinkPCM16(providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 128),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := maxAbsPCM16(pcm); got != 128 {
		t.Fatalf("tiny noise pcm peak = %d, want unchanged 128", got)
	}
}

func TestXiaozhiDownlinkPCM16Accepts48KProviderFrames(t *testing.T) {
	pcm, err := xiaozhiDownlinkPCM16(providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 48000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(48000, 60, 6000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 2880 {
		t.Fatalf("pcm samples = %d, want 2880", len(pcm))
	}
}

func TestWriteXiaozhiOpusDownlinkSkipsStaleTurn(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-stale-turn",
		sessionID: "a21-session-xiaozhi-stale-turn",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	session.cancelCurrentXiaozhiTurn("abort")

	ok, err := server.writeXiaozhiOpusDownlink(context.Background(), nil, session, turn, providers.VoiceAudioChunk{
		Codec:        string(protocol.AudioCodecPCMS16LE),
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   xiaozhiTestPCM16Base64(24000, 60, 6000),
	})
	if err != nil {
		t.Fatalf("stale turn err = %v", err)
	}
	if ok {
		t.Fatal("stale turn downlink unexpectedly sent")
	}
	if !traceContains(server.traceEvents("a21-trace-xiaozhi-stale-turn"), "xiaozhi.tts.stale_frame_suppressed") {
		t.Fatalf("trace missing stale suppression marker: %+v", server.traceEvents("a21-trace-xiaozhi-stale-turn"))
	}
}
