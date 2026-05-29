package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestProviderNetworkPolicyDefaultsToDirectAndIgnoresAmbientProxy(t *testing.T) {
	policy, report := NetworkPolicyFromEnv([]string{
		"HTTPS_PROXY=http://user:secret@127.0.0.1:7890",
	})

	if policy.Mode != NetworkModeDirect {
		t.Fatalf("mode = %q, want direct", policy.Mode)
	}
	if !report.AmbientProxyIgnored {
		t.Fatal("expected ambient proxy to be ignored")
	}
	if len(report.AmbientProxyEnv) != 1 || report.AmbientProxyEnv[0] != "HTTPS_PROXY" {
		t.Fatalf("ambient proxy env = %#v", report.AmbientProxyEnv)
	}
	client, err := NewProviderHTTPClient(policy)
	if err != nil {
		t.Fatal(err)
	}
	if client.Transport == nil {
		t.Fatal("expected explicit transport")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://dashscope.aliyuncs.com/compatible-mode/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, err := providerTransportProxy(client.Transport, req)
	if err != nil {
		t.Fatal(err)
	}
	if proxyURL != nil {
		t.Fatalf("direct provider client used proxy %s", proxyURL.Redacted())
	}
}

func TestProviderNetworkPolicyUsesExplicitProviderProxy(t *testing.T) {
	policy, report := NetworkPolicyFromEnv([]string{
		"HTTPS_PROXY=http://ambient-secret@127.0.0.1:7890",
		"A21_PROVIDER_PROXY_URL=http://provider-secret@127.0.0.1:7891",
	})

	if policy.Mode != NetworkModeExplicitProxy {
		t.Fatalf("mode = %q, want explicit_proxy", policy.Mode)
	}
	if !report.ProxyConfigured {
		t.Fatal("expected provider proxy to be configured")
	}
	if len(report.ProviderProxyEnv) != 1 || report.ProviderProxyEnv[0] != "A21_PROVIDER_PROXY_URL" {
		t.Fatalf("provider proxy env = %#v", report.ProviderProxyEnv)
	}
	client, err := NewProviderHTTPClient(policy)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://dashscope.aliyuncs.com/compatible-mode/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, err := providerTransportProxy(client.Transport, req)
	if err != nil {
		t.Fatal(err)
	}
	if proxyURL == nil || proxyURL.Host != "127.0.0.1:7891" {
		t.Fatalf("proxy URL = %v, want 127.0.0.1:7891", proxyURL)
	}
}

func TestProviderNetworkPolicyRejectsUnsupportedProxyScheme(t *testing.T) {
	policy, _ := NetworkPolicyFromEnv([]string{
		"A21_PROVIDER_PROXY_URL=socks5://127.0.0.1:7891",
	})

	_, err := NewProviderHTTPClient(policy)
	if err == nil {
		t.Fatal("expected unsupported proxy scheme error")
	}
	if !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("err = %v", err)
	}
}

func TestProviderNetworkReportDoesNotLeakProxyValues(t *testing.T) {
	_, report := NetworkPolicyFromEnv([]string{
		"HTTPS_PROXY=http://user:secret@127.0.0.1:7890",
		"A21_PROVIDER_PROXY_URL=http://provider-secret@127.0.0.1:7891",
	})

	rendered := report.Provider + strings.Join(report.AmbientProxyEnv, ",") + strings.Join(report.ProviderProxyEnv, ",") + string(report.Mode)
	for _, forbidden := range []string{"secret", "7890", "7891", "provider-secret"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("provider network report leaked %q: %#v", forbidden, report)
		}
	}
}
