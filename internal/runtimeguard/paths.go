package runtimeguard

import (
	"path/filepath"
	"strings"
)

func CheckWorkingDirectory(cfg Config, cwd string) []Finding {
	for _, legacyPath := range cfg.LegacyPathParts {
		if samePathOrChild(cwd, legacyPath) {
			return []Finding{{
				Code:     "legacy_cwd",
				Severity: SeverityBlock,
				Message:  "A21 runtime refuses to start from a legacy project working directory",
				Detail:   cwd,
			}}
		}
	}
	return nil
}

func samePathOrChild(path string, parent string) bool {
	cleanPath := filepath.Clean(path)
	cleanParent := filepath.Clean(parent)
	return cleanPath == cleanParent || strings.HasPrefix(cleanPath, cleanParent+string(filepath.Separator))
}
