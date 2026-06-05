package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
)

func TestProfessionalReadRecordsCompleteWithRedactedScopeMetadata(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: readRecordV21Client{}})
	handler := server.Handler()
	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_read","workspace_id":"a21_workspace_read","query_scope":"personal_plus_public"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"RAW_PRIVATE_QUERY_FOR_READ_LEDGER","mode":"professional","trace_id":"a21-trace-read-record","session_id":"a21-session-read-record"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("turn status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	recordsReq := httptest.NewRequest(http.MethodGet, "/v1/professional-read-records?trace_id=a21-trace-read-record", nil)
	recordsRec := httptest.NewRecorder()
	handler.ServeHTTP(recordsRec, recordsReq)
	if recordsRec.Code != http.StatusOK {
		t.Fatalf("read-record status = %d, want 200: %s", recordsRec.Code, recordsRec.Body.String())
	}
	var response ProfessionalReadRecordsResponse
	if err := json.Unmarshal(recordsRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.professional_read_records.v1" || response.Status != "ok" || len(response.Records) != 1 {
		t.Fatalf("read-record response = %+v", response)
	}
	record := response.Records[0]
	if record.RecordID == "" ||
		record.Status != "completed" ||
		record.TraceID != "a21-trace-read-record" ||
		record.SessionID != "a21-session-read-record" ||
		record.DeviceID != "stackchan-sim-001" ||
		record.UserID != "a21_user_read" ||
		record.WorkspaceID != "a21_workspace_read" ||
		record.QueryScope != "personal_plus_public" ||
		record.PrivacyScope != "professional_only" ||
		record.LatencyProfile != "fast_first" ||
		record.AnswerStyle != "voice_first_with_citations" ||
		record.UtteranceBucket != "length_17_64" ||
		record.WorkspaceStatus != "searchable" ||
		record.SourceScopeCounts["public"] != 2 ||
		record.SourceScopeCounts["personal"] != 1 ||
		record.StartedAtMS == 0 ||
		record.CompletedAtMS == 0 ||
		record.CompletedAtMS < record.StartedAtMS {
		t.Fatalf("read record = %+v", record)
	}
	if response.Redaction.QueryTextStored || response.Redaction.RetrievedTextStored || response.Redaction.ProviderOutputStored ||
		response.Redaction.DocumentTextStored || response.Redaction.FullURLStored || response.Redaction.LocalPathStored ||
		response.Redaction.CredentialValueStored || response.Redaction.VoiceTranscriptStored {
		t.Fatalf("response redaction = %+v", response.Redaction)
	}
	if record.Redaction.QueryTextStored || record.Redaction.RetrievedTextStored || record.Redaction.ProviderOutputStored ||
		record.Redaction.DocumentTextStored || record.Redaction.FullURLStored || record.Redaction.LocalPathStored ||
		record.Redaction.CredentialValueStored || record.Redaction.VoiceTranscriptStored {
		t.Fatalf("record redaction = %+v", record.Redaction)
	}
	for _, forbidden := range []string{"RAW_PRIVATE_QUERY", "RAW_SECRET", "https://", "/Users/", "api_key", "token"} {
		if strings.Contains(strings.ToLower(recordsRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("read records leaked %q: %s", forbidden, recordsRec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-read-record", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"professional.read_record.started", "professional.read_record.completed"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestProfessionalReadRecordsFailSafelyWhenV21Unavailable(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: failingV21Client{}})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"RAW_PRIVATE_QUERY_FAILURE","mode":"professional","trace_id":"a21-trace-read-failed","session_id":"a21-session-read-failed"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("turn status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	recordsReq := httptest.NewRequest(http.MethodGet, "/v1/professional-read-records?trace_id=a21-trace-read-failed", nil)
	recordsRec := httptest.NewRecorder()
	handler.ServeHTTP(recordsRec, recordsReq)
	if recordsRec.Code != http.StatusOK {
		t.Fatalf("read-record status = %d, want 200: %s", recordsRec.Code, recordsRec.Body.String())
	}
	var response ProfessionalReadRecordsResponse
	if err := json.Unmarshal(recordsRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("read-record response = %+v", response)
	}
	record := response.Records[0]
	if record.Status != "failed" ||
		record.FailureCode != "query_error" ||
		record.TraceID != "a21-trace-read-failed" ||
		record.UtteranceBucket != "length_17_64" ||
		record.CompletedAtMS == 0 {
		t.Fatalf("failed read record = %+v", record)
	}
	for _, forbidden := range []string{"RAW_PRIVATE_QUERY", "v21 unavailable", "https://", "/Users/", "api_key", "token"} {
		if strings.Contains(strings.ToLower(recordsRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("failed read records leaked %q: %s", forbidden, recordsRec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-read-failed", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "professional.read_record.failed") {
		t.Fatalf("trace missing failed read-record marker: %s", traceRec.Body.String())
	}
}

func TestFastCompanionRejectsProfessionalVoiceModeWithoutProviderOrV21Execution(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true}},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: provider,
		V21Client:     v21,
	})
	handler := server.Handler()

	selectReq := httptest.NewRequest(http.MethodPost, "/v1/voice-modes", bytes.NewBufferString(`{"voice_mode":"professional"}`))
	selectRec := httptest.NewRecorder()
	handler.ServeHTTP(selectRec, selectReq)
	if selectRec.Code != http.StatusOK {
		t.Fatalf("select status = %d, want 200: %s", selectRec.Code, selectRec.Body.String())
	}

	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","mode":"workmate","local_audio":{"asr_provider":"mock_asr","first_partial_ms":42,"final_transcript_chars":4}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for professional voice mode: %s", rec.Code, rec.Body.String())
	}
	if provider.startCalls != 0 || v21.calls != 0 {
		t.Fatalf("professional voice mode executed provider/v21 from dialogue endpoint: provider=%d v21=%d", provider.startCalls, v21.calls)
	}
	if !strings.Contains(rec.Body.String(), "voice_mode") || strings.Contains(rec.Body.String(), "should not run") {
		t.Fatalf("unexpected error body: %s", rec.Body.String())
	}
}

