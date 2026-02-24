package semantic

// SearchResult represents a single vector search hit.
type SearchResult struct {
	ID      string            `json:"id"`
	Score   float32           `json:"score"`
	Content string            `json:"content"`
	DocID   string            `json:"doc_id"`
	Source  string            `json:"source"`
	Meta    map[string]string `json:"meta"`
}

// VectorRecord represents a single vector to store in Qdrant.
type VectorRecord struct {
	ID        string
	Embedding []float32
	Payload   map[string]any // content, doc_id, source, vehicle, chunk_index
}
