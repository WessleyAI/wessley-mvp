package graph

import (
	"context"
	"strings"
	"testing"
)

// trackingTx records all cypher queries executed.
type trackingTx struct {
	queries []string
	params  []map[string]any
}

func (t *trackingTx) Run(_ context.Context, cypher string, params map[string]any) (CypherResult, error) {
	t.queries = append(t.queries, cypher)
	t.params = append(t.params, params)
	return newMockResult(), nil
}

type trackingSession struct {
	tx *trackingTx
}

func (s *trackingSession) Run(_ context.Context, cypher string, params map[string]any) (CypherResult, error) {
	return s.tx.Run(context.Background(), cypher, params)
}
func (s *trackingSession) Close(_ context.Context) error { return nil }
func (s *trackingSession) ExecuteWrite(_ context.Context, work func(tx CypherRunner) (any, error)) (any, error) {
	return work(s.tx)
}

type trackingOpener struct {
	session *trackingSession
}

func (o *trackingOpener) OpenSession(_ context.Context) CypherSession {
	return o.session
}

func newTrackingStore() (*GraphStore, *trackingTx) {
	tx := &trackingTx{}
	sess := &trackingSession{tx: tx}
	opener := &trackingOpener{session: sess}
	return NewWithOpener(opener), tx
}

func TestClassifySection(t *testing.T) {
	tests := []struct {
		title, content   string
		wantSys, wantSub string
	}{
		{"Engine", "fuel injection system overview", "Engine", "Fuel Injection"},
		{"Electrical System", "battery and alternator", "Electrical", "Alternator"},
		{"BRAKES", "disc brake inspection", "Brakes", "Disc Brakes"},
		{"Maintenance Schedule", "oil change every 5000 miles", "", ""},
		{"Random Title", "the radiator needs flushing", "Cooling", "Radiator"},
		{"", "", "", ""},
		{"Steering", "power steering fluid check", "Steering", "Power Steering"},
	}

	for _, tt := range tests {
		sys, sub := ClassifySection(tt.title, tt.content)
		if sys != tt.wantSys || sub != tt.wantSub {
			t.Errorf("ClassifySection(%q, %q) = (%q, %q), want (%q, %q)",
				tt.title, tt.content, sys, sub, tt.wantSys, tt.wantSub)
		}
	}
}

func TestClassifyComponent(t *testing.T) {
	sys, sub := ClassifyComponent("alternator", "12V alternator replacement")
	if sys != "Electrical" || sub != "Alternator" {
		t.Errorf("ClassifyComponent = (%q, %q), want (Electrical, Alternator)", sys, sub)
	}
}

func TestEnrichFromManual(t *testing.T) {
	gs, tx := newTrackingStore()
	enricher := NewEnricher(gs)

	vi := VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2024}
	sections := []ManualSection{
		{
			Title:   "Engine",
			Content: "fuel injection system",
			System:  "Engine",
			Subsystem: "Fuel Injection",
			Components: []ExtractedComponent{
				{Name: "Fuel Injector", PartNumber: "23250-0V030"},
			},
		},
		{
			Title:   "Electrical System",
			Content: "battery specs: 12V",
			System:  "Electrical",
			Subsystem: "Battery",
			Components: []ExtractedComponent{
				{Name: "Battery", Specs: map[string]string{"voltage": "12V"}},
			},
		},
	}

	err := enricher.EnrichFromManual(context.Background(), vi, sections)
	if err != nil {
		t.Fatalf("EnrichFromManual: %v", err)
	}

	// Verify vehicle hierarchy was created (3 queries: Make, VehicleModel, ModelYear).
	// Then for each section: System MERGE, Subsystem MERGE, Component MERGE, link component.
	// 2 sections × (1 sys + 1 sub + 1 comp + 1 link) = 8 section queries
	// Total: 3 (hierarchy) + 8 = 11
	if len(tx.queries) < 10 {
		t.Errorf("expected at least 10 queries, got %d", len(tx.queries))
	}

	// Check that vehicle-scoped IDs are used.
	foundScopedSystem := false
	for _, p := range tx.params {
		if id, ok := p["id"].(string); ok {
			if strings.HasPrefix(id, "toyota-camry-2024:") {
				foundScopedSystem = true
				break
			}
		}
	}
	if !foundScopedSystem {
		t.Error("expected vehicle-scoped system IDs (toyota-camry-2024:*)")
	}
}

func TestEnrichFromManualEmpty(t *testing.T) {
	gs, _ := newTrackingStore()
	enricher := NewEnricher(gs)
	vi := VehicleInfo{Make: "Honda", Model: "Civic", Year: 2023}

	err := enricher.EnrichFromManual(context.Background(), vi, nil)
	if err != nil {
		t.Fatalf("EnrichFromManual with nil sections: %v", err)
	}
}

func TestEnrichFromManualUnclassified(t *testing.T) {
	gs, tx := newTrackingStore()
	enricher := NewEnricher(gs)
	vi := VehicleInfo{Make: "Ford", Model: "F-150", Year: 2023}

	sections := []ManualSection{
		{Title: "Random Stuff", Content: "nothing automotive here"},
	}
	err := enricher.EnrichFromManual(context.Background(), vi, sections)
	if err != nil {
		t.Fatalf("EnrichFromManual: %v", err)
	}

	// Should only have hierarchy queries, no system nodes (unclassifiable).
	for _, p := range tx.params {
		if id, ok := p["id"].(string); ok {
			if strings.Contains(id, "ford-f-150-2023:") {
				t.Error("should not create scoped nodes for unclassifiable sections")
			}
		}
	}
}

func TestEnrichFromSource(t *testing.T) {
	gs, tx := newTrackingStore()
	enricher := NewEnricher(gs)
	vi := VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2024}

	err := enricher.EnrichFromSource(context.Background(), vi, "ELECTRICAL SYSTEM", "doc-123")
	if err != nil {
		t.Fatalf("EnrichFromSource: %v", err)
	}

	// Should create system node + link doc.
	if len(tx.queries) < 2 {
		t.Errorf("expected at least 2 queries, got %d", len(tx.queries))
	}

	foundSys := false
	for _, p := range tx.params {
		if id, ok := p["id"].(string); ok && id == "toyota-camry-2024:electrical" {
			foundSys = true
		}
	}
	if !foundSys {
		t.Error("expected vehicle-scoped electrical system node")
	}
}

func TestEnrichFromSourceUnknown(t *testing.T) {
	gs, tx := newTrackingStore()
	enricher := NewEnricher(gs)
	vi := VehicleInfo{Make: "Honda", Model: "Civic", Year: 2023}

	err := enricher.EnrichFromSource(context.Background(), vi, "UNKNOWN_THING", "doc-456")
	if err != nil {
		t.Fatalf("EnrichFromSource: %v", err)
	}

	// No queries should happen for unclassifiable components (no hierarchy either).
	if len(tx.queries) != 0 {
		t.Errorf("expected 0 queries for unknown component, got %d", len(tx.queries))
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Engine", "engine"},
		{"Fuel Injection", "fuel-injection"},
		{"ECU/PCM", "ecu-pcm"},
		{"Turbo/Supercharger", "turbo-supercharger"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeID(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
