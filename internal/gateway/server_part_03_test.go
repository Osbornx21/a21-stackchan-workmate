package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/protocol"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestPowerLifecycleReportsAndAcceptsOnlyForegroundColdBootEvidence(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{XiaozhiProductPlaybackEvents: true})
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
		"trace_id":   "a21-trace-power-hello",
		"session_id": "a21-session-power-hello",
		"features": map[string]any{
			"mcp":              true,
			"playback_events":  true,
			"keepalive_events": true,
		},
	})
	readXiaozhiJSON(t, ctx, conn)
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":                   "device",
		"kind":                   "heartbeat",
		"trace_id":               "a21-trace-power-hello",
		"session_id":             "a21-session-power-hello",
		"device_id":              "44:1b:f6:e2:6a:60",
		"battery_level":          68,
		"battery_charging":       false,
		"battery_discharging":    true,
		"external_power":         false,
		"power_source":           "battery_discharging",
		"pmic_power_key_profile": "a21_stackchan_axp2101_pwrkey_v1",
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(25 * time.Millisecond)

	resp, err := http.Get(httpServer.URL + "/v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("power lifecycle status = %d: %s", resp.StatusCode, body)
	}
	var pending map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":    "a21.gateway.power_lifecycle.v1",
		"overall_status":    "physical_pending",
		"physical_accepted": false,
		"xiaozhi_ws_online": true,
		"result_redacted":   true,
		"connection_status": "online",
		"battery_telemetry": "diagnostic_runtime_echo",
	} {
		if pending[key] != want {
			t.Fatalf("pending[%s] = %#v, want %#v in %#v", key, pending[key], want, pending)
		}
	}

	acceptResp, err := http.Post(
		httpServer.URL+"/v1/power-lifecycle-acceptance",
		"application/json",
		bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-power-acceptance","session_id":"a21-session-power-acceptance","cold_boot_without_usb":true,"power_button_started":true,"gateway_connected":true,"xiaozhi_socket_online":true,"standalone_runtime_ok":true,"boot_source":"battery_power_key_cold_boot","usb_connected_during_boot":false,"power_key_hold_ms":4200,"pmic_power_key_profile":"a21_stackchan_axp2101_pwrkey_v1","boot_observed_at_ms":1780617600000,"observer":"operator"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(acceptResp.Body)
		t.Fatalf("power lifecycle acceptance status = %d: %s", acceptResp.StatusCode, body)
	}
	var accepted map[string]any
	if err := json.NewDecoder(acceptResp.Body).Decode(&accepted); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"schema_version":         "a21.gateway.power_lifecycle_acceptance.v1",
		"status":                 "accepted",
		"physical_accepted":      true,
		"result_redacted":        true,
		"boot_source":            "battery_power_key_cold_boot",
		"pmic_power_key_profile": "a21_stackchan_axp2101_pwrkey_v1",
	} {
		if accepted[key] != want {
			t.Fatalf("accepted[%s] = %#v, want %#v in %#v", key, accepted[key], want, accepted)
		}
	}

	traceResp, err := http.Get(httpServer.URL + "/v1/traces?trace_id=a21-trace-power-acceptance")
	if err != nil {
		t.Fatal(err)
	}
	defer traceResp.Body.Close()
	var traces TraceResponse
	if err := json.NewDecoder(traceResp.Body).Decode(&traces); err != nil {
		t.Fatal(err)
	}
	if !traceContains(traces.Events, "power_lifecycle.physical_acceptance.accepted") {
		t.Fatalf("trace missing power acceptance marker: %+v", traces.Events)
	}
	registry := fetchSingleDeviceRegistryItem(t, httpServer.URL)
	capabilities := registry["capabilities"].(map[string]any)
	for key, want := range map[string]any{
		"power_lifecycle_physical_accepted":              "true",
		"no_cable_cold_boot_physical_accepted":           "true",
		"physical_power_button_start_physical_accepted":  "true",
		"last_power_lifecycle_acceptance_status":         "operator_power_button_accepted",
		"last_power_lifecycle_boot_source":               "battery_power_key_cold_boot",
		"last_power_lifecycle_usb_connected_during_boot": "false",
		"last_power_lifecycle_power_key_hold_ms":         "4200",
		"last_power_lifecycle_pmic_profile":              "a21_stackchan_axp2101_pwrkey_v1",
		"stackchan_pmic_power_key_profile":               "a21_stackchan_axp2101_pwrkey_v1",
	} {
		if capabilities[key] != want {
			t.Fatalf("capabilities[%s] = %#v, want %#v in %#v", key, capabilities[key], want, capabilities)
		}
	}
}

