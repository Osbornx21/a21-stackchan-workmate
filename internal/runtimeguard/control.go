package runtimeguard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ControlGuardSchema = "a21.control_guard.v1"

type ControlCommandSpec struct {
	Command                    string `json:"command"`
	Tier                       string `json:"tier"`
	Label                      string `json:"label"`
	RequiresHardwareWindow     bool   `json:"requires_hardware_window"`
	BlockedUntilADR            bool   `json:"blocked_until_adr"`
	RequiresCleanWorktree      bool   `json:"requires_clean_worktree"`
	RejectsBackgroundWorktree  bool   `json:"rejects_background_worktree"`
	RejectsDetachedHead        bool   `json:"rejects_detached_head"`
	AllowsProviderNetwork      bool   `json:"allows_provider_network"`
	AllowsExplicitV21Execution bool   `json:"allows_explicit_v21_execution"`
}

type ControlGuardInput struct {
	Config  Config
	Command string
	Tier    string
	CWD     string
	Env     []string
	Runner  CommandRunner
}

type ControlGuardReport struct {
	SchemaVersion string             `json:"schema_version"`
	GeneratedAtMS int64              `json:"generated_at_ms"`
	Command       string             `json:"command"`
	Tier          string             `json:"tier"`
	Spec          ControlCommandSpec `json:"spec"`
	Git           ControlGitState    `json:"git"`
	Result        Result             `json:"result"`
}

