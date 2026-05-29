package firmwarecheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePlatformIORejectsLiteralWiFiPasswordBuildFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platformio.ini")
	content := `[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
  -D A21_WIFI_PASSWORD=\"plain-secret\"
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validatePlatformIO(path, Manifest{Board: "m5stack-cores3"})
	if err == nil {
		t.Fatalf("expected literal Wi-Fi password build flag to be rejected")
	}
}

func TestValidatePlatformIOAllowsNoWiFiPasswordBuildFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platformio.ini")
	content := `[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validatePlatformIO(path, Manifest{Board: "m5stack-cores3"}); err != nil {
		t.Fatalf("expected platformio.ini without Wi-Fi password to pass, got %v", err)
	}
}

func TestValidatePlatformIORejectsMissingFirmwareBoardBuildFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platformio.ini")
	content := `[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts =
  pre:scripts/a21_block_raw_upload.py
  pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validatePlatformIO(path, Manifest{Board: "m5stack-cores3"})
	if err == nil {
		t.Fatalf("expected missing firmware board build flag to be rejected")
	}
}

func TestValidatePlatformIORejectsMissingBuildIdentityScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platformio.ini")
	content := `[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validatePlatformIO(path, Manifest{Board: "m5stack-cores3"})
	if err == nil {
		t.Fatalf("expected missing build identity script to be rejected")
	}
}

func TestValidatePlatformIORejectsMissingRawUploadBlockerScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platformio.ini")
	content := `[env:a21_stackchan_cores3]
platform = espressif32@7.0.1
board = m5stack-cores3
build_flags =
  -D A21_FIRMWARE_ID=\"a21-stackchan\"
  -D A21_FIRMWARE_VERSION=\"0.1.0\"
  -D A21_FIRMWARE_BOARD=\"m5stack-cores3\"
  -D A21_GATEWAY_HOST=\"10.21.0.1\"
  -D A21_GATEWAY_PORT=21080
extra_scripts = pre:scripts/a21_build_identity.py
lib_deps =
  m5stack/M5Unified @ 0.2.16
  bblanchon/ArduinoJson @ 7.4.3
  links2004/WebSockets @ 2.7.3

[env:a21_stackchan_native]
platform = native
test_framework = unity
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validatePlatformIO(path, Manifest{Board: "m5stack-cores3"})
	if err == nil {
		t.Fatalf("expected missing raw upload blocker script to be rejected")
	}
}
