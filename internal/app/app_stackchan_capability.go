package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type stackChanCapabilityAcceptanceOptions struct {
	IdentityAcceptancePath string
	EvidencePath           string
	DeviceID               string
	Commit                 string
	OutputDir              string
}
type stackChanCapabilityResult struct {
	Capability     string `json:"capability"`
	DeclaredStatus string `json:"declared_status,omitempty"`
	EvidenceStatus string `json:"evidence_status,omitempty"`
	EvidenceType   string `json:"evidence_type,omitempty"`
	ObservedAtMS   int64  `json:"observed_at_ms,omitempty"`
	Accepted       bool   `json:"accepted"`
}
type stackChanCapabilityAcceptanceReport struct {
	SchemaVersion                string                      `json:"schema_version"`
	GeneratedAtMS                int64                       `json:"generated_at_ms"`
	Metadata                     latencyBenchMetadata        `json:"metadata"`
	DryRun                       bool                        `json:"dry_run"`
	FlashAllowed                 bool                        `json:"flash_allowed"`
	DeleteAllowed                bool                        `json:"delete_allowed"`
	HardwareAcceptanceScope      string                      `json:"hardware_acceptance_scope"`
	CapabilityAcceptanceStatus   string                      `json:"capability_acceptance_status"`
	IdentityAcceptanceReportPath string                      `json:"identity_acceptance_report_path"`
	EvidenceReportPath           string                      `json:"evidence_report_path"`
	DeviceID                     string                      `json:"device_id"`
	Commit                       string                      `json:"commit"`
	ArtifactPath                 string                      `json:"artifact_path,omitempty"`
	ArtifactSHA256               string                      `json:"artifact_sha256,omitempty"`
	RequiredCapabilities         []string                    `json:"required_capabilities"`
	CapabilityResults            []stackChanCapabilityResult `json:"capability_results"`
	NextRequiredActions          []string                    `json:"next_required_actions"`
	ReportPath                   string                      `json:"report_path,omitempty"`
	Findings                     []officePreflightFinding    `json:"findings,omitempty"`
}

func runStackChanCapabilityAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanCapabilityAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-capability-acceptance --identity-acceptance reports/a21-stackchan-identity-acceptance-...json --evidence reports/a21-stackchan-physical-evidence.json --device-id stackchan-001 --commit <git-sha> [--output-dir reports]")
			return 0
		case "--identity-acceptance":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--identity-acceptance requires a value")
				return 2
			}
			i++
			options.IdentityAcceptancePath = args[i]
		case "--evidence":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--evidence requires a value")
				return 2
			}
			i++
			options.EvidencePath = args[i]
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
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-capability-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.IdentityAcceptancePath == "" {
		fmt.Fprintln(stderr, "--identity-acceptance requires a value")
		return 2
	}
	if options.EvidencePath == "" {
		fmt.Fprintln(stderr, "--evidence requires a value")
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
	for _, path := range []string{options.IdentityAcceptancePath, options.EvidencePath} {
		if err := validateA21InputPath(path); err != nil {
			fmt.Fprintf(stderr, "stackchan capability acceptance path invalid: %v\n", err)
			return 1
		}
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan capability acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := buildStackChanCapabilityAcceptanceReport(options)
	reportPath, err := writeStackChanCapabilityAcceptanceReport(options.OutputDir, report)
	if err != nil {
		fmt.Fprintf(stderr, "write stackchan capability acceptance report: %v\n", err)
		return 1
	}
	report.ReportPath = reportPath
	if err := writeJSONStackChanCapabilityAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan capability acceptance report: %v\n", err)
		return 1
	}
	if len(report.Findings) > 0 {
		fmt.Fprintln(stdout, "stackchan capability acceptance failed (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan capability acceptance ok (no flash performed)")
	return 0
}
func defaultStackChanCapabilityAcceptanceOptions() stackChanCapabilityAcceptanceOptions {
	cwd, _ := os.Getwd()
	projectRoot := findProjectRoot(cwd)
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	return stackChanCapabilityAcceptanceOptions{
		DeviceID:  deviceID,
		Commit:    currentGitCommit(projectRoot),
		OutputDir: "reports",
	}
}
func buildStackChanCapabilityAcceptanceReport(options stackChanCapabilityAcceptanceOptions) stackChanCapabilityAcceptanceReport {
	report := stackChanCapabilityAcceptanceReport{
		SchemaVersion:                "a21.stackchan_capability_acceptance.v1",
		GeneratedAtMS:                time.Now().UnixMilli(),
		Metadata:                     buildLatencyBenchMetadata(),
		DryRun:                       true,
		FlashAllowed:                 false,
		DeleteAllowed:                false,
		HardwareAcceptanceScope:      "physical_capability_evidence",
		CapabilityAcceptanceStatus:   "confirmed",
		IdentityAcceptanceReportPath: options.IdentityAcceptancePath,
		EvidenceReportPath:           options.EvidencePath,
		DeviceID:                     options.DeviceID,
		Commit:                       options.Commit,
		RequiredCapabilities:         append([]string(nil), requiredStackChanCapabilities...),
		NextRequiredActions: []string{
			"Keep this report with identity acceptance, physical evidence, firmware artifact, and office acceptance receipts.",
			"Capability acceptance is evidence-gated and still does not enable flashing.",
			"Run latency and full-duplex physical acceptance separately before claiming the complete A21 voice experience.",
		},
	}

	var identity stackChanIdentityAcceptanceReport
	if err := readJSONFile(options.IdentityAcceptancePath, &identity); err != nil {
		report.addFinding("identity_acceptance_unreadable", err.Error())
	} else {
		validateIdentityAcceptanceForStackChanCapability(&report, identity)
	}

	var evidence stackChanPhysicalEvidenceReport
	if err := readJSONFile(options.EvidencePath, &evidence); err != nil {
		report.addFinding("physical_evidence_unreadable", err.Error())
	} else {
		validatePhysicalEvidenceForStackChanCapability(&report, identity, evidence)
	}

	if len(report.Findings) > 0 {
		report.CapabilityAcceptanceStatus = "blocked"
	}
	return report
}
func validateIdentityAcceptanceForStackChanCapability(report *stackChanCapabilityAcceptanceReport, identity stackChanIdentityAcceptanceReport) {
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
	if identity.ArtifactPath != "" {
		report.ArtifactPath = identity.ArtifactPath
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
func validatePhysicalEvidenceForStackChanCapability(report *stackChanCapabilityAcceptanceReport, identity stackChanIdentityAcceptanceReport, evidence stackChanPhysicalEvidenceReport) {
	if evidence.SchemaVersion != "a21.stackchan_physical_evidence.v1" {
		report.addFinding("physical_evidence_schema_invalid", "physical evidence report schema is not a21.stackchan_physical_evidence.v1")
	}
	if evidence.FlashAllowed || evidence.DeleteAllowed {
		report.addFinding("unsafe_permission", "physical evidence unexpectedly allowed flash/delete")
	}
	if evidence.DeviceID != "" && evidence.DeviceID != report.DeviceID {
		report.addFinding("physical_evidence_device_mismatch", "physical evidence device differs from expected device")
	}
	if evidence.Commit != "" && !sameCLICommit(evidence.Commit, report.Commit) {
		report.addFinding("physical_evidence_commit_mismatch", "physical evidence commit differs from expected commit")
	}
	if evidence.ArtifactSHA256 != "" && report.ArtifactSHA256 != "" && !strings.EqualFold(evidence.ArtifactSHA256, report.ArtifactSHA256) {
		report.addFinding("physical_evidence_artifact_sha_mismatch", "physical evidence artifact sha256 differs from identity acceptance")
	}

	observations := map[string]stackChanPhysicalEvidenceObservation{}
	for _, observation := range evidence.Observations {
		if containsLegacyIdentity(observation.Capability) || containsLegacyIdentity(observation.Status) || containsLegacyIdentity(observation.EvidenceType) {
			report.addFinding("physical_evidence_legacy_identity", "physical evidence contains forbidden legacy identity")
			continue
		}
		capability := strings.TrimSpace(observation.Capability)
		if capability == "" {
			report.addFinding("capability_evidence_invalid", "physical evidence contains an observation without a capability")
			continue
		}
		if _, exists := observations[capability]; !exists || observation.Status == "passed" {
			observations[capability] = observation
		}
	}

	declared := map[string]string{}
	if identity.DeviceIdentity != nil {
		declared = identity.DeviceIdentity.Device.Capabilities
	}
	for _, capability := range requiredStackChanCapabilities {
		result := stackChanCapabilityResult{
			Capability:     capability,
			DeclaredStatus: declared[capability],
		}
		if result.DeclaredStatus != "available" {
			report.addFinding("capability_not_declared_available", fmt.Sprintf("required StackChan capability %q is not declared available by the fresh device identity", capability))
		}
		observation, ok := observations[capability]
		if !ok {
			report.addFinding("capability_evidence_missing", fmt.Sprintf("physical evidence missing for required StackChan capability %q", capability))
			report.CapabilityResults = append(report.CapabilityResults, result)
			continue
		}
		result.EvidenceStatus = observation.Status
		result.EvidenceType = observation.EvidenceType
		result.ObservedAtMS = observation.ObservedAtMS
		if observation.Status != "passed" {
			report.addFinding("capability_evidence_not_passed", fmt.Sprintf("physical evidence for required StackChan capability %q is not passed", capability))
		}
		if strings.TrimSpace(observation.EvidenceType) == "" {
			report.addFinding("capability_evidence_type_missing", fmt.Sprintf("physical evidence for required StackChan capability %q is missing evidence_type", capability))
		}
		if observation.ObservedAtMS <= 0 {
			report.addFinding("capability_evidence_time_missing", fmt.Sprintf("physical evidence for required StackChan capability %q is missing observed_at_ms", capability))
		}
		result.Accepted = result.DeclaredStatus == "available" &&
			result.EvidenceStatus == "passed" &&
			strings.TrimSpace(result.EvidenceType) != "" &&
			result.ObservedAtMS > 0
		report.CapabilityResults = append(report.CapabilityResults, result)
	}
}
func (report *stackChanCapabilityAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func writeStackChanCapabilityAcceptanceReport(outputDir string, report stackChanCapabilityAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-capability-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanCapabilityAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanCapabilityAcceptance(writer io.Writer, report stackChanCapabilityAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
