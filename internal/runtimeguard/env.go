package runtimeguard

import (
	"net/url"
	"strconv"
	"strings"
)

func CheckEnv(cfg Config, env []string) []Finding {
	findings := make([]Finding, 0)
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		for _, legacy := range cfg.LegacyEnvPrefixes {
			if strings.HasPrefix(name, legacy) {
				findings = append(findings, Finding{
					Code:     "legacy_env",
					Severity: SeverityBlock,
					Message:  "A21 runtime refuses legacy project environment variables",
					Detail:   name,
				})
			}
		}
		if strings.HasPrefix(name, cfg.EnvPrefix) && isEndpointEnvName(name) {
			if port, ok := legacyEndpointPort(value, cfg.LegacyEndpointPorts); ok {
				findings = append(findings, Finding{
					Code:     "legacy_endpoint",
					Severity: SeverityBlock,
					Message:  "A21 runtime refuses endpoints that point to legacy project ports",
					Detail:   name + " uses legacy port " + strconv.Itoa(port),
				})
			}
		}
	}
	return findings
}

func isEndpointEnvName(name string) bool {
	upper := strings.ToUpper(name)
	return strings.Contains(upper, "URL") ||
		strings.Contains(upper, "ENDPOINT") ||
		strings.HasSuffix(upper, "_ADDR") ||
		strings.HasSuffix(upper, "_HOST")
}

func legacyEndpointPort(value string, legacyPorts []int) (int, bool) {
	for _, candidate := range endpointCandidates(value) {
		port, ok := candidatePort(candidate)
		if !ok {
			continue
		}
		for _, legacyPort := range legacyPorts {
			if port == legacyPort {
				return port, true
			}
		}
	}
	return 0, false
}

func endpointCandidates(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
}

func candidatePort(candidate string) (int, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return 0, false
	}
	if !strings.Contains(candidate, "://") && strings.Contains(candidate, ":") {
		candidate = "a21://" + candidate
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Port() == "" {
		return 0, false
	}
	port, err := strconv.Atoi(parsed.Port())
	return port, err == nil
}
