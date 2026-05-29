package firmwarecheck

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type PortUsage struct {
	Exists bool   `json:"exists"`
	InUse  bool   `json:"in_use"`
	Detail string `json:"detail,omitempty"`
}

type SerialDevice struct {
	Path     string    `json:"path"`
	USBModem bool      `json:"usb_modem"`
	Usage    PortUsage `json:"usage"`
}

func DetectPortUsage(ctx context.Context, port string) (PortUsage, error) {
	if _, err := os.Stat(port); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PortUsage{Exists: false}, nil
		}
		return PortUsage{}, err
	}
	output, err := exec.CommandContext(ctx, "lsof", lsofPortUsageArgs(port)...).CombinedOutput()
	usage := portUsageFromLsofOutput(string(output))
	usage.Exists = true
	if usage.InUse || err == nil {
		return usage, nil
	}
	return usage, nil
}

func lsofPortUsageArgs(port string) []string {
	return []string{"-n", "-F", "pc", port}
}

func portUsageFromLsofOutput(output string) PortUsage {
	detail := strings.TrimSpace(output)
	if detail == "" {
		return PortUsage{}
	}
	if strings.Contains(detail, "No such file or directory") {
		return PortUsage{}
	}
	return PortUsage{InUse: true, Detail: detail}
}

func ListSerialDevices(ctx context.Context) ([]SerialDevice, error) {
	paths, err := filepath.Glob("/dev/cu.*")
	if err != nil {
		return nil, err
	}
	return serialDevicesFromPaths(paths, func(port string) (PortUsage, error) {
		return DetectPortUsage(ctx, port)
	}), nil
}

func serialDevicesFromPaths(paths []string, detect func(port string) (PortUsage, error)) []SerialDevice {
	sort.Strings(paths)
	devices := make([]SerialDevice, 0, len(paths))
	for _, path := range paths {
		usage, err := detect(path)
		if err != nil {
			usage = PortUsage{Exists: true, Detail: err.Error()}
		}
		devices = append(devices, SerialDevice{
			Path:     path,
			USBModem: strings.Contains(strings.ToLower(filepath.Base(path)), "usbmodem"),
			Usage:    usage,
		})
	}
	return devices
}
