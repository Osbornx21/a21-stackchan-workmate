package runtimeguard

import (
	"context"
	"testing"
)

func TestRunPreflightBlocksLegacyEnv(t *testing.T) {
	cfg := DefaultConfig()
	input := PreflightInput{
		Config: cfg,
		Env:    []string{"X21_BACKEND=http://127.0.0.1:8000"},
		CWD:    "/Users/jiyurun/Documents/New project",
	}

	result := RunPreflight(context.Background(), input)
	if result.OK {
		t.Fatal("expected preflight to block legacy env")
	}
	if result.Findings[0].Code != "legacy_env" {
		t.Fatalf("first code = %q, want legacy_env", result.Findings[0].Code)
	}
}

func TestRunPreflightPassesCleanConfigWithNoPortChecks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	input := PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
	}

	result := RunPreflight(context.Background(), input)
	if !result.OK {
		t.Fatalf("expected clean preflight, got %#v", result.Findings)
	}
}

func TestRunPreflightReportBlocksMissingFingerprint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	report := RunPreflightReport(context.Background(), PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
	})

	if report.Result.OK {
		t.Fatal("expected missing fingerprint to block preflight report")
	}
	if report.Result.Findings[0].Code != "fingerprint_missing" {
		t.Fatalf("first code = %q, want fingerprint_missing", report.Result.Findings[0].Code)
	}
}

func TestRunPreflightReportAllowsCompleteFingerprint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReservedPorts = nil
	report := RunPreflightReport(context.Background(), PreflightInput{
		Config: cfg,
		Env:    []string{"A21_MODE=dev"},
		CWD:    "/Users/jiyurun/Documents/New project",
		Runner: fakeRunner{
			"route -n get default":              "interface: en0\n",
			"dig +short " + DefaultDNSProbeHost: "198.18.0.145\n",
		},
	})

	if !report.Result.OK {
		t.Fatalf("expected complete fingerprint to pass, got %#v", report.Result.Findings)
	}
}
