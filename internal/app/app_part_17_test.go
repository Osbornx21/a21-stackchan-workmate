package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"a21.local/a21/internal/firmwarecheck"
)

func TestBuildV21DoctorReportRedactsHealthFailureSecrets(t *testing.T) {
	originalProbe := probeV21AdapterHealth
	probeV21AdapterHealth = func(ctx context.Context, baseURL string) error {
		return errors.New(`Get "http://user:secret-token@127.0.0.1:21121/healthz": connection refused`)
	}
	defer func() {
		probeV21AdapterHealth = originalProbe
	}()

	report := buildV21DoctorReport("http://user:secret-token@127.0.0.1:21121")

	if report.Status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", report.Status)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	if strings.Contains(report.Findings[0].Detail, "secret-token") || strings.Contains(report.Findings[0].Detail, "user:") {
		t.Fatalf("detail leaked credentials: %q", report.Findings[0].Detail)
	}
}

func TestBuildFirmwareDoctorReportFindsCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	manifest := writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	_ = manifest
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-venv", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio"), []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if !report.ManifestOK {
		t.Fatalf("ManifestOK = false, findings = %#v", report.Findings)
	}
	if !report.PlatformIOVenvOK {
		t.Fatal("expected local PlatformIO venv to be detected")
	}
	if !report.PlatformIOVersionOK {
		t.Fatalf("expected pinned PlatformIO version, got %#v", report)
	}
	if !report.PlatformIOCoreOK {
		t.Fatal("expected local PlatformIO core to be detected")
	}
	if report.CurrentArtifactPath != artifact {
		t.Fatalf("CurrentArtifactPath = %q, want %q", report.CurrentArtifactPath, artifact)
	}
}

func TestBuildFirmwareDoctorReportRejectsLooseCurrentArtifact(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.1.19'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDir := filepath.Join(root, "firmware", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	content := []byte("firmware\na21-stackchan\n0.1.0\nm5stack-cores3\nabcdef1\n")
	if err := os.WriteFile(artifact, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := os.WriteFile(artifact+".sha256", []byte(hex.EncodeToString(sum[:])+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "abcdef1")

	if report.CurrentArtifactPath != "" {
		t.Fatalf("CurrentArtifactPath = %q, want empty for loose artifact", report.CurrentArtifactPath)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "firmware_current_artifact_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_current_artifact_missing: %#v", report.Findings)
	}
}

func TestBuildFirmwareDoctorReportWarnsOnPlatformIOVersionMismatch(t *testing.T) {
	root := t.TempDir()
	writeTestFirmwareManifest(t, filepath.Join(root, "firmware", "stackchan"))
	pioPath := filepath.Join(root, ".a21-tools", "platformio-venv", "bin", "pio")
	if err := os.MkdirAll(filepath.Dir(pioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pioPath, []byte("#!/bin/sh\necho 'PlatformIO Core, version 6.2.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".a21-tools", "platformio-core"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := buildFirmwareDoctorReport(root, "")

	if report.PlatformIOVersionOK {
		t.Fatalf("PlatformIOVersionOK = true, want false: %#v", report)
	}
	if report.PlatformIOVersion != "6.2.0" || report.ExpectedPlatformIOVersion != "6.1.19" {
		t.Fatalf("version fields = got %q expected %q", report.PlatformIOVersion, report.ExpectedPlatformIOVersion)
	}
	var found bool
	for _, finding := range report.Findings {
		if finding.Code == "firmware_platformio_version_mismatch" {
			found = true
			if strings.Contains(strings.ToLower(finding.Detail), "x21") || strings.Contains(strings.ToLower(finding.Detail), "v21") {
				t.Fatalf("finding detail leaked legacy identity: %#v", finding)
			}
		}
	}
	if !found {
		t.Fatalf("findings missing firmware_platformio_version_mismatch: %#v", report.Findings)
	}
}

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunGatewayHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"gateway", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "a21 gateway") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGatewayRejectsUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"gateway", "--wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunLatencyBenchMockEmitsPercentileReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{"latency-bench", "--mock", "--iterations", "3"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"mode": "mock"`,
		`"iterations": 3`,
		`"mock_turn_ms"`,
		`"professional_turn_ms"`,
		`"barge_in_stop_ms"`,
		`"audio_ws_downlink_ms"`,
		`"audio_ws_barge_in_stop_ms"`,
		`"p50_ms"`,
		`"p95_ms"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunLatencyBenchWritesReportWhenOutputDirProvided(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunLab([]string{"latency-bench", "--mock", "--iterations", "2", "--output-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"report_path"`) {
		t.Fatalf("stdout missing report_path: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"mode": "mock"`, `"iterations": 2`, `"audio_ws_downlink_ms"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report missing %q: %s", want, string(data))
		}
	}
}

