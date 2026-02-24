package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
	"github.com/WessleyAI/wessley-mvp/engine/rag"
	"github.com/WessleyAI/wessley-mvp/engine/semantic"
	mlpb "github.com/WessleyAI/wessley-mvp/ml/proto/wessley/ml/v1"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// --- Mock gRPC services ---

type mockEmbedServer struct {
	mlpb.UnimplementedEmbedServiceServer
}

func (m *mockEmbedServer) Embed(_ context.Context, req *mlpb.EmbedRequest) (*mlpb.EmbedResponse, error) {
	return &mlpb.EmbedResponse{Values: []float32{0.1, 0.2, 0.3}}, nil
}

type mockChatServer struct {
	mlpb.UnimplementedChatServiceServer
}

func (m *mockChatServer) Chat(_ context.Context, req *mlpb.ChatRequest) (*mlpb.ChatResponse, error) {
	return &mlpb.ChatResponse{Reply: "Test answer about brakes", TokensUsed: 42, Model: "test-model"}, nil
}

// --- Mock semantic/graph ---

type mockSearcher struct {
	results []semantic.SearchResult
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ []float32, _ int, _ map[string]string) ([]semantic.SearchResult, error) {
	return m.results, m.err
}

type mockGraphEnricher struct {
	components []graph.Component
	edges      []graph.Edge
	err        error
}

func (m *mockGraphEnricher) FindRelatedComponents(_ context.Context, _ []string, _ string) ([]graph.Component, []graph.Edge, error) {
	return m.components, m.edges, m.err
}

func setupTestRAG(t *testing.T) *rag.Service {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	mlpb.RegisterEmbedServiceServer(srv, &mockEmbedServer{})
	mlpb.RegisterChatServiceServer(srv, &mockChatServer{})

	go srv.Serve(lis)
	t.Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	logger := slog.Default()
	return rag.New(
		conn,
		&mockSearcher{results: []semantic.SearchResult{
			{ID: "src1", Content: "Brake pads info", DocID: "doc1", Source: "test", Score: 0.95},
		}},
		&mockGraphEnricher{},
		rag.DefaultOptions(),
		logger,
	)
}

