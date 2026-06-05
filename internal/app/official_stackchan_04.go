package app

import (
	"context"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func collectOfficialFlashPartsForApp(buildDir string, expectedAppName string) ([]stackChanOfficialSmokeFlashPart, error) {
	entries := readOfficialFlashArgsEntries(buildDir)
	if len(entries) == 0 {
		return nil, fmt.Errorf("official smoke flash_args missing or empty")
	}
	required := map[string]string{
		"0x0":      "bootloader",
		"0x8000":   "partition_table",
		"0xd000":   "ota_data_initial",
		"0x20000":  "app",
		"0xa00000": "assets",
	}
	seenOffsets := make(map[string]bool)
	parts := make([]stackChanOfficialSmokeFlashPart, 0, len(entries))
	for _, entry := range entries {
		name, ok := required[entry.offset]
		if !ok {
			continue
		}
		fullPath := filepath.Join(buildDir, filepath.FromSlash(entry.path))
		if containsLegacyIdentityPathToken(fullPath) {
			return nil, fmt.Errorf("flash part path contains forbidden legacy identity")
		}
		if name == "app" && filepath.Base(fullPath) != expectedAppName {
			return nil, fmt.Errorf("official app must be %s", expectedAppName)
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("flash part %s missing: %w", name, err)
		}
		sum, err := sha256File(fullPath)
		if err != nil {
			return nil, fmt.Errorf("hash flash part %s: %w", name, err)
		}
		seenOffsets[entry.offset] = true
		parts = append(parts, stackChanOfficialSmokeFlashPart{
			Name:      name,
			Offset:    entry.offset,
			Path:      fullPath,
			SHA256:    sum,
			SizeBytes: stat.Size(),
		})
	}
	for offset, name := range required {
		if !seenOffsets[offset] {
			return nil, fmt.Errorf("flash_args missing required %s at %s", name, offset)
		}
	}
	if err := validateOfficialAssetsPartitionCapacity(buildDir); err != nil {
		return nil, err
	}
	return parts, nil
}

type officialBinaryPartitionEntry struct {
	Label  string
	Offset uint32
	Size   uint32
}

func validateOfficialAssetsPartitionCapacity(buildDir string) error {
	entries, parsed, err := readOfficialBinaryPartitionTable(filepath.Join(buildDir, "partition_table", "partition-table.bin"))
	if err != nil {
		return err
	}
	if !parsed {
		return nil
	}
	var assets *officialBinaryPartitionEntry
	for i := range entries {
		if entries[i].Label == "assets" {
			assets = &entries[i]
			break
		}
	}
	if assets == nil {
		return fmt.Errorf("partition table missing assets partition")
	}
	assetsPath := filepath.Join(buildDir, "generated_assets.bin")
	stat, err := os.Stat(assetsPath)
	if err != nil {
		return fmt.Errorf("generated assets image missing: %w", err)
	}
	if stat.Size() > int64(assets.Size) {
		return fmt.Errorf("generated assets image size %d exceeds assets partition size %d at 0x%x", stat.Size(), assets.Size, assets.Offset)
	}
	return nil
}

func readOfficialBinaryPartitionTable(path string) ([]officialBinaryPartitionEntry, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read partition table: %w", err)
	}
	if len(data) < 32 || data[0] != 0xaa || data[1] != 0x50 {
		return nil, false, nil
	}
	entries := make([]officialBinaryPartitionEntry, 0, 8)
	for offset := 0; offset+32 <= len(data); offset += 32 {
		entry := data[offset : offset+32]
		if entry[0] == 0xeb && entry[1] == 0xeb {
			break
		}
		if entry[0] != 0xaa || entry[1] != 0x50 {
			break
		}
		labelBytes := entry[12:28]
		labelEnd := len(labelBytes)
		for i, b := range labelBytes {
			if b == 0 {
				labelEnd = i
				break
			}
		}
		entries = append(entries, officialBinaryPartitionEntry{
			Label:  string(labelBytes[:labelEnd]),
			Offset: binary.LittleEndian.Uint32(entry[4:8]),
			Size:   binary.LittleEndian.Uint32(entry[8:12]),
		})
	}
	return entries, true, nil
}

