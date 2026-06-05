package app

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"a21.local/a21/internal/runtimeguard"
)

const stackChanOfficialBaselineSchema = "a21.stackchan.official_baseline.v1"
const stackChanOfficialAudioSmokeFlashSchema = "a21.stackchan.official_audio_smoke_flash.v1"
const stackChanOfficialPCMBridgeFlashPlanSchema = "a21.stackchan.official_pcm_bridge_flash_plan.v1"
const stackChanOfficialPCMBridgeFlashExecutionSchema = "a21.stackchan.official_pcm_bridge_flash_execution.v1"
const stackChanOfficialXiaozhiCompatibleFlashPlanSchema = "a21.stackchan.official_xiaozhi_compatible_flash_plan.v1"
const stackChanOfficialXiaozhiCompatibleFlashExecutionSchema = "a21.stackchan.official_xiaozhi_compatible_flash_execution.v1"
const stackChanOfficialXiaozhiCompatibleNVSPlanSchema = "a21.stackchan.official_xiaozhi_compatible_nvs_plan.v1"
const stackChanOfficialXiaozhiCompatibleNVSExecutionSchema = "a21.stackchan.official_xiaozhi_compatible_nvs_execution.v1"
const stackChanOfficialPCMBridgeNVSPlanSchema = "a21.stackchan.official_pcm_bridge_nvs_plan.v1"
const stackChanOfficialPCMBridgeNVSExecutionSchema = "a21.stackchan.official_pcm_bridge_nvs_execution.v1"
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
			fmt.Fprintln(stdout, "a21 stackchan-official-audio-smoke-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE] [--idf-export /path/to/export.sh] [--output-dir reports]")
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

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialAudioSmokeFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official audio smoke flash requires --confirm %s\n", stackChanOfficialAudioSmokeFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-audio-smoke-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialSmokeFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official audio smoke flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
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

func runStackChanOfficialPCMBridgeFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialPCMBridgeFlashPlanOptions{
		BuildDir:   firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:       strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		AudioWSURL: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL")),
		OutputDir:  "",
		Confirm:    "",
		Execute:    execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-pcm-bridge-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX --device-id stackchan-001 --audio-ws-url ws://host:21080/ws/audio?device_id=stackchan-001 [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP] [--idf-export /path/to/export.sh] [--output-dir reports]")
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
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official pcm bridge flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialPCMBridgeAppFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official pcm bridge app flash requires --confirm %s\n", stackChanOfficialPCMBridgeAppFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-pcm-bridge-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialPCMBridgeFlashPlanReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official pcm bridge flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialPCMBridgeFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official pcm bridge flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialPCMBridgeFlashPlanReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official pcm bridge flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialPCMBridgeFlashPlan(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official pcm bridge flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialXiaozhiCompatibleFlash(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialXiaozhiCompatibleFlashOptions{
		BuildDir:              firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_BUILD_DIR"), filepath.Join(os.TempDir(), "a21-stackchan-official-build")),
		IDFExport:             firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:                  strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		EsptoolBefore:         firstNonEmpty(strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_ESPTOOL_BEFORE")), "default_reset"),
		WaitROM:               appEnvBool(os.Environ(), "A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM"),
		WaitROMTimeoutSeconds: parsePositiveIntOrDefault(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM_TIMEOUT_SECONDS"), stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds),
		OutputDir:             "",
		Confirm:               "",
		Execute:               execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 a21-stackchan-official-xiaozhi-compatible-flash --build-dir /tmp/a21-stackchan-official-build --port /dev/cu.usbmodemXXXX [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP] [--idf-export /path/to/export.sh] [--esptool-before default_reset|usb_reset|no_reset] [--wait-rom --wait-rom-timeout-seconds 60] [--output-dir reports]")
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
		case "--esptool-before":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--esptool-before requires a value")
				return 2
			}
			i++
			options.EsptoolBefore = args[i]
		case "--wait-rom":
			options.WaitROM = true
		case "--wait-rom-timeout-seconds":
			value, ok := parsePositiveIntCLIOption(args, &i, stderr, "--wait-rom-timeout-seconds")
			if !ok {
				return 2
			}
			options.WaitROMTimeoutSeconds = value
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
			fmt.Fprintf(stderr, "unknown stackchan official xiaozhi compatible flash option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialXiaozhiCompatibleAppFlashConfirm {
			fmt.Fprintf(stderr, "stackchan official xiaozhi compatible app flash requires --confirm %s\n", stackChanOfficialXiaozhiCompatibleAppFlashConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("a21-stackchan-official-xiaozhi-compatible-flash --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialXiaozhiCompatibleFlashReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official xiaozhi compatible flash: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialXiaozhiCompatibleFlash(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "flash_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official xiaozhi compatible flash report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialXiaozhiCompatibleFlashReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official xiaozhi compatible flash report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialXiaozhiCompatibleFlash(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official xiaozhi compatible flash report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialXiaozhiCompatibleNVS(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialXiaozhiCompatibleNVSOptions{
		IDFExport:        firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:             strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		RunDir:           firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS_RUN_DIR"), filepath.Join(".a21-run", "firmware", "official-xiaozhi-compatible-nvs")),
		OTAURL:           strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_OTA_URL")),
		WebSocketURL:     strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WEBSOCKET_URL")),
		WiFiSSID:         strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_SSID")),
		WiFiPassword:     strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_PASSWORD")),
		WebSocketVersion: 1,
		Execute:          execute,
	}
	if version := strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WEBSOCKET_VERSION")); version != "" {
		parsed, err := strconv.Atoi(version)
		if err == nil {
			options.WebSocketVersion = parsed
		}
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 a21-stackchan-official-xiaozhi-compatible-nvs --port /dev/cu.usbmodemXXXX --ota-url http://LAN:21080/xiaozhi/ota/ --websocket-url ws://LAN:21080/v1/xiaozhi [--websocket-version 1] [--wifi-ssid SSID --wifi-password PASSWORD] [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_NVS] [--idf-export /path/to/export.sh] [--run-dir .a21-run/firmware/official-xiaozhi-compatible-nvs] [--output-dir reports]")
			return 0
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
		case "--run-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--run-dir requires a value")
				return 2
			}
			i++
			options.RunDir = args[i]
		case "--ota-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--ota-url requires a value")
				return 2
			}
			i++
			options.OTAURL = args[i]
		case "--websocket-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--websocket-url requires a value")
				return 2
			}
			i++
			options.WebSocketURL = args[i]
		case "--websocket-version":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--websocket-version requires a value")
				return 2
			}
			i++
			version, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintln(stderr, "--websocket-version must be an integer")
				return 2
			}
			options.WebSocketVersion = version
		case "--wifi-ssid":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wifi-ssid requires a value")
				return 2
			}
			i++
			options.WiFiSSID = strings.TrimSpace(args[i])
		case "--wifi-password":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--wifi-password requires a value")
				return 2
			}
			i++
			options.WiFiPassword = strings.TrimSpace(args[i])
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
			fmt.Fprintf(stderr, "unknown stackchan official xiaozhi compatible nvs option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialXiaozhiCompatibleNVSConfirm {
			fmt.Fprintf(stderr, "stackchan official xiaozhi compatible nvs execute requires --confirm %s\n", stackChanOfficialXiaozhiCompatibleNVSConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("a21-stackchan-official-xiaozhi-compatible-nvs --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialXiaozhiCompatibleNVSReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official xiaozhi compatible nvs: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
		if err := executeStackChanOfficialXiaozhiCompatibleNVS(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "nvs_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official xiaozhi compatible nvs report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialXiaozhiCompatibleNVSReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official xiaozhi compatible nvs report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialXiaozhiCompatibleNVS(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official xiaozhi compatible nvs report: %v\n", err)
		return 1
	}
	if report.Status == "failed" {
		return 1
	}
	return 0
}

func runStackChanOfficialPCMBridgeNVS(args []string, execute bool, stdout io.Writer, stderr io.Writer) int {
	options := stackChanOfficialPCMBridgeNVSOptions{
		IDFExport:  firstNonEmpty(os.Getenv("A21_IDF_EXPORT"), "/Users/jiyurun/esp/esp-idf-v5.5.2/export.sh"),
		Port:       strings.TrimSpace(os.Getenv("A21_UPLOAD_PORT")),
		RunDir:     firstNonEmpty(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_RUN_DIR"), filepath.Join(".a21-run", "firmware", "official-pcm-bridge-nvs")),
		DeviceID:   firstNonEmpty(strings.TrimSpace(os.Getenv("A21_DEVICE_ID")), "stackchan-001"),
		AudioWSURL: strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL")),
		Execute:    execute,
	}
	if execute {
		options.Confirm = strings.TrimSpace(os.Getenv("A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM"))
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-official-pcm-bridge-nvs --port /dev/cu.usbmodemXXXX --device-id stackchan-001 --audio-ws-url ws://host:21080/ws/audio?device_id=stackchan-001 [--execute --confirm WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS] [--idf-export /path/to/export.sh] [--run-dir .a21-run/firmware/official-pcm-bridge-nvs] [--output-dir reports]")
			return 0
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
		case "--run-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--run-dir requires a value")
				return 2
			}
			i++
			options.RunDir = args[i]
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
		case "--confirm":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--confirm requires a value")
				return 2
			}
			i++
			options.Confirm = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan official pcm bridge nvs option %q\n", args[i])
			return 2
		}
	}

	var controlGuard runtimeguard.ControlGuardReport
	if execute {
		if options.Confirm != stackChanOfficialPCMBridgeNVSConfirm {
			fmt.Fprintf(stderr, "stackchan official pcm bridge nvs execute requires --confirm %s\n", stackChanOfficialPCMBridgeNVSConfirm)
			return 2
		}
		var code int
		controlGuard, code = requireA21ControlAllowed("stackchan-official-pcm-bridge-nvs --execute", stderr)
		if code != 0 {
			return code
		}
	}
	report, err := buildStackChanOfficialPCMBridgeNVSReport(options)
	if err != nil {
		fmt.Fprintf(stderr, "stackchan official pcm bridge nvs: %v\n", err)
		return 1
	}
	if execute {
		report.ControlGuard = &controlGuard
	}
	if execute {
		if err := executeStackChanOfficialPCMBridgeNVS(context.Background(), options, &report); err != nil {
			report.Status = "failed"
			report.Findings = append(report.Findings, stackChanOfficialBaselineFinding{
				Code:    "nvs_execute_failed",
				Message: err.Error(),
			})
		}
	}
	if options.OutputDir != "" {
		if err := validateA21ReportDir(options.OutputDir); err != nil {
			fmt.Fprintf(stderr, "official pcm bridge nvs report dir invalid: %v\n", err)
			return 1
		}
		reportPath, err := writeStackChanOfficialPCMBridgeNVSReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write official pcm bridge nvs report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanOfficialPCMBridgeNVS(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode official pcm bridge nvs report: %v\n", err)
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
		SchemaVersion:    stackChanOfficialBaselineSchema,
		GeneratedAtMS:    time.Now().UnixMilli(),
		Status:           "ready",
		SourceExportMode: "git_head_archive_read_only",
		Execute:          options.Execute,
		SourceRoot:       sourceRoot,
		SourceCommit:     strings.TrimSpace(commit),
		SourceClean:      dirtyFiles == 0,
		DirtyFileCount:   dirtyFiles,
		WorkDir:          filepath.Clean(options.WorkDir),
		BuildDir:         filepath.Clean(options.BuildDir),
		IDFExport:        filepath.Clean(options.IDFExport),
	}
	for _, overlay := range options.Overlays {
		report.Overlays = append(report.Overlays, stackChanOfficialBaselineOverlay{
			Path: filepath.Clean(overlay),
		})
	}
	report.Evidence = inspectStackChanOfficialEvidence(sourceRoot)
	applyStackChanOfficialCandidateContract(&report, sourceRoot, options.Overlays)
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
	if containsLegacyIdentityPathToken(buildDir) {
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
	if containsLegacyIdentityPathToken(buildDir) {
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
	schema := stackChanOfficialPCMBridgeFlashPlanSchema
	if options.Execute {
		schema = stackChanOfficialPCMBridgeFlashExecutionSchema
	}
	return stackChanOfficialPCMBridgeFlashPlanReport{
		SchemaVersion:            schema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		BuildDir:                 buildDir,
		IDFExport:                filepath.Clean(options.IDFExport),
		DeviceID:                 deviceID,
		AudioWS:                  audioWS,
		NextRequiredConfirmation: "bridge_nvs_provisioning_and_flash_execute_guard_not_implemented",
		Parts:                    parts,
	}, nil
}

func buildStackChanOfficialXiaozhiCompatibleFlashReport(options stackChanOfficialXiaozhiCompatibleFlashOptions) (stackChanOfficialXiaozhiCompatibleFlashReport, error) {
	buildDir := filepath.Clean(options.BuildDir)
	if err := validateA21OfficialScratchDir(buildDir); err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("build dir invalid: %w", err)
	}
	if containsLegacyIdentityPathToken(buildDir) {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("build dir contains forbidden legacy identity")
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	esptoolBefore, err := validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(options.EsptoolBefore)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	waitROMTimeoutSeconds := stackChanOfficialXiaozhiCompatibleWaitROMTimeoutSeconds(options)
	if options.WaitROM {
		if esptoolBefore != "no_reset" {
			return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("wait-rom requires --esptool-before no_reset so the manual ROM download state is not reset before flashing")
		}
		if waitROMTimeoutSeconds <= 0 || waitROMTimeoutSeconds > 300 {
			return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("wait-rom timeout seconds must be between 1 and 300")
		}
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}
	parts, err := collectOfficialXiaozhiCompatibleFlashParts(buildDir)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleFlashReport{}, err
	}
	schema := stackChanOfficialXiaozhiCompatibleFlashPlanSchema
	if options.Execute {
		schema = stackChanOfficialXiaozhiCompatibleFlashExecutionSchema
	}
	return stackChanOfficialXiaozhiCompatibleFlashReport{
		SchemaVersion:            schema,
		GeneratedAtMS:            time.Now().UnixMilli(),
		Status:                   "ready",
		FirmwareCandidate:        stackChanOfficialXiaozhiCompatibleFirmwareCandidate,
		BuildLaneRole:            "product_candidate",
		DryRun:                   !options.Execute,
		FlashAllowed:             false,
		FlashExecuted:            false,
		Port:                     options.Port,
		EsptoolBefore:            esptoolBefore,
		WaitROM:                  options.WaitROM,
		WaitROMTimeoutSeconds:    waitROMTimeoutSeconds,
		BuildDirName:             filepath.Base(buildDir),
		IDFExportName:            filepath.Base(filepath.Clean(options.IDFExport)),
		NextRequiredConfirmation: "a21-stackchan-official-xiaozhi-compatible-flash-execute_with_confirmation_token",
		Parts:                    parts,
	}, nil
}

func buildStackChanOfficialXiaozhiCompatibleNVSReport(options stackChanOfficialXiaozhiCompatibleNVSOptions) (stackChanOfficialXiaozhiCompatibleNVSReport, error) {
	runDir := filepath.Clean(options.RunDir)
	if err := validateA21OfficialRunDir(runDir); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("run dir invalid: %w", err)
	}
	ota, err := parseOfficialXiaozhiNVSOTAURL(options.OTAURL)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	websocket, err := parseOfficialXiaozhiNVSWebSocketURL(options.WebSocketURL, options.WebSocketVersion)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	wifiCredentialsRequested, err := validateOfficialXiaozhiNVSWiFiCredentials(options.WiFiSSID, options.WiFiPassword)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialXiaozhiCompatibleNVSReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}

	schema := stackChanOfficialXiaozhiCompatibleNVSPlanSchema
	if options.Execute {
		schema = stackChanOfficialXiaozhiCompatibleNVSExecutionSchema
	}
	return stackChanOfficialXiaozhiCompatibleNVSReport{
		SchemaVersion: schema,
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "ready",
		DryRun:        !options.Execute,
		WriteAllowed:  false,
		WriteExecuted: false,
		Port:          options.Port,
		IDFExport:     filepath.Clean(options.IDFExport),
		RunDir:        runDir,
		OTA:           ota,
		WebSocket:     websocket,
		Partition: stackChanOfficialPCMBridgeNVSPartition{
			Offset:    stackChanOfficialPCMBridgeNVSOffset,
			SizeHex:   stackChanOfficialPCMBridgeNVSSizeHex,
			SizeBytes: stackChanOfficialPCMBridgeNVSSizeBytes,
		},
		Safety: stackChanOfficialXiaozhiCompatibleNVSSafety{
			BackupBeforeWrite:                 true,
			PreserveExistingEntries:           true,
			OnlyMutatesXiaozhiConnectionKeys:  true,
			PreservesWiFiCredentials:          !wifiCredentialsRequested,
			AllowsExplicitWiFiCredentialWrite: wifiCredentialsRequested,
			ReportRedactsValues:               true,
		},
		Tools: stackChanOfficialPCMBridgeNVSTools{
			NVSToolPath:       officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_tool/nvs_tool.py"),
			NVSGeneratorPath:  officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_generator/nvs_partition_gen.py"),
			EsptoolModuleName: "esptool",
		},
		NextRequiredConfirmation: "a21-stackchan-official-xiaozhi-compatible-nvs-execute_with_confirmation_token",
	}, nil
}

