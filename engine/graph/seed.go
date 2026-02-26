package graph

import "context"

// StandardSystems defines the standard vehicle systems and their subsystems.
var StandardSystems = map[string][]string{
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

// SeedSystems creates all standard systems and subsystems in the graph.
func (g *GraphStore) SeedSystems(ctx context.Context) error {
	sess := g.opener.OpenSession(ctx)
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx CypherRunner) (any, error) {
		for sysName, subs := range StandardSystems {
			sysID := sanitizeID(sysName)
			cypher := `MERGE (s:System {id: $id}) SET s.name = $name`
			if _, err := tx.Run(ctx, cypher, map[string]any{"id": sysID, "name": sysName}); err != nil {
				return nil, err
			}

			for _, subName := range subs {
				subID := sysID + "-" + sanitizeID(subName)
				cypher = `MERGE (ss:Subsystem {id: $id}) SET ss.name = $name, ss.system_id = $sysID
				          WITH ss
				          MATCH (s:System {id: $sysID})
				          MERGE (s)-[:HAS_SUBSYSTEM]->(ss)`
				if _, err := tx.Run(ctx, cypher, map[string]any{"id": subID, "name": subName, "sysID": sysID}); err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
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
	// Trim trailing dash
	if len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return string(b)
}
