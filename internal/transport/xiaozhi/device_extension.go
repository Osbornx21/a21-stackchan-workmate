package xiaozhi

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	DeviceEventKindPlayback  DeviceEventKind = "playback"
	DeviceEventKindTouch     DeviceEventKind = "touch"
)

type DeviceExtensionEvent struct {
	Kind        DeviceEventKind
	Value       string
	YAngle      int
	StreamID    string
	Source      string
	Text        string
	Reason      string
	RuntimeEcho map[string]string
}

type InlineDeviceMarks struct {
	SpokenText string
	Events     []DeviceExtensionEvent
}

type deviceExtensionWire struct {
	Type      string `json:"type"`
	Kind      string `json:"kind,omitempty"`
	Event     string `json:"event,omitempty"`
	State     string `json:"state,omitempty"`
	Face      string `json:"face,omitempty"`
	Emotion   string `json:"emotion,omitempty"`
	Display   string `json:"display,omitempty"`
	Slot      string `json:"slot,omitempty"`
	Motion    string `json:"motion,omitempty"`
	Name      string `json:"name,omitempty"`
	Playback  string `json:"playback,omitempty"`
	Touch     string `json:"touch,omitempty"`
	Source    string `json:"source,omitempty"`
	YAngle    *int   `json:"y_angle,omitempty"`
	StreamID  string `json:"stream_id,omitempty"`
	Text      string `json:"text,omitempty"`
	Reason    string `json:"reason,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`

	BatteryLevel       *int   `json:"battery_level,omitempty"`
	BatteryCharging    *bool  `json:"battery_charging,omitempty"`
	BatteryDischarging *bool  `json:"battery_discharging,omitempty"`
	ExternalPower      *bool  `json:"external_power,omitempty"`
	PowerSource        string `json:"power_source,omitempty"`
	PMICProfile        string `json:"pmic_power_key_profile,omitempty"`
	PMICPowerStatus    string `json:"pmic_power_status,omitempty"`
}

var inlineDeviceMarkPattern = regexp.MustCompile(`\[(state|face|display|motion):([^\]\s]+)\]`)
var repeatedInlineSpacePattern = regexp.MustCompile(`[ \t]{2,}`)
var safePlaybackStreamIDPattern = regexp.MustCompile(`^a21-[a-z0-9][a-z0-9._-]{0,91}$`)

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
	event := DeviceExtensionEvent{
		Kind:        DeviceEventKind(firstNonEmpty(wire.Kind, wire.Event)),
		Text:        wire.Text,
		Reason:      wire.Reason,
		RuntimeEcho: deviceExtensionRuntimeEchoFromWire(wire),
	}
	switch event.Kind {
	case DeviceEventKindState:
		event.Value = wire.State
	case DeviceEventKindFace:
		event.Value = firstNonEmpty(wire.Face, wire.Emotion)
	case DeviceEventKindDisplay:
		event.Value = firstNonEmpty(wire.Display, wire.Slot)
	case DeviceEventKindMotion:
		event.Value = firstNonEmpty(wire.Motion, wire.Name)
		if wire.YAngle != nil {
			event.YAngle = *wire.YAngle
		}
	case DeviceEventKindHeartbeat:
	case DeviceEventKindPlayback:
		event.Value = wire.Playback
		event.StreamID = wire.StreamID
	case DeviceEventKindTouch:
		event.Value = firstNonEmpty(wire.Touch, wire.Name)
		event.Source = wire.Source
	default:
		event.Value = firstNonEmpty(wire.State, wire.Face, wire.Display, wire.Motion, wire.Playback, wire.Touch)
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
		Kind:        kind,
		Value:       value,
		YAngle:      event.YAngle,
		StreamID:    strings.TrimSpace(event.StreamID),
		Source:      strings.TrimSpace(strings.ToLower(event.Source)),
		Text:        strings.TrimSpace(event.Text),
		Reason:      strings.TrimSpace(event.Reason),
		RuntimeEcho: normalizeDeviceRuntimeEcho(event.RuntimeEcho),
	}
	if containsLegacyIdentity(normalized.StreamID) ||
		containsLegacyIdentity(normalized.Text) ||
		containsLegacyIdentity(normalized.Reason) {
		return DeviceExtensionEvent{}, fmt.Errorf("%w: device extension", ErrLegacyIdentity)
	}
	for key, value := range normalized.RuntimeEcho {
		if containsLegacyIdentity(key) || containsLegacyIdentity(value) {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: device extension", ErrLegacyIdentity)
		}
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
	case DeviceEventKindPlayback:
		if !allowedDeviceEventValue(value, "start", "stop_done") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: playback", ErrUnsupportedDeviceEventValue)
		}
		if !safePlaybackStreamID(normalized.StreamID) {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: playback stream_id", ErrUnsupportedDeviceEventValue)
		}
	case DeviceEventKindTouch:
		if !allowedDeviceEventValue(value, "screen_tap", "screen_barge_in", "top_tap", "top_swipe_forward", "top_swipe_backward", "top_barge_in") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: touch", ErrUnsupportedDeviceEventValue)
		}
		if normalized.Source != "" && !allowedDeviceEventValue(normalized.Source, "screen", "top_sensor") {
			return DeviceExtensionEvent{}, fmt.Errorf("%w: touch source", ErrUnsupportedDeviceEventValue)
		}
	default:
		return DeviceExtensionEvent{}, fmt.Errorf("%w: kind", ErrUnsupportedDeviceEventKind)
	}
	return normalized, nil
}

