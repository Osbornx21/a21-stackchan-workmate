package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"a21.local/a21/internal/runtimeguard"
)

const stackChanOfficialBaselineSchema = "a21.stackchan.official_baseline.v1"

const stackChanOfficialBaselineFlashSchema = "a21.stackchan.official_baseline_flash.v1"

const stackChanOfficialAudioSmokeFlashSchema = "a21.stackchan.official_audio_smoke_flash.v1"

const stackChanOfficialPCMBridgeFlashPlanSchema = "a21.stackchan.official_pcm_bridge_flash_plan.v1"

const stackChanOfficialPCMBridgeFlashExecutionSchema = "a21.stackchan.official_pcm_bridge_flash_execution.v1"

const stackChanOfficialXiaozhiCompatibleFlashPlanSchema = "a21.stackchan.official_xiaozhi_compatible_flash_plan.v1"

const stackChanOfficialXiaozhiCompatibleFlashExecutionSchema = "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1"

const stackChanOfficialXiaozhiCompatibleNVSPlanSchema = "a21.stackchan.official_xiaozhi_compatible_nvs_plan.v1"

const stackChanOfficialXiaozhiCompatibleNVSExecutionSchema = "a21.stackchan.official_xiaozhi_compatible_nvs_execution.v1"

const stackChanOfficialPCMBridgeNVSPlanSchema = "a21.stackchan.official_pcm_bridge_nvs_plan.v1"

const stackChanOfficialPCMBridgeNVSExecutionSchema = "a21.stackchan.official_pcm_bridge_nvs_execution.v1"

const stackChanOfficialBaselineFlashConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_BASELINE_DIAGNOSTIC"

const stackChanOfficialAudioSmokeFlashConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE"

const stackChanOfficialPCMBridgeNVSConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS"

const stackChanOfficialPCMBridgeAppFlashConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP"

const stackChanOfficialXiaozhiCompatibleAppFlashConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP"

const stackChanOfficialXiaozhiCompatibleNVSConfirm = "WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS"

const stackChanOfficialPCMBridgeNVSOffset = "0x9000"

const stackChanOfficialPCMBridgeNVSSizeHex = "0x4000"

const stackChanOfficialPCMBridgeNVSSizeBytes = 0x4000

const officialProductLaneArtifactEvidenceSchema = "a21.firmware.product_lane_artifact_evidence.v1"

const stackChanOfficialXiaozhiCompatibleFirmwareCandidate = "a21-stackchan-official-xiaozhi-compatible"

const stackChanOfficialXiaozhiCompatibleAppBinary = "a21-stackchan-official-xiaozhi-compatible.bin"

const stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds = 60

var runStackChanOfficialSmokeFlashCommand = runStackChanOfficialSmokeFlashCommandExec

var runStackChanOfficialBaselineFlashCommand = runStackChanOfficialSmokeFlashCommandExec

var runStackChanOfficialPCMBridgeFlashCommand = runStackChanOfficialSmokeFlashCommandExec

var runStackChanOfficialXiaozhiCompatibleFlashCommand = runStackChanOfficialSmokeFlashCommandExec

var runStackChanOfficialXiaozhiCompatibleNVSCommand = runStackChanOfficialSmokeFlashCommandExec

var runStackChanOfficialPCMBridgeNVSCommand = runStackChanOfficialSmokeFlashCommandExec

type stackChanOfficialBaselineOptions struct {
	SourceRoot string
	WorkDir    string
	BuildDir   string
	IDFExport  string
	OutputDir  string
	DepCache   string
	Overlays   []string
	Execute    bool
}

type stackChanOfficialBaselineReport struct {
	SchemaVersion                 string                             `json:"schema_version"`
	GeneratedAtMS                 int64                              `json:"generated_at_ms"`
	Status                        string                             `json:"status"`
	FirmwareCandidate             string                             `json:"firmware_candidate,omitempty"`
	BuildLaneRole                 string                             `json:"build_lane_role,omitempty"`
	OfficialAvatarActionPreserved bool                               `json:"official_avatar_action_preserved"`
	OfficialXiaozhiStartPreserved bool                               `json:"official_xiaozhi_start_preserved"`
	MinimalBridgeScreen           bool                               `json:"minimal_bridge_screen"`
	SourceExportMode              string                             `json:"source_export_mode"`
	Execute                       bool                               `json:"execute"`
	SourceRoot                    string                             `json:"source_root"`
	SourceCommit                  string                             `json:"source_commit,omitempty"`
	SourceClean                   bool                               `json:"source_clean"`
	DirtyFileCount                int                                `json:"dirty_file_count"`
	WorkDir                       string                             `json:"work_dir"`
	BuildDir                      string                             `json:"build_dir"`
	IDFExport                     string                             `json:"idf_export"`
	Overlays                      []stackChanOfficialBaselineOverlay `json:"overlays,omitempty"`
	Evidence                      stackChanOfficialBaselineEvidence  `json:"evidence"`
	Build                         stackChanOfficialBaselineBuild     `json:"build"`
	Findings                      []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath                    string                             `json:"report_path,omitempty"`
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
	FetchExecuted       bool                                     `json:"fetch_executed"`
	BuildExecuted       bool                                     `json:"build_executed"`
	DependencyCacheUsed bool                                     `json:"dependency_cache_used,omitempty"`
	DependencyCacheRoot string                                   `json:"dependency_cache_root,omitempty"`
	FetchLogPath        string                                   `json:"fetch_log_path,omitempty"`
	BuildLogPath        string                                   `json:"build_log_path,omitempty"`
	Artifacts           []stackChanOfficialBaselineBuildArtifact `json:"artifacts,omitempty"`
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
	Confirm    string
	Execute    bool
}

