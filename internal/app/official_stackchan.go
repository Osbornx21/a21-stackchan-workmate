package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const stackChanOfficialBaselineSchema = "a21.stackchan.official_baseline.v1"
const stackChanOfficialAudioSmokeFlashSchema = "a21.stackchan.official_audio_smoke_flash.v1"
const stackChanOfficialPCMBridgeFlashPlanSchema = "a21.stackchan.official_pcm_bridge_flash_plan.v1"
const stackChanOfficialAudioSmokeFlashConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE"

var runStackChanOfficialSmokeFlashCommand = runStackChanOfficialSmokeFlashCommandExec

type stackChanOfficialBaselineOptions struct {
	SourceRoot string
	WorkDir    string
	BuildDir   string
	IDFExport  string
	OutputDir  string
	Overlays   []string
	Execute    bool
}

type stackChanOfficialBaselineReport struct {
	SchemaVersion  string                             `json:"schema_version"`
	GeneratedAtMS  int64                              `json:"generated_at_ms"`
	Status         string                             `json:"status"`
	Execute        bool                               `json:"execute"`
	SourceRoot     string                             `json:"source_root"`
	SourceCommit   string                             `json:"source_commit,omitempty"`
	SourceClean    bool                               `json:"source_clean"`
	DirtyFileCount int                                `json:"dirty_file_count"`
	WorkDir        string                             `json:"work_dir"`
	BuildDir       string                             `json:"build_dir"`
	IDFExport      string                             `json:"idf_export"`
	Overlays       []stackChanOfficialBaselineOverlay `json:"overlays,omitempty"`
	Evidence       stackChanOfficialBaselineEvidence  `json:"evidence"`
	Build          stackChanOfficialBaselineBuild     `json:"build"`
	Findings       []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath     string                             `json:"report_path,omitempty"`
}

type stackChanOfficialBaselineEvidence struct {
	TrackedSDKConfigHasNoLegacyIdentity  bool     `json:"tracked_sdkconfig_has_no_legacy_identity"`
	HalMicTestUsesOutputData             bool     `json:"hal_mic_test_uses_output_data"`
	CoreS3UsesESPCodecDev                bool     `json:"cores3_uses_esp_codec_dev"`
	CoreS3CreatesDuplexChannels          bool     `json:"cores3_creates_duplex_channels"`
	ReposDeclareXiaoZhiAudioService      bool     `json:"repos_declare_xiaozhi_audio_service"`
	XiaoZhiOutputTaskUsesCodecOutputData bool     `json:"xiaozhi_output_task_uses_codec_output_data"`
	XiaoZhiOpusFrameDurationMS           int      `json:"xiaozhi_opus_frame_duration_ms,omitempty"`
	MatureComponents                     []string `json:"mature_components"`
	ReferenceFiles                       []string `json:"reference_files"`
}

type stackChanOfficialBaselineBuild struct {
	FetchExecuted bool                                     `json:"fetch_executed"`
	BuildExecuted bool                                     `json:"build_executed"`
	FetchLogPath  string                                   `json:"fetch_log_path,omitempty"`
	BuildLogPath  string                                   `json:"build_log_path,omitempty"`
	Artifacts     []stackChanOfficialBaselineBuildArtifact `json:"artifacts,omitempty"`
}

type stackChanOfficialBaselineOverlay struct {
	Path    string `json:"path"`
	Applied bool   `json:"applied"`
}

type stackChanOfficialBaselineBuildArtifact struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
	FlashOffset string `json:"flash_offset,omitempty"`
}

type stackChanOfficialBaselineFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type stackChanOfficialSmokeFlashOptions struct {
	BuildDir  string
	IDFExport string
	Port      string
	OutputDir string
	Confirm   string
	Execute   bool
}

type stackChanOfficialPCMBridgeFlashPlanOptions struct {
	BuildDir   string
	IDFExport  string
	Port       string
	OutputDir  string
	DeviceID   string
	AudioWSURL string
}

