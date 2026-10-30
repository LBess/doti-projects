import { UnmappedProjects } from '../components/UnmappedProjects';
import { useProjectsContext } from '../hooks/useProjectsContext';
import styles from './UnmappedProjectsPage.module.scss';

export const UnmappedProjectsPage = () => {
    const { status, unmapped, error } = useProjectsContext();

    return (
        <div className={styles.page}>
            {status === 'error' && <p className={styles.error}>{error}</p>}
            {(status === 'idle' || status === 'loading') && <p>Loading current projects…</p>}
            {status === 'ready' && <UnmappedProjects projects={unmapped} />}
        </div>
    );
};
