package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWakeWordFirmwarePlanBuildsCustomNoFlashReport(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "a21-wake-word.json")
	outputDir := filepath.Join(dir, "reports")
	if err := os.WriteFile(configPath, []byte(`{"mode":"custom_multinet","desired_phrase":"小阿二一","desired_pinyin":"xiao a er yi","threshold":35}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"wake-word-firmware-plan", "--config", configPath, "--output-dir", outputDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var report wakeWordFirmwarePlanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw stdout: %v\n%s", err, stdout.String())
	}
	if report.SchemaVersion != "a21.wake_word_firmware_plan.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.Status != "pending_firmware_build" || !report.DryRun || report.BuildAllowed || report.FlashAllowed {
		t.Fatalf("status/dry/build/flash = %q/%v/%v/%v, want pending dry no-build no-flash", report.Status, report.DryRun, report.BuildAllowed, report.FlashAllowed)
	}
	if report.Mode != "custom_multinet" || report.DesiredPhrase != "小阿二一" || report.DesiredPinyin != "xiao a er yi" || report.Threshold != 35 {
		t.Fatalf("wake word intent = %+v", report)
	}
	if report.TargetBoard != "m5stack-cores3" || report.FirmwareID != "a21-stackchan" || report.TargetProfile != "xiaozhi_esp_sr_multinet" {
		t.Fatalf("target = board:%q firmware:%q profile:%q", report.TargetBoard, report.FirmwareID, report.TargetProfile)
	}
	if report.NextRequiredConfirmation == "" || len(report.NextRequiredActions) == 0 {
		t.Fatalf("confirmation/actions missing: %+v", report)
	}
	for key, want := range map[string]any{
		"firmware_status":            "custom_pending_firmware",
		"active_runtime_profile":     "builtin_xiaozhi_wakenet",
		"desired_firmware_profile":   "custom_multinet",
		"runtime_hot_swap_supported": false,
		"custom_runtime_active":      false,
	} {
		if got := raw[key]; got != want {
			t.Fatalf("%s = %#v, want %#v in %s", key, got, want, stdout.String())
		}
	}
	if !hasWakeWordFirmwareFinding(report.Findings, "wake_word_firmware_build_required") {
		t.Fatalf("findings = %#v, want wake_word_firmware_build_required", report.Findings)
	}
	if report.ReportPath == "" || strings.Contains(report.ReportPath, string(os.PathSeparator)) {
		t.Fatalf("report path = %q, want basename only", report.ReportPath)
	}
	reportBytes, err := os.ReadFile(filepath.Join(outputDir, report.ReportPath))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	for _, output := range []string{stdout.String(), string(reportBytes)} {
		for _, forbidden := range []string{configPath, outputDir, dir, "http://", "https://", "secret", "token", "A21_WAKE_WORD_CONFIG_PATH"} {
			if strings.Contains(output, forbidden) {
				t.Fatalf("wake word firmware plan leaked %q: %s", forbidden, output)
			}
		}
	}
}

func TestRunWakeWordFirmwarePlanNoopsForBuiltinProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "a21-wake-word.json")
	if err := os.WriteFile(configPath, []byte(`{"mode":"builtin_xiaozhi","threshold":30}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"wake-word-firmware-plan", "--config", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var report wakeWordFirmwarePlanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw stdout: %v\n%s", err, stdout.String())
	}
	if report.Status != "builtin_noop" || report.Mode != "builtin_xiaozhi" {
		t.Fatalf("status/mode = %q/%q, want builtin_noop/builtin_xiaozhi", report.Status, report.Mode)
	}
	if report.FirmwareBuildRequired || report.BuildAllowed || report.FlashAllowed {
		t.Fatalf("build/flash = %v/%v/%v, want all false", report.FirmwareBuildRequired, report.BuildAllowed, report.FlashAllowed)
	}
	for key, want := range map[string]any{
		"firmware_status":            "builtin_active",
		"active_runtime_profile":     "builtin_xiaozhi_wakenet",
		"runtime_hot_swap_supported": false,
		"custom_runtime_active":      false,
	} {
		if got := raw[key]; got != want {
			t.Fatalf("%s = %#v, want %#v in %s", key, got, want, stdout.String())
		}
	}
	if _, ok := raw["desired_firmware_profile"]; ok {
		t.Fatalf("builtin noop should not declare desired firmware profile: %s", stdout.String())
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want none for builtin noop", report.Findings)
	}
}

