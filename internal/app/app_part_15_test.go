package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/v21adapter"
)

func TestRunStackChanFastCompanionTurnConsumesStackChanMicEvidenceWithoutLeakingAudio(t *testing.T) {
	original := synthesizeMacOSSay
	t.Cleanup(func() { synthesizeMacOSSay = original })
	synthesizeMacOSSay = func(ctx context.Context, options audio.LocalTTSOptions) (audio.LocalTTSReport, error) {
		outputPath := filepath.Join(options.OutputDir, fmt.Sprintf("a21-fast-companion-mic-%d.wav", time.Now().UnixNano()))
		writeAppTestWAV(t, outputPath, 16000, bytes.Repeat([]byte{1, 0}, 640))
		return audio.LocalTTSReport{
			SchemaVersion:   "a21.audio.local_tts.v1",
			GeneratedAtMS:   time.Now().UnixMilli(),
			Status:          "passed",
			Provider:        "macos_say",
			Voice:           "Tingting",
			OutputFormat:    "wav_pcm_s16le_16000_mono",
			OutputPath:      outputPath,
			OutputBytes:     1280,
			TextBytes:       len([]byte(options.Text)),
			DurationMS:      11,
			TTSFirstAudioMS: 11,
		}, nil
	}
	payload := appTestPCM16Base64(12000)
	frames := make([]string, 0, 20)
	for seq := 1; seq <= 20; seq++ {
		rms := 0.21
		if seq == 1 {
			rms = 0.23
		}
		frames = append(frames, fmt.Sprintf(`{"device_id":"stackchan-001","trace_id":"a21-trace-audio-%06d","session_id":"%%s","seq":%d,"sample_rate_hz":16000,"channels":1,"duration_ms":20,"data_bytes":640,"data_base64":"%s","rms":%.2f,"vad_detector":"a21-rms-vad","speech_detected":true,"speech_active":true,"ingress_buffer_frames":%d}`, seq, seq, payload, rms, seq))
	}
	var sawListening bool
	var sawIdle bool
	var audioRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/audio/recent":
			if r.URL.Query().Get("include_audio") != "1" {
				t.Fatalf("recent audio request missing include_audio=1: %s", r.URL.RawQuery)
			}
			if r.URL.Query().Get("device_id") != "stackchan-001" {
				t.Fatalf("recent audio device = %q", r.URL.Query().Get("device_id"))
			}
			limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil || limit < 64 {
				t.Fatalf("recent audio limit = %q, want enough frames for capture window", r.URL.Query().Get("limit"))
			}
			sessionID := r.URL.Query().Get("session_id")
			responseFrames := strings.ReplaceAll(strings.Join(frames, ","), "%s", sessionID)
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.audio_recent.v1","device_id":"stackchan-001","session_id":"%s","include_audio":true,"frames":[%s]}`, sessionID, responseFrames)
		case "/v1/devices/control":
			var request struct {
				DeviceID       string `json:"device_id"`
				State          string `json:"state"`
				Text           string `json:"text"`
				AudioProbeOnly bool   `json:"audio_probe_only"`
				AudioChunks    []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.DeviceID != "stackchan-001" {
				t.Fatalf("device id = %q", request.DeviceID)
			}
			if request.State == "listening" && request.AudioProbeOnly {
				sawListening = true
			}
			if request.State == "idle" {
				sawIdle = true
			}
			if len(request.AudioChunks) > 0 {
				audioRequests++
			}
			fmt.Fprint(w, `{"trace_id":"a21-trace-fast","session_id":"a21-session-fast","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/traces":
			traceID := r.URL.Query().Get("trace_id")
			if !strings.HasPrefix(traceID, "a21-trace-fast-companion-") {
				t.Fatalf("trace id = %q, want fast companion trace", traceID)
			}
			fmt.Fprintf(w, `{"trace_id":"%s","events":[{"name":"barge_in.detected","trace_id":"%s","session_id":"a21-session-fast","device_id":"stackchan-001","at_ms":1000,"offset_ms":100},{"name":"playback.stop","trace_id":"%s","session_id":"a21-session-fast","device_id":"stackchan-001","at_ms":1024,"offset_ms":124}],"summary":{"event_count":2,"last_offset_ms":124,"barge_in_stop_ms":24}}`, traceID, traceID, traceID)
		case "/ws/audio":
			http.NotFound(w, r)
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--engine", "macos_say", "--listen-source", "stackchan_mic", "--repeat", "1", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s\n%s", code, stderr.String(), stdout.String())
	}
	if !sawListening || !sawIdle || audioRequests < 2 {
		t.Fatalf("listening/idle/audio = %t/%t/%d, want physical listen and ack/answer playback", sawListening, sawIdle, audioRequests)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"listen_source": "stackchan_mic"`,
		`"listen_detected": true`,
		`"physical_mic_frame_count": 20`,
		`"physical_mic_audio_bytes": 12800`,
		`"physical_mic_duration_ms": 400`,
		`"physical_mic_capture_window_ms": 1200`,
		`"physical_mic_rms_max": 0.23`,
		`"physical_mic_wav_name"`,
		`"barge_in_stop_ms": 24`,
		`"barge_in_stop_p95_ms": 24`,
		`"m3_candidate": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-stackchan-fast-companion-turn-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{payload, "data_base64", "Authorization", "Bearer", "sk-"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportData), forbidden) {
			t.Fatalf("stackchan mic report leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportData)
		}
	}
}

func TestRunStackChanFastCompanionTurnRejectsStackChanMicWithoutPhysicalEvidence(t *testing.T) {
	var controlRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","rgb":"available","servo_y":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/v1/audio/recent":
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.audio_recent.v1","device_id":"stackchan-001","session_id":"%s","include_audio":true,"frames":[]}`, r.URL.Query().Get("session_id"))
		case "/v1/devices/control":
			controlRequests++
			var request struct {
				State       string `json:"state"`
				AudioChunks []struct {
					DataBase64 string `json:"data_base64"`
				} `json:"audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request.AudioChunks) > 0 {
				t.Fatalf("stackchan_mic without physical evidence must not deliver synthetic playback")
			}
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"stackchan-fast-companion-turn", "--gateway-url", server.URL, "--device-id", "stackchan-001", "--listen-source", "stackchan_mic", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if controlRequests == 0 {
		t.Fatalf("control requests = %d, want listening and cleanup controls", controlRequests)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_fast_companion_turn.v1"`,
		`"status": "failed"`,
		`"listen_source": "stackchan_mic"`,
		`"m3_candidate": false`,
		`"physical mic evidence missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunV21AdapterSmokeExecutesQueryAndWritesRedactedReport(t *testing.T) {
	var sawProfessionalRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("path = %q, want /a21/v21/query", r.URL.Path)
		}
		var request struct {
			TraceID            string `json:"trace_id"`
			SessionID          string `json:"session_id"`
			Mode               string `json:"mode"`
			Utterance          string `json:"utterance"`
			LatencyProfile     string `json:"latency_profile"`
			AnswerStyle        string `json:"answer_style"`
			PrivacyScope       string `json:"privacy_scope"`
			MaxFirstResponseMS int    `json:"max_first_response_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawProfessionalRequest = request.TraceID != "" &&
			request.SessionID != "" &&
			request.Mode == "professional" &&
			request.Utterance == "查一下语音唤醒误触发" &&
			request.LatencyProfile == "fast_first" &&
			request.AnswerStyle == "voice_first_with_citations" &&
			request.PrivacyScope == "professional_only" &&
			request.MaxFirstResponseMS == 1200
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-v21-smoke",
			"fast_answer":"历史讨论集中在多人说话和相似音节误唤醒。",
			"confidence":0.82,
			"evidence":[{"title":"语音唤醒体验复盘","type":"meeting","source_id":"v21-doc-001","summary":"提到多人说话导致误唤醒。"}],
			"speech_blocks":["我先说结论。"],
			"screen_cards":[{"label":"结论","text":"误唤醒集中在 2 类场景"}],
			"follow_ups":["要不要按车型展开？"]
		}`))
	}))
	defer server.Close()
	dir, err := os.MkdirTemp("", "a21-adapter-smoke-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-adapter-smoke",
		"--adapter-url", server.URL,
		"--query", "查一下语音唤醒误触发",
		"--execute",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawProfessionalRequest {
		t.Fatal("server did not receive professional V21 adapter smoke request")
	}
	for _, want := range []string{
		`"schema_version": "a21.v21_adapter_smoke.v1"`,
		`"generated_at_ms"`,
		`"adapter": "a21-v21-adapter"`,
		`"status": "passed"`,
		`"executed": true`,
		`"mode": "professional"`,
		`"latency_profile": "fast_first"`,
		`"answer_style": "voice_first_with_citations"`,
		`"privacy_scope": "professional_only"`,
		`"max_first_response_ms": 1200`,
		`"query_path": "/a21/v21/query"`,
		`"evidence_count": 1`,
		`"speech_block_count": 1`,
		`"screen_card_count": 1`,
		`"follow_up_count": 1`,
		`"redaction_ok": true`,
		`"report_path": "a21-v21-adapter-smoke-`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-v21-adapter-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("v21 adapter smoke reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{"语音唤醒", "历史讨论", server.URL, dir, filepath.ToSlash(dir)} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("v21 adapter smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunV21ProfessionalReadinessWritesRedactedHostReport(t *testing.T) {
	dir, err := os.MkdirTemp("", "a21-prof-readiness-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-professional-readiness",
		"--adapter-url", "http://127.0.0.1:21121",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.v21_professional_readiness.v1"`,
		`"checking_ack_available": true`,
		`"checking_ack_within_1200": true`,
		`"evidence_available": true`,
		`"cards_available": true`,
		`"follow_ups_available": true`,
		`"adapter_configured": true`,
		`"adapter_executed": false`,
		`"redaction_ok": true`,
		`"professional_acceptance_status": "host_mock_ready"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %s: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-v21-professional-readiness-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("professional readiness reports = %d, %v", len(matches), err)
	}
	reportJSON, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"professional readiness fixture query",
		"语音唤醒体验复盘",
		"提到多人说话",
		"要不要按车型",
		"http://",
		"https://",
		dir,
		"secret",
		"api_key",
		"provider output",
		"reasoning",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(string(reportJSON), forbidden) {
			t.Fatalf("professional readiness leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
	if !strings.Contains(stdout.String(), `"report_path": "a21-v21-professional-readiness-`) {
		t.Fatalf("stdout report_path should be basename-only: %s", stdout.String())
	}
}

func TestRunV21ProfessionalReadinessReportsMisconfigurationWithoutLeak(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"v21-professional-readiness",
		"--adapter-url", "http://user:secret-token@127.0.0.1:21121/a21/v21/query",
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"adapter_configured": true`,
		`"adapter_executed": false`,
		`"professional_acceptance_status": "adapter_misconfigured"`,
		`"code": "v21_adapter_misconfigured"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %s: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret-token", "user:", "http://", "127.0.0.1:21121", "/a21/v21/query"} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("misconfigured readiness leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestV21AdapterBridgeExecutesNativeVoiceQueryScopeContract(t *testing.T) {
	activeReleaseID := "rel_active"
	var sawNativeVoiceQuery bool
	var sawRetrievalQuery bool
	var voiceRequest struct {
		DeviceID      string   `json:"device_id"`
		UserID        string   `json:"user_id"`
		WorkspaceID   string   `json:"workspace_id"`
		QueryScope    string   `json:"query_scope"`
		CollectionIDs []string `json:"collection_ids"`
		Question      string   `json:"question"`
		Mode          string   `json:"mode"`
	}
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			if r.Header.Get("X-Dev-Role") != "VIEWER" {
				t.Fatalf("missing viewer dev header")
			}
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/internal/v1/knowledge/voice-query":
			if err := json.NewDecoder(r.Body).Decode(&voiceRequest); err != nil {
				t.Fatal(err)
			}
			sawNativeVoiceQuery = true
			writeV21BridgeJSON(w, http.StatusOK, v21VoiceQueryResponse{
				QueryRunID:   "qr_child_lock",
				SpokenAnswer: "G02ES 和 G02ESVR 支持座椅儿童锁。",
				FullAnswer:   "G02ES、G02ESVR 支持座椅儿童锁，证据来自儿童锁证据。",
				Confidence:   0.91,
				Evidence: []v21VoiceEvidence{{
					AnchorID:    "anchor_child_lock",
					ChunkID:     "chunk_child_lock",
					SourceLabel: "儿童锁证据",
					SourceScope: "public",
					Excerpt:     "G02ES、G02ESVR 支持座椅儿童锁。",
					Score:       0.91,
				}},
				SourceScopeCounts: map[string]int{"public": 1},
				WorkspaceStatus:   v21adapter.WorkspaceSearchable,
			})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			sawRetrievalQuery = true
			http.Error(w, "direct retrieval should not be called on native voice-query success", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"v21-adapter-smoke", "--adapter-url", adapter.URL, "--query", "哪些车有儿童锁", "--execute"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !sawNativeVoiceQuery {
		t.Fatal("bridge did not call V21 native voice-query")
	}
	if sawRetrievalQuery {
		t.Fatal("bridge called direct retrieval on native voice-query success")
	}
	if voiceRequest.DeviceID != "" {
		t.Fatalf("device_id = %q, want smoke default empty", voiceRequest.DeviceID)
	}
	if voiceRequest.UserID != v21adapter.DefaultUserID {
		t.Fatalf("user_id = %q, want default", voiceRequest.UserID)
	}
	if voiceRequest.WorkspaceID != v21adapter.DefaultWorkspaceID {
		t.Fatalf("workspace_id = %q, want default", voiceRequest.WorkspaceID)
	}
	if voiceRequest.QueryScope != v21adapter.QueryScopePublic {
		t.Fatalf("query_scope = %q, want public_only", voiceRequest.QueryScope)
	}
	if !reflect.DeepEqual(voiceRequest.CollectionIDs, []string{"col_vehicle"}) {
		t.Fatalf("collection_ids = %#v, want col_vehicle", voiceRequest.CollectionIDs)
	}
	if voiceRequest.Question != "哪些车有儿童锁" || voiceRequest.Mode != "grounded_qa" {
		t.Fatalf("voice request = %+v, want question and grounded_qa", voiceRequest)
	}
	for _, want := range []string{
		`"status": "passed"`,
		`"evidence_count": 1`,
		`"follow_up_count": 1`,
		`"confidence": 0.91`,
		`"query_scope": "public_only"`,
		`"workspace_status": "searchable"`,
		`"source_scope_counts": {`,
		`"public": 1`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestV21AdapterBridgePassesExplicitWorkspaceScopeToNativeVoiceQuery(t *testing.T) {
	activeReleaseID := "rel_active"
	var retrievalCalls int
	var voiceRequest struct {
		DeviceID      string   `json:"device_id"`
		UserID        string   `json:"user_id"`
		WorkspaceID   string   `json:"workspace_id"`
		QueryScope    string   `json:"query_scope"`
		CollectionIDs []string `json:"collection_ids"`
		Question      string   `json:"question"`
		TraceID       string   `json:"trace_id"`
		SessionID     string   `json:"session_id"`
	}
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/internal/v1/knowledge/voice-query":
			if err := json.NewDecoder(r.Body).Decode(&voiceRequest); err != nil {
				t.Fatal(err)
			}
			writeV21BridgeJSON(w, http.StatusOK, v21VoiceQueryResponse{
				QueryRunID:   "qr_scope_combined",
				SpokenAnswer: "公共资料和个人资料各有一条可引用证据。",
				FullAnswer:   "公共资料和个人资料各有一条可引用证据，已按来源范围返回计数。",
				Confidence:   0.86,
				Evidence: []v21VoiceEvidence{
					{
						AnchorID:    "anchor_public",
						ChunkID:     "chunk_public",
						SourceLabel: "公共资料",
						SourceScope: "public",
						Excerpt:     "公共资料里记录了儿童锁配置。",
						Score:       0.86,
					},
					{
						AnchorID:    "anchor_personal",
						ChunkID:     "chunk_personal",
						SourceLabel: "个人资料",
						SourceScope: "personal",
						Excerpt:     "个人资料里记录了本次标注。",
						Score:       0.84,
					},
				},
				SourceScopeCounts: map[string]int{"public": 1, "personal": 1},
				WorkspaceStatus:   v21adapter.WorkspaceSearchable,
			})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			retrievalCalls++
			http.Error(w, "direct retrieval should not be called on native voice-query success", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	resp, err := http.Post(adapter.URL+v21adapter.QueryPath, "application/json", strings.NewReader(`{
		"trace_id":"a21-trace-v21-explicit",
		"session_id":"a21-session-v21-explicit",
		"device_id":"stackchan-sim-001",
		"user_id":"a21_user_test",
		"workspace_id":"a21_workspace_test",
		"mode":"professional",
		"privacy_scope":"professional_only",
		"query_scope":"personal_plus_public",
		"utterance":"查儿童锁并结合我的标注"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var response v21adapter.QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %+v", resp.StatusCode, response)
	}
	if voiceRequest.DeviceID != "stackchan-sim-001" || voiceRequest.UserID != "a21_user_test" || voiceRequest.WorkspaceID != "a21_workspace_test" {
		t.Fatalf("voice request ids = %+v, want explicit A21 workspace scope labels", voiceRequest)
	}
	if voiceRequest.QueryScope != v21adapter.QueryScopeCombined {
		t.Fatalf("query_scope = %q, want personal_plus_public", voiceRequest.QueryScope)
	}
	if !reflect.DeepEqual(voiceRequest.CollectionIDs, []string{"col_vehicle"}) {
		t.Fatalf("collection_ids = %#v, want col_vehicle", voiceRequest.CollectionIDs)
	}
	if voiceRequest.Question != "查儿童锁并结合我的标注" {
		t.Fatalf("question = %q, want explicit utterance", voiceRequest.Question)
	}
	if retrievalCalls != 0 {
		t.Fatalf("retrieval calls = %d, want 0", retrievalCalls)
	}
	if !reflect.DeepEqual(response.SourceScopeCounts, map[string]int{"public": 1, "personal": 1}) {
		t.Fatalf("source_scope_counts = %#v, want public+personal from V21", response.SourceScopeCounts)
	}
	if response.WorkspaceStatus != v21adapter.WorkspaceSearchable {
		t.Fatalf("workspace_status = %q, want searchable", response.WorkspaceStatus)
	}
}

func TestV21AdapterBridgeRetriesChildLockASRFragmentWithoutLeakingQuery(t *testing.T) {
	activeReleaseID := "rel_active"
	var queries []string
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/internal/v1/knowledge/voice-query":
			writeV21BridgeJSON(w, http.StatusFailedDependency, v21BridgeQueryError{
				Code:        "no_evidence",
				StatusClass: "status_4xx",
			})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			var request struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			queries = append(queries, request.Query)
			if request.Query != "儿童锁 车型 车门" {
				writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
					CollectionID: "col_vehicle",
					Results:      []v21RetrievalResult{},
				})
				return
			}
			writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
				CollectionID: "col_vehicle",
				Results: []v21RetrievalResult{{
					AnchorID:    "ca_child_lock",
					SourceLabel: "儿童锁证据",
					SourceScope: "public",
					Excerpt:     "G02ES、G02ESVR 支持座椅儿童锁。",
					Score:       0.91,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"v21-adapter-smoke", "--adapter-url", adapter.URL, "--query", "儿童是", "--execute"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !reflect.DeepEqual(queries, []string{"儿童是", "儿童锁 车型 车门"}) {
		t.Fatalf("queries = %#v, want original then child-lock expansion", queries)
	}
	for _, want := range []string{`"status": "passed"`, `"evidence_count": 1`, `"speech_block_count": 1`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"儿童是", "儿童锁 车型 车门", v21Backend.URL} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("adapter smoke leaked forbidden query fragment %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestV21AdapterBridgeRejectsNonProfessionalModeBeforeRetrieval(t *testing.T) {
	activeReleaseID := "rel_active"
	var retrievalCalls int
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			retrievalCalls++
			http.Error(w, "should not be called", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	resp, err := http.Post(adapter.URL+v21adapter.QueryPath, "application/json", strings.NewReader(`{
		"trace_id":"a21-trace-v21-unsafe",
		"session_id":"a21-session-v21-unsafe",
		"mode":"workmate",
		"privacy_scope":"professional_only",
		"utterance":"查一下语音唤醒误触发"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if retrievalCalls != 0 {
		t.Fatalf("retrieval calls = %d, want 0", retrievalCalls)
	}
}

func TestV21AdapterBridgeNoResultsReturnsControlledRedactedFailure(t *testing.T) {
	activeReleaseID := "rel_active"
	v21Backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/healthz":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/collections":
			writeV21BridgeJSON(w, http.StatusOK, []v21CollectionView{{
				ID:              "col_vehicle",
				Name:            "Vehicle Knowledge",
				ActiveReleaseID: &activeReleaseID,
			}})
		case "/internal/v1/knowledge/voice-query":
			writeV21BridgeJSON(w, http.StatusFailedDependency, v21BridgeQueryError{
				Code:        "no_evidence",
				StatusClass: "status_4xx",
			})
		case "/api/v1/collections/col_vehicle/retrieval/query":
			writeV21BridgeJSON(w, http.StatusOK, v21RetrievalQueryResponse{
				CollectionID: "col_vehicle",
				Results:      []v21RetrievalResult{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v21Backend.Close()
	handler, err := newV21AdapterBridgeHandler(context.Background(), v21AdapterBridgeOptions{V21URL: v21Backend.URL})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httptest.NewServer(handler)
	defer adapter.Close()

	resp, err := http.Post(adapter.URL+v21adapter.QueryPath, "application/json", strings.NewReader(`{
		"trace_id":"a21-trace-v21-no-results",
		"session_id":"a21-session-v21-no-results",
		"mode":"professional",
		"privacy_scope":"professional_only",
		"utterance":"查一下语音唤醒误触发"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFailedDependency {
		t.Fatalf("status = %d, want 424: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"code":"no_evidence"`) {
		t.Fatalf("body missing redacted no_evidence code: %s", body)
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", v21Backend.URL, "col_vehicle"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("bridge failure leaked %q: %s", forbidden, body)
		}
	}
}

func TestRunLANProbeWritesDirectRedactedReport(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=" + listener.Addr().String(),
		"--samples", "3",
		"--output-dir", dir,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.lan_probe.v1"`,
		`"ok": true`,
		`"name": "a21-gateway"`,
		`"status": "passed"`,
		`"direct": true`,
		`"samples": 3`,
		`"passed_samples": 3`,
		`"failed_samples": 0`,
		`"p50_ms"`,
		`"p95_ms"`,
		`"jitter_ms"`,
		`"metadata"`,
		`"proxy"`,
		`"report_path"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-lan-probe-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("lan probe reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "x21"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}
