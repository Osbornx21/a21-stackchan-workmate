package app

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestHydrateStackChanOfficialDependenciesFromCacheCopiesPinnedRepos(t *testing.T) {
	workDir := t.TempDir()
	cacheRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workDir, "firmware", "repos.json"), `[
  {
    "url": "https://github.com/Forairaaaaa/mooncake.git",
    "path": "components/mooncake",
    "branch": "v2.3.3",
    "patch": "patches/mooncake.patch"
  }
]`)
	writeTestFile(t, filepath.Join(workDir, "firmware", "patches", "mooncake.patch"), strings.Join([]string{
		"diff --git a/include/mooncake.h b/include/mooncake.h",
		"--- a/include/mooncake.h",
		"+++ b/include/mooncake.h",
		"@@ -1 +1 @@",
		"-moon",
		"+patched moon",
	}, "\n")+"\n")
	repoDir := filepath.Join(cacheRoot, "firmware", "components", "mooncake")
	writeTestFile(t, filepath.Join(repoDir, "include", "mooncake.h"), "moon\n")
	runGitForTest(t, repoDir, "init")
	runGitForTest(t, repoDir, "add", ".")
	runGitForTest(t, repoDir, "-c", "user.name=A21 Test", "-c", "user.email=a21@example.invalid", "commit", "-m", "cache repo")
	runGitForTest(t, repoDir, "tag", "v2.3.3")

	if err := hydrateStackChanOfficialDependenciesFromCache(workDir, cacheRoot); err != nil {
		t.Fatalf("hydrate dependency cache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "firmware", "components", "mooncake", "include", "mooncake.h")); err != nil {
		t.Fatalf("cached dependency file missing: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(workDir, "firmware", "components", "mooncake", "include", "mooncake.h"))
	if err != nil {
		t.Fatalf("read cached dependency file: %v", err)
	}
	if string(data) != "patched moon\n" {
		t.Fatalf("cached dependency patch was not applied: %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(workDir, "firmware", "components", "mooncake", ".git")); !os.IsNotExist(err) {
		t.Fatalf("dependency cache copy must not include .git, stat err=%v", err)
	}
}

func TestHydrateStackChanOfficialDependenciesFromCacheRejectsWrongRef(t *testing.T) {
	workDir := t.TempDir()
	cacheRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workDir, "firmware", "repos.json"), `[
  {
    "url": "https://github.com/Forairaaaaa/mooncake.git",
    "path": "components/mooncake",
    "branch": "v2.3.3"
  }
]`)
	repoDir := filepath.Join(cacheRoot, "firmware", "components", "mooncake")
	writeTestFile(t, filepath.Join(repoDir, "include", "mooncake.h"), "moon")
	runGitForTest(t, repoDir, "init")
	runGitForTest(t, repoDir, "add", ".")
	runGitForTest(t, repoDir, "-c", "user.name=A21 Test", "-c", "user.email=a21@example.invalid", "commit", "-m", "cache repo")
	runGitForTest(t, repoDir, "tag", "v2.3.2")

	err := hydrateStackChanOfficialDependenciesFromCache(workDir, cacheRoot)
	if err == nil || !strings.Contains(err.Error(), `resolve ref "v2.3.3"`) {
		t.Fatalf("hydrate error = %v, want ref mismatch", err)
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

func TestOfficialXiaozhiCompatibleOverlayPreservesOfficialLauncherEntryLifecycle(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, forbidden := range []string{
		`A21_OEM_AUTOSTART_XIAOZHI`,
		`diff --git a/firmware/main/main.cpp b/firmware/main/main.cpp`,
		`a21_start_avatar_relay_task`,
		`A21 starting Xiaozhi mode directly after official apps preload`,
		`+    GetHAL().startXiaozhi();`,
		`+    GetHAL().requestXiaozhiStart();`,
		`+        GetHAL().feedTheDog();`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("official Xiaozhi-compatible overlay must preserve official launcher/setup lifecycle; found %q", forbidden)
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlayUsesOfficialFrontendThenA21AgentRuntime(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`CONFIG_OTA_URL="http://47.103.57.217/xiaozhi/ota/"`,
		`CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL="ws://47.103.57.217"`,
		`return CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL;`,
		`/stackChan/ws?deviceType=StackChan&device_id={}`,
		`CONFIG_USE_HOTSPOT_WIFI_PROVISIONING=y`,
		`CONFIG_A21_STACKCHAN_KEEP_CONTROL_CHANNEL=y`,
		`CONFIG_A21_PRODUCT_PLAYBACK_EVENTS=y`,
		`CONFIG_A21_PRODUCT_TOUCH_EVENTS=y`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official frontend to A21 agent runtime contract missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`diff --git a/firmware/main/main.cpp b/firmware/main/main.cpp`,
		`A21_OEM_AUTOSTART_XIAOZHI`,
		`CONFIG_X21_STACKCHAN_AUTO_START_XIAOZHI=y`,
		`diff --git a/firmware/main/hal/hal_ble.cpp b/firmware/main/hal/hal_ble.cpp`,
		`diff --git a/firmware/main/apps/app_setup/`,
		`+        WriteReg(0x27, 0x10);`,
		`AddAuth(ssid, password)`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("official frontend/Wi-Fi/PMIC timing must remain official before AI.AGENT entry; found %q", forbidden)
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlayPreservesOfficialAvatarRelayWorkerWithA21DeviceId(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`config A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL`,
		`CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL="ws://47.103.57.217"`,
		`#include "secret_logic.h"`,
		`return CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL;`,
		`GetHAL().getFactoryMacString(":")`,
		`escaped_device_id += "%3A";`,
		`escaped_device_id`,
		`device_id={}`,
		`CONFIG_ESP_SYSTEM_EVENT_TASK_STACK_SIZE=8192`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing official avatar relay runtime contract %q", required)
		}
	}
	for _, forbidden := range []string{
		`startA21WebSocketAvatarRuntime`,
		`updateA21WebSocketAvatarRuntime`,
		`_a21_avatar_runtime`,
		`start A21 direct websocket avatar runtime`,
		`A21 direct avatar runtime`,
		`a21_start_avatar_relay_task`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("official Xiaozhi-compatible overlay must use the official AppAvatar worker; found %q", forbidden)
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlayPreservesStackChanPmicStartupParity(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`diff --git a/firmware/main/hal/board/stackchan.cc b/firmware/main/hal/board/stackchan.cc`,
		`LogPowerDiagnosticSnapshot("boot-after-init");`,
		`A21 PMIC %s: %s`,
		`r61=%02x,r62=%02x,r63=%02x,r64=%02x`,
		`r80=%02x,r82=%02x,r90=%02x,r91=%02x,r92=%02x`,
		`std::string GetPowerDiagnosticSnapshot()`,
		`std::string board_get_power_diagnostic_snapshot();`,
		`return board.GetPmicPowerDiagnosticSnapshot();`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing StackChan power-key parity contract %q", required)
		}
	}
	for _, forbidden := range []string{
		`WriteReg(0x10, common_config | 0x04);`,
		`WriteReg(0x22, 0b110);`,
		`WriteReg(0x24, 0x00);`,
		`+        WriteReg(0x27, 0x10);`,
		`Enable 16s PWRON hardware PMIC shutdown fallback.`,
		`PWRON and OFFLEVEL can request PMIC power-off.`,
		`Lower VSYS shutdown threshold to 2.6V for battery cold boot inrush.`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("official Xiaozhi-compatible overlay must keep StackChan PMIC startup writes official; found %q", forbidden)
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`project(a21-stackchan-official-xiaozhi-compatible)`,
		`diff --git a/firmware/partitions.csv b/firmware/partitions.csv`,
		`+assets,   data, spiffs,  0xA00000,  5M,`,
		`CONFIG_BOARD_TYPE_M5STACK_STACK_CHAN=y`,
		`CONFIG_SEND_WAKE_WORD_DATA=n`,
		`# CONFIG_USE_AFE_WAKE_WORD is not set`,
		`CONFIG_USE_CUSTOM_WAKE_WORD=y`,
		`CONFIG_CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`,
		`CONFIG_CUSTOM_WAKE_WORD_DISPLAY="紫悦"`,
		`CONFIG_CUSTOM_WAKE_WORD_THRESHOLD=20`,
		`A21 overriding asset multinet commands with sdkconfig custom wake commands`,
		`command_list.find('|', start)`,
		`commands_.push_back({command, CONFIG_CUSTOM_WAKE_WORD_DISPLAY, "wake"});`,
		`Loaded %d A21 sdkconfig custom wake command(s) for %s`,
		`CONFIG_SR_MN_CN_MULTINET7_QUANT=y`,
		`# CONFIG_SR_WN_WN9_HISTACKCHAN_TTS3 is not set`,
		`Only AFE wake word can be detected in speaking mode`,
		`audio_service_.EnableWakeWordDetection(audio_service_.IsAfeWakeWord());`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing custom wake contract %q", required)
		}
	}
	for _, line := range strings.Split(overlay, "\n") {
		if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, " ")) &&
			strings.Contains(line, `CONFIG_SR_WN_WN9_HISTACKCHAN_TTS3=y`) {
			t.Fatalf("official Xiaozhi-compatible overlay must not leave the stock HiStackChan WakeNet model active with custom Zi Yue wake: %q", line)
		}
	}
	if strings.Contains(overlay, `project(xiaozhi)`) || strings.Contains(overlay, `xiaozhi.bin`) {
		t.Fatalf("official Xiaozhi-compatible overlay must not point to the bare Xiaozhi app lane")
	}
}

func TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`CONFIG_A21_STACKCHAN_KEEP_CONTROL_CHANNEL=y`,
		`config A21_STACKCHAN_KEEP_CONTROL_CHANNEL`,
		`a21_ready_notified_`,
		`a21_control_retry_notified_`,
		`EnsureA21ControlChannel`,
		`A21 keeping quiet control websocket open`,
		`正在连接紫悦服务`,
		`紫悦服务连接中，请稍等`,
		`紫悦已就绪，可以叫我`,
		`SendA21Keepalive`,
		`A21 quiet control websocket heartbeat failed`,
		`a21_ws_keepalive`,
		`A21 product websocket keepalive failed; reconnecting stale channel`,
		`a21_ws_reconnect`,
		`OpenAudioChannel()`,
		`A21 product websocket reconnect failed`,
		`esp_timer_start_periodic(a21_keepalive_timer_handle_, 10000000);`,
		`esp_timer_stop(a21_keepalive_timer_handle_);`,
		`protocol_->OpenAudioChannel()`,
		`state != kDeviceStateConnecting && !(state == kDeviceStateIdle && protocol_->IsAudioChannelOpened())`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing A21 idle socket contract %q", required)
		}
	}
	targetedWakeInvokeHunk := strings.Join([]string{
		` void Application::ContinueWakeWordInvoke(const std::string& wake_word) {`,
		`     // Check state again in case it was changed during scheduling`,
		`-    if (GetDeviceState() != kDeviceStateConnecting) {`,
		`+    auto state = GetDeviceState();`,
		`+    if (state != kDeviceStateConnecting && !(state == kDeviceStateIdle && protocol_->IsAudioChannelOpened())) {`,
		`         return;`,
	}, "\n")
	if !strings.Contains(overlay, targetedWakeInvokeHunk) {
		t.Fatal("official Xiaozhi-compatible overlay must patch ContinueWakeWordInvoke itself, not only a nearby helper with similar context")
	}
	forbiddenAdded := []string{
		`+            ListeningMode mode = GetDefaultListeningMode();`,
		`+                ContinueOpenAudioChannel(mode);`,
		`+        SetListeningMode(GetDefaultListeningMode());`,
		`+    if (mode == kListeningModeAutoStop && vad_stop_timer_handle_ != nullptr) {`,
		`+            listening_mode_ != kListeningModeAutoStop ||`,
		`A21_VAD_STOP_DEBOUNCE_MS`,
		`A21_NO_SPEECH_LISTENING_TIMEOUT_MS`,
		`MAIN_EVENT_VAD_STOP_TIMEOUT`,
		`HandleVadChange`,
		`HandleVadStopTimeoutEvent`,
		`vad_stop_timer_handle_`,
		`A21 no-speech timeout`,
	}
	for _, forbidden := range forbiddenAdded {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("A21 overlay must preserve official manual-start listening semantics; found %q", forbidden)
		}
	}
	heartbeatBeforeIdleGuard := strings.Index(overlay, `if (protocol_->IsAudioChannelOpened()) {`)
	idleGuard := strings.Index(overlay, `if (GetDeviceState() != kDeviceStateIdle) {`)
	if heartbeatBeforeIdleGuard < 0 || idleGuard < 0 || heartbeatBeforeIdleGuard > idleGuard {
		t.Fatalf("official Xiaozhi-compatible overlay must heartbeat an already-open control channel before applying the idle-only open guard")
	}
	for _, line := range strings.Split(overlay, "\n") {
		if !strings.HasPrefix(line, "+") {
			continue
		}
		for _, forbidden := range []string{
			`CONFIG_X21_STACKCHAN_KEEP_CONTROL_CHANNEL`,
			`EnsureX21ControlChannel`,
			`x21_ready_notified_`,
			`x21_control_retry_notified_`,
			`大头`,
		} {
			if strings.Contains(line, forbidden) {
				t.Fatalf("A21 overlay added forbidden legacy/control copy %q in line %q", forbidden, line)
			}
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlayAddsProductPlaybackAckOnly(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`config A21_PRODUCT_PLAYBACK_EVENTS`,
		`CONFIG_A21_PRODUCT_PLAYBACK_EVENTS=y`,
		`config A21_PRODUCT_TOUCH_EVENTS`,
		`CONFIG_A21_PRODUCT_TOUCH_EVENTS=y`,
		`cJSON_AddBoolToObject(features, "playback_events", true);`,
		`cJSON_AddBoolToObject(features, "keepalive_events", true);`,
		`cJSON_AddBoolToObject(features, "touch_events", true);`,
		`strcmp(profile->valuestring, "product") == 0`,
		`cJSON_IsTrue(playback_events)`,
		`cJSON_IsTrue(keepalive_events)`,
		`cJSON_IsTrue(touch_events)`,
		`a21_product_playback_events_allowed_`,
		`a21_product_keepalive_events_allowed_`,
		`a21_product_touch_events_allowed_`,
		`esp_timer_handle_t a21_keepalive_timer_handle_ = nullptr;`,
		`SendA21Keepalive`,
		`cJSON_AddStringToObject(root, "kind", "heartbeat");`,
		`Board::GetInstance().GetBatteryLevel`,
		`cJSON_AddNumberToObject(root, "battery_level", battery_level);`,
		`cJSON_AddBoolToObject(root, "battery_charging", battery_charging);`,
		`cJSON_AddBoolToObject(root, "battery_discharging", battery_discharging);`,
		`cJSON_AddBoolToObject(root, "external_power", !battery_discharging);`,
		`cJSON_AddStringToObject(root, "pmic_power_key_profile", "a21_stackchan_axp2101_pwrkey_v1");`,
		`hal_bridge::board_get_power_diagnostic_snapshot`,
		`cJSON_AddStringToObject(root, "pmic_power_status", pmic_status.c_str());`,
		`SendA21PlaybackStart`,
		`SendA21PlaybackStopDone`,
		`SendA21TouchEvent`,
		`cJSON_AddStringToObject(root, "kind", "touch");`,
		`cJSON_AddStringToObject(root, "touch", touch);`,
		`cJSON_AddStringToObject(root, "source", source);`,
		`onScreenTouch`,
		`HandleA21ScreenTouchEvent`,
		`HandleA21HeadPetGesture`,
		`ApplyA21TouchLocalFeedback`,
		`Lang::Sounds::OGG_VIBRATION`,
		`leftNeonLight().setColor`,
		`rightNeonLight().setColor`,
		`top_swipe_forward`,
		`top_barge_in`,
		`yaw = 900;`,
		`yaw = -900;`,
		`pitch = 760;`,
		`speed = 950;`,
		`callbacks.on_playback_started`,
		`AudioOutputTask()`,
		`audio_service_.ResetDecoder();`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing product playback ack contract %q", required)
		}
	}
	for _, forbidden := range []string{
		`CONFIG_A21_DEBUG_DEVICE_EVENTS`,
		`a21_debug_device_events_allowed_`,
		`cJSON_AddBoolToObject(features, "device_events", true);`,
		`"profile", "debug"`,
		`"device_events"`,
		`ENABLE_X21_DEVICE_EVENTS`,
		`!motion.isMoving()`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("product playback ack overlay must not enable debug/legacy device events %q", forbidden)
		}
	}
}

