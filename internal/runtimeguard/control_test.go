package runtimeguard

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeControlGitRunner struct {
	branch string
	commit string
	root   string
	status string
}

func (r fakeControlGitRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if name != "git" {
		return "", errors.New("unexpected command")
	}
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "rev-parse --abbrev-ref HEAD"):
		return r.branch + "\n", nil
	case strings.Contains(joined, "rev-parse --short=12 HEAD"):
		return r.commit + "\n", nil
	case strings.Contains(joined, "rev-parse --show-toplevel"):
		return r.root + "\n", nil
	case strings.Contains(joined, "status --porcelain --untracked-files=all"):
		return r.status, nil
	default:
		return "", errors.New("unexpected git args: " + joined)
	}
}

func TestControlGuardBlocksT7OutsideHardwareWindowBranch(t *testing.T) {
	report := EvaluateControlGuard(context.Background(), ControlGuardInput{
		Command: "stackchan-official-pcm-bridge-nvs-execute",
		CWD:     "/work/a21",
		Runner: fakeControlGitRunner{
			branch: "codex/a21-project-control",
			commit: "abc123def456",
			root:   "/work/a21",
		},
	})

	if report.Result.OK {
		t.Fatalf("guard unexpectedly passed: %+v", report)
	}
	if report.Tier != "T7" {
		t.Fatalf("tier = %q, want T7", report.Tier)
	}
	if !hasFindingCode(report.Result.Findings, "control_hardware_window_branch_required") {
		t.Fatalf("missing branch finding: %+v", report.Result.Findings)
	}
}

func TestControlGuardAllowsT7HardwareWindowBranch(t *testing.T) {
	report := EvaluateControlGuard(context.Background(), ControlGuardInput{
		Command: "stackchan-official-pcm-bridge-nvs-execute",
		CWD:     "/work/a21",
		Runner: fakeControlGitRunner{
			branch: "codex/a21-hardware-window-20260531-pcm-bridge",
			commit: "abc123def456",
			root:   "/work/a21",
		},
	})

	if !report.Result.OK {
		t.Fatalf("guard failed: %+v", report.Result.Findings)
	}
	if report.Git.DetachedHead {
		t.Fatalf("detached head should be false: %+v", report.Git)
	}
}

func TestControlGuardBlocksDetachedT7Worktree(t *testing.T) {
	report := EvaluateControlGuard(context.Background(), ControlGuardInput{
		Command: "firmware-bootstrap-flash-execute",
		CWD:     "/Users/jiyurun/.codex/worktrees/abcd/New project",
		Runner: fakeControlGitRunner{
			branch: "HEAD",
			commit: "abc123def456",
			root:   "/Users/jiyurun/.codex/worktrees/abcd/New project",
		},
	})

	if report.Result.OK {
		t.Fatalf("guard unexpectedly passed: %+v", report)
	}
	for _, want := range []string{"control_detached_head", "control_background_worktree"} {
		if !hasFindingCode(report.Result.Findings, want) {
			t.Fatalf("missing %s finding: %+v", want, report.Result.Findings)
		}
	}
}

func TestControlGuardBlocksT8UntilADR(t *testing.T) {
	report := EvaluateControlGuard(context.Background(), ControlGuardInput{
		Command: "stackchan-official-pcm-bridge-flash-execute",
		CWD:     "/work/a21",
		Runner: fakeControlGitRunner{
			branch: "codex/a21-hardware-window-20260531-pcm-bridge",
			commit: "abc123def456",
			root:   "/work/a21",
		},
	})

	if report.Result.OK {
		t.Fatalf("guard unexpectedly passed: %+v", report)
	}
	if report.Tier != "T8" {
		t.Fatalf("tier = %q, want T8", report.Tier)
	}
	if !hasFindingCode(report.Result.Findings, "control_t8_blocked_until_adr") {
		t.Fatalf("missing T8 finding: %+v", report.Result.Findings)
	}
}

func TestControlGuardClassifiesExecuteFlagCommands(t *testing.T) {
	for _, command := range []string{
		"provider-smoke --execute",
		"v21-adapter-smoke --execute",
		"local-voice-loopback --execute-text-provider",
		"stackchan-fast-companion-turn --execute-text-provider",
	} {
		spec, ok := LookupControlCommandSpec(command)
		if !ok {
			t.Fatalf("missing spec for %q", command)
		}
		if spec.Tier != "T4" {
			t.Fatalf("%q tier = %q, want T4", command, spec.Tier)
		}
	}
}

func hasFindingCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
