// Package repo defines the generic Repository interface and list options.
package repo

import "context"

// Repository is a generic CRUD interface.
type Repository[T any, ID comparable] interface {
	Get(ctx context.Context, id ID) (T, error)
	List(ctx context.Context, opts ListOpts) ([]T, error)
	Create(ctx context.Context, entity T) (T, error)
	Update(ctx context.Context, entity T) (T, error)
	Delete(ctx context.Context, id ID) error
}

// ListOpts controls pagination and filtering for List operations.
type ListOpts struct {
	Offset int
	Limit  int
	Filter map[string]any
}
