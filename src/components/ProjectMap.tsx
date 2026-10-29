import { useState } from 'react';
import {
    Map,
    MapControl,
    ControlPosition,
    AdvancedMarker,
    Pin,
    InfoWindow,
} from '@vis.gl/react-google-maps';
import { useDenverBoundary } from '../hooks/useDenverBoundary';
import type { MappedProject } from '../types';
import styles from './ProjectMap.module.scss';

const DENVER_CENTER = { lat: 39.7392, lng: -104.9903 };

// AdvancedMarker requires a Map ID. DEMO_MAP_ID works for development; set
// VITE_GOOGLE_MAP_ID for a real one.
const MAP_ID = import.meta.env.VITE_GOOGLE_MAP_ID ?? 'DEMO_MAP_ID';

const NEIGHBORHOOD_COLOR = '#0b4a9e';
const LOCATION_COLOR = '#e81d1a';

interface ProjectMapProps {
    projects: MappedProject[];
}

export const ProjectMap = ({ projects }: ProjectMapProps) => {
    const [selected, setSelected] = useState<MappedProject | null>(null);

    useDenverBoundary();

    return (
        <Map
            className={styles.projectMap}
            mapId={MAP_ID}
            defaultCenter={DENVER_CENTER}
            defaultZoom={12}
            gestureHandling="cooperative"
            disableDefaultUI={false}
            onClick={() => setSelected(null)}
        >
            <MapControl position={ControlPosition.LEFT_TOP}>
                <div className={styles.legend}>
                    <div className={styles.legendItem}>
                        <span
                            className={styles.legendSwatch}
                            style={{ background: NEIGHBORHOOD_COLOR }}
                        />
                        Neighborhood
                    </div>
                    <div className={styles.legendItem}>
                        <span
                            className={styles.legendSwatch}
                            style={{ background: LOCATION_COLOR }}
                        />
                        Point or corridor
                    </div>
                </div>
            </MapControl>

            {projects.map((project) => (
                <AdvancedMarker
                    key={project.id}
                    position={project.position}
                    title={project.title}
                    onClick={() => setSelected(project)}
                >
                    {project.location.kind === 'neighborhood' ? (
                        <Pin
                            background={NEIGHBORHOOD_COLOR}
                            borderColor={NEIGHBORHOOD_COLOR}
                            glyphColor="#ffffff"
                        />
                    ) : (
                        <Pin
                            background={LOCATION_COLOR}
                            borderColor={LOCATION_COLOR}
                            glyphColor="#ffffff"
                        />
                    )}
                </AdvancedMarker>
            ))}

            {selected && (
                <InfoWindow
                    headerDisabled
                    position={selected.position}
                    onCloseClick={() => setSelected(null)}
                >
                    <div className={styles.infoWindow}>
                        <h3>{selected.title}</h3>
                        <p className={styles.infoWindowDescription}>{selected.description}</p>
                        {selected.detailUrl && (
                            <a href={selected.detailUrl} target="_blank" rel="noreferrer">
                                Read more about {selected.title}
                            </a>
                        )}
                    </div>
                </InfoWindow>
            )}
        </Map>
    );
};
