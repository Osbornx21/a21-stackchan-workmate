package app

import (
	"a21.local/a21/internal/firmwarecheck"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanHardwareMainlineOptions struct {
	GatewayURL string
	DeviceID   string
	OutputDir  string
}
type stackChanHardwareTrack struct {
	Capability       string `json:"capability"`
	DeclaredStatus   string `json:"declared_status,omitempty"`
	TargetStatus     string `json:"target_status"`
	Track            string `json:"track"`
	PromotionAllowed bool   `json:"promotion_allowed"`
	NextAction       string `json:"next_action"`
}
type stackChanHardwareCapabilityInvariant struct {
	Capability     string `json:"capability"`
	DeclaredStatus string `json:"declared_status,omitempty"`
	Requirement    string `json:"requirement"`
	Accepted       bool   `json:"accepted"`
}
type stackChanHardwareCapabilityInvariantSpec struct {
	Capability      string
	Requirement     string
	AllowedExact    []string
	AllowedPrefixes []string
}
type stackChanHardwareMainlineReport struct {
	SchemaVersion          string                                 `json:"schema_version"`
	GeneratedAtMS          int64                                  `json:"generated_at_ms"`
	Metadata               latencyBenchMetadata                   `json:"metadata"`
	DryRun                 bool                                   `json:"dry_run"`
	FlashAllowed           bool                                   `json:"flash_allowed"`
	DeleteAllowed          bool                                   `json:"delete_allowed"`
	HardwareMainlineStatus string                                 `json:"hardware_mainline_status"`
	GatewayURL             string                                 `json:"gateway_url"`
	DeviceID               string                                 `json:"device_id"`
	Firmware               firmwarecheck.DeviceIdentityFirmware   `json:"firmware"`
	Capabilities           map[string]string                      `json:"capabilities,omitempty"`
	RuntimeEcho            map[string]string                      `json:"runtime_echo,omitempty"`
	CapabilityInvariants   []stackChanHardwareCapabilityInvariant `json:"capability_invariants"`
	OrderedTracks          []stackChanHardwareTrack               `json:"ordered_tracks"`
	NextRequiredActions    []string                               `json:"next_required_actions"`
	ReportPath             string                                 `json:"report_path,omitempty"`
	Findings               []officePreflightFinding               `json:"findings,omitempty"`
}

var stackChanHardwareMainlineTracks = []stackChanHardwareTrack{
	{
		Capability:       "imu",
		TargetStatus:     "planned_9_axis_imu",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add posture, bump, pickup, and safe expression transition telemetry before any product promotion.",
	},
	{
		Capability:       "ambient_light",
		TargetStatus:     "planned_ambient_light_sensor",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add adaptive brightness telemetry and acceptance evidence without changing expression behavior first.",
	},
	{
		Capability:       "proximity",
		TargetStatus:     "planned_proximity_sensor",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add wake, sleep, and interaction-distance telemetry with false-positive protection.",
	},
	{
		Capability:       "battery",
		TargetStatus:     "planned_550mah_battery",
		Track:            "read_only_diagnostic_probe",
		PromotionAllowed: false,
		NextAction:       "Add honest power-state reporting and low-power local fallback evidence.",
	},
	{
		Capability:       "servo_x",
		TargetStatus:     "planned_continuous_rotation_axis",
		Track:            "motion_safety_spike",
		PromotionAllowed: false,
		NextAction:       "Prove second-axis range, clamps, and coordinated head pose before enabling motion.",
	},
	{
		Capability:       "camera",
		TargetStatus:     "planned_core_s3_camera",
		Track:            "privacy_safe_vision_spike",
		PromotionAllowed: false,
		NextAction:       "Design explicit opt-in local presence or gesture context with no silent surveillance.",
	},
	{
		Capability:       "nfc",
		TargetStatus:     "planned_nfc",
		Track:            "explicit_opt_in_interaction_spike",
		PromotionAllowed: false,
		NextAction:       "Define visible consent and desk interaction semantics before runtime use.",
	},
	{
		Capability:       "infrared",
		TargetStatus:     "planned_infrared_tx_rx",
		Track:            "explicit_opt_in_interaction_spike",
		PromotionAllowed: false,
		NextAction:       "Define explicit opt-in remote interaction semantics before runtime use.",
	},
}
var stackChanHardwareCapabilityInvariantSpecs = []stackChanHardwareCapabilityInvariantSpec{
	{
		Capability:      "microphone",
		Requirement:     "release microphone remains disabled or diagnostic-only until separate production evidence exists",
		AllowedPrefixes: []string{"disabled_", "diagnostic_"},
	},
	{Capability: "speaker", Requirement: "stable expression/playback surface must be available", AllowedExact: []string{"available"}},
	{Capability: "screen", Requirement: "stable expression/playback surface must be available", AllowedExact: []string{"available"}},
	{Capability: "screen_touch", Requirement: "stable touch surface must be available", AllowedExact: []string{"available"}},
	{Capability: "top_touch", Requirement: "stable touch surface must be available", AllowedExact: []string{"available"}},
	{Capability: "servo_y", Requirement: "stable Y-axis expression surface must be available", AllowedExact: []string{"available"}},
	{Capability: "rgb", Requirement: "stable RGB expression surface must be available", AllowedExact: []string{"available"}},
	{
		Capability:      "imu",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "ambient_light",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "proximity",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "battery",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "servo_x",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "camera",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "nfc",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
	{
		Capability:      "infrared",
		Requirement:     "planned hardware must stay planned, disabled, or diagnostic-only; product availability needs separate evidence",
		AllowedPrefixes: []string{"planned_", "disabled_", "diagnostic_"},
	},
}

func runStackChanHardwareMainline(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanHardwareMainlineOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 [--output-dir reports]")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-hardware-mainline option %q\n", args[i])
			return 2
		}
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if containsLegacyIdentity(options.DeviceID) {
		fmt.Fprintln(stderr, "device id contains forbidden legacy identity")
		return 1
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan hardware mainline report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanHardwareMainlineReport(options)
	if options.OutputDir != "" {
		reportPath, err := writeStackChanHardwareMainlineReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write stackchan hardware mainline report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanHardwareMainline(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan hardware mainline report: %v\n", err)
		return 1
	}
	if report.HardwareMainlineStatus != "ready_for_diagnostic_spikes" {
		fmt.Fprintln(stdout, "stackchan hardware mainline blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan hardware mainline ready (no flash performed)")
	return 0
}
func defaultStackChanHardwareMainlineOptions() stackChanHardwareMainlineOptions {
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanHardwareMainlineOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   deviceID,
		OutputDir:  "reports",
	}
}
func buildStackChanHardwareMainlineReport(options stackChanHardwareMainlineOptions) stackChanHardwareMainlineReport {
	report := stackChanHardwareMainlineReport{
		SchemaVersion:          "a21.stackchan_hardware_mainline.v1",
		GeneratedAtMS:          time.Now().UnixMilli(),
		Metadata:               buildLatencyBenchMetadata(),
		DryRun:                 true,
		FlashAllowed:           false,
		DeleteAllowed:          false,
		HardwareMainlineStatus: "blocked",
		GatewayURL:             sanitizedOfficeGatewayURL(options.GatewayURL),
		DeviceID:               options.DeviceID,
	}
	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		report.NextRequiredActions = []string{"Start the A21 Gateway, keep LAN traffic direct, and rerun stackchan-hardware-mainline before touching firmware."}
		return report
	}
	device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("device_missing", "expected StackChan device is missing from Gateway")
		report.NextRequiredActions = []string{"Connect the intended A21 StackChan device to Gateway and rerun stackchan-hardware-mainline."}
		return report
	}
	report.Firmware = device.Firmware
	report.Capabilities = cloneStringMap(device.Capabilities)
	report.RuntimeEcho = cloneStringMap(device.RuntimeEcho)
	for _, spec := range stackChanHardwareCapabilityInvariantSpecs {
		declaredStatus := strings.TrimSpace(report.Capabilities[spec.Capability])
		invariant := stackChanHardwareCapabilityInvariant{
			Capability:     spec.Capability,
			DeclaredStatus: declaredStatus,
			Requirement:    spec.Requirement,
			Accepted:       stackChanHardwareCapabilityStatusAccepted(declaredStatus, spec),
		}
		if declaredStatus == "" {
			report.addFinding("capability_missing", fmt.Sprintf("StackChan capability %q must be declared before the hardware mainline can proceed", spec.Capability))
		} else if !invariant.Accepted {
			report.addFinding("capability_status_not_honest", fmt.Sprintf("StackChan capability %q declared status %q violates hardware mainline invariant", spec.Capability, declaredStatus))
		}
		report.CapabilityInvariants = append(report.CapabilityInvariants, invariant)
	}
	for _, spec := range stackChanHardwareMainlineTracks {
		track := spec
		declaredStatus, ok := report.Capabilities[track.Capability]
		if !ok || strings.TrimSpace(declaredStatus) == "" {
			report.addFinding("capability_missing", fmt.Sprintf("StackChan capability %q must be declared before this hardware track can proceed", track.Capability))
			track.DeclaredStatus = ""
		} else {
			track.DeclaredStatus = declaredStatus
		}
		report.OrderedTracks = append(report.OrderedTracks, track)
	}
	report.NextRequiredActions = []string{
		"Run hardware tracks in ordered_tracks order and keep unavailable capabilities planned until instrumented evidence exists.",
		"Add firmware diagnostics as separate guarded builds before promoting any planned capability to product firmware.",
		"Preserve A21 package identity, artifact checks, and no raw upload discipline for every hardware slice.",
	}
	if len(report.Findings) == 0 {
		report.HardwareMainlineStatus = "ready_for_diagnostic_spikes"
	}
	return report
}
func stackChanHardwareCapabilityStatusAccepted(status string, spec stackChanHardwareCapabilityInvariantSpec) bool {
	status = strings.TrimSpace(status)
	if status == "" {
		return false
	}
	for _, exact := range spec.AllowedExact {
		if status == exact {
			return true
		}
	}
	for _, prefix := range spec.AllowedPrefixes {
		if strings.HasPrefix(status, prefix) {
			return true
		}
	}
	return false
}
func (report *stackChanHardwareMainlineReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanHardwareMainlineReport(outputDir string, report stackChanHardwareMainlineReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-hardware-mainline-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanHardwareMainline(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanHardwareMainline(writer io.Writer, report stackChanHardwareMainlineReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
