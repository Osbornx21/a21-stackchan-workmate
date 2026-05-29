package providers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type NetworkMode string

const (
	NetworkModeDirect        NetworkMode = "direct"
	NetworkModeExplicitProxy NetworkMode = "explicit_proxy"
)

type NetworkPolicy struct {
	Mode     NetworkMode
	ProxyURL string
	Timeout  time.Duration
}

type NetworkReport struct {
	Provider            string      `json:"provider"`
	Mode                NetworkMode `json:"mode"`
	ProxyConfigured     bool        `json:"proxy_configured"`
	AmbientProxyIgnored bool        `json:"ambient_proxy_ignored"`
	AmbientProxyEnv     []string    `json:"ambient_proxy_env,omitempty"`
	ProviderProxyEnv    []string    `json:"provider_proxy_env,omitempty"`
}

func NetworkPolicyFromEnv(env []string) (NetworkPolicy, NetworkReport) {
	policy := NetworkPolicy{Mode: NetworkModeDirect, Timeout: 15 * time.Second}
	report := NetworkReport{Provider: "a21-provider-network", Mode: policy.Mode}
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		trimmedName := strings.TrimSpace(name)
		trimmedValue := strings.TrimSpace(value)
		if trimmedName == "" || trimmedValue == "" {
			continue
		}
		switch strings.ToUpper(trimmedName) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY":
			report.AmbientProxyIgnored = true
			report.AmbientProxyEnv = appendUniqueEnv(report.AmbientProxyEnv, trimmedName)
		case "A21_PROVIDER_PROXY_URL":
			policy.Mode = NetworkModeExplicitProxy
			policy.ProxyURL = trimmedValue
			report.Mode = policy.Mode
			report.ProxyConfigured = true
			report.ProviderProxyEnv = appendUniqueEnv(report.ProviderProxyEnv, trimmedName)
		}
	}
	return policy, report
}

func NewProviderHTTPClient(policy NetworkPolicy) (*http.Client, error) {
	transport, err := providerHTTPTransport(policy)
	if err != nil {
		return nil, err
	}
	timeout := policy.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

func providerHTTPTransport(policy NetworkPolicy) (*http.Transport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport is not configurable")
	}
	transport := base.Clone()
	transport.Proxy = nil
	mode := policy.Mode
	if mode == "" {
		mode = NetworkModeDirect
	}
	switch mode {
	case NetworkModeDirect:
		return transport, nil
	case NetworkModeExplicitProxy:
		proxyURL, err := parseProviderProxyURL(policy.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		return transport, nil
	default:
		return nil, fmt.Errorf("unsupported provider network mode %q", mode)
	}
}

func parseProviderProxyURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("provider proxy URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("provider proxy URL host is required")
	}
	return parsed, nil
}

func providerTransportProxy(roundTripper http.RoundTripper, req *http.Request) (*url.URL, error) {
	transport, ok := roundTripper.(*http.Transport)
	if !ok || transport.Proxy == nil {
		return nil, nil
	}
	return transport.Proxy(req)
}

func appendUniqueEnv(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
