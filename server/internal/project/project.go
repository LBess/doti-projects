// Package project defines the shape of a DOTI project as it is scraped,
// stored, and served.
package project

// Project is a single DOTI project. SourceKey is its stable identity across
// scrapes (see scraper.sourceKey) and is served to the frontend as `id`.
//
// The scraper fills in only the first four fields. Position, Location and
// UnmappedReason come from geocoding (see Placement) and are populated when a
// project is read back from the store: a project has a Position and Location
// when it can be mapped, and an UnmappedReason otherwise.
type Project struct {
	SourceKey   string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DetailURL   *string `json:"detailUrl"`

	Position       *Point    `json:"position"`
	Location       *Location `json:"location"`
	UnmappedReason *string   `json:"unmappedReason"`
}

// Point is a WGS84 coordinate.
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Location describes how a project's pin was derived from its text.
type Location struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

// Kinds of Location.
const (
	KindPoint        = "point"        // a single geocoded query, e.g. an intersection
	KindCorridor     = "corridor"     // the midpoint between two cross streets
	KindNeighborhood = "neighborhood" // the centroid of one or more neighborhoods
)

// Placement states. Pending means the project hasn't been geocoded yet (or
// the attempt failed in a way worth retrying); the rest are final until the
// project's text changes. Every state other than Mapped is also the
// UnmappedReason served to the frontend.
const (
	Pending         = "pending"
	Mapped          = "mapped"
	NoLocationFound = "no-location-found"
	GeocodeFailed   = "geocode-failed"
	OutsideDenver   = "outside-denver"
)

// Placement is the outcome of trying to put a project on the map. Status is
// one of the states above; Location and Position are set only when it is
// Mapped.
type Placement struct {
	Status   string
	Location *Location
	Position *Point
}
