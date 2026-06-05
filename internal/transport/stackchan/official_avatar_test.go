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
	if pitch["angle"] != 850 || pitch["speed"] != 650 {
		t.Fatalf("pitchServo = %#v, want clamped 85deg as 850 units at speed 650", pitch)
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

func TestOfficialActionPlanMapsSemanticStatesDeterministically(t *testing.T) {
	tests := map[string]struct {
		wantPitch int
		wantRGB   string
	}{
		"idle": {
			wantPitch: 450,
			wantRGB:   "soft_idle_semantic_no_rgb_frame",
		},
		"listening": {
			wantPitch: 620,
			wantRGB:   "listening_semantic_no_rgb_frame",
		},
		"thinking": {
			wantPitch: 700,
			wantRGB:   "thinking_semantic_no_rgb_frame",
		},
		"speaking": {
			wantPitch: 580,
			wantRGB:   "speaking_semantic_no_rgb_frame",
		},
		"error": {
			wantPitch: 450,
			wantRGB:   "error_semantic_no_rgb_frame",
		},
	}

	for state, tc := range tests {
		t.Run(state, func(t *testing.T) {
			plan, err := BuildOfficialActionPlan(xiaozhi.DeviceExtensionEvent{
				Kind:  xiaozhi.DeviceEventKindState,
				Value: state,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Packets) != 2 || plan.Metadata.PacketCount != 2 {
				t.Fatalf("packets=%d metadata=%+v, want exactly avatar + motion", len(plan.Packets), plan.Metadata)
			}
			if plan.Packets[0].Type != DataTypeControlAvatar || plan.Packets[1].Type != DataTypeControlMotion {
				t.Fatalf("packet types = %#x %#x, want avatar then motion", plan.Packets[0].Type, plan.Packets[1].Type)
			}
			var motion map[string]map[string]int
			if err := json.Unmarshal(plan.Packets[1].Payload, &motion); err != nil {
				t.Fatal(err)
			}
			if motion["pitchServo"]["angle"] != tc.wantPitch {
				t.Fatalf("pitchServo = %#v, want angle %d", motion["pitchServo"], tc.wantPitch)
			}
			if plan.Metadata.PhysicalAccepted {
				t.Fatalf("metadata = %+v, must not claim physical acceptance", plan.Metadata)
			}
			if plan.Metadata.Surfaces["rgb"] != tc.wantRGB {
				t.Fatalf("rgb metadata = %q, want %q", plan.Metadata.Surfaces["rgb"], tc.wantRGB)
			}
			if plan.Metadata.Surfaces["packet_contract"] != "official_stackchan_binary" {
				t.Fatalf("metadata = %+v, want official packet contract", plan.Metadata)
			}
		})
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

func TestOfficialActionPlanMarksYawCandidateWithoutPhysicalAcceptance(t *testing.T) {
	plan, err := BuildOfficialActionPlan(xiaozhi.DeviceExtensionEvent{
		Kind:  xiaozhi.DeviceEventKindMotion,
		Value: "shake",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Packets) != 1 || plan.Packets[0].Type != DataTypeDanceSequence {
		t.Fatalf("packets = %+v, want one official DanceSequence for shake", plan.Packets)
	}
	if plan.Metadata.PhysicalAccepted {
		t.Fatalf("metadata = %+v, must not claim physical acceptance", plan.Metadata)
	}
	if plan.Metadata.Surfaces["servo_x"] != "servo_x_candidate_yaw_sequence" {
		t.Fatalf("metadata = %+v, want yaw candidate marker", plan.Metadata)
	}
	if plan.Metadata.Surfaces["rgb"] != "unchanged_no_rgb_frame" {
		t.Fatalf("metadata = %+v, want no RGB frame marker", plan.Metadata)
	}

	var sequence []struct {
		YawServo *officialServo `json:"yawServo,omitempty"`
	}
	if err := json.Unmarshal(plan.Packets[0].Payload, &sequence); err != nil {
		t.Fatal(err)
	}
	for _, keyframe := range sequence {
		if keyframe.YawServo == nil {
			continue
		}
		if keyframe.YawServo.Angle < -1280 || keyframe.YawServo.Angle > 1280 {
			t.Fatalf("yaw keyframe = %+v, want clamped -1280..1280", keyframe.YawServo)
		}
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

func TestOfficialActionPlanKeepsUnsupportedFrameClassesOut(t *testing.T) {
	for _, kind := range []xiaozhi.DeviceEventKind{
		xiaozhi.DeviceEventKindHeartbeat,
		xiaozhi.DeviceEventKind("camera"),
		xiaozhi.DeviceEventKind("video"),
		xiaozhi.DeviceEventKind("call"),
	} {
		if _, err := BuildOfficialActionPlan(xiaozhi.DeviceExtensionEvent{Kind: kind}); err == nil {
			t.Fatalf("BuildOfficialActionPlan(%q) succeeded, want unsupported", kind)
		}
	}
}
