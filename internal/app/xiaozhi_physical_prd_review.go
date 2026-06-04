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

const xiaozhiPhysicalPRDReviewConfirm = "ACCEPT_A21_XIAOZHI_PHYSICAL_PRD"

type xiaozhiPhysicalPRDReviewOptions struct {
	PhysicalEvidenceReport string
	HalfDuplexReport       string
	OutputDir              string
	Confirm                string
}

func runXiaozhiPhysicalPRDReview(args []string, stdout io.Writer, stderr io.Writer) int {
	options := xiaozhiPhysicalPRDReviewOptions{
		OutputDir: "reports",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintf(stdout, "a21 xiaozhi-physical-prd-review --physical-evidence-report report.json --half-duplex-report report.json --confirm %s [--output-dir reports]\n", xiaozhiPhysicalPRDReviewConfirm)
			return 0
		case "--physical-evidence-report":
			if !readStringOption(args, &i, stderr, "--physical-evidence-report", &options.PhysicalEvidenceReport) {
				return 2
			}
		case "--half-duplex-report":
			if !readStringOption(args, &i, stderr, "--half-duplex-report", &options.HalfDuplexReport) {
				return 2
			}
		case "--confirm":
			if !readStringOption(args, &i, stderr, "--confirm", &options.Confirm) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown xiaozhi-physical-prd-review option %q\n", args[i])
			return 2
		}
	}
	if strings.TrimSpace(options.Confirm) != xiaozhiPhysicalPRDReviewConfirm {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review requires --confirm %s\n", xiaozhiPhysicalPRDReviewConfirm)
		return 2
	}
	if err := validateXiaozhiPhysicalPRDReviewOptions(options); err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review option invalid: %v\n", err)
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review report dir invalid: %v\n", err)
		return 1
	}
	physical, err := loadXiaozhiPhysicalEvidenceReviewReport(options.PhysicalEvidenceReport)
	if err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review physical report invalid: %v\n", err)
		return 1
	}
	halfDuplex, err := loadXiaozhiHalfDuplexReviewReport(options.HalfDuplexReport)
	if err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review half-duplex report invalid: %v\n", err)
		return 1
	}
	report, err := buildXiaozhiPhysicalPRDAcceptedReport(options, physical, halfDuplex)
	if err != nil {
		fmt.Fprintf(stderr, "xiaozhi physical PRD review blocked: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		reportPath, err := writeXiaozhiPhysicalEvidenceReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintln(stderr, "xiaozhi physical PRD review report write failed")
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONXiaozhiPhysicalEvidence(stdout, report); err != nil {
		fmt.Fprintln(stderr, "xiaozhi physical PRD review report encode failed")
		return 1
	}
	return 0
}

func validateXiaozhiPhysicalPRDReviewOptions(options xiaozhiPhysicalPRDReviewOptions) error {
	for name, path := range map[string]string{
		"physical_evidence_report": options.PhysicalEvidenceReport,
		"half_duplex_report":       options.HalfDuplexReport,
	} {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("%s is required", name)
		}
		if err := validateA21InputPath(path); err != nil {
			return err
		}
	}
	return nil
}

func loadXiaozhiPhysicalEvidenceReviewReport(path string) (xiaozhiPhysicalEvidenceReport, error) {
	data, err := readSafeXiaozhiReviewReportBytes(path)
	if err != nil {
		return xiaozhiPhysicalEvidenceReport{}, err
	}
	var report xiaozhiPhysicalEvidenceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("invalid json")
	}
	if report.SchemaVersion != xiaozhiPhysicalEvidenceSchemaVersion {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("unexpected schema")
	}
	if !productPhysicalStackChanRedactionOK(report.Redaction) {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("redaction contract failed")
	}
	return report, nil
}

func loadXiaozhiHalfDuplexReviewReport(path string) (xiaozhiHalfDuplexAcceptanceReport, error) {
	data, err := readBoundedSingleJSONReportBytes(path)
	if err != nil {
		return xiaozhiHalfDuplexAcceptanceReport{}, err
	}
	var report xiaozhiHalfDuplexAcceptanceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return xiaozhiHalfDuplexAcceptanceReport{}, fmt.Errorf("invalid json")
	}
	if report.SchemaVersion != xiaozhiHalfDuplexAcceptanceSchemaVersion {
		return xiaozhiHalfDuplexAcceptanceReport{}, fmt.Errorf("unexpected schema")
	}
	if !productPhysicalStackChanRedactionOK(report.Redaction) {
		return xiaozhiHalfDuplexAcceptanceReport{}, fmt.Errorf("redaction contract failed")
	}
	for name, value := range map[string]string{
		"device_id":  report.DeviceID,
		"trace_id":   report.TraceID,
		"session_id": report.SessionID,
		"profile":    report.Profile,
	} {
		if !xiaozhiPhysicalSafeID(value) {
			return xiaozhiHalfDuplexAcceptanceReport{}, fmt.Errorf("%s is invalid or unsafe", name)
		}
	}
	return report, nil
}

