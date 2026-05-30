package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/providers"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := stdout.String(); got != "A21 0.1.0-dev (a21-core)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"nope"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunNamespaceAuditReadsTrackedFiles(t *testing.T) {
	dir := t.TempDir()
	writeNamespaceAuditGitScript(t, dir, "cmd/a21/main.go\ninternal/v21adapter/client.go\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"namespace-audit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"ok": true`,
		`"files_scanned": 2`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunNamespaceAuditRejectsLegacyPathWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	writeNamespaceAuditGitScript(t, dir, "cmd/a21/main.go\napps/x21-gateway/main.go\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"namespace-audit"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"namespace_legacy_path"`) {
		t.Fatalf("stdout missing namespace finding: %s", stdout.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21-gateway") || strings.Contains(strings.ToLower(stderr.String()), "x21-gateway") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestGatewayServerFromEnvDefaultsToMockDespiteSelectedPrimary(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"a21-mock-voice"`) {
		t.Fatalf("health = %s, want mock provider", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sk-a21-secret") || strings.Contains(rec.Body.String(), "doubao-tts") {
		t.Fatalf("health leaked provider config: %s", rec.Body.String())
	}
}

func TestGatewayServerFromEnvUsesSelectedProviderOnlyWhenExplicit(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"a21-doubao-realtime-tts"`) {
		t.Fatalf("health = %s, want selected Doubao provider", rec.Body.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerFromEnvUsesSelectedDoubaoRealtimeAsDegradedBoundary(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"provider":"a21-doubao-realtime-voice"`,
		`"status":"degraded"`,
		`"configured":true`,
		`"detail":"execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("health missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerSelectedRealtimeProviderKeepsTextTurnGuarded(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_tts_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_TTS_MODEL=doubao-tts",
		"A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts",
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"普通文本轮不应该误拨 provider","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "StartRealtimeTTSSession") {
		t.Fatalf("response missing guarded realtime-session guidance: %s", rec.Body.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerSelectedDoubaoRealtimeRejectsRealtimeSessionWithoutSecrets(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_GATEWAY_VOICE_PROVIDER=selected",
		"A21_PROVIDER_PRIMARY=doubao_realtime",
		"A21_DOUBAO_API_KEY=sk-a21-secret",
		"A21_DOUBAO_APP_ID=app-a21-secret",
		"A21_DOUBAO_RESOURCE_ID=resource-a21-secret",
		"A21_DOUBAO_REALTIME_MODEL=doubao-s2s",
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先不要真的连接豆包","mode":"workmate","trace_id":"a21-trace-doubao-guard","session_id":"a21-session-doubao-guard"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"provider":"a21-doubao-realtime-voice"`,
		`"status":"error"`,
		`"trace_id":"a21-trace-doubao-guard"`,
		`"text":"Doubao realtime speech-to-speech execution is disabled until the A21 adapter has verified official API shape, credentialed smoke, cancellation, and latency behavior"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("response missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestRunSerialListEmitsSerialDevices(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{
			{Path: "/dev/cu.usbmodemA21", USBModem: true, Usage: firmwarecheck.PortUsage{Exists: true}},
		}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"serial-list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"serial_devices"`, "/dev/cu.usbmodemA21", `"usb_modem": true`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorEmitsReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	if !strings.Contains(stdout.String(), `"result"`) {
		t.Fatalf("stdout missing result report: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"fingerprint"`) {
		t.Fatalf("stdout missing fingerprint report: %q", stdout.String())
	}
}

func TestRunDoctorWritesReportFile(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-doctor-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1", len(matches))
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"fingerprint"`) {
		t.Fatalf("report file missing fingerprint: %s", data)
	}
}

func TestRunDoctorIncludesFirmwareSection(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	if !strings.Contains(stdout.String(), `"firmware"`) {
		t.Fatalf("stdout missing firmware section: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"serial_devices"`) {
		t.Fatalf("stdout missing serial devices: %q", stdout.String())
	}
}

func TestRunDoctorIncludesProxyPolicy(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"proxy"`, `"global_proxy_configured"`, `"direct_connect_ok"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorIncludesVoiceProviderHealth(t *testing.T) {
	originalProbe := probeVoiceProviderHealth
	probeVoiceProviderHealth = func(ctx context.Context) (providers.VoiceProviderHealth, error) {
		return providers.VoiceProviderHealth{
			Provider:   "a21-test-voice",
			Status:     providers.VoiceProviderHealthy,
			Configured: true,
			Realtime:   true,
		}, nil
	}
	defer func() {
		probeVoiceProviderHealth = originalProbe
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"voice"`, `"provider": "a21-test-voice"`, `"status": "healthy"`, `"healthy": true`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProviderNetworkPolicy(t *testing.T) {
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@127.0.0.1:7891")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"network"`, `"mode": "explicit_proxy"`, `"proxy_configured": true`, `"provider_proxy_env"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"provider-secret", "7891"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked provider proxy value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProviderCatalogWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "openai_realtime")
	t.Setenv("A21_OPENAI_API_KEY", "sk-a21-secret")
	t.Setenv("A21_OPENAI_REALTIME_MODEL", "gpt-realtime")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"providers"`, `"primary": "openai_realtime"`, `"name": "openai_realtime"`, `"configured": true`, `"A21_OPENAI_API_KEY"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "sk-a21-secret") {
		t.Fatalf("stdout leaked provider key: %s", stdout.String())
	}
}

func TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-tts"`,
		`"gateway_provider": "a21-mock-voice"`,
		`"status": "healthy"`,
		`"configured": true`,
		`"primary": "doubao_tts_realtime"`,
		`"provider": "doubao_tts_realtime"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"provider": "a21-mock-voice"`) {
		t.Fatalf("doctor voice health still reports mock provider: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorVoiceHealthReportsDoubaoRealtimeDegradedWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-voice"`,
		`"gateway_provider": "a21-mock-voice"`,
		`"status": "degraded"`,
		`"healthy": false`,
		`"configured": true`,
		`"detail": "execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
		`"primary": "doubao_realtime"`,
		`"provider": "doubao_realtime"`,
		`"status": "unsupported"`,
		`"status": "ready"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime(t *testing.T) {
	t.Setenv("A21_GATEWAY_VOICE_PROVIDER", "selected")
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"gateway_provider": "a21-doubao-realtime-tts"`) {
		t.Fatalf("stdout missing explicit gateway provider: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorReportsExplicitGatewayDoubaoRealtimeRuntimeAsDegraded(t *testing.T) {
	t.Setenv("A21_GATEWAY_VOICE_PROVIDER", "selected")
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21-doubao-realtime-voice"`,
		`"gateway_provider": "a21-doubao-realtime-voice"`,
		`"status": "degraded"`,
		`"detail": "execution disabled pending verified Doubao realtime speech-to-speech adapter"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesRealtimePlanWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "openai_realtime")
	t.Setenv("A21_OPENAI_API_KEY", "sk-a21-secret")
	t.Setenv("A21_OPENAI_REALTIME_MODEL", "gpt-realtime-2")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"realtime_plan"`,
		`"provider": "openai_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"endpoint_host": "api.openai.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "gpt-realtime-2", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorBlocksLegacyProviderPrimaryWithoutEchoingValue(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "x21_voice")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider_legacy_identity"`) {
		t.Fatalf("stdout missing provider legacy finding: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_voice") {
		t.Fatalf("stdout leaked legacy provider value: %s", stdout.String())
	}
}

func TestRunProviderSmokeDryRunDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_DEEPSEEK_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DEEPSEEK_MODEL", "deepseek-v4-flash")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", "deepseek"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"provider": "deepseek"`, `"status": "ready"`, `"executed": false`, `"api_key_env": "A21_DEEPSEEK_API_KEY"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-v4-flash"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderSmokeWritesRedactedReportWhenOutputDirProvided(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_DEEPSEEK_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DEEPSEEK_MODEL", "deepseek-v4-flash")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-smoke", "--provider", "deepseek", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-provider-smoke-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("provider smoke reports = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"provider": "deepseek"`,
		`"status": "ready"`,
		`"executed": false`,
		`"report_path"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("provider smoke report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-v4-flash"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("provider smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
		}
	}
}

func TestRunProviderSmokeRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", "x21_voice"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider": "invalid_legacy_provider"`) {
		t.Fatalf("stdout missing redacted provider: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_voice") || strings.Contains(stderr.String(), "x21_voice") {
		t.Fatalf("legacy provider leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
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
		`"adapter": "a21-v21-adapter"`,
		`"status": "passed"`,
		`"executed": true`,
		`"query_path": "/a21/v21/query"`,
		`"evidence_count": 1`,
		`"speech_block_count": 1`,
		`"screen_card_count": 1`,
		`"report_path"`,
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
	for _, forbidden := range []string{"语音唤醒", "历史讨论", server.URL} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(reportJSON, forbidden) {
			t.Fatalf("v21 adapter smoke leaked %q: stdout=%s report=%s", forbidden, stdout.String(), reportJSON)
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

func TestRunLANProbeRejectsCredentialTargetWithoutEchoingSecret(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=http://user:secret-token@127.0.0.1:21080",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "must not include credentials") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, forbidden := range []string{"secret-token", "user:", "127.0.0.1:21080"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunLANProbeRejectsLegacyInternalPortWithoutEchoingTarget(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"lan-probe",
		"--target", "a21-gateway=127.0.0.1:18080",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "legacy internal port") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, forbidden := range []string{"a21-gateway", "127.0.0.1", "18080"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("lan probe leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanDoubaoTTSDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "doubao_tts_realtime"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_tts_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
		`"api_key_env": "A21_DOUBAO_API_KEY"`,
		`"model_env": "A21_DOUBAO_TTS_MODEL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie_moon_bigtts", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider realtime plan leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanDoubaoS2SDoesNotLeakSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_APP_ID", "app-a21-secret")
	t.Setenv("A21_DOUBAO_RESOURCE_ID", "resource-a21-secret")
	t.Setenv("A21_DOUBAO_REALTIME_MODEL", "doubao-s2s")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "doubao_realtime"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_realtime"`,
		`"protocol": "websocket_realtime"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
		`"api_key_env": "A21_DOUBAO_API_KEY"`,
		`"model_env": "A21_DOUBAO_REALTIME_MODEL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider realtime plan leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimePlanRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--provider", "x21_voice"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider": "invalid_legacy_provider"`) {
		t.Fatalf("stdout missing redacted provider: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_voice") || strings.Contains(stderr.String(), "x21_voice") {
		t.Fatalf("legacy provider leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunProviderRealtimePlanRejectsExecuteFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-plan", "--execute"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "does not support --execute") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunAudioFrontEndPlanListsMatureCandidates(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-plan"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "plan_only"`,
		`"id": "webrtc_apm"`,
		`"id": "provider_side_vad"`,
		`"id": "silero_vad"`,
		`"id": "a21_rms_vad"`,
		`"status": "dev_only"`,
		`"a21_vad_detector_decisions_total{detector,result}"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"x21", "v21"} {
		if strings.Contains(strings.ToLower(stdout.String()), forbidden) {
			t.Fatalf("audio front-end plan leaked forbidden identity %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalMockReportsQualityMetrics(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "mock_only"`,
		`"dataset": "a21_mock_vad_fixture_v1"`,
		`"detector": "a21-rms-vad"`,
		`"frames_total": 5`,
		`"true_positive": 2`,
		`"true_negative": 3`,
		`"false_positive": 0`,
		`"false_negative": 0`,
		`"speech_start_lag_ms": 0`,
		`"speech_end_lag_ms": 20`,
		`"promotion_gate": "not_production"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalRequiresMockFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "requires --mock") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunAudioFrontEndEvalFixtureReportsQualityMetrics(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "a21-front-end-fixture.json")
	data := `{
  "schema_version": "a21.audio.frontend_fixture.v1",
  "dataset": "a21_cli_fixture_test",
  "detector": "a21-rms-vad",
  "sample_rate_hz": 16000,
  "channels": 1,
  "duration_ms": 20,
  "frames": [
    {"seq": 1, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"},
    {"seq": 2, "expected_speech": true, "pcm_s16le_base64": "` + appTestPCM16Base64(12000) + `"},
    {"seq": 3, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"},
    {"seq": 4, "expected_speech": false, "pcm_s16le_base64": "` + appTestPCM16Base64(0) + `"}
  ]
}`
	if err := os.WriteFile(fixture, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--fixture", fixture}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"status": "fixture"`,
		`"dataset": "a21_cli_fixture_test"`,
		`"frames_total": 4`,
		`"true_positive": 1`,
		`"true_negative": 3`,
		`"speech_start_lag_ms": 0`,
		`"speech_end_lag_ms": 20`,
		`"promotion_gate": "requires_recorded_office_and_physical_acceptance"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunAudioFrontEndEvalWritesReportArtifact(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-audio-front-end-eval-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1", len(matches))
	}
	if !strings.Contains(stdout.String(), `"report_path":`) || !strings.Contains(stdout.String(), filepath.Base(matches[0])) {
		t.Fatalf("stdout missing report path: %s", stdout.String())
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"schema_version": "a21.audio.frontend_eval.v1"`, `"status": "mock_only"`, `"speech_start_lag_ms": 0`, `"speech_end_lag_ms": 20`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report file missing %q: %s", want, data)
		}
	}
	for _, forbidden := range []string{"pcm_s16le_base64", "x21", "v21"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("report file leaked %q: %s", forbidden, data)
		}
	}
}

func TestRunAudioFrontEndEvalReportIncludesTraceableEnvironmentMetadataWithoutProxySecrets(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"metadata"`,
		`"generated_at"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"HTTPS_PROXY"`,
		`"provider_proxy_env"`,
		`"A21_PROVIDER_PROXY_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "pcm_s16le_base64"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-audio-front-end-eval-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"metadata"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"provider_proxy_env"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000", "pcm_s16le_base64"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunAudioFrontEndEvalRejectsLegacyReportDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"audio-front-end-eval", "--mock", "--output-dir", filepath.Join("reports", "x21-audio")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21") {
		t.Fatalf("legacy report path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceReportWritesA21Report(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "connection_status": "online",
      "device_age_ms": 42,
      "current_mode": "workmate",
      "current_expression": "speaking",
      "last_seen_ms": 1780000000000
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.device_report.v1"`,
		`"gateway_schema_version": "a21.gateway.devices.v1"`,
		`"gateway_service": "a21-gateway"`,
		`"device_report_path":`,
		`"gateway_url":`,
		`"device_id": "stackchan-001"`,
		`"capabilities":`,
		`"screen_touch": "available"`,
		`"servo_y": "available"`,
		`"connection_status": "online"`,
		`"device_age_ms": 42`,
		`"current_mode": "workmate"`,
		`"current_expression": "speaking"`,
		`"last_seen_ms": 1780000000000`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("device report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"devices"`)) {
		t.Fatalf("report missing devices: %s", string(data))
	}
	for _, forbidden := range []string{"x21", "v21"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("report contains forbidden identity %q: %s", forbidden, string(data))
		}
	}
}

func TestRunFirmwareDeviceReportRejectsGatewayWithoutA21Identity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "A21 Gateway identity") {
		t.Fatalf("stderr = %q, want gateway identity rejection", stderr.String())
	}
	if strings.Contains(stdout.String(), "stackchan-001") {
		t.Fatalf("stdout should not contain accepted device report: %s", stdout.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayServiceWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "x21-gateway",
  "devices": []
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy gateway identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-gateway") {
		t.Fatalf("legacy gateway identity leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyOutputDirWithoutEchoingPath(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://127.0.0.1:21080",
		"--output-dir", filepath.Join(t.TempDir(), "x21-reports"),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want forbidden legacy identity", stderr.String())
	}
	if strings.Contains(strings.ToLower(stderr.String()), "x21-reports") {
		t.Fatalf("stderr should not echo legacy path: %q", stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsGatewayCredentialsWithoutEchoingSecret(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://user:secret@127.0.0.1:21080",
		"--output-dir", t.TempDir(),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "must not include credentials") {
		t.Fatalf("stderr = %q, want credential rejection", stderr.String())
	}
	if strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), "user:") {
		t.Fatalf("stderr leaked gateway credentials: %q", stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayPortWithoutDialingOrEchoingIt(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", "http://127.0.0.1:8080",
		"--output-dir", t.TempDir(),
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "legacy internal port") {
		t.Fatalf("stderr = %q, want legacy internal port rejection", stderr.String())
	}
	for _, forbidden := range []string{"127.0.0.1", "8080"} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("stderr leaked %q: %s", forbidden, stderr.String())
		}
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyGatewayDeviceWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "x21-stackchan",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-stackchan") {
		t.Fatalf("legacy identity leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceReportRejectsLegacyCapabilityWithoutEchoingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "screen": "x21-compatible"
      }
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-report",
		"--gateway-url", server.URL,
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy identity rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-compatible") {
		t.Fatalf("legacy capability leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunProviderRealtimeFixtureExecutesDoubaoTTSWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "doubao_tts_realtime")
	t.Setenv("A21_DOUBAO_API_KEY", "sk-a21-secret")
	t.Setenv("A21_DOUBAO_TTS_MODEL", "doubao-tts")
	t.Setenv("A21_DOUBAO_TTS_VOICE", "zh_female_kailangjiejie_moon_bigtts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-fixture", "--provider", "doubao_tts_realtime", "--execute"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "doubao_tts_realtime"`,
		`"protocol": "websocket_realtime_fixture"`,
		`"status": "passed"`,
		`"executed": true`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "doubao-tts", "zh_female_kailangjiejie", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("fixture output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestRunProviderRealtimeFixtureRejectsLegacyProviderWithoutEchoingValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-realtime-fixture", "--provider", "x21_realtime", "--execute"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"provider": "invalid_legacy_provider"`) {
		t.Fatalf("stdout missing invalid legacy provider: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "x21_realtime") || strings.Contains(stderr.String(), "x21_realtime") {
		t.Fatalf("legacy provider leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunDoctorIncludesV21AdapterHealthWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	t.Setenv("A21_V21_ADAPTER_URL", server.URL)

	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"v21"`,
		`"configured": true`,
		`"healthy": true`,
		`"health_path": "/healthz"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorReportsV21AdapterSkippedWhenUnconfigured(t *testing.T) {
	t.Setenv("A21_V21_ADAPTER_URL", "")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"v21"`,
		`"configured": false`,
		`"status": "skipped"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestBuildV21DoctorReportRedactsHealthFailureSecrets(t *testing.T) {
	originalProbe := probeV21AdapterHealth
	probeV21AdapterHealth = func(ctx context.Context, baseURL string) error {
		return errors.New(`Get "http://user:secret-token@127.0.0.1:21121/healthz": connection refused`)
	}
	defer func() {
		probeV21AdapterHealth = originalProbe
	}()

	report := buildV21DoctorReport("http://user:secret-token@127.0.0.1:21121")

	if report.Status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", report.Status)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	if strings.Contains(report.Findings[0].Detail, "secret-token") || strings.Contains(report.Findings[0].Detail, "user:") {
		t.Fatalf("detail leaked credentials: %q", report.Findings[0].Detail)
	}
}

func TestBuildFirmwareDoctorReportFindsCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	manifest := writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	_ = manifest
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-venv", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio"), []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if !report.ManifestOK {
		t.Fatalf("ManifestOK = false, findings = %#v", report.Findings)
	}
	if !report.PlatformIOVenvOK {
		t.Fatal("expected local PlatformIO venv to be detected")
	}
	if !report.PlatformIOVersionOK {
		t.Fatalf("expected pinned PlatformIO version, got %#v", report)
	}
	if !report.PlatformIOCoreOK {
		t.Fatal("expected local PlatformIO core to be detected")
	}
	if report.CurrentArtifactPath != artifact {
		t.Fatalf("CurrentArtifactPath = %q, want %q", report.CurrentArtifactPath, artifact)
	}
}

func TestBuildFirmwareDoctorReportRejectsLooseCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	content := []byte("firmware\na21-stackchan\n0.1.0\nm5stack-cores3\nabcdef1\n")
	if err := os.WriteFile(artifact, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := os.WriteFile(artifact+".sha256", []byte(hex.EncodeToString(sum[:])+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if report.CurrentArtifactPath != "" {
		t.Fatalf("CurrentArtifactPath = %q, want empty for loose artifact", report.CurrentArtifactPath)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "firmware_current_artifact_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_current_artifact_missing: %#v", report.Findings)
	}
}

func TestBuildFirmwareDoctorReportWarnsOnPlatformIOVersionMismatch(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.2.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "")

	if report.PlatformIOVersionOK {
		t.Fatalf("PlatformIOVersionOK = true, want false: %#v", report)
	}
	if report.PlatformIOVersion != "6.2.0" || report.ExpectedPlatformIOVersion != "6.1.19" {
		t.Fatalf("version fields = got %q expected %q", report.PlatformIOVersion, report.ExpectedPlatformIOVersion)
	}
	var found bool
	for _, finding := range report.Findings {
		if finding.Code == "firmware_platformio_version_mismatch" {
			found = true
			if strings.Contains(strings.ToLower(finding.Detail), "x21") || strings.Contains(strings.ToLower(finding.Detail), "v21") {
				t.Fatalf("finding detail leaked legacy identity: %#v", finding)
			}
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_platformio_version_mismatch: %#v", report.Findings)
	}
}

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunGatewayHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"gateway", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "a21 gateway") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGatewayRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"gateway", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunLatencyBenchMockEmitsPercentileReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench", "--mock", "--iterations", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"mode": "mock"`,
		`"iterations": 3`,
		`"mock_turn_ms"`,
		`"professional_turn_ms"`,
		`"barge_in_stop_ms"`,
		`"audio_ws_downlink_ms"`,
		`"audio_ws_barge_in_stop_ms"`,
		`"p50_ms"`,
		`"p95_ms"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunLatencyBenchWritesReportWhenOutputDirProvided(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench", "--mock", "--iterations", "2", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"mode": "mock"`, `"iterations": 2`, `"audio_ws_downlink_ms"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report missing %q: %s", want, string(data))
		}
	}
}

func TestRunLatencyBenchReportIncludesTraceableEnvironmentMetadataWithoutProxySecrets(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"latency-bench", "--mock", "--iterations", "1", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"metadata"`,
		`"generated_at"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"HTTPS_PROXY"`,
		`"provider_proxy_env"`,
		`"A21_PROVIDER_PROXY_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked proxy value fragment %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"metadata"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"provider_proxy_env"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked proxy value fragment %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunLatencyBenchRequiresMockMode(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"latency-bench"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--mock is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLatencyBenchPercentileUsesNearestRank(t *testing.T) {
	got := percentileMS([]time.Duration{time.Millisecond, 2 * time.Millisecond, 100 * time.Millisecond}, 0.95)
	if got != 100 {
		t.Fatalf("p95 = %v, want 100", got)
	}
}

func TestRunFirmwareCheckRejectsMissingManifest(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", filepath.Join(t.TempDir(), "missing.json")}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunFirmwareCheckAcceptsA21Manifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	platformio := filepath.Join(dir, "platformio.ini")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(platformio, []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware manifest ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCheckRejectsWrongPlatformIOBoard(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-core2")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platformio board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwarePackageCreatesVersionedArtifact(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin")
	if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("firmware a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", outputDir,
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	artifact := filepath.Join(outputDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if _, err := os.Stat(artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".manifest.json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Base(artifact)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "release_manifest_path") {
		t.Fatalf("stdout missing release_manifest_path: %s", stdout.String())
	}
}

func TestRunFirmwarePackageRejectsDirtySourceTree(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{
			Root:   "/tmp/a21",
			Clean:  false,
			Detail: " M firmware/stackchan/src/main.cpp\n?? scratch.bin",
		}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", filepath.Join(dir, "artifacts"),
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "source tree is dirty") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "firmware/stackchan/src/main.cpp") {
		t.Fatalf("stderr missing dirty detail: %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsUnpinnedPlatformIOPlatform(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "platform = espressif32@7.0.1", "platform = espressif32")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platform") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsLegacyGatewayPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "-D A21_GATEWAY_PORT=21080", "-D A21_GATEWAY_PORT=8080")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "gateway port") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckAcceptsMatchingArtifactAndChecksum(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsWrongBoardInArtifactName(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-core2-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.WriteFile(artifact, []byte("firmware"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact+".sha256", []byte(strings.Repeat("0", 64)+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "checksum") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckAcceptsNewestCommitArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	oldArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	newArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-005500.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, newArtifact, []byte("new firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-current-artifact-check",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), newArtifact) {
		t.Fatalf("stdout missing newest artifact %q: %s", newArtifact, stdout.String())
	}
	if !strings.Contains(stdout.String(), "firmware current artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckRequiresCommit(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-current-artifact-check"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactPrunePlanWritesNoDeleteReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-1111111-20260530-010000.bin")
	currentArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-020000.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, currentArtifact, []byte("current firmware"))
	reportDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-prune-plan",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "2222222",
		"--keep-recent", "1",
		"--output-dir", reportDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.summary.v1"`,
		`"delete_allowed": false`,
		`"keep_count": 1`,
		`"prune_candidate_count": 1`,
		`"manual_review_count": 0`,
		`"report_path":`,
		"firmware artifact prune plan ok (no files deleted)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), oldArtifact) {
		t.Fatalf("stdout should be summary-only when --output-dir is set: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(reportDir, "a21-firmware-artifact-prune-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one prune-plan report", matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.v1"`,
		`"reason": "current"`,
		`"reason": "older_than_keep_recent"`,
		oldArtifact,
		currentArtifact,
	} {
		if !strings.Contains(string(reportData), want) {
			t.Fatalf("report missing %q: %s", want, string(reportData))
		}
	}
	for _, path := range []string{oldArtifact, currentArtifact} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %q was removed by prune plan: %v", path, err)
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExplicitPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckHelpShowsRequiredCommit(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"firmware-upload-check", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	for _, want := range []string{"--artifact", "--port", "--commit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckAcceptsExplicitDevicePort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware upload dry-run guard ok (no flash performed)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.upload_guard.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"port_usage"`,
		`"exists": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRejectsUnexpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "1234567",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "commit") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsAmbiguousPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "auto",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "explicit serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsNonSerialDevPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/null",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsNonUSBMacSerialPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.Bluetooth-Incoming-Port",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "USB serial device path") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsBusyPort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: true, Detail: "p1234 cpio"}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "already in use") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsMissingPort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckRejectsLegacyArtifactPathWithoutEchoingPath(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	legacyArtifact := filepath.Join(dir, "x21-artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", legacyArtifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceCheckAcceptsMatchingGatewayReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware device identity guard ok (no flash performed)",
		`"guard_id": "a21.firmware.device_identity_guard.v1"`,
		`"device_identity_confirmed": true`,
		`"flash_allowed": false`,
		`"expected_device_id": "stackchan-001"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareDeviceCheckRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	legacyReport := filepath.Join(dir, "x21-reports", "devices.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", legacyReport,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-reports") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRequiresFreshnessGuard(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--max-device-age-ms requires a value") {
		t.Fatalf("stderr = %q, want max age usage error", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRequiresDeviceID(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{"devices":[]}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--device-id requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRejectsStaleGatewayReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": 1000
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "5000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("stderr = %q, want stale", stderr.String())
	}
}

func TestRunFirmwareDeviceCheckRejectsStaleConnectionStatus(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "stale",
      "device_age_ms": 300001,
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-device-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "connection_status") || !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("stderr = %q, want connection_status stale", stderr.String())
	}
}

func TestRunFirmwareFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware flash plan guard ok (no flash performed)",
		`"guard_id": "a21.firmware.flash_plan_guard.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"port": "/dev/cu.usbmodemA21"`,
		`"device_id": "stackchan-001"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareFlashPlanRejectsLegacyInputPathWithoutEchoingPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	legacyReport := filepath.Join(dir, "v21-reports", "devices.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", legacyReport,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "v21") || strings.Contains(strings.ToLower(stderr.String()), "v21-reports") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareFlashPlanRequiresFreshnessGuard(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--max-device-age-ms requires a value") {
		t.Fatalf("stderr = %q, want max device age usage error", stderr.String())
	}
}

func TestRunFirmwareFlashPlanRejectsActivePlaybackDevice(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	reportPayload := map[string]any{
		"schema_version": "a21.gateway.devices.v1",
		"service":        "a21-gateway",
		"devices": []map[string]any{{
			"device_id":          "stackchan-001",
			"identity_status":    "ok",
			"connection_status":  "online",
			"current_expression": "speaking",
			"playback_stream_id": "a21-stream-active",
			"firmware": map[string]string{
				"id":      "a21-stackchan",
				"version": "0.1.0",
				"board":   "m5stack-cores3",
				"commit":  "abcdef1",
			},
			"last_seen_ms": time.Now().UnixMilli(),
		}},
	}
	reportBytes, err := json.Marshal(reportPayload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, reportBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "active playback") {
		t.Fatalf("stderr = %q, want active playback", stderr.String())
	}
}

func TestRunFirmwareFlashPlanWritesReportWhenOutputDirProvided(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	report := filepath.Join(dir, "devices.json")
	writeTestGatewayDeviceReport(t, report, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": `+fmt.Sprint(time.Now().UnixMilli())+`
    }
  ]
}`)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--device-report", report,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("flash plan report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"guard_id": "a21.firmware.flash_plan_guard.v1"`,
		`"generated_at_ms"`,
		`"flash_allowed": false`,
		`"report_path"`,
		`"device_id": "stackchan-001"`,
		`"port": "/dev/cu.usbmodemA21"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("flash plan report missing %q: %s", want, reportJSON)
		}
	}
	if strings.Contains(reportJSON, `"flash_allowed": true`) {
		t.Fatalf("flash plan report unexpectedly allows flash: %s", reportJSON)
	}
}

func TestRunFirmwareBootstrapFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"firmware bootstrap flash plan ok (no flash performed)",
		`"guard_id": "a21.firmware.bootstrap_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"next_required_confirmation": "firmware-bootstrap-flash-execute_with_confirmation_token"`,
		`"port": "/dev/cu.usbmodemA21"`,
		`"offset": "0x0000"`,
		`"offset": "0x10000"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareBootstrapFlashPlanRejectsLegacyInputPathWithoutEchoingPath(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "v21-artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "v21-artifacts") || strings.Contains(strings.ToLower(stderr.String()), "v21-artifacts") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunFirmwareBootstrapFlashExecuteRequiresConfirmationToken(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareBootstrapFlashExecuteRunsEsptoolCommandWithPlan(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runFirmwareBootstrapFlashCommand
	var command []string
	runFirmwareBootstrapFlashCommand = func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
		command = append([]string(nil), args...)
		_, _ = fmt.Fprintln(stdout, "stub esptool ok")
		return nil
	}
	defer func() {
		runFirmwareBootstrapFlashCommand = originalRunner
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	buildDir, coreDir := writeTestBootstrapFlashImages(t, dir)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-bootstrap-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--confirm", "WRITE_A21_STACKCHAN_FIRMWARE",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"stub esptool ok",
		"firmware bootstrap flash executed",
		`"schema_version": "a21.firmware.bootstrap_flash_execution.v1"`,
		`"flash_executed": true`,
		`"dry_run": false`,
		`"port": "/dev/cu.usbmodemA21"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	commandText := strings.Join(command, " ")
	for _, want := range []string{"esptool.py", "--chip esp32s3", "--port /dev/cu.usbmodemA21", "write_flash", "0x0000", "0x8000", "0xe000", "0x10000"} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("command missing %q: %v", want, command)
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-bootstrap-flash-execution-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one bootstrap execution report", matches)
	}
}

func TestRunFirmwareMicProbeFlashPlanWritesNoFlashReport(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-plan",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.mic_probe_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"platformio_env": "a21_stackchan_cores3_mic_probe"`,
		"firmware mic probe flash plan ok",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-mic-probe-flash-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("plan reports = %d, want 1", len(matches))
	}
}

func TestRunFirmwareMicProbeFlashExecuteRequiresConfirmationToken(t *testing.T) {
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--confirm WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareMicProbeFlashExecuteRunsEsptoolCommandWithPlan(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalSourceDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalSourceDetector
	}()
	originalRunner := runFirmwareBootstrapFlashCommand
	var command []string
	runFirmwareBootstrapFlashCommand = func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
		command = append([]string(nil), args...)
		_, _ = fmt.Fprintln(stdout, "stub esptool ok")
		return nil
	}
	defer func() {
		runFirmwareBootstrapFlashCommand = originalRunner
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	buildDir, coreDir, artifact := writeTestMicProbeFlashImages(t, dir, "abcdef1")
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-mic-probe-flash-execute",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
		"--build-dir", buildDir,
		"--core-dir", coreDir,
		"--confirm", "WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		"stub esptool ok",
		"firmware mic probe flash executed",
		`"schema_version": "a21.firmware.mic_probe_flash_execution.v1"`,
		`"flash_executed": true`,
		`"platformio_env": "a21_stackchan_cores3_mic_probe"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	commandText := strings.Join(command, " ")
	for _, want := range []string{"esptool.py", "--chip esp32s3", "--port /dev/cu.usbmodemA21", "write_flash", "0x0000", "0x8000", "0xe000", "0x10000"} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("command missing %q: %v", want, command)
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-firmware-mic-probe-flash-execution-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("execution reports = %d, want 1", len(matches))
	}
}

func TestRunOfficePreflightBuildsNoFlashReport(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_preflight.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"ready_for_flash_plan": true`,
		`"next_required_confirmation": "firmware-flash-plan_with_explicit_usb_port"`,
		`"gateway_schema_version": "a21.gateway.devices.v1"`,
		`"gateway_service": "a21-gateway"`,
		`"device_report_path"`,
		`"report_path"`,
		`"/dev/cu.usbmodemA21"`,
		"office preflight ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	reportMatches, err := filepath.Glob(filepath.Join(outputDir, "a21-office-preflight-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reportMatches) != 1 {
		t.Fatalf("office preflight reports = %d, want 1: %v", len(reportMatches), reportMatches)
	}
	deviceMatches, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deviceMatches) != 1 {
		t.Fatalf("device reports = %d, want 1: %v", len(deviceMatches), deviceMatches)
	}
}

