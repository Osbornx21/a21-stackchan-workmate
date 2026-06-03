package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/audio/opuscodec"
	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/providers"
	xiaozhitransport "a21.local/a21/internal/transport/xiaozhi"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestHealthz(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"service":"a21-gateway"`)) {
		t.Fatalf("body missing service: %s", rec.Body.String())
	}
}

func TestXiaozhiOTAEndpointReturnsStockWebSocketConfig(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/xiaozhi/ota/", strings.NewReader(`{"application":{"version":"0.0.1"}}`))
	req.Host = "192.0.2.10:21080"
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		WebSocket struct {
			URL     string `json:"url"`
			Token   string `json:"token"`
			Version int    `json:"version"`
		} `json:"websocket"`
		ServerTime struct {
			Timestamp      int64 `json:"timestamp"`
			TimezoneOffset int   `json:"timezone_offset"`
		} `json:"server_time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.WebSocket.URL != "ws://192.0.2.10:21080/v1/xiaozhi" {
		t.Fatalf("websocket url = %q", body.WebSocket.URL)
	}
	if body.WebSocket.Version != 1 {
		t.Fatalf("websocket version = %d, want stock v1", body.WebSocket.Version)
	}
	if body.WebSocket.Token != "" {
		t.Fatalf("websocket token should be empty in local stock profile")
	}
	if body.ServerTime.Timestamp == 0 || body.ServerTime.TimezoneOffset != 480 {
		t.Fatalf("server_time = %+v", body.ServerTime)
	}
}

func TestXiaozhiOTAEndpointUsesWSSBehindTLSReverseProxy(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/xiaozhi/ota/", strings.NewReader(`{}`))
	req.Host = "a21.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		WebSocket struct {
			URL string `json:"url"`
		} `json:"websocket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.WebSocket.URL != "wss://a21.example.com/v1/xiaozhi" {
		t.Fatalf("websocket url = %q", body.WebSocket.URL)
	}
}

func TestGatewayProfilesCatalogDefaultsToPublicWSSAndAllowsMacLocalSwitch(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		MacLocalGatewayURL: "ws://192.168.1.20:21081/v1/xiaozhi",
		PublicGatewayURL:   "https://a21.example.com",
	})
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/v1/gateway-profiles", nil)
	req.Host = "127.0.0.1:21080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var catalog struct {
		SchemaVersion string `json:"schema_version"`
		Selected      string `json:"selected_gateway_profile"`
		Profiles      []struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			Default      bool   `json:"default"`
			WebSocketURL string `json:"websocket_url"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.SchemaVersion != "a21.gateway.profiles.v1" || catalog.Selected != "public_wss" {
		t.Fatalf("catalog = %+v", catalog)
	}
	seen := map[string]struct {
		status  string
		url     string
		Default bool
	}{}
	defaults := 0
	for _, profile := range catalog.Profiles {
		seen[profile.ID] = struct {
			status  string
			url     string
			Default bool
		}{status: profile.Status, url: profile.WebSocketURL, Default: profile.Default}
		if profile.Default {
			defaults++
		}
	}
	if seen["mac_local"].status != "available" || seen["public_wss"].status != "available" || defaults != 1 {
		t.Fatalf("profiles = %+v, seen=%v defaults=%d", catalog.Profiles, seen, defaults)
	}
	if seen["mac_local"].Default || !seen["public_wss"].Default {
		t.Fatalf("default flags = %+v", seen)
	}
	if seen["mac_local"].url != "ws://192.168.1.20:21081/v1/xiaozhi" {
		t.Fatalf("mac local websocket url = %q", seen["mac_local"].url)
	}
	if seen["public_wss"].url != "wss://a21.example.com/v1/xiaozhi" {
		t.Fatalf("public websocket url = %q", seen["public_wss"].url)
	}

	selectReq := httptest.NewRequest(http.MethodPost, "/v1/gateway-profiles", strings.NewReader(`{"gateway_profile":"mac_local"}`))
	selectReq.Host = "127.0.0.1:21080"
	selectRec := httptest.NewRecorder()
	handler.ServeHTTP(selectRec, selectReq)
	if selectRec.Code != http.StatusOK {
		t.Fatalf("select status = %d, want 200: %s", selectRec.Code, selectRec.Body.String())
	}
	if !bytes.Contains(selectRec.Body.Bytes(), []byte(`"selected_gateway_profile":"mac_local"`)) {
		t.Fatalf("selected mac profile missing: %s", selectRec.Body.String())
	}

	otaReq := httptest.NewRequest(http.MethodPost, "/xiaozhi/ota/", strings.NewReader(`{}`))
	otaReq.Host = "127.0.0.1:21080"
	otaRec := httptest.NewRecorder()
	handler.ServeHTTP(otaRec, otaReq)
	if otaRec.Code != http.StatusOK {
		t.Fatalf("ota status = %d, want 200: %s", otaRec.Code, otaRec.Body.String())
	}
	var ota struct {
		WebSocket struct {
			URL string `json:"url"`
		} `json:"websocket"`
	}
	if err := json.Unmarshal(otaRec.Body.Bytes(), &ota); err != nil {
		t.Fatal(err)
	}
	if ota.WebSocket.URL != "ws://192.168.1.20:21081/v1/xiaozhi" {
		t.Fatalf("ota websocket url = %q", ota.WebSocket.URL)
	}
}

func TestGatewayProfilesRejectsPublicWSSWithoutConfiguredPublicURL(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway-profiles", strings.NewReader(`{"gateway_profile":"public_wss"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for unconfigured public gateway: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayProfilesAcceptsPublicHTTPBringupURL(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{PublicGatewayURL: "http://47.103.57.217"})
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/v1/gateway-profiles", nil)
	req.Host = "127.0.0.1:21080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"selected_gateway_profile":"public_wss"`)) {
		t.Fatalf("public profile not selected for HTTP bring-up: %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"websocket_url":"ws://47.103.57.217/v1/xiaozhi"`)) {
		t.Fatalf("HTTP public URL was not normalized to ws: %s", rec.Body.String())
	}

	otaReq := httptest.NewRequest(http.MethodGet, "/xiaozhi/ota/", nil)
	otaReq.Host = "127.0.0.1:21080"
	otaRec := httptest.NewRecorder()
	handler.ServeHTTP(otaRec, otaReq)
	if otaRec.Code != http.StatusOK {
		t.Fatalf("ota status = %d, want 200: %s", otaRec.Code, otaRec.Body.String())
	}
	if !bytes.Contains(otaRec.Body.Bytes(), []byte(`"url":"ws://47.103.57.217/v1/xiaozhi"`)) {
		t.Fatalf("ota missing public ws URL: %s", otaRec.Body.String())
	}
}

func TestGatewayProfilesRejectsPublicURLWithQuery(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{PublicGatewayURL: "https://a21.example.com/v1/xiaozhi?key=redacted"})
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway-profiles", strings.NewReader(`{"gateway_profile":"public_wss"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for query-bearing public gateway URL: %s", rec.Code, rec.Body.String())
	}
}

func TestXiaozhiOTAEndpointRejectsUnsafeHost(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/xiaozhi/ota/", strings.NewReader(`{}`))
	req.Host = "127.0.0.1:21080@evil.example"
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestVoiceProviderHealthEndpoint(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{events: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "ready", Final: true},
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var health providers.VoiceProviderHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Provider != "scripted" || health.Status != providers.VoiceProviderHealthy || !health.Realtime || !health.Configured {
		t.Fatalf("health = %+v", health)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"provider":"scripted"`)) {
		t.Fatalf("body missing provider json field: %s", rec.Body.String())
	}
}

func TestVoiceProviderHealthEndpointReturnsUnavailable(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{VoiceProvider: unavailableVoiceProvider{}})
	req := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", rec.Code, rec.Body.String())
	}
	var health providers.VoiceProviderHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Provider != "unavailable" || health.Status != providers.VoiceProviderUnavailable {
		t.Fatalf("health = %+v", health)
	}
}

func TestSimulatorPageServed(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/simulator", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type = %q, want text/html", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"A21 Device Simulator",
		`data-testid="simulator-root"`,
		"/ws/control",
		"/v1/devices",
		"/v1/voice-modes",
		"/v1/roleplay-profile",
		"/v1/professional-workspace",
		"/v1/workspace-upload-jobs",
		"/v1/traces",
		"Device Registry",
		`id="registryConnection"`,
		`id="registryMode"`,
		`id="voiceMode"`,
		`id="roleplayScenario"`,
		`id="roleplayMemoryHint"`,
		`id="saveRoleplayMemory"`,
		`id="clearRoleplayMemory"`,
		`id="roleplayMemoryReadout"`,
		`id="modeRitualReadout"`,
		`id="professionalQueryScope"`,
		`id="workspaceDocumentLabel"`,
		`id="workspaceJob"`,
		`id="gatewayProfile"`,
		`id="cloudVoiceProfile"`,
		`id="registryVoiceMode"`,
		`id="registryCloudVoiceProfile"`,
		`id="registryExpression"`,
		`id="gatewayProfileReadout"`,
		`id="cloudVoiceProfileReadout"`,
		"Waterfall",
		"Latency Summary",
		"Professional Evidence",
		"Audio Link",
		"Office Visibility",
		`id="visibilityBadge"`,
		`id="privacyBadge"`,
		`id="listeningBadge"`,
		"PUBLIC",
		"PRIVATE",
		"PRO",
		"MUTED",
		"LOCAL",
		`data-state="local_fallback"`,
		"updateVisibilityBadges",
		`id="startMic"`,
		`id="stopMic"`,
		`id="mockAudioBurst"`,
		`id="audioFramesSent"`,
		`id="playbackState"`,
		`id="playbackChunksReceived"`,
		`id="playbackBufferedChunks"`,
		`id="playbackStream"`,
		`id="playbackScheduledChunks"`,
		`id="latencyAudioPlayback"`,
		`id="latencyV21"`,
		`id="latencyBargeIn"`,
		`id="latencyProviderFirstAudio"`,
		"renderLatencySummary",
		"startMicrophoneStream",
		"stopMicrophoneStream",
		"sendMockAudioBurst",
		"Wake Word",
		"/v1/wake-word",
		`id="wakeWordMode"`,
		`id="wakeWordPhrase"`,
		`id="wakeWordPinyin"`,
		`id="wakeWordThreshold"`,
		`id="saveWakeWord"`,
		`id="wakeWordStatus"`,
		"refreshWakeWordConfig",
		"saveWakeWordConfig",
		"/v1/gateway-profiles",
		"refreshGatewayProfiles",
		"saveGatewayProfile",
		"/v1/cloud-voice-profiles",
		"refreshCloudVoiceProfiles",
		"saveCloudVoiceProfile",
		"/v1/voice-chain-profiles",
		"refreshVoiceChainProfiles",
		"saveVoiceChainProfile",
		`id="voiceChainMode"`,
		`id="cascadeASRProfile"`,
		`id="cascadeLLMProfile"`,
		`id="realtimeProvider"`,
		`id="voiceCloneProfile"`,
		`id="registryVoiceChainMode"`,
		`id="registryASRProfile"`,
		`id="registryLLMProfile"`,
		`id="registryTTSProfile"`,
		`id="registryRealtimeProvider"`,
		`id="registryVoiceCloneProfile"`,
		"handleAudioPlaybackChunk",
		"decodePCM16Base64",
		"schedulePCMPlayback",
		"stopScheduledPlayback",
		"audio.playback.chunk",
		`id="professionalEvidence"`,
		"firmware_id",
		"a21-stackchan",
		"m5stack-cores3",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestVoiceChainProfilesCatalogDefaultsToCascadeStepFunAndShowsRealtimeProviders(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/voice-chain-profiles", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response VoiceChainProfilesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != VoiceChainProfileSchemaVersion || response.Service != DeviceRegistryServiceName || !response.HotSwitch {
		t.Fatalf("response metadata = %+v", response)
	}
	if response.SelectedVoiceChainMode != VoiceChainModeCascade {
		t.Fatalf("selected chain mode = %q, want cascade", response.SelectedVoiceChainMode)
	}
	if response.SelectedLLMProfile != "stepfun" {
		t.Fatalf("selected LLM = %q, want stepfun", response.SelectedLLMProfile)
	}
	if response.FixedTTSProfile != "dashscope_qwen_tts_realtime" || response.SelectedTTSProfile != "dashscope_qwen_tts_realtime" {
		t.Fatalf("tts profiles = fixed %q selected %q", response.FixedTTSProfile, response.SelectedTTSProfile)
	}
	seenLLM := map[string]VoiceChainProfileOption{}
	for _, profile := range response.Cascade.LLMProfiles {
		seenLLM[profile.ID] = profile
	}
	if !seenLLM["stepfun"].Recommended || seenLLM["stepfun"].Status != "recommended" {
		t.Fatalf("stepfun option = %+v, want recommended", seenLLM["stepfun"])
	}
	if seenLLM["deepseek"].Recommended || seenLLM["deepseek"].Status != "fallback" {
		t.Fatalf("deepseek option = %+v, want fallback only", seenLLM["deepseek"])
	}
	seenRealtime := map[string]VoiceChainProfileOption{}
	for _, profile := range response.Realtime.Providers {
		seenRealtime[profile.ID] = profile
	}
	for _, want := range []string{"doubao_realtime", "openai_realtime", "doubao_tts_realtime"} {
		if seenRealtime[want].ID == "" {
			t.Fatalf("realtime providers missing %q: %+v", want, response.Realtime.Providers)
		}
	}
	if response.SelectedVoiceCloneProfile != DefaultVoiceCloneProfile || len(response.Voices) == 0 {
		t.Fatalf("voice clone catalog = selected %q voices %+v", response.SelectedVoiceCloneProfile, response.Voices)
	}
	for _, forbidden := range []string{"sk-", "Bearer", "Authorization", "http://", "https://", "/Users/"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("voice chain catalog leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestVoiceChainProfileHotSwitchUpdatesCascadeAndRegistryWithoutChangingModes(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		PublicGatewayURL: "https://a21.example.com",
	})
	handler := server.Handler()

	voiceReq := httptest.NewRequest(http.MethodPost, "/v1/voice-modes", bytes.NewBufferString(`{"voice_mode":"professional"}`))
	voiceRec := httptest.NewRecorder()
	handler.ServeHTTP(voiceRec, voiceReq)
	if voiceRec.Code != http.StatusOK {
		t.Fatalf("voice mode status = %d: %s", voiceRec.Code, voiceRec.Body.String())
	}

	gatewayReq := httptest.NewRequest(http.MethodPost, "/v1/gateway-profiles", bytes.NewBufferString(`{"gateway_profile":"public_wss"}`))
	gatewayRec := httptest.NewRecorder()
	handler.ServeHTTP(gatewayRec, gatewayReq)
	if gatewayRec.Code != http.StatusOK {
		t.Fatalf("gateway profile status = %d: %s", gatewayRec.Code, gatewayRec.Body.String())
	}

	chainReq := httptest.NewRequest(http.MethodPost, "/v1/voice-chain-profiles", bytes.NewBufferString(`{"voice_chain_mode":"cascade","asr_profile":"doubao_asr_realtime","llm_profile":"stepfun","voice_clone_profile":"a21_voice_clone_default"}`))
	chainRec := httptest.NewRecorder()
	handler.ServeHTTP(chainRec, chainReq)
	if chainRec.Code != http.StatusOK {
		t.Fatalf("voice chain status = %d, want 200: %s", chainRec.Code, chainRec.Body.String())
	}
	var chain VoiceChainProfilesResponse
	if err := json.Unmarshal(chainRec.Body.Bytes(), &chain); err != nil {
		t.Fatal(err)
	}
	if chain.SelectedVoiceChainMode != "cascade" || chain.SelectedASRProfile != "doubao_asr_realtime" || chain.SelectedLLMProfile != "stepfun" {
		t.Fatalf("chain selection = %+v", chain)
	}
	if chain.SelectedTTSProfile != "voice_clone_cli" || chain.SelectedVoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("voice clone mapping = selected_tts %q voice %q", chain.SelectedTTSProfile, chain.SelectedVoiceCloneProfile)
	}

	checkVoiceReq := httptest.NewRequest(http.MethodGet, "/v1/voice-modes", nil)
	checkVoiceRec := httptest.NewRecorder()
	handler.ServeHTTP(checkVoiceRec, checkVoiceReq)
	if !bytes.Contains(checkVoiceRec.Body.Bytes(), []byte(`"selected_voice_mode":"professional"`)) {
		t.Fatalf("voice chain changed voice mode: %s", checkVoiceRec.Body.String())
	}
	checkGatewayReq := httptest.NewRequest(http.MethodGet, "/v1/gateway-profiles", nil)
	checkGatewayRec := httptest.NewRecorder()
	handler.ServeHTTP(checkGatewayRec, checkGatewayReq)
	if !bytes.Contains(checkGatewayRec.Body.Bytes(), []byte(`"selected_gateway_profile":"public_wss"`)) {
		t.Fatalf("voice chain changed gateway profile: %s", checkGatewayRec.Body.String())
	}

	server.controlSequence("stackchan-sim-001", "a21-trace-voice-chain", "a21-session-voice-chain", []protocol.ControlEventPayload{{
		State: protocol.ExpressionListening,
		Mode:  protocol.ModeWorkmate,
		Text:  "voice chain registry must not leak this text",
	}})
	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	handler.ServeHTTP(devicesRec, devicesReq)
	if devicesRec.Code != http.StatusOK {
		t.Fatalf("devices status = %d: %s", devicesRec.Code, devicesRec.Body.String())
	}
	var registry DeviceRegistryResponse
	if err := json.Unmarshal(devicesRec.Body.Bytes(), &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("device count = %d: %s", len(registry.Devices), devicesRec.Body.String())
	}
	device := registry.Devices[0]
	if device.CurrentMode != protocol.ModeWorkmate ||
		device.CurrentVoiceMode != "professional" ||
		device.CurrentVoiceChainMode != "cascade" ||
		device.CurrentASRProfile != "doubao_asr_realtime" ||
		device.CurrentLLMProfile != "stepfun" ||
		device.CurrentTTSProfile != "voice_clone_cli" ||
		device.CurrentVoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("device state = %+v", device)
	}
	if strings.Contains(devicesRec.Body.String(), "voice chain registry must not leak this text") {
		t.Fatalf("registry leaked control text: %s", devicesRec.Body.String())
	}
}

func TestVoiceChainRealtimeSelectionUpdatesRealtimeProviderWithoutProviderExecution(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/voice-chain-profiles", bytes.NewBufferString(`{"voice_chain_mode":"realtime","realtime_provider":"openai_realtime"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"selected_voice_chain_mode":"realtime"`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"selected_realtime_provider":"openai_realtime"`)) {
		t.Fatalf("realtime selection missing: %s", rec.Body.String())
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/v1/providers/voice/health", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("health status = %d, want 503 without provider secrets: %s", healthRec.Code, healthRec.Body.String())
	}
	if !bytes.Contains(healthRec.Body.Bytes(), []byte(`"provider":"a21-openai-realtime-voice"`)) {
		t.Fatalf("health did not use selected realtime provider: %s", healthRec.Body.String())
	}
	for _, forbidden := range []string{"sk-", "Bearer", "Authorization"} {
		if strings.Contains(healthRec.Body.String(), forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, healthRec.Body.String())
		}
	}
}

func TestVoiceChainProfilesRejectUnknownValuesWithoutEchoOrStateReset(t *testing.T) {
	server := NewServer()
	handler := server.Handler()
	validReq := httptest.NewRequest(http.MethodPost, "/v1/voice-chain-profiles", bytes.NewBufferString(`{"llm_profile":"stepfun","voice_clone_profile":"a21_voice_clone_default"}`))
	validRec := httptest.NewRecorder()
	handler.ServeHTTP(validRec, validReq)
	if validRec.Code != http.StatusOK {
		t.Fatalf("valid select status = %d: %s", validRec.Code, validRec.Body.String())
	}

	for _, body := range []string{
		`{"voice_chain_mode":"x21_mode"}`,
		`{"asr_profile":"x21_asr"}`,
		`{"llm_profile":"secret_llm"}`,
		`{"realtime_provider":"v21_voice"}`,
		`{"voice_clone_profile":"secret_clone"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/voice-chain-profiles", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 for %s: %s", rec.Code, body, rec.Body.String())
		}
		for _, forbidden := range []string{"x21", "v21", "secret"} {
			if strings.Contains(strings.ToLower(rec.Body.String()), forbidden) {
				t.Fatalf("error body leaked %q for %s: %s", forbidden, body, rec.Body.String())
			}
		}
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/voice-chain-profiles", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"selected_llm_profile":"stepfun"`)) ||
		!bytes.Contains(getRec.Body.Bytes(), []byte(`"selected_voice_clone_profile":"a21_voice_clone_default"`)) ||
		!bytes.Contains(getRec.Body.Bytes(), []byte(`"selected_tts_profile":"voice_clone_cli"`)) {
		t.Fatalf("invalid selection reset valid state: %s", getRec.Body.String())
	}
}

