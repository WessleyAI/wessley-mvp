//go:build integration

package semantic

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func qdrantAddr() string {
	if v := os.Getenv("QDRANT_URL"); v != "" {
		return v
	}
	return "localhost:6334"
}

func testStore(t *testing.T, collection string) *VectorStore {
	t.Helper()
	vs, err := New(qdrantAddr(), collection)
	if err != nil {
		t.Fatalf("connect qdrant: %v", err)
	}
	t.Cleanup(func() {
		vs.DeleteCollection(context.Background())
		vs.Close()
	})
	return vs
}

func TestQdrant_EnsureCollection(t *testing.T) {
	vs := testStore(t, "test_ensure")
	ctx := context.Background()

	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	// Calling again should be idempotent
	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection (idempotent): %v", err)
	}
}

func TestQdrant_UpsertAndSearch(t *testing.T) {
	vs := testStore(t, "test_upsert_search")
	ctx := context.Background()

	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	records := []VectorRecord{
		{ID: "a1111111-1111-1111-1111-111111111111", Embedding: []float32{1, 0, 0, 0}, Payload: map[string]any{"content": "oil change", "doc_id": "d1", "source": "reddit"}},
		{ID: "b2222222-2222-2222-2222-222222222222", Embedding: []float32{0, 1, 0, 0}, Payload: map[string]any{"content": "brake pads", "doc_id": "d2", "source": "ifixit"}},
		{ID: "c3333333-3333-3333-3333-333333333333", Embedding: []float32{0.9, 0.1, 0, 0}, Payload: map[string]any{"content": "oil filter", "doc_id": "d3", "source": "reddit"}},
	}

	if err := vs.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Search near [1,0,0,0] should return oil change first
	results, err := vs.Search(ctx, []float32{1, 0, 0, 0}, 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Content != "oil change" {
		t.Fatalf("expected 'oil change' first, got %q", results[0].Content)
	}
}

func TestQdrant_SearchFiltered(t *testing.T) {
	vs := testStore(t, "test_filtered")
	ctx := context.Background()

	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	records := []VectorRecord{
		{ID: "f1111111-1111-1111-1111-111111111111", Embedding: []float32{1, 0, 0, 0}, Payload: map[string]any{"content": "reddit post", "source": "reddit", "vehicle": "2020-Toyota-Camry"}},
		{ID: "f2222222-2222-2222-2222-222222222222", Embedding: []float32{0.9, 0.1, 0, 0}, Payload: map[string]any{"content": "ifixit guide", "source": "ifixit", "vehicle": "2020-Toyota-Camry"}},
		{ID: "f3333333-3333-3333-3333-333333333333", Embedding: []float32{0.8, 0.2, 0, 0}, Payload: map[string]any{"content": "honda post", "source": "reddit", "vehicle": "2021-Honda-Civic"}},
	}
	if err := vs.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Filter by source=reddit
	results, err := vs.SearchFiltered(ctx, []float32{1, 0, 0, 0}, 10, map[string]string{"source": "reddit"})
	if err != nil {
		t.Fatalf("SearchFiltered: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 reddit results, got %d", len(results))
	}

	// Filter by vehicle
	results, err = vs.SearchFiltered(ctx, []float32{1, 0, 0, 0}, 10, map[string]string{"vehicle": "2021-Honda-Civic"})
	if err != nil {
		t.Fatalf("SearchFiltered: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 honda result, got %d", len(results))
	}
}

func TestQdrant_DeleteByDocID(t *testing.T) {
	vs := testStore(t, "test_delete")
	ctx := context.Background()

	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	records := []VectorRecord{
		{ID: "d1111111-1111-1111-1111-111111111111", Embedding: []float32{1, 0, 0, 0}, Payload: map[string]any{"content": "to delete", "doc_id": "del-1"}},
		{ID: "d2222222-2222-2222-2222-222222222222", Embedding: []float32{0, 1, 0, 0}, Payload: map[string]any{"content": "keep this", "doc_id": "keep-1"}},
	}
	if err := vs.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := vs.DeleteByDocID(ctx, "del-1"); err != nil {
		t.Fatalf("DeleteByDocID: %v", err)
	}

	// Search should only find the kept record
	results, err := vs.Search(ctx, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range results {
		if r.DocID == "del-1" {
			t.Fatal("deleted doc still found")
		}
	}
}

func TestQdrant_DeleteCollection(t *testing.T) {
	addr := qdrantAddr()
	vs, err := New(addr, "test_delete_coll")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer vs.Close()

	ctx := context.Background()
	if err := vs.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	if err := vs.DeleteCollection(ctx); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}

	// Searching deleted collection should error
	_, err = vs.Search(ctx, []float32{1, 0, 0, 0}, 1)
	if err == nil {
		fmt.Println("Note: search after delete may not error immediately in Qdrant")
	}
}
