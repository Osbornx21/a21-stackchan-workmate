package audio

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type PCMPlaybackChunk struct {
	SampleRateHz int
	Channels     int
	DurationMS   int
	DataBase64   string
}

func ReadPCM16MonoWAVChunks(path string, durationMS int) ([]PCMPlaybackChunk, error) {
	return ReadPCM16MonoWAVChunksForSampleRate(path, durationMS, 16000)
}

func ReadPCM16MonoWAVChunksForSampleRate(path string, durationMS int, expectedSampleRateHz int) ([]PCMPlaybackChunk, error) {
	if durationMS <= 0 {
		durationMS = 20
	}
	sampleRateHz, pcm, err := readPCM16MonoWAV(path, expectedSampleRateHz)
	if err != nil {
		return nil, err
	}
	bytesPerChunk := sampleRateHz * durationMS * 2 / 1000
	if bytesPerChunk <= 0 {
		return nil, fmt.Errorf("invalid wav chunk duration")
	}
	chunks := make([]PCMPlaybackChunk, 0, (len(pcm)+bytesPerChunk-1)/bytesPerChunk)
	for offset := 0; offset < len(pcm); offset += bytesPerChunk {
		end := offset + bytesPerChunk
		frame := make([]byte, bytesPerChunk)
		if end > len(pcm) {
			copy(frame, pcm[offset:])
		} else {
			copy(frame, pcm[offset:end])
		}
		chunks = append(chunks, PCMPlaybackChunk{
			SampleRateHz: sampleRateHz,
			Channels:     1,
			DurationMS:   durationMS,
			DataBase64:   base64.StdEncoding.EncodeToString(frame),
		})
	}
	return chunks, nil
}

func WritePCM16MonoWAV(path string, sampleRateHz int, pcm []byte) error {
	if sampleRateHz <= 0 {
		return fmt.Errorf("wav sample rate must be positive")
	}
	if len(pcm) == 0 {
		return fmt.Errorf("wav pcm payload is empty")
	}
	if len(pcm)%2 != 0 {
		return fmt.Errorf("wav pcm payload must contain 16-bit samples")
	}
	dataSize := uint32(len(pcm))
	riffSize := uint32(36) + dataSize
	byteRate := uint32(sampleRateHz * 2)
	blockAlign := uint16(2)
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], riffSize)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRateHz))
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)
	data := append(header, pcm...)
	return os.WriteFile(path, data, 0o600)
}

func readPCM16MonoWAV(path string, expectedSampleRateHz int) (int, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	var header [12]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return 0, nil, fmt.Errorf("wav header unreadable: %w", err)
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, nil, fmt.Errorf("wav file is not RIFF/WAVE")
	}

	var sampleRateHz int
	var channels uint16
	var bitsPerSample uint16
	var audioFormat uint16
	var pcm []byte
	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(file, chunkHeader[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return 0, nil, fmt.Errorf("wav chunk header unreadable: %w", err)
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])
		data := make([]byte, chunkSize)
		if _, err := io.ReadFull(file, data); err != nil {
			return 0, nil, fmt.Errorf("wav chunk %s unreadable: %w", chunkID, err)
		}
		if chunkSize%2 == 1 {
			var pad [1]byte
			_, _ = file.Read(pad[:])
		}
		switch chunkID {
		case "fmt ":
			if len(data) < 16 {
				return 0, nil, fmt.Errorf("wav fmt chunk too short")
			}
			audioFormat = binary.LittleEndian.Uint16(data[0:2])
			channels = binary.LittleEndian.Uint16(data[2:4])
			sampleRateHz = int(binary.LittleEndian.Uint32(data[4:8]))
			bitsPerSample = binary.LittleEndian.Uint16(data[14:16])
		case "data":
			pcm = data
		}
	}
	if audioFormat != 1 {
		return 0, nil, fmt.Errorf("wav audio format is not PCM")
	}
	if channels != 1 {
		return 0, nil, fmt.Errorf("wav channels must be mono")
	}
	if expectedSampleRateHz > 0 && sampleRateHz != expectedSampleRateHz {
		return 0, nil, fmt.Errorf("wav sample rate must be %d Hz", expectedSampleRateHz)
	}
	if bitsPerSample != 16 {
		return 0, nil, fmt.Errorf("wav bit depth must be 16")
	}
	if len(pcm) == 0 {
		return 0, nil, fmt.Errorf("wav data chunk is empty")
	}
	return sampleRateHz, pcm, nil
}
