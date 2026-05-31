package audio

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

type FrontEndEvalReport struct {
	SchemaVersion                 string                        `json:"schema_version"`
	Status                        string                        `json:"status"`
	Dataset                       string                        `json:"dataset"`
	Detector                      string                        `json:"detector"`
	TraceID                       string                        `json:"trace_id"`
	SessionID                     string                        `json:"session_id"`
	DeviceID                      string                        `json:"device_id"`
	BaselineScope                 string                        `json:"baseline_scope"`
	ReportPath                    string                        `json:"report_path,omitempty"`
	FramesTotal                   int                           `json:"frames_total"`
	ExpectedSpeechFrames          int                           `json:"expected_speech_frames"`
	DetectedSpeechFrames          int                           `json:"detected_speech_frames"`
	TruePositive                  int                           `json:"true_positive"`
	TrueNegative                  int                           `json:"true_negative"`
	FalsePositive                 int                           `json:"false_positive"`
	FalseNegative                 int                           `json:"false_negative"`
	Accuracy                      float64                       `json:"accuracy"`
	Precision                     float64                       `json:"precision"`
	Recall                        float64                       `json:"recall"`
	SpeechStartEvents             int                           `json:"speech_start_events"`
	SpeechEndEvents               int                           `json:"speech_end_events"`
	SpeechStartLagMS              *int                          `json:"speech_start_lag_ms,omitempty"`
	SpeechEndLagMS                *int                          `json:"speech_end_lag_ms,omitempty"`
	RequiredMetrics               []string                      `json:"required_metrics"`
	CandidateEvidence             []FrontEndCandidate           `json:"candidate_evidence"`
	FastCompanionEvidenceContract []FrontEndEvidenceRequirement `json:"fast_companion_evidence_contract"`
	ProviderExecuted              bool                          `json:"provider_executed"`
	V21Executed                   bool                          `json:"v21_executed"`
	HardwareExecuted              bool                          `json:"hardware_executed"`
	Redaction                     FrontEndEvalRedaction         `json:"redaction"`
	PromotionGate                 string                        `json:"promotion_gate"`
	Notes                         []string                      `json:"notes"`
}

type FrontEndEvalRedaction struct {
	RawAudioStored        bool `json:"raw_audio_stored"`
	Base64AudioStored     bool `json:"base64_audio_stored"`
	TranscriptsStored     bool `json:"transcripts_stored"`
	PromptsStored         bool `json:"prompts_stored"`
	ProviderOutputsStored bool `json:"provider_outputs_stored"`
	ReasoningStored       bool `json:"reasoning_stored"`
	CredentialsStored     bool `json:"credentials_stored"`
	FullURLsStored        bool `json:"full_urls_stored"`
	ProxyURLsStored       bool `json:"proxy_urls_stored"`
	LocalPathsStored      bool `json:"local_paths_stored"`
}

type frontEndEvalFrame struct {
	dataBase64     string
	expectedSpeech bool
	seq            uint64
	sampleRateHz   int
	channels       int
	durationMS     int
}

type FrontEndFixture struct {
	SchemaVersion string                 `json:"schema_version"`
	Dataset       string                 `json:"dataset"`
	Detector      string                 `json:"detector"`
	SampleRateHz  int                    `json:"sample_rate_hz"`
	Channels      int                    `json:"channels"`
	DurationMS    int                    `json:"duration_ms"`
	Frames        []FrontEndFixtureFrame `json:"frames"`
}

type FrontEndFixtureFrame struct {
	Seq            uint64 `json:"seq"`
	ExpectedSpeech bool   `json:"expected_speech"`
	PCMS16LEBase64 string `json:"pcm_s16le_base64"`
}

