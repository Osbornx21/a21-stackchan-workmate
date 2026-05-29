package protocol

const ProtocolVersion = "a21.device.v1"

type Kind string

const (
	KindAudioFrame       Kind = "audio.frame"
	KindDeviceEvent      Kind = "device.event"
	KindAssistantState   Kind = "assistant.state"
	KindScreenExpression Kind = "screen.expression"
	KindMotionCommand    Kind = "motion.command"
)

type Envelope struct {
	Protocol string `json:"protocol"`
	DeviceID string `json:"device_id"`
	Kind     Kind   `json:"kind"`
	Seq      uint64 `json:"seq"`
}