func TestRunOfficePreflightFailsWithoutUSBSerialCandidate(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.Bluetooth-Incoming-Port",
			USBModem: false,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"ready_for_flash_plan": false`,
		`"flash_allowed": false`,
		`"code": "usb_serial_candidate_missing"`,
		"office preflight failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
	if strings.Contains(stdout.String()+stderr.String(), "x21") || strings.Contains(stdout.String()+stderr.String(), "v21") {
		t.Fatalf("office preflight leaked legacy identity stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficePreflightRejectsActivePlaybackDevice(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_expression": "speaking",
      "playback_stream_id": "a21-stream-active",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	for _, want := range []string{
		`"ready_for_flash_plan": false`,
		`"flash_allowed": false`,
		`"code": "device_not_quiescent"`,
		"active playback",
		"office preflight failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunOfficePreflightRejectsLegacyArtifactDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-preflight",
		"--artifact-dir", filepath.Join(t.TempDir(), "x21-artifacts"),
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy artifact dir rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy artifact dir leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficeHandoffWritesNoFlashManifest(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	currentArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, currentArtifact, []byte("firmware"))
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-handoff",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_handoff.v1"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"physical_acceptance_required": true`,
		`"office_preflight_required": true`,
		`"current_artifact_path": "` + currentArtifact + `"`,
		`"usb_serial_candidate_count": 1`,
		"office handoff manifest ok (no flash, no delete)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-office-handoff-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("handoff reports = %d, want 1: %v", len(matches), matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportData), currentArtifact) {
		t.Fatalf("handoff report missing artifact path: %s", string(reportData))
	}
}