type stackChanOfficialSmokeFlashReport struct {
	SchemaVersion            string                             `json:"schema_version"`
	GeneratedAtMS            int64                              `json:"generated_at_ms"`
	Status                   string                             `json:"status"`
	DryRun                   bool                               `json:"dry_run"`
	FlashAllowed             bool                               `json:"flash_allowed"`
	FlashExecuted            bool                               `json:"flash_executed"`
	Port                     string                             `json:"port"`
	BuildDir                 string                             `json:"build_dir"`
	IDFExport                string                             `json:"idf_export"`
	FlashLogPath             string                             `json:"flash_log_path,omitempty"`
	NextRequiredConfirmation string                             `json:"next_required_confirmation,omitempty"`
	Parts                    []stackChanOfficialSmokeFlashPart  `json:"parts"`
	Findings                 []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath               string                             `json:"report_path,omitempty"`
}

type stackChanOfficialPCMBridgeFlashPlanReport struct {
	SchemaVersion            string                             `json:"schema_version"`
	GeneratedAtMS            int64                              `json:"generated_at_ms"`
	Status                   string                             `json:"status"`
	DryRun                   bool                               `json:"dry_run"`
	FlashAllowed             bool                               `json:"flash_allowed"`
	Port                     string                             `json:"port"`
	BuildDir                 string                             `json:"build_dir"`
	IDFExport                string                             `json:"idf_export"`
	DeviceID                 string                             `json:"device_id"`
	AudioWS                  stackChanOfficialPCMBridgeAudioWS  `json:"audio_ws"`
	NextRequiredConfirmation string                             `json:"next_required_confirmation,omitempty"`
	Parts                    []stackChanOfficialSmokeFlashPart  `json:"parts"`
	Findings                 []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath               string                             `json:"report_path,omitempty"`
}

