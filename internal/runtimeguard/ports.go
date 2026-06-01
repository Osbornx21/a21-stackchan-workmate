package runtimeguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

func CheckPorts(ports []int) []Finding {
	return CheckReservedPorts(ports, nil)
}

func CheckReservedPorts(ports []int, activeServicePorts []int) []Finding {
	findings := make([]Finding, 0)
	activeServices := intSet(activeServicePorts)
	for _, port := range ports {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			if !errors.Is(err, syscall.EADDRINUSE) {
				continue
			}
			if activeServices[port] && runningA21GatewayHealth(port) {
				continue
			}
			findings = append(findings, Finding{
				Code:     "reserved_port_in_use",
				Severity: SeverityBlock,
				Message:  "A21 reserved port is already in use",
				Detail:   addr,
			})
			continue
		}
		_ = listener.Close()
	}
	return findings
}

func intSet(values []int) map[int]bool {
	set := make(map[int]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func runningA21GatewayHealth(port int) bool {
	client := http.Client{
		Timeout: 250 * time.Millisecond,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	var payload struct {
		Service string `json:"service"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&payload); err != nil {
		return false
	}
	return payload.Service == "a21-gateway" && payload.Status == "ok"
}
