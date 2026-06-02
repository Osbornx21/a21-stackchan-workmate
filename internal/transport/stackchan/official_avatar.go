package stackchan

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"a21.local/a21/internal/transport/xiaozhi"
)

const (
	DataTypeControlAvatar byte = 0x03
	DataTypeControlMotion byte = 0x04
	DataTypeDanceSequence byte = 0x14
	DataTypeHeartbeatPing byte = 0x10
	DataTypeHeartbeatPong byte = 0x11
)

var ErrUnsupportedOfficialEvent = errors.New("unsupported official stackchan event")

type OfficialPacket struct {
	Type    byte
	Payload []byte
}

func (packet OfficialPacket) Bytes() []byte {
	wire := make([]byte, 5+len(packet.Payload))
	wire[0] = packet.Type
	binary.BigEndian.PutUint32(wire[1:5], uint32(len(packet.Payload)))
	copy(wire[5:], packet.Payload)
	return wire
}

func BuildOfficialPackets(event xiaozhi.DeviceExtensionEvent) ([]OfficialPacket, error) {
	normalized, err := xiaozhi.NormalizeDeviceExtensionEvent(event)
	if err != nil {
		return nil, err
	}

	switch normalized.Kind {
	case xiaozhi.DeviceEventKindFace:
		packet, err := buildAvatarPacket(normalized.Value)
		if err != nil {
			return nil, err
		}
		return []OfficialPacket{packet}, nil
	case xiaozhi.DeviceEventKindState:
		avatar, err := buildAvatarPacket(faceForState(normalized.Value))
		if err != nil {
			return nil, err
		}
		motion, err := buildMotionPitchPacket(yAngleForState(normalized.Value), 300)
		if err != nil {
			return nil, err
		}
		return []OfficialPacket{avatar, motion}, nil
	case xiaozhi.DeviceEventKindMotion:
		return buildMotionPackets(normalized)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedOfficialEvent, normalized.Kind)
	}
}

type officialFeature struct {
	X        int `json:"x"`
	Y        int `json:"y"`
	Rotation int `json:"rotation"`
	Weight   int `json:"weight"`
	Size     int `json:"size"`
}

type officialAvatarPayload struct {
	LeftEye  officialFeature `json:"leftEye"`
	RightEye officialFeature `json:"rightEye"`
	Mouth    officialFeature `json:"mouth"`
}

type officialServo struct {
	Angle int `json:"angle"`
	Speed int `json:"speed,omitempty"`
}

type officialMotionPayload struct {
	YawServo   *officialServo `json:"yawServo,omitempty"`
	PitchServo *officialServo `json:"pitchServo,omitempty"`
}

type officialDanceKeyframe struct {
	LeftEye    *officialFeature `json:"leftEye,omitempty"`
	RightEye   *officialFeature `json:"rightEye,omitempty"`
	Mouth      *officialFeature `json:"mouth,omitempty"`
	YawServo   *officialServo   `json:"yawServo,omitempty"`
	PitchServo *officialServo   `json:"pitchServo,omitempty"`
	DurationMS int              `json:"durationMs"`
}

func buildAvatarPacket(face string) (OfficialPacket, error) {
	payload, err := json.Marshal(officialAvatarForFace(face))
	if err != nil {
		return OfficialPacket{}, err
	}
	return OfficialPacket{Type: DataTypeControlAvatar, Payload: payload}, nil
}

func buildMotionPackets(event xiaozhi.DeviceExtensionEvent) ([]OfficialPacket, error) {
	switch event.Value {
	case "look_up":
		yAngle := event.YAngle
		if yAngle == 0 {
			yAngle = 38
		}
		packet, err := buildMotionPitchPacket(yAngle, 300)
		if err != nil {
			return nil, err
		}
		return []OfficialPacket{packet}, nil
	case "stop":
		packet, err := buildMotionPitchPacket(45, 0)
		if err != nil {
			return nil, err
		}
		return []OfficialPacket{packet}, nil
	case "nod":
		return []OfficialPacket{buildDancePacket(nodSequence())}, nil
	case "shake":
		return []OfficialPacket{buildDancePacket(shakeSequence())}, nil
	case "dance":
		return []OfficialPacket{buildDancePacket(danceSequence())}, nil
	default:
		return nil, fmt.Errorf("%w: motion", ErrUnsupportedOfficialEvent)
	}
}