type stackChanOfficialPCMBridgeAudioWS struct {
	Scheme        string `json:"scheme"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	DeviceIDQuery bool   `json:"device_id_query"`
}

type stackChanOfficialSmokeFlashPart struct {
	Name      string `json:"name"`
	Offset    string `json:"offset"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func runStackChanOfficialBaseline(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialBaselineOptions{
		SourceRoot: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_SOURCE")),
		WorkDir:    firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_WORK_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-clean")),
		BuildDir:   firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-baseline --source <m5stack-stackchan-repo> [--overlay firmware/stackchan-official/overlays/a21-official-audio-smoke.patch] [--execute] [--work-dir /tmp/a21-stackchan-official-clean] [--build-dir /tmp/a21-stackchan-official-build] [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
		case "--source":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--source requires a value")
				return 2
			}
			i++
			options.SourceRoot = args[i]
		case "--work-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--work-dir requires a value")
				return 2
			}
			i++
			options.WorkDir = args[i]
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--overlay":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--overlay requires a value")
				return 2
			}
			i++
			options.Overlays = append(options.Overlays, args[i])
		case "--execute":
			options.Execute = true
		default:
			fmt.Fprintf(stderr, "unknown stackchan-official-baseline option %q\n", args[i])
			return 2
		}
	}

	if options.SourceRoot == "" {
		discovered, ok := discoverLocalOfficialStackChanSource()
		if ok {
			options.SourceRoot = discovered
		}
	}
	if err := normalizeOfficialOverlayPaths(&options); err != nil {
		fmt.Fprintf(stderr, "stackchan official baseline overlay path invalid: %v\n", err)
		return 1
	}

	report, err := buildStackChanOfficialBaselineReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official baseline: %v\n", err)
		return 1
	}
	if options.Execute && report.Status == "ready" {
		executeStackChanOfficialBaseline(context.Background(), options, &report)
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official baseline report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialBaselineReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official baseline report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialBaseline(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official baseline report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialAudioSmokeFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialSmokeFlashOptions{
		BuildDir:  firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport: firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:      strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		OutputDir: "",
		Confirm:   "",
		Execute:   execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-audio-smoke-flash-plan --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--idf-export /path/to/export.sh] [--output-dir reports]")
			fmt.Fprintln(stdout, "a21 stackchan-official-audio-smoke-flash-execute --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX --confirm WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official audio smoke flash option %q\n", args[i])
			return 2
		}
	}

	if execute {
		if options.Confirm != stackChanOfficialAudioSmokeFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official audio smoke flash requires --confirm %s\n", stackChanOfficialAudioSmokeFlashConfirm)
			return 2
		}
	}
	report, err := buildStackChanOfficialSmokeFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official audio smoke flash: %v\n", err)
		return 1
	}
	if execute {
		if err := executeStackChanOfficialSmokeFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official audio smoke flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialSmokeFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official audio smoke flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialSmokeFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official audio smoke flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialPCMBridgeFlashPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialPCMBridgeFlashPlanOptions{
		BuildDir:   firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:       strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		AudioWSURL: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL")),
		OutputDir:  "",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-pcm-bridge-flash-plan --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX --device-id stackchan-001 --audio-ws-url ws://host:21080/ws/audio?device_id=stackchan-001 [--idf-export /path/to/export.sh] [--output-dir reports]")
			return 0
		case "--build-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--build-dir requires a value")
				return 2
			}
			i++
			options.BuildDir = args[i]
		case "--idf-export":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--idf-export requires a value")
				return 2
			}
			i++
			options.IDFExport = args[i]
		case "--port":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--port requires a value")
				return 2
			}
			i++
			options.Port = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--audio-ws-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--audio-ws-url requires a value")
				return 2
			}
			i++
			options.AudioWSURL = args[i]
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official pcm bridge flash plan option %q\n", args[i])
			return 2
		}
	}

	report, err := buildStackChanOfficialPCMBridgeFlashPlanReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official pcm bridge flash plan: %v\n", err)
		return 1
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official pcm bridge flash plan report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialPCMBridgeFlashPlanReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official pcm bridge flash plan report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialPCMBridgeFlashPlan(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official pcm bridge flash plan report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func normalizeOfficialOverlayPaths(options *stackChanOfficialBaselineOptions) error {
	if len(options.Overlays) == 0 {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	for i, overlay := range options.Overlays {
		cleanOverlay := filepath.Clean(overlay)
		if !filepath.IsAbs(cleanOverlay) {
			cleanOverlay = filepath.Join(cwd, cleanOverlay)
		}
		options.Overlays[i] = filepath.Clean(cleanOverlay)
	}
	return nil
}

func buildStackChanOfficialBaselineReport(options stackChanOfficialBaselineOptions) (stackChanOfficialBaselineReport, error) {
	sourceRoot := filepath.Clean(options.SourceRoot)
	if sourceRoot == "." || sourceRoot == "" {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("source root is required")
	}
	if err := validateA21OfficialScratchDir(options.WorkDir); err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("work dir invalid: %w", err)
	}
	if err := validateA21OfficialScratchDir(options.BuildDir); err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	gitRoot, err := gitOutput(sourceRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("source is not a git checkout")
	}
	sourceRoot = strings.TrimSpace(gitRoot)

	commit, err := gitOutput(sourceRoot, "rev-parse", "HEAD")
	if err != nil {
		return stackChanOfficialBaselineReport{}, fmt.Errorf("read source commit: %w", err)
	}
	statusOutput, _ := gitOutput(sourceRoot, "status", "--porcelain", "--untracked-files=all")
	dirtyFiles := countNonEmptyLines(statusOutput)

	report := stackChanOfficialBaselineReport{
		SchemaVersion:  stackChanOfficialBaselineSchema,
		GeneratedAtMS:  time.Now().UnixMilli(),
		Status:         "ready",
		Execute:        options.Execute,
		SourceRoot:     sourceRoot,
		SourceCommit:   strings.TrimSpace(commit),
		SourceClean:    dirtyFiles == 0,
		DirtyFileCount: dirtyFiles,
		WorkDir:        filepath.Clean(options.WorkDir),
		BuildDir:       filepath.Clean(options.BuildDir),
		IDFExport:      filepath.Clean(options.IDFExport),
	}
	for _, overlay := range options.Overlays {
		report.Overlays = append(report.Overlays, stackChanOfficialBaselineOverlay{
			Path: filepath.Clean(overlay),
		})
	}
	report.Evidence = inspectStackChanOfficialEvidence(sourceRoot)
	if !report.SourceClean {
		report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
			Code:    "source_worktree_dirty",
			Message: "source checkout has local changes; A21 will export from git HEAD only",
		})
	}
	if !report.Evidence.TrackedSDKConfigHasNoLegacyIdentity {
		report.addFinding("tracked_config_legacy_identity", "tracked official sdkconfig contains forbidden legacy identity")
	}
	if !report.Evidence.HalMicTestUsesOutputData ||
		!report.Evidence.CoreS3UsesESPCodecDev ||
		!report.Evidence.CoreS3CreatesDuplexChannels {
		report.addFinding("missing_official_audio_codec", "official StackChan audio codec evidence is incomplete")
	}
	if !report.Evidence.ReposDeclareXiaoZhiAudioService {
		report.addFinding("missing_xiaozhi_audio_dependency", "official repos manifest does not declare xiaozhi audio service dependency")
	}
	if len(report.Findings) > 0 {
		for _, finding := range report.Findings {
			if strings.HasPrefix(finding.Code, "missing_") || finding.Code == "tracked_config_legacy_identity" {
				report.Status = "failed"
				break
			}
		}
	}
	return report, nil
}

