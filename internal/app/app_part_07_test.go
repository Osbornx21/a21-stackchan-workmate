package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func productReadinessVoiceChainReadinessReportFixtureJSON(llmProfile string) string {
	llmProfile = firstNonEmpty(strings.TrimSpace(llmProfile), "stepfun")
	findings := []string{"stepfun_selected"}
	if llmProfile != "stepfun" {
		findings = []string{"stepfun_not_selected"}
	}
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{
  "schema_version": "a21.xiaozhi_streaming_provider_readiness.v1",
  "generated_at_ms": 1780335600000,
  "execution_mode": "static_no_execute_provider_capability_gate",
  "product_mode": "dialogue",
  "chain_mode": "dialogue_low_latency",
  "professional_boundary": "v21_adapter_only",
  "selection": {
    "asr_mode": "streaming",
    "asr_profile": "dashscope_qwen_asr_realtime",
    "asr_profile_env": "A21_ASR_LOCAL_PROFILE",
    "llm_profile": "%s",
    "llm_profile_env": "A21_TEXT_STREAM_PROFILE",
    "tts_mode": "streaming",
    "tts_profile": "dashscope_qwen_tts_realtime",
    "tts_profile_env": "A21_TTS_FAST_PROFILE"
  },
  "asr": {
    "profile": "dashscope_qwen_asr_realtime",
    "profile_env": "A21_ASR_LOCAL_PROFILE",
    "required_env": ["A21_DASHSCOPE_API_KEY"],
    "present_env": ["A21_DASHSCOPE_API_KEY"],
    "adapter": "dashscope_realtime_asr_adapter",
    "capability": "streaming_asr_session",
    "ready": true,
    "real_provider": true,
    "streaming": true,
    "uses_mock": false,
    "uses_file_boundary": false,
    "uses_wav_boundary": false,
    "implemented_in_gateway": true
  },
  "llm": {
    "profile": "%s",
    "profile_env": "A21_TEXT_STREAM_PROFILE",
    "selection_role": "launch_policy",
    "required_env": ["A21_LAB_STEPFUN_API_KEY", "A21_STEPFUN_MODEL"],
    "present_env": ["A21_LAB_STEPFUN_API_KEY", "A21_STEPFUN_MODEL"],
    "adapter": "%s",
    "capability": "text_delta_stream",
    "ready": true,
    "real_provider": true,
    "streaming": true,
    "uses_mock": false,
    "uses_file_boundary": false,
    "uses_wav_boundary": false,
    "implemented_in_gateway": true
  },
  "tts": {
    "profile": "dashscope_qwen_tts_realtime",
    "profile_env": "A21_TTS_FAST_PROFILE",
    "required_env": ["A21_DASHSCOPE_API_KEY"],
    "present_env": ["A21_DASHSCOPE_API_KEY"],
    "adapter": "dashscope_realtime_tts_adapter",
    "capability": "incremental_tts_audio_stream",
    "ready": true,
    "real_provider": true,
    "streaming": true,
    "uses_mock": false,
    "uses_file_boundary": false,
    "uses_wav_boundary": false,
    "implemented_in_gateway": true
  },
  "gate_status": "passed",
  "prd_accepted": false,
  "findings": %s,
  "redaction": {
    "payloads_stored": false,
    "credential_values_stored": false,
    "full_urls_stored": false,
    "local_paths_stored": false
  },
  "next_required": []
}`, llmProfile, llmProfile, llmProfile, string(findingsJSON))
}

func productReadinessRoleplayProfileFixtureJSON(mode string) string {
	profile := "a21_roleplay_wry_peer"
	if mode == "unsafe" {
		profile = "secret_roleplay"
	}
	return fmt.Sprintf(`{
  "schema_version": "a21.gateway.roleplay_profile.v1",
  "service": "a21-gateway",
  "selected_roleplay_profile": "%s",
  "selected_scenario": "engineer_pushback",
  "selected_voice_clone_profile": "a21_voice_clone_default",
  "memory": {
    "schema_version": "a21.personality_memory_state.v1",
    "status": "configured",
    "contract_ready": true,
    "configured": true,
    "prompt_input_ready": true,
    "policy": "bounded_prompt_hints",
    "max_items": 6,
    "max_item_chars": 80,
    "user_preference_count": 1,
    "session_memory_count": 0,
    "redaction": {
      "memory_text_stored": false,
      "instruction_text_stored": false,
      "asr_text_stored": false,
      "model_text_stored": false,
      "network_locator_stored": false,
      "filesystem_locator_stored": false,
      "credential_value_stored": false
    }
  },
  "runtime": {
    "schema_version": "a21.roleplay_runtime.v1",
    "mode": "roleplay",
    "roleplay_profile": "%s",
    "scenario": "engineer_pushback",
    "voice_clone_profile": "a21_voice_clone_default",
    "soul_prompt_input_ready": true,
    "prompt_parts": ["role_soul:%s", "scenario:engineer_pushback", "memory:ready"],
    "memory_policy": "bounded_prompt_hints",
    "memory_configured": true,
    "memory_prompt_input_ready": true,
    "memory_hint_count": 1,
    "prompt_composed": true,
    "prompt_stored": false,
    "memory_text_stored": false,
    "transcript_stored": false,
    "provider_output_stored": false,
    "voice_clone_sample_stored": false,
    "professional_route_allowed": false,
    "v21_executed": false
  },
  "expression_plan": {
    "schema_version": "a21.roleplay_expression_plan.v1",
    "adapter": "official_stackchan_action_plan",
    "delivery_policy": "no_send_plan_only",
    "roleplay_profile": "%s",
    "scenario": "engineer_pushback",
    "voice_clone_profile": "a21_voice_clone_default",
    "memory_ready": true,
    "memory_hint_count": 1,
    "action_count": 4,
    "packet_count": 4,
    "physical_accepted": false,
    "actions": [
      {"phase": "baseline_posture", "event": "state", "value": "listening", "packet_count": 1, "physical_accepted": false, "surfaces": {"state": "official_state"}},
      {"phase": "role_soul", "event": "face", "value": "happy", "packet_count": 1, "physical_accepted": false, "surfaces": {"face": "official_avatar_expression"}},
      {"phase": "scenario_emphasis", "event": "state", "value": "thinking", "packet_count": 1, "physical_accepted": false, "surfaces": {"state": "official_state"}},
      {"phase": "memory_cue", "event": "motion", "value": "nod", "packet_count": 1, "physical_accepted": false, "surfaces": {"motion": "official_pitch_sequence"}}
    ],
    "surfaces": {
      "role_soul_face": "official_avatar_expression",
      "memory_cue_motion": "official_pitch_sequence"
    },
    "redaction": {
      "prompt_text_stored": false,
      "memory_text_stored": false,
      "transcript_stored": false,
      "provider_output_stored": false,
      "audio_stored": false,
      "voice_clone_sample_stored": false
    }
  },
  "profiles": [],
  "scenarios": []
}`, profile, profile, profile, profile)
}

func writeProductReadinessRoleplayVoiceReportFixture(t *testing.T, mode string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-roleplay-voice-probe-"+mode+".json")
	if err := os.WriteFile(path, []byte(productReadinessRoleplayVoiceReportFixtureJSON(mode)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessRoleplayVoiceReportFixtureJSON(mode string) string {
	unsafeDetail := ""
	if mode == "unsafe" {
		unsafeDetail = `,
  "unsafe_detail": "raw roleplay prompt text should be rejected before decoding"`
	}
	return `{
  "schema_version": "a21.roleplay_voice_probe.v1",
  "generated_at_ms": 1780549800000,
  "status": "passed",
  "mode": "roleplay",
  "route": "fast_companion_hybrid",
  "execution_mode": "cloud_edge",
  "device_id": "stackchan-sim-001",
  "trace_id": "a21-trace-roleplay-voice-001",
  "session_id": "a21-session-roleplay-voice-001",
  "runtime": {
    "schema_version": "a21.roleplay_runtime.v1",
    "mode": "roleplay",
    "roleplay_profile": "a21_roleplay_wry_peer",
    "scenario": "engineer_pushback",
    "voice_clone_profile": "a21_voice_clone_default",
    "soul_prompt_input_ready": true,
    "prompt_parts": ["role_soul:a21_roleplay_wry_peer", "scenario:engineer_pushback", "memory:ready"],
    "memory_policy": "bounded_prompt_hints",
    "memory_configured": true,
    "memory_prompt_input_ready": true,
    "memory_hint_count": 1,
    "prompt_composed": true,
    "prompt_stored": false,
    "memory_text_stored": false,
    "transcript_stored": false,
    "provider_output_stored": false,
    "voice_clone_sample_stored": false,
    "professional_route_allowed": false,
    "v21_executed": false
  },
  "voice_pipeline": {
    "observed": true,
    "status": "pipeline_completed",
    "text_stream_provider": "stepfun",
    "provider_family": "text_stream",
    "text_stream_executed": true,
    "prompt_input_used": true,
    "voice_clone_profile_used": true,
    "audio_downlink_first_frame_observed": true,
    "device_playback_start_observed": true,
    "audio_chunk_count": 5
  },
  "trace_markers": [
    "roleplay.profile.ready",
    "roleplay.memory.ready",
    "fast_companion.voice_pipeline.start",
    "roleplay.prompt_input.used",
    "roleplay.voice_clone_profile.used",
    "provider.first_content",
    "audio.downlink.first_frame",
    "device.playback.start"
  ],
  "redaction": {
    "prompt_stored": false,
    "memory_text_stored": false,
    "asr_text_stored": false,
    "provider_output_stored": false,
    "audio_stored": false,
    "voice_clone_sample_stored": false,
    "full_urls_stored": false,
    "local_paths_stored": false,
    "credential_values_stored": false
  },
  "physical_accepted": false,
  "prd_accepted": false` + unsafeDetail + `
}`
}

func writeProductReadinessXiaozhiProfessionalGatewayReportFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-xiaozhi-professional-gateway.json")
	data := `{
  "schema_version": "a21.xiaozhi_professional_bench.v1",
  "generated_at_unix_ms": 1780300883954,
  "source_profile": "external_gateway",
  "scenario": "success",
  "acceptance_status": "external_gateway_ready",
  "prd_accepted": false,
  "gateway": "loopback:21080",
  "input_audio": {
    "source": "wav",
    "file": "a21-sherpa-onnx-tts.wav",
    "opus_frame_count": 18
  },
  "trace_id": "a21-trace-xiaozhi-professional-bench-001",
  "session_id": "a21-session-xiaozhi-professional-bench-001",
  "device_id": "stackchan-virtual-a21-professional-bench-001",
  "checking_feedback_observed": true,
  "checking_feedback_ms": 1,
  "checking_feedback_within_1200": true,
  "professional_result_observed": true,
  "professional_result_after_checking": true,
  "abort_stop_observed": true,
  "stale_result_suppressed": true,
  "evidence_count": 5,
  "screen_card_count": 1,
  "follow_up_count": 1,
  "confidence_present": true,
  "no_placeholder_utterance": true,
  "no_asr_text_leak": true,
  "v21_query_first_result_ms": 178,
  "read_record": {
    "observed": true,
    "completed": true,
    "record_count": 1,
    "status": "completed",
    "record_id": "a21-professional-read-000001",
    "trace_id_matched": true,
    "session_id_matched": true,
    "device_id_matched": true,
    "query_scope": "public_only",
    "privacy_scope": "professional_only",
    "latency_profile": "fast_first",
    "answer_style": "voice_first_with_citations",
    "utterance_bucket": "length_17_64",
    "source_scope_counts": {"public": 5},
    "workspace_status": "searchable",
    "redaction": {
      "document_text_stored": false,
      "query_text_stored": false,
      "retrieved_text_stored": false,
      "full_url_stored": false,
      "local_path_stored": false,
      "credential_value_stored": false,
      "provider_output_stored": false,
      "voice_text_stored": false
    }
  },
  "tts_stop_observed": true,
  "failure_count": 0,
  "execution": {
    "provider_executed": false,
    "v21_executed": true,
    "hardware_executed": false,
    "gateway_runtime": "external_gateway"
  },
  "redaction": {
    "payloads_stored": false,
    "asr_text_stored": false,
    "evidence_body_stored": false,
    "full_url_stored": false,
    "local_path_stored": false,
    "prompt_stored": false,
    "provider_output_stored": false
  },
  "report_path": "a21-xiaozhi-professional-gateway.json"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProductReadinessV21AdapterSmokeReportFixture(t *testing.T) string {
	t.Helper()
	return writeProductReadinessV21AdapterSmokeReportFixtureFromData(t, productReadinessV21AdapterSmokeReportFixtureJSON())
}

func writeProductReadinessV21AdapterSmokeReportFixtureFromData(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-v21-adapter-smoke-real.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessV21AdapterSmokeReportFixtureJSON() string {
	return `{
  "schema_version": "a21.v21_adapter_smoke.v1",
  "generated_at_ms": 1780337400000,
  "adapter": "a21-v21-adapter",
  "protocol": "a21_v21_query",
  "status": "passed",
  "configured": true,
  "executed": true,
  "endpoint_host": "127.0.0.1:21121",
  "mode": "professional",
  "query_scope": "public_only",
  "latency_profile": "fast_first",
  "answer_style": "voice_first_with_citations",
  "privacy_scope": "professional_only",
  "max_first_response_ms": 1200,
  "source_scope_counts": {"public": 5},
  "workspace_status": "searchable",
  "query_path": "/a21/v21/query",
  "health_path": "/healthz",
  "duration_ms": 1330.653,
  "confidence": 0.77,
  "evidence_count": 5,
  "speech_block_count": 1,
  "screen_card_count": 1,
  "follow_up_count": 1,
  "redaction_ok": true,
  "report_path": "a21-v21-adapter-smoke-real.json",
  "detail": "v21 adapter query smoke succeeded"
}`
}

func writeProductReadinessPhysicalStackChanReportFixture(t *testing.T, overrides map[string]any) string {
	t.Helper()
	report := map[string]any{
		"schema_version":    "a21.physical_stackchan_evidence.v1",
		"execution_mode":    "physical_stackchan",
		"trace_id":          "a21-trace-physical-stackchan-001",
		"session_id":        "a21-session-physical-stackchan-001",
		"device_id":         "stackchan-001",
		"fixture_path":      "physical-stackchan-fixture.json",
		"promotion_gate":    "candidate",
		"acceptance_status": "physical_review_required",
		"prd_accepted":      false,
		"report_path":       "a21-physical-stackchan-evidence-report.json",
		"execution": map[string]any{
			"provider_executed": false,
			"v21_executed":      false,
			"hardware_executed": true,
		},
		"stage_availability": map[string]any{
			"device.downlink.first_frame":          map[string]any{"available": true, "value_ms": 430, "source": "device_runtime_echo"},
			"device.playback.start":                map[string]any{"available": true, "value_ms": 520, "source": "device_runtime_echo"},
			"speech_end_to_first_audible_response": map[string]any{"available": true, "value_ms": 760, "source": "operator_or_instrument"},
			"barge_in.detected":                    map[string]any{"available": true, "value_ms": 50, "source": "gateway_trace"},
			"barge_in.stop":                        map[string]any{"available": true, "value_ms": 130, "source": "operator_or_instrument"},
			"barge_in.playback_stop_requested":     map[string]any{"available": true, "value_ms": 90, "source": "gateway_trace"},
			"barge_in.playback_stop_done":          map[string]any{"available": true, "value_ms": 130, "source": "device_runtime_echo"},
		},
		"canonical_metrics": map[string]any{
			"device_downlink_first_frame_ms":          map[string]any{"available": true, "value_ms": 430, "source": "device_runtime_echo"},
			"device_playback_start_ms":                map[string]any{"available": true, "value_ms": 520, "source": "device_runtime_echo"},
			"speech_end_to_first_audible_response_ms": map[string]any{"available": true, "value_ms": 760, "source": "operator_or_instrument"},
			"barge_in_detected_ms":                    map[string]any{"available": true, "value_ms": 50, "source": "gateway_trace"},
			"barge_in_stop_ms":                        map[string]any{"available": true, "value_ms": 130, "source": "operator_or_instrument"},
			"barge_in_playback_stop_requested_ms":     map[string]any{"available": true, "value_ms": 90, "source": "gateway_trace"},
			"barge_in_playback_stop_done_ms":          map[string]any{"available": true, "value_ms": 130, "source": "device_runtime_echo"},
		},
		"mic": map[string]any{
			"available":            true,
			"frames_captured":      320,
			"frames_delivered":     318,
			"rms":                  0.13,
			"delivery_ratio":       0.99375,
			"driver_error_count":   0,
			"queue_drop_count":     0,
			"nonzero_sample_count": 4096,
		},
		"observation": map[string]any{
			"available":               true,
			"physical_sound_observed": true,
			"operator_confirmed":      true,
			"method":                  "operator_and_instrument",
			"instrument":              "calibrated_audio_recorder",
			"observed_audible_ms":     760,
			"observed_stop_ms":        130,
		},
		"findings": []map[string]any{{
			"code":     "physical_review_required",
			"severity": "info",
			"message":  "physical metrics are present but still require explicit review before PRD acceptance",
		}},
		"redaction": map[string]any{
			"user_text_stored":             false,
			"instruction_text_stored":      false,
			"model_text_stored":            false,
			"audio_payload_stored":         false,
			"encoded_audio_payload_stored": false,
			"network_locator_stored":       false,
			"network_route_stored":         false,
			"filesystem_locator_stored":    false,
			"secret_material_stored":       false,
			"internal_thought_stored":      false,
		},
	}
	for key, value := range overrides {
		if value == nil {
			delete(report, key)
			continue
		}
		report[key] = value
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a21-physical-stackchan-evidence-report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func productReadinessV21ProfessionalReportFixtureJSON() string {
	return `{
  "schema_version": "a21.v21_professional_readiness.v1",
  "status": "passed",
  "checking_ack_available": true,
  "checking_ack_ms": 4,
  "checking_ack_within_1200": true,
  "evidence_completed_ms": 42,
  "evidence_completed_after_ack": true,
  "evidence_available": true,
  "cards_available": true,
  "follow_ups_available": true,
  "evidence_count": 2,
  "evidence_types": ["meeting", "doc"],
  "card_count": 1,
  "follow_up_count": 2,
  "adapter_configured": true,
  "adapter_executed": false,
  "redaction_ok": true,
  "professional_acceptance_status": "host_mock_ready",
  "report_path": "a21-v21-professional-readiness-host.json"
}`
}

func productReadinessProviderSmokeReportFixtureJSON() string {
	return `{
  "schema_version": "a21.provider_smoke.v1",
  "generated_at_ms": 1780335600000,
  "provider": "deepseek",
  "family": "text_stream",
  "protocol": "openai_chat_completions",
  "status": "passed",
  "configured": true,
  "executed": true,
  "route_eligible": true,
  "stream": true,
  "repeat": 3,
  "http_status": 200,
  "duration_ms": 488.25,
  "attempts": [
    {
      "index": 1,
      "http_status": 200,
      "first_byte_ms": 112.5,
      "first_content_ms": 188.75,
      "total_duration_ms": 488.25,
      "content_delta_count": 2,
      "done": true
    },
    {
      "index": 2,
      "http_status": 200,
      "first_byte_ms": 118.5,
      "first_content_ms": 198.75,
      "total_duration_ms": 492.25,
      "content_delta_count": 2,
      "done": true
    },
    {
      "index": 3,
      "http_status": 200,
      "first_byte_ms": 120.5,
      "first_content_ms": 208.75,
      "total_duration_ms": 500.25,
      "content_delta_count": 2,
      "done": true
    }
  ],
  "timing_summary": {
    "repeat": 3,
    "first_byte_p50_ms": 118.5,
    "first_byte_p95_ms": 140.1,
    "first_byte_p99_ms": 142.1,
    "first_content_p50_ms": 198.75,
    "first_content_p95_ms": 220.2,
    "first_content_p99_ms": 222.2,
    "total_duration_p50_ms": 492.25,
    "total_duration_p95_ms": 510.3,
    "total_duration_p99_ms": 512.3
  },
  "trace_id": "a21-trace-provider-smoke-001",
  "trace_markers": [
    {"name": "provider_first_byte", "value_ms": 112.5},
    {"name": "provider_first_content", "value_ms": 188.75}
  ],
  "metrics": [
    {"name": "a21_provider_first_byte_ms", "value": 112.5},
    {"name": "a21_provider_first_content_ms", "value": 188.75}
  ],
  "network_mode": "direct",
  "endpoint_host": "api.deepseek.com",
  "api_key_env": "A21_LAB_DEEPSEEK_API_KEY",
  "model_env": "A21_DEEPSEEK_MODEL",
  "base_url_env": "A21_DEEPSEEK_BASE_URL",
  "detail": "provider smoke request succeeded",
  "report_path": "a21-provider-smoke-real.json"
}`
}

func productReadinessProviderSmokeReportFixtureForProvider(provider string) string {
	data := productReadinessProviderSmokeReportFixtureJSON()
	switch provider {
	case "stepfun":
		replacements := map[string]string{
			`"provider": "deepseek"`:                        `"provider": "stepfun"`,
			`"endpoint_host": "api.deepseek.com"`:           `"endpoint_host": "api.stepfun.com"`,
			`"api_key_env": "A21_LAB_DEEPSEEK_API_KEY"`:     `"api_key_env": "A21_LAB_STEPFUN_API_KEY"`,
			`"model_env": "A21_DEEPSEEK_MODEL"`:             `"model_env": "A21_STEPFUN_MODEL"`,
			`"base_url_env": "A21_DEEPSEEK_BASE_URL"`:       `"base_url_env": "A21_STEPFUN_BASE_URL"`,
			`"report_path": "a21-provider-smoke-real.json"`: `"report_path": "a21-provider-smoke-stepfun.json"`,
		}
		for old, newValue := range replacements {
			data = strings.ReplaceAll(data, old, newValue)
		}
	case "local_ollama":
		replacements := map[string]string{
			`"provider": "deepseek"`:                        `"provider": "local_ollama"`,
			`"protocol": "openai_chat_completions"`:         `"protocol": "ollama_chat"`,
			`"endpoint_host": "api.deepseek.com"`:           `"endpoint_host": "127.0.0.1:11434"`,
			`"api_key_env": "A21_LAB_DEEPSEEK_API_KEY"`:     `"api_key_env": ""`,
			`"model_env": "A21_DEEPSEEK_MODEL"`:             `"model_env": "A21_LOCAL_OLLAMA_MODEL"`,
			`"base_url_env": "A21_DEEPSEEK_BASE_URL"`:       `"base_url_env": "A21_LOCAL_OLLAMA_BASE_URL"`,
			`"report_path": "a21-provider-smoke-real.json"`: `"report_path": "a21-provider-smoke-local-ollama.json"`,
		}
		for old, newValue := range replacements {
			data = strings.ReplaceAll(data, old, newValue)
		}
	}
	return data
}

func productReadinessRealtimeFixtureReportFixtureJSON() string {
	return `{
  "schema_version": "a21.provider_smoke.v1",
  "generated_at_ms": 1780335605000,
  "provider": "doubao_tts_realtime",
  "family": "voice_hybrid",
  "protocol": "websocket_realtime_fixture",
  "status": "passed",
  "configured": true,
  "executed": true,
  "route_eligible": false,
  "duration_ms": 3.25,
  "network_mode": "direct",
  "endpoint_host": "ai-gateway.vei.volces.com",
  "api_key_env": "A21_DOUBAO_API_KEY",
  "model_env": "A21_DOUBAO_TTS_MODEL",
  "base_url_env": "A21_DOUBAO_TTS_REALTIME_URL",
  "detail": "offline realtime fixture passed; no provider network call performed",
  "report_path": "a21-provider-realtime-fixture-real.json"
}`
}

func productReadinessXiaozhiHostReportFixtureJSON() string {
	return `{
  "schema_version": "a21.xiaozhi_voice_bench.v1",
  "execution_mode": "host_loopback",
  "baseline_scope": "host_only",
  "device_id": "stackchan-virtual-a21-bench-001",
  "repeat": 3,
  "acceptance_status": "candidate_host_only",
  "prd_accepted": false,
  "summary": {
    "answer_first_audio_total_p50_ms": 360,
    "answer_first_audio_total_p95_ms": 386,
    "barge_in_stop_p50_ms": 0,
    "barge_in_stop_p95_ms": 0
  },
  "counts": {
    "answer_turn_count": 3,
    "barge_in_turn_count": 3,
    "failure_count": 0
  },
  "execution": {
    "provider_executed": false,
    "v21_executed": false,
    "hardware_executed": false,
    "voice_pipeline_observed": true,
    "voice_pipeline_execution_mode": "host_local",
    "asr_profile": "sherpa_onnx",
    "asr_profile_env": "A21_ASR_LOCAL_PROFILE",
    "llm_profile": "ollama_local",
    "llm_profile_env": "A21_TEXT_STREAM_PROFILE",
    "tts_profile": "sherpa_onnx_tts",
    "tts_profile_env": "A21_TTS_FAST_PROFILE",
    "host_local_asr_executed": true,
    "host_local_text_executed": true,
    "host_local_tts_executed": true
  },
  "redaction": {
    "payloads_stored": false,
    "credential_values_stored": false,
    "full_urls_stored": false,
    "local_paths_stored": false
  }
}`
}

func TestRunPromotionReadinessBlocksExternalPromotionWithoutTarget(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promotion-readiness"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.promotion_readiness.v1"`,
		`"branch": "codex/a21-integration-governance-slices"`,
		`"review_ready": true`,
		`"external_promotion_ready": false`,
		`"promotion_remote_missing"`,
		`"promotion_target_branch_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunPromotionReadinessBlocksMissingTopicAncestor(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{
		remoteNames:            "origin\n",
		targetBranchConfigured: true,
		missingAncestor:        "6b7fdf0",
	})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promotion-readiness", "--target-remote", "origin", "--target-branch", "codex/a21-mainline-current"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"review_ready": false`,
		`"external_promotion_ready": false`,
		`"promotion_topic_not_ancestor"`,
		`"branch": "codex/a21-provider-spine-deepseek-textstream"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunGateIntegrationWrapsPromotionReadiness(t *testing.T) {
	dir := t.TempDir()
	writePromotionReadinessGitScript(t, dir, promotionGitScriptOptions{})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "integration"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.gate.v1"`,
		`"scope": "integration"`,
		`"promotion_readiness"`,
		`"promotion_remote_missing"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunGateHardwareRequiresCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "hardware"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--command requires a value") {
		t.Fatalf("stderr missing command error: %s", stderr.String())
	}
}

func TestRunGateHardwareWrapsControlGuard(t *testing.T) {
	allowA21ControlGuardForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"gate", "--scope", "hardware", "--command", "provider-smoke --execute"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		`"schema_version": "a21.gate.v1"`,
		`"scope": "hardware"`,
		`"control_guard"`,
		`"command": "provider-smoke --execute"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q: %s", want, stdout.String())
		}
	}
}

func TestRunStackChanAcceptRequiresCheck(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-accept", "--device-id", "stackchan-001"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--check requires a value") {
		t.Fatalf("stderr missing check error: %s", stderr.String())
	}
}

func TestRunStackChanAcceptDispatchesCheckHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-accept", "--check", "touch", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 stackchan-accept --check touch") {
		t.Fatalf("stdout missing touch help: %s", stdout.String())
	}
}

func TestRunDeprecatedStackChanAcceptAliasStillDispatches(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"stackchan-touch-acceptance", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "a21 stackchan-accept --check touch") {
		t.Fatalf("stdout missing touch help: %s", stdout.String())
	}
}
