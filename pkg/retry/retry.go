package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Operation is a function which will be retried
type Operation func(ctx context.Context) error

// IsRetryableFunc is a function which checks if error is retryable
type IsRetryableFunc func(err error) bool

type Retrier struct {
	backoff     []time.Duration
	IsRetryable IsRetryableFunc
}

type Option func(*Retrier)

// WithBackoff functional option sets custom backoff durations
func WithBackoff(durations []time.Duration) Option {
	return func(r *Retrier) {
		r.backoff = durations
	}
}

func NewRetrier(IsRetryable IsRetryableFunc, opts ...Option) *Retrier {
	r := &Retrier{
		backoff: []time.Duration{
			1 * time.Second,
			3 * time.Second,
			5 * time.Second,
		},
		IsRetryable: IsRetryable,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Do executes Operation function and retries it according to configured backoff
func (r *Retrier) Do(ctx context.Context, op Operation) error {
	var lastErr error
	err := op(ctx)
	if err == nil {
		return nil
	}

	if !r.IsRetryable(err) {
		return err
	}
	lastErr = err

	for _, t := range r.backoff {
		log.Printf("operation error, retrying in %v", t)
		time.Sleep(t)

		err = op(ctx)
		if err == nil {
			return nil
		}
		if !r.IsRetryable(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("operation failed after retries: %w", lastErr)
}
