package domain

// SupportedMakes maps make names to their known models.
var SupportedMakes = map[string][]string{
	"Toyota":     {"Camry", "Corolla", "RAV4", "Highlander", "Tacoma", "Tundra", "4Runner", "Prius", "Supra", "Avalon"},
	"Honda":      {"Civic", "Accord", "CR-V", "Pilot", "Odyssey", "HR-V", "Ridgeline", "Fit", "Insight"},
	"Ford":       {"F-150", "Mustang", "Explorer", "Escape", "Ranger", "Bronco", "Edge", "Expedition", "Maverick", "Focus", "Fusion"},
	"Chevrolet":  {"Silverado", "Equinox", "Malibu", "Traverse", "Tahoe", "Suburban", "Colorado", "Camaro", "Corvette", "Blazer"},
	"BMW":        {"3 Series", "5 Series", "7 Series", "X3", "X5", "X7", "M3", "M5", "i4", "iX"},
	"Mercedes":   {"C-Class", "E-Class", "S-Class", "GLC", "GLE", "GLS", "A-Class", "CLA", "AMG GT"},
	"Audi":       {"A3", "A4", "A6", "Q3", "Q5", "Q7", "Q8", "e-tron", "RS6", "TT"},
	"Nissan":     {"Altima", "Sentra", "Rogue", "Pathfinder", "Frontier", "Maxima", "Murano", "Kicks", "Leaf"},
	"Hyundai":    {"Elantra", "Sonata", "Tucson", "Santa Fe", "Kona", "Palisade", "Ioniq 5", "Venue"},
	"Kia":        {"Forte", "K5", "Sportage", "Telluride", "Sorento", "Soul", "Seltos", "EV6", "Carnival"},
	"Volkswagen": {"Golf", "Jetta", "Tiguan", "Atlas", "ID.4", "Passat", "Taos", "Arteon"},
	"Subaru":     {"Outback", "Forester", "Crosstrek", "Impreza", "WRX", "Legacy", "Ascent", "BRZ"},
	"Mazda":      {"Mazda3", "Mazda6", "CX-5", "CX-9", "CX-30", "CX-50", "MX-5 Miata"},
	"Jeep":       {"Wrangler", "Grand Cherokee", "Cherokee", "Compass", "Renegade", "Gladiator", "Wagoneer"},
	"Ram":        {"1500", "2500", "3500", "ProMaster"},
	"GMC":        {"Sierra", "Terrain", "Acadia", "Yukon", "Canyon"},
	"Dodge":      {"Charger", "Challenger", "Durango", "Hornet"},
	"Lexus":      {"ES", "IS", "RX", "NX", "GX", "LS", "LC", "UX"},
	"Acura":      {"TLX", "MDX", "RDX", "Integra"},
	"Tesla":      {"Model 3", "Model Y", "Model S", "Model X", "Cybertruck"},
}

// MinModelYear is the earliest year we accept.
const MinModelYear = 1980

// MaxModelYear is the latest year we accept (current + 1 for next-year models).
const MaxModelYear = 2027
