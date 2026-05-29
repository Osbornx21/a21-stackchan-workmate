package buildinfo

import "testing"

func TestBuildInfoDefaults(t *testing.T) {
	if ServiceName != "a21-core" {
		t.Fatalf("ServiceName = %q, want %q", ServiceName, "a21-core")
	}
	if ProjectName != "A21" {
		t.Fatalf("ProjectName = %q, want %q", ProjectName, "A21")
	}
	if Version != "0.1.0-dev" {
		t.Fatalf("Version = %q, want %q", Version, "0.1.0-dev")
	}
}