func TestCloudVoiceProfilesCatalogListsPureCloudProfilesWithoutSecrets(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		CloudVoiceEnv: []string{
			"A21_DASHSCOPE_API_KEY=sk-a21-bailian-secret",
			"A21_BAILIAN_QWEN_TTS_MODEL=qwen-tts-secret-model",
			"A21_BAILIAN_QWEN_TTS_VOICE_ID=voice-secret-id",
			"A21_DOUBAO_API_KEY=sk-a21-doubao-secret",
			"A21_DOUBAO_TTS_MODEL=doubao-secret-model",
			"A21_DOUBAO_TTS_VOICE=zh_female_secret_voice",
			"A21_MINIMAX_API_KEY=sk-a21-minimax-secret",
			"A21_MINIMAX_GROUP_ID=minimax-secret-group",
			"A21_MINIMAX_TTS_MODEL=minimax-secret-model",
			"A21_MINIMAX_VOICE_ID=minimax-secret-voice",
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/cloud-voice-profiles", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		SchemaVersion string `json:"schema_version"`
		Service       string `json:"service"`
		Selected      string `json:"selected_cloud_voice_profile"`
		Profiles      []struct {
			ID          string   `json:"id"`
			Vendor      string   `json:"vendor"`
			Family      string   `json:"family"`
			Status      string   `json:"status"`
			Configured  bool     `json:"configured"`
			Default     bool     `json:"default"`
			Realtime    bool     `json:"realtime"`
			RequiredEnv []string `json:"required_env"`
			PresentEnv  []string `json:"present_env"`
			MissingEnv  []string `json:"missing_env"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.cloud_voice_profiles.v1" || response.Service != "a21-gateway" {
		t.Fatalf("response metadata = %+v", response)
	}
	if response.Selected != "a21_doubao_tts_realtime" {
		t.Fatalf("selected cloud voice profile = %q, want a21_doubao_tts_realtime", response.Selected)
	}
	seen := map[string]struct {
		status     string
		configured bool
		realtime   bool
		present    []string
		missing    []string
	}{}
	defaults := 0
	for _, profile := range response.Profiles {
		seen[profile.ID] = struct {
			status     string
			configured bool
			realtime   bool
			present    []string
			missing    []string
		}{status: profile.Status, configured: profile.Configured, realtime: profile.Realtime, present: profile.PresentEnv, missing: profile.MissingEnv}
		if profile.Default {
			defaults++
		}
	}
	for _, want := range []string{
		"a21_doubao_tts_realtime",
		"a21_doubao_voice_clone_tts",
		"a21_bailian_qwen_tts_realtime",
		"a21_bailian_qwen3_tts_vc_realtime",
		"a21_bailian_cosyvoice_realtime",
		"a21_bailian_cosyvoice_clone_tts",
		"a21_bailian_qwen_omni_realtime",
		"a21_minimax_t2a_ws",
		"a21_minimax_t2a_http",
		"a21_minimax_voice_clone_tts",
	} {
		if _, ok := seen[want]; !ok {
			t.Fatalf("catalog missing %q: %+v", want, response.Profiles)
		}
	}
	doubao := seen["a21_doubao_tts_realtime"]
	if !doubao.configured || !doubao.realtime || doubao.status != "static_ready" || len(doubao.missing) != 0 {
		t.Fatalf("doubao realtime profile = %+v, want configured static_ready realtime without missing env", doubao)
	}
	for _, wantEnv := range []string{"A21_DOUBAO_API_KEY", "A21_DOUBAO_TTS_MODEL", "A21_DOUBAO_TTS_VOICE"} {
		if !stringSliceContains(doubao.present, wantEnv) {
			t.Fatalf("doubao present env missing %q: %+v", wantEnv, doubao.present)
		}
	}
	bailian := seen["a21_bailian_qwen_tts_realtime"]
	if !bailian.configured || bailian.status != "catalog_only" {
		t.Fatalf("bailian qwen profile = %+v, want configured catalog_only until adapter lands", bailian)
	}
	if defaults != 1 {
		t.Fatalf("defaults = %d, want exactly one default", defaults)
	}
	for _, forbidden := range []string{
		"sk-a21-bailian-secret",
		"sk-a21-doubao-secret",
		"sk-a21-minimax-secret",
		"qwen-tts-secret-model",
		"voice-secret-id",
		"doubao-secret-model",
		"zh_female_secret_voice",
		"minimax-secret-group",
		"minimax-secret-model",
		"minimax-secret-voice",
		"Authorization",
		"Bearer",
		"http://",
		"https://",
		"/Users/",
	} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("cloud voice catalog leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestCloudVoiceProfileSelectionDoesNotChangeVoiceModeOrGatewayProfile(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		PublicGatewayURL: "https://a21.example.com",
	})
	handler := server.Handler()

	voiceReq := httptest.NewRequest(http.MethodPost, "/v1/voice-modes", bytes.NewBufferString(`{"voice_mode":"professional"}`))
	voiceRec := httptest.NewRecorder()
	handler.ServeHTTP(voiceRec, voiceReq)
	if voiceRec.Code != http.StatusOK {
		t.Fatalf("voice select status = %d, want 200: %s", voiceRec.Code, voiceRec.Body.String())
	}

	gatewayReq := httptest.NewRequest(http.MethodPost, "/v1/gateway-profiles", bytes.NewBufferString(`{"gateway_profile":"public_wss"}`))
	gatewayRec := httptest.NewRecorder()
	handler.ServeHTTP(gatewayRec, gatewayReq)
	if gatewayRec.Code != http.StatusOK {
		t.Fatalf("gateway select status = %d, want 200: %s", gatewayRec.Code, gatewayRec.Body.String())
	}

	cloudReq := httptest.NewRequest(http.MethodPost, "/v1/cloud-voice-profiles", bytes.NewBufferString(`{"cloud_voice_profile":"a21_minimax_t2a_ws"}`))
	cloudRec := httptest.NewRecorder()
	handler.ServeHTTP(cloudRec, cloudReq)
	if cloudRec.Code != http.StatusOK {
		t.Fatalf("cloud voice select status = %d, want 200: %s", cloudRec.Code, cloudRec.Body.String())
	}
	if !bytes.Contains(cloudRec.Body.Bytes(), []byte(`"selected_cloud_voice_profile":"a21_minimax_t2a_ws"`)) {
		t.Fatalf("cloud voice selection missing: %s", cloudRec.Body.String())
	}

	checkVoiceReq := httptest.NewRequest(http.MethodGet, "/v1/voice-modes", nil)
	checkVoiceRec := httptest.NewRecorder()
	handler.ServeHTTP(checkVoiceRec, checkVoiceReq)
	if !bytes.Contains(checkVoiceRec.Body.Bytes(), []byte(`"selected_voice_mode":"professional"`)) {
		t.Fatalf("cloud voice selection changed voice mode: %s", checkVoiceRec.Body.String())
	}

	checkGatewayReq := httptest.NewRequest(http.MethodGet, "/v1/gateway-profiles", nil)
	checkGatewayRec := httptest.NewRecorder()
	handler.ServeHTTP(checkGatewayRec, checkGatewayReq)
	if !bytes.Contains(checkGatewayRec.Body.Bytes(), []byte(`"selected_gateway_profile":"public_wss"`)) {
		t.Fatalf("cloud voice selection changed gateway profile: %s", checkGatewayRec.Body.String())
	}

	server.controlSequence("stackchan-sim-001", "a21-trace-cloud-voice", "a21-session-cloud-voice", []protocol.ControlEventPayload{{
		State: protocol.ExpressionListening,
		Mode:  protocol.ModeWorkmate,
		Text:  "cloud voice profile must not leak this user text",
	}})
	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	handler.ServeHTTP(devicesRec, devicesReq)
	if devicesRec.Code != http.StatusOK {
		t.Fatalf("devices status = %d, want 200: %s", devicesRec.Code, devicesRec.Body.String())
	}
	var registry DeviceRegistryResponse
	if err := json.Unmarshal(devicesRec.Body.Bytes(), &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1: %s", len(registry.Devices), devicesRec.Body.String())
	}
	device := registry.Devices[0]
	if device.CurrentMode != protocol.ModeWorkmate || device.CurrentVoiceMode != "professional" || device.CurrentCloudVoiceProfile != "a21_minimax_t2a_ws" {
		t.Fatalf("device state = %+v, want transport mode workmate, voice_mode professional, cloud voice profile minimax", device)
	}
	if strings.Contains(devicesRec.Body.String(), "cloud voice profile must not leak this user text") {
		t.Fatalf("registry leaked control text: %s", devicesRec.Body.String())
	}
}

func TestCloudVoiceProfilesRejectUnknownAndLegacyValuesWithoutEcho(t *testing.T) {
	server := NewServer()
	handler := server.Handler()
	for _, body := range []string{
		`{"cloud_voice_profile":"x21_voice"}`,
		`{"cloud_voice_profile":"pure_cloud"}`,
		`{"cloud_voice_profile":"unknown-secret-profile"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/cloud-voice-profiles", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 for %s: %s", rec.Code, body, rec.Body.String())
		}
		for _, forbidden := range []string{"x21_voice", "pure_cloud", "unknown-secret-profile", "secret"} {
			if strings.Contains(rec.Body.String(), forbidden) {
				t.Fatalf("error body leaked %q for %s: %s", forbidden, body, rec.Body.String())
			}
		}
	}
	getReq := httptest.NewRequest(http.MethodGet, "/v1/cloud-voice-profiles", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"selected_cloud_voice_profile":"a21_doubao_tts_realtime"`)) {
		t.Fatalf("invalid selection changed default profile: %s", getRec.Body.String())
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestVoiceModesCatalogDefaultsToRoleplayAndListsProfessional(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/voice-modes", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		SchemaVersion string `json:"schema_version"`
		Selected      string `json:"selected_voice_mode"`
		Ritual        struct {
			Mode             string `json:"mode"`
			ScreenLabel      string `json:"screen_label"`
			V21Allowed       bool   `json:"v21_allowed"`
			PhysicalAccepted bool   `json:"physical_accepted"`
		} `json:"selected_ritual"`
		Modes []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Default bool   `json:"default"`
			Ritual  struct {
				Mode             string `json:"mode"`
				PhysicalAccepted bool   `json:"physical_accepted"`
			} `json:"ritual"`
		} `json:"modes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.voice_modes.v1" || response.Selected != "roleplay" {
		t.Fatalf("catalog = %+v", response)
	}
	if response.Ritual.Mode != "roleplay" ||
		response.Ritual.ScreenLabel != "A21" ||
		response.Ritual.V21Allowed ||
		response.Ritual.PhysicalAccepted {
		t.Fatalf("selected ritual = %+v, want roleplay no-v21 no-physical", response.Ritual)
	}
	seen := map[string]string{}
	defaults := 0
	for _, mode := range response.Modes {
		seen[mode.ID] = mode.Status
		if mode.Default {
			defaults++
		}
		if mode.Ritual.Mode != mode.ID || mode.Ritual.PhysicalAccepted {
			t.Fatalf("mode ritual = %+v for mode %+v, want matching non-physical ritual", mode.Ritual, mode)
		}
	}
	if seen["roleplay"] != "available" || seen["professional"] != "available" || defaults != 1 {
		t.Fatalf("voice modes = %+v, statuses=%v defaults=%d", response.Modes, seen, defaults)
	}
	if _, ok := seen["dialogue"]; ok {
		t.Fatalf("catalog still exposes dialogue as a user product mode: %+v", response.Modes)
	}
	if _, ok := seen["edge_cloud"]; ok {
		t.Fatalf("catalog still exposes old route selector as product mode: %+v", response.Modes)
	}
	if _, ok := seen["pure_cloud"]; ok {
		t.Fatalf("catalog still exposes planned pure-cloud spike as product mode: %+v", response.Modes)
	}
}

func TestVoiceModeSelectionProfessionalReturnsRitualContract(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/voice-modes", bytes.NewBufferString(`{"voice_mode":"professional"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response VoiceModeCatalogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	ritual := response.SelectedRitual
	if response.SelectedVoiceMode != VoiceModeProfessional ||
		ritual.Mode != VoiceModeProfessional ||
		ritual.ScreenLabel != "PRO" ||
		ritual.CueText != v21adapter.ProfessionalCheckingFeedbackText ||
		ritual.Expression != protocol.ExpressionProfessional ||
		ritual.TraceMarker != "professional.checking_feedback.sent" ||
		ritual.WorkspacePolicy != "professional_only" ||
		!ritual.V21Allowed ||
		ritual.PhysicalAccepted {
		t.Fatalf("professional ritual = %+v selected=%s", ritual, response.SelectedVoiceMode)
	}
	for _, forbidden := range []string{"http://", "https://", "/Users/", "sk-", "secret", "source_text", "evidence_body"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("professional ritual leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestVoiceModeSelectionAcceptsDialogueAliasAsRoleplayWithoutChangingLegacyTransportMode(t *testing.T) {
	server := NewServer()
	selectReq := httptest.NewRequest(http.MethodPost, "/v1/voice-modes", bytes.NewBufferString(`{"voice_mode":"dialogue"}`))
	selectRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(selectRec, selectReq)
	if selectRec.Code != http.StatusOK {
		t.Fatalf("select status = %d, want 200: %s", selectRec.Code, selectRec.Body.String())
	}

	server.controlSequence("stackchan-sim-001", "a21-trace-voice-mode", "a21-session-voice-mode", []protocol.ControlEventPayload{{
		State: protocol.ExpressionListening,
		Mode:  protocol.ModeWorkmate,
		Text:  "selected voice mode must not rewrite product mode",
	}})

	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(devicesRec, devicesReq)
	if devicesRec.Code != http.StatusOK {
		t.Fatalf("devices status = %d, want 200: %s", devicesRec.Code, devicesRec.Body.String())
	}
	var registry DeviceRegistryResponse
	if err := json.Unmarshal(devicesRec.Body.Bytes(), &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1: %s", len(registry.Devices), devicesRec.Body.String())
	}
	device := registry.Devices[0]
	if device.CurrentMode != protocol.ModeWorkmate || device.CurrentVoiceMode != "roleplay" {
		t.Fatalf("device state = %+v, want legacy transport mode workmate and voice_mode roleplay", device)
	}
	if strings.Contains(devicesRec.Body.String(), "selected voice mode must not rewrite product mode") {
		t.Fatalf("registry leaked control text: %s", devicesRec.Body.String())
	}
}

func TestRoleplayProfileEndpointPersistsScenarioVoiceCloneAndRedactsMemory(t *testing.T) {
	t.Setenv("A21_MEMORY_USER_PREFERENCES", "偏好短句")
	t.Setenv("A21_MEMORY_SESSION_NOTES", "当前任务是座舱 PRD 评审\nhttp://secret.example/leak\n/Users/me/a21-secret.txt")
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{"scenario":"boss_challenge","voice_clone_profile":"a21_voice_clone_default"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RoleplayProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.roleplay_profile.v1" ||
		response.SelectedRoleplayProfile != "a21_roleplay_default" ||
		response.SelectedScenario != "boss_challenge" ||
		response.SelectedVoiceCloneProfile != "a21_voice_clone_default" {
		t.Fatalf("roleplay profile = %+v", response)
	}
	if !response.Runtime.PromptComposed || response.Runtime.PromptStored || response.Runtime.MemoryTextStored || response.Runtime.V21Executed || response.Runtime.ProfessionalRouteAllowed {
		t.Fatalf("runtime redaction/route = %+v", response.Runtime)
	}
	if response.Runtime.MemoryHintCount != 2 || !response.Memory.PromptInputReady {
		t.Fatalf("memory = %+v runtime=%+v, want two safe hints", response.Memory, response.Runtime)
	}
	for _, forbidden := range []string{"偏好短句", "座舱 PRD", "secret.example", "/Users/me", "http://", "https://"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("roleplay profile leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	chainReq := httptest.NewRequest(http.MethodGet, "/v1/voice-chain-profiles", nil)
	chainRec := httptest.NewRecorder()
	handler.ServeHTTP(chainRec, chainReq)
	if !strings.Contains(chainRec.Body.String(), `"selected_voice_clone_profile":"a21_voice_clone_default"`) ||
		!strings.Contains(chainRec.Body.String(), `"selected_tts_profile":"voice_clone_cli"`) {
		t.Fatalf("voice clone selection did not reach voice chain: %s", chainRec.Body.String())
	}
}

func TestRoleplayProfileEndpointSetsRuntimeMemoryHintsForFastCompanion(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{
		"scenario":"desk_mouthpiece",
		"voice_clone_profile":"a21_voice_clone_default",
		"memory_hints":["只用短句接话","http://secret.example/leak","/Users/me/a21-secret.txt"]
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var profile RoleplayProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Runtime.MemoryHintCount != 1 || !profile.Memory.Configured || !profile.Memory.PromptInputReady {
		t.Fatalf("memory = %+v runtime=%+v, want one safe runtime hint", profile.Memory, profile.Runtime)
	}
	if len(profile.Memory.Findings) == 0 {
		t.Fatalf("memory findings = none, want unsafe hint finding")
	}
	for _, forbidden := range []string{"只用短句接话", "secret.example", "/Users/me", "http://", "https://"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("roleplay profile leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	turnReq := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(`{
		"device_id":"stackchan-sim-001",
		"mode":"roleplay",
		"trace_id":"a21-trace-runtime-memory-001",
		"session_id":"a21-session-runtime-memory-001",
		"local_audio":{"asr_provider":"mock_asr","first_partial_ms":33,"final_transcript_chars":9}
	}`))
	turnRec := httptest.NewRecorder()
	handler.ServeHTTP(turnRec, turnReq)
	if turnRec.Code != http.StatusOK {
		t.Fatalf("fast companion status = %d, want 200: %s", turnRec.Code, turnRec.Body.String())
	}
	var turn struct {
		Roleplay RoleplayRuntimeSummary `json:"roleplay"`
	}
	if err := json.Unmarshal(turnRec.Body.Bytes(), &turn); err != nil {
		t.Fatal(err)
	}
	if turn.Roleplay.MemoryHintCount != 1 || !turn.Roleplay.MemoryPromptInputReady || turn.Roleplay.V21Executed {
		t.Fatalf("fast companion roleplay = %+v, want one ready memory hint and no V21", turn.Roleplay)
	}
	if strings.Contains(turnRec.Body.String(), "只用短句接话") {
		t.Fatalf("fast companion response leaked memory hint: %s", turnRec.Body.String())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-runtime-memory-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	if !strings.Contains(traceRec.Body.String(), "roleplay.memory.ready") {
		t.Fatalf("trace missing roleplay.memory.ready: %s", traceRec.Body.String())
	}
	if strings.Contains(traceRec.Body.String(), "只用短句接话") {
		t.Fatalf("trace leaked memory hint: %s", traceRec.Body.String())
	}

	clearReq := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{"clear_memory":true}`))
	clearRec := httptest.NewRecorder()
	handler.ServeHTTP(clearRec, clearReq)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want 200: %s", clearRec.Code, clearRec.Body.String())
	}
	var cleared RoleplayProfileResponse
	if err := json.Unmarshal(clearRec.Body.Bytes(), &cleared); err != nil {
		t.Fatal(err)
	}
	if cleared.Runtime.MemoryHintCount != 0 || cleared.Memory.Configured || cleared.Memory.PromptInputReady {
		t.Fatalf("cleared memory = %+v runtime=%+v, want empty runtime memory", cleared.Memory, cleared.Runtime)
	}
}

func TestProfessionalWorkspaceEndpointPersistsQueryScopeAndRedactsDocumentBoundary(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_demo","workspace_id":"a21_workspace_demo","query_scope":"personal_plus_public"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response ProfessionalWorkspaceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.professional_workspace.v1" ||
		response.AdapterContractVersion != "a21.v21_adapter_query.v2" ||
		response.SelectedUserID != "a21_user_demo" ||
		response.SelectedWorkspaceID != "a21_workspace_demo" ||
		response.SelectedQueryScope != "personal_plus_public" ||
		response.PrivacyScope != "professional_only" {
		t.Fatalf("workspace response = %+v", response)
	}
	if !response.Runtime.QueryScopeReady || response.Runtime.UploadAPIReady || response.Runtime.IndexingAPIReady || response.V21ExecutionAllowed {
		t.Fatalf("runtime gates = %+v", response.Runtime)
	}
	if response.Redaction.DocumentTextStored || response.Redaction.QueryTextStored || response.Redaction.RetrievedTextStored ||
		response.Redaction.FullURLStored || response.Redaction.LocalPathStored || response.Redaction.CredentialValueStored ||
		response.Redaction.ProviderOutputStored || response.Redaction.VoiceTranscriptStored {
		t.Fatalf("workspace redaction = %+v", response.Redaction)
	}
	for _, forbidden := range []string{"http://", "https://", "/Users/", "secret", "api_key", "raw document", "retrieved text"} {
		if strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("professional workspace leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	badReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"workspace_id":"https://secret.example/workspace","query_scope":"personal_plus_public"}`))
	badRec := httptest.NewRecorder()
	handler.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("bad workspace status = %d, want 400: %s", badRec.Code, badRec.Body.String())
	}
}

func TestWorkspaceUploadJobsLifecycleIsNoExecuteAndRedacted(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"user_id":"a21_user_demo",
		"workspace_id":"a21_workspace_demo",
		"source_scope":"personal",
		"source_kind":"upload",
		"document_label":"PRD pack",
		"content_type":"application/pdf",
		"size_bytes":2048,
		"trace_id":"a21-trace-upload-job",
		"session_id":"a21-session-upload-job",
		"device_id":"stackchan-sim-001"
	}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200: %s", createRec.Code, createRec.Body.String())
	}
	var createResp WorkspaceUploadJobsResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if createResp.SchemaVersion != "a21.gateway.workspace_upload_jobs.v1" || len(createResp.Jobs) != 1 {
		t.Fatalf("create response = %+v", createResp)
	}
	job := createResp.Jobs[0]
	if job.JobID == "" ||
		job.UserID != "a21_user_demo" ||
		job.WorkspaceID != "a21_workspace_demo" ||
		job.SourceScope != "personal" ||
		job.SourceKind != "upload" ||
		job.DocumentLabel != "PRD pack" ||
		job.ContentType != "application/pdf" ||
		job.Status != "accepted_no_execute" ||
		job.IndexStatus != "not_started_no_execute" ||
		!job.UploadAPIReady ||
		!job.ImportAPIReady ||
		job.IndexingAPIReady ||
		job.ExecutionStarted ||
		!job.RetryAllowed ||
		!job.DeleteAllowed {
		t.Fatalf("job = %+v", job)
	}
	if job.Redaction.DocumentTextStored || job.Redaction.DocumentBytesStored || job.Redaction.Base64PayloadStored ||
		job.Redaction.ImportURLStored || job.Redaction.LocalPathStored || job.Redaction.CredentialValueStored ||
		job.Redaction.ProviderOutputStored {
		t.Fatalf("job redaction = %+v", job.Redaction)
	}

	failReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"mark_failed"}`))
	failRec := httptest.NewRecorder()
	handler.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusOK {
		t.Fatalf("fail status = %d: %s", failRec.Code, failRec.Body.String())
	}
	if !strings.Contains(failRec.Body.String(), `"status":"failed"`) || !strings.Contains(failRec.Body.String(), `"index_status":"failed_no_execute"`) {
		t.Fatalf("fail response = %s", failRec.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"retry"}`))
	retryRec := httptest.NewRecorder()
	handler.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry status = %d: %s", retryRec.Code, retryRec.Body.String())
	}
	if !strings.Contains(retryRec.Body.String(), `"attempt":2`) || !strings.Contains(retryRec.Body.String(), `"status":"accepted_no_execute"`) {
		t.Fatalf("retry response = %s", retryRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"delete"}`))
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), `"status":"deleted"`) || !strings.Contains(deleteRec.Body.String(), `"document_label":"deleted"`) {
		t.Fatalf("delete response = %s", deleteRec.Body.String())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-upload-job", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"workspace.upload_job.accepted_no_execute", "workspace.index_job.not_started_no_execute", "workspace.upload_job.deleted"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"raw document", "http://", "https://", "/Users/", "secret", "api_key", "content_base64"} {
		if strings.Contains(strings.ToLower(createRec.Body.String()+failRec.Body.String()+retryRec.Body.String()+deleteRec.Body.String()+traceRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("workspace upload job leaked %q", forbidden)
		}
	}
}

func TestWorkspaceUploadJobsRejectRawPayloadFields(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"workspace_id":"a21_workspace_demo",
		"document_label":"secret.pdf",
		"content_base64":"UkFX",
		"import_url":"https://secret.example/file.pdf"
	}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"UkFX", "secret.example", "secret.pdf"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("rejection leaked %q: %s", forbidden, rec.Body.String())
		}
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
	profileReq := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{"voice_clone_profile":"a21_voice_clone_default","memory_hints":["角色语气只给短句"]}`))
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
	if !strings.Contains(captured.TextPrompt, "Memory Hints") ||
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

