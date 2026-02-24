package fn

// Map applies f to each element.
func Map[T, U any](items []T, f func(T) U) []U {
	out := make([]U, len(items))
	for i, v := range items {
		out[i] = f(v)
	}
	return out
}

// Filter returns elements where pred is true.
func Filter[T any](items []T, pred func(T) bool) []T {
	var out []T
	for _, v := range items {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// FilterMap applies f and keeps results where ok is true.
func FilterMap[T, U any](items []T, f func(T) (U, bool)) []U {
	var out []U
	for _, v := range items {
		if u, ok := f(v); ok {
			out = append(out, u)
		}
	}
	return out
}

// Reduce folds items into a single value.
func Reduce[T, Acc any](items []T, init Acc, f func(Acc, T) Acc) Acc {
	acc := init
	for _, v := range items {
		acc = f(acc, v)
	}
	return acc
}

// GroupBy groups items by a key function.
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T {
	out := make(map[K][]T)
	for _, v := range items {
		k := key(v)
		out[k] = append(out[k], v)
	}
	return out
}

// Chunk splits items into chunks of size n. Returns nil if n <= 0.
func Chunk[T any](items []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	var out [][]T
	for i := 0; i < len(items); i += n {
		end := i + n
		if end > len(items) {
			end = len(items)
		}
		out = append(out, items[i:end])
	}
	return out
}

// Unique returns unique elements preserving order.
func Unique[T comparable](items []T) []T {
	seen := make(map[T]struct{})
	var out []T
	for _, v := range items {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// UniqueBy returns elements with unique keys, preserving order.
func UniqueBy[T any, K comparable](items []T, key func(T) K) []T {
	seen := make(map[K]struct{})
	var out []T
	for _, v := range items {
		k := key(v)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// FlatMap applies f to each element and flattens the results.
func FlatMap[T, U any](items []T, f func(T) []U) []U {
	var out []U
	for _, v := range items {
		out = append(out, f(v)...)
	}
	return out
}
