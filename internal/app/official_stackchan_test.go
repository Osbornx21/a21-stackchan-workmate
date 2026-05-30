package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
)

func TestRunStackChanOfficialBaselinePlansFromCleanGitHead(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, true)
	workDir := filepath.Join(t.TempDir(), "a21-stackchan-official-clean")
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline",
		"--source", source,
		"--work-dir", workDir,
		"--build-dir", buildDir,
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}

	var report stackChanOfficialBaselineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ready" {
		t.Fatalf("status = %q, want ready: %+v", report.Status, report.Findings)
	}
	if report.Execute {
		t.Fatal("plan mode must not execute a build")
	}
	if report.SourceCommit == "" || !report.SourceClean {
		t.Fatalf("source git evidence missing: %+v", report)
	}
	if !report.Evidence.HalMicTestUsesOutputData ||
		!report.Evidence.CoreS3UsesESPCodecDev ||
		!report.Evidence.CoreS3CreatesDuplexChannels ||
		!report.Evidence.ReposDeclareXiaoZhiAudioService {
		t.Fatalf("official audio evidence incomplete: %+v", report.Evidence)
	}
	if strings.Contains(strings.ToLower(stdout.String()), "x21") || strings.Contains(strings.ToLower(stderr.String()), "x21") {
		t.Fatalf("legacy identity leaked stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanOfficialBaselineRejectsMissingCodecEvidence(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline",
		"--source", source,
		"--work-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-clean"),
		"--build-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-build"),
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"missing_official_audio_codec"`) {
		t.Fatalf("stdout missing codec finding: %s", stdout.String())
	}
}

func TestRunStackChanOfficialBaselineNormalizesRelativeOverlayPath(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, true)
	cwd := t.TempDir()
	t.Chdir(cwd)
	relativeOverlay := filepath.Join("overlays", "a21-audio-smoke.patch")
	writeTestFile(t, filepath.Join(cwd, relativeOverlay), "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline",
		"--source", source,
		"--work-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-clean"),
		"--build-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-build"),
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--overlay", relativeOverlay,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stderr.String())
	}

	var report stackChanOfficialBaselineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if len(report.Overlays) != 1 {
		t.Fatalf("overlay count = %d, want 1", len(report.Overlays))
	}
	if !filepath.IsAbs(report.Overlays[0].Path) {
		t.Fatalf("overlay path = %q, want absolute", report.Overlays[0].Path)
	}
}

func TestCollectOfficialStackChanBuildArtifactsFindsAppFromFlashArgs(t *testing.T) {
	buildDir := t.TempDir()
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-audio-smoke.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-audio-smoke.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")

	artifacts := collectOfficialStackChanBuildArtifacts(buildDir)
	var appArtifact stackChanOfficialBaselineBuildArtifact
	for _, artifact := range artifacts {
		if artifact.Name == "app" {
			appArtifact = artifact
			break
		}
	}
	if appArtifact.Path == "" {
		t.Fatalf("app artifact missing: %+v", artifacts)
	}
	if appArtifact.FlashOffset != "0x20000" {
		t.Fatalf("app flash offset = %q, want 0x20000", appArtifact.FlashOffset)
	}
	if !strings.HasSuffix(appArtifact.Path, "a21-stackchan-official-audio-smoke.bin") {
		t.Fatalf("app artifact path = %q", appArtifact.Path)
	}
}

func TestRunStackChanOfficialAudioSmokeFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialAudioSmokeBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-audio-smoke-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"flash_allowed": false`) ||
		!strings.Contains(stdout.String(), `"app"`) ||
		!strings.Contains(stdout.String(), `"0x20000"`) {
		t.Fatalf("flash plan missing no-flash app receipt: %s", stdout.String())
	}
}

func TestRunStackChanOfficialAudioSmokeFlashExecuteRequiresConfirmationToken(t *testing.T) {
	buildDir := writeTestOfficialAudioSmokeBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-audio-smoke-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("execute without confirmation unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanOfficialAudioSmokeFlashExecuteRunsGuardedCommand(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialSmokeFlashCommand
	var ranScript string
	runStackChanOfficialSmokeFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialSmokeFlashCommand = originalRunner
	}()

	buildDir := writeTestOfficialAudioSmokeBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-audio-smoke-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", idfExport,
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"python -m esptool", "--chip esp32s3", "--port '/dev/cu.usbmodemA21'", "write_flash @flash_args"} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	if !strings.Contains(stdout.String(), `"flash_executed": true`) {
		t.Fatalf("execution receipt missing flash_executed: %s", stdout.String())
	}
}

func writeTestOfficialStackChanRepo(t *testing.T, includeCodecEvidence bool) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "firmware", "README.md"), "idf.py build\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "sdkconfig.defaults"), strings.Join([]string{
		`CONFIG_IDF_TARGET="esp32s3"`,
		`CONFIG_LANGUAGE_EN_US=y`,
		`CONFIG_BOARD_TYPE_M5STACK_STACK_CHAN=y`,
		`CONFIG_SEND_WAKE_WORD_DATA=n`,
	}, "\n")+"\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "repos.json"), `[
  {
    "url": "https://github.com/78/xiaozhi-esp32.git",
    "path": "xiaozhi-esp32",
    "branch": "v2.2.4",
    "patch": "patches/xiaozhi-esp32.patch"
  }
]`)

	audioCPP := "void Hal::startMicTest() {}\n"
	codecCC := "class CoreS3AudioCodec {};\n"
	if includeCodecEvidence {
		audioCPP = "void Hal::startMicTest() { audio_codec->OutputData(output_chunk); }\n"
		codecCC = "void CoreS3AudioCodec::CreateDuplexChannels() {}\nvoid CoreS3AudioCodec::EnableOutput() { esp_codec_dev_open(output_dev_, &fs); }\nint CoreS3AudioCodec::Write(const int16_t* data, int samples) { esp_codec_dev_write(output_dev_, (void*)data, samples * sizeof(int16_t)); return samples; }\n"
	}
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "hal", "audio.cpp"), audioCPP)
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "hal", "board", "cores3_audio_codec.cc"), codecCC)
	writeTestFile(t, filepath.Join(dir, "firmware", "xiaozhi-esp32", "main", "audio", "audio_service.cc"), "void AudioService::AudioOutputTask() { codec_->OutputData(task->pcm); }\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "xiaozhi-esp32", "main", "audio", "audio_service.h"), "#define OPUS_FRAME_DURATION_MS 60\n")

	runGitForTest(t, dir, "init")
	runGitForTest(t, dir, "add", ".")
	runGitForTest(t, dir, "-c", "user.name=A21 Test", "-c", "user.email=a21@example.invalid", "commit", "-m", "official baseline")
	return dir
}

func writeTestOfficialAudioSmokeBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-audio-smoke.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-audio-smoke.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(output))
	}
}