func TestFastCompanionVoicePipelineRecordsProviderFallbackObservability(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return fallbackReportingXiaozhiPipelineRunner{}
	}
	handler := server.Handler()
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"workmate",
		"trace_id":"a21-trace-fast-fallback",
		"session_id":"a21-session-fast-fallback",
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
	var response struct {
		Status                     string `json:"status"`
		TextStreamProvider         string `json:"text_stream_provider"`
		TextStreamFallbackUsed     bool   `json:"text_stream_fallback_used"`
		TextStreamFallbackProvider string `json:"text_stream_fallback_provider"`
		TextStreamFallbackReason   string `json:"text_stream_fallback_reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "pipeline_completed" ||
		response.TextStreamProvider != "a21_voice_fallback" ||
		!response.TextStreamFallbackUsed ||
		response.TextStreamFallbackProvider != "a21_voice_fallback" ||
		response.TextStreamFallbackReason != "primary_failed" {
		t.Fatalf("fallback response = %+v", response)
	}
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", frameBase64, "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("fast companion fallback response leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-fallback", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fallback.used", "provider.failover", "fast_companion.voice_pipeline.completed"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(traceRec.Body.String(), forbidden) {
			t.Fatalf("fallback trace leaked %q: %s", forbidden, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_provider_failover_total 1", "a21_fallback_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestFastCompanionVoicePipelineUnavailableEntersLocalFallbackState(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return unavailableXiaozhiPipelineRunner{}
	}
	handler := server.Handler()
	frameBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1, 0}, 960))
	req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(fmt.Sprintf(`{
		"device_id":"stackchan-sim-001",
		"mode":"workmate",
		"trace_id":"a21-trace-fast-local-fallback",
		"session_id":"a21-session-fast-local-fallback",
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
	var response struct {
		Status string              `json:"status"`
		Mode   protocol.Mode       `json:"mode"`
		Events []protocol.Envelope `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "local_fallback" || response.Mode != protocol.ModeLocalFallback {
		t.Fatalf("response status/mode = %s/%s, want local_fallback", response.Status, response.Mode)
	}
	var fallback protocol.ControlEventPayload
	for _, event := range response.Events {
		if event.Kind != protocol.KindControlEvent {
			continue
		}
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State == protocol.ExpressionLocalFallback {
			fallback = payload
		}
	}
	if fallback.State != protocol.ExpressionLocalFallback ||
		fallback.Mode != protocol.ModeLocalFallback ||
		!fallback.Final ||
		!strings.Contains(fallback.Text, "外部大脑连不上") ||
		!strings.Contains(fallback.Text, "我还在") {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	for _, forbidden := range []string{frameBase64, "fixture transcript", "provider output", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("local fallback response leaked %q: %s", forbidden, rec.Body.String())
		}
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-fast-local-fallback", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fast_companion.voice_pipeline.unavailable", "local_fallback.entered", "control.local_fallback.sent"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_fallback_total 1") {
		t.Fatalf("metrics missing local fallback count:\n%s", metricsRec.Body.String())
	}
	if strings.Contains(metricsRec.Body.String(), "a21_provider_failover_total 1") {
		t.Fatalf("local fallback incremented provider failover count:\n%s", metricsRec.Body.String())
	}

	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	handler.ServeHTTP(devicesRec, devicesReq)
	for _, want := range []string{`"current_mode":"local_fallback"`, `"current_expression":"local_fallback"`} {
		if !strings.Contains(devicesRec.Body.String(), want) {
			t.Fatalf("devices missing %q: %s", want, devicesRec.Body.String())
		}
	}
}

func TestFastCompanionHybridRejectsUnsupportedModesAndMissingLocalAudio(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: provider,
		V21Client:     v21,
	})
	handler := server.Handler()
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "unsupported mode",
			body: `{"device_id":"stackchan-sim-001","mode":"public","local_audio":{"asr_provider":"mock_asr"}}`,
		},
		{
			name: "missing local audio provider",
			body: `{"device_id":"stackchan-sim-001","mode":"workmate","local_audio":{"first_partial_ms":42}}`,
		},
		{
			name: "negative local audio counters",
			body: `{"device_id":"stackchan-sim-001","mode":"companion","local_audio":{"asr_provider":"mock_asr","first_partial_ms":-1}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/fast-companion/turn", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
			if provider.startCalls != 0 {
				t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}
		})
	}
}

func TestFastCompanionHybridKeepsProfessionalRealtimeOutAndV21EvidenceIn(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true, StreamID: "rt-stream-pro"},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	handler := server.Handler()

	realtimeReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-boundary","session_id":"a21-session-pro-boundary"}`))
	realtimeRec := httptest.NewRecorder()
	handler.ServeHTTP(realtimeRec, realtimeReq)

	if realtimeRec.Code != http.StatusBadRequest {
		t.Fatalf("realtime status = %d, want 400: %s", realtimeRec.Code, realtimeRec.Body.String())
	}
	if provider.startCalls != 0 {
		t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
	}

	proReq := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-boundary","session_id":"a21-session-pro-boundary"}`))
	proRec := httptest.NewRecorder()
	handler.ServeHTTP(proRec, proReq)

	if proRec.Code != http.StatusOK {
		t.Fatalf("professional status = %d, want 200: %s", proRec.Code, proRec.Body.String())
	}
	if v21.calls != 1 {
		t.Fatalf("v21 calls = %d, want 1", v21.calls)
	}
	var response MockTurnResponse
	if err := json.Unmarshal(proRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.Mode != protocol.ModeProfessional || answer.State != protocol.ExpressionSpeaking || len(answer.Evidence) == 0 {
		t.Fatalf("professional answer = %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-boundary", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"v21.query.start", "v21.query.first_result"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartUsesVoiceProviderAndRecordsMetrics(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{Kind: providers.VoiceEventThinking, Text: "provider realtime thinking"},
				{Kind: providers.VoiceEventSpeaking, Text: "provider realtime speaking", Final: true, StreamID: "rt-stream-001"},
			},
		},
	})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-rt-001","session_id":"a21-session-rt-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Provider != "scripted" || response.Status != "completed" {
		t.Fatalf("response provider/status = %q/%q", response.Provider, response.Status)
	}
	if response.TraceID != "a21-trace-rt-001" || response.SessionID != "a21-session-rt-001" || response.DeviceID != "stackchan-001" {
		t.Fatalf("response identity = %+v", response)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(response.Events))
	}
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[1].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "rt-stream-001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-001", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"realtime.session.start.received", "provider.start_turn.start", "provider.start_turn.first_event", "provider.start_turn.end"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_session_total 1", "a21_voice_provider_start_turn_ms_count 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartMapsProviderAudioToPlaybackChunk(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		VoiceProvider: scriptedVoiceProvider{
			events: []providers.VoiceEvent{
				{
					Kind:     providers.VoiceEventSpeaking,
					Final:    true,
					StreamID: "rt-audio-stream-001",
					Audio: &providers.VoiceAudioChunk{
						Codec:        string(protocol.AudioCodecPCMS16LE),
						SampleRateHz: 16000,
						Channels:     1,
						DurationMS:   20,
						DataBase64:   "AAAA",
					},
				},
			},
		},
	})
	handler := server.Handler()
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"播一小段","mode":"workmate","trace_id":"a21-trace-rt-audio","session_id":"a21-session-rt-audio"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want control plus playback: %+v", len(response.Events), response.Events)
	}
	if response.Events[0].Kind != protocol.KindControlEvent {
		t.Fatalf("event 0 kind = %q, want control", response.Events[0].Kind)
	}
	if response.Events[1].Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("event 1 kind = %q, want audio playback", response.Events[1].Kind)
	}
	var playback protocol.AudioPlaybackChunk
	if err := json.Unmarshal(response.Events[1].Payload, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.StreamID != "rt-audio-stream-001" || playback.Codec != protocol.AudioCodecPCMS16LE || playback.SampleRateHz != 16000 || playback.Channels != 1 || playback.DurationMS != 20 || playback.DataBase64 != "AAAA" {
		t.Fatalf("playback = %+v", playback)
	}
	if response.Events[1].TraceID != "a21-trace-rt-audio" || response.Events[1].SessionID != "a21-session-rt-audio" {
		t.Fatalf("playback trace/session = %q/%q", response.Events[1].TraceID, response.Events[1].SessionID)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-audio", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "audio.playback.chunk.sent") {
		t.Fatalf("trace missing audio playback marker: %s", traceRec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_audio_playback_chunk_total 1") {
		t.Fatalf("metrics missing playback count:\n%s", metricsRec.Body.String())
	}
}

func TestRealtimeSessionCancelUsesActiveStreamAndRecordsMetrics(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "provider realtime speaking", Final: true, StreamID: "rt-stream-002"},
		},
	}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider})
	handler := server.Handler()
	startReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"先说，我在","mode":"workmate","trace_id":"a21-trace-rt-002","session_id":"a21-session-rt-002"}`))
	handler.ServeHTTP(httptest.NewRecorder(), startReq)

	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/realtime/session/cancel", bytes.NewBufferString(`{"device_id":"stackchan-001","mode":"workmate","trace_id":"a21-trace-rt-002","session_id":"a21-session-rt-002","reason":"barge_in"}`))
	cancelRec := httptest.NewRecorder()
	handler.ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", cancelRec.Code, cancelRec.Body.String())
	}
	if provider.cancelRequest.StreamID != "rt-stream-002" {
		t.Fatalf("cancel stream = %q, want active stream", provider.cancelRequest.StreamID)
	}
	if provider.cancelRequest.Reason != providers.CancelBargeIn {
		t.Fatalf("cancel reason = %q", provider.cancelRequest.Reason)
	}
	var response RealtimeSessionResponse
	if err := json.Unmarshal(cancelRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "cancelled" || len(response.Events) != 1 {
		t.Fatalf("response = %+v", response)
	}
	var interrupted protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[0].Payload, &interrupted); err != nil {
		t.Fatal(err)
	}
	if interrupted.State != protocol.ExpressionInterrupted || interrupted.StreamID != "rt-stream-002" {
		t.Fatalf("interrupted payload = %+v", interrupted)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-rt-002", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"realtime.session.cancel.received", "provider.cancel.start", "provider.cancel.end"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_realtime_session_cancel_total 1", "a21_voice_provider_cancel_ms_count 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestRealtimeSessionStartRejectsProfessionalModeWithoutCallingProviders(t *testing.T) {
	provider := &capturingVoiceProvider{
		startEvents: []providers.VoiceEvent{
			{Kind: providers.VoiceEventSpeaking, Text: "should not run", Final: true, StreamID: "rt-stream-pro"},
		},
	}
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{VoiceProvider: provider, V21Client: v21})
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/session", bytes.NewBufferString(`{"device_id":"stackchan-001","text":"查一下证据","mode":"professional"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if provider.startCalls != 0 {
		t.Fatalf("provider start calls = %d, want 0", provider.startCalls)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}
	if !strings.Contains(rec.Body.String(), "professional path") {
		t.Fatalf("body = %q, want professional path guidance", rec.Body.String())
	}
}

func TestProfessionalModeUsesV21AdapterEvidence(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21adapter.NewMockClient(),
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-001","session_id":"a21-session-pro-001"}`)
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
	if len(response.Events) != 4 {
		t.Fatalf("events = %d, want 4", len(response.Events))
	}
	var pro protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[1].Payload, &pro); err != nil {
		t.Fatal(err)
	}
	if pro.State != protocol.ExpressionProfessional || pro.Mode != protocol.ModeProfessional {
		t.Fatalf("professional payload = %+v", pro)
	}
	var checking protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[2].Payload, &checking); err != nil {
		t.Fatal(err)
	}
	if checking.State != protocol.ExpressionThinking || checking.Mode != protocol.ModeProfessional || !strings.Contains(checking.Text, "我在查") {
		t.Fatalf("checking payload = %+v", checking)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[3].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.State != protocol.ExpressionSpeaking || answer.Mode != protocol.ModeProfessional {
		t.Fatalf("answer payload = %+v", answer)
	}
	if answer.Text == "" || len(answer.Evidence) == 0 || len(answer.ScreenCards) == 0 {
		t.Fatalf("professional answer missing evidence fields: %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-001", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	if !strings.Contains(traceRec.Body.String(), "v21.query.start") || !strings.Contains(traceRec.Body.String(), "v21.query.first_result") {
		t.Fatalf("trace missing v21 markers: %s", traceRec.Body.String())
	}
}

func TestProfessionalModeEmitsCheckingFeedbackBeforeV21Query(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client: v21adapter.NewMockClient(),
	})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-checking","session_id":"a21-session-pro-checking"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	checkingEventIndex := -1
	for i, event := range response.Events {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Mode == protocol.ModeProfessional && payload.State == protocol.ExpressionThinking && strings.Contains(payload.Text, "我在查") {
			checkingEventIndex = i
			break
		}
	}
	if checkingEventIndex < 0 {
		t.Fatalf("professional response missing checking feedback event: %+v", response.Events)
	}
	if checkingEventIndex >= len(response.Events)-1 {
		t.Fatalf("checking feedback must be before final evidence answer: index=%d events=%d", checkingEventIndex, len(response.Events))
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-checking", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traces); err != nil {
		t.Fatal(err)
	}
	checkingAt, ok := traceEventAtMS(traces.Events, "professional.checking_feedback.sent")
	if !ok {
		t.Fatalf("trace missing professional.checking_feedback.sent: %s", traceRec.Body.String())
	}
	v21StartAt, ok := traceEventAtMS(traces.Events, "v21.query.start")
	if !ok {
		t.Fatalf("trace missing v21.query.start: %s", traceRec.Body.String())
	}
	if checkingAt > v21StartAt {
		t.Fatalf("checking feedback at %d must be before v21 query start at %d", checkingAt, v21StartAt)
	}
	if v21StartAt-checkingAt > 1200 {
		t.Fatalf("checking feedback to v21 start = %dms, want <=1200ms", v21StartAt-checkingAt)
	}
}

func TestProfessionalModeSendsExplicitV21PlaceholderContract(t *testing.T) {
	v21 := &capturingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	handler := server.Handler()
	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_contract","workspace_id":"a21_workspace_contract","query_scope":"personal_plus_public"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下证据","mode":"professional","trace_id":"a21-trace-pro-contract","session_id":"a21-session-pro-contract"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if v21.request.Mode != "professional" ||
		v21.request.PrivacyScope != "professional_only" ||
		v21.request.DeviceID != "stackchan-sim-001" ||
		v21.request.UserID != "a21_user_contract" ||
		v21.request.WorkspaceID != "a21_workspace_contract" ||
		v21.request.QueryScope != "personal_plus_public" ||
		v21.request.LatencyProfile != "fast_first" ||
		v21.request.AnswerStyle != "voice_first_with_citations" ||
		v21.request.MaxFirstResponseMS != 1200 {
		t.Fatalf("gateway sent incomplete V21 professional placeholder contract: %+v", v21.request)
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var answer protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Evidence) != 1 || len(answer.ScreenCards) != 1 || len(answer.FollowUps) != 1 {
		t.Fatalf("professional response missing evidence/cards/follow-ups: %+v", answer)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-contract", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	var traces TraceResponse
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traces); err != nil {
		t.Fatal(err)
	}
	checkingAt, ok := traceEventAtMS(traces.Events, "professional.checking_feedback.sent")
	if !ok {
		t.Fatalf("trace missing checking feedback marker: %s", traceRec.Body.String())
	}
	v21StartAt, ok := traceEventAtMS(traces.Events, "v21.query.start")
	if !ok {
		t.Fatalf("trace missing V21 start marker: %s", traceRec.Body.String())
	}
	for _, want := range []string{"professional.workspace.ready", "professional.query_scope.personal_plus_public"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"a21_user_contract", "a21_workspace_contract", "查一下证据"} {
		if strings.Contains(traceRec.Body.String(), forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceRec.Body.String())
		}
	}
	if v21StartAt-checkingAt > 1200 {
		t.Fatalf("placeholder boundary = %dms, want <=1200ms", v21StartAt-checkingAt)
	}
}

func TestWorkmateModeDoesNotCallV21Adapter(t *testing.T) {
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"先说，我在","mode":"workmate"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls = %d, want 0", v21.calls)
	}
}

func TestOfficePrivacyModesDoNotCallV21Adapter(t *testing.T) {
	for _, mode := range []protocol.Mode{protocol.ModeWorkmate, protocol.ModePublic, protocol.ModePrivate, protocol.ModeFocus, protocol.ModeMuted} {
		t.Run(string(mode), func(t *testing.T) {
			v21 := &countingV21Client{}
			server := NewServerWithOptions(ServerOptions{V21Client: v21})
			body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"不要进专业检索","mode":"` + string(mode) + `"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}
		})
	}
}

func TestOrdinaryOfficeModesDoNotExposeProfessionalEvidenceOrV21Trace(t *testing.T) {
	for _, mode := range []protocol.Mode{protocol.ModePublic, protocol.ModePrivate, protocol.ModeFocus} {
		t.Run(string(mode), func(t *testing.T) {
			v21 := &countingV21Client{}
			server := NewServerWithOptions(ServerOptions{V21Client: v21})
			body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"private raw evidence body must stay out","mode":"` + string(mode) + `","trace_id":"a21-trace-ordinary-` + string(mode) + `","session_id":"a21-session-ordinary-` + string(mode) + `"}`)
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
			for _, event := range response.Events {
				var payload protocol.ControlEventPayload
				if err := json.Unmarshal(event.Payload, &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Mode == protocol.ModeProfessional || len(payload.Evidence) > 0 || len(payload.ScreenCards) > 0 || len(payload.SpeechBlocks) > 0 {
					t.Fatalf("ordinary %s mode exposed professional payload: %+v", mode, payload)
				}
				if strings.Contains(payload.Text, "raw evidence body") {
					t.Fatalf("ordinary %s mode echoed sensitive text: %+v", mode, payload)
				}
			}
			if v21.calls != 0 {
				t.Fatalf("v21 calls = %d, want 0", v21.calls)
			}

			traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-ordinary-"+string(mode), nil)
			traceRec := httptest.NewRecorder()
			server.Handler().ServeHTTP(traceRec, traceReq)
			if traceRec.Code != http.StatusOK {
				t.Fatalf("trace status = %d, want 200: %s", traceRec.Code, traceRec.Body.String())
			}
			for _, forbidden := range []string{"v21.query", "professional.", "raw evidence body"} {
				if strings.Contains(traceRec.Body.String(), forbidden) {
					t.Fatalf("ordinary %s trace leaked %q: %s", mode, forbidden, traceRec.Body.String())
				}
			}
		})
	}
}

func TestProfessionalModeV21FailureIsHonestFallback(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: failingV21Client{}})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-fail","session_id":"a21-session-pro-fail"}`)
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
	var fallback protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &fallback); err != nil {
		t.Fatal(err)
	}
	if fallback.State != protocol.ExpressionError || fallback.Mode != protocol.ModeProfessional {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	if !strings.Contains(fallback.Text, "V21 现在没接上") {
		t.Fatalf("fallback text = %q", fallback.Text)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-fail", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "v21.query.error") {
		t.Fatalf("trace missing v21 error marker: %s", traceRec.Body.String())
	}
}

func TestProfessionalModeV21StatusFailureRecordsRedactedReasonMarkers(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "raw query 查一下语音唤醒误触发 http://127.0.0.1:18080/internal", http.StatusBadGateway)
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-status","session_id":"a21-session-pro-status"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-status", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	for _, want := range []string{"v21.query.error", "v21.query.error.upstream_status", "v21.query.error.status_5xx", "v21.query.utterance.length_"} {
		if !strings.Contains(traceBody, want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", adapter.URL, "127.0.0.1:18080", "/internal"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
	}
}

func TestProfessionalModeV21JSONErrorBodyOverridesStatusMarker(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{
			"code":"no_evidence",
			"status_class":"status_4xx",
			"detail":"raw query 查一下语音唤醒误触发 http://127.0.0.1:18080/internal"
		}`))
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-json-error","session_id":"a21-session-pro-json-error"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-json-error", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	for _, want := range []string{"v21.query.error", "v21.query.error.no_evidence", "v21.query.error.status_4xx"} {
		if !strings.Contains(traceBody, want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
	if strings.Contains(traceBody, "v21.query.error.upstream_status") {
		t.Fatalf("trace kept generic upstream_status instead of body code: %s", traceBody)
	}
	for _, forbidden := range []string{"查一下语音唤醒误触发", adapter.URL, "127.0.0.1:18080", "/internal"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
	}
}

func TestProfessionalModeV21ContractFailureRecordsReasonMarker(t *testing.T) {
	adapter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trace_id":"a21-trace-pro-contract-invalid",
			"fast_answer":"raw answer that must not leak",
			"confidence":0.8,
			"speech_blocks":["raw speech block"]
		}`))
	}))
	defer adapter.Close()
	client, err := v21adapter.NewHTTPClient(adapter.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerWithOptions(ServerOptions{V21Client: client})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下合同证据","mode":"professional","trace_id":"a21-trace-pro-contract-invalid","session_id":"a21-session-pro-contract-invalid"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-contract-invalid", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	traceBody := traceRec.Body.String()
	if !strings.Contains(traceBody, "v21.query.error.contract_invalid") {
		t.Fatalf("trace missing contract_invalid marker: %s", traceBody)
	}
	for _, forbidden := range []string{"查一下合同证据", "raw answer", "raw speech"} {
		if strings.Contains(traceBody, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, traceBody)
		}
	}
}

func TestProfessionalModeRecordsV21QueryLatencyMetric(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{V21Client: v21adapter.NewMockClient()})
	handler := server.Handler()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", metricsRec.Code)
	}
	bodyText := metricsRec.Body.String()
	for _, want := range []string{"a21_v21_query_ms_bucket", "a21_v21_query_ms_count 1"} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("metrics missing %q:\n%s", want, bodyText)
		}
	}
}

func TestProfessionalModeV21TimeoutCancelsQueryAndFallsBack(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		V21Client:  slowV21Client{delay: 200 * time.Millisecond},
		V21Timeout: 10 * time.Millisecond,
	})
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","text":"查一下语音唤醒误触发","mode":"professional","trace_id":"a21-trace-pro-timeout","session_id":"a21-session-pro-timeout"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", body)
	rec := httptest.NewRecorder()
	start := time.Now()

	server.Handler().ServeHTTP(rec, req)

	if elapsed := time.Since(start); elapsed >= 150*time.Millisecond {
		t.Fatalf("request took %s, want timeout before slow V21 delay", elapsed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var fallback protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[len(response.Events)-1].Payload, &fallback); err != nil {
		t.Fatal(err)
	}
	if fallback.State != protocol.ExpressionError || fallback.Mode != protocol.ModeProfessional {
		t.Fatalf("fallback payload = %+v", fallback)
	}
	if !strings.Contains(fallback.Text, "V21 现在没接上") {
		t.Fatalf("fallback text = %q", fallback.Text)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-pro-timeout", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), "v21.query.timeout") {
		t.Fatalf("trace missing v21 timeout marker: %s", traceRec.Body.String())
	}
}

