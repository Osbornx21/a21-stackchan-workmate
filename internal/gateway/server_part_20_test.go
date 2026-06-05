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
	"sync"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDeviceControlArmsRealtimeProviderForNextPhysicalSpeech(t *testing.T) {
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

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-physical-realtime-armed","session_id":"a21-session-physical-realtime-armed","realtime_on_next_speech":true}`)
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

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-realtime-armed", "a21-session-physical-realtime-armed", pcm16Base64WithSample(12000))
	events := readControlEvents(t, ctx, conn, 1)
	var listening protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &listening); err != nil {
		t.Fatal(err)
	}
	if listening.State != protocol.ExpressionListening {
		t.Fatalf("state = %q, want listening", listening.State)
	}
	if provider.startCalls != 1 {
		t.Fatalf("realtime provider startCalls = %d, want 1 after explicit arm", provider.startCalls)
	}
	if provider.session.DeviceID != "stackchan-001" || provider.session.SessionID != "a21-session-physical-realtime-armed" {
		t.Fatalf("provider session = %+v", provider.session)
	}
	if got := len(provider.sessionHandle.audioChunks); got != 1 {
		t.Fatalf("provider audio chunks = %d, want 1", got)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-physical-realtime-armed", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.realtime_audio.physical_armed", "provider.realtime_session.start", "provider.audio.append"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketClosesRealtimeProviderSessionWhenSocketCloses(t *testing.T) {
	provider := &capturingRealtimeAudioProvider{closed: make(chan struct{}, 1)}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio"), nil)
	if err != nil {
		t.Fatal(err)
	}

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-close", "a21-session-realtime-close", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 1)
	if err := conn.Close(websocket.StatusNormalClosure, "test done"); err != nil {
		t.Fatal(err)
	}

	select {
	case <-provider.closed:
	case <-time.After(time.Second):
		t.Fatal("realtime provider session was not closed after audio WebSocket closed")
	}
}

func TestAudioWebSocketStreamsRealtimeProviderOutputEventsAfterCommit(t *testing.T) {
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

	writeAudioFrameEnvelope(t, ctx, conn, 1, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 1)
	writeAudioFrameEnvelope(t, ctx, conn, 2, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(0))
	writeAudioFrameEnvelope(t, ctx, conn, 3, "a21-trace-realtime-downlink", "a21-session-realtime-downlink", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 1)

	provider.sessionHandle.events <- providers.VoiceEvent{
		Session:  provider.session,
		Kind:     providers.VoiceEventSpeaking,
		Final:    false,
		StreamID: "a21-provider-stream-001",
		Audio: &providers.VoiceAudioChunk{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   "AAAA",
		},
	}

	events := readControlEvents(t, ctx, conn, 1)
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "a21-provider-stream-001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("playback kind = %q, want audio.playback.chunk", playback.Kind)
	}
	var chunk protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &chunk); err != nil {
		t.Fatal(err)
	}
	if chunk.StreamID != "a21-provider-stream-001" || chunk.DataBase64 != "AAAA" {
		t.Fatalf("chunk = %+v", chunk)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-realtime-downlink", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"provider.audio.downlink", "provider.audio.first_downlink", "audio.playback.chunk.sent"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{
		"a21_realtime_audio_downlink_events_total 1",
		"a21_realtime_first_audio_ms_count 1",
	} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func writeDeviceEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, envelope protocol.Envelope, payload protocol.DeviceEventPayload) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Payload = data
	if envelope.Protocol == "" {
		envelope.Protocol = protocol.ProtocolVersion
	}
	if err := wsjson.Write(ctx, conn, envelope); err != nil {
		t.Fatal(err)
	}
}

func writeAudioFrame(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string) protocol.AudioPlaybackChunk {
	t.Helper()
	return writeAudioFrameWithPayload(t, ctx, conn, seq, traceID, sessionID, pcm16Base64WithSample(0))
}