type stackChanOfficialXiaozhiCompatibleFlashOptions struct {
	BuildDir              string
	IDFExport             string
	Port                  string
	EsptoolBefore         string
	WaitROM               bool
	WaitROMTimeoutSeconds int
	OutputDir             string
	Confirm               string
	Execute               bool
}

type stackChanOfficialXiaozhiCompatibleNVSOptions struct {
	IDFExport        string
	IDFPython        string
	Port             string
	OutputDir        string
	RunDir           string
	OTAURL           string
	WebSocketURL     string
	WebSocketVersion int
	WiFiSSID         string
	WiFiPassword     string
	Confirm          string
	Execute          bool
}

type stackChanOfficialPCMBridgeNVSOptions struct {
	IDFExport  string
	Port       string
	OutputDir  string
	RunDir     string
	DeviceID   string
	AudioWSURL string
	Confirm    string
	Execute    bool
}

type stackChanOfficialSmokeFlashReport struct {
	SchemaVersion            string                             `json:"schema_version"`
	GeneratedAtMS            int64                              `json:"generated_at_ms"`
	Status                   string                             `json:"status"`
	DryRun                   bool                               `json:"dry_run"`
	FlashAllowed             bool                               `json:"flash_allowed"`
	FlashExecuted            bool                               `json:"flash_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport   `json:"control_guard,omitempty"`
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
	FlashExecuted            bool                               `json:"flash_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport   `json:"control_guard,omitempty"`
	Port                     string                             `json:"port"`
	BuildDir                 string                             `json:"build_dir"`
	IDFExport                string                             `json:"idf_export"`
	DeviceID                 string                             `json:"device_id"`
	AudioWS                  stackChanOfficialPCMBridgeAudioWS  `json:"audio_ws"`
	FlashLogPath             string                             `json:"flash_log_path,omitempty"`
	NextRequiredConfirmation string                             `json:"next_required_confirmation,omitempty"`
	Parts                    []stackChanOfficialSmokeFlashPart  `json:"parts"`
	Findings                 []stackChanOfficialBaselineFinding `json:"findings,omitempty"`
	ReportPath               string                             `json:"report_path,omitempty"`
}

