package semantic

import (
	"context"
	"errors"
	"testing"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// --- Mocks ---

type mockPoints struct {
	upsertResp *pb.PointsOperationResponse
	upsertErr  error
	deleteResp *pb.PointsOperationResponse
	deleteErr  error
	searchResp *pb.SearchResponse
	searchErr  error
}

func (m *mockPoints) Upsert(_ context.Context, _ *pb.UpsertPoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return m.upsertResp, m.upsertErr
}
func (m *mockPoints) Delete(_ context.Context, _ *pb.DeletePoints, _ ...grpc.CallOption) (*pb.PointsOperationResponse, error) {
	return m.deleteResp, m.deleteErr
}
func (m *mockPoints) Search(_ context.Context, _ *pb.SearchPoints, _ ...grpc.CallOption) (*pb.SearchResponse, error) {
	return m.searchResp, m.searchErr
}

type mockCollections struct {
	listResp   *pb.ListCollectionsResponse
	listErr    error
	createResp *pb.CollectionOperationResponse
	createErr  error
	deleteResp *pb.CollectionOperationResponse
	deleteErr  error
}

func (m *mockCollections) List(_ context.Context, _ *pb.ListCollectionsRequest, _ ...grpc.CallOption) (*pb.ListCollectionsResponse, error) {
	return m.listResp, m.listErr
}
func (m *mockCollections) Create(_ context.Context, _ *pb.CreateCollection, _ ...grpc.CallOption) (*pb.CollectionOperationResponse, error) {
	return m.createResp, m.createErr
}
func (m *mockCollections) Delete(_ context.Context, _ *pb.DeleteCollection, _ ...grpc.CallOption) (*pb.CollectionOperationResponse, error) {
	return m.deleteResp, m.deleteErr
}

// --- Tests ---

func TestNewWithClients(t *testing.T) {
	vs := NewWithClients(&mockPoints{}, &mockCollections{}, "test")
	if vs == nil {
		t.Fatal("expected non-nil")
	}
	if err := vs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestEnsureCollection_AlreadyExists(t *testing.T) {
	cols := &mockCollections{
		listResp: &pb.ListCollectionsResponse{
			Collections: []*pb.CollectionDescription{{Name: "test"}},
		},
	}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.EnsureCollection(context.Background(), 4); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureCollection_Creates(t *testing.T) {
	cols := &mockCollections{
		listResp:   &pb.ListCollectionsResponse{Collections: []*pb.CollectionDescription{}},
		createResp: &pb.CollectionOperationResponse{Result: true},
	}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.EnsureCollection(context.Background(), 128); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureCollection_ListError(t *testing.T) {
	cols := &mockCollections{listErr: errors.New("rpc fail")}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.EnsureCollection(context.Background(), 4); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureCollection_CreateError(t *testing.T) {
	cols := &mockCollections{
		listResp:  &pb.ListCollectionsResponse{Collections: []*pb.CollectionDescription{}},
		createErr: errors.New("create fail"),
	}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.EnsureCollection(context.Background(), 4); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteCollection_Success(t *testing.T) {
	cols := &mockCollections{deleteResp: &pb.CollectionOperationResponse{Result: true}}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.DeleteCollection(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCollection_Error(t *testing.T) {
	cols := &mockCollections{deleteErr: errors.New("fail")}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.DeleteCollection(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpsert_Empty(t *testing.T) {
	vs := NewWithClients(&mockPoints{}, &mockCollections{}, "test")
	if err := vs.Upsert(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpsert_Success(t *testing.T) {
	pts := &mockPoints{upsertResp: &pb.PointsOperationResponse{}}
	vs := NewWithClients(pts, &mockCollections{}, "test")

	records := []VectorRecord{
		{
			ID:        "id1",
			Embedding: []float32{1, 0, 0, 0},
			Payload: map[string]any{
				"content":  "hello",
				"count":    42,
				"count64":  int64(99),
				"score":    3.14,
				"active":   true,
				"other":    []int{1, 2}, // default case
			},
		},
	}
	if err := vs.Upsert(context.Background(), records); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpsert_Error(t *testing.T) {
	pts := &mockPoints{upsertErr: errors.New("fail")}
	vs := NewWithClients(pts, &mockCollections{}, "test")

	records := []VectorRecord{{ID: "id1", Embedding: []float32{1, 0}}}
	if err := vs.Upsert(context.Background(), records); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteByDocID_Success(t *testing.T) {
	pts := &mockPoints{deleteResp: &pb.PointsOperationResponse{}}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	if err := vs.DeleteByDocID(context.Background(), "doc1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteByDocID_Error(t *testing.T) {
	pts := &mockPoints{deleteErr: errors.New("fail")}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	if err := vs.DeleteByDocID(context.Background(), "doc1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSearch_Success(t *testing.T) {
	pts := &mockPoints{
		searchResp: &pb.SearchResponse{
			Result: []*pb.ScoredPoint{
				{
					Id:    &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: "p1"}},
					Score: 0.95,
					Payload: map[string]*pb.Value{
						"content": {Kind: &pb.Value_StringValue{StringValue: "oil change"}},
						"doc_id":  {Kind: &pb.Value_StringValue{StringValue: "d1"}},
						"source":  {Kind: &pb.Value_StringValue{StringValue: "reddit"}},
						"extra":   {Kind: &pb.Value_StringValue{StringValue: "val"}},
					},
				},
			},
		},
	}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	results, err := vs.Search(context.Background(), []float32{1, 0}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Content != "oil change" {
		t.Errorf("wrong content: %s", results[0].Content)
	}
	if results[0].DocID != "d1" {
		t.Errorf("wrong doc_id: %s", results[0].DocID)
	}
	if results[0].Source != "reddit" {
		t.Errorf("wrong source: %s", results[0].Source)
	}
	if results[0].Meta["extra"] != "val" {
		t.Errorf("wrong meta: %v", results[0].Meta)
	}
	if results[0].ID != "p1" || results[0].Score != 0.95 {
		t.Error("wrong id/score")
	}
}

func TestSearch_Error(t *testing.T) {
	pts := &mockPoints{searchErr: errors.New("fail")}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	_, err := vs.Search(context.Background(), []float32{1}, 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchFiltered_WithFilters(t *testing.T) {
	pts := &mockPoints{
		searchResp: &pb.SearchResponse{
			Result: []*pb.ScoredPoint{
				{
					Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: "p1"}},
					Score:   0.8,
					Payload: map[string]*pb.Value{},
				},
			},
		},
	}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	results, err := vs.SearchFiltered(context.Background(), []float32{1}, 5, map[string]string{"make": "honda"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
}

func TestSearchFiltered_EmptyResults(t *testing.T) {
	pts := &mockPoints{searchResp: &pb.SearchResponse{}}
	vs := NewWithClients(pts, &mockCollections{}, "test")
	results, err := vs.SearchFiltered(context.Background(), []float32{1}, 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

func TestFieldMatch(t *testing.T) {
	cond := fieldMatch("key", "value")
	fc := cond.GetField()
	if fc.Key != "key" {
		t.Fatalf("expected key, got %s", fc.Key)
	}
	if fc.Match.GetKeyword() != "value" {
		t.Fatalf("expected value, got %s", fc.Match.GetKeyword())
	}
}

func TestEnsureCollection_OtherCollectionExists(t *testing.T) {
	cols := &mockCollections{
		listResp: &pb.ListCollectionsResponse{
			Collections: []*pb.CollectionDescription{{Name: "other"}},
		},
		createResp: &pb.CollectionOperationResponse{Result: true},
	}
	vs := NewWithClients(&mockPoints{}, cols, "test")
	if err := vs.EnsureCollection(context.Background(), 4); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
