package opuscodec

import (
	"math"
	"testing"
)

func TestCodecRoundTripPCM16Mono60MS(t *testing.T) {
	codec, err := New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm := sinePCM16(16000, 60, 440, 0.45)

	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet) == 0 || len(packet) > MaxOpusPacketBytes {
		t.Fatalf("packet bytes = %d, want 1..%d", len(packet), MaxOpusPacketBytes)
	}

	decoded, err := codec.DecodePCM16(packet)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(pcm) {
		t.Fatalf("decoded samples = %d, want %d", len(decoded), len(pcm))
	}
	if pcm16RMS(decoded) <= 0 {
		t.Fatalf("decoded audio is silent: rms=%f", pcm16RMS(decoded))
	}
}

func TestCodecSupportsXiaozhiServerDownlinkRate(t *testing.T) {
	codec, err := New(24000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]int16, codec.FrameSamples())

	packet, err := codec.EncodePCM16(pcm)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.DecodePCM16(packet)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1440 {
		t.Fatalf("decoded samples = %d, want 1440", len(decoded))
	}
}

func TestCodecRejectsInvalidConfigAndFrames(t *testing.T) {
	if _, err := New(44100, 1, 60); err == nil {
		t.Fatal("New accepted unsupported sample rate")
	}
	if _, err := New(16000, 2, 60); err == nil {
		t.Fatal("New accepted stereo for xiaozhi mono seam")
	}
	if _, err := New(16000, 1, 20); err == nil {
		t.Fatal("New accepted non-xiaozhi frame duration")
	}

	codec, err := New(16000, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.EncodePCM16(make([]int16, codec.FrameSamples()-1)); err == nil {
		t.Fatal("EncodePCM16 accepted wrong frame sample count")
	}
	if _, err := codec.DecodePCM16(nil); err == nil {
		t.Fatal("DecodePCM16 accepted empty packet")
	}
}

func sinePCM16(sampleRate int, durationMS int, frequencyHz float64, amplitude float64) []int16 {
	samples := sampleRate * durationMS / 1000
	pcm := make([]int16, samples)
	for i := range pcm {
		value := math.Sin(2 * math.Pi * frequencyHz * float64(i) / float64(sampleRate))
		pcm[i] = int16(value * amplitude * math.MaxInt16)
	}
	return pcm
}

func pcm16RMS(pcm []int16) float64 {
	if len(pcm) == 0 {
		return 0
	}
	var sum float64
	for _, sample := range pcm {
		normalized := float64(sample) / math.MaxInt16
		sum += normalized * normalized
	}
	return math.Sqrt(sum / float64(len(pcm)))
}
