package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for validation failures.
var (
	ErrInvalidVehicle = errors.New("invalid vehicle")
	ErrInvalidQuery   = errors.New("invalid query")
	ErrInvalidVIN     = errors.New("invalid VIN")
	ErrUnsupportedMake  = errors.New("unsupported make")
	ErrUnsupportedModel = errors.New("unsupported model")
	ErrYearOutOfRange   = errors.New("year out of range")
	ErrQueryTooShort    = errors.New("query too short")
	ErrQueryInjection   = errors.New("query contains suspicious content")
	ErrQueryProfanity   = errors.New("query contains profanity")
)

// ValidationError wraps a sentinel with context.
type ValidationError struct {
	Field   string
	Value   string
	Wrapped error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s (value=%q)", e.Wrapped, e.Field, e.Value)
}

func (e *ValidationError) Unwrap() error { return e.Wrapped }

// NewValidationError creates a ValidationError.
func NewValidationError(field, value string, wrapped error) *ValidationError {
	return &ValidationError{Field: field, Value: value, Wrapped: wrapped}
}
