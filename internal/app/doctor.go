package app

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/providers"
	"a21.local/a21/internal/runtimeguard"
	"a21.local/a21/internal/v21adapter"
)

var listFirmwareSerialDevices = func() ([]firmwarecheck.SerialDevice, error) {
	return firmwarecheck.ListSerialDevices(context.Background())
}

var probeV21AdapterHealth = func(ctx context.Context, baseURL string) error {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	return v21adapter.ProbeHealth(ctx, baseURL, client)
}

var probeVoiceProviderHealth = func(ctx context.Context) (providers.VoiceProviderHealth, error) {
	return providers.NewMockVoiceProvider().Health(ctx)
}

type doctorReport struct {
	Result      runtimeguard.Result            `json:"result"`
	Fingerprint runtimeguard.Fingerprint       `json:"fingerprint"`
	Proxy       runtimeguard.ProxyPolicyReport `json:"proxy"`
	Firmware    firmwareDoctorReport           `json:"firmware"`
	Voice       voiceDoctorReport              `json:"voice"`
	V21         v21DoctorReport                `json:"v21"`
}

type firmwareDoctorReport struct {
	ManifestPath        string                       `json:"manifest_path"`
	ManifestOK          bool                         `json:"manifest_ok"`
	PlatformIOVenvPath  string                       `json:"platformio_venv_path"`
	PlatformIOVenvOK    bool                         `json:"platformio_venv_ok"`
	PlatformIOCoreDir   string                       `json:"platformio_core_dir"`
	PlatformIOCoreOK    bool                         `json:"platformio_core_ok"`
	CurrentCommit       string                       `json:"current_commit,omitempty"`
	CurrentArtifactPath string                       `json:"current_artifact_path,omitempty"`
	ArtifactCount       int                          `json:"artifact_count"`
	SerialDevices       []firmwarecheck.SerialDevice `json:"serial_devices"`
	Findings            []runtimeguard.Finding       `json:"findings"`
}

type v21DoctorReport struct {
	Configured bool                   `json:"configured"`
	Status     string                 `json:"status"`
	Healthy    bool                   `json:"healthy"`
	HealthPath string                 `json:"health_path"`
	Findings   []runtimeguard.Finding `json:"findings"`
}

type voiceDoctorReport struct {
	Provider       string                 `json:"provider"`
	Status         string                 `json:"status"`
	Healthy        bool                   `json:"healthy"`
	Configured     bool                   `json:"configured"`
	Realtime       bool                   `json:"realtime"`
	ActiveProvider string                 `json:"active_provider,omitempty"`
	Detail         string                 `json:"detail,omitempty"`
	Findings       []runtimeguard.Finding `json:"findings"`
}

func buildDoctorReport(preflight runtimeguard.PreflightReport, projectRoot string, currentCommit string) doctorReport {
	firmware := buildFirmwareDoctorReport(projectRoot, currentCommit)
	voice := buildVoiceDoctorReport()
	v21 := buildV21DoctorReport(os.Getenv("A21_V21_ADAPTER_URL"))
	findings := append([]runtimeguard.Finding{}, preflight.Result.Findings...)
	findings = append(findings, firmware.Findings...)
	findings = append(findings, voice.Findings...)
	findings = append(findings, v21.Findings...)
	return doctorReport{
		Result:      runtimeguard.NewResult(findings),
		Fingerprint: preflight.Fingerprint,
		Proxy:       preflight.Proxy,
		Firmware:    firmware,
		Voice:       voice,
		V21:         v21,
	}
}

func buildVoiceDoctorReport() voiceDoctorReport {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	health, err := probeVoiceProviderHealth(ctx)
	report := voiceDoctorReport{
		Provider:       health.Provider,
		Status:         string(health.Status),
		Healthy:        health.Status == providers.VoiceProviderHealthy,
		Configured:     health.Configured,
		Realtime:       health.Realtime,
		ActiveProvider: health.ActiveProvider,
		Detail:         health.Detail,
	}
	if report.Provider == "" {
		report.Provider = "unknown"
	}
	if report.Status == "" {
		report.Status = string(providers.VoiceProviderUnavailable)
	}
	if err != nil || health.Status == providers.VoiceProviderUnavailable {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "voice_provider_health_failed",
			Severity: runtimeguard.SeverityWarn,
			Message:  "A21 voice provider health check failed",
			Detail:   redactDoctorSecret(errString(err, report.Detail)),
		})
	}
	return report
}

func buildV21DoctorReport(adapterURL string) v21DoctorReport {
	report := v21DoctorReport{
		Configured: strings.TrimSpace(adapterURL) != "",
		HealthPath: v21adapter.HealthPath,
	}
	if !report.Configured {
		report.Status = "skipped"
		return report
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := probeV21AdapterHealth(ctx, adapterURL); err != nil {
		report.Status = "unhealthy"
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "v21_adapter_health_failed",
			Severity: runtimeguard.SeverityWarn,
			Message:  "A21 V21 adapter health check failed",
			Detail:   redactDoctorSecret(err.Error()),
		})
		return report
	}
	report.Status = "healthy"
	report.Healthy = true
	return report
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
	devices, err := listFirmwareSerialDevices()
	if err != nil {
		report.Findings = append(report.Findings, runtimeguard.Finding{
			Code:     "firmware_serial_inventory_failed",
			Severity: runtimeguard.SeverityWarn,
			Message:  "A21 could not list local serial devices",
			Detail:   err.Error(),
		})
	} else {
		report.SerialDevices = devices
	}
	return report
}

func errString(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

var doctorURLCredentialPattern = regexp.MustCompile(`(https?://)[^/\s"']+@`)

func redactDoctorSecret(text string) string {
	return doctorURLCredentialPattern.ReplaceAllString(text, "${1}<redacted>@")
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
