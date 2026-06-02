package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
	"a21.local/a21/internal/runtimeguard"
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

func TestRunStackChanOfficialXiaozhiCompatiblePlanReportsProductCandidateContract(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, true)
	overlay := filepath.Join("firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline",
		"--source", source,
		"--work-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-clean"),
		"--build-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-build"),
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--overlay", overlay,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var report stackChanOfficialBaselineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.FirmwareCandidate != "a21-stackchan-official-xiaozhi-compatible" {
		t.Fatalf("firmware candidate = %q", report.FirmwareCandidate)
	}
	if !report.OfficialAvatarActionPreserved || !report.OfficialXiaozhiStartPreserved || report.MinimalBridgeScreen {
		t.Fatalf("product candidate contract not preserved: avatar=%v xiaozhi=%v minimal_bridge=%v\n%s",
			report.OfficialAvatarActionPreserved,
			report.OfficialXiaozhiStartPreserved,
			report.MinimalBridgeScreen,
			stdout.String())
	}
	if strings.Contains(stdout.String(), "a21-stackchan-official-pcm-bridge") {
		t.Fatalf("product candidate report should not identify as old pcm bridge: %s", stdout.String())
	}
}

func TestApplyStackChanOfficialCandidateContractKeepsXiaozhiCompatibleAfterExecute(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, true)
	overlay := filepath.Join("firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	report := stackChanOfficialBaselineReport{}

	applyStackChanOfficialCandidateContract(&report, source, []string{overlay})

	if report.FirmwareCandidate != "a21-stackchan-official-xiaozhi-compatible" ||
		report.BuildLaneRole != "product_candidate" ||
		!report.OfficialAvatarActionPreserved ||
		!report.OfficialXiaozhiStartPreserved ||
		report.MinimalBridgeScreen {
		t.Fatalf("execute candidate contract = %+v", report)
	}
}