func TestRunLatencyBenchReportIncludesTraceableEnvironmentMetadataWithoutProxySecrets(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:secret@example.invalid:8080")
	t.Setenv("A21_PROVIDER_PROXY_URL", "http://provider-secret@example.invalid:9000")
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunLab([]string{"latency-bench", "--mock", "--iterations", "1", "--output-dir", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"metadata"`,
		`"generated_at"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"HTTPS_PROXY"`,
		`"provider_proxy_env"`,
		`"A21_PROVIDER_PROXY_URL"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout leaked proxy value fragment %q: %s", forbidden, stdout.String())
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "a21-latency-bench-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("report files = %d, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := string(data)
	for _, want := range []string{
		`"metadata"`,
		`"current_commit"`,
		`"fingerprint"`,
		`"proxy"`,
		`"global_proxy_env"`,
		`"provider_proxy_env"`,
	} {
		if !strings.Contains(reportJSON, want) {
			t.Fatalf("report missing %q: %s", want, reportJSON)
		}
	}
	for _, forbidden := range []string{"secret", "example.invalid", "8080", "9000"} {
		if strings.Contains(reportJSON, forbidden) {
			t.Fatalf("report leaked proxy value fragment %q: %s", forbidden, reportJSON)
		}
	}
}

func TestRunLatencyBenchRequiresMockMode(t *testing.T) {
	var stderr bytes.Buffer
	code := RunLab([]string{"latency-bench"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--mock is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLatencyBenchPercentileUsesNearestRank(t *testing.T) {
	got := percentileMS([]time.Duration{time.Millisecond, 2 * time.Millisecond, 100 * time.Millisecond}, 0.95)
	if got != 100 {
		t.Fatalf("p95 = %v, want 100", got)
	}
}

func TestRunFirmwareCheckRejectsMissingManifest(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", filepath.Join(t.TempDir(), "missing.json")}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error text")
	}
}

func TestRunFirmwareCheckAcceptsA21Manifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	platformio := filepath.Join(dir, "platformio.ini")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(platformio, []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware manifest ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCheckDispatchesKindHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--kind", "upload", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"a21 firmware-check --kind upload", "--artifact", "--port", "--commit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunDeprecatedFirmwareCheckAliasStillDispatches(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"firmware-upload-check", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 firmware-check --kind upload") {
		t.Fatalf("stdout missing upload help: %s", stdout.String())
	}
}

func TestRunPlanExecuteCommandDispatchesHelp(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"firmware-bootstrap-flash", "--help"}, want: "a21 firmware-bootstrap-flash"},
		{args: []string{"firmware-bootstrap-flash-execute", "--help"}, want: "a21 firmware-bootstrap-flash --execute"},
		{args: []string{"stackchan-official-pcm-bridge-nvs", "--help"}, want: "a21 stackchan-official-pcm-bridge-nvs"},
		{args: []string{"stackchan-official-pcm-bridge-nvs-execute", "--help"}, want: "a21 stackchan-official-pcm-bridge-nvs"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("stdout missing %q: %s", tc.want, stdout.String())
			}
		})
	}
}

