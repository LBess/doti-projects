package geocode

import (
	"context"
	"errors"
	"fmt"

	"github.com/LBess/doti-projects/server/internal/city"
	"github.com/LBess/doti-projects/server/internal/location"
	"github.com/LBess/doti-projects/server/internal/project"
)

// Geocoder resolves a query to a coordinate within a city, returning
// ErrNotFound when there is no match.
type Geocoder interface {
	Geocode(ctx context.Context, query string, in city.City) (project.Point, error)
}

// Locator decides where on the map a project belongs.
type Locator struct {
	Geocoder Geocoder
}

// Locate guesses a place from the project's text, geocodes it within the
// project's city, and places the
// pin at the centroid of the results. A project that can't be placed still
// gets a Placement saying why. An error means the attempt itself failed
// (network, quota) and should be retried later; nothing is recorded for it.
func (l *Locator) Locate(ctx context.Context, p project.Project) (project.Placement, error) {
	in, ok := city.ByID(p.City)
	if !ok {
		return project.Placement{}, fmt.Errorf("project %s: unknown city %q", p.SourceKey, p.City)
	}

	guess := location.Extract(p.Title, p.Description, in.Neighborhoods)
	if guess == nil {
		return project.Placement{Status: project.NoLocationFound}, nil
	}

	points := make([]project.Point, len(guess.Queries))
	for i, query := range guess.Queries {
		point, err := l.Geocoder.Geocode(ctx, query, in)
		if errors.Is(err, ErrNotFound) {
			return project.Placement{Status: project.GeocodeFailed}, nil
		}
		if err != nil {
			return project.Placement{}, err
		}
		points[i] = point
	}

	position := centroid(points)
	if !in.Contains(position) {
		return project.Placement{Status: project.OutsideCity}, nil
	}
	return project.Placement{
		Status:   project.Mapped,
		Location: &project.Location{Kind: guess.Kind, Label: guess.Label},
		Position: &position,
	}, nil
}

// centroid is the average position of one or more points — a corridor's
// midpoint, or a centroid across several neighborhoods.
func centroid(points []project.Point) project.Point {
	var lat, lng float64
	for _, p := range points {
		lat += p.Lat
		lng += p.Lng
	}
	n := float64(len(points))
	return project.Point{Lat: lat / n, Lng: lng / n}
}