func buildStackChanOfficialSmokeFlashReport(options stackChanOfficialSmokeFlashOptions) (stackChanOfficialSmokeFlashReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentity(buildDir) {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialSmokeFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialSmokeFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialSmokeFlashReport{}, err
	}
	report := stackChanOfficialSmokeFlashReport{
		SchemaVersion:            stackChanOfficialAudioSmokeFlashSchema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		NextRequiredConfirmation: "stackchan-official-audio-smoke-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}
	return report, nil
}

func buildStackChanOfficialPCMBridgeFlashPlanReport(options stackChanOfficialPCMBridgeFlashPlanOptions) (stackChanOfficialPCMBridgeFlashPlanReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentity(buildDir) {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	deviceID := strings.TrimSpace(options.DeviceID)
	if err := validateOfficialPCMBridgeDeviceID(deviceID); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	audioWS, err := parseOfficialPCMBridgeAudioWSURL(options.AudioWSURL, deviceID)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialPCMBridgeFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialPCMBridgeFlashPlanReport{}, err
	}
	return stackChanOfficialPCMBridgeFlashPlanReport{
		SchemaVersion:            stackChanOfficialPCMBridgeFlashPlanSchema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   true,
		FlashAllowed:             false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		DeviceID:                 deviceID,
		AudioWS:                  audioWS,
		NextRequiredConfirmation: "bridge_nvs_provisioning_and_flash_execute_guard_not_implemented",
		Parts:                    parts,
	}, nil
}

func collectOfficialSmokeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-audio-smoke.bin")
}

func collectOfficialPCMBridgeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-pcm-bridge.bin")
}