func TestMockInterruptReturnsInterruptedThenListening(t *testing.T) {
	server := NewServer()
	body := bytes.NewBufferString(`{"device_id":"stackchan-sim-001","trace_id":"a21-trace-000009","session_id":"a21-session-000009"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mock-interrupt", body)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response MockTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(response.Events))
	}
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(response.Events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
}

func TestMetricsEndpointRecordsMockHTTPFlow(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	mockTurn := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-sim-001"}`))
	handler.ServeHTTP(httptest.NewRecorder(), mockTurn)
	interrupt := httptest.NewRequest(http.MethodPost, "/v1/mock-interrupt", bytes.NewBufferString(`{"device_id":"stackchan-sim-001"}`))
	handler.ServeHTTP(httptest.NewRecorder(), interrupt)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"a21_mock_turn_total 1", "a21_barge_in_total 1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}

func TestMetricsEndpointRecordsAudioFrame(t *testing.T) {
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
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-sim-001",
		Kind:     protocol.KindAudioFrame,
		Seq:      1,
		Payload:  audio,
	}); err != nil {
		t.Fatal(err)
	}
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("playback kind = %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}

	resp, err := http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{"a21_audio_frame_total 1", "a21_audio_playback_chunk_total 1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, data)
		}
	}
}

func TestControlWebSocketMockTurn(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-sim-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event: protocol.DeviceEventMockTurn,
		Mode:  protocol.ModeWorkmate,
		Text:  "先说，我在",
	})

	events := readControlEvents(t, ctx, conn, 3)
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
		if events[i].TraceID != "a21-trace-000001" {
			t.Fatalf("event %d trace = %q, want a21-trace-000001", i, events[i].TraceID)
		}
	}
}

func TestControlWebSocketAcceptsArduinoClientHandshake(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, resp, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), &websocket.DialOptions{
		HTTPHeader:   http.Header{"Origin": []string{"file://"}},
		Subprotocols: []string{"arduino"},
	})
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("arduino websocket dial failed status=%d err=%v", status, err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })
	if conn.Subprotocol() != "arduino" {
		t.Fatalf("subprotocol = %q, want arduino", conn.Subprotocol())
	}

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       11,
		TraceID:   "a21-trace-arduino",
		SessionID: "a21-session-arduino",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":  "local_fallback",
			"servo_y": "45deg",
			"rgb":     "#080808",
		},
	})

	resp, err = http.Get(httpServer.URL + "/v1/devices")
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
	if registry.Devices[0].LastEvent != protocol.DeviceEventRuntimeEcho {
		t.Fatalf("last event = %q, want runtime echo", registry.Devices[0].LastEvent)
	}
}

func TestXiaozhiWebSocketHelloAcceptsStockProtocol(t *testing.T) {
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
		"device_id": "stackchan-001",
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "hello" || reply["transport"] != "websocket" {
		t.Fatalf("hello reply = %#v", reply)
	}
	if reply["device_id"] != "stackchan-001" {
		t.Fatalf("device_id = %#v, want stackchan-001", reply["device_id"])
	}
	sessionID, _ := reply["session_id"].(string)
	if !strings.HasPrefix(sessionID, "a21-session-") {
		t.Fatalf("session_id = %q, want generated A21 session", sessionID)
	}
	audio, ok := reply["audio"].(map[string]any)
	if !ok {
		t.Fatalf("reply audio = %#v", reply["audio"])
	}
	if audio["format"] != "opus" || audio["sample_rate"] != float64(24000) || audio["channels"] != float64(1) || audio["frame_duration"] != float64(60) {
		t.Fatalf("server audio = %#v", audio)
	}
	replyJSON, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"x21", "v21", "debug_metrics", "device_events"} {
		if strings.Contains(strings.ToLower(string(replyJSON)), forbidden) {
			t.Fatalf("hello reply leaked forbidden/debug field %q: %s", forbidden, replyJSON)
		}
	}
	if _, ok := reply["a21"]; ok {
		t.Fatalf("stock hello reply leaked a21 debug allowance: %#v", reply)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["device_id"] != "stackchan-001" || registry["identity_status"] != "unknown" {
		t.Fatalf("registry = %#v, want xiaozhi device with unknown firmware identity", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("registry capabilities = %#v, want sanitized xiaozhi feature map", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"xiaozhi_profile":     "stock",
		"xiaozhi_feature_mcp": "true",
		"xiaozhi_feature_aec": "true",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
	for _, forbidden := range []string{"xiaozhi_feature_debug_metrics", "xiaozhi_feature_device_events"} {
		if _, ok := capabilities[forbidden]; ok {
			t.Fatalf("stock registry leaked debug feature %q: %#v", forbidden, capabilities)
		}
	}
}

func TestXiaozhiWebSocketDebugProfileHelloReplyIncludesA21DeviceEventsAllowance(t *testing.T) {
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
		"device_id": "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
			"debug_metrics": true,
		},
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"debug_metrics", `"features"`} {
		if strings.Contains(strings.ToLower(string(replyJSON)), forbidden) {
			t.Fatalf("hello reply leaked debug field %q: %s", forbidden, replyJSON)
		}
	}
	a21, ok := reply["a21"].(map[string]any)
	if !ok {
		t.Fatalf("hello reply missing a21 debug allowance: %#v", reply)
	}
	if a21["profile"] != "debug" {
		t.Fatalf("a21 profile = %#v, want debug in %#v", a21["profile"], a21)
	}
	deviceEvents, ok := a21["device_events"].(bool)
	if !ok || !deviceEvents {
		t.Fatalf("a21 device_events = %#v, want true in %#v", a21["device_events"], a21)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("registry capabilities = %#v, want xiaozhi debug feature map", registry["capabilities"])
	}
	for key, want := range map[string]any{
		"xiaozhi_profile":                  "debug",
		"xiaozhi_feature_mcp":              "true",
		"xiaozhi_feature_aec":              "true",
		"xiaozhi_feature_device_events":    "true",
		"xiaozhi_feature_debug_metrics":    "true",
		"xiaozhi_debug_extension_isolated": "true",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestXiaozhiControlEndpointDeliversFirmwareCompatibleDeviceEvent(t *testing.T) {
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
		"device_id": "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	body := bytes.NewBufferString(`{"device_id":"stackchan-debug-001","event":"motion","name":"nod","reason":"acceptance","y_angle":120,"trace_id":"a21-trace-xiaozhi-control","session_id":"a21-session-xiaozhi-control"}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	var delivered map[string]any
	if err := wsjson.Read(ctx, conn, &delivered); err != nil {
		t.Fatal(err)
	}
	if delivered["type"] != "device" || delivered["event"] != "motion" || delivered["name"] != "nod" {
		t.Fatalf("delivered = %#v, want firmware-compatible motion command", delivered)
	}
	if delivered["kind"] != nil || delivered["motion"] != nil {
		t.Fatalf("delivered leaked legacy server-only device fields: %#v", delivered)
	}
	if delivered["y_angle"] != float64(85) {
		t.Fatalf("y_angle = %#v, want clamped 85", delivered["y_angle"])
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
		var response XiaozhiDeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Status != "delivered" || response.DeliveredTransport != "xiaozhi_ws" || response.Event != "motion" || response.Value != "nod" {
			t.Fatalf("response = %+v, want delivered motion nod", response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-control")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	traceBody, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.device_command.motion", "xiaozhi.device_command.delivered"} {
		if !strings.Contains(string(traceBody), want) {
			t.Fatalf("trace missing %q: %s", want, traceBody)
		}
	}
}

func TestXiaozhiControlEndpointRejectsStockProfile(t *testing.T) {
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
		"device_id": "stackchan-stock-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-stock-001","event":"face","emotion":"happy"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestOfficialStackChanControlEndpointDeliversOfficialMotionFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?deviceType=StackChan&device_id=stackchan-official-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-official-001","event":"motion","name":"look_up","y_angle":120,"trace_id":"a21-trace-stackchan-official-control","session_id":"a21-session-stackchan-official-control"}`)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Post(httpServer.URL+"/v1/stackchan/official/control", "application/json", body)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	msgType, frame, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if msgType != websocket.MessageBinary {
		t.Fatalf("message type = %v, want binary", msgType)
	}
	if len(frame) < 5 || frame[0] != 0x04 {
		t.Fatalf("frame header = %#v, want official ControlMotion", frame[:min(len(frame), 5)])
	}
	if got := binary.BigEndian.Uint32(frame[1:5]); got != uint32(len(frame)-5) {
		t.Fatalf("frame length = %d, want %d", got, len(frame)-5)
	}
	var payload map[string]map[string]int
	if err := json.Unmarshal(frame[5:], &payload); err != nil {
		t.Fatal(err)
	}
	if payload["pitchServo"]["angle"] != 850 {
		t.Fatalf("payload = %#v, want y_angle 120 clamped to official 850 units", payload)
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
		var response XiaozhiDeviceControlResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Status != "delivered" || response.DeliveredTransport != "stackchan_official_ws" {
			t.Fatalf("response = %+v, want official delivery", response)
		}
		if response.PacketCount != 1 ||
			response.OfficialActionPhysicalAccepted == nil ||
			*response.OfficialActionPhysicalAccepted ||
			response.OfficialActionSurfaces["servo_y"] != "pitch_clamped" ||
			response.OfficialActionSurfaces["servo_x"] != "not_used" ||
			response.OfficialActionSurfaces["rgb"] != "unchanged_no_rgb_frame" {
			t.Fatalf("response = %+v, want official action metadata without physical acceptance", response)
		}
		registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
		runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
		if !ok {
			t.Fatalf("registry = %#v, want runtime_echo", registry)
		}
		if runtimeEcho["official_stackchan_packets"] != "1" ||
			runtimeEcho["official_stackchan_physical_accepted"] != "false" ||
			runtimeEcho["official_stackchan_servo_y"] != "pitch_clamped" ||
			runtimeEcho["official_stackchan_servo_x"] != "not_used" ||
			runtimeEcho["official_stackchan_rgb"] != "unchanged_no_rgb_frame" {
			t.Fatalf("runtime_echo = %#v, want official action metadata without physical acceptance", runtimeEcho)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestOfficialStackChanControlEndpointRequiresConnectedOfficialSocket(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Post(httpServer.URL+"/v1/stackchan/official/control", "application/json", bytes.NewBufferString(`{"device_id":"stackchan-official-001","event":"face","emotion":"happy"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409: %s", resp.StatusCode, data)
	}
}

func TestXiaozhiTurnLifecycleFansOutToOfficialStackChanState(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	officialConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = officialConn.Close(websocket.StatusNormalClosure, "test done") })

	xiaozhiConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = xiaozhiConn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, xiaozhiConn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-official-state",
		"session_id": "a21-session-xiaozhi-official-state",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, xiaozhiConn)

	if err := wsjson.Write(ctx, xiaozhiConn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, xiaozhiConn)
	assertOfficialStackChanState(t, ctx, officialConn, "listening")

	if err := wsjson.Write(ctx, xiaozhiConn, map[string]any{"type": "abort", "reason": "wake_word_detected"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, xiaozhiConn)
	if stop["type"] != "tts" || stop["state"] != "stop" {
		t.Fatalf("abort stop = %#v", stop)
	}
	assertOfficialStackChanState(t, ctx, officialConn, "listening")

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-official-state")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"stackchan.official_auto.state",
		"stackchan.official_auto.delivered",
		"stackchan.display_state.received",
		"stackchan.display_state.normalized",
		"stackchan.display_state.registry_updated",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != string(protocol.DisplayStateListening) ||
		registry["display_state_source"] != "xiaozhi" ||
		registry["display_state_trace_id"] != "a21-trace-xiaozhi-official-state" ||
		registry["display_state_session_id"] != "a21-session-xiaozhi-official-state" ||
		registry["display_state_physical_accepted"] != false {
		t.Fatalf("display registry = %#v, want listening xiaozhi non-accepted state", registry)
	}
	if updated, ok := registry["display_state_updated_at_ms"].(float64); !ok || updated <= 0 {
		t.Fatalf("display updated_at = %#v, want positive ms", registry["display_state_updated_at_ms"])
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["xiaozhi_feature_mcp"] != "true" {
		t.Fatalf("display update erased xiaozhi capabilities: %#v", registry)
	}
}

func TestDeviceEventDisplayStateRegistryNormalizesAndPreservesState(t *testing.T) {
	server := NewServer()
	event := protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-display-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       1,
		TraceID:   "a21-trace-display-device-event",
		SessionID: "a21-session-display-device-event",
	}
	server.recordDeviceEvent(event, protocol.DeviceEventPayload{
		Event:        protocol.DeviceEventRuntimeEcho,
		DisplayState: protocol.NormalizeOfficialDisplayState("x21-render"),
		Capabilities: map[string]string{
			"screen": "available",
		},
		RuntimeEcho: map[string]string{
			"battery": "planned_diagnostic",
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["display_state"] != string(protocol.DisplayStateError) ||
		registry["display_state_source"] != "device_event" ||
		registry["display_state_trace_id"] != "a21-trace-display-device-event" ||
		registry["display_state_session_id"] != "a21-session-display-device-event" ||
		registry["display_state_physical_accepted"] != false {
		t.Fatalf("display registry = %#v, want normalized error device_event state", registry)
	}
	capabilities, ok := registry["capabilities"].(map[string]any)
	if !ok || capabilities["screen"] != "available" {
		t.Fatalf("capabilities erased by display state update: %#v", registry)
	}
	runtimeEcho, ok := registry["runtime_echo"].(map[string]any)
	if !ok || runtimeEcho["battery"] != "planned_diagnostic" {
		t.Fatalf("runtime_echo erased by display state update: %#v", registry)
	}
}

func TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking(t *testing.T) {
	server := NewServer()
	session := &xiaozhiSession{
		deviceID:  "stackchan-001",
		traceID:   "a21-trace-xiaozhi-official-speaking",
		sessionID: "a21-session-xiaozhi-official-speaking",
	}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	officialConn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/stackChan/ws?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = officialConn.Close(websocket.StatusNormalClosure, "test done") })

	xiaozhiRelay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(xiaozhiRelay.Close)

	xiaozhiConn, _, err := websocket.Dial(ctx, webSocketURL(xiaozhiRelay.URL, ""), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = xiaozhiConn.Close(websocket.StatusNormalClosure, "test done") })
	readXiaozhiBinary(t, ctx, xiaozhiConn)
	assertOfficialStackChanState(t, ctx, officialConn, "speaking")

	if !traceContains(server.traceEvents("a21-trace-xiaozhi-official-speaking"), "stackchan.official_auto.delivered") {
		t.Fatalf("trace missing official speaking fanout: %+v", server.traceEvents("a21-trace-xiaozhi-official-speaking"))
	}
}

func TestXiaozhiDebugProfileRecordsPlaybackStartDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-playback",
		"session_id": "a21-session-xiaozhi-playback",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "start",
		"stream_id":  "a21-xiaozhi-stream-001",
		"trace_id":   "a21-trace-xiaozhi-playback",
		"session_id": "a21-session-xiaozhi-playback",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("trace missing device playback start: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.playback.start" {
		t.Fatalf("last event = %#v, want device.playback.start", registry["last_event"])
	}
}

func TestXiaozhiDebugProfileRecordsPlaybackStopDoneDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-playback-stop-done",
		"session_id": "a21-session-xiaozhi-playback-stop-done",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "stop_done",
		"stream_id":  "a21-xiaozhi-stream-001",
		"trace_id":   "a21-trace-xiaozhi-playback-stop-done",
		"session_id": "a21-session-xiaozhi-playback-stop-done",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-playback-stop-done")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "device.playback.stop_done") {
		t.Fatalf("trace missing device playback stop_done: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["last_event"] != "device.playback.stop_done" {
		t.Fatalf("last event = %#v, want device.playback.stop_done", registry["last_event"])
	}
}

func TestXiaozhiDebugProfileRejectsUnsafePlaybackStreamID(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-unsafe-playback",
		"session_id": "a21-session-xiaozhi-unsafe-playback",
		"device_id":  "stackchan-debug-001",
		"features": map[string]any{
			"mcp":           true,
			"aec":           true,
			"device_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":       "device",
		"kind":       "playback",
		"playback":   "start",
		"stream_id":  "sk-test-secret",
		"trace_id":   "a21-trace-xiaozhi-unsafe-playback",
		"session_id": "a21-session-xiaozhi-unsafe-playback",
		"device_id":  "stackchan-debug-001",
	}); err != nil {
		t.Fatal(err)
	}
	reply := readXiaozhiJSON(t, ctx, conn)
	replyJSON := mustJSON(t, reply)
	if reply["type"] != "error" || reply["code"] != "unsupported_device_event_value" {
		t.Fatalf("reply = %#v, want unsupported_device_event_value error", reply)
	}
	for _, forbidden := range []string{"sk-test-secret", "secret", "token", "/Users", "http://"} {
		if strings.Contains(strings.ToLower(replyJSON), strings.ToLower(forbidden)) {
			t.Fatalf("unsafe playback error leaked %q: %s", forbidden, replyJSON)
		}
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-unsafe-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("unsafe stream id recorded playback start: %s", body)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if _, ok := registry["playback_stream_id"]; ok {
		t.Fatalf("unsafe stream id stored in registry: %#v", registry)
	}
}

func TestXiaozhiStockProfileRejectsPlaybackStartDeviceEvent(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-stock-playback",
		"session_id": "a21-session-xiaozhi-stock-playback",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)

	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":      "device",
		"kind":      "playback",
		"playback":  "start",
		"stream_id": "a21-xiaozhi-stream-001",
	}); err != nil {
		t.Fatal(err)
	}
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "error" || reply["code"] != "device_events_disabled" {
		t.Fatalf("reply = %#v, want device_events_disabled error", reply)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stock-playback")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceResp.Body.Close() })
	body, err := io.ReadAll(traceResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "device.playback.start") {
		t.Fatalf("stock profile recorded playback start: %s", body)
	}
}

func TestXiaozhiSessionTurnCancelInvalidatesCurrentTurnAndResetsPacer(t *testing.T) {
	session := &xiaozhiSession{}
	turn := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	if turn.id != 1 {
		t.Fatalf("turn id = %d, want 1", turn.id)
	}
	if session.shouldAbortXiaozhiTurn(turn) {
		t.Fatal("fresh current turn should not abort")
	}

	ok, err := turn.pacer.Send(context.Background(), []byte("a"), func(context.Context, []byte) error {
		return nil
	}, nil)
	if err != nil || !ok {
		t.Fatalf("pacer send = ok:%v err:%v", ok, err)
	}
	if turn.pacer.SentFrames() != 1 {
		t.Fatalf("sent frames = %d, want 1 before cancel", turn.pacer.SentFrames())
	}

	session.cancelCurrentXiaozhiTurn("abort")
	if session.currentTurn != nil {
		t.Fatalf("current turn = %+v, want cleared", session.currentTurn)
	}
	if turn.ctx.Err() == nil {
		t.Fatal("cancelled turn context is still active")
	}
	if !session.shouldAbortXiaozhiTurn(turn) {
		t.Fatal("cancelled turn should abort frame send checks")
	}
	if turn.pacer.SentFrames() != 0 {
		t.Fatalf("sent frames = %d, want pacer reset on cancel", turn.pacer.SentFrames())
	}

	next := session.startXiaozhiTurn(context.Background(), protocol.ModeWorkmate)
	if next.id != 2 {
		t.Fatalf("next turn id = %d, want 2", next.id)
	}
	if session.shouldAbortXiaozhiTurn(next) {
		t.Fatal("new current turn should not abort")
	}
}

func TestXiaozhiSessionRecentDownlinkCanBeInterruptedAfterTurnCompletion(t *testing.T) {
	session := &xiaozhiSession{
		traceID:            "a21-trace-xiaozhi-recent-downlink",
		sessionID:          "a21-session-xiaozhi-recent-downlink",
		deviceID:           "stackchan-001",
		lastDownlinkAtMS:   1000,
		lastDownlinkTurnID: "a21-xiaozhi-turn-000007",
	}

	task, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 2500, xiaozhiPlaybackInterruptWindowMS)
	if !ok {
		t.Fatal("recent downlink should allow a listen/start playback stop")
	}
	if task.turn != nil {
		t.Fatalf("recent playback task turn = %+v, want no active turn", task.turn)
	}
	if task.turnID != "a21-xiaozhi-turn-000007" || task.traceID != session.traceID || task.sessionID != session.sessionID || task.deviceID != session.deviceID {
		t.Fatalf("recent playback task = %+v", task)
	}

	if _, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 5000, xiaozhiPlaybackInterruptWindowMS); ok {
		t.Fatal("stale downlink should not be interrupted as playback")
	}
}

func TestXiaozhiSessionRecentDownlinkAfterPlaybackStopDoneIsNotBargeIn(t *testing.T) {
	session := &xiaozhiSession{
		traceID:                  "a21-trace-xiaozhi-playback-drained",
		sessionID:                "a21-session-xiaozhi-playback-drained",
		deviceID:                 "stackchan-001",
		lastDownlinkAtMS:         1000,
		lastDownlinkTurnID:       "a21-xiaozhi-turn-000007",
		lastPlaybackStopDoneAtMS: 1400,
	}

	if _, ok := session.prepareXiaozhiListenStartBargeIn("barge_in", 1500, xiaozhiPlaybackInterruptWindowMS); ok {
		t.Fatal("recent downlink with later playback stop_done should not be mislabeled as barge-in")
	}
}

func TestXiaozhiWebSocketAbortCancelsCurrentTurn(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-turn-cancel",
		"session_id": "a21-session-xiaozhi-turn-cancel",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	startAck := readXiaozhiJSON(t, ctx, conn)
	firstTurnID, ok := startAck["turn_id"].(string)
	if !ok || firstTurnID == "" {
		t.Fatalf("listen/start ack turn_id = %#v, want stable turn id", startAck["turn_id"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["turn_id"] != firstTurnID {
		t.Fatalf("abort stop turn_id = %#v, want %q", stop["turn_id"], firstTurnID)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	nextStartAck := readXiaozhiJSON(t, ctx, conn)
	nextTurnID, ok := nextStartAck["turn_id"].(string)
	if !ok || nextTurnID == "" || nextTurnID == firstTurnID {
		t.Fatalf("new listen turn_id = %#v, want new non-empty id different from %q", nextStartAck["turn_id"], firstTurnID)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
		t.Fatal(err)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-turn-cancel", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.turn.start", "xiaozhi.turn.cancel", "turn_cancelled", "downlink_queue_cleared", "barge_in_detected"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice(t *testing.T) {
	if (&xiaozhiSession{deviceID: "44:1b:f6:e2:6a:60"}).shouldSendXiaozhiListenReply() {
		t.Fatal("stock physical MAC device should not receive server listen replies")
	}
	if !(&xiaozhiSession{deviceID: "44:1b:f6:e2:6a:60", features: xiaozhitransport.HelloFeatures{DebugMetrics: true}}).shouldSendXiaozhiListenReply() {
		t.Fatal("debug profile should keep listen replies for diagnostics")
	}
	if !(&xiaozhiSession{deviceID: "stackchan-001"}).shouldSendXiaozhiListenReply() {
		t.Fatal("virtual/test device should keep listen replies")
	}
}

func TestXiaozhiSpeakerVolumeUsesStockMCPToolCall(t *testing.T) {
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
		"trace_id":   "a21-trace-volume-mcp",
		"session_id": "a21-session-volume-mcp",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/speaker-volume", "application/json", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","volume":100,"trace_id":"a21-trace-volume-mcp","session_id":"a21-session-volume-mcp"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("volume status = %d: %s", resp.StatusCode, string(body))
	}

	var response XiaozhiSpeakerVolumeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.DeliveredTransport != "xiaozhi_mcp" || response.ToolName != xiaozhiSpeakerVolumeToolName || response.Volume != 100 {
		t.Fatalf("volume response = %+v", response)
	}

	message := readXiaozhiJSON(t, ctx, conn)
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
	if params["name"] != xiaozhiSpeakerVolumeToolName {
		t.Fatalf("mcp tool name = %#v", params["name"])
	}
	args, ok := params["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("mcp arguments = %#v", params["arguments"])
	}
	if args["volume"] != float64(100) {
		t.Fatalf("mcp volume = %#v", args["volume"])
	}
}

func TestXiaozhiMCPStatusParityAllowsOnlyScopedTools(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		toolName string
		marker   string
		argKey   string
		argValue any
	}{
		{
			name:     "device status",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","trace_id":"a21-trace-mcp-device-status","session_id":"a21-session-mcp-device-status"}`,
			toolName: xiaozhiMCPGetDeviceStatusToolName,
			marker:   "xiaozhi.mcp.device_status.sent",
		},
		{
			name:     "screen brightness",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness","brightness":72,"trace_id":"a21-trace-mcp-brightness","session_id":"a21-session-mcp-brightness"}`,
			toolName: xiaozhiMCPScreenSetBrightnessToolName,
			marker:   "xiaozhi.mcp.screen_brightness.sent",
			argKey:   "brightness",
			argValue: float64(72),
		},
		{
			name:     "screen theme",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"dark","trace_id":"a21-trace-mcp-theme","session_id":"a21-session-mcp-theme"}`,
			toolName: xiaozhiMCPScreenSetThemeToolName,
			marker:   "xiaozhi.mcp.screen_theme.sent",
			argKey:   "theme",
			argValue: "dark",
		},
		{
			name:     "screen info",
			body:     `{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","trace_id":"a21-trace-mcp-info","session_id":"a21-session-mcp-info"}`,
			toolName: xiaozhiMCPScreenGetInfoToolName,
			marker:   "xiaozhi.mcp.screen_info.sent",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
				"device_id": "44:1b:f6:e2:6a:60",
			})
			readXiaozhiJSON(t, ctx, conn)

			resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/mcp-control", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("mcp control status = %d: %s", resp.StatusCode, string(body))
			}

			var response XiaozhiMCPControlResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.DeliveredTransport != "xiaozhi_mcp" || response.ToolName != tc.toolName {
				t.Fatalf("mcp control response = %+v", response)
			}
			if response.TraceID == "" || response.SessionID == "" || response.DeviceID != "44:1b:f6:e2:6a:60" {
				t.Fatalf("mcp control identity = %+v", response)
			}

			message := readXiaozhiJSON(t, ctx, conn)
			if message["type"] != "mcp" || message["trace_id"] == "" || message["session_id"] == "" || message["device_id"] != "44:1b:f6:e2:6a:60" {
				t.Fatalf("mcp wrapper = %#v", message)
			}
			payload, ok := message["payload"].(map[string]any)
			if !ok {
				t.Fatalf("mcp payload = %#v", message["payload"])
			}
			params, ok := payload["params"].(map[string]any)
			if !ok {
				t.Fatalf("mcp params = %#v", payload["params"])
			}
			if params["name"] != tc.toolName {
				t.Fatalf("mcp tool name = %#v", params["name"])
			}
			args, ok := params["arguments"].(map[string]any)
			if tc.argKey == "" {
				if ok && len(args) != 0 {
					t.Fatalf("mcp args = %#v, want none", args)
				}
			} else {
				if !ok || args[tc.argKey] != tc.argValue {
					t.Fatalf("mcp args = %#v, want %s=%#v", args, tc.argKey, tc.argValue)
				}
			}

			traceURL := httpServer.URL + "/v1/traces?trace_id=" + response.TraceID
			traceResp, err := http.Get(traceURL)
			if err != nil {
				t.Fatal(err)
			}
			defer traceResp.Body.Close()
			var traces TraceResponse
			if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
				t.Fatal(err)
			}
			if !traceContains(traces.Events, tc.marker) {
				t.Fatalf("trace missing %q: %+v", tc.marker, traces.Events)
			}
			traceBody, _ := json.Marshal(traces)
			for _, forbidden := range []string{"raw_result", "provider_output", "transcript", "data_base64", "full_url", "local_path"} {
				if strings.Contains(string(traceBody), forbidden) {
					t.Fatalf("trace leaked forbidden token %q: %s", forbidden, string(traceBody))
				}
			}
		})
	}
}

func TestXiaozhiMCPStatusParityBlocksHighRiskTools(t *testing.T) {
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
		"device_id": "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)

	blockedTools := []string{
		"self.reboot",
		"self.upgrade_firmware",
		"self.camera.take_photo",
		"self.screen.snapshot",
		"self.camera.start_stream",
		"self.nfc.read",
		"self.infrared.send",
		"self.app.launch",
	}
	for _, tool := range blockedTools {
		body := fmt.Sprintf(`{"device_id":"44:1b:f6:e2:6a:60","tool_name":%q,"trace_id":"a21-trace-blocked-mcp","session_id":"a21-session-blocked-mcp"}`, tool)
		resp, err := http.Post(httpServer.URL+"/v1/xiaozhi/mcp-control", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("%s status = %d: %s", tool, resp.StatusCode, string(body))
		}
		_ = resp.Body.Close()
	}
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)
}

func TestXiaozhiMCPStatusParityRequiresSafeArguments(t *testing.T) {
	tests := []string{
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness","brightness":101}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_brightness"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"http://example.test/theme"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.set_theme","theme":"../../theme"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.get_device_status","theme":"dark"}`,
		`{"device_id":"44:1b:f6:e2:6a:60","tool_name":"self.screen.get_info","brightness":10}`,
	}
	for _, body := range tests {
		req := httptest.NewRequest(http.MethodPost, "/v1/xiaozhi/mcp-control", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		NewServer().Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want 400", body, rec.Code)
		}
	}
}

func TestXiaozhiSayDeliversTextAsStockTTSDownlink(t *testing.T) {
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
		"trace_id":   "a21-trace-say-physical",
		"session_id": "a21-session-say-physical",
	})
	readXiaozhiJSON(t, ctx, conn)

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(
			httpServer.URL+"/v1/xiaozhi/say",
			"application/json",
			bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","text":"这是一段实体小智长文本播放测试。","trace_id":"a21-trace-say-physical","session_id":"a21-session-say-physical"}`),
		)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	start := readXiaozhiJSON(t, ctx, conn)
	if start["type"] != "tts" || start["state"] != "start" || start["phase"] != "host_say" {
		t.Fatalf("say start = %#v", start)
	}
	sentence := readXiaozhiJSON(t, ctx, conn)
	if sentence["type"] != "tts" || sentence["state"] != "sentence_start" || sentence["phase"] != "host_say" {
		t.Fatalf("say sentence = %#v", sentence)
	}
	if packet := readXiaozhiBinary(t, ctx, conn); len(packet) == 0 {
		t.Fatal("say binary packet is empty")
	}
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
		var response XiaozhiSayResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.DeliveredTransport != "xiaozhi_ws" || response.AudioChunks != 1 || response.TextChars == 0 {
			t.Fatalf("say response = %+v", response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

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

func TestXiaozhiWebSocketStockPhysicalDrainsImmediatePostAnswerListenStop(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
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
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive the product speech frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()
	assertXiaozhiProductAnswerSequence(t, ctx, conn)

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
	assertNoXiaozhiWebSocketMessage(t, conn, 150*time.Millisecond)

	traces := server.traceEvents("a21-trace-xiaozhi-product-post-answer-drain")
	for _, want := range []string{
		"xiaozhi.tts.stop.input_suppression_armed",
		"xiaozhi.listen.start.input_suppressed",
		"xiaozhi.listen.start.suppressed_post_tts_drain",
		"xiaozhi.opus_frame.ignored_suppressed_listen",
		"xiaozhi.listen.stop.suppressed_session_ended",
		"xiaozhi.listen.stop.suppressed_session_drain_armed",
	} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after suppressed post-answer drain: %+v", want, traces)
		}
	}
	if got := traceEventCount(traces, "xiaozhi.voice_pipeline.start"); got != beforeDrain {
		t.Fatalf("voice pipeline start count = %d, want unchanged %d after suppressed drain", got, beforeDrain)
	}
	if got := traceEventCount(traces, "xiaozhi.tts.opus_frame.downlink"); got != beforeDownlink {
		t.Fatalf("downlink count = %d, want unchanged %d after suppressed drain", got, beforeDownlink)
	}
	if traceContains(traces, "xiaozhi.wake_preroll.opus_frame.buffered") {
		t.Fatalf("suppressed post-answer tail audio entered wake preroll: %+v", traces)
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

func TestXiaozhiWebSocketStockPhysicalProductOrderAfterListenStop(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        streamingASR,
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
		"trace_id":   "a21-trace-xiaozhi-product-order",
		"session_id": "a21-session-xiaozhi-product-order",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive the product speech frame")
	}
	time.Sleep(150 * time.Millisecond)

	traces := server.traceEvents("a21-trace-xiaozhi-product-order")
	for _, forbidden := range []string{
		"xiaozhi.listen.stop",
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.tts.stop",
	} {
		if traceContains(traces, forbidden) {
			t.Fatalf("trace should not contain %q before product listen.stop: %+v", forbidden, traces)
		}
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()
	assertXiaozhiProductAnswerSequence(t, ctx, conn)
	assertNoXiaozhiWebSocketMessage(t, conn, 120*time.Millisecond)

	traces = server.traceEvents("a21-trace-xiaozhi-product-order")
	for _, want := range []string{
		"xiaozhi.listen.stop",
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.stt.sent",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.voice_pipeline.completed",
		"xiaozhi.tts.stop",
	} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after product answer: %+v", want, traces)
		}
	}
	stopAt, ok := traceEventAtMS(traces, "xiaozhi.listen.stop")
	if !ok {
		t.Fatalf("missing product listen.stop: %+v", traces)
	}
	for _, name := range []string{
		"asr.stream.commit",
		"asr.final",
		"xiaozhi.stt.sent",
		"xiaozhi.voice_pipeline.start",
		"tts.first_audio",
		"audio.downlink.first_frame",
		"xiaozhi.tts.stop",
	} {
		at, ok := traceEventAtMS(traces, name)
		if !ok {
			t.Fatalf("missing %s: %+v", name, traces)
		}
		if at < stopAt {
			t.Fatalf("%s at %d, want after product listen.stop at %d", name, at, stopAt)
		}
	}
}

