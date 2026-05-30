package firmwarecheck

import (
	"fmt"
	"path/filepath"
	"sort"
)

const ArtifactPrunePlanSchemaVersion = "a21.firmware.artifact_prune_plan.v1"

type ArtifactPrunePlanOptions struct {
	ManifestPath string
	ArtifactDir  string
	Commit       string
	KeepRecent   int
}

type ArtifactPrunePlan struct {
	SchemaVersion       string                    `json:"schema_version"`
	DryRun              bool                      `json:"dry_run"`
	DeleteAllowed       bool                      `json:"delete_allowed"`
	ArtifactDir         string                    `json:"artifact_dir"`
	Commit              string                    `json:"commit"`
	KeepRecent          int                       `json:"keep_recent"`
	CurrentArtifactPath string                    `json:"current_artifact_path"`
	TotalArtifacts      int                       `json:"total_artifacts"`
	ReleaseValidCount   int                       `json:"release_valid_count"`
	Keep                []ArtifactRetentionRecord `json:"keep"`
	PruneCandidates     []ArtifactRetentionRecord `json:"prune_candidates"`
	ManualReview        []ArtifactRetentionRecord `json:"manual_review,omitempty"`
	ReportPath          string                    `json:"report_path,omitempty"`
}

type ArtifactRetentionRecord struct {
	ArtifactPath string   `json:"artifact_path"`
	SHA256Path   string   `json:"sha256_path,omitempty"`
	ManifestPath string   `json:"manifest_path,omitempty"`
	Timestamp    string   `json:"timestamp,omitempty"`
	Commit       string   `json:"commit,omitempty"`
	Reason       string   `json:"reason"`
	Files        []string `json:"files,omitempty"`
	Detail       string   `json:"detail,omitempty"`
}

func BuildArtifactPrunePlan(options ArtifactPrunePlanOptions) (ArtifactPrunePlan, error) {
	if options.ArtifactDir == "" {
		return ArtifactPrunePlan{}, fmt.Errorf("artifact directory is required")
	}
	if containsForbiddenPackagePathIdentity(options.ArtifactDir) {
		return ArtifactPrunePlan{}, fmt.Errorf("artifact directory contains forbidden legacy identity")
	}
	if options.KeepRecent < 0 {
		return ArtifactPrunePlan{}, fmt.Errorf("keep recent must be non-negative")
	}
	keepRecent := options.KeepRecent
	if keepRecent == 0 {
		keepRecent = 5
	}
	current, err := ValidateLatestArtifactForCommit(LatestArtifactOptions{
		ManifestPath: options.ManifestPath,
		ArtifactDir:  options.ArtifactDir,
		Commit:       options.Commit,
	})
	if err != nil {
		return ArtifactPrunePlan{}, fmt.Errorf("current release-ledger artifact invalid: %w", err)
	}
	matches, err := filepath.Glob(filepath.Join(options.ArtifactDir, "a21-stackchan-*.bin"))
	if err != nil {
		return ArtifactPrunePlan{}, err
	}
	sort.Strings(matches)

	valid := make([]ArtifactResult, 0, len(matches))
	manual := make([]ArtifactRetentionRecord, 0)
	for _, artifactPath := range matches {
		result, err := ValidateArtifact(ArtifactOptions{
			ManifestPath:           options.ManifestPath,
			ArtifactPath:           artifactPath,
			RequireReleaseIndex:    true,
			RequireReleaseManifest: true,
		})
		if err != nil {
			manual = append(manual, ArtifactRetentionRecord{
				ArtifactPath: artifactPath,
				SHA256Path:   artifactPath + ".sha256",
				ManifestPath: artifactPath + ".manifest.json",
				Reason:       "manual_review",
				Files:        artifactCompanionFiles(artifactPath),
				Detail:       err.Error(),
			})
			continue
		}
		valid = append(valid, result)
	}
	sort.SliceStable(valid, func(i, j int) bool {
		if valid[i].Timestamp == valid[j].Timestamp {
			return valid[i].ArtifactPath > valid[j].ArtifactPath
		}
		return valid[i].Timestamp > valid[j].Timestamp
	})

	keep := make([]ArtifactRetentionRecord, 0)
	prune := make([]ArtifactRetentionRecord, 0)
	kept := map[string]bool{}
	addKeep := func(result ArtifactResult, reason string) {
		key := filepath.Clean(result.ArtifactPath)
		if kept[key] {
			return
		}
		kept[key] = true
		keep = append(keep, artifactRetentionRecord(result, reason))
	}
	for _, result := range valid {
		if sameCleanAbsPath(result.ArtifactPath, current.ArtifactPath) {
			addKeep(result, "current")
			break
		}
	}
	for _, result := range valid {
		if len(keep) >= keepRecent {
			break
		}
		addKeep(result, "recent")
	}
	for _, result := range valid {
		if kept[filepath.Clean(result.ArtifactPath)] {
			continue
		}
		prune = append(prune, artifactRetentionRecord(result, "older_than_keep_recent"))
	}

	return ArtifactPrunePlan{
		SchemaVersion:       ArtifactPrunePlanSchemaVersion,
		DryRun:              true,
		DeleteAllowed:       false,
		ArtifactDir:         options.ArtifactDir,
		Commit:              options.Commit,
		KeepRecent:          keepRecent,
		CurrentArtifactPath: current.ArtifactPath,
		TotalArtifacts:      len(matches),
		ReleaseValidCount:   len(valid),
		Keep:                keep,
		PruneCandidates:     prune,
		ManualReview:        manual,
	}, nil
}

func artifactRetentionRecord(result ArtifactResult, reason string) ArtifactRetentionRecord {
	return ArtifactRetentionRecord{
		ArtifactPath: result.ArtifactPath,
		SHA256Path:   result.SHA256Path,
		ManifestPath: result.ReleaseManifestPath,
		Timestamp:    result.Timestamp,
		Commit:       result.Commit,
		Reason:       reason,
		Files:        artifactCompanionFiles(result.ArtifactPath),
	}
}

func artifactCompanionFiles(artifactPath string) []string {
	return []string{
		artifactPath,
		artifactPath + ".sha256",
		artifactPath + ".manifest.json",
	}
}
