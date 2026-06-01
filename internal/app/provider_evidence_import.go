package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type providerEvidenceImportOptions struct {
	Bundle    string
	OutputDir string
}

type providerEvidenceImportReport struct {
	SchemaVersion      string                          `json:"schema_version"`
	GeneratedAtMS      int64                           `json:"generated_at_ms"`
	Status             string                          `json:"status"`
	SourceBundle       string                          `json:"source_bundle,omitempty"`
	ProviderSmokeReady bool                            `json:"provider_smoke_ready"`
	SourceReport       string                          `json:"source_report,omitempty"`
	ImportedReports    []string                        `json:"imported_reports,omitempty"`
	Findings           []providerEvidenceImportFinding `json:"findings,omitempty"`
	Redaction          providerEvidenceImportRedaction `json:"redaction"`
	ReportPath         string                          `json:"report_path,omitempty"`
}

type providerEvidenceImportFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type providerEvidenceImportRedaction struct {
	PayloadsStored         bool `json:"payloads_stored"`
	PromptTextStored       bool `json:"prompt_text_stored"`
	TranscriptStored       bool `json:"transcript_stored"`
	ProviderOutputStored   bool `json:"provider_output_stored"`
	FullURLsStored         bool `json:"full_urls_stored"`
	LocalPathsStored       bool `json:"local_paths_stored"`
	CredentialValuesStored bool `json:"credential_values_stored"`
}

type providerEvidenceImportEntry struct {
	Name string
	Data []byte
}

func runProviderEvidenceImport(args []string, stdout io.Writer, stderr io.Writer) int {
	options := providerEvidenceImportOptions{OutputDir: "reports"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 provider-evidence-import --bundle reports/a21-5080lab-provider-evidence-YYYYMMDD-HHMMSS.tgz [--output-dir reports]")
			return 0
		case "--bundle":
			if !readStringOption(args, &i, stderr, "--bundle", &options.Bundle) {
				return 2
			}
		case "--output-dir":
			if !readStringOption(args, &i, stderr, "--output-dir", &options.OutputDir) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown provider-evidence-import option %q\n", args[i])
			return 2
		}
	}
	report := importProviderEvidenceBundle(options)
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			report.Status = "rejected"
			report.Findings = append(report.Findings, providerEvidenceImportFinding{
				Code:    "output_dir_invalid",
				Message: "Provider evidence import output dir is invalid",
			})
		} else if err := writeProviderEvidenceImportReport(options.OutputDir, &report); err != nil {
			fmt.Fprintln(stderr, "write provider evidence import report failed")
			return 1
		}
	}
	if err := writeJSONProviderEvidenceImport(stdout, report); err != nil {
		fmt.Fprintln(stderr, "encode provider evidence import report failed")
		return 1
	}
	if report.Status != "accepted" {
		return 1
	}
	return 0
}

func importProviderEvidenceBundle(options providerEvidenceImportOptions) providerEvidenceImportReport {
	report := providerEvidenceImportReport{
		SchemaVersion: "a21.provider_evidence_import.v1",
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "rejected",
		SourceBundle:  filepath.Base(filepath.Clean(options.Bundle)),
		Redaction:     providerEvidenceImportRedaction{},
	}
	if !validProviderEvidenceBundleName(report.SourceBundle) {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "bundle_name_invalid",
			Message: "Provider evidence bundle name is invalid",
		})
		return report
	}
	entries, findings := readProviderEvidenceBundleEntries(options.Bundle)
	report.Findings = append(report.Findings, findings...)
	if len(findings) != 0 {
		return report
	}
	if len(entries) == 0 {
		report.Findings = append(report.Findings, providerEvidenceImportFinding{
			Code:    "bundle_empty",
			Message: "Provider evidence bundle has no importable A21 reports",
		})
		return report
	}
	selectedProviderSmoke, providerFindings := selectedProviderEvidenceImportSmoke(entries, buildProductProviderReadiness(os.Environ()))
	report.Findings = append(report.Findings, providerFindings...)
	if selectedProviderSmoke == "" {
		return report
	}
	if findings := writeProviderEvidenceImportEntries(options.OutputDir, entries); len(findings) != 0 {
		report.Findings = append(report.Findings, findings...)
		return report
	}
	report.Status = "accepted"
	report.ProviderSmokeReady = true
	report.SourceReport = selectedProviderSmoke
	for _, entry := range entries {
		report.ImportedReports = append(report.ImportedReports, entry.Name)
	}
	sort.Strings(report.ImportedReports)
	return report
}

func validProviderEvidenceBundleName(name string) bool {
	return strings.HasPrefix(name, "a21-5080lab-provider-evidence-") && strings.HasSuffix(name, ".tgz")
}

