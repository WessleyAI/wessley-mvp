package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// --- Mocks ---

type mockRecord struct {
	values map[string]any
}

func (r *mockRecord) Get(key string) (any, bool) {
	v, ok := r.values[key]
	return v, ok
}

type mockResult struct {
	records []*neo4j.Record
	idx     int
}

func (r *mockResult) Next(_ context.Context) bool {
	if r.idx < len(r.records) {
		r.idx++
		return true
	}
	return false
}

func (r *mockResult) Record() *neo4j.Record {
	if r.idx <= 0 || r.idx > len(r.records) {
		return nil
	}
	return r.records[r.idx-1]
}

func newMockResult(records ...*neo4j.Record) *mockResult {
	return &mockResult{records: records}
}

type mockSession struct {
	runResult CypherResult
	runErr    error
	writeWork func(tx CypherRunner) (any, error)
	writeErr  error
	closed    bool
}

func (s *mockSession) Run(_ context.Context, _ string, _ map[string]any) (CypherResult, error) {
	return s.runResult, s.runErr
}

func (s *mockSession) Close(_ context.Context) error {
	s.closed = true
	return nil
}

func (s *mockSession) ExecuteWrite(_ context.Context, work func(tx CypherRunner) (any, error)) (any, error) {
	if s.writeErr != nil {
		return nil, s.writeErr
	}
	return work(&mockTx{})
}

type mockTx struct {
	runErr error
}

func (t *mockTx) Run(_ context.Context, _ string, _ map[string]any) (CypherResult, error) {
	return newMockResult(), t.runErr
}

type mockOpener struct {
	session *mockSession
}

func (o *mockOpener) OpenSession(_ context.Context) CypherSession {
	return o.session
}

// Helper to make a neo4j.Record with "n" field as a dbtype.Node
func makeNodeRecord(props map[string]any) *neo4j.Record {
	node := dbtype.Node{Props: props}
	return &neo4j.Record{
		Keys:   []string{"n"},
		Values: []any{node},
	}
}

func makeNodesRecord(nodes []dbtype.Node) *neo4j.Record {
	raw := make([]any, len(nodes))
	for i, n := range nodes {
		raw[i] = n
	}
	return &neo4j.Record{
		Keys:   []string{"nodes"},
		Values: []any{raw},
	}
}

// --- Pure function tests ---

