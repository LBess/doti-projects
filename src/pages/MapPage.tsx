import clsx from 'clsx';
import { useState } from 'react';
import { ProjectDrawer } from '../components/ProjectDrawer';
import { ProjectMap } from '../components/ProjectMap';
import { useProjectsContext } from '../hooks/useProjectsContext';
import { ListIcon } from '../icons/ListIcon';
import styles from './MapPage.module.scss';

/** The index page: the map, filling everything below the header. */
export const MapPage = () => {
    const { status, cities, mapped, error } = useProjectsContext();
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [drawerOpen, setDrawerOpen] = useState(true);

    const selected = mapped.find((project) => project.id === selectedId) ?? null;

    const select = (id: string) => {
        setSelectedId(id);
        setDrawerOpen(true);
    };

    return (
        <div className={styles.mapPage}>
            <div className={styles.mapArea}>
                <ProjectMap
                    cities={cities}
                    projects={mapped}
                    selectedId={selectedId}
                    onSelect={select}
                />

                {!drawerOpen && (
                    <button
                        type="button"
                        className={styles.drawerToggle}
                        onClick={() => setDrawerOpen(true)}
                    >
                        <ListIcon />
                        Projects
                    </button>
                )}

                {status === 'loading' && <p className={styles.status}>Loading current projects…</p>}
                {status === 'error' && (
                    <p className={clsx(styles.status, styles.statusError)}>{error}</p>
                )}
            </div>

            <ProjectDrawer
                projects={mapped}
                selected={selected}
                open={drawerOpen}
                onSelect={select}
                onBack={() => setSelectedId(null)}
                onClose={() => setDrawerOpen(false)}
            />
        </div>
    );
};
