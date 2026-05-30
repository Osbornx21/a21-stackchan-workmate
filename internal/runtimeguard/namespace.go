package runtimeguard

import "strings"

type NamespaceAuditReport struct {
	Result       Result `json:"result"`
	FilesScanned int    `json:"files_scanned"`
}

func AuditNamespacePaths(paths []string) NamespaceAuditReport {
	findings := make([]Finding, 0)
	for _, path := range paths {
		normalized := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
		if !containsLegacyNamespace(normalized) || isAllowedLegacyNamespacePath(normalized) {
			continue
		}
		findings = append(findings, Finding{
			Code:     "namespace_legacy_path",
			Severity: SeverityBlock,
			Message:  "A21 repository path contains forbidden legacy project identity",
			Detail:   "legacy-looking path rejected",
		})
	}
	return NamespaceAuditReport{
		Result:       NewResult(findings),
		FilesScanned: len(paths),
	}
}

func containsLegacyNamespace(path string) bool {
	return strings.Contains(path, "x21") || strings.Contains(path, "v21")
}

func isAllowedLegacyNamespacePath(path string) bool {
	switch {
	case strings.HasPrefix(path, "internal/v21adapter/"):
		return true
	case path == "docs/engineering/v21_integration.md":
		return true
	case path == "docs/engineering/phase6a_v21_adapter.md":
		return true
	default:
		return false
	}
}
