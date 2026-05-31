package audio

type FrontEndPlan struct {
	Status                        string                        `json:"status"`
	Recommendation                []string                      `json:"recommendation"`
	Guardrails                    []string                      `json:"guardrails"`
	RequiredMetrics               []string                      `json:"required_metrics"`
	FastCompanionEvidenceContract []FrontEndEvidenceRequirement `json:"fast_companion_evidence_contract"`
	Candidates                    []FrontEndCandidate           `json:"candidates"`
}

type FrontEndCandidate struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Role              string   `json:"role"`
	DeploymentTarget  string   `json:"deployment_target"`
	Maturity          string   `json:"maturity"`
	Status            string   `json:"status"`
	Available         bool     `json:"available"`
	Placeholder       bool     `json:"placeholder"`
	PlaceholderReason string   `json:"placeholder_reason"`
	A21Fit            string   `json:"a21_fit"`
	Risks             []string `json:"risks"`
	RequiredEvidence  []string `json:"required_evidence"`
	References        []string `json:"references"`
}

type FrontEndEvidenceRequirement struct {
	ID                string   `json:"id"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	DeploymentTarget  string   `json:"deployment_target"`
	Required          bool     `json:"required"`
	Available         bool     `json:"available"`
	Placeholder       bool     `json:"placeholder"`
	PlaceholderReason string   `json:"placeholder_reason"`
	RequiredEvidence  []string `json:"required_evidence"`
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
		FastCompanionEvidenceContract: BaselineFastCompanionEvidenceContract(),
		Candidates:                    BaselineFrontEndCandidates(),
	}
}

func BaselineFrontEndCandidates() []FrontEndCandidate {
	return []FrontEndCandidate{
		{
			ID:                "webrtc_apm",
			Name:              "WebRTC Audio Processing Module",
			Role:              "aec_noise_suppression_agc_vad_front_end",
			DeploymentTarget:  "gateway_or_desktop_lab_first",
			Maturity:          "mature_voip_audio_processing_stack",
			Status:            "evaluate_first",
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "native_adapter_not_implemented_no_echo_or_full_duplex_evidence",
			A21Fit:            "Best first candidate for AEC and classic real-time speech front-end behavior before claiming full-duplex quality.",
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
				"webrtc_apm_module_docs",
				"webrtc_vad_source_tree",
			},
		},
		{
			ID:                "esp_sr_afe",
			Name:              "ESP-SR AFE/AEC/VAD",
			Role:              "firmware_side_afe_aec_vad_diagnostic",
			DeploymentTarget:  "stackchan_firmware_diagnostic_only",
			Maturity:          "mature_esp32s3_speech_front_end",
			Status:            "planned_unavailable",
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "firmware_diagnostic_adapter_not_implemented_no_flash_or_physical_acceptance",
			A21Fit:            "Best firmware-side candidate to evaluate acoustic echo and wake or VAD behavior without hiding StackChan hardware limits.",
			Risks: []string{
				"Firmware integration needs a separate guarded hardware window.",
				"Diagnostic evidence must not imply release-firmware promotion.",
			},
			RequiredEvidence: []string{
				"Guarded diagnostic build plan before any device write.",
				"Speaker-to-mic echo fixture and physical StackChan report.",
				"CPU, memory, and audio quality notes from the target CoreS3 path.",
			},
			References: []string{
				"esp_sr_esp32s3_guide",
				"esp_sr_aec_guide",
			},
		},
		{
			ID:                "provider_side_vad",
			Name:              "Provider-side realtime turn detection",
			Role:              "turn_detection_and_response_commit",
			DeploymentTarget:  "provider_adapter_only",
			Maturity:          "mature_when_provider_contract_is_explicit",
			Status:            "evaluate_per_provider",
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "provider_runtime_not_executed_no_turn_detection_or_cancel_evidence",
			A21Fit:            "Potentially lowest integration cost for realtime speech providers, but must remain observable through A21 metrics and cancel semantics.",
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
			ID:                "silero_vad",
			Name:              "Silero VAD",
			Role:              "server_side_neural_vad",
			DeploymentTarget:  "gateway_optional_adapter",
			Maturity:          "widely_used_open_source_neural_vad",
			Status:            "evaluate_after_webrtc_apm",
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "model_runtime_not_installed_no_office_noise_cpu_or_memory_evidence",
			A21Fit:            "Good candidate for speech/non-speech quality if Gateway CPU and model packaging remain simple.",
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
				"silero_vad_project",
			},
		},
		{
			ID:                "a21_rms_vad",
			Name:              "A21 deterministic RMS detector",
			Role:              "development_baseline",
			DeploymentTarget:  "gateway_tests_only",
			Maturity:          "simple_test_utility",
			Status:            "dev_only",
			Available:         true,
			Placeholder:       false,
			PlaceholderReason: "deterministic_host_only_baseline_not_production_candidate",
			A21Fit:            "Useful for deterministic transport, trace, metrics, and barge-in tests; not a production voice front end.",
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
	}
}

func BaselineFastCompanionEvidenceContract() []FrontEndEvidenceRequirement {
	return []FrontEndEvidenceRequirement{
		{
			ID:               "mock_benchmark_preservation",
			Description:      "Preserve the deterministic A21 RMS mock benchmark so future adapters prove they did not break the host-only baseline.",
			Status:           "available_host_only",
			DeploymentTarget: "host_fixture",
			Required:         true,
			Available:        true,
			Placeholder:      false,
			RequiredEvidence: []string{"audio-front-end-eval --mock report"},
		},
		{
			ID:               "labelled_fixture_preservation",
			Description:      "Preserve labelled fixture evaluation without storing raw PCM or base64 audio in reports.",
			Status:           "available_host_fixture",
			DeploymentTarget: "host_fixture",
			Required:         true,
			Available:        true,
			Placeholder:      false,
			RequiredEvidence: []string{"audio-front-end-eval --fixture report with fixture basename or redacted metadata only"},
		},
		{
			ID:                "office_noise_benchmark",
			Description:       "Compare candidates against recorded office noise before any production VAD claim.",
			Status:            "unavailable",
			DeploymentTarget:  "gateway_or_desktop_lab",
			Required:          true,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "recorded_office_noise_dataset_not_captured",
			RequiredEvidence:  []string{"Shanghai office noise fixture precision, recall, and false-start report"},
		},
		{
			ID:                "speaker_to_mic_echo_report",
			Description:       "Measure playback echo and false barge-in behavior before AEC or full-duplex acceptance.",
			Status:            "unavailable",
			DeploymentTarget:  "physical_stackchan",
			Required:          true,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "speaker_to_mic_echo_fixture_not_captured",
			RequiredEvidence:  []string{"StackChan speaker-to-mic echo report with playback reference and false barge-in counts"},
		},
		{
			ID:               "speech_start_end_lag",
			Description:      "Track speech start and end lag for mock and labelled fixtures.",
			Status:           "available_host_only",
			DeploymentTarget: "host_fixture",
			Required:         true,
			Available:        true,
			Placeholder:      false,
			RequiredEvidence: []string{"speech_start_lag_ms", "speech_end_lag_ms"},
		},
		{
			ID:                "barge_in_stop_timing",
			Description:       "Measure interrupting speech to playback stop and provider cancel timing.",
			Status:            "placeholder_only",
			DeploymentTarget:  "gateway_then_physical_stackchan",
			Required:          true,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "current_audio_front_end_eval_does_not_execute_playback_or_provider_cancel",
			RequiredEvidence:  []string{"barge_in_stop_ms", "provider_cancel_ms", "playback_stop_ms"},
		},
		{
			ID:                "first_audio_waterfall_impact",
			Description:       "Show candidate impact on first-audio waterfall stages before Fast Companion promotion.",
			Status:            "placeholder_only",
			DeploymentTarget:  "provider_latency_bench_then_stackchan",
			Required:          true,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "no_asr_provider_tts_downlink_or_device_first_audio_execution",
			RequiredEvidence:  []string{"asr_first_partial_ms", "provider_first_content_ms", "tts_first_audio_ms", "audio_downlink_first_frame_ms", "device_playback_start_ms"},
		},
		{
			ID:                "cpu_memory_profile",
			Description:       "Profile candidate CPU and memory on the target Gateway or device before promotion.",
			Status:            "unavailable",
			DeploymentTarget:  "gateway_or_stackchan_target",
			Required:          true,
			Available:         false,
			Placeholder:       true,
			PlaceholderReason: "no_candidate_runtime_profile_collected",
			RequiredEvidence:  []string{"CPU profile", "memory profile", "target machine label"},
		},
		{
			ID:               "metrics_continuity",
			Description:      "Keep detector-labelled metrics continuous as adapters change.",
			Status:           "metric_names_reserved",
			DeploymentTarget: "gateway_metrics",
			Required:         true,
			Available:        true,
			Placeholder:      false,
			RequiredEvidence: []string{"a21_vad_detector_decisions_total{detector,result}", "a21_vad_speech_start_total", "a21_vad_speech_end_total"},
		},
	}
}
