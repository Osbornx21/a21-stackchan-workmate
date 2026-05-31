package v21adapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const ProfessionalReadinessSchemaVersion = "a21.v21_professional_readiness.v1"

type ProfessionalReadinessReport struct {
	SchemaVersion                string                         `json:"schema_version"`
	Status                       string                         `json:"status"`
	CheckingAckAvailable         bool                           `json:"checking_ack_available"`
	CheckingAckMS                float64                        `json:"checking_ack_ms"`
	CheckingAckWithin1200        bool                           `json:"checking_ack_within_1200"`
	EvidenceCompletedMS          float64                        `json:"evidence_completed_ms"`
	EvidenceCompletedAfterAck    bool                           `json:"evidence_completed_after_ack"`
	EvidenceAvailable            bool                           `json:"evidence_available"`
	CardsAvailable               bool                           `json:"cards_available"`
	FollowUpsAvailable           bool                           `json:"follow_ups_available"`
	EvidenceCount                int                            `json:"evidence_count"`
	EvidenceTypes                []string                       `json:"evidence_types,omitempty"`
	CardCount                    int                            `json:"card_count"`
	FollowUpCount                int                            `json:"follow_up_count"`
	AdapterConfigured            bool                           `json:"adapter_configured"`
	AdapterExecuted              bool                           `json:"adapter_executed"`
	RedactionOK                  bool                           `json:"redaction_ok"`
	ProfessionalAcceptanceStatus string                         `json:"professional_acceptance_status"`
	Findings                     []ProfessionalReadinessFinding `json:"findings,omitempty"`
	ReportPath                   string                         `json:"report_path,omitempty"`
}

type ProfessionalReadinessFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func ProfessionalReadiness(ctx context.Context, adapterURL string, client Client) ProfessionalReadinessReport {
	report := ProfessionalReadinessReport{
		SchemaVersion:   ProfessionalReadinessSchemaVersion,
		Status:          "blocked",
		AdapterExecuted: false,
	}

	adapter := NewProfessionalBridgeReadiness(adapterURL)
	report.AdapterConfigured = adapter.Configured
	switch adapter.Status {
	case "disabled":
		report.ProfessionalAcceptanceStatus = "adapter_disabled"
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "v21_adapter_disabled",
			Severity: "warn",
			Message:  "A21 V21 adapter URL is not configured",
		})
	case "misconfigured":
		report.ProfessionalAcceptanceStatus = "adapter_misconfigured"
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "v21_adapter_misconfigured",
			Severity: "block",
			Message:  "A21 V21 adapter configuration is invalid",
		})
	default:
		report.ProfessionalAcceptanceStatus = "host_mock_ready"
	}

	if client == nil {
		client = NewMockClient()
	}

	started := time.Now()
	receipt, err := NewProfessionalBridgeReceipt(professionalReadinessRequest())
	report.CheckingAckMS = elapsedMS(started)
	if err != nil {
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "professional_checking_ack_unavailable",
			Severity: "block",
			Message:  "A21 professional checking acknowledgement is unavailable",
		})
	} else {
		report.CheckingAckAvailable = receipt.Status == "checking" && strings.TrimSpace(receipt.Text) != ""
		report.CheckingAckWithin1200 = report.CheckingAckMS <= float64(ProfessionalMaxFirstResponseMS)
		if !report.CheckingAckAvailable {
			report.Findings = append(report.Findings, ProfessionalReadinessFinding{
				Code:     "professional_checking_ack_unavailable",
				Severity: "block",
				Message:  "A21 professional checking acknowledgement is unavailable",
			})
		}
		if !report.CheckingAckWithin1200 {
			report.Findings = append(report.Findings, ProfessionalReadinessFinding{
				Code:     "professional_checking_ack_slow",
				Severity: "block",
				Message:  "A21 professional checking acknowledgement exceeded 1200 ms",
			})
		}
	}

	response, err := client.Query(ctx, professionalReadinessRequest())
	report.EvidenceCompletedMS = elapsedMS(started)
	report.EvidenceCompletedAfterAck = report.EvidenceCompletedMS >= report.CheckingAckMS
	if err != nil {
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "professional_mock_evidence_unavailable",
			Severity: "block",
			Message:  "A21 mock professional evidence contract is unavailable",
		})
	} else if evidence, err := NewProfessionalBridgeEvidenceReport(response); err != nil {
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "professional_mock_evidence_invalid",
			Severity: "block",
			Message:  "A21 mock professional evidence contract is invalid",
		})
	} else {
		report.EvidenceAvailable = evidence.EvidenceCount > 0
		report.CardsAvailable = evidence.ScreenCardCount > 0
		report.FollowUpsAvailable = evidence.FollowUpCount > 0
		report.EvidenceCount = evidence.EvidenceCount
		report.CardCount = evidence.ScreenCardCount
		report.FollowUpCount = evidence.FollowUpCount
		for _, card := range evidence.EvidenceCards {
			if card.Type != "" {
				report.EvidenceTypes = append(report.EvidenceTypes, card.Type)
			}
		}
		if !report.EvidenceAvailable || !report.CardsAvailable || !report.FollowUpsAvailable {
			report.Findings = append(report.Findings, ProfessionalReadinessFinding{
				Code:     "professional_mock_evidence_incomplete",
				Severity: "block",
				Message:  "A21 mock professional evidence/cards/follow-ups are incomplete",
			})
		}
	}

	report.RedactionOK = professionalReadinessRedactionOK(report)
	if !report.RedactionOK {
		report.Findings = append(report.Findings, ProfessionalReadinessFinding{
			Code:     "professional_report_redaction_failed",
			Severity: "block",
			Message:  "A21 professional readiness report redaction failed",
		})
	}
	if len(report.Findings) == 0 {
		report.Status = "passed"
	}
	return report
}

func professionalReadinessRequest() QueryRequest {
	return QueryRequest{
		TraceID:      "a21-trace-professional-readiness",
		SessionID:    "a21-session-professional-readiness",
		Mode:         "professional",
		Utterance:    "professional readiness fixture query",
		PrivacyScope: "professional_only",
	}
}

func professionalReadinessRedactionOK(report ProfessionalReadinessReport) bool {
	encoded, err := json.Marshal(report)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(encoded))
	for _, forbidden := range []string{
		"professional readiness fixture query",
		"raw retrieved evidence",
		"raw prompt text",
		"raw transcript text",
		"raw provider output",
		"raw reasoning text",
		"http://",
		"https://",
		"/users/",
		"api_key",
		"secret-token",
	} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}

func elapsedMS(started time.Time) float64 {
	return float64(time.Since(started).Microseconds()) / 1000
}
