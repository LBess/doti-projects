export interface GeocodedPoint {
    lat: number;
    lng: number;
}

export interface ProjectLocation {
    // How the pin was derived: an intersection or bridge crossing ('point'),
    // the midpoint of a corridor's two cross streets ('corridor'), or the
    // centroid of one or more named neighborhoods ('neighborhood').
    kind: 'point' | 'corridor' | 'neighborhood';
    label: string;
}

export type UnmappedReason = 'pending' | 'no-location-found' | 'geocode-failed' | 'outside-city';

// A project as served by the backend, which works out where each one goes on
// the map: `position` and `location` are set when it can be mapped, and
// `unmappedReason` says why not otherwise.
export interface Project {
    id: string;
    // The `id` of the City this project belongs to.
    city: string;
    title: string;
    description: string;
    detailUrl?: string;
    position?: GeocodedPoint;
    location?: ProjectLocation;
    unmappedReason?: UnmappedReason;
}

// A city we map projects for. `boundary` is its limit, laid out like a GeoJSON
// MultiPolygon's coordinates: a list of polygons, each an outer ring followed
// by any holes, each ring a list of [lng, lat] vertices.
export interface City {
    id: string;
    name: string;
    // The color its boundary is drawn in.
    color: string;
    boundary: number[][][][];
}

export interface MappedProject extends Project {
    position: GeocodedPoint;
    location: ProjectLocation;
}

export interface UnmappedProject extends Project {
    unmappedReason: UnmappedReason;
}
