package fn

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

// --- Result ---

func TestOkAndErr(t *testing.T) {
	r := Ok(42)
	if !r.IsOk() || r.IsErr() {
		t.Fatal("Ok should be ok")
	}
	v, err := r.Unwrap()
	if v != 42 || err != nil {
		t.Fatal("wrong unwrap")
	}

	e := Err[int](errors.New("fail"))
	if e.IsOk() || !e.IsErr() {
		t.Fatal("Err should be err")
	}
}

func TestErrf(t *testing.T) {
	r := Errf[string]("code %d", 404)
	_, err := r.Unwrap()
	if err == nil || err.Error() != "code 404" {
		t.Fatal("Errf wrong message")
	}
}

func TestMustPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Must should panic on Err")
		}
	}()
	Err[int](errors.New("boom")).Must()
}

func TestMustOk(t *testing.T) {
	v := Ok(7).Must()
	if v != 7 {
		t.Fatal("Must should return value")
	}
}

func TestUnwrapOr(t *testing.T) {
	if Ok(1).UnwrapOr(9) != 1 {
		t.Fatal("should return value")
	}
	if Err[int](errors.New("x")).UnwrapOr(9) != 9 {
		t.Fatal("should return fallback")
	}
}

func TestResultMap(t *testing.T) {
	r := Ok(2).Map(func(v int) int { return v * 3 })
	if r.Must() != 6 {
		t.Fatal("Map failed")
	}
	e := Err[int](errors.New("x")).Map(func(v int) int { return v * 3 })
	if e.IsOk() {
		t.Fatal("Map on Err should stay Err")
	}
}

func TestAndThen(t *testing.T) {
	r := Ok(2).AndThen(func(v int) Result[int] { return Ok(v + 1) })
	if r.Must() != 3 {
		t.Fatal("AndThen failed")
	}
	e := Err[int](errors.New("x")).AndThen(func(v int) Result[int] { return Ok(v + 1) })
	if e.IsOk() {
		t.Fatal("AndThen on Err should stay Err")
	}
}

func TestMapResult(t *testing.T) {
	r := MapResult(Ok(5), func(v int) string { return strconv.Itoa(v) })
	if r.Must() != "5" {
		t.Fatal("MapResult failed")
	}
}

func TestFromPair(t *testing.T) {
	r := FromPair(strconv.Atoi("42"))
	if r.Must() != 42 {
		t.Fatal("FromPair failed")
	}
	e := FromPair(strconv.Atoi("nope"))
	if e.IsOk() {
		t.Fatal("FromPair should fail")
	}
}

func TestCollect(t *testing.T) {
	all := Collect([]Result[int]{Ok(1), Ok(2), Ok(3)})
	v := all.Must()
	if len(v) != 3 || v[0] != 1 {
		t.Fatal("Collect failed")
	}

	bad := Collect([]Result[int]{Ok(1), Err[int](errors.New("e1")), Err[int](errors.New("e2"))})
	_, err := bad.Unwrap()
	if err == nil || err.Error() != "e1" {
		t.Fatal("Collect should return first error")
	}

	empty := Collect([]Result[int]{})
	if !empty.IsOk() || len(empty.Must()) != 0 {
		t.Fatal("Collect empty should be ok")
	}
}

// --- Slice ---

func TestMap(t *testing.T) {
	out := Map([]int{1, 2, 3}, func(v int) int { return v * 2 })
	if len(out) != 3 || out[2] != 6 {
		t.Fatal("Map failed")
	}
	empty := Map([]int{}, func(v int) int { return v })
	if len(empty) != 0 {
		t.Fatal("Map empty failed")
	}
}

func TestFilter(t *testing.T) {
	out := Filter([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })
	if len(out) != 2 || out[0] != 2 {
		t.Fatal("Filter failed")
	}
}

func TestFilterMap(t *testing.T) {
	out := FilterMap([]string{"1", "x", "3"}, func(s string) (int, bool) {
		v, err := strconv.Atoi(s)
		return v, err == nil
	})
	if len(out) != 2 || out[1] != 3 {
		t.Fatal("FilterMap failed")
	}
}

