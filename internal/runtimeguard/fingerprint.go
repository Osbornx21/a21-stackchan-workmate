package runtimeguard

import (
	"context"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

type Fingerprint struct {
	NetworkInterface    string   `json:"network_interface,omitempty"`
	ExternalDNSMappedIP string   `json:"external_dns_mapped_ip,omitempty"`
	ProxyEnv            []string `json:"proxy_env,omitempty"`
}

const DefaultDNSProbeHost = "example.com"

func (f Fingerprint) UsesTun() bool {
	return strings.HasPrefix(f.NetworkInterface, "utun")
}

func CheckFingerprint(fp Fingerprint) []Finding {
	findings := make([]Finding, 0)
	if fp.NetworkInterface == "" {
		findings = append(findings, Finding{
			Code:     "fingerprint_missing",
			Severity: SeverityBlock,
			Message:  "A21 runtime requires a network interface fingerprint",
			Detail:   "network_interface",
		})
	}
	if fp.ExternalDNSMappedIP == "" {
		findings = append(findings, Finding{
			Code:     "fingerprint_missing",
			Severity: SeverityBlock,
			Message:  "A21 runtime requires an external DNS fingerprint",
			Detail:   "external_dns_mapped_ip",
		})
	}
	return findings
}

func DetectFingerprint(ctx context.Context, runner CommandRunner, env []string) Fingerprint {
	fp := Fingerprint{ProxyEnv: proxyEnv(env)}
	if runner == nil {
		return fp
	}

	if out, err := runner.Run(ctx, "route", "-n", "get", "default"); err == nil {
		fp.NetworkInterface = parseRouteInterface(out)
	}
	if out, err := runner.Run(ctx, "dig", "+short", DefaultDNSProbeHost); err == nil {
		fp.ExternalDNSMappedIP = firstNonEmptyLine(out)
	}
	return fp
}

func proxyEnv(env []string) []string {
	keys := make([]string, 0)
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(name)
		if upper == "HTTP_PROXY" || upper == "HTTPS_PROXY" || upper == "ALL_PROXY" || upper == "NO_PROXY" {
			keys = append(keys, name)
		}
	}
	return keys
}

func parseRouteInterface(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	return ""
}

func firstNonEmptyLine(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
