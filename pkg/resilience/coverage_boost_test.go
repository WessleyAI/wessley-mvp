package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestNewBreakerDefaults(t *testing.T) {
	b := NewBreaker(BreakerOpts{}) // all zero → should use defaults
	if b.opts.FailThreshold != DefaultBreakerOpts.FailThreshold {
		t.Errorf("FailThreshold = %d, want %d", b.opts.FailThreshold, DefaultBreakerOpts.FailThreshold)
	}
	if b.opts.Timeout != DefaultBreakerOpts.Timeout {
		t.Errorf("Timeout = %v, want %v", b.opts.Timeout, DefaultBreakerOpts.Timeout)
	}
	if b.opts.HalfOpenMax != DefaultBreakerOpts.HalfOpenMax {
		t.Errorf("HalfOpenMax = %d, want %d", b.opts.HalfOpenMax, DefaultBreakerOpts.HalfOpenMax)
	}
}

func TestCallResultOpenAndHalfOpen(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()

	fail := func(_ context.Context) fn.Result[int] { return fn.Err[int](errors.New("fail")) }
	ok := func(_ context.Context) fn.Result[int] { return fn.Ok(42) }

	// Trip via CallResult
	CallResult(b, ctx, fail)
	CallResult(b, ctx, fail)

	// Open → reject
	r := CallResult(b, ctx, ok)
	if r.IsOk() {
		t.Fatal("expected reject when open")
	}

	// Advance to half-open
	now = now.Add(6 * time.Second)

	// First probe succeeds → closed
	r = CallResult(b, ctx, ok)
	if !r.IsOk() {
		t.Fatal("expected success in half-open")
	}
	v, _ := r.Unwrap()
	if v != 42 {
		t.Fatalf("got %d, want 42", v)
	}
}

func TestCallResultHalfOpenExceedsMax(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()

	fail := func(_ context.Context) fn.Result[int] { return fn.Err[int](errors.New("fail")) }

	// Trip
	CallResult(b, ctx, fail)
	CallResult(b, ctx, fail)

	// half-open
	now = now.Add(6 * time.Second)

	// First probe (will fail → back to open)
	CallResult(b, ctx, fail)

	// advance again
	now = now.Add(6 * time.Second)

	// Use one probe slot
	CallResult(b, ctx, fail)

	// Second should be rejected (halfOpenCount >= HalfOpenMax)
	r := CallResult(b, ctx, func(_ context.Context) fn.Result[int] { return fn.Ok(1) })
	if r.IsOk() {
		t.Fatal("expected reject when half-open max exceeded")
	}
}

func TestCallResultHalfOpenFailure(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()

	fail := func(_ context.Context) fn.Result[int] { return fn.Err[int](errors.New("fail")) }

	CallResult(b, ctx, fail)
	CallResult(b, ctx, fail)

	now = now.Add(6 * time.Second)

	// Fail in half-open → back to open
	r := CallResult(b, ctx, fail)
	if !r.IsErr() {
		t.Fatal("expected error")
	}
	if b.State() != StateOpen {
		t.Fatalf("expected open, got %v", b.State())
	}
}

func TestCallWait(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1000, Burst: 1})
	ctx := context.Background()

	// CallWait should succeed
	err := l.CallWait(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	// Drain, then CallWait with fast refill
	err = l.CallWait(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestCallWaitCancelled(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 0.001, Burst: 1})
	l.Allow() // drain
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := l.CallWait(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}

func TestLimiterStageWait(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1000, Burst: 2})
	ctx := context.Background()

	stage := LimiterStageWait(l, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Ok(in * 3)
	})

	r := stage(ctx, 5)
	if r.IsErr() {
		t.Fatal("expected success")
	}
	v, _ := r.Unwrap()
	if v != 15 {
		t.Fatalf("got %d, want 15", v)
	}
}

func TestLimiterStageWaitCancelled(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 0.001, Burst: 1})
	l.Allow() // drain
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	stage := LimiterStageWait(l, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Ok(in)
	})

	r := stage(ctx, 1)
	if r.IsOk() {
		t.Fatal("expected error from cancelled wait")
	}
}

func TestNewLimiterDefaultBurst(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 0})
	if l.opts.Burst != 1 {
		t.Fatalf("expected default burst=1, got %d", l.opts.Burst)
	}
}

func TestBreakerCallHalfOpenMaxExceeded(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()
	fail := errors.New("fail")

	// Trip
	b.Call(ctx, func(context.Context) error { return fail })
	b.Call(ctx, func(context.Context) error { return fail })

	// half-open
	now = now.Add(6 * time.Second)

	// Use the one probe slot (fail → back to open)
	b.Call(ctx, func(context.Context) error { return fail })

	// Advance again to half-open
	now = now.Add(6 * time.Second)

	// Use probe
	b.Call(ctx, func(context.Context) error { return fail })

	// Now halfOpenCount >= HalfOpenMax, next call should reject
	err := b.Call(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}
