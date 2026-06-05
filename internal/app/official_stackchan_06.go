package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

func writeStackChanOfficialBaselineFlashReport(outputDir string, report stackChanOfficialSmokeFlashReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now()
	reportPath := filepath.Join(outputDir, fmt.Sprintf("a21-stackchan-official-baseline-flash-%s-%d.json", now.Format("20060102-150405"), now.UnixNano()))
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