func TestRunWakeWordFirmwarePackageWritesArtifactWithoutFlashOrPathLeaks(t *testing.T) {
	dir := t.TempDir()
	planPath := writeWakeWordFirmwarePlanFixture(t, dir, "a21-wake-word-firmware-plan-20260602-010000.json", "小阿二一", "xiao a er yi", 35)
	buildDir := writeWakeWordFirmwareBuildFixture(t, dir, "小阿二一", "xiao a er yi", 35)
	outputDir := filepath.Join(dir, "wake-artifacts")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"wake-word-firmware-package",
		"--plan", planPath,
		"--build-dir", buildDir,
		"--output-dir", outputDir,
		"--commit", "abcdef123456",
		"--timestamp", "20260602-060000",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var report wakeWordFirmwarePackageReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if report.SchemaVersion != "a21.wake_word_firmware_package.v1" ||
		report.Status != "packaged" ||
		!report.PackageWritten ||
		report.FlashAllowed ||
		report.FlashExecuted ||
		report.ProductReady {
		t.Fatalf("package report = %+v, want packaged written/no-flash/not-product-ready", report)
	}
	if report.SourcePlanReport != "a21-wake-word-firmware-plan-20260602-010000.json" ||
		report.BuildReceipt != "a21-wake-word-build.json" ||
		report.Mode != "custom_multinet" ||
		report.DesiredPhrase != "小阿二一" ||
		report.DesiredPinyin != "xiao a er yi" ||
		report.Threshold != 35 {
		t.Fatalf("package intent = %+v", report)
	}
	if report.ArtifactName != "a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-abcdef123456-20260602-060000.bin" {
		t.Fatalf("artifact name = %q", report.ArtifactName)
	}
	for _, name := range []string{report.ArtifactName, report.SHA256Name, report.ManifestName, report.ReportPath} {
		if name == "" || strings.Contains(name, string(os.PathSeparator)) {
			t.Fatalf("artifact/report name = %q, want basename only in %+v", name, report)
		}
	}
	if report.SHA256 == "" || len(report.Parts) != 5 {
		t.Fatalf("sha/parts = %q/%#v, want package hash and flash part inventory", report.SHA256, report.Parts)
	}
	for _, name := range []string{report.ArtifactName, report.SHA256Name, report.ManifestName, report.ReportPath} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected output file %s: %v", name, err)
		}
	}
	for _, output := range []string{stdout.String(), readTextFile(t, filepath.Join(outputDir, report.ManifestName)), readTextFile(t, filepath.Join(outputDir, report.ReportPath))} {
		for _, forbidden := range []string{dir, planPath, buildDir, outputDir, "http://", "https://", "secret", "token", "A21_WAKE_WORD"} {
			if strings.Contains(output, forbidden) {
				t.Fatalf("wake word firmware package leaked %q: %s", forbidden, output)
			}
		}
	}
}

func TestRunWakeWordFirmwarePackageReportsMissingBuildInputsWithoutPathLeaks(t *testing.T) {
	dir := t.TempDir()
	planPath := writeWakeWordFirmwarePlanFixture(t, dir, "a21-wake-word-firmware-plan-20260602-010000.json", "小阿二一", "xiao a er yi", 35)

	tests := []struct {
		name         string
		buildDir     string
		wantStatus   string
		wantFinding  string
		wantBuildDir string
	}{
		{
			name:         "missing build dir option",
			wantStatus:   "missing_build_dir",
			wantFinding:  "wake_word_firmware_build_dir_missing",
			wantBuildDir: "",
		},
		{
			name:         "missing build receipt",
			buildDir:     filepath.Join(dir, "xiaozhi-build-without-receipt"),
			wantStatus:   "missing_build_receipt",
			wantFinding:  "wake_word_firmware_build_receipt_missing",
			wantBuildDir: "xiaozhi-build-without-receipt",
		},
	}

	if err := os.MkdirAll(filepath.Join(dir, "xiaozhi-build-without-receipt"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := filepath.Join(dir, "diagnostic-"+strings.ReplaceAll(tt.name, " ", "-"))
			args := []string{
				"wake-word-firmware-package",
				"--plan", planPath,
				"--output-dir", outputDir,
				"--commit", "abcdef123456",
				"--timestamp", "20260602-060000",
			}
			if tt.buildDir != "" {
				args = append(args, "--build-dir", tt.buildDir)
			}

			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("code = 0, want missing build input diagnostic: stdout=%s stderr=%s", stdout.String(), stderr.String())
			}

			var report wakeWordFirmwarePackageReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("decode stdout diagnostic: %v\nstdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			if report.SchemaVersion != "a21.wake_word_firmware_package.v1" ||
				report.Status != tt.wantStatus ||
				report.PackageWritten ||
				report.FlashAllowed ||
				report.FlashExecuted ||
				report.ProductReady {
				t.Fatalf("diagnostic report = %+v, want blocked no-write/no-flash/not-product-ready", report)
			}
			if report.SourcePlanReport != "a21-wake-word-firmware-plan-20260602-010000.json" ||
				report.BuildReceipt != "a21-wake-word-build.json" ||
				report.BuildDirName != tt.wantBuildDir ||
				report.DesiredPhrase != "小阿二一" ||
				report.DesiredPinyin != "xiao a er yi" ||
				report.Threshold != 35 {
				t.Fatalf("diagnostic intent = %+v", report)
			}
			if !hasWakeWordFirmwarePackageFinding(report.Findings, tt.wantFinding) || len(report.NextRequiredActions) == 0 {
				t.Fatalf("diagnostic findings/actions = %#v / %#v, want %s and next action", report.Findings, report.NextRequiredActions, tt.wantFinding)
			}
			if report.ReportPath == "" || strings.Contains(report.ReportPath, string(os.PathSeparator)) {
				t.Fatalf("report path = %q, want basename only", report.ReportPath)
			}
			reportBytes, err := os.ReadFile(filepath.Join(outputDir, report.ReportPath))
			if err != nil {
				t.Fatalf("read diagnostic report: %v", err)
			}
			for _, output := range []string{stdout.String(), stderr.String(), string(reportBytes)} {
				for _, forbidden := range []string{dir, planPath, outputDir, tt.buildDir, "http://", "https://", "secret", "token", "A21_WAKE_WORD"} {
					if forbidden != "" && strings.Contains(output, forbidden) {
						t.Fatalf("wake word firmware package diagnostic leaked %q: %s", forbidden, output)
					}
				}
			}
		})
	}
}

