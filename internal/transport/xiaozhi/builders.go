package xiaozhi

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

var (
	ErrInvalidMCPEnvelope         = errors.New("invalid mcp json-rpc envelope")
	ErrUnsupportedMCPMethod       = errors.New("unsupported mcp method")
	ErrInvalidMCPToolName         = errors.New("invalid mcp tool name")
	ErrUnsupportedExpressionState = errors.New("unsupported expression state")
)

type MCPMethod string

const (
	MCPMethodInitialize MCPMethod = "initialize"
	MCPMethodToolsList  MCPMethod = "tools/list"
	MCPMethodToolsCall  MCPMethod = "tools/call"
)

type MCPEnvelope struct {
	JSONRPC  string          `json:"jsonrpc"`
	ID       string          `json:"id,omitempty"`
	Method   MCPMethod       `json:"method"`
	Params   json.RawMessage `json:"params,omitempty"`
	ToolName string          `json:"-"`
}

type mcpRequestWire struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Method  MCPMethod `json:"method"`
	Params  any       `json:"params,omitempty"`
}

type mcpParseWire struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  MCPMethod       `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpInitializeParamsWire struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]any    `json:"capabilities"`
	ClientInfo      mcpClientInfoWire `json:"clientInfo"`
}

type mcpClientInfoWire struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpToolsCallParamsWire struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type serverHelloWire struct {
	Type        MessageType     `json:"type"`
	Version     int             `json:"version"`
	Transport   string          `json:"transport"`
	TraceID     string          `json:"trace_id,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	DeviceID    string          `json:"device_id,omitempty"`
	Audio       audioParamsWire `json:"audio"`
	AudioParams audioParamsWire `json:"audio_params"`
}

