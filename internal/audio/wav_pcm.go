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
	if durationMS <= 0 {
		durationMS = 20
	}
	sampleRateHz, pcm, err := readPCM16MonoWAV(path)
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

func readPCM16MonoWAV(path string) (int, []byte, error) {
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
	if sampleRateHz != 16000 {
		return 0, nil, fmt.Errorf("wav sample rate must be 16000 Hz")
	}
	if bitsPerSample != 16 {
		return 0, nil, fmt.Errorf("wav bit depth must be 16")
	}
	if len(pcm) == 0 {
		return 0, nil, fmt.Errorf("wav data chunk is empty")
	}
	return sampleRateHz, pcm, nil
}