func validateOfficialPCMBridgeDeviceID(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device-id is required")
	}
	if containsLegacyIdentity(deviceID) {
		return fmt.Errorf("device-id contains forbidden legacy identity")
	}
	if !strings.HasPrefix(deviceID, "stackchan-") {
		return fmt.Errorf("device-id must use stackchan-*")
	}
	return nil
}

func parseOfficialPCMBridgeAudioWSURL(rawURL string, deviceID string) (stackChanOfficialPCMBridgeAudioWS, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("--audio-ws-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url contains forbidden legacy identity")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("parse audio ws url: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url must use ws or wss")
	}
	if parsed.User != nil {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url host is required")
	}
	if parsed.Path != "/ws/audio" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url path must be /ws/audio")
	}
	if parsed.Query().Get("device_id") != deviceID {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url device_id query must match --device-id")
	}
	return stackChanOfficialPCMBridgeAudioWS{
		Scheme:        parsed.Scheme,
		Host:          parsed.Host,
		Path:          parsed.Path,
		DeviceIDQuery: true,
	}, nil
}

func parseOfficialXiaozhiNVSOTAURL(rawURL string) (stackChanOfficialXiaozhiNVSOTA, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("--ota-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url contains forbidden legacy identity")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("parse ota url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url must use http or https")
	}
	if parsed.User != nil {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url host is required")
	}
	if parsed.Path != "/xiaozhi/ota/" && parsed.Path != "/xiaozhi/ota" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url path must be /xiaozhi/ota/")
	}
	if isLoopbackOrUnspecifiedHost(parsed.Hostname()) {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url host must be reachable by the physical device")
	}
	return stackChanOfficialXiaozhiNVSOTA{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
	}, nil
}

func parseOfficialXiaozhiNVSWebSocketURL(rawURL string, version int) (stackChanOfficialXiaozhiNVSWebSocket, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("--websocket-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url contains forbidden legacy identity")
	}
	if version <= 0 {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket version must be positive")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("parse websocket url: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url must use ws or wss")
	}
	if parsed.User != nil {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url host is required")
	}
	if parsed.Path != "/v1/xiaozhi" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url path must be /v1/xiaozhi")
	}
	if isLoopbackOrUnspecifiedHost(parsed.Hostname()) {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url host must be reachable by the physical device")
	}
	return stackChanOfficialXiaozhiNVSWebSocket{
		Scheme:          parsed.Scheme,
		Host:            parsed.Host,
		Path:            parsed.Path,
		Version:         version,
		TokenConfigured: false,
	}, nil
}

func validateOfficialXiaozhiNVSWiFiCredentials(ssid string, password string) (bool, error) {
	ssid = strings.TrimSpace(ssid)
	password = strings.TrimSpace(password)
	if ssid == "" && password == "" {
		return false, nil
	}
	if ssid == "" || password == "" {
		return false, fmt.Errorf("wifi ssid and password must be provided together")
	}
	if len(ssid) > 32 {
		return false, fmt.Errorf("wifi ssid is too long")
	}
	if len(password) < 8 || len(password) > 63 {
		return false, fmt.Errorf("wifi password length is invalid")
	}
	for _, value := range []string{ssid, password} {
		if containsLegacyIdentity(value) || strings.Contains(value, "\n") || strings.Contains(value, "\r") || strings.Contains(value, "\x00") {
			return false, fmt.Errorf("wifi credentials are invalid")
		}
	}
	return true, nil
}

type stackChanNVSMinimalEntry struct {
	Namespace string      `json:"namespace"`
	Key       string      `json:"key"`
	Encoding  string      `json:"encoding"`
	Data      interface{} `json:"data"`
	State     string      `json:"state"`
	IsEmpty   bool        `json:"is_empty"`
}

