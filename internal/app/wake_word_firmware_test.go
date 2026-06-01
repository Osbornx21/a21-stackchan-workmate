package app

import (
	"bytes"
	"encoding/json"
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
	if report.Status != "builtin_noop" || report.Mode != "builtin_xiaozhi" {
		t.Fatalf("status/mode = %q/%q, want builtin_noop/builtin_xiaozhi", report.Status, report.Mode)
	}
	if report.FirmwareBuildRequired || report.BuildAllowed || report.FlashAllowed {
		t.Fatalf("build/flash = %v/%v/%v, want all false", report.FirmwareBuildRequired, report.BuildAllowed, report.FlashAllowed)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want none for builtin noop", report.Findings)
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