func TestSanitizeRelType(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"connects_to", "CONNECTS_TO"},
		{"part_of", "PART_OF"},
		{"powers", "POWERS"},
		{"grounds", "GROUNDS"},
		{"", "RELATED_TO"},
		{"has-wire", "HASWIRE"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
		{"a1b2", "A1B2"},
		{"---", "RELATED_TO"},
		{"MiXeD_123", "MIXED_123"},
	}
	for _, tt := range tests {
		got := sanitizeRelType(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeRelType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestComponentFromProps(t *testing.T) {
	props := map[string]any{
		"id":         "c1",
		"name":       "ECU Main",
		"type":       "ecu",
		"vehicle":    "2020-Toyota-Camry",
		"prop_color": "blue",
		"prop_pin":   "A12",
	}
	c := componentFromProps(props)
	if c.ID != "c1" {
		t.Fatalf("expected id=c1, got %s", c.ID)
	}
	if c.Properties["color"] != "blue" {
		t.Fatalf("expected prop color=blue, got %s", c.Properties["color"])
	}
	if c.Properties["pin"] != "A12" {
		t.Fatalf("expected prop pin=A12, got %s", c.Properties["pin"])
	}
}

func TestComponentFromProps_Empty(t *testing.T) {
	c := componentFromProps(map[string]any{})
	if c.ID != "" || c.Name != "" || c.Type != "" {
		t.Fatal("expected empty component")
	}
	if len(c.Properties) != 0 {
		t.Fatal("expected no properties")
	}
}

func TestComponentFromProps_NonStringProp(t *testing.T) {
	props := map[string]any{
		"id":          "c1",
		"prop_number": 42, // not a string — should be skipped
		"prop_ok":     "yes",
	}
	c := componentFromProps(props)
	if _, ok := c.Properties["number"]; ok {
		t.Fatal("non-string prop should be skipped")
	}
	if c.Properties["ok"] != "yes" {
		t.Fatal("expected prop ok=yes")
	}
}

func TestComponentFromProps_ShortKeys(t *testing.T) {
	// Keys shorter than 5 chars shouldn't be treated as prop_
	props := map[string]any{
		"id":   "c1",
		"prop": "not_a_prop_key", // len == 4
	}
	c := componentFromProps(props)
	if len(c.Properties) != 0 {
		t.Fatal("expected no properties for short key")
	}
}

func TestComponentToMap(t *testing.T) {
	c := Component{
		ID:      "c1",
		Name:    "Fuse Box",
		Type:    "fuse",
		Vehicle: "2020-Honda-Civic",
		Properties: map[string]string{
			"rating": "15A",
		},
	}
	m := componentToMap(c)
	if m["id"] != "c1" {
		t.Fatal("missing id")
	}
	if m["prop_rating"] != "15A" {
		t.Fatal("missing prop_rating")
	}
}

func TestComponentToMap_NoProperties(t *testing.T) {
	c := Component{ID: "c1", Name: "X"}
	m := componentToMap(c)
	if m["id"] != "c1" || m["name"] != "X" {
		t.Fatal("basic fields missing")
	}
}

func TestNewGraphStore(t *testing.T) {
	gs := New(nil)
	if gs == nil {
		t.Fatal("expected non-nil GraphStore")
	}
	if gs.components == nil {
		t.Fatal("expected non-nil components repo")
	}
}

func TestStrProp(t *testing.T) {
	props := map[string]any{"a": "hello", "b": 42, "c": nil}
	if strProp(props, "a") != "hello" {
		t.Fatal("expected hello")
	}
	if strProp(props, "b") != "" {
		t.Fatal("non-string should return empty")
	}
	if strProp(props, "missing") != "" {
		t.Fatal("missing key should return empty")
	}
}

// --- GraphStore method tests with mocks ---

func TestSaveComponent_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveComponent(context.Background(), Component{ID: "c1", Name: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sess.closed {
		t.Fatal("session not closed")
	}
}

func TestSaveComponent_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("db error")}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveComponent(context.Background(), Component{ID: "c1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveEdge_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveEdge(context.Background(), Edge{ID: "e1", From: "a", To: "b", Type: "connects_to", Wire: "w1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveEdge_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveEdge(context.Background(), Edge{ID: "e1", From: "a", To: "b", Type: "connects_to"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNeighbors_Success(t *testing.T) {
	records := []*neo4j.Record{
		makeNodeRecord(map[string]any{"id": "n1", "name": "Relay", "type": "relay", "vehicle": "v1"}),
		makeNodeRecord(map[string]any{"id": "n2", "name": "Fuse", "type": "fuse", "vehicle": "v1"}),
	}
	sess := &mockSession{runResult: newMockResult(records...)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.Neighbors(context.Background(), "c1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(comps))
	}
	if comps[0].ID != "n1" {
		t.Errorf("expected n1, got %s", comps[0].ID)
	}
}

func TestNeighbors_DefaultDepth(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.Neighbors(context.Background(), "c1", 0) // should default to 1
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comps != nil {
		t.Fatalf("expected nil for no results, got %v", comps)
	}
}

func TestNeighbors_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.Neighbors(context.Background(), "c1", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByVehicle_Success(t *testing.T) {
	records := []*neo4j.Record{
		makeNodeRecord(map[string]any{"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "2020-Toyota-Camry"}),
	}
	sess := &mockSession{runResult: newMockResult(records...)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.FindByVehicle(context.Background(), 2020, "Toyota", "Camry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 1 || comps[0].ID != "c1" {
		t.Fatalf("unexpected results: %v", comps)
	}
}

func TestFindByVehicle_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.FindByVehicle(context.Background(), 2020, "Toyota", "Camry")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindByType_Success(t *testing.T) {
	records := []*neo4j.Record{
		makeNodeRecord(map[string]any{"id": "c1", "name": "Fuse", "type": "fuse", "vehicle": "v1"}),
	}
	sess := &mockSession{runResult: newMockResult(records...)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.FindByType(context.Background(), "fuse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1, got %d", len(comps))
	}
}

func TestFindByType_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.FindByType(context.Background(), "fuse")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTracePath_Success(t *testing.T) {
	nodes := []dbtype.Node{
		{Props: map[string]any{"id": "a", "name": "Start", "type": "ecu", "vehicle": "v1"}},
		{Props: map[string]any{"id": "b", "name": "End", "type": "sensor", "vehicle": "v1"}},
	}
	rec := makeNodesRecord(nodes)
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.TracePath(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}
}

func TestTracePath_NoPath(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()} // empty result
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error for no path")
	}
}

func TestTracePath_RunError(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTracePath_NoNodesField(t *testing.T) {
	rec := &neo4j.Record{Keys: []string{"other"}, Values: []any{"stuff"}}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error for missing nodes field")
	}
}

func TestTracePath_WrongNodesType(t *testing.T) {
	rec := &neo4j.Record{Keys: []string{"nodes"}, Values: []any{"not-a-list"}}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.TracePath(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error for wrong nodes type")
	}
}

func TestTracePath_NonNodeInList(t *testing.T) {
	// Mix of valid node and non-node items
	raw := []any{
		dbtype.Node{Props: map[string]any{"id": "a"}},
		"not-a-node",
	}
	rec := &neo4j.Record{Keys: []string{"nodes"}, Values: []any{raw}}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.TracePath(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 valid component, got %d", len(comps))
	}
}

func TestSaveBatch_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps := []Component{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	}
	edges := []Edge{
		{ID: "e1", From: "c1", To: "c2", Type: "connects_to", Wire: "w1"},
	}
	err := gs.SaveBatch(context.Background(), comps, edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveBatch_TxRunError(t *testing.T) {
	failTx := &mockSession{
		runResult: newMockResult(),
	}
	// Override ExecuteWrite to use a failing tx
	gs := NewWithOpener(&mockOpener{session: failTx})

	// Override: use a session that returns a tx error
	failSess := &mockSessionWithTxErr{err: errors.New("tx fail")}
	gs2 := NewWithOpener(&mockOpener2{session: failSess})

	err := gs2.SaveBatch(context.Background(), []Component{{ID: "c1"}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	_ = gs // suppress unused
}

func TestSaveBatch_Empty(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveBatch(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetComponent_WithOpener(t *testing.T) {
	rec := makeNodeRecord(map[string]any{"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "v1"})
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comp, err := gs.GetComponent(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.ID != "c1" {
		t.Fatalf("expected c1, got %s", comp.ID)
	}
}

func TestGetComponent_NotFound(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()} // no records
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestGetComponent_RunError(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetComponent_NoNField(t *testing.T) {
	rec := &neo4j.Record{Keys: []string{"other"}, Values: []any{"x"}}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error for missing n field")
	}
}

func TestGetComponent_WrongNType(t *testing.T) {
	rec := &neo4j.Record{Keys: []string{"n"}, Values: []any{"not-a-node"}}
	sess := &mockSession{runResult: newMockResult(rec)}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.GetComponent(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestCollectComponents_MixedRecords(t *testing.T) {
	// Record with valid node
	r1 := makeNodeRecord(map[string]any{"id": "c1", "name": "A", "type": "t", "vehicle": "v"})
	// Record with wrong "n" type
	r2 := &neo4j.Record{Keys: []string{"n"}, Values: []any{"not-node"}}
	// Record missing "n"
	r3 := &neo4j.Record{Keys: []string{"x"}, Values: []any{1}}

	result := newMockResult(r1, r2, r3)
	comps, err := collectComponents(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 valid component, got %d", len(comps))
	}
}

// --- Additional mock types for tx error testing ---

type mockSessionWithTxErr struct {
	err error
}

func (s *mockSessionWithTxErr) Run(_ context.Context, _ string, _ map[string]any) (CypherResult, error) {
	return newMockResult(), nil
}
func (s *mockSessionWithTxErr) Close(_ context.Context) error { return nil }
func (s *mockSessionWithTxErr) ExecuteWrite(_ context.Context, _ func(tx CypherRunner) (any, error)) (any, error) {
	return nil, s.err
}

type mockOpener2 struct {
	session CypherSession
}

func (o *mockOpener2) OpenSession(_ context.Context) CypherSession {
	return o.session
}

func TestComponentFromRecord(t *testing.T) {
	node := dbtype.Node{
		Props: map[string]any{
			"id":         "c1",
			"name":       "ECU",
			"type":       "ecu",
			"vehicle":    "v1",
			"prop_color": "red",
		},
	}
	rec := &neo4j.Record{Keys: []string{"n"}, Values: []any{node}}
	c, err := componentFromRecord(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != "c1" || c.Properties["color"] != "red" {
		t.Fatal("wrong component parsed")
	}
}
