package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func productSurfaceLabel(gatewayURL string, suffix string) string {
	parsed, err := url.Parse(gatewayURL)
	if err != nil || parsed.Host == "" {
		return "invalid_gateway"
	}
	host := parsed.Hostname()
	port := parsed.Port()
	scope := "remote"
	switch host {
	case "127.0.0.1", "localhost", "::1":
		scope = "loopback"
	}
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "" || suffix == "/" {
		return scope + ":" + port
	}
	return scope + ":" + port + "/" + strings.TrimLeft(suffix, "/")
}

func gatewayHealthOK(ctx context.Context, gatewayURL string) bool {
	endpoint := strings.TrimRight(gatewayURL, "/") + "/healthz"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	client := *a21DirectHTTPClient(700 * time.Millisecond)
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func gatewaySimulatorOK(ctx context.Context, gatewayURL string) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, productSimulatorURL(gatewayURL), nil)
	if err != nil {
		return false
	}
	client := *a21DirectHTTPClient(700 * time.Millisecond)
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func openBrowserURL(rawURL string) {
	if strings.TrimSpace(rawURL) == "" {
		return
	}
	_ = exec.Command("open", rawURL).Start()
}

func writeProductReadinessReport(outputDir string, report productReadinessReport) (string, error) {
	if outputDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-product-readiness-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = filepath.Base(reportPath)
	if err := writeJSONProductReadiness(file, report); err != nil {
		return "", err
	}
	return filepath.Base(reportPath), nil
}

func writeJSONProductReadiness(writer io.Writer, report productReadinessReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