func collectOfficialFlashPartsForApp(buildDir string, expectedAppName string) ([]stackChanOfficialSmokeFlashPart, error) {
	entries := readOfficialFlashArgsEntries(buildDir)
	if len(entries) == 0 {
		return nil, fmt.Errorf("official smoke flash_args missing or empty")
	}
	required := map[string]string{
		"0x0":      "bootloader",
		"0x8000":   "partition_table",
		"0xd000":   "ota_data_initial",
		"0x20000":  "app",
		"0xa00000": "assets",
	}
	seenOffsets := make(map[string]bool)
	parts := make([]stackChanOfficialSmokeFlashPart, 0, len(entries))
	for _, entry := range entries {
		name, ok := required[entry.offset]
		if !ok {
			continue
		}
		fullPath := filepath.Join(buildDir, filepath.FromSlash(entry.path))
		if containsLegacyIdentity(fullPath) {
			return nil, fmt.Errorf("flash part path contains forbidden legacy identity")
		}
		if name == "app" && filepath.Base(fullPath) != expectedAppName {
			return nil, fmt.Errorf("official app must be %s", expectedAppName)
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("flash part %s missing: %w", name, err)
		}
		sum, err := sha256File(fullPath)
		if err != nil {
			return nil, fmt.Errorf("hash flash part %s: %w", name, err)
		}
		seenOffsets[entry.offset] = true
		parts = append(parts, stackChanOfficialSmokeFlashPart{
			Name:      name,
			Offset:    entry.offset,
			Path:      fullPath,
			SHA256:    sum,
			SizeBytes: stat.Size(),
		})
	}
	for offset, name := range required {
		if !seenOffsets[offset] {
			return nil, fmt.Errorf("flash_args missing required %s at %s", name, offset)
		}
	}
	return parts, nil
}

func validateOfficialPCMBridgeDeviceID(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device-id is required")
	}
	if containsLegacyIdentity(deviceID) {
		return fmt.Errorf("device-id contains forbidden legacy identity")
	}
	if !strings.HasPrefix(deviceID, "stackchan-") {
		return fmt.Errorf("device-id must use stackchan-*")
	}
	return nil
}

func parseOfficialPCMBridgeAudioWSURL(rawURL string, deviceID string) (stackChanOfficialPCMBridgeAudioWS, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("--audio-ws-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url contains forbidden legacy identity")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("parse audio ws url: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url must use ws or wss")
	}
	if parsed.User != nil {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url host is required")
	}
	if parsed.Path != "/ws/audio" {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url path must be /ws/audio")
	}
	if parsed.Query().Get("device_id") != deviceID {
		return stackChanOfficialPCMBridgeAudioWS{}, fmt.Errorf("audio ws url device_id query must match --device-id")
	}
	return stackChanOfficialPCMBridgeAudioWS{
		Scheme:        parsed.Scheme,
		Host:          parsed.Host,
		Path:          parsed.Path,
		DeviceIDQuery: true,
	}, nil
}

