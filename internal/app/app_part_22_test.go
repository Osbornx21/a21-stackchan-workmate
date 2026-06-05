package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
)

func TestRunStackChanPhysicalEvidenceRejectsLegacyPassWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	identityPath := writeTestStackChanIdentityAcceptanceReport(t, dir, strings.Repeat("e", 64))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-physical-evidence",
		"--identity-acceptance", identityPath,
		"--device-id", "stackchan-001",
		"--commit", "abcdef1",
		"--pass", "microphone=x21_probe",
		"--output-dir", filepath.Join(dir, "reports"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr = %q, want legacy rejection", stderr.String())
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21_probe") {
		t.Fatalf("legacy pass leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func writeTestStackChanIdentityAcceptanceReport(t *testing.T, dir string, artifactSHA string) string {
	t.Helper()
	identityPath := filepath.Join(dir, "a21-stackchan-identity-acceptance.json")
	artifactPath := filepath.Join(dir, "artifacts", "a21-stackchan-0.1.0-m5stack-cores3-abcdef1-20260530-004500.bin")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identityPath, []byte(`{
  "schema_version": "a21.stackchan_identity_acceptance.v1",
  "dry_run": true,
  "flash_allowed": false,
  "delete_allowed": false,
  "hardware_acceptance_scope": "identity_only",
  "identity_acceptance_status": "identity_confirmed",
  "device_id": "stackchan-001",
  "commit": "abcdef1",
  "artifact_path": "`+artifactPath+`",
  "artifact_sha256": "`+artifactSHA+`",
  "device_identity": {
    "device_identity_confirmed": true,
    "device": {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": "workmate",
      "current_expression": "idle",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef1"
      },
      "capabilities": {
        "microphone": "available",
        "speaker": "available",
        "screen": "available",
        "screen_touch": "available",
        "top_touch": "available",
        "servo_y": "available",
        "rgb": "available"
      },
      "last_seen_ms": 1780000000000
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return identityPath
}

func writeTestFirmwareManifest(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
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
	return manifest
}

func writeTestGatewayDeviceReport(t *testing.T, reportPath string, content string) {
	t.Helper()
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") && !strings.Contains(content, `"schema_version"`) {
		content = strings.TrimPrefix(content, "{")
		content = `{
  "schema_version": "a21.gateway.devices.v1",
  "service": "a21-gateway",` + content
	}
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newestGlob(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no files match %q", pattern)
	}
	return matches[len(matches)-1]
}

func testPlatformIOConfig(board string) string {
	return `[platformio]
default_envs = a21_stackchan_cores3

[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = ` + board + `
framework = arduino
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  bblanchon/ArduinoJson @ 7.4.3
`
}

func writeFirmwareArtifactWithChecksum(t *testing.T, artifactPath string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	commit := testArtifactCommitFromName(t, artifactPath)
	content = append(content, []byte("\na21-stackchan\n0.1.0\nm5stack-cores3\n"+commit+"\n")...)
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])
	if err := os.WriteFile(artifactPath+".sha256", []byte(checksum+"  "+filepath.Base(artifactPath)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := firmwarecheck.ReleaseIndexEntry{
		SchemaVersion: "a21.firmware.release.v1",
		FirmwareID:    "a21-stackchan",
		Version:       "0.1.0",
		Board:         "m5stack-cores3",
		Commit:        commit,
		Timestamp:     testArtifactTimestampFromName(t, artifactPath),
		ArtifactPath:  artifactPath,
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}
	manifestData, err := json.MarshalIndent(firmwarecheck.FirmwareReleaseManifest{
		SchemaVersion: firmwarecheck.ReleaseManifestSchemaVersion,
		Project:       "A21",
		FirmwareID:    entry.FirmwareID,
		Version:       entry.Version,
		Board:         entry.Board,
		Commit:        entry.Commit,
		Timestamp:     entry.Timestamp,
		ArtifactPath:  artifactPath,
		ArtifactName:  filepath.Base(artifactPath),
		SHA256Path:    artifactPath + ".sha256",
		SHA256:        checksum,
		Build:         testFirmwareBuildProvenance(artifactPath),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath+".manifest.json", append(manifestData, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(filepath.Dir(artifactPath), firmwarecheck.ReleaseIndexFileName)
	file, err := os.OpenFile(indexPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func writeTestBootstrapFlashImages(t *testing.T, dir string) (string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "build")
	coreDir := filepath.Join(dir, "platformio-core")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 boot app"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir
}

func writeTestMicProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_mic_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 mic probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 mic probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 mic probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_m5unified_i2s_capture\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func writeTestIMUProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_imu_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 IMU probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 IMU probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 IMU probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_m5unified_imu\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func writeTestSensorProbeFlashImages(t *testing.T, dir string, commit string) (string, string, string) {
	t.Helper()
	buildDir := filepath.Join(dir, "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3_sensor_probe")
	coreDir := filepath.Join(dir, "platformio-core")
	artifact := filepath.Join(buildDir, "firmware.bin")
	for path, content := range map[string][]byte{
		filepath.Join(buildDir, "bootloader.bin"): []byte("a21 sensor probe bootloader"),
		filepath.Join(buildDir, "partitions.bin"): []byte("a21 sensor probe partitions"),
		filepath.Join(coreDir, "packages", "framework-arduinoespressif32", "tools", "partitions", "boot_app0.bin"): []byte("a21 sensor probe boot app"),
		artifact: []byte("a21-stackchan\n0.1.0\nm5stack-cores3\n" + commit + "\ndiagnostic_probe_ltr553_ambient_light\ndiagnostic_probe_ltr553_proximity\ndiagnostic_probe_ina226_battery\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return buildDir, coreDir, artifact
}

func readTestSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		t.Fatalf("sha256 file %q is empty", path)
	}
	return fields[0]
}

func testFirmwareBuildProvenance(artifactPath string) firmwarecheck.FirmwareBuildProvenance {
	return firmwarecheck.FirmwareBuildProvenance{
		BuildSystem:     "platformio",
		PlatformIOEnv:   "a21_stackchan_cores3",
		PlatformIOBoard: "m5stack-cores3",
		SourcePath:      filepath.Join(filepath.Dir(artifactPath), "firmware", "stackchan", ".pio", "build", "a21_stackchan_cores3", "firmware.bin"),
		SourceName:      "firmware.bin",
	}
}

func writeNamespaceAuditGitScript(t *testing.T, dir string, files string) {
	t.Helper()
	path := filepath.Join(dir, "git")
	content := "#!/bin/sh\nif [ \"$1\" = \"ls-files\" ]; then\ncat <<'EOF'\n" + files + "EOF\nelse\nexit 2\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

type promotionGitScriptOptions struct {
	remoteNames            string
	targetBranchConfigured bool
	missingAncestor        string
}

func writePromotionReadinessGitScript(t *testing.T, dir string, options promotionGitScriptOptions) {
	t.Helper()
	path := filepath.Join(dir, "git")
	mainExit := "1"
	masterExit := "1"
	remoteNames := options.remoteNames
	if options.targetBranchConfigured {
		mainExit = "0"
	}
	content := `#!/bin/sh
if [ "$1" = "-C" ]; then
  shift
  shift
fi
case "$1 $2 $3" in
  "rev-parse --abbrev-ref HEAD")
    echo "codex/a21-integration-governance-slices"
    exit 0
    ;;
  "rev-parse --short=12 HEAD")
    echo "abcdef123456"
    exit 0
    ;;
  "status --porcelain --untracked-files=all")
    exit 0
    ;;
  "remote  ")
    cat <<'EOF'
` + remoteNames + `EOF
    exit 0
    ;;
  "show-ref --verify --quiet")
    if [ "$4" = "refs/heads/main" ]; then
      exit ` + mainExit + `
    fi
    if [ "$4" = "refs/heads/master" ]; then
      exit ` + masterExit + `
    fi
    exit 1
    ;;
  "merge-base --is-ancestor "*)
    if [ "$3" = "` + options.missingAncestor + `" ]; then
      exit 1
    fi
    exit 0
    ;;
