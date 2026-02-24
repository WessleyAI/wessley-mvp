// Package nhtsa provides a scraper for the NHTSA complaints database.
package nhtsa

import "time"

// Complaint represents a raw NHTSA complaint record.
type Complaint struct {
	ODINumber   int    `json:"ODINumber"`
	Manufacturer string `json:"Manufacturer"`
	MakeName    string `json:"MakeName"`
	ModelName   string `json:"ModelName"`
	ModelYear   int    `json:"ModelYear"`
	Component   string `json:"Component"`
	Summary     string `json:"Summary"`
	DateOfIncident string `json:"DateOfIncident"`
	DateComplaintFiled string `json:"DateComplaintFiled"`
	NumberOfInjuries int `json:"NumberOfInjuries"`
	NumberOfDeaths int `json:"NumberOfDeaths"`
}

// Config controls NHTSA scraper behavior.
type Config struct {
	Makes     []string
	ModelYear int
	MaxPerMake int
	RateLimit time.Duration
}