func TestWakeWordConfigEndpointPersistsCustomMultinetRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21-wake-word.json")
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	handler := server.Handler()

	getReq := httptest.NewRequest(http.MethodGet, "/v1/wake-word", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", getRec.Code, getRec.Body.String())
	}
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"active_phrase":"你好小智"`)) ||
		!bytes.Contains(getRec.Body.Bytes(), []byte(`"runtime_status":"active_builtin_model"`)) ||
		!bytes.Contains(getRec.Body.Bytes(), []byte(`"runtime_configurable":false`)) {
		t.Fatalf("default wake word response = %s", getRec.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`))
	putReq.RemoteAddr = "127.0.0.1:12345"
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", putRec.Code, putRec.Body.String())
	}
	for _, want := range []string{
		`"desired_phrase":"小阿二一"`,
		`"desired_pinyin":"xiao a er yi"`,
		`"threshold":35`,
		`"runtime_status":"pending_firmware_build"`,
		`"firmware_build_required":true`,
		`"active_phrase":"你好小智"`,
		`"code":"a21_wake_word_firmware_build_required"`,
	} {
		if !bytes.Contains(putRec.Body.Bytes(), []byte(want)) {
			t.Fatalf("put response missing %q: %s", want, putRec.Body.String())
		}
	}

	reloaded := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	reloadedReq := httptest.NewRequest(http.MethodGet, "/v1/wake-word", nil)
	reloadedRec := httptest.NewRecorder()
	reloaded.Handler().ServeHTTP(reloadedRec, reloadedReq)

	if reloadedRec.Code != http.StatusOK {
		t.Fatalf("reload status = %d, want 200: %s", reloadedRec.Code, reloadedRec.Body.String())
	}
	if !bytes.Contains(reloadedRec.Body.Bytes(), []byte(`"desired_phrase":"小阿二一"`)) ||
		!bytes.Contains(reloadedRec.Body.Bytes(), []byte(`"runtime_status":"pending_firmware_build"`)) {
		t.Fatalf("reloaded wake word response = %s", reloadedRec.Body.String())
	}
}

