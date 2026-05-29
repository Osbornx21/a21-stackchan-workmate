package runtimeguard

import (
	"net"
	"strconv"
	"testing"
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