func buildStackChanOfficialPCMBridgeNVSReport(options stackChanOfficialPCMBridgeNVSOptions) (stackChanOfficialPCMBridgeNVSReport, error) {
	runDir := filepath.Clean(options.RunDir)
	if err := validateA21OfficialRunDir(runDir); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("run dir invalid: %w", err)
	}
	deviceID := strings.TrimSpace(options.DeviceID)
	if err := validateOfficialPCMBridgeDeviceID(deviceID); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	audioWS, err := parseOfficialPCMBridgeAudioWSURL(options.AudioWSURL, deviceID)
	if err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	if err := validateOfficialSmokeUploadPort(options.Port); err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, err
	}
	usage, err := detectFirmwareUploadPortUsage(options.Port)
	if err != nil {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("inspect upload port: %w", err)
	}
	if !usage.Exists {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("upload port %s does not exist", options.Port)
	}
	if usage.InUse {
		return stackChanOfficialPCMBridgeNVSReport{}, fmt.Errorf("upload port %s is already in use: %s", options.Port, usage.Detail)
	}

	schema := stackChanOfficialPCMBridgeNVSPlanSchema
	if options.Execute {
		schema = stackChanOfficialPCMBridgeNVSExecutionSchema
	}
	return stackChanOfficialPCMBridgeNVSReport{
		SchemaVersion: schema,
		GeneratedAtMS: time.Now().UnixMilli(),
		Status:        "ready",
		DryRun:        !options.Execute,
		WriteAllowed:  false,
		WriteExecuted: false,
		Port:          options.Port,
		IDFExport:     filepath.Clean(options.IDFExport),
		RunDir:        runDir,
		DeviceID:      deviceID,
		AudioWS:       audioWS,
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
		Tools: stackChanOfficialPCMBridgeNVSTools{
			NVSToolPath:       officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_tool/nvs_tool.py"),
			NVSGeneratorPath:  officialIDFToolPath(options.IDFExport, "components/nvs_flash/nvs_partition_generator/nvs_partition_gen.py"),
			EsptoolModuleName: "esptool",
		},
		NextRequiredConfirmation: "stackchan-official-pcm-bridge-nvs-execute_with_confirmation_token",
	}, nil
}

func collectOfficialSmokeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-audio-smoke.bin")
}

func collectOfficialPCMBridgeFlashParts(buildDir string) ([]stackChanOfficialSmokeFlashPart, error) {
	return collectOfficialFlashPartsForApp(buildDir, "a21-stackchan-official-pcm-bridge.bin")
}

func collectOfficialXiaozhiCompatibleFlashParts(buildDir string) ([]stackChanOfficialXiaozhiFlashPart, error) {
	fullParts, err := collectOfficialFlashPartsForApp(buildDir, stackChanOfficialXiaozhiCompatibleAppBinary)
	if err != nil {
		return nil, err
	}
	parts := make([]stackChanOfficialXiaozhiFlashPart, 0, len(fullParts))
	for _, part := range fullParts {
		parts = append(parts, stackChanOfficialXiaozhiFlashPart{
			Name:      part.Name,
			Offset:    part.Offset,
			File:      filepath.Base(part.Path),
			SHA256:    part.SHA256,
			SizeBytes: part.SizeBytes,
		})
	}
	return parts, nil
}

type officialStackChanProductLaneEvidenceCandidate struct {
	evidence     officialStackChanProductLaneArtifactEvidence
	priority     int
	observedAtMS int64
}

