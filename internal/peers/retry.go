package peers

import (
	"context"
	"time"
)

// Retry executes fn with retries, backoff, and cancellation support.
//
// fn must return nil on success.
// Any non-nil error is treated as retryable.
func Retry(
	ctx context.Context,
	policy RetryPolicy,
	fn func() error,
) error {

	var attempt int
	var backoff = policy.BaseBackoff

	for {
		err := fn()
		if err == nil {
			return nil
		}

		attempt++
		if attempt > policy.MaxRetries {
			return err
		}

		delay := backoff
		if policy.JitterFn != nil {
			delay += policy.JitterFn(backoff)
		}
		if delay > policy.MaxBackoff {
			delay = policy.MaxBackoff
		}

		select {
		case <-time.After(delay):
			backoff *= 2
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
