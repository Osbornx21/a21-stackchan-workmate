package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/protocol"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDeviceControlExplicitZeroMockAudioChunksConsumesArmWithoutPlayback(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-zero-chunks","session_id":"a21-session-zero-chunks","mock_playback_on_next_audio_frame":true,"mock_audio_chunks":0}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	readControlEvents(t, ctx, conn, 1)

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-zero-chunks", "a21-session-zero-chunks", pcm16Base64WithSample(0))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
	assertNoValidationArmState(t, server, "stackchan-001", "a21-trace-zero-chunks", "a21-session-zero-chunks")
}

func TestDeviceControlArmWithoutSocketDoesNotLeaveValidationState(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-no-socket","session_id":"a21-session-no-socket","audio_probe_only":true,"mock_playback_on_next_audio_frame":true,"mock_audio_chunks":4,"realtime_on_next_speech":true}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
	assertNoValidationArmState(t, server, "stackchan-001", "a21-trace-no-socket", "a21-session-no-socket")
}

func TestDeviceControlWriteFailureRollsBackValidationState(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	requestCtx, requestCancel := context.WithCancel(context.Background())
	requestCancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/control", bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-write-fail","session_id":"a21-session-write-fail","audio_probe_only":true,"mock_playback_on_next_audio_frame":true,"mock_audio_chunks":4,"realtime_on_next_speech":true}`)).WithContext(requestCtx)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502: %s", rec.Code, rec.Body.String())
	}
	assertNoValidationArmState(t, server, "stackchan-001", "a21-trace-write-fail", "a21-session-write-fail")
}

func TestPhysicalStackChanDeviceIDExcludesSimulatorAndBenchDevices(t *testing.T) {
	tests := []struct {
		deviceID string
		want     bool
	}{
		{deviceID: "stackchan-001", want: true},
		{deviceID: "stackchan-sim-001", want: false},
		{deviceID: "stackchan-bench-001", want: false},
	}
	for _, tc := range tests {
		if got := physicalStackChanDeviceID(tc.deviceID); got != tc.want {
			t.Fatalf("physicalStackChanDeviceID(%q) = %v, want %v", tc.deviceID, got, tc.want)
		}
	}
}

func TestDeviceControlEndpointDeliversControlAndAudioToRegisteredDevice(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","text":"beep","trace_id":"a21-trace-device-control","session_id":"a21-session-device-control","stream_id":"a21-device-command-stream-001","mock_audio_chunks":2}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	events := readControlEvents(t, ctx, conn, 1)
	var control protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &control); err != nil {
		t.Fatal(err)
	}
	if control.State != protocol.ExpressionSpeaking || control.StreamID != "a21-device-command-stream-001" {
		t.Fatalf("control = %+v", control)
	}

	for i := 0; i < 2; i++ {
		var playback protocol.Envelope
		if err := wsjson.Read(ctx, conn, &playback); err != nil {
			t.Fatal(err)
		}
		if playback.Kind != protocol.KindAudioPlaybackChunk {
			t.Fatalf("playback %d kind = %q", i, playback.Kind)
		}
		var chunk protocol.AudioPlaybackChunk
		if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
			t.Fatal(err)
		}
		if chunk.StreamID != "a21-device-command-stream-001" || chunk.DataBase64 == "" {
			t.Fatalf("chunk = %+v", chunk)
		}
		decoded, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 640 || bytes.Equal(decoded, make([]byte, len(decoded))) {
			t.Fatalf("expected non-silent 640-byte validation audio, got len=%d", len(decoded))
		}
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if delivered.Status != "delivered" || delivered.DeliveredTransport != "audio_ws" || len(delivered.Events) != 3 {
			t.Fatalf("delivered = %+v", delivered)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlEndpointDeliversDiagnosticSpeakerTone(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","text":"TONE","trace_id":"a21-trace-speaker-tone","session_id":"a21-session-speaker-tone","diagnostic_tone_hz":1000,"diagnostic_tone_duration_ms":3000,"diagnostic_tone_volume":160}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	events := readControlEvents(t, ctx, conn, 1)
	var control protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &control); err != nil {
		t.Fatal(err)
	}
	if control.DiagnosticToneHz != 1000 || control.DiagnosticToneDurationMS != 3000 || control.DiagnosticToneVolume != 160 {
		t.Fatalf("diagnostic tone payload = %+v", control)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if delivered.Status != "delivered" || len(delivered.Events) != 1 {
			t.Fatalf("delivered = %+v", delivered)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlEndpointDeliversProvidedAudioChunks(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	pcm := mockPCM16SquareWaveBase64(16000, 20)
	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","trace_id":"a21-trace-provided-audio","session_id":"a21-session-provided-audio","stream_id":"a21-provided-audio-stream","audio_chunks":[{"stream_id":"a21-provided-audio-stream","codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":20,"data_base64":"` + pcm + `"}]}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	readControlEvents(t, ctx, conn, 1)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}
	var chunk protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
		t.Fatal(err)
	}
	if chunk.StreamID != "a21-provided-audio-stream" || chunk.DataBase64 != pcm {
		t.Fatalf("chunk = %+v", chunk)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d: %s", resp.StatusCode, data)
		}
		var delivered DeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		if len(delivered.Events) != 2 {
			t.Fatalf("events = %d, want control + provided audio", len(delivered.Events))
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDeviceControlIdleClearsPlaybackStreamFromRegistry(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	speakingBody := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"speaking","mode":"workmate","trace_id":"a21-trace-clear-stream","session_id":"a21-session-clear-stream","stream_id":"a21-clear-stream","mock_audio_chunks":1}`)
	speakingResp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", speakingBody)
	if err != nil {
		t.Fatal(err)
	}
	speakingResp.Body.Close()
	readControlEvents(t, ctx, conn, 1)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}

	idleBody := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"idle","mode":"workmate","trace_id":"a21-trace-clear-stream","session_id":"a21-session-clear-stream","stream_id":"a21-clear-stream"}`)
	idleResp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", idleBody)
	if err != nil {
		t.Fatal(err)
	}
	idleResp.Body.Close()
	readControlEvents(t, ctx, conn, 1)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var registry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	devices := registry["devices"].([]any)
	device := devices[0].(map[string]any)
	if device["current_expression"] != string(protocol.ExpressionIdle) {
		t.Fatalf("current_expression = %#v, want idle; registry=%v", device["current_expression"], device)
	}
	if stream, ok := device["playback_stream_id"].(string); ok && stream != "" {
		t.Fatalf("playback_stream_id = %q, want empty after idle", stream)
	}
}

func TestAudioProbeOnlyDeviceControlSuppressesMockPlayback(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","text":"probe","trace_id":"a21-trace-probe-control","session_id":"a21-session-probe","audio_probe_only":true}`)
	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, data)
	}
	readControlEvents(t, ctx, conn, 1)

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-probe-audio", "a21-session-probe", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-probe-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"audio.frame.received", "audio.ingress.buffered", "audio.probe.frame.accepted"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketRegistersRuntimeEchoWithoutAudioPlayback(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       12,
		TraceID:   "a21-trace-audio-runtime-echo",
		SessionID: "a21-session-audio-runtime-echo",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":               "listening",
			"audio_ws_sent_frames": "24",
		},
	})
	assertNoEnvelope(t, conn, 100*time.Millisecond)

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
	if device.Firmware.ID != "a21-stackchan" || device.IdentityStatus != "ok" {
		t.Fatalf("device identity = %+v status=%q, want ok a21-stackchan", device.Firmware, device.IdentityStatus)
	}
	if device.RuntimeEcho["screen"] != "listening" {
		t.Fatalf("runtime echo screen = %q, want listening; all=%#v", device.RuntimeEcho["screen"], device.RuntimeEcho)
	}
}

