package repo

import (
	"testing"
)

// TestNeo4jRepoCompileCheck verifies the interface is satisfied at compile time.
// The actual var _ check is in neo4j.go. This test ensures defaults are set correctly.
func TestNewNeo4jRepoDefaults(t *testing.T) {
	// We can't run Neo4j integration tests without a driver, but we verify construction.
	// The compile-time check in neo4j.go ensures interface compliance.

	// Verify WithIDKey option works by constructing with nil driver (won't call any methods).
	r := NewNeo4jRepo[map[string]any, string](
		nil,
		"TestNode",
		func(m map[string]any) map[string]any { return m },
		nil,
		WithIDKey[map[string]any, string]("uuid"),
	)
	if r.idKey != "uuid" {
		t.Fatalf("expected idKey=uuid, got %s", r.idKey)
	}
	if r.label != "TestNode" {
		t.Fatalf("expected label=TestNode, got %s", r.label)
	}
}

func TestNewNeo4jRepoDefaultIDKey(t *testing.T) {
	r := NewNeo4jRepo[map[string]any, string](nil, "Node", nil, nil)
	if r.idKey != "id" {
		t.Fatalf("expected default idKey=id, got %s", r.idKey)
	}
}
