package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// VIN format: 17 alphanumeric characters, excluding I, O, Q.
var vinRegex = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

// Injection patterns — SQL/NoSQL fragments that should never appear in a user query.
var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(DROP|DELETE|INSERT|UPDATE|ALTER|EXEC|UNION)\b.*\b(TABLE|FROM|INTO|SELECT|SET)\b`),
	regexp.MustCompile(`(?i)(--|;)\s*(DROP|DELETE|SELECT)`),
	regexp.MustCompile(`(?i)\$\{.*\}`),                // template injection
	regexp.MustCompile(`(?i)\{\s*"\$[a-z]+"\s*:`),     // NoSQL operator injection
}

// Profanity word list (lowercase, basic set — extend as needed).
var profanityWords = map[string]bool{
	"fuck": true, "shit": true, "ass": true, "bitch": true,
	"damn": true, "cunt": true, "dick": true, "piss": true,
}

const minQueryLength = 5

// ValidateVehicle validates a Vehicle struct.
func ValidateVehicle(v Vehicle) error {
	// Make
	models, ok := SupportedMakes[v.Make]
	if !ok {
		return NewValidationError("make", v.Make, ErrUnsupportedMake)
	}

	// Model
	found := false
	for _, m := range models {
		if strings.EqualFold(m, v.Model) {
			found = true
			break
		}
	}
	if !found {
		return NewValidationError("model", v.Model, ErrUnsupportedModel)
	}

	// Year
	if v.Year < MinModelYear || v.Year > MaxModelYear {
		return NewValidationError("year", fmt.Sprintf("%d", v.Year), ErrYearOutOfRange)
	}

	// VIN (optional but if provided must be valid)
	if v.VIN != "" && !vinRegex.MatchString(strings.ToUpper(v.VIN)) {
		return NewValidationError("vin", v.VIN, ErrInvalidVIN)
	}

	return nil
}

// ValidateQuery validates a diagnostic query.
func ValidateQuery(q Query) error {
	text := strings.TrimSpace(q.Text)

	// Length check
	if utf8.RuneCountInString(text) < minQueryLength {
		return NewValidationError("text", text, ErrQueryTooShort)
	}

	// Injection check
	for _, pat := range injectionPatterns {
		if pat.MatchString(text) {
			return NewValidationError("text", text, ErrQueryInjection)
		}
	}

	// Profanity check (word-boundary split)
	lower := strings.ToLower(text)
	for _, word := range strings.Fields(lower) {
		// Strip common punctuation from edges
		cleaned := strings.Trim(word, ".,!?;:'\"()-")
		if profanityWords[cleaned] {
			return NewValidationError("text", cleaned, ErrQueryProfanity)
		}
	}

	// Validate the embedded vehicle
	if err := ValidateVehicle(q.Vehicle); err != nil {
		return err
	}

	return nil
}
