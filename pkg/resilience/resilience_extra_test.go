package resilience

import (
	"context"
	"errors"
	"sync"
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

func TestBreakerDefaultOpts(t *testing.T) {
	b := NewBreaker(BreakerOpts{}) // all zeros → defaults
	if b.opts.FailThreshold != DefaultBreakerOpts.FailThreshold {
		t.Fatal("expected default FailThreshold")
	}
	if b.opts.Timeout != DefaultBreakerOpts.Timeout {
		t.Fatal("expected default Timeout")
	}
	if b.opts.HalfOpenMax != DefaultBreakerOpts.HalfOpenMax {
		t.Fatal("expected default HalfOpenMax")
	}
}

func TestCallResult_Success(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 3, Timeout: time.Second})
	ctx := context.Background()

	r := CallResult(b, ctx, func(ctx context.Context) fn.Result[string] {
		return fn.Ok("hello")
	})
	if !r.IsOk() {
		t.Fatal("expected success")
	}
	v, _ := r.Unwrap()
	if v != "hello" {
		t.Fatalf("expected hello, got %s", v)
	}
}

func TestCallResult_TripsAndRejects(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 2, Timeout: time.Second})
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		CallResult(b, ctx, func(ctx context.Context) fn.Result[int] {
			return fn.Err[int](errors.New("fail"))
		})
	}

	r := CallResult(b, ctx, func(ctx context.Context) fn.Result[int] {
		return fn.Ok(42)
	})
	if r.IsOk() {
		t.Fatal("expected circuit open rejection")
	}
	_, err := r.Unwrap()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerHalfOpenMaxExceeded(t *testing.T) {
	now := time.Now()
	b := NewBreaker(BreakerOpts{FailThreshold: 1, Timeout: time.Second, HalfOpenMax: 1})
	b.now = func() time.Time { return now }
	ctx := context.Background()

	// Trip
	_ = b.Call(ctx, func(context.Context) error { return errors.New("fail") })

	// Advance to half-open
	now = now.Add(2 * time.Second)

	// First probe allowed
	err := b.Call(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("first half-open call should succeed: %v", err)
	}
}

func TestBreakerConcurrentAccess(t *testing.T) {
	b := NewBreaker(BreakerOpts{FailThreshold: 100, Timeout: time.Second})
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Call(ctx, func(context.Context) error { return nil })
		}()
	}
	wg.Wait()

	if b.State() != StateClosed {
		t.Fatal("expected closed after concurrent successes")
	}
}

func TestLimiterBurstDefault(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 0}) // 0 → default 1
	if !l.Allow() {
		t.Fatal("expected at least 1 token")
	}
	if l.Allow() {
		t.Fatal("expected rejection with burst=1")
	}
}

func TestLimiterCallWait(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1000, Burst: 1})
	ctx := context.Background()

	// Drain
	l.Allow()

	err := l.CallWait(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("CallWait should succeed: %v", err)
	}
}

func TestLimiterCallWaitCancelled(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 0.001, Burst: 1})
	l.Allow() // drain
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := l.CallWait(ctx, func(context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestLimiterStageWait(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1000, Burst: 1})
	ctx := context.Background()

	stage := LimiterStageWait(l, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Ok(in * 3)
	})

	r := stage(ctx, 5)
	v, _ := r.Unwrap()
	if v != 15 {
		t.Fatalf("expected 15, got %d", v)
	}
}

func TestLimiterStageWaitCancelled(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 0.001, Burst: 1})
	l.Allow()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	stage := LimiterStageWait(l, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Ok(in)
	})

	r := stage(ctx, 1)
	if r.IsOk() {
		t.Fatal("expected rate limit timeout")
	}
}

func TestLimiterRefillCap(t *testing.T) {
	now := time.Now()
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 3})
	l.now = func() time.Time { return now }

	// Drain all
	l.Allow()
	l.Allow()
	l.Allow()

	// Advance 10 seconds → 100 tokens, but cap at burst=3
	now = now.Add(10 * time.Second)
	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("expected token %d after refill", i)
		}
	}
	if l.Allow() {
		t.Fatal("should be capped at burst")
	}
}

func TestLimiterCallPassesThroughFuncError(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 1})
	ctx := context.Background()
	expected := errors.New("func error")

	err := l.Call(ctx, func(context.Context) error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected func error to pass through, got %v", err)
	}
}
