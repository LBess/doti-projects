// Package city describes the cities whose projects we map: where each one's
// limits are, how it is drawn, and how to geocode inside it.
//
// To add a city: drop its boundary in boundaries/<id>.json, add an entry to
// All, and register a scraper for it in cmd/server/main.go.
package city

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/LBess/doti-projects/server/internal/location"
	"github.com/LBess/doti-projects/server/internal/project"
)

// City is one city in the Denver metro. The json tags are what GET
// /api/cities serves to the frontend.
type City struct {
	ID    string `json:"id"`    // stable key stored with every project
	Name  string `json:"name"`  // for display
	Color string `json:"color"` // boundary outline color on the map
	// Boundary is the city limit, embedded from boundaries/<ID>.json (a copy of
	// a GeoJSON MultiPolygon's coordinates). Projects geocoded outside it are
	// left off the map.
	Boundary Boundary `json:"boundary"`

	// Label is appended to geocoding queries to say which city is meant, e.g.
	// "Colfax Avenue & Downing Street, Denver, CO".
	Label string `json:"-"`
	// GeocodeBounds biases Google's geocoder toward the city so ambiguous
	// street names resolve here. Format is "south,west|north,east".
	GeocodeBounds string `json:"-"`
	// Neighborhoods are the named neighborhoods projects may refer to, as a
	// last-resort location. Nil if we don't know the city's.
	Neighborhoods *location.Neighborhoods `json:"-"`
}

// Contains reports whether p is inside the city limit.
func (c City) Contains(p project.Point) bool { return c.Boundary.Contains(p) }

//go:embed boundaries/*.json
var boundaryFiles embed.FS

func loadBoundary(id string) Boundary {
	data, err := boundaryFiles.ReadFile("boundaries/" + id + ".json")
	if err != nil {
		panic(fmt.Sprintf("city %s: %v", id, err))
	}
	var b Boundary
	if err := json.Unmarshal(data, &b); err != nil {
		panic(fmt.Sprintf("city %s: boundaries/%s.json: %v", id, id, err))
	}
	return b
}

var (
	Denver = City{
		ID:            "denver",
		Name:          "Denver",
		Color:         "#0b4a9e",
		Boundary:      loadBoundary("denver"),
		Label:         "Denver, CO",
		GeocodeBounds: "39.6,-105.15|39.95,-104.6",
		Neighborhoods: location.DenverNeighborhoods,
	}
	Lakewood = City{
		ID:            "lakewood",
		Name:          "Lakewood",
		Color:         "#7b1fa2",
		Boundary:      loadBoundary("lakewood"),
		Label:         "Lakewood, CO",
		GeocodeBounds: "39.59,-105.22|39.78,-105.03",
	}
)

// All lists every city, in display order.
func All() []City { return []City{Denver, Lakewood} }

// ByID finds a city by its ID.
func ByID(id string) (City, bool) {
	for _, c := range All() {
		if c.ID == id {
			return c, true
		}
	}
	return City{}, false
}
