package app

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type providerEvidencePackageOptions struct {
	InputDir  string
	OutputDir string
}

type providerEvidencePackageReport struct {
	SchemaVersion      string                          `json:"schema_version"`
	GeneratedAtMS      int64                           `json:"generated_at_ms"`
	Status             string                          `json:"status"`
	BundlePath         string                          `json:"bundle_path,omitempty"`
	ProviderSmokeReady bool                            `json:"provider_smoke_ready"`
	SourceReport       string                          `json:"source_report,omitempty"`
	PackagedReports    []string                        `json:"packaged_reports,omitempty"`
	Findings           []providerEvidenceImportFinding `json:"findings,omitempty"`
	Redaction          providerEvidenceImportRedaction `json:"redaction"`
}

func runProviderEvidencePackage(args []string, stdout io.Writer, stderr io.Writer) int {
	options := providerEvidencePackageOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-evidence-package --input-dir reports/5080lab-provider [--output-dir reports]")
			return 0
		case "--input-dir":
			if !readStringOption(args, &i, stderr, "--input-dir", &options.InputDir) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown provider-evidence-package option %q\n", args[i])
			return 2
		}
	}
	report := packageProviderEvidenceReports(options)
	if err := writeJSONProviderEvidencePackage(stdout, report); err != nil {
		fmt.Fprintln(stderr, "encode provider evidence package report failed")
		return 1
	}
	if report.Status != "accepted" {
		return 1
	}
	return 0
}

func packageProviderEvidenceReports(options providerEvidencePackageOptions) providerEvidencePackageReport {
	report := providerEvidencePackageReport{
		SchemaVersion: "a21.provider_evidence_package.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "rejected",
		Redaction:     providerEvidenceImportRedaction{},
	}
	if options.InputDir == "" {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "input_dir_required",
			Message: "Provider evidence package input dir is required",
		})
		return report
	}
	if err := validateA21InputPath(options.InputDir); err != nil {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "input_dir_invalid",
			Message: "Provider evidence package input dir is invalid",
		})
		return report
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "output_dir_invalid",
			Message: "Provider evidence package output dir is invalid",
		})
		return report
	}
	entries, findings := readProviderEvidencePackageEntries(options.InputDir)
	report.Findings = append(report.Findings, findings...)
	if len(findings) != 0 {
		return report
	}
	if len(entries) == 0 {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "package_empty",
			Message: "Provider evidence package has no importable A21 reports",
		})
		return report
	}
	selectedProviderSmoke, providerFindings := selectedProviderEvidenceImportSmoke(entries)
	report.Findings = append(report.Findings, providerFindings...)
	if selectedProviderSmoke == "" {
		return report
	}
	bundlePath, writeFindings := writeProviderEvidencePackageBundle(options.OutputDir, entries)
	report.Findings = append(report.Findings, writeFindings...)
	if len(writeFindings) != 0 {
		return report
	}
	report.Status = "accepted"
	report.ProviderSmokeReady = true
	report.SourceReport = selectedProviderSmoke
	report.BundlePath = bundlePath
	for _, entry := range entries {
		report.PackagedReports = append(report.PackagedReports, entry.Name)
	}
	sort.Strings(report.PackagedReports)
	return report
}

func readProviderEvidencePackageEntries(inputDir string) ([]providerEvidenceImportEntry, []providerEvidenceImportFinding) {
	dirEntries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, []providerEvidenceImportFinding{{
			Code:    "input_dir_unreadable",
			Message: "Provider evidence package input dir is unreadable",
		}}
	}
	entriesByName := map[string][]byte{}
	var findings []providerEvidenceImportFinding
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		name, ok := providerEvidenceImportEntryName(dirEntry.Name())
		if !ok {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_name_invalid",
				Message: "Provider evidence package contains an unsafe entry name",
			})
			continue
		}
		if !providerEvidenceImportAllowedReportName(name) {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_not_allowed",
				Message: "Provider evidence package contains a non-whitelisted report",
				Detail:  safeProviderEvidenceImportDetail(name),
			})
			continue
		}
		info, err := dirEntry.Info()
		if err != nil || !info.Mode().IsRegular() {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_invalid",
				Message: "Provider evidence package contains an unsupported entry type",
				Detail:  safeProviderEvidenceImportDetail(name),
			})
			continue
		}
		if info.Size() < 0 || info.Size() > providerLatencyFixtureSidecarMaxBytes {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_too_large",
				Message: "Provider evidence package report is too large",
				Detail:  name,
			})
			continue
		}
		data, err := os.ReadFile(filepath.Join(inputDir, name))
		if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_too_large",
				Message: "Provider evidence package report is too large",
				Detail:  name,
			})
			continue
		}
		if providerEvidenceImportJSONUnsafe(data) {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "package_entry_unsafe",
				Message: "Provider evidence package report failed redaction checks",
				Detail:  name,
			})
			continue
		}
		entriesByName[name] = data
	}
	if len(findings) != 0 {
		return nil, findings
	}
	names := make([]string, 0, len(entriesByName))
	for name := range entriesByName {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]providerEvidenceImportEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, providerEvidenceImportEntry{Name: name, Data: entriesByName[name]})
	}
	return entries, nil
}

func writeProviderEvidencePackageBundle(outputDir string, entries []providerEvidenceImportEntry) (string, []providerEvidenceImportFinding) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", []providerEvidenceImportFinding{{
			Code:    "output_dir_unavailable",
			Message: "Provider evidence package output dir is unavailable",
		}}
	}
	bundleName := "a21-5080lab-provider-evidence-" + time.Now().Format("20060102-150405") + ".tgz"
	target := filepath.Join(outputDir, bundleName)
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", []providerEvidenceImportFinding{{
			Code:    "output_bundle_unavailable",
			Message: "Provider evidence package could not write bundle",
		}}
	}
	writeOK := false
	defer func() {
		if !writeOK {
			_ = os.Remove(target)
		}
	}()
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.Name,
			Mode: 0o644,
			Size: int64(len(entry.Data)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			_ = file.Close()
			return "", []providerEvidenceImportFinding{{
				Code:    "output_bundle_unavailable",
				Message: "Provider evidence package could not write bundle",
			}}
		}
		if _, err := tarWriter.Write(entry.Data); err != nil {
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			_ = file.Close()
			return "", []providerEvidenceImportFinding{{
				Code:    "output_bundle_unavailable",
				Message: "Provider evidence package could not write bundle",
			}}
		}
	}
	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		_ = file.Close()
		return "", []providerEvidenceImportFinding{{
			Code:    "output_bundle_unavailable",
			Message: "Provider evidence package could not write bundle",
		}}
	}
	if err := gzipWriter.Close(); err != nil {
		_ = file.Close()
		return "", []providerEvidenceImportFinding{{
			Code:    "output_bundle_unavailable",
			Message: "Provider evidence package could not write bundle",
		}}
	}
	if err := file.Close(); err != nil {
		return "", []providerEvidenceImportFinding{{
			Code:    "output_bundle_unavailable",
			Message: "Provider evidence package could not write bundle",
		}}
	}
	writeOK = true
	return bundleName, nil
}

func writeJSONProviderEvidencePackage(writer io.Writer, report providerEvidencePackageReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
