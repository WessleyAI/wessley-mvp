# Spec: pkg-fn — Generic Functional Toolkit

**Branch:** `spec/pkg-fn`  
**Effort:** 3-4 days (1 dev)  
**Priority:** P0 — Foundation, everything depends on this

---

## Scope

Implement `pkg/fn/` — the generic functional toolkit used across all services. This is the core building block for pipelines, error handling, and collection processing.

### Files to Create

```
pkg/fn/
├── result.go      # Result[T] type
├── slice.go       # Map, Filter, FilterMap, Reduce, GroupBy, Chunk, Unique, UniqueBy, FlatMap
├── parallel.go    # ParMap, ParMapResult, FanOut, FanOutResult
├── pipeline.go    # Stage[In,Out], Then, Pipeline, BatchStage, MapStage, TapStage
├── retry.go       # RetryOpts, DefaultRetry, Retry, RetryStage
└── fn_test.go     # Comprehensive tests
```

---

## Key Types & Interfaces

### Result[T] (`result.go`)

```go
type Result[T any] struct {
	val T
	err error
	ok  bool
}

func Ok[T any](v T) Result[T]
func Err[T any](err error) Result[T]
func Errf[T any](format string, args ...any) Result[T]

func (r Result[T]) IsOk() bool
func (r Result[T]) IsErr() bool
func (r Result[T]) Unwrap() (T, error)
func (r Result[T]) Must() T                        // panics on error
func (r Result[T]) UnwrapOr(fallback T) T
func (r Result[T]) Map(f func(T) T) Result[T]
func (r Result[T]) AndThen(f func(T) Result[T]) Result[T]

func MapResult[T, U any](r Result[T], f func(T) U) Result[U]
func FromPair[T any](v T, err error) Result[T]
func Collect[T any](results []Result[T]) Result[[]T]  // first error wins
```

### Slice Operations (`slice.go`)

```go
func Map[T, U any](items []T, f func(T) U) []U
func Filter[T any](items []T, pred func(T) bool) []T
func FilterMap[T, U any](items []T, f func(T) (U, bool)) []U
func Reduce[T, Acc any](items []T, init Acc, f func(Acc, T) Acc) Acc
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T
func Chunk[T any](items []T, n int) [][]T
func Unique[T comparable](items []T) []T
func UniqueBy[T any, K comparable](items []T, key func(T) K) []T
func FlatMap[T, U any](items []T, f func(T) []U) []U
```

### Parallel (`parallel.go`)

```go
func ParMap[T, U any](items []T, workers int, f func(T) U) []U
func ParMapResult[T, U any](items []T, workers int, f func(T) Result[U]) []Result[U]
func FanOut[T any](fns ...func() T) []T
func FanOutResult[T any](fns ...func() Result[T]) Result[[]T]
```

- `ParMap` uses bounded concurrency via semaphore channel
- Results preserve input order (index-based assignment)
- `workers <= 0` means unbounded

### Pipeline (`pipeline.go`)

```go
type Stage[In, Out any] func(context.Context, In) Result[Out]

func Then[A, B, C any](first Stage[A, B], second Stage[B, C]) Stage[A, C]
func Pipeline[T any](stages ...Stage[T, T]) Stage[T, T]
func BatchStage[T, U any](workers int, stage Stage[T, U]) Stage[[]T, []U]
func MapStage[In, Out any](f func(In) Out) Stage[In, Out]
func TapStage[T any](f func(context.Context, T)) Stage[T, T]
```

### Retry (`retry.go`)

```go
type RetryOpts struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Jitter      bool
}

var DefaultRetry = RetryOpts{MaxAttempts: 3, InitialWait: time.Second, MaxWait: 30 * time.Second, Jitter: true}

func Retry[T any](ctx context.Context, opts RetryOpts, f func(context.Context) Result[T]) Result[T]
func RetryStage[In, Out any](opts RetryOpts, stage Stage[In, Out]) Stage[In, Out]
```

---

## Acceptance Criteria

- [ ] All types and functions implemented exactly as specified
- [ ] `Result[T]` fields unexported — access only through methods
- [ ] `Must()` panics with stored error on `IsErr()`
- [ ] `Collect` returns first error (left-to-right)
- [ ] `ParMap` preserves order — `out[i]` corresponds to `items[i]`
- [ ] `Chunk` with `n <= 0` returns nil
- [ ] `Then` short-circuits — second stage never called if first errors
- [ ] `Retry` respects `ctx.Done()` between attempts
- [ ] Unit tests for every function with edge cases (empty slices, context cancellation, etc.)
- [ ] No external dependencies (stdlib only)
- [ ] `go vet` and `golangci-lint` clean

## Dependencies

None — this is the foundation package.

## Reference

FINAL_ARCHITECTURE.md §5 — contains complete implementation code.
