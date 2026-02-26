package graph

import "strings"

// SystemTaxonomy maps known system names to their possible subsystems.
// Used for CLASSIFYING extracted content — NOT for seeding the graph.
// Only systems/subsystems actually found in a vehicle's manual get created as nodes.
var SystemTaxonomy = map[string][]string{
	"Engine": {
		"Fuel Injection", "Ignition", "Cooling", "Lubrication",
		"Exhaust", "Intake", "Timing", "Turbo/Supercharger",
	},
	"Transmission": {
		"Automatic", "Manual", "CVT", "Clutch", "Torque Converter",
		"Shift Mechanism", "Differential",
	},
	"Brakes": {
		"Disc Brakes", "Drum Brakes", "ABS", "Brake Lines",
		"Master Cylinder", "Parking Brake", "Brake Pads",
	},
	"Suspension": {
		"Front Suspension", "Rear Suspension", "Shocks/Struts",
		"Springs", "Control Arms", "Sway Bars", "Bushings",
	},
	"Electrical": {
		"Battery", "Alternator", "Starter", "Wiring Harness",
		"Fuse Box", "Lighting", "Sensors", "ECU/PCM",
	},
	"HVAC": {
		"Compressor", "Condenser", "Evaporator", "Heater Core",
		"Blower Motor", "Thermostat", "Refrigerant",
	},
	"Fuel System": {
		"Fuel Pump", "Fuel Filter", "Fuel Tank", "Fuel Lines",
		"Fuel Injectors", "Fuel Rail",
	},
	"Steering": {
		"Power Steering", "Steering Rack", "Steering Column",
		"Tie Rods", "Ball Joints",
	},
	"Body": {
		"Doors", "Windows", "Mirrors", "Bumpers",
		"Hood", "Trunk", "Weatherstripping",
	},
	"Exhaust": {
		"Catalytic Converter", "Muffler", "Exhaust Manifold",
		"O2 Sensors", "Exhaust Pipes", "EGR",
	},
	"Cooling": {
		"Radiator", "Water Pump", "Thermostat", "Coolant Hoses",
		"Fan", "Coolant Reservoir",
	},
	"Safety": {
		"Airbags", "Seatbelts", "TPMS", "Backup Camera",
		"Collision Avoidance", "Lane Departure",
	},
}

// systemKeywords maps lowercase keywords found in titles/content to system names.
var systemKeywords = map[string]string{
	"engine":              "Engine",
	"motor":               "Engine",
	"cylinder":            "Engine",
	"piston":              "Engine",
	"crankshaft":          "Engine",
	"camshaft":            "Engine",
	"valve train":         "Engine",
	"transmission":        "Transmission",
	"transaxle":           "Transmission",
	"gearbox":             "Transmission",
	"brake":               "Brakes",
	"braking":             "Brakes",
	"abs":                 "Brakes",
	"suspension":          "Suspension",
	"shock":               "Suspension",
	"strut":               "Suspension",
	"spring":              "Suspension",
	"electrical":          "Electrical",
	"battery":             "Electrical",
	"alternator":          "Electrical",
	"starter":             "Electrical",
	"wiring":              "Electrical",
	"fuse":                "Electrical",
	"relay":               "Electrical",
	"ecu":                 "Electrical",
	"pcm":                 "Electrical",
	"sensor":              "Electrical",
	"hvac":                "HVAC",
	"air conditioning":    "HVAC",
	"a/c":                 "HVAC",
	"heater":              "HVAC",
	"climate":             "HVAC",
	"fuel system":         "Fuel System",
	"fuel pump":           "Fuel System",
	"fuel filter":         "Fuel System",
	"fuel tank":           "Fuel System",
	"fuel injection":      "Engine",
	"fuel injector":       "Fuel System",
	"steering":            "Steering",
	"power steering":      "Steering",
	"tie rod":             "Steering",
	"body":                "Body",
	"door":                "Body",
	"window":              "Body",
	"mirror":              "Body",
	"exhaust":             "Exhaust",
	"catalytic converter": "Exhaust",
	"muffler":             "Exhaust",
	"egr":                 "Exhaust",
	"cooling":             "Cooling",
	"radiator":            "Cooling",
	"water pump":          "Cooling",
	"coolant":             "Cooling",
	"thermostat":          "Cooling",
	"safety":              "Safety",
	"airbag":              "Safety",
	"seatbelt":            "Safety",
	"tpms":                "Safety",
	"tire pressure":       "Safety",
	"collision":           "Safety",
	"lane departure":      "Safety",
	"maintenance":         "",
	"maintenance schedule": "",
}

