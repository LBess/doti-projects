package city

import "github.com/LBess/doti-projects/server/internal/project"

// Ring is a closed loop of [lng, lat] vertices (GeoJSON order). The closing
// vertex is implied, not repeated.
type Ring [][2]float64

// Polygon is an outer ring followed by any holes: areas inside the outer ring
// that belong to someone else (e.g. an enclave belonging to another city).
type Polygon []Ring

// Boundary is a city limit. Cities can be made of several pieces, so it is a
// list of polygons — the same layout as a GeoJSON MultiPolygon's coordinates.
type Boundary []Polygon

// inRing is a ray-casting test: an odd number of edge crossings means the
// point is inside.
func inRing(p project.Point, r Ring) bool {
	inside := false
	for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
		xi, yi := r[i][0], r[i][1]
		xj, yj := r[j][0], r[j][1]
		if (yi > p.Lat) != (yj > p.Lat) && p.Lng < (xj-xi)*(p.Lat-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

// contains reports whether p is inside the polygon's outer ring and outside
// every hole.
func (poly Polygon) contains(p project.Point) bool {
	if len(poly) == 0 || !inRing(p, poly[0]) {
		return false
	}
	for _, hole := range poly[1:] {
		if inRing(p, hole) {
			return false
		}
	}
	return true
}

// Contains reports whether p is inside any piece of the boundary.
func (b Boundary) Contains(p project.Point) bool {
	for _, poly := range b {
		if poly.contains(p) {
			return true
		}
	}
	return false
}
