package app

import (
	"a21.local/a21/internal/protocol"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type stackChanTouchCaseSpec struct {
	Name            string
	AcceptanceScope string
	CoreUX          bool
	ExpectedEvent   string
	ExpectedTrace   string
	ExpectedSource  string
	ControlState    protocol.ExpressionState
	ControlMode     protocol.Mode
	DevicePrompt    string
	HumanGuidance   string
	MockAudioChunks int
	StreamID        string
	NeedsAffordance bool
}
type stackChanTouchAcceptanceOptions struct {
	GatewayURL string
	DeviceID   string
	Case       string
	WindowMS   int
	OutputDir  string
}
type stackChanTouchAcceptanceObservation struct {
	Event     string `json:"event"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	AtMS      int64  `json:"at_ms,omitempty"`
	OffsetMS  int64  `json:"offset_ms,omitempty"`
	Source    string `json:"source,omitempty"`
}
type stackChanTouchAcceptanceReport struct {
	SchemaVersion      string                                `json:"schema_version"`
	GeneratedAtMS      int64                                 `json:"generated_at_ms"`
	DryRun             bool                                  `json:"dry_run"`
	FlashAllowed       bool                                  `json:"flash_allowed"`
	DeleteAllowed      bool                                  `json:"delete_allowed"`
	AcceptanceStatus   string                                `json:"touch_acceptance_status"`
	AcceptanceScope    string                                `json:"acceptance_scope"`
	CoreUX             bool                                  `json:"core_ux"`
	NeedsAffordance    bool                                  `json:"needs_affordance"`
	GatewayURL         string                                `json:"gateway_url"`
	DeviceID           string                                `json:"device_id"`
	Case               string                                `json:"case"`
	WindowMS           int                                   `json:"window_ms"`
	ExpectedEvent      string                                `json:"expected_event"`
	ExpectedTraceEvent string                                `json:"expected_trace_event"`
	ExpectedSource     string                                `json:"expected_source"`
	DevicePrompt       string                                `json:"device_prompt"`
	HumanGuidance      string                                `json:"human_guidance"`
	ControlTraceID     string                                `json:"control_trace_id,omitempty"`
	ControlSessionID   string                                `json:"control_session_id,omitempty"`
	ObservedEvents     []stackChanTouchAcceptanceObservation `json:"observed_events,omitempty"`
	ReportPath         string                                `json:"report_path,omitempty"`
	Findings           []officePreflightFinding              `json:"findings,omitempty"`
}

func (report *stackChanTouchAcceptanceReport) addFinding(code string, message string) {
	report.Findings = append(report.Findings, officePreflightFinding{Code: code, Message: message})
}
func runStackChanTouchAcceptance(args []string, stdout io.Writer, stderr io.Writer) int {
	options := defaultStackChanTouchAcceptanceOptions()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 stackchan-touch-acceptance --case screen_touch|top_tap|top_swipe_forward|top_swipe_backward|top_barge_in --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 [--window-ms 15000] [--output-dir reports]")
			return 0
		case "--gateway-url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--gateway-url requires a value")
				return 2
			}
			i++
			options.GatewayURL = args[i]
		case "--device-id":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--device-id requires a value")
				return 2
			}
			i++
			options.DeviceID = args[i]
		case "--case":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--case requires a value")
				return 2
			}
			i++
			options.Case = args[i]
		case "--window-ms":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--window-ms requires a value")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 || value > 120000 {
				fmt.Fprintln(stderr, "--window-ms must be between 1 and 120000")
				return 2
			}
			options.WindowMS = value
		case "--output-dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(stderr, "--output-dir requires a value")
				return 2
			}
			i++
			options.OutputDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown stackchan-touch-acceptance option %q\n", args[i])
			return 2
		}
	}
	if options.Case == "" {
		fmt.Fprintln(stderr, "--case requires a value")
		return 2
	}
	if options.DeviceID == "" {
		fmt.Fprintln(stderr, "--device-id requires a value")
		return 2
	}
	if err := validateA21ReportDir(options.OutputDir); err != nil {
		fmt.Fprintf(stderr, "stackchan touch acceptance report dir invalid: %v\n", err)
		return 1
	}
	report := runStackChanTouchAcceptanceCase(options)
	if options.OutputDir != "" {
		reportPath, err := writeStackChanTouchAcceptanceReport(options.OutputDir, report)
		if err != nil {
			fmt.Fprintf(stderr, "write stackchan touch acceptance report: %v\n", err)
			return 1
		}
		report.ReportPath = reportPath
	}
	if err := writeJSONStackChanTouchAcceptance(stdout, report); err != nil {
		fmt.Fprintf(stderr, "encode stackchan touch acceptance report: %v\n", err)
		return 1
	}
	if report.AcceptanceStatus != "passed" {
		fmt.Fprintln(stdout, "stackchan touch acceptance blocked (no flash performed)")
		return 1
	}
	fmt.Fprintln(stdout, "stackchan touch acceptance ok (no flash performed)")
	return 0
}
func defaultStackChanTouchAcceptanceOptions() stackChanTouchAcceptanceOptions {
	deviceID := strings.TrimSpace(os.Getenv("A21_DEVICE_ID"))
	if deviceID == "" {
		deviceID = "stackchan-001"
	}
	return stackChanTouchAcceptanceOptions{
		GatewayURL: "http://127.0.0.1:21080",
		DeviceID:   deviceID,
		WindowMS:   15000,
		OutputDir:  "reports",
	}
}
func runStackChanTouchAcceptanceCase(options stackChanTouchAcceptanceOptions) stackChanTouchAcceptanceReport {
	spec, ok := stackChanTouchCase(options.Case)
	report := stackChanTouchAcceptanceReport{
		SchemaVersion:    "a21.stackchan_touch_acceptance.v1",
		GeneratedAtMS:    time.Now().UnixMilli(),
		DryRun:           true,
		FlashAllowed:     false,
		DeleteAllowed:    false,
		AcceptanceStatus: "blocked",
		GatewayURL:       options.GatewayURL,
		DeviceID:         options.DeviceID,
		Case:             options.Case,
		WindowMS:         options.WindowMS,
	}
	if !ok {
		report.addFinding("unknown_touch_case", "touch acceptance case is not supported")
		return report
	}
	report.AcceptanceScope = spec.AcceptanceScope
	report.CoreUX = spec.CoreUX
	report.NeedsAffordance = spec.NeedsAffordance
	report.ExpectedEvent = spec.ExpectedEvent
	report.ExpectedTraceEvent = spec.ExpectedTrace
	report.ExpectedSource = spec.ExpectedSource
	report.DevicePrompt = spec.DevicePrompt
	report.HumanGuidance = spec.HumanGuidance

	before, err := fetchFirmwareDeviceReport(options.GatewayURL)
	if err != nil {
		report.addFinding("gateway_device_report_failed", err.Error())
		return report
	}
	if _, ok := findFirmwareDeviceRecord(before.Devices, options.DeviceID); !ok {
		report.addFinding("device_missing", "expected StackChan device is missing from Gateway")
		return report
	}

	control, err := postStackChanTouchControl(options.GatewayURL, options.DeviceID, spec)
	if err != nil {
		report.addFinding("device_control_failed", err.Error())
		return report
	}
	report.ControlTraceID = control.TraceID
	report.ControlSessionID = control.SessionID

	controlStartedAtMS := time.Now().UnixMilli()
	deadline := time.Now().Add(time.Duration(options.WindowMS) * time.Millisecond)
	seen := map[string]bool{}
	for time.Now().Before(deadline) {
		gatewayReport, err := fetchFirmwareDeviceReport(options.GatewayURL)
		if err != nil {
			report.addFinding("gateway_poll_failed", err.Error())
			return report
		}
		device, ok := findFirmwareDeviceRecord(gatewayReport.Devices, options.DeviceID)
		if !ok {
			report.addFinding("device_disconnected", "expected StackChan device disappeared during touch acceptance")
			return report
		}
		for _, traceID := range candidateStackChanTouchTraceIDs(device.LastTraceID) {
			trace, err := fetchGatewayTrace(options.GatewayURL, traceID)
			if err == nil {
				for _, event := range trace.Events {
					if event.AtMS > 0 && event.AtMS+1000 < controlStartedAtMS {
						continue
					}
					if !strings.HasPrefix(event.Name, "device.touch.") {
						continue
					}
					key := event.TraceID + "|" + event.Name + "|" + strconv.FormatInt(event.AtMS, 10)
					if !seen[key] {
						seen[key] = true
						report.ObservedEvents = append(report.ObservedEvents, stackChanTouchAcceptanceObservation{
							Event:     event.Name,
							TraceID:   event.TraceID,
							SessionID: event.SessionID,
							DeviceID:  event.DeviceID,
							AtMS:      event.AtMS,
							OffsetMS:  event.OffsetMS,
							Source:    string(device.LastTouchSource),
						})
					}
					if event.Name == spec.ExpectedTrace && (spec.ExpectedSource == "" || device.LastTouchSource == spec.ExpectedSource) {
						report.AcceptanceStatus = "passed"
						return report
					}
				}
			}
		}
		if string(device.LastEvent) == spec.ExpectedEvent && (spec.ExpectedSource == "" || device.LastTouchSource == spec.ExpectedSource) {
			report.ObservedEvents = append(report.ObservedEvents, stackChanTouchAcceptanceObservation{
				Event:     "device." + string(device.LastEvent) + ".received",
				TraceID:   device.LastTraceID,
				SessionID: device.LastSessionID,
				DeviceID:  device.DeviceID,
				AtMS:      device.LastSeenMS,
				Source:    string(device.LastTouchSource),
			})
			report.AcceptanceStatus = "passed"
			return report
		}
		time.Sleep(250 * time.Millisecond)
	}
	if len(report.ObservedEvents) == 0 {
		report.addFinding("touch_event_not_observed", "no touch event was observed inside the acceptance window")
	} else {
		report.addFinding("unexpected_touch_event", "touch events were observed, but not the expected event/source for this case")
	}
	if spec.NeedsAffordance {
		report.addFinding("physical_affordance_required", "directional top gestures need screen guidance, operator training, or physical labeling before they can be treated as core UX")
	}
	return report
}
func stackChanTouchCase(name string) (stackChanTouchCaseSpec, bool) {
	switch strings.TrimSpace(name) {
	case "screen_touch":
		return stackChanTouchCaseSpec{
			Name:            "screen_touch",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.wake_or_listen",
			ExpectedTrace:   "device.touch.wake_or_listen.received",
			ExpectedSource:  "screen",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Tap screen once",
			HumanGuidance:   "Tap the screen once and release. Do not touch the top strip.",
		}, true
	case "top_tap":
		return stackChanTouchCaseSpec{
			Name:            "top_tap",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.top.tap",
			ExpectedTrace:   "device.touch.top.tap.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: tap once",
			HumanGuidance:   "Tap the top touch strip once and release. Do not slide.",
		}, true
	case "top_swipe_forward":
		return stackChanTouchCaseSpec{
			Name:            "top_swipe_forward",
			AcceptanceScope: "guided_directional_touch",
			CoreUX:          false,
			NeedsAffordance: true,
			ExpectedEvent:   "touch.top.swipe_forward",
			ExpectedTrace:   "device.touch.top.swipe_forward.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: FACE -> USB",
			HumanGuidance:   "Slide once along the top strip from the screen/face side toward the USB/back side, then fully release.",
		}, true
	case "top_swipe_backward":
		return stackChanTouchCaseSpec{
			Name:            "top_swipe_backward",
			AcceptanceScope: "guided_directional_touch",
			CoreUX:          false,
			NeedsAffordance: true,
			ExpectedEvent:   "touch.top.swipe_backward",
			ExpectedTrace:   "device.touch.top.swipe_backward.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionIdle,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: USB -> FACE",
			HumanGuidance:   "Slide once along the top strip from the USB/back side toward the screen/face side, then fully release.",
		}, true
	case "top_barge_in":
		return stackChanTouchCaseSpec{
			Name:            "top_barge_in",
			AcceptanceScope: "core_touch",
			CoreUX:          true,
			ExpectedEvent:   "touch.barge_in",
			ExpectedTrace:   "device.touch.barge_in.received",
			ExpectedSource:  "top_sensor",
			ControlState:    protocol.ExpressionSpeaking,
			ControlMode:     protocol.ModeWorkmate,
			DevicePrompt:    "Top: interrupt",
			HumanGuidance:   "While A21 is in SPEAKING state, tap any part of the top touch strip once and release.",
			StreamID:        "a21-touch-acceptance-stream",
			MockAudioChunks: 2,
		}, true
	default:
		return stackChanTouchCaseSpec{}, false
	}
}
func writeStackChanTouchAcceptanceReport(outputDir string, report stackChanTouchAcceptanceReport) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(outputDir, "a21-stackchan-touch-acceptance-"+time.Now().Format("20060102-150405")+".json")
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	report.ReportPath = reportPath
	if err := writeJSONStackChanTouchAcceptance(file, report); err != nil {
		return "", err
	}
	return reportPath, nil
}
func writeJSONStackChanTouchAcceptance(writer io.Writer, report stackChanTouchAcceptanceReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