func readProviderEvidenceBundleEntries(bundlePath string) ([]providerEvidenceImportEntry, []providerEvidenceImportFinding) {
	file, err := os.Open(bundlePath)
	if err != nil {
		return nil, []providerEvidenceImportFinding{{
			Code:    "bundle_unreadable",
			Message: "Provider evidence bundle is unreadable",
		}}
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, []providerEvidenceImportFinding{{
			Code:    "bundle_invalid",
			Message: "Provider evidence bundle is not a valid gzip archive",
		}}
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	entriesByName := map[string][]byte{}
	var findings []providerEvidenceImportFinding
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, []providerEvidenceImportFinding{{
				Code:    "bundle_invalid",
				Message: "Provider evidence bundle tar stream is invalid",
			}}
		}
		if header.FileInfo().IsDir() {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_invalid",
				Message: "Provider evidence bundle contains an unsupported entry type",
			})
			continue
		}
		name, ok := providerEvidenceImportEntryName(header.Name)
		if !ok {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_name_invalid",
				Message: "Provider evidence bundle contains an unsafe entry name",
			})
			continue
		}
		if !providerEvidenceImportAllowedReportName(name) {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_not_allowed",
				Message: "Provider evidence bundle contains a non-whitelisted report",
				Detail:  safeProviderEvidenceImportDetail(name),
			})
			continue
		}
		if _, exists := entriesByName[name]; exists {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_duplicate",
				Message: "Provider evidence bundle contains a duplicate report",
				Detail:  name,
			})
			continue
		}
		if header.Size < 0 || header.Size > providerLatencyFixtureSidecarMaxBytes {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_too_large",
				Message: "Provider evidence bundle report is too large",
				Detail:  name,
			})
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tarReader, providerLatencyFixtureSidecarMaxBytes+1))
		if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_too_large",
				Message: "Provider evidence bundle report is too large",
				Detail:  name,
			})
			continue
		}
		if providerEvidenceImportJSONUnsafe(data) {
			findings = append(findings, providerEvidenceImportFinding{
				Code:    "bundle_entry_unsafe",
				Message: "Provider evidence bundle report failed redaction checks",
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

func providerEvidenceImportEntryName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "/") {
		return "", false
	}
	clean := path.Clean(name)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/") {
		return "", false
	}
	if filepath.Base(clean) != clean {
		return "", false
	}
	return clean, true
}

func providerEvidenceImportAllowedReportName(name string) bool {
	for _, pattern := range []string{
		"a21-provider-smoke-*.json",
		"a21-doctor-*.json",
		"a21-product-readiness-*.json",
		"a21-server-side-readiness-bundle-*.json",
	} {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

func providerEvidenceImportJSONUnsafe(data []byte) bool {
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return true
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return true
	}
	return providerLatencyFixtureContainsForbiddenKey(raw) || productProviderSmokeReportContainsForbiddenValue(raw)
}

func selectedProviderEvidenceImportSmoke(entries []providerEvidenceImportEntry, readiness productProviderReadiness) (string, []providerEvidenceImportFinding) {
	tempDir, err := os.MkdirTemp("", "a21-provider-evidence-import-*")
	if err != nil {
		return "", []providerEvidenceImportFinding{{
			Code:    "temp_dir_unavailable",
			Message: "Provider evidence import could not validate reports",
		}}
	}
	defer os.RemoveAll(tempDir)
	var providerNames []string
	for _, entry := range entries {
		if ok, _ := filepath.Match("a21-provider-smoke-*.json", entry.Name); ok {
			providerNames = append(providerNames, entry.Name)
			if err := os.WriteFile(filepath.Join(tempDir, entry.Name), entry.Data, 0o644); err != nil {
				return "", []providerEvidenceImportFinding{{
					Code:    "temp_report_unavailable",
					Message: "Provider evidence import could not stage a report",
				}}
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(providerNames)))
	var mismatchedSource string
	requireSelectedMatch := productProviderReadinessRequiresSmokeMatch(readiness)
	for _, name := range providerNames {
		evidence, _ := loadProductProviderSmokeReportEvidence(filepath.Join(tempDir, name))
		if evidence.Valid && evidence.Executed && evidence.Stream {
			if requireSelectedMatch && !productProviderSmokeEvidenceMatchesSelected(readiness, evidence) {
				if mismatchedSource == "" {
					mismatchedSource = name
				}
				continue
			}
			return name, nil
		}
	}
	if mismatchedSource != "" {
		return "", []providerEvidenceImportFinding{{
			Code:    "provider_smoke_report_mismatch",
			Message: "Provider evidence bundle smoke report does not match the currently selected configured A21 provider",
			Detail:  mismatchedSource,
		}}
	}
	return "", []providerEvidenceImportFinding{{
		Code:    "provider_smoke_executed_missing",
		Message: "Provider evidence bundle does not contain an accepted executed streaming provider smoke report",
	}}
}

func writeProviderEvidenceImportEntries(outputDir string, entries []providerEvidenceImportEntry) []providerEvidenceImportFinding {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return []providerEvidenceImportFinding{{
			Code:    "output_dir_unavailable",
			Message: "Provider evidence import output dir is unavailable",
		}}
	}
	for _, entry := range entries {
		target := filepath.Join(outputDir, entry.Name)
		if existing, err := os.ReadFile(target); err == nil {
			if !bytes.Equal(existing, entry.Data) {
				return []providerEvidenceImportFinding{{
					Code:    "output_report_conflict",
					Message: "Provider evidence import would overwrite an existing report",
					Detail:  entry.Name,
				}}
			}
			continue
		}
		if err := os.WriteFile(target, entry.Data, 0o644); err != nil {
			return []providerEvidenceImportFinding{{
				Code:    "output_report_unavailable",
				Message: "Provider evidence import could not write a report",
				Detail:  entry.Name,
			}}
		}
	}
	return nil
}

func safeProviderEvidenceImportDetail(name string) string {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") || containsLegacyIdentity(name) {
		return "unsafe_entry"
	}
	return name
}

func writeProviderEvidenceImportReport(outputDir string, report *providerEvidenceImportReport) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	reportPath := filepath.Join(outputDir, "a21-provider-evidence-import-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	return writeJSONProviderEvidenceImport(file, *report)
}

func writeJSONProviderEvidenceImport(writer io.Writer, report providerEvidenceImportReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