func readStackChanNVSMinimalEntries(path string) ([]stackChanNVSMinimalEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read NVS JSON dump: %w", err)
	}
	var entries []stackChanNVSMinimalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode NVS JSON dump: %w", err)
	}
	return entries, nil
}

func writeOfficialPCMBridgeNVSCSV(writer io.Writer, entries []stackChanNVSMinimalEntry, deviceID string, audioWSURL string) (stackChanOfficialPCMBridgeNVSSummary, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{"key", "type", "encoding", "value"}); err != nil {
		return stackChanOfficialPCMBridgeNVSSummary{}, err
	}

	namespaceOrder := make([]string, 0)
	seenNamespaces := make(map[string]bool)
	grouped := make(map[string][][]string)
	summary := stackChanOfficialPCMBridgeNVSSummary{}
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		namespace := strings.TrimSpace(entry.Namespace)
		key := strings.TrimSpace(entry.Key)
		if namespace == "" || key == "" {
			continue
		}
		if namespace == "a21" && (key == "device_id" || key == "audio_ws_url") {
			summary.ExistingA21EntryCount += 1
			continue
		}
		encoding, err := nvsCSVEncoding(entry.Encoding)
		if err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		value, err := nvsCSVValue(entry.Data)
		if err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
		grouped[namespace] = append(grouped[namespace], []string{key, "data", encoding, value})
		summary.PreservedEntryCount += 1
		if namespace == "servo" && (key == "zero_pos_1" || key == "zero_pos_2") {
			if hasNVSEntry(entries, "servo", "zero_pos_1") && hasNVSEntry(entries, "servo", "zero_pos_2") {
				summary.ServoCalibrationPresent = true
			}
		}
	}
	if !seenNamespaces["a21"] {
		seenNamespaces["a21"] = true
		namespaceOrder = append(namespaceOrder, "a21")
	}
	grouped["a21"] = append(grouped["a21"],
		[]string{"device_id", "data", "string", deviceID},
		[]string{"audio_ws_url", "data", "string", audioWSURL},
	)
	summary.MutatedEntryCount = 2

	for _, namespace := range namespaceOrder {
		if err := csvWriter.Write([]string{namespace, "namespace", "", ""}); err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		for _, row := range grouped[namespace] {
			if err := csvWriter.Write(row); err != nil {
				return stackChanOfficialPCMBridgeNVSSummary{}, err
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return stackChanOfficialPCMBridgeNVSSummary{}, err
	}
	return summary, nil
}

func writeOfficialXiaozhiCompatibleNVSCSV(writer io.Writer, entries []stackChanNVSMinimalEntry, otaURL string, websocketURL string, websocketVersion int, wifiSSID string, wifiPassword string) (stackChanOfficialXiaozhiCompatibleNVSSummary, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{"key", "type", "encoding", "value"}); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
	}
	wifiCredentialsRequested := strings.TrimSpace(wifiSSID) != "" || strings.TrimSpace(wifiPassword) != ""

	namespaceOrder := make([]string, 0)
	seenNamespaces := make(map[string]bool)
	grouped := make(map[string][][]string)
	summary := stackChanOfficialXiaozhiCompatibleNVSSummary{}
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		namespace := strings.TrimSpace(entry.Namespace)
		key := strings.TrimSpace(entry.Key)
		if namespace == "" || key == "" {
			continue
		}
		if isOfficialXiaozhiConnectionNVSKey(namespace, key) {
			summary.ExistingConnectionEntryCount += 1
			continue
		}
		encoding, err := nvsCSVEncoding(entry.Encoding)
		if err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		value, err := nvsCSVValue(entry.Data)
		if err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
		grouped[namespace] = append(grouped[namespace], []string{key, "data", encoding, value})
		summary.PreservedEntryCount += 1
		if namespace == "servo" && (key == "zero_pos_1" || key == "zero_pos_2") {
			if hasNVSEntry(entries, "servo", "zero_pos_1") && hasNVSEntry(entries, "servo", "zero_pos_2") {
				summary.ServoCalibrationPresent = true
			}
		}
	}
	summary.WiFiCredentialsPreserved = !wifiCredentialsRequested && hasNVSEntry(entries, "wifi", "ssid") && hasNVSEntry(entries, "wifi", "password")
	appConfigShouldMarkConfigured := wifiCredentialsRequested || summary.WiFiCredentialsPreserved
	for _, namespace := range []string{"wifi", "websocket"} {
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
	}
	if appConfigShouldMarkConfigured && !seenNamespaces["app_config"] {
		seenNamespaces["app_config"] = true
		namespaceOrder = append(namespaceOrder, "app_config")
	}
	if wifiCredentialsRequested {
		grouped["wifi"] = removeNVSRows(grouped["wifi"], "ssid", "password")
		grouped["wifi"] = append(grouped["wifi"],
			[]string{"ssid", "data", "string", strings.TrimSpace(wifiSSID)},
			[]string{"password", "data", "string", strings.TrimSpace(wifiPassword)},
		)
		summary.WiFiCredentialsWritten = true
	}
	grouped["wifi"] = append(grouped["wifi"], []string{"ota_url", "data", "string", otaURL})
	grouped["websocket"] = append(grouped["websocket"],
		[]string{"url", "data", "string", websocketURL},
		[]string{"version", "data", "u32", strconv.Itoa(websocketVersion)},
	)
	if appConfigShouldMarkConfigured {
		grouped["app_config"] = removeNVSRows(grouped["app_config"], "is_configed")
		grouped["app_config"] = append(grouped["app_config"], []string{"is_configed", "data", "u8", "1"})
		summary.AppConfigMarkedConfigured = true
	}
	summary.MutatedEntryCount = 3
	if wifiCredentialsRequested {
		summary.MutatedEntryCount += 2
	}
	if appConfigShouldMarkConfigured {
		summary.MutatedEntryCount += 1
	}

	for _, namespace := range namespaceOrder {
		if err := csvWriter.Write([]string{namespace, "namespace", "", ""}); err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		for _, row := range grouped[namespace] {
			if err := csvWriter.Write(row); err != nil {
				return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
	}
	return summary, nil
}

func removeNVSRows(rows [][]string, keys ...string) [][]string {
	if len(rows) == 0 || len(keys) == 0 {
		return rows
	}
	remove := make(map[string]bool, len(keys))
	for _, key := range keys {
		remove[key] = true
	}
	filtered := rows[:0]
	for _, row := range rows {
		if len(row) > 0 && remove[row[0]] {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func isOfficialXiaozhiConnectionNVSKey(namespace string, key string) bool {
	switch namespace {
	case "wifi":
		return key == "ota_url"
	case "websocket":
		return key == "url" || key == "token" || key == "version"
	case "app_config":
		return key == "is_configed"
	default:
		return false
	}
}

func hasNVSEntry(entries []stackChanNVSMinimalEntry, namespace string, key string) bool {
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		if entry.Namespace == namespace && entry.Key == key {
			return true
		}
	}
	return false
}

func nvsCSVEncoding(encoding string) (string, error) {
	switch encoding {
	case "string":
		return "string", nil
	case "blob_data":
		return "base64", nil
	case "uint8_t":
		return "u8", nil
	case "int8_t":
		return "i8", nil
	case "uint16_t":
		return "u16", nil
	case "int16_t":
		return "i16", nil
	case "uint32_t":
		return "u32", nil
	case "int32_t":
		return "i32", nil
	case "uint64_t":
		return "u64", nil
	case "int64_t":
		return "i64", nil
	default:
		return "", fmt.Errorf("unsupported NVS encoding %q", encoding)
	}
}

func nvsCSVValue(value interface{}) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case float64:
		if typed != float64(int64(typed)) {
			return "", fmt.Errorf("unsupported non-integer NVS number")
		}
		return strconv.FormatInt(int64(typed), 10), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case uint64:
		return strconv.FormatUint(typed, 10), nil
	default:
		return "", fmt.Errorf("unsupported NVS value type %T", value)
	}
}

func verifyOfficialPCMBridgeNVSProvision(path string, deviceID string, audioWSURL string) error {
	entries, err := readStackChanNVSMinimalEntries(path)
	if err != nil {
		return err
	}
	if !nvsEntryEquals(entries, "a21", "device_id", deviceID) {
		return fmt.Errorf("provisioned NVS missing a21/device_id")
	}
	if !nvsEntryEquals(entries, "a21", "audio_ws_url", audioWSURL) {
		return fmt.Errorf("provisioned NVS missing a21/audio_ws_url")
	}
	return nil
}

func verifyOfficialXiaozhiCompatibleNVSProvision(path string, otaURL string, websocketURL string, websocketVersion int, wifiSSID string, wifiPassword string) error {
	entries, err := readStackChanNVSMinimalEntries(path)
	if err != nil {
		return err
	}
	wifiCredentialsRequested := strings.TrimSpace(wifiSSID) != "" || strings.TrimSpace(wifiPassword) != ""
	if wifiCredentialsRequested {
		if !nvsEntryEquals(entries, "wifi", "ssid", strings.TrimSpace(wifiSSID)) {
			return fmt.Errorf("provisioned NVS missing wifi/ssid")
		}
		if !nvsEntryEquals(entries, "wifi", "password", strings.TrimSpace(wifiPassword)) {
			return fmt.Errorf("provisioned NVS missing wifi/password")
		}
	}
	hasProvisionedWiFiCredentials := hasNVSEntry(entries, "wifi", "ssid") && hasNVSEntry(entries, "wifi", "password")
	if hasProvisionedWiFiCredentials && !nvsEntryEquals(entries, "app_config", "is_configed", "1") {
		return fmt.Errorf("provisioned NVS missing app_config/is_configed")
	}
	if !nvsEntryEquals(entries, "wifi", "ota_url", otaURL) {
		return fmt.Errorf("provisioned NVS missing wifi/ota_url")
	}
	if !nvsEntryEquals(entries, "websocket", "url", websocketURL) {
		return fmt.Errorf("provisioned NVS missing websocket/url")
	}
	if !nvsEntryEquals(entries, "websocket", "version", strconv.Itoa(websocketVersion)) {
		return fmt.Errorf("provisioned NVS missing websocket/version")
	}
	if hasNVSEntry(entries, "websocket", "token") {
		return fmt.Errorf("provisioned NVS must clear websocket/token")
	}
	return nil
}

func nvsEntryEquals(entries []stackChanNVSMinimalEntry, namespace string, key string, expected string) bool {
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		if entry.Namespace == namespace && entry.Key == key {
			actual, err := nvsCSVValue(entry.Data)
			return err == nil && actual == expected
		}
	}
	return false
}

func officialIDFToolPath(idfExport string, relativePath string) string {
	return filepath.Join(filepath.Dir(filepath.Clean(idfExport)), filepath.FromSlash(relativePath))
}

func executeStackChanOfficialSmokeFlash(ctx context.Context, options stackChanOfficialSmokeFlashOptions, report *stackChanOfficialSmokeFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	report.FlashLogPath = filepath.Join(options.BuildDir, fmt.Sprintf("a21-official-audio-smoke-flash-%s.log", time.Now().Format("20060102-150405")))
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runStackChanOfficialSmokeFlashCommand(ctx, report.FlashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialBaselineFlash(ctx context.Context, options stackChanOfficialSmokeFlashOptions, report *stackChanOfficialSmokeFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	report.FlashLogPath = filepath.Join(options.BuildDir, fmt.Sprintf("a21-official-baseline-flash-%s.log", time.Now().Format("20060102-150405")))
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runStackChanOfficialBaselineFlashCommand(ctx, report.FlashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}
