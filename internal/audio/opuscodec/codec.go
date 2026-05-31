package opuscodec

import (
	"errors"
	"fmt"

	"github.com/thesyncim/gopus"
)

const MaxOpusPacketBytes = 4000

var (
	ErrUnsupportedConfig = errors.New("unsupported opus codec config")
	ErrInvalidPCMFrame   = errors.New("invalid pcm frame")
	ErrInvalidOpusPacket = errors.New("invalid opus packet")
)

type Codec struct {
	sampleRateHz       int
	channels           int
	durationMS         int
	frameSamples       int
	encodeFrameSamples int
	encoder            *gopus.Encoder
	decoder            *gopus.Decoder
}

func New(sampleRateHz int, channels int, durationMS int) (*Codec, error) {
	if channels != 1 {
		return nil, fmt.Errorf("%w: xiaozhi seam requires mono audio", ErrUnsupportedConfig)
	}
	if durationMS != 60 {
		return nil, fmt.Errorf("%w: xiaozhi seam requires 60ms frames", ErrUnsupportedConfig)
	}
	switch sampleRateHz {
	case 16000, 24000, 48000:
	default:
		return nil, fmt.Errorf("%w: unsupported sample rate %d", ErrUnsupportedConfig, sampleRateHz)
	}
	frameSamples := sampleRateHz * durationMS / 1000
	encodeFrameSamples := 48000 * durationMS / 1000
	// gopus exposes Opus frame size as a 48 kHz-equivalent count. Normalize
	// encode input to 48 kHz so the packet TOC advertises the correct duration.
	encoder, err := gopus.NewEncoder(gopus.EncoderConfig{
		SampleRate:  48000,
		Channels:    channels,
		Application: gopus.ApplicationVoIP,
	})
	if err != nil {
		return nil, err
	}
	if err := encoder.SetFrameSize(encodeFrameSamples); err != nil {
		return nil, err
	}
	decoder, err := gopus.NewDecoder(gopus.DecoderConfig{
		SampleRate:       sampleRateHz,
		Channels:         channels,
		MaxPacketSamples: frameSamples,
		MaxPacketBytes:   MaxOpusPacketBytes,
	})
	if err != nil {
		return nil, err
	}
	return &Codec{
		sampleRateHz:       sampleRateHz,
		channels:           channels,
		durationMS:         durationMS,
		frameSamples:       frameSamples,
		encodeFrameSamples: encodeFrameSamples,
		encoder:            encoder,
		decoder:            decoder,
	}, nil
}

func (c *Codec) FrameSamples() int {
	if c == nil {
		return 0
	}
	return c.frameSamples
}

func (c *Codec) EncodePCM16(pcm []int16) ([]byte, error) {
	if c == nil || c.encoder == nil {
		return nil, fmt.Errorf("%w: nil codec", ErrUnsupportedConfig)
	}
	if len(pcm) != c.frameSamples*c.channels {
		return nil, fmt.Errorf("%w: got %d samples, want %d", ErrInvalidPCMFrame, len(pcm), c.frameSamples*c.channels)
	}
	pcm48, err := c.pcm16To48k(pcm)
	if err != nil {
		return nil, err
	}
	packet := make([]byte, MaxOpusPacketBytes)
	n, err := c.encoder.EncodeInt16(pcm48, packet)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), packet[:n]...), nil
}

func (c *Codec) DecodePCM16(packet []byte) ([]int16, error) {
	if c == nil || c.decoder == nil {
		return nil, fmt.Errorf("%w: nil codec", ErrUnsupportedConfig)
	}
	if len(packet) == 0 {
		return nil, ErrInvalidOpusPacket
	}
	if len(packet) > MaxOpusPacketBytes {
		return nil, fmt.Errorf("%w: packet has %d bytes", ErrInvalidOpusPacket, len(packet))
	}
	pcm := make([]int16, c.frameSamples*c.channels)
	n, err := c.decoder.DecodeInt16(packet, pcm)
	if err != nil {
		return nil, err
	}
	samples := n * c.channels
	if samples != c.frameSamples*c.channels {
		return nil, fmt.Errorf("%w: decoded %d samples, want %d", ErrInvalidOpusPacket, samples, c.frameSamples*c.channels)
	}
	return append([]int16(nil), pcm[:samples]...), nil
}

func (c *Codec) pcm16To48k(pcm []int16) ([]int16, error) {
	if c.sampleRateHz == 48000 {
		return append([]int16(nil), pcm...), nil
	}
	if 48000%c.sampleRateHz != 0 {
		return nil, fmt.Errorf("%w: cannot scale %dHz to 48000Hz", ErrUnsupportedConfig, c.sampleRateHz)
	}
	factor := 48000 / c.sampleRateHz
	out := make([]int16, len(pcm)*factor)
	for i, sample := range pcm {
		next := sample
		if i+1 < len(pcm) {
			next = pcm[i+1]
		}
		base := i * factor
		for step := 0; step < factor; step++ {
			delta := int32(next) - int32(sample)
			out[base+step] = int16(int32(sample) + delta*int32(step)/int32(factor))
		}
	}
	if len(out) != c.encodeFrameSamples*c.channels {
		return nil, fmt.Errorf("%w: scaled frame has %d samples, want %d", ErrInvalidPCMFrame, len(out), c.encodeFrameSamples*c.channels)
	}
	return out, nil
}