func TestOfficialXiaozhiCompatibleOverlayPreservesXiaozhiWifiProvisioning(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	projectRoot := findProjectRoot(cwd)
	overlayPath := filepath.Join(projectRoot, "firmware", "stackchan-official", "overlays", "a21-official-xiaozhi-compatible.patch")
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("read overlay: %v", err)
	}
	overlay := string(data)

	for _, required := range []string{
		`CONFIG_USE_HOTSPOT_WIFI_PROVISIONING=y`,
		`# CONFIG_USE_ESP_BLUFI_WIFI_PROVISIONING is not set`,
		`# CONFIG_USE_ACOUSTIC_WIFI_PROVISIONING is not set`,
		`CONFIG_OTA_URL="http://47.103.57.217/xiaozhi/ota/"`,
	} {
		if !strings.Contains(overlay, required) {
			t.Fatalf("official Xiaozhi-compatible overlay missing Wi-Fi provisioning contract %q", required)
		}
	}
	for _, forbidden := range []string{
		`CONFIG_WIFI_SSID=`,
		`CONFIG_WIFI_PASSWORD=`,
		`A21_WIFI_PASSWORD`,
		`101.132.117.182`,
	} {
		if strings.Contains(overlay, forbidden) {
			t.Fatalf("official Xiaozhi-compatible overlay must not hardcode stale Wi-Fi/provisioning value %q", forbidden)
		}
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

func TestRunStackChanOfficialBaselineFlashPlanBuildsNoFlashReceipt(t *testing.T) {
	originalDetector := detectFirmwareUploadPortUsage
	detectFirmwareUploadPortUsage = func(port string) (firmwarecheck.PortUsage, error) {
		return firmwarecheck.PortUsage{Exists: true, InUse: false}, nil
	}
	defer func() {
		detectFirmwareUploadPortUsage = originalDetector
	}()

	buildDir := writeTestOfficialBaselineBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline-flash-plan",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.stackchan.official_baseline_flash.v1"`,
		`"flash_allowed": false`,
		`"path": "`,
		`stack-chan.bin`,
		`"0x20000"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("baseline flash plan missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanOfficialBaselineFlashExecuteRequiresConfirmationToken(t *testing.T) {
	buildDir := writeTestOfficialBaselineBuild(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"stackchan-official-baseline-flash-execute",
		"--build-dir", buildDir,
		"--port", "/dev/cu.usbmodemA21",
		"--idf-export", filepath.Join(t.TempDir(), "export.sh"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("execute without confirmation unexpectedly passed: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "WRITE_A21_STACKCHAN_OFFICIAL_BASELINE_DIAGNOSTIC") {
		t.Fatalf("stderr missing confirmation token: %s", stderr.String())
	}
}