func buildMotionPitchPacket(yAngle int, speed int) (OfficialPacket, error) {
	payload, err := json.Marshal(officialMotionPayload{
		PitchServo: &officialServo{
			Angle: xiaozhi.ClampYAngle(yAngle) * 10,
			Speed: speed,
		},
	})
	if err != nil {
		return OfficialPacket{}, err
	}
	return OfficialPacket{Type: DataTypeControlMotion, Payload: payload}, nil
}

func buildDancePacket(sequence []officialDanceKeyframe) OfficialPacket {
	payload, _ := json.Marshal(sequence)
	return OfficialPacket{Type: DataTypeDanceSequence, Payload: payload}
}

func officialAvatarForFace(face string) officialAvatarPayload {
	switch face {
	case "attentive":
		return avatarPayload(100, 10, 0, 0, 0, -18, 0)
	case "thinking":
		return avatarPayload(75, 0, 0, 0, 10, 0, -8)
	case "speaking":
		return avatarPayload(100, 0, 0, 0, 82, 0, 0)
	case "happy":
		return avatarPayload(72, 0, 1550, -1550, 42, 0, 0)
	case "error":
		return avatarPayload(70, 0, -400, 400, 12, 0, 8)
	case "idle":
		fallthrough
	default:
		return avatarPayload(100, 0, 0, 0, 0, 0, 0)
	}
}

func avatarPayload(eyeWeight int, eyeSize int, leftRotation int, rightRotation int, mouthWeight int, featureX int, featureY int) officialAvatarPayload {
	return officialAvatarPayload{
		LeftEye: officialFeature{
			X:        featureX,
			Y:        featureY,
			Rotation: leftRotation,
			Weight:   eyeWeight,
			Size:     eyeSize,
		},
		RightEye: officialFeature{
			X:        featureX,
			Y:        featureY,
			Rotation: rightRotation,
			Weight:   eyeWeight,
			Size:     eyeSize,
		},
		Mouth: officialFeature{
			X:        0,
			Y:        0,
			Rotation: 0,
			Weight:   mouthWeight,
			Size:     0,
		},
	}
}

func faceForState(state string) string {
	switch state {
	case "listening":
		return "attentive"
	case "thinking":
		return "thinking"
	case "speaking":
		return "speaking"
	case "error":
		return "error"
	default:
		return "idle"
	}
}

func yAngleForState(state string) int {
	switch state {
	case "listening":
		return 38
	case "thinking":
		return 52
	case "speaking":
		return 48
	default:
		return 45
	}
}

func nodSequence() []officialDanceKeyframe {
	return []officialDanceKeyframe{
		{PitchServo: &officialServo{Angle: 380, Speed: 450}, DurationMS: 120},
		{PitchServo: &officialServo{Angle: 520, Speed: 450}, DurationMS: 120},
		{PitchServo: &officialServo{Angle: 450, Speed: 350}, DurationMS: 160},
	}
}

func shakeSequence() []officialDanceKeyframe {
	return []officialDanceKeyframe{
		{YawServo: &officialServo{Angle: -120, Speed: 450}, DurationMS: 120},
		{YawServo: &officialServo{Angle: 120, Speed: 450}, DurationMS: 120},
		{YawServo: &officialServo{Angle: 0, Speed: 350}, DurationMS: 160},
	}
}

func danceSequence() []officialDanceKeyframe {
	happy := officialAvatarForFace("happy")
	speaking := officialAvatarForFace("speaking")
	return []officialDanceKeyframe{
		{LeftEye: &happy.LeftEye, RightEye: &happy.RightEye, Mouth: &happy.Mouth, PitchServo: &officialServo{Angle: 420, Speed: 500}, DurationMS: 140},
		{YawServo: &officialServo{Angle: -180, Speed: 600}, PitchServo: &officialServo{Angle: 540, Speed: 500}, DurationMS: 160},
		{LeftEye: &speaking.LeftEye, RightEye: &speaking.RightEye, Mouth: &speaking.Mouth, YawServo: &officialServo{Angle: 180, Speed: 600}, DurationMS: 160},
		{YawServo: &officialServo{Angle: 0, Speed: 450}, PitchServo: &officialServo{Angle: 450, Speed: 350}, DurationMS: 180},
	}
}
