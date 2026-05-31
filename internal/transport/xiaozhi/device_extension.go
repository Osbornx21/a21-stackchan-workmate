package xiaozhi

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrDeviceEventsDisabled        = errors.New("device events disabled")
	ErrUnsupportedDeviceEventKind  = errors.New("unsupported device event kind")
	ErrUnsupportedDeviceEventValue = errors.New("unsupported device event value")
)

type DeviceExtensionProfile string

const (
	DeviceExtensionProfileStock     DeviceExtensionProfile = "stock"
	DeviceExtensionProfileDebug     DeviceExtensionProfile = "debug"
	DeviceExtensionProfileStackChan DeviceExtensionProfile = "stackchan_extension"
)

type DeviceEventKind string

const (
	DeviceEventKindState     DeviceEventKind = "state"
	DeviceEventKindFace      DeviceEventKind = "face"
	DeviceEventKindDisplay   DeviceEventKind = "display"
	DeviceEventKindMotion    DeviceEventKind = "motion"
	DeviceEventKindHeartbeat DeviceEventKind = "heartbeat"
)

type DeviceExtensionEvent struct {
	Kind   DeviceEventKind
	Value  string
	YAngle int
}

type InlineDeviceMarks struct {
	SpokenText string
	Events     []DeviceExtensionEvent
}

type deviceExtensionWire struct {
	Type      string `json:"type"`
	Kind      string `json:"kind"`
	State     string `json:"state,omitempty"`
	Face      string `json:"face,omitempty"`
	Display   string `json:"display,omitempty"`
	Motion    string `json:"motion,omitempty"`
	YAngle    *int   `json:"y_angle,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
}

var inlineDeviceMarkPattern = regexp.MustCompile(`\[(state|face|display|motion):([^\]\s]+)\]`)
var repeatedInlineSpacePattern = regexp.MustCompile(`[ \t]{2,}`)

func BuildDeviceExtensionEvent(identity Identity, features HelloFeatures, profile DeviceExtensionProfile, event DeviceExtensionEvent) ([]byte, error) {
	if !deviceExtensionAllowed(features, profile) {
		return nil, ErrDeviceEventsDisabled
	}
	resolved, err := resolveIdentity(identity, Identity{})
	if err != nil {
		return nil, err
	}
	normalized, err := NormalizeDeviceExtensionEvent(event)
	if err != nil {
		return nil, err
	}
	wire := deviceExtensionWire{
		Type:      "device",
		Kind:      string(normalized.Kind),
		TraceID:   resolved.TraceID,
		SessionID: resolved.SessionID,
		DeviceID:  resolved.DeviceID,
	}
	assignDeviceEventValue(&wire, normalized)
	return json.Marshal(wire)
}

func ParseDeviceExtensionEvent(data []byte) (DeviceExtensionEvent, error) {
	var wire deviceExtensionWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return DeviceExtensionEvent{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if strings.TrimSpace(wire.Type) != "device" {
		return DeviceExtensionEvent{}, fmt.Errorf("%w: device", ErrUnsupportedMessageType)
	}
	event := DeviceExtensionEvent{Kind: DeviceEventKind(strings.TrimSpace(wire.Kind))}
	switch event.Kind {
	case DeviceEventKindState:
		event.Value = wire.State
	case DeviceEventKindFace:
		event.Value = wire.Face
	case DeviceEventKindDisplay:
		event.Value = wire.Display
	case DeviceEventKindMotion:
		event.Value = wire.Motion
		if wire.YAngle != nil {
			event.YAngle = *wire.YAngle
		}
	case DeviceEventKindHeartbeat:
	default:
		event.Value = firstNonEmpty(wire.State, wire.Face, wire.Display, wire.Motion)
	}
	return NormalizeDeviceExtensionEvent(event)
}

func NormalizeDeviceExtensionEvent(event DeviceExtensionEvent) (DeviceExtensionEvent, error) {
	kind := DeviceEventKind(strings.TrimSpace(strings.ToLower(string(event.Kind))))
	value := strings.TrimSpace(strings.ToLower(event.Value))
	if containsLegacyIdentity(string(kind)) || containsLegacyIdentity(value) {
		return DeviceExtensionEvent{}, fmt.Errorf("%w: device extension", ErrLegacyIdentity)
	}

	normalized := DeviceExtensionEvent{
		Kind:   kind,
		Value:  value,
		YAngle: event.YAngle,
	}
	switch kind {
	case DeviceEventKindState:
		if !allowedDeviceEventValue(value, "idle", "listening", "thinking", "speaking", "error") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: state", ErrUnsupportedDeviceEventValue)
		}
	case DeviceEventKindFace:
		if !allowedDeviceEventValue(value, "idle", "attentive", "thinking", "speaking", "happy", "error") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: face", ErrUnsupportedDeviceEventValue)
		}
	case DeviceEventKindDisplay:
		if !allowedDeviceEventValue(value, "status", "asr", "tts") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: display", ErrUnsupportedDeviceEventValue)
		}
	case DeviceEventKindMotion:
		if !allowedDeviceEventValue(value, "look_up", "nod", "shake", "stop", "dance") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: motion", ErrUnsupportedDeviceEventValue)
		}
		if normalized.YAngle != 0 {
			normalized.YAngle = ClampYAngle(normalized.YAngle)
		}
	case DeviceEventKindHeartbeat:
		if value != "" {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: heartbeat", ErrUnsupportedDeviceEventValue)
		}
	default:
		return DeviceExtensionEvent{}, fmt.Errorf("%w: kind", ErrUnsupportedDeviceEventKind)
	}
	return normalized, nil
}

func ParseInlineDeviceMarks(text string) (InlineDeviceMarks, error) {
	events := make([]DeviceExtensionEvent, 0)
	var firstErr error
	cleaned := inlineDeviceMarkPattern.ReplaceAllStringFunc(text, func(mark string) string {
		if firstErr != nil {
			return ""
		}
		matches := inlineDeviceMarkPattern.FindStringSubmatch(mark)
		if len(matches) != 3 {
			return ""
		}
		event, err := NormalizeDeviceExtensionEvent(DeviceExtensionEvent{
			Kind:  DeviceEventKind(matches[1]),
			Value: matches[2],
		})
		if err != nil {
			firstErr = err
			return ""
		}
		events = append(events, event)
		return ""
	})
	if firstErr != nil {
		return InlineDeviceMarks{}, firstErr
	}
	return InlineDeviceMarks{
		SpokenText: repeatedInlineSpacePattern.ReplaceAllString(cleaned, " "),
		Events:     events,
	}, nil
}

func deviceExtensionAllowed(features HelloFeatures, profile DeviceExtensionProfile) bool {
	if features.DeviceEvents {
		return true
	}
	switch profile {
	case DeviceExtensionProfileDebug, DeviceExtensionProfileStackChan:
		return true
	default:
		return false
	}
}

func assignDeviceEventValue(wire *deviceExtensionWire, event DeviceExtensionEvent) {
	switch event.Kind {
	case DeviceEventKindState:
		wire.State = event.Value
	case DeviceEventKindFace:
		wire.Face = event.Value
	case DeviceEventKindDisplay:
		wire.Display = event.Value
	case DeviceEventKindMotion:
		wire.Motion = event.Value
		if event.YAngle != 0 {
			yAngle := event.YAngle
			wire.YAngle = &yAngle
		}
	}
}

func allowedDeviceEventValue(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