func TestDeviceControlEndpointRequiresConnectedAudioSocket(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Post(httpServer.URL+"/v1/devices/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
}

func TestAudioWebSocketRecordsIngressAndVADTrace(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	_ = writeAudioFrameWithPayload(t, ctx, conn, 1, "a21-trace-vad-ws", "a21-session-vad-ws", pcm16Base64WithSample(12000))

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-vad-ws", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	body := traceRec.Body.String()
	for _, want := range []string{"audio.ingress.buffered", "vad.speech.start"} {
		if !strings.Contains(body, want) {
			t.Fatalf("trace missing %q: %s", want, body)
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{
		"a21_audio_ingress_frames_total 1",
		"a21_audio_ingress_rms",
		"a21_vad_speech_start_total 1",
		`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
	} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestAudioWebSocketUsesConfiguredSileroVADLabelsWithoutAudioLeak(t *testing.T) {
	runner := &gatewaySileroVADRunner{
		decision: audio.VADDecision{
			SpeechDetected: true,
			Score:          0.93,
			Detector:       "runner-private-detail",
		},
	}
	server := NewServerWithOptions(ServerOptions{
		AudioIngressConfig: audio.IngressConfig{
			VADPreference: audio.VADDetectorPreferenceSilero,
			SileroRunner:  runner,
			VADTimeout:    50 * time.Millisecond,
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	payload := pcm16Base64WithSample(0)
	_ = writeAudioFrameWithPayload(t, ctx, conn, 1, "a21-trace-silero-vad-ws", "a21-session-silero-vad-ws", payload)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{
		`a21_vad_detector_decisions_total{detector="a21-silero-vad",result="speech"} 1`,
		"a21_vad_speech_start_total 1",
	} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
	if strings.Contains(metricsRec.Body.String(), "runner-private-detail") {
		t.Fatalf("metrics leaked runner detail:\n%s", metricsRec.Body.String())
	}

	recentReq := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?session_id=a21-session-silero-vad-ws", nil)
	recentReq.RemoteAddr = "127.0.0.1:45678"
	recentRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(recentRec, recentReq)
	if recentRec.Code != http.StatusOK {
		t.Fatalf("recent status = %d: %s", recentRec.Code, recentRec.Body.String())
	}
	for _, want := range []string{
		`"vad_detector":"a21-silero-vad"`,
		`"vad_status":"silero_available"`,
		`"vad_finding":"silero_runner_available"`,
	} {
		if !strings.Contains(recentRec.Body.String(), want) {
			t.Fatalf("recent response missing %q: %s", want, recentRec.Body.String())
		}
	}
	for _, forbidden := range []string{payload, "runner-private-detail"} {
		if strings.Contains(recentRec.Body.String(), forbidden) {
			t.Fatalf("recent response leaked %q: %s", forbidden, recentRec.Body.String())
		}
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestAudioWebSocketRejectsInvalidPCMFrame(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   "AAAA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-invalid",
		SessionID: "a21-session-audio-invalid",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError || !strings.Contains(payload.Text, "invalid audio frame") {
		t.Fatalf("payload = %+v, want invalid audio frame error", payload)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer readCancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope after invalid audio: %+v", unexpected)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-audio-invalid", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "audio.ingress.invalid") {
		t.Fatalf("trace missing invalid marker: %s", traceRec.Body.String())
	}
	if strings.Contains(traceRec.Body.String(), "audio.playback.chunk.sent") {
		t.Fatalf("invalid audio should not emit playback trace: %s", traceRec.Body.String())
	}
}

func TestAudioWebSocketVADStartDuringPlaybackTriggersBargeIn(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	firstChunk := writeAudioFrameWithPayload(t, ctx, conn, 1, "a21-trace-barge-audio", "a21-session-barge-audio", pcm16Base64WithSample(0))
	if firstChunk.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("first stream = %q, want a21-audio-stream-000001", firstChunk.StreamID)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-barge-audio", "a21-session-barge-audio", pcm16Base64WithSample(12000))

	events := readControlEvents(t, ctx, conn, 2)
	var interrupted protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &interrupted); err != nil {
		t.Fatal(err)
	}
	if interrupted.State != protocol.ExpressionInterrupted || interrupted.StreamID != firstChunk.StreamID {
		t.Fatalf("interrupted payload = %+v, want interrupted stream %q", interrupted, firstChunk.StreamID)
	}
	var listening protocol.ControlEventPayload
	if err := json.Unmarshal(events[1].Payload, &listening); err != nil {
		t.Fatal(err)
	}
	if listening.State != protocol.ExpressionListening {
		t.Fatalf("second state = %q, want listening", listening.State)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer readCancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope after barge-in: %+v", unexpected)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-barge-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"barge_in.detected", "provider.cancel", "playback.stop"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_barge_in_total 1") {
		t.Fatalf("metrics missing barge-in count:\n%s", metricsRec.Body.String())
	}
}

func TestAudioWebSocketKeepsPlaybackStreamStableForTrace(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	first := writeAudioFrame(t, ctx, conn, 11, "a21-trace-audio-stable", "a21-session-audio-stable")
	second := writeAudioFrame(t, ctx, conn, 12, "a21-trace-audio-stable", "a21-session-audio-stable")

	if first.StreamID == "" {
		t.Fatal("first stream id is empty")
	}
	if second.StreamID != first.StreamID {
		t.Fatalf("second stream = %q, want stable stream %q", second.StreamID, first.StreamID)
	}
}

func TestAudioWebSocketForwardsSpeechToRealtimeProviderAndCommitsOnVADEnd(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(12000))
	listening := readControlEvents(t, ctx, conn, 1)
	var listeningPayload protocol.ControlEventPayload
	if err := json.Unmarshal(listening[0].Payload, &listeningPayload); err != nil {
		t.Fatal(err)
	}
	if listeningPayload.State != protocol.ExpressionListening {
		t.Fatalf("first state = %q, want listening", listeningPayload.State)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(0))
	writeAudioFrameEnvelope(t, ctx, conn, 3, "a21-trace-realtime-audio", "a21-session-realtime-audio", pcm16Base64WithSample(0))
	thinking := readControlEvents(t, ctx, conn, 1)
	var thinkingPayload protocol.ControlEventPayload
	if err := json.Unmarshal(thinking[0].Payload, &thinkingPayload); err != nil {
		t.Fatal(err)
	}
	if thinkingPayload.State != protocol.ExpressionThinking {
		t.Fatalf("commit state = %q, want thinking", thinkingPayload.State)
	}
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	if provider.startCalls != 1 {
		t.Fatalf("startCalls = %d, want 1", provider.startCalls)
	}
	if provider.session.SessionID != "a21-session-realtime-audio" || provider.session.DeviceID != "stackchan-sim-001" {
		t.Fatalf("provider session = %+v", provider.session)
	}
	if got := len(provider.sessionHandle.audioChunks); got != 1 {
		t.Fatalf("audio chunks = %d, want speech frame only", got)
	}
	if provider.sessionHandle.audioChunks[0].SampleRateHz != 16000 || provider.sessionHandle.audioChunks[0].DurationMS != 20 {
		t.Fatalf("audio chunk = %+v", provider.sessionHandle.audioChunks[0])
	}
	if provider.sessionHandle.commitCalls != 1 {
		t.Fatalf("commitCalls = %d, want 1", provider.sessionHandle.commitCalls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-realtime-audio", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.realtime_session.start", "provider.audio.append", "provider.audio.commit"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_audio_uplink_frames_total 1", "a21_realtime_audio_commit_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestAudioWebSocketDoesNotStartRealtimeProviderForUnarmedPhysicalStackChan(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-realtime-unarmed", "a21-session-physical-realtime-unarmed", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)

	if provider.startCalls != 0 {
		t.Fatalf("realtime provider startCalls = %d, want 0 for unarmed physical StackChan audio", provider.startCalls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-physical-realtime-unarmed", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	body := traceRec.Body.String()
	for _, want := range []string{"audio.ingress.buffered", "vad.speech.start", "provider.realtime_audio.physical_suppressed"} {
		if !strings.Contains(body, want) {
			t.Fatalf("trace missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "provider.realtime_session.start") {
		t.Fatalf("unarmed physical audio should not start realtime provider: %s", body)
	}
}