type stackChanOfficialXiaozhiCompatibleFlashReport struct {
	SchemaVersion            string                              `json:"schema_version"`
	GeneratedAtMS            int64                               `json:"generated_at_ms"`
	Status                   string                              `json:"status"`
	FirmwareCandidate        string                              `json:"firmware_candidate"`
	BuildLaneRole            string                              `json:"build_lane_role"`
	DryRun                   bool                                `json:"dry_run"`
	FlashAllowed             bool                                `json:"flash_allowed"`
	FlashExecuted            bool                                `json:"flash_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport    `json:"control_guard,omitempty"`
	Port                     string                              `json:"port"`
	EsptoolBefore            string                              `json:"esptool_before"`
	WaitROM                  bool                                `json:"wait_rom_download_mode,omitempty"`
	WaitROMTimeoutSeconds    int                                 `json:"wait_rom_timeout_seconds,omitempty"`
	BuildDirName             string                              `json:"build_dir_name"`
	IDFExportName            string                              `json:"idf_export_name,omitempty"`
	FlashLogFile             string                              `json:"flash_log_file,omitempty"`
	NextRequiredConfirmation string                              `json:"next_required_confirmation,omitempty"`
	Parts                    []stackChanOfficialXiaozhiFlashPart `json:"parts"`
	Findings                 []stackChanOfficialBaselineFinding  `json:"findings,omitempty"`
	ReportPath               string                              `json:"report_path,omitempty"`
}

type stackChanOfficialXiaozhiCompatibleNVSReport struct {
	SchemaVersion            string                                        `json:"schema_version"`
	GeneratedAtMS            int64                                         `json:"generated_at_ms"`
	Status                   string                                        `json:"status"`
	DryRun                   bool                                          `json:"dry_run"`
	WriteAllowed             bool                                          `json:"write_allowed"`
	WriteExecuted            bool                                          `json:"write_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport              `json:"control_guard,omitempty"`
	Port                     string                                        `json:"port"`
	IDFExport                string                                        `json:"idf_export"`
	RunDir                   string                                        `json:"run_dir"`
	OTA                      stackChanOfficialXiaozhiNVSOTA                `json:"ota"`
	WebSocket                stackChanOfficialXiaozhiNVSWebSocket          `json:"websocket"`
	Partition                stackChanOfficialPCMBridgeNVSPartition        `json:"partition"`
	Safety                   stackChanOfficialXiaozhiCompatibleNVSSafety   `json:"safety"`
	Tools                    stackChanOfficialPCMBridgeNVSTools            `json:"tools"`
	Summary                  *stackChanOfficialXiaozhiCompatibleNVSSummary `json:"summary,omitempty"`
	BackupPath               string                                        `json:"backup_path,omitempty"`
	BackupSHA256             string                                        `json:"backup_sha256,omitempty"`
	ProvisionCSVPath         string                                        `json:"provision_csv_path,omitempty"`
	ProvisionedBinPath       string                                        `json:"provisioned_bin_path,omitempty"`
	ProvisionedBinSHA256     string                                        `json:"provisioned_bin_sha256,omitempty"`
	ReadLogPath              string                                        `json:"read_log_path,omitempty"`
	ParseLogPath             string                                        `json:"parse_log_path,omitempty"`
	GenerateLogPath          string                                        `json:"generate_log_path,omitempty"`
	VerifyLogPath            string                                        `json:"verify_log_path,omitempty"`
	WriteLogPath             string                                        `json:"write_log_path,omitempty"`
	NextRequiredConfirmation string                                        `json:"next_required_confirmation,omitempty"`
	Findings                 []stackChanOfficialBaselineFinding            `json:"findings,omitempty"`
	ReportPath               string                                        `json:"report_path,omitempty"`
}

type stackChanOfficialPCMBridgeNVSReport struct {
	SchemaVersion            string                                 `json:"schema_version"`
	GeneratedAtMS            int64                                  `json:"generated_at_ms"`
	Status                   string                                 `json:"status"`
	DryRun                   bool                                   `json:"dry_run"`
	WriteAllowed             bool                                   `json:"write_allowed"`
	WriteExecuted            bool                                   `json:"write_executed"`
	ControlGuard             *runtimeguard.ControlGuardReport       `json:"control_guard,omitempty"`
	Port                     string                                 `json:"port"`
	IDFExport                string                                 `json:"idf_export"`
	RunDir                   string                                 `json:"run_dir"`
	DeviceID                 string                                 `json:"device_id"`
	AudioWS                  stackChanOfficialPCMBridgeAudioWS      `json:"audio_ws"`
	Partition                stackChanOfficialPCMBridgeNVSPartition `json:"partition"`
	Safety                   stackChanOfficialPCMBridgeNVSSafety    `json:"safety"`
	Tools                    stackChanOfficialPCMBridgeNVSTools     `json:"tools"`
	Summary                  *stackChanOfficialPCMBridgeNVSSummary  `json:"summary,omitempty"`
	BackupPath               string                                 `json:"backup_path,omitempty"`
	BackupSHA256             string                                 `json:"backup_sha256,omitempty"`
	ProvisionCSVPath         string                                 `json:"provision_csv_path,omitempty"`
	ProvisionedBinPath       string                                 `json:"provisioned_bin_path,omitempty"`
	ProvisionedBinSHA256     string                                 `json:"provisioned_bin_sha256,omitempty"`
	ReadLogPath              string                                 `json:"read_log_path,omitempty"`
	ParseLogPath             string                                 `json:"parse_log_path,omitempty"`
	GenerateLogPath          string                                 `json:"generate_log_path,omitempty"`
	VerifyLogPath            string                                 `json:"verify_log_path,omitempty"`
	WriteLogPath             string                                 `json:"write_log_path,omitempty"`
	NextRequiredConfirmation string                                 `json:"next_required_confirmation,omitempty"`
	Findings                 []stackChanOfficialBaselineFinding     `json:"findings,omitempty"`
	ReportPath               string                                 `json:"report_path,omitempty"`
}

type stackChanOfficialPCMBridgeNVSPartition struct {
	Offset    string `json:"offset"`
	SizeHex   string `json:"size_hex"`
	SizeBytes int    `json:"size_bytes"`
}

type stackChanOfficialPCMBridgeNVSSafety struct {
	BackupBeforeWrite       bool `json:"backup_before_write"`
	PreserveExistingEntries bool `json:"preserve_existing_entries"`
	OnlyMutatesA21Namespace bool `json:"only_mutates_a21_namespace"`
	ReportRedactsValues     bool `json:"report_redacts_values"`
}