func executeStackChanOfficialSmokeFlash(ctx context.Context, options stackChanOfficialSmokeFlashOptions, report *stackChanOfficialSmokeFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	report.FlashLogPath = filepath.Join(options.BuildDir, fmt.Sprintf("a21-official-audio-smoke-flash-%s.log", time.Now().Format("20060102-150405")))
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runStackChanOfficialSmokeFlashCommand(ctx, report.FlashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func runStackChanOfficialSmokeFlashCommandExec(ctx context.Context, logPath string, script string) error {
	return runLoggedCommand(ctx, "", logPath, "bash", "-lc", script)
}

func validateOfficialSmokeUploadPort(port string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return fmt.Errorf("upload port is required")
	}
	base := filepath.Base(port)
	lowerBase := strings.ToLower(base)
	if !(strings.HasPrefix(base, "cu.") || strings.HasPrefix(base, "tty.")) {
		return fmt.Errorf("upload port must be an explicit serial device")
	}
	if !strings.Contains(lowerBase, "usbmodem") && !strings.Contains(lowerBase, "usbserial") {
		return fmt.Errorf("upload port must be an explicit USB serial device")
	}
	return nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func inspectStackChanOfficialEvidence(sourceRoot string) stackChanOfficialBaselineEvidence {
	sdkConfig := readGitTrackedOrFile(sourceRoot, "firmware/sdkconfig.defaults")
	halAudio := readGitTrackedOrFile(sourceRoot, "firmware/main/hal/audio.cpp")
	coreS3Codec := readGitTrackedOrFile(sourceRoot, "firmware/main/hal/board/cores3_audio_codec.cc")
	repos := readGitTrackedOrFile(sourceRoot, "firmware/repos.json")
	xiaozhiService := readGitTrackedOrFile(sourceRoot, "firmware/xiaozhi-esp32/main/audio/audio_service.cc")
	xiaozhiHeader := readGitTrackedOrFile(sourceRoot, "firmware/xiaozhi-esp32/main/audio/audio_service.h")

	evidence := stackChanOfficialBaselineEvidence{
		TrackedSDKConfigHasNoLegacyIdentity:  !containsLegacyIdentity(sdkConfig),
		HalMicTestUsesOutputData:             strings.Contains(halAudio, "audio_codec->OutputData"),
		CoreS3UsesESPCodecDev:                strings.Contains(coreS3Codec, "esp_codec_dev_write") && strings.Contains(coreS3Codec, "esp_codec_dev_open"),
		CoreS3CreatesDuplexChannels:          strings.Contains(coreS3Codec, "CreateDuplexChannels"),
		ReposDeclareXiaoZhiAudioService:      strings.Contains(repos, "xiaozhi-esp32") && strings.Contains(repos, "v2.2.4"),
		XiaoZhiOutputTaskUsesCodecOutputData: strings.Contains(xiaozhiService, "AudioOutputTask") && strings.Contains(xiaozhiService, "codec_->OutputData"),
		XiaoZhiOpusFrameDurationMS:           parseOpusFrameDurationMS(xiaozhiHeader),
		MatureComponents: []string{
			"ESP-IDF",
			"esp_codec_dev",
			"esp_audio_codec",
			"esp-sr",
			"xiaozhi AudioService pattern",
			"Opus frame pipeline",
		},
		ReferenceFiles: []string{
			"firmware/main/hal/audio.cpp",
			"firmware/main/hal/board/cores3_audio_codec.cc",
			"firmware/main/hal/board/config.h",
			"firmware/repos.json",
			"firmware/xiaozhi-esp32/main/audio/audio_service.cc",
			"firmware/xiaozhi-esp32/main/audio/audio_service.h",
		},
	}
	return evidence
}

func executeStackChanOfficialBaseline(ctx context.Context, options stackChanOfficialBaselineOptions, report *stackChanOfficialBaselineReport) {
	report.Status = "passed"
	if err := os.RemoveAll(options.WorkDir); err != nil {
		report.fail("clean_work_dir_failed", "failed to clean official baseline work dir")
		return
	}
	if err := os.RemoveAll(options.BuildDir); err != nil {
		report.fail("clean_build_dir_failed", "failed to clean official baseline build dir")
		return
	}
	if err := os.MkdirAll(options.WorkDir, 0o755); err != nil {
		report.fail("create_work_dir_failed", "failed to create official baseline work dir")
		return
	}
	if err := exportGitHEAD(ctx, report.SourceRoot, options.WorkDir); err != nil {
		report.fail("export_head_failed", "failed to export official source HEAD")
		return
	}
	for index, overlay := range options.Overlays {
		cleanOverlay := filepath.Clean(overlay)
		if containsLegacyIdentity(cleanOverlay) {
			report.fail("overlay_path_legacy_identity", "overlay path contains forbidden legacy identity")
			return
		}
		if err := runLoggedCommand(ctx, options.WorkDir, filepath.Join(options.BuildDir, "a21-official-overlay.log"), "git", "apply", cleanOverlay); err != nil {
			report.fail("overlay_apply_failed", "failed to apply A21 official baseline overlay")
			return
		}
		if index < len(report.Overlays) {
			report.Overlays[index].Applied = true
		}
	}

	fetchLog := filepath.Join(options.BuildDir, "a21-official-fetch.log")
	buildLog := filepath.Join(options.BuildDir, "a21-official-build.log")
	_ = os.MkdirAll(options.BuildDir, 0o755)

	report.Build.FetchExecuted = true
	report.Build.FetchLogPath = fetchLog
	if err := runLoggedCommand(ctx, filepath.Join(options.WorkDir, "firmware"), fetchLog, "python3", "./fetch_repos.py"); err != nil {
		report.fail("fetch_repos_failed", "official fetch_repos.py failed")
		return
	}

	report.Evidence = inspectStackChanOfficialEvidence(options.WorkDir)
	report.Build.BuildExecuted = true
	report.Build.BuildLogPath = buildLog
	if _, err := os.Stat(options.IDFExport); err != nil {
		report.fail("idf_export_missing", "ESP-IDF export.sh is missing")
		return
	}
	buildScript := fmt.Sprintf("set -euo pipefail\nsource %q >/dev/null\nidf.py -C %q -B %q build", options.IDFExport, filepath.Join(options.WorkDir, "firmware"), options.BuildDir)
	if err := runLoggedCommand(ctx, "", buildLog, "bash", "-lc", buildScript); err != nil {
		report.fail("idf_build_failed", "official ESP-IDF build failed")
		return
	}
	report.Build.Artifacts = collectOfficialStackChanBuildArtifacts(options.BuildDir)
	if len(report.Build.Artifacts) == 0 {
		report.fail("build_artifacts_missing", "official build completed without expected artifacts")
		return
	}
}

func exportGitHEAD(ctx context.Context, sourceRoot string, workDir string) error {
	archive := exec.CommandContext(ctx, "git", "-C", sourceRoot, "archive", "HEAD")
	tar := exec.CommandContext(ctx, "tar", "-x", "-C", workDir)
	reader, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	tar.Stdin = reader
	if err := tar.Start(); err != nil {
		return err
	}
	if err := archive.Start(); err != nil {
		return err
	}
	archiveErr := archive.Wait()
	tarErr := tar.Wait()
	if archiveErr != nil {
		return archiveErr
	}
	return tarErr
}

func runLoggedCommand(ctx context.Context, dir string, logPath string, name string, args ...string) error {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	return cmd.Run()
}

func collectOfficialStackChanBuildArtifacts(buildDir string) []stackChanOfficialBaselineBuildArtifact {
	flashEntries := readOfficialFlashArgsEntries(buildDir)
	artifacts := make([]stackChanOfficialBaselineBuildArtifact, 0, len(flashEntries)+1)
	seen := make(map[string]bool)
	for _, entry := range flashEntries {
		fullPath := filepath.Join(buildDir, filepath.FromSlash(entry.path))
		name := nameOfficialFlashArtifact(entry.path, entry.offset)
		sum, err := sha256File(fullPath)
		if err != nil {
			continue
		}
		seen[filepath.Clean(fullPath)] = true
		artifacts = append(artifacts, stackChanOfficialBaselineBuildArtifact{
			Name:        name,
			Path:        fullPath,
			SHA256:      sum,
			FlashOffset: entry.offset,
		})
	}
	fallbackCandidates := []struct {
		name string
		path string
	}{
		{name: "bootloader", path: filepath.Join(buildDir, "bootloader", "bootloader.bin")},
		{name: "partition_table", path: filepath.Join(buildDir, "partition_table", "partition-table.bin")},
		{name: "ota_data_initial", path: filepath.Join(buildDir, "ota_data_initial.bin")},
		{name: "app", path: filepath.Join(buildDir, "stack-chan.bin")},
		{name: "app", path: filepath.Join(buildDir, "a21-stackchan-official-audio-smoke.bin")},
		{name: "app", path: filepath.Join(buildDir, "a21-stackchan-official-pcm-bridge.bin")},
		{name: "assets", path: filepath.Join(buildDir, "generated_assets.bin")},
	}
	for _, candidate := range fallbackCandidates {
		cleanPath := filepath.Clean(candidate.path)
		if seen[cleanPath] {
			continue
		}
		sum, err := sha256File(candidate.path)
		if err != nil {
			continue
		}
		seen[cleanPath] = true
		artifacts = append(artifacts, stackChanOfficialBaselineBuildArtifact{
			Name:   candidate.name,
			Path:   candidate.path,
			SHA256: sum,
		})
	}
	if sum, err := sha256File(filepath.Join(buildDir, "flash_args")); err == nil {
		artifacts = append(artifacts, stackChanOfficialBaselineBuildArtifact{
			Name:   "flash_args",
			Path:   filepath.Join(buildDir, "flash_args"),
			SHA256: sum,
		})
	}
	return artifacts
}

type officialFlashArgEntry struct {
	offset string
	path   string
}

func readOfficialFlashArgsEntries(buildDir string) []officialFlashArgEntry {
	data, err := os.ReadFile(filepath.Join(buildDir, "flash_args"))
	if err != nil {
		return nil
	}
	var entries []officialFlashArgEntry
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || !strings.HasPrefix(fields[0], "0x") {
			continue
		}
		entries = append(entries, officialFlashArgEntry{
			offset: fields[0],
			path:   fields[1],
		})
	}
	return entries
}

func nameOfficialFlashArtifact(path string, offset string) string {
	base := filepath.Base(path)
	switch {
	case base == "bootloader.bin":
		return "bootloader"
	case base == "partition-table.bin":
		return "partition_table"
	case base == "ota_data_initial.bin":
		return "ota_data_initial"
	case base == "generated_assets.bin":
		return "assets"
	case offset == "0x20000":
		return "app"
	default:
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func parseOpusFrameDurationMS(header string) int {
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#define OPUS_FRAME_DURATION_MS") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return 0
		}
		if fields[2] == "60" {
			return 60
		}
	}
	return 0
}

func readGitTrackedOrFile(sourceRoot string, repoPath string) string {
	if output, err := gitOutput(sourceRoot, "show", "HEAD:"+repoPath); err == nil {
		return output
	}
	data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(repoPath)))
	if err != nil {
		return ""
	}
	return string(data)
}

