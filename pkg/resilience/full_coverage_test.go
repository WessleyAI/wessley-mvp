package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

func TestCall_HalfOpenMaxExceeded(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Millisecond, HalfOpenMax: 1})
	now := time.Now()
	b.now = func() time.Time { return now }

	// Trip the breaker
	b.Call(context.Background(), func(_ context.Context) error { return errors.New("fail") })

	// Advance past timeout → half-open
	now = now.Add(2 * time.Millisecond)

	// First call in half-open should be allowed
	err := b.Call(context.Background(), func(_ context.Context) error { return errors.New("fail again") })
	if err == nil || err.Error() != "fail again" {
		t.Fatalf("expected fail again, got %v", err)
	}

	// Second call should be rejected (half-open max exceeded)
	// After the failure above, it should be open again
	err = b.Call(context.Background(), func(_ context.Context) error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}
}

func TestCallResult_HalfOpenSuccess(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Millisecond, HalfOpenMax: 1})
	now := time.Now()
	b.now = func() time.Time { return now }

	// Trip breaker
	CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] {
		return fn.Err[int](errors.New("fail"))
	})

	// Advance past timeout → half-open
	now = now.Add(2 * time.Millisecond)

	// Success in half-open → closed
	r := CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] {
		return fn.Ok(42)
	})
	if r.Must() != 42 {
		t.Fatal("expected 42")
	}
	if b.State() != StateClosed {
		t.Fatalf("expected closed, got %v", b.State())
	}
}

func TestCallResult_HalfOpenFailure(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Millisecond, HalfOpenMax: 1})
	now := time.Now()
	b.now = func() time.Time { return now }

	// Trip breaker
	CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] {
		return fn.Err[int](errors.New("fail"))
	})

	// Advance → half-open
	now = now.Add(2 * time.Millisecond)

	// Fail in half-open → open again
	r := CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] {
		return fn.Err[int](errors.New("fail2"))
	})
	if r.IsOk() {
		t.Fatal("expected error")
	}

	// Should be open now, reject
	r = CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] {
		return fn.Ok(1)
	})
	if r.IsOk() {
		t.Fatal("expected circuit open error")
	}
}

func TestCallResult_HalfOpenMaxExceeded(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Millisecond, HalfOpenMax: 1})
	now := time.Now()
	b.now = func() time.Time { return now }

	// Trip
	b.Call(context.Background(), func(_ context.Context) error { return errors.New("f") })

	// Half-open
	now = now.Add(2 * time.Millisecond)

	// Use Call to consume the half-open slot
	b.mu.Lock()
	b.currentState() // force transition
	b.halfOpenCount = 1
	b.mu.Unlock()

	r := CallResult(b, context.Background(), func(_ context.Context) fn.Result[int] { return fn.Ok(1) })
	if r.IsOk() {
		t.Fatal("expected circuit open")
	}
}

func TestBreakerStage_Simple(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: time.Second, HalfOpenMax: 1})
	stage := func(_ context.Context, v int) fn.Result[int] { return fn.Ok(v * 2) }
	bs := BreakerStage(b, stage)
	r := bs(context.Background(), 5)
	if r.Must() != 10 {
		t.Fatal("expected 10")
	}
}

func TestCall_ClosedSuccess(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 3, Timeout: time.Second, HalfOpenMax: 1})
	err := b.Call(context.Background(), func(_ context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.State() != StateClosed {
		t.Fatal("should be closed")
	}
}

func TestCall_HalfOpenSuccess(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Millisecond, HalfOpenMax: 1})
	now := time.Now()
	b.now = func() time.Time { return now }

	// Trip
	b.Call(context.Background(), func(_ context.Context) error { return errors.New("f") })

	// Half-open
	now = now.Add(2 * time.Millisecond)

	// Success → closed
	err := b.Call(context.Background(), func(_ context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.State() != StateClosed {
		t.Fatal("should be closed after half-open success")
	}
}

func TestStateString_AllValues(t *testing.T) {
	if StateClosed.String() != "closed" {
		t.Fatal("wrong")
	}
	if StateOpen.String() != "open" {
		t.Fatal("wrong")
	}
	if StateHalfOpen.String() != "half-open" {
		t.Fatal("wrong")
	}
	if State(99).String() != "unknown" {
		t.Fatal("wrong")
	}
}

func TestNewBreaker_DefaultOpts(t *testing.T) {
	b := NewBreaker(BreakerOpts{})
	if b.opts.FailThreshold != DefaultBreakerOpts.FailThreshold {
		t.Fatal("should use default threshold")
	}
	if b.opts.Timeout != DefaultBreakerOpts.Timeout {
		t.Fatal("should use default timeout")
	}
	if b.opts.HalfOpenMax != DefaultBreakerOpts.HalfOpenMax {
		t.Fatal("should use default half-open max")
	}
}