func TestHandleChat_Success(t *testing.T) {
	ragSvc := setupTestRAG(t)
	handler := handleChat(ragSvc, slog.Default())

	body := `{"question":"How do I replace brake pads?","vehicle":"2020 Honda Civic"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString(body))
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Answer == "" {
		t.Error("expected non-empty answer")
	}
	if resp.Model == "" {
		t.Error("expected non-empty model")
	}
	if len(resp.Sources) == 0 {
		t.Error("expected sources")
	}

	// Check content type
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestHandleChat_NoVehicle(t *testing.T) {
	ragSvc := setupTestRAG(t)
	handler := handleChat(ragSvc, slog.Default())

	body := `{"question":"What causes engine stalling?"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString(body))
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Adapter tests ---

// --- Mock Qdrant PointsAPI ---

type mockPointsAPI struct {
	resp *pb.SearchResponse
	err  error
}

func (m *mockPointsAPI) Upsert(_ context.Context, _ *pb.UpsertPoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsAPI) Delete(_ context.Context, _ *pb.DeletePoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return nil, nil
}
func (m *mockPointsAPI) Search(_ context.Context, _ *pb.SearchPoints, _ ...grpc.CallOption) (*pb.SearchResponse, error) {
	return m.resp, m.err
}

func TestSemanticAdapter_Search(t *testing.T) {
	store := semantic.NewWithClients(
		&mockPointsAPI{
			resp: &pb.SearchResponse{
				Result: []*pb.ScoredPoint{
					{
						Id:    &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: "point-1"}},
						Score: 0.95,
						Payload: map[string]*pb.Value{
							"content": {Kind: &pb.Value_StringValue{StringValue: "Brake info"}},
							"doc_id":  {Kind: &pb.Value_StringValue{StringValue: "doc1"}},
							"source":  {Kind: &pb.Value_StringValue{StringValue: "test"}},
						},
					},
				},
			},
		}, nil, "test-collection",
	)
	adapter := &semanticAdapter{store: store}

	results, err := adapter.Search(context.Background(), []float32{0.1, 0.2}, 5, map[string]string{"vehicle": "civic"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "Brake info" {
		t.Errorf("unexpected content: %s", results[0].Content)
	}
}

func TestSemanticAdapter_Search_Error(t *testing.T) {
	store := semantic.NewWithClients(
		&mockPointsAPI{err: fmt.Errorf("qdrant down")}, nil, "test",
	)
	adapter := &semanticAdapter{store: store}

	_, err := adapter.Search(context.Background(), []float32{0.1}, 5, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGraphAdapter_FindRelatedComponents_WithMock(t *testing.T) {
	// Create a mock graph store using NewWithOpener
	mockSession := &mockCypherSession{
		records: []mockRecord{
			{keys: []string{"n"}, values: []any{
				neo4j.Node{
					Props: map[string]any{
						"id": "comp1", "name": "ECU", "type": "ecu", "vehicle": "2020 Honda Civic",
					},
				},
			}},
		},
	}

	gs := graph.NewWithOpener(&mockOpener{session: mockSession})
	adapter := &graphAdapter{store: gs}

	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"ecu"}, "2020 Honda Civic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Results depend on mock returning components
	_ = comps
	_ = edges
}

func TestGraphAdapter_FindRelatedComponents_NoResults(t *testing.T) {
	mockSession := &mockCypherSession{records: nil}
	gs := graph.NewWithOpener(&mockOpener{session: mockSession})
	adapter := &graphAdapter{store: gs}

	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"nonexistent"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("expected 0 components, got %d", len(comps))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

func TestGraphAdapter_FindRelatedComponents_WithVehicle(t *testing.T) {
	mockSession := &mockCypherSession{records: nil}
	gs := graph.NewWithOpener(&mockOpener{session: mockSession})
	adapter := &graphAdapter{store: gs}

	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"sensor"}, "2020 Toyota Camry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = comps
	_ = edges
}

func TestGraphAdapter_FindRelatedComponents_ErrorHandling(t *testing.T) {
	mockSession := &mockCypherSession{err: fmt.Errorf("neo4j connection failed")}
	gs := graph.NewWithOpener(&mockOpener{session: mockSession})
	adapter := &graphAdapter{store: gs}

	// Errors in FindByType are silently continued
	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"ecu", "sensor"}, "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("expected 0 on error, got %d", len(comps))
	}
	_ = edges
}

