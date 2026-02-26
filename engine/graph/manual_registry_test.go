package graph

import (
	"testing"
	"time"
)

func TestManualEntryID(t *testing.T) {
	id1 := ManualEntryID("https://example.com/manual.pdf")
	id2 := ManualEntryID("https://example.com/manual.pdf")
	id3 := ManualEntryID("https://example.com/other.pdf")

	if id1 != id2 {
		t.Error("same URL should produce same ID")
	}
	if id1 == id3 {
		t.Error("different URLs should produce different IDs")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-char hex ID, got len %d", len(id1))
	}
}

func TestManualEntryFromProps(t *testing.T) {
	now := time.Now()
	props := map[string]any{
		"id":            "test123",
		"url":           "https://example.com/manual.pdf",
		"source_site":   "example.com",
		"make":          "Toyota",
		"model":         "Camry",
		"year":          int64(2024),
		"trim":          "LE",
		"manual_type":   "owner",
		"language":      "en",
		"file_size":     int64(1024000),
		"page_count":    int64(350),
		"discovered_at": now.Unix(),
		"status":        "discovered",
		"error":         "",
		"local_path":    "",
	}

	m := manualEntryFromProps(props)

	if m.ID != "test123" {
		t.Errorf("ID = %q", m.ID)
	}
	if m.Make != "Toyota" {
		t.Errorf("Make = %q", m.Make)
	}
	if m.Year != 2024 {
		t.Errorf("Year = %d", m.Year)
	}
	if m.FileSize != 1024000 {
		t.Errorf("FileSize = %d", m.FileSize)
	}
	if m.PageCount != 350 {
		t.Errorf("PageCount = %d", m.PageCount)
	}
	if m.Status != "discovered" {
		t.Errorf("Status = %q", m.Status)
	}
}
