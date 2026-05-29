package firmwarecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Manifest struct {
	Project        string `json:"project"`
	FirmwareID     string `json:"firmware_id"`
	Version        string `json:"version"`
	Board          string `json:"board"`
	ArtifactPrefix string `json:"artifact_prefix"`
}

type Result struct {
	Manifest Manifest `json:"manifest"`
	OK       bool     `json:"ok"`
}

func LoadAndValidate(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Result{}, err
	}
	if err := validate(manifest); err != nil {
		return Result{}, err
	}
	if err := validatePlatformIO(filepath.Join(filepath.Dir(path), "platformio.ini"), manifest); err != nil {
		return Result{}, err
	}
	return Result{Manifest: manifest, OK: true}, nil
}

func validate(manifest Manifest) error {
	if manifest.Project != "A21" {
		return fmt.Errorf("project must be A21")
	}
	if manifest.FirmwareID != "a21-stackchan" {
		return fmt.Errorf("firmware_id must be a21-stackchan")
	}
	if manifest.Board != "m5stack-cores3" {
		return fmt.Errorf("board must be m5stack-cores3")
	}
	if manifest.ArtifactPrefix != "a21-stackchan" {
		return fmt.Errorf("artifact_prefix must be a21-stackchan")
	}
	if ok, _ := regexp.MatchString(`^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$`, manifest.Version); !ok {
		return fmt.Errorf("version must be semver-like")
	}
	joined := strings.ToLower(strings.Join([]string{manifest.Project, manifest.FirmwareID, manifest.Board, manifest.ArtifactPrefix}, " "))
	if strings.Contains(joined, "x21") || strings.Contains(joined, "v21") {
		return fmt.Errorf("manifest contains forbidden legacy identity")
	}
	return nil
}

func validatePlatformIO(path string, manifest Manifest) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	if strings.Contains(content, "x21") || strings.Contains(content, "v21") {
		return fmt.Errorf("platformio.ini contains forbidden legacy identity")
	}
	if !strings.Contains(content, "[env:a21_") {
		return fmt.Errorf("platformio env must use a21 prefix")
	}
	if !strings.Contains(content, "platform = espressif32@7.0.1") {
		return fmt.Errorf("platformio platform must pin espressif32@7.0.1")
	}
	if !strings.Contains(content, "m5stack/m5unified @ 0.2.16") {
		return fmt.Errorf("platformio m5unified dependency must pin 0.2.16")
	}
	if !strings.Contains(content, "bblanchon/arduinojson @ 7.4.3") {
		return fmt.Errorf("platformio arduinojson dependency must pin 7.4.3")
	}
	if !strings.Contains(content, "links2004/websockets @ 2.7.3") {
		return fmt.Errorf("platformio websockets dependency must pin links2004/WebSockets 2.7.3")
	}
	if !strings.Contains(content, "[env:a21_stackchan_native]") || !strings.Contains(content, "test_framework = unity") {
		return fmt.Errorf("platformio native unit test environment is required")
	}
	if !strings.Contains(content, "a21_gateway_host") {
		return fmt.Errorf("platformio A21 gateway host build flag is required")
	}
	if !strings.Contains(content, "a21_gateway_port=21080") {
		return fmt.Errorf("platformio A21 gateway port must be 21080")
	}
	if strings.Contains(content, "a21_wifi_password") {
		return fmt.Errorf("platformio.ini must not contain A21 Wi-Fi password build flags")
	}
	boardPattern := regexp.MustCompile(`(?m)^\s*board\s*=\s*([A-Za-z0-9_-]+)\s*$`)
	match := boardPattern.FindStringSubmatch(content)
	if len(match) != 2 {
		return fmt.Errorf("platformio board is missing")
	}
	if match[1] != manifest.Board {
		return fmt.Errorf("platformio board %q does not match manifest board %q", match[1], manifest.Board)
	}
	if strings.Contains(content, "-t upload") || strings.Contains(content, "upload_port") {
		return fmt.Errorf("platformio upload configuration is forbidden before A21 upload guard")
	}
	return nil
}
