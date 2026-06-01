package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	pcm16FullScale            = 32768.0
	pcm16ClipThreshold        = 32760
	pcm16LowHeadroomThreshold = 29490
	pcm16SilenceThreshold     = 32
)

type PCMQualityReport struct {
	Status         string   `json:"status"`
	Codec          string   `json:"codec"`
	SampleRateHz   int      `json:"sample_rate_hz"`
	Channels       int      `json:"channels"`
	DurationMS     float64  `json:"duration_ms"`
	SampleCount    int      `json:"sample_count"`
	PeakAbs        int      `json:"peak_abs"`
	PeakDBFS       float64  `json:"peak_dbfs"`
	RMS            float64  `json:"rms"`
	RMSDBFS        float64  `json:"rms_dbfs"`
	ClippedSamples int      `json:"clipped_samples,omitempty"`
	ClipRatio      float64  `json:"clip_ratio,omitempty"`
	SilenceRatio   float64  `json:"silence_ratio,omitempty"`
	DCOffset       float64  `json:"dc_offset,omitempty"`
	Findings       []string `json:"findings,omitempty"`
}

func AnalyzePCM16MonoWAVQuality(path string, expectedSampleRateHz int) (PCMQualityReport, error) {
	sampleRateHz, pcm, err := readPCM16MonoWAV(path, expectedSampleRateHz)
	if err != nil {
		return PCMQualityReport{Status: "failed", Codec: "pcm_s16le", SampleRateHz: expectedSampleRateHz, Channels: 1, Findings: []string{"audio_quality_unavailable"}}, err
	}
	return AnalyzePCM16LEQuality("pcm_s16le", sampleRateHz, 1, 0, pcm)
}

func AnalyzePCM16LEQuality(codec string, sampleRateHz int, channels int, chunkDurationMS int, pcm []byte) (PCMQualityReport, error) {
	report := PCMQualityReport{
		Status:       "failed",
		Codec:        strings.ToLower(strings.TrimSpace(codec)),
		SampleRateHz: sampleRateHz,
		Channels:     channels,
	}
	if report.Codec == "" {
		report.Codec = "pcm_s16le"
	}
	if report.Codec != "pcm_s16le" {
		report.Findings = append(report.Findings, "audio_quality_unsupported_codec")
		return report, fmt.Errorf("audio quality requires pcm_s16le")
	}
	if sampleRateHz <= 0 || channels <= 0 {
		report.Findings = append(report.Findings, "audio_quality_invalid_format")
		return report, fmt.Errorf("audio quality requires positive sample rate and channels")
	}
	if len(pcm) == 0 || len(pcm)%2 != 0 {
		report.Findings = append(report.Findings, "audio_quality_invalid_payload")
		return report, fmt.Errorf("audio quality requires non-empty 16-bit PCM")
	}

	samples := len(pcm) / 2
	report.SampleCount = samples / channels
	if chunkDurationMS > 0 {
		report.DurationMS = float64(chunkDurationMS)
	} else {
		report.DurationMS = float64(report.SampleCount) * 1000 / float64(sampleRateHz)
	}

	var sumSquares float64
	var sumSamples float64
	var silentSamples int
	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		abs := absPCM16(sample)
		if abs > report.PeakAbs {
			report.PeakAbs = abs
		}
		if abs >= pcm16ClipThreshold {
			report.ClippedSamples++
		}
		if abs <= pcm16SilenceThreshold {
			silentSamples++
		}
		value := float64(sample)
		sumSamples += value
		sumSquares += value * value
	}

	report.RMS = math.Sqrt(sumSquares/float64(samples)) / pcm16FullScale
	report.RMSDBFS = dbFS(report.RMS)
	report.PeakDBFS = dbFS(float64(report.PeakAbs) / pcm16FullScale)
	report.ClipRatio = float64(report.ClippedSamples) / float64(samples)
	report.SilenceRatio = float64(silentSamples) / float64(samples)
	report.DCOffset = (sumSamples / float64(samples)) / pcm16FullScale

	if report.ClippedSamples > 0 {
		report.Findings = append(report.Findings, "audio_quality_clipping_detected")
	}
	if report.PeakAbs > pcm16LowHeadroomThreshold {
		report.Findings = append(report.Findings, "audio_quality_low_headroom")
	}
	if report.RMSDBFS <= -60 {
		report.Findings = append(report.Findings, "audio_quality_near_silence")
	}
	if math.Abs(report.DCOffset) >= 0.05 {
		report.Findings = append(report.Findings, "audio_quality_dc_offset_detected")
	}

	report.Status = "passed"
	if len(report.Findings) > 0 {
		report.Status = "warning"
	}
	return report, nil
}

func absPCM16(sample int16) int {
	value := int32(sample)
	if value < 0 {
		value = -value
	}
	return int(value)
}

func dbFS(value float64) float64 {
	if value <= 0 {
		return -120
	}
	return math.Round(20*math.Log10(value)*1000) / 1000
}