func TestRunOfficeHandoffRejectsLegacyArtifactDirWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-handoff",
		"--artifact-dir", filepath.Join(t.TempDir(), "x21-artifacts"),
		"--commit", "abcdef1",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy artifact dir rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-artifacts") {
		t.Fatalf("legacy artifact dir leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunOfficeAcceptanceAcceptsMatchingHandoffAndPreflight(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()
	outputDir := filepath.Join(dir, "reports")

	var handoffOut bytes.Buffer
	if code := Run([]string{
		"office-handoff",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &handoffOut, &bytes.Buffer{}); code != 0 {
		t.Fatalf("office-handoff code = %d: %s", code, handoffOut.String())
	}
	var preflightOut bytes.Buffer
	var preflightErr bytes.Buffer
	if code := Run([]string{
		"office-preflight",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &preflightOut, &preflightErr); code != 0 {
		t.Fatalf("office-preflight code = %d: stdout=%s stderr=%s", code, preflightOut.String(), preflightErr.String())
	}
	handoffPath := newestGlob(t, filepath.Join(outputDir, "a21-office-handoff-*.json"))
	preflightPath := newestGlob(t, filepath.Join(outputDir, "a21-office-preflight-*.json"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", handoffPath,
		"--office-preflight", preflightPath,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("office-acceptance code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.office_acceptance.v1"`,
		`"physical_acceptance_status": "ready_for_physical_acceptance"`,
		`"flash_allowed": false`,
		`"delete_allowed": false`,
		`"handoff_report_path": "` + handoffPath + `"`,
		`"office_preflight_report_path": "` + preflightPath + `"`,
		`"artifact_path": "` + artifact + `"`,
		`"device_id": "stackchan-001"`,
		"office acceptance gate ok (no flash, no delete)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	acceptancePath := newestGlob(t, filepath.Join(outputDir, "a21-office-acceptance-*.json"))
	data, err := os.ReadFile(acceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"physical_acceptance_status": "ready_for_physical_acceptance"`) {
		t.Fatalf("acceptance report missing ready status: %s", string(data))
	}
}

func TestRunOfficeAcceptanceRejectsMismatchedReports(t *testing.T) {
	dir := t.TempDir()
	handoffPath := filepath.Join(dir, "a21-office-handoff.json")
	preflightPath := filepath.Join(dir, "a21-office-preflight.json")
	if err := os.WriteFile(handoffPath, []byte(`{
  "schema_version": "a21.office_handoff.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "commit": "abcdef1",
  "current_artifact_path": "firmware/artifacts/a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preflightPath, []byte(`{
  "schema_version": "a21.office_preflight.v1",
  "dry_run": true,
  "flash_allowed": false,
  "ready_for_flash_plan": true,
  "device_id": "stackchan-001",
  "commit": "2222222",
  "artifact": {
    "artifact_path": "firmware/artifacts/a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-004500.bin",
    "sha256": "abc"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", handoffPath,
		"--office-preflight", preflightPath,
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"physical_acceptance_status": "blocked"`,
		`"code": "commit_mismatch"`,
		`"code": "artifact_mismatch"`,
		"office acceptance gate failed",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunOfficeAcceptanceRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"office-acceptance",
		"--handoff", filepath.Join(t.TempDir(), "x21-office-handoff.json"),
		"--office-preflight", filepath.Join(t.TempDir(), "a21-office-preflight.json"),
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-office") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanIdentityAcceptanceConfirmsFreshDevice(t *testing.T) {
	originalLister := listFirmwareSerialDevices
	listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
		return []firmwarecheck.SerialDevice{{
			Path:     "/dev/cu.usbmodemA21",
			USBModem: true,
			Usage:    firmwarecheck.PortUsage{Exists: true},
		}}, nil
	}
	defer func() {
		listFirmwareSerialDevices = originalLister
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))
	artifactSHA := readTestSHA256(t, artifact+".sha256")
	officeAcceptancePath := filepath.Join(dir, "a21-office-acceptance.json")
	if err := os.WriteFile(officeAcceptancePath, []byte(`{
  "schema_version": "a21.office_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "physical_acceptance_status": "ready_for_physical_acceptance",
  "commit": "abcdef1",
  "artifact_path": "`+artifact+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_id": "stackchan-001"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Fatalf("path = %q, want /v1/devices", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "last_seen_ms": ` + fmt.Sprint(time.Now().UnixMilli()) + `
    }
  ]
}`))
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", officeAcceptancePath,
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_identity_acceptance.v1"`,
		`"hardware_acceptance_scope": "identity_only"`,
		`"identity_acceptance_status": "identity_confirmed"`,
		`"flash_allowed": false`,
		`"office_acceptance_report_path": "` + officeAcceptancePath + `"`,
		`"artifact_path": "` + artifact + `"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"device_id": "stackchan-001"`,
		`"device_identity_confirmed": true`,
		`"/dev/cu.usbmodemA21"`,
		"stackchan identity acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-identity-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("identity acceptance reports = %d, want 1: %v", len(matches), matches)
	}
	deviceReports, err := filepath.Glob(filepath.Join(outputDir, "a21-devices-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deviceReports) != 1 {
		t.Fatalf("device reports = %d, want 1: %v", len(deviceReports), deviceReports)
	}
}

func TestRunStackChanIdentityAcceptanceRejectsBlockedOfficeAcceptance(t *testing.T) {
	dir := t.TempDir()
	officeAcceptancePath := filepath.Join(dir, "a21-office-acceptance.json")
	if err := os.WriteFile(officeAcceptancePath, []byte(`{
  "schema_version": "a21.office_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "physical_acceptance_status": "blocked",
  "commit": "abcdef1",
  "device_id": "stackchan-001"
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", officeAcceptancePath,
		"--gateway-url", "http://127.0.0.1:1",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_identity_acceptance.v1"`,
		`"identity_acceptance_status": "blocked"`,
		`"flash_allowed": false`,
		`"code": "office_acceptance_not_ready"`,
		"stackchan identity acceptance failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunStackChanIdentityAcceptanceRejectsLegacyReportPathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-identity-acceptance",
		"--office-acceptance", filepath.Join(t.TempDir(), "x21-office-acceptance.json"),
		"--gateway-url", "http://127.0.0.1:21080",
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--max-device-age-ms", "300000",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-office") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanCapabilityAcceptanceConfirmsPhysicalEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	artifactSHA := strings.Repeat("a", 64)
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_path": "`+artifactPath+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(dir, "a21-stackchan-physical-evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+artifactSHA+`",
  "observations": [
    {"capability": "microphone", "status": "passed", "evidence_type": "gateway_audio_frame", "observed_at_ms": 1780000001000},
    {"capability": "speaker", "status": "passed", "evidence_type": "audible_playback", "observed_at_ms": 1780000002000},
    {"capability": "screen", "status": "passed", "evidence_type": "operator_visible_state", "observed_at_ms": 1780000003000},
    {"capability": "screen_touch", "status": "passed", "evidence_type": "touch_event", "observed_at_ms": 1780000004000},
    {"capability": "top_touch", "status": "passed", "evidence_type": "touch_event", "observed_at_ms": 1780000005000},
    {"capability": "servo_y", "status": "passed", "evidence_type": "servo_clamped_motion", "observed_at_ms": 1780000006000},
    {"capability": "rgb", "status": "passed", "evidence_type": "operator_visible_state", "observed_at_ms": 1780000007000}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_capability_acceptance.v1"`,
		`"hardware_acceptance_scope": "physical_capability_evidence"`,
		`"capability_acceptance_status": "confirmed"`,
		`"flash_allowed": false`,
		`"identity_acceptance_report_path": "` + identityPath + `"`,
		`"evidence_report_path": "` + evidencePath + `"`,
		`"device_id": "stackchan-001"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"capability": "microphone"`,
		`"capability": "servo_y"`,
		"stackchan capability acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-capability-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("capability acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanCapabilityAcceptanceBlocksMissingCapabilityEvidence(t *testing.T) {
	dir := t.TempDir()
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+strings.Repeat("b", 64)+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {"id": "a21-stackchan", "version": "0.1.0", "board": "m5stack-cores3", "commit": "abcdef1"},
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      }
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(dir, "a21-stackchan-physical-evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_sha256": "`+strings.Repeat("b", 64)+`",
  "observations": [
    {"capability": "microphone", "status": "passed", "evidence_type": "gateway_audio_frame", "observed_at_ms": 1780000001000}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_capability_acceptance.v1"`,
		`"capability_acceptance_status": "blocked"`,
		`"code": "capability_evidence_missing"`,
		`"message": "physical evidence missing for required StackChan capability \"speaker\""`,
		"stackchan capability acceptance failed",
	} {
		if !strings.Contains(stdout.String()+stderr.String(), want) {
			t.Fatalf("output missing %q: stdout=%s stderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestRunStackChanCapabilityAcceptanceRejectsLegacyEvidencePathWithoutEchoingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", filepath.Join(t.TempDir(), "a21-stackchan-identity-acceptance.json"),
		"--evidence", filepath.Join(t.TempDir(), "x21-physical-evidence.json"),
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy path rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21-physical") {
		t.Fatalf("legacy path leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanMicProbeAcceptanceConfirmsDiagnosticMicPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"listening","servo_y":"38deg","rgb":"#003010","mic_frames_captured":"321","audio_ws_sent_audio_frames":"321","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"1052","mic_last_nonzero_samples":"320"},"identity_status":"ok","connection_status":"online","current_expression":"listening","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 374",
				"a21_audio_ingress_frames_total 374",
				"a21_audio_ingress_rms 0.01296457627255574",
				"a21_audio_playback_chunk_total 0",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 99`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--min-frames", "300",
		"--min-abs-peak", "100",
		"--min-nonzero-samples", "300",
		"--min-gateway-rms", "0.001",
		"--min-vad-speech", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_mic_probe_acceptance.v1"`,
		`"mic_probe_acceptance_status": "confirmed"`,
		`"hardware_acceptance_scope": "diagnostic_microphone_only"`,
		`"production_capability_promoted": false`,
		`"microphone": "diagnostic_probe_m5unified_i2s_capture"`,
		`"mic_frames_captured": 321`,
		`"mic_last_abs_peak": 1052`,
		`"mic_last_nonzero_samples": 320`,
		`"gateway_audio_frame_total": 374`,
		`"gateway_audio_ingress_rms": 0.01296457627255574`,
		`"gateway_vad_speech_total": 99`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-mic-probe-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("mic probe acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanMicProbeAcceptanceRunsWindowedProbeWithDeltas(t *testing.T) {
	probeStarted := false
	probeStopped := false
	var probeControlSeen bool
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID       string `json:"device_id"`
				State          string `json:"state"`
				Mode           string `json:"mode"`
				Text           string `json:"text"`
				TraceID        string `json:"trace_id"`
				SessionID      string `json:"session_id"`
				AudioProbeOnly bool   `json:"audio_probe_only"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "listening" {
				probeControlSeen = true
				probeStarted = true
				if !payload.AudioProbeOnly {
					t.Fatalf("probe control did not set audio_probe_only: %+v", payload)
				}
				if payload.TraceID == "" || payload.SessionID == "" {
					t.Fatalf("probe control missing trace/session: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
				probeStopped = true
				if payload.AudioProbeOnly {
					t.Fatalf("idle control should clear audio_probe_only: %+v", payload)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"350","audio_ws_sent_audio_frames":"350","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"900","mic_last_nonzero_samples":"319"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"40","audio_ws_sent_audio_frames":"40","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 430",
					"a21_audio_ingress_frames_total 430",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 7",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 11`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 100",
				"a21_audio_ingress_frames_total 100",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 7",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-frames", "300",
		"--min-abs-peak", "100",
		"--min-nonzero-samples", "300",
		"--min-gateway-rms", "0.001",
		"--min-vad-speech", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !probeControlSeen || !idleControlSeen || !probeStopped {
		t.Fatalf("probeControlSeen=%v idleControlSeen=%v probeStopped=%v", probeControlSeen, idleControlSeen, probeStopped)
	}
	for _, want := range []string{
		`"window_ms": 1`,
		`"mic_frames_captured_delta": 310`,
		`"audio_ws_sent_audio_frames_delta": 310`,
		`"audio_ws_delivery_ratio": 1`,
		`"gateway_audio_frame_delta": 330`,
		`"gateway_audio_ingress_frames_delta": 330`,
		`"gateway_ingress_delivery_ratio": 1`,
		`"gateway_audio_playback_chunk_total": 7`,
		`"gateway_audio_playback_chunk_delta": 0`,
		`"gateway_vad_speech_delta": 10`,
		`"mic_probe_acceptance_status": "confirmed"`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanMicProbeAcceptanceBlocksLowDeliveryRatio(t *testing.T) {
	probeStarted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				State string `json:"state"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.State == "listening" {
				probeStarted = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-mic-probe","session_id":"a21-session-mic-probe","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"listening","servo_y":"38deg","rgb":"#003010","mic_frames_captured":"350","audio_ws_sent_audio_frames":"200","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"900","mic_last_nonzero_samples":"319"},"identity_status":"ok","connection_status":"online","current_expression":"listening","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"40","audio_ws_sent_audio_frames":"40","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"20","mic_last_nonzero_samples":"20"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, strings.Join([]string{
					"a21_audio_frame_total 170",
					"a21_audio_ingress_frames_total 170",
					"a21_audio_ingress_rms 0.0042",
					"a21_audio_playback_chunk_total 7",
					`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 11`,
				}, "\n"))
				return
			}
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 100",
				"a21_audio_ingress_frames_total 100",
				"a21_audio_ingress_rms 0.0001",
				"a21_audio_playback_chunk_total 7",
				`a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"} 1`,
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--min-frames", "50",
		"--min-delivery-ratio", "0.95",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"audio_ws_delivery_ratio": 0.516`,
		`"gateway_ingress_delivery_ratio": 0.438`,
		`"code": "audio_ws_delivery_ratio_below_threshold"`,
		`"code": "gateway_ingress_delivery_ratio_below_threshold"`,
		"stackchan mic probe acceptance blocked (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanMicProbeAcceptanceTreatsMissingSpeechSeriesAsZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"diagnostic_probe_m5unified_i2s_capture","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","mic_frames_captured":"90","audio_ws_sent_audio_frames":"90","mic_driver_errors":"0","mic_queue_depth":"0","mic_queue_dropped_frames":"0","mic_last_abs_peak":"120","mic_last_nonzero_samples":"300"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-mic-probe","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, strings.Join([]string{
				"a21_audio_frame_total 90",
				"a21_audio_ingress_frames_total 90",
				"a21_audio_ingress_rms 0.0042",
				"a21_audio_playback_chunk_total 0",
			}, "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-mic-probe-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--min-frames", "90",
		"--min-vad-speech", "0",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"gateway_vad_speech_total": 0`,
		`"mic_probe_acceptance_status": "confirmed"`,
		"stackchan mic probe acceptance ok (diagnostic only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanSpeakerAcceptanceConfirmsInstrumentedDownlink(t *testing.T) {
	probeStarted := false
	probeStopped := false
	var speakingControlSeen bool
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				Mode            string `json:"mode"`
				Text            string `json:"text"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingControlSeen = true
				probeStarted = true
				if payload.MockAudioChunks != 4 {
					t.Fatalf("mock_audio_chunks = %d, want 4", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
				if payload.TraceID == "" || payload.SessionID == "" {
					t.Fatalf("speaker control missing trace/session: %+v", payload)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
				probeStopped = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if probeStarted {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"14","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"24","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if probeStarted {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 13\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "4",
		"--min-played-frames", "4",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !speakingControlSeen || !idleControlSeen || !probeStopped {
		t.Fatalf("speakingControlSeen=%v idleControlSeen=%v probeStopped=%v", speakingControlSeen, idleControlSeen, probeStopped)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_speaker_acceptance.v1"`,
		`"speaker_acceptance_status": "confirmed"`,
		`"hardware_acceptance_scope": "instrumented_speaker_downlink"`,
		`"physical_sound_observed": false`,
		`"mock_audio_chunks": 4`,
		`"stream_id": "a21-speaker-acceptance-stream"`,
		`"playback_buffer_total_chunks_delta": 4`,
		`"speaker_frames_played_delta": 4`,
		`"gateway_audio_playback_chunk_delta": 4`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-speaker-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("speaker acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanSpeakerAcceptanceBatchesAudibleProbe(t *testing.T) {
	var totalChunks int
	var speakingCalls int
	var idleControlSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			var payload struct {
				DeviceID        string `json:"device_id"`
				State           string `json:"state"`
				TraceID         string `json:"trace_id"`
				SessionID       string `json:"session_id"`
				StreamID        string `json:"stream_id"`
				MockAudioChunks int    `json:"mock_audio_chunks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.DeviceID != "stackchan-001" {
				t.Fatalf("device_id = %q", payload.DeviceID)
			}
			if payload.State == "speaking" {
				speakingCalls++
				totalChunks += payload.MockAudioChunks
				if payload.MockAudioChunks < 1 || payload.MockAudioChunks > 4 {
					t.Fatalf("speaker batch mock_audio_chunks = %d, want 1..4", payload.MockAudioChunks)
				}
				if payload.StreamID != "a21-speaker-acceptance-stream" {
					t.Fatalf("stream_id = %q", payload.StreamID)
				}
			}
			if payload.State == "idle" {
				idleControlSeen = true
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":%q,"session_id":%q,"device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`, payload.TraceID, payload.SessionID)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			if totalChunks >= 50 {
				fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"speaking","servo_y":"48deg","rgb":"#002430","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"60","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"70","speaker_busy_ticks":"4","speaker_driver_errors":"0","speaker_last_stream_id":"a21-speaker-acceptance-stream"},"identity_status":"ok","connection_status":"online","current_expression":"speaking","playback_stream_id":"a21-speaker-acceptance-stream","last_session_id":"a21-session-speaker-acceptance","last_seen_ms":%d}]}`, time.Now().UnixMilli())
				return
			}
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abcdef1"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"3","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_session_id":"a21-session-before","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			if totalChunks >= 50 {
				fmt.Fprint(w, "a21_audio_playback_chunk_total 59\n")
				return
			}
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--window-ms", "1",
		"--mock-audio-chunks", "50",
		"--min-played-frames", "50",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if speakingCalls < 2 || totalChunks != 50 || !idleControlSeen {
		t.Fatalf("speakingCalls=%d totalChunks=%d idleControlSeen=%v", speakingCalls, totalChunks, idleControlSeen)
	}
	for _, want := range []string{
		`"mock_audio_chunks": 50`,
		`"expected_audio_duration_ms": 1000`,
		`"playback_buffer_total_chunks_delta": 50`,
		`"speaker_frames_played_delta": 50`,
		`"gateway_audio_playback_chunk_delta": 50`,
		"stackchan speaker acceptance ok (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanSpeakerAcceptanceBlocksBeforePlaybackOnFirmwareMismatch(t *testing.T) {
	controlCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			controlCalled = true
			http.Error(w, "control should not be called", http.StatusInternalServerError)
		case "/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"older123"},"capabilities":{"microphone":"disabled_m5unified_i2s_stop_crash_guard","speaker":"available","screen":"available","screen_touch":"available","top_touch":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle","servo_y":"45deg","rgb":"#101010","playback_buffer_queued_chunks":"0","playback_buffer_total_chunks":"10","playback_buffer_dropped_chunks":"0","playback_buffer_clear_count":"1","speaker_frames_played":"20","speaker_busy_ticks":"0","speaker_driver_errors":"0","speaker_last_stream_id":"a21-old-stream"},"identity_status":"ok","connection_status":"online","current_expression":"idle","last_seen_ms":%d}]}`, time.Now().UnixMilli())
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "a21_audio_playback_chunk_total 9\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-speaker-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--commit", "5a4936f9a993",
		"--window-ms", "1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code = 0, want blocked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if controlCalled {
		t.Fatalf("speaker control was called despite firmware commit mismatch")
	}
	for _, want := range []string{
		`"speaker_acceptance_status": "blocked"`,
		`"code": "firmware_commit_mismatch"`,
		"stackchan speaker acceptance blocked (instrumented only, no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanTouchAcceptancePassesTopTapWithGatewayTrace(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			if r.Method != http.MethodPost {
				t.Fatalf("control method = %s", r.Method)
			}
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_tap","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastSource := ""
			lastTrace := ""
			lastSeen := int64(1780000001000)
			if armed {
				lastEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000123"
				lastSeen = time.Now().UnixMilli()
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastSource, lastTrace, lastSeen)
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-device-000123" {
				t.Fatalf("trace id = %q", r.URL.Query().Get("trace_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000123","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000123","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_tap",
		"--window-ms", "1000",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_touch_acceptance.v1"`,
		`"touch_acceptance_status": "passed"`,
		`"expected_event": "touch.top.tap"`,
		"stackchan touch acceptance ok (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-touch-acceptance-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("touch acceptance reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanTouchAcceptanceMarksDirectionalCaseAsAffordanceBoundWhenMissed(t *testing.T) {
	var armed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/devices/control":
			armed = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"trace_id":"a21-trace-touch-acceptance-top_swipe_backward","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","status":"delivered","delivered_transport":"audio_ws","events":[]}`)
		case "/v1/devices":
			lastEvent := ""
			lastSource := ""
			lastTrace := ""
			lastSeen := int64(1780000001000)
			if armed {
				lastEvent = "touch.top.tap"
				lastSource = "top_sensor"
				lastTrace = "a21-trace-device-000124"
				lastSeen = time.Now().UnixMilli()
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"schema_version":"a21.gateway.devices.v1","service":"a21-gateway","devices":[{"device_id":"stackchan-001","firmware":{"id":"a21-stackchan","version":"0.1.0","board":"m5stack-cores3","commit":"abc123"},"capabilities":{"screen":"available","screen_touch":"available","top_touch":"available","speaker":"available","servo_y":"available","rgb":"available"},"runtime_echo":{"screen":"idle"},"identity_status":"ok","connection_status":"online","last_event":%q,"last_touch_source":%q,"last_trace_id":%q,"last_session_id":"a21-session-touch-acceptance","last_seen_ms":%d,"first_seen_ms":1780000000000}]}`, lastEvent, lastSource, lastTrace, lastSeen)
		case "/v1/traces":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"trace_id":"a21-trace-device-000124","events":[{"name":"device.touch.top.tap.received","trace_id":"a21-trace-device-000124","session_id":"a21-session-touch-acceptance","device_id":"stackchan-001","at_ms":%d,"offset_ms":0}],"summary":{"event_count":1,"last_offset_ms":0}}`, time.Now().UnixMilli())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-touch-acceptance",
		"--gateway-url", server.URL,
		"--device-id", "stackchan-001",
		"--case", "top_swipe_backward",
		"--window-ms", "50",
		"--output-dir", t.TempDir(),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"acceptance_scope": "guided_directional_touch"`,
		`"needs_affordance": true`,
		`"code": "physical_affordance_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanPhysicalEvidenceWritesPendingTemplateFromIdentity(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("c", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan_physical_evidence.v1"`,
		`"identity_acceptance_report_path": "` + identityPath + `"`,
		`"device_id": "stackchan-001"`,
		`"commit": "abcdef1"`,
		`"artifact_sha256": "` + artifactSHA + `"`,
		`"capability": "microphone"`,
		`"capability": "rgb"`,
		`"status": "pending"`,
		`"evidence_type": "operator_observation_required"`,
		`"flash_allowed": false`,
		"stackchan physical evidence template written (no flash performed)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("physical evidence reports = %d, want 1: %v", len(matches), matches)
	}
}

func TestRunStackChanPhysicalEvidenceCanFeedCapabilityAcceptance(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("d", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	outputDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}
	for _, pass := range []string{
		"microphone=gateway_audio_frame",
		"speaker=audible_playback",
		"screen=operator_visible_state",
		"screen_touch=touch_event",
		"top_touch=touch_event",
		"servo_y=servo_clamped_motion",
		"rgb=operator_visible_state",
	} {
		args = append(args, "--pass", pass)
	}
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("physical evidence code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"stackchan-capability-acceptance",
		"--identity-acceptance", identityPath,
		"--evidence", evidencePath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("capability acceptance code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"capability_acceptance_status": "confirmed"`) {
		t.Fatalf("capability acceptance not confirmed: %s", stdout.String())
	}
}

func TestRunStackChanPhysicalEvidenceDerivesGatewayObservableSignals(t *testing.T) {
	dir := t.TempDir()
	artifactSHA := strings.Repeat("f", 64)
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, artifactSHA)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/devices":
			_, _ = w.Write([]byte(`{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "speaking",
      "runtime_echo": {"screen": "speaking", "servo_y": "48deg", "rgb": "#002430"},
      "last_event": "touch.wake_or_listen",
      "last_touch_source": "screen",
      "last_trace_id": "a21-trace-physical-auto",
      "firmware": {"id": "a21-stackchan", "version": "0.1.0", "board": "m5stack-cores3", "commit": "abcdef1"},
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`))
		case "/v1/traces":
			if r.URL.Query().Get("trace_id") != "a21-trace-physical-auto" {
				t.Fatalf("trace id = %q, want a21-trace-physical-auto", r.URL.Query().Get("trace_id"))
			}
			_, _ = w.Write([]byte(`{
  "trace_id": "a21-trace-physical-auto",
  "events": [
    {"name": "audio.frame.received", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000001000},
    {"name": "audio.playback.chunk.sent", "trace_id": "a21-trace-physical-auto", "device_id": "stackchan-001", "at_ms": 1780000002000}
  ],
  "summary": {"event_count": 2}
}`))
		default:
			t.Fatalf("path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	outputDir := filepath.Join(dir, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--derive-gateway",
		"--gateway-url", server.URL,
		"--output-dir", outputDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	evidencePath := newestGlob(t, filepath.Join(outputDir, "a21-stackchan-physical-evidence-*.json"))
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"gateway_url": "` + server.URL + `"`,
		`"gateway_trace_id": "a21-trace-physical-auto"`,
		`"capability": "microphone"`,
		`"evidence_type": "gateway_audio_frame"`,
		`"capability": "speaker"`,
		`"evidence_type": "gateway_audio_downlink"`,
		`"capability": "screen"`,
		`"evidence_type": "device_screen_echo"`,
		`"capability": "servo_y"`,
		`"evidence_type": "device_servo_echo"`,
		`"capability": "rgb"`,
		`"evidence_type": "device_rgb_echo"`,
		`"capability": "screen_touch"`,
		`"evidence_type": "gateway_touch_event"`,
		`"capability": "top_touch"`,
		`"status": "pending"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("evidence missing %q: %s", want, text)
		}
	}
}

func TestRunStackChanPhysicalEvidenceRejectsLegacyPassWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, strings.Repeat("e", 64))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--pass", "microphone=x21_probe",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21_probe") {
		t.Fatalf("legacy pass leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func writeTestStackChanIdentityAcceptanceReport(t *testing.T, dir string, artifactSHA string) string {
	t.Helper()
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	artifactPath := filepath.Join(dir, "artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_path": "`+artifactPath+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return identityPath
}

func writeTestFirmwareManifest(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeTestGatewayDeviceReport(t *testing.T, reportPath string, content string) {
	t.Helper()
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") && !strings.Contains(content, `"schema_version"`) {
		content = strings.TrimPrefix(content, "{")
		content = `{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",` + content
	}
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newestGlob(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no files match %q", pattern)
	}
	return matches[len(matches)-1]
}

func testPlatformIOConfig(board string) string {
	return `[platformio]
default_envs = a21_stackchan_cores3

[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = ` + board + `
framework = arduino
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  bblanchon/ArduinoJson @ 7.4.3
`
}

func writeFirmwareArtifactWithChecksum(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	commit := testArtifactCommitFromName(t, artifactPath)
	content = append(content, []byte("\na21-stackchan\n0.1.0\nm5stack-cores3\n"+commit+"\n")...)
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(artifactPath+".sha256", []byte(checksum+"  "+filepath.Base(artifactPath)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := firmwarecheck.ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        commit,
		Timestamp:     testArtifactTimestampFromName(t, artifactPath),
		ArtifactPath:  artifactPath,
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}
	manifestData, err := json.MarshalIndent(firmwarecheck.FirmwareReleaseManifest{
		SchemaVersion: firmwarecheck.ReleaseManifestSchemaVersion,
		Project:       "A21",
		FirmwareID:    entry.FirmwareID,
		Version:       entry.Version,
		Board:         entry.Board,
		Commit:        entry.Commit,
		Timestamp:     entry.Timestamp,
		ArtifactPath:  artifactPath,
		ArtifactName:  filepath.Base(artifactPath),
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath+".manifest.json", append(manifestData, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(filepath.Dir(artifactPath), firmwarecheck.ReleaseIndexFileName)
	file, err := os.OpenFile(indexPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func writeTestBootstrapFlashImages(t *testing.T, dir string) (string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "build")
	coreDir := filepath.Join(dir, "platformio-core")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 boot app"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir
}

func writeTestMicProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_mic_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 mic probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 mic probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 mic probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_m5unified_i2s_capture\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func readTestSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		t.Fatalf("sha256 file %q is empty", path)
	}
	return fields[0]
}

func testFirmwareBuildProvenance(artifactPath string) firmwarecheck.FirmwareBuildProvenance {
	return firmwarecheck.FirmwareBuildProvenance{
		BuildSystem:     "platformio",
		PlatformIOEnv:   "a21_stackchan_cores3",
		PlatformIOBoard: "m5stack-cores3",
		SourcePath:      filepath.Join(filepath.Dir(artifactPath), "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin"),
		SourceName:      "firmware.bin",
	}
}

func writeNamespaceAuditGitScript(t *testing.T, dir string, files string) {
	t.Helper()
	path := filepath.Join(dir, "git")
	content := "#!/bin/sh\nif [ \"$1\" = \"ls-files\" ]; then\ncat <<'EOF'\n" + files + "EOF\nelse\nexit 2\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func testArtifactCommitFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		t.Fatalf("artifact name %q lacks commit field", name)
	}
	return parts[len(parts)-3]
}

func testArtifactTimestampFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 5 {
		t.Fatalf("artifact name %q lacks timestamp field", name)
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}

func appTestPCM16Base64(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}