type officialStackChanProductLaneRawPart struct {
	Name        string `json:"name"`
	Offset      string `json:"offset"`
	FlashOffset string `json:"flash_offset"`
	File        string `json:"file"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
}

type officialStackChanProductLaneRawReport struct {
	SchemaVersion     string                                `json:"schema_version"`
	GeneratedAtMS     int64                                 `json:"generated_at_ms"`
	Status            string                                `json:"status"`
	FirmwareCandidate string                                `json:"firmware_candidate"`
	BuildLaneRole     string                                `json:"build_lane_role"`
	DryRun            bool                                  `json:"dry_run"`
	FlashExecuted     bool                                  `json:"flash_executed"`
	Parts             []officialStackChanProductLaneRawPart `json:"parts"`
	Build             officialStackChanProductLaneRawBuild  `json:"build"`
}

type officialStackChanProductLaneRawBuild struct {
	BuildExecuted bool                                  `json:"build_executed"`
	Artifacts     []officialStackChanProductLaneRawPart `json:"artifacts"`
}

func discoverOfficialStackChanProductLaneArtifactEvidence(reportDir string) *officialStackChanProductLaneArtifactEvidence {
	reportDir = filepath.Clean(strings.TrimSpace(reportDir))
	if reportDir == "" || reportDir == "." || containsLegacyIdentityPathToken(reportDir) {
		return nil
	}
	info, err := os.Stat(reportDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	patterns := []string{
		"a21-stackchan-official-xiaozhi-compatible-flash-*.json",
		"a21-stackchan-official-baseline-*.json",
	}
	candidates := make([]officialStackChanProductLaneEvidenceCandidate, 0)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(reportDir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			candidate, ok := readOfficialStackChanProductLaneArtifactEvidence(match)
			if ok {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority > candidates[j].priority
		}
		if candidates[i].observedAtMS != candidates[j].observedAtMS {
			return candidates[i].observedAtMS > candidates[j].observedAtMS
		}
		return candidates[i].evidence.ReportPath > candidates[j].evidence.ReportPath
	})
	evidence := candidates[0].evidence
	return &evidence
}

func readOfficialStackChanProductLaneArtifactEvidence(reportPath string) (officialStackChanProductLaneEvidenceCandidate, bool) {
	cleanPath := filepath.Clean(reportPath)
	if containsLegacyIdentityPathToken(cleanPath) {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	var raw officialStackChanProductLaneRawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
	generatedAtMS := raw.GeneratedAtMS
	if generatedAtMS == 0 {
		if info, err := os.Stat(cleanPath); err == nil {
			generatedAtMS = info.ModTime().UnixMilli()
		}
	}
	switch raw.SchemaVersion {
	case stackChanOfficialXiaozhiCompatibleFlashExecutionSchema, stackChanOfficialXiaozhiCompatibleFlashPlanSchema:
		file, sha, ok := officialXiaozhiCompatibleArtifactFromParts(raw.Parts)
		if !ok ||
			strings.TrimSpace(raw.FirmwareCandidate) != stackChanOfficialXiaozhiCompatibleFirmwareCandidate ||
			strings.TrimSpace(raw.BuildLaneRole) != "product_candidate" {
			return officialStackChanProductLaneEvidenceCandidate{}, false
		}
		status := "planned"
		source := "official_xiaozhi_compatible_flash_plan"
		priority := 1
		if raw.SchemaVersion == stackChanOfficialXiaozhiCompatibleFlashExecutionSchema &&
			raw.FlashExecuted &&
			!raw.DryRun &&
			strings.TrimSpace(raw.Status) == "passed" {
			status = "satisfied"
			source = "official_xiaozhi_compatible_flash_execution"
			priority = 3
		}
		return officialStackChanProductLaneEvidenceCandidate{
			evidence: officialStackChanProductLaneArtifactEvidence{
				SchemaVersion:       officialProductLaneArtifactEvidenceSchema,
				Status:              status,
				Source:              source,
				ReportPath:          cleanPath,
				SourceSchemaVersion: raw.SchemaVersion,
				FirmwareCandidate:   raw.FirmwareCandidate,
				BuildLaneRole:       raw.BuildLaneRole,
				ArtifactFile:        file,
				ArtifactSHA256:      sha,
				FlashExecuted:       raw.FlashExecuted,
				DryRun:              raw.DryRun,
				GeneratedAtMS:       generatedAtMS,
			},
			priority:     priority,
			observedAtMS: generatedAtMS,
		}, true
	case stackChanOfficialBaselineSchema:
		file, sha, ok := officialXiaozhiCompatibleArtifactFromParts(raw.Build.Artifacts)
		if !ok ||
			strings.TrimSpace(raw.FirmwareCandidate) != stackChanOfficialXiaozhiCompatibleFirmwareCandidate ||
			strings.TrimSpace(raw.BuildLaneRole) != "product_candidate" ||
			strings.TrimSpace(raw.Status) != "passed" ||
			!raw.Build.BuildExecuted {
			return officialStackChanProductLaneEvidenceCandidate{}, false
		}
		return officialStackChanProductLaneEvidenceCandidate{
			evidence: officialStackChanProductLaneArtifactEvidence{
				SchemaVersion:       officialProductLaneArtifactEvidenceSchema,
				Status:              "build_available",
				Source:              "official_xiaozhi_compatible_build",
				ReportPath:          cleanPath,
				SourceSchemaVersion: raw.SchemaVersion,
				FirmwareCandidate:   raw.FirmwareCandidate,
				BuildLaneRole:       raw.BuildLaneRole,
				ArtifactFile:        file,
				ArtifactSHA256:      sha,
				DryRun:              false,
				GeneratedAtMS:       generatedAtMS,
			},
			priority:     2,
			observedAtMS: generatedAtMS,
		}, true
	default:
		return officialStackChanProductLaneEvidenceCandidate{}, false
	}
}

func officialXiaozhiCompatibleArtifactFromParts(parts []officialStackChanProductLaneRawPart) (string, string, bool) {
	for _, part := range parts {
		if strings.TrimSpace(part.Name) != "app" {
			continue
		}
		name := filepath.Base(firstNonEmpty(strings.TrimSpace(part.File), strings.TrimSpace(part.Path)))
		if name != stackChanOfficialXiaozhiCompatibleAppBinary {
			return "", "", false
		}
		offset := firstNonEmpty(strings.TrimSpace(part.Offset), strings.TrimSpace(part.FlashOffset))
		if offset != "" && offset != "0x20000" {
			return "", "", false
		}
		sha := strings.ToLower(strings.TrimSpace(part.SHA256))
		if sha != "" && !isSHA256Hex(sha) {
			return "", "", false
		}
		return name, sha, true
	}
	return "", "", false
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func officialProductLaneEvidenceSatisfiesArtifact(evidence *officialStackChanProductLaneArtifactEvidence) bool {
	if evidence == nil {
		return false
	}
	return evidence.Status == "satisfied" || evidence.Status == "build_available"
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
		if containsLegacyIdentityPathToken(fullPath) {
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
	if err := validateOfficialAssetsPartitionCapacity(buildDir); err != nil {
		return nil, err
	}
	return parts, nil
}

type officialBinaryPartitionEntry struct {
	Label  string
	Offset uint32
	Size   uint32
}

func validateOfficialAssetsPartitionCapacity(buildDir string) error {
	entries, parsed, err := readOfficialBinaryPartitionTable(filepath.Join(buildDir, "partition_table", "partition-table.bin"))
	if err != nil {
		return err
	}
	if !parsed {
		return nil
	}
	var assets *officialBinaryPartitionEntry
	for i := range entries {
		if entries[i].Label == "assets" {
			assets = &entries[i]
			break
		}
	}
	if assets == nil {
		return fmt.Errorf("partition table missing assets partition")
	}
	assetsPath := filepath.Join(buildDir, "generated_assets.bin")
	stat, err := os.Stat(assetsPath)
	if err != nil {
		return fmt.Errorf("generated assets image missing: %w", err)
	}
	if stat.Size() > int64(assets.Size) {
		return fmt.Errorf("generated assets image size %d exceeds assets partition size %d at 0x%x", stat.Size(), assets.Size, assets.Offset)
	}
	return nil
}

func readOfficialBinaryPartitionTable(path string) ([]officialBinaryPartitionEntry, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read partition table: %w", err)
	}
	if len(data) < 32 || data[0] != 0xaa || data[1] != 0x50 {
		return nil, false, nil
	}
	entries := make([]officialBinaryPartitionEntry, 0, 8)
	for offset := 0; offset+32 <= len(data); offset += 32 {
		entry := data[offset : offset+32]
		if entry[0] == 0xeb && entry[1] == 0xeb {
			break
		}
		if entry[0] != 0xaa || entry[1] != 0x50 {
			break
		}
		labelBytes := entry[12:28]
		labelEnd := len(labelBytes)
		for i, b := range labelBytes {
			if b == 0 {
				labelEnd = i
				break
			}
		}
		entries = append(entries, officialBinaryPartitionEntry{
			Label:  string(labelBytes[:labelEnd]),
			Offset: binary.LittleEndian.Uint32(entry[4:8]),
			Size:   binary.LittleEndian.Uint32(entry[8:12]),
		})
	}
	return entries, true, nil
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

func parseOfficialXiaozhiNVSOTAURL(rawURL string) (stackChanOfficialXiaozhiNVSOTA, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("--ota-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url contains forbidden legacy identity")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("parse ota url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url must use http or https")
	}
	if parsed.User != nil {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url host is required")
	}
	if parsed.Path != "/xiaozhi/ota/" && parsed.Path != "/xiaozhi/ota" {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url path must be /xiaozhi/ota/")
	}
	if isLoopbackOrUnspecifiedHost(parsed.Hostname()) {
		return stackChanOfficialXiaozhiNVSOTA{}, fmt.Errorf("ota url host must be reachable by the physical device")
	}
	return stackChanOfficialXiaozhiNVSOTA{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
	}, nil
}

func parseOfficialXiaozhiNVSWebSocketURL(rawURL string, version int) (stackChanOfficialXiaozhiNVSWebSocket, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("--websocket-url is required")
	}
	if containsLegacyIdentity(rawURL) {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url contains forbidden legacy identity")
	}
	if version <= 0 {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket version must be positive")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("parse websocket url: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url must use ws or wss")
	}
	if parsed.User != nil {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url must not contain credentials")
	}
	if parsed.Host == "" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url host is required")
	}
	if parsed.Path != "/v1/xiaozhi" {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url path must be /v1/xiaozhi")
	}
	if isLoopbackOrUnspecifiedHost(parsed.Hostname()) {
		return stackChanOfficialXiaozhiNVSWebSocket{}, fmt.Errorf("websocket url host must be reachable by the physical device")
	}
	return stackChanOfficialXiaozhiNVSWebSocket{
		Scheme:          parsed.Scheme,
		Host:            parsed.Host,
		Path:            parsed.Path,
		Version:         version,
		TokenConfigured: false,
	}, nil
}

func validateOfficialXiaozhiNVSWiFiCredentials(ssid string, password string) (bool, error) {
	ssid = strings.TrimSpace(ssid)
	password = strings.TrimSpace(password)
	if ssid == "" && password == "" {
		return false, nil
	}
	if ssid == "" || password == "" {
		return false, fmt.Errorf("wifi ssid and password must be provided together")
	}
	if len(ssid) > 32 {
		return false, fmt.Errorf("wifi ssid is too long")
	}
	if len(password) < 8 || len(password) > 63 {
		return false, fmt.Errorf("wifi password length is invalid")
	}
	for _, value := range []string{ssid, password} {
		if containsLegacyIdentity(value) || strings.Contains(value, "\n") || strings.Contains(value, "\r") || strings.Contains(value, "\x00") {
			return false, fmt.Errorf("wifi credentials are invalid")
		}
	}
	return true, nil
}

type stackChanNVSMinimalEntry struct {
	Namespace string      `json:"namespace"`
	Key       string      `json:"key"`
	Encoding  string      `json:"encoding"`
	Data      interface{} `json:"data"`
	State     string      `json:"state"`
	IsEmpty   bool        `json:"is_empty"`
}

func readStackChanNVSMinimalEntries(path string) ([]stackChanNVSMinimalEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read NVS JSON dump: %w", err)
	}
	var entries []stackChanNVSMinimalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode NVS JSON dump: %w", err)
	}
	return entries, nil
}

func writeOfficialPCMBridgeNVSCSV(writer io.Writer, entries []stackChanNVSMinimalEntry, deviceID string, audioWSURL string) (stackChanOfficialPCMBridgeNVSSummary, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{"key", "type", "encoding", "value"}); err != nil {
		return stackChanOfficialPCMBridgeNVSSummary{}, err
	}

	namespaceOrder := make([]string, 0)
	seenNamespaces := make(map[string]bool)
	grouped := make(map[string][][]string)
	summary := stackChanOfficialPCMBridgeNVSSummary{}
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		namespace := strings.TrimSpace(entry.Namespace)
		key := strings.TrimSpace(entry.Key)
		if namespace == "" || key == "" {
			continue
		}
		if namespace == "a21" && (key == "device_id" || key == "audio_ws_url") {
			summary.ExistingA21EntryCount += 1
			continue
		}
		encoding, err := nvsCSVEncoding(entry.Encoding)
		if err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		value, err := nvsCSVValue(entry.Data)
		if err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
		grouped[namespace] = append(grouped[namespace], []string{key, "data", encoding, value})
		summary.PreservedEntryCount += 1
		if namespace == "servo" && (key == "zero_pos_1" || key == "zero_pos_2") {
			if hasNVSEntry(entries, "servo", "zero_pos_1") && hasNVSEntry(entries, "servo", "zero_pos_2") {
				summary.ServoCalibrationPresent = true
			}
		}
	}
	if !seenNamespaces["a21"] {
		seenNamespaces["a21"] = true
		namespaceOrder = append(namespaceOrder, "a21")
	}
	grouped["a21"] = append(grouped["a21"],
		[]string{"device_id", "data", "string", deviceID},
		[]string{"audio_ws_url", "data", "string", audioWSURL},
	)
	summary.MutatedEntryCount = 2

	for _, namespace := range namespaceOrder {
		if err := csvWriter.Write([]string{namespace, "namespace", "", ""}); err != nil {
			return stackChanOfficialPCMBridgeNVSSummary{}, err
		}
		for _, row := range grouped[namespace] {
			if err := csvWriter.Write(row); err != nil {
				return stackChanOfficialPCMBridgeNVSSummary{}, err
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return stackChanOfficialPCMBridgeNVSSummary{}, err
	}
	return summary, nil
}

func writeOfficialXiaozhiCompatibleNVSCSV(writer io.Writer, entries []stackChanNVSMinimalEntry, otaURL string, websocketURL string, websocketVersion int, wifiSSID string, wifiPassword string) (stackChanOfficialXiaozhiCompatibleNVSSummary, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{"key", "type", "encoding", "value"}); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
	}
	wifiCredentialsRequested := strings.TrimSpace(wifiSSID) != "" || strings.TrimSpace(wifiPassword) != ""

	namespaceOrder := make([]string, 0)
	seenNamespaces := make(map[string]bool)
	grouped := make(map[string][][]string)
	summary := stackChanOfficialXiaozhiCompatibleNVSSummary{}
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		namespace := strings.TrimSpace(entry.Namespace)
		key := strings.TrimSpace(entry.Key)
		if namespace == "" || key == "" {
			continue
		}
		if isOfficialXiaozhiConnectionNVSKey(namespace, key) {
			summary.ExistingConnectionEntryCount += 1
			continue
		}
		encoding, err := nvsCSVEncoding(entry.Encoding)
		if err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		value, err := nvsCSVValue(entry.Data)
		if err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
		grouped[namespace] = append(grouped[namespace], []string{key, "data", encoding, value})
		summary.PreservedEntryCount += 1
		if namespace == "servo" && (key == "zero_pos_1" || key == "zero_pos_2") {
			if hasNVSEntry(entries, "servo", "zero_pos_1") && hasNVSEntry(entries, "servo", "zero_pos_2") {
				summary.ServoCalibrationPresent = true
			}
		}
	}
	summary.WiFiCredentialsPreserved = !wifiCredentialsRequested && hasNVSEntry(entries, "wifi", "ssid") && hasNVSEntry(entries, "wifi", "password")
	appConfigShouldMarkConfigured := wifiCredentialsRequested || summary.WiFiCredentialsPreserved
	for _, namespace := range []string{"wifi", "websocket"} {
		if !seenNamespaces[namespace] {
			seenNamespaces[namespace] = true
			namespaceOrder = append(namespaceOrder, namespace)
		}
	}
	if appConfigShouldMarkConfigured && !seenNamespaces["app_config"] {
		seenNamespaces["app_config"] = true
		namespaceOrder = append(namespaceOrder, "app_config")
	}
	if wifiCredentialsRequested {
		grouped["wifi"] = removeNVSRows(grouped["wifi"], "ssid", "password")
		grouped["wifi"] = append(grouped["wifi"],
			[]string{"ssid", "data", "string", strings.TrimSpace(wifiSSID)},
			[]string{"password", "data", "string", strings.TrimSpace(wifiPassword)},
		)
		summary.WiFiCredentialsWritten = true
	}
	grouped["wifi"] = append(grouped["wifi"], []string{"ota_url", "data", "string", otaURL})
	grouped["websocket"] = append(grouped["websocket"],
		[]string{"url", "data", "string", websocketURL},
		[]string{"version", "data", "u32", strconv.Itoa(websocketVersion)},
	)
	if appConfigShouldMarkConfigured {
		grouped["app_config"] = removeNVSRows(grouped["app_config"], "is_configed")
		grouped["app_config"] = append(grouped["app_config"], []string{"is_configed", "data", "u8", "1"})
		summary.AppConfigMarkedConfigured = true
	}
	summary.MutatedEntryCount = 3
	if wifiCredentialsRequested {
		summary.MutatedEntryCount += 2
	}
	if appConfigShouldMarkConfigured {
		summary.MutatedEntryCount += 1
	}

	for _, namespace := range namespaceOrder {
		if err := csvWriter.Write([]string{namespace, "namespace", "", ""}); err != nil {
			return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
		}
		for _, row := range grouped[namespace] {
			if err := csvWriter.Write(row); err != nil {
				return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return stackChanOfficialXiaozhiCompatibleNVSSummary{}, err
	}
	return summary, nil
}

func removeNVSRows(rows [][]string, keys ...string) [][]string {
	if len(rows) == 0 || len(keys) == 0 {
		return rows
	}
	remove := make(map[string]bool, len(keys))
	for _, key := range keys {
		remove[key] = true
	}
	filtered := rows[:0]
	for _, row := range rows {
		if len(row) > 0 && remove[row[0]] {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func isOfficialXiaozhiConnectionNVSKey(namespace string, key string) bool {
	switch namespace {
	case "wifi":
		return key == "ota_url"
	case "websocket":
		return key == "url" || key == "token" || key == "version"
	case "app_config":
		return key == "is_configed"
	default:
		return false
	}
}

func hasNVSEntry(entries []stackChanNVSMinimalEntry, namespace string, key string) bool {
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		if entry.Namespace == namespace && entry.Key == key {
			return true
		}
	}
	return false
}

func nvsCSVEncoding(encoding string) (string, error) {
	switch encoding {
	case "string":
		return "string", nil
	case "blob_data":
		return "base64", nil
	case "uint8_t":
		return "u8", nil
	case "int8_t":
		return "i8", nil
	case "uint16_t":
		return "u16", nil
	case "int16_t":
		return "i16", nil
	case "uint32_t":
		return "u32", nil
	case "int32_t":
		return "i32", nil
	case "uint64_t":
		return "u64", nil
	case "int64_t":
		return "i64", nil
	default:
		return "", fmt.Errorf("unsupported NVS encoding %q", encoding)
	}
}

func nvsCSVValue(value interface{}) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case float64:
		if typed != float64(int64(typed)) {
			return "", fmt.Errorf("unsupported non-integer NVS number")
		}
		return strconv.FormatInt(int64(typed), 10), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case uint64:
		return strconv.FormatUint(typed, 10), nil
	default:
		return "", fmt.Errorf("unsupported NVS value type %T", value)
	}
}

func verifyOfficialPCMBridgeNVSProvision(path string, deviceID string, audioWSURL string) error {
	entries, err := readStackChanNVSMinimalEntries(path)
	if err != nil {
		return err
	}
	if !nvsEntryEquals(entries, "a21", "device_id", deviceID) {
		return fmt.Errorf("provisioned NVS missing a21/device_id")
	}
	if !nvsEntryEquals(entries, "a21", "audio_ws_url", audioWSURL) {
		return fmt.Errorf("provisioned NVS missing a21/audio_ws_url")
	}
	return nil
}

func verifyOfficialXiaozhiCompatibleNVSProvision(path string, otaURL string, websocketURL string, websocketVersion int, wifiSSID string, wifiPassword string) error {
	entries, err := readStackChanNVSMinimalEntries(path)
	if err != nil {
		return err
	}
	wifiCredentialsRequested := strings.TrimSpace(wifiSSID) != "" || strings.TrimSpace(wifiPassword) != ""
	if wifiCredentialsRequested {
		if !nvsEntryEquals(entries, "wifi", "ssid", strings.TrimSpace(wifiSSID)) {
			return fmt.Errorf("provisioned NVS missing wifi/ssid")
		}
		if !nvsEntryEquals(entries, "wifi", "password", strings.TrimSpace(wifiPassword)) {
			return fmt.Errorf("provisioned NVS missing wifi/password")
		}
	}
	hasProvisionedWiFiCredentials := hasNVSEntry(entries, "wifi", "ssid") && hasNVSEntry(entries, "wifi", "password")
	if hasProvisionedWiFiCredentials && !nvsEntryEquals(entries, "app_config", "is_configed", "1") {
		return fmt.Errorf("provisioned NVS missing app_config/is_configed")
	}
	if !nvsEntryEquals(entries, "wifi", "ota_url", otaURL) {
		return fmt.Errorf("provisioned NVS missing wifi/ota_url")
	}
	if !nvsEntryEquals(entries, "websocket", "url", websocketURL) {
		return fmt.Errorf("provisioned NVS missing websocket/url")
	}
	if !nvsEntryEquals(entries, "websocket", "version", strconv.Itoa(websocketVersion)) {
		return fmt.Errorf("provisioned NVS missing websocket/version")
	}
	if hasNVSEntry(entries, "websocket", "token") {
		return fmt.Errorf("provisioned NVS must clear websocket/token")
	}
	return nil
}

func nvsEntryEquals(entries []stackChanNVSMinimalEntry, namespace string, key string, expected string) bool {
	for _, entry := range entries {
		if entry.IsEmpty || entry.State != "Written" {
			continue
		}
		if entry.Namespace == namespace && entry.Key == key {
			actual, err := nvsCSVValue(entry.Data)
			return err == nil && actual == expected
		}
	}
	return false
}

func officialIDFToolPath(idfExport string, relativePath string) string {
	return filepath.Join(filepath.Dir(filepath.Clean(idfExport)), filepath.FromSlash(relativePath))
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

func executeStackChanOfficialPCMBridgeFlash(ctx context.Context, options stackChanOfficialPCMBridgeFlashPlanOptions, report *stackChanOfficialPCMBridgeFlashPlanReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	report.FlashLogPath = filepath.Join(options.BuildDir, fmt.Sprintf("a21-official-pcm-bridge-flash-%s.log", time.Now().Format("20060102-150405")))
	script := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash @flash_args", shellSingleQuote(options.Port)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeFlashCommand(ctx, report.FlashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialXiaozhiCompatibleFlash(ctx context.Context, options stackChanOfficialXiaozhiCompatibleFlashOptions, report *stackChanOfficialXiaozhiCompatibleFlashReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	report.DryRun = false
	report.FlashAllowed = true
	report.NextRequiredConfirmation = ""
	flashLogFile := fmt.Sprintf("a21-official-xiaozhi-compatible-flash-%s.log", time.Now().Format("20060102-150405"))
	flashLogPath := filepath.Join(options.BuildDir, flashLogFile)
	report.FlashLogFile = flashLogFile
	esptoolBefore := report.EsptoolBefore
	if esptoolBefore == "" {
		var err error
		esptoolBefore, err = validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(options.EsptoolBefore)
		if err != nil {
			return err
		}
	}
	scriptLines := []string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("cd %s", shellSingleQuote(options.BuildDir)),
	}
	flashPortArg := shellSingleQuote(options.Port)
	if report.WaitROM {
		waitTimeout := report.WaitROMTimeoutSeconds
		if waitTimeout <= 0 {
			waitTimeout = stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds
		}
		flashPortArg = `"${A21_FLASH_PORT}"`
		scriptLines = append(scriptLines,
			fmt.Sprintf("A21_FLASH_PORT=%s", shellSingleQuote(options.Port)),
			`A21_WAIT_ROM_PROBE_LOG="$(mktemp -t a21-wait-rom-probe.XXXXXX)"`,
			`trap 'rm -f "${A21_WAIT_ROM_PROBE_LOG:-}"' EXIT`,
			fmt.Sprintf("A21_WAIT_ROM_DEADLINE=$((SECONDS + %d))", waitTimeout),
			fmt.Sprintf("echo %s", shellSingleQuote("Waiting for ESP32-S3 ROM download mode on "+options.Port)),
			"A21_WAIT_ROM_ATTEMPT=0",
			"while true; do",
			"  A21_WAIT_ROM_ATTEMPT=$((A21_WAIT_ROM_ATTEMPT + 1))",
			`  A21_WAIT_ROM_CANDIDATES=("${A21_FLASH_PORT}")`,
			"  for candidate in /dev/cu.usbmodem*; do",
			`    [ -e "$candidate" ] || continue`,
			`    [ "$candidate" = "$A21_FLASH_PORT" ] || A21_WAIT_ROM_CANDIDATES+=("$candidate")`,
			"  done",
			`  for candidate in "${A21_WAIT_ROM_CANDIDATES[@]}"; do`,
			`    [ -e "$candidate" ] || continue`,
			`    if python -m esptool --chip esp32s3 --port "$candidate" -b 115200 --before no_reset --after no_reset --no-stub chip_id >"${A21_WAIT_ROM_PROBE_LOG}" 2>&1; then`,
			`      A21_FLASH_PORT="$candidate"`,
			`      echo "ESP32-S3 ROM download mode detected on ${A21_FLASH_PORT}; starting guarded product app flash."`,
			"      break 2",
			"    fi",
			"  done",
			"  if (( SECONDS >= A21_WAIT_ROM_DEADLINE )); then",
			fmt.Sprintf("    echo %s >&2", shellSingleQuote("Timed out waiting for ESP32-S3 ROM download mode; hold BOOT, press/release RESET, keep holding BOOT, then retry.")),
			`    echo "ROM candidates checked: ${A21_WAIT_ROM_CANDIDATES[*]}" >&2`,
			`    echo "Last esptool probe output:" >&2`,
			`    tail -n 12 "${A21_WAIT_ROM_PROBE_LOG}" >&2 || true`,
			"    exit 1",
			"  fi",
			`  if (( A21_WAIT_ROM_ATTEMPT % 10 == 0 )); then`,
			`    echo "Still waiting for ESP32-S3 ROM download mode; candidates: ${A21_WAIT_ROM_CANDIDATES[*]}"`,
			"  fi",
			"  sleep 1",
			"done",
		)
	}
	scriptLines = append(scriptLines, fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before %s --after hard_reset write_flash @flash_args", flashPortArg, shellSingleQuote(esptoolBefore)))
	script := strings.Join(scriptLines, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleFlashCommand(ctx, flashLogPath, script); err != nil {
		return err
	}
	report.FlashExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialXiaozhiCompatibleNVS(ctx context.Context, options stackChanOfficialXiaozhiCompatibleNVSOptions, report *stackChanOfficialXiaozhiCompatibleNVSReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSToolPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_tool.py is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSGeneratorPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_partition_gen.py is missing: %w", err)
	}
	if err := os.MkdirAll(options.RunDir, 0o700); err != nil {
		return fmt.Errorf("create nvs run dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-before-"+timestamp+".bin")
	beforeJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-before-"+timestamp+".json")
	provisionCSVPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".csv")
	provisionedBinPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".bin")
	afterJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-provision-"+timestamp+".json")
	report.ReadLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-read-"+timestamp+".log")
	report.ParseLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-parse-"+timestamp+".log")
	report.GenerateLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-generate-"+timestamp+".log")
	report.VerifyLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-verify-"+timestamp+".log")
	report.WriteLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-xiaozhi-compatible-nvs-write-"+timestamp+".log")

	report.DryRun = false
	report.WriteAllowed = true
	report.NextRequiredConfirmation = ""
	report.BackupPath = backupPath
	report.ProvisionCSVPath = provisionCSVPath
	report.ProvisionedBinPath = provisionedBinPath

	readScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after no_reset read_flash %s %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			stackChanOfficialPCMBridgeNVSSizeHex,
			shellSingleQuote(backupPath)),
	}, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.ReadLogPath, readScript); err != nil {
		return fmt.Errorf("read current NVS partition: %w", err)
	}
	backupSHA, err := sha256File(backupPath)
	if err != nil {
		return fmt.Errorf("hash NVS backup: %w", err)
	}
	report.BackupSHA256 = backupSHA

	parseScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(backupPath),
			shellSingleQuote(beforeJSONPath)),
	}, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.ParseLogPath, parseScript); err != nil {
		return fmt.Errorf("parse current NVS partition: %w", err)
	}
	entries, err := readStackChanNVSMinimalEntries(beforeJSONPath)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(provisionCSVPath)
	if err != nil {
		return fmt.Errorf("create NVS provision CSV: %w", err)
	}
	summary, csvErr := writeOfficialXiaozhiCompatibleNVSCSV(csvFile, entries, options.OTAURL, options.WebSocketURL, options.WebSocketVersion, options.WiFiSSID, options.WiFiPassword)
	closeErr := csvFile.Close()
	if csvErr != nil {
		return csvErr
	}
	if closeErr != nil {
		return fmt.Errorf("close NVS provision CSV: %w", closeErr)
	}
	report.Summary = &summary

	generateScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s generate %s %s %s",
			shellSingleQuote(report.Tools.NVSGeneratorPath),
			shellSingleQuote(provisionCSVPath),
			shellSingleQuote(provisionedBinPath),
			stackChanOfficialPCMBridgeNVSSizeHex),
	}, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.GenerateLogPath, generateScript); err != nil {
		return fmt.Errorf("generate provisioned NVS partition: %w", err)
	}
	provisionedSHA, err := sha256File(provisionedBinPath)
	if err != nil {
		return fmt.Errorf("hash provisioned NVS partition: %w", err)
	}
	report.ProvisionedBinSHA256 = provisionedSHA

	verifyScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(provisionedBinPath),
			shellSingleQuote(afterJSONPath)),
	}, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.VerifyLogPath, verifyScript); err != nil {
		return fmt.Errorf("verify provisioned NVS partition: %w", err)
	}
	if err := verifyOfficialXiaozhiCompatibleNVSProvision(afterJSONPath, options.OTAURL, options.WebSocketURL, options.WebSocketVersion, options.WiFiSSID, options.WiFiPassword); err != nil {
		return err
	}

	writeScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			shellSingleQuote(provisionedBinPath)),
	}, "\n")
	if err := runStackChanOfficialXiaozhiCompatibleNVSCommand(ctx, report.WriteLogPath, writeScript); err != nil {
		return fmt.Errorf("write provisioned NVS partition: %w", err)
	}
	report.WriteExecuted = true
	report.Status = "passed"
	return nil
}

func executeStackChanOfficialPCMBridgeNVS(ctx context.Context, options stackChanOfficialPCMBridgeNVSOptions, report *stackChanOfficialPCMBridgeNVSReport) error {
	if _, err := os.Stat(options.IDFExport); err != nil {
		return fmt.Errorf("ESP-IDF export.sh is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSToolPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_tool.py is missing: %w", err)
	}
	if _, err := os.Stat(report.Tools.NVSGeneratorPath); err != nil {
		return fmt.Errorf("ESP-IDF nvs_partition_gen.py is missing: %w", err)
	}
	if err := os.MkdirAll(options.RunDir, 0o700); err != nil {
		return fmt.Errorf("create nvs run dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-before-"+timestamp+".bin")
	beforeJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-before-"+timestamp+".json")
	provisionCSVPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".csv")
	provisionedBinPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".bin")
	afterJSONPath := filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-provision-"+timestamp+".json")
	report.ReadLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-read-"+timestamp+".log")
	report.ParseLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-parse-"+timestamp+".log")
	report.GenerateLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-generate-"+timestamp+".log")
	report.VerifyLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-verify-"+timestamp+".log")
	report.WriteLogPath = filepath.Join(options.RunDir, "a21-stackchan-official-pcm-bridge-nvs-write-"+timestamp+".log")

	report.DryRun = false
	report.WriteAllowed = true
	report.NextRequiredConfirmation = ""
	report.BackupPath = backupPath
	report.ProvisionCSVPath = provisionCSVPath
	report.ProvisionedBinPath = provisionedBinPath

	readScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after no_reset read_flash %s %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			stackChanOfficialPCMBridgeNVSSizeHex,
			shellSingleQuote(backupPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.ReadLogPath, readScript); err != nil {
		return fmt.Errorf("read current NVS partition: %w", err)
	}
	backupSHA, err := sha256File(backupPath)
	if err != nil {
		return fmt.Errorf("hash NVS backup: %w", err)
	}
	report.BackupSHA256 = backupSHA

	parseScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(backupPath),
			shellSingleQuote(beforeJSONPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.ParseLogPath, parseScript); err != nil {
		return fmt.Errorf("parse current NVS partition: %w", err)
	}
	entries, err := readStackChanNVSMinimalEntries(beforeJSONPath)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(provisionCSVPath)
	if err != nil {
		return fmt.Errorf("create NVS provision CSV: %w", err)
	}
	summary, csvErr := writeOfficialPCMBridgeNVSCSV(csvFile, entries, report.DeviceID, options.AudioWSURL)
	closeErr := csvFile.Close()
	if csvErr != nil {
		return csvErr
	}
	if closeErr != nil {
		return fmt.Errorf("close NVS provision CSV: %w", closeErr)
	}
	report.Summary = &summary

	generateScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s generate %s %s %s",
			shellSingleQuote(report.Tools.NVSGeneratorPath),
			shellSingleQuote(provisionCSVPath),
			shellSingleQuote(provisionedBinPath),
			stackChanOfficialPCMBridgeNVSSizeHex),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.GenerateLogPath, generateScript); err != nil {
		return fmt.Errorf("generate provisioned NVS partition: %w", err)
	}
	provisionedSHA, err := sha256File(provisionedBinPath)
	if err != nil {
		return fmt.Errorf("hash provisioned NVS partition: %w", err)
	}
	report.ProvisionedBinSHA256 = provisionedSHA

	verifyScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python %s -d minimal -f json %s > %s",
			shellSingleQuote(report.Tools.NVSToolPath),
			shellSingleQuote(provisionedBinPath),
			shellSingleQuote(afterJSONPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.VerifyLogPath, verifyScript); err != nil {
		return fmt.Errorf("verify provisioned NVS partition: %w", err)
	}
	if err := verifyOfficialPCMBridgeNVSProvision(afterJSONPath, report.DeviceID, options.AudioWSURL); err != nil {
		return err
	}

	writeScript := strings.Join([]string{
		"set -euo pipefail",
		fmt.Sprintf("source %s >/dev/null", shellSingleQuote(options.IDFExport)),
		fmt.Sprintf("python -m esptool --chip esp32s3 --port %s -b 460800 --before default_reset --after hard_reset write_flash %s %s",
			shellSingleQuote(options.Port),
			stackChanOfficialPCMBridgeNVSOffset,
			shellSingleQuote(provisionedBinPath)),
	}, "\n")
	if err := runStackChanOfficialPCMBridgeNVSCommand(ctx, report.WriteLogPath, writeScript); err != nil {
		return fmt.Errorf("write provisioned NVS partition: %w", err)
	}
	report.WriteExecuted = true
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

func validateStackChanOfficialXiaozhiCompatibleEsptoolBefore(mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return "default_reset", nil
	}
	switch mode {
	case "default_reset", "usb_reset", "no_reset":
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported esptool before mode %q; want default_reset, usb_reset, or no_reset", mode)
	}
}

func stackChanOfficialXiaozhiCompatibleWaitROMTimeoutSeconds(options stackChanOfficialXiaozhiCompatibleFlashOptions) int {
	if !options.WaitROM {
		return 0
	}
	if options.WaitROMTimeoutSeconds > 0 {
		return options.WaitROMTimeoutSeconds
	}
	return stackChanOfficialXiaozhiCompatibleDefaultWaitROMTimeoutSeconds
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

func applyStackChanOfficialCandidateContract(report *stackChanOfficialBaselineReport, sourceRoot string, overlays []string) {
	if report == nil {
		return
	}
	mainCPP := readGitTrackedOrFile(sourceRoot, "firmware/main/main.cpp")
	overlayText := readOfficialOverlayText(overlays)
	candidate, role := classifyStackChanOfficialCandidate(overlays, mainCPP)

	avatarPreserved := strings.Contains(mainCPP, "AppAvatar") &&
		strings.Contains(mainCPP, "AppAiAgent") &&
		strings.Contains(mainCPP, "GetMooncake().installApp")
	xiaozhiStartPreserved := strings.Contains(mainCPP, "GetHAL().startXiaozhi()") &&
		strings.Contains(mainCPP, "isXiaozhiStartRequested")
	minimalBridgeScreen := strings.Contains(mainCPP, "A21 BRIDGE") ||
		strings.Contains(mainCPP, "renderStatus()") ||
		strings.Contains(overlayText, "A21 BRIDGE") ||
		strings.Contains(overlayText, "renderStatus()")

	if strings.Contains(overlayText, "-    GetMooncake().installApp(std::make_unique<AppAvatar>())") ||
		strings.Contains(overlayText, "-    GetMooncake().installApp(std::make_unique<AppAiAgent>())") {
		avatarPreserved = false
	}
	if strings.Contains(overlayText, "-    GetHAL().startXiaozhi()") {
		xiaozhiStartPreserved = false
	}
	if strings.Contains(candidate, "pcm-bridge") {
		role = "diagnostic_m3_prep"
		minimalBridgeScreen = true
		avatarPreserved = false
		xiaozhiStartPreserved = false
	}
	if strings.Contains(candidate, "audio-smoke") {
		role = "diagnostic_audio_smoke"
		avatarPreserved = false
		xiaozhiStartPreserved = false
	}

	report.FirmwareCandidate = candidate
	report.BuildLaneRole = role
	report.OfficialAvatarActionPreserved = avatarPreserved
	report.OfficialXiaozhiStartPreserved = xiaozhiStartPreserved
	report.MinimalBridgeScreen = minimalBridgeScreen
}

func classifyStackChanOfficialCandidate(overlays []string, mainCPP string) (string, string) {
	for _, overlay := range overlays {
		base := filepath.Base(overlay)
		switch {
		case strings.Contains(base, "xiaozhi-compatible"):
			return stackChanOfficialXiaozhiCompatibleFirmwareCandidate, "product_candidate"
		case strings.Contains(base, "pcm-bridge"):
			return "a21-stackchan-official-pcm-bridge", "diagnostic_m3_prep"
		case strings.Contains(base, "audio-smoke"):
			return "a21-stackchan-official-audio-smoke", "diagnostic_audio_smoke"
		}
	}
	switch {
	case strings.Contains(mainCPP, "A21 BRIDGE"):
		return "a21-stackchan-official-pcm-bridge", "diagnostic_m3_prep"
	case strings.Contains(mainCPP, "a21-stackchan-official-xiaozhi-compatible"):
		return stackChanOfficialXiaozhiCompatibleFirmwareCandidate, "product_candidate"
	case strings.Contains(mainCPP, "a21-stackchan-official-audio-smoke"):
		return "a21-stackchan-official-audio-smoke", "diagnostic_audio_smoke"
	default:
		return "stack-chan-official-baseline", "official_reference"
	}
}

func readOfficialOverlayText(overlays []string) string {
	var builder strings.Builder
	for _, overlay := range overlays {
		if containsLegacyIdentityPathToken(overlay) {
			continue
		}
		data, err := os.ReadFile(overlay)
		if err != nil {
			continue
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	return builder.String()
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

	buildLog := filepath.Join(options.BuildDir, "a21-official-build.log")
	_ = os.MkdirAll(options.BuildDir, 0o755)

	if options.DepCache != "" {
		report.Build.DependencyCacheUsed = true
		report.Build.DependencyCacheRoot = options.DepCache
		if err := hydrateStackChanOfficialDependenciesFromCache(options.WorkDir, options.DepCache); err != nil {
			report.fail("dependency_cache_failed", err.Error())
			return
		}
	} else {
		fetchLog := filepath.Join(options.BuildDir, "a21-official-fetch.log")
		report.Build.FetchExecuted = true
		report.Build.FetchLogPath = fetchLog
		if err := runLoggedCommand(ctx, filepath.Join(options.WorkDir, "firmware"), fetchLog, "python3", "./fetch_repos.py"); err != nil {
			report.fail("fetch_repos_failed", "official fetch_repos.py failed")
			return
		}
	}
	for index, overlay := range options.Overlays {
		cleanOverlay := filepath.Clean(overlay)
		if containsLegacyIdentityPathToken(cleanOverlay) {
			report.fail("overlay_path_legacy_identity", "overlay path contains forbidden legacy identity")
			return
		}
		if err := runLoggedCommand(ctx, options.WorkDir, filepath.Join(options.BuildDir, "a21-official-overlay.log"), "git", "apply", "--recount", cleanOverlay); err != nil {
			report.fail("overlay_apply_failed", "failed to apply A21 official baseline overlay")
			return
		}
		if index < len(report.Overlays) {
			report.Overlays[index].Applied = true
		}
	}

	report.Evidence = inspectStackChanOfficialEvidence(options.WorkDir)
	applyStackChanOfficialCandidateContract(report, options.WorkDir, options.Overlays)
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
	if err := validateOfficialAssetsPartitionCapacity(options.BuildDir); err != nil {
		report.fail("assets_partition_capacity_failed", err.Error())
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

type stackChanOfficialRepoConfig struct {
	URL            string `json:"url"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	WithSubmodules bool   `json:"with_submodules"`
	Patch          string `json:"patch"`
}

