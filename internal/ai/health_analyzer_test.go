package ai

import (
	"testing"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"

	"github.com/stretchr/testify/assert"
)

func TestHealthAnalyzer_OK(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusOK, report.OverallStatus)
	assert.Empty(t, report.Signals)
}

func TestHealthAnalyzer_DegradedReplicationMetric(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	reg.Inc(metrics.ReplicationRetriesTotal)

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusDegraded, report.OverallStatus)
	assert.Contains(t, report.Signals, "Replication retries detected")
}

func TestHealthAnalyzer_CriticalPeerFailure(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	reg.Inc(metrics.PeersUnhealthy)

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusCritical, report.OverallStatus)
	assert.Contains(t, report.Signals, "One or more peers are unhealthy")
}

func TestHealthAnalyzer_MultipleMetricSignals(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	reg.Inc(metrics.ReplicationRetriesTotal)
	reg.Inc(metrics.HeartbeatFailuresTotal)

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusDegraded, report.OverallStatus)
	assert.Len(t, report.Signals, 2)
}

func TestHealthAnalyzer_LogBasedReplicationFailures(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	logger.Warn("replication failed to peer node-1")
	logger.Warn("replication failed to peer node-1")
	logger.Warn("replication failed to peer node-1")

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusDegraded, report.OverallStatus)
	assert.Contains(
		t,
		report.Signals,
		"Repeated replication failures detected in logs",
	)
}

func TestHealthAnalyzer_LogBasedPanicDetection(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	logger.Error("panic: runtime error")

	analyzer := NewHealthAnalyzer(reg, logger)
	report := analyzer.Analyze()

	assert.Equal(t, StatusCritical, report.OverallStatus)
	assert.Contains(
		t,
		report.Signals,
		"Application panics detected in logs",
	)
}
