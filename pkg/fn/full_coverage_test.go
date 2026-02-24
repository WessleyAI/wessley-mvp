package fn

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- ParMap edge cases ---

func TestParMap_EmptySlice(t *testing.T) {
	out := ParMap([]int{}, 4, func(v int) int { return v * 2 })
	if len(out) != 0 {
		t.Fatal("expected empty output")
	}
}

func TestParMap_WorkersZero(t *testing.T) {
	// workers <= 0 with empty items → workers = len(items) = 0 → early return
	out := ParMap([]int{}, 0, func(v int) int { return v * 2 })
	if len(out) != 0 {
		t.Fatal("expected empty output")
	}
}

func TestParMap_NegativeWorkers(t *testing.T) {
	// workers <= 0 → workers = len(items), then proceeds normally
	out := ParMap([]int{1, 2, 3}, -1, func(v int) int { return v * 2 })
	if len(out) != 3 || out[0] != 2 || out[1] != 4 || out[2] != 6 {
		t.Fatalf("unexpected: %v", out)
	}
}

// --- ParMapResult edge cases ---

func TestParMapResult_EmptySlice(t *testing.T) {
	out := ParMapResult([]int{}, 4, func(v int) Result[int] { return Ok(v) })
	if len(out) != 0 {
		t.Fatal("expected empty output")
	}
}

func TestParMapResult_WorkersZero(t *testing.T) {
	out := ParMapResult([]int{}, 0, func(v int) Result[int] { return Ok(v) })
	if len(out) != 0 {
		t.Fatal("expected empty output")
	}
}

func TestParMapResult_NegativeWorkers(t *testing.T) {
	out := ParMapResult([]int{1, 2}, -1, func(v int) Result[int] { return Ok(v * 3) })
	if len(out) != 2 || !out[0].IsOk() || out[0].Must() != 3 {
		t.Fatal("unexpected result")
	}
}

func TestParMapResult_WithErrors(t *testing.T) {
	out := ParMapResult([]int{1, 2, 3}, 2, func(v int) Result[int] {
		if v == 2 {
			return Err[int](errors.New("fail"))
		}
		return Ok(v)
	})
	if len(out) != 3 {
		t.Fatal("expected 3 results")
	}
	if !out[1].IsErr() {
		t.Fatal("expected error for index 1")
	}
}

// --- Pipeline edge cases ---

func TestPipeline_Empty(t *testing.T) {
	p := Pipeline[int]()
	r := p(context.Background(), 42)
	if r.Must() != 42 {
		t.Fatal("empty pipeline should pass through")
	}
}

