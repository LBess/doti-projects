import type { UnmappedProject } from '../types';
import styles from './UnmappedProjects.module.scss';

interface UnmappedProjectsProps {
    projects: UnmappedProject[];
}

const REASON_LABEL: Record<UnmappedProject['unmappedReason'], string> = {
    pending: 'Location not determined yet',
    'no-location-found': 'No specific location could be determined',
    'geocode-failed': "Location couldn't be geocoded",
    'outside-denver': 'Geocoded location falls outside Denver',
};

export const UnmappedProjects = ({ projects }: UnmappedProjectsProps) => {
    if (projects.length === 0) return <p>Every project is on the map.</p>;

    return (
        <section className={styles.unmappedProjects}>
            <p className={styles.unmappedProjectsHint}>
                These {projects.length} projects are city-wide, span many neighborhoods, or don't
                name a specific street location, so they aren't shown on the map.
            </p>
            <ul>
                {projects.map((project) => (
                    <li key={project.id}>
                        {project.detailUrl ? (
                            <a href={project.detailUrl} target="_blank" rel="noreferrer">
                                {project.title}
                            </a>
                        ) : (
                            <span>{project.title}</span>
                        )}
                        <span className={styles.unmappedProjectsReason}>
                            {' '}
                            — {REASON_LABEL[project.unmappedReason]}
                        </span>
                    </li>
                ))}
            </ul>
        </section>
    );
};
