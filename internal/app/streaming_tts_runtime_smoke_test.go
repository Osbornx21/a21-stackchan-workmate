package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/providers"
)

func TestRunStreamingTTSRuntimeSmokeBlocksWithoutExecuteAndWritesRedactedReport(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts-secret")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "voice-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"streaming-tts-runtime-smoke", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	report := readStreamingTTSRuntimeSmokeReportFromDir(t, dir)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !appStringSliceContains(report.Findings, "execute_flag_required") {
		t.Fatalf("findings = %#v, want execute_flag_required", report.Findings)
	}
	if report.Executed || report.ProviderConfigured {
		t.Fatalf("report executed/configured = %v/%v, want false/false", report.Executed, report.ProviderConfigured)
	}
	rendered := stdout.String() + mustJSONForAppTest(t, report)
	for _, forbidden := range []string{
		"sk-a21-secret",
		"doubao-tts-secret",
		"voice-secret",
		"Authorization",
		"Bearer",
		"data_base64",
		"audio_base64",
		"raw_audio",
		"http://",
		"https://",
		"wss://",
		"/Users/",
		".wav",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("streaming TTS no-exec report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRunStreamingTTSRuntimeSmokeBlocksExecuteWhenEnvMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("A21_TTS_FAST_PROFILE", "")
	t.Setenv("A21_DOUBAO_API_KEY", "")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"streaming-tts-runtime-smoke", "--execute", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	report := readStreamingTTSRuntimeSmokeReportFromDir(t, dir)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	for _, want := range []string{
		"A21_TTS_FAST_PROFILE",
		"A21_DOUBAO_API_KEY",
		"A21_DOUBAO_TTS_MODEL",
		"A21_DOUBAO_TTS_VOICE",
	} {
		if !appStringSliceContains(report.MissingEnv, want) {
			t.Fatalf("missing_env = %#v, want %s", report.MissingEnv, want)
		}
	}
	if !appStringSliceContains(report.Findings, "missing_required_env") {
		t.Fatalf("findings = %#v, want missing_required_env", report.Findings)
	}
	rendered := stdout.String() + mustJSONForAppTest(t, report)
	for _, forbidden := range []string{"secret", "Authorization", "Bearer", "http://", "wss://", "proxy", "/Users/"} {
		if strings.Contains(strings.ToLower(rendered), strings.ToLower(forbidden)) {
			t.Fatalf("streaming TTS missing-env report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRunStreamingTTSRuntimeSmokeFakeRuntimeProvesFirstChunkBeforeEOF(t *testing.T) {
	dir := t.TempDir()
	deltaPCM := make([]byte, 24000*60/1000*2)
	for i := range deltaPCM {
		deltaPCM[i] = byte(i % 251)
	}
	delta := base64.StdEncoding.EncodeToString(deltaPCM)
	conn := &appFakeRealtimeConn{
		serverMessages: []map[string]any{{
			"type":    "response.audio.delta",
			"item_id": "a21-runtime-smoke-item",
			"delta":   delta,
		}},
	}
	previousDialer := streamingTTSRuntimeSmokeDialer
	streamingTTSRuntimeSmokeDialer = appFakeRealtimeDialer{conn: conn}
	t.Cleanup(func() { streamingTTSRuntimeSmokeDialer = previousDialer })
	t.Setenv("A21_TTS_FAST_PROFILE", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts-secret")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "voice-secret")
	t.Setenv("A21_DOUBAO_TTS_SAMPLE_RATE_HZ", "24000")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"streaming-tts-runtime-smoke", "--execute", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	report := readStreamingTTSRuntimeSmokeReportFromDir(t, dir)
	if report.Status != "passed" {
		t.Fatalf("status = %q, want passed: %#v", report.Status, report.Findings)
	}
	if !report.FirstProviderAudioDeltaObserved || !report.FirstAudioDeltaBeforeEOF {
		t.Fatalf("delta/eof flags = observed:%v before_eof:%v eof:%v", report.FirstProviderAudioDeltaObserved, report.FirstAudioDeltaBeforeEOF, report.ProviderEOFObserved)
	}
	if report.Exact60MSPCM16MonoChunks != 1 || report.SampleRateHz != 24000 || report.Channels != 1 || report.TargetChunkDurationMS != 60 {
		t.Fatalf("chunk proof = chunks:%d rate:%d channels:%d duration:%d", report.Exact60MSPCM16MonoChunks, report.SampleRateHz, report.Channels, report.TargetChunkDurationMS)
	}
	if !report.SessionUpdateSent || !report.TextAppendSent || !report.TextDoneSent || !conn.closed {
		t.Fatalf("session state update/text/done/closed = %v/%v/%v/%v", report.SessionUpdateSent, report.TextAppendSent, report.TextDoneSent, conn.closed)
	}
	gotTypes := appRealtimeEventTypes(conn.messages)
	wantTypes := []string{"tts_session.update", "input_text.append", "input_text.done"}
	if strings.Join(gotTypes, ",") != strings.Join(wantTypes, ",") {
		t.Fatalf("event types = %#v, want %#v", gotTypes, wantTypes)
	}
	rendered := stdout.String() + mustJSONForAppTest(t, report)
	for _, forbidden := range []string{
		"sk-a21-secret",
		"doubao-tts-secret",
		"voice-secret",
		delta,
		"Authorization",
		"Bearer",
		"data_base64",
		"audio_base64",
		"raw_audio",
		"http://",
		"https://",
		"wss://",
		"/Users/",
		".wav",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("streaming TTS runtime report leaked %q: %s", forbidden, rendered)
		}
	}
}

func TestRunStreamingTTSRuntimeSmokeFailureFindingIsRedacted(t *testing.T) {
	dir := t.TempDir()
	previousDialer := streamingTTSRuntimeSmokeDialer
	streamingTTSRuntimeSmokeDialer = appFailingRealtimeDialer{
		err: fmt.Errorf("failed to dial wss://example.invalid/v1/realtime?token=secret-value: Authorization Bearer sk-a21-secret rejected"),
	}
	t.Cleanup(func() { streamingTTSRuntimeSmokeDialer = previousDialer })
	t.Setenv("A21_TTS_FAST_PROFILE", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_ACCESS_TOKEN", "access-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts-secret")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "voice-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"streaming-tts-runtime-smoke", "--execute", "--output-dir", dir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1: stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	report := readStreamingTTSRuntimeSmokeReportFromDir(t, dir)
	if report.Status != "failed" || !report.Executed || !report.ProviderConfigured {
		t.Fatalf("status/executed/configured = %q/%v/%v", report.Status, report.Executed, report.ProviderConfigured)
	}
	if !appStringSliceContains(report.Findings, "provider_runtime_failed_redacted") {
		t.Fatalf("findings = %#v, want redacted runtime failure", report.Findings)
	}
	rendered := stdout.String() + mustJSONForAppTest(t, report)
	for _, forbidden := range []string{"access-a21-secret", "sk-a21-secret", "secret-value", "doubao-tts-secret", "voice-secret", "Authorization", "Bearer", "wss://", "https://", "token="} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("streaming TTS failure report leaked %q: %s", forbidden, rendered)
		}
	}
}

func readStreamingTTSRuntimeSmokeReportFromDir(t *testing.T, dir string) streamingTTSRuntimeSmokeReport {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "a21-streaming-tts-runtime-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var report streamingTTSRuntimeSmokeReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, string(data))
	}
	return report
}

func mustJSONForAppTest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

type appFakeRealtimeConn struct {
	messages       []map[string]any
	serverMessages []map[string]any
	readIndex      int
	closed         bool
}

func (c *appFakeRealtimeConn) WriteJSON(_ context.Context, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var message map[string]any
	if err := json.Unmarshal(encoded, &message); err != nil {
		return err
	}
	c.messages = append(c.messages, message)
	return nil
}

func (c *appFakeRealtimeConn) ReadJSON(_ context.Context, value any) error {
	if c.readIndex >= len(c.serverMessages) {
		return io.EOF
	}
	encoded, err := json.Marshal(c.serverMessages[c.readIndex])
	if err != nil {
		return err
	}
	c.readIndex++
	return json.Unmarshal(encoded, value)
}

func (c *appFakeRealtimeConn) Close(_ context.Context) error {
	c.closed = true
	return nil
}

type appFakeRealtimeDialer struct {
	conn providers.RealtimeConn
}

func (d appFakeRealtimeDialer) Dial(_ context.Context, _ string, _ http.Header, _ providers.NetworkPolicy) (providers.RealtimeConn, error) {
	return d.conn, nil
}

type appFailingRealtimeDialer struct {
	err error
}

func (d appFailingRealtimeDialer) Dial(_ context.Context, _ string, _ http.Header, _ providers.NetworkPolicy) (providers.RealtimeConn, error) {
	return nil, d.err
}

func appRealtimeEventTypes(messages []map[string]any) []string {
	types := make([]string, 0, len(messages))
	for _, message := range messages {
		if eventType, ok := message["type"].(string); ok {
			types = append(types, eventType)
		}
	}
	return types
}

func appStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
