package audio

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestReadPCM16MonoWAVChunksPadsLastFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21.wav")
	writeTestWAV(t, path, 16000, []byte{1, 0, 2, 0, 3, 0})

	chunks, err := ReadPCM16MonoWAVChunks(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1", len(chunks))
	}
	if chunks[0].SampleRateHz != 16000 || chunks[0].Channels != 1 || chunks[0].DurationMS != 20 {
		t.Fatalf("chunk format = %+v", chunks[0])
	}
	decoded, err := base64.StdEncoding.DecodeString(chunks[0].DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 640 {
		t.Fatalf("decoded bytes = %d, want padded 640", len(decoded))
	}
	if decoded[0] != 1 || decoded[2] != 2 || decoded[4] != 3 {
		t.Fatalf("decoded prefix = %v", decoded[:6])
	}
}

func TestReadPCM16MonoWAVChunksRejectsNonA21Format(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21-stereo.wav")
	writeTestWAVHeader(t, path, 16000, 2, 16, []byte{1, 0, 2, 0})

	if _, err := ReadPCM16MonoWAVChunks(path, 20); err == nil {
		t.Fatal("ReadPCM16MonoWAVChunks() error = nil, want stereo rejection")
	}
}

func TestWritePCM16MonoWAVAndRead24KDownlinkChunks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a21-24k.wav")
	pcm := make([]byte, 3000)
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}

	if err := WritePCM16MonoWAV(path, 24000, pcm); err != nil {
		t.Fatal(err)
	}
	chunks, err := ReadPCM16MonoWAVChunksForSampleRate(path, 60, 24000)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2 padded 24k chunks", len(chunks))
	}
	for _, chunk := range chunks {
		if chunk.SampleRateHz != 24000 || chunk.Channels != 1 || chunk.DurationMS != 60 {
			t.Fatalf("chunk format = %+v, want 24k mono 60ms", chunk)
		}
		decoded, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 2880 {
			t.Fatalf("decoded bytes = %d, want 2880", len(decoded))
		}
	}
	if _, err := ReadPCM16MonoWAVChunks(path, 60); err == nil {
		t.Fatal("ReadPCM16MonoWAVChunks() error = nil, want default 16k reader to reject 24k")
	}
}

func writeTestWAV(t *testing.T, path string, sampleRate int, pcm []byte) {
	t.Helper()
	writeTestWAVHeader(t, path, sampleRate, 1, 16, pcm)
}

func writeTestWAVHeader(t *testing.T, path string, sampleRate int, channels int, bitsPerSample int, pcm []byte) {
	t.Helper()
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	data := []byte{
		'R', 'I', 'F', 'F',
		0, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0,
		byte(channels), byte(channels >> 8),
		byte(sampleRate), byte(sampleRate >> 8), byte(sampleRate >> 16), byte(sampleRate >> 24),
		byte(byteRate), byte(byteRate >> 8), byte(byteRate >> 16), byte(byteRate >> 24),
		byte(blockAlign), byte(blockAlign >> 8),
		byte(bitsPerSample), byte(bitsPerSample >> 8),
		'd', 'a', 't', 'a',
		byte(len(pcm)), byte(len(pcm) >> 8), byte(len(pcm) >> 16), byte(len(pcm) >> 24),
	}
	riffSize := uint32(len(data) - 8 + len(pcm))
	data[4] = byte(riffSize)
	data[5] = byte(riffSize >> 8)
	data[6] = byte(riffSize >> 16)
	data[7] = byte(riffSize >> 24)
	data = append(data, pcm...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
