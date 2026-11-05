package city

import (
	"testing"

	"github.com/LBess/doti-projects/server/internal/project"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name string
		city City
		p    project.Point
		want bool
	}{
		{"downtown Denver", Denver, project.Point{Lat: 39.7392, Lng: -104.9903}, true},
		{"Golden is not Denver", Denver, project.Point{Lat: 39.7294, Lng: -105.1319}, false},
		{"far west", Denver, project.Point{Lat: 39.5501, Lng: -105.7821}, false},
		{"Belmar, Lakewood", Lakewood, project.Point{Lat: 39.7068, Lng: -105.0848}, true},
		{"Lakewood is not Denver", Denver, project.Point{Lat: 39.7068, Lng: -105.0848}, false},
		{"downtown Denver is not Lakewood", Lakewood, project.Point{Lat: 39.7392, Lng: -104.9903}, false},
		{"Arvada is not Lakewood", Lakewood, project.Point{Lat: 39.8028, Lng: -105.0875}, false},
		{"Arvada is not Denver", Denver, project.Point{Lat: 39.8028, Lng: -105.0875}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.city.Contains(tt.p); got != tt.want {
				t.Errorf("%s.Contains(%v) = %v, want %v", tt.city.ID, tt.p, got, tt.want)
			}
		})
	}
}

func TestBoundaryHolesAndPieces(t *testing.T) {
	square := func(x, y, s float64) Ring { return Ring{{x, y}, {x + s, y}, {x + s, y + s}, {x, y + s}} }
	b := Boundary{
		{square(0, 0, 10), square(4, 4, 2)}, // with a hole
		{square(20, 20, 2)},                 // a separate piece
	}
	for _, tt := range []struct {
		name string
		p    project.Point
		want bool
	}{
		{"in the first piece", project.Point{Lng: 1, Lat: 1}, true},
		{"in a hole", project.Point{Lng: 5, Lat: 5}, false},
		{"in the second piece", project.Point{Lng: 21, Lat: 21}, true},
		{"between pieces", project.Point{Lng: 15, Lat: 15}, false},
	} {
		if got := b.Contains(tt.p); got != tt.want {
			t.Errorf("%s: Contains(%v) = %v, want %v", tt.name, tt.p, got, tt.want)
		}
	}
}

func TestRegistry(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range All() {
		if c.ID == "" || c.Name == "" || c.Color == "" || c.Label == "" || c.GeocodeBounds == "" || len(c.Boundary) == 0 {
			t.Errorf("city %+v is missing a field", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("duplicate city ID %q", c.ID)
		}
		seen[c.ID] = true
		if got, ok := ByID(c.ID); !ok || got.ID != c.ID {
			t.Errorf("ByID(%q) = %v, %v", c.ID, got.ID, ok)
		}
	}
	if _, ok := ByID("nowhere"); ok {
		t.Error("ByID found a city that does not exist")
	}
}
