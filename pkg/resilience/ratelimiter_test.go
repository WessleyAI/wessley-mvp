package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/pkg/fn"
)

func TestLimiterAllow(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 3})
	// Should allow burst
	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("expected allow on call %d", i)
		}
	}
	// 4th should be rejected
	if l.Allow() {
		t.Fatal("expected rejection after burst exhausted")
	}
}

func TestLimiterRefill(t *testing.T) {
	now := time.Now()
	l := NewLimiter(LimiterOpts{Rate: 10, Burst: 5})
	l.now = func() time.Time { return now }

	// Drain all tokens
	for i := 0; i < 5; i++ {
		l.Allow()
	}
	if l.Allow() {
		t.Fatal("should be empty")
	}

	// Advance 500ms → 5 tokens refilled
	now = now.Add(500 * time.Millisecond)
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("expected allow after refill, call %d", i)
		}
	}
	if l.Allow() {
		t.Fatal("should be empty again")
	}
}

func TestLimiterCall(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1, Burst: 1})
	ctx := context.Background()

	err := l.Call(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = l.Call(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestLimiterWait(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1000, Burst: 1}) // fast refill
	ctx := context.Background()

	l.Allow() // drain

	// Should refill quickly
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("expected Wait to succeed, got %v", err)
	}
}

func TestLimiterWaitCancelled(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 0.001, Burst: 1}) // very slow refill
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	l.Allow() // drain

	err := l.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestLimiterStage(t *testing.T) {
	l := NewLimiter(LimiterOpts{Rate: 1, Burst: 1})
	ctx := context.Background()

	stage := LimiterStage(l, func(ctx context.Context, in int) fn.Result[int] {
		return fn.Ok(in * 2)
	})

	r := stage(ctx, 5)
	if r.IsErr() {
		t.Fatal("expected success")
	}
	v, _ := r.Unwrap()
	if v != 10 {
		t.Fatalf("expected 10, got %d", v)
	}

	// Should be rate limited now
	r = stage(ctx, 5)
	if r.IsOk() {
		t.Fatal("expected rate limit error")
	}
	_, err := r.Unwrap()
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}
