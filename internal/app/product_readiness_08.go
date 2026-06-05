package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"a21.local/a21/internal/audio"
	"a21.local/a21/internal/v21adapter"
)

func loadProductV21ProfessionalReportEvidence(path string) (productV21ProfessionalReadiness, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productV21ProfessionalReadiness{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productV21ProfessionalReportContainsForbiddenValue(raw) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if strings.TrimSpace(header.SchemaVersion) == "a21.xiaozhi_professional_bench.v1" {
		return productXiaozhiProfessionalBenchReportEvidence(path, data)
	}
	var fixture productV21ProfessionalReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if missingField := missingProductV21ProfessionalReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21ProfessionalReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != v21adapter.ProfessionalReadinessSchemaVersion ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		!*fixture.CheckingAckWithin1200 ||
		!*fixture.EvidenceAvailable ||
		!*fixture.CardsAvailable ||
		!*fixture.FollowUpsAvailable ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.CardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		*fixture.AdapterExecuted ||
		!*fixture.RedactionOK ||
		strings.TrimSpace(*fixture.ProfessionalAcceptanceStatus) == "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        *fixture.CheckingAckWithin1200,
		EvidenceAvailable:            *fixture.EvidenceAvailable,
		CardsAvailable:               *fixture.CardsAvailable,
		FollowUpsAvailable:           *fixture.FollowUpsAvailable,
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.CardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: strings.TrimSpace(*fixture.ProfessionalAcceptanceStatus),
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              *fixture.AdapterExecuted,
		PRDAccepted:                  false,
	}, nil
}

type productXiaozhiProfessionalBenchReportFixture struct {
	SchemaVersion                   string                                           `json:"schema_version"`
	SourceProfile                   string                                           `json:"source_profile"`
	AcceptanceStatus                string                                           `json:"acceptance_status"`
	PRDAccepted                     bool                                             `json:"prd_accepted"`
	CheckingFeedbackWithin1200      *bool                                            `json:"checking_feedback_within_1200"`
	ProfessionalResultObserved      *bool                                            `json:"professional_result_observed"`
	ProfessionalResultAfterChecking *bool                                            `json:"professional_result_after_checking"`
	EvidenceCount                   *int                                             `json:"evidence_count"`
	ScreenCardCount                 *int                                             `json:"screen_card_count"`
	FollowUpCount                   *int                                             `json:"follow_up_count"`
	ConfidencePresent               *bool                                            `json:"confidence_present"`
	NoPlaceholderUtterance          *bool                                            `json:"no_placeholder_utterance"`
	NoASRTextLeak                   *bool                                            `json:"no_asr_text_leak"`
	FailureCount                    *int                                             `json:"failure_count"`
	ReadRecord                      productXiaozhiProfessionalBenchReadRecordFixture `json:"read_record"`
	Execution                       productXiaozhiProfessionalBenchExecutionFixture  `json:"execution"`
	Redaction                       productXiaozhiProfessionalBenchRedactionFixture  `json:"redaction"`
}

type productXiaozhiProfessionalBenchReadRecordFixture struct {
	Observed          *bool                                              `json:"observed"`
	Completed         *bool                                              `json:"completed"`
	RecordCount       *int                                               `json:"record_count"`
	Status            string                                             `json:"status"`
	RecordID          string                                             `json:"record_id"`
	TraceIDMatched    *bool                                              `json:"trace_id_matched"`
	SessionIDMatched  *bool                                              `json:"session_id_matched"`
	DeviceIDMatched   *bool                                              `json:"device_id_matched"`
	QueryScope        string                                             `json:"query_scope"`
	PrivacyScope      string                                             `json:"privacy_scope"`
	LatencyProfile    string                                             `json:"latency_profile"`
	AnswerStyle       string                                             `json:"answer_style"`
	UtteranceBucket   string                                             `json:"utterance_bucket"`
	SourceScopeCounts map[string]int                                     `json:"source_scope_counts"`
	WorkspaceStatus   string                                             `json:"workspace_status"`
	Redaction         productXiaozhiProfessionalBenchReadRecordRedaction `json:"redaction"`
}

