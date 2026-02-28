package fn

import (
	"context"
	"errors"
	"testing"
)

// --- Additional Result tests ---

func TestErrZeroValue(t *testing.T) {
	r := Err[string](errors.New("x"))
	v, _ := r.Unwrap()
	if v != "" {
		t.Fatal("Err value should be zero")
	}
}

func TestResultMapChangeType(t *testing.T) {
	r := MapResult(Err[int](errors.New("boom")), func(v int) string { return "x" })
	if r.IsOk() {
		t.Fatal("MapResult on Err should stay Err")
	}
	_, err := r.Unwrap()
	if err.Error() != "boom" {
		t.Fatal("error should propagate through MapResult")
	}
}

func TestCollectSingleError(t *testing.T) {
	r := Collect([]Result[int]{Err[int](errors.New("only"))})
	_, err := r.Unwrap()
	if err == nil || err.Error() != "only" {
		t.Fatal("Collect single error")
	}
}

// --- Additional Slice tests ---

func TestFilterNoneMatch(t *testing.T) {
	out := Filter([]int{1, 3, 5}, func(v int) bool { return v%2 == 0 })
	if len(out) != 0 {
		t.Fatal("Filter should return empty when none match")
	}
}

func TestReduceEmpty(t *testing.T) {
	sum := Reduce([]int{}, 10, func(acc, v int) int { return acc + v })
	if sum != 10 {
		t.Fatal("Reduce empty should return init")
	}
}

func TestGroupByEmpty(t *testing.T) {
	g := GroupBy([]int{}, func(v int) string { return "x" })
	if len(g) != 0 {
		t.Fatal("GroupBy empty should return empty map")
	}
}

func TestChunkExact(t *testing.T) {
	c := Chunk([]int{1, 2, 3, 4}, 2)
	if len(c) != 2 || len(c[0]) != 2 || len(c[1]) != 2 {
		t.Fatal("Chunk exact division")
	}
}

func TestChunkSingleElement(t *testing.T) {
	c := Chunk([]int{1}, 5)
	if len(c) != 1 || len(c[0]) != 1 {
		t.Fatal("Chunk single element")
	}
}

func TestUniqueEmpty(t *testing.T) {
	out := Unique([]int{})
	if len(out) != 0 {
		t.Fatal("Unique empty should return empty")
	}
}

func TestFlatMapEmpty(t *testing.T) {
	out := FlatMap([]int{}, func(v int) []int { return []int{v} })
	if len(out) != 0 {
		t.Fatal("FlatMap empty should return empty")
	}
}

func TestFilterMapNoneMatch(t *testing.T) {
	out := FilterMap([]string{"a", "b"}, func(s string) (int, bool) { return 0, false })
	if len(out) != 0 {
		t.Fatal("FilterMap none match should return empty")
	}
}

// --- Additional Pipeline tests ---

func TestPipelineEmpty(t *testing.T) {
	p := Pipeline[int]()
	r := p(context.Background(), 42)
	if r.Must() != 42 {
		t.Fatal("Pipeline with no stages should pass through")
	}
}

func TestPipelineShortCircuits(t *testing.T) {
	called := false
	fail := Stage[int, int](func(_ context.Context, _ int) Result[int] { return Err[int](errors.New("fail")) })
	track := Stage[int, int](func(_ context.Context, v int) Result[int] {
		called = true
		return Ok(v)
	})
	p := Pipeline(fail, track)
	r := p(context.Background(), 1)
	if r.IsOk() {
		t.Fatal("Pipeline should short-circuit on error")
	}
	if called {
		t.Fatal("second stage should not be called after error")
	}
}

func TestBatchStageWithError(t *testing.T) {
	fail := Stage[int, int](func(_ context.Context, v int) Result[int] {
		if v == 2 {
			return Err[int](errors.New("fail on 2"))
		}
		return Ok(v * 2)
	})
	batch := BatchStage(2, fail)
	r := batch(context.Background(), []int{1, 2, 3})
	if r.IsOk() {
		t.Fatal("BatchStage should fail if any item fails")
	}
}

// --- Additional Parallel tests ---

func TestParMapSingleWorker(t *testing.T) {
	out := ParMap([]int{1, 2, 3}, 1, func(v int) int { return v * 2 })
	if out[0] != 2 || out[1] != 4 || out[2] != 6 {
		t.Fatal("ParMap single worker failed")
	}
}

func TestFanOutSingle(t *testing.T) {
	out := FanOut(func() int { return 42 })
	if len(out) != 1 || out[0] != 42 {
		t.Fatal("FanOut single")
	}
}

func TestFanOutResultAllErrors(t *testing.T) {
	r := FanOutResult(
		func() Result[int] { return Err[int](errors.New("e1")) },
		func() Result[int] { return Err[int](errors.New("e2")) },
	)
	if r.IsOk() {
		t.Fatal("FanOutResult all errors should fail")
	}
}

// --- Additional Retry tests ---

func TestRetryImmediateSuccess(t *testing.T) {
	r := Retry(context.Background(), RetryOpts{MaxAttempts: 3, InitialWait: 0, Jitter: false}, func(_ context.Context) Result[int] {
		return Ok(1)
	})
	if r.Must() != 1 {
		t.Fatal("Retry immediate success")
	}
}

func TestRetryMaxAttemptsOne(t *testing.T) {
	r := Retry(context.Background(), RetryOpts{MaxAttempts: 1, InitialWait: 0, Jitter: false}, func(_ context.Context) Result[int] {
		return Err[int](errors.New("fail"))
	})
	if r.IsOk() {
		t.Fatal("Retry with 1 attempt should fail")
	}
}
