import { useEffect } from 'react';
import { useMap } from '@vis.gl/react-google-maps';
import rings from '../data/denverBoundary.json';

type LatLng = { lat: number; lng: number };

// GeoJSON rings are [lng, lat]; Google wants {lat, lng}. Ring 0 is the city
// limit; any further rings are enclaves (separate cities inside Denver).
const toPath = (ring: number[][]): LatLng[] => ring.map(([lng, lat]) => ({ lat, lng }));
const [cityRing, ...enclaveRings] = (rings as number[][][]).map(toPath);

const OUTLINE = { strokeColor: '#0b4a9e', strokeWeight: 3, clickable: false };
const ENCLAVE = { fillColor: '#5f6368', fillOpacity: 0.5, strokeWeight: 0, clickable: false };

/**
 * Adds an outline for the city of Denver's boundaries to
 * the Google Map component
 */
export const useDenverBoundary = () => {
    const map = useMap();

    useEffect(() => {
        if (!map) return;

        const enclaves = enclaveRings.map(
            (path) => new google.maps.Polygon({ ...ENCLAVE, paths: [path], map }),
        );
        const outlines = [cityRing, ...enclaveRings].map(
            (path) => new google.maps.Polyline({ ...OUTLINE, path: [...path, path[0]], map }),
        );

        return () => {
            [...enclaves, ...outlines].forEach((overlay) => overlay.setMap(null));
        };
    }, [map]);
};