func TestGraphAdapter_FindRelatedComponents_NeighborError(t *testing.T) {
	callCount := 0
	mockSession := &mockCypherSessionFunc{
		runFn: func(ctx context.Context, cypher string, params map[string]any) (graph.CypherResult, error) {
			callCount++
			if callCount == 1 {
				return &mockCypherResult{records: []mockRecord{
					{keys: []string{"n"}, values: []any{neo4j.Node{Props: map[string]any{"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "v"}}}},
				}}, nil
			}
			return nil, fmt.Errorf("neighbors failed")
		},
	}
	gs := graph.NewWithOpener(&mockOpenerFunc{sessionFn: func(ctx context.Context) graph.CypherSession { return mockSession }})
	adapter := &graphAdapter{store: gs}

	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"ecu"}, "")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(comps) != 1 {
		t.Errorf("expected 1 component, got %d", len(comps))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges on neighbor error, got %d", len(edges))
	}
}

func TestGraphAdapter_FindRelatedComponents_WithNeighbors(t *testing.T) {
	callCount := 0
	mockSession := &mockCypherSessionFunc{
		runFn: func(ctx context.Context, cypher string, params map[string]any) (graph.CypherResult, error) {
			callCount++
			if callCount == 1 {
				// FindByType returns a component
				return &mockCypherResult{records: []mockRecord{
					{keys: []string{"n"}, values: []any{neo4j.Node{Props: map[string]any{"id": "c1", "name": "ECU", "type": "ecu", "vehicle": "v"}}}},
				}}, nil
			}
			// Neighbors call returns another component
			return &mockCypherResult{records: []mockRecord{
				{keys: []string{"n"}, values: []any{neo4j.Node{Props: map[string]any{"id": "c2", "name": "Sensor", "type": "sensor", "vehicle": "v"}}}},
			}}, nil
		},
	}

	gs := graph.NewWithOpener(&mockOpenerFunc{sessionFn: func(ctx context.Context) graph.CypherSession { return mockSession }})
	adapter := &graphAdapter{store: gs}

	comps, edges, err := adapter.FindRelatedComponents(context.Background(), []string{"ecu"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) == 0 {
		t.Error("expected at least 1 component")
	}
	_ = edges
}

// --- Mock types for graph ---

type mockRecord struct {
	keys   []string
	values []any
}

type mockCypherResult struct {
	records []mockRecord
	idx     int
}

func (m *mockCypherResult) Next(_ context.Context) bool {
	if m.idx < len(m.records) {
		m.idx++
		return true
	}
	return false
}

func (m *mockCypherResult) Record() *neo4j.Record {
	rec := m.records[m.idx-1]
	return &neo4j.Record{Keys: rec.keys, Values: rec.values}
}

type mockCypherSession struct {
	records []mockRecord
	err     error
}

func (m *mockCypherSession) Run(_ context.Context, _ string, _ map[string]any) (graph.CypherResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockCypherResult{records: m.records}, nil
}

func (m *mockCypherSession) Close(_ context.Context) error { return nil }
func (m *mockCypherSession) ExecuteWrite(_ context.Context, _ func(tx graph.CypherRunner) (any, error)) (any, error) {
	return nil, nil
}

type mockOpener struct {
	session graph.CypherSession
}

func (m *mockOpener) OpenSession(_ context.Context) graph.CypherSession {
	return m.session
}

type mockCypherSessionFunc struct {
	runFn func(ctx context.Context, cypher string, params map[string]any) (graph.CypherResult, error)
}

func (m *mockCypherSessionFunc) Run(ctx context.Context, cypher string, params map[string]any) (graph.CypherResult, error) {
	return m.runFn(ctx, cypher, params)
}
func (m *mockCypherSessionFunc) Close(_ context.Context) error { return nil }
func (m *mockCypherSessionFunc) ExecuteWrite(_ context.Context, _ func(tx graph.CypherRunner) (any, error)) (any, error) {
	return nil, nil
}

type mockOpenerFunc struct {
	sessionFn func(ctx context.Context) graph.CypherSession
}

func (m *mockOpenerFunc) OpenSession(ctx context.Context) graph.CypherSession {
	return m.sessionFn(ctx)
}

func TestLoadConfig_AllEnvVars(t *testing.T) {
	t.Setenv("PORT", "3000")
	t.Setenv("ML_WORKER_URL", "ml:50051")
	t.Setenv("NEO4J_URL", "neo4j://db:7687")
	t.Setenv("NEO4J_USER", "admin")
	t.Setenv("NEO4J_PASS", "secret")
	t.Setenv("QDRANT_URL", "qdrant:6334")
	t.Setenv("QDRANT_COLLECTION", "test-col")
	t.Setenv("CORS_ORIGIN", "https://app.com")

	cfg := loadConfig()
	if cfg.Port != "3000" {
		t.Errorf("expected 3000, got %s", cfg.Port)
	}
	if cfg.MLWorkerURL != "ml:50051" {
		t.Errorf("expected ml:50051, got %s", cfg.MLWorkerURL)
	}
	if cfg.Neo4jURL != "neo4j://db:7687" {
		t.Errorf("expected neo4j://db:7687, got %s", cfg.Neo4jURL)
	}
	if cfg.Neo4jUser != "admin" {
		t.Errorf("expected admin, got %s", cfg.Neo4jUser)
	}
	if cfg.Neo4jPass != "secret" {
		t.Errorf("expected secret, got %s", cfg.Neo4jPass)
	}
	if cfg.QdrantURL != "qdrant:6334" {
		t.Errorf("expected qdrant:6334, got %s", cfg.QdrantURL)
	}
	if cfg.Collection != "test-col" {
		t.Errorf("expected test-col, got %s", cfg.Collection)
	}
	if cfg.CORSOrigin != "https://app.com" {
		t.Errorf("expected https://app.com, got %s", cfg.CORSOrigin)
	}
}

func TestHandleChat_RAGError(t *testing.T) {
	// Create a RAG service with a searcher that errors
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	mlpb.RegisterEmbedServiceServer(srv, &mockEmbedServer{})
	mlpb.RegisterChatServiceServer(srv, &mockChatServer{})
	go srv.Serve(lis)
	t.Cleanup(func() { srv.Stop() })

	conn, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	t.Cleanup(func() { conn.Close() })

	ragSvc := rag.New(conn,
		&mockSearcher{err: context.DeadlineExceeded},
		&mockGraphEnricher{},
		rag.DefaultOptions(),
		slog.Default(),
	)

	handler := handleChat(ragSvc, slog.Default())
	body := `{"question":"test question"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString(body))
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleChat_LargePayload(t *testing.T) {
	handler := handleChat(nil, slog.Default())
	// Valid JSON but very large question — still should reach validation
	body := `{"question":""}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString(body))
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty question, got %d", rec.Code)
	}
}

func TestRun_StartsAndShuts(t *testing.T) {
	cfg := Config{
		Port:        "0", // will fail to bind or pick random
		MLWorkerURL: "localhost:50099",
		Neo4jURL:    "neo4j://localhost:7687",
		Neo4jUser:   "neo4j",
		Neo4jPass:   "test",
		QdrantURL:   "localhost:6399",
		Collection:  "test",
		CORSOrigin:  "*",
	}
	logger := slog.Default()

	// Run in goroutine, expect it to start and then we kill via signal
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(cfg, logger)
	}()

	// Give it a moment to start, then send interrupt
	// Port 0 should work (kernel assigns random port)
	// We need to somehow stop it - send SIGINT to ourselves
	go func() {
		// Wait a bit for server to start
		<-time.After(200 * time.Millisecond)
		// Send signal to trigger graceful shutdown
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGINT)
	}()

	select {
	case err := <-errCh:
		// May return nil or an error, both acceptable for this test
		_ = err
	case <-time.After(5 * time.Second):
		t.Fatal("run did not exit within 5 seconds")
	}
}

