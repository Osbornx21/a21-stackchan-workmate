package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"a21.local/a21/internal/v21adapter"
	"github.com/coder/websocket"
)

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

func TestVoiceModeRitualProfessionalSendsHardwareSequence(t *testing.T) {
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
		"trace_id":   "a21-trace-mode-ritual-hello",
		"session_id": "a21-session-mode-ritual-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	resp, err := http.Post(
		httpServer.URL+"/v1/voice-mode-ritual",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","voice_mode":"professional","trace_id":"a21-trace-mode-ritual-professional","session_id":"a21-session-mode-ritual-professional"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("voice mode ritual status = %d: %s", resp.StatusCode, string(body))
	}
	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":         "a21.gateway.voice_mode_ritual.v1",
		"selected_voice_mode":    "professional",
		"status":                 "delivered",
		"delivered_transport":    "xiaozhi_mcp_sequence",
		"screen_label":           "PRO",
		"physical_accepted":      false,
		"provider_executed":      false,
		"v21_executed":           false,
		"result_redacted":        true,
		"workspace_policy":       "professional_only",
		"mode_switch_visible":    true,
		"official_relay_claimed": false,
	} {
		if response[key] != want {
			t.Fatalf("response[%s] = %#v, want %#v in %#v", key, response[key], want, response)
		}
	}
	if response["step_delay_ms"] != float64(20) {
		t.Fatalf("step_delay_ms = %#v, want 20 in %#v", response["step_delay_ms"], response)
	}
	if response["total_planned_delay_ms"] != float64(60) {
		t.Fatalf("total_planned_delay_ms = %#v, want 60 in %#v", response["total_planned_delay_ms"], response)
	}
	steps, ok := response["steps"].([]any)
	if !ok || len(steps) != 4 {
		t.Fatalf("steps = %#v, want 4 redacted hardware steps", response["steps"])
	}

	expected := []struct {
		tool string
		args map[string]any
	}{
		{tool: xiaozhiMCPScreenSetThemeToolName, args: map[string]any{"theme": "dark"}},
		{tool: xiaozhiMCPScreenSetBrightnessToolName, args: map[string]any{"brightness": float64(78)}},
		{tool: xiaozhiMCPRobotSetLEDColorToolName, args: map[string]any{"red": float64(0), "green": float64(84), "blue": float64(168)}},
		{tool: xiaozhiMCPRobotSetHeadAnglesToolName, args: map[string]any{"yaw": float64(0), "pitch": float64(32), "speed": float64(220)}},
	}
	for _, want := range expected {
		message := readXiaozhiJSON(t, ctx, conn)
		assertXiaozhiMCPMessage(t, message, want.tool, want.args)
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-mode-ritual-professional")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"voice_mode.ritual.professional.step1.screen_theme.sent",
		"voice_mode.ritual.professional.step4.robot_head_angles_set.sent",
		"voice_mode.ritual.professional.completed",
	} {
		if !traceContains(traces.Events, want) {
			t.Fatalf("trace missing %q: %+v", want, traces.Events)
		}
	}
	if traces.Summary.LastOffsetMS < 55 {
		t.Fatalf("trace last offset = %dms, want visible ritual pacing: %+v", traces.Summary.LastOffsetMS, traces.Events)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	if registry["current_voice_mode"] != "professional" {
		t.Fatalf("current_voice_mode = %#v, want professional", registry["current_voice_mode"])
	}
	capabilities := registry["capabilities"].(map[string]any)
	for key, want := range map[string]any{
		"last_voice_mode_ritual":        "professional",
		"last_voice_mode_ritual_status": "delivered",
		"last_voice_mode_ritual_step":   "4",
		"screen_theme":                  "dark",
		"screen_brightness":             "78",
		"robot_head_pitch":              "32",
		"robot_led_blue":                "168",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestVoiceModeRitualRejectsUnknownModeAndDoesNotFallbackToProvider(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/voice-mode-ritual", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","voice_mode":"camera"}`))
	rec := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("voice mode ritual status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestVoiceModeRitualPhysicalAcceptanceRecordsOperatorEvidence(t *testing.T) {
	server := NewServer()
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
		"device_id":  "44:1b:f6:e2:6a:60",
		"trace_id":   "a21-trace-mode-ritual-acceptance-hello",
		"session_id": "a21-session-mode-ritual-acceptance-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	ritualResp, err := http.Post(
		httpServer.URL+"/v1/voice-mode-ritual",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","voice_mode":"professional","trace_id":"a21-trace-mode-ritual-acceptance","session_id":"a21-session-mode-ritual-acceptance"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer ritualResp.Body.Close()
	if ritualResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(ritualResp.Body)
		t.Fatalf("mode ritual status = %d: %s", ritualResp.StatusCode, string(body))
	}
	for i := 0; i < 4; i++ {
		readXiaozhiJSON(t, ctx, conn)
	}

	acceptResp, err := http.Post(
		httpServer.URL+"/v1/voice-mode-ritual-acceptance",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","voice_mode":"professional","trace_id":"a21-trace-mode-ritual-acceptance","session_id":"a21-session-mode-ritual-acceptance","screen_visible":true,"rgb_visible":true,"servo_visible":true,"observer":"operator"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(acceptResp.Body)
		t.Fatalf("mode ritual acceptance status = %d: %s", acceptResp.StatusCode, string(body))
	}
	var response map[string]any
	if err := json.NewDecoder(acceptResp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":      "a21.gateway.voice_mode_ritual_acceptance.v1",
		"status":              "accepted",
		"selected_voice_mode": "professional",
		"physical_accepted":   true,
		"result_redacted":     true,
	} {
		if response[key] != want {
			t.Fatalf("response[%s] = %#v, want %#v in %#v", key, response[key], want, response)
		}
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-mode-ritual-acceptance")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "voice_mode.ritual.professional.physical_acceptance.accepted") {
		t.Fatalf("trace missing mode ritual physical acceptance marker: %+v", traces.Events)
	}

	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	for key, want := range map[string]any{
		"voice_mode_ritual_physical_accepted":        "true",
		"voice_mode_ritual_screen_physical_accepted": "true",
		"voice_mode_ritual_rgb_physical_accepted":    "true",
		"voice_mode_ritual_servo_physical_accepted":  "true",
		"last_voice_mode_ritual_acceptance_status":   "operator_visible_accepted",
		"last_voice_mode_ritual_acceptance_mode":     "professional",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestVoiceModeRitualPhysicalAcceptanceRequiresMatchingEvidence(t *testing.T) {
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/voice-mode-ritual-acceptance", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","voice_mode":"professional","trace_id":"a21-trace-missing-mode-ritual","session_id":"a21-session-missing-mode-ritual","screen_visible":true,"rgb_visible":true,"servo_visible":true,"observer":"operator"}`))
	NewServer().Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("mode ritual acceptance status = %d, want 409: %s", resp.Code, resp.Body.String())
	}
}