func TestRunWakeWordFirmwarePackageRejectsBuildReceiptMismatch(t *testing.T) {
	dir := t.TempDir()
	planPath := writeWakeWordFirmwarePlanFixture(t, dir, "a21-wake-word-firmware-plan-20260602-010000.json", "小阿二一", "xiao a er yi", 35)
	buildDir := writeWakeWordFirmwareBuildFixture(t, dir, "其他唤醒词", "qi ta huan xing ci", 35)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"wake-word-firmware-package",
		"--plan", planPath,
		"--build-dir", buildDir,
		"--output-dir", filepath.Join(dir, "wake-artifacts"),
		"--commit", "abcdef123456",
		"--timestamp", "20260602-060000",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("code = 0, want mismatch rejection: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), planPath) || strings.Contains(stderr.String(), buildDir) || strings.Contains(stderr.String(), dir) {
		t.Fatalf("stderr leaked local path: %s", stderr.String())
	}
}

func hasWakeWordFirmwareFinding(findings []wakeWordFirmwarePlanFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func hasWakeWordFirmwarePackageFinding(findings []wakeWordFirmwarePackageFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func writeWakeWordFirmwarePlanFixture(t *testing.T, dir, name, phrase, pinyin string, threshold int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{
  "schema_version": "a21.wake_word_firmware_plan.v1",
  "status": "pending_firmware_build",
  "dry_run": true,
  "firmware_build_required": true,
  "build_allowed": false,
  "flash_allowed": false,
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "guard_tier": "T7",
  "mode": "custom_multinet",
  "active_phrase": "你好小智",
  "active_pinyin": "ni hao xiao zhi",
  "desired_phrase": %q,
  "desired_pinyin": %q,
  "threshold": %d,
  "runtime_status": "pending_firmware_build",
  "firmware_status": "custom_pending_firmware",
  "active_runtime_profile": "builtin_xiaozhi_wakenet",
  "desired_firmware_profile": "custom_multinet",
  "runtime_hot_swap_supported": false,
  "custom_runtime_active": false,
  "next_required_confirmation": "BUILD_A21_WAKE_WORD_FIRMWARE",
  "report_path": %q
}`, phrase, pinyin, threshold, name)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeWakeWordFirmwareBuildFixture(t *testing.T, dir, phrase, pinyin string, threshold int) string {
	t.Helper()
	buildDir := filepath.Join(dir, "xiaozhi-build")
	if err := os.MkdirAll(filepath.Join(buildDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	sdkconfig := `{"BOARD_TYPE_M5STACK_CORE_S3": true, "OTA_URL": "https://a21.example.com/xiaozhi/ota/"}`
	if err := os.WriteFile(filepath.Join(buildDir, "config", "sdkconfig.json"), []byte(sdkconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{
		"bootloader.bin":       "a21 bootloader",
		"partition-table.bin":  "a21 partition",
		"ota_data_initial.bin": "a21 ota",
		"xiaozhi.bin":          "a21 xiaozhi app " + phrase + " " + pinyin,
		"generated_assets.bin": "a21 assets " + phrase + " " + pinyin,
	}
	for name, content := range parts {
		if err := os.WriteFile(filepath.Join(buildDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	flashArgs := strings.Join([]string{
		"0x0 bootloader.bin",
		"0x8000 partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0x20000 xiaozhi.bin",
		"0x800000 generated_assets.bin",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(buildDir, "flash_args"), []byte(flashArgs), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt := fmt.Sprintf(`{
  "schema_version": "a21.wake_word_firmware_build.v1",
  "status": "built",
  "firmware_id": "a21-stackchan",
  "target_board": "m5stack-cores3",
  "target_profile": "xiaozhi_esp_sr_multinet",
  "mode": "custom_multinet",
  "desired_phrase": %q,
  "desired_pinyin": %q,
  "threshold": %d,
  "app_binary": "xiaozhi.bin"
}`, phrase, pinyin, threshold)
	if err := os.WriteFile(filepath.Join(buildDir, "a21-wake-word-build.json"), []byte(receipt), 0o600); err != nil {
		t.Fatal(err)
	}
	return buildDir
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
