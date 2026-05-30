package firmwarecheck

import (
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
      "firmware": {
        "id": "a21-stackchan",
        "version": "0.1.0",
        "board": "m5stack-cores3",
        "commit": "abcdef123456"
      }
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
