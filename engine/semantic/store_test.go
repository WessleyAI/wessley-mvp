package semantic

import (
	"context"
	"os"
	"testing"
)

// Integration test — requires QDRANT_ADDR env (e.g. localhost:6334).
// Skipped by default; run with: QDRANT_ADDR=localhost:6334 go test ./engine/semantic/
func qdrantAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("QDRANT_ADDR")
	if addr == "" {
		t.Skip("QDRANT_ADDR not set, skipping integration test")
	}
	return addr
}

func TestUpsertAndSearch(t *testing.T) {
	addr := qdrantAddr(t)
	ctx := context.Background()

	store, err := New(addr, "test_semantic")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer store.Close()
	defer store.DeleteCollection(ctx) //nolint:errcheck

	if err := store.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	records := []VectorRecord{
		{ID: "00000000-0000-0000-0000-000000000001", Embedding: []float32{1, 0, 0, 0}, Payload: map[string]any{"content": "oil change", "doc_id": "doc1", "source": "reddit", "vehicle_make": "toyota"}},
		{ID: "00000000-0000-0000-0000-000000000002", Embedding: []float32{0, 1, 0, 0}, Payload: map[string]any{"content": "brake pads", "doc_id": "doc2", "source": "youtube", "vehicle_make": "honda"}},
		{ID: "00000000-0000-0000-0000-000000000003", Embedding: []float32{0.9, 0.1, 0, 0}, Payload: map[string]any{"content": "synthetic oil", "doc_id": "doc1", "source": "reddit", "vehicle_make": "toyota"}},
	}

	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Unfiltered search — should return closest to [1,0,0,0].
	results, err := store.Search(ctx, []float32{1, 0, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Content != "oil change" {
		t.Errorf("expected 'oil change', got %q", results[0].Content)
	}

	// Filtered search — only honda.
	filtered, err := store.SearchFiltered(ctx, []float32{1, 0, 0, 0}, 10, map[string]string{"vehicle_make": "honda"})
	if err != nil {
		t.Fatalf("SearchFiltered: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(filtered))
	}
	if filtered[0].Content != "brake pads" {
		t.Errorf("expected 'brake pads', got %q", filtered[0].Content)
	}

	// DeleteByDocID — remove doc1.
	if err := store.DeleteByDocID(ctx, "doc1"); err != nil {
		t.Fatalf("DeleteByDocID: %v", err)
	}
	afterDel, err := store.Search(ctx, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if len(afterDel) != 1 {
		t.Fatalf("expected 1 result after delete, got %d", len(afterDel))
	}
}

func TestUpsertEmpty(t *testing.T) {
	// Upsert with empty records should be a no-op (no connection needed).
	store := &VectorStore{collection: "test"}
	if err := store.Upsert(context.Background(), nil); err != nil {
		t.Errorf("Upsert(nil): %v", err)
	}
}

func TestEnsureCollectionIdempotent(t *testing.T) {
	addr := qdrantAddr(t)
	ctx := context.Background()

	store, err := New(addr, "test_idempotent")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer store.Close()
	defer store.DeleteCollection(ctx) //nolint:errcheck

	if err := store.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("first EnsureCollection: %v", err)
	}
	// Second call should be a no-op.
	if err := store.EnsureCollection(ctx, 4); err != nil {
		t.Fatalf("second EnsureCollection: %v", err)
	}
}
