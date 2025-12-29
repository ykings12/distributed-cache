package peers

import "time"

// RetryPolicy controls retry behavior for network operations
type RetryPolicy struct {
	MaxRetries  int           //max retry attempts
	BaseBackoff time.Duration //intial backoff duration
	MaxBackoff  time.Duration // upper bound on backoff
	JitterFn    func(time.Duration) time.Duration
}

// TimeoutPolicy defines request-level timeout
type TimeoutPolicy struct {
	ReplicationTimeout time.Duration
	HeartbeatTimeout   time.Duration
}

// HealthPolicy defines when a peer is considered healthy or recovered
type HealthPolicy struct {
	FailureThreshold int //consecutive failures to mark unhealthy
	SuccessThreshold int //consecutive successes to mark healthy again
}

type HeartbeatPolicy struct {
	Interval time.Duration
}

type PeerConfig struct {
	Retry     RetryPolicy
	Timeout   TimeoutPolicy
	Health    HealthPolicy
	Heartbeat HeartbeatPolicy
}

func DefaultPeerConfig() PeerConfig {
	return PeerConfig{
		Retry: RetryPolicy{
			MaxRetries:  3,
			BaseBackoff: 100 * time.Millisecond,
			MaxBackoff:  2 * time.Second,
			JitterFn:    func(d time.Duration) time.Duration { return d / 2 }, //default jitter:50%
		},
		Timeout: TimeoutPolicy{
			ReplicationTimeout: 2 * time.Second,
			HeartbeatTimeout:   1 * time.Second,
		},
		Health: HealthPolicy{
			FailureThreshold: 3,
			SuccessThreshold: 2,
		},
		Heartbeat: HeartbeatPolicy{
			Interval: 5 * time.Second,
		},
	}
}
