package fn

import "fmt"

// Result[T] is a generic result type for error handling.
type Result[T any] struct {
	val T
	err error
	ok  bool
}

// Ok creates a successful Result.
func Ok[T any](v T) Result[T] {
	return Result[T]{val: v, ok: true}
}

// Err creates a failed Result from an error.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// Errf creates a failed Result from a formatted string.
func Errf[T any](format string, args ...any) Result[T] {
	return Result[T]{err: fmt.Errorf(format, args...)}
}

// IsOk returns true if the result is successful.
func (r Result[T]) IsOk() bool { return r.ok }

// IsErr returns true if the result is an error.
func (r Result[T]) IsErr() bool { return !r.ok }

// Unwrap returns the value and error.
func (r Result[T]) Unwrap() (T, error) { return r.val, r.err }

// Must returns the value or panics on error.
func (r Result[T]) Must() T {
	if !r.ok {
		panic(r.err)
	}
	return r.val
}

// UnwrapOr returns the value or a fallback on error.
func (r Result[T]) UnwrapOr(fallback T) T {
	if !r.ok {
		return fallback
	}
	return r.val
}

// Map transforms the value if ok.
func (r Result[T]) Map(f func(T) T) Result[T] {
	if !r.ok {
		return r
	}
	return Ok(f(r.val))
}

// AndThen chains a function that returns a Result.
func (r Result[T]) AndThen(f func(T) Result[T]) Result[T] {
	if !r.ok {
		return r
	}
	return f(r.val)
}

// MapResult transforms Result[T] to Result[U].
func MapResult[T, U any](r Result[T], f func(T) U) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return Ok(f(r.val))
}

// FromPair creates a Result from a (value, error) pair.
func FromPair[T any](v T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(v)
}

// Collect returns Ok with all values if all results are ok, or the first error.
func Collect[T any](results []Result[T]) Result[[]T] {
	out := make([]T, len(results))
	for i, r := range results {
		if !r.ok {
			return Err[[]T](r.err)
		}
		out[i] = r.val
	}
	return Ok(out)
}
