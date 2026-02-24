package repo

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// --- Mock infrastructure ---

type mockResult struct {
	records []*neo4j.Record
	idx     int
}

func (m *mockResult) Next(ctx context.Context) bool {
	if m.idx < len(m.records) {
		m.idx++
		return true
	}
	return false
}

func (m *mockResult) Record() *neo4j.Record {
	return m.records[m.idx-1]
}

type mockRunner struct {
	result  *mockResult
	err     error
	cyphers []string
}

func (m *mockRunner) Run(ctx context.Context, cypher string, params map[string]any) (result, error) {
	m.cyphers = append(m.cyphers, cypher)
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockRunner) Close(ctx context.Context) error { return nil }

// helper types

type entity struct {
	ID   string
	Name string
}

func makeRecord(id, name string) *neo4j.Record {
	return &neo4j.Record{
		Values: []any{map[string]any{"id": id, "name": name}},
		Keys:   []string{"n"},
	}
}

func newTestRepo(r *mockRunner) *Neo4jRepo[entity, string] {
	repo := NewNeo4jRepo[entity, string](
		nil, "Entity",
		func(e entity) map[string]any { return map[string]any{"id": e.ID, "name": e.Name} },
		func(rec *neo4j.Record) (entity, error) {
			if len(rec.Values) == 0 {
				return entity{}, errors.New("empty")
			}
			m, ok := rec.Values[0].(map[string]any)
			if !ok {
				return entity{}, errors.New("bad type")
			}
			return entity{ID: m["id"].(string), Name: m["name"].(string)}, nil
		},
	)
	repo.newSession = func(ctx context.Context) runner { return r }
	return repo
}

// --- Tests ---

func TestGet_Success(t *testing.T) {
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{makeRecord("1", "Alice")}}}
	repo := newTestRepo(r)

	e, err := repo.Get(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "1" || e.Name != "Alice" {
		t.Fatalf("got %+v", e)
	}
}

func TestGet_NotFound(t *testing.T) {
	r := &mockRunner{result: &mockResult{}}
	repo := newTestRepo(r)
	_, err := repo.Get(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGet_RunError(t *testing.T) {
	r := &mockRunner{err: errors.New("db down")}
	repo := newTestRepo(r)
	_, err := repo.Get(context.Background(), "x")
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected db down, got %v", err)
	}
}

func TestList_Success(t *testing.T) {
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{makeRecord("1", "A"), makeRecord("2", "B")}}}
	repo := newTestRepo(r)

	items, err := repo.List(context.Background(), ListOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
}

func TestList_DefaultLimit(t *testing.T) {
	r := &mockRunner{result: &mockResult{}}
	repo := newTestRepo(r)
	_, err := repo.List(context.Background(), ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestList_RunError(t *testing.T) {
	r := &mockRunner{err: errors.New("fail")}
	repo := newTestRepo(r)
	_, err := repo.List(context.Background(), ListOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestList_FromRecordError(t *testing.T) {
	bad := &neo4j.Record{Values: []any{"not a map"}, Keys: []string{"n"}}
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{bad}}}
	repo := newTestRepo(r)
	_, err := repo.List(context.Background(), ListOpts{Limit: 10})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_Success(t *testing.T) {
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{makeRecord("3", "C")}}}
	repo := newTestRepo(r)
	e, err := repo.Create(context.Background(), entity{ID: "3", Name: "C"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "C" {
		t.Fatalf("got %+v", e)
	}
}

func TestCreate_RunError(t *testing.T) {
	r := &mockRunner{err: errors.New("fail")}
	repo := newTestRepo(r)
	_, err := repo.Create(context.Background(), entity{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_NoResult(t *testing.T) {
	r := &mockRunner{result: &mockResult{}}
	repo := newTestRepo(r)
	_, err := repo.Create(context.Background(), entity{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_Success(t *testing.T) {
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{makeRecord("1", "Updated")}}}
	repo := newTestRepo(r)
	e, err := repo.Update(context.Background(), entity{ID: "1", Name: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "Updated" {
		t.Fatalf("got %+v", e)
	}
}

func TestUpdate_RunError(t *testing.T) {
	r := &mockRunner{err: errors.New("fail")}
	repo := newTestRepo(r)
	_, err := repo.Update(context.Background(), entity{ID: "1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	r := &mockRunner{result: &mockResult{}}
	repo := newTestRepo(r)
	_, err := repo.Update(context.Background(), entity{ID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_Success(t *testing.T) {
	r := &mockRunner{result: &mockResult{}}
	repo := newTestRepo(r)
	err := repo.Delete(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDelete_RunError(t *testing.T) {
	r := &mockRunner{err: errors.New("fail")}
	repo := newTestRepo(r)
	err := repo.Delete(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCypherGeneration(t *testing.T) {
	r := &mockRunner{result: &mockResult{records: []*neo4j.Record{makeRecord("1", "A")}}}
	repo := NewNeo4jRepo[entity, string](
		nil, "Vehicle",
		func(e entity) map[string]any { return map[string]any{"vin": e.ID, "name": e.Name} },
		func(rec *neo4j.Record) (entity, error) {
			m := rec.Values[0].(map[string]any)
			return entity{ID: m["id"].(string), Name: m["name"].(string)}, nil
		},
		WithIDKey[entity, string]("vin"),
	)
	repo.newSession = func(ctx context.Context) runner {
		// Reset result index for each call
		r.result = &mockResult{records: []*neo4j.Record{makeRecord("1", "A")}}
		return r
	}

	ctx := context.Background()
	repo.Get(ctx, "ABC")
	repo.List(ctx, ListOpts{Limit: 50})
	repo.Create(ctx, entity{ID: "ABC", Name: "A"})
	repo.Update(ctx, entity{ID: "ABC", Name: "A"})
	repo.Delete(ctx, "ABC")

	expected := []string{
		"MATCH (n:Vehicle {vin: $id}) RETURN n",
		"MATCH (n:Vehicle) RETURN n SKIP $offset LIMIT $limit",
		"CREATE (n:Vehicle $props) RETURN n",
		"MATCH (n:Vehicle {vin: $id}) SET n += $props RETURN n",
		"MATCH (n:Vehicle {vin: $id}) DELETE n",
	}

	if len(r.cyphers) != len(expected) {
		t.Fatalf("got %d cyphers, want %d", len(r.cyphers), len(expected))
	}
	for i, want := range expected {
		if r.cyphers[i] != want {
			t.Errorf("[%d] got %q, want %q", i, r.cyphers[i], want)
		}
	}
}

func TestSessionFallback(t *testing.T) {
	// When newSession is nil, session() should call driver.NewSession
	// We can't test this without a real driver, but verify newSession=nil path
	repo := NewNeo4jRepo[entity, string](nil, "X", nil, nil)
	if repo.newSession != nil {
		t.Fatal("newSession should be nil by default")
	}
}
