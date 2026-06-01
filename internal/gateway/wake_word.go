package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	wakeWordConfigSchemaVersion   = "a21.gateway.wake_word.v1"
	wakeWordModeBuiltin           = "builtin_xiaozhi"
	wakeWordModeCustomMultinet    = "custom_multinet"
	wakeWordActivePhrase          = "你好小智"
	wakeWordActivePinyin          = "ni hao xiao zhi"
	wakeWordActiveRuntimeProfile  = "builtin_xiaozhi_wakenet"
	wakeWordFirmwareBuiltinActive = "builtin_active"
	wakeWordFirmwareCustomPending = "custom_pending_firmware"
	defaultWakeWordThreshold      = 30
	wakeWordConfigRequestMaxBytes = 4096
)

var wakeWordPinyinPattern = regexp.MustCompile(`^[a-z ]{3,80}$`)

type WakeWordConfigRequest struct {
	Mode          string `json:"mode"`
	DesiredPhrase string `json:"desired_phrase"`
	DesiredPinyin string `json:"desired_pinyin"`
	Threshold     int    `json:"threshold"`
}

type WakeWordConfigResponse struct {
	SchemaVersion           string `json:"schema_version"`
	Mode                    string `json:"mode"`
	ActivePhrase            string `json:"active_phrase"`
	ActivePinyin            string `json:"active_pinyin"`
	DesiredPhrase           string `json:"desired_phrase,omitempty"`
	DesiredPinyin           string `json:"desired_pinyin,omitempty"`
	Threshold               int    `json:"threshold"`
	RuntimeStatus           string `json:"runtime_status"`
	RuntimeConfigurable     bool   `json:"runtime_configurable"`
	FirmwareBuildRequired   bool   `json:"firmware_build_required"`
	FirmwareStatus          string `json:"firmware_status"`
	ActiveRuntimeProfile    string `json:"active_runtime_profile"`
	DesiredFirmwareProfile  string `json:"desired_firmware_profile,omitempty"`
	RuntimeHotSwapSupported bool   `json:"runtime_hot_swap_supported"`
	CustomRuntimeActive     bool   `json:"custom_runtime_active"`
	Code                    string `json:"code,omitempty"`
	Message                 string `json:"message"`
}

func wakeWordConfigPath(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if envPath := strings.TrimSpace(os.Getenv("A21_WAKE_WORD_CONFIG_PATH")); envPath != "" {
		return envPath
	}
	return filepath.Join(".a21-run", "gateway", "a21-wake-word.json")
}