func writeAudioFrameWithPayload(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, payloadBase64 string) protocol.AudioPlaybackChunk {
	t.Helper()
	writeAudioFrameEnvelope(t, ctx, conn, seq, traceID, sessionID, payloadBase64)
	readControlEvents(t, ctx, conn, 2)
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
	return chunk
}

func writeAudioFrameEnvelope(t *testing.T, ctx context.Context, conn *websocket.Conn, seq uint64, traceID string, sessionID string, payloadBase64 string) {
	t.Helper()
	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-sim-001", seq, traceID, sessionID, payloadBase64)
}

func writeAudioFrameEnvelopeForDevice(t *testing.T, ctx context.Context, conn *websocket.Conn, deviceID string, seq uint64, traceID string, sessionID string, payloadBase64 string) {
	t.Helper()
	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   payloadBase64,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  deviceID,
		Kind:      protocol.KindAudioFrame,
		Seq:       seq,
		TraceID:   traceID,
		SessionID: sessionID,
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
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

func readControlEvents(t *testing.T, ctx context.Context, conn *websocket.Conn, count int) []protocol.Envelope {
	t.Helper()
	events := make([]protocol.Envelope, 0, count)
	for i := 0; i < count; i++ {
		var event protocol.Envelope
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			t.Fatal(err)
		}
		if event.Kind != protocol.KindControlEvent {
			t.Fatalf("event %d kind = %q, want control.event", i, event.Kind)
		}
		events = append(events, event)
	}
	return events
}

func fetchSingleDeviceRegistryItem(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(serverURL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("device registry status = %d, want 200", resp.StatusCode)
	}
	var registry struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	devices := registry.Devices
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(devices))
	}
	return devices[0]
}

func traceContains(events []TraceEvent, name string) bool {
	for _, event := range events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func assertXiaozhiMCPMessage(t *testing.T, message map[string]any, toolName string, wantArgs map[string]any) {
	t.Helper()
	if message["type"] != "mcp" {
		t.Fatalf("mcp message type = %#v in %#v", message["type"], message)
	}
	payload, ok := message["payload"].(map[string]any)
	if !ok {
		t.Fatalf("mcp payload = %#v", message["payload"])
	}
	params, ok := payload["params"].(map[string]any)
	if !ok {
		t.Fatalf("mcp params = %#v", payload["params"])
	}
	if params["name"] != toolName {
		t.Fatalf("mcp tool name = %#v, want %q", params["name"], toolName)
	}
	args, ok := params["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("mcp arguments = %#v", params["arguments"])
	}
	if len(args) != len(wantArgs) {
		t.Fatalf("mcp args = %#v, want %#v", args, wantArgs)
	}
	for key, want := range wantArgs {
		if args[key] != want {
			t.Fatalf("mcp args = %#v, want %s=%#v", args, key, want)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countTraceEvents(events []TraceEvent, name string) int {
	count := 0
	for _, event := range events {
		if event.Name == name {
			count++
		}
	}
	return count
}

func assertSummaryDelta(t *testing.T, name string, got *int64, want int64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", name, got, want)
	}
}

func assertNoEnvelope(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var unexpected protocol.Envelope
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected envelope: %+v", unexpected)
	}
}

func assertNoValidationArmState(t *testing.T, server *Server, deviceID string, traceID string, sessionID string) {
	t.Helper()
	key := streamStateKey(traceID, sessionID, deviceID)
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.audioProbeSessions[key] {
		t.Fatalf("audio probe state still armed for %s", key)
	}
	if chunks, ok := server.mockPlaybackArmedSessions[key]; ok {
		t.Fatalf("mock playback state still armed for %s with %d chunk(s)", key, chunks)
	}
	if server.realtimeArmedSessions[key] {
		t.Fatalf("realtime state still armed for %s", key)
	}
}

func assertNoXiaozhiMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var unexpected map[string]any
	if err := wsjson.Read(readCtx, conn, &unexpected); err == nil {
		t.Fatalf("unexpected xiaozhi message: %#v", unexpected)
	}
}

func assertNoXiaozhiWebSocketMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	messageType, data, err := conn.Read(readCtx)
	if err == nil {
		t.Fatalf("unexpected xiaozhi websocket message type=%v bytes=%d", messageType, len(data))
	}
}

