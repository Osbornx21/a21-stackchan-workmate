package runtimeguard

import "testing"

func TestDefaultConfigKeepsA21Isolated(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ProjectName != "A21" {
		t.Fatalf("ProjectName = %q, want A21", cfg.ProjectName)
	}
	if cfg.EnvPrefix != "A21_" {
		t.Fatalf("EnvPrefix = %q, want A21_", cfg.EnvPrefix)
	}
	if len(cfg.LegacyEnvPrefixes) == 0 {
		t.Fatal("expected legacy env prefixes")
	}
	for _, port := range []int{8000, 8080, 10095, 18080} {
		if cfg.IsReservedPort(port) {
			t.Fatalf("legacy port %d must not be reserved for A21", port)
		}
	}
	for _, port := range []int{21080, 21081, 21073, 21086, 21095, 21114, 21434} {
		if !cfg.IsReservedPort(port) {
			t.Fatalf("A21 port %d should be reserved", port)
		}
	}
}
