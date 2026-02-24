// Package domain defines core domain types, constants, and validation for the
// Wessley engine pipeline. It acts as the validation gate at pipeline entry points.
package domain

import "time"

// Vehicle represents a vehicle for diagnostic queries.
type Vehicle struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  int    `json:"year"`
	VIN   string `json:"vin,omitempty"`
}

// Query represents a user diagnostic query.
type Query struct {
	Text    string  `json:"text"`
	Vehicle Vehicle `json:"vehicle"`
}

// ScrapedPost represents a scraped forum/social post.
type ScrapedPost struct {
	Source    string    `json:"source"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	URL      string    `json:"url"`
	AuthorID string    `json:"author_id,omitempty"`
	PostedAt time.Time `json:"posted_at"`
}

// Component represents a vehicle component/part.
type Component struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	PartNo   string `json:"part_no,omitempty"`
}

// SymptomCategory classifies vehicle symptoms.
type SymptomCategory string

const (
	SymptomEngine       SymptomCategory = "engine"
	SymptomTransmission SymptomCategory = "transmission"
	SymptomBrakes       SymptomCategory = "brakes"
	SymptomSuspension   SymptomCategory = "suspension"
	SymptomElectrical   SymptomCategory = "electrical"
	SymptomExhaust      SymptomCategory = "exhaust"
	SymptomCooling      SymptomCategory = "cooling"
	SymptomFuel         SymptomCategory = "fuel"
	SymptomHVAC         SymptomCategory = "hvac"
	SymptomBody         SymptomCategory = "body"
	SymptomOther        SymptomCategory = "other"
)

// ValidSymptomCategories is the set of recognised symptom categories.
var ValidSymptomCategories = map[SymptomCategory]bool{
	SymptomEngine: true, SymptomTransmission: true, SymptomBrakes: true,
	SymptomSuspension: true, SymptomElectrical: true, SymptomExhaust: true,
	SymptomCooling: true, SymptomFuel: true, SymptomHVAC: true,
	SymptomBody: true, SymptomOther: true,
}

// FixCategory classifies types of fixes/repairs.
type FixCategory string

const (
	FixReplacement  FixCategory = "replacement"
	FixRepair       FixCategory = "repair"
	FixAdjustment   FixCategory = "adjustment"
	FixSoftware     FixCategory = "software_update"
	FixRecall       FixCategory = "recall"
	FixMaintenance  FixCategory = "maintenance"
	FixDiagnostic   FixCategory = "diagnostic"
)

// ValidFixCategories is the set of recognised fix categories.
var ValidFixCategories = map[FixCategory]bool{
	FixReplacement: true, FixRepair: true, FixAdjustment: true,
	FixSoftware: true, FixRecall: true, FixMaintenance: true,
	FixDiagnostic: true,
}
