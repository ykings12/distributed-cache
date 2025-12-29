package peers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	t.Run("success_on_first_attempt", func(t *testing.T) {
		cfg := RetryPolicy{
			MaxRetries:  3,
			BaseBackoff: 10 * time.Millisecond,
			MaxBackoff:  100 * time.Millisecond,
			JitterFn: func(d time.Duration) time.Duration {
				return 0
			},
		}

		err := Retry(context.Background(), cfg, func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("success_after_retry", func(t *testing.T) {
		attempts := 0

		cfg := RetryPolicy{
			MaxRetries:  3,
			BaseBackoff: 1 * time.Millisecond,
			MaxBackoff:  10 * time.Millisecond,
			JitterFn: func(d time.Duration) time.Duration {
				return 0
			},
		}

		err := Retry(context.Background(), cfg, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("failed")
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("exhaust_retries", func(t *testing.T) {
		attempts := 0

		cfg := RetryPolicy{
			MaxRetries:  2,
			BaseBackoff: 1 * time.Millisecond,
			MaxBackoff:  5 * time.Millisecond,
			JitterFn: func(d time.Duration) time.Duration {
				return 0
			},
		}

		err := Retry(context.Background(), cfg, func() error {
			attempts++
			return errors.New("failed")
		})
		assert.Error(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := RetryPolicy{
			MaxRetries:  5,
			BaseBackoff: 10 * time.Millisecond,
			MaxBackoff:  100 * time.Millisecond,
			JitterFn: func(d time.Duration) time.Duration {
				return 0
			},
		}

		err := Retry(ctx, cfg, func() error {
			return errors.New("failed")
		})
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("max backoff cap", func(t *testing.T) {
		attempts := 0

		cfg := RetryPolicy{
			MaxRetries:  3,
			BaseBackoff: 50 * time.Millisecond,
			MaxBackoff:  60 * time.Millisecond,
			JitterFn: func(d time.Duration) time.Duration {
				return 0
			},
		}
		start := time.Now()

		_ = Retry(context.Background(), cfg, func() error {
			attempts++
			return errors.New("failed")
		})

		elapsed := time.Since(start)
		assert.Greater(t, elapsed, cfg.BaseBackoff, "expected backoff to be greater than base backoff")
		assert.Equal(t, 4, attempts, "expected 4 attempts")
	})

}