func TestPipeline_ErrorStopsEarly(t *testing.T) {
	calls := 0
	s1 := func(_ context.Context, v int) Result[int] {
		calls++
		return Err[int](errors.New("fail"))
	}
	s2 := func(_ context.Context, v int) Result[int] {
		calls++
		return Ok(v + 1)
	}
	p := Pipeline(s1, s2)
	r := p(context.Background(), 1)
	if r.IsOk() {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestPipeline_MultipleStages(t *testing.T) {
	add := func(_ context.Context, v int) Result[int] { return Ok(v + 1) }
	double := func(_ context.Context, v int) Result[int] { return Ok(v * 2) }
	p := Pipeline(add, double, add)
	r := p(context.Background(), 5)
	// (5+1)*2+1 = 13
	if r.Must() != 13 {
		t.Fatalf("expected 13, got %d", r.Must())
	}
}

// --- MapResult on error ---

func TestMapResult_OnError(t *testing.T) {
	r := MapResult(Err[int](errors.New("bad")), func(v int) string { return "nope" })
	if r.IsOk() {
		t.Fatal("MapResult on Err should be Err")
	}
	_, err := r.Unwrap()
	if err.Error() != "bad" {
		t.Fatalf("wrong error: %v", err)
	}
}

// --- Retry edge cases ---

func TestRetry_ContextCancelledBeforeSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	opts := RetryOpts{
		MaxAttempts: 5,
		InitialWait: time.Hour, // long wait, will be cancelled
		MaxWait:     time.Hour,
		Jitter:      false,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	r := Retry(ctx, opts, func(ctx context.Context) Result[int] {
		attempts++
		return Err[int](errors.New("fail"))
	})
	if r.IsOk() {
		t.Fatal("expected error")
	}
	_, err := r.Unwrap()
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRetry_ContextCancelledBeforeFirstSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	opts := RetryOpts{
		MaxAttempts: 3,
		InitialWait: time.Millisecond,
		MaxWait:     time.Millisecond,
		Jitter:      false,
	}

	r := Retry(ctx, opts, func(ctx context.Context) Result[int] {
		return Err[int](errors.New("fail"))
	})
	if r.IsOk() {
		t.Fatal("expected error")
	}
	_, err := r.Unwrap()
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRetry_NoJitter(t *testing.T) {
	opts := RetryOpts{
		MaxAttempts: 2,
		InitialWait: time.Millisecond,
		MaxWait:     time.Millisecond,
		Jitter:      false,
	}

	attempts := 0
	r := Retry(context.Background(), opts, func(ctx context.Context) Result[int] {
		attempts++
		if attempts < 2 {
			return Err[int](errors.New("fail"))
		}
		return Ok(42)
	})
	if r.Must() != 42 {
		t.Fatal("expected success")
	}
}

func TestRetry_MaxWaitCap(t *testing.T) {
	opts := RetryOpts{
		MaxAttempts: 3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     5 * time.Millisecond, // lower than initial
		Jitter:      false,
	}

	attempts := 0
	r := Retry(context.Background(), opts, func(ctx context.Context) Result[int] {
		attempts++
		if attempts < 3 {
			return Err[int](errors.New("fail"))
		}
		return Ok(1)
	})
	if r.Must() != 1 {
		t.Fatal("expected success on 3rd attempt")
	}
}

func TestRetry_AllFail(t *testing.T) {
	opts := RetryOpts{
		MaxAttempts: 2,
		InitialWait: time.Millisecond,
		MaxWait:     time.Millisecond,
		Jitter:      true,
	}

	r := Retry(context.Background(), opts, func(ctx context.Context) Result[int] {
		return Err[int](errors.New("always fail"))
	})
	if r.IsOk() {
		t.Fatal("expected error")
	}
}

// --- Then error propagation ---

func TestThen_FirstStageError(t *testing.T) {
	first := func(_ context.Context, v int) Result[string] {
		return Err[string](errors.New("first failed"))
	}
	second := func(_ context.Context, v string) Result[bool] {
		t.Fatal("should not be called")
		return Ok(true)
	}
	composed := Then(first, second)
	r := composed(context.Background(), 42)
	if r.IsOk() {
		t.Fatal("expected error from first stage")
	}
}

// --- RetryStage ---

func TestRetryStage_SuccessAfterRetry(t *testing.T) {
	attempts := 0
	stage := func(_ context.Context, v int) Result[int] {
		attempts++
		if attempts < 2 {
			return Err[int](errors.New("fail"))
		}
		return Ok(v * 2)
	}
	opts := RetryOpts{MaxAttempts: 3, InitialWait: time.Millisecond, MaxWait: time.Millisecond}
	rs := RetryStage(opts, stage)
	r := rs(context.Background(), 5)
	if r.Must() != 10 {
		t.Fatal("expected 10")
	}
}

// --- TracedStage error path ---

func TestTracedStage_Error(t *testing.T) {
	stage := func(_ context.Context, v int) Result[int] {
		return Err[int](errors.New("trace-fail"))
	}
	ts := TracedStage("test-stage", stage)
	r := ts(context.Background(), 1)
	if r.IsOk() {
		t.Fatal("expected error")
	}
}

func TestTracedStage_Success(t *testing.T) {
	stage := func(_ context.Context, v int) Result[int] {
		return Ok(v + 1)
	}
	ts := TracedStage("ok-stage", stage)
	r := ts(context.Background(), 1)
	if r.Must() != 2 {
		t.Fatal("expected 2")
	}
}

// --- BatchStage ---

func TestBatchStage_WithError(t *testing.T) {
	stage := func(_ context.Context, v int) Result[int] {
		if v == 2 {
			return Err[int](errors.New("fail"))
		}
		return Ok(v * 10)
	}
	bs := BatchStage(2, stage)
	r := bs(context.Background(), []int{1, 2, 3})
	if r.IsOk() {
		t.Fatal("expected error from batch")
	}
}

// --- MapStage ---

func TestMapStage_Simple(t *testing.T) {
	ms := MapStage(func(v int) string { return "x" })
	r := ms(context.Background(), 1)
	if r.Must() != "x" {
		t.Fatal("expected x")
	}
}

// --- TapStage ---

func TestTapStage_SideEffect(t *testing.T) {
	called := false
	ts := TapStage(func(_ context.Context, v int) {
		called = true
	})
	r := ts(context.Background(), 42)
	if r.Must() != 42 || !called {
		t.Fatal("TapStage should pass through and call side-effect")
	}
}
