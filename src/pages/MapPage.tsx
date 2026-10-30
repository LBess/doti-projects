import { ProjectMap } from '../components/ProjectMap';
import { useProjectsContext } from '../hooks/useProjectsContext';
import styles from './MapPage.module.scss';

/** The index page: the map, filling everything below the header. */
export const MapPage = () => {
    const { status, mapped, error } = useProjectsContext();

    return (
        <div className={styles.mapPage}>
            <ProjectMap projects={mapped} />
            {status === 'loading' && <p className={styles.status}>Loading current projects…</p>}
            {status === 'error' && (
                <p className={`${styles.status} ${styles.statusError}`}>{error}</p>
            )}
        </div>
    );
};
