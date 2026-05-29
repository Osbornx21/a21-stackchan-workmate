package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"a21.local/a21/internal/buildinfo"
	"a21.local/a21/internal/runtimeguard"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"version"}
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "%s %s (%s)\n", buildinfo.ProjectName, buildinfo.Version, buildinfo.ServiceName)
		return 0
	case "preflight", "doctor":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		report := runtimeguard.RunPreflightReport(context.Background(), runtimeguard.PreflightInput{
			Config: runtimeguard.DefaultConfig(),
			Env:    os.Environ(),
			CWD:    cwd,
			Runner: runtimeguard.OSRunner{},
		})
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode preflight report: %v\n", err)
			return 1
		}
		if !report.Result.OK {
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}
