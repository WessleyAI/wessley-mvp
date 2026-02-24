package semantic

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// VectorStore is the sole owner of all Qdrant operations.
type VectorStore struct {
	conn       *grpc.ClientConn
	points     pb.PointsClient
	collections pb.CollectionsClient
	collection string
}

// New creates a VectorStore connected to Qdrant at the given gRPC address.
func New(addr string, collection string) (*VectorStore, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("semantic: dial qdrant %s: %w", addr, err)
	}
	return &VectorStore{
		conn:        conn,
		points:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
		collection:  collection,
	}, nil
}

// Close closes the underlying gRPC connection.
func (v *VectorStore) Close() error {
	return v.conn.Close()
}

// EnsureCollection creates the collection if it doesn't exist.
func (v *VectorStore) EnsureCollection(ctx context.Context, dims int) error {
	// Check if collection exists.
	list, err := v.collections.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("semantic: list collections: %w", err)
	}
	for _, c := range list.GetCollections() {
		if c.GetName() == v.collection {
			return nil
		}
	}

	d := uint64(dims)
	_, err = v.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: v.collection,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     d,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("semantic: create collection %s: %w", v.collection, err)
	}
	return nil
}

// DeleteCollection deletes the collection.
func (v *VectorStore) DeleteCollection(ctx context.Context) error {
	_, err := v.collections.Delete(ctx, &pb.DeleteCollection{
		CollectionName: v.collection,
	})
	if err != nil {
		return fmt.Errorf("semantic: delete collection %s: %w", v.collection, err)
	}
	return nil
}

// Upsert stores embedding records into Qdrant. Called by engine/ingest.
func (v *VectorStore) Upsert(ctx context.Context, records []VectorRecord) error {
	if len(records) == 0 {
		return nil
	}

	points := make([]*pb.PointStruct, len(records))
	for i, r := range records {
		payload := make(map[string]*pb.Value, len(r.Payload))
		for k, val := range r.Payload {
			switch tv := val.(type) {
			case string:
				payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: tv}}
			case int:
				payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(tv)}}
			case int64:
				payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: tv}}
			case float64:
				payload[k] = &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: tv}}
			case bool:
				payload[k] = &pb.Value{Kind: &pb.Value_BoolValue{BoolValue: tv}}
			default:
				payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: fmt.Sprint(tv)}}
			}
		}

		points[i] = &pb.PointStruct{
			Id: &pb.PointId{
				PointIdOptions: &pb.PointId_Uuid{Uuid: r.ID},
			},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: r.Embedding},
				},
			},
			Payload: payload,
		}
	}

	wait := true
	_, err := v.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: v.collection,
		Wait:           &wait,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("semantic: upsert %d points: %w", len(records), err)
	}
	return nil
}

// DeleteByDocID removes all points matching a doc_id. Used for re-ingestion.
func (v *VectorStore) DeleteByDocID(ctx context.Context, docID string) error {
	wait := true
	_, err := v.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: v.collection,
		Wait:           &wait,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						fieldMatch("doc_id", docID),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("semantic: delete by doc_id %s: %w", docID, err)
	}
	return nil
}

// Search performs k-NN similarity search. Called by engine/rag.
func (v *VectorStore) Search(ctx context.Context, embedding []float32, topK int) ([]SearchResult, error) {
	return v.SearchFiltered(ctx, embedding, topK, nil)
}

// SearchFiltered performs similarity search with optional metadata filters.
func (v *VectorStore) SearchFiltered(ctx context.Context, embedding []float32, topK int, filters map[string]string) ([]SearchResult, error) {
	req := &pb.SearchPoints{
		CollectionName: v.collection,
		Vector:         embedding,
		Limit:          uint64(topK),
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	}

	if len(filters) > 0 {
		must := make([]*pb.Condition, 0, len(filters))
		for k, val := range filters {
			must = append(must, fieldMatch(k, val))
		}
		req.Filter = &pb.Filter{Must: must}
	}

	resp, err := v.points.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("semantic: search: %w", err)
	}

	results := make([]SearchResult, len(resp.GetResult()))
	for i, r := range resp.GetResult() {
		sr := SearchResult{
			ID:    r.GetId().GetUuid(),
			Score: r.GetScore(),
			Meta:  make(map[string]string),
		}
		for k, val := range r.GetPayload() {
			s := val.GetStringValue()
			switch k {
			case "content":
				sr.Content = s
			case "doc_id":
				sr.DocID = s
			case "source":
				sr.Source = s
			default:
				sr.Meta[k] = s
			}
		}
		results[i] = sr
	}
	return results, nil
}

func fieldMatch(key, value string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: key,
				Match: &pb.Match{
					MatchValue: &pb.Match_Keyword{Keyword: value},
				},
			},
		},
	}
}
