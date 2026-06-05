package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/providers"
)

func TestGatewayServerOptionsFromEnvWiresXiaozhiListenMaxDuration(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_XIAOZHI_LISTEN_MAX_MS=4500",
	})
	if options.XiaozhiListenMaxDuration != 4500*time.Millisecond {
		t.Fatalf("listen max = %s, want 4500ms", options.XiaozhiListenMaxDuration)
	}
}

func TestGatewayServerOptionsFromEnvWiresBodySceneStepDelay(t *testing.T) {
	defaultOptions := newGatewayServerOptionsFromEnv(nil)
	if defaultOptions.BodySceneStepDelay != 180*time.Millisecond {
		t.Fatalf("default body scene step delay = %s, want 180ms", defaultOptions.BodySceneStepDelay)
	}

	options := newGatewayServerOptionsFromEnv([]string{
		"A21_BODY_SCENE_STEP_DELAY_MS=240",
	})
	if options.BodySceneStepDelay != 240*time.Millisecond {
		t.Fatalf("body scene step delay = %s, want 240ms", options.BodySceneStepDelay)
	}

	zeroOptions := newGatewayServerOptionsFromEnv([]string{
		"A21_BODY_SCENE_STEP_DELAY_MS=0",
	})
	if zeroOptions.BodySceneStepDelay != 180*time.Millisecond {
		t.Fatalf("zero body scene step delay = %s, want default 180ms", zeroOptions.BodySceneStepDelay)
	}
}

