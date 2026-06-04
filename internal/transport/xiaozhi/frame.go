package xiaozhi

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMalformedJSON              = errors.New("malformed json")
	ErrMissingDeviceIdentity      = errors.New("missing device identity")
	ErrUnsupportedMessageType     = errors.New("unsupported message type")
	ErrUnsupportedListenState     = errors.New("unsupported listen state")
	ErrUnsupportedHelloVersion    = errors.New("unsupported hello version")
	ErrUnsupportedTransport       = errors.New("unsupported transport")
	ErrUnsupportedAudioParams     = errors.New("unsupported audio params")
	ErrUnsupportedBinaryProtocol  = errors.New("unsupported binary protocol version")
	ErrMalformedBinaryFrame       = errors.New("malformed binary frame")
	ErrUnsupportedBinaryFrameType = errors.New("unsupported binary frame type")
	ErrEmptyBinaryPayload         = errors.New("empty binary payload")
	ErrUnexpectedBinaryDirection  = errors.New("unexpected binary frame direction")
	ErrLegacyIdentity             = errors.New("legacy identity")
)

type Direction string

const (
	DirectionDeviceToServer Direction = "device_to_server"
	DirectionServerToDevice Direction = "server_to_device"
)

type FrameKind string

const (
	FrameKindControl FrameKind = "control"
	FrameKindOpus    FrameKind = "opus"
)

type MessageType string

const (
	MessageTypeHello  MessageType = "hello"
	MessageTypeListen MessageType = "listen"
	MessageTypeAbort  MessageType = "abort"
	MessageTypeMCP    MessageType = "mcp"
)

type Identity struct {
	DeviceID  string
	TraceID   string
	SessionID string
}

type Frame struct {
	Kind      FrameKind
	Direction Direction
	DeviceID  string
	TraceID   string
	SessionID string
	Control   *ControlMessage
	Opus      *OpusFrame
}

type ControlMessage struct {
	Type   MessageType
	Raw    json.RawMessage
	Hello  *HelloMessage
	Listen *ListenMessage
	Abort  *AbortMessage
}

type HelloMessage struct {
	Version     int
	Transport   string
	AudioParams AudioParams
	Features    HelloFeatures
}

type HelloFeatures struct {
	MCP             bool
	AEC             bool
	DeviceEvents    bool
	PlaybackEvents  bool
	KeepaliveEvents bool
	DebugMetrics    bool
}

type AudioParams struct {
	Format                string
	SampleRate            int
	Channels              int
	FrameDuration         int
	BinaryProtocolVersion int
}

type ListenMessage struct {
	State string
	Mode  string
}

type AbortMessage struct {
	Reason string
}

type OpusFrame struct {
	Codec         string
	BinaryVersion int
	TimestampMS   uint32
	PayloadBytes  int
	Payload       []byte
}

type controlWire struct {
	Type      MessageType `json:"type"`
	DeviceID  string      `json:"device_id,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
}

type helloWire struct {
	Type        MessageType       `json:"type"`
	DeviceID    string            `json:"device_id,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Version     int               `json:"version"`
	Transport   string            `json:"transport"`
	Features    helloFeaturesWire `json:"features"`
	Audio       audioParamsWire   `json:"audio"`
	AudioParams audioParamsWire   `json:"audio_params"`
}

type helloFeaturesWire struct {
	MCP             bool `json:"mcp"`
	AEC             bool `json:"aec"`
	DeviceEvents    bool `json:"device_events"`
	PlaybackEvents  bool `json:"playback_events"`
	KeepaliveEvents bool `json:"keepalive_events"`
	DebugMetrics    bool `json:"debug_metrics"`
}

type audioParamsWire struct {
	Format        string `json:"format"`
	SampleRate    int    `json:"sample_rate"`
	Channels      int    `json:"channels"`
	FrameDuration int    `json:"frame_duration"`
	BinaryVersion int    `json:"binary_protocol_version,omitempty"`
}

type listenWire struct {
	Type      MessageType `json:"type"`
	DeviceID  string      `json:"device_id,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	State     string      `json:"state"`
	Mode      string      `json:"mode,omitempty"`
}

type abortWire struct {
	Type      MessageType `json:"type"`
	DeviceID  string      `json:"device_id,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Reason    string      `json:"reason,omitempty"`
}

