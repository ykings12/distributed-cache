package peers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPeerConfig_JitterFn(t *testing.T) {
	cfg := DefaultPeerConfig()

	assert.NotNil(t, cfg.Retry.JitterFn)

	backoff := 100 * time.Millisecond
	jitter := cfg.Retry.JitterFn(backoff)

	assert.Equal(t, 50*time.Millisecond, jitter,
		"default jitter should be 50% of backoff")
}
