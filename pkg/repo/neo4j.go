package repo

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// result is the minimal interface needed from a neo4j result.
type result interface {
	Next(ctx context.Context) bool
	Record() *neo4j.Record
}

// runner is the minimal interface needed from a neo4j session.
type runner interface {
	Run(ctx context.Context, cypher string, params map[string]any) (result, error)
	Close(ctx context.Context) error
}

// Neo4jRepo is a generic Neo4j-backed repository.
type Neo4jRepo[T any, ID comparable] struct {
	driver     neo4j.DriverWithContext
	label      string
	idKey      string
	toMap      func(T) map[string]any
	fromRecord func(*neo4j.Record) (T, error)
	newSession func(ctx context.Context) runner // for testing
}

// Neo4jOption configures a Neo4jRepo.
type Neo4jOption[T any, ID comparable] func(*Neo4jRepo[T, ID])

// WithIDKey sets the property name used as the ID (default "id").
func WithIDKey[T any, ID comparable](key string) Neo4jOption[T, ID] {
	return func(r *Neo4jRepo[T, ID]) { r.idKey = key }
}

// NewNeo4jRepo creates a new Neo4j-backed repository.
func NewNeo4jRepo[T any, ID comparable](
	driver neo4j.DriverWithContext,
	label string,
	toMap func(T) map[string]any,
	fromRecord func(*neo4j.Record) (T, error),
	opts ...Neo4jOption[T, ID],
) *Neo4jRepo[T, ID] {
	r := &Neo4jRepo[T, ID]{
		driver:     driver,
		label:      label,
		idKey:      "id",
		toMap:      toMap,
		fromRecord: fromRecord,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Compile-time interface check.
var _ Repository[any, string] = (*Neo4jRepo[any, string])(nil)

// neo4jSessionAdapter adapts neo4j.SessionWithContext to the runner interface.
type neo4jSessionAdapter struct {
	sess neo4j.SessionWithContext
}

func (a *neo4jSessionAdapter) Run(ctx context.Context, cypher string, params map[string]any) (result, error) {
	return a.sess.Run(ctx, cypher, params)
}

func (a *neo4jSessionAdapter) Close(ctx context.Context) error {
	return a.sess.Close(ctx)
}

func (r *Neo4jRepo[T, ID]) session(ctx context.Context) runner {
	if r.newSession != nil {
		return r.newSession(ctx)
	}
	return &neo4jSessionAdapter{sess: r.driver.NewSession(ctx, neo4j.SessionConfig{})}
}

func (r *Neo4jRepo[T, ID]) Get(ctx context.Context, id ID) (T, error) {
	var zero T
	sess := r.session(ctx)
	defer sess.Close(ctx)

	cypher := fmt.Sprintf("MATCH (n:%s {%s: $id}) RETURN n", r.label, r.idKey)
	result, err := sess.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return zero, err
	}
	if !result.Next(ctx) {
		return zero, fmt.Errorf("%s not found", r.label)
	}
	record := result.Record()
	return r.fromRecord(record)
}

func (r *Neo4jRepo[T, ID]) List(ctx context.Context, opts ListOpts) ([]T, error) {
	sess := r.session(ctx)
	defer sess.Close(ctx)

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	cypher := fmt.Sprintf("MATCH (n:%s) RETURN n SKIP $offset LIMIT $limit", r.label)
	params := map[string]any{"offset": opts.Offset, "limit": limit}

	result, err := sess.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}

	var items []T
	for result.Next(ctx) {
		item, err := r.fromRecord(result.Record())
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Neo4jRepo[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	var zero T
	sess := r.session(ctx)
	defer sess.Close(ctx)

	cypher := fmt.Sprintf("CREATE (n:%s $props) RETURN n", r.label)
	result, err := sess.Run(ctx, cypher, map[string]any{"props": r.toMap(entity)})
	if err != nil {
		return zero, err
	}
	if !result.Next(ctx) {
		return zero, fmt.Errorf("failed to create %s", r.label)
	}
	return r.fromRecord(result.Record())
}

func (r *Neo4jRepo[T, ID]) Update(ctx context.Context, entity T) (T, error) {
	var zero T
	sess := r.session(ctx)
	defer sess.Close(ctx)

	props := r.toMap(entity)
	cypher := fmt.Sprintf("MATCH (n:%s {%s: $id}) SET n += $props RETURN n", r.label, r.idKey)
	result, err := sess.Run(ctx, cypher, map[string]any{"id": props[r.idKey], "props": props})
	if err != nil {
		return zero, err
	}
	if !result.Next(ctx) {
		return zero, fmt.Errorf("%s not found", r.label)
	}
	return r.fromRecord(result.Record())
}

func (r *Neo4jRepo[T, ID]) Delete(ctx context.Context, id ID) error {
	sess := r.session(ctx)
	defer sess.Close(ctx)

	cypher := fmt.Sprintf("MATCH (n:%s {%s: $id}) DELETE n", r.label, r.idKey)
	_, err := sess.Run(ctx, cypher, map[string]any{"id": id})
	return err
}
