package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"a21.local/a21/internal/gateway"
)

const (
	wakeWordFirmwarePlanSchema       = "a21.wake_word_firmware_plan.v1"
	wakeWordFirmwareBuildConfirm     = "BUILD_A21_WAKE_WORD_FIRMWARE"
	wakeWordFirmwareTargetBoard      = "m5stack-cores3"
	wakeWordFirmwareID               = "a21-stackchan"
	wakeWordFirmwareTargetProfile    = "xiaozhi_esp_sr_multinet"
	wakeWordFirmwareGuardTier        = "T7"
	wakeWordFirmwareCustomMode       = "custom_multinet"
	wakeWordFirmwareBuiltinMode      = "builtin_xiaozhi"
	wakeWordFirmwareBuildFindingCode = "wake_word_firmware_build_required"
)

type wakeWordFirmwarePlanOptions struct {
	ConfigPath string
	OutputDir  string
}

type wakeWordFirmwarePlanReport struct {
	SchemaVersion            string                        `json:"schema_version"`
	GeneratedAtMS            int64                         `json:"generated_at_ms"`
	Status                   string                        `json:"status"`
	DryRun                   bool                          `json:"dry_run"`
	FirmwareBuildRequired    bool                          `json:"firmware_build_required"`
	BuildAllowed             bool                          `json:"build_allowed"`
	FlashAllowed             bool                          `json:"flash_allowed"`
	FirmwareID               string                        `json:"firmware_id"`
	TargetBoard              string                        `json:"target_board"`
	TargetProfile            string                        `json:"target_profile"`
	GuardTier                string                        `json:"guard_tier"`
	Mode                     string                        `json:"mode"`
	ActivePhrase             string                        `json:"active_phrase"`
	ActivePinyin             string                        `json:"active_pinyin"`
	DesiredPhrase            string                        `json:"desired_phrase,omitempty"`
	DesiredPinyin            string                        `json:"desired_pinyin,omitempty"`
	Threshold                int                           `json:"threshold"`
	RuntimeStatus            string                        `json:"runtime_status"`
	RuntimeConfigurable      bool                          `json:"runtime_configurable"`
	NextRequiredConfirmation string                        `json:"next_required_confirmation,omitempty"`
	NextRequiredActions      []string                      `json:"next_required_actions,omitempty"`
	Findings                 []wakeWordFirmwarePlanFinding `json:"findings,omitempty"`
	ReportPath               string                        `json:"report_path,omitempty"`
}

type wakeWordFirmwarePlanFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func runWakeWordFirmwarePlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := wakeWordFirmwarePlanOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 wake-word-firmware-plan [--config .a21-run/gateway/a21-wake-word.json] [--output-dir reports]")
			return 0
		case "--config":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--config requires a value")
				return 2
			}
			i++
			options.ConfigPath = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown wake-word-firmware-plan option %q\n", args[i])
			return 2
		}
	}
	if strings.TrimSpace(options.ConfigPath) != "" {
		if err := validateA21InputPath(options.ConfigPath); err != nil {
			fmt.Fprintln(stderr, "wake word firmware plan config path invalid")
			return 1
		}
	}
	if strings.TrimSpace(options.OutputDir) != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintln(stderr, "wake word firmware plan report dir invalid")
			return 1
		}
	}
	status, err := gateway.LoadWakeWordConfigStatus(options.ConfigPath)
	if err != nil {
		fmt.Fprintln(stderr, "wake word firmware plan failed: wake word config invalid or unavailable")
		return 1
	}
	report := buildWakeWordFirmwarePlanReport(status)
	if strings.TrimSpace(options.OutputDir) != "" {
		if err := writeWakeWordFirmwarePlanReport(options.OutputDir, &report); err != nil {
			fmt.Fprintf(stderr, "write wake word firmware plan report: %v\n", err)
			return 1
		}
	}
	if err := writeJSONWakeWordFirmwarePlan(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode wake word firmware plan: %v\n", err)
		return 1
	}
	return 0
}

func buildWakeWordFirmwarePlanReport(status gateway.WakeWordConfigResponse) wakeWordFirmwarePlanReport {
	report := wakeWordFirmwarePlanReport{
		SchemaVersion:       wakeWordFirmwarePlanSchema,
		GeneratedAtMS:       time.Now().UnixMilli(),
		Status:              "builtin_noop",
		DryRun:              true,
		FirmwareID:          wakeWordFirmwareID,
		TargetBoard:         wakeWordFirmwareTargetBoard,
		TargetProfile:       wakeWordFirmwareTargetProfile,
		GuardTier:           wakeWordFirmwareGuardTier,
		Mode:                status.Mode,
		ActivePhrase:        status.ActivePhrase,
		ActivePinyin:        status.ActivePinyin,
		DesiredPhrase:       status.DesiredPhrase,
		DesiredPinyin:       status.DesiredPinyin,
		Threshold:           status.Threshold,
		RuntimeStatus:       status.RuntimeStatus,
		RuntimeConfigurable: status.RuntimeConfigurable,
		NextRequiredActions: []string{
			"Keep stock xiaozhi WakeNet active; no A21 firmware build is required for the builtin profile.",
		},
	}
	if report.Mode == "" {
		report.Mode = wakeWordFirmwareBuiltinMode
	}
	if status.FirmwareBuildRequired || status.Mode == wakeWordFirmwareCustomMode {
		report.Status = "pending_firmware_build"
		report.FirmwareBuildRequired = true
		report.NextRequiredConfirmation = wakeWordFirmwareBuildConfirm
		report.NextRequiredActions = []string{
			"Prepare a reviewed A21 xiaozhi/ESP-SR MultiNet firmware build package for the requested wake word.",
			"Run the guarded firmware flash plan with explicit board, artifact, USB serial target, and current device report in a hardware window.",
			"Only execute flashing after the T7 foreground guard, confirmation value, and physical acceptance checklist pass.",
		}
		report.Findings = []wakeWordFirmwarePlanFinding{{
			Code:     wakeWordFirmwareBuildFindingCode,
			Severity: "warning",
			Message:  "Custom wake words require a dedicated guarded A21 xiaozhi/ESP-SR MultiNet firmware build before launch readiness can pass.",
		}}
	}
	return report
}

func writeWakeWordFirmwarePlanReport(outputDir string, report *wakeWordFirmwarePlanReport) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-wake-word-firmware-plan-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	return writeJSONWakeWordFirmwarePlan(file, *report)
}

func writeJSONWakeWordFirmwarePlan(writer io.Writer, report wakeWordFirmwarePlanReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
