package firmwarecheck

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFlashPlanConfirmsUploadAndDeviceIdentityWithoutEnablingFlash(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 123456000
    }
  ]
}`)

	result, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		MaxDeviceAgeMS:    5000,
		NowMS:             123456789,
	})
	if err != nil {
		t.Fatalf("BuildFlashPlan returned error: %v", err)
	}
	if result.GuardID != "a21.firmware.flash_plan_guard.v1" {
		t.Fatalf("GuardID = %q", result.GuardID)
	}
	if !result.DryRun {
		t.Fatal("DryRun = false, want true")
	}
	if result.FlashAllowed {
		t.Fatal("FlashAllowed = true, want false")
	}
	if !result.Upload.OK || !result.DeviceIdentity.OK {
		t.Fatalf("upload/device guards not ok: %#v", result)
	}
	if result.ArtifactSHA256 == "" || result.ArtifactSHA256 != result.Upload.Artifact.SHA256 {
		t.Fatalf("ArtifactSHA256 = %q, upload sha = %q", result.ArtifactSHA256, result.Upload.Artifact.SHA256)
	}
	if result.Port != "/dev/cu.usbmodemA21" || result.DeviceID != "stackchan-001" {
		t.Fatalf("port/device = %q/%q", result.Port, result.DeviceID)
	}
	if result.GeneratedAtMS != 123456789 {
		t.Fatalf("GeneratedAtMS = %d, want 123456789", result.GeneratedAtMS)
	}
}

func TestBuildFlashPlanRejectsMissingFreshnessGuard(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 123456000
    }
  ]
}`)

	_, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		NowMS:             123456789,
	})
	if err == nil {
		t.Fatal("expected missing max device age guard to be rejected")
	}
	if !strings.Contains(err.Error(), "max device age") {
		t.Fatalf("error = %q, want max device age guard", err)
	}
}

func TestBuildFlashPlanRequiresOnlineConnectionStatus(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 123456000
    }
  ]
}`)

	_, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		MaxDeviceAgeMS:    5000,
		NowMS:             123456789,
	})
	if err == nil {
		t.Fatal("expected missing online connection status to be rejected")
	}
	if !strings.Contains(err.Error(), "connection_status") {
		t.Fatalf("error = %q, want connection_status guard", err)
	}
}

func TestBuildFlashPlanRejectsBusyPort(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{"devices":[]}`)

	_, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true, InUse: true, Detail: "p1234 pio"},
		MaxDeviceAgeMS:    5000,
	})
	if err == nil {
		t.Fatal("expected busy port to be rejected")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("error = %q, want already in use", err)
	}
}

func TestBuildFlashPlanRejectsStaleDeviceReport(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 1000
    }
  ]
}`)

	_, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		MaxDeviceAgeMS:    5000,
		NowMS:             8000,
	})
	if err == nil {
		t.Fatal("expected stale device report to be rejected")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("error = %q, want stale", err)
	}
}

func TestBuildFlashPlanRejectsActivePlaybackDevice(t *testing.T) {
	dir := t.TempDir()
	manifest := writeDeviceIdentityManifest(t, dir)
	artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
	writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
	report := writeDeviceIdentityReport(t, dir, `{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_expression": "speaking",
      "playback_stream_id": "a21-stream-active",
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`)

	_, err := BuildFlashPlan(FlashPlanOptions{
		ManifestPath:      manifest,
		ArtifactPath:      artifact,
		Port:              "/dev/cu.usbmodemA21",
		ReportPath:        report,
		ExpectedDeviceID:  "stackchan-001",
		ExpectedGitCommit: "abcdef123456",
		PortUsage:         PortUsage{Exists: true},
		MaxDeviceAgeMS:    300000,
		NowMS:             1780000000100,
	})
	if err == nil {
		t.Fatal("expected active playback device to be rejected")
	}
	if !strings.Contains(err.Error(), "active playback") {
		t.Fatalf("error = %q, want active playback", err)
	}
}

func TestBuildFlashPlanRejectsUnsafeDeviceRuntimeState(t *testing.T) {
	tests := []struct {
		name              string
		currentMode       string
		currentExpression string
		want              string
	}{
		{name: "thinking expression", currentExpression: "thinking", want: "current_expression"},
		{name: "professional expression", currentExpression: "professional", want: "current_expression"},
		{name: "error expression", currentExpression: "error", want: "current_expression"},
		{name: "professional mode", currentMode: "professional", want: "current_mode"},
		{name: "local fallback mode", currentMode: "local_fallback", want: "current_mode"},
		{name: "error mode", currentMode: "error", want: "current_mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			manifest := writeDeviceIdentityManifest(t, dir)
			artifact := filepath.Join(dir, "a21-stackchan-0.1.0-m5stack-cores3-abcdef123456-20260530-004500.bin")
			writeDeviceIdentityArtifact(t, artifact, []byte("firmware"))
			report := writeDeviceIdentityReportWithRuntimeState(t, dir, tt.currentMode, tt.currentExpression)

			_, err := BuildFlashPlan(FlashPlanOptions{
				ManifestPath:      manifest,
				ArtifactPath:      artifact,
				Port:              "/dev/cu.usbmodemA21",
				ReportPath:        report,
				ExpectedDeviceID:  "stackchan-001",
				ExpectedGitCommit: "abcdef123456",
				PortUsage:         PortUsage{Exists: true},
				MaxDeviceAgeMS:    300000,
				NowMS:             1780000000100,
			})
			if err == nil {
				t.Fatal("expected unsafe runtime state to be rejected")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func writeDeviceIdentityReportWithRuntimeState(t *testing.T, dir string, currentMode string, currentExpression string) string {
	t.Helper()
	return writeDeviceIdentityReport(t, dir, fmt.Sprintf(`{
  "devices": [
    {
      "device_id": "stackchan-001",
      "identity_status": "ok",
      "connection_status": "online",
      "current_mode": %q,
      "current_expression": %q,
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      },
      "last_seen_ms": 1780000000000
    }
  ]
}`, currentMode, currentExpression))
}
