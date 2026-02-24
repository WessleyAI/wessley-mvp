package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

func TestBreakerStartsClosed(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 3, Timeout: time.Second})
	if b.State() != StateClosed {
		t.Fatalf("expected closed, got %v", b.State())
	}
}

func TestBreakerTripsAfterThreshold(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 3, Timeout: time.Second})
	ctx := context.Background()
	fail := errors.New("fail")

	for i := 0; i < 3; i++ {
		_ = b.Call(ctx, func(context.Context) error { return fail })
	}
	if b.State() != StateOpen {
		t.Fatalf("expected open, got %v", b.State())
	}

	// Calls should be rejected
	err := b.Call(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 3, Timeout: time.Second})
	ctx := context.Background()
	fail := errors.New("fail")

	// 2 failures then success should reset counter
	_ = b.Call(ctx, func(context.Context) error { return fail })
	_ = b.Call(ctx, func(context.Context) error { return fail })
	_ = b.Call(ctx, func(context.Context) error { return nil })
	if b.State() != StateClosed {
		t.Fatalf("expected closed after success, got %v", b.State())
	}

	// Should need 3 more failures to trip
	_ = b.Call(ctx, func(context.Context) error { return fail })
	_ = b.Call(ctx, func(context.Context) error { return fail })
	if b.State() != StateClosed {
		t.Fatalf("expected still closed, got %v", b.State())
	}
}

func TestBreakerHalfOpen(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()
	fail := errors.New("fail")

	// Trip the breaker
	_ = b.Call(ctx, func(context.Context) error { return fail })
	_ = b.Call(ctx, func(context.Context) error { return fail })
	if b.State() != StateOpen {
		t.Fatalf("expected open, got %v", b.State())
	}

	// Advance time past timeout
	now = now.Add(6 * time.Second)
	if b.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %v", b.State())
	}

	// Success in half-open → closed
	_ = b.Call(ctx, func(context.Context) error { return nil })
	if b.State() != StateClosed {
		t.Fatalf("expected closed after half-open success, got %v", b.State())
	}
}

func TestBreakerHalfOpenFailure(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: 5 * time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()
	fail := errors.New("fail")

	// Trip
	_ = b.Call(ctx, func(context.Context) error { return fail })
	_ = b.Call(ctx, func(context.Context) error { return fail })

	// Advance to half-open
	now = now.Add(6 * time.Second)

	// Fail in half-open → back to open
	_ = b.Call(ctx, func(context.Context) error { return fail })
	if b.State() != StateOpen {
		t.Fatalf("expected open after half-open failure, got %v", b.State())
	}
}

func TestBreakerStage(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: time.Second})
	ctx := context.Background()

	stage := BreakerStage(b, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Err[int](errors.New("fail"))
	})

	_ = stage(ctx, 1)
	_ = stage(ctx, 2)

	r := stage(ctx, 3)
	if r.IsOk() {
		t.Fatal("expected error from tripped breaker")
	}
	_, err := r.Unwrap()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}
