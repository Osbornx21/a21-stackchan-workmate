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

type stackChanPhysicalEvidenceOptions struct {
	IdentityAcceptancePath string
	GatewayURL             string
	DeriveGateway          bool
	DeviceID               string
	Commit                 string
	OutputDir              string
	Passed                 map[string]string
}
type stackChanPhysicalEvidenceReport struct {
	SchemaVersion                string                                 `json:"schema_version"`
	GeneratedAtMS                int64                                  `json:"generated_at_ms,omitempty"`
	DryRun                       bool                                   `json:"dry_run,omitempty"`
	FlashAllowed                 bool                                   `json:"flash_allowed"`
	DeleteAllowed                bool                                   `json:"delete_allowed"`
	IdentityAcceptanceReportPath string                                 `json:"identity_acceptance_report_path,omitempty"`
	GatewayURL                   string                                 `json:"gateway_url,omitempty"`
	GatewayTraceID               string                                 `json:"gateway_trace_id,omitempty"`
	DeviceID                     string                                 `json:"device_id"`
	Commit                       string                                 `json:"commit"`
	ArtifactSHA256               string                                 `json:"artifact_sha256,omitempty"`
	RequiredCapabilities         []string                               `json:"required_capabilities,omitempty"`
	Observations                 []stackChanPhysicalEvidenceObservation `json:"observations"`
	ReportPath                   string                                 `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding               `json:"findings,omitempty"`
}
type stackChanPhysicalEvidenceObservation struct {
	Capability   string `json:"capability"`
	Status       string `json:"status"`
	EvidenceType string `json:"evidence_type"`
	ObservedAtMS int64  `json:"observed_at_ms,omitempty"`
}

func runStackChanPhysicalEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanPhysicalEvidenceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-physical-evidence --identity-acceptance reports/a21-stackchan-identity-acceptance-...json --device-id stackchan-001 --commit <git-sha> [--derive-gateway --gateway-url http://127.0.0.1:21080] [--pass capability=evidence_type] [--output-dir reports]")
			return 0
		case "--identity-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--identity-acceptance requires a value")
				return 2
			}
			i++
			options.IdentityAcceptancePath = args[i]
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--derive-gateway":
			options.DeriveGateway = true
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--commit":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--commit requires a value")
				return 2
			}
			i++
			options.Commit = args[i]
		case "--pass":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--pass requires capability=evidence_type")
				return 2
			}
			i++
			capability, evidenceType, err := parseStackChanPhysicalEvidencePass(args[i])
			if err != nil {
				fmt.Fprintf(stderr, "stackchan physical evidence pass invalid: %v\n", err)
				return 1
			}
			options.Passed[capability] = evidenceType
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-physical-evidence option %q\n", args[i])
			return 2
		}
	}
	if options.IdentityAcceptancePath == "" {
		fmt.Fprintln(stderr, "--identity-acceptance requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if options.Commit == "" {
		fmt.Fprintln(stderr, "--commit requires a value")
		return 2
	}
	if err := validateA21InputPath(options.IdentityAcceptancePath); err != nil {
		fmt.Fprintf(stderr, "stackchan physical evidence path invalid: %v\n", err)
		return 1
	}
	if options.DeriveGateway {
		if _, _, err := firmwareGatewayEndpoint(options.GatewayURL, "/v1/devices", nil); err != nil {
			fmt.Fprintf(stderr, "stackchan physical evidence gateway URL invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan physical evidence report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanPhysicalEvidenceReport(options)
	reportPath, err := writeStackChanPhysicalEvidenceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan physical evidence report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanPhysicalEvidence(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan physical evidence report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan physical evidence blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan physical evidence template written (no flash performed)")
	return 0
}
func defaultStackChanPhysicalEvidenceOptions() stackChanPhysicalEvidenceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	return stackChanPhysicalEvidenceOptions{
		GatewayURL: firstNonEmpty(strings.TrimSpace(os.Getenv("A21_GATEWAY_URL")), "http://127.0.0.1:21080"),
		DeviceID:   deviceID,
		Commit:     currentGitCommit(projectRoot),
		OutputDir:  "reports",
		Passed:     map[string]string{},
	}
}
func parseStackChanPhysicalEvidencePass(value string) (string, string, error) {
	if containsLegacyIdentity(value) {
		return "", "", fmt.Errorf("contains forbidden legacy identity")
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("must use capability=evidence_type")
	}
	capability := strings.TrimSpace(parts[0])
	evidenceType := strings.TrimSpace(parts[1])
	if capability == "" || evidenceType == "" {
		return "", "", fmt.Errorf("capability and evidence_type are required")
	}
	if !isRequiredStackChanCapability(capability) {
		return "", "", fmt.Errorf("unknown StackChan capability")
	}
	return capability, evidenceType, nil
}
func buildStackChanPhysicalEvidenceReport(options stackChanPhysicalEvidenceOptions) stackChanPhysicalEvidenceReport {
	nowMS := time.Now().UnixMilli()
	report := stackChanPhysicalEvidenceReport{
		SchemaVersion:                "a21.stackchan_physical_evidence.v1",
		GeneratedAtMS:                nowMS,
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		IdentityAcceptanceReportPath: options.IdentityAcceptancePath,
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		RequiredCapabilities:         append([]string(nil), requiredStackChanCapabilities...),
	}
	var identity stackChanIdentityAcceptanceReport
	if err := readJSONFile(options.IdentityAcceptancePath, &identity); err != nil {
		report.addFinding("identity_acceptance_unreadable", err.Error())
	} else {
		validateIdentityAcceptanceForPhysicalEvidence(&report, identity)
	}
	declared := map[string]string{}
	if identity.DeviceIdentity != nil {
		declared = identity.DeviceIdentity.Device.Capabilities
	}
	observed := map[string]stackChanPhysicalEvidenceObservation{}
	if options.DeriveGateway {
		deriveGatewayPhysicalEvidence(&report, options, observed)
	}
	for capability, evidenceType := range options.Passed {
		observed[capability] = stackChanPhysicalEvidenceObservation{
			Capability:   capability,
			Status:       "passed",
			EvidenceType: evidenceType,
			ObservedAtMS: nowMS,
		}
	}
	for _, capability := range requiredStackChanCapabilities {
		observation := stackChanPhysicalEvidenceObservation{
			Capability:   capability,
			Status:       "pending",
			EvidenceType: "operator_observation_required",
		}
		if derived, ok := observed[capability]; ok {
			observation = derived
		}
		if declared[capability] != "available" {
			report.addFinding("capability_not_declared_available", fmt.Sprintf("required StackChan capability %q is not declared available by the identity acceptance report", capability))
		}
		report.Observations = append(report.Observations, observation)
	}
	return report
}
func deriveGatewayPhysicalEvidence(report *stackChanPhysicalEvidenceReport, options stackChanPhysicalEvidenceOptions, observed map[string]stackChanPhysicalEvidenceObservation) {
	gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		return
	}
	report.GatewayURL = gatewayReport.GatewayURL
	device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
	if !ok {
		report.addFinding("gateway_device_missing", "expected device is missing from Gateway report")
		return
	}
	if device.RuntimeEcho["screen"] != "" {
		observed["screen"] = stackChanPhysicalEvidenceObservation{
			Capability:   "screen",
			Status:       "passed",
			EvidenceType: "device_screen_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.RuntimeEcho["servo_y"] != "" {
		observed["servo_y"] = stackChanPhysicalEvidenceObservation{
			Capability:   "servo_y",
			Status:       "passed",
			EvidenceType: "device_servo_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.RuntimeEcho["rgb"] != "" {
		observed["rgb"] = stackChanPhysicalEvidenceObservation{
			Capability:   "rgb",
			Status:       "passed",
			EvidenceType: "device_rgb_echo",
			ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
		}
	}
	if device.CurrentExpression != "" || device.CurrentMode != "" {
		if _, ok := observed["screen"]; !ok {
			observed["screen"] = stackChanPhysicalEvidenceObservation{
				Capability:   "screen",
				Status:       "passed",
				EvidenceType: "gateway_render_state",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		}
	}
	if strings.HasPrefix(device.LastEvent, "touch.") {
		switch device.LastTouchSource {
		case "screen":
			observed["screen_touch"] = stackChanPhysicalEvidenceObservation{
				Capability:   "screen_touch",
				Status:       "passed",
				EvidenceType: "gateway_touch_event",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		case "top_sensor":
			observed["top_touch"] = stackChanPhysicalEvidenceObservation{
				Capability:   "top_touch",
				Status:       "passed",
				EvidenceType: "gateway_touch_event",
				ObservedAtMS: bestObservedAtMS(device.LastSeenMS, report.GeneratedAtMS),
			}
		}
	}
	if device.LastTraceID == "" {
		return
	}
	trace, err := fetchGatewayTrace(options.GatewayURL, device.LastTraceID)
	if err != nil {
		report.addFinding("gateway_trace_fetch_failed", err.Error())
		return
	}
	report.GatewayTraceID = trace.TraceID
	for _, event := range trace.Events {
		switch event.Name {
		case "audio.frame.received":
			observed["microphone"] = stackChanPhysicalEvidenceObservation{
				Capability:   "microphone",
				Status:       "passed",
				EvidenceType: "gateway_audio_frame",
				ObservedAtMS: bestObservedAtMS(event.AtMS, report.GeneratedAtMS),
			}
		case "audio.playback.chunk.sent":
			observed["speaker"] = stackChanPhysicalEvidenceObservation{
				Capability:   "speaker",
				Status:       "passed",
				EvidenceType: "gateway_audio_downlink",
				ObservedAtMS: bestObservedAtMS(event.AtMS, report.GeneratedAtMS),
			}
		}
	}
}
func findFirmwareDeviceRecord(devices []firmwarecheck.DeviceIdentityRecord, deviceID string) (firmwarecheck.DeviceIdentityRecord, bool) {
	for _, device := range devices {
		if device.DeviceID == deviceID {
			return device, true
		}
	}
	return firmwarecheck.DeviceIdentityRecord{}, false
}
func bestObservedAtMS(candidate int64, fallback int64) int64 {
	if candidate > 0 {
		return candidate
	}
	return fallback
}
func validateIdentityAcceptanceForPhysicalEvidence(report *stackChanPhysicalEvidenceReport, identity stackChanIdentityAcceptanceReport) {
	if identity.SchemaVersion != "a21.stackchan_identity_acceptance.v1" {
		report.addFinding("identity_acceptance_schema_invalid", "identity acceptance report schema is not a21.stackchan_identity_acceptance.v1")
	}
	if identity.FlashAllowed || identity.DeleteAllowed {
		report.addFinding("unsafe_permission", "identity acceptance unexpectedly allowed flash/delete")
	}
	if identity.IdentityAcceptanceStatus != "identity_confirmed" || len(identity.Findings) > 0 {
		report.addFinding("identity_acceptance_not_confirmed", "identity acceptance report is not confirmed")
	}
	if identity.DeviceID != "" && identity.DeviceID != report.DeviceID {
		report.addFinding("identity_acceptance_device_mismatch", "identity acceptance device differs from expected device")
	}
	if identity.Commit != "" && !sameCLICommit(identity.Commit, report.Commit) {
		report.addFinding("identity_acceptance_commit_mismatch", "identity acceptance commit differs from expected commit")
	}
	if identity.ArtifactSHA256 != "" {
		report.ArtifactSHA256 = identity.ArtifactSHA256
	}
	if identity.DeviceIdentity == nil || !identity.DeviceIdentity.DeviceIdentityConfirmed {
		report.addFinding("device_identity_missing", "identity acceptance report is missing confirmed device identity")
		return
	}
	device := identity.DeviceIdentity.Device
	for _, value := range []string{device.DeviceID, device.Firmware.ID, device.Firmware.Version, device.Firmware.Board, device.Firmware.Commit} {
		if containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
	for key, value := range device.Capabilities {
		if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
			report.addFinding("identity_acceptance_legacy_identity", "identity acceptance contains forbidden legacy identity")
			return
		}
	}
}
func (report *stackChanPhysicalEvidenceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanPhysicalEvidenceReport(outputDir string, report stackChanPhysicalEvidenceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-physical-evidence-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanPhysicalEvidence(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanPhysicalEvidence(writer io.Writer, report stackChanPhysicalEvidenceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
