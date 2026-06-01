package runtimeguard

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestCheckPortsBlocksOccupiedReservedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	findings := CheckPorts([]int{port})
	if len(findings) != 1 {
		t.Fatalf("findings length = %d, want 1", len(findings))
	}
	if findings[0].Code != "reserved_port_in_use" {
		t.Fatalf("Code = %q, want reserved_port_in_use", findings[0].Code)
	}
}

func TestCheckPortsAllowsFreePort(t *testing.T) {
	findings := CheckPorts([]int{0})
	if len(findings) != 0 {
		t.Fatalf("expected no findings for kernel-assigned port, got %#v", findings)
	}
}

func TestCheckReservedPortsAllowsRunningA21GatewayOnActiveLaunchPort(t *testing.T) {
	port := startPortHealthServer(t, `{"service":"a21-gateway","status":"ok"}`)

	findings := CheckReservedPorts([]int{port}, []int{port})
	if len(findings) != 0 {
		t.Fatalf("expected running A21 Gateway to be accepted on active launch port, got %#v", findings)
	}
}

func TestCheckReservedPortsBlocksUnknownServiceOnActiveLaunchPort(t *testing.T) {
	port := startPortHealthServer(t, `{"service":"other","status":"ok"}`)

	findings := CheckReservedPorts([]int{port}, []int{port})
	if len(findings) != 1 {
		t.Fatalf("findings length = %d, want 1", len(findings))
	}
	if findings[0].Code != "reserved_port_in_use" {
		t.Fatalf("Code = %q, want reserved_port_in_use", findings[0].Code)
	}
}

func TestCheckReservedPortsBlocksA21GatewayOnNonActiveReservedPort(t *testing.T) {
	port := startPortHealthServer(t, `{"service":"a21-gateway","status":"ok"}`)

	findings := CheckReservedPorts([]int{port}, nil)
	if len(findings) != 1 {
		t.Fatalf("findings length = %d, want 1", len(findings))
	}
	if findings[0].Code != "reserved_port_in_use" {
		t.Fatalf("Code = %q, want reserved_port_in_use", findings[0].Code)
	}
}

func startPortHealthServer(t *testing.T, body string) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})
	return port
}