func TestWakeWordConfigEndpointRejectsUnsafeRequests(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	for _, body := range []string{
		`{"mode":"custom_multinet","desired_phrase":"小阿二一","desired_pinyin":"http://bad","threshold":20}`,
		`{"mode":"custom_multinet","desired_phrase":"token-secret","desired_pinyin":"xiao a er yi","threshold":20}`,
		`{"mode":"custom_multinet","desired_phrase":"x21唤醒","desired_pinyin":"xiao a er yi","threshold":20}`,
		`{"mode":"custom_multinet","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":120}`,
		`{"mode":"custom_multinet","desired_phrase":"小阿二一","desired_pinyin":"","threshold":20}`,
		`{"mode":"unknown","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":20}`,
	} {
		req := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want 400: %s", body, rec.Code, rec.Body.String())
		}
	}
}

func TestWakeWordConfigEndpointRejectsTrailingPayloadWithoutPersisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21-wake-word.json")
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	handler := server.Handler()
	body := `{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}
	{"prompt":"secret trailing wake payload"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"secret trailing wake payload", "prompt", "小阿二一"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("error response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
	reloaded := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	getReq := httptest.NewRequest(http.MethodGet, "/v1/wake-word", nil)
	getRec := httptest.NewRecorder()
	reloaded.Handler().ServeHTTP(getRec, getReq)
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"runtime_status":"active_builtin_model"`)) ||
		bytes.Contains(getRec.Body.Bytes(), []byte(`"desired_phrase":"小阿二一"`)) {
		t.Fatalf("trailing payload should not persist custom wake word: %s", getRec.Body.String())
	}
}

