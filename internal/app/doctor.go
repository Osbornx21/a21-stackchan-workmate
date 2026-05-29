package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
)

type doctorReport struct {
	Result      runtimeguard.Result      `json:"result"`
	Fingerprint runtimeguard.Fingerprint `json:"fingerprint"`
	Firmware    firmwareDoctorReport     `json:"firmware"`
}

type firmwareDoctorReport struct {
	ManifestPath        string                 `json:"manifest_path"`
	ManifestOK          bool                   `json:"manifest_ok"`
	PlatformIOVenvPath  string                 `json:"platformio_venv_path"`
	PlatformIOVenvOK    bool                   `json:"platformio_venv_ok"`
	PlatformIOCoreDir   string                 `json:"platformio_core_dir"`
	PlatformIOCoreOK    bool                   `json:"platformio_core_ok"`
	CurrentCommit       string                 `json:"current_commit,omitempty"`
	CurrentArtifactPath string                 `json:"current_artifact_path,omitempty"`
	ArtifactCount       int                    `json:"artifact_count"`
	Findings            []runtimeguard.Finding `json:"findings"`
}

func buildDoctorReport(preflight runtimeguard.PreflightReport, projectRoot string, currentCommit string) doctorReport {
	firmware := buildFirmwareDoctorReport(projectRoot, currentCommit)
	findings := append([]runtimeguard.Finding{}, preflight.Result.Findings...)
	findings = append(findings, firmware.Findings...)
	return doctorReport{
		Result:      runtimeguard.NewResult(findings),
		Fingerprint: preflight.Fingerprint,
		Firmware:    firmware,
	}
}

func buildFirmwareDoctorReport(projectRoot string, currentCommit string) firmwareDoctorReport {
	report := firmwareDoctorReport{
		ManifestPath:       filepath.Join(projectRoot, "firmware", "stackchan", "a21-firmware.json"),
		PlatformIOVenvPath: filepath.Join(projectRoot, ".a21-tools", "platformio-venv", "bin", "pio"),
		PlatformIOCoreDir:  filepath.Join(projectRoot, ".a21-tools", "platformio-core"),
		CurrentCommit:      currentCommit,
	}

	if _, err := firmwarecheck.LoadAndValidate(report.ManifestPath); err != nil {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "firmware_manifest_invalid",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 firmware manifest or PlatformIO config is invalid",
			Detail:   err.Error(),
		})
	} else {
		report.ManifestOK = true
	}

	if info, err := os.Stat(report.PlatformIOVenvPath); err == nil && !info.IsDir() {
		report.PlatformIOVenvOK = true
	} else {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "firmware_platformio_venv_missing",
			Severity: runtimeguard.SeverityWarn,
			Message:  "A21 repository-local PlatformIO executable is missing",
			Detail:   report.PlatformIOVenvPath,
		})
	}

	if info, err := os.Stat(report.PlatformIOCoreDir); err == nil && info.IsDir() {
		report.PlatformIOCoreOK = true
	} else {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "firmware_platformio_core_missing",
			Severity: runtimeguard.SeverityWarn,
			Message:  "A21 repository-local PlatformIO core directory is missing",
			Detail:   report.PlatformIOCoreDir,
		})
	}

	artifacts := findFirmwareArtifacts(projectRoot)
	report.ArtifactCount = len(artifacts)
	for _, artifact := range artifacts {
		result, err := firmwarecheck.ValidateArtifact(firmwarecheck.ArtifactOptions{
			ManifestPath: report.ManifestPath,
			ArtifactPath: artifact,
		})
		if err != nil {
			report.Findings = append(report.Findings, runtimeguard.Finding{
				Code:     "firmware_artifact_invalid",
				Severity: runtimeguard.SeverityWarn,
				Message:  "A21 firmware artifact failed identity or checksum validation",
				Detail:   artifact + ": " + err.Error(),
			})
			continue
		}
		if currentCommit != "" && sameDoctorCommit(currentCommit, result.Commit) {
			report.CurrentArtifactPath = artifact
		}
	}
	if currentCommit != "" && report.CurrentArtifactPath == "" {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "firmware_current_artifact_missing",
			Severity: runtimeguard.SeverityWarn,
			Message:  "No validated A21 firmware artifact matches the current git commit",
			Detail:   currentCommit,
		})
	}
	return report
}

func findProjectRoot(start string) string {
	dir := start
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil && strings.Contains(string(data), "module a21.local/a21") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}

func currentGitCommit(projectRoot string) string {
	output, err := exec.Command("git", "-C", projectRoot, "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func findFirmwareArtifacts(projectRoot string) []string {
	matches, err := filepath.Glob(filepath.Join(projectRoot, "firmware", "artifacts", "a21-stackchan-*.bin"))
	if err != nil {
		return nil
	}
	sort.Strings(matches)
	return matches
}

func sameDoctorCommit(expected string, actual string) bool {
	expected = strings.ToLower(expected)
	actual = strings.ToLower(actual)
	return expected == actual || strings.HasPrefix(expected, actual) || strings.HasPrefix(actual, expected)
}