func TestXiaozhiWebSocketListenStopDoesNotBlockAbortWhileStreamingASRCommitPending(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-nonblocking-commit",
		"session_id": "a21-session-xiaozhi-nonblocking-commit",
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
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		if traceContains(server.traceEvents("a21-trace-xiaozhi-nonblocking-commit"), "xiaozhi.abort.received") {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("abort was not processed while streaming ASR commit was pending: %+v", server.traceEvents("a21-trace-xiaozhi-nonblocking-commit"))
}

func TestXiaozhiWebSocketOpusAppendDoesNotBlockAbortControlFrame(t *testing.T) {
	streamingASR := newBlockingAppendStreamingASRAdapter()
	defer streamingASR.releaseAppend()
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
		"trace_id":   "a21-trace-xiaozhi-opus-ingress-queue",
		"session_id": "a21-session-xiaozhi-opus-ingress-queue",
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
	case <-streamingASR.appendEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR append did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if traceContains(server.traceEvents("a21-trace-xiaozhi-opus-ingress-queue"), "xiaozhi.abort.received") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("abort was not processed while Opus append was blocked: %+v", server.traceEvents("a21-trace-xiaozhi-opus-ingress-queue"))
}

func TestXiaozhiWebSocketAbortSuppressesQueuedOldTurnOpusFrames(t *testing.T) {
	streamingASR := newBlockingAppendStreamingASRAdapter()
	defer streamingASR.releaseAppend()
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
		"trace_id":   "a21-trace-xiaozhi-abort-queued-opus",
		"session_id": "a21-session-xiaozhi-abort-queued-opus",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	for i := 0; i < 3; i++ {
		if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-streamingASR.appendEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR append did not block on first Opus frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		traces := server.traceEvents("a21-trace-xiaozhi-abort-queued-opus")
		if traceEventCount(traces, "xiaozhi.opus_ingress.stale_frame_suppressed") >= 2 {
			if got := traceEventCount(traces, "audio.ingress.buffered"); got != 1 {
				t.Fatalf("audio ingress buffered = %d, want only first pre-abort frame; traces=%+v", got, traces)
			}
			if traceContains(traces, "xiaozhi.voice_pipeline.start") || traceContains(traces, "xiaozhi.tts.opus_frame.downlink") {
				t.Fatalf("stale queued frames started pipeline/downlink after abort: %+v", traces)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("queued stale Opus frames were not suppressed after abort: %+v", server.traceEvents("a21-trace-xiaozhi-abort-queued-opus"))
}

func TestXiaozhiWebSocketStreamingASRFinalStartsPipelineWithoutBatchFallback(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-streaming-final-no-batch",
		"session_id": "a21-session-xiaozhi-streaming-final-no-batch",
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
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()

	var traces []TraceEvent
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		traces = server.traceEvents("a21-trace-xiaozhi-streaming-final-no-batch")
		if traceContains(traces, "asr.final") && traceContains(traces, "xiaozhi.voice_pipeline.start") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"asr.stream.commit", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after streaming final: %+v", want, traces)
		}
	}
	select {
	case <-streamingASR.transcribe:
		t.Fatalf("batch Transcribe was called despite streaming ASR final: %+v", traces)
	default:
	}
}

func TestXiaozhiWebSocketLateStreamingASRFinalStillStartsPipeline(t *testing.T) {
	streamingASR := newLateFinalStreamingASRAdapter(250 * time.Millisecond)
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-late-final",
		"session_id": "a21-session-xiaozhi-late-final",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.appended:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}

	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" || stt["text"] != "late streaming final after commit timeout" {
		t.Fatalf("late final stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("late final tts start = %#v", ttsStart)
	}
	traces := server.traceEvents("a21-trace-xiaozhi-late-final")
	for _, want := range []string{"asr.stream.commit", "asr.stream.final_timeout", "asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q: %+v", want, traces)
		}
	}
	if got := traceEventCount(traces, "xiaozhi.voice_pipeline.start"); got != 1 {
		t.Fatalf("voice pipeline start count = %d, want exactly one late-final answer: %+v", got, traces)
	}
}

func TestXiaozhiWebSocketStreamingASRFinalStartsPipelineWhenGatewayVADMisses(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-vad-miss-final",
		"session_id": "a21-session-xiaozhi-vad-miss-final",
		"device_id":  "44:1b:f6:e2:6a:60",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" {
		t.Fatalf("stt = %#v", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start = %#v", ttsStart)
	}
	traces := server.traceEvents("a21-trace-xiaozhi-vad-miss-final")
	for _, want := range []string{"asr.final", "xiaozhi.voice_pipeline.start"} {
		if !traceContains(traces, want) {
			t.Fatalf("trace missing %q after streaming final despite VAD miss: %+v", want, traces)
		}
	}
	if traceContains(traces, "vad.speech.start") {
		t.Fatalf("test must prove VAD-miss path, got vad.speech.start: %+v", traces)
	}
}

func TestXiaozhiWebSocketStreamingASRFinalSendsStockSTTBeforeTTS(t *testing.T) {
	streamingASR := newBlockingCommitStreamingASRAdapter()
	defer streamingASR.releaseCommit()
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-streaming-final-stt",
		"session_id": "a21-session-xiaozhi-streaming-final-stt",
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
	case <-time.After(time.Second):
		t.Fatal("streaming ASR did not receive an audio frame")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop", "mode": "realtime"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-streamingASR.commitEntered:
	case <-time.After(time.Second):
		t.Fatal("streaming ASR commit did not start")
	}
	streamingASR.releaseCommit()

	stt := readXiaozhiJSON(t, ctx, conn)
	if stt["type"] != "stt" || stt["text"] != "streaming final after async commit" {
		t.Fatalf("first post-ASR message = %#v, want stock stt text before tts", stt)
	}
	ttsStart := readXiaozhiJSON(t, ctx, conn)
	if ttsStart["type"] != "tts" || ttsStart["state"] != "start" {
		t.Fatalf("tts start after stt = %#v", ttsStart)
	}
	tracePayload := mustJSON(t, server.traceEvents("a21-trace-xiaozhi-streaming-final-stt"))
	if strings.Contains(tracePayload, "streaming final after async commit") {
		t.Fatalf("trace leaked ASR transcript text: %s", tracePayload)
	}
}

