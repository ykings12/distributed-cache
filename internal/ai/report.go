package ai

import "distributed-cache/internal/metrics"

// RuleResult represents the outcome of a single rule.
type RuleResult struct {
	Triggered      bool
	Signal         string
	Recommendation string
	Severity       HealthStatus
}

// Rule evaluates a metrics snapshot.
type Rule func(snapshot map[string]int64) RuleResult

// ---------- RULES ----------

// Replication retries indicate instability.
func ReplicationRetryRule(snapshot map[string]int64) RuleResult {
	retries := snapshot[string(metrics.ReplicationRetriesTotal)]

	if retries > 0 {
		return RuleResult{
			Triggered:      true,
			Signal:         "Replication retries detected",
			Recommendation: "Check network connectivity or replication timeouts",
			Severity:       StatusDegraded,
		}
	}
	return RuleResult{}
}

// Unhealthy peers indicate cluster instability.
func PeerUnhealthyRule(snapshot map[string]int64) RuleResult {
	unhealthy := snapshot[string(metrics.PeersUnhealthy)]

	if unhealthy > 0 {
		return RuleResult{
			Triggered:      true,
			Signal:         "One or more peers are unhealthy",
			Recommendation: "Inspect peer health and heartbeat configuration",
			Severity:       StatusCritical,
		}
	}
	return RuleResult{}
}

// Frequent heartbeat failures indicate liveness issues.
func HeartbeatFailureRule(snapshot map[string]int64) RuleResult {
	failures := snapshot[string(metrics.HeartbeatFailuresTotal)]

	if failures > 0 {
		return RuleResult{
			Triggered:      true,
			Signal:         "Heartbeat failures detected",
			Recommendation: "Check peer availability and heartbeat endpoints",
			Severity:       StatusDegraded,
		}
	}
	return RuleResult{}
}
