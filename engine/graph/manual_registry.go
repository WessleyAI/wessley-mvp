package graph

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// ManualEntry represents a discovered vehicle manual PDF on the web.
type ManualEntry struct {
	ID           string     `json:"id"`
	URL          string     `json:"url"`
	SourceSite   string     `json:"source_site"`
	Make         string     `json:"make"`
	Model        string     `json:"model"`
	Year         int        `json:"year"`
	Trim         string     `json:"trim,omitempty"`
	ManualType   string     `json:"manual_type"`
	Language     string     `json:"language"`
	FileSize     int64      `json:"file_size"`
	PageCount    int        `json:"page_count"`
	DiscoveredAt time.Time  `json:"discovered_at"`
	DownloadedAt *time.Time `json:"downloaded_at,omitempty"`
	IngestedAt   *time.Time `json:"ingested_at,omitempty"`
	LocalPath    string     `json:"local_path,omitempty"`
	Status       string     `json:"status"`
	Error        string     `json:"error,omitempty"`
}

// ManualFilter specifies criteria for finding manuals.
type ManualFilter struct {
	Make   string
	Model  string
	Year   int
	Status string
}

// ManualStats holds aggregate counts for manual entries.
type ManualStats struct {
	Total        int            `json:"total"`
	ByStatus     map[string]int `json:"by_status"`
	BySource     map[string]int `json:"by_source"`
}

// ManualEntryID produces a deterministic ID from a URL.
func ManualEntryID(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", h[:16])
}

// SaveManualEntry creates or updates a ManualEntry node in the graph.
func (g *GraphStore) SaveManualEntry(ctx context.Context, m ManualEntry) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	props := map[string]any{
		"id":            m.ID,
		"url":           m.URL,
		"source_site":   m.SourceSite,
		"make":          m.Make,
		"model":         m.Model,
		"year":          m.Year,
		"trim":          m.Trim,
		"manual_type":   m.ManualType,
		"language":      m.Language,
		"file_size":     m.FileSize,
		"page_count":    m.PageCount,
		"discovered_at": m.DiscoveredAt.Unix(),
		"status":        m.Status,
		"error":         m.Error,
		"local_path":    m.LocalPath,
	}
	if m.DownloadedAt != nil {
		props["downloaded_at"] = m.DownloadedAt.Unix()
	}
	if m.IngestedAt != nil {
		props["ingested_at"] = m.IngestedAt.Unix()
	}

	cypher := `MERGE (n:ManualEntry {id: $id}) SET n += $props`
	_, err := sess.Run(ctx, cypher, map[string]any{"id": m.ID, "props": props})
	return err
}

// FindManuals returns manuals matching the given filter.
func (g *GraphStore) FindManuals(ctx context.Context, f ManualFilter) ([]ManualEntry, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	where := "WHERE 1=1"
	params := map[string]any{}
	if f.Make != "" {
		where += " AND n.make = $make"
		params["make"] = f.Make
	}
	if f.Model != "" {
		where += " AND n.model = $model"
		params["model"] = f.Model
	}
	if f.Year > 0 {
		where += " AND n.year = $year"
		params["year"] = f.Year
	}
	if f.Status != "" {
		where += " AND n.status = $status"
		params["status"] = f.Status
	}

	cypher := fmt.Sprintf("MATCH (n:ManualEntry) %s RETURN n", where)
	result, err := sess.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	return collectManualEntries(ctx, result)
}

// UpdateManualStatus updates the status and error fields of a manual entry.
func (g *GraphStore) UpdateManualStatus(ctx context.Context, id, status, errMsg string) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (n:ManualEntry {id: $id}) SET n.status = $status, n.error = $error`
	_, err := sess.Run(ctx, cypher, map[string]any{"id": id, "status": status, "error": errMsg})
	return err
}

// GetPendingDownloads returns manuals with status "discovered" up to the limit.
func (g *GraphStore) GetPendingDownloads(ctx context.Context, limit int) ([]ManualEntry, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (n:ManualEntry {status: "discovered"}) RETURN n LIMIT $limit`
	result, err := sess.Run(ctx, cypher, map[string]any{"limit": limit})
	if err != nil {
		return nil, err
	}
	return collectManualEntries(ctx, result)
}

