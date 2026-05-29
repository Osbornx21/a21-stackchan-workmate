package runtimeguard

import "context"

type PreflightInput struct {
	Config Config
	Env    []string
	CWD    string
	Runner CommandRunner
}

type PreflightReport struct {
	Result      Result      `json:"result"`
	Fingerprint Fingerprint `json:"fingerprint"`
}

func RunPreflight(ctx context.Context, input PreflightInput) Result {
	cfg := input.Config
	if cfg.ProjectName == "" {
		cfg = DefaultConfig()
	}

	findings := make([]Finding, 0)
	findings = append(findings, CheckEnv(cfg, input.Env)...)
	findings = append(findings, CheckWorkingDirectory(cfg, input.CWD)...)
	findings = append(findings, CheckPorts(cfg.ReservedPorts)...)

	return NewResult(findings)
}

func RunPreflightReport(ctx context.Context, input PreflightInput) PreflightReport {
	fingerprint := DetectFingerprint(ctx, input.Runner, input.Env)
	result := RunPreflight(ctx, input)
	findings := append([]Finding{}, result.Findings...)
	findings = append(findings, CheckFingerprint(fingerprint)...)
	return PreflightReport{
		Result:      NewResult(findings),
		Fingerprint: fingerprint,
	}
}