type ControlGitState struct {
	ProjectRoot    string `json:"project_root,omitempty"`
	WorktreePath   string `json:"worktree_path,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Commit         string `json:"commit,omitempty"`
	DetachedHead   bool   `json:"detached_head"`
	DirtyFileCount int    `json:"dirty_file_count"`
}

func LookupControlCommandSpec(command string) (ControlCommandSpec, bool) {
	normalized := normalizeControlCommand(command)
	if spec, ok := controlCommandSpecs()[normalized]; ok {
		return spec, true
	}
	if commandHasFlag(normalized, "provider-smoke", "--execute") {
		return controlCommandSpecs()["provider-smoke --execute"], true
	}
	if commandHasFlag(normalized, "v21-adapter-smoke", "--execute") {
		return controlCommandSpecs()["v21-adapter-smoke --execute"], true
	}
	if commandHasFlag(normalized, "local-voice-loopback", "--execute-text-provider") {
		return controlCommandSpecs()["local-voice-loopback --execute-text-provider"], true
	}
	if commandHasFlag(normalized, "stackchan-fast-companion-turn", "--execute-text-provider") {
		return controlCommandSpecs()["stackchan-fast-companion-turn --execute-text-provider"], true
	}
	if commandHasFlag(normalized, "provider-realtime-fixture", "--execute") {
		return controlCommandSpecs()["provider-realtime-fixture --execute"], true
	}
	for _, command := range []string{
		"stackchan-official-audio-smoke-flash",
		"stackchan-official-pcm-bridge-flash",
		"stackchan-official-pcm-bridge-nvs",
		"firmware-bootstrap-flash",
		"firmware-mic-probe-flash",
		"firmware-imu-probe-flash",
		"firmware-sensor-probe-flash",
		"xiaozhi-firmware-flash",
	} {
		if commandHasFlag(normalized, command, "--execute") {
			return controlCommandSpecs()[command+" --execute"], true
		}
	}
	if strings.Contains(normalized, "pio run -t upload") ||
		strings.Contains(normalized, "idf.py flash") ||
		strings.Contains(normalized, "esptool") && strings.Contains(normalized, "write_flash") {
		return controlCommandSpecs()["raw-firmware-write"], true
	}
	return ControlCommandSpec{}, false
}

func EvaluateControlGuard(ctx context.Context, input ControlGuardInput) ControlGuardReport {
	command := normalizeControlCommand(input.Command)
	spec, known := LookupControlCommandSpec(command)
	tier := strings.TrimSpace(input.Tier)
	if known {
		tier = spec.Tier
	} else if tier != "" {
		spec = ControlCommandSpec{
			Command: command,
			Tier:    tier,
			Label:   "caller declared command",
		}
	}
	findings := make([]Finding, 0)
	if command == "" {
		findings = append(findings, Finding{
			Code:     "control_command_required",
			Severity: SeverityBlock,
			Message:  "A21 control guard requires a command name",
		})
	}
	if tier == "" {
		findings = append(findings, Finding{
			Code:     "control_tier_required",
			Severity: SeverityBlock,
			Message:  "A21 control guard requires a tool tier or known command",
		})
	}

	gitState, gitFindings := detectControlGitState(ctx, input)
	findings = append(findings, gitFindings...)

	if spec.BlockedUntilADR || tier == "T8" {
		findings = append(findings, Finding{
			Code:     "control_t8_blocked_until_adr",
			Severity: SeverityBlock,
			Message:  "A21 T8 command is blocked until an ADR and reviewed execute guard approve it",
			Detail:   command,
		})
	}

	if tier == "T7" || spec.RequiresHardwareWindow {
		if spec.RejectsDetachedHead || spec.RequiresHardwareWindow {
			if gitState.DetachedHead {
				findings = append(findings, Finding{
					Code:     "control_detached_head",
					Severity: SeverityBlock,
					Message:  "A21 hardware writes must not run from detached HEAD",
					Detail:   gitState.Commit,
				})
			}
		}
		if spec.RequiresHardwareWindow && !strings.HasPrefix(gitState.Branch, "codex/a21-hardware-window-") {
			findings = append(findings, Finding{
				Code:     "control_hardware_window_branch_required",
				Severity: SeverityBlock,
				Message:  "A21 hardware writes require a codex/a21-hardware-window-* branch",
				Detail:   gitState.Branch,
			})
		}
		if spec.RequiresCleanWorktree && gitState.DirtyFileCount > 0 {
			findings = append(findings, Finding{
				Code:     "control_dirty_worktree",
				Severity: SeverityBlock,
				Message:  "A21 hardware writes require a clean worktree",
				Detail:   fmt.Sprintf("%d dirty files", gitState.DirtyFileCount),
			})
		}
		if spec.RejectsBackgroundWorktree && (isCodexBackgroundWorktree(gitState.WorktreePath) || isCodexBackgroundWorktree(input.CWD)) {
			findings = append(findings, Finding{
				Code:     "control_background_worktree",
				Severity: SeverityBlock,
				Message:  "A21 hardware writes must run in the foreground hardware window, not a background Codex worktree",
				Detail:   gitState.WorktreePath,
			})
		}
	}

	return ControlGuardReport{
		SchemaVersion: ControlGuardSchema,
		GeneratedAtMS: time.Now().UnixMilli(),
		Command:       command,
		Tier:          tier,
		Spec:          spec,
		Git:           gitState,
		Result:        NewResult(findings),
	}
}

func controlCommandSpecs() map[string]ControlCommandSpec {
	t7HardwareWrite := func(command string, label string) ControlCommandSpec {
		return ControlCommandSpec{
			Command:                   command,
			Tier:                      "T7",
			Label:                     label,
			RequiresHardwareWindow:    true,
			RequiresCleanWorktree:     true,
			RejectsBackgroundWorktree: true,
			RejectsDetachedHead:       true,
		}
	}
	return map[string]ControlCommandSpec{
		"stackchan-official-audio-smoke-flash --execute": t7HardwareWrite("stackchan-official-audio-smoke-flash --execute", "official StackChan app flash execute"),
		"stackchan-official-audio-smoke-flash-execute":   t7HardwareWrite("stackchan-official-audio-smoke-flash-execute", "official StackChan app flash execute"),
		"stackchan-official-pcm-bridge-nvs --execute":    t7HardwareWrite("stackchan-official-pcm-bridge-nvs --execute", "official StackChan PCM bridge NVS write"),
		"stackchan-official-pcm-bridge-nvs-execute":      t7HardwareWrite("stackchan-official-pcm-bridge-nvs-execute", "official StackChan PCM bridge NVS write"),
		"firmware-bootstrap-flash --execute":             t7HardwareWrite("firmware-bootstrap-flash --execute", "release firmware flash execute"),
		"firmware-bootstrap-flash-execute":               t7HardwareWrite("firmware-bootstrap-flash-execute", "release firmware flash execute"),
		"firmware-mic-probe-flash --execute":             t7HardwareWrite("firmware-mic-probe-flash --execute", "mic probe firmware flash execute"),
		"firmware-mic-probe-flash-execute":               t7HardwareWrite("firmware-mic-probe-flash-execute", "mic probe firmware flash execute"),
		"firmware-imu-probe-flash --execute":             t7HardwareWrite("firmware-imu-probe-flash --execute", "IMU probe firmware flash execute"),
		"firmware-imu-probe-flash-execute":               t7HardwareWrite("firmware-imu-probe-flash-execute", "IMU probe firmware flash execute"),
		"firmware-sensor-probe-flash --execute":          t7HardwareWrite("firmware-sensor-probe-flash --execute", "sensor probe firmware flash execute"),
		"firmware-sensor-probe-flash-execute":            t7HardwareWrite("firmware-sensor-probe-flash-execute", "sensor probe firmware flash execute"),
		"stackchan-official-pcm-bridge-flash --execute":  t7HardwareWrite("stackchan-official-pcm-bridge-flash --execute", "official StackChan PCM bridge app flash execute"),
		"stackchan-official-pcm-bridge-flash-execute":    t7HardwareWrite("stackchan-official-pcm-bridge-flash-execute", "official StackChan PCM bridge app flash execute"),
		"xiaozhi-firmware-flash --execute":               t7HardwareWrite("xiaozhi-firmware-flash --execute", "xiaozhi firmware flash execute"),
		"xiaozhi-firmware-flash-execute":                 t7HardwareWrite("xiaozhi-firmware-flash-execute", "xiaozhi firmware flash execute"),
		"raw-firmware-write": {
			Command:         "raw-firmware-write",
			Tier:            "T8",
			Label:           "raw firmware write command",
			BlockedUntilADR: true,
		},
		"provider-smoke --execute": {
			Command:               "provider-smoke --execute",
			Tier:                  "T4",
			Label:                 "provider smoke with external execution",
			AllowsProviderNetwork: true,
		},
		"v21-adapter-smoke --execute": {
			Command:                    "v21-adapter-smoke --execute",
			Tier:                       "T4",
			Label:                      "V21 adapter smoke with explicit execution",
			AllowsExplicitV21Execution: true,
		},
		"local-voice-loopback --execute-text-provider": {
			Command:               "local-voice-loopback --execute-text-provider",
			Tier:                  "T4",
			Label:                 "local voice loopback with paid text-provider execution",
			AllowsProviderNetwork: true,
		},
		"stackchan-fast-companion-turn --execute-text-provider": {
			Command:               "stackchan-fast-companion-turn --execute-text-provider",
			Tier:                  "T4",
			Label:                 "StackChan fast companion turn with paid text-provider execution",
			AllowsProviderNetwork: true,
		},
		"provider-realtime-fixture --execute": {
			Command: "provider-realtime-fixture --execute",
			Tier:    "T2",
			Label:   "offline realtime provider fixture execute",
		},
	}
}

func detectControlGitState(ctx context.Context, input ControlGuardInput) (ControlGitState, []Finding) {
	cwd := strings.TrimSpace(input.CWD)
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ControlGitState{}, []Finding{{
				Code:     "control_cwd_unavailable",
				Severity: SeverityBlock,
				Message:  "A21 control guard could not read current working directory",
				Detail:   err.Error(),
			}}
		}
	}
	runner := input.Runner
	if runner == nil {
		runner = OSRunner{}
	}
	runGit := func(args ...string) (string, error) {
		allArgs := append([]string{"-C", cwd}, args...)
		out, err := runner.Run(ctx, "git", allArgs...)
		return strings.TrimSpace(out), err
	}
	root, err := runGit("rev-parse", "--show-toplevel")
	if err != nil {
		return ControlGitState{WorktreePath: filepath.Clean(cwd)}, []Finding{{
			Code:     "control_git_root_unavailable",
			Severity: SeverityBlock,
			Message:  "A21 control guard requires a git worktree",
			Detail:   err.Error(),
		}}
	}
	branch, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ControlGitState{ProjectRoot: root, WorktreePath: filepath.Clean(cwd)}, []Finding{{
			Code:     "control_git_branch_unavailable",
			Severity: SeverityBlock,
			Message:  "A21 control guard could not read git branch",
			Detail:   err.Error(),
		}}
	}
	commit, err := runGit("rev-parse", "--short=12", "HEAD")
	if err != nil {
		return ControlGitState{ProjectRoot: root, WorktreePath: filepath.Clean(cwd), Branch: branch}, []Finding{{
			Code:     "control_git_commit_unavailable",
			Severity: SeverityBlock,
			Message:  "A21 control guard could not read git commit",
			Detail:   err.Error(),
		}}
	}
	status, err := runGit("status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return ControlGitState{ProjectRoot: root, WorktreePath: filepath.Clean(cwd), Branch: branch, Commit: commit, DetachedHead: branch == "HEAD"}, []Finding{{
			Code:     "control_git_status_unavailable",
			Severity: SeverityBlock,
			Message:  "A21 control guard could not read git status",
			Detail:   err.Error(),
		}}
	}
	return ControlGitState{
		ProjectRoot:    filepath.Clean(root),
		WorktreePath:   filepath.Clean(cwd),
		Branch:         branch,
		Commit:         commit,
		DetachedHead:   branch == "HEAD",
		DirtyFileCount: countControlStatusLines(status),
	}, nil
}

func normalizeControlCommand(command string) string {
	command = strings.TrimSpace(command)
	for _, prefix := range []string{"go run ./cmd/a21 ", "go run cmd/a21 ", "a21 "} {
		command = strings.TrimPrefix(command, prefix)
	}
	return strings.Join(strings.Fields(command), " ")
}

func commandHasFlag(command string, name string, flag string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 || fields[0] != name {
		return false
	}
	for _, field := range fields[1:] {
		if field == flag {
			return true
		}
	}
	return false
}

func countControlStatusLines(status string) int {
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func isCodexBackgroundWorktree(path string) bool {
	clean := filepath.Clean(path)
	sep := string(filepath.Separator)
	return strings.Contains(clean, sep+".codex"+sep+"worktrees"+sep)
}
