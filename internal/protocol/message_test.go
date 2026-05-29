package protocol

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeJSONUsesA21Protocol(t *testing.T) {
	msg := Envelope{
		Protocol: ProtocolVersion,
		DeviceID: "stackchan-home-01",
		Kind:     KindAudioFrame,
		Seq:      42,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"protocol":"a21.device.v1","device_id":"stackchan-home-01","kind":"audio.frame","seq":42}` {
		t.Fatalf("json = %s", data)
	}
}

func TestMessageKindsCoverCoreDeviceLoop(t *testing.T) {
	tests := map[string]Kind{
		"audio.frame":       KindAudioFrame,
		"device.event":      KindDeviceEvent,
		"assistant.state":   KindAssistantState,
		"screen.expression": KindScreenExpression,
		"motion.command":    KindMotionCommand,
	}

	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("kind = %q, want %q", got, want)
		}
	}
}
