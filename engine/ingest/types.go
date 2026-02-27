package ingest

import "github.com/WessleyAI/wessley-mvp/engine/scraper"

// ParsedDoc represents a scraped post after parsing/extraction.
type ParsedDoc struct {
	ID          string
	Source      string
	Title       string
	Content     string
	Vehicle     string
	VehicleInfo *scraper.VehicleInfo
	Sentences   []string
	Metadata    map[string]string
}

// ChunkedDoc is a parsed document split into embeddable chunks.
type ChunkedDoc struct {
	ParsedDoc
	Chunks []Chunk
}

// Chunk is a text segment ready for embedding.
type Chunk struct {
	Text  string
	Index int
	DocID string
}

// EmbeddedDoc is a chunked document with embeddings.
type EmbeddedDoc struct {
	ChunkedDoc
	Embeddings [][]float32
}

// parsedDocFromPost converts a ScrapedPost into a ParsedDoc.
func parsedDocFromPost(post scraper.ScrapedPost) ParsedDoc {
	meta := map[string]string{
		"source":     post.Source,
		"author":     post.Author,
		"url":        post.URL,
		"vehicle":    post.Metadata.Vehicle,
		"components": post.Metadata.Components,
		"section":    post.Metadata.Section,
	}
	return ParsedDoc{
		ID:          post.Source + ":" + post.SourceID,
		Source:      post.Source,
		Title:       post.Title,
		Content:     post.Content,
		Vehicle:     post.Metadata.Vehicle,
		VehicleInfo: post.Metadata.VehicleInfo,
		Sentences:   splitSentences(post.Content),
		Metadata:    meta,
	}
}
