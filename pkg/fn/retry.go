package fn

import (
	"context"
	"math/rand"
	"time"
)

// RetryOpts configures retry behavior.
type RetryOpts struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Jitter      bool
}

// DefaultRetry provides sensible retry defaults.
var DefaultRetry = RetryOpts{
	MaxAttempts: 3,
	InitialWait: time.Second,
	MaxWait:     30 * time.Second,
	Jitter:      true,
}

// Retry retries f up to MaxAttempts times with exponential backoff.
func Retry[T any](ctx context.Context, opts RetryOpts, f func(context.Context) Result[T]) Result[T] {
	var result Result[T]
	wait := opts.InitialWait

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		result = f(ctx)
		if result.IsOk() {
			return result
		}
		if attempt == opts.MaxAttempts-1 {
			break
		}
		// Check context before sleeping
		select {
		case <-ctx.Done():
			return Err[T](ctx.Err())
		default:
		}

		sleepDur := wait
		if opts.Jitter {
			sleepDur = time.Duration(float64(wait) * (0.5 + rand.Float64()))
		}
		if sleepDur > opts.MaxWait {
			sleepDur = opts.MaxWait
		}

		select {
		case <-ctx.Done():
			return Err[T](ctx.Err())
		case <-time.After(sleepDur):
		}

		wait *= 2
		if wait > opts.MaxWait {
			wait = opts.MaxWait
		}
	}
	return result
}

// RetryStage wraps a Stage with retry logic.
func RetryStage[In, Out any](opts RetryOpts, stage Stage[In, Out]) Stage[In, Out] {
	return func(ctx context.Context, in In) Result[Out] {
		return Retry(ctx, opts, func(ctx context.Context) Result[Out] {
			return stage(ctx, in)
		})
	}
}
