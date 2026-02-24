package fn

import "sync"

// ParMap applies f to each item with bounded concurrency, preserving order.
func ParMap[T, U any](items []T, workers int, f func(T) U) []U {
	out := make([]U, len(items))
	var wg sync.WaitGroup

	if workers <= 0 {
		workers = len(items)
	}
	if workers == 0 {
		return out
	}

	sem := make(chan struct{}, workers)
	for i, v := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, v T) {
			defer func() { <-sem; wg.Done() }()
			out[i] = f(v)
		}(i, v)
	}
	wg.Wait()
	return out
}

// ParMapResult applies f with bounded concurrency, returning Results in order.
func ParMapResult[T, U any](items []T, workers int, f func(T) Result[U]) []Result[U] {
	out := make([]Result[U], len(items))
	var wg sync.WaitGroup

	if workers <= 0 {
		workers = len(items)
	}
	if workers == 0 {
		return out
	}

	sem := make(chan struct{}, workers)
	for i, v := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, v T) {
			defer func() { <-sem; wg.Done() }()
			out[i] = f(v)
		}(i, v)
	}
	wg.Wait()
	return out
}

// FanOut runs functions concurrently and returns results in order.
func FanOut[T any](fns ...func() T) []T {
	out := make([]T, len(fns))
	var wg sync.WaitGroup
	for i, f := range fns {
		wg.Add(1)
		go func(i int, f func() T) {
			defer wg.Done()
			out[i] = f()
		}(i, f)
	}
	wg.Wait()
	return out
}

// FanOutResult runs functions concurrently; returns first error or all values.
func FanOutResult[T any](fns ...func() Result[T]) Result[[]T] {
	results := make([]Result[T], len(fns))
	var wg sync.WaitGroup
	for i, f := range fns {
		wg.Add(1)
		go func(i int, f func() Result[T]) {
			defer wg.Done()
			results[i] = f()
		}(i, f)
	}
	wg.Wait()
	return Collect(results)
}