type productXiaozhiProfessionalBenchReadRecordRedaction struct {
	DocumentTextStored    *bool `json:"document_text_stored"`
	QueryTextStored       *bool `json:"query_text_stored"`
	RetrievedTextStored   *bool `json:"retrieved_text_stored"`
	FullURLStored         *bool `json:"full_url_stored"`
	LocalPathStored       *bool `json:"local_path_stored"`
	CredentialValueStored *bool `json:"credential_value_stored"`
	ProviderOutputStored  *bool `json:"provider_output_stored"`
	VoiceTextStored       *bool `json:"voice_text_stored"`
}

type productXiaozhiProfessionalBenchExecutionFixture struct {
	ProviderExecuted *bool  `json:"provider_executed"`
	V21Executed      *bool  `json:"v21_executed"`
	HardwareExecuted *bool  `json:"hardware_executed"`
	GatewayRuntime   string `json:"gateway_runtime"`
}

type productXiaozhiProfessionalBenchRedactionFixture struct {
	PayloadsStored       *bool `json:"payloads_stored"`
	ASRTextStored        *bool `json:"asr_text_stored"`
	EvidenceBodyStored   *bool `json:"evidence_body_stored"`
	FullURLStored        *bool `json:"full_url_stored"`
	LocalPathStored      *bool `json:"local_path_stored"`
	PromptStored         *bool `json:"prompt_stored"`
	ProviderOutputStored *bool `json:"provider_output_stored"`
}

func productXiaozhiProfessionalBenchReportEvidence(path string, data []byte) (productV21ProfessionalReadiness, []productReadinessFinding) {
	var fixture productXiaozhiProfessionalBenchReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	if missingField := missingProductXiaozhiProfessionalBenchReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21ProfessionalReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != "a21.xiaozhi_professional_bench.v1" ||
		strings.TrimSpace(fixture.SourceProfile) != "external_gateway" ||
		strings.TrimSpace(fixture.AcceptanceStatus) != "external_gateway_ready" ||
		fixture.PRDAccepted ||
		!*fixture.CheckingFeedbackWithin1200 ||
		!*fixture.ProfessionalResultObserved ||
		!*fixture.ProfessionalResultAfterChecking ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.ScreenCardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		!*fixture.ConfidencePresent ||
		!*fixture.NoPlaceholderUtterance ||
		!*fixture.NoASRTextLeak ||
		*fixture.FailureCount != 0 ||
		!productXiaozhiProfessionalBenchReadRecordReady(fixture.ReadRecord) ||
		*fixture.Execution.ProviderExecuted ||
		!*fixture.Execution.V21Executed ||
		*fixture.Execution.HardwareExecuted ||
		strings.TrimSpace(fixture.Execution.GatewayRuntime) != "external_gateway" ||
		*fixture.Redaction.PayloadsStored ||
		*fixture.Redaction.ASRTextStored ||
		*fixture.Redaction.EvidenceBodyStored ||
		*fixture.Redaction.FullURLStored ||
		*fixture.Redaction.LocalPathStored ||
		*fixture.Redaction.PromptStored ||
		*fixture.Redaction.ProviderOutputStored {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21ProfessionalReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        *fixture.CheckingFeedbackWithin1200,
		EvidenceAvailable:            *fixture.EvidenceCount > 0,
		CardsAvailable:               *fixture.ScreenCardCount > 0,
		FollowUpsAvailable:           *fixture.FollowUpCount > 0,
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.ScreenCardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: strings.TrimSpace(fixture.AcceptanceStatus),
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              *fixture.Execution.V21Executed,
		ReadRecordObserved:           *fixture.ReadRecord.Observed,
		ReadRecordCompleted:          *fixture.ReadRecord.Completed,
		ReadRecordQueryScope:         strings.TrimSpace(fixture.ReadRecord.QueryScope),
		ReadRecordWorkspaceStatus:    strings.TrimSpace(fixture.ReadRecord.WorkspaceStatus),
		ReadRecordSourceScopeCounts:  copyStringIntMap(fixture.ReadRecord.SourceScopeCounts),
		PRDAccepted:                  false,
	}, nil
}

