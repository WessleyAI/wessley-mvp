package manuals

import (
	"context"

	"github.com/WessleyAI/wessley-mvp/engine/graph"
)

// ManualSource discovers vehicle manual PDFs from a specific website or API.
type ManualSource interface {
	// Name returns the source identifier (e.g. "toyota", "archive").
	Name() string
	// Discover crawls the source and returns discovered manual entries.
	// It does NOT download PDFs — just discovers URLs and metadata.
	Discover(ctx context.Context, makes []string, years []int) ([]graph.ManualEntry, error)
}
