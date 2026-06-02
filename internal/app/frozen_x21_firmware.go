package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

const frozenExternalFirmwareSourceError = "build dir belongs to frozen external firmware source"

func validateNotFrozenExternalXiaozhiFirmwareBuildDir(buildDir string) error {
	if isFrozenExternalXiaozhiFirmwareBuildDir(buildDir) {
		return fmt.Errorf(frozenExternalFirmwareSourceError)
	}
	return nil
}

func isFrozenExternalXiaozhiFirmwareBuildDir(buildDir string) bool {
	cleaned := strings.TrimSpace(buildDir)
	if cleaned == "" {
		return false
	}
	abs, err := filepath.Abs(filepath.Clean(cleaned))
	if err != nil {
		abs = filepath.Clean(cleaned)
	}
	normalized := filepath.ToSlash(filepath.Clean(abs))
	knownSource := filepath.ToSlash(filepath.Clean("/Users/jiyurun/Documents/小马暴力/sources/xiaozhi-esp32"))
	if pathIsUnder(normalized, knownSource) {
		return true
	}
	knownProject := filepath.ToSlash(filepath.Clean("/Users/jiyurun/Documents/小马暴力"))
	if pathIsUnder(normalized, knownProject) && pathHasComponent(normalized, "xiaozhi-esp32") {
		return true
	}
	return pathHasComponentSequence(normalized, []string{"小马暴力", "sources", "xiaozhi-esp32"})
}

func pathIsUnder(path string, root string) bool {
	path = strings.TrimRight(path, "/")
	root = strings.TrimRight(root, "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func pathHasComponent(path string, component string) bool {
	for _, part := range strings.Split(path, "/") {
		if part == component {
			return true
		}
	}
	return false
}

func pathHasComponentSequence(path string, sequence []string) bool {
	if len(sequence) == 0 {
		return false
	}
	parts := strings.Split(path, "/")
	for i := 0; i+len(sequence) <= len(parts); i++ {
		matched := true
		for j, want := range sequence {
			if parts[i+j] != want {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