func TestPowerLifecycleAcceptanceRequiresOnlineXiaozhiSocket(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/power-lifecycle-acceptance", bytes.NewBufferString(`{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-power-missing","session_id":"a21-session-power-missing","cold_boot_without_usb":true,"power_button_started":true,"gateway_connected":true,"xiaozhi_socket_online":true,"standalone_runtime_ok":true,"boot_source":"battery_power_key_cold_boot","usb_connected_during_boot":false,"power_key_hold_ms":4200,"pmic_power_key_profile":"a21_stackchan_axp2101_pwrkey_v1","boot_observed_at_ms":1780617600000,"observer":"operator"}`))
	rec := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("power lifecycle acceptance status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestPowerLifecycleAcceptanceRequiresColdBootPMICEvidence(t *testing.T) {
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
		"trace_id":   "a21-trace-power-contract-hello",
		"session_id": "a21-session-power-contract-hello",
	})
	readXiaozhiJSON(t, ctx, conn)

	cases := []struct {
		name string
		body string
	}{
		{
			name: "missing boot source",
			body: `{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-power-contract","session_id":"a21-session-power-contract","cold_boot_without_usb":true,"power_button_started":true,"gateway_connected":true,"xiaozhi_socket_online":true,"standalone_runtime_ok":true,"usb_connected_during_boot":false,"power_key_hold_ms":4200,"pmic_power_key_profile":"a21_stackchan_axp2101_pwrkey_v1","boot_observed_at_ms":1780617600000,"observer":"operator"}`,
		},
		{
			name: "usb still connected",
			body: `{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-power-contract","session_id":"a21-session-power-contract","cold_boot_without_usb":true,"power_button_started":true,"gateway_connected":true,"xiaozhi_socket_online":true,"standalone_runtime_ok":true,"boot_source":"battery_power_key_cold_boot","usb_connected_during_boot":true,"power_key_hold_ms":4200,"pmic_power_key_profile":"a21_stackchan_axp2101_pwrkey_v1","boot_observed_at_ms":1780617600000,"observer":"operator"}`,
		},
		{
			name: "wrong PMIC profile",
			body: `{"device_id":"44:1b:f6:e2:6a:60","trace_id":"a21-trace-power-contract","session_id":"a21-session-power-contract","cold_boot_without_usb":true,"power_button_started":true,"gateway_connected":true,"xiaozhi_socket_online":true,"standalone_runtime_ok":true,"boot_source":"battery_power_key_cold_boot","usb_connected_during_boot":false,"power_key_hold_ms":4200,"pmic_power_key_profile":"old_profile","boot_observed_at_ms":1780617600000,"observer":"operator"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(httpServer.URL+"/v1/power-lifecycle-acceptance", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status = %d, want 400: %s", resp.StatusCode, body)
			}
		})
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
	if device.CurrentMode != protocol.ModeWorkmate ||
		device.CurrentVoiceMode != "roleplay" ||
		device.CurrentRoleplayProfile != "a21_roleplay_default" ||
		device.CurrentRoleplayScenario != "desk_mouthpiece" ||
		!device.RoleplaySoulReady ||
		device.RoleplayMemoryReady ||
		device.RoleplayMemoryHintCount != 0 ||
		device.RoleplayPhysicalAccepted {
		t.Fatalf("device state = %+v, want legacy transport mode workmate and voice_mode roleplay", device)
	}
	if strings.Contains(devicesRec.Body.String(), "selected voice mode must not rewrite product mode") {
		t.Fatalf("registry leaked control text: %s", devicesRec.Body.String())
	}
}

func TestRoleplayProfileReflectsSafeDeviceRegistryState(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	profileReq := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{
		"roleplay_profile":"a21_roleplay_wry_peer",
		"scenario":"engineer_pushback",
		"voice_clone_profile":"a21_voice_clone_default",
		"memory_hints":["角色语气只给短句","http://secret.example/leak"]
	}`))
	profileRec := httptest.NewRecorder()
	handler.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK {
		t.Fatalf("roleplay profile status = %d, want 200: %s", profileRec.Code, profileRec.Body.String())
	}

	server.controlSequence("stackchan-sim-001", "a21-trace-roleplay-registry", "a21-session-roleplay-registry", []protocol.ControlEventPayload{{
		State: protocol.ExpressionThinking,
		Mode:  protocol.ModeRoleplay,
		Text:  "roleplay registry must not store this text",
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
	if device.CurrentRoleplayProfile != "a21_roleplay_wry_peer" ||
		device.CurrentRoleplayScenario != "engineer_pushback" ||
		device.CurrentVoiceCloneProfile != "a21_voice_clone_default" ||
		!device.RoleplaySoulReady ||
		!device.RoleplayMemoryReady ||
		device.RoleplayMemoryHintCount != 1 ||
		device.RoleplayPhysicalAccepted {
		t.Fatalf("roleplay device state = %+v", device)
	}
	if device.RuntimeEcho["roleplay_profile"] != "a21_roleplay_wry_peer" ||
		device.RuntimeEcho["roleplay_scenario"] != "engineer_pushback" ||
		device.RuntimeEcho["roleplay_memory_ready"] != "true" ||
		device.RuntimeEcho["roleplay_memory_hint_count"] != "1" ||
		device.RuntimeEcho["roleplay_physical_accepted"] != "false" {
		t.Fatalf("runtime echo = %+v, want safe roleplay state", device.RuntimeEcho)
	}
	for _, forbidden := range []string{
		"角色语气只给短句",
		"secret.example",
		"roleplay registry must not store this text",
		"http://",
		"https://",
	} {
		if strings.Contains(devicesRec.Body.String(), forbidden) {
			t.Fatalf("device registry leaked %q: %s", forbidden, devicesRec.Body.String())
		}
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

func TestRoleplayProfileEndpointSelectsSoulProfileAndReturnsSafeCatalog(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{
		"roleplay_profile":"a21_roleplay_wry_peer",
		"scenario":"engineer_pushback",
		"voice_clone_profile":"a21_voice_clone_default"
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RoleplayProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SelectedRoleplayProfile != "a21_roleplay_wry_peer" ||
		response.SelectedScenario != "engineer_pushback" ||
		response.Runtime.RoleplayProfile != "a21_roleplay_wry_peer" ||
		!response.Runtime.SoulPromptInputReady ||
		!response.Runtime.PromptComposed {
		t.Fatalf("roleplay soul response = %+v", response)
	}
	if len(response.Profiles) < 3 {
		t.Fatalf("profiles = %+v, want multiple selectable role souls", response.Profiles)
	}
	if !stringSliceContains(response.Runtime.PromptParts, "role_soul:a21_roleplay_wry_peer") ||
		!stringSliceContains(response.Runtime.PromptParts, "scenario:engineer_pushback") {
		t.Fatalf("prompt parts = %+v, want selected soul and scenario", response.Runtime.PromptParts)
	}
	for _, forbidden := range []string{
		"Role Soul: Wry Peer",
		"这句会上说会炸",
		"Roleplay Mode",
		"Engineer Pushback Playbook",
		"provider output",
		"voice clone sample",
		"/Users/",
		"http://",
		"https://",
	} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("roleplay profile leaked prompt/private text %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestRoleplayProfileEndpointReturnsOfficialExpressionPlanWithoutSendingHardware(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/roleplay-profile", bytes.NewBufferString(`{
		"roleplay_profile":"a21_roleplay_wry_peer",
		"scenario":"engineer_pushback",
		"voice_clone_profile":"a21_voice_clone_default",
		"memory_hints":["角色语气只给短句","https://secret.example/leak"]
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response RoleplayProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	plan := response.ExpressionPlan
	if plan.SchemaVersion != "a21.roleplay_expression_plan.v1" ||
		plan.Adapter != "official_stackchan_action_plan" ||
		plan.DeliveryPolicy != "no_send_plan_only" ||
		plan.RoleplayProfile != "a21_roleplay_wry_peer" ||
		plan.Scenario != "engineer_pushback" ||
		plan.VoiceCloneProfile != "a21_voice_clone_default" ||
		!plan.MemoryReady ||
		plan.MemoryHintCount != 1 ||
		plan.PhysicalAccepted ||
		plan.ActionCount != 4 ||
		plan.PacketCount != 6 {
		t.Fatalf("expression plan = %+v", plan)
	}
	if plan.Redaction.PromptTextStored ||
		plan.Redaction.MemoryTextStored ||
		plan.Redaction.TranscriptStored ||
		plan.Redaction.ProviderOutputStored ||
		plan.Redaction.AudioStored ||
		plan.Redaction.VoiceCloneSampleStored {
		t.Fatalf("expression redaction = %+v, want no stored private payloads", plan.Redaction)
	}
	if !roleplayExpressionPhase(plan.Actions, "baseline_posture", "state", "listening") ||
		!roleplayExpressionPhase(plan.Actions, "role_soul", "face", "happy") ||
		!roleplayExpressionPhase(plan.Actions, "scenario_emphasis", "state", "thinking") ||
		!roleplayExpressionPhase(plan.Actions, "memory_cue", "motion", "nod") {
		t.Fatalf("expression actions = %+v", plan.Actions)
	}
	if plan.Surfaces["role_soul_avatar"] != "happy" ||
		plan.Surfaces["scenario_emphasis_avatar"] != "thinking" ||
		plan.Surfaces["memory_cue_motion"] != "official_pitch_sequence" ||
		plan.Surfaces["baseline_posture_packet_contract"] != "official_stackchan_binary" {
		t.Fatalf("expression surfaces = %+v", plan.Surfaces)
	}
	for _, action := range plan.Actions {
		if action.PhysicalAccepted {
			t.Fatalf("action = %+v, must not claim physical acceptance", action)
		}
		if action.PacketCount <= 0 {
			t.Fatalf("action = %+v, want official packet metadata", action)
		}
	}
	for _, forbidden := range []string{
		"角色语气只给短句",
		"secret.example",
		"Role Soul: Wry Peer",
		"provider output",
		"voice clone sample",
		"http://",
		"https://",
	} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("roleplay expression plan leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func roleplayExpressionPhase(actions []RoleplayExpressionAction, phase string, event string, value string) bool {
	for _, action := range actions {
		if action.Phase == phase && action.Event == event && action.Value == value {
			return true
		}
	}
	return false
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

func TestWorkspaceDocumentUploadIntakeStoresBytesAndRedactsResponse(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "a21-workspace-documents")
	server := NewServerWithOptions(ServerOptions{
		WorkspaceDocumentStoreDir: storeDir,
		WorkspaceDocumentMaxBytes: 1 << 20,
	})
	handler := server.Handler()
	privateContent := "RAW_PRIVATE_DOCUMENT_CONTENT_FOR_A21_UPLOAD_TEST"
	body, contentType := multipartWorkspaceDocumentBody(t, map[string]string{
		"user_id":        "a21_user_upload",
		"workspace_id":   "a21_workspace_upload",
		"source_scope":   "personal",
		"document_label": "PRD pack",
		"content_type":   "text/plain",
		"trace_id":       "a21-trace-document-upload",
		"session_id":     "a21-session-document-upload",
		"device_id":      "stackchan-sim-001",
	}, "SECRET-roadmap.txt", privateContent)
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response WorkspaceDocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != "a21.gateway.workspace_documents.v1" || len(response.Documents) != 1 {
		t.Fatalf("document response = %+v", response)
	}
	document := response.Documents[0]
	if document.DocumentID == "" ||
		document.JobID == "" ||
		document.SourceID == "" ||
		document.UserID != "a21_user_upload" ||
		document.WorkspaceID != "a21_workspace_upload" ||
		document.SourceScope != "personal" ||
		document.DocumentLabel != "PRD pack" ||
		document.ContentType != "text/plain" ||
		document.SizeBytes != int64(len(privateContent)) ||
		!strings.HasPrefix(document.DocumentHash, "sha256:") ||
		document.Status != "stored_local_pending_index" ||
		document.StorageStatus != "stored_local" ||
		document.IndexStatus != "not_started_no_execute" ||
		document.Readiness != "stored_local_pending_index" {
		t.Fatalf("document = %+v", document)
	}
	if document.Redaction.DocumentTextStored ||
		!document.Redaction.DocumentBytesStored ||
		document.Redaction.DocumentBytesReturned ||
		document.Redaction.OriginalFilenameStored ||
		document.Redaction.LocalPathStored ||
		document.Redaction.LocalPathReturned ||
		document.Redaction.CredentialValueStored ||
		document.Redaction.ProviderOutputStored {
		t.Fatalf("document redaction = %+v", document.Redaction)
	}
	for _, forbidden := range []string{privateContent, "SECRET-roadmap", storeDir, "/Users/", "api_key", "bearer "} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("document upload leaked %q: %s", forbidden, rec.Body.String())
		}
	}
	if stored := readAllFilesUnder(t, storeDir); !strings.Contains(stored, privateContent) {
		t.Fatalf("stored files did not contain uploaded bytes, got %q", stored)
	}

	var jobs WorkspaceUploadJobsResponse
	jobReq := httptest.NewRequest(http.MethodGet, "/v1/workspace-upload-jobs?job_id="+url.QueryEscape(document.JobID), nil)
	jobRec := httptest.NewRecorder()
	handler.ServeHTTP(jobRec, jobReq)
	if jobRec.Code != http.StatusOK {
		t.Fatalf("job status = %d, want 200: %s", jobRec.Code, jobRec.Body.String())
	}
	if err := json.Unmarshal(jobRec.Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs.Jobs) != 1 ||
		jobs.Jobs[0].Status != "stored_local_pending_index" ||
		jobs.Jobs[0].IndexStatus != "not_started_no_execute" ||
		jobs.Jobs[0].DocumentID != document.DocumentID ||
		jobs.Jobs[0].DocumentHash != document.DocumentHash ||
		jobs.Jobs[0].StorageStatus != "stored_local" ||
		!jobs.Jobs[0].Redaction.DocumentBytesStored {
		t.Fatalf("linked job = %+v", jobs)
	}

	source := fetchSingleWorkspaceSource(t, handler, document.SourceID)
	if source.Readiness != "stored_local_pending_index" ||
		source.IndexStatus != "not_started_no_execute" ||
		source.DocumentID != document.DocumentID ||
		source.DocumentHash != document.DocumentHash ||
		source.StorageStatus != "stored_local" ||
		!source.StoredLocal ||
		source.MetadataOnly ||
		source.Searchable {
		t.Fatalf("linked source = %+v", source)
	}

	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_upload","workspace_id":"a21_workspace_upload","query_scope":"personal_only"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d, want 200: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	var workspace ProfessionalWorkspaceResponse
	if err := json.Unmarshal(workspaceRec.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.WorkspaceStatus != "stored_local_pending_index" ||
		workspace.Runtime.SourceScopeCounts["personal"] != 1 ||
		workspace.Runtime.SearchableSourceScopeCounts["personal"] != 0 ||
		workspace.Runtime.QueryScopeReadiness != "stored_local_pending_index" ||
		workspace.Runtime.V21ExecutionAllowed {
		t.Fatalf("workspace = %+v runtime=%+v", workspace, workspace.Runtime)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-document-upload", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"workspace.document.stored_local", "workspace.source.stored_local_pending_index", "workspace.index_job.not_started_no_execute"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{privateContent, "SECRET-roadmap", storeDir, "/Users/", "api_key", "bearer "} {
		if strings.Contains(jobRec.Body.String()+fetchWorkspaceSourcesBody(t, handler, document.SourceID)+workspaceRec.Body.String()+traceRec.Body.String(), forbidden) {
			t.Fatalf("linked workspace surfaces leaked %q", forbidden)
		}
	}
}