func productXiaozhiProfessionalBenchReadRecordReady(record productXiaozhiProfessionalBenchReadRecordFixture) bool {
	return record.Observed != nil && *record.Observed &&
		record.Completed != nil && *record.Completed &&
		record.RecordCount != nil && *record.RecordCount == 1 &&
		strings.TrimSpace(record.Status) == "completed" &&
		strings.TrimSpace(record.RecordID) != "" &&
		record.TraceIDMatched != nil && *record.TraceIDMatched &&
		record.SessionIDMatched != nil && *record.SessionIDMatched &&
		record.DeviceIDMatched != nil && *record.DeviceIDMatched &&
		v21adapter.ValidQueryScope(record.QueryScope) &&
		strings.TrimSpace(record.PrivacyScope) == "professional_only" &&
		strings.TrimSpace(record.LatencyProfile) == "fast_first" &&
		strings.TrimSpace(record.AnswerStyle) == "voice_first_with_citations" &&
		validProductV21WorkspaceStatus(record.WorkspaceStatus) &&
		validProductV21SourceScopeCounts(record.SourceScopeCounts) &&
		productXiaozhiProfessionalBenchReadRecordRedactionOK(record.Redaction)
}

func productXiaozhiProfessionalBenchReadRecordRedactionOK(redaction productXiaozhiProfessionalBenchReadRecordRedaction) bool {
	return redaction.DocumentTextStored != nil && !*redaction.DocumentTextStored &&
		redaction.QueryTextStored != nil && !*redaction.QueryTextStored &&
		redaction.RetrievedTextStored != nil && !*redaction.RetrievedTextStored &&
		redaction.FullURLStored != nil && !*redaction.FullURLStored &&
		redaction.LocalPathStored != nil && !*redaction.LocalPathStored &&
		redaction.CredentialValueStored != nil && !*redaction.CredentialValueStored &&
		redaction.ProviderOutputStored != nil && !*redaction.ProviderOutputStored &&
		redaction.VoiceTextStored != nil && !*redaction.VoiceTextStored
}

func missingProductXiaozhiProfessionalBenchReportField(fixture productXiaozhiProfessionalBenchReportFixture) string {
	switch {
	case fixture.CheckingFeedbackWithin1200 == nil:
		return "checking_feedback_within_1200"
	case fixture.ProfessionalResultObserved == nil:
		return "professional_result_observed"
	case fixture.ProfessionalResultAfterChecking == nil:
		return "professional_result_after_checking"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.ScreenCardCount == nil:
		return "screen_card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.ConfidencePresent == nil:
		return "confidence_present"
	case fixture.NoPlaceholderUtterance == nil:
		return "no_placeholder_utterance"
	case fixture.NoASRTextLeak == nil:
		return "no_asr_text_leak"
	case fixture.FailureCount == nil:
		return "failure_count"
	case fixture.ReadRecord.Observed == nil:
		return "read_record.observed"
	case fixture.ReadRecord.Completed == nil:
		return "read_record.completed"
	case fixture.ReadRecord.RecordCount == nil:
		return "read_record.record_count"
	case fixture.ReadRecord.TraceIDMatched == nil:
		return "read_record.trace_id_matched"
	case fixture.ReadRecord.SessionIDMatched == nil:
		return "read_record.session_id_matched"
	case fixture.ReadRecord.DeviceIDMatched == nil:
		return "read_record.device_id_matched"
	case fixture.ReadRecord.Redaction.DocumentTextStored == nil:
		return "read_record.redaction.document_text_stored"
	case fixture.ReadRecord.Redaction.QueryTextStored == nil:
		return "read_record.redaction.query_text_stored"
	case fixture.ReadRecord.Redaction.RetrievedTextStored == nil:
		return "read_record.redaction.retrieved_text_stored"
	case fixture.ReadRecord.Redaction.FullURLStored == nil:
		return "read_record.redaction.full_url_stored"
	case fixture.ReadRecord.Redaction.LocalPathStored == nil:
		return "read_record.redaction.local_path_stored"
	case fixture.ReadRecord.Redaction.CredentialValueStored == nil:
		return "read_record.redaction.credential_value_stored"
	case fixture.ReadRecord.Redaction.ProviderOutputStored == nil:
		return "read_record.redaction.provider_output_stored"
	case fixture.ReadRecord.Redaction.VoiceTextStored == nil:
		return "read_record.redaction.voice_text_stored"
	case fixture.Execution.ProviderExecuted == nil:
		return "execution.provider_executed"
	case fixture.Execution.V21Executed == nil:
		return "execution.v21_executed"
	case fixture.Execution.HardwareExecuted == nil:
		return "execution.hardware_executed"
	case fixture.Redaction.PayloadsStored == nil:
		return "redaction.payloads_stored"
	case fixture.Redaction.ASRTextStored == nil:
		return "redaction.asr_text_stored"
	case fixture.Redaction.EvidenceBodyStored == nil:
		return "redaction.evidence_body_stored"
	case fixture.Redaction.FullURLStored == nil:
		return "redaction.full_url_stored"
	case fixture.Redaction.LocalPathStored == nil:
		return "redaction.local_path_stored"
	case fixture.Redaction.PromptStored == nil:
		return "redaction.prompt_stored"
	case fixture.Redaction.ProviderOutputStored == nil:
		return "redaction.provider_output_stored"
	default:
		return ""
	}
}

