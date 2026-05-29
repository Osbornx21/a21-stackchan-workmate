package runtimeguard

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityBlock Severity = "block"
)

type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Detail   string   `json:"detail,omitempty"`
}

type Result struct {
	OK       bool      `json:"ok"`
	Findings []Finding `json:"findings"`
}

func NewResult(findings []Finding) Result {
	ok := true
	for _, finding := range findings {
		if finding.Severity == SeverityBlock {
			ok = false
			break
		}
	}
	return Result{OK: ok, Findings: findings}
}
