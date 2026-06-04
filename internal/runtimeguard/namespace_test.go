package runtimeguard

import "testing"

func TestAuditNamespacePathsAllowsA21AndV21AdapterBoundary(t *testing.T) {
	report := AuditNamespacePaths([]string{
		"cmd/a21/main.go",
		"internal/v21adapter/client.go",
		"docs/engineering/V21_INTEGRATION.md",
		"docs/engineering/PHASE6A_V21_ADAPTER.md",
		"docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md",
		"docs/plans/2026-06-04-v21-a21-v2-workspace-query-scope-native-contract.md",
		"docs/plans/2026-06-04-v21-professional-execution-validation.md",
	})

	if !report.Result.OK {
		t.Fatalf("Result.OK = false, findings = %#v", report.Result.Findings)
	}
	if report.FilesScanned != 7 {
		t.Fatalf("FilesScanned = %d, want 7", report.FilesScanned)
	}
}

func TestAuditNamespacePathsRejectsLegacyRuntimePathsWithoutEchoingPath(t *testing.T) {
	report := AuditNamespacePaths([]string{
		"cmd/a21/main.go",
		"apps/x21-gateway/main.go",
		"services/v21-gateway/main.go",
	})

	if report.Result.OK {
		t.Fatal("Result.OK = true, want false")
	}
	if len(report.Result.Findings) != 2 {
		t.Fatalf("findings = %d, want 2: %#v", len(report.Result.Findings), report.Result.Findings)
	}
	for _, finding := range report.Result.Findings {
		if finding.Code != "namespace_legacy_path" {
			t.Fatalf("Code = %q, want namespace_legacy_path", finding.Code)
		}
		if finding.Detail != "legacy-looking path rejected" {
			t.Fatalf("Detail = %q, want redacted legacy detail", finding.Detail)
		}
	}
}
