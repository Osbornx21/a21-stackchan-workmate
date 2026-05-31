package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"a21.local/a21/internal/runtimeguard"
)

const promotionReadinessSchema = "a21.promotion_readiness.v1"

type promotionReadinessTopicSpec struct {
	Branch string
	Commit string
}

type promotionReadinessTopic struct {
	Branch   string `json:"branch"`
	Commit   string `json:"commit"`
	Ancestor bool   `json:"ancestor"`
}

type promotionReadinessReport struct {
	SchemaVersion          string                    `json:"schema_version"`
	GeneratedAtMS          int64                     `json:"generated_at_ms"`
	WorktreePath           string                    `json:"worktree_path"`
	ProjectRoot            string                    `json:"project_root"`
	Branch                 string                    `json:"branch"`
	Commit                 string                    `json:"commit"`
	DirtyFileCount         int                       `json:"dirty_file_count"`
	RemoteNames            []string                  `json:"remote_names"`
	LocalMainExists        bool                      `json:"local_main_exists"`
	LocalMasterExists      bool                      `json:"local_master_exists"`
	TargetRemote           string                    `json:"target_remote,omitempty"`
	TargetBranch           string                    `json:"target_branch,omitempty"`
	Topics                 []promotionReadinessTopic `json:"topics"`
	ReviewReady            bool                      `json:"review_ready"`
	ExternalPromotionReady bool                      `json:"external_promotion_ready"`
	Result                 runtimeguard.Result       `json:"result"`
}

var promotionReadinessTopics = []promotionReadinessTopicSpec{
	{Branch: "codex/a21-docs-pcm-bridge-flash-adr", Commit: "a031f3d"},
	{Branch: "codex/a21-provider-spine-deepseek-textstream", Commit: "6b7fdf0"},
	{Branch: "codex/a21-mainline-professional-v21-contract", Commit: "fc61793"},
	{Branch: "codex/a21-mainline-stackchan-hardware-diagnostic", Commit: "1e38804"},
}

func runPromotionReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	var targetRemote string
	var targetBranch string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 promotion-readiness [--target-remote <name>] [--target-branch <branch>]")
			return 0
		case "--target-remote":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--target-remote requires a value")
				return 2
			}
			i++
			targetRemote = strings.TrimSpace(args[i])
		case "--target-branch":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--target-branch requires a value")
				return 2
			}
			i++
			targetBranch = strings.TrimSpace(args[i])
		default:
			fmt.Fprintf(stderr, "unknown promotion-readiness option %q\n", args[i])
			return 2
		}
	}

	report, err := buildPromotionReadinessReport(targetRemote, targetBranch)
	if err != nil {
		fmt.Fprintf(stderr, "promotion-readiness failed: %v\n", err)
		return 1
	}
	if err := writeJSONPromotionReadiness(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode promotion readiness report: %v\n", err)
		return 1
	}
	if !report.ExternalPromotionReady {
		return 1
	}
	return 0
}

