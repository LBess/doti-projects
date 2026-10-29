package geocode

import (
	_ "embed"
	"encoding/json"

	"github.com/LBess/doti-projects/server/internal/project"
)

// denver_boundary.json is a copy of src/data/denverBoundary.json, which the
// frontend draws on the map; keep the two in sync. It is a list of rings of
// [lng, lat] vertices: ring 0 is the city limit, any further rings are
// enclaves (separate cities inside Denver).
//
//go:embed denver_boundary.json
var boundaryJSON []byte

type ring [][2]float64

var denverRings = func() []ring {
	var rings []ring
	if err := json.Unmarshal(boundaryJSON, &rings); err != nil {
		panic("denver_boundary.json: " + err.Error())
	}
	return rings
}()

// inRing is a ray-casting test: an odd number of edge crossings means the
// point is inside.
func inRing(p project.Point, r ring) bool {
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

// inRings reports whether p is inside the first ring and outside every other.
func inRings(p project.Point, rings []ring) bool {
	if len(rings) == 0 || !inRing(p, rings[0]) {
		return false
	}
	for _, enclave := range rings[1:] {
		if inRing(p, enclave) {
			return false
		}
	}
	return true
}

// InDenver reports whether p is inside the city limit and outside every
// enclave.
func InDenver(p project.Point) bool { return inRings(p, denverRings) }
