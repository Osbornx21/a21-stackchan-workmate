package audio

import (
	"encoding/base64"
	"math"
)

type FrontEndEvalReport struct {
	Status               string   `json:"status"`
	Dataset              string   `json:"dataset"`
	Detector             string   `json:"detector"`
	FramesTotal          int      `json:"frames_total"`
	ExpectedSpeechFrames int      `json:"expected_speech_frames"`
	DetectedSpeechFrames int      `json:"detected_speech_frames"`
	TruePositive         int      `json:"true_positive"`
	TrueNegative         int      `json:"true_negative"`
	FalsePositive        int      `json:"false_positive"`
	FalseNegative        int      `json:"false_negative"`
	Accuracy             float64  `json:"accuracy"`
	Precision            float64  `json:"precision"`
	Recall               float64  `json:"recall"`
	SpeechStartEvents    int      `json:"speech_start_events"`
	SpeechEndEvents      int      `json:"speech_end_events"`
	RequiredMetrics      []string `json:"required_metrics"`
	PromotionGate        string   `json:"promotion_gate"`
	Notes                []string `json:"notes"`
}

type frontEndEvalFrame struct {
	sample         int16
	expectedSpeech bool
}

func RunMockFrontEndEval() FrontEndEvalReport {
	frames := []frontEndEvalFrame{
		{sample: 0, expectedSpeech: false},
		{sample: 12000, expectedSpeech: true},
		{sample: 12000, expectedSpeech: true},
		{sample: 0, expectedSpeech: false},
		{sample: 0, expectedSpeech: false},
	}
	ingress := NewIngress(IngressConfig{
		MaxBufferedFrames: 8,
		SpeechThreshold:   0.02,
		SilenceHangover:   2,
	})
	report := FrontEndEvalReport{
		Status:        "mock_only",
		Dataset:       "a21_mock_vad_fixture_v1",
		Detector:      "a21-rms-vad",
		FramesTotal:   len(frames),
		PromotionGate: "not_production",
		RequiredMetrics: []string{
			"a21_vad_detector_decisions_total{detector,result}",
			"a21_vad_speech_start_total",
			"a21_vad_speech_end_total",
			"audio_ws_barge_in_stop_ms",
			"speech_start_lag_ms",
			"speech_end_lag_ms",
			"echo_false_barge_in_per_minute",
			"false_start_per_minute",
		},
		Notes: []string{
			"Deterministic mock fixture only; not a production VAD, AEC, full-duplex, or office-noise claim.",
			"Future WebRTC APM, provider-side VAD, and neural VAD adapters must run through comparable reports before promotion.",
		},
	}

	for index, frame := range frames {
		if frame.expectedSpeech {
			report.ExpectedSpeechFrames++
		}
		result := ingress.Push(Frame{
			DeviceID:     "stackchan-sim-001",
			TraceID:      "a21-trace-front-end-eval",
			SessionID:    "a21-session-front-end-eval",
			Seq:          uint64(index + 1),
			SampleRateHz: 16000,
			Channels:     1,
			DurationMS:   20,
			DataBase64:   frontEndEvalPCM16Base64(frame.sample),
		})
		if result.SpeechDetected {
			report.DetectedSpeechFrames++
		}
		switch {
		case frame.expectedSpeech && result.SpeechDetected:
			report.TruePositive++
		case !frame.expectedSpeech && !result.SpeechDetected:
			report.TrueNegative++
		case !frame.expectedSpeech && result.SpeechDetected:
			report.FalsePositive++
		case frame.expectedSpeech && !result.SpeechDetected:
			report.FalseNegative++
		}
		for _, event := range result.Events {
			switch event {
			case EventVADSpeechStart:
				report.SpeechStartEvents++
			case EventVADSpeechEnd:
				report.SpeechEndEvents++
			}
		}
	}

	report.Accuracy = roundedRatio(report.TruePositive+report.TrueNegative, report.FramesTotal)
	report.Precision = roundedRatio(report.TruePositive, report.TruePositive+report.FalsePositive)
	report.Recall = roundedRatio(report.TruePositive, report.TruePositive+report.FalseNegative)
	return report
}

func roundedRatio(numerator int, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round((float64(numerator)/float64(denominator))*10000) / 10000
}

func frontEndEvalPCM16Base64(sample int16) string {
	const samplesPer20MS16K = 320
	data := make([]byte, samplesPer20MS16K*2)
	for i := 0; i < samplesPer20MS16K; i++ {
		data[i*2] = byte(sample)
		data[i*2+1] = byte(uint16(sample) >> 8)
	}
	return base64.StdEncoding.EncodeToString(data)
}