func hydrateStackChanOfficialDependenciesFromCache(workDir string, cacheRoot string) error {
	reposPath := filepath.Join(workDir, "firmware", "repos.json")
	data, err := os.ReadFile(reposPath)
	if err != nil {
		return fmt.Errorf("read official repos.json: %w", err)
	}
	var repos []stackChanOfficialRepoConfig
	if err := json.Unmarshal(data, &repos); err != nil {
		return fmt.Errorf("parse official repos.json: %w", err)
	}
	if len(repos) == 0 {
		return fmt.Errorf("official repos.json has no dependencies")
	}
	for _, repo := range repos {
		if repo.Path == "" {
			return fmt.Errorf("official dependency path is required")
		}
		if filepath.IsAbs(repo.Path) || strings.Contains(repo.Path, "..") || containsLegacyIdentityPathToken(repo.Path) {
			return fmt.Errorf("official dependency path %q is not allowed", repo.Path)
		}
		src, err := resolveStackChanOfficialCacheRepo(cacheRoot, repo.Path)
		if err != nil {
			return err
		}
		if repo.Branch != "" {
			if err := validateStackChanOfficialCacheRef(src, repo.Branch); err != nil {
				return fmt.Errorf("validate dependency cache %s: %w", repo.Path, err)
			}
		}
		dst := filepath.Join(workDir, "firmware", filepath.FromSlash(repo.Path))
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("clean dependency destination %s: %w", repo.Path, err)
		}
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return fmt.Errorf("create dependency destination %s: %w", repo.Path, err)
		}
		if err := exportGitHEAD(context.Background(), src, dst); err != nil {
			return fmt.Errorf("export dependency cache %s: %w", repo.Path, err)
		}
		if repo.Patch != "" {
			patchPath := repo.Patch
			if !filepath.IsAbs(patchPath) {
				patchPath = filepath.Join(workDir, "firmware", filepath.FromSlash(repo.Patch))
			}
			if err := applyStackChanOfficialDependencyPatch(dst, patchPath); err != nil {
				return fmt.Errorf("apply dependency patch %s: %w", repo.Path, err)
			}
		}
	}
	return nil
}

