package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
)

func TestRunStackChanOfficialPCMBridgeNVSExecuteStopsWhenControlGuardBlocks(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalGuard := runA21ControlGuard
	runA21ControlGuard = func(ctx context.Context, input runtimeguard.ControlGuardInput) runtimeguard.ControlGuardReport {
		return runtimeguard.ControlGuardReport{
			Command: input.Command,
			Tier:    "T7",
			Result: runtimeguard.NewResult([]runtimeguard.Finding{{
				Code:     "control_hardware_window_branch_required",
				Severity: runtimeguard.SeverityBlock,
				Message:  "A21 hardware writes require a hardware-window branch",
			}}),
		}
	}
	defer func() {
		runA21ControlGuard = originalGuard
	}()
	originalRunner := runStackChanOfficialPCMBridgeNVSCommand
	var scripts []string
	runStackChanOfficialPCMBridgeNVSCommand = func(ctx context.Context, logPath string, script string) error {
		scripts = append(scripts, script)
		return nil
	}
	defer func() {
		runStackChanOfficialPCMBridgeNVSCommand = originalRunner
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if len(scripts) != 0 {
		t.Fatalf("scripts ran despite control guard block: %v", scripts)
	}
	if !strings.Contains(stderr.String(), "control_hardware_window_branch_required") {
		t.Fatalf("stderr missing control guard finding: %s", stderr.String())
	}
}

func TestRunStackChanOfficialPCMBridgeNVSExecuteRunsGuardedReadGenerateWriteFlow(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialPCMBridgeNVSCommand
	var scripts []string
	runStackChanOfficialPCMBridgeNVSCommand = func(ctx context.Context, logPath string, script string) error {
		scripts = append(scripts, script)
		switch {
		case strings.Contains(script, " read_flash "):
			writeTestFile(t, lastSingleQuotedPath(script), "backup")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "before-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForBridgeProvision("old-device", "ws://old/ws/audio?device_id=old-device"))
		case strings.Contains(script, "nvs_partition_generator/nvs_partition_gen.py"):
			writeTestFile(t, lastSingleQuotedPath(script), "provisioned")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "provision-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForBridgeProvision("stackchan-001", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001"))
		}
		return nil
	}
	defer func() {
		runStackChanOfficialPCMBridgeNVSCommand = originalRunner
	}()

	idfRoot := filepath.Join(t.TempDir(), "esp-idf-v5.5.2")
	idfExport := filepath.Join(idfRoot, "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_tool", "nvs_tool.py"), "#!/usr/bin/env python\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_generator", "nvs_partition_gen.py"), "#!/usr/bin/env python\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", idfExport,
		"--run-dir", filepath.Join(t.TempDir(), "a21-official-nvs-run"),
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if len(scripts) != 5 {
		t.Fatalf("scripts = %d, want 5: %v", len(scripts), scripts)
	}
	joined := strings.Join(scripts, "\n")
	for _, want := range []string{
		"read_flash 0x9000 0x4000",
		"nvs_partition_tool/nvs_tool.py",
		"nvs_partition_generator/nvs_partition_gen.py",
		"write_flash 0x9000",
		"--after hard_reset",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("scripts missing %q:\n%s", want, joined)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_pcm_bridge_nvs_execution.v1"`,
		`"write_allowed": true`,
		`"write_executed": true`,
		`"control_guard"`,
		`"preserved_entry_count": 5`,
		`"mutated_entry_count": 2`,
		`"servo_calibration_present": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001") ||
		strings.Contains(stdout.String(), "existing-secret") {
		t.Fatalf("execution report leaked sensitive values: %s", stdout.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleNVSPlanBuildsRedactedNoWriteReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-nvs-plan",
		"--port", "/dev/cu.usbmodemA21",
		"--ota-url", "http://192.0.2.10:21080/xiaozhi/ota/",
		"--websocket-url", "ws://192.0.2.10:21080/v1/xiaozhi",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_xiaozhi_compatible_nvs_plan.v1"`,
		`"dry_run": true`,
		`"write_allowed": false`,
		`"write_executed": false`,
		`"path": "/xiaozhi/ota/"`,
		`"path": "/v1/xiaozhi"`,
		`"version": 1`,
		`"only_mutates_xiaozhi_connection_keys": true`,
		`"preserves_wifi_credentials": true`,
		`"next_required_confirmation": "a21-stackchan-official-xiaozhi-compatible-nvs-execute_with_confirmation_token"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("nvs plan missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		"http://192.0.2.10:21080/xiaozhi/ota/",
		"ws://192.0.2.10:21080/v1/xiaozhi",
		"old-token",
		"Authorization",
		"existing-secret",
	} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("nvs plan leaked forbidden value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleNVSPlanRedactsExplicitWiFiCredentials(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-nvs-plan",
		"--port", "/dev/cu.usbmodemA21",
		"--ota-url", "http://192.0.2.10:21080/xiaozhi/ota/",
		"--websocket-url", "ws://192.0.2.10:21080/v1/xiaozhi",
		"--wifi-ssid", "A21-Lab-WiFi",
		"--wifi-password", "secret-password-123",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"preserves_wifi_credentials": false`,
		`"allows_explicit_wifi_credential_write": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("nvs plan missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		"A21-Lab-WiFi",
		"secret-password-123",
		"ssid",
		"password",
	} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("nvs plan leaked forbidden Wi-Fi value/key %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleNVSPlanRejectsLoopbackGateway(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-nvs-plan",
		"--port", "/dev/cu.usbmodemA21",
		"--ota-url", "http://127.0.0.1:21080/xiaozhi/ota/",
		"--websocket-url", "ws://127.0.0.1:21080/v1/xiaozhi",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("loopback nvs plan unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "host must be reachable by the physical device") {
		t.Fatalf("stderr missing physical reachability rejection: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), "127.0.0.1") {
		t.Fatalf("stdout leaked rejected loopback url: %s", stdout.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleNVSExecuteRequiresConfirmationToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--ota-url", "http://192.0.2.10:21080/xiaozhi/ota/",
		"--websocket-url", "ws://192.0.2.10:21080/v1/xiaozhi",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleNVSExecuteRunsGuardedReadGenerateWriteFlow(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialXiaozhiCompatibleNVSCommand
	var scripts []string
	runStackChanOfficialXiaozhiCompatibleNVSCommand = func(ctx context.Context, logPath string, script string) error {
		scripts = append(scripts, script)
		switch {
		case strings.Contains(script, " read_flash "):
			writeTestFile(t, lastSingleQuotedPath(script), "backup")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "before-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForXiaozhiCompatibleProvision("http://old.example/xiaozhi/ota/", "wss://old.example/v1/xiaozhi", true))
		case strings.Contains(script, "nvs_partition_generator/nvs_partition_gen.py"):
			writeTestFile(t, lastSingleQuotedPath(script), "provisioned")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "provision-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForXiaozhiCompatibleProvision("http://192.0.2.10:21080/xiaozhi/ota/", "ws://192.0.2.10:21080/v1/xiaozhi", false))
		}
		return nil
	}
	defer func() {
		runStackChanOfficialXiaozhiCompatibleNVSCommand = originalRunner
	}()

	idfRoot := filepath.Join(t.TempDir(), "esp-idf-v5.5.2")
	idfExport := filepath.Join(idfRoot, "export.sh")
	idfPython := filepath.Join(idfRoot, "python-env", "bin", "python")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	writeTestFile(t, idfPython, "#!/usr/bin/env python\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_tool", "nvs_tool.py"), "#!/usr/bin/env python\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_generator", "nvs_partition_gen.py"), "#!/usr/bin/env python\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--ota-url", "http://192.0.2.10:21080/xiaozhi/ota/",
		"--websocket-url", "ws://192.0.2.10:21080/v1/xiaozhi",
		"--idf-export", idfExport,
		"--idf-python", idfPython,
		"--run-dir", filepath.Join(t.TempDir(), "a21-official-xiaozhi-nvs-run"),
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if len(scripts) != 5 {
		t.Fatalf("scripts = %d, want 5: %v", len(scripts), scripts)
	}
	joined := strings.Join(scripts, "\n")
	for _, want := range []string{
		shellSingleQuote(idfPython),
		"read_flash 0x9000 0x4000",
		"nvs_partition_tool/nvs_tool.py",
		"nvs_partition_generator/nvs_partition_gen.py",
		"write_flash 0x9000",
		"--after hard_reset",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("scripts missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "source ") {
		t.Fatalf("idf python override scripts must not source export.sh:\n%s", joined)
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_xiaozhi_compatible_nvs_execution.v1"`,
		`"write_allowed": true`,
		`"write_executed": true`,
		`"control_guard"`,
		`"idf_python_path":`,
		`"preserved_entry_count": 5`,
		`"mutated_entry_count": 4`,
		`"existing_connection_entry_count": 5`,
		`"wifi_credentials_preserved": true`,
		`"app_config_marked_configured": true`,
		`"servo_calibration_present": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{
		"http://192.0.2.10:21080/xiaozhi/ota/",
		"ws://192.0.2.10:21080/v1/xiaozhi",
		"old-token",
		"existing-secret",
	} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("execution report leaked forbidden value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestOfficialPCMBridgeNVSCSVPreservesExistingEntriesAndOnlyOverwritesA21Keys(t *testing.T) {
	entries := []stackChanNVSMinimalEntry{
		{Namespace: "board", Key: "uuid", Encoding: "string", Data: "device-uuid", State: "Written"},
		{Namespace: "wifi", Key: "ssid", Encoding: "string", Data: "existing-wifi", State: "Written"},
		{Namespace: "wifi", Key: "password", Encoding: "string", Data: "existing-secret", State: "Written"},
		{Namespace: "servo", Key: "zero_pos_1", Encoding: "int32_t", Data: float64(460), State: "Written"},
		{Namespace: "servo", Key: "zero_pos_2", Encoding: "int32_t", Data: float64(620), State: "Written"},
		{Namespace: "a21", Key: "device_id", Encoding: "string", Data: "old-device", State: "Written"},
		{Namespace: "a21", Key: "audio_ws_url", Encoding: "string", Data: "ws://old/ws/audio?device_id=old-device", State: "Written"},
	}

	var csv bytes.Buffer
	summary, err := writeOfficialPCMBridgeNVSCSV(&csv, entries, "stackchan-001", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001")
	if err != nil {
		t.Fatalf("write csv: %v", err)
	}
	text := csv.String()
	for _, want := range []string{
		"board,namespace,,",
		"uuid,data,string,device-uuid",
		"wifi,namespace,,",
		"ssid,data,string,existing-wifi",
		"password,data,string,existing-secret",
		"servo,namespace,,",
		"zero_pos_1,data,i32,460",
		"zero_pos_2,data,i32,620",
		"a21,namespace,,",
		"device_id,data,string,stackchan-001",
		"audio_ws_url,data,string,ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("csv missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "old-device") {
		t.Fatalf("csv retained stale a21 value:\n%s", text)
	}
	if summary.PreservedEntryCount != 5 || summary.MutatedEntryCount != 2 || !summary.ServoCalibrationPresent {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestOfficialXiaozhiCompatibleNVSCSVPreservesWiFiAndClearsWebsocketToken(t *testing.T) {
	entries := []stackChanNVSMinimalEntry{
		{Namespace: "board", Key: "uuid", Encoding: "string", Data: "device-uuid", State: "Written"},
		{Namespace: "wifi", Key: "ssid", Encoding: "string", Data: "existing-wifi", State: "Written"},
		{Namespace: "wifi", Key: "password", Encoding: "string", Data: "existing-secret", State: "Written"},
		{Namespace: "wifi", Key: "ota_url", Encoding: "string", Data: "http://old.example/xiaozhi/ota/", State: "Written"},
		{Namespace: "websocket", Key: "url", Encoding: "string", Data: "wss://old.example/v1/xiaozhi", State: "Written"},
		{Namespace: "websocket", Key: "token", Encoding: "string", Data: "old-token", State: "Written"},
		{Namespace: "websocket", Key: "version", Encoding: "uint32_t", Data: float64(3), State: "Written"},
		{Namespace: "servo", Key: "zero_pos_1", Encoding: "int32_t", Data: float64(460), State: "Written"},
		{Namespace: "servo", Key: "zero_pos_2", Encoding: "int32_t", Data: float64(620), State: "Written"},
	}

	var csv bytes.Buffer
	summary, err := writeOfficialXiaozhiCompatibleNVSCSV(&csv, entries, "http://192.0.2.10:21080/xiaozhi/ota/", "ws://192.0.2.10:21080/v1/xiaozhi", 1, "", "")
	if err != nil {
		t.Fatalf("write csv: %v", err)
	}
	text := csv.String()
	for _, want := range []string{
		"board,namespace,,",
		"uuid,data,string,device-uuid",
		"wifi,namespace,,",
		"ssid,data,string,existing-wifi",
		"password,data,string,existing-secret",
		"ota_url,data,string,http://192.0.2.10:21080/xiaozhi/ota/",
		"websocket,namespace,,",
		"url,data,string,ws://192.0.2.10:21080/v1/xiaozhi",
		"version,data,u32,1",
		"app_config,namespace,,",
		"is_configed,data,u8,1",
		"servo,namespace,,",
		"zero_pos_1,data,i32,460",
		"zero_pos_2,data,i32,620",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("csv missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"old-token", "wss://old.example", "http://old.example", "token,data"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("csv retained forbidden value %q:\n%s", forbidden, text)
		}
	}
	if summary.PreservedEntryCount != 5 ||
		summary.MutatedEntryCount != 4 ||
		summary.ExistingConnectionEntryCount != 4 ||
		!summary.WiFiCredentialsPreserved ||
		!summary.AppConfigMarkedConfigured ||
		!summary.ServoCalibrationPresent {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestOfficialXiaozhiCompatibleNVSCSVCanWriteExplicitWiFiCredentials(t *testing.T) {
	entries := []stackChanNVSMinimalEntry{
		{Namespace: "wifi", Key: "ssid", Encoding: "string", Data: "old-wifi", State: "Written"},
		{Namespace: "wifi", Key: "password", Encoding: "string", Data: "old-secret", State: "Written"},
		{Namespace: "servo", Key: "zero_pos_1", Encoding: "int32_t", Data: float64(460), State: "Written"},
		{Namespace: "servo", Key: "zero_pos_2", Encoding: "int32_t", Data: float64(620), State: "Written"},
	}

	var csv bytes.Buffer
	summary, err := writeOfficialXiaozhiCompatibleNVSCSV(&csv, entries, "http://192.0.2.10:21080/xiaozhi/ota/", "ws://192.0.2.10:21080/v1/xiaozhi", 1, "A21-Lab-WiFi", "secret-password-123")
	if err != nil {
		t.Fatalf("write csv: %v", err)
	}
	text := csv.String()
	for _, want := range []string{
		"wifi,namespace,,",
		"ssid,data,string,A21-Lab-WiFi",
		"password,data,string,secret-password-123",
		"ota_url,data,string,http://192.0.2.10:21080/xiaozhi/ota/",
		"url,data,string,ws://192.0.2.10:21080/v1/xiaozhi",
		"version,data,u32,1",
		"app_config,namespace,,",
		"is_configed,data,u8,1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("csv missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"old-wifi", "old-secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("csv retained stale Wi-Fi credential %q:\n%s", forbidden, text)
		}
	}
	if summary.MutatedEntryCount != 6 ||
		summary.WiFiCredentialsPreserved ||
		!summary.WiFiCredentialsWritten ||
		!summary.AppConfigMarkedConfigured ||
		!summary.ServoCalibrationPresent {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestOfficialXiaozhiCompatibleNVSCSVReplacesStaleAppConfigGate(t *testing.T) {
	entries := []stackChanNVSMinimalEntry{
		{Namespace: "wifi", Key: "ssid", Encoding: "string", Data: "old-wifi", State: "Written"},
		{Namespace: "wifi", Key: "password", Encoding: "string", Data: "old-secret", State: "Written"},
		{Namespace: "app_config", Key: "is_configed", Encoding: "uint8_t", Data: float64(0), State: "Written"},
	}

	var csv bytes.Buffer
	summary, err := writeOfficialXiaozhiCompatibleNVSCSV(&csv, entries, "http://192.0.2.10:21080/xiaozhi/ota/", "ws://192.0.2.10:21080/v1/xiaozhi", 1, "A21-Lab-WiFi", "secret-password-123")
	if err != nil {
		t.Fatalf("write csv: %v", err)
	}
	text := csv.String()
	for _, want := range []string{
		"app_config,namespace,,",
		"is_configed,data,u8,1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("csv missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "is_configed,data,u8,0") {
		t.Fatalf("csv retained stale app config gate:\n%s", text)
	}
	if summary.ExistingConnectionEntryCount != 1 || !summary.AppConfigMarkedConfigured {
		t.Fatalf("summary = %+v", summary)
	}
}

func lastSingleQuotedPath(text string) string {
	end := strings.LastIndex(text, "'")
	if end <= 0 {
		return ""
	}
	start := strings.LastIndex(text[:end], "'")
	if start < 0 {
		return ""
	}
	return text[start+1 : end]
}

func redirectSingleQuotedPath(text string) string {
	redirect := strings.LastIndex(text, "> ")
	if redirect < 0 {
		return ""
	}
	return lastSingleQuotedPath(text[redirect:])
}

func testNVSJSONForBridgeProvision(deviceID string, audioWSURL string) string {
	return fmt.Sprintf(`[
  {"namespace":"board","key":"uuid","encoding":"string","data":"device-uuid","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"ssid","encoding":"string","data":"existing-wifi","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"password","encoding":"string","data":"existing-secret","state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_1","encoding":"int32_t","data":460,"state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_2","encoding":"int32_t","data":620,"state":"Written","is_empty":false},
  {"namespace":"a21","key":"device_id","encoding":"string","data":%q,"state":"Written","is_empty":false},
  {"namespace":"a21","key":"audio_ws_url","encoding":"string","data":%q,"state":"Written","is_empty":false}
]`, deviceID, audioWSURL)
}

func testNVSJSONForXiaozhiCompatibleProvision(otaURL string, websocketURL string, includeToken bool) string {
	tokenEntry := ""
	if includeToken {
		tokenEntry = `  {"namespace":"websocket","key":"token","encoding":"string","data":"old-token","state":"Written","is_empty":false},` + "\n"
	}
	return fmt.Sprintf(`[
  {"namespace":"board","key":"uuid","encoding":"string","data":"device-uuid","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"ssid","encoding":"string","data":"existing-wifi","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"password","encoding":"string","data":"existing-secret","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"ota_url","encoding":"string","data":%q,"state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_1","encoding":"int32_t","data":460,"state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_2","encoding":"int32_t","data":620,"state":"Written","is_empty":false},
  {"namespace":"websocket","key":"url","encoding":"string","data":%q,"state":"Written","is_empty":false},
%s  {"namespace":"websocket","key":"version","encoding":"uint32_t","data":1,"state":"Written","is_empty":false},
  {"namespace":"app_config","key":"is_configed","encoding":"uint8_t","data":1,"state":"Written","is_empty":false}
]`, otaURL, websocketURL, tokenEntry)
}

func writeTestOfficialPCMBridgeNVSExecutionReport(t *testing.T, deviceID string, host string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a21-stackchan-official-pcm-bridge-nvs-execution.json")
	report := stackChanOfficialPCMBridgeNVSReport{
		SchemaVersion: stackChanOfficialPCMBridgeNVSExecutionSchema,
		Status:        "passed",
		DryRun:        false,
		WriteAllowed:  true,
		WriteExecuted: true,
		Port:          "/dev/cu.usbmodemA21",
		DeviceID:      deviceID,
		AudioWS: stackChanOfficialPCMBridgeAudioWS{
			Scheme:        "ws",
			Host:          host,
			Path:          "/ws/audio",
			DeviceIDQuery: true,
		},
		Partition: stackChanOfficialPCMBridgeNVSPartition{
			Offset:    stackChanOfficialPCMBridgeNVSOffset,
			SizeHex:   stackChanOfficialPCMBridgeNVSSizeHex,
			SizeBytes: stackChanOfficialPCMBridgeNVSSizeBytes,
		},
		Safety: stackChanOfficialPCMBridgeNVSSafety{
			BackupBeforeWrite:       true,
			PreserveExistingEntries: true,
			OnlyMutatesA21Namespace: true,
			ReportRedactsValues:     true,
		},
		Summary: &stackChanOfficialPCMBridgeNVSSummary{
			PreservedEntryCount:     39,
			MutatedEntryCount:       2,
			ServoCalibrationPresent: true,
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal nvs report: %v", err)
	}
	writeTestFile(t, path, string(data))
	return path
}

func writeTestOfficialStackChanRepo(t *testing.T, includeCodecEvidence bool) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "firmware", "README.md"), "idf.py build\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "sdkconfig.defaults"), strings.Join([]string{
		`CONFIG_IDF_TARGET="esp32s3"`,
		`CONFIG_LANGUAGE_EN_US=y`,
		`CONFIG_BOARD_TYPE_M5STACK_STACK_CHAN=y`,
		`CONFIG_SEND_WAKE_WORD_DATA=n`,
	}, "\n")+"\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "CMakeLists.txt"), strings.Join([]string{
		`cmake_minimum_required(VERSION 3.16)`,
		`set(PROJECT_VER "1.4.1")`,
		`add_definitions(-DFIRMWARE_VERSION="${PROJECT_VER}")`,
		`include($ENV{IDF_PATH}/tools/cmake/project.cmake)`,
		`project(stack-chan)`,
	}, "\n")+"\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "main.cpp"), strings.Join([]string{
		`#include <smooth_ui_toolkit.hpp>`,
		`#include <uitk/short_namespace.hpp>`,
		`#include <mooncake.h>`,
		`#include <apps/apps.h>`,
		`#include <hal/hal.h>`,
		`extern "C" void app_main(void)`,
		`{`,
		`    GetHAL().init();`,
		`    GetMooncake().installApp(std::make_unique<AppLauncher>());`,
		`    GetMooncake().installApp(std::make_unique<AppAiAgent>());`,
		`    GetMooncake().installApp(std::make_unique<AppAvatar>());`,
		`    while (1) {`,
		`        GetMooncake().update();`,
		`        if (GetHAL().isXiaozhiStartRequested()) {`,
		`            break;`,
		`        }`,
		`    }`,
		`    GetHAL().startXiaozhi();`,
		`}`,
	}, "\n")+"\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "repos.json"), `[
  {
    "url": "https://github.com/78/xiaozhi-esp32.git",
    "path": "xiaozhi-esp32",
    "branch": "v2.2.4",
    "patch": "patches/xiaozhi-esp32.patch"
  }
]`)

	audioCPP := "void Hal::startMicTest() {}\n"
	codecCC := "class CoreS3AudioCodec {};\n"
	if includeCodecEvidence {
		audioCPP = "void Hal::startMicTest() { audio_codec->OutputData(output_chunk); }\n"
		codecCC = "void CoreS3AudioCodec::CreateDuplexChannels() {}\nvoid CoreS3AudioCodec::EnableOutput() { esp_codec_dev_open(output_dev_, &fs); }\nint CoreS3AudioCodec::Write(const int16_t* data, int samples) { esp_codec_dev_write(output_dev_, (void*)data, samples * sizeof(int16_t)); return samples; }\n"
	}
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "hal", "audio.cpp"), audioCPP)
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "hal", "board", "cores3_audio_codec.cc"), codecCC)
	writeTestFile(t, filepath.Join(dir, "firmware", "xiaozhi-esp32", "main", "audio", "audio_service.cc"), "void AudioService::AudioOutputTask() { codec_->OutputData(task->pcm); }\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "xiaozhi-esp32", "main", "audio", "audio_service.h"), "#define OPUS_FRAME_DURATION_MS 60\n")

	runGitForTest(t, dir, "init")
	runGitForTest(t, dir, "add", ".")
	runGitForTest(t, dir, "-c", "user.name=A21 Test", "-c", "user.email=a21@example.invalid", "commit", "-m", "official baseline")
	return dir
}

func writeTestOfficialAudioSmokeBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-audio-smoke.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-audio-smoke.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestOfficialBaselineBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "stack-chan.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 stack-chan.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestOfficialPCMBridgeBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-pcm-bridge.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-pcm-bridge.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestOfficialXiaozhiCompatibleBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-xiaozhi-compatible.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-xiaozhi-compatible.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestXiaozhiFirmwareBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-xiaozhi-firmware-build")
	return writeTestXiaozhiFirmwareBuildAt(t, buildDir)
}

func writeTestXiaozhiFirmwareBuildAt(t *testing.T, buildDir string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "xiaozhi.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "config", "sdkconfig.json"), `{
  "BOARD_TYPE_M5STACK_CORE_S3": true,
  "OTA_URL": "http://192.0.2.10:21080/xiaozhi/ota/",
  "ENABLE_X21_DEVICE_EVENTS": false
}`)
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 xiaozhi.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0x800000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestOfficialPartitionTable(t *testing.T, path string, assetsSize uint32) {
	t.Helper()
	data := make([]byte, 0, 32*8)
	appendPartition := func(label string, partitionType byte, subtype byte, offset uint32, size uint32) {
		entry := make([]byte, 32)
		entry[0] = 0xaa
		entry[1] = 0x50
		entry[2] = partitionType
		entry[3] = subtype
		binary.LittleEndian.PutUint32(entry[4:8], offset)
		binary.LittleEndian.PutUint32(entry[8:12], size)
		copy(entry[12:28], []byte(label))
		data = append(data, entry...)
	}
	appendPartition("nvs", 0x01, 0x02, 0x9000, 0x4000)
	appendPartition("otadata", 0x01, 0x00, 0xd000, 0x2000)
	appendPartition("phy_init", 0x01, 0x01, 0xf000, 0x1000)
	appendPartition("ota_0", 0x00, 0x10, 0x20000, 0x4f0000)
	appendPartition("ota_1", 0x00, 0x11, 0x510000, 0x4f0000)
	appendPartition("assets", 0x01, 0x82, 0xa00000, assetsSize)
	appendPartition("coredump", 0x01, 0x03, 0xe00000, 0x10000)
	data = append(data, bytes.Repeat([]byte{0xff}, 32)...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write partition table %s: %v", path, err)
	}
}

func writeTestSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()
	if err := file.Truncate(int64(size)); err != nil {
		t.Fatalf("truncate %s: %v", path, err)
	}
}

func frozenLegacyXiaozhiFirmwareBuildDirFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "小马暴力", "sources", "xiaozhi-esp32", "build-m5stack-core-s3")
}

func assertNoFrozenLegacyXiaozhiFirmwareSourceLeak(t *testing.T, buildDir string, outputs ...string) {
	t.Helper()
	for _, output := range outputs {
		for _, forbidden := range []string{
			buildDir,
			filepath.Dir(buildDir),
			filepath.Dir(filepath.Dir(buildDir)),
			"小马暴力",
			"xiaozhi-esp32",
		} {
			if forbidden != "" && strings.Contains(output, forbidden) {
				t.Fatalf("frozen firmware source rejection leaked %q: %s", forbidden, output)
			}
		}
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
