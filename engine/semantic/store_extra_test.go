package semantic

import (
	"context"
	"testing"
)

func TestUpsertEmptySlice(t *testing.T) {
	store := &VectorStore{collection: "test"}
	if err := store.Upsert(context.Background(), []VectorRecord{}); err != nil {
		t.Errorf("Upsert empty slice: %v", err)
	}
}

func TestSearchResultFields(t *testing.T) {
	sr := SearchResult{
		ID:      "id1",
		Score:   0.95,
		Content: "some content",
		DocID:   "doc1",
		Source:  "reddit",
		Meta:    map[string]string{"key": "val"},
	}
	if sr.ID != "id1" || sr.Score != 0.95 || sr.Content != "some content" {
		t.Error("field mismatch")
	}
	if sr.Meta["key"] != "val" {
		t.Error("meta mismatch")
	}
}

func TestVectorRecordFields(t *testing.T) {
	vr := VectorRecord{
		ID:        "uuid-1",
		Embedding: []float32{0.1, 0.2, 0.3},
		Payload:   map[string]any{"content": "text", "count": 5},
	}
	if vr.ID != "uuid-1" {
		t.Error("ID mismatch")
	}
	if len(vr.Embedding) != 3 {
		t.Error("embedding length mismatch")
	}
	if vr.Payload["content"] != "text" {
		t.Error("payload mismatch")
	}
}

func TestFieldMatch(t *testing.T) {
	cond := fieldMatch("vehicle", "toyota")
	fc := cond.GetField()
	if fc == nil {
		t.Fatal("expected field condition")
	}
	if fc.Key != "vehicle" {
		t.Fatalf("expected key=vehicle, got %s", fc.Key)
	}
	if fc.Match.GetKeyword() != "toyota" {
		t.Fatalf("expected keyword=toyota, got %s", fc.Match.GetKeyword())
	}
}
