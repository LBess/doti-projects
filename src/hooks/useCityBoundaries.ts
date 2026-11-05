import { useEffect } from 'react';
import { useMap } from '@vis.gl/react-google-maps';
import type { City } from '../types';

type LatLng = { lat: number; lng: number };

// Boundary rings are [lng, lat]; Google wants {lat, lng}.
const toPath = (ring: number[][]): LatLng[] => ring.map(([lng, lat]) => ({ lat, lng }));

const OUTLINE = { strokeWeight: 3, clickable: false };
const HOLE = { fillColor: '#5f6368', fillOpacity: 0.5, strokeWeight: 0, clickable: false };

/**
 * Outlines each city's limits on the Google Map component, in the city's
 * color. A polygon's outer ring is the limit; any further rings are holes
 * (other cities inside it), which are shaded grey and outlined too.
 */
export const useCityBoundaries = (cities: City[]) => {
    const map = useMap();

    useEffect(() => {
        if (!map) return;

        const overlays: (google.maps.Polygon | google.maps.Polyline)[] = [];
        for (const city of cities) {
            for (const [outer, ...holes] of city.boundary) {
                const holePaths = holes.map(toPath);
                for (const path of holePaths) {
                    overlays.push(new google.maps.Polygon({ ...HOLE, paths: [path], map }));
                }
                for (const path of [toPath(outer), ...holePaths]) {
                    overlays.push(
                        new google.maps.Polyline({
                            ...OUTLINE,
                            strokeColor: city.color,
                            path: [...path, path[0]],
                            map,
                        }),
                    );
                }
            }
        }

        return () => {
            overlays.forEach((overlay) => overlay.setMap(null));
        };
    }, [map, cities]);
};
