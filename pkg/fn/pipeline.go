package fn

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// Stage is a function that transforms In to Out within a context.
type Stage[In, Out any] func(context.Context, In) Result[Out]

// Then composes two stages, short-circuiting on error. Both stages get child spans.
func Then[A, B, C any](first Stage[A, B], second Stage[B, C]) Stage[A, C] {
	return func(ctx context.Context, a A) Result[C] {
		ctx1, span1 := otel.Tracer("pkg/fn").Start(ctx, "stage.first")
		r := first(ctx1, a)
		span1.End()
		if r.IsErr() {
			_, err := r.Unwrap()
			return Err[C](err)
		}
		ctx2, span2 := otel.Tracer("pkg/fn").Start(ctx, "stage.second")
		defer span2.End()
		v, _ := r.Unwrap()
		return second(ctx2, v)
	}
}

// Pipeline composes multiple same-typed stages.
func Pipeline[T any](stages ...Stage[T, T]) Stage[T, T] {
	return func(ctx context.Context, t T) Result[T] {
		r := Ok(t)
		for _, s := range stages {
			if r.IsErr() {
				return r
			}
			v, _ := r.Unwrap()
			r = s(ctx, v)
		}
		return r
	}
}

// BatchStage runs a stage over a slice with bounded concurrency.
func BatchStage[T, U any](workers int, stage Stage[T, U]) Stage[[]T, []U] {
	return func(ctx context.Context, items []T) Result[[]U] {
		results := ParMapResult(items, workers, func(item T) Result[U] {
			return stage(ctx, item)
		})
		return Collect(results)
	}
}

// MapStage wraps a pure function as a Stage.
func MapStage[In, Out any](f func(In) Out) Stage[In, Out] {
	return func(_ context.Context, in In) Result[Out] {
		return Ok(f(in))
	}
}

// TapStage runs a side-effect and passes the value through.
func TapStage[T any](f func(context.Context, T)) Stage[T, T] {
	return func(ctx context.Context, t T) Result[T] {
		f(ctx, t)
		return Ok(t)
	}
}

// TracedStage wraps a stage with OTel span creation.
func TracedStage[In, Out any](name string, stage Stage[In, Out]) Stage[In, Out] {
	return func(ctx context.Context, in In) Result[Out] {
		ctx, span := otel.Tracer("pkg/fn").Start(ctx, name)
		defer span.End()
		result := stage(ctx, in)
		if result.IsErr() {
			_, err := result.Unwrap()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return result
	}
}
