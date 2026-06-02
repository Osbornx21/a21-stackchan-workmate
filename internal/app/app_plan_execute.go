package app

import "io"

func splitExecuteFlag(args []string) ([]string, bool) {
	clean := make([]string, 0, len(args))
	execute := false
	for _, arg := range args {
		if arg == "--execute" {
			execute = true
			continue
		}
		clean = append(clean, arg)
	}
	return clean, execute
}

func withExecuteFlag(args []string) []string {
	next := make([]string, 0, len(args)+1)
	next = append(next, "--execute")
	next = append(next, args...)
	return next
}

func runStackChanOfficialAudioSmokeFlashCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	return runStackChanOfficialAudioSmokeFlash(clean, execute, stdout, stderr)
}

func runStackChanOfficialPCMBridgeFlashCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	return runStackChanOfficialPCMBridgeFlash(clean, execute, stdout, stderr)
}

func runStackChanOfficialPCMBridgeNVSCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	return runStackChanOfficialPCMBridgeNVS(clean, execute, stdout, stderr)
}

func runFirmwareBootstrapFlash(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	if execute {
		return runFirmwareBootstrapFlashExecute(clean, stdout, stderr)
	}
	return runFirmwareBootstrapFlashPlan(clean, stdout, stderr)
}

func runFirmwareMicProbeFlash(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	if execute {
		return runFirmwareMicProbeFlashExecute(clean, stdout, stderr)
	}
	return runFirmwareMicProbeFlashPlan(clean, stdout, stderr)
}

func runFirmwareIMUProbeFlash(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	if execute {
		return runFirmwareIMUProbeFlashExecute(clean, stdout, stderr)
	}
	return runFirmwareIMUProbeFlashPlan(clean, stdout, stderr)
}

func runFirmwareSensorProbeFlash(args []string, stdout io.Writer, stderr io.Writer) int {
	clean, execute := splitExecuteFlag(args)
	if execute {
		return runFirmwareSensorProbeFlashExecute(clean, stdout, stderr)
	}
	return runFirmwareSensorProbeFlashPlan(clean, stdout, stderr)
}

func runDeprecatedPlanExecuteAlias(args []string, stdout io.Writer, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	switch args[0] {
	case "stackchan-official-audio-smoke-flash-plan":
		return runStackChanOfficialAudioSmokeFlashCLI(args[1:], stdout, stderr), true
	case "stackchan-official-audio-smoke-flash-execute":
		return runStackChanOfficialAudioSmokeFlashCLI(withExecuteFlag(args[1:]), stdout, stderr), true
	case "stackchan-official-pcm-bridge-flash-plan":
		return runStackChanOfficialPCMBridgeFlashCLI(args[1:], stdout, stderr), true
	case "stackchan-official-pcm-bridge-flash-execute":
		return runStackChanOfficialPCMBridgeFlashCLI(withExecuteFlag(args[1:]), stdout, stderr), true
	case "stackchan-official-pcm-bridge-nvs-plan":
		return runStackChanOfficialPCMBridgeNVSCLI(args[1:], stdout, stderr), true
	case "stackchan-official-pcm-bridge-nvs-execute":
		return runStackChanOfficialPCMBridgeNVSCLI(withExecuteFlag(args[1:]), stdout, stderr), true
	case "firmware-bootstrap-flash-plan":
		return runFirmwareBootstrapFlash(args[1:], stdout, stderr), true
	case "firmware-bootstrap-flash-execute":
		return runFirmwareBootstrapFlash(withExecuteFlag(args[1:]), stdout, stderr), true
	case "firmware-mic-probe-flash-plan":
		return runFirmwareMicProbeFlash(args[1:], stdout, stderr), true
	case "firmware-mic-probe-flash-execute":
		return runFirmwareMicProbeFlash(withExecuteFlag(args[1:]), stdout, stderr), true
	case "firmware-imu-probe-flash-plan":
		return runFirmwareIMUProbeFlash(args[1:], stdout, stderr), true
	case "firmware-imu-probe-flash-execute":
		return runFirmwareIMUProbeFlash(withExecuteFlag(args[1:]), stdout, stderr), true
	case "firmware-sensor-probe-flash-plan":
		return runFirmwareSensorProbeFlash(args[1:], stdout, stderr), true
	case "firmware-sensor-probe-flash-execute":
		return runFirmwareSensorProbeFlash(withExecuteFlag(args[1:]), stdout, stderr), true
	case "xiaozhi-firmware-flash-plan":
		return runXiaozhiFirmwareFlash(args[1:], stdout, stderr), true
	case "xiaozhi-firmware-flash-execute":
		return runXiaozhiFirmwareFlash(withExecuteFlag(args[1:]), stdout, stderr), true
	default:
		return 0, false
	}
}