func TestXiaozhiVoicePipelineRecordsProviderFallbackObservability(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: gatewayFallbackTextStreamAdapter{},
			TTS:        segmentChunkTTSAdapter{},
			Selection: providers.VoicePipelineSelection{
				ASRMode:               "local",
				ASRProfile:            "a21-gateway-final-asr",
				ASRProfileEnv:         "A21_ASR_LOCAL_PROFILE",
				LLMProfile:            "deepseek",
				LLMProfileEnv:         "A21_PROVIDER_PRIMARY",
				LLMFallbackProfile:    "a21_voice_fallback",
				LLMFallbackProfileEnv: "A21_TEXT_STREAM_FALLBACK_PROFILE",
				TTSMode:               "fast",
				TTSProfile:            "a21-segment-chunk-tts",
				TTSProfileEnv:         "A21_TTS_FAST_PROFILE",
			},
			ExecutionMode: "fixture",
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
		"trace_id":   "a21-trace-xiaozhi-fallback",
		"session_id": "a21-session-xiaozhi-fallback",
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
		t.Fatalf("fast ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	fallback, ok := answerPipeline["fallback"].(map[string]any)
	if !ok || fallback["activated"] != true || fallback["provider"] != "a21_voice_fallback" || fallback["reason"] != "primary_failed" {
		t.Fatalf("fallback summary = %#v", answerPipeline["fallback"])
	}
	answerPayload := mustJSON(t, answerSentence)
	for _, forbidden := range []string{"fallback voice answer", "fallback reasoning", "http://", "https://", "/Users/", "sk-"} {
		if strings.Contains(answerPayload, forbidden) {
			t.Fatalf("xiaozhi fallback summary leaked %q: %s", forbidden, answerPayload)
		}
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	readXiaozhiJSON(t, ctx, conn)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fallback", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"fallback.used", "provider.failover", "xiaozhi.voice_pipeline.completed"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	for _, want := range []string{"a21_provider_failover_total 1", "a21_fallback_total 1"} {
		if !strings.Contains(metricsRec.Body.String(), want) {
			t.Fatalf("metrics missing %q:\n%s", want, metricsRec.Body.String())
		}
	}
}

func TestXiaozhiVoicePipelineUnavailableEmitsLocalFallbackState(t *testing.T) {
	server := NewServer()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return unavailableXiaozhiPipelineRunner{}
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
		"trace_id":   "a21-trace-xiaozhi-local-fallback",
		"session_id": "a21-session-xiaozhi-local-fallback",
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
	readXiaozhiBinary(t, ctx, conn)
	fallback := readXiaozhiJSON(t, ctx, conn)
	if fallback["type"] != "tts" ||
		fallback["state"] != "sentence_start" ||
		fallback["phase"] != "local_fallback" ||
		fallback["mode"] != "local_fallback" ||
		!strings.Contains(asString(fallback["text"]), "外部大脑连不上") ||
		!strings.Contains(asString(fallback["text"]), "我还在") {
		t.Fatalf("fallback sentence = %#v", fallback)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "local_fallback" {
		t.Fatalf("fallback stop = %#v", stop)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-local-fallback", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	for _, want := range []string{
		"xiaozhi.voice_pipeline.llm.real_streaming",
		"xiaozhi.voice_pipeline.failed.text_stream_adapter_failed",
		"xiaozhi.voice_pipeline.failed.no_llm_first_content",
		"xiaozhi.voice_pipeline.failed.no_tts_first_audio",
		"xiaozhi.voice_pipeline.failed.empty_audio",
		"xiaozhi.voice_pipeline.unavailable",
		"local_fallback.entered",
		"xiaozhi.local_fallback.sent",
	} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, metricsReq)
	if !strings.Contains(metricsRec.Body.String(), "a21_fallback_total 1") {
		t.Fatalf("metrics missing local fallback count:\n%s", metricsRec.Body.String())
	}
	if strings.Contains(metricsRec.Body.String(), "a21_provider_failover_total 1") {
		t.Fatalf("local fallback incremented provider failover count:\n%s", metricsRec.Body.String())
	}

	devicesReq := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	devicesRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(devicesRec, devicesReq)
	for _, want := range []string{`"current_mode":"local_fallback"`, `"current_expression":"local_fallback"`} {
		if !strings.Contains(devicesRec.Body.String(), want) {
			t.Fatalf("devices missing %q: %s", want, devicesRec.Body.String())
		}
	}
}

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
	nowMS := int64(1000)
	server.now = func() time.Time {
		nowMS += 60
		return time.UnixMilli(nowMS)
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
	if got := server.xiaozhiListenMode("realtime", xiaozhitransport.HelloFeatures{MCP: true, AEC: true}); got != protocol.ModeProfessional {
		t.Fatalf("stock mode = %q, want professional", got)
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

func TestXiaozhiWebSocketProfessionalModeSendsStructuredRedactedResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(0)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-structured",
		"session_id": "a21-session-xiaozhi-pro-structured",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	result := readXiaozhiJSON(t, ctx, conn)
	professional, ok := result["professional"].(map[string]any)
	if !ok {
		t.Fatalf("professional report = %#v", result["professional"])
	}
	for key, want := range map[string]float64{
		"evidence_count":     1,
		"screen_card_count":  1,
		"follow_up_count":    1,
		"speech_block_count": 1,
	} {
		if professional[key] != want {
			t.Fatalf("professional[%s] = %#v, want %.0f", key, professional[key], want)
		}
	}
	if result["text"] != "结论：需要按可引用证据复核。" || result["confidence"] != 0.77 {
		t.Fatalf("professional conclusion/confidence = %#v", result)
	}
	encoded := mustJSON(t, result)
	for _, forbidden := range []string{"RAW_SECRET_EVIDENCE_BODY", "RAW_SECRET_QUOTE", "RAW_SECRET_CARD_TEXT", "v21-doc-secret-raw"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("professional result leaked %q: %s", forbidden, encoded)
		}
	}
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketProfessionalAbortSuppressesStaleResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(-1)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-abort",
		"session_id": "a21-session-xiaozhi-pro-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}

	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	v21.release()
	select {
	case <-v21.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("professional V21 query did not observe abort cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketProfessionalNewTurnSuppressesStaleResult(t *testing.T) {
	v21 := newDelayedXiaozhiProfessionalV21Client(-1)
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
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
		"trace_id":   "a21-trace-xiaozhi-pro-new-turn",
		"session_id": "a21-session-xiaozhi-pro-new-turn",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start", "mode": "professional"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	select {
	case <-v21.started:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("professional V21 query did not start")
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "barge_in" {
		t.Fatalf("new turn stop = %#v", stop)
	}
	newTurnAck := readXiaozhiJSON(t, ctx, conn)
	if newTurnAck["type"] != "listen" || newTurnAck["state"] != "start" || newTurnAck["status"] != "accepted" {
		t.Fatalf("new turn ack = %#v", newTurnAck)
	}
	v21.release()
	select {
	case <-v21.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("professional V21 query did not observe new-turn cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketSendsFastAckBeforeVoicePipelineCompletes(t *testing.T) {
	server := NewServer()
	runner := newSlowAnswerXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.releaseAnswer)

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
		"trace_id":   "a21-trace-xiaozhi-fast-ack",
		"session_id": "a21-session-xiaozhi-fast-ack",
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
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("voice pipeline did not start")
	}

	ackCtx, ackCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer ackCancel()
	ttsStart := readXiaozhiJSON(t, ackCtx, conn)
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
	if pipeline["status"] != "running" || pipeline["stage"] != "fast_ack" {
		t.Fatalf("voice pipeline = %#v, want fast_ack running", pipeline)
	}
	for _, forbidden := range []string{"data_base64", "raw_audio", "provider output", "fast ack", "http://", "https://", "/Users/"} {
		if strings.Contains(mustJSON(t, ttsStart), forbidden) {
			t.Fatalf("tts start leaked %q: %s", forbidden, mustJSON(t, ttsStart))
		}
	}

	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-runner.released:
		t.Fatal("answer pipeline completed before fast ack was read")
	default:
	}

	runner.releaseAnswer()
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	if answerPipeline["stage"] != "answer" || answerPipeline["audio_chunk_count"] != float64(1) {
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
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestXiaozhiWebSocketStreamsFirstAnswerSegmentBeforeTextStreamDone(t *testing.T) {
	textStream := newBlockingSegmentTextStreamAdapter()
	tts := segmentChunkTTSAdapter{}
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: textStream,
			TTS:        tts,
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	t.Cleanup(textStream.releaseFinal)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-stream-answer",
		"session_id": "a21-session-xiaozhi-stream-answer",
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
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-textStream.firstSegmentSent:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("text stream did not emit first segment")
	}

	answerCtx, answerCancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer answerCancel()
	answerSentence := readXiaozhiJSON(t, answerCtx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	answerPipeline, ok := answerSentence["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("answer voice pipeline = %#v", answerSentence["voice_pipeline"])
	}
	if answerPipeline["stage"] != "answer" || answerPipeline["streaming"] != true {
		t.Fatalf("answer voice pipeline = %#v, want streaming answer", answerPipeline)
	}
	messageType, data, err = conn.Read(answerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-textStream.finalReleased:
		t.Fatal("final text segment was released before first answer binary")
	default:
	}

	textStream.releaseFinal()
	_ = readXiaozhiJSON(t, ctx, conn)
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("final answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_answer_completed" {
		t.Fatalf("tts stop = %#v", ttsStop)
	}
}

func TestXiaozhiWebSocketStreamingAnswerErrorAfterAudioDoesNotFallbackLoop(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: singleSentenceTextStreamAdapter{},
			TTS:        chunkThenErrorTTSAdapter{},
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
		"trace_id":   "a21-trace-xiaozhi-stream-degraded",
		"session_id": "a21-session-xiaozhi-stream-degraded",
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
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	ttsStop := readXiaozhiJSON(t, ctx, conn)
	if ttsStop["type"] != "tts" || ttsStop["state"] != "stop" || ttsStop["reason"] != "voice_pipeline_answer_completed_degraded" {
		t.Fatalf("tts stop after degraded streaming audio = %#v", ttsStop)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-stream-degraded", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "xiaozhi.voice_pipeline.completed_degraded_after_audio") {
		t.Fatalf("trace missing degraded-after-audio marker: %+v", traces.Events)
	}
	if traceContains(traces.Events, "xiaozhi.voice_pipeline.unavailable") || traceContains(traces.Events, "xiaozhi.local_fallback.sent") {
		t.Fatalf("streaming audio already played; must not local fallback loop: %+v", traces.Events)
	}
	if !traceContains(traces.Events, "xiaozhi.tts.stop.input_suppression_armed") {
		t.Fatalf("trace missing post-tts input suppression: %+v", traces.Events)
	}
}

func TestXiaozhiWebSocketAbortDuringStreamingAnswerSuppressesStaleSegments(t *testing.T) {
	textStream := newBlockingSegmentTextStreamAdapter()
	server := NewServerWithOptions(ServerOptions{
		XiaozhiVoicePipelineAdapters: &providers.VoicePipelineAdapters{
			ASR:        gatewayFinalASRAdapter{},
			TextStream: textStream,
			TTS:        segmentChunkTTSAdapter{},
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		},
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	t.Cleanup(textStream.releaseFinal)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"trace_id":   "a21-trace-xiaozhi-stream-abort",
		"session_id": "a21-session-xiaozhi-stream-abort",
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
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("answer downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	select {
	case <-textStream.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("streaming text adapter did not observe cancellation")
	}
	textStream.releaseFinal()
	assertNoXiaozhiWebSocketMessage(t, conn, 150*time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-xiaozhi-stream-abort")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var traces TraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"xiaozhi.tts.opus_frame.downlink",
		"xiaozhi.turn.cancel",
		"downlink_queue_cleared",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestXiaozhiWebSocketAbortAfterFastAckSuppressesAnswerFrames(t *testing.T) {
	server := NewServer()
	runner := newSlowAnswerXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.releaseAnswer)

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
		"trace_id":   "a21-trace-xiaozhi-fast-ack-abort",
		"session_id": "a21-session-xiaozhi-fast-ack-abort",
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
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("voice pipeline did not start")
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("slow answer pipeline did not observe cancellation")
	}
	runner.releaseAnswer()
	assertNoXiaozhiMessage(t, conn, 150*time.Millisecond)
}

func TestXiaozhiWebSocketContinuesAnswerWhenFastAckUnavailable(t *testing.T) {
	server := NewServer()
	server.xiaozhiFastAckTTS = failingXiaozhiTTSAdapter{}
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return fallbackReportingXiaozhiPipelineRunner{}
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
		"trace_id":   "a21-trace-xiaozhi-fast-ack-unavailable",
		"session_id": "a21-session-xiaozhi-fast-ack-unavailable",
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
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
	messageType, data, err := conn.Read(ctx)
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
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-fast-ack-unavailable", nil)
	traceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d: %s", traceRec.Code, traceRec.Body.String())
	}
	var traces TraceResponse
	if err := json.NewDecoder(traceRec.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"xiaozhi.fast_ack.unavailable", "provider.first_content", "tts.first_audio", "audio.downlink.first_frame", "xiaozhi.voice_pipeline.completed"} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
}

func TestNewServerWithOptionsUsesConfiguredXiaozhiVoicePipelineAdapters(t *testing.T) {
	adapters := providers.VoicePipelineAdapters{
		ASR:        providers.NewMockASRAdapter("a21-test-asr"),
		TextStream: providers.NewMockTextStreamAdapter("a21-test-text"),
		TTS:        providers.NewMockTTSAdapter("a21-test-tts"),
		Selection: providers.VoicePipelineSelection{
			ASRMode:    "local",
			ASRProfile: "a21-test-asr",
			LLMProfile: "a21-test-text",
			TTSMode:    "fast",
			TTSProfile: "a21-test-tts",
		},
		ExecutionMode: "host_local",
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
		"trace_id":   "a21-trace-xiaozhi-configured-pipeline",
		"session_id": "a21-session-xiaozhi-configured-pipeline",
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
	pipeline, ok := ttsStart["voice_pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("voice pipeline = %#v", ttsStart["voice_pipeline"])
	}
	for _, want := range []string{
		`"execution_mode":"host_local"`,
		`"schema_version":"a21.voice_pipeline.fast_ack.v1"`,
		`"stage":"fast_ack"`,
		`"status":"running"`,
		`"asr_profile":"a21-test-asr"`,
		`"llm_profile":"a21-test-text"`,
		`"tts_profile":"a21-test-tts"`,
	} {
		if !strings.Contains(mustJSON(t, pipeline), want) {
			t.Fatalf("voice pipeline missing %q: %#v", want, pipeline)
		}
	}
	readXiaozhiJSON(t, ctx, conn)
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("fast ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	answerSentence := readXiaozhiJSON(t, ctx, conn)
	if answerSentence["type"] != "tts" || answerSentence["state"] != "sentence_start" || answerSentence["phase"] != "answer" {
		t.Fatalf("answer sentence = %#v", answerSentence)
	}
}

func TestXiaozhiWebSocketAbortCancelsBlockedTurnTaskWithinBargeInBudget(t *testing.T) {
	server := NewServer()
	runner := newBlockingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.unblock)

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
		"trace_id":   "a21-trace-xiaozhi-blocked-abort",
		"session_id": "a21-session-xiaozhi-blocked-abort",
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
	ackSentence := readXiaozhiJSON(t, ctx, conn)
	if ackSentence["type"] != "tts" || ackSentence["state"] != "sentence_start" || ackSentence["phase"] != "fast_ack" {
		t.Fatalf("ack sentence = %#v", ackSentence)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("xiaozhi turn task did not enter blocking pipeline")
	}

	abortAt := time.Now()
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "barge_in"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	stopAfter := time.Since(abortAt)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	if stopAfter >= 300*time.Millisecond {
		t.Fatalf("abort stop latency = %s, want <300ms", stopAfter)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("blocked xiaozhi pipeline did not observe turn cancellation")
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-blocked-abort", nil)
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
		"barge_in.detected",
		"provider.cancel.end",
		"playback.stop",
		"xiaozhi.turn.cancel",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traces.Summary.BargeInStopMS == nil || *traces.Summary.BargeInStopMS >= 300 {
		t.Fatalf("barge-in summary = %v, want <300ms", traces.Summary.BargeInStopMS)
	}
	t.Logf("xiaozhi abort stop latency=%s trace_barge_in_stop_ms=%d", stopAfter, *traces.Summary.BargeInStopMS)
}

func TestXiaozhiWebSocketListenStartBargeInStopsActiveTTS(t *testing.T) {
	server := NewServer()
	runner := newBlockingXiaozhiPipelineRunner()
	server.xiaozhiVoicePipelineRunner = func() xiaozhiVoicePipelineRunner {
		return runner
	}
	t.Cleanup(runner.unblock)

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
		"trace_id":   "a21-trace-xiaozhi-listen-barge",
		"session_id": "a21-session-xiaozhi-listen-barge",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	firstStart := readXiaozhiJSON(t, ctx, conn)
	firstTurnID, ok := firstStart["turn_id"].(string)
	if !ok || firstTurnID == "" {
		t.Fatalf("first listen ack turn_id = %#v", firstStart["turn_id"])
	}
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiTestSpeechOpusPacket(t)); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
	if messageType, data, err := conn.Read(ctx); err != nil {
		t.Fatal(err)
	} else if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("ack downlink message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	select {
	case <-runner.entered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("xiaozhi turn task did not enter blocking pipeline")
	}

	bargeAt := time.Now()
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	stopAfter := time.Since(bargeAt)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "barge_in" {
		t.Fatalf("listen/start barge stop = %#v", stop)
	}
	if stop["turn_id"] != firstTurnID {
		t.Fatalf("listen/start barge stop turn_id = %#v, want %q", stop["turn_id"], firstTurnID)
	}
	if stopAfter >= 300*time.Millisecond {
		t.Fatalf("listen/start barge stop latency = %s, want <300ms", stopAfter)
	}
	nextStart := readXiaozhiJSON(t, ctx, conn)
	if nextStart["type"] != "listen" || nextStart["state"] != "start" || nextStart["status"] != "accepted" {
		t.Fatalf("next listen ack = %#v", nextStart)
	}
	nextTurnID, ok := nextStart["turn_id"].(string)
	if !ok || nextTurnID == "" || nextTurnID == firstTurnID {
		t.Fatalf("next turn_id = %#v, first=%q", nextStart["turn_id"], firstTurnID)
	}
	select {
	case <-runner.canceled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("blocked xiaozhi pipeline did not observe listen/start barge-in cancellation")
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-xiaozhi-listen-barge", nil)
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
		"xiaozhi.listen.barge_in",
		"barge_in.detected",
		"provider.cancel.end",
		"playback.stop",
		"xiaozhi.turn.cancel",
		"downlink_queue_cleared",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traceContains(traces.Events, "xiaozhi.abort.received") {
		t.Fatalf("listen/start barge-in must not be mislabeled as abort: %+v", traces.Events)
	}
	if traces.Summary.BargeInStopMS == nil || *traces.Summary.BargeInStopMS >= 300 {
		t.Fatalf("barge-in summary = %v, want <300ms", traces.Summary.BargeInStopMS)
	}
}

func TestXiaozhiWebSocketAcceptsProtocolVersion3BinaryFrames(t *testing.T) {
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
		"version":    3,
		"trace_id":   "a21-trace-xiaozhi-v3",
		"session_id": "a21-session-xiaozhi-v3",
		"device_id":  "stackchan-001",
	})
	hello := readXiaozhiJSON(t, ctx, conn)
	if hello["version"] != float64(3) {
		t.Fatalf("hello version = %#v, want 3", hello["version"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	payload := xiaozhiTestOpusPacket(t)
	wire := make([]byte, 4+len(payload))
	wire[0] = 0
	binary.BigEndian.PutUint16(wire[2:4], uint16(len(payload)))
	copy(wire[4:], payload)
	if err := conn.Write(ctx, websocket.MessageBinary, wire); err != nil {
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
	if summary["profile"] != "xiaozhi_binary_v3" || summary["frame_count"] != float64(1) || summary["byte_count"] != float64(len(payload)) || summary["decode_status"] != XiaozhiOpusDecodedPCMState {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketAcceptsProtocolVersion2BinaryFrames(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/v1/xiaozhi"), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Device-Id":        []string{"stackchan-v2-001"},
			"Protocol-Version": []string{"2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeXiaozhiHello(t, ctx, conn, map[string]any{
		"version":    2,
		"trace_id":   "a21-trace-xiaozhi-v2",
		"session_id": "a21-session-xiaozhi-v2",
	})
	hello := readXiaozhiJSON(t, ctx, conn)
	if hello["version"] != float64(2) || hello["device_id"] != "stackchan-v2-001" {
		t.Fatalf("hello version/device = %#v/%#v, want v2 header device", hello["version"], hello["device_id"])
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)

	payload := xiaozhiTestOpusPacket(t)
	if err := conn.Write(ctx, websocket.MessageBinary, xiaozhiProtocol2Wire(payload, 240)); err != nil {
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
	if summary["profile"] != "xiaozhi_binary_v2" || summary["frame_count"] != float64(1) || summary["byte_count"] != float64(len(payload)) || summary["decode_status"] != XiaozhiOpusDecodedPCMState {
		t.Fatalf("audio summary = %#v", summary)
	}
	readXiaozhiJSON(t, ctx, conn)
	readXiaozhiJSON(t, ctx, conn)
}

func TestXiaozhiWebSocketAbortStopsPlaceholderTTSAndPreventsStaleBinary(t *testing.T) {
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
		"trace_id":   "a21-trace-xiaozhi-abort",
		"session_id": "a21-session-xiaozhi-abort",
		"device_id":  "stackchan-001",
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "listen", "state": "start"}); err != nil {
		t.Fatal(err)
	}
	readXiaozhiJSON(t, ctx, conn)
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "abort", "reason": "wake_word_detected"}); err != nil {
		t.Fatal(err)
	}
	stop := readXiaozhiJSON(t, ctx, conn)
	if stop["type"] != "tts" || stop["state"] != "stop" || stop["reason"] != "abort" {
		t.Fatalf("abort stop = %#v", stop)
	}
	if _, ok := stop["turn_id"].(string); !ok {
		t.Fatalf("abort stop turn_id = %#v, want cancelled turn id", stop["turn_id"])
	}
	if err := conn.Write(ctx, websocket.MessageBinary, []byte{0x04, 0x05}); err != nil {
		t.Fatal(err)
	}
	assertNoXiaozhiMessage(t, conn, 100*time.Millisecond)
}

func TestXiaozhiWebSocketRejectsLegacyIdentity(t *testing.T) {
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
		"device_id": "x21-device",
	})
	reply := readXiaozhiJSON(t, ctx, conn)
	if reply["type"] != "error" || reply["code"] != "invalid_device_id" {
		t.Fatalf("reply = %#v, want invalid_device_id", reply)
	}
}

func TestControlWebSocketRegistersFirmwareIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       7,
		TraceID:   "a21-trace-device-000007",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "先说，我在",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var registry DeviceRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(registry.Devices))
	}
	device := registry.Devices[0]
	if device.DeviceID != "stackchan-001" {
		t.Fatalf("device id = %q", device.DeviceID)
	}
	if device.IdentityStatus != "ok" {
		t.Fatalf("identity status = %q, want ok", device.IdentityStatus)
	}
	if device.Firmware.ID != "a21-stackchan" || device.Firmware.Version != "0.1.0" || device.Firmware.Board != "m5stack-cores3" || device.Firmware.Commit != "082eb938b713" {
		t.Fatalf("firmware identity = %+v", device.Firmware)
	}
	if device.LastEvent != "mock.turn" {
		t.Fatalf("last event = %q, want mock.turn", device.LastEvent)
	}
	if device.LastTraceID != "a21-trace-device-000007" {
		t.Fatalf("last trace = %q", device.LastTraceID)
	}
}

func TestControlWebSocketRegistersStackChanCapabilities(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       8,
		TraceID:   "a21-trace-device-000008",
		SessionID: "a21-session-device",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "能力上报",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		Capabilities: map[string]string{
			"microphone":    "available",
			"speaker":       "available",
			"screen":        "available",
			"screen_touch":  "available",
			"top_touch":     "available",
			"servo_y":       "available",
			"servo_x":       "planned_continuous_rotation_axis",
			"rgb":           "available",
			"camera":        "planned_core_s3_camera",
			"imu":           "planned_9_axis_imu",
			"ambient_light": "planned_ambient_light_sensor",
			"proximity":     "planned_proximity_sensor",
			"battery":       "planned_550mah_battery",
			"nfc":           "planned_nfc",
			"infrared":      "planned_infrared_tx_rx",
		},
	})
	readControlEvents(t, ctx, conn, 3)

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
	capabilities := registry.Devices[0].Capabilities
	wantCapabilities := map[string]string{
		"microphone":    "available",
		"speaker":       "available",
		"screen":        "available",
		"screen_touch":  "available",
		"top_touch":     "available",
		"servo_y":       "available",
		"servo_x":       "planned_continuous_rotation_axis",
		"rgb":           "available",
		"camera":        "planned_core_s3_camera",
		"imu":           "planned_9_axis_imu",
		"ambient_light": "planned_ambient_light_sensor",
		"proximity":     "planned_proximity_sensor",
		"battery":       "planned_550mah_battery",
		"nfc":           "planned_nfc",
		"infrared":      "planned_infrared_tx_rx",
	}
	for key, want := range wantCapabilities {
		if capabilities[key] != want {
			t.Fatalf("capability %s = %q, want %q; all=%#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestControlWebSocketRegistersRuntimeEchoWithoutAssistantReply(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       11,
		TraceID:   "a21-trace-runtime-echo",
		SessionID: "a21-session-runtime-echo",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
		RuntimeEcho: map[string]string{
			"screen":  "speaking",
			"servo_y": "48deg",
			"rgb":     "#002430",
		},
	})

	ctxShort, cancelShort := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelShort()
	var reply protocol.Envelope
	if err := wsjson.Read(ctxShort, conn, &reply); err == nil {
		t.Fatalf("runtime echo produced unexpected assistant reply: %+v", reply)
	}

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
	if device.LastEvent != protocol.DeviceEventRuntimeEcho {
		t.Fatalf("last event = %q, want runtime echo", device.LastEvent)
	}
	for key, want := range map[string]string{
		"screen":  "speaking",
		"servo_y": "48deg",
		"rgb":     "#002430",
	} {
		if device.RuntimeEcho[key] != want {
			t.Fatalf("runtime echo %s = %q, want %q; all=%#v", key, device.RuntimeEcho[key], want, device.RuntimeEcho)
		}
	}
}

func TestControlWebSocketRejectsLegacyRuntimeEchoIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      12,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventRuntimeEcho,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
		RuntimeEcho: map[string]string{
			"screen": "x21-render",
		},
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}
	if strings.Contains(payload.Text, "x21-render") {
		t.Fatalf("error leaked forbidden echo value: %q", payload.Text)
	}
}

func TestControlWebSocketRejectsLegacyCapabilityIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
		Capabilities: map[string]string{
			"screen": "x21-compatible",
		},
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}
}

func TestDevicesEndpointDeclaresA21GatewayIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["schema_version"] != "a21.gateway.devices.v1" {
		t.Fatalf("schema_version = %#v, want a21.gateway.devices.v1", payload["schema_version"])
	}
	if payload["service"] != "a21-gateway" {
		t.Fatalf("service = %#v, want a21-gateway", payload["service"])
	}
}

