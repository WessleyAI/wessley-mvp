package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

var ErrRateLimited = errors.New("rate limited")

// LimiterOpts configures the token bucket rate limiter.
type LimiterOpts struct {
	// Rate is the number of tokens added per second.
	Rate float64
	// Burst is the maximum number of tokens (bucket capacity).
	Burst int
}

// Limiter implements a token bucket rate limiter.
type Limiter struct {
	mu     sync.Mutex
	opts   LimiterOpts
	tokens float64
	last   time.Time
	now    func() time.Time
}

// NewLimiter creates a token bucket rate limiter.
func NewLimiter(opts LimiterOpts) *Limiter {
	if opts.Burst <= 0 {
		opts.Burst = 1
	}
	return &Limiter{
		opts:   opts,
		tokens: float64(opts.Burst),
		now:    time.Now,
	}
}

// Allow checks if a request is allowed (non-blocking).
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or ctx is cancelled.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		l.refill()
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		// Calculate time until next token
		deficit := 1.0 - l.tokens
		waitDur := time.Duration(deficit / l.opts.Rate * float64(time.Second))
		l.mu.Unlock()

		if waitDur < time.Millisecond {
			waitDur = time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDur):
		}
	}
}

// refill adds tokens based on elapsed time. Must hold mu.
func (l *Limiter) refill() {
	now := l.now()
	if l.last.IsZero() {
		l.last = now
		return
	}
	elapsed := now.Sub(l.last).Seconds()
	l.tokens += elapsed * l.opts.Rate
	if l.tokens > float64(l.opts.Burst) {
		l.tokens = float64(l.opts.Burst)
	}
	l.last = now
}

// Call executes f if a token is available, otherwise returns ErrRateLimited.
func (l *Limiter) Call(ctx context.Context, f func(context.Context) error) error {
	if !l.Allow() {
		return ErrRateLimited
	}
	return f(ctx)
}

// CallWait waits for a token then executes f.
func (l *Limiter) CallWait(ctx context.Context, f func(context.Context) error) error {
	if err := l.Wait(ctx); err != nil {
		return err
	}
	return f(ctx)
}

// LimiterStage wraps an fn.Stage with rate limiting (non-blocking, returns error if limited).
func LimiterStage[In, Out any](l *Limiter, stage fn.Stage[In, Out]) fn.Stage[In, Out] {
	return func(ctx context.Context, in In) fn.Result[Out] {
		if !l.Allow() {
			return fn.Err[Out](ErrRateLimited)
		}
		return stage(ctx, in)
	}
}

// LimiterStageWait wraps an fn.Stage with rate limiting (blocking, waits for token).
func LimiterStageWait[In, Out any](l *Limiter, stage fn.Stage[In, Out]) fn.Stage[In, Out] {
	return func(ctx context.Context, in In) fn.Result[Out] {
		if err := l.Wait(ctx); err != nil {
			return fn.Err[Out](err)
		}
		return stage(ctx, in)
	}
}
