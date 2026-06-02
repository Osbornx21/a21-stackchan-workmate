package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestWakeWordBuiltinResponseDeclaresActiveBuiltinAndNoHotSwap(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: filepath.Join(t.TempDir(), "a21-wake-word.json")})
	req := httptest.NewRequest(http.MethodGet, "/v1/wake-word", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, rec.Body.String())
	}
	for key, want := range map[string]any{
		"firmware_status":            "builtin_active",
		"active_runtime_profile":     "builtin_xiaozhi_wakenet",
		"runtime_hot_swap_supported": false,
		"custom_runtime_active":      false,
	} {
		if got := response[key]; got != want {
			t.Fatalf("%s = %#v, want %#v in %s", key, got, want, rec.Body.String())
		}
	}
	if _, ok := response["desired_firmware_profile"]; ok {
		t.Fatalf("builtin response should not declare desired firmware profile: %s", rec.Body.String())
	}
}

func TestWakeWordCustomResponseKeepsBuiltinActiveAndCustomPendingFirmware(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: filepath.Join(t.TempDir(), "a21-wake-word.json")})
	req := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, rec.Body.String())
	}
	for key, want := range map[string]any{
		"firmware_status":            "custom_pending_firmware",
		"active_runtime_profile":     "builtin_xiaozhi_wakenet",
		"desired_firmware_profile":   "custom_multinet",
		"runtime_hot_swap_supported": false,
		"custom_runtime_active":      false,
		"firmware_build_required":    true,
	} {
		if got := response[key]; got != want {
			t.Fatalf("%s = %#v, want %#v in %s", key, got, want, rec.Body.String())
		}
	}
	if got := response["active_phrase"]; got != wakeWordActivePhrase {
		t.Fatalf("active_phrase = %#v, want builtin phrase %q", got, wakeWordActivePhrase)
	}
}

func TestSimulatorWakeWordPanelShowsRuntimeAndFirmwareSeparation(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/simulator", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="wakeWordFirmwareStatus"`,
		`id="wakeWordHotSwap"`,
		`firmware_status`,
		`runtime_hot_swap_supported`,
		`custom_runtime_active`,
		"builtin_active",
		"custom_pending_firmware",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("simulator page missing %q", want)
		}
	}
}

func TestSimulatorWakeWordPanelShowsResetAndExportAffordances(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/simulator", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="resetWakeWord"`,
		`id="exportWakeWord"`,
		"resetWakeWordConfig",
		"exportWakeWordConfig",
		`mode: 'builtin_xiaozhi'`,
		"a21-wake-word-config.json",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("simulator page missing %q", want)
		}
	}
}

func TestWakeWordBuiltinResetClearsCustomIntentWithoutRuntimeClaim(t *testing.T) {
	server := NewServerWithOptions(ServerOptions{WakeWordConfigPath: filepath.Join(t.TempDir(), "a21-wake-word.json")})
	customReq := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(`{
		"mode":"custom_multinet",
		"desired_phrase":"小阿二一",
		"desired_pinyin":"xiao a er yi",
		"threshold":35
	}`))
	customRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(customRec, customReq)
	if customRec.Code != http.StatusOK {
		t.Fatalf("custom status = %d, want 200: %s", customRec.Code, customRec.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPut, "/v1/wake-word", strings.NewReader(`{
		"mode":"builtin_xiaozhi",
		"threshold":30
	}`))
	resetRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(resetRec, resetReq)

	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want 200: %s", resetRec.Code, resetRec.Body.String())
	}
	body := resetRec.Body.String()
	for _, want := range []string{
		`"mode":"builtin_xiaozhi"`,
		`"active_phrase":"你好小智"`,
		`"runtime_status":"active_builtin_model"`,
		`"firmware_build_required":false`,
		`"firmware_status":"builtin_active"`,
		`"custom_runtime_active":false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("reset response missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"desired_phrase", "custom_pending_firmware", "pending_firmware_build", "product_ready"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("reset response overclaimed or kept custom field %q: %s", forbidden, body)
		}
	}
}
