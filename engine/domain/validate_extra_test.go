package domain

import (
	"errors"
	"testing"

	"github.com/WessleyAI/wessley-mvp/engine/scraper"
)

func TestValidateVehicle_CaseInsensitiveModel(t *testing.T) {
	// Model matching is case-insensitive
	err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "camry", Year: 2020})
	if err != nil {
		t.Errorf("expected case-insensitive model match, got %v", err)
	}
}

func TestValidateVehicle_BoundaryYears(t *testing.T) {
	// Exactly at boundaries
	if err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: MinModelYear}); err != nil {
		t.Errorf("MinModelYear should be valid: %v", err)
	}
	if err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: MaxModelYear}); err != nil {
		t.Errorf("MaxModelYear should be valid: %v", err)
	}
	// Just outside
	if err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: MinModelYear - 1}); !errors.Is(err, ErrYearOutOfRange) {
		t.Error("MinModelYear-1 should be out of range")
	}
	if err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: MaxModelYear + 1}); !errors.Is(err, ErrYearOutOfRange) {
		t.Error("MaxModelYear+1 should be out of range")
	}
}

func TestValidateVehicle_VINWithForbiddenChars(t *testing.T) {
	// O and Q are forbidden in VINs
	cases := []string{
		"5YJ3E1EA1OF123456", // O
		"5YJ3E1EA1QF123456", // Q
	}
	for _, vin := range cases {
		err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: 2020, VIN: vin})
		if !errors.Is(err, ErrInvalidVIN) {
			t.Errorf("VIN %q should be invalid, got %v", vin, err)
		}
	}
}

func TestValidateVehicle_LowercaseVIN(t *testing.T) {
	// VIN validation should uppercase before checking
	err := ValidateVehicle(Vehicle{Make: "Tesla", Model: "Model 3", Year: 2024, VIN: "5yj3e1ea1nf123456"})
	if err != nil {
		t.Errorf("lowercase VIN should be valid after uppercasing: %v", err)
	}
}

func TestValidateQuery_ProfanityWithPunctuation(t *testing.T) {
	q := Query{
		Text:    "this shit! engine is broken.",
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if !errors.Is(ValidateQuery(q), ErrQueryProfanity) {
		t.Error("profanity with punctuation should still be caught")
	}
}

func TestValidateQuery_WhitespaceOnly(t *testing.T) {
	q := Query{
		Text:    "    ",
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if !errors.Is(ValidateQuery(q), ErrQueryTooShort) {
		t.Error("whitespace-only should be too short")
	}
}

func TestValidateQuery_ExactlyMinLength(t *testing.T) {
	q := Query{
		Text:    "abcde", // exactly 5 runes
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	err := ValidateQuery(q)
	if err != nil {
		t.Errorf("exactly min length should be valid: %v", err)
	}
}

func TestValidateQuery_NoSQLInjection(t *testing.T) {
	q := Query{
		Text:    `check {"$gt": 100} something`,
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if !errors.Is(ValidateQuery(q), ErrQueryInjection) {
		t.Error("NoSQL injection should be caught")
	}
}

func TestValidateScrapedPost_Valid(t *testing.T) {
	post := scraper.ScrapedPost{
		Source:   "reddit",
		SourceID: "abc",
		Title:    "Test",
		Content:  "Some content",
	}
	if err := ValidateScrapedPost(post); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestValidateScrapedPost_EmptyFields(t *testing.T) {
	tests := []struct {
		name string
		post scraper.ScrapedPost
	}{
		{"empty content", scraper.ScrapedPost{Source: "reddit", SourceID: "a", Title: "t", Content: ""}},
		{"bad source", scraper.ScrapedPost{Source: "unknown", SourceID: "a", Title: "t", Content: "c"}},
		{"empty source_id", scraper.ScrapedPost{Source: "reddit", SourceID: "", Title: "t", Content: "c"}},
		{"empty title", scraper.ScrapedPost{Source: "reddit", SourceID: "a", Title: "", Content: "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateScrapedPost(tt.post); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestValidateScrapedPost_AllValidSources(t *testing.T) {
	for src := range ValidSources {
		post := scraper.ScrapedPost{Source: src, SourceID: "x", Title: "t", Content: "c"}
		if err := ValidateScrapedPost(post); err != nil {
			t.Errorf("source %q should be valid: %v", src, err)
		}
	}
}

func TestValidationErrorFormat(t *testing.T) {
	ve := NewValidationError("field", "value", ErrInvalidVehicle)
	msg := ve.Error()
	if msg == "" {
		t.Fatal("error message should not be empty")
	}
	// Should contain field and value
	if !errors.Is(ve, ErrInvalidVehicle) {
		t.Fatal("should unwrap to ErrInvalidVehicle")
	}
}

func TestAllMakesHaveModels(t *testing.T) {
	for make, models := range SupportedMakes {
		if len(models) == 0 {
			t.Errorf("make %q has no models", make)
		}
	}
}