func TestHardwareAcceptanceSummaryReportsMachineEvidenceAndPhysicalPending(t *testing.T) {
	server := NewServer()
	deviceID := "44:1b:f6:e2:6a:60"
	server.recordVoiceModeRitualCompleted(deviceID, VoiceModeRoleplay, "a21-trace-mode-summary", "a21-session-mode-summary", server.now().UnixMilli())
	server.mu.Lock()
	record := server.devices[deviceID]
	record.Capabilities = mergeDeviceCapabilities(record.Capabilities, map[string]string{
		"last_body_scene":                 "full_check",
		"last_body_scene_status":          "delivered",
		"last_body_scene_trace_id":        "a21-trace-full-check-summary",
		"last_body_scene_session_id":      "a21-session-full-check-summary",
		"body_scene_physical_accepted":    "false",
		"xiaozhi_product_touch_reactions": "true",
	})
	server.devices[deviceID] = record
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/hardware-acceptance?device_id="+url.QueryEscape(deviceID), nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("hardware acceptance status = %d: %s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":    "a21.gateway.hardware_acceptance.v1",
		"device_id":         deviceID,
		"overall_status":    "physical_pending",
		"physical_accepted": false,
	} {
		if response[key] != want {
			t.Fatalf("response[%s] = %#v, want %#v in %#v", key, response[key], want, response)
		}
	}
	items, ok := response["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("items = %#v, want three acceptance items", response["items"])
	}
	byID := map[string]map[string]any{}
	for _, item := range items {
		value, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item = %#v, want object", item)
		}
		byID[value["id"].(string)] = value
	}
	mode := byID["mode_ritual"]
	if mode["delivery_status"] != "delivered" || mode["physical_accepted"] != false || mode["next_action"] != "accept_visible_mode_ritual" {
		t.Fatalf("mode item = %#v, want delivered physical pending", mode)
	}
	body := byID["full_check"]
	if body["delivery_status"] != "delivered" || body["physical_accepted"] != false || body["next_action"] != "accept_visible_full_check" {
		t.Fatalf("body item = %#v, want delivered physical pending", body)
	}
	power := byID["power_lifecycle"]
	if power["delivery_status"] != "delivered" || power["physical_accepted"] != false || power["next_action"] != "accept_no_cable_power_button_boot" {
		t.Fatalf("power item = %#v, want online but physical pending", power)
	}
}