func RunMockFrontEndEval() FrontEndEvalReport {
	frames := []frontEndEvalFrame{
		{dataBase64: frontEndEvalPCM16Base64(0), expectedSpeech: false, seq: 1, sampleRateHz: 16000, channels: 1, durationMS: 20},
		{dataBase64: frontEndEvalPCM16Base64(12000), expectedSpeech: true, seq: 2, sampleRateHz: 16000, channels: 1, durationMS: 20},
		{dataBase64: frontEndEvalPCM16Base64(12000), expectedSpeech: true, seq: 3, sampleRateHz: 16000, channels: 1, durationMS: 20},
		{dataBase64: frontEndEvalPCM16Base64(0), expectedSpeech: false, seq: 4, sampleRateHz: 16000, channels: 1, durationMS: 20},
		{dataBase64: frontEndEvalPCM16Base64(0), expectedSpeech: false, seq: 5, sampleRateHz: 16000, channels: 1, durationMS: 20},
	}
	return runFrontEndEval(frontEndEvalInput{
		status:        "mock_only",
		dataset:       "a21_mock_vad_fixture_v1",
		detector:      "a21-rms-vad",
		promotionGate: "not_production",
		frames:        frames,
		notes: []string{
			"Deterministic mock fixture only; not a production VAD, AEC, full-duplex, or office-noise claim.",
			"Future WebRTC APM, provider-side VAD, and neural VAD adapters must run through comparable reports before promotion.",
		},
	})
}

func RunFrontEndEvalFromFixture(path string) (FrontEndEvalReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FrontEndEvalReport{}, err
	}
	var fixture FrontEndFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return FrontEndEvalReport{}, err
	}
	if err := validateFrontEndFixture(fixture); err != nil {
		return FrontEndEvalReport{}, err
	}
	frames := make([]frontEndEvalFrame, 0, len(fixture.Frames))
	expectedBytes := fixture.SampleRateHz * fixture.DurationMS / 1000 * fixture.Channels * 2
	for index, frame := range fixture.Frames {
		decoded, err := base64.StdEncoding.DecodeString(frame.PCMS16LEBase64)
		if err != nil {
			return FrontEndEvalReport{}, fmt.Errorf("fixture frame %d pcm_s16le_base64 invalid: %w", index+1, err)
		}
		if len(decoded) != expectedBytes {
			return FrontEndEvalReport{}, fmt.Errorf("fixture frame %d has %d PCM bytes, want %d bytes", index+1, len(decoded), expectedBytes)
		}
		seq := frame.Seq
		if seq == 0 {
			seq = uint64(index + 1)
		}
		frames = append(frames, frontEndEvalFrame{
			dataBase64:     frame.PCMS16LEBase64,
			expectedSpeech: frame.ExpectedSpeech,
			seq:            seq,
			sampleRateHz:   fixture.SampleRateHz,
			channels:       fixture.Channels,
			durationMS:     fixture.DurationMS,
		})
	}
	report := runFrontEndEval(frontEndEvalInput{
		status:        "fixture",
		dataset:       fixture.Dataset,
		detector:      fixture.Detector,
		promotionGate: "requires_recorded_office_and_physical_acceptance",
		frames:        frames,
		notes: []string{
			"Fixture evaluation uses labelled PCM frames; it is stronger than the synthetic mock baseline but still not a full production VAD/AEC claim.",
			"Promotion still requires Shanghai-office recordings, playback-reference echo checks, and physical StackChan acceptance.",
		},
	})
	return report, nil
}

type frontEndEvalInput struct {
	status        string
	dataset       string
	detector      string
	promotionGate string
	frames        []frontEndEvalFrame
	notes         []string
}