func TestReduce(t *testing.T) {
	sum := Reduce([]int{1, 2, 3}, 0, func(acc, v int) int { return acc + v })
	if sum != 6 {
		t.Fatal("Reduce failed")
	}
}

func TestGroupBy(t *testing.T) {
	g := GroupBy([]int{1, 2, 3, 4}, func(v int) string {
		if v%2 == 0 {
			return "even"
		}
		return "odd"
	})
	if len(g["even"]) != 2 || len(g["odd"]) != 2 {
		t.Fatal("GroupBy failed")
	}
}

func TestChunk(t *testing.T) {
	c := Chunk([]int{1, 2, 3, 4, 5}, 2)
	if len(c) != 3 || len(c[2]) != 1 {
		t.Fatal("Chunk failed")
	}
	if Chunk([]int{1}, 0) != nil {
		t.Fatal("Chunk n<=0 should return nil")
	}
	if Chunk([]int{1}, -1) != nil {
		t.Fatal("Chunk negative should return nil")
	}
}

func TestUnique(t *testing.T) {
	out := Unique([]int{1, 2, 2, 3, 1})
	if len(out) != 3 || out[0] != 1 || out[1] != 2 || out[2] != 3 {
		t.Fatal("Unique failed")
	}
}

func TestUniqueBy(t *testing.T) {
	type item struct {
		id   int
		name string
	}
	out := UniqueBy([]item{{1, "a"}, {2, "b"}, {1, "c"}}, func(i item) int { return i.id })
	if len(out) != 2 {
		t.Fatal("UniqueBy failed")
	}
}

func TestFlatMap(t *testing.T) {
	out := FlatMap([]int{1, 2, 3}, func(v int) []int { return []int{v, v * 10} })
	if len(out) != 6 || out[1] != 10 {
		t.Fatal("FlatMap failed")
	}
}

// --- Parallel ---

func TestParMap(t *testing.T) {
	out := ParMap([]int{1, 2, 3, 4}, 2, func(v int) int { return v * 2 })
	for i, v := range out {
		if v != (i+1)*2 {
			t.Fatalf("ParMap order broken at %d", i)
		}
	}
}

func TestParMapEmpty(t *testing.T) {
	out := ParMap([]int{}, 2, func(v int) int { return v })
	if len(out) != 0 {
		t.Fatal("ParMap empty should return empty")
	}
}

func TestParMapUnbounded(t *testing.T) {
	out := ParMap([]int{1, 2, 3}, 0, func(v int) int { return v + 1 })
	if out[0] != 2 || out[2] != 4 {
		t.Fatal("ParMap unbounded failed")
	}
}

func TestParMapResult(t *testing.T) {
	out := ParMapResult([]int{1, 2, 3}, 2, func(v int) Result[int] { return Ok(v * 2) })
	for i, r := range out {
		if r.Must() != (i+1)*2 {
			t.Fatal("ParMapResult failed")
		}
	}
}

func TestFanOut(t *testing.T) {
	out := FanOut(func() int { return 1 }, func() int { return 2 })
	if out[0] != 1 || out[1] != 2 {
		t.Fatal("FanOut failed")
	}
}

func TestFanOutResult(t *testing.T) {
	r := FanOutResult(func() Result[int] { return Ok(1) }, func() Result[int] { return Ok(2) })
	v := r.Must()
	if v[0] != 1 || v[1] != 2 {
		t.Fatal("FanOutResult failed")
	}

	e := FanOutResult(func() Result[int] { return Ok(1) }, func() Result[int] { return Err[int](errors.New("fail")) })
	if e.IsOk() {
		t.Fatal("FanOutResult should fail")
	}
}

// --- Pipeline ---

func TestThen(t *testing.T) {
	double := Stage[int, int](func(_ context.Context, v int) Result[int] { return Ok(v * 2) })
	addOne := Stage[int, int](func(_ context.Context, v int) Result[int] { return Ok(v + 1) })

	composed := Then(double, addOne)
	r := composed(context.Background(), 5)
	if r.Must() != 11 {
		t.Fatal("Then failed")
	}
}