// subsystemKeywords maps lowercase keywords to (system, subsystem) pairs.
var subsystemKeywords = map[string][2]string{
	"fuel injection":      {"Engine", "Fuel Injection"},
	"ignition":            {"Engine", "Ignition"},
	"turbo":               {"Engine", "Turbo/Supercharger"},
	"supercharger":        {"Engine", "Turbo/Supercharger"},
	"automatic":           {"Transmission", "Automatic"},
	"cvt":                 {"Transmission", "CVT"},
	"clutch":              {"Transmission", "Clutch"},
	"torque converter":    {"Transmission", "Torque Converter"},
	"differential":        {"Transmission", "Differential"},
	"disc brake":          {"Brakes", "Disc Brakes"},
	"drum brake":          {"Brakes", "Drum Brakes"},
	"abs":                 {"Brakes", "ABS"},
	"brake line":          {"Brakes", "Brake Lines"},
	"master cylinder":     {"Brakes", "Master Cylinder"},
	"parking brake":       {"Brakes", "Parking Brake"},
	"brake pad":           {"Brakes", "Brake Pads"},
	"front suspension":    {"Suspension", "Front Suspension"},
	"rear suspension":     {"Suspension", "Rear Suspension"},
	"shock":               {"Suspension", "Shocks/Struts"},
	"strut":               {"Suspension", "Shocks/Struts"},
	"control arm":         {"Suspension", "Control Arms"},
	"sway bar":            {"Suspension", "Sway Bars"},
	"battery":             {"Electrical", "Battery"},
	"alternator":          {"Electrical", "Alternator"},
	"starter":             {"Electrical", "Starter"},
	"wiring harness":      {"Electrical", "Wiring Harness"},
	"fuse box":            {"Electrical", "Fuse Box"},
	"fuse":                {"Electrical", "Fuse Box"},
	"lighting":            {"Electrical", "Lighting"},
	"headlight":           {"Electrical", "Lighting"},
	"taillight":           {"Electrical", "Lighting"},
	"ecu":                 {"Electrical", "ECU/PCM"},
	"pcm":                 {"Electrical", "ECU/PCM"},
	"compressor":          {"HVAC", "Compressor"},
	"condenser":           {"HVAC", "Condenser"},
	"evaporator":          {"HVAC", "Evaporator"},
	"heater core":         {"HVAC", "Heater Core"},
	"blower motor":        {"HVAC", "Blower Motor"},
	"fuel pump":           {"Fuel System", "Fuel Pump"},
	"fuel filter":         {"Fuel System", "Fuel Filter"},
	"fuel tank":           {"Fuel System", "Fuel Tank"},
	"fuel injector":       {"Fuel System", "Fuel Injectors"},
	"fuel rail":           {"Fuel System", "Fuel Rail"},
	"power steering":      {"Steering", "Power Steering"},
	"steering rack":       {"Steering", "Steering Rack"},
	"steering column":     {"Steering", "Steering Column"},
	"tie rod":             {"Steering", "Tie Rods"},
	"ball joint":          {"Steering", "Ball Joints"},
	"catalytic converter": {"Exhaust", "Catalytic Converter"},
	"muffler":             {"Exhaust", "Muffler"},
	"exhaust manifold":    {"Exhaust", "Exhaust Manifold"},
	"o2 sensor":           {"Exhaust", "O2 Sensors"},
	"oxygen sensor":       {"Exhaust", "O2 Sensors"},
	"egr":                 {"Exhaust", "EGR"},
	"radiator":            {"Cooling", "Radiator"},
	"water pump":          {"Cooling", "Water Pump"},
	"coolant hose":        {"Cooling", "Coolant Hoses"},
	"coolant reservoir":   {"Cooling", "Coolant Reservoir"},
	"airbag":              {"Safety", "Airbags"},
	"seatbelt":            {"Safety", "Seatbelts"},
	"tpms":                {"Safety", "TPMS"},
	"tire pressure":       {"Safety", "TPMS"},
	"backup camera":       {"Safety", "Backup Camera"},
	"collision avoidance": {"Safety", "Collision Avoidance"},
	"lane departure":      {"Safety", "Lane Departure"},
}

// ClassifySection takes section title and content, returns (system, subsystem).
// Returns empty strings if no match is found.
func ClassifySection(title, content string) (system, subsystem string) {
	lowerTitle := strings.ToLower(title)
	lowerContent := strings.ToLower(content)

	// Check subsystem keywords first (more specific) — prefer title matches.
	for kw, pair := range subsystemKeywords {
		if strings.Contains(lowerTitle, kw) {
			return pair[0], pair[1]
		}
	}

	// Check system keywords in title.
	for kw, sys := range systemKeywords {
		if sys != "" && strings.Contains(lowerTitle, kw) {
			system = sys
			break
		}
	}

	// If we got a system from title, try to find subsystem from content.
	if system != "" {
		for kw, pair := range subsystemKeywords {
			if pair[0] == system && strings.Contains(lowerContent, kw) {
				return system, pair[1]
			}
		}
		return system, ""
	}

	// Fall back to content for subsystem keywords.
	for kw, pair := range subsystemKeywords {
		if strings.Contains(lowerContent, kw) {
			return pair[0], pair[1]
		}
	}

	// Fall back to content for system keywords.
	for kw, sys := range systemKeywords {
		if sys != "" && strings.Contains(lowerContent, kw) {
			return sys, ""
		}
	}

	return "", ""
}

// ClassifyComponent takes a component name/description and returns (system, subsystem).
func ClassifyComponent(name, description string) (system, subsystem string) {
	return ClassifySection(name, description)
}

// sanitizeID converts a name to a lowercase dash-separated ID.
func sanitizeID(name string) string {
	b := make([]byte, 0, len(name))
	for i := range name {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+32) // lowercase
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b = append(b, c)
		case c == ' ' || c == '/' || c == '_':
			if len(b) > 0 && b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	if len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return string(b)
}