esac
exit 2
`
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func testArtifactCommitFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		t.Fatalf("artifact name %q lacks commit field", name)
	}
	return parts[len(parts)-3]
}

func testArtifactTimestampFromName(t *testing.T, artifactPath string) string {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(artifactPath), ".bin")
	parts := strings.Split(name, "-")
	if len(parts) < 5 {
		t.Fatalf("artifact name %q lacks timestamp field", name)
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}

func writeAppTestWAV(t *testing.T, path string, sampleRate int, pcm []byte) {
	t.Helper()
	byteRate := sampleRate * 2
	blockAlign := 2
	header := []byte{
		'R', 'I', 'F', 'F',
		0, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0,
		1, 0,
		byte(sampleRate), byte(sampleRate >> 8), byte(sampleRate >> 16), byte(sampleRate >> 24),
		byte(byteRate), byte(byteRate >> 8), byte(byteRate >> 16), byte(byteRate >> 24),
		byte(blockAlign), byte(blockAlign >> 8),
		16, 0,
		'd', 'a', 't', 'a',
		byte(len(pcm)), byte(len(pcm) >> 8), byte(len(pcm) >> 16), byte(len(pcm) >> 24),
	}
	riffSize := uint32(len(header) - 8 + len(pcm))
	header[4] = byte(riffSize)
	header[5] = byte(riffSize >> 8)
	header[6] = byte(riffSize >> 16)
	header[7] = byte(riffSize >> 24)
	if err := os.WriteFile(path, append(header, pcm...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appTestPCM16Base64(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}