func TestGatewayServerFromEnvExposesConfiguredPublicGatewayProfile(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_MAC_LOCAL_GATEWAY_URL=ws://192.168.1.20:21081/v1/xiaozhi",
		"A21_PUBLIC_GATEWAY_URL=https://a21.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/gateway-profiles", nil)
	req.Host = "127.0.0.1:21080"
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"selected_gateway_profile":"public_wss"`,
		`"id":"public_wss"`,
		`"status":"available"`,
		`"websocket_url":"ws://192.168.1.20:21081/v1/xiaozhi"`,
		`"websocket_url":"wss://a21.example.com/v1/xiaozhi"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("catalog missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{"token", "secret", "Authorization", "Bearer"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("catalog leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerFromEnvExposesCloudVoiceProfileWithoutSecrets(t *testing.T) {
	server := newGatewayServerFromEnv([]string{
		"A21_CLOUD_VOICE_PROFILE=a21_minimax_t2a_ws",
		"A21_MINIMAX_API_KEY=sk-a21-minimax-secret",
		"A21_MINIMAX_GROUP_ID=minimax-secret-group",
		"A21_MINIMAX_TTS_MODEL=minimax-secret-model",
		"A21_MINIMAX_VOICE_ID=minimax-secret-voice",
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/cloud-voice-profiles", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"schema_version":"a21.gateway.cloud_voice_profiles.v1"`,
		`"selected_cloud_voice_profile":"a21_minimax_t2a_ws"`,
		`"id":"a21_minimax_t2a_ws"`,
		`"configured":true`,
		`"A21_MINIMAX_API_KEY"`,
		`"A21_MINIMAX_GROUP_ID"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("cloud voice catalog missing %q: %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{
		"sk-a21-minimax-secret",
		"minimax-secret-group",
		"minimax-secret-model",
		"minimax-secret-voice",
		"Authorization",
		"Bearer",
	} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("cloud voice catalog leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestGatewayServerOptionsFromEnvWiresSileroVADConfig(t *testing.T) {
	options := newGatewayServerOptionsFromEnv([]string{
		"A21_VAD_PREFERENCE=silero",
		"A21_SILERO_VAD_COMMAND=/tmp/a21-silero-runner",
		"A21_SILERO_VAD_MODEL=/tmp/a21-silero.onnx",
		"A21_SILERO_VAD_TIMEOUT_MS=90",
	})
	if options.AudioIngressConfig.VADPreference != audio.VADDetectorPreferenceSilero {
		t.Fatalf("VAD preference = %q, want silero", options.AudioIngressConfig.VADPreference)
	}
	if options.AudioIngressConfig.SileroRunner == nil {
		t.Fatal("Silero runner was not configured from env")
	}
	if options.AudioIngressConfig.VADTimeout != 90*time.Millisecond {
		t.Fatalf("VAD timeout = %s, want 90ms", options.AudioIngressConfig.VADTimeout)
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

func TestGatewayServerFromEnvUsesConfiguredV21AdapterForProfessionalMode(t *testing.T) {
	var sawProfessionalRequest bool
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a21/v21/query" {
			t.Fatalf("path = %q, want /a21/v21/query", r.URL.Path)
		}
		var request struct {
			Mode         string `json:"mode"`
			Utterance    string `json:"utterance"`
			PrivacyScope string `json:"privacy_scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		sawProfessionalRequest = request.Mode == "professional" &&
			request.Utterance == "查一下真实 adapter" &&
			request.PrivacyScope == "professional_only"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-env-v21",
			"fast_answer":"adapter answer marker",
			"confidence":0.91,
			"evidence":[{"title":"adapter source","type":"doc","source_id":"v21-adapter-source","summary":"adapter summary"}],
			"speech_blocks":["adapter speech"],
			"screen_cards":[{"label":"adapter","text":"adapter card"}],
			"follow_ups":["adapter followup"]
		}`))
	}))
	defer adapter.Close()
	server := newGatewayServerFromEnv([]string{"A21_V21_ADAPTER_URL=" + adapter.URL})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下真实 adapter","mode":"professional","trace_id":"a21-trace-env-v21","session_id":"a21-session-env-v21"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !sawProfessionalRequest {
		t.Fatal("gateway did not call configured A21 V21 adapter")
	}
	if !strings.Contains(rec.Body.String(), "adapter answer marker") {
		t.Fatalf("response did not use adapter answer: %s", rec.Body.String())
	}
}

func TestGatewayServerFromEnvInvalidV21AdapterURLFailsProfessionalModeWithoutMockEvidence(t *testing.T) {
	server := newGatewayServerFromEnv([]string{"A21_V21_ADAPTER_URL=http://127.0.0.1:18080"})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下真实 adapter","mode":"professional","trace_id":"a21-trace-env-v21-invalid","session_id":"a21-session-env-v21-invalid"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "V21 现在没接上") {
		t.Fatalf("response should fail honestly instead of using mock evidence: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "历史讨论主要集中") {
		t.Fatalf("response used mock V21 evidence despite invalid configured adapter: %s", rec.Body.String())
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

func TestRunDoctorIncludesWakeWordPendingFirmwareBuildWithoutPathLeaks(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "a21-wake-word.json")
	if err := os.WriteFile(configPath, []byte(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_WAKE_WORD_CONFIG_PATH", configPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)

	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"wake_word"`,
		`"schema_version": "a21.gateway.wake_word.v1"`,
		`"mode": "custom_multinet"`,
		`"active_phrase": "你好小智"`,
		`"desired_phrase": "小阿二一"`,
		`"desired_pinyin": "xiao a er yi"`,
		`"threshold": 35`,
		`"runtime_status": "pending_firmware_build"`,
		`"firmware_build_required": true`,
		`"code": "a21_wake_word_firmware_build_required"`,
		`"wake_word_firmware_build_required"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{configPath, dir, "A21_WAKE_WORD_CONFIG_PATH", "secret", "token", "http://", "https://"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor wake-word report leaked %q: %s", forbidden, stdout.String())
		}
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
	for _, forbidden := range []string{"provider-secret", "http://provider-secret@127.0.0.1:7891", "provider-secret@127.0.0.1:7891", "127.0.0.1:7891"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked provider proxy value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesProviderCatalogWithoutSecrets(t *testing.T) {
	t.Setenv("A21_PROVIDER_PRIMARY", "deepseek")
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1", code)
	}
	for _, want := range []string{`"providers"`, `"primary": "deepseek"`, `"name": "deepseek"`, `"family": "text_stream"`, `"configured": true`, `"A21_LAB_DEEPSEEK_API_KEY"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked provider value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorIncludesCloudVoiceProfilesWithoutSecrets(t *testing.T) {
	setA21DirectProxyBypassForTest(t)
	t.Setenv("A21_CLOUD_VOICE_PROFILE", "a21_bailian_qwen_tts_realtime")
	t.Setenv("A21_DASHSCOPE_API_KEY", "sk-a21-bailian-secret")
	t.Setenv("A21_BAILIAN_QWEN_TTS_MODEL", "qwen-tts-secret-model")
	t.Setenv("A21_BAILIAN_QWEN_TTS_VOICE_ID", "voice-secret-id")
	t.Setenv("A21_MINIMAX_API_KEY", "sk-a21-minimax-secret")
	t.Setenv("A21_MINIMAX_GROUP_ID", "minimax-secret-group")
	t.Setenv("A21_MINIMAX_TTS_MODEL", "minimax-secret-model")
	t.Setenv("A21_MINIMAX_VOICE_ID", "minimax-secret-voice")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor", "--output-dir", t.TempDir()}, &stdout, &stderr)

	if code != 0 && code != 1 {
		t.Fatalf("code = %d, want 0 or 1: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"cloud_voice"`,
		`"schema_version": "a21.cloud_voice_profiles.v1"`,
		`"selected_cloud_voice_profile": "a21_bailian_qwen_tts_realtime"`,
		`"id": "a21_bailian_qwen_tts_realtime"`,
		`"id": "a21_minimax_t2a_ws"`,
		`"status": "catalog_only"`,
		`"configured": true`,
		`"A21_DASHSCOPE_API_KEY"`,
		`"A21_BAILIAN_QWEN_TTS_MODEL"`,
		`"A21_MINIMAX_GROUP_ID"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		"sk-a21-bailian-secret",
		"sk-a21-minimax-secret",
		"qwen-tts-secret-model",
		"voice-secret-id",
		"minimax-secret-group",
		"minimax-secret-model",
		"minimax-secret-voice",
		"Authorization",
		"Bearer",
	} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets(t *testing.T) {
	setA21DirectProxyBypassForTest(t)
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
		`"name": "doubao_tts_realtime"`,
		`"family": "voice_hybrid"`,
		`"status": "unsupported"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"provider_unknown"`) {
		t.Fatalf("doctor reported selected realtime TTS profile as unknown: %s", stdout.String())
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
	setA21DirectProxyBypassForTest(t)
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
		`"name": "doubao_realtime"`,
		`"family": "voice_realtime"`,
		`"status": "unsupported"`,
		`"status": "ready"`,
		`"endpoint_host": "ai-gateway.vei.volces.com"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"provider_unknown"`) {
		t.Fatalf("doctor reported selected realtime profile as unknown: %s", stdout.String())
	}
	for _, forbidden := range []string{"sk-a21-secret", "app-a21-secret", "resource-a21-secret", "doubao-s2s", "Authorization", "Bearer"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("doctor leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime(t *testing.T) {
	setA21DirectProxyBypassForTest(t)
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
	setA21DirectProxyBypassForTest(t)
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
	t.Setenv("A21_LAB_DEEPSEEK_API_KEY", "sk-a21-secret")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"provider-smoke", "--provider", "deepseek"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{`"provider": "deepseek"`, `"status": "ready"`, `"executed": false`, `"api_key_env": "A21_LAB_DEEPSEEK_API_KEY"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"sk-a21-secret", "deepseek-chat"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunProviderSmokeReportsHotPlugProfileRouteEligibility(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "a21-provider-profiles.json")
	if err := os.WriteFile(profilePath, []byte(`{
		"name": "a21_ws7a_cli_vendor",
		"label": "A21 WS7A CLI Vendor",
		"family": "text_stream",
		"protocol": "openai_chat_completions",
		"capabilities": ["llm", "text_stream"],
		"api_key_env": "A21_WS7A_CLI_VENDOR_API_KEY",
		"model_env": "A21_WS7A_CLI_VENDOR_MODEL",
		"base_url_env": "A21_WS7A_CLI_VENDOR_BASE_URL",
		"endpoint_path": "/chat/completions",
		"route_eligible": true
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A21_PROVIDER_PROFILES_PATH", profilePath)
	t.Setenv("A21_PROVIDER_PRIMARY", "a21_ws7a_cli_vendor")
	t.Setenv("A21_WS7A_CLI_VENDOR_API_KEY", "sk-a21-cli-secret")
	t.Setenv("A21_WS7A_CLI_VENDOR_MODEL", "cli-hidden-model")
	t.Setenv("A21_WS7A_CLI_VENDOR_BASE_URL", "http://127.0.0.1:9/v1")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"provider-smoke", "--provider", "a21_ws7a_cli_vendor", "--stream", "--repeat", "2"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"provider": "a21_ws7a_cli_vendor"`,
		`"status": "ready"`,
		`"configured": true`,
		`"executed": false`,
		`"stream": true`,
		`"repeat": 2`,
		`"route_eligible": true`,
		`"api_key_env": "A21_WS7A_CLI_VENDOR_API_KEY"`,
		`"model_env": "A21_WS7A_CLI_VENDOR_MODEL"`,
		`"base_url_env": "A21_WS7A_CLI_VENDOR_BASE_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		profilePath,
		filepath.Dir(profilePath),
		"sk-a21-cli-secret",
		"cli-hidden-model",
		"http://127.0.0.1:9/v1",
	} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("provider smoke CLI leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
}
