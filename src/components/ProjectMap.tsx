import { useEffect } from 'react';
import { Map, AdvancedMarker, Pin, useMap } from '@vis.gl/react-google-maps';
import { useDenverBoundary } from '../hooks/useDenverBoundary';
import type { GeocodedPoint, MappedProject } from '../types';
import styles from './ProjectMap.module.scss';

const DENVER_CENTER = { lat: 39.7392, lng: -104.9903 };

// AdvancedMarker requires a Map ID. DEMO_MAP_ID works for development; set
// VITE_GOOGLE_MAP_ID for a real one.
const MAP_ID = import.meta.env.VITE_GOOGLE_MAP_ID ?? 'DEMO_MAP_ID';

const NEIGHBORHOOD_COLOR = '#0b4a9e';
const LOCATION_COLOR = '#e81d1a';

interface ProjectMapProps {
    projects: MappedProject[];
    selectedId: string | null;
    onSelect: (id: string) => void;
}

/** Pans the surrounding map to `position` whenever it changes. */
const PanTo = ({ position }: { position: GeocodedPoint | undefined }) => {
    const map = useMap();

    useEffect(() => {
        if (map && position) map.panTo(position);
    }, [map, position]);

    return null;
};

export const ProjectMap = ({ projects, selectedId, onSelect }: ProjectMapProps) => {
    useDenverBoundary();

    return (
        <Map
            className={styles.projectMap}
            mapId={MAP_ID}
            defaultCenter={DENVER_CENTER}
            defaultZoom={12}
            gestureHandling="greedy"
            disableDefaultUI={false}
            fullscreenControl={false}
        >
            <div className={styles.legend}>
                <div className={styles.legendItem}>
                    <span
                        className={styles.legendSwatch}
                        style={{ background: NEIGHBORHOOD_COLOR }}
                    />
                    Neighborhood
                </div>
                <div className={styles.legendItem}>
                    <span className={styles.legendSwatch} style={{ background: LOCATION_COLOR }} />
                    Point or corridor
                </div>
            </div>

            {projects.map((project) => {
                const color =
                    project.location.kind === 'neighborhood' ? NEIGHBORHOOD_COLOR : LOCATION_COLOR;
                const isSelected = project.id === selectedId;
                return (
                    <AdvancedMarker
                        key={project.id}
                        position={project.position}
                        title={project.title}
                        zIndex={isSelected ? 1 : 0}
                        onClick={() => onSelect(project.id)}
                    >
                        <Pin
                            background={color}
                            borderColor={color}
                            glyphColor="#ffffff"
                            scale={isSelected ? 1.4 : 1}
                        />
                    </AdvancedMarker>
                );
            })}

            <PanTo position={projects.find((p) => p.id === selectedId)?.position} />
        </Map>
    );
};
