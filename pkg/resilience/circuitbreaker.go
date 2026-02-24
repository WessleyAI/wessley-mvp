// Package resilience provides circuit breaker and rate limiter primitives.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

// Circuit breaker states.
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // tripping, reject calls
	StateHalfOpen              // allowing a probe call
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// BreakerOpts configures the circuit breaker.
type BreakerOpts struct {
	// FailThreshold is how many consecutive failures trip the breaker.
	FailThreshold int
	// Timeout is how long the breaker stays open before entering half-open.
	Timeout time.Duration
	// HalfOpenMax is the number of probe calls allowed in half-open state.
	HalfOpenMax int
}

// DefaultBreakerOpts provides sensible defaults.
var DefaultBreakerOpts = BreakerOpts{
	FailThreshold: 5,
	Timeout:       30 * time.Second,
	HalfOpenMax:   1,
}

// Breaker implements a circuit breaker with closed/open/half-open states.
type Breaker struct {
	mu            sync.Mutex
	opts          BreakerOpts
	state         State
	failures      int
	openedAt      time.Time
	halfOpenCount int
	now           func() time.Time // for testing
}

// NewBreaker creates a circuit breaker with the given options.
func NewBreaker(opts BreakerOpts) *Breaker {
	if opts.FailThreshold <= 0 {
		opts.FailThreshold = DefaultBreakerOpts.FailThreshold
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultBreakerOpts.Timeout
	}
	if opts.HalfOpenMax <= 0 {
		opts.HalfOpenMax = DefaultBreakerOpts.HalfOpenMax
	}
	return &Breaker{opts: opts, now: time.Now}
}

// State returns the current breaker state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentState()
}

// currentState returns state, transitioning open→half-open if timeout elapsed. Must hold mu.
func (b *Breaker) currentState() State {
	if b.state == StateOpen && b.now().Sub(b.openedAt) >= b.opts.Timeout {
		b.state = StateHalfOpen
		b.halfOpenCount = 0
	}
	return b.state
}

// Call executes f through the circuit breaker.
func (b *Breaker) Call(ctx context.Context, f func(context.Context) error) error {
	b.mu.Lock()
	st := b.currentState()

	switch st {
	case StateOpen:
		b.mu.Unlock()
		return ErrCircuitOpen
	case StateHalfOpen:
		if b.halfOpenCount >= b.opts.HalfOpenMax {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
		b.halfOpenCount++
	}
	b.mu.Unlock()

	err := f(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		if b.state == StateHalfOpen || b.failures >= b.opts.FailThreshold {
			b.state = StateOpen
			b.openedAt = b.now()
			b.failures = 0
			b.halfOpenCount = 0
		}
		return err
	}

	// Success
	if b.state == StateHalfOpen {
		b.state = StateClosed
	}
	b.failures = 0
	return nil
}

// CallResult is a generic version of Call that works with fn.Result.
func CallResult[T any](b *Breaker, ctx context.Context, f func(context.Context) fn.Result[T]) fn.Result[T] {
	b.mu.Lock()
	st := b.currentState()

	switch st {
	case StateOpen:
		b.mu.Unlock()
		return fn.Err[T](ErrCircuitOpen)
	case StateHalfOpen:
		if b.halfOpenCount >= b.opts.HalfOpenMax {
			b.mu.Unlock()
			return fn.Err[T](ErrCircuitOpen)
		}
		b.halfOpenCount++
	}
	b.mu.Unlock()

	result := f(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	if result.IsErr() {
		b.failures++
		if b.state == StateHalfOpen || b.failures >= b.opts.FailThreshold {
			b.state = StateOpen
			b.openedAt = b.now()
			b.failures = 0
			b.halfOpenCount = 0
		}
		return result
	}

	if b.state == StateHalfOpen {
		b.state = StateClosed
	}
	b.failures = 0
	return result
}

// BreakerStage wraps an fn.Stage with circuit breaker protection.
func BreakerStage[In, Out any](b *Breaker, stage fn.Stage[In, Out]) fn.Stage[In, Out] {
	return func(ctx context.Context, in In) fn.Result[Out] {
		return CallResult(b, ctx, func(ctx context.Context) fn.Result[Out] {
			return stage(ctx, in)
		})
	}
}