func writeXiaozhiHello(t *testing.T, ctx context.Context, conn *websocket.Conn, overrides map[string]any) {
	t.Helper()
	hello := map[string]any{
		"type":      "hello",
		"version":   1,
		"transport": "websocket",
		"features": map[string]any{
			"mcp": true,
			"aec": true,
		},
		"audio": map[string]any{
			"format":         "opus",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": 60,
		},
	}
	for key, value := range overrides {
		hello[key] = value
	}
	if err := wsjson.Write(ctx, conn, hello); err != nil {
		t.Fatal(err)
	}
}

type blockingXiaozhiPipelineRunner struct {
	entered  chan struct{}
	canceled chan struct{}
	release  chan struct{}
	once     sync.Once
}

func newBlockingXiaozhiPipelineRunner() *blockingXiaozhiPipelineRunner {
	return &blockingXiaozhiPipelineRunner{
		entered:  make(chan struct{}),
		canceled: make(chan struct{}),
		release:  make(chan struct{}),
	}
}

func (r *blockingXiaozhiPipelineRunner) unblock() {
	r.once.Do(func() {
		close(r.release)
	})
}

func (r *blockingXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	close(r.entered)
	select {
	case <-ctx.Done():
		close(r.canceled)
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
			Status: providers.VoicePipelineStatusFailed,
			Report: providers.VoicePipelineReport{
				SchemaVersion: "a21.voice_pipeline.fixture.v1",
				Status:        string(providers.VoicePipelineStatusFailed),
				TraceID:       req.Session.TraceID,
				SessionID:     req.Session.SessionID,
				DeviceID:      req.Session.DeviceID,
				Mode:          req.Mode,
				ExecutionMode: "fixture",
			},
		}, nil
	}
}

type gatewayFinalASRAdapter struct{}

func (gatewayFinalASRAdapter) Name() string {
	return "a21-gateway-final-asr"
}

func (gatewayFinalASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.ASRAdapterEvent{Text: "gateway streaming test transcript", Final: true}:
		}
	}()
	return out, nil
}

type blockingCommitStreamingASRAdapter struct {
	events        chan providers.ASRAdapterEvent
	appended      chan struct{}
	commitEntered chan struct{}
	release       chan struct{}
	appendOnce    sync.Once
	commitOnce    sync.Once
	releaseOnce   sync.Once
	transcribe    chan struct{}
}

type reusableStreamingASRAdapter struct {
	name string
}

type errorEventStreamingASRAdapter struct {
	finding string
	err     error
}

type errorEventStreamingASRSession struct {
	events chan providers.ASRAdapterEvent
}

func (a reusableStreamingASRAdapter) Name() string {
	if strings.TrimSpace(a.name) == "" {
		return "mock-streaming-asr"
	}
	return a.name
}

func (a reusableStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	return providers.NewMockASRAdapter(a.Name()).Transcribe(ctx, req)
}

func (a reusableStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	return providers.NewMockStreamingASRAdapter(a.Name()).StartStreamingASR(ctx, req)
}

func newErrorEventStreamingASRAdapter(finding string, err error) errorEventStreamingASRAdapter {
	return errorEventStreamingASRAdapter{finding: finding, err: err}
}

func (a errorEventStreamingASRAdapter) Name() string {
	return "a21-error-event-streaming-asr"
}

func (a errorEventStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent)
	close(out)
	return out, nil
}

func (a errorEventStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	events := make(chan providers.ASRAdapterEvent, 1)
	events <- providers.ASRAdapterEvent{Finding: a.finding, Err: a.err}
	close(events)
	return &errorEventStreamingASRSession{events: events}, nil
}

func (s *errorEventStreamingASRSession) AppendFrame(ctx context.Context, frame providers.VoicePipelinePCMFrame) error {
	return nil
}