func TestControlWebSocketRegistryExposesCurrentModeAndExpressionWithoutText(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       9,
		TraceID:   "a21-trace-device-state",
		SessionID: "a21-session-device-state",
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModePrivate,
		Text:            "这句私人吐槽不应该进入设备 registry",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	resp, err := http.Get(httpServer.URL + "/v1/devices")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.Unmarshal(body, &registry); err != nil {
		t.Fatal(err)
	}
	devices := registry.Devices
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1: %s", len(devices), string(body))
	}
	device := devices[0]
	if device["current_mode"] != string(protocol.ModePrivate) {
		t.Fatalf("current_mode = %#v, want private; body=%s", device["current_mode"], string(body))
	}
	if device["current_expression"] != string(protocol.ExpressionSpeaking) {
		t.Fatalf("current_expression = %#v, want speaking; body=%s", device["current_expression"], string(body))
	}
	if strings.Contains(string(body), "私人吐槽") {
		t.Fatalf("registry leaked utterance text: %s", string(body))
	}
}

func TestGatewayDeviceRegistryExposesExplicitModeStatesWithoutProfessionalBodies(t *testing.T) {
	tests := []struct {
		mode protocol.Mode
		expr protocol.ExpressionState
	}{
		{mode: protocol.ModePublic, expr: protocol.ExpressionListening},
		{mode: protocol.ModePrivate, expr: protocol.ExpressionListening},
		{mode: protocol.ModeFocus, expr: protocol.ExpressionIdle},
		{mode: protocol.ModeProfessional, expr: protocol.ExpressionProfessional},
		{mode: protocol.ModeMuted, expr: protocol.ExpressionIdle},
		{mode: protocol.ModeWorkmate, expr: protocol.ExpressionListening},
	}
	for _, tc := range tests {
		t.Run(string(tc.mode), func(t *testing.T) {
			server := NewServer()
			traceID := "a21-trace-registry-" + string(tc.mode)
			sessionID := "a21-session-registry-" + string(tc.mode)
			server.controlSequence("stackchan-sim-001", traceID, sessionID, []protocol.ControlEventPayload{{
				State:        tc.expr,
				Mode:         tc.mode,
				Text:         "professional answer body must not persist",
				Evidence:     []protocol.EvidenceItem{{Title: "private evidence title", Summary: "private evidence summary"}},
				ScreenCards:  []protocol.ScreenCard{{Label: "card", Text: "private card body"}},
				SpeechBlocks: []string{"private speech block"},
			}})

			req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			var registry DeviceRegistryResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &registry); err != nil {
				t.Fatal(err)
			}
			if len(registry.Devices) != 1 {
				t.Fatalf("devices = %d, want 1: %s", len(registry.Devices), rec.Body.String())
			}
			device := registry.Devices[0]
			if device.CurrentMode != tc.mode || device.CurrentExpr != tc.expr {
				t.Fatalf("device state = %+v, want mode=%s expr=%s", device, tc.mode, tc.expr)
			}
			for _, forbidden := range []string{"professional answer body", "private evidence title", "private evidence summary", "private card body", "private speech block"} {
				if strings.Contains(rec.Body.String(), forbidden) {
					t.Fatalf("registry leaked %q: %s", forbidden, rec.Body.String())
				}
			}
		})
	}
}

func TestDeviceRegistryMarksOnlineAndStaleByLastSeenAge(t *testing.T) {
	server := NewServer()
	current := time.UnixMilli(2_000_000)
	server.now = func() time.Time { return current }
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      10,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		Text:            "heartbeat",
		FirmwareID:      "a21-stackchan",
		FirmwareVersion: "0.1.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "082eb938b713",
	})
	readControlEvents(t, ctx, conn, 3)

	current = time.UnixMilli(2_000_000 + 299_999)
	online := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if online["connection_status"] != "online" || int64(online["device_age_ms"].(float64)) != 299_997 {
		t.Fatalf("online status/age = %#v/%#v, want online/299997", online["connection_status"], online["device_age_ms"])
	}

	current = time.UnixMilli(2_000_000 + 300_003)
	stale := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if stale["connection_status"] != "stale" || int64(stale["device_age_ms"].(float64)) != 300_001 {
		t.Fatalf("stale status/age = %#v/%#v, want stale/300001", stale["connection_status"], stale["device_age_ms"])
	}
}

func TestControlWebSocketRejectsForbiddenFirmwareIdentity(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol: protocol.ProtocolVersion,
		DeviceID: "stackchan-legacy-001",
		Kind:     protocol.KindDeviceEvent,
		Seq:      1,
	}, protocol.DeviceEventPayload{
		Event:           protocol.DeviceEventMockTurn,
		Mode:            protocol.ModeWorkmate,
		FirmwareID:      "x21-stackchan",
		FirmwareVersion: "1.0.0",
		FirmwareBoard:   "m5stack-cores3",
		FirmwareCommit:  "abcdef1",
	})

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionError {
		t.Fatalf("state = %q, want error", payload.State)
	}
	if !strings.Contains(payload.Text, "invalid device firmware identity") {
		t.Fatalf("text = %q", payload.Text)
	}

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
	if registry.Devices[0].IdentityStatus != "invalid" {
		t.Fatalf("identity status = %q, want invalid", registry.Devices[0].IdentityStatus)
	}
	if !strings.Contains(registry.Devices[0].IdentityError, "forbidden") {
		t.Fatalf("identity error = %q", registry.Devices[0].IdentityError)
	}

	metricsResp, err := http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = metricsResp.Body.Close() })
	metricsBody, err := io.ReadAll(metricsResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metricsBody), "a21_device_identity_invalid_total 1") {
		t.Fatalf("metrics missing invalid device identity count:\n%s", metricsBody)
	}
}

func TestControlWebSocketInterruptKeepsTrace(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       1,
		TraceID:   "a21-trace-000009",
		SessionID: "a21-session-000009",
	}, protocol.DeviceEventPayload{
		Event: protocol.DeviceEventInterrupt,
		Mode:  protocol.ModeWorkmate,
	})

	events := readControlEvents(t, ctx, conn, 2)
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
	if events[0].TraceID != "a21-trace-000009" {
		t.Fatalf("trace = %q, want a21-trace-000009", events[0].TraceID)
	}
	if events[0].SessionID != "a21-session-000009" {
		t.Fatalf("session = %q, want a21-session-000009", events[0].SessionID)
	}
}

func TestControlWebSocketTouchWakeOrListenMapsToMockTurn(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       3,
		TraceID:   "a21-trace-touch-001",
		SessionID: "a21-session-touch-001",
	}, protocol.DeviceEventPayload{
		Event:       protocol.DeviceEventTouchWakeOrListen,
		Mode:        protocol.ModeWorkmate,
		Text:        "先说，我在",
		TouchSource: protocol.TouchSourceScreen,
	})

	events := readControlEvents(t, ctx, conn, 3)
	wantStates := []protocol.ExpressionState{
		protocol.ExpressionListening,
		protocol.ExpressionThinking,
		protocol.ExpressionSpeaking,
	}
	for i, want := range wantStates {
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != want {
			t.Fatalf("event %d state = %q, want %q", i, payload.State, want)
		}
	}

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
	if registry.Devices[0].LastEvent != protocol.DeviceEventTouchWakeOrListen {
		t.Fatalf("last event = %q, want touch wake", registry.Devices[0].LastEvent)
	}
	if registry.Devices[0].LastTouchSource != protocol.TouchSourceScreen {
		t.Fatalf("last touch source = %q, want screen", registry.Devices[0].LastTouchSource)
	}
}

func TestControlWebSocketTouchBargeInMapsToInterrupt(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeDeviceEvent(t, ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindDeviceEvent,
		Seq:       4,
		TraceID:   "a21-trace-touch-002",
		SessionID: "a21-session-touch-002",
	}, protocol.DeviceEventPayload{
		Event:       protocol.DeviceEventTouchBargeIn,
		Mode:        protocol.ModeWorkmate,
		TouchSource: protocol.TouchSourceTopSensor,
	})

	events := readControlEvents(t, ctx, conn, 2)
	var first protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if first.State != protocol.ExpressionInterrupted {
		t.Fatalf("first state = %q, want interrupted", first.State)
	}
	if events[0].TraceID != "a21-trace-touch-002" {
		t.Fatalf("trace = %q, want a21-trace-touch-002", events[0].TraceID)
	}
}

func TestControlWebSocketTopTouchGesturesStayDistinct(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/control"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	gestureCases := []struct {
		name      string
		seq       uint64
		traceID   string
		event     protocol.DeviceEventKind
		wantState protocol.ExpressionState
		wantMode  protocol.Mode
	}{
		{
			name:      "tap",
			seq:       5,
			traceID:   "a21-trace-touch-top-tap",
			event:     protocol.DeviceEventTouchTopTap,
			wantState: protocol.ExpressionListening,
			wantMode:  protocol.ModeWorkmate,
		},
		{
			name:      "swipe forward",
			seq:       6,
			traceID:   "a21-trace-touch-top-forward",
			event:     protocol.DeviceEventTouchTopSwipeForward,
			wantState: protocol.ExpressionThinking,
			wantMode:  protocol.ModeCoCreation,
		},
		{
			name:      "swipe backward",
			seq:       7,
			traceID:   "a21-trace-touch-top-backward",
			event:     protocol.DeviceEventTouchTopSwipeBackward,
			wantState: protocol.ExpressionIdle,
			wantMode:  protocol.ModeFocus,
		},
	}

	for _, tc := range gestureCases {
		writeDeviceEvent(t, ctx, conn, protocol.Envelope{
			Protocol:  protocol.ProtocolVersion,
			DeviceID:  "stackchan-001",
			Kind:      protocol.KindDeviceEvent,
			Seq:       tc.seq,
			TraceID:   tc.traceID,
			SessionID: "a21-session-touch-top",
		}, protocol.DeviceEventPayload{
			Event:       tc.event,
			Mode:        protocol.ModeWorkmate,
			TouchSource: protocol.TouchSourceTopSensor,
		})

		events := readControlEvents(t, ctx, conn, 1)
		var payload protocol.ControlEventPayload
		if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != tc.wantState || payload.Mode != tc.wantMode {
			t.Fatalf("%s control = %+v, want state=%q mode=%q", tc.name, payload, tc.wantState, tc.wantMode)
		}

		traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id="+tc.traceID, nil)
		traceRec := httptest.NewRecorder()
		httpServer.Config.Handler.ServeHTTP(traceRec, traceReq)
		if !strings.Contains(traceRec.Body.String(), "device."+string(tc.event)+".received") {
			t.Fatalf("%s trace missing device event marker: %s", tc.name, traceRec.Body.String())
		}
	}
}

func TestAudioWebSocketAcceptsAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
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
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-001",
		SessionID: "a21-session-audio-001",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 1)
	var payload protocol.ControlEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State != protocol.ExpressionListening {
		t.Fatalf("state = %q, want listening", payload.State)
	}
	if events[0].TraceID != "a21-trace-audio-001" {
		t.Fatalf("trace = %q, want a21-trace-audio-001", events[0].TraceID)
	}
}

func TestAudioRecentEndpointReturnsLoopbackOnlyRedactedFrames(t *testing.T) {
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

	payload := pcm16Base64WithSample(12000)
	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 7, "a21-trace-physical-capture", "a21-session-physical-capture", payload)
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), payload) {
		t.Fatalf("default recent audio response leaked raw audio: %s", rec.Body.String())
	}
	for _, want := range []string{
		`"schema_version":"a21.gateway.audio_recent.v1"`,
		`"device_id":"stackchan-001"`,
		`"session_id":"a21-session-physical-capture"`,
		`"frames"`,
		`"data_bytes":640`,
		`"speech_detected":true`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("recent response missing %q: %s", want, rec.Body.String())
		}
	}

	withAudioReq := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture&include_audio=1", nil)
	withAudioReq.RemoteAddr = "127.0.0.1:45678"
	withAudioRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(withAudioRec, withAudioReq)
	if withAudioRec.Code != http.StatusOK {
		t.Fatalf("include audio status = %d: %s", withAudioRec.Code, withAudioRec.Body.String())
	}
	if !strings.Contains(withAudioRec.Body.String(), payload) {
		t.Fatalf("include audio response missing payload: %s", withAudioRec.Body.String())
	}

	remoteReq := httptest.NewRequest(http.MethodGet, "/v1/audio/recent?device_id=stackchan-001&session_id=a21-session-physical-capture&include_audio=1", nil)
	remoteReq.RemoteAddr = "192.168.1.42:45678"
	remoteRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(remoteRec, remoteReq)
	if remoteRec.Code != http.StatusForbidden {
		t.Fatalf("remote status = %d, want 403: %s", remoteRec.Code, remoteRec.Body.String())
	}
	if strings.Contains(remoteRec.Body.String(), payload) {
		t.Fatalf("remote rejection leaked raw audio: %s", remoteRec.Body.String())
	}
}

func TestAudioWebSocketReturnsMockPlaybackChunk(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
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
		DataBase64:   pcm16Base64WithSample(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-sim-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-audio-downlink",
		SessionID: "a21-session-audio-downlink",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	events := readControlEvents(t, ctx, conn, 2)
	var speaking protocol.ControlEventPayload
	if err := json.Unmarshal(events[1].Payload, &speaking); err != nil {
		t.Fatal(err)
	}
	if speaking.State != protocol.ExpressionSpeaking || speaking.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("speaking payload = %+v", speaking)
	}
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want %q", playback.Kind, protocol.KindAudioPlaybackChunk)
	}
	if playback.TraceID != "a21-trace-audio-downlink" {
		t.Fatalf("trace = %q, want a21-trace-audio-downlink", playback.TraceID)
	}
	var payload protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.StreamID != "a21-audio-stream-000001" {
		t.Fatalf("stream = %q, want a21-audio-stream-000001", payload.StreamID)
	}
	if payload.Codec != protocol.AudioCodecPCMS16LE || payload.SampleRateHz != 16000 || payload.Channels != 1 || payload.DurationMS != 20 {
		t.Fatalf("playback payload = %+v", payload)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 640 {
		t.Fatalf("decoded payload bytes = %d, want 640", len(decoded))
	}
}