func (s *Server) handleWakeWordConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config, err := s.loadWakeWordConfig()
		if err != nil {
			http.Error(w, "wake word config unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, wakeWordResponse(config))
	case http.MethodPut:
		var req WakeWordConfigRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, wakeWordConfigRequestMaxBytes))
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		config, err := validateWakeWordConfigRequest(req)
		if err != nil {
			http.Error(w, "invalid wake word config", http.StatusBadRequest)
			return
		}
		if err := s.storeWakeWordConfig(config); err != nil {
			http.Error(w, "wake word config unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, wakeWordResponse(config))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) loadWakeWordConfig() (WakeWordConfigRequest, error) {
	return loadWakeWordConfigFromPath(s.wakeWordConfigPath)
}

func LoadWakeWordConfigStatus(configuredPath string) (WakeWordConfigResponse, error) {
	config, err := loadWakeWordConfigFromPath(wakeWordConfigPath(configuredPath))
	if err != nil {
		return WakeWordConfigResponse{}, err
	}
	return wakeWordResponse(config), nil
}

func loadWakeWordConfigFromPath(configuredPath string) (WakeWordConfigRequest, error) {
	path := strings.TrimSpace(configuredPath)
	if path == "" {
		return defaultWakeWordConfig(), nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultWakeWordConfig(), nil
	}
	if err != nil {
		return WakeWordConfigRequest{}, err
	}
	var config WakeWordConfigRequest
	if err := json.Unmarshal(data, &config); err != nil {
		return WakeWordConfigRequest{}, err
	}
	return validateWakeWordConfigRequest(config)
}

func (s *Server) storeWakeWordConfig(config WakeWordConfigRequest) error {
	path := strings.TrimSpace(s.wakeWordConfigPath)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func defaultWakeWordConfig() WakeWordConfigRequest {
	return WakeWordConfigRequest{
		Mode:      wakeWordModeBuiltin,
		Threshold: defaultWakeWordThreshold,
	}
}

func validateWakeWordConfigRequest(req WakeWordConfigRequest) (WakeWordConfigRequest, error) {
	req.Mode = strings.TrimSpace(req.Mode)
	req.DesiredPhrase = strings.TrimSpace(req.DesiredPhrase)
	req.DesiredPinyin = strings.ToLower(strings.Join(strings.Fields(req.DesiredPinyin), " "))
	if req.Threshold == 0 {
		req.Threshold = defaultWakeWordThreshold
	}
	switch req.Mode {
	case "", wakeWordModeBuiltin:
		return WakeWordConfigRequest{
			Mode:      wakeWordModeBuiltin,
			Threshold: req.Threshold,
		}, validateWakeWordThreshold(req.Threshold)
	case wakeWordModeCustomMultinet:
		if err := validateWakeWordThreshold(req.Threshold); err != nil {
			return WakeWordConfigRequest{}, err
		}
		if !safeWakeWordPhrase(req.DesiredPhrase) || !safeWakeWordPinyin(req.DesiredPinyin) {
			return WakeWordConfigRequest{}, errors.New("unsafe wake word")
		}
		return req, nil
	default:
		return WakeWordConfigRequest{}, errors.New("unsupported wake word mode")
	}
}

func validateWakeWordThreshold(threshold int) error {
	if threshold < 1 || threshold > 100 {
		return errors.New("wake word threshold out of range")
	}
	return nil
}

func safeWakeWordPhrase(value string) bool {
	if value == "" || len([]rune(value)) > 12 {
		return false
	}
	lower := strings.ToLower(value)
	for _, blocked := range []string{"http://", "https://", "/users/", "token", "secret", "bearer", "sk-", "x21", "v21"} {
		if strings.Contains(lower, blocked) {
			return false
		}
	}
	return !strings.ContainsAny(value, "\r\n\t")
}

func safeWakeWordPinyin(value string) bool {
	if value == "" || !wakeWordPinyinPattern.MatchString(value) {
		return false
	}
	lower := strings.ToLower(value)
	return !strings.Contains(lower, "http") &&
		!strings.Contains(lower, "token") &&
		!strings.Contains(lower, "secret")
}

func wakeWordResponse(config WakeWordConfigRequest) WakeWordConfigResponse {
	response := WakeWordConfigResponse{
		SchemaVersion:        wakeWordConfigSchemaVersion,
		Mode:                 wakeWordModeBuiltin,
		ActivePhrase:         wakeWordActivePhrase,
		ActivePinyin:         wakeWordActivePinyin,
		Threshold:            config.Threshold,
		RuntimeStatus:        "active_builtin_model",
		RuntimeConfigurable:  false,
		FirmwareStatus:       wakeWordFirmwareBuiltinActive,
		ActiveRuntimeProfile: wakeWordActiveRuntimeProfile,
		Message:              "Stock xiaozhi firmware keeps the built-in WakeNet model active until an explicit firmware build is flashed.",
	}
	if config.Mode == wakeWordModeCustomMultinet {
		response.Mode = wakeWordModeCustomMultinet
		response.DesiredPhrase = config.DesiredPhrase
		response.DesiredPinyin = config.DesiredPinyin
		response.DesiredFirmwareProfile = wakeWordModeCustomMultinet
		response.RuntimeStatus = "pending_firmware_build"
		response.FirmwareBuildRequired = true
		response.FirmwareStatus = wakeWordFirmwareCustomPending
		response.Code = "a21_wake_word_firmware_build_required"
		response.Message = "Custom wake words require a dedicated xiaozhi/ESP-SR MultiNet firmware build; Gateway persists the requested profile but cannot hot-swap the stock wake model."
	}
	return response
}
