package firmwarecheck

import (
	"context"
	"os/exec"
	"strings"
)

type PortUsage struct {
	InUse  bool   `json:"in_use"`
	Detail string `json:"detail,omitempty"`
}

func DetectPortUsage(ctx context.Context, port string) (PortUsage, error) {
	output, err := exec.CommandContext(ctx, "lsof", lsofPortUsageArgs(port)...).CombinedOutput()
	usage := portUsageFromLsofOutput(string(output))
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