func TestAudioWebSocketReturnsAudibleMockPlaybackForPhysicalStackChan(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	audio, err := json.Marshal(protocol.AudioChunk{
		Codec:        protocol.AudioCodecPCMS16LE,
		SampleRateHz: 16000,
		Channels:     1,
		DurationMS:   20,
		DataBase64:   pcm16Base64WithSample(12000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, protocol.Envelope{
		Protocol:  protocol.ProtocolVersion,
		DeviceID:  "stackchan-001",
		Kind:      protocol.KindAudioFrame,
		Seq:       1,
		TraceID:   "a21-trace-physical-audio",
		SessionID: "a21-session-physical-audio",
		Payload:   audio,
	}); err != nil {
		t.Fatal(err)
	}

	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	var payload protocol.AudioPlaybackChunk
	if err := json.Unmarshal(playback.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 640 || bytes.Equal(decoded, make([]byte, len(decoded))) {
		t.Fatalf("physical StackChan mock playback should be audible non-silent PCM, got %d bytes", len(decoded))
	}
}

func TestAudioWebSocketDoesNotLoopPhysicalMockPlaybackForSameSpeech(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-physical-loop", "a21-session-physical-loop", pcm16Base64WithSample(12000))
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-physical-loop", "a21-session-physical-loop", pcm16Base64WithSample(12000))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}

func TestDeviceControlArmsSinglePhysicalMockPlaybackForNextAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-armed-loop","session_id":"a21-session-armed-loop","mock_playback_on_next_audio_frame":true}`)
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

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-armed-loop", "a21-session-armed-loop", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 2)
	var playback protocol.Envelope
	if err := wsjson.Read(ctx, conn, &playback); err != nil {
		t.Fatal(err)
	}
	if playback.Kind != protocol.KindAudioPlaybackChunk {
		t.Fatalf("kind = %q, want playback chunk", playback.Kind)
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-armed-loop", "a21-session-armed-loop", pcm16Base64WithSample(0))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}

func TestDeviceControlArmsMultiplePhysicalMockPlaybackChunksForNextAudioFrame(t *testing.T) {
	httpServer := httptest.NewServer(NewServer().Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, webSocketURL(httpServer.URL, "/ws/audio?device_id=stackchan-001"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	body := bytes.NewBufferString(`{"device_id":"stackchan-001","state":"listening","mode":"workmate","trace_id":"a21-trace-armed-multi","session_id":"a21-session-armed-multi","mock_playback_on_next_audio_frame":true,"mock_audio_chunks":4}`)
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

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 1, "a21-trace-armed-multi", "a21-session-armed-multi", pcm16Base64WithSample(0))
	readControlEvents(t, ctx, conn, 2)
	for i := 0; i < 4; i++ {
		var playback protocol.Envelope
		if err := wsjson.Read(ctx, conn, &playback); err != nil {
			t.Fatal(err)
		}
		if playback.Kind != protocol.KindAudioPlaybackChunk {
			t.Fatalf("envelope %d kind = %q, want playback chunk", i, playback.Kind)
		}
	}

	writeAudioFrameEnvelopeForDevice(t, ctx, conn, "stackchan-001", 2, "a21-trace-armed-multi", "a21-session-armed-multi", pcm16Base64WithSample(0))
	assertNoEnvelope(t, conn, 100*time.Millisecond)
}

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

type blockingProfessionalASRAdapter struct {
	started     chan struct{}
	released    chan struct{}
	canceled    chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
	cancelOnce  sync.Once
}

func newBlockingProfessionalASRAdapter() *blockingProfessionalASRAdapter {
	return &blockingProfessionalASRAdapter{
		started:  make(chan struct{}),
		released: make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (a *blockingProfessionalASRAdapter) Name() string {
	return "a21-blocking-professional-asr"
}

func (a *blockingProfessionalASRAdapter) release() {
	a.releaseOnce.Do(func() {
		close(a.released)
	})
}

func (a *blockingProfessionalASRAdapter) Transcribe(ctx context.Context, req providers.ASRAdapterRequest) (<-chan providers.ASRAdapterEvent, error) {
	out := make(chan providers.ASRAdapterEvent, 1)
	a.startOnce.Do(func() {
		close(a.started)
	})
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			a.cancelOnce.Do(func() {
				close(a.canceled)
			})
		case <-a.released:
			select {
			case <-ctx.Done():
				a.cancelOnce.Do(func() {
					close(a.canceled)
				})
			case out <- providers.ASRAdapterEvent{Text: "释放后的专业查询", Final: true}:
			}
		}
	}()
	return out, nil
}

type passthroughTextStreamAdapter struct{}

func (passthroughTextStreamAdapter) Name() string {
	return "a21-passthrough-text-stream"
}

func (passthroughTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type blockingSegmentTextStreamAdapter struct {
	firstSegmentSent chan struct{}
	release          chan struct{}
	finalReleased    chan struct{}
	canceled         chan struct{}
	firstOnce        sync.Once
	releaseOnce      sync.Once
	finalOnce        sync.Once
	cancelOnce       sync.Once
}

func newBlockingSegmentTextStreamAdapter() *blockingSegmentTextStreamAdapter {
	return &blockingSegmentTextStreamAdapter{
		firstSegmentSent: make(chan struct{}),
		release:          make(chan struct{}),
		finalReleased:    make(chan struct{}),
		canceled:         make(chan struct{}),
	}
}

func (a *blockingSegmentTextStreamAdapter) Name() string {
	return "a21-blocking-segment-text-stream"
}

func (a *blockingSegmentTextStreamAdapter) releaseFinal() {
	a.releaseOnce.Do(func() {
		close(a.release)
	})
}

func (a *blockingSegmentTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent)
	go func() {
		defer close(out)
		cancel := func() {
			a.cancelOnce.Do(func() {
				close(a.canceled)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第一句。"}:
			a.firstOnce.Do(func() {
				close(a.firstSegmentSent)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-a.release:
			a.finalOnce.Do(func() {
				close(a.finalReleased)
			})
		}
		select {
		case <-ctx.Done():
			cancel()
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第二句。"}:
		}
		select {
		case <-ctx.Done():
			cancel()
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type segmentChunkTTSAdapter struct{}

func (segmentChunkTTSAdapter) Name() string {
	return "a21-segment-chunk-tts"
}

func (segmentChunkTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	out := make(chan providers.VoiceAudioChunk, 1)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case out <- xiaozhiTestVoiceAudioChunk():
		}
	}()
	return out, nil
}

type singleSentenceTextStreamAdapter struct{}

func (singleSentenceTextStreamAdapter) Name() string {
	return "a21-single-sentence-text-stream"
}

func (singleSentenceTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 2)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "第一句。"}:
		}
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type chunkThenErrorTTSAdapter struct{}

func (chunkThenErrorTTSAdapter) Name() string {
	return "a21-chunk-then-error-tts"
}

func (chunkThenErrorTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	out := make(chan providers.VoiceAudioChunk, 2)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- xiaozhiTestVoiceAudioChunk():
		}
		select {
		case <-ctx.Done():
		case out <- providers.VoiceAudioChunk{Err: errors.New("provider closed after first audio")}:
		}
	}()
	return out, nil
}

type gatewayFallbackTextStreamAdapter struct{}

func (gatewayFallbackTextStreamAdapter) Name() string {
	return "a21-gateway-fallback-text-stream"
}

func (gatewayFallbackTextStreamAdapter) StreamText(ctx context.Context, req providers.TextStreamAdapterRequest) (<-chan providers.TextStreamEvent, error) {
	out := make(chan providers.TextStreamEvent, 3)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{
			Finding: "provider_fallback_used",
			Fallback: &providers.TextStreamFallbackEvent{
				Activated: true,
				Provider:  "a21_voice_fallback",
				Reason:    "primary_failed",
			},
		}:
		}
		select {
		case <-ctx.Done():
			return
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaContent, Text: "fallback voice answer。"}:
		}
		select {
		case <-ctx.Done():
		case out <- providers.TextStreamEvent{Kind: providers.TextStreamDeltaDone}:
		}
	}()
	return out, nil
}

type recordingXiaozhiPipelineRunner struct {
	requests chan providers.VoicePipelineRequest
	delegate xiaozhiVoicePipelineRunner
}

func newRecordingXiaozhiPipelineRunner() *recordingXiaozhiPipelineRunner {
	return &recordingXiaozhiPipelineRunner{
		requests: make(chan providers.VoicePipelineRequest, 1),
		delegate: providers.NewVoicePipelineRunner(providers.VoicePipelineAdapters{
			ASR:        providers.NewMockASRAdapter("mock-local-asr"),
			TextStream: providers.NewMockTextStreamAdapter("mock-text-stream"),
			TTS:        providers.NewMockTTSAdapter("mock-fast-tts"),
			Selection:  providers.VoicePipelineSelectionFromEnv(nil),
		}),
	}
}

func (r *recordingXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	captured := req
	captured.Frames = append([]providers.VoicePipelinePCMFrame(nil), req.Frames...)
	for i := range captured.Frames {
		captured.Frames[i].PCM16LE = append([]byte(nil), req.Frames[i].PCM16LE...)
	}
	select {
	case r.requests <- captured:
	default:
	}
	return r.delegate.Run(ctx, req)
}

type fallbackReportingXiaozhiPipelineRunner struct{}

func (fallbackReportingXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	select {
	case <-ctx.Done():
		return providers.VoicePipelineResult{Status: providers.VoicePipelineStatusCancelled}, ctx.Err()
	default:
	}
	return providers.VoicePipelineResult{
		Status: providers.VoicePipelineStatusCompleted,
		Timing: providers.VoicePipelineTiming{
			ASRFirstPartialMS:       10,
			ASRFinalMS:              20,
			LLMFirstContentMS:       30,
			TTSFirstAudioMS:         40,
			AudioDownlinkFirstMS:    50,
			SpeechEndToFinalASRMS:   20,
			SpeechEndToFirstTokenMS: 30,
		},
		AudioChunks: []providers.VoiceAudioChunk{{
			Codec:        string(protocol.AudioCodecPCMS16LE),
			SampleRateHz: 24000,
			Channels:     1,
			DurationMS:   60,
			DataBase64:   base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11, 0x00}, 1440)),
		}},
		Report: providers.VoicePipelineReport{
			SchemaVersion: "a21.voice_pipeline.fixture.v1",
			Status:        string(providers.VoicePipelineStatusCompleted),
			TraceID:       req.Session.TraceID,
			SessionID:     req.Session.SessionID,
			DeviceID:      req.Session.DeviceID,
			Mode:          req.Mode,
			ExecutionMode: "fixture",
			Selection: providers.VoicePipelineSelection{
				LLMProfile:            "deepseek",
				LLMProfileEnv:         "A21_PROVIDER_PRIMARY",
				LLMFallbackProfile:    "a21_voice_fallback",
				LLMFallbackProfileEnv: "A21_TEXT_STREAM_FALLBACK_PROFILE",
			},
			Output: providers.VoicePipelineOutputReport{
				AudioChunkCount: 1,
				AudioCodec:      string(protocol.AudioCodecPCMS16LE),
				SampleRateHz:    24000,
				Channels:        1,
				ChunkDurationMS: 60,
				LLMContentChars: 21,
			},
			Fallback: &providers.VoicePipelineFallbackReport{
				Activated: true,
				Provider:  "a21_voice_fallback",
				Reason:    "primary_failed",
			},
			Findings: []string{"provider_fallback_used"},
		},
	}, nil
}

type unavailableXiaozhiPipelineRunner struct{}

func (unavailableXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	return unavailableXiaozhiPipelineResult(req), errors.New("a21 provider unavailable")
}

func (unavailableXiaozhiPipelineRunner) RunStream(ctx context.Context, req providers.VoicePipelineRequest) (<-chan providers.VoicePipelineStreamEvent, error) {
	events := make(chan providers.VoicePipelineStreamEvent, 1)
	go func() {
		defer close(events)
		select {
		case <-ctx.Done():
			return
		case events <- providers.VoicePipelineStreamEvent{
			Kind:   providers.VoicePipelineStreamDone,
			Result: unavailableXiaozhiPipelineResult(req),
			Err:    errors.New("a21 provider unavailable"),
		}:
		}
	}()
	return events, nil
}

func unavailableXiaozhiPipelineResult(req providers.VoicePipelineRequest) providers.VoicePipelineResult {
	return providers.VoicePipelineResult{
		Status: providers.VoicePipelineStatusFailed,
		Timing: providers.VoicePipelineTiming{
			ASRFirstPartialMS:       -1,
			ASRFinalMS:              -1,
			LLMFirstContentMS:       -1,
			TTSFirstAudioMS:         -1,
			AudioDownlinkFirstMS:    -1,
			ProviderCancelMS:        -1,
			BargeInStopMS:           -1,
			SpeechEndToFinalASRMS:   -1,
			SpeechEndToFirstTokenMS: -1,
		},
		Report: providers.VoicePipelineReport{
			SchemaVersion: "a21.voice_pipeline.fixture.v1",
			Status:        string(providers.VoicePipelineStatusFailed),
			TraceID:       req.Session.TraceID,
			SessionID:     req.Session.SessionID,
			DeviceID:      req.Session.DeviceID,
			Mode:          req.Mode,
			ExecutionMode: "fixture",
			Selection: providers.VoicePipelineSelection{
				LLMProfile:    "deepseek",
				LLMProfileEnv: "A21_PROVIDER_PRIMARY",
			},
			Findings: []string{"text stream adapter failed"},
		},
	}
}

type failingXiaozhiTTSAdapter struct{}

func (failingXiaozhiTTSAdapter) Name() string {
	return "a21-failing-tts"
}

func (failingXiaozhiTTSAdapter) Synthesize(ctx context.Context, req providers.TTSAdapterRequest) (<-chan providers.VoiceAudioChunk, error) {
	return nil, errors.New("a21 tts unavailable")
}

type slowAnswerXiaozhiPipelineRunner struct {
	entered      chan struct{}
	canceled     chan struct{}
	release      chan struct{}
	released     chan struct{}
	enteredOnce  sync.Once
	canceledOnce sync.Once
	releaseOnce  sync.Once
}

func newSlowAnswerXiaozhiPipelineRunner() *slowAnswerXiaozhiPipelineRunner {
	return &slowAnswerXiaozhiPipelineRunner{
		entered:  make(chan struct{}),
		canceled: make(chan struct{}),
		release:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

func (r *slowAnswerXiaozhiPipelineRunner) releaseAnswer() {
	r.releaseOnce.Do(func() {
		close(r.release)
		close(r.released)
	})
}

func (r *slowAnswerXiaozhiPipelineRunner) Run(ctx context.Context, req providers.VoicePipelineRequest) (providers.VoicePipelineResult, error) {
	r.enteredOnce.Do(func() {
		close(r.entered)
	})
	select {
	case <-ctx.Done():
		r.canceledOnce.Do(func() {
			close(r.canceled)
		})
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
			Status: providers.VoicePipelineStatusCompleted,
			Timing: providers.VoicePipelineTiming{
				ASRFirstPartialMS:       10,
				ASRFinalMS:              20,
				LLMFirstContentMS:       30,
				TTSFirstAudioMS:         40,
				AudioDownlinkFirstMS:    40,
				ProviderCancelMS:        -1,
				BargeInStopMS:           -1,
				SpeechEndToFinalASRMS:   20,
				SpeechEndToFirstTokenMS: 30,
			},
			AudioChunks: []providers.VoiceAudioChunk{xiaozhiTestVoiceAudioChunk()},
			Report: providers.VoicePipelineReport{
				SchemaVersion: "a21.voice_pipeline.fixture.v1",
				Status:        string(providers.VoicePipelineStatusCompleted),
				TraceID:       req.Session.TraceID,
				SessionID:     req.Session.SessionID,
				DeviceID:      req.Session.DeviceID,
				Mode:          req.Mode,
				ExecutionMode: "fixture",
				Output: providers.VoicePipelineOutputReport{
					AudioChunkCount: 1,
				},
				Redaction: providers.VoicePipelineRedactionPolicies{
					TranscriptPolicy:     "transcript_not_recorded",
					ProviderOutputPolicy: "provider_output_not_recorded",
					AudioPayloadPolicy:   "audio_payload_not_recorded",
					URLPolicy:            "full_url_not_recorded",
					ProxyPolicy:          "proxy_value_not_recorded",
					LocalPathPolicy:      "local_path_not_recorded",
				},
			},
		}, nil
	}
}

func xiaozhiTestVoiceAudioChunk() providers.VoiceAudioChunk {
	return providers.VoiceAudioChunk{
		Codec:        "pcm_s16le",
		SampleRateHz: 24000,
		Channels:     1,
		DurationMS:   60,
		DataBase64:   base64.StdEncoding.EncodeToString(make([]byte, 2880)),
	}
}

func xiaozhiTestOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := codec.EncodePCM16(make([]int16, codec.FrameSamples()))
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestSpeechOpusPacket(t *testing.T) []byte {
	t.Helper()
	codec, err := opuscodec.New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]int16, codec.FrameSamples())
	for i := range pcm {
		if (i/10)%2 == 0 {
			pcm[i] = 12000
		} else {
			pcm[i] = -12000
		}
	}
	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func xiaozhiTestPCM16Base64(sampleRate int, durationMS int, sample int16) string {
	sampleCount := sampleRate * durationMS / 1000
	data := make([]byte, sampleCount*2)
	for i := 0; i < sampleCount; i++ {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return base64.StdEncoding.EncodeToString(data)
}

func maxAbsPCM16(pcm []int16) int {
	maxAbs := 0
	for _, sample := range pcm {
		abs := int(sample)
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	return maxAbs
}

func readXiaozhiJSON(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if messageType == websocket.MessageBinary {
			continue
		}
		if messageType != websocket.MessageText {
			t.Fatalf("xiaozhi message type=%v, want text JSON", messageType)
		}
		var message map[string]any
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("failed to read JSON message: %v", err)
		}
		return message
	}
}

func readXiaozhiBinary(t *testing.T, ctx context.Context, conn *websocket.Conn) []byte {
	t.Helper()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageBinary || len(data) == 0 {
		t.Fatalf("xiaozhi binary message type=%v bytes=%d, want non-empty binary opus", messageType, len(data))
	}
	return data
}

type xiaozhiObservedFrame struct {
	messageType websocket.MessageType
	json        map[string]any
	byteCount   int
}

func readXiaozhiFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) xiaozhiObservedFrame {
	t.Helper()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType == websocket.MessageBinary {
		if len(data) == 0 {
			t.Fatal("xiaozhi binary frame was empty")
		}
		return xiaozhiObservedFrame{messageType: messageType, byteCount: len(data)}
	}
	if messageType != websocket.MessageText {
		t.Fatalf("xiaozhi message type=%v, want text JSON or binary Opus", messageType)
	}
	var message map[string]any
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatalf("failed to parse xiaozhi JSON frame: %v", err)
	}
	return xiaozhiObservedFrame{messageType: messageType, json: message, byteCount: len(data)}
}

func assertXiaozhiProductAnswerSequence(t *testing.T, ctx context.Context, conn *websocket.Conn) {
	t.Helper()
	want := []struct {
		kind  string
		state string
		phase string
	}{
		{kind: "stt"},
		{kind: "tts", state: "start"},
		{kind: "tts", state: "sentence_start", phase: "fast_ack"},
		{kind: "binary"},
		{kind: "tts", state: "sentence_start", phase: "answer"},
		{kind: "binary"},
		{kind: "tts", state: "stop"},
	}
	for i, expected := range want {
		frame := readXiaozhiFrame(t, ctx, conn)
		if expected.kind == "binary" {
			if frame.messageType != websocket.MessageBinary || frame.byteCount == 0 {
				t.Fatalf("frame %d = %s, want non-empty binary Opus", i, xiaozhiFrameSummary(frame))
			}
			continue
		}
		if frame.messageType != websocket.MessageText {
			t.Fatalf("frame %d = %s, want JSON %s", i, xiaozhiFrameSummary(frame), expected.kind)
		}
		gotKind, _ := frame.json["type"].(string)
		if gotKind != expected.kind {
			t.Fatalf("frame %d type = %q, want %q", i, gotKind, expected.kind)
		}
		if expected.state != "" {
			gotState, _ := frame.json["state"].(string)
			if gotState != expected.state {
				t.Fatalf("frame %d state = %q, want %q for %s", i, gotState, expected.state, expected.kind)
			}
		}
		if expected.phase != "" {
			gotPhase, _ := frame.json["phase"].(string)
			if gotPhase != expected.phase {
				t.Fatalf("frame %d phase = %q, want %q for %s/%s", i, gotPhase, expected.phase, expected.kind, expected.state)
			}
		}
		if expected.kind == "stt" {
			text, _ := frame.json["text"].(string)
			if strings.TrimSpace(text) == "" {
				t.Fatalf("frame %d stt text was empty", i)
			}
		}
	}
}

func xiaozhiFrameSummary(frame xiaozhiObservedFrame) string {
	if frame.messageType == websocket.MessageBinary {
		return fmt.Sprintf("binary bytes=%d", frame.byteCount)
	}
	msgType, _ := frame.json["type"].(string)
	state, _ := frame.json["state"].(string)
	phase, _ := frame.json["phase"].(string)
	return fmt.Sprintf("json type=%q state=%q phase=%q", msgType, state, phase)
}

func xiaozhiProtocol2Wire(payload []byte, timestampMS uint32) []byte {
	wire := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint16(wire[0:2], 2)
	binary.BigEndian.PutUint16(wire[2:4], 0)
	binary.BigEndian.PutUint32(wire[8:12], timestampMS)
	binary.BigEndian.PutUint32(wire[12:16], uint32(len(payload)))
	copy(wire[16:], payload)
	return wire
}

func assertOfficialStackChanState(t *testing.T, ctx context.Context, conn *websocket.Conn, state string) {
	t.Helper()
	seenAvatar := false
	seenMotion := false
	for i := 0; i < 2; i++ {
		messageType, frame, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if messageType != websocket.MessageBinary {
			t.Fatalf("official stackchan message type = %v, want binary", messageType)
		}
		if len(frame) < 5 {
			t.Fatalf("official stackchan frame too short: %#v", frame)
		}
		if got := binary.BigEndian.Uint32(frame[1:5]); got != uint32(len(frame)-5) {
			t.Fatalf("official stackchan frame length = %d, want %d", got, len(frame)-5)
		}
		switch frame[0] {
		case 0x03:
			seenAvatar = true
			var payload map[string]map[string]int
			if err := json.Unmarshal(frame[5:], &payload); err != nil {
				t.Fatal(err)
			}
			wantMouthWeight := 0
			if state == "speaking" {
				wantMouthWeight = 82
			}
			if payload["mouth"]["weight"] != wantMouthWeight {
				t.Fatalf("official avatar payload = %#v, want %s mouth weight %d", payload, state, wantMouthWeight)
			}
		case 0x04:
			seenMotion = true
			var payload map[string]map[string]int
			if err := json.Unmarshal(frame[5:], &payload); err != nil {
				t.Fatal(err)
			}
			wantAngle := map[string]int{
				"listening": 380,
				"thinking":  520,
				"speaking":  480,
			}[state]
			if wantAngle == 0 {
				wantAngle = 450
			}
			if payload["pitchServo"]["angle"] != wantAngle {
				t.Fatalf("official motion payload = %#v, want %s pitch %d", payload, state, wantAngle)
			}
		default:
			t.Fatalf("official stackchan frame type = %#x, want avatar or motion", frame[0])
		}
	}
	if !seenAvatar || !seenMotion {
		t.Fatalf("official stackchan state %s frames missing avatar=%v motion=%v", state, seenAvatar, seenMotion)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func webSocketURL(serverURL string, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}

type scriptedVoiceProvider struct {
	events []providers.VoiceEvent
}

func (p scriptedVoiceProvider) Name() string {
	return "scripted"
}

func (p scriptedVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, len(p.events))
	defer close(events)
	for _, event := range p.events {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p scriptedVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{Session: req.Session, Kind: providers.VoiceEventCancelled, Text: "cancelled", Final: true}
	return events, nil
}

func (p scriptedVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p scriptedVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type unavailableVoiceProvider struct{}

func (p unavailableVoiceProvider) Name() string {
	return "unavailable"
}

func (p unavailableVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	return nil, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{
		Provider:   p.Name(),
		Status:     providers.VoiceProviderUnavailable,
		Configured: false,
		Realtime:   false,
		Detail:     "test unavailable",
	}, providers.ErrVoiceProviderUnavailable
}

func (p unavailableVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingVoiceProvider struct {
	startEvents   []providers.VoiceEvent
	startCalls    int
	cancelRequest providers.VoiceCancelRequest
}

func (p *capturingVoiceProvider) Name() string {
	return "capturing"
}

func (p *capturingVoiceProvider) StartTurn(ctx context.Context, req providers.VoiceTurnRequest) (<-chan providers.VoiceEvent, error) {
	p.startCalls++
	events := make(chan providers.VoiceEvent, len(p.startEvents))
	defer close(events)
	for _, event := range p.startEvents {
		event.Session = req.Session
		events <- event
	}
	return events, nil
}

func (p *capturingVoiceProvider) Cancel(ctx context.Context, req providers.VoiceCancelRequest) (<-chan providers.VoiceEvent, error) {
	p.cancelRequest = req
	events := make(chan providers.VoiceEvent, 1)
	defer close(events)
	events <- providers.VoiceEvent{
		Session:  req.Session,
		Kind:     providers.VoiceEventCancelled,
		Text:     "capturing cancelled",
		Final:    true,
		StreamID: req.StreamID,
	}
	return events, nil
}

func (p *capturingVoiceProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

func (p *capturingVoiceProvider) Close(ctx context.Context) error {
	return ctx.Err()
}

type capturingRealtimeAudioProvider struct {
	capturingVoiceProvider
	startCalls    int
	session       providers.VoiceSession
	sessionHandle *capturingRealtimeVoiceSession
	closed        chan struct{}
}

func (p *capturingRealtimeAudioProvider) Name() string {
	return "capturing-realtime-audio"
}

func (p *capturingRealtimeAudioProvider) StartRealtimeSession(ctx context.Context, session providers.VoiceSession) (providers.RealtimeVoiceSession, error) {
	p.startCalls++
	p.session = session
	p.sessionHandle = &capturingRealtimeVoiceSession{closed: p.closed, events: make(chan providers.VoiceEvent, 4)}
	return p.sessionHandle, ctx.Err()
}

func (p *capturingRealtimeAudioProvider) Health(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.VoiceProviderHealth{Provider: p.Name(), Status: providers.VoiceProviderHealthy, Configured: true, Realtime: true}, ctx.Err()
}

type capturingRealtimeVoiceSession struct {
	audioChunks []protocol.AudioChunk
	commitCalls int
	cancelCalls int
	closed      chan struct{}
	events      chan providers.VoiceEvent
}

func (s *capturingRealtimeVoiceSession) SendAudio(ctx context.Context, chunk protocol.AudioChunk) error {
	s.audioChunks = append(s.audioChunks, chunk)
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) CommitAndCreateResponse(ctx context.Context) error {
	s.commitCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Cancel(ctx context.Context, _ providers.VoiceCancelRequest) error {
	s.cancelCalls++
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Close(ctx context.Context) error {
	if s.closed != nil {
		select {
		case s.closed <- struct{}{}:
		default:
		}
	}
	return ctx.Err()
}

func (s *capturingRealtimeVoiceSession) Events() <-chan providers.VoiceEvent {
	return s.events
}

type countingV21Client struct {
	calls int
}

func (c *countingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.calls++
	return v21adapter.NewMockClient().Query(ctx, request)
}

type capturingV21Client struct {
	request v21adapter.QueryRequest
}

func (c *capturingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.request = request
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "V21 找到一条可引用证据。",
		Confidence: 0.9,
		Evidence: []v21adapter.Evidence{{
			Title:    "adapter source",
			Type:     "v21_retrieval_evidence",
			SourceID: "v21-doc-contract-001",
			Summary:  "sanitized evidence fixture",
		}},
		SpeechBlocks: []string{"V21 找到一条可引用证据。"},
		ScreenCards:  []v21adapter.ScreenCard{{Label: "结论", Text: "1 条证据"}},
		FollowUps:    []string{"要不要展开来源？"},
	}, nil
}

type failingV21Client struct{}

func (failingV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	return v21adapter.QueryResponse{}, errors.New("v21 unavailable")
}

type slowV21Client struct {
	delay time.Duration
}

func (c slowV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	timer := time.NewTimer(c.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return v21adapter.QueryResponse{}, ctx.Err()
	case <-timer.C:
		return v21adapter.NewMockClient().Query(ctx, request)
	}
}

type delayedXiaozhiProfessionalV21Client struct {
	delay    time.Duration
	started  chan struct{}
	released chan struct{}
	canceled chan struct{}
	once     sync.Once
	mu       sync.Mutex
	requests []v21adapter.QueryRequest
}

func newDelayedXiaozhiProfessionalV21Client(delay time.Duration) *delayedXiaozhiProfessionalV21Client {
	return &delayedXiaozhiProfessionalV21Client{
		delay:    delay,
		started:  make(chan struct{}),
		released: make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (c *delayedXiaozhiProfessionalV21Client) release() {
	c.once.Do(func() {
		close(c.released)
	})
}

func (c *delayedXiaozhiProfessionalV21Client) lastUtterance() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		return ""
	}
	return c.requests[len(c.requests)-1].Utterance
}

func (c *delayedXiaozhiProfessionalV21Client) Query(ctx context.Context, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	c.mu.Lock()
	c.requests = append(c.requests, request)
	c.mu.Unlock()
	select {
	case <-c.started:
	default:
		close(c.started)
	}
	if c.delay > 0 {
		timer := time.NewTimer(c.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			close(c.canceled)
			return v21adapter.QueryResponse{}, ctx.Err()
		case <-timer.C:
		}
	} else if c.delay < 0 {
		select {
		case <-ctx.Done():
			close(c.canceled)
			return v21adapter.QueryResponse{}, ctx.Err()
		case <-c.released:
		}
	}
	select {
	case <-ctx.Done():
		close(c.canceled)
		return v21adapter.QueryResponse{}, ctx.Err()
	default:
	}
	return v21adapter.QueryResponse{
		TraceID:    request.TraceID,
		FastAnswer: "结论：需要按可引用证据复核。",
		Confidence: 0.77,
		Evidence: []v21adapter.Evidence{{
			Title:    "raw evidence title",
			Type:     "meeting",
			SourceID: "v21-doc-secret-raw",
			Summary:  "RAW_SECRET_EVIDENCE_BODY",
			Quote:    "RAW_SECRET_QUOTE",
		}},
		SpeechBlocks: []string{"RAW_SECRET_SPEECH_BLOCK"},
		ScreenCards:  []v21adapter.ScreenCard{{Label: "结论", Text: "RAW_SECRET_CARD_TEXT"}},
		FollowUps:    []string{"RAW_SECRET_FOLLOW_UP"},
	}, nil
}

type gatewaySileroVADRunner struct {
	decision audio.VADDecision
	err      error
	calls    int
}

func (r *gatewaySileroVADRunner) Detect(ctx context.Context, frame audio.Frame) (audio.VADDecision, error) {
	r.calls++
	if r.err != nil {
		return audio.VADDecision{}, r.err
	}
	return r.decision, ctx.Err()
}

func traceEventAtMS(events []TraceEvent, name string) (int64, bool) {
	for _, event := range events {
		if event.Name == name {
			return event.AtMS, true
		}
	}
	return 0, false
}

func traceEventCount(events []TraceEvent, name string) int {
	count := 0
	for _, event := range events {
		if event.Name == name {
			count++
		}
	}
	return count
}