func (s *errorEventStreamingASRSession) Events() <-chan providers.ASRAdapterEvent {
	return s.events
}

func (s *errorEventStreamingASRSession) Commit(ctx context.Context) error {
	return nil
}

func (s *errorEventStreamingASRSession) Cancel(error) {}

func newBlockingCommitStreamingASRAdapter() *blockingCommitStreamingASRAdapter {
	return &blockingCommitStreamingASRAdapter{
		events:        make(chan providers.ASRAdapterEvent, 2),
		appended:      make(chan struct{}),
		commitEntered: make(chan struct{}),
		release:       make(chan struct{}),
		transcribe:    make(chan struct{}, 1),
	}
}

func (a *blockingCommitStreamingASRAdapter) Name() string {
	return "a21-blocking-commit-streaming-asr"
}

func (a *blockingCommitStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	select {
	case a.transcribe <- struct{}{}:
	default:
	}
	out := make(chan providers.ASRAdapterEvent)
	close(out)
	return out, nil
}

func (a *blockingCommitStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	return a, nil
}

func (a *blockingCommitStreamingASRAdapter) AppendFrame(ctx context.Context, frame providers.VoicePipelinePCMFrame) error {
	a.appendOnce.Do(func() {
		close(a.appended)
	})
	return nil
}

func (a *blockingCommitStreamingASRAdapter) Events() <-chan providers.ASRAdapterEvent {
	return a.events
}

func (a *blockingCommitStreamingASRAdapter) Commit(ctx context.Context) error {
	a.commitOnce.Do(func() {
		close(a.commitEntered)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.release:
		a.events <- providers.ASRAdapterEvent{Text: "streaming final after async commit", Final: true}
		close(a.events)
		return nil
	}
}

func (a *blockingCommitStreamingASRAdapter) Cancel(error) {
	a.releaseCommit()
}

func (a *blockingCommitStreamingASRAdapter) releaseCommit() {
	a.releaseOnce.Do(func() {
		close(a.release)
	})
}

type blockingAppendStreamingASRAdapter struct {
	events        chan providers.ASRAdapterEvent
	appendEntered chan struct{}
	release       chan struct{}
	appendOnce    sync.Once
	releaseOnce   sync.Once
}

func newBlockingAppendStreamingASRAdapter() *blockingAppendStreamingASRAdapter {
	return &blockingAppendStreamingASRAdapter{
		events:        make(chan providers.ASRAdapterEvent),
		appendEntered: make(chan struct{}),
		release:       make(chan struct{}),
	}
}

func (a *blockingAppendStreamingASRAdapter) Name() string {
	return "a21-blocking-append-streaming-asr"
}

func (a *blockingAppendStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent)
	close(out)
	return out, nil
}

func (a *blockingAppendStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	return a, nil
}

func (a *blockingAppendStreamingASRAdapter) AppendFrame(ctx context.Context, frame providers.VoicePipelinePCMFrame) error {
	a.appendOnce.Do(func() {
		close(a.appendEntered)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.release:
		return nil
	}
}

func (a *blockingAppendStreamingASRAdapter) Events() <-chan providers.ASRAdapterEvent {
	return a.events
}

func (a *blockingAppendStreamingASRAdapter) Commit(ctx context.Context) error {
	return nil
}

func (a *blockingAppendStreamingASRAdapter) Cancel(error) {
	a.releaseAppend()
}

func (a *blockingAppendStreamingASRAdapter) releaseAppend() {
	a.releaseOnce.Do(func() {
		close(a.release)
		close(a.events)
	})
}

type lateFinalStreamingASRAdapter struct {
	events        chan providers.ASRAdapterEvent
	appended      chan struct{}
	commitEntered chan struct{}
	delay         time.Duration
	appendOnce    sync.Once
	commitOnce    sync.Once
}

func newLateFinalStreamingASRAdapter(delay time.Duration) *lateFinalStreamingASRAdapter {
	return &lateFinalStreamingASRAdapter{
		events:        make(chan providers.ASRAdapterEvent, 2),
		appended:      make(chan struct{}),
		commitEntered: make(chan struct{}),
		delay:         delay,
	}
}

func (a *lateFinalStreamingASRAdapter) Name() string {
	return "a21-late-final-streaming-asr"
}

func (a *lateFinalStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent)
	close(out)
	return out, nil
}