func TestRunStackChanOfficialPCMBridgePlanReportsDiagnosticContract(t *testing.T) {
	source := writeTestOfficialStackChanRepo(t, true)
	overlay := filepath.Join("firmware", "stackchan-official", "overlays", "a21-official-pcm-bridge.patch")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline",
		"--source", source,
		"--work-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-clean"),
		"--build-dir", filepath.Join(t.TempDir(), "a21-stackchan-official-build"),
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--overlay", overlay,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var report stackChanOfficialBaselineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.FirmwareCandidate != "a21-stackchan-official-pcm-bridge" || report.BuildLaneRole != "diagnostic_m3_prep" {
		t.Fatalf("candidate/role = %q/%q", report.FirmwareCandidate, report.BuildLaneRole)
	}
	if report.OfficialAvatarActionPreserved || report.OfficialXiaozhiStartPreserved || !report.MinimalBridgeScreen {
		t.Fatalf("pcm bridge should remain diagnostic, not product candidate: avatar=%v xiaozhi=%v minimal_bridge=%v\n%s",
			report.OfficialAvatarActionPreserved,
			report.OfficialXiaozhiStartPreserved,
			report.MinimalBridgeScreen,
			stdout.String())
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

func TestCollectOfficialStackChanBuildArtifactsFindsPCMBridgeAppFromFlashArgs(t *testing.T) {
	buildDir := t.TempDir()
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-pcm-bridge.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-pcm-bridge.bin",
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
	if !strings.HasSuffix(appArtifact.Path, "a21-stackchan-official-pcm-bridge.bin") {
		t.Fatalf("app artifact path = %q", appArtifact.Path)
	}
}

func TestCollectOfficialStackChanBuildArtifactsFindsXiaozhiCompatibleAppFromFlashArgs(t *testing.T) {
	buildDir := t.TempDir()
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-xiaozhi-compatible.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-xiaozhi-compatible.bin",
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
	if !strings.HasSuffix(appArtifact.Path, "a21-stackchan-official-xiaozhi-compatible.bin") {
		t.Fatalf("app artifact path = %q", appArtifact.Path)
	}
}

func TestCollectOfficialStackChanBuildArtifactsFindsPCMBridgeFallbackApp(t *testing.T) {
	buildDir := t.TempDir()
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-pcm-bridge.bin"), "app")

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
	if !strings.HasSuffix(appArtifact.Path, "a21-stackchan-official-pcm-bridge.bin") {
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
	allowA21ControlGuardForTest(t)
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
	if !strings.Contains(stdout.String(), `"control_guard"`) {
		t.Fatalf("execution receipt missing control guard evidence: %s", stdout.String())
	}
}

func TestRunStackChanOfficialPCMBridgeFlashPlanRequiresAudioWSURL(t *testing.T) {
	buildDir := writeTestOfficialPCMBridgeBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("plan without audio ws url unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--audio-ws-url is required") {
		t.Fatalf("stderr missing audio ws url requirement: %s", stderr.String())
	}
}

func TestRunStackChanOfficialPCMBridgeFlashPlanBuildsRedactedNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialPCMBridgeBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_pcm_bridge_flash_plan.v1"`,
		`"flash_allowed": false`,
		`"app"`,
		`a21-stackchan-official-pcm-bridge.bin`,
		`"host": "127.0.0.1:21080"`,
		`"path": "/ws/audio"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001") {
		t.Fatalf("plan leaked full audio ws url: %s", stdout.String())
	}
}

func TestRunStackChanOfficialPCMBridgeFlashExecuteRequiresConfirmation(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-flash-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanOfficialPCMBridgeFlashExecuteRunsGuardedCommand(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialPCMBridgeFlashCommand
	var ranScript string
	runStackChanOfficialPCMBridgeFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialPCMBridgeFlashCommand = originalRunner
	}()

	buildDir := writeTestOfficialPCMBridgeBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", idfExport,
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"python -m esptool", "--chip esp32s3", "--port '/dev/cu.usbmodemA21'", "write_flash @flash_args"} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_pcm_bridge_flash_execution.v1"`,
		`"flash_executed": true`,
		`"control_guard"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("execution receipt missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001") {
		t.Fatalf("execution receipt leaked full audio ws url: %s", stdout.String())
	}
}

func TestRunXiaozhiFirmwareFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestXiaozhiFirmwareBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-firmware-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_firmware_flash_plan.v1"`,
		`"flash_allowed": false`,
		`"flash_executed": false`,
		`"app"`,
		`"xiaozhi.bin"`,
		`"offset": "0x800000"`,
		`"next_required_confirmation": "xiaozhi-firmware-flash-execute_with_confirmation_token"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan missing %q: %s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{buildDir, "x21", "X21", `"token":`, "secret"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("plan leaked forbidden value %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunXiaozhiFirmwareFlashExecuteRequiresConfirmation(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-firmware-flash-execute",
		"--port", "/dev/cu.usbmodemA21",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_XIAOZHI_FIRMWARE") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunXiaozhiFirmwareFlashPlanRejectsLegacyDeviceEvents(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestXiaozhiFirmwareBuild(t)
	writeTestFile(t, filepath.Join(buildDir, "config", "sdkconfig.json"), `{
  "BOARD_TYPE_M5STACK_CORE_S3": true,
  "OTA_URL": "http://192.0.2.10:21080/xiaozhi/ota/",
  "ENABLE_X21_DEVICE_EVENTS": true
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-firmware-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("legacy-enabled plan unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "forbidden legacy identity") {
		t.Fatalf("stderr missing legacy rejection: %s", stderr.String())
	}
}

func TestRunXiaozhiFirmwareFlashPlanRejectsFrozenLegacyFirmwareSourceWithoutPathLeaks(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestXiaozhiFirmwareBuildAt(t, frozenLegacyXiaozhiFirmwareBuildDirFixture(t))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-firmware-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("frozen legacy source unexpectedly passed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout = %q, want empty frozen-source rejection", stdout.String())
	}
	if !strings.Contains(stderr.String(), "build dir belongs to frozen external firmware source") {
		t.Fatalf("stderr missing frozen-source rejection: %s", stderr.String())
	}
	assertNoFrozenLegacyXiaozhiFirmwareSourceLeak(t, buildDir, stdout.String(), stderr.String())
}

func TestRunXiaozhiFirmwareFlashExecuteRunsGuardedCommand(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runXiaozhiFirmwareFlashCommand
	var ranScript string
	runXiaozhiFirmwareFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runXiaozhiFirmwareFlashCommand = originalRunner
	}()

	buildDir := writeTestXiaozhiFirmwareBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"xiaozhi-firmware-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", idfExport,
		"--confirm", "WRITE_A21_XIAOZHI_FIRMWARE",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"python -m esptool", "--chip esp32s3", "--port '/dev/cu.usbmodemA21'", "write_flash @flash_args"} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_firmware_flash_execution.v1"`,
		`"flash_executed": true`,
		`"control_guard"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("execution receipt missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) {
		t.Fatalf("execution receipt leaked build dir: %s", stdout.String())
	}
}

func TestRunStackChanOfficialPCMBridgeNVSPlanBuildsRedactedNoWriteReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-plan",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_pcm_bridge_nvs_plan.v1"`,
		`"dry_run": true`,
		`"write_allowed": false`,
		`"write_executed": false`,
		`"offset": "0x9000"`,
		`"size_hex": "0x4000"`,
		`"preserve_existing_entries": true`,
		`"only_mutates_a21_namespace": true`,
		`"next_required_confirmation": "stackchan-official-pcm-bridge-nvs-execute_with_confirmation_token"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("nvs plan missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001") {
		t.Fatalf("nvs plan leaked full audio ws url: %s", stdout.String())
	}
}

func TestRunStackChanOfficialPCMBridgeNVSExecuteRequiresConfirmationToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanOfficialPCMBridgeNVSExecuteStopsWhenControlGuardBlocks(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalGuard := runA21ControlGuard
	runA21ControlGuard = func(ctx context.Context, input runtimeguard.ControlGuardInput) runtimeguard.ControlGuardReport {
		return runtimeguard.ControlGuardReport{
			Command: input.Command,
			Tier:    "T7",
			Result: runtimeguard.NewResult([]runtimeguard.Finding{{
				Code:     "control_hardware_window_branch_required",
				Severity: runtimeguard.SeverityBlock,
				Message:  "A21 hardware writes require a hardware-window branch",
			}}),
		}
	}
	defer func() {
		runA21ControlGuard = originalGuard
	}()
	originalRunner := runStackChanOfficialPCMBridgeNVSCommand
	var scripts []string
	runStackChanOfficialPCMBridgeNVSCommand = func(ctx context.Context, logPath string, script string) error {
		scripts = append(scripts, script)
		return nil
	}
	defer func() {
		runStackChanOfficialPCMBridgeNVSCommand = originalRunner
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if len(scripts) != 0 {
		t.Fatalf("scripts ran despite control guard block: %v", scripts)
	}
	if !strings.Contains(stderr.String(), "control_hardware_window_branch_required") {
		t.Fatalf("stderr missing control guard finding: %s", stderr.String())
	}
}

func TestRunStackChanOfficialPCMBridgeNVSExecuteRunsGuardedReadGenerateWriteFlow(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialPCMBridgeNVSCommand
	var scripts []string
	runStackChanOfficialPCMBridgeNVSCommand = func(ctx context.Context, logPath string, script string) error {
		scripts = append(scripts, script)
		switch {
		case strings.Contains(script, " read_flash "):
			writeTestFile(t, lastSingleQuotedPath(script), "backup")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "before-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForBridgeProvision("old-device", "ws://old/ws/audio?device_id=old-device"))
		case strings.Contains(script, "nvs_partition_generator/nvs_partition_gen.py"):
			writeTestFile(t, lastSingleQuotedPath(script), "provisioned")
		case strings.Contains(script, "nvs_partition_tool/nvs_tool.py") && strings.Contains(script, "provision-"):
			writeTestFile(t, redirectSingleQuotedPath(script), testNVSJSONForBridgeProvision("stackchan-001", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001"))
		}
		return nil
	}
	defer func() {
		runStackChanOfficialPCMBridgeNVSCommand = originalRunner
	}()

	idfRoot := filepath.Join(t.TempDir(), "esp-idf-v5.5.2")
	idfExport := filepath.Join(idfRoot, "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_tool", "nvs_tool.py"), "#!/usr/bin/env python\n")
	writeTestFile(t, filepath.Join(idfRoot, "components", "nvs_flash", "nvs_partition_generator", "nvs_partition_gen.py"), "#!/usr/bin/env python\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-pcm-bridge-nvs-execute",
		"--port", "/dev/cu.usbmodemA21",
		"--device-id", "stackchan-001",
		"--audio-ws-url", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
		"--idf-export", idfExport,
		"--run-dir", filepath.Join(t.TempDir(), "a21-official-nvs-run"),
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if len(scripts) != 5 {
		t.Fatalf("scripts = %d, want 5: %v", len(scripts), scripts)
	}
	joined := strings.Join(scripts, "\n")
	for _, want := range []string{
		"read_flash 0x9000 0x4000",
		"nvs_partition_tool/nvs_tool.py",
		"nvs_partition_generator/nvs_partition_gen.py",
		"write_flash 0x9000",
		"--after hard_reset",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("scripts missing %q:\n%s", want, joined)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_pcm_bridge_nvs_execution.v1"`,
		`"write_allowed": true`,
		`"write_executed": true`,
		`"control_guard"`,
		`"preserved_entry_count": 5`,
		`"mutated_entry_count": 2`,
		`"servo_calibration_present": true`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001") ||
		strings.Contains(stdout.String(), "existing-secret") {
		t.Fatalf("execution report leaked sensitive values: %s", stdout.String())
	}
}