func missingProductV21ProfessionalReportField(fixture productV21ProfessionalReportFixture) string {
	switch {
	case fixture.CheckingAckWithin1200 == nil:
		return "checking_ack_within_1200"
	case fixture.EvidenceAvailable == nil:
		return "evidence_available"
	case fixture.CardsAvailable == nil:
		return "cards_available"
	case fixture.FollowUpsAvailable == nil:
		return "follow_ups_available"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.CardCount == nil:
		return "card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.AdapterExecuted == nil:
		return "adapter_executed"
	case fixture.RedactionOK == nil:
		return "redaction_ok"
	case fixture.ProfessionalAcceptanceStatus == nil:
		return "professional_acceptance_status"
	default:
		return ""
	}
}

func missingProductV21ProfessionalReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_professional_report_missing_field",
		Message: "V21 professional readiness report is missing a required field",
		Detail:  field,
	}
}

func invalidProductV21ProfessionalReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_professional_report_invalid",
		Message: "V21 professional readiness report is invalid or unsafe",
	}
}

func productV21ProfessionalReportContainsForbiddenValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			if productV21ProfessionalReportContainsForbiddenValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if productV21ProfessionalReportContainsForbiddenValue(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(typed)
		for _, forbidden := range []string{
			"professional readiness fixture query",
			"raw evidence text",
			"raw retrieved evidence",
			"raw prompt text",
			"raw transcript text",
			"raw provider output",
			"provider output",
			"raw reasoning text",
			"http://",
			"https://",
			"/users/",
			"api_key",
			"secret-token",
		} {
			if strings.Contains(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

type productV21AdapterSmokeReportFixture struct {
	SchemaVersion      string         `json:"schema_version"`
	GeneratedAtMS      *int64         `json:"generated_at_ms"`
	Adapter            string         `json:"adapter"`
	Protocol           string         `json:"protocol"`
	Status             string         `json:"status"`
	Configured         *bool          `json:"configured"`
	Executed           *bool          `json:"executed"`
	Mode               string         `json:"mode"`
	QueryScope         string         `json:"query_scope"`
	LatencyProfile     string         `json:"latency_profile"`
	AnswerStyle        string         `json:"answer_style"`
	PrivacyScope       string         `json:"privacy_scope"`
	MaxFirstResponseMS *int           `json:"max_first_response_ms"`
	WorkspaceStatus    string         `json:"workspace_status"`
	SourceScopeCounts  map[string]int `json:"source_scope_counts"`
	QueryPath          string         `json:"query_path"`
	HealthPath         string         `json:"health_path"`
	EvidenceCount      *int           `json:"evidence_count"`
	SpeechBlockCount   *int           `json:"speech_block_count"`
	ScreenCardCount    *int           `json:"screen_card_count"`
	FollowUpCount      *int           `json:"follow_up_count"`
	Confidence         *float64       `json:"confidence"`
	RedactionOK        *bool          `json:"redaction_ok"`
}

func loadProductV21AdapterSmokeReportEvidence(path string, v21 productV21Readiness) (productV21ProfessionalReadiness, []productReadinessFinding) {
	path = strings.TrimSpace(path)
	if path == "" {
		return productV21ProfessionalReadiness{}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > providerLatencyFixtureSidecarMaxBytes {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	var raw any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if providerLatencyFixtureContainsForbiddenKey(raw) || productV21ProfessionalReportContainsForbiddenValue(raw) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	var fixture productV21AdapterSmokeReportFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if missingField := missingProductV21AdapterSmokeReportField(fixture); missingField != "" {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{missingProductV21AdapterSmokeReportFieldFinding(missingField)}
	}
	if fixture.SchemaVersion != v21adapter.SmokeSchemaVersion ||
		fixture.GeneratedAtMS == nil ||
		*fixture.GeneratedAtMS <= 0 ||
		fixture.Adapter != "a21-v21-adapter" ||
		fixture.Protocol != "a21_v21_query" ||
		strings.TrimSpace(fixture.Status) != "passed" ||
		!*fixture.Configured ||
		!*fixture.Executed ||
		strings.TrimSpace(fixture.Mode) != "professional" ||
		strings.TrimSpace(fixture.LatencyProfile) != "fast_first" ||
		strings.TrimSpace(fixture.AnswerStyle) != "voice_first_with_citations" ||
		strings.TrimSpace(fixture.PrivacyScope) != "professional_only" ||
		*fixture.MaxFirstResponseMS != v21adapter.ProfessionalMaxFirstResponseMS ||
		fixture.QueryPath != v21adapter.QueryPath ||
		fixture.HealthPath != v21adapter.HealthPath ||
		*fixture.Confidence <= 0 ||
		*fixture.Confidence > 1 ||
		*fixture.EvidenceCount <= 0 ||
		*fixture.SpeechBlockCount <= 0 ||
		*fixture.ScreenCardCount <= 0 ||
		*fixture.FollowUpCount <= 0 ||
		!*fixture.RedactionOK {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	queryScope := strings.TrimSpace(fixture.QueryScope)
	if queryScope != "" && !v21adapter.ValidQueryScope(queryScope) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	workspaceStatus := strings.TrimSpace(fixture.WorkspaceStatus)
	if workspaceStatus != "" && !validProductV21WorkspaceStatus(workspaceStatus) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	if len(fixture.SourceScopeCounts) > 0 && !validProductV21SourceScopeCounts(fixture.SourceScopeCounts) {
		return productV21ProfessionalReadiness{}, []productReadinessFinding{invalidProductV21AdapterSmokeReportFinding()}
	}
	return productV21ProfessionalReadiness{
		Valid:                        true,
		CheckingAckWithin1200:        v21.CheckingFeedbackSupported && v21.MaxFirstResponseMS <= v21adapter.ProfessionalMaxFirstResponseMS,
		EvidenceAvailable:            *fixture.EvidenceCount > 0,
		CardsAvailable:               *fixture.ScreenCardCount > 0,
		FollowUpsAvailable:           *fixture.FollowUpCount > 0,
		QueryScope:                   queryScope,
		WorkspaceStatus:              workspaceStatus,
		SourceScopeCounts:            copyStringIntMap(fixture.SourceScopeCounts),
		EvidenceCount:                *fixture.EvidenceCount,
		CardCount:                    *fixture.ScreenCardCount,
		FollowUpCount:                *fixture.FollowUpCount,
		ProfessionalAcceptanceStatus: "adapter_smoke_passed",
		SourceReport:                 filepath.Base(filepath.Clean(path)),
		AdapterExecuted:              true,
		PRDAccepted:                  false,
	}, nil
}

func missingProductV21AdapterSmokeReportField(fixture productV21AdapterSmokeReportFixture) string {
	switch {
	case fixture.GeneratedAtMS == nil:
		return "generated_at_ms"
	case fixture.Configured == nil:
		return "configured"
	case fixture.Executed == nil:
		return "executed"
	case strings.TrimSpace(fixture.Mode) == "":
		return "mode"
	case strings.TrimSpace(fixture.LatencyProfile) == "":
		return "latency_profile"
	case strings.TrimSpace(fixture.AnswerStyle) == "":
		return "answer_style"
	case strings.TrimSpace(fixture.PrivacyScope) == "":
		return "privacy_scope"
	case fixture.MaxFirstResponseMS == nil:
		return "max_first_response_ms"
	case fixture.Confidence == nil:
		return "confidence"
	case fixture.EvidenceCount == nil:
		return "evidence_count"
	case fixture.SpeechBlockCount == nil:
		return "speech_block_count"
	case fixture.ScreenCardCount == nil:
		return "screen_card_count"
	case fixture.FollowUpCount == nil:
		return "follow_up_count"
	case fixture.RedactionOK == nil:
		return "redaction_ok"
	default:
		return ""
	}
}

func missingProductV21AdapterSmokeReportFieldFinding(field string) productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_adapter_smoke_report_missing_field",
		Message: "V21 adapter smoke report is missing a required field",
		Detail:  field,
	}
}

func invalidProductV21AdapterSmokeReportFinding() productReadinessFinding {
	return productReadinessFinding{
		Code:    "v21_adapter_smoke_report_invalid",
		Message: "V21 adapter smoke report is invalid or unsafe",
	}
}

func productSherpaTTSReady(env []string) bool {
	modelDir := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_MODEL_DIR")), audio.DefaultSherpaONNXTTSModelDir())
	return audio.SherpaONNXTTSModelDirReady(modelDir)
}

func productVoiceCloneTTSReady(env []string) bool {
	command := voiceCloneCommandFromEnv(env)
	refAudio := strings.TrimSpace(appEnvValue(env, "A21_VOICE_CLONE_REF_AUDIO"))
	if command == "" || refAudio == "" {
		return false
	}
	if containsLegacyIdentityPathToken(command) || containsLegacyIdentityPathToken(refAudio) {
		return false
	}
	if _, err := os.Stat(command); err != nil {
		if _, lookErr := exec.LookPath(command); lookErr != nil {
			return false
		}
	}
	info, err := os.Stat(refAudio)
	return err == nil && info.Mode().IsRegular()
}

func productSherpaASRReady(env []string) bool {
	modelDir := firstNonEmpty(strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_ASR_MODEL_DIR")), audio.DefaultSherpaONNXASRModelDir())
	family := strings.TrimSpace(appEnvValue(env, "A21_SHERPA_ONNX_ASR_FAMILY"))
	return audio.SherpaONNXASRModelDirReady(modelDir, family)
}

func buildProductNextActions(report productReadinessReport) []string {
	var actions []string
	if !report.Gateway.Healthy {
		actions = append(actions, "start A21 Gateway with `go run ./cmd/a21 gateway --addr 127.0.0.1:21080`")
	}
	if !report.Provider.RealProviderReady {
		if report.Provider.Selected != "" && report.Provider.Selected != "mock" && report.Provider.SelectedConfigured {
			actions = append(actions, fmt.Sprintf("run `go run ./cmd/a21 provider-smoke --provider %s --execute --stream --repeat 3 --output-dir reports` and pass it to product-readiness with --provider-smoke-report", report.Provider.Selected))
		} else {
			actions = append(actions, "configure a real A21 provider with A21_PROVIDER_PRIMARY plus its required env names")
		}
	}
	if !report.V21.Healthy && !productProfessionalRitualReady(report.V21) {
		actions = append(actions, "start/configure the A21 V21 adapter boundary with A21_V21_ADAPTER_URL")
	}
	if !productV21ProfessionalReady(report.V21) {
		actions = append(actions, "run `go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports` and pass it to product-readiness with --v21-adapter-smoke-report")
	} else if !productProfessionalRitualReady(report.V21) {
		actions = append(actions, "run `go run ./cmd/a21-lab xiaozhi-professional-bench --gateway-url <gateway> --output-dir reports` and pass it to product-readiness with --v21-professional-report")
	} else if !productProfessionalReadRecordReady(report.V21) {
		actions = append(actions, "rerun `go run ./cmd/a21-lab xiaozhi-professional-bench --gateway-url <gateway> --output-dir reports` against a Gateway that exposes completed professional read records")
	}
	if !report.StackChan.PhysicalDeviceOnline {
		actions = append(actions, "bring a physical StackChan online against the A21 Gateway")
	}
	if report.StackChan.PhysicalDeviceOnline && !report.StackChan.PhysicalMicrophoneReady {
		status := firstNonEmpty(report.StackChan.MicrophoneStatus, "unknown")
		actions = append(actions, "promote StackChan microphone to a product-ready firmware capability; current status: "+status)
	}
	if (report.StackChan.PhysicalDeviceOnline || report.StackChan.PhysicalEvidence.Valid) && !report.StackChan.PhysicalEvidence.PRDPhysicalAccepted {
		switch {
		case report.StackChan.PhysicalEvidence.CandidatePhysicalVoiceEvidence:
			actions = append(actions, productXiaozhiPhysicalEvidenceNextAction(report.StackChan.PhysicalEvidence))
		case report.StackChan.PhysicalEvidence.CandidatePhysicalEvidence:
			actions = append(actions, "complete human physical StackChan review for the candidate evidence report")
		case report.StackChan.PhysicalEvidence.HostLoopbackOnly:
			actions = append(actions, "collect physical StackChan evidence; host-loopback evidence is not production acceptance")
		default:
			actions = append(actions, "attach a PRD-accepted physical StackChan evidence report")
		}
	}
	if !report.Voice.RealASRReady {
		actions = append(actions, "install or configure real local ASR with A21_LOCAL_ASR_PROVIDER=sherpa_onnx and A21_SHERPA_ONNX_ASR_MODEL_DIR")
	}
	if !report.WakeWord.ProductReady {
		if report.WakeWord.FirmwareBuildRequired {
			if report.WakeWord.FirmwarePackageAvailable {
				actions = append(actions, "use the wake word firmware package in a guarded hardware-window flash plan and collect physical custom wake proof")
			} else if report.WakeWord.FirmwarePlanAvailable {
				actions = append(actions, "use the wake word firmware plan to prepare the guarded build/package and hardware-window flash")
			} else {
				actions = append(actions, "run wake word firmware plan with `go run ./cmd/a21 wake-word-firmware-plan --output-dir reports` for the stored custom MultiNet profile")
			}
		} else {
			actions = append(actions, "restore A21 Gateway wake word readiness before launch")
		}
	} else if !productWakeWordPhysicalAccepted(report.WakeWord) {
		actions = append(actions, "collect physical wake-word acceptance evidence for the active A21 product wake path")
	}
	if !productVoiceChainLaunchPolicySatisfied(report.Voice.VoiceChain) {
		if productVoiceChainHasFinding(report.Voice.VoiceChain, "stepfun_not_selected") {
			actions = append(actions, "select StepFun in the A21 voice-chain profile before full launch evidence is treated as server-side ready")
		} else if !report.Voice.VoiceChain.Available {
			actions = append(actions, "refresh A21 Gateway voice-chain selector readiness before full launch")
		} else {
			actions = append(actions, "align the A21 voice-chain selector with launch policy before full launch")
		}
	}
	return actions
}

func productXiaozhiPhysicalEvidenceNextAction(physical productPhysicalStackChanReadiness) string {
	availability := physical.CanonicalMetricAvailability
	var missing []string
	if !availability["device_playback_start_ms"] {
		missing = append(missing, "device playback ack")
	}
	if !physical.OperatorInstrumentObservationAvailable {
		missing = append(missing, "operator audible observation or instrumented playback observation")
	}
	if !availability["device_downlink_first_frame_ms"] {
		missing = append(missing, "device downlink first-frame timing")
	}
	if !availability["speech_end_to_first_audible_response_ms"] {
		missing = append(missing, "speech-end to first audible response timing")
	}
	if !availability["barge_in_stop_ms"] {
		missing = append(missing, "barge-in stop timing")
	}
	if !availability["barge_in_playback_stop_done_ms"] {
		missing = append(missing, "barge-in playback stop_done")
	}
	if len(missing) == 0 {
		return "run three consecutive physical Xiaozhi PRD acceptance rounds and attach the evidence report"
	}
	return "collect missing " + strings.Join(missing, ", ") + " for the Xiaozhi physical evidence report"
}

func productSimulatorURL(gatewayURL string) string {
	base := strings.TrimRight(gatewayURL, "/")
	return base + "/simulator"
}
