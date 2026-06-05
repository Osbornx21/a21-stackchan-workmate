package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"a21.local/a21/internal/providers"
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

func TestSimulatorRouteNotServedByProductGateway(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/simulator", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestWorkspaceConsolePageServed(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/workspace", nil)
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
		"A21 工作台控制台",
		`data-testid="workspace-console-root"`,
		"/v1/professional-workspace",
		"/v1/workspace-documents",
		"/v1/workspace-index-jobs",
		"/v1/workspace-sources",
		"/v1/workspace-device-bindings",
		"/v1/workspace-upload-jobs",
		"/v1/professional-read-records",
		"/v1/devices",
		"/v1/roleplay-profile",
		"/v1/voice-chain-profiles",
		"/v1/wake-word",
		"/v1/voice-modes",
		"/v1/voice-mode-ritual",
		"/v1/voice-mode-ritual-acceptance",
		"/v1/hardware-acceptance",
		"/v1/fast-companion/turn",
		"/v1/professional-query",
		"/v1/xiaozhi/body-preset",
		"/v1/xiaozhi/body-motion",
		"/v1/xiaozhi/body-scene",
		"/v1/xiaozhi/body-scene-acceptance",
		"/v1/xiaozhi/device-status",
		"/v1/xiaozhi/screen-brightness",
		"/v1/xiaozhi/screen-theme",
		"/v1/xiaozhi/mcp-capabilities",
		"/v1/xiaozhi/mcp-control",
		"/v1/stackchan/official/control",
		"/v1/stackchan/official/status",
		"/v1/traces",
		`id="queryScope"`,
		`id="documentLabel"`,
		`id="deviceId"`,
		`id="documentFile"`,
		`id="uploadDocument"`,
		`id="requestIndex"`,
		`id="bindDevice"`,
		`id="revokeDevice"`,
		`id="refreshConnectedDevice"`,
		`id="refreshDeviceBindings"`,
		`id="deleteSource"`,
		`id="exportMetadata"`,
		`id="refreshSources"`,
		`id="refreshReads"`,
		`id="readRecordFilter"`,
		`id="readTraceFilter"`,
		`id="readSessionFilter"`,
		`id="clearReadFilters"`,
		`id="deviceBindingStatus"`,
		`id="deviceBindingCount"`,
		`id="roleplayProfileSelect"`,
		`id="roleplayScenarioSelect"`,
		`id="roleplayVoiceSelect"`,
		`id="roleplayMemoryHint"`,
		`id="saveRoleplaySetup"`,
		`id="clearRoleplayMemory"`,
		`id="roleplayPromptStatus"`,
		`id="roleplayProfileStatus"`,
		`id="roleplayScenarioStatus"`,
		`id="roleplayVoiceStatus"`,
		`id="roleplayMemoryStatus"`,
		`id="voiceChainModeSelect"`,
		`id="voiceChainASRSelect"`,
		`id="voiceChainLLMSelect"`,
		`id="voiceChainRealtimeSelect"`,
		`id="saveVoiceChainSetup"`,
		`id="voiceChainModeStatus"`,
		`id="voiceChainASRStatus"`,
		`id="voiceChainLLMStatus"`,
		`id="voiceChainTTSStatus"`,
		`id="voiceChainRealtimeStatus"`,
		`id="wakeWordModeSelect"`,
		`id="wakeWordPhrase"`,
		`id="wakeWordPinyin"`,
		`id="wakeWordThreshold"`,
		`id="saveWakeWordSetup"`,
		`id="resetWakeWordSetup"`,
		`id="wakeWordBuildStatus"`,
		`id="wakeWordHotSwapStatus"`,
		`id="wakeWordActiveStatus"`,
		`id="wakeWordRuntimeStatus"`,
		`id="wakeWordFirmwareStatus"`,
		`id="wakeWordCodeStatus"`,
		`id="voiceProbeModeSelect"`,
		`id="voiceProbeInput"`,
		`id="runRoleplayProbe"`,
		`id="runProfessionalProbe"`,
		`id="refreshVoiceProbeTrace"`,
		`id="voiceProbeRouteStatus"`,
		`id="voiceProbeTraceStatus"`,
		`id="voiceProbeRoleplayStatus"`,
		`id="voiceProbeVoiceStatus"`,
		`id="voiceProbeMemoryStatus"`,
		`id="voiceProbeProfessionalStatus"`,
		`id="voiceProbeTraceList"`,
		`id="voiceProbeReadList"`,
		`id="bodyPresetActions"`,
		`id="bodyPresetStatus"`,
		`id="bodyPresetPhysicalStatus"`,
		`id="bodyPresetTraceStatus"`,
		`id="bodyPresetLEDStatus"`,
		`id="bodyPresetHeadStatus"`,
		`id="bodyPresetTransportStatus"`,
		`id="bodyPresetTraceList"`,
		`id="refreshBodyPresetTrace"`,
		`id="hardwareSceneActions"`,
		`id="refreshHardwareSceneTrace"`,
		`id="hardwareSceneStatus"`,
		`id="hardwareScenePhysicalStatus"`,
		`id="hardwareSceneTraceStatus"`,
		`id="hardwareSceneScreenStatus"`,
		`id="hardwareSceneBodyStatus"`,
		`id="hardwareSceneStepStatus"`,
		`id="hardwareSceneTransportStatus"`,
		`id="acceptHardwareScenePhysical"`,
		`id="hardwareSceneAcceptanceStatus"`,
		`id="hardwareSceneTraceList"`,
		`id="hardwareScreenStatus"`,
		`id="hardwareScreenPhysicalStatus"`,
		`id="screenBrightness"`,
		`id="screenBrightnessValue"`,
		`id="applyScreenBrightness"`,
		`id="runDeviceStatus"`,
		`id="runScreenInfo"`,
		`id="refreshMCPCapabilities"`,
		`id="refreshHardwareScreenTrace"`,
		`id="hardwareScreenTraceStatus"`,
		`id="screenBrightnessStatus"`,
		`id="screenThemeStatus"`,
		`id="screenToolStatus"`,
		`id="mcpCapabilitiesStatus"`,
		`id="hardwareScreenTraceList"`,
		`id="officialActionControls"`,
		`id="officialActionYAngle"`,
		`id="officialActionYAngleValue"`,
		`id="refreshOfficialActionTrace"`,
		`id="officialActionStatus"`,
		`id="officialActionPhysicalStatus"`,
		`id="officialActionTraceStatus"`,
		`id="officialActionEventStatus"`,
		`id="officialActionPacketStatus"`,
		`id="officialActionTransportStatus"`,
		`id="officialActionSurfaceStatus"`,
		`id="officialActionTraceList"`,
		`id="modeRitualActions"`,
		`id="modeRitualStatus"`,
		`id="modeRitualTraceStatus"`,
		`id="modeRitualPhysicalStatus"`,
		`id="acceptModeRitualPhysical"`,
		`data-mode-ritual="roleplay"`,
		`data-mode-ritual="professional"`,
		`运行陪伴切换仪式`,
		`运行专业切换仪式`,
		`确认可见模式仪式`,
		`id="hardwareAcceptanceStatus"`,
		`id="hardwareAcceptanceItems"`,
		`连接设备`,
		`刷新验收`,
		`验收看板`,
		`data-body-preset="ready"`,
		`data-body-preset="listening"`,
		`data-body-preset="thinking"`,
		`data-body-preset="speaking"`,
		`data-body-preset="celebrate"`,
		`data-body-preset="reset_idle"`,
		`data-body-motion="look_up"`,
		`data-body-motion="nod"`,
		`data-body-motion="shake"`,
		`data-body-motion="dance"`,
		`data-body-motion="stop"`,
		`data-hardware-scene="showtime"`,
		`data-hardware-scene="focus"`,
		`data-hardware-scene="reset"`,
		`data-hardware-scene="full_check"`,
		`确认可见全量检查`,
		`acceptHardwareScenePhysical`,
		`data-screen-theme="light"`,
		`data-screen-theme="dark"`,
		`data-screen-theme="auto"`,
		`data-official-state="idle"`,
		`data-official-state="listening"`,
		`data-official-state="thinking"`,
		`data-official-state="speaking"`,
		`data-official-face="happy"`,
		`data-official-face="attentive"`,
		`data-official-motion="look_up"`,
		`data-official-motion="nod"`,
		`data-official-motion="shake"`,
		`data-official-motion="dance"`,
		`data-official-motion="stop"`,
		`id="sourceList"`,
		`id="readList"`,
		`id="roleplayExpression"`,
		"陪伴模式设置",
		"语音链路设置",
		"唤醒词设置",
		"语音探针",
		"机身预设",
		"硬件场景",
		"屏幕控制",
		"官方动作",
		"运行陪伴探针",
		"运行专业探针",
		"提示输入就绪=否",
		"热切换=是",
		"需要构建=否",
		"运行时热切换=否",
		"路径=空闲",
		"追踪=无",
		"professional_query",
		"fast_companion_hybrid",
		"a21_roleplay_default",
		"desk_mouthpiece",
		"a21_voice_default_dashscope",
		"dashscope_qwen_asr_realtime",
		"stepfun",
		"doubao_realtime",
		"builtin_xiaozhi",
		"custom_multinet",
		"active_builtin_model",
		"builtin_active",
		"本地已存，待索引",
		"not_started_no_execute",
		"binding_not_configured",
		"open_until_binding_configured",
		"indexing_requested_no_execute",
		"deleted_metadata_only",
		"a21.workspace_console_export.v1",
		"voice_probe_trace_id",
		"body_preset_trace_id",
		"body_motion_trace_id",
		"hardware_scene_trace_id",
		"hardware_scene_step_count",
		"connected_device_count",
		"refreshConnectedDevice",
		"preferredConnectedDevice",
		"connection_status",
		"xiaozhi_profile",
		"runBodyPreset",
		"runBodyMotion",
		"runHardwareScene",
		"refreshBodyPresetTrace",
		"refreshHardwareSceneTrace",
		"screen_control_trace_id",
		"runHardwareScreenAction",
		"refreshHardwareScreenTrace",
		"refreshMCPCapabilities",
		"self.screen.get_info",
		"official_action_trace_id",
		"official_action_fallback",
		"official_action_blocked_reason",
		"allow_mcp_fallback",
		"runOfficialAction",
		"runOfficialActionFallback",
		"officialActionFallback",
		"fallback_delivered",
		"refreshOfficialActionTrace",
		"stackchan_official_ws",
		"searchable=false",
		"实体验收=否",
		"V21 执行=否",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("workspace console missing %q", want)
		}
	}
	lowerBody := strings.ToLower(body)
	for _, forbidden := range []string{
		"provider output",
		"voice sample",
		"local path",
		"credential",
		"document text",
		"transcript",
		"audio payload",
		"http://",
		"https://",
	} {
		if strings.Contains(lowerBody, forbidden) {
			t.Fatalf("workspace console leaked forbidden term %q", forbidden)
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