func (a *lateFinalStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	return a, nil
}

func (a *lateFinalStreamingASRAdapter) AppendFrame(ctx context.Context, frame providers.VoicePipelinePCMFrame) error {
	a.appendOnce.Do(func() {
		close(a.appended)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case a.events <- providers.ASRAdapterEvent{Text: "late partial before stop"}:
		return nil
	}
}

func (a *lateFinalStreamingASRAdapter) Events() <-chan providers.ASRAdapterEvent {
	return a.events
}

func (a *lateFinalStreamingASRAdapter) Commit(ctx context.Context) error {
	a.commitOnce.Do(func() {
		close(a.commitEntered)
		go func() {
			time.Sleep(a.delay)
			a.events <- providers.ASRAdapterEvent{Text: "late streaming final after commit timeout", Final: true}
			close(a.events)
		}()
	})
	return nil
}

func (a *lateFinalStreamingASRAdapter) Cancel(error) {
}

type scriptedProfessionalASRAdapter struct {
	text string
	err  error
}

func (a scriptedProfessionalASRAdapter) Name() string {
	return "a21-scripted-professional-asr"
}

func (a scriptedProfessionalASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.ASRAdapterEvent{Text: a.text, Final: true, Err: a.err}:
		}
	}()
	return out, nil
}

type triggerPhraseStreamingASRAdapter struct {
	finalText      string
	events         chan providers.ASRAdapterEvent
	appended       chan struct{}
	transcribed    chan struct{}
	appendOnce     sync.Once
	commitOnce     sync.Once
	transcribeOnce sync.Once
}

func newTriggerPhraseStreamingASRAdapter(finalText string) *triggerPhraseStreamingASRAdapter {
	return &triggerPhraseStreamingASRAdapter{
		finalText:   finalText,
		events:      make(chan providers.ASRAdapterEvent, 2),
		appended:    make(chan struct{}),
		transcribed: make(chan struct{}),
	}
}

func (a *triggerPhraseStreamingASRAdapter) Name() string {
	return "a21-trigger-phrase-streaming-asr"
}

func (a *triggerPhraseStreamingASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	a.transcribeOnce.Do(func() {
		close(a.transcribed)
	})
	out := make(chan providers.ASRAdapterEvent)
	close(out)
	return out, ctx.Err()
}

func (a *triggerPhraseStreamingASRAdapter) StartStreamingASR(ctx context.Context, req providers.StreamingASRStartRequest) (providers.StreamingASRSession, error) {
	return a, ctx.Err()
}

func (a *triggerPhraseStreamingASRAdapter) AppendFrame(ctx context.Context, frame providers.VoicePipelinePCMFrame) error {
	a.appendOnce.Do(func() {
		close(a.appended)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case a.events <- providers.ASRAdapterEvent{Text: "给我证据", Final: false}:
		return nil
	}
}

func (a *triggerPhraseStreamingASRAdapter) Events() <-chan providers.ASRAdapterEvent {
	return a.events
}

func (a *triggerPhraseStreamingASRAdapter) Commit(ctx context.Context) error {
	a.commitOnce.Do(func() {
		select {
		case <-ctx.Done():
		case a.events <- providers.ASRAdapterEvent{Text: a.finalText, Final: true}:
		}
		close(a.events)
	})
	return ctx.Err()
}

func (a *triggerPhraseStreamingASRAdapter) Cancel(error) {
	a.commitOnce.Do(func() {
		close(a.events)
	})
}

type blockingProfessionalASRAdapter struct {
	started     chan struct{}
	released    chan struct{}
	canceled    chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
	cancelOnce  sync.Once
}