type stackChanOfficialPCMBridgeNVSTools struct {
	NVSToolPath       string `json:"nvs_tool_path"`
	NVSGeneratorPath  string `json:"nvs_generator_path"`
	EsptoolModuleName string `json:"esptool_module_name"`
	IDFPythonPath     string `json:"idf_python_path,omitempty"`
}

type stackChanOfficialPCMBridgeNVSSummary struct {
	PreservedEntryCount     int  `json:"preserved_entry_count"`
	MutatedEntryCount       int  `json:"mutated_entry_count"`
	ExistingA21EntryCount   int  `json:"existing_a21_entry_count"`
	ServoCalibrationPresent bool `json:"servo_calibration_present"`
}

type stackChanOfficialXiaozhiNVSOTA struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Path   string `json:"path"`
}

type stackChanOfficialXiaozhiNVSWebSocket struct {
	Scheme          string `json:"scheme"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	Version         int    `json:"version"`
	TokenConfigured bool   `json:"token_configured"`
}

type stackChanOfficialXiaozhiCompatibleNVSSafety struct {
	BackupBeforeWrite                 bool `json:"backup_before_write"`
	PreserveExistingEntries           bool `json:"preserve_existing_entries"`
	OnlyMutatesXiaozhiConnectionKeys  bool `json:"only_mutates_xiaozhi_connection_keys"`
	PreservesWiFiCredentials          bool `json:"preserves_wifi_credentials"`
	AllowsExplicitWiFiCredentialWrite bool `json:"allows_explicit_wifi_credential_write"`
	ReportRedactsValues               bool `json:"report_redacts_values"`
}

type stackChanOfficialXiaozhiCompatibleNVSSummary struct {
	PreservedEntryCount          int  `json:"preserved_entry_count"`
	MutatedEntryCount            int  `json:"mutated_entry_count"`
	ExistingConnectionEntryCount int  `json:"existing_connection_entry_count"`
	ServoCalibrationPresent      bool `json:"servo_calibration_present"`
	WiFiCredentialsPreserved     bool `json:"wifi_credentials_preserved"`
	WiFiCredentialsWritten       bool `json:"wifi_credentials_written"`
	AppConfigMarkedConfigured    bool `json:"app_config_marked_configured"`
}

type officialStackChanProductLaneArtifactEvidence struct {
	SchemaVersion       string `json:"schema_version"`
	Status              string `json:"status"`
	Source              string `json:"source"`
	ReportPath          string `json:"report_path,omitempty"`
	SourceSchemaVersion string `json:"source_schema_version"`
	FirmwareCandidate   string `json:"firmware_candidate"`
	BuildLaneRole       string `json:"build_lane_role"`
	ArtifactFile        string `json:"artifact_file"`
	ArtifactSHA256      string `json:"artifact_sha256,omitempty"`
	FlashExecuted       bool   `json:"flash_executed,omitempty"`
	DryRun              bool   `json:"dry_run"`
	GeneratedAtMS       int64  `json:"generated_at_ms,omitempty"`
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

type stackChanOfficialXiaozhiFlashPart struct {
	Name      string `json:"name"`
	Offset    string `json:"offset"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func runStackChanOfficialBaseline(args []string, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialBaselineOptions{
		SourceRoot: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_SOURCE")),
		WorkDir:    firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_WORK_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-clean")),
		BuildDir:   firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		DepCache:   strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_DEP_CACHE")),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-baseline --source <m5stack-stackchan-repo> [--overlay firmware/stackchan-official/overlays/a21-official-audio-smoke.patch] [--execute] [--dep-cache <m5stack-stackchan-repo-or-firmware-dir>] [--work-dir /tmp/a21-stackchan-official-clean] [--build-dir /tmp/a21-stackchan-official-build] [--idf-export /path/to/export.sh] [--output-dir reports]")
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
		case "--dep-cache":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--dep-cache requires a value")
				return 2
			}
			i++
			options.DepCache = args[i]
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

func runStackChanOfficialBaselineFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialSmokeFlashOptions{
		BuildDir:  firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport: firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:      strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		OutputDir: "",
		Confirm:   "",
		Execute:   execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_BASELINE_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-baseline-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_BASELINE_DIAGNOSTIC] [--idf-export /path/to/export.sh] [--output-dir reports]")
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
			fmt.Fprintf(stderr, "unknown stackchan official baseline flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialBaselineFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official baseline flash requires --confirm %s\n", stackChanOfficialBaselineFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-baseline-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialBaselineFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official baseline flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialBaselineFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official baseline flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialBaselineFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official baseline flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialSmokeFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official baseline flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}