func TestRunFirmwareCheckRejectsWrongPlatformIOBoard(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	data := []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-core2")), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platformio board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwarePackageCreatesVersionedArtifact(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{Root: "/tmp/a21", Clean: true}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "a21-firmware.json")
	if err := os.WriteFile(manifest, []byte(`{
  "project": "A21",
  "firmware_id": "a21-stackchan",
  "version": "0.1.0",
  "board": "m5stack-cores3",
  "artifact_prefix": "a21-stackchan"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(testPlatformIOConfig("m5stack-cores3")), 0o644); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin")
	if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("firmware a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(dir, "artifacts")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", outputDir,
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	artifact := filepath.Join(outputDir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if _, err := os.Stat(artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact + ".manifest.json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Base(artifact)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "release_manifest_path") {
		t.Fatalf("stdout missing release_manifest_path: %s", stdout.String())
	}
}

func TestRunFirmwarePackageRejectsDirtySourceTree(t *testing.T) {
	originalDetector := detectFirmwareSourceState
	detectFirmwareSourceState = func() (firmwareSourceState, error) {
		return firmwareSourceState{
			Root:   "/tmp/a21",
			Clean:  false,
			Detail: " M firmware/stackchan/src/main.cpp\n?? scratch.bin",
		}, nil
	}
	defer func() {
		detectFirmwareSourceState = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	input := filepath.Join(dir, "firmware.bin")
	if err := os.WriteFile(input, []byte("a21-stackchan 0.1.0 m5stack-cores3 abcdef1"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-package",
		"--manifest", manifest,
		"--input", input,
		"--output-dir", filepath.Join(dir, "artifacts"),
		"--commit", "abcdef1",
		"--timestamp", "20260530-004500",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "source tree is dirty") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "firmware/stackchan/src/main.cpp") {
		t.Fatalf("stderr missing dirty detail: %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsUnpinnedPlatformIOPlatform(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "platform = espressif32@7.0.1", "platform = espressif32")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "platform") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCheckRejectsLegacyGatewayPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	config := strings.ReplaceAll(testPlatformIOConfig("m5stack-cores3"), "-D A21_GATEWAY_PORT=21080", "-D A21_GATEWAY_PORT=8080")
	if err := os.WriteFile(filepath.Join(dir, "platformio.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{"firmware-check", "--manifest", manifest}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "gateway port") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckAcceptsMatchingArtifactAndChecksum(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsWrongBoardInArtifactName(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-core2-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "board") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactCheckRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.WriteFile(artifact, []byte("firmware"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact+".sha256", []byte(strings.Repeat("0", 64)+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "checksum") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckAcceptsNewestCommitArtifact(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	oldArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	newArtifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-005500.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, newArtifact, []byte("new firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-current-artifact-check",
		"--manifest", manifest,
		"--artifact-dir", dir,
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), newArtifact) {
		t.Fatalf("stdout missing newest artifact %q: %s", newArtifact, stdout.String())
	}
	if !strings.Contains(stdout.String(), "firmware current artifact ok") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFirmwareCurrentArtifactCheckRequiresCommit(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"firmware-current-artifact-check"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareArtifactPrunePlanWritesNoDeleteReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-1111111-20260530-010000.bin")
	currentArtifact := filepath.Join(artifactDir, "a21-stackchan-0.1.0-m5stack-cores3-2222222-20260530-020000.bin")
	writeFirmwareArtifactWithChecksum(t, oldArtifact, []byte("old firmware"))
	writeFirmwareArtifactWithChecksum(t, currentArtifact, []byte("current firmware"))
	reportDir := filepath.Join(dir, "reports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-artifact-prune-plan",
		"--manifest", manifest,
		"--artifact-dir", artifactDir,
		"--commit", "2222222",
		"--keep-recent", "1",
		"--output-dir", reportDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.summary.v1"`,
		`"delete_allowed": false`,
		`"keep_count": 1`,
		`"prune_candidate_count": 1`,
		`"manual_review_count": 0`,
		`"report_path":`,
		"firmware artifact prune plan ok (no files deleted)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), oldArtifact) {
		t.Fatalf("stdout should be summary-only when --output-dir is set: %s", stdout.String())
	}
	matches, err := filepath.Glob(filepath.Join(reportDir, "a21-firmware-artifact-prune-plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("reports = %v, want one prune-plan report", matches)
	}
	reportData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"schema_version": "a21.firmware.artifact_prune_plan.v1"`,
		`"reason": "current"`,
		`"reason": "older_than_keep_recent"`,
		oldArtifact,
		currentArtifact,
	} {
		if !strings.Contains(string(reportData), want) {
			t.Fatalf("report missing %q: %s", want, string(reportData))
		}
	}
	for _, path := range []string{oldArtifact, currentArtifact} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %q was removed by prune plan: %v", path, err)
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExplicitPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--port requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckHelpShowsRequiredCommit(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"firmware-upload-check", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	for _, want := range []string{"--artifact", "--port", "--commit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRequiresExpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--commit requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFirmwareUploadCheckAcceptsExplicitDevicePort(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "abcdef1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "firmware upload dry-run guard ok (no flash performed)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, want := range []string{
		`"guard_id": "a21.firmware.upload_guard.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"port_usage"`,
		`"exists": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunFirmwareUploadCheckRejectsUnexpectedCommit(t *testing.T) {
	dir := t.TempDir()
	manifest := writeTestFirmwareManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	writeFirmwareArtifactWithChecksum(t, artifact, []byte("firmware"))

	var stderr bytes.Buffer
	code := Run([]string{
		"firmware-upload-check",
		"--manifest", manifest,
		"--artifact", artifact,
		"--port", "/dev/cu.usbmodemA21",
		"--commit", "1234567",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "commit") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