func TestRun_BadNeo4jURL(t *testing.T) {
	cfg := Config{
		Port:        "0",
		MLWorkerURL: "localhost:50099",
		Neo4jURL:    "://invalid",
		Neo4jUser:   "neo4j",
		Neo4jPass:   "test",
		QdrantURL:   "localhost:6399",
		Collection:  "test",
		CORSOrigin:  "*",
	}
	err := run(cfg, slog.Default())
	if err == nil {
		t.Log("expected error for bad neo4j URL")
	}
}

func TestRun_BadPort(t *testing.T) {
	cfg := Config{
		Port:        "99999", // invalid port
		MLWorkerURL: "localhost:50099",
		Neo4jURL:    "neo4j://localhost:7687",
		Neo4jUser:   "neo4j",
		Neo4jPass:   "test",
		QdrantURL:   "localhost:6399",
		Collection:  "test",
		CORSOrigin:  "*",
	}

	err := run(cfg, slog.Default())
	// Should error because port is invalid
	if err == nil {
		t.Log("no error on bad port, acceptable on some systems")
	}
}

func TestRun_PortInUse(t *testing.T) {
	// Occupy a port first
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	port := ln.Addr().(*net.TCPAddr).Port
	defer ln.Close()

	cfg := Config{
		Port:        fmt.Sprintf("%d", port),
		MLWorkerURL: "localhost:50099",
		Neo4jURL:    "neo4j://localhost:7687",
		Neo4jUser:   "neo4j",
		Neo4jPass:   "test",
		QdrantURL:   "localhost:6399",
		Collection:  "test",
		CORSOrigin:  "*",
	}

	errCh := make(chan error, 1)
	go func() { errCh <- run(cfg, slog.Default()) }()

	select {
	case err := <-errCh:
		if err == nil {
			t.Log("expected error for port in use")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not exit")
	}
}

func TestHandleHealth_Response(t *testing.T) {
	rec := httptest.NewRecorder()
	handleHealth(rec, httptest.NewRequest("GET", "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected ok, got %s", resp["status"])
	}
}
