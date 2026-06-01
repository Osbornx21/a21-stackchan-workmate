package xiaozhi

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBuildDeviceExtensionEventRequiresExplicitOptIn(t *testing.T) {
	identity := Identity{
		DeviceID:  "stackchan-001",
		TraceID:   "a21-trace-device-extension",
		SessionID: "a21-session-device-extension",
	}
	event := DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "happy"}

	_, err := BuildDeviceExtensionEvent(identity, HelloFeatures{}, DeviceExtensionProfileStock, event)
	if !errors.Is(err, ErrDeviceEventsDisabled) {
		t.Fatalf("stock profile err = %v, want %v", err, ErrDeviceEventsDisabled)
	}

	data, err := BuildDeviceExtensionEvent(identity, HelloFeatures{DeviceEvents: true}, DeviceExtensionProfileStock, event)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(data)
	if strings.Contains(payload, "debug_metrics") || strings.Contains(payload, "device_events") {
		t.Fatalf("device extension payload leaked feature negotiation fields: %s", payload)
	}
	if strings.Contains(strings.ToLower(payload), "x21") || strings.Contains(strings.ToLower(payload), "v21") {
		t.Fatalf("device extension payload leaked legacy naming: %s", payload)
	}

	parsed, err := ParseDeviceExtensionEvent(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Kind != DeviceEventKindFace || parsed.Value != "happy" {
		t.Fatalf("parsed event = %+v, want face happy", parsed)
	}
}

func TestBuildDeviceExtensionEventAllowsDebugExtensionProfile(t *testing.T) {
	data, err := BuildDeviceExtensionEvent(
		Identity{DeviceID: "stackchan-001"},
		HelloFeatures{},
		DeviceExtensionProfileStackChan,
		DeviceExtensionEvent{Kind: DeviceEventKindMotion, Value: "nod", YAngle: 120},
	)
	if err != nil {
		t.Fatal(err)
	}

	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["type"] != "device" || wire["kind"] != "motion" || wire["motion"] != "nod" {
		t.Fatalf("wire = %#v, want type=device motion nod", wire)
	}
	if wire["y_angle"] != float64(85) {
		t.Fatalf("y_angle = %#v, want clamped 85", wire["y_angle"])
	}
}

func TestDeviceExtensionSchemaValidatesEnumsAndClamp(t *testing.T) {
	tests := []struct {
		name  string
		event DeviceExtensionEvent
		want  DeviceExtensionEvent
	}{
		{name: "state", event: DeviceExtensionEvent{Kind: DeviceEventKindState, Value: "thinking"}, want: DeviceExtensionEvent{Kind: DeviceEventKindState, Value: "thinking"}},
		{name: "face", event: DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "attentive"}, want: DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "attentive"}},
		{name: "display", event: DeviceExtensionEvent{Kind: DeviceEventKindDisplay, Value: "asr"}, want: DeviceExtensionEvent{Kind: DeviceEventKindDisplay, Value: "asr"}},
		{name: "motion low clamp", event: DeviceExtensionEvent{Kind: DeviceEventKindMotion, Value: "look_up", YAngle: -20}, want: DeviceExtensionEvent{Kind: DeviceEventKindMotion, Value: "look_up", YAngle: 5}},
		{name: "heartbeat", event: DeviceExtensionEvent{Kind: DeviceEventKindHeartbeat}, want: DeviceExtensionEvent{Kind: DeviceEventKindHeartbeat}},
		{name: "playback start", event: DeviceExtensionEvent{Kind: DeviceEventKindPlayback, Value: "start", StreamID: "a21-xiaozhi-stream-001"}, want: DeviceExtensionEvent{Kind: DeviceEventKindPlayback, Value: "start", StreamID: "a21-xiaozhi-stream-001"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeDeviceExtensionEvent(tc.event)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tc.want.Kind || got.Value != tc.want.Value || got.YAngle != tc.want.YAngle {
				t.Fatalf("normalized = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseDeviceExtensionPlaybackStartRuntimeEcho(t *testing.T) {
	event, err := ParseDeviceExtensionEvent([]byte(`{
		"type":"device",
		"kind":"playback",
		"playback":"start",
		"stream_id":"a21-xiaozhi-stream-001",
		"trace_id":"a21-trace-xiaozhi-playback",
		"session_id":"a21-session-xiaozhi-playback",
		"device_id":"stackchan-debug-001"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != DeviceEventKindPlayback || event.Value != "start" || event.StreamID != "a21-xiaozhi-stream-001" {
		t.Fatalf("event = %+v, want playback start with stream id", event)
	}
}

func TestDeviceExtensionBadValuesUseStableErrorsWithoutLegacyLeak(t *testing.T) {
	tests := []struct {
		name  string
		event DeviceExtensionEvent
		want  error
	}{
		{name: "bad kind", event: DeviceExtensionEvent{Kind: DeviceEventKind("avatar"), Value: "happy"}, want: ErrUnsupportedDeviceEventKind},
		{name: "bad face", event: DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "grimace"}, want: ErrUnsupportedDeviceEventValue},
		{name: "legacy value", event: DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "x21-happy"}, want: ErrLegacyIdentity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizeDeviceExtensionEvent(tc.event)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if err != nil && (strings.Contains(strings.ToLower(err.Error()), "x21") || strings.Contains(strings.ToLower(err.Error()), "v21")) {
				t.Fatalf("error leaked legacy naming: %v", err)
			}
		})
	}
}

func TestParseInlineDeviceMarksStripsMarksAndReturnsSemanticEvents(t *testing.T) {
	parsed, err := ParseInlineDeviceMarks("我们先稳一下 [face:happy] 然后继续 [motion:nod]。")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.SpokenText != "我们先稳一下 然后继续 。" {
		t.Fatalf("spoken text = %q, want marks stripped", parsed.SpokenText)
	}
	if len(parsed.Events) != 2 {
		t.Fatalf("events = %+v, want two semantic events", parsed.Events)
	}
	if parsed.Events[0] != (DeviceExtensionEvent{Kind: DeviceEventKindFace, Value: "happy"}) {
		t.Fatalf("first event = %+v, want face happy", parsed.Events[0])
	}
	if parsed.Events[1] != (DeviceExtensionEvent{Kind: DeviceEventKindMotion, Value: "nod"}) {
		t.Fatalf("second event = %+v, want motion nod", parsed.Events[1])
	}
}

func TestParseInlineDeviceMarksRejectsBadMarksWithStableErrors(t *testing.T) {
	_, err := ParseInlineDeviceMarks("坏标记 [face:x21-happy] 不应该泄漏。")
	if !errors.Is(err, ErrLegacyIdentity) {
		t.Fatalf("err = %v, want %v", err, ErrLegacyIdentity)
	}
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "x21") {
		t.Fatalf("error leaked legacy naming: %v", err)
	}
}