func runFrontEndEval(input frontEndEvalInput) FrontEndEvalReport {
	ingress := NewIngress(IngressConfig{
		MaxBufferedFrames: 8,
		SpeechThreshold:   0.02,
		SilenceHangover:   2,
	})
	report := FrontEndEvalReport{
		SchemaVersion:                 "a21.audio.frontend_eval.v1",
		Status:                        input.status,
		Dataset:                       input.dataset,
		Detector:                      input.detector,
		TraceID:                       "a21-trace-front-end-eval",
		SessionID:                     "a21-session-front-end-eval",
		DeviceID:                      "none_host_fixture",
		BaselineScope:                 "host_only",
		FramesTotal:                   len(input.frames),
		CandidateEvidence:             BaselineFrontEndCandidates(),
		FastCompanionEvidenceContract: BaselineFastCompanionEvidenceContract(),
		Redaction:                     FrontEndEvalRedaction{},
		PromotionGate:                 input.promotionGate,
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
		Notes: input.notes,
	}

	var expectedSpeechStartIndex *int
	var expectedSpeechEndIndex *int
	var detectedSpeechStartIndex *int
	var detectedSpeechEndIndex *int
	previousExpectedSpeech := false
	for index, frame := range input.frames {
		if frame.expectedSpeech && !previousExpectedSpeech && expectedSpeechStartIndex == nil {
			expectedSpeechStartIndex = intPointer(index)
		}
		if !frame.expectedSpeech && previousExpectedSpeech && expectedSpeechEndIndex == nil {
			expectedSpeechEndIndex = intPointer(index)
		}
		previousExpectedSpeech = frame.expectedSpeech

		if frame.expectedSpeech {
			report.ExpectedSpeechFrames++
		}
		seq := frame.seq
		if seq == 0 {
			seq = uint64(index + 1)
		}
		result := ingress.Push(Frame{
			DeviceID:     "stackchan-sim-001",
			TraceID:      "a21-trace-front-end-eval",
			SessionID:    "a21-session-front-end-eval",
			Seq:          seq,
			SampleRateHz: frame.sampleRateHz,
			Channels:     frame.channels,
			DurationMS:   frame.durationMS,
			DataBase64:   frame.dataBase64,
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
				if detectedSpeechStartIndex == nil {
					detectedSpeechStartIndex = intPointer(index)
				}
			case EventVADSpeechEnd:
				report.SpeechEndEvents++
				if detectedSpeechEndIndex == nil {
					detectedSpeechEndIndex = intPointer(index)
				}
			}
		}
	}
	if expectedSpeechStartIndex != nil && detectedSpeechStartIndex != nil {
		report.SpeechStartLagMS = intPointer(frameIndexLagMS(input.frames, *expectedSpeechStartIndex, *detectedSpeechStartIndex))
	}
	if expectedSpeechEndIndex != nil && detectedSpeechEndIndex != nil {
		report.SpeechEndLagMS = intPointer(frameIndexLagMS(input.frames, *expectedSpeechEndIndex, *detectedSpeechEndIndex))
	}

	report.Accuracy = roundedRatio(report.TruePositive+report.TrueNegative, report.FramesTotal)
	report.Precision = roundedRatio(report.TruePositive, report.TruePositive+report.FalsePositive)
	report.Recall = roundedRatio(report.TruePositive, report.TruePositive+report.FalseNegative)
	return report
}

func validateFrontEndFixture(fixture FrontEndFixture) error {
	if fixture.SchemaVersion != "a21.audio.frontend_fixture.v1" {
		return fmt.Errorf("fixture schema_version must be a21.audio.frontend_fixture.v1")
	}
	if fixture.Dataset == "" {
		return fmt.Errorf("fixture dataset is required")
	}
	lowerIdentity := strings.ToLower(strings.Join([]string{fixture.Dataset, fixture.Detector}, " "))
	if strings.Contains(lowerIdentity, "x21") || strings.Contains(lowerIdentity, "v21") {
		return fmt.Errorf("fixture identity contains forbidden legacy project name")
	}
	if fixture.Detector == "" {
		return fmt.Errorf("fixture detector is required")
	}
	if fixture.Detector != "a21-rms-vad" {
		return fmt.Errorf("fixture detector %q is not supported yet", fixture.Detector)
	}
	if fixture.SampleRateHz != 16000 {
		return fmt.Errorf("fixture sample_rate_hz must be 16000")
	}
	if fixture.Channels != 1 {
		return fmt.Errorf("fixture channels must be 1")
	}
	if fixture.DurationMS != 20 {
		return fmt.Errorf("fixture duration_ms must be 20")
	}
	if len(fixture.Frames) == 0 {
		return fmt.Errorf("fixture frames are required")
	}
	return nil
}

func roundedRatio(numerator int, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round((float64(numerator)/float64(denominator))*10000) / 10000
}

func frameIndexLagMS(frames []frontEndEvalFrame, expectedIndex int, detectedIndex int) int {
	switch {
	case detectedIndex == expectedIndex:
		return 0
	case detectedIndex > expectedIndex:
		lagMS := 0
		for index := expectedIndex; index < detectedIndex && index < len(frames); index++ {
			lagMS += frames[index].durationMS
		}
		return lagMS
	default:
		lagMS := 0
		for index := detectedIndex; index < expectedIndex && index < len(frames); index++ {
			lagMS += frames[index].durationMS
		}
		return -lagMS
	}
}

func intPointer(value int) *int {
	return &value
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