func resolveStackChanOfficialCacheRepo(cacheRoot string, repoPath string) (string, error) {
	if cacheRoot == "" {
		return "", fmt.Errorf("dependency cache root is required")
	}
	candidates := []string{
		filepath.Join(cacheRoot, "firmware", filepath.FromSlash(repoPath)),
		filepath.Join(cacheRoot, filepath.FromSlash(repoPath)),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, ".git")); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("dependency cache missing repo %s", repoPath)
}

func validateStackChanOfficialCacheRef(repoDir string, ref string) error {
	wanted, err := gitRevParse(repoDir, ref+"^{commit}")
	if err != nil {
		return fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	head, err := gitRevParse(repoDir, "HEAD")
	if err != nil {
		return fmt.Errorf("resolve HEAD: %w", err)
	}
	if wanted != head {
		return fmt.Errorf("HEAD %s does not match required ref %q (%s)", head, ref, wanted)
	}
	return nil
}

func gitRevParse(repoDir string, rev string) (string, error) {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", rev).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func applyStackChanOfficialDependencyPatch(repoDir string, patchPath string) error {
	if patchPath == "" {
		return nil
	}
	cleanPatch := filepath.Clean(patchPath)
	if containsLegacyIdentityPathToken(cleanPatch) {
		return fmt.Errorf("dependency patch path contains forbidden legacy identity")
	}
	if _, err := os.Stat(cleanPatch); err != nil {
		return fmt.Errorf("dependency patch missing: %w", err)
	}
	if err := runCommandInDir(repoDir, "git", "init"); err != nil {
		return fmt.Errorf("init temporary dependency repo: %w", err)
	}
	if err := runCommandInDir(repoDir, "git", "apply", "--check", cleanPatch); err != nil {
		_ = os.RemoveAll(filepath.Join(repoDir, ".git"))
		return fmt.Errorf("dependency patch check failed: %w", err)
	}
	if err := runCommandInDir(repoDir, "git", "apply", cleanPatch); err != nil {
		_ = os.RemoveAll(filepath.Join(repoDir, ".git"))
		return fmt.Errorf("dependency patch apply failed: %w", err)
	}
	return os.RemoveAll(filepath.Join(repoDir, ".git"))
}

func runCommandInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
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
		{name: "app", path: filepath.Join(buildDir, stackChanOfficialXiaozhiCompatibleAppBinary)},
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
	if containsLegacyIdentityPathToken(clean) {
		return fmt.Errorf("scratch dir contains forbidden legacy identity")
	}
	if !strings.Contains(filepath.Base(lower), "a21-stackchan-official") {
		return fmt.Errorf("scratch dir basename must contain a21-stackchan-official")
	}
	return nil
}