func TestOfficialPCMBridgeNVSCSVPreservesExistingEntriesAndOnlyOverwritesA21Keys(t *testing.T) {
	entries := []stackChanNVSMinimalEntry{
		{Namespace: "board", Key: "uuid", Encoding: "string", Data: "device-uuid", State: "Written"},
		{Namespace: "wifi", Key: "ssid", Encoding: "string", Data: "existing-wifi", State: "Written"},
		{Namespace: "wifi", Key: "password", Encoding: "string", Data: "existing-secret", State: "Written"},
		{Namespace: "servo", Key: "zero_pos_1", Encoding: "int32_t", Data: float64(460), State: "Written"},
		{Namespace: "servo", Key: "zero_pos_2", Encoding: "int32_t", Data: float64(620), State: "Written"},
		{Namespace: "a21", Key: "device_id", Encoding: "string", Data: "old-device", State: "Written"},
		{Namespace: "a21", Key: "audio_ws_url", Encoding: "string", Data: "ws://old/ws/audio?device_id=old-device", State: "Written"},
	}

	var csv bytes.Buffer
	summary, err := writeOfficialPCMBridgeNVSCSV(&csv, entries, "stackchan-001", "ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001")
	if err != nil {
		t.Fatalf("write csv: %v", err)
	}
	text := csv.String()
	for _, want := range []string{
		"board,namespace,,",
		"uuid,data,string,device-uuid",
		"wifi,namespace,,",
		"ssid,data,string,existing-wifi",
		"password,data,string,existing-secret",
		"servo,namespace,,",
		"zero_pos_1,data,i32,460",
		"zero_pos_2,data,i32,620",
		"a21,namespace,,",
		"device_id,data,string,stackchan-001",
		"audio_ws_url,data,string,ws://127.0.0.1:21080/ws/audio?device_id=stackchan-001",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("csv missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "old-device") {
		t.Fatalf("csv retained stale a21 value:\n%s", text)
	}
	if summary.PreservedEntryCount != 5 || summary.MutatedEntryCount != 2 || !summary.ServoCalibrationPresent {
		t.Fatalf("summary = %+v", summary)
	}
}

func lastSingleQuotedPath(text string) string {
	end := strings.LastIndex(text, "'")
	if end <= 0 {
		return ""
	}
	start := strings.LastIndex(text[:end], "'")
	if start < 0 {
		return ""
	}
	return text[start+1 : end]
}

func redirectSingleQuotedPath(text string) string {
	redirect := strings.LastIndex(text, "> ")
	if redirect < 0 {
		return ""
	}
	return lastSingleQuotedPath(text[redirect:])
}

func testNVSJSONForBridgeProvision(deviceID string, audioWSURL string) string {
	return fmt.Sprintf(`[
  {"namespace":"board","key":"uuid","encoding":"string","data":"device-uuid","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"ssid","encoding":"string","data":"existing-wifi","state":"Written","is_empty":false},
  {"namespace":"wifi","key":"password","encoding":"string","data":"existing-secret","state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_1","encoding":"int32_t","data":460,"state":"Written","is_empty":false},
  {"namespace":"servo","key":"zero_pos_2","encoding":"int32_t","data":620,"state":"Written","is_empty":false},
  {"namespace":"a21","key":"device_id","encoding":"string","data":%q,"state":"Written","is_empty":false},
  {"namespace":"a21","key":"audio_ws_url","encoding":"string","data":%q,"state":"Written","is_empty":false}
]`, deviceID, audioWSURL)
}

func writeTestOfficialPCMBridgeNVSExecutionReport(t *testing.T, deviceID string, host string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a21-stackchan-official-pcm-bridge-nvs-execution.json")
	report := stackChanOfficialPCMBridgeNVSReport{
		SchemaVersion: stackChanOfficialPCMBridgeNVSExecutionSchema,
		Status:        "passed",
		DryRun:        false,
		WriteAllowed:  true,
		WriteExecuted: true,
		Port:          "/dev/cu.usbmodemA21",
		DeviceID:      deviceID,
		AudioWS: stackChanOfficialPCMBridgeAudioWS{
			Scheme:        "ws",
			Host:          host,
			Path:          "/ws/audio",
			DeviceIDQuery: true,
		},
		Partition: stackChanOfficialPCMBridgeNVSPartition{
			Offset:    stackChanOfficialPCMBridgeNVSOffset,
			SizeHex:   stackChanOfficialPCMBridgeNVSSizeHex,
			SizeBytes: stackChanOfficialPCMBridgeNVSSizeBytes,
		},
		Safety: stackChanOfficialPCMBridgeNVSSafety{
			BackupBeforeWrite:       true,
			PreserveExistingEntries: true,
			OnlyMutatesA21Namespace: true,
			ReportRedactsValues:     true,
		},
		Summary: &stackChanOfficialPCMBridgeNVSSummary{
			PreservedEntryCount:     39,
			MutatedEntryCount:       2,
			ServoCalibrationPresent: true,
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal nvs report: %v", err)
	}
	writeTestFile(t, path, string(data))
	return path
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
	writeTestFile(t, filepath.Join(dir, "firmware", "CMakeLists.txt"), strings.Join([]string{
		`cmake_minimum_required(VERSION 3.16)`,
		`set(PROJECT_VER "1.4.1")`,
		`add_definitions(-DFIRMWARE_VERSION="${PROJECT_VER}")`,
		`include($ENV{IDF_PATH}/tools/cmake/project.cmake)`,
		`project(stack-chan)`,
	}, "\n")+"\n")
	writeTestFile(t, filepath.Join(dir, "firmware", "main", "main.cpp"), strings.Join([]string{
		`#include <smooth_ui_toolkit.hpp>`,
		`#include <uitk/short_namespace.hpp>`,
		`#include <mooncake.h>`,
		`#include <apps/apps.h>`,
		`#include <hal/hal.h>`,
		`extern "C" void app_main(void)`,
		`{`,
		`    GetHAL().init();`,
		`    GetMooncake().installApp(std::make_unique<AppLauncher>());`,
		`    GetMooncake().installApp(std::make_unique<AppAiAgent>());`,
		`    GetMooncake().installApp(std::make_unique<AppAvatar>());`,
		`    while (1) {`,
		`        GetMooncake().update();`,
		`        if (GetHAL().isXiaozhiStartRequested()) {`,
		`            break;`,
		`        }`,
		`    }`,
		`    GetHAL().startXiaozhi();`,
		`}`,
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

func writeTestOfficialPCMBridgeBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-stackchan-official-build")
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "a21-stackchan-official-pcm-bridge.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 a21-stackchan-official-pcm-bridge.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0xa00000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func writeTestXiaozhiFirmwareBuild(t *testing.T) string {
	t.Helper()
	buildDir := filepath.Join(t.TempDir(), "a21-xiaozhi-firmware-build")
	return writeTestXiaozhiFirmwareBuildAt(t, buildDir)
}

func writeTestXiaozhiFirmwareBuildAt(t *testing.T, buildDir string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(buildDir, "bootloader", "bootloader.bin"), "boot")
	writeTestFile(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), "part")
	writeTestFile(t, filepath.Join(buildDir, "ota_data_initial.bin"), "ota")
	writeTestFile(t, filepath.Join(buildDir, "generated_assets.bin"), "assets")
	writeTestFile(t, filepath.Join(buildDir, "xiaozhi.bin"), "app")
	writeTestFile(t, filepath.Join(buildDir, "config", "sdkconfig.json"), `{
  "BOARD_TYPE_M5STACK_CORE_S3": true,
  "OTA_URL": "http://192.0.2.10:21080/xiaozhi/ota/",
  "ENABLE_X21_DEVICE_EVENTS": false
}`)
	writeTestFile(t, filepath.Join(buildDir, "flash_args"), strings.Join([]string{
		"--flash_mode dio --flash_freq 80m --flash_size 16MB",
		"0x0 bootloader/bootloader.bin",
		"0x20000 xiaozhi.bin",
		"0x8000 partition_table/partition-table.bin",
		"0xd000 ota_data_initial.bin",
		"0x800000 generated_assets.bin",
	}, "\n")+"\n")
	return buildDir
}

func frozenLegacyXiaozhiFirmwareBuildDirFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "小马暴力", "sources", "xiaozhi-esp32", "build-m5stack-core-s3")
}

func assertNoFrozenLegacyXiaozhiFirmwareSourceLeak(t *testing.T, buildDir string, outputs ...string) {
	t.Helper()
	for _, output := range outputs {
		for _, forbidden := range []string{
			buildDir,
			filepath.Dir(buildDir),
			filepath.Dir(filepath.Dir(buildDir)),
			"小马暴力",
			"xiaozhi-esp32",
		} {
			if forbidden != "" && strings.Contains(output, forbidden) {
				t.Fatalf("frozen firmware source rejection leaked %q: %s", forbidden, output)
			}
		}
	}
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
