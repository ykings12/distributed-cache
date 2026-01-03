package ai

import (
	"strings"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
)

// HealthAnalyzer converts metrics + logs into a health report.
type HealthAnalyzer struct {
	metrics *metrics.Registry
	logger  *logs.Logger
	rules   []Rule
}

// NewHealthAnalyzer creates a new analyzer.
func NewHealthAnalyzer(
	reg *metrics.Registry,
	logger *logs.Logger,
) *HealthAnalyzer {
	return &HealthAnalyzer{
		metrics: reg,
		logger:  logger,
		rules: []Rule{
			ReplicationRetryRule,
			PeerUnhealthyRule,
			HeartbeatFailureRule,
		},
	}
}

// Analyze evaluates metrics and logs and returns a health report.
func (ha *HealthAnalyzer) Analyze() HealthReport {
	snapshot := ha.metrics.Snapshot()

	var (
		signals         = []string{}
		recommendations = []string{}
		status          = StatusOK
	)

	/* ---------- METRICS-BASED RULES ---------- */

	for _, rule := range ha.rules {
		result := rule(snapshot)
		if !result.Triggered {
			continue
		}

		signals = append(signals, result.Signal)
		recommendations = append(recommendations, result.Recommendation)

		// Escalate status
		if result.Severity == StatusCritical {
			status = StatusCritical
		} else if result.Severity == StatusDegraded && status == StatusOK {
			status = StatusDegraded
		}
	}

	/* ---------- LOG-BASED SIGNALS (PHASE 8.1) ---------- */

	logEntries := ha.logger.GetLast(100)

	replicationFailures := 0
	panicCount := 0

	for _, entry := range logEntries {
		if entry.Level == logs.WARN &&
			strings.Contains(entry.Message, "replication failed") {
			replicationFailures++
		}

		if entry.Level == logs.ERROR &&
			strings.Contains(entry.Message, "panic") {
			panicCount++
		}
	}

	if replicationFailures >= 3 {
		signals = append(signals,
			"Repeated replication failures detected in logs",
		)
		recommendations = append(recommendations,
			"Investigate network connectivity or peer health",
		)
		if status == StatusOK {
			status = StatusDegraded
		}
	}

	if panicCount > 0 {
		signals = append(signals,
			"Application panics detected in logs",
		)
		recommendations = append(recommendations,
			"Inspect stack traces and stabilize error handling",
		)
		status = StatusCritical
	}

	/* ---------- SUMMARY ---------- */

	summary := "System is healthy"
	if status != StatusOK {
		summary = "System health issues detected"
	}

	return HealthReport{
		OverallStatus:   status,
		Summary:         summary,
		Signals:         signals,
		Recommendations: recommendations,
	}
}