func validateA21OfficialRunDir(path string) error {
	clean := filepath.Clean(path)
	lower := strings.ToLower(clean)
	if clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("run dir must be an explicit A21 work directory")
	}
	if containsLegacyIdentityPathToken(clean) {
		return fmt.Errorf("run dir contains forbidden legacy identity")
	}
	if !strings.Contains(lower, "a21") {
		return fmt.Errorf("run dir must contain a21")
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

func writeStackChanOfficialXiaozhiCompatibleFlashReport(outputDir string, report stackChanOfficialXiaozhiCompatibleFlashReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-xiaozhi-compatible-flash-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONStackChanOfficialXiaozhiCompatibleFlash(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanOfficialXiaozhiCompatibleNVSReport(outputDir string, report stackChanOfficialXiaozhiCompatibleNVSReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-xiaozhi-compatible-nvs-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONStackChanOfficialXiaozhiCompatibleNVS(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeStackChanOfficialPCMBridgeNVSReport(outputDir string, report stackChanOfficialPCMBridgeNVSReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-pcm-bridge-nvs-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := writeJSONStackChanOfficialPCMBridgeNVS(file, report); err != nil {
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

func writeJSONStackChanOfficialXiaozhiCompatibleFlash(writer io.Writer, report stackChanOfficialXiaozhiCompatibleFlashReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanOfficialXiaozhiCompatibleNVS(writer io.Writer, report stackChanOfficialXiaozhiCompatibleNVSReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeJSONStackChanOfficialPCMBridgeNVS(writer io.Writer, report stackChanOfficialPCMBridgeNVSReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