func deviceExtensionRuntimeEchoFromWire(wire deviceExtensionWire) map[string]string {
	echo := map[string]string{}
	addInt := func(key string, value *int) {
		if value != nil {
			echo[key] = strconv.Itoa(*value)
		}
	}
	addBool := func(key string, value *bool) {
		if value != nil {
			echo[key] = strconv.FormatBool(*value)
		}
	}
	addString := func(key string, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			echo[key] = value
		}
	}
	addInt("battery_level", wire.BatteryLevel)
	addBool("battery_charging", wire.BatteryCharging)
	addBool("battery_discharging", wire.BatteryDischarging)
	addBool("external_power", wire.ExternalPower)
	addString("power_source", wire.PowerSource)
	addString("pmic_power_key_profile", wire.PMICProfile)
	addString("pmic_power_status", wire.PMICPowerStatus)
	if len(echo) == 0 {
		return nil
	}
	return echo
}

func normalizeDeviceRuntimeEcho(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		"battery_level":          {},
		"battery_charging":       {},
		"battery_discharging":    {},
		"external_power":         {},
		"power_source":           {},
		"pmic_power_key_profile": {},
		"pmic_power_status":      {},
	}
	normalized := map[string]string{}
	for key, value := range input {
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			continue
		}
		if len(value) > 96 || !safeDeviceRuntimeEchoValue(value) {
			continue
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func safeDeviceRuntimeEchoValue(value string) bool {
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '_', '-', '.', ':', '+':
			continue
		default:
			return false
		}
	}
	return true
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
	wire.Event = string(event.Kind)
	switch event.Kind {
	case DeviceEventKindState:
		wire.State = event.Value
	case DeviceEventKindFace:
		wire.Emotion = event.Value
	case DeviceEventKindDisplay:
		wire.Slot = event.Value
		wire.Text = event.Text
	case DeviceEventKindMotion:
		wire.Name = event.Value
		wire.Reason = event.Reason
		if event.YAngle != 0 {
			yAngle := event.YAngle
			wire.YAngle = &yAngle
		}
	case DeviceEventKindPlayback:
		wire.Playback = event.Value
		wire.StreamID = event.StreamID
	case DeviceEventKindTouch:
		wire.Touch = event.Value
		wire.Source = event.Source
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

func safePlaybackStreamID(streamID string) bool {
	if streamID == "" {
		return true
	}
	if len(streamID) > 96 || !safePlaybackStreamIDPattern.MatchString(streamID) {
		return false
	}
	lower := strings.ToLower(streamID)
	for _, forbidden := range []string{"secret", "token", "password", "passwd", "credential", "api_key", "apikey", "bearer", "sk-"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}
