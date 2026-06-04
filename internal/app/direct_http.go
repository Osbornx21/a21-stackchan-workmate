package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const a21DirectSourceIPEnv = "A21_DIRECT_SOURCE_IP"

func a21DirectHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: a21DirectHTTPTransport(),
	}
}

func a21DirectHTTPTransport() *http.Transport {
	transport := &http.Transport{Proxy: nil}
	sourceIP := strings.TrimSpace(os.Getenv(a21DirectSourceIPEnv))
	if sourceIP == "" {
		return transport
	}
	parsed := net.ParseIP(sourceIP)
	if parsed == nil {
		transport.DialContext = func(context.Context, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("%s must be a valid IP address", a21DirectSourceIPEnv)
		}
		return transport
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		LocalAddr: &net.TCPAddr{IP: parsed},
	}
	transport.DialContext = dialer.DialContext
	return transport
}