func TestWakeWordConfigEndpointRejectsOversizedPayloadWithoutPersisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21-wake-word.json")
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	handler := server.Handler()
	body := `{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}` + strings.Repeat(" ", 5000)
	req := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	reloaded := NewServerWithOptions(ServerOptions{WakeWordConfigPath: path})
	getReq := httptest.NewRequest(http.MethodGet, "/v1/wake-word", nil)
	getRec := httptest.NewRecorder()
	reloaded.Handler().ServeHTTP(getRec, getReq)
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"runtime_status":"active_builtin_model"`)) ||
		bytes.Contains(getRec.Body.Bytes(), []byte(`"desired_phrase":"小阿二一"`)) {
		t.Fatalf("oversized payload should not persist custom wake word: %s", getRec.Body.String())
	}
}

func TestMockTurnReturnsDeterministicStateSequence(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-000001" {
		t.Fatalf("TraceID = %q", response.TraceID)
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(response.Events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
		if response.Events[i].TraceID != response.TraceID {
			t.Fatalf("event %d trace = %q, want %q", i, response.Events[i].TraceID, response.TraceID)
		}
	}
}

func TestTraceEndpointRecordsMockTurnWaterfall(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-test-001","session_id":"a21-session-test-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-test-001", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-test-001" {
		t.Fatalf("trace id = %q", response.TraceID)
	}
	if len(response.Events) < 4 {
		t.Fatalf("events = %d, want at least 4: %+v", len(response.Events), response.Events)
	}
	wantNames := []string{
		"http.mock_turn.received",
		"control.listening.sent",
		"control.thinking.sent",
		"control.speaking.sent",
	}
	for i, want := range wantNames {
		if response.Events[i].Name != want {
			t.Fatalf("event %d name = %q, want %q", i, response.Events[i].Name, want)
		}
		if response.Events[i].OffsetMS < 0 {
			t.Fatalf("event %d offset = %d, want non-negative", i, response.Events[i].OffsetMS)
		}
	}
}

func TestTraceEndpointReturnsLatencySummary(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "audio.frame.received", 1000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "audio.playback.chunk.sent", 1123)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "v21.query.start", 2000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "v21.query.first_result", 2456)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "barge_in.detected", 3000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "playback.stop", 3033)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "provider.audio.commit", 4000)
	server.recordTrace("a21-trace-summary-001", "a21-session-summary-001", "stackchan-sim-001", "provider.audio.first_downlink", 4088)
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-summary-001", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Summary.EventCount != 8 || response.Summary.LastOffsetMS != 3088 {
		t.Fatalf("summary shape = %+v", response.Summary)
	}
	if response.Summary.AudioFrameToPlaybackMS == nil || *response.Summary.AudioFrameToPlaybackMS != 123 {
		t.Fatalf("audio frame to playback = %v, want 123", response.Summary.AudioFrameToPlaybackMS)
	}
	if response.Summary.V21QueryFirstResultMS == nil || *response.Summary.V21QueryFirstResultMS != 456 {
		t.Fatalf("v21 first result = %v, want 456", response.Summary.V21QueryFirstResultMS)
	}
	if response.Summary.BargeInStopMS == nil || *response.Summary.BargeInStopMS != 33 {
		t.Fatalf("barge-in stop = %v, want 33", response.Summary.BargeInStopMS)
	}
	if response.Summary.ProviderCommitToFirstAudioMS == nil || *response.Summary.ProviderCommitToFirstAudioMS != 88 {
		t.Fatalf("provider commit to first audio = %v, want 88", response.Summary.ProviderCommitToFirstAudioMS)
	}
}

func TestTraceEndpointReturnsVoicePipelineSplitSummary(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.listen.start", 1000)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.opus_frame.received", 1010)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "xiaozhi.opus_frame.decoded", 1024)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "audio.ingress.buffered", 1030)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "asr.first_partial", 1140)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "provider.first_content", 1300)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "tts.first_audio", 1375)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "audio.downlink.first_frame", 1400)
	server.recordTrace("a21-trace-pipeline-001", "a21-session-pipeline-001", "stackchan-sim-001", "device.playback.start", 1460)

	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pipeline-001", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	assertSummaryDelta(t, "xiaozhi_listen_to_audio_ingress_ms", response.Summary.XiaozhiListenToAudioIngressMS, 30)
	assertSummaryDelta(t, "xiaozhi_opus_decode_ms", response.Summary.XiaozhiOpusDecodeMS, 14)
	assertSummaryDelta(t, "asr_first_partial_ms", response.Summary.ASRFirstPartialMS, 110)
	assertSummaryDelta(t, "llm_first_content_ms", response.Summary.LLMFirstContentMS, 160)
	assertSummaryDelta(t, "tts_first_audio_ms", response.Summary.TTSFirstAudioMS, 75)
	assertSummaryDelta(t, "audio_downlink_first_frame_ms", response.Summary.AudioDownlinkFirstFrameMS, 25)
	assertSummaryDelta(t, "device_playback_start_ms", response.Summary.DevicePlaybackStartMS, 60)
	assertSummaryDelta(t, "answer_first_audio_total_ms", response.Summary.AnswerFirstAudioTotalMS, 370)
}

func TestTraceEndpointUsesASRFinalWhenPartialIsUnavailable(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-pipeline-final-001", "a21-session-pipeline-final-001", "stackchan-sim-001", "audio.ingress.buffered", 2000)
	server.recordTrace("a21-trace-pipeline-final-001", "a21-session-pipeline-final-001", "stackchan-sim-001", "asr.final", 2550)
	server.recordTrace("a21-trace-pipeline-final-001", "a21-session-pipeline-final-001", "stackchan-sim-001", "provider.first_content", 2830)
	server.recordTrace("a21-trace-pipeline-final-001", "a21-session-pipeline-final-001", "stackchan-sim-001", "tts.first_audio", 3240)
	server.recordTrace("a21-trace-pipeline-final-001", "a21-session-pipeline-final-001", "stackchan-sim-001", "audio.downlink.first_frame", 3640)

	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pipeline-final-001", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	assertSummaryDelta(t, "asr_first_partial_ms", response.Summary.ASRFirstPartialMS, 550)
	assertSummaryDelta(t, "llm_first_content_ms", response.Summary.LLMFirstContentMS, 280)
	assertSummaryDelta(t, "tts_first_audio_ms", response.Summary.TTSFirstAudioMS, 410)
	assertSummaryDelta(t, "audio_downlink_first_frame_ms", response.Summary.AudioDownlinkFirstFrameMS, 400)
	assertSummaryDelta(t, "answer_first_audio_total_ms", response.Summary.AnswerFirstAudioTotalMS, 1640)
}

func TestTraceEndpointUsesLatestCompletePairForReusedHardwareTrace(t *testing.T) {
	server := NewServer()
	server.recordTrace("a21-trace-reused-001", "s1", "stackchan-001", "audio.ingress.buffered", 1000)
	server.recordTrace("a21-trace-reused-001", "s1", "stackchan-001", "asr.final", 100000)
	server.recordTrace("a21-trace-reused-001", "s2", "stackchan-001", "audio.ingress.buffered", 110000)
	server.recordTrace("a21-trace-reused-001", "s2", "stackchan-001", "asr.final", 110550)
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-reused-001", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response TraceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	assertSummaryDelta(t, "asr_first_partial_ms", response.Summary.ASRFirstPartialMS, 550)
}

func TestTraceEndpointRequiresTraceID(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/traces", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMockTurnUsesVoiceProviderEvents(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{Kind: providers.VoiceEventThinking, Text: "provider thinking"},
				{Kind: providers.VoiceEventSpeaking, Text: "provider speaking", Final: true, StreamID: "provider-stream"},
			},
		},
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"custom","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.Text != "provider speaking" {
		t.Fatalf("speaking text = %q, want provider speaking", speaking.Text)
	}
	if speaking.StreamID != "provider-stream" {
		t.Fatalf("stream = %q, want provider-stream", speaking.StreamID)
	}
}

func TestFastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary(t *testing.T) {
	t.Setenv("A21_MEMORY_USER_PREFERENCES", "角色记住我喜欢短句")
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "selected provider should not run", Final: true},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	handler := server.Handler()
	profileReq := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{"scenario":"desk_mouthpiece","voice_clone_profile":"a21_voice_clone_default"}`))
	profileRec := httptest.NewRecorder()
	handler.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK {
		t.Fatalf("roleplay profile status = %d: %s", profileRec.Code, profileRec.Body.String())
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(`{
		"device_id":"stackchan-sim-001",
		"mode":"roleplay",
		"trace_id":"a21-trace-fast-hybrid-001",
		"session_id":"a21-session-fast-hybrid-001",
		"local_audio":{"asr_provider":"mock_asr","first_partial_ms":42,"final_transcript_chars":11}
	}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		TraceID            string                 `json:"trace_id"`
		SessionID          string                 `json:"session_id"`
		DeviceID           string                 `json:"device_id"`
		Mode               protocol.Mode          `json:"mode"`
		Status             string                 `json:"status"`
		Route              string                 `json:"route"`
		AudioFrontend      string                 `json:"audio_frontend"`
		TextStreamProvider string                 `json:"text_stream_provider"`
		ProviderFamily     string                 `json:"provider_family"`
		TextStreamExecuted bool                   `json:"text_stream_executed"`
		Roleplay           RoleplayRuntimeSummary `json:"roleplay"`
		Events             []protocol.Envelope    `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.TraceID != "a21-trace-fast-hybrid-001" || response.SessionID != "a21-session-fast-hybrid-001" || response.DeviceID != "stackchan-sim-001" {
		t.Fatalf("response identity = %+v", response)
	}
	if response.Mode != protocol.ModeRoleplay || response.Status != "boundary_ready" || response.Route != "fast_companion_hybrid" {
		t.Fatalf("response route = %+v", response)
	}
	if response.AudioFrontend != "local_audio" || response.ProviderFamily != "text_stream" || response.TextStreamProvider != "mock_text_stream" || response.TextStreamExecuted {
		t.Fatalf("provider boundary = %+v", response)
	}
	if response.Roleplay.RoleplayProfile != "a21_roleplay_default" ||
		response.Roleplay.Scenario != "desk_mouthpiece" ||
		response.Roleplay.VoiceCloneProfile != "a21_voice_clone_default" ||
		response.Roleplay.MemoryHintCount != 1 ||
		!response.Roleplay.PromptComposed ||
		response.Roleplay.PromptStored ||
		response.Roleplay.MemoryTextStored ||
		response.Roleplay.TranscriptStored ||
		response.Roleplay.ProviderOutputStored ||
		response.Roleplay.VoiceCloneSampleStored ||
		response.Roleplay.ProfessionalRouteAllowed ||
		response.Roleplay.V21Executed {
		t.Fatalf("roleplay runtime = %+v", response.Roleplay)
	}
	if strings.Contains(rec.Body.String(), "角色记住我喜欢短句") {
		t.Fatalf("fast companion response leaked memory text: %s", rec.Body.String())
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(response.Events))
	}
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.Mode != protocol.ModeRoleplay || speaking.StreamID != "a21-fast-companion-placeholder-stream" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	if provider.startCalls != 0 {
		t.Fatalf("voice provider start calls = %d, want 0", provider.startCalls)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-hybrid-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var traceResponse TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traceResponse); err != nil {
		t.Fatal(err)
	}
	foundASRFirstPartial := false
	for _, event := range traceResponse.Events {
		if event.Name == "asr.first_partial" {
			foundASRFirstPartial = true
			if event.OffsetMS != 42 {
				t.Fatalf("asr first partial offset = %d, want 42", event.OffsetMS)
			}
		}
	}
	if !foundASRFirstPartial {
		t.Fatalf("trace missing asr.first_partial event: %s", traceRec.Body.String())
	}
	for _, want := range []string{
		"roleplay.profile.ready",
		"roleplay.memory.ready",
		"fast_companion.local_audio.frontend.accepted",
		"asr.first_partial",
		"provider.text_stream.route.placeholder",
		"provider.first_byte",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"device.playback.start",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	if strings.Contains(traceRec.Body.String(), "角色记住我喜欢短句") {
		t.Fatalf("trace leaked memory text: %s", traceRec.Body.String())
	}
}

func TestFastCompanionHybridRunsVoicePipelineWhenFramesProvided(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "selected provider should not run", Final: true},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	runner := newRecordingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	handler := server.Handler()
	profileReq := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{"roleplay_profile":"a21_roleplay_wry_peer","voice_clone_profile":"a21_voice_clone_default","memory_hints":["角色语气只给短句"]}`))
	profileRec := httptest.NewRecorder()
	handler.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK {
		t.Fatalf("roleplay profile status = %d: %s", profileRec.Code, profileRec.Body.String())
	}
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"roleplay",
		"trace_id":"a21-trace-fast-pipeline-001",
		"session_id":"a21-session-fast-pipeline-001",
		"local_audio":{
			"asr_provider":"mock_asr",
			"first_partial_ms":42,
			"final_transcript_chars":11,
			"frames":[{"seq":1,"codec":"pcm_s16le","sample_rate_hz":16000,"channels":1,"duration_ms":60,"byte_count":1920,"rms":0.04,"data_base64":%q}]
		}
	}`, frameBase64)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"fixture transcript should never be stored", "mock provider output should never be stored", "角色语气只给短句", frameBase64} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("fast companion response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
	var response struct {
		Status             string              `json:"status"`
		TextStreamProvider string              `json:"text_stream_provider"`
		ProviderFamily     string              `json:"provider_family"`
		TextStreamExecuted bool                `json:"text_stream_executed"`
		Events             []protocol.Envelope `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "pipeline_completed" || response.ProviderFamily != "text_stream" || response.TextStreamProvider != "mock" || !response.TextStreamExecuted {
		t.Fatalf("response pipeline status = %+v", response)
	}
	var audioChunks int
	var speaking protocol.ControlEventPayload
	for _, event := range response.Events {
		switch event.Kind {
		case protocol.KindAudioPlaybackChunk:
			audioChunks++
		case protocol.KindControlEvent:
			var payload protocol.ControlEventPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload.State == protocol.ExpressionSpeaking {
				speaking = payload
			}
		}
	}
	if audioChunks == 0 {
		t.Fatalf("events = %#v, want at least one audio playback chunk", response.Events)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "a21-fast-companion-voice-pipeline" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	var captured providers.VoicePipelineRequest
	select {
	case captured = <-runner.requests:
	case <-time.After(time.Second):
		t.Fatal("voice pipeline runner did not capture request")
	}
	if !strings.Contains(captured.TextPrompt, "Role Soul: Wry Peer") ||
		strings.Contains(captured.TextPrompt, "Role Soul: Calm Anchor") ||
		!strings.Contains(captured.TextPrompt, "Memory Hints") ||
		!strings.Contains(captured.TextPrompt, "session_memory:session_memory_1") ||
		!strings.Contains(captured.TextPrompt, "角色语气只给短句") {
		t.Fatalf("voice pipeline request missing redacted roleplay prompt input")
	}
	if captured.VoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("captured voice clone profile = %q, want selected clone", captured.VoiceCloneProfile)
	}
	if provider.startCalls != 0 {
		t.Fatalf("voice provider start calls = %d, want 0", provider.startCalls)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-pipeline-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	for _, want := range []string{
		"fast_companion.voice_pipeline.start",
		"provider.first_content",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"device.playback.start",
		"roleplay.prompt_input.used",
		"roleplay.voice_clone_profile.used",
		"fast_companion.voice_pipeline.completed",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	if strings.Contains(traceRec.Body.String(), "角色语气只给短句") {
		t.Fatalf("trace leaked roleplay prompt hint")
	}
}
