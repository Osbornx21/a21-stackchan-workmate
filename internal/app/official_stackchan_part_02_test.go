package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"a21.local/a21/internal/firmwarecheck"
)

func TestRunStackChanOfficialBaselineFlashExecuteRunsGuardedCommand(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialBaselineFlashCommand
	var ranScript string
	runStackChanOfficialBaselineFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialBaselineFlashCommand = originalRunner
	}()

	buildDir := writeTestOfficialBaselineBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", idfExport,
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_BASELINE_DIAGNOSTIC",
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

func TestRunStackChanOfficialXiaozhiCompatibleFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_plan.v1"`,
		`"dry_run": true`,
		`"flash_allowed": false`,
		`"flash_executed": false`,
		`"esptool_before": "default_reset"`,
		`"build_dir_name": "a21-stackchan-official-build"`,
		`"app"`,
		`"file": "a21-stackchan-official-xiaozhi-compatible.bin"`,
		`"offset": "0x20000"`,
		`"next_required_confirmation": "a21-stackchan-official-xiaozhi-compatible-flash-execute_with_confirmation_token"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) {
		t.Fatalf("plan leaked full build dir: %s", stdout.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashPlanRejectsWrongAppCandidateWithoutPathLeaks(t *testing.T) {
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
		"a21-stackchan-official-xiaozhi-compatible-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("wrong app candidate unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "a21-stackchan-official-xiaozhi-compatible.bin") {
		t.Fatalf("stderr missing required candidate name: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("wrong-app rejection leaked full build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashPlanRejectsOversizedAssetsPartition(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	writeTestOfficialPartitionTable(t, filepath.Join(buildDir, "partition_table", "partition-table.bin"), 0x400000)
	writeTestSizedFile(t, filepath.Join(buildDir, "generated_assets.bin"), 0x400001)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("oversized assets unexpectedly passed: %s", stdout.String())
	}
	for _, want := range []string{"generated assets image size", "exceeds assets partition size"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q: %s", want, stderr.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("oversized-assets rejection leaked full build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashPlanRejectsUnsupportedBeforeMode(t *testing.T) {
	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--esptool-before", "hard_reset",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("unsupported esptool before mode unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unsupported esptool before mode") {
		t.Fatalf("stderr missing unsupported mode finding: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("unsupported-mode rejection leaked full build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashPlanRejectsWaitROMWithoutNoReset(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
		"--wait-rom",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("wait-rom without no_reset unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "wait-rom requires --esptool-before no_reset") {
		t.Fatalf("stderr missing wait-rom/no_reset guard: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("wait-rom rejection leaked full build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashExecuteRequiresConfirmationToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-execute",
		"--port", "/dev/cu.usbmodemA21",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashExecuteRunsGuardedCommand(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialXiaozhiCompatibleFlashCommand
	var ranScript string
	runStackChanOfficialXiaozhiCompatibleFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialXiaozhiCompatibleFlashCommand = originalRunner
	}()

	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", idfExport,
		"--esptool-before", "no_reset",
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"python -m esptool", "--chip esp32s3", "--port '/dev/cu.usbmodemA21'", "--before 'no_reset'", "write_flash @flash_args"} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1"`,
		`"flash_executed": true`,
		`"esptool_before": "no_reset"`,
		`"control_guard"`,
		`"file": "a21-stackchan-official-xiaozhi-compatible.bin"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("execution receipt missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) {
		t.Fatalf("execution receipt leaked full build dir: %s", stdout.String())
	}
}

func TestRunStackChanOfficialXiaozhiCompatibleFlashExecuteCanWaitForManualROM(t *testing.T) {
	allowA21ControlGuardForTest(t)
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runStackChanOfficialXiaozhiCompatibleFlashCommand
	var ranScript string
	runStackChanOfficialXiaozhiCompatibleFlashCommand = func(ctx context.Context, logPath string, script string) error {
		ranScript = script
		return nil
	}
	defer func() {
		runStackChanOfficialXiaozhiCompatibleFlashCommand = originalRunner
	}()

	buildDir := writeTestOfficialXiaozhiCompatibleBuild(t)
	idfExport := filepath.Join(t.TempDir(), "export.sh")
	writeTestFile(t, idfExport, "#!/bin/sh\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"a21-stackchan-official-xiaozhi-compatible-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", idfExport,
		"--esptool-before", "no_reset",
		"--wait-rom",
		"--wait-rom-timeout-seconds", "75",
		"--confirm", "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"A21_FLASH_PORT='/dev/cu.usbmodemA21'",
		"A21_WAIT_ROM_DEADLINE=$((SECONDS + 75))",
		"for candidate in /dev/cu.usbmodem*; do",
		`python -m esptool --chip esp32s3 --port "$candidate" -b 115200 --before no_reset --after no_reset --no-stub chip_id`,
		`ROM candidates checked: ${A21_WAIT_ROM_CANDIDATES[*]}`,
		`Last esptool probe output:`,
		`--port "${A21_FLASH_PORT}"`,
		"--before 'no_reset'",
		"write_flash @flash_args",
	} {
		if !strings.Contains(ranScript, want) {
			t.Fatalf("flash script missing %q: %s", want, ranScript)
		}
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1"`,
		`"wait_rom_download_mode": true`,
		`"wait_rom_timeout_seconds": 75`,
		`"flash_executed": true`,
		`"esptool_before": "no_reset"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("execution receipt missing %q: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) {
		t.Fatalf("execution receipt leaked full build dir: %s", stdout.String())
	}
}

func TestRunXiaozhiFirmwareFlashPlanRejectsProductStackChanWithoutDevMarker(t *testing.T) {
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
	if code == 0 {
		t.Fatalf("unmarked xiaozhi product-looking flash unexpectedly passed: %s", stdout.String())
	}
	for _, want := range []string{
		"product StackChan",
		"a21-stackchan-official-xiaozhi-compatible-flash-execute",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q: %s", want, stderr.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("rejection leaked build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunXiaozhiFirmwareFlashExecuteRejectsProductStackChanWithoutDevMarkerBeforeControlGuard(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()
	originalRunner := runXiaozhiFirmwareFlashCommand
	runXiaozhiFirmwareFlashCommand = func(ctx context.Context, logPath string, script string) error {
		t.Fatalf("generic xiaozhi execute command should not run for unmarked product-looking StackChan app")
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
	if code == 0 {
		t.Fatalf("unmarked xiaozhi product-looking execute unexpectedly passed: %s", stdout.String())
	}
	for _, want := range []string{
		"product StackChan",
		"a21-stackchan-official-xiaozhi-compatible-flash-execute",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q: %s", want, stderr.String())
		}
	}
	if strings.Contains(stdout.String(), buildDir) || strings.Contains(stderr.String(), buildDir) {
		t.Fatalf("execute rejection leaked build dir: stdout=%s stderr=%s", stdout.String(), stderr.String())
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
		"--non-product-dev",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.xiaozhi_firmware_flash_plan.v1"`,
		`"flash_allowed": false`,
		`"flash_executed": false`,
		`"build_lane_role": "non_product_dev"`,
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
		"--non-product-dev",
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
