package domain

import (
	"errors"
	"testing"
)

func TestValidateVehicle_Valid(t *testing.T) {
	cases := []Vehicle{
		{Make: "Toyota", Model: "Camry", Year: 2020},
		{Make: "Tesla", Model: "Model 3", Year: 2024, VIN: "5YJ3E1EA1NF123456"},
		{Make: "Ford", Model: "F-150", Year: 1980},
		{Make: "BMW", Model: "3 Series", Year: 2027},
	}
	for _, v := range cases {
		if err := ValidateVehicle(v); err != nil {
			t.Errorf("expected valid for %+v, got %v", v, err)
		}
	}
}

func TestValidateVehicle_InvalidMake(t *testing.T) {
	err := ValidateVehicle(Vehicle{Make: "Lada", Model: "Niva", Year: 2020})
	if !errors.Is(err, ErrUnsupportedMake) {
		t.Errorf("expected ErrUnsupportedMake, got %v", err)
	}
}

func TestValidateVehicle_InvalidModel(t *testing.T) {
	err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "FakeModel", Year: 2020})
	if !errors.Is(err, ErrUnsupportedModel) {
		t.Errorf("expected ErrUnsupportedModel, got %v", err)
	}
}

func TestValidateVehicle_YearOutOfRange(t *testing.T) {
	err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: 1970})
	if !errors.Is(err, ErrYearOutOfRange) {
		t.Errorf("expected ErrYearOutOfRange, got %v", err)
	}
	err = ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: 2099})
	if !errors.Is(err, ErrYearOutOfRange) {
		t.Errorf("expected ErrYearOutOfRange, got %v", err)
	}
}

func TestValidateVehicle_InvalidVIN(t *testing.T) {
	err := ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: 2020, VIN: "INVALID"})
	if !errors.Is(err, ErrInvalidVIN) {
		t.Errorf("expected ErrInvalidVIN, got %v", err)
	}
	// VIN with I (forbidden)
	err = ValidateVehicle(Vehicle{Make: "Toyota", Model: "Camry", Year: 2020, VIN: "5YJ3E1EA1IF123456"})
	if !errors.Is(err, ErrInvalidVIN) {
		t.Errorf("expected ErrInvalidVIN for VIN with I, got %v", err)
	}
}

func TestValidateQuery_Valid(t *testing.T) {
	q := Query{
		Text:    "My car makes a rattling noise when accelerating",
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if err := ValidateQuery(q); err != nil {
		t.Errorf("expected valid query, got %v", err)
	}
}

func TestValidateQuery_TooShort(t *testing.T) {
	q := Query{
		Text:    "hi",
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if !errors.Is(ValidateQuery(q), ErrQueryTooShort) {
		t.Errorf("expected ErrQueryTooShort")
	}
}

func TestValidateQuery_Injection(t *testing.T) {
	cases := []string{
		"car problem; DROP TABLE users",
		"noise ${process.env.SECRET}",
		`rattle {"$gt": 1}`,
	}
	for _, text := range cases {
		q := Query{Text: text, Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019}}
		if !errors.Is(ValidateQuery(q), ErrQueryInjection) {
			t.Errorf("expected ErrQueryInjection for %q", text)
		}
	}
}

func TestValidateQuery_Profanity(t *testing.T) {
	q := Query{
		Text:    "this fuck engine is broken",
		Vehicle: Vehicle{Make: "Honda", Model: "Civic", Year: 2019},
	}
	if !errors.Is(ValidateQuery(q), ErrQueryProfanity) {
		t.Errorf("expected ErrQueryProfanity")
	}
}

func TestValidateQuery_InvalidVehicle(t *testing.T) {
	q := Query{
		Text:    "engine makes noise on cold start",
		Vehicle: Vehicle{Make: "Lada", Model: "Niva", Year: 2020},
	}
	if !errors.Is(ValidateQuery(q), ErrUnsupportedMake) {
		t.Errorf("expected ErrUnsupportedMake from embedded vehicle validation")
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	ve := NewValidationError("make", "Lada", ErrUnsupportedMake)
	if !errors.Is(ve, ErrUnsupportedMake) {
		t.Errorf("Unwrap should expose ErrUnsupportedMake")
	}
	var target *ValidationError
	if !errors.As(ve, &target) {
		t.Errorf("errors.As should work for *ValidationError")
	}
	if target.Field != "make" {
		t.Errorf("expected field=make, got %s", target.Field)
	}
}

func TestSymptomAndFixCategories(t *testing.T) {
	if !ValidSymptomCategories[SymptomEngine] {
		t.Error("SymptomEngine should be valid")
	}
	if ValidSymptomCategories["nonexistent"] {
		t.Error("nonexistent should not be valid")
	}
	if !ValidFixCategories[FixReplacement] {
		t.Error("FixReplacement should be valid")
	}
}
