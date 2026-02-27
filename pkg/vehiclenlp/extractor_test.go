package vehiclenlp

import "testing"

func TestExtractBest(t *testing.T) {
	tests := []struct {
		input     string
		wantMake  string
		wantModel string
		wantYear  int
	}{
		{"My 2019 Honda Civic is making a clicking noise", "Honda", "Civic", 2019},
		{"F-150 EcoBoost turbo replacement", "Ford", "F-150", 0},
		{"2022 Camry hybrid battery issue", "Toyota", "Camry", 2022},
		{"Having trouble with my '18 Chevy Silverado", "Chevrolet", "Silverado", 2018},
		{"BMW 3 Series N54 turbo problems", "BMW", "3 Series", 0},
		{"Just bought a Tesla Model 3", "Tesla", "Model 3", 0},
		{"Jeep Grand Cherokee 2020 death wobble", "Jeep", "Grand Cherokee", 2020},
		{"Help with VW Golf R tune", "Volkswagen", "Golf", 0},
		{"2024 Hyundai Tucson transmission shudder", "Hyundai", "Tucson", 2024},
		{"My 2020 Subaru Outback CVT is slipping", "Subaru", "Outback", 2020},
		{"Audi A4 B8 timing chain rattle", "Audi", "A4", 0},
		{"2023 Kia Telluride towing capacity", "Kia", "Telluride", 2023},
		{"Dodge Charger 2019 transmission issue", "Dodge", "Charger", 2019},
		{"Lexus RX 350 2021 infotainment problems", "Lexus", "RX", 2021},
		{"2018 Mazda CX-5 oil dilution", "Mazda", "CX-5", 2018},
		{"Porsche 911 Carrera engine tick", "Porsche", "911", 0},
		{"2022 Ford Bronco soft top issues", "Ford", "Bronco", 2022},
		{"Toyota Tacoma 2021 rear differential noise", "Toyota", "Tacoma", 2021},
		{"Honda CR-V 2020 AC not blowing cold", "Honda", "CR-V", 2020},
		{"Nissan Altima 2019 CVT problems", "Nissan", "Altima", 2019},
		{"2023 Genesis GV70 review", "Genesis", "GV70", 2023},
		{"Mercedes C-Class 2020 oil leak", "Mercedes-Benz", "C-Class", 2020},
		{"Volvo XC90 2019 air suspension failure", "Volvo", "XC90", 2019},
		{"GMC Sierra 1500 2022 6.2L issues", "GMC", "Sierra", 2022},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := ExtractBest(tt.input)
			if m == nil {
				t.Fatalf("ExtractBest(%q) = nil, want match", tt.input)
			}
			if m.Make != tt.wantMake {
				t.Errorf("Make = %q, want %q", m.Make, tt.wantMake)
			}
			if m.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", m.Model, tt.wantModel)
			}
			if m.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", m.Year, tt.wantYear)
			}
		})
	}
}

func TestExtractEmpty(t *testing.T) {
	if m := ExtractBest(""); m != nil {
		t.Error("expected nil for empty string")
	}
	if m := ExtractBest("nothing about cars here"); m != nil {
		t.Errorf("expected nil, got %+v", m)
	}
}

func TestExtractMultiple(t *testing.T) {
	matches := Extract("I traded my 2019 Honda Civic for a 2023 Toyota RAV4")
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
}

func TestCaseInsensitive(t *testing.T) {
	m := ExtractBest("my 2020 HONDA civic overheating")
	if m == nil || m.Make != "Honda" || m.Model != "Civic" {
		t.Errorf("case insensitive failed: %+v", m)
	}
}

func TestAbbreviatedYear(t *testing.T) {
	m := ExtractBest("'19 Ford Mustang GT exhaust")
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Year != 2019 {
		t.Errorf("Year = %d, want 2019", m.Year)
	}
	if m.Make != "Ford" || m.Model != "Mustang" {
		t.Errorf("got %s %s, want Ford Mustang", m.Make, m.Model)
	}
}
