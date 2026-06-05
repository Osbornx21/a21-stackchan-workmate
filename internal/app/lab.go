package app

import (
	"fmt"
	"io"
)

func RunLab(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "a21-lab commands: demo, provider-latency-bench, xiaozhi-voice-bench, xiaozhi-professional-bench, xiaozhi-physical-evidence, xiaozhi-physical-prd-review, xiaozhi-instrument-observation, physical-stackchan-evidence, latency-bench")
		return 0
	}
	switch args[0] {
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "provider-latency-bench":
		return runProviderLatencyBench(args[1:], stdout, stderr)
	case "xiaozhi-voice-bench":
		return runXiaozhiVoiceBench(args[1:], stdout, stderr)
	case "xiaozhi-professional-bench":
		return runXiaozhiProfessionalBench(args[1:], stdout, stderr)
	case "xiaozhi-physical-evidence":
		return runXiaozhiPhysicalEvidence(args[1:], stdout, stderr)
	case "xiaozhi-physical-prd-review":
		return runXiaozhiPhysicalPRDReview(args[1:], stdout, stderr)
	case "xiaozhi-instrument-observation":
		return runXiaozhiInstrumentObservation(args[1:], stdout, stderr)
	case "physical-stackchan-evidence":
		return runPhysicalStackChanEvidence(args[1:], stdout, stderr)
	case "latency-bench":
		return runLatencyBench(args[1:], stdout, stderr)
	default:
		return Run(args, stdout, stderr)
	}
}
