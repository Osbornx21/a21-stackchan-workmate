package audio

type FrontEndPlan struct {
	Status          string              `json:"status"`
	Recommendation  []string            `json:"recommendation"`
	Guardrails      []string            `json:"guardrails"`
	RequiredMetrics []string            `json:"required_metrics"`
	Candidates      []FrontEndCandidate `json:"candidates"`
}

type FrontEndCandidate struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	DeploymentTarget string   `json:"deployment_target"`
	Maturity         string   `json:"maturity"`
	Status           string   `json:"status"`
	A21Fit           string   `json:"a21_fit"`
	Risks            []string `json:"risks"`
	RequiredEvidence []string `json:"required_evidence"`
	References       []string `json:"references"`
}

func BaselineFrontEndPlan() FrontEndPlan {
	return FrontEndPlan{
		Status: "plan_only",
		Recommendation: []string{
			"Keep a21_rms_vad as the deterministic development baseline only.",
			"Evaluate WebRTC APM first for Gateway or desktop-side echo cancellation, gain control, noise suppression, and classic voice front-end behavior.",
			"Evaluate provider_side_vad only behind provider adapters because it can be low-latency but opaque.",
			"Evaluate Silero VAD as a server-side neural VAD candidate only after measuring deployment cost and false-start behavior.",
		},
		Guardrails: []string{
			"No candidate may change the A21 firmware audio envelope without an explicit protocol migration.",
			"No candidate may bypass barge-in stop, provider cancel, trace, or metrics contracts.",
			"No candidate is production until it has A21 mock, recorded-office, and physical StackChan acceptance reports.",
			"Detector labels must stay low-cardinality and A21-owned.",
		},
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
		Candidates: []FrontEndCandidate{
			{
				ID:               "webrtc_apm",
				Name:             "WebRTC Audio Processing Module",
				Role:             "aec_noise_suppression_agc_vad_front_end",
				DeploymentTarget: "gateway_or_desktop_lab_first",
				Maturity:         "mature_voip_audio_processing_stack",
				Status:           "evaluate_first",
				A21Fit:           "Best first candidate for AEC and classic real-time speech front-end behavior before claiming full-duplex quality.",
				Risks: []string{
					"Native integration and build complexity can be higher than pure Go.",
					"ESP32 firmware-side integration may be too heavy for the first hardware phase.",
				},
				RequiredEvidence: []string{
					"Gateway benchmark with playback reference enabled.",
					"Office noise recording benchmark.",
					"StackChan speaker-to-mic echo false barge-in report.",
				},
				References: []string{
					"https://webrtc.googlesource.com/src/+/refs/heads/main/modules/audio_processing/g3doc/audio_processing_module.md",
					"https://webrtc.googlesource.com/src/+/main/modules/audio_processing/vad/",
				},
			},
			{
				ID:               "provider_side_vad",
				Name:             "Provider-side realtime turn detection",
				Role:             "turn_detection_and_response_commit",
				DeploymentTarget: "provider_adapter_only",
				Maturity:         "mature_when_provider_contract_is_explicit",
				Status:           "evaluate_per_provider",
				A21Fit:           "Potentially lowest integration cost for realtime speech providers, but must remain observable through A21 metrics and cancel semantics.",
				Risks: []string{
					"Opaque thresholds can make false starts and missed speech harder to diagnose.",
					"Provider-specific behavior must not leak into firmware, protocol, or product mode logic.",
				},
				RequiredEvidence: []string{
					"Provider fixture smoke for append, commit, response, and cancel.",
					"Real-provider first-audio waterfall.",
					"Barge-in stop report with provider cancel timing.",
				},
				References: []string{
					"docs/engineering/PHASE7B_REALTIME_AUDIO_UPLINK.md",
					"docs/engineering/PHASE7D_REALTIME_FIRST_AUDIO.md",
				},
			},
			{
				ID:               "silero_vad",
				Name:             "Silero VAD",
				Role:             "server_side_neural_vad",
				DeploymentTarget: "gateway_optional_adapter",
				Maturity:         "widely_used_open_source_neural_vad",
				Status:           "evaluate_after_webrtc_apm",
				A21Fit:           "Good candidate for speech/non-speech quality if Gateway CPU and model packaging remain simple.",
				Risks: []string{
					"Does not solve acoustic echo cancellation by itself.",
					"Model runtime and packaging must not complicate the Go Gateway without evidence.",
				},
				RequiredEvidence: []string{
					"Recorded-office precision and recall report.",
					"CPU and memory profile on the Gateway machines.",
					"Comparison against WebRTC APM and provider-side detection using the same traces.",
				},
				References: []string{
					"https://github.com/snakers4/silero-vad",
				},
			},
			{
				ID:               "a21_rms_vad",
				Name:             "A21 deterministic RMS detector",
				Role:             "development_baseline",
				DeploymentTarget: "gateway_tests_only",
				Maturity:         "simple_test_utility",
				Status:           "dev_only",
				A21Fit:           "Useful for deterministic transport, trace, metrics, and barge-in tests; not a production voice front end.",
				Risks: []string{
					"Susceptible to noise and echo.",
					"Cannot distinguish user speech from device playback.",
				},
				RequiredEvidence: []string{
					"Must be replaced or supplemented before any production full-duplex claim.",
				},
				References: []string{
					"docs/engineering/PHASE7E_VAD_ADAPTER_BOUNDARY.md",
					"docs/engineering/PHASE7F_VAD_DETECTOR_OBSERVABILITY.md",
				},
			},
		},
	}
}