func gitOutput(sourceRoot string, args ...string) (string, error) {
	allArgs := append([]string{"-C", sourceRoot}, args...)
	output, err := exec.Command("git", allArgs...).CombinedOutput()
	return string(output), err
}

func countNonEmptyLines(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func validateA21OfficialScratchDir(path string) error {
	clean := filepath.Clean(path)
	lower := strings.ToLower(clean)
	if strings.Contains(lower, "x21") || strings.Contains(lower, "v21") {
		return fmt.Errorf("scratch dir contains forbidden legacy identity")
	}
	if !strings.Contains(filepath.Base(lower), "a21-stackchan-official") {
		return fmt.Errorf("scratch dir basename must contain a21-stackchan-official")
	}
	return nil
}

func discoverLocalOfficialStackChanSource() (string, bool) {
	candidates := []string{
		"/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "firmware", "repos.json")); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func (report *stackChanOfficialBaselineReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{Code: code, Message: message})
}

func (report *stackChanOfficialBaselineReport) fail(code string, message string) {
	report.Status = "failed"
	report.addFinding(code, message)
}

func writeStackChanOfficialBaselineReport(outputDir string, report stackChanOfficialBaselineReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-baseline-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanOfficialBaseline(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanOfficialSmokeFlashReport(outputDir string, report stackChanOfficialSmokeFlashReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-audio-smoke-flash-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanOfficialSmokeFlash(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanOfficialPCMBridgeFlashPlanReport(outputDir string, report stackChanOfficialPCMBridgeFlashPlanReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-pcm-bridge-flash-plan-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONStackChanOfficialPCMBridgeFlashPlan(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeJSONStackChanOfficialBaseline(writer io.Writer, report stackChanOfficialBaselineReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanOfficialSmokeFlash(writer io.Writer, report stackChanOfficialSmokeFlashReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanOfficialPCMBridgeFlashPlan(writer io.Writer, report stackChanOfficialPCMBridgeFlashPlanReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
