package runtimeguard

import "testing"

func TestCheckWorkingDirectoryBlocksLegacyProjectPath(t *testing.T) {
	cfg := DefaultConfig()
	findings := CheckWorkingDirectory(cfg, "/Users/jiyurun/Documents/小马暴力")
	result := NewResult(findings)

	if result.OK {
		t.Fatal("expected legacy working directory to block startup")
	}
	if result.Findings[0].Code != "legacy_cwd" {
		t.Fatalf("code = %q, want legacy_cwd", result.Findings[0].Code)
	}
}

func TestCheckWorkingDirectoryAllowsA21Workspace(t *testing.T) {
	cfg := DefaultConfig()
	result := NewResult(CheckWorkingDirectory(cfg, "/Users/jiyurun/Documents/New project"))
	if !result.OK {
		t.Fatalf("expected A21 workspace to pass, got %#v", result.Findings)
	}
}

func TestCheckWorkingDirectoryUsesPathBoundaries(t *testing.T) {
	cfg := Config{LegacyPathParts: []string{"/tmp/legacy"}}

	blocked := NewResult(CheckWorkingDirectory(cfg, "/tmp/legacy/child"))
	if blocked.OK {
		t.Fatal("expected child path to be blocked")
	}

	allowed := NewResult(CheckWorkingDirectory(cfg, "/tmp/legacy-backup"))
	if !allowed.OK {
		t.Fatalf("expected sibling path to pass, got %#v", allowed.Findings)
	}
}
