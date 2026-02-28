// Package nhtsa provides a scraper for the NHTSA complaints database.
package nhtsa

import "time"

// Complaint represents a raw NHTSA complaint record.
type Complaint struct {
	ODINumber          int       `json:"odiNumber"`
	Manufacturer       string    `json:"manufacturer"`
	Crash              bool      `json:"crash"`
	Fire               bool      `json:"fire"`
	NumberOfInjuries   int       `json:"numberOfInjuries"`
	NumberOfDeaths     int       `json:"numberOfDeaths"`
	DateOfIncident     string    `json:"dateOfIncident"`
	DateComplaintFiled string    `json:"dateComplaintFiled"`
	VIN                string    `json:"vin"`
	Components         string    `json:"components"`
	Summary            string    `json:"summary"`
	Products           []Product `json:"products"`
}

// Product represents a vehicle/tire product in a complaint.
type Product struct {
	Type         string `json:"type"`
	ProductYear  string `json:"productYear"`
	ProductMake  string `json:"productMake"`
	ProductModel string `json:"productModel"`
	Manufacturer string `json:"manufacturer"`
}

// VehicleProduct returns the first Vehicle-type product, if any.
func (c *Complaint) VehicleProduct() *Product {
	for i := range c.Products {
		if c.Products[i].Type == "Vehicle" {
			return &c.Products[i]
		}
	}
	return nil
}

// Config controls NHTSA scraper behavior.
type Config struct {
	Makes      []string
	ModelYear  int      // single year (legacy); ignored if ModelYears is set
	ModelYears []int    // year list to iterate; takes precedence over ModelYear
	MaxPerMake int
	RateLimit  time.Duration
}

// Years returns the list of model years to scrape.
func (c Config) Years() []int {
	if len(c.ModelYears) > 0 {
		return c.ModelYears
	}
	return []int{c.ModelYear}
}