func readSafeXiaozhiReviewReportBytes(path string) ([]byte, error) {
	data, err := readBoundedSingleJSONReportBytes(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid json")
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || physicalStackChanValueUnsafe(raw) {
		return nil, fmt.Errorf("unsafe report content")
	}
	return data, nil
}

func readBoundedSingleJSONReportBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return nil, fmt.Errorf("report unreadable or too large")
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid json")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("invalid json")
	}
	return data, nil
}

func buildXiaozhiPhysicalPRDAcceptedReport(options xiaozhiPhysicalPRDReviewOptions, physical xiaozhiPhysicalEvidenceReport, halfDuplex xiaozhiHalfDuplexAcceptanceReport) (xiaozhiPhysicalEvidenceReport, error) {
	if !xiaozhiPhysicalReportsMatch(physical, halfDuplex) {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("physical and half-duplex reports target different evidence")
	}
	if physical.ExecutionMode != "physical_xiaozhi_gateway" ||
		physical.PromotionGate != "candidate" ||
		physical.AcceptanceStatus != "physical_review_required" ||
		physical.PRDAccepted {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("physical evidence is not a reviewable Xiaozhi physical candidate")
	}
	if halfDuplex.HalfDuplexAcceptanceStatus != "physical_review_required" ||
		halfDuplex.PRDAccepted ||
		len(halfDuplex.Findings) > 0 {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("half-duplex evidence is not ready for PRD review")
	}
	if !xiaozhiPhysicalHalfDuplexSummaryMatches(physical, halfDuplex.SourcePhysicalEvidenceSummary) {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("half-duplex report does not summarize the supplied physical evidence candidate")
	}
	if hasPhysicalStackChanErrorFinding(physical.Findings) {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("physical evidence still has error findings")
	}
	readiness := buildProductXiaozhiPhysicalReadiness(filepath.Base(filepath.Clean(options.PhysicalEvidenceReport)), physical)
	if !readiness.GatewayDownlinkPhysicalDeviceEvidence ||
		!readiness.RequiredPhysicalMetricsAvailable ||
		!readiness.MicEvidenceAvailable ||
		!readiness.OperatorInstrumentObservationAvailable ||
		!readiness.CandidatePhysicalVoiceEvidence {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("physical evidence is missing PRD-required metrics")
	}
	if !halfDuplex.PhysicalDeviceOnline ||
		!halfDuplex.MicAvailable ||
		!halfDuplex.DownlinkAvailable ||
		!halfDuplex.PlaybackAckAvailable ||
		!halfDuplex.BargeInDetectedAvailable ||
		!halfDuplex.BargeInStopAvailable ||
		!halfDuplex.BargeInStopDoneAvailable ||
		!halfDuplex.PhysicalSoundObserved {
		return xiaozhiPhysicalEvidenceReport{}, fmt.Errorf("half-duplex evidence is missing physical review signals")
	}
	accepted := physical
	accepted.GeneratedAtMS = time.Now().UnixMilli()
	accepted.PromotionGate = "accepted"
	accepted.AcceptanceStatus = "prd_accepted"
	accepted.PRDAccepted = true
	accepted.ReportPath = ""
	accepted.Findings = append(filterAcceptedXiaozhiPhysicalFindings(accepted.Findings), physicalStackChanFinding("xiaozhi_physical_prd_accepted", "info", "Xiaozhi physical evidence satisfies PRD acceptance after physical review"))
	return accepted, nil
}

func xiaozhiPhysicalReportsMatch(physical xiaozhiPhysicalEvidenceReport, halfDuplex xiaozhiHalfDuplexAcceptanceReport) bool {
	return strings.TrimSpace(physical.DeviceID) == strings.TrimSpace(halfDuplex.DeviceID) &&
		strings.TrimSpace(physical.TraceID) == strings.TrimSpace(halfDuplex.TraceID) &&
		strings.TrimSpace(physical.SessionID) == strings.TrimSpace(halfDuplex.SessionID) &&
		strings.TrimSpace(physical.Profile) == strings.TrimSpace(halfDuplex.Profile)
}

func xiaozhiPhysicalHalfDuplexSummaryMatches(physical xiaozhiPhysicalEvidenceReport, summary xiaozhiPhysicalEvidenceSummary) bool {
	return summary.SchemaVersion == physical.SchemaVersion &&
		summary.AcceptanceStatus == physical.AcceptanceStatus &&
		summary.PromotionGate == physical.PromotionGate &&
		summary.PRDAccepted == physical.PRDAccepted
}

func hasPhysicalStackChanErrorFinding(findings []physicalStackChanEvidenceFinding) bool {
	for _, finding := range findings {
		if strings.TrimSpace(finding.Severity) == "error" {
			return true
		}
	}
	return false
}

func filterAcceptedXiaozhiPhysicalFindings(findings []physicalStackChanEvidenceFinding) []physicalStackChanEvidenceFinding {
	filtered := make([]physicalStackChanEvidenceFinding, 0, len(findings))
	for _, finding := range findings {
		switch strings.TrimSpace(finding.Code) {
		case "xiaozhi_physical_gateway_downlink_candidate", "xiaozhi_physical_prd_accepted":
			continue
		default:
			filtered = append(filtered, finding)
		}
	}
	return filtered
}