func runAuxiliaryCommandAlias(args []string, stdout io.Writer, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	switch args[0] {
	case "agent-plan":
		return runAgentPlan(args[1:], stdout, stderr), true
	case "agent-io-smoke":
		return runAgentIOSmoke(args[1:], stdout, stderr), true
	case "provider-realtime-plan":
		return runProviderRealtimePlan(args[1:], stdout, stderr), true
	case "provider-realtime-fixture":
		return runProviderRealtimeFixture(args[1:], stdout, stderr), true
	case "v21-adapter-smoke":
		return runV21AdapterSmoke(args[1:], stdout, stderr), true
	case "v21-professional-readiness":
		return runV21ProfessionalReadiness(args[1:], stdout, stderr), true
	case "v21-adapter-bridge":
		return runV21AdapterBridge(args[1:], stdout, stderr), true
	case "audio-front-end-plan":
		return runAudioFrontEndPlan(args[1:], stdout, stderr), true
	case "audio-front-end-eval":
		return runAudioFrontEndEval(args[1:], stdout, stderr), true
	case "local-tts-smoke":
		return runLocalTTSSmoke(args[1:], stdout, stderr), true
	case "local-asr-smoke":
		return runLocalASRSmoke(args[1:], stdout, stderr), true
	case "firmware-device-report":
		return runFirmwareDeviceReport(args[1:], stdout, stderr), true
	case "wake-word-firmware-plan":
		return runWakeWordFirmwarePlan(args[1:], stdout, stderr), true
	case "wake-word-firmware-build-receipt":
		return runWakeWordFirmwareBuildReceipt(args[1:], stdout, stderr), true
	case "wake-word-firmware-package":
		return runWakeWordFirmwarePackage(args[1:], stdout, stderr), true
	case "wake-word-physical-acceptance":
		return runWakeWordPhysicalAcceptance(args[1:], stdout, stderr), true
	case "stackchan-official-baseline":
		return runStackChanOfficialBaseline(args[1:], stdout, stderr), true
	case "stackchan-official-audio-smoke-flash":
		return runStackChanOfficialAudioSmokeFlashCLI(args[1:], stdout, stderr), true
	case "stackchan-official-pcm-bridge-flash":
		return runStackChanOfficialPCMBridgeFlashCLI(args[1:], stdout, stderr), true
	case "stackchan-official-pcm-bridge-nvs":
		return runStackChanOfficialPCMBridgeNVSCLI(args[1:], stdout, stderr), true
	case "serial-list":
		return runSerialList(args[1:], stdout, stderr), true
	case "firmware-package":
		return runFirmwarePackage(args[1:], stdout, stderr), true
	case "firmware-artifact-prune-plan":
		return runFirmwareArtifactPrunePlan(args[1:], stdout, stderr), true
	case "firmware-flash-plan":
		return runFirmwareFlashPlan(args[1:], stdout, stderr), true
	case "firmware-bootstrap-flash":
		return runFirmwareBootstrapFlash(args[1:], stdout, stderr), true
	case "firmware-mic-probe-flash":
		return runFirmwareMicProbeFlash(args[1:], stdout, stderr), true
	case "firmware-imu-probe-flash":
		return runFirmwareIMUProbeFlash(args[1:], stdout, stderr), true
	case "firmware-sensor-probe-flash":
		return runFirmwareSensorProbeFlash(args[1:], stdout, stderr), true
	case "xiaozhi-firmware-flash":
		return runXiaozhiFirmwareFlash(args[1:], stdout, stderr), true
	default:
		return 0, false
	}
}
