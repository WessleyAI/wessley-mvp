package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// --- GetComponent fallback path (no repo, via opener) ---

func TestGetComponent_Fallback_Success(t *testing.T) {
	rec := makeNodeRecord(map[string]any{
		"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "corolla",
	})
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	c, err := gs.GetComponent(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.ID != "c1" || c.Name != "ECU" {
		t.Fatalf("wrong component: %+v", c)
	}
}

func TestGetComponent_Fallback_RunError(t *testing.T) {
	sess := &mockSession{runErr: errors.New("run fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetComponent_Fallback_NotFound(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestGetComponent_Fallback_NoNField(t *testing.T) {
	// Record with key "x" instead of "n"
	rec := &neo4j.Record{
		Keys:   []string{"x"},
		Values: []any{"something"},
	}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error about no n field")
	}
}

func TestGetComponent_Fallback_WrongType(t *testing.T) {
	// Record has "n" but it's not a Node
	rec := &neo4j.Record{
		Keys:   []string{"n"},
		Values: []any{"not-a-node"},
	}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error about unexpected type")
	}
}

// --- SaveBatch error paths ---

func TestSaveBatch_WriteError(t *testing.T) {
	sess := &mockSession{writeErr: errors.New("write fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveBatch(context.Background(), []Component{{ID: "c1"}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveBatch_EdgeTxRunError(t *testing.T) {
	callCount := 0
	sess := &txErrorSession{
		failAt: 1, // fail on 2nd Run call (edge)
		count:  &callCount,
	}
	gs := NewWithOpener(&mockOpener2{session: sess})

	err := gs.SaveBatch(context.Background(),
		[]Component{{ID: "c1"}},
		[]Edge{{ID: "e1", From: "c1", To: "c2", Type: "CONNECTS"}},
	)
	if err == nil {
		t.Fatal("expected tx error")
	}
}

func TestSaveBatch_ComponentTxRunError(t *testing.T) {
	callCount := 0
	sess := &txErrorSession{
		failAt: 0, // fail on 1st Run call (component)
		count:  &callCount,
	}
	gs := NewWithOpener(&mockOpener2{session: sess})

	err := gs.SaveBatch(context.Background(),
		[]Component{{ID: "c1"}},
		nil,
	)
	if err == nil {
		t.Fatal("expected tx error")
	}
}

// --- componentFromProps with prop_ properties ---

func TestComponentFromProps_WithProps(t *testing.T) {
	props := map[string]any{
		"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "v1",
		"prop_color": "blue",
		"prop_size":  42, // non-string, should be skipped
	}
	c := componentFromProps(props)
	if c.Properties["color"] != "blue" {
		t.Fatal("missing color")
	}
	if _, ok := c.Properties["size"]; ok {
		t.Fatal("non-string prop_ should not be included")
	}
}

// --- componentToMap round-trip ---

func TestComponentToMap_WithProps(t *testing.T) {
	c := Component{
		ID: "c1", Name: "ECU", Type: "ecu", Vehicle: "v",
		Properties: map[string]string{"source": "manual"},
	}
	m := componentToMap(c)
	if m["prop_source"] != "manual" {
		t.Fatal("expected prop_source")
	}
}

// --- TracePath error paths ---

func TestTracePath_NoNodesField2(t *testing.T) {
	// Record without "nodes" key
	rec := &neo4j.Record{
		Keys:   []string{"other"},
		Values: []any{[]any{}},
	}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error about no nodes")
	}
}

func TestTracePath_WrongNodesType2(t *testing.T) {
	rec := &neo4j.Record{
		Keys:   []string{"nodes"},
		Values: []any{"not-a-list"},
	}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error about unexpected nodes type")
	}
}

func TestTracePath_MixedNodeTypes(t *testing.T) {
	// Some nodes are valid, some are not (non-Node items skipped)
	nodeList := []any{
		dbtype.Node{Props: map[string]any{"id": "c1", "name": "A", "type": "t", "vehicle": "v"}},
		"not-a-node", // should be skipped
		dbtype.Node{Props: map[string]any{"id": "c2", "name": "B", "type": "t", "vehicle": "v"}},
	}
	rec := &neo4j.Record{
		Keys:   []string{"nodes"},
		Values: []any{nodeList},
	}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.TracePath(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2, got %d", len(comps))
	}
}

// --- Helper mocks ---

type mockOpener3 struct {
	session CypherSession
}

func (o *mockOpener3) OpenSession(_ context.Context) CypherSession {
	return o.session
}

type txErrorSession struct {
	failAt int
	count  *int
}

func (s *txErrorSession) Run(_ context.Context, _ string, _ map[string]any) (CypherResult, error) {
	return newMockResult(), nil
}

func (s *txErrorSession) Close(_ context.Context) error { return nil }

func (s *txErrorSession) ExecuteWrite(_ context.Context, work func(tx CypherRunner) (any, error)) (any, error) {
	return work(&txErrorRunner{failAt: s.failAt, count: s.count})
}

type txErrorRunner struct {
	failAt int
	count  *int
}

func (r *txErrorRunner) Run(_ context.Context, _ string, _ map[string]any) (CypherResult, error) {
	current := *r.count
	*r.count++
	if current == r.failAt {
		return nil, errors.New("tx run error")
	}
	return newMockResult(), nil
}
