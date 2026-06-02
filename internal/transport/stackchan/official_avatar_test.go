package stackchan

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"a21.local/a21/internal/transport/xiaozhi"
)

func TestOfficialPacketBytesUseStackChanBinaryHeader(t *testing.T) {
	packet := OfficialPacket{
		Type:    DataTypeControlMotion,
		Payload: []byte(`{"pitchServo":{"angle":450,"speed":300}}`),
	}

	wire := packet.Bytes()
	if wire[0] != DataTypeControlMotion {
		t.Fatalf("type byte = %#x, want %#x", wire[0], DataTypeControlMotion)
	}
	if got := binary.BigEndian.Uint32(wire[1:5]); got != uint32(len(packet.Payload)) {
		t.Fatalf("length = %d, want %d", got, len(packet.Payload))
	}
	if string(wire[5:]) != string(packet.Payload) {
		t.Fatalf("payload = %s, want %s", string(wire[5:]), string(packet.Payload))
	}
}

func TestOfficialPacketsMapMotionToOfficialControlMotion(t *testing.T) {
	packets, err := BuildOfficialPackets(xiaozhi.DeviceExtensionEvent{
		Kind:   xiaozhi.DeviceEventKindMotion,
		Value:  "look_up",
		YAngle: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1", len(packets))
	}
	if packets[0].Type != DataTypeControlMotion {
		t.Fatalf("packet type = %#x, want ControlMotion", packets[0].Type)
	}

	var payload map[string]map[string]int
	if err := json.Unmarshal(packets[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	pitch := payload["pitchServo"]
	if pitch["angle"] != 850 || pitch["speed"] != 300 {
		t.Fatalf("pitchServo = %#v, want clamped 85deg as 850 units at speed 300", pitch)
	}
}

func TestOfficialPacketsMapFaceToOfficialAvatarFeatures(t *testing.T) {
	packets, err := BuildOfficialPackets(xiaozhi.DeviceExtensionEvent{
		Kind:  xiaozhi.DeviceEventKindFace,
		Value: "speaking",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 1 || packets[0].Type != DataTypeControlAvatar {
		t.Fatalf("packets = %+v, want one ControlAvatar packet", packets)
	}

	var payload map[string]map[string]int
	if err := json.Unmarshal(packets[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"leftEye", "rightEye", "mouth"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload missing official avatar feature %q: %s", key, string(packets[0].Payload))
		}
	}
	if payload["mouth"]["weight"] < 50 {
		t.Fatalf("speaking mouth = %#v, want open official mouth weight", payload["mouth"])
	}
}

func TestOfficialPacketsMapStateToAvatarAndMotion(t *testing.T) {
	packets, err := BuildOfficialPackets(xiaozhi.DeviceExtensionEvent{
		Kind:  xiaozhi.DeviceEventKindState,
		Value: "thinking",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 2 {
		t.Fatalf("packets = %d, want avatar + motion", len(packets))
	}
	if packets[0].Type != DataTypeControlAvatar || packets[1].Type != DataTypeControlMotion {
		t.Fatalf("packet types = %#x %#x, want avatar then motion", packets[0].Type, packets[1].Type)
	}
}

func TestOfficialPacketsMapDanceToOfficialDanceSequence(t *testing.T) {
	packets, err := BuildOfficialPackets(xiaozhi.DeviceExtensionEvent{
		Kind:  xiaozhi.DeviceEventKindMotion,
		Value: "dance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 1 || packets[0].Type != DataTypeDanceSequence {
		t.Fatalf("packets = %+v, want one DanceSequence packet", packets)
	}

	var sequence []map[string]any
	if err := json.Unmarshal(packets[0].Payload, &sequence); err != nil {
		t.Fatal(err)
	}
	if len(sequence) < 3 {
		t.Fatalf("sequence length = %d, want at least 3 keyframes", len(sequence))
	}
}

func TestOfficialPacketsRejectDisplayEventsOutsideAvatarActionAdapter(t *testing.T) {
	_, err := BuildOfficialPackets(xiaozhi.DeviceExtensionEvent{
		Kind:  xiaozhi.DeviceEventKindDisplay,
		Value: "status",
	})
	if err == nil {
		t.Fatal("expected unsupported event error")
	}
}
