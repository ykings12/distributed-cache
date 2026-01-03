package ai

// HealthStatus represents overall system health.
type HealthStatus string

const (
	StatusOK       HealthStatus = "OK"
	StatusDegraded HealthStatus = "DEGRADED"
	StatusCritical HealthStatus = "CRITICAL"
)

// HealthReport is the AI-style health summary.
type HealthReport struct {
	OverallStatus   HealthStatus `json:"overall_status"`
	Summary         string       `json:"summary"`
	Signals         []string     `json:"signals"`
	Recommendations []string     `json:"recommendations"`
}
