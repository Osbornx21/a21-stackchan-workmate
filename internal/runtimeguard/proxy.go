package runtimeguard

import (
	"net"
	"net/netip"
	"net/url"
	"strings"
)

type ProxyPolicyReport struct {
	GlobalProxyConfigured   bool     `json:"global_proxy_configured"`
	ProviderProxyConfigured bool     `json:"provider_proxy_configured"`
	GlobalProxyEnv          []string `json:"global_proxy_env,omitempty"`
	ProviderProxyEnv        []string `json:"provider_proxy_env,omitempty"`
	NoProxyEnv              []string `json:"no_proxy_env,omitempty"`
	DirectConnectRequired   []string `json:"direct_connect_required"`
	DirectConnectCovered    []string `json:"direct_connect_covered,omitempty"`
	DirectConnectMissing    []string `json:"direct_connect_missing,omitempty"`
	DirectConnectOK         bool     `json:"direct_connect_ok"`
}

var a21DirectConnectRequired = []string{
	"localhost",
	"127.0.0.1",
	"::1",
	".local",
	"10.0.0.0/8",
	"10.21.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

func EvaluateProxyPolicy(env []string) ProxyPolicyReport {
	report := ProxyPolicyReport{
		DirectConnectRequired: append([]string(nil), a21DirectConnectRequired...),
		DirectConnectOK:       true,
	}
	noProxyTokens := make([]string, 0)
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		trimmedValue := strings.TrimSpace(value)
		upperName := strings.ToUpper(strings.TrimSpace(name))
		switch upperName {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY":
			if trimmedValue != "" {
				report.GlobalProxyConfigured = true
				report.GlobalProxyEnv = appendUnique(report.GlobalProxyEnv, name)
			}
		case "A21_PROVIDER_PROXY_URL":
			if trimmedValue != "" {
				report.ProviderProxyConfigured = true
				report.ProviderProxyEnv = appendUnique(report.ProviderProxyEnv, name)
			}
		case "NO_PROXY", "A21_NO_PROXY":
			if trimmedValue != "" {
				report.NoProxyEnv = appendUnique(report.NoProxyEnv, name)
				noProxyTokens = append(noProxyTokens, splitNoProxy(trimmedValue)...)
			}
		}
	}

	if !report.GlobalProxyConfigured {
		return report
	}
	for _, target := range a21DirectConnectRequired {
		if noProxyCoversTarget(noProxyTokens, target) {
			report.DirectConnectCovered = append(report.DirectConnectCovered, target)
		} else {
			report.DirectConnectMissing = append(report.DirectConnectMissing, target)
		}
	}
	report.DirectConnectOK = len(report.DirectConnectMissing) == 0
	return report
}

func CheckProxyPolicy(report ProxyPolicyReport) []Finding {
	if !report.GlobalProxyConfigured || report.DirectConnectOK {
		return nil
	}
	return []Finding{{
		Code:     "proxy_direct_bypass_missing",
		Severity: SeverityBlock,
		Message:  "A21 runtime refuses global proxy settings without direct-connect NO_PROXY coverage",
		Detail:   strings.Join(report.DirectConnectMissing, ","),
	}}
}

func splitNoProxy(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		token := normalizeNoProxyToken(part)
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func normalizeNoProxyToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	if token == "" {
		return ""
	}
	if parsed, err := url.Parse(token); err == nil && parsed.Host != "" {
		token = parsed.Host
	}
	if host, _, err := net.SplitHostPort(token); err == nil {
		token = strings.Trim(host, "[]")
	}
	token = strings.Trim(token, "[]")
	if strings.HasPrefix(token, "*.") {
		token = strings.TrimPrefix(token, "*")
	}
	return token
}

func noProxyCoversTarget(tokens []string, target string) bool {
	for _, token := range tokens {
		if token == "*" || token == strings.ToLower(target) {
			return true
		}
		if target == ".local" && (token == ".local" || token == "local") {
			return true
		}
		if targetPrefix, ok := parsePrefix(target); ok {
			if tokenPrefix, ok := parsePrefix(token); ok && prefixCovers(tokenPrefix, targetPrefix) {
				return true
			}
		}
		if targetAddr, ok := parseAddr(target); ok {
			if tokenPrefix, ok := parsePrefix(token); ok && tokenPrefix.Contains(targetAddr) {
				return true
			}
		}
	}
	return false
}

func parsePrefix(value string) (netip.Prefix, bool) {
	prefix, err := netip.ParsePrefix(value)
	return prefix, err == nil
}

func parseAddr(value string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(value)
	return addr, err == nil
}

func prefixCovers(candidate netip.Prefix, target netip.Prefix) bool {
	return candidate.Contains(target.Addr()) && candidate.Bits() <= target.Bits()
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
