package runtimeguard

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

func CheckPorts(ports []int) []Finding {
	findings := make([]Finding, 0)
	for _, port := range ports {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			if !errors.Is(err, syscall.EADDRINUSE) {
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
