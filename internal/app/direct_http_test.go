package app

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestA21DirectHTTPClientIgnoresAmbientProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:9")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:9")

	client := a21DirectHTTPClient(100 * time.Millisecond)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		req, _ := http.NewRequest(http.MethodGet, "http://a21.example.com/healthz", nil)
		proxyURL, _ := transport.Proxy(req)
		t.Fatalf("direct HTTP client inherited proxy %v", proxyURL)
	}
}

func TestA21DirectHTTPClientRejectsInvalidSourceIP(t *testing.T) {
	t.Setenv(a21DirectSourceIPEnv, "not-an-ip")

	client := a21DirectHTTPClient(100 * time.Millisecond)
	_, err := client.Get("http://127.0.0.1:1/healthz")
	if err == nil {
		t.Fatal("expected invalid source IP error")
	}
	if !strings.Contains(err.Error(), a21DirectSourceIPEnv) {
		t.Fatalf("error = %v, want %s", err, a21DirectSourceIPEnv)
	}
}