func ParseTextFrame(data []byte, direction Direction, identity Identity) (Frame, error) {
	var common controlWire
	if err := json.Unmarshal(data, &common); err != nil {
		return Frame{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if !supportedMessageType(common.Type) {
		return Frame{}, fmt.Errorf("%w: %q", ErrUnsupportedMessageType, common.Type)
	}
	resolved, err := resolveIdentity(identity, Identity{
		DeviceID:  common.DeviceID,
		TraceID:   common.TraceID,
		SessionID: common.SessionID,
	})
	if err != nil {
		return Frame{}, err
	}

	control := ControlMessage{
		Type: common.Type,
		Raw:  append(json.RawMessage(nil), data...),
	}
	switch common.Type {
	case MessageTypeHello:
		var msg helloWire
		if err := json.Unmarshal(data, &msg); err != nil {
			return Frame{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
		}
		params := firstAudioParams(msg.AudioParams, msg.Audio)
		hello := &HelloMessage{
			Version:   msg.Version,
			Transport: strings.TrimSpace(msg.Transport),
			Features: HelloFeatures{
				MCP:             msg.Features.MCP,
				AEC:             msg.Features.AEC,
				DeviceEvents:    msg.Features.DeviceEvents,
				PlaybackEvents:  msg.Features.PlaybackEvents,
				KeepaliveEvents: msg.Features.KeepaliveEvents,
				DebugMetrics:    msg.Features.DebugMetrics,
			},
			AudioParams: AudioParams{
				Format:                strings.TrimSpace(params.Format),
				SampleRate:            params.SampleRate,
				Channels:              params.Channels,
				FrameDuration:         params.FrameDuration,
				BinaryProtocolVersion: params.BinaryVersion,
			},
		}
		if err := validateHello(hello); err != nil {
			return Frame{}, err
		}
		control.Hello = &HelloMessage{
			Version:     hello.Version,
			Transport:   hello.Transport,
			AudioParams: hello.AudioParams,
			Features:    hello.Features,
		}
	case MessageTypeListen:
		var msg listenWire
		if err := json.Unmarshal(data, &msg); err != nil {
			return Frame{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
		}
		state := strings.TrimSpace(msg.State)
		if !supportedListenState(state) {
			return Frame{}, fmt.Errorf("%w: %q", ErrUnsupportedListenState, state)
		}
		control.Listen = &ListenMessage{
			State: state,
			Mode:  strings.TrimSpace(msg.Mode),
		}
	case MessageTypeAbort:
		var msg abortWire
		if err := json.Unmarshal(data, &msg); err != nil {
			return Frame{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
		}
		control.Abort = &AbortMessage{
			Reason: strings.TrimSpace(msg.Reason),
		}
	}

	return Frame{
		Kind:      FrameKindControl,
		Direction: direction,
		DeviceID:  resolved.DeviceID,
		TraceID:   resolved.TraceID,
		SessionID: resolved.SessionID,
		Control:   &control,
	}, nil
}

func ParseBinaryFrame(payload []byte, direction Direction, identity Identity) (Frame, error) {
	return ParseBinaryFrameVersion(payload, direction, identity, 1)
}

func ParseBinaryFrameVersion(payload []byte, direction Direction, identity Identity, binaryProtocolVersion int) (Frame, error) {
	if direction != DirectionDeviceToServer && direction != DirectionServerToDevice {
		return Frame{}, fmt.Errorf("%w: %q", ErrUnexpectedBinaryDirection, direction)
	}
	if len(payload) == 0 {
		return Frame{}, ErrEmptyBinaryPayload
	}
	if binaryProtocolVersion == 0 {
		binaryProtocolVersion = 1
	}
	var opusPayload []byte
	var timestampMS uint32
	switch binaryProtocolVersion {
	case 1:
		opusPayload = payload
	case 2:
		var err error
		opusPayload, timestampMS, err = unwrapBinaryProtocol2(payload)
		if err != nil {
			return Frame{}, err
		}
	case 3:
		var err error
		opusPayload, timestampMS, err = unwrapBinaryProtocol3(payload)
		if err != nil {
			return Frame{}, err
		}
	default:
		return Frame{}, fmt.Errorf("%w: %d", ErrUnsupportedBinaryProtocol, binaryProtocolVersion)
	}
	if len(opusPayload) == 0 {
		return Frame{}, ErrEmptyBinaryPayload
	}
	resolved, err := resolveIdentity(identity, Identity{})
	if err != nil {
		return Frame{}, err
	}
	copied := append([]byte(nil), opusPayload...)
	return Frame{
		Kind:      FrameKindOpus,
		Direction: direction,
		DeviceID:  resolved.DeviceID,
		TraceID:   resolved.TraceID,
		SessionID: resolved.SessionID,
		Opus: &OpusFrame{
			Codec:         "opus",
			BinaryVersion: binaryProtocolVersion,
			TimestampMS:   timestampMS,
			PayloadBytes:  len(copied),
			Payload:       copied,
		},
	}, nil
}

func unwrapBinaryProtocol2(payload []byte) ([]byte, uint32, error) {
	const headerBytes = 16
	if len(payload) < headerBytes {
		return nil, 0, fmt.Errorf("%w: v2 header requires %d bytes", ErrMalformedBinaryFrame, headerBytes)
	}
	version := binary.BigEndian.Uint16(payload[0:2])
	if version != 2 {
		return nil, 0, fmt.Errorf("%w: v2 header version %d", ErrUnsupportedBinaryProtocol, version)
	}
	frameType := binary.BigEndian.Uint16(payload[2:4])
	if frameType != 0 {
		return nil, 0, fmt.Errorf("%w: %d", ErrUnsupportedBinaryFrameType, frameType)
	}
	timestampMS := binary.BigEndian.Uint32(payload[8:12])
	payloadSize := int(binary.BigEndian.Uint32(payload[12:16]))
	if payloadSize < 0 || len(payload)-headerBytes != payloadSize {
		return nil, 0, fmt.Errorf("%w: v2 payload size mismatch", ErrMalformedBinaryFrame)
	}
	return payload[headerBytes:], timestampMS, nil
}

func unwrapBinaryProtocol3(payload []byte) ([]byte, uint32, error) {
	const headerBytes = 4
	if len(payload) < headerBytes {
		return nil, 0, fmt.Errorf("%w: v3 header requires %d bytes", ErrMalformedBinaryFrame, headerBytes)
	}
	frameType := payload[0]
	if frameType != 0 {
		return nil, 0, fmt.Errorf("%w: %d", ErrUnsupportedBinaryFrameType, frameType)
	}
	payloadSize := int(binary.BigEndian.Uint16(payload[2:4]))
	if payloadSize < 0 || len(payload)-headerBytes != payloadSize {
		return nil, 0, fmt.Errorf("%w: v3 payload size mismatch", ErrMalformedBinaryFrame)
	}
	return payload[headerBytes:], 0, nil
}

func validateHello(message *HelloMessage) error {
	if message.Version == 0 {
		message.Version = 1
	}
	if !SupportedBinaryProtocolVersion(message.Version) {
		return fmt.Errorf("%w: %d", ErrUnsupportedHelloVersion, message.Version)
	}
	if message.Transport == "" {
		message.Transport = "websocket"
	}
	if message.Transport != "websocket" {
		return fmt.Errorf("%w: %q", ErrUnsupportedTransport, message.Transport)
	}
	if !strings.EqualFold(message.AudioParams.Format, "opus") {
		return fmt.Errorf("%w: format must be opus", ErrUnsupportedAudioParams)
	}
	message.AudioParams.Format = "opus"
	if message.AudioParams.SampleRate != 16000 {
		return fmt.Errorf("%w: sample_rate must be 16000", ErrUnsupportedAudioParams)
	}
	if message.AudioParams.Channels != 1 {
		return fmt.Errorf("%w: channels must be 1", ErrUnsupportedAudioParams)
	}
	if message.AudioParams.FrameDuration != 60 {
		return fmt.Errorf("%w: frame_duration must be 60", ErrUnsupportedAudioParams)
	}
	if message.AudioParams.BinaryProtocolVersion == 0 {
		message.AudioParams.BinaryProtocolVersion = message.Version
	}
	if !SupportedBinaryProtocolVersion(message.AudioParams.BinaryProtocolVersion) {
		return fmt.Errorf("%w: %d", ErrUnsupportedBinaryProtocol, message.AudioParams.BinaryProtocolVersion)
	}
	return nil
}

func SupportedBinaryProtocolVersion(version int) bool {
	switch version {
	case 1, 2, 3:
		return true
	default:
		return false
	}
}

func firstAudioParams(primary audioParamsWire, fallback audioParamsWire) audioParamsWire {
	if primary.Format != "" || primary.SampleRate != 0 || primary.Channels != 0 || primary.FrameDuration != 0 || primary.BinaryVersion != 0 {
		return primary
	}
	return fallback
}

func supportedListenState(state string) bool {
	switch state {
	case "start", "detect", "stop":
		return true
	default:
		return false
	}
}

func supportedMessageType(messageType MessageType) bool {
	switch messageType {
	case MessageTypeHello, MessageTypeListen, MessageTypeAbort, MessageTypeMCP:
		return true
	default:
		return false
	}
}

func resolveIdentity(connection Identity, message Identity) (Identity, error) {
	deviceID := firstNonEmpty(message.DeviceID, connection.DeviceID)
	if deviceID == "" {
		return Identity{}, ErrMissingDeviceIdentity
	}
	if containsLegacyIdentity(deviceID) {
		return Identity{}, fmt.Errorf("%w: device_id", ErrLegacyIdentity)
	}
	traceID := firstNonEmpty(message.TraceID, connection.TraceID)
	if traceID == "" {
		traceID = "a21-trace-" + safeIDComponent(deviceID)
	}
	if containsLegacyIdentity(traceID) {
		return Identity{}, fmt.Errorf("%w: trace_id", ErrLegacyIdentity)
	}
	sessionID := firstNonEmpty(message.SessionID, connection.SessionID)
	if sessionID == "" {
		sessionID = "a21-session-" + safeIDComponent(deviceID)
	}
	if containsLegacyIdentity(sessionID) {
		return Identity{}, fmt.Errorf("%w: session_id", ErrLegacyIdentity)
	}
	return Identity{
		DeviceID:  deviceID,
		TraceID:   traceID,
		SessionID: sessionID,
	}, nil
}

func containsLegacyIdentity(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "x21") || strings.Contains(lower, "v21")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func safeIDComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
		if valid {
			b.WriteRune(r)
			lastDash = r == '-'
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	component := strings.Trim(b.String(), "-._")
	if component == "" {
		return "device"
	}
	return component
}
