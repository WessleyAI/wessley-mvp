package graph

import (
	"testing"
)

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
	if c.Name != "ECU Main" {
		t.Fatalf("expected name=ECU Main, got %s", c.Name)
	}
	if c.Type != "ecu" {
		t.Fatalf("expected type=ecu, got %s", c.Type)
	}
	if c.Vehicle != "2020-Toyota-Camry" {
		t.Fatalf("expected vehicle, got %s", c.Vehicle)
	}
	if c.Properties["color"] != "blue" {
		t.Fatalf("expected prop color=blue, got %s", c.Properties["color"])
	}
	if c.Properties["pin"] != "A12" {
		t.Fatalf("expected prop pin=A12, got %s", c.Properties["pin"])
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

func TestNewGraphStore(t *testing.T) {
	// Verify construction with nil driver (no actual Neo4j needed).
	gs := New(nil)
	if gs == nil {
		t.Fatal("expected non-nil GraphStore")
	}
	if gs.components == nil {
		t.Fatal("expected non-nil components repo")
	}
}