type llmEmotionWire struct {
	Type      string `json:"type"`
	Emotion   string `json:"emotion"`
	TraceID   string `json:"trace_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
}

func BuildServerHello(identity Identity, binaryProtocolVersion int) ([]byte, error) {
	if binaryProtocolVersion == 0 {
		binaryProtocolVersion = 1
	}
	if !SupportedBinaryProtocolVersion(binaryProtocolVersion) {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedBinaryProtocol, binaryProtocolVersion)
	}
	resolved, err := resolveIdentity(identity, Identity{})
	if err != nil {
		return nil, err
	}
	audio := audioParamsWire{
		Format:        "opus",
		SampleRate:    24000,
		Channels:      1,
		FrameDuration: 60,
	}
	return json.Marshal(serverHelloWire{
		Type:        MessageTypeHello,
		Version:     binaryProtocolVersion,
		Transport:   "websocket",
		TraceID:     resolved.TraceID,
		SessionID:   resolved.SessionID,
		DeviceID:    resolved.DeviceID,
		Audio:       audio,
		AudioParams: audio,
	})
}

func BuildMCPInitializeRequest(id string, clientName string) ([]byte, error) {
	id = strings.TrimSpace(id)
	clientName = strings.TrimSpace(clientName)
	if id == "" {
		return nil, fmt.Errorf("%w: id", ErrInvalidMCPEnvelope)
	}
	if clientName == "" {
		clientName = "a21-xiaozhi-transport"
	}
	if containsLegacyIdentity(id) || containsLegacyIdentity(clientName) {
		return nil, ErrLegacyIdentity
	}
	return json.Marshal(mcpRequestWire{
		JSONRPC: "2.0",
		ID:      numericMCPRequestID(id),
		Method:  MCPMethodInitialize,
		Params: mcpInitializeParamsWire{
			ProtocolVersion: "2024-11-05",
			Capabilities:    map[string]any{},
			ClientInfo: mcpClientInfoWire{
				Name:    clientName,
				Version: "a21-ws5",
			},
		},
	})
}

func BuildMCPToolsListRequest(id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id", ErrInvalidMCPEnvelope)
	}
	if containsLegacyIdentity(id) {
		return nil, ErrLegacyIdentity
	}
	return json.Marshal(mcpRequestWire{
		JSONRPC: "2.0",
		ID:      numericMCPRequestID(id),
		Method:  MCPMethodToolsList,
		Params:  map[string]any{},
	})
}

func BuildMCPToolsCallRequest(id string, toolName string, arguments map[string]any) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id", ErrInvalidMCPEnvelope)
	}
	if containsLegacyIdentity(id) {
		return nil, ErrLegacyIdentity
	}
	toolName = strings.TrimSpace(toolName)
	if err := validateMCPToolName(toolName); err != nil {
		return nil, err
	}
	return json.Marshal(mcpRequestWire{
		JSONRPC: "2.0",
		ID:      numericMCPRequestID(id),
		Method:  MCPMethodToolsCall,
		Params: mcpToolsCallParamsWire{
			Name:      toolName,
			Arguments: redactMCPArguments(arguments),
		},
	})
}

func ParseMCPEnvelope(data []byte) (MCPEnvelope, error) {
	var wire mcpParseWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return MCPEnvelope{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if wire.JSONRPC != "2.0" {
		return MCPEnvelope{}, fmt.Errorf("%w: jsonrpc", ErrInvalidMCPEnvelope)
	}
	id, err := parseMCPWireID(wire.ID)
	if err != nil {
		return MCPEnvelope{}, err
	}
	if containsLegacyIdentity(id) {
		return MCPEnvelope{}, ErrLegacyIdentity
	}
	if !supportedMCPMethod(wire.Method) {
		return MCPEnvelope{}, fmt.Errorf("%w: %q", ErrUnsupportedMCPMethod, wire.Method)
	}
	envelope := MCPEnvelope{
		JSONRPC: wire.JSONRPC,
		ID:      id,
		Method:  wire.Method,
		Params:  append(json.RawMessage(nil), wire.Params...),
	}
	if wire.Method == MCPMethodToolsCall {
		var params mcpToolsCallParamsWire
		if err := json.Unmarshal(wire.Params, &params); err != nil {
			return MCPEnvelope{}, fmt.Errorf("%w: tools/call params", ErrInvalidMCPEnvelope)
		}
		if err := validateMCPToolName(params.Name); err != nil {
			return MCPEnvelope{}, err
		}
		envelope.ToolName = params.Name
	}
	return envelope, nil
}

func numericMCPRequestID(token string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.TrimSpace(token)))
	return int(hash.Sum32()%2147483646) + 1
}

func parseMCPWireID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text), nil
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		if number <= 0 {
			return "", fmt.Errorf("%w: id", ErrInvalidMCPEnvelope)
		}
		return strconv.FormatInt(number, 10), nil
	}
	return "", fmt.Errorf("%w: id", ErrInvalidMCPEnvelope)
}

func BuildLLMEmotionMessage(identity Identity, state string) ([]byte, error) {
	resolved, err := resolveIdentity(identity, Identity{})
	if err != nil {
		return nil, err
	}
	emotion, err := XiaozhiEmotionForA21State(state)
	if err != nil {
		return nil, err
	}
	return json.Marshal(llmEmotionWire{
		Type:      "llm",
		Emotion:   emotion,
		TraceID:   resolved.TraceID,
		SessionID: resolved.SessionID,
		DeviceID:  resolved.DeviceID,
	})
}

func XiaozhiEmotionForA21State(state string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(state)) {
	case "", "idle":
		return "idle", nil
	case "listening":
		return "listening", nil
	case "thinking":
		return "thinking", nil
	case "speaking":
		return "speaking", nil
	case "interrupted":
		return "interrupted", nil
	case "professional":
		return "professional", nil
	case "error":
		return "error", nil
	default:
		if containsLegacyIdentity(state) {
			return "", ErrLegacyIdentity
		}
		return "", fmt.Errorf("%w: %q", ErrUnsupportedExpressionState, state)
	}
}

func ClampYAngle(yAngle int) int {
	if yAngle < 5 {
		return 5
	}
	if yAngle > 85 {
		return 85
	}
	return yAngle
}

func BuildMotionParams(yAngle int) map[string]int {
	return map[string]int{"y_angle": ClampYAngle(yAngle)}
}

func supportedMCPMethod(method MCPMethod) bool {
	switch method {
	case MCPMethodInitialize, MCPMethodToolsList, MCPMethodToolsCall:
		return true
	default:
		return false
	}
}

func validateMCPToolName(toolName string) error {
	if toolName == "" {
		return fmt.Errorf("%w: empty", ErrInvalidMCPToolName)
	}
	if containsLegacyIdentity(toolName) {
		return ErrLegacyIdentity
	}
	for _, r := range toolName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
			continue
		}
		return fmt.Errorf("%w: %q", ErrInvalidMCPToolName, toolName)
	}
	return nil
}

func redactMCPArguments(arguments map[string]any) map[string]any {
	if len(arguments) == 0 {
		return nil
	}
	redacted := make(map[string]any, len(arguments))
	for key, value := range arguments {
		if isSensitiveMCPArgumentKey(key) {
			redacted[key] = "<redacted>"
			continue
		}
		redacted[key] = redactMCPValue(value)
	}
	return redacted
}

func redactMCPValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return redactMCPArguments(typed)
	case []any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = redactMCPValue(item)
		}
		return items
	default:
		return value
	}
}

func isSensitiveMCPArgumentKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, token := range []string{
		"api_key",
		"access_token",
		"token",
		"secret",
		"password",
		"prompt",
		"transcript",
		"provider_output",
		"full_url",
		"url",
		"proxy",
		"local_path",
		"path",
		"audio_base64",
		"data_base64",
		"base64",
		"raw_audio",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return normalized == "raw"
}
