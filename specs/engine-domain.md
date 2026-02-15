# Spec: engine-domain — Validation Layer

**Branch:** `spec/engine-domain`
**Effort:** 1-2 days
**Priority:** P1 — Phase 2

---

## Scope

Domain validation types used at pipeline entry points. Ensures data integrity before processing — catches bad data early, keeps pipeline stages focused on logic.

### Files

```
engine/domain/vehicle.go       # Vehicle type + validation
engine/domain/document.go      # ScrapedPost validation
engine/domain/component.go     # Component type validation
engine/domain/errors.go        # Validation error types
engine/domain/domain_test.go
```

## Vehicle Validation

```go
// engine/domain/vehicle.go

type Vehicle struct {
    Year  int    `json:"year"`
    Make  string `json:"make"`
    Model string `json:"model"`
}

func (v Vehicle) Validate() error {
    // Year: 1900–current+2
    // Make: non-empty, known makes list (optional)
    // Model: non-empty
}
```

## Document Validation

```go
// engine/domain/document.go

// ValidSources enumerates accepted scrape sources
var ValidSources = map[string]bool{
    "reddit": true, "youtube": true, "forum": true,
}

// ValidateScrapedPost checks a ScrapedPost before ingestion
func ValidateScrapedPost(post scraper.ScrapedPost) error {
    // Content: non-empty
    // Source: must be in ValidSources
    // SourceID: non-empty
    // Title: non-empty
}
```

## Component Validation

```go
// engine/domain/component.go

type ComponentType string

const (
    ComponentWire       ComponentType = "wire"
    ComponentFuse       ComponentType = "fuse"
    ComponentRelay      ComponentType = "relay"
    ComponentConnector  ComponentType = "connector"
    ComponentSensor     ComponentType = "sensor"
    ComponentModule     ComponentType = "module"
    ComponentSwitch     ComponentType = "switch"
    ComponentMotor      ComponentType = "motor"
    ComponentLight      ComponentType = "light"
    ComponentBattery    ComponentType = "battery"
    ComponentAlternator ComponentType = "alternator"
    ComponentGround     ComponentType = "ground"
)

func (t ComponentType) Valid() bool

// ValidateComponent checks that a component has a valid type and non-empty name
func ValidateComponent(name string, ctype ComponentType) error
```

## Error Types

```go
// engine/domain/errors.go

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e *ValidationError) Error() string
```

## Usage

Called at pipeline entry points:

```go
// In engine/ingest before processing
if err := domain.ValidateScrapedPost(post); err != nil {
    // reject / send to DLQ
}
```

## Acceptance Criteria

- [ ] Vehicle validation (year range, non-empty make/model)
- [ ] ScrapedPost validation (non-empty content, valid source, non-empty ID/title)
- [ ] Component type enum with Valid() check
- [ ] Structured ValidationError type
- [ ] Used at pipeline entry points (ingest, scraper output)
- [ ] Unit tests for all validators

## Dependencies

- None (pure validation, no external deps)

## Reference

- FINAL_ARCHITECTURE.md §8.7 (ingestion pipeline validation)