func buildPromotionReadinessReport(targetRemote string, targetBranch string) (promotionReadinessReport, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return promotionReadinessReport{}, fmt.Errorf("get working directory: %w", err)
	}
	worktreePath, err := filepath.Abs(cwd)
	if err != nil {
		worktreePath = filepath.Clean(cwd)
	}
	projectRoot := findProjectRoot(worktreePath)

	branch, err := runPromotionGit(projectRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return promotionReadinessReport{}, fmt.Errorf("read branch: %w", err)
	}
	commit, err := runPromotionGit(projectRoot, "rev-parse", "--short=12", "HEAD")
	if err != nil {
		return promotionReadinessReport{}, fmt.Errorf("read commit: %w", err)
	}
	status, err := runPromotionGit(projectRoot, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return promotionReadinessReport{}, fmt.Errorf("read git status: %w", err)
	}
	remoteOutput, err := runPromotionGit(projectRoot, "remote")
	if err != nil {
		return promotionReadinessReport{}, fmt.Errorf("read git remotes: %w", err)
	}

	remoteNames := splitNonEmptyLines(remoteOutput)
	sort.Strings(remoteNames)
	localMainExists := promotionGitExitOK(projectRoot, "show-ref", "--verify", "--quiet", "refs/heads/main")
	localMasterExists := promotionGitExitOK(projectRoot, "show-ref", "--verify", "--quiet", "refs/heads/master")

	topics := make([]promotionReadinessTopic, 0, len(promotionReadinessTopics))
	findings := make([]runtimeguard.Finding, 0)
	reviewReady := true

	detachedHead := branch == "HEAD" || strings.TrimSpace(branch) == ""
	if detachedHead {
		reviewReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_detached_head",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 promotion candidate must not be detached HEAD",
			Detail:   commit,
		})
	}
	if !strings.HasPrefix(branch, "codex/a21-integration-") {
		reviewReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_branch_not_integration",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 promotion candidate must run from a codex/a21-integration-* branch",
			Detail:   branch,
		})
	}

	dirtyFileCount := len(splitNonEmptyLines(status))
	if dirtyFileCount > 0 {
		reviewReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_dirty_worktree",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 promotion candidate requires a clean worktree",
			Detail:   fmt.Sprintf("%d dirty files", dirtyFileCount),
		})
	}

	for _, spec := range promotionReadinessTopics {
		ancestor := promotionGitExitOK(projectRoot, "merge-base", "--is-ancestor", spec.Commit, "HEAD")
		topics = append(topics, promotionReadinessTopic{
			Branch:   spec.Branch,
			Commit:   spec.Commit,
			Ancestor: ancestor,
		})
		if !ancestor {
			reviewReady = false
			findings = append(findings, runtimeguard.Finding{
				Code:     "promotion_topic_not_ancestor",
				Severity: runtimeguard.SeverityBlock,
				Message:  "A21 integration candidate is missing an accepted slice commit",
				Detail:   spec.Branch + " " + spec.Commit,
			})
		}
	}

	externalPromotionReady := reviewReady
	if len(remoteNames) == 0 {
		externalPromotionReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_remote_missing",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 external promotion requires a configured git remote",
		})
	}
	if targetRemote == "" {
		externalPromotionReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_target_remote_missing",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 external promotion requires an explicit target remote",
		})
	} else if !containsString(remoteNames, targetRemote) {
		externalPromotionReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_target_remote_not_configured",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 external promotion target remote is not configured locally",
			Detail:   targetRemote,
		})
	}
	if targetBranch == "" {
		externalPromotionReady = false
		findings = append(findings, runtimeguard.Finding{
			Code:     "promotion_target_branch_missing",
			Severity: runtimeguard.SeverityBlock,
			Message:  "A21 external promotion requires an explicit target branch",
		})
	}

	return promotionReadinessReport{
		SchemaVersion:          promotionReadinessSchema,
		GeneratedAtMS:          time.Now().UnixMilli(),
		WorktreePath:           worktreePath,
		ProjectRoot:            projectRoot,
		Branch:                 branch,
		Commit:                 commit,
		DirtyFileCount:         dirtyFileCount,
		RemoteNames:            remoteNames,
		LocalMainExists:        localMainExists,
		LocalMasterExists:      localMasterExists,
		TargetRemote:           targetRemote,
		TargetBranch:           targetBranch,
		Topics:                 topics,
		ReviewReady:            reviewReady,
		ExternalPromotionReady: externalPromotionReady,
		Result:                 runtimeguard.NewResult(findings),
	}, nil
}

func runPromotionGit(projectRoot string, args ...string) (string, error) {
	allArgs := append([]string{"-C", projectRoot}, args...)
	output, err := exec.Command("git", allArgs...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func promotionGitExitOK(projectRoot string, args ...string) bool {
	allArgs := append([]string{"-C", projectRoot}, args...)
	return exec.Command("git", allArgs...).Run() == nil
}

func splitNonEmptyLines(text string) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeJSONPromotionReadiness(writer io.Writer, report promotionReadinessReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