// GetPendingIngestion returns manuals with status "downloaded" up to the limit.
func (g *GraphStore) GetPendingIngestion(ctx context.Context, limit int) ([]ManualEntry, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	cypher := `MATCH (n:ManualEntry {status: "downloaded"}) RETURN n LIMIT $limit`
	result, err := sess.Run(ctx, cypher, map[string]any{"limit": limit})
	if err != nil {
		return nil, err
	}
	return collectManualEntries(ctx, result)
}

// ManualStats returns aggregate counts for manual entries.
func (g *GraphStore) ManualStats(ctx context.Context) (ManualStats, error) {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	stats := ManualStats{
		ByStatus: make(map[string]int),
		BySource: make(map[string]int),
	}

	// Count by status
	cypher := `MATCH (n:ManualEntry) RETURN n.status AS status, count(n) AS cnt`
	result, err := sess.Run(ctx, cypher, nil)
	if err != nil {
		return stats, err
	}
	for result.Next(ctx) {
		rec := result.Record()
		s, _ := rec.Get("status")
		c, _ := rec.Get("cnt")
		if status, ok := s.(string); ok {
			if cnt, ok := c.(int64); ok {
				stats.ByStatus[status] = int(cnt)
				stats.Total += int(cnt)
			}
		}
	}

	// Count by source
	cypher = `MATCH (n:ManualEntry) RETURN n.source_site AS src, count(n) AS cnt`
	result, err = sess.Run(ctx, cypher, nil)
	if err != nil {
		return stats, err
	}
	for result.Next(ctx) {
		rec := result.Record()
		s, _ := rec.Get("src")
		c, _ := rec.Get("cnt")
		if src, ok := s.(string); ok {
			if cnt, ok := c.(int64); ok {
				stats.BySource[src] = int(cnt)
			}
		}
	}

	return stats, nil
}

func collectManualEntries(ctx context.Context, result CypherResult) ([]ManualEntry, error) {
	var entries []ManualEntry
	for result.Next(ctx) {
		nVal, ok := result.Record().Get("n")
		if !ok {
			continue
		}
		node, ok := nVal.(dbtype.Node)
		if !ok {
			continue
		}
		props := node.Props
		entries = append(entries, manualEntryFromProps(props))
	}
	return entries, nil
}

func manualEntryFromProps(p map[string]any) ManualEntry {
	m := ManualEntry{
		ID:         strProp(p, "id"),
		URL:        strProp(p, "url"),
		SourceSite: strProp(p, "source_site"),
		Make:       strProp(p, "make"),
		Model:      strProp(p, "model"),
		Trim:       strProp(p, "trim"),
		ManualType: strProp(p, "manual_type"),
		Language:   strProp(p, "language"),
		Status:     strProp(p, "status"),
		Error:      strProp(p, "error"),
		LocalPath:  strProp(p, "local_path"),
	}
	if v, ok := p["year"]; ok {
		switch y := v.(type) {
		case int64:
			m.Year = int(y)
		case float64:
			m.Year = int(y)
		}
	}
	if v, ok := p["file_size"]; ok {
		if fs, ok := v.(int64); ok {
			m.FileSize = fs
		}
	}
	if v, ok := p["page_count"]; ok {
		if pc, ok := v.(int64); ok {
			m.PageCount = int(pc)
		}
	}
	if v, ok := p["discovered_at"]; ok {
		if ts, ok := v.(int64); ok {
			m.DiscoveredAt = time.Unix(ts, 0)
		}
	}
	if v, ok := p["downloaded_at"]; ok {
		if ts, ok := v.(int64); ok {
			t := time.Unix(ts, 0)
			m.DownloadedAt = &t
		}
	}
	if v, ok := p["ingested_at"]; ok {
		if ts, ok := v.(int64); ok {
			t := time.Unix(ts, 0)
			m.IngestedAt = &t
		}
	}
	return m
}