func TestThenShortCircuits(t *testing.T) {
	fail := Stage[int, int](func(_ context.Context, _ int) Result[int] { return Err[int](errors.New("fail")) })
	called := false
	second := Stage[int, int](func(_ context.Context, v int) Result[int] {
		called = true
		return Ok(v)
	})

	r := Then(fail, second)(context.Background(), 1)
	if r.IsOk() || called {
		t.Fatal("Then should short-circuit")
	}
}

func TestPipeline(t *testing.T) {
	inc := Stage[int, int](func(_ context.Context, v int) Result[int] { return Ok(v + 1) })
	p := Pipeline(inc, inc, inc)
	if p(context.Background(), 0).Must() != 3 {
		t.Fatal("Pipeline failed")
	}
}

func TestMapStage(t *testing.T) {
	s := MapStage(func(v int) string { return strconv.Itoa(v) })
	r := s(context.Background(), 42)
	if r.Must() != "42" {
		t.Fatal("MapStage failed")
	}
}

func TestTapStage(t *testing.T) {
	var captured int
	s := TapStage(func(_ context.Context, v int) { captured = v })
	r := s(context.Background(), 7)
	if r.Must() != 7 || captured != 7 {
		t.Fatal("TapStage failed")
	}
}

func TestBatchStage(t *testing.T) {
	double := Stage[int, int](func(_ context.Context, v int) Result[int] { return Ok(v * 2) })
	batch := BatchStage(2, double)
	r := batch(context.Background(), []int{1, 2, 3})
	v := r.Must()
	if len(v) != 3 || v[0] != 2 || v[2] != 6 {
		t.Fatal("BatchStage failed")
	}
}

func TestTracedStage(t *testing.T) {
	s := TracedStage("test-stage", Stage[int, int](func(_ context.Context, v int) Result[int] { return Ok(v + 1) }))
	r := s(context.Background(), 1)
	if r.Must() != 2 {
		t.Fatal("TracedStage failed")
	}

	// Error case
	e := TracedStage("err-stage", Stage[int, int](func(_ context.Context, _ int) Result[int] { return Err[int](errors.New("x")) }))
	if e(context.Background(), 1).IsOk() {
		t.Fatal("TracedStage error should propagate")
	}
}

// --- Retry ---

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	r := Retry(context.Background(), RetryOpts{MaxAttempts: 3, InitialWait: time.Millisecond, Jitter: false}, func(_ context.Context) Result[int] {
		attempts++
		if attempts < 3 {
			return Err[int](errors.New("not yet"))
		}
		return Ok(42)
	})
	if r.Must() != 42 || attempts != 3 {
		t.Fatal("Retry should succeed on 3rd attempt")
	}
}

func TestRetryExhausted(t *testing.T) {
	r := Retry(context.Background(), RetryOpts{MaxAttempts: 2, InitialWait: time.Millisecond, Jitter: false}, func(_ context.Context) Result[int] {
		return Err[int](errors.New("fail"))
	})
	if r.IsOk() {
		t.Fatal("Retry should fail after exhausting attempts")
	}
}

func TestRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	r := Retry(ctx, RetryOpts{MaxAttempts: 100, InitialWait: 10 * time.Millisecond, Jitter: false}, func(ctx context.Context) Result[int] {
		attempts++
		return Err[int](errors.New("fail"))
	})
	if r.IsOk() {
		t.Fatal("Retry should fail on context cancel")
	}
}

func TestRetryStage(t *testing.T) {
	attempts := 0
	s := RetryStage(RetryOpts{MaxAttempts: 2, InitialWait: time.Millisecond, Jitter: false},
		Stage[int, int](func(_ context.Context, v int) Result[int] {
			attempts++
			if attempts < 2 {
				return Err[int](errors.New("fail"))
			}
			return Ok(v * 2)
		}))
	r := s(context.Background(), 5)
	if r.Must() != 10 {
		t.Fatal("RetryStage failed")
	}
}
